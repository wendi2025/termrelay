#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
preflight="$repo_root/deploy/recovery-preflight.sh"
restore="$repo_root/deploy/restore-postgres.sh"

for file in "$preflight" "$restore"; do
  if [ ! -f "$file" ]; then
    echo "missing recovery script: $file" >&2
    exit 1
  fi
done

# Recovery must require an owner-controlled, explicit production image.
grep -Fq 'ghcr.io/yuchenm1303-png/sub2api' "$preflight" || {
  echo "recovery preflight must enforce the TermRelay GHCR source" >&2
  exit 1
}

grep -Fq 'refusing mutable production image tag' "$preflight" || {
  echo "recovery preflight must reject mutable image tags" >&2
  exit 1
}

# Database restore is intentionally destructive, so explicit confirmation and
# application shutdown must remain mandatory.
grep -Fq -- '--confirm-restore' "$restore" || {
  echo "restore script must require explicit confirmation" >&2
  exit 1
}

grep -Fq 'stop sub2api' "$restore" || {
  echo "restore script must stop the application before replacing the database" >&2
  exit 1
}

grep -Fq 'dropdb --force --if-exists' "$restore" || {
  echo "restore script must use the guarded database recreation path" >&2
  exit 1
}

grep -Fq 'pg_restore' "$restore" || {
  echo "restore script must restore with pg_restore" >&2
  exit 1
}

# The recovered application must come back through the TermRelay overlay rather
# than silently falling back to the upstream mutable image in the base compose.
grep -Fq 'Starting application with TermRelay production overlay' "$restore" || {
  echo "restore script must restart through the TermRelay production overlay" >&2
  exit 1
}

grep -Fq -- '-f "$OVERLAY_FILE" up -d sub2api' "$restore" || {
  echo "restore script must use the production overlay when starting the application" >&2
  exit 1
}

# Recovery tooling must never casually delete Docker volumes or the whole
# deployment directory. Those actions would turn disaster recovery into data
# destruction.
if grep -Eq 'docker[[:space:]]+compose.*down[[:space:]]+.*-v|docker-compose.*down[[:space:]]+.*-v' "$restore" "$preflight"; then
  echo "recovery scripts must not run docker compose down -v" >&2
  exit 1
fi

if grep -Eq 'rm[[:space:]]+-rf[[:space:]]+/' "$restore" "$preflight"; then
  echo "recovery scripts must not recursively delete absolute paths" >&2
  exit 1
fi

printf 'Recovery safety test passed\n'
