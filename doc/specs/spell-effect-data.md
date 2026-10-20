# Spell effect data: the structured effects of a spell

| | |
|---|---|
| Status | partially implemented |
| Context | `content` |
| Roadmap phase | Types 2; bulk data 5 |
| Depends on | `dice-and-rolls.md`, `ability-scores.md`, `content-schema-and-decoder.md` |

## 1. Purpose and scope

This is the content half of the spell-effects design, split out of
`spell-effects.md` under shared-types-and-ownership.md FR-059 (that file stays the
`combat` spec: resolver, use case, plan and telemetry). It defines what a spell
may carry beside its text so that the package can say what the spell does:

- **The extension point.** Spell effects are **data** in `content`: a closed set of effect variants
  (each a closed variant set: an interface with an unexported marker method, one struct per variant, switched exhaustively), optional per spell, decodable from JSON and
  validated by the decoder. The interpreter lives in `combat` (`spell-effects.md`).
  No strategy interface per spell is introduced in this phase (section 5.1
  compares the options; Q1 records the trigger for revisiting).
- **What is structured.** Only what SRD 5.2.1 names as the substance of a
  spell's effects ("targets, saving throws, attack rolls, or all three", p. 106)
  and the rules those hooks use: Targets and areas of effect, spell attack rolls,
  saving throws and half damage, damage dice and Damage Types, Hit Point
  restoration, Temporary Hit Points, Conditions, the static facts of a lingering
  area (FR-048), and scaling with the slot level
  (Using a Higher-Level Spell Slot) and with the character level (Cantrip
  Upgrade).
- **What stays text.** Everything else the description says (movement, summoned
  creatures, terrain, light, illusions, information, objects, choices among
  modes, effects on later turns, riders on later rolls). A spell whose text has
  such parts is marked `partial`, which is the SRD's own rule ("any outcomes
  beyond those effects are under the GM's purview", p. 106).
- **Derived facts.** The model answers the spell facts that class specs need
  (does it force a saving throw, make a spell attack roll, deal damage, of which
  types, can a higher slot add targets) as tri-state facts that never guess for a
  text-only spell (FR-022 to FR-024).
- **Decoder shape and bundling.** The optional `effects` field of a spell, its
  problem kinds, and the offline checks that tie a model to the verbatim text
  (FR-030 to FR-038).
- **Population.** The model is optional per spell. A spell with no structured
  data is valid and resolves to "text only". This spec pins a pilot set of 16
  spells (table P) and the verification rules for annotating the rest. It does
  not annotate all 339.

**Out of scope:**

- Resolving anything: rolling, saving throws, attack rolls, validity of a listed
  target, the plan of damage and conditions, telemetry. All of that is
  `spell-effects.md`. This spec has no use case, no port and no signal.
- Choosing targets, deciding who is inside an area of effect, line of sight,
  visibility and range checks. The host (the GM) supplies the creatures; the
  shapes are stored as data so the host can compute membership
  (`spell-effects.md` FR-015).
