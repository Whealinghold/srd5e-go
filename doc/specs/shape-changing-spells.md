# Shape-changing spells

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `combat-foundations.md`, `hit-points-and-damage.md`, `magical-effects.md`, `content-monsters.md`, `content-schema-and-decoder.md`, `d20-tests.md` |
| Referenced, not imported | `content-spells.md` and `spell-effect-data.md` (test oracle for each spell's level, Concentration and duration; these spells stay text-only there); `casting-and-concentration.md` (owner of Concentration; the host passes its `ConcentrationEnded` as the plain fact of FR-049); `dispel-and-counterspell.md` (owner of the Dispel Magic plan, conclusion and the registry of ongoing spells; the host passes only the result, FR-047); `class-druid.md` (Wild Shape, FR-079 to FR-085: a different feature with its own retained list; never mixed, Q-10); `spell-effects.md` (resolves the saving throw; this spec takes its outcome) |
| Consumed by | `combat-action-resolution.md` (amendment RA-1, section 7) |

Rank 28 (`shared-types-and-ownership.md` 4.11, shared with `encounter-aggregate` and `dispel-and-counterspell`; the row to add is listed in section 9): right of
`hit-points-and-damage` (21), `magical-effects` (26) and `spell-effects` (27); left of
`combat-action-resolution` (29). "Depends on" lists only the specs whose types sections 5 and 6
import. Under decisions FR-048 this spec owns the four spells' behaviour once they are cast (form
eligibility, kept and replaced statistics, Temporary Hit Points, ending); `magical-effects.md` keeps
the carry-over and revert rules ("Shape-Shifting", p. 187), `hit-points-and-damage.md` the Temporary
Hit Point arithmetic, `class-druid.md` Wild Shape.

## 1. Purpose and scope

The four spells of SRD 5.2.1 that shape-shift a creature: Polymorph (p. 153), True Polymorph (p. 170),
Shapechange (p. 161) and Animal Shapes (p. 108). Pure domain in package `combat`, state in and
new state and events out, no dice, no port, no use case (`shared-types-and-ownership.md` FR-044):

- which forms each spell allows (type, Challenge Rating or level, size, "seen before");
- which statistics the target keeps, which the form replaces, and what the form forbids (speaking,
  casting, using gear);
- the Temporary Hit Points equal to the form's Hit Points, and the early end when Polymorph's run out;
- changing form, the Concentration link, the end causes and True Polymorph's permanence.

**Out of scope:**

- Casting (slot, components, Concentration state: `casting-and-concentration.md`), choosing targets,
  range, line of sight, and the Wisdom saving throw roll (`spell-effects.md`, `d20-tests.md`); the
  outcome arrives as an input.
- Condition and ongoing-effect carry-over and the revert to true form (`magical-effects.md` FR-047 to
  FR-055): this spec only names the `CarryOverRule` and the `EffectSource` to pass (FR-024).
- The form's Armor Class, Speed, attacks and other statistics: they are the `Monster` entry's
  (`content-monsters.md`). Object statistics, Hit Points of an object, and what becomes of a creature
  whose object form is destroyed (Q-7); the SRD pages read give no rule.
- Wild Shape, the lycanthrope and other stat-block shape-shifting traits, Truesight revealing a true
  form (`hiding-searching-and-senses.md` FR-045), and Dispel Magic itself (only its result, FR-047).
- Anything outside SRD 5.2.1. Nothing core to these four spells was left out.

## 2. Source

PDF page numbers equal printed page numbers. The full text of True Polymorph is on p. 170; p. 171
holds none of it (Q-8).

- SRD 5.2.1, "Polymorph", pp. 153-154; "True Polymorph", p. 170; "Shapechange", p. 161; "Animal Shapes",
  p. 108.
- "Shape-Shifting", p. 187; "Temporary Hit Points" ("Lose Temporary Hit Points First", "Duration",
  "They Don't Stack", "They're Not Hit Points or Healing"), p. 18; "Saving Throws" (choosing to fail), p. 7;
  "Concentration", p. 179; "Challenge Rating", p. 178; "Creature Type", p. 179; "Monster Stat Blocks", p. 256; "Size", p. 254;
  "Dispel Magic", p. 124.
- Package design: `shared-types-and-ownership.md` FR-017 to FR-019, FR-044, FR-048, FR-062;
  `doc/rules/architecture.md` §3, §4, §6; `doc/rules/observability.md` §2, §3; `doc/rules/testing.md` §4, §5, §8.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Polymorph, True Polymorph, Shapechange, Animal Shapes | `ShapeShiftSpell` (enum: `polymorph`, `TruePolymorph`, `shapechange`, `AnimalShapes`) | `SpellID` is the `SpellID` `polymorph`, `true_polymorph`, `shapechange`, `animal_shapes`. Behaviour owner; the ids are content's. |
| shape-shifts, new form | `AssumedForm` (closed variant set: `CreatureForm(FormFacts, CreatureSize)`, `ObjectForm(CreatureSize)`) | The size is one of the form's printed size options (FR-017). |
| the stat block of the chosen form | `FormFacts` (`MonsterID`, `CreatureType`, sizes, `ChallengeRating`, printed average Hit Points) | `NewFormFacts(content.Monster) (FormFacts, error)`, as `class-druid.md` FR-171 does for `BeastFormFacts`. |
| Challenge Rating ... (or level) | `TargetRating` (closed variant set: `RatedByChallenge(ChallengeRating)`, `RatedByLevel(int)`) | Level is 1 to 20; compared on the `ChallengeRating` order (FR-008). |
| nonmagical object | `ObjectFacts` (`CreatureSize size`, `IsMagical`, `IsWornOrCarried`) | Host facts; the SRD gives objects the creature size categories (Q-7). |
| transformation of True Polymorph | `TrueTransformation` (closed variant set: `CreatureIntoCreature`, `ObjectIntoCreature`, `CreatureIntoObject`) | The three modes of p. 170. |
| willing, unwilling | `TargetResponse` (closed variant set: `Willing`, `Unwilling{Save d20.Outcome}`) | A willing target may choose to fail without rolling (p. 7), so `Willing` needs no roll. |
| retains | `RetainedStatistic` (enum) and `FormRestrictions` | Table B. |
| Temporary Hit Points of the form | `FormTemporaryHitPoints` (enum: `FromForm`, `NotGranted`, `declined`, `Replaced`) | Provenance (FR-028). |
| equipment | `EquipmentOutcome` (enum: `MeldsUnusable`, `DropsToGround`, `ResizesToFit`, `TransformsWithCreature`) | FR-036, FR-037. |
| the spell on one target | `ActiveTransformation` | The value the host holds per transformed creature; no identity of its own beyond target and spell; keyed by the target's `CombatantID` (FR-062). |
| the spell ends | `ShapeShiftEndCause` (enum, FR-043) | |
| Concentration | `MaintainedBy` (closed variant set: `MaintainedByConcentration`, `NotMaintained`) | Package name; the Concentration itself is `casting-and-concentration.md`'s. |
| monster Spellcasting entry of Shapechange | `ShapechangeTerms` (closed variant set: `StandardShapechange`, `StatBlockShapechange`) | SRD stat blocks, p. 269: "Beast or Humanoid form only, no Temporary Hit Points gained ... no Concentration". |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3). Package names with no SRD source:
`ActiveTransformation`, `MaintainedBy`, `FormFacts`, `TargetRating`, `ObjectFacts`, `TargetResponse`,
`FormTemporaryHitPoints`, `EquipmentOutcome`, `ShapeShiftEndCause`.

