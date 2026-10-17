# Monster stat-block actions: Multiattack, attack and save notation, Bonus Actions, Reactions

| | |
|---|---|
| Status | draft |
| Context | `combat` |
| Roadmap phase | 4 |
| Depends on | `errors-and-results.md`, `dice-and-rolls.md`, `modifiers-and-advantage.md`, `observability-ports.md`, `d20-tests.md`, `content-schema-and-decoder.md`, `content-monsters.md`, `combat-foundations.md`, `hit-points-and-damage.md`, `conditions.md`, `attacks.md`, `initiative-and-actions.md`, `monster-limited-use.md` |

Types ship with the `combat` phase (`shared-types-and-ownership.md` FR-014). Rank 27 in table 4.11 (amendment A-1, defined in section 5.1 with A-2 to A-9): above `monster-limited-use` (26), the highest dependency, and below `combat-action-resolution` (29), which consumes this spec and so cannot be listed here (`shared-types-and-ownership.md` FR-056). `conditions.md` (22) is listed for `ConditionActionAvailability` (FR-008). `attack-and-help-actions.md` (24) is related by prose only; nothing of it is imported.

## 1. Purpose and scope

`content-monsters.md` stores a stat block as data: attack notation, saving throw notation, damage notation, effect clauses and prose (its FR-036 to FR-041). `monster-limited-use.md` counts uses. `attacks.md` resolves one attack and `initiative-and-actions.md` owns the turn economy. Nothing says what a stat-block entry **does** when a monster uses it: how Multiattack allots attacks, what the printed attack bonus is, which of the number and the die expression is rolled, which clauses apply on a hit or a miss, how a saving throw effect treats a success ("Half damage only"), what a Bonus Action or Reaction entry needs, and how the "immune for 24 hours" rider works. This spec is that owner (`shared-types-and-ownership.md` FR-048: the topic-specific spec owns the mechanics). It is pure state-in, state-out rules over `Monster`, plus one small use case, `RollStatBlockDamage`, because rolling damage dice needs `dice.Roller` (`shared-types-and-ownership.md` FR-023, FR-044).

It composes, never repeats: `attacks.md` rolls the attack and applies the Critical Hit dice rule; `hit-points-and-damage.md` owns `Damage`, `DamageAdjustment` (`HalfDamage`) and rounding; `monster-limited-use.md` owns uses, Recharge and Legendary Action timing; `initiative-and-actions.md` owns the action, Bonus Action and Reaction slots. This spec produces the typed inputs those take and the outcome of the stat-block-specific rules between them. Hosts and `combat-action-resolution.md` (amendment A-4) do the wiring.

**Out of scope:**

- Rolling the attack, the Critical Hit rule, Cover and range (`attacks.md`); applying damage, Resistance, Immunity (`hit-points-and-damage.md`); applying a condition named in a clause's prose (the host reads the verbatim text and uses `conditions.md`).
- Spells a monster casts through Spellcasting: resolved by `spell-effects.md`. Only the use accounting (`spendSpell`) is touched, to gate a Multiattack replacement (FR-018).
- Repeated saves ("repeats the save", "after every 24 hours"), curses and other effects that last or recur: `active-effects.md`. The 24-hour **immunity** after a success is here (FR-039 to FR-043); the effect on a failure is not.
- Summoned creatures whose Multiattack count depends on a spell's level or the summoner's Proficiency Bonus (SRD 5.2.1, "Spirit" spells, "Summon" spells): outside pp. 258-364; `content-spells.md` and the host build a fixed-count plan for the level chosen.
- Lair actions: SRD 5.2.1 has none (`monster-limited-use.md` section 1). "Running a Monster" (p. 255) is advice to the Game Master, not a rule; no requirement (FR-019).
- Anything outside SRD 5.2.1. Nothing core was found missing from the PDF.

## 2. Source

- SRD 5.2.1, "Stat Block Overview" and "Parts of a Stat Block", pp. 254-257: "Actions", "Attack Notation", "Saving Throw Effect Notation" (p. 256); "Damage Notation", "Multiattack", "Spellcasting", "Bonus Action", "Reactions", "Legendary Actions", "Limited Usage" (p. 257); "Running a Monster" (p. 255).
- SRD 5.2.1, "Actions", "Bonus Actions", "Reactions", pp. 9-10; "Damage Rolls", "Critical Hits", "Saving Throws and Damage" (damage against multiple targets), "Half Damage", p. 16; "Saving Throws", p. 7; "Bonus Action", "Attack [Action]", p. 177; "Reaction", p. 186; "Weapon Attack", p. 191.
- Stat blocks cited by name and page: "Aboleth", p. 258; "Assassin", p. 260; "Balor", p. 261; "Bandit", p. 262; "Barbed Devil", p. 262; "Chimera", p. 273; "Chain Devil", p. 273; "Cloaker", p. 274; "Clay Golem", p. 274; "Ghast", p. 287; "Ghost", pp. 287-288; "Gladiator", p. 289; "Bulette", p. 272; "Guardian Naga", p. 296; "Homunculus", p. 299; "Mummy", p. 309; "Steam Mephit", p. 308; "Merfolk Skirmisher", p. 308; "Death Dog", p. 279; brass dragon blocks, pp. 268-269; "Animated Armor", p. 259; "Awakened Shrub", p. 260; "Priest", p. 316; "Bugbear Stalker", p. 271; "Bugbear Warrior", p. 272.
- Package design: `doc/rules/architecture.md` §2-§4, §6; `observability.md` §2, §3, §5; `testing.md` §4, §5, §8; `shared-types-and-ownership.md` FR-017, FR-019, FR-021, FR-044, FR-048, FR-056, FR-062; `combat-foundations.md`.
- Research (informs judgement only; nothing copied): SRD 5.2.1 p. 257 wording is "can make", so a multiattack entry describes what a creature **can** do, not a mandatory script (interpretation; FR-019); no source rules on a Critical Hit when the printed number is used (Q-3).

## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| Multiattack | `MultiattackPlan` (`content`, amendment A-2), `MultiattackProgress` | "details the attacks a creature can make, as well as any additional abilities it can use, as part of the Attack action" (p. 257). Plan = `MultiattackVariant`s; variant = `AttackSlotGroup`s, `AdditionalUse`s and `ReplacementRule`s. |
| "replace one attack with" | `ReplacementRule`, `ReplacementOption`, `ReplacementScope` (`anyAttack`, `oneAttackPart`), `ReplacementLimit` (`once`, `anyNumber`) | Package terms for "replace one attack with a use of ...", "replace any attack ..." (pp. 264-296 stat blocks). |
| Attack Notation, bonus | `StatBlockAttack` (`content-monsters.md` FR-037) | The printed bonus is the whole attack modifier (FR-024). |
| Damage Notation | `StatBlockDamage`, `DamageNotationChoice` (`number`, `dice`), `DamageNotationChoices` | "You decide whether to use the number or the die expression ... don't use both" (p. 257). |
| Hit, Miss, Hit or Miss | `EffectClause`, `EffectClauseLabel` (`content-monsters.md` FR-038), `AppliedClauses` | Which clauses a given outcome applies (FR-031). |
| Saving Throw Effect Notation, "Half damage only", plain "Half damage" | `StatBlockSavingThrow`, `SaveSuccessRider` (`content`, amendment A-3: `HalfDamageOnly`, `HalfDamage`, `ImmuneToPartForHours`), `SaveEffectDamageAdjustment` (`none`, `halve`) | p. 256 (the only notation defined); "Half Damage", p. 16 (plain "Success: Half damage", far more common). |
| immune "for 24 hours" | `SaveImmunityWindow`, `SaveImmunities` | Typed, scoped to one source creature and one part (FR-039). |
| Bonus Action, Reactions entries | `StatBlockUseRequest`, `StatBlockUseOutcome` | Produce initiative's `BonusActionGrant`, `FeatureReaction`, `FeatureActionChoice` (FR-002 to FR-004). |
| (the trigger happened) | `ReactionTriggerReport` (`occurred`, `notOccurred`) | Host input, as `AssistanceJudgement` in `attack-and-help-actions.md`. The trigger stays prose (`content-monsters.md` FR-041). |
| (action cost of a part) | `StatBlockActionCost` (`action`, `bonusAction`, `reaction`, `legendaryAction`, `passive`) | Package term (FR-001). |
| Legendary Actions, Limited Usage | `LegendaryActionWindow`, `LimitedUseState`, `LimitedUsePartRef`, `SpellUseRef` (`monster-limited-use.md`) | Reused. `LimitedUsePartRef` is the position of any part (amendment A-7). |

## 4. Requirements

Stable ids. Never renumber. "Part" is a `StatBlockPart` addressed by `LimitedUsePartRef(section, index)`.

