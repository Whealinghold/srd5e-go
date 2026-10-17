# Magical effects and Glossary phenomena

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `conditions.md`, `attacks.md`, `hit-points-and-damage.md`, `content-schema-and-decoder.md`, `spell-effect-data.md` |
| Referenced, not imported | `casting-and-concentration.md` (owner of Concentration, FR-075 to FR-079, FR-099; the host passes its results and events); `magic-item-rules.md` (owner of Attunement, FR-054, FR-117); `hiding-searching-and-senses.md` (takes `IllusionTraits.DeceivesSight` as a plain boolean, its FR-095); `toolbox-curses-and-contagions.md` (downstream, `shared-types-and-ownership.md` FR-045: imports `CurseRemovalMeans`, `PossessionPrevention`, `PossessionEnding` from here); `spell-effects.md` (reads `AreaMembership.Assess`, its FR-015); `telepathy.md`, `returning-to-life.md`, `readying-a-spell.md` (split out of this spec); `rules-glossary-coverage.md` (names this spec as owner of GL-008b, GL-034b, GL-040, GL-050b, GL-084, GL-099, GL-112a, GL-117b, GL-125, GL-127, GL-144, GL-145b) |
| Consumed by | `combat-action-resolution.md` (amendment RA-1), `hiding-searching-and-senses.md` (its FR-095), `toolbox-curses-and-contagions.md` |

Rank 26 (`shared-types-and-ownership.md` 4.11): right of `conditions`, `attacks`,
`hit-points-and-damage` and `initiative-and-actions` (25); left of `encounter-aggregate`
(28) and `combat-action-resolution` (29). "Depends on" lists only the specs whose types
sections 5 and 6 import; the rest are prose. `hiding-searching-and-senses` (23) takes
`DeceivesSight` as a plain boolean. Concentration, readied-action rules, Possession
state, Attunement and Truesight belong to their owners (decisions FR-021, FR-045, FR-048).

**Split.** Telepathy, returning to life and readying a spell moved to
`telepathy.md` (rank 23), `returning-to-life.md` (27) and `readying-a-spell.md` (26);
each moved id is a Withdrawn row in 4.16 naming the live id of the new owner.
**Supersedes** no numbered requirement of `spellcasting.md`, `content-spells.md` or
`spell-effects.md` (decisions FR-048); FR-109 and FR-034 to FR-041 are withdrawn to
their owners.

