# Class: Fighter

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `derived-stats.md`, `character-aggregate.md`, `character-creation.md`, `rests-and-hit-dice.md`, `level-advancement-and-multiclassing.md`, `feats-behaviour.md`, `heroic-inspiration.md`, `weapon-properties-and-mastery.md`, `d20-tests.md`, `ability-scores.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `content-classes.md`, `content-source-port.md`, `content-species-backgrounds-feats.md`, `content-equipment.md`, `content-schema-and-decoder.md`, `errors-and-results.md`, `observability-ports.md` |

## 1. Purpose and scope

Specifies the **behaviour** of the Fighter class from SRD 5.2.1 (printed pp. 47-49) in the
`character` context: level-derived numbers, resource spending and recovery, the Fighter-specific
choices (Fighting Style, Weapon Mastery) and the declared effects (critical-hit range, Advantage
sources, Death Saving Throw benefits) that other contexts apply. Rules are pure domain functions
from (character state, inputs) to (new state, events) taking a roll as a value
(shared-types-and-ownership.md FR-023, `doc/rules/architecture.md` §4); use cases call only the
`dice.Roller` and `ContentSource` ports and report telemetry. State is a `FighterState` in the
aggregate's `FeatureStates` (FR-145); effects on hit points, action economy or a roll in progress
are returned as descriptions for the owner to apply.

**Content is not specified here.** The Fighter `ClassDefinition` (traits, Features table, features,
Champion) is bundled data owned by content-classes.md (shared-types-and-ownership.md FR-025); its
former requirements are in section 4.11. Features per level, pending ASI, Epic Boon and subclass
reports and the generic skill and Starting Equipment rules belong to content-classes.md,
level-advancement-and-multiclassing.md (FR-016, FR-017) and character-creation.md (FR-015 to
FR-020). Class specs define no `ClassDefinition`, `SubclassDefinition`, `ProgressionRow` or
`FeatureEntry`.

**Out of scope:** hit point bookkeeping and the cap at the maximum (SRD 5.2.1, "Damage and Healing",
p. 17; results carry the uncapped amount); turn tracking, action economy, Opportunity Attacks,
Initiative, Death Saving Throws and start-of-turn triggers (FR-134, FR-135): `combat` and the
consumer; feat content, ASI and Epic Boon grants, subclass choice, level 1 and multiclass grants
(shared-types-and-ownership.md FR-020; the chosen subclass is an input); Extra Attack across classes
(SRD 5.2.1, "Multiclassing", p. 25; FR-027); spellcasting (the SRD Fighter and Champion have none);
Mastery Properties' own rules (weapon-properties-and-mastery.md); starting equipment; persistence.
Fighter subclasses other than Champion are **not found anywhere in the SRD 5.2.1 PDF**; none
included.

## 2. Source

PDF page numbers equal printed page numbers.

- SRD 5.2.1, "Fighter" (Core Fighter Traits, "Level 1: Fighting Style"), "Fighter Features" table,
  p. 47; features "Level 1: Second Wind" to "Level 20: Three Extra Attacks", p. 48 (Tactical Shift,
  Tactical Master, Studied Attacks are **base Fighter features**); "Fighter Subclass: Champion", p.
  49.
- Supporting pages are cited per row. `n/a (package design)` rows derive from `doc/rules/*.md` and
  shared-types-and-ownership.md, not the SRD.

## 3. Vocabulary

SRD wording is used exactly (`doc/rules/architecture.md` §3).

| SRD term | Type | Notes |
|---|---|---|
| Fighter, Core Fighter Traits, Fighter Features table, Champion | content (content-classes.md) | Data only |
| Fighter level | `FighterLevel` | The Fighter entry of `ClassLevels` (derived-stats.md): Fighter levels only, not total level |
| Fighting Style, Weapon Mastery | class features | `FightingStyleChoice` (feats-behaviour.md); `WeaponMasteryChoice` (weapon-properties-and-mastery.md) |
| Second Wind, Action Surge, Indomitable | features; resources | Limited uses; recovery by `RestKind` (rests-and-hit-dice.md) |
| Tactical Mind, Tactical Shift, Tactical Master, Studied Attacks | base features | 1d10 on a failed check; move on Second Wind; Push, Sap or Slow replacement; Advantage after a miss |
| Improved/Superior Critical, Remarkable Athlete, Additional Fighting Style, Heroic Warrior, Survivor | Champion features | Critical on 19-20 then 18-20; Advantage on Initiative and Athletics; second style; Heroic Inspiration; Death Save benefits and Heroic Rally |

**Go conventions in this spec** (binding vocabulary: errors-and-results.md,
shared-types-and-ownership.md). Function and method names written lowerCamelCase
below (`rowAt`, `activateRage`) are exported PascalCase in Go (`RowAt`,
`ActivateRage`); enum-like values written lowerCamelCase (`notABarbarian`) are typed
constants with a `String()` method. A "closed" result or choice is an interface with
an unexported marker method, one struct per variant, handled by an exhaustive type
switch; the variant list is in its doc comment. Untrusted input is validated by
`NewX` / `ParseX`, which return `(T, error)` with a typed, `errors.As`-able error and
report every problem at once with `errors.Join`; a port failure is a returned
`*srderr.PortError` (`errors.Is` / `errors.As`); a panic carrying an error value is
reserved for programmer errors (violated precondition) and appears only in `MustX`
helpers and unreachable branches. A function that may do I/O or emit spans takes
`context.Context` first; pure rule functions do not.

## 4. Requirements

Stable ids, never renumbered; ids moved to another spec are in "Withdrawn ids" (section 4.11).
"Fighter level `N`" means levels in the Fighter class. "Rejected" means a closed result variant, not
an error (section 5). Reason names are lowerCamel, enumerated per result type in section 5.
Content a lookup cannot find is the `...ContentMissing` variant carrying `MissingContent`, not a
reason (shared-types-and-ownership.md FR-017, FR-018).

### 4.1 Progression

Table A is the SRD Fighter Features table (p. 47), one row per band of Fighter levels where every
column is constant, tested at every level in its band (FR-020 to FR-027). Proficiency Bonus is the
single-class column as printed; multiclass follows total level (FR-114), so `FighterLevelStats` does
not carry it.

Second Wind and Weapon Mastery are `count` columns of the Fighter definition (content-classes.md
Table CC: `second_wind`, `weapon_mastery`) read by `ClassColumnId`; Action Surge, Indomitable and
attacks ("Attack [Action]", p. 177) are **rule constants** of `FighterProgression`, cross-checked
against the row annotations by a test (FR-150). It is built once by the host (FR-149) and has no
port and no clock; level features come from content-classes.md FR-048, so there is no
`FighterFeature`.

| Row | Fighter level | Proficiency Bonus (single-class) | Second Wind uses | Weapon Mastery kinds | Action Surge uses | Indomitable uses | Attacks per Attack action |
|---|---|---|---|---|---|---|---|
| L01 | 1 | +2 | 2 | 3 | 0 | 0 | 1 |
| L02 | 2-3 | +2 | 2 | 3 | 1 | 0 | 1 |
| L04 | 4 | +2 | 3 | 4 | 1 | 0 | 1 |
| L05 | 5-8 | +3 | 3 | 4 | 1 | 0 | 2 |
| L09 | 9 | +4 | 3 | 4 | 1 | 1 | 2 |
| L10 | 10 | +4 | 4 | 5 | 1 | 1 | 2 |
| L11 | 11-12 | +4 | 4 | 5 | 1 | 1 | 3 |
| L13 | 13-15 | +5 | 4 | 5 | 1 | 2 | 3 |
| L16 | 16 | +5 | 4 | 6 | 1 | 2 | 3 |
| L17 | 17-19 | +6 | 4 | 6 | 2 | 3 | 3 |
| L20 | 20 | +6 | 4 | 6 | 2 | 3 | 4 |

Table B, Fighter-specific choices on gaining a level (FR-028, FR-115); other levels produce none (ASI,
Epic Boon and subclass choices: level-advancement-and-multiclassing.md FR-016, FR-017).

| Row | Gained level | Subclass input | Choices |
|---|---|---|---|
| C01 | 1 | any | Fighting Style feat; 3 Weapon Mastery kinds |
| C04 | 4 | any | 1 additional Weapon Mastery kind |
| C07a | 7 | Champion | Additional Fighting Style feat |
| C07b | 7 | none chosen | none |
| C10 | 10 | any | 1 additional Weapon Mastery kind |
| C16 | 16 | any | 1 additional Weapon Mastery kind |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | A Fighter level below 1 or above 20 MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) from the constructor; `NewFighterLevel` MUST return an error instead. | SRD 5.2.1, "Fighter Features", p. 47 | unhappy (boundary) |
| FR-021 | For a single-class Fighter, the Proficiency Bonus in each Table A row MUST equal the rule for a total level equal to that Fighter level (+2 at 1-4, +3 at 5-8, +4 at 9-12, +5 at 13-16, +6 at 17-20). A cross-check of the Fighter `ProgressionRow`s (content-classes.md) against derived-stats.md; `FighterLevelStats` does not carry it (FR-114). | SRD 5.2.1, "Fighter Features", p. 47; "Proficiency Bonus" table, p. 8 | unhappy (rule interaction) |
| FR-022 | Second Wind maximum uses MUST follow Table A (2 at levels 1-3, 3 at 4-9, 4 at 10-20). | SRD 5.2.1, "Fighter Features", p. 47 | happy |
| FR-023 | Weapon Mastery kind count MUST follow Table A (3 at levels 1-3, 4 at 4-9, 5 at 10-15, 6 at 16-20). | SRD 5.2.1, "Fighter Features", p. 47 | happy |
| FR-024 | Action Surge maximum uses MUST be 0 below level 2, 1 at levels 2-16, 2 at levels 17-20. | SRD 5.2.1, "Level 2: Action Surge", p. 48 | unhappy (boundary) |
| FR-025 | Indomitable maximum uses MUST be 0 below level 9, 1 at 9-12, 2 at 13-16, 3 at 17-20. | SRD 5.2.1, "Fighter Features", p. 47; "Level 9: Indomitable", p. 48 | unhappy (boundary) |
| FR-026 | Attacks per Attack action MUST be 1 below level 5, 2 at levels 5-10, 3 at 11-19, 4 at level 20. | SRD 5.2.1, "Level 5: Extra Attack", "Level 11: Two Extra Attacks", "Level 20: Three Extra Attacks", p. 48; "Attack [Action]", p. 177 | unhappy (boundary) |
| FR-027 | The number of attacks MUST come from the Fighter level alone, not total level; Fighter 5 / another class 3 gives 2. | SRD 5.2.1, "Level 5: Extra Attack", p. 48; "Multiclassing", p. 25 | unhappy (rule interaction) |
| FR-028 | `FighterProgression.choicesOnGaining(level, subclass)` MUST return only the Fighter-specific choices now due, per Table B: the Fighting Style feat at level 1, the Weapon Mastery kinds, and the Additional Fighting Style at level 7 for Champion. A level with none produces an empty list. `subclass` is the chosen subclass or none. | SRD 5.2.1, "Fighter Class Features", p. 47 | happy |
| FR-115 | At level 7 `choicesOnGaining` MUST include Additional Fighting Style when `subclass` is Champion (row C07a), and MUST NOT when no subclass is chosen (row C07b). | SRD 5.2.1, "Level 7: Additional Fighting Style", p. 49 | unhappy (rule interaction) |
| FR-031 | The "Recommended" Fighting Style feat (Defense) MUST be a data hint on the Fighting Style choice and MUST NOT restrict what may be chosen. | SRD 5.2.1, "Level 1: Fighting Style", p. 47 | unhappy (rule interaction) |
| FR-114 | The Proficiency Bonus of a multiclass character MUST come from total level, never Fighter level; `FighterLevelStats` MUST NOT expose it. Fighter 3 / Rogue 2 (total level 5) has +3, not the +2 of row L02 (asserted in derived-stats.md; FR-021 holds one row). | SRD 5.2.1, "Proficiency Bonus" table, p. 8; "Multiclassing", p. 25 | unhappy (rule interaction) |
| FR-149 | `FighterProgression` MUST be built once by the host from the decoded Fighter `ClassDefinition` and passed to `UseSecondWind`, `UseTacticalMind`, `UseIndomitable`, `UseActionSurge` and `ApplyFighterOptionChoice` as a constructor-injected collaborator. A use case MUST obtain a `FighterLevelStats` (`statsAt(level)`) and hand it to the domain services, which MUST take the stats as an argument and hold no progression or hidden state (`SecondWind.availability(stats, state)` gets the maximum through `stats`). A missing or malformed Fighter `ClassDefinition` is a host construction problem (`NewFighterProgression`, FR-160), not a use-case result. | n/a (package design); shared-types-and-ownership.md FR-023; `doc/rules/architecture.md` §3, §4 | happy |
| FR-150 | `FighterProgression` MUST read Second Wind uses and Weapon Mastery kinds from the `second_wind` and `weapon_mastery` columns by `ClassColumnId` (content-classes.md Table CC), and hold Action Surge, Indomitable and attack counts as rule constants. A test MUST assert the constants equal the counts annotated on the row features (Action Surge "one use" at 2, "two uses" at 17; Indomitable "one use", "two uses", "three uses" at 9, 13, 17) at every level; a mismatch is a test failure, not a run-time error. | SRD 5.2.1, "Fighter Features", p. 47; "Level 2: Action Surge", "Level 9: Indomitable", p. 48; content-classes.md Table CC | unhappy (rule interaction) |
| FR-160 | `NewFighterProgression(ClassDefinition)` MUST return `(FighterProgression, error)`, the error joining typed reasons (`notTheFighterDefinition`, `columnMissing`, `rowsIncomplete`); the constructor MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) for the same inputs. Use cases never see a missing Fighter definition. | n/a (package design); `doc/rules/architecture.md` §6 | unhappy (missing content) |

### 4.2 Fighting Style

Every Fighting Style feat is taken through the pure `FeatTakingRules.take` (via
`FightingStyleChoice.toTakeFeat()`, feats-behaviour.md FR-042); the held-feat check is its
`FeatNotRepeatable` rejection. `ApplyFighterOptionChoice` carries and returns the `CharacterFeats`
and never calls `TakeFeat` (shared-types-and-ownership.md FR-044). `FighterState.fightingStyle`
marks only the level 1 style. No bundled Fighting Style feat is Repeatable
(content-species-backgrounds-feats.md FR-039); that branch is tested through `FakeContentSource`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | At Fighter level 1 the character MUST be able to choose one Fighting Style feat as a `FightingStyleChoice`, taken with `FeatTakingRules.take` and held in `CharacterFeats`; `FighterState.fightingStyle` MUST record only the marker of the **level 1 style**, kept apart from the Additional Fighting Style (FR-158). | SRD 5.2.1, "Level 1: Fighting Style", p. 47 | happy |
| FR-034 | A chosen feat id the `ContentSource` cannot find MUST return `FighterOptionContentMissing` carrying `MissingContent` for it (shared-types-and-ownership.md FR-018), state unchanged. | SRD 5.2.1, "Parts of a Feat", p. 87 | unhappy (missing content) |
| FR-035 | A Fighter of level 1 or more MUST be treated as having the Fighting Style feature when evaluating a Fighting Style feat's prerequisite. | SRD 5.2.1, "Fighting Style Feats", pp. 87-88 | happy |
| FR-036 | Whenever the character gains a Fighter level, it MAY replace the level 1 style (`FighterState.fightingStyle`; never the Additional Fighting Style, FR-158) with a different Fighting Style feat of the same category, in `CharacterFeats`; the swap MUST be `FeatReplacement.replace` (feats-behaviour.md FR-082) with old = `state.fightingStyle` and a `FeatFromCategory(Fighting Style)` grant, which removes the old feat before the held-feat, grant and prerequisite checks; no local re-check of them (section 9). | SRD 5.2.1, "Level 1: Fighting Style", p. 47; "Parts of a Feat" (Repeatable), p. 87 | happy |
| FR-037 | A replacement MUST be rejected with `noLevelGained` unless the command carries the `LevelGained` event (character-aggregate.md) for class Fighter at the current Fighter level, and with `eventNotForCharacter` (FR-159) when that event names another character. | SRD 5.2.1, "Level 1: Fighting Style", p. 47 | unhappy (state error) |
| FR-038 | A replacement naming the feat it replaces MUST be rejected with `sameFeat`. | SRD 5.2.1, "Level 1: Fighting Style", p. 47 | unhappy (boundary) |
| FR-039 | At Champion level 7 (Fighter level 7, Champion) the character MUST be able to choose another Fighting Style feat, taken with `FeatTakingRules.take` into `CharacterFeats`; a feat already held is rejected under feats-behaviour.md FR-005 to FR-007. | SRD 5.2.1, "Level 7: Additional Fighting Style", p. 49; "Parts of a Feat" (Repeatable), p. 87 | happy |
| FR-040 | The Additional Fighting Style choice MUST be rejected with `featureNotGained` before Fighter level 7 or without the Champion subclass; the command carries the chosen `SubclassId` (or none). | SRD 5.2.1, "Level 7: Additional Fighting Style", p. 49 | unhappy (boundary) |
| FR-156 | A second `ChooseFightingStyle` when `FighterState.fightingStyle` is set (same or different feat) MUST be rejected with `fightingStyleAlreadyChosen`; changing the level 1 style is only the replacement of FR-036. The one exception to FR-121: repeatability is not consulted, so `featNotRepeatable` is never reported with it; `notAFightingStyleFeat` still is. | SRD 5.2.1, "Level 1: Fighting Style", p. 47 | unhappy (state error) |
| FR-157 | A `ReplaceFightingStyle` when `FighterState.fightingStyle` is not set MUST be rejected with `noStyleToReplace`. The command names no replaced feat: replacement always targets the level 1 marker (a new feat equal to the held Additional Fighting Style is `featNotRepeatable` unless Repeatable). Like FR-036 it uses `FeatReplacement.replace`. | SRD 5.2.1, "Level 1: Fighting Style", p. 47; "Level 7: Additional Fighting Style", p. 49 | unhappy (state error) |
| FR-158 | A `ChooseAdditionalFightingStyle` when `FighterState.additionalFightingStyle` is set MUST be rejected with `additionalStyleAlreadyChosen`. It is recorded in its own field and never replaceable (section 9). | SRD 5.2.1, "Level 7: Additional Fighting Style", p. 49 | unhappy (state error) |
| FR-159 | A `ReplaceFightingStyle` carrying a `LevelGained`, or a `SwapWeaponMastery` carrying a `RestApplied`, whose `characterId` differs from the character's MUST be rejected with `eventNotForCharacter`, checked before the event's level or rest kind. | n/a (package design); character-aggregate.md FR-085; rests-and-hit-dice.md FR-036 | unhappy (invalid input) |

### 4.3 Weapon Mastery and Tactical Master

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-046 | When the character finishes a Long Rest it MAY change exactly one chosen kind to a different valid kind. The command MUST carry the `RestApplied` event (rests-and-hit-dice.md FR-036) for a Long Rest and name the kind to remove and add; the use case maps the event's `benefitsOf` (`RestKind.longRest`) to `MasteryChangeMoment.longRest` and passes it as `moment` to `WeaponMasteryChoice.change` with `ReplaceOneKind` (weapon-properties-and-mastery.md FR-024, FR-025), which decides the swap; `wrongMoment` is reported as `noLongRestFinished` (FR-047). | SRD 5.2.1, "Level 1: Weapon Mastery", p. 48; "Long Rest", p. 185 | happy |
| FR-047 | A swap without a `RestApplied` for a Long Rest MUST be rejected with `noLongRestFinished` (the Fighter report of `wrongMoment`, weapon-properties-and-mastery.md FR-024). | SRD 5.2.1, "Level 1: Weapon Mastery", p. 48 | unhappy (state error) |
| FR-049 | From Fighter level 9, `FighterEffects.tacticalMasterOptions(level, access, weaponKind)` (`access` a `WeaponMasteryAccess`, weapon-properties-and-mastery.md FR-004) MUST, when `access.permits(weaponKind)` holds, return `TacticalMasterOffered` with exactly {Push, Sap, Slow} as replacements for that weapon's property for that attack, without changing the stored selection. The host passes its choice as `ReplacedMastery(m)` (weapon-properties-and-mastery.md FR-006), which decides the effect. Tactical Master is a base feature: a non-Champion Fighter has it. | SRD 5.2.1, "Level 9: Tactical Master", p. 48; "Mastery Properties", p. 90 | happy |
| FR-050 | `tacticalMasterOptions` MUST return `TacticalMasterUnavailable(featureNotGained)` below Fighter level 9. | SRD 5.2.1, "Level 9: Tactical Master", p. 48 | unhappy (boundary) |
| FR-125 | `tacticalMasterOptions` MUST return `TacticalMasterUnavailable(masteryNotUsable)` when `access.permits(weaponKind)` is false (kind not chosen, or `NoMasteryFeature`). The operation takes no requested property, so no `NotATacticalMasterProperty` rejection exists; the offered set is tested to be exactly {Push, Sap, Slow} (Topple is not in it). Check order: `featureNotGained` (FR-050), then `masteryNotUsable`. | SRD 5.2.1, "Level 9: Tactical Master", p. 48 | unhappy (state error) |
| FR-120 | A selection naming an item the `ContentSource` cannot find MUST return `FighterOptionContentMissing` carrying `MissingContent` for every such item id. | SRD 5.2.1, "Mastery Properties", p. 90 | unhappy (missing content) |
| FR-121 | All rule problems in one selection or swap MUST be reported together in one rejection, each carrying the offending ids, in the order of the section 5 reason table, with one exception: for `ChooseFightingStyle`, `fightingStyleAlreadyChosen` suppresses `featNotRepeatable` (FR-156). Precedence for `ApplyFighterOptionChoice` (also FR-129): (1) `notAFighter`, decided before any `ContentSource` call; (2) the lookups, and if any feat or item is missing `FighterOptionContentMissing` is returned for all missing ids and no rule reason, even `featureNotGained`; (3) the remaining reasons in the section 5 order. | n/a (package design); `doc/rules/architecture.md` §5 (every problem, not the first) | unhappy (invalid input) |
| FR-163 | `FighterChoiceRules` MUST decide the Fighting Style and Weapon Mastery rules by delegating to the owning specs and only map their outcomes to Fighter reasons: `NewWeaponMasteryChoice`, `extend` and `change` (weapon-properties-and-mastery.md FR-014 to FR-027) with `wrongCount` to `wrongMasteryCount`, `duplicateWeapon` to `duplicateMastery`, `weaponNotResolved` (found, not a weapon) to `notSimpleOrMartialWeapon`, `wrongMoment` to `noLongRestFinished`, and `dropsExistingMastery`, `notCurrentlyChosen`, `alreadyChosen` unchanged; and the feat checks of `FeatTakingRules.take` over the command's `CharacterFeats`, whose `FeatOutsideGrant` maps to `notAFightingStyleFeat` and `FeatNotRepeatable` to `featNotRepeatable`. It MUST NOT restate those rules and calls no other use case (shared-types-and-ownership.md FR-044). | n/a (package design); shared-types-and-ownership.md FR-044, FR-048 | happy |
| FR-164 | `FighterEffects.masteryAccess(classLevels, state)` MUST return `NoMasteryFeature` (weapon-properties-and-mastery.md FR-005) for a character with no Fighter levels or no `WeaponMasteryChoice` in `state`, and `MasteryFromChoice(choice)` otherwise. It is the `WeaponMasteryAccess` the host passes to an attack and to `tacticalMasterOptions` (FR-049). | n/a (package design); weapon-properties-and-mastery.md FR-004, FR-005; shared-types-and-ownership.md FR-048 | happy |

### 4.4 Second Wind and Tactical Shift

Table C, Second Wind healing (FR-053): d10 roll plus Fighter level; capping is out of scope.

| Row | Fighter level | Other classes | d10 roll | Hit Points to regain |
|---|---|---|---|---|
| SW1 | 1 | none | 1 | 2 |
| SW2 | 1 | none | 10 | 11 |
| SW3 | 5 | none | 6 | 11 |
| SW4 | 20 | none | 1 | 21 |
| SW5 | 20 | none | 10 | 30 |
| SW6 | 3 | Wizard 2 (total level 5) | 4 | 7 |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | `UseSecondWind` MUST succeed for Fighter level 1 or more with at least one unexpended use, returning `SecondWindApplied` with the new state and a `SecondWindUsed` event. | SRD 5.2.1, "Level 1: Second Wind", p. 48 | happy |
| FR-052 | The result MUST state the activation cost is a Bonus Action. The use case MUST NOT track or check the action economy. | SRD 5.2.1, "Level 1: Second Wind", p. 48; "Bonus Action", p. 177 | happy |
| FR-053 | Hit Points to regain MUST equal one `1d10` roll plus the Fighter level (Table C), computed by `SecondWind.resolve`. | SRD 5.2.1, "Level 1: Second Wind", p. 48 | happy |
| FR-054 | The level used MUST be the Fighter class level, not total level (row SW6). | SRD 5.2.1, "Level 1: Second Wind", p. 48 | unhappy (rule interaction) |
| FR-055 | A successful use MUST increase the expended count by exactly 1 and leave Action Surge and Indomitable counts untouched. | SRD 5.2.1, "Level 1: Second Wind", p. 48 | happy |
| FR-056 | With no unexpended use the result MUST be `SecondWindNotAvailable(noUsesRemaining)`, the character unchanged, and the `dice.Roller` MUST NOT be called. | SRD 5.2.1, "Level 1: Second Wind", p. 48 | unhappy (state error) |
| FR-057 | A character with no Fighter levels MUST get `SecondWindNotAvailable(notAFighter)`, unchanged, no roll (precedence FR-129). | SRD 5.2.1, "Fighter Class Features", p. 47 | unhappy (state error) |
| FR-058 | The result MUST carry the uncapped Hit Points to regain and MUST NOT read or change current Hit Points; the consumer caps at the maximum (excess is lost). | SRD 5.2.1, "Damage and Healing" ("Healing"), p. 17 | unhappy (boundary) |
| FR-059 | At Fighter level 5 or more the result MUST carry `TacticalShiftOffered` (closed `TacticalShiftOffer`, from `FighterEffects.tacticalShift`): a move up to half Speed that does not provoke Opportunity Attacks. Below level 5 it MUST carry `NoTacticalShift`. | SRD 5.2.1, "Level 5: Tactical Shift", p. 48; "Opportunity Attacks", p. 185 | unhappy (boundary) |
| FR-060 | The offer MUST carry the fraction one half, not a distance. `halfOf(speed)` MUST round down (30 gives 15, 25 gives 12, 0 gives 0). | SRD 5.2.1, "Level 5: Tactical Shift", p. 48; "Round Down", p. 187 | unhappy (boundary) |
| FR-061 | Tactical Shift MUST be offered only with Second Wind activated through `UseSecondWind`, never standalone. | SRD 5.2.1, "Level 5: Tactical Shift", p. 48 | unhappy (state error) |
| FR-062 | When Second Wind uses grow with level, the new uses MUST be unexpended: remaining equals new maximum minus the expended count (level 3 with 2 expended becomes 1 remaining at level 4). | SRD 5.2.1, "Level 1: Second Wind", p. 48 ("gain more uses") | unhappy (boundary) |
| FR-063 | A Short Rest MUST regain one expended Second Wind use (none if none expended). A Long Rest MUST regain all. | SRD 5.2.1, "Level 1: Second Wind", p. 48; "Short Rest", p. 187; "Long Rest", p. 185 | happy |

### 4.5 Tactical Mind

Table D, outcomes at DC 15 (FR-065 to FR-068).

| Row | Failed check total | d10 roll | New total | Result | Second Wind use |
|---|---|---|---|---|---|
| TM1 | 12 | 3 | 15 | `TacticalMindSucceeded` (equals the DC) | expended |
| TM2 | 12 | 2 | 14 | `TacticalMindStillFailed` | not expended |
| TM3 | 14 | 1 | 15 | `TacticalMindSucceeded` | expended |
| TM4 | 5 | 10 | 15 | `TacticalMindSucceeded` | expended |
| TM5 | 5 | 9 | 14 | `TacticalMindStillFailed` | not expended |
| TM6 | 15 | n/a | n/a | `TacticalMindNotAvailable(checkNotFailed)`, no roll | unchanged |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-064 | `UseTacticalMind` MUST be available only at Fighter level 2 or more; below it MUST return `TacticalMindNotAvailable(featureNotGained)`, no roll. | SRD 5.2.1, "Level 2: Tactical Mind", p. 48 | unhappy (boundary) |
| FR-065 | The command MUST carry the failed ability check as the d20-tests.md `CheckResult` (kind, `Roll`, `RollModifiers`, total, `DifficultyClass`, `D20TestOutcome`), not a Fighter-defined shape. A kind other than `D20TestKind.abilityCheck` MUST return `TacticalMindNotAvailable(notAnAbilityCheck)`. An outcome of `success` MUST return `TacticalMindNotAvailable(checkNotFailed)`, no roll, no use spent (row TM6). | SRD 5.2.1, "Level 2: Tactical Mind", p. 48; "D20 Tests", p. 6 | unhappy (boundary) |
| FR-066 | A use MUST require an unexpended Second Wind use; otherwise `TacticalMindNotAvailable(noUsesRemaining)`, no roll. | SRD 5.2.1, "Level 2: Tactical Mind", p. 48 | unhappy (state error) |
| FR-067 | When available the use case MUST roll `1d10` and return the roll and the new total, the failed `CheckResult` after `adjustedBy(rolled)` (d20-tests.md FR-065). `rolled` is validated by FR-162. | SRD 5.2.1, "Level 2: Tactical Mind", p. 48 | happy |
| FR-068 | If the new total meets or exceeds the DC the result MUST be `TacticalMindSucceeded` and one Second Wind use MUST be expended (TM1, TM3, TM4). | SRD 5.2.1, "Level 2: Tactical Mind", p. 48 | happy |
| FR-069 | If the new total is still below the DC the result MUST be `TacticalMindStillFailed`, carry the roll, and the use MUST NOT be expended (TM2, TM5). | SRD 5.2.1, "Level 2: Tactical Mind", p. 48 | unhappy (boundary) |
| FR-070 | Tactical Mind MUST NOT regain Hit Points and MUST NOT produce Tactical Shift. | SRD 5.2.1, "Level 2: Tactical Mind", p. 48 ("Rather than regaining Hit Points") | unhappy (rule interaction) |
| FR-128 | Tactical Mind adds `1d10` to the total; it does not reroll or replace the d20, so "Interactions with Rerolls" does not apply: only the total of the carried `CheckResult` is adjusted (the `Roll` is untouched), identically with Advantage or Disadvantage. | SRD 5.2.1, "Level 2: Tactical Mind", p. 48; "Interactions with Rerolls", p. 8 | unhappy (rule interaction) |

### 4.6 Action Surge

Table E, check order (FR-072); the first failing check is the reported reason; rejections leave the
character unchanged.

| Row | Fighter level | Turn context | Per-turn state | Expended uses | Result | Path |
|---|---|---|---|---|---|---|
| AS1 | 1 | `ownTurn` | `unused` | 0 | `featureNotGained` | unhappy (boundary) |
| AS2 | 2 | `otherTurn` | `used` | 1 | `notOwnTurn` (before `alreadyUsedThisTurn` and `noUsesRemaining`) | unhappy (rule interaction) |
| AS3 | 17 | `ownTurn` | `used` | 0 | `alreadyUsedThisTurn` | unhappy (state error) |
| AS4 | 17 | `ownTurn` | `used` | 2 | `alreadyUsedThisTurn` (before `noUsesRemaining`) | unhappy (rule interaction) |
| AS5 | 2 | `ownTurn` | `unused` | 1 | `noUsesRemaining` | unhappy (state error) |
| AS6 | 2 | `ownTurn` | `unused` | 0 | applied | happy |
| AS7 | 17 | `ownTurn` | `unused` | 1 | applied (second use on a later turn) | happy |
| AS8 | no Fighter levels | `ownTurn` | `unused` | n/a | `notAFighter` (before `featureNotGained`, FR-129) | unhappy (state error) |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | `UseActionSurge` MUST return `ActionSurgeApplied` for a character with the feature, on its own turn, with an unexpended use and no earlier use this turn. The result MUST describe one additional action that is not the Magic action. | SRD 5.2.1, "Level 2: Action Surge", p. 48 | happy |
| FR-072 | Checks MUST be made in this order, the first failing one reported: `notAFighter` (FR-129), `featureNotGained` (level below 2), `notOwnTurn`, `alreadyUsedThisTurn`, `noUsesRemaining` (Table E). A failing check MUST leave the character unchanged. | SRD 5.2.1, "Level 2: Action Surge", p. 48 | unhappy (rule interaction) |
| FR-073 | The command MUST carry the turn context as the enum `FighterTurnOwnership` (`ownTurn`, `otherTurn`) and the per-turn state as `FighterPerTurnUse` (`unused`, `used`), not booleans (FR-154). | SRD 5.2.1, "Level 2: Action Surge", p. 48 ("On your turn"; "only once on a turn") | happy |
| FR-074 | A Short Rest or a Long Rest MUST regain all expended Action Surge uses. | SRD 5.2.1, "Level 2: Action Surge", p. 48 | happy |
| FR-075 | At level 17 two uses MUST be available before a rest but only one per turn: a second use in the same turn is `alreadyUsedThisTurn`; on a later turn it succeeds (AS3, AS7). | SRD 5.2.1, "Level 2: Action Surge", p. 48 | unhappy (boundary) |
| FR-076 | When Fighter level goes from 16 to 17 the maximum MUST rise from 1 to 2; an expended use stays expended. | SRD 5.2.1, "Fighter Features", p. 47 | unhappy (boundary) |

### 4.7 Indomitable

The failed save is the d20-tests.md `CheckResult` of kind `D20TestKind.savingThrow`
(shared-types-and-ownership.md FR-002, FR-003); the caller names the die replaced (`replacedDie`, an
index into the dice rolled). The new result is `CheckResult.resolve(savingThrow,
roll.replaceDie(replacedDie, rolled), modifiers, target)` then `adjustedBy(fighterLevel)`:
recomputed used die plus the `RollModifiers` total plus Fighter level.

Table F: Fighter level 9, save modifier total +4, DC 15 (FR-081). IN1 to IN3 have one d20 (original
8, total 12, `replacedDie` 0); IN4 to IN7 apply "Interactions with Rerolls".

| Row | Mode | Original dice | Replaced die (index) | New d20 | Dice after | Used die | New total | Outcome | Use |
|---|---|---|---|---|---|---|---|---|---|
| IN1 | neither | [8] | 0 | 1 | [1] | 1 | 14 | `IndomitableStillFailed` | expended |
| IN2 | neither | [8] | 0 | 2 | [2] | 2 | 15 | `IndomitableSucceeded` (equals the DC) | expended |
| IN3 | neither | [8] | 0 | 20 | [20] | 20 | 33 | `IndomitableSucceeded` | expended |
| IN4 | Advantage | [3, 8] (used 8, total 12) | 1 (the used die) | 2 | [3, 2] | 3 | 16 | `IndomitableSucceeded` | expended |
| IN5 | Advantage | [3, 8] | 0 (the other die) | 20 | [20, 8] | 20 | 33 | `IndomitableSucceeded` | expended |
| IN6 | Disadvantage | [1, 12] (used 1, total 5) | 1 (the discarded die) | 15 | [1, 15] | 1 | 14 | `IndomitableStillFailed` (wrong die gains nothing) | expended |
| IN7 | Disadvantage | [1, 12] | 0 (the used die) | 10 | [10, 12] | 10 | 23 | `IndomitableSucceeded` | expended |
| IN8 | Advantage | [3, 8] | 2 | n/a | n/a | n/a | n/a | `IndomitableNotAvailable(dieNotRolled)`, no roll | unchanged |

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-077 | `UseIndomitable` MUST be unavailable below Fighter level 9 (`IndomitableNotAvailable(featureNotGained)`), no roll. | SRD 5.2.1, "Level 9: Indomitable", p. 48 | unhappy (boundary) |
| FR-078 | The command MUST carry the failed save as the `CheckResult`. A kind other than `savingThrow` MUST return `IndomitableNotAvailable(notASavingThrow)`; an outcome of `success` MUST return `IndomitableNotAvailable(saveNotFailed)`, no roll, no use spent. | SRD 5.2.1, "Level 9: Indomitable", p. 48; "D20 Tests", p. 6 | unhappy (boundary) |
| FR-079 | A use MUST require an unexpended use (maximum per FR-025); otherwise `IndomitableNotAvailable(noUsesRemaining)`, no roll. | SRD 5.2.1, "Level 9: Indomitable", p. 48 | unhappy (state error) |
| FR-080 | When available the use case MUST roll one `1d20` and replace exactly the die the caller named (FR-126); the result MUST NOT offer the original roll as an alternative ("You must use the new roll"). | SRD 5.2.1, "Level 9: Indomitable", p. 48; "Interactions with Rerolls", p. 8 | happy |
| FR-081 | The new total MUST equal the used die of the recomputed `Roll`, plus the failed save's modifiers, plus the Fighter level (not total level; Table F). | SRD 5.2.1, "Level 9: Indomitable", p. 48 | happy |
| FR-082 | One use MUST be expended whether the reroll succeeds or not (IN1, IN2). | SRD 5.2.1, "Level 9: Indomitable", p. 48 | unhappy (boundary) |
| FR-083 | A Long Rest MUST regain all expended Indomitable uses; a Short Rest MUST NOT regain any. | SRD 5.2.1, "Level 9: Indomitable", p. 48; "Long Rest", p. 185 | unhappy (state error) |
| FR-126 | With Advantage or Disadvantage the caller MUST name the die replaced and only it is replaced; the other keeps its value and the used die is recomputed by the mode (IN4 to IN7). Without either there is one die and it is the one replaced (IN1 to IN3). | SRD 5.2.1, "Interactions with Rerolls", p. 8; "Level 9: Indomitable", p. 48 | unhappy (rule interaction) |
| FR-127 | A `replacedDie` that does not index a die of the failed roll MUST return `IndomitableNotAvailable(dieNotRolled)`, no roll, no use spent (IN8). | SRD 5.2.1, "Interactions with Rerolls", p. 8 | unhappy (invalid input) |

### 4.8 Studied Attacks and other declared effects

Pure domain queries over the character; `combat` and derived-stats.md apply the effects.
`FighterEffects` reports base Fighter features, `ChampionEffects` the Champion's.

Table G, Champion critical-hit threshold (FR-087, FR-130 to FR-132); the default is a natural 20
only. `ChampionAttackKind`: `weapon`, `unarmedStrike`, `other`.

| Row | Subclass | Fighter level | Attack kind | Critical Hit range |
|---|---|---|---|---|
| CR1 | Champion | 2 | weapon | panic (FR-132) |
| CR2 | Champion | 3 | weapon | 19 or 20 |
| CR3 | Champion | 3 | Unarmed Strike | 19 or 20 |
| CR4 | Champion | 14 | weapon | 19 or 20 |
| CR5 | Champion | 15 | weapon | 18 to 20 |
| CR6 | Champion | 20 | Unarmed Strike | 18 to 20 |
| CR7 | none chosen | 3 | weapon | 20 |
| CR8 | none chosen | 15 | weapon | 20 |
| CR9 | Champion | 15 | other attack roll | 20 |

Table H, Defy Death (FR-095), Death Saving Throw d20:

| Row | Fighter level | Subclass | Natural roll | Roll counts as |
|---|---|---|---|---|
| DD1 | 18 | Champion | 17 | 17 |
| DD2 | 18 | Champion | 18 | 20 |
| DD3 | 18 | Champion | 19 | 20 |
| DD4 | 18 | Champion | 20 | 20 |
| DD5 | 17 | Champion | 18 | 18 |
| DD6 | 20 | none chosen | 18 | 18 |

Table I, Heroic Rally (FR-097 to FR-099, FR-133): healing is 5 plus the Constitution
`AbilityModifier` (-5 to +10, SRD 5.2.1, p. 6).

| Row | Con modifier | Current HP | HP maximum | Bloodied | Result |
|---|---|---|---|---|---|
| HR1 | +2 | 10 | 20 | yes | `HeroicRallyRegains` 7 |
| HR2 | +2 | 11 | 20 | no | `HeroicRallyNotEligible(notBloodied)` |
| HR3 | -5 | 5 | 20 | yes | `HeroicRallyRegains` 0 |
| HR4 | +2 | 0 | 20 | n/a | `HeroicRallyNotEligible(noHitPoints)` |
| HR5 | +2 | 1 | 1 | no (1 is more than half of 1) | `HeroicRallyNotEligible(notBloodied)` |
| HR6 | +2 | 1 | 2 | yes | `HeroicRallyRegains` 7 |
| HR7 | +10 | 10 | 20 | yes | `HeroicRallyRegains` 15 |
| HR8 | +2 | 21 | 20 | n/a | panic (FR-133) |

A maximum below 1 is not representable as a `HitPointMaximum`, so no row exists for it (FR-133).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-084 | Studied Attacks MUST be reported by `FighterEffects` from Fighter level 13 for every Fighter whatever the subclass: after an attack roll against a creature misses, Advantage on the next attack roll against that creature before the end of the character's next turn. Not at level 12. | SRD 5.2.1, "Level 13: Studied Attacks", p. 48 | unhappy (boundary) |
| FR-085 | The effect MUST be data (trigger: missed attack roll; applies to: next attack roll against the same creature; expires: end of the character's next turn) and hold no encounter state. | SRD 5.2.1, "Level 13: Studied Attacks", p. 48 | happy |
| FR-086 | Extra Attack, Two Extra Attacks and Three Extra Attacks MUST be reported by `FighterEffects` as the attack count (FR-026), not as separate stacking effects. | SRD 5.2.1, "Level 5: Extra Attack", p. 48 | happy |
| FR-087 | `ChampionEffects.criticalHitThreshold(subclass, level, attackKind)` MUST return 19 with Champion and Fighter level 3 to 14 for weapon attack rolls and Unarmed Strikes (CR2, CR3, CR4). | SRD 5.2.1, "Level 3: Improved Critical", p. 49; "Critical Hit", p. 179 | unhappy (boundary) |
| FR-130 | With Champion and Fighter level 15 or more it MUST return 18 for weapon attack rolls and Unarmed Strikes (CR5, CR6). | SRD 5.2.1, "Level 15: Superior Critical", p. 49; "Critical Hit", p. 179 | unhappy (boundary) |
| FR-131 | In every other consistent case it MUST return 20 (the default). | SRD 5.2.1, "Critical Hit", p. 179 | happy |
| FR-132 | A Champion `subclass` with a Fighter `level` below 3 MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) (unreachable: the subclass is chosen at level 3; CR1). | n/a (package design); `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-088 | A Fighter without Champion MUST get 20 at every level (CR7, CR8). | SRD 5.2.1, "Fighter Subclass: Champion", p. 49 | unhappy (state error) |
| FR-089 | An attack roll that is neither a weapon attack roll nor an Unarmed Strike MUST get 20 (CR9). | SRD 5.2.1, "Level 3: Improved Critical", p. 49 | unhappy (boundary) |
| FR-090 | `criticalHitThreshold` MUST only return the threshold (20, 19 or 18); it takes no roll and decides no Critical Hit. A threshold never excludes a natural 20 and never involves a natural 1. The host passes the returned value as the optional critical-range input of the attack roll (d20-tests.md; section 9), where a natural 20 always remains a Critical Hit. | SRD 5.2.1, "Attack Rolls" ("Rolling 20 or 1"), p. 7; "Level 3: Improved Critical", p. 49 | unhappy (boundary) |
| FR-091 | At Champion level 3 or more the character MUST be reported as having Advantage on Initiative rolls and Strength (Athletics) checks, as two declared `RollMode.advantage` sources of the closed `ChampionAdvantageSource` (`ChampionInitiativeAdvantage`; `ChampionSkillCheckAdvantage` carrying `Skill` Athletics and `Ability` Strength; FR-154). Below level 3 or without Champion, none. | SRD 5.2.1, "Level 3: Remarkable Athlete", p. 49; "Initiative", p. 184 | unhappy (boundary) |
| FR-092 | At Champion level 3 or more, immediately after a Critical Hit, the character MUST be offered a move up to half its Speed that does not provoke Opportunity Attacks (one half, rounded down, FR-060). | SRD 5.2.1, "Level 3: Remarkable Athlete", p. 49; "Opportunity Attacks", p. 185 | happy |
| FR-093 | `HeroicWarrior.evaluate(subclass, level, inspiration, presence)` (`inspiration` the `HeroicInspiration`; `presence` the enum `FighterCombatPresence`, `inCombat` / `notInCombat`, FR-154) MUST return `HeroicWarriorGive` at Champion level 10 or more when `NotInspired` and `inCombat`. | SRD 5.2.1, "Level 10: Heroic Warrior", p. 49; "Heroic Inspiration", p. 8 | happy |
| FR-094 | Heroic Warrior MUST apply only "during combat"; with `notInCombat` it MUST return `HeroicWarriorNotAvailable(notInCombat)`. | SRD 5.2.1, "Level 10: Heroic Warrior", p. 49 | unhappy (state error) |
| FR-135 | `HeroicWarrior.evaluate` MUST return `HeroicWarriorNotAvailable(alreadyHasInspiration)` when `Inspired` (only one at a time; the gain, including giving an excess one away, is heroic-inspiration.md FR-003 to FR-007, which the host calls on `HeroicWarriorGive`; shared-types-and-ownership.md FR-048) and `HeroicWarriorNotAvailable(featureNotGained)` below level 10 or without Champion. Check order: `featureNotGained`, `notInCombat`, `alreadyHasInspiration`. The start-of-turn trigger is effect data (`startOfEachOfYourTurns`) applied by the consumer; the state is owned by character-aggregate.md (shared-types-and-ownership.md FR-010). | SRD 5.2.1, "Level 10: Heroic Warrior", p. 49; "Heroic Inspiration", p. 8 | unhappy (boundary) |
| FR-095 | At Champion level 18 or more, Death Saving Throws MUST have Advantage and a natural 18, 19 or 20 MUST count as 20 (Table H); below 18 or without Champion neither applies. | SRD 5.2.1, "Level 18: Survivor" (Defy Death), p. 49; "Death Saving Throws", pp. 17-18 | unhappy (boundary) |
| FR-096 | A natural roll counting as 20 under Defy Death MUST give the benefit of a 20: the character regains 1 Hit Point. The result carries that fact and the roll counted; it does not change current Hit Points. | SRD 5.2.1, "Level 18: Survivor" (Defy Death), p. 49; "Death Saving Throws" ("Rolling a 1 or 20"), p. 18 | happy |
| FR-097 | `HeroicRally.healing(conModifier: AbilityModifier, currentHp: int, hitPointMaximum: HitPointMaximum)` MUST return `HeroicRallyRegains` with 5 plus the Constitution modifier when the character is Bloodied and has at least 1 Hit Point (HR1, HR3, HR6, HR7). | SRD 5.2.1, "Level 18: Survivor" (Heroic Rally), p. 49 | happy |
| FR-134 | The Heroic Rally effect MUST carry the trigger `startOfEachOfYourTurns` as data. Firing it is the consumer's (`combat`); `HeroicRally.healing` is the per-trigger calculation and holds no turn state. | SRD 5.2.1, "Level 18: Survivor" (Heroic Rally), p. 49 | happy |
| FR-098 | Bloodied MUST be decided by the one shared predicate `isBloodied(currentHitPoints, HitPointMaximum)` (derived-stats.md FR-071; shared-types-and-ownership.md FR-049). This spec defines and exports none; class-cleric.md FR-106 references the same owner. HR2 and HR5 return `HeroicRallyNotEligible(notBloodied)`. | SRD 5.2.1, "Bloodied", p. 177 | unhappy (boundary) |
| FR-099 | A character at 0 Hit Points MUST return `HeroicRallyNotEligible(noHitPoints)` (HR4). | SRD 5.2.1, "Level 18: Survivor" (Heroic Rally), p. 49 | unhappy (boundary) |
| FR-133 | `HeroicRally.healing` MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error) for a negative `currentHp` or a `currentHp` above `hitPointMaximum` (HR8). `HitPointMaximum` is a validated type, so a maximum below 1 is ruled out by the type (asserted in derived-stats.md). Only `currentHp` stays a plain `int` (current Hit Points live in `combat`). Every `AbilityModifier` is valid; -5 gives healing 0 (HR3). | n/a (package design); `doc/rules/architecture.md` §6; SRD 5.2.1, "Ability Modifiers", p. 6 | unhappy (invalid input) |
| FR-100 | Heroic Rally and Defy Death MUST NOT be reported below Fighter level 18. | SRD 5.2.1, "Level 18: Survivor", p. 49 | unhappy (boundary) |
| FR-137 | Heroic Rally and Defy Death MUST NOT be reported without Champion, whatever the level. | SRD 5.2.1, "Fighter Subclass: Champion", p. 49 | unhappy (state error) |
| FR-105 | Subclass features reported for a character MUST be those of the chosen subclass's `SubclassDefinition` (content-classes.md FR-075, `featuresAtOrBelow`, used not reimplemented) at or below the character's Fighter level: Champion gives Improved Critical and Remarkable Athlete at 3, Additional Fighting Style at 7, Heroic Warrior at 10, Superior Critical at 15, Survivor at 18. The chosen subclass is an input (`SubclassId` or none), not Fighter state. | SRD 5.2.1, "Fighter Subclass: Champion", p. 49; "Level 3: Fighter Subclass", p. 48 | happy |

### 4.9 Cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-107 | Every use case MUST be deterministic: same command and fixed rolls give the same result, including roll extremes (d10 1 and 10, d20 1 and 20). | n/a (package design); `doc/rules/testing.md` §4, `doc/rules/architecture.md` §3 | unhappy (determinism) |
| FR-108 | No use case or domain function MAY mutate its input; each returns a new `Character` and its events. A rejected command returns no events and a character equal to the input. | n/a (package design); `doc/rules/architecture.md` §3, `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-109 | Every collection on a result or state (features, choices, mastery kinds, options) MUST be a defensive copy. | n/a (package design); `doc/rules/architecture.md` §4, `doc/rules/testing.md` §4 | unhappy (immutability) |
| FR-110 | A `dice.Roller` that returns its typed `*srderr.PortError` MUST propagate from the use case, character unchanged. | n/a (package design); `doc/rules/architecture.md` §4, §6 | unhappy (collaborator failure) |
| FR-111 | A `ContentSource` that returns its typed `*srderr.PortError` MUST propagate from `ApplyFighterOptionChoice`, character unchanged. | n/a (package design); `doc/rules/architecture.md` §4, §6 | unhappy (collaborator failure) |
| FR-112 | A result MUST be identical with the no-op `telemetry.Telemetry` and with a recording one. | n/a (package design); `doc/rules/observability.md` §1, §9 | happy |
| FR-113 | A command naming a character with no Fighter levels MUST return `notAFighter` for every use case, including `ApplyFighterOptionChoice` naming a feat or item the source could not find (no lookup is made, so never `FighterOptionContentMissing`). | SRD 5.2.1, "Fighter Class Features" ("As a Fighter"), p. 47 | unhappy (state error) |
| FR-129 | Reason precedence for every use case: `notAFighter` first (from `ClassLevels`, before any roll or lookup), then `featureNotGained`, then the rest in the section 5 order. For `ApplyFighterOptionChoice` the lookups sit between `notAFighter` and every other reason (FR-121). A character with no Fighter levels always gets `notAFighter`. | n/a (package design); `doc/rules/architecture.md` §6 | unhappy (rule interaction) |
| FR-144 | `FighterCreationChoices` MUST implement both pure methods of the aggregate's `ClassCreationChoices` (character-aggregate.md FR-094) so it travels in `CreateCharacter`: (1) seeding, taking a `FeatureStates` and returning it with `FighterState.initial()` under the `'fighter'` key, other keys unchanged; (2) the `ContentKey`s the choices name, an empty unmodifiable list (skills and the Starting Equipment letter belong to character-creation.md FR-015 to FR-020). It holds no field and has no the `New` constructor (returns an error). | n/a (package design); shared-types-and-ownership.md FR-024; character-creation.md FR-102, FR-103 | happy |
| FR-145 | Fighter state MUST be a `FighterState` implementing `FeatureState`, stored under one typed `FeatureStateKey[FighterState]` named `'fighter'` (`[a-z][a-z0-9_]*`, character-aggregate.md FR-093) defined here. `FighterState` MUST compare by value (`==`, `hashCode` over counts, styles, masteries) and its collections be unmodifiable; it is constructible in unit tests with no aggregate. | n/a (package design); shared-types-and-ownership.md FR-022 | happy |
| FR-146 | Every Fighter feature function MUST be pure from (character state, inputs) to (new state, events), with dice and content supplied through the use case only. Level-derived numbers arrive as a `FighterLevelStats` built by the injected `FighterProgression` (FR-149); no domain function reads content or hidden state. | n/a (package design); shared-types-and-ownership.md FR-023 | happy |
| FR-147 | This spec MUST define none of `SubclassChosen`, `SubclassRejected`, `FightingStyleChoice`, `WeaponMasteryChoice`, `WeaponMasteryAccess`, `CharacterFeats`, `FeatTakingRules`, `RestApplied`, `MissingContent`, `HeroicInspiration`, `Roll`, `ProficiencyBonus`, `ClassLevels`. Fighter events are past-tense value objects of the closed `FighterEvent` family (FR-155), defined only here. | n/a (package design); shared-types-and-ownership.md FR-019, FR-020 | unhappy (duplicate) |
| FR-148 | `FighterState.afterRest(kind)` MUST be a pure function of `RestKind` applying FR-063, FR-074 and FR-083 (Short Rest: one Second Wind use and all Action Surge uses; Long Rest: all uses of all three). The host, or orchestration above both specs, applies it when a `RestApplied` arrives (passing `RestApplied.benefitsOf` and checking `characterId`); it is not a use case and rests-and-hit-dice.md MUST NOT name or call `FighterState` (shared-types-and-ownership.md FR-016). | n/a (package design); shared-types-and-ownership.md FR-016 | happy |
| FR-151 | `FighterState.initial()` MUST be the empty state: all expended counts 0, no level 1 style marker, no Additional Fighting Style marker, no mastery choice. `FighterCreationChoices` MUST seed exactly this under `'fighter'` (FR-144). A multiclass first Fighter level needs no seeding by another spec: an absent state is initial (FR-152; section 9). | n/a (package design); character-aggregate.md FR-094; shared-types-and-ownership.md FR-022, FR-024 | happy |
| FR-152 | When `FeatureStates.read` returns nothing for `'fighter'` for a character with Fighter levels, every use case and domain function MUST treat it as `FighterState.initial()` (not `notAFighter`, decided from `ClassLevels` alone, FR-129) and on success write the state under the key. A character with no Fighter levels is `notAFighter` whatever the store holds. | n/a (package design); character-aggregate.md FR-092, FR-093 | unhappy (state error) |
| FR-153 | The `FighterState` constructor MUST enforce only level-free invariants (every expended count 0 or more; a negative count is panic). A held `WeaponMasteryChoice` is distinct by its own type (weapon-properties-and-mastery.md FR-014). Level-dependent bounds (expended at most the maximum, FR-062 and FR-076; masteries at most the Table A count) are enforced by the domain functions: remaining uses are maximum minus expended, never below 0, and `FighterChoiceRules` rejects a selection over the count (weapon-properties-and-mastery.md FR-015 via FR-163). | n/a (package design); `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |
| FR-154 | Input types, Fighter- or Champion-prefixed so they cannot collide with `AttackKind` of class-druid.md (attacks.md uses `AttackForm`): `ChampionAttackKind` (`weapon`, `unarmedStrike`, `other`), `FighterTurnOwnership` (`ownTurn`, `otherTurn`), `FighterPerTurnUse` (`unused`, `used`), `FighterCombatPresence` (`inCombat`, `notInCombat`), and the closed `ChampionAdvantageSource` (`ChampionInitiativeAdvantage`, `ChampionSkillCheckAdvantage(skill)`), published from the `character` package API with one definition, here. A caller maps its own attack kind to `ChampionAttackKind`. | n/a (package design); shared-types-and-ownership.md FR-013, FR-017, FR-021; `doc/rules/architecture.md` §3, §10 | unhappy (name collision) |
| FR-155 | Fighter events MUST be the closed `FighterEvent` family implementing the open `CharacterEvent` (character-aggregate.md FR-085, so `DomainEvent`), not variants of the closed `CharacterRecordEvent` (shared-types-and-ownership.md FR-018). Every event MUST carry the `CharacterId`. | n/a (package design); character-aggregate.md FR-085; shared-types-and-ownership.md FR-019 | happy |
| FR-161 | Domain-service return types MUST be named, closed or value types with Fighter-prefixed variants, none generic: `SecondWindAvailability` (`SecondWindAvailable`, `SecondWindUnavailable(reason)`), `SecondWindResolution`, `TacticalMindAvailability` (`TacticalMindAvailable`, `TacticalMindUnavailable(reason)`), `FighterChoice` (`FighterChoiceFightingStyle`, `FighterChoiceWeaponMasteries(count)`, `FighterChoiceAdditionalFightingStyle`), `TacticalMindResolution` (`TacticalMindResolutionSucceeded`, `TacticalMindResolutionStillFailed`), `IndomitableAvailability` (`IndomitableAvailable`, `IndomitableUnavailable(reason)`), `IndomitableResolution` (`IndomitableResolutionSucceeded`, `IndomitableResolutionStillFailed`), `ActionSurgeEvaluation` (`ActionSurgeGranted`, `ActionSurgeRefused(reason)`), `FighterChoiceVerdict` (`FighterChoiceAccepted`, `FighterChoiceRefused`). The package API scan (section 8) MUST find no `Available`, `Unavailable` or other generic variant name. | n/a (package design); shared-types-and-ownership.md FR-017, FR-021 | unhappy (name collision) |
| FR-162 | `rolled` MUST be an `int` validated at the domain function: 1 to 10 for `SecondWind.resolve` and `TacticalMind.resolve`, 1 to 20 for `Indomitable.resolve`; outside the range MUST panic with an error value (programmer error; `MustX` helper, with `NewX` returning an error). The use case obtains it from one `dice.Roller` call (`1d10` or `1d20`); the domain never rolls. | n/a (package design); `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |

### 4.10 telemetry.Telemetry requirements

Signals are listed in section 7; the shared `srd5e.character.feature_used` counter, `.duration` and
`.errors` are owned by character-aggregate.md (shared-types-and-ownership.md FR-042, FR-063) and
this spec defines none.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-138 | Each use case MUST run inside one span named as in section 7 (`srd5e.character.use_second_wind`, `use_tactical_mind`, `use_indomitable`, `use_action_surge`, `apply_fighter_option`), with attributes `srd5e.class` (`fighter`), `srd5e.feature` and `srd5e.outcome` (`success` or `failure`, shared-types-and-ownership.md FR-041), and `srd5e.character.rejection` on rejection and content-missing results only. The four synchronous use cases MUST use `Tracer.trace`; `ApplyFighterOptionChoice` returns a and MUST use `Tracer.Start` with the caller's `context.Context`, so the span covers the lookups and ends with error status when one returns an error. | n/a (package design); `doc/rules/observability.md` §4, §5, §6, §9; `doc/rules/architecture.md` §4; shared-types-and-ownership.md FR-041, FR-042 | happy |
| FR-139 | Each call MUST increment the shared counter `srd5e.character.feature_used` once with `srd5e.class`, `srd5e.feature` and `srd5e.outcome`, plus `srd5e.character.rejection` on a rejection or content-missing result. No per-feature metric exists. A returned port failure also increments it once, with `srd5e.outcome` `failure` and no `error.type` or rejection (character-aggregate.md FR-118; FR-141). | n/a (package design); `doc/rules/observability.md` §5, §9; shared-types-and-ownership.md FR-042 | happy |
| FR-140 | The duration of each call MUST be recorded in the shared `srd5e.character.feature_used.duration` histogram through the `Meter` (seconds; `srd5e.class`, `srd5e.feature`): `telemetry.Timer` for the four synchronous use cases and for `ApplyFighterOptionChoice`, recorded after the lookups finish and also when one returns an error. The use case MUST NOT read the clock. | n/a (package design); `doc/rules/observability.md` §4, §5 | happy |
| FR-141 | A `dice.Roller` or `ContentSource` failure (a returned `*srderr.PortError`) MUST set the span error status, record the error, increment `srd5e.character.feature_used` once (FR-139) and the shared `srd5e.character.feature_used.errors` counter exactly once, with `srd5e.class`, `srd5e.feature` and `error.type` set to `srderr.ErrorType(err)` (errors-and-results.md; owner character-aggregate.md FR-118), and MUST NOT be logged; `feature_used` MUST NOT carry `error.type`. A `FighterOptionContentMissing` result MUST set the span error status, increment `feature_used` once with `srd5e.outcome` `failure` and `srd5e.character.rejection` `content_missing`, carry no `error.type`, and MUST NOT be logged. | n/a (package design); `doc/rules/observability.md` §6, §7, §9; shared-types-and-ownership.md FR-041 | unhappy (collaborator failure) |
| FR-142 | A rule rejection MUST NOT set error status and MUST NOT log above `FINE`. A call ending in a success, rule-rejection or `...StillFailed` result logs exactly one `FINE` event `class feature resolved` with constant message and lazily supplied attributes. `TacticalMindStillFailed` and `IndomitableStillFailed` are normal rule outcomes: no error status, one FINE log, `srd5e.outcome` `failure`, no `srd5e.character.rejection`. A returned port failure and a `FighterOptionContentMissing` log nothing (FR-141). | n/a (package design); `doc/rules/observability.md` §7 | unhappy (state error) |
| FR-143 | When one rejection carries several reasons (`FighterOptionRejected`), the `srd5e.character.rejection` attribute MUST be the first reason in the section 5 order for the chosen option, one finite value. | n/a (package design); `doc/rules/observability.md` §5 | unhappy (rule interaction) |

### 4.11 Withdrawn ids

Moved to the owner shown; Fighter reason names for moved rules are mapped by FR-163. FR-017 left no
`notAllowedForMulticlass` reason: `FighterCreationChoices` takes no skill, equipment or entry input.

| Id | Withdrawn |
|---|---|
| FR-001 | Withdrawn: moved to content-classes.md (FR-017: Hit Point Die D10) |
| FR-002 | Withdrawn: moved to content-classes.md (FR-011 to FR-016: Primary Ability `AnyOf` Strength, Dexterity) |
| FR-003 | Withdrawn: moved to content-classes.md (FR-018: saving throws Strength, Constitution) |
| FR-004 | Withdrawn: moved to content-classes.md (FR-019, FR-020: skills, choose 2 of 9); validated by character-creation.md (FR-015 to FR-018) |
| FR-005 | Withdrawn: moved to content-classes.md (FR-021: Simple and Martial weapons) |
| FR-006 | Withdrawn: moved to content-classes.md (FR-022: Armor Training Light, Medium, Heavy, Shields) |
| FR-007 | Withdrawn: moved to content-classes.md (FR-024, FR-025: three Starting Equipment options A, B, C); letter validated by character-creation.md (FR-020) |
| FR-008 | Withdrawn: moved to content-classes.md (FR-029: no `spellcasting` trait) |
| FR-009 | Withdrawn: moved to content-classes.md (FR-076: exactly one subclass, Champion) |
| FR-010 | Withdrawn: moved to character-creation.md (FR-015: the class's `picks` distinct skills) |
| FR-011 | Withdrawn: moved to character-creation.md (FR-016: `classSkillCountWrong`) |
| FR-012 | Withdrawn: moved to character-creation.md (FR-017: `classSkillNotOffered`) |
| FR-013 | Withdrawn: moved to character-creation.md (FR-018: `classSkillRepeated`) |
| FR-014 | Withdrawn: moved to character-creation.md (FR-020: an option selects exactly its items and coin) |
| FR-015 | Withdrawn: moved to character-creation.md (FR-104: level 1 grants are the content-classes.md level 1 `ProgressionRow`; `FighterProgression` is not its dependency) |
| FR-016 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-108: multiclass first Fighter level grants) |
| FR-017 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-108, FR-110) and character-creation.md (FR-112) |
| FR-018 | Withdrawn: moved to content-classes.md (FR-048: `entriesGainedAt`, `rowFeatures`) |
| FR-019 | Withdrawn: moved to content-classes.md (FR-048, FR-049: features per level, ASI at 4, 6, 8, 12, 14, 16) |
| FR-029 | Withdrawn: moved to content-classes.md (FR-049) and level-advancement-and-multiclassing.md (FR-016) |
| FR-030 | Withdrawn: moved to content-classes.md (FR-049) and level-advancement-and-multiclassing.md (FR-016: Epic Boon at 19) |
| FR-033 | Withdrawn: moved to feats-behaviour.md (FR-043: `FeatOutsideGrant`, reported as `notAFightingStyleFeat`) |
| FR-041 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-014, FR-018: `allowedCount` distinct kinds) |
| FR-042 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-015: `wrongCount`, reported as `wrongMasteryCount`) |
| FR-043 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-016: `duplicateWeapon`, reported as `duplicateMastery`) |
| FR-044 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-018, FR-020: `weaponNotResolved`, reported as `notSimpleOrMartialWeapon`) |
| FR-045 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-022, FR-023: `dropsExistingMastery`) |
| FR-048 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-004, FR-005: `WeaponMasteryAccess.permits`; supplied by FR-164 here) |
| FR-101 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-041, FR-044: subclass chosen at level 3) |
| FR-102 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-044: `belowSubclassLevel`) |
| FR-103 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-045: second choice `alreadyChosen`) |
| FR-104 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-046: `unknownSubclass`) |
| FR-106 | Withdrawn: moved to content-classes.md (FR-077: table levels equal Champion feature levels 7, 10, 15, 18) |
| FR-116 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-016, FR-017); `ApplyFighterOptionChoice` has no variant for ASI, Epic Boon or subclass |
| FR-117 | Withdrawn: moved to character-creation.md (FR-020: `classEquipmentOptionUnknown`) |
| FR-118 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-110: no multiclass saves, extra skills, Heavy armor or equipment) |
| FR-119 | Withdrawn: moved to feats-behaviour.md (FR-005 to FR-007: `FeatNotRepeatable`, reported as `featNotRepeatable`) |
| FR-122, FR-123 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-027: `notCurrentlyChosen`, `alreadyChosen`) |
| FR-124 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-018, FR-020, FR-027: a swap's added kind) |
| FR-136 | Withdrawn: moved to level-advancement-and-multiclassing.md (FR-046: another class's subclass is `unknownSubclass`) |

## 5. Model

Types live in `character`; all are immutable value objects. Types from other owners are not
redefined (FR-147).

- **`FighterLevel`**: 1 to 20 (FR-020). **`FighterProgression`**: stateless, built by the host
  (the `New` constructor (returns an error), FR-160) and injected into the five use cases (FR-149); `statsAt(level)` returns
  `FighterLevelStats` (level, Second Wind uses, Weapon Mastery kinds, Action Surge uses, Indomitable
  uses, attacks; no Proficiency Bonus, FR-114); `choicesOnGaining(level, subclass)` returns the
  closed `FighterChoice` of Table B (`FighterChoiceFightingStyle`,
  `FighterChoiceWeaponMasteries(count)` with the kinds to add, FR-023, and
  `FighterChoiceAdditionalFightingStyle`), not the command `FighterOptionChoice` of section 6.
- **`FighterState`** (a `FeatureState`, key `'fighter'`, FR-145): `secondWindExpended`,
  `actionSurgeExpended`, `indomitableExpended`, `fightingStyle` (level 1 style marker, a `FeatId`;
  the feat lives in `CharacterFeats`), `additionalFightingStyle` (same, never replaceable),
  `weaponMasteries` (`WeaponMasteryChoice`). No level, no subclass. `initial()` (FR-151, FR-152),
  `afterRest(kind)` (FR-148), invariants FR-153.
- **Domain services** (pure, synchronous; roll as a value, levels as `FighterLevelStats`; the use
  case has decided `notAFighter` and substituted `initial()` for an absent state; named types
  FR-161): `SecondWind` (`availability`, `resolve`), `TacticalMind.availability(stats, state,
  check)` and `TacticalMind.resolve(stats, state, check, rolled)`, `Indomitable.availability(stats,
  state, save, replacedDie)` and `resolve(..., rolled)`, `ActionSurge.evaluate(stats, state, turn,
  perTurn)` and `FighterChoiceRules` (validates an option choice against facts resolved from
  content, the `Character` and `CharacterFeats`, delegating per FR-163; returns
  `FighterChoiceAccepted` or `FighterChoiceRefused(reasons)`).
- **Declared effects** (queries, no state): `FighterEffects`, `ChampionEffects` (takes the chosen
  `SubclassId`), and the Champion-only `HeroicWarrior` and `HeroicRally` (section 4.8); return types
  `HeroicRallyOutcome`, `HeroicWarriorOutcome`, `TacticalShiftOffer`, `TacticalMasterOffer`; input
  enums FR-154. **`FighterCreationChoices`**: field-less `const` `ClassCreationChoices` (FR-144,
  FR-151).
- **Events** (closed `FighterEvent`, FR-155): `SecondWindUsed`, `TacticalMindUsed`,
  `ActionSurgeUsed`, `IndomitableUsed`, `FighterFightingStyleChosen`,
  `FighterFightingStyleReplaced`, `FighterWeaponMasteriesChosen`, `FighterWeaponMasterySwapped`;
  rejections produce none (FR-108).
- **Results** are closed per use case with operation-prefixed variants
  (shared-types-and-ownership.md FR-017). Reasons are lowerCamel in this check order (FR-129); a
  rejection carries the offending ids, all reported together (FR-121);
  `FighterOptionContentMissing(missing)` holds `MissingContent`s.

  | Result type | Reasons (in order) |
  |---|---|
  | `SecondWindNotAvailable` | `notAFighter`, `noUsesRemaining` |
  | `TacticalMindNotAvailable` | `notAFighter`, `featureNotGained`, `notAnAbilityCheck`, `checkNotFailed`, `noUsesRemaining` |
  | `IndomitableNotAvailable` | `notAFighter`, `featureNotGained`, `notASavingThrow`, `dieNotRolled`, `saveNotFailed`, `noUsesRemaining` |
  | `ActionSurgeNotAvailable` | `notAFighter`, `featureNotGained`, `notOwnTurn`, `alreadyUsedThisTurn`, `noUsesRemaining` |
  | `FighterOptionRejected`, `ChooseFightingStyle` | `notAFighter`, (lookups), `fightingStyleAlreadyChosen`, `notAFightingStyleFeat`, `featNotRepeatable` |
  | `FighterOptionRejected`, `ReplaceFightingStyle` | `notAFighter`, (lookups), `noStyleToReplace`, `eventNotForCharacter`, `noLevelGained`, `notAFightingStyleFeat`, `sameFeat`, `featNotRepeatable` |
  | `FighterOptionRejected`, `ChooseAdditionalFightingStyle` | `notAFighter`, (lookups), `featureNotGained`, `additionalStyleAlreadyChosen`, `notAFightingStyleFeat`, `featNotRepeatable` |
  | `FighterOptionRejected`, `ChooseWeaponMasteries` | `notAFighter`, (lookups), `wrongMasteryCount`, `duplicateMastery`, `notSimpleOrMartialWeapon`, `dropsExistingMastery` |
  | `FighterOptionRejected`, `SwapWeaponMastery` | `notAFighter`, (lookups), `eventNotForCharacter`, `noLongRestFinished`, `notCurrentlyChosen`, `alreadyChosen`, `notSimpleOrMartialWeapon` |
  | `TacticalMasterUnavailable` | `featureNotGained`, `masteryNotUsable` |
  | `HeroicWarriorNotAvailable` | `featureNotGained`, `notInCombat`, `alreadyHasInspiration` |
  | `HeroicRallyNotEligible` | `noHitPoints`, `notBloodied` |

## 6. Interface

Use cases (driving ports), one `call` each, each returning its own closed result
(`UseSecondWindResult`, `UseTacticalMindResult`, `UseIndomitableResult`, `UseActionSurgeResult`,
`ApplyFighterOptionChoiceResult`; shared-types-and-ownership.md FR-017).

| Use case | Command (immutable) | Result | Sync / takes ctx |
|---|---|---|---|
| `UseSecondWind` | `character` | `SecondWindApplied(character, hitPointsToRegain, rolled, tacticalShift: TacticalShiftOffer, cost: bonusAction, events)` or `SecondWindNotAvailable(reason)` | sync; `Tracer.trace`, `telemetry.Timer` |
| `UseTacticalMind` | `character`, `failedCheck: CheckResult` (kind `abilityCheck`, outcome `failure`) | `TacticalMindSucceeded` / `TacticalMindStillFailed` (each with `rolled`, `newTotal`, character, events) or `TacticalMindNotAvailable(reason)` | sync |
| `UseIndomitable` | `character`, `failedSave: CheckResult` (kind `savingThrow`, outcome `failure`), `replacedDie` (index into its `Roll`) | `IndomitableSucceeded` / `IndomitableStillFailed` (each with `rolled`, recomputed `Roll`, `newTotal`, character, events) or `IndomitableNotAvailable(reason)` | sync |
| `UseActionSurge` | `character`, `turn: FighterTurnOwnership`, `perTurn: FighterPerTurnUse` | `ActionSurgeApplied(character, additionalAction: notMagic, events)` or `ActionSurgeNotAvailable(reason)` | sync (no port besides observability) |
| `ApplyFighterOptionChoice` | `character`, `feats: CharacterFeats`, and a closed `FighterOptionChoice`: `ChooseFightingStyle(choice)`; `ReplaceFightingStyle(choice, levelGained: LevelGained)` (names no replaced feat, FR-157); `ChooseAdditionalFightingStyle(choice, subclass: SubclassId (optional))`; `ChooseWeaponMasteries(choice: WeaponMasteryChoice)`; `SwapWeaponMastery(remove: ItemId, add: ItemId, restApplied: RestApplied)` (Long Rest, this character, FR-046, FR-159). No ASI, Epic Boon or subclass variant. Style variants use `FeatTakingRules.take`, never `TakeFeat`. | `FighterOptionApplied(character, feats, events)` (`feats` unchanged for mastery variants), `FighterOptionRejected(reasons)` or `FighterOptionContentMissing(missing)` | context-aware; `Tracer.Start` with the caller's `context.Context`, `telemetry.Timer` (FR-138, FR-140) |

Creation choices are the field-less `FighterCreationChoices` (FR-144). Collaborators are
constructor-injected (FR-149): `FighterProgression`, `telemetry.Telemetry` (optional; default
the `telemetry.Noop*` zero values), `dice.Roller` (sync; the first three use cases) and `ContentSource` (context-aware;
`ApplyFighterOptionChoice` only; not-found becomes `FighterOptionContentMissing`, infrastructure
failure a typed `*srderr.PortError`). A use case reads the Fighter level from `ClassLevels` (none:
`notAFighter`) and passes `statsAt(level)` on. Downstream domain code MUST NOT call the use cases.
Everything named here is exported from `character.go` and the module's public packages.

## 7. telemetry.Telemetry

Per `doc/rules/observability.md` §2: what can go wrong is a `ContentSource` or `dice.Roller` failure,
a missing feat or weapon, or a rule rejection; operators diagnose by class, feature, outcome and
reason (FR-138 to FR-143), not by rolled values. Monitor any rate on
`srd5e.character.feature_used.errors`, rejection spikes per feature and duration p99. Domain pieces
are **silent by design** (§3). Names and keys are constants defined once in `character` (or
package `telemetry`) and feed `doc/observability/signals.md`.

The shared counter `srd5e.character.feature_used`, histogram `srd5e.character.feature_used.duration`
and counter `srd5e.character.feature_used.errors` are owned by character-aggregate.md FR-118
(shared-types-and-ownership.md FR-042, FR-063); this spec defines none. Attribute values:
`srd5e.class` `fighter`; `srd5e.feature` `second_wind`, `tactical_mind`, `action_surge`,
`indomitable`, `fighting_style`, `weapon_mastery` (the last two by option variant); `srd5e.outcome`
`success` for `SecondWindApplied`, `ActionSurgeApplied`, `FighterOptionApplied`,
`TacticalMindSucceeded`, `IndomitableSucceeded`, otherwise `failure` (every other result, and a
returned port failure on `feature_used`); `srd5e.character.rejection` the snake_case of the section 5
reasons and `content_missing` (FR-143); `error.type` only on `.errors` (FR-141).

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.use_second_wind`, `use_tactical_mind`, `use_indomitable`, `use_action_surge`, `apply_fighter_option` | span per call | `srd5e.class`, `srd5e.feature`, `srd5e.outcome`, `srd5e.character.rejection` (rejections and content-missing only); error status on port failure or content-missing | Path of one feature use (`apply_fighter_option` uses ``Tracer.Start``, covering the lookups) | trace sampling; look for error status on port failure or content-missing |
| `class feature resolved` | log, `FINE` | `srd5e.class`, `srd5e.feature`, `srd5e.outcome` | One summary per success, rule-rejection or `...StillFailed` call | off by default; enable when chasing one request |

## 8. Test plan

Doubles: a fixed `dice.Roller`, a recording `telemetrytest.Recorder` (the `*test` packages) and
`FakeContentSource` (shared-types-and-ownership.md FR-038; reports a feat or weapon not found and
throws its typed error). No port is added, so no contract suite. Unit tests build
`FighterState`, `FighterLevelStats`, `CheckResult` and decoded `ProgressionRow`s directly.
Table-driven rows (A to I) are **unit** tests in `character_test` package (`*_test.go`); **application** tests
cover roller call counts and die (never on an unavailable path), delegation, lookups, failures,
result wrapping and telemetry (`doc/rules/testing.md` §3). changed domain code is mutated with
`gomu` at 80 or more (`doc/rules/testing.md` §8, capped Docker run).

| Id | Level | Notes |
|---|---|---|
| FR-020 to FR-028, FR-031, FR-114, FR-115 | unit | Table A as a generated loop over levels 1 to 20 against each band row from decoded `ProgressionRow`s; FR-020 with levels 0, 1, 20, 21; FR-021 cross-checks the proficiency rule; FR-114 Fighter 3 / Rogue 2; Table B rows C01, C04, C07a, C07b, C10, C16, every other level empty; the Defense hint rejects nothing |
| FR-149, FR-150, FR-160 | unit, application | the `New` constructor (returns an error) accepts the Fighter definition and rejects another class, a missing column, an incomplete table; columns read by `ClassColumnId`; constants equal the row annotations at levels 1 to 20; a stub progression changing the Second Wind maximum changes availability |
| FR-032, FR-035 to FR-040, FR-156 to FR-159 | unit | `FighterChoiceRules` with resolved facts and `CharacterFeats` (held-feat check via `FeatTakingRules`, never `TakeFeat`): both categories; replacement with and without `LevelGained`; same feat; held non-Repeatable and Repeatable feats (Repeatable from a double only); second `ChooseFightingStyle` `fightingStyleAlreadyChosen` with no `featNotRepeatable`; `noStyleToReplace`; replacement never touches the Additional style; `additionalStyleAlreadyChosen`; another character's event `eventNotForCharacter` before level or rest kind; a successful replacement removes the old feat (`CharacterFeats.replacing`) and keeps the Additional style |
| FR-034, FR-113, FR-120, FR-121, FR-129 | application | `FakeContentSource` not-found: `FighterOptionContentMissing` for every missing id, no rule reason, state unchanged, span error status, no `error.type`. No Fighter levels (any use case, including a missing feat or item: source never called): `notAFighter`, not `featureNotGained`. Fighter below level 7 naming a missing feat for Additional Fighting Style: content-missing |
| FR-046, FR-047, FR-163 | unit | Mapping of `WeaponMasteryChoice` outcomes (rules tested in weapon-properties-and-mastery.md): counts at levels 1, 3, 4, 10, 16; duplicates; non-weapon; several problems in section 5 order; swap with and without a Long Rest `RestApplied` (`RestKind.longRest` mapped to `MasteryChangeMoment.longRest`) |
| FR-049, FR-050, FR-125, FR-164 | unit | `masteryAccess` for no Fighter levels, no choice, and a choice; `tacticalMasterOptions` at levels 8 and 9, with and without subclass: exactly {Push, Sap, Slow}; unchosen weapon or `NoMasteryFeature` gives `masteryNotUsable`; `featureNotGained` first |
| FR-051 to FR-083, FR-126 to FR-128, FR-162 | unit | Second Wind: Table C rows SW1 to SW6, `rolled` 0 and 11 panics, level 4 vs 5 Tactical Shift, `halfOf` 30, 25, 0, zero uses, recovery. Tactical Mind: Table D rows TM1 to TM6, a `savingThrow` result `notAnAbilityCheck`, level 1 vs 2, Advantage and Disadvantage identical. Action Surge: Table E rows AS1 to AS8, level 17 same and next turn, level 16 to 17 growth. Indomitable: Table F rows IN1 to IN8, an `abilityCheck` result `notASavingThrow`, levels 8 and 9, save not failed, `rolled` 0 and 21 panics, Short Rest regains none |
| FR-051, FR-056, FR-057, FR-064, FR-067, FR-077, FR-080 | application | Roller called exactly once with `1d10` (`UseSecondWind`, `UseTacticalMind`) or `1d20` (`UseIndomitable`) when available, never otherwise (including `dieNotRolled`); result wrapping |
| FR-084 to FR-092, FR-130 to FR-132 | unit | `FighterEffects` level 12 vs 13, with and without subclass, data shape, counts match FR-026; Table G rows CR1 to CR9 over each `ChampionAttackKind`, threshold only (none above 20 or below 18), CR1 panics, the threshold is not combined with a die here (d20-tests.md tests that); Champion Advantage sources at levels 2 and 3 are `RollMode.advantage` |
| FR-093 to FR-100, FR-105, FR-133 to FR-135, FR-137 | unit | Heroic Warrior: `Inspired`/`NotInspired`, combat and not, level 9 vs 10, check order, trigger data. Defy Death: Table H rows DD1 to DD6, level 17 vs 18, level 18 without Champion. Heroic Rally: Table I rows HR1 to HR8, HR8 and negative `currentHp` throw, HR1, HR2, HR5, HR6 pin the `isBloodied` boundary. FR-105: Champion feature levels reported only up to the Fighter level, subclass as input |
| FR-107 to FR-112 | unit, application | Fixed rolls at extremes; input unchanged and equal on rejection; `a defensive copy` on mutating returned lists; failing doubles propagate typed errors, character unchanged; same result with no-op and recording `telemetrytest.Recorder` |
| FR-138 to FR-140 | application | Recording `telemetry.Telemetry`: span names, attribute keys, counter attributes by outcome and reason; durations via `telemetry.Timer` (four sync use cases and `ApplyFighterOptionChoice`), in seconds; for the context-aware one the span and duration cover the lookups and a failing source ends the span with error status and still records the duration |
| FR-141 to FR-143 | application | Throwing roller or source: error status, one `feature_used` increment with `failure` and no `error.type` or rejection, one `srd5e.character.feature_used.errors` increment with snake_case `error.type`, nothing logged; content-missing: error status, `content_missing`, no `error.type`, no log; rejection, success and `...StillFailed`: no error status, one `FINE` log (`StillFailed`: `failure`, no rejection attribute); multi-reason `ChooseWeaponMasteries` attribute is the first reason |
| FR-144 to FR-148, FR-151 to FR-153 | unit, application | Both `ClassCreationChoices` methods (seeding writes `initial()` under `'fighter'`, other keys unchanged; content keys an unmodifiable empty list); `FighterState` value equality; features run with plain values and no port; `afterRest` for Short and Long Rest; absent state treated as initial with Fighter levels, `notAFighter` with none whatever the store holds; negative count panics; count above maximum leaves 0 remaining; package API scan finds none of the FR-147 names |
| FR-154, FR-155, FR-161 | architecture, unit | Package API scan: no `Available`, `Unavailable`, `AttackKind` or other generic or colliding name; FR-161 types present (including `FighterChoice` and its variants); every `FighterEvent` is a `CharacterEvent` and `DomainEvent` carrying `CharacterId`; a test-only `CharacterEvent` implementor compiles outside `FighterEvent` |
| Scenario, architecture | scenario, architecture | Through `the public packages (`github.com/Whealinghold/srd5e-go/...`) and their `*test` packages only: create a Fighter, gain levels to 5, choose masteries, use Second Wind with Tactical Shift, take a Short Rest (content from `FakeContentSource`; no FR is covered only here). Import and signal-name guardrails are `doc/rules/architecture.md` §9 and the observability architecture tests |

Withdrawn ids are tested by their new owners; combining the Champion threshold with a die is tested
by d20-tests.md.

## 9. Open questions and decisions

No question is open. Settled, one line each: FR-074 (a Short Rest regains all Action Surge uses even
at 17: "twice before a rest" is the full allowance, SRD p. 48), FR-128, FR-145 to FR-152 and
shared-types-and-ownership.md FR-022, FR-024, FR-025, FR-049, FR-063; only the level 1 style is
replaceable, since the SRD gives the replacement under "Level 1: Fighting Style" and none for
Additional Fighting Style (FR-036, FR-158). **Multiclass seeding**: no other spec seeds
`FighterState`; an absent state is `initial()` and written on first use (FR-152; package design).
**Replacing a style**: `FeatReplacement.replace` (feats-behaviour.md FR-082, Q20; wraps `CharacterFeats.replacing`), old feat removed first (SRD p. 47, "replace the feat you chose with a different Fighting Style feat"); `FeatNotHeld`/`FeatNotReplaceable` cannot occur (no style gives an increase). **Critical threshold**: d20-tests.md takes an optional critical-range input (18 to 20,
default 20); a roll at or above it is a Critical Hit that hits regardless of modifiers and AC, a
natural 20 always is, and the host passes `ChampionEffects`' value through attacks.md and
combat-action-resolution.md (SRD "Improved Critical", p. 49; "Critical Hit", p. 179;
interpretation: a Critical Hit is defined as hitting regardless of AC).

## 10. Changelog impact

Under `Unreleased`, **Added**: in `character`, `FighterLevel`, `FighterProgression`, `FighterState`
(a `FeatureState` with its typed key), `FighterCreationChoices`, the domain services and effects of
section 5, and the five use cases with their commands, results and events. Fighter content data is
in content-classes.md's entry. Not **BREAKING**: all new API; it uses the aggregate's
`FeatureStates` slot and `ClassCreationChoices`, so mark that change **BREAKING** in
character-aggregate.md if those shipped differently.
