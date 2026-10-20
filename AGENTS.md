# srd5e-go — contributor/agent rules

Go implementation of the fifth-edition rules in System Reference Document
5.2.1, published as a library that other Go projects import.

## Read the rules first

Before doing anything else on a task, read each of these files in full and
quote the `RULE-TAG` from the "Read receipt" at the end of each, verbatim, in
your first message:

- `doc/rules/architecture.md`
- `doc/rules/observability.md`
- `doc/rules/testing.md`
- `doc/rules/go.md`

Format: `Rule tags: architecture=<tag>, observability=<tag>, testing=<tag>, go=<tag>`

The tags are only in the files. Do not guess or reuse one from memory. A wrong
or missing tag means you did not read the file. When you edit a rule file,
replace its tag with a fresh random value; new rule files go on this list.

## Hard constraints

- **No consumer-specific code.** Nothing may reference any consuming
  project. If a consumer needs something, expose a general extension point
  instead. The one exception is the module path
  `github.com/Whealinghold/srd5e-go` (the `module` line in `go.mod`, and the
  import paths that derive from it), which is where this repo is hosted: the
  `whealinghold` GitHub organisation. It is not a code dependency, and
  nothing else may name the organisation. `internal/archtest` (check A10,
  rule `consumer-name`) enforces this over every text file, hidden directories
  included (only `.git` is skipped): it ignores occurrences of the module path
  and exempts only this file, which has to state the rule.
- **Pure Go standard library.** No third-party runtime dependencies (no
  logging, telemetry, SDK or test-assertion packages) without discussion.
- **Public API = the public packages.** Every non-`internal/` package listed
  below is API under the compatibility promise in `doc/rules/go.md`. There is
  no facade or re-export package. `internal/` is never imported from outside
  this module.
- **Never include anything outside SRD 5.2.1.** See "Licensing and source
  material" below. This applies to code, data, tests, docs, examples and
  commit messages alike.

Everything else (layering, ports, purity, errors, telemetry, tests,
versioning) is in the rule files.

`doc/specs/shared-types-and-ownership.md` is binding on every other spec and on
the code: it settles who owns each shared type, the dependency order inside
contexts, result and event naming, content delivery, and where ports, no-ops and
test doubles live. Where another spec conflicts with it, it wins and the other
spec changes.

## Licensing and source material

- **Code** is MIT-licensed (`LICENSE`).
- **Game content** comes from one source only: the **System Reference
  Document 5.2.1** (SRD 5.2.1), which Wizards of the Coast LLC provides under
  the **Creative Commons Attribution 4.0 International License** (CC-BY-4.0).
  The document is the PDF linked from `doc/research/sources.md`.
- **If it is not in that PDF, it does not go in this repository,** however
  well known it is. That excludes every other fifth-edition book, the earlier
  SRD 5.1 (a different rules edition; do not mix them), third-party and
  community material, and anything remembered from play. When something looks
  like core fifth-edition material but cannot be found in the PDF, leave it
  out and flag it in your summary.
- Cite the SRD by section and page (`SRD 5.2.1, "Damage Rolls", p. 16`).
  Where the SRD's text is reused in data, keep it faithful to the source.

### Trademarks and attribution

CC-BY-4.0 licenses the SRD's content. It does not license Wizards' trademarks.

- **Attribution** uses the statement from the SRD's own "Legal Information"
  page, verbatim, in `NOTICE.md` (and wherever the content is redistributed).
  Do not add any other attribution to Wizards or its parent or affiliates.
- The package MAY say it is "compatible with fifth edition" or "5E
  compatible" (the SRD allows exactly this). It MUST NOT use Wizards'
  trademarked names or logos as a brand, in package or identifier names, or
  imply endorsement.
- Changes to the SRD material (restructuring it into code and data) are
  indicated in `NOTICE.md`, as CC-BY-4.0 requires.

### The notice

- The notice lives in the repository root as `NOTICE.md`, next to `LICENSE`.
  It MUST state which paths it covers (`content/data/` only, the bundled game content) and that
  the MIT `LICENSE` covers the code, so the two licences stay clearly
  distinguishable. Bundled game content lives in `content/data/`, separate
  from code.
- `NOTICE.md` exists from the start. The first change that bundles any game
  content MUST update its Scope status and cite the pages, taken from the
  SRD's own "Legal Information" page. Do not write the attribution from
  memory.
- A new SRD version is a deliberate change: update the statement, the cited
  pages and `doc/research/sources.md` together.

## Reporting a change

