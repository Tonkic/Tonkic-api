#!/usr/bin/env bash
set -Eeuo pipefail

: "${OSS_BUCKET:?Set OSS_BUCKET to the Alibaba Cloud OSS bucket name}"
: "${OSS_ENDPOINT:?Set OSS_ENDPOINT, for example oss-cn-hangzhou-internal.aliyuncs.com}"

install_path=${INSTALL_PATH:-/usr/local/bin/new-api}
service_name=${SERVICE_NAME:-new-api}
oss_prefix=${OSS_PREFIX:-releases/latest}
ossutil_bin=${OSSUTIL_BIN:-/usr/local/bin/ossutil}
lock_file=${LOCK_FILE:-/var/lock/tonkic-api-oss-update.lock}

if [[ $EUID -ne 0 ]]; then
  echo "Run this updater as root (normally through systemd)." >&2
  exit 1
fi
for command_name in flock sha256sum systemctl; do
  command -v "$command_name" >/dev/null 2>&1 || {
    echo "Required command is unavailable: $command_name" >&2
    exit 1
  }
done
if [[ ! -x $ossutil_bin ]]; then
  echo "ossutil is missing or not executable: $ossutil_bin" >&2
  exit 1
fi

exec 9>"$lock_file"
flock -n 9 || exit 0

case $(uname -m) in
  x86_64|amd64) asset=new-api-linux-amd64 ;;
  aarch64|arm64) asset=new-api-linux-arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

tmp_dir=$(mktemp -d)
trap 'rm -rf -- "$tmp_dir"' EXIT
oss_base="oss://${OSS_BUCKET}/${oss_prefix}"

ossutil_args=(-e "$OSS_ENDPOINT")
if [[ -n ${OSS_ACCESS_KEY_ID:-} && -n ${OSS_ACCESS_KEY_SECRET:-} ]]; then
  ossutil_args+=(-i "$OSS_ACCESS_KEY_ID" -k "$OSS_ACCESS_KEY_SECRET")
fi

"$ossutil_bin" "${ossutil_args[@]}" cp -f "$oss_base/$asset" "$tmp_dir/$asset"
"$ossutil_bin" "${ossutil_args[@]}" cp -f "$oss_base/checksums-latest.txt" "$tmp_dir/checksums-latest.txt"
"$ossutil_bin" "${ossutil_args[@]}" cp -f "$oss_base/version.txt" "$tmp_dir/version.txt"

(
  cd "$tmp_dir"
  expected=$(awk -v file="$asset" '$2 == file { print $1 }' checksums-latest.txt)
  [[ -n $expected ]] || { echo "Checksum for $asset is missing" >&2; exit 1; }
  printf '%s  %s\n' "$expected" "$asset" | sha256sum --check --strict -
)

current_version=unknown
if [[ -x $install_path ]]; then
  current_version=$($install_path --version 2>/dev/null | head -n 1 || true)
fi
target_version=$(tr -d '\r\n' < "$tmp_dir/version.txt")
chmod 0755 "$tmp_dir/$asset"
install -m 0755 "$tmp_dir/$asset" "${install_path}.new"
mv -f "${install_path}.new" "$install_path"
systemctl restart "$service_name"
systemctl is-active --quiet "$service_name"
echo "Updated $service_name from $current_version to $target_version."
