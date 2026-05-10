#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }

if [[ -z "${DOCKER_HOST:-}" ]]; then
  ep="$(docker context inspect --format '{{.Endpoints.docker.Host}}' 2>/dev/null || true)"
  if [[ -n "${ep}" && "${ep}" != "unix:///var/run/docker.sock" ]]; then
    export DOCKER_HOST="${ep}"
  elif [[ -S "${HOME}/.colima/default/docker.sock" ]]; then
    export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
  elif [[ -S /var/run/docker.sock ]]; then
    export DOCKER_HOST="unix:///var/run/docker.sock"
  fi
fi

log "Flushing Redis ..."
docker exec ts-redis redis-cli FLUSHALL >/dev/null
log "Redis: done"

log "Truncating PostgreSQL tables ..."
docker exec ts-postgres psql -U postgres -d tornado_seeker -c "
DO \$\$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT tablename FROM pg_tables
        WHERE schemaname = 'public'
    LOOP
        EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END
\$\$;
" >/dev/null
log "PostgreSQL: all tables truncated"

TOPICS=(market.report market.fenbi market.mkttbl market.finance market.filedata.history market.filedata.minute market.filedata.5minute market.filedata.power market_data_normalized trading_signals system_events)
PARTS=(1 1 1 1 1 1 1 1 1 1 1)

log "Purging Kafka topics ..."
for i in "${!TOPICS[@]}"; do
  topic="${TOPICS[$i]}"
  parts="${PARTS[$i]}"
  docker exec ts-kafka kafka-topics \
    --bootstrap-server 127.0.0.1:9092 \
    --delete --topic "${topic}" 2>/dev/null || true
done

sleep 3

for i in "${!TOPICS[@]}"; do
  topic="${TOPICS[$i]}"
  parts="${PARTS[$i]}"
  docker exec ts-kafka kafka-topics \
    --bootstrap-server 127.0.0.1:9092 \
    --create --if-not-exists \
    --topic "${topic}" \
    --partitions "${parts}" --replication-factor 1 >/dev/null 2>&1
  log "  Kafka topic recreated: ${topic} (partitions=${parts})"
done
log "Kafka: done"

log "All data purged (Kafka / PostgreSQL / Redis)."
log "You can now restart services via: bash scripts/start_all.sh"
