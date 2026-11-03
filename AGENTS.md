# TestStudio

Build the independent Go and Angular product described in `docs/plan.md`.
Keep the complete acceptance ledger in `docs/implementation.md`.

- Use standalone Angular components, OnPush, signals, small services, and pure utilities.
- Keep components below 450 lines and keep workflows outside components.
- Use semantic HTML and shared light and dark theme tokens.
- Use `github.com/wtiger001/go-permissions` through `permissions.Service`.
- Store identity and memberships in TestStudio. Enforce authorization after parsing resource context.
- Keep permission checks inside handlers or domain services.
- Report progress against the stable 57 acceptance numbers in docs/implementation.md.
- Report verified totals, advanced item numbers, and the targeted local checks used.
- Use PostgreSQL for hosted deployments and SQLite for single-instance local deployments.
- Add no code comments unless the user requests them.
- Write new tests and examples. Copy no existing application tests, data, or infrastructure configuration.
- Focus on implementation, then run targeted local tests for changed behavior.
- Do not run AWS, Kubernetes, or Azure tests in this work session.
- Do not repeat broad suites or fixture builds without a concrete unresolved concern.
- Commit verified features frequently, as the user requests.
- Do not push without an explicit request.
- Close every browser session started for verification and verify shutdown. Preserve unrelated browser windows.
- Use short, active sentences in documentation.
