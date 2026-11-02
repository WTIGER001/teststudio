#!/usr/bin/env bash
set -euo pipefail

teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
mkdir -p .data/go-runner-build
teststudio_run_root=$(mktemp -d "$teststudio_root/.data/go-verification.XXXXXX")
teststudio_controller_pid=''
cleanup() {
  if [ -n "$teststudio_controller_pid" ]; then
    kill -TERM "$teststudio_controller_pid" 2>/dev/null || true
    wait "$teststudio_controller_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

teststudio_base='golang:1.26.8-alpine3.23@sha256:33ce311e5eecedee48ec1b84419c1306e9fbd71009f0d5c3f2a6904b579c1ecc'
docker image inspect "$teststudio_base" >/dev/null 2>&1 || docker pull "$teststudio_base" >/dev/null
teststudio_arch=$(docker image inspect "$teststudio_base" --format '{{.Architecture}}')
CGO_ENABLED=0 GOOS=linux GOARCH="$teststudio_arch" go build -o .data/go-runner-build/teststudio-go ./runners/go/cmd/teststudio-go
docker build -q --build-arg "TARGETARCH=$teststudio_arch" -f tests/selftest/go/Dockerfile -t teststudio/go-selftests:local . >/dev/null
export TESTSTUDIO_GO_IMAGE=$(docker image inspect teststudio/go-selftests:local --format '{{.Id}}')
go build -o "$teststudio_run_root/teststudio" ./cmd/teststudio
teststudio_api_port=$(node -e 'const s=require("node:net").createServer();s.listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})')
teststudio_artifact_port=$(node -e 'const s=require("node:net").createServer();s.listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})')
if [ "$teststudio_api_port" = "$teststudio_artifact_port" ]; then echo 'Selected ports collide. Run the script again.' >&2; exit 1; fi
export TESTSTUDIO_ORIGIN="http://127.0.0.1:$teststudio_api_port"
teststudio_artifact_origin="http://127.0.0.1:$teststudio_artifact_port"
node -e 'require("node:fs").writeFileSync(process.argv[1],JSON.stringify({origin:process.env.TESTSTUDIO_ORIGIN,artifactOrigin:process.argv[2],image:process.env.TESTSTUDIO_GO_IMAGE},null,2))' "$teststudio_run_root/installation.json" "$teststudio_artifact_origin"
"$teststudio_run_root/teststudio" --local --addr "127.0.0.1:$teststudio_api_port" --public-url "$TESTSTUDIO_ORIGIN" \
  --artifact-addr "127.0.0.1:$teststudio_artifact_port" --artifact-origin "$teststudio_artifact_origin" \
  --database "$teststudio_run_root/teststudio.db" --artifact-directory "$teststudio_run_root/artifacts" \
  --go-images "$TESTSTUDIO_GO_IMAGE" >"$teststudio_run_root/server.log" 2>&1 &
teststudio_controller_pid=$!
node -e 'let ready=false;for(let n=0;n<100;n++){try{ready=(await fetch(process.env.TESTSTUDIO_ORIGIN+"/healthz")).ok}catch{}if(ready)break;await new Promise(r=>setTimeout(r,100))}if(!ready)throw new Error("The isolated Go controller did not start")'
node scripts/go-selftest.mjs
echo "Retained Go verification installation: $teststudio_run_root"
