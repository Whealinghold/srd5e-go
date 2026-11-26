# Mutation results: combat, Phase 4

This file records a risk-ranked mutation sweep of the `combat` package and what it found. It follows `doc/rules/testing.md` §8.

## Method

- Tool: gomu v0.2.1 through `just mutate ./combat/<file>.go`, in Docker, with CPU and workers capped at one quarter of the processors and `MUTATE_TIMEOUT=300`.
- Scope: 25 of the 45 most logic-dense `combat` files, ranked by size and branching. The sweep stopped by choice before `item_use_thrown.go` (file 26). The remaining 20 `combat` files and the `character` files were not mutated.
- Each surviving mutant was read in the source and put in one of three classes: a real gap (a test can fail on it), an equivalent mutant (it cannot change behavior), or a gomu artifact.
- Every real gap got a test in a `combat/mutgap*_test.go` file. The agents did not re-run gomu on the new tests. They tried a sample of 106 mutants by hand on a private copy of the module. Each one made a new test fail, except one proven equivalent mutant.

## Scores before the new tests

| File | Mutants | Killed | Score |
|---|---|---|---|
| `spell_effects_resolver.go` | 579 | 465 | 80% |
| `summoned_summon.go` | 558 | 478 | 86% |
| `summoned_change.go` | 533 | 443 | 83% |
| `stat_block_multiattack.go` | 372 | 309 | 83% |
| `turn_tracker_ops.go` | 290 | 243 | 84% |
| `containment_terms.go` | 430 | 399 | 93% |
| `summoned_stat_block.go` | 378 | 290 | 77% |
| `limited_use_state.go` | 299 | 238 | 80% |
| `gear_hold.go` | 272 | 233 | 86% |
| `stat_block_use.go` | 319 | 273 | 86% |
| `monster_spellcasting_rules.go` | 286 | 211 | 74% |
| `active_effect_template.go` | 265 | 210 | 79% |
| `turn_usecases.go` | 149 | 126 | 85% |
| `condition_effects.go` | 233 | 195 | 84% |
| `movement_segment.go` | 312 | 273 | 88% |
| `turn_actions.go` | 194 | 182 | 94% |
| `enter_container.go` | 216 | 186 | 86% |
| `gear_deployed.go` | 174 | 151 | 87% |
| `active_effect_advance.go` | 230 | 175 | 76% |
| `special_senses.go` | 184 | 159 | 86% |
| `containment.go` | 235 | 211 | 90% |
| `summoned_roster.go` | 240 | 181 | 75% |
| `encounter_hit_points.go` | 129 | 109 | 84% |
| `gear_kind.go` | 176 | 169 | 96% |
| `encounter_usecases.go` | 142 | 142 | 100% |

After the new tests I re-ran four files. `summoned_stat_block.go` went from 290 of 378 killed (77%) to 362 of 378 (96%). `active_effect_advance.go` went from 175 of 230 (76%) to 191 of 230 (83%). `summoned_roster.go` went from 181 of 240 (75%) to 225 of 240 (94%). `active_effect_template.go` went from 210 of 265 (79%) to 226 of 265 (85%). The other files were not re-run. `monster_spellcasting_rules.go` (74%) got no new test, because all of its survivors are equivalent mutants, so it stays under 80%.

Of the 1144 survivors (7195 mutants, 6051 killed), most were equivalent mutants. The triage counted the real gaps per file, not in total, and one new test often kills several.

## Why the equivalent mutants are equivalent

Each rule below made a mutant impossible to tell apart from the original. If you change the enum order or a validation, one of these can become a real gap again. When you edit such code, recheck them.

- **No-op mutants.** gomu replaces `return ""`, `return 0` or `return false` with the same value.
- **Non-negative values.** A length, a count, an index, a cost or a reason is never negative, so `> 0` equals `!= 0` and `== 0` equals `<= 0`. A string never sorts below `""`.
- **Extreme enum values.** An enum compared with `==` against its lowest or highest value gives the same result as `<=` or `>=`. No value lies beyond it, and the input is already valid.
- **Unreachable error paths.** A constructor or an earlier case already validated the input, so the later error branch cannot run.
- **Slack bounds.** Some loop bounds are looser than the data (`stat_block_multiattack.go:567-577`: a simulation over every count up to 300 gives identical results).
- **Same-value arithmetic.** `iota + 1` changed to `iota * 1` or `iota / 1` leaves the values distinct, and `(hi - lo) % 2` has the same parity as `(hi + lo) % 2`.

