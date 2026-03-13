# CLI helpers.

# help
help:
    @just -l

# Build a snapshot of gofer
[working-directory: 'cmd/gofer']
gofer-snapshot:
    goreleaser build --snapshot --single-target --clean -f .goreleaser.yml

# Build a gofer release
[working-directory: 'cmd/gofer']
gofer-release:
    goreleaser release --skip=publish --clean -f .goreleaser.yml --skip=sign

# Build a gofer release
[working-directory: 'cmd/gofer']
gofer-release-sign:
    cd cmd/gofer;
    goreleaser release --skip=publish --clean -f .goreleaser.yml

# Lint the source code
lint:
    - goimports -w .
    - go fmt ./...
    - go vet ./...
    - staticcheck ./...

# Run the tests
test:
    - go test ./...

# Run pre-commit-checks
pre-commit-checks:
    pre-commit run --all-files

# Upgrade golang dependencies
upgrade:
    go get -u ./...
    go mod tidy

# Verify checksum signing (useful for outputting GPG info)
verify-checksum-signature:
    gpg --verify dist/checksums_sha256.txt.sig
