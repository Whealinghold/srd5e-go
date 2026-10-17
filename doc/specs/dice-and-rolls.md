# Dice, Rolls and the Roller Port

| | |
|---|---|
| Status | implemented |
| Context | `dice`, `dice/dicetest`, `dice/randroller` |
| Roadmap phase | 0 |
| Depends on | `errors-and-results.md` |

## 1. Purpose and scope

Placement: dice types, roll results, the `Roller` port and the `SeededRoller` adapter in package `dice` (`shared-types-and-ownership.md` FR-001, FR-037, FR-068); doubles and the contract suite in package `dice/dicetest` (FR-038, FR-039 there); the unseeded, concurrency-safe adapter in sub-package `dice/randroller` (section 4.10). Type phase 0 (FR-014 there). Roadmap: [../roadmap.md](../roadmap.md).

Defines the vocabulary every rule needs to talk about dice (a die, a die
expression such as `2d8 + 2`, the result of rolling one), the `dice.Roller`
port through which all randomness enters the module and its failure contract,
the test doubles and contract suite in `dicetest`, and one generic bundled
adapter (`dice.SeededRoller`) so the port has a real implementation and the
contract suite has more than one subject. The domain never rolls: use cases
obtain a `dice.Result` through the port (`dice.Roll`) and hand it to pure
domain code (`doc/rules/architecture.md` §3, "Purity").

**Out of scope:**

- **A "Rolling Dice" section.** SRD 5.2.1 has none (contents page, p. 2, and
  Rules Glossary, pp. 176-191, searched); everything below comes from where the
  SRD *uses* dice (section 2).
- **Notation the PDF never defines:** keep-highest or drop-lowest, exploding or
  rerolled dice, `d%`, `d66`. The SRD prints `1d3` (pp. 200, 215; `Nd3` appears
  on other pages too), so `d3` is a named constructor (FR-001); `d2`, `d7`, `d30`
  are not printed. FR-017 to FR-029 are the minimal reading of the single-term
  forms printed (`d20`, `1d4`, `2d8 + 2`, `D12`, `1d20 − 1`, `1d100`).
- **Multiplied dice** (`1d4 × 10 minutes`, p. 120; `6d6 × 10 feet`, p. 192): a
  consumer multiplies `Total` itself; `ParseExpression` rejects it as malformed.
- **Rules that consume dice**, owned elsewhere: Advantage and Disadvantage, D20
  Tests and natural 1 / 20, Critical Hit triggering, the "0 damage but not
  negative" clamp ("Damage Rolls", p. 16), Heroic Inspiration, Hit Point
  maximum, ability score generation.
- **Types owned elsewhere** (`shared-types-and-ownership.md` FR-001, FR-002,
  FR-010): `d20.Roll` and the rest of `d20`, `modifier.RollMode`,
  `modifier.Modifier`, `ability.Ability`, `ability.Score`, each context's `Event`
  interface, the Heroic Inspiration state.
- Multi-term expressions (`1d6 + 2d4`), dice as modifiers, rerolling policy,
  statistical quality of `SeededRoller`, I/O-backed rollers (consumer adapters
  meeting this contract; the port is synchronous and in-memory).

## 2. Source


- "D20 Tests", p. 6: "Roll 1d20 ... roll two d20s ... the higher one ... the
  lower one". Source of `d20` and `1d20`.
- "Advantage/Disadvantage", "Roll Two D20s", p. 8: an 18 and a 3; the 3 with
  Disadvantage, the 18 with Advantage. Source of "highest" and "lowest".
- "Round Down" sidebar, p. 5: round down when dividing or multiplying leaves a
  fraction.
- "Damage Rolls", p. 16: "You roll the damage dice, add any modifiers".
- "Critical Hits", p. 16: "Roll the attack's damage dice twice, add them
  together ... roll 2d4 for the damage rather than 1d4".
- "Random Generation", p. 21: "Roll four d6s and record the total of the
  highest three dice." (prints "four d6s", not `4d6`).
- "Rogue Features", p. 62: `4d6` (also pp. 46, 107).
- "Traps" (Poisoned Darts), p. 200: "struck by 1d3 darts"; "Magic Items A-Z",
  Cloak of Invisibility, p. 215: "regains 1d3 expended charges". Source of `d3`.
- Multiplied forms, pp. 120 and 192. "Trinkets", p. 26: the `1d100` table (row
  labels such as `01`). "Barbarian", Core Barbarian Traits, p. 28: "D12 per
  Barbarian level".
- "Stat Block Overview", "Hit Points" and Hit Dice by Size, p. 255: d4, d6, d8,
  d10, d12, d20 and per-die averages (2½, 3½, 4½, 5½, 6½, 10½); "roll the die
  expression in parentheses"; "2d8 + 2 Hit Points (average 11)". No maximum and
  no face range is given.
- "Damage Notation", p. 257: "4 (1d4 + 2) damage"; only the spaced `+` form.
- "Deck of Illusions", p. 217: "1d20 − 1" with U+2212. The SRD prints negative
  modifiers with U+2212 (pp. 217, 239, 302) and once with U+2013 (p. 356); never an
  ASCII hyphen. "Stat Block" HP, p. 302: "HP 7 (3d6 − 3)".
- The face range of a die (1 to its sides) is a package-design reading of "Roll
  1d20" (p. 6).


## 3. Vocabulary

| SRD term | Type | Notes |
|---|---|---|
| die / dice (d3 ... d100) | `dice.Die` | Number of sides. Printed: d3, d4, d6, d8, d10, d12, d20, d100; not d7 or d30. Comparable; the zero `Die` is invalid (`IsZero`). |
| die expression | `dice.Expression` | Count of identical dice plus a flat modifier. SRD says "die expression" (pp. 255, 257); the binding name is `Expression` (`shared-types-and-ownership.md` FR-001, FR-068; no stutter, no type `Dice`). Comparable; the zero value is invalid. |
| roll (noun) | `dice.DieRoll` | One die and the face it showed. |
| the roll made for an expression | `dice.Result` | The expression, every `DieRoll`, and the totals. Not comparable (holds a slice); use `Equal`. |
| "roll the damage dice twice" | `Expression.Double()` | Doubles the count only; the modifier is "added as normal" (p. 16). |
| "the highest three dice", "the higher", "the lower" | `Result.SumOfHighest`, `Result.SumOfLowest` | pp. 6, 8, 21. |
| "average" | `Expression.Average` | Rounded down (p. 5). |

`Expression.Modifier()` is a plain `int` (the flat term printed in the
expression), not the kernel `modifier.Modifier` (a roll-time bonus). Exported
names are the single list in FR-067.

## 4. Requirements

Stable ids. Never renumber; new requirements get new ids.

Package-design limits (exported constants): `dice.MinSides` = 2,
`dice.MaxSides` = 1000, `dice.MaxCount` = 1000, `dice.MaxModifier` = 1000000.
"Rejected" for a constructor means it returns an `*dice.InvalidError`: it
matches `srderr.ErrInvalidInput` with `errors.Is`, and `errors.As` yields a typed
`Reason` (`ReasonEmpty`, `ReasonMalformed`, `ReasonCountOutOfRange`,
`ReasonSidesOutOfRange`, `ReasonModifierOutOfRange`, `ReasonValueOutOfRange`,
`ReasonRollCountMismatch`, `ReasonDieMismatch`, `ReasonSelectionOutOfRange`)
and an optional `Detail` string for humans only (`errors-and-results.md`
FR-027, FR-035). `errors.Is(err, &dice.InvalidError{Reason: r})` matches on the
reason alone. Tests assert the reason, never the text. Every `NewX` returning a value object, and `ParseExpression`, has a `MustX`
(except `NewResult` and `NewSeededRoller`) that panics with the error the former
would return; only known-good constants go there.