`enter_container.go`, `gear_kind.go` and `monster_spellcasting_rules.go` had only equivalent survivors. The per-line lists were not kept in the repository. Rebuild them with `MUTATE_OUTPUT=json just mutate ./combat/<file>.go`.

## Second sweep: 8 more `combat` files

This sweep ran gomu v0.2.1 as in the method above, on eight more files chosen by branch count. The user then stopped mutation testing as part of the normal process. Survivors found in these eight files are triaged and the real gaps have tests. No more files were mutated.

| File | Mutants | Killed | Raw score | Real gaps found |
|---|---|---|---|---|
| `resolve_spell_outcome_table_usecase.go` | 281 | 206 | 73% | 6 (roller failure at a later roll) |
| `monster_traits_recognise.go` | 287 | 184 | 64% | 3 |
| `summoned_request.go` | 172 | 140 | 81% | 8 (one bad field at a time) |
| `perform_unarmed_strike_usecase.go` | 156 | 103 | 66% | 2 |
| `take_combatant_action_usecase.go` | 140 | 98 | 70% | 3 |
| `perform_attack_usecase.go` | 142 | 103 | 73% | 0 new (earlier tests cover them) |
| `magical_effects_usecase.go` | 28 | 22 | 79% | 0 (all 4 equivalent) |
| `zone_assess.go` | 240 | 178 | 74% | 2 |

The raw score counts mutants that do not compile (`NOT_VIABLE`) and equivalent mutants as not killed, so it is lower than the real gap rate. For `resolve_spell_outcome_table_usecase.go`, 25 of the 75 non-killed mutants did not compile, and the score of the 256 runnable mutants is 80.5% before any new test. I did not re-run gomu after the new tests. Each fix was checked by re-applying its mutation by hand in a private copy.

New tests are in `combat/mutgap_spell_outcome_table_test.go` (fails the roller at every roll of eight table scenarios), `mutgap_traits_unarmed_test.go`, `mutgap_action_attack_zone_test.go` and `mutgap_summoned_magical_test.go`. The triage found no source bug.

Equivalent survivors in this sweep fall in these classes, in addition to the rules above:

- **Error branches after a validated input.** `NewDie`, `NewExpression`, `NewDamage`, `NewArrivesOffTarget` and the face lookups (`TeleportKindFor`, `ConfusionRowFor`, `BlinkVanishesOn`, `ReincarnationSpeciesFor`, `PrismaticRayFor`, `PrismaticStrikeFor`) cannot fail for a face that the die can roll. About half of all survivors are of this kind.
- **Redundant guards.** `ResolveSpellOutcomeTable.Resolve` checks `u.roller == nil`, and `tableRoller` checks it again.
- **Highest or lowest enum value.** `Melee` and `Ranged`, `LimitExceeded`, `OrderProblemLimitExceeded` and the `AutomaticCritical` values sit at the ends of their enums.
- **Never-empty collections.** A hit's `Damage()` always has one entry, and a never-negative length makes `> 0` equal to `!= 0`.
- **Unreachable default.** The `default:` of a switch over a closed variant set in `zone_assess.go`. I did not confirm that the listed event types are the only implementers.
- **Loop mutants.** One mutant makes a loop infinite, and the test timeout catches it. It is not a kill by an assertion.

Not mutated, and no longer planned: the other 12 files of the ranked list, the rest of `combat`, and the whole `character` package.

## gomu quirks seen

- If you run an existing test alone, it kills some survivors (`movement_segment.go:197`).
- Mutant ids repeat inside a report (`limited_use_state.go`: 63 duplicate ids of 232), so a survivor can name the wrong mutant.
- A mutant that crashes the test binary can show as SURVIVED. No such case was found in this sweep.
- `TIMED_OUT` is not a kill. The default 30 second timeout (`MUTATE_TIMEOUT`) timed out every `character` mutant, so use 900 there.

## Not covered

- The 12 ranked `combat` files not in either sweep, the rest of `combat`, and the whole `character` package.
- Behavior that needs bundled equipment, tool, feat or monster data (Phase 5).
