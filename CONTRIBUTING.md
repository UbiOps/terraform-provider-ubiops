# Contributing to terraform-provider-ubiops

Thank you for your interest in contributing! This guide walks you through
setting up a local development environment and running the full suite of checks
that CI enforces.

## Prerequisites

Install the following tools before you begin:

- [Go](https://go.dev/dl/) 1.25+ — build and test the provider
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.13+ — run
  acceptance tests, format examples
- [golangci-lint](https://golangci-lint.run/welcome/install/) — Go linting (see
  `.golangci.yml`)
- [Biome](https://biomejs.dev/guides/getting-started/) — JSON linting and
  formatting
- [markdownlint-cli2](https://github.com/DavidAnson/markdownlint-cli2) —
  Markdown linting
- [ShellCheck](https://www.shellcheck.net/) — shell script linting
- [shfmt](https://github.com/mvdan/sh) — shell script formatting
- [yamlfmt](https://github.com/google/yamlfmt) — YAML formatting

On macOS with Homebrew:

```sh
brew install go terraform golangci-lint biome markdownlint-cli2 \
  shellcheck shfmt
go install github.com/google/yamlfmt/cmd/yamlfmt@latest
```

## Getting started

Clone the repository and download dependencies:

```sh
git clone https://github.com/ubiops/terraform-provider-ubiops.git
cd terraform-provider-ubiops
go mod download
```

Build and install the provider locally:

```sh
make build    # compile
make install  # compile + install into $GOBIN
```

Run the full default target (format, lint, install, generate docs):

```sh
make
```

## Using the local build with Terraform

After `make install`, configure Terraform to use your local build instead of the
registry version. Add a `dev_overrides` block to your `~/.terraformrc` file (on
Windows: `%APPDATA%\terraform.rc`):

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/ubiops/ubiops" = "<GOBIN>"
  }

  direct {}
}
```

Replace `<GOBIN>` with the output of `go env GOBIN`. If it is empty, the default
is `$(go env GOPATH)/bin` (typically `~/go/bin`).

With `dev_overrides` in place, Terraform skips `terraform init` for this
provider and uses the binary from your `GOBIN` directly. Other providers still
install normally via the `direct {}` block.

Create a test configuration to verify everything works:

```hcl
terraform {
  required_providers {
    ubiops = {
      source = "registry.terraform.io/ubiops/ubiops"
    }
  }
}

provider "ubiops" {}
```

Run `terraform plan` — you should see a warning that `dev_overrides` is active,
which confirms Terraform is using your local build.

> **Note:** Remove or comment out the `dev_overrides` block when you are done
> developing. It is only intended for local testing.

## Linting

CI runs all of the linters listed below. Make sure they pass before opening a
pull request.

### Go — golangci-lint

```sh
make lint
```

Configuration is in `.golangci.yml`. Notable rules:

- **depguard** blocks imports of the deprecated `terraform-plugin-sdk/v2` — use
  `terraform-plugin-framework` instead.
- **forcetypeassert** requires checked type assertions (`v, ok := x.(T)`).
- **godot** requires comments to end with a period.
- **gofmt** is the formatter.

### JSON — Biome

```sh
biome ci
```

Configuration is in `biome.json`. Uses tabs for indentation and double quotes
for JavaScript/JSON strings.

### Markdown — markdownlint-cli2

```sh
markdownlint-cli2
```

Configuration is in `.markdownlint-cli2.jsonc`. The `docs/` directory is
excluded because it is generated.

### Shell — ShellCheck + shfmt

```sh
shellcheck **/*.sh
shfmt --diff **/*.sh
```

ShellCheck catches common shell scripting issues. shfmt enforces consistent
formatting.

### Terraform — terraform fmt

```sh
terraform fmt -check -recursive examples/
```

All `.tf` files in `examples/` must be formatted with `terraform fmt`.

### YAML — yamlfmt

```sh
yamlfmt -lint
```

## Testing

### Unit tests

```sh
make test
```

No credentials needed. Resource CRUD error paths (`*_unit_test.go`) run
against a local `httptest.Server` mock via `resource.UnitTest` — see
`internal/provider/helpers_unit_test.go`. They check error propagation and
import parsing only; response-attribute mapping is covered by acceptance
tests below.

### Acceptance tests

Acceptance tests run against a real UbiOps environment. Set these environment
variables first:

```sh
export UBIOPS_API_TOKEN="Token <your-api-token>"
export UBIOPS_PROJECT="<your-test-project>"
export UBIOPS_ORGANIZATION="<your-organization>"
```

Then run:

```sh
make testacc
```

To run a single test:

```sh
TF_ACC=1 go test -v ./internal/provider -run TestAccDeploymentResource
```

## Documentation

Provider documentation is generated from schema descriptions and example files
using [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs). After
changing a schema or an example, regenerate the docs:

```sh
make generate
```

Generated files go into `docs/` and must be committed. CI verifies there is no
diff after running `make generate`.

### Example files

Each resource and data source has example files under `examples/`:

- `examples/resources/ubiops_<name>/resource.tf` — usage example
- `examples/resources/ubiops_<name>/import.sh` — import command
- `examples/data-sources/ubiops_<name>/data-source.tf` — data source example

## Code style

- Follow the patterns in existing resources.
- Comments end with a period.
- All comments should explain _why_, not _what_.
- Use `terraform-plugin-framework` types (`types.String`, `types.Int64`, etc.)
  in models, not Go primitives.

## Pull requests

- Keep feature branches small and merge quickly.
- One logical change per commit.
- Rebase on `main` before merging.
- Commits are reviewed individually — do not squash.

## License

By contributing, you agree that your contributions will be licensed under the
[MPL-2.0 License](LICENSE). No CLA is required.
