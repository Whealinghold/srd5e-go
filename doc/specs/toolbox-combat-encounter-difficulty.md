# Combat Encounter Difficulty (XP Budget)

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `content-monsters.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `errors-and-results.md`, `observability-ports.md`, `toolbox-hazard-effects.md` |
| Supersedes (shared-types-and-ownership.md FR-048) | Nothing: no older spec defines an XP budget or encounter difficulty; the content-monsters.md lines that name the encounter budget are out-of-scope pointers, not requirements. |

Context: `toolbox` (shared-types-and-ownership.md FR-045, type phase FR-014) is downstream of `content`, `d20`, `character` and `combat`; this spec uses only the `content` edge (FR-028). Not a dependency: `encounter-aggregate.md`, `character-aggregate.md`, `derived-stats.md` (Q-2).

## 1. Purpose and scope

Specifies the pre-fight calculation of "Combat Encounter Difficulty" (SRD 5.2.1,
pp. 202-203): choose a difficulty, look up the XP Budget per Character for the
party's level, multiply by the party size, deduct the XP of each creature, and
report whether the plan fits and how much XP is unspent, plus optional
**advisories** mirroring the Troubleshooting guidance (p. 203). It is a pure
calculation over an integer party level, a party size and the printed XP and
Challenge Rating of `content` monsters. It plans a combat, does not run one,
shares no state with the `combat` `Encounter` aggregate and imports nothing from
`combat`; the use case calls no other use case (shared-types-and-ownership.md FR-044).

**Out of scope:**

- Choosing creatures (the model reports spent and unspent XP, no best-fit search).
- Qualitative guidance (p. 202: "Changes in Elevation", "Defensive Positions",
  "Mixed Monster Groups", "Reasons to Move"; the prose definitions of the
  difficulties; the single-monster guideline), "Adjustments" and "Unusual
  Features" (p. 203): advice, not computable rules.
- Awarding XP and XP "in lair" (content-monsters.md FR-032; Q-4).
- Non-monster XP sources: a custom creature goes straight to the domain service
  with a host-chosen valid `MonsterID` and host-supplied rating and XP, validated
  by `NewPlannedCreature` and `NewXPBudget` (FR-044, FR-045).
- Mixed-level parties: the SRD gives no averaging rule (Q-2).

## 2. Source

- SRD 5.2.1, "Combat Encounter Difficulty", p. 202 (Steps 1 to 3, the XP Budget
  per Character table, Examples 1 and 2) and Example 3, p. 203; "Troubleshooting"
  ("Many Creatures", "CR 0 Creatures", "Number of Stat Blocks", "Powerful
  Creatures"), p. 203; "Experience Points by Challenge Rating", p. 256.
- Stat blocks of the examples ("Monsters A-Z", "Animals", pp. 258-364): Adult Red
  Dragon (CR 17, XP 18,000), Fire Giant (CR 9, 5,000), Druid (CR 2, 450), Stirge
  (CR 1/8, 25), Wight (CR 3, 700), Warhorse Skeleton (CR 1/2, 100), Skeleton
  (CR 1/4, 50), Giant Wasp (CR 1/2, 100), Giant Rat (CR 1/8, 25), Bugbear Warrior
  (CR 1, 200).
- Package design: `doc/rules/architecture.md`, `go.md`, `observability.md`, `testing.md`.

Page numbers are printed numbers; the PDF page index matches them.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Low, Moderate, High Difficulty | `EncounterDifficulty` (`low`, `moderate`, `high`) | Step 1, p. 202. |
| Party's Level | `int`, `EncounterParty.Level`, 1 to 20 | p. 202. Not `CharacterLevel` (Q-2). |
| number of characters | `int`, `EncounterParty.Size`, at least 1 | Step 2, p. 202. |
| XP Budget per Character | `XPBudgetTable.PerCharacter(level, difficulty)` | Package-level read-only table behind a function, p. 202 (shared-types-and-ownership.md FR-045; Q-5). |
| XP budget | `XPBudget` (`int` XP, at least 0; Q-13) | Step 2, p. 202. |
| XP of a creature | `PlannedCreature.ExperiencePoints` | Printed value (content-monsters.md FR-032), p. 202. |
| Challenge Rating | `ChallengeRating` (content-monsters.md FR-030, FR-097) | Reused; members needed are in FR-031 (Q-3). |
| (a creature in the plan) | `PlannedCreature` | `content.MonsterID` (content-schema-and-decoder.md FR-114), `ChallengeRating`, XP, count. |
| spend / unspent XP | `CombatEncounterPlan.SpentXP`; `EncounterWithinBudget.UnspentXP`; `EncounterOverBudget.ExcessXP` | Step 3, p. 202. |
| (inputs and plan; not SRD nouns) | `EncounterParty`, `PlannedCreatureRequest`, `CombatEncounterPlan` | Host-facing names; `CombatEncounterPlan` avoids the `Encounter` aggregate of `combat` (FR-029; Q-11). |
| Troubleshooting guidance | `EncounterAdvisory`, an interface with an unexported marker method and exactly the variants `ManyCreatures`, `ChallengeRatingZeroCreatures`, `ManyStatBlocks`, `PowerfulCreatures` (doc comment lists them; switched exhaustively) | Optional, p. 203; never changes the outcome. |

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids.

### 4.1 Difficulty, party and the budget table

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `EncounterDifficulty` MUST have exactly the three values `low`, `moderate` and `high`, in that order, and no other. | SRD 5.2.1, "Step 1: Choose a Difficulty", p. 202 | happy |
| FR-002 | `NewEncounterParty(level, size int) (EncounterParty, error)` MUST accept a level from 1 to 20 and a size of at least 1, and MUST return an error for a level below 1, a level above 20, or a size below 1 (the failure of FR-003). `MustEncounterParty` MUST panic with that error. Cases in table 4.1b. | SRD 5.2.1, "Step 2", XP Budget per Character table (levels 1 to 20), p. 202 | unhappy (boundary) |
| FR-003 | `NewEncounterParty` MUST NOT panic for any pair of integers. On failure it MUST return the zero `EncounterParty` and an error that is `errors.Join` of every problem found, each an `EncounterPartyProblem` value (`PartyLevelBelowOne`, `PartyLevelAboveTwenty`, `PartySizeBelowOne`) in that order; the error `errors.Is` `srderr.ErrInvalidInput` and each problem is reachable with `errors.As`. It MUST fail exactly when `MustEncounterParty` panics. A level 0 with size 0 MUST report two problems (`PartyLevelBelowOne`, `PartySizeBelowOne`). The zero `EncounterParty` is invalid and reports `IsZero`. | SRD 5.2.1, p. 202; `doc/rules/architecture.md` section 6; errors-and-results.md FR-023, FR-025 | unhappy (invalid input) |
| FR-004 | `XPBudgetTable.PerCharacter(level, difficulty)` MUST return the printed value for every one of the 20 levels and three difficulties (table 4.1a, 60 cells), and MUST return an error wrapping `srderr.ErrInvalidInput` for a level outside 1 to 20. | SRD 5.2.1, "XP Budget per Character", p. 202 | happy |
| FR-005 | In `XPBudgetTable`, the value MUST increase strictly from `low` to `moderate` to `high` in every row, and MUST increase strictly with level in every column. This is a transcription check on table 4.1a, not an additional rule; it is the monotonic pattern visible in the printed table, p. 202. | SRD 5.2.1, "XP Budget per Character", p. 202 (the printed values; the pattern is read from them, no sentence states it) | unhappy (rule interaction) |
| FR-006 | `ForParty(party EncounterParty, difficulty EncounterDifficulty) XPBudget` MUST equal `PerCharacter(party.Level, difficulty)` multiplied by `party.Size`. Cases in table 4.1c, including the SRD's three worked budgets (200, 1,125 and 46,800 XP). | SRD 5.2.1, "Step 2: Determine Your XP Budget", p. 202; Examples 1 to 3, pp. 202-203 | happy |

Table 4.1a, XP Budget per Character (FR-004, FR-005). Source: SRD 5.2.1, p. 202. Path: happy; levels 1 and 20 also unhappy (boundary).

| Level | Low | Moderate | High | Level | Low | Moderate | High |
|---|---|---|---|---|---|---|---|
| 1 | 50 | 75 | 100 | 11 | 1,900 | 2,900 | 4,100 |
| 2 | 100 | 150 | 200 | 12 | 2,200 | 3,700 | 4,700 |
| 3 | 150 | 225 | 400 | 13 | 2,600 | 4,200 | 5,400 |
| 4 | 250 | 375 | 500 | 14 | 2,900 | 4,900 | 6,200 |
| 5 | 500 | 750 | 1,100 | 15 | 3,300 | 5,400 | 7,800 |
| 6 | 600 | 1,000 | 1,400 | 16 | 3,800 | 6,100 | 9,800 |
| 7 | 750 | 1,300 | 1,700 | 17 | 4,500 | 7,200 | 11,700 |
| 8 | 1,000 | 1,700 | 2,100 | 18 | 5,000 | 8,700 | 14,200 |
| 9 | 1,300 | 2,000 | 2,600 | 19 | 5,500 | 10,700 | 17,200 |
| 10 | 1,600 | 2,300 | 3,100 | 20 | 6,400 | 13,200 | 22,000 |

Table 4.1b, party validation (FR-002, FR-003). Source: SRD 5.2.1, p. 202.

| Label | Level | Size | `NewEncounterParty` | Problems | Path |
|---|---|---|---|---|---|
| lowest valid | 1 | 1 | accepts | none | unhappy (boundary) |
| highest level | 20 | 1 | accepts | none | unhappy (boundary) |
| level zero | 0 | 4 | error | `PartyLevelBelowOne` | unhappy (boundary) |
| negative level | -3 | 4 | error | `PartyLevelBelowOne` | unhappy (invalid input) |
| level above table | 21 | 4 | error | `PartyLevelAboveTwenty` | unhappy (boundary) |
| party size zero | 5 | 0 | error | `PartySizeBelowOne` | unhappy (boundary) |
| negative size | 5 | -1 | error | `PartySizeBelowOne` | unhappy (invalid input) |
| two problems | 0 | 0 | error | `PartyLevelBelowOne`, `PartySizeBelowOne` | unhappy (invalid input) |
| large party | 5 | 1000 | accepts (the SRD sets no upper size) | none | unhappy (boundary) |

Table 4.1c, budget cases (FR-006). Source: SRD 5.2.1, p. 202; examples as labelled.

| Label | Level | Difficulty | Size | Expected budget | Path |
|---|---|---|---|---|---|
| Example 1 | 1 | low | 4 | 200 (50 x 4), p. 202 | happy |
| Example 2 | 3 | moderate | 5 | 1,125 (225 x 5), p. 202 | happy |
| Example 3 | 15 | high | 6 | 46,800 (7,800 x 6), p. 203 | happy |
| minimum | 1 | low | 1 | 50 | unhappy (boundary) |
| maximum row | 20 | high | 1 | 22,000 | unhappy (boundary) |
| uneven step | 3 | high | 2 | 800 (the Moderate to High jump is 225 to 400) | unhappy (boundary) |
| large party | 20 | high | 1000 | 22,000,000 | unhappy (boundary) |

### 4.2 Spending the budget

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `NewPlannedCreature(id content.MonsterID, challengeRating ChallengeRating, ExperiencePoints, count int) (PlannedCreature, error)` MUST hold a `MonsterID` (from content, or host-chosen for a custom creature), a `ChallengeRating`, an `ExperiencePoints` of at least 0 and a `Count` of at least 1, and MUST return an error for negative XP or a count below 1 (FR-044). `MustPlannedCreature` panics with that error. Its total XP MUST be `ExperiencePoints` multiplied by `Count`. | SRD 5.2.1, "Step 3: Spend Your Budget", p. 202 | unhappy (invalid input) |
| FR-008 | The signature MUST be `EvaluateCombatEncounter(party EncounterParty, difficulty EncounterDifficulty, creatures []PlannedCreature) CombatEncounterPlan` (a pure domain function). It MUST build the `XPBudget` itself with `ForParty(party, difficulty)`, MUST keep the party and difficulty on the plan, and MUST compute `SpentXP` as the sum of the total XP of every creature, deducting each creature's XP from the budget as Step 3 describes. Order of creatures MUST NOT change the totals. | SRD 5.2.1, "Step 3: Spend Your Budget", p. 202 | happy |
| FR-009 | When `SpentXP` is at most the budget, `plan.Outcome` MUST be `EncounterWithinBudget` whose `UnspentXP` is the budget minus `SpentXP`. An exact fit MUST be `EncounterWithinBudget` with `UnspentXP` 0. | SRD 5.2.1, "Step 3", p. 202 ("without going over"; "OK if you have a few unspent XP left over") | unhappy (boundary) |
| FR-010 | When `SpentXP` exceeds the budget by any amount, `plan.Outcome` MUST be `EncounterOverBudget` whose `ExcessXP` is `SpentXP` minus the budget. One XP over MUST be `EncounterOverBudget` with `ExcessXP` 1. An over-budget plan MUST be a normal result (a returned plan), not an error. | SRD 5.2.1, "Step 3", p. 202 ("without going over") | unhappy (boundary) |
| FR-011 | An empty creature slice MUST give a plan whose outcome is `EncounterWithinBudget` with `SpentXP` 0 and `UnspentXP` equal to the budget, and no advisories. | SRD 5.2.1, "Step 3", p. 202 | unhappy (empty) |
| FR-012 | A creature with 0 XP MUST add nothing to `SpentXP` but MUST count as a creature in every advisory count (FR-014 to FR-017). | SRD 5.2.1, "CR 0 Creatures", p. 203; "Step 3", p. 202 | unhappy (boundary) |
| FR-013 | `XPBudget.CanAfford(spentSoFar, creatureXP int) (bool, error)` MUST be true if and only if `spentSoFar + creatureXP` is at most the budget, so a host can add creatures one at a time and stop before going over. Negative arguments are FR-033. | SRD 5.2.1, "Step 3", p. 202 | unhappy (boundary) |

Table 4.2a, plans (FR-008 to FR-012, FR-014 to FR-018): printed Cr and XP from the stat blocks of section 2; "built" creatures carry only the numbers needed (not SRD content).

| Label | Party level / size / difficulty | Creatures | Budget | Spent | Result | Advisories | Source | Path |
|---|---|---|---|---|---|---|---|---|
| Example 1a | 1 / 4 / low | 1 Bugbear Warrior (CR 1, XP 200) | 200 | 200 | within, unspent 0 | none | p. 202 | happy |
| Example 1b | 1 / 4 / low | 2 Giant Wasps (CR 1/2, XP 100 each) | 200 | 200 | within, unspent 0 | none | p. 202 | happy |
| Example 1c | 1 / 4 / low | 6 Giant Rats (CR 1/8, XP 25 each) | 200 | 150 | within, unspent 50 | none | p. 202 | happy |
| Example 2a | 3 / 5 / moderate | 2 Druids (CR 2, XP 450 each), 9 Stirges (CR 1/8, XP 25 each) | 1,125 | 1,125 | within, unspent 0 | `ManyCreatures` (11 > 10) | p. 202 | happy |
| Example 2b | 3 / 5 / moderate | 1 Wight (CR 3, XP 700), 1 Warhorse Skeleton (CR 1/2, XP 100), 6 Skeletons (CR 1/4, XP 50 each) | 1,125 | 1,100 | within, unspent 25 | none (8 creatures, 3 stat blocks, no CR above 3) | p. 202 | happy |
| Example 3 | 15 / 6 / high | 2 Adult Red Dragons (CR 17, XP 18,000 each), 2 Fire Giants (CR 9, XP 5,000 each) | 46,800 | 46,000 | within, unspent 800 | `PowerfulCreatures` (Adult Red Dragon, CR 17 > 15) | p. 203 | happy |
| exact fit | 1 / 1 / low | 1 Skeleton (CR 1/4, XP 50) | 50 | 50 | within, unspent 0 | none | p. 202 | unhappy (boundary) |
| one over | 1 / 1 / low | 1 built creature (CR 1/4, XP 51) | 50 | 51 | over, excess 1 | none | p. 202 | unhappy (boundary) |
| well over | 1 / 4 / low | 1 Druid (CR 2, XP 450) | 200 | 450 | over, excess 250 | `PowerfulCreatures` (CR 2 > 1) | p. 202, p. 203 | unhappy (rule interaction) |
| no creatures | 10 / 3 / moderate | none | 6,900 | 0 | within, unspent 6,900 | none | p. 202 | unhappy (empty) |
| zero-XP creature | 1 / 1 / low | 1 built creature (CR 0, XP 0) | 50 | 0 | within, unspent 50 | `ChallengeRatingZeroCreatures` (1) | p. 203 | unhappy (boundary) |

### 4.3 Advisories (optional, from Troubleshooting)

Advisories are structured data with no display text; none changes the outcome (FR-018).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `ManyCreatures` MUST be reported if and only if the total creature count (sum of counts) is greater than twice the party size. It MUST carry the creature count, the party size, and whether the party level is 1 or 2 (the SRD says the guideline "is especially important for characters of level 1 or 2"). Exactly twice the party size MUST NOT report it. Cases in table 4.3a. | SRD 5.2.1, "Many Creatures", p. 203 | unhappy (boundary) |
| FR-015 | `ChallengeRatingZeroCreatures` MUST be reported if and only if at least one creature has Challenge Rating 0 (`isChallengeRatingZero`, FR-031), and MUST carry the number of CR 0 creatures (sum of counts). The SRD gives no numeric threshold for "used sparingly", so any CR 0 creature reports it (Q-6). | SRD 5.2.1, "CR 0 Creatures", p. 203 | unhappy (boundary) |
| FR-016 | `ManyStatBlocks` MUST be reported if and only if the number of distinct stat blocks (distinct `MonsterID`s) is greater than three, and MUST carry that number. Exactly three MUST NOT report it. The SRD says "more than two or three stat blocks"; three is the chosen threshold (Q-7). | SRD 5.2.1, "Number of Stat Blocks", p. 203 | unhappy (boundary) |
| FR-017 | `PowerfulCreatures` MUST be reported if and only if at least one creature has a Challenge Rating strictly greater than the party's level (compared as FR-031 defines), and MUST carry the `MonsterID` of every such creature in plan order. A Challenge Rating equal to the party's level MUST NOT report it. Fractional ratings (1/8, 1/4, 1/2) are below every level. | SRD 5.2.1, "Powerful Creatures", p. 203 | unhappy (boundary) |
| FR-018 | Advisories MUST be independent of one another and MUST NOT change the budget outcome: the same creatures with and without a triggered advisory give the same `SpentXP`, `UnspentXP` or `ExcessXP`. Several advisories MAY be reported together. | SRD 5.2.1, "Troubleshooting", p. 203 ("keep the following in mind") | unhappy (rule interaction) |
| FR-019 | The advisories of a plan MUST be listed in the order the SRD prints the guidance: `ManyCreatures`, `ChallengeRatingZeroCreatures`, `ManyStatBlocks`, `PowerfulCreatures`, so equal inputs give an equal list. | SRD 5.2.1, "Troubleshooting", p. 203 | happy |

Table 4.3a, advisory boundaries (FR-014 to FR-017); creatures hand-built. Source: SRD 5.2.1, "Troubleshooting", p. 203.

| Label | Party level / size | Creatures (CR x count, distinct blocks) | Advisories | Path |
|---|---|---|---|---|
| exactly two per character | 5 / 2 | 1/4 x 4, one block | none | unhappy (boundary) |
| one above two per character | 5 / 2 | 1/4 x 5, one block | `ManyCreatures` (5 > 4, level 1 or 2: false) | unhappy (boundary) |
| low level emphasis | 2 / 2 | 1/8 x 5, one block | `ManyCreatures` (level 1 or 2: true) | unhappy (boundary) |
| single character | 1 / 1 | 1/8 x 3, one block | `ManyCreatures` (3 > 2) | unhappy (boundary) |
| three stat blocks | 5 / 4 | CR 1 x 1, CR 1 x 1, CR 1 x 1; three blocks | none | unhappy (boundary) |
| four stat blocks | 5 / 4 | CR 1 x 1 each, four blocks | `ManyStatBlocks` (4) | unhappy (boundary) |
| CR equals level | 3 / 4 | CR 3 x 1 | none | unhappy (boundary) |
| CR one above level | 3 / 4 | CR 4 x 1 | `PowerfulCreatures` | unhappy (boundary) |
| fractional CR at level 1 | 1 / 4 | CR 1/2 x 1 | none | unhappy (boundary) |
| two powerful creatures | 3 / 4 | CR 4 x 1, CR 5 x 1, CR 2 x 1 | `PowerfulCreatures` (the CR 4 and CR 5 ids, in plan order) | unhappy (rule interaction) |
| all four together | 1 / 1 | CR 0 x 3, CR 2 x 1, CR 1/8 x 1, CR 1/4 x 1 (four blocks, six creatures) | `ManyCreatures`, `ChallengeRatingZeroCreatures` (3), `ManyStatBlocks` (4), `PowerfulCreatures` | unhappy (rule interaction) |

### 4.4 Planning from content

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | `PlanCombatEncounter.Run` MUST resolve each `MonsterID` of the command through the `ContentSource` port with `monsterID.Key()` (`content.NewKey(content.KindMonster, id)`, content-schema-and-decoder.md FR-114), and MUST build each `PlannedCreature` from the monster's printed `ExperiencePoints` and `ChallengeRating`, never from the CR row of the XP table. A monster whose printed XP differs from the table row ("Archmage", CR 12, XP 8,000; content-monsters.md FR-082) MUST count its printed 8,000. | SRD 5.2.1, "Step 3", p. 202 ("XP value in its stat block"); content-monsters.md FR-032 | happy |
| FR-021 | `PlanCombatEncounter.Run` MUST NOT use a monster's XP "in lair" (content-monsters.md FR-032); it MUST use the standard printed XP. | SRD 5.2.1, "Step 3", p. 202; "Adult Red Dragon" (XP 18,000, or 20,000 in lair), p. 318; Example 3, p. 203 (counts 18,000) | unhappy (rule interaction) |
| FR-022 | If any `MonsterID` is not found, the use case MUST return the result `PlanCombatEncounterContentMissing` (and a nil error) carrying one `content.MissingContent{Kind, ID}` (`monsterID.Missing()`; shared-types-and-ownership.md FR-018) for every missing id, in command order, and MUST NOT return a partial plan. An id named by two entries and missing is reported once, at its first position (FR-035). A lookup that yields content which is not a `Monster` MUST be reported as missing for that id (a source contract breach is not a plan). | `doc/rules/architecture.md` section 4 (failure contract); testing.md section 4 ("Missing or bad content") | unhappy (missing content) |
| FR-023 | An error returned by the `ContentSource` port MUST be wrapped once with `srderr.WrapPort(content.Port, "lookup", err)` and returned as the `error` result, with a nil `PlanCombatEncounterResult`; `errors.Is` and `errors.As` still reach the original cause. No plan MUST be returned. A cancelled `ctx` reaches the caller as `context.Canceled` (errors-and-results.md FR-054). | architecture.md section 4; errors-and-results.md FR-031 | unhappy (collaborator failure) |
| FR-024 | An invalid command MUST return `PlanCombatEncounterRejected` carrying every problem as the closed interface `PlanCombatEncounterProblem` (unexported marker method; variants `PlanCombatEncounterPartyInvalid{Problem EncounterPartyProblem}` and `PlanCombatEncounterCountBelowOne{Index int}`): one `PlanCombatEncounterPartyInvalid` for each problem of FR-003 (in FR-003 order), then one `PlanCombatEncounterCountBelowOne` for each entry with a count below 1 (`Index` is the zero-based position in the command), and MUST NOT call `ContentSource`. | architecture.md section 6 | unhappy (invalid input) |
| FR-025 | Two entries naming the same `MonsterID` MUST be combined into one creature whose count is the sum, so they are one stat block for FR-016. The merged creature takes the position of the first occurrence, which fixes the "plan order" of FR-017 (FR-035). | SRD 5.2.1, "Number of Stat Blocks", p. 203 | unhappy (rule interaction) |
| FR-026 | `PlanCombatEncounter.Run` MUST NOT read randomness or the clock for its result. Equal commands against an unchanged source MUST return equal results, and the command, the monsters and the returned collections MUST NOT be altered or be mutable by the caller (accessors return copies). | testing.md section 4 (determinism, immutability); architecture.md section 3 | unhappy (immutability) |
| FR-027 | `PlanCombatEncounter.Run` MUST return the same result with the default no-op telemetry as with a recording one. The telemetry itself is pinned by FR-036 to FR-041. | observability.md section 9 | happy |

### 4.5 Package rules

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | The domain files of this spec (pure, synchronous) MUST import only the kernel packages and the `content` package (`ChallengeRating`, `MonsterID`); the use-case file (`plan_combat_encounter_usecase.go`) MUST import only the kernel packages (`srderr`, `telemetry`) and `content` (for example `content.Key`, `content.Kind`, `content.MissingContent`, `Monster`, `MonsterID`, `ChallengeRating`, `content.Port`). The port the use case needs is the consumer-side interface `ContentSource` declared in package `toolbox` with the single `Lookup` method of `content.Source`, so no content implementation is imported. No file of this spec may import `combat`, `character` or `d20`, although the context map permits those edges for the `toolbox` context (shared-types-and-ownership.md FR-045; toolbox-hazard-effects.md FR-002; architecture-guardrails.md FR-018 to FR-021, upstream-set check FR-019). An architecture test (`internal/archtest`) MUST fail on such an import and on any `toolbox` file importing a context that is not upstream of it in the context map. | architecture.md sections 2, 9 | unhappy (forbidden edge) |
| FR-029 | The exported names of package `toolbox` for this spec MUST NOT collide with any other exported name in the module's public packages, in particular `Encounter` and `EncounterID` of `combat`. No exported variant MAY use a generic name (`Applied`, `Rejected`, `Found`, `NotFound`, `WithinBudget`, `OverBudget`, `BudgetOutcome`); the use-case variants and problems are `PlanCombatEncounter`-prefixed and the plan's outcome variants are `Encounter`-prefixed (shared-types-and-ownership.md FR-017). The exported-name rule is toolbox-hazard-effects.md FR-006. | architecture.md sections 4, 9 (exported-name scan) | unhappy (invalid design) |
| FR-030 | Package `toolbox` and every file of this spec MUST NOT name any consuming project or organisation, and the budget table MUST NOT contain a value not on p. 202. | AGENTS.md ("Hard constraints") | unhappy (invalid design) |

### 4.6 Additions (new ids; no earlier id changed)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | The toolbox domain MUST compare a rating with the integer party level through one unexported function `challengeRatingExceedsLevel(cr, level)` that is true if and only if the rating is strictly greater than the level (a level is an integer, so a fractional rating is below every level), and MUST test for CR 0 through one unexported `isChallengeRatingZero(cr)`. It MUST use the ordering (`Compare`) and `IsZero` of `ChallengeRating`, which content-monsters.md FR-097 declares (total order 0 < 1/8 < 1/4 < 1/2 < 1 < 2 < ... < 30), and MUST NOT keep its own list of ratings. Test fixtures build a rating from its printed string (`content.MustParseChallengeRating("1/8")` or `content.ParseChallengeRating`). Cases in table 4.6a. | SRD 5.2.1, "Powerful Creatures", p. 203; "Experience Points by Challenge Rating", p. 256 (ratings 0, 1/8, 1/4, 1/2, 1 to 30) | unhappy (boundary) |
| FR-032 | `NewXPBudget(xp int) (XPBudget, error)` MUST accept an XP of 0 or more and MUST return an error wrapping `srderr.ErrInvalidInput` for a negative value (FR-045). `ForParty` MUST always produce a budget of at least 50 (table 4.1a). Zero is valid for a hand-built budget. | n/a (architecture.md section 6; Q-13) | unhappy (invalid input) |
| FR-033 | `XPBudget.CanAfford` MUST return an error wrapping `srderr.ErrInvalidInput` when `spentSoFar` or `creatureXP` is negative, and MUST accept 0 for either. | n/a (architecture.md section 6) | unhappy (invalid input) |
| FR-034 | The command entry MUST be the immutable `PlannedCreatureRequest` (`MonsterID content.MonsterID`, `Count int`; a plain value struct); `PlanCombatEncounterCommand` MUST hold `[]PlannedCreatureRequest` and copy it on use. A count below 1 is not a constructor error (untrusted input): it is a `PlanCombatEncounterCountBelowOne` problem (FR-024). | architecture.md section 4 (explicit immutable types), section 6 | unhappy (invalid input) |
| FR-035 | When the same `MonsterID` appears in several entries, the merged creature MUST sit at the position of the first occurrence, with the summed count, and the `PowerfulCreatures` ids MUST follow that order. A duplicate id that is missing MUST appear once in `PlanCombatEncounterContentMissing`, at its first position. A count below 1 in any duplicate entry MUST be reported by FR-024 (by index) before any lookup. | SRD 5.2.1, "Number of Stat Blocks", p. 203 | unhappy (rule interaction) |
| FR-036 | Each call MUST emit exactly one span named `srd5e.toolbox.plan_combat_encounter` (started with `tracer.Start(ctx, name)` and ended with `defer span.End()`) with attributes `srd5e.outcome` (`success` for `within_budget` and `over_budget`; `failure` for `rejected_invalid_command`, `rejected_content_missing` and `error`; values per shared-types-and-ownership.md FR-041), `srd5e.toolbox.result` and `srd5e.toolbox.difficulty`; `difficulty` is always present, for every `result` including `rejected_invalid_command`, `rejected_content_missing` and `error`, because the command's difficulty is a valid enum in every case. `within_budget` and `over_budget` are taken from `plan.Outcome` (FR-043). The call runs as FR-046 states. | observability.md sections 5, 6 | happy |
| FR-037 | Each call MUST increment `srd5e.toolbox.operations` once with `srd5e.toolbox.operation` = `plan_combat_encounter`, the `srd5e.outcome` of FR-036 and `srd5e.toolbox.result` = `within_budget`, `over_budget`, `rejected_invalid_command`, `rejected_content_missing` or `error` (a returned port error, FR-041), and MUST record the call duration in `srd5e.toolbox.operation_duration` with the same operation value, for every outcome (toolbox-hazard-effects.md FR-072, FR-078, FR-081). | observability.md sections 5, 6; toolbox-hazard-effects.md FR-072, FR-078, FR-081 | happy |
| FR-038 | For a returned plan, `srd5e.toolbox.encounter_advisories` MUST be incremented once per advisory present, with its `srd5e.toolbox.advisory` kind, and `srd5e.toolbox.encounter_creatures` MUST record the total creature count with `srd5e.toolbox.difficulty`. Neither MUST be recorded for a rejected or content-missing call. | observability.md section 6 | happy |
| FR-039 | A rejected command (`PlanCombatEncounterRejected`) MUST set the span error status once, increment `srd5e.toolbox.operations` once with `srd5e.outcome` `failure` and `srd5e.toolbox.result` `rejected_invalid_command`, MUST NOT carry an `error.type`, MUST NOT increment `srd5e.toolbox.errors` (a rejection is a result about caller input, as in toolbox-hazard-effects.md section 7), and MUST NOT log. | observability.md section 7 | unhappy (failure) |
| FR-040 | A content-missing result is a returned failure result: it MUST set the span error status once, increment `srd5e.toolbox.operations` once with `srd5e.outcome` `failure` and `srd5e.toolbox.result` `rejected_content_missing` (the variant attribute that distinguishes it), MUST NOT carry an `error.type` and MUST NOT increment `srd5e.toolbox.errors` (that counter is for returned port errors only), and MUST NOT log. | observability.md section 7 ("Errors: who records what": a returned failure result has no `error.type`); shared-types-and-ownership.md FR-051 | unhappy (failure) |
| FR-041 | A port error from `ContentSource` MUST be recorded once at the use-case boundary: `span.RecordError(err)` is called once on the wrapped `*srderr.PortError` (exactly one recorded error status, FR-046), with one increment of `srd5e.toolbox.errors` with `srd5e.toolbox.operation` and `error.type` = `srderr.ErrorType(err)` (`content_error`, errors-and-results.md FR-034), one `srd5e.toolbox.operations` increment with `srd5e.outcome` `failure` and `srd5e.toolbox.result` `error`, one `operation_duration` record, and no log; the error is then returned (FR-023). | observability.md section 7; toolbox-hazard-effects.md FR-074, FR-078 | unhappy (failure) |
| FR-042 | `PlanCombatEncounter` MUST NOT call another use case and MUST NOT construct a nested `telemetry.Telemetry` bundle. It composes only the `ContentSource` port and this spec's pure domain (`EvaluateCombatEncounter`), and records only its own signals, taking attribute values from the returned plan. The no-use-case rule is checked by architecture-guardrails.md FR-097. | shared-types-and-ownership.md FR-044; architecture-guardrails.md FR-097 | unhappy (nested telemetry) |
| FR-043 | `CombatEncounterPlan` MUST expose exactly these accessors: `Party`, `Difficulty`, `Budget` (`XPBudget`), `SpentXP` (`int`), `Outcome` (the closed interface `EncounterBudgetOutcome`, unexported marker method, variants `EncounterWithinBudget{UnspentXP}` and `EncounterOverBudget{ExcessXP}`), `Creatures` (a copy) and `Advisories` (a copy). `UnspentXP` and `ExcessXP` exist only on the outcome variants, not on the plan, so an inconsistent pair cannot be built. `Outcome` is the single source of truth for within or over budget. `PlanCombatEncounterResult` has one success variant, `PlanCombatEncounterPlanned{Plan}`, returned for every fully resolved command whatever `plan.Outcome` is; there are no separate within-budget and over-budget use-case variants. | shared-types-and-ownership.md principle 4 (no illegal state representable); architecture.md section 3 | unhappy (invalid design) |
| FR-044 | `NewPlannedCreature` MUST NOT panic for any integers. On failure it MUST return the zero `PlannedCreature` and an error that is `errors.Join` of every `PlannedCreatureProblem` (`CreatureXPBelowZero`, `CreatureCountBelowOne`, in that order), `errors.Is` `srderr.ErrInvalidInput`, and MUST fail exactly when `MustPlannedCreature` panics. | `doc/rules/architecture.md` section 6 | unhappy (invalid input) |
| FR-045 | `NewXPBudget` MUST NOT panic for any integer. On failure it MUST return the zero `XPBudget` and an `XPBudgetProblem` value (`XPBelowZero`) as the error (`errors.Is` `srderr.ErrInvalidInput`), and MUST fail exactly when `MustXPBudget` panics (FR-032). | `doc/rules/architecture.md` section 6 | unhappy (invalid input) |
| FR-046 | `PlanCombatEncounter.Run(ctx, cmd)` MUST run its whole body inside one span (`srd5e.toolbox.plan_combat_encounter`, `tracer.Start` with `defer span.End()`) and time itself with `telemetry.StartTimer(clock, operationDuration)` and `timer.Stop` (`srd5e.toolbox.operation_duration`), both from the injected `telemetry.Telemetry` (`WithTelemetry` option, default no-op) and the injected clock (`WithClock`, default `time.Now`). The use case MUST set the `srd5e.toolbox.result` and `srd5e.outcome` attributes and increment `srd5e.toolbox.operations` (and `srd5e.toolbox.errors` for a returned port error) in one deferred step that reads the outcome, then return the error. The use case is the sole recorder of a returned port error on the span (`span.RecordError` exactly once); `recover` is not used (errors-and-results.md FR-047). | `doc/rules/observability.md` section 4; `doc/rules/architecture.md` section 6; errors-and-results.md FR-047 | unhappy (failure) |

Table 4.6a, `challengeRatingExceedsLevel` (FR-031). Source: SRD 5.2.1, pp. 203, 256. Path: unhappy (boundary).

| CR | Level 1 | Level 2 | Level 17 | Level 20 |
|---|---|---|---|---|
| 0 | false | false | false | false |
| 1/8 | false | false | false | false |
| 1/4 | false | false | false | false |
| 1/2 | false | false | false | false |
| 1 | false | false | false | false |
| 2 | true | false | false | false |
| 17 | true | true | false | false |
| 30 | true | true | true | true |

## 5. Model

Types are those of section 3 and the FRs, all in package `toolbox` (`doc/rules/architecture.md` section 1: one package per context). The domain types and functions (pure, immutable, synchronous) sit in their own files; the use case, command, request, result and problem types sit in `plan_combat_encounter_usecase.go`. `PlanCombatEncounterProblem` is a closed interface (unexported marker method) with the variants `PlanCombatEncounterPartyInvalid{Problem}` and `PlanCombatEncounterCountBelowOne{Index}`. The domain keeps `challengeRatingExceedsLevel` and `isChallengeRatingZero` unexported (FR-031). The party is on the plan because the advisories need its size and level.

Invariants: no negative XP, no count below 1, no party outside 1 to 20 or of size 0; totals equal the sum of the creatures; within or over exists once, on `plan.Outcome` (FR-043); accessors return copies. Failures: rejected input and missing content are results; a `ContentSource` failure is a returned `*srderr.PortError`; invalid constructor input is an error wrapping `srderr.ErrInvalidInput` (`MustX` panics); over budget is a normal outcome (FR-010).

## 6. Interface

- `func NewPlanCombatEncounter(source ContentSource, opts ...Option) (PlanCombatEncounter, error)` returns an error when `source` is nil. `(PlanCombatEncounter).Run(ctx context.Context, cmd PlanCombatEncounterCommand) (PlanCombatEncounterResult, error)`; `ctx` is first because `ContentSource.Lookup` may do I/O. `ContentSource` is declared in package `toolbox` as a one-method interface and is satisfied by `content.Source`. The options are `WithTelemetry` and `WithClock` (default `telemetry.Noop`, shared-types-and-ownership.md FR-036); telemetry FR-046. `PlanCombatEncounterCommand`: `PartyLevel`, `PartySize` (`int`, validated by FR-024), `Difficulty`, ordered `[]PlannedCreatureRequest` (FR-034).
- `PlanCombatEncounterResult` is a closed interface (unexported marker method) with exactly the variants `PlanCombatEncounterPlanned{Plan}` (the only success variant, FR-043), `PlanCombatEncounterRejected{Problems}` and `PlanCombatEncounterContentMissing{Missing}` (shared-types-and-ownership.md FR-018); callers switch on it exhaustively. A port failure is the returned `error`, not a variant.
- No new driven port. A host with XP values or custom creatures calls `EvaluateCombatEncounter` directly (FR-007, FR-008).
- Exports: the exported identifiers of package `toolbox`, every one with a doc comment starting with its name; `toolbox/doc.go` lists inputs and outputs. No event is defined.

## 7. Observability

`srd5e.toolbox.operations`, `operation_duration`, `errors` and the signal-name constants (`toolbox/signals.go`) are
owned by toolbox-hazard-effects.md section 7. The span, histogram and counter names below are added as constants there (toolbox-hazard-effects.md FR-006). This spec adds, by amendment (Q-12b):
operation `plan_combat_encounter`; `srd5e.toolbox.result` values `within_budget`,
`over_budget` (`success`), `rejected_invalid_command`, `rejected_content_missing`,
`error` (`failure`); `error.type` `content_error` (the `content` port's value from errors-and-results.md section 7; no
`content_not_found`, FR-040). It owns only these signals:

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.plan_combat_encounter` | span | `srd5e.outcome`; `srd5e.toolbox.result`; `srd5e.toolbox.difficulty` (always); error status on failure (once, by the use case, FR-046) | One span per call (FR-036), lookups as children | Trace sampling; error rate |
| `srd5e.toolbox.encounter_advisories` | counter `{advisory}` | `srd5e.toolbox.advisory` (`many_creatures`, `challenge_rating_zero_creatures`, `many_stat_blocks`, `powerful_creatures`) | Which advisories fire (FR-038) | Dashboard only |
| `srd5e.toolbox.encounter_creatures` | histogram `{creature}` | `srd5e.toolbox.difficulty` | Plan size (FR-038) | Dashboard only |

