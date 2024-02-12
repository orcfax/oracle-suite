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

[gr-1]: https://goreleaser.com/install/
[gh-1]: .github/workflows/release.yml

## License

[The GNU Affero General Public License][affero-1]

[affero-1]: https://www.tldrlegal.com/license/gnu-affero-general-public-license-v3-agpl-3-0
