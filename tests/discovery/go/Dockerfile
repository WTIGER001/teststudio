FROM golang:1.26.8-alpine3.23@sha256:33ce311e5eecedee48ec1b84419c1306e9fbd71009f0d5c3f2a6904b579c1ecc
ARG TARGETARCH
LABEL teststudio.runtime="go1.26.8-linux-${TARGETARCH}-cgo0-source-v1"
COPY --chown=1000:1000 .data/discovery-build/teststudio-go /runner/teststudio-go
COPY tests/discovery/go /suite
COPY --chown=1000:1000 .data/discovery-build/go-origin.json /suite/teststudio-origin.json
RUN mkdir -p /output && chmod 1777 /output
ENV HOME=/tmp GOCACHE=/tmp/go-cache GOPATH=/tmp/go-path
WORKDIR /suite
ENTRYPOINT ["/runner/teststudio-go"]
