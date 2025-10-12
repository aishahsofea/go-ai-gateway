#!/bin/bash

echo "🚀 Full Gateway Test Suite"
echo "--------------------------------"

chmod +x scripts/test-gateway.sh

echo "Starting main gateway..."
docker compose up -d gateway

sleep 5

go run cmd/test-runner/main.go