### 4.1 Using a stat-block entry

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `StatBlockActionCost.Of(MonsterPartSection)` MUST return `passive` for Traits, `action` for Actions, `bonusAction` for Bonus Actions, `reaction` for Reactions and `legendaryAction` for Legendary Actions. A `passive` part is never "used" (FR-006). | SRD 5.2.1, "Traits", "Actions", p. 256; "Bonus Action", "Reactions", "Legendary Actions", p. 257 | happy |
| FR-002 | Using an Actions-section part other than Multiattack MUST consume the monster's action for the turn: the outcome carries the `FeatureActionChoice` of `initiative-and-actions.md` FR-049 with the part's `FeatureID` (FR-005) and the supplied `GrantedAction` (default `NoStandardAction`). The monster MAY instead take any standard action; that is the tracker's choice, not a stat-block use. | SRD 5.2.1, "Actions" (monster section), p. 256 ("take the actions in this section or take one of the actions available to all creatures") | happy |
| FR-003 | Using a Bonus Actions-section part MUST carry the `BonusActionGrant` (`initiative-and-actions.md` FR-060, FR-151) naming the part's `FeatureID`; the monster has a Bonus Action only because its stat block lists one. A monster with no Bonus Actions part has no part to request, so `StatBlockPartNotFound`. One Bonus Action a turn stays the tracker's rule (`initiative-and-actions.md` FR-061). | SRD 5.2.1, "Bonus Action", p. 257; "Bonus Actions", p. 10; "Bonus Action", p. 177 | happy / unhappy (missing content) |
| FR-004 | Using a Reactions-section part MUST require `ReactionTriggerReport.Occurred`; `notOccurred` MUST be refused with `StatBlockReactionTriggerNotReported`, state unchanged. On success the outcome carries the `FeatureReaction` (`initiative-and-actions.md` FR-121) naming the part. The trigger text is never read or matched (`content-monsters.md` FR-041); one Reaction a round stays the tracker's rule (`initiative-and-actions.md` FR-066). | SRD 5.2.1, "Reactions", p. 257; "Reactions", p. 10 ("immediately after its trigger") | unhappy (state error) |
| FR-005 | A part's `FeatureID` MUST be `<monster id>__<part id>`, the part id being the part name derived as `content-monsters.md` FR-002 derives a `MonsterID` (lower-case, apostrophes dropped, each other non-`a-z0-9` run one underscore, ends trimmed). The two-underscore joint cannot occur inside a part id, so the id determines the part name's derived form. Two parts of one monster MAY derive the same part id (a name repeated across sections, or names differing only in punctuation); the id is not section-qualified, to keep it inside the 64-character limit. `StatBlockFeatureID(monster, part)` MUST therefore return an error that is a `*StatBlockFeatureIDRejected` (matching `srderr.ErrInvalidInput`) carrying the reason (`invalidName`: no letter or digit; `tooLong`: over the limit of `content-schema-and-decoder.md` FR-030; `duplicate`: another part of the monster derives the same id); there is no separate throwing form. Hosts check `StatBlockFeatureIDProblems(monster)` once per monster. | package design; SRD 5.2.1, "Stat Block Overview", p. 254 | unhappy (invalid input) |
| FR-006 | A request naming a part in the wrong section for the kind of use (a Trait used as an action, a Reaction requested as a Bonus Action) or a `passive` part MUST be refused with `StatBlockPartInWrongSection`; a ref with no part MUST be refused with `StatBlockPartNotFound`. Neither changes state. | SRD 5.2.1, "Traits", p. 256 ("active at all times or in certain situations") | unhappy (invalid input) |
| FR-007 | Before authorising a use, `LimitedUseState` MUST be consulted: a part with a limit (`X/Day`, Recharge, Recharge after a Short or Long Rest) is spent exactly once through `spend` (a Spellcasting spell through `spendSpell`) and the outcome carries the new `LimitedUseState` and its events; a spent part MUST be refused with `StatBlockPartUsageExhausted`. A `notLimitedUse` refusal from `spend` is not a refusal here: an unlimited part has nothing to spend and the state is returned unchanged. | SRD 5.2.1, "Limited Usage", p. 257 | happy / unhappy (boundary) |
| FR-008 | Using a Legendary Actions part MUST first pass `takeLegendaryAction(window, availability)`, the `LegendaryActionWindow` and the `ConditionActionAvailability` (`conditions.md` FR-064) both read from the `StatBlockUseRequest` (a missing window or availability for this kind is an `srderr.ErrInvalidInput` error); its refusal MUST map one to one onto `StatBlockLegendaryUseRefused(reason)` carrying the `LegendaryActionRefusal`. The use costs the monster none of its action, Bonus Action or Reaction (the cost is `legendaryAction`) and no `FeatureActionChoice` is produced. The part's own limit (FR-007) is checked after the Legendary check, so a refused Legendary Action spends nothing. | SRD 5.2.1, "Legendary Actions", p. 257 | unhappy (rule interaction) |
| FR-009 | When several refusals apply the order MUST be `StatBlockPartNotFound`, `StatBlockPartInWrongSection`, `StatBlockReactionTriggerNotReported`, `StatBlockLegendaryUseRefused`, `StatBlockPartUsageExhausted`; the first is returned, state unchanged, no event. Package design, so the result does not depend on evaluation order. | package design | unhappy (rule interaction) |

### 4.2 Multiattack

The plan is data (amendment A-2); this spec specifies how it is spent. Table MA-1 gives the cases.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | `StartMultiattack(monster, combatant, limitedUse, opts...)` (options `WithGrantedAction`, `WithUsedThisTurn` taking a unique set of `LimitedUsePartRef`) MUST return the closed `MultiattackStartOutcome` (interface with an unexported marker method): `MultiattackStarted(progress, transition)` for a monster whose `multiattack` part carries a plan, the `StatBlockUseTransition.Economy` carrying the action consumed like FR-002 (the `FeatureActionChoice` names the Multiattack part, `granted` defaulting to `NoStandardAction`), or `MultiattackNotAvailable` for a monster with no Multiattack part or no plan. Each Multiattack taken starts its own progress; counts never carry between turns. | SRD 5.2.1, "Multiattack", p. 257 ("as part of the Attack action") | happy / unhappy (missing content) |
| FR-011 | The plan types MUST enforce: a plan holds at least one variant; a variant holds at least one `AttackSlotGroup`; a group has a count of at least 1 and a non-empty part set, and the groups of one variant have pairwise disjoint part sets (so every attack belongs to one group); an `AdditionalUse` holds one or more options and a `ReplacementRule` one or more options, a count of at least 1 and, for `oneAttackPart`, a named part in some group. Violations MUST return an error matching `srderr.ErrInvalidInput`; the decoder reports each as `constraintViolation` (amendment A-2). | SRD 5.2.1, "Multiattack", p. 257 | unhappy (invalid input) |
| FR-012 | At start the **candidate variants** are those whose precondition holds: a variant with no precondition always; a variant requiring a part "used this turn" (Clay Golem: three Slam attacks if it used Hasten) only when that part is in `usedThisTurn`. At least one variant is always a candidate or the plan is invalid (FR-011: a conditional variant needs an unconditional sibling, else an error matching `srderr.ErrInvalidInput`). | SRD 5.2.1, "Clay Golem", p. 274 | unhappy (rule interaction) |
| FR-013 | `recordAttack(part, limitedUse)` MUST succeed when some candidate variant has a group containing `part` with at least one slot left: it consumes one slot of that group, narrows the candidates to the variants that allow everything recorded so far (FR-015), gates the part through FR-007 and returns `MultiattackStepRecorded` with one `MultiattackAttackRecorded` event (part, 1-based attack number). Otherwise `MultiattackNotInAllowance(attackNotAllowed)` if no candidate lists the part, or `MultiattackNotInAllowance(allowanceSpent)` if one does but its slots are used. | SRD 5.2.1, "Multiattack", p. 257 | per row: happy / unhappy (boundary) |
| FR-014 | A group with several parts and a count n ("makes three attacks, using Shortsword or Light Crossbow in any combination"; "makes two Javelin or Morningstar attacks") MUST allow n attacks in total with any mix of its parts, in any order, no part being limited below n. | SRD 5.2.1, "Assassin", p. 260; "Bandit", p. 262; "Bugbear Stalker", p. 271 | happy |
| FR-015 | Variants are alternatives ("one Claws attack and one Tail attack, or it makes two Hurl Flame attacks"). A step MUST be allowed only if some candidate variant allows the whole recorded multiset plus the step; the candidates narrow with each step. Mixing steps that no single variant allows (Claws, then Hurl Flame) MUST be refused with `attackNotAllowed`; once only one variant is left the others never come back. | SRD 5.2.1, "Barbed Devil", p. 262 | unhappy (rule interaction) |
| FR-016 | An `AdditionalUse` ("and uses either Consume Memories or Dominate Mind"; "and one Lightning Blade attack" is a group, not this) MUST be recordable once per entry through `recordAdditionalUse(part, limitedUse)` when `part` is one of its options in a candidate variant: it consumes no attack slot, counts as the entry's one choice (a second option of the same entry MUST be refused `additionalUseAlreadyMade`), and gates the part through FR-007. A part not listed MUST be refused `additionalUseNotListed`. | SRD 5.2.1, "Aboleth", p. 258; "Behir", p. 263; "Chuul", p. 273 | happy / unhappy (boundary) |
| FR-017 | Attacks, additional uses and replacements MAY be recorded in any order; the progress MUST NOT require the order in which the entry's prose lists them (interpretation, Q-1). | SRD 5.2.1, "Multiattack" ("can make"), p. 257 | unhappy (rule interaction) |
| FR-018 | `recordReplacement(option, replaced, limitedUse)` MUST succeed when a candidate variant has a `ReplacementRule` listing `option` whose scope allows `replaced` (`anyAttack`: any group part; `oneAttackPart(p)`: only `p`), whose limit is not reached and whose count n has at least n unused slots in `replaced`'s group. It consumes n slots of that group, gates the option's part through FR-007 (an option that names a spell also through `spendSpell`), and records one `MultiattackAttackReplaced` event. The refusals are `replacementNotListed`, `replacementLimitReached` (`once`: a second replacement under the same rule; `anyNumber`: never), `notEnoughAttacksToReplace`. A replaced attack is not an attack made: no attack roll. "can replace one attack" reads as at most one replacement per rule; "replace any attack" (`anyNumber`) as no cap (interpretation, Q-2). | SRD 5.2.1, "Multiattack", p. 257; "Gladiator", p. 289; "Chimera", p. 273; "Guardian Naga", p. 296; brass dragons, pp. 268-269 | per row: happy / unhappy (boundary) |
| FR-019 | The plan is a **ceiling**: a progress MUST NOT require every slot, additional use or replacement to be used, and exposes `attacksRemaining(group)` and no `isComplete` fault. "Running a Monster" ("use Multiattack on any of its turns in which it is not using one of its more powerful abilities") is advice; no rule enforces it. | SRD 5.2.1, "Multiattack", p. 257; "Running a Monster", p. 255 | unhappy (empty and minimal) |
| FR-020 | A step refused for the allowance (FR-013, FR-016, FR-018) MUST NOT touch `LimitedUseState`; a step refused by the part gate (FR-007) MUST return `MultiattackStepUsageRefused(refusal)` with the progress and `LimitedUseState` unchanged. Allowance is checked before usage. | package design | unhappy (rule interaction) |
| FR-021 | `MultiattackPlan.ProblemsFor(Monster)` MUST return one `MultiattackPlanProblem` per name or reference that does not resolve: a part name that is not an Actions part (Multiattack itself excluded), a name shared by two Actions parts, a replacement spell absent from the monster's Spellcasting groups. Empty when sound. | SRD 5.2.1, "Multiattack", p. 257 | unhappy (missing content) |

