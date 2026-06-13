# Deployment

`domain-parking` runs as a systemd unit on a single VPS. Releases are kept under
`/opt/domain-parking/releases/<release>/` and a `current` symlink points at the live one.
GitHub Actions builds the binary, connects to the host over Tailscale, and deploys it over
ssh/rsync.

```
/opt/domain-parking/
├── releases/
│   └── 20260613T115800-a4f2584/
│       └── server
└── current -> releases/20260613T115800-a4f2584
```

## Files

- `domain-parking.service` - systemd unit (`ExecStart=/opt/domain-parking/current/server`).
- `domain-parking.sudoers` - lets the `deploy` user run `systemctl restart domain-parking`.
- `bootstrap.sh` - one-time host setup (users, directories, unit, sudoers). Run as root.
- `release.sh` - remote activate/health-check/rollback/prune, run by the workflow over ssh.

## One-time host setup

Run the bootstrap as root on the VPS (from a checkout of this repo):

```sh
sudo deploy/bootstrap.sh
```

It creates two users - `deploy` (SSH/rsync target, owns the tree) and `domain-parking`
(`nologin` runtime user, read+execute only) - creates `/opt/domain-parking`, installs the unit
and sudoers drop-in, and `enable`s the service (it starts on the first deploy).

Then, on the host:

1. Add the CI deploy **public** key to `~deploy/.ssh/authorized_keys`.
2. Block external access to `:8080` at the host firewall - the app binds `127.0.0.1` by default,
   but defense in depth.
3. Point the existing reverse proxy at `127.0.0.1:8080`, forwarding the `X-Forwarded-Host` header.

> The sudoers file hardcodes `/usr/bin/systemctl`. On a non usr-merged distro adjust the path to
> match `command -v systemctl`.

## Tailscale

The deploy job joins the runner to the tailnet with `tailscale/github-action` using an OAuth
client tagged `tag:ci`. In the tailnet policy:

- define `tag:ci`, and
- allow `tag:ci` to reach the VPS on `tcp/22`, e.g.:

```jsonc
{
  "action": "accept",
  "src": ["tag:ci"],
  "dst": ["<vps>:22"]
}
```

## GitHub secrets

Set these as repository (or `live` environment) secrets:

| Secret | Value |
|--------|-------|
| `TS_OAUTH_CLIENT_ID` | Tailscale OAuth client id (scoped to `tag:ci`) |
| `TS_OAUTH_SECRET` | Tailscale OAuth client secret |
| `SSH_PRIVATE_KEY` | Private half of the deploy key |
| `SSH_KNOWN_HOSTS` | Pinned host key line(s), e.g. `ssh-keyscan <host>` output |
| `DEPLOY_HOST` | VPS MagicDNS name or tailnet IP |
| `DEPLOY_USER` | `deploy` |

## Rollback

Releases are retained (newest 10), so rolling back is a symlink flip on the host:

```sh
cd /opt/domain-parking
ln -sfn releases/<previous> current
sudo systemctl restart domain-parking
```

The deploy workflow also rolls back automatically: if the post-restart health check fails, it
restores the previous release and fails the job.

## Post-migration cleanup

After the first successful VPS deploy, remove the leftovers from the Kubernetes setup:

- Delete the unused repo secrets `KUBE_CONFIG_DATA` and `KUBE_CONTEXT`.
- Delete the now-unused container image package `ghcr.io/vojtechmares/domain-parking`.
