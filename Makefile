.PHONY: build

build:
	@mkdir -p bin
	go build -o bin/teller ./cmd/teller

install: build
	sudo cp bin/teller /usr/local/bin/teller

.DEFAULT_GOAL := build
