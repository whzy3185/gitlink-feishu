MODULE   := github.com/gitlink-org/gitlink-cli
BINARY   := gitlink-cli
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X '$(MODULE)/cmd.Version=$(VERSION)'

.PHONY: build install clean test check vet fmt cover lint

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -f $(BINARY)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	@unformatted=$$(gofmt -s -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files not formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

check: fmt vet lint test
	@echo "All checks passed."

hooks:
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed."
