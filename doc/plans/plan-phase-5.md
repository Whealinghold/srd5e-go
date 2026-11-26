# Phase 5 plan: breadth, the `toolbox` context and the v0.1.0 release candidate

| | |
|---|---|
| Status | Proposed. Nothing is built. Revised after three `plan-review` passes. |
| Roadmap phase | 5 ([../roadmap.md](../roadmap.md)) |
| Specs sequenced | Bulk data (rank 5 to 7, `content`): [content-spells](../specs/content-spells.md) with [spell-effect-data](../specs/spell-effect-data.md), [content-species-backgrounds-feats](../specs/content-species-backgrounds-feats.md), [content-equipment](../specs/content-equipment.md), [content-magic-items](../specs/content-magic-items.md), [content-monsters](../specs/content-monsters.md), and the class-list coverage of [content-classes-data](../specs/content-classes-data.md). `toolbox` (rank 40 and 41, 8 specs): [toolbox-hazard-effects](../specs/toolbox-hazard-effects.md), [toolbox-travel-pace](../specs/toolbox-travel-pace.md), [toolbox-environmental-effects](../specs/toolbox-environmental-effects.md), [toolbox-poison](../specs/toolbox-poison.md), [toolbox-traps](../specs/toolbox-traps.md), [toolbox-curses-and-contagions](../specs/toolbox-curses-and-contagions.md), [toolbox-fear-and-mental-stress](../specs/toolbox-fear-and-mental-stress.md), [toolbox-combat-encounter-difficulty](../specs/toolbox-combat-encounter-difficulty.md). Closing: [rule-options](../specs/rule-options.md), [rules-glossary-coverage](../specs/rules-glossary-coverage.md) (rank 91), [release-0-1-0](../specs/release-0-1-0.md) (rank 99). Applied inside every data step, with no step of its own: [content-licensing-and-notice](../specs/content-licensing-and-notice.md) (`NOTICE.md`). Rank table: `shared-types-and-ownership.md` §4.11 |
| Depends on | Phases 0 to 3 for the data track (steps 0 to 5). Phase 4 for the `toolbox` track (steps 6 to 7g) and for steps 8 to 10; the exact Phase 4 steps are in P1 |
| Format | follows [plan-phase-4.md](plan-phase-4.md) |

Naming note: `A1` to `A13` mean the **architecture checks** of `architecture.md`
section 9. Plan items are numbered `P1` onward. Steps are numbered by track.

## 1. Goal

Finish the library's breadth and prepare its first release: bundle the rest of
SRD 5.2.1 content (spells, equipment, magic items, monsters, the full species,
background and feat sets), ship the `toolbox` context (Gameplay Toolbox, SRD pp.
192-203), settle the optional-rule value objects, and end with a **release
candidate**. The roadmap's last bullet, "Tag v0.1.0", is step 12: it is a gated
step that an agent performs only on the user's explicit instruction in that
session (P12), because release-0-1-0 FR-063 reserves the tag to a maintainer.

This plan sequences decisions already made in the specs and in
[../specs/shared-types-and-ownership.md](../specs/shared-types-and-ownership.md)
(binding). It decides nothing; where a spec and this plan disagree, the spec
wins. Where the three review passes raised a choice no spec makes, section 4
records it as an **assumption** with a default the user can override, or as
**OPEN** where only the user can answer.

## 2. What already exists (reused as-is)

| Asset | Where | Used for |
|---|---|---|
| Decoder, `ValidateSet`, `Loader`, the `Problem` joined error, the revision table of kinds, `Kind` registry; `decode_origins.go` (backgrounds and feats already decode under `backgrounds` and `feats`, `KindBackground`, `KindFeat`); `hygiene.go` (`validName`, `validText`, `validLine`, `cleanCharacters`); `problem.go` (`QuoteToASCII` escaping, `MaxRenderedPath`) | `content/` (Phase 2) | every bulk document goes through the one decoder (shared-types FR-030 to FR-032); a new data file is a `content/data/*.json` document, never a second loader. New decoder rows call the hygiene and path-escaping helpers; they add no checks of their own |
| `data.BundledSource`, `WithDocuments`, `Files`, the pin-test pattern (`data_pin_test.go`, `data_<class>_test.go`), `checkDataLayout` (`internal/archtest/embedded.go`, flags any non-`.json`, non-test file other than the one `.go` file) | `content/data/`, `internal/archtest/` | each data step adds a document and its pin tests beside the existing ones; no change to `BundledSource` unless a spec says so |
| `species.json` (4 species), `spells.json` (73 spells), twelve `class_*.json`; no background or feat document exists yet | `content/data/` | step 1 extends `spells.json` to the full set; step 2 extends `species.json` and creates the background and feat documents with the existing decode keys; neither replaces an existing file |
| `content/data/testdata/srd/` fixtures and the attribution pointer | `content/data/` | fixtures stay synthetic or under `testdata/srd/` with the pointer (content-licensing-and-notice FR-027) |
| `TestSpellFilesArePureAndToolsIsNotImported` (`internal/archtest/spell_types_test.go:82`), `Load` and its `go.mod` parser (`load.go`, `Module.HasRequire`, `RuleGoMod`), `checkConsumerNames` (`names.go`), `checkTagUniqueness` (`ruletags.go`) | `internal/archtest/` | `Load` skips only `testdata`, `vendor`, `.x` and `_x` today; `tools/` does not exist. Step 0b changes the loader and scopes the two whole-tree walkers (P9) |
| `character.RuleOptions` (`character/rule_options.go`: `NewRuleOptions`, `StandardRuleOptions`, `With*`, `Equal`, `IsZero`) | `character/` | the Phase 3 options object; the specs now use the same name (P8) |
| `NOTICE.md` Scope and the `content/data/` coverage check (A12) | repo root, `internal/archtest` | each data step updates the Scope status and cited pages from the SRD's own Legal Information page (AGENTS.md), never from memory |
| *Planned by Phase 4, not yet present* (`combat/` holds only `doc.go` today): `Event`, `CombatantID`, `HitPointStatus`, damage and condition functions, `ConditionSet`, the Exhaustion rules, `combat.RuleOptions` | `combat/` (Phase 4 steps 1 to 3 and later) | the `toolbox` specs import these types upward only (shared-types FR-045); `toolbox` calls no `combat` use case (`architecture.md` §4) |
| `character` published language: the travel files (`travel_pace.go`: `TravelPace`, `Row()`, `RaisedOneStep`; `travel_rules.go`; `travel_mode.go`; `vehicle_speed.go`; `mounted_pace_boost.go`), `ExhaustionLevelReduced` (`rest_benefits.go`), `RestApplied` (`rest_event.go`), `CharacterLevel`, derived stats | `character/` (Phase 3) | `toolbox-travel-pace` (needs `exploration.md`), `toolbox-traps` (needs `derived-stats.md`); only value objects and domain functions, never a `character` use case |
| `d20.ResolveCheck`, rerolls, `RollModifiers`, `dice.Roller`, `dicetest`, `telemetrytest` | `d20/`, kernel | every save the toolbox rolls; test support. A `toolbox` call composes `d20` domain functions, as in plan-phase-4 P12 |
| `telemetry.StartTimer`, `WithDefaults`, `d20/signals.go` and `character/operation_signals.go` as patterns; `combat.Instrumenter` is **not** reused (unexported to `combat`) | `telemetry/`, `d20/`, `character/` | `toolbox` gets its own `signals.go` per the hazard spec; copy the shape, import nothing |
| `internal/archtest` `DefaultPolicy` rows, the `toolbox` row included (`policy.go:61`), the AST helpers (`load.go`, `decls.go`, `names.go`) | `internal/archtest/` | step 6 verifies the existing `toolbox` row and adds cases; steps 8 and 10 build on `archtest.Load` instead of a new loader |
| justfile `cover-check` (90 percent floor), `fuzz-smoke`, `test-386` (`./content/...` only), `mutate`; CI runs these through `just` | `justfile`, `.github/workflows/ci.yml` | tool-module recipes loop over `tools/*/` and call these; `test-386` gets the new package list; no second spelling |

