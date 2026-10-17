#!/bin/sh
set -eu
# Read externally provisioned secrets without logging or generating identities.
# Hex database passwords avoid URI quoting ambiguity; use at least 32 hex bytes.
IFS= read -r database_password < /run/secrets/database_password || [ -n "${database_password:-}" ]
case "$database_password" in ''|*[!0-9a-fA-F]*) echo 'database_password must be externally provisioned hexadecimal' >&2; exit 1;; esac
[ "${#database_password}" -ge 64 ] || { echo 'database_password needs at least 64 hexadecimal characters' >&2; exit 1; }
WOS_POSTGRES_DSN="postgres://wos:${database_password}@postgres:5432/wos?sslmode=disable"
export WOS_POSTGRES_DSN
# A delivery-only worker needs no bootstrap credential or private issuer key.
# Its administrative listener exposes only process health and operator metrics.
if [ "${WOS_PROCESS_ROLE:-complete}" != worker ]; then
    IFS= read -r WOS_BOOTSTRAP_TOKEN < /run/secrets/bootstrap_token || [ -n "${WOS_BOOTSTRAP_TOKEN:-}" ]
    IFS= read -r WOS_SERVER_SIGNING_SEED < /run/secrets/issuer_seed || [ -n "${WOS_SERVER_SIGNING_SEED:-}" ]
    [ "${#WOS_BOOTSTRAP_TOKEN}" -ge 32 ] || { echo 'bootstrap_token needs at least 32 characters' >&2; exit 1; }
    export WOS_BOOTSTRAP_TOKEN WOS_SERVER_SIGNING_SEED
fi
unset database_password
exec wos "$@"
