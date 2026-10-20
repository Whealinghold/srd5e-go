# Phase 2 plan: the `content` context

| | |
|---|---|
| Status | implemented through step 7 (code, scenarios, examples, README, changelog, roadmap). No spec is `implemented`: each stays `partially implemented` with its own "Not yet implemented" line. Remaining: the manual fidelity review of every long text against the PDF (thin-slice FR-079), and the mutation run recorded by its own phase |
| Roadmap phase | 2 ([../roadmap.md](../roadmap.md)) |
| Specs sequenced | [content-schema-and-decoder](../specs/content-schema-and-decoder.md), [content-source-port](../specs/content-source-port.md), [content-species-backgrounds-feats](../specs/content-species-backgrounds-feats.md), [content-classes](../specs/content-classes.md), [content-equipment](../specs/content-equipment.md), [content-magic-items](../specs/content-magic-items.md), [content-monsters](../specs/content-monsters.md), [spell-effect-data](../specs/spell-effect-data.md), [content-spells](../specs/content-spells.md), [content-thin-slice](../specs/content-thin-slice.md), [content-licensing-and-notice](../specs/content-licensing-and-notice.md) (NOTICE part only) |
| Depends on | Phases 0 and 1 (kernel packages, `telemetry`, `srderr`, `internal/archtest`, `d20` as the use-case template) |
| Format | follows [plan-phase-1.md](plan-phase-1.md); there is no `doc/plans/README.md` |

Naming note: `A1` to `A13` in this document always mean the **architecture
checks** of `architecture.md` section 9. Plan items are numbered `P1` onward.

## 1. Goal

Ship the `content` bounded context: the types every later context imports, the
strict JSON decoder, the `Source` port with its adapters and test support, and a
17-entry bundled slice of SRD 5.2.1 (4 species, the Wizard, 12 spells) that
proves the schema end to end.

This plan sequences decisions already made in the specs and in
[../specs/shared-types-and-ownership.md](../specs/shared-types-and-ownership.md)
(binding). It decides nothing; where a spec and this plan disagree, the spec
wins. Gaps found while planning are in section 4 for the user.

**Scope was chosen by the user:** everything the specs mark `Types 2`, not only
the roadmap's bullet list. So the **types** of species, backgrounds, feats,
classes, equipment, magic items, monsters, spell effects and spells all ship now
([shared-types-and-ownership.md](../specs/shared-types-and-ownership.md) FR-014),
but **bulk data does not**: only the 17-entry slice is bundled.

## 2. What already exists (reused as-is)

| Asset | Where | Used for |
|---|---|---|
| `srderr.PortError`, `WrapPort`, `GuardValue`, `ErrInvalidInput`, `ErrContractViolation`, `ErrorType` | `srderr/` | `content.Port` failures, every validation error (`Unwrap` to `ErrInvalidInput`), `Loader` |
| `telemetry.Telemetry`, no-ops, `StartTimer`, attribute constructors | `telemetry/timer.go`, `telemetry/` | `Loader` signals; same shape as `d20.NewResolver` (`WithTelemetry`, `WithClock`) |
| `ability.Ability`, `ability.All()`, `ability.Scores` | `ability/` | `Skill.Ability()`, class and monster ability data |
| `dice.Expression`, `dice.ParseExpression`, `dice.NewDie` | `dice/` | weapon and monster damage, spell-effect amounts, `HitPointDie.Faces()` conversion |
| `InvalidError` with a typed `Reason` | `ability/error.go`, `modifier/error.go`, `d20/error.go` | the shape every new `content` validation error copies |
| `telemetrytest` doubles | `telemetry/telemetrytest` | `Loader` telemetry tests |
| `dicetest.RunRollerContract` | `dice/dicetest/contract.go` | shape of the `contenttest` suites |
| `d20` layout, `signals.go`, `signals.md` rows | `d20/`, `doc/observability/signals.md` | template for `load_usecase.go` and the `srd5e.content.*` catalogue rows |
| `checkTopLevel`, `checkDecls`, `File.Imports`, `Package.Production`, `Policy` | `internal/archtest/{decls,imports,policy,check}.go` | the new scans are rows or extensions of these, not new walkers |

No kernel change is needed (checked against all seven kind specs). There is no
shared enum helper in the repo today; step 2 adds one (below).

## 3. Scope fence

**In scope:** the Phase 2 ("T") requirements of the specs above, per section 5;
the slice data; `NOTICE.md` coverage of `content/data/`.

**Out of scope** (each is tracked where it already lives):

- Bulk SRD data: 339 spells, the other species, backgrounds and feats, all
  classes and subclasses (`content-classes-data.md`, Phase 3), equipment rows,
  monsters, magic items; extraction tools under `tools/`, completeness and
  distribution pins, PDF verification (Phase 5).
- The toolbox catalogue kinds `poison`, `trap`, `contagion` and their ids
  (rows R3-*, "applies in phase 5"; schema FR-129).
- Any behaviour: Armor Class, spell effects as rules, monster combat, casting
  (`character` and `combat`).
