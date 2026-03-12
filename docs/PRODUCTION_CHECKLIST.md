## Production Checklist

This is the single production-readiness checklist for the workspace. It replaces older TODO-style production notes and should be treated as the authoritative operational baseline for the public repositories.

- **Required secrets**: define `AUTH_ENCRYPTION_KEY`, session secrets, internal tokens, metrics tokens, and queue encryption keys through environment variables or a secret manager. Do not rely on example values or fallbacks.
- **Secure cookies**: enable `Secure`, `HttpOnly`, and the intended `SameSite` policy, and configure a real domain. Do not ship localhost-oriented cookie settings to production.
- **Database and Redis credentials**: replace development usernames and passwords, store them in a secret manager, and restrict network reachability.
- **Trusted proxy CIDRs**: set `GATEWAY_TRUSTED_CIDRS` only for proxies you actually trust. Otherwise the gateway will correctly overwrite forwarded headers, but the deployment intent is already wrong.
- **Private surfaces**: keep PDP admin and all internal-only endpoints off the public gateway. Enforce this with network policy, ACLs, or mTLS.
- **Internal routing**: PEP, gateway, smoke tooling, and service-to-service flows must use internal bases for `auth`, `pdp`, `notification`, and `audit`. Do not expose `/auth/internal/*` or `/pdp/v1/decision` publicly.
- **Admin tokens**: set `PDP_ADMIN_TOKEN` and similar control-plane tokens explicitly and keep them in secrets, never in examples or static files.
- **Scoped tokens**: keep `AUTH_INTERNAL_TOKEN`, `AUTH_METRICS_TOKEN`, `AUTH_EMAIL_VERIFICATION_INTERNAL_TOKEN`, `PDP_INTERNAL_TOKEN`, `PDP_METRICS_TOKEN`, `NOTIFICATION_INTERNAL_TOKEN`, and `AUDIT_INTERNAL_TOKEN` distinct.
- **Redis hardening**: require authentication or use a managed Redis offering. Ensure all Redis URLs used for rate limit and lockout are production-grade.
- **Headers and timeouts**: review request header limits, body limits, and timeouts against your real SLOs and traffic profile. Tune `MaxHeaderBytes` and edge proxy limits together.
- **Authorization context**: when a policy depends on `context.ip`, `context.method`, `context.path`, or `context.user_agent`, make sure the trusted caller sends those values explicitly. The PDP should not invent them.
- **Logs and audit**: avoid logging tokens, cookies, or sensitive query strings. Monitor audit fanout, spool growth, and delivery failures.
- **TLS posture**: terminate TLS at a trusted boundary. If you use an upstream proxy, ensure `X-Forwarded-Proto` is set correctly and certificates are valid. Internal HTTP should stay limited to development-only scenarios.
- **Rate limiting**: calibrate `GATEWAY_RATE_LIMIT_RPS`, `GATEWAY_RATE_LIMIT_BURST`, `GATEWAY_RATE_LIMIT_MAX_KEYS`, and email-related limiters using production traffic and abuse models.
- **Dependency patching**: review container base images and Go/tooling versions regularly. Apply security patches promptly.
- **Observability**: `expvar` and breaker telemetry are useful only if you actually monitor them. Alert on breaker state changes, repeated audit delivery failures, and sustained queue backlogs.

## Current Scope

The platform is intentionally opinionated and security-focused, but it is still a foundation. Some integrations remain deliberately basic:

- delivery and audit pipelines are functional but still operationally simple
- deployment defaults are conservative, but production still depends on correct secret and network management
- service-to-service transport hardening should be completed with environment-specific TLS and infrastructure controls

That is acceptable for a public starting point, as long as teams adopting the stack understand that production maturity still requires operational work.