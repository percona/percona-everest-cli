FILES = $(shell find . -type f -name '*.go')

default: help

help:                   ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

init:                   ## Install development tools
	cd tools && go generate -x -tags=tools

build:                  ## Build binaries
	go build -race -o bin/everest-cli ./cmd/everest-cli

gen:                    ## Generate code
	go generate ./...
	make format

format:                 ## Format source code
	bin/gofumpt -l -w $(FILES)
	bin/goimports -local github.com/percona/percona-everest-cli -l -w $(FILES)
	bin/gci write --section Standard --section Default --section "Prefix(github.com/percona/percona-everest-cli)" $(FILES)

check:                  ## Run checks/linters for the whole project
	bin/go-consistent -pedantic ./...
	LOG_LEVEL=error bin/golangci-lint run

test:                   ## Run tests
	go test -race -timeout=30s ./...

test-cover:             ## Run tests and collect per-package coverage information
	go test -race -timeout=30s -count=1 -coverprofile=cover.out -covermode=atomic ./...

test-crosscover:        ## Run tests and collect cross-package coverage information
	go test -race -timeout=30s -count=1 -coverprofile=crosscover.out -covermode=atomic -p=1 -coverpkg=./... ./...
