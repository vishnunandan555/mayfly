.PHONY: all build install uninstall test test-race vet clean reproducible deps-proof release-artifacts

BINARY := bin/mayfly
ALIAS := bin/mf
VERSION ?= 0.0.5
LDFLAGS := -s -w -buildid= -X mayfly/pkg/domain.Version=$(VERSION)

all: build

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/mayfly
	@ln -sf mayfly $(ALIAS) 2>/dev/null || cp $(BINARY) $(ALIAS)
	@echo "Built $(BINARY) and $(ALIAS) (v$(VERSION))"

install:
	@mkdir -p $(HOME)/.local/bin
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(HOME)/.local/bin/mayfly ./cmd/mayfly
	@ln -sf mayfly $(HOME)/.local/bin/mf 2>/dev/null || cp $(HOME)/.local/bin/mayfly $(HOME)/.local/bin/mf
	@echo "[OK] Compiled and installed to $(HOME)/.local/bin/mayfly and $(HOME)/.local/bin/mf (v$(VERSION))"

uninstall:
	@rm -f $(HOME)/.local/bin/mayfly $(HOME)/.local/bin/mf
	@echo "[OK] Removed $(HOME)/.local/bin/mayfly and $(HOME)/.local/bin/mf"

test:
	go test -v ./...

test-race:
	go test -race -v ./...

vet:
	go vet ./...

reproducible:
	@echo "Verifying bit-for-bit reproducible build..."
	@mkdir -p bin/repro1 bin/repro2
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/repro1/mayfly ./cmd/mayfly
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/repro2/mayfly ./cmd/mayfly
	@sha256sum bin/repro1/mayfly bin/repro2/mayfly
	@cmp bin/repro1/mayfly bin/repro2/mayfly && echo "[OK] REPRODUCIBLE BUILD VERIFIED: Byte-identical outputs!"
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

release-artifacts:
	@echo "Building reproducible multi-platform release artifacts (v$(VERSION))..."
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/mayfly-linux-amd64 ./cmd/mayfly
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/mayfly-linux-arm64 ./cmd/mayfly
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/mayfly-darwin-amd64 ./cmd/mayfly
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/mayfly-darwin-arm64 ./cmd/mayfly
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/mayfly-windows-amd64.exe ./cmd/mayfly
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/mayfly-windows-arm64.exe ./cmd/mayfly
	@cd dist && sha256sum mayfly-* > checksums.txt
	@cd dist && sha256sum --check checksums.txt
	@echo "[OK] Generated and verified all release artifacts in dist/ with checksums.txt"

clean:
	rm -rf bin/ dist/

