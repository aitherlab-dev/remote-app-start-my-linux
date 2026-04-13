.PHONY: help build test run clean fmt vet

help:
	@echo "Available targets:"
	@echo "  build  - build the remotelauncher binary into bin/"
	@echo "  test   - run all Go tests"
	@echo "  run    - build and run the binary"
	@echo "  clean  - remove build artifacts"
	@echo "  fmt    - run go fmt on all packages"
	@echo "  vet    - run go vet on all packages"

build:
	go build -o bin/remotelauncher ./cmd/remotelauncher

test:
	go test ./...

run: build
	./bin/remotelauncher

clean:
	rm -rf bin coverage.out coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...
