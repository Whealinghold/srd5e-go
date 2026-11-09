package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// ClassSpellcastingSummary is what the tables and the ability scores give one
// spellcasting class: its stats, cantrip and prepared limits and the highest
// spell level it may prepare. It is comparable.
type ClassSpellcastingSummary struct {
	class         SpellcastingClass
	level         int
	stats         SpellcastingStats
	cantripLimit  int
	preparedLimit int
	maxPreparable int
}

// Class returns the class.
func (s ClassSpellcastingSummary) Class() SpellcastingClass { return s.class }

// Level returns the class level.
func (s ClassSpellcastingSummary) Level() int { return s.level }

// Stats returns the spell save DC, attack bonus and the terms behind them.
func (s ClassSpellcastingSummary) Stats() SpellcastingStats { return s.stats }

// CantripLimit returns the cantrips known from the feature (Table F).
func (s ClassSpellcastingSummary) CantripLimit() int { return s.cantripLimit }

// PreparedLimit returns the prepared level 1+ spells (Table G).
func (s ClassSpellcastingSummary) PreparedLimit() int { return s.preparedLimit }

// MaxPreparableLevel returns the highest spell level the class may prepare.
func (s ClassSpellcastingSummary) MaxPreparableLevel() int { return s.maxPreparable }

// SpellcastingSummary is the value of CalculateSpellcastingCalculated: a summary
// per spellcasting class, in class-level order, and the slots of both pools.
type SpellcastingSummary struct {
	classes []ClassSpellcastingSummary
	pools   SpellSlotPools
}

// Classes returns a new slice of the per-class summaries.
func (s SpellcastingSummary) Classes() []ClassSpellcastingSummary { return slices.Clone(s.classes) }

// Pools returns the slots of both pools, at full strength.
func (s SpellcastingSummary) Pools() SpellSlotPools { return s.pools }

// summarizeSpellcasting is the domain computation behind CalculateSpellcasting.
func summarizeSpellcasting(l ClassLevels, scores ability.Scores, pb ProficiencyBonus) SpellcastingSummary {
	out := SpellcastingSummary{pools: SlotsFor(l)}
	for _, e := range l.Entries() {
		c, ok := SpellcastingClassFor(e.Class)
		if !ok {
			continue
		}
		out.classes = append(out.classes, ClassSpellcastingSummary{
			class: c, level: e.Level, stats: Stats(c, scores.Modifier(c.Ability()), pb),
			cantripLimit: CantripLimit(c, e.Level), preparedLimit: PreparedLimit(c, e.Level), maxPreparable: MaxPreparableLevel(c, e.Level),
		})
	}
	return out
}

// CalculateSpellcastingCommand is the input of CalculateSpellcasting.Do. Construct
// it with a keyed literal. Do does not alter it.
type CalculateSpellcastingCommand struct {
	Levels      ClassLevels
	Scores      ability.Scores
	Proficiency ProficiencyBonus
}

// CalculateSpellcastingResult is the closed set of outcomes of
// CalculateSpellcasting.Do. Variants: CalculateSpellcastingCalculated.
type CalculateSpellcastingResult interface{ isCalculateSpellcastingResult() }

// CalculateSpellcastingCalculated is the outcome of a calculation.
type CalculateSpellcastingCalculated struct{ summary SpellcastingSummary }

// Summary returns the summary.
func (r CalculateSpellcastingCalculated) Summary() SpellcastingSummary { return r.summary }

func (CalculateSpellcastingCalculated) isCalculateSpellcastingResult() {}

// CalculateSpellcasting is the use case that derives spellcasting stats, limits
// and slots from class levels and ability scores. It holds no rules logic and
// needs no port beyond telemetry. Create it with NewCalculateSpellcasting; the
// zero value is not usable. It is safe for concurrent use only if its telemetry
// and clock are.
type CalculateSpellcasting struct {
	sig spellcastingSignals
	set bool
}

// NewCalculateSpellcasting returns the use case. Options: WithTelemetry,
// WithClock; the others are ignored.
func NewCalculateSpellcasting(opts ...Option) CalculateSpellcasting {
	return CalculateSpellcasting{sig: newSpellcastingSignals(newOptions(opts)), set: true}
}

// Do calculates the summary. It returns an error matching srderr.ErrInvalidInput,
// before any signal, for the zero use case or a zero command member, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. A valid
// command always succeeds; it calls no port.
func (u CalculateSpellcasting) Do(ctx context.Context, cmd CalculateSpellcastingCommand) (CalculateSpellcastingResult, error) {
	if !u.set || cmd.Levels.IsZero() || cmd.Scores.IsZero() || cmd.Proficiency.IsZero() {
		return nil, invalidInput("calculate spellcasting: zero use case, class levels, scores or Proficiency Bonus")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.begin(ctx, SpanCalculateSpellcasting, ValueOperationCalculateSpellcasting)
	defer run.end(ctx)
	res := CalculateSpellcastingCalculated{summary: summarizeSpellcasting(cmd.Levels, cmd.Scores, cmd.Proficiency)}
	run.finish(ctx, ValueResultCalculated, "")
	return res, nil
}
