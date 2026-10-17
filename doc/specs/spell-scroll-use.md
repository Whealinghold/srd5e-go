# Spell Scroll use

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `errors-and-results.md`, `ability-scores.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-spells.md`, `content-magic-items.md`, `derived-stats.md`, `spellcasting.md`, `magic-item-rules.md`, `casting-and-concentration.md` |

Dependency notes (prose only; shared-types-and-ownership.md FR-056): rank 15 (with `feats-behaviour`), above `casting-and-concentration.md` (rank 14), `spellcasting.md` and `magic-item-rules.md` (13) and `derived-stats.md` (10, for `CharacterLevel`, shared-types-and-ownership.md FR-008). The rank is requested here and must be entered in the section 4.11 table, `documentation_index.md` and the Phase 3 list of `roadmap.md` (section 10). `class-rogue.md` and `class-wizard.md` (rank 17) depend on this spec, not the reverse; they are cited in prose only. `equipment-rules.md` (its FR-049, the Spell Scroll Costs lookup) is cited in prose: this spec never reads a price or a time. No `combat` spec is listed (shared-types-and-ownership.md FR-013 to FR-016). No port, no dice: the checks are made by the host and arrive as `CheckResult` values.

## 1. Purpose and scope

Models the rules for using a Spell Scroll that no other spec owns: who can read it, what the reader's ability check is when the spell is above the level they can normally cast, the save DC and attack bonus the cast uses, what happens to the scroll after a cast, an interrupted cast and a failed check, how a scroll that a character scribed differs from one found, and copying a Wizard spell from a scroll into a spellbook. It generalises the Rogue's Use Magic Device (Scrolls) so `class-rogue.md` reuses it instead of keeping a private copy (`class-rogue.md` FR-097 to FR-099 (withdrawn), moved here).

All of it is **domain only**: pure functions and value objects (shared-types-and-ownership.md FR-023, FR-044). The cast itself is `casting-and-concentration.md`'s `CastSpell` (components, casting time, Concentration, armor); this spec supplies the terms of the cast and settles the scroll afterwards. Consuming the scroll item is `magic-item-rules.md` FR-023.

Reused, not redefined: `Ability`, `Skill`, `DifficultyClass`, `CheckResult`, `D20TestKind` (shared-types-and-ownership.md FR-001, FR-002, FR-005); `Spell`, `SpellLevel` (content-spells.md); `ClassID`, `SpellID`, `ItemRarity` (content-schema-and-decoder.md, content-magic-items.md FR-083); `SpellSaveDC`, `SpellAttackBonus`, `SpellcastingStats`, `SpellcastingClass` (spellcasting.md); `ScrollReading`, `MagicItemScrollNotReadable`, `MagicItemUseState` (magic-item-rules.md FR-009, FR-023, FR-024); `ComponentKind` (casting-and-concentration.md).

**Out of scope:**

- Casting rules for the spell itself (casting time, V and S components, Concentration, Casting in Armor, a free hand): `casting-and-concentration.md`. The scroll's spell effects: `magical-effects.md` and `spell-effects.md`.
- Price, time and cost to scribe (the Spell Scroll Costs table): `equipment-rules.md` FR-049. Crafting by several helpers, a Spell Scroll's market value (twice its scribing cost, "Magic Item Rarities and Values", p. 206): `magic-item-rules.md`.
- Spellbook mechanics beyond the scroll rule: the Wizard's level, time and cost of copying (`class-wizard.md` FR-035, FR-036).
- Other scrolls that are not Spell Scrolls (the SRD has none; "Scrolls", p. 205, names the Spell Scroll as the most prevalent). Identify, Counterspell and Dispel Magic on a scroll: the host reports an interruption (FR-016).
- Checking that the host rolled the option it passes to `ResolveScrollCheck`: `CheckResult` has no ability or skill field (d20-tests.md FR-041), so C3 and C4 do not cover it (FR-009).
- Unidentified-scroll discovery, scroll rarity tables for treasure (p. 244 lists rarity; it is carried as data only, FR-001), and which other magic items let a creature cast spells.

## 2. Source

PDF page numbers equal printed page numbers in SRD 5.2.1.

- SRD 5.2.1, "Magic Items A-Z", "Spell Scroll" (reading, higher-level check, Spell Scroll table, "Copying a Scroll into a Spellbook"), p. 244.
- SRD 5.2.1, "Magic Items", "Scrolls" ("Using a Scroll"), p. 205; "Consumable Items", "Spells Cast from Items", p. 206.
- SRD 5.2.1, "Equipment", "Spell Scroll (Cantrip, 30 GP; Level 1, 50 GP)", p. 99; "Scribing Spell Scrolls" (Time and Cost, Prerequisites for the Scribe, Cantrips, Spell Scroll Costs), p. 103.
- SRD 5.2.1, "Spells", "Gaining Spells", "Spell Slots", "Casting without Slots" (Magic Items), p. 104; "Multiclassing", "Spell Slots", "Pact Magic", pp. 25-26.
- SRD 5.2.1, "Rogue", "Level 13: Use Magic Device" (Scrolls), p. 64; "Wizard", "Expanding and Replacing a Spellbook", p. 78.
- SRD 5.2.1, "D20 Tests", p. 6; "Rules Glossary", "Ability Check", pp. 176-177 (read, not quoted).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Spell Scroll | `SpellScroll{spell, origin}` built by `NewSpellScroll` | One spell, a spell level 0 to 9; `origin` is found or scribed (FR-023) |
| unintelligible | `ScrollNotOnSpellList` | The spell is not on the reader's spell list (FR-004) |
| your spell list | `ScrollClassRoute(classID, ability)` | One route per class whose list the reader has |
| a higher level than you can normally cast | `ScrollReaderProfile.HighestSlotLevel` | Highest spell slot level of the reader (FR-007) |
| ability check using your spellcasting ability | `ScrollCheckOption(ability, skill)`, `ScrollCheckRequired` | DC 10 plus the spell's level (FR-008) |
| the spell disappears from the scroll | `ScrollSpellErased`, `DispositionErased` | FR-011 |
| Spell Scroll table | `SpellScrollTable`, `SpellScrollRow` | Rarity, Save DC, Attack Bonus by level (FR-001) |
| cast reliably, use any Spell Scroll | `AnyScrollRoute(ability, ReliableUpToLevel, CheckSkill)` | Use Magic Device (FR-019) |
| lowest possible spell and caster level | `ScrollLowestCasterLevel` (wraps `ItemCasterLevel.lowest`) | magic-item-rules.md FR-025; FR-014 |
| as if the caster were your level | `ScribedCantripCasterLevel(level)` | Scribed cantrip (FR-025) |
| Prerequisites for the Scribe | `ScribeFacts`, `ScribeProblem` | FR-027 |
| Copying a Scroll into a Spellbook | `ScrollCopying`, `ScrollCopyCheckRequired` | FR-029 to FR-031 |

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids; never renumber. Source cites are by section and printed page. Where the SRD is silent the row says "interpretation" and section 9 gives the reasoning.

