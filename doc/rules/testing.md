# srd5e Testing Rules

Normative. MUST / MUST NOT / SHOULD are binding on all code and review.
Read together with [architecture.md](architecture.md) (the per-layer test
approach), [go.md](go.md) (Go conventions) and
[observability.md](observability.md) (testing telemetry).

## 1. Principles

1. **Every change is tested.** No behaviour is added, changed, fixed or removed
   without tests that would fail if it broke. "Tests pass" only means something
   if the right tests exist.
2. **Follow the test pyramid.** Many fast, isolated tests at the base; fewer,
   broader tests as you go up. Do not invert it.
3. **Test the unhappy paths as seriously as the happy path.** Rules engines fail
   at edges: boundaries, invalid input, stacking conflicts, missing content.
4. **Tests document requirements.** Each test says what requirement or rule it
   verifies, so a reader can trace from requirement to test and back.
5. **Tests are deterministic and independent.** The same result every run, in
   any order.

---

## 2. Evaluating test coverage is mandatory

For **every change** (new feature, changed behaviour, new port or adapter,
refactor, bug fix) you MUST work out what tests it needs before you finish. Do
the thinking; do not just add one test and move on.

1. **What behaviour changed?** List each observable behaviour added, altered or
   removed.
2. **Which pyramid level tests each?** Prefer the lowest level that can prove
   it (§3).
3. **What is the happy path?** The ordinary, valid, expected case.
4. **What are the unhappy paths?** Walk the checklist in §4 and cover every item
   that applies.
5. **Which requirement or rule does each test verify?** Identify it and cite it
   (§5). If there is none, say so.
6. **Is existing coverage now wrong or stale?** Update or remove tests that
   encode old behaviour. Do not leave them failing, skipped or lying.
7. **What could still break without a test noticing?** Check the tests would
   fail if the implementation were wrong (§8).

### The outcome

- Write the tests the evaluation calls for.
- Your PR description (see `.github/pull_request_template.md`) and final
  summary MUST contain a `Testing` entry:
  tests added or changed, by pyramid level; which happy and unhappy paths and
  requirements they cover; and anything deliberately not covered, with the
  reason.
