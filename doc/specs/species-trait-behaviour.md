# Species traits: choices, uses and declared effects

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-source-port.md`, `content-species-backgrounds-feats.md`, `content-spells.md`, `derived-stats.md`, `character-aggregate.md`, `rests-and-hit-dice.md`, `heroic-inspiration.md`, `spellcasting.md`, `casting-and-concentration.md`, `feats-behaviour.md`, `character-creation.md` |

Prose only (shared-types-and-ownership.md FR-056): `feats-behaviour.md` (15) supplies the `SpeciesTraitFeat` source (Versatile) and the
state and use-case conventions this spec parallels; `character-creation.md` (16) supplies `SpeciesTraitSelection` and `SpeciesGrantSelections`;
`rests-and-hit-dice.md` (12) `RestApplied`, `RestKind`, `LongRestVariant`; `heroic-inspiration.md` (12) the gain use case the host calls;
`derived-stats.md` (10) `HitPointMaximumBonus`, `ProficiencyBonus`; `d20-tests.md` (4) `Roll`, `RerollDecision`. No `combat` spec is listed
(FR-013 there): `combat` consumes the declared effects of this spec, never the reverse. Rank 17 (section 4.11 there).

## 1. Purpose and scope

content-species-backgrounds-feats.md is **data only** and declined structured trait facts (its section 5, "Declined requests": revisit
"when a behaviour spec names a fact it cannot get from text"). This spec is that behaviour spec for the nine SRD species, parallel to
feats-behaviour.md. It owns, in the `character` context: turning the creation selections into typed `SpeciesTraitChoices`; the trait
facts as code constants (tables below, not data); limited-use state for traits with uses (Breath Weapon, Stonecunning, Giant Ancestry,
Draconic Flight, Large Form, Adrenaline Rush, Relentless Endurance, lineage spell free casts); the rest windows (Resourceful, High Elf
cantrip swap); and the **declared effects and handoff values** (resistance, save and check Advantage sources, rerolls, temporary Hit
Points, Hit Point maximum bonus, senses, Speed) that the host, `combat` or the spell specs compose. State in, new state and events out
(architecture.md §4); trait state is a `SpeciesTraits` value beside the `Character`, as `CharacterFeats` is (feats-behaviour.md Q4).

**Out of scope:** species data and the five `choices` elements (content-species-backgrounds-feats.md); creating the character and recording
selections (character-creation.md FR-040, FR-041, FR-106, FR-107, which records and applies nothing); taking the Versatile Origin feat
(`TakeFeat`, feats-behaviour.md table G: `SpeciesTraitFeat`, `FeatFromCategory(Origin)`); granting Heroic Inspiration (heroic-inspiration.md
`GainHeroicInspiration`, called by the host; this spec only reports the Resourceful window); casting the granted spells (casting-and-concentration.md
`CastMode` value `FeatureGranted`); applying damage, resistance, temporary Hit Points, conditions, movement, Hide, grapple escape and the
Breath Weapon attack replacement (the `combat` specs: hit-points-and-damage.md, attack-and-help-actions.md FR-062, grappling.md,
hiding-searching-and-senses.md, movement-lifting-and-flight.md); rolling damage dice of declared effects and critical hits (`combat`);
the carrying-capacity table (equipment-rules.md; this spec supplies the size to look up); a species outside pp. 83-86 and house species
(`SpeciesBehaviourUnknown`). Nothing core was found missing from SRD 5.2.1 for these species.

## 2. Source

SRD 5.2.1 pp. 83-86 (species, "Parts of a Species", the nine descriptions and the two lineage tables); p. 8 ("Heroic Inspiration",
"Interactions with Rerolls", "Proficiency", "They Don't Stack"); pp. 17-18 ("Resistance and Vulnerability", "Dropping to 0 Hit Points",
"Temporary Hit Points"); p. 104 ("Always-Prepared Spells", "Special Abilities"); pp. 178-190 glossary ("Carrying Capacity" p. 178, "Charmed"
p. 178, "Dash" and "Darkvision" p. 180, "Frightened" and "Grappled" p. 182, "Hide" p. 183, "Poisoned" and "Prone" p. 186, "Long Rest" p. 185,
"Short Rest" p. 187, "Tremorsense" p. 190). Page numbers are printed pages. Rule files: architecture.md §2 to §6, observability.md §2 to §7,
testing.md §4, §5, §8. Web research informed the interpretations of section 9 only; nothing from outside the SRD is used.

## 3. Vocabulary

Reused, not redefined: `SpeciesID`, `SpellID`, `FeatID`, `Skill`, `DamageType`, `ConditionKind`, `CreatureSize`, `RechargeTrigger`, `Spell`,
`LineageOption`, `TextOption`, `DamageTypeOption` (content); `Ability`, `AbilityModifier`, `dice.Expression`, `RollCircumstance` (kernel packages `ability`, `modifier`, `dice`);
`Roll`, `RerollDecision`, `DifficultyClass` (d20_test); `CharacterLevel`, `ProficiencyBonus`, `HitPointMaximumBonus` (derived-stats.md);
`RestApplied`, `RestKind`, `RestSource`, `LongRestVariant` (rests-and-hit-dice.md); `CharacterEvent`, `CharacterID` (character-aggregate.md);
`SpeciesTraitSelection`, `SpeciesGrantSelections` (character-creation.md).

| SRD term | Type | Notes |
|---|---|---|
| Draconic Ancestry; Elven Lineage; Gnomish Lineage; Giant Ancestry; Fiendish Legacy | `DraconicAncestry` (10), `ElvenLineage` (`drow`, `HighElf`, `WoodElf`), `GnomishLineage` (`ForestGnome`, `RockGnome`), `GiantAncestry` (6), `FiendishLegacy` (`abyssal`, `chthonic`, `infernal`) | Enums of this spec; the parse from option names is FR-002, FR-003 |
| a species' chosen traits | `SpeciesTraitChoices` (closed variant set, one variant per species), `SpeciesTraits` | Held beside the `Character` |
| a trait with limited uses | `SpeciesUse` (enum), `SpeciesUseLimit`, `SpeciesUseRequest` (closed variant set) | Table U |
| a trait's declared effect | `SpeciesTraitEffect` (closed variant set, published language) | Section 5 |
| spells a species grants | `SpeciesGrantedSpells` | Table L |

## 4. Requirements

Stable ids. Ids are allocated in blocks; an id absent from this section and not in the Withdrawn ids was never used by this spec. "Table X" is a case table after the requirements, one test per row. Species ids are those of content-species-backgrounds-feats.md
(`dragonborn`, `dwarf`, `elf`, `gnome`, `goliath`, `halfling`, `human`, `orc`, `tiefling`; where a bundled id differs, that spec's id wins).

### 4.1 Choices

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | Each bundled species MUST accept exactly the `SpeciesTraitChoices` variant of table C and no other. `NewSpeciesTraits(SpeciesID, SpeciesTraitChoices) (SpeciesTraits, error)` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` (a typed `*SpeciesMismatchError{Species, Choices}`) for a variant that does not match the species; the zero `SpeciesTraits` is invalid. A species id with no behaviour here MUST give `SpeciesBehaviourUnknown` from the parse (FR-008), never a guess. | SRD 5.2.1, "Parts of a Species", "Special Traits", pp. 83-84 | unhappy (invalid input) |
| FR-002 | `DraconicAncestry` MUST map to the damage type of table D (Black and Copper Acid; Blue and Bronze Lightning; Brass, Gold and Red Fire; Green Poison; Silver and White Cold), the same pairs as the content `DamageTypeOption`s. A selection name not among the ten gives `OptionUnknown`. | SRD 5.2.1, "Dragonborn", Draconic Ancestors, p. 84 | happy |
| FR-003 | `ElvenLineage`, `GnomishLineage`, `GiantAncestry` and `FiendishLegacy` MUST be parsed from the option `name` of a `SpeciesTraitSelection` by exact, case-sensitive equality with the printed heading (table C; the Giant Ancestry names include the descriptor and the typographic apostrophe, e.g. `Cloud’s Jaunt (Cloud Giant)`). Any other name gives `OptionUnknown`. | SRD 5.2.1, "Elf" p. 85, "Gnome" p. 85, "Goliath" pp. 85-86, "Tiefling" p. 86 | unhappy (invalid input) |
| FR-004 | The spellcasting ability of Elven Lineage, Gnomish Lineage and Fiendish Legacy MUST be Intelligence, Wisdom or Charisma, chosen when the lineage is selected; another ability gives `AbilityNotAllowed`, none `missing`. One ability applies to every spell of that trait, including Thaumaturgy of Otherworldly Presence. | SRD 5.2.1, "Elf" p. 85, "Gnome" p. 85, "Tiefling" p. 86 | unhappy (invalid input) |
| FR-005 | Keen Senses MUST take one `Skill` of Insight, Perception or Survival (`SkillNotAllowed` otherwise); Skillful MUST take any one `Skill`; a missing one is `missing`. A skill given to a species that has no such trait is `unexpected`. | SRD 5.2.1, "Elf" Keen Senses p. 85; "Human" Skillful p. 86 | unhappy (invalid input) |
| FR-006 | Choices not offered by the species MUST be `unexpected` (an ancestry for a Dwarf, an ability for a Halfling); a missing required choice MUST be `missing`. A failed parse MUST return an error that is `errors.Join` of every `SpeciesChoiceProblem` (a typed value with name and kind, no display text, implementing `error` and matching `srderr.ErrInvalidInput`), in input order; the joined error is never empty on failure (`errors-and-results.md` FR-023, FR-024). | architecture.md §5, §6; SRD 5.2.1, pp. 84-86 | unhappy (invalid input) |
| FR-007 | `ParseSpeciesTraitChoices(SpeciesID, selections, SpeciesGrantSelections) (SpeciesTraitChoices, error)` MUST NOT panic: it returns the choices and a nil error, or the zero value and the joined error of FR-006 (an unknown species is `ErrSpeciesBehaviourUnknown`, FR-001). `MustSpeciesTraitChoices` panics with that error and is for test literals. The Versatile feat of `SpeciesGrantSelections` is ignored here (taken through `TakeFeat`). | architecture.md §6; feats-behaviour.md table G | unhappy (invalid input) |
| FR-008 | Parsing a `SpeciesID` outside the nine bundled ids MUST give `SpeciesBehaviourUnknown` carrying the id and change no state. | architecture.md §3 (house content as data); feats-behaviour.md FR-018 | unhappy (missing content) |

