# Terraform Provider for CUPS

A project to manage a CUPS printing server through Terraform, starting with printer queues.

## Status

This repository contains planning documentation and a disposable Docker Compose CUPS test environment. There is no provider implementation, Go build system, release, or published Terraform Registry package yet. Resource names and configuration examples below are proposals, not a supported API.

The initial user request is to create a Terraform provider to manage a Linux CUPS installation, beginning with documentation and agent instructions that let later chats continue the work.

## Proposed scope

Start by managing printer queues on an **existing CUPS server**:

- Configure a connection to a local or remote CUPS server.
- Create, read, update, delete, and import individual printer queues.
- Manage a queue's device URI, description, and location.
- Detect configuration changes made outside Terraform.

Installing CUPS packages, managing the Linux service, and editing server configuration are deferred. This boundary is a proposed starting point; confirm it before expanding the implementation. Printer classes, default-printer selection, queue policies, and additional print options can follow once the first resource works reliably.

Print jobs and consumables are operational data rather than the initial Terraform-managed configuration. Creating or refreshing a resource should not print a test page.

## Proposed approach

Use Go and the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework), which HashiCorp recommends for provider development. Prefer a client that uses IPP and CUPS administrative operations over parsing command output or editing CUPS-managed files. CUPS documents these operations in its [IPP implementation reference](https://openprinting.github.io/cups/doc/spec-ipp.html).

The client library, authentication approach, supported CUPS versions, Go version, and Terraform version still need to be selected and verified. See [the design](docs/design.md) for behavior requirements and open decisions.

## Illustrative configuration

This example expresses the intended experience. It cannot run yet, and the schema may change. The registry source address is intentionally omitted until an owner and publishing namespace are chosen.

```hcl
provider "cups" {
  endpoint = "http://localhost:631"
}

resource "cups_printer" "office" {
  name        = "office"
  device_uri  = "ipp://printer.example.test/ipp/print"
  description = "Office printer"
  location    = "First floor"
}
```

`endpoint` refers to the CUPS server managed by Terraform. `device_uri` refers to the printer or backend used by that server. The example does not select a driver or print model; that part of the schema must be resolved before creating real queues.

## Continue development

Read [AGENTS.md](AGENTS.md), [the design](docs/design.md), and [the roadmap](docs/roadmap.md). The next milestone is a minimal provider skeleton and a verified CUPS client approach.

Start the isolated CUPS server and simulated printer, then check the fixture:

```sh
docker compose up --build --wait --wait-timeout 120
docker compose exec -T cups python3 /opt/testenv/smoke.py
```

The server is available at `http://127.0.0.1:8631`. See [testing instructions](docs/testing.md) for credentials, isolation, reset commands, and limitations. No Terraform provider tests exist yet.

As implementation lands, replace illustrative examples with runnable ones and document actual prerequisites, credentials, supported versions, import behavior, and development commands.

## References

- [Terraform Plugin Framework documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [HashiCorp provider scaffolding](https://github.com/hashicorp/terraform-provider-scaffolding-framework)
- [CUPS documentation](https://openprinting.github.io/cups/)
- [CUPS IPP operations](https://openprinting.github.io/cups/doc/spec-ipp.html)

## License

This project is licensed under the [MIT License](LICENSE).
