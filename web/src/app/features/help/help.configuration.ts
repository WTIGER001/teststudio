import type { HelpPage } from './help.models'

export const applicationConfigurationHelp: HelpPage = {
  id: 'application-configuration', title: 'Environment configuration',
  introduction: ['Each application owns its suites, test catalog, and environments. An environment is one deployment, such as Development or Production. Define required configuration here and compare every readable environment.'],
  concepts: [
    { title: 'Configuration contract', description: 'Application requirements define named parameters, base URLs, and secret references. Required for every run applies to each selected environment. Other entries apply when selected tests or their dependencies need them.' },
    { title: 'Base URLs', description: 'An HTTP probe combines an environment endpoint with its relative path. Choose another environment to run the same check against a different deployment. Additional endpoint names support separate website and API addresses.' },
    { title: 'Parameters', description: 'Parameters contain nonsecret text, HTTPS URLs, whole numbers, or true/false values. An environment override replaces an application default. A blank saved entry removes the override. Tests declare requiredParameters; HTTP paths and text assertions support {{name}} references.' },
    { title: 'Secrets', description: 'Choose an operator-approved credential reference. Secret values never appear here or in run snapshots. Configured means the reference exists and permits an endpoint origin. Actual availability is checked before execution; credential acceptance remains an execution result. Browser login credentials are not automatically supplied by API credential bindings.' },
    { title: 'Readiness and execution', description: 'The matrix covers application requirements and enabled readable tests. An incomplete environment can be saved. Run review, enabled schedules, dispatch, and retries enforce the requirements of selected tests and their dependencies. Changed parameter values require a new run.' },
    { title: 'Review and conflicts', description: 'Review changes before saving. Save starts no run. Conflicts retain edits and require reloading current configuration before another save. Help and normal refresh preserve edits; Reload configuration explicitly discards them after confirmation.' }
  ],
  tutorials: [
    { title: 'Define a parameter', steps: ['Select Edit requirements, then Add requirement.', 'Name the parameter, choose its value type, and optionally supply a nonsecret default.', 'Choose whether every run needs it. Select Review requirements, then Save configuration.', 'Choose Configure on an environment. Enter its value, review it, and save.'] },
    { title: 'Resolve missing configuration', steps: ['Find missing, invalid, or unavailable entries in the readiness matrix.', 'Select Configure for that environment and enter a base URL or parameter, or choose an approved secret reference.', 'Review and save. If no secret references exist, ask an installation operator to register one for the exact target origin.', 'Review an explicit run of the intended tests. Unrelated optional requirements do not block it.'] }
  ]
}
