#!/usr/bin/env bash
#
# Remote release activation. Run on the VPS by the deploy workflow:
#
#   ssh deploy@host "REL=<release> bash -s" < deploy/release.sh
#
# Flips the current symlink to releases/$REL, restarts the service, health
# checks it, rolls back to the previous release on failure, and prunes old
# releases (keeping the newest 10, never the live one).
set -euo pipefail

: "${REL:?REL must be set to the release id}"
base=/opt/domain-parking
rel="$base/releases/$REL"

if [ ! -x "$rel/server" ]; then
  echo "release binary not found or not executable: $rel/server" >&2
  exit 1
fi

# Remember the live release so we can roll back (empty on the first deploy).
prev=""
if [ -L "$base/current" ]; then
  prev="$(readlink "$base/current")"
fi

ln -sfn "$rel" "$base/current"
sudo systemctl restart domain-parking

# Health gate: up to 10 attempts, 1s apart.
ok=""
for _ in $(seq 1 10); do
  if curl -fsS -H 'X-Forwarded-Host: deploy-check' http://127.0.0.1:8080/ >/dev/null; then
    ok=1
    break
  fi
  sleep 1
done

if [ -z "$ok" ]; then
  echo "health check failed for $REL" >&2
  if [ -n "$prev" ]; then
    ln -sfn "$prev" "$base/current"
    sudo systemctl restart domain-parking
    echo "rolled back to $(basename "$prev")" >&2
  fi
  exit 1
fi

echo "deployed $REL"

# Prune: keep the newest 10 releases by name, never remove the live one.
# Release ids are date-prefixed, so reverse lexical sort lists newest first.
# Compare by release id (basename), not full path, to stay robust to symlinks.
keep="$(basename "$(readlink "$base/current")")"
shopt -s nullglob
mapfile -t releases < <(cd "$base/releases" && printf '%s\n' */ | sort -r)
for dir in "${releases[@]:10}"; do
  name="${dir%/}"
  [ "$name" = "$keep" ] || rm -rf -- "${base:?}/releases/${name:?}"
done
