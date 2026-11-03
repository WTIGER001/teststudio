#!/usr/bin/env bash
set -euo pipefail

teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
mkdir -p .data
export TESTSTUDIO_NOTIFICATION_VERIFY_ROOT=$(mktemp -d "$teststudio_root/.data/notification-verification.XXXXXX")
go build -o "$TESTSTUDIO_NOTIFICATION_VERIFY_ROOT/teststudio" ./cmd/teststudio
node scripts/verify-notifications.mjs
