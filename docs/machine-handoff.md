# Continue on another machine

## Source and reading order

Continue on branch codex/implementation. Transfer the current local Git history before starting from a new clone.
The 2026-11-16 audit finds diverged cached origin tracking and a failed GitHub SSH check. The cached references do not establish the current remote state.
Do not assume a GitHub clone includes the local work. Do not reset, force-push, or merge the divergence merely to start development.
The source must include f08dbdf, ea5d4bc, and 048b3de, followed by this handoff documentation.
These commits retire item 54, add the staged worker tooling, and record verified worker infrastructure.

Read these files in order:

1. [Repository instructions](../AGENTS.md).
2. The active sections at the start of [continuation notes](continue.md).
3. The active sections at the start of [the plan](plan.md).
4. Acceptance criteria in [the ledger](implementation.md#acceptance-criteria).
5. [AWS worker stages](aws-workers-package.md) and [MMDev operations](aws-mmdev.md).

The current scope is 50 verified of 56 active criteria, with six open. Item 54 is retired; do not restore the multiple-provider acceptance requirement.
Continue item 51. Azure implementation and verification remain deferred.
The next milestone is private DNS verification, disabled worker-VPC DNS failure-open, and reviewed connect and ready stages for the existing worker stack.
The tracked deployment helper currently stops at prepare. Extend it for later stages instead of rerunning initial provisioning.

## Portable source transfer

A Git bundle transfers committed history without relying on GitHub authentication or publishing the branch.
The prepared handoff files are under the old checkout's ignored output/handoff/2026-11-16 directory:

| File | Contents |
| --- | --- |
| teststudio-source.bundle | Committed Git history and local refs, including codex/implementation and this guide. |
| teststudio-worker-receipts.tar.gz | Only the four named worker deployment and recovery receipts below. |
| SHA256SUMS | Checksums for both transfer artifacts. |
| START-HERE.txt | Exported commit identity and restore commands. |

Copy these files privately to the new machine. Verify their checksums against SHA256SUMS before importing.
The bundle contains committed Git content. It does not transfer uncommitted edits from other worktrees, ignored local data, credentials, or running services.
The receipt archive contains private operational references and historical evidence, without credential values.
Neither artifact creates a Git push or changes an AWS resource.

From the directory containing the transferred files, run:

```sh
git clone --branch codex/implementation teststudio-source.bundle teststudio
git -C teststudio bundle verify ../teststudio-source.bundle
tar -xzf teststudio-worker-receipts.tar.gz -C teststudio
cd teststudio
git status --short
git branch --show-current
git log -4 --oneline
```

The clone initially uses the bundle path as origin. After configuring GitHub access, restore the intended remote URL:

```sh
git remote set-url origin git@github.com:wtiger001/teststudio.git
```

A later fetch can inspect the real remote divergence. Reconcile it deliberately before any separately authorized push.
The export can be recreated from the current checkout with git bundle create and the same four receipt files. A later source change requires a new export.

## Local files and credentials

Git ignores .data and output. A plain clone does not include deployment receipts, test evidence, runtime images, virtual environments, local databases, private origin maps, or the demo's private UI snapshot.
The current AWS handoff needs these private receipts at their original repository-relative locations:

- output/aws-mmdev/workers-051/bootstrap.json
- output/aws-mmdev/workers-051/workers.json
- output/aws-mmdev/workers-051/workers-failed.json
- output/aws-mmdev/workers-051/restore.json

Keep these files private and outside Git. The first two record verified deployed inputs; the last two preserve rollback and retained-resource import history.
Without them, source development can continue. AWS continuation needs a read-only inventory and an explicit new baseline before extending the tooling.
Do not fabricate historical review or verification results, bypass receipt guards, or provision duplicate stacks because receipts are missing.
The tracked notes preserve the verified findings. Missing old screenshots or raw self-test output alone does not require repeating passing tests.

Authenticate AWS and GitHub independently on the new machine. The authorized AWS target remains profile mm_dev, account 253455159142, region us-west-2, and controller cluster marketmaker-testing-dev.
Use approved access to existing Secrets Manager references. Do not put credential payloads in the checkout, transfer archive, or chat.
The hosted GitHub credential already exists in AWS. Continuing the worker infrastructure does not require copying the old workstation's GitHub token.
No Azure credentials or inherited Kubernetes context are required for this milestone.

Recreate tools for the new machine rather than copying virtual environments or build caches:

```sh
python3 -m venv .data/aws-tools/venv
.data/aws-tools/venv/bin/pip install -r scripts/aws-check-requirements.txt
```

For application work, use the Go and Node requirements in [the README](../README.md#local-development), the locked web dependencies, and Docker when the selected workflow requires it.
The deployed service and worker architectures follow their AWS settings, not the new development machine's architecture.
The separate marketmaker-tests adapter remains in its own repository and is not required to build TestStudio.
The old workstation's demo, local cluster, browsers, databases, and workloads stay on that workstation. Do not start a replacement demo merely to resume item 51.

## Inspect before changing AWS

The last verified state is from 2026-11-14. It is historical evidence, not a fresh live check.
The bootstrap stack is teststudioapi-workers-bootstrap. The worker stack is teststudioapi-workers at Stage=prepare with 46 resources.
Preparation, new builds, and environment workers are disabled. No worker settings secret or consumer exchange endpoint is released in that stage.
The core application remains 0.16.4-dev, build 7dbb3638289f. The worker milestone does not roll out the core or recording service.
Both original ECS service deployments are preserved in the receipts. Recheck current state before mutations.

Start with read-only identity and stack inspection:

```sh
aws sts get-caller-identity --profile mm_dev --region us-west-2 --query Account --output text
aws cloudformation describe-stacks --stack-name teststudioapi-workers \
  --profile mm_dev --region us-west-2 \
  --query 'Stacks[0].{Status:StackStatus,Outputs:Outputs}'
```

Confirm the expected account, existing stack ownership, current stage, and any deployment already in progress.
Preserve the core, testing-server, recording configuration, GitHub bindings, and retained journals and repositories.
Review each concrete CloudFormation change set before execution. Core health does not prove coded-worker readiness.

## Starting prompt

Use this prompt from the new checkout:

```text
Read AGENTS.md, docs/machine-handoff.md, the active sections of docs/continue.md and docs/plan.md, and the acceptance criteria in docs/implementation.md.
Continue acceptance 51 from the existing MMDev worker prepare stage. Item 54 is retired. Azure remains deferred.
Check source identity and private receipt availability. Inspect AWS read-only before proposing mutations; do not recreate the existing stacks.
Extend tracked tooling for private DNS verification, fail-closed DNS setup, and reviewed connect and ready stages. Keep preparation and execution disabled until their prerequisites pass.
Use targeted local checks, preserve existing workloads, update continuation notes, and commit verified work locally. Do not push.
```
