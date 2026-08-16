#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <release-tag>" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
upstream_version="$(tr -d '[:space:]' < "$repo_root/UPSTREAM_VERSION")"
release_tag="$1"

if [[ ! "$release_tag" =~ ^${upstream_version//./\\.}\.[1-9][0-9]*$ ]]; then
  echo "Release tag must keep the upstream version and append a revision: ${upstream_version}.N" >&2
  exit 1
fi
