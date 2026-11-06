FROM scratch
ARG TARGETARCH
ARG SOURCE_KEY
LABEL org.opencontainers.image.title="TestStudio" org.opencontainers.image.source="https://github.com/wtiger001/teststudio"
LABEL teststudio.package="service/v1" teststudio.architecture="${TARGETARCH}" teststudio.source-key="${SOURCE_KEY}"
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chown=65532:65532 data /data
COPY --chown=65532:65532 temporary /tmp
COPY --chown=65532:65532 journals /private/source-journals
COPY teststudio teststudio-health teststudio-prepare rds-ca-bundle.pem /app/
COPY ui /app/ui
COPY trace-viewer /app/trace-viewer
ENV HOME=/tmp TESTSTUDIO_DATABASE=/data/teststudio.db
USER 65532:65532
VOLUME ["/tmp"]
WORKDIR /app
EXPOSE 8040 8041
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/teststudio-health"]
ENTRYPOINT ["/app/teststudio", "--addr=0.0.0.0:8040", "--artifact-addr=0.0.0.0:8041", "--database-env=TESTSTUDIO_DATABASE", "--artifact-directory=/data/artifacts", "--ui=/app/ui", "--trace-viewer-directory=/app/trace-viewer"]