## 4. Requirements

Ids are stable. Case tables follow; each row is one test citing its FR.

### 4.1 The four spells

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `ShapeShiftSpell` MUST have exactly the four members above, each with a `SpellID`, and `ShapeShiftSpellOf(id content.SpellID) (ShapeShiftSpell, bool)` MUST return the member and `true` for the four ids, and the zero value and `false` for any other id (no error, no panic). | SRD 5.2.1, "Polymorph", p. 153; "True Polymorph", p. 170; "Shapechange", p. 161; "Animal Shapes", p. 108 | happy |
| FR-002 | Each member MUST expose the profile of table A: Concentration, maximum duration, whether the target saves, whether it grants Temporary Hit Points, whether it ends early when they are gone, whether the form can change. A test compares level, Concentration and duration with the bundled `Spell` entry of the same id. | SRD 5.2.1, the four spell entries (table A pages) | happy |
| FR-003 | `StatisticsOf(spell, mode)` MUST return the `RetainedStatistic`s of table B and state that every other game statistic is replaced by the form's stat block. The returned sets are copies (mutating one changes nothing) and equal by value. | SRD 5.2.1, "Polymorph", p. 153; "Shapechange", p. 161; "Animal Shapes", p. 108; "True Polymorph", p. 170 | happy |
| FR-004 | `RestrictionsOf(spell, mode)` MUST return `FormRestrictions` (`ActionsLimitedByAnatomy`, `speech`: `CannotSpeak` or `AbilityToCommunicateRetained`, `spellcasting`: `CannotCast` or `SpellcastingFeatureRetained`) per table B. Shapechange states none of the three limits, so it has `ActionsLimitedByAnatomy` false. | SRD 5.2.1, "Polymorph", p. 154; "Animal Shapes", p. 108; "Shapechange", p. 161; "True Polymorph", p. 170 | unhappy (rule interaction) |
| FR-005 | The `SpellcastingFeatureRetained` of Shapechange MUST apply only when the caster has the Spellcasting feature (an input fact); a caster without it gets `NoSpellcastingFeature` (nothing retained, nothing forbidden by the spell). | SRD 5.2.1, "Shapechange" ("If you have the Spellcasting feature, you retain it too"), p. 161 | unhappy (boundary) |

#### Table A: spell profiles (FR-002)

| Spell | Level | Concentration | Max duration | Target and save | Temporary Hit Points | Ends early | Form change | Page |
|---|---|---|---|---|---|---|---|---|
| Polymorph | 4 | yes | 1 hour | one creature seen, 60 ft; Wisdom save | form's Hit Points | when none left | no | 153 |
| True Polymorph | 9 | yes | 1 hour; until dispelled if held to the end | one creature or nonmagical object, 30 ft; unwilling creature saves Wisdom | creature into creature only | on death or destruction only | no | 170 |
| Shapechange | 9 | yes | 1 hour | self | first form's Hit Points | no | Magic action | 161 |
| Animal Shapes | 8 | no | 24 hours | any number of willing creatures, 30 ft; no save | first form's Hit Points | target's Bonus Action | Magic action (caster) | 108 |

#### Table B: kept and replaced (FR-003, FR-004)

| Spell and mode | Retained statistics | Anatomy limits actions | Speech | Spellcasting |
|---|---|---|---|---|
| Polymorph | alignment, personality, creature type, Hit Points, Hit Point Dice | yes | cannot speak | cannot cast |
| True Polymorph, creature into creature | Hit Points, Hit Point Dice, alignment, personality | yes | cannot speak | cannot cast |
| True Polymorph, other modes | none (object statistics replace them; Q-7) | n/a | n/a | n/a |
| Shapechange | creature type, alignment, personality, Intelligence, Wisdom and Charisma scores, Hit Points, Hit Point Dice, proficiencies, ability to communicate, Spellcasting feature if held | no | ability to communicate retained | feature retained |
| Animal Shapes | creature type, Hit Points, Hit Point Dice, alignment, ability to communicate, Intelligence, Wisdom and Charisma scores | yes | ability to communicate retained | cannot cast |

