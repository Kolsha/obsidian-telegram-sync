BINARY_NAME := obsidian-telegram-sync
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test lint clean run docker-build

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/obsidian-telegram-sync/

test:
	go test ./... -v -race -coverprofile=coverage.out

test-short:
	go test ./... -short

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ coverage.out

run: build
	./bin/$(BINARY_NAME) -config config.yaml

docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run:
	docker run --rm -v $(PWD)/config.yaml:/app/config.yaml -v $(PWD)/vault:/app/vault $(BINARY_NAME):$(VERSION)
