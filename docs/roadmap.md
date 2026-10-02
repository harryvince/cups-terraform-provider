# Implementation roadmap

The disposable CUPS test environment is implemented; provider implementation remains outstanding. Update the checkboxes when the work and its verification are complete, and keep design decisions in [design.md](design.md).

## 1. Resolve foundations and scaffold

- [ ] Confirm the first scope: queues on an existing CUPS server.
- [x] Add the MIT license selected by the repository owner.
- [ ] Select the module path and registry namespace with the repository owner.
- [ ] Verify a CUPS client approach, including administrative authentication and driver/model handling.
- [x] Add an isolated Docker Compose CUPS server, simulated printer, and lifecycle smoke check (see [testing.md](testing.md)).
- [ ] Select initial supported provider versions based on acceptance test coverage.
- [ ] Add a minimal Go provider using the Terraform Plugin Framework.
- [ ] Add build, formatting, unit-test, and local provider development instructions.

Completion: the provider builds, Terraform can load it locally, and the client approach has evidence for the operations the first resource needs.

## 2. Implement the client and provider configuration

- [ ] Implement endpoint, authentication, TLS, timeout, and cancellation behavior.
- [ ] Implement queue reads, add/modify, and deletion with precise error handling.
- [ ] Test response mapping, missing queues, transport failures, and authentication failures.
- [ ] Document supported connection methods and credential handling.

Completion: the client can safely exercise the required operations against the disposable CUPS server, and resource code can use it without embedding transport details.

## 3. Deliver the first queue resource

- [ ] Finalize and document the `cups_printer` schema.
- [ ] Implement create, read, update, delete, and import.
- [ ] Verify duplicate queue protection, drift detection, external deletion, and stable plans.
- [ ] Add runnable examples and provider/resource reference documentation.
- [ ] Document deletion effects, endpoint migration considerations, and known limitations.

Completion: acceptance tests prove the full lifecycle and import against the declared supported environment, including a plan with no changes after apply.

## 4. Prepare a release

- [ ] Add CI for the checks and supported platforms selected during implementation.
- [ ] Select versioning, packaging, signing, and release tooling.
- [ ] Publish installation instructions using the real provider source address.
- [ ] Review documentation and examples against the released schema.

Publishing is a separate action to perform when requested; this roadmap does not authorize a release.

## Later candidates

Printer data sources, printer classes, default-printer management, sharing and queue controls, additional print defaults, and wider CUPS compatibility. Select follow-up work based on user needs after the first resource is reliable.
