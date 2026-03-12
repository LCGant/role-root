#!/usr/bin/env sh
set -eu

COMPOSE_FILE="${1:-deploy/docker-compose.yml}"
ENV_FILE="${2:-deploy/.env}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-60}"

wait_healthy() {
  deadline=$(( $(date +%s) + TIMEOUT_SECONDS ))
  while [ "$(date +%s)" -le "$deadline" ]; do
    ps_raw="$(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps --format json 2>/dev/null || true)"
    [ -n "$ps_raw" ] || return 1

    all_good=1
    for svc in "$@"; do
      entry="$(printf '%s\n' "$ps_raw" | grep "\"Service\":\"$svc\"" || true)"
      [ -n "$entry" ] || { all_good=0; break; }

      health="$(printf '%s' "$entry" | sed -n 's/.*"Health":"\([^"]*\)".*/\1/p')"
      state="$(printf '%s' "$entry" | sed -n 's/.*"State":"\([^"]*\)".*/\1/p')"

      if [ "$health" = "unhealthy" ]; then
        return 1
      fi
      if [ "$health" != "healthy" ] && [ "$state" != "running" ] && [ "$state" != "exited" ]; then
        all_good=0
        break
      fi
    done

    if [ "$all_good" -eq 1 ]; then
      return 0
    fi
    sleep 2
  done
  return 1
}

echo "Bringing up base dependencies (token-gen, postgres, redis, notification, audit, migrations)..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d token-gen postgres redis notification audit auth-migrate pdp-migrate

if ! wait_healthy token-gen postgres redis notification audit auth-migrate pdp-migrate; then
  echo "Base services not healthy within ${TIMEOUT_SECONDS} seconds" >&2
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" logs token-gen notification audit auth-migrate pdp-migrate
  exit 1
fi

echo "Starting app services (auth, pdp, gateway)..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d auth pdp gateway

if ! wait_healthy auth pdp gateway; then
  echo "App services not healthy within ${TIMEOUT_SECONDS} seconds" >&2
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" logs auth pdp gateway
  exit 1
fi

docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps
echo "All services healthy."
