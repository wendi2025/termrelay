#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
overlay="$repo_root/deploy/docker-compose.termrelay.yml"
goreleaser="$repo_root/.goreleaser.yaml"
goreleaser_simple="$repo_root/.goreleaser.simple.yaml"
release_workflow="$repo_root/.github/workflows/release.yml"

if [ ! -f "$overlay" ]; then
  echo "missing TermRelay production compose overlay" >&2
  exit 1
fi

for config in "$goreleaser" "$goreleaser_simple" "$release_workflow"; do
  if [ ! -f "$config" ]; then
    echo "missing production/release control file: $config" >&2
    exit 1
  fi
done

if ! grep -Fq 'image: ${TERMRELAY_IMAGE_REF:?' "$overlay"; then
  echo "TermRelay production overlay must require an explicit TERMRELAY_IMAGE_REF" >&2
  exit 1
fi

# Inspect actual Compose image directives only. Documentation comments may name
# unsafe examples such as an upstream :latest tag in order to warn against it.
if grep -Eq '^[[:space:]]*image:[[:space:]]*.*weishaw/sub2api:(latest|main)([[:space:]]|$)' "$overlay"; then
  echo "TermRelay production overlay must not pin production to a mutable upstream image" >&2
  exit 1
fi

if ! grep -Fq 'ghcr.io/yuchenm1303-png/sub2api' "$overlay"; then
  echo "TermRelay production overlay should document the repository-owner GHCR image source" >&2
  exit 1
fi

# Full and simple releases must both publish the same owner-controlled,
# versioned image path used by production. Floating :latest may also exist for
# convenience, but production is never instructed to use it.
for config in "$goreleaser" "$goreleaser_simple"; do
  if ! grep -Fq 'ghcr.io/{{ .Env.GITHUB_REPO_OWNER_LOWER }}/sub2api:{{ .Version }}' "$config"; then
    echo "GoReleaser config must publish the versioned owner-controlled GHCR image: $config" >&2
    exit 1
  fi
done

# A protected main branch must remain authoritative. Release automation may
# create Releases/packages with its scoped token, but it must not push commits
# straight back into main or rely on an administrator bypass token.
if grep -Eq '^[[:space:]]*git[[:space:]]+push([[:space:]]|$)' "$release_workflow"; then
  echo "Release workflow must not git-push directly to the repository" >&2
  exit 1
fi

if grep -Fq 'sync-version-file:' "$release_workflow"; then
  echo "Release workflow must not reintroduce direct VERSION syncing to main" >&2
  exit 1
fi

if ! grep -Fq 'GHCR_IMAGE="ghcr.io/${{ steps.lowercase.outputs.owner }}/sub2api"' "$release_workflow"; then
  echo "Release notification must use the same owner-controlled GHCR image path as GoReleaser" >&2
  exit 1
fi

printf 'TermRelay production and release control test passed\n'
