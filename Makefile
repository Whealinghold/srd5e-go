.PHONY: fmt vet lint test cover cover-check fuzz-smoke arch tidy-check check mutate

FUZZTIME ?= 5s
MINCOVER ?= 90
# COVERLOG: reuse an existing `go test -cover` output file in cover-check instead of rerunning the tests.
COVERLOG ?=

fmt:
	gofmt -l -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test -race -shuffle=on -cover ./...

cover:
	go test -race -shuffle=on -coverprofile=coverage.txt ./...
	go tool cover -func=coverage.txt | tail -n 1

# Runs every Fuzz* target briefly (go test allows one -fuzz target per package).
fuzz-smoke:
	@for pkg in $$(go list ./...); do \
	  for f in $$(go test -list '^Fuzz' $$pkg | grep '^Fuzz'); do \
	    echo "fuzz $$pkg $$f"; \
	    go test -run='^$$' -fuzz="^$$f$$" -fuzztime=$(FUZZTIME) $$pkg || exit 1; \
	  done; \
	done

# Fails when any package is below MINCOVER percent (scenarios, internal and packages without statements are exempt).
cover-check:
	@{ if [ -n "$(COVERLOG)" ]; then cat "$(COVERLOG)"; else go test -cover ./...; fi; } | awk -v min=$(MINCOVER) '/coverage:/ && !/\[no statements\]/ && $$2 !~ /\/(scenarios|internal)(\/|$$)/ { v=$$5; sub(/%/,"",v); if (v+0 < min) { print "below " min "%: " $$0; bad=1 } } END { exit bad }'

arch:
	@if [ -d internal/archtest ]; then go test ./internal/archtest/...; else echo "no internal/archtest yet"; fi

tidy-check:
	go mod tidy -diff

# Same sequence as CI (mutation testing runs in capped Docker, in CI on pull requests; see doc/rules/testing.md).
check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt needed"; exit 1; }
	$(MAKE) vet lint cover cover-check tidy-check arch fuzz-smoke FUZZTIME=10s

# Mutation testing, Docker only, CPUs and workers capped at 1/4 of nproc (doc/rules/testing.md section 8).
# gomu takes ONE path and not ./...; "." recurses over the whole module. Override: make mutate PKG=./srderr
# Without --ci-mode gomu exits 0 even below the threshold.
PKG ?= .
mutate:
	@cpus=$$(( $$(nproc) / 4 )); cpus=$$(( cpus < 1 ? 1 : cpus )); \
	docker run --rm --cpus="$$cpus" -v "$$PWD":/workspace -w /workspace golang:1.26 \
	  sh -c "go install github.com/sivchari/gomu/cmd/gomu@v0.2.1 && gomu run --workers=$$cpus --incremental=false --ci-mode --threshold=80 $(PKG)"
