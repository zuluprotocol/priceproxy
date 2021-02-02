# Makefile

.PHONY: default
default: install test

.PHONY: coverage
coverage:
	@go test -covermode=count -coverprofile="coverage.txt" ./...
	@go tool cover -func="coverage.txt"

.PHONY: coveragehtml
coveragehtml: coverage
	@go tool cover -html=coverage.txt -o coverage.html

.PHONY: deps
deps: ## Get the dependencies
	@go mod download

.PHONY: gosec
gosec:
	gosec ./...

.PHONY: install
install: ## install the binary in GOPATH/bin
	@v=vv \
	vh=vhvh ; \
	echo "Version: $$v ($$vh)" ; \
	go install -v -ldflags "-X main.Version=$$v -X main.VersionHash=$$vh" ./cmd/priceproxy

.PHONY: lint
lint:
	@go install golang.org/x/lint/golint
	@outputfile="$$(mktemp)" ; \
	go list ./... | xargs -r golint 2>&1 | \
		sed -e "s#^$$GOPATH/src/##" | tee "$$outputfile" ; \
	lines="$$(wc -l <"$$outputfile")" ; \
	rm -f "$$outputfile" ; \
	exit "$$lines"

.PHONY: race
race: ## Run data race detector
	@env CGO_ENABLED=1 go test -race ./...

.PHONY: retest
retest: ## Force re-run of all tests
	@go test -count=1 ./...

.PHONY: staticcheck
staticcheck: ## Run statick analysis checks
	@staticcheck ./...

.PHONY: test
test: ## Run tests
	@go test ./...

.PHONY: vet
vet: deps
	@go vet ./...

.PHONY: clean
clean: ## Remove previous build
	@rm -f ./cmd/priceproxy/priceproxy
