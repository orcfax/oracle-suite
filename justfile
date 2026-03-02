# CLI helpers.

# help
help:
	@just -l

# Build a snapshot of gofer.
gofer-snapshot:
	cd cmd/gofer;
	goreleaser build --snapshot --single-target --clean -f cmd/gofer/.goreleaser.yml

# Build a gofer release..
gofer-release:
	cd cmd/gofer;
	goreleaser release --skip=publish --clean -f cmd/gofer/.goreleaser.yml --skip=sign

# Build a gofer release..
gofer-release-sign:
	cd cmd/gofer;
	goreleaser release --skip=publish --clean -f cmd/gofer/.goreleaser.yml

# Lint the source code (--ignore-errors to ignore errs)
lint:
	@echo ignore errors with "--ignore-errors"
	go fmt ./...
	staticcheck ./...
	golint ./...
	go vet ./...

# Run pre-commit-checks.
pre-commit-checks:
	pre-commit run --all-files

# Upgrade golang dependencies.
upgrade:
	go get -u ./...
	go mod tidy

# Verify checksum signing (useful for outputting GPG info).
verify-checksum-signature:
	gpg --verify dist/checksums_sha256.txt.sig
