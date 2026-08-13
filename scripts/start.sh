#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
PORT="${PORT:-9876}"

echo "Arrêt des processus sur le port $PORT…"
PIDS=$(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null || true)
if [ -n "${PIDS:-}" ]; then
  # shellcheck disable=SC2086
  kill -9 $PIDS 2>/dev/null || true
  sleep 0.5
fi

echo "Démarrage → http://127.0.0.1:$PORT/"
exec go run ./cmd/server
