package character

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// UseMetamagicCommand is the input of UseMetamagic.Run: the Character, the option and the casting it is used on. Construct it
// with a keyed literal. Run does not alter it.
type UseMetamagicCommand struct {
	Character  Character
	Option     content.FeatureOptionID
	Facts      MetamagicSpellFacts
	Context    MetamagicCastContext
	Parameters MetamagicParameters
}

// UseMetamagicResult is the closed set of outcomes of UseMetamagic.Run. Variants: UseMetamagicApplied, UseMetamagicRejected,
// UseMetamagicContentMissing.
type UseMetamagicResult interface{ isUseMetamagicResult() }

// UseMetamagicApplied is the outcome of a use the rules allowed.
type UseMetamagicApplied struct {
	character   Character
	application MetamagicApplication
	events      []Event
}

// Character returns the new Character: the Sorcery Points spent from its SorcererState.
func (r UseMetamagicApplied) Character() Character { return r.character }

// Application returns what the option did to the casting, with its effect and the points it cost.
func (r UseMetamagicApplied) Application() MetamagicApplication { return r.application }

// Events returns a new slice holding the MetamagicUsed event.
func (r UseMetamagicApplied) Events() []Event { return slices.Clone(r.events) }

// UseMetamagicRejected is the outcome of a use a rule refused.
type UseMetamagicRejected struct{ Rejection SorcererRejection }

// UseMetamagicContentMissing is the outcome when the sorcerer class is not in the source.
type UseMetamagicContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r UseMetamagicContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (UseMetamagicApplied) isUseMetamagicResult()        {}
func (UseMetamagicRejected) isUseMetamagicResult()       {}
func (UseMetamagicContentMissing) isUseMetamagicResult() {}

// UseMetamagic is the use case that uses a Metamagic option on a casting (SRD 5.2.1, "Level 2: Metamagic", "Metamagic Options",
// pp. 66-67; class-sorcerer.md FR-045, FR-046, FR-059 to FR-082, FR-110 to FR-118, FR-120, FR-123), reading the option's cost
// from the class definition. Create it with NewUseMetamagic; the zero value is not usable.
type UseMetamagic struct{ sorcererUse }

// NewUseMetamagic returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// for a nil source.
func NewUseMetamagic(source ContentSource, opts ...Option) (UseMetamagic, error) {
	u, err := newSorcererUse("use Metamagic", source, opts)
	return UseMetamagic{u}, err
}

// Run uses the option. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError when the source fails. An option
// the class definition does not list is a rejection (SorcererMetamagicOptionNotKnown), as is every other refusal of
// ApplyMetamagic.
func (u UseMetamagic) Run(ctx context.Context, cmd UseMetamagicCommand) (UseMetamagicResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanUseMetamagic, ValueFeatureMetamagic, cmd.Character)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notSorcerer(ctx); ok {
		return UseMetamagicRejected{Rejection: rej}, nil
	}
	if call.missing(ctx) {
		return UseMetamagicContentMissing{missing: sorcererMissingContent()}, nil
	}
	effect, known := MetamagicEffectOf(cmd.Option)
	idx := slices.IndexFunc(metamagicOptions(call.def), func(o content.FeatureOption) bool { return o.ID() == cmd.Option })
	if !known || idx < 0 {
		rej := refuse(SorcererMetamagicOptionNotKnown)
		call.run.rejected(ctx, rej.Kind.String())
		return UseMetamagicRejected{Rejection: rej}, nil
	}
	cost := metamagicOptions(call.def)[idx].PointCost()
	cha := cmd.Character.AbilityScores().Modifier(ability.Charisma)
	req := MetamagicRequest{Effect: effect, Cost: cost, Facts: cmd.Facts, Context: cmd.Context, Parameters: cmd.Parameters}
	switch t := ApplyMetamagic(cmd.Character.ID(), call.state, call.profile, cha, req).(type) {
	case MetamagicRefused:
		call.run.rejected(ctx, t.rejection.Kind.String())
		return UseMetamagicRejected{Rejection: t.rejection}, nil
	case MetamagicApplied:
		next, err := writeBack(cmd.Character, t.state)
		if err != nil { // unreachable: the key is valid and a held state of another type was reported by open
			call.run.commandError(err)
			return nil, err
		}
		call.run.succeeded(ctx)
		return UseMetamagicApplied{character: next, application: t.application, events: t.events}, nil
	default:
		panic(fmt.Errorf("unreachable: %T", t))
	}
}
