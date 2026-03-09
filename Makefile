.PHONY: build run test clean install lint release-dry docs

BINARY := mcpx
CMD := ./cmd/mcpx
VERSION ?= dev
LDFLAGS := -s -w -X github.com/codestz/mcpx/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

# Preview goreleaser without publishing
release-dry:
	goreleaser release --snapshot --clean

# Dev server for docs
docs:
	cd website && npx vitepress dev

# Cross-compile for distribution
build-all:
	GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64   $(CMD)
	GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(CMD)
