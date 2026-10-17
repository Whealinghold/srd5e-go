# Weapon properties and Mastery properties

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `content-schema-and-decoder.md`, `content-source-port.md`, `content-equipment.md`, `content-classes.md`, `ability-scores.md`, `modifiers-and-advantage.md`, `dice-and-rolls.md`, `d20-tests.md`, `resolve-d20-test-use-case.md`, `derived-stats.md`, `character-aggregate.md`, `errors-and-results.md`, `observability-ports.md` |

Whole spec is `character` (package `github.com/Whealinghold/srd5e-go/character`, files `weapon_*.go`, `mastery_*.go`, `topple_save_usecase.go`; `shared-types-and-ownership.md` FR-016, FR-013, FR-014; type phase 3, see [../roadmap.md](../roadmap.md)). Consumers: `class-barbarian.md`, `class-fighter.md`, `class-paladin.md`, `class-ranger.md`, `class-rogue.md` (Weapon Mastery choice and access), `class-monk.md` (Light facts). `combat` maps this spec's plan and offers onto its own types (`attacks.md`, `combat-action-resolution.md` Q-36, `encounter-aggregate.md`); this spec does not depend on them.

## 1. Purpose and scope

Defines what a weapon's **properties** and **mastery property** add to an attack, and how a character gets to use a mastery property.

- **Weapon proficiency**, including the narrower class proficiencies that depend on a property (Finesse, Light).
- **Weapon Mastery choice**: the validated set of weapon kinds a character may use mastery properties with, how it grows and how it changes. `shared-types-and-ownership.md` FR-020 makes this spec the single owner of `WeaponMasteryChoice`; FR-123 defines the aggregate slot.
- **Property rules** for an attack (Ammunition, Finesse, Heavy, Light, Loading, Reach, Thrown, Two-Handed, Versatile) as a pure planning step whose output is published language of `character` (the ability, damage, range and reach that `attacks.md` section 1 says its caller supplies). `character` may not name a `combat` type (FR-013), so the `combat` host maps the plan (section 6).
- **Light and two-weapon attacks**: the extra attack as a Bonus Action, the damage-modifier rule, and Nick's change of cost.
- **Mastery effects** (Cleave, Graze, Nick, Push, Sap, Slow, Topple, Vex), each reported as an **offer** ("you can ..."), as data. The engine never applies damage, movement, a condition or a Speed change.
- **One use case**, `ResolveToppleSave` (a method of `ToppleSaveResolver`), because Topple is the only effect that needs a die. It composes the pure rules functions of `d20` (`ResolveCheck`, `FailWithoutRoll`) and calls no other use case, `d20.Resolver` included (`shared-types-and-ownership.md` FR-044).

The engine is state in, result out (`doc/rules/architecture.md` §4); pure rules are package-level functions and value types, the one use case is a small struct holding its injected dice and telemetry ports. Enum values are written by their short name (`wrongCount`, `attackDidNotQualify`); the Go constant carries the type prefix and `String()` returns the short name. Closed families (marked "closed") are interfaces with an unexported marker method (`//sumtype:decl`) and one struct type per variant, consumed by a type switch with no `default`; each interface's doc comment lists every variant. The **host** is the consumer's own code; it supplies the facts the GM or table decides (how a weapon is held, a free hand, ammunition on hand, which creature is within 5 feet of another).

**Supersedes** (`shared-types-and-ownership.md` FR-048; the older specs withdraw these in their own reconcile):

| Older requirement | Replaced by |
|---|---|
| `class-fighter.md` FR-046, FR-047 (change one kind after a Long Rest; `noLongRestFinished`) | FR-024, FR-025 (the Fighter keeps carrying `RestApplied`, `class-fighter.md` FR-163) |
| The Fighter's `masteryFor`, `FighterMasteryUsable`, `FighterMasteryUnusable` requirement | FR-004, FR-005; `noFighterLevels` stays the Fighter's `NoMasteryFeature` |
| `class-ranger.md` FR-040, FR-041 (change after a Long Rest, `ReplaceAnyKinds`) | FR-024, FR-026 |
| `class-barbarian.md` FR-043 (Simple or Martial Melee eligibility) | FR-017 |
| `attacks.md` thrown-weapon ability requirement and the Finesse and thrown sentence of FR-007 | FR-031, FR-032 |

Requirements that merely pass a count or eligibility rule here (`class-barbarian.md` FR-041, `class-rogue.md` FR-111, `class-ranger.md` FR-037) stay where they are.

**Out of scope:**

