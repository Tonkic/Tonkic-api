# Tonkic API updates through Alibaba Cloud OSS

The server updater downloads the Linux binaries, checksums, and version
metadata only from Alibaba Cloud OSS. It does not download update files from
GitHub Releases.

## One-time server setup

Configure `ossutil` for the RAM user `power-user-access`, then download the
stable updater from the Shenzhen public endpoint:

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

The updater downloads the architecture-matched binary, checksum, and version
metadata from `oss://update-cpa-plus/tonkic-api/releases/latest/`.

## Existing deployment layout

The updater is tailored to this deployment:

- application directory: `/root/new-api`;
- binary: `/root/new-api/new-api`;
- SQLite database: `/root/new-api/one-api.db`;
- environment: `/root/new-api/.env`;
- preferred service: `new-api.service` under systemd;
- legacy fallback: tmux session `new-api`;
- health check: `http://127.0.0.1:3000/api/status`;
- backups: `/root/new-api-backups`.

The script automatically selects amd64 or arm64, verifies the OSS SHA-256
checksum, creates a consistent SQLite online backup, replaces only the binary,
restarts the detected service, and checks the health endpoint. If startup
fails, it restores the previous binary and database. Release binaries are
statically linked, so a Docker container is not required to bridge host glibc
versions. Before stopping a healthy service, the updater also executes the
downloaded binary's `--version` command and refuses an incompatible binary.

It does not check out source code, download from GitHub, rewrite `.env`, delete
the database, or replace the application directory.

## Optional daily update

```bash
(crontab -l 2>/dev/null; echo '0 4 * * * /root/bin/update-tonkic-api >> /root/new-api/logs/oss-update.log 2>&1') | crontab -
```
