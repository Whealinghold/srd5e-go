# Travel Pace, Travel Terrain and Extended Travel

| | |
|---|---|
| Status | draft |
| Context | `toolbox` |
| Roadmap phase | 5 |
| Depends on | `exploration.md`, `toolbox-hazard-effects.md`, `d20-tests.md`, `dice-and-rolls.md`, `errors-and-results.md`, `observability-ports.md` |

Context: the Gameplay Toolbox context (`shared-types-and-ownership.md`
FR-045), downstream of `content`, `d20`, `character` and `combat`; nothing
depends on it. Type phase 5 (FR-014, FR-045); the imported `character` types
ship in phase 3; see [../roadmap.md](../roadmap.md). Not a dependency (same
rank): the host composes `toolbox-environmental-effects.md` (`AssessHighAltitude`,
which owns the doubling: its FR-090, FR-091, D-4) with this spec.

**Supersedes** (FR-048): no requirement of any other spec. **Superseded by**
`exploration.md` for the p. 12 rules (withdrawn ids: section 4.7).

## 1. Purpose and scope

The overland travel rules of the Gameplay Toolbox: the Travel Terrain table
(maximum pace, encounter distance, Foraging, Navigation and Search DCs for eleven
terrains), the factors that change a group's pace (Good Roads, Slower
Travelers, flying or ignoring Difficult Terrain), how a pace becomes a
distance (Special Movement formulas), Vehicles, and Extended Travel (a
Constitution saving throw per hour past 8 or 1 Exhaustion level).

**Not owned here (imported).** The Travel Paces, the p. 12 table, the game
effect of each pace, the vehicle speed type and the mounted hour:
`exploration.md` (FR-001 to FR-012, FR-014 to FR-025, FR-046, FR-047). Whether
an hour is a High Altitude hour and the doubling ("counts as 2 hours"):
`toolbox-environmental-effects.md`; this spec never doubles an hour (FR-031).
The save-then-effect mechanism: `toolbox-hazard-effects.md`. This spec adds the
Travel Terrain data, the maximum-pace rules, the Special Movement formulas,
the plan, the Extended Travel schedule and the encounter-distance roll. The
small tables are constants in the toolbox domain, not a content kind
(`shared-types-and-ownership.md` FR-045).

The domain computes. `PlanTravel` plans a journey and `RollEncounterDistance`
rolls through `Roller`. Each Extended Travel save is a `SaveGatedEffect`
the host passes to `ResolveSaveGatedEffect`; the Exhaustion level is handed
back for the host to apply with `conditions.md`. Nothing here changes a
condition (architecture.md section 4). **Input:** hours as two sums,
`actualHours` and `effectiveHours`, summed by the host from the results of
`AssessHighAltitude` (SRD 5.2.1, "High Altitude", p. 195, Q-6). No use case
calls another (`shared-types-and-ownership.md` FR-044).

**Out of scope:**

- **Which check uses a Foraging, Navigation or Search DC.** p. 192 prints the
  DCs as columns with no procedure; carried as data (FR-005).
- **Mounts doubling the distance for 1 hour** and the mount's rest need
  (p. 12): `exploration.md` (`TravelRules.mountedHourMiles`, `MountedPaceBoost`,
  FR-018 to FR-025). A plan does not apply the boost (Q-10).
- **Marching Order**, carrying capacity, summarising a journey (p. 12).
- **Speed of a spell or magic item** granting high-speed travel (Wind Walk,
  Carpet of Flying are only examples on p. 192); the host passes the Speed in.
- **Speed adjustments printed elsewhere, applied by the host before calling:**
  - Exhaustion lowers Speed 5 feet per level (p. 181); it can trigger Slower
    Travelers (FR-015). The host passes `TravelerSpeed.current` (`conditions.md` FR-107).
  - Vehicle adjustments (p. 100: wind, calm, current, draft animals; p. 221:
    "Feather Token" Fan, Swan Boat). The host passes the adjusted `VehicleSpeed`, rounded down to a half mile (Q-19).
  - Horseshoes of a Zephyr (p. 226): 12 hours a day without Exhaustion. The
    schedule (FR-035) ignores it; the host skips the exempted saves.
- **Land-vehicle pace choice** (p. 12 "choose a pace as normal"): the host uses
  `TravelOnFoot` with the draft animals' Speed (Q-8).
- **Creating a Background** (p. 193): `content-species-backgrounds-feats.md`.
- **Applying Exhaustion**, immunity, death at level 6: `conditions.md`,
  `hit-points-and-damage.md` FR-095.
- **Partial hours** (FR-030).
- **Nothing core was found missing** from the pages in section 2.

## 2. Source

PDF page numbers equal printed page numbers.

- SRD 5.2.1, "Gameplay Toolbox", "Travel Pace", "Good Roads", "Slower
  Travelers", "Extended Travel", "Special Movement", "Vehicles", and the
  "Travel Terrain" table with its two footnotes, p. 192.
- SRD 5.2.1, "Playing the Game", "Travel", "Travel Pace" table and "Vehicles"
  (the 24-hour sentence speaks of ships), p. 12.
- SRD 5.2.1, "Environmental Effects", "High Altitude", p. 195.
- SRD 5.2.1, "Rules Glossary": "Exhaustion", p. 181; "Difficult Terrain",
  p. 181; "Speed", p. 188 (does not say what Speed 0 means); "Grappled", p. 182
  and "Paralyzed", p. 186 (set "Speed 0"); "Saving Throws", p. 7.
- SRD 5.2.1, "Airborne and Waterborne Vehicles" table, p. 101, notes, p. 100.
- Out-of-scope list only: "Horseshoes of a Zephyr", p. 226; "Feather Token",
  p. 221.

## 3. Vocabulary

Types **owned by another spec** are imports, not redefined.

| SRD term | Type | Notes |
|---|---|---|
| Travel Pace; "one step" | `TravelPace`, `raisedOneStep` (**imported**, `exploration.md` FR-001, FR-046) | Order slow < normal < fast by enum order; no pace type here (FR-059). |
| Distance Traveled Per Minute / Hour / Day | `TravelPaceRow` (**imported**, `exploration.md` FR-002 to FR-006) | p. 12 table as printed. |
| game effect of a pace | `TravelRules.circumstances(mode, skill)` (**imported**, `exploration.md` FR-007 to FR-012) | See FR-063. |
| a group's pace, a vehicle's speed | `TravelMode` (`LandTravel(pace)`, `VehicleTravel(speed)`), `VehicleSpeed` (**imported**, `exploration.md` FR-014 to FR-017) | `VehicleTravel` also serves the Airship (Q-8). |
| mounts' doubled hour | `TravelRules.mountedHourMiles`, `MountedPaceBoost` (**imported**, `exploration.md` FR-018 to FR-025) | Unused by a plan (Q-10). |
| the eleven terrains | `TravelTerrain` (enum) | Arctic, Coastal, Desert, Forest, Grassland, Hill, Mountain, Swamp, Underdark, Urban, Waterborne. |
| a Travel Terrain row | `TravelTerrainProfile` | Terrain, maximum pace, encounter distance, three DCs. |
| terrain parsed from text | `ParseTravelTerrain(name) (TravelTerrain, error)`; the error is `*UnknownTravelTerrainError{Name}` | FR-006. |
| Maximum Pace column | `TerrainMaximumPace` (`PaceCap`, `PaceCapNeedingEquipment`, `PaceSetByVehicle`) | `Fast*` (Arctic), `Special†` (Waterborne). |
| Encounter Distance | `EncounterDistanceRule` (`feetFor(dice.Result)`, FR-044) | "6d6 x 10 feet"; uses `dice.Expression`. |
| Foraging / Navigation / Search DC | `DifficultyClass` fields | Type owned by `d20-tests.md`, `shared-types-and-ownership.md` FR-002. |
| Good Roads | `RoadPresence` (`goodRoad`, `noRoad`) | p. 192. |
| "Appropriate equipment (such as skis)" | `TerrainEquipment` (`suitable`, `none`) | p. 192 footnote. |
| Slower Travelers | `TravelerSpeed(normal, current)` (`int` feet) | "reduced to half or less of normal". |
| flying / ignoring Difficult Terrain | `TravelMovement` (`ordinary`, `flyingOrIgnoresDifficultTerrain`) | Group-level (FR-062). |
| Speed (feet) | `int` feet, as `conditions.md` D-14 and `MovementSpeeds.walk` | No `TravelSpeed` type (Q-13). |
| miles per hour / day | `TravelDistance` (tenths of a mile) | Speed in feet = tenths of a mile per hour. |
| the distance a plan reports | `TravelPlanDistance` (`TravelPlanDistanceTable(row)`, `TravelPlanDistanceComputed(distance)`) | FR-026. |
| how the rate is chosen | `TravelRates` (`StandardPaceRates`, `SpecialMovementRates(speedFeet)`) | p. 12 table or p. 192 formulas. |
| who is travelling | `TravelJourney` (`TravelOnFoot`, `TravelByVehicle`) | Section 6. |
| hours traveled | `TravelHours` built with `NewTravelHours(actualHours, effectiveHours) (TravelHours, error)` (`MustTravelHours` panics) | Sums from `AssessHighAltitudeAssessed` (`toolbox-environmental-effects.md` FR-090, FR-091, D-4); never doubled here. |
| why hours were refused | `TravelHoursInvalidReason` (`actualBelowOne`, `actualAboveTwentyFour`, `effectiveBelowActual`, `effectiveAboveDoubleActual`), carried by `*TravelHoursError{Reason}` | FR-030. |
| Extended Travel | `ExtendedTravel.scheduleFor(hours)`, `ExtendedTravelSchedule` | DC of each hour past 8. |
| save at one hour | `ExtendedTravelSaveLookup` (`...Due(effect)`, `...NotDue(reason)`), `ExtendedTravelNotDueReason` (`withinEightHours`, `beyondEffectiveHours`) | FR-037, FR-040. |
| services of the plan | `TravelTerrainTable`, `PaceLimits`, `TravelDistances`, `TravelPlanner`, `TravelPlanning` (`TravelPlanningPlanned(plan)`, `TravelPlanningRejected(reason)`) | FR-061. |
| "Constitution saving throw or gain 1 Exhaustion level" | `SaveGatedEffect` with `SaveGatedExhaustionGain(1, untilRemoved)` (**imported**, `toolbox-hazard-effects.md` FR-008, FR-013, FR-044) | FR-037; `ExhaustionLevel` stays in `conditions.md`. |

