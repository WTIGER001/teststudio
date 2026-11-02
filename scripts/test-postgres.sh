#!/usr/bin/env bash
set -euo pipefail

teststudio_container="teststudio-contract-db-$$"
teststudio_image="${TESTSTUDIO_POSTGRES_IMAGE:-postgres@sha256:16bc17c64a573ef34162af9298258d1aec548232985b33ed7b1eac33ba35c229}"
teststudio_password="$(openssl rand -hex 24)"

cleanup() {
  docker rm -f "$teststudio_container" >/dev/null 2>&1 || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

docker run --detach --rm --name "$teststudio_container" \
  --publish 127.0.0.1::5432 \
  --env POSTGRES_USER=teststudio --env POSTGRES_PASSWORD="$teststudio_password" \
  --tmpfs /var/lib/postgresql/data:rw \
  "$teststudio_image" >/dev/null

teststudio_ready=false
for ((teststudio_attempt=0; teststudio_attempt<60; teststudio_attempt++)); do
  if docker exec "$teststudio_container" pg_isready -U teststudio >/dev/null 2>&1; then
    teststudio_ready=true
    break
  fi
  sleep 0.5
done
if [[ "$teststudio_ready" != true ]]; then
  docker logs "$teststudio_container"
  exit 1
fi

teststudio_port="$(docker port "$teststudio_container" 5432/tcp)"
teststudio_port="${teststudio_port##*:}"
export TESTSTUDIO_TEST_DATABASE="postgres://teststudio:$teststudio_password@127.0.0.1:$teststudio_port/postgres?sslmode=disable"

if [[ "$#" -eq 0 ]]; then
  set -- ./internal/storage ./internal/authorization ./internal/catalog ./internal/runs ./internal/evidence ./internal/httpapi ./internal/sessions ./adapters/auth/oidc
fi
go test -race "$@" -count=1
