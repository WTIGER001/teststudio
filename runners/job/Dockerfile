ARG RUNTIME_IMAGE
FROM ${RUNTIME_IMAGE}
ARG RUNTIME_DIGEST
ARG SOURCE_KEY
LABEL teststudio.discovery-supervisor="worker-exchange/v1" teststudio.job-supervisor="worker-exchange/v1" teststudio.job-runtime-base="${RUNTIME_DIGEST}" teststudio.job-source-key="${SOURCE_KEY}"
COPY --chown=0:0 --chmod=0555 teststudio-job /runner/teststudio-job
USER 1000:1000
