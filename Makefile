FILES = $(shell find . -type f -name '*.go')

default: help

help:                   ## Display this help message
	@echo "Please use \`make <target>\` where <target> is one of:"
	@grep '^[a-zA-Z]' $(MAKEFILE_LIST) | \
		awk -F ':.*?## ' 'NF==2 {printf "  %-26s%s\n", $$1, $$2}'

init:                   ## Install development tools
	cd tools && go generate -x -tags=tools

build:                  ## Build binaries
	go build -race -o bin/everest ./cmd/everest

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

k8s: ## Create a local minikube cluster
	minikube start --nodes=3 --cpus=4 --memory=4g --apiserver-names host.docker.internal
	minikube addons disable storage-provisioner
	kubectl delete storageclass standard
	kubectl apply -f ./dev/kubevirt-hostpath-provisioner.yaml

release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./dist/everestctl-linux-amd64 ./cmd/everest
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -o ./dist/everestctl-linux-arm64 ./cmd/everest
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -o ./dist/everestctl-darwin-amd64 ./cmd/everest
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -v -o ./dist/everestctl-darwin-arm64 ./cmd/everest
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -o ./dist/everestctl.exe ./cmd/everest
