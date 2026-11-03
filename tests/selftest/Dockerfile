FROM mcr.microsoft.com/playwright:v1.64.1-noble@sha256:2f05f8a734509a7b3f26a912639488ee08f58a7b292d14d62271bcff9038768f
WORKDIR /runner
COPY runners/playwright/package.json runners/playwright/package-lock.json ./
RUN npm ci --ignore-scripts --omit=dev --no-audit --no-fund
COPY runners/playwright/run.mjs runners/playwright/reporter.cjs ./
COPY tests/selftest/playwright /suite
RUN ln -s /runner/node_modules /suite/node_modules && mkdir -p /output && chmod 1777 /output
COPY .data/selftest-build/teststudio /app/teststudio
COPY .data/selftest-build/access-fixture /app/access-fixture
COPY web/dist/web/browser /app/ui
WORKDIR /suite
ENV HOME=/tmp
ENTRYPOINT ["node", "/runner/run.mjs"]