Monitor: `srd5e.toolbox.errors` non-zero over 5 minutes; `rejected_content_missing`
above 5% of `plan_combat_encounter` operations (filter on `srd5e.toolbox.operation`) over an hour; `over_budget` share moving 20 points from its
7-day baseline over at least 1,000 plans. Missing content is counted only through
(FR-040), a port failure also in `errors` (FR-041); no log in any path
(FR-039 to FR-041), the domain is silent, and party level and size are not
attributes. Failure modes: unknown monster id, content source failure, invalid party or count, over budget (normal); distinguished by `srd5e.toolbox.result`. The result carries budget, spent XP, difference, creatures and
advisories, so diagnosis needs no log.

The new keys `srd5e.toolbox.difficulty` and `srd5e.toolbox.advisory` go into
`doc/observability/signals.md` (Q-12c).

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-003 | unit | Enum values and order; table 4.1b one test per row, `NewEncounterParty` and `MustEncounterParty` agree (`errors.Is` `ErrInvalidInput`, `errors.As` each problem). |
| FR-004 to FR-006 | unit | Table 4.1a one test per cell (60) plus levels 0 and 21, monotonic check; table 4.1c including the three worked budgets. |
| FR-007, FR-008, FR-044 | unit | `EvaluateCombatEncounter` with a host-chosen `MonsterID` absent from every source (custom creature path); negative XP, count 0, -1, 1; total XP; `NewPlannedCreature` and `MustPlannedCreature` agree; XP -1 with count 0 reports two problems in order. |
| FR-008 to FR-012 | unit | Table 4.2a; permuted-slice order-independence; exact fit and one over. |
| FR-013, FR-032, FR-033, FR-045 | unit | `CanAfford` at equal, one below, one above; negative arguments and negative budget return errors; 0 accepted; `NewXPBudget` -1 invalid, 0 and 50 valid. |
| FR-014 to FR-019 | unit | Table 4.3a one test per row; same creatures with an advisory removed give identical totals; "all four together" pins order. |
| FR-031 | unit | Table 4.6a one test per cell (32); `isChallengeRatingZero` only for CR 0. |
| FR-020, FR-021 | use case | `contenttest.FakeSource` with the ten example monsters, an Archmage-like monster (XP 8,000 at CR 12) and an in-lair figure; printed XP used. |
| FR-022, FR-025, FR-035 | use case | Two unknown ids give two `MissingContent` in order, no partial plan; a missing id named twice appears once at its first position; non-`Monster` content reported missing; the same id twice is one block at its first position; a merged duplicate counts once for `ManyStatBlocks`; `PowerfulCreatures` ids follow first-occurrence order. |
| FR-023, FR-041, FR-046 | use case | A failing `contenttest` source: the `*srderr.PortError` is returned once and `errors.Is` reaches the double's error (errors-and-results.md FR-031); exactly one `RecordError` (the use case's); `result` attribute and counters set despite the error; duration via the injected clock and `StartTimer`; one `errors` increment (`content_error`); one `operations` increment (`failure`, `error`); no log. A panicking telemetry double propagates the panic, the span `End` double is still called and no counter is incremented (errors-and-results.md FR-051). A cancelled `ctx` returns `context.Canceled` and is not a result (FR-054). |
| FR-024, FR-034 | use case | Party 0/0 and counts 0 and -1 give `Rejected` with `PlanCombatEncounterPartyInvalid` x2 then `PlanCombatEncounterCountBelowOne{Index}` x2; zero source calls. |
| FR-026, FR-027 | unit, use case | Inputs unchanged; mutating a returned slice leaves the plan unchanged; repeat equal; no roller in the constructor; each Example equal under the default no-op telemetry and a recording one. |
| FR-036 to FR-040 | use case | Recording `telemetrytest` bundle: span name, attribute keys and values for both outcomes (from `plan.Outcome`); `difficulty` on every span; one operations increment and one duration record; one advisory increment per kind (Example 2a, Example 3, "all four together"); creatures histogram; no advisory/creature signal on failures; rejected and content-missing: span error once, operations `failure` with the matching result, no `error.type`, `errors` untouched, no log; signal and attribute names asserted through the `toolbox` signal constants. |
| FR-042, FR-043 | unit, use case | One span and no other use-case signal; the constructor takes no use case; plan accessor set pinned in a unit test; `plan.Outcome` has two variants, `UnspentXP`/`ExcessXP` are not plan accessors; `PlanCombatEncounterResult` has no within/over variant; the distinction is read only from `plan.Outcome`; `PlanCombatEncounterPlanned` for exact fit, one over, well over. |
| FR-028 to FR-030 | architecture | `internal/archtest` import allow-list (domain files: kernel and `content`; use-case file: plus `srderr` and `telemetry`; none from `combat`/`character`/`d20`); exported-name scan for generic and duplicate names; consumer-name scan. |
| Scenario | scenario | `scenarios_test` through the public API only: Examples 1 to 3 with bundled thin-slice content when the ten monsters are bundled. |

