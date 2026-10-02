# Local provider development

Prerequisites: Go 1.25 or newer, Terraform CLI, and Docker with Compose. The tested tool versions are in the [README](../README.md). No host CUPS installation is needed.

## Build and check

From the repository root:

```sh
make fmt
make build
make test
make vet
```

The build produces `bin/terraform-provider-cups`. Unit tests use local HTTP test servers; acceptance tests skip unless explicitly enabled. The module path matches the existing repository: `github.com/harryvince/cups-terraform-provider`.

## Run Terraform against Compose

Start the fixture and build the provider:

```sh
make testenv-up
make build
```

Create an ignored, repository-local Terraform CLI configuration. These commands must be run from the repository root:

```sh
cat > .terraformrc.local <<EOF
provider_installation {
  dev_overrides {
    "terraform.local/local/cups" = "$PWD/bin"
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
make testenv-down
unset TF_CLI_CONFIG_FILE CUPS_ENDPOINT CUPS_USERNAME CUPS_PASSWORD
```

Destroy deletes the queue; removing the stack resets all its container state. Rebuilding the provider is enough for subsequent CLI commands to load code changes. Keep state files and the local CLI configuration out of Git; the repository's `.gitignore` covers them.

For an automated lifecycle loop that builds its own binary and uses temporary Terraform configuration and state, see [acceptance testing](testing.md#provider-acceptance-loop).
