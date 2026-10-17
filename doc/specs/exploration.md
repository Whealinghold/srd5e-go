# Exploration

| | |
|---|---|
| Status | draft |
| Context | `character` |
| Roadmap phase | 3 |
| Depends on | `modifiers-and-advantage.md`, `content-schema-and-decoder.md`, `derived-stats.md`, `character-aggregate.md` |

Pure domain, upstream of the `toolbox` context (`shared-types-and-ownership.md` FR-045). Two slices, both in `character`: travel (4.1 to 4.4) and light sources and fuel (4.5). `LightLevel` is a `content` type owned by `content-schema-and-decoder.md` FR-133 (`shared-types-and-ownership.md` FR-053), so the light slice needs no `combat` type. See [../roadmap.md](../roadmap.md); `shared-types-and-ownership.md` FR-014.

**Consumed by (non-normative).** `toolbox-travel-pace.md` imports `TravelPace`, `TravelPaceRow`, `TravelMode`, `VehicleSpeed` and `TravelRules` (Q-06). Its own pace, pace-row, pace-effect and vehicle-speed requirements are withdrawn there (`shared-types-and-ownership.md` FR-048): `toolbox` is downstream of `character`, so a type both need lives here (`shared-types-and-ownership.md` FR-013, FR-045).

## 1. Purpose and scope

The "Exploration" section of "Playing the Game" (SRD 5.2.1, pp. 11-12) has seven parts, most specified by sibling specs. This spec is the **index and the owner of the gaps**:

1. It decides, part by part, what is modellable and what is Game Master (GM) adjudication (table 1a), and points each modellable part at its one owning spec.
2. It **owns Travel** (p. 12): the Travel Pace table (Fast, Normal, Slow), each pace's game effect (Advantage or Disadvantage on named checks), the mounts' doubled hour and the rest they then need, and the rule that travellers in a vehicle with a Speed use that speed and choose no pace. Single owner of `TravelPace`, `TravelPaceRow`, `TravelRules.Circumstances` and `VehicleSpeed`.
3. It **owns light sources** (4.5): how far a Torch, Candle, Lamp or Lantern casts Bright Light and Dim Light, and how long its fuel burns (SRD 5.2.1, "Adventuring Gear", pp. 96-100). The engine has no map: the host reports distances.

Everything is a pure function or immutable value over host-supplied values (distances, elapsed time, the pace chosen, whether a rest finished). No dice, no port, no clock.

### 1a. What is modellable and what is GM adjudication

| SRD part (printed page) | Modellable? | Where it lives |
|---|---|---|
| Adventuring Equipment (p. 11) | No rule ("can help"). | Items: `content-equipment.md`. Light radii and fuel: 4.5. |
| Vision and Light: Obscured Areas, Light, Special Senses (p. 11) | Yes. | `hiding-searching-and-senses.md` FR-003 to FR-050 (`LightLevel`, `Obscurement`, `SpecialSense` are `content` types). |
| Which light level a place has (daylight, a full moon, "an unlit dungeon") (p. 11) | **GM adjudication**: examples only, no rule. | The host, into `SpaceVisibility` (`hiding-searching-and-senses.md` FR-006, FR-007). |
| "Patchy fog", "moderate foliage", "heavy fog", "dense foliage" (p. 11) | **GM adjudication.** | `hiding-searching-and-senses.md` FR-007 (`otherObscurement`). |
| Hiding (p. 11; "Hide [Action]", p. 183) | Yes. "The Game Master decides when circumstances are appropriate for hiding." | `hiding-searching-and-senses.md` FR-051 to FR-073. |
| Interacting with Objects: What Is an Object? (p. 12) | Yes, as definition and data. Discrete item or composite is the GM's call. | `environment-hazards-and-objects.md` FR-066 to FR-093. |
| Time-Limited Object Interactions (p. 12) | Yes: one free interaction per turn. | `initiative-and-actions.md` FR-058, FR-040. |
| Finding Hidden Objects (p. 12) | Partly. A search "nowhere near a hidden object" reveals nothing: an input. **GM adjudication:** the vicinity and the DC. | `hiding-searching-and-senses.md` FR-074 to FR-082 (`SearchProximity`, FR-082). |
| Carrying Objects (p. 12) | Yes, when the GM asks. | `equipment-rules.md` FR-011 to FR-020. |
| Breaking Objects (p. 12) | Yes for the Object AC and HP tables and Damage Threshold. **GM adjudication:** automatic breaks, damage types, composite collapse. | `environment-hazards-and-objects.md` FR-066 to FR-093 (FR-086). |
| Marching Order sidebar (p. 12) | **Not modelled**: advice, no rule, no number. The host keeps an ordered list of ids. | Nothing (Q-09). |
| Hazards (p. 12; glossary p. 182) | Yes. | `environment-hazards-and-objects.md` FR-001 to FR-065. |
| Travel Pace, pace effects, mounts, Vehicles (p. 12) | Yes. | **This spec**, 4.1 to 4.4. |
| "The GM can summarize this travel without calculating exact distances" (p. 12); "ships might travel up to 24 hours per day" | **GM adjudication.** | The host. |
| Travel Terrain, roads, slower travelers, extended travel, special movement, high altitude, weather | Yes, on Gameplay Toolbox pages (192, 195). | The `toolbox` specs (Q-06). |

Non-normative (the `SpaceVisibility` and other downstream names in this table and in Q-06 are host-side pointers, not types this spec assumes): a host connects these slices to `initiative-and-actions.md`, `environment-hazards-and-objects.md`, `hiding-searching-and-senses.md`, `equipment-rules.md`, `content-equipment.md` (vehicle `speedMph`) and the toolbox specs. This spec imports nothing from them, and `character` assumes no type from `combat` or `toolbox`.

**Out of scope:**

- Everything marked "Not modelled" or "GM adjudication" in table 1a.
- **Rolling.** A check made while travelling is a D20 Test resolved by `d20.Resolver.Resolve`; this spec supplies only the pace's Advantage or Disadvantage (FR-007 to FR-012).
- **Combat.** "While traveling outside combat" (p. 12): the host calls `TravelRules.Circumstances` only for checks made while travelling (Q-08). Combat movement: `initiative-and-actions.md`, `movement-lifting-and-flight.md`.
- **Effects that create light** (Light, Dancing Lights, Daylight, magic items): spell and magic-item content. Only gear in the "Adventuring Gear" table is modelled (FR-034).
- **Lighting and extinguishing.** A Tinderbox lights a source as a Bonus Action (p. 100), the hood is a Bonus Action (p. 98), extinguishing Oil is Utilize (p. 99): `initiative-and-actions.md`. This spec holds no lit state; the host reports time burned.
- **Gameplay Toolbox travel** (Travel Terrain, Good Roads, Slower Travelers, Extended Travel, Special Movement, Deep Water, Extreme Cold and Heat, Frigid Water, Heavy Precipitation, High Altitude, Slippery Ice, Strong Wind, Thin Ice; pp. 192, 195): the toolbox specs (Q-06).
- **Jumping, Climbing, Swimming, Flying, Burrowing, Difficult Terrain** (glossary pp. 178-189): `movement-lifting-and-flight.md`, `initiative-and-actions.md`.
- **Mount statistics and prices** (p. 12): `content-equipment.md`.
- **Anything not in SRD 5.2.1.** Nothing core to pp. 11-12 was found missing from the PDF. The SRD gives no rule for foraging, navigation or watches on those pages; the Toolbox does (Q-06).

## 2. Source

