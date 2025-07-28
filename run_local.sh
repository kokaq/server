#!/bin/bash

echo "Starting control plane..."
go run ./cmd/controlplane/main.go -control=9001 -shard=9000&
CONTROL_PID=$!

echo "Starting data plane..."
go run ./cmd/dataplane/main.go -port=9002 -shard=9000&
DATA_PID=$!

# Wait for all processes (optional: so the script doesn't exit immediately)
wait $CONTROL_PID $DATA_PID