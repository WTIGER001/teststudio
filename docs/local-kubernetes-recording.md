# Local Kubernetes desktop recording

## Introduction

The dedicated local Kubernetes installation now accepts a real desktop recording through its normal HTTPS ingress.
The native companion records an independent local target, stops on the owner's request, and returns source for a durable draft.
This extends the retained authoring UI and desktop package evidence. It does not establish AWS or Azure acceptance.
The ledger remains 50 verified of 57. Acceptance 57 advances and stays open across the required deployment profiles.

## Concepts

### Installed profile

Helm revision 15 runs chart 0.1.1 with recordingCompanion enabled on the API.
The updated immutable ARM64 service image also supplies both execution controllers and preparation.
Their commands do not enable recording. All three deployments report their expected Ready replicas.
The gateway, worker runtime images, two identity providers, PostgreSQL, and shared evidence configuration remain selected from the previous profile.
No coded suite executes for this recorder check.

The source image is sha256:0e6a770abf0da9e7323a5608275de24ce5329e2cca84de46e7a10f064160d470.
The current private installation values remain .data/local-k3s/controller-values.json.
The previous values remain .data/local-k3s/before-recording-values.json.
The user demo on port 8040 remains running. No host DNS or system certificate trust changes occur.

### Actual handoff

Normal OIDC sign-in creates the author's managed session. The verifier uses that session to create a project, suite, and owner-private recording.
The companion uses a separate random device credential. It receives no browser cookie or application session.
Native requests pass through the installed API's normal pairing, control, ordered snapshot, and current-authority checks.
The test transport maps only studio.teststudio.test:443 to the owned loopback ingress.
It trusts the existing development certificate and retains certificate and hostname verification. It does not disable TLS checks.
This private routing adaptation belongs only to the local test. Ordinary desktop packages use the server's normal DNS and operating-system trust.

The recorder opens dedicated native Chromium and Inspector windows and captures the target's initial navigation.
The owner requests Stop and keep recording after the generated source reaches the installed service.
The companion closes its browser and acknowledges final source. The server records ready.
Source conversion adds the permanent tag, visible environment binding, and needs-review safety label.
Saving the returned document creates one PostgreSQL-backed source draft at revision 1.
No preparation, publication, or target test execution occurs.

The retained recording is desktop-8bdd35ed-8b4b-410c-a7c7-dd2c4365924e.
The saved draft is 20261105T220101.897259977-94680fe9e468d63c7a720d59 in project k3s-recording, suite desktop.
The native check passes in 6.46 seconds. It captures navigation, not an additional click or assertion sequence.
The separate earlier authoring UI check already verifies the wizard, help, focus, review, saving, themes, and narrow layout.

### Cleanup and retained history

All 23 owned verification processes exit. The local HTTP target closes and the OIDC session signs out.
Temporary pairing input, device credentials, journal, and desktop files are removed.
The installed recording and saved source remain private under the normal owner and suite permissions.
All twelve k3s-coded runs and seven k3s-network runs retain their original identifiers. No additional run appears in the new recording project.
No worker workload remains in the dedicated worker namespace.

## Tutorials

### Inspect retained evidence

Evidence is output/local-k3s/recording.
installation.json records image identity, Helm revision, Ready replicas, API-only enablement, and previous run identifiers.
references.json records the permanent recording and draft identifiers without tokens.
summary.json and cleanup.json record the observed result and shutdown proof.
native-companion.log contains the focused native check result. recording.json and reviewed-draft.json retain the owner-authored source.
No cookie, request-forgery token, pairing token, or device credential enters these evidence files.

### Reinspect without another browser

```sh
python3 scripts/verify-kubernetes-recording.py
```

The verifier reuses the saved recording and draft when native capture is complete.
Reinspection passes in about 0.2 seconds and opens no browser or local target.
An uncertain native attempt requires inspection. Use --retry-after-fix only after its original session is terminal and owned processes are absent.

### Recreate the changed service deployment

Build only the changed common service image. Reuse worker and gateway images.

```sh
node scripts/build-service-image.mjs --arch=arm64
python3 scripts/setup-kubernetes-recording.py --image IMMUTABLE_LOCAL_IMAGE_ID
```

The setup command checks the owned Ready node and idle workers, preserves private previous values, and imports the selected ARM64 image.
It checkpoints the expected Helm revision before upgrading. An uncertain upgrade is observed instead of being replayed.
Every cluster command uses the dedicated Docker socket or private kubeconfig and explicit context.
The setup does not create or select another cluster. Keep its matching image and checkpoint when inspecting the completed upgrade.

Related guides: [desktop companion](recorder-companion.md), [Kubernetes](kubernetes.md), and [deployment readiness](deployment-readiness.md).