No type here is named `Race`, `Terrain` alone (clashes with
`initiative-and-actions.md` `Terrain.open/difficult`) or `Journey`.
`TravelRates` is the toolbox's rates type; `exploration.md` deliberately does
not use that name.

## 4. Requirements

A Source cell saying "Interpretation (Q-n)" marks a rule the SRD text does not
state outright; the SRD part and our reading are both written in the cell.  Case tables
abbreviate variants (`Found`, `Unknown`, `Due`, `NotDue`, `Invalid`, `Planned`,
`Rejected`, `Saved`) for the prefixed names in the FR rows. Table sources: p. 192
and the FR rows tested, except package-design rows (FR-016 Speed 0 and negative;
FR-030 hours bounds in 4.3a to 4.3c); 4.3b adds p. 101, p. 195.

### 4.1 Pace and terrain data

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-002 | The terrain table MUST be a toolbox-domain constant (not a content kind, not decoded) with exactly the eleven terrains, each once, with the values of table 4.1a. | SRD 5.2.1, "Travel Terrain", p. 192; `shared-types-and-ownership.md` FR-045 | happy |
| FR-003 | Each encounter distance MUST be the printed die expression times 10 feet (`dice.Expression` `count` and `die` as printed, no flat modifier). | p. 192 | happy |
| FR-004 | Maximum Pace MUST be `PaceCap(pace)` for Coastal, Desert, Forest, Hill, Underdark, Urban (Normal), Grassland (Fast), Mountain, Swamp (Slow); `PaceCapNeedingEquipment(Fast)` for Arctic (footnote *); `PaceSetByVehicle` for Waterborne (footnote dagger). | p. 192, table and footnotes | happy |
| FR-005 | Each profile MUST carry its Foraging, Navigation and Search DCs as `DifficultyClass` exactly as printed (table 4.1a). | p. 192 | happy |
| FR-006 | `ParseTravelTerrain(name)` MUST return the `TravelTerrain` for each of the eleven names, ignoring letter case and leading and trailing whitespace, and a nil error; for any other text (empty, blank, "Plains", "Jungle") it MUST return the zero `TravelTerrain` and a `*UnknownTravelTerrainError` (an error wrapping `srderr.ErrInvalidInput`, `errors.As`-able). Inner whitespace is not collapsed ("Under dark" is unknown). `UnknownTravelTerrainError.Name` MUST carry the text as given, untrimmed. It MUST NOT panic. | p. 192; architecture.md section 6 (untrusted input returns `(T, error)`). Trimming and case rules are package design. | unhappy (invalid input) |
| FR-009 | The terrain table and each profile MUST be immutable: the table is returned by a function as a fresh copy, so mutating a returned slice does not change the next call; two parses of one terrain MUST be equal (`==`). | architecture.md section 4; testing.md section 4 | unhappy (immutability) |

Table 4.1a, Travel Terrain (one test per row, p. 192, happy; encounter distance
is the expression times 10 feet; min and max are the least and greatest
product, used by FR-045):

| Terrain | Maximum Pace | Encounter Distance | min / max (ft) | Foraging DC | Navigation DC | Search DC |
|---|---|---|---|---|---|---|
| Arctic | `PaceCapNeedingEquipment(Fast)` | 6d6 | 60 / 360 | 20 | 10 | 10 |
| Coastal | `PaceCap(Normal)` | 2d10 | 20 / 200 | 10 | 5 | 15 |
| Desert | `PaceCap(Normal)` | 6d6 | 60 / 360 | 20 | 10 | 10 |
| Forest | `PaceCap(Normal)` | 2d8 | 20 / 160 | 10 | 15 | 15 |
| Grassland | `PaceCap(Fast)` | 6d6 | 60 / 360 | 15 | 5 | 15 |
| Hill | `PaceCap(Normal)` | 2d10 | 20 / 200 | 15 | 10 | 15 |
| Mountain | `PaceCap(Slow)` | 4d10 | 40 / 400 | 20 | 15 | 20 |
| Swamp | `PaceCap(Slow)` | 2d8 | 20 / 160 | 10 | 15 | 20 |
| Underdark | `PaceCap(Normal)` | 2d6 | 20 / 120 | 20 | 10 | 20 |
| Urban | `PaceCap(Normal)` | 2d6 | 20 / 120 | 20 | 15 | 15 |
| Waterborne | `PaceSetByVehicle` | 6d6 | 60 / 360 | 15 | 10 | 15 |

Cases 4.1b, terrain lookup (FR-006; one test per row):

| Label | Input | Expected | Source | Path |
|---|---|---|---|---|
| exact name | `"Hill"` | Found, Hill | p. 192 | happy |
| other case | `"mountain"`, `"URBAN"` | Found, Mountain, Urban | p. 192 | unhappy (boundary: case) |
| padded | `"  Hill\t"` | Found, Hill | package design | unhappy (boundary: whitespace) |
| inner space | `"Under dark"` | Unknown, text as given | package design | unhappy (invalid input) |
| not a terrain | `"Plains"`, `"Jungle"` | Unknown | p. 192 | unhappy (invalid input) |
| empty, blank | `""`, `"   "` | Unknown, text as given | architecture.md s.6 | unhappy (empty and minimal) |

### 4.2 Maximum pace and the chosen pace