- Doubles: `contenttest.FakeSource` (and a failing source) and the `telemetrytest` recorder
  (`content/contenttest`, `telemetry/telemetrytest`; shared-types-and-ownership.md
  FR-038). No new port, so no new contract suite; the `ContentSource` consumer interface is
  exercised by the `content` contract suite.
- Mutation testing: `gomu` on `toolbox` only, run inside Docker with `--cpus` and `--workers`
  capped at a quarter of the host's processors (command in testing.md section 8), score at least
  80; the `>`/`>=` boundaries of FR-009, FR-010, FR-014,
  FR-016, FR-017 are covered by boundary rows. FR-005 is a transcription check.

## 9. Open questions and decisions

Amendments requested, decided (Q-12): (a) content-monsters.md's out-of-scope line (Gameplay Toolbox, pp. 192-203) will name this spec (decided; the edit belongs to content-monsters.md, whose out-of-scope line does not yet name it). (b) Done by Q-15 of
toolbox-hazard-effects.md (its section 7 lists only its own operations and the shared keys, and names `content_error` as this spec's error type; the operation and result values stay in this spec's section 7). (c) `doc/observability/signals.md` gets the new signals and
attributes when the code lands (a code-time task, not a question). Open: none; escalated: none.

Settled: Q-1, Q-5 by shared-types-and-ownership.md FR-045. Q-2: party level is `int`, not `CharacterLevel`. Q-3: FR-031 uses content-monsters.md FR-097 only. Q-4: XP "in lair" ignored (Example 3 counts 18,000). Q-6, Q-7: any CR 0 creature reports; threshold three stat blocks. Q-8: Examples 2a and 3 trigger advisories (table 4.2a). Q-9: duplicates combine (FR-025). Q-10: stored integer XP of a CR 0 monster is used. Q-11: names avoid `Encounter` (FR-029). Q-13: raw non-negative `int`, not `ExperienceAmount` (positive-only, character-aggregate.md FR-022).

## 10. Changelog impact

`CHANGELOG.md`, under `Unreleased`, **Added**: "Combat Encounter Difficulty:
`EncounterParty`, `XPBudget` with the XP Budget per Character table (levels 1 to
20), `CombatEncounterPlan`, optional `EncounterAdvisory` values from the SRD's
Troubleshooting guidance, and the `PlanCombatEncounter` use case, in the new
`toolbox` context (SRD 5.2.1, pp. 202-203)." Not **BREAKING**: new API only,
before 0.1.0. The `toolbox` context is decided by shared-types-and-ownership.md
FR-045 and carried by toolbox-hazard-effects.md FR-002 and FR-005.
