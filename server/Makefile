.PHONY: help build test run clean fmt vet lint cover cover-html scan-debug

GOBIN ?= $(shell go env GOPATH)/bin

help:
	@echo "Available targets:"
	@echo "  build       - build the remotelauncher binary into bin/"
	@echo "  test        - run all Go tests"
	@echo "  run         - build and run the binary"
	@echo "  clean       - remove build artifacts"
	@echo "  fmt         - format code with gofumpt and goimports"
	@echo "  vet         - run go vet on all packages"
	@echo "  lint        - run golangci-lint on all packages"
	@echo "  cover       - run tests with coverage and print summary"
	@echo "  cover-html  - generate HTML coverage report"
	@echo "  scan-debug  - scan real system .desktop files and print summary"

build:
	go build -o bin/remotelauncher ./cmd/remotelauncher

test:
	go test ./...

run: build
	./bin/remotelauncher

clean:
	rm -rf bin coverage.out coverage.html

fmt:
	$(GOBIN)/gofumpt -w .
	$(GOBIN)/goimports -w .

vet:
	go vet ./...

lint:
	$(GOBIN)/golangci-lint run ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

cover-html: cover
	go tool cover -html=coverage.out -o coverage.html

scan-debug:
	go test -tags=manual -run TestScan_RealSystem ./internal/desktop/... -v
