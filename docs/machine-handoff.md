# Continue on another machine

## Source and reading order

Continue on branch main, which is now the GitHub default. The user selects main on 2026-11-16. It starts from published implementation tip 465d2aa and includes the complete merged implementation and delivery history.
The earlier codex/implementation and codex/status-page-logo branches remain preserved. Use main for a new clone, or restore the older committed source bundle and then fetch main.
The initial 2026-11-16 audit finds diverged cached origin tracking and a failed GitHub SSH check.
GitHub access is then repaired for this checkout using HTTPS and the existing authenticated GitHub CLI credential helper. A normal fetch succeeds.
That fetch confirms 44 local-only commits and one remote-only documentation commit before the authentication documentation update. The user then authorizes merging and pushing on 2026-11-16.
The merge preserves the remote 0.8.2 deployment documentation and the later local product and worker history. Publication uses an ordinary push without rewriting history.
Verify the published branch before continuing. Do not reset or force-push over another machine's work.
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

Authenticate GitHub CLI on the new machine and clone main directly:

```sh
git -c credential.helper= -c 'credential.helper=!gh auth git-credential' \
  clone --branch main https://github.com/WTIGER001/teststudio.git teststudio
git -C teststudio config --local --replace-all credential.https://github.com.helper ''
git -C teststudio config --local --add credential.https://github.com.helper '!gh auth git-credential'
```

Transfer the private receipt archive separately and extract it with tar -xzf teststudio-worker-receipts.tar.gz -C teststudio. GitHub does not contain these ignored receipts.

A Git bundle transfers committed history without relying on GitHub authentication or publishing the branch.
The fallback handoff files are under the old checkout's ignored output/handoff/2026-11-16-github directory. Their source snapshot is db6d69b, before the publication merge. Fetch the published branch after restoring that snapshot to obtain the later merge documentation.
The private receipt archive remains valid because publication changes no AWS resources or receipts. These files supersede the earlier 2026-11-16 export:

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

The clone initially uses the bundle path as origin. Authenticate GitHub CLI on the new machine, then configure this checkout to use HTTPS and that credential helper:

```sh
git remote set-url origin https://github.com/WTIGER001/teststudio.git
git config --local --replace-all credential.https://github.com.helper ''
git config --local --add credential.https://github.com.helper '!gh auth git-credential'
git fetch origin
git switch --track origin/main
```

The configuration stays local to this checkout and uses GitHub CLI's credential storage. It does not put a token in the remote URL or repository files.
Inspect the fetched branch state before continuing. The authorized publication incorporates the earlier remote-only documentation commit. Reconcile any later divergence deliberately before another separately authorized push.
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
