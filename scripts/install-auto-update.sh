#!/usr/bin/env bash
set -Eeuo pipefail

schedule="04:00"
rebuild_docker=false
service_name="tonkic-api-auto-update"

usage() {
  cat <<'EOF'
Usage: sudo ./scripts/install-auto-update.sh [options]

Options:
  --time HH:MM       Daily update time (default: 04:00)
  --rebuild-docker   Build and restart the Docker Compose service after updating
  -h, --help         Show this help
EOF
}

while (($# > 0)); do
  case "$1" in
    --time)
      schedule="${2:?--time requires a value}"
      shift 2
      ;;
    --rebuild-docker)
      rebuild_docker=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ $EUID -ne 0 ]]; then
  echo "Run this installer with sudo." >&2
  exit 1
fi
if [[ ! $schedule =~ ^([01][0-9]|2[0-3]):[0-5][0-9]$ ]]; then
  echo "Invalid time '$schedule'; expected HH:MM in 24-hour format." >&2
  exit 2
fi
if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemd is required." >&2
  exit 1
fi

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/.." && pwd)
update_script="$script_dir/auto-update.sh"
run_user="${SUDO_USER:-$(stat -c '%U' "$repo_root")}"
service_file="/etc/systemd/system/$service_name.service"
timer_file="/etc/systemd/system/$service_name.timer"
exec_arguments=""

if [[ ! -x $update_script ]]; then
  echo "Update script is missing or not executable: $update_script" >&2
  exit 1
fi
if $rebuild_docker; then
  exec_arguments=" --rebuild-docker"
fi

cat >"$service_file" <<EOF
[Unit]
Description=Update Tonkic API and merge QuantumNous/new-api PR #5062
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=$run_user
WorkingDirectory="$repo_root"
ExecStart=/bin/bash "$update_script"$exec_arguments
EOF

cat >"$timer_file" <<EOF
[Unit]
Description=Run the Tonkic API automatic updater daily

[Timer]
OnCalendar=*-*-* $schedule:00
Persistent=true
RandomizedDelaySec=5m
Unit=$service_name.service

[Install]
WantedBy=timers.target
EOF

systemctl daemon-reload
systemctl enable --now "$service_name.timer"

echo "Installed $service_name.timer for $schedule daily as user $run_user."
echo "Check it with: systemctl status $service_name.timer"
echo "View logs with: journalctl -u $service_name.service"
