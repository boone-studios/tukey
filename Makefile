# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Binary info
BINARY_NAME=tukey
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_MAC=$(BINARY_NAME)_darwin

# Build info
VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Linker flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build clean test deps fmt vet install

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./cmd/tukey

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_UNIX) -v ./cmd/tukey

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_WINDOWS) -v ./cmd/tukey

build-mac:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_MAC) -v ./cmd/tukey

build-all: build-linux build-windows build-mac

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)
	rm -f $(BINARY_MAC)

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

deps:
	$(GOMOD) download
	$(GOMOD) verify

fmt:
	$(GOFMT) -s -w .

vet:
	$(GOCMD) vet ./...

lint:
	golangci-lint run

install: build
	cp $(BINARY_NAME) /usr/local/bin/

dev:
	$(GOCMD) run ./cmd/tukey $(ARGS)

# Example usage: make dev ARGS="-v ./testdata/sample_project"
example:
	./$(BINARY_NAME) -v ./testdata/sample_project

# Release targets
release: clean deps test build-all
	@echo "Built binaries:"
	@ls -la $(BINARY_NAME) $(BINARY_UNIX) $(BINARY_WINDOWS) $(BINARY_MAC)

.DEFAULT_GOAL := build