New, with no existing home: `internal/archtest/testdata/` (steps 8 and 10 create it), `ExtractionOutput`, `tools/` (optional, P9), the `poison`, `trap` and `contagion` kinds.

No kernel change is expected. If a step finds one is needed it stops and goes to
the user. No Phase 1 to 4 public API changes either, except the additive decoder
limits of step 0a (P7) and the additive `Kind` rows of 7c to 7e (P10); a spec
that needs another is an amendment request recorded in the spec and sent to the
user.

## 3. Scope fence

**In scope:** the bulk data of every content kind named in the specs above;
the `toolbox` context (8 specs); the P8 options reconciliation; the
glossary-coverage check; the release-candidate work of `release-0-1-0` (public
API golden, examples, bridge docs, README, module-zip checks, release
checklist); the decoder limits of P7; the doc, policy and catalogue edits each
step needs.

**Out of scope** (each tracked where it already lives):

- Pushing the `v0.1.0` tag, unless the user instructs it (P12, step 12).
- Extraction and PDF-check tooling under `tools/`: **optional**, a separate
  follow-up PR per data step, never on the critical path (P9). Hand
  transcription with pin tests is the default route.
- The `content-classes-data` extraction tool, drift check and FR-091 log
  applier (FR-086 to FR-092): Phase 3 deferred them; they stay on that spec's
  "Not yet implemented" line. Only its class-list coverage (FR-100, FR-101)
  and two scenarios are in step 1.
- The open `architecture-guardrails` ids (the spec-set checks FR-015, FR-016,
  FR-052, FR-056 and the observability checks O1 to O5): repository tooling,
  not a roadmap deliverable. Recorded as a follow-up (section 10); O3 is not
  enabled here, so the catalogue is diffed by hand at review.
- Any Phase 4 requirement still open when Phase 5 starts; it stays on
  [plan-phase-4.md](plan-phase-4.md). Phase 5 steps that need it wait (P1).
- Anything not in SRD 5.2.1. Lair actions are not in the PDF and stay
  unmodelled (rules-glossary-coverage Q-20).
- A documentation site, a `/v2` path, shipped bridges to OpenTelemetry or other
  SDKs (release-0-1-0 section 1).
- A person's reading of the bundled text against the PDF is a gate (P4) but is
  not work an agent can do or sign off.

## 4. Open items, assumptions and spec defects

**Decided**: a spec or the user settled it. **Assumption**: a choice the plan
makes, with a default, that the user can override. **Spec defect**: fix in the
spec in the PR that hits it. **OPEN**: only the user can answer.