- File or network sources (host adapters live in the host's repo).
- The release-0-1-0 content-loading example (`release-0-1-0.md` FR-020) and
  that spec's other items.

## 4. Open items, assumptions and spec defects

Nothing here is decided by this plan. **Decided**: a spec or the user already
settled it. **Assumption**: a sequencing choice the user can override.
**Spec defect**: fix in the spec in the same PR as the code that hits it.
**OPEN**: needs the user (or the spec owner) before the named step.

| # | Item | Class | Handling |
|---|---|---|---|
| P1 | Scope covers all "Types 2" specs, not bulk data or R3 kinds | decided (user) | section 3 |
| P2 | Build order puts `spell-effect-data` before `content-spells` (spells FR-122 imports `SpellEffectModel`) although the rank table says the reverse and the spells header omits it | assumption | spec defect, fix the header and rank note in the 5g PR; optional |
| P3 | `R2-monsters` and `R2-magic-items` say they land "with the first bulk data file", but their types and decoder tests need the root array registered now | assumption: register `monsters` and `magicItems` in Phase 2 | spec defect: amend the two row statuses in the schema revision table |
| P4 | Equipment FR-030, FR-039, FR-040, FR-042 need a "target has category X" reference constraint that schema FR-126 lacks | decided: schema section 5 says existence-only until a row adds `TargetHasEquipmentCategory` | step 5c checks existence only; those ids stay "not yet implemented" |
| P5 | Equipment FR-067 says references never resolve across documents; schema FR-035 and shared FR-054 say whole-set only | spec defect | rewrite in the 5c PR |
| P6 | A package-level `var` for tables, limits or the decoder's registration table breaks A4, and `init()` registration is an anti-pattern (architecture.md section 10) | decided: A4 wins | `const` blocks, funcs or `switch` only; `MonsterLimits`, magic-item fact tables and `decode.go`'s root-array table follow this; amend the spec wording in the same PR |
| P7 | Spells FR-120 names both a type and a function `ClassSpellList`; Go cannot export both | assumption: the function is `DeriveClassSpellList` | spec defect, rename in the 5g PR |
| P8 | `FactOverride[T]` (magic items FR-091) is an exported generic domain type; go.md section 6 requires a spec justification and nothing says several fact types need it | decided (user, 2026-10-18): drop the generic | the 5d PR replaces it with concrete per-fact override types and amends the magic-items spec |
| P9 | `R1-classes-spell-ids` (classes spec) vs `R1-classes-spell-sites` (schema); stale `ContentFound` / `MissingContent` names in classes and equipment | spec defect | fix only the occurrences a step trips over; optional otherwise |
| P10 | Overrides-file path conflict (monsters FR-060, FR-066, FR-069) and stale `data/srd_5_2_1/` paths (spells) | spec defect, bulk data only | not touched here |
| P11 | The 32-bit decoder run (schema FR-099) requires three edits: `ci.yml`, `AGENTS.md` "Commands" and `release-0-1-0.md`'s CI checklist | decided | step 4 |
| P12 | `//go:embed *.json` does not compile with no JSON file, and A12 allows exactly one non-test Go file in `content/data` | assumption: `bundled_source.go` (holding `BundledSource`, `BundledInvalidError`, the options and `Files()`) and the first data file land together; data lands `spells.json`, `class_wizard.json`, `species.json` so the merged set is never self-inconsistent | step 6 |
| P13 | **No spec limits document size or the number of problems.** A large hostile document yields one `Problem` per bad item (FR-012, FR-022), and `Loader` adds to `decode_errors` once per problem (FR-082): memory and CPU far above input size | decided (user, 2026-10-18): add caps | an exported `MaxDocumentBytes` constant (default proposed: 16 MiB) rejected as one `outOfRange` problem at path `""`, an exported `MaxProblems` cap (default proposed: 1000) with a final truncation problem, and `decode_errors` aggregated per kind before `Add`. The step 4 PR amends the schema spec; the two numeric defaults were not chosen by the user and are flagged for review |
| P14 | Input hygiene the specs leave open: duplicate keys inside one JSON object are last-wins (schema Q5, out of scope); `name` and `text` are kept "character for character" yet `encoding/json` turns invalid UTF-8 and lone surrogates into U+FFFD, and control or bidi characters pass; a path embeds attacker-chosen keys verbatim | decided (user, 2026-10-18): fix in the decoder | step 4 adds a token-level duplicate-key problem (`duplicate`), `invalidValue` for control and bidi characters and for invalid UTF-8 or lone surrogates in strings, and escaped, length-bounded paths in `Problem.Error()`. Each amends the schema spec in the step 4 PR; the exact character sets are chosen there and flagged for review |
| P15 | `JSONDecoder` lives in package `content`, not an adapter package, although architecture.md section 5 calls decoders adapters | decided: schema FR-078 and shared FR-031 settle it; the `encoding/json` confinement scan (step 4) is the guard | none |
| P16 | `ValidateSet` is a pure function outside `Loader.LoadSet` (schema FR-131), so cross-document `duplicate` and `unresolvedReference` problems reach no signal | decided: schema FR-131 | consumers must read the returned error; stated in step 4 and the catalogue, not mitigated with a signal |
| P17 | Roadmap Phase 3 says class data ships in Phase 3; the slice bundles the Wizard now | decided: class types ship in Phase 2 (shared FR-014), bulk class data in Phase 3 (thin-slice section 1) | none |

## 5. Delivery sequence

Eleven reviewable PRs, each green on its own. Order is forced by imports.

| Step | Ships | Why here |
|---|---|---|
| 0 | Guardrail groundwork (5.0) | the first production file in `content/` must already be inside the allow-list |
| 1 | Identity, problems, `Set`, `SourceCitation`, typed ids | every other type implements `Entry` or uses `Key` |
| 2 | Vocabulary enums, building blocks, `TextBlock` | kind shapes are built from these |
| 3 | `Source` port, `Catalogue`, `InMemorySource`, `contenttest` fakes and suites | needs only step 1; unblocks all adapters |
| 4 | `JSONDecoder` envelope, `ValidateSet`, `Loader`, signals, 32-bit CI | needs 1 to 3; kind rows plug into it; **needs P13 decided** |
| 5a | species, backgrounds, feats | independent of 5b to 5e |
| 5b | classes (Wizard-complete) | independent |
| 5c | equipment | independent |
| 5d | magic items | independent; **needs P8 decided** |
| 5e | monsters | independent |
| 5f | spell-effect-data (types and constructors only; no decoder key) | must precede 5g (P2) |
| 5g | spells, plus the `effects` decoder key | needs 5f |
| 6 | `data.BundledSource`, `BundledInvalidError`, `Files()`, slice data, `NOTICE.md` | needs 4, 5a, 5b, 5g |
| 7 | Whole-slice scenario, examples, README, CHANGELOG, roadmap | closes the phase |

Steps 5a to 5e touch the shared decoder registration (a func or `switch`, P6),
so merge them one at a time. The registration order must equal the schema
revision-table order (schema FR-016), not the order the PRs land in; step 4
adds a test that pins it, and each 5x PR extends it.

After step 4 the library is a coherent but empty decoder: every root array is
rejected as `unknownField` until its kind registers, which departs from the
schema table's "R1 baseline; landed". That is acceptable before v0.1.0 and each
5x PR flips its row status. Nothing is released before step 7.

### Process contract for every step

- Every new test cites its requirement (`Requirement:` and `Path:` comments;
  `Requirement: per case` on tables). `testing.md` section 5 is still "planned",
  so apply it to all new tests.
- Each PR carries a **requirement-to-test list**: every T id of the step maps to
  a test, and any id moved to D or left untested is named in the `Testing` entry.
- `just check` green. Each changed package at least 90 percent statement
  coverage, reported per package; no decoder row merges untested.
- Mutation testing with `gomu`, only inside Docker, `--cpus` and `--workers`
  capped at a quarter of the processors (`just mutate ./content`, pinned
  `gomu@v0.2.1` per `testing.md`); score at least 80 for domain files,
  survivors fixed unless provably equivalent. Where gomu cannot parse a file
  (generics), use the manual-mutation fallback of `testing.md` section 8 and
  list the mutations tried.
- `CHANGELOG.md` entry under `Unreleased`.
- The PR description carries `Testing` and `Observability` entries. The
  `Observability` entry answers **all seven questions** of `observability.md`
  section 2 for the change; a bare "none" or "silent by design" is rejected.
- A spec's `Status` becomes `partially implemented` in the PR that first ships
  code for it, with `Implementation status` and a `Not yet implemented:` line
  naming the deferred data and tooling ids. None becomes `implemented` in
  Phase 2.
- A step that changes a rule file refreshes its `RULE-TAG` (archtest A13 requires
  the new value to appear nowhere else in the repo, so never quote it in a plan
  or PR text) and updates the matching archtest table in the same PR.
- Fixtures are synthetic or the slice; no SRD text outside `testdata/srd/`
  (licensing FR-033); no consumer names; nothing from outside SRD 5.2.1.

### 5.0 Step 0: guardrail groundwork

- `internal/archtest/policy.go`: add `content/contenttest` (`KindSupport`; may
  import `content` and `srderr` only, which is all `FakeSource` and
  `FakeDecoder` need).
- `doc/rules/architecture.md` section 1: add the `content/contenttest` row.
  Refresh its `RULE-TAG`.
- `AGENTS.md` "Directory structure": add `content/`, `content/data/`,
  `content/contenttest/`. Leave the "Read the rules first" list untouched.
- `.gomuignore`: add `content/contenttest/` (test support is excluded).
- `content/doc.go`: package comment naming inputs, outputs and handoff types.

**Tests:** `internal/archtest` table test for the new row (violating overlay:
`contenttest` importing a context; clean overlay: importing `content`).
**Observability:** the seven questions, answered. What can go wrong: a new
package slips outside the allow-list (A1, A11) or imports a forbidden package.
Operator signal: `just arch` fails the build. Nothing needs counting or timing;
no existing signal is affected; no consumer monitoring applies; nothing is noisy.
**Rollback:** revert the PR (policy and docs only).
**Acceptance:** `just arch` passes with an empty `content/` plus `doc.go`.

### 5.1 Step 1: identity and envelope types

Schema FR-142 to FR-148 (`Kind`, `ID`, `ParseID`, `IDError`, `IDProblem`, `Key`,
`Entry`, `MissingContent`), FR-114 (typed ids; `MagicItemID` and `MonsterID`
included, the three toolbox ids not), FR-149 (`DurationUnit`), FR-014, FR-015,
FR-017, FR-097 (`Problem`, `ProblemKind`, `Problems`), FR-071, FR-072, FR-093
(`Set`, `NewSet`), FR-115, FR-138 (`SourceCitation`). `Kind` declares the eight
kinds up to `monster` now; `poison`, `trap`, `contagion` arrive with their rows
in Phase 5 and the pin test says so.

Files in `content/`: `kind.go`, `id.go`, `key.go`, `entry.go`,
`missing_content.go`, `duration_unit.go`, `problem.go`, `set.go`,
`source_citation.go`, `typed_ids.go`. Every error type copies the
`ability.InvalidError` shape (typed `Reason`, `Unwrap` to `ErrInvalidInput`, no
display-text fields; `errshape.go` enforces it).

**Tests (unit, plain values):** exact members and ordinals of `Kind`;
`IDProblem` order (empty, tooLong, invalidFormat) and the 64-character
boundary; typed-id table; `Set` order, copy, `Equal`, duplicate key;
`SourceCitation` boundaries (page 0, 1, 364, 365). `Fuzz*`: `FuzzParseID`.
Archtest: schema FR-148 import ban for the identity files, as a row in the
existing import checks.
**Observability:** silent. Failure modes (bad id, bad citation, duplicate key)
surface only as returned typed errors; `Loader` (step 4) later counts them as
`decode_errors` by `problem_kind`. Nothing to count at this layer; nothing noisy.
**Rollback:** revert the PR.
**Acceptance:** every listed id has a test; `just check` green.

### 5.2 Step 2: vocabulary and building blocks

Schema FR-105 to FR-113 (`Skill`, `DamageType`, `CreatureType`, `CreatureSize`,
`ConditionKind`, `Language`, `RechargeTrigger`, `WeaponProperty`,
`MasteryProperty`), FR-117 to FR-119 (`FeatCategory`, `Prerequisite`,
`ToolProficiency`), FR-116 (`NamedText`), FR-132 to FR-137, FR-139, FR-150 to
FR-152 (`TextBlock`). Wire spelling is the SRD spelling; every enum has
`String()` and `ParseX`; the zero value is invalid.

**Reuse and care:**

- One unexported helper (`enum.go`: a names table, an exact lookup and the typed
  parse error) serves every enum, instead of 17 hand-written loops like
  `ability.ParseAbility`.
- Wire parsing is **case-sensitive** and exact. Do not call
  `ability.ParseAbility` (it uses `EqualFold`); match against `ability.All()`
  and `Ability.String()`.
- `HitPointDie` stays a separate enum (schema FR-132); `Faces()` converts through
  `dice.NewDie`.

Files: `enum.go`, `skill.go`, `damage_type.go`, `creature_type.go`,
`creature_size.go`, `condition_kind.go`, `language.go`, `recharge_trigger.go`,
`weapon_property.go`, `feat_category.go`, `hit_point_die.go`,
`senses_and_light.go`, `alignment.go`, `named_text.go`, `prerequisite.go`,
`tool_proficiency.go`, `text_block.go`.

**Tests (unit):** one table per enum against the SRD member lists (counts 18,
13, 14, 6, 15, 19, 3, 9; order; case-sensitive rejection; padded spelling
rejected); constructor rows for each building block; `text_block_test.go`
(ragged table, blank item). **Fuzz:** `FuzzParseEnums`, one target that feeds
the same input to every `ParseX` (never panics; accepted input round-trips
through `String()`). Archtest: FR-128 and FR-130 single-declaration scan,
added as an extension of `decls.go`.
**Observability:** silent; enum parse failures are returned `ParseX` errors,
counted later by `Loader` as `invalid_value`. Nothing noisy.
**Rollback:** revert the PR.
**Acceptance:** exactly one declaration of each vocabulary type in the module.

### 5.3 Step 3: the `Source` port and its test support

Source-port FR-007 to FR-031, FR-033, FR-037 to FR-039, FR-041 to FR-044,
FR-046, FR-049 to FR-051, FR-057, FR-058: `Source`, `Port = "content"`,
`LookupResult` (`LookupFound`, `LookupNotFound`), `Query`, `Catalogue`,
`DuplicateKeyError`, `SourceError`, `InMemorySource`; in `content/contenttest`:
`FakeSource`, `RunSourceContract`, `RunReadOnlyContract`. Not here:
`data.BundledSource` (FR-059 to FR-064) and FR-047's `BundledSource` registration
(step 6).

Files: `source.go`, `source_error.go`, `lookup_result.go`, `query.go`,
`catalogue.go`, `in_memory_source.go`; `contenttest/fake_source.go`,
`contenttest/contract.go`, `contenttest/doc.go`.

**Tests:** unit on `Catalogue` with plain values (the one place duplicate, order
and lookup rules are tested; the ordering table in the spec); contract suites
against `InMemorySource`, `FakeSource` and the spec's test-only wrapping adapter
(FR-027), with self-tests using deliberately bad adapters (FR-042, FR-058);
concurrent calls under `-race` and a cancelled-context call on `InMemorySource`
(FR-024: the result is unchanged); a use-case-style `srderr.GuardValue` call
yields `*srderr.PortError{Port: "content"}` with `ErrorType` `content_error`;
the one scenario `scenarios/content_source_test.go` that FR-043 requires.
Architecture: port-name uniqueness (A8), closed-set docs (A7), FR-033 import
ban.
**Observability:** silent by spec (source-port section 7). Failure modes: backend
failure (`*SourceError`), absent key (`LookupNotFound`), duplicate keys
(`*DuplicateKeyError`). **Accepted gap:** no signal exists for any of them until
a Phase 3 or 4 use case calls `Source`; the advice to alert on `content_error`
and watch `ContentMissing` rates has nothing to act on in Phase 2.
**Rollback:** revert the PR.
**Acceptance:** `RunSourceContract` passes for every shipped `Source`.

### 5.4 Step 4: decoder, whole-set validation and `Loader`

Schema FR-001 to FR-031, FR-071 to FR-076, FR-093 to FR-098, FR-125 to FR-127,
FR-131, FR-140, FR-077 to FR-087, FR-100 to FR-102, FR-141 (`Decoder`,
`JSONDecoder`, `Loader`, `LoadSetResult`, `ValidateSet`, `FakeDecoder`,
`RunDecoderContract`) and FR-099 (32-bit CI). **P13 and P14 must be settled
first**; whatever the spec owner decides is built here.

Implementation requirements that follow from the specs and the review (none
adds a spec rule):

- Parse with `encoding/json` (`json.Decoder`, numbers as `json.Number`); no
  `math/big`. Decide integer wholeness on the decimal text (mantissa and
  exponent), not on a rounded `float64`, with a bounded exponent. Boundary rows:
  2^53 - 1, 2^53, 2^53 + 1, negatives, `-0`, `1e999999999`, `2.0`, `2.000...0`.
- For a bounded array, check the length first, emit one `outOfRange` and skip
  the children (as FR-013 does for wrong types). Use map-based sets for FR-026
  and FR-031 and an index map in `ValidateSet`: no quadratic scans.
- The root-array registration is a func or `switch` (P6) in revision-table
  order; a test pins that order.
- On a port failure spans and counters carry only the outcome and
  `srderr.ErrorType`; the recovered panic value stays out of telemetry
  (schema FR-085).
- `Decoder.Decode` takes no `ctx`, so cancellation is not honoured mid-decode;
  the `Decoder` doc comment says so.

Files: `decode.go`, `decode_value.go`, `validate_set.go`, `reference.go`,
`load_usecase.go`, `signals.go`; `contenttest/fake_decoder.go`,
`contenttest/decoder_contract.go`.
**Reuse:** `d20.NewResolver` for options; `srderr.GuardValue`; `telemetrytest`.

**Tests:**

- `decode_test.go`: table over every FR with problem path and kind (never
  message text), including the number boundary table above and 10000- and
  10001-deep nesting.
- `agreement_test.go` (external package `content_test`, unit level): every
  invalid value of the case tables goes to the decoder and to the matching
  `NewX`/`ParseX`; every valid value to both (schema FR-094).
- `validate_set_test.go`: whole-set rows including same key in two sets and a
  dangling reference.
- `load_usecase_test.go` and `telemetry_test.go`: spans, counters, failure
  recorded once, identical result with Noop and a recording bundle, a fixed
  clock via `WithClock`; an **already-cancelled `ctx`** (returns `ctx.Err()`,
  counts nothing, FR-086); a panicking `Decoder` and a `Decoder` returning a
  non-`Problem` error (`*srderr.PortError`, FR-086); signal names pinned.
- Concurrent `Decode` and `LoadSet` under `-race`; input not retained or
  mutated (FR-076).
- `RunDecoderContract` against `JSONDecoder` and `FakeDecoder`.
- **Fuzz:** `FuzzJSONDecoder` (no panic, deterministic, problems typed), with
  seeds covering exponent forms, a long fraction, a 10 MB flat array, a 100k
  element bounded array and each registered kind row as it lands.
- The 32-bit run `GOARCH=386 go test ./content/...` (no `-race`), plus the three
  edits of P11: a `ci.yml` job, the `AGENTS.md` "Commands" line and the
  `release-0-1-0.md` CI checklist entry (a spec scan fails if any is missing).
- Archtest: only `decode*.go` import `encoding/json`; no `os`, `net`, `io/fs` in
  `content`; only `load_usecase.go` refers to `PortError` and `Guard`. Each is a
  row or extension of the existing checks, with an `architecture.md` section 9
  update and a refreshed `RULE-TAG`.

**Observability:** adds six signals (`srd5e.content.load` and `.decode` spans,
`loads`, `load_duration`, `entries_loaded`, `decode_errors`) with their
attribute-key constants and closed value sets (`srd5e.content.problem_kind`,
`srd5e.content.kind`) in `signals.go`, in the `signals.md` catalogue (Go
constant, purpose, suggested monitoring) and in the `observability.md` section 5
attribute list (rule file: refresh its `RULE-TAG`). Alert on any `decode_errors`
and on `loads` with `srd5e.outcome=failure`.
**Known blind spots, accepted:** `ValidateSet` problems (P16), a cancelled `ctx`
(counted nowhere, by FR-086), duplicate keys inside one object (P14), and
oversized input if P13 is not adopted. Consumers must read the returned error;
the catalogue and `doc.go` say so.
**Rollback:** revert the PR; the 386 CI job reverts with it. No consumer exists
before v0.1.0.
**Acceptance:** a rejected document reports every problem in one pass in the
documented order; the 32-bit job passes.

### 5.5 Steps 5a to 5g: kind shapes

Each step adds: the entry types and constructors (immutable, validated, zero
value documented), the decoder row, the revision-table row status, reference-site
registration, and tests on **synthetic** entries unless the slice supplies them.
No bulk data. The T sets below come from a per-spec classification (types,
shapes, decoder rules, unit-testable tables) against the spec's own test plan;
**the classification is by judgement and unverified for 5c, 5d and 5e.** The
implementer re-checks every id against the spec's section 8 at the start of the
step and records moves between T and D in the PR.

| Step | Spec | T requirements (ids per spec) | Types (representative) | Files |
|---|---|---|---|---|
| 5a | content-species-backgrounds-feats | 005, 006, 008, 010, 012, 021 to 030, 032, 033, 035, 056 to 058, 060, 061, 064, 066, 067, 069, 071 to 073; rules of 013, 014, 016, 017, 039, 063 | `Species`, `SpeciesChoice`, `ChoiceOption` (`DamageTypeOption`, `LineageOption`, `TextOption`), `LevelBenefit`, `Background`, `Feat`, custom-background valuation helpers | `species.go`, `background.go`, `feat.go`, `custom_background.go`, `decode_species.go`, `decode_origins.go` |
| 5b | content-classes | 001, 002, 004 to 008, 010 to 015, 017 to 019, 021 to 026, 028, 030 to 032, 036, 037, 039, 040, 043 to 045, 047 to 053, 055, 056, 058, 059, 061 to 064, 066, 069 to 075, 099 (003, 009, 038, 102 as subset assertions) | `ClassDefinition`, `SubclassDefinition`, `ProgressionRow`, `FeatureEntry`, `FeatureFact`, `AbilityRequirement`, `ColumnValue`, `SpellcastingRow` | `class_*.go`, `decode_class.go` |
| 5c | content-equipment | 001, 002, 004, 007 to 014, 022, 024, 029 to 031, 037, 039, 040, 042, 057, 064, 067 to 069, 076 to 078, 080, 086 to 091, 093 to 095, 097, 098, 101 to 119 (P4, P5, P6) | `Weapon`, `Armor`, `Tool`, `AdventuringGear`, `Mount`, vehicles, services, `Cost`, `Coin`, `WeightSpec` | `equipment_*.go`, `decode_equipment.go` |
| 5d | content-magic-items | 001 to 005, 007 to 009, 011 to 019, 022, 037 to 044, 056, 078 to 097 (P3, P6, P8) | `MagicItemEntry`, `MagicItemCategory`, `RaritySpec`, `Attunement`, section 4.8 fact types | `magic_item*.go`, `decode_magic_item.go` |
| 5e | content-monsters | 001 to 006, 008, 009, 011, 013 to 021, 024 to 054, 058, 072 to 081, 089, 090, 093, 095 to 101 (P3, P6) | `Monster`, `ChallengeRating`, `StatBlockPart`, `MultiattackPlan`, `MonsterSpellcasting` | `monster*.go`, `decode_monster.go` |
| 5f | spell-effect-data | 001 to 029, 039 to 049 (types and constructors; FR-030 to FR-032 are the decoder key and move to 5g) | `SpellEffectModel`, `SpellTargeting`, `SpellEffectPart`, `SpellOutcome`, `AmountScaling`, `ZoneFacts` | `spell_effects*.go` |
| 5g | content-spells | 007, 009 to 012, 017, 026 to 030, 036, 037, 041, 049 to 052, 057 to 059, 086 to 089, 092, 098, 101, 102, 107 to 111, 113 to 115, 119 to 122, plus spell-effect-data FR-030 to FR-032 (P2, P7) | `Spell`, `SpellLevel`, `CastingTime`, `SpellRange`, `SpellComponents`, `SpellDuration`, `DeriveClassSpellList` | `spell*.go`, `decode_spell.go` |

Per-step specifics:

- **5a:** the baseline site `(background, feat/featId, feat,
  TargetHasFeatCategory(Origin))` and the species spell-id sites are registered
  and exercised with synthetic sets; `Feat` implements `HasFeatCategory`.
- **5b:** the Wizard shape is fully covered because the slice ships it (20 rows,
  ten feature entries, `GrantsSubclass`, empty `subclasses`), including the
  progression boundaries (level 0, 1, 20, 21) and `AllOf`/`AnyOf` boundaries.
  Pact rows, options and tables use synthetic classes.
- **5c:** reference sites check existence only (P4). Weapon `damage` is parsed
  with `dice.ParseExpression`, rejecting when `Modifier() != 0` or the text
  differs from `Notation()`: no new scanner.
- **5d:** amounts are Copper Pieces `int`s; no import of equipment.
- **5f:** must not import `Spell` (FR-009, FR-039).
- **5g:** adds the `effects` key, `FuzzDecodeSpellEffects` (spell-effect-data
  spec) and the P7 rename; class lists derive from `Spell.classes`.

**Tests per step:** unit tables for every constructor and decoder rule (happy,
every unhappy path, boundaries, empty and nil, immutability, copy on read); the
step's rows added to `agreement_test.go` and to the `FuzzJSONDecoder` seeds; a
named fuzz target for every new parser (`FuzzParseChallengeRating`, the equipment
cost and spell form parsers); architecture scans named in each spec, as rows or
extensions of `decls.go`, `imports.go` and `errshape.go`.
**Observability:** each step answers the seven questions for its decoder row. No
new signal. What can go wrong: new problem paths and reference sites. Operator
signal: `decode_errors{problem_kind}` and `entries_loaded{kind}` already cover
them. The `srd5e.content.kind` allowed values gain one entry per kind
(`species`, `background`, `feat`, `character_class`, `equipment`, `spell`,
`monster`, `magic_item`), updated in `signals.md` and the `observability.md`
section 5 list in the same PR (new rule tag).
**Rollback:** revert the PR.
**Acceptance:** a document using the step's array decodes; a bad one reports
path and kind for every problem; the spec status is `partially implemented` with
its deferred ids named.

### 5.6 Step 6: `BundledSource`, the slice and the notice

Source-port FR-047, FR-059 to FR-064; schema FR-089, FR-090, FR-104, FR-120,
FR-122; the slice data of content-thin-slice FR-001 to FR-032, FR-048, FR-050 to
FR-067, FR-078, FR-083, FR-084; content-licensing FR-001 to FR-013, FR-027 to
FR-029, FR-033, FR-035, FR-037 (`NOTICE.md` status, scope, changes, attribution
copied from the SRD's "Legal Information" page, p. 1, not from memory).

`content/data/bundled_source.go` is the one non-test Go file in the package (A12)
and holds `BundledSource`, `BundledInvalidError`, the options and `Files()`. The
document index comes from the embed variable (`fs.ReadDir`); no second
package-level index (A4). `NewBundledSource` decodes and validates on every call
and caches nothing; the doc comment says to build it once per process.
`MustBundledSource` panics on a bad bundle, intentionally, for build-time-fixed
embedded data.

Sequence, each commit keeping the merged set valid:

1. `bundled_source.go`, `spells.json` (12 spells; nothing in a spell document
   references another entry, and `Spell.classes` is not a reference site), the
   `NOTICE.md` status change and the FR-029 file header.
2. `class_wizard.json` (Wizard; its feature entries reference only each other).
3. `species.json` (4 species; the nine lineage `spellId`s resolve against 1).

**Tests:** `bundled_source_test.go` (injected `fakeDecoder`: one decode call per
text, empty index, determinism, copy semantics, no background goroutine; real
decoder: a synthetic corrupted text, a dangling `spellId` and a duplicate key
each yield `*BundledInvalidError`, with `DecodeProblems` and `ReferenceProblems`
as specified); a fixture-FS test for `Files()` on an empty directory (the real
embed can never be empty); `RunSourceContract` and `RunReadOnlyContract`
registrations; `data_pin_test.go` for the slice (tables T1 to T7 from
`testdata/srd/thin_slice/`) which also **pins the expected file list**
(`spells.json`, `class_wizard.json`, `species.json`) and 17 entries, so a
well-formed extra file in `content/data/` fails; `LoadSet` three times over the
slice with `telemetrytest` and a fixed clock for the FR-078 counts (4, 1, 12; no
`decode_errors`). Archtest A12 and the `NOTICE.md` covered-path check become
non-vacuous.
**Process (not CI):** fidelity of every long text against the PDF is reviewed by a
person and recorded in the PR checklist (thin-slice FR-079); the denylists catch
names, not unsourced content.
**Observability:** answers the seven questions. Failure modes: transcription
error, bad bundle, dangling reference. Signals fire only if a consumer runs
`LoadSet` over `Files()` (thin-slice section 7); `NewBundledSource` and
`MustBundledSource` emit nothing. A bad bundle is caught by the CI pin tests and
the returned `*BundledInvalidError`. A transcription error has no runtime signal;
the correction path is a data patch release with a `CHANGELOG.md` entry.
**Rollback:** revert the PR, which also restores `NOTICE.md` to "no game content
is bundled".
**Acceptance:** `data.NewBundledSource()` builds with no problems, holds exactly
17 entries, and `ValidateSet` passes over the merged set.

### 5.7 Step 7: close the phase

- `scenarios/content_test.go`: the one whole-slice scenario (decode a document
  with problems, load the slice through `BundledSource`, look up present and
  absent keys); step 3's scenario covers the port alone.
- `Example*` functions for `JSONDecoder`, `Loader`, `InMemorySource` and
  `BundledSource` (`testing.md` section 10: public types SHOULD have one), with
  deterministic `// Output:` and no-op telemetry.
- README: the adapter-author section naming `contenttest.RunSourceContract`
  (source-port FR-050) and the attribution statement (licensing FR-028).
- `CHANGELOG.md`: Added entries for every package; no **BREAKING**.
- Roadmap: update the Phase 2 status line and the status table counts (specs
  were flipped in their own PRs).

## 6. Verification commands

```sh
just check
GOARCH=386 go test ./content/...
go test -race -shuffle=on -coverprofile=coverage.txt ./content/... && go tool cover -func=coverage.txt | tail -n 1
just mutate ./content          # Docker only, capped at a quarter of the processors
```

## 7. Security review

The new trust boundary is **untrusted JSON from a host**, through
`JSONDecoder.Decode` and `Loader.LoadSet`.

| Concern | Worst case | Mitigation |
|---|---|---|
| Resource exhaustion by size | a hostile document yields a `Problem` per bad item, an `errors.Join` chain and one counter `Add` per problem; memory far above input size, an OOM that `GuardValue` cannot recover | **P13 (OPEN)**: size cap, problem cap, per-kind aggregation. Until decided the limit is the caller's and the doc comments say so |
| Resource exhaustion by shape | huge arrays or many keys cause quadratic work | step 4: length checked before children, map-based sets, a 100k-element fuzz seed |
| Number handling | `1e999999999` or a megabyte of digits as a DoS vector; platform-dependent results | step 4: no `math/big`; wholeness from decimal text with a bounded exponent; the 32-bit CI run |
| Nesting depth | stack exhaustion | `encoding/json` limits depth (10000); tests at 10000 and 10001 |
| Panic on hostile bytes | consumer crash | schema FR-018: `JSONDecoder` never panics; `Loader` converts a third-party `Decoder` panic to `*srderr.PortError` (FR-086). Direct `Decode` callers do not get that recovery, and stack overflow or OOM is not recoverable; the fuzz corpus is the control |
| Unknown or legacy keys ignored | other-edition content slips in | schema FR-022, FR-074: `unknownField` at any depth |
| Duplicate keys in one object | last-wins can hide an unknown key; a host validating with first-wins sees different content | **P14 (OPEN)**; accepted risk, documented |
| Control and bidi characters, invalid UTF-8, lone surrogates in `name` and `text` | spoofed display text; FR-029's "character for character" claim is false for U+FFFD cases | **P14 (OPEN)**; hosts must sanitise before display |
| Log injection through paths | an attacker-chosen key with newlines or escapes reaches a consumer's logs via `Problem.Error()` | **P14 (OPEN)**; document "do not log `Error()` unescaped"; fix needs a spec change |
| Telemetry disclosure | document bytes in a recovered panic value or error string reach spans | step 4: only outcome and `ErrorType` are recorded (FR-085) |
| Embedded data trusted more than host data | a bad bundle reaches consumers | schema FR-104: same decoder; a bad bundle returns `*BundledInvalidError`, never a partial source; `MustBundledSource` panics at init, intentionally; the pin test fixes the file list |
| Licensing and provenance | non-SRD or trademarked content ships; `testdata/srd/` ships in the module zip but is not a `NOTICE.md` covered path | AGENTS.md rule, licensing FR-023 and FR-031 denylists, A10; fixtures kept minimal; the manual transcription review is the only control for unsourced content |
| Supply chain of tooling | `gomu` unverified binary | `testing.md` pins `gomu@v0.2.1`; the runs stay in capped Docker |
| Authn, authz, secrets, cross-tenant data | none | N/A: no network, storage, identity or secrets; `os` and `net` are denied in `content` (A3) |

## 8. Compatibility and rollout

Before v0.1.0 nothing is published, so every change is an addition and every
step is an independent revert. After the first release these become
**BREAKING**: raising `schemaVersion`, adding a `Kind`, `ProblemKind` or
vocabulary member, adding a variant to a closed set (`LookupResult`, `Query`,
`TextBlock`, `ChoiceOption`, and so on).

Rule-file and policy changes (`architecture.md` rows and section 9,
`observability.md` section 5) land with the code that needs them and refresh the
rule tag. A new archtest check that fails an existing file is fixed in the same
PR, never skipped.

**Post-release correction path** (not needed before v0.1.0): a defect in bundled
data is fixed by a patch release with a `CHANGELOG.md` entry and, if needed, a
retraction; a consumer can fall back to a host-supplied `InMemorySource`.

## 9. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Size: the largest phase (nine type specs, about 650 requirement ids) | one PR per step; 5a to 5e independent; T/D lists re-checked per step; stopping after step 4 leaves a coherent, empty decoder that is not worth releasing |
| Spec defects (section 4) found mid-step | fix the spec in the same PR; contradictions needing a decision stop the step and go to the user |
| T/D classification is by judgement for equipment, magic items and monsters | implementer re-checks every id; moves recorded in the PR requirement-to-test list |
| Transcription error in the slice | pin tests plus the manual PDF review (step 6) |
| A4 conflicts with table-like facts and the decoder registration | constants, funcs and `switch` only (P6) |
| `tools/` is a nested module that `go test ./...` skips | out of scope (all tools work is bulk data); revisit in Phase 5 |
| A hostile document amplifies into memory and counter calls | P13 decision before step 4 |

## 10. Out-of-scope observations (recorded, not acted on)

- Spec headers and ranks disagree for spells and spell-effect-data (P2), and
  several specs still say `ContentFound` or `MissingContent` (P9).
- `content-classes-data.md`'s phase label (3) and shared FR-014's bulk-data
  phase (5) disagree; the roadmap says 3 and the slice needs neither.
- Persistent-spell-zones fact table Z (a `combat` spec) is cited by
  spell-effect-data FR-050; the zone types ship here, the pilot table is a
  fixture only.
