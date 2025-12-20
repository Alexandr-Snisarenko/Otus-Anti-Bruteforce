# -------- Project meta --------
MODULE := github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce
APP_NAME      := anti-bruteforce
CLI_NAME      := abfctl

BIN_DIR       := ./bin
DOCKER_IMG= anti-bruteforce:develop
CONTAINER_NAME := abf-app-develop

GIT_HASH := $(shell git log --format="%h" -n 1)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := \
  -X $(MODULE)/internal/version.Release=develop \
  -X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE) \
  -X $(MODULE)/internal/version.GitHash=$(GIT_HASH)

# -------- Proto --------
PROTO_DIR = api/proto/anti_bruteforce/v1
generate:
	protoc \
		--proto_path=api/proto/anti_bruteforce/v1 \
		--go_out=$(PROTO_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/anti_bruteforce.proto


# -------- Build targets --------
build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/anti-bruteforce

build-cli:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(CLI_NAME) ./cmd/abfctl

run: build
	$(BIN_DIR)/$(APP_NAME) -config ./configs/config.yml

build-img:
	docker build \
		--build-arg=LDFLAGS="$(LDFLAGS)" \
		-t $(DOCKER_IMG) \
		-f build/app.Dockerfile .

run-img: build-img
	docker run --rm --name $(CONTAINER_NAME) -p 50051:50051 $(DOCKER_IMG) 


version: build
	$(BIN_DIR)/$(APP_NAME) version

test:
	go test ./...

test-race:
	go test -race -count=10 -v ./...

test-integration:
	go test ./internal/integration -tags=integration -count=1 -v	

# -------- Lint --------
GOLANGCI_LINT_VERSION := v1.64.2
GOLANGCI_LINT_BIN := $(shell go env GOPATH)/bin/golangci-lint

install-lint-deps:
	@mkdir -p $(shell go env GOPATH)/bin
	@($(GOLANGCI_LINT_BIN) version 2>/dev/null | grep -q "$(GOLANGCI_LINT_VERSION)") || \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

lint: install-lint-deps
	@$(GOLANGCI_LINT_BIN) run ./...


# -------- Documentation --------
docs:
	plantuml -tpng docs/architecture/*.puml -o ../generated/docs/architecture/	

.PHONY: generate build build-cli run build-img run-img version test test-race test-integration install-lint-deps lint docs