| # | Item | Class | Handling |
|---|---|---|---|
| P1 | Phase 4 and Phase 5 overlap. The data track needs only Phases 0 to 3 (data specs depend on `content-schema-and-decoder` and `content-source-port` only, shared-types FR-047). The `toolbox` track imports `combat`: `toolbox-hazard-effects` needs `combat-foundations`, `hit-points-and-damage`, `conditions` (Phase 4 steps 1 to 3); `toolbox-traps` also needs `environment-hazards-and-objects` (4c), `initiative-and-actions` (6a), `hiding-searching-and-senses` (4b); `toolbox-curses-and-contagions` also needs `magical-effects` (7b) and `rests-and-hit-dice`. Soft data dependencies: 7b reads equipment ids (step 3), 7g reads monsters (step 5) | assumption | the data track starts now, in parallel with Phase 4. Step 6 starts after Phase 4 step 3; 7d and 7e wait for the Phase 4 steps named. Where 7b or 7g needs data not yet bundled, those ids are D until the step lands. Override: wait for all of Phase 4 before any `toolbox` code |
| P2 | The SRD PDF is not in the repository | assumption | the default route is **hand transcription** from the cited pages, checked by pins whose oracle is not the data (P15). If the user supplies a PDF path at run time, an optional tool may draft or check (P9) |
| P3 | Classification of requirement ids as T (testable with synthetic fixtures or bundled data) or D (needs a person, a PDF check or an unmerged Phase 4 step) | assumption | the implementer re-checks every id; D ids go into the PR `Testing` entry and the spec's "Not yet implemented" line |
| P4 | Fidelity review of the bundled SRD text against the PDF (roadmap Phase 2 open item) | decided, partly done (the user assigned the review to the agent on 2026-11-10) | **Done for the existing bundle on 2026-11-10** against SRD 5.2.1 (pdftotext of the PDF at media.dndbeyond.com): 73 spells, 4 species, 12 classes with their subclasses (about 480 features), by seven independent reviewers; **no discrepancy found**. The only items raised were the Wizard "Expanding and Replacing a Spellbook" sidebar (p. 78, not feature text; modelled in `character` by class-wizard FR-035 to FR-039) and spell tables held in `facts.alwaysPreparedSpells` (they match). Limits, stated plainly: comparison was by reading and partial substring checks, not a full machine diff; the `facts` entries, spell-id resolution, `source.page` values for some files and the class spell lists were not all checked; it is an agent review, not a person's sign-off. **Still to do:** every document steps 1 to 5 add gets the same review in its PR, and a human spot-check of this review is advisable before the tag. The tag is not blocked by P4 alone (P12 still applies) |
| P5 | Content volume: 339 spells (266 to add), 258 equipment entries, 258 magic items, 330 monsters, 9 species, 4 backgrounds, 17 feats | assumption | split a document by SRD section (spells by level band, monsters by letter range) into reviewable PRs, each with the pins for the part it adds, plus a final PR per document that adds the whole-set pins. A split keeps the document one JSON file. Splits are an option, not a requirement; a PR is as large as a reviewer accepts |
| P6 | Package-level `var` for tables breaks A4 | decided: A4 wins | `const`, funcs and `switch` only; embedded JSON is the sole `go:embed` var |
| P7 | Library limits (plan-phase-4 P10 asks the same). No spec bounds decoder input or the host numbers `toolbox` takes | decided (numbers confirmed by the user 2026-11-10; design numbers, not SRD rules) | **Decoder (step 0a):** document size 16 MiB; nesting depth 32; entries per document 4,096; array length inside an entry 1,024; name 128 bytes; text field 16 KiB; invalid UTF-8, NUL, control and bidi-override characters (U+202A to U+202E, U+2066 to U+2069) rejected. **Dice:** at most 1,000 dice per expression, sides from the valid `dice.Die` set. **Collections and cascades:** the numbers of plan-phase-4 P10 (roster 256, and so on) apply to any `toolbox` collection. **Host numbers a spec ranges** (levels, hours, durations, counts) use the spec's range; where a spec gives none the PR amends the spec with a bound before the step ships. Each cap is a typed `invalid` rejection tested at cap and cap plus one. **Gate:** 7c, 7d and 7e (the new decoder trust boundaries) do not ship until step 0a is merged; the other `toolbox` steps may ship, recording any cap test deferred to the numbers in their `Testing` entry, and step 11 audits that every deferred cap test landed |
| P8 | Options-type naming: the specs named the three options objects `CharacterRuleOptions`, `CombatRuleOptions` and `ToolboxRuleOptions`, each of which repeats its package name (`go.md` §1 forbids that), while `character/rule_options.go` and `combat/rule_options.go` already ship `RuleOptions` | decided (user, 2026-11-10: names settled; P8 amendment approved) | all three contexts use `RuleOptions`: `character.RuleOptions`, `combat.RuleOptions`, `toolbox.RuleOptions`, with `StandardRuleOptions`, `NewRuleOptions` and `MustRuleOptions`. The specs were amended to these names (spec-only, no API change); `toolbox` follows the same pattern when step 6 lands |
| P9 | `tools/` extractors and PDF checks: each its own module (`go.mod`, stdlib only, no `replace`, no `go.work`), never imported by the root module (schema FR-124, content-spells FR-089, content-monsters FR-090) | assumption | **optional**, as separate PRs after the data they draft. Step 0b lands only when the first tool does and is its first PR. If the user prefers no tools in the repository, step 0b is dropped and the PDF-check items of the specs stay D |
| P10 | Revision-table rows. `content/decode.go` already registers the equipment arrays, `monsters` and `magicItems`; absent are the `poison`, `trap` and `contagion` kinds (`content/kind.go` reserves them), their typed ids and decoder rows. Order is fixed (decoder FR-142: poison, trap, contagion). Their **entry types live in `content`** (`PoisonEntry` and the trap and contagion entries, since `content` cannot import `toolbox`); the definitions and rules live in `toolbox` | spec defect | the first of 7c, 7d, 7e to land declares the whole ordered block of three `Kind` constants and the `srd5e.content.kind` values in `observability.md` §5 and in the `entries_loaded` and `decode_errors` lists of `signals.md`; the later two add their decoder rows. A decoder test pins that a host document of a kind whose decoder has not landed is rejected cleanly. A `Kind` rename or removal after the tag is BREAKING |
| P11 | The glossary check's Warning (FR-022) is due to be promoted to a failure no later than 2026-12-31 (rules-glossary-coverage Q-8); today is 2026-11-10 | decided (the spec) | step 8 promotes it in the change that accepts the spec. If the date is at risk, the promotion ships as its own small PR **once the table is already green** (no P14 gate on that PR alone) |
| P12 | The roadmap lists "Tag v0.1.0"; release-0-1-0 FR-063 reserves the tag to a maintainer | **OPEN** (needs the user) | default: step 12 exists and an agent does not run it unless the user instructs it in that session; skipped while P4 or P14 is unmet. The user may instead confirm the tag is out of scope |
| P13 | Existing hard-coded pins break on the first data PR: `content/data/data_pin_test.go` (per-document `want` map around line 63, 4 species, 73 spells, 89 total, the `entries_loaded` kind map around line 305, 12 classes with 174 and 58 features around line 412), `bundled_source_test.go` (lines 106 and 171, including `len(bad.ReferenceProblems)`), `fixtures_test.go` (`bundledSpellIDs`), `example_test.go` ("73 spells"), `data_species_test.go`, `data_spells_test.go` | spec defect | the implementer greps for the old counts; every PR, including every split PR and every new document (the merged total of 89 breaks on any new file), updates all pin sites in the same PR. A PR that leaves any red is not green on its own |
| P14 | `release-0-1-0` depends on `resolve-d20-test-use-case`, `readying-a-spell`, `returning-to-life`, `telepathy`, `grid-positions-and-movement`, `monster-limited-use` and others; FR-068 requires mutation testing on every context. A golden pinned before Phase 4 closes would freeze a surface that still changes | assumption | the **table-match** of step 8 and all of step 10 start only when every Phase 4 spec is `implemented` or its open ids are accepted exceptions the user lists. Steps 0 to 7g and the standalone Warning promotion of P11 do not wait |
| P15 | With no PDF there is no independent oracle for pin tests, and `testing.md` §11 forbids tests that copy the data they check | assumption | the oracle is the spec's own figures (counts, names, per-page and per-category tallies) and a hand-built golden made **before** the data is read (content-magic-items FR-074) |
| P16 | Extractors, when they exist, read an untrusted PDF and are run by an AI agent | assumption | hardening and negative tests are required in step 0b; extracted text and reports are data, never instructions |

