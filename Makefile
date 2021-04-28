# Go parameters
GO_BUILD=go build
BINARY_NAME=build/notifier
BINARY_MAC_SUFFIX=_darwin_amd64

.PHONY: build

build: build-mac

# Build for mac
build-mac:
	$(GO_BUILD) -o $(BINARY_NAME)$(BINARY_MAC_SUFFIX) -v cmd/notifier.go

lint:
	docker run --rm -v `pwd`:/app -w /app golangci/golangci-lint:latest golangci-lint run

lint-verbose:
	docker run --rm -v `pwd`:/app -w /app golangci/golangci-lint:latest golangci-lint run -v
