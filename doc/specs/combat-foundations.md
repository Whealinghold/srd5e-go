# Combat foundations

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `observability-ports.md`, `content-schema-and-decoder.md`, `character-aggregate.md` |
| Deviations from the decisions | None. |

> Go notation used below. A name in lowerCamelCase (a field, command argument or enum member) stands for the exported PascalCase Go identifier (`attackerID` is the field `AttackerID`). A trailing `?` marks an optional value (the zero value or a `(T, bool)` pair). `[]T` is a slice; `set of T` is a collection without duplicates. A `(T, error)` pair replaces an accepted/rejected result pair: the nil-error case is the accepted case and the error is the typed rejection, satisfying `errors.Is(err, srderr.ErrInvalidInput)`; a parser reports every problem at once through `errors.Join`. Rule outcomes (refusals, not-possible stops) stay result values. A use case is a struct with `Run(ctx, cmd)`; its `error` is non-nil only for a port failure.

## 1. Purpose and scope

Rank 20, the leftmost spec of the `combat` order (`shared-types-and-ownership.md` FR-015); it depends on no other `combat` spec. Prose dependencies (the header lists only lower-rank specs, FR-056): `CharacterID` from the `character` package (`character-aggregate.md`); `MonsterID` from the `content` package (`content-schema-and-decoder.md`, FR-114); `error.type` and the scalar `NewX` / `ParseX` constructors of `errors-and-results.md` (FR-043, OQ-11); `telemetry.Telemetry`, `telemetry.Noop`, `telemetry.AttrOutcome` and `telemetry.AttrErrorType` of `observability-ports.md`. `shared-types-and-ownership.md` is called "decisions" below. It applies decisions FR-012, FR-015, FR-017, FR-019, FR-021, FR-041, FR-044, FR-046, FR-051.

The earlier `combat` specs each assumed something different about who a combatant is, what an event is and how signals are named. This spec is the one owner of:

1. `CombatantID`, the typed id of a creature in a fight.
2. `CombatantKind` (which character or monster) and `CombatantRef` (id plus kind).
3. `Event`, the base of the combat event family.
4. Naming conventions for commands, results, rejections and events in the `combat` package.
5. The `srd5e.combat.*` observability conventions: shared signals, shared attributes, result-to-outcome mapping.
6. `CombatRuleOptions` and `PlayingOnAGrid` (section 4.6; decisions FR-057; pattern in `rule-options.md`).

**Out of scope:**

- Any rule. Hit points, conditions, turn state and side belong to specs to the right; `Side`, `Combatant` and `EncounterID` stay in `encounter-aggregate.md`.
- Assigning ids: the host supplies every `CombatantID`; the package never generates one (no randomness or clock in the domain).
- Anything about a creature that is not an SRD 5.2.1 fact. The SRD names participants ("each participant in a battle", "The Order of Combat", p. 13) but defines no combatant record; these types are package design. Whether a combatant is a Player Character or a Monster is identity only; no rule reads it.
- Movement costs on a grid and range counting (p. 13): `initiative-and-actions.md` keeps feet; `grid-positions-and-movement.md` FR-001 reads `CombatRuleOptions.Grid` and owns grid rules; FR-042 to FR-045 hold only the grid choice and the feet-to-squares conversion.
- Tracking an object or location as a creature (`attacks.md` defines no `AttackTarget`).

## 2. Source

