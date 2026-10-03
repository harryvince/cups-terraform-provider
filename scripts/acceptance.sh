#!/usr/bin/env bash
set -euo pipefail
cleanup() {
  status=$?
  if (( status != 0 )); then
    docker-compose logs --no-color || true
  fi
  docker-compose down
}
trap cleanup EXIT

docker-compose up --build --wait --wait-timeout 120
docker-compose exec -T cups python3 /opt/testenv/smoke.py
CUPS_ACC=1 \
  CUPS_ACC_ENDPOINT="http://127.0.0.1:${CUPS_TEST_PORT:-8631}" \
  CUPS_ACC_USERNAME=cups-admin \
  CUPS_ACC_PASSWORD="${CUPS_TEST_ADMIN_PASSWORD:-cups-test-password}" \
  go test -v -count=1 -timeout=10m ./internal/provider -run TestAcceptance
