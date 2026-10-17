set shell := ["bash", "-euo", "pipefail", "-c"]

# Minimum per-package statement coverage, in percent.
mincover := "90"

# Running `just` with no arguments lists the recipes.
[private]
default:
    @just --list --unsorted

[doc("Format all Go files in place (gofmt)")]
fmt:
    gofmt -l -w .

[doc("Run go vet")]
vet:
    go vet ./...

[doc("Run golangci-lint")]
lint:
    golangci-lint run ./...

[doc("Run the test suite with -race -shuffle=on -cover")]
test:
    go test -race -shuffle=on -cover ./...

[doc("Write coverage.txt and print the total statement coverage")]
cover:
    go test -race -shuffle=on -coverprofile=coverage.txt ./...
    go tool cover -func=coverage.txt | tail -n 1

# go test allows one -fuzz target per package, so each Fuzz* target runs in turn.
[doc("Run every Fuzz* target briefly; arg: fuzztime (default 5s)")]
fuzz-smoke fuzztime="5s":
    #!/usr/bin/env bash
    set -euo pipefail
    for pkg in $(go list ./...); do
      for f in $(go test -list '^Fuzz' "$pkg" | grep '^Fuzz' || true); do
        echo "fuzz $pkg $f"
        go test -run='^$' -fuzz="^$f$" -fuzztime={{ fuzztime }} "$pkg"
      done
    done

# scenarios, internal and packages without statements are exempt.
[doc("Fail if any package is below the coverage floor; arg: covlog = saved `go test -cover` output to reuse")]
cover-check covlog="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -n "{{ covlog }}" ]; then cat "{{ covlog }}"; else go test -cover ./...; fi |
      awk -v min={{ mincover }} '/coverage:/ && !/\[no statements\]/ && $2 !~ /\/(scenarios|internal)(\/|$)/ { v=$5; sub(/%/,"",v); if (v+0 < min) { print "below " min "%: " $0; bad=1 } } END { exit bad }'

[doc("Run the architecture tests (internal/archtest)")]
arch:
    @if [ -d internal/archtest ]; then go test ./internal/archtest/...; else echo "no internal/archtest yet"; fi

[doc("Fail if go mod tidy would change go.mod or go.sum")]
tidy-check:
    go mod tidy -diff

# Mutation testing runs in capped Docker, in CI on pull requests; see doc/rules/testing.md.
[doc("Everything CI runs: gofmt, vet, lint, coverage, tidy, arch, fuzz-smoke")]
check:
    @test -z "$(gofmt -l .)" || { gofmt -l .; echo "gofmt needed"; exit 1; }
    just vet lint cover cover-check tidy-check arch
    just fuzz-smoke 10s

# Docker only, CPUs and workers capped at 1/4 of nproc (doc/rules/testing.md section 8).
# gomu takes ONE path and not ./...; "." recurses over the whole module.
# Without --ci-mode gomu exits 0 even below the threshold.
[doc("Mutation tests in capped Docker; arg: pkg (default ., e.g. ./srderr)")]
mutate pkg=".":
    #!/usr/bin/env bash
    set -euo pipefail
    cpus=$(( $(nproc) / 4 )); cpus=$(( cpus < 1 ? 1 : cpus ))
    docker run --rm --cpus="$cpus" -v "$PWD":/workspace -w /workspace golang:1.26 \
      sh -c "go install github.com/sivchari/gomu/cmd/gomu@v0.2.1 && gomu run --workers=$cpus --incremental=false --ci-mode --threshold=80 {{ pkg }}"
