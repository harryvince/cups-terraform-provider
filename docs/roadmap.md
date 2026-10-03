# Implementation roadmap

The minimal provider POC and its disposable test environment are implemented. Update the checkboxes when the work and its verification are complete, and keep design decisions in [design.md](design.md).

## 1. Resolve foundations and scaffold

- [x] Implement the first scope: driverless queues on an existing CUPS server.
- [x] Add the MIT license selected by the repository owner.
- [x] Use the existing repository path as the Go module path; use a development override until Registry publication.
- [x] Select `harryvince/cups` with the repository owner.
- [x] Verify the CUPS client approach, including administrative authentication and driverless model handling.
- [x] Add an isolated Docker Compose CUPS server, simulated printer, and lifecycle smoke check (see [testing.md](testing.md)).
- [x] Record tested Go, Terraform, and CUPS versions based on the POC test loop.
- [x] Add a minimal Go provider using the Terraform Plugin Framework.
- [x] Add build, formatting, unit-test, and local provider development instructions.

Completion: the provider builds, Terraform can load it locally, and the client approach has evidence for the operations the first resource needs.

## 2. Implement the client and provider configuration

- [x] Implement explicit endpoint, Basic authentication, verified HTTPS, timeout, and cancellation behavior.
- [x] Implement queue reads, add/modify, and deletion with precise error handling.
- [x] Test response mapping, missing queues, transport failures, and authentication failures.
- [x] Document supported connection methods and credential handling.
- [ ] Verify HTTPS against a real CUPS fixture, beyond unit-level transport verification.

Completion: the client can safely exercise the required operations against the disposable CUPS server, and resource code can use it without embedding transport details.

## 3. Deliver the first queue resource

- [x] Document the provisional `cups_printer` POC schema.
- [x] Implement create, read, update, delete, and import.
- [x] Verify duplicate queue protection, drift detection, external deletion, and stable plans.
- [x] Add runnable examples and provider/resource reference documentation.
- [x] Document endpoint migration considerations and known limitations.
- [ ] Verify and document deletion effects on pending jobs before production use.

Completion: acceptance tests prove the full lifecycle and import against the declared supported environment, including a plan with no changes after apply.

## 4. Prepare a release

- [x] Add CI for unit/protocol checks, isolated acceptance tests, and six-platform packaging previews.
- [x] Configure Semantic Version tags, GoReleaser, RSA signing, and GitHub Releases.
- [x] Select the Registry identity and rename the repository to `terraform-provider-cups`.
- [ ] Register the provider and its public signing key with the Registry.
- [ ] Publish a selected version and verify clean installation from the Registry.
- [ ] Publish installation instructions using the real provider source address.
- [ ] Review documentation and examples against the released schema.

See [releasing.md](releasing.md) for the configured workflows and remaining registration steps. Publishing is a separate action to perform when requested; this roadmap does not authorize a release.

## Later candidates

Printer data sources, printer classes, default-printer management, sharing and queue controls, additional print defaults, and wider CUPS compatibility. Select follow-up work based on user needs after the first resource is reliable.
