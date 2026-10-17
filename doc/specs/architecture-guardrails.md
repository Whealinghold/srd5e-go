# Architecture guardrails

| | |
|---|---|
| Status | partially implemented |
| Context | `package` |
| Roadmap phase | 0 |
| Depends on | `ability-scores.md`, `active-effects.md`, `attack-and-help-actions.md`, `attacks.md`, `casting-and-concentration.md`, `character-aggregate.md`, `character-creation.md`, `class-barbarian.md`, `class-bard.md`, `class-cleric.md`, `class-druid.md`, `class-fighter.md`, `class-monk.md`, `class-paladin.md`, `class-ranger.md`, `class-rogue.md`, `class-sorcerer.md`, `class-warlock.md`, `class-wizard.md`, `class-wizard-evoker.md`, `combat-action-resolution.md`, `combat-foundations.md`, `combatant-roster-changes.md`, `conditions.md`, `content-classes.md`, `content-classes-data.md`, `content-equipment.md`, `content-licensing-and-notice.md`, `content-magic-items.md`, `content-monsters.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `content-species-backgrounds-feats.md`, `content-spells.md`, `content-thin-slice.md`, `d20-tests.md`, `deployed-and-restraining-gear.md`, `derived-stats.md`, `dice-and-rolls.md`, `dispel-and-counterspell.md`, `encounter-aggregate.md`, `environment-hazards-and-objects.md`, `equipment-rules.md`, `errors-and-results.md`, `exploration.md`, `feats-behaviour.md`, `grappling.md`, `grid-positions-and-movement.md`, `heroic-inspiration.md`, `hiding-searching-and-senses.md`, `higher-level-start-equipment-and-trinkets.md`, `hit-points-and-damage.md`, `initiative-and-actions.md`, `item-use-effects.md`, `level-advancement-and-multiclassing.md`, `magic-item-effects.md`, `magic-item-rules.md`, `magical-effects.md`, `modifiers-and-advantage.md`, `monster-combatant-setup.md`, `monster-engulf-and-swallow.md`, `monster-limited-use.md`, `monster-spellcasting.md`, `monster-stat-block-actions.md`, `monster-traits.md`, `mounted-and-underwater-combat.md`, `movement-lifting-and-flight.md`, `observability-ports.md`, `persistent-spell-zones.md`, `readying-a-spell.md`, `resolve-d20-test-use-case.md`, `rests-and-hit-dice.md`, `returning-to-life.md`, `revival-and-restoration-spells.md`, `rule-options.md`, `shape-changing-spells.md`, `social-interaction.md`, `species-trait-behaviour.md`, `spell-effect-data.md`, `spell-effects.md`, `spell-granted-modifiers.md`, `spell-outcome-tables.md`, `spell-scroll-use.md`, `spellcasting.md`, `summoned-and-companion-creatures.md`, `telepathy.md`, `tool-proficiency-and-crafting.md`, `toolbox-combat-encounter-difficulty.md`, `toolbox-curses-and-contagions.md`, `toolbox-environmental-effects.md`, `toolbox-fear-and-mental-stress.md`, `toolbox-hazard-effects.md`, `toolbox-poison.md`, `toolbox-traps.md`, `toolbox-travel-pace.md`, `weapon-properties-and-mastery.md`, `wish-spell.md` |

## 1. Purpose and scope

The mechanical checks that make the rules in `doc/rules/` binding: every check
listed in `architecture.md` §9, `observability.md` §10 and `testing.md` §9, as
tests under `internal/archtest/`. Each check is a pure function over an
in-memory snapshot of the module (parsed with `go/parser`, standard library
only), so it can be tested against a violating fixture overlaid on a clean
module as well as run against the real module tree. The suite includes the
read-receipt tag check and the consistency check between
`doc/observability/signals.md` and the signal constants in code. It also
carries the mechanical decisions of `shared-types-and-ownership.md` (FR-013,
FR-052): the test-support and `internal/` import rules, the generic-variant
name scan, the `NOTICE.md` covered-paths check (FR-079; decisions FR-050), no
use case calling another (FR-097; decisions FR-044), the `toolbox` leaf context
(FR-098; decisions FR-045), and the spec checks: template shape, context map,
the token-only "Depends on" header (FR-100), the rank rule from table 4.11 and
the cycle check (FR-101, FR-089), phase, the content data specs' dependency
rule (FR-099; decisions FR-047, FR-067), citations, and the documentation index
(FR-102), section 4.7.

This spec is rank 90 and lists every spec it verifies as a header token
(decisions FR-061); other specs reference it in prose only.

The bootstrap rule (`architecture.md` §9) requires the first change that adds
production code to bring this suite with it. Checks for code that does not
exist yet (the `content/data` embed rules, the use-case rules) ship with
fixtures and pass vacuously on the real module until that code exists. Where a
check needs a file that does not exist yet, it takes the missing input as a
parameter and the real-module wiring is added with that file.

**Implementation status.** `internal/archtest` implements today: A1 to A13 of `architecture.md` §9 (rules `unlisted-package`,
`import-not-allowed`, `kernel-imports-context`, `test-support-isolation`,
`internal-import`, `stdlib-denied`, `third-party-import`, `go-mod-require`,
`init-function`, `package-level-var`, `hidden-effect`, `panic-discipline`,
`missing-doc`, `missing-package-doc`, `stutter`, `consumer-name`,
`import-cycle`, `port-use`, `variant-set`, `error-shape`, `notice-paths`, `embedded-data`, `rule-tag`, `docs-only`). Only the spec checks of 4.4 and 4.7 and the observability signal checks are specified here and not yet enforced; section 5 maps every check id to its rule and status, so the Status of this spec stays `partially implemented`.

**Out of scope:**

- No game rules: no SRD content and no public API.
- Process checks that cannot be automated: that the observability and testing
  evaluations were done, and that a rule file's tag was replaced when the file
  was edited (`observability.md` §10; `AGENTS.md`). Reviewers enforce them;
  detecting the latter needs git history, so it is a CI step outside the unit tests (section 9).
- Duplicate ownership of a type or event across specs: reviewed by doc
  (`shared-types-and-ownership.md` §8). The code-level duplicate (two exported
  declarations of one name) is a compile error in Go, so there is no check.
- Topic mechanics owned elsewhere (decisions FR-048): the `go:embed` layout of
  `content/data/` (`content-schema-and-decoder.md`), the `NOTICE.md` wording
  (`content-licensing-and-notice.md`), release packaging
  (`release-0-1-0.md`).
- Other specs' own check ids (`CLN-` in `content-licensing-and-notice.md`, E1 to
  E6 of `errors-and-results.md`, `rules-glossary-coverage.md`): not registered
  here (section 9).
- Lint rules already in `.golangci.yml` (`errcheck`, `govet`, `staticcheck`,
  `revive`, `errorlint`, `exhaustive`, `gocritic`, `gosec`, `godot`,
  `misspell`). The `forbidigo`-style overlap with FR-028 stays as defence in
  depth; there is no inline `//nolint` for an architecture rule (FR-009).
- Mutation testing of production code (`testing.md` §8): the guardrail code is
  test code; its self-tests (FR-005) play that role.

## 2. Source

n/a (package design). Derived from the rule files and
`doc/specs/shared-types-and-ownership.md`. "decisions" below means that file; its ids are cited as "decisions FR-n". A check that goes beyond the rule
text (for example FR-013, FR-015, FR-016, FR-032, FR-076, FR-082, FR-083, FR-084) says "extends <rule> (package design)" in its Source cell
(section 9).

- `doc/rules/architecture.md` §1 (dependency rule and allow-list, test
  support, no-ops), §2 (context map, kernel), §3 (purity), §4 (a use case MUST
  NOT call another), §5 (adapters, bundled content), §7 (doubles, contract
  suites), §9 (enforcement, bootstrap).
- `doc/rules/observability.md` §2 (catalogue), §3, §4 (ports, no-ops), §5
  (naming, constants, attribute values), §10 (enforcement).
- `doc/rules/testing.md` §5 (citation format), §6 (no skipped tests), §9
  (guardrail tests), §7 (contract suites).
