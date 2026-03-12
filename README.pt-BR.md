# Role Root

[Read in English](README.md)

Role Root e o repositorio de workspace da plataforma. Ele conecta deploy, smoke tooling, wiring de integracao e os repositorios que implementam os servicos de seguranca.

Este repositorio deve ser visto como o ponto de coordenacao da plataforma, nao como o unico lugar onde o codigo vive.

## Repositorios

- [`role-root`](https://github.com/LCGant/role-root): workspace, deploy, docs, smoke tooling e cola de orquestracao
- [`role-gateway`](https://github.com/LCGant/role-gateway): base de gateway e servico de borda publica
- [`role-auth`](https://github.com/LCGant/role-auth): base de autenticacao
- [`role-pdp`](https://github.com/LCGant/role-pdp): base de decisao de autorizacao
- [`role-pep`](https://github.com/LCGant/role-pep): biblioteca de policy enforcement
- [`role-notification`](https://github.com/LCGant/role-notification): base interna de entrega de notificacoes
- [`role-audit`](https://github.com/LCGant/role-audit): base interna de coleta de auditoria

## O que existe neste repositorio

- `deploy`: Dockerfiles, Compose, bootstrap e scripts operacionais
- `tools/smoke`: smoke checks do stack integrado
- `docs`: invariantes de seguranca, checklist de producao e notas operacionais
- referencias de workspace para os repositorios de servico listados acima

## Estado do projeto

A plataforma ja e um bom ponto de partida para equipes que querem uma base seria de auth/authz em Go. Ela nao esta sendo apresentada como software totalmente finalizado. Algumas integracoes e partes operacionais continuam intencionalmente basicas, o que faz do projeto uma boa fundacao e nao uma plataforma turnkey.

## Ordem sugerida de leitura

1. Leia este repositorio primeiro para entender o workspace.
2. Leia os repositorios de servico na ordem `role-gateway`, `role-auth`, `role-pdp`, `role-pep`.
3. Revise `docs/SECURITY_INVARIANTS.md` e `docs/PRODUCTION_CHECKLIST.md` antes de mexer em fluxos sensiveis.
