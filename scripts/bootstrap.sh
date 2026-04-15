#!/usr/bin/env bash
set -euo pipefail

REPO_URL_DEFAULT="https://github.com/laukkw/brale-core.git"
REF_DEFAULT="master"
TARGET_DIR_DEFAULT="${HOME}/brale-core"
ONBOARDING_URL_DEFAULT="http://127.0.0.1:9992"

repo_url="${BRALE_REPO_URL:-${REPO_URL_DEFAULT}}"
ref="${BRALE_REF:-${REF_DEFAULT}}"
target_dir="${BRALE_DIR:-${TARGET_DIR_DEFAULT}}"
onboarding_url="${BRALE_ONBOARDING_URL:-${ONBOARDING_URL_DEFAULT}}"
compose_project_name="${BRALE_COMPOSE_PROJECT_NAME:-brale-core}"
open_browser=1
start_onboarding=1
with_mcp=0
run_setup=0
setup_lang=""
host_uid="$(id -u)"
host_gid="$(id -g)"

usage() {
  cat <<'EOF'
Usage: bootstrap.sh [options]

Options:
  --dir PATH          Target checkout directory (default: ~/brale-core)
  --ref REF           Git ref to checkout (default: master)
  --repo-url URL      Repository URL
  --onboarding-url U  Expected onboarding URL (default: http://127.0.0.1:9992)
  --no-onboarding     Skip launching the onboarding UI
  --with-mcp          Start the stack with the optional MCP SSE service
  --setup             Run 'make setup' after clone/update
  --setup-lang LANG   Preselect setup wizard language (zh or en)
  --no-open           Do not try to open the browser automatically
  -h, --help          Show this help text
EOF
}

log() {
  printf '%s\n' "$1"
}

fail() {
  printf '[ERR] %s\n' "$1" >&2
  exit 1
}

require_cmd() {
  local cmd="$1"
  local hint="$2"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    fail "$cmd is required. $hint"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)
      target_dir="$2"
      shift 2
      ;;
    --ref)
      ref="$2"
      shift 2
      ;;
    --repo-url)
      repo_url="$2"
      shift 2
      ;;
    --onboarding-url)
      onboarding_url="$2"
      shift 2
      ;;
    --no-onboarding)
      start_onboarding=0
      shift
      ;;
    --with-mcp)
      with_mcp=1
      shift
      ;;
    --setup)
      run_setup=1
      shift
      ;;
    --setup-lang)
      case "$2" in
        zh|en)
          setup_lang="$2"
          ;;
        *)
          fail "invalid --setup-lang: must be 'zh' or 'en'"
          ;;
      esac
      shift 2
      ;;
    --no-open)
      open_browser=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

require_cmd git "Install Git and retry."
require_cmd docker "Install Docker Desktop / Docker Engine with Compose V2 and retry."

if ! docker compose version >/dev/null 2>&1; then
  fail "docker compose is required. Install Docker Compose V2 and retry."
fi

if ! docker info >/dev/null 2>&1; then
  fail "Docker daemon is not running. Start Docker first."
fi

target_dir="${target_dir/#\~/${HOME}}"
parent_dir="$(dirname "$target_dir")"

if [[ ! -d "$parent_dir" ]]; then
  mkdir -p "$parent_dir"
fi

if [[ ! -d "$target_dir/.git" ]]; then
  log "[INFO] cloning brale-core into $target_dir"
  git clone "$repo_url" "$target_dir"
else
  log "[INFO] reusing existing checkout at $target_dir"
  current_origin="$(git -C "$target_dir" remote get-url origin 2>/dev/null || true)"
  if [[ -z "$current_origin" ]]; then
    fail "existing directory is not a usable git checkout: $target_dir"
  fi
  if [[ "$current_origin" != "$repo_url" && "$current_origin" != git@github.com:laukkw/brale-core.git ]]; then
    fail "existing checkout origin mismatch: $current_origin"
  fi
fi

if [[ -n "$(git -C "$target_dir" status --porcelain)" ]]; then
  log "[WARN] working tree is dirty; skipping git fetch/reset and using local files as-is"
