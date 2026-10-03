# Publishing the provider

GitHub Actions runs checks on pull requests and `main`. Pushing a version tag runs those checks again and uses GoReleaser to build, sign, and publish a GitHub Release. [GitHub Release v0.1.0](https://github.com/harryvince/terraform-provider-cups/releases/tag/v0.1.0) is published. Terraform Registry registration remains a separate setup step.

The first release passed the GitHub Actions checks. Downloaded assets were independently checked for the expected signing fingerprint, all SHA256 sums, protocol-6 manifest, and six ZIP layouts. The released Linux amd64 binary was installed through a temporary filesystem mirror and loaded by Terraform to obtain the `cups_printer` schema. This verifies the downloadable binary, not Registry ingestion or installation.

## Checks and packages

[CI](../.github/workflows/ci.yml) runs formatting, dependency verification, race-enabled unit/protocol tests, vet, builds, the disposable CUPS smoke check, and the real Terraform acceptance loop. It checks Go 1.25 and 1.27 and previews all release archives without signing secrets. The acceptance fixture uses Go 1.27.1 and Terraform 1.16.4.

[GoReleaser](../.goreleaser.yaml) builds Linux, macOS, and Windows packages for amd64 and arm64 with CGO disabled. Each ZIP contains a versioned provider binary and the MIT license. Release assets include SHA256 checksums, their detached GPG signature, and a protocol-6 Registry manifest included in the checksums. Cross-compilation verifies packaging; runtime lifecycle compatibility is currently verified only on Linux amd64.

Actions are pinned to commit hashes. Tool versions, including GoReleaser and mise itself, are maintained in `mise.toml`; `mise.ci-min.toml` supplies the minimum-Go override. Update pins deliberately and run the checks when changing them. The release job alone has repository write permission; pull-request checks do not need secrets.

## Signing key

The repository has `GPG_PRIVATE_KEY` and `PASSPHRASE` Actions secrets configured for a dedicated RSA-4096 release key. Its public key is committed as [release-signing-key.asc](../release-signing-key.asc), with fingerprint:

```text
2E88C555E207F2C146D513919FA1EE4765021189
```

The key expires on 2 October 2028. A protected local backup is in the ignored `.release-signing/` directory: `private-key.asc`, `passphrase.txt`, and `fingerprint.txt`. Back up the private key and passphrase outside this checkout in secure storage. GitHub cannot retrieve a stored secret for recovery. Never commit that directory or include its contents in logs or release assets.

To use another key, replace both Actions secrets and the committed public key, update this fingerprint, and register the replacement public key in the Registry before using it for Registry releases. See HashiCorp's [signing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing).

## Publish a GitHub Release

Release preparation uses [commit-and-tag-version](https://github.com/absolute-version/commit-and-tag-version), the maintained fork of the deprecated standard-version. Mise pins both Node and the release CLI. Conventional Commits determine the suggested version; the current version comes from the latest Git tag, so this Go project needs no package.json or duplicated version file. Configuration is in [.versionrc.json](../.versionrc.json).

From an up-to-date, clean `main`, install the pinned tools and preview changes:

```sh
mise run setup
mise run release:preview
```

When a release is requested, prepare its changelog, Conventional Commit, and local annotated `v` tag:

```sh
mise run release:prepare
# Or choose the version explicitly:
# mise run release:prepare -- --release-as 0.2.0
```

The task rejects a dirty working tree or a branch other than `main`. It does not push or publish. Review `CHANGELOG.md`, the release commit, and the generated tag, then push that specific tag with the branch. For example, if preparation selected `v0.1.1`:

```sh
git push --atomic origin main v0.1.1
```

This starts the publishing workflow. Preview supports the same CLI arguments, including `--release-as` and `--prerelease rc`. No new release is created when setting up or testing this tooling. The existing `v0.1.0` history is recorded in [CHANGELOG.md](../CHANGELOG.md).

[Release](../.github/workflows/release.yml) reruns CI before publishing. A tag such as `v0.1.0-rc.1` produces a GitHub prerelease. The workflow refuses to replace an existing release; use a new version for corrected published artifacts. Inspect the Actions run and verify the release's archives, manifest, checksum file, and `.sig` asset. Tags and GitHub Releases alone do not make the provider discoverable in the Terraform Registry.

The provider uses the selected address `registry.terraform.io/harryvince/cups`, which requires a development override until Registry publication. GitHub downloads can be used with the development override described in [development.md](development.md), by extracting the matching archive into the configured binary directory. This is not a Registry installation workflow.

## Connect the public Terraform Registry

HashiCorp requires a public GitHub repository named `terraform-provider-{NAME}`. The owner selected `harryvince/cups`, and the repository is now [harryvince/terraform-provider-cups](https://github.com/harryvince/terraform-provider-cups). The Go module, provider server address, documentation, and examples match this identity. GitHub Release `v0.1.0` is available; Registry registration and installation remain unverified.

To finish registration:

1. Sign in to the [Terraform Registry](https://registry.terraform.io/) using the GitHub account that owns the public repository.
2. Add the committed armored public signing key in the Registry's user settings under signing keys.
3. Select **Publish → Provider**, choose the repository, and complete registration. The Registry connects release notifications to package ingestion.
4. Check that the Registry ingests the existing `v0.1.0` release, then verify `terraform init` from a clean configuration using `harryvince/cups` with version `0.1.0` and no development override. Future versions use the tag workflow.

Follow the current [provider publishing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing). Public Registry registration and clean installation must be verified before documenting the provider as Registry-published.

## Local packaging verification

Install the mise tools and run:

```sh
mise run lint
mise run release:snapshot
```

Snapshots land in ignored `dist/` and do not create a tag or GitHub Release. The snapshot version includes the commit identifier. To verify a signed release, import the committed public key, verify the detached signature, then check the SHA256 sums in a directory containing all named assets:

```sh
gpg --import release-signing-key.asc
gpg --verify terraform-provider-cups_0.1.0_SHA256SUMS.sig terraform-provider-cups_0.1.0_SHA256SUMS
sha256sum --check terraform-provider-cups_0.1.0_SHA256SUMS
```

Use the chosen release version in the filenames. Verify the public key fingerprint independently before trusting a signature.
