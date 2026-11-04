ARG BUILDPLATFORM
FROM --platform=${BUILDPLATFORM} golang:1.26.8-alpine3.23@sha256:33ce311e5eecedee48ec1b84419c1306e9fbd71009f0d5c3f2a6904b579c1ecc AS files
RUN mkdir -p /data /tmp && chmod 700 /data && chmod 1777 /tmp
FROM scratch
ARG TARGETARCH
ARG SOURCE_KEY
LABEL org.opencontainers.image.title="TestStudio" org.opencontainers.image.source="https://github.com/wtiger001/teststudio"
LABEL teststudio.package="service/v1" teststudio.architecture="${TARGETARCH}" teststudio.source-key="${SOURCE_KEY}"
COPY --from=files /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=files --chown=65532:65532 /data /data
COPY --from=files --chown=65532:65532 /tmp /tmp
COPY teststudio teststudio-health /app/
COPY ui /app/ui
COPY trace-viewer /app/trace-viewer
ENV HOME=/tmp TESTSTUDIO_DATABASE=/data/teststudio.db
USER 65532:65532
WORKDIR /app
EXPOSE 8040 8041
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/teststudio-health"]
ENTRYPOINT ["/app/teststudio", "--addr=0.0.0.0:8040", "--artifact-addr=0.0.0.0:8041", "--database-env=TESTSTUDIO_DATABASE", "--artifact-directory=/data/artifacts", "--ui=/app/ui", "--trace-viewer-directory=/app/trace-viewer"]