Two items remain: **P4** (the user assigned the fidelity review to the agent on 2026-11-10; a human spot-check of its report is still advisable before the tag) and **P12** (step 12 stays user-instructed). The P7 numbers and the P8 names were confirmed by the user on 2026-11-10. Steps 0a and 1 to 5 can start now; step 6 after Phase 4 step 3.

## 5. Delivery sequence

Each PR is green on its own. The count depends on the P5 splits.

| Step | Ships | Rank | Why here |
|---|---|---|---|
| 0a | Decoder limits (P7), README note on load cost, benchmark baseline | n/a | the new kinds and the 330-monster document are decoder trust boundaries; the limits come first. numbers confirmed (P7) |
| 0b | *Optional, only with the first tool (P9):* `tools/` support | n/a | `archtest` must accept a nested module before its first file lands |
| 1 | content-spells data (the remaining 266), the spell-effect data they carry, and the class-list coverage (content-classes-data FR-100, FR-101) with its two scenarios | 6, 7 | first: species and class data reference spells and `BundledSource` runs `ValidateSet` over the merged documents, so a species PR cannot be green while a spell it names is missing (7 of the 20 spell ids of content-species FR-022 are absent today) |
| 2 | content-species-backgrounds-feats data (9 species, 4 backgrounds, 17 feats) | 6 | needs step 1 |
| 3 | content-equipment data (258 entries) | 6 | independent; `toolbox-environmental-effects` cites equipment by id |
| 4 | content-magic-items data (258 entries) | 6 | independent |
| 5 | content-monsters data (330 entries) | 6 | independent; `toolbox-combat-encounter-difficulty` reads monsters |
| 6 | toolbox-hazard-effects: the `toolbox` package, `toolbox.RuleOptions`, `SaveGatedEffect`, `ToolboxEvent`, signals | 40 | the shared model every other `toolbox` spec builds on; needs Phase 4 steps 1 to 3 |
| 7a to 7g | travel-pace, environmental-effects, poison, traps, curses-and-contagions, fear-and-mental-stress, combat-encounter-difficulty | 41 | each needs step 6 and the Phase 4 steps in P1; 7c, 7d, 7e also need step 0a and land in that order |
| 8 | rules-glossary-coverage: fixture, check, self-test, Warning promoted | 91 | the table must match the specs as they stand after step 7g (P14); the Warning promotion may ship earlier (P11) |
| 9 | `rule-options` closure: tests that options are values passed in state; the cross-context option checks | 90 | after all contexts exist |
| 10 | release-0-1-0: public API golden, `Example*` for the main use cases, bridge docs, README coverage table, CHANGELOG, module-zip and release-checklist checks | 99 | last; describes what exists (P14) |
| 11 | Closing: roadmap, documentation index, spec status audit, re-run of D ids | n/a | closes the phase |
| 12 | Tag v0.1.0 | n/a | only on the user's instruction (P12), after step 10's gates pass and P4 is met |

Natural stopping points: after step 5 (all SRD content bundled), after step 7g
(the full feature set), after step 10 (a release candidate).

### Process contract for every step

- Read the four rule files first (`AGENTS.md`); never quote a `RULE-TAG` in a plan
  or PR text (A13).
- Every new test cites its requirement (`Requirement:` and `Path:`; `per case`
  on tables). Apply `testing.md` §5.
- Each PR carries a **requirement-to-test list**: every T id of the spec maps to a
  test; D ids and any id deferred, **cap tests included (P7)**, are named in the
  `Testing` entry.
- `just check` green. Each changed package at least 90 percent statement coverage,
  reported per package. `GOARCH=386 go test ./content/...` stays green (the
  `test-386` recipe's package list is extended to `./combat/... ./toolbox/...` when
  those packages have tests; the recipe is the one spelling).
- Mutation testing with `gomu`, only inside Docker, `--cpus` and `--workers`
  capped at a quarter of the processors (`just mutate ./<path>.go`, one file at a
  time; a package-level `just mutate ./toolbox` run before step 11). Score at least
  80 for domain files; survivors fixed unless provably equivalent; manual fallback
  of `testing.md` §8 where gomu cannot parse a file. Data JSON is not mutated; its
  guard is the pins. Tool-module parsers are exempt from `gomu` (the recipe is
  root-only) and are held to coverage, `-race -shuffle=on` and fuzzing instead
  (assumption).
- `CHANGELOG.md` entry under `Unreleased`. The PR description carries `Testing`
  and `Observability` entries, the latter answering **all seven questions** of
  `observability.md` §2.
- Signals: every new signal is a constant in the context's `signals.go`, a row in
  `doc/observability/signals.md` with purpose and a **filled** suggested
  monitoring column, and a pin test. New attribute values go into
  `observability.md` §5 and the matching `signals.md` attribute lists in the same
  PR (tag refreshed). A test pins that a host-supplied unknown kind or id never
  becomes an attribute value.
- A spec's `Status` becomes `partially implemented` in the PR that first ships
  code or data for it, with an `Implementation status` list and a `Not yet
  implemented:` line; `implemented` only when that line is empty.
- Pure domain files import no `telemetry` (O1); use-case files hold every
  `telemetry` and `srderr.WrapPort` call (A6). A use case never calls another
  use case (`architecture.md` §4).
- **Unhappy paths** (`testing.md` §4) for every use-case step: unknown id, a
  failing and a panicking roller and content source, a cancelled `ctx`, inputs
  unchanged after a refusal or panic, returned slices are copies, Advantage and
  Disadvantage together where a roll is involved, boundaries at, one below and
  one above every threshold of the spec.
- **Fuzzing:** every new `NewX` or `ParseX` taking untrusted text or numbers has a
  `Fuzz*` test, and so does every new content decoder (seeds include nested,
  oversize and duplicate-key documents); tool parsers too, in their module.
