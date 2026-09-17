# Release process

`main` (GitHub) is the working branch. When ready to publish, create a
tag matching `vX.Y.Z` (e.g. `v0.2.0`) on `main` and push it to GitHub.

`.github/workflows/release.yml` fires on that tag push and runs
GoReleaser directly: build, GPG-sign, and publish the GitHub Release.
No mirroring or manual PR step - this repo is maintained on GitHub
only.

Required repository secrets (GitHub -> Settings -> Secrets and
variables -> Actions):

- `GPG_PRIVATE_KEY` / `PASSPHRASE` - signing key for release artifacts
- `GITHUB_TOKEN` - provided automatically by GitHub Actions