Table MA-1 (one test per row; FR-010 to FR-020). Written "slots" per group; plans from the cited blocks.

| Label | Plan and history | Step | Expected | Source | Path |
|---|---|---|---|---|---|
| two Tentacle slots | Aboleth: Tentacle x2 + one of Consume Memories or Dominate Mind; none | Tentacle, Tentacle | both recorded (attack 1, 2) | "Aboleth", p. 258 | happy |
| third Tentacle | same; two Tentacle | Tentacle | `allowanceSpent` | p. 258 | unhappy (boundary) |
| additional use is free | same; two Tentacle | Dominate Mind | recorded; no slot used | p. 258 | happy |
| either, not both | same; Dominate Mind used | Consume Memories | `additionalUseAlreadyMade` | p. 258 | unhappy (rule interaction) |
| any combination | Assassin: three of Shortsword or Light Crossbow | S, C, S then C | first three recorded; fourth `allowanceSpent` | "Assassin", p. 260 | unhappy (boundary) |
| order free | Balor: one Flame Whip, one Lightning Blade | Lightning Blade first, Flame Whip | both recorded | "Balor", p. 261 | happy |
| one each | Balor; Flame Whip used | Flame Whip | `allowanceSpent` | p. 261 | unhappy (boundary) |
| alternative A | Barbed Devil: (Claws + Tail) or (Hurl Flame x2) | Claws, Tail | recorded, variant B gone | "Barbed Devil", p. 262 | happy |
| alternatives mixed | same; Claws used | Hurl Flame | `attackNotAllowed` | p. 262 | unhappy (rule interaction) |
| alternative B | same; none | Hurl Flame x2, then a third | two recorded; third `allowanceSpent` | p. 262 | unhappy (boundary) |
| conditional, not used | Clay Golem: Slam x2, or x3 if Hasten used | start without Hasten; third Slam | `allowanceSpent` | "Clay Golem", p. 274 | unhappy (rule interaction) |
| conditional, used | same, `usedThisTurn` holds Hasten | three Slam | three recorded | p. 274 | happy |
| replacement | Gladiator: Spear x3, replace one with Shield Bash | Shield Bash replacing Spear | recorded; two Spear slots left | "Gladiator", p. 289 | happy |
| replacement once | same; one replacement made | Shield Bash again | `replacementLimitReached` | p. 289 | unhappy (boundary) |
| replacement scope | Chimera: replace the Claw attack with Fire Breath | Fire Breath replacing Bite | `replacementNotListed` | "Chimera", p. 273 | unhappy (rule interaction) |
| any attack, no cap | Guardian Naga: Bite x2, replace any with Poisonous Spittle | replace Bite, replace Bite | both recorded; no Bite slot left | "Guardian Naga", p. 296 | happy |
| replace two | Brass dragon: Rend x3, replace two with Sleep Breath | replace two; Rend; Rend | recorded; one Rend; second `allowanceSpent` | brass dragons, p. 268 | unhappy (boundary) |
| not enough | same; two Rend already made | replace two | `notEnoughAttacksToReplace` | p. 268 | unhappy (boundary) |
| option with spell | Adult Brass Dragon: replace one with (A) Sleep Breath or (B) Spellcasting for Scorching Ray (level 3) | option B | recorded; the spell entry spent if limited | p. 269 | happy |
| option not listed | same | Fire Breath | `replacementNotListed` | p. 269 | unhappy (invalid input) |
| usage refused | replacement part Recharge, spent | replace | `MultiattackStepUsageRefused(StatBlockPartUsageExhausted)`; progress and limited-use unchanged | "Limited Usage", p. 257 | unhappy (state error) |
| replacement first | Gladiator; none | Shield Bash first, then Spear x2 | all recorded | p. 289 | unhappy (rule interaction) |
| no plan | monster without a Multiattack part | start | `MultiattackNotAvailable` | p. 257 | unhappy (missing content) |
| overlapping groups | two groups sharing a part | build plan | `srderr.ErrInvalidInput` error | FR-011 | unhappy (invalid input) |

