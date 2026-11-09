package character

import (
	"context"
	"errors"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// SpendHitPointDieCommandProblem is a reason NewSpendHitPointDieCommand refused
// the host's spent counts. The zero value is not a problem. It is an error
// matching srderr.ErrInvalidInput.
//
//nolint:errname // an enum of problems, named by the spec
type SpendHitPointDieCommandProblem int

// SpentCountBelowOne: a die is listed with a spent count below 1. A count above
// the pool's is not a problem here but the returned rejection SpentExceedsPool.
const SpentCountBelowOne SpendHitPointDieCommandProblem = iota + 1 //nolint:errname // a member of the problem enum

// String returns "spent_count_below_one" or "unknown".
func (p SpendHitPointDieCommandProblem) String() string {
	if p == SpentCountBelowOne {
		return "spent_count_below_one"
	}
	return "unknown"
}

// Error renders the problem. The text is not a contract.
func (p SpendHitPointDieCommandProblem) Error() string {
	return "character: invalid spend Hit Point Die command: " + p.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (SpendHitPointDieCommandProblem) Unwrap() error { return srderr.ErrInvalidInput }

// SpendHitPointDieCommand is the input of HitDieSpender.SpendHitPointDie: spend
// one die of the chosen size, claiming the benefits of a RestKind (a finished
// Short Rest, or a Long Rest that earned them, both claim ShortRest). Build it
// with NewSpendHitPointDieCommand; the zero command is invalid.
// SpendHitPointDie does not alter it.
type SpendHitPointDieCommand struct {
	id           Identifier
	pool         HitDicePool
	spent        SpentHitDice
	die          content.HitPointDie
	constitution ability.Modifier
	kind         RestKind
}

// NewSpendHitPointDieCommand returns the command. spent holds the host's stored
// spent counts.
//
// It returns an error matching srderr.ErrInvalidInput for a zero id, an invalid
// die or kind, or an invalid die among the spent counts, and otherwise an
// errors.Join of SpentCountBelowOne when a spent count is below 1. It never
// panics.
func NewSpendHitPointDieCommand(id Identifier, pool HitDicePool, spent map[content.HitPointDie]int, die content.HitPointDie, constitution ability.Modifier, kind RestKind) (SpendHitPointDieCommand, error) {
	if id.IsZero() || !die.Valid() || !kind.Valid() {
		return SpendHitPointDieCommand{}, invalidInput("spend Hit Point Die command with a zero id, die or kind")
	}
	s, err := NewSpentHitDice(spent)
	if errors.Is(err, CountBelowOne) {
		return SpendHitPointDieCommand{}, errors.Join(SpentCountBelowOne)
	}
	if err != nil {
		return SpendHitPointDieCommand{}, err
	}
	return SpendHitPointDieCommand{id: id, pool: pool, spent: s, die: die, constitution: constitution, kind: kind}, nil
}

// SpendHitPointDieResult is the closed set of outcomes of
// HitDieSpender.SpendHitPointDie. Variants: SpendHitPointDieApplied,
// SpendHitPointDieRejected.
type SpendHitPointDieResult interface{ isSpendHitPointDieResult() }

// SpendHitPointDieApplied: a die was spent and rolled. The Hit Points regained are
// not capped by the Hit Point maximum; the host that applies them does that.
type SpendHitPointDieApplied struct {
	roll, hitPointsRegained int
	constitution            ability.Modifier
	spent                   SpentHitDice
	event                   HitPointDieSpent
}

// Roll returns the face rolled.
func (r SpendHitPointDieApplied) Roll() int { return r.roll }

// ConstitutionModifier returns the Constitution modifier added to the roll.
func (r SpendHitPointDieApplied) ConstitutionModifier() ability.Modifier { return r.constitution }

// HitPointsRegained returns the Hit Points regained, at least 1.
func (r SpendHitPointDieApplied) HitPointsRegained() int { return r.hitPointsRegained }

// SpentHitDice returns the spent dice after the spend.
func (r SpendHitPointDieApplied) SpentHitDice() SpentHitDice { return r.spent }

// HitPointDieSpent returns the HitPointDieSpent event.
func (r SpendHitPointDieApplied) HitPointDieSpent() HitPointDieSpent { return r.event }

// SpendHitPointDieRejected: no die was spent and the roller was not called.
type SpendHitPointDieRejected struct{ reason HitDieSpendRejection }

// Reason returns why the spend was refused.
func (r SpendHitPointDieRejected) Reason() HitDieSpendRejection { return r.reason }

func (SpendHitPointDieApplied) isSpendHitPointDieResult()  {}
func (SpendHitPointDieRejected) isSpendHitPointDieResult() {}

// HitDieSpender is the use case that spends one Hit Point Die: it checks the
// spend, rolls the die through the dice port once, and emits the
// srd5e.character.spend_hit_point_die span and the hit_point_dice_spent counter.
// It holds no rules logic and reads no current Hit Points, clock or Character.
// Create it with NewHitDieSpender; the zero value is not usable. It is safe for
// concurrent use only if its roller, telemetry and clock are.
type HitDieSpender struct {
	roller Roller
	tel    telemetry.Telemetry
	clock  func() time.Time
	spends telemetry.Counter
	hp     telemetry.Histogram
	dur    telemetry.Histogram
}

// NewHitDieSpender returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewHitDieSpender(roller Roller, opts ...Option) (HitDieSpender, error) {
	if roller == nil {
		return HitDieSpender{}, invalidInput("spend Hit Point Die without a dice roller")
	}
	o := newOptions(opts)
	m := o.tel.Meter
	return HitDieSpender{
		roller: roller,
		tel:    o.tel,
		clock:  o.clock,
		spends: m.Counter(telemetry.Descriptor{Name: MetricHitPointDiceSpent, Unit: "{spend}", Description: "Hit Point Die spends, by die size and result."}),
		hp:     m.Histogram(telemetry.Descriptor{Name: MetricHitDieHitPoints, Unit: "{hit_point}", Description: "Hit Points regained per spent Hit Point Die."}),
		dur:    m.Histogram(telemetry.Descriptor{Name: MetricRestOperationDuration, Unit: "s", Description: "Duration of one rest use case call."}),
	}, nil
}

// SpendHitPointDie checks the spend with SpentHitDice.Spend before it rolls, and
// on a rejection does not call the roller. On an allowed spend it rolls one die
// of the chosen size exactly once and returns the roll, the Hit Points regained
// and the new spent dice.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the
// zero use case or a command not built by NewSpendHitPointDieCommand, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It
// returns a *srderr.PortError (error type "dice_roller_error") with a nil result
// when the roller fails, panics or breaks its contract, and the state is
// unchanged; the failure is recorded once and never logged.
func (u HitDieSpender) SpendHitPointDie(ctx context.Context, cmd SpendHitPointDieCommand) (SpendHitPointDieResult, error) {
	if u.roller == nil || cmd.id.IsZero() {
		return nil, invalidInput("zero HitDieSpender or spend Hit Point Die command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	faces := telemetry.Int(AttrHitDieFaces, cmd.die.Faces())
	ctx, span := u.tel.Tracer.Start(ctx, SpanSpendHitPointDie, faces)
	defer span.End()
	timer := telemetry.StartTimer(u.clock, u.dur)
	defer timer.Stop(ctx, telemetry.String(AttrRestOperation, ValueRestOperationSpendHitPointDie))

	spend := cmd.spent.Spend(cmd.pool, cmd.die, cmd.kind)
	allowed, ok := spend.(HitDieSpendAllowed)
	if !ok {
		reason := spend.(HitDieSpendRejected).reason
		u.record(ctx, span, faces, telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure), telemetry.String(AttrResult, ValueResultRejected), telemetry.String(AttrRejection, reason.String()))
		return SpendHitPointDieRejected{reason: reason}, nil
	}
	roll, err := u.roll(cmd.die)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry.StatusError)
		u.record(ctx, span, faces, telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure), telemetry.String(telemetry.AttrErrorType, srderr.ErrorType(err)))
		return nil, err
	}
	hp := regainedHitPoints(roll, cmd.constitution)
	u.hp.Record(ctx, float64(hp), faces)
	u.record(ctx, span, faces, telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess), telemetry.String(AttrResult, ValueResultSpent))
	return SpendHitPointDieApplied{
		roll: roll, hitPointsRegained: hp, constitution: cmd.constitution, spent: allowed.spent,
		event: HitPointDieSpent{id: cmd.id, die: cmd.die, roll: roll, constitutionModifier: cmd.constitution, hitPointsRegained: hp},
	}, nil
}

// record sets the attributes on the span and counts the call once.
func (u HitDieSpender) record(ctx context.Context, span telemetry.Span, faces telemetry.Attr, attrs ...telemetry.Attr) {
	span.SetAttributes(attrs...)
	u.spends.Add(ctx, 1, append(attrs, faces)...)
}

// roll rolls one die of the Hit Point Die's size once and returns the face. A
// roller failure, panic or broken contract comes back as a *srderr.PortError.
func (u HitDieSpender) roll(die content.HitPointDie) (int, error) {
	d, err := dice.NewDie(die.Faces())
	if err != nil { // unreachable: the command holds a valid die
		return 0, err
	}
	expr, err := dice.NewExpression(1, d, 0)
	if err != nil { // unreachable: one die, no modifier
		return 0, err
	}
	res, err := dice.Roll(u.roller, expr)
	if err != nil {
		return 0, err
	}
	return res.DiceTotal(), nil
}