### 4.2 Passive traits: handoff values

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `SpeciesPassives.Resistances` MUST return the `DamageType`s of table R: Dragonborn the ancestry's type, Dwarf Poison, Tiefling the legacy's type (Abyssal Poison, Chthonic Necrotic, Infernal Fire), every other species none. No Resistance stacks with itself; the returned set is a copy (mutating it changes nothing). | SRD 5.2.1, "Damage Resistance" p. 84; "Dwarven Resilience" p. 84; "Fiendish Legacies" p. 86; "No Stacking" p. 17 | happy |
| FR-011 | `SpeciesPassives.SaveCircumstances(SavingThrowSituation)` MUST return one `SpeciesAdvantage{Source, modifier.Advantage{}}` per matching trait of table A: Dwarven Resilience when the save avoids or ends Poisoned, Fey Ancestry Charmed, Brave Frightened, Gnomish Cunning any Intelligence, Wisdom or Charisma saving throw. A situation is an `ability.Ability` and an optional `content.ConditionKind` avoided or ended (a pointer or a `Valid` flag in the struct; the zero situation names no condition). `SpeciesPassives.CheckCircumstances(AbilityCheckSituation)` MUST do the same for ability checks (FR-013, FR-039). Several matches are all returned and the kernel's `modifier.Resolve` makes them one Advantage. | SRD 5.2.1, "Dwarf", "Elf", "Gnome", "Halfling" pp. 84-86; "They Don't Stack" p. 8 | unhappy (rule interaction) |
| FR-012 | A trait MUST NOT give Advantage on a situation outside table A (Brave on a Poisoned save, Gnomish Cunning on a Dexterity save, any trait of a species that lacks it), and a character without the species MUST get none. | SRD 5.2.1, pp. 84-86 | unhappy (boundary) |
| FR-013 | Goliath Powerful Build MUST give `SpeciesAdvantage` for an ability check made to end the Grappled condition (the Strength (Athletics) or Dexterity (Acrobatics) escape; an `AbilityCheckSituation` holds the `Ability`, whether it ends Grappled, and whether Large Form is active), and none for any other check. `CarryingCapacitySize(CreatureSize)` MUST return the next larger `CreatureSize`; Gargantuan stays Gargantuan. | SRD 5.2.1, "Powerful Build" p. 86; "Grappled" p. 182; "Carrying Capacity" p. 178 | unhappy (rule interaction) |
| FR-014 | Dwarven Toughness MUST give `HitPointMaximumBonus(flat: 1, perLevel: 1, grantedAtLevel: 1)` unscoped (total level), for the host to pass to derived-stats.md FR-029: +1 at level 1, +1 more per level gained. No other species gives a bonus. | SRD 5.2.1, "Dwarven Toughness" p. 84; derived-stats.md FR-051 | happy |
| FR-015 | `SpeciesPassives.darkvisionFeet` MUST follow table S (60; Dwarf, Orc and the Drow lineage 120; none for Goliath, Halfling and Human), and `SpeedFeet(baseSpeed, largeFormActive)` MUST return 35 for the Wood Elf lineage and `baseSpeed + 10` while Large Form is active. Content `speed` stays the base. | SRD 5.2.1, "Darkvision" and lineage tables pp. 84-86; "Large Form" p. 86 | unhappy (rule interaction) |
| FR-016 | Elf Trance MUST report that magic cannot put the character to sleep and permit `TranceLongRest` (rests-and-hit-dice.md FR-004); no other species permits it. | SRD 5.2.1, "Trance" p. 85 | happy |
| FR-017 | Halfling Nimbleness `MayMoveThrough(own, other)` MUST be true exactly when `other` is one `CreatureSize` larger than `own`; Naturally Stealthy `ObscuresForHide(own, obscurer)` MUST be true exactly when `obscurer` is at least one size larger than `own`. These widen only the cover or obscurement test of the Hide action (hiding-searching-and-senses.md) and moving through a space; they never remove other Hide requirements. The "can't stop in the same space" clause is a movement rule enforced by movement-lifting-and-flight.md; this spec declares only the pass-through permission. The character's current `CreatureSize` (species size, the Human or Tiefling Medium or Small choice, Large Form) is supplied by the host from character-creation.md; this spec never stores it. | SRD 5.2.1, "Halfling Nimbleness", "Naturally Stealthy" p. 86; "Hide" p. 183 | unhappy (boundary) |
| FR-018 | `SpeciesPassives.skillProficiencies` MUST return the Keen Senses skill (Elf) and the Skillful skill (Human) of the choices, for the host to merge into `SkillProficiencies`. Creation already records them (character-creation.md FR-107); a skill already held is not an error. | SRD 5.2.1, "Keen Senses" p. 85, "Skillful" p. 86 | unhappy (rule interaction) |

