# Agent instructions

## Project context

This repository is for a Terraform provider that manages a Linux CUPS installation. The user requested documentation and these instructions first so new chats can resume with project context.

At the initial documentation milestone there is no implementation, Go module, build tooling, or published provider. Verify the current repository before relying on this statement; update it as code is added.

Read these files at the start of substantive work:

1. `README.md` for project scope and status.
2. `docs/design.md` for proposed architecture, lifecycle behavior, and unresolved decisions.
3. `docs/roadmap.md` for milestones and completion criteria.

Inspect `git status` and relevant existing code before making changes. Preserve unrelated user changes. Do not publish release artifacts or modify a real CUPS installation unless the user requests those actions.

## Commit and push workflow

Always use Conventional Commits: `<type>[optional scope]: <description>`, with types such as `feat`, `fix`, `docs`, `refactor`, `test`, and `chore`. For example: `docs: document provider design and agent workflow`. Mark breaking changes with `!` or a `BREAKING CHANGE:` footer when appropriate.

After completing changes and the relevant verification, commit the task's changes and push the working branch to its configured remote. This is the default workflow and does not require a separate user request. Stage only files belonging to the task; leave unrelated changes untouched. If the branch has no upstream, set it when pushing to the intended remote. Never force-push unless explicitly requested. If committing or pushing fails, report the failure and whether the changes are committed locally.

## Working direction

The proposed first milestone manages queues on an existing CUPS server. Go and the Terraform Plugin Framework are the starting direction. Prefer IPP/CUPS administrative operations behind a dedicated client abstraction.

These are documented proposals, not claims that the user has finalized the design. Resolve routine implementation details using the task context. Ask for missing information when it materially affects scope or public identity, such as the module path, publishing namespace, or license. Do not invent these or treat illustrative examples as a frozen schema.

Keep Linux package installation, service management, print jobs, and direct CUPS configuration-file editing outside the first resource unless the user expands scope. Verify driver/model requirements before claiming that a queue can be created from its device URI alone.

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

There are no runnable checks at the documentation-only milestone. Do not report tests as passing until they actually exist and run.

Once Go code exists, run formatting and the relevant unit tests; add exact commands here when the tooling is established. Use meaningful tests for protocol mapping, errors, Terraform lifecycle, drift, import, and stable plans. Run acceptance tests only against an isolated, explicitly selected CUPS instance. Never default tests to the host's real printing service.

Check current primary documentation when choosing dependencies, framework APIs, or CUPS operations. Record compatibility based on verification, not guesswork. Avoid adding generated framework boilerplate or dependencies before an implementation task calls for them.

Keep the README, design decisions, roadmap, and examples consistent with the implementation. Clearly distinguish implemented behavior from planned behavior. When adding tooling, document how to build, test, and run a local provider and how to provision the test server.

In a handoff, describe what changed, what was verified, what remains uncertain, and the next concrete step. Do not claim the provider is usable or published before that is true.
