# Tonkic API updates through Alibaba Cloud OSS

Release files are stored only under this prefix:

```text
oss://update-cpa-plus/tonkic-api/
```

The existing `CPA/` prefix is never listed, modified, copied, or deleted.
GitHub Actions and the Guangzhou ECS server use the Shenzhen public endpoint
because OSS internal endpoints do not work across Alibaba Cloud regions. The
server connects only to OSS and does not connect to GitHub.

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

The server needs only `update.sh` in addition to the standard `ossutil` client.
Configure `ossutil` once for the RAM user `power-user-access`, using the
Shenzhen public endpoint. Then download the updater:

```bash
sudo mkdir -p /root/bin
sudo ossutil -e oss-cn-shenzhen.aliyuncs.com \
  cp oss://update-cpa-plus/tonkic-api/update.sh /root/bin/update-tonkic-api
sudo chmod 700 /root/bin/update-tonkic-api
```

Run an update manually:

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

No source checkout, GitHub access, systemd service conversion, `.env` rewrite,
database deletion, or directory replacement is performed.

## Optional cron schedule

To check at 04:00 every day while keeping only one updater script:

```bash
(crontab -l 2>/dev/null; echo '0 4 * * * /root/bin/update-tonkic-api >> /root/new-api/logs/oss-update.log 2>&1') | crontab -
```
