Param(
    [string]$ComposeFile = "deploy/docker-compose.yml",
    [string]$EnvFile = "deploy/.env",
    [int]$TimeoutSeconds = 60
)

function Wait-Healthy {
    param(
        [string[]]$Services,
        [int]$TimeoutSeconds
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        $psRaw = docker compose -f $ComposeFile --env-file $EnvFile ps --format json 2>$null
        if (-not $?) { return $false }
        $ps = $psRaw | ConvertFrom-Json
        $allGood = $true
        foreach ($svc in $Services) {
            $entry = $ps | Where-Object { $_.Service -eq $svc }
            if (-not $entry) { $allGood = $false; break }
            if ($entry.Health -eq "unhealthy") { return $false }
            if ($entry.Health -ne "healthy" -and $entry.State -notin @("running","exited")) {
                $allGood = $false
                break
            }
        }
        if ($allGood) { return $true }
        Start-Sleep -Seconds 2
    } until ((Get-Date) -gt $deadline)
    return $false
}

Write-Host "Bringing up base dependencies (token-gen, postgres, redis, notification, audit, migrations)..."
docker compose -f $ComposeFile --env-file $EnvFile up -d token-gen postgres redis notification audit auth-migrate pdp-migrate | Out-Host

if (-not (Wait-Healthy -Services @("token-gen","postgres","redis","notification","audit","auth-migrate","pdp-migrate") -TimeoutSeconds $TimeoutSeconds)) {
    Write-Error "Base services not healthy within $TimeoutSeconds seconds"
    docker compose -f $ComposeFile --env-file $EnvFile ps
    docker compose -f $ComposeFile --env-file $EnvFile logs token-gen notification audit auth-migrate pdp-migrate
    exit 1
}

Write-Host "Starting app services (auth, pdp, gateway)..."
docker compose -f $ComposeFile --env-file $EnvFile up -d auth pdp gateway | Out-Host

if (-not (Wait-Healthy -Services @("auth","pdp","gateway") -TimeoutSeconds $TimeoutSeconds)) {
    Write-Error "App services not healthy within $TimeoutSeconds seconds"
    docker compose -f $ComposeFile --env-file $EnvFile ps
    docker compose -f $ComposeFile --env-file $EnvFile logs auth pdp gateway
    exit 1
}

docker compose -f $ComposeFile --env-file $EnvFile ps
Write-Host "All services healthy."