### 4.2 Form eligibility

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `NewFormFacts(content.Monster)` MUST map the id, `Monster.Type`, the size options, `ChallengeRating` and the printed average Hit Points (`HitPointsEntry.Average`, at least 1); it returns an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a zero `Monster` or an average below 1. | `content-monsters.md` FR-015, FR-030; SRD 5.2.1, "Hit Points", p. 255 | happy |
| FR-007 | `FormFacts` and `AssumedForm` MUST be immutable and equal by value; `NewCreatureForm(facts, size)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (a typed `*SizeError{Size, Allowed}`) for a size not among the form's printed size options; the zero `AssumedForm` is invalid. | `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |
| FR-008 | `NewRatedByLevel(n)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (a typed range error with `Min` 1 and `Max` 20) for n below 1 or above 20. A rating allows a form when the form's `ChallengeRating` is equal to or less than the rating's, a level n comparing as the Challenge Rating n (table E1: level 1 allows 1/4 and 1 but not 2; a rating of Challenge Rating 0 allows only 0). | SRD 5.2.1, "Polymorph" ("equal to or less than the target's (or the target's level ...)"), p. 153; `content-monsters.md` FR-097 | unhappy (boundary) |
| FR-009 | Polymorph MUST allow a form only if it is a Beast and its Challenge Rating is within the target's rating. | SRD 5.2.1, "Polymorph", p. 153 | happy |
| FR-010 | Animal Shapes MUST allow a form only if it is a Beast, its chosen size is Large or smaller, and its Challenge Rating is 4 or lower. | SRD 5.2.1, "Animal Shapes", p. 108 | unhappy (boundary) |
| FR-011 | Standard Shapechange MUST allow a form whose Challenge Rating is no higher than the caster's level or Challenge Rating (the caster's rating), that the caster has seen the sort of before (an input fact `HasSeenKind`), and that is neither a Construct nor an Undead. Any other creature type is allowed. | SRD 5.2.1, "Shapechange", p. 161 | happy |
| FR-012 | `StatBlockShapechange` MUST additionally allow only Beast and Humanoid forms. It follows the other rules of FR-011. | SRD 5.2.1, "Adult Brass Dragon" (Spellcasting), p. 269; "Shapechange", p. 161 | unhappy (rule interaction) |
| FR-013 | True Polymorph into a creature MUST allow a form of any creature type whose Challenge Rating is within the target's rating. | SRD 5.2.1, "True Polymorph" (Creature into Creature), p. 170 | happy |
| FR-014 | True Polymorph of an object into a creature MUST require a nonmagical object that is neither worn nor carried, a form whose chosen size is no larger than the object's, and a Challenge Rating of 9 or lower. | SRD 5.2.1, "True Polymorph" (Object into Creature, and the target sentence), p. 170 | unhappy (boundary) |
| FR-015 | True Polymorph of a creature into an object MUST require a nonmagical object form no larger than the creature's size. | SRD 5.2.1, "True Polymorph" (Creature into Object), p. 170 | unhappy (boundary) |
| FR-016 | An ineligible form MUST be reported as `FormIneligible(reasons)`, every violated `FormIneligibility` in the fixed order `NotABeast`, `ConstructOrUndead`, `NotBeastOrHumanoid`, `SizeTooLarge`, `ChallengeRatingTooHigh`, `KindNotSeen`, `ObjectIsMagical`, `ObjectWornOrCarried`; an eligible form is `FormEligible`. The list is never empty. `Begin` and `ChangeForm` carry the list in their rejection (FR-058). | `doc/rules/architecture.md` §5, §6 (all problems, fixed order) | unhappy (rule interaction) |
| FR-017 | A form with several printed sizes ("Medium or Small") MUST be judged by the size the host chose, never by the largest or smallest. | SRD 5.2.1, "Size", p. 254; "Animal Shapes" ("Large or smaller"), p. 108 | unhappy (boundary) |

#### Table E1: eligibility (FR-008 to FR-016; one test per row)

| Case | Spell | Target rating | Form | Expected | FR | Path |
|---|---|---|---|---|---|---|
| level 1, CR 1 | Polymorph | level 1 | Beast CR 1 | eligible | FR-008 | unhappy (boundary) |
| level 1, CR 2 | Polymorph | level 1 | Beast CR 2 | `ChallengeRatingTooHigh` | FR-008 | unhappy (boundary) |
| CR 0 target | Polymorph | CR 0 | Beast CR 0; CR 1/8 | eligible; `ChallengeRatingTooHigh` | FR-008 | unhappy (boundary) |
| non-Beast | Polymorph | CR 5 | Humanoid CR 1 | `NotABeast` | FR-009 | unhappy (invalid input) |
| two reasons | Polymorph | level 1 | Dragon CR 5 | `NotABeast`, `ChallengeRatingTooHigh` | FR-016 | unhappy (rule interaction) |
| Animal Shapes CR 4 / 5 | Animal Shapes | n/a | Large Beast CR 4; CR 5 | eligible; `ChallengeRatingTooHigh` | FR-010 | unhappy (boundary) |
| Animal Shapes Large / Huge | Animal Shapes | n/a | Beast CR 1 Large; Huge | eligible; `SizeTooLarge` | FR-010 | unhappy (boundary) |
| size options | Animal Shapes | n/a | Beast sizes Huge or Large, chosen Large; chosen Huge | eligible; `SizeTooLarge` | FR-017 | unhappy (boundary) |
| Shapechange Construct | Shapechange | caster CR 10 | Construct CR 1 | `ConstructOrUndead` | FR-011 | unhappy (rule interaction) |
| Shapechange Undead | Shapechange | caster level 12 | Undead CR 3 | `ConstructOrUndead` | FR-011 | unhappy (rule interaction) |
| Shapechange Dragon | Shapechange | caster level 17 | Dragon CR 17; CR 18 | eligible; `ChallengeRatingTooHigh` | FR-011 | unhappy (boundary) |
| Shapechange unseen | Shapechange | caster level 17 | Dragon CR 1, not seen | `KindNotSeen` | FR-011 | unhappy (state error) |
| stat-block Shapechange | `StatBlockShapechange` | CR 13 | Beast; Humanoid; Dragon | eligible; eligible; `NotBeastOrHumanoid` | FR-012 | unhappy (rule interaction) |
| True Polymorph Construct | True Polymorph, creature | level 20 | Construct CR 20 | eligible | FR-013 | happy |
| object to Large creature | True Polymorph, object | Medium object | Large Beast CR 1 | `SizeTooLarge` | FR-014 | unhappy (boundary) |
| object to CR 9 / 10 | True Polymorph, object | Huge object | Huge CR 9; CR 10 | eligible; `ChallengeRatingTooHigh` | FR-014 | unhappy (boundary) |
| magic or worn object | True Polymorph, object | Medium object | magical; worn | `ObjectIsMagical`; `ObjectWornOrCarried` | FR-014 | unhappy (rule interaction) |
| creature to larger object | True Polymorph, creature | Medium creature | Large object | `SizeTooLarge` | FR-015 | unhappy (boundary) |
| creature to equal object | True Polymorph, creature | Medium creature | Medium object | eligible | FR-015 | unhappy (boundary) |
| object to equal creature | True Polymorph, object | Medium object | Medium Beast CR 1 | eligible | FR-014 | unhappy (boundary) |
| creature CR boundary | True Polymorph, creature | CR 5 | Beast CR 5; CR 6 | eligible; `ChallengeRatingTooHigh` | FR-013 | unhappy (boundary) |

### 4.3 Beginning a transformation

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | `BeginShapeShift` for Polymorph MUST treat `Unwilling(success)` as `BeginShapeShiftResisted` (nothing changes, no events) and `Willing` or `Unwilling(failure)` as shape-shifted: a willing target may choose to fail. | SRD 5.2.1, "Polymorph" ("must succeed on a Wisdom saving throw or shape-shift"), p. 153; "Saving Throws", p. 7 | happy |
| FR-019 | `BeginShapeShift` for True Polymorph MUST shape-shift a `Willing` target with no save. An `Unwilling(success)` target MUST give `BeginShapeShiftResisted` ("it isn't affected"); an `Unwilling(failure)` target is shape-shifted. This holds for a creature target in all three modes; an object has no response and is given `Willing`. | SRD 5.2.1, "True Polymorph" ("An unwilling creature can make a Wisdom saving throw"), p. 170 | unhappy (rule interaction) |
| FR-020 | `BeginShapeShift` for Animal Shapes MUST refuse an `Unwilling` target with `TargetNotWilling`: the spell chooses only willing creatures and has no save. One call serves one target; the caster's list is the host's. | SRD 5.2.1, "Animal Shapes" ("Choose any number of willing creatures"), p. 108 | unhappy (invalid input) |
| FR-021 | `BeginShapeShift` for Shapechange MUST refuse a target other than the caster with `ShapechangeNotSelf`, and an `Unwilling` response with `TargetNotWilling`. | SRD 5.2.1, "Shapechange" (Range: Self; "You shape-shift"), p. 161 | unhappy (invalid input) |
| FR-022 | `BeginShapeShift` on a target whose `LifeState` is Dead MUST be refused with `TargetIsDead` and change nothing. | SRD 5.2.1, "Dead", p. 180 | unhappy (state error) |
| FR-023 | The checks MUST run in the order of table R: dead, spell and target mismatch, response, form eligibility, then the saving throw outcome. Only the first failing check is reported, except FR-016's list inside `FormIneligible`. | `doc/rules/architecture.md` §6 (deterministic order) | unhappy (rule interaction) |
| FR-024 | A successful `BeginShapeShiftApplied` MUST carry the new `ActiveTransformation`, the new `HitPointStatus` and the ordered events, and the two inputs for `magical-effects.md` `ShiftShape`: `effect` = `SpellSource(spellID)` and `CarryOver` = `CarryOverAll`, because none of the four descriptions says otherwise ("Unless that description says otherwise ... carry over"). | SRD 5.2.1, "Shape-Shifting", p. 187 | happy |
| FR-025 | The new `ActiveTransformation` MUST be `MaintainedByConcentration` for Polymorph, True Polymorph and standard Shapechange, and `NotMaintained` for Animal Shapes and `StatBlockShapechange`. It records the caster, so the caster's Concentration ending can be matched (FR-049). | SRD 5.2.1, "Polymorph" (Duration), p. 153; "Animal Shapes" (Duration: 24 hours), p. 108; "Adult Brass Dragon", p. 269 | unhappy (rule interaction) |
| FR-026 | `BeginShapeShift` on a target already holding an `ActiveTransformation` MUST end the earlier one first (cause `Replaced`, FR-047, with its form Temporary Hit Points vanishing per FR-034) and then begin the new one; the events are the ending's, then the beginning's. The creature has one true form (`magical-effects.md` FR-051). | SRD 5.2.1, "Shape-Shifting" ("revert to your true form"), p. 187; `magical-effects.md` Q-13 (interpretation, Q-4) | unhappy (rule interaction) |

#### Table R: begin order (FR-023; one test per row)

| Case | Dead | Spell, target | Response | Form | Save | Expected | FR | Path |
|---|---|---|---|---|---|---|---|---|
| all good | no | Polymorph | `Unwilling(failure)` | eligible | n/a | Applied | FR-018 | happy |
| willing chooses to fail | no | Polymorph | `Willing` | eligible | n/a | Applied | FR-018 | unhappy (rule interaction) |
| save succeeds | no | Polymorph | `Unwilling(success)` | eligible | n/a | Resisted | FR-018 | unhappy (rule interaction) |
| TP willing, no roll | no | True Polymorph | `Willing` | eligible | n/a | Applied | FR-019 | happy |
| TP unwilling saves | no | True Polymorph | `Unwilling(success)` | eligible | n/a | Resisted | FR-019 | unhappy (rule interaction) |
| animal shapes unwilling | no | Animal Shapes | `Unwilling(failure)` | eligible | n/a | `TargetNotWilling` | FR-020 | unhappy (invalid input) |
| shapechange other target | no | Shapechange, target is not the caster | `Willing` | eligible | n/a | `ShapechangeNotSelf` | FR-021 | unhappy (invalid input) |
| dead and ineligible | yes | Polymorph | `Willing` | ineligible | n/a | `TargetIsDead` | FR-023 | unhappy (rule interaction) |
| ineligible and saves | no | Polymorph | `Unwilling(success)` | ineligible | n/a | `FormIneligible` | FR-023 | unhappy (rule interaction) |
| replaces earlier | no | Polymorph, one held | `Willing` | eligible | n/a | Applied, earlier ended `Replaced` | FR-026 | unhappy (rule interaction) |
| replaces permanent | no | Polymorph, permanent True Polymorph held | `Willing` | eligible | n/a | Applied, earlier ended `Replaced` | FR-046 | unhappy (rule interaction) |
| object becomes creature | no | True Polymorph, object | `Willing` | eligible | n/a | Applied under the host's id | FR-062 | happy |

### 4.4 Temporary Hit Points

Hit Points, Hit Point Dice and (for Polymorph) the rest are retained (table B); the form adds a buffer.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-027 | Beginning with a form that grants Temporary Hit Points (Polymorph, Animal Shapes, standard Shapechange, True Polymorph creature into creature) MUST call `HitPointStatus.GainTemporary` (`hit-points-and-damage.md` FR-109) with the form's printed average Hit Points (at least 1) and the host's `TemporaryHitPointsChoice`, and return the new status and its event, if any (FR-059). | SRD 5.2.1, "Polymorph" ("equal to the Hit Points of the Beast form"), p. 153; "Temporary Hit Points", p. 18; Q-1 | happy |
| FR-028 | The `FormTemporaryHitPoints` of the new state MUST be `FromForm` when the target held no Temporary Hit Points before or the choice was `AcceptNew`, and `declined` when it held some and the choice was `KeepCurrent` (the totals do not stack: 12 or 10, never 22). | SRD 5.2.1, "They Don't Stack", p. 18 | unhappy (rule interaction) |
| FR-029 | `StatBlockShapechange` MUST gain no Temporary Hit Points (`NotGranted`, status unchanged, no hit-point event). | SRD 5.2.1, "Adult Brass Dragon" (Spellcasting), p. 269 | unhappy (rule interaction) |
| FR-030 | True Polymorph into an object and of an object into a creature MUST gain no Temporary Hit Points (`NotGranted`); only the Creature into Creature paragraph grants them. `UntrackedHitPoints` MUST be accepted exactly where none are granted, otherwise `srderr.ErrInvalidInput`. | SRD 5.2.1, "True Polymorph", p. 170 | unhappy (rule interaction) |
| FR-031 | `ChangeForm` and a later transformation by Animal Shapes or Shapechange MUST gain no further Temporary Hit Points: the amount is that of "the first form". | SRD 5.2.1, "Shapechange" ("the first form into which you shape-shift"), p. 161; "Animal Shapes", p. 108 | unhappy (rule interaction) |
| FR-032 | `AssessShapeShift(state, HitPointStatus)` for Polymorph MUST return `AssessShapeShiftEnded` with cause `FormHitPointsDepleted` when the target's Temporary Hit Points are 0, and `AssessShapeShiftContinues` otherwise; the test reads the actual Temporary Hit Points, whatever their source. For the other three spells it MUST always continue. | SRD 5.2.1, "Polymorph" ("ends early on the target if it has no Temporary Hit Points left"), p. 153 | unhappy (boundary) |
| FR-033 | `AssessShapeShift` MUST NOT change Hit Points. The damage beyond the Temporary Hit Points was already taken from the target's own Hit Points by `ApplyDamage`, because Hit Points are retained: the target reverts with them reduced. | SRD 5.2.1, "Lose Temporary Hit Points First", p. 18; "Polymorph" (retains Hit Points), p. 153 | unhappy (rule interaction) |
| FR-034 | When a transformation ends with `FromForm` provenance and Temporary Hit Points are still held, `EndShapeShift` MUST end them through `EndTemporary` (`TemporaryHitPointsEnded`); with `declined`, `Replaced` or `NotGranted`, or none held, it MUST leave the Temporary Hit Points untouched. A target that is Dead MUST skip the call. | SRD 5.2.1, "Polymorph" ("vanish if any remain when the spell ends"), p. 153; "Animal Shapes", p. 108 | unhappy (rule interaction) |
| FR-035 | `RecordTemporaryHitPointsReplaced(state)` MUST set `FromForm` to `Replaced` when the host reports that another effect's Temporary Hit Points took the place of the form's, and MUST leave any other provenance unchanged. | SRD 5.2.1, "They Don't Stack", p. 18 | unhappy (state error) |
| FR-059 | When the form's Hit Points equal the Temporary Hit Points already held, under either choice, `GainTemporary` leaves the state unchanged, emits no event and reports `no_effect` (`hit-points-and-damage.md` FR-029); the provenance MUST then be `declined` (the held points are not the form's) and `EndShapeShift` MUST NOT end them (FR-034). The result states that no Temporary Hit Point event exists. | SRD 5.2.1, "They Don't Stack", p. 18; `hit-points-and-damage.md` FR-029 | unhappy (boundary) |

#### Table T: Temporary Hit Points (FR-027 to FR-035; one test per row)

| Case | Spell | Form Hit Points | Held before | Choice | Then | Expected | FR | Path |
|---|---|---|---|---|---|---|---|---|
| none held | Polymorph | 11 | 0 | either | n/a | temp 11, `FromForm` | FR-027, FR-028 | happy |
| more offered | Polymorph | 12 | 10 | `AcceptNew` | n/a | temp 12, `FromForm` | FR-028 | unhappy (rule interaction) |
| keep larger | Polymorph | 12 | 10 | `KeepCurrent` | n/a | temp 10, `declined` | FR-028 | unhappy (rule interaction) |
| equal amount | Polymorph | 12 | 12 | `AcceptNew` | n/a | temp 12 unchanged, no event, `declined`; end leaves it | FR-059 | unhappy (boundary) |
| damage within temp | Polymorph | 11 | 0 | n/a | 7 damage, 4 temp left | continues | FR-032 | happy |
| damage exactly temp | Polymorph | 11 | 0 | n/a | 11 damage, 0 left | ends `FormHitPointsDepleted` | FR-032 | unhappy (boundary) |
| damage past temp | Polymorph | 11 | 0 | n/a | 15 damage: 4 Hit Points lost | ends; Hit Points 4 lower | FR-033 | unhappy (rule interaction) |
| declined, then 0 | Polymorph | 12 | 10 | `KeepCurrent` | 10 damage | ends (none left) | FR-032 | unhappy (rule interaction) |
| temp 0 other spells | True Polymorph | 30 | 0 | n/a | 30 damage | continues | FR-032 | unhappy (rule interaction) |
| end, form temp held | Animal Shapes | 20 | 0 | n/a | ended by target, 6 left | `TemporaryHitPointsEnded` | FR-034 | happy |
| end, declined | Polymorph | 12 | 10 | `KeepCurrent` | ended by caster | temp 10 kept | FR-034 | unhappy (rule interaction) |
| end, replaced | Animal Shapes | 20 | 0 | n/a | other temp took over, then ended | temp kept | FR-035 | unhappy (rule interaction) |
| stat block | `StatBlockShapechange` | 20 | 0 | n/a | n/a | no temp, `NotGranted` | FR-029 | unhappy (rule interaction) |
| form change | Shapechange | 40 then 70 | 0 | n/a | `ChangeForm` | temp stays 40 | FR-031 | unhappy (rule interaction) |

### 4.5 Equipment, Concentration and speech

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-036 | The result of beginning MUST name the `EquipmentOutcome`: `MeldsUnusable` for Polymorph, Animal Shapes and True Polymorph into a creature ("melds into the new form ... can't use or otherwise benefit"); `TransformsWithCreature` for a creature turned into an object; for Shapechange the caster's choice of `DropsToGround` or `ResizesToFit`, supplied in the command and required for Shapechange; passing `MeldsUnusable` or `TransformsWithCreature` as the choice is an `srderr.ErrInvalidInput`. | SRD 5.2.1, "Polymorph", p. 154; "Animal Shapes", p. 108; "True Polymorph", p. 170; "Shapechange", p. 161 | happy |
| FR-037 | Each `ChangeForm` of Shapechange MUST take a fresh `EquipmentOutcome` choice ("Upon shape-shifting"). Animal Shapes MUST repeat `MeldsUnusable`. | SRD 5.2.1, "Shapechange", p. 161; "Animal Shapes", p. 108 | unhappy (rule interaction) |
| FR-038 | No operation MUST read, write or report the Concentration the target holds on a different spell: the SRD ties no loss of Concentration to shape-shifting for these spells (interpretation, Q-11). The only Concentration an operation reads is the maintained spell's, as the plain fact of FR-049. | SRD 5.2.1, "Concentration", p. 179; "Shape-Shifting" (spells carry over), p. 187 | unhappy (rule interaction) |

### 4.6 Changing form

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-039 | `ChangeForm` MUST be allowed only for Shapechange (actor: the caster) and Animal Shapes (actor: the caster, any target); for Polymorph and True Polymorph it MUST return `ChangeFormRefused(formChangeNotAllowed)`. | SRD 5.2.1, "Shapechange", p. 161; "Animal Shapes" ("On later turns, you can take a Magic action"), p. 108 | unhappy (state error) |
| FR-040 | A successful `ChangeForm` MUST carry `ActionCost` = Magic action as data, apply the same eligibility rules as beginning (FR-009 to FR-017, with the same `TargetRating`) to the new form, replace the `AssumedForm`, and emit one `ShapeShiftFormChanged`. An ineligible form MUST leave the state unchanged. | SRD 5.2.1, "Shapechange" ("a different eligible form"), p. 161; "Animal Shapes", p. 108 | happy |
| FR-041 | `ChangeForm` MUST leave the Hit Points, the Temporary Hit Points and their provenance, the caster and the `MaintainedBy` of the state unchanged. | SRD 5.2.1, "Shapechange", p. 161 | unhappy (immutability) |
| FR-042 | `ChangeForm` on a permanent transformation or one that has ended MUST return `ChangeFormRefused(notTransformed)`. | `doc/rules/testing.md` §4 (state errors) | unhappy (state error) |
| FR-060 | `ChangeForm` MUST refuse an actor other than the transformation's caster with `ActorNotCaster`, and a Dead target with `TargetIsDead` (the host calls `End` with `TargetDied` instead, FR-048). The check order is: dead, `FormChangeNotAllowed`, `NotTransformed`, `ActorNotCaster`, then form eligibility; only the first failure is reported. | SRD 5.2.1, "Shapechange" ("you can take a Magic action"), p. 161; "Animal Shapes", p. 108; "Dead", p. 180 | unhappy (state error) |
| FR-061 | `ChangeForm` for Shapechange without an `EquipmentOutcome` choice, or with `MeldsUnusable` or `TransformsWithCreature`, MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` and no result (FR-036); for Animal Shapes a supplied choice MUST return such an error (the outcome is always `MeldsUnusable`, FR-037). | SRD 5.2.1, "Shapechange", p. 161; "Animal Shapes", p. 108 | unhappy (invalid input) |

