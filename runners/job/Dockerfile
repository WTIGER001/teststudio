ARG RUNTIME_IMAGE
FROM ${RUNTIME_IMAGE}
ARG RUNTIME_DIGEST
ARG SOURCE_KEY
LABEL teststudio.discovery-supervisor="worker-exchange/v1" teststudio.job-supervisor="worker-exchange/v1" teststudio.job-runtime-base="${RUNTIME_DIGEST}" teststudio.job-source-key="${SOURCE_KEY}"
LABEL teststudio.task-storage="task-volumes/v1"
LABEL teststudio.job-gateway="tls-execution/v1"
LABEL teststudio.sandbox-supervisor="azure-sandbox-command/v1"
USER 0:0
RUN mkdir -p /tmp /output && chown 1000:1000 /tmp /output && chmod 1777 /tmp && chmod 0700 /output
VOLUME ["/tmp", "/output"]
COPY --chown=0:0 --chmod=0555 teststudio-job /runner/teststudio-job
USER 1000:1000
