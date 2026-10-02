# Agent instructions

## Project context

This repository is for a Terraform provider that manages a Linux CUPS installation. The user requested documentation and these instructions first so new chats can resume with project context.

The repository has a working Go provider POC, a Docker Compose CUPS test environment, unit/framework protocol tests, and opt-in acceptance tests using the real Terraform CLI. GitHub Actions CI and signed GoReleaser releases are configured. The provider is not published. Verify the current repository before relying on this statement; update it as code is added.

Read these files at the start of substantive work:

1. `README.md` for project scope and status.
2. `docs/design.md` for proposed architecture, lifecycle behavior, and unresolved decisions.
3. `docs/roadmap.md` for milestones and completion criteria.
4. `docs/testing.md` when working on the test environment or integration tests.
5. `docs/development.md`, `docs/index.md`, and `docs/resources/printer.md` for local usage and the implemented schema.
6. `docs/releasing.md` when working on CI, signing, packaging, or publishing.

Inspect `git status` and relevant existing code before making changes. Preserve unrelated user changes. Do not publish release artifacts or modify a real CUPS installation unless the user requests those actions.

## Commit and push workflow

Always use Conventional Commits: `<type>[optional scope]: <description>`, with types such as `feat`, `fix`, `docs`, `refactor`, `test`, and `chore`. For example: `docs: document provider design and agent workflow`. Mark breaking changes with `!` or a `BREAKING CHANGE:` footer when appropriate.

After completing changes and the relevant verification, commit the task's changes and push the working branch to its configured remote. This is the default workflow and does not require a separate user request. Stage only files belonging to the task; leave unrelated changes untouched. If the branch has no upstream, set it when pushing to the intended remote. Never force-push unless explicitly requested. If committing or pushing fails, report the failure and whether the changes are committed locally.

## Working direction

The POC manages driverless IPP queues on an existing CUPS server. It uses Go with the Terraform Plugin Framework and a separate IPP/CUPS client. The module path matches the repository, `github.com/harryvince/cups-terraform-provider`; the local-only provider address is `terraform.local/local/cups`.

The implemented POC schema remains provisional until release. Resolve routine implementation details using the task context. Ask for missing information when it materially affects scope or public identity, such as the publishing namespace. Do not invent a registry namespace. The user has selected the MIT license; retain the root `LICENSE` file.

Keep Linux package installation, service management, print jobs, and direct CUPS configuration-file editing outside the first resource unless the user expands scope. Creation uses the `everywhere` model and requires a device reachable from CUPS. Name/device URI changes replace a queue; description/location changes update in place and omitted metadata clears it. Enablement and accepting-jobs settings remain outside the POC.

## Implementation expectations

- Keep Terraform schema/state/lifecycle logic separate from CUPS protocol and transport code.
- Prefer explicit, typed attributes to a generic option map until their lifecycle semantics are understood.
- Handle null and unknown Terraform values deliberately. Normalize server responses without causing perpetual diffs.
- Make reads and plans free of CUPS mutations.
- Never interpret transport or authentication errors as a missing queue.
- Prevent ordinary creates from silently overwriting pre-existing queues; provide an import path and investigate server concurrency limitations.
- Make deletion of an already absent queue succeed. Confine deletion to the managed queue.
- Respect context cancellation, bounded timeouts, and the possibility of an ambiguous mutation result.
- Verify TLS by default. Redact credentials and credential-bearing URIs from logs and diagnostics.
- Mark credential inputs sensitive, while documenting the limits of Terraform's sensitive flag and state storage.
- Avoid unnecessary shell commands; if a CLI fallback is justified, use argument-based execution rather than interpolated shell strings.
- Do not print physical test pages as part of resource operations or automated verification.

## Verification and documentation

The test fixture has these runnable checks, from the repository root:

```sh
docker compose config --quiet
docker compose up --build --wait --wait-timeout 120
docker compose exec -T cups python3 /opt/testenv/smoke.py
docker compose down
```

The smoke check validates the isolated CUPS environment directly. Provider checks, from the repository root:

```sh
make fmt
make build
make test
make vet
CUPS_ACC_ENDPOINT=http://127.0.0.1:8631 \
  CUPS_ACC_USERNAME=cups-admin \
  CUPS_ACC_PASSWORD=cups-test-password \
  make test-acc
```

Start Compose before the acceptance command and shut it down afterward. Acceptance tests require explicit opt-in and settings, use temporary Terraform state and CLI configuration, and reject non-loopback endpoints and port 631. See `docs/testing.md` for credentials and endpoint details. Do not report tests as passing until they actually run.

Run formatting and relevant unit/protocol tests for Go changes; run the acceptance loop for lifecycle or client behavior changes. Use meaningful tests for protocol mapping, errors, Terraform lifecycle, drift, import, and stable plans. Run acceptance tests only against an isolated, explicitly selected CUPS instance. Never default tests to the host's real printing service.

Check current primary documentation when choosing dependencies, framework APIs, or CUPS operations. Record compatibility based on verification, not guesswork. Avoid adding generated framework boilerplate or dependencies before an implementation task calls for them.

Keep the README, design decisions, roadmap, and examples consistent with the implementation. Clearly distinguish implemented behavior from planned behavior. When adding tooling, document how to build, test, and run a local provider and how to provision the test server.

In a handoff, describe what changed, what was verified, what remains uncertain, and the next concrete step. Do not claim the provider is usable or published before that is true.

## Release automation

CI checks Go 1.25/1.27, runs isolated Terraform/CUPS acceptance tests, and builds unsigned six-platform snapshots. Version tags trigger CI followed by signed GitHub Releases. Actions and GoReleaser are pinned; verify upstream documentation when updating them. Dedicated RSA signing secrets are configured in GitHub; the public key is `release-signing-key.asc`. Never read or print the ignored `.release-signing/` private material in routine work or stage it. Preserve its local backup.

The current repository name is not eligible for public Terraform Registry registration. The owner must select the namespace and authorize the associated repository rename/source-address updates. Do not claim Registry publication until registration, a selected release, and a clean installation have been verified. Do not create release tags solely to test the workflow.
