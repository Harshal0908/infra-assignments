#!/bin/sh

set -eu

NAMESPACE="config-service"
LOCAL_PORT="18080"
BASE_URL="http://localhost:${LOCAL_PORT}"

cleanup() {
  if [ -n "${PORT_FORWARD_PID:-}" ]; then
    kill "${PORT_FORWARD_PID}" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

echo "Starting temporary port-forward..."

kubectl -n "${NAMESPACE}" port-forward \
  service/config-service \
  "${LOCAL_PORT}:8080" \
  >/tmp/config-service-port-forward.log 2>&1 &

PORT_FORWARD_PID=$!

echo "Waiting for application..."

attempt=0
until curl -fsS "${BASE_URL}/ping" >/dev/null 2>&1; do
  attempt=$((attempt + 1))

  if [ "${attempt}" -ge 30 ]; then
    echo "Application did not become available."
    cat /tmp/config-service-port-forward.log
    exit 1
  fi

  sleep 1
done

echo "Checking liveness..."
PING_RESPONSE=$(curl -fsS "${BASE_URL}/ping")

if [ "${PING_RESPONSE}" != "pong" ]; then
  echo "Unexpected /ping response: ${PING_RESPONSE}"
  exit 1
fi

echo "Checking readiness..."
READY_RESPONSE=$(curl -fsS "${BASE_URL}/readyz")

echo "${READY_RESPONSE}" | grep -q '"status":"ready"'

echo "Creating configuration..."
CREATE_RESPONSE=$(curl -fsS \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "id": "smoke_test",
    "host": "localhost",
    "port": 8080,
    "app_name": "config-service",
    "log_level": "INFO"
  }' \
  "${BASE_URL}/configs")

echo "${CREATE_RESPONSE}" | grep -q '"id":"smoke_test"'

echo "Retrieving configuration..."
GET_RESPONSE=$(curl -fsS "${BASE_URL}/configs/smoke_test")

echo "${GET_RESPONSE}" | grep -q '"host":"localhost"'

echo "Updating configuration..."
UPDATE_RESPONSE=$(curl -fsS \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "id": "smoke_test",
    "host": "updated.internal",
    "port": 9090,
    "app_name": "updated-service",
    "log_level": "DEBUG"
  }' \
  "${BASE_URL}/configs")

echo "${UPDATE_RESPONSE}" | grep -q '"host":"updated.internal"'

echo "Checking missing configuration..."
NOT_FOUND_STATUS=$(curl -s \
  -o /dev/null \
  -w "%{http_code}" \
  "${BASE_URL}/configs/does-not-exist")

if [ "${NOT_FOUND_STATUS}" != "404" ]; then
  echo "Expected 404, received ${NOT_FOUND_STATUS}"
  exit 1
fi

echo "All smoke tests passed."