### 4.3 Attack notation and damage notation

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-022 | `StatBlockAttackPlanner.Plan` MUST build the attack roll's `RollModifiers` as `RollModifiers(ability: Modifier(bonus))` with no Proficiency Bonus and no other modifier: the printed bonus is the whole attack modifier, so adding the monster's Proficiency Bonus or an ability modifier again would double count. A negative or zero bonus is valid. | SRD 5.2.1, "Attack Notation" ("the attack roll's bonus"), "Proficiency Bonus" ("reflected in ... other statistics"), p. 256 | happy / unhappy (boundary) |
| FR-023 | The printed qualifier of an attack (the parenthetical after the bonus) MUST be exposed verbatim and never interpreted; any Advantage it grants reaches `ResolveAttack` only as a `RollCircumstance` the host supplies. | SRD 5.2.1, "Ankheg", p. 259; `content-monsters.md` FR-037 | unhappy (rule interaction) |
| FR-024 | A stat-block attack MUST map to the `AttackForm` of `attacks.md` FR-087 by delivery: melee to `MeleeWeaponAttack`, ranged to `RangedWeaponAttack`. A `meleeOrRanged` attack MUST be planned with an explicit `StatBlockAttackDelivery` (`melee` or `ranged`); the choice is made per attack, never defaulted, and a missing choice is an `srderr.ErrInvalidInput` error. Interpretation: the SRD gives the five legal pairs only; it names no natural-weapon source, and no rule of `attacks.md` keyed to `weapon` applies without a `WeaponUse` (Q-4). | SRD 5.2.1, "Attack Notation", p. 256; "Weapon Attack", p. 191 | happy / unhappy (invalid input) |
| FR-025 | The melee plan MUST carry the printed reach as `Reach`; the ranged plan the printed range as `SingleRange` (no long range) or `NormalAndLongRange` ("range 80/320 ft."). Choosing melee on an attack with no reach, or ranged on one with no range, MUST be an `srderr.ErrInvalidInput` error. | SRD 5.2.1, "Attack Notation", p. 256; "Assassin", p. 260; "Priest", p. 316 | unhappy (invalid input) |
| FR-026 | Each damage instance MUST be rolled by exactly one of the two notations, chosen per instance through a `DamageNotationChoice`: `number` yields `FixedDamage(amount: printed number, type)`, `dice` yields `DiceDamage(printed dice including its flat modifier, type)`. Never both, never the number plus the dice. | SRD 5.2.1, "Damage Notation", p. 257 | happy |
| FR-027 | The choices MUST be fixed before any roll and held in `DamageNotationChoices`, one per instance in printed order; a list of another length, or `dice` for an instance printed with no die expression ("1 Slashing damage"), MUST be an `srderr.ErrInvalidInput` error. No operation re-chooses after a roll. `DamageNotationChoices.All(choice)` is the same choice for every instance. | SRD 5.2.1, "Damage Notation", p. 257; "Awakened Shrub", p. 260 | unhappy (boundary) |
| FR-028 | The planned `DamageComponent`s MUST NOT flag `addsAbilityModifier`: the printed number and dice already include the modifier. The printed number is not recomputed from the dice. | SRD 5.2.1, "Damage Rolls", p. 16; "Damage Notation", p. 257 | unhappy (rule interaction) |
| FR-029 | A Critical Hit MUST add no stat-block rule: `ResolveAttack` doubles dice components only and leaves a fixed component unchanged (`attacks.md` FR-056 to FR-059). Choosing `number` for an instance therefore gives it no extra on a Critical Hit; a host that wants the doubled dice chooses `dice` (interpretation, Q-3). | SRD 5.2.1, "Critical Hits", p. 16; "Damage Notation", p. 257 | unhappy (rule interaction) |
| FR-030 | The attack plan's `DamageSpec` MUST be built from the `Hit` clause's damage instances in printed order, and be absent when that clause has none (`attacks.md` FR-062). Damage printed in a `Miss` or `HitOrMiss` clause is not part of it (FR-031). | SRD 5.2.1, "Attack Notation" ("Hit", p. 256); `attacks.md` FR-062 | happy / unhappy (empty and minimal) |
| FR-031 | `AppliedClauses.ForAttack(part, hit)` MUST list the part's clauses in printed order: for a hit (a Critical Hit included) `Hit` and `HitOrMiss`; for a miss (a natural 1, an automatic miss and a forced miss included) `Miss` and `HitOrMiss`. `Hit` and `Miss` MUST never both apply; `HitOrMiss` always applies once the attack was made. An attack that was not possible (`attacks.md` `AttackNotPossible`) applies no clause. Damage of an applied non-`Hit` clause is rolled with `RollStatBlockDamage`. | SRD 5.2.1, "Attack Notation" ("Hit", "Miss", "Hit or Miss"), p. 256; "Merfolk Skirmisher", p. 308; "Salamander", p. 321 | per row: happy / unhappy (boundary) |

Table AT-1 (one test per row; FR-022 to FR-031).

| Label | Input | Expected | Source | Path |
|---|---|---|---|---|
| whole bonus | Aboleth Tentacle, melee +9, reach 15 | `RollModifiers(9)` only, `Reach(15)`, `MeleeWeaponAttack` | "Aboleth", p. 258 | happy |
| ranged with long range | Assassin Light Crossbow +7, 80/320 | `NormalAndLongRange(80, 320)`, `RangedWeaponAttack` | p. 260 | happy |
| single range | Priest ranged +4, range 60 | `SingleRange(60)` | "Priest", p. 316 | unhappy (boundary) |
| either delivery | Javelin melee or ranged, melee chosen / ranged chosen | reach 10 / range 30/120 | "Bugbear Stalker", p. 271 | unhappy (rule interaction) |
| either delivery, other range | Light Hammer melee or ranged, ranged chosen | range 20/60 | "Bugbear Warrior", p. 272 | unhappy (boundary) |
| no attack notation | Trait part, or a save-only part, to `plan` / `forAttack` | `srderr.ErrInvalidInput` error (name `part`) | FR-053 | unhappy (invalid input) |
| fixed delivery omitted | Aboleth Tentacle (melee only), no delivery | melee plan | p. 258 | happy |
| delivery missing | melee-or-ranged, none chosen | `srderr.ErrInvalidInput` error | FR-024 | unhappy (invalid input) |
| melee without reach | synthetic ranged-only attack planned melee | `srderr.ErrInvalidInput` error | FR-025 | unhappy (invalid input) |
| negative bonus | synthetic bonus -1 | `RollModifiers(-1)` | p. 256 | unhappy (boundary) |
| number chosen | Animated Armor Slam 5 (1d6 + 2) Bludgeoning, `number` | `FixedDamage(5, Bludgeoning)` | p. 259 | happy |
| dice chosen | same, `dice` | `DiceDamage(1d6 + 2, Bludgeoning)` | p. 259 | happy |
| two instances, mixed | Assassin 7 (1d6 + 4) Piercing plus 17 (5d6) Poison, `number` then `dice` | `FixedDamage(7)`, `DiceDamage(5d6)` | p. 260 | unhappy (rule interaction) |
| dice not printed | Awakened Shrub 1 Slashing, `dice` | `srderr.ErrInvalidInput` error | p. 260 | unhappy (boundary) |
| wrong count | two instances, one choice | `srderr.ErrInvalidInput` error | FR-027 | unhappy (invalid input) |
| no ability modifier | any planned component | `addsAbilityModifier` false | FR-028 | unhappy (rule interaction) |
| Hit damage only | part with Hit and Hit or Miss clauses | `DamageSpec` from `Hit` only | p. 256 | happy |
| no Hit damage | attack whose Hit clause has no damage | no `DamageSpec` | FR-030 | unhappy (empty and minimal) |
| clauses on a hit | Merfolk Skirmisher Ocean Spear, hit | `Hit`, `HitOrMiss` in printed order | p. 308 | happy |
| clauses on a miss | same, miss | `HitOrMiss` only (no `Miss` printed) | p. 308 | unhappy (rule interaction) |
| a printed Miss | synthetic part with `Miss` and `HitOrMiss` | miss: `Miss`, `HitOrMiss`; hit: `Hit`, `HitOrMiss` | p. 256 | unhappy (boundary) |
| not possible | `AttackNotPossible` | no clause | FR-031 | unhappy (state error) |

