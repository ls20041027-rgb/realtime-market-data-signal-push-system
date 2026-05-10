#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="/tmp/ts_runs"
LOG_DIR="${RUN_DIR}"
COMPOSE_FILE="${REPO_ROOT}/deployments/docker-compose.yml"
COMPOSE_PROJECT="tornado-seeker"

STREAM_ENGINE_DIR="${REPO_ROOT}/services/stream_engine"
PUSH_GATEWAY_BIN="${REPO_ROOT}/services/push_gateway/push_gateway"
FRONTEND_DIR="${REPO_ROOT}/frontend"

CONDA_ENV="stock"

mkdir -p "${RUN_DIR}"

HOST_LAN_IP="${HOST_LAN_IP:-$(ipconfig getifaddr en0 2>/dev/null || true)}"
if [[ -z "${HOST_LAN_IP}" ]]; then
  HOST_LAN_IP="$(ipconfig getifaddr en1 2>/dev/null || true)"
fi
if [[ -z "${HOST_LAN_IP}" ]]; then
  echo "[FATAL] cannot detect LAN IP from en0/en1, please export HOST_LAN_IP=x.x.x.x" >&2
  exit 1
fi
export KAFKA_ADVERTISED_LISTENERS="PLAINTEXT://${HOST_LAN_IP}:9092"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }

ensure_docker_host() {
  if [[ -n "${DOCKER_HOST:-}" ]]; then return 0; fi
  local ep
  ep="$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)"
  if [[ -n "${ep}" && "${ep}" != "unix:///var/run/docker.sock" ]]; then
    export DOCKER_HOST="${ep}"
    log "DOCKER_HOST=${DOCKER_HOST} (from docker context)"
    return 0
  fi
  if [[ -S "${HOME}/.colima/default/docker.sock" ]]; then
    export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
    log "DOCKER_HOST=${DOCKER_HOST} (colima fallback)"
    return 0
  fi
  if [[ -S /var/run/docker.sock ]]; then
    export DOCKER_HOST="unix:///var/run/docker.sock"
    log "DOCKER_HOST=${DOCKER_HOST}"
    return 0
  fi
  log "[FATAL] no docker socket found; start Docker Desktop or 'colima start' first"
  exit 1
}

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" "$@"
  else
    docker-compose -f "${COMPOSE_FILE}" -p "${COMPOSE_PROJECT}" "$@"
  fi
}

ensure_port_free() {
  local port="$1" label="$2"
  local pids
  pids="$(lsof -nP -iTCP:"${port}" -sTCP:LISTEN -t 2>/dev/null || true)"
  if [[ -z "${pids}" ]]; then return 0; fi

  if docker ps --filter "publish=${port}" --filter "name=ts-" --format '{{.Names}}' 2>/dev/null | grep -q '^ts-'; then
    log "port ${port} (${label}) already served by ts-* container, skip"
    return 0
  fi

  local pidfile_name="${label%_vite}"
  local own_pidfile="${RUN_DIR}/${pidfile_name}.pid"
  if [[ -f "${own_pidfile}" ]]; then
    local own_pid
    own_pid="$(cat "${own_pidfile}" 2>/dev/null || true)"
    if [[ -n "${own_pid}" ]] && kill -0 "${own_pid}" 2>/dev/null; then
      log "port ${port} (${label}) already held by our ${pidfile_name} (pid=${own_pid}), skip"
      return 0
    fi
  fi

  local owners
  owners="$(ps -p ${pids} -o comm= 2>/dev/null || true)"
  if [[ "${port}" == "9092" && "${owners}" == *ssh* ]]; then
    log "killing stale ssh tunnel on :9092 (pids=${pids})"
    kill ${pids} 2>/dev/null || true
    sleep 1
    pids="$(lsof -nP -iTCP:"${port}" -sTCP:LISTEN -t 2>/dev/null || true)"
  fi

  if [[ -n "${pids}" ]]; then
    log "[FATAL] port ${port} (${label}) is occupied by pid(s) ${pids} (${owners}); free it then retry"
    exit 1
  fi
}

wait_tcp() {
  local host="$1" port="$2" label="$3" timeout="${4:-60}"
  local t=0
  while (( t < timeout )); do
    if nc -z "${host}" "${port}" >/dev/null 2>&1; then
      log "${label} is up on ${host}:${port}"
      return 0
    fi
    sleep 1
    t=$((t + 1))
  done
  log "[FATAL] ${label} (${host}:${port}) not ready after ${timeout}s"
  exit 1
}

start_bg() {
  local name="$1" pidfile="${RUN_DIR}/${1}.pid" logfile="${LOG_DIR}/${1}.log"
  shift
  if [[ -f "${pidfile}" ]] && kill -0 "$(cat "${pidfile}")" 2>/dev/null; then
    log "${name} already running (pid=$(cat "${pidfile}")), skip"
    return 0
  fi
  log "starting ${name} ..."
  ( nohup "$@" >"${logfile}" 2>&1 & echo $! >"${pidfile}" )
  sleep 1
  if kill -0 "$(cat "${pidfile}")" 2>/dev/null; then
    log "${name} started (pid=$(cat "${pidfile}"), log=${logfile})"
  else
    log "[FATAL] ${name} failed to start, tail of log:"
    tail -n 40 "${logfile}" >&2 || true
    exit 1
  fi
}

