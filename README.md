# Oracle Suite

A set of tools that can be used to run Oracles. Original source from
[ChronicleLabs][chronicle-1] and updated to provide price-feed data according to
Orcfax's collector format.

[chronicle-1]: https://github.com/chronicleprotocol/oracle-suite

## Gofer

A tool to fetch and calculate reliable asset prices.

see: [Gofer CLI Readme](cmd/gofer/README.md)

### To build

[Goreleaser][gr-1] is required. Once installed, users can run a command such
as:

```sh
make gofer-snapshot
```

Release builds can be made using:

```sh
make gofer-release
```

Releases are currently managed by the GitHub [release][gh-1] action in this
repository.

#### Signing

It is possible to sign the checksums and binaries associated with a release but
this is still in testing. A GPG signature is required. Currently the process
is configured to select an `admin@orcfax.io` signing key.

To create a key follow the instructions on running:

```sh
gpg --full-generate-key
```

The [GitHub documentation][gh-2] provides useful information about generating a
GPG key.

[gr-1]: https://goreleaser.com/install/
[gh-1]: .github/workflows/release.yml
[gh-2]: https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key

## License

[The GNU Affero General Public License][affero-1]

[affero-1]: https://www.tldrlegal.com/license/gnu-affero-general-public-license-v3-agpl-3-0
