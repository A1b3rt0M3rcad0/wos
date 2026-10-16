#!/bin/sh
# Opt-in external scheduler entrypoint. Credentials remain in protected env.
set -eu
umask 077
: "${WOS_BACKUP_ROOT:?set an existing protected backup parent}"
: "${WOS_BACKUP_SOURCE_COMMIT:?set the reviewed deployed Git SHA}"
: "${WOS_BACKUP_PRODUCT_VERSION:?set the deployed WOS version}"
: "${WOS_BACKUP_BACKEND:?set sqlite or postgres}"
case "$WOS_BACKUP_ROOT" in /*) ;; *) echo 'WOS_BACKUP_ROOT must be absolute' >&2; exit 1;; esac
test -d "$WOS_BACKUP_ROOT"
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
# mkdir inside backup_restore.py is exclusive; a collision fails safely.
destination="$WOS_BACKUP_ROOT/$(date -u +%Y%m%dT%H%M%SZ)-$$"
case "$WOS_BACKUP_BACKEND" in
  sqlite)
    : "${WOS_BACKUP_SQLITE:?set the explicit SQLite source file}"
    exec python3 "$script_dir/backup_restore.py" backup --backend sqlite \
      --sqlite "$WOS_BACKUP_SQLITE" --output "$destination" \
      --source-commit "$WOS_BACKUP_SOURCE_COMMIT" --product-version "$WOS_BACKUP_PRODUCT_VERSION"
    ;;
  postgres)
    exec python3 "$script_dir/backup_restore.py" backup --backend postgres \
      --dsn-env WOS_POSTGRES_DSN --schema "${WOS_BACKUP_SCHEMA:-public}" --output "$destination" \
      --source-commit "$WOS_BACKUP_SOURCE_COMMIT" --product-version "$WOS_BACKUP_PRODUCT_VERSION"
    ;;
  *) echo 'WOS_BACKUP_BACKEND must be sqlite or postgres' >&2; exit 1;;
esac
