# FILE IS AUTOMATICALLY MANAGED BY github.com/vegaprotocol/terraform//github
export REPO_NAME := priceproxy



GO_FLAGS := -v
ifneq ($(RELEASE_VERSION),)
	GO_FLAGS += -ldflags "-X main.Version=$(RELEASE_VERSION)"
endif

.PHONY: build
build: ## install the binary in GOPATH/bin
	@env CGO_ENABLED=1 go build -v -o bin/${REPO_NAME} $(GO_FLAGS) ./cmd/${REPO_NAME}



.PHONY: build-simple
build-simple:
	@env go build -v -o bin/${REPO_NAME} ./cmd/${REPO_NAME}

.PHONY: all
default: deps build test lint

.PHONY: coverage
coverage:
	@go test -covermode=count -coverprofile="coverage.txt" ./...
	@go tool cover -func="coverage.txt"

.PHONY: docker
docker: ## Build docker image
	@docker build -t vegaprotocol/${REPO_NAME}:local .

.PHONY: coveragehtml
coveragehtml: coverage
	@go tool cover -html=coverage.txt -o coverage.html

.PHONY: deps
deps: ## Get the dependencies
	@go mod download

.PHONY: install
install:
	@go install $(GO_FLAGS) ./cmd/${REPO_NAME}

.PHONY: release-ubuntu-latest
release-ubuntu-latest:
	@mkdir -p build
	@env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -v -o build/${REPO_NAME}-linux-amd64 $(GO_FLAGS) ./cmd/${REPO_NAME}
	@cd build && zip ${REPO_NAME}-linux-amd64.zip ${REPO_NAME}-linux-amd64

.PHONY: release-macos-latest
release-macos-latest:
	@mkdir -p build
	@env GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -v -o build/${REPO_NAME}-darwin-amd64 $(GO_FLAGS) ./cmd/${REPO_NAME}
	@cd build && zip ${REPO_NAME}-darwin-amd64.zip ${REPO_NAME}-darwin-amd64

.PHONY: release-windows-latest
release-windows-latest:
	@env GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -v -o build/${REPO_NAME}-amd64.exe $(GO_FLAGS) ./cmd/${REPO_NAME}
	@cd build && 7z a -tzip ${REPO_NAME}-windows-amd64.zip ${REPO_NAME}-amd64.exe

.PHONY: lint
lint:
	@golangci-lint run -v --config .golangci.toml

.PHONY: mocks
mocks: ## Make mocks
	@find -name '*_mock.go' -print0 | xargs -0r rm
	@go generate ./...

.PHONY: race
race: ## Run data race detector
	@env CGO_ENABLED=1 go test -race ./...

.PHONY: retest
retest: ## Force re-run of all tests
	@go test -count=1 ./...

.PHONY: test
test: ## Run tests
	@go test ./...

.PHONY: clean
clean: ## Remove previous build
	@rm -f ./bin/${REPO_NAME}

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
