package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// RollInitiativeCommand is the immutable input of RollInitiative.Run. Build it with
// NewRollInitiativeCommand; the zero value is invalid.
type RollInitiativeCommand struct {
	method       InitiativeMethod
	participants []InitiativeParticipant
}

// NewRollInitiativeCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for a method that is not defined, no participant, a zero
// participant or a combatant id that appears twice (in one entry or in two). It returns a
// LimitExceededError for more than 256 participants (plan P10), before it reads one. The
// participants are copied. It never panics.
func NewRollInitiativeCommand(method InitiativeMethod, participants []InitiativeParticipant) (RollInitiativeCommand, error) {
	if err := overCap("initiative participants", len(participants), maxInputList); err != nil {
		return RollInitiativeCommand{}, err
	}
	if !method.Valid() || len(participants) == 0 {
		return RollInitiativeCommand{}, invalidInput("rolling Initiative needs a method and at least one participant")
	}
	seen := map[CombatantID]bool{}
	for _, p := range participants {
		if p.IsZero() {
			return RollInitiativeCommand{}, invalidInput("zero Initiative participant")
		}
		for _, id := range p.ids {
			if seen[id] {
				return RollInitiativeCommand{}, invalidInput("a combatant appears twice in the Initiative entries")
			}
			seen[id] = true
		}
	}
	return RollInitiativeCommand{method, slices.Clone(participants)}, nil
}

// RollInitiativeResult is the closed set of outcomes of RollInitiative.Run. Variants:
// RollInitiativeRolled.
type RollInitiativeResult interface{ isRollInitiativeResult() }

// RollInitiativeRolled: one breakdown per participant entry, in the order listed.
type RollInitiativeRolled struct{ breakdowns []InitiativeBreakdown }

func (RollInitiativeRolled) isRollInitiativeResult() {}

// Breakdowns returns a copy of the breakdowns.
func (r RollInitiativeRolled) Breakdowns() []InitiativeBreakdown { return slices.Clone(r.breakdowns) }

// RollInitiative is the use case of making Initiative counts (SRD 5.2.1, "Initiative",
// p. 13). It rolls through the dice port, once per rolled entry, and never through
// d20.Resolver. Create it with NewRollInitiative; the zero value is not usable.
type RollInitiative struct {
	turnUse
	roller Roller
}

// NewRollInitiative returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewRollInitiative(roller Roller, opts ...Option) (RollInitiative, error) {
	if roller == nil {
		return RollInitiative{}, invalidInput("rolling Initiative without a dice roller")
	}
	return RollInitiative{newTurnUse(opts), roller}, nil
}

// Run makes one request of 1d20 (2d20 under Advantage or Disadvantage) per rolled
// entry in the order listed, builds the Roll from the faces and adds the modifiers;
// a score entry rolls nothing. It records the shared signals, one
// srd5e.combat.initiative_counts per entry and one srd5e.combat.initiative_participants.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx.
// It returns a *srderr.PortError (error type "dice_roller_error") with a nil result when
// the roller fails, panics or answers another expression; the failure is recorded once,
// never logged, and no count is recorded.
func (u RollInitiative) Run(ctx context.Context, cmd RollInitiativeCommand) (RollInitiativeResult, error) {
	if u.roller == nil || len(cmd.participants) == 0 {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationRollInitiative, spanRollInitiative},
		func(context.Context) (RollInitiativeResult, error) {
			var out []InitiativeBreakdown
			for _, p := range cmd.participants {
				mode := p.Mode()
				if cmd.method == InitiativeMethodScore {
					out = append(out, ScoredInitiative{p.IDs(), mode, p.mod, p.score()})
					continue
				}
				rolled, err := dice.Roll(u.roller, dice.MustExpression(mode.DiceCount(), dice.D20(), 0))
				if err != nil {
					return nil, err
				}
				faces := make([]int, 0, mode.DiceCount())
				for _, r := range rolled.Rolls() {
					faces = append(faces, r.Value())
				}
				roll, err := d20.NewRoll(mode, faces)
				if err != nil {
					return nil, err
				}
				out = append(out, RolledInitiative{p.IDs(), mode, roll, p.mod, p.rollOnly, InitiativeCount(0).FromRoll(roll, p.mod, p.rollOnly)})
			}
			return RollInitiativeRolled{out}, nil
		},
		func(RollInitiativeResult) description {
			return description{Class: resultApplied, Name: valueResultRolled}
		})
	if err != nil {
		return nil, err
	}
	meter := u.tel.WithDefaults().Meter
	counts := meter.Counter(telemetry.Descriptor{Name: metricInitiativeCounts, Unit: "{initiative_count}",
		Description: "Initiative counts produced, by method and roll mode."})
	method := telemetry.String(attrInitiativeMethod, cmd.method.String())
	ids := 0
	for _, b := range res.(RollInitiativeRolled).breakdowns {
		counts.Add(ctx, 1, method, telemetry.String(d20.AttrRollMode, b.Mode().String()))
		ids += len(b.IDs())
	}
	meter.Histogram(telemetry.Descriptor{Name: metricInitiativeParticipants, Unit: "{combatant}",
		Description: "Combatant ids per RollInitiative call."}).Record(ctx, float64(ids), method)
	return res, nil
}
