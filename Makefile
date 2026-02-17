BINARY_NAME := ai-instructions
BUILD_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

.PHONY: build test lint clean install docker

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ai-instructions

test:
	go test -race -count=1 ./...

lint:
	@which golangci-lint > /dev/null 2>&1 || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
	go clean -testcache

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):$(VERSION) .

.DEFAULT_GOAL := build