The group's maximum pace is found in this order: the terrain's maximum; Good
Roads; flying or ignoring Difficult Terrain; Slower Travelers last, because it
says the group "must" move Slow while the others say "can" (Q-5).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-010 | For a `PaceCap(p)` terrain and no other factor, the maximum pace MUST be `p`. | p. 192 | happy |
| FR-011 | On Arctic the base maximum MUST be Fast with `TerrainEquipment.suitable` and Normal with `none`. | p. 192 footnote * ("necessary to keep up a Fast pace"). Interpretation (Q-4): the Normal fallback is ours. | unhappy (rule interaction) |
| FR-012 | `RoadPresence.goodRoad` MUST raise the maximum one step (`raisedOneStep`): Slow gives Normal, Normal gives Fast, Fast stays Fast. | "Good Roads", p. 192 | happy; unhappy (boundary) |
| FR-013 | `goodRoad` on Arctic without suitable equipment MUST NOT raise the maximum above Normal; with equipment it gives Fast. | "Good Roads", footnote *, p. 192. Interpretation (Q-4): equipment still being needed on a road is our reading. | unhappy (rule interaction) |
| FR-014 | `flyingOrIgnoresDifficultTerrain` MUST make the maximum Fast regardless of terrain (Mountain, Swamp, Arctic without equipment, Waterborne) and of `RoadPresence`. | "Special Movement", p. 192 ("regardless of the terrain"). Interpretation (Q-17) for Waterborne: the row is a terrain. | happy; unhappy (rule interaction) |
| FR-015 | If any `TravelerSpeed` has `current * 2 <= normal` the maximum MUST be Slow, whatever the terrain, road or movement. | "Slower Travelers", p. 192 ("half or less of normal"). Interpretation (Q-5): "must" beats flying's "can". | happy; unhappy (rule interaction) |
| FR-016 | A member with `current * 2 > normal` (one foot above half, or Speed increased), or an empty traveler list, MUST NOT trigger FR-015. `TravelerSpeed` with `normal` below 1 or `current` below 0 MUST return an error wrapping `srderr.ErrInvalidInput`. Cases 4.2b. | p. 192; testing.md section 4 | unhappy (boundary; invalid input) |
| FR-017 | `TravelPlanner.plan` on foot MUST accept any requested pace at or below the maximum (Slow always) and MUST return `PlanTravelPaceAboveMaximum` carrying the maximum when above it, changing nothing. | p. 192 ("maximum travel pace"); p. 12 | unhappy (boundary) |
| FR-018 | On foot with `ordinary` movement on Waterborne, `plan` MUST return `PlanTravelVehicleRequired` whatever the pace and Slower Travelers: waterborne rate "depends on the vehicle". With `flyingOrIgnoresDifficultTerrain` it MUST plan normally under FR-014 (Fast, or Slow under FR-015). | footnote dagger, p. 192. Interpretation (Q-17): the footnote applies to a group on foot or swimming. | unhappy (state error; rule interaction) |
| FR-019 | The plan MUST list every factor that changed or fixed the final maximum as `PaceLimit` values (`terrainMaximum`, `noEquipment`, `goodRoad`, `flyingOrIgnoresDifficultTerrain`, `slowerTravelers`) in algorithm order. `goodRoad` is listed whenever Good Roads was applied, even when the maximum stays Fast, and is not listed when `noEquipment` blocks it. `flyingOrIgnoresDifficultTerrain` starts a fresh chain (earlier entries dropped); `slowerTravelers` is appended after it. | architecture.md section 6 | happy |
| FR-050 | A `TravelerSpeed` with `current` 0 and `normal` at least 1 MUST be accepted and trigger FR-015. The plan is still produced, in Slow, under `StandardPaceRates`; it MUST NOT return `PlanTravelGroupCannotMove` (only FR-025 does). | "Slower Travelers", p. 192. Interpretation (Q-18): the SRD does not say what a group does when a member cannot move (Grappled, "Speed 0", p. 182). | unhappy (boundary; rule interaction) |
| FR-062 | `TravelMovement` MUST be one value for the whole group; no mixed-group model. The host passes `flyingOrIgnoresDifficultTerrain` only when every member flies or ignores Difficult Terrain. | "Special Movement", p. 192. Interpretation (Q-20): "the characters" read as all of them. | unhappy (rule interaction) |

Cases 4.2a, maximum pace (one test per row, p. 192; request is the highest
pace; "road" good or none; "flying" is `flyingOrIgnoresDifficultTerrain`;
"slower" is Normal to Current Speed):

| Label | Terrain | Equipment | Road | Movement | Slower | Maximum | Limits (in order) | Path |
|---|---|---|---|---|---|---|---|---|
| plain Hill | Hill | n/a | none | ordinary | none | Normal | terrainMaximum | happy |
| plain Grassland | Grassland | n/a | none | ordinary | none | Fast | terrainMaximum | happy |
| plain Mountain | Mountain | n/a | none | ordinary | none | Slow | terrainMaximum | happy |
| Mountain with road | Mountain | n/a | good | ordinary | none | Normal | terrainMaximum, goodRoad | happy |
| Hill with road | Hill | n/a | good | ordinary | none | Fast | terrainMaximum, goodRoad | happy |
| Grassland with road | Grassland | n/a | good | ordinary | none | Fast | terrainMaximum, goodRoad | unhappy (boundary) |
| Arctic with skis | Arctic | suitable | none | ordinary | none | Fast | terrainMaximum | happy |
| Arctic, no equipment | Arctic | none | none | ordinary | none | Normal | terrainMaximum, noEquipment | unhappy (rule interaction) |
| Arctic, no equipment, road | Arctic | none | good | ordinary | none | Normal | terrainMaximum, noEquipment | unhappy (rule interaction) |
| Arctic, skis, road | Arctic | suitable | good | ordinary | none | Fast | terrainMaximum, goodRoad | happy |
| flying over Mountain | Mountain | n/a | none | flying | none | Fast | flyingOrIgnoresDifficultTerrain | happy |
| flying, Arctic, no equipment | Arctic | none | none | flying | none | Fast | flyingOrIgnoresDifficultTerrain | unhappy (rule interaction) |
| flying over Waterborne | Waterborne | n/a | none | flying | none | Fast | flyingOrIgnoresDifficultTerrain | unhappy (rule interaction) |
| flying over Waterborne, slower | Waterborne | n/a | none | flying | 30 to 15 | Slow | flyingOrIgnoresDifficultTerrain, slowerTravelers | unhappy (rule interaction) |
| on foot over Waterborne | Waterborne | n/a | none | ordinary | none | `PlanTravelVehicleRequired` | none | unhappy (state error) |
| slower on Grassland | Grassland | n/a | none | ordinary | 30 to 15 | Slow | terrainMaximum, slowerTravelers | happy |
| slower beats flying | Grassland | n/a | good | flying | 30 to 15 | Slow | flyingOrIgnoresDifficultTerrain, slowerTravelers | unhappy (rule interaction) |
| slower beats road | Hill | n/a | good | ordinary | 30 to 10 | Slow | terrainMaximum, goodRoad, slowerTravelers | unhappy (rule interaction) |
| a traveler at Speed 0 | Hill | n/a | none | ordinary | 30 to 0 | Slow | terrainMaximum, slowerTravelers | unhappy (boundary) |

Cases 4.2b, Slower Travelers (one test per row, p. 192):

| Label | Normal | Current | `current * 2 <= normal`? | Expected | Path |
|---|---|---|---|---|---|
| exactly half | 30 | 15 | yes | Slow | unhappy (boundary) |
| one above half | 30 | 16 | no | not triggered | unhappy (boundary) |
| below half | 30 | 10 | yes | Slow | happy |
| odd, just under half | 25 | 12 | yes | Slow | unhappy (boundary) |
| negative current | 30 | -1 | n/a | `ErrInvalidInput` error | unhappy (invalid input) |
| Speed 0 | 30 | 0 | yes | Slow (FR-050) | unhappy (boundary) |
| Speed increased | 30 | 40 | no | not triggered | unhappy (boundary) |
| unchanged | 30 | 30 | no | not triggered | happy |
| normal Speed 0 | 0 | 0 | n/a | `ErrInvalidInput` error | unhappy (invalid input) |

### 4.3 Rates, distances, hours and vehicles

