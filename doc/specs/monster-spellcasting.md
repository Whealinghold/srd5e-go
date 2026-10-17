# Monster spellcasting

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `d20-tests.md`, `content-monsters.md`, `content-spells.md`, `spell-effect-data.md`, `spellcasting.md`, `casting-and-concentration.md`, `combat-foundations.md`, `conditions.md`, `monster-limited-use.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `errors-and-results.md` |

Dependency notes (prose only; shared-types-and-ownership.md FR-056): rank 27 (after `monster-limited-use` at 26, beside `spell-effects`, which this spec does not depend on; the added dependencies are ranks 1 to 2, so the order holds). The rank row is listed in table 4.11 (rank 27). `spell-effects.md` is the consumer of the report built here (section 6), not a dependency. Types ship with the `combat` phase (FR-014; [../roadmap.md](../roadmap.md)).

## 1. Purpose and scope

A monster's stat block can have a Spellcasting trait that lists spells, the spellcasting ability, a printed spell save DC and spell attack bonus, and a statement about components (SRD 5.2.1, "Spellcasting", p. 257). Nothing owns how such a monster casts. `casting-and-concentration.md` (section 1) and `spell-effects.md` (section 1) both exclude monster Spellcasting; `content-monsters.md` FR-046 stores `castAtLevel` and nothing enforces it; `monster-limited-use.md` counts `N/Day` uses and leaves "which spell a monster casts" to the host.

This spec is the pure, state-in/state-out rule set for one cast by a monster:

- the spell is cast at its lowest possible level, or at the version the stat block states, and never higher (FR-003 to FR-007);
- the printed save DC and spell attack bonus are used as printed, never recomputed (FR-009 to FR-012);
- the "self only" restriction, a "no Concentration" statement and the components statement are honoured (FR-013 to FR-019);
- a spell with a casting time of 1 minute or more needs the Magic action on each of the monster's turns and Concentration (FR-022 to FR-027);
- per-day uses are counted through `monster-limited-use.md` (FR-020, FR-021);
- the result is a `MonsterCastReport` that `spell-effects.md` consumes (FR-030).

It is domain-only: no dice, no port, no use case, silent by design (section 7).

**Out of scope:**

- What a cast does (targets, saves, damage): `spell-effects.md`; the roll of a spell attack: `attacks.md`.
- Which spell a monster chooses, and the Magic action's place in the turn: the host and `initiative-and-actions.md`.
- The Concentration saving throw after damage: a D20 Test the host runs (`ConcentrationSaveDC.ForDamage`, `casting-and-concentration.md` FR-078, FR-079); this spec only ends Concentration when told (FR-028).
- Spell slots, Spell Scrolls, Rituals, preparing spells and Pact Magic: a monster has none of them. A monster cast is never a Ritual (interpretation: the stat block says the monster "casts" the spell, p. 257).
- The Bonus Action and Reaction triggers: the host reports that the monster's part may be used (`monster-limited-use.md` section 1 leaves what a part does to the host). A stat-block part that casts a spell ("Misty Step (3/Day)"), with or without a Spellcasting trait, is castable here through `PartCast` (FR-021, FR-039).
- Lair, Legendary Action timing and Recharge: `monster-limited-use.md`.
- Content that is not in SRD 5.2.1: nothing core was found missing. The SRD gives no cantrip upgrade rule for creatures without a character level; FR-006 records the reading.

## 2. Source

- SRD 5.2.1, "Monsters", "Spellcasting" (lowest possible level, "self only", Spell Components, Casting Times of 1+ Minutes), p. 257; "Limited Usage", p. 257.
- "Spells": "Casting Time" ("One Spell with a Spell Slot per Turn", "Longer Casting Times"), p. 105; "Components" (Verbal, Somatic, Material), pp. 105-106; "Duration", "Saving Throws", "Attack Rolls", p. 106; "Using a Higher-Level Spell Slot", pp. 104-105.
- Rules Glossary, "Concentration", p. 179; "Incapacitated", p. 184; "Cantrip", p. 178; "Multiclassing", "Spellcasting" (Cantrips by total character level), p. 25.
- Stat blocks: "Archmage" (Lightning Bolt and Cone of Cold versions, printed DC 17), "Mage" (Misty Step 3/Day, Fireball level 4 version), p. 305; "Cloaker" (Phantasms), p. 274; "Pit Fiend" (Hellfire Spellcasting), p. 315; "Priest", "Priest Acolyte", p. 316; "Green Hag" (Invisibility "self only"), p. 295; Plane Shift "self only", p. 311; "Adult Brass Dragon", p. 269 (Shapechange note); "Deva", p. 279; "Ancient Green Dragon" (Mind Spike "level 5 version"), p. 295; spells with 1+ minute casting times: Geas (1 minute), p. 135; Scrying (10 minutes), p. 160.
- Package design: `doc/rules/architecture.md` §3, §4, §6; `doc/rules/observability.md` §2, §3; `doc/rules/testing.md` §4, §5, §8; shared-types-and-ownership.md FR-011, FR-044, FR-048.
- Web research (informs judgement only, nothing is copied): community consensus agrees that a creature's spell with no stated level is cast at its base level and its cantrips do not scale; the SRD sentence "always cast at its lowest possible level" (p. 257) is the authority used.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Spellcasting trait, spellcasting ability, spell save DC, spell attack bonus | `MonsterSpellcasting` (`content-monsters.md` FR-044) held by `MonsterSpellcaster` | Printed values, kept as printed. |
| (a spell in the list) | `MonsterSpellEntry` (name, note, `castAtLevel?`; `content-monsters.md` FR-046), addressed by `SpellUseRef(groupIndex, spellIndex)` (`monster-limited-use.md` FR-044) | Package ref reused. |
| "lowest possible level", "(level N version)" | `castAtLevel` of the entry | p. 257. |
| "self only" | `MonsterSpellRestrictions.SelfOnly` | p. 257. |
| Spell Components | `ComponentWaiver`, `ComponentKind` (`casting-and-concentration.md`) | "requiring no Material components". |
| Casting Times of 1+ Minutes | `CastingInProgress` (`spellcasting.md` FR-118 family), `MonsterCastStarted` | Magic action each turn plus Concentration. |
| Concentration | `Concentration`, `ConcentrationSubject`, `ConcentrationKind`, `ConcentrationBreakReason` (`spellcasting.md`, `casting-and-concentration.md`) | Reused value types. |
| Incapacitated | `ConditionActionAvailability` (`conditions.md` FR-064) | `none` means no action. |
| (a stat-block part that casts) | `PartCast(LimitedUsePartRef, ...)` | Package term. |
| (a monster in a fight) | `MonsterSpellcaster(combatant, spellcasting, abilityModifier)`, `MonsterSpellcastingState` | Package terms. |
| (hand-off to effects) | `MonsterCastReport` | Package term; builds a `CastReport` (FR-030). |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. "The trait" is the `MonsterSpellcasting` of the caster.

### 4.1 Which spell, at what level

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `NewMonsterSpellcaster(monster, combatant)` MUST find the part named `Spellcasting` and hold its `MonsterSpellcasting` and the Modifier of each of the monster's six scores (the trait's ability is the default); it MUST return an error matching `srderr.ErrInvalidInput` for a monster without one (the same error for a monster with two such parts); there is no nil-returning variant. `MonsterSpellcasterForParts(monster, combatant)` MUST build a caster with no trait for a monster whose parts cast on their own (FR-039); a `ListCast` by it is refused with `notInSpellList`. | SRD 5.2.1, "Spellcasting", p. 257 | unhappy (missing content) |
| FR-002 | A cast from the list (`ListCast(SpellUseRef)`) MUST take the entry at the ref and the caller's `Spell`; the `Spell.Name` MUST equal the entry's name exactly, else an error matching `srderr.ErrInvalidInput`. A ref naming no group or no spell MUST be refused with `notInSpellList`. | SRD 5.2.1, "Spellcasting" (stat block lists the spells), p. 257 | unhappy (invalid input) |
| FR-003 | A level 1+ spell MUST be cast at the entry's stated `castAtLevel` when it has one ("(level N version)"), else at its own level ("lowest possible level"). The cast level is the report's `castAtLevel`. Table CL-1. | SRD 5.2.1, "Spellcasting" ("always cast at its lowest possible level"), p. 257; "Archmage", p. 305 | happy |
| FR-004 | A stated `castAtLevel` below the spell's level, or any `castAtLevel` on a cantrip, MUST return an error matching `srderr.ErrInvalidInput` (a content or caller defect, not a rule). A stated level above the spell's level is the only way a level exceeds the spell's own. | SRD 5.2.1, "Spell Level", p. 104 | unhappy (boundary) |
| FR-005 | No input of any cast MAY raise the level: a `ListCast` has no level parameter, and a `PartCast` takes the part's stated version only. A monster never "upcasts" (higher-level slot): there is no slot to choose. | SRD 5.2.1, "Spellcasting" ("can't be cast at a higher level"), p. 257 | unhappy (rule interaction) |
| FR-006 | A cantrip MUST have no `castAtLevel` and `scalingLevel` 1 (the base Cantrip Upgrade tier). Interpretation: Cantrip Upgrade runs on total character level (p. 25), a monster has none, and "lowest possible level" (p. 257) is the base; no SRD stat block lists a damaging cantrip, so nothing is lost. | SRD 5.2.1, "Spellcasting", p. 257; "Multiclassing", "Spellcasting", p. 25 | unhappy (boundary) |
| FR-007 | A level 1+ cast MUST have `scalingLevel` equal to `castAtLevel`, as for a character slot cast. | `casting-and-concentration.md` FR-082; SRD 5.2.1, "Using a Higher-Level Spell Slot", pp. 104-105 | happy |
| FR-008 | A monster cast MUST expend no spell slot and MUST NOT be limited by "One Spell with a Spell Slot per Turn": two casts in one turn are accepted. Interpretation: that rule limits only the expenditure of a slot. An action that casts twice (Pit Fiend "Hellfire Spellcasting" casts Fireball twice in one action, p. 315) is two `cast` calls by the host; no per-action cast count is enforced here (interpretation: the part text states the count, p. 257 leaves it to the stat block). | SRD 5.2.1, "One Spell with a Spell Slot per Turn", p. 105; "Spellcasting", p. 257 | unhappy (rule interaction) |

### 4.2 Printed numbers

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-009 | The report's `saveDC` MUST be the trait's printed spell save DC, unchanged. It MUST NOT be recomputed as 8 + modifier + Proficiency Bonus, even when the formula would differ (table PN-1). | SRD 5.2.1, "Spellcasting" ("provides the monster's ... spell save DC"), p. 257 | happy |
| FR-010 | The report's `attackModifiers` MUST be a `RollModifiers` whose single ability slot holds the printed spell attack bonus as one `Modifier`, with no Proficiency Bonus and no others, so its sum is the printed bonus. | SRD 5.2.1, "Spellcasting", p. 257; `spellcasting.md` FR-116 | happy |
| FR-011 | The report MUST carry the spellcasting `Ability` and the Modifier of the monster's score in it (for a spell that adds "your spellcasting ability modifier"). | SRD 5.2.1, "Spellcasting", p. 257 | happy |
| FR-012 | When the spell's `SpellEffectFacts` (`spell-effect-data.md` FR-022) say `forcesSavingThrow` is `FactKnown(true)` and no DC is printed, the cast MUST be refused with `missingSaveDC`; likewise `makesSpellAttackRoll` with no printed bonus gives `missingSpellAttackBonus`. A `FactUnknown`, a false fact and a text-only spell MUST NOT be refused (the GM decides, p. 106). A trait with no printed numbers casting a spell that needs neither (Priest Acolyte: Light, Thaumaturgy) MUST succeed with both absent. | SRD 5.2.1, "Spellcasting" ("if any spells require a saving throw"), p. 257; "Priest Acolyte", p. 316 | unhappy (missing content) |

### 4.3 Restrictions and components

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | `MonsterSpellRestrictions.Of(note)` MUST set `selfOnly` when the entry's note contains "self only" (case-insensitive, whitespace collapsed, matched at word boundaries) and MUST set `waivesConcentration` when it contains "no Concentration" (same matching); other note text ("included in AC", "cast before combat", "24-hour duration", "Beast or Humanoid form only") sets nothing and stays in the entry. An empty note sets nothing. Table RS-1. | SRD 5.2.1, "Spellcasting" ("self only"), p. 257; "Archmage", p. 305; "Adult Brass Dragon", p. 269 | happy |
| FR-014 | A `selfOnly` spell MUST be refused with `selfOnlyViolated` when the request's `targets` (the creatures the host says it affects) contain any `CombatantID` other than the caster's; an empty list and the caster alone MUST be accepted. | SRD 5.2.1, "Spellcasting" ("the spell affects only the hag"), p. 257 | unhappy (rule interaction) |
| FR-015 | A spell without `selfOnly` MUST NOT be limited in its targets by this spec. The report carries `targetLimit` (`selfOnly` or `unrestricted`) so the host passes only the caster to `spell-effects.md`. | SRD 5.2.1, "Spellcasting", p. 257; `spell-effects.md` FR-019 | unhappy (rule interaction) |
| FR-016 | `waivesConcentration` MUST mean the spell's own Concentration is not begun and (interpretation, Q-8) no Concentration in progress is ended by this cast, because a spell whose Concentration is not required cannot be "another Concentration effect" (p. 179); the cast still succeeds and the spell's other duration rules are the effects spec's. It does not waive the Concentration a 1+ minute casting needs (FR-023). Shapechange of the Adult Brass Dragon is the SRD case; its Temporary Hit Points clause stays in the `RestrictionNote` for `shape-changing-spells.md`. | SRD 5.2.1, "Adult Brass Dragon" ("no Concentration or Temporary Hit Points required to maintain the spell"), p. 269; "Concentration", p. 179 | unhappy (rule interaction) |
| FR-017 | `ComponentWaiver.Parse(statement)` MUST normalise (trim, lower-case, collapse whitespace, strip one trailing period) and then require the grammar `requiring no ` followed by `spell components` (V, S and M) or a list of `verbal`, `somatic`, `material` joined by `, `, ` or `, ` and ` or `, or ` and followed by ` components` (each waives its kind; a repeated kind is unrecognised). An empty statement waives nothing. Any other text, including a missing `requiring ` prefix or a trailing clause (the caller passes only the components clause; an ability or DC goes in `PartSpellcasting`, FR-039), MUST be `ComponentStatementUnrecognized`, and a cast under it MUST be refused with `componentsStatementUnrecognized`: never guessed. Table CS-1. | SRD 5.2.1, "Spellcasting" (Spell Components), p. 257; "Deva", p. 279 (Material), p. 296 (Somatic or Material), p. 278 (spell components); "Cloaker" and "Pit Fiend" as `PartCast` statements, pp. 274, 315 | unhappy (invalid input) |
| FR-018 | For a spell with a Verbal component not waived, `MonsterCastContext.CanSpeak` false MUST refuse with `verbalComponentMissing`; for Somatic not waived, `canMakeGestures` false MUST refuse with `somaticComponentMissing`. A waived component MUST NOT be checked (a silenced monster whose statement waives Verbal casts). | SRD 5.2.1, "Verbal (V)", "Somatic (S)", pp. 105-106; "Spellcasting", p. 257 | unhappy (state error) |
| FR-019 | A Material component MUST NEVER refuse a monster cast, waived or not ("A monster that requires Material components has them"), including one with a gp cost, and the cast needs no free hand. Interpretation: the sentence supplies the materials; the hand rule (p. 106) is for a spellcaster who must reach them. | SRD 5.2.1, "Spellcasting" (Spell Components), p. 257; "Material (M)", p. 106 | unhappy (rule interaction) |

### 4.4 Actions and uses

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | `ConditionActionAvailability.None` in the context MUST refuse any cast with `cannotTakeActions`. A `ListCast` is the Spellcasting action, a `PartCast` the Bonus Action or Reaction of its part: both need the monster to be able to act. | SRD 5.2.1, "Incapacitated", p. 184 | unhappy (state error) |
| FR-021 | Uses MUST be counted through `monster-limited-use.md` in the same call. A `ListCast` of a `N/Day` or `N/Day Each` spell MUST spend its entry with `SpendSpell`; an At Will spell spends nothing (`notLimitedUse` is accepted). A `PartCast(part, castAtLevel?, componentsStatement?)` with the caller's `Spell` MUST spend `Spend(part)`; an unlimited part spends nothing. A spent-out entry MUST refuse with `usesExhausted`. Every refusal MUST be decided before anything is spent, and the returned `LimitedUseState` is the input unchanged on any refusal, the spent state on success. Uses are per spell (`monster-limited-use.md` FR-046). A part cast takes its ability, DC, bonus and components from the part (`PartSpellcasting`, FR-039), falling back to the trait only for "the same spellcasting ability as Spellcasting"; the trait's components statement never applies to it. A part cast is immediate whatever the spell's printed casting time (the part states the monster casts it). | SRD 5.2.1, "Limited Usage", p. 257; "Mage" (Misty Step 3/Day), p. 305; "Deva", p. 279 | happy |

### 4.5 Casting times of 1+ minutes

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | A `ListCast` of a spell whose chosen casting time (`CastingTimeMinutes`, `CastingTimeHours`; a spell with modes needs the caller's `CastingTimeMode`, as `casting-and-concentration.md` FR-115) is 1 minute or more MUST NOT complete: it MUST return `MonsterCastStarted` with a `CastingInProgress` (`totalMinutes`, `casting-and-concentration.md` FR-131; no Ritual minutes), spend no use and emit `MonsterCastingStarted`. A Magic action, Bonus Action or Reaction casting time is cast at once: the Spellcasting action replaces it. Table LC-1. | SRD 5.2.1, "Spellcasting" (Casting Times of 1+ Minutes), p. 257; "Longer Casting Times", p. 105; "Geas", p. 135 | happy |
| FR-023 | Starting a longer casting MUST end Concentration in progress (`MonsterConcentrationEnded`, reason `anotherEffect`) and begin Concentration of kind `casting` on the spell (`MonsterConcentrationBegan`). | SRD 5.2.1, "Spellcasting", p. 257; "Concentration", p. 179; `casting-and-concentration.md` FR-101 | unhappy (rule interaction) |
| FR-024 | While a casting is in progress any further cast MUST be refused with `castingAlreadyInProgress`, cantrips included. | `casting-and-concentration.md` FR-102; SRD 5.2.1, "Longer Casting Times", p. 105 | unhappy (state error) |
| FR-025 | `endMagicActionTurn(tookMagicAction)` MUST, with a casting in progress and `tookMagicAction` false, fail the casting: Concentration ends (reason `ended`), the casting is cleared, no use is spent, and `MonsterCastingFailed` is emitted. With `tookMagicAction` true, or no casting in progress, the state MUST be returned equal with no event. The starting cast is itself the Magic action of its turn. Interpretation: "must take the Magic action on each of its turns" means a turn without it ends the casting. | SRD 5.2.1, "Spellcasting" (Casting Times of 1+ Minutes), p. 257; "Longer Casting Times", p. 105 | unhappy (state error) |
| FR-026 | `completeCasting` MUST, for the spell of the casting in progress (a different `Spell` MUST return an error matching `srderr.ErrInvalidInput`; a request that is not a `ListCast` of the stored ref is refused, FR-040), re-run the refusals of FR-012, FR-014, FR-017, FR-018, FR-020 and the use check, spend the use once, clear the casting and emit `MonsterSpellCast`. A refusal leaves the casting in progress and both states unchanged. The host calls it when the casting time has passed (`casting-and-concentration.md` FR-071 does the same for characters). With none in progress it MUST be refused with `castingNotInProgress`. The use is spent at completion because a broken Concentration means the spell fails (p. 105); this extends `casting-and-concentration.md` FR-070 to a use (interpretation). | SRD 5.2.1, "Longer Casting Times", p. 105; "Limited Usage", p. 257 | happy |
| FR-027 | At completion a spell that requires Concentration and is not `waivesConcentration` MUST continue as Concentration of kind `effect` with no new event; any other spell MUST end the casting Concentration with `MonsterConcentrationEnded`, reason `castingCompleted`. | SRD 5.2.1, "Concentration", p. 179; `casting-and-concentration.md` FR-138 | unhappy (rule interaction) |
| FR-028 | `breakConcentration(reason)` MUST accept the `ConcentrationBreakReason`s (`incapacitated`, `dead`, `ended`, `failedSave`, `anotherEffect`), end Concentration (`MonsterConcentrationEnded`), and, with a casting in progress, clear it, spend no use and emit `MonsterCastingFailed`. With no Concentration it MUST return an equal state and no events. The save after damage is the host's (`ConcentrationSaveDC.ForDamage`). | SRD 5.2.1, "Concentration", p. 179; "Longer Casting Times", p. 105; `casting-and-concentration.md` FR-099, FR-119, FR-128 | unhappy (state error) |

### 4.6 Immediate casts and Concentration spells

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-029 | An immediate cast MUST return `MonsterCastPerformed` with the report, the new states and `MonsterSpellCast`. A spell that requires Concentration and is not `waivesConcentration` MUST end Concentration in progress (`anotherEffect`) and begin Concentration of kind `effect`, events in the order ended, began, cast. A cast of such a spell during a casting in progress is refused by FR-024. | SRD 5.2.1, "Concentration" (activating another effect), p. 179; `casting-and-concentration.md` FR-075, FR-083 | happy |

### 4.7 Hand-off, values and placement

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | `MonsterCastReport` MUST hold the spell id, `castAtLevel?`, `scalingLevel`, `saveDC?`, `attackModifiers?`, the `Ability` and its Modifier, `targetLimit` and the `RestrictionNote` text unchanged. The Temporary Hit Points waiver in a note is carried in that text and is the shape-changing spec's. `toCastReport()` MUST return the `CastReport` of `casting-and-concentration.md` FR-082 with no casting class, no slot and source `monster`, so a host passes it to `ResolveSpellEffects` unchanged (amendment A1). | `spell-effects.md` section 6; package design | happy |
| FR-031 | The refusal reasons, in precedence order (the first is returned), MUST be: `notInSpellList`, `castingRefMismatch` (`completeCasting` only), `castingAlreadyInProgress`, `cannotTakeActions`, `componentsStatementUnrecognized`, `verbalComponentMissing`, `somaticComponentMissing`, `selfOnlyViolated`, `missingSaveDC`, `missingSpellAttackBonus`, `usesExhausted`. A refusal MUST leave every state equal and emit nothing. Package design, so the result is independent of check order. | package design; SRD 5.2.1, p. 257 | unhappy (rule interaction) |
| FR-032 | `NewMonsterSpellcastingState(combatant, opts...)` (options for the optional `concentration`, `castingInProgress`) MUST return an error matching `srderr.ErrInvalidInput` when a casting is present without Concentration of kind `casting` on the same spell or without its `castingRef` (FR-040), or when its `combatant` differs from the caster's in a call. | architecture.md §3, §6 | unhappy (invalid input) |
| FR-033 | Every operation MUST leave its inputs unchanged and return new values; event lists and the report MUST be copies (mutating a returned slice changes nothing); all value objects MUST be comparable by value (`==`, or an `Equal` method where a field is a slice). | architecture.md §3, §4; `combat-foundations.md` FR-021 | unhappy (immutability) |
| FR-034 | The events `MonsterSpellCast`, `MonsterCastingStarted`, `MonsterConcentrationBegan`, `MonsterConcentrationEnded` and `MonsterCastingFailed` MUST be the members of one closed `MonsterSpellcastingEvent` implementing `CombatEvent`, with constant unique kinds (`monster_spell_cast`, `monster_casting_started`, `monster_concentration_began`, `monster_concentration_ended`, `monster_casting_failed`) and `combatants` equal to `[combatant]`. They carry the spell or `ConcentrationSubject`, not text. | `combat-foundations.md` FR-015 to FR-020 | happy |
| FR-035 | Operations MUST NOT return an error for a refusal. Only FR-001, FR-002, FR-004, FR-026 (wrong spell), FR-032, FR-039 (a part without own values on a caster with no trait) and an invalid `CastingTimeMode` (`casting-and-concentration.md` FR-115) returns an error matching `srderr.ErrInvalidInput`. Results: `MonsterCastOutcome` (closed: `MonsterCastPerformed`, `MonsterCastStarted`, `MonsterCastRefused(MonsterCastRefusal)`), `MonsterSpellcastingTransition` (states, events) for the others. | architecture.md §6; shared-types-and-ownership.md FR-017 | unhappy (invalid input) |
| FR-036 | The code MUST be synchronous and deterministic (no `math/rand`, `time.Now`, goroutines, `context`, I/O), live in the `monster_spellcasting*.go` files of package `combat`, contain no `telemetry.Logger`, `Tracer`, `Meter` or signal constant, and import only the packages `content` and `character`, the kernel packages, `d20`, and `combat` files of lower rank (checked by `internal/archtest`). No use case, port or double exists. | architecture.md §1, §3, §9; observability.md §3; shared-types-and-ownership.md FR-044 | happy |
| FR-037 | Every type of sections 5 and 6 MUST be exported from package `combat` with a godoc each; no name MAY duplicate another exported name or be generic (`Refused`, `Applied`). | architecture.md §4, §7; `combat-foundations.md` FR-024 | happy |
| FR-038 | Nothing in the code, tests or data MAY name a consuming project or use content outside SRD 5.2.1. Test inputs that quote the SRD live under `combat/testdata/srd/`. | AGENTS.Md ("Hard constraints"); shared-types-and-ownership.md FR-033 | happy |
| FR-039 | A `PartCast` MUST take its ability, save DC, spell attack bonus and components from a caller-supplied `PartSpellcasting(ability, saveDC?, attackBonus?, componentsStatement?)` when the part prints any of them (Pit Fiend "Hellfire Spellcasting": Charisma, DC 21, "requiring no Material components"; Cloaker "Phantasms": Wisdom, "requiring no spell components"), using the monster's score in that ability for the Modifier (FR-011). Absent `PartSpellcasting`, it MUST use the trait ("the same spellcasting ability as Spellcasting", Planetar p. 315) and returns an error matching `srderr.ErrInvalidInput` on a caster with no trait. FR-012 and FR-018 apply to the part's values. Casting level is the part's stated version (FR-003, FR-005). | SRD 5.2.1, "Cloaker", p. 274; "Pit Fiend", p. 315; "Spellcasting", p. 257 | unhappy (missing content) |
| FR-040 | `MonsterSpellcastingState` MUST store `castingRef` (the `SpellUseRef` that started the casting). `completeCasting` MUST take only a `ListCast` and MUST refuse with `castingRefMismatch` any other request or a different ref, so the use is spent on the entry that began the casting, never another group's entry for the same spell. | package design; SRD 5.2.1, "Limited Usage", p. 257 | unhappy (state error) |

### Cases CL-1: Cast level (FR-003 to FR-007)

| Label | Entry | Expected | Source | Path |
|---|---|---|---|---|
| lowest level | Priest `1/Day: Spirit Guardians` (level 3), no version | `castAtLevel` 3, `scalingLevel` 3 | "Priest", p. 316 | happy |
| stated version | Archmage `Lightning Bolt (level 7 version)` | 7 | "Archmage", p. 305 | happy |
| version equal to level | synthetic `Fireball (level 3 version)` | 3 | FR-003 | unhappy (boundary) |
| version below level | synthetic `Fireball (level 2 version)` | `srderr.ErrInvalidInput` error | FR-004 | unhappy (boundary) |
| version on a cantrip | synthetic `Light (level 1 version)` | `srderr.ErrInvalidInput` error | FR-004 | unhappy (invalid input) |
| cantrip | `At Will: Light` | `castAtLevel` absent, `scalingLevel` 1 | FR-006 | unhappy (boundary) |
| top version | Archmage `Cone of Cold (level 9 version)` | 9 | p. 305 | unhappy (boundary) |
| two casts, one turn | synthetic caster casts twice in one turn | both performed, no slot reported | FR-008 | unhappy (rule interaction) |

### Cases PN-1: Printed numbers (FR-009 to FR-012)

| Label | Trait | Expected | Source | Path |
|---|---|---|---|---|
| DC as printed | synthetic Int +3, PB +2, printed DC 14 (formula gives 13) | `saveDC` 14 | FR-009 | unhappy (rule interaction) |
| archmage DC | Archmage, printed 17 | 17 | p. 305 | happy |
| attack bonus | trait printing "+10 to hit with spell attacks" | sum 10, no PB slot | p. 270 | happy |
| no numbers, no need | Priest Acolyte, Light | success, both absent | p. 316 | unhappy (empty and minimal) |
| DC absent, save spell | same trait, Fireball | `missingSaveDC` | FR-012 | unhappy (missing content) |
| bonus absent, attack spell | trait with DC only, spell attack spell | `missingSpellAttackBonus` | FR-012 | unhappy (missing content) |
| text-only spell | DC absent, `FactUnknown` | success | FR-012 | unhappy (boundary) |

### Cases RS-1 and CS-1: Notes and component statements (FR-013 to FR-019)

| Label | Input | Expected | Source | Path |
|---|---|---|---|---|
| self only (Plane Shift) | note `self only` | `selfOnly` | p. 311 | happy |
| self only with tail | `self only, and the hag leaves no tracks while Invisible` | `selfOnly` | "Green Hag", p. 295 | happy |
| case | `Self Only` | `selfOnly` | FR-013 | unhappy (boundary) |
| informational | `included in AC`, `cast before combat` | no flag | p. 305 | unhappy (rule interaction) |
| shapechange | `Beast or Humanoid form only, ..., and no Concentration or Temporary Hit Points required` | `waivesConcentration`, not `selfOnly` | "Adult Brass Dragon", p. 269 | unhappy (rule interaction) |
| empty note | "" | no flag | FR-013 | unhappy (empty and minimal) |
| other creature | self-only spell, targets `{caster, other}` | `selfOnlyViolated` | FR-014 | unhappy (rule interaction) |
| statement material | `requiring no Material components` | M waived | p. 279 | happy |
| statement S and M | `requiring no Somatic or Material components` | S, M | p. 296 | happy |
| statement all | `requiring no spell components` | V, S, M | p. 278 | happy |
| no statement | "" | none waived | FR-017 | unhappy (empty and minimal) |
| unknown statement | `requiring no Verbal components` | unrecognized; cast refused | FR-017 | unhappy (invalid input) |
| verbal or somatic | `requiring no Verbal or Somatic components` | V, S | FR-017 | happy |
| spacing, period | `  Requiring  no Material components.  ` | M | FR-017 | unhappy (boundary) |
| trailing clause | `requiring no spell components and using Wisdom` | unrecognized | FR-017 | unhappy (invalid input) |
| repeated kind | `requiring no Material or Material components` | unrecognized | FR-017 | unhappy (boundary) |
| word boundary | note `selfonly`, `no Concentrations` | no flag | FR-013 | unhappy (boundary) |
| cloaker part | no trait, `PartSpellcasting(Wisdom, statement all)`, Mirror Image | performed, Wisdom Modifier, no DC | "Cloaker", p. 274 | happy |
| hellfire part | `PartSpellcasting(Charisma, DC 21, no Material)`, Fireball level 5, silenced | performed, `saveDC` 21, Verbal checked | "Pit Fiend", p. 315 | happy |
| part, no trait, no own values | caster from `forParts`, `PartCast` without `PartSpellcasting` | `srderr.ErrInvalidInput` error | FR-039 | unhappy (invalid input) |
| silenced, V waived | `no spell components`, `canSpeak` false | success | FR-018 | unhappy (state error) |
| silenced, V needed | no statement, `canSpeak` false | `verbalComponentMissing` | FR-018 | unhappy (state error) |
| no hands, S needed | no statement, `canMakeGestures` false | `somaticComponentMissing` | FR-018 | unhappy (state error) |
| costly material | no statement, M with cost | success | FR-019 | unhappy (boundary) |

### Cases LC-1: Longer casting (FR-022 to FR-029)

| Label | State and input | Expected | Source | Path |
|---|---|---|---|---|
| geas starts | Geas (1 minute), none in progress | `MonsterCastStarted`, 1 minute, no use spent, Concentration `casting` | p. 135 | happy |
| scrying starts | Scrying (10 minutes), `1/Day Each` | 10 minutes, use unspent | p. 160 | happy |
| action spell | Mind Blank (Magic action) | `MonsterCastPerformed` | FR-022 | unhappy (boundary) |
| replaces Concentration | holding `effect`, start Geas | ended `anotherEffect` then began `casting` | FR-023 | unhappy (rule interaction) |
| second cast | casting in progress, cast cantrip | `castingAlreadyInProgress` | FR-024 | unhappy (state error) |
| action each turn | in progress, `tookMagicAction` true | equal state | FR-025 | happy |
| turn skipped | in progress, false | failed, no use spent | FR-025 | unhappy (state error) |
| no casting | none, false | equal, no event | FR-025 | unhappy (empty and minimal) |
| complete | in progress, completes | use spent once, `MonsterSpellCast` | FR-026 | happy |
| complete, none | none in progress | `castingNotInProgress` | FR-026 | unhappy (state error) |
| complete, wrong spell | other `Spell` | `srderr.ErrInvalidInput` error | FR-026 | unhappy (invalid input) |
| complete, exhausted | use at 0 at completion | `usesExhausted`, still in progress | FR-026 | unhappy (boundary) |
| complete, other ref | Scrying begun from group A, complete with group B entry or a `PartCast` | `castingRefMismatch`, still in progress | FR-040 | unhappy (state error) |
| bad mode | spell with modes, no or invalid `CastingTimeMode` | `srderr.ErrInvalidInput` error | FR-035 | unhappy (invalid input) |
| damage breaks | in progress, `failedSave` | failed, no use spent | FR-028 | unhappy (state error) |
| incapacitated | in progress, `incapacitated` | failed | FR-028 | unhappy (state error) |
| Concentration spell done | completes, requires Concentration | continues as `effect` | FR-027 | happy |
| waived done | completes, `waivesConcentration` | ends `castingCompleted` | FR-027 | unhappy (rule interaction) |
| immediate Concentration | Magic action spell, requires it, holding other | ended, began, cast | FR-029 | happy |
| waived immediate | Shapechange with waiver, holding other | held Concentration kept | FR-016 | unhappy (rule interaction) |
| incapacitated cast | availability `none` | `cannotTakeActions` | FR-020 | unhappy (state error) |
| part cast | Misty Step (3/Day), 3 left | performed, 2 left | p. 305 | happy |
| exhausted | Dominate-style `1/Day` at 0 | `usesExhausted`, states equal | FR-021 | unhappy (boundary) |
| at will | At Will spell | performed, `LimitedUseState` equal | FR-021 | unhappy (empty and minimal) |
| precedence | out of list and incapacitated | `notInSpellList` | FR-031 | unhappy (rule interaction) |

## 5. Model

- **Value objects:** `MonsterSpellcaster` (`combatant`, `monsterID`, `spellcasting?`, `abilityModifiers`); `MonsterSpellcastingState` (`combatant`, `concentration?`, `castingInProgress?`, `castingRef?`; reused `Concentration`, `CastingInProgress`); closed `MonsterCastRequest` (interface with an unexported marker method) (`ListCast(SpellUseRef, castingTimeMode?)`, `PartCast(LimitedUsePartRef, castAtLevel?, PartSpellcasting?)`), `PartSpellcasting(ability, saveDC?, attackBonus?, componentsStatement?)` with the caller's `Spell`, `targets` and `MonsterCastContext(canSpeak, canMakeGestures, actions)` (`ConditionActionAvailability`); `MonsterSpellRestrictions(selfOnly, waivesConcentration)`; `ComponentWaiver` (closed: `ComponentsWaived(kinds)` (a unique set of `ComponentKind`), `ComponentStatementUnrecognized(text)`); `MonsterCastReport`; `TargetLimit` (`selfOnly`, `unrestricted`).
- **Results:** `MonsterCastOutcome` (`MonsterCastPerformed(report, transition)`, `MonsterCastStarted(transition)`, `MonsterCastRefused(refusal)`); `MonsterCastRefusal` (enum of FR-031 plus `castingNotInProgress`); `MonsterSpellcastingTransition(state, limitedUse, events, limitedUseEvents)`.
- **Events:** closed `MonsterSpellcastingEvent` (FR-034). Limited-use events are the `MonsterLimitedUseEvent`s of `monster-limited-use.md`, kept in their own list.
- **Domain service:** `MonsterSpellcastingRules` (stateless, zero-size struct): `Cast`, `CompleteCasting`, `EndMagicActionTurn`, `BreakConcentration`.
- **Invariants:** FR-032 in the state constructor; FR-004 in the request constructors; `Spell.Name` match (FR-002) in `cast`.
- **Failures:** results for rule outcomes; an error matching `srderr.ErrInvalidInput` for caller defects (constructors and `Cast` return `(T, error)`; `MustX` helpers panic and are for literals only); no `*srderr.PortError` (no port).

## 6. Interface

```go
// Package combat, files monster_spellcasting*.go (indicative). Pure: no ctx, no ports.
type MonsterSpellcastingRules struct{} // stateless; the zero value is usable

