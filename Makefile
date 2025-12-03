BINARY_NAME = woodpecker
REPO_NAME = github.com/operantai/$(BINARY_NAME)
GIT_COMMIT = $(shell git rev-list -1 HEAD)
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION = $(shell git describe --tags --always --dirty)
LD_FLAGS = "-X $(REPO_NAME)/cmd/woodpecker/cmd.GitCommit=$(GIT_COMMIT) -X $(REPO_NAME)/cmd/woodpecker/cmd.Version=$(GIT_COMMIT) -X $(REPO_NAME)/cmd/woodpecker/cmd.BuildDate=$(BUILD_DATE)"

all: fmt vet test build

.PHONY: build
build: ## Build binary
	@go build -o "bin/$(BINARY_NAME)" -ldflags $(LD_FLAGS) cmd/woodpecker/main.go

build-woodpecker-ai-verifier: ## Build woodpecker AI verifier container
	@docker build -f build/Dockerfile.woodpecker-ai-verifier .

build-woodpecker-ai-app: ## Build woodpecker AI app container
	@docker build -f build/Dockerfile.woodpecker-ai-app .

# ==================================================================================== #
# QUALITY
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	@go install mvdan.cc/gofumpt@latest
	@go fmt ./...
	@go mod tidy -v

.PHONY: lint-full
lint-full: generate lint-extras
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@golangci-lint run --timeout 1h

lint-extras: lint-check-focused-tests lint-check-wrap-in-logs lint-check-vulns

lint-check-focused-tests:
	@(\
         (\
             grep -r "FDescribe(" pkg || \
             grep -r "FIt(" pkg || \
             grep -r "FContext(" pkg || \
             grep -r "FEntry(" pkg || \
             grep -r "FWhen(" pkg \
         ) || \
         ( \
             grep -r "FDescribe(" internal || \
             grep -r "FIt(" internal || \
             grep -r "FContext(" internal || \
             grep -r "FEntry(" internal || \
             grep -r "FWhen(" internal \
         ) \
     ) && \
	  echo "Focused tests detected, remove them" && exit 1 || true

lint-check-wrap-in-logs:
	@(\
		grep -r log pkg | grep -v "fmt.Errorf" | grep "%w" \
		) && \
		echo "Do not use %w in logs for errors, use %s instead" && exit 1 || true

lint-check-vulns:
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@govulncheck -show verbose ./...

generate: clean
	@go generate -x ./...

clean:
	find . -name *.coverprofile | xargs rm -fr

.PHONY: fmt
fmt: ## Run go fmt
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@go vet ./...

.PHONY: lint
lint: ## Run linter
	@golangci-lint run --timeout 1h

.PHONY: test
test:
	@go test -v -race -buildvcs ./...

.PHONY: test/cover
test/cover:
	@go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	@go tool cover -html=/tmp/coverage.out

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