- **Data steps (1 to 5) additionally:** bundled text only from the cited SRD
  pages and kept faithful; `NOTICE.md` Scope status and cited pages updated from
  the Legal Information page; counts pinned (spec figures: 339 spells, 258
  equipment entries, 258 magic items, 330 monsters) as the expected
  `srd5e.content.entries_loaded` baseline per kind; every cross-reference
  resolves under `ValidateSet`; the id slug rule of content-equipment FR-003; no
  entry whose name is not a heading in the cited pages; pins per P15 and P13;
  the 32-bit decode; decoded free text through the `content` hygiene helpers; an
  operation-count test of every cross-reference pass that could be quadratic
  (class lists against spells in step 1, monster and spell references in step 5);
  the load benchmark must show time per entry at the full bundle no worse than
  twice the step 0a per-entry baseline (linear scaling); the loader's per-entry
  logging stays off (`observability.md` §7); `load_duration` suggested
  monitoring is re-baselined from the benchmark in the step 5 PR.
- **Fixtures** are synthetic, or under `testdata/srd/` with the attribution
  pointer; nothing from outside SRD 5.2.1; no consumer names.
- **Acceptance bar for every step:** every T id has a test, the unhappy paths are
  covered or justified, and D ids are listed.

### 5.0a Step 0a: decoder limits and housekeeping

- The P7 decoder caps in `content`'s decoder, as additive typed `invalid`
  rejections (`ErrInvalidInput`), using the existing `Problem` path escaping.
- (P8 is done: the spec amendment landed with the plan revision.)
- `README` note that `NewBundledSource` decodes and validates the whole set on
  each call and should be built once and shared (the bundle grows about tenfold);
  a `BundledSource` load benchmark whose per-entry baseline step 1 onward is held
  to.

**Tests:** each cap at the cap and cap plus one; invalid UTF-8, NUL, a bidi
override and a control character in a name and in a text field; nesting at 32 and
33; a 32-bit run; fuzz of the decoder with oversize and deeply nested seeds.
**Observability:** the seven questions. Failure mode: a document over a limit is
an `invalid` result with the problem path and kind (existing load signals, no new
signal); path and kind are bounded and escaped, ids never attributes. Monitor any
`content` decode failure after upgrade. **Rollback:** revert the PR (caps are
additive and no data depends on them yet; after 7c to 7e land, revert those
first). **Acceptance:** `just check` and the 386 run green.

### 5.0b Step 0b: `tools/` support (optional, P9; the first PR of the first tool)

- Change `archtest.Load` to skip any directory that holds its own `go.mod`
  (otherwise every file under `tools/<x>/` is loaded as an unlisted root package,
  A1, and the A3 deny-list is applied to it). Scope `checkConsumerNames` (A10) and
  `checkTagUniqueness` (A13), which also walk every file under `tools/`, or state
  that tool content must pass them.
