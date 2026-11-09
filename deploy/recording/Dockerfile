ARG RUNTIME_IMAGE
FROM ${RUNTIME_IMAGE}
USER root
RUN mkdir -p /recording /output && chown 1000:1000 /output && chmod 0700 /output
COPY record-browser-embedded.mjs recording-proxy.mjs recording-target.mjs /recording/
LABEL teststudio.recording-runtime="isolated-browser/v1" teststudio.playwright-version="1.64.1"
ENV HOME=/tmp
USER 1000:1000
WORKDIR /recording
ENTRYPOINT ["node", "/recording/record-browser-embedded.mjs", "/runner/node_modules/playwright/cli.js"]
