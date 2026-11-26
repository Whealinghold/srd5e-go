package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// EstablishInitiativeOrderCommand is the immutable input of EstablishInitiativeOrder.Run.
// Build it with NewEstablishInitiativeOrderCommand; the zero value is invalid.
type EstablishInitiativeOrderCommand struct {
	standings []InitiativeStanding
	profiles  []CombatantProfile
	tieBreaks []TieBreak
}

// NewEstablishInitiativeOrderCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for no standing, a blank or repeated id among the standings, a
// profile with a blank id or a size that is not defined, a profile that is missing, extra
// or repeated against the standings, or a zero tie-break. It returns a
// LimitExceededError for more than 256 standings, profiles or tie-breaks (plan P10), before it reads
// one. The lists are copied. Ties are
// not an error here: they are reported by the result. It never panics.
func NewEstablishInitiativeOrderCommand(standings []InitiativeStanding, profiles []CombatantProfile, tieBreaks []TieBreak) (EstablishInitiativeOrderCommand, error) {
	for _, c := range []struct {
		name string
		n    int
	}{{"standings", len(standings)}, {"profiles", len(profiles)}, {"tie-breaks", len(tieBreaks)}} {
		if err := overCap(c.name, c.n, maxInputList); err != nil {
			return EstablishInitiativeOrderCommand{}, err
		}
	}
	if len(standings) == 0 {
		return EstablishInitiativeOrderCommand{}, invalidInput("an Initiative order needs at least one combatant")
	}
	counted := map[CombatantID]bool{}
	for _, s := range standings {
		if s.Combatant.IsZero() || counted[s.Combatant] {
			return EstablishInitiativeOrderCommand{}, invalidInput("a combatant is blank or has two counts")
		}
		counted[s.Combatant] = true
	}
	profiled := map[CombatantID]bool{}
	for _, p := range profiles {
		if !counted[p.ID] || profiled[p.ID] || !p.Size.Valid() {
			return EstablishInitiativeOrderCommand{}, invalidInput("a profile is extra, repeated or has no size")
		}
		profiled[p.ID] = true
	}
	if len(profiled) != len(counted) {
		return EstablishInitiativeOrderCommand{}, invalidInput("a combatant has no profile")
	}
	if slices.ContainsFunc(tieBreaks, func(t TieBreak) bool { return len(t.ids) == 0 }) {
		return EstablishInitiativeOrderCommand{}, invalidInput("zero tie-break")
	}
	return EstablishInitiativeOrderCommand{slices.Clone(standings), slices.Clone(profiles), slices.Clone(tieBreaks)}, nil
}

// EstablishInitiativeOrderResult is the closed set of outcomes of
// EstablishInitiativeOrder.Run. Variants: InitiativeOrderEstablished,
// InitiativeTiesUnresolved, InitiativeTieBreakRejected.
type EstablishInitiativeOrderResult interface{ isEstablishInitiativeOrderResult() }

// InitiativeOrderEstablished: the order is ranked and tie-free; round 1 has begun with
// the first combatant active. The events are TrackerRoundStarted(1) then TrackerTurnStarted.
type InitiativeOrderEstablished struct {
	appliedTurnStep
	ties []string
}

// InitiativeTiesUnresolved: tied groups have no tie-break; nothing is established. The
// host chooses an order for each group and calls again.
type InitiativeTiesUnresolved struct {
	groups []TiedGroup
	ties   []string
}

// InitiativeTieBreakRejected: at least one tie-break was unusable (host input the engine
// could not use); nothing is established.
type InitiativeTieBreakRejected struct {
	rejections []TieBreakRejection
	ties       []string
}

func (InitiativeOrderEstablished) isEstablishInitiativeOrderResult() {}
func (InitiativeTiesUnresolved) isEstablishInitiativeOrderResult()   {}
func (InitiativeTieBreakRejected) isEstablishInitiativeOrderResult() {}

// Groups returns a copy of the unresolved groups, highest count first.
func (r InitiativeTiesUnresolved) Groups() []TiedGroup { return slices.Clone(r.groups) }

// Rejections returns a copy of the rejected tie-breaks in submitted order.
func (r InitiativeTieBreakRejected) Rejections() []TieBreakRejection {
	return slices.Clone(r.rejections)
}

// EstablishInitiativeOrder is the use case of fixing the Initiative order and starting
// round 1 (SRD 5.2.1, "Combat Step by Step", p. 13). It rolls nothing. Create it with
// NewEstablishInitiativeOrder; the zero value is usable and silent.
type EstablishInitiativeOrder struct{ turnUse }

// NewEstablishInitiativeOrder returns the use case. Options: WithTelemetry, WithClock.
func NewEstablishInitiativeOrder(opts ...Option) EstablishInitiativeOrder {
	return EstablishInitiativeOrder{newTurnUse(opts)}
}

// Run ranks the standings from the highest count to the lowest and applies the valid
// tie-breaks. With no unresolved tie and no rejected tie-break it returns the tracker of
// round 1; otherwise InitiativeTieBreakRejected (class invalid, any rejection) or
// InitiativeTiesUnresolved (class refused). It records the shared signals and one
// srd5e.combat.initiative_ties per tied group, plus one rejected per tie-break that names
// no tied group.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It rolls
// no dice and never fails otherwise.
func (u EstablishInitiativeOrder) Run(ctx context.Context, cmd EstablishInitiativeOrderCommand) (EstablishInitiativeOrderResult, error) {
	if len(cmd.standings) == 0 {
		return nil, invalidInput("zero command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var ties []string
	res, err := run(ctx, u.in, combatOperation{valueOperationEstablishInitiativeOrder, spanEstablishInitiativeOrder},
		func(context.Context) (EstablishInitiativeOrderResult, error) {
			r := rankStandings(cmd.standings, cmd.tieBreaks)
			ties = r.ties
			switch {
			case len(r.rejections) > 0:
				return InitiativeTieBreakRejected{r.rejections, r.ties}, nil
			case len(r.unresolved) > 0:
				return InitiativeTiesUnresolved{r.unresolved, r.ties}, nil
			}
			t := establishTracker(r.order, cmd.profiles)
			first := r.order[0].Combatant
			return InitiativeOrderEstablished{appliedTurnStep{t, []TurnEvent{
				TrackerRoundStarted{round: 1}, TrackerTurnStarted{combatant: first, round: 1}}}, r.ties}, nil
		},
		func(r EstablishInitiativeOrderResult) description {
			switch v := r.(type) {
			case InitiativeOrderEstablished:
				return description{Class: resultApplied, Name: valueResultEstablished, Events: widen(v.events)}
			case InitiativeTiesUnresolved:
				return description{Class: resultRefused, Name: valueResultTiesUnresolved}
			case InitiativeTieBreakRejected:
				return description{Class: resultInvalid, Name: valueResultTieBreakRejected}
			default:
				panic(unreachableError("establish result", r))
			}
		})
	if err != nil {
		return nil, err
	}
	counter := u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricInitiativeTies, Unit: "{tie}",
		Description: "Tied groups and unusable tie-breaks, by outcome."})
	for _, o := range ties {
		counter.Add(ctx, 1, telemetry.String(attrTieOutcome, o))
	}
	return res, nil
}
