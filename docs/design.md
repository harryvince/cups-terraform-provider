# Design direction

## Intent and decision status

The confirmed goal is a Terraform provider for managing a Linux CUPS installation. The user asked for documentation and `AGENTS.md` first so future chats can continue development.

The user selected the [MIT License](../LICENSE) for this repository.

The repository now implements a minimal provider POC. This document records its decisions and remaining limitations. The schema is provisional until a release; deferred capabilities are not implemented.

## Initial boundary

Treat CUPS as an existing service reached from the machine running Terraform. The first resource is `cups_printer`, managing driverless IPP queues and their metadata. Package installation, service management, and direct configuration-file management remain outside the POC.

A provider configuration targets one CUPS server. Terraform provider aliases can later support managing multiple servers. The server address and the queue's device URI are separate concepts.

## Architecture

Implemented architecture:

- Go with the Terraform Plugin Framework for schemas, diagnostics, and resource lifecycle.
- A separate CUPS client package that owns transport, authentication, IPP encoding, response decoding, and server errors.
- Resource code that translates Terraform configuration and state into client operations.
- Unit/protocol tests at the client and provider boundaries, plus opt-in tests using the real Terraform CLI against disposable CUPS.

Use `go-ipp` v1.7.0 as the IPP codec and a dedicated Go HTTP client as transport. This avoids the upstream convenience client's transport defaults and lets this provider enforce context deadlines, Basic authentication, TLS verification, redirect rejection, and bounded response sizes. Administrative lifecycle operations have been verified against the Compose fixture. The upstream decoder expects buffered input and can panic on malformed signed length fields; the client buffers responses and confines panic recovery to the decoder boundary. Unit tests cover these cases.

The Go module path is `github.com/harryvince/cups-terraform-provider`, matching the existing repository. The local provider address is `terraform.local/local/cups`; no Terraform Registry namespace has been selected or published. Framework v1.19.0 requires Go 1.25 or newer. Tested versions are Go 1.27.1 and Terraform 1.16.4 on Linux amd64; no broader compatibility claim is made.

Current layout:

```text
main.go                 Provider entry point
internal/provider/      Provider and resource schemas/lifecycle
internal/cups/          CUPS client and protocol mapping
examples/               Runnable local Terraform example
docs/                   Design, development, testing, provider/resource reference
```

## Provider configuration

Settings are `endpoint`, `username`, `password`, and `request_timeout`. The first three use `CUPS_ENDPOINT`, `CUPS_USERNAME`, and `CUPS_PASSWORD` only when omitted/null; explicit configuration wins, including explicit empty values which produce errors. There is no implicit local endpoint. Timeout defaults to 30 seconds and is bounded to 1–300. Unknown connection settings are errors before resource operations.

HTTPS verifies server certificates using system trust. HTTP is explicitly supported for the isolated fixture. There is no TLS bypass, TLS upgrade negotiation, Unix socket, client-certificate, or Kerberos implementation. The provider never silently changes the configured protocol.

The password input is sensitive. Credentials are not resource attributes, and diagnostics do not echo server-provided status messages or raw response bodies. Terraform's `sensitive` marking hides normal display but does not by itself prevent configured credentials from being stored in saved plans; prefer environment inputs. Device URIs containing credentials are rejected. See [provider settings](index.md) for the actual configuration contract.

## First resource

Implemented attributes:

| Attribute | Meaning | Behavior |
| --- | --- | --- |
| `name` | Queue name on the selected CUPS server | Required; replacement on change |
| `device_uri` | Credential-free IPP/IPPS device URI | Required; replacement on change |
| `description` | Human-readable description | Optional; defaults to empty; omission clears |
| `location` | Human-readable physical location | Optional; defaults to empty; omission clears |
| `id` | Terraform identity within the configured server | Computed; queue name |

Create uses `ppd-name=everywhere` and waits for CUPS to generate a driverless PPD. CUPS must be able to reach the device. Legacy PPD uploads, raw queues, and non-IPP backends are deferred. A device URI change replaces the queue to generate a fresh model, avoiding stale PPD reuse. Imported queues' existing models are preserved on metadata updates, but cannot be recovered or validated by this POC.

The Docker Compose fixture verified this flow against Debian's CUPS `2.4.2-3+deb12u9`. Generation may finish after the create response and may fill an empty description with the device model name. Creation therefore waits and reapplies managed metadata before reading state. A confirmed create followed by a verification failure returns partial state so Terraform can retain the queue for cleanup.

Administrative settings such as sharing, accepting jobs, and enabled/paused state are candidates for a later milestone. Avoid mixing transient device health with managed configuration.

## Lifecycle contract

- **Create:** detect an existing queue and return a diagnostic directing the user to import it. Serialize creates within one configured client. CUPS has no atomic create-only operation; another process can still race the check. This limitation remains documented.
- **Read:** obtain authoritative configuration. Remove the resource from state only when the server definitively reports that the queue does not exist. Network, authentication, and permission failures are errors, not absence.
- **Update:** update description/location in place, then read back server configuration. Name/device URI changes require replacement.
- **Delete:** remove only the managed queue. An already absent queue is successful deletion. Tests use empty queues; effects on pending jobs remain unverified and this POC should not manage active production queues.
- **Import:** adopt by queue name under the configured endpoint and populate state through the normal read path. Model/driver and operational settings are outside the schema.

Reads and plans do not mutate CUPS. Active jobs and device health are outside desired configuration. Unknown resource values defer validation until they become known; null optional metadata becomes an empty string. Omitted metadata clears rather than preserves server defaults.

Changing the provider endpoint changes the server addressed by existing state. Document that risk and require a deliberate migration workflow before claiming endpoint changes are safe.

Each lifecycle operation has a context deadline. Mutations are not automatically retried. PPD readiness polls retry only HTTP 404 within that deadline. A create transport failure explains that creation might have succeeded remotely and requires inspection/import before retrying. Diagnostics identify the operation, queue name, and relevant HTTP/IPP status without credentials.

## Verification

Unit and framework protocol tests cover mapping, validation, unknown/null handling, environment fallback, error classification, authentication failures, cancellation, PPD timeouts/partial creation, malformed responses, redirects, TLS verification, and missing queues.

Acceptance tests invoke the real Terraform CLI and verify create/read/update/delete/import, external drift, external deletion, replacement, metadata clearing, duplicate-name preservation, and stable plans. The Compose printer simulator removes any requirement for physical hardware.

Acceptance tests are opt-in, require explicit settings, and reject non-loopback endpoints and port 631. Use the [Docker Compose fixture](testing.md); its smoke check exercises the server directly, while the acceptance loop exercises the provider through Terraform.

## Decisions still open

1. Choose a Terraform Registry namespace, confirm its required repository rename, and complete registration. GitHub Actions and signed release packaging are configured; see [releasing.md](releasing.md).
2. Expand compatibility coverage across CUPS, Terraform, Go, and host platforms.
3. Add end-to-end TLS tests and decide whether additional authentication/transport methods are needed.
4. Decide whether to add enablement, accepting-jobs settings, model selection, or legacy PPD support.
5. Verify deletion effects on pending jobs before production use.
6. Decide whether stronger ownership/concurrency controls are needed beyond the documented CUPS API limitation.
7. Stabilize the schema and consider documentation generation before a release. CI is configured.

## Primary references

- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [CUPS IPP implementation](https://openprinting.github.io/cups/doc/spec-ipp.html)

Check current upstream documentation when selecting dependencies or implementing operations. Record tested behavior rather than assuming every CUPS version has the same capabilities.
