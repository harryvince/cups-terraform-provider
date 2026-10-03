# Publishing the provider

GitHub Actions runs checks on pull requests and `main`. Pushing a version tag runs those checks again and uses GoReleaser to build, sign, and publish a GitHub Release. No version has been published yet. Terraform Registry registration remains a separate setup step.

## Checks and packages

[CI](../.github/workflows/ci.yml) runs formatting, dependency verification, race-enabled unit/protocol tests, vet, builds, the disposable CUPS smoke check, and the real Terraform acceptance loop. It checks Go 1.25 and 1.27 and previews all release archives without signing secrets. The acceptance fixture uses Go 1.27.1 and Terraform 1.16.4.

[GoReleaser](../.goreleaser.yaml) builds Linux, macOS, and Windows packages for amd64 and arm64 with CGO disabled. Each ZIP contains a versioned provider binary and the MIT license. Release assets include SHA256 checksums, their detached GPG signature, and a protocol-6 Registry manifest included in the checksums. Cross-compilation verifies packaging; runtime lifecycle compatibility is currently verified only on Linux amd64.

Actions are pinned to commit hashes, and GoReleaser is pinned to v2.18.2. Update pins deliberately and run the checks when changing them. The release job alone has repository write permission; pull-request checks do not need secrets.

## Signing key

The repository has `GPG_PRIVATE_KEY` and `PASSPHRASE` Actions secrets configured for a dedicated RSA-4096 release key. Its public key is committed as [release-signing-key.asc](../release-signing-key.asc), with fingerprint:

```text
2E88C555E207F2C146D513919FA1EE4765021189
```

The key expires on 2 October 2028. A protected local backup is in the ignored `.release-signing/` directory: `private-key.asc`, `passphrase.txt`, and `fingerprint.txt`. Back up the private key and passphrase outside this checkout in secure storage. GitHub cannot retrieve a stored secret for recovery. Never commit that directory or include its contents in logs or release assets.

To use another key, replace both Actions secrets and the committed public key, update this fingerprint, and register the replacement public key in the Registry before using it for Registry releases. See HashiCorp's [signing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing).

## Publish a GitHub Release

Review the schema and documentation, choose a Semantic Version, and ensure CI passes on the intended commit. From an up-to-date `main`, publish a tag; these commands illustrate a first version and must be run only when that version is selected:

```sh
git tag v0.1.0
git push origin v0.1.0
```

[Release](../.github/workflows/release.yml) reruns CI before publishing. A tag such as `v0.1.0-rc.1` produces a GitHub prerelease. The workflow refuses to replace an existing release; use a new version for corrected published artifacts. Inspect the Actions run and verify the release's archives, manifest, checksum file, and `.sig` asset. Tags and GitHub Releases alone do not make the provider discoverable in the Terraform Registry.

The provider uses the selected address `registry.terraform.io/harryvince/cups`, which requires a development override until Registry publication. GitHub downloads can be used with the development override described in [development.md](development.md), by extracting the matching archive into the configured binary directory. This is not a Registry installation workflow.

## Connect the public Terraform Registry

HashiCorp requires a public GitHub repository named `terraform-provider-{NAME}`. The owner selected `harryvince/cups`, and the repository is now [harryvince/terraform-provider-cups](https://github.com/harryvince/terraform-provider-cups). The Go module, provider server address, documentation, and examples match this identity. Registration and publication remain pending.

To finish registration:

1. Sign in to the [Terraform Registry](https://registry.terraform.io/) using the GitHub account that owns the public repository.
2. Add the committed armored public signing key in the Registry's user settings under signing keys.
3. Select **Publish → Provider**, choose the repository, and complete registration. The Registry connects release notifications to package ingestion.
4. Publish the selected version through the tag workflow, check that the Registry ingests its signed assets, and verify `terraform init` from a clean configuration using the real source address and version constraint.

Follow the current [provider publishing requirements](https://developer.hashicorp.com/terraform/registry/providers/publishing). Public Registry registration and clean installation must be verified before documenting the provider as Registry-published.

## Local packaging verification

Install GoReleaser v2.18.2 and run:

```sh
goreleaser check
goreleaser release --snapshot --clean --skip=publish,sign
```

Snapshots land in ignored `dist/` and do not create a tag or GitHub Release. The snapshot version includes the commit identifier. To verify a signed release, import the committed public key, verify the detached signature, then check the SHA256 sums in a directory containing all named assets:

```sh
gpg --import release-signing-key.asc
gpg --verify terraform-provider-cups_0.1.0_SHA256SUMS.sig terraform-provider-cups_0.1.0_SHA256SUMS
sha256sum --check terraform-provider-cups_0.1.0_SHA256SUMS
```

Use the chosen release version in the filenames. Verify the public key fingerprint independently before trusting a signature.
