# TestStudio

Build the independent Go and Angular product described in `docs/plan.md`.
Keep the complete acceptance ledger in `docs/implementation.md`.
Use the active delivery sequence in `docs/plan.md`. Historical checkpoints are reference material, not the current task queue.

- Use standalone Angular components, OnPush, signals, small services, and pure utilities.
- Keep components below 450 lines and keep workflows outside components.
- Use semantic HTML and shared light and dark theme tokens.
- Follow `docs/ui.md` for tester workflows, navigation, wizards, Advanced options, and accessible feedback.
- Update affected page help and user, API, or operator guides with behavior changes. Follow the completion checklist in `docs/plan.md`.
- Keep documentation-only validation to document consistency and links. Do not rebuild or rerun browser suites for prose changes.
- Use `github.com/wtiger001/go-permissions` through `permissions.Service`.
- Store identity and memberships in TestStudio. Enforce authorization after parsing resource context.
- Keep permission checks inside handlers or domain services.
- Report progress against the stable 57 acceptance numbers in docs/implementation.md.
- Report verified totals, advanced item numbers, and the targeted local checks used.
- Use PostgreSQL for hosted deployments and SQLite for single-instance local deployments.
- Add no code comments unless the user requests them.
- Write new tests and examples. Copy no existing application tests, data, or infrastructure configuration into TestStudio.
- The user authorizes adapting the separate marketmaker-tests repository and documenting a reusable test-repository contract. Keep application-specific adapters in that repository.
- Focus on implementation, then run targeted local tests for changed behavior.
- The user defers Azure implementation and verification. Preserve staged code. Prepare an Azure deployment script only when the user requests that work.
- Run Kubernetes verification only on a dedicated local k3s/k3d cluster with an explicit private kubeconfig. Do not use a remote or inherited cluster context.
- The user authorizes AWS deployment to the MMDev ECS cluster as teststudioapi. Use the mm_dev profile and verify the selected account before mutations.
- Keep AWS changes scoped to TestStudio-owned resources and the new service. Preserve existing ECS services. Azure remains deferred.
- The user selects external Cognito pool us-west-2_p1Epu0go8 and authorizes a dedicated TestStudio OIDC client through mm_prod. Keep the application generic OIDC. Identity-provider federation stays in Cognito.
- Do not repeat broad suites or fixture builds without a concrete unresolved concern.
- Follow docs/versioning.md. Use scripts/version.mjs to bump the release before publishing changed application behavior. Documentation-only changes need no version bump.
- Build published release artifacts from committed source. Keep immutable build dates distinct from deployment and startup dates.
- Commit verified features frequently, as the user requests.
- Do not push without an explicit request.
- Close every browser session started for verification and verify shutdown. Preserve unrelated browser windows.
- Use short, active sentences in documentation.