### 4.1 Die

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-001 | `dice` MUST provide the constructors `D3()`, `D4()`, `D6()`, `D8()`, `D10()`, `D12()`, `D20()`, `D100()` returning dice with 3, 4, 6, 8, 10, 12, 20, 100 sides. | SRD 5.2.1, "Traps" (Poisoned Darts), p. 200, and "Magic Items A-Z" (Cloak of Invisibility), p. 215 (`1d3`, examples); "Hit Points" (Hit Dice by Size), p. 255; "Trinkets", p. 26 | happy |
| FR-002 | `NewDie(sides)` MUST accept any integer from `MinSides` (2) to `MaxSides` inclusive, so content and house rules can express a die the SRD does not print (d2, d7, d30). 2 and 1000 are accepted. | n/a (package design; `architecture.md` §3, "Optional rules") | unhappy (boundary) |
| FR-003 | `NewDie(sides)` MUST be rejected for sides below 2 (1, 0, negative) with `ReasonSidesOutOfRange`. | n/a (package design; a one-sided die cannot be rolled meaningfully) | unhappy (invalid input) |
| FR-004 | `NewDie(sides)` MUST be rejected for sides above `MaxSides` (1001) with `ReasonSidesOutOfRange`. | n/a (package design; keeps arithmetic within `int` on every platform) | unhappy (boundary) |
| FR-005 | Two `Die` values with the same sides MUST compare equal with `==`; different sides MUST be unequal. | n/a (value object, `architecture.md` §3) | happy |

