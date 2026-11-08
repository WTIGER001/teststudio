# Independent Angular status consumer

This application imports only the built @wtiger001/teststudio-angular package.
It has its own dependencies, lockfile, Angular build configuration, and styles.
It imports no TestStudio application internals and starts no tests on page load.

Run node scripts/build-status-library.mjs --consumer from the repository root.
This installs the built library as a package copy, compiles this application, and checks server rendering.
The output is web/dist/status-consumer/browser.

Host data demonstrates a complete page and compact bar. Appearance changes show light, dark, and host brand styles.
The local self-test host supplies a published page through the page query parameter and serves this build from a separate origin.
Remove components exercises subscription cleanup. Show components loads fresh status.
The example uses generic illustrative browser-test data for host-data mode. Public mode uses actual reports from the isolated local verification installation.

The example's loopback API URL belongs to the local test fixture. Configure the public source URL for your own host.
Package creation and example compilation do not publish to a registry or deploy an application.
