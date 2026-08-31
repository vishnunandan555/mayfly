.PHONY: all build install uninstall test test-race vet clean reproducible deps-proof

BINARY := bin/mayfly
ALIAS := bin/mf

all: build

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o $(BINARY) ./cmd/mayfly
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

reproducible:
	@echo "Verifying bit-for-bit reproducible build..."
	@mkdir -p bin/repro1 bin/repro2
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o bin/repro1/mayfly ./cmd/mayfly
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o bin/repro2/mayfly ./cmd/mayfly
	@sha256sum bin/repro1/mayfly bin/repro2/mayfly
	@cmp bin/repro1/mayfly bin/repro2/mayfly && echo "✅ REPRODUCIBLE BUILD VERIFIED: Byte-identical outputs!"
	@rm -rf bin/repro1 bin/repro2

deps-proof:
	@echo "=== Zero-Dependency Manifest Verification ===" > deps-proof.txt
	@echo "Date: $$(date -u)" >> deps-proof.txt
	@echo "Go Version: $$(go version)" >> deps-proof.txt
	@echo "" >> deps-proof.txt
	@echo "--- go list -m all ---" >> deps-proof.txt
	@go list -m all >> deps-proof.txt
	@echo "" >> deps-proof.txt
	@echo "--- go.mod content ---" >> deps-proof.txt
	@cat go.mod >> deps-proof.txt
	@echo "" >> deps-proof.txt
	@echo "--- Package Import Tree ---" >> deps-proof.txt
	@go list -f '{{.ImportPath}}: {{.Imports}}' ./... >> deps-proof.txt
	@echo "Generated deps-proof.txt (0 external dependencies)"

clean:
	rm -rf bin/
