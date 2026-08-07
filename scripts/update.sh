#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

# Tonkic API updater for the existing /root/new-api deployment.
# Binaries and release metadata are downloaded only from Alibaba Cloud OSS.
oss_bucket="update-cpa-plus"
oss_endpoint="oss-cn-shenzhen.aliyuncs.com"
oss_prefix="tonkic-api/releases/latest"
app_dir="/root/new-api"
binary="$app_dir/new-api"
database="$app_dir/one-api.db"
tmux_session="new-api"
systemd_service="new-api.service"
backup_dir="/root/new-api-backups"
health_url="http://127.0.0.1:3000/api/status"
lock_file="/var/lock/tonkic-api-update.lock"

timestamp=$(date +%Y%m%d-%H%M%S)
tmp_dir=""
database_backup=""
binary_backup=""
replacement_started=false
rollback_running=false
deployment_backend=""

log() {
  printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

detect_deployment_backend() {
  if command -v systemctl >/dev/null 2>&1 \
    && systemctl cat "$systemd_service" >/dev/null 2>&1; then
    deployment_backend="systemd"
    return
  fi
  if command -v tmux >/dev/null 2>&1 \
    && tmux has-session -t "$tmux_session" 2>/dev/null; then
    deployment_backend="tmux"
    return
  fi
  log "Neither $systemd_service nor tmux session '$tmux_session' is installed."
  return 1
}

stop_app() {
  case "$deployment_backend" in
    systemd) systemctl stop "$systemd_service" ;;
    tmux) tmux kill-session -t "$tmux_session" ;;
    *) log "Unknown deployment backend: $deployment_backend"; return 1 ;;
  esac
}

start_app() {
  case "$deployment_backend" in
    systemd) systemctl start "$systemd_service" ;;
    tmux) tmux new-session -d -s "$tmux_session" -c "$app_dir" "./new-api" ;;
    *) log "Unknown deployment backend: $deployment_backend"; return 1 ;;
  esac
}

wait_for_health() {
  local attempt
  for attempt in {1..30}; do
    if curl --fail --silent --show-error --max-time 3 "$health_url" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  return 1
}

rollback() {
  local original_exit=$?
  trap - ERR
  if $rollback_running; then
    exit "$original_exit"
  fi
  rollback_running=true
  log "Update failed; starting automatic rollback."

  if $replacement_started && [[ -f $binary_backup && -f $database_backup ]]; then
    stop_app || true
    install -m 0755 "$binary_backup" "$binary"
    cp -a "$database_backup" "$database"
    start_app
    if wait_for_health; then
      log "Rollback succeeded. Old binary and SQLite database were restored."
    else
      log "Rollback files were restored, but the service is still unhealthy."
    fi
  else
    log "The running installation was not changed."
  fi
  exit "$original_exit"
}

cleanup() {
  [[ -z $tmp_dir ]] || rm -rf -- "$tmp_dir"
}
trap rollback ERR
trap cleanup EXIT

if [[ $EUID -ne 0 ]]; then
  log "Run this script as root."
  exit 1
fi
for command_name in curl flock python3 sha256sum; do
  command -v "$command_name" >/dev/null 2>&1 || {
    log "Required command is unavailable: $command_name"
    exit 1
  }
done
[[ -x $binary ]] || { log "Binary is missing: $binary"; exit 1; }
[[ -f $database ]] || { log "SQLite database is missing: $database"; exit 1; }
[[ -f $app_dir/.env ]] || { log "Environment file is missing: $app_dir/.env"; exit 1; }
detect_deployment_backend
log "Detected deployment backend: $deployment_backend."

exec 9>"$lock_file"
if ! flock -n 9; then
  log "Another update is already running."
  exit 0
fi

case $(uname -m) in
  x86_64|amd64) architecture="amd64" ;;
  aarch64|arm64) architecture="arm64" ;;
  *) log "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

ossutil_bin="${OSSUTIL_BIN:-/usr/local/bin/ossutil}"
if [[ ! -x $ossutil_bin ]]; then
  log "ossutil is required. Install/configure it before updating."
  exit 1