Only a group on foot has rates (`TravelRates`): `StandardPaceRates`, the p. 12
table through `TravelPaceRow` (as printed, no computed total, FR-026), or
`SpecialMovementRates(speedFeet)`, the p. 192 formulas for a party that "can
travel at a high Speed for an extended time". The host picks the basis. They
disagree for Fast and Slow and a plan never reconciles them (Q-2).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-020 | Under `SpecialMovementRates` Normal miles per hour MUST be Speed / 10, held exactly as `speedFeet` tenths of a mile (Speed 25 gives 25 tenths). | "Special Movement", p. 192 | happy |
| FR-021 | Normal distance MUST be miles per hour times actual hours, exactly in tenths (`speedFeet * actualHours`), unrounded. | p. 192 (Miles per day, Normal) | happy |
| FR-022 | Fast distance MUST be Normal miles per day times 1 and one third, rounded down to a whole mile (`floor(speedFeet * actualHours * 4 / 30)`). | p. 192 (Fast) | happy; unhappy (boundary: rounding) |
| FR-023 | Slow distance MUST be Normal miles per day times two thirds, rounded down (`floor(speedFeet * actualHours * 2 / 30)`). | p. 192 (Slow) | happy; unhappy (boundary: rounding) |
| FR-024 | Rounding MUST be applied once, to the final Fast or Slow figure, from the unrounded Normal figure, never to Normal. | p. 192. Interpretation (Q-7): only Fast and Slow say "round down". | unhappy (rule interaction) |
| FR-025 | `SpecialMovementRates` MUST return an error wrapping `srderr.ErrInvalidInput` for a negative `speedFeet`. A `speedFeet` of 0 MUST be accepted and `plan` with it MUST return `PlanTravelGroupCannotMove`, changing nothing. Check order: `PlanTravelVehicleRequired` (FR-018), then `PlanTravelGroupCannotMove`, then `PlanTravelPaceAboveMaximum` (FR-017). | "Speed", p. 188 (inference: the page defines Speed as distance covered when moving); "Grappled", p. 182; "Paralyzed", p. 186. Interpretation (Q-18); architecture.md section 6. | unhappy (boundary; invalid input) |
| FR-026 | Under `StandardPaceRates` the distance MUST be `TravelPlanDistanceTable` carrying `TravelPaceRow.of(pace)` as printed and no computed distance: minute, hour and day figures are never derived from one another and no total is computed for arbitrary hours. `TravelPlanDistanceComputed` comes only from `SpecialMovementRates` (FR-020 to FR-024) and vehicles (FR-028). | "Travel Pace" table, p. 12; `exploration.md` FR-005, FR-047, Q-07 (decided; the topic-specific spec wins, `shared-types-and-ownership.md` FR-048) | happy; unhappy (rule interaction) |
| FR-028 | `TravelByVehicle(mode: VehicleTravel)` MUST NOT take or report a pace or maximum pace. Its distance MUST be `TravelRules.milesPerHour(mode)` (`exploration.md` FR-017) times actual hours, exact in tenths (a `VehicleSpeed` is a multiple of 0.5), ignoring the terrain's maximum pace (the terrain still supplies encounter distance and DCs). Serves an Airship too (Q-8). | "Vehicles", p. 192 ("they don't choose a travel pace"); "Vehicles", p. 12 (waterborne vessels only: "don't choose a travel pace", 24-hour days). p. 192 and p. 12 differ for land vehicles; see Q-8; `exploration.md` FR-014, FR-017 | happy |
| FR-030 | `NewTravelHours(actualHours, effectiveHours)` MUST require `actualHours` 1 to 24 whole hours and `effectiveHours` whole, at least `actualHours` and at most twice it (a consistency check mirroring `toolbox-environmental-effects.md` FR-091; computes nothing). Otherwise it MUST return the zero `TravelHours` and a `*TravelHoursError` (an error wrapping `srderr.ErrInvalidInput`) carrying the reason; `MustTravelHours` panics with that error. It never panics otherwise. Negative hours are invalid. When several bounds are broken the reason is the first in this order: `actualBelowOne`, `actualAboveTwentyFour`, `effectiveBelowActual`, `effectiveAboveDoubleActual`. | The 24-hour bound is package design (the only 24-hours-per-day figure is the ship sentence, p. 12); consistency bound: "High Altitude", p. 195, `toolbox-environmental-effects.md` FR-091. | unhappy (boundary; invalid input) |
| FR-031 | This spec never doubles an hour and has no `fromAltitudeBuckets` or `tryFromAltitudeBuckets`. The host runs `AssessHighAltitude` per stretch, sums `actualHours` and `effectiveHours`, and passes both sums to `TravelHours`. Whether an hour is High Altitude and the factor 2 are decided only there; this spec reads no altitude, day count, basis or bucket. | "High Altitude", p. 195 ("counts as 2 hours for the purpose of determining how long that creature can travel"); `toolbox-environmental-effects.md` FR-090, FR-091, D-4 | happy; unhappy (rule interaction) |
| FR-032 | A computed distance MUST use `actualHours`, not `effectiveHours`; the plan MUST also carry `effectiveHours`. | p. 195; p. 192. Interpretation (Q-6): altitude changes "how long that creature can travel", not how far it gets in an hour. | unhappy (rule interaction) |
| FR-033 | `TravelPlanner.plan` and `PlanTravel` MUST be deterministic and MUST NOT alter the command. | testing.md section 4 | unhappy (determinism, immutability) |

Cases 4.3a, Special Movement (Speed in feet, hours whole, distances in miles,
Normal exact; one test per row, p. 192 unless noted):

| Label | Speed | Hours | Normal | Fast | Slow | Path |
|---|---|---|---|---|---|---|
| Speed 30, day of 8 | 30 | 8 | 24 | 32 | 16 | happy |
| Speed 25, day of 8 | 25 | 8 | 20 | 26 (26.67 down) | 13 (13.33 down) | unhappy (boundary: rounding) |
| Speed 35, 7 hours | 35 | 7 | 24.5 | 32 (32.67 down) | 16 (16.33 down) | unhappy (boundary: rounding) |
| high Speed | 120 | 8 | 96 | 128 | 64 | happy |
| rounds to zero | 10 | 1 | 1 | 1 (1.33 down) | 0 (0.67 down) | unhappy (boundary: rounding) |
| full day | 30 | 24 | 72 | 96 | 48 | unhappy (boundary: maximum hours) |
| Speed 0 (p. 188) | 0 | 8 | n/a | n/a | n/a | unhappy (boundary): `PlanTravelGroupCannotMove` |
| negative Speed | -5 | 8 | n/a | n/a | n/a | unhappy (invalid input): `ErrInvalidInput` error |

Cases 4.3b, vehicles and hours (one test per row; Hours is `actual / effective`;
distance in miles, from `actualHours`; sources p. 101, p. 192, p. 195):

| Label | Vehicle speed | Hours | Distance | Effective hours | Path |
|---|---|---|---|---|---|
| Longship, a day of 8 | 3 mph | 8 / 8 | 24 | 8 | happy |
| Rowboat half steps | 1 1/2 mph | 8 / 8 | 12 | 8 | unhappy (boundary: half step) |
| Rowboat, odd hours | 1 1/2 mph | 7 / 7 | 10.5 | 7 | unhappy (boundary: half step) |
| Keelboat | 1 mph | 8 / 8 | 8 | 8 | happy |
| Airship, 24 hours (8 mph p. 101; 24-hour day by analogy with waterborne vessels, p. 12, Q-8) | 8 mph | 24 / 24 | 192 | 24 | unhappy (boundary: maximum hours) |
| altitude counts twice | 3 mph | 8 / 12 | 24 (8 hours actual) | 12 | unhappy (rule interaction) |

Cases 4.3c, `NewTravelHours` (FR-030, FR-031; values are sums from
`AssessHighAltitudeAssessed`, `toolbox-environmental-effects.md` table 4.7b;
one test per row; the constructor returns an error wrapping `srderr.ErrInvalidInput` on each refused row):

| Label | Actual | Effective | Expected | Path |
|---|---|---|---|---|
| no altitude | 8 | 8 | accepted | happy |
| half and half | 8 | 12 | accepted | happy |
| mixed day | 8 | 11 | accepted | unhappy (rule interaction) |
| altitude only | 8 | 16 | accepted (exactly double) | unhappy (boundary: double) |
| altitude all day | 24 | 48 | accepted | unhappy (boundary: maximum) |
| one hour | 1 | 2 | accepted | unhappy (boundary: minimum) |
| no hours | 0 | 0 | `Invalid(actualBelowOne)` | unhappy (empty and minimal) |
| over a day | 25 | 25 | `Invalid(actualAboveTwentyFour)` | unhappy (boundary) |
| negative | -1 | -1 | `Invalid(actualBelowOne)` | unhappy (invalid input) |
| two bounds broken, low | 0 | 5 | `Invalid(actualBelowOne)` (check order) | unhappy (boundary) |
| effective below actual | 8 | 7 | `Invalid(effectiveBelowActual)` | unhappy (boundary) |
| effective just above double | 8 | 17 | `Invalid(effectiveAboveDoubleActual)` | unhappy (boundary) |

### 4.4 Extended Travel

