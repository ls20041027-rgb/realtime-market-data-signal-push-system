#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="/tmp/ts_runs"
COMPOSE_FILE="${REPO_ROOT}/deployments/docker-compose.yml"
COMPOSE_PROJECT="tornado-seeker"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }

ensure_docker_host() {
  if [[ -n "${DOCKER_HOST:-}" ]]; then return 0; fi
  local ep
  ep="$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)"
  if [[ -n "${ep}" && "${ep}" != "unix:///var/run/docker.sock" ]]; then
    export DOCKER_HOST="${ep}"; return 0
  fi
  if [[ -S "${HOME}/.colima/default/docker.sock" ]]; then
    export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"; return 0
  fi
  if [[ -S /var/run/docker.sock ]]; then
    export DOCKER_HOST="unix:///var/run/docker.sock"; return 0
  fi
  log "[WARN] no docker socket found; compose commands will fail"
}
ensure_docker_host

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" "$@"
  else
    docker-compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" "$@"
  fi
}

stop_bg() {
  local name="$1" match_pattern="${2:-}" pidfile="${RUN_DIR}/${1}.pid"
  if [[ -f "${pidfile}" ]]; then
    local pid
    pid="$(cat "${pidfile}")"
    if kill -0 "${pid}" 2>/dev/null; then
      log "stopping ${name} (pid=${pid}) ..."
      kill "${pid}" 2>/dev/null || true
      for i in 1 2 3 4 5; do
        if ! kill -0 "${pid}" 2>/dev/null; then break; fi
        sleep 1
      done
      if kill -0 "${pid}" 2>/dev/null; then
        log "${name} did not exit, sending SIGKILL"
        kill -9 "${pid}" 2>/dev/null || true
      fi
      pkill -P "${pid}" 2>/dev/null || true
    else
      log "${name} already stopped"
    fi
    rm -f "${pidfile}"
  else
    log "${name} pidfile missing, will try pattern-match fallback"
  fi

  if [[ -n "${match_pattern}" ]]; then
    local orphan_pids
    orphan_pids="$(pgrep -f "${match_pattern}" 2>/dev/null || true)"
    if [[ -n "${orphan_pids}" ]]; then
      log "killing orphan ${name} procs (pattern=${match_pattern}): ${orphan_pids}"
      kill ${orphan_pids} 2>/dev/null || true
      sleep 1
      kill -9 ${orphan_pids} 2>/dev/null || true
    fi
  fi
}

stop_bg frontend         'node .*frontend/.*vite'
stop_bg push_gateway     'services/push_gateway/push_gateway'
stop_bg stream_engine    'services/stream_engine/main\.py'

vite_pids="$(lsof -nP -iTCP:5173 -sTCP:LISTEN -t 2>/dev/null || true)"
if [[ -n "${vite_pids}" ]]; then
  log "killing leftover vite on :5173 (pids=${vite_pids})"
  kill ${vite_pids} 2>/dev/null || true
fi

if [[ "${1:-}" == "--purge" ]]; then
log "compose down -v (WILL DELETE redis/postgres volumes)"
  dc down -v
else
  log "compose down (keep volumes)"
  dc down
fi

log "done."
