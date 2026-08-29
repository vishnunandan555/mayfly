.POSIX:
.PHONY: all build test test-race vet audit demo clean

all: build test vet audit

build:
	mkdir -p bin
	go build -o bin/mayfly ./cmd/mayfly
	go build -o bin/tui-demo ./cmd/tui-demo
	go build -o bin/mayfly-child ./cmd/mayfly-child

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

audit:
	./zero-dep-audit.sh

demo:
	go run ./cmd/tui-demo

clean:
	rm -rf bin
