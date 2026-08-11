# Release process (internal, GitLab-side only)

This file documents this repo's release mechanics. It is intentionally
excluded from the GitHub mirror (see `scripts/git_push_github.sh`'s
exclusion list) - it references internal GitLab CI/CD secrets and
pipeline internals that have no place in the public repo's README.

`main` (GitLab) is the working branch. When ready to publish, create a
tag matching `vX.Y.Z` (e.g. `v0.2.0`). GitLab CI (`.gitlab-ci.yml` ->
`scripts/git_push_github.sh`) then:

1. Mirrors the repo content (minus `.gitlab-ci.yml`/`scripts/`/this
   file) into a `release/vX.Y.Z` branch on
   `github.com/${GITHUB_REPOSITORY}`.
2. Opens a pull request from that branch into `$GITHUB_DEFAULT_BRANCH`
   automatically via the GitHub API - no manual "create a PR" step.

Merging the PR does **not** publish a release by itself -
`.github/workflows/release.yml` (GoReleaser) only fires on a tag push
on the GitHub side. After merging, create the same `vX.Y.Z` tag on
GitHub's default branch to trigger the actual build, GPG-signing, and
GitHub Release publish.

Required CI/CD variables (GitLab -> Settings -> CI/CD -> Variables):

- `SSH_PRIVATE_KEY_BASE64` - base64-encoded deploy key with write
  access to the GitHub repo
- `GITHUB_TOKEN` - fine-grained PAT with **Contents: Read-only** +
  **Pull requests: Read and write**. Contents read is required or PR
  creation fails with "not all refs are readable"
- `GITHUB_REPOSITORY` - `UbiOps/terraform-provider-ubiops`
- `GITHUB_DEFAULT_BRANCH` - `main`
- `GITHUB_USER_EMAIL` / `GITHUB_USER_NAME` - commit author for the
  mirrored commit

After a failed/retried release attempt, delete the stale
`release/vX.Y.Z` branch on GitHub before re-tagging - the script does a
plain (non-force) branch push each run.
