#!/usr/bin/env bash
set -euo pipefail

teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
mkdir -p .data
teststudio_run_root=$(mktemp -d "$teststudio_root/.data/access-verification.XXXXXX")
go build -o "$teststudio_run_root/teststudio" ./cmd/teststudio
node scripts/verify-access.mjs "$teststudio_run_root"
