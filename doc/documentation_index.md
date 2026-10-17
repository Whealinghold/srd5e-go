# Documentation index

Every file under `doc/`, with what it holds and when to read it. Paths are
relative to `doc/`. Read this file first when you need to find something; open
only the files whose "Read when" matches your task.

| File | Summary | Read when |
|---|---|---|
| `documentation_index.md` | This index: one row per file in `doc/`, plus the rules for keeping it current. | You need to find which document answers a question, or you have added, removed, renamed or repurposed a file in `doc/`. |
| `roadmap.md` | The phased plan (Phase 0 foundation through Phase 5 breadth and release), what each phase delivers, which phases are implemented in Go (Phase 1 implemented, Phase 0 partly), and the planning notes (specs first, revisit the context map, content volume risk). | You are choosing what to build next, deciding which phase a piece of work belongs to, or checking the intended order of contexts. |
| `plans/plan-phase-1.md` | The Phase 1 delivery plan for the `d20` package: the three-step sequence (domain values, domain results and resolution, the `Resolver` use case with telemetry), with per-step files and tests, the process contract (checks, coverage, mutation testing, changelog), marked done. | You want the model a later phase plan should follow, or to see how `d20` was sequenced. |
| `rules/go.md` | Normative Go conventions: naming, documentation, error handling idioms, context rules, zero values and constructors, generics policy, interfaces, file and package organisation, the compatibility promise and deprecation, forbidden constructs. | You write or review any Go code, add an exported identifier, or change the public API. |
| `rules/architecture.md` | Normative architecture rules: package structure, dependency rule, bounded contexts and the context map, domain model, use cases and ports, adapters, errors and results, public packages and versioning, mechanical enforcement, anti-patterns. | You write or review any code, add a context, port, adapter or use case, change an export, or touch the dependency rules. |
| `rules/observability.md` | Normative rules for logging, metrics and traces: the observability ports, what to emit and where, naming, attributes, who records which failure, the mandatory observability evaluation for every change, enforcement. | You add or change a use case, port, adapter or failure mode, add a signal, or write the `Observability` entry of a PR or summary. |
| `rules/testing.md` | Normative testing rules: the test pyramid, happy and unhappy path checklist, requirement citations on every test, writing good tests, coverage, mutation testing (gomu, in Docker only), architecture tests, the mandatory test evaluation. | You write or review any test, finish any change, or write the `Testing` entry of a PR or summary. |
| `research/sources.md` | The one source document for game rules (SRD 5.2.1, CC-BY-4.0): download location, licence and attribution pointers, a table mapping SRD sections and page ranges to contexts, and what is out of scope. | You need to find an SRD section or page, decide whether some content is in scope, or cite a source. |
| `observability/signals.md` | The signal catalogue: every metric, span and log event the package emits, with type, unit, attributes, emitter, purpose and suggested monitoring — today the `d20` resolve signals. Rows are added in the same change as their code constants; signals whose code has not landed live in their owning spec until then. | You add, change or remove a signal, or a consumer wants to know what to dashboard or alert on. |
| `specs/_template.md` | The ten-section template every spec follows: purpose and scope, source, vocabulary, requirements with stable FR-ids, model, interface, observability, test plan, open questions, changelog impact. | You write a new spec or check that an existing one is complete. |
| `specs/shared-types-and-ownership.md` | Binding decisions (FR-001 to FR-072) on which spec owns each shared type, dependency order inside and between contexts, result and event naming, content delivery, where ports, no-ops and test doubles live, telemetry conventions, and the `toolbox` context. It wins over any conflicting spec. | You write or edit any spec, or code that defines or imports a type used across contexts, or you are unsure who owns a type. |
| `specs/dice-and-rolls.md` | Dice expressions, roll results, the `dice.Roller` port with its failure contract, the seeded roller and its test double. | You implement dice or randomness, or any code that calls `dice.Roller`. |
| `specs/modifiers-and-advantage.md` | The `modifier.Modifier` value type and Advantage and Disadvantage (`modifier.RollMode`), including how they cancel. | You implement roll modifiers or roll modes in the kernel or in `d20`. |
| `specs/ability-scores.md` | The six abilities, `ability.Score`, `ability.Modifier`, the score range, modifier derivation and the invalid-score reasons. | You implement or use ability scores and modifiers. |
| `specs/errors-and-results.md` | How the module reports failure: `srderr` (`PortError`, sentinels, `Guard`), result-type conventions, the result versus error versus panic split. | You add an error, a failure result, a port failure contract, or code that handles port errors. |
| `specs/observability-ports.md` | The `Logger`, `Tracer` and `Meter` ports, the bundled `telemetry.Telemetry` value, the no-op defaults, and the recording doubles in `telemetry/telemetrytest`. | You implement or consume an observability port, or write a test that asserts telemetry. |
| `specs/architecture-guardrails.md` | Every mechanical check in `internal/archtest` (imports, layering, context map, banned calls, signal catalogue consistency, tag and notice checks, spec cross-references), each as a requirement. | You write or change an architecture test, or want to know why a build rule fails. |
| `specs/d20-tests.md` | The D20 Test domain: ability checks, saving throws, attack rolls, DCs, natural 1 and 20, success and failure, Advantage application, rerolls, and the pure resolution functions in `d20`. Domain only; silent. | You implement or consume D20 Test rules, rerolls or DC resolution. |
| `specs/resolve-d20-test-use-case.md` | The `d20.Resolver.Resolve` use case: command, result, dice port use and the full telemetry contract. The reference template for all later use cases. | You write any use case, or add telemetry to a use case. |
| `specs/content-source-port.md` | The `content.Source` port, the `content.LookupResult` variants (`LookupFound`, `LookupNotFound`), the in-memory and bundled sources, the shared contract suite and the test double. | You implement a content source or code that looks up content. |
| `specs/content-schema-and-decoder.md` | The content JSON envelope, the content identity types (`content.ID`, `content.Key`, `content.Kind`, `content.Entry`, `content.MissingContent`), the closed vocabulary types and typed ids, the decoder that reports every problem, the single schema version and its revision table. | You define or change any content shape, vocabulary type or id, or the decoder. |
| `specs/content-licensing-and-notice.md` | What `NOTICE.md` must contain, the covered paths, how the MIT and CC-BY-4.0 parts stay distinct, change indication and trademark constraints. | You bundle or move game content, edit `NOTICE.md`, or prepare a release. |
| `specs/content-thin-slice.md` | The first slice of bundled SRD data (a few species, one class, a few spells) chosen to validate the schema. | You build the first bundled data or need an end-to-end content example. |
| `specs/content-classes.md` | The class and subclass content model, `AbilityRequirement`, and the bulk class and subclass data for all SRD classes. | You define or load class data, or write a class behaviour spec. |
| `specs/content-classes-data.md` | Bundled data for all twelve classes and subclasses, the extraction and verification process, and the oracle tables the class content requirements cite. | You transcribe or verify class and subclass data. |
| `specs/content-species-backgrounds-feats.md` | Bulk data for species, backgrounds and feats, plus the validation rule for creating a custom background. | You transcribe or load species, backgrounds or feats. |
| `specs/content-equipment.md` | Bulk data for weapons, armor, tools, adventuring gear, mounts and the rest of SRD equipment, with extraction and verification rules. | You transcribe or load equipment data. |
| `specs/content-spells.md` | The `Spell` content entry and the bulk bundling of every SRD spell, with scaling facts and class lists. | You transcribe or load spells. |
| `specs/content-monsters.md` | The stat block schema and bulk data for SRD monsters and animals, with extraction and verification. | You transcribe or load monsters, or use stat blocks. |
| `specs/content-magic-items.md` | The magic item catalogue (A to Z): schema entries, extraction and verification. | You transcribe or load magic items. |
| `specs/spell-effect-data.md` | The `content` half of spell effects: the structured effect data a spell can carry beside its text, its facts, decoder shape and bundling. | You define or load spell effect data. |
| `specs/derived-stats.md` | The character foundation: `CharacterLevel`, `ClassLevels`, Proficiency Bonus, hit point maximum and Hit Dice, Armor Class, saves, skills, `IsBloodied`. Pure functions. | You compute any derived character value. |
| `specs/character-aggregate.md` | The `Character` aggregate: root, invariants, typed id, state in and new state and events out, `FeatureStates` and the `HeroicInspiration` slot. | You change or consume character state. |
| `specs/character-creation.md` | The five steps of creating a character and the `CreateCharacter` flow: choices, ability scores, origin, equipment, languages. | You implement or call character creation. |
| `specs/wish-spell.md` | The Wish spell after the cast: duplication and the seven effects as checks and handoffs, the stress (Necrotic cast damage, Strength 3, lockout) as persistent state, recovery days. | You implement or resolve Wish. |
| `specs/level-advancement-and-multiclassing.md` | Experience and level-up, starting above level 1, Ability Score Improvement and Epic Boon, multiclass rules, multiclass spell slots, Trinkets. | You implement leveling or multiclassing. |
| `specs/higher-level-start-equipment-and-trinkets.md` | Building a character above level 1 (Starting at Higher Levels, p. 24) and the Trinkets rules (pp. 26-27), split from level advancement. | You implement starting at higher levels or Trinkets. |
| `specs/feats-behaviour.md` | Taking a feat and applying its benefits, prerequisites, repeatable feats and feat choices. | You implement feat effects. |
| `specs/species-trait-behaviour.md` | Species trait choices, limited uses, rest windows and declared effects for the nine SRD species. | You implement species traits. |
| `specs/weapon-properties-and-mastery.md` | Weapon proficiency, Weapon Mastery choice, the property rules and the Mastery effects. | You implement weapon properties, mastery or two-weapon fighting. |
| `specs/rests-and-hit-dice.md` | Short and Long Rests, spending Hit Dice, what recharges on which rest, `RestKind`. | You implement rests or recharging features. |
| `specs/spellcasting.md` | Spellcasting and Pact Magic state for a character: spellcasting ability, spell save DC and attack bonus, spell slots (including multiclass), cantrips and prepared spells, and the shapes that hold them. | You implement spell slots, prepared spells or spellcasting state. |
| `specs/casting-and-concentration.md` | Casting rules (slot use, casting time, components, Rituals, casting in armor, feature-granted casts) and Concentration (beginning, ending, breaking, the save DC from a plain damage amount). | You implement casting a spell or Concentration. |
| `specs/heroic-inspiration.md` | Heroic Inspiration mechanics: granting, spending and the reroll rule, with its use cases. | You implement or use Heroic Inspiration. |
| `specs/equipment-rules.md` | Rule maths for equipment: crafting, ship repair, drawn load, barding, scroll scribing, services, Armor Class from armor. | You implement equipment rules or carrying capacity. |
| `specs/magic-item-rules.md` | Behaviour and state of magic items (a `character` spec): Attunement, wearing and wielding, activation, charges and "The Next Dawn", curses, durability, crafting requirements, sentient items. | You implement how magic items behave. |
| `specs/magic-item-effects.md` | The numeric effects of SRD magic items (a `character` spec): bonus items, set and capped ability scores, Potion of Healing and Resistance, charged spell items and the last-charge roll, Ring of Spell Storing, and extradimensional container capacity and nesting. | You implement what a particular magic item does in numbers. |
| `specs/spell-scroll-use.md` | Using a Spell Scroll (a `character` spec): who can read it, the ability check for a spell above the reader's castable level, the save DC and attack bonus, what happens to the scroll afterwards, and copying a Wizard spell from a scroll. | You implement reading or casting from a Spell Scroll. |
| `specs/tool-proficiency-and-crafting.md` | Tool Proficiency on an ability check, Utilize DCs of a tool, and the prerequisites of crafting a nonmagical item, brewing a Potion of Healing and scribing a Spell Scroll (a `character` spec, pure functions). | You implement tool checks, crafting or scribing prerequisites. |
| `specs/social-interaction.md` | Monster attitudes and the Influence action as the SRD states them. | You implement social interaction rules. |
| `specs/exploration.md` | An index of the seven Exploration parts, deciding which are modellable and which are Game Master judgment, pointing each to its owning spec. It owns Travel (pace table and effects, mounts, vehicles) and light sources and fuel. | You implement exploration or travel, or need to know which spec owns an Exploration rule. |
| `specs/class-barbarian.md` | Barbarian behaviour: core traits, features by level and the Berserker subclass. | You implement the Barbarian. |
| `specs/class-bard.md` | Bard behaviour: core traits, features, spellcasting and the College of Lore. | You implement the Bard. |
| `specs/class-cleric.md` | Cleric behaviour: core traits, features, spellcasting and the Life Domain. | You implement the Cleric. |
| `specs/class-druid.md` | Druid behaviour: core traits, features, spellcasting and the Circle of the Land. | You implement the Druid. |
| `specs/class-fighter.md` | Fighter behaviour: core traits, features and the Champion subclass. | You implement the Fighter. |
| `specs/class-monk.md` | Monk behaviour: core traits, features and the Warrior of the Open Hand. | You implement the Monk. |
| `specs/class-paladin.md` | Paladin behaviour: core traits, features, spellcasting and the Oath of Devotion. | You implement the Paladin. |
| `specs/class-ranger.md` | Ranger behaviour: core traits, features, spellcasting and the Hunter subclass. | You implement the Ranger. |
| `specs/class-rogue.md` | Rogue behaviour: core traits, features and the Thief subclass. | You implement the Rogue. |
| `specs/class-sorcerer.md` | Sorcerer behaviour: core traits, features, Metamagic and the Draconic Sorcery subclass. | You implement the Sorcerer. |
| `specs/class-warlock.md` | Warlock behaviour: core traits, Pact Magic, Eldritch Invocations and the Fiend Patron. | You implement the Warlock. |
| `specs/class-wizard.md` | Wizard behaviour: core traits, spellcasting, spellbook and the Evoker subclass. | You implement the Wizard. |
| `specs/class-wizard-evoker.md` | Evoker subclass behaviour: Evocation Savant, Potent Cantrip, Sculpt Spells, Empowered Evocation, Overchannel. | You implement the Evoker. |
| `specs/combat-foundations.md` | The bottom of the combat order: `CombatantId`, shared combatant references, `CombatEvent`, naming and signal conventions for `combat`. | You start any `combat` spec or need a combatant type. |
| `specs/hit-points-and-damage.md` | Hit points, applying damage and healing, temporary hit points, resistance, vulnerability, immunity, dropping to 0, death saves. | You implement damage, healing or dying. |
| `specs/conditions.md` | The full set of SRD conditions as data, their effects (`ConditionEffects`), stacking and Exhaustion, and `ConditionApplied` events. | You implement or apply a condition. |
| `specs/attacks.md` | Resolving one attack: cover, range, attack roll, critical hits, unarmed strikes, composing the pure D20 resolver. | You implement attack resolution. |
| `specs/attack-and-help-actions.md` | The detail of the Attack action, Extra Attack across sources and the Help action. | You implement the Attack or Help action. |
| `specs/deployed-and-restraining-gear.md` | Deployed gear (Caltrops, Ball Bearings, Oil on a square, Hunting Trap) and holds (Net, Chain, Manacles, Rope): placement, area-entry saves, binding, escape, burst and unlock routes. | You implement set-out or restraining adventuring gear. |
| `specs/grappling.md` | The grapple option, escape, movement while grappling and ending a grapple. | You implement grappling. |
| `specs/monster-engulf-and-swallow.md` | The state machine for a creature held inside another: the Swallow, Engulf and Whelm actions, containers and capacity, conditions imposed and removed, periodic damage due, regurgitation and exits. | You implement Swallow, Engulf or Whelm monster actions. |
| `specs/initiative-and-actions.md` | Initiative, turn order, actions, bonus actions, reactions and basic movement. | You implement turn flow or action economy. |
| `specs/item-use-effects.md` | Item-use effects of Adventuring Gear: thrown Acid, Alchemist's Fire, Holy Water, Oil and Net, Oil coat, Basic Poison, Antitoxin, Potion of Healing and Healer's Kit, returned as consequences to apply. | You implement using or throwing consumable gear in combat. |
| `specs/monster-combatant-setup.md` | Turning one `Monster` stat block into combat inputs: Hit Points (average or rolled), Initiative, affinities, condition immunities, Proficiency Bonus from Challenge Rating, Speeds and the combatant profile. | You build a monster combatant from a stat block. |
| `specs/monster-limited-use.md` | Monster limited usage: X/Day, Recharge X-Y, Recharge after a Short or Long Rest, Legendary Action timing and Lair actions (SRD pp. 254-257). | You implement monster abilities with usage limits. |
| `specs/monster-stat-block-actions.md` | What a stat-block entry does when used: Multiattack allotment, printed attack bonus, hit and miss clauses, save effects with half damage, Bonus Action and Reaction entries and the 24-hour immunity rider. | You implement monster actions from a stat block. |
| `specs/monster-traits.md` | Twelve recurring monster traits (Legendary Resistance, Magic Resistance, Undead Fortitude, Regeneration, Pack Tactics, Swarm and others) as typed facts, roll circumstances and state rules. | You implement monster traits. |
| `specs/monster-spellcasting.md` | How a monster casts a spell from its Spellcasting trait: lowest possible level, printed save DC and attack bonus, "self only", waived components, casting times of 1+ minutes (SRD p. 257). | You implement a monster casting spells. |
| `specs/movement-lifting-and-flight.md` | Jumping, flying, burrowing, movement checks and the load Speed limit. | You implement movement beyond walking. |
| `specs/grid-positions-and-movement.md` | The optional square-grid rules: positions, spaces by size, range in squares, entering squares, corners and planning a move into segments. | You implement grid play, grid distance or square-by-square movement. |
| `specs/hiding-searching-and-senses.md` | Pure functions for what light and obscurement do to seeing, what Blindsight, Darkvision, Tremorsense and Truesight change, whether an observer can see a target, and the Hide, Search and Study checks. | You implement vision, senses, hiding, searching or studying. |
| `specs/mounted-and-underwater-combat.md` | Mounted Combat rules and the underwater fire resistance rule. | You implement mounted or underwater combat. |
| `specs/environment-hazards-and-objects.md` | The glossary Hazards (Burning, Falling, Dehydration, Malnutrition, Suffocation and similar, with the Exhaustion they cause) and the Objects rules, which the hit-point and condition specs exclude. | You implement environmental damage or Exhaustion sources, or attacks on and damage to objects. |
| `specs/magical-effects.md` | Glossary phenomena independent of any one spell: Magical Effect and its source, Area of Effect origin and membership, Curses, Illusions, Shape-Shifting, Teleportation, Telepathy, Simultaneous Effects, returning to life, and readying a spell. | You implement area-of-effect membership, those glossary effects, resurrection or readied spells. |
| `specs/readying-a-spell.md` | The spell half of the Ready action (SRD pp. 186-187): readying a spell, its Concentration and what happens when the trigger fires. | You implement readied spells. |
| `specs/returning-to-life.md` | Returning a Dead creature to life (SRD p. 180) as a value-level transformation of the hit point state. | You implement resurrection or revival. |
| `specs/revival-and-restoration-spells.md` | Revival spells (Revivify, Raise Dead, Resurrection, True Resurrection), Hit Point maximum increases (Aid, Heroes' Feast), the restoration spells and Death Ward as pure domain operations, composing `returning-to-life.md`. | You implement revival spells, Aid, Heroes' Feast, restoration spells or Death Ward. |
| `specs/telepathy.md` | The Telepathy glossary rules (SRD p. 190): who may start a telepathic contact, what breaks it, who may reply. | You implement telepathy. |
| `specs/spell-effects.md` | The `combat` interpreter for spell effects: `ResolveSpellEffects` resolves saves, attack rolls, damage and conditions from the effect data, composing the pure D20 resolver. | You implement or consume spell effect resolution. |
| `specs/spell-outcome-tables.md` | The printed spell tables (Teleport, Confusion, Prismatic Spray and Wall, Reincarnate, Blink) and `ResolveSpellOutcomeTable`, which rolls them and returns plain values for the host to apply. | You roll or resolve a spell's outcome table. |
| `specs/active-effects.md` | The registry of active effects that span turns: durations, Concentration binding, repeated saves, recurring and growing damage, staged effects, moving a mark and Combining Spell Effects. | You implement spell durations, expiry or recurring effects. |
| `specs/spell-granted-modifiers.md` | What an active spell adds to rolls, Armor Class, Speed and damage (Guidance, Haste, Slow, Shield, Mage Armor, Longstrider, Bestow Curse and similar), as pure functions. | You implement a spell's effect on rolls or stats. |
| `specs/persistent-spell-zones.md` | Persistent zones left by spells (Spirit Guardians, Cloudkill, Moonbeam, Spike Growth and kin): entering, turn-start and turn-end triggers, once-per-turn ledger and moving areas. | You implement spells that leave a damaging or hindering area. |
| `specs/dispel-and-counterspell.md` | Dispel Magic, Counterspell and Antimagic Field: the ongoing-spell registry, dispel checks, countering a cast and magic suppression. | You implement Dispel Magic, Counterspell or Antimagic Field. |
| `specs/shape-changing-spells.md` | Polymorph, True Polymorph, Shapechange and Animal Shapes: allowed forms, kept and replaced statistics, Temporary Hit Points of the form, form ending and permanence. | You implement shape-changing spells. |
| `specs/encounter-aggregate.md` | The `Encounter` aggregate: root, invariants, typed id, state in and new state and events out. | You change or consume encounter state. |
| `specs/combat-action-resolution.md` | The wiring use cases that chain attack, damage, conditions and the encounter, plus the combat scenario test. | You wire combat pieces together or write the combat example. |
| `specs/combatant-roster-changes.md` | Adding, removing and repositioning a combatant of an `Encounter` after combat starts: roster position, turn-tracker transitions, events and clean-up of grapples and Help benefits. | You add or remove combatants during a fight. |
| `specs/summoned-and-companion-creatures.md` | Summoned and companion creatures: ally status, turn-order place, commands, departure, level-dependent stat blocks, Animate Dead, Simulacrum, familiars and steeds. | You implement summoned creatures, familiars or steeds. |
| `specs/toolbox-hazard-effects.md` | The `toolbox` context bootstrap and the shared save-gated effect model used by the other toolbox specs. | You start any `toolbox` work or need the shared save-then-effect model. |
| `specs/toolbox-travel-pace.md` | Travel Pace, Travel Terrain and Extended Travel. | You implement overland travel. |
| `specs/toolbox-environmental-effects.md` | The nine Environmental Effects entries (deep water, cold, heat, altitude and others). | You implement environmental hazards. |
| `specs/toolbox-poison.md` | Poison types, harvesting and the catalogue of sample poisons. | You implement poison. |
| `specs/toolbox-traps.md` | Trap structure, scaling tables and the example traps. | You implement traps. |
| `specs/toolbox-curses-and-contagions.md` | Demonic Possession, the contagion rules and the example contagions. | You implement curses or contagions. |
| `specs/toolbox-fear-and-mental-stress.md` | The optional fear and mental stress rules, sample DCs and prolonged effects. | You implement fear or mental stress. |
| `specs/toolbox-combat-encounter-difficulty.md` | The XP Budget calculation for planning encounter difficulty. | You implement encounter planning. |
| `specs/rule-options.md` | The pattern for immutable optional-rule value objects (in the command, never global) and the rules every option object shares. Each context's own options live in its owning spec. | You add or consume an optional rule or house-rule hook, or define a new option object. |
| `specs/rules-glossary-coverage.md` | A traceability table mapping every Rules Glossary entry (SRD pp. 176-191) to the spec that owns it. | You need to find which spec owns a glossary rule, or check nothing is unowned. |
| `specs/release-0-1-0.md` | What must be true before the first release: public API surface, examples, changelog, the v0.1.0 git tag, publish checklist. | You prepare, review or publish a release. |

## Instructions for agents: keeping this file current

You MUST update this file in the same change that adds, removes, renames or
repurposes any file under `doc/`. A stale index is a defect.

What the file must hold:

1. **One row per file under `doc/`, including this one.** No file missing, no
   row for a file that does not exist. Check with `find doc -type f | sort`
   against the first column. Paths are relative to `doc/` and written in
   backticks.
2. **Three columns, in this order:** `File`, `Summary`, `Read when`.
3. **Summary:** what the file defines or contains, in one or two sentences.
   Describe the content (the types, rules, decisions, data it holds), not its
   status or history. Do not copy requirement text. Do not restate the title.
4. **Read when:** a concrete trigger, phrased as the task or question that
   makes the file worth opening ("You implement X", "You need to find Y").
   Never "for reference" or "to understand the project".
5. **Order:** top-level files, then `plans/`, `rules/`, `research/`, `observability/`,
   then `specs/` with the template and the binding decisions first, then
   shared and phase 0 specs, then by context in dependency order (`d20`,
   `content`, `character`, `combat`, `toolbox`), then package-level specs.
6. **Update an existing row** when a file's purpose or scope changes enough that
   its summary or its trigger is no longer true. A change to wording or a
   status field inside a spec does not need an index update.
7. **Do not copy a rule file's read-receipt tag** (or any `RULE-TAG` value)
   into this file. The architecture tests fail the build if a tag appears in any
   file other than its own.
8. **Keep the rows to one line each** and the section below this table as the
   last section, so the rules for maintaining the index stay at the bottom.
