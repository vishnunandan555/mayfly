#!/usr/bin/env bash
set -euo pipefail

echo "== go.mod =="
cat go.mod
echo

echo "== module graph =="
go list -m all
echo

echo "== non-stdlib module paths =="
if go list -deps ./... | grep -E '^(github\.com/|gitlab\.com/|bitbucket\.org/|golang\.org/x/)'; then
    echo
    echo "FAIL: third-party dependency detected"
    exit 1
else
    echo "PASS"
fi
echo

echo "== source imports =="
if grep -RInE '"(github\.com|gitlab\.com|bitbucket\.org|golang\.org/x)/' \
    --include='*.go' .; then
    echo
    echo "FAIL: third-party import found"
    exit 1
else
    echo "PASS"
fi
echo

echo "== vendor directories =="
if find . -type d \( -name vendor -o -name third_party -o -name external -o -name deps \) -print | grep .; then
    echo
    echo "WARNING: suspicious dependency directory found"
    exit 1
else
    echo "PASS"
fi
echo

echo "== external command usage =="
if grep -RInE 'exec\.Command|exec\.CommandContext' --include='*.go' .; then
    echo
    echo "WARNING: external command execution found; review manually"
else
    echo "PASS"
fi
echo

echo "== build =="
go build ./...
echo "PASS"

echo
echo "== tests =="
go test ./...
echo "PASS"

echo
echo "== vet =="
go vet ./...
echo "PASS"

echo
echo "ZERO-DEPENDENCY AUDIT PASSED"
