BINARY_NAME := gemini-web-cli
BIN_DIR := bin
BIN_PATH := $(BIN_DIR)/$(BINARY_NAME)
OUT_DIR := dist
CMD := .
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X github.com/Leechael/gemini-web-cli/cmd.Version=$(VERSION)"
GOFLAGS ?= -buildvcs=false

export GOFLAGS

.PHONY: tidy fmt test ci build run install clean cross-build help

help:
	@echo "Targets:"
	@echo "  make tidy         - go mod tidy"
	@echo "  make fmt          - gofmt ./cmd ./internal"
	@echo "  make test         - run unit tests"
	@echo "  make ci           - fmt check + vet + tests + build"
	@echo "  make build        - build local binary to ./$(BIN_PATH)"
	@echo "  make run          - run CLI (pass ARGS='...')"
	@echo "  make install      - install binary to GOPATH/bin"
	@echo "  make clean        - remove build artifacts"
	@echo "  make cross-build  - build darwin/linux amd64/arm64 binaries"

tidy:
	go mod tidy

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./... -count=1

ci:
	@unformatted=$$(gofmt -l ./cmd ./internal); \
	if [ -n "$$unformatted" ]; then echo "Unformatted files:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go test ./... -count=1
	go build -v $(CMD)

build:
	mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_PATH) $(CMD)

run:
	go run $(CMD) $(ARGS)

install:
	go install $(CMD)

clean:
	rm -rf $(OUT_DIR) $(BIN_DIR)

cross-build: clean
	mkdir -p $(OUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(OUT_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -o $(OUT_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD)
	GOOS=linux GOARCH=amd64 go build -o $(OUT_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD)
	GOOS=linux GOARCH=arm64 go build -o $(OUT_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD)
