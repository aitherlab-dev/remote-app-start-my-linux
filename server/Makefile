.PHONY: help build test run clean fmt vet lint cover cover-html scan-debug icons-debug integration install uninstall package

GOBIN ?= $(shell go env GOPATH)/bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PACKAGING_DIR := $(abspath ../packaging)
DIST_DIR := dist
PACKAGE_NAME := remotelauncher-$(VERSION)-linux-amd64
PACKAGE_TARBALL := $(DIST_DIR)/$(PACKAGE_NAME).tar.gz

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
	@echo "  icons-debug - probe real system icon themes for a few well-known apps"
	@echo "  integration - run integration tests (build tag 'integration')"
	@echo "  install     - install binary + systemd user unit under \$$HOME"
	@echo "  uninstall   - remove the binary + systemd user unit"
	@echo "  package     - produce dist/remotelauncher-<version>-linux-amd64.tar.gz"

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

icons-debug:
	go test -tags=manual -run TestFinder_RealSystem ./internal/icons/... -v

integration:
	go test -tags=integration -count=1 ./cmd/remotelauncher/...

install: build
	$(PACKAGING_DIR)/install.sh

uninstall:
	$(PACKAGING_DIR)/uninstall.sh

package: build
	rm -rf $(DIST_DIR)/$(PACKAGE_NAME) $(PACKAGE_TARBALL)
	mkdir -p $(DIST_DIR)/$(PACKAGE_NAME)
	install -m 0755 bin/remotelauncher $(DIST_DIR)/$(PACKAGE_NAME)/remotelauncher
	install -m 0644 $(PACKAGING_DIR)/remotelauncher.service $(DIST_DIR)/$(PACKAGE_NAME)/remotelauncher.service
	install -m 0755 $(PACKAGING_DIR)/install.sh $(DIST_DIR)/$(PACKAGE_NAME)/install.sh
	install -m 0755 $(PACKAGING_DIR)/uninstall.sh $(DIST_DIR)/$(PACKAGE_NAME)/uninstall.sh
	install -m 0644 config.example.toml $(DIST_DIR)/$(PACKAGE_NAME)/config.example.toml
	@if [ -f README.md ]; then install -m 0644 README.md $(DIST_DIR)/$(PACKAGE_NAME)/README.md; fi
	tar -C $(DIST_DIR) -czf $(PACKAGE_TARBALL) $(PACKAGE_NAME)
	rm -rf $(DIST_DIR)/$(PACKAGE_NAME)
	@echo "packaged: $(PACKAGE_TARBALL)"