### 4.2 Expression

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-006 | `NewExpression(count, die, modifier)` MUST accept a count from 1 to `MaxCount` and expose `Count()`, `Die()` and `Modifier()` unchanged. `1d20` and `2d8 + 2` are valid. | SRD 5.2.1, "D20 Tests", p. 6; "Hit Points", p. 255 | happy |
| FR-007 | `NewExpression` MUST be rejected for a count below 1 (0, negative) with `ReasonCountOutOfRange`. A zero `Die` is rejected with `ReasonSidesOutOfRange`. | n/a (package design; a roll has at least one die) | unhappy (boundary) |
| FR-008 | `NewExpression` MUST be rejected for a count above `MaxCount` (1001) with `ReasonCountOutOfRange`; 1000 is accepted. | n/a (package design; bounds the work a roller may be asked to do) | unhappy (boundary) |
| FR-009 | `NewExpression` MUST be rejected for a modifier whose magnitude exceeds `MaxModifier` (1000001 and -1000001) with `ReasonModifierOutOfRange`; 1000000 and -1000000 are accepted. | n/a (package design) | unhappy (boundary) |
| FR-010 | `Expression.Minimum` MUST equal `count + modifier` (every die shows 1). See the case table. | n/a (package design; follows from "Roll 1d20", SRD 5.2.1 p. 6; the SRD does not state a die's face range) | happy |
| FR-011 | `Expression.Maximum` MUST equal `count * sides + modifier` (every die shows its highest face). | n/a (package design; the SRD states no maximum, p. 255 gives only per-die averages) | happy |
| FR-012 | `Expression.Average` MUST equal `count * (sides + 1) / 2 + modifier` rounded down (towards negative infinity). See the case table. | SRD 5.2.1, "Hit Points" and Hit Dice by Size, p. 255; "Damage Notation", p. 257; "Round Down", p. 5 | happy |
| FR-013 | Two `Expression` values with equal count, die and modifier MUST compare equal with `==`; any differing field makes them unequal. | n/a (value object) | happy |
| FR-014 | `Double()` MUST return an expression with the count doubled and the same die and modifier. `1d4` gives `2d4`; `1d4 + 3` gives `2d4 + 3`. | SRD 5.2.1, "Critical Hits", p. 16 | happy |
| FR-015 | `Double()` on a count above `MaxCount / 2` (501), or on the zero `Expression`, MUST return an error with `ReasonCountOutOfRange` and the zero `Expression`; 500 gives 1000. A caller avoids it by checking `CanDouble` (FR-069). | n/a (package design; same limit as FR-008) | unhappy (boundary) |
| FR-016 | `Double()` MUST NOT alter the expression it is called on (value receiver). | n/a (immutability, `architecture.md` §3) | unhappy (immutability) |

Case table for FR-010, FR-011, FR-012 (one table-driven test per row):

| Label | Expression | Minimum | Maximum | Average | Source | Path |
|---|---|---|---|---|---|---|
| one d20 | `1d20` | 1 | 20 | 10 (10½ rounded down) | p. 255 (d20 average 10½); p. 5 | happy |
| one d4 plus 2 | `1d4 + 2` | 3 | 6 | 4 (4½ rounded down) | p. 257 prints "4 (1d4 + 2)" | happy |
| two d8 plus 2 | `2d8 + 2` | 4 | 18 | 11 | p. 255 | happy |
| two d4 | `2d4` | 2 | 8 | 5 (2½ per die, twice) | p. 255 | happy |
| one d6 | `1d6` | 1 | 6 | 3 (3½ rounded down) | p. 255 | happy |
| four d6 | `4d6` | 4 | 24 | 14 (exact) | n/a (arithmetic; `4d6` printed on p. 62; p. 21 says "four d6s") | happy |
| one d100 | `1d100` | 1 | 100 | 50 (50½ rounded down) | p. 26 | happy |
| negative modifier (printed) | `3d6 - 3` | 0 | 15 | 7 (10½ - 3 = 7½ rounded down) | p. 302 prints "HP 7 (3d6 − 3)"; rounding p. 5 | happy |
| negative modifier, negative average | `1d4 - 5` | -4 | -1 | -3 (-2½ rounded down) | n/a (package design; rounding per p. 5) | unhappy (boundary) |
| largest allowed | `1000d1000 + 1000000` | 1001000 | 2000000 | 1500500 (500500 + 1000000) | n/a (limits) | unhappy (boundary) |
| smallest allowed | `1d2 - 1000000` | -999999 | -999998 | -999999 (1½ - 1000000 = -999998½ rounded down) | n/a (limits) | unhappy (boundary) |


### 4.3 Parsing and notation

`dice.ParseExpression(string) (Expression, error)` is the function for untrusted
text (`architecture.md` §6). On failure it returns the zero `Expression` and an
`*InvalidError` with one `Reason`: `ReasonEmpty`, `ReasonMalformed`,
`ReasonCountOutOfRange`, `ReasonSidesOutOfRange`, `ReasonModifierOutOfRange`.
One reason per parse; count is checked before sides, sides before modifier (a
parser, not a decoder, so `errors-and-results.md` FR-035 permits one reason; its
FR-023 is for decoders). `MustParseExpression` is the panicking form for
constants.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-017 | `ParseExpression` MUST read the count as optional: `d20` is `1d20`. | SRD 5.2.1, "D20 Tests", p. 6 | happy |
| FR-018 | `ParseExpression` MUST accept `<count>d<sides>` for ASCII-digit counts and sides, for example `1d4`, `2d8`, `4d6`, `1d100`. | SRD 5.2.1, pp. 16 (`2d4`, `1d4`), 26 (`1d100`), 62 (`4d6`) | happy |
| FR-019 | `ParseExpression` MUST accept an upper-case `D` as the die letter (`D12`). | SRD 5.2.1, "Barbarian", p. 28 | happy |
| FR-020 | `ParseExpression` MUST accept an optional modifier `+<n>` or `-<n>` (ASCII `-`), written either with no spaces (`1d4+2`) or with exactly one space on each side of the sign (`2d8 + 2`, `1d4 - 5`). Asymmetric spacing (`1d4 +2`, `1d4+ 2`) and runs of more than one space (`1d4  +  2`) are malformed. `1d20+0`, `1d20-0` and `1d20 + 0` parse to `1d20` with modifier 0. | SRD 5.2.1, "Damage Notation", p. 257; "Hit Points", p. 255 (spaced `+` form only); unspaced, ASCII-minus and zero forms are n/a (package design) | happy |
| FR-021 | `ParseExpression` MUST accept the Unicode minus sign U+2212 as the modifier sign (`1d20 − 1`, `1d20−1`), under the same spacing rule as FR-020. | SRD 5.2.1, "Deck of Illusions", p. 217 | happy |
| FR-022 | `ParseExpression("")` MUST fail with `ReasonEmpty`. (In the tables below a bare reason `x` is shorthand for an `*InvalidError` with `Reason` `x`.) | n/a (invalid input) | unhappy (invalid input) |
| FR-023 | `ParseExpression` MUST fail with `ReasonMalformed` for text that is not of the form in FR-017 to FR-021. See the case table. | n/a (invalid input) | unhappy (invalid input) |
| FR-024 | `ParseExpression` MUST fail with `ReasonCountOutOfRange` for a syntactically valid count below 1 or above `MaxCount` (`0d6`, `1001d6`). | n/a (limits, FR-007, FR-008) | unhappy (boundary) |
| FR-025 | `ParseExpression` MUST fail with `ReasonSidesOutOfRange` for sides below 2 or above `MaxSides` (`1d1`, `1d0`, `1d1001`). | n/a (limits, FR-003, FR-004) | unhappy (boundary) |
| FR-026 | `ParseExpression` MUST fail with `ReasonModifierOutOfRange` for a modifier magnitude above `MaxModifier` (`1d6+1000001`, `1d6-1000001`). | n/a (limits, FR-009) | unhappy (boundary) |
| FR-027 | `ParseExpression` MUST classify numbers too large to represent as an integer (`99999999999999999999d6`, `1d99999999999999999999`, `1d6+99999999999999999999`) as the matching out-of-range reason and MUST NOT panic. | n/a (invalid input) | unhappy (invalid input) |
| FR-028 | `ParseExpression` MUST reject surrounding whitespace, whitespace-only input (`"   "`; only the zero-length string is `ReasonEmpty`), whitespace inside the count or sides, the en dash U+2013 and every other non-ASCII sign or digit as `ReasonMalformed`. | n/a (package design: callers trim deliberately, a lenient parse hides malformed content; spaces around the sign are accepted because the SRD prints them; the en dash is rejected, section 9) | unhappy (invalid input) |
| FR-029 | `ParseExpression` MUST NOT panic for any input string, including very long strings, invalid UTF-8 and strings with control characters. A `Fuzz` test covers this. | `architecture.md` §6 (untrusted input is an error, never a panic) | unhappy (invalid input) |
| FR-081 | `ParseExpression` MUST accept leading zeros in the count, sides and modifier digits (`01d6`, `1d006`, `1d6+007`) and read them as decimal; the value is judged after the zeros are stripped, so `0000000000000000000001d6` is `1d6` and `00d6` is `ReasonCountOutOfRange`. `Notation` (FR-030) never prints leading zeros. | n/a (package design; the SRD's trinket table prints `01`, p. 26, as a row label, not a die expression) | unhappy (boundary) |
| FR-030 | `Expression.Notation` MUST return the canonical machine form: the count always present, lower-case `d`, no spaces, ASCII sign, modifier omitted when 0 (`1d20`, `2d8+2`, `1d4-5`). It is an identifier for content and logs, not display text; `String` returns it. `ParseExpression(e.Notation())` MUST return an expression equal to `e` for every valid expression. | n/a (package design; `architecture.md` §6) | happy |

Case table for FR-017 to FR-021 (accepted input):

| Label | Input | Parsed | Source | Path |
|---|---|---|---|---|
| count omitted or one | `d20`; `1d20` | `1d20` | p. 6 | happy |
| upper-case D | `D12` | `1d12` | p. 28 | happy |
| several dice; percentile-sized die | `4d6`; `1d100` | same | pp. 62, 26 | happy |
| spaced plus | `2d8 + 2`; `1d4 + 2` | same | pp. 255, 257 | happy |
| unspaced plus | `1d4+2` | `1d4 + 2` | n/a (package design) | happy |
| spaced minus | `1d4 - 5` | `1d4 - 5` | n/a (package design; ASCII hyphen, the SRD prints only U+2212) | happy |
| Unicode minus | `1d20 − 1`; `1d20−1` | `1d20 - 1` | p. 217 | happy |
| zero modifier | `1d20+0`; `1d20-0`; `1d20 + 0` | `1d20`, modifier 0 | n/a (package design; follows from "Roll 1d20", p. 6) | unhappy (boundary) |
| leading zeros (FR-081) | `01d6`; `1d006`; `1d6+007`; `0000000000000000000001d6` | `1d6`; `1d6`; `1d6 + 7`; `1d6` | n/a (package design) | unhappy (boundary) |
| d3 | `1d3` | `1d3` | pp. 200, 215 (examples) | happy |
| every limit | `1000d1000+1000000` | `1000d1000 + 1000000` | n/a (limits) | unhappy (boundary) |


Case table for FR-022 to FR-028 (rejected input; every listed input is its own
test case, all path unhappy):

| Reason | Inputs | Kind |
|---|---|---|
| `ReasonEmpty` | `` (zero-length string only) | invalid input |
| `ReasonMalformed` | `d`; `2d`; `2x6`; `1d6+`; `1d6++1`; `1d6 + + 1`; `d+2`; `2 d6`; `2d 6`; `1.5d6`; `1d6+1.5`; `-1d6`; `+1d6` | invalid input |
| `ReasonMalformed` (whitespace) | `   ` (three spaces; not `empty`); ` 1d20`; `1d20 `; `1d6\n`; `1d4 +2`; `1d4+ 2`; `1d4  +  2`; `1d20 −1`; `1d20− 1` (U+2212, asymmetric spacing) | invalid input |
| `ReasonMalformed` (characters) | `1d20 – 1` (en dash U+2013, section 9); `１d20` (full-width 1) | invalid input |
| `ReasonMalformed` (out of scope, section 1) | `1d4 × 10` (multiplied); `1d6+2d4` (two terms) | invalid input |
| `ReasonCountOutOfRange` | `0d6`; `00d6`; `1001d6`; `0d0` (count is checked first); `99999999999999999999d6` | boundary (overflow row: invalid input) |
| `ReasonSidesOutOfRange` | `1d0`; `1d1`; `1d1001`; `1d99999999999999999999` | boundary (overflow row: invalid input) |
| `ReasonModifierOutOfRange` | `1d6+1000001`; `1d6-1000001`; `1d6+99999999999999999999` | boundary (overflow row: invalid input) |

### 4.4 DieRoll and Result

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-031 | `NewDieRoll(die, value)` MUST accept a value from 1 to `die.Sides()` inclusive (1 and `sides` are accepted). | n/a (package design; follows from "Roll 1d20", SRD 5.2.1 p. 6; the SRD does not state a die's face range) | unhappy (boundary) |
| FR-032 | `NewDieRoll(die, value)` MUST be rejected for a value of 0, a negative value, or `sides + 1`, with `ReasonValueOutOfRange`. | n/a (invalid input; a die cannot show a face it does not have) | unhappy (invalid input) |
| FR-033 | `NewResult(expression, rolls)` MUST accept exactly `expression.Count()` rolls, each of `expression.Die()`, and expose `Expression()`, `Rolls()` (in the order given) and `Modifier()`. | SRD 5.2.1, "Damage Rolls", p. 16 | happy |
| FR-034 | `NewResult` MUST be rejected when the number of rolls differs from `expression.Count()` (none, one fewer, one more), with `ReasonRollCountMismatch`. A zero `Expression` is rejected with `ReasonCountOutOfRange`. | n/a (invariant) | unhappy (invalid input) |
| FR-035 | `NewResult` MUST be rejected when any roll is of a different die than `expression.Die()`, with `ReasonDieMismatch`. | n/a (invariant) | unhappy (invalid input) |
| FR-036 | `Result.DiceTotal` MUST be the sum of the face values and `Total` MUST be `DiceTotal + Modifier`. See the case table. | SRD 5.2.1, "Damage Rolls", p. 16; "D20 Tests", p. 6 | happy |
| FR-037 | A result with every die at 1 MUST have `Total() == expression.Minimum()`, and with every die at its highest face `Total() == expression.Maximum()`. | n/a (package design; consistency of FR-010, FR-011 and FR-036) | unhappy (boundary) |
| FR-038 | `Total` MAY be negative or zero (`1d4 - 5` rolling 1 gives -4). `Result` MUST NOT clamp it; the damage clamp belongs to the rules that apply it. | SRD 5.2.1, "Damage Rolls", p. 16 | unhappy (boundary) |
| FR-039 | `Result.Rolls` MUST return a copy: mutating the returned slice MUST NOT change the result. | n/a (`architecture.md` §4, Handoffs) | unhappy (immutability) |
| FR-040 | `NewResult` MUST copy the slice it is given; mutating the original slice afterwards MUST NOT change the result. | n/a (immutability) | unhappy (immutability) |
| FR-041 | `SumOfHighest(n)` MUST return the sum of the `n` highest face values, ignoring the modifier. Ties count normally; roll order does not matter. `Rolls()` MUST be left unchanged in order and content (no in-place sort). See the case table. | SRD 5.2.1, "Random Generation", p. 21; "Roll Two D20s", p. 8 | happy |
| FR-042 | `SumOfLowest(n)` MUST return the sum of the `n` lowest face values, ignoring the modifier. `Rolls()` MUST be left unchanged. | SRD 5.2.1, "D20 Tests", p. 6 ("the lower one"); "Roll Two D20s", p. 8 | happy |
| FR-043 | `SumOfHighest(n)` and `SumOfLowest(n)` MUST return an error with `ReasonSelectionOutOfRange` for `n` below 1 or above `expression.Count()`. With `n == count` they MUST equal `DiceTotal`. | n/a (invalid input) | unhappy (boundary) |
| FR-044 | Two `Result` values with equal expression and equal rolls in the same order MUST be `Equal`; any differing roll or order makes them unequal. | n/a (value object) | happy |

Case table for FR-036 and FR-038:

| Label | Expression | Faces | DiceTotal | Total | Source | Path |
|---|---|---|---|---|---|---|
| two d8 plus 2 | `2d8 + 2` | 3, 6 | 9 | 11 | p. 255 | happy |
| natural 20 | `1d20` | 20 | 20 | 20 | p. 6 | happy |
| natural 1 | `1d20` | 1 | 1 | 1 | p. 6 | unhappy (boundary) |
| d4 plus 2 high | `1d4 + 2` | 4 | 4 | 6 | p. 257 | happy |
| four d6 | `4d6` | 6, 5, 4, 1 | 16 | 16 | n/a (arithmetic; p. 21 "four d6s") | happy |
| negative total | `1d4 - 5` | 1 | 1 | -4 | p. 16 | unhappy (boundary) |
| zero total | `1d4 - 3` | 3 | 3 | 0 | p. 16 | unhappy (boundary) |


Case table for FR-041, FR-042, FR-043:

| Label | Expression | Faces | Call | Result | Source | Path |
|---|---|---|---|---|---|---|
| keep the higher of two d20s | `2d20` | 18, 3 | `SumOfHighest(1)` | 18 | p. 8 | happy |
| keep the lower of two d20s | `2d20` | 18, 3 | `SumOfLowest(1)` | 3 | p. 8 | happy |
| highest three of four d6 | `4d6` | 6, 5, 4, 1 | `SumOfHighest(3)` | 15 | p. 21 | happy |
| order does not matter | `4d6` | 1, 4, 6, 5 | `SumOfHighest(3)` | 15 | p. 21 | happy |
| all dice equal | `4d6` | 1, 1, 1, 1 | `SumOfHighest(3)` | 3 | p. 21 | unhappy (boundary) |
| ties at the cut | `4d6` | 3, 3, 3, 2 | `SumOfHighest(3)` | 9 | p. 21 | unhappy (boundary) |
| two lowest | `4d6` | 6, 1, 6, 1 | `SumOfLowest(2)` | 2 | n/a (package design; p. 6 and p. 8 only cover the lower of two d20s) | happy |
| modifier ignored | `2d20 + 5` | 18, 3 | `SumOfHighest(1)` | 18 | p. 8 | unhappy (rule interaction) |
| n equals count | `4d6` | 6, 5, 4, 1 | `SumOfHighest(4)` | 16 (`DiceTotal`) | p. 21 | unhappy (boundary) |
| n is zero | `4d6` | 6, 5, 4, 1 | `SumOfHighest(0)` | error (`ReasonSelectionOutOfRange`) | n/a | unhappy (boundary) |
| n above count | `4d6` | 6, 5, 4, 1 | `SumOfLowest(5)` | error (`ReasonSelectionOutOfRange`) | n/a | unhappy (boundary) |
| n negative | `4d6` | 6, 5, 4, 1 | `SumOfLowest(-1)` | error (`ReasonSelectionOutOfRange`) | n/a | unhappy (invalid input) |

### 4.4a Construction rules and doubling (added ids)

There is no separate non-throwing factory family: the plain constructors
`NewDie`, `NewExpression`, `NewDieRoll` and `NewResult` are the non-panicking
path for untrusted values and return `(T, error)`. Each carries exactly one typed
`Reason` (`errors-and-results.md` FR-035) and MUST NOT panic for any input
(`errors-and-results.md` FR-025, FR-026). `MustX` panics if and only if `NewX`
returns an error, with that error as the panic value. Reasons are checked in
argument order.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-069 | `Expression.CanDouble` MUST be `true` exactly when `count * 2 <= MaxCount` and the expression is not zero (500 true, 501 false), and `Double()` MUST fail only when it is `false`. | n/a (package design; lets a caller handle content near the limit without knowing the limit) | unhappy (boundary) |
| FR-070 | `NewDie(sides)` MUST succeed for 2 to `MaxSides` and fail with `ReasonSidesOutOfRange` otherwise (1, 0, negative, 1001, `int` extremes). | n/a (`architecture.md` §6) | unhappy (boundary) |
| FR-071 | `NewExpression(count, die, modifier)` MUST succeed for every argument set FR-006 to FR-009 accept, and otherwise fail with `ReasonCountOutOfRange`, `ReasonSidesOutOfRange` (zero `Die`) or `ReasonModifierOutOfRange`, count checked first, then the die, then the modifier. It never returns `ReasonEmpty` or `ReasonMalformed`. | n/a (`architecture.md` §6) | unhappy (boundary) |
| FR-072 | `NewDieRoll(die, value)` MUST succeed for 1 to `die.Sides()` and fail with `ReasonValueOutOfRange` otherwise (including every value for the zero `Die`). | n/a (`architecture.md` §6; physical dice entered by a host) | unhappy (boundary) |
| FR-073 | `NewResult(expression, rolls)` MUST succeed for every input FR-033 accepts, fail with `ReasonRollCountMismatch` for a wrong number of rolls, and fail with `ReasonDieMismatch` for a roll of another die (count checked first). It MUST copy the slice (FR-040) and not mutate it. | n/a (`architecture.md` §6) | unhappy (invalid input) |
| FR-074 | Property: for a table of argument sets (valid, each boundary, each invalid kind), `MustX` (where it exists, section 4) panics if and only if `NewX` returns an error, and the panic value is that error (`errors.As` works on the recovered value). | n/a (`errors-and-results.md` FR-026) | unhappy (rule conflict) |
| FR-075 | The rejection error `*InvalidError` MUST carry only the typed `Reason` for branching (the `Detail` string is for humans and never branched on) and MUST match `srderr.ErrInvalidInput` with `errors.Is`. `Reason` is a closed typed enum with `String()`; adding a value is a specification change. | n/a (`errors-and-results.md` FR-018, FR-020, FR-027) | unhappy (invalid input) |
| FR-076 | Package `dice` MUST export every name in the FR-067 list and nothing else from this spec; there is no facade or re-export package. FR-067 is the single list. | n/a (`architecture.md` §7) | happy |
| FR-077 | Every rejection returned by a constructor, `Double`, `SumOfHighest`, `SumOfLowest`, `dicetest.NewFixedRoller` or `dice.NewSeededRoller` argument check in this spec MUST be an `*InvalidError` (or, for the doubles, the documented `dicetest` sentinel, FR-056, FR-057) so a test asserts `errors.As`/`errors.Is` on the reason, never the message. | n/a (`errors-and-results.md` FR-027) | unhappy (invalid input) |
| FR-085 | No generic result or variant name (`Created`, `Rejected`, `Parsed`, `Applied`, `Succeeded`) is exported from `dice`. This package has no closed variant set: failures are errors and the rejection type is `InvalidError`. Any shared failure detail would be a value object held as a field, not a shared variant. | `shared-types-and-ownership.md` FR-017, FR-018 | unhappy (name collision) |
| FR-086 | The port name is the exported constant `dice.Port = "dice_roller"`. `srderr.ErrorType` of a roller failure MUST equal `"dice_roller_error"` (`<Port>_error`), the value of `error.type` wherever a use case records this failure. `errors-and-results.md` holds the registry of values and lists this one. | `shared-types-and-ownership.md` FR-041 | unhappy (failure) |
| FR-087 | The dice types and the roller emit no log, span or metric, and no type in this spec takes a telemetry argument. The d20 test's signals are owned by `resolve-d20-test-use-case.md`; this spec only supplies the `error.type` value (FR-086). | `shared-types-and-ownership.md` FR-040, FR-043 | happy |

### 4.5 Roller port and its failure contract

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-045 | `dice.Roller` MUST be an interface in package `dice` with the single synchronous method `Roll(Expression) (Result, error)`. It takes no `context.Context` (in-memory, `shared-types-and-ownership.md` FR-071). Expected absence does not exist for rolling, so there is no absence result; a failure is the `error`. | n/a (`architecture.md` §4: computational ports are synchronous; rolling has no expected-absence outcome) | happy |
| FR-046 | Every implementation MUST return a `Result` whose `Expression()` equals the requested expression (same count, die and modifier). | n/a (port contract) | happy |
| FR-047 | Every implementation MUST roll exactly `expression.Count()` dice, each within 1 to `sides`, so `Total` lies between `expression.Minimum()` and `expression.Maximum()`. | n/a (package design; follows from FR-031 and FR-033; SRD 5.2.1 p. 16 says only "roll the damage dice") | happy |
| FR-048 | An adapter whose backend can fail MUST return that failure as an `error` and the zero `Result`; it MUST NOT return a partial or default result. The caller goes through `dice.Roll`, which wraps the error with `srderr.WrapPort(dice.Port, "roll", err)` (`errors-and-results.md` FR-010, FR-031), so the failure surfaces as a `*srderr.PortError` that still matches the original cause with `errors.Is`. | n/a (`architecture.md` §4, failure contract; `errors-and-results.md` FR-009, FR-032) | unhappy (failure) |
| FR-049 | A roller failure MUST be a `*srderr.PortError` with `Port == dice.Port` and `Op == "roll"`; there is no dice-specific error type. A panic in the roller is recovered by `dice.Roll` into a `*srderr.PortError` wrapping `*srderr.PanicError` and matching `srderr.ErrPortPanic` (`errors-and-results.md` FR-009). It carries data, not display text. | n/a (`architecture.md` §6; `errors-and-results.md` FR-002, FR-005) | unhappy (failure) |
| FR-050 | `dice.Roll(r Roller, e Expression) (Result, error)` is the checked call: it MUST return the roller's result when it returns no error and `result.Expression() == e`. | n/a (port contract, FR-046) | happy |
| FR-051 | `dice.Roll` MUST return a `*srderr.PortError` matching `srderr.ErrContractViolation` (`srderr.ContractViolation`, `errors-and-results.md` FR-036) and the zero `Result` when the roller returns a result for a different expression (including the zero `Result`). One case per differing field: count, die, modifier. | n/a (testing.md §4, "collaborator returns invalid data") | unhappy (failure) |
| FR-052 | `dice.Roll` MUST NOT double-wrap: a `*srderr.PortError` for `dice.Port` returned by the roller is returned unchanged (the same instance; `errors-and-results.md` FR-010). | `observability.md` §7 (recorded once, by the use case) | unhappy (failure) |
| FR-053 | `dice.Roll` MUST NOT let a roller panic escape (a runtime error, a string); it is a port failure under FR-049. A nil roller is not a panic: it is a `*srderr.PortError` wrapping `srderr.ErrInvalidInput`. A panic raised by this module's own `MustX` outside the roller call propagates (`errors-and-results.md` FR-011). | `architecture.md` §6; `errors-and-results.md` FR-009, OQ-12 | unhappy (state error) |

Application code that needs a roll calls `dice.Roll`. Reroll-style rules (a
second roll of a single die, "Heroic Inspiration", p. 8) are further calls to
the same port, one die at a time; the reroll and replace-a-die operations live
on `d20.Roll` (`shared-types-and-ownership.md` FR-002, FR-010).

### 4.6 Test doubles (`dice/dicetest`)

The doubles live in package `dice/dicetest`, imported only by `_test.go` files
(`shared-types-and-ownership.md` FR-038); they depend only on the port and the
kernel. The package also exports the contract suite (FR-045 to FR-048; section 6).

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-054 | `dicetest.NewFixedRoller(values ...int)` MUST build each `dice.Result` from the next unused values in order, across dice and across calls (`18, 3` serves one `2d20`, or two `1d20` calls). The supply is finite: a call needing more dice than remain (FR-056) consumes nothing and is not recorded in `Requests()`; likewise a call that fails FR-057. | n/a (package design; testing.md §4, determinism) | happy |
| FR-055 | `dicetest.NewMinimumRoller()` MUST roll 1 on every die and `dicetest.NewMaximumRoller()` MUST roll each die's highest face, for any expression. | n/a (testing.md §4, "results for fixed rolls, including extremes") | unhappy (boundary) |
| FR-056 | `FixedRoller` MUST return the error `dicetest.ErrExhausted` (and the zero `Result`) when asked for more dice than it has values (a test set-up bug, reported as an ordinary error so a test asserts it with `errors.Is`; it never panics). | n/a (`architecture.md` §6, programmer error in a test script) | unhappy (state error) |
| FR-057 | `FixedRoller` MUST return `dicetest.ErrValueOutOfRange` when the next value is outside 1 to the requested die's sides (for example 21 for a d20). | n/a (invalid input) | unhappy (invalid input) |
| FR-058 | `FixedRoller.Requests()` MUST return a copy listing the requested expressions in call order, and `Remaining()` MUST report the number of unused values. | n/a (testing.md §6, specific assertions) | happy |
| FR-082 | `NewFixedRoller` MUST copy the `values` it is given; mutating the original slice after construction MUST NOT change later rolls. `NewFixedRoller()` with no values is valid: it serves nothing, so any `Roll` returns `ErrExhausted` (FR-056) and `Remaining()` is 0. The constructor does not validate the values; `Roll` validates each value it consumes (FR-057). | n/a (package design; immutability) | unhappy (immutability) |
| FR-083 | For the minimum and maximum rollers, supply is unbounded: `Remaining()` MUST return `(0, false)` (the second result `bounded` is false), they never return `ErrExhausted`, and `Requests()` MUST still list every expression in call order. | n/a (package design) | unhappy (boundary) |
| FR-059 | `dicetest.CallbackRoller` (a `func(dice.Expression) (dice.Result, error)`) MUST return whatever the function returns and let anything it panics with propagate to the caller of `Roll`, so tests can simulate failures, panics and wrong results through `dice.Roll`. | n/a (testing.md §4, "collaborator failure") | unhappy (failure) |

### 4.7 Bundled adapter (`dice.SeededRoller`)

`SeededRoller` lives in package `dice` and imports only `srderr` and the standard
library (`shared-types-and-ownership.md` FR-037). Its constructor
`NewSeededRoller(rng *rand.Rand)` takes a caller-supplied `math/rand/v2`
generator and nothing else; the package never creates one, so a consumer or test
passes `rand.New(rand.NewPCG(7, 7))` itself, and no unseeded randomness enters
the module. `var _ Roller = (*SeededRoller)(nil)` asserts the port. It is NOT
safe for concurrent use: a `*rand.Rand` is not; give each goroutine its own
roller or guard calls with a mutex.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-060 | `SeededRoller.Roll` MUST return a result satisfying FR-046 and FR-047 for every valid expression, including `1000d1000`. | n/a (port contract) | happy |
| FR-084 | `SeededRoller.Roll` MUST draw from the generator inside `srderr.GuardValue` so a generator that panics yields a `*srderr.PortError` for `dice.Port` with the panic recovered and its cause reachable (FR-048, FR-049). A nil receiver or nil generator is not a panic: it is a `*srderr.PortError` for `dice.Port` wrapping `srderr.ErrInvalidInput`. A zero expression fails with `srderr.ErrInvalidInput` as the cause. | n/a (`errors-and-results.md` FR-009, FR-032) | unhappy (failure of collaborator) |
| FR-061 | `SeededRoller` MUST call `rng.IntN(sides)` exactly once per die, in order, and map the returned `v` to the face `v + 1`. A generator returning 0 yields face 1; one returning `sides - 1` yields the highest face. | n/a (package design) | unhappy (boundary) |
| FR-062 | Two `SeededRoller`s over equal seeded generators (the test builds both) MUST give identical results for the same sequence of expressions. Tests MUST NOT pin concrete dice values, only equality between runs. | n/a (testing.md §4, determinism) | happy |
| FR-063 | A `SeededRoller` over a seeded generator MUST give the same faces for one `2d6` call as for two successive `1d6` calls on an equal generator (the sequence advances per die, in order). | n/a (package design) | happy |
| FR-064 | `SeededRoller` MUST use only the generator it was given. The file `dice/seeded_roller.go` MUST contain no call that creates a generator (`rand.New`, `rand.NewPCG`, `rand.NewChaCha8`) and no use of the package-level `rand` functions, so an unseeded roller is always the consumer's explicit choice (the opt-in unseeded adapter is the separate package `dice/randroller`, FR-088). Verified by the contract test only; no archtest scan exists. | n/a (`architecture.md` §3, no hidden non-determinism) | unhappy (determinism) |

### 4.8 Boundaries and entry points

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-065 | The import-direction checks of `architecture-guardrails.md` apply to the dice files and are not restated here: the kernel package `dice` imports only `srderr` and the standard library, `dice/dicetest` imports only `dice`, `srderr` and the standard library, and nothing but `_test.go` files imports `dice/dicetest` (the `internal/archtest` allow-list). This spec adds only: "imports" in those checks means module imports; `math/rand/v2` in the adapter file is a standard-library import. | n/a (`architecture-guardrails.md`; `architecture.md` §1, §9) | unhappy (state error) |
| FR-066 | The files of `dice` and `dice/dicetest` contain no `time.Now(`, `rand.New(` or package-level `math/rand` function call outside tests. (`dice` creates no generator and reads no clock.) Enforced today by archtest A3 (`math/rand` deny-list, `math/rand/v2` only in `dice`) and A4 (no `time.Now` and similar); a dedicated `rand.New` scan is not implemented, so FR-064 rests on the contract test. | n/a (`architecture.md` §3, §9) | unhappy (determinism) |
| FR-067 | Package `dice` MUST export exactly this list (the single export list for this spec): the types `Die`, `Expression`, `DieRoll`, `Result`, `Reason`, `InvalidError`, `Roller`, `SeededRoller`; the constants `MinSides`, `MaxSides`, `MaxCount`, `MaxModifier`, `Port` and the `Reason...` values; the functions `NewDie`, `MustDie`, `D3`, `D4`, `D6`, `D8`, `D10`, `D12`, `D20`, `D100`, `NewExpression`, `MustExpression`, `ParseExpression`, `MustParseExpression`, `NewDieRoll`, `MustDieRoll`, `NewResult`, `NewSeededRoller`, `Roll`; and their methods. Package `dice/dicetest` MUST export `FixedRoller`, `NewFixedRoller`, `NewMinimumRoller`, `NewMaximumRoller`, `CallbackRoller`, `ErrExhausted`, `ErrValueOutOfRange`, `RunRollerContract` and `RunFailureContract`. Nothing else imports `dice/dicetest` except tests (`shared-types-and-ownership.md` FR-038). Each exported identifier MUST have a doc comment starting with its name. | n/a (`architecture.md` §7) | happy |
| FR-068 | The public entry points expose a working roller and result: importing only `dice` and `dice/dicetest` from `package scenarios_test`, a `FixedRoller` serving `18, 3` for `2d20` MUST yield a `dice.Result` through `dice.Roll`. Entry-point wiring smoke test; the selection arithmetic is proven by FR-041 and FR-042. | SRD 5.2.1, "Roll Two D20s", p. 8 | happy |

### 4.9 Withdrawn ids

| Id | Status |
|---|---|
| FR-078 | Withdrawn: never issued (reserved, MUST NOT be reused) |
| FR-079 | Withdrawn: never issued (reserved, MUST NOT be reused) |
| FR-080 | Withdrawn: never issued (reserved, MUST NOT be reused) |

### 4.10 Concurrency-safe adapter (`dice/randroller`)

`SeededRoller` is reproducible but not safe for concurrent use. A host that
shares one `Resolver` between goroutines (a server) needs a roller that is. The
sub-package `dice/randroller` provides it (`shared-types-and-ownership.md`
FR-037: a sub-package, because it is an opt-in adapter). It is deliberately the
unseeded counterpart of FR-064: a consumer opts in by importing it, and nothing
in the module imports it. Reproducible runs use `SeededRoller` (one per game or
request) or a `dicetest` double.

| Id | Requirement | Source | Path |
|---|---|---|---|
| FR-088 | Package `dice/randroller` MUST export `Roller` (an empty struct whose zero value is ready to use) and `New() Roller`, which returns the zero value. `var _ dice.Roller = Roller{}` asserts the port. It imports only `dice` and the standard library, and nothing else in the module imports it. | `shared-types-and-ownership.md` FR-037; `architecture.md` §1 | happy |
| FR-089 | `Roller.Roll` MUST return a result satisfying FR-046 and FR-047 for every valid expression, including `1000d1000`: each face is drawn uniformly from `1..Sides()`, so the total lies within `Minimum()..Maximum()` and the result carries the requested expression. | n/a (port contract) | happy |
| FR-090 | `Roller` MUST be safe for concurrent use: it holds no state and draws only through the goroutine-safe package-level functions of `math/rand/v2`, so one value may be shared by any number of goroutines (verified under `-race`). | n/a (`architecture.md` §4, Concurrency) | happy |
| FR-091 | `Roller.Roll` of the zero `Expression` MUST return the zero `Result` and the `*InvalidError` that `NewResult` reports, never a partial result and never a panic; `dice.Roll` wraps it as a `*srderr.PortError` matching `srderr.ErrInvalidInput`. | n/a (`errors-and-results.md` FR-009) | unhappy (invalid input) |
| FR-092 | `Roller` is silent: it emits no telemetry and takes no telemetry argument (extends FR-087). Rolls are not reproducible, so tests MUST assert ranges and equality of the requested expression only, never concrete dice values. | `observability.md` §3; n/a (testing.md §4, determinism) | happy |

FR-103 of `architecture-guardrails.md` bans package-level `math/rand` functions
in adapter packages; `dice/randroller` is its single named exception, because
drawing from the process-wide generator is the adapter's whole purpose.

## 5. Model

- **Value objects** (immutable, unexported fields, value receivers, compared by
  value): `Die`, `Expression`, `DieRoll`, `Result`, in package `dice`. `Die`,
  `Expression` and `DieRoll` are comparable; `Result` holds a slice and is
  compared with `Equal`. The zero value of each is explicitly invalid (`IsZero`).
  No entities, aggregates, events or services. Invariants are enforced in
  constructors (FR-002 to FR-009, FR-031 to FR-035, FR-039, FR-040).
- **Failures:** constructors and the `n`-argument methods return `(T, error)`
  with an `*InvalidError`; untrusted text goes through `ParseExpression`,
  untrusted numbers through `NewX` (FR-070 to FR-073), `CanDouble` (FR-069)
  guards `Double`; a port infrastructure failure is a `*srderr.PortError`; a
  `MustX` panics only for a programmer error.
- The domain cannot depend on the port implementation: a use case calls
  `dice.Roll` and passes the `dice.Result` to the domain as a value. A context
  that needs a roller declares its own one-method consumer-side interface
  (`Roll(dice.Expression) (dice.Result, error)`) and is satisfied by any
  `dice.Roller`.

## 6. Interface

- **Use cases:** none; other specs' use cases take a roller through their
  constructor or options (`architecture.md` §4).
- **Port `dice.Roller`:** `Roll(Expression) (Result, error)`, synchronous, no
  `ctx`. A roller backed by I/O is outside the contract: a consumer resolves the
  roll first, validates host-supplied faces with `NewDieRoll` and `NewResult`,
  and serves the result through its own adapter.
- **Failure contract:** no expected-absence outcome. A roller returns an `error`
  for infrastructure failure (FR-048); `dice.Roll` wraps it once as a
  `*srderr.PortError` (`Port` `dice.Port`), reports a wrong answer as
  `srderr.ErrContractViolation` (FR-051) and a panic as `srderr.ErrPortPanic`
  (FR-053). A `*InvalidError` from a constructor is never wrapped.
- **Contract suite:** `dicetest.RunRollerContract(t, newRoller)` asserts FR-045
  to FR-047 against any roller (run in this module against `FixedRoller` extremes,
  `SeededRoller` over a seeded generator built by the test, `randroller.Roller`, and a
  `CallbackRoller` wrapping `SeededRoller`).
  `dicetest.RunFailureContract(t, newFailing)` asserts FR-048 and FR-049: given a
  factory for a roller whose backend fails with a known cause (by returning it or
  panicking with it), `dice.Roll` returns the zero `Result` and a `*srderr.PortError`
  for `dice.Port` that matches the cause and whose `srderr.ErrorType` is
  `dice_roller_error`. Consumer adapters reuse both suites
  (`shared-types-and-ownership.md` FR-039).
- **File layout:** `dice/` (one file per concept: `die.go`, `die_roll.go`,
  `expression.go`, `scan.go`, `result.go`, `reason.go`, `roller.go`,
  `seeded_roller.go`, `doc.go`); `dice/dicetest/` (`fixed_roller.go`,
  `callback_roller.go`, `contract.go`, `doc.go`); `dice/randroller/` (`roller.go`, `doc.go`).

## 7. Observability

**Silent by design:** value objects and an in-memory port emit nothing (FR-087).

| Signal | Type | Attributes | Emitted by | Cardinality |
|---|---|---|---|---|
| none | n/a | n/a | n/a | n/a |

Signals owned by this spec: none. The shared `error.type` value is
FR-086; the use case that lets the failure leave srd5e records it once
(`observability.md` §7; `shared-types-and-ownership.md` FR-040, FR-041; for
example `resolve-d20-test-use-case.md`). The seven questions (`observability.md`
§2):

1. **What can go wrong?** A roller cannot roll (an error or a panic); an adapter
   returns a result for another expression (`ErrContractViolation`); untrusted
   dice text is malformed (`ReasonMalformed`); an invalid argument.
2. **How would an operator know?** The use case's span error status, error
   record and error counter with `error.type=dice_roller_error`. The port and
   adapter neither log nor count, so nothing is counted twice. A rejected parse
   is an error the consumer that parsed returns; it decides whether to count it.
3. **What is needed to diagnose it?** The `*srderr.PortError` carries the port,
   the operation and the cause (and the panic stack when the roller panicked).
   The expression (canonical `Notation`, FR-030) belongs in the caller's log, never
   in a metric or span attribute (unbounded cardinality).
4. **What should be counted or timed?** Roll throughput and latency belong to
   the rolling use case; a port-level counter would duplicate it.
5. **Is existing instrumentation still correct?** None exists yet (phase 0).
6. **What should a consumer alert on?** A non-zero rate of
   `error.type=dice_roller_error` in the use cases' error counters; the cause
   (failure, panic, contract violation) is told apart from the span's error
   record (no extra attribute).
7. **Is anything now noisy?** No; `SeededRoller` stays silent per roll (hot
   loop, `observability.md` §7).

## 8. Test plan

Doubles: `FixedRoller`, the minimum and maximum rollers and `CallbackRoller`
(section 4.6), tested by FR-054 to FR-059. Mutation testing (`testing.md` §8):
`gomu` on `./dice`, run only inside Docker with `--cpus` and `--workers` capped
at a quarter of the host's processors (command in `testing.md`), threshold 80;
FR-012 rounding and the limits (FR-002 to FR-009, FR-015) get one case on each
side of every limit. SHOULD for the adapter and doubles. All tests are
table-driven with `t.Run`; the parser has `FuzzParseExpression`; the files are
`dice/die_test.go`, `expression_test.go`, `result_test.go`, `roller_test.go`,
`example_test.go`, `dice/dicetest/dicetest_test.go`, `dice/dicetest/contract_internal_test.go` and
`scenarios/dice_test.go`.

| Ids | Level | Notes |
|---|---|---|
| FR-001 to FR-005 | unit | `die_test.go`; the named constructors' sides; boundary table 1, 2, 1000, 1001, 0, negative; `==` equality. |
| FR-006 to FR-016, FR-069 | unit | `expression_test.go`; limit tables; FR-010 to FR-012 table-driven, one test per row of the first case table (includes the negative-average rounding row); equality; `Double` limit at 501, `CanDouble` at 500 and 501, original unchanged. |
| FR-017 to FR-030, FR-081 | unit, fuzz | `expression_test.go`; table-driven over the accepted and rejected tables, asserting the exact `Reason` through `errors.As`; FR-081 rows plus `00d6`; FR-029 `FuzzParseExpression` and a fixed list of hostile strings (very long, control characters, invalid UTF-8) asserting an error and no panic; FR-030 canonical output table plus round trip over every accepted row. |
| FR-031 to FR-044 | unit | `die_test.go` (`NewDieRoll` boundary values); `result_test.go`: count and die mismatch; totals table (extremes via `Minimum` and `Maximum`); source slice and returned slice edited after construction; selection table including error rows, asserting `Rolls()` unchanged in order and content after each call; `Equal`. |
| FR-045 to FR-048 | contract | `dicetest.RunRollerContract` and `RunFailureContract` run against every implementation (section 6); FR-045 is a compile-time `var _ dice.Roller` assertion; FR-048 through a failing factory on `SeededRoller` over a panicking source, asserting the `*srderr.PortError`, the cause and `dice_roller_error`. |
| FR-049, FR-086 | unit | `roller_test.go`: `errors.As` a `*srderr.PortError` with `Port == dice.Port` and `Op == "roll"`, `errors.Is` the cause and `srderr.ErrPortPanic`, `srderr.ErrorType == "dice_roller_error"`. |
| FR-050 to FR-053 | unit (port helper, with double) | `TestRoll` in `roller_test.go`, using `dicetest.CallbackRoller` to return a wrong result (one case per differing field), return a `PortError` for `dice.Port` (same instance), and panic. |
| FR-054 to FR-059, FR-082, FR-083 | unit | `dicetest_test.go`; FR-058: `Requests` order and copy semantics, `Remaining` counts down after each roll; empty supply, defensive copy, atomic failure (nothing consumed or recorded), `bounded == false` for the minimum and maximum rollers; `CallbackRoller` for FR-059. |
| FR-070 to FR-077 | unit | `NewX` tables, one case each side of every limit; the iff property (FR-074) as a table over valid and invalid argument sets through `NewX` and `MustX`; `Reason` asserted on every error; FR-076 in the public-surface test. |
| FR-060, FR-084 | contract | `roller_test.go`; also a `1000d1000` call; FR-084 through the failing factory plus a nil generator (`ErrInvalidInput`) and a panicking source. |
| FR-061 to FR-063 | contract | A scripted `rand.Source` recording `IntN` bounds (bound, count, order, `+ 1` mapping at both extremes); FR-062/FR-063 compare two rollers over equal seeded generators built by the test, never pinning concrete values. |
| FR-064 | contract | A scripted source records every draw, so only the injected generator is used; no archtest scan exists. |
| FR-088 to FR-092 | contract, unit | `dice/randroller/roller_test.go`: `dicetest.RunRollerContract`; a bounds table (one row per roller value and expression, including `100d100`); 32 goroutines sharing one `Roller` under `-race`; the zero expression through `dice.Roll` asserted with `errors.As`/`errors.Is`; an `ExampleRoller` that prints only a range check. No test pins a concrete roll. |
| FR-065, FR-066 | architecture | `internal/archtest` allow-list (A1) covers import direction; A3 and A4 cover the `math/rand` and `time.Now` patterns. No `rand.New` scan exists (FR-066 note). |
| FR-067, FR-085, FR-087 | architecture (FR-087 also unit) | Public-surface test from `package scenarios_test` references each exported name; the doc-comment check covers every exported identifier; scan for generic variant names (`shared-types-and-ownership.md` FR-017); FR-087: no dice type takes a telemetry argument, and a scan of `dice/` finds no logger, tracer or meter use. Every FR appears above; none is left uncovered. `AGENTS.md` updated in the same change. |
| FR-068 | scenario | `scenarios/dice_test.go`, public API only; a wiring smoke test, not repeating FR-041/FR-042. |

Deliberately not covered: FR-065 - a pointer row; the import-direction checks it names are owned by `architecture-guardrails.md` and `internal/archtest`, so the dice files get them from that allow-list.
Deliberately not covered: FR-078 - withdrawn, never issued; the id is reserved and has no behaviour to test.
Deliberately not covered: FR-079 - withdrawn, never issued; the id is reserved and has no behaviour to test.
Deliberately not covered: FR-080 - withdrawn, never issued; the id is reserved and has no behaviour to test.

## 9. Open questions and decisions

Settled (not repeated): name `Expression`, `Modifier()` as `int`, `SeededRoller`
and the adapter and double placement (`shared-types-and-ownership.md` FR-001,
FR-037, FR-038); one rejection reason (`errors-and-results.md` FR-035);
`error.type` registration (FR-086); other choices are FR-002, FR-030, FR-064,
FR-069.

| Question | Recommendation | Status |
|---|---|---|
| The SRD has no "Rolling Dice" section. Is resting on the usages in section 2 acceptable? | Yes: the only reading compliant with "if it is not in that PDF, it does not go in". Anything else (keep-highest, `d%`, multiple terms) stays out until a rule needs it. | decided |
| Limits `maxSides` 1000, `maxCount` 1000, `maxModifier` 1000000. | Keep: safe integer arithmetic on web targets and a bound on roller work, not an SRD rule. Raising later is non-breaking, lowering breaking. | decided |
| Accept the en dash U+2013 as a minus (the SRD prints it once, `1d4 – 1`, p. 356, against U+2212 everywhere else)? (FR-028) | Reject as `malformed`: one printed instance is a typographic inconsistency, not a notation; a parser that accepted every dash look-alike would hide bad content. Revisit if content needs it (a FR-021 extension is non-breaking). | decided |
| `average` rounds negative values by floor. The SRD's "Round Down" (p. 5) gives no negative example. | Floor towards negative infinity, pinned by the `1d4 - 5` row; revisit only if a rule needs truncation. Package-design decision (interpretation: "Round Down" says to round down a fraction, which for a negative value is towards negative infinity). | decided |

## 10. Changelog impact

Under `Unreleased` / **Added**: package `dice` with the value objects (`Die`,
`Expression`, `DieRoll`, `Result`), `ParseExpression`, the `Roller` port with
`Roll` (port name `dice_roller`, error type `dice_roller_error`),
`SeededRoller`; and package `dice/dicetest` with `FixedRoller`, `CallbackRoller`
and the contract suites. Not **BREAKING**: first code in the module, no earlier
release exists.
