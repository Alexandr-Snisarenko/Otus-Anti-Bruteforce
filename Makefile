# -------- Project meta --------
APP_NAME      := anti-bruteforce
CLI_NAME      := abfctl

BIN_DIR       := ./bin
DOCKER_IMG="anti-bruteforce:develop"

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X version.release="develop" -X version.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X version.gitHash=$(GIT_HASH)

#Postrges
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= password
POSTGRES_DB ?= backend
POSTGRES_PORT ?= 5435
POSTGRES_CONTAINER := postgres-calendar

run-postgres:
	docker run -d --name $(POSTGRES_CONTAINER) \
	-e POSTGRES_USER=$(POSTGRES_USER) \
	-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
	-e POSTGRES_DB=$(POSTGRES_DB) \
	-p $(POSTGRES_PORT):5432 \
	-v postgres-data:/var/lib/postgresql/data \
	postgres:latest

stop-postgres:
	docker stop $(POSTGRES_CONTAINER) || true
	docker rm $(POSTGRES_CONTAINER) || true	

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
	go build -ldflags "$(LDFLAGS)" -v -o $(BIN_DIR)/$(APP_NAME) ./cmd/anti-bruteforce
#	go build -ldflags "$(LDFLAGS)" -v -o $(BIN_DIR)/$(CLI_NAME) ./cmd/abfctl


run: build
	$(BIN_DIR)/$(APP_NAME) -config ./configs/config.yml

build-img:
	docker build \
		--build-arg=LDFLAGS="$(LDFLAGS)" \
		-t $(DOCKER_IMG) \
		-f build/Dockerfile .

run-img: build-img
	docker run $(DOCKER_IMG)

version: build
	$(BIN) version

test:
	go test ./...

test-race:
	go test -race -count=100 -v ./...

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

.PHONY: build run build-img run-img version test lint generate-openapi
