#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <backup.dump|backup.dump.age>" >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

if [[ -f .backup.env ]]; then
  # shellcheck disable=SC1091
  set -a
  source .backup.env
  set +a
fi

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
backup="$1"

if [[ ! -f "$backup" ]]; then
  echo "ERROR: backup not found: $backup" >&2
  exit 1
fi

checksum_file="${backup}.sha256"
if [[ -f "$checksum_file" ]]; then
  echo "Checking SHA-256..."
  (cd "$(dirname "$backup")" && sha256sum -c "$(basename "$checksum_file")")
else
  echo "WARNING: checksum file not found: $checksum_file" >&2
fi

if [[ "$backup" == *.age ]]; then
  if ! command -v age >/dev/null 2>&1; then
    echo "ERROR: encrypted backup requires the 'age' command." >&2
    exit 1
  fi
  if [[ -z "${BACKUP_AGE_IDENTITY:-}" ]]; then
    echo "ERROR: set BACKUP_AGE_IDENTITY to the path of the private age identity file." >&2
    exit 1
  fi
  if [[ ! -f "$BACKUP_AGE_IDENTITY" ]]; then
    echo "ERROR: age identity file not found: $BACKUP_AGE_IDENTITY" >&2
    exit 1
  fi

  echo "Checking PostgreSQL archive structure..."
  age -d -i "$BACKUP_AGE_IDENTITY" "$backup" | \
    docker compose -f "$COMPOSE_FILE" exec -T postgres pg_restore --list >/dev/null
else
  echo "Checking PostgreSQL archive structure..."
  docker compose -f "$COMPOSE_FILE" exec -T postgres pg_restore --list < "$backup" >/dev/null
fi

echo "Backup verification passed: $backup"