`ExtendedTravel.scheduleFor(hours)` computes which saves are due. Each save is
a `SaveGatedEffect` (`toolbox-hazard-effects.md` FR-008) that the host resolves
with `ResolveSaveGatedEffect`; there is no save-result hierarchy here (Q-11).
Saves are numbered by **effective hour** `k`; which actual hour carries an
effective hour is the host's call (Q-6).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-034 | A character with `effectiveHours` of 8 or fewer MUST have no saves (empty `ExtendedTravelSchedule`). | "Extended Travel", p. 192 ("more than 8 hours per day") | unhappy (boundary: 8) |
| FR-035 | A character with `effectiveHours` `h` above 8 MUST have exactly `h - 8` saves, one at the end of each additional hour; the save at effective hour `k` (9 and up) has DC `10 + (k - 8)` (hour 9 is DC 11). | p. 192 ("10 plus 1 for each hour past 8 hours"). Interpretation (Q-6): "hour" is an effective hour, so an altitude hour can carry two saves. | happy; unhappy (boundary) |
| FR-036 | `ExtendedTravelSchedule` MUST be built per character from that character's `effectiveHours` (each character "must succeed"); different hours give different schedules. It MUST NOT map an effective hour to an actual hour. | p. 192; p. 195. Interpretation (Q-6). | unhappy (rule interaction) |
| FR-037 | `saveAt(k)` for an effective hour with a save MUST return `ExtendedTravelSaveLookupDue` carrying a `SaveGatedEffect` with: save `Ability.constitution`; `DifficultyClass(10 + (k - 8))`; no Disadvantage circumstances; empty automatic-success set; `SaveGatedOnsetImmediate`; `SaveGatedFailure` with empty damage and condition lists and exhaustion `SaveGatedExhaustionGain(1, untilRemoved)`; `SaveGatedNoEffectOnSuccess`; `SaveGatedNoRepeat`; `SaveGatedDurationInstantaneous`. The engine builds the effect only; it rolls and resolves nothing. | p. 192; "Saving Throws", p. 7; `toolbox-hazard-effects.md` FR-008, FR-009, FR-013, FR-016, FR-020 | happy |
| FR-038 | Passing that effect to the real `ResolveSaveGatedEffect` with `SaveGatedInitialExposure(effect, target)`, a `SaveGatedSubject` (no affinities, no circumstances, `SaveGatedNoExemption`) and a rolled success (`D20TestOutcome.success` through `SaveGatedSaveRolled`) MUST give `ResolveSaveGatedEffectSaved` with empty consequences and no Exhaustion. No code added; the test proves FR-037's effect is accepted. | p. 192; `toolbox-hazard-effects.md` FR-033, section 6 | happy |
| FR-039 | With a rolled failure it MUST give `ResolveSaveGatedEffectFailed` whose consequences hold `SaveGatedExhaustionGain` of exactly 1 level at any hour (9, 10, 24, 48), and no damage or condition. The engine MUST NOT touch a `ConditionSet`: the host applies the level through `conditions.md` (immunity, cap, death: FR-123, FR-125). | p. 192 ("gain 1 Exhaustion level"); "Exhaustion", p. 181; `toolbox-hazard-effects.md` FR-036, FR-044, FR-046 | happy |
| FR-040 | `saveAt(k)` for `k` of 8 or less (including 0 and negatives), or beyond the character's `effectiveHours`, MUST return `ExtendedTravelSaveLookupNotDue` with `withinEightHours` or `beyondEffectiveHours`, changing nothing. | testing.md section 4 (state errors) | unhappy (state error) |
| FR-041 | `PlanTravel` on foot MUST include the `ExtendedTravelSchedule` for the command's `TravelHours` (other characters' schedules come from `scheduleFor`, FR-036). `TravelByVehicle` MUST report an empty schedule. | p. 192; "Vehicles", p. 12. Interpretation (Q-8): the SRD says nothing about vehicle passengers' saves. | unhappy (rule interaction) |
| FR-042 | A failed save MUST NOT change the saves of later hours: each `saveAt(k)` is independent and carries no state between calls (same `k`, equal effect). | p. 192 ("At the end of each additional hour") | unhappy (rule interaction) |
| FR-043 | `ExtendedTravelSchedule`, the lookup variants and their fields MUST be immutable; the DC list defensively copied. | architecture.md section 4 | unhappy (immutability) |

Cases 4.4a, schedule (one test per row, p. 192; altitude rows also p. 195):

| Label | Actual | Effective | Saves (DC list) | Path |
|---|---|---|---|---|
| a normal day | 8 | 8 | none | unhappy (boundary) |
| one hour over | 9 | 9 | 11 | unhappy (boundary) |
| three hours over | 11 | 11 | 11, 12, 13 | happy |
| the longest day (p. 12) | 24 | 24 | 11 to 26 (16 saves) | unhappy (boundary: maximum) |
| altitude pushes past 8 | 8 | 12 | 11, 12, 13, 14 | unhappy (rule interaction) |
| a mixed day | 8 | 11 | 11, 12, 13 | unhappy (rule interaction) |
| altitude only reaches 8 | 4 | 8 | none | unhappy (boundary) |
| altitude all day | 24 | 48 | 11 to 50 (40 saves) | unhappy (boundary: maximum) |

Cases 4.4b, `saveAt(k)` (one test per row; schedule of 8 actual, 12 effective
hours; p. 192 unless noted):

| Label | k | Expected | Path |
|---|---|---|---|
| first save | 9 | Due, DC 11, Constitution, Exhaustion gain 1 | happy |
| last save | 12 | Due, DC 14 | unhappy (boundary) |
| the threshold hour | 8 | NotDue, `withinEightHours` | unhappy (boundary) |
| zero | 0 | NotDue, `withinEightHours` | unhappy (empty and minimal) |
| beyond the day (testing.md s.4) | 13 | NotDue, `beyondEffectiveHours` | unhappy (state error) |
| same hour twice | 10, 10 | equal effects | unhappy (determinism) |
| failed at 9 then 10 | 9, 10 | DC 11 effect, DC 12 effect; unchanged by the first | unhappy (rule interaction) |

### 4.5 Encounter distance

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-044 | The rule that the roll total is multiplied by 10 feet MUST be the domain function `EncounterDistanceRule.feetFor(dice.Result)`. `RollEncounterDistance` MUST only request the terrain's `dice.Expression` once through `dice.Roll`, call that function and record telemetry. A fixed roller fed 3, 4, 3, 4, 3, 4 (sum 21) on 6d6 gives 210 feet. | p. 192 | happy |
| FR-045 | For each terrain the distance MUST lie between the table 4.1a `min` and `max` (all dice at 1; all at their sides) and be a multiple of 10 feet. | p. 192 | unhappy (boundary: extremes) |
| FR-046 | `RollEncounterDistanceRolled` MUST carry the `dice.Result`, the terrain and the distance in feet. | architecture.md section 6 | happy |
| FR-047 | A `Roller` returning an error MUST be returned from `RollEncounterDistance` wrapped exactly once as `*srderr.PortError` (`srderr.WrapPort(dice.Port, "roll", err)`), with a nil result, so `errors.Is` and `errors.As` reach the roller's own error; the span (`span.RecordError`) and `srd5e.toolbox.errors` (`error.type` = `srderr.ErrorType(err)`, `dice_roller_error`) record it once; `operations` increments once with result `error` and the duration is recorded (`toolbox-hazard-effects.md` FR-074, FR-078). A cancelled `ctx` is returned as `context.Canceled`, not a result (errors-and-results.md FR-054). | observability.md section 7; testing.md section 4; `dice-and-rolls.md` FR-052; `errors-and-results.md`, `shared-types-and-ownership.md` FR-041 | unhappy (failure) |
| FR-048 | Waterborne encounter distance MUST be 6d6 x 10 feet like any other; it MUST NOT depend on a pace or vehicle. | p. 192 | unhappy (rule interaction) |
| FR-049 | Telemetry MUST NOT change any result: equal results with the no-op telemetry and a recording one. | observability.md section 9 | unhappy (determinism) |
| FR-058 | `RollEncounterDistance` MUST roll through `dice.Roll`, one request. A roller returning a `dice.Result` for a different expression (count, die or modifier) makes `dice.Roll` return an error wrapping `srderr.ErrContractViolation`; the use case MUST return it wrapped as in FR-047, record it once as in FR-047, and return no distance. | `dice-and-rolls.md` FR-050, FR-051, FR-052; testing.md section 4 | unhappy (invalid input from a collaborator) |