- **Weapon data** (Weapons table, p. 91; property and mastery names): `content-equipment.md`, `content-schema-and-decoder.md`.
- **Making the attack roll and rolling damage**, cover, range bands, Critical Hits: `attacks.md`. This spec consumes only `AttackRollResult`; the combat-side facts it needs (forced miss, miss with no roll) arrive as `MasteryAttackOutcome` (section 4.6).
- **Improvised weapons** (SRD 5.2.1, "Improvised Weapons", p. 183): `attacks.md` FR-046 to FR-049, FR-100. This spec only refuses to plan the two SRD cases (FR-034).
- **Unarmed Strike and spell attacks**: not weapon attacks (`attacks.md` FR-113).
- **Class features that use weapons** (Extra Attack, Reckless Attack, Brutal Strike, Martial Arts, `class-monk.md` FR-024, Sneak Attack, Studied Attacks, Hunter's Prey). Tactical Master (p. 48, set {Push, Sap, Slow}) belongs to `class-fighter.md` FR-049, FR-125; this spec accepts the chosen replacement (FR-006).
- **Fighting Style feats** (p. 88): `feats-behaviour.md` owns them; this spec only accepts `ExtraAttackDamageSource` (FR-066). Great Weapon Fighting is not applied here.
- **The action economy** (Bonus Action available, attack counts, Opportunity Attacks as a Reaction): `initiative-and-actions.md`, `attack-and-help-actions.md`. The extra attack's cost is data (FR-063, FR-064).
- **Applying effects** (movement, Prone, Speed reduction tracking, expiry): `conditions.md`, `initiative-and-actions.md`, `encounter-aggregate.md`. Immunity to Prone (p. 183): the host does not call `ResolveToppleSave` for such a target or ignores the effect.
- **Equipping and unequipping weapons**: `attack-and-help-actions.md`. Hands and held items are host facts; `WeaponWielding` only carries them (p. 177).
- **Monster use of mastery properties** (stat blocks carry their own attacks; only FR-013 applies), **magic weapons**, Armor, Shields (`content-equipment.md`).

## 2. Source

- `SRD 5.2.1, "Equipment", "Weapons", "Weapon Proficiency", "Properties" (Ammunition, Finesse, Heavy, Light), p. 89`; `SRD 5.2.1, "Equipment", "Properties" (Loading, Range, Reach, Thrown, Two-Handed, Versatile), "Mastery Properties" (Cleave, Graze, Nick, Push, Sap, Slow, Topple, Vex), "Improvised Weapons" (sidebar), p. 90`; `SRD 5.2.1, "Equipment", the Weapons table, p. 91`
- `SRD 5.2.1, "Fighter", "Level 1: Weapon Mastery", "Level 9: Tactical Master", p. 48`; `"Barbarian", "Level 1: Weapon Mastery", p. 29`; `"Paladin", "Level 1: Weapon Mastery", p. 54`; `"Ranger", "Level 1: Weapon Mastery", pp. 58-59`; `"Rogue", "Level 1: Weapon Mastery", p. 62`
- `SRD 5.2.1, "Core Monk Traits", p. 49; "Core Rogue Traits", p. 61`; `SRD 5.2.1, "Fighting Style Feats" (Two-Weapon Fighting, Great Weapon Fighting), p. 88` (extension point only)
- `SRD 5.2.1, "Attack Rolls" (Attack Roll Abilities, Proficiency Bonus), p. 7; "They Don't Stack", p. 8; "Creature Size", "Making an Attack", p. 14; "Reach", "Opportunity Attacks", "Melee Attacks", "Ranged Attacks", "Range", p. 15; "Damage Rolls", p. 16`
- `SRD 5.2.1, "Rules Glossary": "Attack [Action]", "Bonus Action", p. 177; "Immunity", "Improvised Weapons", p. 183; "Opportunity Attacks", p. 185; "Prone", "Reach", "Reaction", p. 186; "Temporary Hit Points", "Unarmed Strike", p. 190; "Weapon", "Weapon Attack", p. 191`

Rule files: `doc/rules/architecture.md` §2, §3, §4, §6; `doc/rules/observability.md` §2, §5, §6, §7; `doc/rules/testing.md` §4, §5. Printed page numbers equal PDF page numbers.

## 3. Vocabulary

Types marked "content" are defined by the spec named; this spec never redefines them.

| SRD term | Type | Notes |
|---|---|---|
| Weapon, Category, Melee or Ranged, Damage, Properties, Mastery | `Weapon`, `WeaponCategory`, `WeaponType`, `WeaponDamage`, `WeaponPropertyUse`, `MasteryProperty` (content) | `content-equipment.md` FR-016 to FR-020, FR-101 to FR-103, FR-106; `MasteryProperty` by `content-schema-and-decoder.md` FR-113 (`shared-types-and-ownership.md` FR-005, FR-046). |
| property (nine) | `WeaponProperty` (content enum) | "Range" (p. 90) is the parenthetical on Ammunition and Thrown, not a tenth property. |
| mastery property (eight) | `MasteryProperty` (content enum) | One per weapon (p. 89). |
| Weapon Mastery (class feature) | `WeaponMasteryChoice` | FR-014 to FR-029. |
| "kinds of weapons" | `ItemID` of a `Weapon` | Bundled or host-supplied entry (`content-source-port.md`). |
| Weapon Proficiency | `WeaponProficiencyRule`, `WeaponProficiencies`, `WeaponProficiencyStatus` | pp. 89, 7, 61, 49. Class data is `WeaponProficiency` (`content-classes.md` FR-021); `WeaponProficienciesFromClassData` (FR-120). |
| "a feature that lets you use it" | `WeaponMasteryAccess` | `NoMasteryFeature` or `MasteryFromChoice(choice)`. |
| Strength or Dexterity (Finesse) | `AbilityPreference` | `DefaultAbility` or `ChosenAbility(Ability)`. |
| one-handed, two hands; free hand | `Grip`, `HandState` | Host facts (pp. 89-90). |
| how a weapon is held and fed | `WeaponWielding` | `Grip`, `TwoHandedExemption`, `HandState`, `LoadingState`, optional `AmmunitionSupply`. |
| Loading limit; piece of ammunition | `LoadingState`, `AmmunitionSupply` | Supply counts pieces of one `AmmunitionType` (`content-equipment.md` FR-103; p. 89). |
| reach; range | `ReachInFeet`, `PlannedReach`; `RangePair` (`content-equipment.md` FR-103), `PlannedRange` | Package terms; `combat` maps them. |
| Heavy Disadvantage | `modifier.Disadvantage` (a `modifier.RollCircumstance`) | `modifiers-and-advantage.md`. |
| Light extra attack; its damage modifier | `LightExtraAttackPlanner`, `LightExtraAttackResult`, `AbilityModifierToDamage`, `ExtraAttackDamageSource` | p. 89. Modifier: `omitUnlessNegative` (rule), `alwaysAdd`, `neverAdd` (Blowgun, FR-114; the Light planner gives only the first two). Input: `NoExtraAttackDamageEffect` or `ExtraAttackDamageEffectActive()` (`feats-behaviour.md` maps its effect to it) (p. 88). |
| Bonus Action, "as part of the Attack action" | `ExtraAttackCost` | `spendsBonusAction`, `partOfAttackAction` (Nick). |
| a different Light weapon; "once per turn" | `WeaponInstanceKey`, `WeaponTurnState` | Package terms. |
| Cleave ... Vex effects | `MasteryOffer` (closed) | `CleaveOffer`, `GrazeOffer`, `PushOffer`, `SapOffer`, `SlowOffer`, `ToppleOffer`, `VexOffer`, `MasteryNone`. |
| Tactical Master | `MasteryInPlay` | `WeaponsOwnMastery`, `ReplacedMastery(MasteryProperty)` (p. 48). |
| hit, miss, Critical Hit | `AttackRollResult` (`d20`), `MasteryAttackOutcome` | No parallel outcome enum (`shared-types-and-ownership.md` FR-004); `MasteryAttackOutcome` wraps the roll and adds two no-hit cases (FR-118). |
| the plan's damage | `WeaponDamage` (`WeaponDiceDamage`, `WeaponFixedDamage`), `AbilityModifierToDamage` | `content-equipment.md` FR-102; FR-054. |
| Large or smaller; Speed reduced by 10 feet; Prone, DC | `CreatureSize`, `SpeedReduction`, `ConditionKind`, `Ability`, `DifficultyClass` | pp. 14, 90. |
| Proficiency Bonus | `ProficiencyBonus` (`derived-stats.md`) | |

**Package terms** (no SRD collision): the `Weapon*` planner, plan and proficiency types, the `Mastery*` choice, offer and outcome types, `Light*`, `Push*`, `Cleave*`, `ReachInFeet`, `Planned*`, `WeaponInstanceKey`, `WeaponTurnState`, `ToppleSaveResolver`, `ResolveToppleSave` and its command, results and detail. Content types (`WeaponProficiency`, `WeaponDiceDamage`, `WeaponFixedDamage`) are never redefined.

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. One behaviour each.

### 4.1 Closed sets and access

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | The rules MUST treat the weapon properties as exactly the nine of `WeaponProperty` (Ammunition, Finesse, Heavy, Light, Loading, Reach, Thrown, Two-Handed, Versatile) and switch over them exhaustively (no default branch), so a new value fails to compile. Range is read from the Ammunition and Thrown parameters. | SRD 5.2.1, "Properties", pp. 89-90 | happy |
| FR-002 | The mastery effects MUST switch exhaustively over exactly the eight `MasteryProperty` values: Cleave, Graze, Nick, Push, Sap, Slow, Topple, Vex. | SRD 5.2.1, "Mastery Properties", p. 90 | happy |
| FR-003 | Every effect computation MUST take exactly one mastery property per weapon, the weapon's own (`Weapon.mastery`) unless FR-006 replaces it for one attack. A weapon has no mastery-less state. | SRD 5.2.1, "Weapons" (Mastery), p. 89 | happy |
| FR-004 | A mastery property MUST be usable only when the character has a feature that lets them use it for that weapon kind: `WeaponMasteryAccess.Permits(weaponID)` is true exactly for the kinds in the character's `WeaponMasteryChoice`. Otherwise it MUST be false and an offer MUST be `MasteryNone(featureNotAvailable)`. | SRD 5.2.1, "Weapons" (Mastery), p. 89; "Mastery Properties", p. 90 | unhappy (state error) |
| FR-005 | `NoMasteryFeature` MUST permit no weapon kind, for every weapon, including every kind a `WeaponMasteryChoice` could contain. | SRD 5.2.1, "Mastery Properties", p. 90 ("unlocks the property for the character") | unhappy (empty and minimal case) |
| FR-006 | When the host supplies `ReplacedMastery(m)` (Tactical Master), the effects MUST use `m` instead of the weapon's own mastery property for that attack only, and only when `Permits(weaponID)` holds; the stored `WeaponMasteryChoice` MUST NOT change. Which replacements are legal is `class-fighter.md` FR-049. | SRD 5.2.1, "Level 9: Tactical Master", p. 48 | unhappy (rule interaction) |

### 4.2 Weapon proficiency

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | `WeaponProficiencies.StatusFor(weapon)` MUST return `proficient` or `notProficient`, and `notProficient` MUST NOT reject the attack: only the Proficiency Bonus is withheld (the host passes the status to `attacks.md`). | SRD 5.2.1, "Weapon Proficiency", p. 89; "Attack Rolls", "Proficiency Bonus", p. 7 | happy |
| FR-008 | `CategoryProficiency(category)` MUST make a character proficient with every weapon of that category and none of the other (Simple covers the 14 Simple weapons, Martial the 24 Martial weapons). | SRD 5.2.1, "Weapons" (Category), p. 89; Weapons table, p. 91 | happy |
| FR-009 | `CategoryWithAnyPropertyProficiency(Martial, {Finesse, Light})` (Rogue) MUST make a character proficient with a Martial weapon that has Finesse, Light or both, and not one with neither. `{Light}` alone (Monk) MUST NOT match a Martial weapon that only has Finesse. | SRD 5.2.1, "Core Rogue Traits", p. 61; "Core Monk Traits", p. 49 | unhappy (boundary) |
| FR-010 | `WeaponProficiencies` MUST be proficient when at least one rule matches (the Rogue's Simple plus Martial-with-property is two rules), and `notProficient` for every weapon when it has no rule. | SRD 5.2.1, "Core Rogue Traits", p. 61 | unhappy (empty and minimal case) |
| FR-011 | A `CategoryWithAnyPropertyProficiency` with an empty property set MUST be rejected: `NewCategoryWithAnyPropertyProficiency(category, properties)` returns an `*InvalidError` (matching `srderr.ErrInvalidInput`, naming `properties`) and `MustCategoryWithAnyPropertyProficiency` panics with it (the decoder already validated the set, FR-120). | `doc/rules/architecture.md` §3, §6 (package design) | unhappy (invalid input) |
| FR-012 | A property-based rule MUST match a weapon by its `WeaponPropertyUse` names only, ignoring parameters (the Range of Thrown, the damage of Versatile). | SRD 5.2.1, "Properties", pp. 89-90 | unhappy (rule interaction) |
| FR-013 | `AnyStatBlockWeaponProficiency` (a monster) MUST be proficient with every weapon the host asks about (the host asks only about stat-block weapons). | SRD 5.2.1, "Weapon Proficiency", p. 89 | happy |

### 4.3 Weapon Mastery choice (character)

`WeaponMasteryChoice` is a set of weapon kinds. Class specs pass the count and eligibility rule (Barbarian `SimpleOrMartialMeleeEligibility`; Fighter `SimpleOrMartialEligibility`; Paladin, Ranger, Rogue `ProficientWeaponEligibility`) and map the problems to their own result types. Kinds compare as sets (membership, not position); an accepted choice still keeps a defined order. `CreateWeaponMasteryChoice` and `ExtendWeaponMasteryChoice` take full candidate lists; `ChangeWeaponMasteryChoice` takes `MasterySwap{Remove, Add}` pairs. All three are pure functions that return a closed `WeaponMasteryChoiceResult` (a rejection is a result) and an `error` only for an invalid `allowedCount` (FR-028). `MasteryChangeMoment` (`shortRest`, `longRest`) is owned here so this spec does not depend on `rests-and-hit-dice.md` (`shared-types-and-ownership.md` FR-056); the host maps its `RestKind`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `CreateWeaponMasteryChoice(candidates []ItemID, resolved map[ItemID]Weapon, allowedCount int, eligibility MasteryEligibility) (WeaponMasteryChoiceResult, error)` MUST accept exactly `allowedCount` distinct kinds that satisfy the eligibility rule and return `WeaponMasteryChoiceAccepted(choice)` with the kinds in the order given. | SRD 5.2.1, "Level 1: Weapon Mastery" (Fighter p. 48, Barbarian p. 29, Paladin p. 54, Ranger pp. 58-59, Rogue p. 62) | happy |
| FR-015 | A candidate list with fewer or more kinds than `allowedCount` (including none) MUST be rejected with `MasteryProblemKind.wrongCount`. | SRD 5.2.1, "Level 1: Weapon Mastery", pp. 29, 48, 54, 58, 62 | unhappy (boundary) |
| FR-016 | A candidate that appears twice MUST be rejected with `duplicateWeapon` naming the kind. | SRD 5.2.1, "Level 1: Weapon Mastery" ("two kinds", "three kinds"), pp. 29, 48 | unhappy (invalid input) |
| FR-017 | Under `SimpleOrMartialMeleeEligibility` a resolved weapon whose `WeaponType` is Ranged MUST be rejected with `notMelee` naming the kind; a Melee Simple or Martial weapon (Greataxe, Handaxe) MUST be accepted. | SRD 5.2.1, "Barbarian", "Level 1: Weapon Mastery" ("Simple or Martial Melee weapons"), p. 29 | unhappy (invalid input) |
| FR-018 | Under `SimpleOrMartialEligibility` every Melee and Ranged weapon MUST be eligible. | SRD 5.2.1, "Fighter", "Level 1: Weapon Mastery", p. 48; Weapons table, p. 91 | happy |
| FR-019 | Under `ProficientWeaponEligibility(proficiencies)` a weapon for which `StatusFor` is `notProficient` MUST be rejected with `notProficient` naming the kind. | SRD 5.2.1, "Level 1: Weapon Mastery" ("with which you have proficiency"), pp. 54, 58-59, 62 | unhappy (invalid input) |
| FR-020 | A candidate with no entry in `resolved` MUST be rejected with `weaponNotResolved` naming the kind (the class use case maps a content `NotFound`, or an id that is not a `Weapon`, to it). Content lookups come first: when any candidate is unresolved the rejection MUST list only the `weaponNotResolved` problems, one per unresolved kind, and no rule problem. `class-fighter.md` states the same order. | `doc/rules/architecture.md` §4 (port failure contract) | unhappy (missing content) |
| FR-021 | A rejection MUST list every rule problem found, each as `MasteryProblem(kind, weaponID?)`, with no display text. Order: `wrongCount` (no weapon id) first, then per candidate in candidate order, where `duplicateWeapon` is reported at the second and later occurrence of a kind, and `notMelee` and `notProficient` once per distinct kind at its first occurrence. | `doc/rules/architecture.md` §5, §6 (package design) | unhappy (invalid input) |
| FR-022 | `ExtendWeaponMasteryChoice(current WeaponMasteryChoice, proposed []ItemID, newAllowedCount int, resolved map[ItemID]Weapon, eligibility MasteryEligibility) (WeaponMasteryChoiceResult, error)` MUST take the full proposed list and accept when it holds exactly `newAllowedCount` distinct kinds, contains every kind of `current` (as sets), and satisfies FR-016 to FR-020. The accepted choice keeps the order of `proposed`. | SRD 5.2.1, "Level 1: Weapon Mastery" ("When you reach certain ... levels, you gain the ability to use the mastery properties of more kinds of weapons"), pp. 29, 48 | happy |
| FR-023 | An extension whose `proposed` omits a kind of `current` MUST be rejected with `dropsExistingMastery` naming each dropped kind (set difference). A `current` kind repeated in `proposed` is a `duplicateWeapon` (FR-016); `alreadyChosen` does not apply to `ExtendWeaponMasteryChoice`. | SRD 5.2.1, "Level 1: Weapon Mastery", pp. 29, 48 | unhappy (rule interaction) |
| FR-024 | `ChangeWeaponMasteryChoice(req WeaponMasteryChange) (WeaponMasteryChoiceResult, error)`, where `WeaponMasteryChange{Current WeaponMasteryChoice; Swaps []MasterySwap; Moment MasteryChangeMoment; Rule MasteryRetrainRule; Resolved map[ItemID]Weapon; Eligibility MasteryEligibility}`, MUST accept a change only when `Moment` is `MasteryChangeMoment` `longRest` (the host maps `RestKind.longRest` to it); `shortRest` MUST be rejected with `wrongMoment`. A rejection does not alter `Current` (FR-029). | SRD 5.2.1, "Level 1: Weapon Mastery" ("Whenever you finish a Long Rest"), pp. 29, 48, 54, 59, 62 | unhappy (state error) |
| FR-025 | Under `ReplaceOneKind` (Barbarian, Fighter) a `swaps` list of at most one pair MUST be accepted; two or more pairs MUST be rejected with `changesMoreThanOne`. An empty list is a valid no-change returning a choice equal to `current`. | SRD 5.2.1, "Barbarian", "Level 1: Weapon Mastery" ("change one of those weapon choices"), p. 29; "Fighter", p. 48 | unhappy (boundary) |
| FR-026 | Under `ReplaceAnyKinds` (Paladin, Ranger, Rogue) a `swaps` list of one, several or all kinds MUST be accepted when it satisfies FR-027 and the result satisfies FR-014 to FR-020. | SRD 5.2.1, "Paladin", "Level 1: Weapon Mastery" ("change the kinds of weapons you chose"), p. 54; "Ranger", p. 59; "Rogue", p. 62 | happy |
| FR-027 | A swap whose `remove` is not in `current` MUST be rejected with `notCurrentlyChosen`; a swap whose `add` is already in `current` and not removed by another pair with `alreadyChosen`; a kind removed or added by two pairs with `duplicateWeapon`. The accepted choice replaces each removed kind in place with its added kind, leaving the order of the others. Comparison is by set membership; the order of `swaps` does not matter. | SRD 5.2.1, "Level 1: Weapon Mastery", pp. 29, 48 | unhappy (invalid input) |
| FR-125 | `ExtendWeaponMasteryChoice` MUST report problems in this order: `weaponNotResolved` alone (FR-020); otherwise `wrongCount` (no weapon id), then `dropsExistingMastery` once per dropped kind in the order of `current`, then the per-candidate problems of FR-021 in the order of `proposed`. `ChangeWeaponMasteryChoice` MUST report: `wrongMoment` alone (short-circuits); otherwise `weaponNotResolved` alone for the `add` kinds (FR-020); otherwise `changesMoreThanOne` (no weapon id); then per swap in order, `notCurrentlyChosen` (its `remove`), `alreadyChosen` (its `add`), `duplicateWeapon` (second or later use of a kind); then `notMelee` and `notProficient` once per added kind that raised no earlier problem. | `doc/rules/architecture.md` §6 (package design) | unhappy (rule interaction) |
| FR-028 | `WeaponMasteryChoice` MUST expose its kinds through `Kinds()` as a copy (mutating it leaves the choice unchanged) and compare equal by value through `Equal` (same kinds, same order). `allowedCount` and `newAllowedCount` are class-progression constants, so a value below 1 is a programmer error: the function MUST return an `*InvalidError` naming `allowedCount` (matching `srderr.ErrInvalidInput`) and an undefined result; it MUST NOT panic. | `doc/rules/architecture.md` §3, §4 (handoffs), §6 | unhappy (immutability) |
| FR-029 | `CreateWeaponMasteryChoice`, `ExtendWeaponMasteryChoice` and `ChangeWeaponMasteryChoice` MUST NOT panic and MUST NOT return an error for any well-typed input (except FR-028's `allowedCount` precondition) and MUST NOT alter their inputs (the candidate, swap and `resolved` collections are only read). | `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 | unhappy (immutability) |

### 4.4 Weapon property rules for an attack

`PlanWeaponAttack(in WeaponAttackInput) WeaponAttackPlanResult` takes one weapon, a `WeaponAttackMode` (closed: `MeleeAttackMode`, `ThrownAttackMode`, `RangedAttackMode`), the attacker's Strength and Dexterity scores (`ability.Score`), an `AbilityPreference`, a `WeaponWielding`, the attacker's `ReachInFeet` (the zero value reads as the default 5) and the `WeaponProficiencies`, and returns `WeaponAttackPlanned(plan)` or `WeaponAttackPlanBlocked(reason)`. It is pure, takes no `ctx`, returns no `error` and does not panic.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | A `MeleeAttackMode` use MUST plan `WeaponAttackKind.meleeWeaponAttack`; `ThrownAttackMode` and `RangedAttackMode` MUST plan `rangedWeaponAttack`. | SRD 5.2.1, "Weapons" (Melee or Ranged), p. 89; "Thrown", p. 90 | happy |
| FR-031 | With no Finesse, the plan's ability MUST be Strength for a melee use and a thrown Melee weapon, and Dexterity for a ranged use and a thrown Ranged weapon. | SRD 5.2.1, "Attack Rolls", p. 7; "Thrown", p. 90 | happy |
| FR-032 | With Finesse and `ChosenAbility(Strength)` or `ChosenAbility(Dexterity)`, the plan's ability MUST be the chosen one in every mode, whichever score is higher; `DefaultAbility` MUST give FR-031's default. | SRD 5.2.1, "Finesse", p. 89; "Attack Rolls", p. 7 | happy |
| FR-033 | The plan MUST expose one ability modifier used for the attack roll and, through its `AbilityModifierToDamage` (FR-054), for the damage roll, with no separate damage modifier value. A fixed damage amount is the exception (FR-114). | SRD 5.2.1, "Finesse" ("You must use the same modifier for both rolls"), p. 89; "Damage Rolls", p. 16 | unhappy (rule interaction) |
| FR-034 | A use the SRD names as improvised MUST be blocked with `improvisedUse`: exactly a Ranged weapon in `MeleeAttackMode`, and a `ThrownAttackMode` with a Melee weapon that lacks Thrown. The `combat` host then applies the improvised rules (`attacks.md` FR-046 to FR-049). Other unsupported uses get `modeNotSupportedByWeapon` (FR-116). | SRD 5.2.1, "Improvised Weapons", p. 183 ("if you use a Ranged weapon to make a melee attack or throw a Melee weapon that lacks the Thrown property"); "Thrown", p. 90 | unhappy (rule interaction) |
| FR-035 | `ChosenAbility(a)` for a weapon without Finesse MUST be blocked with `finesseNotAvailable`; `NewChosenAbility(a ability.Ability) (ChosenAbility, error)` with an ability other than Strength or Dexterity MUST return an `*InvalidError` naming `ability` (`MustChosenAbility` panics with it). | SRD 5.2.1, "Finesse", p. 89 | unhappy (invalid input) |
| FR-036 | A Heavy Melee weapon MUST add a `Disadvantage` circumstance when the attacker's Strength score is below 13 (12 yes, 13 no), and a Heavy Ranged weapon when the Dexterity score is below 13. | SRD 5.2.1, "Heavy", p. 89 | unhappy (boundary) |
| FR-037 | The Heavy test MUST use the ability score of the weapon's classification (Melee: Strength; Ranged: Dexterity), never the chosen ability and never the modifier. | SRD 5.2.1, "Heavy" ("Strength score", "Dexterity score"), p. 89 | unhappy (rule interaction) |
| FR-038 | A weapon without Heavy MUST add no circumstance, whatever the scores. | SRD 5.2.1, "Heavy", p. 89 | unhappy (empty and minimal case) |
| FR-039 | A melee use of a Reach weapon MUST plan a `PlannedReach` equal to the attacker's `ReachInFeet` plus 5 feet (5 gives 10; 10 gives 15). `ReachInFeet` is validated by FR-121. | SRD 5.2.1, "Reach", p. 90; "Reach", pp. 15, 186 | happy |
| FR-040 | A melee use of a weapon without Reach MUST plan the attacker's own reach unchanged; a thrown or ranged use MUST plan a `PlannedRange` and no reach. | SRD 5.2.1, "Reach", p. 90; "Range", p. 15 | unhappy (boundary) |
| FR-041 | `OpportunityAttackReach(weapon, attackerReach)` MUST return FR-039's extended reach for a Reach weapon and the attacker's reach otherwise ("when determining your reach for Opportunity Attacks with it"). | SRD 5.2.1, "Reach", p. 90; "Opportunity Attacks", p. 185 | happy |
| FR-042 | A `ThrownAttackMode` plan MUST carry the `RangePair` of the weapon's Thrown parameter (20/60 Dagger, Handaxe, Light Hammer, Spear, Trident, Dart; 30/120 Javelin), and a `RangedAttackMode` plan the Ammunition parameter's range (Shortbow 80/320). | SRD 5.2.1, "Range", "Thrown", p. 90; Weapons table, p. 91 | happy |
| FR-043 | A Thrown weapon MUST NOT require a free hand: `HandState` MUST NOT block a `ThrownAttackMode` plan. Package decision: the SRD says only that the weapon can be drawn "as part of the attack". | Package decision (context: SRD 5.2.1, "Thrown", p. 90) | unhappy (rule interaction) |
| FR-044 | An Ammunition weapon in `RangedAttackMode` MUST be blocked with `noAmmunition` when its `AmmunitionSupply` holds 0 pieces, and MUST plan with 1 or more. | SRD 5.2.1, "Ammunition", p. 89 | unhappy (boundary) |
| FR-045 | `AmmunitionSupply.Expend()` MUST return a supply with exactly one piece fewer, carried by the plan as `ammunitionAfter`; each attack expends one piece, whatever its outcome. | SRD 5.2.1, "Ammunition" ("Each attack expends one piece of ammunition"), p. 89 | happy |
| FR-046 | `AmmunitionSupply.RecoverAfterFight(piecesUsedInFight)` MUST return half the pieces used, rounded down (0 gives 0, 1 gives 0, 2 gives 1, 3 gives 1, 9 gives 4); a negative count MUST be an `*InvalidError` (`RecoverAfterFight(n int) (int, error)`). The 1 minute is the host's. | SRD 5.2.1, "Ammunition" ("recover half the ammunition ... (round down)"), p. 89 | unhappy (boundary) |
| FR-047 | An Ammunition weapon without Two-Handed (Hand Crossbow, Pistol) MUST be blocked with `noFreeHandToLoad` when `HandState` is `noFreeHand`; one with Two-Handed MUST NOT need a free hand. | SRD 5.2.1, "Ammunition" ("you need a free hand to load a one-handed weapon"), p. 89 | unhappy (state error) |
| FR-048 | A Loading weapon MUST be blocked with `loadingLimit` when `LoadingState` is `alreadyFiredThisActivation` (a second shot in one action, Bonus Action or Reaction), however many attacks the creature has, and MUST plan with `ready`; the plan carries `loadingAfter = alreadyFiredThisActivation`. The host resets it per activation. | SRD 5.2.1, "Loading", p. 90 | unhappy (state error) |
| FR-049 | A weapon without Loading MUST ignore `LoadingState` for blocking, and `loadingAfter` MUST equal the input unchanged. | SRD 5.2.1, "Loading", p. 90 | unhappy (rule interaction) |
| FR-050 | A Two-Handed weapon MUST be blocked with `needsTwoHands` unless `Grip` is `twoHanded`, or its Two-Handed parameter has a condition and the host reports `TwoHandedExemption.exemptionApplies` (the Lance "unless mounted"). | SRD 5.2.1, "Two-Handed", p. 90; Weapons table, p. 91 | unhappy (state error) |
| FR-051 | `exemptionApplies` reported for a weapon whose Two-Handed has no condition, or without Two-Handed, is a contradictory host fact: the plan MUST be blocked with `exemptionNotApplicable` (a result; the engine does not interpret the condition text). | `doc/rules/architecture.md` §6 (package design) | unhappy (invalid input) |
| FR-052 | A melee use of a Versatile weapon with `Grip.twoHanded` MUST plan the Versatile damage dice (Longsword 1d10, Spear 1d8, Quarterstaff 1d8) of the same type; with `oneHanded` the base dice; a `ThrownAttackMode` use the base dice whatever the grip. | SRD 5.2.1, "Versatile" ("when used with two hands to make a melee attack"), p. 90; Weapons table, p. 91 | unhappy (boundary) |
| FR-053 | A non-Versatile, non-Two-Handed weapon held with `Grip.twoHanded` (a Mace) MUST plan its base dice and MUST NOT be blocked. Package decision: the SRD restricts no grip. | Package decision (context: SRD 5.2.1, "Versatile", p. 90) | unhappy (rule interaction) |
| FR-054 | The plan MUST carry a content `WeaponDamage`, never a rolled amount: a `WeaponDiceDamage` (the weapon's dice, or Versatile under FR-052) with `AbilityModifierToDamage.alwaysAdd`, or a `WeaponFixedDamage` with `neverAdd` (FR-114). It MUST also carry the `WeaponProficiencyStatus` of FR-007; the roll belongs to `combat`, which adds the one ability modifier once. | `doc/rules/architecture.md` §3 (determinism); SRD 5.2.1, "Damage Rolls", p. 16 | happy |
| FR-055 | When several block reasons apply, the one reported MUST be, in order: `improvisedUse`, `modeNotSupportedByWeapon` (never together), `finesseNotAvailable`, `exemptionNotApplicable`, `needsTwoHands`, `wrongAmmunitionType`, `noAmmunition`, `noFreeHandToLoad`, `loadingLimit`. | `doc/rules/architecture.md` §6 (package design) | unhappy (rule interaction) |
| FR-056 | A non-proficient attacker and a Heavy shortfall MUST NOT block the plan: the first changes the proficiency status, the second adds a Disadvantage. | SRD 5.2.1, "Weapon Proficiency", "Heavy", p. 89 | unhappy (rule interaction) |
| FR-114 | A weapon whose `WeaponDamage` is a fixed amount that uses no roll (the Blowgun: 1 Piercing) MUST plan a `WeaponFixedDamage` with `neverAdd`, for a ranged use as any other; the attack roll still uses the planned ability. | SRD 5.2.1, "Damage Rolls" ("you don't add your ability modifier to a fixed damage amount that doesn't use a roll, such as the damage of a Blowgun"), p. 16; Weapons table, p. 91; `content-equipment.md` FR-102 | unhappy (rule interaction) |
| FR-115 | `AmmunitionSupply` MUST carry the ammunition type of its pieces. A `RangedAttackMode` plan for an Ammunition weapon whose parameter names another type (Shortbow, Arrow, with Bolt) MUST be blocked with `wrongAmmunitionType`, checked before `noAmmunition`. The type compares as `content-equipment.md` keeps it (printed label; its Q10). | SRD 5.2.1, "Ammunition" ("The type of ammunition required is specified with the weapon's range"), p. 89; Weapons table, p. 91 | unhappy (invalid input) |
| FR-116 | A use that is not an SRD improvised case but that the weapon cannot support MUST be blocked with `modeNotSupportedByWeapon`: `RangedAttackMode` with a weapon lacking Ammunition (a Melee weapon, or a Dart), and `ThrownAttackMode` with a Ranged weapon lacking Thrown. A Dart (Ranged, Thrown) is planned with `ThrownAttackMode`, which gets the Proficiency Bonus. Package decision: the SRD's general clause (p. 183) arguably makes a thrown Shortbow improvised too, but names only FR-034's two cases, so the `combat` host may apply the improvised rules itself. | Package decision (context: SRD 5.2.1, "Improvised Weapons", p. 183; "Ammunition", p. 89; "Thrown", p. 90; Weapons table, p. 91) | unhappy (rule interaction) |
| FR-117 | `AmmunitionSupply.Expend() (AmmunitionSupply, error)` on 0 pieces MUST return an error matching `ErrSupplyEmpty` (and no supply); it MUST NOT panic: `PlanWeaponAttack` already blocks that case (FR-044). | `doc/rules/architecture.md` §6 | unhappy (boundary) |
| FR-123 | `WeaponMasteryChoiceState` MUST implement `FeatureState` (`character-aggregate.md` FR-092, FR-093), hold exactly one `WeaponMasteryChoice`, be immutable and compare by value. `WeaponMasteryChoiceKey()` MUST be the one `FeatureStateKey<WeaponMasteryChoiceState>` named `weapon_mastery_choice` (`[a-z][a-z0-9_]*`, equal by name). The five classes MUST store and read the choice only through this key and keep no copy. When the key holds nothing, `WeaponMasteryAccess` is `NoMasteryFeature`. | `shared-types-and-ownership.md` FR-020, FR-022; `character-aggregate.md` FR-092, FR-093 | happy |
| FR-124 | An Ammunition weapon in `RangedAttackMode` whose `WeaponWielding` has no `AmmunitionSupply` MUST be blocked with `noAmmunition`, as a supply of 0 (FR-044); `wrongAmmunitionType` does not apply. For a weapon without Ammunition a carried supply MUST be ignored and `AmmunitionAfter` MUST be absent. | SRD 5.2.1, "Ammunition", p. 89 | unhappy (state error) |

### 4.5 Light property and two-weapon attacks

The per-turn record is the immutable `WeaponTurnState`. A new combatant turn starts from the zero `WeaponTurnState` (the zero value is the initial state); the engine never resets it (section 9).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-057 | `LightExtraAttackPlanner.plan` MUST allow the extra attack when, this turn, the Attack action was taken, an attack was made with a Light weapon, the extra-attack weapon has Light and is a different weapon (`WeaponInstanceKey`) from a Light weapon already attacked with, and no Light extra attack was made yet; it returns `LightExtraAttackAllowed` with FR-063's cost. | SRD 5.2.1, "Light", p. 89 | happy |
| FR-058 | Without the Attack action taken this turn the extra attack MUST be blocked with `attackActionNotTaken`. | SRD 5.2.1, "Light" ("When you take the Attack action on your turn"), p. 89 | unhappy (state error) |
| FR-059 | A weapon without Light MUST be blocked with `weaponNotLight` (a Longsword), even when a Light weapon attacked earlier. | SRD 5.2.1, "Light" ("a different Light weapon"), p. 89 | unhappy (rule interaction) |
| FR-060 | When no attack with a Light weapon is recorded this turn the extra attack MUST be blocked with `noLightWeaponAttackMade`. | SRD 5.2.1, "Light" ("attack with a Light weapon"), p. 89 | unhappy (state error) |
| FR-061 | When every recorded Light attack used the very weapon proposed, the extra attack MUST be blocked with `sameLightWeapon`; two daggers with different `WeaponInstanceKey`s are different weapons. When any recorded Light attack used another weapon it is allowed. | SRD 5.2.1, "Light" ("a different Light weapon"; the Shortsword and Dagger example), p. 89 | unhappy (boundary) |
| FR-062 | A second Light extra attack in one turn MUST be blocked with `alreadyMadeThisTurn`. | SRD 5.2.1, "Light" ("one extra attack"), p. 89; "Nick", p. 90 | unhappy (state error) |
| FR-063 | The cost of an allowed Light extra attack MUST be `ExtraAttackCost.spendsBonusAction`, except FR-064. Whether a Bonus Action is available is `combat`'s (section 9). | SRD 5.2.1, "Light" ("as a Bonus Action"), p. 89; "Bonus Action", p. 177 | happy |
| FR-064 | When the mastery property in play for the extra-attack weapon is Nick and `Permits` holds for it, the cost MUST be `partOfAttackAction`. With a Nick weapon the character cannot use mastery with, or a non-Nick weapon, it stays `spendsBonusAction`. | SRD 5.2.1, "Nick", p. 90 | unhappy (rule interaction) |
| FR-065 | By default the extra attack's damage MUST NOT add the ability modifier unless it is negative: `omitUnlessNegative` with +3 or 0 adds nothing, with -1 adds -1. | SRD 5.2.1, "Light" ("you don't add your ability modifier ... unless that modifier is negative"), p. 89 | unhappy (boundary) |
| FR-066 | When the host reports `ExtraAttackDamageEffectActive()` (the Two-Weapon Fighting effect of `feats-behaviour.md`: "add your ability modifier to the damage of that attack if you aren't already adding it"), the planner MUST give `alwaysAdd` whatever the sign. With `NoExtraAttackDamageEffect` it gives FR-065's rule. `feats-behaviour.md` owns the feat and maps to this input. | SRD 5.2.1, "Two-Weapon Fighting", p. 88 | unhappy (rule interaction) |
| FR-067 | Reasons that apply together MUST be reported in the order: `attackActionNotTaken`, `weaponNotLight`, `alreadyMadeThisTurn`, `noLightWeaponAttackMade`, `sameLightWeapon`. | `doc/rules/architecture.md` §6 (package design) | unhappy (rule interaction) |
| FR-068 | `WeaponTurnState` MUST record the Attack action taken, each weapon attacked with (its key and whether it has Light), the Light extra attack made and the Cleave extra attack made; each recording operation MUST return a new instance and leave the receiver unchanged. The host calls `RecordAttackAction` when its Attack action starts; no Attack-action attack count is tracked and no `combat` type is read. | `doc/rules/architecture.md` §3 (state transitions return new instances) | unhappy (immutability) |
| FR-069 | The zero `WeaponTurnState` (its initial state) MUST record nothing. | SRD 5.2.1, "Light", "Cleave", "Nick" ("once per turn"), pp. 89-90 | unhappy (empty and minimal case) |
| FR-070 | Recording an attack with a non-Light weapon MUST NOT make a Light extra attack possible; with a Light weapon (and the Attack action) it MUST. | SRD 5.2.1, "Light", p. 89 | unhappy (rule interaction) |

### 4.6 Mastery effects

`OfferMastery(facts MasteryAttackFacts) (MasteryOffer, error)` takes a `MasteryAttackFacts` (weapon and `ItemID`, `WeaponMasteryAccess`, `MasteryInPlay`, `MasteryAttackOutcome`, `WeaponAttackMode`, the ability modifier of the attack roll, `ProficiencyBonus`, the target's `CreatureSize`, damage dealt as a whole number of 0 or more, `WeaponTurnState`) and returns a closed `MasteryOffer`; the `error` is non-nil only for a negative `DamageDealt` (an `*InvalidError`, FR-107). `OfferMastery` is pure and takes no `ctx`. "Damage dealt" is after Resistance, Vulnerability and Immunity (section 9). `combat` decides a forced miss (`attacks.md` FR-091) and the underwater automatic miss (`attacks.md` FR-041); this spec does not depend on them.

`MasteryAttackOutcome` is spec-local and built by the host from the published `AttackRollResult` (FR-013, no parallel enum, `shared-types-and-ownership.md` FR-004): `MasteryAttackRolled(roll)` for a roll that stands; `MasteryAttackForcedMiss(roll)` for a roll a rule turned into a miss whatever it showed (target not at the targeted location); `MasteryAttackUnrolledMiss` for a miss decided without a roll (underwater automatic miss). A **hit** (including a Critical Hit) is `MasteryAttackRolled` with `roll.Hit()` (FR-118); every other outcome is a miss.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-071 | Every effect MUST be reported as an offer the host may take or leave; `OfferMastery` MUST NOT apply damage, movement, a condition or a Speed change. | SRD 5.2.1, "Mastery Properties" (Cleave to Vex), p. 90 | happy |
| FR-072 | When `Permits` is false the offer MUST be `MasteryNone(featureNotAvailable)` whatever the attack did. Otherwise reasons are checked in order: `attackDidNotQualify`, `notMeleeAttack`, `alreadyUsedThisTurn`, `targetTooLarge`, `noDamageDealt`. Each belongs to one or two properties (Cleave: the first three; Push: `attackDidNotQualify`, `targetTooLarge`; Slow and Vex: `attackDidNotQualify`, `noDamageDealt`), so only pairs that can co-occur have a precedence row. | SRD 5.2.1, "Weapons" (Mastery), p. 89 | unhappy (rule interaction) |
| FR-073 | Mastery properties MUST apply to any attack with the weapon (melee, thrown or ranged) unless the property says melee (Cleave): a thrown Handaxe, Sling, Heavy Crossbow and Pike all trigger on a hit. | SRD 5.2.1, "Mastery Properties" ("with this weapon"), p. 90 | unhappy (rule interaction) |
| FR-074 | **Cleave.** On a hit with a melee attack roll the offer MUST be `CleaveOffer` when the Cleave extra attack has not been made this turn. It carries a melee weapon attack with the same weapon (`meleeWeaponAttack`), its `WeaponDamage` (FR-054) and `omitUnlessNegative` (FR-065's rule). | SRD 5.2.1, "Cleave", p. 90 | happy |
| FR-075 | Cleave on a miss MUST be `MasteryNone(attackDidNotQualify)`; on a hit that was thrown or ranged, `MasteryNone(notMeleeAttack)`. | SRD 5.2.1, "Cleave" ("hit a creature with a melee attack roll"), p. 90 | unhappy (rule interaction) |
| FR-076 | `CleaveOffer.ValidateTarget(p CleaveTargetPlacement, reach ReachInFeet) CleaveTargetResult`, with `p` built by `NewCleaveTargetPlacement(distanceFromFirst, distanceFromAttacker int)`, MUST return `CleaveTargetAccepted` for a second creature at most 5 feet from the first and at most the attacker's reach from the attacker (5 and 5 yes; 6 from the first no; Reach weapon and reach 10: 10 yes, 11 no), otherwise `CleaveTargetRejected(failure)` with `CleaveTargetFailure` `notWithinFiveFeetOfFirst`, `notWithinReach` or `both`. A negative distance MUST make `NewCleaveTargetPlacement` return an `*InvalidError`. | SRD 5.2.1, "Cleave" ("within 5 feet of the first that is also within your reach"), p. 90 | unhappy (boundary) |
| FR-077 | The Cleave extra attack MUST be made only once per turn: after `RecordCleaveAttack` the next offer MUST be `MasteryNone(alreadyUsedThisTurn)`, whether or not it hit. | SRD 5.2.1, "Cleave" ("only once per turn"), p. 90 | unhappy (state error) |
| FR-078 | **Graze.** On a miss with a roll that stands (`MasteryAttackRolled`, `Hit()` false, including a natural 1) the offer MUST be `GrazeOffer(amount, damageType)` with the weapon's damage type and an amount equal to the ability modifier of the attack roll, minimum 0 (+3 gives 3, 0 gives 0, -1 gives 0). | SRD 5.2.1, "Graze", p. 90; "Damage Rolls", p. 16 | unhappy (boundary) |
| FR-079 | Graze MUST NOT be offered on a hit or Critical Hit, nor when no creature was missed: `MasteryAttackForcedMiss` (a forced miss wins even over a natural 20) and `MasteryAttackUnrolledMiss` are both `attackDidNotQualify` (section 9). | SRD 5.2.1, "Graze" ("misses a creature"), p. 90 | unhappy (rule interaction) |
| FR-080 | Graze damage MUST be only the ability modifier: no other bonus, no dice, no Critical Hit doubling. The offer exposes `amount` and `damageType`; the host turns them into damage. | SRD 5.2.1, "Graze" ("can be increased only by increasing the ability modifier"), p. 90 | unhappy (rule interaction) |
| FR-081 | **Nick.** `offer` for a Nick weapon MUST return `MasteryNone(attackDidNotQualify)`: Nick acts only through FR-064. | SRD 5.2.1, "Nick", p. 90 | unhappy (rule interaction) |
| FR-082 | **Push.** On a hit against a target Large or smaller (Tiny, Small, Medium, Large) the offer MUST be `PushOffer` with a maximum of 10 feet, straight away from the attacker. | SRD 5.2.1, "Push", p. 90; "Creature Size", p. 14 | happy |
| FR-083 | Push on a hit against a Huge or Gargantuan target MUST be `MasteryNone(targetTooLarge)`; on a miss `attackDidNotQualify`. | SRD 5.2.1, "Push" ("if it is Large or smaller"), p. 90 | unhappy (boundary) |
| FR-084 | `PushOffer.ChooseDistance(feet int) PushDistanceResult` MUST return `PushDistanceAccepted(PushMovement(feet))` for 0 to 10, and `PushDistanceRejected(reason)` with `PushDistanceRejection` (`negative`, `aboveMaximum`) below 0 or above 10. It MUST NOT return an error or panic: the distance is a table choice. | SRD 5.2.1, "Push" ("up to 10 feet"), p. 90; `doc/rules/architecture.md` §6 | unhappy (boundary) |
| FR-085 | **Sap.** On a hit the offer MUST be `SapOffer`: a `Disadvantage` circumstance on the target's next attack roll, ending at the start of the attacker's next turn or when that roll is made. No damage is required. | SRD 5.2.1, "Sap", p. 90 | happy |
| FR-086 | **Slow.** On a hit that dealt damage above 0 the offer MUST be `SlowOffer` with a `SpeedReduction` of 10 feet until the start of the attacker's next turn. | SRD 5.2.1, "Slow", p. 90 | happy |
| FR-087 | Slow on a hit that dealt 0 damage (for example against Immunity) MUST be `MasteryNone(noDamageDealt)`. | SRD 5.2.1, "Slow" ("and deal damage to it"), p. 90; "Immunity", p. 183 | unhappy (boundary) |
| FR-088 | `SpeedReduction.Combine(other)` MUST never exceed 10 feet: two or more Slow hits still reduce Speed by 10 feet. | SRD 5.2.1, "Slow" ("the Speed reduction doesn't exceed 10 feet"), p. 90 | unhappy (rule interaction) |
| FR-089 | `SpeedReduction.ApplyTo(speed int) (int, error)` MUST return Speed reduced by 10 feet, minimum 0 (30 gives 20, 5 gives 0, 0 gives 0); a negative Speed MUST be an `*InvalidError`. | SRD 5.2.1, "Slow", p. 90; package design (a Speed is never negative) | unhappy (boundary) |
| FR-090 | **Topple.** On a hit, whatever the target's size, the offer MUST be `ToppleOffer` with a Constitution saving throw and DC 8 plus the ability modifier of the attack roll plus the attacker's Proficiency Bonus (+3 and +2 gives 13; -1 and +2 gives 9; +5 and +6 gives 19). | SRD 5.2.1, "Topple", p. 90 | unhappy (boundary) |
| FR-091 | Topple on a miss MUST be `MasteryNone(attackDidNotQualify)`; no DC is computed. | SRD 5.2.1, "Topple" ("If you hit a creature"), p. 90 | unhappy (rule interaction) |
| FR-092 | **Vex.** On a hit that dealt damage above 0 the offer MUST be `VexOffer`: an `Advantage` circumstance on the attacker's next attack roll against that creature, ending at the end of the attacker's next turn or when that roll is made. | SRD 5.2.1, "Vex", p. 90 | happy |
| FR-093 | Vex on a hit that dealt 0 damage MUST be `MasteryNone(noDamageDealt)`; on a miss `attackDidNotQualify`. | SRD 5.2.1, "Vex" ("and deal damage to the creature"), p. 90 | unhappy (boundary) |
| FR-094 | Expiry MUST be exposed as data: Sap and Slow `startOfAttackersNextTurn`, Vex `endOfAttackersNextTurn`; Sap and Vex also expire when the one next attack roll they modify is made (`whenNextAttackRollMade`), Slow does not. | SRD 5.2.1, "Sap", "Slow", "Vex", p. 90 | happy |
| FR-095 | Hit and miss for every effect MUST come from `AttackRollResult.Hit()` of the carried outcome (natural 1 a miss, Critical Hit a hit, as `d20` reported them); effects MUST NOT re-derive it from the d20 or totals. | SRD 5.2.1, "Rolling 20 or 1", p. 7; `d20-tests.md` FR-029, FR-030; `shared-types-and-ownership.md` FR-004 | unhappy (boundary) |
| FR-096 | A Critical Hit MUST give the same offer as an ordinary hit for Cleave, Push, Sap, Topple, Slow and Vex. | SRD 5.2.1, "Mastery Properties" ("hit"), p. 90 | unhappy (boundary) |
| FR-118 | `MasteryAttackOutcome` MUST be a closed family of exactly `MasteryAttackRolled(roll)`, `MasteryAttackForcedMiss(roll)` and `MasteryAttackUnrolledMiss`, and be a hit exactly when it is `MasteryAttackRolled` with `roll.Hit()` true; every other outcome (a forced miss wins over a natural 20) is a miss. | Package design; `shared-types-and-ownership.md` FR-004 | unhappy (rule interaction) |
| FR-119 | A `MasteryAttackUnrolledMiss` (such as the underwater automatic miss) MUST give `MasteryNone(attackDidNotQualify)` for every property that needs a hit or a miss with a roll. | SRD 5.2.1, "Graze" ("misses a creature"), "Underwater Combat", p. 16, p. 90 | unhappy (rule interaction) |

### 4.7 The Topple save use case and cross-cutting rules

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-097 | `ResolveToppleSave` MUST make the target's Constitution saving throw as one `d20.SavingThrow` check. For a `RollAttempt` (modifiers and circumstances built by the host) it resolves with the pure `d20.ResolveCheck` against the `ToppleOffer`'s `d20.DifficultyClass` (FR-122) and MUST add no arithmetic of its own: a total equal to or above the DC succeeds (`d20`'s comparison). | SRD 5.2.1, "Topple" (Constitution saving throw), p. 90; "D20 Tests", p. 6 | happy |
| FR-098 | On a failed save (`CheckResult` outcome `failure`) the result MUST be `ToppleSaveTargetProne` describing Prone as data (`ConditionKind.prone`) without applying it; on success `ToppleSaveTargetResisted` with no effect. | SRD 5.2.1, "Topple" ("On a failed save, the creature has the Prone condition"), p. 90 | happy |
| FR-099 | A `ChooseToFail` or `AutomaticFailure` attempt MUST resolve through `d20.FailWithoutRoll(d20.SavingThrow, reason)` to a failed save, `ToppleSaveTargetProne`, with no d20 rolled and the `ForcedFailureReason` carried. | SRD 5.2.1, "Saving Throws", p. 7 | unhappy (rule interaction) |
| FR-100 | Both results MUST carry `ToppleSaveDetail(dc, basis)` with the `d20.DifficultyClass` and a closed `ToppleSaveBasis`: `ToppleSaveRolled(check)` (a `d20.CheckResult`) or `ToppleSaveForced(failure)` (a `d20.ForcedFailureResult`). | `doc/rules/architecture.md` §6 | happy |
| FR-101 | A failure of the dice port MUST be returned by `ResolveToppleSave` as the `*srderr.PortError` (port `dice`, wrapped once by `dice.Roll`; `errors.Is` reaches the roller's own error, `srderr.ErrPortPanic` for a panicking roller and `srderr.ErrContractViolation` for a result to another expression), with a zero result and no partial result, and be recorded once (span error status and one increment of the operations counter with `srd5e.outcome` `failure` and `error.type` = `srderr.ErrorType(err)`, i.e. `dice_roller_error`, no `srd5e.character.result`, no log). | `doc/rules/architecture.md` §4, §6; `doc/rules/observability.md` §7 | unhappy (failure of collaborator) |
| FR-102 | `ResolveToppleSave` MUST NOT build, hold or call `d20.Resolver` or any other use case. It composes the pure functions of `d20` (FR-122) and records only its own signals: exactly one `srd5e.character.operations` increment per call and no `srd5e.d20_test.*` signal. | `shared-types-and-ownership.md` FR-044; `doc/rules/architecture.md` §4; `doc/rules/observability.md` §7 | unhappy (rule interaction) |
| FR-103 | The constructor MUST be `NewToppleSaveResolver(roller Roller, opts ...Option) (ToppleSaveResolver, error)` with `WithTelemetry(telemetry.Telemetry)`; `Roller` is `character.Roller`, owned by `character-aggregate.md` and only used here (one-method `Roll(dice.Expression) (dice.Result, error)`; any `dice.Roller` satisfies it), a nil `roller` returns an `*InvalidError`, omitting `WithTelemetry` MUST work (no-op telemetry) and MUST NOT change any result. | `doc/rules/architecture.md` §4; `doc/rules/observability.md` §4 | happy |
| FR-104 | With fixed rolls, `OfferMastery` and every planner MUST return the same value for the same input on every run; only `ResolveToppleSave` draws a die: one d20 for a rolling attempt (two with Advantage or Disadvantage), none for a forced failure. | `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-105 | Every value object and closed result MUST be immutable and compare by value; every collection on a plan, offer, choice or result MUST be a copy on construction and on read, so mutating it leaves the value unchanged. | `doc/rules/architecture.md` §3, §4 | unhappy (immutability) |
| FR-106 | No planner or effect service MUST alter its inputs (turn state, choice, weapon, supply); a state change returns a new value. | `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-107 | Host-supplied numbers MUST be validated, returning an `*InvalidError` (matching `srderr.ErrInvalidInput`) that names the field: `NewAmmunitionSupply` pieces 0 or more, `NewCleaveTargetPlacement` distances 0 or more, `DamageDealt` 0 or more (checked by `OfferMastery`), `NewReachInFeet` 1 or more (FR-121). These typed numbers have no result type (the host computes them; they are not decoded content); the `MustX` helpers panic with the same error. `allowedCount` is validated by FR-028. | `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |
| FR-108 | The domain types and services MUST be silent: no logging, metric or span, no goroutine or channel, no `context`, `time`, `math/rand` or `sync` import, no telemetry import. Only `ResolveToppleSave` is instrumented. | `doc/rules/observability.md` §3; `doc/rules/architecture.md` §3, §9 | happy |
| FR-109 | No file of this spec MAY import a `combat` file or name a `combat` type, and no file of package `character` MAY import another context's package except `content` and `d20`, besides the kernel packages (`dice`, `ability`, `modifier`, `srderr`, `telemetry`); no non-test file imports a `*test` package. The spec sits in `character` order (`shared-types-and-ownership.md` FR-016). | `doc/rules/architecture.md` §2; `shared-types-and-ownership.md` FR-013, FR-016 | unhappy (architecture) |
| FR-110 | Package `character` MUST export (no facade or re-export package), each identifier once with a doc comment, every type of section 5 including every type a public operation returns or takes and the slot of FR-123; no generic variant name (`Applied`, `Rejected`, `NotFound`) is exported. A test importing only public packages (`character`, `content`, `d20`) builds each command and plan and switches exhaustively over each closed result. | `doc/rules/architecture.md` §4, §7, §9 | happy |
| FR-111 | `ResolveToppleSave` telemetry MUST follow the shared `character` signals (`character-aggregate.md` section 7): span `srd5e.character.resolve_topple_save`; the shared counter `srd5e.character.operations` and histogram `srd5e.character.duration` with the new `srd5e.character.operation` value `resolve_topple_save` (the histogram's only attribute, `character-aggregate.md` FR-103); `srd5e.outcome` (`failure` for `target_prone`, `success` for `target_resisted`) from the domain result; `srd5e.character.result` (counter and span only) exactly `target_prone` or `target_resisted`. Both are normal rule outcomes: span status not error, no `error.type`, no log. A roller failure is recorded once (span error status; counter with `srd5e.outcome` `failure` and `error.type` `dice_roller_error` from `srderr.ErrorType`, no `srd5e.character.result`), not logged. | `doc/rules/observability.md` §5, §6, §7, §9, §10; `character-aggregate.md` FR-070, FR-071, FR-088; `shared-types-and-ownership.md` FR-041, FR-044 | unhappy (failure of collaborator) |
| FR-112 | Exported identifiers MUST NOT name any consuming project, and no type MUST exist only to serve one class: the five classes reach this spec through `WeaponMasteryChoice`, `MasteryEligibility`, `MasteryRetrainRule`, `MasterySwap` and `WeaponMasteryAccess`. | `AGENTS.md` ("No consumer-specific code"); `doc/rules/architecture.md` §10 | unhappy (architecture) |
| FR-120 | `WeaponProficienciesFromClassData([]WeaponProficiency)` (`content-classes.md` FR-021) MUST build one rule per element: `CategoryProficiency(category)` without `requiresAnyProperty`, else `CategoryWithAnyPropertyProficiency(category, properties)`; an empty slice gives no rule (FR-010). It does not validate the data. | SRD 5.2.1, "Core Monk Traits", p. 49; "Core Rogue Traits", p. 61; `content-classes.md` FR-021 | happy |
| FR-121 | `NewReachInFeet(feet int) (ReachInFeet, error)` MUST hold a whole number of feet of at least 1, otherwise return an `*InvalidError` naming `feet` (FR-107); `MustReachInFeet` panics with it. The zero `ReachInFeet` is invalid except as an input field, where it reads as the default 5 feet (FR-039). The SRD states no maximum. | Package design; SRD 5.2.1, "Reach", p. 186 | unhappy (invalid input) |
| FR-122 | `ResolveToppleSave` MUST obtain its D20 Test result only from the pure functions of `d20` (`d20-tests.md` FR-069), not from `d20.Resolver` (a use case): call `modifier.Resolve` once on the attempt's circumstances, make exactly one `dice.Roll` request for `dice.NewExpression(mode.DiceCount(), dice.D20(), 0)` (`1d20` normal, `2d20` under Advantage or Disadvantage; `shared-types-and-ownership.md` FR-001, `dice-and-rolls.md` FR-050), build the `d20.Roll` from the mode and faces with `d20.NewRoll` (`d20-tests.md` FR-055) and call `d20.ResolveCheck(d20.SavingThrow, roll, modifiers, dc)`. For `ChooseToFail` and `AutomaticFailure` make no request and call `d20.FailWithoutRoll`. | `shared-types-and-ownership.md` FR-044; `d20-tests.md` FR-035, FR-036, FR-063, FR-069; `resolve-d20-test-use-case.md` FR-006, FR-007 (same request shape, not a call) | happy |

### Case tables

Each row is one table-driven test (`doc/rules/testing.md` §5, §6).

**Table 4.1a. Weapon proficiency and property rules ignoring parameters (FR-008 to FR-010, FR-012, FR-013).** Source: SRD 5.2.1, "Weapon Proficiency", "Properties", pp. 89-90; "Core Rogue Traits", p. 61; "Core Monk Traits", p. 49; Weapons table, p. 91.

| Label | Rules | Weapon | Expected | Path |
|---|---|---|---|---|
| Simple covers Club | Simple | Club (Simple Melee, Light) | proficient | happy |
| Simple excludes Longsword | Simple | Longsword (Martial Melee, Versatile) | notProficient | unhappy (boundary) |
| Martial covers Longsword | Martial | Longsword | proficient | happy |
| Rogue: Simple Dart | Simple; Martial with Finesse or Light | Dart (Simple Ranged, Finesse) | proficient | happy |
| Rogue: Rapier | same | Rapier (Martial, Finesse) | proficient | unhappy (boundary) |
| Rogue: Scimitar (both) | same | Scimitar (Martial, Finesse, Light) | proficient | unhappy (boundary) |
| Rogue: Greatsword | same | Greatsword (Martial, Heavy, Two-Handed) | notProficient | unhappy (rule interaction) |
| Rogue: Whip (Finesse, Reach) | same | Whip (Martial, Finesse, Reach) | proficient | unhappy (boundary) |
| Monk: Whip | Simple; Martial with Light | Whip | notProficient | unhappy (boundary) |
| Monk: Shortsword | same | Shortsword (Martial, Finesse, Light) | proficient | happy |
| Monk: Longbow | same | Longbow (Martial Ranged) | notProficient | unhappy (rule interaction) |
| No rule | (empty) | Dagger | notProficient | unhappy (empty case) |
| Monster | AnyStatBlockWeapon | Pike | proficient | happy |
| Thrown range 20/60 | Simple with Thrown | Dagger (Thrown 20/60) | proficient | happy |
| Thrown range 30/120 | Simple with Thrown | Javelin (Thrown 30/120) | proficient | unhappy (rule interaction) |
| Versatile damage differs | Martial with Versatile | Longsword (Versatile 1d10) | proficient | unhappy (rule interaction) |
| Versatile on a Simple weapon | Martial with Versatile | Spear (Simple, Versatile 1d8) | notProficient | unhappy (boundary) |

**Table 4.3a. Weapon Mastery choices (FR-014 to FR-029, FR-125).** Source: SRD 5.2.1, "Level 1: Weapon Mastery", pp. 29, 48, 54, 58-59, 62. For extend rows `Current` is Longsword, Longbow, Dagger and `proposed` the full list. For `change` rows `current` is the same three kinds (Paladin and Ranger rows: two) and the input is `swaps` plus a `MasteryChangeMoment`.

| Label | Eligibility, count | Candidates | Expected | Path |
|---|---|---|---|---|
| Barbarian level 1 | Simple or Martial Melee, 2 | Greataxe, Handaxe | accepted | happy |
| Barbarian ranged | same | Greataxe, Shortbow | `notMelee` (Shortbow) | unhappy (invalid input) |
| Barbarian one kind | same | Greataxe | `wrongCount` | unhappy (boundary) |
| Barbarian three kinds | same | Greataxe, Handaxe, Mace | `wrongCount` | unhappy (boundary) |
| Barbarian empty | same | (none) | `wrongCount` | unhappy (empty case) |
| Duplicate | same | Handaxe, Handaxe | `duplicateWeapon` | unhappy (invalid input) |
| Fighter level 1 | Simple or Martial, 3 | Longsword, Longbow, Dagger | accepted | happy |
| Fighter level 4, one more | extend, 4 | proposed Longsword, Longbow, Dagger, Spear | accepted, order as proposed | happy |
| Fighter level 4, reordered | extend, 4 | proposed Spear, Dagger, Longbow, Longsword | accepted (set-based containment), order as proposed | unhappy (boundary) |
| Fighter drops one | extend, 4 | proposed Longsword, Longbow, Spear, Mace (no Dagger) | `dropsExistingMastery` (Dagger) | unhappy (rule interaction) |
| Fighter extend, too few | extend, 4 | proposed Longsword, Longbow, Dagger | `wrongCount` | unhappy (boundary) |
| Paladin proficient | proficient, 2 | Longsword, Javelin | accepted | happy |
| Rogue unproficient | proficient (Rogue rules), 2 | Dagger, Greatsword | `notProficient` (Greatsword) | unhappy (invalid input) |
| Ranger missing weapon | proficient, 2 | Longbow, (no definition) | `weaponNotResolved` | unhappy (missing content) |
| Missing content first | proficient (Rogue), 2 | Greatsword, Greatsword, (no definition) | `weaponNotResolved` only; no `duplicateWeapon`, `notProficient` or `wrongCount` | unhappy (missing content) |
| Several rule problems | proficient (Rogue), 2 | Greatsword, Greatsword | `notProficient` (first Greatsword), `duplicateWeapon` (second), in that order | unhappy (invalid input) |
| Fighter swap, Long Rest | `ReplaceOneKind`, `longRest` | swap Dagger for Rapier | accepted; Rapier replaces Dagger in place | happy |
| Fighter swap, two | `ReplaceOneKind`, `longRest` | swap two kinds | `changesMoreThanOne` | unhappy (boundary) |
| Fighter swap, Short Rest | `ReplaceOneKind`, `shortRest` | swap one | `wrongMoment`; `current` not altered | unhappy (state error) |
| Fighter swap, not chosen | `ReplaceOneKind`, `longRest` | remove Greataxe (not chosen) | `notCurrentlyChosen` (Greataxe) | unhappy (invalid input) |
| Fighter swap, already chosen | `ReplaceOneKind`, `longRest` | remove Dagger, add Longsword | `alreadyChosen` (Longsword) | unhappy (invalid input) |
| Paladin swap, same kind twice | `ReplaceAnyKinds`, `longRest` | remove Dagger twice | `duplicateWeapon` (Dagger) | unhappy (invalid input) |
| Paladin swap both | `ReplaceAnyKinds`, `longRest` | swap both kinds | accepted | happy |
| Ranger swap, Short Rest | `ReplaceAnyKinds`, `shortRest` | swap one | `wrongMoment` | unhappy (state error) |
| No change | `ReplaceOneKind`, `longRest` | no swaps | accepted, equal to `current` | unhappy (empty case) |
| Count 0 | any, `allowedCount` 0 | any | `*InvalidError` on `allowedCount` | unhappy (invalid input) |
| Short Rest and two swaps | `ReplaceOneKind`, `shortRest` | swap two kinds | `wrongMoment` only | unhappy (rule interaction) |
| Two swaps and not chosen | `ReplaceOneKind`, `longRest` | remove Greataxe (not chosen) add Rapier; remove Dagger add Spear | `changesMoreThanOne`, then `notCurrentlyChosen` (Greataxe) | unhappy (rule interaction) |
| Unresolved add first | `ReplaceAnyKinds`, `longRest` | remove Greataxe (not chosen), add (no definition) | `weaponNotResolved` only | unhappy (missing content) |
| Extend, unresolved | extend, 4 | proposed Longsword, Longbow, Dagger, (no definition) | `weaponNotResolved` only | unhappy (missing content) |
| Extend, wrong count and dropped | extend, 4 | proposed Longsword, Longbow, Spear (no Dagger) | `wrongCount`, then `dropsExistingMastery` (Dagger) | unhappy (rule interaction) |

**Table 4.4a. Ability, Heavy and reach (FR-031, FR-032, FR-036, FR-037, FR-039, FR-040).** Source: SRD 5.2.1, pp. 7, 15, 89, 90.

| Label | Weapon, mode | Preference | Str / Dex score | Attacker reach | Expected ability | Heavy Disadvantage | Reach in plan | Path |
|---|---|---|---|---|---|---|---|---|
| Longsword melee | Longsword, melee | default | 14 / 10 | 5 | Strength | no | 5 | happy |
| Rapier, chosen Dexterity | Rapier (Finesse), melee | Dexterity | 8 / 16 | 5 | Dexterity | no | 5 | happy |
| Rapier, chosen Strength | Rapier, melee | Strength | 16 / 8 | 5 | Strength | no | 5 | happy |
| Rapier, default | Rapier, melee | default | 8 / 16 | 5 | Strength | no | 5 | unhappy (boundary) |
| Dagger thrown, Finesse | Dagger, thrown | Dexterity | 10 / 16 | 5 | Dexterity | no | range 20/60 | happy |
| Handaxe thrown (no Finesse) | Handaxe, thrown | default | 16 / 10 | 5 | Strength | no | range 20/60 | unhappy (rule interaction) |
| Dart thrown | Dart (Ranged, Finesse), thrown | default | 10 / 14 | 5 | Dexterity | no | range 20/60 | happy |
| Shortbow ranged | Shortbow, ranged | default | 10 / 14 | 5 | Dexterity | no | range 80/320 | happy |
| Greatsword Str 13 | Greatsword (Heavy, Melee) | default | 13 / 8 | 5 | Strength | no | 5 | unhappy (boundary) |
| Greatsword Str 12 | Greatsword | default | 12 / 20 | 5 | Strength | yes | 5 | unhappy (boundary) |
| Heavy Crossbow Dex 13 | Heavy Crossbow (Heavy, Ranged) | default | 8 / 13 | 5 | Dexterity | no | range 100/400 | unhappy (boundary) |
| Heavy Crossbow Dex 12 | Heavy Crossbow | default | 20 / 12 | 5 | Dexterity | yes | range 100/400 | unhappy (boundary) |
| Longsword Str 1 | Longsword | default | 1 / 10 | 5 | Strength | no | 5 | unhappy (empty case) |
| Glaive reach 5 | Glaive (Heavy, Reach, Two-Handed), melee, two hands | default | 14 / 10 | 5 | Strength | no | 10 | happy |
| Glaive reach 10 | Glaive | default | 14 / 10 | 10 | Strength | no | 15 | unhappy (boundary) |
| Whip (Finesse, Reach) | Whip, melee | Dexterity | 8 / 15 | 5 | Dexterity | no | 10 | unhappy (rule interaction) |

**Table 4.4b. Versatile, Two-Handed, Ammunition, Loading, improvised uses and block order (FR-034, FR-043 to FR-056, FR-114 to FR-116, FR-124).** Source: SRD 5.2.1, pp. 89, 90, 91.

| Label | Weapon, mode, handling | Expected | Path |
|---|---|---|---|
| Longsword two hands | Longsword, melee, `twoHanded` | damage 1d10 Slashing | happy |
| Longsword one hand | Longsword, melee, `oneHanded` | damage 1d8 Slashing | happy |
| Spear thrown, two hands | Spear, thrown, `twoHanded` | damage 1d6 Piercing | unhappy (boundary) |
| Spear melee, two hands | Spear, melee, `twoHanded` | damage 1d8 Piercing | happy |
| Mace two hands | Mace, melee, `twoHanded` | 1d6 Bludgeoning, not blocked | unhappy (rule interaction) |
| Greatsword one hand | Greatsword, melee, `oneHanded` | blocked `needsTwoHands` | unhappy (state error) |
| Lance, mounted | Lance, melee, `oneHanded`, `exemptionApplies` | planned | unhappy (rule interaction) |
| Lance, not mounted | Lance, melee, `oneHanded`, `noExemption` | blocked `needsTwoHands` | unhappy (state error) |
| Greatsword exemption | Greatsword (Two-Handed, no condition), `oneHanded`, `exemptionApplies` | blocked `exemptionNotApplicable` | unhappy (invalid input) |
| Shortbow 0 arrows | Shortbow, ranged, supply 0 | blocked `noAmmunition` | unhappy (boundary) |
| Shortbow 1 arrow | Shortbow, ranged, supply 1 | planned, `ammunitionAfter` 0 | unhappy (boundary) |
| Pistol, no free hand | Pistol (one-handed), ranged, supply 5, `noFreeHand` | blocked `noFreeHandToLoad` | unhappy (state error) |
| Pistol, free hand | same, `freeHandAvailable` | planned | happy |
| Light Crossbow, no free hand | Light Crossbow (Two-Handed), `twoHanded`, `noFreeHand` | planned | unhappy (rule interaction) |
| Hand Crossbow second shot | Hand Crossbow, `freeHandAvailable`, `alreadyFiredThisActivation` | blocked `loadingLimit` | unhappy (state error) |
| Hand Crossbow first shot | Hand Crossbow, `freeHandAvailable`, `ready` | planned, `loadingAfter` `alreadyFiredThisActivation` | happy |
| Shortbow second shot | Shortbow (no Loading), `alreadyFiredThisActivation` | planned, `loadingAfter` `alreadyFiredThisActivation` (input unchanged) | unhappy (rule interaction) |
| Shortbow first shot | Shortbow (no Loading), `ready` | planned, `loadingAfter` `ready` | unhappy (rule interaction) |
| Sling in melee | Sling, melee | blocked `improvisedUse` | unhappy (rule interaction) |
| Club thrown | Club (no Thrown), thrown | blocked `improvisedUse` | unhappy (rule interaction) |
| Dart ranged | Dart (Ranged, Thrown, no Ammunition), `RangedAttackMode` | blocked `modeNotSupportedByWeapon` (host uses `ThrownAttackMode`) | unhappy (rule interaction) |
| Chosen ability without Finesse | Mace, `ChosenAbility(Strength)` | blocked `finesseNotAvailable` | unhappy (invalid input) |
| Longsword ranged | Longsword (Melee, no Ammunition), `RangedAttackMode` | blocked `modeNotSupportedByWeapon` | unhappy (rule interaction) |
| Shortbow thrown | Shortbow (Ranged, no Thrown), `ThrownAttackMode` | blocked `modeNotSupportedByWeapon` | unhappy (rule interaction) |
| Dart thrown keeps proficiency | Dart, `ThrownAttackMode`, proficient | planned, status `proficient` | unhappy (rule interaction) |
| Thrown, no free hand | Dagger, `ThrownAttackMode`, `noFreeHand` | planned | unhappy (rule interaction) |
| Shortbow, no supply | Shortbow, ranged, `ammunition` absent | blocked `noAmmunition` | unhappy (state error) |
| Longsword, supply ignored | Longsword, melee, supply 5 of Arrow | planned, `ammunitionAfter` absent | unhappy (rule interaction) |
| Shortbow, wrong ammunition | Shortbow (Arrow), ranged, supply 20 of Bolt | blocked `wrongAmmunitionType` | unhappy (invalid input) |
| Blowgun, Needle | Blowgun (Ammunition Needle, Loading), ranged, supply 5 of Needle, Dex 14, `ready` | planned; `WeaponFixedDamage` 1 Piercing, `neverAdd`; ability Dexterity | unhappy (rule interaction) |
| Blowgun, Strength choice | Blowgun, ranged, `ChosenAbility(Strength)` | blocked `finesseNotAvailable` | unhappy (invalid input) |
| Longsword damage is dice | Longsword, melee, `oneHanded` | `WeaponDiceDamage` 1d8 Slashing, `alwaysAdd` | happy |
| Order 1 | Sling, melee, `ChosenAbility(Strength)` | blocked `improvisedUse` (wins over `finesseNotAvailable`) | unhappy (rule interaction) |
| Order 2 | Longsword, ranged, `ChosenAbility(Strength)` | blocked `modeNotSupportedByWeapon` (wins over `finesseNotAvailable`) | unhappy (rule interaction) |
| Order 3 | Mace, melee, `ChosenAbility(Strength)`, `exemptionApplies` | blocked `finesseNotAvailable` (wins over `exemptionNotApplicable`) | unhappy (rule interaction) |
| Order 4 | Greatsword, `oneHanded`, `exemptionApplies` (no condition) | blocked `exemptionNotApplicable` (wins over `needsTwoHands`) | unhappy (rule interaction) |
| Order 5 | Longbow (Two-Handed), ranged, `oneHanded`, supply of Bolt | blocked `needsTwoHands` (wins over `wrongAmmunitionType`) | unhappy (rule interaction) |
| Order 6 | Shortbow (Arrow), ranged, supply of Bolt, 0 pieces | blocked `wrongAmmunitionType` (wins over `noAmmunition`) | unhappy (rule interaction) |
| Order 7 | Pistol (Bullet), ranged, supply of Bullet, 0 pieces, `noFreeHand` | blocked `noAmmunition` (wins over `noFreeHandToLoad`) | unhappy (rule interaction) |
| Order 8 | Pistol, ranged, supply of Bullet 5, `noFreeHand`, `alreadyFiredThisActivation` | blocked `noFreeHandToLoad` (wins over `loadingLimit`) | unhappy (rule interaction) |
| Heavy and non-proficient | Greatsword, Strength 12, not proficient | planned, Disadvantage and `notProficient` | unhappy (rule interaction) |

**Table 4.4c. Ammunition, Opportunity Attack reach, Cleave placement and Slow (FR-041, FR-045, FR-046, FR-076, FR-088, FR-089, FR-117).** Source: SRD 5.2.1, "Ammunition", p. 89; "Reach", p. 90; "Opportunity Attacks", p. 185; "Cleave", "Slow", p. 90.

| Label | Input | Result | Path |
|---|---|---|---|
| none used | `RecoverAfterFight(0)` | 0 | unhappy (empty case) |
| one used | `RecoverAfterFight(1)` | 0 | unhappy (boundary) |
| two used | `RecoverAfterFight(2)` | 1 | unhappy (boundary) |
| three used | `RecoverAfterFight(3)` | 1 | unhappy (boundary) |
| nine used | `RecoverAfterFight(9)` | 4 | happy |
| negative | `RecoverAfterFight(-1)` | `*InvalidError` | unhappy (invalid input) |
| one piece | 5 Arrow, `Expend()` | 4 Arrow | happy |
| last piece | 1 Arrow, `Expend()` | 0 Arrow | unhappy (boundary) |
| empty | 0 Arrow, `Expend()` | error matching `ErrSupplyEmpty` | unhappy (boundary) |
| Opportunity reach: Reach weapon | Glaive (Reach), attacker reach 5 | 10 | happy |
| Opportunity reach: Reach weapon, reach 10 | Whip (Reach), attacker reach 10 | 15 | unhappy (boundary) |
| Opportunity reach: No Reach property | Longsword, attacker reach 5 | 5 | unhappy (boundary) |
| Opportunity reach: No Reach property, reach 10 | Longsword, attacker reach 10 | 10 | unhappy (boundary) |
| Cleave placement: inside both | first 5, attacker 5, reach 5 | accepted | happy |
| Cleave placement: 6 from the first | first 6, attacker 5, reach 5 | rejected (not within 5 feet of the first) | unhappy (boundary) |
| Cleave placement: 6 from the attacker | first 5, attacker 6, reach 5 | rejected (not within reach) | unhappy (boundary) |
| Cleave placement: Reach weapon | first 5, attacker 10, reach 10 | accepted | unhappy (boundary) |
| Cleave placement: Reach weapon, beyond | first 5, attacker 11, reach 10 | rejected (not within reach) | unhappy (boundary) |
| Cleave placement: both fail | first 6, attacker 6, reach 5 | rejected naming both | unhappy (rule interaction) |
| Cleave placement: negative | first -1, attacker 5, reach 5 | `*InvalidError` from `NewCleaveTargetPlacement` | unhappy (invalid input) |
| Slow: once | reduction 10 | 10 | happy |
| Slow: twice | combine 10 and 10 | 10 | unhappy (rule interaction) |
| Slow: three times | combine three | 10 | unhappy (rule interaction) |
| Slow: Speed 30 | ApplyTo(30) | 20 | happy |
| Slow: Speed 5 | ApplyTo(5) | 0 | unhappy (boundary) |
| Slow: Speed 0 | ApplyTo(0) | 0 | unhappy (boundary) |
| Slow: Speed -1 | ApplyTo(-1) | `*InvalidError` | unhappy (invalid input) |

**Table 4.5a. The Light extra attack (FR-057 to FR-065, FR-067).** Source: SRD 5.2.1, "Light", p. 89; "Nick", p. 90. Rows "Order 0" to "Order 3" pin FR-067 pair by pair; `noLightWeaponAttackMade` and `sameLightWeapon` exclude each other, so no row pairs them.

| Label | State | Extra-attack weapon | Mastery access | Expected | Path |
|---|---|---|---|---|---|
| Shortsword then Dagger | Attack action taken; attacked with Shortsword (Light) | Dagger (Light, other instance) | none | allowed, `spendsBonusAction` | happy |
| Dagger then Dagger (two instances) | attacked with Dagger A | Dagger B | none | allowed, `spendsBonusAction` | unhappy (boundary) |
| Same dagger | attacked with Dagger A | Dagger A | none | blocked `sameLightWeapon` | unhappy (boundary) |
| Non-Light weapon | attacked with Shortsword | Longsword | none | blocked `weaponNotLight` | unhappy (rule interaction) |
| No Attack action | not taken; attacked with Shortsword | Dagger | none | blocked `attackActionNotTaken` | unhappy (state error) |
| Non-Light first | Attack action taken; attacked with Longsword | Dagger | none | blocked `noLightWeaponAttackMade` | unhappy (state error) |
| Already made | extra attack made | Dagger | none | blocked `alreadyMadeThisTurn` | unhappy (state error) |
| Nick in the chosen set | attacked with Shortsword; Dagger in the choice, mastery Nick | Dagger | Dagger chosen | allowed, `partOfAttackAction` | unhappy (rule interaction) |
| Nick not chosen | same | Dagger | Dagger not chosen | allowed, `spendsBonusAction` | unhappy (state error) |
| Non-Nick Light weapon | attacked with Dagger; Handaxe (Light) has Vex | Handaxe | Handaxe chosen | allowed, `spendsBonusAction` | unhappy (rule interaction) |
| Order 0: no Attack action beats not Light | no Attack action, Longsword | Longsword | none | blocked `attackActionNotTaken` (wins over `weaponNotLight`) | unhappy (rule interaction) |
| Order 1: not Light beats already made | Attack action taken; extra attack made; attacked with Shortsword | Longsword | none | blocked `weaponNotLight` (wins over `alreadyMadeThisTurn`) | unhappy (rule interaction) |
| Order 2: already made beats no Light attack | Attack action taken; extra attack made; no weapon attacked with | Dagger | none | blocked `alreadyMadeThisTurn` (wins over `noLightWeaponAttackMade`) | unhappy (rule interaction) |
| Order 3: already made beats same weapon | Attack action taken; extra attack made; attacked with Dagger A | Dagger A | none | blocked `alreadyMadeThisTurn` (wins over `sameLightWeapon`) | unhappy (rule interaction) |

| Label | Modifier | Damage source input | Damage modifier added | Path |
|---|---|---|---|---|
| positive omitted | +3 | `NoExtraAttackDamageEffect` (omitUnlessNegative) | none | unhappy (boundary) |
| zero omitted | 0 | `NoExtraAttackDamageEffect` | none | unhappy (boundary) |
| negative added | -1 | `NoExtraAttackDamageEffect` | -1 | unhappy (boundary) |
| feature adds positive | +3 | `ExtraAttackDamageEffectActive` (alwaysAdd) | +3 | unhappy (rule interaction) |
| feature adds negative | -2 | `ExtraAttackDamageEffectActive` | -2 | unhappy (rule interaction) |

**Table 4.6a. Mastery offers (FR-072 to FR-096, FR-118, FR-119).** Every row repeats: mastery usable unless stated; ability modifier +3; Proficiency Bonus +2; target Medium; delivery melee. Source: SRD 5.2.1, "Mastery Properties", p. 90. Outcome shorthand: "hit" is `MasteryAttackRolled` with a hit; "critical" the same with `IsCriticalHit()`; "miss (below AC)" and "miss (natural 1)" are `MasteryAttackRolled` with `Hit()` false; "miss (target not at the location)" is `MasteryAttackForcedMiss`; "no roll" is `MasteryAttackUnrolledMiss`.

| Label | Mastery, conclusion, other facts | Expected | Path |
|---|---|---|---|
| Cleave, hit | Cleave, hit | `CleaveOffer`, no modifier on damage | happy |
| Cleave, Critical Hit | Cleave, critical | `CleaveOffer` | unhappy (boundary) |
| Cleave, miss | Cleave, miss (below AC) | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Cleave, thrown hit | Cleave, hit, delivery ranged | `MasteryNone(notMeleeAttack)` | unhappy (rule interaction) |
| Cleave, used | Cleave, hit, Cleave already recorded | `MasteryNone(alreadyUsedThisTurn)` | unhappy (state error) |
| Graze, miss below AC | Graze, miss (below AC) | `GrazeOffer`, 3 of the weapon's type | unhappy (boundary) |
| Graze, natural 1 | Graze, miss (natural 1) | `GrazeOffer`, 3 | unhappy (boundary) |
| Graze, modifier 0 | Graze, miss, modifier 0 | `GrazeOffer`, 0 | unhappy (boundary) |
| Graze, modifier -1 | Graze, miss, modifier -1 | `GrazeOffer`, 0 | unhappy (boundary) |
| Graze, hit | Graze, hit | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Graze, vacated location | Graze, miss (target not at the location) | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Graze, underwater automatic miss | Graze, miss (no roll) | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Push, Large | Push, hit, target Large | `PushOffer`, 10 | unhappy (boundary) |
| Push, Huge | Push, hit, target Huge | `MasteryNone(targetTooLarge)` | unhappy (boundary) |
| Push, Tiny | Push, hit, target Tiny | `PushOffer`, 10 | unhappy (boundary) |
| Push, miss | Push, miss, Huge | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Sap, hit, no damage | Sap, hit, damage dealt 0 | `SapOffer` | unhappy (boundary) |
| Slow, damage | Slow, hit, damage dealt 1 | `SlowOffer`, 10 feet | unhappy (boundary) |
| Slow, no damage | Slow, hit, damage dealt 0 | `MasteryNone(noDamageDealt)` | unhappy (boundary) |
| Topple, Gargantuan | Topple, hit, target Gargantuan | `ToppleOffer`, DC 13 | unhappy (boundary) |
| Topple, modifier -1 | Topple, hit, modifier -1, bonus +2 | `ToppleOffer`, DC 9 | unhappy (boundary) |
| Topple, high level | Topple, hit, modifier +5, bonus +6 | `ToppleOffer`, DC 19 | unhappy (boundary) |
| Topple, miss | Topple, miss | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Vex, damage | Vex, hit, damage 4 | `VexOffer` | happy |
| Vex, no damage | Vex, hit, damage 0 | `MasteryNone(noDamageDealt)` | unhappy (boundary) |
| Vex, thrown | Vex (Handaxe), hit, delivery ranged | `VexOffer` | unhappy (rule interaction) |
| Nick | Nick, hit | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Not chosen | Vex, hit, damage 4, mastery not usable | `MasteryNone(featureNotAvailable)` | unhappy (state error) |
| No feature | Vex, hit, `NoMasteryFeature` | `MasteryNone(featureNotAvailable)` | unhappy (empty case) |
| Tactical Master | Longsword (Sap) hit, `ReplacedMastery(Slow)`, damage 5 | `SlowOffer` | unhappy (rule interaction) |
| Tactical Master, not chosen | same, weapon not chosen | `MasteryNone(featureNotAvailable)` | unhappy (state error) |
| Unusable wins | Vex, miss, not chosen | `MasteryNone(featureNotAvailable)` | unhappy (rule interaction) |
| Graze, natural 20 at a vacated location | Graze, `MasteryAttackForcedMiss` carrying a critical roll | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Cleave, miss then ranged | Cleave, miss, delivery ranged | `MasteryNone(attackDidNotQualify)` (wins over `notMeleeAttack`) | unhappy (rule interaction) |
| Cleave, ranged and used | Cleave, hit, delivery ranged, Cleave already recorded | `MasteryNone(notMeleeAttack)` (wins over `alreadyUsedThisTurn`) | unhappy (rule interaction) |
| Push, Huge and 0 damage | Push, hit, target Huge, damage dealt 0 | `MasteryNone(targetTooLarge)` (Push does not read damage) | unhappy (rule interaction) |
| Slow, miss and 0 damage | Slow, miss, damage dealt 0 | `MasteryNone(attackDidNotQualify)` (wins over `noDamageDealt`) | unhappy (rule interaction) |
| Vex, no roll | Vex, no roll, damage dealt 0 | `MasteryNone(attackDidNotQualify)` | unhappy (rule interaction) |
| Vex, Blowgun fixed damage | Vex (Blowgun, 1 fixed Piercing), hit, damage dealt 1 | `VexOffer` | unhappy (boundary) |

### Withdrawn ids

| Id | Withdrawn |
|---|---|
| FR-113 | Withdrawn: moved to weapon-properties-and-mastery.md (FR-122) |

## 5. Model

Domain types are immutable, validated in their constructors, pure and synchronous (`doc/rules/architecture.md` §3), all in package `character` and using only `character`, `content`, `d20`, kernel (`dice`, `ability`, `modifier`, `srderr`, `telemetry`) and port types (FR-109). Value objects have unexported fields, validating constructors (`NewX`, with a panicking `MustX` beside each) and value receivers; types holding a slice are not compared with `==` and provide `Equal`.

**Pure rules (files `weapon_proficiency.go`, `weapon_mastery_choice.go`, `weapon_attack_plan.go`, `weapon_light_attack.go`, `weapon_mastery_offer.go`)**

- *Proficiency and choice (FR-007 to FR-029, FR-120):* `WeaponProficiencyRule` (closed: `CategoryProficiency`, `CategoryWithAnyPropertyProficiency(category, properties)` non-empty per FR-011, `AnyStatBlockWeaponProficiency`); `WeaponProficiencies` (`StatusFor`; `WeaponProficienciesFromClassData`); `WeaponProficiencyStatus`; `MasteryEligibility` (closed: `SimpleOrMartialMeleeEligibility`, `SimpleOrMartialEligibility`, `ProficientWeaponEligibility{Proficiencies}`); `MasteryRetrainRule` (`ReplaceOneKind`, `ReplaceAnyKinds`); `MasteryChangeMoment` (FR-024); `MasterySwap{Remove, Add}`; `WeaponMasteryChoice` (ordered list of `ItemID`, `Kinds()` a copy, `Equal`) built by the pure functions `CreateWeaponMasteryChoice`, `ExtendWeaponMasteryChoice` and `ChangeWeaponMasteryChoice` (with `resolved` a `map[ItemID]Weapon` that is only read), each returning a closed `WeaponMasteryChoiceResult` (`WeaponMasteryChoiceAccepted(choice)`, `WeaponMasteryChoiceRejected(problems)`) and an `error`; `MasteryProblem{Kind, WeaponID}` (the weapon id absent where the kind has none) with `MasteryProblemKind` (`wrongCount`, `duplicateWeapon`, `notMelee`, `notProficient`, `weaponNotResolved`, `dropsExistingMastery`, `alreadyChosen`, `notCurrentlyChosen`, `changesMoreThanOne`, `wrongMoment`); `WeaponMasteryAccess` (closed: `NoMasteryFeature`, `MasteryFromChoice{Choice}`; `Permits(ItemID)`); `WeaponMasteryChoiceState{Choice}` and `WeaponMasteryChoiceKey()` (FR-123; the `FeatureStateKey` named `weapon_mastery_choice`, a function so no package-level variable exists).
- *Invariants:* `allowedCount` at least 1 (an `*InvalidError`, FR-028) and the non-empty property set (an `*InvalidError`, FR-011) by constructors; count, distinctness, eligibility and missing-content rules by the three choice functions (results). *No events* (`shared-types-and-ownership.md` FR-019: no transition here is an aggregate fact). Events carrying a `CharacterID` belong to the class use cases.
- *Property rules (FR-030 to FR-056, FR-114 to FR-117, FR-121, FR-124):* `WeaponAttackMode` (closed: `MeleeAttackMode`, `ThrownAttackMode`, `RangedAttackMode`); `WeaponAttackKind`; `AbilityPreference` (closed: `DefaultAbility`, `ChosenAbility{Ability}`); `Grip` (`oneHanded`, `twoHanded`), `HandState` (`freeHandAvailable`, `noFreeHand`), `LoadingState` (`ready`, `alreadyFiredThisActivation`), `TwoHandedExemption` (`exemptionApplies`, `noExemption`); `AmmunitionSupply` (`NewAmmunitionSupply(pieces, ammunitionType)`, `Expend`, `RecoverAfterFight`); `WeaponWielding{Grip, Exemption, Hands, Loading}` with `WithAmmunition(AmmunitionSupply)` and `Ammunition() (AmmunitionSupply, bool)`; `ReachInFeet`; `PlanWeaponAttack(WeaponAttackInput{Weapon, Mode, Strength, Dexterity, Preference, Wielding, AttackerReach, Proficiencies})` returning closed `WeaponAttackPlanResult` (`WeaponAttackPlanned{Plan}`, `WeaponAttackPlanBlocked{Reason}`) and `OpportunityAttackReach`; `WeaponAttackPlan{Kind, Ability, AbilityModifier, Proficiency, Damage, DamageModifier, ReachOrRange, Circumstances, AmmunitionAfter, LoadingAfter}` where `ReachOrRange` is the closed `PlannedReach{Reach}` / `PlannedRange{Range RangePair}` and `AmmunitionAfter` is read through `AmmunitionAfter() (AmmunitionSupply, bool)` (absent when the weapon has no Ammunition); `WeaponAttackBlockReason` (enum in FR-055 order).
- *Light extra attack (FR-057 to FR-070):* `WeaponInstanceKey` (a comparable value over a non-blank string, `NewWeaponInstanceKey`); `WeaponTurnState` (zero value is the initial state; `RecordAttackAction()`, `RecordAttack(key, hasLight)`, `RecordLightExtraAttack()`, `RecordCleaveAttack()` each return a new value; the "made or not" fields are enums, not booleans); `PlanLightExtraAttack(LightExtraAttackInput{State, Weapon, InstanceKey, Access, InPlay, AbilityModifier, DamageSource})` returning closed `LightExtraAttackResult` (`LightExtraAttackAllowed{Cost, DamageModifier, AddedModifier}`, `LightExtraAttackBlocked{Reason}`); `ExtraAttackCost`, `AbilityModifierToDamage`, `LightExtraAttackBlockReason` (FR-067 order), closed `ExtraAttackDamageSource` (`NoExtraAttackDamageEffect`, `ExtraAttackDamageEffectActive`).
- *Mastery effects (FR-071 to FR-096, FR-118, FR-119):* `OfferMastery(MasteryAttackFacts)` returning closed `MasteryOffer`; `MasteryAttackFacts{WeaponID, Weapon, Access, InPlay, Outcome, Mode, AbilityModifier, ProficiencyBonus, TargetSize, DamageDealt, TurnState}`; `MasteryAttackOutcome` (closed, FR-118); offers `CleaveOffer{WeaponID, Kind, Damage, DamageModifier}` with `ValidateTarget` returning closed `CleaveTargetResult` (`CleaveTargetAccepted`, `CleaveTargetRejected{Failure}`; `CleaveTargetFailure`), `GrazeOffer{Amount, DamageType}`, `PushOffer{MaximumFeet}` with `ChooseDistance` returning closed `PushDistanceResult` (`PushDistanceAccepted{Movement}`, `PushDistanceRejected{Reason}`; `PushDistanceRejection`), `SapOffer{Circumstance, Expiries}`, `SlowOffer{Reduction, Expiries}`, `VexOffer{Circumstance, Expiries}`, `ToppleOffer{Ability, DC}`, `MasteryNone{Reason}`; `MasteryNoneReason`, `MasteryEffectExpiry`, `SpeedReduction`, `PushMovement`, `CleaveTargetPlacement`, `MasteryInPlay` (closed: `WeaponsOwnMastery`, `ReplacedMastery{Property}`).
- *Failures:* a blocked plan, a `MasteryNone`, a rejected choice, Cleave target or Push distance are results. A bad constructor argument or a precondition of a pure function (a negative Speed, a negative `damageDealt`, `Expend()` at 0) is an `error` (`*InvalidError`, or `ErrSupplyEmpty`), never a panic; only the `MustX` helpers and unreachable branches panic, with an error value (`doc/rules/architecture.md` §6). The only collaborator failure is the dice port's `*srderr.PortError` (FR-101).

**Use case (file `topple_save_usecase.go`)**

- `ToppleSaveResolver` with the method `ResolveToppleSave(ctx, ResolveToppleSaveCommand{Offer, Attempt})` (`Offer` a `ToppleOffer`, `Attempt` the `d20.SaveAttempt`: `RollAttempt`, `ChooseToFail`, `AutomaticFailure`) returning the closed `ResolveToppleSaveResult`: `ToppleSaveTargetProne{Detail}`, `ToppleSaveTargetResisted{Detail}`; `ToppleSaveDetail{DC, Basis}` with closed `ToppleSaveBasis` (`ToppleSaveRolled{Check}`, `ToppleSaveForced{Failure}`). Variants carry the operation prefix; no generic name is exported.
- Telemetry constants are added to the `character` signal constants (`character-aggregate.md` section 7); no signal library of its own (FR-111).

## 6. Interface

- **Pure rules are called directly by other specs' use cases.** Class use cases (for example `ChooseBarbarianWeaponMasteries`, `class-barbarian.md` FR-103) resolve kinds through the content source and pass the `Weapon` values to `CreateWeaponMasteryChoice`. No pure rule calls a use case (`doc/rules/architecture.md` §2, §4).
- **Handoff to `combat`.** The host may import this spec's published language and `combat`'s. It calls `PlanWeaponAttack`, maps the plan onto the attack-resolution inputs of `attacks.md` (attack form from `WeaponAttackKind`, reach and range from `PlannedReach` and `PlannedRange`, damage from `WeaponDamage` and `AbilityModifierToDamage`), resolves the attack, builds a `MasteryAttackOutcome` (adding `MasteryAttackForcedMiss` or `MasteryAttackUnrolledMiss` when its own rules decide so), calls `OfferMastery`, and for a `ToppleOffer` calls `ToppleSaveResolver.ResolveToppleSave`. That mapping is `combat`'s to specify (`attacks.md`, `combat-action-resolution.md` Q-36). A `combat` use case MUST NOT call `ResolveToppleSave` (`doc/rules/architecture.md` §1; `shared-types-and-ownership.md` FR-044); it composes the pure `d20` functions itself, taking the DC from the `ToppleOffer`. The planners and offer function are pure, which `combat` domain code may call.
- **`NewToppleSaveResolver(roller Roller, opts ...Option) (ToppleSaveResolver, error)`**: one method, `ResolveToppleSave(ctx context.Context, cmd ResolveToppleSaveCommand) (ResolveToppleSaveResult, error)`. `ctx` is first because the call emits a span, is passed to the telemetry only and is never stored (go.md §4); the dice call itself is synchronous and in memory. It composes the pure `d20` functions (FR-122), calls no use case and contains no rules logic. **Ports used:** the consumer-side one-method `Roller` and the `telemetry` ports only; no new port, test double or contract suite. Errors: a dice-port failure is a `*srderr.PortError` (FR-101); an invalid command (nil offer or attempt, a foreign `Attempt` variant) is an `*InvalidError` before any signal; a cancelled `ctx` is never turned into a result. A missing weapon is `weaponNotResolved` (FR-020). The resolver is safe for concurrent use when its `Roller` is.
- **Exports.** Package `character` exports every type named in section 5 (FR-110), including each closed family's variants, `WeaponMasteryChoiceState` and `WeaponMasteryChoiceKey`; there is no facade or re-export package. Types from `content` (`Weapon`, `WeaponCategory`, `WeaponType`, `WeaponDamage`, `WeaponPropertyUse`, `RangePair`, `AmmunitionType`, `WeaponProficiency`, `WeaponProperty`, `MasteryProperty`, `DamageType`, `CreatureSize`, `ConditionKind`, `ItemID`), `d20` (`AttackRollResult`, `DifficultyClass`, `CheckResult`, `ForcedFailureResult`, `SaveAttempt`) and `derived-stats.md` (`ProficiencyBonus`) are not re-exported. The package `doc.go` lists inputs and outputs.

## 7. Observability

Answers to the seven evaluation questions (`doc/rules/observability.md` §2):

1. **What can go wrong?** The roller fails, panics or returns the wrong shape during the Topple save (a `*srderr.PortError`); the host feeds impossible facts (a stale `LoadingState`, a wrong `HandState`, a `DamageDealt` always 0).
2. **How would an operator know?** A roller failure is a span error status plus `srd5e.character.operations` with `srd5e.outcome` `failure` and `error.type` `dice_roller_error` (from `srderr.ErrorType`); a failed save is a normal outcome (`srd5e.outcome` = `failure`, `srd5e.character.result` = `target_prone`). Domain rules are silent and return reasons as data.
3. **What to diagnose it?** The result carries the DC and the `d20.CheckResult` or `d20.ForcedFailureResult` (FR-100); telemetry carries the operation and the closed result value.
4. **What to count or time?** The shared `character` counter and histogram already count and time every call by operation and result (`character-aggregate.md` section 7); offers and plans are not counted (silent domain).
5. **Is existing instrumentation still correct?** Yes. `ResolveToppleSave` calls no use case (FR-102, FR-122), so `srd5e.d20_test.*` does not fire and nothing is counted twice. The shared counter gains one operation value and two result values; the histogram only the operation value (`character-aggregate.md` FR-103).
6. **What to monitor?** Any `error.type` increment for `resolve_topple_save` and a duration regression; no threshold on the `target_resisted` share (it depends on the host's DC mix).
7. **Is anything noisy?** No: no log at any level (FR-111).

**Silent by design:** proficiency, the choice, the planners, the offers and the turn state are pure value computation with no I/O (`doc/rules/observability.md` §3); the rejection reasons are the diagnostic surface.

**Signals owned here.** Only the span below; the counter and histogram are the shared ones of `character-aggregate.md` section 7, which this spec MUST NOT redefine. Closed `srd5e.character.result` values for this operation: `target_prone` (`ToppleSaveTargetProne`) and `target_resisted` (`ToppleSaveTargetResisted`). The former `d20` `basis` attribute is withdrawn (`resolve-d20-test-use-case.md` FR-052). `error.type` appears only on a thrown `*srderr.PortError (port dice)`, with no `srd5e.character.result` (`character-aggregate.md` FR-088). No event is produced.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.resolve_topple_save` | span | `srd5e.outcome` (`success`, `failure`), `srd5e.character.result` (`target_prone`, `target_resisted`) | Path of one Topple save; failure record when the dice port fails | Any error status |

New attribute values on shared signals: `srd5e.character.operation` = `resolve_topple_save` (counter and histogram) and `srd5e.character.result` = `target_prone`, `target_resisted` (counter and span). A `target_prone` share far above what the DCs imply may mean the host passes `ChooseToFail` by mistake, but no threshold is given. The constants live in the `character` signal constants; the span and values are added to `doc/observability/signals.md`.

## 8. Test plan

Doubles: `dicetest.FixedRoller` and `dicetest.CallbackRoller (returning an error)` plus the recording telemetry of `telemetry/telemetrytest`, for the use case only; pure-rule tests use none. Tests are external (`package character_test`) files such as `weapon_proficiency_test.go`, `weapon_mastery_choice_test.go`, `weapon_attack_plan_test.go`, `weapon_light_attack_test.go`, `weapon_mastery_offer_test.go` and `topple_save_usecase_test.go`, table-driven with `t.Run`. Fixtures build `Weapon` values from the golden Weapons table rows of `content-equipment.md` (`testdata/`), never from memory.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006 | unit | exhaustive `switch` over `WeaponProperty` (nine) and `MasteryProperty` (eight) without a default (checked by `gochecksumtype`/`exhaustive`); every bundled weapon yields one mastery; `Permits`, `NoMasteryFeature`, `ReplacedMastery`; choice unchanged after a replacement |
| FR-007 to FR-013, FR-120 | unit | table 4.1a; empty property set `*InvalidError`; `WeaponProficienciesFromClassData` for Rogue and Monk values |
| FR-014 to FR-029, FR-125 | unit, fuzz | table 4.3a (create, extend, swap and ordering rows); missing content alone; `wrongMoment` leaves `Current` unaltered; `Kinds()` copy, value `Equal`, `allowedCount` 0 returns `*InvalidError`, no input altered; `FuzzChangeWeaponMasteryChoice` never panics and a rejection never returns a choice |
| FR-123 | unit | key name matches `[a-z][a-z0-9_]*`, two keys equal; a state round-trips through `FeatureStates`; absent key gives `NoMasteryFeature`; equal choices give equal states |
| FR-030 to FR-033, FR-036 to FR-040 | unit | table 4.4a |
| FR-034, FR-035, FR-116 | unit | improvised, `modeNotSupportedByWeapon` and Finesse choice rows of 4.4b; `NewChosenAbility(Constitution)` `*InvalidError` |
| FR-041 to FR-043 | unit | table 4.4c (reach rows); thrown and ammunition ranges per weapon; Thrown with `noFreeHand` row of 4.4b |
| FR-044 to FR-049, FR-115, FR-117, FR-124 | unit | tables 4.4b, 4.4c (ammunition rows) |
| FR-050 to FR-054, FR-056, FR-114 | unit | table 4.4b (Versatile, Two-Handed, exemption, Blowgun, dice damage); proficiency and Heavy do not block |
| FR-055 | unit | 4.4b "Order 1" to "Order 8", one per adjacent pair of block reasons |
| FR-057 to FR-064, FR-067 | unit | table 4.5a first part and "Order 0" to "Order 3" |
| FR-065, FR-066, FR-068 to FR-070 | unit | table 4.5a damage rows (both `ExtraAttackDamageSource` variants); `WeaponTurnState` transitions, zero value, receiver unchanged |
| FR-071 to FR-073, FR-119 | unit | table 4.6a kind and precedence rows; `OfferMastery` has no side effect |
| FR-074 to FR-081 | unit | table 4.6a Cleave, Graze (forced miss over a natural 20) and Nick rows; table 4.4c placement rows |
| FR-082 to FR-084 | unit | table 4.6a Push rows; `ChooseDistance` -1, 0, 10, 11 as results, never an error or panic |
| FR-085 to FR-094 | unit | Sap offer; expiry data of all three timed effects; table 4.6a Slow, Topple, Vex rows (including Blowgun), table 4.4c (Slow rows) |
| FR-095, FR-096, FR-118 | unit | natural 1, natural 20, Critical Hit rows from real `d20.AttackRollResult` values with forced and unrolled misses |
| FR-097 to FR-100, FR-122 | application | real `d20` functions with the fixed roller: DC 13 with totals 12, 13, 14; one `dice.Roll` request of `1d20` or `2d20`; `ChooseToFail` and `AutomaticFailure` draw no die and give `ToppleSaveForced` |
| FR-101 to FR-103 | application | `dicetest.CallbackRoller (returning an error)` (one error record, no log, no `d20_test.*`; `errors.As` a `*srderr.PortError`, `errors.Is` the roller's error); a panicking roller matches `srderr.ErrPortPanic`; works with no telemetry option; wrong-shape roll matches `srderr.ErrContractViolation`; architecture scan finds no use-case import |
| FR-104 to FR-107, FR-121 | unit, application | same input twice; all-minimum and all-maximum rolls; immutability; constructor `*InvalidError`s (`ReachInFeet` 0, supply -1, negative Cleave distance, `DamageDealt` -1) and their `MustX` panics carry the same error |
| FR-108 to FR-110, FR-112 | architecture | `internal/archtest`: the pure files import no `context`, `time`, `math/rand`, `sync`, telemetry and no `combat` package; exported-identifier scan: every type of section 5 documented, no duplicate or generic name, no consumer name |
| FR-110 | scenario | a test in `scenarios/` importing only public packages builds each command and plan and switches exhaustively over each closed result |
| FR-111 | application | recording telemetry: span name, shared counter and histogram values, attribute keys and closed result values; `srd5e.outcome` `failure` for `target_prone`, `success` for `target_resisted`; span status not error; one error record with `error.type` = `srderr.ErrorType(err)` on a roller failure; no log; catalogue matches the constants |
| scenario | scenario | `scenarios/` (`package scenarios_test`), public API only: a level 4 Fighter with Dagger, Shortsword, Greatsword and Maul chosen plans a Shortsword attack, takes the Nick extra attack as part of the Attack action with a negative modifier, gets a Greatsword Graze offer on a miss, and a Maul Topple offer resolved with a fixed roller (attack roll from a `d20` fixture, no `combat` type) |
| errors-and-results.md FR-031, FR-034, FR-051, FR-054 | application | the dice failure is wrapped once and returned (FR-101); the recorded `error.type` is `srderr.ErrorType(err)`; a panicking telemetry double propagates; a cancelled `ctx` is not a result |

- Mutation testing: changed lines of the weapon and mastery files of package `character`, score at least 80; the use case SHOULD be (`doc/rules/testing.md` §8). `gomu` runs only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors, never as a bare host process. Deliberately not covered: equality of bundled weapon data to the Weapons table (`content-equipment.md` FR-017); saving-throw total arithmetic (`d20`); the mapping onto `combat` types (`combat` specs).

## 9. Open questions and decisions

No open question. Decisions, one line each:

- Context and ownership: `shared-types-and-ownership.md` FR-016, FR-013, FR-014; `MasteryProperty` name: its FR-005; no parallel proficiency or outcome types: its FR-004, FR-027; no use case wraps `d20.Resolver`: its FR-044; Weapon Mastery slot shared by the five classes: FR-123.
- Class-owned: counts come from the class progression (`class-barbarian.md` FR-041, `class-fighter.md` FR-023); eligibility has one variant per rule, and every weapon has a mastery property (FR-003); Tactical Master's set is restricted by `class-fighter.md` (FR-049); class proficiency data is `content-classes.md` FR-021 (FR-120).
- Package decisions (not SRD text): Bonus Action and attack counts are `combat`'s (FR-063, section 9); Nick uses the extra-attack weapon's mastery (FR-064); "different Light weapon" is any recorded other weapon (FR-061); no Graze on a forced or unrolled miss (FR-079); damage dealt is after Resistance, Vulnerability and Immunity and before Temporary Hit Points (p. 190); `combat` stores Sap, Slow and Vex between turns; one `WeaponTurnState` per turn, reset by the host (FR-068); `Grip` and `HandState` are host facts (FR-043, FR-053); `LoadingState` per activation (FR-048); the host reports `TwoHandedExemption` (FR-050, FR-051); Prone Immunity is the host's (section 1). Planners and the offer service are domain services, not use cases; telemetry uses the shared `character` signals (FR-111).

### 9.1 Edits required in other specs

None is done here. The five class specs hold the choice only as `WeaponMasteryChoiceState` under `WeaponMasteryChoiceKey()` (FR-123); `class-fighter.md`, `class-ranger.md`, `class-barbarian.md` withdraw the "Supersedes" requirements of section 1 (the Fighter's `FighterOptionContentMissing` maps to `weaponNotResolved`, `noLongRestFinished` to `wrongMoment`); `attacks.md` reduces its thrown-ability and FR-007 Finesse sentence to a reference to FR-031, FR-032, and `attacks.md` and `combat-action-resolution.md` map `WeaponAttackPlan`, `MasteryOffer` and `MasteryAttackOutcome` (section 6) and name `WeaponWielding` for hand, grip, loading and ammunition facts; `character-aggregate.md` and `doc/observability/signals.md` add the operation value `resolve_topple_save`, the span and the result values `target_prone`, `target_resisted`.

## 10. Changelog impact

Added: `WeaponMasteryChoice`, `MasteryEligibility`, `WeaponMasteryAccess`, `WeaponProficiencies`, `PlanWeaponAttack`, `PlanLightExtraAttack`, `OfferMastery`, `WeaponTurnState`, `ResolveToppleSave` and their value objects and results, the span `srd5e.character.resolve_topple_save`, and one operation value and two result values on the shared `srd5e.character.*` signals. Not **BREAKING**: new public API and new attribute values only (the class specs that define their own mastery types are unreleased drafts).