else
  log "[INFO] updating checkout to $ref"
  git -C "$target_dir" fetch --tags origin
  git -C "$target_dir" checkout "$ref"
  if git -C "$target_dir" show-ref --verify --quiet "refs/remotes/origin/$ref"; then
    git -C "$target_dir" pull --ff-only origin "$ref"
  fi
fi

log "[INFO] ensuring .env exists"
env_output="$(cd "$target_dir" && HOST_UID="$host_uid" HOST_GID="$host_gid" COMPOSE_PROJECT_NAME="$compose_project_name" make env-init 2>&1)" || {
  printf '%s\n' "$env_output"
  fail "failed to initialize .env from .env.example"
}
printf '%s\n' "$env_output"

if [[ "$run_setup" -eq 1 ]]; then
  log "[INFO] running interactive setup wizard"
  setup_cmd=(make setup)
  if [[ -n "$setup_lang" ]]; then
    setup_cmd+=("SETUP_LANG=$setup_lang")
  fi
  if ! (cd "$target_dir" && HOST_UID="$host_uid" HOST_GID="$host_gid" COMPOSE_PROJECT_NAME="$compose_project_name" "${setup_cmd[@]}"); then
    fail "setup wizard failed"
  fi
fi

if [[ "$start_onboarding" -ne 1 ]]; then
  log "[INFO] onboarding disabled; checking whether the stack can start headlessly"
  if check_output="$(cd "$target_dir" && HOST_UID="$host_uid" HOST_GID="$host_gid" COMPOSE_PROJECT_NAME="$compose_project_name" make check 2>&1)"; then
    printf '%s\n' "$check_output"
    log "[INFO] starting core stack"
    start_cmd=(make start "ENABLE_ONBOARDING=0")
    if [[ "$with_mcp" -eq 1 ]]; then
      start_cmd+=("ENABLE_MCP=1")
    fi
    start_output="$(cd "$target_dir" && HOST_UID="$host_uid" HOST_GID="$host_gid" COMPOSE_PROJECT_NAME="$compose_project_name" "${start_cmd[@]}" 2>&1)" || {
      printf '%s\n' "$start_output"
      fail "failed to start stack without onboarding"
    }
    printf '%s\n' "$start_output"
    log "[OK] stack started"
    if [[ "$with_mcp" -eq 1 ]]; then
      log "[OPEN] MCP SSE will listen on http://127.0.0.1:8765/sse"
    fi
    exit 0
  fi
  printf '%s\n' "$check_output"
  log "[WARN] .env is not ready for headless startup; skipping docker compose up"
  log "[NEXT] edit .env manually, run 'make setup', or rerun bootstrap without --no-onboarding"
  exit 0
fi

log "[INFO] building and starting onboarding container"
make_output="$(cd "$target_dir" && HOST_UID="$host_uid" HOST_GID="$host_gid" COMPOSE_PROJECT_NAME="$compose_project_name" make init 2>&1)" || {
  printf '%s\n' "$make_output"
  fail "failed to start onboarding via make init"
}
printf '%s\n' "$make_output"

ready=0
for _ in $(seq 1 60); do
  if curl -fsS "$onboarding_url/api/status" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done

if [[ "$ready" -ne 1 ]]; then
  (cd "$target_dir" && HOST_UID="$host_uid" HOST_GID="$host_gid" COMPOSE_PROJECT_NAME="$compose_project_name" docker compose -f docker-compose.yml logs --tail=200 onboarding) || true
  fail "onboarding did not become ready in time"
fi

log "[OK] onboarding is ready"
log "[OPEN] $onboarding_url"
if [[ "$with_mcp" -eq 1 ]]; then
  log "[INFO] MCP SSE is not started during onboarding-only bootstrap. After filling .env, run: make start ENABLE_MCP=1"
fi

if [[ "$open_browser" -eq 1 ]]; then
  if command -v open >/dev/null 2>&1; then
    open "$onboarding_url" >/dev/null 2>&1 || true
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$onboarding_url" >/dev/null 2>&1 || true
  fi
fi
