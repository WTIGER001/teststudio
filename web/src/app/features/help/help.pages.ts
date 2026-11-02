import type { HelpPage } from './help.models'

export const helpPages: Record<string, HelpPage> = {
  signin: {
    id: 'signin', title: 'Sign-in',
    introduction: ['Sign in to open your TestStudio workspace. Your installation controls which sign-in options appear.'],
    concepts: [
      { title: 'Your workspace', description: 'Projects organize applications, suites, test definitions, and run history. Your permissions control which actions are available.' },
      { title: 'Local access', description: 'Open local workspace appears only when local administrator sign-in is enabled for this installation.' },
      { title: 'Session', description: 'If your session expires, sign in again. The page address still identifies the work you were viewing.' }
    ],
    tutorials: [
      { title: 'Open your workspace', steps: ['Choose your organization’s sign-in provider, or select Open local workspace on a local installation.', 'Complete sign-in. TestStudio opens the requested page.', 'Use the Workspace selector to choose a project.'] },
      { title: 'If no sign-in option appears', steps: ['Ask your installation administrator to configure a sign-in provider.', 'Reload this page after the administrator confirms the setup.'] }
    ]
  },
  projects: {
    id: 'projects', title: 'Projects',
    introduction: ['Projects keep related applications, test suites, and results together. Open a project to manage its catalog and runs.'],
    concepts: [
      { title: 'Project', description: 'A project provides the shared home for a test catalog, configuration, and run history.' },
      { title: 'Permanent identifier', description: 'Identifiers stay stable when display names change. Use letters, numbers, periods, underscores, or hyphens.' },
      { title: 'Permissions', description: 'The projects and controls you see depend on your access. Creating a project requires project creation permission.' }
    ],
    tutorials: [
      { title: 'Create your first project', steps: ['Enter a project name and permanent identifier in Create a project.', 'Add an optional description, then select Create project.', 'Open the project and use Project settings to register a suite.'] },
      { title: 'Return to existing work', steps: ['Open a project card to view its test catalog.', 'Use Test runs for execution history or Project settings for applications, suites, and environments.', 'Use the Workspace selector to switch projects.'] }
    ]
  },
  catalog: {
    id: 'catalog', title: 'Test catalog',
    introduction: ['The catalog describes what each test checks and how TestStudio runs it. Draft edits become executable through a published suite release.'],
    concepts: [
      { title: 'Test identity and lifecycle', description: 'A permanent identifier connects a test to its history. Enabled tests can be selected. Disabled and archived tests retain their definitions.' },
      { title: 'Draft and release', description: 'Editing changes the draft. Publication freezes a release. Validation checks definitions; it does not execute tests or prove they pass.' },
      { title: 'Safety', description: 'Each test has a safety label and rationale. Protected targets require safe definitions and the required permissions.' },
      { title: 'Timing policy', description: 'Comparable clean runs build a duration baseline. Timing rules can warn or fail on slow execution and recommend bounded timeouts.' },
      { title: 'Catalog transfer', description: 'Import validates a JSON catalog before changes are applied. Export downloads definitions. Neither action executes tests.' }
    ],
    tutorials: [
      { title: 'Create and publish an HTTPS check', steps: ['Create a suite in Project settings if none exists.', 'Select + Create test. Choose the suite and enter an identifier, title, and category.', 'Enter the registered endpoint name and relative path. The editor creates a GET check that expects HTTP 200.', 'Set the timeout ceiling, safety label, and rationale. Select Save draft changes.', 'Choose the suite under Publish a release. Select Validate and preview, review the result, then Publish and activate.'] },
      { title: 'Find and maintain a test', steps: ['Search by title, identifier, category, or tag. Use suite and lifecycle filters to narrow the list.', 'Use History to inspect retained definitions, or Edit to change the draft.', 'Use Archive to remove a test from normal selection. Choose Archived in the lifecycle filter to find and restore it.', 'Publish a new release when draft changes should apply to future runs.'] }
    ]
  },
  configuration: {
    id: 'configuration', title: 'Project settings',
    introduction: ['Register the applications you test, the suites that group their tests, and the environments those tests can target.'],
    concepts: [
      { title: 'Application', description: 'An application identifies the system being tested. Environments belong to an application.' },
      { title: 'Suite', description: 'A suite groups test definitions and published releases. A suite can require a target environment or run without one.' },
      { title: 'Environment and endpoint', description: 'An environment records a stage and named endpoint URLs. Test definitions refer to endpoint names instead of embedding a target URL.' },
      { title: 'Protection', description: 'Protection is separate from the stage name. A protected environment requires safe tests and protected-target permissions.' }
    ],
    tutorials: [
      { title: 'Prepare a target for HTTPS checks', steps: ['Choose Applications. Enter a name and permanent identifier, then select Create.', 'Choose Environments. Select the application, enter a name and identifier, and choose the stage.', 'Enter an endpoint name and URL. Use this endpoint name in test definitions.', 'Choose whether the environment needs production-safe tests, then select Create.'] },
      { title: 'Register a suite', steps: ['Choose Suites and enter a name and permanent identifier.', 'Choose an application when the suite belongs to one.', 'Keep Requires a target environment enabled for checks that use registered endpoints. Turn it off for tests that need no target.', 'Select Create, then open Test catalog to add and publish definitions.'] }
    ]
  },
  runs: {
    id: 'runs', title: 'Test runs',
    introduction: ['Start tests from a published suite and browse earlier runs. Each run captures its definitions, target settings, and timing criteria.'],
    concepts: [
      { title: 'Published suite', description: 'The run form lists suites with active published releases. Draft edits do not change existing runs.' },
      { title: 'Selection and dependencies', description: 'Leave test identifiers empty to select all eligible tests. An explicit selection also includes required dependencies.' },
      { title: 'Time limits and retries', description: 'The overall run deadline includes queue time. Each attempt also has a timeout. Retries remain separate attempts within one logical test result.' },
      { title: 'Timing estimate', description: 'Estimates use comparable history. Missing history stays unknown. The timeout budget includes permitted retries; expected execution excludes preparation and capture.' }
    ],
    tutorials: [
      { title: 'Start a run', steps: ['Select + New run, then choose a published suite in Start a run.', 'Choose a target environment when the suite requires one. Use No target only for suites that support it.', 'Optionally enter test identifiers. Set the run time limit and retries per test.', 'Select Estimate timing to review available history and timeout budgets.', 'Select Start run. Open the resulting details to follow progress and inspect evidence.'] },
      { title: 'Find an earlier run', steps: ['Use the history filters to narrow the suite, environment, state, or selection.', 'Open a run to inspect its frozen settings, results, attempts, and evidence.', 'Use the pagination controls to browse more history.'] }
    ]
  },
  run: {
    id: 'run', title: 'Run details',
    introduction: ['This page explains one run: what it used, what happened, how long tests took, and which evidence remains available.'],
    concepts: [
      { title: 'Run state and outcome', description: 'State describes execution progress. Outcome describes the result. An incomplete run retains uncertainty instead of becoming a pass.' },
      { title: 'Results and attempts', description: 'Final counts use logical test results. Attempt history keeps each retry. Timeouts have a separate timed-out status and count. A successful retry can leave a flaky result.' },
      { title: 'Test timing', description: 'Frozen predictions explain the criteria used for this run. Worker elapsed time includes coordination. Duration unavailable means no valid test measurement exists.' },
      { title: 'Evidence and retention', description: 'Logs and artifacts have capture and retention policies. Capture gaps do not erase test outcomes. Pins preserve evidence; deletion removes files while retaining results.' },
      { title: 'Worker recovery', description: 'After an interruption, recovery retains available diagnostics and removes owned resources. Recovered observations keep the original run outcome.' }
    ],
    tutorials: [
      { title: 'Investigate a failed or incomplete run', steps: ['Read the run summary and any execution or capture warning.', 'Compare final results with Attempt history. Expand recorded steps when available.', 'Inspect Run logs and Execution events. Filter logs by stream or test identifier when needed.', 'Open retained screenshots or other artifacts. Check Worker resources for pending cleanup or recovery gaps.', 'Use Rerun failures to repeat failed, timed-out, and flaky tests in a separate run. The original record remains unchanged.'] },
      { title: 'Understand timing and preserve evidence', steps: ['Review Test timing for the predictions and timeout criteria frozen when this run was queued.', 'Compare measured durations, expected values, and slowdown ratios. Functional outcomes remain visible beside timing failures.', 'Use the retention controls to pin evidence that needs further review.', 'If current timing history needs a reset, open Manage comparable timing history and provide a reason. Reset requires permission and does not change historical results.'] }
    ]
  }
}
