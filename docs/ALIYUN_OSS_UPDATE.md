# Optional Tonkic API release mirror on Alibaba Cloud OSS

Direct GitHub Release updates are the primary server update path. See
`docs/GITHUB_RELEASE_UPDATE.md`. The updater itself no longer requires OSS or
`ossutil`; this document describes the optional release mirror retained for
download redundancy.

Release files are stored only under this prefix:

```text
oss://update-cpa-plus/tonkic-api/
```

The existing `CPA/` prefix is never listed, modified, copied, or deleted.
GitHub Actions use the Shenzhen public endpoint because OSS internal endpoints
do not work across Alibaba Cloud regions. Servers using `update.sh` connect
directly to GitHub Releases; OSS is only an optional bootstrap mirror.

`power-user-access` is the RAM username, not an AccessKey ID. GitHub Actions
secrets must contain an AccessKey pair created under that RAM user:

- `ALIYUN_OSS_ACCESS_KEY_ID`: the generated ID, typically beginning with `LTAI`;
- `ALIYUN_OSS_ACCESS_KEY_SECRET`: the generated secret shown at creation time.

## OSS layout

```text
tonkic-api/update.sh
tonkic-api/releases/v0.1.0/...
tonkic-api/releases/latest/new-api-linux-amd64
tonkic-api/releases/latest/new-api-linux-arm64
tonkic-api/releases/latest/checksums-latest.txt
tonkic-api/releases/latest/version.txt
```

## One-time server setup

To bootstrap the GitHub-based updater from the optional mirror, configure
`ossutil` once for the RAM user `power-user-access`, using the Shenzhen public
endpoint, and download the updater:

```bash
sudo mkdir -p /root/bin
sudo ossutil -e oss-cn-shenzhen.aliyuncs.com \
  cp oss://update-cpa-plus/tonkic-api/update.sh /root/bin/update-tonkic-api
sudo chmod 700 /root/bin/update-tonkic-api
```

After this one-time download, updates connect directly to GitHub and no longer
use `ossutil`. Run an update manually:

```bash
sudo /root/bin/update-tonkic-api
```

The script is tailored to the existing deployment:

- application directory: `/root/new-api`;
- binary: `/root/new-api/new-api`;
- SQLite database: `/root/new-api/one-api.db`;
- environment: `/root/new-api/.env`;
- tmux session: `new-api`;
- health check: `http://127.0.0.1:3000/api/status`;
- backups: `/root/new-api-backups`.

Before replacement it creates both a full directory archive and a consistent
SQLite online backup. It stops only the process whose executable resolves to
`/root/new-api/new-api`, replaces only that binary, restarts the tmux session,
and waits for the health endpoint. If startup fails, it automatically restores
the old binary and SQLite database and starts the old version.

No source checkout, systemd service conversion, `.env` rewrite, database
deletion, or directory replacement is performed.

## Optional cron schedule

To check at 04:00 every day while keeping only one updater script:

```bash
(crontab -l 2>/dev/null; echo '0 4 * * * /root/bin/update-tonkic-api >> /root/new-api/logs/github-update.log 2>&1') | crontab -
```
