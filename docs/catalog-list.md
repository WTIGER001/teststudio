# Catalog lists and pagination

## Browse working definitions

The catalog uses the same filtered list API as external clients.
Search, suite, application, type, metadata, lifecycle, category scope, and sort controls combine on the server.
The table starts with 30 matching tests. Load more tests appends another page.
Showing N of M reports loaded tests and the complete authorized match count.
Category branches count matching records across all pages, with the category filter itself excluded.
This preserves sibling navigation while a category is selected.
Project summary metrics count readable active definitions. Archived definitions remain available through the lifecycle filter.
Metadata options contain only values from readable definitions.

Filters and page changes preserve selected test identities across suites.
Hidden selection counts include selected tests outside the loaded results.
Select visible tests replaces selection with loaded matching rows. Bulk operations still require one suite and at most 30 tests.
Saved selections retain their separate limits and resolved release preview.

## Recover from changed data

A cursor belongs to one actor, project, route, normalized query, page size, and authorized catalog snapshot.
An edited or newly denied definition changes that snapshot.
The server rejects the old cursor instead of silently skipping or duplicating results.
Reload results starts at the first page with the same filter criteria. Select tests again after this conflict.
Access errors hide results and selections. Refresh access before reloading results.
Unavailable data shows a recovery message instead of an empty-catalog claim.

## List API

Use `GET /api/v1/projects/{project}/tests` for project-wide browsing.
Use `GET /api/v1/projects/{project}/suites/{suite}/tests` for a fixed suite.
Both routes default to 30 records. Follow `cursor` until it is absent.
The self-test registrar follows suite cursors before updating existing definitions.

| Query | Behavior |
| --- | --- |
| `query` | Match identifier, title, description, category, suite name, or tags without case distinctions. |
| `suite`, `application` | Match exact resource identifiers. A fixed suite route rejects a different suite filter. |
| `category` | Match a normalized slash-separated category path. Spaces beside separators do not change the path. |
| `descendants` | Include child categories. HTTP requests default to true. |
| `uncategorized` | Select definitions without a category. Do not combine it with a category path. |
| `tag`, `owner`, `type`, `runner`, `safety` | Match an exact metadata value. |
| `source` | Match the executable source or registered definition source identifier. |
| `state` | Use all, active, draft, enabled, disabled, or archived. API requests default to all; the UI starts with active. |
| `sort` | Use title, title-desc, id, category, or updated. The default is title. |
| `limit` | Return 1 through 100 records per page. The default is 30. |
| `cursor` | Continue the same query. Treat the value as opaque. |

Name sorting ignores case first. Exact title, suite identifier, and test identifier resolve equal values.
Updated sorts the most recent record first. Sorting does not change execution order.
Filter text accepts at most 200 bytes per field without control characters.

Responses include items, total readable definitions, matched count, active summary counts, category branches, metadata options, and an optional cursor.
Item data uses the existing test definition contract. Suite names and revision information accompany each item.
Null dependency and binding arrays remain unchanged when the UI reads or edits a definition.

Cursors use the installation's persistent authenticated encryption helper with a separate catalog key namespace.
They expire after 15 minutes from issue. Query, actor, route, and project mismatches return a validation error.
Expired cursors or changed authorized catalog data return a conflict. Start a fresh first page after that response.
Current project, suite, and exact test permissions apply before rows, counts, categories, or options enter the response.
The current implementation computes the authorized view before slicing its page. It does not claim indexed catalog search.
OpenAPI 0.28 and the generated client describe both list operations.
