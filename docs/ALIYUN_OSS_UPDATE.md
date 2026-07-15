# Alibaba Cloud OSS release delivery

GitHub Releases remain the source of truth. After a successful release build,
`.github/workflows/release-to-oss.yml` copies Linux binaries to OSS under both
versioned and stable paths:

```text
oss://BUCKET/releases/v0.1.0/...
oss://BUCKET/releases/latest/new-api-linux-amd64
oss://BUCKET/releases/latest/new-api-linux-arm64
oss://BUCKET/releases/latest/checksums-latest.txt
oss://BUCKET/releases/latest/version.txt
```

## GitHub repository settings

Create these Actions variables under **Settings > Secrets and variables >
Actions > Variables**:

- `ALIYUN_OSS_BUCKET`: bucket name without `oss://`.
- `ALIYUN_OSS_ENDPOINT`: public endpoint such as
  `oss-cn-hangzhou.aliyuncs.com`.

Create these Actions secrets:

- `ALIYUN_OSS_ACCESS_KEY_ID`
- `ALIYUN_OSS_ACCESS_KEY_SECRET`

Use a dedicated RAM user restricted to this bucket and the `releases/` prefix.
Do not use the Alibaba Cloud account owner AccessKey. A minimal custom RAM
policy is:

```json
{
  "Version": "1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["oss:PutObject", "oss:GetObject", "oss:ListObjects"],
      "Resource": [
        "acs:oss:*:*:YOUR_BUCKET",
        "acs:oss:*:*:YOUR_BUCKET/releases/*"
      ]
    }
  ]
}
```

After adding the settings, open **Actions > Sync Release to Alibaba Cloud OSS
> Run workflow**, enter the release tag, and run it once. Later releases sync
automatically.

## ECS server setup

For ECS in the same OSS region, use the internal endpoint, for example
`oss-cn-hangzhou-internal.aliyuncs.com`, to avoid public traffic charges. The
Bucket must be reachable from that ECS instance.

Copy the repository (or at least the two scripts) to the server, then run:

```bash
sudo -E env \
  OSS_BUCKET=YOUR_BUCKET \
  OSS_ENDPOINT=oss-cn-hangzhou-internal.aliyuncs.com \
  OSS_ACCESS_KEY_ID=YOUR_SERVER_RAM_USER_KEY \
  OSS_ACCESS_KEY_SECRET=YOUR_SERVER_RAM_USER_SECRET \
  SERVICE_NAME=new-api \
  INSTALL_PATH=/usr/local/bin/new-api \
  ./scripts/install-oss-updater.sh
```

The installer stores credentials in `/etc/tonkic-api/oss-update.env` with mode
`0600`, installs `ossutil`, and enables a daily systemd timer. Prefer a separate
read-only RAM user for the server, restricted to `GetObject` on
`YOUR_BUCKET/releases/*`.

Check or trigger the updater with:

```bash
systemctl status tonkic-api-oss-update.timer
sudo systemctl start tonkic-api-oss-update.service
journalctl -u tonkic-api-oss-update.service
```

The updater detects amd64/arm64, validates SHA-256, atomically replaces the
binary, restarts the configured service, and verifies that it is active.
