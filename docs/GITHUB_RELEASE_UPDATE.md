# Tonkic API updates through GitHub Releases

The server updater downloads release metadata, Linux binaries, and checksums
directly from:

```text
https://github.com/Tonkic/Tonkic-api/releases
```

It does not require Alibaba Cloud OSS, `ossutil`, or OSS credentials.

## One-time server setup

Download the stable `update.sh` asset from the latest GitHub Release:

```bash
sudo mkdir -p /root/bin
sudo curl -fL \
  https://github.com/Tonkic/Tonkic-api/releases/latest/download/update.sh \
  -o /root/bin/update-tonkic-api
sudo chmod 700 /root/bin/update-tonkic-api
```

Run an update manually:

```bash
sudo /root/bin/update-tonkic-api
```

The updater follows GitHub's public `/releases/latest` redirect and downloads
the matching assets directly. It does not use the GitHub API and therefore
does not require an API token or consume API rate limits.

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

The script automatically selects amd64 or arm64, verifies the published
SHA-256 checksum, creates a consistent SQLite online backup, replaces only the
binary, restarts the detected service, and checks the health endpoint. If
startup fails, it restores the previous binary and database. Release binaries
are statically linked, so a Docker container is not required to bridge host
glibc versions.

It does not check out source code, rewrite `.env`, delete the database, or
replace the application directory.

## Optional daily update

```bash
(crontab -l 2>/dev/null; echo '0 4 * * * /root/bin/update-tonkic-api >> /root/new-api/logs/github-update.log 2>&1') | crontab -
```