### 4.7 Ending and permanence

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | `ShapeShiftEndCause` MUST have exactly: `DurationElapsed`, `ConcentrationEnded`, `FormHitPointsDepleted`, `TargetDied`, `TargetDestroyed`, `EndedByTarget`, `Dispelled`, `Replaced`. | SRD 5.2.1, the four spell entries; "Shape-Shifting", p. 187 | happy |
| FR-044 | `EndShapeShift(state, cause, HitPointStatus)` MUST return `EndShapeShiftApplied`: the transformation cleared, the events of FR-034, then one `ShapeShiftEnded(cause)` that states `RevertsToTrueForm` true and carries `CarryOver` = `CarryOverAll` for `magical-effects.md` `RevertToTrueForm`. | SRD 5.2.1, "Shape-Shifting" ("from one form to the other"), p. 187 | happy |
| FR-045 | `DurationElapsed` MUST end every transformation that is not permanent. `ElapseFullDuration(state, concentrationHeldToTheEnd, status)` for True Polymorph MUST, when Concentration was kept for the full duration, return `ElapseShapeShiftMadePermanent` (state `permanent`, no end, one `ShapeShiftMadePermanent`) and otherwise end it with `DurationElapsed`; for the other spells it always ends. | SRD 5.2.1, "True Polymorph" ("if you maintain Concentration on this spell for the full duration, the spell lasts until dispelled"), p. 170 | unhappy (boundary) |
| FR-046 | A permanent transformation MUST end only by `Dispelled`, `TargetDied` or `TargetDestroyed`; `DurationElapsed`, `ConcentrationEnded`, `EndedByTarget` and `FormHitPointsDepleted` MUST return `EndShapeShiftUnaffected`, but `Replaced` MUST be accepted: the earlier spell is superseded (FR-026, Q-4). `FormHitPointsDepleted` is never produced for a permanent state (FR-032 reads only Polymorph). Making an Object into Creature permanent MUST report `ControlLost` true ("If the spell lasts more than an hour, you no longer control the creature"); otherwise the event carries the control terms of FR-051. | SRD 5.2.1, "True Polymorph", p. 170 | unhappy (state error) |
| FR-047 | `Dispelled` MUST be accepted for every spell: the host applies Dispel Magic through `dispel-and-counterspell.md` (an automatic end when the spell's level is at most the slot's, otherwise an ability check, p. 124) and passes only that it ended; the end is then as FR-044. | SRD 5.2.1, "Dispel Magic", p. 124; "True Polymorph" ("until dispelled"), p. 170 | happy |
| FR-048 | `TargetDied` MUST end the transformation whatever the spell or permanence, with `RevertsToTrueForm` true, and skip the Temporary Hit Point call (FR-034). `TargetDestroyed` is the same for an object form or an object-turned-creature target (the creature or the object "dies or is destroyed"). | SRD 5.2.1, "Shape-Shifting" ("You revert to your true form if you die"), p. 187; "True Polymorph", p. 170 | unhappy (state error) |
| FR-049 | `EndShapeShift` with `ConcentrationEnded` and the id of the creature whose Concentration ended MUST end a `MaintainedByConcentration` transformation whose caster is that creature; with another creature, or with `NotMaintained`, or a permanent state, it MUST return `EndShapeShiftUnaffected`. The host derives the fact from `casting-and-concentration.md` `ConcentrationEnded`. | SRD 5.2.1, "Concentration", p. 179; "Polymorph" (Duration), p. 153 | unhappy (rule interaction) |
| FR-050 | `EndedByTarget` MUST be accepted for Animal Shapes only (a Bonus Action of the target; the result carries `ActionCost` Bonus Action) and MUST return `EndShapeShiftRefused(cannotEndAtWill)` for the other spells. It ends that target's transformation alone. | SRD 5.2.1, "Animal Shapes" ("until the target ends it as a Bonus Action"), p. 108 | unhappy (state error) |
| FR-051 | Beginning True Polymorph of an object into a creature MUST carry `AnimatedObjectTerms`: the creature is Friendly to the caster and the caster's allies, takes its turns immediately after the caster's, and obeys the caster's commands. Ending a creature into an object MUST set `MemoryOfObjectFormLost` true on `ShapeShiftEnded`. | SRD 5.2.1, "True Polymorph" (Object into Creature, Creature into Object), p. 170 | happy |
| FR-052 | `EndShapeShift` on a state with no transformation MUST return `EndShapeShiftUnaffected` and no events; on a Dead target only `TargetDied` and `TargetDestroyed` are accepted, any other cause returns `EndShapeShiftRefused(targetIsDead)`. | `doc/rules/testing.md` §4 | unhappy (empty and minimal) |
| FR-053 | `ElapseFullDuration` and `AssessShapeShift` MUST NOT end a transformation for a Dead target: the death ends it through `TargetDied` (FR-048). | SRD 5.2.1, "Shape-Shifting", p. 187 | unhappy (state error) |