- If a change adds no tests, justify it specifically (for example: "comment
  only change"; "pure rename, covered unchanged by existing `TestCheck`").
  "Covered by existing tests" or "N/A" alone is not acceptable.
- **Bug fixes:** first write a test that reproduces the bug and watch it fail,
  then fix, then watch it pass. Keep the test as the regression test and cite
  the bug.
- Reviewers MUST reject a change that lacks the evaluation, the tests or the
  report.

---

## 3. The test pyramid

```
              /\
             /  \        Scenario tests         few, slow-ish, whole slices
            /----\       through the public API
           /      \
          /--------\     Application + contract tests    some
         /          \    real domain, test-double ports; adapters vs port contract
        /------------\
       /              \  Unit tests (domain)            many, tiny, instant
      /----------------\ no ports, no I/O, no doubles
```

Cross-cutting guardrail, outside the pyramid: architecture tests (§9).

| Level | Location | What it proves | Uses | Share |
|---|---|---|---|---|
| **Unit** | `<pkg>/*_test.go` | Rules logic: value objects, parsers, pure domain functions. Every rule, edge and invariant. | Nothing but the code under test | The large majority |
| **Application** | `<context>/*_test.go` (a use case in a context package such as `d20`) | A use case orchestrates correctly: calls ports, returns the right result, emits the right telemetry. | Real domain; **test doubles** from `dicetest` / `telemetrytest` | Substantial |
| **Contract** | `<port pkg>test/` (the exported suite) run from each adapter's and double's own tests | Each adapter and double honours its port's contract; every implementation passes the same suite. | The real adapter | Moderate |
| **Scenario** | `scenarios/` (`package scenarios_test`) | Several packages work together through public API only, end to end for a realistic situation. | Public API, test doubles for the outside world | Few |

Rules:

- **Test at the lowest level that can prove the behaviour.** If a unit test can
  prove it, do not prove it with a scenario test.
- **Every rule gets unit tests.** Scenario tests supplement them; they never
  replace them.
- **Do not build an ice-cream cone.** Many scenario tests and few unit tests is
  a defect. Slow, broad tests catch regressions late and point nowhere.
- **No inverted dependencies.** A unit test of a pure domain function imports no
  ports or doubles. If it needs one, the design or the level is wrong.
- **External test packages by default.** Public-behaviour tests live in
  `package x_test` in the same directory, so they see only what a consumer
  sees. Use an internal `package x` test file only for unexported edge logic
  that cannot be reached through the public API, and name it `x_internal_test.go`.
  `scenarios/` MUST use `package scenarios_test` and import only public
  packages, never `internal/`, so it also protects the consumer-facing contract.
- Test files sit next to the code they test (`check.go` / `check_test.go`).
  Fixtures live in `testdata/` (ignored by the Go tool); golden files are
  compared byte for byte and regenerated only by an explicit `-update` flag
  declared in the test.

---

## 4. Happy and unhappy paths

Every behaviour is tested for the happy path **and** for each applicable
unhappy path. A suite with only happy-path tests is incomplete.

### Happy path

The normal, valid case with typical inputs and the expected outcome. Cover every
distinct valid outcome, not just one (for a D20 Test: both success and failure).

### Unhappy-path checklist

Work through every item and cover the ones that apply.

- **Boundaries:** exactly at, one below and one above every threshold (DC
  differences of ±10, natural 1 and natural 20, min/max level, zero, negative,
  maximum values).
- **Invalid input:** out of range, empty, zero value, malformed or contradictory
  data. Constructors and parsers reject invalid input with a typed error.
- **Empty and minimal cases:** no modifiers, no targets, a nil or empty slice,
  the lowest level. Nil and empty are both tried where the API accepts them.
- **Rule interactions and conflicts:** several modifiers on one roll, Advantage
  and Disadvantage together, several conditions at once, ordering effects.
- **Failure of collaborators:** a port that returns its error (wrapped with
  `srderr.WrapPort`), its not-found or rejected result, or invalid data
  (through the test doubles). Include a port that panics, where `srderr.Guard`
  applies.
- **Missing or bad content:** unknown id, missing field, unsupported version,
  duplicate entries, content that fails to decode.
- **State errors:** an action that is illegal in the current state, a repeated
  or out-of-order action.
- **Determinism and randomness:** results for fixed rolls, including extremes.
  Never depend on an unseeded roller.
- **Immutability:** an operation does not alter its inputs; a returned slice or
  map is a copy that cannot corrupt the value.
- **Cancellation:** a function that takes a `context.Context` is tried with an
  already-cancelled context.
- **Telemetry on failure:** failures are observable as required by
  [observability.md](observability.md).

### Expected outcomes vs errors vs panics

Follow the error model in [architecture.md](architecture.md) and
[errors-and-results.md](../specs/errors-and-results.md):

- A failed check, an illegal-but-expected action or missing content is a
  **returned result**; assert the exact result value or enum and its data.
- Untrusted input that fails validation is an **error** from `NewX` / `ParseX`;
  assert it with `errors.Is` for a sentinel or `errors.As` for a typed error,
  and for decoders assert **every** problem in the `errors.Join`.
- A port's infrastructure failure is an **error** the use case wraps with
  `srderr.WrapPort`; assert `errors.As` to `*srderr.PortError` with the right
  `Port` and `Op`, and `errors.Is` to the cause.
- A programmer error **panics** only in `MustX` helpers and unreachable
  branches; assert with a small `recover` helper that checks the panic value is
  an `error` and matches with `errors.Is`.

Never branch on or compare error message text; error strings are not API. For
each unhappy test, assert something specific. A test that only checks
"err != nil" or "does not crash" is not an unhappy-path test.

---

## 5. Tests document requirements

> Status: planned. The existing Go suite does not yet carry `Requirement:` /
> `Path:` comments and the archtest check below is not implemented; apply the
> convention to new tests and backfill.

Every test and test group SHOULD record the requirement or rule it verifies.
Where a PRD, spec, ticket or rule source exists, it MUST.

### Format

Place a comment immediately above the test function, the `t.Run` call, or the
table:

```go
// Requirement: doc/specs/d20-tests.md FR-023 - a check total that equals the DC
//   succeeds (SRD 5.2.1, "D20 Tests", p. 6).
// Path: happy
func TestCheckTotalEqualToDCSucceeds(t *testing.T) { ... }

// Requirement: doc/specs/d20-tests.md FR-024 - a total one below the DC
//   fails (SRD 5.2.1, "D20 Tests", p. 6).
// Path: unhappy (boundary)
func TestCheckTotalOneBelowDCFails(t *testing.T) { ... }
```

Fields:

| Field | Meaning |
|---|---|
| `Requirement:` | Path to the PRD/spec/doc and the requirement id or section, plus a short paraphrase. For an SRD rule, cite the section and page, e.g. `SRD 5.2.1, "D20 Tests", p. 6`. |
| `Path:` | `happy`, or `unhappy` with the kind in brackets (boundary, invalid input, failure, ...) |

Additional fields when useful: `Rule:` (game rule source), `Bug:` (ticket or
issue for a regression test), `Note:` (a non-obvious reason).

### Rules

- **Cite precisely.** A path and requirement id (or section heading), not just
  "see the spec". A reader MUST be able to find the requirement from the
  comment.
- **If implementing a PRD or spec, every test MUST cite the requirement it
  covers,** and every requirement SHOULD be covered by at least one test. After
  implementing, check each requirement has a test; list any that do not.
- **If the spec has no ids, add them** (or cite the section heading) rather than
  skipping the citation.
- **If there is no governing document,** say what behaviour the test pins in a
  `Requirement:` comment in your own words (for example `Requirement: none -
  pins current behaviour of ...`). Do not invent a citation.
- **One requirement per behaviour, not one per file.** Cite at function or
  table level. A file-level comment alone is not enough.
- **Table-driven tests cite per row.** Put the comment on the test function with
  `Requirement: per case` and `Path: per case`. Each case struct then carries
  its own `name`, `requirement` and `path` fields, and `t.Run(c.name, ...)`
  makes a failure point at one row and one requirement:

  ```go
  // Requirement: per case
  // Path: per case
  func TestCheck(t *testing.T) {
      t.Parallel()
      cases := []struct {
          name, requirement, path string
          natural, dc            int
          want                    d20.Outcome
      }{
          {"total equals the DC", `SRD 5.2.1, "Compare the Total to a Target Number", p. 6`, "unhappy (boundary)", 15, 15, d20.Success},
          {"total one below the DC", `SRD 5.2.1, "Compare the Total to a Target Number", p. 6`, "unhappy (boundary)", 14, 15, d20.Failure},
          // ...
      }
      for _, c := range cases {
          t.Run(c.name, func(t *testing.T) {
              t.Parallel()
              roll, err := d20.NewRoll(modifier.Normal, []int{c.natural})
              if err != nil {
                  t.Fatal(err)
              }
              res, err := d20.ResolveCheck(d20.AbilityCheck, roll,
                  d20.NewRollModifiers(modifier.New(0)), d20.DifficultyClass(c.dc))
              if err != nil {
                  t.Fatal(err)
              }
              if got := res.Outcome(); got != c.want {
                  t.Errorf("ResolveCheck(natural %d, DC %d) = %v, want %v [%s]", c.natural, c.dc, got, c.want, c.requirement)
              }
          })
      }
  }
  ```

- **Keep citations true.** When a requirement changes, update or remove the
  tests and their comments.
- **Cite, do not copy.** Reference an SRD rule by section and page. Quote only
  what a test needs to be understood, and only from SRD 5.2.1. Nothing from
  outside the SRD belongs in tests (see "Licensing and source material" in
  `AGENTS.md`).

---

## 6. Writing good tests

- **Standard library only.** Use `testing`; no testify, gomock or similar.
  Compare with `==`, `reflect.DeepEqual` or `slices.Equal` / `maps.Equal`, and
  report with `t.Errorf("F(%v) = %v, want %v", in, got, want)`. Use `t.Fatalf`
  only when the test cannot continue.
- **One behaviour per test.** A failing test names one thing that broke.
- **Names state behaviour and condition** in plain language:
  `TestNatural20ImprovesSuccessToCriticalSuccess`, not `TestCheck1`. Subtest
  names are short phrases; `t.Run` replaces spaces with underscores.
- **Arrange, act, assert,** visibly separated. No branching logic in the
  assertion path; use table-driven tests for variations.
- **Table-driven tests for rule maths.** Enumerate inputs and expected outputs
  in a readable slice of structs, each row with a name, requirement and path
  (§5). The loop that generates subtests is not "logic in the assertion path".
- **Parallel where safe.** Call `t.Parallel()` in top-level tests and subtests
  that share no mutable state. Since Go 1.22 the loop variable is per
  iteration, so no `c := c` copy is needed.
- **Helpers call `t.Helper()`** and take `testing.TB` when benchmarks or fuzz
  targets share them. Use `t.Cleanup`, `t.TempDir` and `t.Setenv` instead of
  manual teardown.
- **Assert outcomes, not implementation.** Check results and observable
  effects, not private calls or call order, unless order is the requirement.
- **Specific assertions.** Exact values and types over a truthiness check on a
  vague expression.
- **Deterministic.** Use a `dicetest` roller. No `rand` without a fixed seed, no
  `time.Now()`, real network or real filesystem. Durations come through the
  observability ports, so `telemetrytest` doubles control them. Never `time.Sleep`.
- **Independent.** No package-level mutable state, no dependency on test order,
  no leftover state. Build fresh fixtures per test. The suite MUST pass with
  `go test -shuffle=on` (§10).
- **Fast.** The unit suite runs in seconds.
- **Readable fixtures.** Small constructor helpers with sensible defaults,
  overridden per test. No giant shared fixtures.
- **Doubles live in `dicetest` and `telemetrytest`.** Use them for ports. Prefer
  fakes and fixed doubles over mocks; verify interactions only when the
  interaction is the requirement. A context declares its own small consumer-side
  interface, and a test may satisfy it with a local fake when no shared double
  fits.
- **No skipped or commented-out tests.** Delete tests that no longer apply, or
  fix them. A `t.Skip` needs a linked issue and a reason.
- **No test-only code in production packages.** Doubles consumers may use belong
  in `<pkg>test`; the rest belong in `_test.go` files.

### Doubles

| Package | Provides | Use for |
|---|---|---|
| `dice/dicetest` | `FixedRoller`, `CallbackRoller`, minimum and maximum rollers | Every dice-dependent test; a fixed sequence for exact results, min/max for extremes, a callback for failure injection |
| `telemetry/telemetrytest` | Recording `Logger`, `Tracer`, `Meter` | Asserting the signals a use case emits and their attributes |

Doubles MUST be tiny and obviously correct, and each MUST pass its port's
contract suite (§7).

---

## 7. Contract suites

A port (`dice.Roller`, `telemetry.Logger` / `Tracer` / `Meter`, and later the
content source) has an exported contract suite (more than one when it has separate
contracts, as `dice` has `RunRollerContract` and `RunFailureContract`) in its
`*test` package, shaped like `testing/iotest` or `testing/fstest`:

```go
// RunRollerContract runs the behavioural contract of dice.Roller against
// rollers built by factory. Each subtest gets a fresh roller.
func RunRollerContract(t *testing.T, factory func(t *testing.T) dice.Roller)
```

- **Every adapter and every double MUST pass the suite,** from its own
  `_test.go`: `dicetest.RunRollerContract(t, func(*testing.T) dice.Roller { return dice.NewSeededRoller(rand.New(rand.NewPCG(1, 2))) })`.
- **Consumers reuse it** against their own adapters; the suite is part of the
  public API and covered by the compatibility promise in [go.md](go.md).
  Exporting `testing` imports from a `*test` package is acceptable (like
  `net/http/httptest`); production packages MUST NOT import `testing`.
- A contract states what is true for **all** implementations (range of results,
  error behaviour, no panics on valid input), never behaviour of one adapter.
  Adapter-specific behaviour is tested in the adapter's own tests.
- When a port's contract changes, update the suite in the same change.

---

## 8. Coverage and checking that tests actually test

### Coverage

- Measure with the standard tool:

  ```sh
  go test -race -shuffle=on -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out      # per-function and total
  go tool cover -html=coverage.out      # browsable report
  ```

  `-race` already implies `-covermode=atomic`. Do not add `-coverpkg=./...`:
  it reports each package as a share of the whole module, which cannot be
  compared with the 90% floor (use it only for a module-wide report). `just
  cover-check` (plain per-package `-cover`) is the authoritative gate; it
  exempts `scenarios`, `internal/` and packages with no statements. Use
  `-cover` for a quick per-package percentage while developing. Report the statement coverage of
  each changed package in your `Testing` entry.
- **Each package MUST reach at least 90% statement coverage.** Every changed or
  added statement in domain and use-case code MUST be covered, including error
  branches. Uncovered statements need a stated reason (an unreachable `panic`
  guard is the usual one).
- **Coverage is necessary, not sufficient.** A line executed with no meaningful
  assertion is not tested. Review tests for quality, not just for the number.
- Do not write tests whose only purpose is to raise the number.

### Mutation testing

For new behaviour, run mutation testing on the changed code. A surviving mutant
means a test is missing or too weak: add or strengthen a test, do not just
re-run. For bug fixes, the new test MUST fail without the fix (§2). A test that
cannot fail is a defect.

The project uses [`gomu`](https://github.com/sivchari/gomu). It mutates the
source (small changes such as `<` to `<=` or `&&` to `||`) and checks that a
test fails for each.

**`gomu` MUST NEVER run directly on the host.** Run raw, it oversubscribes
every core and has crashed machines. It runs only inside Docker, with both the
container CPU quota (`--cpus`) and gomu's `--workers` capped at one quarter of
the host's processors (minimum 1):

```sh
just mutate            # whole module
just mutate ./srderr     # one package
```

`just mutate` wraps this (verified on this host with gomu v0.2.1, 2026-10-17):

```sh
cpus=$(( $(nproc) / 4 )); cpus=$(( cpus < 1 ? 1 : cpus ))
docker run --rm --cpus="$cpus" -v "$PWD":/workspace -w /workspace golang:1.26 \
  sh -c "go install github.com/sivchari/gomu/cmd/gomu@v0.2.1 && gomu run --workers=$cpus --incremental=false --ci-mode --threshold=80 ."
```

Verified gomu v0.2.1 CLI (`gomu run --help`):

- Flags: `--base-branch` (default `main`), `--ci-mode`, `--fail-on-gate`
  (default true), `--incremental` (default true), `--output`
  (console, json, html, text), `--threshold` (default 80), `--timeout`
  (seconds, default 30), `--workers` (default 4), `-v`. There is no `--`
  passthrough to `go test`.
- Takes at most ONE path argument. `./...` fails ("work directory does not
  exist"); use `.` to recurse over the module or one package such as `./srderr`.
- Without `--ci-mode` the exit code is 0 even below the threshold; with
  `--ci-mode` a score under `--threshold` exits non-zero ("quality gate failed").
- `--ci-mode` writes `mutation-report.json`, and runs write `.gomu_history.json`
  (both git-ignored).
- Incremental mode (`--incremental`, `--base-branch`) tests only files changed
  against the base branch. It is UNVERIFIED here; the repo has no commits to diff.
  When trying it, the base must exist locally (for example `origin/main`).
- Whether `.gomuignore` is honoured is UNVERIFIED (not observed in output).
- Scores varied between identical runs on ./srderr (67.7% vs 65.5%), so treat a
  score near the threshold as noisy.
- First run had normal DNS in Docker; `go install` took seconds.

- **Scope:** changed files in kernel packages (`dice`, `ability`, `modifier`,
  `srderr`, `telemetry`) and context domain code MUST be mutation tested with a
  score of at least **80**. Use cases SHOULD be. Report the score and the number
  of survivors in your `Testing` entry. `*test` packages, `internal/archtest`
  and `examples` are excluded (list them in `.gomuignore`).
- **Run one mutation process at a time.** Never raise `--workers` or `--cpus`
  above the quarter cap, and never run gomu outside the container.
- **Tests first.** gomu runs the existing suite for each mutant; make it pass
  and fast before mutating. Set `--timeout` if a legitimate test is slow.
- **Surviving mutants** are fixed with a better test. Only a mutant that
  provably cannot change behaviour (an *equivalent* mutant) may be excluded, and
  the reason is written beside it or in `.gomuignore`.
- **Pinned version.** The justfile `mutate` recipe (also used by CI) holds the fixed tag
  (`@v0.2.1`); upgrading is a deliberate change to both. The tool is young and its behaviour may change.
- **Fallback:** if the tool cannot analyse a file (for example, a very new Go
  language feature, or generics it does not parse), do the manual check
  instead: break the implementation (flip a comparison, drop a branch, off-by-one
  a boundary), confirm a test fails, restore it, and list the mutations tried.
  Do this on a clean working tree and restore with `git restore`/`git stash`,
  never by hand-reverting.

---

## 9. Architecture and guardrail tests

`internal/archtest/` holds stdlib-only tests (`go/parser` AST scan) that
enforce the rules in [architecture.md](architecture.md) (checks A1 to A13, listed
in its section 9): the dependency allow-list as a table (kernel packages import
no context; contexts import kernel and upstream public APIs only; nothing but
tests imports `*test` packages; `internal/` is not imported from outside the
module), forbidden imports and banned calls (no `init()` side effects, no `time.Now`
call in domain code; the `math/rand` import ban holds in kernel and context
packages), the doc-comment rule for exported identifiers, port-use, sealed-set,
error-shape, embedded-data and rule-tag checks, and one `RULE-TAG:` line per
`doc/rules/*.md`. What is not yet mechanised is listed in architecture.md section 9
and in [architecture-guardrails.md](../specs/architecture-guardrails.md); the
observability checks O1 to O5 (signal catalogue consistency among them) are
specified in observability.md section 10 and not yet enforced. Add the following
check too:

- (Planned, not yet enforced.) Every top-level `Test*` function and table-driven test in `_test.go` files
  (outside `internal/archtest/`) is preceded by a `Requirement:` or `Rule:`
  comment, or by an explicit `Requirement: none - ...`. Subtests of a table
  satisfy this through the comment on the enclosing function (`Requirement: per
  case`, §5). `Example*` and `Fuzz*` functions are exempt.

Until the suite exists, the bootstrap rule in
[architecture.md](architecture.md) applies.

A change to a rule or boundary MUST update or add the matching architecture
test.

---

## 10. Fuzzing, examples and how tests run

### Fuzzing

Every parser and decoder (`dice.ParseExpression`, the content decoders, any
`ParseX`) MUST have a `Fuzz*` test in its package's `_test.go`:

```go
func FuzzParseExpression(f *testing.F) {
    for _, s := range []string{"1d20", "2d8+2", "", "d", "99999999999d6"} {
        f.Add(s)
    }
    f.Fuzz(func(t *testing.T, in string) {
        e, err := dice.ParseExpression(in)   // MUST NOT panic
        if err != nil {
            return
        }
        if _, err := dice.ParseExpression(e.String()); err != nil { // round trip
            t.Fatalf("String() output %q does not re-parse: %v", e.String(), err)
        }
    })
}
```

- The fuzz target asserts **properties** (never panics, round-trips, error is
  typed, result within bounds), not exact outputs.
- Seed corpus: `f.Add` for edge inputs plus files in `testdata/fuzz/<Name>/`.
  A crasher found by fuzzing is committed there as a regression test.
- Plain `go test` runs the seeds as ordinary tests. A real fuzz run is
  `go test -fuzz=FuzzParseExpression -fuzztime=30s ./dice`, done when a parser
  changes; CI MAY run a short fuzz pass.

### Examples

Public types and functions SHOULD have an `Example*` function in a
`package x_test` file. Examples are testable documentation: with an
`// Output:` comment they are compiled and verified by `go test`, and they
appear in `go doc`. Use a `dicetest` roller so the output is deterministic.
Name them `ExampleType`, `ExampleType_Method`, `ExampleFunc` (a suffix starting
with a lowercase letter names a variant).

### Commands

Run these from the repository root (Go and `golangci-lint` are installed
locally; Python and PHP are not involved):

```sh
go build ./...
go vet ./...
golangci-lint run
go test -race -shuffle=on -cover ./...                 # the full suite
go test -run 'TestCheck' -race ./d20               # narrow, while developing
go test -count=1 ./...                                 # bypass the test cache
```

- Run the narrowest set while developing, and the whole suite with `-race`,
  `-shuffle=on` and coverage before you finish.
- CI MUST run the whole suite with `-race -shuffle=on` and coverage, and a
  failing or flaky test blocks the change. Fix flakiness; never retry it away.
  Record the seed `go test -shuffle=on` prints so an order-dependent failure can
  be replayed with `-shuffle=<seed>`.
- Add `-count=N` when hunting a flaky test; do not commit that to CI.
- Benchmarks (`Benchmark*`) are optional and live beside the code; they are
  never the proof of correctness.

### Layout

```
dice/ ability/ modifier/ srderr/ telemetry/    # unit tests: x_test.go (external), x_internal_test.go (rare)
dice/dicetest/ telemetry/telemetrytest/        # doubles + exported contract suites
d20/                                           # unit + application tests, doubles injected
scenarios/                                     # cross-package, package scenarios_test, public API only
internal/archtest/                             # guardrails (§9)
<pkg>/testdata/                                # fixtures, goldens, fuzz corpora
```

---

## 11. Anti-patterns (prohibited)

- **Happy-path-only suites.**
- **Ice-cream cone:** mostly broad, slow tests with thin unit coverage.
- **Testing the mock:** tests that assert only on what a double was told to do.
- **Implementation-coupled tests** that break on refactors and not on bugs.
- **Tests with logic,** or assertions hidden in helpers that swallow failures.
- **Vague assertions** (`err != nil`, `got != nil`) on behaviour that has an
  exact expected value.
- **Matching error text** instead of `errors.Is` / `errors.As`.
- **Shared mutable fixtures,** package-level test state, and tests that depend
  on order.
- **Hidden non-determinism:** unseeded randomness, real time, real I/O,
  `time.Sleep`.
- **Skipped or commented-out tests** with no issue.
- **Uncited tests** where a spec or rule exists (applies to new tests; backfill pending, see §5).
- **Coverage theatre:** executing code without asserting anything.
- **Fixing a bug without a regression test.**
- **Leaving stale tests** that encode old behaviour.
- **Skipping the evaluation:** finishing a change without working out its tests,
  or reporting "covered by existing tests" without saying which.
- **Running `gomu` on the host,** or with workers or CPUs above the quarter cap.
- **Content from outside SRD 5.2.1** in tests, comments or fixtures.

---

## Checklist before submitting

- [ ] I evaluated what tests this change needs (§2) and wrote them.
- [ ] Each behaviour is tested at the lowest pyramid level that can prove it.
- [ ] Happy path covered for every distinct valid outcome.
- [ ] Applicable unhappy paths from §4 covered, with specific assertions.
- [ ] Every new test cites its requirement or rule (`Requirement:` and `Path:`;
      backfill of existing tests pending, §5), and every requirement in the
      spec I implemented has a test.
- [ ] Bug fix: regression test written first and seen to fail.
- [ ] Tests are deterministic, independent, fast, and use `dicetest` /
      `telemetrytest`.
- [ ] New parsers have a `Fuzz*` test; new public API has an `Example*`.
- [ ] New or changed adapters and doubles pass the port's contract suite.
- [ ] Each changed package has at least 90% statement coverage and meaningful
      assertions.
- [ ] Changed kernel and domain files are mutation tested (in Docker, capped at
      a quarter of the CPUs) with a score of at least 80, survivors handled
      (§8), and the score is in my `Testing` entry.
- [ ] Stale tests updated or removed; nothing skipped or commented out.
- [ ] The commit/PR and my final summary have a `Testing` entry.
- [ ] `go vet`, `golangci-lint run` and `go test -race -shuffle=on -cover ./...`
      pass, including the architecture tests.

---

## Read receipt

RULE-TAG: `testing-1525ed0d301e`

This tag proves you read this document in this session. Quote it verbatim as
required by `AGENTS.md`. Do not copy it anywhere else. Whoever edits this
document MUST replace the tag with a fresh random value.
