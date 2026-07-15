#!/usr/bin/env bash
set -Eeuo pipefail

# Tonkic API updater for the existing /root/new-api tmux deployment.
# This script only reads from oss://update-cpa-plus/tonkic-api/.
oss_bucket="update-cpa-plus"
oss_endpoint="oss-cn-shenzhen-internal.aliyuncs.com"
oss_prefix="tonkic-api/releases/latest"
app_dir="/root/new-api"
binary="$app_dir/new-api"
database="$app_dir/one-api.db"
tmux_session="new-api"
backup_dir="/root/new-api-backups"
health_url="http://127.0.0.1:3000/api/status"
ossutil_bin="/usr/local/bin/ossutil"
lock_file="/var/lock/tonkic-api-update.lock"

timestamp=$(date +%Y%m%d-%H%M%S)
tmp_dir=""
database_backup=""
binary_backup=""
replacement_started=false
rollback_running=false

log() {
  printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

find_app_pids() {
  local proc_exe resolved
  for proc_exe in /proc/[0-9]*/exe; do
    resolved=$(readlink -f "$proc_exe" 2>/dev/null || true)
    if [[ $resolved == "$binary" ]]; then
      printf '%s\n' "${proc_exe#/proc/}" | cut -d/ -f1
    fi
  done
}

stop_app() {
  local pids deadline
  if tmux has-session -t "$tmux_session" 2>/dev/null; then
    tmux send-keys -t "$tmux_session" C-c
  fi

  deadline=$((SECONDS + 30))
  while [[ -n $(find_app_pids) && $SECONDS -lt $deadline ]]; do
    sleep 1
  done

  pids=$(find_app_pids)
  if [[ -n $pids ]]; then
    log "Graceful shutdown timed out; sending SIGTERM to new-api only: $pids"
    # shellcheck disable=SC2086
    kill $pids
    deadline=$((SECONDS + 10))
    while [[ -n $(find_app_pids) && $SECONDS -lt $deadline ]]; do
      sleep 1
    done
  fi

  if [[ -n $(find_app_pids) ]]; then
    log "new-api did not stop; refusing to replace the binary."
    return 1
  fi
  tmux kill-session -t "$tmux_session" 2>/dev/null || true
}

start_app() {
  tmux new-session -d -s "$tmux_session" -c "$app_dir" "./new-api"
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
for command_name in curl flock python3 sha256sum tar tmux; do
  command -v "$command_name" >/dev/null 2>&1 || {
    log "Required command is unavailable: $command_name"
    exit 1
  }
done
[[ -x $binary ]] || { log "Binary is missing: $binary"; exit 1; }
[[ -f $database ]] || { log "SQLite database is missing: $database"; exit 1; }
[[ -f $app_dir/.env ]] || { log "Environment file is missing: $app_dir/.env"; exit 1; }

exec 9>"$lock_file"
if ! flock -n 9; then
  log "Another update is already running."
  exit 0
fi

[[ -x $ossutil_bin ]] || {
  log "ossutil is missing: $ossutil_bin. Install and configure it before updating."
  exit 1
}

case $(uname -m) in
  x86_64|amd64) asset="new-api-linux-amd64" ;;
  aarch64|arm64) asset="new-api-linux-arm64" ;;
  *) log "Unsupported architecture: $(uname -m)"; exit 1 ;;
esac

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

(
  cd "$tmp_dir"
  expected=$(awk -v file="$asset" '$2 == file { print $1 }' checksums-latest.txt)
  [[ -n $expected ]] || { log "Checksum for $asset is missing."; exit 1; }
  printf '%s  %s\n' "$expected" "$asset" | sha256sum --check --strict -
)

current_sha=$(sha256sum "$binary" | awk '{print $1}')
target_sha=$(sha256sum "$tmp_dir/$asset" | awk '{print $1}')
target_version=$(tr -d '\r\n' < "$tmp_dir/version.txt")
if [[ $current_sha == "$target_sha" ]]; then
  log "Already running $target_version ($target_sha); nothing to update."
  exit 0
fi

mkdir -p -- "$backup_dir"
log "Creating a full pre-update archive."
tar -C /root -czf "$backup_dir/new-api-backup-$timestamp.tar.gz" new-api

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

log "Stopping only the new-api process in tmux session '$tmux_session'."
stop_app
replacement_started=true
install -m 0755 "$tmp_dir/$asset" "$binary.new"
mv -f "$binary.new" "$binary"

log "Starting new-api $target_version."
start_app
wait_for_health
replacement_started=false
printf '%s\n' "$target_version" > "$app_dir/.oss-version"
log "Update succeeded: $current_sha -> $target_sha ($target_version)."

# Keep the five newest full archives and ten newest database/binary backups.
find "$backup_dir" -maxdepth 1 -type f -name 'new-api-backup-*.tar.gz' -printf '%T@ %p\n' \
  | sort -nr | tail -n +6 | cut -d' ' -f2- | xargs -r rm -f --
find "$backup_dir" -maxdepth 1 -type f \( -name 'one-api.db.backup-*-before-update' -o -name 'new-api.prev-*' \) -printf '%T@ %p\n' \
  | sort -nr | tail -n +21 | cut -d' ' -f2- | xargs -r rm -f --
