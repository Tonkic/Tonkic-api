#!/usr/bin/env bash
set -Eeuo pipefail

: "${OSS_BUCKET:?Set OSS_BUCKET before running the installer}"
: "${OSS_ENDPOINT:?Set OSS_ENDPOINT before running the installer}"

schedule=${UPDATE_SCHEDULE:-*-*-* 04:00:00}
service_name=tonkic-api-oss-update
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
updater=$script_dir/update-from-oss.sh
environment_file=/etc/tonkic-api/oss-update.env

if [[ $EUID -ne 0 ]]; then
  echo "Run this installer with sudo -E." >&2
  exit 1
fi
[[ -x $updater ]] || { echo "Updater is missing or not executable: $updater" >&2; exit 1; }

install -d -m 0750 /etc/tonkic-api
{
  printf 'OSS_BUCKET=%q\n' "$OSS_BUCKET"
  printf 'OSS_ENDPOINT=%q\n' "$OSS_ENDPOINT"
  printf 'INSTALL_PATH=%q\n' "${INSTALL_PATH:-/usr/local/bin/new-api}"
  printf 'SERVICE_NAME=%q\n' "${SERVICE_NAME:-new-api}"
  if [[ -n ${OSS_ACCESS_KEY_ID:-} ]]; then
    printf 'OSS_ACCESS_KEY_ID=%q\n' "$OSS_ACCESS_KEY_ID"
  fi
  if [[ -n ${OSS_ACCESS_KEY_SECRET:-} ]]; then
    printf 'OSS_ACCESS_KEY_SECRET=%q\n' "$OSS_ACCESS_KEY_SECRET"
  fi
} > "$environment_file"
chmod 0600 "$environment_file"

if [[ ! -x /usr/local/bin/ossutil ]]; then
  curl --fail --location --retry 3 \
    --output /usr/local/bin/ossutil \
    https://gosspublic.alicdn.com/ossutil/1.7.19/ossutil64
  chmod 0755 /usr/local/bin/ossutil
fi

cat >/etc/systemd/system/$service_name.service <<EOF
[Unit]
Description=Update Tonkic API from Alibaba Cloud OSS
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
EnvironmentFile=$environment_file
ExecStart=/bin/bash "$updater"
EOF

cat >/etc/systemd/system/$service_name.timer <<EOF
[Unit]
Description=Check Alibaba Cloud OSS for Tonkic API updates

[Timer]
OnCalendar=$schedule
Persistent=true
RandomizedDelaySec=5m
Unit=$service_name.service

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now $service_name.timer
echo "Installed $service_name.timer with schedule: $schedule"