### 4.6 Planning, telemetry and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-051 | Each use-case call MUST emit exactly one span, `srd5e.toolbox.plan_travel` or `srd5e.toolbox.roll_encounter_distance`, carrying `srd5e.outcome`, `srd5e.toolbox.result` and `srd5e.toolbox.travel.terrain`. | observability.md sections 5, 6; `toolbox-hazard-effects.md` FR-072 | happy |
| FR-052 | Each call MUST increment `srd5e.toolbox.operations` once with `srd5e.toolbox.operation` (`plan_travel` or `roll_encounter_distance`), `srd5e.outcome` and `srd5e.toolbox.result` from section 7: `planned`, `rejected_pace_above_maximum`, `rejected_vehicle_required`, `rejected_group_cannot_move` for `plan_travel`; `rolled` for `roll_encounter_distance`; `error` when a dice port error leaves the call (only `roll_encounter_distance`), with outcome `failure` (`toolbox-hazard-effects.md` FR-078). | `toolbox-hazard-effects.md` FR-072, section 7 | happy |
| FR-053 | A `PlanTravelPaceAboveMaximum`, `PlanTravelVehicleRequired` or `PlanTravelGroupCannotMove` result MUST set `srd5e.outcome` `failure`, counted once, and leave the span status unset (a refused rule is a normal outcome; error status is for failures such as a returned port error). It MUST emit no log and no `error.type`. `planned` and `rolled` set `success`. | observability.md sections 6, 7; architecture.md section 6; `toolbox-hazard-effects.md` FR-073 (its rejection rows set error status; divergence in Q-22) | unhappy (failure) |
| FR-054 | A `planned` result MUST increment `srd5e.toolbox.travel.mode_chosen` once with `srd5e.toolbox.travel.mode` (`slow`, `normal`, `fast`, `vehicle`) and `srd5e.toolbox.travel.terrain`. A rejection MUST NOT. | observability.md section 5 | happy; unhappy (rejection records nothing extra) |
| FR-055 | A `rolled` result MUST record the feet once in histogram `srd5e.toolbox.travel.encounter_distance` with the terrain. A returned dice port failure MUST record nothing there. | observability.md section 5 | happy; unhappy (failure) |
| FR-056 | Each call MUST record its duration once in `srd5e.toolbox.operation_duration` with `srd5e.toolbox.operation`, via the `Meter` timing operation (no clock read in the package). | `observability-ports.md`; `toolbox-hazard-effects.md` section 7 | happy |
| FR-057 | Metric, span and attribute names and values in section 7 MUST be pinned by tests against the `toolbox` signal constants. No attribute MAY hold an id, DC, distance or amount; neither use case MAY emit a log. | observability.md sections 5, 9; `toolbox-hazard-effects.md` FR-075 | unhappy (invalid input) |
| FR-059 | Package `toolbox` MUST export this spec's names from sections 5 and 6, each once with a doc comment starting with its name (no re-export package) and no generic variant name (`Applied`, `Rejected`, `Succeeded`, `Found`, `NotFound`). No file here MAY declare `TravelPace`, `TravelPaceRow`, `TravelMode`, `LandTravel`, `VehicleTravel`, `VehicleSpeed`, `TravelRules`, `MountedPaceBoost` or `PaceCheckEffect`, and no exported name may duplicate one of the `character` package. | architecture.md sections 4, 7, 9; `exploration.md` FR-043, FR-045, Q-06 | unhappy (name collision) |
| FR-060 | Package `toolbox` MUST import only the kernel packages and the published API of `content`, `d20`, `character` and `combat`, have no cycle, and appear in the context map (architecture.md section 2). The names in section 7 MUST appear in `doc/observability/signals.md`. | architecture.md sections 2, 9; `toolbox-hazard-effects.md` FR-001, FR-002; `architecture-guardrails.md` FR-021 | unhappy (forbidden edge) |
| FR-061 | The rejection and plan-assembly rules (FR-010 to FR-019, FR-025, FR-026, FR-028, FR-032, FR-041, FR-050) MUST live in `TravelPlanner.plan`, returning closed interface `TravelPlanning` (`TravelPlanningPlanned(plan)` or `TravelPlanningRejected(reason: PlanTravelRejection)`). `PlanTravel` MUST only call it, map its result one to one and record telemetry. | architecture.md section 4; testing.md section 3 | happy |
| FR-063 | A `TravelPlan` MUST carry the `TravelMode` it was computed under and MUST NOT carry a list of pace effects; the host calls `TravelRules.circumstances(plan.mode, skill)` (`exploration.md` FR-007 to FR-012, FR-014, FR-015). | p. 12; `exploration.md` FR-007 to FR-011, Q-06 | happy |
| FR-064 | `PlanTravel` and `RollEncounterDistance` MUST NOT call another use case. The host reads `plan.schedule.saveAt(k)`, calls `ResolveSaveGatedEffect` and `AssessHighAltitude` itself; each use case records only its own signals. | `shared-types-and-ownership.md` FR-044; architecture.md section 4 | unhappy (nested telemetry) |

### 4.7 Withdrawn ids

| Id | Status |
|---|---|
| FR-001 | Withdrawn: moved to exploration.md (FR-001); `raisedOneStep`: `exploration.md` FR-046 |
| FR-007 | Withdrawn: moved to exploration.md (FR-002); rows: FR-002 to FR-006 |
| FR-008 | Withdrawn: moved to exploration.md (FR-007); effects: FR-007 to FR-012 |
| FR-027 | Withdrawn: moved to exploration.md (FR-007); the plan's mode: FR-063 |
| FR-029 | Withdrawn: moved to exploration.md (FR-016); `VehicleSpeed` validation |

## 5. Model

Domain files (every file of the travel slice not named `*_usecase.go`): immutable (unexported
fields, validating constructors, value receivers, no setters), pure, synchronous, no ports.

- **Imported, never redefined:** `TravelPace`, `TravelPaceRow`, `TravelMode`,
  `LandTravel`, `VehicleTravel`, `VehicleSpeed`, `TravelRules`
  (`exploration.md`, `character` package); `SaveGatedEffect` and parts,
  `SaveGatedExhaustionGain` (`toolbox-hazard-effects.md`); `DifficultyClass`,
  `D20TestOutcome`, `Ability`, `dice.Expression`, `dice.Result`.
  `AssessHighAltitudeAssessed` is read by the host only (the domain imports
  nothing from it).
- **Value objects:** `TravelTerrain`, `TravelTerrainProfile`,
  `TerrainMaximumPace`, `EncounterDistanceRule`, `RoadPresence`,
  `TerrainEquipment`, `TravelMovement`, `TravelerSpeed`, `TravelHours`,
  `TravelDistance`, `TravelPlanDistance`, `TravelRates`, `TravelJourney`,
  `PaceLimit`, `ExtendedTravelSchedule`, `TravelPlan` (`mode`, maximum pace,
  `PaceLimit` list, rate basis, `TravelPlanDistance`, actual and effective
  hours, schedule). A computed `TravelDistance` is an integer count of tenths
  of a mile; Fast and Slow results are multiples of 10 tenths (FR-022, FR-023).
- **Reason enums and closed interface results:** `TravelHoursInvalidReason`,
  `ExtendedTravelNotDueReason`; `ExtendedTravelSaveLookup`, `TravelPlanning`, `PlanTravelResult`,
  `RollEncounterDistanceResult`, `PlanTravelRejection`
  (`PlanTravelPaceAboveMaximum(maximum)`, `PlanTravelVehicleRequired`,
  `PlanTravelGroupCannotMove`).
- **Domain services (stateless):** `TravelTerrainTable.of`,
  `TravelTerrain.lookup`, `EncounterDistanceRule.feetFor` (FR-002 to FR-006,
  FR-044); `PaceLimits.maximum(profile, equipment, road, movement, travelers)`
  (FR-010 to FR-016, FR-019); `TravelDistances.special(speedFeet, pace, hours)`
  and `.byVehicle(mode, hours)` (FR-020 to FR-024, FR-028; standard rates call
  no distance function, FR-026); `TravelPlanner.plan(terrain, journey, hours)`
  (FR-017, FR-018, FR-025, FR-026, FR-028, FR-032, FR-041, FR-061);
  `ExtendedTravel.scheduleFor(hours)` and `ExtendedTravelSchedule.saveAt(k)`
  (FR-034 to FR-042).
- **Invariants:** `SpecialMovementRates.speedFeet` at least 0,
  `TravelerSpeed.normal` at least 1 and `current` at least 0, `TravelHours`
  actual 1 to 24 and effective from actual to twice actual: the constructors return an
  error wrapping `srderr.ErrInvalidInput` (`MustX` panics). `NewTravelHours` and `ParseTravelTerrain` are the
  non-panicking entry points for host input. `VehicleSpeed` is checked by
  `exploration.md`.
