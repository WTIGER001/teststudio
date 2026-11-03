#!/usr/bin/env bash
set -euo pipefail

teststudio_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$teststudio_root"
mkdir -p .data/discovery-build
teststudio_verification=$(mktemp -d "$teststudio_root/.data/discovery-verification.XXXXXX")
export TESTSTUDIO_DISCOVERY_INSTALLATION=$(node -e 'process.stdout.write(require("node:crypto").randomUUID())')
teststudio_child_pid=''
cleanup() {
  local teststudio_status=$?
  if [ -n "$teststudio_child_pid" ]; then kill -TERM "$teststudio_child_pid" 2>/dev/null || true; wait "$teststudio_child_pid" 2>/dev/null || true; fi
  if [ -x "$teststudio_verification/teststudio-publish" ]; then
    for teststudio_state in "$teststudio_verification"/state/*.json; do
      [ -f "$teststudio_state" ] || continue
      "$teststudio_verification/teststudio-publish" cleanup --state-file "$teststudio_state" || teststudio_status=1
    done
  fi
  for teststudio_namespace in "preparation-$TESTSTUDIO_DISCOVERY_INSTALLATION" "runtime-check-$TESTSTUDIO_DISCOVERY_INSTALLATION"; do
    teststudio_containers=$(docker ps -aq --filter "label=teststudio.installation=$teststudio_namespace" --filter 'label=teststudio.project=discovery-tests') || teststudio_status=1
    while IFS= read -r teststudio_container; do [ -z "$teststudio_container" ] || docker rm -f "$teststudio_container" >/dev/null || teststudio_status=1; done <<<"$teststudio_containers"
    teststudio_volumes=$(docker volume ls -q --filter "label=teststudio.installation=$teststudio_namespace" --filter 'label=teststudio.project=discovery-tests') || teststudio_status=1
    while IFS= read -r teststudio_volume; do [ -z "$teststudio_volume" ] || docker volume rm "$teststudio_volume" >/dev/null || teststudio_status=1; done <<<"$teststudio_volumes"
  done
  exit "$teststudio_status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
run_child() { "$@" & teststudio_child_pid=$!; wait "$teststudio_child_pid"; teststudio_child_pid=''; }

teststudio_go_base='golang:1.26.8-alpine3.23@sha256:33ce311e5eecedee48ec1b84419c1306e9fbd71009f0d5c3f2a6904b579c1ecc'
teststudio_playwright_base='mcr.microsoft.com/playwright:v1.64.1-noble@sha256:2f05f8a734509a7b3f26a912639488ee08f58a7b292d14d62271bcff9038768f'
docker image inspect "$teststudio_go_base" >/dev/null 2>&1 || docker pull "$teststudio_go_base" >/dev/null
docker image inspect "$teststudio_playwright_base" >/dev/null 2>&1 || docker pull "$teststudio_playwright_base" >/dev/null
teststudio_go_arch=$(docker image inspect "$teststudio_go_base" --format '{{.Architecture}}')
teststudio_playwright_arch=$(docker image inspect "$teststudio_playwright_base" --format '{{.Architecture}}')
CGO_ENABLED=0 GOOS=linux GOARCH="$teststudio_go_arch" go build -o .data/discovery-build/teststudio-go ./runners/go/cmd/teststudio-go
node --input-type=module - <<'JS'
import fs from 'node:fs';import crypto from 'node:crypto';import path from 'node:path';
for(const kind of ['go','playwright']){
 const root=`tests/discovery/${kind}`,hash=crypto.createHash('sha256');
 const visit=relative=>{for(const entry of fs.readdirSync(path.join(root,relative),{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name))){const name=path.posix.join(relative,entry.name);if(entry.isDirectory())visit(name);else{hash.update(name+'\0');hash.update(fs.readFileSync(path.join(root,name)));hash.update('\0')}}};
 visit('');fs.writeFileSync(`.data/discovery-build/${kind}-origin.json`,JSON.stringify({snapshotDigest:`sha256:${hash.digest('hex')}`}),{mode:0o600});
}
JS
docker build -q --build-arg "TARGETARCH=$teststudio_go_arch" -f tests/discovery/go/Dockerfile -t teststudio/discovery-go:local . >/dev/null
docker build -q --build-arg "TARGETARCH=$teststudio_playwright_arch" -f tests/discovery/playwright/Dockerfile -t teststudio/discovery-playwright:local . >/dev/null
export TESTSTUDIO_DISCOVERY_GO_IMAGE=$(docker image inspect teststudio/discovery-go:local --format '{{.Id}}')
export TESTSTUDIO_DISCOVERY_PLAYWRIGHT_IMAGE=$(docker image inspect teststudio/discovery-playwright:local --format '{{.Id}}')
go test -race -c -o "$teststudio_verification/inspection.test" ./adapters/execution/localcontainer
run_child "$teststudio_verification/inspection.test" -test.run 'Test(IsolatedArtifact|DiscoveryDestination|InspectionRefuses|DiscoveryRespects|InspectedArtifacts)' -test.count=1
go build -o "$teststudio_verification/teststudio-publish" ./cmd/teststudio-publish
run_child "$teststudio_verification/teststudio-publish" inspect --runner go --image "$TESTSTUDIO_DISCOVERY_GO_IMAGE" --project cli-discovery-checks --suite discovery-go --output "$teststudio_verification/go.json" --state-directory "$teststudio_verification/state"
run_child "$teststudio_verification/teststudio-publish" inspect --runner playwright --image "$TESTSTUDIO_DISCOVERY_PLAYWRIGHT_IMAGE" --project cli-discovery-checks --suite discovery-browser --output "$teststudio_verification/playwright.json" --state-directory "$teststudio_verification/state"
node --input-type=module - "$teststudio_verification" <<'JS'
import fs from 'node:fs';import {spawnSync} from 'node:child_process';
const root=process.argv[2];
for(const runner of ['go','playwright']){
 const filename=`${root}/${runner}.json`,report=JSON.parse(fs.readFileSync(filename));
 if(report.status!=='inspected'||report.inspection.discovery.tests.length!==2||(fs.statSync(filename).mode&0o777)!==0o600)throw new Error('CLI inspection contract or privacy failed');
 for(const kind of ['container','volume']){const result=spawnSync('docker',[kind,'inspect',report.reference[kind]],{encoding:'utf8'});if(result.status===0||!/no such/i.test(result.stderr))throw new Error('Owned CLI preparation cleanup is unverified')}
}
if(fs.readdirSync(`${root}/state`).length)throw new Error('Successful CLI inspection retained cleanup references');
console.log(`Retained private artifact inspections: ${root}`);
JS
