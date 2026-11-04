# Independent product material

Acceptance 02 covers the material distributed in this repository and the records created by a normal initial installation.
TestStudio contains newly authored product tests, generic fixtures, selected examples, and independent configuration.
It imports no MarketMaker tests, catalogs, credentials, environments, reports, or historical records.
User-selected uploads after installation remain ordinary project input. They are outside this initial distribution audit.

## Evidence boundaries

The repository starts at beb1c7b with five files: the plan, ledger, agent instructions, ignore rules, and a module declaration.
That root contains no tests, application data, configuration bundle, imported repository history, or executable implementation.
Later feature commits introduce the product and its validation tests together.
The first browser self-test commit is 554739d. It creates a new isolated TestStudio verification suite.
New Go and browser ZIP fixtures appear in 3b90043 and 8a18764 beside their new upload workflows.
Their ten archive members contain generic tests, TestStudio catalogs, runtime declarations, and source manifests.
The audit reads their contents without extracting them into a legacy checkout.

Configuration examples use TestStudio identifiers, sample identity endpoints, approved secret references, and operator-selected trust paths.
They contain no inherited application account, identity pool, test credential, or existing test history.
The runtime, gateway, discovery, self-test, and example container definitions belong to this product.
Build dependencies include the explicitly selected permission library and normal published third-party packages.
Shared third-party dependencies do not constitute imported application tests or data.

Acceptance 01 retains a passing build from an isolated source copy with no sibling checkout or local module replacement.
Its evidence is .data/independent-build._o3cvq4v/result.json and package-dependencies.json.
Acceptance 03 retains a passing fresh-start and restart check with empty testing registries and explicitly selected examples.
Its evidence is output/empty-installation/local-qoUlbQ/summary.json.
Generated databases, browser reports, recordings, local credentials, build output, and demonstration data remain outside tracked distribution files.
Normal startup adds no demonstration or self-test records.

## Read-only content comparison

Use scripts/verify-product-provenance.py with explicitly selected reference repositories.
It compares tracked product files and checked-in ZIP members with reference HEAD and the latest HEAD ancestor before product creation.
It also compares current tracked reference files, including uncommitted content.
Exact comparisons use Git blob identifiers. Current text comparisons also remove whitespace before comparison.
The report includes a SHA-256 inventory of the product candidates.
Any overlap returns exit status 1 and requires review. The script never imports or modifies reference material.

```sh
python3 scripts/verify-product-provenance.py \
  --reference-repository /path/to/legacy-checkout \
  --reference-repository /path/to/another-legacy-checkout \
  --output output/provenance/content-comparison.json
```

The completed local audit covers nine available legacy checkouts, including the original testing-platform and dedicated test repositories.
It examines their reference snapshots and 16942 tracked working files.
The product files and all ten ZIP members have no exact or whitespace-normalized matches.
A newly authored temporary reference confirms that deliberate exact and whitespace-only copies return a failing audit.
That temporary fixture is removed after the check.

Content comparison alone does not establish authorship.
This acceptance combines comparison with repository history, fixture review, example configuration review, independent builds, and empty-start evidence.
Automated comparison does not cover arbitrary rewritten derivations or untracked legacy material.
Whitespace comparison covers text of at least 80 normalized characters and tracked reference files up to 8 MiB.
The report records large, missing, and symbolic-link working files. Committed snapshot comparison remains separate.
Reports can contain local reference paths. Keep them in the ignored output directory and outside published application artifacts.
No browser, product rebuild, cloud request, or deployment is required for this audit.
