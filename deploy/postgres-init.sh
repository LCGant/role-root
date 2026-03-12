#!/usr/bin/env sh
set -eu

: "${POSTGRES_USER:?set POSTGRES_USER}"
: "${POSTGRES_DB:?set POSTGRES_DB}"
: "${AUTH_DB_USER:?set AUTH_DB_USER}"
: "${AUTH_DB_PASSWORD:?set AUTH_DB_PASSWORD}"
: "${AUTH_DB_NAME:?set AUTH_DB_NAME}"
: "${PDP_DB_USER:?set PDP_DB_USER}"
: "${PDP_DB_PASSWORD:?set PDP_DB_PASSWORD}"
: "${PDP_DB_NAME:?set PDP_DB_NAME}"

psql -v ON_ERROR_STOP=1 \
  --username "$POSTGRES_USER" \
  --dbname "$POSTGRES_DB" \
  -v auth_user="$AUTH_DB_USER" \
  -v auth_pass="$AUTH_DB_PASSWORD" \
  -v auth_db="$AUTH_DB_NAME" \
  -v pdp_user="$PDP_DB_USER" \
  -v pdp_pass="$PDP_DB_PASSWORD" \
  -v pdp_db="$PDP_DB_NAME" <<'SQL'
DO
$$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'auth_user') THEN
        EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', :'auth_user', :'auth_pass');
    END IF;
END
$$;

DO
$$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'pdp_user') THEN
        EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', :'pdp_user', :'pdp_pass');
    END IF;
END
$$;

DO
$$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'auth_db') THEN
        EXECUTE format('CREATE DATABASE %I OWNER %I', :'auth_db', :'auth_user');
    END IF;
END
$$;

DO
$$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'pdp_db') THEN
        EXECUTE format('CREATE DATABASE %I OWNER %I', :'pdp_db', :'pdp_user');
    END IF;
END
$$;
SQL

