FROM scratch
ARG TARGETARCH
ARG SOURCE_KEY
LABEL org.opencontainers.image.title="TestStudio worker relay" org.opencontainers.image.source="https://github.com/wtiger001/teststudio"
LABEL teststudio.package="worker-relay/v1" teststudio.architecture="${TARGETARCH}" teststudio.source-key="${SOURCE_KEY}"
COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY teststudio-exchange teststudio-health /app/
USER 65532:65532
WORKDIR /app
EXPOSE 8043
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/app/teststudio-health", "--url=http://127.0.0.1:8043/healthz"]
ENTRYPOINT ["/app/teststudio-exchange", "--addr=0.0.0.0:8043"]