- Layout cases: `tools/` holds only `go.mod`-rooted modules; each module's
  `go.mod` has no `require` (reuse `Load`'s parser), no `replace`, no `go.work`.
  `checkDataLayout` already flags stray non-JSON files in `content/data/`; this
  step adds only a stray-`.json` rule if the spec wants one.
- `architecture.md` §1 gets a `tools/` row (outside the module, never imported);
  refresh the tag.
- `just tools-test`, looping over `tools/*/` and calling the existing
  `cover-check` and `fuzz-smoke`; a CI job that needs no PDF, secret or network
  and gets no token on fork PRs. It is **advisory** until the first module's tests
  exist in the same PR, then required.
- Hardening, with a negative test for each: `pdftotext` runs in a Docker image
  pinned by digest with `--network none`, a read-only PDF mount, a non-root
  user, `--pids-limit`, `--cap-drop ALL`, `no-new-privileges`, a read-only root
  filesystem and a tmpfs-only writable path, CPU, memory and time limits, a cap
  on PDF size and page count and on `pdftotext` output size; the Go side uses
  `exec.Command` with a fixed argument list, no shell, and `--` before the path;
  output file names are constants; both the report path and the `content/data/`
  output are cleaned, must stay inside a given directory and are opened without
  following symlinks; no PDF-derived text (a monster name containing `../` or a
  NUL) ever reaches a file name or path; reports hold the PDF base name, page
  numbers and counts only, never absolute paths or excerpts.

**Tests:** a root package importing `tools/` (violating); a nested module with a
banned import stays clean (proves the loader skip); a tool `go.mod` with a
`require` (violating) and without (clean); and for the hardening: path traversal,
a symlinked output and report directory, a leading-dash path, oversize output, an
oversize PDF, a crafted `../` and NUL name, and a report that contains no
free-text excerpt. **Observability:** the seven questions. A tool imported by the
library or writing outside its two paths fails `just arch` or its own test;
nothing to count or time; no existing signal affected. **Rollback:** revert the
PR. **Acceptance:** `just arch` and `just tools-test` green.

### 5.1 to 5.5 Steps 1 to 5: bulk data

| Step | Spec | Adds | Named boundaries and checks |
|---|---|---|---|
| 1 | content-spells | the remaining 266 spells to the full 339 in `spells.json`; the spell-effect data they carry; class-list coverage (content-classes-data FR-100, FR-101) and its two scenarios (a feat taken at level 4, a Wizard scroll copy) | 339 names pinned; 189 spells with a Material component, 56 with `costsGp`, 31 with `consumesMaterial`; class lists from the eight spellcasting classes; spell-effect decoder negative rows (one per problem kind, all problems reported) and fuzz seeds |
| 2 | content-species-backgrounds-feats | the nine species (extends `species.json`), four backgrounds, 17 feats (categories 4/2/4/7) as new documents using the existing `backgrounds` and `feats` decode keys | decoded ids equal the lists; every species Humanoid; trait names; referenced spell ids resolve (the 20 the spec names) |
| 3 | content-equipment | `equipment.json`, 258 entries in one document; tools; the Craft cross-references; the Potion of Healing row and the four magic entries named by FR-094 and FR-095 | counts per category; slug rule; every cross-reference resolves; Gaming Set and Musical Instrument as `ToolFamily` with 4 and 10 variants |
| 4 | content-magic-items | `magic_items.json`, 258 entries | per-page and per-category counts; id length at most 64; no entry duplicated in equipment (FR-095) |
| 5 | content-monsters | `monsters.json`, 330 entries | CR ordering over all 34 values; ability scores at 0, 1, 30 and 31; invalid CR and negative XP rows; the stat-block rows of the spec (speeds, qualifiers, recharge, multiattack) |

**Tests (all data steps):** unit for decoder revision rows (P10), data pins (P15),
contract (`contenttest.RunSourceContract` over the real `BundledSource`), the
adapter-level decoder tests and whole-set check functions with corrupted-input
cases and the stale-exception assertion that the data specs list in their §8, the
`BundledSource` scenario for monsters (`monster_index`), the golden fixture where
the spec defines one, the 32-bit decode, the operation-count and benchmark gates
above. **Observability:** data specs are silent by design (decoding is in-memory,
no clock, so the specs add no signal); the failure mode is bad bundled data,
surfaced by the existing `content` load signals as a failure result with the
problem path and kind; the entry ids stay out of attributes. Monitor: any
`content` decode failure after an upgrade, and `entries_loaded` per kind against
the pinned counts. **Rollback:** each PR reverts alone **in reverse dependency
order**: step 1 cannot be reverted while step 2 (species) is in, step 3 not while
7b cites equipment, step 5 not while 7g cites monsters, and a later split PR of a
document not before the PRs built on it. Removing a document changes no API.
**Acceptance:** per PR, every T id of the spec's part has a test and the P4
checklist is attached.

### 5.6 Step 6: toolbox-hazard-effects (rank 40)

The `toolbox` package: `doc.go`, `toolbox.RuleOptions` (`FearOptions`, `FearRider`,
`MentalStressOnSuccess`; FR-099 to FR-104) and `toolbox.StandardRuleOptions`,
`SaveGatedEffect` and its parts, `ActiveSaveGatedEffect`, `SaveGatedConsequences`,
`RollSaveGatedDamage`, `ResolveSaveGatedEffect`, `ToolboxEvent`, the `toolbox`
signal constants. Also: the `DefaultPolicy` row for `toolbox` and the §2
context-map row already exist and are **verified, not re-added**; this step adds
the missing §1 package row, removes the "reserved" wording, updates `AGENTS.md`
(structure, "Bounded contexts") and refreshes the tag, with archtest cases for the
existing row (`toolbox` importing `*test` packages in production files violates; a
`combat` file importing `toolbox` violates) and the exported-name collision scan of
hazard FR-002 and FR-006 (no clash with `combat.Encounter` and similar).
**Tests:** unit for every save-gated rule row with boundaries (save total at the
DC, one below, natural 1 and 20, half damage on a success, resistance
interaction); application tests for each use case with `dicetest`,
`telemetrytest`, a failing and a panicking roller, a cancelled `ctx`, state
unchanged after refusal; the `Standard*` constructors; the signal pin tests; the
`scenarios/` end-to-end test the hazard spec's §8 assigns to Phase 5 (apply damage
through `ApplyDamage`, Poisoned through `ApplyConditionToSet`, six successes, a
seventh ends it). No `toolbox` use case calls another and none calls
`d20.Resolver` (plan-phase-4 P12); a recording test pins that no `srd5e.d20_test.*`
signal fires. Dice and damage sums: a count at the P7 limit and one over,
multiplication and sums at `math.MaxInt` and `math.MaxInt32`.
**Observability:** the seven questions from the spec's section 7; the `toolbox`
operation and result signals are defined here and reused by 7a to 7g; this PR
lists the signals, constants, `signals.go` and attribute values it adds to
`observability.md` §5 and `signals.md`, with a filled monitoring column.
**Rollback:** revert only after reverting 7a to 7g; after v0.1.0 a rename is
BREAKING. **Acceptance:** `just check` green; all T ids tested.

### 5.7 Steps 7a to 7g: rank 41

| Step | Spec | Notes |
|---|---|---|
| 7a | toolbox-travel-pace | needs `exploration.md`; **reuses** the existing `character` travel files (§2) and checks each reused symbol is a value object or domain function, never a `character` use case; reimplements none |
| 7b | toolbox-environmental-effects | high altitude, extreme cold and heat; Exhaustion and condition results handed to the host; equipment ids from step 3 (D until bundled) |
| 7c | toolbox-poison | the `poison` kind: entry type and decoder rows in `content`, rules in `toolbox` (P10); its data ships here, with count pins and the golden table 4.11 (poison FR-017 to FR-019) |
| 7d | toolbox-traps | `TrapDefinition` in `toolbox`, the trap entry and decoder rows in `content` (P10); level bands; needs Phase 4 4b, 4c, 6a |
| 7e | toolbox-curses-and-contagions | the `contagion` entry and decoder rows in `content` (P10); needs Phase 4 7b (`magical-effects`) and `rests-and-hit-dice`; 137 ids, split into types then rules if too large |
| 7f | toolbox-fear-and-mental-stress | uses step 6's options types |
| 7g | toolbox-combat-encounter-difficulty | reads `content-monsters` types only; needs step 5's data for any "over the bundled monsters" requirement (D until then); adds the archtest of its spec FR-028 (its files import none of `combat`, `character`, `d20` although the policy row allows them) |

