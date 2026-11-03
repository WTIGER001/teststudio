# Local demonstration

This demonstration uses new generic checks and a separate SQLite database.
It uses no cloud account, existing application tests, or production endpoints.
Normal installations stay empty. Only the explicit demo command loads these examples.

## Start

Use the Go and Node versions listed in the README. Install npm and start Docker.
From the repository directory, run:

```sh
node scripts/demo.mjs start
```

The first start installs UI dependencies, builds the application, and prepares a trusted browser runtime.
Initial dependency downloads require network access. Test execution uses local resources.
Later starts reuse unchanged builds and preserve your edits, history, and evidence.
Each start serves a private snapshot of its compiled UI files.
Later development builds do not replace assets used by that running demo.
Setup creates three HTTP runs and one browser run only once.
The terminal prints the application address, project address, and local target address.
Ports default to 8040, 8041, and 8060. A new installation selects free ports when needed.
An existing installation refuses to start if another process uses its saved ports.

Open the application address and select **Open local workspace**.
Open **TestStudio local demo**.

## Explore applications

Select Applications in the sidebar. Open an application to see its environments and linked suites.
View tests opens a suite-filtered catalog. Create test keeps that suite selected through the type chooser.
Authorized users add applications, environments, and linked suites through Name, Options, and Review steps.
See applications.md for details.

## Create a test

Select Create test from the catalog. Choose Playwright browser test, HTTPS probe, or Go suite.
The dedicated creation page shows only the selected type's workflow.
HTTPS probes follow Name, Request, and Review. Named Advanced sections hold uncommon options.
Save draft retains a definition. Save and run explicitly publishes a snapshot and starts the chosen local check.
The suite's active release stays unchanged.
Projects also provides a wizard for a first application, optional target, and empty suite.
Test runs uses Selection and Review steps. Saved selections, limits, and retries stay under Advanced options.
The local demo enables native Playwright recording on this computer.
Recorded journeys can be reviewed and saved. Website execution remains pending.
See create-tests.md and browser-recording.md for details.

## Walkthrough

1. Open **Test catalog**. Inspect the three availability checks and the browser check.
   The catalog contains a healthy check, a recoverable failure, and a deliberate one-second timeout.
2. Open **Test runs**. The prepared history contains passing, failed, and timed-out results.
   Open the availability run with three checks. Inspect its result counts, attempts, logs, and measured durations.
3. Open the passing browser run. In **Run artifacts**, select **View** for its PNG screenshot.
   The screenshot shows the new Parcel desk example. The test needs no external website.
4. Return to the catalog. Edit the healthy check's title and save its draft changes.
   Under **Publish a release**, select **Availability examples**, then **Validate and preview** and **Publish and activate**.
   The existing runs keep their original release and results.
5. Open **Test runs** and start an availability run against **Local demo target**.
   The healthy check passes, the unavailable check fails, and the slow check times out.
6. Open the printed local target address in another tab. Select **Recover the example service**.
   Return to the three-check run and select **Rerun failures**.
   The rerun selects the failed and timed-out checks. The availability check passes; the deliberate timeout remains.
7. Open **Test timing** on a run. Expand **Manage comparable timing history** and inspect the healthy check's baseline.
   Setup records three healthy observations. New comparable runs can use their duration history.
   Predictions need enough matching observations. Failed and timed-out attempts do not train a healthy baseline.
8. Open **Trends**. Inspect the stacked areas and their accessible values.
   Enter `availability` in **Suite identifier** and select **Apply scope** to focus on the controlled status changes.
   Select **Help** on any page to open its Introduction, Concepts, and Tutorials.

The deliberate timeout remains a distinct status. It is expected demonstration behavior.
The target resets to unavailable after a restart. Stored test results do not change.

## Stop and resume

Use Ctrl+C in the launcher terminal, or run:

```sh
node scripts/demo.mjs stop
node scripts/demo.mjs status
```

The launcher stops its server, source worker, target, and control listener.
The server cleans up its own execution resources. Shared Docker runtimes stay cached.
The database, source images, and evidence remain available in `.data/demo`.
Run the same start command to resume. Existing demonstrations are not seeded again.
Process errors are recorded in `.data/demo/processes.log`.

For a separate clean demonstration, choose a new directory:

```sh
node scripts/demo.mjs start --data .data/another-demo
node scripts/demo.mjs stop --data .data/another-demo
```

The launcher refuses an existing nonempty directory without its demo marker.
Local mode provides administrator access on this computer. All listeners bind to loopback.
Cloud deployment verification remains separate work.
