FROM mcr.microsoft.com/playwright:v1.64.1-noble@sha256:2f05f8a734509a7b3f26a912639488ee08f58a7b292d14d62271bcff9038768f
ARG TARGETARCH
LABEL teststudio.runtime="playwright1.64.1-node26.13.0-noble-linux-${TARGETARCH}-source-v2"
WORKDIR /runner
COPY runners/playwright/package.json runners/playwright/package-lock.json ./
RUN npm ci --ignore-scripts --omit=dev --no-audit --no-fund
COPY runners/playwright/run.mjs runners/playwright/reporter.cjs runners/playwright/suite-context.mjs runners/playwright/discovery-reporter.cjs runners/playwright/egress.mjs runners/playwright/execution-config.mjs runners/playwright/selection.mjs ./
COPY tests/discovery/playwright /suite
COPY --chown=1000:1000 .data/discovery-build/playwright-origin.json /suite/teststudio-origin.json
RUN ln -s /runner/node_modules /suite/node_modules && mkdir -p /output && chmod 1777 /output
WORKDIR /suite
ENV HOME=/tmp
ENTRYPOINT ["node", "/runner/run.mjs"]