- `doc/rules/go.md` (naming, doc comments, compatibility).
- `doc/specs/shared-types-and-ownership.md` FR-013 to FR-017, FR-021, FR-030 to
  FR-033, FR-036 to FR-038, FR-041, FR-044, FR-045, FR-047, FR-050, FR-052,
  FR-056, FR-061 and table 4.11; `doc/specs/_template.md`; `AGENTS.md` ("Read
  the rules first", "Hard constraints", "Directory structure").

## 3. Vocabulary

| Term | Type | Notes |
|---|---|---|
| Public package | directory | A directory with Go files that another module may import, such as `dice` or `d20`. There are no re-export files: the package is the published API (`architecture.md` §1). |
| Kernel package | package | `dice`, `ability`, `modifier`, `srderr`, `telemetry` (`architecture.md` §1). They are not contexts: FR-018 and FR-019 do not apply to them (FR-023 and FR-024 govern them). |
| Context package | package | A bounded context directly under the module root (`d20`, `content`, `character`, `combat`, `toolbox`), in the context map. A context may have sub-packages such as `content/data`. |
| Test-support package | package | A `<pkg>test` package (`dice/dicetest`, `telemetry/telemetrytest`): the only place for doubles and contract suites; imported only by `_test.go` files (decisions FR-038). |
| Adapter package | package | A package that adapts a port to a standard-library facility and may import a package the kernel must not (`telemetry/slogtelemetry`). Nothing but tests and the consumer imports it. |
| Embedded data package | package | `content/data`, the only package that holds `go:embed` and the embedded JSON files (decisions FR-030). |
| Entry package | package | The module root, `package srd5e` in `doc.go`: documentation only, imports nothing, is imported by nothing. |
| Signals file | file | A file named `signals.go` in a kernel or context package. The only place where signal and attribute names may be spelled as string literals. |
| Signal constant | declaration | A string `const` in a signals file whose identifier begins `Span`, `Metric` or `Log`; the prefix gives the kind (span, metric, log event). |
| Attribute constant | declaration | A string `const` in a signals file whose identifier begins `Attr`. |
| Attribute value constant | declaration | A string `const` in a signals file whose identifier begins `Value`; its doc comment names the attribute key in backticks (FR-085). |
| Module | value | The parsed tree (`archtest.Module`): module path, `go.mod` facts, and every package with its parsed files (`go/ast`, comments included). Immutable once loaded (FR-002, FR-010). |
| Policy | value | The allow-list table (`archtest.Policy`): for each module-relative directory its `Kind` (docs, kernel, context, support, adapter, tooling, tests) and the module packages its production files may import. |
| Use case type | declaration | An exported type with an exported `Run` or equivalent entry method that holds injected ports and emits telemetry (file names are free; `go.md` does not require a `_usecase.go` suffix) (`architecture.md` §4), other than a result or command type. |
| Spec | file | A `doc/specs/*.md` file whose name does not start with `_`; its header table has Status, Context, Roadmap phase and Depends on (FR-094). |
| Context map | table | Allowed upstream edges between contexts, the single source of truth in `architecture.md` §2 and encoded in `DefaultPolicy` (FR-022). |
| Moved-id row | table row | A requirement-table row kept so an id is never reused: its second cell begins with the word `Withdrawn` and a colon, then `moved to <spec>.md (<id>)` (shared-types-and-ownership.md FR-062). A citation of one may carry the suffix `(withdrawn)` in parentheses (FR-096). |
| Rank | integer | A spec's number in table 4.11 of the decisions spec. Only the order matters; the values have gaps (FR-101). |
| Documentation index | file | `doc/documentation_index.md`: a three-column table (`File`, `Summary`, `Read when`) with one row per file under `doc/`, the first cell one backticked path relative to `doc/` (FR-102). |
| Violation | value | One finding (`archtest.Violation`): rule name, module-relative file, 1-based line, message. A check is one FR id that yields Violations; one function may implement several (FR-005). |

## 4. Requirements

Stable ids, never renumbered; new ids sit where they belong, not in numeric
order. "Source" gives the rule file and section, or "package design". Check ids
are those in sections 4.1 (FR-007, FR-010), 4.2, 4.3, 4.4 and 4.7 (FR-005).
Every check has a clean fixture (happy) and a violating fixture (unhappy), so
most rows read `unhappy (<kind>)`. Ids whose original wording has no Go
counterpart are withdrawn in 4.8 with a pointer to the Go requirement that
replaces them.

### 4.1 Harness

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `archtest.Load(fsys)` MUST read `go.mod` at the root of the given file system and return an error naming the problem when `go.mod` is missing or has no `module` line. The real-module test loads `os.DirFS` of the module root (two directories above `internal/archtest`). | package design | unhappy (state error) |
| FR-002 | A Module MUST contain every Go file of the module: every `.go` file below the root, keyed by slash-separated module-relative path, grouped by directory, excluding directories named `testdata` or `vendor` and directories whose name starts with `.` or `_` (as the go tool does; named in the loader, FR-009). Test files (`_test.go`) are kept and flagged. Non-Go text files are read by the checks that name them (`NOTICE.md`, `AGENTS.md`, `doc/`, `go.mod`). `.github/`, `LICENSE` and `.golangci.yml` are therefore in scope for the checks that name them. | architecture.md §9; observability.md §10; AGENTS.md "Hard constraints" | happy |
| FR-105 | FR-002's skipped directories are extended by `.zcode/` **at the repository root only**: machine-owned agent/workflow tooling state (drafts and run journals that name the hosting organisation in their paths and contents), git-ignored and never part of the module. The exclusion joins the FR-002 named constants (FR-009); everything else in FR-002 is unchanged. (A directory starting with `.` is already skipped by the loader and by the consumer-name scan; `.zcode/` is named in the spec so the exemption is deliberate.) | package design (new id for an FR-002 extension found while coding Phase 1; FR-002 keeps its original wording) | happy |
| FR-003 | Each check MUST be a pure function from a Module (plus its Policy and constants) to a list of Violation, doing no I/O after `Load`. | architecture.md §9 | happy |
| FR-004 | A failing check MUST report every violation it finds, one per line as `<file>:<line>: <rule>: <message>` (`Violation.String`; `module:` when the rule is module-wide), not only the first. | package design | unhappy (multiple violations) |
| FR-005 | A check is one FR id that sits in section 4.2, 4.3, 4.4 or 4.7 (FR-013 to FR-056 as they appear there, plus FR-061, FR-062, FR-068, FR-074 to FR-086, FR-088 to FR-090, FR-092 and FR-095 to FR-104), plus FR-007 and FR-010; one function MAY implement several ids. Withdrawn rows (4.8), FR-087 (a support rule in section 4.6) and FR-091 (a restatement of FR-101; its violations are reported as FR-101) are not checks. Every check id MUST have a self-test pair: a minimal violating fixture asserting the exact rule and file, and a clean fixture asserting none. A meta-test MUST read the rule names from the `Rule*` constants of `archtest`, and fail unless each rule has a fixture in the table-driven violation test that raises it. A rule with no violating fixture is a defect. | testing.md §8 ("a test that cannot fail is a defect") | unhappy (failure) |
| FR-006 | Import extraction MUST read every import of every file, test files included, from the parsed AST (`File.Imports`), whatever its form: plain, aliased, dot (`.`), blank (`_`) or grouped; each is an edge. Import paths are matched against the module path (`Module.Local`) to give a module-relative package. | architecture.md §9 | happy |
| FR-007 | An import of a path that starts with the module path and resolves to no package directory in the Module MUST be a Violation (a typo must not hide an edge); it is reported as `unlisted-package` or `import-not-allowed` because no policy row lists it. | architecture.md §9 | unhappy (invalid input) |
| FR-008 | Checks MUST work on the AST: text inside comments and string literals MUST never match an import, call or declaration rule (no false positive for `// import "os"` or `"time.Now"`). Doc and comment checks (FR-049 to FR-054) read `ast.CommentGroup`s. | architecture.md §9 | unhappy (false positive) |
| FR-009 | The suite MUST provide no inline suppression: a `//nolint`, `// archtest:ignore` or similar comment MUST NOT change any check's result. The only exemptions are the named constants in `archtest` (`guardFile`, `stdlibExceptions`, `bannedCall`, `exemptFromNames`, the reserved attribute keys, the shared attribute value sets of FR-085, the double-name prefixes of FR-081, the port handles of FR-082, the covered paths of FR-079, the leaf contexts of FR-098, the rank-90 threshold of FR-101, the placeholder spec names of FR-096, the data spec names of FR-099, the documentation index path of FR-102, the skipped directories of FR-002). | architecture.md §9 ("MUST fail the build") | unhappy (rule interaction) |
| FR-010 | `Load` MUST return an error that names the file when a `.go` file cannot be read or parsed, and when a file in the Module cannot be decoded. The loader MUST NOT skip such a file silently. | package design | unhappy (failure) |
| FR-011 | Checks MUST be deterministic and independent: same Module gives the same violations sorted by file, line, rule (`archtest.Check` sorts); no network, clock, randomness or writes; passes under `go test -shuffle=on -race`. | testing.md §6, §10 | unhappy (determinism) |
| FR-012 | The real-module test MUST fail if the Module has fewer than five packages, so a wrong root cannot pass every check vacuously; and a second test MUST fail when a package directory exists that the Policy does not list (`TestPolicyCoversEveryPackage`). | package design | unhappy (empty case) |
| FR-059 | A test MUST fail (`t.Error` per violation) when `Check` returns a non-empty list, printing them per FR-004. | architecture.md §9 | unhappy (failure) |
| FR-060 | Import extraction MUST handle aliased, dot and blank imports and import groups, and an import split across lines. | architecture.md §9 | happy |
| FR-064 | Import extraction MUST include the imports of every file regardless of build constraints (`//go:build` tags, `_linux.go` suffixes): the loader parses all `.go` files, so an import behind a tag is still an edge. | architecture.md §9 | happy |
| FR-073 | Each real-module test MUST finish in under 10 seconds on CI (`go test -timeout` for the package is set accordingly). The suite parses the module once per test; no total budget is set for the whole suite, CI's own job duration is watched (section 7). | testing.md §6 | unhappy (boundary) |

### 4.2 Architecture checks (architecture.md §9)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | No non-test or test file may import a third-party package (an import path whose first element contains a dot and that is not the module), rule `third-party-import`; and no non-test file of a kernel or context package may import `os`, `os/exec`, `net`, `net/...`, `io/ioutil`, `reflect`, `unsafe`, `syscall`, `log`, `math/rand`, `math/rand/v2`, `sync` or `sync/atomic`, rule `stdlib-denied`; the only exception is `math/rand/v2` in package `dice` (`stdlibExceptions`); `sync` and `sync/atomic` are denied everywhere until a mutex-guarded roller is specified. | architecture.md §9 A3; §3 Purity; AGENTS.md "Hard constraints" | unhappy (invalid input) |
| FR-062 | The entry package (module root, `doc.go`) MUST import nothing from the module or elsewhere beyond what its Policy row allows (nothing), and MUST contain documentation only. | architecture.md §7, §9 A1; AGENTS.md | unhappy (invalid input) |
| FR-014 | An adapter package (a package whose Kind is adapter, such as `telemetry/slogtelemetry`) MUST NOT be imported by a non-test file of any kernel, context or support package, and no package may import it unless it is listed in that package's Policy row. Adapters are reached only by a consumer or a test. | architecture.md §9 A1; §5 | unhappy (rule interaction) |
| FR-015 | A kernel package MUST NOT import a context package (rule `kernel-imports-context`), and a package outside the Policy MUST NOT exist. This is the Go form of the domain-purity direction rule: Go has one package per context, so there is no `domain/` to `application/` folder rule (withdrawn, 4.8). | architecture.md §9 A1; §1; extends §3 Purity (package design) | unhappy (invalid input) |
| FR-016 | A non-test file in a kernel or context package MUST NOT contain a `go` statement, a call to `time.Sleep`, `time.After` or `time.Tick`, or an import of `sync` or `sync/atomic` (no package is exempt today) (rule `hidden-effect` and `stdlib-denied`): domain code spawns no goroutines and does not wait. Channel types in a signature are not flagged (the AST check cannot judge them; review does). | architecture.md §9 A4; §3 Purity; extends it (package design) | unhappy (invalid input) |
| FR-017 | A non-test file MUST NOT import an adapter package unless its Policy row lists it; in particular a port package (`telemetry`, `dice`) MUST NOT import its own adapters (`telemetry/slogtelemetry`, `dice.SeededRoller` lives in `dice` itself and needs no import). | architecture.md §1, §9 A1; shared-types-and-ownership.md FR-037 | unhappy (invalid input) |
| FR-018 | A non-test file in context X importing a package of context Y (Y not X) MUST import the public package `Y` itself, never a sub-package of Y (such as `Y/data`) or an `internal/` package; any other import is a Violation. Kernel packages are not contexts; FR-023 and FR-024 govern them. | architecture.md §9 A1 ("a context importing"); §2 | unhappy (invalid input) |
| FR-019 | A non-test file in context X importing context Y MUST have Y in X's `Allow` set in the Policy (the encoding of the context map); otherwise Violation `import-not-allowed`. | architecture.md §9 A1; §2 | unhappy (rule interaction) |
| FR-020 | The package import graph of non-test files MUST be acyclic, and a self-edge between distinct packages counts as a cycle. A Violation (`import-cycle`) names the packages on the cycle. | architecture.md §9 A11; §2 | unhappy (invalid input) |
| FR-021 | Every package directory MUST appear in the Policy (rule `unlisted-package`), whether kernel, context, support, adapter, tooling or tests. A Policy entry with no directory is allowed (planned contexts: `content`, `character`, `combat`, `toolbox`). | architecture.md §1, §9 A11; shared-types-and-ownership.md FR-038 | unhappy (missing content) |
| FR-022 | The Policy MUST equal the tables of `architecture.md` §1 (package allow-list) and §2 (context map): the table with header cells `Package`, `MAY import (module packages)`, `MUST NOT import` and the one with `Context`, `Purpose`, `May depend on (upstream)` are parsed (the first cell one backticked package, the import cell `nothing` or backticked names separated by commas) and compared with `DefaultPolicy`; a missing table, another header, a cell outside the format, or a difference is a Violation. The `AGENTS.md` "Current contexts" list MUST name exactly the same contexts. Not yet enforced (section 5). | architecture.md §2 ("update this table, AGENTS.md and the architecture tests in the same change") | unhappy (state error) |
| FR-023 | A non-test file in a kernel package MUST import only the module packages its Policy row lists (`modifier` and `dice` import `srderr`; `ability` imports `srderr`; `srderr` and `telemetry` import none) and never the entry package or `internal/`. | architecture.md §9 A1; §2 | unhappy (invalid input) |
| FR-024 | A non-test file in a test-support package (`dice/dicetest`, `telemetry/telemetrytest`) or an adapter package MUST import only its own port package and the kernel packages its Policy row lists (`dicetest`: `dice`, `srderr`; `telemetrytest`: `telemetry`; `slogtelemetry`: `telemetry`), never a context or another test-support package. | architecture.md §9 A1/A2; §1; shared-types-and-ownership.md FR-036, FR-037 | unhappy (invalid input) |
| FR-061 | No file may import the entry package (the module root, `package srd5e`). It is documentation only, so such an import adds nothing and bypasses FR-015, FR-018 and FR-019. | architecture.md §2, §9 A1 | unhappy (rule interaction) |
| FR-025 | No file of the module (path, comment, code or text) may contain, case-insensitively, the name of a consuming project or the hosting organisation from the denylist, except as part of the module path itself (rule `consumer-name`; the occurrences of the module path are removed before matching). The denylist is assembled from fragments at run time (FR-068). | architecture.md §9 A10; AGENTS.md "Hard constraints" | unhappy (invalid input) |
| FR-026 | The scan of FR-025 covers every non-hidden file of the repository (Go files, `doc/`, `.github/`, root files) except the exempt paths, kept as named constants (FR-009): `AGENTS.md`, which states the rule and so must name its subject, `go.sum`, and `go.mod` (whose `module` line is the module path); directories starting with `.` are skipped. | AGENTS.md "Hard constraints" | unhappy (invalid input) |
| FR-068 | The denylist's own source file (`internal/archtest/names.go`) MUST NOT contain a denylist name in plain text; it assembles the names from fragments at run time. The same holds for the trademark and affiliate lists that `content-licensing-and-notice.md` FR-031 keeps in `internal/archtest`: the plain-text self-test covers every list there. | AGENTS.md "Hard constraints" | unhappy (invalid input) |
| FR-027 | A non-test file in a kernel or context package MUST NOT call `time.Now`, `time.Since`, `time.Sleep`, `time.After` or `time.Tick`, nor import `math/rand` or `math/rand/v2` (except `dice`, for the caller-supplied `*rand.Rand` type, which it never constructs: FR-103). Taking `time.Now` as a value (`WithClock(time.Now)`) is not a call and is allowed. | architecture.md §9 A3/A4; §3 Purity | unhappy (invalid input) |
| FR-028 | No non-test file may call `fmt.Print`, `fmt.Printf`, `fmt.Println` (any `fmt.Print*`), the builtins `print` or `println`, or `os.Exit` (rule `hidden-effect`). A method named `Print` on another object is allowed. | architecture.md §9 A4; observability.md §10 | unhappy (invalid input) |
| FR-029 | Each `doc/rules/*.md` file MUST have exactly one line starting with `RULE-TAG:`, whose value is a backticked token `<file stem>-<12 lowercase hex digits>` (rule A13). Zero or two such lines is a Violation. Enforced: rule `rule-tag`, `internal/archtest/ruletags.go`. The check is vacuous when `doc/rules/` holds no Markdown file; `TestRealModuleHasRuleFiles` pins that the real module has four. Tag lines in `doc/specs/` files are outside this check. | architecture.md §9 A13; AGENTS.md | unhappy (boundary) |
| FR-030 | A rule tag's value MUST NOT appear in any other file of the repository (`.github/`, `CLAUDE.md` and `internal/` are covered), nor on another line of its own file; two rule files sharing a value is a Violation in both. Enforced: rule `rule-tag`, `internal/archtest/ruletags.go`. The check is vacuous when `doc/rules/` holds no Markdown file; `TestRealModuleHasRuleFiles` pins that the real module has four. Tag lines in `doc/specs/` files are outside this check. | architecture.md §9 A13 | unhappy (invalid input) |
| FR-031 | Every `doc/rules/*.md` file MUST be listed in the `AGENTS.md` "Read the rules first" list, and every path listed there MUST exist. Enforced: rule `rule-tag`, `internal/archtest/ruletags.go`. The check is vacuous when `doc/rules/` holds no Markdown file; `TestRealModuleHasRuleFiles` pins that the real module has four. Tag lines in `doc/specs/` files are outside this check. | AGENTS.md ("new rule files go on this list") | unhappy (missing content) |
| FR-074 | No non-test file may import a test-support package (`dice/dicetest`, `telemetry/telemetrytest`); only `_test.go` files may (rule `test-support-isolation`, A2). | architecture.md §1, §9 A2; observability.md §10, last bullet; shared-types-and-ownership.md FR-038 | unhappy (rule interaction) |
| FR-075 | A non-test file in a test-support package MAY import only its own port package and the kernel packages its Policy row lists (FR-024). It MUST NOT import a context, an adapter, the entry package, `internal/` or another test-support package. | architecture.md §1 (test support rows), §9 A2; shared-types-and-ownership.md FR-038 | unhappy (invalid input) |
| FR-076 | No file may import an `internal/` package other than the package's own (rule `internal-import`): `internal/archtest` is executed by `go test` and never imported. Additionally, a non-test file outside a test-support package MUST NOT import `testing` (dev-only: contract suites live in `*test` packages, so `testing` must not become a runtime dependency of production code). The `internal/` rule is enforced; the `testing` rule is not yet. | architecture.md §5, §7, §9 A2; extends it (package design); shared-types-and-ownership.md FR-032, FR-039 | unhappy (invalid input) |
| FR-077 | Variant-name scan (rule A7). (a) An exported top-level type or constant named `Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found` or `NotFound` in a context package is a Violation. (b) Withdrawn for Go: two declarations of the same name in one package do not compile, and distinct packages may reuse a name. A `Signals`-style constants file legitimately exports its constants; the scan does not flag them. Enforced as rule `variant-set` (`variants.go`). The type or const is looked for at top level, grouped or not. The same rule enforces `errors-and-results.md` FR-044: a sealed interface is any interface with at least one unexported method (broader than "only markers", so `d20.Result`, which also has exported methods, is covered), and its doc comment must name each implementer as a whole word; embedding-promoted methods are a blind spot. | architecture.md §4, §9 A7; shared-types-and-ownership.md FR-017, FR-021 | unhappy (invalid input) |
| FR-079 | Notice paths (rule A12; rule name `notice-paths`, kebab-case like every other rule; owner: `shared-types-and-ownership.md` FR-050; `content-licensing-and-notice.md` defines the list and the Scope wording, FR-005 there). (a) While bundled content is present (the module holds a file under `content/data/` other than `.gitkeep` and `README.md`), the set of backticked tokens in the "Scope" section of `NOTICE.md` that match `^[A-Za-z0-9_.-]+(/[A-Za-z0-9_.-]+)*/$` (a path ending in `/`) MUST equal the named constant `noticeCoveredPaths` = {`content/data/`}; a missing or extra path is a Violation naming it, at the line of the token (or of the heading when missing). A missing `NOTICE.md`, a missing "Scope" section and the statement wording are not this check: they belong to `content-licensing-and-notice.md` FR-002 and FR-005. (b) Withdrawn for Go: there is no `.pubignore`; the Go module proxy serves the whole repository, so excluding `doc/research/` from distribution is not possible and is dropped (see 4.8). (c) A self-test fails when the constant is not equal to the set of backticked tokens, by the token rule of (a), in the first sentence of the FR-033 row of `shared-types-and-ownership.md`, so the two cannot drift. Later sentences of that row, such as the backticked `doc/research/` in the one about notes that are not distributed, are not part of the set. Also, when `content/data/` exists it MUST hold only `.json` files and the Go file holding `go:embed`. Enforced (`embedded.go`, rules `notice-paths` and `embedded-data`). Resolved: `.gitkeep`, `README.md` and `_test.go` files are allowed in `content/data/`, and the whole check is vacuous while the directory is absent or holds only those. The layout rule needs exactly one non-test Go file, and it must contain `//go:embed`. The (c) self-test is `TestNoticeCoveredPathsMatchDecisionsSpec`. | shared-types-and-ownership.md FR-033, FR-050; architecture.md §9 A12; `AGENTS.md`, "The notice" | unhappy (state error) |
| FR-081 | An exported type in a non-test package that is not a test-support package, whose name starts with one of the double prefixes (`Fake`, `Recording`, `Fixed`, `Scripted`, `Stub`, `Spy`, `Mock`; a named constant, FR-009) MUST be declared in a test-support package instead. A `Noop*` type is not a double (FR-084). The import side is FR-074. Not yet enforced. | observability.md §10, last bullet; architecture.md §7; shared-types-and-ownership.md FR-038 | unhappy (invalid input) |
| FR-082 | Every driven port (an exported interface declared in a kernel or context package that the package's own functions take as a dependency, such as `dice.Roller`, `telemetry.Logger`, `telemetry.Tracer`, `telemetry.Meter`, and whose name is not in the named constant `portHandles`, FR-009) MUST have (a) at least one type in the matching test-support package that implements it (`dicetest.FixedRoller`, `telemetrytest.Logger`), and (b) an exported contract suite function `Run<Port>Contract` in that package (`dicetest.RunRollerContract`, `telemetrytest.RunLoggerContract`). Both are vacuous when no port exists. `portHandles` is `{Span, Counter, Histogram, Gauge}`: handles returned by ports, not ports, so they need no double or contract function of their own (`observability-ports.md` FR-065 requests the exemption). A consumer-side interface a context declares for what it needs (`d20.Roller`) is not a port of this module and is exempt: it is satisfied by `dice.Roller`. `release-0-1-0.md` FR-084 takes its port definition from this requirement; it must also apply `portHandles` (a change request to that spec, section 9). Not yet enforced. | extends architecture.md §7 (package design: the `Run<Port>Contract` naming and the handle exemption); architecture.md §9 (Bootstrap), Checklist; observability.md §4 (recording double); testing.md §7; shared-types-and-ownership.md FR-038, FR-039 | unhappy (missing content) |
| FR-083 | `go.mod` MUST have no `require` directive (rule `go-mod-require`): the module depends on the standard library only. The rule is the allowlist; changing it is the "discussion" `AGENTS.md` requires, made in the same change as the new dependency. FR-033 stays as the observability-specific statement of the same rule. `release-0-1-0.md` FR-060 restates the rule and cites this check. | AGENTS.md "Hard constraints"; extends observability.md §10, bullet 1 (package design) | unhappy (invalid input) |
| FR-095 | The embedded data package `content/data` MUST be the only package that imports `embed`, MUST import no module package except `content` (to implement `content.Source`) and the kernel packages its Policy row lists, and MAY be imported only by `_test.go` files and by consumers (never by a context). One decode path: embedded strings reach the domain only through the decoder. Enforced (rule `embedded-data` for `embed`; `import-not-allowed` through the `content/data` Policy row for the rest: adapter kind, allows `content` and the kernel). The package does not exist yet, so the real-module run is vacuous; the fixtures in `guardrails_test.go` prove each edge. Its `//go:embed` var is the one package-level var A4 lets through besides sentinels. | architecture.md §5; shared-types-and-ownership.md FR-030, FR-031 | unhappy (rule interaction) |
| FR-097 | No use case calls another use case. A use-case file (one declaring a use case type) MUST NOT mention, as an identifier in the AST, a use case type declared in a different file, whether of its own context or another context's published API. A use case composes the pure domain services that upstream packages publish (for example pure helpers such as `d20` rules and roll modifiers; `d20.Resolver` is itself the use case of `resolve-d20-test-use-case.md`, not a domain service) and takes dice and content through its own ports; nesting a use case would hide telemetry and count a signal twice. The use case set is derived from the source (types with an entry method), not listed. Test files and test-support packages MAY mention use cases. Not yet enforced. | architecture.md §2, §4; shared-types-and-ownership.md FR-044 | unhappy (rule interaction) |
| FR-098 | Shape of the context map in the Policy, for the leaf contexts in the named constant `leafContexts` = {`toolbox`}: (a) no package's `Allow` set may contain a leaf context (nothing depends on `toolbox`); (b) a leaf context present in the Policy MUST list only packages that are themselves in the Policy (no unknown name); (c) the leaf context's directory `toolbox` is a context package like any other (FR-021) and its files obey FR-018 and FR-019 in the import legality cases of section 4.5. The Policy, not a copy, stays the source of truth: the constant only names which entries are leaves, and a self-test fails when `shared-types-and-ownership.md` no longer contains `toolbox` in its FR-045 row. A context not in `leafContexts` is not constrained here, so a future leaf is a constant edit in the change that adds it. Not yet enforced as a dedicated test. | architecture.md §2 (context map, "Nothing depends on `toolbox`"); shared-types-and-ownership.md FR-045 | unhappy (rule interaction) |
| FR-103 | No non-test file in a kernel package, test-support package or adapter package, including `dice.SeededRoller` and the no-ops, may import `math/rand` or call `rand.New`, `rand.NewSource`, `rand.Seed`, `time.Now`, `time.Since` or `rand.`-package top-level functions: the seeded `*rand.Rand` is supplied by the caller (`dice-and-rolls.md`) and only consumer adapters read the clock. `dice` is the single package that may import `math/rand/v2`, for the `*rand.Rand` type in `SeededRoller`'s signature. FR-027 covers contexts; this extends the same ban to the kernel and support packages with one named carve-out: `dice/randroller`, the opt-in unseeded roller, may call the package-level `math/rand/v2` functions (`dice-and-rolls.md` FR-088 to FR-092). `print` stays FR-028's. The import ban is enforced for kernel and context packages only (rule A3 does not scan test-support or adapter packages, so `telemetrytest` may import `sync` today); the extension to those packages and the `rand.New` call ban are not yet enforced. | observability-ports.md FR-053; architecture.md §3, §9 A3/A4; observability.md §4 "Timing"; extends FR-027 (package design) | unhappy (determinism) |
| FR-104 | In package `telemetry`, `Noop`, `NoopLogger`, `NoopTracer` and `NoopMeter` MUST each be a struct type declared in that package, and `Noop` MUST embed the three; each MUST be usable as its zero value (no exported or unexported field that needs initialising, no constructor required) with only value receivers, and a compile-time assertion `var _ Logger = Noop{}` (and for `Tracer`, `Meter`) MUST exist. Go has no `final`, so a consumer may embed them; the guarantee is statelessness, not sealing. Not yet enforced as a check (the package carries the assertions). | observability-ports.md FR-062; shared-types-and-ownership.md FR-036; observability.md §4 | unhappy (invalid input) |

### 4.3 Observability checks (observability.md §10)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | No file may import an observability or error-tracking package: any third-party import is rule `third-party-import` (so `go.opentelemetry.io/...`, `github.com/getsentry/...`, `go.uber.org/zap`, `github.com/sirupsen/logrus` and the like fail), and the standard `log` package is rule `stdlib-denied` in kernel and context packages. `log/slog` MAY be imported only by `telemetry/slogtelemetry`; that restriction is not yet enforced. | observability.md §10, bullet 1; §4; the `log` and `log/slog` restriction extends it (package design) | unhappy (invalid input) |
| FR-033 | `go.mod` MUST NOT list any observability or error-tracking package; since FR-083 forbids every `require`, this is covered by rule `go-mod-require`. | observability.md §10, bullet 1 | unhappy (invalid input) |
| FR-034 | A non-test file in a context package that declares no use case type and is not `options.go` MUST NOT import `telemetry` nor mention an identifier that is, or ends in a CamelCase word, `Logger`, `Tracer`, `Meter` or `Telemetry`, nor a variable named `logger`, `tracer`, `meter` or `telemetry`. Domain code stays silent (`observability.md` §3). Not yet enforced. | observability.md §10, bullet 2; §3 | unhappy (invalid input) |
| FR-035 | A package-level `var` whose declared type or initialiser names a `Logger`, `Tracer`, `Meter` or `Telemetry` type MUST be a Violation. The general package-level `var` ban (rule `package-level-var`, A4) already forbids every package-level `var` other than `_` assertions and `Err*` sentinels, so this is enforced by it; the zero-value `Noop` is used through a struct field or option, never a global. | observability.md §10, bullet 4; §4 | unhappy (invalid input) |
| FR-036 | A string literal whose value begins with `srd5e.` or equals a reserved attribute key (`error.type`) MUST appear only in a signals file (`signals.go`), never at a call site, in a domain file, or built by concatenation (`"srd5e." + x`). Test files MAY spell them (FR-072). Not yet enforced. | observability.md §10, bullet 5; §5 Naming | unhappy (invalid input) |
| FR-037 | In a signals file, every `srd5e.` literal MUST be the initialiser of a string `const` whose identifier begins `Span`, `Metric`, `Log`, `Attr` or `Value` (an attribute value constant, FR-085). The shared keys in `telemetry/signals.go` are exempt from the context-name rule of FR-038 (they are attribute keys: `observability-ports.md` FR-052). A `signals.go` file MUST be in a kernel or context package, and a package-level string `const` in a signals file with no such prefix is a Violation. Not yet enforced. | observability.md §5 Naming ("defined in one place per context") | unhappy (invalid input) |
| FR-038 | A `Metric` or `Span` signal constant's value MUST match `^srd5e\.[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`, and its second segment MUST be the context name of the defining package (the package name with its inner underscores removed equals the segment with underscores removed: package `d20` owns `srd5e.d20_test.resolve`; the published names are a telemetry contract and are kept). A `Log` constant is the constant log message (the Logger has no event-name field): its value MUST match `^[a-z][a-z0-9]*( [a-z0-9]+)*$` and has no context segment. A signal constant of any kind in `telemetry` is a Violation (shared constants are the attribute keys). The rule is about shape only: the check names no use case, and which spec owns a given signal is settled by `shared-types-and-ownership.md` FR-040 (one authority per signal; FR-040 here catches a duplicate value). Not yet enforced. | observability.md §5 Naming, §7; observability-ports.md FR-002; shared-types-and-ownership.md FR-040, FR-043 | unhappy (boundary) |
| FR-039 | An attribute constant's value (an `Attr*` constant in a signals file) MUST match `^srd5e\.[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$` or be a reserved key (`error.type`). Not yet enforced. | observability.md §5 Attributes | unhappy (boundary) |
| FR-040 | No two signal constants anywhere in the module may have the same value. Not yet enforced. | observability.md §10, bullet 6 | unhappy (invalid input) |
| FR-041 | `doc/observability/signals.md` MUST contain a table whose header cells are, in order, Name, Type, Unit, Attributes, Emitted by, Purpose, Suggested monitoring; a row with a different cell count is a Violation. Zero data rows is valid. Backticks around a Name cell are stripped. Not yet enforced. | observability.md §2 (catalogue fields) | unhappy (invalid input) |
| FR-042 | Every signal constant's value MUST appear as a Name in the catalogue. Not yet enforced. | observability.md §10, bullet 6 | unhappy (missing content) |
| FR-043 | Every catalogue Name MUST equal some signal constant's value. Not yet enforced. | observability.md §10, bullet 6 | unhappy (missing content) |
| FR-044 | Catalogue Names MUST be unique. Not yet enforced. | observability.md §2 ("single list") | unhappy (invalid input) |
| FR-045 | A catalogue row's Purpose MUST NOT be empty or a placeholder (`TODO`, `TBD`, `N/A`, `-`, case-insensitive). Not yet enforced. | observability.md §10, bullet 7 | unhappy (boundary) |
| FR-046 | A catalogue row's Suggested monitoring MUST NOT be empty or a placeholder (same set as FR-045). Not yet enforced. | observability.md §10, bullet 7; §2 | unhappy (boundary) |
| FR-047 | A catalogue row's Type MUST start, case-insensitively, with `counter`, `histogram`, `gauge`, `span` or `log event`, and trailing text after that word is ignored. It MUST agree with the constant-name prefix in the signals file (`Metric...` is counter, histogram or gauge; `Span...` is span; `Log...` is log event). The unit is the Unit column's content, never part of Type. The conversion from the template's five-column signal table (`_template.md` section 7: Signal, Type, Attributes, Purpose, Suggested monitoring) to these seven catalogue columns happens when a row is copied into `doc/observability/signals.md`: Unit and Emitted by are added then, not in the specs. Not yet enforced. | observability.md §2 (catalogue fields) | unhappy (invalid input) |
| FR-048 | Every attribute constant's value (an `Attr*` constant, FR-039) MUST appear in backticks in the Attributes cell of at least one catalogue row, and every backticked `srd5e.`-prefixed or reserved key in an Attributes cell MUST be an attribute constant's value. Not yet enforced. | observability.md §5 Attributes; §2 | unhappy (missing content) |
| FR-084 | Every exported type whose name starts with `Noop` MUST be declared in package `telemetry` and MUST be usable as its zero value (FR-104). A `Noop*` type anywhere else, or one with a constructor that is required before use, is a Violation. Not yet enforced. | extends observability.md §4, §10 (no-op bullet) to every `Noop*` type, not only the observability ones (package design); architecture.md §1; shared-types-and-ownership.md FR-036 | unhappy (invalid input) |
| FR-085 | Shared attribute values. The closed sets are named constants in the check code (FR-009): `srd5e.outcome` is `success`, `failure`; `srd5e.d20_test.kind` is `ability_check`, `saving_throw`, `attack_roll`; `srd5e.d20_test.roll_mode` is `normal`, `advantage`, `disadvantage`; `srd5e.d20_test.critical_hit` is `true`, `false`. (a) In code, an attribute value constant (a string `const` in a signals file whose identifier begins `Value`) MUST have a doc comment that names its attribute key in backticks; if the key is one of the four above its value MUST be in that key's set; if the key is `error.type` its value MUST equal `<port>_error` for a port name constant (`dice.Port` and the like) and MUST match `^[a-z][a-z0-9_]*$`. (b) In the catalogue, a shared key in an Attributes cell lists its values as backticked tokens in parentheses straight after it, for example `` `srd5e.outcome` (`success`, `failure`) ``; each listed value MUST be in the key's set (for `error.type`, as in (a)). (c) A self-test fails when any set value is absent, in backticks, from `observability.md` §5, so the two cannot drift. Not yet enforced. | observability.md §5 Attributes, §10 (shared attribute value bullet); shared-types-and-ownership.md FR-041 | unhappy (boundary) |

### 4.4 Test-citation checks (testing.md §9)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-049 | Every top-level `Test*` function in `_test.go` files, outside `internal/archtest/`, MUST be immediately preceded (no blank line between) by a `//` comment block containing a line starting `Requirement:` or `Rule:`, unless FR-051 applies. `Example*` and `Fuzz*` functions are exempt (`testing.md` §9). Not yet enforced. | testing.md §9 | unhappy (invalid input) |
| FR-050 | `Requirement: none - <reason>` MUST have a non-empty reason after the dash; a bare `Requirement: none`, or an empty `Requirement:`, is a Violation. Not yet enforced. | testing.md §5, §9 | unhappy (boundary) |
| FR-051 | A `t.Run` call inside a loop (FR-066) is satisfied by the comment of the enclosing `Test*` function that carries `Requirement: per case`. That comment MUST contain `Requirement: per case`. Not yet enforced. | testing.md §5, §9 | unhappy (rule interaction) |
| FR-052 | `Requirement: per case` MUST appear only on the comment of a `Test*` function whose body contains a loop (FR-066) with at least one `t.Run` call. Not yet enforced. | testing.md §5 | unhappy (state error) |
| FR-053 | Each comment with a `Requirement:` line (including `Requirement: none - ...`) MUST contain a `Path:` line whose value is `happy`, `unhappy (<kind>)` with a non-empty kind, or `per case`; `per case` is valid only with `Requirement: per case`. A comment with only `Rule:` need not carry `Path:` (decided, section 9). Not yet enforced. | testing.md §5 (format) | unhappy (invalid input) |
| FR-054 | A citation (FR-070) MUST name an existing spec file and ids present in that spec's requirements table (FR-071). A range (`FR-1 to FR-3`, `FR-1-FR-3`, `FR-1..3`) is a Violation, and so is a moved id (FR-087). Not yet enforced. | testing.md §5 ("keep citations true") | unhappy (missing content) |
| FR-055 | For every spec whose Status is `implemented`, every FR id in its requirements table MUST be cited by at least one test, except a moved id (FR-087) and an id its section 8 lists on a line `Deliberately not covered: FR-<n> - <reason>` with a non-empty reason (one id per line; `none` lists nothing). Not yet enforced. | testing.md §5 ("every requirement SHOULD be covered"); doc/specs/_template.md §8 | unhappy (missing content) |
| FR-056 | A spec's Status value MUST be one of `draft`, `accepted`, `partially implemented`, `implemented` (the template's status values). Not yet enforced. | doc/specs/_template.md | unhappy (invalid input) |
| FR-086 | No `_test.go` file may contain a comment line that is a commented-out test (a comment whose text, after `//`, starts with `func Test`, `t.Run(` or `t.Parallel(`; this check reads the AST comment groups), and a `t.Skip`, `t.Skipf` or `t.SkipNow` call is allowed only when the comment above the test function carries an `Issue:` line with a non-empty value (a link or id and a reason). Not yet enforced. | testing.md §6 ("No skipped or commented-out tests"), §11; the `Issue:` field extends it (package design) | unhappy (invalid input) |

### 4.5 Import legality cases

A `;` in the Importer and Imports cells pairs them by position; each pair is a separate test case. Each row is one table-driven self-test (an `fstest.MapFS` overlay on the clean module) for FR-013 to FR-019, FR-023, FR-024,
FR-027, FR-032, FR-061, FR-062, FR-074 to FR-076, FR-095, FR-097, FR-098 and FR-103.
"Violates" lists the FR ids expected; `none` means the import is allowed.
FR-020 to FR-022 and FR-007 have their own fixtures (section 8). Importers and
imports are module-relative package directories; "(test)" means a `_test.go`
file of that package.

| Label | Importer | Imports | Violates | Source | Path |
|---|---|---|---|---|---|
| kernel to kernel | `ability`; `dice` | `modifier`, `srderr`; `srderr` | none | architecture.md §1 | happy |
| context to kernel | `d20` | `dice`, `ability`, `modifier`, `srderr`, `telemetry` | none | architecture.md §1 | happy |
| context to upstream public package | `character` | `content`, `d20` | none | architecture.md §2 | happy |
| combat to every upstream | `combat` | `character`, `d20`, `content` | none | architecture.md §2 | happy |
| toolbox to every upstream | `toolbox` | `content`, `d20`, `character`, `combat` | none | architecture.md §2 | happy |
| adapter to its port | `telemetry/slogtelemetry` | `telemetry` | none | architecture.md §1 | happy |
| double to port and kernel | `dice/dicetest`; `telemetry/telemetrytest` | `dice`, `srderr`; `telemetry` | none | architecture.md §1 | happy |
| test file imports a double | `d20` (test) | `dice/dicetest` | none | architecture.md §1 | happy |
| external test imports its own package | `d20` (test, `package d20_test`) | `d20` | none | go.md §6 | happy |
| scenarios import every public package | `scenarios` (test) | `dice/dicetest`, `telemetry/telemetrytest`, `d20`, `ability` | none | architecture.md §1 | happy |
| seeded randomness type in dice | `dice` | `math/rand/v2` | none | architecture.md §3 | happy |
| use case uses a domain service | `combat` use-case file mentioning a pure `d20` rule function (a domain service, not a use case type) | `d20` | none | architecture.md §4 | happy |
| kernel imports a context | `dice` | `d20` | FR-015 | architecture.md §1 | unhappy (invalid input) |
| kernel imports an unlisted kernel package | `dice` | `ability` | FR-023 | architecture.md §1 | unhappy (invalid input) |
| port package imports the module | `telemetry` | `srderr` | FR-023 | architecture.md §1 | unhappy (invalid input) |
| adapter or double imports a context | `telemetry/slogtelemetry`; `dice/dicetest` | `d20` | FR-024, FR-075 | architecture.md §1 | unhappy (invalid input) |
| double imports another double | `dice/dicetest` | `telemetry/telemetrytest` | FR-075 | architecture.md §1 | unhappy (invalid input) |
| double imports an adapter | `telemetry/telemetrytest` | `telemetry/slogtelemetry` | FR-075, FR-014 | architecture.md §1 | unhappy (invalid input) |
| production file imports a double | `d20` | `dice/dicetest` | FR-074 | architecture.md §9 | unhappy (rule interaction) |
| context imports an adapter | `d20` | `telemetry/slogtelemetry` | FR-014, FR-017 | architecture.md §9 | unhappy (invalid input) |
| port package imports its adapter | `telemetry` | `telemetry/slogtelemetry` | FR-017, FR-023 | architecture.md §1 | unhappy (invalid input) |
| context imports a sub-package of another | `character` | `content/data` | FR-018, FR-095 | architecture.md §2 | unhappy (invalid input) |
| non-upstream context | `character` | `combat` | FR-019 | architecture.md §2 | unhappy (rule interaction) |
| upstream context imports downstream | `d20` | `character` | FR-019 | architecture.md §2 | unhappy (rule interaction) |
| root context has no upstream | `content` | `d20` | FR-019 | architecture.md §2 | unhappy (rule interaction) |
| downstream of toolbox | `combat` | `toolbox` | FR-019, FR-098 | architecture.md §2 | unhappy (rule interaction) |
| embedded data imports a context other than content | `content/data` | `character` | FR-095 | architecture.md §5 | unhappy (rule interaction) |
| use case calls a use case | `toolbox` use-case file mentioning `d20.Resolve` where it is a use case type | `d20` | FR-097 | architecture.md §4 | unhappy (rule interaction) |
| import of the entry package | `d20`; `dice` | module root | FR-061 | architecture.md §2, §9 | unhappy (rule interaction) |
| entry package imports anything | module root `doc.go` | `dice` | FR-062 | architecture.md §7 | unhappy (invalid input) |
| denied standard library in kernel | `srderr` | `os` | FR-013 | architecture.md §9 | unhappy (invalid input) |
| nested denied standard library | `d20` | `net/http` | FR-013 | architecture.md §9 | unhappy (invalid input) |
| `sync` in a kernel or context package | `srderr`; `d20` | `sync` | FR-013, FR-016 | architecture.md §9 | unhappy (invalid input) |
| random outside dice | `srderr`; `dice/dicetest` | `math/rand/v2` | FR-027, FR-103 | architecture.md §3 | unhappy (determinism) |
| `log` in a kernel package | `telemetry` | `log` | FR-032 | observability.md §10 | unhappy (invalid input) |
| third-party import, even in a test | `dice` (test) | `github.com/stretchr/testify/assert` | FR-013 | architecture.md §1 | unhappy (invalid input) |
| import of `internal/` | `dice` (test) | `internal/archtest` | FR-076 | architecture.md §1 | unhappy (invalid input) |
| production code imports `testing` | `d20` | `testing` | FR-076 | testing.md §7 | unhappy (rule interaction) |
| unresolved module path | `d20` | `<module>/nope` | FR-007 | architecture.md §9 | unhappy (invalid input) |

### 4.6 Support rules (not checks)

These shape the support code. They yield no Violation of their own, so they are
not check ids (FR-005); each is a unit-level test of the support code
(section 8).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | `t.Run(` and `func Test` detection MUST work on the AST: a call to a method named `Run` on another receiver (`helper.Run(`), a `Test`-prefixed identifier that is not a top-level function with a `*testing.T` parameter, and a `TestMain` function MUST NOT be treated as a test. | testing.md §9 | unhappy (false positive) |
| FR-058 | A fixture module holding only `go.mod`, a root `doc.go`, and empty package directories (no `.go` file below the root beyond `doc.go` files) MUST yield zero Violations from every check (`TestCleanFixtureHasNoViolations`). FR-003's real-module run covers the repository itself. | architecture.md §9 Bootstrap | happy |
| FR-069 | A new consuming project is added to the denylist by adding its fragments to `consumerNames` in the same change that first introduces the dependency. The self-test asserts every denylist entry matches a fixture in each form of FR-025 (comment, string, path, case, separators). | AGENTS.md "Hard constraints" | unhappy (state error) |
| FR-070 | Citation grammar: in a `Requirement:` comment, including its `//   ` continuation lines, and in the string value of a `requirement` field of a table row, a citation is `doc/specs/<name>.md` followed by one or more `FR-<n>` ids separated by commas, `and` or `&`. Ids in `requirement` fields count for FR-054 and FR-055 as much as comment ids. | testing.md §5 (format, table-driven tests) | happy |
| FR-071 | The requirements table of a spec is every row in its section 4 whose first cell matches `^FR-\d+$`; rows of other section 4 tables (such as section 4.5's label column) are not ids. | testing.md §5; doc/specs/_template.md | happy |
| FR-087 | A moved-id row (section 3) is kept so ids are never renumbered. It is excluded from FR-005's meta-test, from FR-055's coverage rule and from the section 8 id-coverage rule. FR-087 is a support rule, not a check: it yields no Violation of its own. A citation (FR-070) of a moved id is a Violation of FR-054 ("keep citations true"). | testing.md §5 ("Keep citations true"); package design (ids are stable, so a moved requirement stays as a moved-id row) | unhappy (state error) |
| FR-072 | FR-036 to FR-040 scan non-test files only. `_test.go` files and the archtest fixtures MAY spell signal names, attribute keys and log messages as literals (they pin the values: `observability.md` §9). | observability.md §9 | happy |
| FR-066 | A `t.Run` call counts as inside a loop when it is lexically inside the body of a `for` statement (including `for ... range`) in the same function, Other generators (a helper that calls the subtest closure, `slices.Values`, a function literal passed to a table runner) are not recognised: only `for` and `for range` statements are. A subtest generated another way needs its own comment (FR-049). | testing.md §5 (table-driven tests) | happy |
| FR-094 | Spec header. The support code parses the first table of a spec (the two-column table under the title) into Status, Context, Roadmap phase and Depends on. Context is the first backticked token of its cell. Roadmap phase is the type phase: the first integer in the cell, or the first range `a-b` or `a–b` (which gives a lowest and a highest); everything after it is ignored, including later integers, `bulk data N`, `Phase 5` and any parenthesised text, as for Depends on. A cell with no integer is a Violation under FR-092. So `Types 2; bulk data 5` is phase 2, `3 for the Armor...; 5 for the rest` is phase 3, and `0 (see ../roadmap.md)` is phase 0. Depends on is the set of `` `name.md` `` tokens in the cell, separated by commas; `none` or empty means an empty set. Anything else in the cell (a link, prose, an FR id, a parenthesis) is a Violation of FR-100. Every token counts as a real dependency, so a spec MUST NOT name one there that it does not depend on; explanations belong in section 1. Any other header row (for example `Consumed by`) is ignored: its tokens are consumers, never dependencies (decisions FR-056). | doc/specs/_template.md; shared-types-and-ownership.md FR-014 | happy |

### 4.7 Specification checks (shared-types-and-ownership.md)

The decisions bind specs, not only code (`shared-types-and-ownership.md`
FR-013). These checks read `doc/specs/*.md` (files not starting with `_`) and
parse headers per FR-094. They are checks (FR-005), run on the real
repository, and sit in `internal/archtest/specs.go` (not yet written).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-088 | A spec MUST contain, in order, exactly the ten level-2 headings of the template: `1. Purpose and scope`, `2. Source`, `3. Vocabulary`, `4. Requirements`, `5. Model`, `6. Interface`, `7. Observability`, `8. Test plan`, `9. Open questions and decisions`, `10. Changelog impact`; and a header table with the rows Status, Context, Roadmap phase and Depends on. Status MUST be one of the FR-056 values. Context MUST be a backticked `package`, a kernel package name (`dice`, `ability`, `modifier`, `srderr`, `telemetry`), a test-support or adapter package name, or a context in the map (FR-022). Not yet enforced. | doc/specs/_template.md | unhappy (missing content) |
| FR-089 | Every file name in a spec's Depends on (FR-094) MUST exist in `doc/specs/`, MUST NOT be the spec itself, and the graph of all specs' Depends on edges MUST be acyclic; a Violation names the specs on the cycle. This is the cycle check of `shared-types-and-ownership.md` FR-056; FR-101 makes cycles impossible for ranked specs, and FR-089 still guards the rank-90 specs and any spec missing from the table. Not yet enforced. | shared-types-and-ownership.md FR-015, FR-016 (no cycles); FR-013 | unhappy (rule interaction) |
| FR-090 | A spec whose Context is a context (A) listing a spec whose Context is a different context (B) MUST have B upstream of A in the map (FR-022). A spec in context X MUST NOT list a spec of a context that is not upstream (forbidden edge). A kernel-package, test-support or adapter spec MAY be listed by any spec; a kernel spec lists only kernel specs, and a test-support spec lists only kernel and its own port's specs. A spec whose Context is `package` MAY be listed by any spec (`shared-types-and-ownership.md`, `errors-and-results.md` and this spec are `package` specs, and nearly every spec lists them); a `package` spec is constrained only by FR-089 (acyclic) and FR-092 (phase), and a `package` spec listing a context spec is not a forbidden edge. Not yet enforced. | shared-types-and-ownership.md FR-013; architecture.md §2 | unhappy (rule interaction) |
| FR-091 | In-context order is the rank table, not a copy. For two specs of the same context FR-101 already requires a strictly lower rank, so the old peer exception is gone (`feats-behaviour` and `spellcasting` are different ranks; the twelve `class-*` specs share rank 17 and so cannot list each other, and `class-wizard-evoker`, rank 18, may list `class-wizard`). Not yet enforced. | shared-types-and-ownership.md FR-015, FR-016, FR-056, 4.11 | unhappy (rule interaction) |
| FR-092 | Phase. A spec MUST list only specs whose type-phase upper bound (the end of its range, or its single integer, per FR-094; never a data phase) is at most the listing spec's lowest type phase (`shared-types-and-ownership.md` FR-014: same or earlier phase; bulk data may ship later without changing the cell). A Roadmap phase cell with no integer is a Violation. A spec of rank 90 or higher (FR-101) is exempt from this phase rule, because it lists every spec it verifies. Not yet enforced. | shared-types-and-ownership.md FR-014, FR-056 | unhappy (boundary) |
| FR-096 | Spec cross-references. In every spec (FR-094's file set), outside fenced code blocks, every spec citation MUST name an existing file in `doc/specs/` and ids that exist in that spec's requirements table (FR-071) and are not moved (FR-087). Grammar: the file token is `[a-z0-9_-]+\.md`, optionally preceded by `doc/specs/`, inside backticks, as link text or as a link target. After it, between the token and the first `FR-<n>`, only the characters `` ` `` `)` `]` `(` and spaces may stand; the id run then continues over the separators `,`, `and`, `&`, `/` and `to`, each followed by another `FR-<n>`, and ends at any other text (so in `<file> FR-005 for a, FR-034 for b` only FR-005 is cited; repeat the file name before an id that follows prose). A range `FR-a to FR-b` checks both endpoints. The pointer inside a moved-id row (section 3) is a citation like any other. A bare `FR-<n>` with no file token is local and is not checked here. A citation of a moved id is accepted only when the run is immediately followed by the moved-id suffix (section 3) and the id is in fact moved, so the statement stays true. The names in the constant `placeholderSpecNames` (`x`, `y`, `z`) are exempt: they are the fixture spec names of section 8. Mangled names are also Violations: any token matching `\b(doc-specs\|toolbox)-[a-z0-9-]+-md\b` (they mean `<x>.md` and `toolbox-<x>.md`). Test citations are FR-054, not this check. Not yet enforced. | shared-types-and-ownership.md FR-052; testing.md §5 ("keep citations true") | unhappy (missing content) |
| FR-099 | Content data specs. A spec named in the constant `dataSpecs` = {`content-thin-slice`, `content-species-backgrounds-feats`, `content-equipment`, `content-spells`, `content-monsters`, `content-classes`, `content-magic-items`} MAY list (FR-094) only `content-schema-and-decoder.md`, `content-source-port.md` and specs whose Context is `package`; any other file name in its Depends on is a Violation naming the data spec and the listed file (this includes another data spec, so FR-089's cycle check is not the only guard). A rank 7 bundle spec (`content-classes-data`; decisions FR-067) is not in `dataSpecs`: it MAY list the rank 6 data specs it validates against, and only FR-101 constrains it. The toolbox catalogues join `dataSpecs` when a `toolbox-*` spec is a pure data catalogue; no `toolbox-*` spec is in the set today because each lists use-case and rule specs (section 9). The constant only names the stems; the rule itself is not copied. A self-test fails when the FR-047 row of `shared-types-and-ownership.md` no longer contains, in backticks, each stem in `dataSpecs`, `content-schema-and-decoder` and `content-source-port`, or its FR-067 row no longer names `content-classes-data`. Not yet enforced. | shared-types-and-ownership.md FR-047, FR-067 | unhappy (rule interaction) |
| FR-100 | Header tokens only (`shared-types-and-ownership.md` FR-056). A token matches `` `[a-z0-9-]+\.md` ``, and tokens are separated by a comma and optional spaces. The Depends on cell, after removing every `` `name.md` `` token, commas and spaces, MUST be empty or `none`; a link, prose, a bare file name, an FR id or a parenthesis left over is a Violation naming the leftover text. Listing `shared-types-and-ownership.md` is allowed and not required. Consumers are not listed; the check cannot see intent, so it relies on FR-101 and FR-089. Not yet enforced. | shared-types-and-ownership.md FR-056 | unhappy (invalid input) |
| FR-101 | Rank rule. The rank table is parsed from section 4.11 of `shared-types-and-ownership.md` (rows `\| <integer> \| `a`, `b` \|`; the header row and any row whose first cell is not an integer are skipped), never copied. Ranks are arbitrary integers and only their order counts: the table has gaps and several specs per rank (the twelve `class-*` specs share rank 17; `class-wizard-evoker` is 18; `spell-effect-data` and `content-classes-data` are 7; `equipment-rules` is 9; `telepathy` and `grappling` are 23; `returning-to-life` and `spell-effects` are 27). A spec absent from the table is a Violation (a new spec needs a rank in the same change), as is a table name with no file or a name listed under two ranks. For a spec of rank below 90, every Depends on token other than `shared-types-and-ownership.md` (rank 0, always allowed, need not be listed) MUST name a spec of strictly lower rank; an equal or higher rank is a Violation naming both ranks. A spec of rank 90 or higher MAY list any spec; this spec is rank 90 and lists every spec it verifies as tokens (FR-061 of the decisions), and the rank-91 and rank-99 specs are not listed here, so no cycle. Because ranks strictly decrease along every edge, a cycle among ranked specs below 90 cannot exist; FR-089 reports the cycle if one appears through a rank-90 spec. The threshold 90 is a named constant (FR-009). A self-test fails when the decisions spec no longer has a table with a `Rank` header, or when the parse of the real table does not give `spell-effect-data` 7, `equipment-rules` 9, `higher-level-start-equipment-and-trinkets` 15, `class-wizard` 17, `class-wizard-evoker` 18, `telepathy` 23 and `returning-to-life` 27 (multi-name cells and gaps parse). Not yet enforced. | shared-types-and-ownership.md FR-056, FR-061, 4.11 | unhappy (rule interaction) |
| FR-102 | Documentation index. The file `doc/documentation_index.md` (named constant `documentationIndexPath`, FR-009) MUST be present and hold a table whose header cells are, in order, `File`, `Summary`, `Read when`; each data row's first cell is one backticked path relative to `doc/`, and no path appears twice. (a) Every `doc/specs/*.md` file in the repository, `_template.md` included, MUST have a row `specs/<name>.md`; a spec file with no row is a Violation naming the file, at the line of the table header. (b) Every row's path MUST name a file under `doc/`; a row for a deleted, renamed or never-created file is a Violation at the row's line. (c) A missing index, another header, a first cell that is not one backticked path and a duplicate path are Violations. Only the presence and validity of rows is checked; summaries and triggers are reviewed under the index's own rules. Not yet enforced. | `doc/documentation_index.md` ("One row per file under `doc/`"; no row for a file that does not exist); shared-types-and-ownership.md FR-052 | unhappy (missing content) |

### 4.8 Withdrawn ids

Ids are never reused. These requirements have no Go counterpart; the
Go requirement that replaces the intent, where there is one, is named.

| Id | Requirement |
|---|---|
| FR-063 | Withdrawn: there is no hand-written lexer; `go/parser` returns the AST and comment groups (FR-006, FR-008). |
| FR-065 | Withdrawn: a directive inside a comment or string cannot be an edge in a parsed AST (covered by FR-006 and FR-008). |
| FR-067 | Withdrawn: reachability of platform-specific adapters from entry points has no Go counterpart; adapters are never imported by kernel or context packages (FR-014). |
| FR-078 | Withdrawn: no generator and no drift check; `content/data/` JSON is the source of truth and a test decodes every embedded file (decisions FR-032). |
| FR-080 | Withdrawn: a Go package cannot import itself (compile error); an external test package may import the package it tests. |
| FR-093 | Withdrawn: there are no re-export files and no export closure; the exported names of a Go package are its declarations (FR-077). |

## 5. Model

- **Package `internal/archtest`** (standard library only: `go/parser`,
  `go/ast`, `go/token`, `io/fs`, `path`; tests add `testing/fstest`). It is a
  tooling package: executed by `go test`, never imported (Policy Kind
  `KindTooling`).
- **Types.** `Module` (module path, `HasRequire`, `Packages`, the `fs.FS` it
  was loaded from), `Package` (directory, `Files`), `File` (slash path, `Test`
  flag, `*ast.File`, `*token.FileSet`; `Imports()`), `Import`, `Policy`
  (`Entries map[string]Entry`), `Entry` (`Kind`, `Allow`), `Kind`
  (`KindDocs`, `KindKernel`, `KindContext`, `KindSupport`, `KindAdapter`,
  `KindTooling`, `KindTests`), `Violation` (`Rule`, `File`, `Line`, `Msg`).
  Planned support types for the unimplemented checks: `ContextMap` (parsed per
  FR-022), `SignalConstant` (identifier, value, kind, file, line),
  `CatalogueRow` (seven cells, line), `SpecHeader` (FR-094) and `RankTable`
  (FR-101).
- **Entry points.** `Load(fs.FS) (Module, error)` does all reading and
  parsing (FR-001, FR-002, FR-010); `DefaultPolicy() Policy` returns the
  allow-list (FR-022); `Check(Module, Policy) []Violation` runs every rule and
  returns the violations sorted (FR-011). The scanner reads from an `fs.FS`, so
  the same code runs on `os.DirFS` of the module root and on `fstest.MapFS`
  fixtures.
- **Rule names** (constants carried by `Violation.Rule`) and the check ids
  they enforce:

| Rule constant (name) | Check ids | Architecture check | Status |
|---|---|---|---|
| `RuleUnlisted` (`unlisted-package`) | FR-021, FR-007 | A1, A11 | implemented |
| `RuleImport` (`import-not-allowed`) | FR-017 to FR-019, FR-023, FR-024, FR-061, FR-062, FR-075, FR-007 | A1 | implemented |
| `RuleKernelUp` (`kernel-imports-context`) | FR-015 | A1 | implemented |
| `RuleIsolation` (`test-support-isolation`) | FR-014, FR-074 | A2 | implemented |
| `RuleInternal` (`internal-import`) | FR-076 (first sentence) | A2 | implemented |
| `RuleStdlib` (`stdlib-denied`) | FR-013, FR-016, FR-027, FR-032, FR-103 (imports) | A3 | implemented |
| `RuleThirdParty` (`third-party-import`) | FR-013, FR-032 | A3 | implemented |
| `RuleGoMod` (`go-mod-require`) | FR-083, FR-033 | A3, A10 | implemented |
| `RuleInit`, `RuleVar`, `RuleEffect` (`init-function`, `package-level-var`, `hidden-effect`) | FR-016, FR-027, FR-028, FR-035, FR-103 (calls) | A4 | implemented |
| `RulePanic` (`panic-discipline`) | none of this spec's ids; owned by `errors-and-results.md` FR-030, FR-046 | A5 | implemented |
| `RuleDoc`, `RulePackageDoc`, `RuleStutter` (`missing-doc`, `missing-package-doc`, `stutter`) | none of this spec's ids; owned by `go.md` | A9 | implemented |
| `RuleConsumer` (`consumer-name`) | FR-025, FR-026, FR-068, FR-069 | A10 | implemented |
| `RuleCycle` (`import-cycle`) | FR-020 | A11 | implemented |
| `RulePortUse` (`port-use`) | `errors-and-results.md` FR-029 | A6 | implemented |
| `RuleVariant` (`variant-set`) | FR-077 (a), `errors-and-results.md` FR-044 | A7 | implemented |
| `RuleErrorShape` (`error-shape`) | `errors-and-results.md` FR-006, FR-007, FR-020 | A8 | implemented |
| `RuleNoticePaths` (`notice-paths`), `RuleEmbed` (`embedded-data`) | FR-079, FR-095 | A12 | implemented |
| `RuleRuleTag` (`rule-tag`) | FR-029, FR-030, FR-031 | A13 | implemented |
| `RuleDocsOnly` (`docs-only`) | FR-062 (documentation only) and the "tests only" row of `architecture.md` section 1 | A1 | implemented |
| (planned) signals, catalogue, citations, specs | FR-034, FR-036 to FR-048, FR-049 to FR-056, FR-081, FR-082, FR-084 to FR-092, FR-094, FR-096 to FR-102, FR-104 | n/a | specified |

- **Invariants.** A Module is built once per test and never mutated; paths use
  `/`; violations are sorted before reporting (FR-011); a rule applies to
  production files unless it says test files.
- **Failures.** Violations are test failures, not `srderr.PortError`s. `Load`
  returns an `error` for a missing `go.mod`, a missing module line or an
  unparsable file (FR-001, FR-010); a helper given a malformed fixture returns
  an error or fails the test, and panics only for programmer error
  (`architecture.md` §6).
- **Constants** (each constant that copies a decision has a self-test that
  fails when its source text no longer contains it, FR-079 (c), FR-085 (c),
  FR-098, FR-099): the denied standard-library set and its exceptions
  (`deniedStdlib`, `stdlibExceptions`), the banned calls (`bannedCall`), the
  recover location (`guardFile`), the exempt paths of FR-026
  (`exemptFromNames`), the generic variant names (FR-077), the double prefixes
  (FR-081), `portHandles` (`Span`, `Counter`, `Histogram`, `Gauge`, FR-082),
  the reserved attribute keys (`error.type`), the shared attribute value sets
  (FR-085), `noticeCoveredPaths` (FR-079), `leafContexts` (FR-098),
  `placeholderSpecNames` (FR-096), `dataSpecs` (FR-099),
  `documentationIndexPath` (FR-102), the skipped directories (FR-002), the
  rank-90 threshold (FR-101), and the consumer-name denylist assembled from
  fragments so no file names a consumer. The context map and rank table are
  parsed, not constants, once FR-022 and FR-101 are enforced; today the
  context map is the `DefaultPolicy` table.

## 6. Interface

- No exported API of the library changes: nothing is added to any public
  package, and `internal/archtest` is not importable outside the module. No
  new `go.mod` dependency (FR-083): the suite uses `io/fs`, `go/parser` and
  `testing` only, and parses Markdown tables line by line.
- Files today: `internal/archtest/{doc,load,policy,check,imports,decls,docs,names,ports,variants,errshape,embedded,ruletags}.go`
  with `check_test.go` (the in-memory fixtures: `TestCleanFixtureHasNoViolations`,
  `TestEachRuleFires`, `TestAllowedFormsAreSilent`, `TestImportCycleIsReported`,
  `TestLoadErrors`, `TestViolationString`), `guardrails_test.go` (one table per new rule: `TestPortUse`, `TestVariantSets`, `TestErrorShape`, `TestEmbeddedData`, `TestDocsOnly`, `TestSentinelVars`, `TestHiddenEffectsFire`, `TestRuleTags`), `embedded_internal_test.go` (the FR-079 (c) self-test) and `module_test.go` (the real
  module: `TestModuleArchitecture`, `TestPolicyCoversEveryPackage`). Planned,
  one file per concept as they land: `portcheck.go` (FR-081, FR-082), `signals.go`
  (FR-036 to FR-040, FR-085), `catalogue.go` (FR-041 to FR-048), `citations.go`
  (FR-049 to FR-056, FR-070, FR-071) and `specs.go` (FR-088 to FR-102), each with a `_test.go`.
- Everything is synchronous and in-memory; no ports, so no port failure
  contract and no double. Run with `go test ./internal/archtest/...`; each
  real-module test MUST finish in under 10 seconds (FR-073).

## 7. Observability

Silent by design: test code run in development and CI, never inside a
consumer's process, so no use case, port or adapter to instrument. It owns no
signal; `doc/observability/signals.md` gets no entry from this spec. It enforces
the rules for everyone else's signals (FR-032 to FR-048, FR-084, FR-085).

Evaluation: (1) what can go wrong is a vacuous, loose or drifting check
(FR-001, FR-005, FR-012, FR-021, FR-022); (2) an operator sees `file:line`,
rule and message (FR-004), and a vacuous pass is exposed by FR-005 and
FR-012; (3) diagnosis needs only that line and the requirement id; (4) nothing
is counted in-package, and FR-073 budgets each real-module test at 10
seconds; (5) there is no existing instrumentation, and the value sets FR-085
checks are fixed in decisions FR-041; (6) a failing `internal/archtest` CI job
MUST block merge, maintainers watch suite duration (module growth, a
quadratic check), a rule change must update its guardrail test (`testing.md`
§9) and reviewers check the reverse; (7) nothing is noisy, since reports list
only violations.

## 8. Test plan

Check ids are verified at level `architecture`: the check on the real module
plus a self-test with fixtures (FR-005). Support code (import extraction,
citation and table parsers) is tested at level `unit` in `internal/archtest`
(`testing.md` §3). Guardrail tests are exempt from FR-049 only (`testing.md`
§9) and MUST cite this spec like any other test (`testing.md` §5); FR-055
applies once this spec is `implemented`. Fixtures are in-memory
(`testing/fstest.MapFS` overlays on the clean module of `check_test.go`), not
files under `testdata/`, except for the spec and catalogue checks, which use
small Markdown fixtures under `internal/archtest/testdata/`.

"Status" says whether the test exists today; a row marked "planned" is owed by
the change that enforces the check.

| Ids | Level | Notes | Status |
|---|---|---|---|
| FR-001, FR-012 | architecture | `TestLoadErrors`: missing `go.mod`, no module line; module with fewer than five packages fails; `TestPolicyCoversEveryPackage`. | implemented |
| FR-002, FR-010, FR-105 | architecture | Overlay with `testdata/`, `vendor/`, `.hidden/`, `_skip/` directories holding invalid Go (skipped; `TestAllowedFormsAreSilent` "testdata ignored"); an unparsable `.go` file (`TestLoadErrors`). | implemented |
| FR-003, FR-004, FR-005, FR-059 | architecture | `TestViolationString` (format); a fixture with three violations: all three lines, sorted. Meta-test (planned): read the `Rule*` constants, assert `TestEachRuleFires` has a case for each; a rule missing from the table fails. | partly implemented |
| FR-006, FR-007, FR-060, FR-064 | architecture, unit | Import cases: aliased, dot, blank, grouped, multi-line, behind a `//go:build` tag, a `_test.go` file, an import of a missing module path (FR-007). | planned |
| FR-008, FR-009 | architecture, unit | A banned import or call named only in a comment or string (no Violation); `//nolint` next to a violation still reports. | planned |
| FR-011 | architecture | Each check twice on a shuffled package order; equal output (sorted by `Check`). | planned |
| FR-013 to FR-019, FR-023, FR-024, FR-027, FR-032, FR-061, FR-062, FR-074 to FR-076, FR-095, FR-097, FR-098, FR-103 | architecture, unit | Rows of section 4.5, table-driven, one case per row (`TestEachRuleFires`, `TestAllowedFormsAreSilent`). Plus the cases FR-097 owes once enforced: two use cases in separate files, one mentioning the other (Violation, at the mention); the same in a non-use-case file (Violation); a domain service mentioned from a use case (clean); a mention in a comment or string (clean, FR-008); a `Command` or `Result` type of another use case (clean); a mention in a `_test.go` file (clean). | partly implemented |
| FR-020, FR-021, FR-022, FR-098 | architecture | `TestImportCycleIsReported` (A to B to A); unlisted package (`TestEachRuleFires`). Planned: self-edge, three-cycle, `content` allowing `toolbox` upstream; the Policy parsed from `architecture.md` §1 and §2 against `DefaultPolicy` and the `AGENTS.md` "Current contexts" list (other header, a cell without backticks, a context missing from the table); FR-098: `toolbox` with four upstreams and nobody listing it (clean); `character` listing `toolbox`; `toolbox` listing an unknown package; self-test that the decisions spec FR-045 row still contains `toolbox`. | partly implemented |
| FR-077, `errors-and-results.md` FR-044 | architecture | `TestVariantSets`: generic type, const and grouped const in a context package (Violation); the same names in a kernel package, near-miss names, a method or field of that name, a test file (clean). Sealed sets: doc missing an implementer, a pointer-receiver implementer, an implementer in another file, a doc with no mention, a plural that is not a word match (Violation); a fully documented set, an exported-only interface, no implementers, a partial implementer (clean). | implemented |
| FR-079, FR-095 | architecture, unit | `TestEmbeddedData`: missing, extra and slash-less Scope paths, a non-JSON file, two Go files, JSON with no Go file, a Go file without `go:embed`, `embed` outside `content/data`, `content/data` importing a context, `content` or a context importing `content/data` (Violation); exact Scope, tokens outside Scope, no content, placeholders only, no directory, a test file importing the data package or `embed` (clean). `embedded_internal_test.go` is the constant self-test and its drift mutation. | implemented |
| FR-081, FR-082, FR-084 | architecture | Planned: `FakeX` type in a production package (FR-081); a port with no double, or no `Run<Port>Contract` (FR-082); `Span`, `Counter`, `Histogram`, `Gauge` in `telemetry` with no double or contract (clean, `portHandles`); a `FakeSpan` in a `*test` package (clean); no ports (vacuous); a `NoopX` type outside `telemetry` (FR-084). | planned |
| FR-083, FR-033 | architecture | `go.mod` with a `require` directive (`TestEachRuleFires` "go.mod require"); without (clean). | implemented |
| FR-025, FR-026, FR-068, FR-069 | architecture | Denylist fragments joined at run time; the name in a comment, a string, a file path, other case and separators (`TestEachRuleFires` "consumer name", `TestAllowedFormsAreSilent` "module path in text"); `AGENTS.md`, `go.sum`, `go.mod` exempt; the denylist source scanned for plain text. | partly implemented |
| FR-028, FR-016, FR-035 | architecture | `fmt.Println`, `println`, `go` statement, `time.Now()` call (Violation) and `time.Now` as a value (clean); an `init` function and a package-level `var` (`TestEachRuleFires`). | implemented |
| FR-029, FR-030, FR-031 | architecture | `TestRuleTags`: zero, two and indented tag lines, stem mismatch, uppercase hex, 11 and 13 digits, no backticks, a value shared by two rule files, copied into `AGENTS.md`, `.github/` and a Go comment, repeated in its own file, an unlisted rule file, a listed missing file (Violation); clean files, a non-Markdown file, a spec-file tag, tag-like prose (clean). `TestRealModuleHasRuleFiles` pins four real rule files. | implemented |
| FR-034 | architecture | Planned: a context file declaring no use case importing `telemetry` (Violation); an options file (clean). | planned |
| FR-036 to FR-040, FR-072 | architecture | Planned: the same literal in a `_test.go` file passes (FR-072). Signals-file fixtures: `LogContentSetLoaded = "content set loaded"` (valid; a fixture message only) and `content.set.loaded` or `Content set` (FR-038); literal at a call site; concatenated name; unprefixed constant (FR-037); wrong context segment; uppercase; duplicate value; signal constant in `telemetry`. An attribute constant `"Outcome"` or `"outcome"` with no `srd5e.` prefix (FR-039). | planned |
| FR-041 to FR-048 | architecture | Planned: catalogue: empty table (valid), extra column, short row, name only in code, name only in catalogue, duplicate row; cells empty, whitespace, `TODO`, `n/a`; wrong Type word; lower-case `span` and `counter, unit {test}` (valid); Type vs prefix mismatch; attribute only in code; attribute only in catalogue. | planned |
| FR-085 | architecture, unit | Planned: `ValueOutcomeSuccess = "success"` with the key in its doc comment (clean), `"miss"`, `"neither"` for roll mode, `"critical_hit"` for `srd5e.outcome`, a value with no key in its comment, an `error.type` value `dice_roller_error` against a port name constant `dice` (clean) and a value not ending `_error` (Violation). Catalogue: `` `srd5e.outcome` (`success`, `failure`) `` clean; `` (`success`, `hit`) `` Violation. The self-test deletes one value from a copy of `observability.md` §5 and fails. | planned |
| FR-049 to FR-054, FR-057, FR-066, FR-086 | architecture, unit | Planned: the cases table below; a commented-out `func Test`; `t.Skip` without and with an `Issue:` line; `t.Skip` with an empty `Issue:`. | planned |
| FR-055, FR-056, FR-070, FR-071, FR-087 | architecture, unit | Planned: fixture specs: `implemented` with an uncited id, unknown file, unknown id, bad status; an id cited only in a table row's `requirement` field (covered); a continuation-line citation; several ids in one citation; a range (FR-054); an id listed `Deliberately not covered: FR-3 - reason` (accepted); a 4.5-style table whose first cell is a label (not an id). FR-087 is a support rule, so its fixture expects FR-054: a spec with a moved-id row cited by a test (Violation), uncited under `implemented` (clean), skipped by the FR-005 meta-test. Real run covers the actual `doc/specs/`. | planned |
| FR-088 to FR-092, FR-094, FR-099 to FR-101 | architecture, unit | Planned: data spec (`content-equipment`) listing `content-schema-and-decoder.md`, `content-source-port.md` and a `package` spec (clean), another data spec (Violation), a non-package spec such as a kernel spec (Violation); a non-data spec listing a data spec (clean); self-test that the FR-047 row of the decisions spec still names each `dataSpecs` stem. Header cell with prose, a link, a bare file name or an FR id (FR-100 Violation) vs only tokens (clean). A rank-90 phase-0 spec listing a phase-4 spec (clean, FR-092 exemption); rank-4 spec listing a rank-4 and a rank-5 spec (FR-101), rank-5 listing rank-4 (clean), a spec missing from the table (FR-101), a rank-90 spec listing rank-1 and rank-60 specs (clean). Missing or reordered heading; missing header row; Depends on naming a missing file, itself, a two-spec cycle. `combat` listing `character` and `character` listing `combat` (forbidden edge); a `content` spec listing `shared-types-and-ownership.md` (clean); `conditions` listing `attacks` (FR-101), `attacks` listing `conditions` (clean); a `class-*` listing `spellcasting` (clean), `spellcasting` listing a `class-*` (Violation), two `class-*` listing each other (FR-101). Ranks of table 4.11 with gaps: `class-wizard-evoker` (18) listing `class-wizard` (17) clean, `class-wizard` listing it Violation; `content-classes-data` (7) listing a rank 6 data spec clean, a rank 6 data spec listing another rank 6 spec Violation (FR-099 and FR-101); `equipment-rules` (9) listing `content-equipment` (6) clean; `higher-level-start-equipment-and-trinkets` (15) listing `level-advancement-and-multiclassing` (14) clean and `feats-behaviour` (15) Violation; `telepathy` (23) listing `conditions` (22) clean and `grappling` (23) Violation; `readying-a-spell` (26) listing `initiative-and-actions` (25) clean; `returning-to-life` (27) listing `magical-effects` (26) clean and `spell-effects` (27) Violation; a `Consumed by` header row naming a higher-rank spec (ignored, clean); a spec listed under two ranks; a table name with no file. Phase `0` parsed as 0, `3-4` as 3 and 4, `Types 2; bulk data 5` as 2 (a phase-3 spec listing it is clean), `3 for the Armor...; 5 for the rest` as 3; phase-2 listing phase-2 (clean); phase-0 listing phase-1 (Violation); phase-3 listing a `3-4` spec (Violation); a cell with no integer; `none. ... foo.md ...` as no dependencies. Real run covers the actual `doc/specs/`. | planned |
| FR-102 | architecture | Planned: index fixtures: every spec file has a row (clean); a spec file with no row, as when a new spec is added without its row (Violation naming it); a row `specs/gone.md` for a missing file (Violation at the row); a row for an existing non-spec file such as `rules/testing.md` (clean); a duplicate path; another header; a first cell without backticks; no index at all. The real run reads the repository's index. | planned |
| FR-096 | architecture, unit | Planned: `x.md` placeholder (clean); a citation of a missing file; of a missing id; of a moved id (Violation) and the same followed by the suffix (clean when it is moved, Violation when not); a range whose end is moved; ids joined by `and` and `&`; prose after the first id (second unchecked); a fenced block (ignored); a spec name with its final dot turned into a hyphen under the `doc-specs` and `toolbox` prefixes (mangled Violations; assembled at run time so this spec stays clean); a moved-id row's pointer to another moved id. The real run MUST pass before a spec change is accepted (decisions FR-052). | planned |
| FR-058, FR-073 | architecture | `TestCleanFixtureHasNoViolations`: the clean fixture through every check yields zero Violations. Each real-module test runs under the package `go test` timeout; a fixture-free assertion that no test sleeps (planned). | partly implemented |

Cases for FR-049 to FR-053 (table-driven in the self-test, planned):

| Label | Fixture | Expected |
|---|---|---|
| comment directly above | `// Requirement: ...` then `// Path: happy` then `func TestX(` | none |
| blank line between | comment, blank line, `func TestX(` | FR-049 |
| `Rule:` instead | `// Rule: ...` plus `Path:` | none |
| none with reason | `Requirement: none - pins current behaviour` | none |
| none bare | `Requirement: none` | FR-050 |
| subtest in loop, loop commented | `Requirement: per case`, `Path: per case`, `for` containing `t.Run(` | none |
| subtest in loop, function uncommented | `for` containing `t.Run(` | FR-049 |
| function comment not per case | `Requirement: x` (not `per case`) above a function whose loop contains `t.Run(` | FR-051 |
| per case on plain test | `Requirement: per case` above a `Test` function with no loop | FR-052 |
| per case path mismatch | `Requirement: x`, `Path: per case` | FR-053 |
| empty unhappy kind | `Path: unhappy ()` | FR-053 |
| tooling method | `helper.Run(` | none (FR-057) |
| `Example` and `Fuzz` | `func ExampleX()`, `func FuzzX(f *testing.F)` with no comment | none |
| unsupported generator | `slices.Values(cases)` driving `t.Run` with a comment only above the call | FR-049 |
| table-row citation | `for` with per case comment and a case whose `requirement` field cites FR-2 of the placeholder spec, FR-2 exists | none |
| table-row unknown id | case cites FR-99 of the placeholder spec | FR-054 |
| several ids | `Requirement:` citing FR-1 and FR-2 of the placeholder spec | none |
| range | `Requirement:` citing the range FR-1 to FR-3 of the placeholder spec | FR-054 |
| continuation line | id on a `//   ` line | none |
| `none` without `Path:` | `Requirement: none - reason` only | FR-053 |

Mutation-testing scope: none (section 1).

Deliberately not covered: none.

**Implementation status.**

- Done (`internal/archtest`): the harness and the architecture checks A1 to A13 (4.1, and the import, naming, declaration and documentation rows of 4.2 and 4.5), and the rows whose section 8 Status is `implemented`.
- Not done: the rows whose section 8 Status is `planned`, the harness meta-tests (FR-003 to FR-009, FR-011, FR-060, FR-064), and every row marked "Not yet enforced" in section 4.

Not yet implemented: FR-021, FR-022, FR-032, FR-034, FR-036 to FR-048, FR-049 to FR-056, FR-081, FR-082, FR-084, FR-085, FR-086, FR-088 to FR-092, FR-096 to FR-102, FR-104.

## 9. Open questions and decisions

Decided (one line each; the rationale is kept here because Source cells point at it):

| Decision | Outcome |
|---|---|
| Rule-file tag replaced on edit? | Decided: a CI step outside the unit tests, because it needs git history; out of scope for this spec. |
| FR-073 loading budget with all data in every Module | Decided: keep the 10 second per-test budget; if bulk data pushes a test over it, the loader skips parsing `content/data/` (JSON only, no Go beyond the embed file). Trigger: the first bulk-data change whose real-module test exceeds 5 seconds on CI. |
| Signal and attribute constants | String `const`s with the prefixes `Span`, `Metric`, `Log`, `Attr`, `Value` in the `signals.go` file of the owning package (`d20/signals.go`, `telemetry/signals.go`); shared keys are in `telemetry/signals.go`; this follows the existing use-case specs (`resolve-d20-test-use-case.md` FR-040, `combat-foundations.md` FR-027, `observability-ports.md` FR-052). Kind is told by the identifier prefix (FR-037). |
| Log event | The constant message (`observability-ports.md` FR-002); lowercase pattern, not dotted (FR-038). No event-name field on the port. |
| Which interfaces are not driven ports? | The handles in `portHandles` (FR-082; `observability-ports.md` FR-065) and consumer-side interfaces a context declares for itself (`d20.Roller`). `release-0-1-0.md` FR-084 takes its port definition from FR-082 and applies `portHandles` too. |
| Who owns the dependency rule? | FR-083 here. `release-0-1-0.md` FR-060 cites FR-083 and adds only the version and `replace` directive rules. |
| Do the current data-spec headers pass FR-099 (decisions FR-047)? | Yes: every spec in `dataSpecs` lists only `content-schema-and-decoder.md`, `content-source-port.md` and, where used, the package-context `errors-and-results.md`; the kernel types they use are reached through the decoder spec. Toolbox catalogues join `dataSpecs` when a `toolbox-*` spec is pure data. |
| Does the documentation index have rows for every spec? | Yes: every file in `doc/specs/` has a row in `doc/documentation_index.md` and no row names a missing file, so FR-102's real run passes once enforced. |
| Lexer vs `go/types` | `go/parser` with AST inspection and in-memory self-tests; no type checking, so each check documents its blind spots (a banned call through a variable). Revisit with `go/types` if a check needs types. |
| Beyond the rule text (FR-013, FR-015, FR-016, FR-032, FR-076, FR-082 to FR-084) | `log`, `log/slog` outside the adapter, `sync`, `testing` in production code and the extra third-party names close obvious bypasses; Source cells say "package design". |
| Call-site string literals (`Meter`, `Tracer`, `Logger`, attribute values) | Not checkable without types; the recording-double tests that pin names (`observability.md` §9) catch them. |
| Registering other specs' checks (E1 to E6, R-1, R-4, Q-13, Q-17) | R-1, E1 to E6, Q-13, Q-17 declined: FR-005 ids would collide and each spec has its own self-test duty (`testing.md` §8). R-2 accepted (FR-002 scope of non-Go text files); R-4 accepted (FR-068); R-3 met by FR-079. Those specs must not claim registry slots. |
| `.pubignore` and `doc/research/` distribution (old FR-079 (b)) | Dropped: the Go module proxy serves the whole repository, so there is no distribution exclusion to check. Research notes stay Markdown only with attribution (decisions FR-033). |

Decisions:

- **Q-1 (decided).** A use-case file is defined by role (it declares a use case type), not by the name `*_usecase.go`; `go.md` does not require the suffix. `d20/resolver.go` is a use-case file and may import `telemetry` and `srderr`. FR-034, FR-097 and `errors-and-results.md` FR-029 follow that definition. When A6 or FR-034 is enforced, the check keys off the declared use case type.
- **Q-2 (decided).** `d20` publishes `srd5e.d20_test.*` signal names while the Go package is `d20`. FR-038 accepts the underscore-insensitive match. The names are a telemetry contract and are not renamed.
- **Q-3 (decided, owner to confirm).** The owner of a port (a package declaring a top-level `const Port`, today `dice`) may call `srderr.WrapPort`, `GuardValue` and `ContractViolation` for its own port, because `dice.Roll` is the port's guard and is not a use case. `errors-and-results.md` FR-029 and `architecture.md` section 9 A6 say so. A use-case file is a file of a context package that imports `telemetry` and declares an exported type with an exported method (`d20/resolver.go`).
- **Q-4 (decided).** A sealed interface is any interface with at least one unexported method. The rule name for A12 is `notice-paths`. `content/data` may hold `.gitkeep` and `README.md`. Architecture rule files were re-tagged when A13 landed (their old tags were not `<stem>-<12 hex>`).
- **Not enforced, review only.** A new `Port` value being added to the errors-and-results port table (A8 checks form and uniqueness only); A5 does not see `recover` or `panic` hidden behind an alias; A6 does not follow a dot import of `srderr`; A7 does not see methods promoted by embedding; the "only `*test` packages and `scenarios` import more than one parent package" rule; FR-022 (Policy equals the markdown tables).

Other decisions are recorded in the decisions spec: ownership of FR-079 (FR-050), toolbox leaf (FR-045), use-case rule (FR-044), rank table (FR-056), citations (FR-052), port doubles and contract suites (FR-038, FR-039).

## 10. Changelog impact

None. Test and tooling only: no user-visible, API or signal change. Not
**BREAKING**.