#### Table X: ending (FR-044 to FR-053; one test per row)

| Case | Spell, state | Cause | Expected | FR | Path |
|---|---|---|---|---|---|
| caster stops Concentration | Polymorph, caster c | `ConcentrationEnded` by c | ended | FR-049 | happy |
| other creature | Polymorph, caster c | `ConcentrationEnded` by d | unaffected | FR-049 | unhappy (rule interaction) |
| not maintained | Animal Shapes | `ConcentrationEnded` by caster | unaffected | FR-049 | unhappy (rule interaction) |
| stat block | `StatBlockShapechange` | `ConcentrationEnded` | unaffected | FR-025, FR-049 | unhappy (rule interaction) |
| duration, Polymorph | Polymorph | `DurationElapsed` | ended | FR-045 | happy |
| duration, TP held | True Polymorph | full duration, Concentration kept | permanent, no end event | FR-045 | unhappy (boundary) |
| duration, TP lost | True Polymorph | full duration, Concentration not kept | ended | FR-045 | unhappy (boundary) |
| permanent, Concentration | permanent TP | `ConcentrationEnded` | unaffected | FR-046 | unhappy (state error) |
| permanent, dispelled | permanent TP | `Dispelled` | ended | FR-046, FR-047 | happy |
| permanent, replaced | permanent TP | `Replaced` | ended | FR-046 | unhappy (rule interaction) |
| permanent, form temp | permanent TP | `FormHitPointsDepleted` | unaffected | FR-046 | unhappy (state error) |
| object-turned-creature dies | TP, object | `TargetDestroyed` under the host's id | ended, reverts | FR-048, FR-062 | unhappy (state error) |
| permanent object to creature | TP, object | made permanent | `ControlLost` true | FR-046 | unhappy (rule interaction) |
| permanent creature to creature | TP, creature | made permanent | `ControlLost` not reported | FR-046 | unhappy (rule interaction) |
| death | Shapechange | `TargetDied` | ended, reverts, no Temporary Hit Point call | FR-048 | unhappy (state error) |
| death while permanent | permanent TP | `TargetDied` | ended | FR-048 | unhappy (state error) |
| Bonus Action, Animal Shapes | Animal Shapes | `EndedByTarget` | ended, Bonus Action | FR-050 | happy |
| Bonus Action, Polymorph | Polymorph | `EndedByTarget` | `CannotEndAtWill` | FR-050 | unhappy (state error) |
| creature into object ends | TP, creature into object | `Dispelled` | `MemoryOfObjectFormLost` | FR-051 | unhappy (rule interaction) |
| no transformation | none | any | unaffected | FR-052 | unhappy (empty and minimal) |

