#!/usr/bin/env sh
set -e

if command -v openssl >/dev/null 2>&1; then
  openssl rand -base64 32
  exit 0
fi

if [ -r /dev/urandom ]; then
  head -c 32 /dev/urandom | base64
  exit 0
fi

echo "Erro: instale openssl ou disponibilize /dev/urandom" >&2
exit 1
