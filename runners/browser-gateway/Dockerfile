FROM scratch
COPY teststudio-browser-gateway /gateway
COPY --chown=1000:1000 output /output
USER 1000:1000
HEALTHCHECK --interval=1s --timeout=1s --retries=1 CMD ["/gateway", "--ready"]
ENTRYPOINT ["/gateway"]
