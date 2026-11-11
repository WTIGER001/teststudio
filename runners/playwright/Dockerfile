FROM mcr.microsoft.com/playwright:v1.64.1-noble@sha256:2f05f8a734509a7b3f26a912639488ee08f58a7b292d14d62271bcff9038768f
WORKDIR /runner
COPY runners/playwright/package.json runners/playwright/package-lock.json ./
RUN npm ci --ignore-scripts --omit=dev --no-audit --no-fund
ARG TARGETARCH
LABEL teststudio.runtime="playwright1.64.1-noble-linux-${TARGETARCH}-source-v2"
LABEL teststudio.runtime-kind="source-base/v1"
LABEL teststudio.playwright-version="1.64.1"
LABEL teststudio.api-policy="request-policy/v1"
LABEL teststudio.work-item-bytes="4194304"
COPY runners/playwright/run.mjs runners/playwright/reporter.cjs runners/playwright/suite-context.mjs runners/playwright/discovery-reporter.cjs runners/playwright/egress.mjs runners/playwright/selection.mjs runners/playwright/execution-config.mjs ./
RUN mkdir -p /suite /output && ln -s /runner/node_modules /suite/node_modules && chmod 1777 /output
ENV HOME=/tmp
WORKDIR /suite
ENTRYPOINT ["node", "/runner/run.mjs"]
