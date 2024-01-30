.DEFAULT_GOAL := help

.PHONY: lint snapshot help

gofer-snapshot: 						## Build a snapshot of gofer.
	cd cmd/gofer;
	goreleaser build --snapshot --single-target --clean -f cmd/gofer/.goreleaser.yml

gofer-release:                          ## Build a gofer release..
	cd cmd/gofer;
	goreleaser release --skip-publish --clean -f cmd/gofer/.goreleaser.yml

lint:                                   ## Lint the source code (--ignore-errors to ignore errs)
	go fmt ./...
	staticcheck ./...
	golint ./...
	go vet ./...

pre-commit-checks:                      ## Run pre-commit-checks.
	pre-commit run --all-files

help:                                   ## Print this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
