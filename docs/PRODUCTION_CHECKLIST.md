## Checklist para ir a produção

- **Segredos obrigatórios**: definir AUTH_ENCRYPTION_KEY e chaves de sessão/tokens em variáveis de ambiente/secret manager (sem valores de exemplo ou fallback).
- **Cookies seguros**: habilitar Secure, HttpOnly, SameSite adequados e usar domínio real; não usar localhost em produção.
- **Credenciais de banco/redis**: substituir senhas e usuários de dev; armazenar em secrets; restringir rede.
- **Trusted CIDRs**: configurar GATEWAY_TRUSTED_CIDRS apenas com proxies internos confiáveis (caso contrário XFF é sobrescrito por segurança).
- **Admin/privado**: manter PDP admin fora do gateway público; restringir via rede/mTLS/ACL.
- **Rotas internas**: PEP/gateway e smoke devem usar bases internas separadas para auth/PDP; não tente expor `/auth/internal/*` ou `/pdp/v1/decision` pelo gateway público.
- **PDP admin token**: definir PDP_ADMIN_TOKEN não-vazio e guardado em secret; não usar placeholder.
- **Tokens por escopo**: separar `AUTH_INTERNAL_TOKEN`, `AUTH_METRICS_TOKEN`, `PDP_INTERNAL_TOKEN` e `PDP_METRICS_TOKEN`; não reutilizar o mesmo segredo entre introspecção, decisão, admin e observabilidade.
- **Redis**: usar senha obrigatória ou serviço gerenciado; atualizar URLs redis://user:pass@host para rate-limit/lockout.
- **Headers/timeout**: confirmar limites de header/body e timeouts conforme SLO; ajustar MaxHeaderBytes se necessário.
- **Contexto de autorização**: garantir que o chamador confiável envie `context.ip/method/path/user_agent` quando a policy depender disso; o PDP não deve inferir esses campos.
- **Logs**: evitar incluir tokens/queries sensíveis; monitorar para não logar cookies/Authorization.
- **TLS**: terminar TLS em ponto confiável; se houver proxy TLS, garantir X-Forwarded-Proto correto e certificados válidos.
- **Rate limit**: calibrar GATEWAY_RATE_LIMIT_RPS, GATEWAY_RATE_LIMIT_BURST, GATEWAY_RATE_LIMIT_MAX_KEYS conforme tráfego real.
- **Dependências**: revisar imagens/container base e atualizar patches de segurança.
- **Breaker/Observabilidade**: expvar publica breaker.<route>.state/trips; monitorar se abrir com frequência.
