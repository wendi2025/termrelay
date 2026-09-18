#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Optional local-only configuration. This file is gitignored.
if [[ -f .backup.env ]]; then
  # shellcheck disable=SC1091
  set -a
  source .backup.env
  set +a
fi

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
BACKUP_DIR="${BACKUP_DIR:-$SCRIPT_DIR/backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"
BACKUP_RCLONE_REMOTE="${BACKUP_RCLONE_REMOTE:-}"
BACKUP_AGE_RECIPIENT="${BACKUP_AGE_RECIPIENT:-}"
ALLOW_UNENCRYPTED_REMOTE_BACKUP="${ALLOW_UNENCRYPTED_REMOTE_BACKUP:-false}"

umask 077
mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR" 2>/dev/null || true

for cmd in docker sha256sum; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "ERROR: required command not found: $cmd" >&2
    exit 1
  fi
done

if ! [[ "$BACKUP_RETENTION_DAYS" =~ ^[0-9]+$ ]]; then
  echo "ERROR: BACKUP_RETENTION_DAYS must be a non-negative integer" >&2
  exit 1
fi

if ! docker compose -f "$COMPOSE_FILE" ps --status running postgres 2>/dev/null | grep -q postgres; then
  echo "ERROR: PostgreSQL compose service is not running." >&2
  echo "Run this script from the deploy directory of the active TermRelay stack or set COMPOSE_FILE." >&2
  exit 1
fi

timestamp="$(date -u +'%Y%m%dT%H%M%SZ')"
base_name="termrelay-postgres-${timestamp}.dump"
tmp_file="$BACKUP_DIR/.${base_name}.tmp"
dump_file="$BACKUP_DIR/$base_name"

cleanup() {
  rm -f "$tmp_file"
}
trap cleanup EXIT

echo "Creating PostgreSQL backup..."
docker compose -f "$COMPOSE_FILE" exec -T postgres sh -ec \
  'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "$tmp_file"

if [[ ! -s "$tmp_file" ]]; then
  echo "ERROR: pg_dump produced an empty backup." >&2
  exit 1
fi

# Validate the custom-format archive before accepting it as a backup.
if ! docker compose -f "$COMPOSE_FILE" exec -T postgres pg_restore --list < "$tmp_file" >/dev/null; then
  echo "ERROR: pg_restore could not read the generated backup." >&2
  exit 1
fi

mv "$tmp_file" "$dump_file"
artifact="$dump_file"

# Encrypt before any off-server upload when an age recipient is configured.
if [[ -n "$BACKUP_AGE_RECIPIENT" ]]; then
  if ! command -v age >/dev/null 2>&1; then
    echo "ERROR: BACKUP_AGE_RECIPIENT is set but the 'age' command is not installed." >&2
    exit 1
  fi
  encrypted_file="${dump_file}.age"
  age -r "$BACKUP_AGE_RECIPIENT" -o "$encrypted_file" "$dump_file"
  rm -f "$dump_file"
  artifact="$encrypted_file"
fi

sha256sum "$artifact" > "${artifact}.sha256"
chmod 600 "$artifact" "${artifact}.sha256" 2>/dev/null || true

if [[ -n "$BACKUP_RCLONE_REMOTE" ]]; then
  if [[ "$artifact" != *.age && "$ALLOW_UNENCRYPTED_REMOTE_BACKUP" != "true" ]]; then
    echo "ERROR: refusing to upload an unencrypted database backup." >&2
    echo "Set BACKUP_AGE_RECIPIENT, or explicitly set ALLOW_UNENCRYPTED_REMOTE_BACKUP=true if you accept the risk." >&2
    exit 1
  fi
  if ! command -v rclone >/dev/null 2>&1; then
    echo "ERROR: BACKUP_RCLONE_REMOTE is set but rclone is not installed." >&2
    exit 1
  fi

  echo "Uploading encrypted backup to remote storage..."
  rclone copyto "$artifact" "${BACKUP_RCLONE_REMOTE%/}/$(basename "$artifact")"
  rclone copyto "${artifact}.sha256" "${BACKUP_RCLONE_REMOTE%/}/$(basename "${artifact}.sha256")"
fi

# Local retention only. Remote retention should be configured independently on
# the user-controlled storage account so a compromised server cannot erase all history.
find "$BACKUP_DIR" -maxdepth 1 -type f \
  \( -name 'termrelay-postgres-*.dump' -o -name 'termrelay-postgres-*.dump.age' -o -name 'termrelay-postgres-*.sha256' \) \
  -mtime "+$BACKUP_RETENTION_DAYS" -delete

echo "Backup complete: $artifact"
echo "Checksum: ${artifact}.sha256"
if [[ -n "$BACKUP_RCLONE_REMOTE" ]]; then
  echo "Remote copy: ${BACKUP_RCLONE_REMOTE%/}/$(basename "$artifact")"
fi