Every PR description and every agent's final summary has a `Testing` entry
(see `doc/rules/testing.md` §2) and an `Observability` entry (see
`doc/rules/observability.md` §2). `.github/pull_request_template.md` has the
headings. User-visible changes also get a `CHANGELOG.md` entry
(`doc/rules/architecture.md` §7).


## Directory structure

```
srd5e-go/
├── go.mod                  # module github.com/Whealinghold/srd5e-go; stdlib only
├── doc.go                  # package srd5e: documentation only, no logic or re-exports
├── .golangci.yml
├── .gitignore
├── .codacy.yml
├── .gomuignore             # packages excluded from mutation testing
├── justfile
├── CHANGELOG.md
├── LICENSE                 # MIT, covers code
├── NOTICE.md               # CC-BY-4.0 attribution; covers content/data/ only
├── README.md
├── CLAUDE.md               # imports this file
├── AGENTS.md               # this file
├── .github/
│   ├── workflows/ci.yml
│   └── pull_request_template.md  # Testing / Observability / Changelog headings
├── doc/
│   ├── rules/              # normative rules (architecture, observability, testing, go)
│   ├── observability/      # signals.md: catalogue of every metric/span/log event
│   ├── plans/              # implementation plans
│   ├── roadmap.md          # what is built and what is next
│   ├── documentation_index.md  # index of every document under doc/
│   ├── research/           # sources.md: which document the rules come from
│   └── specs/              # one spec per unit of work; shape in _template.md
├── dice/                   # Die, Expression, Roll, Result, Roller port, SeededRoller
│   ├── dicetest/           # FixedRoller, CallbackRoller, min/max rollers, RunRollerContract
│   └── randroller/         # unseeded, concurrency-safe dice.Roller over math/rand/v2
├── ability/                # Ability enum, Score, derived modifier
├── modifier/               # Modifier, advantage/disadvantage RollMode, RollCircumstance
├── srderr/                 # PortError, WrapPort, Guard, sentinel errors
├── telemetry/              # Logger, Tracer, Meter ports, Noop implementations, signal names
│   ├── telemetrytest/      # recording doubles + contract suite
│   └── slogtelemetry/      # optional adapter from log/slog to telemetry.Logger
├── content/                # content context (ports, value objects, Set, decoder)
│   ├── data/               # embedded SRD JSON (go:embed), created with the first content
│   └── contenttest/        # FakeSource, FakeDecoder, contract suite
├── d20/                    # first bounded context (domain + use case, one package)
├── internal/
│   └── archtest/           # architecture tests (import scan against the allow-list)
└── scenarios/              # cross-package tests, public API only (package scenarios_test)
```

Future contexts get their own top-level package (`content/`, `character/`,
`combat/`, `toolbox/`); bundled content lives in `content/data/` as JSON loaded
with `go:embed` (no generator step). Do not create them before their spec.

Dependency rule: kernel packages (`dice`, `ability`, `modifier`, `srderr`,
`telemetry`) must not import any context package. Contexts import kernel
packages and the published API of upstream contexts only. Nothing but tests
imports `*test` packages. `internal/archtest` encodes the allow-list as a table
and fails when a package imports something not allowed; update the table with
the structure.

## Bounded contexts

Current and planned contexts (provisional). Allowed dependencies between them
are the context map in `doc/rules/architecture.md` §2; change it there.

- `d20/`: D20 Tests (ability checks, saving throws, attack rolls), DCs, Advantage and Disadvantage, success and failure resolution (exists)
- `content/` (in progress, Phase 2): species, backgrounds, classes, feats, spells, equipment, monsters, and the content-source ports
- `character/` (planned): abilities, Proficiency Bonus, level, derived stats, spellcasting, equipment rule maths
- `combat/` (planned): initiative, actions, attacks, conditions, hit points, damage
- `toolbox/` (planned): Gameplay Toolbox rules; downstream of all the others, nothing depends on it

Update this section when the structure changes.

## Commands

Go is installed locally; no Docker is needed except for mutation testing. Run through the `justfile` (`just --list` shows the recipes):

```sh
just fmt        # gofmt
just vet        # go vet ./...
just lint       # golangci-lint run ./...
just test       # go test -race -shuffle=on -cover ./...
just arch       # architecture tests
just test-386   # GOARCH=386 go test ./content/... (32-bit integer width, no -race; CI runs it as its own job)
just check      # what CI runs: gofmt, vet, lint, coverage (cover-check), the 32-bit content tests, go mod tidy -diff, arch, fuzz-smoke
```

Mutation testing (`gomu`) is never run on the host. Run it only inside Docker
with CPU capped at 1/4 of the processors; the exact command is in
`doc/rules/testing.md`.