### 4.1 The Spell Scroll table

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `SpellScrollTableRow(SpellLevel)` MUST be total over levels 0 to 9 and return the row of table T1 (rarity as `ItemRarity`, Save DC, Attack Bonus). Rarity is data for hosts; no rule here reads it. The table is a typed code projection of the ten-row table of the `spell_scroll` catalogue entry (content-magic-items.md FR-052), kept as a constant because package `character` reads no content source; a test MUST assert row-for-row equality with the bundled entry (Q12). | SRD 5.2.1, "Spell Scroll" (table), p. 244; content-magic-items.md FR-052 | happy |
| FR-002 | A **found** scroll's cast MUST use the table's Save DC and Attack Bonus for its spell level, whatever the reader's own spellcasting numbers are (no reader ability modifier or Proficiency Bonus enters them). A cantrip uses the cantrip row. | SRD 5.2.1, "Spell Scroll" ("The level of the spell on the scroll determines the spell's saving throw DC and attack bonus"), p. 244; "Spell Scroll (Cantrip)", p. 99 | happy |

Table T1 (one test per row; FR-001, FR-002; source p. 244): Cantrip Common 13 +5; level 1 Common 13 +5; 2 Uncommon 13 +5; 3 Uncommon 15 +7; 4 Rare 15 +7; 5 Rare 17 +9; 6 Very Rare 17 +9; 7 Very Rare 18 +10; 8 Very Rare 18 +10; 9 Legendary 19 +11. Boundaries: cantrip (lowest), 9 (highest), 2 to 3 (first DC step), 6 to 7 (first +10).

### 4.2 Reading and the level check

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-003 | `AssessScrollReading` MUST first apply `ScrollReading.CanAttempt` (magic-item-rules.md FR-009) to `UnderstandsWrittenLanguage` and `DescriptionRestricts`; a failure MUST be returned as `ScrollNotReadable` carrying that `MagicItemScrollNotReadable`, with no further check and nothing consumed. | SRD 5.2.1, "Using a Scroll" ("Any creature that can understand a written language can read a scroll and attempt to activate it unless its description notes otherwise"), p. 205 | unhappy (state error) |
| FR-004 | A readable scroll whose spell is on none of the reader's routes (FR-005) MUST give `ScrollNotOnSpellList` ("unintelligible"): no cast, no check, scroll retained. | SRD 5.2.1, "Spell Scroll" ("Otherwise, the scroll is unintelligible"), p. 244 | unhappy (rule interaction) |
| FR-005 | A class route applies when the scroll's `Spell.Classes` contains its `ClassID` (spellcasting.md FR-040). A reader with several classes has a route per class; a spell on several of their lists has several candidate routes. A reader with no route and no `AnyScrollRoute` has an empty spell list, so every scroll is unintelligible. | SRD 5.2.1, "Spell Scroll", p. 244; "Spell Scroll (Cantrip)" ("your class's spell list"), p. 99 | unhappy (empty and minimal) |
| FR-006 | When a candidate route can cast reliably (FR-007, FR-019) `AssessScrollReading` MUST return `ScrollReady(terms)` and no check. A cantrip on the list is always reliable. | SRD 5.2.1, "Spell Scroll" ("If the spell is on your spell list but of a higher level than you can normally cast, you make an ability check"), p. 244 | happy |
| FR-007 | A class route is reliable when the spell level is at most the reader's castable level: the greater of `HighestSlotLevel` (the highest spell slot level in the maximum slots across the Spellcasting and Pact Magic pools, 0 with none) and `GrantedCastLevel` (FR-038). Expended slots and used-up daily casts do not matter, and the level is the character's, not the route class's alone. Interpretation (section 9, Q1). | SRD 5.2.1, "Spell Slots", p. 104; "Multiclassing", "Spell Slots" and "Pact Magic", pp. 25-26 | unhappy (boundary) |
| FR-008 | When no candidate route is reliable, `AssessScrollReading` MUST return `ScrollCheckRequired` with `DC` equal to 10 plus the spell level, one `ScrollCheckOption(ability, skill)` per candidate route, in route order (class routes in the order given, then the `AnyScrollRoute`), and `TermsFor(option)`, the provisional `ScrollCastTerms` for each option (FR-036). A class route's option is the class's spellcasting ability and **no skill**: a bare ability check, so no Proficiency Bonus is added. | SRD 5.2.1, "Spell Scroll" ("an ability check using your spellcasting ability ... The DC equals 10 plus the spell's level"), p. 244; "D20 Tests", p. 6 | happy |
| FR-009 | `ResolveScrollCheck(required, option, CheckResult) (ScrollCheckResolutionResult, error)` MUST return a zero resolution and an error that is `errors.Is` `ErrScrollCheckMismatch` (which wraps `srderr.ErrInvalidInput`) when `option` is not one of `required.Options`, when the result's kind is not an ability check, or when its target differs from `required.DC`; it never panics. The reader picks the option; `AssessScrollReading` does not roll. The host MUST roll the option it passes: `CheckResult` has no ability or skill field (d20-tests.md FR-041), so only kind and target are validated, and a result rolled with another ability or with the Arcana bonus on a bare class option is not detected. Interpretation (section 9, Q2). | `doc/rules/architecture.md` §6 (untrusted host input is an error, `errors-and-results.md` FR-027); SRD 5.2.1, "D20 Tests", p. 6 | unhappy (invalid input) |
| FR-010 | A check whose outcome is success (total equal to or above the DC) MUST give `ScrollCheckPassed(terms)`, equal to `required.TermsFor(option)`, whose `SpellcastingAbility` is the chosen option's ability; a failure MUST give `ScrollSpellErased`. A total equal to the DC succeeds. | SRD 5.2.1, "Spell Scroll", p. 244; "D20 Tests", p. 6 | unhappy (boundary) |
| FR-011 | `ScrollSpellErased` means the spell is gone from the scroll "with no other effect": no spell is cast, no slot is expended, no Concentration begins, and the disposition is `DispositionErased` (spent: FR-016). | SRD 5.2.1, "Spell Scroll" ("the spell disappears from the scroll with no other effect"), p. 244 | unhappy (state error) |
| FR-012 | The check is made when the casting is completed, not when it is begun. A casting interrupted first makes no check (FR-016). The host order, and what a failed check does to a casting in progress, is FR-036. Interpretation (section 9, Q3). | SRD 5.2.1, "Spell Scroll" ("to determine whether you cast the spell"), p. 244 | unhappy (rule interaction) |
| FR-013 | `AssessScrollReading` of a scroll whose `MagicItemUseState` is `Consumed` MUST return `ScrollAlreadyUsed` (magic-item-rules.md FR-024) and evaluate nothing else. | SRD 5.2.1, "Using a Scroll" ("the scroll can't be used again"), p. 205 | unhappy (state error) |