- **Results and errors:** `PlanTravelRejection` variants and
  `ExtendedTravelSaveLookupNotDue` are results; `*UnknownTravelTerrainError` and
  `*TravelHoursError` are the validation errors of host input. **Domain events:** none.

## 6. Interface

Both use cases are synchronous (`Roller` is). `ctx` is the first parameter of each `Run` because each emits a span; both are built with `NewX(...)` and options `WithTelemetry` (default `telemetry.Noop`) and `WithClock`. `PlanTravel.Run(ctx, cmd) PlanTravelResult` has no port and returns no error; `RollEncounterDistance.Run(ctx, cmd) (RollEncounterDistanceResult, error)` returns the error only for a port failure or a cancelled `ctx`.

| Use case | Command | Result | Ports |
|---|---|---|---|
| `PlanTravel` | `PlanTravelCommand(terrain, journey, hours)`; `journey` is `TravelOnFoot(mode: LandTravel, movement, road, equipment, travelers, rates)` or `TravelByVehicle(mode: VehicleTravel)`; `rates` defaults to `StandardPaceRates` | `PlanTravelResult`: `PlanTravelPlanned(plan)`, `PlanTravelRejected(reason: PlanTravelRejection)` | `telemetry.Telemetry` |
| `RollEncounterDistance` | `RollEncounterDistanceCommand(terrain)` | `RollEncounterDistanceResult`: `RollEncounterDistanceRolled(terrain, roll, feet)` | `Roller`, `telemetry.Telemetry` |

- `rates` belongs to `TravelOnFoot` only, so a vehicle with rates or a pace
  cannot be built. `TravelOnFoot.mode` is the `LandTravel` whose pace is the
  requested pace (FR-017).
- **Extended Travel has no use case.** The host takes `plan.schedule.saveAt(k)`
  per effective hour, builds `SaveGatedInitialExposure(effect, target)` and a
  `SaveGatedSubject`, calls `ResolveSaveGatedEffect` with
  `SaveGatedSaveNotRolled`, rolls the save with `ResolveD20Test`, and calls
  again with `SaveGatedSaveRolled`: the two-call pattern of
  `toolbox-hazard-effects.md` section 6 and `environment-hazards-and-objects.md`.
- **Port failure:** `Roller` failures are returned as a `*srderr.PortError`
  (FR-047, FR-058). No port has an expected-absence result.
- **Exports:** package `toolbox` exports the two use cases, commands, results,
  value objects and closed-interface hierarchies declared here; there is no re-export package;
  exploration types come from the `character` package only
  (FR-059).
- **Handoff to the host:** `SaveGatedExhaustionGain.levels` of a failed save
  goes to `ConditionSet.apply` with `Exhaustion`; `plan.mode` goes to
  `TravelRules.circumstances` for each Perception, Survival and Stealth check;
  the host sums altitude hours (FR-031) and supplies adjusted Speeds (section 1).

## 7. Observability

The domain is silent; the use cases carry telemetry. The context-wide signals
`srd5e.toolbox.operations`, `.operation_duration`, `.errors` and attributes
`srd5e.toolbox.operation`, `.result` are **owned by `toolbox-hazard-effects.md`**
section 7; this spec amends their value tables and defines only its own extras.
The seven evaluation questions of observability.md section 2:

1. **What can go wrong?** A rejected plan (`PlanTravelRejected`); the
   `Roller` throws or returns a wrong expression. Out-of-range hours or
   negative Speeds are programmer errors (`ErrInvalidInput` error), recorded by nobody.
2. **Operator signal.** The operations counter by result. A failed Extended
   Travel save is a normal outcome recorded by the hazard use case
   (`srd5e.toolbox.saves`, `save_result` `failed`, `save_ability`
   `constitution`; hazard FR-079).
3. **Diagnosis.** Terrain, mode and result as low-cardinality span attributes.
   The `PaceLimit` list, rate and distance are in the plan.
4. **Counted and timed.** Operations by result, duration per use case, mode per
   terrain, encounter distance.
5. **Existing instrumentation.** Unchanged. `srd5e.toolbox.travel.*` is new and
   collides with neither `srd5e.toolbox.environment.*` nor the hazard family
   (Q-21).
6. **Monitor.** Any `rejected_*` share above a few percent (host misuse), any
   `errors` count, the mode mix per terrain.
7. **Noise.** No logging; one span and metrics per call, no hot loop.

Amendments to the hazard-owned tables:

| Table | Added |
|---|---|
| `srd5e.toolbox.operation` | `plan_travel`, `roll_encounter_distance` |
| `srd5e.toolbox.result`, `plan_travel` | `planned` (`success`); `rejected_pace_above_maximum`, `rejected_vehicle_required`, `rejected_group_cannot_move` (`failure`) |
| `srd5e.toolbox.result`, `roll_encounter_distance` | `rolled` (`success`) |
| spans | `srd5e.toolbox.plan_travel`, `srd5e.toolbox.roll_encounter_distance`, each with `srd5e.outcome`, `srd5e.toolbox.result`, `srd5e.toolbox.travel.terrain` (`arctic`, `coastal`, `desert`, `forest`, `grassland`, `hill`, `mountain`, `swamp`, `underdark`, `urban`, `waterborne`) |
| `srd5e.toolbox.errors` | `error.type` `dice_roller_error` from `roll_encounter_distance` (hazard's value, unchanged) |

Error span status is set on a returned dice port failure only; a `rejected_*`
result is `failure` with the span status unset (FR-053, Q-22), counted once by
the outermost use case.

This spec's own signals (they feed `doc/observability/signals.md`; constants
live once in the `toolbox` signal constants):

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| `srd5e.toolbox.travel.mode_chosen` | counter, `{plan}` | `srd5e.toolbox.travel.mode` (`slow`, `normal`, `fast`, `vehicle`), `srd5e.toolbox.travel.terrain` | Which pace (or a vehicle) groups use, by terrain | Dashboard only; a high Fast share on Mountain means flying or roads dominate, check the host passes the movement mode |
| `srd5e.toolbox.travel.encounter_distance` | histogram, `[ft_i]` | `srd5e.toolbox.travel.terrain` | Rolled encounter distances | Compare per-terrain p50 to the printed expectation (6d6 x 10 averages 210 feet); a flat value means a fixed roller leaked into production |

## 8. Test plan

Levels: unit (`toolbox/travel_*_test.go`, external package `toolbox_test`), use case
(`toolbox/*_usecase_test.go`, real domain, `dicetest.FixedRoller` and the
recording `telemetrytest` bundle from `dicetest`, `telemetrytest`, `contenttest`), architecture
(`internal/archtest`). No contract suite (no new port) and no scenario test
yet. Table-driven tests (4.1a, 4.1b, 4.2a, 4.2b, 4.3a to 4.3c, 4.4a, 4.4b) cite
per case. Rules logic is tested at unit level; use case tests cover
orchestration and telemetry only (testing.md section 3). Parsers (`ParseTravelTerrain`) have a `Fuzz` test.

