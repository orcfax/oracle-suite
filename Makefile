.DEFAULT_GOAL := help

.PHONY: lint snapshot help upgrade

gofer-snapshot: 						## Build a snapshot of gofer.
	cd cmd/gofer;
	goreleaser build --snapshot --single-target --clean -f cmd/gofer/.goreleaser.yml

gofer-release:                          ## Build a gofer release..
	cd cmd/gofer;
	goreleaser release --skip=publish --clean -f cmd/gofer/.goreleaser.yml --skip=sign

gofer-release-sign:                     ## Build a gofer release..
	cd cmd/gofer;
	goreleaser release --skip=publish --clean -f cmd/gofer/.goreleaser.yml

lint:                                   ## Lint the source code (--ignore-errors to ignore errs)
	@echo ignore errors with "--ignore-errors"
	go fmt ./...
	staticcheck ./...
	golint ./...
	go vet ./...

pre-commit-checks:                      ## Run pre-commit-checks.
	pre-commit run --all-files

upgrade:                                ## Upgrade golang dependencies.
	go get -u ./...
	go mod tidy

verify-checksum-signature:              ## Verify checksum signing (useful for outputting GPG info).
	gpg --verify dist/checksums_sha256.txt.sig

help:                                   ## Print this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
