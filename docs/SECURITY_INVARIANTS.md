# Security Invariants

This document captures the security properties that must remain true across the
gateway, auth service, PEP, and PDP. If a code change breaks one of these
invariants, treat it as a security regression.

## Session lifecycle

- A live session is identified only by the hash of a high-entropy opaque token.
- Each live session is bound to a server-issued opaque device token.
- Sessions without `device_id` are treated as invalid legacy state and must not
  continue validating after the rollout of device-bound sessions.
- Device-bound sessions must not validate without the matching device token at
  runtime.
- Session expiration has two bounds: absolute TTL and idle TTL.
- Password reset revokes all sessions for the user.
- Sensitive reauthentication rotates the current session token instead of only
  refreshing metadata on the existing token.
- Enabling or disabling MFA rotates the current session and revokes sibling
  sessions for the same user.
- Revoking a device revokes every session currently bound to that device.

## AAL and auth_time

- Password login starts at `aal=1`.
- MFA-completed login starts at `aal=2`.
- A fresh step-up or sensitive reauthentication must advance `auth_time`.
- `auth_time` must never be fabricated by downstream callers when the upstream
  session introspection did not provide it.

## OAuth and passwordless accounts

- OAuth login for an MFA-enabled account must not issue a final session until
  `/oauth/login/complete` succeeds with TOTP or backup code.
- Auto-provisioned OAuth accounts are passwordless by default and must have
  `password_auth_enabled=false`.
- Locally registered accounts remain `pending_verification` until the email
  verification flow marks them as `email_verified=true` and `status=active`.
- Sensitive operations for passwordless accounts may rely on a fresh primary
  session, but must not pretend that a local password exists.
- Unlinking an identity must preserve at least one usable login method.

## CSRF and browser boundaries

- Mutating authenticated routes require double-submit CSRF validation.
- Public mutating routes must enforce origin checks for browser traffic.
- Non-browser clients that do not send `Origin` or `Referer` must provide an
  explicit API client signal such as `X-Client-Family`; this signal is only a
  browser-boundary hint and must not grant extra privilege by itself.
- OAuth state and pending-login cookies must be scoped to the public auth path
  actually exposed to the client.

## Gateway and trust boundaries

- The public gateway must block auth internal routes and PDP decision/admin
  routes unless an explicit internal-only policy exists.
- `X-Forwarded-For` is trusted only when the immediate peer belongs to an
  explicitly configured trusted CIDR.
- Internal service, metrics, and admin tokens are separate scopes.
- Auth introspection and internal email-verification issuance use separate
  secrets.

## PDP and policy context

- The PDP must not invent policy-relevant request context such as `ip`,
  `method`, `path`, or `user_agent`.
- If a policy depends on context, the trusted caller must provide that context.
- Tenant mismatch between subject and resource is always a deny.

## Operational checks

- Smoke tests must use real subject data from auth introspection for PDP
  scenarios.
- Smoke tests that exercise internal auth or PDP endpoints must talk to the
  internal services directly, not through the public gateway.
