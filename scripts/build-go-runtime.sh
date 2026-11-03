#!/usr/bin/env bash
set -euo pipefail
teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
teststudio_go_base='golang:1.26.8-alpine3.23@sha256:33ce311e5eecedee48ec1b84419c1306e9fbd71009f0d5c3f2a6904b579c1ecc'
teststudio_go_arch=$(docker image inspect "$teststudio_go_base" --format '{{.Architecture}}')
mkdir -p .data/runtime-build
CGO_ENABLED=0 GOOS=linux GOARCH="$teststudio_go_arch" go build -o .data/runtime-build/teststudio-go ./runners/go/cmd/teststudio-go
docker build --pull=false --network=none --build-arg "TARGETARCH=$teststudio_go_arch" -f "$teststudio_root/runners/go/Dockerfile" -t teststudio/go-source-runtime:local .data/runtime-build
docker image inspect teststudio/go-source-runtime:local --format '{{.Id}}'
