#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ENV_FILE="${ENV_FILE:-.env}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
OVERLAY_FILE="${OVERLAY_FILE:-docker-compose.termrelay.yml}"
IMAGE_REF="${TERMRELAY_IMAGE_REF:-}"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

read_env_value() {
  local key="$1"
  awk -v key="$key" '
    $0 ~ "^[[:space:]]*" key "=" {
      sub(/^[^=]*=/, "")
      gsub(/^[[:space:]]+|[[:space:]]+$/, "")
      if (($0 ~ /^\".*\"$/) || ($0 ~ /^\047.*\047$/)) {
        $0 = substr($0, 2, length($0) - 2)
      }
      print
      exit
    }
  ' "$ENV_FILE"
}

need_cmd docker
need_cmd awk
need_cmd grep

docker compose version >/dev/null 2>&1 || fail "Docker Compose plugin is not available"
[[ -f "$ENV_FILE" ]] || fail "missing environment file: $ENV_FILE"
[[ -f "$COMPOSE_FILE" ]] || fail "missing compose file: $COMPOSE_FILE"
[[ -f "$OVERLAY_FILE" ]] || fail "missing TermRelay production overlay: $OVERLAY_FILE"

# These values are required for a faithful disaster recovery. Losing them may
# invalidate sessions, break TOTP, or make the database inaccessible.
for key in POSTGRES_PASSWORD JWT_SECRET TOTP_ENCRYPTION_KEY; do
  value="$(read_env_value "$key")"
  [[ -n "$value" ]] || fail "$key is empty or missing in $ENV_FILE"
done

if [[ -z "$IMAGE_REF" ]]; then
  IMAGE_REF="$(read_env_value TERMRELAY_IMAGE_REF)"
fi

[[ -n "$IMAGE_REF" ]] || fail "TERMRELAY_IMAGE_REF must be set to an explicit TermRelay release tag or digest"

case "$IMAGE_REF" in
  ghcr.io/yuchenm1303-png/sub2api:*) ;;
  ghcr.io/yuchenm1303-png/sub2api@sha256:*) ;;
  *) fail "TERMRELAY_IMAGE_REF must use ghcr.io/yuchenm1303-png/sub2api" ;;
esac

case "$IMAGE_REF" in
  *:latest|*:main) fail "refusing mutable production image tag: $IMAGE_REF" ;;
esac

# Compose interpolation must succeed before a recovery attempt touches state.
TERMRELAY_IMAGE_REF="$IMAGE_REF" docker compose \
  --env-file "$ENV_FILE" \
  -f "$COMPOSE_FILE" \
  -f "$OVERLAY_FILE" \
  config >/dev/null

if [[ $# -gt 1 ]]; then
  fail "usage: $0 [backup.dump|backup.dump.age]"
fi

if [[ $# -eq 1 ]]; then
  backup="$1"
  [[ -f "$backup" ]] || fail "backup not found: $backup"
  if [[ -f "${backup}.sha256" ]]; then
    need_cmd sha256sum
    (
      cd "$(dirname "$backup")"
      sha256sum -c "$(basename "${backup}.sha256")"
    ) >/dev/null
  else
    echo "WARNING: checksum file not found: ${backup}.sha256" >&2
  fi

  if [[ "$backup" == *.age ]]; then
    need_cmd age
    [[ -n "${BACKUP_AGE_IDENTITY:-}" ]] || echo "WARNING: BACKUP_AGE_IDENTITY is not set; encrypted backup can be integrity-checked but not decrypted yet." >&2
  fi
fi

echo "Recovery preflight passed."
echo "Environment: $ENV_FILE"
echo "Image: $IMAGE_REF"
