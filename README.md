# Oracle Data Collector Suite

A set of tools that can be used to run data collectors for oracle networks. Original source from
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

### Configuring gofer

The oracle suite comes packaged with a number of api sources which can be
leveraged. The desired combination of these sources, the data requeested from
each, and the minimum number of responses are set within the
[config-gofer.hcl](config/config-gofer.hcl) file.

Additional api sources must be added along with parameters for how responses
will be passed into json.

eg
```
  origin "coinbase" {
    type = "tick_generic_jq"
    url  = "https://api.pro.coinbase.com/products/$${ucbase}-$${ucquote}/ticker"
    jq   = "{price: .price, time: .time, volume: .volume}"
  }
```
Then sources can be grouped into a `data_model` and the `min_values` for
publication set; the min establishes how many sources must be included in a
publication.

The Orcfax system requires that a minimum of 3 sources participate in each
publication in order to triangulate the data being reported.

eg
```
  data_model "ADA/USD" {
    median {
      min_values = 3
      origin "bitstamp" { query = "ADA/USD" }
      origin "coinbase" { query = "ADA/USD" }
      origin "kraken" { query = "ADA/USD" }
      origin "kucoin_prices_simple" { query = "ADA/USD" }
      origin "bitfinex_simple" { query = "ADA/USD" }
      origin "hitbtc" { query = "ADA/USD" }
    }
  }
```
It is advisable to group more than the minimum necessary sources within the data
model in order to provide contingencies for when api sources fail.

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