- SRD 5.2.1, "The Order of Combat" and "Combat Step by Step", p. 13.
- SRD 5.2.1, "Playing on a Grid" sidebar ("If you play using a square grid"; "Squares. Each square represents 5 feet"; "Speed ... dividing it by 5"), p. 13; "Round Down", p. 5.
- SRD 5.2.1, Rules Glossary, "Player Character", p. 186, and "Monster", p. 185.
- Package design: `doc/rules/architecture.md` §2, §3, §4, §6; `doc/rules/observability.md` §2, §5-§7; `doc/rules/testing.md` §4-§5; `shared-types-and-ownership.md`.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| (combatant identity) | `CombatantID` (named type over `string`) | Not an SRD term ("combatant", "participant", p. 13). No `CreatureID`, `ParticipantID` or `TargetID` exists. |
| Player Character / Monster | `CombatantKind` (closed): `PlayerCharacterCombatant{ID character.CharacterID}`, `MonsterCombatant{ID content.MonsterID}` | "Player Character", p. 186; "Monster", p. 185. Identity only. Variant names avoid `Monster` and `PlayerCharacter`, which `content` and `character` own. |
| (combatant reference) | `CombatantRef` (`ID`, `Kind`) | Not an SRD term. What a host gives the engine to introduce a combatant. |
| (combat event) | `Event` (`interface`, unexported marker) | Base of every combat domain event (decisions FR-019, D-1). |
| (event kind) | `Event.Kind()` (`string`) | Constant `snake_case` name per event type (an envelope reports the wrapped event's, FR-038); the value of `srd5e.combat.event`. Not display text. |
| (event envelope) | an `...Occurred` event implementing `transportEnvelope` | Past-tense event wrapping another spec's event for transport (FR-038). |
| (envelope marker) | `transportEnvelope` (interface with an unexported marker method, not exported) | Declared once here; only envelopes implement it. |
| (use case name) | `CombatOperation` | The `srd5e.combat.operation` value and span-name suffix. |
| (result class) | `CombatResultClass` (`ResultApplied`, `ResultRefused`, `ResultInvalid`; telemetry values `applied`, `refused`, `invalid`) | Classifies every result variant for telemetry (FR-031). |
| Playing on a Grid | closed `PlayingOnAGrid`: `NoGrid`, `SquareGrid` | p. 13. Default `NoGrid` (the SRD baseline; a package term). `SquareGrid` is named from the SRD's "Squares" entry. |
| Speed | `int` feet | Converted to squares by `SquareGrid` (p. 13). |

Terms use SRD 5.2.1 wording (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber. One behaviour each.

### 4.1 `CombatantID`

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `CombatantID` MUST be a named string type (`type CombatantID string`), defined once in `combat` and exported from the `combat` package. The zero value (the empty string) is an invalid id and `IsZero` reports it. No other `combat` spec defines an id for a creature in a fight. | `shared-types-and-ownership.md` FR-012; architecture §3 | happy |
| FR-002 | `NewCombatantID(value string) (CombatantID, error)` MUST accept any string for which `strings.TrimSpace(value) == ""` is false (`TrimSpace` removes every `unicode.IsSpace` rune: `\r`, no-break space and the Unicode space separators as well as space, tab, newline) and MUST return it unchanged, with a nil error. `CharacterID` and `EncounterID` are to use the same predicate so the three id types agree on blank (D-7); `character-aggregate.md` FR-002 and `encounter-aggregate.md` FR-001 say only "whitespace", and their owners are asked to pin it. | architecture §3 | happy |
| FR-003 | `NewCombatantID` MUST return the zero id and an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (and `errors.As` to `*CombatantIDError` with reason `CombatantIDEmpty`) for the empty string. | architecture §6 | unhappy (invalid input) |
| FR-004 | `NewCombatantID` MUST return the same error for a blank string (`strings.TrimSpace(value) == ""`): only space, tab, newline, `'\r'` or `'\u00A0'` (no-break space), in any mix. | architecture §6 | unhappy (boundary) |
| FR-005 | `MustCombatantID(value string) CombatantID` MUST return the same id as `NewCombatantID` for a string it accepts and MUST panic with the error `NewCombatantID` returns for one it rejects (the panic value is that error, so `errors.As` works on the recovered value). `NewCombatantID` MUST NOT panic for any string. | `errors-and-results.md` FR-025, FR-035; architecture §6 | unhappy (invalid input) |
| FR-006 | A `CombatantID` MUST NOT normalise its value: no trimming, no case folding. `MustCombatantID("Goblin 1")`, `MustCombatantID("goblin 1")` and `MustCombatantID(" Goblin 1")` MUST be three different ids. | architecture §3 | unhappy (boundary) |
| FR-007 | Two `CombatantID`s built from equal strings MUST be equal (`==`) and usable as the same map key; different strings MUST differ. | architecture §3 | happy |
| FR-008 | A `CombatantID` MUST NOT be assignable where an `EncounterID`, `CharacterID` or `MonsterID` is expected, or the reverse. Static property of the Go type system, verified by a `go/types` test that type-checks a small source snippet and expects an assignment error (section 8), not a runtime test; runtime equality across id types is unspecified and no code may rely on it. | architecture §3, §10 | unhappy (invalid input) |
| FR-009 | The package MUST NOT generate `CombatantID`s. No file in package `combat` may import `math/rand`, `math/rand/v2`, `crypto/rand` or a uuid package, and no function may return a `CombatantID` other than `NewCombatantID` and `MustCombatantID`. Uniqueness within one fight is the host's responsibility, checked by the encounter (`DuplicateCombatantID`, `encounter-aggregate.md`). | architecture §3, §9 | unhappy (determinism) |

### 4.2 Combatant reference types

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `CombatantKind` MUST be a closed interface with exactly two variants, `PlayerCharacterCombatant` (carrying a `character.CharacterID`) and `MonsterCombatant` (carrying a `content.MonsterID`), both value structs implementing the interface through an unexported marker method, so a type `switch` is exhaustive. | SRD 5.2.1, "Player Character", p. 186; "Monster", p. 185 | happy |
| FR-011 | Each `CombatantKind` variant MUST be immutable and equal by value: the same variant with an equal reference id is `==`, so the values are comparable and usable as map keys; a different variant or reference id MUST NOT be equal, even when the underlying strings match. | architecture §3 | unhappy (boundary) |
| FR-012 | A `CombatantKind` MUST carry identity only. Its variants' only exported members MUST be the reference id field; no rule, predicate or behaviour is attached. | `encounter-aggregate.md`; architecture §3 | happy |
| FR-013 | `CombatantRef{ID CombatantID; Kind CombatantKind}` (constructor `NewCombatantRef(id, kind)`, which rejects a zero id or nil kind with an `srderr.ErrInvalidInput` error) MUST be a comparable value equal over both fields. Two refs with the same `CombatantID` and different kinds MUST NOT be equal. | architecture §3 | unhappy (rule interaction) |
| FR-014 | `CombatantKind` MUST NOT reference `Side`, hit points, conditions or any other `combat` type. The reference types MAY import the standard library, the `character` and `content` packages' published API and the kernel packages. They MUST NOT import another `combat` spec's files, `d20`, `toolbox` (a downstream leaf: `architecture.md` §2, `architecture-guardrails.md` FR-098), or an `internal/` package of another context. | `shared-types-and-ownership.md` FR-013, FR-015 | unhappy (forbidden edge) |

### 4.3 The `Event` family

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-015 | `Event` MUST be an interface with an unexported marker method, declaring `Kind() string` and `Combatants() []CombatantID`, defined once here and exported from the `combat` package; there is no kernel base type. Each sub-family closes itself with its own marker (D-1). | `shared-types-and-ownership.md` FR-019 | happy |
| FR-016 | Every domain event exported by the `combat` package MUST implement `Event` through exactly one closed sub-family owned by one spec (`HitPointEvent`, the condition events, `EncounterEvent`, ...), so a consumer's `switch` over a sub-family is exhaustive. An architecture test lists the implementers of `Event` and fails on one that implements it directly or is not in a closed sub-family. An envelope (FR-038) is a member of its owning spec's closed family (for example `EncounterEvent`, `encounter-aggregate.md` FR-050), never a direct implementer. | `shared-types-and-ownership.md` FR-019 | happy |
| FR-017 | For every non-envelope `Event` implementer, `Kind()` MUST return a constant string per type, match `[a-z][a-z0-9_]*`, and be unique across the package's `Event` implementers (`character.Event` and class events have no `kind` and are out of this rule). An envelope reports the wrapped event's dynamic `kind`, so it is exempt from the constant rule and uniqueness (FR-038). Equal events MUST report equal kinds. `Kind()` MUST NOT be derived from reflection (`reflect`, `%T`) or from `String()`. | observability §5; `errors-and-results.md` FR-034 | unhappy (name collision) |
| FR-018 | `Event.Combatants()` MUST return a fresh copy of the list of the creatures the event is about, in a deterministic order its defining spec documents, empty only for an event about no creature (for example a round starting), with no duplicates. | architecture §4 | unhappy (boundary) |
| FR-019 | Every `Event` implementer MUST be an immutable value (unexported fields, value receivers, no setters) and be equal by value including `Kind()` and `Combatants()`. Events carry no timestamp and no display text. | architecture §3, §6 | unhappy (immutability) |
| FR-020 | An event MUST be named in the past tense and state a fact that happened. It MUST NOT end in `Required` or `Needed`, and MUST NOT end in `Due` unless the word before `Due` is `Became` (the pattern `(?<!Became)Due$\|Required$\|Needed$` fails). A spec that must tell the host something is due names the happening: `...BecameDue` (`MovementSpeedsReviewBecameDue`, `GrappleCheckBecameDue`, both in `combat-action-resolution.md`) or the causing fact (`DyingCombatantTurnStarted`, `encounter-aggregate.md` FR-018, which replaced `DeathSavingThrowDue`). A transport wrapper MAY end in `Occurred` and follows FR-038. Each named event has exactly one defining spec. | architecture §3; `shared-types-and-ownership.md` FR-019 | happy |
| FR-021 | A transition result MUST hold its events as a `[]Event` (or the owning sub-family type), in the order they happened, and its accessor MUST return a copy: appending to or modifying the returned slice MUST NOT change the stored list. | architecture §3, §4; `encounter-aggregate.md` FR-050 | unhappy (immutability) |

### 4.4 Naming conventions in the `combat` package

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | Every `combat` use case MUST take a command named `<UseCase>Command` and return a closed `<UseCase>Result` (an interface with an unexported marker method; the variant list is in its doc comment) and an `error` that is non-nil only for a port failure or invalid command construction. Variants MUST be prefixed with the operation noun (`DamageApplied`, `DamageRejected`, `StartEncounterInvalid`). A generic name (`Applied`, `Rejected`, `Succeeded`, `Invalid`, `ContentMissing`, `Found`, `NotFound`) MUST NOT be exported from the `combat` package. | `shared-types-and-ownership.md` FR-017; architecture §4, §9 | unhappy (name collision) |
| FR-023 | A reason a use case refuses or rejects MUST be a value object or enum named `<Subject>Rejection` (or `<Subject>Problem` for a list), held as a field of the `<Operation>Rejected` or `<Operation>Invalid` variant, not a variant of a result shared across use cases. Exception (reason is a domain refusal, class `refused`, not invalid input): `AttackNotPossible` and `UnarmedStrikeNotPossible` (`attacks.md` FR-120) keep the `...NotPossible` suffix and hold an `AttackNotPossibleReason`; `combat-action-resolution.md` consumes the names. | `shared-types-and-ownership.md` FR-018; architecture §4 | happy |
| FR-024 | Every name exported by the `combat` package MUST be unique within package `combat`. The in-context order is the rank table `shared-types-and-ownership.md` 4.11 (ranks 20 to 29); a `combat` spec MAY depend only on strictly lower ranks (FR-056), so equal-rank specs never depend on each other. Where two `combat` specs named the same thing, the lower-rank spec keeps the plain name and the higher uses an `...ToCombatant`, `...OnSet` or similar qualified form; at equal rank the spec file name that sorts first alphabetically keeps it. A new `combat` spec takes its rank from the table (changing the table changes the decisions). `architecture-guardrails.md` FR-091 is kept in sync. | `shared-types-and-ownership.md` FR-015, FR-021, 4.11 | unhappy (name collision) |
| FR-025 | Names this spec reserves MUST NOT be defined by any other spec: `CombatantID`, `CombatantIDError`, `CombatantIDRejection`, `CombatantKind`, `PlayerCharacterCombatant`, `MonsterCombatant`, `CombatantRef`, `Event`, `CombatOperation`, `CombatResultClass`, `Instrumenter`, the signal constants of FR-039 (`CombatSignals` below), `transportEnvelope`. | `shared-types-and-ownership.md` FR-012 | unhappy (duplicate) |
| FR-026 | A `combat` spec MUST NOT import, list or assume a type from a `combat` spec of equal or higher rank (FR-024), or from `toolbox`. It MAY use only the package APIs of `character`, `d20` and `content`: their published language (value objects, ids, events) in domain files, and that plus the domain services their packages export (for example `d20.Resolver`) in use-case files (`architecture.md` §1, `shared-types-and-ownership.md` FR-044). It MUST NOT import another context's use cases or `internal/` packages. | `shared-types-and-ownership.md` FR-013, FR-015, FR-044; architecture §1, §2 | unhappy (cycle) |

### 4.5 Observability conventions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | Every combat-specific signal name and attribute key MUST begin `srd5e.combat.`, be lowercase, dot-separated, and be a constant in one file, `combat/signals.go` (called `CombatSignals` in this spec, FR-039). Exempt: the shared keys `telemetry.AttrOutcome` and `telemetry.AttrErrorType` (`observability-ports.md` FR-052), which `CombatSignals` MUST NOT re-declare or contain as a literal; and keys owned by another context's signals (`srd5e.d20_test.*`, `srd5e.ability`), which `combat` code reads from the constants that context's package exports (for `d20`, the exported `Span...`, `Metric...` and `Attr...` constants of `d20/signals.go`: `resolve-d20-test-use-case.md` FR-040, FR-042), never as a literal and never re-declared in `CombatSignals`. Later `combat` specs add their constants to `CombatSignals` and their rows to the catalogue; no call site uses a string literal. | observability §5, §10 | unhappy (invalid input) |
| FR-028 | The operation attribute MUST be `srd5e.combat.operation`, whose value is the `snake_case` use-case name (`apply_damage`, `advance_tracker_turn`). `NewCombatOperation` MUST return an `srderr.ErrInvalidInput` error for a value not matching `[a-z][a-z0-9_]*`, and for a span name not ending in `.` followed by the value (FR-040). This key replaces `srd5e.combat.use_case`, `srd5e.operation` and `srd5e.combat.condition.operation`. | observability §5 | unhappy (invalid input) |
| FR-029 | Every `combat` use case MUST run inside one span named `srd5e.combat.<operation>`. When the work returns a result it MUST record exactly once the counter `srd5e.combat.operations` (`{operation}`) and, per event produced, the counter `srd5e.combat.events` (`{event}`, attribute `srd5e.combat.event` = the event's `kind`; FR-038 for envelopes). It MUST record the histogram `srd5e.combat.operation_duration` (`s`, through `telemetry.StartTimer` and `Timer.Stop`, deferred) exactly once per call, whether the work returns a result or an error. **Error path:** `operations` is **not** incremented and no `events` are counted (no result exists to classify); the call is counted by `srd5e.combat.errors` (FR-032) for an `*srderr.PortError` and by the span only for any other error. Extras from `Description.Extras` (FR-041) go on the span and the `operations` increment only. Use-case-specific signals are added by their owning spec and MUST NOT repeat these. | observability §4-§6 | happy |
| FR-030 | The shared attributes (a closed list; more only through FR-041) are: `srd5e.combat.operation` (FR-028); `srd5e.outcome` (`success` or `failure` only, `shared-types-and-ownership.md` FR-041); `srd5e.combat.result` (a closed `snake_case` set per use case, owned by that spec; each value names one reason or outcome and maps to exactly one `CombatResultClass`, for example `damaged`, `killed`, `rejected_dead`, `not_dying`; not derived from the variant type name, finer than `result_class`; `Run` obtains it from the caller's `Description.Name` and `Description.Class`); `srd5e.combat.result_class` (`applied`, `refused`, `invalid`); `srd5e.combat.event` (an event `kind`); `error.type` (only on `srd5e.combat.errors`). A spec MUST NOT add a value to `srd5e.outcome` or `srd5e.combat.result_class`. Each `srd5e.combat.result` value is a `...Value` constant added to `CombatSignals` by its owning spec (FR-039); the `srd5e.outcome` and `result_class` values come from `CombatSignals` `...Value` constants (`CombatResultClass.Outcome` returns them). | `errors-and-results.md` FR-043; observability §5 | unhappy (boundary) |
| FR-031 | Every result value of every `combat` use case MUST be classified as one `CombatResultClass`. `applied` records `srd5e.outcome` = `success` and no error status. `refused` (legal input the rules do not allow in the current state: a dead creature making a death saving throw, the wrong combatant's turn, nothing to stabilise) records `failure` and **no** error status, being a normal rule outcome. `invalid` (host input the engine cannot use: an unknown combatant, a malformed start, a rejected tie-break) records `failure` and sets the span error status. `CombatResultClass` MUST expose `Outcome() string` and `IsSpanError() bool` so no use case re-derives them. | observability §6, §7; `errors-and-results.md` FR-043 | unhappy (rule interaction) |
| FR-032 | `srd5e.combat.errors` (`{error}`) MUST count only a returned `*srderr.PortError` from a port, once, with `srd5e.combat.operation` and `telemetry.AttrErrorType` (`errors-and-results.md` FR-034). `Instrumenter` only increments this counter and ends the span; the span status and the recorded error are set once through `Span.SetStatus` and `Span.RecordError` (`observability-ports.md` FR-012), never also by the use case. "Once" holds because no use case calls another (`shared-types-and-ownership.md` FR-044), so each failure passes through exactly one `Instrumenter`. A `refused` or `invalid` result MUST NOT increment it and MUST NOT be given an `error.type`; it is counted by `srd5e.combat.operations` with its `result_class` (D-9). | `errors-and-results.md` FR-034, FR-043, OQ-11; `observability-ports.md` FR-012; observability §5, §7; `shared-types-and-ownership.md` FR-044, FR-051 | unhappy (failure) |
| FR-033 | No `combat` attribute may carry a combatant id, `CharacterID`, `MonsterID`, an amount, or free text. Event kinds, operations, result names and every value passed through `Description.Extras` (FR-041) are closed sets. | observability §5 | unhappy (invalid input) |
| FR-034 | `combat` use cases MUST NOT log a rule outcome, refusal or invalid input at `LevelInfo` or above; the span, counter and result record it (observability §7). A spec needing a `LevelWarn` (a handled fallback) declares it in its own table. | observability §7 | unhappy (telemetry) |
| FR-035 | `Instrumenter` MUST be the one helper through which `combat` use cases record FR-029 to FR-032. It is built by `NewInstrumenter(t telemetry.Telemetry, clock func() time.Time)` (a zero `telemetry.Telemetry` and a nil clock default to `telemetry.Noop` and `time.Now`; `Instrumenter` is a value, its zero value is a silent instrumenter), wraps the work in the span, times it, classifies the result with a caller-supplied `Description`, applies the optional extras (FR-041), counts events, and for a returned `*srderr.PortError` increments `srd5e.combat.errors` once (FR-032) and returns the same error value (the span records it once). A panic raised by domain code inside the work is a bug (architecture §6): it MUST propagate unchanged, with the span ended and no `errors` increment. | observability §3, §4, §9; architecture §6 | happy |
| FR-036 | With the no-op and with the recording `telemetry.Telemetry`, a use case run through `Instrumenter` MUST return equal results. | observability §9 | happy |
| FR-037 | `doc/observability/signals.md` MUST NOT list the names this spec supersedes (section 5). The shared names `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.errors` and `srd5e.combat.events` each have exactly one catalogue row, owned by this spec (`architecture-guardrails.md` FR-044). A later spec needing extra attributes on a shared signal lists its extra keys and closed values in its own section 7 prose or extras table (no `Name` repeating a shared name) and in the Attributes cell of the shared row here, never a second row. Each per-operation span `srd5e.combat.<operation>` has one catalogue row, owned by its use-case spec (FR-040). | observability §10; `architecture-guardrails.md` FR-042 to FR-044 | unhappy (duplicate) |
| FR-038 | An event wrapping another spec's event for transport (an `...Occurred` envelope) MUST implement `transportEnvelope` (`interface` in `combat/`, not exported) and MUST report the wrapped event's `kind` and `combatants` from its methods. `Instrumenter` counts it as that one wrapped event, once, under the wrapped `kind` (for example `tracker_round_started`). An envelope is exempt from FR-017's constant and uniqueness rules, still belongs to its owning spec's closed family (FR-016), and its type name never appears as a `kind`. Architecture tests recognise envelopes by the marker and check the marked set equals the envelopes the owning spec names (for `encounter-aggregate.md`: `TurnEventOccurred`, `HitPointEventOccurred`, `ConditionEventOccurred`, `GrappleEventOccurred`). | `encounter-aggregate.md` FR-050, FR-084; observability §5 | unhappy (rule interaction) |
| FR-039 | `CombatSignals` MUST be a block of unexported untyped string `const`s in `combat/signals.go` (no type), named with a prefix `metric`, `span`, `msg`, `attr` or `value`, as `d20/signals.go` does for its exported ones (`architecture-guardrails.md` FR-037). This spec declares: `metricOperations` (`srd5e.combat.operations`), `metricOperationDuration` (`srd5e.combat.operation_duration`), `metricErrors` (`srd5e.combat.errors`), `metricEvents` (`srd5e.combat.events`); `attrOperation` (`srd5e.combat.operation`), `attrResult` (`srd5e.combat.result`), `attrResultClass` (`srd5e.combat.result_class`), `attrEvent` (`srd5e.combat.event`); `valueResultClassApplied` (`applied`), `valueResultClassRefused` (`refused`), `valueResultClassInvalid` (`invalid`) (each with a doc comment naming `srd5e.combat.result_class`); `valueOutcomeSuccess` (`success`), `valueOutcomeFailure` (`failure`) (doc comment naming `srd5e.outcome`; the key is `telemetry.AttrOutcome`, FR-027). `CombatResultClass.Outcome` MUST return these two constants and `Instrumenter` MUST record `result_class` from the three `valueResultClass...` constants. Later specs add their own `value...` constants for `srd5e.combat.result` (FR-030) and operation constants (FR-040). | `architecture-guardrails.md` FR-037, FR-085; observability §5 | happy |
| FR-040 | `NewCombatOperation(value, spanName string) (CombatOperation, error)` MUST hold the `snake_case` value (`Value()`) and its span name (`SpanName()`); `Instrumenter` uses the span name as given, never by concatenation. For each operation the owning spec adds to `CombatSignals` a pair `valueOperation<Operation>` (doc comment naming `srd5e.combat.operation`) and `span<Operation>` (`srd5e.combat.<operation>`), and one catalogue row for the span, so every catalogue Name equals a signal constant (`architecture-guardrails.md` FR-042, FR-043). The constructor returns an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when `spanName` does not end in `.` followed by `value`; an architecture test checks every `valueOperation...` has a `span...` of `srd5e.combat.` plus it, and the reverse. | `architecture-guardrails.md` FR-042, FR-043; observability §5 | unhappy (invalid input) |
| FR-041 | `Instrumenter.Run` MUST accept optional extras in `Description.Extras` (`[]telemetry.Attr`). They MUST be added to the span and the `srd5e.combat.operations` increment of that call, and nothing else (not `operation_duration`, `errors` or `events`). Every key MUST be a `CombatSignals` `attr...` constant starting `srd5e.combat.`, or a key owned by another context's signals under FR-027; every value MUST come from a closed, low-cardinality set (FR-033). An extra whose key is a shared attribute of FR-030 MUST make `Run` return, before the work runs, an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. A use case omits an extra that does not apply (no d20 rolled) by leaving the key out. The owning spec lists its extra keys and values (FR-037). Registered extras, all `CombatSignals` `attr...` constants: `srd5e.combat.roster_reason` (`summoned`, `arrived`, `fled`, `surrendered`, `disappeared`, `departed`, `repositioned`; `combatant-roster-changes.md` FR-043, with its three operation constants and spans per FR-040); `srd5e.combat.spell_table` (`teleport`, `confusion`, `prismatic_spray`, `prismatic_wall`, `reincarnate`, `blink`; `spell-outcome-tables.md` FR-101); `srd5e.combat.monster_hit_points_method` (`average`, `rolled`; `monster-combatant-setup.md` FR-049); `srd5e.combat.damage_notation` (`number`, `dice`, `mixed`; `monster-stat-block-actions.md` FR-047). `srd5e.combat.summon_kind` (`animated_object`, `giant_insect`, `draconic_spirit`, `otherworldly_steed`, `animated_undead`, `created_undead`, `familiar`, `simulacrum`) and `srd5e.combat.summon_end_cause` (`dropped_to_zero_hit_points`, `spell_ended`, `summoner_died`, `long_rest_finished`, `dismissed_forever`, `dismissed_temporarily`, `replaced`, `control_lapsed`) are attributes of the owner's own counters `summons` and `summon_endings` (`summoned-and-companion-creatures.md` section 7), not of `operations`. | `attacks.md` FR-116, FR-118; `mounted-and-underwater-combat.md` FR-061; observability §5 | unhappy (boundary) |

### 4.6 `CombatRuleOptions`

Moved here from `rule-options.md` (decisions FR-057); follows that spec's pattern and shared rules (FR-023, FR-029, FR-033, FR-034). The one option is the SRD sidebar "Playing on a Grid" (p. 13). `grid-positions-and-movement.md` FR-001 reads `grid` (an exhaustive `switch` yields the `SquareGrid` its operations take), so `CombatRuleOptions` and `PlayingOnAGrid` (`NoGrid`, `SquareGrid`) are exported from the `combat` package (`rule-options.md` FR-029).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | `CombatRuleOptions` MUST be a comparable value holding one unexported field, `grid` (`PlayingOnAGrid`), read through `Grid()`. `StandardCombatRuleOptions()` MUST return the value using `NoGrid` (the zero value is the same standard value). `NewCombatRuleOptions(grid PlayingOnAGrid)` and `WithGrid(grid PlayingOnAGrid) CombatRuleOptions` MUST return a new value changing only the grid and leave the receiver unchanged. It has no invalid state: no error return, no result type. | SRD 5.2.1, "Playing on a Grid", p. 13; architecture.md §3 | happy / unhappy (rule interaction) |
| FR-043 | `PlayingOnAGrid` MUST be a closed interface (unexported marker method) with exactly two variants, `NoGrid` and `SquareGrid`, each a comparable value struct (equal by value). | SRD 5.2.1, "Playing on a Grid", p. 13 | happy |
| FR-044 | `SquareGrid.FeetPerSquare()` MUST return 5. `SquareGrid.SpeedInSquares(feet int) (int, error)` MUST return `feet` divided by 5, rounded down (table below). | SRD 5.2.1, "Playing on a Grid", "Squares" and "Speed", p. 13; "Round Down", p. 5 | happy / unhappy (boundary) |
| FR-045 | `SquareGrid.SpeedInSquares` MUST return 0 and an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative number of feet. | architecture.md §6 | unhappy (invalid input) |

`SquareGrid.SpeedInSquares` (FR-044). Each row is one table-driven test.

| Label | feet | Expected | Source | Path |
|---|---|---|---|---|
| the SRD's own example | 30 | 6 | p. 13 | happy |
| zero Speed | 0 | 0 | p. 13 | unhappy (boundary, minimal) |
| one square | 5 | 1 | p. 13 | unhappy (boundary) |
| one foot short of a square | 4 | 0 | p. 13; p. 5 | unhappy (boundary) |
| one foot over a square | 6 | 1 | p. 13; p. 5 | unhappy (boundary) |
| non-multiple rounds down | 32 | 6 | p. 13; p. 5 | unhappy (rule interaction) |
| large Speed | 120 | 24 | p. 13 | happy |

### Withdrawn ids

None.

## 5. Model

- `CombatantID` (named string type; `NewCombatantID` enforces FR-002 to FR-004 and returns a `*CombatantIDError` carrying a `CombatantIDRejection`, an enum with the single value `CombatantIDEmpty`, the constructor's only failure; it satisfies `errors.Is(err, srderr.ErrInvalidInput)`).
- `CombatantKind` closed; `CombatantRef{ID, Kind}`; `Event` (FR-015); `transportEnvelope` (FR-038); `CombatOperation` (FR-040); `CombatSignals` (FR-039); `CombatResultClass` enum with `Outcome` and `IsSpanError`.
- `CombatRuleOptions` (one `grid` field) and closed `PlayingOnAGrid` (`NoGrid`, `SquareGrid`; only `SquareGrid` has behaviour, FR-043 to FR-045).

**Failures.** `NewX` constructors return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for invalid arguments; `MustX` helpers panic with that error and are for literals and tests only. No `*srderr.PortError` is created here: nothing calls a port; `Instrumenter` only counts and passes through the port errors of the work.

**Files.** One package, `combat`. `combatant_id.go`, `combatant_kind.go`, `combatant_ref.go`, `combat_event.go` are pure and silent. `combat_operation.go`, `result_class.go`, `signals.go`, `instrumenter.go` are the only files that touch `telemetry.Telemetry`.

### Superseded definitions

The supersession list: where a row says superseded, the other spec changes to match; this spec wins (decisions FR-012, FR-015).

| Spec | What it assumed | Outcome |
|---|---|---|
| `hit-points-and-damage.md` | `CombatantID` and `Event` come from here; `HitPointEvent` implements `Event`. FR-080: rejections (`CreatureIsDead`, `NotDying`, `NotStable`) set error status. FR-079: defines `operations`, `errors`, `operation_duration`. | Confirmed: events gain `kind` and `combatants` (FR-015, FR-017, FR-018). FR-080 now says `refused`, `failure`, no error status (FR-031); its result values (`damaged`, `killed`, `rejected_dead`, `not_dying`, `not_stable`) model `srd5e.combat.result` (FR-030). The three shared signals **moved** here (FR-029, FR-032); its use-case-specific signals stay. |
| `conditions.md` | D-1: `Charmed`, `Frightened`, `Grappled` and condition events carry `CombatantID`. Own signals `srd5e.combat.condition.apply`, `.remove`, `.reduce_exhaustion` (spans), `.operations`, `.duration`, and attributes `srd5e.combat.condition.operation`, `.outcome`. | Confirmed for ids; events gain `kind` and `combatants`. Signals **superseded**: spans are `srd5e.combat.<operation>`; throughput, duration, errors use the shared signals (FR-028, FR-029); the attributes become `srd5e.combat.operation`, `srd5e.combat.result` plus `result_class`. `.active` and per-kind detail stay. |
| `attacks.md` | No creature type (no `AttackTarget`). Own signals `srd5e.combat.attack.resolve` (span), `.resolved`, `.duration`, `.errors`, `.modifiers`, and `srd5e.combat.unarmed_strike.resolve` (span), `.resolved`, `.duration`, `.errors`. | No-creature assumption confirmed. The `.errors` signals duplicate `srd5e.combat.errors` and are removed (FR-037); spans are `srd5e.combat.resolve_attack` and `srd5e.combat.resolve_unarmed_strike` (FR-040) with the shared `operations` and `operation_duration`. `.attack.modifiers` stays (`attacks.md` section 7 lists it as a non-duplicate). Extras (`srd5e.combat.attack.form`, `.cover`, `.automatic_critical`, `.reason`, `srd5e.combat.unarmed_strike.option`, `srd5e.d20_test.roll_mode`, `srd5e.d20_test.critical_hit`) go through `attributes` (FR-041); its second `operations` row becomes prose (FR-037). |
| `initiative-and-actions.md` | Ids and naming come from here; refusals set no error status; shares `operation_duration` and `errors`. | Confirmed, including the unknown-combatant case: `invalid` (failure plus error status, FR-031), as in `encounter-aggregate.md` FR-046, FR-059 and `initiative-and-actions.md` FR-095, FR-099; an error is `invalid`. `srd5e.combat.use_case` becomes `srd5e.combat.operation` (FR-028); `srd5e.combat.refusal` values become `srd5e.combat.result` values. |
| `encounter-aggregate.md` | `CombatantKind` with `PlayerCharacter` / `Monster`; a parse function. Own signals `encounter_transitions`, `encounter_errors`, `encounter_duration`, `encounter_events`; attributes `srd5e.combat.rejection`, `.error_kind`; result values `applied` / `rejected` / `invalid`. `DeathSavingThrowDue` and `...Occurred` envelopes. | **Superseded**: variants are `PlayerCharacterCombatant` / `MonsterCombatant` (FR-010); the constructor is `NewCombatantID` (FR-005, `character-aggregate.md` FR-003); `Combatant` holds a `CombatantRef`. Signals use the shared four; the coarse values become `result_class` (`rejected` is `refused`) and `result` holds a finer owned value (FR-030). `DeathSavingThrowDue` became `DyingCombatantTurnStarted` (FR-020; done, `encounter-aggregate.md` FR-018); the four envelopes follow FR-038. `encounter_size`, `encounter_deaths`, `death_cause` stay. `EncounterEvent` is a closed sub-family (FR-016, D-1). |
| `combat-action-resolution.md` | FR-094 named `MovementSpeedsReviewDue`, `GrappleCheckDue`. Sibling use cases built with `telemetry.Noop`. | **Superseded**: renamed `...BecameDue` (FR-020; done). No use case calls another (decisions FR-044), so FR-032's "once" needs no silenced telemetry (done: its FR-007). |
| `magical-effects.md` | A second section 7 row named `srd5e.combat.operations`. | **Superseded** by FR-037: prose; extras via FR-041. |
| `mounted-and-underwater-combat.md` | FR-061: extras `srd5e.combat.mount_control`, `.falling_off_trigger` "need a `Instrumenter` extension". | The extension is FR-041; the keys are `attr...` constants in `CombatSignals` (FR-039) and listed in the shared rows' Attributes cells (FR-037). |

## 6. Interface

No ports, no I/O; everything is synchronous.

```go
// Identity (combatant_id.go, combatant_kind.go, combatant_ref.go).
type CombatantID string
func NewCombatantID(value string) (CombatantID, error) // *CombatantIDError (Is srderr.ErrInvalidInput) if blank
func MustCombatantID(value string) CombatantID         // panics with that error
func (id CombatantID) IsZero() bool
type CombatantIDRejection int                           // CombatantIDEmpty
type CombatantIDError struct{ Reason CombatantIDRejection }

type CombatantKind interface{ /* unexported marker */ } // PlayerCharacterCombatant | MonsterCombatant
type PlayerCharacterCombatant struct{ ID character.CharacterID }
type MonsterCombatant struct{ ID content.MonsterID }
type CombatantRef struct {
	ID   CombatantID
	Kind CombatantKind
}
func NewCombatantRef(id CombatantID, kind CombatantKind) (CombatantRef, error)

// Events (combat_event.go).
type Event interface {
	event() // unexported marker
	Kind() string
	Combatants() []CombatantID // a copy
}
type transportEnvelope interface{ transportEnvelope() } // marker for the ...Occurred envelopes (unexported)

// Telemetry helpers (unexported in Go; written capitalised in the specs).
type CombatOperation struct{ /* value, spanName */ }
func NewCombatOperation(value, spanName string) (CombatOperation, error)
func (o CombatOperation) Value() string
func (o CombatOperation) SpanName() string
// const block metric..., span..., msg..., attr..., value... in signals.go (FR-039)
type CombatResultClass int // ResultApplied, ResultRefused, ResultInvalid
func (c CombatResultClass) Outcome() string
func (c CombatResultClass) IsSpanError() bool
func (c CombatResultClass) String() string

type Description struct {
	Class  CombatResultClass
	Name   string           // the srd5e.combat.result value
	Events []Event
	Extras []telemetry.Attr // span and `operations` only (FR-041)
}
type Instrumenter struct{ /* telemetry.Telemetry, clock */ }
func NewInstrumenter(t telemetry.Telemetry, clock func() time.Time) Instrumenter // zero value is silent
func Run[R any](ctx context.Context, in Instrumenter, op CombatOperation,
	work func(context.Context) (R, error), describe func(R) Description) (R, error)
```


`Run` is a generic function because Go methods cannot have type parameters. It is synchronous and takes `ctx` only to hand it to the span, the instruments and the work.

Exports: the `combat` package exports `CombatantID`, `NewCombatantID`, `MustCombatantID`, `CombatantIDError`, `CombatantIDRejection`, `CombatantKind` and its variants, `CombatantRef` and `Event`. `CombatOperation`, `CombatResultClass`, `Instrumenter`, `CombatSignals` and `transportEnvelope` stay unexported (written capitalised in the specs as concepts; the Go identifiers are lower-case; they are application helpers, not handoff types; an export is a compatibility promise, architecture §7, §10; signal names are already public contract through the catalogue). `CombatRuleOptions` and `PlayingOnAGrid` with its variants are exported because `grid-positions-and-movement.md` FR-001 reads `grid` (`rule-options.md` FR-029); the `combat` library doc comment states that an option is a value in the command, never a global setting (`rule-options.md` FR-034). No new test double is needed: `Instrumenter` uses the existing recording `telemetry.Telemetry`.

## 7. Observability

The domain pieces (ids, kinds, events) are **silent by design**. The signals below are emitted by later `combat` use cases through `Instrumenter`; this spec owns the shared rows and each later spec its own extras. All rows feed `doc/observability/signals.md`.

1. **What can go wrong?** A blank id (rejected by `NewCombatantID` before any use case); two specs defining a signal differently; a port returning; a refused action (normal); invalid host input; a reused event kind; an id leaking into an attribute.
2. **How would an operator know?** Port failures: `srd5e.combat.errors` with `error.type` plus span error status. Invalid input: `operations` with `result_class` = `invalid` plus span error status. Refusals: `refused`, no error status. Duplicate kinds and leaked ids are caught at build time (FR-017, FR-033).
3. **What is needed to diagnose?** Operation, result name, result class and event kinds, all low-cardinality; ids stay in results and events (FR-033).
4. **What to count or time?** Calls and outcome per operation; duration per operation; events per kind; port failures per error type.
5. **Is existing instrumentation still correct?** Not entirely: section 5 lists what the earlier specs defined and this spec replaces.
6. **What to monitor or alert on?** See the table.
7. **Is anything noisy?** No logs (FR-034). Counters are bounded by the use-case and event-class lists.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.operations` | counter `{operation}` | `srd5e.combat.operation`, `srd5e.outcome` (`success`, `failure`), `srd5e.combat.result`, `srd5e.combat.result_class` (`applied`, `refused`, `invalid`), plus closed-value extras (FR-037, FR-041): `srd5e.combat.attack.form`, `srd5e.combat.attack.cover`, `srd5e.combat.attack.automatic_critical`, `srd5e.combat.attack.reason`, `srd5e.combat.unarmed_strike.option`, `srd5e.d20_test.roll_mode`, `srd5e.d20_test.critical_hit` (`attacks.md`); `srd5e.combat.mount_control`, `srd5e.combat.falling_off_trigger` (`mounted-and-underwater-combat.md`); `srd5e.combat.roster_reason` (`summoned`, `arrived`, `fled`, `surrendered`, `disappeared`, `departed`, `repositioned`; `combatant-roster-changes.md`); `srd5e.combat.spell_table` (6 values; `spell-outcome-tables.md`); `srd5e.combat.monster_hit_points_method` (`average`, `rolled`; `monster-combatant-setup.md`); `srd5e.combat.damage_notation` (`number`, `dice`, `mixed`; `monster-stat-block-actions.md`) | Throughput and outcome of every combat use case | Rate by operation and `result_class`; alert if `invalid` exceeds 1% of an operation's calls over 15 minutes; watch a sudden shift in the `refused` share |
| `srd5e.combat.operation_duration` | histogram `s` | `srd5e.combat.operation` | Latency of each use case, through `telemetry.StartTimer` | p99 above 10 ms for 5 minutes, or 3 times its 7-day baseline; usually a slow host adapter |
| `srd5e.combat.errors` | counter `{error}` | `srd5e.combat.operation`, `error.type` | Port failures leaving a use case (for example `dice_roller_error`) | Alert on any increment; page if above 0.1% of calls over 5 minutes |
| `srd5e.combat.events` | counter `{event}` | `srd5e.combat.event` | Domain events produced, by kind | Dashboard rate by kind; investigate a kind dropping to zero after a release |

**Spans.** Each use case runs in one span `srd5e.combat.<operation>` (FR-029) carrying `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result`, `srd5e.combat.result_class` and any FR-041 extras; error status only for `invalid` results and returned errors. The template is not a signal name, so no span row is listed here; each use-case spec lists the row for its own span (FR-040). Suggested monitoring: trace sampling, alert on a sustained rise in error-status spans for one operation.

Value sets are closed: `srd5e.outcome` is `success` or `failure`; `result_class` is `applied`, `refused` or `invalid`; `error.type` is the `*srderr.PortError` constant set of `errors-and-results.md`; `operation`, `result` and `event` are closed sets listed by the owning specs.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-009, FR-012, FR-014 | architecture | `internal/archtest` scan: one `CombatantID`, declared in `combat`, no second creature-id type. Scan of package `combat` for `math/rand`, `crypto/rand` and uuid imports, and any function returning `CombatantID` other than `NewCombatantID` and `MustCombatantID`. Kind variants' exported members are the reference id only. Reference-type import allowlist: standard library, `character` and `content` published API, kernel packages; fails on another `combat` spec file, `d20`, `toolbox`, or an `internal/` package of another context |
| FR-002 to FR-006 | unit, fuzz | Accepted and unchanged: `"a"`, `"goblin_1"`, `" a"`, `"a "`, `"\u00A0a"`. Rejected (`srderr.ErrInvalidInput`, all `strings.TrimSpace(s) == ""`): `""`, `" "`, `"\t"`, `"\n"`, `"\r"`, `"\u00A0"`, `"  \n\r "`. `NewCombatantID` over the same table returns the id with a nil error or an error (reason `CombatantIDEmpty`, `errors.As`-able); `MustCombatantID` panics with that error iff `NewCombatantID` returns one. `FuzzNewCombatantID` over extreme strings (very long, unicode, empty): never panics. Three differently cased/padded ids are pairwise unequal |
| FR-007, FR-011, FR-013 | unit | Equal strings: equal and same hash; different strings differ; usable as map keys. Kind and ref equality pairs, including `PlayerCharacterCombatant(CharacterID('x'))` versus `MonsterCombatant(MonsterID('x'))` and the same id with a different kind |
| FR-008 | architecture | `go/types` fixture: type-checking a snippet that assigns a `CombatantID` to an `EncounterID`, `CharacterID` or `MonsterID` variable (and back) reports an assignment error |
| FR-010 | unit | Exhaustive `switch` over `CombatantKind` compiles with two cases; each variant exposes its id |
| FR-015 to FR-017 | unit, architecture | A test event embeds `Event` to satisfy the marker; `Event` is an interface with `Kind` and `Combatants`. Lists every implementer; fails on a direct implementer or a non-closed sub-family; an envelope in a closed family passes. Scan collects every non-envelope `kind` constant: matches the pattern, unique; envelopes (marker) skipped; `character.Event` ignored; no `reflect` or `%T` in event code; equal events, equal kinds |
| FR-018, FR-021 | unit | Self-test of helper `combat/combattest/event_family.go` (package `combattest`) against a fake family (a good one passes; a returned slice aliasing stored state, a duplicate, an unordered list, and a result with a modifiable events list each fail). Per owning spec using the helper: `combatants` read-only (accessors return copies), ordered, duplicate-free, empty only for no-creature events; appending to a returned events slice leaves the stored events unchanged; order preserved |
| FR-019 | architecture | Scan: unexported fields and value receivers only, on every implementer. Unit (owning specs): value equality including `kind` and `combatants` |
| FR-020 | architecture | No event type name matches `(?<!Became)Due$\|Required$\|Needed$` (`MovementSpeedsReviewBecameDue` passes; `DeathSavingThrowDue` and `RollRequired` fail). Tense is otherwise a naming review |
| FR-022 to FR-025 | architecture | Package scan fails on the banned generic names, a result or command without the `<UseCase>` prefix, a `*Rejected` / `*Invalid` variant without a `*Rejection` / `*Problem` field, and a duplicate exported name (lower-rank spec wins; alphabetical at equal rank). Each reserved name is declared exactly once, in this spec's files |
| FR-026 | architecture | Import-direction check inside `combat` using the 4.11 ranks; fails on an import of `toolbox`, of an `internal/` package of `character`, `d20` or `content`, of another context's use case, and of a non-published-language type in a domain file. Passes a `d20.Resolver` import in a use-case file |
| FR-027, FR-033 | architecture | Every `srd5e.combat.` name appears only in `CombatSignals`; no literal at a call site; `CombatSignals` contains neither `srd5e.outcome` nor `error.type` nor any `srd5e.d20_test.*` key; `combat` code uses `telemetry.AttrOutcome`, `telemetry.AttrErrorType` and the other context's exported constants; catalogue and constants match. Attribute-building code in the `combat` use-case files reads no id, amount or free text; a recording-double test asserts no attribute value equals a supplied `CombatantID` |
| FR-028 | unit, architecture | `NewCombatOperation("apply_damage", "srd5e.combat.apply_damage")` accepted; `""`, `"Apply"`, `"apply damage"`, `"1x"` return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`, as does a span name not ending in `.apply_damage`. No `srd5e.combat.use_case` or `srd5e.operation` remains in the catalogue |
| FR-029, FR-041 | application, unit | `Instrumenter` with the recording `telemetry.Telemetry`: one span name, one `operations` increment, one `operation_duration` sample, one `events` increment per event (zero events, zero increments). Extras from `Description.Extras` appear on the span and the `operations` increment only; an absent entry is omitted; a shared key in the extras makes `Run` return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`; no extras records the shared set only. Error path: one `operation_duration` sample, no `operations` or `events` increment |
| FR-030, FR-031 | unit, application, architecture | Recorded attribute sets equal FR-030's lists (plus FR-041 extras); fails on a `srd5e.outcome` or `result_class` value outside its set and on a catalogue `srd5e.combat.result` value with no `CombatResultClass`; a fake use case whose `resultName` values each map to one class; one `result` value never appears under two classes. `CombatResultClass` table: `applied` is `success`, not error; `refused` is `failure`, not error; `invalid` is `failure`, error; span status set accordingly |
| FR-032, FR-035 | application, architecture | A work function returning a test `*srderr.PortError`: one `errors` increment with `error.type`, error span, exception returned, recorded once (no `Span.RecordError` by `Instrumenter`). A `refused` or `invalid` result: no `errors` increment. A panic from domain code propagates unchanged, the span records it, no `errors` increment. No `combat` use case holds or calls another use case (decisions FR-044) |
| FR-034, FR-036 | application | Recording logger sees no record at `LevelInfo` or above for applied, refused and invalid results. Same work, same result with `telemetry.Noop` and the recording one |
| FR-037 | architecture | Catalogue consistency: each shared name has one row; no superseded name listed; no second row repeats a shared name (`attacks.md` and `magical-effects.md` extras are prose); the shared rows' Attributes cells write `srd5e.outcome` (`success`, `failure`) and `srd5e.combat.result_class` (`applied`, `refused`, `invalid`) |
| FR-038 | unit, application, architecture | A fake envelope wrapping a fake event: `kind` and `combatants` equal the wrapped event's; `events` incremented once under the wrapped `kind`; scans accept the marked envelope and fail on a non-envelope duplicate or constant-less kind; the set of `transportEnvelope` types equals the named list |
| FR-039, FR-040 | architecture, unit | `CombatSignals` is a block of unexported string `const`s with legal prefixes; the FR-039 constants and values exist; `CombatResultClass.Outcome` returns the two `valueOutcome...` constants (unit); `result_class` is recorded from the three `resultClass...Value` constants. `CombatOperation` accepts a matching pair and rejects a mismatched span name; every `valueOperation...` has a `span...` of `srd5e.combat.` plus it, and the reverse; each per-operation span has one catalogue row and every `srd5e.combat.` span Name in the catalogue equals a constant |
| FR-042, FR-043 | unit, architecture | `StandardCombatRuleOptions().Grid()` is `NoGrid` and equals the zero value; `==` by value, `WithGrid` changes only the grid and leaves the receiver unchanged; the variants are comparable value structs (`errors-and-results.md` FR-018 scan); an exhaustive `switch` over `PlayingOnAGrid` maps the two variants to two distinct values; option files are covered by the `rule-options.md` FR-023 scan |
| FR-044, FR-045 | unit | `FeetPerSquare() == 5`; the seven table rows of 4.6; -1 and `math.MinInt` return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`; boundary rows kill `<` / `<=` and off-by-one mutants |

- **Doubles and contract suites.** None: there is no port. The recording `telemetry.Telemetry` comes from the `<pkg>test` packages. Event-family tests (FR-018, FR-019, FR-021) reuse the `combattest` event-family checks; each `combat` spec's test plan MUST cite it for its event family and result types (checked by the spec-consistency critic).
- **Mutation testing.** `combat/` and `CombatResultClass` at score 80 or more (`doc/rules/testing.md` §8). `Instrumenter` SHOULD be mutation tested.
- **Not covered by a runtime test:** FR-008 (static property, `go/types` fixture) and the tense half of FR-020 (naming review).
- **Layout.** `combat/combatant_id_test.go`, `combatant_kind_test.go`, `combatant_ref_test.go`, `instrumenter_test.go` (external `combat_test` package for public behaviour, `Fuzz*` for `NewCombatantID`); `combat/combattest/event_family.go` with self-test `event_family_test.go`; `internal/archtest/combat_*_test.go`. Each test cites its FR id.

## 9. Open questions and decisions

No open questions. Settled:

- D-1 (`Event` has an unexported marker, FR-019): outside packages cannot add events, and test doubles in `combattest` embed `Event` to implement it (FR-038 fake envelopes); naming the sub-families here would point the leftmost spec at later ones; each sub-family closes itself with its own marker; `shared-types-and-ownership.md` FR-019; FR-015, FR-016.
- D-2 (ids carry no kind): the host may field two goblins; an id derived from the kind would make two goblins equal; the id is host-assigned and unique, the `MonsterID` is a separate fact.
- D-3 (`kind` and `combatants` on every event): routing, filtering and counting need no reflection.
- D-4 (a refusal is not an error status): observability §6-§7; FR-031.
- D-5 (one `Instrumenter` helper): uniform telemetry by construction; FR-035.
- D-6 (the "source of fear" need not be a creature): out of scope; settled in `conditions.md` D-2: the source of fear is a `CombatantID` and no `FearSource` type is added.
- D-7 (no trimming; blank means `strings.TrimSpace(value) == ""`): FR-002, FR-006.
- D-8 (`MonsterID` is owned by `content-schema-and-decoder.md`): decisions FR-046.
- D-9 (a returned failure carries no `error.type`): decisions FR-051; FR-032.
- D-10 (the signal constants are exported from the `d20` package): `resolve-d20-test-use-case.md` FR-042.
- D-11 (header lists only lower-rank specs): decisions FR-056.
- D-12 (`CombatRuleOptions` lives here, type phase 4): decisions FR-057; Monster Death and the Initiative method are not options (`hit-points-and-damage.md` FR-087; `initiative-and-actions.md` FR-010).

## 10. Changelog impact

`Added`: `CombatantID`, `NewCombatantID`, `MustCombatantID`, `CombatantKind` (`PlayerCharacterCombatant`, `MonsterCombatant`), `CombatantRef`, `Event` (`Instrumenter`, `CombatOperation`, `CombatResultClass`, `CombatSignals`, `transportEnvelope` are not exported); the shared signals `srd5e.combat.operations`, `srd5e.combat.operation_duration`, `srd5e.combat.errors`, `srd5e.combat.events`.

Also **Added** (exported under `rule-options.md` FR-029): "`combat`: `CombatRuleOptions` (playing on a grid)". Purely additive; `StandardCombatRuleOptions` is the SRD baseline.

Not **BREAKING** if it ships in the first release containing `combat`. Once any of those signals or names is released, renaming one is **BREAKING** (`doc/rules/architecture.md` §7, `doc/rules/observability.md` §5).