| Ids | Level, file | Notes |
|---|---|---|
| FR-002 to FR-006, FR-009 | unit, `travel_terrain_test.go` | Table 4.1a one test per row, eleven unique terrains, die count and sides, no modifier, Arctic and Waterborne variants; table 4.1b (`ParseTravelTerrain` never panics, `*UnknownTravelTerrainError.Name` keeps untrimmed text, `errors.Is` `ErrInvalidInput`); mutating a returned table slice leaves the next call unchanged, equal values on repeat; `FuzzParseTravelTerrain` |
| FR-010 to FR-016, FR-019, FR-050, FR-062 | unit, `pace_limits_test.go` | Tables 4.2a and 4.2b incl. `PaceLimit` lists in order, Fast stays Fast, Arctic with and without equipment and road, flying incl. Waterborne, empty list, invalid `TravelerSpeed` returns an `ErrInvalidInput` error; Speed 0 slower traveler gives no `GroupCannotMove`; no per-member movement field (compile-level) |
| FR-017, FR-018, FR-025, FR-026, FR-033 (planner), FR-041, FR-063 | unit, `travel_planner_test.go` | Below, equal and above the maximum for each pace, maximum in the rejection; Waterborne on foot with and without a slower traveler, flying plans; Speed 0 gives `GroupCannotMove`, negative returns an `ErrInvalidInput` error, check order; standard rates carry `TravelPlanDistanceTable` for actual hours 1, 8, 24 and no computed distance, special and vehicle plans carry `Computed`; equal commands equal plans; schedule on foot, empty for vehicle; plan carries `LandTravel` or `VehicleTravel`, no effect list, `TravelRules.circumstances(plan.mode, skill)` returns `exploration.md`'s values for one Fast and one Slow plan |
| FR-020 to FR-024, FR-028, FR-032 | unit, `travel_distances_test.go` | Table 4.3a per column (Speed 25 gives 25 tenths; 26.67, 32.67, 0.67 prove single final rounding); table 4.3b vehicle rows, no pace field, Waterborne and non-Waterborne, an Airship row; "altitude counts twice" uses 8 actual hours |
| FR-030, FR-031 | unit, `travel_hours_test.go` | Table 4.3c: actual 0, 1, 24, 25, negative; effective below, equal, double, above double; each reason; `NewTravelHours` returns a `*TravelHoursError` with the first reason, `MustTravelHours` panics, neither panics otherwise; sums carried unchanged, no `fromAltitudeBuckets` or altitude input (compile-level); 4.3b and 4.4a agree |
| FR-034 to FR-037, FR-040, FR-042, FR-043 | unit, `extended_travel_test.go` | Tables 4.4a and 4.4b (hour 9 gives 11, 24 gives 26, 48 gives 50; hours 8, 0, beyond); two characters with different hours, no actual-hour mapping; `saveAt` field by field against FR-037 and accepted by its constructor (hazard FR-009, FR-013, FR-016, FR-020); two calls in either order equal; immutability |
| FR-038, FR-039 | application | Real `ResolveSaveGatedEffect`: rolled success gives `ResolveSaveGatedEffectSaved` with empty consequences; rolled failure at hours 9, 10, 24, 48 gives exactly 1 level; no port other than observability |
| FR-044 to FR-048, FR-058 | unit + application, `roll_encounter_distance_test.go` | `feetFor` and a `dicetest.FixedRoller` fed 3, 4, 3, 4, 3, 4 give 210; all-ones and all-maximum rolls give table 4.1a min and max; result carries roll, terrain, feet; Waterborne rolls 6d6; failing roller: one wrapped `*srderr.PortError` returned (`errors.Is` reaches the double's error), span error once, `errors` once, `operations` once with `error`, duration recorded, no log; a wrong count, die and modifier in turn: an `ErrContractViolation` error is returned wrapped once, no distance |
| FR-033 (use case), FR-049, FR-051 to FR-057, FR-064 | application, `travel_telemetry_test.go` | Equal `PlanTravelCommand`s give equal results and the command is unchanged; equal results with the no-op telemetry and recording; one span per call with pinned name and attributes; one `operations` increment per call, every result value once incl. `error`; three rejections `failure` with span status unset, no log, no `error.type`; `mode_chosen` for four modes, none on rejection; `encounter_distance` once, not when the roller fails; one duration sample; names pinned against the `toolbox` signal constants, no id or amount attribute, logger stays empty; neither use case takes a use case collaborator (constructor scan), `PlanTravel` records nothing nested |
| FR-061 | unit + application | `PlanTravel` returns the same variant and payload as a direct planner call for every row of tables 4.2a and 4.3b; no rules branching |
| FR-057, FR-059, FR-060, FR-064 | architecture | Exported-name scan: each name once with a doc comment, no generic variant name (including the prefixed variants of FR-006, FR-037, FR-040, FR-061), no duplicate of a `character` exported name, no exploration-owned name declared; `toolbox` imports only the kernel packages and the four contexts' published API, no cycle, in the context map; signal names in `doc/observability/signals.md`; section 7 names pinned against the `toolbox` signal constants (FR-057); constructor scan: neither use case takes a use case collaborator (FR-064) |

- **Doubles:** `dicetest.FixedRoller` (`dice-and-rolls.md` FR-054 to FR-058, FR-082,
  FR-083), a wrong-expression roller, a failing roller (`dicetest.CallbackRoller`) and the recording
  `telemetrytest` bundle come from `dicetest`, `telemetrytest`, `contenttest` (`shared-types-and-ownership.md`
  FR-038). No new double.
- Withdrawn ids (4.7) are tested by their owner.
- **Mutation testing** (testing.md section 8, threshold 80): `gomu` on package `toolbox` (the travel files), run only inside
  Docker with `--cpus` and `--workers` capped at a quarter of the host's processors. Risk lines: comparisons (`current * 2 <= normal`,
  hours 8 and 25, `h - 8`, hours-bound order FR-030), FR-022 and FR-023
  rounding, the planner check order.

## 9. Open questions and decisions

Open: none. Escalated: none. Settled, one line each ("interpretation" = the
SRD is silent or ambiguous; this is the reasonable reading):

- Q-2: `StandardPaceRates` reports the p. 12 table as printed; formulas only under
  `SpecialMovementRates`; never reconciled (`exploration.md` FR-047, Q-07; FR-026).
- Q-3: the host chooses the Speed for `SpecialMovementRates` (FR-025).
- Q-4 (interpretation): Arctic without equipment caps at Normal, even on a Good
  Road: the p. 192 footnote makes equipment "necessary" for Fast (FR-011, FR-013, FR-014).
- Q-5 (interpretation): Slower Travelers wins (FR-015): "must" beats "can" (p. 192).
- Q-6 (interpretation): p. 195 doubles altitude hours for "how long that creature
  can travel", so distance uses actual hours (FR-032) and the schedule effective
  hours, an altitude hour can add two saves (FR-035); the host picks which actual
  hour carries a save (FR-036). Rejected: a second rule for splitting the
  8-hour crossing. Doubling: `toolbox-environmental-effects.md`.
- Q-7: Normal exact in tenths; Fast and Slow rounded once (FR-024).
- Q-8: p. 192 ("Vehicles": vehicle travellers "don't choose a travel pace") and p. 12
  ("Vehicles": land-vehicle travellers "choose a pace as normal"; only waterborne
  vessels "don't choose", up to 24 hours a day) differ. Interpretation following
  p. 12: animal-drawn land vehicles are `TravelOnFoot`; waterborne vessels and the
  Airship (`exploration.md` FR-014) are `VehicleTravel` with no Extended Travel
  schedule (FR-041); a host ruling otherwise calls `ExtendedTravel.scheduleFor`.
- Q-10: `exploration.md` owns mounts (FR-018 to FR-025); a plan never applies the
  boost, the host adds the doubled hour; no `TravelPlan` field (p. 12).
- Q-11: Extended Travel's save is a `SaveGatedEffect` with that hour's DC (FR-037;
  `toolbox-hazard-effects.md` section 5.1).
- Q-12: `exploration.md` owns pace effects; `hiding-searching-and-senses.md`
  should point its "Travel Pace modifiers" exclusion there (outside-spec).
- Q-13: No Speed type; `int` feet (`conditions.md` D-14). Q-20: one
  `TravelMovement` per group (FR-062).
- Q-17 (interpretation): flying or terrain-ignoring groups on Waterborne plan at
  maximum Fast (FR-014); on foot a vehicle is required (FR-018). The footnote
  describes a group carried by a vehicle; flight is not carried.
- Q-18 (interpretation): p. 188 does not define Speed 0 travel. Speed 0 under
  `SpecialMovementRates` is `PlanTravelGroupCannotMove`; one member at Speed 0
  only triggers Slower Travelers (FR-050).
- Q-19 (package design): `VehicleSpeed` stays on the half-mile grid (p. 101). The
  p. 100 adjustments can leave it (Warship 2.5 mph against wind: 1.25) and the
  SRD gives no rounding rule, so the host rounds the adjusted speed down to a
  half mile (section 1); `exploration.md` documents it (amendment).
- Q-21: Own extras live under `srd5e.toolbox.travel.*` (FR-060 checks duplicates).
- Q-22: A refused plan is `failure` with the span status unset (FR-053,
  observability.md section 7); `toolbox-hazard-effects.md` reconciles its FR-073.

## 10. Changelog impact

Added: the `toolbox` travel rules (`PlanTravel`, `RollEncounterDistance`,
`TravelPlanner`, `ExtendedTravel`, the Travel Terrain table), the
`srd5e.toolbox.travel.*` signals and the two new operations. Not **BREAKING**;
`doc/observability/signals.md` entries ride with it.