### 4.3 Spells granted by species

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | `SpeciesGrantedSpells.of(traits, CharacterLevel)` MUST list per table L: the level 1 benefit spells and cantrips from character level 1, the third-level spell from character level 3, the fifth-level spell from character level 5. Below its level a spell is absent. Boundaries: levels 2, 3, 4, 5. | SRD 5.2.1, "Elf" p. 84; "Tiefling" p. 86 | unhappy (boundary) |
| FR-021 | A level 3 or 5 lineage or legacy spell MUST be marked always prepared and not counted against a prepared-spell limit, and MUST carry its own free-cast `SpeciesUse` (table U): castable once without a spell slot, regained when a Long Rest is finished, also castable with a spell slot of the appropriate level. This spec exposes the spell, ability and use state; casting is casting-and-concentration.md. | SRD 5.2.1, "Elven Lineage" p. 84; "Fiendish Legacy" p. 86; "Always-Prepared Spells" p. 104 | happy |
| FR-022 | Every granted spell MUST carry the chosen spellcasting ability (FR-004) as `ability`; Thaumaturgy (Otherworldly Presence, always a Tiefling cantrip) MUST use the Fiendish Legacy ability. Spell save DC and attack bonus are derived from that ability by spellcasting.md. | SRD 5.2.1, "Otherworldly Presence" p. 86; "Elven Lineage" p. 85 | happy |
| FR-023 | Forest Gnome MUST grant Minor Illusion (cantrip) and Speak with Animals (always prepared, `SpeciesUse` `GnomeSpell` with Proficiency Bonus free casts, also castable with slots); Rock Gnome Mending and Prestidigitation (cantrips) and `ClockworkDevice` facts: 10 minutes to create, Tiny, AC 5, 1 Hit Point, at most three at a time, each ends after 8 hours. | SRD 5.2.1, "Gnomish Lineage" p. 85 | happy |
| FR-024 | `SpeciesGrantedSpells` MUST hold the `SpellID`s from the content option (`LineageOption.LevelBenefits()[i].SpellID`, `TextOption.SpellIDs`, content FR-018 to FR-021), not names; it makes no content lookup. Thaumaturgy is the one exception: content keeps it as text (content FR-022), so a constant `SpellID` `thaumaturgy` is used and the host resolves it when casting. | content-species-backgrounds-feats.md FR-021; architecture.md §2 | happy |
| FR-025 | The High Elf cantrip swap MUST be open only after `RenewSpeciesTraitUses` sees a `RestApplied` with `BenefitsOf` `LongRest` (FR-033). `ReplaceHighElfCantrip` MUST replace the lineage cantrip (Prestidigitation, or the previous replacement) with a spell of level 0 on the Wizard list that is not the current one, close the window and emit `HighElfCantripReplaced`. Not a High Elf: `SpeciesUseNotOffered`; window closed: `CantripWindowClosed`; other problems `SpeciesChoicesInvalid` (`ReplacementCantrip`: `NotACantrip`, `NotOnSpellList`, `duplicate`). | SRD 5.2.1, "High Elf" p. 85 | unhappy (state error) |
| FR-026 | A replaced cantrip MUST keep the chosen ability and the lineage's level 3 and 5 spells unchanged. The swap MAY be repeated at every later Long Rest. | SRD 5.2.1, "High Elf" p. 85 (interpretation Q6) | unhappy (rule interaction) |