func (MonsterSpellcastingRules) Cast(caster MonsterSpellcaster, state MonsterSpellcastingState,
	uses LimitedUseState, req MonsterCastRequest, spell content.Spell,
	targets []CombatantID, mc MonsterCastContext) (MonsterCastOutcome, error) // FR-001..FR-031
func (MonsterSpellcastingRules) CompleteCasting(caster MonsterSpellcaster, state MonsterSpellcastingState,
	uses LimitedUseState, req MonsterCastRequest, // ListCast of the stored ref (FR-040)
	spell content.Spell, targets []CombatantID, mc MonsterCastContext) (MonsterCastOutcome, error) // FR-026, FR-027
func (MonsterSpellcastingRules) EndMagicActionTurn(state MonsterSpellcastingState,
	uses LimitedUseState, tookMagicAction bool) MonsterSpellcastingTransition // FR-025
func (MonsterSpellcastingRules) BreakConcentration(state MonsterSpellcastingState,
	uses LimitedUseState, reason ConcentrationBreakReason) MonsterSpellcastingTransition // FR-028

func NewMonsterSpellcaster(m content.Monster, combatant CombatantID) (MonsterSpellcaster, error)         // FR-001
func MonsterSpellcasterForParts(m content.Monster, combatant CombatantID) (MonsterSpellcaster, error)    // FR-001, FR-039
func NewMonsterSpellcastingState(combatant CombatantID, opts ...StateOption) (MonsterSpellcastingState, error) // FR-032
func (r MonsterCastReport) ToCastReport() CastReport                                                     // FR-030
```

The `error` results are non-nil only for caller defects (untrusted or inconsistent input, matching `srderr.ErrInvalidInput`); every rule refusal is a `MonsterCastRefused` value. `MonsterCastOutcome` is a closed variant set (interface with an unexported marker method; variants `MonsterCastPerformed`, `MonsterCastStarted`, `MonsterCastRefused`). A trailing `?` in this spec marks an optional value; in Go it is a value type with an `IsZero` check (for example `Concentration` and `CastingInProgress` are value structs whose zero value means "none"), never a nil pointer to a small struct. A target set is a `[]CombatantID` with unique members; a duplicate is dropped.

- **Driving it:** the host stores a `MonsterSpellcastingState` per monster beside its `LimitedUseState` (as `monster-limited-use.md` Q-5). To cast, it builds the request and calls `Cast`; on `MonsterCastPerformed` it passes `report.ToCastReport()`, `report.AttackModifiers()`, `report.AbilityModifier()` and, for `targetLimit` `selfOnly`, only the caster to `ResolveSpellEffects`. After damage to a concentrating monster it rolls the Constitution save (`ConcentrationSaveDC.ForDamage`) and on failure calls `BreakConcentration(failedSave)`.
- **Exports:** all types above from package `combat` (no facade or re-export package). `dice/dicetest` and `telemetry/telemetrytest` gain nothing.

## 7. Observability

Silent by design (`doc/rules/observability.md` §3): pure domain, no use case and no port, so there is nothing to time, count or fail. Evaluation (§2): (1) failure mode: a wrongly recognised note or statement (a cast allowed or refused wrongly); (2) it shows in the returned refusal and events; (3) no attributes; (4) nothing is counted, the events are the record; (5) `monster-limited-use.md`'s signals are unchanged; (6) no logs; (7) no cardinality. Any span or counter is the host's, around `Cast`; a future use case would follow `combat-foundations.md` FR-027 to FR-041. No rows are added to `doc/observability/signals.md`.

## 8. Test plan

Tests are in package `combat_test` (`combat/monster_spellcasting_*_test.go`). Unit tests use synthetic and SRD-shaped stat blocks, no ports. Each case-table row is one table-driven test.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-008 | unit | `cast_level_test.go`: CL-1; `NewMonsterSpellcaster` (success and error); no level parameter on the request types (reflection-free: the type surface asserted by a compile-time fixture); two casts in a turn |
| FR-009 to FR-012 | unit | `printed_numbers_test.go`: PN-1; `RollModifiers` shape |
| FR-013 to FR-019, FR-039 | unit | `restrictions_test.go`, `components_test.go`: RS-1, CS-1 |
| FR-020, FR-021, FR-031 | unit | `uses_test.go`: spend, exhausted, At Will, part cast; refusal leaves both states equal; precedence rows |
| FR-022 to FR-029 | unit | `longer_casting_test.go`: LC-1, event order, invariants of the state |
| FR-030 | unit | `report_test.go`: `ToCastReport` fields, absent class and slot |
| FR-032 to FR-035, FR-040 | unit | `state_events_test.go`: constructor errors, equality, copied slices, exhaustive type switch over events, kinds unique, no error or panic for refusals |
| FR-036 to FR-038 | architecture | `internal/archtest`: file purity, import allow-list, no signal code, exported-name scan (exports, godoc, no generic names), consumer-name scan |

- **Doubles and suites:** none new.
- **Mutation testing:** the `monster_spellcasting*.go` files of package `combat`, with gomu run only inside Docker with `--cpus` and `--workers` capped at 1/4 of `nproc` (testing.md §8; never run locally), watching the precedence order of FR-031, `castAtLevel` falling back to `level` in FR-003, the three statement forms of FR-017 (waiver set per form), `selfOnly` against `waivesConcentration` in FR-013, and "spend at completion, not at start" in FR-022 and FR-026.
- **Not covered:** nothing.

## 9. Open questions and decisions

Settled, one line each:

- Q-1: cantrips do not scale (FR-006); SRD silent, "lowest possible level" read as base; web research agrees; no SRD monster casts a damaging cantrip.
- Q-2: Material components never refuse a monster (FR-019); p. 257 says the monster has them.
- Q-3: a longer casting spends its use at completion (FR-026), by analogy to the slot decision in `casting-and-concentration.md` FR-070; interpretation.
- Q-4: a turn without the Magic action ends a longer casting (FR-025); interpretation of "must take the Magic action on each of its turns".
- Q-5: the notes and the component grammar are recognised here (FR-013, FR-017) because the content layer keeps verbatim text; structured fields are amendment A4. Unknown text is refused, never guessed.
- Q-6: a part cast is immediate (FR-021); p. 257 "unless the action's description states otherwise". A stat-block part that prints its own ability, DC or components is cast through `PartSpellcasting` (FR-039), not the trait.
- Q-7: a known "forces a save" fact with no printed DC refuses (FR-012); unknown facts do not.
- Q-8: a no-Concentration spell (Shapechange, p. 269) leaves held Concentration alone (FR-016); interpretation, p. 179 "another Concentration effect" needs Concentration to be required.

Open: none. Escalated: none.

Amendments requested (other specs are not edited here): A1 `casting-and-concentration.md` FR-082: `CastReport` allows no casting class, no slot, and absent DC and bonus (source `monster`); `spell-effects.md` FR-050 treats an absent value a part needs as `srderr.ErrInvalidInput` error. A2 withdrawn: the rank 27 row is already in `shared-types-and-ownership.md` 4.11. A3 `casting-and-concentration.md` section 1 and `spell-effects.md` section 1: the "monster Spellcasting traits" exclusion cites this spec; `monster-limited-use.md` section 1 likewise for "which spell a monster casts". A4 `content-monsters.md` FR-046 (optional): structured `restrictions` and `componentWaiver` fields; this spec's parsers then become the decoder's. A5 index, `rules-glossary-coverage.md` and `roadmap.md`: add the spec.

## 10. Changelog impact

`Added` (combat): `MonsterSpellcastingRules` (casting from a Spellcasting trait: lowest level or stated version, printed DC and bonus, "self only", component statements, longer casting times, per-day uses via `LimitedUseState`), `MonsterSpellcastingState`, `MonsterCastReport` and the `MonsterSpellcastingEvent` family. Not **BREAKING**: nothing is released.
