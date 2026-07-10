MODULE   := github.com/gitlink-org/gitlink-cli
BINARY   := gitlink-cli
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X '$(MODULE)/cmd.Version=$(VERSION)'

.PHONY: build install clean test test-cover lint ci

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -f $(BINARY)

test:
	go test -v -race ./...

test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

ci: lint test

vet:
	go vet ./...