- Running state of anything that lasts or recurs: durations, Concentration,
  repeated saves, riders (Hunter's Mark, Hex), delayed damage, "Combining Spell
  Effects" (p. 106) and the state of a lingering area. These are `combat`:
  active-effects.md and persistent-spell-zones.md. This spec holds only the
  static zone facts (FR-048) and duration data to hand on (FR-046).
- Summoned creatures, Hit Point maximum changes (Aid, Harm), removing conditions
  (Heal), terrain, light, objects, illusions, divinations, modes (Eyebite,
  Command): all text; table E gives the reasoning and any rule owner.
- Spell Scrolls and other magic items; the spellcasting traits of monsters; rule
  options (the SRD names none for spell effects, `doc/rules/architecture.md` §3).
- Other books. The SRD gives spell effects as prose; the structure here is
  package design within SRD vocabulary. Nothing core was missing from the PDF.

**Ownership and supersession (shared-types-and-ownership.md FR-048).** This spec
owns the effect data model, scaling shapes, area shapes and the derived facts.
The older, broader specs keep their entry and a reference. This spec
**supersedes**:

- content-spells.md Q5 and FR-115 where they say the package has no structured
  scaling: structure and scaling live in `Spell.effects` (FR-001, FR-016 to
  FR-021). FR-115's ban on a `SpellScaling` type in the text model stands, and
  content-spells.md FR-087 (equality over all fields) is extended by FR-003 here,
  not replaced.
- content-schema-and-decoder.md Q7 (spell effects as text only): answered by
  FR-030 here as one row in its revision table.
- class-sorcerer.md FR-123 and the spell-fact fields of its `MetamagicSpellFacts`
  (saving throw, spell attack roll, damage types, an extra target by slot
  level), class-wizard.md `WizardSpellFacts.DealsDamage`, and class-warlock.md
  `CantripFacts` (its FR-156, table 4.6.8): all read `SpellEffectFacts`
  (FR-022 to FR-024) when the spell has a model, and keep refusing with an
  unknown fact when it has none (FR-024).
- Nothing in spellcasting.md is superseded: its out-of-scope list (targets,
  saving throws, areas of effect) is the entry that `spell-effects.md` and this
  spec fill. magical-effects.md reads `AreaShape` from here and keeps the origin
  override and membership; this spec does not depend on it.

## 2. Source

SRD 5.2.1 (`doc/research/sources.md`). Printed page numbers equal PDF page
numbers (checked on pp. 104-107, 129-132, 139-141, 158-160 while drafting).

- "Spells", "Effects" (what the effects section says, and that effects
  typically deal with targets, saving throws, attack rolls, or all three),
  "Targets" ("A Clear Path to the Target", "Targeting Yourself", "Areas of
  Effect"), "Invalid Targets", "Saving Throws" (spell save DC), "Attack Rolls",
  "Combining Spell Effects", p. 106.
- "Spells", "Spell Level", "Spell Slots", "Casting without Slots" and "Using a
  Higher-Level Spell Slot", pp. 104-105; "Range" (Self, Touch, Distance), p. 105.
- "Character Creation", "Multiclassing", "Spellcasting", "Cantrips" (a cantrip's
  increase is based on total character level), p. 25.
- "Playing the Game", "Damage Rolls", "Critical Hits", "Saving Throws and
  Damage" ("Damage against Multiple Targets", "Half Damage"), p. 16; "Damage
  Types", pp. 16-17; "Healing", p. 17; "Temporary Hit Points", p. 18; "Melee
  Attacks" and "Ranged Attacks", p. 15; "Saving Throws" and "Attack Rolls", p. 7.
- "Rules Glossary": "Area of Effect", p. 177; "Cone" and "Cube", p. 179;
  "Cylinder" and "Damage Types" table, p. 180; "Emanation", p. 181; "Line",
  p. 184; "Sphere" and "Spell Attack", p. 188; "Cantrip", p. 178; "Condition"
  (the fifteen conditions) and "Concentration", p. 179; "Magical Effect", p. 185.
- Spell entries read for the pilot set and the pattern tables, with the page of
  each heading: Acid Splash, p. 107; Aid, p. 107; Acid Arrow, p. 107; Bane,
  p. 112; Bestow Curse, pp. 112-113; Blade Barrier, p. 113; Blink, pp. 113-114;
  Burning Hands, p. 114; Chain Lightning, p. 114; Chromatic Orb, p. 115;
  Cure Wounds, p. 121; Eldritch Blast, p. 127; Eyebite, p. 129; Faerie Fire,
  p. 129; False Life, p. 129; Fear, p. 130; Find Steed, p. 130; Finger of Death,
  p. 131; Fireball, p. 131; Fire Bolt, p. 132; Flame Strike, p. 132; Flaming
  Sphere, p. 132; Harm, p. 139; Heal, p. 139; Healing Word, p. 139; Hex, p. 140;
  Hold Person, p. 141; Hunter's Mark, p. 141; Ice Knife, p. 141; Ice Storm,
  p. 141; Magic Missile, p. 146; Ray of Sickness, p. 158; Command, p. 116; True
  Strike, p. 171; Sacred Flame, p. 159; Scorching Ray, p. 159; Mass Heal,
  p. 147; Create or Destroy Water, p. 121.

Package rules: `doc/rules/architecture.md` §2 (vocabulary types are static
facts; behaviour lives where the rule is applied), §3 (effects expressed as data;
strategy interfaces only for a real need), §5, §6; `doc/rules/testing.md` §4, §5,
§8. Binding ownership decisions: shared-types-and-ownership.md FR-005, FR-013,
FR-034, FR-035, FR-046, FR-048, FR-059.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Spell, Effects (of a spell) | `Spell.effects`, a `SpellEffectModel` | Optional structure beside the text (FR-001). Owner of `Spell`: content-spells.md, which imports the model from here. |
| (a spell's structured effects) | `ModelledEffects` | Package term. Targeting, ordered parts, coverage. |
| (a spell with no structure) | `TextOnlyEffects` | Package term. The description text is the only source. |
| (how much of the text is modelled) | `EffectCoverage` (`CoverageComplete`, `CoveragePartial`) | Package term. FR-004. |
| Targets | `SpellTargeting` (`TargetsSelf`, `TargetsChosen`, `TargetsInArea`), `TargetSubject`, `TargetCount` | "creatures, objects, or something else" (p. 106). |
| Targeting Yourself | `SelfTargetPolicy` (`MayIncludeCaster`, `ExcludesCaster`) on `TargetsChosen` | p. 106 (FR-040). |
| Area of Effect | `AreaShape` | Six shapes: Cone, Cube, Cylinder, Emanation, Line, Sphere (p. 177). Owned here (FR-013, FR-014) and reused by magical-effects.md, which owns the origin override (`CreatorIncludesOrigin`) and membership (`AreaMembership`), not the shapes. |
| Attack Rolls (of a spell) | `AttackRollResolution`, `SpellAttackKind` | Melee or ranged spell attack (FR-042). Resolved by `spell-effects.md`. |
| Saving Throws (of a spell) | `SavingThrowResolution` | The ability, what a failure does, what a success does (FR-042). |
| (no roll) | `AutomaticResolution` | Magic Missile, Cure Wounds. |
| (one effect of a part) | `SpellOutcome` (`SpellDealsDamage`, `SpellRestoresHitPoints`, `SpellGrantsTemporaryHitPoints`, `SpellAppliesCondition`) | Package term (FR-047). |
| Damage Types, Damage Rolls | `DamageType` (content vocabulary), `SpellAmount`, `DamageTypeSelection` | `DamageType` is owned by content-schema-and-decoder.md (FR-107; shared-types-and-ownership.md FR-046). |
| Half Damage | `SuccessEffect.HalfDamageOnSuccess` | p. 16. Rounding is hit-points-and-damage.md's. |
| Using a Higher-Level Spell Slot | `AmountScaling.PerSlotLevelAmount`, `CountScaling.PerSlotLevelCount` | pp. 104-105. |
| Cantrip Upgrade | `AmountScaling.ByCharacterLevelAmount`, `CountScaling.ByCharacterLevelCount` | p. 107. Selected by total character level. |
| Temporary Hit Points, Healing | `SpellGrantsTemporaryHitPoints`, `SpellRestoresHitPoints` | pp. 17-18. |
| Condition | `ConditionKind` (content vocabulary), `SpellAppliesCondition`, `ConditionDuration` | p. 179. `ConditionKind` is owned by content-schema-and-decoder.md (FR-110). |
| Creature Type | `CreatureType` (content vocabulary), `CreatureTypeRestriction` | Hold Person targets "a Humanoid". `CreatureType` is owned by content-schema-and-decoder.md (FR-108). |
| (derived spell facts) | `SpellEffectFacts`, `EffectFact` (`FactKnown`, `FactUnknown`) | Package terms. FR-022 to FR-024. |
| `ConditionDuration.UntilNextTurnBoundary` owner | `TurnOwner` (`OwnerCaster`, `OwnerTarget`) | Package term. A different type from the `TurnHolder` (`player`, `GameMaster`) of magical-effects.md, which renamed its type to avoid this name; the two never share a name in the module's packages (`spell-effects.md` FR-089). |

Terms follow SRD 5.2.1 wording (`doc/rules/architecture.md` §3); the SRD says
"Using a Higher-Level Spell Slot", never "upcast".

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. "The model" means
`SpellEffectModel`; "a part" means a `SpellEffectPart`. Ids FR-001 to FR-039 were
moved from `spell-effects.md` (its withdrawn-ids table gives the old ids); FR-040
onwards are new. The case tables follow the requirements; their labels are not ids.

### 4.1 Representation and the extension point

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `Spell` MUST expose its structured effects as `Spell.effects`, a closed `SpellEffectModel` (an interface with an unexported marker method) with exactly two variants: `TextOnlyEffects` (an empty struct value, `TextOnlyEffects{}`, no data) and `ModelledEffects`. A spell with no structured data MUST carry `TextOnlyEffects`, and that MUST be a valid `Spell`. | SRD 5.2.1, "Effects", p. 106; content-spells.md FR-007 and FR-018 (a host's own spell is a valid `Spell`) | happy |
| FR-002 | The model MUST be plain data held in `content` and decodable from JSON: value objects with validation and static queries only. It MUST NOT hold a function, a callback, a class name or a spell-specific type: no field, method result or constructor parameter of a model type may have a function type (`func(...)`) or the type `any` (`interface{}`), apart from the unexported marker interfaces, and no model type may be named after a spell. This is checked by an architecture test in `internal/archtest` that parses the model files with `go/parser` and `go/ast` (stdlib only; architecture-guardrails.md FR-063) and scans the field, result and parameter types for a function type, `any`, `interface{}` and spell-named identifiers. This phase MUST NOT add a per-spell behaviour or strategy interface; the standard rules are the interpreter in `combat` (section 5.1). | `doc/rules/architecture.md` §2 (vocabulary types are static facts), §3 ("effects expressed as data"; add a strategy "only when a real need appears") | happy |
| FR-003 | The text fields of `Spell` (`description`, `HigherLevel`, `CantripUpgrade`) MUST stay the authoritative, verbatim source. Adding `effects` MUST NOT change, reorder or drop any text field, and `Spell` equality (`Spell.Equal`; the type holds slices so `==` is not available) MUST include `effects` (content-spells.md FR-087 reads "all fields"). | content-spells.md FR-064, FR-087; `AGENTS.md`, "Licensing and source material" | happy |
| FR-004 | `ModelledEffects` MUST carry an `EffectCoverage`. `CoverageComplete` states that every outcome the effect text describes is modelled: attack rolls, saving throws, damage, Hit Point restoration, Temporary Hit Points, Conditions, and the scaling the spell prints. Any other described outcome (movement, summoned creatures, terrain, light, objects, Hit Point maximum changes, removing a condition, effects on later turns, riders on later rolls) MUST make the coverage `partial`. Limits on who may be chosen beyond FR-011, FR-015 and `spell-effects.md` FR-018 (for example "that you can see") are the host's and do not affect coverage. | SRD 5.2.1, "Effects", p. 106 ("any outcomes beyond those effects are under the GM's purview") | happy |
| FR-005 | An outcome of a `partial` spell that no variant models exactly MUST stay in the text. The model MUST NOT encode it with a nearest variant (for example Aid's Hit Point maximum increase as Temporary Hit Points). | `AGENTS.md`, "Licensing and source material" (faithful to the source); SRD 5.2.1, "Effects", p. 106 | unhappy (rule interaction: model and text disagree) |
| FR-006 | `ModelledEffects` MUST have at least one part. A spell with nothing structurally modellable MUST be `TextOnlyEffects`, and an empty parts list MUST be rejected with an error. | `doc/rules/testing.md` §4 (empty and minimal); package design | unhappy (empty and minimal) |
| FR-007 | Every slice in the model (parts, outcomes, restriction sets, scaling tables) MUST be copied on construction and copied again on read, so later mutation of the caller's slice, or of a slice a getter returned, does not change the model (`errors-and-results.md` FR-021). | `doc/rules/architecture.md` §3, §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-008 | Every model type MUST have value equality: equal iff all fields are equal, with set-valued fields order-insensitive. A model type whose fields are all comparable is comparable with `==`; the others expose an `Equal` method (no hash codes exist in Go; a map key uses a comparable key type). | `doc/rules/architecture.md` §3 (value object) | happy |
| FR-009 | `SpellEffectModel` MUST offer a pure level check `CheckSpellLevel(spellLevel int) error` (taking a plain integer, so this spec imports no `Spell`; content-spells.md FR-122 has the `Spell` constructor call it) that MUST reject, with an error, a model that contradicts the spell's level: a per-slot scaling (`PerSlotLevelAmount`, `PerSlotLevelCount`) on a cantrip; a character-level table (`ByCharacterLevelAmount`, `ByCharacterLevelCount`) on a level 1+ spell; and a per-slot `AboveLevel` below the spell's level or above 9. | SRD 5.2.1, "Spell Level" and "Casting without Slots", p. 104 (a cantrip is cast without a spell slot); "Acid Splash", p. 107 (Cantrip Upgrade by level); content-spells.md FR-063 | unhappy (rule interaction) |

### 4.2 Targeting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `SpellTargeting` MUST be closed with exactly `TargetsSelf`, `TargetsChosen{Subject, Count, Restriction, SelfPolicy}` and `TargetsInArea{Shape, Subject, Restriction}`, so a `switch` is exhaustive. `SelfPolicy` is FR-040. | SRD 5.2.1, "Targets", p. 106; "Range" (Self), p. 105 | happy |
| FR-011 | `TargetSubject` (typed enum with `String()`) MUST enumerate exactly `SubjectCreature`, `SubjectObject` and `SubjectCreatureOrObject`. A target that is "something else" (a point, a location) is not modelled, and a spell whose effect depends on one is `partial` or `TextOnlyEffects`. | SRD 5.2.1, "Targets" ("creatures, objects, or something else"), p. 106 | happy |
| FR-012 | `TargetCount` MUST be closed with `ExactlyTargets{N}`, `UpToTargets{N}` and `TargetsByInstance`, with `N` at least 1; 0 and negative values MUST be rejected with an error. `TargetsByInstance` means the targets are those the part's instances are assigned to (`spell-effects.md` FR-031). Either counted variant MAY carry a `CountScaling` (FR-017). | SRD 5.2.1, "Targets" ("one or more creatures"), p. 106; "Magic Missile", p. 146 | unhappy (boundary) |
| FR-013 | `AreaShape` MUST be closed with exactly the six shapes of the Area of Effect glossary entry, each holding its dimensions in whole feet: `ConeArea{LengthFeet}`, `CubeArea{SideFeet}`, `CylinderArea{RadiusFeet, HeightFeet}`, `EmanationArea{DistanceFeet}`, `LineArea{LengthFeet, WidthFeet}` and `SphereArea{RadiusFeet}` (each built by a `New...Area` constructor returning `(T, error)`). Every dimension MUST be at least 1, else an error. These are the only definitions of the six shapes in the package. See table A. | SRD 5.2.1, "Area of Effect", p. 177; "Cone", "Cube", p. 179; "Cylinder", p. 180; "Emanation", p. 181; "Line", p. 184; "Sphere", p. 188 | unhappy (boundary) |
| FR-014 | `AreaShape.IncludesOrigin()` MUST be a static fact per shape: false for Cone, Cube, Emanation and Line, true for Cylinder and Sphere. The SRD's "unless its creator decides otherwise" is modelled by magical-effects.md FR-016 (`CreatorIncludesOrigin`, `OverridableAreaShape`), which leaves `IncludesOrigin` a static default per shape. See table A. | SRD 5.2.1, "Cone" and "Cube", p. 179; "Cylinder", p. 180; "Emanation", p. 181; "Line", p. 184; "Sphere", p. 188 | happy |
| FR-015 | `CreatureTypeRestriction` MUST be closed: `AnyCreatureType` or `OnlyCreatureTypes{Types}` built by `NewOnlyCreatureTypes([]CreatureType) (OnlyCreatureTypes, error)`, with a non-empty set of `CreatureType`; an empty set MUST be rejected with an error. A restriction other than `AnyCreatureType` MUST be rejected with an error when the subject is `object`. A creature restriction describes creatures only; which listed target is invalid under it (a creature of another type, an object under `SubjectCreatureOrObject`) is `spell-effects.md` FR-017 and FR-094. See table T. | SRD 5.2.1, "Invalid Targets", p. 106; "Hold Person", p. 141 ("Choose a Humanoid") | unhappy (invalid input) |
| FR-040 | `TargetsChosen` MUST hold a `SelfTargetPolicy`, `MayIncludeCaster` (the default) or `ExcludesCaster`; `TargetsSelf` and `TargetsInArea` hold none. `ExcludesCaster` states the SRD exceptions "the creature must be Hostile or specifically a creature other than you". That a creature really is Hostile is the host's fact and is not modelled (FR-004). What a resolver does with the policy is `spell-effects.md` FR-092. | SRD 5.2.1, "Targeting Yourself", p. 106 | happy |

### 4.3 Parts, outcomes and amounts

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | `PartResolution` MUST be closed with exactly `AttackRollResolution{Kind, OnHit, OnMiss}`, `SavingThrowResolution{Ability, OnFailure, OnSuccess}` and `AutomaticResolution{Outcomes}`, where `kind` is a `SpellAttackKind` (`SpellAttackMelee` or `SpellAttackRanged`) and `ability` is the save's `Ability`. The first two are the SRD's attack rolls and saving throws; `AutomaticResolution` (no roll) is package design for effects such as Magic Missile and Cure Wounds, which p. 106 does not name as a third hook. A `SpellEffectPart` holds one `PartResolution` and its `PartInstances` (FR-029). | SRD 5.2.1, "Effects" ("targets, saving throws, attack rolls, or all three"), p. 106; "Saving Throws" (the spell names the ability), p. 106; package design for `AutomaticResolution` | happy |
| FR-043 | `HalfDamageOnSuccess` (a `SuccessEffect`, beside `NothingOnSuccess`) MUST require at least one `SpellDealsDamage` in the `OnFailure` outcomes of its `SavingThrowResolution`, else an error (nothing to halve). | SRD 5.2.1, "Half Damage", p. 16; package design | unhappy (invalid input) |
| FR-044 | A `DamageTypeSelection` MUST be closed: `FixedDamageType(type)` or `ChosenDamageType{Types}` (a set); a `SpellDealsDamage` holds one. What the command must supply for a `ChosenDamageType` is `spell-effects.md` FR-033. | SRD 5.2.1, "Chromatic Orb" ("Choose Acid, Cold, Fire, Lightning, Poison, or Thunder"), p. 115; "Damage Types", p. 180 | happy |
| FR-045 | A model MUST have at most one `ChosenDamageType` among all its outcomes, because the command carries a single choice; more MUST be rejected with an error in `ModelledEffects`. A spell that makes several independent choices is `partial` and keeps them in text. | Package design (the command carries one choice); no SRD citation, because the pilot set has no spell with two independent damage type choices | unhappy (invalid input) |
| FR-046 | `ConditionDuration` MUST be closed with exactly `ForSpellDuration`, `TimedConditionDuration{Amount, Unit DurationUnit}` and `UntilNextTurnBoundary{Owner, Boundary}` (`TurnOwner` `OwnerCaster` or `OwnerTarget`; `TurnBoundary` `BoundaryStart` or `BoundaryEnd`). `amount` MUST be at least 1, else an error. `DurationUnit` is content-schema-and-decoder.md's (FR-149; shared-types-and-ownership.md FR-065). | SRD 5.2.1, "Hold Person" ("for the duration"), p. 141; "Ray of Sickness" ("until the end of your next turn"), p. 158; "Duration", p. 106 | unhappy (boundary) |
| FR-047 | `SpellOutcome` MUST be closed with exactly `SpellDealsDamage{Amount, Type, AddsModifier}` (`type` a `DamageTypeSelection`), `SpellRestoresHitPoints{Amount, AddsModifier}`, `SpellGrantsTemporaryHitPoints{Amount, AddsModifier}` and `SpellAppliesCondition{Kind, Duration}`. An `amount` is a `ScalableAmount{Base, Scaling}` over a closed `SpellAmount` (`RolledSpellAmount{dice.Expression}` or `FixedSpellAmount{int}`); `AddsModifier` is a `ModifierAddition` (`AddsNothing` or `AddsSpellcastingAbilityModifier`). | SRD 5.2.1, "Effects", p. 106; "Damage Rolls", p. 16; "Healing", p. 17; "Temporary Hit Points", p. 18; "Condition", p. 179; "Cure Wounds" ("2d8 plus your spellcasting ability modifier"), p. 121 | happy |
| FR-025 | A `FixedSpellAmount` MUST be at least 1, else an error; a `RolledSpellAmount` wraps a `dice.Expression` and inherits its validation (count at least 1). | SRD 5.2.1, "Heal" (70 Hit Points), p. 139; dice-and-rolls.md FR-006, FR-007 | unhappy (boundary) |
| FR-026 | Outcome lists MUST be non-empty where an effect cannot be empty: `AttackRollResolution.OnHit`, `SavingThrowResolution.OnFailure` and `AutomaticResolution.Outcomes`, else an error. `AttackRollResolution.OnMiss` MAY be empty. | `doc/rules/testing.md` §4 (empty and minimal); package design | unhappy (empty and minimal) |
| FR-027 | A `ChosenDamageType` set MUST hold at least 2 distinct `DamageType`s; one type MUST be a `FixedDamageType`. | SRD 5.2.1, "Chromatic Orb", p. 115 | unhappy (boundary) |
| FR-028 | A `SpellAppliesCondition` outcome MUST hold a `ConditionKind` and a `ConditionDuration` (FR-046) and MUST accept any of the fifteen `ConditionKind` values (the closed set is owned upstream: content-schema-and-decoder.md FR-110). For `Exhaustion` it carries no level: one application adds one level (`spell-effects.md` FR-038). | SRD 5.2.1, "Condition", p. 179 | happy |
| FR-029 | `PartInstances` MUST have a base of at least 1 and an optional `CountScaling`; 0 and negative bases MUST be rejected with an error. | SRD 5.2.1, "Eldritch Blast" (one beam at level 1), p. 127; package design | unhappy (boundary) |

### 4.4 Scaling

Scaling is a pure query on the model. It takes plain integers, so `content`
imports nothing from `character` (FR-041).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-041 | The scaling queries ((`ScalableAmount`, `CountScaling` and the character-level tables)) MUST take plain integers: `n`, the number of slot levels above `AboveLevel` (at least 0; a negative `n` MUST be rejected with an error), and the total character level (at least 1, at most 20, else an error). `PerSlotLevel...` reads `n`, `ByCharacterLevel...` reads the level, and `No...Scaling` reads neither. They MUST NOT import `CastReport` or any other `character` type; the caller derives `n` and the level from the cast (`spell-effects.md` FR-043, FR-050). | `doc/rules/architecture.md` §2 (context map: `content` is upstream of `character`); package design | unhappy (invalid input) |
| FR-016 | `PerSlotLevelAmount{AboveLevel, Step}` MUST apply an `AmountStep{ExtraDice, ExtraFlat}` n times: for a `RolledSpellAmount` the dice count grows by n times `ExtraDice` (the base die) and the flat modifier by n times `ExtraFlat`; for a `FixedSpellAmount` only the flat value grows. A step with both parts 0, a negative part, or `ExtraDice` on a fixed base MUST be rejected with an error. See table L. | SRD 5.2.1, "Fireball", p. 131 (1d6 per level above 3); "False Life", p. 129 (5 per level above 1); "Heal", p. 139 (10 per level above 6) | unhappy (invalid input) |
| FR-017 | `PerSlotLevelCount{AboveLevel, Extra}` MUST add n times `Extra` (at least 1) to a target count or an instance count. | SRD 5.2.1, "Hold Person" (one additional Humanoid per slot level above 2), p. 141; "Magic Missile" (one more dart per level above 1), p. 146 | happy |
| FR-018 | A character-level table MUST select the value of the entry with the greatest level not above the total character level it is given (never a class level, unless the spell says otherwise). See table C. | SRD 5.2.1, "Acid Splash" (Cantrip Upgrade), p. 107; "Multiclassing", "Cantrips", p. 25; "Cantrip", p. 178; spellcasting.md FR-038 | unhappy (boundary) |
| FR-019 | A character-level table (`AmountLevelTable` or `CountLevelTable`) MUST have at least one entry, its first entry at level 1, strictly ascending levels, and each level between 1 and 20 inclusive (not an entry for every level: the printed tables have four), else an error. A table starts at level 1 with a valid amount, so a Cantrip Upgrade that only adds a new effect from a later level (True Strike: extra Radiant damage from level 5, SRD 5.2.1 p. 171) cannot be modelled and makes the spell `partial` (FR-034; table C, C19). The model does not require the levels 5, 11 and 17; that the SRD's cantrips use them is a bundle check (FR-033). | SRD 5.2.1, "Fire Bolt", p. 132 (levels 5, 11 and 17); package design | unhappy (invalid input) |
| FR-020 | A `ByCharacterLevelAmount` table MUST hold the printed totals as replacement values (1d10, 2d10, 3d10, 4d10), not increments, and a `ByCharacterLevelCount` table the printed counts (Eldritch Blast: 1, 2, 3, 4 beams). | SRD 5.2.1, "Fire Bolt", p. 132; "Eldritch Blast", p. 127 | happy |
| FR-021 | Two outcomes of one part MUST scale independently, each by its own scaling (Flame Strike: Fire and Radiant damage each grow by 1d6 per level above 5). | SRD 5.2.1, "Flame Strike", p. 132 | happy |

### 4.5 Derived facts

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | `SpellEffectModel` MUST answer `SpellEffectFacts` with these facts, each an `EffectFact` (`FactKnown{Value}`, `FactKnownDamageTypes` for the `DamageTypes` set, or `FactUnknown`): `ForcesSavingThrow`, `MakesSpellAttackRoll`, `DealsDamage`, `DamageTypes` (a set of `DamageType`) and `CanTargetMoreWithHigherSlot` (true when the targets' count has a `PerSlotLevelCount`). More instances (darts, rays) are not additional targets for this fact. | SRD 5.2.1, "Effects", p. 106; consumers of these facts: class-sorcerer.md `MetamagicSpellFacts`, class-wizard.md `WizardSpellFacts.DealsDamage` and class-warlock.md `CantripFacts` (its FR-156, table 4.6.8). Ownership: `SpellEffectFacts` is the authoritative definition of these five facts (shared-types-and-ownership.md FR-048); the consumers change to read it (RA-1) | happy |
| FR-023 | A fact is `FactKnown{true}` when the modelled parts show it, in any coverage. `FactKnown{false}` and `DamageTypes` are known only when the coverage is `complete`; with `partial` coverage they MUST be `FactUnknown`, because the text may hold what the model leaves out. See table F. | SRD 5.2.1, "Effects", p. 106; `AGENTS.md` (never guess content) | unhappy (rule interaction) |
| FR-024 | `TextOnlyEffects` MUST answer `FactUnknown` for every fact. A consumer MUST NOT read "unknown" as "no". | `AGENTS.md`, "Licensing and source material"; class-sorcerer.md (a missing fact gives `SpellFactUnknown`, not a guess) | unhappy (missing content) |

### 4.6 Decoder, bundling and verification

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | The spell JSON object MUST accept an optional field `effects` holding the model; absent means `TextOnlyEffects`. It is an additive change, requested as one row (`R2-spell-effects`) in the revision table of content-schema-and-decoder.md under the single `SchemaVersion` (shared-types-and-ownership.md FR-034); no separate version is claimed here. | shared-types-and-ownership.md FR-034, FR-035; content-schema-and-decoder.md Q7 (structure "as additive fields in a later `SchemaVersion`") | happy |
| FR-031 | The decoder MUST report every problem in effect data, each with a path and a kind, not only the first: an unknown variant name as `InvalidValue`, a missing required field as `MissingField`, a wrong JSON type as `WrongType`, an unknown key as `UnknownField`, and a value that breaks a constructor rule with the kind of the table below (content-schema-and-decoder.md FR-015, FR-023, FR-024, FR-026, FR-094). The constructor-level error and the decoder kind are two reports of one rule (FR-032).<br>**Rule to kind:** `OutOfRange` (an integer or an array length outside its bounds): FR-006 (empty parts list), FR-012 (count below 1), FR-013 (dimension below 1), FR-015 (empty creature type set), FR-046 (`amount` below 1), FR-016 (negative step part or both parts 0), FR-017 (`extra` below 1), FR-019 (empty table, a level outside 1 to 20), FR-025 (amount below 1), FR-026 (empty outcome list), FR-027 (set of fewer than 2 types), FR-029 (base below 1). `ConstraintViolation` (a rule across fields): FR-009 (scaling that contradicts the spell's level), FR-015 (restriction other than `AnyCreatureType` with subject `object`), FR-043 (`HalfDamageOnSuccess` with no `SpellDealsDamage`), FR-045 (more than one `ChosenDamageType`), FR-016 (`ExtraDice` on a fixed base), FR-019 (first level not 1; levels not strictly ascending). `duplicate`: a repeated member of a set (content-schema-and-decoder.md FR-026). `InvalidValue`: an unknown variant or enum spelling. | `doc/rules/architecture.md` §5 (all problems at once); content-schema-and-decoder.md (problem kinds) | unhappy (invalid input) |
| FR-032 | A table-driven agreement test MUST feed every invalid and valid value of this spec's constructor rules both to the decoder and to the matching constructor, as content-schema-and-decoder.md FR-094 requires for other types, so a rule changed in one place and not the other fails. | content-schema-and-decoder.md FR-094; `doc/rules/testing.md` §8 | unhappy (rule interaction) |
| FR-033 | Bundle verification MUST check a bundled spell's model against its verbatim text, and report a finding for each modelled token that the text does not print: every dice notation (whitespace ignored, so `2d4 + 4` matches), damage type, save ability with "saving throw", condition name, attack kind ("ranged spell attack" or "melee spell attack"), and area shape with its size. It MUST also check every number the model holds against the text: fixed amounts (Heal's 70), target counts ("up to three creatures"), instance counts ("three darts"), every `AmountStep` value, `AboveLevel`, `PerSlotLevelCount.Extra`, each value of a character-level table, and each condition `Amount`. A number may be printed as a numeral or a number word; for a target or instance count, "a", "an" or an unprinted singular means 1 and satisfies `ExactlyTargets{1}` (Fire Bolt "a creature or an object", Eldritch Blast's base beam). It MUST also report any bundled cantrip table whose levels are not 1, 5, 11 and 17. A model is a reading of the text, never a source of facts. | content-spells.md FR-064 (text is verbatim); SRD 5.2.1, "Acid Splash", p. 107 | unhappy (rule interaction: model and text disagree) |
| FR-034 | Scaling in the model MUST agree with the spell's `HigherLevel` and `CantripUpgrade` text: each modelled scaling number is printed there, and a spell whose `HigherLevel` or `CantripUpgrade` text describes a change the model has no scaling for (a longer duration, a larger radius, a changed stat block) MUST be `partial`. | SRD 5.2.1, "Using a Higher-Level Spell Slot", pp. 104-105; "Bestow Curse", pp. 112-113 (longer Concentration by slot level); "Create or Destroy Water", p. 121 (larger Cube); "Hex", p. 140 (longer Concentration); "True Strike", p. 171 (extra Radiant damage only from level 5, which a table starting at level 1 cannot hold, FR-019, so the spell is `partial`) | unhappy (rule interaction) |
| FR-035 | The extraction tool of content-spells.md (its FR-071 to FR-078) MUST NOT drop, alter or reorder `effects` when it rewrites the text of a bundled spell, its drift check (content-spells.md FR-078) MUST ignore `effects`, and an annotation for a spell that no longer exists MUST be reported as a problem. | content-spells.md FR-073, FR-077, FR-078 | unhappy (state error: regeneration drops annotations) |
| FR-036 | Population MUST be incremental: a spell with no annotation stays `TextOnlyEffects`. The bundle tooling MUST report the counts of `complete`, `partial` and text-only spells. A count is pinned in a test only after the first annotation run has been reviewed; this spec pins no count except the pilot set (FR-037). | content-spells.md section 2 (counts confirmed by the first run); package design | unhappy (missing content) |
| FR-037 | The 16 spells of table P MUST carry exactly the model that table states, asserted field by field in spot-check fixtures. | SRD 5.2.1, as cited per row in table P | happy |
| FR-038 | Each annotated spell MUST have a human review record (spell, page, reviewer, date) in the extraction notes, as content-spells.md FR-080 requires for text; the annotator MUST NOT supply a value the text does not print. | content-spells.md FR-080; `AGENTS.md`, "Licensing and source material" | unhappy (missing content: unreviewed entry) |

### 4.7 Layering and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-039 | The model types (section 5.2) MUST live in package `content`, in files named `spell_effects*.go`, import only kernel packages (`dice`, `ability`, `srderr`) and other `content` files, and be exported from package `content`. They MUST contain no `context.Context`, goroutine, clock, random or I/O, and import nothing from `combat` or `character`. | `doc/rules/architecture.md` §2, §3, §9; `spell-effects.md` FR-090 | happy |

### Table A: area shapes (FR-013, FR-014; one test per row)

| Label | Shape | Dimensions (feet, each at least 1) | `IncludesOrigin` | Source | Path |
|---|---|---|---|---|---|
| A1 | `ConeArea` | `LengthFeet` | false | "Cone", p. 179 | happy |
| A2 | `CubeArea` | `SideFeet` | false | "Cube", p. 179 | happy |
| A3 | `CylinderArea` | `RadiusFeet`, `HeightFeet` | true | "Cylinder", p. 180 | happy |
| A4 | `EmanationArea` | `DistanceFeet` | false | "Emanation", p. 181 | happy |
| A5 | `LineArea` | `LengthFeet`, `WidthFeet` | false | "Line", p. 184 | happy |
| A6 | `SphereArea` | `RadiusFeet` | true | "Sphere", p. 188 | happy |
| A7 | `NewSphereArea(0)` | radius 0 | n/a | FR-013 | unhappy (boundary) |
| A8 | `NewSphereArea(1)` | radius 1 | true | FR-013 | unhappy (boundary) |
| A9 | `NewLineArea(100, 0)` | width 0 | n/a | FR-013 | unhappy (boundary) |
| A10 | `NewCylinderArea(10, -40)` | negative height | n/a | FR-013 | unhappy (invalid input) |

### Table T: creature type restriction (FR-015; one test per row)

| Label | Construction | Expected | Path |
|---|---|---|---|
| T1 | `NewOnlyCreatureTypes(nil)` | error (`errors.Is` `srderr.ErrInvalidInput`) | unhappy (invalid input) |
| T2 | `NewOnlyCreatureTypes([Humanoid])` with subject `object` | error (`errors.Is` `srderr.ErrInvalidInput`) | unhappy (invalid input) |
| T3 | `NewOnlyCreatureTypes([Humanoid])` with subject `creature` (Hold Person, p. 141) | valid | happy |

### Table L: scaling with the slot level (FR-016, FR-017, FR-021, FR-041; one test per row)

`n` is the input of the query (`spell-effects.md` FR-043 derives it from the
cast: the larger of 0 and `CastAtLevel` minus `AboveLevel`).

| Label | Spell and page | Base | Step | n | Expected | Path |
|---|---|---|---|---|---|---|
| L1 | Fireball, p. 131 | 8d6 | +1d6 above 3 | 0 | 8d6 | unhappy (boundary) |
| L2 | Fireball, p. 131 | 8d6 | +1d6 above 3 | 1 | 9d6 | happy |
| L3 | Fireball, p. 131 | 8d6 | +1d6 above 3 | 6 | 14d6 | unhappy (boundary) |
| L4 | Cure Wounds, p. 121 | 2d8 | +2d8 above 1 | 0 | 2d8 | unhappy (boundary) |
| L5 | Cure Wounds, p. 121 | 2d8 | +2d8 above 1 | 1 | 4d8 | happy |
| L6 | Cure Wounds, p. 121 | 2d8 | +2d8 above 1 | 8 | 18d8 | unhappy (boundary) |
| L7 | False Life, p. 129 | 2d4 + 4 | +5 flat above 1 | 0 | 2d4 + 4 | unhappy (boundary) |
| L8 | False Life, p. 129 | 2d4 + 4 | +5 flat above 1 | 2 | 2d4 + 14 | happy |
| L9 | False Life, p. 129 | 2d4 + 4 | +5 flat above 1 | 8 | 2d4 + 44 | unhappy (boundary) |
| L10 | Heal, p. 139 | fixed 70 | +10 above 6 | 0 | 70 | unhappy (boundary) |
| L11 | Heal, p. 139 | fixed 70 | +10 above 6 | 1 | 80 | happy |
| L12 | Heal, p. 139 | fixed 70 | +10 above 6 | 3 | 100 | unhappy (boundary) |
| L13 | Flame Strike, p. 132 | 5d6 Fire and 5d6 Radiant | each +1d6 above 5 | 2 | 7d6 Fire and 7d6 Radiant | happy |
| L14 | Magic Missile, p. 146 | 3 darts | +1 above 1 | 0 | 3 | unhappy (boundary) |
| L15 | Magic Missile, p. 146 | 3 darts | +1 above 1 | 1 | 4 | happy |
| L16 | Magic Missile, p. 146 | 3 darts | +1 above 1 | 8 | 11 | unhappy (boundary) |
| L17 | Hold Person, p. 141 | exactly 1 target | +1 above 2 | 0 | exactly 1 | unhappy (boundary) |
| L18 | Hold Person, p. 141 | exactly 1 target | +1 above 2 | 1 | exactly 2 | happy |
| L19 | Hold Person, p. 141 | exactly 1 target | +1 above 2 | 7 | exactly 8 | unhappy (boundary) |
| L21 | `PerSlotLevelAmount` with step 0d6 and flat 0 | n/a | n/a | n/a | an error (FR-016) | unhappy (invalid input) |
| L22 | `AmountStep` with `ExtraDice` on a fixed base | fixed 70 | +1d6 | n/a | an error (FR-016) | unhappy (invalid input) |
| L24 | Fireball, p. 131, query with a negative `n` | 8d6 | +1d6 above 3 | -1 | an error (FR-041) | unhappy (invalid input) |

L20 and L23 (a Ritual with no `CastAtLevel`; a cantrip with no `CastAtLevel`)
test the derivation of `n` and are rows N6 and N7 of `spell-effects.md`.

### Table C: scaling with the character level (FR-018, FR-019, FR-020, FR-041; one test per row)

| Label | Spell and page | Total character level | Expected | Path |
|---|---|---|---|---|
| C1 | Fire Bolt, p. 132 (1d10, 2d10, 3d10, 4d10 at 1, 5, 11, 17) | 1 | 1d10 | unhappy (boundary) |
| C2 | Fire Bolt | 4 | 1d10 | unhappy (boundary) |
| C3 | Fire Bolt | 5 | 2d10 | unhappy (boundary) |
| C4 | Fire Bolt | 10 | 2d10 | unhappy (boundary) |
| C5 | Fire Bolt | 11 | 3d10 | unhappy (boundary) |
| C6 | Fire Bolt | 16 | 3d10 | unhappy (boundary) |
| C7 | Fire Bolt | 17 | 4d10 | unhappy (boundary) |
| C8 | Fire Bolt | 20 | 4d10 | unhappy (boundary) |
| C9 | Eldritch Blast, p. 127 (1, 2, 3, 4 beams at 1, 5, 11, 17) | 4 | 1 beam | unhappy (boundary) |
| C10 | Eldritch Blast | 5 | 2 beams | unhappy (boundary) |
| C11 | Eldritch Blast | 11 | 3 beams | unhappy (boundary) |
| C12 | Eldritch Blast | 17 | 4 beams | unhappy (boundary) |
| C13 | Acid Splash, p. 107 (1d6, 2d6, 3d6, 4d6) | 17 | 4d6 | happy |
| C14 | Multiclass: character level 5 from Wizard 3 and Cleric 2, Fire Bolt | 5 (total character level, not 3) | 2d10 | unhappy (rule interaction) |
| C15 | Table whose first entry is level 2 | n/a | an error (FR-019) | unhappy (invalid input) |
| C16 | Table with levels 5 then 5 | n/a | an error (FR-019) | unhappy (invalid input) |
| C17 | Table with a level 21 entry | n/a | an error (FR-019) | unhappy (boundary) |
| C19 | True Strike, p. 171 (Cantrip Upgrade adds extra Radiant damage only from level 5: 1d6, 2d6, 3d6 at 5, 11, 17) | n/a | not expressible as a table that starts at level 1 (FR-019); the spell is `partial` and the upgrade stays text | unhappy (rule interaction) |
| C20 | Fire Bolt, query with level 0 or 21 | 0, 21 | an error (FR-041) | unhappy (boundary) |

C18 (a cantrip cast whose report has no `ScalingLevel`) is the use case's
an error and is row N8 of `spell-effects.md` (FR-050).

### Table F: derived facts (FR-022, FR-023, FR-024; one test per row)

| Label | Spell and coverage | Fact | Answer | Path |
|---|---|---|---|---|
| F1 | Eldritch Blast, `complete` | `ForcesSavingThrow` | `FactKnown{false}` | happy |
| F2 | Eldritch Blast, `complete` | `MakesSpellAttackRoll` | `FactKnown{true}` | happy |
| F3 | Eldritch Blast, `complete` | `DamageTypes` | `FactKnown({Force})` | happy |
| F4 | Eldritch Blast, `complete` | `CanTargetMoreWithHigherSlot` | `FactKnown{false}` | unhappy (boundary) |
| F5 | Fire Bolt, `partial` | `MakesSpellAttackRoll` | `FactKnown{true}` | happy |
| F6 | Fire Bolt, `partial` | `ForcesSavingThrow` | `FactUnknown` | unhappy (rule interaction) |
| F7 | Fire Bolt, `partial` | `DamageTypes` | `FactUnknown` | unhappy (rule interaction) |
| F8 | Fireball, `partial` | `ForcesSavingThrow` | `FactKnown{true}` | happy |
| F9 | Hold Person, `partial` | `CanTargetMoreWithHigherSlot` | `FactKnown{true}` | happy |
| F10 | Magic Missile, `complete` | `CanTargetMoreWithHigherSlot` | `FactKnown{false}` (more darts, not more targets) | unhappy (boundary) |
| F11 | a synthetic `complete` spell with a `ChosenDamageType` of six types | `DamageTypes` | `FactKnown` of the six types | happy |
| F12 | any `TextOnlyEffects` | every fact | `FactUnknown` | unhappy (missing content) |

### Table P: pilot set (FR-037; one spot-check fixture per row)

Every spell below is read from the cited page, including its `TargetSubject` (checked against the printed text: Fire Bolt, Eldritch Blast and Chromatic Orb print a target or a "creature or an object"; Ice Knife prints "one creature"). "Residue" is the text the model does not cover, which is why the coverage is `partial` where it is.

| Label | Spell and page | Targeting | Part and outcomes | Scaling | Coverage and residue |
|---|---|---|---|---|---|
| P1 | Acid Splash, p. 107 | area `SphereArea(5)`, creature | Dexterity save; failure: 1d6 Acid; `NothingOnSuccess` | table 1d6, 2d6, 3d6, 4d6 at 1, 5, 11, 17 | `complete` |
| P2 | Fire Bolt, p. 132 | chosen, `SubjectCreatureOrObject`, exactly 1 | ranged attack; hit: 1d10 Fire | table 1d10 to 4d10 | `partial`: a flammable object hit starts burning |
| P3 | Eldritch Blast, p. 127 | chosen, `SubjectCreatureOrObject`, by instance | ranged attack; hit: 1d10 Force; instances 1 | instance table 1, 2, 3, 4 at 1, 5, 11, 17 | `complete` |
| P4 | Magic Missile, p. 146 | chosen, `creature`, by instance | automatic; 1d4 + 1 Force per dart; instances 3 | +1 dart per level above 1 | `complete` |
| P5 | Cure Wounds, p. 121 | chosen, `creature`, exactly 1 | automatic; restores 2d8 plus the spellcasting ability modifier | +2d8 per level above 1 | `complete` |
| P6 | Healing Word, p. 139 | chosen, `creature`, exactly 1 | automatic; restores 2d4 plus the spellcasting ability modifier | +2d4 per level above 1 (`AmountStep(extraDice: 2)` on a 2d4 base) | `complete` |
| P7 | False Life, p. 129 | self | automatic; Temporary Hit Points 2d4 + 4 | +5 flat per level above 1 | `complete` |
| P8 | Ray of Sickness, p. 158 | chosen, `creature`, exactly 1 | ranged attack; hit: 2d8 Poison and Poisoned until the next end of the caster's turn | +1d8 per level above 1 | `complete` |
| P9 | Burning Hands, p. 114 | area `ConeArea(15)`, creature | Dexterity save; failure: 3d6 Fire; `HalfDamageOnSuccess` | +1d6 per level above 1 | `partial`: flammable objects start burning |
| P10 | Fireball, p. 131 | area `SphereArea(20)`, creature | Dexterity save; failure: 8d6 Fire; `HalfDamageOnSuccess` | +1d6 per level above 3 | `partial`: flammable objects start burning |
| P11 | Flame Strike, p. 132 | area `CylinderArea(10, 40)`, creature | Dexterity save; failure: 5d6 Fire and 5d6 Radiant; `HalfDamageOnSuccess` | each +1d6 per level above 5 | `complete` |
| P12 | Hold Person, p. 141 | chosen, `creature`, `NewOnlyCreatureTypes([Humanoid])`, exactly 1 | Wisdom save; failure: Paralyzed for the spell duration; `NothingOnSuccess` | +1 target per level above 2 | `partial`: the save repeated at the end of each of its turns |
| P13 | Heal, p. 139 | chosen, `creature`, exactly 1 | automatic; restores a fixed 70 | +10 per level above 6 | `partial`: ends Blinded, Deafened and Poisoned |
| P14 | Harm, pp. 139-140 | chosen, `creature`, exactly 1 | Constitution save; failure: 14d6 Necrotic; `HalfDamageOnSuccess` | none | `partial`: Hit Point maximum reduced by the damage taken |
| P15 | Ice Knife, p. 141 ("fling it at one creature within range") | chosen, `creature`, exactly 1 (the attack target) | ranged attack; hit: 1d10 Piercing | +1d6 Cold per level above 1 is the explosion's, not modelled | `partial`: the explosion (Dexterity save, 2d6 Cold, "the target and each creature within 5 feet of it") is relative to a target and stays text |
| P16 | Chromatic Orb, p. 115 | chosen, `SubjectCreatureOrObject` ("a target within range"), exactly 1 | ranged attack; hit: 3d8 of a `ChosenDamageType` {Acid, Cold, Fire, Lightning, Poison, Thunder} | +1d8 per level above 1 | `partial`: the leap to a different target when two or more d8s show the same number (a target set that depends on rolled values, table E E17) |

### Table E: SRD effect patterns and the decision for each (informs FR-004, FR-005, FR-034, FR-042; one classification fixture per row)

The verdict column is the decision of this spec: "modelled" means a variant
exists; "text" means it stays in the description (a new variant is a Q1 matter).

| Label | Pattern | Example (page) | Verdict | Reason | Path |
|---|---|---|---|---|---|
| E1 | spell attack roll, damage on a hit | Fire Bolt (p. 132) | modelled | `AttackRollResolution` | happy |
| E2 | several attack rolls, each its own damage | Eldritch Blast (p. 127), Scorching Ray (p. 159) | modelled | `PartInstances` with an assignment | happy |
| E3 | saving throw, damage, half on a success | Fireball (p. 131) | modelled | `HalfDamageOnSuccess` | happy |
| E4 | saving throw, damage, nothing on a success | Acid Splash (p. 107) | modelled | `NothingOnSuccess` | happy |
| E5 | saving throw, condition for the duration | Hold Person (p. 141) | modelled | `SpellAppliesCondition`; the repeated save is E15 | happy |
| E6 | no roll, fixed dice per dart | Magic Missile (p. 146) | modelled | `AutomaticResolution` | happy |
| E7 | healing with the spellcasting ability modifier | Cure Wounds (p. 121) | modelled | `ModifierAddition` | happy |
| E8 | Temporary Hit Points | False Life (p. 129) | modelled | `SpellGrantsTemporaryHitPoints` | happy |
| E9 | two Damage Types in one effect | Flame Strike (p. 132), Ice Storm (p. 141) | modelled | two outcomes | happy |
| E10 | Damage Type chosen by the caster | Chromatic Orb (p. 115, pilot P16) | modelled | `ChosenDamageType` (one choice, FR-045); the spell is `partial` because the leap is E17 | happy |
| E11 | dice, targets or instances grow per slot level | Fireball, Hold Person, Magic Missile | modelled | `PerSlotLevel...` | happy |
| E12 | cantrip grows at levels 5, 11 and 17 | Fire Bolt (p. 132), Eldritch Blast (p. 127) | modelled | `ByCharacterLevel...` | happy |
| E13 | Hit Point maximum increased or reduced | Aid (p. 107), Harm (pp. 139-140) | text | pure domain operations outside the effect data: Aid and Heroes' Feast in revival-and-restoration-spells.md (FR-023 to FR-030), the reduction in hit-points-and-damage.md; the spell stays `partial` | unhappy (rule interaction) |
| E14 | duration or Concentration extended by slot level | Hex (p. 140), Bestow Curse (pp. 112-113) | text | runtime state is active-effects.md; a later data spec maps `Spell.effects` to its `EffectTemplate`, until then the row stays text | unhappy (rule interaction) |
| E15 | repeated saves, effects each turn, delayed damage | Hold Person (p. 141), Acid Arrow (p. 107) | text | as E14 (active-effects.md); a lingering area (Flaming Sphere, p. 132) is zone facts, FR-048 | unhappy (rule interaction) |
| E16 | riders on later attack rolls or saves | Hunter's Mark (p. 141), Hex (p. 140), Bane (p. 112) | text | riders (Bless, Bane, Hex, Hunter's Mark damage) are `EffectRiders` of active-effects.md; numeric modifiers while active (Guidance, Haste, Slow, Shield) are spell-granted-modifiers.md; as E14 for the data mapping | unhappy (rule interaction) |
| E17 | an effect that jumps to another target when dice show matching numbers | Chromatic Orb (p. 115, pilot P16) | text | the leap depends on rolled values; this is the residue that makes P16 `partial` | unhappy (rule interaction) |
| E18 | a target set relative to another target | Ice Knife (p. 141), Chain Lightning (p. 114: a target you choose, then up to three other targets of your choice within 30 feet of the first) | text | `TargetsInArea` is about a point, not a target; the later targets are relative to the first, not to rolled values | unhappy (rule interaction) |
| E19 | summoned creatures and stat blocks | Find Steed (p. 130) | text | the numeric level expressions of the four spell stat blocks are typed in summoned-and-companion-creatures.md (FR-029 to FR-037) with the text kept embedded (content-spells.md FR-069); no summon variant here | unhappy (rule interaction) |
| E20 | terrain, light, walls, objects | Ice Storm (p. 141), Blade Barrier (p. 113) | text | not creature outcomes | unhappy (rule interaction) |
| E21 | choose one of several modes | Eyebite (p. 129), Command (p. 116) | text | modes need a choice input per mode (Q1) | unhappy (rule interaction) |
| E22 | removing a condition | Heal (p. 139) | text | a pure domain operation in revival-and-restoration-spells.md (Heal: FR-037 to FR-040), not an outcome variant; the spell stays `partial` | unhappy (rule interaction) |
| E23 | cover denied to a save | Sacred Flame (p. 159) | text | the host builds the `SaveAttempt`, so it applies the rule from the text | unhappy (rule interaction) |
| E24 | a divided pool of Hit Points | Mass Heal (p. 147) | text | needs a distribution input | unhappy (rule interaction) |
| E25 | movement, teleportation, perception | Blink (pp. 113-114), Fear (p. 130) | text | not an outcome the engine can compute; Blink's d6 table is rolled by spell-outcome-tables.md (with Teleport, Confusion, Prismatic Spray and Wall, Reincarnate), which owns those die-chosen tables (FR-048 of shared-types-and-ownership.md) | unhappy (rule interaction) |
| E26 | a Cantrip Upgrade that adds a new effect from a later level, not a larger value | True Strike (p. 171, extra Radiant damage from level 5) | text | a table starts at level 1 with a valid amount (FR-019), so the spell is `partial` | unhappy (rule interaction) |

### 4.8 Zone facts (the static half of persistent-spell-zones.md)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-048 | `ModelledEffects` MAY hold one optional `ZoneFacts` (default none; a zone spell with `TextOnlyEffects` has none). `ZoneFacts{Rules, Mobility, OncePerTurn, Designation, DamageBudget, StopsMoverOnContact, Movement}` MUST have at least one `ZoneRuleFact{Trigger, PartIndex, Scope, Region, UnitFeet, CasterChoice}`. `ZoneTrigger`: `CreatureEnters`, `CreatureFirstEntryOnTurn`, `CreatureEndsTurnInside`, `CreatureStartsTurnInside`, `ZoneMovesIntoSpace`, `TravelsWithin`; `UnitFeet` (at least 1) MUST be set for `TravelsWithin` and for no other trigger. `ZoneTargetScope`: `AnyCreature`, `EnemiesOfCaster`, `CreaturesCasterCanSee`. `ZoneRegionID`: non-blank, default `body`. `ZoneMobility` (closed): `StationaryZone`, `MovesWithOrigin`, `MovedByCaster{MaxFeet, Action, LaterTurnsOnly}`, `DriftsAtTurnStart{Feet, Direction}`; `ZoneActionKind`: `BonusAction`, `MagicAction`, `WithCasterMovement`; `DriftDirectionRule`: `AwayFromCaster`, `AwayFromCasterChosen`; `MaxFeet`, `Feet` and `DamageBudget` at least 1. `DesignationPolicy`: `none`, `AtCastingOnly`. `ZoneMovementFacts{DifficultTerrain, SpeedHalved, ExtraFeetPerFoot}` with `ExtraFeetPerFoot` at least 0 (the zero value means none). The types hold facts only: no state, ledger or event (those are persistent-spell-zones.md). A rule's `PartIndex` names a part of the same model but is not range-checked by the constructor (persistent-spell-zones.md FR-003 refuses it at creation); FR-049 reports it. | SRD 5.2.1, "Spirit Guardians", pp. 164-165; "Cloudkill", p. 116; "Moonbeam", pp. 150-151; "Spike Growth", p. 164; "Wall of Thorns", p. 173; "Guardian of Faith", p. 137; "Flaming Sphere", p. 132; "Difficult Terrain", p. 181 | unhappy (boundary) |
| FR-049 | The decoder MUST report zone-fact problems with FR-031's kinds: `OutOfRange` for an empty rule list, `UnitFeet`, `MaxFeet`, `Feet` or `DamageBudget` below 1, `ExtraFeetPerFoot` below 0; `ConstraintViolation` for `UnitFeet` missing on `TravelsWithin` or present on another trigger, and for a `PartIndex` outside the model's parts (the cross-field check is a decoder and bundle check, not a constructor rule, so a hand-made `ZoneFacts` is valid until used); `duplicate` for a repeated identical rule. Bundle verification MUST also check every number the facts hold against the spell text (`UnitFeet`, `MaxFeet`, `Feet`, `DamageBudget`; `ExtraFeetPerFoot` is the printed cost minus 1: Wall of Thorns' 4 gives 3), that `DifficultTerrain` and `SpeedHalved` are printed ("Difficult Terrain", "Speed is halved"), that `Designation` `DesignationAtCastingOnly` has a printed "designate", and that `MovedByCaster.Action` matches the printed action. Facts the text does not print MUST NOT be supplied (FR-033). | `doc/rules/architecture.md` §5; SRD 5.2.1, "Wall of Thorns", p. 173; "Guardian of Faith", p. 137 | unhappy (rule interaction: model and text disagree) |
| FR-050 | The zone pilot set is the spells of persistent-spell-zones.md table Z (shape, mobility, rules, ledger and other facts per row, stated there once and not repeated here); each MUST carry exactly the `ZoneFacts` that row states, asserted field by field with FR-037's fixtures. Coverage follows FR-004: a spell whose text has an outcome that is the host's (a push, a strong wind, a stated Advantage) is `partial`. | SRD 5.2.1, as cited per row in table Z | happy |

## 5. Model

### 5.1 The extension-point decision (architecture.md section 3)

The question is how the package lets spell effects be resolved, and who can add
one. Four options were weighed against the architecture rules.

| Option | What it is | For | Against | Verdict |
|---|---|---|---|---|
| A. Text only | Keep effects as description text (today) | Nothing to build; faithful | Nothing resolves a spell; every consumer re-reads prose; class specs invent their own spell facts (class-sorcerer.md `MetamagicSpellFacts`, class-wizard.md `WizardSpellFacts`, class-warlock.md `CantripFacts`, table 4.6.8) | Rejected as the end state; kept as the default per spell |
| B. Data-described effects | A closed, decodable model in `content`, interpreted by the standard rules in `combat` | Matches "effects expressed as data" and "house rules and new content arrive as content" (architecture.md §3); a house-rule spell needs no Go code; decoder-validated with all problems at once (`errors.Join` of typed problems); every rule is table-testable; `content` stays upstream and holds static facts only | A closed variant set cannot express every spell; mis-modelling risk | **Recommended** |
| C. A strategy object per spell | A `SpellBehaviour` keyed by `SpellID`, with code per spell | Total freedom per spell | 339 classes of code; code cannot arrive as content, so consumers need Go code for every house rule; the behaviour would sit upstream of or keyed from `content`, against the context map; nothing for a decoder to validate; tests per spell instead of per rule | Rejected |
| D. Hybrid | B plus a registry of C for what B cannot express | Escape hatch | Adds a code path nobody has asked for ("only when a real need appears") | Deferred (Q1) |

Reasons for B (architecture.md §1): the resolver is a pure function of a model,
a cast and rolled dice; the handoff is a value object, not a callback; the
variants are the SRD's own hooks (p. 106). The risk is mis-modelling, so the
model is **optional and honest**: `EffectCoverage` says how much text it covers,
facts never answer "no" for a `partial` spell (FR-023), and bundle verification
checks every token against the text (FR-033). What B cannot express stays text,
and `partial` tells the host the rest is the GM's (p. 106).

### 5.2 Content model (package `content`, files `spell_effects*.go`; published language of `content`)

All immutable (unexported fields where invariants exist, value receivers, no setters), validated in `New...` constructors that return `(T, error)` (a `Must...` form panics with that error), pure and synchronous. A closed variant set is an interface with an unexported marker method plus one struct per variant, its doc comment naming every variant. A constructor error is `errors.Is` `srderr.ErrInvalidInput` and, where useful, `errors.As` a typed `*InvalidError{Field, Value}`.

- **`SpellEffectModel`** (closed): `TextOnlyEffects` (empty struct) or
  `ModelledEffects{Coverage, Targeting, Parts}` (built by `NewModelledEffects`). `Spell.effects` (default
  `TextOnlyEffects`) is the only change to `Spell` (Q5).
- **`EffectCoverage`**: typed enum `CoverageComplete`, `CoveragePartial`.
- **Targeting.** `SpellTargeting` (closed): `TargetsSelf`;
  `TargetsChosen{Subject, Count, Restriction, SelfPolicy}`; `TargetsInArea(shape, subject,
  restriction)`. `TargetSubject`: `SubjectCreature`, `SubjectObject`, `SubjectCreatureOrObject`.
  `TargetCount` (closed): `ExactlyTargets(n, scaling)`, `UpToTargets(n,
  scaling)`, `TargetsByInstance`. `CreatureTypeRestriction` (closed):
  `AnyCreatureType`, `OnlyCreatureTypes` (a set of `CreatureType`). `SelfTargetPolicy`: `MayIncludeCaster`, `ExcludesCaster`. `AreaShape`
  (closed): `ConeArea`, `CubeArea`, `CylinderArea`, `EmanationArea`, `LineArea`,
  `SphereArea`, each with `IncludesOrigin`. These are the only definitions of the six shapes in the package (magical-effects.md reuses them and defines no second set).
- **Parts.** `SpellEffectPart{Resolution, Instances}`. `PartResolution` (closed):
  `AttackRollResolution{Kind, OnHit, OnMiss}`, `SavingThrowResolution(ability,
  onFailure, onSuccess)`, `AutomaticResolution{Outcomes}`. `SpellAttackKind`:
  `melee`, `ranged`. `SuccessEffect` (closed): `NothingOnSuccess`,
  `HalfDamageOnSuccess`. `PartInstances{Base, Scaling}`.
- **Outcomes.** `SpellOutcome` (closed): `SpellDealsDamage(amount, type,
  addsModifier)`, `SpellRestoresHitPoints{Amount, AddsModifier}`,
  `SpellGrantsTemporaryHitPoints{Amount, AddsModifier}`,
  `SpellAppliesCondition{Kind, Duration}`. `ModifierAddition`: `none`,
  `SpellcastingAbilityModifier`. `DamageTypeSelection` (closed):
  `FixedDamageType(type)`, `ChosenDamageType` (a set of `DamageType`).
  `ConditionDuration` (closed): `ForSpellDuration`,
  `TimedConditionDuration{Amount, Unit DurationUnit}`, `UntilNextTurnBoundary(owner,
  boundary)`; `TurnOwner` (`OwnerCaster`, `OwnerTarget`; not the `TurnHolder` of magical-effects.md); `TurnBoundary` (`BoundaryStart`, `BoundaryEnd`).
- **Amounts and scaling.** `ScalableAmount{Base, Scaling}`; `SpellAmount`
  (closed): `RolledSpellAmount{dice.Expression}`, `FixedSpellAmount{int}`.
  `AmountScaling` (closed): `NoAmountScaling`, `PerSlotLevelAmount(aboveLevel,
  AmountStep)`, `ByCharacterLevelAmount(CharacterLevelTable<SpellAmount>)`.
  `CountScaling` (closed): `NoCountScaling`, `PerSlotLevelCount(aboveLevel,
  extra)`, `ByCharacterLevelCount(CharacterLevelTable<int>)`. `AmountStep(extraDice,
  extraFlat)`. `AmountLevelTable` and `CountLevelTable` (entries of level and value; both are thin wrappers over one unexported generic helper, so no exported type is generic, `doc/rules/go.md` §6).
- **Facts.** `SpellEffectFacts`, `EffectFact` (closed: `FactKnown{Value}`,
  `FactKnownDamageTypes{Types}`, `FactUnknown`; no generic variant carries both a
  boolean and a set, so the set has its own variant).
- **Zone facts** (FR-048): `ZoneFacts`, `ZoneRuleFact`, `ZoneTrigger`,
  `ZoneTargetScope`, `ZoneMobility` (`StationaryZone`, `MovesWithOrigin`,
  `MovedByCaster`, `DriftsAtTurnStart`), `ZoneActionKind`, `DriftDirectionRule`,
  `ZoneRegionID`, `DesignationPolicy`, `ZoneMovementFacts`; optional on `ModelledEffects`.
- **Reused, not defined here:** `DurationUnit` (content-schema-and-decoder.md FR-149); `DamageType`, `CreatureType`
  (content-schema-and-decoder.md; shared-types-and-ownership.md FR-005);
  `ConditionKind` (content vocabulary, content-schema-and-decoder.md FR-110;
  settled by shared-types-and-ownership.md FR-046); `Ability`,
  `AbilityModifier` (ability-scores.md); `dice.Expression` (dice-and-rolls.md).
- **Invariants:** FR-006 and FR-045 in `ModelledEffects`; FR-009 in `SpellEffectModel.CheckSpellLevel`, called by `Spell`;
  FR-043 in `SavingThrowResolution`; the other constructor rules in the named
  type (FR-012, FR-013, FR-015, FR-016, FR-019, FR-025 to FR-027, FR-029, FR-041,
  FR-046). Bundle checks (FR-033, FR-034, FR-036) are verification, not
  constructor rules, so a host's spell is valid without a page.
- **Failures:** constructor violations are returned errors (never panics outside `Must...`); untrusted JSON goes
  through the decoder and returns a problem list (FR-031); there is no `*srderr.PortError` in the model.

## 6. Interface

No use case, no port, no new test double and no contract suite: the model is
static data and pure queries (synchronous, no I/O). The decoder andthe `content.Source` port are those of content-schema-and-decoder.md and
content-source-port.md; the caller resolves the `Spell` and hands it to
`spell-effects.md`.

- **On the model (illustrative; the requirements fix behaviour, not spelling):**
  `Spell.effects`; `SpellEffectModel.Facts()` (FR-022); `AreaShape.IncludesOrigin()`
  (FR-014); `ScalableAmount` and `CountScaling` scaled-value queries taking `n`
  and the character level as plain integers (FR-041); `AmountLevelTable.ValueAt(level)` and `CountLevelTable.ValueAt(level)`
  (FR-018).
- **JSON.** The spell object gains the optional `effects` field (FR-030). The
  variant-selection and key conventions are those of content-schema-and-decoder.md;
  the shape of the effects object is section 6.1; every problem carries a path and a kind (FR-031).
- **Exports.** Package `content` exports the
  section 5.2 types, including `SelfTargetPolicy`, each with a godoc comment starting with its name.
  Every exported name is unique across the module's packages, none is a generic variant
  name (`Applied`, `Rejected`, `Succeeded`, `ContentMissing`, `Found`,
  `NotFound`), and none refers to a consuming project
  (shared-types-and-ownership.md FR-017, FR-021).

### 6.1 JSON shape of the effects object (normative; FR-030 to FR-032, FR-049)

The `effects` key of the spell object is optional by absence and, present, holds one
`ModelledEffects` object. `TextOnlyEffects` is written by omitting the key (an
`effects` of `null` or `{}` is `wrongType` or `missingField`, never text-only). Each
variant is an object with a string `kind`, the lower camel case of the variant name; a
missing, non-string or unknown `kind` is the only problem of that object. Keys are read
in the order listed (the order problems are reported). `?` is optional by absence. An
integer is read as everywhere (schema FR-021) and is at most 2^31 - 1 (`outOfRange`
above), so a decoded value is the same on every platform integer width. Wire spellings of
a closed set are the `String()` of its members (`ParseX`, exact and case-sensitive);
`DamageType`, `CreatureType`, `ConditionKind` and `DurationUnit` keep their schema
spelling.

| Object | Keys | Rules |
|---|---|---|
| effects | `coverage`, `targeting`, `parts`, `zone?` | `coverage`: `complete`, `partial`. `parts`: array of at least 1 part. |
| targeting | `kind`: `targetsSelf`, `targetsChosen`, `targetsInArea` | `targetsChosen`: `subject`, `count`, `restriction?`, `selfPolicy?`. `targetsInArea`: `shape`, `subject`, `restriction?`. `subject`: `creature`, `object`, `creatureOrObject`. `selfPolicy` absent is `mayIncludeCaster` (FR-040). A restriction on an `object` subject is `constraintViolation` at `restriction` (FR-015). |
| count | `kind`: `exactlyTargets`, `upToTargets`, `targetsByInstance` | The first two: `n` (at least 1), `scaling?` (a count scaling). |
| restriction | `kind`: `anyCreatureType`, `onlyCreatureTypes` | `onlyCreatureTypes`: `types`, a set of at least 1 `CreatureType`. |
| shape | `kind`: `coneArea` (`lengthFeet`), `cubeArea` (`sideFeet`), `cylinderArea` (`radiusFeet`, `heightFeet`), `emanationArea` (`distanceFeet`), `lineArea` (`lengthFeet`, `widthFeet`), `sphereArea` (`radiusFeet`) | Each dimension an integer of at least 1. |
| count scaling | `kind`: `noCountScaling`, `perSlotLevelCount`, `byCharacterLevelCount` | `perSlotLevelCount`: `aboveLevel`, `extra` (at least 1). `byCharacterLevelCount`: `entries`, each `{level, count}`. |
| amount scaling | `kind`: `noAmountScaling`, `perSlotLevelAmount`, `byCharacterLevelAmount` | `perSlotLevelAmount`: `aboveLevel`, `extraDice?`, `extraFlat?` (each at least 0, absent is 0; both 0 is `outOfRange` at the scaling). `byCharacterLevelAmount`: `entries`, each `{level, value}` with `value` an amount base. Entries: at least 1, `level` 1 to 20 (`outOfRange` otherwise), first level 1 and strictly ascending (`constraintViolation` at `entries`, FR-019). |
| part | `resolution`, `instances?` | `instances`: `{base, scaling?}`, `base` at least 1 with a count scaling; absent is one instance with no scaling. |
| resolution | `kind`: `attackRollResolution` (`attack`, `onHit`, `onMiss?`), `savingThrowResolution` (`save`, `onFailure`, `onSuccess`), `automaticResolution` (`outcomes`) | `attack`: `melee`, `ranged`. `save`: an ability name as the class rows write it (`strength` ... `charisma`). `onSuccess`: `nothingOnSuccess`, `halfDamageOnSuccess` (with no damage outcome: `constraintViolation` at `onSuccess`, FR-043). Outcome lists have at least 1 element, except `onMiss`, which may be empty. |
| outcome | `kind`: `spellDealsDamage` (`amount`, `damageType`, `addsModifier?`), `spellRestoresHitPoints`, `spellGrantsTemporaryHitPoints` (`amount`, `addsModifier?`), `spellAppliesCondition` (`condition`, `duration`) | `addsModifier`: `none`, `spellcastingAbilityModifier`; absent is `none`. `condition`: a `ConditionKind` spelling. More than one `chosenDamageType` in the model is `constraintViolation` at `parts` (FR-045). |
| amount | `base`, `scaling?` | `base`: `{kind: rolledSpellAmount, roll}` (dice notation exactly as `dice.Expression.Notation` prints it, a modifier allowed: `1d4+1`) or `{kind: fixedSpellAmount, value}` (at least 1). A step that adds dice to a fixed base is `constraintViolation` at `scaling` (FR-016). |
| damageType | `kind`: `fixedDamageType` (`type`), `chosenDamageType` (`types`) | `types`: a set of at least 2 `DamageType`s (`outOfRange` below). |
| duration (of a condition) | `kind`: `forSpellDuration`, `timedConditionDuration` (`amount` at least 1, `unit`), `untilNextTurnBoundary` (`owner`, `boundary`) | `owner`: `caster`, `target`. `boundary`: `start`, `end`. |
| zone | `rules`, `mobility?`, `oncePerTurn?`, `designation?`, `damageBudget?`, `stopsMoverOnContact?`, `movement?` | `rules`: at least 1; a repeat of an identical rule is `duplicate` at the later index. `designation`: `none`, `atCastingOnly`. Absent booleans are false. |
| zone rule | `trigger`, `partIndex`, `scope`, `region?`, `unitFeet?`, `casterChoice?` | `trigger`, `scope`: the `ZoneTrigger` and `ZoneTargetScope` spellings. `partIndex` outside the parts of the model is `constraintViolation` (checked when every part decoded). `region` follows the name rule (schema FR-027); absent is `body`. |
| zone mobility | `kind`: `stationaryZone`, `movesWithOrigin`, `movedByCaster` (`maxFeet`, `action`, `laterTurnsOnly?`), `driftsAtTurnStart` (`feet`, `direction`) | Numbers at least 1. `action`: `bonusAction`, `magicAction`, `withCasterMovement`. `direction`: `awayFromCaster`, `awayFromCasterChosen`. |
| zone movement | `difficultTerrain?`, `speedHalved?`, `extraFeetPerFoot?` | `extraFeetPerFoot` at least 0; absent is 0. |

**Key presence of the zero-for-absent fields (FR-049).** `ZoneFacts.DamageBudget` and
`ZoneRuleFact.UnitFeet` hold 0 for absent, so the constructors cannot tell an absent key
from a zero. The decoder decides by the key: a `damageBudget` or `unitFeet` key that is
present holds an integer of at least 1 (0 and below are `outOfRange`), and an absent key is
0. `unitFeet` is then required exactly for `travelsWithin` (`constraintViolation` at the
`unitFeet` key when missing there or present on another trigger). The agreement test of
FR-032 therefore lists a present zero as the one documented divergence from the
constructors, which read a 0 field as absent. `extraFeetPerFoot` has no such ambiguity (0
is the stated "none") and accepts a present 0.

**Where a rule across fields is reported.** A rule the decoder leaves to the constructor
or to `SpellEffectModel.CheckSpellLevel` is reported with the kind of the constructor's
`BlockProblem` (`outOfRange` for `BlockEmpty` and `BlockOutOfRange`, `duplicate`,
`constraintViolation` for `BlockConstraint` and `BlockRaggedRow`, `missingField`,
`invalidValue` otherwise) at the key or object the paragraph above names, else at the
object that holds the values. A scaling that contradicts the spell's level (FR-009) is
`constraintViolation` at the `effects` key of the spell, reported only when the effects
and the level decoded cleanly.

## 7. Observability

Silent by design: the model is pure data and queries, so it emits nothing
(`doc/rules/observability.md` §3). This spec owns no signal and changes none. Answers to the seven evaluation questions (`doc/rules/observability.md` §2):

1. **What can go wrong?** A model that disagrees with its verbatim text; effect
   JSON that fails to decode; a host reading a `partial` or text-only spell as
   complete.
2. **How would an operator know?** FR-033 and FR-034 catch disagreement offline;
   decode failures are returned results counted by the decoder's owners
   (content-schema-and-decoder.md, content-source-port.md).
3. **How would you diagnose it?** The path and kind of each problem (FR-031) and the bundle
   check findings, which name the spell and the token.
4. **What is counted or timed?** Nothing here; the coverage mix is counted by
   `srd5e.combat.spell_effects.resolutions` (`spell-effects.md` section 7).
5. **Existing instrumentation.** Unchanged (decoder signals). 6. **Consumer
   monitoring.** Content decode failures and the `text_only` plus `partial` share.
7. **What is noise?** None.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-003, FR-006, FR-007, FR-008, FR-009 | unit | `content/spell_effects*_test.go` (package `content_test`): `Spell.effects` default, text unchanged, equality including `effects`, empty parts, defensive copies of slices, cross-field rules (cantrip with per-slot, level 1+ with a table, `AboveLevel` 2 below level and 10). |
| FR-002 | architecture | A `go/parser` test in `internal/archtest` (architecture-guardrails.md FR-063) scans every struct field, method result and constructor parameter in the model files and fails on a function type, `any` or `interface{}` (other than unexported marker interfaces); a name scan of the exported API finds no spell-named type and no `SpellBehaviour`-style export. |
| FR-004, FR-005 | unit and review | Coverage values in the pilot fixtures. A fixture check for FR-005 (unit): a table of the spells of table E rows E13 to E26 and P2, P9, P10, P12, P13, P14, P15, P16 (P1, P3 to P8 and P11 are `complete` and carry no residue), each with its unmodelled outcome and the expected `partial` (or text-only) result, and an assertion that no model encodes the unmodelled outcome with a nearest variant (for example Aid is not Temporary Hit Points). A reviewer checks every `complete` pilot spell against its page (FR-004 has no mechanical test beyond the fixture table). |
| FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, FR-040 | unit | `content/spell_effects*_test.go` (package `content_test`): table A rows A1 to A10, table T rows T1 to T3, the count and subject cases, and the two `SelfTargetPolicy` values on `TargetsChosen` only. |
| FR-016, FR-017, FR-018, FR-019, FR-020, FR-021, FR-041 | unit | Tables L (L1 to L19, L21, L22, L24) and C (C1 to C17, C19, C20), one test per row; character level versus class level (C14); independent scaling of two outcomes (L13). |
| FR-022, FR-023, FR-024 | unit | Table F rows F1 to F12. |
| FR-025, FR-026, FR-027, FR-028, FR-029 | unit | Constructor boundaries: fixed amount 0 and 1, empty outcome lists, a one-element chosen set, all fifteen condition kinds accepted, instance base 0 and 1. |
| FR-042, FR-043, FR-044, FR-045, FR-046, FR-047 | unit | The three `PartResolution` variants and their fields; `HalfDamageOnSuccess` with and without a damage outcome; closed `DamageTypeSelection`; two `ChosenDamageType` outcomes in one model; condition duration variants and boundaries (amount 0, 1); the four `SpellOutcome` variants and `ModifierAddition`. |
| FR-030, FR-031, FR-032 | contract (decoder level, testing.md §3) | `content/spell_effects_decode_test.go`: decode a spell with and without `effects`; every problem kind with paths, including the rule-to-kind table of FR-031 (`OutOfRange`, `ConstraintViolation`, `duplicate`, `InvalidValue`); the agreement test over the constructor rules. Fuzz: `FuzzDecodeSpellEffects` feeds arbitrary JSON to the decoder (it must return problems, never panic). Golden inputs live under `testdata/`. |
| FR-033, FR-034, FR-035, FR-036, FR-037 | unit | Verification tooling (outside the importable packages, never imported) is tested as unit tests. Tests on synthetic spells for the model-text check, the cantrip level check and the scaling-text check; the regeneration test keeps `effects`; the count report; the 16 pilot fixtures (table P) under `testdata/`; a number-check test (FR-033): a mistranscribed Heal amount, target count and dart count each produce a finding. |
| FR-048, FR-049, FR-050 | unit and contract (decoder) | Constructor boundaries (`UnitFeet` 0, 1 and on a non-`TravelsWithin` rule; `MaxFeet`, `Feet`, `DamageBudget` 0 and 1; `ExtraFeetPerFoot` -1, 0; empty rules); decoder kinds and the constructor agreement (FR-032); bundle verification tests on synthetic spells (a mistranscribed budget, a `PartIndex` past the parts); the table Z fixtures. |
| FR-039 | architecture | Import rules (the model imports only kernel packages and `content`; nothing from `combat` or `character`), export scan for duplicate and generic names (including `TurnOwner` against `TurnHolder` and the six `AreaShape` names, which exist once). |

- **Levels:** unit, contract (decoder), architecture. Doubles: none new; decoder tests reuse `contenttest.FakeSource` and the decoder contract of content-schema-and-decoder.md. No application level or
  scenario (no use case); the cast-and-resolve scenario is in `spell-effects.md`.
- **Mutation testing (`doc/rules/testing.md` §8):** the model files (the effect types) at a score of at least 80 using `gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors (most worth: the step count in FR-016, table selection in FR-018, level bounds in FR-019).
- **Every FR id appears above or below.** Deliberately not covered: FR-038 - the human review record is a process requirement, checked in review.

**Implementation status.**

- Done (Phase 2, step 5f, package `content`, files `spell_effects.go`, `spell_effects_enums.go`, `spell_effects_targeting.go`, `spell_effects_area.go`, `spell_effects_count.go`, `spell_effects_amount.go`, `spell_effects_table.go`, `spell_effects_part.go`, `spell_effects_outcome.go`, `spell_effects_facts.go`, `spell_effects_zone.go`): the closed `SpellEffectModel` (`TextOnlyEffects`, `ModelledEffects` with `NewModelledEffects`, `WithZoneFacts`, `Equal`, `Facts`, `CheckSpellLevel`), `EffectCoverage`, `SpellTargeting` (`TargetsSelf`, `TargetsChosen`, `TargetsInArea`), `TargetSubject`, `SelfTargetPolicy`, `TargetCount` (`ExactlyTargets`, `UpToTargets`, `TargetsByInstance`), `CreatureTypeRestriction` (`AnyCreatureType`, `OnlyCreatureTypes`), `AreaShape` and its six shapes, `SpellEffectPart`, `PartResolution` (`AttackRollResolution`, `SavingThrowResolution`, `AutomaticResolution`), `SpellAttackKind`, `SuccessEffect`, `PartInstances`, `SpellOutcome` (the four variants), `ModifierAddition`, `DamageTypeSelection` (`FixedDamageType`, `ChosenDamageType`), `ConditionDuration` (`ForSpellDuration`, `TimedConditionDuration`, `UntilNextTurnBoundary`), `TurnOwner`, `TurnBoundary`, `SpellAmount` (`RolledSpellAmount`, `FixedSpellAmount`), `ScalableAmount`, `AmountStep`, `AmountScaling` and `CountScaling` with their variants, `AmountLevelTable`, `CountLevelTable`, `SpellEffectFacts`, `EffectFact`, and the zone facts (`ZoneFacts`, `ZoneRuleFact`, `ZoneTrigger`, `ZoneTargetScope`, `ZoneMobility` and its four variants, `ZoneActionKind`, `DriftDirectionRule`, `ZoneRegionID`, `DesignationPolicy`, `ZoneMovementFacts`). Tested on synthetic and pilot-shaped models: FR-001 (the model; `Spell.effects` itself is step 5g), FR-002 and FR-039 (an architecture test in `internal/archtest/spell_effects_model_test.go`, with a violating overlay for the rule itself), FR-004 to FR-029, FR-040 to FR-048 and the constructor half of FR-049. Every constructor error is a `*BlockError` (reasons `BlockEmpty`, `BlockOutOfRange`, `BlockConstraint`, `BlockDuplicate`, `BlockInvalidMember`, `BlockMissing`, `BlockInvalidName`), which maps one to one to the decoder kinds of FR-031.
- Done (Phase 2, step 5g, package `content`, files `spell_effects_parse.go`, `decode_spell_effects.go`, `spell.go`): FR-030 to FR-032, the decoder key `effects` of the spell object with the wire shape of section 6.1 (every variant of targeting, restriction, area shape, count, scaling, resolution, outcome, amount, damage type, condition duration and zone read; the problem kinds of FR-031 at the paths the table in 6.1 names), the `ParseX` of the thirteen small closed sets (`ParseEffectCoverage` ... `ParseDesignationPolicy`), `FuzzDecodeSpellEffects`, the table-driven agreement test `TestAgreementSpellEffects` (FR-032), the decoder half of FR-049 (`outOfRange` for an empty rule list and for a unit, `MaxFeet`, `Feet` or budget below 1 and an `ExtraFeetPerFoot` below 0, `constraintViolation` for `UnitFeet` missing on `TravelsWithin` or present elsewhere and for a `PartIndex` outside the parts, `duplicate` for a repeated rule), `Spell.effects` with `Spell.Equal` over it (FR-001, FR-003) and the call of `CheckSpellLevel` by `NewSpell` (FR-009). Choices the spec left open: the shape of section 6.1 (discriminators are the lower camel case of the full variant names; `selfPolicy` absent is `mayIncludeCaster`; `instances` absent is one instance; `addsModifier` absent is `none`; `extraDice` and `extraFlat` absent are 0); key presence of the zero-for-absent fields (section 6.1, FR-049: a present `damageBudget` or `unitFeet` of 0 is `outOfRange`); integers are at most 2^31 - 1 in the decoder; `PartIndex` is range-checked only when every part decoded; the level check of FR-009 is reported at the `effects` key and only when the effects and the level decoded.
- Choices the spec left open: the package never imports `Spell` (the 5g constructor calls `CheckSpellLevel`). Scaled values are queried with `ScalableAmount.At(slotLevelsAbove, characterLevel)`, `ExactlyTargets.At`, `UpToTargets.At`, `PartInstances.At` and `AmountLevelTable.ValueAt` / `CountLevelTable.ValueAt`; each input is validated only by the scaling that reads it, and a result past the dice limits or `int` is `BlockOutOfRange`, never wrapped around. `AboveLevel` is stored as given (any integer); only `CheckSpellLevel` judges it, so an `AboveLevel` of 10 can be built and is then rejected there (FR-009). The two level tables are arrays behind one unexported generic helper, so every amount, count, scaling and outcome is comparable with `==`; the types that hold slices (`AttackRollResolution`, `SavingThrowResolution`, `AutomaticResolution`, `SpellEffectPart`, `ModelledEffects`, `ZoneFacts`) have `Equal`, with the rule set of a zone compared as a set. The two sets (`OnlyCreatureTypes`, `ChosenDamageType`) are bit sets, so they are comparable and order-insensitive. `SuccessEffect` and `ModifierAddition` are typed enums, not variant structs. A nil `CreatureTypeRestriction`, `CountScaling`, `AmountScaling` and `ZoneMobility` read as `AnyCreatureType`, `NoCountScaling`, `NoAmountScaling` and `StationaryZone`, and a zero `DesignationPolicy` reads as `NoDesignation` and an empty `ZoneRegionID` as `body`. `EffectFact` has a third variant `FactKnownDamageTypes` for the `DamageTypes` set (the spec's single `FactKnown{Value}` cannot carry a set without `any`); the zero `SpellEffectFacts` answers `FactUnknown` throughout. `ZoneFacts.DamageBudget` and `ZoneRuleFact.UnitFeet` are 0 when absent, so the decoder reads a missing key as 0; `UnitFeet` missing on `TravelsWithin` or present on another trigger is `BlockConstraint`, a negative one `BlockOutOfRange`. The constructors with several optional members take a keyed fields struct (`ZoneRuleFields`, `ZoneFactsFields`). Wire spellings are the lower camel case names returned by `String()`.
- Mutation status (`just mutate "./content/<file>.go"`, one file at a time, capped at a quarter of the processors): `spell_effects.go` 93.3 (re-run 2026-10-20, 6 survivors), `spell_effects_table.go` 92.5, `spell_effects_amount.go` 89.9, `spell_effects_count.go` 86.0, `spell_effects_targeting.go` 93.4 (re-run 2026-10-20, 7 survivors), `spell_effects_area.go` 100.0, `spell_effects_part.go` 93.7, `spell_effects_outcome.go` 96.6, `spell_effects_facts.go` 95.2, `spell_effects_zone.go` 91.7 (a partial run: the 10 minute limit of the foreground command ended it with the last of its 151 mutants unreported). `spell_effects_enums.go` holds only name lists and `Valid` / `String` methods through the shared `enumValid` and `enumString`.
- Manual-mutation pass (2026-10-20) for the four files with no same-named test (`spell_effects_enums.go`, `spell_effects_parse.go`, `spell_effects_facts.go`, `spell_effects_table.go`): 44 distinct mutations (name-list swaps, off-by-ones, flipped comparisons, dropped branches), each applied alone against `go test ./content`. 42 caught; 2 survived in `spell_effects_facts.go` (a chosen damage-type set overwriting a fixed type, and `UpToTargets` per-slot scaling ignored) and `TestDerivedFactsUnionAndUpToTargets` now kills both; 1 further mutation did not compile and was dropped. No equivalent survivors.
- Manual-mutation pass (2026-10-20) for `spell_effects_area.go` and `spell_effects_count.go` (no same-named test): 21 + 23 distinct mutations against `go test ./content`. Survivors killed by new tests (`TestZeroAreaShapesAreRejected`, `TestUnscaledCountOfOneAndZeroUpToTargets`): zeroed shape dimensions in `checkAreaShape`, `base <= 1` in `scaledCount`, `UpToTargets{}` in `checkTargetCount`. Equivalent survivors: `v.height`/`v.width` in `checkAreaShape` and `v.scaling` in `checkTargetCount` (a value with one valid and one zero field cannot be built outside the package); swapped `slotLevelsAbove`/`extra` in `mulAdd` (the overflow test `n > max/extra` is symmetric). Two mutations did not compile and were dropped.
- Mutation pass 2026-10-20 (spell effects 3): `spell_effects_outcome.go` 98.3 (gomu, 1 survivor, not identified: the per-file run prints only the score), `spell_effects_zone.go` 91.0 (gomu, 13 survivors, none yet analysed; above 80). `spell_effects_part.go` and `spell_effects_amount.go` have no same-named test, so 33 + 27 manual mutations ran against `go test ./content`. Survivors killed by `spell_effects_part_test.go` (`TestResolutionsOwnTheirOutcomes`, `TestResolutionEquality`, `TestSpellEffectPartEquality`, `TestScalingArithmeticLimits`): dropped outcome-list copies in constructors and getters, the `onHit`/`onMiss`/`onFailure` comparisons in `Equal`, and the `okDice || okMod`, `>=` and `max(base, 0)` mutations in `mulAdd`/`grow`. Equivalent survivors in `spell_effects_part.go`: `return true` for `o.resolution == nil` in `SpellEffectPart.Equal` (a part with a resolution always has non-zero instances, so a zero part differs earlier), and dropping the clones in `cloneResolution` and `Resolution()` (every resolution getter copies and the fields are private, so no caller can reach the shared slices). Three mutations of `NewScalableAmount`/`NewPerSlotLevelAmount` and one of `NewSpellEffectPart` did not compile.
- Mutation pass 2026-10-20 (spell effects 4): `decode_spell_effects.go` has no same-named test, so 23 manual mutations ran against `go test ./content` (blockKind case, selfPolicy default, count kind and absence, scaling `aboveLevel` lower bounds, addsModifier default, roll notation round trip, partIndex bounds, unitFeet rule, duplicate rule, damageBudget minimum, part clean flag, table level maximum, optional onMiss, damage type validity, set minimums, step and movement minimums, laterTurnsOnly). Survivors killed by new rows `negative scaling level` and `negative amount scaling level` in `TestDecodeSpellEffectsProblems` and by `TestDecodeSpellAttackWithoutOnMiss`. Equivalent survivors: `clean || err == nil` in `decodeRestriction` and `hasZone || zoneOK` in `decodeSpellEffects` (the caller drops the value whenever a problem was recorded, and `zoneOK` implies `hasZone`).

Not yet implemented: FR-033 to FR-038 (the bundle checks and the pilot annotations, Phase 5), the bundle verification half of FR-049 and FR-050 (the numbers against the spell text, and the table Z fixtures), and the mutation score of the 5g files.


## 9. Open questions and decisions

| Question | Recommendation | Status |
|---|---|---|
| Q1. Data-described effects or strategy interfaces (architecture.md §3)? | Data now (option B, section 5.1). Revisit only when a consumer shows a spell no variant can express that the host must compute, or the same unmodelled pattern recurs in several class specs (table E "text" rows); then add one narrow domain interface (the standard rule as its default) or a new data variant. A mode choice (E21) would first be a data variant. | decided: section 5.1 option B; the revisit trigger stands |
| Q2. Who owns `ConditionKind`? | `ConditionKind`, `DamageType`, `CreatureType` are defined in content-schema-and-decoder.md | decided: shared-types-and-ownership.md FR-005, FR-046 |
| Q3. Who writes the structured data, and by what method? | Hand-annotated and reviewed, in a layer beside the extracted text, never inferred by the PDF tool: reading "half as much damage" or "up to three creatures" is interpretation. The checks that make this safe are mechanical: every modelled token must appear in the verbatim text (FR-033) and scaling must match the printed numbers (FR-034). The alternative, heuristic extraction from the prose, was rejected as unreviewable. | decided: hand-annotated, mechanically checked (FR-033, FR-034, FR-038); basis architecture.md §3 |
| Q4. Who reads spell facts? | Class-spec private spell facts read `SpellEffectFacts` (FR-022 to FR-024); `CanTargetMoreWithHigherSlot` counts targets only (F10); class-sorcerer.md to confirm | decided: shared-types-and-ownership.md FR-048; consumers to change (RA-1) |
| Q5. `Spell.effects` is a new field on a type owned by content-spells.md, and the JSON field needs a revision-table row in content-schema-and-decoder.md. | Both owners add the field and the row when this spec is accepted (content-spells.md section 9 Q5 already points here); the default is `TextOnlyEffects`, so existing spell data and the six thin-slice spells stay valid. content-spells.md FR-087 (equality over all fields) and its logical record change with it. | decided: this spec owns the field; the owners apply it via RA-3 and RA-4 |
| Q6. Is "that you can see" modelled? | `TargetsInArea` with an unmodelled "that you can see" is `complete` (FR-004); a consumer wanting enforcement adds a new restriction variant | decided |
| Q7. Where did the shape rules go? | Which ids moved from `spell-effects.md`: its Q3 assignment, made binding by shared-types-and-ownership.md FR-059; four model-shape rules (old FR-021, FR-027, FR-034, FR-039) moved as FR-042, FR-043, FR-045, FR-046 (shape parts FR-042, FR-044, FR-028, FR-040) because the decoder rule-to-kind table (FR-031) needs them here | decided |

Change requests to other specs, to be made in the same change that accepts this one:

| Id | Spec | Change |
|---|---|---|
| RA-1 | class-sorcerer.md, class-wizard.md, class-warlock.md | Read `SpellEffectFacts` (Q4) instead of private spell fact tables. |
| RA-2 | magical-effects.md, rules-glossary-coverage.md | Their citations of the six shapes and `IncludesOrigin` (the former FR-013, FR-014 and FR-016 (withdrawn) of `spell-effects.md`) point to `spell-effect-data.md` FR-013, FR-014 and FR-015; the owner of the shape types is now this spec. |
| RA-3 | content-schema-and-decoder.md | One revision-table row (`R2-spell-effects`) for the optional spell field `effects` (FR-030) and the problem-kind mapping of FR-031. |
| RA-4 | content-spells.md | Q5 and FR-107: add `Spell.effects`, equality over it (FR-003), and the optional last key `effects` to the declared key list. |

## 10. Changelog impact

Under `Unreleased`:

- **Added:** the spell effect model in package `content` (`SpellEffectModel`
  and its variants, `AreaShape`, scaling types, `SpellEffectFacts`, optional `ZoneFacts`), `Spell.effects`
  (default `TextOnlyEffects`), and the pilot spell annotations as bundled game
  content (`NOTICE.md` covers the data path and MUST state that the annotations
  restructure SRD text, per content-licensing-and-notice.md). The resolver and its
  signals are in `spell-effects.md`.
- **Not BREAKING while unreleased** (a field with a default breaks no consumer). After the first release, adding a variant to `SpellEffectModel`,
  `PartResolution`, `SpellOutcome`, `SpellTargeting`, `AreaShape`,
  `ConditionDuration`, `AmountScaling`, `CountScaling`, `ZoneTrigger` or
  `ZoneMobility` is **BREAKING** (consumers switch over them), as is changing a
  modelled spell's data other than to fix a transcription error.
