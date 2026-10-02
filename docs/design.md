# Design direction

## Intent and decision status

The confirmed goal is a Terraform provider for managing a Linux CUPS installation. The user asked for documentation and `AGENTS.md` first so future chats can continue development.

The rest of this document records proposed defaults and behavior requirements for implementation. It does not claim that these features exist or that the user has selected every design choice.

## Initial boundary

Treat CUPS as an existing service reached from the machine running Terraform. Begin with one printer queue resource, tentatively named `cups_printer`. Keep package installation, service management, and direct configuration-file management outside this first milestone.

A provider configuration targets one CUPS server. Terraform provider aliases can later support managing multiple servers. The server address and the queue's device URI are separate concepts.

## Architecture

Proposed implementation:

- Go with the Terraform Plugin Framework for schemas, diagnostics, and resource lifecycle.
- A separate CUPS client package that owns transport, authentication, IPP encoding, response decoding, and server errors.
- Resource code that translates Terraform configuration and state into client operations.
- Tests at the client and resource boundaries, plus opt-in acceptance tests against a disposable CUPS server.

Prefer IPP and CUPS administrative operations. Before selecting a Go library, prove it can read queues, add or modify them, delete them, authenticate administrative requests, and cancel requests through context deadlines. Evaluate TLS behavior and maintenance as well as API coverage. Record the result here. Use CLI commands only if a demonstrated API gap warrants them; document that decision and avoid shell interpolation.

An anticipated layout, to be created when implementation starts:

```text
main.go                 Provider entry point
internal/provider/      Provider and resource schemas/lifecycle
internal/cups/          CUPS client and protocol mapping
examples/               Runnable Terraform examples
docs/                   Design notes and eventual provider reference
```

## Provider configuration

Proposed settings include a server endpoint, explicit authentication, and a bounded request timeout. Determine whether Unix sockets and local authentication are needed before committing to the transport interface.

Use HTTPS with certificate verification for remote deployments. If a certificate-verification override is needed for development, make it explicit and disabled by default. Never silently downgrade transport security.

Define authentication and environment-variable precedence before writing configuration documentation. Mark credential fields sensitive and redact them from diagnostics and logs. Terraform's `sensitive` marking hides normal display; it does not by itself prevent values from being persisted in state or saved plans. Avoid persisting credentials as resource attributes. Device URIs may also embed credentials, so settle their handling before logging them or exposing them in examples.

## First resource

Candidate attributes:

| Attribute | Intended meaning | Proposed behavior |
| --- | --- | --- |
| `name` | Queue name on the selected CUPS server | Required; replacement on change |
| `device_uri` | Device/backend used by the queue | Required; mutable if the server supports it |
| `description` | Human-readable description | Optional; define clear/reset semantics |
| `location` | Human-readable physical location | Optional; define clear/reset semantics |
| `id` | Terraform identity within the configured server | Computed; proposed queue name |

Do not freeze the schema until driver/model handling is settled. Verify how to create a persistent driverless queue and what printer reachability is required. Decide whether the first version supports legacy PPDs, supports only a specific driverless flow, or needs another model selection mechanism. Do not assume a device URI alone is sufficient to create a working queue.

Administrative settings such as sharing, accepting jobs, and enabled/paused state are candidates for a later milestone. Avoid mixing transient device health with managed configuration.

## Lifecycle contract

- **Create:** detect an existing queue and return a diagnostic directing the user to import it. Do not silently adopt or overwrite it. Investigate whether the server offers enough atomicity to handle a concurrent creator safely; CUPS add/modify semantics must not be treated as an atomic create-only operation.
- **Read:** obtain authoritative configuration. Remove the resource from state only when the server definitively reports that the queue does not exist. Network, authentication, and permission failures are errors, not absence.
- **Update:** change supported configuration in place. Re-read after mutation and account for server normalization without producing perpetual diffs.
- **Delete:** remove only the managed queue. Treat an already absent queue as successful deletion. Document the server's effect on queued jobs before shipping this behavior.
- **Import:** adopt an existing queue by its queue name under the configured provider endpoint. Populate state through the normal read path. Document any attributes the API cannot recover.

Reads and plans must not mutate CUPS. Avoid managing volatile status such as active jobs or printer health as desired configuration. Explicitly distinguish Terraform null, unknown, empty, and configured values; define whether omission preserves a server default or clears an attribute.

Changing the provider endpoint changes the server addressed by existing state. Document that risk and require a deliberate migration workflow before claiming endpoint changes are safe.

Respect context cancellation and use bounded timeouts. Retry only failures known to be transient, and account for a mutation that may have succeeded before its response was lost. Return actionable diagnostics with the operation, queue name, and relevant server status, without credentials.

## Verification

Unit tests should cover attribute mapping, normalization, validation, unknown/null handling, and useful error classification. Client tests should exercise protocol success and error responses, authentication failures, timeouts, and missing queues.

Acceptance tests should use a disposable, explicitly configured CUPS instance and cover create/read/update/delete/import, external changes, external deletion, and a second plan with no changes. Test duplicate-name handling so existing queues are preserved. Queue setup should not require a physical printer unless a specific integration test explicitly declares that requirement; verify this against the selected driver/model strategy.

Keep acceptance tests opt-in. They must never automatically target the developer's system printing service or real printer queues. The isolated test setup and relevant environment variables need to be documented when implemented.

## Decisions still open

1. Confirm whether managing an existing server matches the desired first scope, or whether installation/service configuration is required.
2. Choose the repository/module identity, Terraform Registry namespace, and license.
3. Choose and verify the CUPS client, transport, and authentication mechanisms.
4. Define the supported CUPS, Terraform, and Go versions from actual test coverage.
5. Resolve driverless queue creation and any legacy driver/PPD support.
6. Finalize attribute names, defaults, clear/reset behavior, import ID, and secret handling.
7. Choose a reproducible disposable CUPS test environment.

## Primary references

- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [CUPS IPP implementation](https://openprinting.github.io/cups/doc/spec-ipp.html)

Check current upstream documentation when selecting dependencies or implementing operations. Record tested behavior rather than assuming every CUPS version has the same capabilities.
