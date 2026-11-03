#!/usr/bin/env bash
set -euo pipefail

teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
command -v docker >/dev/null
command -v go >/dev/null
command -v npm >/dev/null
command -v node >/dev/null

mkdir -p .data
teststudio_controller_pid=''
teststudio_run_root=$(mktemp -d "$teststudio_root/.data/selftest-verification.XXXXXX")
cleanup() {
  if [ -n "$teststudio_controller_pid" ]; then
    kill -TERM "$teststudio_controller_pid" 2>/dev/null || true
    wait "$teststudio_controller_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

teststudio_image=$(node scripts/build-selftest.mjs)
cp .data/selftest-build/controller "$teststudio_run_root/teststudio"

teststudio_api_port=$(node -e 'const s=require("node:net").createServer();s.listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})')
teststudio_artifact_port=$(node -e 'const s=require("node:net").createServer();s.listen(0,"127.0.0.1",()=>{console.log(s.address().port);s.close()})')
if [ "$teststudio_api_port" = "$teststudio_artifact_port" ]; then echo 'Selected ports collide. Run the script again.' >&2; exit 1; fi
teststudio_origin="http://127.0.0.1:$teststudio_api_port"
teststudio_artifact_origin="http://127.0.0.1:$teststudio_artifact_port"
export TESTSTUDIO_ORIGIN="$teststudio_origin"
export TESTSTUDIO_SELFTEST_IMAGE="$teststudio_image"
export TESTSTUDIO_SELFTEST_REVISION="$teststudio_image"
node -e 'require("node:fs").writeFileSync(process.argv[1],JSON.stringify({origin:process.env.TESTSTUDIO_ORIGIN,artifactOrigin:process.argv[2],image:process.env.TESTSTUDIO_SELFTEST_IMAGE},null,2))' "$teststudio_run_root/installation.json" "$teststudio_artifact_origin"
"$teststudio_run_root/teststudio" --local --addr "127.0.0.1:$teststudio_api_port" --public-url "$teststudio_origin" \
  --artifact-addr "127.0.0.1:$teststudio_artifact_port" --artifact-origin "$teststudio_artifact_origin" \
  --database "$teststudio_run_root/teststudio.db" --artifact-directory "$teststudio_run_root/artifacts" \
  --playwright-images "$teststudio_image" >"$teststudio_run_root/server.log" 2>&1 &
teststudio_controller_pid=$!
node -e 'const origin=process.env.TESTSTUDIO_ORIGIN;let ready=false;for(let n=0;n<100;n++){try{ready=(await fetch(origin+"/healthz")).ok}catch{}if(ready)break;await new Promise(r=>setTimeout(r,100))}if(!ready)throw new Error("The isolated self-test controller did not start")'
node scripts/selftest.mjs "$@"
echo "Retained TestStudio verification installation: $teststudio_run_root"