### 4.4 Saving throw effects and the 24-hour immunity

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-032 | `StatBlockSaveEffectPlanner.Plan` MUST use the printed ability as a `D20TestKind.SavingThrow` check and the printed DC unchanged as `DifficultyClass(dc)`. The DC MUST NOT be recomputed from the monster's ability scores or Proficiency Bonus. The target description and a `prefix` stay verbatim; which creatures make the save is the host's choice. | SRD 5.2.1, "Saving Throw Effect Notation", p. 256; "Saving Throws", p. 7 | happy |
| FR-033 | The damage of one use MUST be rolled once and shared by every target that saves, not once per target. The use case of FR-044 rolls it; the per-target resolver takes the rolled `Damage` list. | SRD 5.2.1, "Saving Throws and Damage" ("roll the damage once for all the targets"), p. 16 | happy |
| FR-034 | `StatBlockSaveEffectResolver.Resolve` MUST take one `CheckResult` per target and return, per target, the clauses that apply in printed order: on a failed save `Failure` and `FailureOrSuccess`, plus `FailureByFiveOrMore` when the total is at most the DC minus 5, plus the step label `FirstFailure`, `SecondFailure` or `SubsequentFailures` when the host supplies a `SaveFailureStep` (the host counts failures); on a success `Success` (if any) and `FailureOrSuccess`. Labels the part does not print are skipped. The clauses are returned in printed order and their text is never parsed: a `FailureByFiveOrMore` clause may add to `Failure` (Pseudodragon Sting: also Unconscious while Poisoned) or restate it with another duration (Homunculus: Poisoned until its next turn, or for 1 minute and Unconscious). Interpretation, Q-10: the by-five clause is the worse outcome and governs where the two conflict; the host reads the verbatim text. | SRD 5.2.1, "Saving Throw Effect Notation", p. 256; "Homunculus", p. 299; "Pseudodragon", p. 316; "Death Dog", p. 279; "Steam Mephit", p. 308 | per row: happy / unhappy (boundary) |
| FR-035 | A `Success` clause whose `SaveSuccessRider` is `HalfDamageOnly` MUST give the target the `halve` damage adjustment (applied as the `HalfDamage` adjustment of `hit-points-and-damage.md` FR-014, FR-099, which halves and rounds down) and MUST NOT apply any other part of the failure effect: the `Failure` clause and its conditions are not applied. `FailureOrSuccess` is a separate clause that names both outcomes and still applies. The SRD says "ignoring all other parts of the effect"; reading that to cover a clause labelled as applying on either outcome would make Steam Mephit's "Failure or Success: Being underwater doesn't grant Resistance" unusable on a success, so this is an interpretation (Q-11). This spec computes no halved amount. | SRD 5.2.1, "Saving Throw Effect Notation" ("Half damage only"), p. 256; "Half Damage", p. 16; "Steam Mephit", p. 308 | unhappy (rule interaction) |
| FR-036 | A success whose part prints no half damage on success (no `Success` clause, or a `Success` clause with no rider and no damage) MUST apply nothing from `Failure`: the success has no effect beyond its clause, and the adjustment is `none`. This rule MUST NOT be reached for a `Success` clause that prints "Half damage" (FR-056). | SRD 5.2.1, "Saving Throw Effect Notation" ("what happens on a failed or successful save"), p. 256 | unhappy (empty and minimal) |
| FR-037 | A failed save MUST carry the `none` adjustment and the full rolled damage. | SRD 5.2.1, "Half Damage" ("half the damage that would be dealt on a failed save"), p. 16 | happy |
| FR-038 | A target that is immune to this source and part (FR-040) MUST NOT save: the host calls `SaveImmunities.IsImmune` before rolling, and `resolve` returns `StatBlockSaveTargetImmune` for such a target, ignoring any `CheckResult` and any step supplied for it; nothing applies. | SRD 5.2.1, "Ghost", p. 288 ("immune to this ghost's Horrific Visage for 24 hours") | unhappy (rule interaction) |
| FR-039 | A success on a `Success` clause whose rider is `ImmuneToPartForHours(n)` MUST grant a `SaveImmunityWindow(target, source, part, remaining: n hours)` (n is 24 in the SRD), emitted as `SaveImmunityGranted`. The window names the individual source creature and the part: another monster of the same kind, or another part of the same monster, is not covered (interpretation: "this devil's Unnerving Gaze", Q-5). | SRD 5.2.1, "Chain Devil", p. 273; "Mummy", p. 309; "Cloaker", p. 274; "Ghast", p. 287 | happy |
| FR-040 | `SaveImmunities.IsImmune(target, source, part)` MUST be true while a window for exactly that triple has `remaining` above zero and false otherwise. | SRD 5.2.1, as FR-039 | happy / unhappy (boundary) |
| FR-041 | `SaveImmunities.Elapse(Duration)` MUST reduce every window by the elapsed time, drop each window that reaches zero or below and emit one `SaveImmunityExpired` per dropped window (in granted order); 23 hours 59 minutes after the grant it is still immune, 24 hours it is not (interpretation: a 24-hour duration ends when 24 hours have elapsed). A negative duration MUST be an `srderr.ErrInvalidInput` error; zero changes nothing. The engine reads no clock: the host reports elapsed time, and a rest is not a reset. | SRD 5.2.1, "Ghost", "Mummy", pp. 288, 309 ("for 24 hours") | unhappy (boundary) |
| FR-042 | A grant for a triple that already has a window MUST keep one window with the larger `remaining`; windows never add. | SRD 5.2.1, as FR-039 (immunity, not a duration extension) | unhappy (rule interaction) |
| FR-043 | A failed save, a success with another rider or none, and an effect that ended some other way (the ghost's Possession ending) MUST grant no automatic window. A host MAY build a `SaveImmunityWindow` directly for such a case; the constructor requires a positive `remaining` and distinct `target` and `source`, else an error matching `srderr.ErrInvalidInput`. | SRD 5.2.1, "Ghost", p. 288 | unhappy (invalid input) |
| FR-053 | `StatBlockAttackPlanner.Plan` and `AppliedClauses.ForAttack` given a part with no `StatBlockAttack` MUST return an error matching `srderr.ErrInvalidInput` (name `part`); `StatBlockSaveEffectPlanner.Plan` and the resolver given a part with no `StatBlockSavingThrow` likewise. A part whose attack is `melee` only or `ranged` only takes that delivery when none is supplied; a supplied delivery that differs is FR-025. | SRD 5.2.1, "Attack Notation", "Saving Throw Effect Notation", p. 256 | unhappy (invalid input) |
| FR-054 | For a non-immune target `resolve` MUST return `StatBlockSaveTargetRefused(reason)` and apply nothing when `saves` has no `CheckResult` for it (`missingSave`), or its `CheckResult` is not of kind `D20TestKind.SavingThrow` or its `target` differs from `DifficultyClass(dc)` (`notThisSave`). `CheckResult` carries no `Ability` (`d20-tests.md` FR-041), so that the roll used the printed ability stays the host's duty. One target's refusal does not affect the others. | `d20-tests.md` FR-041; SRD 5.2.1, "Saving Throws", p. 7 | unhappy (state error) |
| FR-055 | A `SaveFailureStep` supplied for a target that succeeded, or has no entry in `saves`, MUST be ignored; extra keys in `steps` never cause a refusal. | package design | unhappy (rule interaction) |
| FR-056 | A `Success` clause whose rider is `HalfDamage` (printed "Half damage" ending the sentence or followed by further text: "Half damage." on most blocks; Bulette Deadly Leap "Half damage, and the target is pushed 5 feet") MUST give the `halve` adjustment and apply the rest of the `Success` clause and `FailureOrSuccess`; unlike `HalfDamageOnly` it keeps the other text of the clause. The `Failure` clause is not applied. | SRD 5.2.1, "Half Damage" ("Many saving throw effects deal half damage ... when the target succeeds"), p. 16; "Balor", p. 261; "Chimera", p. 273; "Bulette", p. 272 | unhappy (rule interaction) |

Table SV-1 (one test per row; FR-032 to FR-043). DC is the printed DC; "total" is the save's total.

| Label | Part and input | Expected | Source | Path |
|---|---|---|---|---|
| fail | Guardian Naga Poisonous Spittle, total below DC | `Failure`; adjustment `none` | "Guardian Naga", p. 296 | happy |
| half damage only | same, success | adjustment `halve`; no `Failure` clause | p. 296 | unhappy (rule interaction) |
| failure or success rider | Steam Mephit Steam Breath, success | adjustment `halve`; `FailureOrSuccess` applies (interpretation, Q-11) | "Steam Mephit", p. 308 | unhappy (rule interaction) |
| plain half damage | Chimera Fire Breath, success | adjustment `halve`; no `Failure` clause | "Chimera", p. 273 | unhappy (rule interaction) |
| half damage with more | Balor, success | `halve`; `FailureOrSuccess` applies | "Balor", p. 261 | unhappy (rule interaction) |
| half damage and a push | Bulette Deadly Leap, success | `halve`; the push text of `Success` kept | "Bulette", p. 272 | unhappy (rule interaction) |
| missing save | non-immune target absent from `saves` | `StatBlockSaveTargetRefused(missingSave)`; others unaffected | FR-054 | unhappy (state error) |
| wrong roll | `CheckResult` of kind `abilityCheck`, or DC unlike printed | `StatBlockSaveTargetRefused(notThisSave)` | FR-054 | unhappy (invalid input) |
| immune with a roll | window held; a `CheckResult` supplied anyway | `StatBlockSaveTargetImmune`; roll ignored | FR-038 | unhappy (rule interaction) |
| step on a success | success with step `first` | step ignored | FR-055 | unhappy (rule interaction) |
| no save notation | attack-only part to the resolver | `srderr.ErrInvalidInput` error (name `part`) | FR-053 | unhappy (invalid input) |
| both by-five readings | Homunculus and Pseudodragon Sting, total = DC - 5 | both clauses in printed order, text unparsed | pp. 299, 316 | unhappy (rule interaction) |
| no success clause | synthetic part with only `Failure`, success | no clause; adjustment `none` | FR-036 | unhappy (empty and minimal) |
| by five or more | Homunculus, total = DC - 5 | `Failure` and `FailureByFiveOrMore` | "Homunculus", p. 299 | unhappy (boundary) |
| by four | same, total = DC - 4 | `Failure` only | p. 299 | unhappy (boundary) |
| first failure | Death Dog, failed, step `first` | `FirstFailure` | "Death Dog", p. 279 | happy |
| subsequent | same, step `subsequent` | `SubsequentFailures` | p. 279 | unhappy (rule interaction) |
| no step given | same, no step | `Failure` clauses only | p. 279 | unhappy (empty and minimal) |
| damage rolled once | three targets, 2d6 chosen | one roll, one `Damage` list reused | p. 16 | unhappy (rule interaction) |
| grant | Mummy Dreadful Glare, success | `SaveImmunityGranted`, 24 hours | "Mummy", p. 309 | happy |
| no grant on failure | same, failure | no window | FR-043 | unhappy (rule interaction) |
| immune target | window held, same source and part | skipped, `StatBlockSaveTargetImmune` | "Ghost", p. 288 | unhappy (rule interaction) |
| other source | window for source A, effect from B | not immune | FR-039 | unhappy (rule interaction) |
| other part | window for part P, effect part Q | not immune | FR-039 | unhappy (rule interaction) |
| 23h59 | window, `elapse(23h59)` | still immune | FR-041 | unhappy (boundary) |
| 24h | window, `elapse(24h)` | not immune, `SaveImmunityExpired` | FR-041 | unhappy (boundary) |
| negative time | `elapse(-1h)` | `srderr.ErrInvalidInput` error | FR-041 | unhappy (invalid input) |
| second grant | window at 5h, regrant 24h | one window, 24h | FR-042 | unhappy (rule interaction) |
| bad window | `remaining` zero, or target equals source | `srderr.ErrInvalidInput` error | FR-043 | unhappy (invalid input) |

### 4.5 The use case, events and placement

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | `RollStatBlockDamage` MUST take `dice.Roller` and an optional `telemetry.Telemetry` (default `telemetry.Telemetry.none`), accept a `RollStatBlockDamageCommand` (the part's `StatBlockDamage` list and `DamageNotationChoices`, FR-027; at least one instance, else an error matching `srderr.ErrInvalidInput`) and return the closed `RollStatBlockDamageResult` whose only variant `StatBlockDamageRolled` holds the `Damage` list in printed order. It holds no rule beyond FR-026 and FR-027 and calls no use case. | `shared-types-and-ownership.md` FR-017, FR-023, FR-044; SRD 5.2.1, "Damage Notation", p. 257 | happy |
| FR-045 | For each instance in order it MUST roll `dice.Roller.dice.Roll` once for a `dice` choice and none for `number`, add the printed flat modifier, floor the result at 0 and return a `Damage(amount, type)`; a `number` instance returns the printed number and type. No Critical Hit doubling exists here (FR-029, FR-033). | SRD 5.2.1, "Damage Rolls" (no negative damage), p. 16; `dice-and-rolls.md` FR-045, FR-050 | happy / unhappy (boundary) |
| FR-046 | A `dice.Roller` error MUST be returned wrapped by `srderr.WrapPort` (a `*srderr.PortError` for port `dice`) with no result; the span and `srd5e.combat.errors` record it once (`combat-foundations.md` FR-032). | `errors-and-results.md` FR-034 | unhappy (failure) |
| FR-047 | The use case MUST record the shared `combat` signals through the `combat` telemetry helper over `telemetry.Telemetry` under operation `roll_stat_block_damage`, class `applied`; `srd5e.combat.result` is `stat_block_damage_rolled` (at least one die rolled) or `stat_block_damage_fixed` (numbers only); the extra attribute `srd5e.combat.damage_notation` (FR-041 of `combat-foundations.md`) is `number`, `dice` or `mixed` (section 7). It MUST behave identically with the no-op and a recording `telemetrytest` double. | `combat-foundations.md` FR-027 to FR-032, FR-041; observability.md §9 | happy |
| FR-048 | The events `StatBlockPartUsed`, `MultiattackAttackRecorded`, `MultiattackAdditionalUseRecorded`, `MultiattackAttackReplaced`, `SaveImmunityGranted` and `SaveImmunityExpired` MUST be the members of one closed `StatBlockActionEvent` implementing `CombatEvent`, each with a constant unique `kind` (`stat_block_part_used`, `multiattack_attack_recorded`, `multiattack_additional_use_recorded`, `multiattack_attack_replaced`, `save_immunity_granted`, `save_immunity_expired`) and `combatants` equal to `[source]` (the monster), or `[target, source]` for the two immunity events; no duplicates. | `combat-foundations.md` FR-015 to FR-021 | happy |
| FR-049 | Every state type (`MultiattackProgress`, `SaveImmunities`, the plan and window types, outcomes) MUST be an immutable value object: equal fields give equal values (`==`, or an `Equal` method where a field is a slice); every operation returns a new value and leaves the receiver and inputs unchanged; returned slices are copies (mutating one changes nothing). Operations return results; only the constructors and the conditions named above return an error matching `srderr.ErrInvalidInput`; nothing panics outside `MustX` helpers. | architecture.md §3, §4, §6; `combat-foundations.md` FR-021 | unhappy (immutability) |
| FR-050 | The domain MUST live in the `stat_block_*.go` files of package `combat`, be synchronous and deterministic (no `math/rand`, `time.Now`, goroutines, `context`, I/O) and contain no `telemetry.Logger`, `Tracer`, `Meter` or signal constant. The use case lives in `roll_stat_block_damage.go`. Imports: the published API of packages `content` and `d20`, the kernel packages (`modifier`; `dice` and `telemetry` for the use case only) and `combat` files of strictly lower rank (including `conditions`); no `character` import. | architecture.md §1, §3, §9; observability.md §3; `combat-foundations.md` FR-026 | happy |
| FR-051 | The types of sections 5 and 6 MUST be exported from package `combat` with a godoc each, no exported name duplicating another in the package and none generic. | architecture.md §4, §7; `combat-foundations.md` FR-022, FR-024 | happy |
| FR-052 | Nothing in the code, tests or data of this spec MAY name a consuming project or use content from outside SRD 5.2.1. | AGENTS.Md ("Hard constraints") | happy |

### Withdrawn ids

None.

Table EC-1 (FR-001 to FR-009; one test per row).

| Label | Input | Expected | Source | Path |
|---|---|---|---|---|
| cost per section | each `MonsterPartSection` | `passive`, `action`, `bonusAction`, `reaction`, `legendaryAction` | pp. 256-257 | happy |
| action entry | synthetic Actions part (Slam-shaped), `NoStandardAction` | `FeatureActionChoice`, part used | p. 256 | happy |
| bonus action | Boar-shaped Bonus Actions part | `BonusActionGrant` with the part `FeatureID` | p. 257 | happy |
| no bonus section | monster with none, request index 0 | `StatBlockPartNotFound` | p. 257 | unhappy (missing content) |
| reaction, trigger | Chain Devil Unnerving Gaze, `occurred` | `FeatureReaction` | "Chain Devil", p. 273 | happy |
| reaction, no trigger | same, `notOccurred` | `StatBlockReactionTriggerNotReported`, unchanged | p. 257 | unhappy (state error) |
| wrong section | Trait requested as action | `StatBlockPartInWrongSection` | p. 256 | unhappy (invalid input) |
| X/Day spend | Aboleth Dominate Mind (2/Day) twice, third | uses 1, 0, then `StatBlockPartUsageExhausted` | p. 258 | unhappy (boundary) |
| Recharge spent | Recharge 5-6 part, spent | `StatBlockPartUsageExhausted` | p. 257 | unhappy (boundary) |
| unlimited part | Aboleth Tentacle | allowed, `LimitedUseState` equal | p. 258 | happy |
| legendary use | Aboleth legendary part, window by another creature | allowed; no action consumed; uses 3 to 2 | p. 257 | happy |
| legendary refused | same, own-turn window | `StatBlockLegendaryUseRefused(ownTurn)`; nothing spent | p. 257 | unhappy (rule interaction) |
| legendary, incapacitated | same, other creature's window, availability `none` | `StatBlockLegendaryUseRefused(incapacitated)`; nothing spent | p. 257 | unhappy (rule interaction) |
| precedence | wrong section and exhausted | `StatBlockPartInWrongSection` | FR-009 | unhappy (rule interaction) |
| feature id | Aboleth, part `Consume Memories` | `aboleth__consume_memories` | FR-005 | happy |
| id rejected | part named `' - '`; name making an id over 64 characters | `invalidName` / `tooLong`; `srderr.ErrInvalidInput` error from `featureIdFor` | FR-005 | unhappy (invalid input) |
| id duplicate | synthetic monster with a Spellcasting trait and a Spellcasting action | both refs `duplicate`; `featureIdProblems` lists both | FR-005 | unhappy (rule interaction) |

## 5. Model

- **Content-owned data (amendments A-2, A-3):** `MultiattackPlan` (`variants`), `MultiattackVariant` (`requiresUsed`: part name or none, `groups`, `additionalUses`, `replacements`), `AttackSlotGroup(count, parts)`, `AdditionalUse(options)`, `ReplacementRule(count, scope, options, limit)`, `ReplacementOption(part, spell?)`, `ReplacementScope`, `ReplacementLimit`; `SaveSuccessRider` (closed `HalfDamageOnly`, `HalfDamage`, `ImmuneToPartForHours(hours)`), an optional member of a `Success` `EffectClause`. Parts are named by their printed name (JSON friendly); this spec resolves names to `LimitedUsePartRef` through the `Monster`. If A-2 or A-3 is rejected, the same types are defined in `combat` with identical shape and the host builds them (Q-6); no requirement changes.
- **Domain (`combat`):** `StatBlockActionCost`, `StatBlockUseRequest` (kind, ref, `GrantedAction`, `ReactionTriggerReport`, legendary window, `ConditionActionAvailability`), `StatBlockUseOutcome` (`StatBlockUseAllowed(transition)`, `StatBlockUseRefused(refusal)`), closed `StatBlockUseRefusal` (`StatBlockPartNotFound`, `StatBlockPartInWrongSection`, `StatBlockReactionTriggerNotReported`, `StatBlockLegendaryUseRefused(LegendaryActionRefusal)`, `StatBlockPartUsageExhausted`), `StatBlockUseTransition(limitedUse, economy, events)`, `MultiattackProgress` (candidate variants, slots per group, additional-use and replacement counters, attacks made), closed `MultiattackStartOutcome` (`MultiattackStarted(progress, transition)`, `MultiattackNotAvailable`), `MultiattackTransition(progress, limitedUse, events)`, closed `MultiattackStepOutcome` (`MultiattackStepRecorded(transition)`, `MultiattackNotInAllowance(reason)`, `MultiattackStepUsageRefused(refusal)`), enum `MultiattackAllowanceRefusal` (`attackNotAllowed`, `allowanceSpent`, `additionalUseNotListed`, `additionalUseAlreadyMade`, `replacementNotListed`, `replacementLimitReached`, `notEnoughAttacksToReplace`), `MultiattackPlanProblem`, `StatBlockAttackPlanner` and `StatBlockAttackPlan` (`AttackForm`, `RollModifiers`, `Reach` or `AttackRange`, qualifier, `DamageSpec?`), `StatBlockAttackDelivery`, `DamageNotationChoice`, `DamageNotationChoices`, `AppliedClauses`, `StatBlockSaveEffectPlanner` and `StatBlockSaveEffectPlan` (kind `savingThrow`, ability, `DifficultyClass`, target description, prefix, damage instances), `StatBlockSaveEffectResolver`, `StatBlockSaveTargetResult` (`StatBlockSaveEffectApplies(clauses, adjustment, grant?)`, `StatBlockSaveTargetImmune`, `StatBlockSaveTargetRefused(StatBlockSaveRefusal: missingSave, notThisSave)`), `SaveFailureStep`, `SaveEffectDamageAdjustment`, `SaveImmunityWindow`, `SaveImmunities`, closed `StatBlockActionEvent` (FR-048), closed `StatBlockFeatureIdOutcome` (`StatBlockFeatureIdFound(FeatureID)`, `StatBlockFeatureIdRejected(reason)`).
- **Application:** `RollStatBlockDamage`, `RollStatBlockDamageCommand`, closed `RollStatBlockDamageResult` (`StatBlockDamageRolled`).
- **5.1 Amendments** (each names the text it changes in the target file; the target's owner applies it):
  - **A-1** `shared-types-and-ownership.md` §4.11: row 27 gains `monster-stat-block-actions`; FR-015's chain gains it between `initiative-and-actions` and `encounter-aggregate`.
  - **A-2** `content-monsters.md` FR-036: "The model MUST NOT interpret which attacks it names." becomes: the Multiattack part keeps its prose verbatim and gains an optional `MultiattackPlan` (types above), decoded from a `plan` key and exposed as `Monster.Multiattack`; the bundled Multiattack blocks carry plans.
  - **A-3** `content-monsters.md` FR-038, FR-039: "'Half damage only' on a success stays verbatim" becomes: the `Success` clause stays verbatim and gains an optional `SaveSuccessRider` (at most one), `HalfDamageOnly` for "Half damage only", `HalfDamage` for other text beginning "Half damage", `ImmuneToPartForHours(n)` for "immune ... for n hours" (also "for the next n hours"); the `MonsterPart` shape gains the field.
  - **A-4** `combat-action-resolution.md`: wires the flows of section 6 ("Driving it").
  - **A-5** `content-schema-and-decoder.md`: the plan shape of FR-011 reports `constraintViolation` problems; the rider is an enum-like field with `invalidValue`.
  - **A-6** `shared-types-and-ownership.md` §5: lists this amendment set.
  - **A-7** `monster-limited-use.md`: `LimitedUsePartRef` addresses any part of any section, not only limited ones (no text change; this spec relies on it).
  - **A-8** `doc/observability/signals.md` and the signal-name constants of package `combat`: the constants and span row of section 7.
  - **A-9** `combat-foundations.md` FR-041: the `operations` counter of `roll_stat_block_damage` carries `srd5e.combat.damage_notation`.
- **Invariants:** plan shape (FR-011); `0 <= slots used <= count`; windows positive and distinct target and source (FR-043); choices one per instance (FR-027). **Failures:** refusals are results; an error matching `srderr.ErrInvalidInput` for untrusted-input defects (`MustX` helpers panic and are for literals only); the only port failure is a `*srderr.PortError` for port `dice` (FR-046). No new error type for ports is defined.

## 6. Interface

```go
// Package combat, files stat_block_*.go (indicative signatures). Rules are package-level
// functions; the closed outcomes are interfaces with an unexported marker method.
// Pure functions take no ctx.
func UseStatBlock(m content.Monster, combatant CombatantID, req StatBlockUseRequest,
	limitedUse LimitedUseState) StatBlockUseOutcome // FR-001..FR-009
func StatBlockFeatureID(m content.Monster, part LimitedUsePartRef) (FeatureID, error) // FR-005; error is *StatBlockFeatureIDRejected (matches srderr.ErrInvalidInput)
func StatBlockFeatureIDProblems(m content.Monster) []StatBlockFeatureIDRejected       // FR-005

type MultiattackProgress struct{ /* unexported */ }
func StartMultiattack(m content.Monster, combatant CombatantID, limitedUse LimitedUseState,
	opts ...MultiattackOption) MultiattackStartOutcome // FR-010..FR-012; options WithGrantedAction (default NoStandardAction), WithUsedThisTurn
func (p MultiattackProgress) RecordAttack(part LimitedUsePartRef, limitedUse LimitedUseState) MultiattackStepOutcome        // FR-013
func (p MultiattackProgress) RecordAdditionalUse(part LimitedUsePartRef, limitedUse LimitedUseState) MultiattackStepOutcome // FR-016
func (p MultiattackProgress) RecordReplacement(option ReplacementOption, replaced LimitedUsePartRef,
	limitedUse LimitedUseState) MultiattackStepOutcome // FR-018
func (p MultiattackProgress) AttacksRemaining(part LimitedUsePartRef) int // FR-019

func PlanStatBlockAttack(part content.MonsterPart, choices DamageNotationChoices,
	opts ...AttackPlanOption) (StatBlockAttackPlan, error) // FR-022..FR-030; option WithDelivery
func PlanStatBlockSaveEffect(part content.MonsterPart) (StatBlockSaveEffectPlan, error) // FR-032, FR-053
func AppliedClausesForAttack(part content.MonsterPart, hit, possible bool) []content.EffectClause // FR-031
func ResolveStatBlockSaveEffect(part content.MonsterPart, ref LimitedUsePartRef, source CombatantID,
	saves map[CombatantID]d20.CheckResult, immunities SaveImmunities,
	steps map[CombatantID]SaveFailureStep) ([]StatBlockSaveTargetResult, error) // FR-034..FR-039, FR-053..FR-056

type SaveImmunities struct{ /* unexported */ }
func (s SaveImmunities) IsImmune(target, source CombatantID, part LimitedUsePartRef) bool // FR-040
func (s SaveImmunities) Elapse(d time.Duration) SaveImmunityTransition                    // FR-041

// File roll_stat_block_damage.go.
type RollStatBlockDamageCommand struct {
	Damage  []StatBlockDamage
	Choices DamageNotationChoices
} // validated by the entry point (FR-044)
type RollStatBlockDamage struct{ /* unexported */ }
func NewRollStatBlockDamage(r dice.Roller, opts ...Option) RollStatBlockDamage // Option: WithTelemetry (zero value = Noop)
func (u RollStatBlockDamage) Do(ctx context.Context, c RollStatBlockDamageCommand) (RollStatBlockDamageResult, error) // FR-044..FR-047
```

Error results are non-nil only for untrusted-input defects (matching `srderr.ErrInvalidInput`) or, from `Do`, a port failure (`*srderr.PortError`). Every refusal is a value of a closed outcome type. Set-like arguments are unique slices, and maps are keyed by comparable value types (`LimitedUsePartRef`, `CombatantID`).

- **Ports:** `dice.Roller` only (synchronous, no `ctx`, `dice-and-rolls.md` FR-045; the use case takes `ctx` first because it starts a span, decision 2b); no new port, double or contract suite. `dice/dicetest` and `telemetry/telemetrytest` gain nothing.
- **Driving it:** the host reads `Monster`, plans an attack with `StatBlockAttackPlanner`, passes the plan's pieces to `ResolveAttack` (`attacks.md`), asks `AppliedClauses` which clauses the outcome applies, then applies any listed condition from the clause text. For a save effect it rolls with `RollStatBlockDamage`, rolls each target's save with the `d20` resolver against `DifficultyClass(dc)`, (having called `SaveImmunities.IsImmune` first, FR-038) calls the resolver, and applies the returned `HalfDamage` adjustment through `ApplyDamage`. `combat-action-resolution.md` wires this (amendment A-4).
- **Exports:** every type of sections 5 and 6 from package `combat`; the plan and rider types are exported from package `content`.

## 7. Observability

The domain is **silent by design** (observability.md §3): rules over values, results and events carry the *what* and *why*. Only `RollStatBlockDamage` emits, through the `combat` telemetry helper over `telemetry.Telemetry`; the four shared signals are owned by `combat-foundations.md` FR-037.

1. **What can go wrong?** A dice adapter that throws (`srd5e.combat.errors`, `dice_roller_error`, span error); a host that always chooses `number` (monsters never roll) or always `dice`; a stuck roller. A refused stat-block use or Multiattack step is a normal rule outcome returned to the host, not an error.
2. **How would an operator know?** Span error status and `errors` for port failures; the `damage_notation` share for the host's choice pattern.
3. **What is needed to diagnose?** Operation, result name and notation, all closed sets; part names, ids and amounts stay in the result (`combat-foundations.md` FR-033).
4. **What to count?** Every use through the shared `operations` counter, with the notation attribute.
5. **Existing instrumentation?** Unchanged; the shared `operations` row gains one extra attribute (amendment A-9).
6. **Monitor or alert?** Any `errors` increment; a notation share pinned at 100% `number` is a host choice, not a fault.
7. **Noisy?** No logs (`combat-foundations.md` FR-034); one value per attribute.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.combat.roll_stat_block_damage` | span | `srd5e.combat.operation`, `srd5e.outcome`, `srd5e.combat.result` (`stat_block_damage_rolled`, `stat_block_damage_fixed`), `srd5e.combat.result_class` (`applied`), `srd5e.combat.damage_notation` (`number`, `dice`, `mixed`) | The path of one stat-block damage roll | Sample traces; alert on error-status spans (a dice adapter failure) |

Constants added to the signal-name constants of package `combat`: `rollStatBlockDamageOperationValue`, `rollStatBlockDamageSpan`, `damageNotationAttribute` (`srd5e.combat.damage_notation`), `damageNotationNumberValue`, `damageNotationDiceValue`, `damageNotationMixedValue`, `statBlockDamageRolledValue`, `statBlockDamageFixedValue`, each `...Value` with a doc comment naming its attribute. The span row feeds `doc/observability/signals.md`.

## 8. Test plan

Tests are in package `combat_test` (`combat/stat_block_*_test.go`, `combat/roll_stat_block_damage_test.go`). Units use synthetic and bundled-shape parts, no ports. Each case-table row is one table-driven test citing its row.

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-009 | unit | `stat_block_use_test.go`: EC-1 rows; feature id derivation, rejection and duplicates; refusal precedence; limited-use state unchanged on refusal; unlimited part leaves state equal |
| FR-010 to FR-021 | unit | `multiattack_progress_test.go`: MA-1 rows; plan constructor invariants; `problemsFor`; allowance refusal leaves limited-use untouched, usage refusal leaves progress equal; a property test over random orders of a legal multiset that every order is accepted |
| FR-022 to FR-031 | unit | `stat_block_attack_planner_test.go`, `applied_clauses_test.go`: AT-1 rows; planned `AttackForm`, `RollModifiers`, `Reach`/`AttackRange`; a planned command accepted by the real `ResolveAttack` with a fixed roller (application, one test) |
| FR-032 to FR-043, FR-053 to FR-056 | unit | `stat_block_save_effect_test.go`, `save_immunities_test.go`: SV-1 rows (plain and only half damage, refusals, ignored steps); `halve` applied through the real `hit-points-and-damage.md` pipeline gives round-down halves (14 to 7, 15 to 7); window boundaries; event order |
| FR-044 to FR-047 | application | `roll_stat_block_damage_test.go` with `dicetest.FixedRoller` and the recording `telemetrytest` double: one roll per `dice` instance in order, none for `number`; a negative flat modifier floors at 0; a roller error (also on the second roll) is returned wrapped as a `*srderr.PortError` with one `errors` increment and no log; equal results with zero-value (Noop) and recording `telemetrytest` double; span, result and notation values; no id in any attribute |
| FR-048, FR-049 | unit, architecture | `events_test.go` kinds, `combatants`, exhaustive type switch; the event-family check helper of `combat-foundations.md` section 8; equality, comparison, copied slices; inputs unchanged |
| FR-050 to FR-052 | architecture | File purity, import allow-list (the pure files import no `character`; `conditions` allowed; `internal/archtest`), no signal code in the pure files, exported-name scan (exports, godoc, duplicate and generic names, the reserved-name list), consumer-name scan |
| errors-and-results.md FR-031, FR-034, FR-051 | application | The roller failure of FR-046 is returned wrapped as a `*srderr.PortError` with no result and one `errors` increment; `error.type` is `srderr.ErrorType(err)`; a panicking roller is recovered by `srderr.Guard`. |

- **Doubles and suites:** none new.
- **Mutation testing:** the `stat_block_*.go` files of package `combat`, with gomu run only inside Docker with `--cpus` and `--workers` capped at 1/4 of `nproc` (`testing.md` §8; never run locally), watching the `-5` margin (FR-034), plain versus only half damage (FR-035, FR-056), `remaining > 0` and `>=` at 24 hours (FR-040, FR-041), the narrowing of candidate variants (FR-015), `once` versus `anyNumber` (FR-018), the refusal order (FR-009, FR-020) and the floor at 0 (FR-045). `RollStatBlockDamage` SHOULD be mutation tested.
- **Not covered:** nothing. The data-side requirements of A-2 and A-3 are tested by `content-monsters.md`.

## 9. Open questions and decisions

Settled, one line each (research per the workflow policy; interpretations are marked):

- Q-1: Multiattack imposes no order and no obligation (FR-017, FR-019). SRD p. 257 wording is "can make"; a descriptive reading fits the package-design rule of imposing no script (interpretation).
- Q-2: "replace one attack" is once per rule, "replace any attack" is uncapped, "replace two attacks" consumes two slots (FR-018; interpretation of the printed wording).
- Q-3: a Critical Hit with the printed number adds nothing; dice double (FR-029). The SRD doubles "damage dice"; a number has none. Interpretation; chosen over doubling the dice average because it invents no figure, and `attacks.md` FR-059 already states it. No official source found.
- Q-4: stat-block attacks use `MeleeWeaponAttack` or `RangedWeaponAttack` (FR-024); the SRD gives no natural-weapon source and `attacks.md` has only five legal forms. Interpretation.
- Q-5: the 24-hour immunity is per source creature and per part (FR-039). The printed text says "this devil's Unnerving Gaze". Interpretation.
- Q-6: the Multiattack plan and the success rider are content data (amendments A-2, A-3), because `content` cannot import `combat`; fallback in section 5 if rejected.
- Q-7: a stat-block `FeatureID` is `<monster>__<part>` (FR-005), so initiative's grant types need no new id type; package design.
- Q-8: "if available" in Multiattack prose adds no behaviour: uses are never mandatory (FR-019), and an unusable part is refused by the gate (FR-007).
- Q-9: no Legendary Action cost beyond one use (`monster-limited-use.md` FR-015); this spec adds no alternative cost.
- Q-10: a by-five failure clause is returned beside `Failure`, unparsed; where durations conflict (Homunculus) the by-five clause governs as the worse outcome, where it adds (Pseudodragon) both apply (FR-034). Interpretation; the notation says only "Failure by 5 or More".
- Q-11: `Failure or Success` clauses survive "Half damage only" (FR-035). The clause is labelled for both outcomes and Steam Mephit's text needs it on a success; the literal "ignoring all other parts" would void it. Interpretation.
- Q-12: plain "Success: Half damage" (about 67 blocks, against 7 "Half damage only") is a rider of its own (FR-056); "Half Damage", p. 16, defines the effect, so no web research was needed.

Open: none. Escalated: none.

## 10. Changelog impact

`Added` (combat): stat-block actions: `StatBlockUse`, `MultiattackProgress`, `StatBlockAttackPlanner`, `AppliedClauses`, `StatBlockSaveEffectResolver`, `SaveImmunities`, `RollStatBlockDamage` with the `StatBlockActionEvent` family, and the signal `srd5e.combat.roll_stat_block_damage` with the attribute `srd5e.combat.damage_notation`. `Added` (content): the Multiattack plan and success rider data (amendments A-2, A-3). Not **BREAKING**: nothing is released.
