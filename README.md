# srd5e

[![Go Reference](https://pkg.go.dev/badge/github.com/Whealinghold/srd5e-go.svg)](https://pkg.go.dev/github.com/Whealinghold/srd5e-go)

A framework-agnostic Go implementation of the fifth-edition rules in System Reference Document 5.2.1, written to be imported by other projects.

Status: early. Dice, modifiers, ability scores, the `dice.Roller` port, the error model, telemetry ports and the first context (`d20`: ability checks, saving throws and attack rolls) exist.

## Goals

- Idiomatic, standard-library-only Go, importable by any project: servers, CLIs, games.
- No dependency on a UI, persistence, framework or specific consuming project.
- Small interfaces defined where they are consumed, so every module can be replaced independently.
- Rules engine (code) kept separate from rules content (data).

## Usage

```sh
go get github.com/Whealinghold/srd5e-go
```

Requires Go 1.24 or later.

There is no facade package: import the package that owns the type (`dice`, `ability`, `modifier`, `d20`, ...). Runnable examples appear with each package on [pkg.go.dev](https://pkg.go.dev/github.com/Whealinghold/srd5e-go).

```go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

func main() {
	ctx := context.Background()

	roller := dice.NewSeededRoller(rand.New(rand.NewPCG(1, 1))) // you own the generator
	resolver, err := d20.NewResolver(roller)
	if err != nil {
		log.Fatal(err) // a nil roller
	}

	res, err := resolver.Resolve(ctx, d20.SavingThrowCommand{
		Ability: ability.Dexterity,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2))},
	})
	if err != nil {
		log.Fatal(err) // invalid command or a roller failure
	}
	switch r := res.(type) { // a failed save is a value, not an error
	case d20.RolledCheck:
		fmt.Println(r.Check().Total(), r.Outcome())
	case d20.RolledAttack, d20.ForcedFailure:
	}
}
```

## Package map

| Package | Holds |
|---|---|
| `dice` | `Die`, `Expression`, `Result`, the `Roller` port, `SeededRoller` |
| `dice/dicetest` | `FixedRoller`, `CallbackRoller`, `RunRollerContract` |
| `dice/randroller` | Unseeded, concurrency-safe `dice.Roller` adapter (stdlib `math/rand/v2`) |
| `modifier` | `Modifier`, `RollMode`, Advantage and Disadvantage |
| `ability` | `Ability`, `Score`, `Scores` |
| `srderr` | `PortError`, `Guard`, shared sentinel errors |
| `telemetry` | `Logger`, `Tracer`, `Meter` ports and no-op defaults |
| `telemetry/telemetrytest` | recording doubles and contract suites |
| `telemetry/slogtelemetry` | `log/slog` adapter for `telemetry.Logger` |
| `d20` | ability checks, saving throws and attack rolls |
| `content` | typed SRD content, `JSONDecoder`, `Loader`, `Source` port, `InMemorySource` |
| `content/contenttest` | `FakeSource`, `FakeDecoder`, `RunSourceContract`, `RunDecoderContract` |
| `content/data` | `BundledSource` and the bundled SRD slice (covered by `NOTICE.md`) |
| `scenarios` | cross-package tests through the public API (tests-only directory; not an importable package) |

Not built yet: `character`, `combat` and `toolbox` (see [doc/roadmap.md](doc/roadmap.md)).

## Dice and rolling

All randomness enters through the `dice.Roller` port (one method, like `io.Reader`), so the rules stay pure and tests stay deterministic.

- **Values:** `dice.Die`, `dice.Expression` (`1d20`, `2d8 + 2`), `dice.DieRoll` and `dice.Result`. Read untrusted text with `dice.ParseExpression` and untrusted numbers with the `NewX` constructors; both return `(T, error)`. The `MustX` helpers panic and are for constants.
- **Rolling:** call `dice.Roll(roller, expr)`. It wraps a roller failure, panic or wrong answer in a `*srderr.PortError`; branch with `errors.Is` and `errors.As`.
- **Adapter:** `dice.NewSeededRoller(rng)` rolls from a `*rand.Rand` you supply. It is not safe for concurrent use.
- **Test doubles** (`dice/dicetest`): `NewFixedRoller(18, 3)` serves the faces you choose, `NewMinimumRoller` and `NewMaximumRoller` roll the extremes, and `CallbackRoller` simulates failures.

```go
rng := rand.New(rand.NewPCG(1, 1)) // you own the generator
roller := dice.NewSeededRoller(rng)

expr, err := dice.ParseExpression("2d8 + 2")
if err != nil { /* errors.Is(err, srderr.ErrInvalidInput) */ }

res, err := dice.Roll(roller, expr) // err is a *srderr.PortError if the roller fails
fmt.Println(res.Total(), res.Rolls())
```

### Testing your own `dice.Roller`

`dicetest.RunRollerContract(t, factory)` and `dicetest.RunFailureContract` check that an implementation honours the port. Every adapter, including yours, should pass them.

```go
func TestMyRoller(t *testing.T) {
	dicetest.RunRollerContract(t, func(t *testing.T) dice.Roller {
		return mypkg.NewRoller()
	})
}
```

## Game content

The `content` package holds typed SRD 5.2.1 content and the ports that deliver it: the strict `content.JSONDecoder` (reports every problem with its path), the `content.Loader` use case, `content.InMemorySource`, and `content/data`, whose `BundledSource` serves the bundled slice (4 species, the Wizard, 12 spells). Look entries up with `Source.Lookup` and switch over the closed `LookupResult`.

```go
src, err := data.NewBundledSource()
res, err := src.Lookup(ctx, content.NewKey(content.KindSpell, content.MustID("fireball")))
```

### Writing your own `content.Source`

`contenttest.RunSourceContract(t, create, createFailing)` checks that a source honours the port (lookups, ordered lists, and typed failures when the backend breaks); `contenttest.RunDecoderContract` does the same for a `content.Decoder`. Every adapter, including yours, should pass them. The bundled content is covered by [NOTICE.md](NOTICE.md); see "Licence and content" below for the attribution statement.

```go
func TestMySource(t *testing.T) {
	contenttest.RunSourceContract(t, func(t *testing.T, entries []content.Entry) content.Source {
		return mypkg.NewSource(entries)
	}, nil)
}
```

## Ability scores

`ability.Ability` names the six abilities. An `ability.Score` is a whole number from 1 to 30; its `Modifier` comes from the SRD's Ability Modifiers table (score 1 gives -5, 10 gives +0, 20 gives +5, 30 gives +10). `ability.NewScore` returns an `*ability.InvalidError` for a number outside 1 to 30. `ability.NewScores` holds one score per ability. An `ability.Modifier` is not a `modifier.Modifier`: convert it with `modifier.New(m.Value())`.

## Observability (logging, tracing, metrics)

srd5e emits logs, traces and metrics but never chooses a backend. The `telemetry` package defines three small ports (`Logger`, `Tracer`, `Meter`) bundled as `telemetry.Telemetry`. You inject your own implementations, for example OpenTelemetry for traces and metrics.

- **Optional.** The zero `telemetry.Telemetry` is a silent no-op.
- **No vendor dependencies.** Your adapters live in your project; `telemetry/slogtelemetry` adapts `log/slog` and uses the standard library only.
- **Injected once,** with `d20.WithTelemetry` on the constructor. There are no globals.
- Only use cases emit telemetry. The domain stays pure and silent.
- Tests can use the recording doubles and contract suites in `telemetry/telemetrytest`.

The signal catalogue is [doc/observability/signals.md](doc/observability/signals.md); the rules are in [doc/rules/observability.md](doc/rules/observability.md).

## Errors and results

srd5e keeps four kinds of "it did not go to plan" apart (see [doc/specs/errors-and-results.md](doc/specs/errors-and-results.md)):

- **Expected outcomes** (a failed check, an illegal action) are returned values, switched over with a type switch or compared with an enum. They are never errors.
- **Untrusted input that fails validation** is a typed error that matches `srderr.ErrInvalidInput` and, for finer branching, the package's own `*InvalidError`.
- **Failures outside the module** (a roller or telemetry backend) are a `*srderr.PortError`. Read the cause with `errors.Is` and `errors.As`, and the telemetry label with `srderr.ErrorType`.
- **Programmer errors** panic, only inside `MustX` helpers.

## Development

```sh
just check   # gofmt, go vet, golangci-lint, tests with -race -shuffle=on -cover, per-package coverage floor, fuzz smoke, go mod tidy, architecture tests
```

CI pins golangci-lint at v2.15.0 (`.github/workflows/ci.yml`); use the same version locally.

`internal/archtest` enforces the dependency allow-list of [doc/rules/architecture.md](doc/rules/architecture.md) as part of `go test ./...`.

## Licence and content

- **Code:** MIT. See [LICENSE](LICENSE).
- **Game content:** rules from the System Reference Document 5.2.1, used under
  the Creative Commons Attribution 4.0 International License. The bundled
  content lives under `content/data/` and is covered by [NOTICE.md](NOTICE.md),
  not by the MIT licence. Nothing outside the SRD is included, and Wizards'
  trademarks are not used.

  The attribution the SRD asks for:

  > This work includes material from the System Reference Document
  > 5.2.1 (“SRD 5.2.1”) by Wizards of the Coast LLC, available at
  > https://www.dndbeyond.com/srd. The SRD 5.2.1 is licensed under the
  > Creative Commons Attribution 4.0 International License, available at
  > https://creativecommons.org/licenses/by/4.0/legalcode.

This package is compatible with fifth edition. It is independent and is not
published, endorsed or approved by Wizards of the Coast.
