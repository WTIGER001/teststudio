# Page help

Select Help in the header on any page, including sign-in.
A drawer slides in from the right and shows the guide for the current page.
The drawer uses the current color theme. Reduced-motion preferences remove its animation.

Each guide contains three sections:

- Introduction explains the page and its purpose.
- Concepts explains the terms and behavior used on the page.
- Tutorials provides numbered steps for common tasks.

Use the section buttons to move through the guide.
The help content scrolls separately from the page.
On narrow screens, the drawer uses the full viewport width.

Select Close or press Escape to return to the page.
On wider screens, selecting the backdrop also closes help.
Focus returns to the Help button. Unsaved page fields remain in place.

## Guide coverage

The guides cover sign-in, projects, the test catalog, project settings, test runs, run details, trends, schedules, and notifications.
Users, User details, Groups, Group details, Project access, and Installation access provide administration guides.
Sign-in help appears whenever the user has no active session.
Signed-in pages select their guide from the route's `helpTopic` data.
Unmapped routes use the projects guide after normal route handling.

## Maintain the guides

Edit structured guide content in `web/src/app/features/help/help.pages.ts`.
Each guide requires a title, introduction paragraphs, concepts, and tutorials.
Write tutorials for available controls and use their visible labels.
Update the guide when its page changes.

Register each new page with a `helpTopic` value in `web/src/app/app.routes.ts`.
Add its content under the same key in `help.pages.ts`.
The shared help service selects content. The drawer component handles display and keyboard interaction.
Help content ships with the application and requires no additional service or database.

## Local verification

The `STUDIO-HELP-001` Playwright test runs through TestStudio's normal execution and evidence flow.
Run only this workflow with:

```sh
bash scripts/test-self.sh --test=STUDIO-HELP-001
```

The test checks all current guides, keyboard focus, dismissal, preserved edits, themes, mobile layout, and reduced motion.
