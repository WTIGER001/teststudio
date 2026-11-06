# Empty installation and optional examples

A normal TestStudio server starts without projects, applications, environments, suites, tests, runs, or schedules.
Authentication setup can create the initial administrator and built-in access policy. It creates no testing records.
Restarting the server preserves existing records. It does not add examples or self-tests.

Build the UI and start the local server with the README commands.
Open Projects and select Create project. The wizard opens on its own page at /projects/new.
The setup wizard can add your first application, target, and empty suite.
Back to projects and Cancel setup return to the project list. Changed entries require confirmation before leaving.
No cloud account, Docker runtime, or database server is required for native HTTP checks.

## Add only the examples you choose

The new file examples/http/catalog.json contains two independent native HTTP examples.
It contains definitions only. It contains no endpoints, credentials, run history, or copied application data.

1. Create an application named Local service with identifier local-service.
2. Add a local environment for that application. Register your local service URL as endpoint api.
3. Add an Availability suite linked to Local service. Require an environment target.
4. Copy examples/http/catalog.json to your working directory.
5. Remove any test objects you do not want from its tests array. Keep schemaVersion and the selected objects.
6. Change /healthz or / to routes your service supports. Review the safe label against your target.
7. Open Import and sync tests. Select the suite and upload or paste your selected JSON document.
8. Validate and review the proposed definitions. Publish explicitly when the preview matches your intent.
9. Open Test runs, choose your local environment, and explicitly start the selected check.

Choose another application identifier by updating applicationId in each selected definition before upload.
Preview does not install definitions. Publication installs only the supplied definitions and creates no run.
Omitted examples stay absent from a fresh suite. Omission does not remove tests already in an existing suite.
Use the catalog's explicit Archive action to remove existing checks from normal selection.
See [Source uploads](sources.md) and [Catalog management](catalog.md) for validation, conflicts, and retained source.

## Full guided demonstration

The explicit node scripts/demo.mjs start command creates its separate demonstration installation.
It selects the complete new demonstration package and runs its passing, failing, timeout, and browser checks.
That command requires Docker for the browser example. It does not populate a normal server database.
See [Local demonstration](demo.md) for its storage, restart behavior, and walkthrough.
