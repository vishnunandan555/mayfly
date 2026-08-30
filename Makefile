.PHONY: all build install uninstall test test-race vet clean

BINARY := bin/mayfly
ALIAS := bin/mf

all: build

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/mayfly
	@ln -sf mayfly $(ALIAS) 2>/dev/null || cp $(BINARY) $(ALIAS)
	@echo "Built $(BINARY) and $(ALIAS)"

install:
	@./install.sh

uninstall:
	@./install.sh --uninstall

test:
	go test -v ./...

test-race:
	go test -race -v ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
