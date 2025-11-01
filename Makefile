.PHONY: build

VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null)
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION_PKG := git.sr.ht/~jakintosh/teller/internal/version
LDFLAGS := -X '$(VERSION_PKG).rawVersion=$(VERSION)' \
	-X '$(VERSION_PKG).rawCommit=$(COMMIT)' \
	-X '$(VERSION_PKG).rawDate=$(BUILD_DATE)'

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/teller ./cmd/teller

install: build
	sudo cp bin/teller /usr/local/bin/teller

.DEFAULT_GOAL := build
