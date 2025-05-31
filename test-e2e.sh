#!/usr/bin/env bash
set -euo pipefail

# Configuration
: "${TEST_USER:?}"
: "${TEST_HOST:?}"
: "${TEST_SSH_KEY:?}"
: "${TARGET_FQDN:?}"
TARGET_USER="${TARGET_USER:-$(id -un)}"
NATTS_PORT="${NATTS_PORT:-30000}"

echo "Building project..."
nix build

echo "Copying nattc to testing host..."
ssh -i "$TEST_SSH_KEY" "$TEST_USER@$TEST_HOST" rm /tmp/nattc
scp -i "$TEST_SSH_KEY" ./result/bin/nattc "$TEST_USER@$TEST_HOST:/tmp/nattc"

# Cleanup function
cleanup() {
  echo "Cleaning up..."
  if [ -n "${NATTS_PID:-}" ]; then
    kill $NATTS_PID 2>/dev/null || true
    wait $NATTS_PID 2>/dev/null || true
  fi
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

echo "Starting natts server locally..."
./result/bin/natts -listen ":$NATTS_PORT" &
NATTS_PID=$!

# Wait for natts to start
sleep 2

echo "Testing SSH connection through NAT traversal..."
RESULT=$(ssh -i "$TEST_SSH_KEY" "$TEST_USER@$TEST_HOST" \
  ssh -o ProxyCommand="'/tmp/nattc -proxy -target $TARGET_FQDN'" \
  "$TARGET_USER@$TARGET_FQDN" \
  echo OK)

# Give a moment for connections to close gracefully
sleep 1

echo "SSH command result: $RESULT"
if [ "$RESULT" = "OK" ]; then
  echo "✅ Test completed successfully!"
else
  echo "❌ Test failed - expected 'OK', got: '$RESULT'"
  exit 1
fi
