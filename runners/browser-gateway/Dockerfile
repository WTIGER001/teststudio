FROM golang:1.26.8-alpine3.23@sha256:33ce311e5eecedee48ec1b84419c1306e9fbd71009f0d5c3f2a6904b579c1ecc AS certificates
FROM scratch
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
LABEL teststudio.api-policy="request-policy/v1"
LABEL teststudio.api-credentials="gateway-injection/v1"
LABEL teststudio.gateway-config="private-config/v1"
LABEL teststudio.gateway-remote="tls-execution/v1"
LABEL teststudio.gateway-certificate="secret-env/v1"
LABEL teststudio.gateway-instance="ecs-task/v1"
COPY teststudio-browser-gateway /gateway
COPY --chown=1000:1000 output /output
USER 1000:1000
HEALTHCHECK --interval=1s --timeout=1s --start-period=10s --retries=1 CMD ["/gateway", "--ready"]
ENTRYPOINT ["/gateway"]
