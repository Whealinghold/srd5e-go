## Summary

<!-- What changed and why. Link the spec, issue or rule this implements. -->

## Testing

<!-- doc/rules/testing.md §2. "N/A" or "covered by existing tests" alone is not acceptable. -->

- Tests added or changed, by pyramid level (unit / use case / contract / scenario):
- Happy and unhappy paths covered:
- Requirements covered (and any spec requirement without a test):
- Coverage of changed files (from `go test -race -shuffle=on -coverprofile=coverage.txt ./...`):
- Mutation score for changed domain lines and surviving mutants, from `gomu` run in Docker with `--cpus` and `--workers` capped at a quarter of the host's processors (or the manual mutations tried):
- Deliberately not covered, and why:

## Go rules

<!-- doc/rules/go.md checklist and doc/rules/architecture.md §9. -->

- [ ] `make check` passes (gofmt, go vet, golangci-lint, `go test -race -shuffle=on -cover`, `go mod tidy`, architecture tests)
- [ ] Every exported identifier has a doc comment; every fallible function documents what it returns on each failure class
- [ ] Public API additions follow the compatibility promise in doc/rules/go.md §9

## Observability

<!-- doc/rules/observability.md §2. If nothing changed, say specifically why. -->

- Signals added, changed or removed (and `doc/observability/signals.md` updated):
- What a consumer should monitor or alert on:
- For a bug fix, the signal that would have exposed it sooner:

## Changelog

- [ ] `CHANGELOG.md` updated under `Unreleased` (or no user-visible change)
- [ ] Breaking changes marked **BREAKING**