**Tests:** as step 6, plus: 7c, 7d and 7e (new decoder trust boundaries) get
negative decoder tests, one row per problem kind with all problems reported at
once (poison FR-082 to FR-084), boundary entries accepted and rejected, and a
`Fuzz` over each new decoder (traps FR-093, FR-094); a host document reusing a
bundled id is rejected as a duplicate, never merged; reference cycles and
self-reference (a contagion that ends in another, a curse that applies itself, a
trap chain, a cycle that crosses sources such as host A to bundled B to host A, and
one that crosses kinds such as contagion to curse to poison) are rejected
iteratively with a visited set, and reference resolution at use time carries the
same bound; every host number that drives a loop or multiplication (dice count and
sides, durations, travel hours, roster size, trap counts) is range-checked in
`New*` and the decoder to the P7 bound, with `math.MaxInt32` rows for the 386 run,
float-to-int rows (`1e10`, `1.5`), negation of the minimum, and division and
rounding rows (XP per character, zero or negative hours). Each step names one
`scenarios/` test its spec plans, or says why it has none (7a has none). Named
boundaries: 7a a pace at each distance threshold and the rounding edge; 7b a save
at the DC for each environment, Exhaustion at the cap of 6; 7c a poison save at the
DC, the damage at the minimum and maximum of the dice; 7d a trap at each level-band
edge (levels 1, 4, 5, 10, 11, 16, 17, 20); 7e a curse that ends and one that does
not, the cumulative success count, the margin comparison (`>=` against `>`) and the
Exhaustion reduction at level 0 and at the current level (FR-108); 7f the three
Mental Stress rows and the option default; 7g an encounter exactly at the budget,
one under and one over, a roster of zero monsters, a party level of 0, 1, 20 and 21,
an empty party, an invalid CR, duplicate and unknown monster ids. Missing-content
rows for the catalogue kinds (unknown poison or trap id, a failing and a panicking
source).
**Observability:** each follows its spec's section 7 and reuses step 6's signals;
each PR states which signals or attribute values it adds (for example the
`content.kind` values for poison, trap and contagion, a closed set in
`observability.md` §5 and in the `entries_loaded` and `decode_errors` lists of
`signals.md`) and fills the monitoring column, so nothing is first written at
phase close; a refusal is a returned result, not an error; a roller failure is
`error.type` once. The 7g catalogue row names the attribute that tells a
missing-content refusal from invalid input, so a consumer can tell bad content from
bad input. Monitor the refused and `invalid` shares after a content change.
**Rollback:** 7a, 7b, 7f and 7g each revert alone; **7c, 7d and 7e revert in
reverse landing order** because they share the P10 `Kind` block. **Acceptance:**
per PR, every T id tested.

### 5.8 Step 8: rules-glossary-coverage (rank 91)

The fixture `internal/archtest/testdata/srd/rules_glossary_entries.json` (155
records, no rule text; `internal/archtest/testdata/` is new), the check, its
self-test with one injected fault per case, the totals test, and the Warning
promoted to a failure (P11). Table rows that cite a spec this phase changed are
updated in the same PR. Gated by P14 except the standalone promotion.
**Tests:** the spec's section 8 cases, including empty and one-entry tables, a
withdrawn FR id, and deterministic ordering. **Observability:** silent by design:
a repository check with no runtime; the failure is a `go test` failure naming the
row. **Rollback:** revert the PR. **Acceptance:** `just check` green with the check
on; a person has re-read pp. 176-191 against the fixture (FR-027, D until then).

### 5.9 Step 9: rule-options closure

Tests that `character.RuleOptions`, `combat`'s options and `toolbox.RuleOptions` are
values passed in state, not globals, that each `Standard*` constructor returns the
SRD baseline, and that the zero value is not assumed to be the baseline
(`rule-options.md` §3). The P8 amendment was made in step 0a; this step checks that
the specs and code agree. **Tests:** unit per options object, one row per option the
SRD names with its page. **Observability:** silent by design: value objects with no
runtime path; a wrong default is caught by the table tests. **Rollback:** revert the
PR. **Acceptance:** every T id of `rule-options.md` tested.

### 5.10 Step 10: release-0-1-0 candidate (rank 99)

The public API golden (`internal/archtest/testdata/public_api.golden`, compared byte
for byte, regenerated only behind an explicit `-update` flag declared in the test,
and CI runs without that flag) and its check, built on `archtest.Load`; `Example*`
functions for the main use cases of every context; the observability bridge
documentation (documented, not shipped, compiled once in a scratch module,
release-0-1-0 FR-030); the `README.md` coverage table (what is included and what is
not); the `CHANGELOG.md` release section; the module-zip and `go.mod` checks,
including that the zip contains no `tools/` content and no PDF-derived file outside
`content/data/`; the release checklist as runnable gates. The tag is not pushed.
**Tests:** the spec's section 8 plan: examples run by `go test`, the golden diff
with a violating case and a test that the golden is rewritten only behind `-update`,
violating cases for the README-claim, module-zip and `go.mod` checks, the 32-bit
run, and `just mutate ./toolbox` at package level. **Observability:** the bridge
docs list the signals a consumer should map; the catalogue equals the constants,
diffed by a reviewer by hand (O3 is not enforced here, section 10). **Rollback:**
revert the PR before the tag. After a tag the module is immutable: recovery is a new
patch version and a CHANGELOG note (a `retract` directive only as release-0-1-0
FR-077 allows). **Acceptance:** every release gate the spec lists runs and the
result is recorded in the PR; the maintainer decides on the tag.

### 5.11 Step 11: closing the phase

