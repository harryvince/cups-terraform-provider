# Local provider development

Prerequisites: [mise](https://mise.jdx.dev/getting-started.html), Git, and a running Docker Engine or Docker Desktop. The tested tool versions are in the [README](../README.md). No host CUPS installation is needed.

## Build and check

Tool versions and task implementations live in [mise.toml](../mise.toml), with the minimum-Go CI override in [mise.ci-min.toml](../mise.ci-min.toml). Committed lockfiles and `.mise/locks/` pin download checksums and the release tool’s npm dependencies. Install the configured mise version (`vars.mise_version`) using the official installation instructions, then from the repository root:

```sh
mise trust
mise run setup
mise tasks
mise run fmt
mise run build
mise run test
mise run vet
mise run check
```

The build produces `bin/terraform-provider-cups`. Unit tests use local HTTP test servers; acceptance tests skip unless explicitly enabled. The module path matches the existing repository: `github.com/harryvince/terraform-provider-cups`.

## Run Terraform against Compose

Start the fixture and build the provider:

```sh
mise run testenv:up
mise run build
```

Create an ignored, repository-local Terraform CLI configuration. These commands must be run from the repository root:

```sh
cat > .terraformrc.local <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/harryvince/cups" = "$PWD/bin"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc.local"
export CUPS_ENDPOINT=http://127.0.0.1:8631
export CUPS_USERNAME=cups-admin
export CUPS_PASSWORD=cups-test-password

terraform -chdir=examples/provider validate
terraform -chdir=examples/provider plan
terraform -chdir=examples/provider apply
terraform -chdir=examples/provider plan -detailed-exitcode
```

After apply, the last command should exit 0 with no changes. Exit 2 means changes are planned; exit 1 means an error. Terraform displays a development-override warning, which is expected. Skip `terraform init` for this example: the override loads the local binary directly and there is no provider package at this address to download. This workflow follows HashiCorp's [development override guidance](https://developer.hashicorp.com/terraform/plugin/debugging#terraform-cli-development-overrides).

Edit the resource's description or location and apply again to exercise updates. For import, use the queue name under the configured endpoint:

```sh
terraform -chdir=examples/provider state rm cups_printer.office
terraform -chdir=examples/provider import cups_printer.office poc-office
terraform -chdir=examples/provider plan -detailed-exitcode
```

`state rm` removes Terraform ownership while leaving this example's queue in CUPS. Import restores ownership. For an already existing queue, omit `state rm` and import it directly into a resource address not yet in state.

## Finish

```sh
terraform -chdir=examples/provider destroy
mise run testenv:down
unset TF_CLI_CONFIG_FILE CUPS_ENDPOINT CUPS_USERNAME CUPS_PASSWORD
```

Destroy deletes the queue; removing the stack resets all its container state. Rebuilding the provider is enough for subsequent CLI commands to load code changes. Keep state files and the local CLI configuration out of Git; the repository's `.gitignore` covers them.

For an automated lifecycle loop that builds its own binary and uses temporary Terraform configuration and state, see [acceptance testing](testing.md#provider-acceptance-loop).

## Migrating an earlier POC checkout

The repository is now `harryvince/terraform-provider-cups`, and the selected provider address is `registry.terraform.io/harryvince/cups`. Update existing checkout remotes and rebuild:

```sh
git remote set-url origin https://github.com/harryvince/terraform-provider-cups.git
mise run build
```

Update `required_providers` and the local `dev_overrides` key to the new address. If you have existing POC state under the old address, back up that state and migrate its provider reference from the configuration directory:

```sh
terraform state replace-provider terraform.local/local/cups registry.terraform.io/harryvince/cups
```

[This command](https://developer.hashicorp.com/terraform/cli/commands/state/replace-provider) changes Terraform's provider reference without modifying CUPS queues. No Registry version is published yet, so continue using the development override and skip `terraform init` until publication.

## Task shortcuts and tool updates

`mise run acceptance` starts the disposable fixture, runs its smoke and Terraform lifecycle checks, and removes it on success or failure. `mise run test:acc` targets a fixture you started separately and still requires explicit connection settings. `mise run release:snapshot` builds unsigned packages without publishing. See [releasing.md](releasing.md) for changelog and tag preparation.

`mise -E ci-min run check` uses the minimum supported Go version. CI uses the same tasks and mise configuration as local development. `mise run setup` installs only the project’s tools in locked mode, so unrelated globally configured tools need no project lock entries. The Makefile remains a compatibility wrapper for older commands; add new tasks in mise.

Change tool pins in the mise configuration, then refresh and commit their locks:

```sh
mise lock
mise -E ci-min lock
mise run setup
mise run check
```

Commit `mise.toml`, `mise.ci-min.toml`, both `.lock` files, and changed files under `.mise/locks/` together. Keep the Go module’s minimum Go requirement in sync with the minimum CI pin. Go library dependencies remain in `go.mod` and `go.sum`. Git, Docker Engine/CLI, and GPG are operating-system prerequisites; mise manages the Compose CLI rather than installing a Docker daemon. The CI bootstrap uses the runner’s Python to read mise’s own version from configuration.
