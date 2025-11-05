# Releasing Better Fabric Monitor

This guide is for maintainers responsible for publishing new versions. It complements the automated release assistant prompt at `.github/prompts/release-workflow-automated.md.prompt.md`.

## Branching Model

- Use GitHub Flow: branch from `main`, commit fixes, and open a pull request.
- Prefer squash merges so the pull-request build matches the merge commit that will be tagged.

## CI Overview

- `Build` workflow runs on pull requests (and manual dispatch) using `.github/workflows/windows-build.yml`.
- The reusable workflow produces the Windows executable artifact that release packaging can reuse.
- Concurrency controls ensure only one build per ref runs at a time.

## Release Steps

1. **Kick off the release assistant** using the prompt in `.github/prompts/release-workflow-automated.md.prompt.md`. It walks through version bumping, branching, and tagging with approval gates.
2. **Review and merge** the version-bump pull request. CI must be green before merging.
3. **Tag the release** via the assistant after the PR merges. No additional build runs on `main`.
4. **Release workflow** (`.github/workflows/release.yml`) triggers on the `v*` tag:
   - Tries to reuse the PR build artifact.
   - Falls back to a fresh build if the artifact is missing.
   - Generates the checksum and drafts the GitHub Release.
5. **Write release notes** in the drafted release entry and publish when satisfied.

## Troubleshooting

- If the release workflow cannot find the PR artifact, inspect the logs to confirm it ran the fallback build.
- Failed releases can be retried after deleting the tag locally and remotely, then repeating the tagging step.
- Keep an eye on GitHub Actions minutes; the streamlined flow should normally produce at most one build per feature branch plus one packaging pass per release.
