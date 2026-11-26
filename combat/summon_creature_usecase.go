package combat

import (
	"context"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// SummonCreatureResult is the closed set of outcomes of SummonCreature.Run. Variants: SummonCreatureApplied,
// SummonCreatureRejected, SummonCreatureContentMissing.
type SummonCreatureResult interface{ isSummonCreatureResult() }

// SummonCreatureApplied: the cast brought creatures, changed a familiar's form, replaced a steed or simulacrum, or reasserted
// control. It holds the new roster, the specs of the creatures to build (with their stat blocks and, for a steed, MountFacts), the
// instructions for the encounter's roster in creation order (FR-077) and the events in the order they happened. Only Run builds
// one.
type SummonCreatureApplied struct {
	roster    SummonedRoster
	creatures []SummonedCreatureSpec
	actions   []SummonedRosterAction
	events    []SummonEvent
}

// Roster returns the new roster.
func (r SummonCreatureApplied) Roster() SummonedRoster { return r.roster }

// Creatures returns a copy of the specs of the creatures made, or of the familiar whose form changed.
func (r SummonCreatureApplied) Creatures() []SummonedCreatureSpec { return slices.Clone(r.creatures) }

// RosterActions returns a copy of the instructions for the encounter's roster, in creation order.
func (r SummonCreatureApplied) RosterActions() []SummonedRosterAction { return slices.Clone(r.actions) }

// Events returns a copy of the events, in the order they happened.
func (r SummonCreatureApplied) Events() []SummonEvent { return slices.Clone(r.events) }

// SummonCreatureRejected: the rules refused the cast. Rejection says why; the roster is as it came in and there are no events.
type SummonCreatureRejected struct {
	Rejection SummonRejection
	roster    SummonedRoster
}

// Roster returns the roster that came in, unchanged.
func (r SummonCreatureRejected) Roster() SummonedRoster { return r.roster }

// SummonCreatureContentMissing: the content source lacks a monster the cast needs. Missing lists them ordered by id. Only Run
// builds one; the roster is as it came in.
type SummonCreatureContentMissing struct {
	missing []content.MissingContent
	roster  SummonedRoster
}

// Missing returns a copy of the entries the source lacks.
func (r SummonCreatureContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

// Roster returns the roster that came in, unchanged.
func (r SummonCreatureContentMissing) Roster() SummonedRoster { return r.roster }

func (SummonCreatureApplied) isSummonCreatureResult()        {}
func (SummonCreatureRejected) isSummonCreatureResult()       {}
func (SummonCreatureContentMissing) isSummonCreatureResult() {}

// SummonCreature is the use case of a cast that brings, keeps or takes control of summoned creatures: it looks up the stat blocks
// the cast names, composes SummonedRoster.Summon and records the signals of summoned-and-companion-creatures.md section 7. It holds
// no rule. Create it with NewSummonCreature; the zero value is not usable. It calls no other use case. It is safe for concurrent use
// if its source, telemetry and clock are.
type SummonCreature struct {
	in     instrumenter
	tel    telemetry.Telemetry
	source ContentSource
}

// NewSummonCreature returns the use case reading monsters through source. Options: WithTelemetry, WithClock. It returns an error
// matching srderr.ErrInvalidInput for a nil source.
func NewSummonCreature(source ContentSource, opts ...Option) (SummonCreature, error) {
	if source == nil {
		return SummonCreature{}, invalidInput("a summons without a content source")
	}
	o := newOptions(opts)
	return SummonCreature{newInstrumenter(o.tel, o.clock), o.tel, source}, nil
}

// Run applies the cast. It looks up the monsters MonsterIDsNeeded names, in that order, and returns
// SummonCreatureContentMissing, class invalid, when the source lacks any; otherwise it returns SummonCreatureApplied or
// SummonCreatureRejected from the transition of SummonedRoster.Summon. It records the shared signals under the operation
// summon_creature with srd5e.combat.result summoned, form_changed, replaced, control_reasserted, rejected_<reason> (refused, or
// limit_exceeded, invalid) or content_missing (invalid), one srd5e.combat.summons count for each creature made and one
// srd5e.combat.summon_endings count for each creature that left. It logs nothing.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or command, and ctx.Err()
// unwrapped, before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type "content_error") with a nil
// result when the source fails, panics or breaks its contract; the failure is recorded once and never logged, and a cancellation
// during a lookup stays visible through errors.Is. A refusal is a result, never an error.
func (u SummonCreature) Run(ctx context.Context, cmd SummonCreatureCommand) (SummonCreatureResult, error) {
	if u.source == nil || cmd.IsZero() {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return run(ctx, u.in, combatOperation{valueOperationSummonCreature, spanSummonCreature},
		func(ctx context.Context) (SummonCreatureResult, error) {
			monsters, missing, err := lookupMonsters(ctx, u.source, MonsterIDsNeeded(cmd.request))
			if err != nil {
				return nil, err
			}
			if len(missing) > 0 {
				return SummonCreatureContentMissing{missing, cmd.roster}, nil
			}
			switch t := cmd.roster.Summon(cmd, monsters...).(type) {
			case SummonedApplied:
				u.record(ctx, t)
				return SummonCreatureApplied{t.roster, t.Creatures(), t.RosterActions(), t.Events()}, nil
			case SummonedRejected:
				return SummonCreatureRejected{t.Rejection, cmd.roster}, nil
			default:
				return nil, invalidInput("the summons made nothing")
			}
		},
		func(res SummonCreatureResult) description {
			switch t := res.(type) {
			case SummonCreatureApplied:
				return description{Class: resultApplied, Name: summonResultName(t.events), Events: summonEventsOf(t.events)}
			case SummonCreatureRejected:
				return description{Class: summonRejectionClass(t.Rejection), Name: rejectionResultName(t.Rejection)}
			}
			return description{Class: resultInvalid, Name: valueResultContentMissing}
		})
}

// record counts the creatures a result brought and the creatures that left.
func (u SummonCreature) record(ctx context.Context, a SummonedApplied) {
	meter := u.tel.WithDefaults().Meter
	if name := summonResultName(a.events); name == valueResultSummoned || name == valueResultReplaced {
		summons := meter.Counter(telemetry.Descriptor{Name: metricSummons, Unit: "{creature}", Description: "Creatures a summons brought, by kind."})
		for _, c := range a.creatures {
			summons.Add(ctx, 1, telemetry.String(attrSummonKind, c.kind.String()))
		}
	}
	recordEndings(ctx, u.tel, a.events)
}

// recordEndings counts each creature of the events that left its summoner or whose control lapsed, by kind and cause.
func recordEndings(ctx context.Context, tel telemetry.Telemetry, events []SummonEvent) {
	var endings telemetry.Counter
	for _, e := range events {
		var kind SummonKind
		var cause SummonEndCause
		switch v := e.(type) {
		case SummonedCreatureDeparted:
			kind, cause = v.kind, v.cause
		case SummonControlLapsed:
			kind, cause = v.kind, SummonEndControlLapsed
		default:
			continue
		}
		if endings == nil {
			endings = tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricSummonEndings, Unit: "{creature}",
				Description: "Summoned creatures that left their summoner or whose control lapsed, by kind and cause."})
		}
		endings.Add(ctx, 1, telemetry.String(attrSummonKind, kind.String()), telemetry.String(attrSummonEndCause, cause.String()))
	}
}

// summonEventsOf widens the events to the Event interface for the shared event counter.
func summonEventsOf(events []SummonEvent) []Event {
	out := make([]Event, len(events))
	for i, e := range events {
		out[i] = e
	}
	return out
}

// summonResultName is the value of srd5e.combat.result of an applied cast: form_changed, control_reasserted, replaced when an old
// creature departed, summoned otherwise.
func summonResultName(events []SummonEvent) string {
	name := valueResultSummoned
	for _, e := range events {
		switch e.(type) {
		case FamiliarFormChanged:
			return valueResultFormChanged
		case SummonControlReasserted:
			return valueResultControlReasserted
		case SummonedCreatureDeparted:
			name = valueResultReplaced
		}
	}
	return name
}

// rejectionResultName is the value of srd5e.combat.result of a refusal: rejected_ and the reason, or the shared limit_exceeded.
func rejectionResultName(r SummonRejection) string {
	if _, over := r.(SummonLimitExceededRejection); over {
		return valueResultLimitExceeded
	}
	return valueResultRejectedPrefix + r.String()
}

// lookupMonsters looks up each id through source, in order, and returns the monsters found and the entries not found ordered by
// id. A source failure, a panic or an answer that breaks the port's contract is a *srderr.PortError and stops the lookups.
func lookupMonsters(ctx context.Context, source ContentSource, ids []content.MonsterID) ([]content.Monster, []content.MissingContent, error) {
	var found []content.Monster
	var missing []content.MissingContent
	for _, id := range ids {
		res, err := srderr.GuardValue(content.Port, "lookup", func() (content.LookupResult, error) { return source.Lookup(ctx, id.Key()) })
		if err != nil {
			return nil, nil, err
		}
		switch r := res.(type) {
		case content.LookupFound:
			m, ok := r.Entry.(content.Monster)
			if !ok || m.ID() != id {
				return nil, nil, srderr.ContractViolation(content.Port, "lookup", "a monster key answered with another entry")
			}
			found = append(found, m)
		case content.LookupNotFound:
			missing = append(missing, id.Missing())
		default:
			return nil, nil, srderr.ContractViolation(content.Port, "lookup", "result is neither found nor not found")
		}
	}
	slices.SortFunc(missing, func(a, b content.MissingContent) int { return strings.Compare(a.ID().String(), b.ID().String()) })
	return found, missing, nil
}
