#!/usr/bin/env bash
set -euo pipefail
teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
teststudio_browser_base='mcr.microsoft.com/playwright:v1.64.1-noble@sha256:2f05f8a734509a7b3f26a912639488ee08f58a7b292d14d62271bcff9038768f'
teststudio_browser_arch=$(docker image inspect "$teststudio_browser_base" --format '{{.Architecture}}')
teststudio_browser_context=$(mktemp -d "${TMPDIR:-/tmp}/teststudio-browser-runtime.XXXXXX")
trap 'rm -rf "$teststudio_browser_context"' EXIT
mkdir -p "$teststudio_browser_context/runners/playwright"
for teststudio_browser_file in package.json package-lock.json run.mjs reporter.cjs suite-context.mjs discovery-reporter.cjs; do
  cp "runners/playwright/$teststudio_browser_file" "$teststudio_browser_context/runners/playwright/$teststudio_browser_file"
done
docker build --pull=false --build-arg "TARGETARCH=$teststudio_browser_arch" -f "$teststudio_root/runners/playwright/Dockerfile" -t teststudio/playwright-source-runtime:local "$teststudio_browser_context"
docker image inspect teststudio/playwright-source-runtime:local --format '{{.Id}}'