### 4.8 Model invariants, events and layering

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-054 | `ActiveTransformation` MUST be immutable, equal by value, with collections copied on construction and on read, and its constructor `NewActiveTransformation(...) (ActiveTransformation, error)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for: `permanent` on any spell but True Polymorph; `FromForm` provenance on a spell or mode that grants none; `MaintainedByConcentration` on Animal Shapes or `StatBlockShapechange`; a Shapechange whose target differs from its caster; an `ObjectForm` outside True Polymorph. When several apply, the error MUST be `errors.Join` of the typed problems, every one, in a fixed order (errors-and-results.md FR-023, FR-024). `MustActiveTransformation` panics with that error and is for test literals only. | `doc/rules/architecture.md` §3, §6; `errors-and-results.md` FR-023 to FR-026 | unhappy (invalid input) |
| FR-055 | Every operation MUST be a pure function: equal inputs give equal results, inputs and returned slices are not mutated, and no operation panics; an operation returns an error only where this spec states `srderr.ErrInvalidInput`. | `doc/rules/architecture.md` §3; `doc/rules/testing.md` §4 | unhappy (determinism) |
| FR-056 | The events `ShapeShiftBegan`, `ShapeShiftFormChanged`, `ShapeShiftMadePermanent` and `ShapeShiftEnded` MUST form a closed `ShapeShiftEvent` interface embedding `combat.Event`, with an unexported marker method, with kinds `shape_shift_began`, `shape_shift_form_changed`, `shape_shift_made_permanent`, `shape_shift_ended`; `combatants` is `[target]`, or `[target, caster]` when they differ; they carry the spell, never display text, and are immutable and comparable by value. `ShapeShiftEnded` carries the cause, `RevertsToTrueForm` and `MemoryOfObjectFormLost`. Order within a result: ending events before beginning events; `ShapeShiftBegan` before `TemporaryHitPointsGained`; `TemporaryHitPointsEnded` before `ShapeShiftEnded`. | `combat-foundations.md` FR-015 to FR-019; `shared-types-and-ownership.md` FR-019 | happy |
| FR-057 | Layering: types live in package `combat`, are pure, and import only kernel packages and the published language of `magical-effects.md`, `hit-points-and-damage.md`, `content` (`Monster`, `ChallengeRating`, `CreatureType`, `CreatureSize`, `SpellID`) and `d20` (`Outcome`). No import of `character`, `toolbox`, `class-druid.md` types or Concentration types. Results are closed variant sets (an interface with an unexported marker method whose doc comment lists every variant) with operation-prefixed variants; no generic variant name is exported. The service is `ShapeShiftSpells`; the name `ShapeShifting` belongs to `magical-effects.md` and MUST NOT be defined here. | `doc/rules/architecture.md` §1, §2, §9; `shared-types-and-ownership.md` FR-013, FR-017, FR-021, FR-044; `combat-foundations.md` FR-024 | unhappy (forbidden edge) |
| FR-058 | Refusals are `<Operation>Refused(rejection)` with a closed rejection type (`combat-foundations.md` FR-023). `BeginShapeShiftRejection` is a closed variant set: `TargetIsDead`, `TargetNotWilling`, `ShapechangeNotSelf`, `FormIneligible(reasons)` (FR-016's list). `ChangeFormRejection` is a closed variant set: `TargetIsDead`, `FormChangeNotAllowed`, `NotTransformed`, `ActorNotCaster`, `FormIneligible(reasons)`. `EndShapeShiftRejection` is an enum: `CannotEndAtWill`, `TargetIsDead`. No other member exists; section 7's `srd5e.combat.result` values are exactly these names in snake_case. | `combat-foundations.md` FR-023, FR-029 to FR-035; `doc/rules/observability.md` §2 | happy |
| FR-062 | The host supplies the `CombatantID` an object turned into a creature will have; the transformation is keyed by it and ends `TargetDied` or `TargetDestroyed` under it. A creature turned into an object keeps its `CombatantID`. `ShapeShiftEvent.combatants` uses those ids. | SRD 5.2.1, "True Polymorph" (Object into Creature, Creature into Object), p. 170; package design (`combat-foundations.md` FR-025) | unhappy (state error) |

### Withdrawn ids

None.

## 5. Model

- **Referenced, not defined here:** `CombatantID`, `combat.Event` (`combat-foundations.md`); `HitPointStatus`,
  `TemporaryHitPointsChoice`, `TemporaryHitPointsGained`, `TemporaryHitPointsEnded`, `LifeState`
  (`hit-points-and-damage.md`); `EffectSource`, `SpellSource`, `CarryOverRule`, `CarryOverAll`
  (`magical-effects.md`); `Monster`, `ChallengeRating` (`content-monsters.md`); `CreatureType`, `CreatureSize`,
  `SpellID` (`content-schema-and-decoder.md`); `d20.Outcome` (`d20-tests.md`).
- **Value objects:** the types of section 3, plus `FormEligibility` (closed variant set: `FormEligible`, `FormIneligible`),
  `FormRestrictions`, `AnimatedObjectTerms`, and the commands `BeginShapeShiftCommand` (spell, caster, target,
  `TrueTransformation` or a `CreatureForm`, `TargetResponse`, `TargetRating`, the caster's rating for
  Shapechange, `HasSeenKind`, `ShapechangeTerms`, `TargetHitPoints` (closed variant set: `TrackedHitPoints(status, choice)`,
  `UntrackedHitPoints`), the Shapechange `EquipmentOutcome`, and the target's earlier `ActiveTransformation`
  or `NoTransformation`), `ChangeFormCommand`, `EndShapeShiftCommand`.
- **Domain service** (stateless; the zero value of the struct `ShapeShiftSpells` is ready to use, methods have value receivers; not `ShapeShifting`, which `magical-effects.md` owns) with `Begin`, `ChangeForm`, `Assess`, `ElapseFullDuration`,
  `End`, `RecordTemporaryHitPointsReplaced`, `StatisticsOf`, `RestrictionsOf`, `Eligibility`. It composes
  `HitPointStatus` operations and calls no use case.
- **Invariants** are enforced by `ActiveTransformation` (FR-054) and `RatedByLevel`/`CreatureForm`
  (`srderr.ErrInvalidInput`). **Failures:** expected outcomes are closed results (`...Resisted`, `...Refused(reason)`,
  `...Unaffected`); a refusal reason is a `<Subject>Rejection` held as a field (FR-058, `combat-foundations.md`
  FR-023). Programmer errors (FR-007, FR-008, FR-030, FR-036, FR-037, FR-054, FR-061) are errors satisfying `errors.Is(err, srderr.ErrInvalidInput)` from `NewX` constructors and `ChangeForm`; no `srderr.PortError` and no panic outside `MustX`.

## 6. Interface

- Synchronous pure methods over host-held values: no `ctx`, no goroutines, no port, no test double; safe for concurrent use.
- Package: exported from `combat` (`github.com/Whealinghold/srd5e-go/combat`); the package scan (check A7) finds no
  generic variant name, no second `ShapeShifting`, and none of `ConcentrationEffect`, `WildShape*`, `BeastFormFacts`.
- The host's loop: cast (`casting-and-concentration.md`) and roll the save (`spell-effects.md`), call
  `Begin`, then `ShiftShape` (`magical-effects.md`) with the returned `effect` and `CarryOver`; after each
  `ApplyDamage` call `Assess`; on `ConcentrationEnded`, a Dead target, a dispel or the clock call `End`.

## 7. Observability

Silent by design (`doc/rules/observability.md` §3): domain-only, no signal of its own. The use cases of the
`combat-action-resolution.md` amendment RA-1 record through `CombatTelemetry` (`combat-foundations.md` FR-029
to FR-035) with these results.

| Operation | Variant | `srd5e.combat.result` | Class |
|---|---|---|---|
| `begin_shape_shift` | `BeginShapeShiftApplied` | `completed` | applied |
| | `BeginShapeShiftResisted` | `shape_shift_resisted` | applied |
| | `BeginShapeShiftRefused` | the rejection name (`target_is_dead`, `target_not_willing`, `shapechange_not_self`, `form_ineligible`) | refused |
| `change_shape_shift_form` | `ChangeFormApplied` / `ChangeFormRefused` | `completed` / the rejection name (`target_is_dead`, `form_change_not_allowed`, `not_transformed`, `actor_not_caster`, `form_ineligible`) | applied / refused |
| `assess_shape_shift` | `AssessShapeShiftContinues` / `AssessShapeShiftEnded` | `completed` / `shape_shift_ended` | applied |
| `end_shape_shift` | `EndShapeShiftApplied` / `EndShapeShiftUnaffected` / `EndShapeShiftRefused` | `completed` / `shape_shift_unaffected` / the rejection name (`cannot_end_at_will`, `target_is_dead`) | applied / applied / refused |

Seven questions: (1) the host passes a wrong state or an ineligible form (`refused`), or forgets to call`Assess` after damage, leaving a Polymorph on after its Temporary Hit Points are gone. (2) By the counter of
`srd5e.combat.operations`; a stuck Polymorph is visible only as `assess_shape_shift` never being called
after `apply_damage`. (3) Results carry every ineligibility reason and the end cause. (4) Calls by operation
and result; events by `kind` (`shape_shift_ended` per cause is in the event). (5) No existing signal changes
meaning. (6) A sustained `form_ineligible` rate is a host form-picker bug; `end_shape_shift` `unaffected` on a
Dead target is a state bug. (7) No log: every outcome is a result or an event.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-005 | unit | Four members and ids; unknown id; table A against the bundled `Spell` entries (oracle: the bundled spell data); table B per row; the Shapechange no-limits row; Spellcasting retained and not held |
| FR-006 to FR-017 | unit | `NewFormFacts` (a Beast, a multi-size block, CR 0); table E1 per row; level 0 and 21 return errors; sizes outside the options return errors; all reasons in fixed order |
| FR-018 to FR-026 | unit | Table R per row; `ShiftShape` inputs (`SpellSource`, `CarryOverAll`) on every Applied; `MaintainedBy` per spell; replacement order of events |
| FR-027 to FR-035 | unit | Table T per row with the real `HitPointStatus` operations (`ApplyDamage`, `GainTemporary`, `EndTemporary`); `UntrackedHitPoints` accepted and rejected; Hit Points unchanged by `Assess` |
| FR-036 to FR-038 | unit | Equipment per spell and mode; Shapechange without a choice returns an error; no Concentration input on any operation |
| FR-039 to FR-042, FR-060, FR-061 | unit | Change on each spell; ineligible form leaves the state equal; permanent and ended states refuse; the Temporary Hit Points unchanged; other actor, Dead target and check order; equipment choice missing, invalid or supplied for Animal Shapes returns an error |
| FR-043 to FR-053 | unit | Table X per row; Dead target; no transformation; a permanent state through every cause |
| FR-054, FR-055 | unit | Each constructor violation; the joined error lists all; determinism; inputs not mutated; returned slices are copies |
| FR-058, FR-059, FR-062 | unit | Every rejection member and its snake_case name; equal-amount Temporary Hit Points (no event, `declined`, end leaves them); an object-turned-creature under the host's id and a creature-turned-object keeping its id |
| FR-056 | unit, architecture | Event kinds, `combatants` order, value equality, order of events in results; event family scan |
| FR-057 | architecture | Import direction; no generic or duplicate exported names |
| Scenario | scenario | A druid-level Polymorph through the public API (`scenarios/shape_shift_test.go`, package `scenarios_test`): cast, save fails, form gained 11 Temporary Hit Points, 15 damage ends the spell and costs 4 Hit Points, `RevertToTrueForm` returns the true form. A True Polymorph held for the full duration becomes permanent and a Dispel Magic result ends it |

No doubles or contract suites. Mutation testing (`doc/rules/testing.md` §8): all changed lines in `combat`,
score at least 80 (`gomu` only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`; not run by this spec); boundary rows kill `<`/`<=` mutants on Challenge Rating, size and Temporary Hit Points.