### 4.3 Terms of the cast and the scroll afterwards

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `ScrollCastTerms` MUST carry `SpellID`, `CastAtLevel` (the scroll's spell level; no slot is expended), `SaveDC`, `AttackBonus`, `SpellcastingAbility`, `CasterLevel` (closed variant set `ScrollCasterLevel`, an interface with an unexported marker method: `ScrollLowestCasterLevel`, wrapping `ItemCasterLevel.lowest` of magic-item-rules.md FR-025, for a found scroll; `ScribedCantripCasterLevel` for a scribed cantrip, FR-025) and `WaivedComponents`. `ScrollCastTerms` specialises `SpellFromItem` (magic-item-rules.md FR-025, FR-026) for the scroll and adds the DC fields it lacks: `CastAtLevel` and the lowest caster level agree with it, the components are `ComponentsPerDescription` (Verbal and Somatic stay), and a scribed cantrip is the one case `SpellFromItem.CasterLevel` cannot express (p. 103 supersedes p. 206 there). For a found scroll `WaivedComponents` MUST be exactly `{material}`; Verbal and Somatic components still apply. Interpretation (section 9, Q4). | SRD 5.2.1, "Spell Scroll" ("without Material components"), p. 244; "Spells Cast from Items", p. 206; "Casting without Slots", p. 104 | happy |
| FR-015 | The cast MUST keep the spell's own casting time and, for a spell that requires Concentration, the user MUST concentrate; this spec adds no casting-time rule and the scroll cast is not a Ritual. | SRD 5.2.1, "Spell Scroll" ("the spell's normal casting time"), p. 244; "Spells Cast from Items", p. 206 | happy |
| FR-016 | `SettleScrollReading(ScrollCastProgress)` MUST return `DispositionConsumed` for `ProgressCompleted`, and `DispositionRetained` for `ProgressInterrupted` and for `ProgressNotStarted` (a rejected `CastSpell`). `DispositionConsumed` and `DispositionErased` both mean the host applies magic-item-rules.md FR-023; `DispositionRetained` leaves the scroll `Intact`. | SRD 5.2.1, "Spell Scroll" ("Once the spell is cast, the scroll crumbles to dust. If the casting is interrupted, the scroll isn't lost"), p. 244; "Spell Scroll (Cantrip)", p. 99 | unhappy (state error) |
| FR-017 | The progress is derived by the host from the cast result: a `SpellCast` event gives `ProgressCompleted`; a `CastingFailed` event (Concentration broken during a longer casting) gives `ProgressInterrupted`; a rejection or a casting never begun gives `ProgressNotStarted`. This spec reads no event type itself. | SRD 5.2.1, "Spell Scroll", p. 244; casting-and-concentration.md FR-072, FR-083 | unhappy (rule interaction) |
| FR-018 | Casting from a scroll MUST NOT count as a spell cast with a slot for "One Spell with a Spell Slot per Turn", and MUST NOT need the spell prepared or on a prepared list. | SRD 5.2.1, "Casting without Slots", "One Spell with a Spell Slot per Turn", pp. 104-105 | unhappy (rule interaction) |

### 4.4 Use Magic Device (any Spell Scroll)

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-019 | `AnyScrollRoute(ability, ReliableUpToLevel, CheckSkill)` MUST make every scroll a candidate whatever its spell lists; it is reliable when the spell level is at most `ReliableUpToLevel`, and otherwise its check option is `(ability, CheckSkill)`. Use Magic Device is `AnyScrollRoute(Intelligence, 1, Arcana)`, built by `class-rogue.md`. | SRD 5.2.1, "Level 13: Use Magic Device" (Scrolls), p. 64 | happy |
| FR-020 | With several routes the scroll MUST be reliable if any route is reliable, and the ability in the terms is that of the first reliable route in route order; the other routes only add options when none is reliable (FR-008). | SRD 5.2.1, "Spell Scroll", p. 244; "Level 13: Use Magic Device", p. 64; interpretation (section 9, Q5) | unhappy (rule interaction) |
| FR-021 | A failed Use Magic Device check MUST give `ScrollSpellErased` (the SRD says the scroll "disintegrates"); it is the same disposition as FR-011. | SRD 5.2.1, "Level 13: Use Magic Device", p. 64 | unhappy (state error) |
| FR-022 | `AnyScrollRoute` waives the spell-list requirement only. The reader still has to be able to read the scroll (FR-003). Interpretation (section 9, Q6). | SRD 5.2.1, "Using a Scroll", p. 205; "Level 13: Use Magic Device", p. 64 | unhappy (rule interaction) |

### 4.5 Scribed scrolls

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-023 | `SpellScroll.origin` MUST be a closed variant set (interface `SpellScrollOrigin` with an unexported marker method, implemented only by: `FoundScroll` (terms from table T1, FR-002) and `ScribedScroll(ScribedScrollStats)`). A scribed scroll's cast MUST use the stats' Save DC and Attack Bonus in place of the table values for every spell level, including a scribed cantrip or level 1 scroll whose stats differ from 13 and +5. Interpretation: p. 103 governs scrolls with a known scribe, p. 99 and p. 244 the found ones (Q13). | SRD 5.2.1, "Scribing Spell Scrolls" ("The scroll's spell uses your spell save DC and spell attack bonus"), p. 103; "Exceptions Supersede General Rules", p. 5 | happy |
| FR-024 | `NewScribedScrollStats(stats SpellcastingStats, level CharacterLevel)` MUST take the DC and attack bonus of the class that prepared the spell, as they were when the scroll was completed, and keep them fixed on the scroll (later level-ups of the scribe do not change them). Interpretation (section 9, Q7). | SRD 5.2.1, "Prerequisites for the Scribe", p. 103 | happy |
| FR-025 | For a scribed **cantrip** `CasterLevel` MUST be `ScribedCantripCasterLevel(level)`, the scribe's character level when the scroll was completed; for a scribed spell of level 1 or higher it MUST be `ScrollLowestCasterLevel`. A found cantrip is always `ScrollLowestCasterLevel`. | SRD 5.2.1, "Scribing Spell Scrolls", "Cantrips", p. 103; "Spells Cast from Items", p. 206 | unhappy (boundary) |
| FR-026 | The reader of a scribed scroll uses their own routes for FR-003 to FR-012 and FR-019 to FR-022; only DC, attack bonus and cantrip level come from the scribe. | SRD 5.2.1, "Spell Scroll", p. 244; "Scribing Spell Scrolls", p. 103 | unhappy (rule interaction) |
| FR-027 | `AssessScribing(ScribeFacts)` MUST be eligible only when the scribe has proficiency in the Arcana skill or with Calligrapher's Supplies, has the spell prepared on each day of the inscription (a known cantrip counts, interpretation Q8), and has at hand any Material components. It MUST list every problem (`NoProficiency`, `SpellNotPrepared`, `MissingMaterials`) in that fixed order. An eligible result reports whether the spell consumes its materials, which are then consumed only on completion of the scroll. | SRD 5.2.1, "Prerequisites for the Scribe", p. 103 | unhappy (invalid input) |

### 4.6 Copying a scroll into a spellbook

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-028 | Only a **Wizard** spell on a scroll can be copied. `AssessScrollCopying(scroll, MagicItemUseState)` MUST return `ScrollAlreadyUsed` for a `Consumed` scroll (FR-013), then `ScrollNotCopyable(NotAWizardSpell)` when the spell's `Spell.Classes` lacks `ClassID` `wizard`, and `ScrollNotCopyable(Cantrip)` for a cantrip. The cantrip branch is an interpretation (Q11): p. 78 puts only level 1+ Wizard spells in a spellbook, and p. 244 does not override that. Nothing is destroyed on these and the scroll is retained. | SRD 5.2.1, "Copying a Scroll into a Spellbook", p. 244; "Copying a Spell into the Book", p. 78 | unhappy (invalid input) |
| FR-029 | A copyable scroll MUST give `ScrollCopyCheckRequired` with the fixed option Intelligence (Arcana) and `DC` equal to 10 plus the spell level. | SRD 5.2.1, "Copying a Scroll into a Spellbook", p. 244 | happy |
| FR-030 | `ResolveScrollCopy(required, CheckResult) (ScrollCopyResolved, error)` MUST give `ScrollCopyResolved(SpellCopied, ScrollDestroyed)` with `ScrollDestroyed` always true and `SpellCopied` true exactly when the check succeeds; a total equal to the DC succeeds. A target other than `required.DC` or a kind other than ability check returns an error that is `errors.Is` `ErrScrollCheckMismatch`. | SRD 5.2.1, "Copying a Scroll into a Spellbook" ("Whether the check succeeds or fails, the Spell Scroll is destroyed"), p. 244 | unhappy (boundary) |
| FR-031 | The Wizard's own conditions for copying (a level the Wizard can prepare, a spell not already in the book, the time and cost per level) are not repeated here. The host order is FR-037; a rejection by `class-wizard.md` FR-036 means no attempt is made and the scroll is retained. Interpretation (section 9, Q9). | SRD 5.2.1, "Expanding and Replacing a Spellbook", p. 78 | unhappy (rule interaction) |

### 4.7 Cross-cutting

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | `NewScrollReaderProfile` MUST return an error (satisfying `errors.Is(err, srderr.ErrInvalidInput)`; `MustScrollReaderProfile` panics with it) for a `HighestSlotLevel` or `GrantedCastLevel` outside 0 to 9, an `AnyScrollRoute.ReliableUpToLevel` outside 0 to 9, or two class routes with the same `ClassID`; its route list is copied on construction and on read. | `doc/rules/architecture.md` §3, §6 | unhappy (invalid input) |
| FR-033 | A host casts the scroll's spell with `CastSpell` and `CastModeSpellScroll` (casting-and-concentration.md amendment, section 9 Q10): no slot, no preparation, no Ritual and no Material check, but `CannotCastSpells`, `UntrainedArmor`, `CastingAlreadyInProgress`, `TriggerMet` and the Verbal and Somatic checks apply, and `CastAtLevel` is the terms' level. `CastModeFeatureGranted` is not reused for it. | SRD 5.2.1, "Casting without Slots" (Magic Items), p. 104; "Spells Cast from Items", p. 206 | happy |
| FR-034 | Every operation MUST be deterministic and mutate none of its inputs; same inputs give equal results. No operation reads dice, time or a port. | `doc/rules/architecture.md` §3, §4 | unhappy (determinism) |
| FR-035 | Every type a result names MUST be exported from package `character`, with no generic variant name (shared-types-and-ownership.md FR-013, FR-017). | shared-types-and-ownership.md FR-017 | happy |
| FR-036 | Host order for a scroll that needs a check: `ScrollCheckRequired.TermsFor(option)` gives the terms to begin with. A casting time of one action or less has no gap, so the host rolls the check first and calls `CastSpell` only on `ScrollCheckPassed`. A longer casting is begun with the provisional terms, and on completion the host rolls the check; only on `ScrollCheckPassed` does it apply `CompleteCasting` and emit `SpellCast`. On `ScrollSpellErased` it ends the casting in progress with no slot, no spell effect and no Concentration, as an interrupted casting ends (casting-and-concentration.md FR-072), and emits no `SpellCast`. Interpretation (Q3). | SRD 5.2.1, "Spell Scroll", p. 244; casting-and-concentration.md FR-072, FR-083 | unhappy (rule interaction) |
| FR-037 | Copy order for a host: (1) `CopySpellIntoBook` rejections, `class-wizard.md` FR-036, otherwise no attempt; (2) `AssessScrollCopying`; (3) the Intelligence (Arcana) check; (4) `ResolveScrollCopy`; (5) `CopySpellIntoBook` is applied only when `SpellCopied` is true. The 2 hours and 50 GP per spell level (`class-wizard.md` FR-035) are spent whether or not the check succeeds, since the transcription is attempted, and the scroll is destroyed either way. Interpretation (Q9). | SRD 5.2.1, "Copying a Scroll into a Spellbook", p. 244; "Copying a Spell into the Book", p. 78 | unhappy (rule interaction) |
| FR-038 | `ScrollReaderProfile.GrantedCastLevel` (default 0) MUST be the highest spell level the reader can cast without a slot through a class feature that grants a regular cast (Mystic Arcanum), which the host raises; this spec reads no feature. Interpretation (Q1). | SRD 5.2.1, "Level 11: Mystic Arcanum", p. 72; "Spell Scroll", p. 244 | unhappy (boundary) |

Withdrawn ids: none.

### Case tables

Profiles. W5: Wizard route (Intelligence), `HighestSlotLevel` 3. R13: `AnyScrollRoute(Intelligence, 1, Arcana)`, no slots (0). MC: Wizard route (Intelligence) and Cleric route (Wisdom), `HighestSlotLevel` 3. All read a language; found scrolls; use state `Intact`.

| Label | Reader | Scroll spell (level, lists) | Expected | Source | Path |
|---|---|---|---|---|---|
| A1 | W5 | 3, Wizard | `ScrollReady`, DC 15, +7, Intelligence | p. 244 | happy |
| A2 | W5 | 4, Wizard | `ScrollCheckRequired` DC 14, [Int] | p. 244 | unhappy (boundary) |
| A3 | W5 | 9, Wizard | DC 19, [Int] | p. 244 | unhappy (boundary) |
| A4 | W5 | cantrip, Wizard | `ScrollReady`, DC 13, +5, `ScrollLowestCasterLevel` | pp. 99, 244 | happy |
| A5 | W5 | 2, Cleric only | `ScrollNotOnSpellList` | p. 244 | unhappy |
| A6 | no routes | 1, Wizard | `ScrollNotOnSpellList` | p. 244 | unhappy (empty and minimal) |
| A7 | slots 0, Wizard route | 1, Wizard | DC 11 [Int] | p. 244 | unhappy (boundary) |
| A8 | slots 0, Wizard route | cantrip, Wizard | `ScrollReady` | p. 244 | unhappy (boundary) |
| A9 | R13 | cantrip, Cleric only | `ScrollReady`, Intelligence | p. 64 | happy |
| A10 | R13 | 1, Bard only | `ScrollReady` | p. 64 | unhappy (boundary) |
| A11 | R13 | 2, Cleric only | DC 12, [Int + Arcana] | p. 64 | unhappy (boundary) |
| A12 | R13 | 9, Wizard | DC 19, [Int + Arcana] | p. 64 | unhappy (boundary) |
| A13 | MC | 3, Cleric only | `ScrollReady`, Wisdom | pp. 25-26, 244 | unhappy (rule interaction) |
| A14 | MC | 4, Wizard and Cleric | DC 14, [Int, Wis] | p. 244 | unhappy (rule interaction) |
| A15 | R13 plus Wizard route, slots 3 | 3, Wizard | `ScrollReady`, Intelligence (Wizard route first) | p. 244 | unhappy (rule interaction) |
| A16 | R13 plus Wizard route, slots 3 | 4, Wizard | DC 14, [Int, Int + Arcana] | p. 64 | unhappy (rule interaction) |
| A17 | W5, cannot read | 1, Wizard | `ScrollNotReadable` | p. 205 | unhappy (state error) |
| A18 | W5, description restricts | 1, Wizard | `ScrollNotReadable` | p. 205 | unhappy (state error) |
| A19 | W5, scroll consumed | 1, Wizard | `ScrollAlreadyUsed` | p. 205 | unhappy (state error) |
| A20 | W5, scribed 15 / +7 | 2, Wizard | `ScrollReady`, DC 15, +7 (not the table's 13, +5) | p. 103 | happy |
| A21 | Warlock route (Charisma), `HighestSlotLevel` 5, `GrantedCastLevel` 6 | 6, Warlock | `ScrollReady` | pp. 72, 244 | unhappy (boundary) |
| A22 | as A21 | 7, Warlock | `ScrollCheckRequired` DC 17, [Cha] | pp. 72, 244 | unhappy (boundary) |
| A23 | W5, scribed 16 / +8 | 1, Wizard | `ScrollReady`, DC 16, +8, asserted unequal to the table's 13, +5 | p. 103 | unhappy (rule interaction) |

Checks, settling and scribing (one test per row):

| Label | Input | Expected | Source | Path |
|---|---|---|---|---|
| C1 | DC 14, total 14 | `ScrollCheckPassed` | p. 244 | unhappy (boundary) |
| C2 | DC 14, total 13 | `ScrollSpellErased`, disposition `DispositionErased` | p. 244 | unhappy (boundary) |
| C3 | option not in the required list | `ErrScrollCheckMismatch` | FR-009 | unhappy (invalid input) |
| C4 | result target 13 against DC 14; a saving-throw kind | `ErrScrollCheckMismatch` each | FR-009 | unhappy (invalid input) |
| C5 | Wizard 5/Cleric 1 passes with option Wis | terms ability Wisdom | FR-010 | unhappy (rule interaction) |
| S1 | `ProgressCompleted` | `DispositionConsumed` | p. 244 | happy |
| S2 | `ProgressInterrupted` | `DispositionRetained` | p. 244 | unhappy (state error) |
| S3 | `ProgressNotStarted` | `DispositionRetained` | p. 244 | unhappy (state error) |
| B1 | scribed level 1, Wizard DC 15 +7, character level 11 | terms DC 15, +7, `ScrollLowestCasterLevel` | p. 103 | happy |
| B2 | scribed cantrip, character level 11 | `ScribedCantripCasterLevel(11)` | p. 103 | unhappy (boundary) |
| B3 | scribe levels up to 12 after completion | same scroll, still DC 15 +7, level 11 | FR-024 | unhappy (rule interaction) |
| B4 | no Arcana or Calligrapher's proficiency, spell not prepared, no materials | all three problems, in that order | p. 103 | unhappy (invalid input) |
| B5 | Calligrapher's Supplies only, all else met | eligible | p. 103 | happy |
| P1 | Wizard level 1 spell, total 11 (DC 11) | `SpellCopied` true, destroyed true | p. 244 | unhappy (boundary) |
| P2 | Wizard level 9 spell, total 18 (DC 19) | `SpellCopied` false, destroyed true | p. 244 | unhappy (boundary) |
| P3 | cantrip; Cleric spell | `ScrollNotCopyable(Cantrip)`; `(NotAWizardSpell)`; scroll retained in both | p. 78, p. 244 | unhappy (invalid input) |
| P4 | Wizard level 2 spell, scroll `Consumed` | `ScrollAlreadyUsed` | p. 205 | unhappy (state error) |
| P5 | scenario: Wizard level 2 spell, total below DC 12 | `SpellCopied` false; `CopySpellIntoBook` not applied; 4 hours and 100 GP still charged; scroll destroyed | FR-037 | unhappy (rule interaction) |
| P6 | scenario: level 3 spell above the Wizard's highest Spell Level | rejected by `CopySpellIntoBook`; no check; scroll retained | FR-037 | unhappy (rule interaction) |
| T2 | scenario: level 5 spell, reader W5, 3-round casting, total below DC 15 | terms from `TermsFor`; casting begun; on failure casting cleared, no `SpellCast`, no slot, no Concentration | FR-036 | unhappy (rule interaction) |

## 5. Model

Notation: `Name(a, b)` lists the fields of a struct type `Name`; a "closed" type is an interface with an unexported marker method, a doc comment naming every variant, and one struct per variant, switched exhaustively (`errors-and-results.md` FR-018). Constructors are `NewX` returning `(T, error)` with errors satisfying `errors.Is(err, srderr.ErrInvalidInput)`; `MustX` panics with that error. Slices and maps are copied on construction and on read.

Everything lives in package `character`, in files named `spell_scroll*.go` (`doc/rules/architecture.md` §3): immutable (unexported fields, value receivers, accessor methods without a Get prefix), pure, synchronous, no `context.Context` (no I/O). Optional values (`skill`, `checkSkill`, `anyScrollRoute`, `cantripCasterLevel`) are a documented zero value or a (value, ok) accessor; no pointers to small structs.

- **Value objects:** `SpellScroll(SpellID, spellLevel, spellClasses, origin)` with `NewSpellScroll(Spell, origin)`; closed variant set `SpellScrollOrigin` (`FoundScroll`, `ScribedScroll(ScribedScrollStats)`); `ScribedScrollStats(SaveDC, AttackBonus, cantripCasterLevel)`; `SpellScrollRow(level, rarity, SaveDC, AttackBonus)`; `ScrollReaderProfile(understandsWrittenLanguage, descriptionRestricts, HighestSlotLevel, GrantedCastLevel, classRoutes, anyScrollRoute)`; `ScrollClassRoute(classID, ability)`; `AnyScrollRoute(ability, ReliableUpToLevel, CheckSkill)`; `ScrollCheckOption(ability, skill)`; `ScrollCastTerms`; closed variant set `ScrollCasterLevel` (`ScrollLowestCasterLevel(ItemCasterLevel)`, `ScribedCantripCasterLevel(level)`); `ScribeFacts`; typed enums with `String()`: `ScrollDisposition` (`DispositionRetained`, `DispositionConsumed`, `DispositionErased`), `ScrollCastProgress` (`ProgressNotStarted`, `ProgressInterrupted`, `ProgressCompleted`), `ScribeProblem`, `ScrollNotCopyableReason` (`NotAWizardSpell`, `Cantrip`).
- **Closed results** (`<Operation>Result` naming, shared-types-and-ownership.md FR-017): `ScrollAssessmentResult` = `ScrollReady`, `ScrollCheckRequired(DC, Options, TermsFor)`, `ScrollNotReadable`, `ScrollNotOnSpellList`, `ScrollAlreadyUsed`; `ScrollCheckResolutionResult` = `ScrollCheckPassed`, `ScrollSpellErased`; `ScribeAssessmentResult` = `ScribeEligible(ConsumesMaterialsOnCompletion)`, `ScribeIneligible(problems)`; `ScrollCopyAssessmentResult` = `ScrollCopyCheckRequired`, `ScrollNotCopyable`, `ScrollAlreadyUsed` (the one type, implementing the marker of both result interfaces); `ScrollCopyResolved`.
- **Domain functions** (stateless package functions, no service types): `SpellScrollTableRow`, `AssessScrollReading`, `ResolveScrollCheck`, `SettleScrollReading`, `AssessScribing`, `NewScribedScrollStats`, `AssessScrollCopying`, `ResolveScrollCopy`.
- **Invariants:** FR-032 in the profile constructor; table total over `SpellLevel` (an out-of-range level is unrepresentable). Expected rule outcomes are results; untrusted host input that breaks a precondition (FR-009, FR-030, FR-032) is a returned error satisfying `errors.Is(err, srderr.ErrInvalidInput)`. `ErrScrollCheckMismatch` is the package sentinel for FR-009 and FR-030 (wraps `srderr.ErrInvalidInput`). No `*srderr.PortError` (no port), no panic outside `MustScrollReaderProfile`.
- **Events:** none. The consumed scroll's event is `MagicItemConsumed` (magic-item-rules.md FR-023); the cast's events are `casting-and-concentration.md`'s.

## 6. Interface

No use case and no port (shared-types-and-ownership.md FR-023, FR-044): the functions are pure and a host composes them. The sequence is: build the profile; `AssessScrollReading`; on `ScrollReady` call `CastSpell` with `CastModeSpellScroll` and the terms (FR-033); on `ScrollCheckRequired` follow FR-036 (check before the cast for a casting time of one action or less; for a longer casting begin with `TermsFor(option)` and make the check at completion, applying `CompleteCasting` only on `ScrollCheckPassed`); then `SettleScrollReading`. The ability check is rolled by the host with `d20.Resolve` or `d20.Resolver` (d20-tests.md) for the option it passes, and arrives as a `CheckResult`; the spec validates only kind and target (FR-009). Copying follows FR-037.

Package `character` exports every type and function named in section 5 (FR-035). `character/charactertest` gains no double.

## 7. Observability

Silent by design: pure domain code only, no I/O and no use case, so there is nothing to time or count (`doc/rules/observability.md` §3). This spec adds no signal. The applied Q10 amendment adds the value `scroll` to the catalogued `cast_source` attribute (`shared-types-and-ownership.md` FR-041 convention), already listed in `doc/observability/signals.md` and `casting-and-concentration.md`. A scroll that is erased, retained or copied is visible through the host's own `MagicItemConsumed` handling. Failure modes (a wrong DC, a scroll consumed on interruption) are rule bugs, caught by the tests, not by operators.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001, FR-002 | unit | Table T1, ten rows, one table-driven test; found terms ignore the reader's numbers (A1, A4) |
| FR-003 to FR-006 | unit | A1 to A10, A17 to A19; collaborators are real value objects, no double |
| FR-007, FR-008, FR-038 | unit | A2, A3, A7, A8, A13, A14, A21, A22; bare ability check carries no skill |
| FR-009 to FR-013 | unit | C1 to C5, A19; error rows assert `errors.Is(err, ErrScrollCheckMismatch)` and `errors.Is(err, srderr.ErrInvalidInput)`, never the message text |
| FR-014, FR-015 | unit | Terms fields; `WaivedComponents` is exactly `{material}`; casting time and Concentration are asserted in the scenario below |
| FR-016, FR-017, FR-018 | unit | S1 to S3; one scenario test composes `AssessScrollReading`, `CastSpell` (`spellScroll`), `SettleScrollReading` over `completed`, `interrupted`, and a rejected cast |
| FR-019 to FR-022 | unit | A9 to A12, A15, A16; failed check gives `erased` |
| FR-023 to FR-026 | unit | A20, A23, B1 to B3; FR-001 also asserts equality with the bundled `spell_scroll` table (golden) |
| FR-027 | unit | B4, B5 |
| FR-028 to FR-031, FR-036, FR-037 | unit, scenario | P1 to P6, T2; FR-031 is a scenario test with `class-wizard.md` `CopySpellIntoBook` in the order the host uses |
| FR-032, FR-034, FR-035 | unit, architecture | Constructor errors; routes copied; determinism; `internal/archtest` import scan (no generic names, every named type exported, no `combat` import) |

Mutation testing (`doc/rules/testing.md` §8) applies: changed code. Boundary rows (C1, C2, A2, A7, A11, P1, P2) kill the `>=`, `>` and `+ 10` mutants. Every FR appears above. Not covered: FR-033 beyond the scenario test.

## 9. Open questions and decisions

None escalated. Each decision rests on the SRD text or a package-design rule; web research (community discussion) was used only to judge which reading is normal and no non-SRD text is copied.

| Question | Decision | Status |
|---|---|---|
| Q1. What is "a higher level than you can normally cast"? | The highest slot level in the reader's maximum slots, both pools, whole character. The SRD says slots of every pool pay for any class's spells (pp. 25-26, FR-035 of `spellcasting.md`), so "can normally cast" is a character fact, not a class fact. Interpretation. Granted regular casts (Mystic Arcanum) are 'normal' casts without a slot, so the host raises `GrantedCastLevel` (FR-038); one-off or item-granted casts are not counted. Interpretation. | settled |
| Q2. Which class's ability for the check if the spell is on several lists? | The reader picks (`ScrollCheckOption`); the host passes the one rolled. Same as picking the casting class (spellcasting.md FR-005, FR-037). | settled |
| Q3. Check at the start or at the end of the casting? | At completion: the check decides "whether you cast the spell", and an interrupted casting loses nothing (p. 244). Alternative (at the start) would erase a spell before the casting time passed. A failed check on a longer casting clears the casting with no effect (FR-036). Interpretation. | settled |
| Q4. Are Verbal and Somatic components waived? | No, only Material. The general rule (p. 206) waives all components "unless the item's description notes otherwise", and the Spell Scroll description names Material alone, so the description governs. Community discussion is split; the narrower reading is the one the text of the entry can carry. Interpretation. | settled |
| Q5. Rogue plus caster routes | Reliable if any route is; options otherwise listed per route, so the Rogue/Wizard picks the better modifier. Not a double penalty. | settled |
| Q6. Does Use Magic Device waive reading? | No, only the spell list ("use any Spell Scroll"). p. 205 still applies. | settled |
| Q7. Scribed numbers fixed or current? | Fixed at completion: the scroll is an item the scribe leaves behind, and "works as if the caster were your level" is read at the time of scribing. Interpretation. | settled |
| Q8. Cantrip "prepared" for scribing | A known cantrip counts; cantrips are never prepared (spellcasting.md FR-044) and the SRD requires "the spell prepared". Interpretation. | settled |
| Q9. Do the p. 78 copy time and cost apply to a scroll? | Yes, and they are spent whether or not the check succeeds: the transcription is attempted before the check, and p. 244 destroys the scroll either way (FR-037). Interpretation. | settled |
| Q10. How does the cast reach `CastSpell`? | Amendment applied in `casting-and-concentration.md` FR-143 (new `CastModeSpellScroll`, `CastSourceScroll`, `cast_source` value `scroll`). `CastModeFeatureGranted` is not reused: it would wrongly still need `HasMaterials`. | decided; amendment applied (casting-and-concentration.md FR-143) |
| Q11. Can a cantrip scroll be copied? | No. p. 244 says "A Wizard spell" can be copied and a cantrip is a Wizard spell, but p. 78 puts only level 1+ Wizard spells in a spellbook, so the narrower rule governs. Scroll retained. Interpretation. | settled |
| Q12. Why a code table when content holds one? | package `character` reads no content source; FR-001 pins the constant to the bundled `spell_scroll` entry with a test, so drift fails CI. | settled |
| Q13. Scribed cantrip or level 1 scroll versus the 13/+5 of p. 99/p. 244? | The scribe's numbers (p. 103) govern scrolls with a known scribe; the printed 13/+5 describes found scrolls whose creator is unknown. Exceptions Supersede General Rules (p. 5). Interpretation. | settled |

## 10. Changelog impact

`CHANGELOG.md`, Added: "Spell Scroll use: reading rules, the level check, the scroll table, scribed scrolls and copying a scroll into a spellbook (pure domain functions)." Not **BREAKING** (new API).

Status is draft. Already made: `spell-scroll-use` at rank 15 (`shared-types-and-ownership.md` section 4.11), its `documentation_index.md` row, the Phase 3 entry in `roadmap.md`, and `CastModeSpellScroll` / `CastSourceScroll` in `casting-and-concentration.md` (FR-106, FR-143, with `scroll` and `duplicated` in its `cast_source` values). Still outstanding in a file this spec does not own:

- `doc/observability/signals.md`: add `scroll` to the `cast_source` values (`shared-types-and-ownership.md` FR-041 convention).
- `class-wizard.md`: none needed; the host order is FR-037.
