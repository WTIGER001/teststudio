# Project images

## Introduction

Projects uses square tiles. Cards have a bounded width and tile across the available space.
Small screens use a single column. A project with no image uses the default background.
Select the tile to open its applications. Select Image to change its background.
Changing an image requires project read and project management permission. Viewing an image requires project read permission.

## Concepts

Images are optional and belong to the project. They do not change applications, tests, releases, or run evidence.
The title stays visible over a dark overlay. The image fills the square and crops at the center.
Uploads accept PNG, JPEG, or WebP files up to 10 MB.
The browser resizes the image and converts it to a small JPEG. The server validates and re-encodes that image.
Saved images have a 256 KiB limit and a maximum of 1600 pixels per side. The browser uses a 1200-pixel maximum.
The server stores images in the installation database. PostgreSQL and SQLite use the same record contract.
Database backup and transfer include these records. Images need no public bucket or external image service.
Authenticated image responses use private, no-store caching. Image bytes stay out of audit events and ordinary project records.

Capture uses your browser sharing prompt. It captures the tab you select, after you open the requested website.
It does not fetch and render a URL on the server. The website can use your existing browser sign-in.
Verify the selected page in the preview. TestStudio captures one frame and stops the sharing tracks immediately afterward.
Cancellation, capture errors, and page departure also stop active tracks. No test runs during image capture.
Capture requires browser support and a secure browser context, such as HTTPS or localhost.
If screen sharing is unavailable or denied, upload a screenshot instead.

## Tutorials

### Upload an image

1. Open Projects and select Image on your project tile.
2. Choose a file under Upload a screenshot or image.
3. Review the square preview and visible project title.
4. Select Save image. Cancel keeps the saved background.

### Capture a website

1. Open the image editor and expand Capture from a website.
2. Enter the website address and select Open website.
3. Return to TestStudio and select Capture website tab.
4. Select the website tab in your browser sharing prompt.
5. Review the captured frame. Select Save image to use it.

### Remove or recover an image

Select Remove image and Save image to restore the default background.
Cancel or Escape asks before discarding an unsaved image change.
A conflicting or unconfirmed save requires Reload current image before further edits. Reload discards the pending preview.

## API contract

| Endpoint | Behavior |
| --- | --- |
| GET /api/v1/project-covers | Returns items with id, revision, hasImage, and canManage for readable projects. Contains no image bytes. |
| GET /api/v1/projects/{project}/cover | Returns metadata and cover with image, source, and optional url. An absent cover has revision zero. |
| PUT /api/v1/projects/{project}/cover | Requires project management permission, CSRF, and the exact If-Match cover revision. Returns the saved cover. |
| GET /api/v1/projects/{project}/cover/content | Returns an authenticated JPEG with private, no-store caching. An absent image returns 404. |

PUT accepts image as a JPEG data URL and source as upload or capture. Optional url records an HTTP or HTTPS address without credentials.
Set image to an empty string to remove it. Removal retains the cover revision and rejects stale updates.
The image revision is separate from the project settings revision. A stale cover revision returns 409.
Invalid image formats, sizes, dimensions, sources, and addresses return 422. Unauthorized access returns 401 or 403.
The server checks current access for each operation. It commits cover changes and bounded audit metadata in one transaction.
After an unconfirmed response, read the saved cover before another write.
