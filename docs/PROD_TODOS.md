# Debt / TODOs para producao

Checklist de pontos aceitaveis em DEV mas que precisam endurecer antes de producao.

- [ ] **AUTH_ENCRYPTION_KEY sem fallback**
  - DEV: fallback em `deploy/.env.example` e compose permite subir rapido.
  - Producao: falhar se `AUTH_ENCRYPTION_KEY` nao estiver definido; sem valor default.
  - Como: remover defaults em `deploy/docker-compose.yml` e exigir `AUTH_ENCRYPTION_KEY` via secret/env de producao.

- [x] **Smoke usa subject real e rotas internas corretas**
  - `tools/smoke` agora usa `subject.user_id`, `tenant_id` e `auth_time` retornados pelo `/internal/sessions/introspect`.
  - O smoke tambem fala com `auth` e `pdp` internos para introspeccao e decisao, em vez de tentar passar pelo gateway publico.

- [ ] **Admin do PDP via gateway**
  - DEV: bloqueado por padrao; no Docker `RemoteAddr` nao e `127.0.0.1`.
  - Producao: nao expor admin via gateway publico; manter apenas em rede interna e/ou proteger com mTLS/ACL/allowlist CIDR.
  - Como: garantir que `/pdp/v1/admin/*` fique inacessivel externamente; avaliar separar servico/admin ou aplicar ACL na rede.

- [x] **Senhas fixas no Postgres init removidas**
  - `deploy/postgres-init.sh` cria usuarios/DBs via variaveis de ambiente.
  - `deploy/postgres-init.sql` foi desativado para evitar segredos hardcoded.
