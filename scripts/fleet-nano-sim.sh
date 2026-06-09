#!/usr/bin/env bash
# Run N real meshyants worker processes (Nano-style: native executor) plus loadsim as oracle publisher.
#
# Usage:
#   NATS_URL=nats://127.0.0.1:4222 ./scripts/fleet-nano-sim.sh [devices] [tasks]
#
# Optional: KEYFILE=./oracle.key to reuse a key (otherwise a temp key is created).
#
# Requires: NATS with JetStream (e.g. docker run --rm -p 4222:4222 nats:2.10-alpine -js)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

N="${1:-10}"
TASKS="${2:-100}"
NATS_URL="${NATS_URL:-nats://127.0.0.1:4222}"
TRUST="${TRUST:-load}"

TMPKEY=0
KEYFILE="${KEYFILE:-}"
if [[ -z "$KEYFILE" ]]; then
  KEYFILE="$(mktemp -t meshyants-oracle-XXXXXX.key)"
  openssl rand -out "$KEYFILE" 32
  TMPKEY=1
fi

BIN="$(mktemp -t meshyants-bin-XXXXXX)"
go build -o "$BIN" ./cmd/meshyants

PIDS=()
cleanup() {
  for pid in "${PIDS[@]:-}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
  rm -f "$BIN"
  if [[ "$TMPKEY" == 1 ]]; then
    rm -f "$KEYFILE"
  fi
}
trap cleanup EXIT

for ((i = 0; i < N; i++)); do
  NATS_URL="$NATS_URL" "$BIN" worker \
    --trust-domain="$TRUST" \
    --subject="mesh.task.$TRUST.dev$i" \
    --issuer=oracle \
    --key-file="$KEYFILE" \
    --executor=native \
    --effect-db="/tmp/meshyants-fleet-$TRUST-$i.db" &
  PIDS+=($!)
done

sleep 1
go run ./cmd/loadsim -nats="$NATS_URL" -publish-only -key-file="$KEYFILE" -devices="$N" -tasks="$TASKS" -trust="$TRUST" -quiet
