# Tonkic API updates through Alibaba Cloud OSS

Release files are stored only under this prefix:

```text
oss://update-cpa-plus/tonkic-api/
```

The existing `CPA/` prefix is never listed, modified, copied, or deleted.
GitHub Actions and servers use the Shenzhen public endpoint
`oss-cn-shenzhen.aliyuncs.com` because OSS internal endpoints do not work
across Alibaba Cloud regions. The server does not download release binaries
from GitHub.

`power-user-access` is the RAM username, not an AccessKey ID. GitHub Actions
secrets must contain an AccessKey pair created under that RAM user:

- `ALIYUN_OSS_ACCESS_KEY_ID`
- `ALIYUN_OSS_ACCESS_KEY_SECRET`

Repository variables must also be set:

- `ALIYUN_OSS_BUCKET=update-cpa-plus`
- `ALIYUN_OSS_ENDPOINT=oss-cn-shenzhen.aliyuncs.com`

## OSS layout

```text
tonkic-api/update.sh
tonkic-api/releases/<tag>/...
tonkic-api/releases/latest/new-api-linux-amd64
tonkic-api/releases/latest/new-api-linux-arm64
tonkic-api/releases/latest/checksums-latest.txt
tonkic-api/releases/latest/version.txt
```

## Server setup and update

```bash
sudo mkdir -p /root/bin
sudo ossutil -e oss-cn-shenzhen.aliyuncs.com \
  cp oss://update-cpa-plus/tonkic-api/update.sh /root/bin/update-tonkic-api
sudo chmod 700 /root/bin/update-tonkic-api
sudo /root/bin/update-tonkic-api
```

The updater keeps SQLite and binary backups, uses `new-api.service` when
present (otherwise the legacy tmux session), verifies SHA-256, checks health,
and rolls back automatically if the new service is unhealthy.