Roadmap (Phase 5 status, implementation roll-up), `doc/documentation_index.md`, a
spec status audit (each spec of the header has an accurate `Status` and "Not yet
implemented" line), a re-run of every D id that waited on Phase 4, on step 3
(equipment) or on step 5 (monsters), and an audit that every cap test deferred in a
`Testing` entry (P7) has landed.
**Acceptance:** `just check` and `GOARCH=386 go test ./content/... ./combat/... ./toolbox/...`
green (once Phase 4 is done, P14); no D id left that depended only on Phase 4 or on
these data steps.

### 5.12 Step 12: tag v0.1.0

Not performed by an agent unless the user instructs it in that session (P12).
Preconditions: step 10's gates pass, P4 is met or explicitly waived, P14 is met.
The tag is `vX.Y.Z` on the merge commit (release-0-1-0 FR-074).

## 6. Verification commands

```sh
just check
just tools-test                     # only once a tool exists (P9)
just test-386                       # package list extended per the process contract
go test -race -shuffle=on -coverprofile=coverage.txt ./toolbox/... && go tool cover -func=coverage.txt | tail -n 1
just mutate ./toolbox/<file>.go     # Docker only, capped at a quarter of the processors, one file at a time
just mutate ./toolbox               # package level, before step 11
```

## 7. Security review

Phase 5 adds no network, storage, identity or secret. Its inputs are **bundled
JSON** (decoded by the Phase 2 decoder), **host JSON** through the same decoder,
**values the host passes in** to `toolbox`, and, only if a tool exists (P9), **a
local PDF path**.

| Concern | Worst case | Handling |
|---|---|---|
| Hostile or corrupt host-supplied content documents | a crash, or an invalid entry accepted | all content goes through the one decoder, which reports every problem; no `map[string]any` reaches the domain; fuzz targets for every new decoder with nested, oversize and duplicate-key seeds, so a panic in a per-kind validator is caught; `ValidateSet` for cross-references |
| Large or deep documents | memory or CPU exhaustion; `ValidateSet` cross-reference passes (class lists against 339 spells, contagions and traps against each other) turning quadratic | the P7 decoder caps of step 0a (size, depth, entries, array length); an operation-count test of each cross-reference pass; the linear-scaling benchmark gate; 7c to 7e do not ship before step 0a |
| Arithmetic on host or content numbers (travel distances, damage dice, XP, levels, durations) | overflow wraps; a huge dice count or duration is a CPU and memory denial of service | bounds from P7 and the specs, enforced in `New*` and the decoder (not only overflow-tested); `math.MaxInt` and `math.MaxInt32` rows; checked multiplication chains (pace times hours, XP times count, level-band sums); float-to-int and minimum-negation rows |
| Dice and randomness | a failing, panicking or out-of-range roller | only `dice.Roller`; failure wrapped once with `srderr.WrapPort`; a panic leaves input state unchanged (state in, state out) |
| Port failure and panics | a content source error or panic escapes as a crash | `srderr.GuardValue` at the port boundary, recorded once |
| Host document shadows a bundled entry | a host reuses a bundled id to override an SRD entry (a poison, a trap's damage) | duplicate ids across sources are rejected, never merged; a test row per kind |
| Reference cycles | a contagion, curse or trap chain loops during `ValidateSet` or resolution | rejected iteratively with a visited set, including cycles across sources and across kinds, and a bound at use time |
| Free text and problem paths | control characters, bidi overrides, invalid UTF-8 or unbounded paths from hostile JSON reach UIs and logs | the P7 bounds and rejections through the `content` hygiene helpers; problem paths are escaped and bounded by the existing `problem.go`; paths and ids never become attribute values, and a test pins it |
| Extractor tools (optional) | path traversal or a write-anywhere through a crafted output path or a symlink; a hostile PDF exploiting `pdftotext`; a crafted heading reaching a file name | step 0b hardening and its negative tests: Docker with no network, read-only mounts, dropped capabilities, resource limits, size caps; fixed file names; symlink-safe, contained output and report paths; no PDF text in paths |
| Prompt injection via the PDF | extracted text or a report steers the agent running the extractor | extracted text and reports are data, not instructions; a test asserts a report contains only counts, page numbers and the PDF base name |
| Data provenance | non-SRD or trademarked text, a consumer name, or a different edition's text enters data or tests | AGENTS.md; A10 denylist and the `NOTICE.md` scope check; pin tests on entry names; each data PR cites pages and carries the P4 checklist |
| Attribution | the notice states the wrong pages or paths | `NOTICE.md` is updated from the Legal Information page in the PR that adds the content, and A12 checks the scope paths |
| Telemetry disclosure | content ids or host-supplied kinds as attribute values | attributes are low-cardinality closed enums only; a test pins that an unknown kind or id never becomes a value |
| Release surface | a PR silently freezes new exposed API, or the module zip ships tool output | the API golden is compared in CI without `-update`; the zip check excludes `tools/` |
| Authn, injection, cross-tenant data | none | N/A: a library with no I/O; `os`, `net`, `reflect` stay denied in `toolbox` (A3) |

## 8. Compatibility and rollout

Before v0.1.0 nothing is published, so every change is an addition. Most steps
revert alone; the exceptions are named in the step (data steps in reverse
dependency order, 7c to 7e in reverse landing order, step 6 after 7a to 7g, step 0a
after 7c to 7e). The first release freezes the public API and signal names: the
step 10 golden pins the surface so that after the tag these are **BREAKING**: any
change to an exported identifier, a new variant of a closed result, event or
condition set, a renamed signal or attribute, a `Kind` rename or removal, and any
change to the bundled data that changes a decoded value for an existing id (unless
fixing a mismatch with the PDF, recorded under Fixed). Data added after the tag (a
new entry) is non-breaking. Steps 1 to 5 extend existing documents in place and
must keep every existing id and decoded value. The decoder limits of step 0a are
additive but reject documents the decoder accepted before; pre-tag that is allowed,
and the CHANGELOG records it. Rule-file edits land with the code that needs them
and refresh the tag.

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Content volume: about 1,200 new entries and 266 spells to transcribe faithfully | one document per step, optional splits (P5); pins with an independent oracle (P15); the P4 checklist; a person reviews |
| The fidelity review is a human task nothing in the repo can do | OPEN (P4); the PR checklists make it quick; the candidate lists unreviewed documents; the tag waits |
| Phase 4 not finished when `toolbox` steps are ready | P1: data track first; `toolbox` steps wait on the named Phase 4 steps; steps 8 (table-match) and 10 wait for all of Phase 4 (P14) |
| The `toolbox` context is created late and its first step touches the architecture tables | step 6 edits `architecture.md`, `AGENTS.md` and archtest cases together (the "changing the map" rule) |
| Hostile or oversized documents | step 0a limits before the new decoders; fuzzing; operation-count and benchmark gates |
| Extractor tooling bloats the repository or leaks into the library | optional and separate (P9); nested modules, archtest cases, no root dependency |
| Spec drift while data is bundled | each PR re-reads the spec's "Not yet implemented" line; the glossary check (step 8) |
| A pin test that merely copies the data it checks | oracle per P15; the golden is made independently |
| The glossary Warning deadline of 2026-12-31 (P11) | the standalone promotion PR |

## 10. Out-of-scope observations (recorded, not acted on)

- The `Requirement:` comment check (`testing.md` §9) is still unmechanised; this
  plan relies on review.
- **Follow-up plan candidate:** the open `architecture-guardrails` ids (spec-set
  checks FR-015, FR-016, FR-052, FR-056; observability checks O1 to O5, with O3
  enabled only once `signals.md` matches the constants) and the `content-classes-data`
  extraction tool, drift check and FR-091 log applier (FR-086 to FR-092). Neither is a
  roadmap deliverable; both stay on their specs' "Not yet implemented" lines.
- The release candidate's lower-bound `go` version test and the pkgsite check
  (release-0-1-0 FR-066, FR-073) need a network or a toolchain the agent may not
  have; they stay maintainer steps if so.
- If step 3 (equipment) is late, 7b ships with synthetic equipment and its
  bundled-data ids are D.
