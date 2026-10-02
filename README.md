# Terraform Provider for CUPS

A project to manage a CUPS printing server through Terraform, starting with printer queues.

## Status

This repository contains a working provider POC, a disposable Docker Compose CUPS environment, and tests using the real Terraform CLI. The provider is for local development; there is no release or Terraform Registry package yet, and its schema may change before release.

GitHub Actions checks and signed, multi-platform release packaging are configured. See [publishing instructions](docs/releasing.md) for version tags, signing-key recovery, and the remaining Terraform Registry setup. No release tag has been pushed.

The initial user request is to create a Terraform provider to manage a Linux CUPS installation, beginning with documentation and agent instructions that let later chats continue the work.

## POC scope

Manage driverless IPP printer queues on an **existing CUPS server**:

- Configure a connection to a local or remote CUPS server.
- Create, read, update, delete, and import individual printer queues.
- Manage a queue's device URI, description, and location.
- Detect configuration changes made outside Terraform.

Changing a queue's name or device URI replaces it. Description and location update in place; omitting either clears it. Newly created queues remain paused and reject jobs under the tested CUPS defaults. Queue enablement and accepting-jobs settings are not managed yet.

Installing CUPS packages, managing the Linux service, and editing server configuration are deferred. Printer classes, default-printer selection, queue policies, and additional print options can follow.

Print jobs and consumables are operational data rather than the initial Terraform-managed configuration. Creating or refreshing a resource should not print a test page.

## Implementation

The provider uses Go and the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework). Its separate CUPS client uses [go-ipp](https://github.com/phin1x/go-ipp) for the protocol codec and Go's HTTP client for transport, cancellation, password authentication, and certificate verification. It uses [CUPS IPP administrative operations](https://openprinting.github.io/cups/doc/spec-ipp.html) directly.

Build with Go 1.25 or newer. The POC has been tested on Linux amd64 with Go 1.27.1, Terraform 1.16.4, and Debian's CUPS `2.4.2-3+deb12u9`. Broader compatibility remains unverified. See [the design](docs/design.md) for decisions and limitations.

## Configuration

This configuration targets the Compose fixture after following [local development setup](docs/development.md). `terraform.local/local/cups` is a local development address, not a published package.

```hcl
terraform {
  required_providers {
    cups = {
      source = "terraform.local/local/cups"
    }
  }
}

# Set CUPS_ENDPOINT, CUPS_USERNAME and CUPS_PASSWORD in your environment.
provider "cups" {}

resource "cups_printer" "office" {
  name        = "poc-office"
  device_uri  = "ipp://printer.local:8000/ipp/print"
  description = "Office printer"
  location    = "First floor"
}
```

`endpoint` refers to the CUPS server managed by Terraform. `device_uri` refers to the printer reached by that server. Creation uses the `everywhere` driverless model and waits for its generated PPD. The CUPS server must be able to reach the device. Device URIs containing credentials are rejected.

See [provider settings](docs/index.md), [the printer resource](docs/resources/printer.md), and [the runnable example](examples/provider/main.tf).

## Continue development

Read [AGENTS.md](AGENTS.md), [the design](docs/design.md), and [the roadmap](docs/roadmap.md). Build and run unit checks with `make build`, `make test`, and `make vet`.

Start the isolated CUPS server and simulated printer, then check the fixture:

```sh
docker compose up --build --wait --wait-timeout 120
docker compose exec -T cups python3 /opt/testenv/smoke.py
```

The server is available at `http://127.0.0.1:8631`. See [testing instructions](docs/testing.md) for the provider acceptance loop, credentials, isolation, reset commands, and limitations. Acceptance tests require explicit opt-in and connection settings.

The POC protects ordinary creates from overwriting existing queues, but CUPS does not provide an atomic create-only operation. Concurrent external administrators can race the existence check. Use an import workflow for existing queues and avoid concurrent management of the same queue.

## References

- [Terraform Plugin Framework documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [HashiCorp provider scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [CUPS documentation](https://openprinting.github.io/cups/)
- [CUPS IPP operations](https://openprinting.github.io/cups/doc/spec-ipp.html)

## License

This project is licensed under the [MIT License](LICENSE).
