#!/bin/bash
set -e

echo "[entrypoint] PORT1=$PORT1, PORT2=$PORT2"

# Pass envs to your Go app as args or keep them as env vars
exec go run ./cmd/dataplane --port1="$PORT1" --port2="$PORT2"