**Amendments asked of other specs (none made here).** RA-1:
`combat-action-resolution.md` use cases and telemetry (FR-148, section 7). RA-4:
`spell-effect-data.md` FR-014 wording ("unless its creator decides otherwise is the
host's input and is not modelled") contradicts FR-016 here. RA-5:
`casting-and-concentration.md` Table K, a row for 62 (Q-26).

## 1. Purpose and scope

The Rules Glossary entries (SRD 5.2.1, pp. 176-191) that describe magical
phenomena independent of any single spell: Magical Effect and effect sources; Area
of Effect origin rules and membership from host facts (the six shapes are
`spell-effect-data.md`'s, FR-013, FR-014, decisions FR-059); Curses, Illusions,
Shape-Shifting, Teleportation, Simultaneous Effects and the spells named by
Possession. Pure domain in `combat`: state in, new state and events
out; no use case, port or dice; use cases and telemetry are RA-1 (FR-148, section
7) and none calls another use case (`shared-types-and-ownership.md` FR-044).

**The map boundary (decision, Q-4).** The package models no map. Positions,
distances, lines of sight and shape geometry are the host's; this spec states the
SRD rules about a shape and takes every spatial fact as input.

**Out of scope:**

- The spells and magic items themselves (Remove Curse, Teleport, Polymorph), their
  DCs, damage and durations; only Glossary rules that name a spell are modelled.
- Positions, distances, grids and shape placements (a Cube's origin "anywhere on a
  face", p. 179; a Line's "straight path along its length", p. 184). Whether a line
  is blocked is a host fact.
- What a particular curse, possession, illusion, shape or teleportation does beyond
  the Glossary rule ("the effect defines" it), including replacing statistics when
  shape-shifted (Wild Shape, `class-druid.md`; the four spells,
  `shape-changing-spells.md`); Truesight against Visual Illusions;
  Damage Threshold and objects; curses on objects (Q-8); Opportunity Attacks
  (FR-064); the equipment a teleporter carries (p. 190), which the engine does not
  move.
- Every other Glossary entry not named above, and anything not in SRD 5.2.1.
  Nothing core to this topic was left out (Q-9, Q-10).

## 2. Source

PDF page numbers equal printed page numbers. SRD 5.2.1: "Area of Effect", p. 177;
"Cone", "Cube", p. 179; "Cylinder", p. 180; "Emanation", p. 181; "Line", p. 184;
"Sphere", p. 188; "Cover", p. 179 and p. 15; "Magical Effect", p. 185; "Spell",
p. 188; "Curses", p. 179; "Possession", p. 186; "Illusions", p. 183; "Truesight",
p. 190 ("Visual Illusions" only); "Shape-Shifting", "Simultaneous Effects", p. 187;
"Teleportation", p. 190; "Occupied Space", p. 185; "Unoccupied Space", p. 191;
"Opportunity Attacks", p. 15 and p. 185; "Dead", p. 180 (what Shape-Shifting
carries over). Rule files: `doc/rules/architecture.md` §2, §3, §6;
`doc/rules/testing.md` §4, §5; `doc/rules/observability.md` §2, §3.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Magical Effect | `EffectSource` (closed) with `IsMagical` | "created by a spell, a magic item, or a phenomenon that a rule labels as magical" (p. 185). |
| (ongoing effect) | `OngoingEffect`, `OngoingEffects`, `OngoingEffectID` | Package names. Non-condition ongoing effects; conditions stay in `conditions.md`'s `ConditionSet`. |
| Area of Effect | `AreaShape` and six variants | Owned by `spell-effect-data.md` FR-013; reused. |
| point of origin | `OriginPlacement`, `AreaOriginResolution` | The origin is a host location; only the rules are here. |
| "unless its creator decides otherwise" | `OriginInclusionDecision` (`ShapeDefaultOrigin`, `CreatorIncludesOrigin`), `OverridableAreaShape` | Only for the four shapes whose origin is not included. |
| instantaneous or stationary effect | `EmanationEffectTiming` (`instantaneous`, `stationary`, `ongoing`) | `ongoing` means "neither". |
| Total Cover | `TotalCover` from `attacks.md` (`Cover`) | Reused. |
| Curses | `OngoingEffect` category `curse`; `CurseRemovalMeans` | "Remove Curse", "Greater Restoration", "magic that explicitly ends curses". |
| Possession | `PossessionPrevention`, `PossessionEnding` | Spell classification only; the state is `toolbox-curses-and-contagions.md`'s. |
| Illusions | `IllusionTraits`, `IllusionSenseID`, `IllusionManifestation` | Truesight stays in the senses spec. `IllusionSenseID` is a local open id (host or effect description, no content kind). |
| Shape-Shifting | `ShapeShiftState` (`TrueForm`, `ShiftedForm`), `CarryOverRule` | |
| Teleportation | `TeleportCommand`, `TeleportResult`, `DestinationSpace`, `DestinationSight`, `TouchedCreaturePolicy` | |
| Simultaneous Effects | `PendingEffectID`, `OrderingAuthority`, `TurnHolder` | |

Each operation returns a closed `<Operation>Result` with operation-prefixed
variants (decisions FR-017; section 7). Package names with no SRD source:
`EffectSource`, `OngoingEffect`, `OngoingEffects`, `OngoingEffectID`,
`PendingEffectID`, `TurnHolder`, `OrderingAuthority`, `ShapeShiftState`,
`IllusionSenseID`. Terms use SRD 5.2.1 wording (`doc/rules/architecture.md` §3).

## 4. Requirements

Ids are stable. Case-table rows are one test each, citing the FR in the row.

### 4.1 Magical Effect and effect sources

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `SpellSource(SpellID)` MUST be a magical effect (`IsMagical` true). | SRD 5.2.1, "Magical Effect", p. 185; "Spell" ("a magical effect"), p. 188 | happy |
| FR-002 | `MagicItemSource(ItemID)` MUST be a magical effect. | SRD 5.2.1, "Magical Effect", p. 185 | happy |
| FR-003 | `LabeledMagicalSource(FeatureID)`, an effect "a rule labels as magical", MUST be a magical effect. | SRD 5.2.1, "Magical Effect", p. 185 | happy |
| FR-004 | `NonMagicalSource(FeatureID)` MUST NOT be a magical effect (`IsMagical` false). | SRD 5.2.1, "Magical Effect", p. 185 | unhappy (boundary) |
| FR-005 | `EffectSource` values MUST be equal by value, and a `SpellSource` and a `MagicItemSource` over the same underlying text MUST NOT be equal (different variants). | `doc/rules/architecture.md` §3 (value object) | unhappy (rule interaction) |

### 4.2 Ongoing effects

An `OngoingEffect` is one non-condition effect on one creature (spell, curse,
magical contagion or other); the host holds the creature's `OngoingEffects`.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | An `OngoingEffect` MUST carry an `OngoingEffectID`, a category (`spell`, `curse`, `MagicalContagion`, `MagicalEffect`) and an `EffectSource`, and MUST be equal by value. | SRD 5.2.1, "Dead" (conditions, magical contagions, curses), p. 180; "Shape-Shifting" (conditions, spells, curses), p. 187 | happy |
| FR-007 | An `OngoingEffect` of category `spell` with a source that is not a `SpellSource` MUST return an error matching `srderr.ErrInvalidInput`. | `doc/rules/architecture.md` §3 (invariants in the type) | unhappy (invalid input) |
| FR-008 | An `OngoingEffect` of category `MagicalContagion` or `MagicalEffect` with a source whose `IsMagical` is false MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Dead" ("magical contagions"), p. 180; "Magical Effect", p. 185 | unhappy (invalid input) |
| FR-009 | `OngoingEffects` MUST accept an empty list, keep the order given, and expose a defensively copied view. | `doc/rules/testing.md` §4 (empty, immutability) | unhappy (empty and minimal) |
| FR-010 | `OngoingEffects` MUST return an error matching `srderr.ErrInvalidInput` when two entries share an `OngoingEffectID`. | `doc/rules/architecture.md` §3 | unhappy (invalid input) |

### 4.3 Area of Effect: origin rules

The six shapes, whole-feet dimensions and `AreaShape.IncludesOrigin` are
`spell-effect-data.md` FR-013 and FR-014. New here: the creator's option for four
shapes, the Cone width, the Emanation rule and the unseen-point origin.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-013 | A Cone's width at a distance along its length MUST equal that distance (case table 4.3b), as an extension on `spell-effect-data.md`'s `ConeArea`. | SRD 5.2.1, "Cone", p. 179 | happy |
| FR-014 | `ConeArea.WidthAt(distance)` MUST return an error matching `srderr.ErrInvalidInput` for a distance below 0 or above the Cone's length. | SRD 5.2.1, "Cone" ("maximum length"), p. 179 | unhappy (boundary) |
| FR-016 | `ShapeDefaultOrigin(shape)` MUST give `OriginIncluded` equal to `shape.IncludesOrigin`. `CreatorIncludesOrigin(shape)` MUST give `OriginIncluded` true and is accepted only for an `OverridableAreaShape`, which has exactly four variants (Cone, Cube, Emanation, Line). A Cylinder or Sphere always includes its origin and cannot be given a creator decision, because the SRD offers the option only to the four shapes whose origin is not included by default (case table 4.3c). | SRD 5.2.1, "Cone", "Cube", p. 179 ("unless its creator decides otherwise"); "Emanation", p. 181; "Line", p. 184; "Cylinder", p. 180; "Sphere", p. 188 | unhappy (rule interaction) |
| FR-017 | An Emanation MUST move with its origin creature or object unless the effect is `instantaneous` or `stationary`: `EmanationEffectTiming.ongoing` gives `MovesWithOrigin` true, `instantaneous` and `stationary` give false. | SRD 5.2.1, "Emanation", p. 181 | happy |
| FR-018 | `AreaOriginResolution` MUST be `NearSideOfObstruction` when the creator places the origin at an unseen point and an obstruction lies between the creator and that point, and `ChosenPoint` otherwise (`SeenPoint` and `UnseenPointNoObstruction` give `ChosenPoint`; `UnseenPointBehindObstruction` gives `NearSideOfObstruction`). | SRD 5.2.1, "Area of Effect", p. 177 | unhappy (rule interaction) |

#### Case table 4.3b: Cone width (FR-013, FR-014)

Source for every row: SRD 5.2.1, "Cone", p. 179 (15 feet wide at 15 feet from the origin).

| Case | Cone length | Distance | Expected width | Path |
|---|---|---|---|---|
| the SRD's own example | 60 | 15 | 15 | happy |
| at the origin | 60 | 0 | 0 | unhappy (boundary) |
| at maximum length | 60 | 60 | 60 | unhappy (boundary) |
| one beyond maximum length | 60 | 61 | an invalid-input error (`srderr.ErrInvalidInput`) | unhappy (boundary) |
| negative distance | 60 | -1 | an invalid-input error (`srderr.ErrInvalidInput`) | unhappy (invalid input) |

#### Case table 4.3c: origin inclusion (FR-016)

| Case | Shape | Decision | Expected `OriginIncluded` | Source | Path |
|---|---|---|---|---|---|
| each shape's default | all six shapes | `ShapeDefaultOrigin` | `shape.IncludesOrigin`: Cone, Cube, Emanation, Line false; Cylinder, Sphere true | p. 179, p. 180, p. 181, p. 184, p. 188 | happy |
| Cone included by creator | Cone | `CreatorIncludesOrigin` | true | p. 179 | unhappy (rule interaction) |
| Cube included by creator | Cube | `CreatorIncludesOrigin` | true | p. 179 | unhappy (rule interaction) |
| Emanation included by creator | Emanation | `CreatorIncludesOrigin` | true | p. 181 | unhappy (rule interaction) |
| Line included by creator | Line | `CreatorIncludesOrigin` | true | p. 184 | unhappy (rule interaction) |
| the four overridable shapes | `OverridableAreaShape` | exhaustive `switch` | exactly Cone, Cube, Emanation, Line; a Cylinder or Sphere override has no constructor | p. 179, p. 181, p. 184 | unhappy (boundary) |

### 4.4 Area membership from host facts

The host reports whether the location is inside the shape (`InsideShape`,
`OutsideShape`) and the `Cover` each straight line from the origin passes through.
The location is in the area unless it is outside the shape or every line is
blocked; only `TotalCover` blocks a line (Q-3).

`AreaMembership.Assess` returns the closed `AreaMembershipResult`:
`AreaMembershipIncluded`, `AreaMembershipExcluded(AreaExclusion)` and
`AreaMembershipNoLinesConsidered`. An excluded location is an applied answer,
not a refusal.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-019 | Containment is checked first. A location the host reports `OutsideShape` MUST NOT be in the area, whatever its lines, including an empty list (reason `OutsideShape`). | SRD 5.2.1, "Area of Effect", p. 177 | happy |
| FR-020 | A location `InsideShape` with at least one line that is not `TotalCover` MUST be in the area. | SRD 5.2.1, "Area of Effect", p. 177 | happy |
| FR-021 | A location `InsideShape` whose every line is `TotalCover` MUST NOT be in the area (reason `AllLinesBlocked`). | SRD 5.2.1, "Area of Effect" ("If all straight lines ... are blocked, that location isn't included"), p. 177 | unhappy (rule interaction) |
| FR-022 | A line passing through Half Cover or Three-Quarters Cover MUST NOT be blocked; only Total Cover blocks (case table 4.4a). | SRD 5.2.1, "Area of Effect" ("To block a line, an obstruction must provide Total Cover"), p. 177; "Cover", p. 15 | unhappy (boundary) |
| FR-023 | A location reported `InsideShape` with an empty list of lines MUST return `AreaMembershipNoLinesConsidered` (no error, no panic; class `invalid`, section 7): "no lines considered" is neither open nor blocked (FR-144). An `OutsideShape` location with an empty list is decided by FR-019 first. | `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 (empty) | unhappy (empty and minimal) |
| FR-024 | The origin location MUST be in the area exactly when the resolved `OriginIncluded` (`spell-effect-data.md` FR-014 and FR-016 here) is true, with no lines or containment given. | SRD 5.2.1, "Cone", p. 179; "Cylinder", p. 180; "Sphere", p. 188 | happy |
| FR-025 | A location both outside the shape and fully blocked MUST report the single reason `OutsideShape` (the first decisive check). | Package design (`doc/rules/testing.md` §4, determinism) | unhappy (rule interaction) |
| FR-026 | Assessing a location MUST NOT change the lines list given, MUST return an equal result for equal inputs, and MUST hold its own defensive copy of any list it keeps. | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |

#### Case table 4.4a: lines and membership (FR-019 to FR-025)

Baseline: Sphere, location reported by the host as shown.

| Case | Containment | Lines | Expected | FR | Source | Path |
|---|---|---|---|---|---|---|
| inside, open line | `InsideShape` | `[HalfCover]` | in area | FR-020, FR-022 | p. 177, p. 15 | happy |
| inside, three-quarters | `InsideShape` | `[ThreeQuartersCover]` | in area | FR-022 | p. 177, p. 15 | unhappy (boundary) |
| inside, one open of many | `InsideShape` | `[TotalCover, TotalCover, HalfCover]` | in area | FR-020 | p. 177 | unhappy (rule interaction) |
| inside, all Total Cover | `InsideShape` | `[TotalCover, TotalCover]` | not in area (`AllLinesBlocked`) | FR-021 | p. 177 | unhappy (rule interaction) |
| inside, a single Total Cover line | `InsideShape` | `[TotalCover]` | not in area (`AllLinesBlocked`) | FR-021 | p. 177 | unhappy (boundary) |
| outside, open | `OutsideShape` | `[HalfCover]` | not in area (`OutsideShape`) | FR-019 | p. 177 | happy |
| outside and blocked | `OutsideShape` | `[TotalCover]` | not in area (`OutsideShape`) | FR-025 | p. 177 | unhappy (rule interaction) |
| no lines | `InsideShape` | `[]` | `AreaMembershipNoLinesConsidered` | FR-023 | package | unhappy (empty and minimal) |
| outside, no lines | `OutsideShape` | `[]` | not in area (`OutsideShape`) | FR-019, FR-023 | p. 177 | unhappy (rule interaction) |
| origin of a Sphere | origin location | none | in area | FR-024 | p. 188 | happy |
| origin of a Cone | origin location | none | not in area (`OriginExcluded`) | FR-024 | p. 179 | unhappy (rule interaction) |

### 4.5 Curses

A curse is an `OngoingEffect` of category `curse` on a creature (objects: Q-8); the engine applies no curse behaviour.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | A curse MUST be an `OngoingEffect` of category `curse`, identified only by its id, category and conferring `EffectSource`: two curses are equal exactly when all three are equal, and `RemoveCurse` (FR-030 to FR-032) reads nothing else, because "the effect that confers a curse defines what the curse does". | SRD 5.2.1, "Curses", p. 179 | happy |
| FR-028 | `CurseRemovalMeansFromSpell(SpellID) (CurseRemovalMeans, bool)` MUST return `RemoveCurseSpell` for `remove_curse`, `GreaterRestorationSpell` for `greater_restoration`, and the zero value with `false` for any other spell (case table 4.5a). | SRD 5.2.1, "Curses" ("the Remove Curse and Greater Restoration spells"), p. 179 | happy |
| FR-029 | `MagicThatEndsCurses(EffectSource)` MUST return an error matching `srderr.ErrInvalidInput` when the source is not magical (FR-001 to FR-004): only "other magic that explicitly ends curses" is a removal means. | SRD 5.2.1, "Curses", p. 179; "Magical Effect", p. 185 | unhappy (invalid input) |
| FR-030 | `RemoveCurse` with an `OngoingEffectID` naming a curse in the creature's `OngoingEffects` MUST return `RemoveCurseApplied` whose effects list lacks that curse and keeps every other entry in order, with one `CurseRemoved` event. | SRD 5.2.1, "Curses", p. 179 | happy |
| FR-031 | `RemoveCurse` with an id not in the list MUST return `RemoveCurseNotPresent` with the effects unchanged and no events. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-032 | `RemoveCurse` with an id naming an effect whose category is not `curse` MUST return `RemoveCurseNotACurse` carrying that category, the effects unchanged and no events. | SRD 5.2.1, "Curses", p. 179 | unhappy (state error) |
| FR-033 | `CurseRemoved` MUST carry the creature's `CombatantID`, the removed `OngoingEffectID` and the `CurseRemovalMeans`, and be equal by value. | `doc/rules/architecture.md` §3 (domain event) | happy |

#### Case table 4.5a: spells that remove curses (FR-028)

| Case | `SpellID` | Expected | Source | Path |
|---|---|---|---|---|
| Remove Curse | `remove_curse` | `RemoveCurse` | p. 179 | happy |
| Greater Restoration | `greater_restoration` | `GreaterRestoration` | p. 179 | happy |
| a different spell | `dispel_magic` | zero value, `false` | p. 179 | unhappy (boundary) |
| id of the possession ending spell | `dispel_evil_and_good` | zero value, `false` | p. 179, p. 186 | unhappy (rule interaction) |
| unknown id | `no_such_spell` | zero value, `false` | p. 179 | unhappy (invalid input) |

### 4.6 Possession

The possession state (`DemonicPossessionState`, `PossessionEndingMagic`) is
`toolbox-curses-and-contagions.md`'s; this spec classifies only the two spells the
entry names. FR-145 says how curse removal and possession combine.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-037 | `PossessionPreventionFromSpell(SpellID) (PossessionPrevention, bool)` MUST return `ProtectionFromEvilAndGood` for `protection_from_evil_and_good` and the zero value with `false` for any other spell (case table 4.6a). | SRD 5.2.1, "Possession", p. 186 | happy |
| FR-039 | `PossessionEndingFromSpell(SpellID) (PossessionEnding, bool)` MUST return `DispelEvilAndGood` for `dispel_evil_and_good` and the zero value with `false` for any other spell; `PossessionEnding.PossessingEffect` MUST also exist for an end the possessing effect itself defines (case table 4.6a). | SRD 5.2.1, "Possession" ("ended by the Dispel Evil and Good spell"), p. 186 | happy |

#### Case table 4.6a: spells named by Possession (FR-037, FR-039)

| Case | `SpellID` | `...FromSpell` function | Expected | Source | Path |
|---|---|---|---|---|---|
| prevention | `protection_from_evil_and_good` | `PossessionPrevention` | `ProtectionFromEvilAndGood` | p. 186 | happy |
| ending | `dispel_evil_and_good` | `PossessionEnding` | `DispelEvilAndGood` | p. 186 | happy |
| ending id given to prevention | `dispel_evil_and_good` | `PossessionPrevention` | zero value, `false` | p. 186 | unhappy (rule interaction) |
| prevention id given to ending | `protection_from_evil_and_good` | `PossessionEnding` | zero value, `false` | p. 186 | unhappy (rule interaction) |
| a curse spell | `remove_curse` | `PossessionEnding` | zero value, `false` | p. 186 | unhappy (boundary) |

### 4.7 Illusions

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-042 | `IllusionTraits` MUST hold a non-empty set of `IllusionSenseID`s the illusion deceives; an id names a sense or mental faculty, is supplied by the host or the effect description (no content kind carries it) and is not a closed list, with `visual` the one known value. An empty set MUST return an error matching `srderr.ErrInvalidInput`. The non-empty rule is package design (Q-10: an illusion that deceives nothing is not an illusion); the SRD says only that the effect "defines ... which senses or mental faculties it deceives" (the visual and audible illusions are its two examples). | Package design (Q-10); SRD 5.2.1, "Illusions", p. 183 | unhappy (invalid input) |
| FR-043 | `IllusionTraits.DeceivesSight` MUST be true exactly when `IllusionSenseID.visual` is among the senses; this is the "Visual Illusion" predicate the senses spec consumes. An id other than `visual` (for example a made-up `hearing`) MUST be accepted and MUST NOT make `DeceivesSight` true. | SRD 5.2.1, "Illusions" (a visual illusion), p. 183; "Truesight" ("Visual Illusions"), p. 190 | happy |
| FR-044 | An illusion that `ManifestsInSpace` MUST have `BodyFacts` of `IllusionBodyFacts` with insubstantial true and weight 0 pounds. | SRD 5.2.1, "Illusions", p. 183 | happy |
| FR-045 | An illusion that `ManifestsInSpace` MUST appear affected by the environment as if real (`EnvironmentAppearance.AsIfReal`, the default) unless its creating effect specifies otherwise (`EnvironmentAppearance.SpecifiedByEffect`). | SRD 5.2.1, "Illusions" (shadows, reflections, wind, an echo), p. 183 | happy |
| FR-046 | `NotInSpace` MUST have no body-facts and no environment-appearance members: only `ManifestsInSpace` declares them, so no nil value stands in (checked by an exhaustive `switch` over `IllusionManifestation`). | SRD 5.2.1, "Illusions" ("If an illusion manifests in space"), p. 183 | unhappy (boundary) |

### 4.8 Shape-Shifting

Statistics and Hit Points of Polymorph, True Polymorph, Shapechange and Animal Shapes belong to `shape-changing-spells.md`; this spec owns what carries over and the revert on death. `OngoingEffects` (no duration) and `ActiveEffects` (`active-effects.md`) are separate lists: a shift keeps the registry, and a host that applies a `CarryOverExcept` removal also ends the matching active effects with `EndingCause.RemovedByEffect` (`active-effects.md` FR-044), which this spec does not call.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-047 | `ShiftShape` with `CarryOverAll` MUST leave the creature's `ConditionSet` and `OngoingEffects` exactly as given (every condition, spell and curse carries over) and report `NotCarriedOver` empty (case table 4.8a). | SRD 5.2.1, "Shape-Shifting" ("any ongoing effects on you—conditions, spells, curses, and the like—carry over"), p. 187 | happy |
| FR-048 | `ShiftShape` with `CarryOverExcept(conditions, effects)` MUST remove each named explicit condition kind from the `ConditionSet` (through `conditions.md` `remove`) and each named `OngoingEffectID` from the `OngoingEffects`, and list what was removed in `NotCarriedOver` ("Unless that description says otherwise"). A named Exhaustion is removed entirely, whatever its level, through `ReduceExhaustion(target, level)` (`conditions.md` FR-044, `ExhaustionEnded`), and is listed once; the single-level `remove` of `conditions.md` FR-036 is not used for it. | SRD 5.2.1, "Shape-Shifting", p. 187 | unhappy (rule interaction) |
| FR-049 | `CarryOverExcept` naming a condition kind the creature does not hold as an explicit condition (absent, or effective only because another explicit condition implies it, `conditions.md` FR-032, FR-033; the implying kind is what to name), or an `OngoingEffectID` not in its list, MUST return `ShiftShapeExceptionNotHeld` listing what was not held, with the state unchanged and no events (FR-144). | `doc/rules/architecture.md` §6 | unhappy (state error) |
| FR-050 | A successful `ShiftShape` MUST return `ShiftShapeApplied`, move the state from `TrueForm` to `ShiftedForm(effect)` and emit one `ShapeShifted` event carrying the creature and the effect `EffectSource`. | SRD 5.2.1, "Shape-Shifting" ("its description specifies what happens to you"), p. 187 | happy |
| FR-051 | `ShiftShape` on a creature already in `ShiftedForm` MUST succeed and replace the effect with the new one, keeping the creature shifted (package decision, Q-13). | Package design (Q-13); SRD 5.2.1, "Shape-Shifting", p. 187 | unhappy (state error) |
| FR-052 | `RevertToTrueForm` MUST take the current `ShapeShiftState`, `ConditionSet`, `OngoingEffects` and a `CarryOverRule` (default `CarryOverAll`, supplied from the effect's description like a shift). It MUST return `RevertToTrueFormApplied` with `TrueForm`, carry the ongoing effects back by the semantics of FR-047 and FR-048 ("from one form to the other"), and emit one `RevertedToTrueForm` event. | SRD 5.2.1, "Shape-Shifting", p. 187 | happy |
| FR-053 | `RevertToTrueForm` on a creature in `TrueForm` MUST return `RevertToTrueFormApplied` with the same state and no events. | `doc/rules/testing.md` §4 (state errors) | unhappy (empty and minimal) |
| FR-054 | `RevertOnDeath` MUST return `RevertOnDeathApplied` with `TrueForm` and one `RevertedToTrueForm` event with cause `died` when the creature is in `ShiftedForm` and is Dead; for a creature in `TrueForm` it MUST return `RevertOnDeathApplied` with no event. | SRD 5.2.1, "Shape-Shifting" ("You revert to your true form if you die"), p. 187 | unhappy (state error) |
| FR-055 | `RevertOnDeath` for a creature that is not Dead MUST return `RevertOnDeathNotDead` with the state unchanged and no events; the host calls it after `Died` (`hit-points-and-damage.md`). | `doc/rules/architecture.md` §6 | unhappy (state error) |

#### Case table 4.8a: carry-over (FR-047, FR-048)

Baseline: `ConditionSet` {Poisoned, Frightened(source)}, `OngoingEffects` [spell `bless`, curse `c1`].

| Case | `CarryOverRule` | Expected conditions | Expected effects | `NotCarriedOver` | FR | Path |
|---|---|---|---|---|---|---|
| default | `CarryOverAll` | Poisoned, Frightened | `bless`, `c1` | empty | FR-047 | happy |
| curse excluded | `CarryOverExcept(effects: {c1})` | Poisoned, Frightened | `bless` | `c1` | FR-048 | unhappy (rule interaction) |
| condition excluded | `CarryOverExcept(conditions: {Poisoned})` | Frightened | `bless`, `c1` | Poisoned | FR-048 | unhappy (rule interaction) |
| nothing to carry | `CarryOverAll`, empty set and list | empty | empty | empty | FR-047 | unhappy (empty and minimal) |
| exception names an absent condition | `CarryOverExcept(conditions: {Blinded})` | unchanged | unchanged | `ShiftShapeExceptionNotHeld` | FR-049 | unhappy (state error) |
| exception names an absent effect | `CarryOverExcept(effects: {nope})` | unchanged | unchanged | `ShiftShapeExceptionNotHeld` | FR-049 | unhappy (state error) |
| exception names an implied-only kind | set {Paralyzed}, `CarryOverExcept(conditions: {Incapacitated})` | unchanged | unchanged | `ShiftShapeExceptionNotHeld` listing Incapacitated | FR-049 | unhappy (rule interaction) |
| exception names Exhaustion | set {Exhaustion 3}, `CarryOverExcept(conditions: {Exhaustion})` | no Exhaustion (all levels) | unchanged | Exhaustion, once | FR-048 | unhappy (rule interaction) |

### 4.9 Teleportation

The host supplies the destination facts: unoccupied, occupied by a creature or
blocked by a solid obstacle, and whether the teleporter sees it. The Teleport spell's
arrival table is `spell-outcome-tables.md` (prose pointer, no dependency).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-056 | `Teleport` to a destination reported `Unoccupied` MUST return `TeleportArrivedAtDestination`. | SRD 5.2.1, "Teleportation", p. 190; "Unoccupied Space", p. 191 | happy |
| FR-057 | `Teleport` to a destination reported `OccupiedByCreature` MUST return `TeleportArrivedNearDestination`, which states that the teleporter chooses the nearest unoccupied space (the host supplies it; the engine has no map). | SRD 5.2.1, "Teleportation" ("the nearest unoccupied space of your choice"), p. 190; "Occupied Space", p. 185 | unhappy (rule interaction) |
| FR-058 | `Teleport` to a destination reported `BlockedBySolidObstacle` MUST return `TeleportArrivedNearDestination`, exactly as in FR-057. | SRD 5.2.1, "Teleportation", p. 190 | unhappy (rule interaction) |
| FR-059 | When the effect requires the teleporter to see the destination (`DestinationSight.required`) and the host reports it is not seen, `Teleport` MUST return `TeleportDestinationNotSeen` and no creature moves. | SRD 5.2.1, "Teleportation" ("tells you if you must see the teleportation's destination"), p. 190 | unhappy (state error) |
| FR-060 | When the effect does not require sight (`DestinationSight.NotRequired`), an unseen destination MUST NOT cause a rejection; when it is required and the destination is seen, `Teleport` MUST proceed as FR-056 to FR-058. | SRD 5.2.1, "Teleportation", p. 190 | unhappy (boundary) |
| FR-061 | `TeleportDestinationNotSeen` MUST take precedence over the destination being occupied or blocked: a required-but-unseen destination is rejected before placement is considered. | Package design (`doc/rules/testing.md` §4, determinism) | unhappy (rule interaction) |
| FR-062 | A teleportation MUST cost no movement unless a rule says so: `MovementCostFeet` MUST be 0 when the effect states no cost. | SRD 5.2.1, "Teleportation" ("doesn't expend movement unless a rule tells you otherwise"), p. 190 | happy |
| FR-063 | A cost the effect states MUST be reported as `MovementCostFeet`; a negative cost MUST return an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Teleportation", p. 190 | unhappy (boundary) |
| FR-064 | Teleportation provokes no Opportunity Attack: `initiative-and-actions.md` decides that it is not a `DepartureMeans` (its decision row), so the teleportation files of this spec MUST NOT import or name `DepartureMeans`. Verified by the name scan in `internal/archtest/` (FR-143), not by a unit test. | SRD 5.2.1, "Teleportation", p. 190; "Opportunity Attacks", p. 15 | unhappy (rule interaction) |
| FR-066 | A creature the teleporter is touching MUST NOT teleport (`TouchedCreaturePolicy.StayBehind`, the default). | SRD 5.2.1, "Teleportation" ("that creature doesn't teleport with you unless the teleportation effect says otherwise"), p. 190 | unhappy (rule interaction) |
| FR-067 | `TeleportingCreatures` holds only the touched creatures, never the teleporter. With `TouchedCreaturePolicy.accompany` (the effect says so) it MUST list every touched creature in the order given; with `StayBehind` it MUST be empty. | SRD 5.2.1, "Teleportation", p. 190 | unhappy (rule interaction) |
| FR-068 | A touched set that contains the teleporter's own `CombatantID`, or a repeated id, MUST return `TeleportInvalidTouchedSet` listing the offending ids, with no creature moved and no events (class `invalid`, section 7; FR-144). | `doc/rules/testing.md` §4 (invalid input); `combat-foundations.md` FR-031 | unhappy (invalid input) |
| FR-069 | An arrival MUST emit one `CreatureTeleported` event carrying the teleporter, the `TeleportingCreatures` and whether the arrival was at the destination or near it; a rejection MUST emit none. | `doc/rules/architecture.md` §3 (domain event) | happy |
| FR-070 | `Teleport` MUST NOT alter its input touched set, MUST return defensively copied lists, and MUST give equal results for equal inputs. | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |

#### Case table 4.9a: destination and sight (FR-056 to FR-061)

| Case | Destination | `DestinationSight` | Seen | Expected | FR | Path |
|---|---|---|---|---|---|---|
| clear and seen | `Unoccupied` | required | seen | arrived at destination | FR-056, FR-060 | happy |
| clear, sight not required, unseen | `Unoccupied` | NotRequired | unseen | arrived at destination | FR-060 | unhappy (boundary) |
| clear, sight required, unseen | `Unoccupied` | required | unseen | `TeleportDestinationNotSeen` | FR-059 | unhappy (state error) |
| occupied by a creature | `OccupiedByCreature` | NotRequired | unseen | arrived near destination | FR-057 | unhappy (rule interaction) |
| blocked by a solid obstacle | `BlockedBySolidObstacle` | required | seen | arrived near destination | FR-058 | unhappy (rule interaction) |
| occupied, sight required, unseen | `OccupiedByCreature` | required | unseen | `TeleportDestinationNotSeen` | FR-061 | unhappy (rule interaction) |

### 4.10 Moved out

FR-071 to FR-090 are now `telepathy.md` (FR-001 to FR-020); FR-097 to FR-108, FR-110,
FR-111 and FR-146 `returning-to-life.md`; FR-130 to FR-136 and FR-138 to FR-142
`readying-a-spell.md`; FR-137 `initiative-and-actions.md` (FR-072). Each Withdrawn
row in 4.16 names the live id. This spec defines no Concentration state, event or
check (`casting-and-concentration.md`).

### 4.11 Simultaneous Effects

The package names who decides the order and validates the order the host brings back.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-091 | `OrderingAuthority.ForTurnHolder(TurnHolder)` MUST be `player` for `TurnHolder.player` and `GameMaster` for `TurnHolder.GameMaster`: the person at the table whose turn it is decides, including for effects at the start of a player character's turn (a player character's turn gives `player`, a monster's turn `GameMaster`). `TurnHolder` is named so as not to collide with `spell-effect-data.md`'s `TurnOwner`. | SRD 5.2.1, "Simultaneous Effects", p. 187 | happy |
| FR-092 | `OrderSimultaneousEffects` with a chosen order that is a permutation of the pending ids MUST return `OrderSimultaneousEffectsOrdered` carrying that order exactly as chosen (never sorted) and the `OrderingAuthority`. | SRD 5.2.1, "Simultaneous Effects" ("decides the order in which those things happen"), p. 187 | happy |
| FR-093 | A chosen order that omits a pending id, repeats an id, or names an id that is not pending MUST return `OrderSimultaneousEffectsRejected` listing every problem (`missing`, `repeated`, `unknown`, and `DuplicatePending` from FR-095) in the order the ids were given (class `invalid`, section 7). | `doc/rules/testing.md` §4 (invalid input); `doc/rules/architecture.md` §5 (all problems at once); `combat-foundations.md` FR-031 | unhappy (invalid input) |
| FR-094 | Fewer than two pending effects MUST return `OrderSimultaneousEffectsOrdered` with the pending list as is (empty included): there is nothing to decide. | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-095 | A pending list that contains the same `PendingEffectID` twice MUST return `OrderSimultaneousEffectsRejected` with the problem `DuplicatePending`, the same result type as FR-093 (class `invalid`, FR-144). | `doc/rules/architecture.md` §6 | unhappy (invalid input) |
| FR-096 | The result MUST NOT alter the lists given and MUST be equal for equal inputs. | `doc/rules/testing.md` §4 (immutability, determinism) | unhappy (immutability) |

### 4.15 Layering, failures, events and telemetry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-143 | Every type here MUST live in `combat`, be pure and synchronous, and import only: `combat`'s own domain files, the kernel packages, and the published language (value objects, ids and events) of `conditions.md`, `attacks.md`, `hit-points-and-damage.md` and `content` (`ConditionKind`, `SpellID`, `ItemID`, `FeatureID`, and `AreaShape` of `spell-effect-data.md`). It MUST NOT import a domain service or use case of `character` (no `ConcentrationRules`), anything of `toolbox` (downstream, FR-045 of the decisions), `telemetry`, any adapter or another context's internals, and MUST NOT import the domain of `hiding-searching-and-senses.md` or `encounter-aggregate.md`. | `doc/rules/architecture.md` §1, §2, §9; `shared-types-and-ownership.md` FR-013, FR-015, FR-045 | unhappy (forbidden edge) |
| FR-144 | One rule for host-supplied input. A list or set the host supplies, whose validity depends on the state it describes, MUST produce an operation-prefixed result and not an error or panic: an empty list of lines for an `InsideShape` location (FR-023), a shift exception naming something not held (FR-049), a revert-on-death for a creature that is not Dead (FR-055), a touched set holding the teleporter or a repeated id (FR-068), and a repeated pending id (FR-095). Each result is classified by `combat-foundations.md` FR-031, which `architecture.md` §6 does not override: malformed host input (FR-023, FR-068, FR-093, FR-095) is `invalid`; state-dependent outcomes (FR-049, FR-055 and the rest) are `refused`. A constructor error matching `srderr.ErrInvalidInput` is kept for invariants of a value (dimension minimums, duplicate ids inside `OngoingEffects`, a negative cost). | `doc/rules/architecture.md` §6; `combat-foundations.md` FR-031 | unhappy (invalid input) |
| FR-145 | `RemoveCurse` acts on a creature's `OngoingEffects`, and Demonic Possession (`toolbox-curses-and-contagions.md` FR-026) on its `DemonicPossessionState` (a downstream type: a pointer, not an import). They are independent states and neither calls the other. When curse-removing magic is applied to a creature that has both, the host MUST call both operations; each is correct alone. This spec holds no possession state. | SRD 5.2.1, "Curses", p. 179; "Possession", p. 186; `toolbox-curses-and-contagions.md` FR-025, FR-026 | unhappy (rule interaction) |
| FR-147 | Each event of the `MagicalEffectEvent` family MUST implement `CombatEvent` with a constant `Kind()` and a defensively copied `Combatants()` slice in the order of table 4.15a, and the `internal/archtest` scan (`combat-foundations.md` FR-016 to FR-019) MUST find every kind unique in the package. The family is a closed variant set (an interface with an unexported marker method) and each event is past tense (`shared-types-and-ownership.md` FR-019). | `combat-foundations.md` FR-015 to FR-019; `shared-types-and-ownership.md` FR-019 | unhappy (name collision) |
| FR-148 | Telemetry is the use-case layer's, through the shared `CombatTelemetry` (`combat-foundations.md` FR-029 to FR-035). Amendment RA-1 to `combat-action-resolution.md` (not made here) adds one use case per operation family, each run in one `CombatTelemetry.Run` call with the `CombatOperation` named in section 7. Each calls the pure services of this spec and never another use case, and records only its own signals (`shared-types-and-ownership.md` FR-044). This spec adds no signal name of its own. | `combat-foundations.md` FR-029 to FR-035; `shared-types-and-ownership.md` FR-044; `doc/rules/observability.md` §2, §3 | happy |

#### Table 4.15a: event kinds and combatants (FR-147)

| Event | `kind` | `combatants` order |
|---|---|---|
| `CurseRemoved` | `curse_removed` | the creature |
| `ShapeShifted` | `shape_shifted` | the creature |
| `RevertedToTrueForm` | `reverted_to_true_form` | the creature |
| `CreatureTeleported` | `creature_teleported` | the teleporter, then `TeleportingCreatures` in order (touched creatures only, FR-067, so no duplicate) |

### 4.16 Withdrawn ids

Withdrawn ids MUST NOT be reused.

| Id | Status |
|---|---|
| FR-011 | Withdrawn: moved to spell-effect-data.md (FR-013) |
| FR-012 | Withdrawn: moved to spell-effect-data.md (FR-013) |
| FR-015 | Withdrawn: moved to spell-effect-data.md (FR-014) |
| FR-034 | Withdrawn: moved to toolbox-curses-and-contagions.md (FR-007) |
| FR-035 | Withdrawn: moved to toolbox-curses-and-contagions.md (FR-007) |
| FR-036 | Withdrawn: moved to toolbox-curses-and-contagions.md (FR-108) |
| FR-038 | Withdrawn: moved to toolbox-curses-and-contagions.md (FR-009) |
| FR-040 | Withdrawn: moved to toolbox-curses-and-contagions.md (FR-025) |
| FR-041 | Withdrawn: moved to toolbox-curses-and-contagions.md (FR-027) |
| FR-065 | Withdrawn: documentation only (section 1, Out of scope) |
| FR-071 | Withdrawn: moved to telepathy.md (FR-001) |
| FR-072 | Withdrawn: moved to telepathy.md (FR-002) |
| FR-073 | Withdrawn: moved to telepathy.md (FR-003) |
| FR-074 | Withdrawn: moved to telepathy.md (FR-004) |
| FR-075 | Withdrawn: moved to telepathy.md (FR-005) |
| FR-076 | Withdrawn: moved to telepathy.md (FR-006) |
| FR-077 | Withdrawn: moved to telepathy.md (FR-007) |
| FR-078 | Withdrawn: moved to telepathy.md (FR-008) |
| FR-079 | Withdrawn: moved to telepathy.md (FR-009) |
| FR-080 | Withdrawn: moved to telepathy.md (FR-010) |
| FR-081 | Withdrawn: moved to telepathy.md (FR-011) |
| FR-082 | Withdrawn: moved to telepathy.md (FR-012) |
| FR-083 | Withdrawn: moved to telepathy.md (FR-013) |
| FR-084 | Withdrawn: moved to telepathy.md (FR-014) |
| FR-085 | Withdrawn: moved to telepathy.md (FR-015) |
| FR-086 | Withdrawn: moved to telepathy.md (FR-016) |
| FR-087 | Withdrawn: moved to telepathy.md (FR-017) |
| FR-088 | Withdrawn: moved to telepathy.md (FR-018) |
| FR-089 | Withdrawn: moved to telepathy.md (FR-019) |
| FR-090 | Withdrawn: moved to telepathy.md (FR-020) |
| FR-097 | Withdrawn: moved to returning-to-life.md (FR-001) |
| FR-098 | Withdrawn: moved to returning-to-life.md (FR-002) |
| FR-099 | Withdrawn: moved to returning-to-life.md (FR-003) |
| FR-100 | Withdrawn: moved to returning-to-life.md (FR-004) |
| FR-101 | Withdrawn: moved to returning-to-life.md (FR-005) |
| FR-102 | Withdrawn: moved to returning-to-life.md (FR-006) |
| FR-103 | Withdrawn: moved to returning-to-life.md (FR-007) |
| FR-104 | Withdrawn: moved to returning-to-life.md (FR-008) |
| FR-105 | Withdrawn: moved to returning-to-life.md (FR-009) |
| FR-106 | Withdrawn: moved to returning-to-life.md (FR-010) |
| FR-107 | Withdrawn: moved to returning-to-life.md (FR-011) |
| FR-108 | Withdrawn: moved to returning-to-life.md (FR-012) |
| FR-109 | Withdrawn: moved to magic-item-rules.md (FR-054) |
| FR-110 | Withdrawn: moved to returning-to-life.md (FR-013) |
| FR-111 | Withdrawn: moved to returning-to-life.md (FR-014) |
| FR-112 | Withdrawn: moved to casting-and-concentration.md (FR-075) |
| FR-113 | Withdrawn: moved to casting-and-concentration.md (FR-075) |
| FR-114 | Withdrawn: moved to casting-and-concentration.md (FR-077) |
| FR-115 | Withdrawn: moved to spellcasting.md (FR-127) |
| FR-116 | Withdrawn: moved to casting-and-concentration.md (FR-075) |
| FR-117 | Withdrawn: moved to casting-and-concentration.md (FR-077) |
| FR-118 | Withdrawn: moved to casting-and-concentration.md (FR-076) |
| FR-119 | Withdrawn: moved to casting-and-concentration.md (FR-076) |
| FR-120 | Withdrawn: moved to casting-and-concentration.md (FR-078) |
| FR-121 | Withdrawn: moved to casting-and-concentration.md (FR-079) |
| FR-122 | Withdrawn: moved to casting-and-concentration.md (FR-078) |
| FR-123 | Withdrawn: moved to casting-and-concentration.md (FR-099) |
| FR-124 | Withdrawn: moved to casting-and-concentration.md (FR-075) |
| FR-125 | Withdrawn: moved to casting-and-concentration.md (FR-099) |
| FR-126 | Withdrawn: moved to casting-and-concentration.md (FR-119) |
| FR-127 | Withdrawn: moved to casting-and-concentration.md (FR-077) |
| FR-128 | Withdrawn: moved to spellcasting.md (FR-028) |
| FR-129 | Withdrawn: moved to spellcasting.md (FR-084) |
| FR-130 | Withdrawn: moved to readying-a-spell.md (FR-001) |
| FR-131 | Withdrawn: moved to readying-a-spell.md (FR-002) |
| FR-132 | Withdrawn: moved to readying-a-spell.md (FR-003) |
| FR-133 | Withdrawn: moved to readying-a-spell.md (FR-004) |
| FR-134 | Withdrawn: moved to readying-a-spell.md (FR-005) |
| FR-135 | Withdrawn: moved to readying-a-spell.md (FR-006) |
| FR-136 | Withdrawn: moved to readying-a-spell.md (FR-007) |
| FR-137 | Withdrawn: moved to initiative-and-actions.md (FR-072) |
| FR-138 | Withdrawn: moved to readying-a-spell.md (FR-008) |
| FR-139 | Withdrawn: moved to readying-a-spell.md (FR-009) |
| FR-140 | Withdrawn: moved to readying-a-spell.md (FR-010) |
| FR-141 | Withdrawn: moved to readying-a-spell.md (FR-011) |
| FR-142 | Withdrawn: moved to readying-a-spell.md (FR-012) |
| FR-146 | Withdrawn: moved to returning-to-life.md (FR-015) |

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

Pure, synchronous and silent: state in, new state and events out. Owners of the
reused types are named in sections 1 to 3 (`CombatantID`, `CombatEvent`,
`ConditionSet`, `Cover`, `HitPointStatus`, `SpellID`, `AreaShape`).

- **Value objects:** the types of section 3, plus `AreaLocationFacts` (closed:
  `OriginLocationFacts`, `OtherLocationFacts(containment, lines)`),
  `ShapeContainment`, `AreaExclusion` (`OutsideShape`, `AllLinesBlocked`,
  `OriginExcluded`), `OriginPlacement` (`SeenPoint`, `UnseenPointNoObstruction`,
  `UnseenPointBehindObstruction`), `IllusionBodyFacts`, `EnvironmentAppearance`,
  `CarryOverRule` (`CarryOverAll`, `CarryOverExcept`). `OverridableAreaShape` is
  closed with exactly four variants wrapping `ConeArea`, `CubeArea`, `EmanationArea`,
  `LineArea`, so a Cylinder or Sphere override cannot be built. `OngoingEffectID`
  and `IllusionSenseID` are defined types over `string` (non-empty / open with
  known value `visual`); `IllusionSenseID` is a local open vocabulary whose ids the
  host or the effect description supplies, and no content kind carries it.
- **Events:** closed family `MagicalEffectEvent implements CombatEvent` (decisions
  FR-019; table 4.15a). **Services** (stateless): `AreaMembership.Assess`,
  `AreaOrigin.Resolve`, `CurseRemoval.Remove`, `ShapeShifting.Shift`/`.Revert`/  `.RevertOnDeath`, `Teleportation.Resolve`, `SimultaneousEffects.Order`.
- **Invariants** (constructor): FR-007, FR-008, FR-010, FR-029, FR-042, FR-063
  (the cost check in `TeleportCommand`); FR-014 is a method argument check
  (`ConeArea.WidthAt`).
- **Failures:** `refused` and `invalid` outcomes (FR-144) are closed results with
  operation-prefixed variants (section 7); constructor violations are
  constructor errors matching `srderr.ErrInvalidInput` (`doc/rules/architecture.md` §6). No `*srderr.PortError`: no port.

## 6. Interface

Every operation is a synchronous pure function over host-held values (no use case,
port or `ctx`; RA-1 is section 7). Exported from package `combat`;
the `internal/archtest` scan (`doc/rules/architecture.md` §9) must find no
duplicate name: `AreaShape` and variants are not exported here, `TurnHolder` is
this spec's while `TurnOwner` stays `spell-effect-data.md`'s, and `NotPossessed`,
`PossessionState` and the Concentration names are not defined here. Handoff types:
`AreaMembershipResult`, `OngoingEffects`, `TeleportResult`, `ShapeShiftState`, the
`MagicalEffectEvent` family. No test doubles (decisions FR-038); `toolbox` imports
`CurseRemovalMeans`, `PossessionPrevention`, `PossessionEnding`, `OngoingEffect` and
`EffectSource` from package `combat` (its FR-045).

## 7. Observability

The domain is silent (`doc/rules/observability.md` §3). This spec owns no signal:
telemetry is RA-1, through the shared `CombatTelemetry` (`combat-foundations.md`
FR-029 to FR-035; `srd5e.outcome` is `success` or `failure` only, decisions
FR-041). RA-1 adds the five `CombatOperation` names below (the CamelCase use cases
of each, `combat-foundations.md` FR-022) and the closed `srd5e.combat.result` set to
`doc/observability/signals.md`. Classes (`combat-foundations.md` FR-030, FR-031):
`applied` (`success`) is a completed rule answer, including `AreaMembershipExcluded`;
`refused` (`failure`, no error status, no log) is legal input the rules do not
allow in the current state; `invalid` (`failure`, span error status, no `error.type`,
no log: decisions FR-051) is host input the engine cannot use. Result values hold
no ids, distances, amounts or Hit Points. Operations: `remove_combatant_curse`
(`CurseRemoval.Remove`), `shift_combatant_shape` (`ShapeShifting.*`),
`teleport_combatant` (`Teleportation.Resolve`), `resolve_effect_order`
(`SimultaneousEffects.Order`), `assess_area_location` (`AreaMembership.Assess`,
`AreaOrigin.Resolve`).

| Operation | Variant | `srd5e.combat.result` | Class |
|---|---|---|---|
| `remove_combatant_curse` | `RemoveCurseApplied` | `completed` | applied |
| | `RemoveCurseNotPresent` | `curse_not_present` | refused |
| | `RemoveCurseNotACurse` | `curse_not_a_curse` | refused |
| `shift_combatant_shape` | `ShiftShapeApplied`, `RevertToTrueFormApplied`, `RevertOnDeathApplied` | `completed` | applied |
| | `ShiftShapeExceptionNotHeld` | `shift_exception_not_held` | refused |
| | `RevertOnDeathNotDead` | `revert_not_dead` | refused |
| `teleport_combatant` | `TeleportArrivedAtDestination`, `TeleportArrivedNearDestination` | `completed` | applied |
| | `TeleportDestinationNotSeen` | `teleport_destination_not_seen` | refused |
| | `TeleportInvalidTouchedSet` | `teleport_invalid_touched_set` | invalid |
| `resolve_effect_order` | `OrderSimultaneousEffectsOrdered` | `completed` | applied |
| | `OrderSimultaneousEffectsRejected` | `order_rejected` | invalid |
| `assess_area_location` | `AreaMembershipIncluded`, resolved `AreaOriginResolution` | `completed` | applied |
| | `AreaMembershipExcluded` | `area_location_excluded` | applied |
| | `AreaMembershipNoLinesConsidered` | `area_no_lines_considered` | invalid |

The seven questions (`doc/rules/observability.md` §2), one line each:

1. **What can go wrong?** Wrong host geometry, a call in the wrong state (`refused`), malformed input (`invalid`), or an invalid dimension, duplicate id or negative cost (a constructor error matching `srderr.ErrInvalidInput`).
2. **How would an operator know?** `srd5e.combat.operations` by operation and result, once RA-1 lands; an `invalid` result sets the span error status, a `refused` one does not; `srd5e.combat.errors` counts only port `*srderr.PortError`s (`combat-foundations.md` FR-032).
3. **What would someone need to diagnose it?** Results and events carry the deciding inputs: exclusion reason, `CurseRemovalMeans`, what carried over and what did not.
4. **What should be counted or timed?** Each call by operation and result, through `CombatTelemetry`.
5. **Is existing instrumentation still correct?** Yes: no Concentration state is defined here.
6. **What should a consumer monitor?** Sustained non-zero `invalid` results: the host builds malformed input.
7. **Is anything now noisy?** No: no log is added.

## 8. Test plan

All unit tests in `combat/*_test.go` (no ports, no doubles, no observability
tests: `doc/rules/observability.md` §9), except architecture tests in
`internal/archtest/`.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-005 | unit | `EffectSource` cases; equality across variants |
| FR-006 to FR-010 | unit | Category and source invariants; empty list; defensively copied view; duplicate ids |
| FR-013 to FR-018 | unit | Tables 4.3b and 4.3c, one test per row; an exhaustive `switch` over `OverridableAreaShape` shows exactly four variants; one test per timing and per placement |
| FR-019 to FR-026 | unit | Table 4.4a, one test per row; input list unchanged, copy defensively copied, equal results for equal inputs |
| FR-027 to FR-033 | unit | Two curses differing only in source are not equal; table 4.5a; `MagicThatEndsCurses` with each non-magical and magical source; `RemoveCurse` present, absent, wrong category; event fields; other entries keep order |
| FR-037, FR-039 | unit | Table 4.6a |
| FR-042 to FR-046 | unit | `IllusionTraits`: empty senses, an unlisted sense id accepted, `DeceivesSight`, body facts on `ManifestsInSpace`; FR-046 by an exhaustive `switch` over `IllusionManifestation` |
| FR-047 to FR-055 | unit | Table 4.8a; shift while shifted; revert with each `CarryOverRule`; revert on death for each state; `ShiftShapeExceptionNotHeld`; `RevertOnDeathNotDead` |
| FR-056 to FR-063, FR-066 to FR-070 | unit | Table 4.9a, one test per row; cost, touched creatures, invalid touched set result, event, immutability |
| FR-064 | architecture | Name scan: the teleportation files neither import nor name `DepartureMeans`; no unit test (an absent field cannot be asserted on in a compiled test) |
| FR-091 to FR-096 | unit | Both turn holders; permutation, never sorted, all problems in input order, fewer than two, duplicate pending ids as a rejection, immutability |
| FR-143 | architecture | Import-direction check for `combat` files of this spec; no `character` service import; the export scan finds no duplicate or generic variant name and none of `spell-effect-data.md`'s `TurnOwner`, `NotPossessed`, `AreaShape` |
| FR-144 | unit | One test per rejection result of the rule, each asserting its class in the section 7 table (`invalid` for FR-023, FR-068, FR-093, FR-095; `refused` for the rest) |
| FR-145 | unit | A creature with a curse effect and a possession state: each operation alone gives its own result |
| FR-147 | architecture, unit | Table 4.15a: each `kind` constant, unique; `combatants` order |
| FR-148 | scenario, deferred | Needs RA-1 in `combat-action-resolution.md`; the signal checks are its tests |

No doubles or contract suites. Mutation testing (`doc/rules/testing.md` §8): all
changed domain lines (`gomu`, threshold 80, run only inside Docker with `--cpus` and `--workers` capped at a quarter of the host processors, never on the host); boundary rows
exist for FR-014. Not covered: FR-148 (deferred); that `initiative-and-actions.md`
treats a teleportation as no departure (its own decision row; only FR-064's test).

## 9. Open questions and decisions

Open: none. Escalated: none.

Settled (one line each):

- Q-8: creatures only. The SRD lets a curse affect "a creature or an object" (p. 179) but the effect defines what it does and the package tracks no object state or object identity (`spell-effects.md` FR-093 keeps a spell-local opaque id only); the host holds a cursed object's state. An object target would be a new requirement with a package-level object id.
- Q-10: `IllusionSenseID` is an open local id for a sense or mental faculty ("which senses or mental faculties it deceives", p. 183), supplied by the host or effect description, `visual` the one known value; non-empty is package design (FR-042).
- Q-11: the host holds `OngoingEffects` and `ShapeShiftState` beside its other per-creature values; the encounter aggregate does not store them (package design, `doc/rules/architecture.md` §2; every operation here takes and returns them).
- Q-13: interpretation. The Glossary says only that the description specifies what happens (p. 187) and gives no stacking rule; a shift while shifted succeeds and replaces the effect (FR-051), because "revert to your true form" names a single true form that a replacement leaves unchanged.
- Q-21: interpretation. The SRD names the nearest unoccupied space only for the teleporter (p. 190); the host places accompanying creatures and the result lists them (FR-057, FR-067).
- Q-25: `CarryOverExcept` may name Exhaustion, removed entirely at any level (the Glossary gives no per-level carry-over, p. 187; FR-048); an implied-only condition is listed as not held (FR-049, `conditions.md` FR-032).
- Q-26: not this spec's. Table K already caps at 30 (K8: 200 gives 30; K7: 61 is an exact half, not a capped value); RA-5 asks for 62, the first capped value (half 31, capped, p. 179).

- Q-1: the order is the rank table 4.11 (`combat-foundations.md` FR-024): rank 26, right of `initiative-and-actions` (25), left of `encounter-aggregate` (28).
- Q-3: keep `Cover`, not `LineOfSight`, so "only Total Cover blocks" is testable (FR-022).
- Q-4: no map; origin and line rules only, containment and per-line Cover are host facts (4.4, section 1).
- Q-5: `AreaShape` is `spell-effect-data.md` FR-013 and FR-014 (decisions FR-059).
- Q-7: no origin override for Cylinder or Sphere, the SRD gives none (pp. 180, 188); `OverridableAreaShape` makes it unbuildable (FR-016).
- Q-9: model the category `MagicalContagion` only (FR-006); the source is supplied by `toolbox-curses-and-contagions.md` FR-109.
- Q-12: Possession state is `toolbox-curses-and-contagions.md` (decisions FR-045, FR-048).
- Q-23: FR-028 classifies the spell for creature curses; `magic-item-rules.md` FR-059 acts on an item's Attunement; neither calls the other, and `magic-item-rules.md` FR-117 takes a `SpellID` because `character` cannot import `combat`.
- Q-6, Q-14 to Q-16, Q-18 to Q-20: moved with their requirements to `telepathy.md`, `returning-to-life.md` and `readying-a-spell.md`.

## 10. Changelog impact

Expected `CHANGELOG.md` entry under `Unreleased`:

- **Added:** the magical-effects rules of the Rules Glossary in `combat`: Area of
  Effect origin and membership, Curse removal, Illusions, Shape-Shifting,
  Teleportation and Simultaneous Effects (`doc/specs/magical-effects.md`);
  telepathy, returning to life and readying a spell are their own specs.
- **BREAKING:** no. Everything is new; no existing exported type, result or
  signal changes. The change requests RA-1, RA-4 and RA-5 may each change another
  spec when settled.
