.PHONY: build

build:
	@mkdir -p bin
	go build -o bin/teller ./cmd/teller

.DEFAULT_GOAL := build
