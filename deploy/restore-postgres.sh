#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

usage() {
  cat >&2 <<'EOF'
Usage:
  ./restore-postgres.sh <backup.dump|backup.dump.age> --confirm-restore

This is a destructive recovery operation for the current PostgreSQL database.
The application service is stopped before the database is recreated.
EOF
}

if [[ $# -ne 2 || "$2" != "--confirm-restore" ]]; then
  usage
  exit 2
fi

backup="$1"
ENV_FILE="${ENV_FILE:-.env}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
OVERLAY_FILE="${OVERLAY_FILE:-docker-compose.termrelay.yml}"

[[ -f "$backup" ]] || { echo "ERROR: backup not found: $backup" >&2; exit 1; }
[[ -f "$ENV_FILE" ]] || { echo "ERROR: missing environment file: $ENV_FILE" >&2; exit 1; }
[[ -f "$OVERLAY_FILE" ]] || { echo "ERROR: missing TermRelay production overlay: $OVERLAY_FILE" >&2; exit 1; }

if [[ -f .backup.env ]]; then
  # shellcheck disable=SC1091
  set -a
  source .backup.env
  set +a
fi

for cmd in docker; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "ERROR: required command not found: $cmd" >&2; exit 1; }
done

docker compose version >/dev/null 2>&1 || { echo "ERROR: Docker Compose plugin is not available" >&2; exit 1; }

# Validate secrets, immutable image source and backup checksum before touching state.
ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" OVERLAY_FILE="$OVERLAY_FILE" \
  "$SCRIPT_DIR/recovery-preflight.sh" "$backup"

# PostgreSQL must already be running so the archive can be verified and restored.
if ! docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps --status running postgres 2>/dev/null | grep -q postgres; then
  echo "ERROR: PostgreSQL compose service is not running." >&2
  echo "Start only postgres/redis first, then retry the restore." >&2
  exit 1
fi

# Verify archive structure before touching the target database.
ENV_FILE="$ENV_FILE" COMPOSE_FILE="$COMPOSE_FILE" "$SCRIPT_DIR/verify-postgres-backup.sh" "$backup"

db_user="$(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres sh -ec 'printf "%s" "$POSTGRES_USER"')"
db_name="$(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres sh -ec 'printf "%s" "$POSTGRES_DB"')"

[[ -n "$db_user" ]] || { echo "ERROR: POSTGRES_USER resolved empty inside postgres container" >&2; exit 1; }
[[ -n "$db_name" ]] || { echo "ERROR: POSTGRES_DB resolved empty inside postgres container" >&2; exit 1; }

app_stopped=false
on_error() {
  status=$?
  if [[ "$app_stopped" == "true" ]]; then
    echo "ERROR: restore failed after the application was stopped." >&2
    echo "The application remains stopped to avoid serving a partially restored database." >&2
    echo "Inspect PostgreSQL, fix the cause, then rerun the restore." >&2
  fi
  exit "$status"
}
trap on_error ERR

echo "Stopping application service before database restore..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" -f "$OVERLAY_FILE" stop sub2api
app_stopped=true

echo "Recreating PostgreSQL database: $db_name"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres \
  dropdb --force --if-exists -U "$db_user" "$db_name"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres \
  createdb -U "$db_user" "$db_name"

restore_args=(--no-owner --no-privileges --exit-on-error -U "$db_user" -d "$db_name")

if [[ "$backup" == *.age ]]; then
  command -v age >/dev/null 2>&1 || { echo "ERROR: encrypted backup requires the 'age' command." >&2; exit 1; }
  [[ -n "${BACKUP_AGE_IDENTITY:-}" ]] || { echo "ERROR: BACKUP_AGE_IDENTITY is required for encrypted backup restore." >&2; exit 1; }
  [[ -f "$BACKUP_AGE_IDENTITY" ]] || { echo "ERROR: age identity file not found: $BACKUP_AGE_IDENTITY" >&2; exit 1; }

  age -d -i "$BACKUP_AGE_IDENTITY" "$backup" | \
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres \
      pg_restore "${restore_args[@]}"
else
  docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres \
    pg_restore "${restore_args[@]}" < "$backup"
fi

echo "Database restore completed. Starting application with TermRelay production overlay..."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" -f "$OVERLAY_FILE" up -d sub2api
app_stopped=false
trap - ERR

echo "Restore complete. Review service health before changing DNS or accepting traffic."
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" -f "$OVERLAY_FILE" ps sub2api postgres redis
