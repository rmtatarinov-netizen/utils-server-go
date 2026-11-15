GO ?= go
BIN := bin/server
IMAGE ?= utils-server-go
TAG ?= latest

.PHONY: build run test test-race cover vet fmt fmt-check ci docker-build docker-up docker-down clean

build:
	$(GO) build -o $(BIN) ./cmd/server

run:
	$(GO) run ./cmd/server

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "gofmt needed for:"; echo "$$files"; exit 1; fi

ci: fmt-check vet test-race

docker-build:
	docker build -t $(IMAGE):$(TAG) .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

clean:
	rm -rf bin coverage.out