log "repo=${REPO_ROOT}"
log "HOST_LAN_IP=${HOST_LAN_IP}  KAFKA_ADVERTISED_LISTENERS=${KAFKA_ADVERTISED_LISTENERS}"
ensure_docker_host
ensure_port_free 9092 kafka
ensure_port_free 5432 postgres
ensure_port_free 6379 redis
ensure_port_free 8080 push_gateway
ensure_port_free 5173 frontend_vite

log "bringing up infra (kafka/zookeeper/redis/postgres) via docker compose ..."
dc up -d

wait_tcp 127.0.0.1 9092 kafka 90
wait_tcp 127.0.0.1 6379 redis 30
wait_tcp 127.0.0.1 5432 postgres 120

log "waiting for postgres schema to be initialized ..."
for i in $(seq 1 60); do
  if docker exec ts-postgres psql -U postgres -d tornado_seeker -c "\dt" 2>/dev/null | grep -qi stock; then
    log "postgres schema is ready"
    break
  fi
  sleep 2
  if [[ "${i}" == "60" ]]; then
    log "[WARN] postgres schema not detected after 120s, continue anyway (may be the first boot)"
  fi
done

log "ensuring kafka topics exist ..."
TS_KAFKA_TOPICS=(market.report market.fenbi market.mkttbl market.finance market.filedata.history market.filedata.minute market.filedata.5minute market.filedata.power market_data_normalized trading_signals system_events)
TS_KAFKA_TOPIC_PARTS=(1 1 1 1 1 1 1 1 1 1 1)
for i in "${!TS_KAFKA_TOPICS[@]}"; do
  topic="${TS_KAFKA_TOPICS[$i]}"
  parts="${TS_KAFKA_TOPIC_PARTS[$i]}"
  if docker exec ts-kafka kafka-topics \
      --bootstrap-server 127.0.0.1:9092 \
      --create --if-not-exists \
      --topic "${topic}" \
      --partitions "${parts}" --replication-factor 1 >/dev/null 2>&1; then
    log "  kafka topic ready: ${topic} (partitions=${parts})"
  else
    log "[FATAL] failed to create kafka topic: ${topic}"
    exit 1
  fi
  cur_parts="$(docker exec ts-kafka kafka-topics \
      --bootstrap-server 127.0.0.1:9092 \
      --describe --topic "${topic}" 2>/dev/null \
      | awk -F'PartitionCount: ' 'NR==1{print $2}' | awk '{print $1}')"
  if [[ -n "${cur_parts}" && "${cur_parts}" -lt "${parts}" ]]; then
    log "  topic ${topic} has ${cur_parts} partitions, altering to ${parts} ..."
    docker exec ts-kafka kafka-topics \
      --bootstrap-server 127.0.0.1:9092 \
      --alter --topic "${topic}" --partitions "${parts}" >/dev/null 2>&1 || {
        log "[FATAL] failed to alter partitions for ${topic}"; exit 1; }
  fi
done

start_bg stream_engine \
  conda run --no-capture-output -n "${CONDA_ENV}" \
  python "${STREAM_ENGINE_DIR}/main.py"

export PUSH_GATEWAY_POSTGRES_DATABASE="tornado_seeker"

log "building push_gateway binary ..."
( cd "${REPO_ROOT}/services/push_gateway" && go build -o push_gateway . )
if [[ ! -x "${PUSH_GATEWAY_BIN}" ]]; then
  log "[FATAL] push_gateway binary not found at ${PUSH_GATEWAY_BIN} after go build"
  exit 1
fi
start_bg push_gateway "${PUSH_GATEWAY_BIN}"
wait_tcp 127.0.0.1 8080 push_gateway_http 30

if [[ ! -d "${FRONTEND_DIR}/node_modules" ]]; then
  log "node_modules missing, running pnpm/npm install ..."
  if command -v pnpm >/dev/null 2>&1; then
    ( cd "${FRONTEND_DIR}" && pnpm install )
  else
    ( cd "${FRONTEND_DIR}" && npm install )
  fi
fi

start_bg frontend \
  bash -lc "cd '${FRONTEND_DIR}' && (command -v pnpm >/dev/null 2>&1 && pnpm dev --host 0.0.0.0 --port 5173 || npm run dev -- --host 0.0.0.0 --port 5173)"

wait_tcp 127.0.0.1 5173 frontend_vite 60

cat <<EOF

============================================================
 tornado-seeker 全部组件已启动
============================================================
  Kafka        : ${HOST_LAN_IP}:9092  (advertised for data_ingestion)
  Redis        : 127.0.0.1:6379
  PostgreSQL    : 127.0.0.1:5432  (db=tornado_seeker, user=postgres, pwd=empty)
  Stream Engine: pid=$(cat "${RUN_DIR}/stream_engine.pid" 2>/dev/null || echo '?')  (实时处理层，Python/pathway)
  Push Gateway : http://127.0.0.1:8080  ws://127.0.0.1:8080/ws
  Frontend     : http://127.0.0.1:5173
------------------------------------------------------------
  logs  : ${LOG_DIR}/{stream_engine,push_gateway,frontend}.log
  pids  : ${RUN_DIR}/*.pid
  stop  : bash scripts/stop_all.sh
============================================================
EOF
