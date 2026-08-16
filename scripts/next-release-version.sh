#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
upstream_version="$(tr -d '[:space:]' < "$repo_root/UPSTREAM_VERSION")"

if [[ ! "$upstream_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-[0-9A-Za-z.-]+$ ]]; then
  echo "Invalid upstream version in UPSTREAM_VERSION: $upstream_version" >&2
  exit 1
fi

latest_revision="$({
  git -C "$repo_root" tag --list "${upstream_version}.*" |
    sed -n "s/^${upstream_version//./\\.}\.\([0-9][0-9]*\)$/\1/p"
} | sort -n | tail -n 1)"

if [[ -z "$latest_revision" ]]; then
  latest_revision=0
fi

printf '%s.%d\n' "$upstream_version" "$((latest_revision + 1))"