fi

if [[ $architecture == "arm64" ]]; then
  asset="new-api-linux-arm64"
else
  asset="new-api-linux-amd64"
fi

tmp_dir=$(mktemp -d)
oss_base="oss://${oss_bucket}/${oss_prefix}"
ossutil_args=(-e "$oss_endpoint")
if [[ -n ${OSS_ACCESS_KEY_ID:-} && -n ${OSS_ACCESS_KEY_SECRET:-} ]]; then
  ossutil_args+=(-i "$OSS_ACCESS_KEY_ID" -k "$OSS_ACCESS_KEY_SECRET")
fi

log "Downloading $asset from $oss_base."
"$ossutil_bin" "${ossutil_args[@]}" cp -f "$oss_base/$asset" "$tmp_dir/$asset"
"$ossutil_bin" "${ossutil_args[@]}" cp -f "$oss_base/checksums-latest.txt" "$tmp_dir/checksums-latest.txt"
"$ossutil_bin" "${ossutil_args[@]}" cp -f "$oss_base/version.txt" "$tmp_dir/version.txt"

expected=$(awk -v file="$asset" '
  $2 == file { count += 1; checksum = $1 }
  END { if (count == 1) print checksum }
' "$tmp_dir/checksums-latest.txt")
if [[ ! $expected =~ ^[0-9a-fA-F]{64}$ ]]; then
  log "A unique valid checksum for $asset was not found."
  exit 1
fi
actual=$(sha256sum "$tmp_dir/$asset" | awk '{print $1}')
if [[ $actual != "$expected" ]]; then
  log "Checksum verification failed for $asset."
  exit 1
fi
log "Checksum verification succeeded for $asset from OSS."

target_version=$(tr -d '\r\n' < "$tmp_dir/version.txt")
if [[ ! $target_version =~ ^[A-Za-z0-9._-]+$ ]]; then
  log "OSS returned an invalid release version: $target_version"
  exit 1
fi

chmod 0755 "$tmp_dir/$asset"
if ! downloaded_version=$("$tmp_dir/$asset" --version 2>&1); then
  log "The downloaded binary cannot run on this host: $downloaded_version"
  log "Refusing to stop the healthy service. Publish a statically linked Linux release first."
  exit 1
fi
log "Downloaded binary passed the host compatibility check: $downloaded_version."

current_sha=$(sha256sum "$binary" | awk '{print $1}')
target_sha=$actual
if [[ $current_sha == "$target_sha" ]]; then
  printf '%s\n' "$target_version" > "$app_dir/.release-version"
  log "Already running $target_version ($target_sha); nothing to update."
  exit 0
fi

install -d -m 0700 "$backup_dir"
database_backup="$backup_dir/one-api.db.backup-$timestamp-before-update"
log "Creating a consistent SQLite backup at $database_backup."
DATABASE_PATH="$database" BACKUP_PATH="$database_backup" python3 <<'PY'
import os
import sqlite3

source = sqlite3.connect(os.environ["DATABASE_PATH"])
destination = sqlite3.connect(os.environ["BACKUP_PATH"])
try:
    source.backup(destination)
finally:
    destination.close()
    source.close()
PY

binary_backup="$backup_dir/new-api.prev-$timestamp"
cp -a "$binary" "$binary_backup"

log "Stopping new-api through $deployment_backend."
stop_app
replacement_started=true
install -m 0755 "$tmp_dir/$asset" "$binary.new"
mv -f "$binary.new" "$binary"

log "Starting new-api $target_version."
start_app
wait_for_health
replacement_started=false
printf '%s\n' "$target_version" > "$app_dir/.release-version"
log "Update succeeded: $current_sha -> $target_sha ($target_version)."

# Keep the ten newest pairs of database/binary backups.
find "$backup_dir" -maxdepth 1 -type f \( -name 'one-api.db.backup-*-before-update' -o -name 'new-api.prev-*' \) -printf '%T@ %p\n' \
  | sort -nr | tail -n +21 | cut -d' ' -f2- | xargs -r rm -f --
