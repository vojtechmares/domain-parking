#!/usr/bin/env bash
#
# One-time VPS bootstrap for domain-parking. Run as root on the target host:
#
#   sudo deploy/bootstrap.sh
#
# Idempotent: safe to re-run after editing the unit or sudoers file.
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "must be run as root" >&2
  exit 1
fi

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
base=/opt/domain-parking

# Runtime user: unprivileged, no login shell, read+execute only.
nologin_shell="$(command -v nologin || true)"
: "${nologin_shell:=/usr/sbin/nologin}"
if ! id domain-parking >/dev/null 2>&1; then
  useradd --system --no-create-home --shell "$nologin_shell" --user-group domain-parking
fi

# Deploy user: SSH/rsync target, owns the release tree.
if ! id deploy >/dev/null 2>&1; then
  useradd --create-home --shell /bin/bash deploy
fi
install -d -o deploy -g deploy -m 0700 ~deploy/.ssh

# Release tree: owned by deploy, group-readable/executable by the runtime user.
# The setgid bit (2750) makes every release dir and the binary rsync'd into it
# inherit group domain-parking, so the runtime user can execute it - without it,
# releases land as deploy:deploy and the service fails to exec (203/EXEC).
install -d -o deploy -g domain-parking -m 2750 "$base" "$base/releases"

# systemd unit. Points at the stable current symlink, so it is installed once.
install -m 0644 "$here/domain-parking.service" /etc/systemd/system/domain-parking.service

# Narrow sudoers grant for the deploy user (restart only).
install -m 0440 "$here/domain-parking.sudoers" /etc/sudoers.d/domain-parking
visudo -cf /etc/sudoers.d/domain-parking

systemctl daemon-reload
# Enable at boot, but do NOT start: the current symlink does not exist until the
# first deploy, which performs the first start via `systemctl restart`.
systemctl enable domain-parking

cat <<'MSG'

bootstrap complete. next steps:
  1. add the CI deploy public key to ~deploy/.ssh/authorized_keys
  2. ensure this host is on the tailnet and the ACL allows tag:ci -> this host on tcp/22
  3. block external access to :8090 at the host firewall (the app binds 127.0.0.1 by default)
  4. point the reverse proxy at 127.0.0.1:8090, forwarding the X-Forwarded-Host header
  5. trigger the first deploy (merge to main) — it creates the first release and starts the service
MSG
