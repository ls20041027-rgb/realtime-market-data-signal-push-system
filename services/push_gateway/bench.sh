#!/bin/bash
# Benchmark script for push_gateway API endpoints
# Measures: avg latency, P50, P90, P99 at 500/1000/2000 concurrency for 10s each

set -euo pipefail

HEY=/opt/homebrew/bin/hey
BASE=${1:-"http://127.0.0.1:8080"}
DURATION=10

SYMBOL="SZ000001"
SYMBOLS="SZ000001,SH600000,SZ000002"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

CONCURRENCIES=(500 1000 2000)

ENDPOINTS=(
    "GET|/api/quotes?symbols=${SYMBOLS}|quotes"
    "GET|/api/kline1m/${SYMBOL}|kline1m"
    "GET|/api/kline/${SYMBOL}|kline_daily"
    "GET|/api/fenbi/${SYMBOL}|fenbi"
)

RESULTS_DIR="bench_results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Push Gateway Benchmark Suite${NC}"
echo -e "${GREEN}  Base URL: ${BASE}${NC}"
echo -e "${GREEN}  Duration: ${DURATION}s per test${NC}"
echo -e "${GREEN}  Concurrency levels: ${CONCURRENCIES[*]}${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

SUMMARY_FILE="${RESULTS_DIR}/summary.txt"
printf "%-12s %-6s %-10s %-10s %-10s %-10s %-10s %-8s %-8s\n" \
    "Endpoint" "Conc" "RPS" "Avg(ms)" "P50(ms)" "P90(ms)" "P99(ms)" "Total" "Errors" > "$SUMMARY_FILE"
printf "%-12s %-6s %-10s %-10s %-10s %-10s %-10s %-8s %-8s\n" \
    "--------" "----" "---" "-------" "-------" "-------" "-------" "-----" "------" >> "$SUMMARY_FILE"

for ep_spec in "${ENDPOINTS[@]}"; do
    IFS='|' read -r METHOD URL NAME <<< "$ep_spec"
    FULL_URL="${BASE}${URL}"

    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  Endpoint: ${NAME} (${METHOD} ${URL})${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    for C in "${CONCURRENCIES[@]}"; do
        echo -ne "${YELLOW}  → c=${C} ... ${NC}"

        OUTFILE="${RESULTS_DIR}/${NAME}_c${C}.txt"

        $HEY -z "${DURATION}s" -c "$C" -m "$METHOD" "$FULL_URL" > "$OUTFILE" 2>&1

        # Parse hey output
        RPS=$(awk '/Requests\/sec:/{print $2}' "$OUTFILE")
        AVG=$(awk '/Average:/{print $2; exit}' "$OUTFILE")
        TOTAL=$(awk '/\[200\]/{print $2}' "$OUTFILE")
        ERRORS=$(awk '/^\[.*\] [0-9]+ responses/{s+=$2} END{print s+0}' "$OUTFILE")
        OK_COUNT=$(awk '/\[200\]/{gsub(/[^0-9]/,"",$2); print $2}' "$OUTFILE")

        # Parse latency distribution percentiles from hey output
        # hey outputs: 10% in X secs, 25% in X secs, 50% in X secs, 75% in X secs, 90% in X secs, 95% in X secs, 99% in X secs
        P50=$(awk '/50% in [0-9]/{print $3}' "$OUTFILE")
        P90=$(awk '/90% in [0-9]/{print $3}' "$OUTFILE")
        P99=$(awk '/99% in [0-9]/{print $3}' "$OUTFILE")

        # Convert seconds to ms
        AVG_MS=$(echo "${AVG:-0}" | awk '{printf "%.2f", $1*1000}')
        P50_MS=$(echo "${P50:-0}" | awk '{printf "%.2f", $1*1000}')
        P90_MS=$(echo "${P90:-0}" | awk '{printf "%.2f", $1*1000}')
        P99_MS=$(echo "${P99:-0}" | awk '{printf "%.2f", $1*1000}')

        # Count non-200 errors from status code distribution
        ERR_COUNT=$(awk '/Status code distribution/,/^$/{if(/\[/ && !/\[200\]/) {gsub(/[^0-9]/,"",$2); total+=$2}} END{print total+0}' "$OUTFILE")
        # Also count connection errors
        CONN_ERR=$(grep -c "^\s*\[" "$OUTFILE" 2>/dev/null || echo "0")

        printf "RPS:%-8s Avg:%-8s P50:%-8s P90:%-8s P99:%-8s\n" \
            "${RPS:-N/A}" "${AVG_MS}ms" "${P50_MS}ms" "${P90_MS}ms" "${P99_MS}ms"

        printf "%-12s %-6s %-10s %-10s %-10s %-10s %-10s %-8s %-8s\n" \
            "$NAME" "$C" "${RPS:-0}" "$AVG_MS" "$P50_MS" "$P90_MS" "$P99_MS" "${OK_COUNT:-0}" "$ERR_COUNT" >> "$SUMMARY_FILE"

        sleep 2
    done
    echo ""
done

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Benchmark Complete!${NC}"
echo -e "${GREEN}  Results: ${RESULTS_DIR}/${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Summary:"
cat "$SUMMARY_FILE"
