# Gofer CLI Readme

> As in a [tool](https://en.wikipedia.org/wiki/Gofer) that specializes in the
delivery of special items.

Gofer is a tool that provides asset prices taken from various sources.

The original documentation from Chronicle Labs may still provide a useful
reference for those looking at this tool. Additional updates are expected to
continue on the main README for this repository.

The original documentation can be found [here][chronicle-readme-1]

[chronicle-readme-1]: https://github.com/orcfax/oracle-suite/blob/master/cmd/gofer/README.md

## Installation

To install Gofer you'll first need [Go][go-1] installed on your machine. Then
you can use standard Go command:

[go-1]: https://go.dev/doc/install

```sh
go install github.com/orcfax/oracle-suite/cmd/gofer@latest
```

Alternatively, you can build Gofer using `Makefile` directly from the
repository. This approach is recommended if you wish to work on Gofer source.

```bash
git clone https://github.com/orcfax/oracle-suite.git
cd oracle-suite
make gofer-snapshot
```

## Usage

Once downloaded, you can begin to explorer the utilities functionality by
checking out the command-line options.

```text
Usage:
  gofer [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  config      Render the config file
  data        Return data points for given models
  help        Help about any command
  models      List all supported models
  run         Run the main service
  version     print the version details

Flags:
  -c, --config strings                                 config file
  -h, --help                                           help for gofer
  -f, --log.format text|json                           log format (default text)
  -v, --log.verbosity panic|error|warning|info|debug   verbosity level (default info)
      --version                                        version for gofer

Use "gofer [command] --help" for more information about a command.
```

## License

[The GNU Affero General Public License][affero-1]

[affero-1]: https://www.tldrlegal.com/license/gnu-affero-general-public-license-v3-agpl-3-0
