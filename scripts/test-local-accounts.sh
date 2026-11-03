#!/usr/bin/env bash
set -euo pipefail

teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
mkdir -p .data
teststudio_run_root=$(mktemp -d "$teststudio_root/.data/local-accounts-verification.XXXXXX")
go build -o "$teststudio_run_root/teststudio" ./cmd/teststudio
node scripts/verify-local-accounts.mjs "$teststudio_run_root"
