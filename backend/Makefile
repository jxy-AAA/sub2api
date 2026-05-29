.PHONY: build generate lint test test-default test-unit test-race test-contract test-integration test-e2e test-e2e-local

VERSION ?= $(shell tr -d '\r\n' < ./cmd/server/VERSION)
LDFLAGS ?= -s -w -X main.Version=$(VERSION)

build:
	CGO_ENABLED=0 go build -tags=embed -ldflags="$(LDFLAGS)" -trimpath -o bin/server ./cmd/server

generate:
	go generate ./ent
	go generate ./cmd/server

test: test-default test-unit test-race test-contract lint

lint:
	golangci-lint run ./...

test-default:
	go test ./...

test-unit:
	go test -tags=unit ./...

test-race:
	go test -race ./...

test-contract:
	go test -tags=contract ./internal/repository -run "TestMigrationsRunner_|TestUsageCleanupRepositoryIntegration_"

test-integration:
	go test -tags=integration ./...

test-e2e:
	./scripts/e2e-test.sh

test-e2e-local:
	go test -tags=e2e -v -timeout=300s ./internal/integration/...