- SRD 5.2.1, "Playing the Game", "Exploration", pp. 11-12: "Adventuring Equipment", "Vision and Light" ("Obscured Areas", "Light", "Special Senses"), "Hiding", "Interacting with Objects", "Marching Order" (sidebar), "Hazards", "Travel" ("Travel Pace", the Travel Pace table, "Vehicles"), p. 12.
- SRD 5.2.1, "Rules Glossary": "Bright Light", p. 178; "Cone [Area of Effect]", p. 179 (places the Cone's geometry with the host); "Darkness", "Darkvision", p. 180; "Dim Light", p. 181; "Hazard", "Heavily Obscured", p. 182; "Hide [Action]", p. 183; "Lightly Obscured", p. 184; "Long Rest", p. 185; "Short Rest", p. 187; "Utilize [Action]", p. 191.
- SRD 5.2.1, "Equipment", "Adventuring Gear": "Candle", p. 96; "Lamp", "Lantern, Bullseye", "Lantern, Hooded", "Oil" (heading p. 98; "Fuel", p. 99); "Tinderbox", "Torch", p. 100. "Mounts and Other Animals", "Airborne and Waterborne Vehicles" (Speed column), pp. 100-101.
- SRD 5.2.1, "Gameplay Toolbox", "Travel Pace", p. 192 ("Vehicles": "they don't choose a travel pace"; "Special Movement": "typically 8" hours) and "Environmental Effects", p. 195 (coordination only, Q-06).
- SRD 5.2.1, "Passive Perception", p. 186 (Advantage or Disadvantage moves the score by 5), cited for FR-013.
- Printed page numbers equal PDF page numbers (checked: PDF pages 11, 12, 98, 100 and 182).
- Package rules: `doc/rules/architecture.md` §2, §3, §6; `doc/rules/observability.md` §2, §3; `doc/rules/testing.md` §4, §5.

## 3. Vocabulary

Types **owned by this spec** are marked "owned"; others name their owner and are not redefined.

| SRD term | Type | Notes |
|---|---|---|
| Travel Pace; Fast, Normal, Slow | `TravelPace` (enum ordered `slow` < `normal` < `fast`, with `raisedOneStep`), owned | p. 12. The order the toolbox's Good Roads rule relies on (`toolbox-travel-pace.md` FR-012; FR-001, FR-046). |
| "Distance Traveled Per ... Minute, Hour, Day" | `TravelPaceRow` (`feetPerMinute`, `MilesPerHour`, `milesPerDay`, all `int`), owned | The table as printed (FR-002 to FR-006). Not named `TravelRates`: that is the toolbox's closed type. |
| "Each travel pace has a game effect" | `TravelRules.Circumstances(mode, skill) []modifier.RollCircumstance`, owned | Advantage and Disadvantage are the kernel's (`modifiers-and-advantage.md`). |
| Wisdom (Perception or Survival), Dexterity (Stealth) checks | `Skill.Perception`, `Skill.Survival`, `Skill.Stealth` (`content`'s `Skill`) | The API takes a `Skill` and assumes its default ability (Q-16). |
| "choose a pace as normal" / "limited to the speed of the vessel" | `TravelMode` (closed interface with an unexported marker method: `LandTravel{Pace}`, `VehicleTravel{Speed}`), owned | A vehicle with a Speed carries no pace (FR-014, FR-015, Q-14). |
| A vehicle's Speed in mph (p. 101; p. 192) | `VehicleSpeed` (whole half-miles per hour, 1 to 2^52), owned | FR-016. |
| "move twice that distance for 1 hour" | `TravelRules.MountedHourMiles(pace)`, owned | FR-018; Q-03. |
| "the mounts need a Short or Long Rest" | `MountedPaceBoost` (states `available`, `spent`), `MountedBoostSpendResult`, `MountedBoostRestResult`, `MountedBoostRefusalReason` (`mountsNeedRest`), events `MountedPaceBoostSpent`, `MountedPaceBoostRestored`, all owned | FR-019 to FR-025. |
| Short Rest, Long Rest | none: `RestFinished()` takes no rest kind | Either rest restores the mounts alike (FR-024). `RestKind` (`rests-and-hit-dice.md`) is not imported. |
| Bright Light, Dim Light, Darkness | `LightLevel`, owned by `content-schema-and-decoder.md` FR-133 | Imported from package `content`. |
| "Bright Light in a 20-foot radius and Dim Light for an additional 20 feet" | `LightEmission` (shape, `brightFeet`, `dimFeet`), owned | `dimFeet` is the **additional** distance. |
| radius, Cone | `LightShape` (`radius`, `cone`), owned | The Cone's geometry (p. 179) is the host's. |
| (a space relative to a source) | `LightSpace` (`distanceFeet`, `withinCone`), owned | Package name; no map. |
| Equipment that sheds light | `SrdLightSources`, `HoodedLanternHood` (`raised`, `lowered`), owned | FR-034, FR-035. |
| "burns for 1 hour", "burns for 6 hours in a Lamp or Lantern" | `FuelSupply`, `FuelKind` (`candle`, `torch`, `oilFlask`), `FuelBurnResult`, owned | FR-036 to FR-042. |
| Obscured Areas; Special Senses | `Obscurement` (`content-schema-and-decoder.md` FR-135); `SpecialSense` (FR-136); `SenseRanges`, `SightRules.CanSee` (`hiding-searching-and-senses.md` FR-017 to FR-050) | Not redefined. |
| Hiding; Searching | `HideRules` (`hiding-searching-and-senses.md` FR-051 to FR-073); `SearchRules` (FR-074 to FR-082) | Not redefined. |
| Object, Breaking Objects; Hazards | `ObjectStatus`, `ObjectArmorClassTable`, `ObjectHitPointsTable` (`environment-hazards-and-objects.md` FR-066 to FR-093); `HazardKind`, `HazardRecord`, `ResolveFall` (FR-001 to FR-065) | Not redefined. |
| Carrying Capacity; one free interaction per turn | `CarryingCapacity` (`equipment-rules.md` FR-011 to FR-020); `InteractWithObject` (`initiative-and-actions.md` FR-058) | Not redefined. |
| Marching Order | none | Not modelled (Q-09). |

Package names with **no SRD source** (descriptive, they add no rules): `TravelPaceRow`, `TravelMode`, `LandTravel`, `VehicleTravel`, `VehicleSpeed`, `MountedPaceBoost`, `MountedBoostRefusalReason`, `LightEmission`, `LightShape`, `LightSpace`, `SrdLightSources`, `HoodedLanternHood`, `FuelSupply`, `FuelKind`, `LightSourceRules`, `TravelRules`. The name `LightRules` is **taken** by the hiding spec (`obscurementOf`), so this spec uses `LightSourceRules` (FR-043).

Terms use SRD 5.2.1 wording exactly (`doc/rules/architecture.md` §3).

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids. Distances are whole numbers except a vehicle's speed, held as whole half-miles per hour (`VehicleSpeed`, FR-016) and read as a `float64` of miles per hour (exact in binary within the bound).

### 4.1 The Travel Pace table

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `TravelPace` MUST have exactly three values, ordered by speed, slowest first: `slow`, `normal`, `fast` (the order `toolbox-travel-pace.md` FR-012 relies on). The SRD prints the table Fast first; printing order is presentation, not a rule. | SRD 5.2.1, "Travel Pace", p. 12; "Good Roads" ("one step"), p. 192 | happy |
| FR-002 | `TravelPaceRow.Of(pace).feetPerMinute` MUST be 400 for `fast`, 300 for `normal` and 200 for `slow`. | SRD 5.2.1, Travel Pace table, p. 12 | happy |
| FR-003 | `TravelPaceRow.Of(pace).milesPerHour` MUST be 4 for `fast`, 3 for `normal` and 2 for `slow`. | SRD 5.2.1, Travel Pace table, p. 12 | happy |
| FR-004 | `TravelPaceRow.Of(pace).milesPerDay` MUST be 30 for `fast`, 24 for `normal` and 18 for `slow`. | SRD 5.2.1, Travel Pace table, p. 12 | happy |
| FR-005 | The three figures of a pace MUST be exactly the printed ones, held as data and never computed from one another. A test pins every printed figure (cases 4.1a). | SRD 5.2.1, Travel Pace table, p. 12 | happy |
| FR-006 | Every figure MUST be a positive whole number (`int`, no floating point), and in each unit MUST fall strictly from `fast` to `normal` to `slow`. | SRD 5.2.1, Travel Pace table, p. 12 | unhappy (boundary) |
| FR-046 | `TravelPace.RaisedOneStep` MUST map `slow` to `normal`, `normal` to `fast` and `fast` to `fast` ("one step"). The toolbox uses it for Good Roads and does not redefine it. | SRD 5.2.1, "Good Roads", p. 192; `toolbox-travel-pace.md` FR-012 | happy; unhappy (boundary: `fast` stays `fast`) |
| FR-047 | The printed figures are not consistent multiples and MUST NOT be derived from one another: Fast is 400 feet per minute, and 400 x 60 = 24,000 feet is not 4 miles (21,120 feet); the day figure is not the hour figure times 8 for Fast (32, not 30) or Slow (16, not 18). A test asserts the derived values differ from the pinned ones. | SRD 5.2.1, Travel Pace table, p. 12; "Special Movement" (hours "typically 8"), p. 192 | unhappy (rule interaction) |

**Cases 4.1a (FR-002 to FR-005, FR-047). One test case per listed value; a row groups values that share a label.**

| Label | Pace | feet/min | miles/hour | miles/day | Source | Path |
|---|---|---|---|---|---|---|
| Fast | `fast` | 400 | 4 | 30 | p. 12 | happy |
| Normal | `normal` | 300 | 3 | 24 | p. 12 | happy |
| Slow | `slow` | 200 | 2 | 18 | p. 12 | happy |
| Fast day is not hour times 8 | `fast` | `milesPerDay` vs `MilesPerHour * 8` | 30 vs 32, unequal | | pp. 12, 192 | unhappy (rule interaction) |
| Slow day is not hour times 8 | `slow` | `milesPerDay` vs `MilesPerHour * 8` | 18 vs 16, unequal | | pp. 12, 192 | unhappy (rule interaction) |
| Fast minute is not hour over 60 | `fast` | `feetPerMinute * 60` vs `MilesPerHour * 5280` | 24,000 vs 21,120, unequal | | p. 12 | unhappy (rule interaction) |

**Cases 4.1b (FR-046).** `slow` gives `normal` (happy); `normal` gives `fast` (happy); `fast` gives `fast` (unhappy, boundary). Source p. 192. One test per row.

### 4.2 The game effect of a pace

`TravelRules.Circumstances(TravelMode, Skill)` returns the circumstances a pace puts on **one ability check**. It takes a `Skill`, so a saving throw or attack roll cannot be passed. It owns the pace game effects; `PaceCheckEffect`, first declared by `toolbox-travel-pace.md`, is withdrawn there (Q-06).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-007 | For `LandTravel(fast)`, `Circumstances` MUST return exactly one `Disadvantage` for `Skill.Perception`, `Skill.Survival` and `Skill.Stealth`. | SRD 5.2.1, "Travel Pace", "Fast", p. 12 | happy |
| FR-008 | For `LandTravel(normal)`, `Circumstances` MUST return exactly one `Disadvantage` for `Skill.Stealth`, and no circumstance (an empty slice) for `Skill.Perception` and `Skill.Survival`. | SRD 5.2.1, "Travel Pace", "Normal", p. 12 | happy |
| FR-009 | For `LandTravel(slow)`, `Circumstances` MUST return exactly one `Advantage` for `Skill.Perception` and `Skill.Survival`, and no circumstance for `Skill.Stealth`. | SRD 5.2.1, "Travel Pace", "Slow", p. 12 | happy |
| FR-010 | No pace MUST change any skill other than Perception, Survival and Stealth: over all 18 skills, `Circumstances` MUST be non-empty for exactly 3 skills at `fast`, 1 at `normal` and 2 at `slow`. This includes Wisdom skills not named (Insight, Medicine, Animal Handling) and Dexterity skills not named (Acrobatics, Sleight of Hand). | SRD 5.2.1, "Travel Pace", p. 12; "Skills" table, p. 9 | unhappy (boundary) |
| FR-011 | The returned list MUST hold at most one `RollCircumstance`. | SRD 5.2.1, "Travel Pace", p. 12 | unhappy (boundary) |
| FR-048 | The returned slice MUST be a fresh copy: appending to or changing it MUST NOT affect a later call (nil for no circumstance). | `doc/rules/architecture.md` §4; `doc/rules/testing.md` §4 (immutability) | unhappy (immutability) |
| FR-049 | Two calls with equal inputs MUST return equal lists. | `doc/rules/testing.md` §4 (determinism) | unhappy (determinism) |
| FR-012 | A pace circumstance MUST join the roll's other Advantage and Disadvantage only through the kernel `modifier.Resolve`, with no rule added here: they do not stack and cancel each other (cases 4.2b). | SRD 5.2.1, "D20 Tests", "They Don't Stack", p. 8; `modifiers-and-advantage.md` FR-013 to FR-016 | unhappy (rule interaction) |
| FR-013 | Feeding `TravelRules.Circumstances(LandTravel(pace), Skill.Perception)` through `modifier.Resolve` into the Passive Perception function of `derived-stats.md` FR-021 MUST give 15 for `slow`, 10 for `normal` and 5 for `fast` from a base of 10. Only the join is pinned here. | SRD 5.2.1, "Passive Perception", p. 186; "Travel Pace", p. 12 | unhappy (rule interaction) |

**Cases 4.2a (FR-007 to FR-011). One test case per listed value; a row groups values that share a label. "none" is an empty list.**

| Label | Mode, skill | Expected | Req. | Path |
|---|---|---|---|---|
| Fast: Perception, Survival, Stealth | `LandTravel(fast)` | `Disadvantage` each | FR-007 | happy |
| Normal: Stealth | `LandTravel(normal)` | `Disadvantage` | FR-008 | happy |
| Slow: Perception, Survival | `LandTravel(slow)` | `Advantage` each | FR-009 | happy |
| Normal: Perception, Survival | `LandTravel(normal)` | none | FR-008 | unhappy (boundary) |
| Slow: Stealth | `LandTravel(slow)` | none | FR-009 | unhappy (boundary) |
| Not named: Insight, Acrobatics, Sleight of Hand at `fast`; Animal Handling, Athletics at `slow`; Medicine at `normal` | as stated | none | FR-010 | unhappy (boundary) |
| Sweep, all 18 skills | `fast` / `normal` / `slow` | 3 / 1 / 2 non-empty | FR-010 | unhappy (boundary) |

Source for every row: p. 12.

**Cases 4.2b (FR-012). The pace list is combined with `others` by `modifier.Resolve`.** Source p. 8. One test per row.

| Label | Pace circumstance | Other | Roll mode | Dice | Path |
|---|---|---|---|---|---|
| pace alone, Fast Perception | `Disadvantage` | none | `disadvantage` | 2 | happy |
| Slow pace plus another Advantage | `Advantage` | `Advantage` (for example Help) | `advantage` (no stacking) | 2 | unhappy (rule interaction) |
| Slow Perception plus another Disadvantage | `Advantage` | `Disadvantage` (for example from a Lightly Obscured area) | `normal` | 1 | unhappy (rule interaction) |
| Fast Perception plus another Disadvantage | `Disadvantage` | `Disadvantage` (same) | `disadvantage` (one only) | 2 | unhappy (rule interaction) |
| Normal Stealth plus Advantage | `Disadvantage` | `Advantage` | `normal` | 1 | unhappy (rule interaction) |
| no pace effect plus Disadvantage | none | `Disadvantage` | `disadvantage` | 2 | unhappy (empty and minimal) |

### 4.3 Land travel and vehicles

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-014 | `TravelMode` MUST be closed with exactly two variants: `LandTravel(pace)` for any group that chooses a pace (on foot, riding, or in "wagons, carriages, or other land vehicles"), and `VehicleTravel(speed)` for a vehicle that has a Speed (Airborne and Waterborne Vehicles table, p. 101). `VehicleTravel` MUST have no pace field. A `switch` over `TravelMode` MUST be exhaustive without a default. | SRD 5.2.1, "Vehicles", p. 12; "Travel Pace" > "Vehicles", p. 192 (Q-14) | happy |
| FR-015 | `Circumstances(VehicleTravel, skill)` MUST return an empty list for every skill (all 18): a traveller in a vehicle with a Speed does not "choose a travel pace". | SRD 5.2.1, "Vehicles", p. 12; p. 192; Q-04 | unhappy (boundary) |
| FR-016 | `NewVehicleSpeed(halfMilesPerHour int64) (VehicleSpeed, error)` MUST accept whole half-miles per hour from 1 to 2^52 inclusive and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for 0, a negative value or a value above 2^52; its `MilesPerHour` is that number halved, as a `double`, exact for every accepted value. `VehicleSpeed.FromMilesPerHour(double)` MUST accept only a finite positive multiple of 0.5 at most 2^51, and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` otherwise (NaN, infinity, 0, negative, 0.3, above 2^51). The bound exists only to keep the conversion exact. `content-equipment.md` FR-091 bounds a decoded speed only from below (0.5) and to the half step, so a content value above 2^51 reaches `VehicleSpeedFromMilesPerHour` and returns an `srderr.ErrInvalidInput` error; the host guards decoded values or content-equipment adds an upper bound. A host-adjusted speed (half speed against a strong wind, the current added downstream, draft animals; "Vehicles", p. 100) MUST be rounded down by the host to a multiple of 0.5 mph before it builds a `VehicleSpeed`; the type is not widened, so travel distance stays exact in tenths (`toolbox-travel-pace.md` FR-028), and the engine never rounds. | SRD 5.2.1, "Airborne and Waterborne Vehicles" table, p. 101 (speeds 1, 1½, 2, 2½, 3, 4 and 8 mph); "Vehicles", p. 100 (adjusted speeds; no rounding rule, so rounding down is an interpretation: it never credits more distance than the adjustment allows); `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-017 | `TravelRules.MilesPerHour(mode)` MUST return a `float64`: for `LandTravel` the table's `MilesPerHour` converted exactly (`3` gives `3.0`), for `VehicleTravel` the speed's `MilesPerHour`. It MUST NOT scale either by hours or days. | SRD 5.2.1, "Travel Pace", "Vehicles", p. 12 | happy |

**Cases 4.3a (FR-014 to FR-017). One test case per listed value; a row groups values that share a label.** The vehicle speeds are those of `content-equipment.md` FR-048 (Rowboat 1½, Keelboat 1, Airship 8 mph, p. 101); the test builds the speed from plain numbers.

| Label | Input | Expected | Source | Path |
|---|---|---|---|---|
| wagon on Normal | `LandTravel(normal)` `MilesPerHour` | `3.0` (a `float64`) | p. 12 | happy |
| Rowboat | `VehicleTravel(VehicleSpeed(3))` `MilesPerHour` | 1.5 | pp. 12, 101 | happy |
| Keelboat | `VehicleTravel(VehicleSpeed(2))` `MilesPerHour` | 1.0 | pp. 12, 101 | happy |
| Airship | `VehicleTravel(VehicleSpeed(16))` `MilesPerHour` | 8.0 | pp. 101, 192 | happy |
| vehicle, Stealth; vehicle, Perception | `VehicleTravel(VehicleSpeed(3))` `Circumstances` | none | p. 12 | unhappy (boundary) |
| zero half-miles; negative | `NewVehicleSpeed(0)`; `NewVehicleSpeed(-1)` | an `srderr.ErrInvalidInput` error | p. 101 | unhappy (invalid input) |
| at the upper bound | `VehicleSpeed(2^52)` `MilesPerHour` | accepted, exactly 2^51 (`2251799813685248.0`) | FR-016 | unhappy (boundary) |
| one over the upper bound | `NewVehicleSpeed(2^52 + 1)` | an `srderr.ErrInvalidInput` error | FR-016 | unhappy (boundary) |
| from the upper bound | `VehicleSpeedFromMilesPerHour(2^51)` | equals `VehicleSpeed(2^52)` | FR-016 | unhappy (boundary) |
| from one half-mile over | `VehicleSpeedFromMilesPerHour(2^51 + 0.5)` | an `srderr.ErrInvalidInput` error | FR-016 | unhappy (boundary) |
| from a huge value | `VehicleSpeedFromMilesPerHour(1e300)` | an `srderr.ErrInvalidInput` error | FR-016 | unhappy (invalid input) |
| one half-mile | `VehicleSpeed(1)` | accepted, 0.5 mph (not a printed speed; `content-equipment.md` FR-091) | FR-091 | unhappy (boundary) |
| from 1.5 | `VehicleSpeedFromMilesPerHour(1.5)` | equals `VehicleSpeed(3)` | p. 101 | happy |
| from 0; NaN; infinity | `VehicleSpeedFromMilesPerHour(0)`, `(NaN)`, `(+Inf)` | an `srderr.ErrInvalidInput` error | p. 101 | unhappy (invalid input) |
| from 0.3 | `VehicleSpeedFromMilesPerHour(0.3)` | an `srderr.ErrInvalidInput` error (not a multiple of 0.5) | FR-016 | unhappy (invalid input) |

### 4.4 Mounts and the doubled hour

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-018 | `TravelRules.MountedHourMiles(pace)` MUST return twice the pace's `MilesPerHour`: 8 for `fast`, 6 for `normal`, 4 for `slow`. | SRD 5.2.1, "Travel Pace", p. 12 ("twice that distance for 1 hour"); Q-03 | happy |
| FR-019 | A new `MountedPaceBoost` MUST be `available`. | SRD 5.2.1, "Travel Pace", p. 12 | happy |
| FR-020 | `Spend()` on an `available` boost MUST return `MountedBoostSpendAccepted` carrying a `spent` boost and exactly one `MountedPaceBoostSpent` event. | SRD 5.2.1, "Travel Pace", p. 12 | happy |
| FR-021 | `Spend()` on a `spent` boost MUST return `MountedBoostSpendRefused` whose `reason` is `MountedBoostRefusalReason.MountsNeedRest`, MUST leave the state `spent`, and MUST produce no event. | SRD 5.2.1, "Travel Pace", p. 12 ("after which the mounts need a Short or Long Rest") | unhappy (state error) |
| FR-022 | `RestFinished()` on a `spent` boost MUST return `MountedBoostRestRestored` carrying an `available` boost and one `MountedPaceBoostRestored` event. It takes no rest kind: a Short Rest and a Long Rest restore the mounts alike. | SRD 5.2.1, "Travel Pace", p. 12 | happy |
| FR-023 | `RestFinished()` on an `available` boost MUST return `MountedBoostRestUnchanged`, the boost unchanged and no event. | `doc/rules/testing.md` §4 (state errors, repeated action) | unhappy (state error) |
| FR-024 | The states MUST follow the cycle available, spent, available: after a refused second `Spend()` and one `RestFinished`, a further `Spend()` MUST be accepted. The host calls `RestFinished()` only after a rest that conferred its benefits; a rest that conferred none (a Short Rest interrupted, p. 187; a Long Rest interrupted before 1 hour, p. 185) is not reported and changes nothing. | SRD 5.2.1, "Travel Pace", p. 12; "Short Rest", "Interrupting the Rest", p. 187; "Long Rest", "Interrupting the Rest", p. 185 | unhappy (state error) |
| FR-025 | `MountedPaceBoost`, its two results and its two events MUST be immutable value objects: equal under `Equal` (or `==` where comparable) when every field is equal, and no operation changes the instance it was called on. | `doc/rules/architecture.md` §3 (immutability) | unhappy (immutability) |

**Cases 4.4a (FR-018).** Fast 4 gives 8; Normal 3 gives 6; Slow 2 gives 4 (happy, p. 12); `fast` is not the day figure 30 doubled, 60 (unhappy, rule interaction, p. 12). One test per row.

**Cases 4.4b (FR-019 to FR-025). Each row is one call on the state shown.** Cases elide the `CharacterID` argument; every call passes the same fixed `CharacterID`. An extra row: boosts built for two different ids are unequal (FR-055).

| Label | State before | Call | Result | State after | Events | Req. | Path |
|---|---|---|---|---|---|---|---|
| new boost | (new) | read | `available` | `available` | none | FR-019 | happy |
| spend | `available` | `Spend()` | `MountedBoostSpendAccepted` | `spent` | `MountedPaceBoostSpent` | FR-020 | happy |
| spend again | `spent` | `Spend()` | `MountedBoostSpendRefused(mountsNeedRest)` | `spent` | none | FR-021 | unhappy (state error) |
| rest | `spent` | `RestFinished()` | `MountedBoostRestRestored` | `available` | `MountedPaceBoostRestored` | FR-022 | happy |
| rest twice | `spent` | `RestFinished()` twice | restored, then unchanged | `available` | one `MountedPaceBoostRestored` | FR-022, FR-023 | unhappy (state error) |
| rest while available | `available` | `RestFinished()` | `MountedBoostRestUnchanged` | `available` | none | FR-023 | unhappy (state error) |
| full cycle | `available` | spend, spend, rest, spend | accepted, refused, restored, accepted | `spent` | Spent, Restored, Spent | FR-024 | unhappy (state error) |
| input untouched | `spent` | `RestFinished()` | restored | original still `spent` | n/a | FR-025 | unhappy (immutability) |

### 4.5 Light sources and fuel

Implemented in `character` (phase 3). It needs only `LightLevel` (`content-schema-and-decoder.md` FR-133; `shared-types-and-ownership.md` FR-005, FR-053), so no `character` to `combat` edge exists. Its event `LightFuelExhausted` is the one member of the closed `LightFuelEvent` family (FR-055; Q-17). It uses nothing from the travel slice. "Bright Light is normal illumination" (p. 178); torches, lanterns and fires within a specific radius provide Bright Light (p. 11). This section is a pure function from a source and a distance to the level that source alone gives.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-026 | `LightEmission` MUST hold a `LightShape`, `brightFeet` (0 or more) and `dimFeet` (0 or more, the distance **beyond** `brightFeet`), and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when either is negative. | SRD 5.2.1, "Torch", p. 100; `doc/rules/architecture.md` §3 | unhappy (invalid input) |
| FR-050 | `LightEmission` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` when `brightFeet` and `dimFeet` are both 0: a source that sheds no light is not a light source. | SRD 5.2.1, "Torch", p. 100; `doc/rules/architecture.md` §3 | unhappy (invalid input) |
| FR-027 | `LightSourceRules.LevelAt(emission, space)` MUST be `LightLevel.BrightLight` when `brightFeet` is greater than 0 and `space.distanceFeet` is at most `brightFeet` (a distance equal to the radius is inside it). | SRD 5.2.1, "Light", "Bright Light", p. 11; "Torch", p. 100. Edge inclusion is a package decision (Q-05). | happy |
| FR-028 | It MUST be `LightLevel.DimLight` when `space.distanceFeet` is greater than `brightFeet` and at most `brightFeet + dimFeet`. A source with `brightFeet` 0 MUST therefore give `dimLight` at distance 0. | SRD 5.2.1, "Light", "Dim Light", p. 11; "Torch", p. 100; "Lantern, Hooded", p. 98. Edge inclusion is a package decision (Q-05). | happy |
| FR-029 | It MUST be `LightLevel.Darkness` when `space.distanceFeet` is greater than `brightFeet + dimFeet`: the value is "what this source alone gives", not the space's final level (FR-033). | SRD 5.2.1, "Light", p. 11 | unhappy (boundary) |
| FR-030 | `LightSpace` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative distance. A distance of 0 (the source's own space) MUST be accepted and be inside. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-031 | For a `cone` emission, a `LightSpace` with `withinCone` false MUST give `darkness` at every distance, and with `withinCone` true MUST be evaluated by distance as in FR-027 to FR-029. Whether a space lies in the Cone is the host's report. | SRD 5.2.1, "Lantern, Bullseye", p. 98; "Cone [Area of Effect]", p. 179 | unhappy (rule interaction) |
| FR-032 | For a `radius` emission, `withinCone` MUST be ignored. | SRD 5.2.1, "Lamp", p. 98; "Torch", p. 100 | unhappy (rule interaction) |
| FR-033 | `LightSourceRules.Strongest(levels)` MUST return the brightest of the levels in the order `brightLight`, `dimLight`, `darkness`, and `darkness` for an empty list. A host combines ambient light and every source with it. A package decision; the SRD gives no combination rule (Q-02). | SRD 5.2.1, "Light", p. 11 | unhappy (rule interaction) |
| FR-034 | `SrdLightSources` MUST hold exactly the six emissions of table 4.5c, with the printed radii and shapes, and no emission for any item whose text gives no radius. | SRD 5.2.1, "Candle", p. 96; "Lamp", "Lantern, Bullseye", "Lantern, Hooded", p. 98; "Torch", p. 100 | happy |
| FR-035 | `SrdLightSources.HoodedLantern(HoodedLanternHood.Raised)` MUST be the 30-foot-radius emission and `HoodedLantern(HoodedLanternHood.Lowered)` MUST be Dim Light in a 5-foot radius with no Bright Light. The hood is a value the host holds. | SRD 5.2.1, "Lantern, Hooded", p. 98 | unhappy (rule interaction) |
| FR-036 | `NewFreshFuelSupply(characterID, kind)` MUST have `remaining` of 1 hour for `candle`, 1 hour for `torch` and 6 hours for `oilFlask`. These are also the capacities of FR-052. | SRD 5.2.1, "Candle", p. 96; "Torch", p. 100; "Oil", "Fuel", pp. 98-99 | happy |
| FR-037 | `Burn(duration)` with a duration below `remaining` MUST return `FuelBurnContinues` with `remaining - duration`. | SRD 5.2.1, "Oil", "Fuel", p. 99 | happy |
| FR-038 | `Burn(duration)` with a duration equal to `remaining` MUST return `FuelBurnExhausted` with `burnedFor` equal to the duration, a supply with `remaining` zero, and one `LightFuelExhausted` event. | SRD 5.2.1, "Torch", p. 100 | unhappy (boundary) |
| FR-039 | `Burn(duration)` with a duration above `remaining` MUST return `FuelBurnExhausted` whose `burnedFor` is the old `remaining` (never more than the fuel held), a supply with `remaining` zero, and one `LightFuelExhausted` event. | SRD 5.2.1, "Candle", p. 96 | unhappy (boundary) |
| FR-040 | `Burn` MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative duration, checked first, whatever the supply holds. A zero duration on a supply with `remaining` above zero MUST return `FuelBurnContinues`, supply unchanged, no event. On a spent supply FR-042 applies. | `doc/rules/testing.md` §4 | unhappy (invalid input) |
| FR-041 | Burn time MUST be additive over several calls and need not be consecutive: for an Oil flask, burns of 2, 3 and 1 hours MUST exhaust it on the third call, with no rule about when the source was lit or put out. | SRD 5.2.1, "Oil", "Fuel", p. 99 ("That duration doesn't need to be consecutive") | unhappy (rule interaction) |
| FR-042 | `Burn` on a supply whose `remaining` is already zero MUST return `FuelBurnExhausted` with `burnedFor` zero and the supply unchanged, and no second `LightFuelExhausted` event. This is checked after FR-040 and before the duration is compared with `remaining`, so `Burn(Duration.Zero)` on a spent supply is `FuelBurnExhausted`, not `FuelBurnContinues`. | `doc/rules/testing.md` §4 (state errors, repeated action) | unhappy (state error) |
| FR-052 | `RestoreFuelSupply(characterID, kind, remaining)` (reloading a part-burned supply) MUST accept `remaining` from zero up to and including the capacity of the kind (FR-036) and MUST return an error satisfying `errors.Is(err, srderr.ErrInvalidInput)` for a negative `remaining` or one above it. `RestoreFuelSupply` with zero gives a spent supply and no event. `NewFreshFuelSupply(characterID, kind)` equals `RestoreFuelSupply(characterID, kind, capacity)`. | `doc/rules/testing.md` §4; SRD 5.2.1, "Candle", p. 96; "Torch", p. 100; "Oil", p. 99 | unhappy (invalid input) |
| FR-054 | Section 4.5 MUST import `LightLevel` from package `content` and MUST NOT declare it (FR-043). Sections 4.1 to 4.4 do not use it. | `shared-types-and-ownership.md` FR-005, FR-053; `doc/rules/architecture.md` §2 | unhappy (forbidden edge) |

**Table 4.5c (FR-034, FR-035). The six emissions. "Dim" is the additional distance.**

| Item | Shape | Bright (feet) | Dim, additional (feet) | Source |
|---|---|---|---|---|
| Candle | radius | 5 | 5 | "Candle", p. 96 |
| Lamp | radius | 15 | 30 | "Lamp", p. 98 |
| Lantern, Bullseye | cone | 60 | 60 | "Lantern, Bullseye", p. 98 |
| Lantern, Hooded, hood raised | radius | 30 | 30 | "Lantern, Hooded", p. 98 |
| Lantern, Hooded, hood lowered | radius | 0 | 5 (Dim Light in a 5-foot radius) | "Lantern, Hooded", p. 98 |
| Torch | radius | 20 | 20 | "Torch", p. 100 |

**Cases 4.5a (FR-026 to FR-032, FR-050). One test case per listed distance; a row groups values that share a label. `d` is `distanceFeet`; `bright`, `dim`, `dark` are `brightLight`, `dimLight`, `darkness`.** Path for every row is unhappy (boundary) except those marked.

| Label | Emission (bright / dim) | d (and `withinCone`) gives level | Req. | Source |
|---|---|---|---|---|
| Torch | radius 20 / 20 | 0 bright (happy); 20 bright; 21 dim; 40 dim; 41 dark | FR-027 to FR-030 | p. 100 |
| Candle | radius 5 / 5 | 5 bright; 6 dim; 10 dim; 11 dark | FR-027 to FR-029 | p. 96 |
| Lamp | radius 15 / 30 | 15 bright; 16 dim; 45 dim; 46 dark | FR-027 to FR-029 | p. 98 |
| hood lowered | radius 0 / 5 | 0 dim (not bright); 5 dim; 6 dark | FR-028, FR-029 | p. 98 |
| hood raised | radius 30 / 30 | 30 bright; 60 dim | FR-027, FR-028 | p. 98 |
| Bullseye, inside the Cone | cone 60 / 60 | 60 bright; 61 dim; 120 dim; 121 dark | FR-031 | p. 98 |
| Bullseye, outside the Cone, close (unhappy, rule interaction) | cone 60 / 60 | d 5 outside: dark | FR-031 | p. 98 |
| radius ignores `withinCone` (unhappy, rule interaction) | radius 20 / 20 | d 10 outside: bright | FR-032 | p. 100 |
| only Dim Light | bright 0 / dim 10 | construct: accepted | FR-026 | p. 98 |
| only Bright Light | bright 10 / dim 0 | d 10 bright; d 11 dark | FR-026, FR-027, FR-029 | p. 100 |
| negative distance (unhappy, invalid input) | any | d -1: an `srderr.ErrInvalidInput` error | FR-030 | n/a |
| emission with no light (unhappy, invalid input) | bright 0 / dim 0 | construct: an `srderr.ErrInvalidInput` error | FR-050 | n/a |
| negative radii (unhappy, invalid input) | bright -1 / dim 5; bright 5 / dim -1 | construct: an `srderr.ErrInvalidInput` error | FR-026 | n/a |

**Cases 4.5b (FR-033). One test per row.** Source p. 11.

| Label | Levels | Expected | Path |
|---|---|---|---|
| none | empty list | dark | unhappy (empty and minimal) |
| one source | dim | dim | happy |
| ambient Dim Light and a Torch in its Bright Light | dim, bright | bright | unhappy (rule interaction) |
| ambient Bright Light and a Candle's Dim Light | bright, dim | bright | unhappy (rule interaction) |
| two Dim Light sources | dim, dim | dim | unhappy (rule interaction) |
| Darkness and Dim Light | dark, dim | dim | unhappy (rule interaction) |
| order does not matter | dark, bright, dim and its reverse | bright for both | unhappy (determinism) |

**Cases 4.5d (FR-036 to FR-042, FR-052). Each row is one call on a fresh supply unless stated.** Cases elide the `CharacterID` argument; every call passes the same fixed `CharacterID`. An extra row: supplies of the same kind for two different ids are unequal (FR-055).

| Label | Supply | Call | Expected | Req. | Source | Path |
|---|---|---|---|---|---|---|
| new supplies | `candle`; `torch`; `oilFlask` | read | 1 hour; 1 hour; 6 hours | FR-036 | pp. 96, 100, 98-99 | happy |
| Torch, 30 minutes | `torch` | burn 30 min | continues, 30 min left | FR-037 | p. 100 | happy |
| Torch, 59 minutes | `torch` | burn 59 min | continues, 1 min left | FR-037 | p. 100 | unhappy (boundary) |
| Torch, exactly 1 hour | `torch` | burn 60 min | exhausted, burned 60 min, 0 left, one event | FR-038 | p. 100 | unhappy (boundary) |
| Torch, 61 minutes | `torch` | burn 61 min | exhausted, burned 60 min, one event | FR-039 | p. 100 | unhappy (boundary) |
| Oil, 5 h 59 min | `oilFlask` | burn 5 h 59 min | continues, 1 min left | FR-037 | p. 99 | unhappy (boundary) |
| Oil, exactly 6 hours | `oilFlask` | burn 6 h | exhausted, burned 6 h | FR-038 | p. 99 | unhappy (boundary) |
| Oil, 7 hours | `oilFlask` | burn 7 h | exhausted, burned 6 h | FR-039 | p. 99 | unhappy (boundary) |
| zero burn | `candle` | burn 0 | continues, 1 hour left, supply unchanged | FR-040 | p. 96 | unhappy (boundary) |
| zero burn on a spent supply | spent `torch` | burn 0 | exhausted, burned 0, unchanged, no event (not continues) | FR-040, FR-042 | p. 100 | unhappy (boundary) |
| negative burn (fresh or spent) | `candle`; spent `torch` | burn -1 s | an `srderr.ErrInvalidInput` error | FR-040 | n/a | unhappy (invalid input) |
| Oil in three burns | `oilFlask` | burn 2 h, 3 h, 1 h in turn | continues (4 h left), continues (1 h left), exhausted (burned 1 h) | FR-041 | p. 99 | unhappy (rule interaction) |
| Oil, two burns that overshoot | `oilFlask` | burn 4 h, then 4 h | continues (2 h left), exhausted (burned 2 h) | FR-041, FR-039 | p. 99 | unhappy (rule interaction) |
| already spent | spent `torch` | burn 10 min | exhausted, burned 0, unchanged, no event | FR-042 | p. 100 | unhappy (state error) |
| restore below zero | `torch` | `Restore(torch, -1 s)` | an `srderr.ErrInvalidInput` error | FR-052 | n/a | unhappy (invalid input) |
| restore to zero | `torch` | `Restore(torch, 0)` | spent supply, no event | FR-052 | p. 100 | unhappy (boundary) |
| restore part-burned | `candle` | `Restore(candle, 30 min)` | 30 min left | FR-052 | p. 96 | happy |
| restore to capacity | `torch`; `oilFlask` | `Restore(torch, 60 min)`; `Restore(oilFlask, 6 h)` | equals `Fresh` | FR-052 | pp. 100, 99 | unhappy (boundary) |
| restore one second over | `torch`; `oilFlask` | `Restore(torch, 60 min 1 s)`; `Restore(oilFlask, 6 h 1 s)` | an `srderr.ErrInvalidInput` error | FR-052 | pp. 100, 99 | unhappy (boundary) |

### 4.6 Ownership, purity and exports

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-043 | This spec MUST NOT declare a type another spec owns. Names it MUST NOT declare: `LightLevel`, `LightRules`, `Obscurement`, `SpecialSense`, `SightRules`, `HideRules`, `SearchRules`, `ObjectStatus`, `HazardKind`, `CarryingCapacity`, `RestKind`, `Skill`, and the toolbox names `TravelRates`, `StandardPaceRates`, `SpecialMovementRates`, `PaceCheckEffect`, `TravelByVehicle`. | `shared-types-and-ownership.md` (principle 5, FR-021); `doc/rules/architecture.md` §2 | unhappy (name collision) |
| FR-051 | Every name exported by this spec MUST be unique across the package (the exported-name collision scan, check A7, covering every name in FR-045 and FR-053) and MUST NOT collide with a name exported by `toolbox`, `combat` or `content`. | `shared-types-and-ownership.md` FR-021; `doc/rules/architecture.md` §2 | unhappy (name collision) |
| FR-044 | Every operation MUST be pure and deterministic: equal inputs give equal results in any order, no input is modified, every returned slice is a copy, and no operation reads the clock, uses `math/rand`, or does I/O. Durations, distances and rests arrive as values. This spec defines no use case (`shared-types-and-ownership.md` FR-044). | `doc/rules/architecture.md` §3; `shared-types-and-ownership.md` FR-044 | unhappy (determinism) |
| FR-045 | Package `character` MUST export `TravelPace`, `TravelPaceRow`, `TravelMode`, `LandTravel`, `VehicleTravel`, `VehicleSpeed`, `TravelRules`, `MountedPaceBoost`, `MountedBoostSpendResult` and its variants, `MountedBoostRefusalReason`, `MountedBoostRestResult` and its variants, `MountedPaceBoostEvent`, `MountedPaceBoostSpent` and `MountedPaceBoostRestored`, each with a godoc comment starting with its name, and MUST NOT export a generic variant name (`Applied`, `Rejected`, `Succeeded`). The package `doc.go` lists the inputs and outputs of this slice. | `doc/rules/architecture.md` §4, §7 | happy |
| FR-053 | Package `character` MUST also export `LightEmission`, `LightShape`, `LightSpace`, `SrdLightSources`, `HoodedLanternHood`, `LightSourceRules`, `FuelKind`, `FuelSupply`, `FuelBurnResult` and its variants, `LightFuelEvent` and `LightFuelExhausted`, each with a godoc comment starting with its name, and MUST NOT export a generic variant name. Package `combat` MUST NOT export any of them. (Moved from `combat` to `character` by Q-01; the id is kept.) | `doc/rules/architecture.md` §2, §4, §7; `shared-types-and-ownership.md` FR-013, FR-014, FR-017 | happy |
| FR-055 | Every event of this spec (`MountedPaceBoostSpent`, `MountedPaceBoostRestored`, `LightFuelExhausted`) MUST satisfy `character.Event` (`character-aggregate.md` FR-085) and carry the `CharacterID`. `MountedPaceBoostEvent` and `LightFuelEvent` stay closed families. `MountedPaceBoost` and `FuelSupply` each hold the `CharacterID` of the character whose mounts or light source they track, given at construction (`NewMountedPaceBoost(characterID)`, `NewFreshFuelSupply(characterID, kind)`, `RestoreFuelSupply(characterID, kind, remaining)`), and copy it into every event. Equality includes the id, and the id never changes. | `shared-types-and-ownership.md` FR-019, FR-063; `character-aggregate.md` FR-085 | happy |

### 4.7 Withdrawn ids

None. (Q-12 and Q-13 were review notes, not requirements, and are deleted from section 9.)

## 5. Model

Go naming: enum members are exported typed constants of their enum type (a member written in lower case in a table, such as `heavily`, is the constant with an upper-case first letter, qualified by the type name where unqualified names would collide); sets are deduplicated slices in a documented order; accessors are methods named after the field (`Position()`); parameters stay lower camel case. A stateless rules service written `XRules.Method` is a zero-size struct `XRules` with value-receiver methods, usable as its zero value; consumers that want to replace one declare their own one-method interface.

- **Value objects** (immutable: unexported fields, value receivers): `TravelPaceRow`, `VehicleSpeed`, `TravelMode` (closed interface with an unexported marker method: `LandTravel{Pace}`, `VehicleTravel{Speed}`), `MountedPaceBoost`, `LightEmission`, `LightSpace`, `FuelSupply`. Enums: `TravelPace`, `MountedBoostRefusalReason`, `LightShape`, `HoodedLanternHood`, `FuelKind`. All in `character`.
- **Domain services** (stateless zero-size types): `TravelRules` (`Circumstances`, `MilesPerHour`, `MountedHourMiles`), `LightSourceRules` (`LevelAt`, `Strongest`), `SrdLightSources` (accessor methods returning the printed emissions, `HoodedLantern(hood)`; no package-level mutable state).
- **State transitions** return a new instance plus events (`doc/rules/architecture.md` §3): `MountedPaceBoost.Spend()` and `RestFinished()`, `FuelSupply.Burn(time.Duration)` (`NewFreshFuelSupply` and `RestoreFuelSupply` construct). Results are closed and operation-prefixed (`shared-types-and-ownership.md` FR-017): `MountedBoostSpendResult` (`MountedBoostSpendAccepted`, `MountedBoostSpendRefused`), `MountedBoostRestResult` (`MountedBoostRestRestored`, `MountedBoostRestUnchanged`), `FuelBurnResult` (`FuelBurnContinues`, `FuelBurnExhausted`).
- **Events** (past tense, `shared-types-and-ownership.md` FR-019): `MountedPaceBoostSpent`, `MountedPaceBoostRestored` in the closed `MountedPaceBoostEvent`; `LightFuelExhausted(kind)` in the closed `LightFuelEvent`. Both families satisfy `character.Event` and carry the `CharacterID` (FR-055; Q-17).
- **Invariants:** `VehicleSpeed` 1 to 2^52 half-miles (FR-016); `brightFeet`, `dimFeet` not negative and not both zero (`LightEmission`, FR-026, FR-050); distance not negative (`LightSpace`, FR-030); burn duration not negative (`FuelSupply.Burn`, FR-040); `remaining` between zero and capacity (`Restore`, FR-052).
- **Failures.** Expected outcomes are results or enum values (`MountedBoostSpendRefused`, `FuelBurnExhausted`, `darkness`). Invalid values (non-positive speed, negative distance, radius or duration) return an `srderr.ErrInvalidInput` error from the `NewX` constructor (`MustX` helpers panic with it) (`doc/rules/architecture.md` §6). Constructors still return `(T, error)` (decision 2): values come from the host or from `content` (`content-equipment.md` FR-091 gives a decoded speed no upper bound, so `VehicleSpeedFromMilesPerHour`, which converts content's `float64`, returns an `srderr.ErrInvalidInput` error above 2^51; FR-016). No port, so no infrastructure failure.
- **Imported, never redefined:** `Skill`, `LightLevel` (`content`), `RollCircumstance`, `Advantage`, `Disadvantage`, `RollMode` (kernel), `CharacterID`, `character.Event` (`character-aggregate.md`).

## 6. Interface

- **No use case, port or test double.** A host that wants a check made while travelling calls `TravelRules.Circumstances(mode, skill)` and passes the list in the `RollAttempt` it gives `d20.Resolver.Resolve` (`d20-tests.md`). A use case composing both would only wrap two calls (`doc/rules/architecture.md` §4).
- **Typical flows:** travel check (`Circumstances`, then `modifier.Resolve`, then `d20.Resolver.Resolve`); Passive Perception while travelling (same call with `Skill.Perception`, turned into +5 or -5 by `derived-stats.md` FR-021); mounted hour (`MountedHourMiles(pace)`, then `Spend()`, and `RestFinished()` after a rest); light at a space (`LevelAt` per source, then `Strongest([...ambient, ...sources])`, then the host puts the result where it keeps light, for example `hiding-searching-and-senses.md`'s `SpaceVisibility.Light`: a host-side pointer, not a type this spec assumes).
- **Exports.** Package `character` exports the names in FR-045 and FR-053; there is no re-export package and no `*test` package change (`shared-types-and-ownership.md` FR-038). Time arrives as `time.Duration`; this package never reads a clock.
- **Context map.** `character` may depend on `content` and `d20` (`doc/rules/architecture.md` §2). Both slices use `Skill` and `LightLevel` (`content`), the kernel and `character-aggregate.md` ids and events only. Downstream `toolbox` imports from here (`shared-types-and-ownership.md` FR-045); nothing here imports `toolbox`.

## 7. Observability

**Silent by design** (`doc/rules/observability.md` §3): domain code only, no use case to instrument. Answers to the seven questions (§2):

1. **What can go wrong?** Programmer errors (non-positive speed, negative distance, radius or duration), a wrong rest or distance reported, a pace effect read into a combat check (Q-08).
2. **How would an operator know?** an `srderr.ErrInvalidInput` error is recorded by the span of any enclosing use case; wrong outcomes show in the returned value (`MountedBoostSpendRefused(mountsNeedRest)`, `FuelBurnExhausted`, a `LightLevel`).
3. **Diagnosis needs?** Inputs are host-held values and outputs name the deciding rule. Nothing extra.
4. **Count or time?** Nothing: cheap in-memory functions.
5. **Existing instrumentation still correct?** Yes: `srd5e.d20_test.*` is unaffected (FR-012).
6. **Consumer monitors?** Nothing. 7. **Noisy?** No: nothing is emitted.

| Signal | Type | Attributes | Purpose | Suggested monitoring |
|---|---|---|---|---|
| none | n/a | n/a | n/a | n/a |

`doc/observability/signals.md` is unchanged.

## 8. Test plan

| Id | Level | Notes |
|---|---|---|
| FR-001 to FR-006 | unit | `TravelPace` values order (a `TravelPaces()` accessor, no mutable package slice); 4.1a table-driven for all nine figures; `int`, positive, strictly decreasing fast to slow |
| FR-007 to FR-011 | unit | 4.2a: named, not-named and the three 18-skill sweeps (counts 3, 1, 2); at most one element for every pace and skill |
| FR-012 | unit | Rows of 4.2b through the real kernel `modifier.Resolve` (no double) |
| FR-013 | unit | `character/travel_test.go` (package `character_test`): circumstances through `modifier.Resolve` into Passive Perception (`derived-stats.md` FR-021): base 10 gives 15, 10, 5. No scenario test |
| FR-014 to FR-017 | unit | Exhaustive type switch with no default (enforced by `gochecksumtype`) and no pace member on `VehicleTravel`; all 18 skills empty for `VehicleTravel`; 4.3a rows incl. every an `srderr.ErrInvalidInput` error and bound row; `3.0`, 1.5, 1.0, 8.0 with no scaling |
| FR-018 to FR-025 | unit | 4.4a incl. "not the day figure doubled"; 4.4b rows incl. refusal reason `mountsNeedRest` (the only enum value), full cycle, equality, "input untouched" |
| FR-026 to FR-033 | unit | 4.5a, 4.5b: constructor rows, exact radius edges, distance 0, `brightFeet` 0 at distance 0, Cone inside and outside, order independence |
| FR-034, FR-035 | unit | Table 4.5c: six rows, exact count of six, every other light-less item absent, hood raised and lowered |
| FR-036 to FR-042, FR-052 | unit | 4.5d: `Fresh`, continues, exact, over, zero and negative precedence, multi-burn, spent supply, `Restore` at -1 s, 0, 30 min, capacity and capacity + 1 s |
| FR-043, FR-054 | architecture | Scan that no type named in FR-043 (including `LightLevel`) is declared by this slice; 4.5 imports `LightLevel` from package `content` |
| FR-044 | unit + architecture | Equal results in two orders; inputs unchanged; returned slices are copies; no `os`, `math/rand` or `time.Now` use in the slice |
| FR-045, FR-053 | architecture | Exported-name scan of `character`: each name exported once with a godoc comment, no generic variant name; nothing in `combat` |
| FR-046, FR-047 | unit | 4.1b rows; the three "not derived" rows of 4.1a |
| FR-048, FR-049 | unit | Mutating the returned slice leaves a later call unchanged; equal slices for equal calls |
| FR-050 | unit | "emission with no light" row of 4.5a |
| FR-051 | architecture | Exported-name collision scan across every public package of the module |
| FR-055 | unit | Each event satisfies `character.Event` with the `CharacterID` of the boost or supply; two ids give unequal events; the id survives every transition |

- **Level discipline.** All behaviour is domain logic: unit tests plus architecture scans, no port, double or contract suite (`doc/rules/testing.md` §3). Tests are `character/*_test.go` in package `character_test` (table-driven, `t.Run`, `t.Parallel` where safe), with `Fuzz*` tests for `NewVehicleSpeed` and `VehicleSpeedFromMilesPerHour` and `Example*` functions for `TravelRules.Circumstances`. Rows needing the roll mode use the real kernel.
- **Mutation testing** (`doc/rules/testing.md` §8) is required for every changed domain file (`gomu`, run only inside Docker with `--cpus` and `--workers` capped at a quarter of `nproc`). Boundary rows kill comparison and constant mutants: radius edges (20, 21, 40, 41; 5, 6, 10, 11; 0, 5, 6), fuel edges (59, 60, 61 minutes; 5 h 59 min, 6 h, 7 h), the three columns of each pace, the 18-skill sweeps, and both rests.
- **Requirement comments** cite as in `doc/rules/testing.md` §5, for example `Requirement: doc/specs/exploration.md FR-021 - spending a spent boost is refused (SRD 5.2.1, "Travel Pace", p. 12)`; table-driven tests cite `Requirement: per case`.

## 9. Open questions and decisions

No open questions. Settled ones, one line each:

| Id | Decision | Status |
|---|---|---|
| Q-01 | Section 4.5 lives in `character`, phase 3; `LightLevel` is imported (FR-053). | decided: `shared-types-and-ownership.md` FR-053 |
| Q-02 | Ambient light and sources combine by brightest wins (FR-033), a package decision. | decided |
| Q-03 | "Twice that distance for 1 hour" doubles the hour figure (FR-018); the mounted day total is not modelled (Q-07). | decided |
| Q-04 | A vessel's passenger gets no pace effect (FR-015); a GM applies one by hand. | decided |
| Q-05 | A space exactly at the radius is inside (FR-027, FR-028): the SRD is silent, so a package decision; the boundary rows pin it. | decided |
| Q-06 | This spec owns `TravelPace`, `TravelPaceRow`, `TravelRules.Circumstances` and `VehicleSpeed`; the toolbox keeps `TravelRates` and wraps a `VehicleSpeed` in `TravelByVehicle(speed)`; weather obscurement goes to `SpaceVisibility.OtherObscurement` (`hiding-searching-and-senses.md` FR-007); Extended Travel's Exhaustion goes through `HazardRecord` (`environment-hazards-and-objects.md` FR-002); High Altitude is a multiplier on hours this spec never counts. | decided: `shared-types-and-ownership.md` FR-013, FR-045 |
| Q-07 | No travel total for arbitrary spans: the printed figures are not multiples and the SRD gives no formula (FR-005). | decided |
| Q-08 | `Circumstances` has no combat flag; the host calls it only for checks made while travelling. | decided |
| Q-09 | Marching Order is not modelled: it states no rule; the host keeps an ordered list of ids. | decided |
| Q-10 | Light radii are constants in `SrdLightSources` for 0.1.0 (`content-equipment.md` stores gear as verbatim text, FR-037); a structured `lightEmission` field is deferred with no owner, and a later test would compare it with the constants (`shared-types-and-ownership.md` FR-034). | decided |
| Q-11 | Re-lighting does not matter: the engine tracks time burned, not lit state (FR-041, FR-042). | decided |
| Q-14 | No conflict between p. 12 and p. 192: only the vehicles table (p. 101) prints a Speed, so drawn land vehicles take a pace (`LandTravel`) and every vehicle with a printed Speed, waterborne or airborne, takes `VehicleTravel(speed)` (FR-014). | decided |
| Q-15 | One `MountedPaceBoost` per set of mounts that travel together; a host with mounts resting separately holds one each. Whether a mount has its own rests is `rests-and-hit-dice.md` Q-8. | decided |
| Q-16 | `Circumstances` takes a `Skill` and assumes its default ability ("most often applies to", p. 9); a variant ability is a host adjustment; `(Ability, Skill)` is deferred. | decided |
| Q-18 | An adjusted vehicle speed (p. 100) may leave the half-mile grid; the host rounds down to a 0.5 mph multiple before `VehicleSpeed` (FR-016); the type is not widened. | decided: `toolbox-travel-pace.md` Q-19 |
| Q-17 | Events carry the `CharacterID` (FR-055). | decided: `shared-types-and-ownership.md` FR-019, FR-063 |

## 10. Changelog impact

Under `Unreleased` / **Added**: travel in `character` (`TravelPace`, `TravelPaceRow`, `TravelMode`, `LandTravel`, `VehicleTravel`, `VehicleSpeed`, `TravelRules`, `MountedPaceBoost` with its results, refusal reason and events); light sources in `character` (`LightEmission`, `LightSourceRules`, `SrdLightSources`, `FuelSupply` with its results, `LightFuelEvent`, `LightFuelExhausted`). Adds no metric, span or log, so `doc/observability/signals.md` is unchanged. Not **BREAKING**: all additions. The toolbox spec imports these names (Q-06), so none ships twice.
