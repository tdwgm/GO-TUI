BIN := tui-example
VERSION ?= 0.1.0

.PHONY: build run test clean tidy

build:
	CGO_ENABLED=0 go build -trimpath \
		-ldflags "-s -w -X $(shell go list -m)/internal/tui.Version=$(VERSION)" \
		-o bin/$(BIN) ./cmd/tui-example
	cp -f bin/$(BIN) bin/go-tui

run:
	go run ./cmd/$(BIN)

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin
