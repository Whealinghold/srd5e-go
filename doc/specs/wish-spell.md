# Wish

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `ability-scores.md`, `observability-ports.md`, `content-schema-and-decoder.md`, `content-spells.md`, `content-equipment.md`, `character-aggregate.md`, `rests-and-hit-dice.md`, `spellcasting.md`, `casting-and-concentration.md`, `feats-behaviour.md` |

Dependency notes (prose only; shared-types-and-ownership.md FR-056): all types named here live in package `character` (`github.com/Whealinghold/srd5e-go/character`). This spec has rank 16 (it imports `feats-behaviour.md`, rank 15) and is listed in the rank table 4.11 and in `documentation_index.md`. `hit-points-and-damage.md` is not listed: it is `combat`, downstream of `character` (shared-types-and-ownership.md FR-013), so the Wish damage leaves as a plain handoff value (FR-044) that `combat` accepts (hit-points-and-damage.md FR-114). `d20-tests.md` is not listed: Roll Redo hands the host a `modifier.RollMode` and a promise to roll (FR-013), it rerolls nothing here. `class-cleric.md` is not listed: a Cleric's Greater Divine Intervention reaches Wish through the host's `CastSpell` (class-cleric.md FR-094), as any other cast.

## 1. Purpose and scope

The Wish spell (SRD 5.2.1, "Wish", p. 175; Level 9 Conjuration, Sorcerer and Wizard, Action, Self, V) after it has been cast: the choice among duplicating a spell of level 8 or lower and the seven listed effects (Object Creation, Instant Health, Resistance, Spell Immunity, Sudden Learning, Roll Redo, Reshape Reality), their limits as checks, and **the stress** of any effect other than duplication: Necrotic damage on each later cast until a Long Rest, Strength 3 for 2d4 days with faster recovery for resting days, and a 33 percent chance of never casting Wish again. The stress is persistent character state (a `FeatureState`, shared-types-and-ownership.md FR-022) plus pure transitions, one dice-using use case per roll (shared-types-and-ownership.md FR-023), and a pure effective-score function so no stored ability score is ever rewritten. The two use cases are the methods `ResolveWish` and `RollWishStressDamage` of one small type, `WishResolver`, which holds only the injected dice and telemetry ports; everything else is a pure function.

Casting Wish (a level 9 slot, a Cleric's Divine Intervention, a magic item) is `CastSpell` and its siblings (casting-and-concentration.md); this spec starts where the cast ends and holds no cast state. The effects that act on creatures other than the caster, or on the world, are **handoffs** (structured facts for the host or `combat`); only Sudden Learning changes the caster's own record, through the published `FeatReplacement` (feats-behaviour.md FR-082).

**Out of scope:**

- **Reshape Reality's outcome.** The SRD gives the Game Master "great latitude"; the package carries the player's statement as an opaque value and decides nothing (FR-014). So are the consequences the text lists (attracting powerful foes, divine intervention, the wish failing, partial results, an unforeseen consequence) and "a Legendary magic item or an Artifact" transport: Game Master play, not rules.
- Applying the handoffs: Hit Points, conditions, Resistance, Immunity, objects in the world (`combat`, host).
- Casting Wish from an item (Ring of Three Wishes, Luck Blade, Deck of Many Things; magic-item-rules.md) or from a djinni or efreeti (monster Wish trait): they bypass `CastSpell`, which is why `ResolveWish` takes no cast event (FR-001). The Wish spell's own data is content (content-spells.md).
- Ending Wish stress by other means (a Wish that heals it, Greater Restoration on the Strength change): the SRD states none; the host may write the state back.
- Optional rules: none in the SRD for Wish, so no `RuleOptions`.

## 2. Source

PDF page numbers equal printed page numbers in SRD 5.2.1.

- SRD 5.2.1, "Wish" (spell text, all effects and the stress), p. 175; the Sorcerer spell list, level 9, p. 69; the Wizard spell list, level 9, p. 82.
- SRD 5.2.1, Cleric, "Level 20: Greater Divine Intervention" (Wish and 2d4 Long Rests), p. 38 (handoff only).
- SRD 5.2.1, "Greater Restoration" (the list of effects Instant Health ends), p. 137.
- SRD 5.2.1, "Spells", "Spell Level" and "Using a Higher-Level Spell Slot" ("the spell takes on the higher level for that casting"), p. 104; "Combining Spell Effects" (same spell cast several times: the most potent effect applies, the most recent if equally potent), p. 106.
- SRD 5.2.1, "Long Rest", p. 185; "Damage", "Resistance", "Immunity", "Temporary Hit Points" (Hit Points section), pp. 17-18.
- Package design: `doc/rules/architecture.md` §3, §4, §6; `doc/rules/observability.md` §2, §3, §7; `doc/rules/testing.md` §4; shared-types-and-ownership.md FR-017, FR-022, FR-023, FR-044.

Web research (used only to judge the readings of section 9, none copied): the community reads "level of that spell" as the level the spell has for that casting, a resting day as taking 2 days off the total (FR-050), and a duplicating Wish as free of stress but not of the cast damage.

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Wish | `WishSpellID()` (`SpellID`), `WishChoice` (closed) | One variant per option of the text (FR-002 to FR-014) |
| duplicate any other spell of level 8 or lower | `DuplicateSpell` | No requirements, "including costly components" (FR-003) |
| Object Creation, Instant Health, Resistance, Spell Immunity, Sudden Learning, Roll Redo, Reshape Reality | `ObjectCreation`, `InstantHealth`, `ResistanceGrant`, `SpellImmunityGrant`, `SuddenLearning`, `RollRedo`, `ReshapeReality` | Variants of `WishChoice` |
| the stress | `WishStress` (a `FeatureState`), `WishStressRoll` | Section 5 |
| 1d10 Necrotic damage per level of that spell | `UnpreventableDamage`, `WishResolver.RollWishStressDamage` | "can't be reduced or prevented in any way" (FR-041 to FR-044) |
| Strength score becomes 3 for 2d4 days | `WishWeakness`, `WishEffectiveScores` | The stored score is untouched (FR-030) |
| unable to cast Wish ever again | `WishCastingAbility` (`CanCastWish`, `CannotCastWishAgain`) | FR-025, FR-031 |
| finish a Long Rest | `RestApplied` with `BenefitsOf()` `longRest` | rests-and-hit-dice.md FR-036, FR-061 |
| resting and doing nothing more than light activity | `WishRecoveryDay.restingWithLightActivity` | Reported by the host (FR-050) |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3). Enum values are written by their short name (`longRest`, `spellLevelTooHigh`); the Go constant carries the type prefix and `String()` returns the short name. Closed families are interfaces with an unexported marker method (`//sumtype:decl`) whose doc comment lists every variant, consumed by a type switch without a `default`.

## 4. Requirements

Stable ids; never renumber. Ids are allocated in blocks; an id absent from this section and not in the Withdrawn ids was never used by this spec. Source cites are by section and printed page. A bare `FR-nnn` is this spec's; another spec's id is written with its file name. Case tables follow the list.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `WishResolver.ResolveWish(ctx context.Context, cmd ResolveWishCommand) (ResolveWishResult, error)` MUST take a command holding the `Character` (its `FeatureStates`) and a `WishChoice` that carries every fact of the choice; it MUST NOT take, check or require a cast event or a slot, because Wish may be cast through a slot, a feature or an item (section 1). `WishSpellID()` MUST return `SpellID("wish")` and `WishLevel()` MUST return `SpellLevel` 9. `ctx` is passed to the telemetry only and is never stored. | SRD 5.2.1, "Wish", p. 175; shared-types-and-ownership.md FR-007 | happy |
| FR-002 | `DuplicateSpell(Spell)` MUST be accepted for a spell of level 0 to 8 and MUST be rejected with `spellLevelTooHigh` for level 9 (which includes Wish itself). | SRD 5.2.1, "Wish" ("any other spell of level 8 or lower"), p. 175 | unhappy (boundary) |
| FR-003 | A duplication MUST apply no list, preparation, known, Ritual, slot, component, armor or casting-time check: the caster "doesn't need to meet any requirements to cast that spell, including costly components" and "the spell simply takes effect". A spell not on any of the caster's class lists MUST be accepted. | SRD 5.2.1, "Wish", p. 175 | unhappy (rule interaction) |
| FR-004 | An accepted duplication MUST return `ResolveWishDuplicated` with the handoff `WishDuplicate(spellID, spellLevel, requiresConcentration)` and the one event `WishDuplicated`; it MUST roll no dice, leave the `Character` equal and add no stress. The host resolves the duplicate through the cast flow of casting-and-concentration.md at the spell's own level (`CastMode.duplicated`, casting-and-concentration.md FR-150); Concentration begins when `requiresConcentration`. When the host later asks for stress damage it marks the Wish cast with `WishCastOrigin` (FR-069). | SRD 5.2.1, "Wish", p. 175; "Concentration", p. 106 | happy |
| FR-005 | `ObjectCreation` MUST be rejected with `objectValueTooHigh` when `Cost.ValueInCopper()` exceeds 25,000 GP (2,500,000 Copper Pieces) and accepted at exactly 25,000 GP. A `Cost` with a `per` rate (Day, Mile) is not a one-off value: `NewObjectCreation` returns an `*InvalidError` (FR-068). | SRD 5.2.1, "Wish" (Object Creation), p. 175; "Coins", p. 89 | unhappy (boundary) |
| FR-006 | `ObjectCreation` MUST be rejected with `objectIsMagicItem` when the caller says the object is a magic item ("that isn't a magic item"). | SRD 5.2.1, "Wish" (Object Creation), p. 175 | unhappy (state error) |
| FR-007 | `ObjectCreation` MUST be rejected with `objectTooLarge` when its largest dimension exceeds 300 feet and accepted at 300; a dimension of zero or less, NaN or infinite MUST make `NewObjectCreation` return an `*InvalidError` (it matches `srderr.ErrInvalidInput`; `MustObjectCreation` panics with it). Placement ("an unoccupied space that you can see on the ground") is the host's. | SRD 5.2.1, "Wish" (Object Creation), p. 175 | unhappy (boundary) |
| FR-008 | `InstantHealth(OtherCreatures)` MUST accept 0 to 20 other creatures (the caster is always included: "yourself and up to twenty creatures") and MUST be rejected with `tooManyCreatures` above 20; a negative count MUST make `NewInstantHealth` return an `*InvalidError`. Its handoff MUST say "regain all Hit Points" and name Greater Restoration (`SpellID('greater_restoration')`) as the list of effects to end; it MUST NOT carry a Hit Point amount. | SRD 5.2.1, "Wish" (Instant Health), p. 175; "Greater Restoration", p. 137 | unhappy (boundary) |
| FR-009 | `ResistanceGrant(DamageType, creatures)` MUST accept 1 to 10 creatures and one `DamageType`, be rejected with `tooManyCreatures` above 10, and `NewResistanceGrant` MUST return an `*InvalidError` for 0 creatures or a creature list with a repeated entry (here and in FR-010). Its handoff MUST be marked permanent. | SRD 5.2.1, "Wish" (Resistance), p. 175 | unhappy (boundary) |
| FR-010 | `SpellImmunityGrant(subject, creatures)` MUST accept 1 to 10 creatures, be rejected with `tooManyCreatures` above 10, and `NewSpellImmunityGrant` MUST return an `*InvalidError` for 0 creatures (as FR-009). The subject is a closed family (`SpellImmunitySubject`): `ImmuneToSpell(SpellID)` or `ImmuneToOtherEffect(WishStatement)` ("a single spell or other magical effect"). The handoff duration MUST be 8 hours (`DurationHours()` 8). | SRD 5.2.1, "Wish" (Spell Immunity), p. 175 | unhappy (boundary) |
| FR-011 | `SuddenLearning` MUST replace a held feat through `ReplaceFeat`, with the grant scope `AnyFeat` and the **feat source of the replaced feat** (read from `CharacterFeats`), so a class spec's "owed minus taken" count of feat grants is unchanged. Every rejection of that service (`FeatNotHeld`, `FeatNotReplaceable`, a failed prerequisite or choice) MUST be returned unchanged as the nested `FeatRejection` of the rejection kind `featReplacementRejected` (a result, not an error). Interpretation inherited from feats-behaviour.md FR-082 and Q20, not stated by the SRD: a feat that gave an ability score increase is `FeatNotReplaceable`, because that spec has no way to reverse an increase. Cost: the SRD's "lose all the benefits of the old feat" would allow replacing such a feat; amendment A8 asks feats-behaviour.md to return a handoff that reverses the increase (`ChangeAbilityScore` exists), after which E21 changes. | SRD 5.2.1, "Wish" (Sudden Learning: "another feat for which you are eligible"), p. 175; feats-behaviour.md FR-082 | unhappy (rule interaction) |
| FR-012 | `SuddenLearning` MUST be rejected with `featRequiredByAnother` when the old feat id is in the caller's `FeatsRequiredByOthers` (feats or features that name it as a prerequisite), before the service is called but after the held check: an old feat that is not held reports `featReplacementRejected(FeatNotHeld)` first, so the order is held, required-by-another, then the service's other checks. Every entry of `FeatsRequiredByOthers` MUST be a held feat id, else `ResolveWish` returns an `*InvalidError` (before any die is rolled or signal emitted). Success returns the new `Character`, `CharacterFeats` and the events of the service (`FeatReplaced`, then its others). | SRD 5.2.1, "Wish" (Sudden Learning: "can't replace a feat that is a prerequisite for any of your other feats or features"), p. 175; feats-behaviour.md FR-082 | unhappy (state error) |
| FR-013 | `RollRedo(modifier.RollMode, RollWithinLastRound)` MUST accept `normal`, `advantage` and `disadvantage` and MUST be rejected with `rollNotRecent` when the caller reports the roll was not made "within the last round (including your last turn)". The handoff MUST carry the `modifier.RollMode` and say the caster chooses between the reroll and the original; the package rerolls nothing and rolls no dice here. | SRD 5.2.1, "Wish" (Roll Redo), p. 175 | unhappy (state error) |
| FR-014 | `ReshapeReality(WishStatement)` MUST accept any non-empty statement and return the handoff `WishReshapeReality(statement)` for the Game Master; the package MUST NOT interpret it, rule on it or reject it for its content. `NewWishStatement(s string) (WishStatement, error)` MUST return an `*InvalidError` for an empty or whitespace-only value, and for one over 2,000 characters (a package sanity bound). | SRD 5.2.1, "Wish" (Reshape Reality: "state your wish to the GM"), p. 175 | unhappy (invalid input) |
| FR-020 | Every accepted choice other than `DuplicateSpell` (Object Creation, Instant Health, Resistance, Spell Immunity, Sudden Learning, Roll Redo and Reshape Reality) MUST incur the stress; a `DuplicateSpell` MUST NOT. Interpretation: Reshape Reality incurs it even if the Game Master rules the wish fails, because the SRD ties the stress to casting Wish "to produce any effect other than duplicating". | SRD 5.2.1, "Wish" (the stress), p. 175 | unhappy (rule interaction) |
| FR-021 | A rejected choice MUST incur no stress, roll no dice and leave the `Character` equal; rejections are checked before any die is rolled. | `doc/rules/architecture.md` §4, §6; SRD 5.2.1, "Wish", p. 175 | unhappy (state error) |
| FR-022 | Incurring the stress MUST roll, in this order through `dice.Roll`, `2d4` (the days) and `1d100` (the lockout). The lockout applies when the percentile is 1 to 33 and not when it is 34 to 100 ("33 percent chance"). The two values are the `WishStressRoll` (`NewWishStressRoll(days, percentile int) (WishStressRoll, error)`) the pure rules take (days 2 to 8, percentile 1 to 100, else an `*InvalidError`). | SRD 5.2.1, "Wish" (the stress), p. 175; dice-and-rolls.md FR-050 | unhappy (boundary) |
| FR-023 | The stress MUST set `WishWeakness` to `WishWeakened(DaysRemaining)` with the rolled days. When already weakened, `DaysRemaining` MUST become the new roll, even if shorter than the old value: the two Strength 3 effects are equally potent, and "the most recent effect applies if the castings are equally potent and their durations overlap". Interpretation: that rule is written for the same spell cast several times, and the stress is the second casting's effect. | SRD 5.2.1, "Wish" (Strength 3), p. 175; "Combining Spell Effects", p. 106 | unhappy (rule interaction) |
| FR-024 | The stress MUST set `WishCastDamage` to `WishCastDamageActive`, the state in which each later cast costs Necrotic damage (FR-041), until a finished Long Rest (FR-052). Already active stays active. | SRD 5.2.1, "Wish" (the stress), p. 175 | happy |
| FR-025 | A percentile of 33 or less MUST set `WishCastingAbility` to `CannotCastWishAgain`, which is permanent: no transition in this spec returns to `CanCastWish`, and a later stress with a percentile above 33 MUST NOT clear it. | SRD 5.2.1, "Wish" ("unable to cast Wish ever again"), p. 175 | unhappy (state error) |
| FR-026 | The stress MUST emit the event `WishStressEndured(days, percentile, lockedOut)` once, after the events of the effect itself (FR-012 included), so a host reads cause before consequence. | `doc/rules/architecture.md` §3 (domain events) | happy |
| FR-030 | `WishEffectiveScores(scores ability.Scores, stress WishStress) ability.Scores` MUST return the scores with Strength exactly 3 (whatever the stored value: 1, 2, 3, 4 or more) while `WishWeakened` and the same scores otherwise; the other five abilities and the stored scores MUST NOT change, and `ChangeAbilityScore` (character-aggregate.md) is not involved. The text says the score "becomes 3", not "is capped at 3": a set, not a cap (literal reading; see section 9). | SRD 5.2.1, "Wish" ("becomes 3"), p. 175 | unhappy (boundary) |
| FR-031 | `WishStress.CanCastWish() bool` MUST be false exactly for `CannotCastWishAgain`. `ResolveWish` MUST reject with `cannotCastWishAgain` before any other check and before any die; this is the enforcement in this package. A host SHOULD also refuse the cast itself (`CastingContext.unavailableSpells`, casting-and-concentration.md FR-146). Interpretation: the SRD says the creature is "unable to cast Wish ever again" and names no route, so the lockout is tied to the creature and covers every route (slot, Cleric feature, item); the stress is the creature's, not the source's. | SRD 5.2.1, "Wish", p. 175 | unhappy (state error) |
| FR-040 | `WishLevelOfCast(cast SpellCast, spell Spell) SpellLevel` MUST return the `SpellCast.CastAtLevel()` when present (a slot cast, or a feature-granted spell of level 1 or higher), otherwise the spell's own level (a Ritual, and 0 for a cantrip). "Level of that spell" is the level the spell takes "for that casting". | SRD 5.2.1, "Spells", "Using a Higher-Level Spell Slot", p. 104; "Wish", p. 175 | unhappy (rule interaction) |
| FR-041 | `RollWishStressDamage` MUST, while `WishCastDamageActive`, roll `<level>d10` through `dice.Roll` (a `dice.Expression` of `level` d10) for a cast of level 1 to 9 and return `RollWishStressDamageRolled` with an `UnpreventableDamage(DamageType.necrotic, amount)`. | SRD 5.2.1, "Wish" (the stress), p. 175 | happy |
| FR-042 | A cast of level 0 MUST return `RollWishStressDamageNone(cantrip)` without rolling (no die count of zero exists); while `WishCastDamageInactive` it MUST return `RollWishStressDamageNone(notActive)` without rolling. | SRD 5.2.1, "Wish" ("1d10 ... per level"), p. 175; dice-and-rolls.md FR-050 | unhappy (boundary) |
| FR-043 | A command whose `WishCastOrigin` is `wishDuplicate` (FR-069) MUST return `RollWishStressDamageNone(duplicatedSpell)`: the duplicate is the effect of the Wish cast, which the host passes in its own right (level 9, so 9d10 while active). Interpretation: one casting, one charge. | SRD 5.2.1, "Wish" ("each time you cast a spell"), p. 175 | unhappy (rule interaction) |
| FR-044 | `UnpreventableDamage(type, amount)` MUST be an exported value object, amount 1 or more, that a damage resolver applies without Resistance, Immunity or any flat reduction ("can't be reduced or prevented in any way"). It is a handoff; this spec applies no damage and `RollWishStressDamage` returns the `Character` equal. Whether Temporary Hit Points absorb it is `combat`'s (hit-points-and-damage.md FR-114). | SRD 5.2.1, "Wish", p. 175; "Temporary Hit Points", p. 18 | happy |
| FR-050 | `WishAfterDay(id CharacterID, stress WishStress, day WishRecoveryDay) WishTransition` MUST take 1 off `DaysRemaining` for `ordinary` and `WishRestingDayReduction` (the named constant, 2) for `restingWithLightActivity`, never below 0; reaching 0 MUST set `NoWishWeakness` and emit `WishWeaknessEnded`. `WishTransition` holds the new `WishStress` and the events (a copy on read); the `CharacterID` is a parameter because every `WishEvent` carries it (FR-061). Interpretation: the text says the remaining time "decreases by 2 days" for each such day and names no extra day for the day passing, so a resting day takes 2 in all; the rejected alternative is 1 for the day plus 2 (3 in all), see section 9. | SRD 5.2.1, "Wish" ("remaining recovery time decreases by 2 days"), p. 175 | unhappy (boundary) |
| FR-051 | `WishAfterDay` with `NoWishWeakness` MUST return an equal state and no events. | `doc/rules/testing.md` §4 (empty and minimal) | unhappy (empty and minimal) |
| FR-052 | `WishAfterRest(id CharacterID, stress WishStress, rest RestApplied) (WishTransition, error)` MUST set `WishCastDamageInactive` and emit `WishCastDamageEnded` when `BenefitsOf()` is `longRest`, and MUST leave the state equal for `shortRest` (including the `RestApplied(shortRest)` of an interrupted Long Rest). It MUST NOT change the weakness or the lockout. | SRD 5.2.1, "Wish" ("until you finish a Long Rest"), p. 175; "Long Rest", p. 185; rests-and-hit-dice.md FR-061 | unhappy (rule interaction) |
| FR-053 | A Long Rest MUST NOT count as a recovery day: days are reported only through `WishAfterDay`. | SRD 5.2.1, "Wish" (days), p. 175 | unhappy (rule interaction) |
| FR-060 | The state MUST be stored under the `FeatureStateKey` for `WishStress` named `wish_stress`, returned by `WishStressKey()` (character-aggregate.md FR-093). An absent state MUST read as the zero `WishStress` (`NoWishWeakness`, `WishCastDamageInactive`, `CanCastWish`; a nil part reads as its first variant). `WishStress` and its parts MUST be immutable and compare by value (`==`). | shared-types-and-ownership.md FR-022; character-aggregate.md FR-092 | happy |
| FR-061 | The events `WishDuplicated`, `WishStressEndured`, `WishWeaknessEnded`, `WishCastDamageEnded` and `WishStressDamageRolled` MUST be members of the closed family `WishEvent`, which implements `CharacterEvent` directly and carries the `CharacterID` (character-aggregate.md FR-085, FR-120). | shared-types-and-ownership.md FR-019; character-aggregate.md FR-085 | happy |
| FR-062 | A `RollWishStressDamage` command whose `SpellCast.CharacterID()` differs from the `Character`'s id, or whose `Spell` id differs from the cast's spell, MUST make `RollWishStressDamage` return an `*InvalidError` (before any die or signal); so MUST a `RestApplied` for another character given to `WishAfterRest`, a `CastAtLevel` below the spell's own level, and a `CastAtLevel` on a cantrip (FR-068). | `doc/rules/architecture.md` §6 (programmer errors) | unhappy (invalid input) |
| FR-063 | With a fixed roller (`dicetest.FixedRoller`), both use cases MUST return equal results for equal commands, and no domain function reads a clock or a random source. | `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-064 | A failure of the dice port MUST be returned from both use cases as the `*srderr.PortError` (port `dice`, produced once by `dice.Roll`, matching the original cause with `errors.Is` / `errors.As`; a roller panic matches `srderr.ErrPortPanic` and a result for another expression matches `srderr.ErrContractViolation`), with a zero result, the `Character` unchanged, no event, and the first roll not applied if the second fails. | dice-and-rolls.md FR-052; errors-and-results.md FR-031, FR-036; `doc/rules/observability.md` §7 | unhappy (collaborator failure) |
| FR-065 | No operation MUST alter its inputs; collections in results and events (events, creature lists) MUST be copies on construction and on read, so mutating one leaves the value unchanged. | `doc/rules/architecture.md` §3 | unhappy (immutability) |
| FR-066 | Each use case MUST record its own signals once (section 7); the domain MUST record nothing. | `doc/rules/observability.md` §3; shared-types-and-ownership.md FR-044 | unhappy (duplicate signal) |
| FR-067 | Every type a command or result names MUST be exported from package `character` (no facade or re-export package) and defined once in section 5; no name is generic (shared-types-and-ownership.md FR-017). | `doc/rules/architecture.md` §4, §7 | happy |
| FR-068 | Invalid input (an `*InvalidError`, matching `srderr.ErrInvalidInput`, returned by the constructor or use case; the `MustX` helpers panic with it) in choice and command construction: an `ObjectCreation` `Cost` with a `per` rate; a non-finite or non-positive `LargestDimensionFeet`; a negative `OtherCreatures`; 0 or repeated creatures (FR-009, FR-010); a `RollWishStressDamage` command whose `CastAtLevel` is below the spell's level or present on a cantrip. A level 1+ spell with no `CastAtLevel` (Ritual) uses the spell's own level (FR-040). | `doc/rules/architecture.md` §6; `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-069 | `RollWishStressDamage` MUST take a `WishCastOrigin` (a typed enum: `ordinaryCast` or `wishDuplicate`) that the host sets to `wishDuplicate` for the cast of a spell duplicated by Wish; this is a plain input of this spec, used only to decide stress damage (FR-043), and independent of `CastMode.duplicated` / `CastSource.duplicated` (casting-and-concentration.md FR-150, spellcasting.md FR-149), which the host uses for the duplicate's own cast (FR-004). | SRD 5.2.1, "Wish", p. 175; shared-types-and-ownership.md FR-007 | happy |

### Withdrawn ids

None.

### Case tables

Each row becomes one table-driven test. Baseline: a Wizard 17 with no stress, Strength 14, `FeatureStates` empty, a fixed roller.

#### Table E: effect checks (FR-002 to FR-014, FR-020, FR-021)

| Label | Choice | Result | FR | Path |
|---|---|---|---|---|
| E1 | `DuplicateSpell`, level 8 spell | `ResolveWishDuplicated`; no dice, no stress | FR-002, FR-004 | unhappy (boundary) |
| E2 | `DuplicateSpell`, level 9 spell (Wish itself) | rejected `spellLevelTooHigh` | FR-002 | unhappy (boundary) |
| E3 | `DuplicateSpell`, cantrip | `ResolveWishDuplicated`, level 0 | FR-002 | unhappy (boundary) |
| E4 | `DuplicateSpell`, spell on no Wizard list, Material cost, 1-hour casting time | accepted; handoff only | FR-003 | unhappy (rule interaction) |
| E5 | `DuplicateSpell`, spell that needs Concentration | handoff `requiresConcentration` true | FR-004 | happy |
| E6, E7 | `ObjectCreation` 25,000 GP; 25,001 GP | stress applied; rejected `objectValueTooHigh` | FR-005 | unhappy (boundary) |
| E8 | `ObjectCreation` a magic item | rejected `objectIsMagicItem` | FR-006 | unhappy (state error) |
| E9, E10, E11 | largest dimension 300 ft; 301 ft; 0 ft | accepted; `objectTooLarge`; `*InvalidError` from `NewObjectCreation` | FR-007 | unhappy (boundary) |
| E12, E13 | `InstantHealth` 20 others; 21 | accepted (21 healed with the caster); `tooManyCreatures` | FR-008 | unhappy (boundary) |
| E14, E15, E16 | `ResistanceGrant` 10 creatures; 11; 0 | accepted; `tooManyCreatures`; `*InvalidError` | FR-009 | unhappy (boundary) |
| E17 | `SpellImmunityGrant` 10 creatures, `ImmuneToSpell` | handoff duration 8 hours | FR-010 | happy |
| E18 | `SpellImmunityGrant` `ImmuneToOtherEffect`, 11 creatures | `tooManyCreatures` | FR-010 | unhappy (boundary) |
| E19 | `SuddenLearning`, Alert replaced by Lucky, old feat held, eligible | `FeatReplaced`, then stress; source of the old feat kept | FR-011 | happy |
| E20 | `SuddenLearning`, old feat not held | rejected `featReplacementRejected(FeatNotHeld)`; no dice | FR-011, FR-021 | unhappy (state error) |
| E21 | `SuddenLearning`, old feat gave an ability score increase | `featReplacementRejected(FeatNotReplaceable)` | FR-011 | unhappy (rule interaction) |
| E22 | `SuddenLearning`, old feat in `FeatsRequiredByOthers` | rejected `featRequiredByAnother` | FR-012 | unhappy (state error) |
| E23 | `SuddenLearning`, new feat prerequisite unmet | `featReplacementRejected(FeatPrerequisiteNotMet)` | FR-011 | unhappy (invalid input) |
| E24, E25, E26 | `RollRedo` normal; advantage; disadvantage, roll in the last round | handoff carries the mode | FR-013 | happy |
| E27 | `RollRedo`, roll made 2 rounds ago | rejected `rollNotRecent` | FR-013 | unhappy (state error) |
| E28 | `ReshapeReality("...")` | handoff; stress applied (also if a GM later refuses) | FR-014, FR-020 | happy |
| E29, E30, E31 | statement `""`; `"   "`; 2,001 characters | `*InvalidError` | FR-014 | unhappy (invalid input) |
| E32 | any rejected choice (E2, E7, E22) | `Character` equal, dice roller never called | FR-021 | unhappy (state error) |
| E33 | `CannotCastWishAgain` state, any choice including E1 | rejected `cannotCastWishAgain` first | FR-031 | unhappy (state error) |
| E22b | `SuddenLearning`, old feat both not held and in `FeatsRequiredByOthers` | `featReplacementRejected(FeatNotHeld)`; a required id not held is `*InvalidError` | FR-012 | unhappy (state error) |
| E34, E35, E36, E37 | `ObjectCreation` with `Cost` per Day; dimension NaN; `InstantHealth` -1 others; `SpellImmunityGrant` 0 creatures or repeated creature | `*InvalidError` | FR-068 | unhappy (invalid input) |

#### Table S: the stress (FR-022 to FR-026, FR-030, FR-060)

Fixed rolls are `(2d4, 1d100)` (a `dicetest.FixedRoller` script).

| Label | Prior state | Rolls | New state | Events after the effect | FR | Path |
|---|---|---|---|---|---|---|
| S1 | none | (2d4 = 2, 34) | `WishWeakened(2)`, `WishCastDamageActive`, `CanCastWish` | `WishStressEndured(2, 34, false)` | FR-022, FR-023, FR-024 | unhappy (boundary: shortest, 34 safe) |
| S2 | none | (8, 33) | `WishWeakened(8)`, `CannotCastWishAgain` | `WishStressEndured(8, 33, true)` | FR-022, FR-025 | unhappy (boundary: longest, 33 locks) |
| S3 | none | (5, 1) | locked out | lockedOut true | FR-025 | unhappy (boundary) |
| S4 | none | (5, 100) | `CanCastWish` | lockedOut false | FR-022 | unhappy (boundary) |
| S5 | `WishWeakened(6)` | (3, 50) | `WishWeakened(3)` (newest roll governs) | `WishStressEndured` | FR-023 | unhappy (rule interaction) |
| S6 | `WishWeakened(2)` | (7, 50) | `WishWeakened(7)` | as above | FR-023 | unhappy (rule interaction) |
| S7 | `CannotCastWishAgain` impossible via the use case | n/a | the lockout survives a transition built with percentile 90 | none | FR-025 | unhappy (state error) |
| S8 | none | (0, 50) or (9, 50) or (5, 0) or (5, 101) | `*InvalidError` from `NewWishStressRoll` | none | FR-022 | unhappy (invalid input) |
| S9 | none, `DuplicateSpell` | not rolled | state equal | `WishDuplicated` only | FR-020 | unhappy (rule interaction) |
| S10 | roller fails on the second roll | (4, error) | `*srderr.PortError` (port `dice`, `errors.Is` the roller's error); state equal | none | FR-064 | unhappy (collaborator failure) |
| S11 | stress with `SuddenLearning` | (4, 90) | events: `FeatReplaced`, then `WishStressEndured` | order | FR-026 | happy |

Effective Strength (FR-030): stored 14 weakened gives 3; stored 4 gives 3; stored 3 gives 3; stored 2 gives 3; stored 1 gives 3; stored 14 not weakened gives 14; the other five abilities are equal in every case; the input `AbilityScores` is unchanged.

#### Table D: cast damage, recovery and rest (FR-040 to FR-053)

| Label | Case | Result | FR | Path |
|---|---|---|---|---|
| D1 | active, level 3 slot spell, roll 3d10 = 17 | `RollWishStressDamageRolled(UnpreventableDamage(necrotic, 17))` | FR-041 | happy |
| D2 | active, level 1 spell cast with a level 3 slot (`CastAtLevel` 3) | 3d10 | FR-040 | unhappy (rule interaction) |
| D3 | active, level 9 slot spell (Wish) | 9d10 | FR-041 | unhappy (boundary) |
| D4 | active, Ritual of a level 2 spell (`CastAtLevel` absent) | 2d10 | FR-040 | unhappy (rule interaction) |
| D5 | active, feature-granted level 5 spell | 5d10 | FR-040 | unhappy (rule interaction) |
| D6 | active, cantrip | `RollWishStressDamageNone(cantrip)`, roller never called | FR-042 | unhappy (boundary) |
| D7 | inactive, level 4 spell | `RollWishStressDamageNone(notActive)`, never called | FR-042 | unhappy (state error) |
| D8 | active, `WishCastOrigin.wishDuplicate` | `RollWishStressDamageNone(duplicatedSpell)` | FR-043 | unhappy (rule interaction) |
| D9 | active, damage result | `Character` equal; no state change | FR-044 | unhappy (immutability) |
| D10, D11 | `afterDay` weakened(5), `ordinary`; `restingWithLightActivity` | weakened(4); weakened(3) | FR-050 | happy |
| D12 | weakened(1), `restingWithLightActivity` | `NoWishWeakness`, `WishWeaknessEnded` | FR-050 | unhappy (boundary) |
| D13 | weakened(2), `ordinary` | weakened(1), no event | FR-050 | unhappy (boundary) |
| D14 | weakened(2), `restingWithLightActivity` | `NoWishWeakness`, `WishWeaknessEnded` | FR-050 | unhappy (boundary) |
| D15 | `NoWishWeakness`, any day | equal, no events | FR-051 | unhappy (empty and minimal) |
| D16 | active, `RestApplied(longRest, finished)` | inactive, `WishCastDamageEnded`; weakness and lockout equal | FR-052 | happy |
| D17 | active, `RestApplied(shortRest, finished)` | equal, no events | FR-052 | unhappy (rule interaction) |
| D18 | active, `RestApplied(shortRest, longRestInterrupted)` | equal, no events | FR-052 | unhappy (rule interaction) |
| D19 | inactive, `RestApplied(longRest)` | equal, no events | FR-052 | unhappy (empty and minimal) |
| D20 | weakened(6), a Long Rest | weakened(6) | FR-053 | unhappy (rule interaction) |
| D21 | a `RestApplied` for another character | `*InvalidError` | FR-062 | unhappy (invalid input) |
| D22, D23 | `CastAtLevel` 2 on a level 3 spell; `CastAtLevel` 1 on a cantrip | `*InvalidError` | FR-062, FR-068 | unhappy (invalid input) |

## 5. Model

Everything outside the two use case methods is immutable (unexported fields, validating constructors, value receivers, no setters), pure, synchronous and free of dice and time (`doc/rules/architecture.md` §3). All types are in package `character` (files `wish_*.go`, the use case in `wish_usecase.go`). Closed families are interfaces with an unexported marker method and one struct type per variant, marked `//sumtype:decl`, each doc comment listing every variant. Upstream types are reused, not redefined: `Character`, `FeatureStates`, `FeatureStateKey`, `FeatureState`, `CharacterEvent`, `CharacterID` (character-aggregate.md); `SpellID`, `FeatID`, `DamageType` (content-schema-and-decoder.md); `Spell`, `SpellLevel` (content-spells.md); `Cost` (content-equipment.md); `ability.Scores`, `ability.Ability` (the `ability` package); `modifier.RollMode` (the `modifier` package); `RestApplied` (rests-and-hit-dice.md); `SpellCast` (casting-and-concentration.md); `CharacterFeats`, `Feat`, `FeatChoices`, `HeldFeatures`, `FeatRejection`, `ReplaceFeat` (feats-behaviour.md).

**State** (`WishStress`, a `FeatureState`; three independent closed parts, no booleans). The zero `WishStress` is "no stress": a nil part reads as its first variant.

- `WishWeakness`: `NoWishWeakness` | `WishWeakened(DaysRemaining)` (1 or more; `NewWishWeakened(days int) (WishWeakened, error)` returns an `*InvalidError` otherwise).
- `WishCastDamage`: `WishCastDamageInactive` | `WishCastDamageActive`.
- `WishCastingAbility`: `CanCastWish` | `CannotCastWishAgain`.
- `WishStressKey()` returns the `FeatureStateKey` for `WishStress` named `wish_stress`.

**Choices and handoffs**

- `WishChoice` (closed): `DuplicateSpell(Spell)`, `ObjectCreation(Value Cost, LargestDimensionFeet, IsMagicItem)`, `InstantHealth(OtherCreatures)`, `ResistanceGrant(DamageType, Creatures)`, `SpellImmunityGrant(Subject, Creatures)`, `SuddenLearning(OldFeatID, NewFeat, Choices, FeatsRequiredByOthers, HeldFeatures, ResolvedSpells, Feats)`, `RollRedo(RollMode, RollWithinLastRound)`, `ReshapeReality(Statement)`. Each non-trivial variant has a validating constructor (`NewObjectCreation`, `NewInstantHealth`, `NewResistanceGrant`, `NewSpellImmunityGrant`, `NewRollRedo`, `NewReshapeReality`) returning `(T, error)`. `WishStatement` is an opaque non-empty string value built by `NewWishStatement` (FR-014); `SpellImmunitySubject` (closed): `ImmuneToSpell(SpellID)`, `ImmuneToOtherEffect(WishStatement)`.
- `WishEffect` (closed handoff, one per choice, structured facts and no display text): `WishDuplicate`, `WishObject`, `WishInstantHealth` (names Greater Restoration), `WishResistance` (permanent), `WishSpellImmunity` (`DurationHours()` 8), `WishSuddenLearning`, `WishRollRedo`, `WishReshapeReality`. The host or `combat` applies them (section 1).
- `UnpreventableDamage(type, amount)` (FR-044); `WishStressRoll(days, percentile)`; `WishRecoveryDay` (`ordinary`, `restingWithLightActivity`); `WishCastOrigin` (`ordinaryCast`, `wishDuplicate`; FR-069); `WishTransition{Stress, Events}`.

**Pure functions** (no port, no double, unit tested with plain values):

- `CheckWish(c Character, choice WishChoice) WishCheck` returning the rejection or the effect (FR-002 to FR-014, FR-021, FR-031); `WishRequiresStress(choice) bool` (FR-020); `ApplyWish(c Character, choice WishChoice, roll WishStressRoll) (WishApplication, error)` returning the new `Character`, the effect and the events, composing `ReplaceFeat` for Sudden Learning (FR-011). They roll nothing. `WishSpellID()`, `WishLevel()`.
- `WishStress.CanCastWish()`, `WishLevelOfCast(SpellCast, Spell)`, `WishDamageDice(level)` (FR-031, FR-040 to FR-043), `IncurWishStress(WishStress, WishStressRoll) WishStress` (FR-023 to FR-025).
- `WishEffectiveScores` (FR-030). `WishAfterDay` and `WishAfterRest`, `WishRestingDayReduction` (FR-050 to FR-053).

**Events** (past tense, FR-061): closed `WishEvent`: `WishDuplicated(spellID, level)`, `WishStressEndured(days, percentile, lockedOut)`, `WishWeaknessEnded`, `WishCastDamageEnded`, `WishStressDamageRolled(level, amount)`. The effect events of Sudden Learning are `feats-behaviour.md`'s (`FeatReplaced`).

**Rejections, errors and panics.** `ResolveWishRejection` holds one `ResolveWishRejectionKind` (a typed enum): `cannotCastWishAgain`, `spellLevelTooHigh`, `objectValueTooHigh`, `objectIsMagicItem`, `objectTooLarge`, `tooManyCreatures`, `rollNotRecent`, `featRequiredByAnother`, `featReplacementRejected` (this one carries the nested `FeatRejection`). Checks run in this fixed order: `cannotCastWishAgain`, then the choice's own checks as listed in FR-002 to FR-014 (for Sudden Learning: old feat held, `featRequiredByAnother`, then the service's remaining checks, FR-012). A rejection is a result, never an error and never a panic. Invalid input (FR-007, FR-009, FR-014, FR-022, FR-062, FR-068) is an `*InvalidError` matching `srderr.ErrInvalidInput`, returned by the constructor or the use case method before any die is rolled or signal emitted; the `MustX` helpers panic with that error and no exported API taking untrusted input panics. The only port failure is the dice port's `*srderr.PortError` (FR-064).

## 6. Interface

Two use cases, methods of `WishResolver`, each with an exported `<UseCase>Command`; both take `ctx` as first parameter because they emit spans (go.md §4), and neither stores it. The dice source is the consumer-side one-method interface `Roller` (`character.Roller`, owned by `character-aggregate.md`: `Roll(dice.Expression) (dice.Result, error)`, used here, not redeclared; any `dice.Roller` and `dicetest.FixedRoller` satisfy it) and every roll goes through `dice.Roll`, which wraps a failure once as a `*srderr.PortError`.

```go
func NewWishResolver(roller Roller, opts ...Option) (WishResolver, error) // roller nil: *InvalidError
func (r WishResolver) ResolveWish(ctx context.Context, cmd ResolveWishCommand) (ResolveWishResult, error)
func (r WishResolver) RollWishStressDamage(ctx context.Context, cmd RollWishStressDamageCommand) (RollWishStressDamageResult, error)
```

`Option` and `WithTelemetry(telemetry.Telemetry)` are the shared options of package `character` (character-aggregate.md); without `WithTelemetry` the resolver is silent (zero-value no-ops). `WishResolver` is safe for concurrent use when its `Roller` is.

| Use case | Command | Result |
|---|---|---|
| `ResolveWish` | `ResolveWishCommand{Character, Choice}` | `ResolveWishDuplicated(Character, Effect, Events)`, `ResolveWishApplied(Character, Feats, Effect, Events)` (with the stress, FR-022 to FR-026), `ResolveWishRejected(Rejection)` |
| `RollWishStressDamage` | `RollWishStressDamageCommand{Character, SpellCast, Spell, Origin}` | `RollWishStressDamageRolled(Damage, Events)`, `RollWishStressDamageNone(Reason, Events)` with reason `cantrip`, `notActive` or `duplicatedSpell` |

Both result families are closed interfaces handled with an exhaustive type switch. Errors: a dice-port failure is a `*srderr.PortError` (FR-064); invalid input is an `*InvalidError`; `context.Canceled` / `context.DeadlineExceeded` from `ctx` are never turned into a result (errors-and-results.md FR-054).

Pure, published and not use cases: `WishEffectiveScores` (hosts feed the result to `derived-stats.md` functions), `WishAfterDay` and `WishAfterRest` (hosts call them as days pass and after `RestApplied`; each returns a `WishTransition` with the new `WishStress`, so a host writes it back with `Character.WithFeatureState`), `WishStress.CanCastWish`. The `ResolveWish` result carries `CharacterFeats` only for Sudden Learning (feats are outside the `Character`, feats-behaviour.md Q4).

Host flow: `CastSpell` for Wish (when `CanCastWish`), `ResolveWish`, apply the effect, then `RollWishStressDamage` for each later `SpellCast` while stress is active (the Wish cast itself included), `WishAfterDay` per day, `WishAfterRest` per `RestApplied`. No use case calls another (shared-types-and-ownership.md FR-044).

**Exports**, from package `character` (FR-067; no facade): the two use case methods with their commands, results and variants; `WishChoice` and its variants, `WishEffect` and its variants, `WishStatement`, `SpellImmunitySubject`, `WishStress` with its parts and `WishStressKey`, `WishStressRoll`, `WishRecoveryDay`, `UnpreventableDamage`, `ResolveWishRejection` and its kind, `WishEvent` and its events, `WishTransition`, and the pure functions above. Not re-exported: other specs' types. `dicetest` and `telemetrytest` gain nothing (no new port).

## 7. Observability

The pure functions are silent (FR-066); the use cases emit the signals below (`doc/rules/observability.md` §2).

1. **What can go wrong:** a rule rejection (normal outcome); a dice-port failure (`*srderr.PortError`); invalid input (`*InvalidError`, returned before any signal). 2. **How an operator knows:** `srd5e.outcome=failure` with `srd5e.character.rejection` for a rejection (span status not error); for a port failure span error status plus `error.type` from `srderr.ErrorType` (`dice_roller_error`) and no result attribute (character-aggregate.md FR-071, FR-088); `defer span.End()` always runs. 3. **Diagnosis:** the chosen effect and the stress outcome are low-cardinality attributes; the amounts are in the result. 4. **Counted or timed:** one operation count and one duration per call; the stress lockouts. 5. **Existing instrumentation:** `srd5e.character.operations` and `.duration` (owned by character-aggregate.md section 7) gain the operation values `resolve_wish` and `roll_wish_stress_damage`; nothing existing is misleading. 6. **Monitor:** the share of `rejected` by rejection kind, any `error.type` increment, and `wish_lockout=true` (a character lost Wish for good: a product-relevant event). 7. **Noise:** no log at `INFO` or above; no log at all, since span and counter already hold the outcome; one result is never counted twice.

`result` values: `duplicated`, `applied`, `rejected` (`resolve_wish`); `damage`, `none` (`roll_wish_stress_damage`). `duplicated`, `applied`, `damage` and `none` are `success`; `rejected` is `failure`.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.character.resolve_wish` span | Span | `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.wish_effect`, `srd5e.character.wish_lockout` (applied only), `srd5e.character.rejection` (rejections only) | Follow one Wish resolution including its two dice | Error status is a failing roller; a rejection is not an error |
| `srd5e.character.roll_wish_stress_damage` span | Span | `srd5e.outcome`, `srd5e.character.result` | Follow one stress damage roll | As above |
| `srd5e.character.operations` | Counter, `{operation}` | adds the two operation values; `srd5e.character.wish_effect` (`duplicate`, `object_creation`, `instant_health`, `resistance`, `spell_immunity`, `sudden_learning`, `roll_redo`, `reshape_reality`) and `srd5e.character.wish_lockout` (`true`, `false`) on `resolve_wish` only | Throughput, effect mix, lockout rate | Alert on any `error.type`; watch the `rejected` share per rejection kind |
| `srd5e.character.duration` | Histogram, `s` | `srd5e.character.operation` only | Latency per use case, through `telemetry.Timer` | p99 regression per operation |

These entries feed `doc/observability/signals.md`; the two attribute keys are new and are added to `character-aggregate.md` section 7's attribute list (made: character-aggregate.md section 7).

## 8. Test plan

Doubles: `dicetest.FixedRoller` (a script of results), `dicetest.CallbackRoller (returning an error)` (including one that fails on its second call and one returning a result for a different expression, which must surface as `srderr.ErrContractViolation`), and the recording telemetry from `telemetry/telemetrytest`. No new port, so no contract suite. Pure-function tests (the large majority) import no port or double. Tests are external (`package character_test`) files such as `wish_rules_test.go`, `wish_recovery_test.go` and `wish_usecase_test.go`; each carries `Requirement:` and `Path:` comments; the tables E, S and D are the rows (`doc/rules/testing.md` §5), table-driven with `t.Run`.

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002, FR-003, FR-004 | unit | E1 to E5; `wish_rules_test.go`; handoff fields; no dice, state equal |
| FR-005, FR-006, FR-007, FR-008, FR-009, FR-010 | unit | E6 to E18; each limit at, one below, one above; the invalid-input rows assert `errors.As` an `*InvalidError` and `errors.Is` `srderr.ErrInvalidInput`, and that `MustX` panics with the same error |
| FR-011, FR-012 | unit | E19 to E23 with `ReplaceFeat` real; source of the old feat kept; nested `FeatRejection` equals the service's |
| FR-013, FR-014 | unit, fuzz | E24 to E31; `FuzzNewWishStatement` (never panics, error iff empty/blank/over 2,000) |
| FR-068, FR-069 | unit | E34 to E37, D22, D23, D8 |
| FR-020, FR-021, FR-031 | unit and application | E28, E32, E33, S9; roller call count zero on rejection; `cannotCastWishAgain` first in the order |
| FR-022, FR-023, FR-024, FR-025, FR-026 | unit and application | Table S; the roll order asserted with a fixed roller; S11 event order; S7 lockout never cleared |
| FR-030 | unit | Effective Strength list under Table S; input unchanged |
| FR-040, FR-041, FR-042, FR-043, FR-044 | unit and application | D1 to D9; the requested `dice.Expression` equals `<level>d10`; zero-dice never rolled |
| FR-050, FR-051, FR-052, FR-053 | unit | D10 to D21 (`wish_recovery_test.go`); boundaries kill `<` versus `<=` |
| FR-060, FR-061, FR-065 | unit | Key name, absent state reads the zero `WishStress`, value equality, a mutated returned slice leaves the value unchanged; exhaustive type switch over `WishEvent` implementing `CharacterEvent` |
| FR-062, FR-063, FR-064 | unit and application | D21; equal results for equal commands; S10: a `*srderr.PortError` propagates (`errors.Is` the roller's error, `srderr.ErrPortPanic` for a panicking roller, `srderr.ErrContractViolation` for a wrong-expression result), state equal |
| FR-066 | application | Span, counter and duration keys pinned for both use cases; rejection is `failure` without error status; a returned port failure is recorded once with `error.type` = `srderr.ErrorType(err)`; recording and no-op telemetry give equal results |
| FR-067 | architecture | `internal/archtest`: every named type exported, no generic name; no `combat` import |
| scenario | scenario | `scenarios/wish_test.go` (`package scenarios_test`), public API: a Wizard 17 casts Wish for Reshape Reality, is stressed, casts a level 3 spell (3d10 handoff), rests two days, finishes a Long Rest; a Cleric 20 duplicates a level 8 spell without stress. Cites FR-004, FR-022, FR-041, FR-050, FR-052 |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The dice `*srderr.PortError` of FR-064 is returned unchanged from both use cases with the state equal; the returned failure recorded once (FR-066) sets `error.type` to `srderr.ErrorType(err)`; a panicking telemetry double propagates the panic and `span.End` still ran. |

- Not covered: none.
- Mutation testing: the Wish files of package `character` (`doc/rules/testing.md` §8; `gomu` only inside Docker with `--cpus` and `--workers` capped at a quarter of the host's processors, threshold 80); the 33 and 34 percentile rows and the 8, 20, 10 and 300 limits kill boundary mutants. Use cases SHOULD be mutation tested too.

## 9. Open questions and decisions

Open: none escalated. Settled by research or package design, each reasoned and not to be relitigated:

| Question | Decision | Status |
|---|---|---|
| Does "per level of that spell" use the slot level on an upcast? | Yes: "the spell takes on the higher level for that casting" (p. 104), so `CastAtLevel`; a Ritual and a feature-granted cantrip use the spell's own level (FR-040). Interpretation; the community reading agrees | decided |
| Does the Wish cast itself pay the damage once stressed, and the duplicated spell too? | The Wish cast pays (level 9); the duplicate is its effect and pays nothing (FR-043). Interpretation of "each time you cast a spell" as one charge per casting | decided |
| How does a resting day count? | 2 days off in all (FR-050), one named constant so it changes in one place; rejected: 1 for the day plus 2 (3 in all), since the text names only "decreases by 2 days". The host reports days (FR-053). Interpretation | decided |
| A second stress while weakened? | The new roll governs (FR-023); "Combining Spell Effects", p. 106: equally potent, most recent applies. Rejected: keep the greater (no SRD basis) | decided |
| A stored Strength below 3? | Becomes 3: the text says "becomes 3" (FR-030). Rejected: `min(stored, 3)` (a "weakens" reading that leaves 1 or 2 alone), a departure from the explicit wording. Interpretation | decided |
| Does Reshape Reality stress if the GM refuses? | Yes (FR-020); the package cannot know the ruling | decided |
| "End all effects ... listed in Greater Restoration": every level and instance? | The handoff names the spell and leaves the effect list to its data; this spec decides no count (FR-008) | decided |
| Temporary Hit Points against the Necrotic damage? | `combat`'s (hit-points-and-damage.md FR-114); consistent with casting-and-concentration.md FR-135, they are not a reduction of the damage | decided |
| How does a lockout reach `CastSpell`? | The `ResolveWish` backstop (FR-031) is the enforcement; the host-side refusal is `CastingContext.unavailableSpells` (casting-and-concentration.md FR-146) | decided |
| 33 percent as a roll | `1d100` of 1 to 33 locks (FR-022) | decided |

Amendments to other specs (A1, A2, A3, A4, A5, A6 and A7 are made; only A8 is outstanding, and nothing in this spec's types depends on it): A1 casting-and-concentration.md (a cast mode for a Wish duplicate: no requirement checks, no slot, no delay, begins Concentration); A2 casting-and-concentration.md (a way to mark a spell unavailable, rejection after `cannotCastSpells`); A8 feats-behaviour.md FR-082 and Q20 (allow replacing a feat that gave an ability score increase, with a handoff reversing it); A4 feats-behaviour.md FR-082 and Q20 (name `wish-spell.md`, not `spell-effects.md`, as the Wish user); A5 hit-points-and-damage.md FR-114 (the host maps `UnpreventableDamage` to the `unpreventable` list, which skips flat reductions, Resistance and Immunity; Vulnerability still applies since it is not a reduction; Temporary Hit Points still absorb first); A6 character-aggregate.md section 7 (the operation values and attribute keys of section 7 here) and `doc/observability/signals.md`; A7 `doc/documentation_index.md` (a row for this file).

## 10. Changelog impact

Under `Unreleased`, **Added** (not breaking, first release of this area): "Wish: `WishResolver.ResolveWish` for duplication and the seven effects with their limits as checks and handoffs, the stress as persistent state (`WishStress`), `WishResolver.RollWishStressDamage` with `UnpreventableDamage`, `WishEffectiveScores`, and `WishAfterDay` / `WishAfterRest` for recovery days and Long Rests." New signals (section 7) go to `doc/observability/signals.md`; renaming any later is **BREAKING**.