### 4.4 Uses, rests and declared effects

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-030 | Each `SpeciesUse` MUST have the limit, renewal and minimum character level of table U. A use is available while expended uses are fewer than the limit; the limit is read at the moment of use, so a rise of Proficiency Bonus gives the extra use at once (Q13). | SRD 5.2.1, pp. 84-86; "Proficiency" p. 8 | unhappy (boundary) |
| FR-031 | `ExpendSpeciesTraitUse` MUST raise the expended count by one and return the `SpeciesTraitEffect` of FR-034 to FR-040, `SpeciesUseExpended`. It MUST reject, with state unchanged, in the order of table O: `SpeciesUseNotOffered` (species or lineage lacks the use), `SpeciesUseHasOwnUseCase` (the `UseStonesEndurance`, `UseRelentlessEndurance` and Stone `UseGiantAncestryBenefit` requests, which belong to `RollStonesEndurance` and `UseRelentlessEndurance`), `SpeciesUseNotYetAvailable(minimumLevel)`, `NotEnoughSpaceForLargeForm`, `SpeciesUseExhausted`. | SRD 5.2.1, pp. 84-86 | unhappy (state error) |
| FR-032 | Use limits that equal the Proficiency Bonus MUST follow it: levels 1 and 4 give 2, 5 and 8 give 3, 9 and 12 give 4, 13 and 16 give 5, 17 and 20 give 6 (one test per level). | SRD 5.2.1, "Proficiency Bonus" table, p. 8; "Level Advancement" p. 23 | unhappy (boundary) |
| FR-033 | `RenewSpeciesTraitUses` MUST take the `RestApplied` of this character and clear expended counts per `RestKind.recharges(RechargeTrigger)` (rests-and-hit-dice.md FR-065): `LongRest` clears every use; `ShortRest` (a Short Rest, or an interrupted Long Rest) clears only `ShortOrLongRest` uses (Adrenaline Rush). It MUST report the windows `resourceful` (Human) and `HighElfCantripReplacement` (High Elf, which also opens the swap) only for `BenefitsOf` `LongRest`; an event for another character gives `RenewSpeciesTraitUsesRejected(CharacterMismatch)`. Nothing to clear and no window: same state, no events, a successful no-op. | SRD 5.2.1, "Long Rest" p. 185, "Short Rest" p. 187; "Resourceful" p. 86; rests-and-hit-dice.md FR-060, FR-061 | unhappy (empty case) |
| FR-034 | Breath Weapon MUST declare `BreathWeaponEffect`: `shape` (the 15-foot Cone or 30-foot Line 5 feet wide, chosen each use), a Dexterity saving throw, DC = 8 + Constitution modifier + Proficiency Bonus, damage of the ancestry's type with 1d10 at levels 1 to 4, 2d10 at 5 to 10, 3d10 at 11 to 16, 4d10 at 17 to 20, half damage on a success, and that it replaces one attack of the Attack action. Cases in table B. | SRD 5.2.1, "Breath Weapon" p. 84 | unhappy (boundary) |
| FR-035 | Draconic Flight MUST be unavailable below character level 5 (`SpeciesUseNotYetAvailable(5)`); available it declares `DraconicFlightEffect`: Bonus Action, 10 minutes, Fly Speed equal to Speed, ends early when retracted or on Incapacitated, once per Long Rest. | SRD 5.2.1, "Draconic Flight" p. 84 | unhappy (boundary) |
| FR-036 | Stonecunning MUST declare `StonecunningEffect`: Bonus Action, Tremorsense 60 feet for 10 minutes, usable only on or touching a stone surface (a host fact, not checked), Proficiency Bonus uses. | SRD 5.2.1, "Stonecunning" p. 84; "Tremorsense" p. 190 | happy |
| FR-037 | Giant Ancestry MUST declare the effect of the chosen benefit only (table G), with Proficiency Bonus uses between Long Rests shared by the one benefit; `UseGiantAncestryBenefit(benefit)` names the benefit; one other than the chosen is `SpeciesUseNotOffered`, and the Stone benefit is declared by `RollStonesEndurance` only (`ExpendSpeciesTraitUse` gives `SpeciesUseHasOwnUseCase`), so `GiantAncestryEffect` has five variants. Damage dice are declared, not rolled (rolling and critical doubling are `combat`'s). | SRD 5.2.1, "Giant Ancestry" pp. 85-86 | happy |
| FR-038 | `RollStonesEndurance` MUST expend a use, roll 1d12 through the roller (`dice.Roll`), add the Constitution modifier, and return `reduction = max(0, roll + modifier)` and the reduced damage `max(0, damage - reduction)` for the incoming damage amount. Cases in table G. Checks run before any roll in table O order: `SpeciesUseNotOffered` (not a Goliath, or another ancestry), then `SpeciesUseExhausted` (the Giant Ancestry pool is shared, so a Stone Goliath who spent all Proficiency Bonus uses is exhausted). The use is spent only after a successful roll. A roll outside 1 to 12 from the port is a port failure: a `*srderr.PortError` matching `srderr.ErrContractViolation`, returned with the zero result and nothing spent. | SRD 5.2.1, "Stone's Endurance" p. 85 (interpretation Q5) | unhappy (boundary) |
| FR-039 | Large Form MUST declare `LargeFormEffect`: Bonus Action, size Large, 10 minutes or until ended, Advantage on Strength checks (a `SpeciesAdvantage`) and Speed +10 feet (FR-015), once per Long Rest, from character level 5. If the host reports `HasEnoughSpace` false the use MUST be rejected with `NotEnoughSpaceForLargeForm` and not spent. | SRD 5.2.1, "Large Form" p. 86 | unhappy (state error) |
| FR-040 | Adrenaline Rush MUST declare `AdrenalineRushEffect`: the Dash action as a Bonus Action and `TemporaryHitPointsGrant(amount = Proficiency Bonus)`. Whether the new Temporary Hit Points replace the old is `combat`'s rule ("They Don't Stack", p. 18). | SRD 5.2.1, "Adrenaline Rush" p. 86; "Dash" p. 180 | happy |
| FR-041 | `UseRelentlessEndurance` MUST take a `ZeroHitPointsOutcome` (`ZeroHitPointsNotKilled` or `ZeroHitPointsKilledOutright`, supplied by the host from combat). Not killed: spend the use, return `RelentlessEnduranceUsed` with `hitPointsInstead = 1`. Killed outright (Massive Damage, Hit Point maximum 0): `UseRelentlessEnduranceNotTriggered` and nothing spent. Already spent: `SpeciesUseExhausted`. Not an Orc: `SpeciesUseNotOffered`. Cases in table E. | SRD 5.2.1, "Relentless Endurance" p. 86; "Instant Death" p. 17 | unhappy (state error) |
| FR-042 | `HalflingLuck.eligibleDice(Roll, rerolledIndexes)` MUST return the indexes whose die shows 1, minus any die already rerolled by an effect; the natural value need not be 1 (a 1 under Advantage qualifies). Cases in table H. | SRD 5.2.1, "Luck" p. 86; "Interactions with Rerolls" p. 8 (interpretation Q1) | unhappy (rule interaction) |
| FR-043 | `RollHalflingLuck` MUST, for an eligible die, request one d20 (`dice.D20()`) through `RollChecked`, return the new face and `Roll.applyReroll(RerollDie(index), newValue)`, and the new roll MUST stand even when lower; an ineligible die gives `RollHalflingLuckNotApplicable` with no roll; a die already rerolled gives `DieAlreadyRerolled`; a non-Halfling `SpeciesUseNotOffered`. `RollHalflingLuckRerolled` returns `RerolledIndexes` with the die added, so a caller who passes it back is refused a second reroll of that die (the rule "you must use the new roll"). Luck has no use limit and no state. | SRD 5.2.1, "Luck" p. 86; "Heroic Inspiration" p. 8 | happy |
| FR-044 | Luck and Heroic Inspiration MUST be usable on one roll in either order, each on a die it has not rerolled; this spec enforces only the per-die rule, and the per-effect "only one die" limit of p. 8 is the caller's (d20-tests.md FR-067). | SRD 5.2.1, "Interactions with Rerolls" p. 8 (interpretation Q1) | unhappy (rule interaction) |
| FR-045 | Human Resourceful MUST be reported as the `resourceful` window (FR-033); the host then calls `GainHeroicInspiration`. This spec holds no `HeroicInspiration` state and never calls that use case. | SRD 5.2.1, "Resourceful" p. 86; "Heroic Inspiration" p. 8; shared-types-and-ownership.md FR-048 | happy |

### 4.5 State, invariants and determinism

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-050 | `SpeciesTraits`, `SpeciesTraitChoices`, every `SpeciesTraitEffect`, `SpeciesGrantedSpells` and every event MUST be immutable, compare by value (`==` where comparable, otherwise `Equal`) and expose collections only as copies (mutating a returned slice or map changes nothing). | architecture.md §3, §4 | unhappy (immutability) |
| FR-051 | Every transition MUST return new instances and leave `Character`, `SpeciesTraits` and the command equal to their prior value. | architecture.md §3 | unhappy (immutability) |
| FR-052 | A rejected transition MUST return the rejection and no events and MUST NOT panic. Invalid arguments (a negative expended count, FR-001's mismatch, a die index outside the roll, a damage amount below 0) MUST be rejected by the constructors (`WithExpended`, `NewRollHalflingLuckCommand`, `NewRollStonesEnduranceCommand`) with an error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. | architecture.md §6 | unhappy (state error) |
| FR-053 | When several rejection reasons hold, the reported reason MUST follow table O. | architecture.md §3 (determinism) | unhappy (rule interaction) |
| FR-054 | The same inputs and fixed dice MUST give an equal result on every call (no clock, randomness or ordering dependence). | architecture.md §3 | unhappy (determinism) |
| FR-055 | `SpeciesTraitEvent` MUST be a closed family implementing `CharacterEvent`: `SpeciesUseExpended`, `SpeciesUsesRenewed`, `HighElfCantripReplaced`, `HalflingLuckUsed`, `RelentlessEnduranceUsed`, `StonesEnduranceUsed`; each carries the `CharacterID` and typed fields, no display text, past tense. | architecture.md §4; character-aggregate.md FR-085; shared-types-and-ownership.md FR-019 | happy |
| FR-056 | `SpeciesTraits` MUST reject (`WithExpended` and the window methods return an error matching `srderr.ErrInvalidInput`) a use state for a `SpeciesUse` the species does not have, and a High Elf cantrip window on a species that is not Elf with the High Elf lineage. | architecture.md §3 | unhappy (invalid input) |

### 4.6 Use cases

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-060 | `ExpendSpeciesTraitUse`, `RenewSpeciesTraitUses` and `UseRelentlessEndurance` MUST be synchronous, take `ctx` first (they emit a span), return no error and use no port but `telemetry.Telemetry`; `RollHalflingLuck` and `RollStonesEndurance` MUST use the `Roller` and `telemetry.Telemetry` only and return `(result, error)`; `ReplaceHighElfCantrip` MUST use the `SpellSource` and `telemetry.Telemetry` and return `(result, error)`. Nothing spawns a goroutine. | architecture.md §4 | happy |
| FR-061 | `ReplaceHighElfCantrip` MUST look up the replacement through the `SpellSource` before returning; a missing spell (found=false) gives `ReplaceHighElfCantripContentMissing` with a `content.MissingContent{Kind, ID}` (kind spell) and a nil error; a source failure is returned as a `*srderr.PortError` (`srderr.WrapPort`, port `content`) with the zero result and no log, and `errors.Is` reaches the cause. | shared-types-and-ownership.md FR-018; architecture.md §4 | unhappy (missing content) |
| FR-062 | A roller failure in `RollHalflingLuck` or `RollStonesEndurance` (error, panic or a result for another expression, all surfaced by `dice.Roll` as a `*srderr.PortError`) MUST be returned unchanged, spend nothing and return the zero result; a die roll outside the die's range is a `*srderr.PortError` matching `srderr.ErrContractViolation`. | dice-and-rolls.md FR-051, FR-052; errors-and-results.md FR-034 | unhappy (collaborator failure) |
| FR-063 | Every use case MUST return its own closed `<UseCase>Result` with operation-prefixed variants (section 6); no generic variant name is exported. | shared-types-and-ownership.md FR-017 | happy |
| FR-064 | No use case MAY call another (shared-types-and-ownership.md FR-044). They compose the domain services `SpeciesTraitChoices`, `SpeciesPassives`, `SpeciesUses`, `SpeciesEffects`, `SpeciesGrantedSpells`, `HalflingLuck`, `RelentlessEndurance`, exported from package `character`, and record only their own signals. | architecture.md §4 | unhappy (nested telemetry) |
| FR-065 | Every use case MUST behave identically with the no-op and a recording `telemetry.Telemetry`, emit the signals of section 7 once per call through defined constants, and not log. | observability.md §7, §9 | unhappy (determinism) |
| FR-066 | No use case MAY alter its command, `Character` or `SpeciesTraits`. | architecture.md §3, §4 | unhappy (immutability) |

### Table C: choices per species (FR-001 to FR-008)

| Label | Species | Variant and fields | Printed option names | Cite | Path |
|---|---|---|---|---|---|
| Dragonborn | `dragonborn` | `DragonbornChoices(ancestry)` | Black, Blue, Brass, Bronze, Copper, Gold, Green, Red, Silver, White | p. 84 | happy |
| Elf | `elf` | `ElfChoices(lineage, ability, keenSensesSkill)` | Drow, High Elf, Wood Elf | p. 85 | happy |
| Gnome | `gnome` | `GnomeChoices(lineage, ability)` | Forest Gnome, Rock Gnome | p. 85 | happy |
| Goliath | `goliath` | `GoliathChoices(ancestry)` | `Cloud’s Jaunt (Cloud Giant)`, `Fire’s Burn (Fire Giant)`, `Frost’s Chill (Frost Giant)`, `Hill’s Tumble (Hill Giant)`, `Stone’s Endurance (Stone Giant)`, `Storm’s Thunder (Storm Giant)` | pp. 85-86 | happy |
| Tiefling | `tiefling` | `TieflingChoices(legacy, ability)` | Abyssal, Chthonic, Infernal | p. 86 | happy |
| Human | `human` | `HumanChoices(skill)` | n/a | p. 86 | happy |
| no choices | `dwarf`, `halfling`, `orc` | `NoSpeciesChoices` | n/a | pp. 84, 86 | happy |
| wrong variant | `dwarf` with `DragonbornChoices` | `srderr.ErrInvalidInput` (FR-001) | n/a | p. 84 | unhappy (invalid input) |
| missing, unexpected | `elf` with no ability; `dwarf` with an ability; `elf` skill Athletics | `missing`; `unexpected`; `SkillNotAllowed` (one test each) | n/a | p. 85 | unhappy (invalid input) |
| ability not allowed | `tiefling` with Strength | `AbilityNotAllowed` | n/a | p. 86 | unhappy (invalid input) |
| every problem | `elf`, unknown lineage, Strength, no skill | `OptionUnknown`, `AbilityNotAllowed`, `missing`, all three listed | n/a | p. 85 | unhappy (rule interaction) |
| unknown species; unknown option | `house_species`; `dragonborn` option `Sonic` | `SpeciesBehaviourUnknown`; `OptionUnknown` | n/a | p. 84 | unhappy (missing content) |

### Table D, R, A and S: passive values (FR-002, FR-010 to FR-017)

| Label | Input | Expected | Cite | Path |
|---|---|---|---|---|
| ancestry to type (ten rows) | Black, Blue, Brass, Bronze, Copper, Gold, Green, Red, Silver, White | Acid, Lightning, Fire, Lightning, Acid, Fire, Poison, Fire, Cold, Cold | p. 84 | happy |
| resistance | Dragonborn Gold; Dwarf; Tiefling Chthonic; Tiefling Abyssal; Tiefling Infernal; Human | {Fire}; {Poison}; {Necrotic}; {Poison}; {Fire}; {} | pp. 84, 86 | happy |
| save: condition | Dwarf avoiding Poisoned; Elf ending Charmed; Halfling avoiding Frightened | one `SpeciesAdvantage` each (one test each) | pp. 84-86 | happy |
| save: Gnomish Cunning | Gnome, Intelligence, Wisdom, Charisma saves | one each; none for Strength, Dexterity, Constitution | p. 85 | unhappy (boundary) |
| save: one source | Gnome, Wisdom save to avoid Charmed; Elf, Wisdom save to avoid Charmed | Gnomish Cunning only; Fey Ancestry only (a Gnome lacks Fey Ancestry, an Elf lacks Cunning) | p. 85 | unhappy (rule interaction) |
| check: two sources | Goliath, Large Form active, Strength check to end Grappled | two `SpeciesAdvantage` (Powerful Build, Large Form); `modifier.Resolve` gives one Advantage | pp. 8, 86 | unhappy (rule interaction) |
| check: Large Form only | Goliath, Large Form active, Strength check not ending Grappled; Dexterity check ending Grappled, no Large Form | Large Form only; Powerful Build only | p. 86 | unhappy (boundary) |
| save: no match | Dwarf avoiding Frightened; Halfling avoiding Poisoned; Human any | none | pp. 84-86 | unhappy (boundary) |
| Powerful Build | Goliath ending Grappled; any other check | Advantage; none | p. 86 | unhappy (rule interaction) |
| capacity size | Medium; Large; Gargantuan | Large; Huge; Gargantuan | pp. 86, 178 | unhappy (boundary) |
| Dwarven Toughness | Dwarf | bonus (1, 1, 1); other species none | p. 84 | happy |
| Darkvision | Dragonborn, Elf (High, Wood), Gnome, Tiefling; Dwarf, Orc, Elf (Drow); Goliath, Halfling, Human | 60; 120; none | pp. 84-86 | unhappy (boundary) |
| Speed | Wood Elf base 30; Goliath 35 with Large Form; Drow 30 | 35; 45; 30 | pp. 85-86 | unhappy (rule interaction) |
| Trance | Elf; Dwarf | sleep immunity and `trance`; neither | p. 85 | happy |
| size checks | Small Halfling vs Medium; vs Large; vs Small; Hide obscured by Medium; by Small; by Large | move: true, false, false; hide: true, false, true | p. 86 | unhappy (boundary) |
| skills | Elf Insight; Human Stealth; Dwarf | {Insight}; {Stealth}; {} | pp. 85-86 | happy |

### Table L: granted spells (FR-020 to FR-024)

Printed lineage tables (pp. 85-86); ids are the `SpellID`s of content FR-022. Level means character level.

| Label | Level 1 spells and cantrips | Level 3 spell | Level 5 spell | Cite | Path |
|---|---|---|---|---|---|
| Drow | `dancing_lights` | `faerie_fire` | `darkness` | p. 85 | happy |
| High Elf | `prestidigitation` | `detect_magic` | `misty_step` | p. 85 | happy |
| Wood Elf | `druidcraft` | `longstrider` | `pass_without_trace` | p. 85 | happy |
| Abyssal | `poison_spray`, `thaumaturgy` | `ray_of_sickness` | `hold_person` | p. 86 | happy |
| Chthonic | `chill_touch`, `thaumaturgy` | `false_life` | `ray_of_enfeeblement` | p. 86 | happy |
| Infernal | `fire_bolt`, `thaumaturgy` | `hellish_rebuke` | `darkness` | p. 86 | happy |
| Forest Gnome | `minor_illusion` and `speak_with_animals` (always prepared, PB free casts) | none | none | p. 85 | happy |
| Rock Gnome | `mending`, `prestidigitation`, device facts | none | none | p. 85 | happy |
| boundaries | any Elf at levels 2, 3, 4, 5 | none, +L3, +L3, +L3 and L5 | | p. 84 | unhappy (boundary) |
| Gnome level 1 | Forest Gnome at level 1 | Speak with Animals present from level 1, as are the cantrips | | p. 85 | unhappy (boundary) |
| non-casting species | Dwarf, Human | `SpeciesGrantedSpells.empty` | | pp. 84-86 | unhappy (empty case) |
| High Elf swap | window open, Wizard cantrip `light` | accepted; window closed; `HighElfCantripReplaced` | | p. 85 | happy |
| swap rejected | no window; a Drow; a level 1 spell; a Cleric-only cantrip; the current cantrip | `CantripWindowClosed`; `SpeciesUseNotOffered`; `NotACantrip`; `NotOnSpellList`; `duplicate` (one test each) | | p. 85 | unhappy (invalid input) |
| swap twice | two Long Rests with a swap each | both accepted; a second swap in one window `CantripWindowClosed` | | p. 85 | unhappy (state error) |

### Table U: uses (FR-030 to FR-033)

| Label | `SpeciesUse` | Species | Limit | Renewal | Min level | Cite | Path |
|---|---|---|---|---|---|---|---|
| Breath Weapon | `BreathWeapon` | Dragonborn | PB | `LongRest` | 1 | p. 84 | happy |
| Draconic Flight | `DraconicFlight` | Dragonborn | 1 | `LongRest` | 5 | p. 84 | unhappy (boundary) |
| Stonecunning | `stonecunning` | Dwarf | PB | `LongRest` | 1 | p. 84 | happy |
| Giant Ancestry | `GiantAncestry` | Goliath | PB | `LongRest` | 1 | p. 85 | happy |
| Large Form | `LargeForm` | Goliath | 1 | `LongRest` | 5 | p. 86 | unhappy (boundary) |
| Adrenaline Rush | `AdrenalineRush` | Orc | PB | `ShortOrLongRest` | 1 | p. 86 | happy |
| Relentless Endurance | `RelentlessEndurance` | Orc | 1 | `LongRest` | 1 | p. 86 | happy |
| lineage spell 3 and 5 | `LineageSpellThree`, `LineageSpellFive` | Elf, Tiefling | 1 each | `LongRest` | 3; 5 | pp. 84, 86 | unhappy (boundary) |
| Speak with Animals | `GnomeSpell` | Forest Gnome | PB | `LongRest` | 1 | p. 85 | happy |
| expend then exhausted | Breath Weapon at level 1 (limit 2) | | | third request: `SpeciesUseExhausted`; third at level 5 accepted | | p. 84 | unhappy (boundary) |
| not offered | Dwarf requests Breath Weapon; Rock Gnome requests `GnomeSpell`; Goliath asks a benefit not chosen | | | `SpeciesUseNotOffered` (one test each) | | pp. 84-85 | unhappy (invalid input) |
| own use case | `ExpendSpeciesTraitUse` with `UseStonesEndurance`, `UseRelentlessEndurance`, Stone `UseGiantAncestryBenefit` | | | `SpeciesUseHasOwnUseCase` | | pp. 85-86 | unhappy (rule interaction) |
| renewal | long rest; short rest; interrupted Long Rest (`ShortRest`, `LongRestInterrupted`) with Adrenaline and Breath Weapon spent | | | all cleared; only Adrenaline; only Adrenaline | | pp. 185, 187 | unhappy (rule interaction) |
| windows | `RestApplied` long (Human; High Elf; Dwarf); short (Human) | | | `resourceful`; the cantrip window; none; none | | pp. 85-86 | unhappy (boundary) |
| other character | `RestApplied` for another id | | | `RenewSpeciesTraitUsesRejected` | | p. 185 | unhappy (state error) |
| nothing spent | long rest, nothing spent, Dwarf | | | same state, no events | | p. 185 | unhappy (empty case) |

### Table B: Breath Weapon (FR-034)

| Label | Level | Constitution modifier | PB | Expected | Cite | Path |
|---|---|---|---|---|---|---|
| level 1 | 1 | +2 | +2 | DC 12, 1d10 | p. 84 | happy |
| level 4 / 5 | 4; 5 | +2 | +2; +3 | 1d10, DC 12; 2d10, DC 13 | p. 84 | unhappy (boundary) |
| level 10 / 11 | 10; 11 | +3 | +4; +4 | 2d10, DC 15; 3d10, DC 15 | p. 84 | unhappy (boundary) |
| level 16 / 17 | 16; 17 | +3 | +5; +6 | 3d10, DC 16; 4d10, DC 17 | p. 84 | unhappy (boundary) |
| negative modifier | 1 | -1 | +2 | DC 9 | p. 84 | unhappy (boundary) |
| shapes | Cone; Line | | | 15-foot Cone; 30-foot Line 5 feet wide | p. 84 | happy |
| type | Red; Silver; Green | | | Fire; Cold; Poison | p. 84 | happy |

### Table G: Giant Ancestry (FR-037, FR-038)

| Label | Benefit | Declared effect | Cite | Path |
|---|---|---|---|---|
| Cloud | `CloudsJaunt` | Bonus Action; teleport up to 30 feet to an unoccupied space you can see | p. 85 | happy |
| Fire | `FiresBurn` | on hit with an attack roll that deals damage, +1d10 Fire | p. 85 | happy |
| Frost | `FrostsChill` | +1d6 Cold, target Speed -10 feet until the start of your next turn | p. 85 | happy |
| Hill | `HillsTumble` | on hit on a Large or smaller target, give it Prone | p. 85 | happy |
| Stone | `StonesEndurance` | Reaction on taking damage; 1d12 + Constitution modifier reduces it | p. 85 | happy |
| Storm | `StormsThunder` | Reaction on damage from a creature within 60 feet; 1d8 Thunder to it | p. 86 | happy |
| Stone roll cases | roll 6, Con +2, damage 20; roll 12, Con +3, damage 10; roll 1, Con -3, damage 5; roll 1, Con -5, damage 9 | reduction 8, 12 left; reduction 15, 0 left; reduction 0, 5 left; reduction 0, 9 left | p. 85 | unhappy (boundary) |
| Stone not offered | Dwarf; Goliath with Hill's Tumble | `SpeciesUseNotOffered`, no roll | p. 85 | unhappy (invalid input) |
| Stone exhausted | Stone Goliath level 1, two uses spent on the shared pool | `SpeciesUseExhausted`, no roll; spec state unchanged | p. 85 | unhappy (state error) |
| Stone via expend | `ExpendSpeciesTraitUse` with `UseStonesEndurance`; with Stone `UseGiantAncestryBenefit` | `SpeciesUseHasOwnUseCase` | p. 85 | unhappy (rule interaction) |
| Stone dice failure | failing roller | `*srderr.PortError` returned, use not spent | p. 85 | unhappy (collaborator failure) |
| Stone zero and negative damage | damage 0; damage -1 | reduction as rolled, 0 left; `NewRollStonesEnduranceCommand` returns an error matching `srderr.ErrInvalidInput` | p. 85 | unhappy (boundary) |

### Table H and E: rerolls and Relentless Endurance (FR-041 to FR-044)

| Label | Input | Expected | Cite | Path |
|---|---|---|---|---|
| single 1 | normal, [1] | eligible {0}; reroll 14 gives roll [14] | p. 86 | happy |
| single 2 | normal, [2] | none; `RollHalflingLuckNotApplicable`, no roll | p. 86 | unhappy (boundary) |
| advantage, one 1 | advantage, [1, 14] | eligible {0}; reroll to 3 gives natural 14 kept index 1 | pp. 8, 86 | unhappy (rule interaction) |
| two 1s | advantage, [1, 1] | eligible {0, 1}; one reroll changes one die | p. 8 | unhappy (rule interaction) |
| new roll stands | normal, [1], reroll gives 1 | roll [1] stands (the new roll is used, even when lower or equal) | p. 86 | unhappy (boundary) |
| second Luck call | normal, [1], reroll gave 1, called again with the returned `RerolledIndexes` | `DieAlreadyRerolled` | p. 86 | unhappy (state error) |
| after Inspiration | [1], die 0 already rerolled | `DieAlreadyRerolled` | p. 8 | unhappy (state error) |
| not a Halfling | Human | `SpeciesUseNotOffered` | p. 86 | unhappy (invalid input) |
| Endurance spent | Orc, not killed | `RelentlessEnduranceUsed`, hit points instead 1, use expended | p. 86 | happy |
| Endurance used twice | second drop | `SpeciesUseExhausted` | p. 86 | unhappy (state error) |
| Endurance after Long Rest | renew then drop | `RelentlessEnduranceUsed` | p. 86 | happy |
| killed outright | Orc, Massive Damage | `NotTriggered`, use unspent | p. 17 | unhappy (rule interaction) |
| not an Orc | Dwarf | `SpeciesUseNotOffered` | p. 86 | unhappy (invalid input) |

### Table O: order of rejection reasons (FR-031, FR-053)

`SpeciesBehaviourUnknown`, `SpeciesUseNotOffered`, `SpeciesUseHasOwnUseCase`, `SpeciesUseNotYetAvailable`,
`NotEnoughSpaceForLargeForm`, `SpeciesUseExhausted`. One test per adjacent pair with both conditions holding: an unknown species
and a use; a use not offered (Dwarf asks `UseRelentlessEndurance`) and own use case (an Orc asks it); own use case and low level; low level and no space (Large Form at level 4 without
space is `SpeciesUseNotYetAvailable`); no space and exhausted (`NotEnoughSpaceForLargeForm`); exhausted alone.

## 5. Model

All in package `character`, pure (no ports, `ctx`, goroutines or clock).

Go forms. Every variant and value object has unexported fields, `NewX` constructors that validate (returning `(T, error)`, errors matching `srderr.ErrInvalidInput`; `MustX` panics with that error for test literals) and read-only accessors named for the fields; collections are copied on construction and read. Closed variant sets are interfaces with an unexported marker method whose doc comment lists every variant. A stateless domain service `Service` with methods `Service.method` in this spec is, in Go, a zero-size struct `type Service struct{}` (usable as its zero value, value receivers, no state) with the capitalised method. Enum zero values are invalid unless stated.

- **Held state.** `SpeciesTraits` (value object): `SpeciesID`, `SpeciesTraitChoices`, expended counts per `SpeciesUse`, High Elf cantrip
  and its swap window; `NewSpeciesTraits(...)`, `WithExpended`, `ExpendedOf`. Not part of the `Character` (feats-behaviour.md Q4).
  `SpeciesTraitChoices` is a closed variant set (table C) built by `ParseSpeciesTraitChoices` (FR-007), whose failure is the joined `SpeciesChoiceProblem` errors; `SpeciesChoiceProblem(name, kind)`, names `DraconicAncestry`, `ElvenLineage`,
  `GnomishLineage`, `GiantAncestry`, `FiendishLegacy`, `SpellcastingAbility`, `KeenSensesSkill`, `SkillfulSkill`, `ReplacementCantrip`,
  `SpeciesChoicesShape`; kinds `missing`, `unexpected`, `OptionUnknown`, `AbilityNotAllowed`, `SkillNotAllowed`, `NotACantrip`,
  `NotOnSpellList`, `duplicate`.
- **Domain services (stateless).** `SpeciesPassives` (FR-010 to FR-018), `SpeciesGrantedSpells` (FR-020 to FR-024, `SpeciesGrantedSpell`
  with `spell`, `ability`, `AlwaysPrepared`, optional `SpeciesUse`; `ClockworkDevice` facts), `SpeciesUses` (`Expend`, `Renew`, FR-030 to
  FR-033), `SpeciesEffects.Of(SpeciesUseRequest, Character, SpeciesTraits)` (FR-034 to FR-040), `StonesEnduranceRule.Reduce(roll, modifier,
  damage)` (FR-038), `RelentlessEndurance.Resolve` (FR-041), `HalflingLuck.EligibleDice` (FR-042).
- **Requests.** `SpeciesUseRequest` (closed variant set): `UseBreathWeapon(shape)`, `UseDraconicFlight`, `UseStonecunning`, `UseGiantAncestryBenefit(benefit)`, `UseStonesEndurance`, `UseRelentlessEndurance` (the last two exist so `SpeciesUseHasOwnUseCase` is real),
  `UseLargeForm(hasEnoughSpace)`, `UseAdrenalineRush`, `UseLineageSpell(level)`, `UseGnomeSpell`. `ZeroHitPointsOutcome` (closed variant set:
`ZeroHitPointsNotKilled`, `ZeroHitPointsKilledOutright`) is defined here as a plain input; the host maps combat's result to it. Enums: `BreathWeaponShape`
  (`cone15`, `line30`), `SpeciesUse` (table U), `SpeciesWindow` (`resourceful`, `HighElfCantripReplacement`), `SpeciesTraitAction`
  (`BonusAction`, `reaction`).
- **Declared effects (closed `SpeciesTraitEffect`, published language).** `BreathWeaponEffect(shape, saveAbility, saveDc, damage, damageType,
  halfOnSave, replacesOneAttack)`, `DraconicFlightEffect(cost, durationMinutes, flySpeedEqualsSpeed, endsWhenIncapacitated)`,
  `StonecunningEffect(cost, tremorsenseFeet, durationMinutes)`, `GiantAncestryEffect` (closed variant set, the five non-Stone variants of table G),
  `LargeFormEffect(cost, size, durationMinutes, speedBonusFeet, strengthCheckAdvantage)`, `AdrenalineRushEffect(dash,
  TemporaryHitPointsGrant(amount))`. Plain facts, no display text; `combat` or the host interprets them, `d20` never sees one.
  `SpeciesAdvantage(source, Advantage)` pairs a kernel circumstance with its trait name for explanation.
- **Events.** `SpeciesTraitEvent` is a closed event family embedding `character.Event` (FR-055).
- **Failures.** `SpeciesTraitRejection` (closed variant set): `SpeciesBehaviourUnknown`, `SpeciesChoicesInvalid(problems)`, `SpeciesUseNotOffered`,
  `SpeciesUseHasOwnUseCase`, `SpeciesUseNotYetAvailable(minimumLevel)`, `NotEnoughSpaceForLargeForm`, `SpeciesUseExhausted`,
  `CantripWindowClosed`, `DieAlreadyRerolled`, `CharacterMismatch`. Invalid constructor input is an error matching `srderr.ErrInvalidInput` (architecture.md §6). The
  domain imports only the `content` and `d20` packages, kernel packages and its own context.

## 6. Interface

| Use case | Command | Result (closed variant set) | Port and error return |
|---|---|---|---|
| `ExpendSpeciesTraitUse` | `Character`, `SpeciesTraits`, `SpeciesUseRequest` | `ExpendSpeciesTraitUseResult`: `...Succeeded` (`SpeciesTraits`, `SpeciesTraitEffect`, `SpeciesUseExpended`), `...Rejected` | none; no error |
| `RenewSpeciesTraitUses` | `Character`, `SpeciesTraits`, `RestApplied` | `RenewSpeciesTraitUsesResult`: `...Succeeded` (`SpeciesTraits`, events, `[]SpeciesWindow`), `...Rejected` | none; no error |
| `UseRelentlessEndurance` | `Character`, `SpeciesTraits`, `ZeroHitPointsOutcome` | `UseRelentlessEnduranceResult`: `...Used` (`SpeciesTraits`, `HitPointsInstead`, event), `...NotTriggered`, `...Rejected` | none; no error |
| `RollHalflingLuck` | `Character`, `SpeciesTraits`, `Roll`, `DieIndex`, `RerolledIndexes` | `RollHalflingLuckResult`: `...Rerolled` (`NewValue`, `Roll`, `RerolledIndexes` with the die added, event), `...NotApplicable`, `...Rejected` | `Roller`; `error` is a port failure |
| `RollStonesEndurance` | `Character`, `SpeciesTraits`, `damage` | `RollStonesEnduranceResult`: `...Rolled` (roll, modifier, `reduction`, `DamageAfter`, `SpeciesTraits`, event), `...Rejected` | `Roller`; `error` is a port failure |
| `ReplaceHighElfCantrip` | `Character`, `SpeciesTraits`, `SpellID` | `ReplaceHighElfCantripResult`: `...Succeeded` (`SpeciesTraits`, event), `...Rejected`, `...ContentMissing` | `SpellSource`; `error` is a port failure |

Each command is `<UseCase>Command` (a value built by `New<UseCase>Command` where it validates); rejected variants carry a `SpeciesTraitRejection`. Missing content is `content.MissingContent{Kind, ID}`; the spell
is looked up by its id and its `Spell.Classes` and level checked in the domain. The use cases are methods of `SpeciesTraitUseCases`, built by `NewSpeciesTraitUseCases(roller Roller, spells SpellSource, opts ...Option) (SpeciesTraitUseCases, error)` (the error is for a nil port; options `WithTelemetry`, `WithClock` of the context). All six methods take `ctx context.Context` first and are safe for concurrent use when the ports are. Ports: `character.Roller` (owned by `character-aggregate.md`; used here, not redeclared; called only through `dice.Roll`), `SpellSource` (declared here as a consumer-side interface) (`LookupSpell(ctx, content.SpellID) (content.Spell, bool, error)`: `false` is absence, `error` an infrastructure failure, wrapped once with `srderr.WrapPort`; its shape follows `content-source-port.md`) and `telemetry.Telemetry`; no new port, double or contract suite.

**Exports.** Package `character` exports every type in sections 3 to 6 with its variants; the package
doc comment lists them. No facade or re-export exists. Every name is unique in the package scan (architecture.md §9); `SpeciesChoice` (content) is not redefined.

## 7. Observability

The domain is **silent by design**: it returns events, rejections and effect values. The six use cases are instrumented (observability.md §2)
with the shared signals of character-aggregate.md; this spec adds operation and rejection values, no parallel counter. Failures: missing
content (`srd5e.outcome=failure`, `result=content_missing`, span error status); port or dice failure (span error status, `error.type`);
rejection (`failure`, `result=rejected`, `srd5e.character.rejection`, span status not error); a luck die that is not 1 or an Orc
killed outright is a normal outcome (`success`, `result=not_applicable`). Ids, scores and spell names stay in results and events.
`srd5e.outcome` and `error.type` are `telemetry.AttrOutcome` and `telemetry.AttrErrorType` (`error.type` is `srderr.ErrorType(err)`; a returned rule failure has no `error.type`). No logs, no per-trait signal.

- `srd5e.character.operation`: `expend_species_trait_use`, `renew_species_trait_uses`, `use_relentless_endurance`,
  `roll_halfling_luck`, `roll_stones_endurance`, `replace_high_elf_cantrip`.
- `srd5e.character.result` reuses `changed`, `rejected`, `content_missing` (character-aggregate.md) and adds the new values `resolved` (a roll) and
  `not_applicable`; signals.md's shared entry must list them.
- `srd5e.character.rejection`: the snake_case of each `SpeciesTraitRejection` variant (`species_behaviour_unknown` to `character_mismatch`).

| Signal | Type, unit | Attributes | Emitted by | Purpose | Monitoring |
|---|---|---|---|---|---|
| `srd5e.character.operations` | counter, `{operation}` | `srd5e.character.operation`, `srd5e.outcome`, `srd5e.character.result`, `srd5e.character.rejection` | all six | Existing; adds values. What is counted | Rise in `species_use_not_offered` or `species_use_exhausted` (a host UI offering unavailable traits); any `species_behaviour_unknown` |
| `srd5e.character.duration` | histogram, `s` | `srd5e.character.operation` only | all six | Existing; adds values. What is timed | Latency regression per operation |
| `srd5e.character.roll_halfling_luck` | span | `outcome`, `result`, `rejection`; error status on a dice failure | `RollHalflingLuck` | Diagnosis of a dice failure or a die offered wrongly | Error status rate above 0 sustained |
| `srd5e.character.roll_stones_endurance` | span | as above | `RollStonesEndurance` | As above | As above |
| `srd5e.character.replace_high_elf_cantrip` | span | `outcome`, `result` (`changed`, `rejected`, `content_missing`), `rejection` | `ReplaceHighElfCantrip` | Which rule or missing entry stopped the swap | Error status rate above 0 sustained |
| `srd5e.character.expend_species_trait_use`, `renew_species_trait_uses`, `use_relentless_endurance` | span | `outcome`, `result`, `rejection` | the matching use case | Diagnosis of a refused use | Counter only |

Attribute values are low-cardinality; no ids, spell names or scores (observability.md §5). Entries feed `doc/observability/signals.md`.

## 8. Test plan

Unit for domain rules (`species_*_test.go` in package `character_test`), application for the six use cases (`*_usecase_test.go`), table-driven with `t.Run`, run with `go test -race -shuffle=on`; no contract suite.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-008 | unit | Table C, one test per row; `ParseSpeciesTraitChoices` and `NewSpeciesTraits` agree; every problem listed in the joined error; `FuzzParseSpeciesTraitChoices` never panics |
| FR-002, FR-010 to FR-018 | unit | Tables D, R, A, S, one test per row; a species without the trait gets nothing; `modifier.Resolve` over the returned circumstances gives one Advantage |
| FR-020 to FR-026 | unit + application | Table L; boundaries at levels 2 to 5; `contenttest.FakeSource` (found, missing, failing) for `ReplaceHighElfCantrip`; window opened by a long `RestApplied` only |
| FR-030 to FR-033 | unit | Table U; limits at every Proficiency Bonus level; renewal by `RestKind`; no-op renewal |
| FR-034 to FR-040 | unit | Tables B, G; each effect built and compared by value; Large Form without space is not spent |
| FR-041 to FR-045 | unit + application | Table H and E; `dicetest.NewFixedRoller`: a new roll of 1 stays; no roll for an ineligible die; Stone's Endurance extremes |
| FR-050 to FR-056 | unit | copies on every collection accessor; value equality; inputs unchanged; table O; `srderr.ErrInvalidInput` errors; determinism; event family type switch lists every variant |
| FR-060 to FR-066 | application | `telemetrytest.Recorder`: span names, attributes, one counter and one histogram per call, no log, `outcome` only `success` or `failure`, failures recorded once, rejection not an error status; a failing roller and source are returned as `*srderr.PortError`; identical results with `telemetry.Noop`; no `*_usecase.go` file calls another use case |
| (architecture) | architecture | the domain files import only `content`, `d20`, kernel packages and the own context; no duplicate exported name (`SpeciesUseExpended`, `SpeciesWindow`, `DraconicAncestry`, `TemporaryHitPointsGrant`, the effect field types); signal constants match `doc/observability/signals.md` |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The failing roller and source of FR-060 to FR-062 are returned as a `*srderr.PortError` that `errors.Is` matches to the cause; a recorded failure sets `error.type` to `srderr.ErrorType(err)`; a panic in a telemetry double propagates (`errors-and-results.md` FR-051); a cancelled `ctx` is not recorded as a failure outcome (FR-054). |

Doubles: `contenttest.FakeSource`, `dicetest.NewFixedRoller` / `dicetest.CallbackRoller`, `telemetrytest.Recorder`. A scenario test (a level 5 Dragonborn
uses Breath Weapon, rests, regains it) is recommended once character creation exists. Mutation testing: every changed domain file of package `character`,
score at least 80 (testing.md §8; `gomu` only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`, not run by this spec).

## 9. Open questions and decisions

No question is open or escalated. Web research was used only to choose between readings; each is an interpretation justified in SRD terms.

- Q1 (interpretation): Luck applies to any die of the roll that shows 1, not only the kept one, because the text says "a 1 on the d20" and p. 8 lets you reroll one chosen die under Advantage or Disadvantage (community consensus agrees). Each die is rerolled at most once ("you must use the new roll"); Luck and Heroic Inspiration combine in either order (FR-043, FR-044).
- Q2 (interpretation): Relentless Endurance does not apply when killed outright (Massive Damage, Hit Point maximum 0, p. 17); knocking a creature out leaves it at 1 Hit Point, so the trait never triggers (p. 17). The host maps combat to `ZeroHitPointsOutcome`.
- Q3 (interpretation): Powerful Build counts one size larger than the creature's current size, so a Goliath in Large Form counts as Huge for carrying capacity; Gargantuan is the ceiling (p. 86). Not escalated: a rare interaction and the size is one input of equipment-rules.md.
- Q4 (interpretation): Halfling Nimbleness covers exactly "a size larger", read as one size category (FR-017).
- Q5 (interpretation): Stone's Endurance reduces damage and cannot increase it, so a negative total gives reduction 0 (FR-038).
- Q6 (interpretation): the High Elf swap window stays open until used or until the next Long Rest, because no clock exists (heroic-inspiration.md D-8 takes the same stance); it replaces the lineage's current cantrip each time (FR-025, FR-026).
- Q7: Breath Weapon replaces one attack of the Attack action (p. 84): declared as `ReplacesOneAttack`; attack-and-help-actions.md FR-062 records it; no use case here calls it.
- Q8: lineage free casts are gated by the host calling `ExpendSpeciesTraitUse` first, since no use case calls another (FR-064, as feats-behaviour.md Q8); amend spellcasting.md and casting-and-concentration.md to accept `SpeciesGrantedSpells.ability`.
- Q9: Resourceful is a window only; the Long Rest it needs is `RestApplied` with `BenefitsOf` `LongRest` (rests-and-hit-dice.md W-10, FR-061) (FR-033, FR-045).
- Q10: species facts (Darkvision ranges, Wood Elf Speed, resistances, Giant Ancestry effects) are code constants because content declined structured trait facts; a data test pins them against the printed text.
- Q11: the dependency on hit-points-and-damage.md of the candidate list was dropped (rank 21 is downstream of `character`, shared-types-and-ownership.md FR-013); `combat` reads the declared effects, `ZeroHitPointsOutcome` is the only input from combat and the host supplies it.
- Q12: Large Form's "big enough space" and Stonecunning's stone surface are host facts (FR-036, FR-039).
- Q13: a rising Proficiency Bonus grants extra uses at once because the limit is read at use time (FR-030, FR-032).
- Q14: Dwarven Resilience and the other save traits cover saves "to avoid or end" the condition, so the situation names the condition regardless of whether the save is first or repeated (FR-011).

## 10. Changelog impact

Added, under `Unreleased` (new API, not breaking before 0.1.0): the six species-trait use cases, `SpeciesTraits`, `SpeciesTraitChoices`,
`SpeciesTraitEffect`, `SpeciesGrantedSpells`, the species events, the signals of section 7 and the new values of `srd5e.character.operation`
and `.rejection`. Not **BREAKING**. New values `resolved` and `not_applicable` for `srd5e.character.result`. Follow-up edits for the integrator: spellcasting.md and
casting-and-concentration.md FR-121 to FR-124 accept a species-granted `SpeciesGrantedSpells.ability` (Q8); attack-and-help-actions.md FR-062 takes
Breath Weapon as an `AttackReplacementSource` (Q7); signals.md lists the two new result values. They become changelog entries when code lands.