## 9. Open questions and decisions

No question is escalated. **Settled** (one line each; research summarised, nothing copied):

- Q-1 The form's "Hit Points" are the printed average of its stat block (interpretation; SRD p. 255 gives the average as the standard value and `content-monsters.md` stores it).
- Q-2 Polymorph ends the moment Temporary Hit Points are 0 and the excess already reduced the retained Hit Points (FR-032, FR-033) (interpretation: the SRD wording "Lose Temporary Hit Points First", p. 18, supports this, and community discussion agrees).
- Q-3 Dropping to 0 Hit Points does not itself end Shapechange, Animal Shapes or True Polymorph (interpretation): their text ends them by death or destruction (p. 170) or never, and Shape-Shifting reverts the creature only if it dies (p. 187); the target stays shifted and Unconscious per `hit-points-and-damage.md`. Polymorph cannot reach 0 first (FR-032).
- Q-4 A second shape-shifting spell replaces the first (FR-026): one true form (`magical-effects.md` Q-13); the earlier spell "ends", so its form Temporary Hit Points vanish first.
- Q-5 "Level" is total character level, 1 to 20 (FR-008), because the SRD calls Multiclassing's sum the character level (p. 25); monsters always have a Challenge Rating.
- Q-6 Shapechange carries no anatomy, speech or casting limit (FR-004): the SRD text states none and retains communication and Spellcasting; adding one would be an invented rule.
- Q-7 Object statistics, an object's Hit Points and what becomes of a creature whose object form is destroyed stay with the host (`environment-hazards-and-objects.md`); True Polymorph's text says only "dies or is destroyed". The SRD applies the creature size categories to objects, so `CreatureSize` is used.
- Q-8 True Polymorph's full text is on p. 170; p. 171 has none of it. Animal Shapes' continuation sits in the second column of p. 108.
- Q-9 Animal Shapes' Temporary Hit Points vanish for a target when its own transformation ends (FR-034, FR-050), because the spell is applied per target.
- Q-10 Wild Shape and these spells never coexist on one creature: a host that polymorphs a Wild Shaped Druid ends Wild Shape first with `class-druid.md` reason `LeftEarly`. This is a host precondition, not enforced here: `Begin` takes no Wild Shape input and is unaffected by its state. The SRD does not rank them, and Q-4 gives the same single-form answer.
- Q-11 Shape-shifting does not break Concentration on another spell (FR-038): no rule in these entries does, and ongoing spells carry over (p. 187); the `StatBlockShapechange` duration stays 1 hour.
- Q-12 A `Willing` Polymorph target gives up its save without rolling (FR-018): "you can choose to fail the save without rolling" (p. 7).

**Rows other specs must carry (not yet present when this was written):** `shape-changing-spells` in the
rank 28 row of `shared-types-and-ownership.md` 4.11 (beside `encounter-aggregate` and
`dispel-and-counterspell`); an entry in `documentation_index.md`; RA-1 in `combat-action-resolution.md`;
pointers in `magical-effects.md` (section 1 out of scope, 4.8 intro, noting this spec's service is
`ShapeShiftSpells`) and in `architecture-guardrails.md` Depends on.

## 10. Changelog impact

- **Added:** the four shape-changing spells' behaviour in `combat` (form eligibility, kept and replaced
  statistics, Temporary Hit Points, ending and True Polymorph permanence;
  `doc/specs/shape-changing-spells.md`).
- **BREAKING:** no. Everything is new.
