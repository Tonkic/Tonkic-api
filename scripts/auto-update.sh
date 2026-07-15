#!/usr/bin/env bash
set -Eeuo pipefail

branch="main"
pull_request="5062"
rebuild_docker=false

usage() {
  cat <<'EOF'
Usage: auto-update.sh [options]

Options:
  --branch NAME          Tonkic branch to update (default: main)
  --pull-request NUMBER  QuantumNous/new-api pull request (default: 5062)
  --rebuild-docker       Build the merged source and restart the Compose service
  -h, --help             Show this help
EOF
}

while (($# > 0)); do
  case "$1" in
    --branch)
      branch="${2:?--branch requires a value}"
      shift 2
      ;;
    --pull-request)
      pull_request="${2:?--pull-request requires a value}"
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

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/.." && pwd)
log_directory="$repo_root/logs"
log_file="$log_directory/auto-update.log"
lock_file="$repo_root/.git/auto-update.lock"
merge_started=false

mkdir -p -- "$log_directory"

log() {
  printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" | tee -a "$log_file"
}

cleanup_on_error() {
  local exit_code=$?
  trap - ERR
  if $merge_started && git -C "$repo_root" rev-parse -q --verify MERGE_HEAD >/dev/null 2>&1; then
    git -C "$repo_root" merge --abort || true
  fi
  log "Update failed with exit code $exit_code."
  exit "$exit_code"
}
trap cleanup_on_error ERR

for command_name in git flock; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    log "Required command is unavailable: $command_name"
    exit 1
  fi
done

if [[ ! -d "$repo_root/.git" ]]; then
  log "Not a Git repository: $repo_root"
  exit 1
fi

exec 9>"$lock_file"
if ! flock -n 9; then
  log "Another update is already running; exiting."
  exit 0
fi

cd -- "$repo_root"
if [[ -n $(git status --porcelain) ]]; then
  log "Working tree is not clean. Commit or stash changes before automatic updating."
  exit 1
fi

if ! git remote get-url upstream >/dev/null 2>&1; then
  log "Adding QuantumNous/new-api as the upstream remote."
  git remote add upstream https://github.com/QuantumNous/new-api.git
fi

log "Fetching Tonkic/$branch."
git fetch --prune origin "$branch"
git checkout "$branch"
merge_started=true
git merge --no-edit "origin/$branch"
merge_started=false

log "Fetching QuantumNous/new-api pull request #$pull_request."
git fetch upstream "pull/$pull_request/head:refs/remotes/upstream/pr-$pull_request" --force
merge_started=true
git merge --no-edit "refs/remotes/upstream/pr-$pull_request"
merge_started=false

if $rebuild_docker; then
  if ! command -v docker >/dev/null 2>&1; then
    log "Docker is not installed or unavailable in PATH."
    exit 1
  fi

  log "Building the merged source as calciumion/new-api:latest."
  docker build --tag calciumion/new-api:latest .
  docker compose up -d --no-deps --force-recreate new-api
fi

log "Update completed at commit $(git rev-parse --short HEAD)."
