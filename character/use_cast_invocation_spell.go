package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// CastInvocationSpellCommand is the input of CastInvocationSpell.Run. Option is the spell-granting invocation and Light the light the
// Warlock stands in (One with Shadows needs Dim Light or Darkness). Construct it with a keyed literal. Run does not alter it.
type CastInvocationSpellCommand struct {
	Character Character
	Option    content.FeatureOptionID
	Light     content.LightLevel
}

// CastInvocationSpellResult is the closed set of outcomes of CastInvocationSpell.Run. Variants: CastInvocationSpellApplied,
// CastInvocationSpellRejected, CastInvocationSpellContentMissing.
type CastInvocationSpellResult interface{ isCastInvocationSpellResult() }

// CastInvocationSpellApplied is the outcome of a cast the rules allowed. The use case does not cast: the host passes Grant to the casting
// service with casting class warlock and no slot.
type CastInvocationSpellApplied struct {
	character Character
	grant     WarlockSpellGrant
	events    []Event
}

// Character returns the new Character, with the Gift of the Depths cast spent when that was the invocation.
func (r CastInvocationSpellApplied) Character() Character { return r.character }

// Grant returns the spell to cast without a slot, with CastModeFeatureGranted and the die rule.
func (r CastInvocationSpellApplied) Grant() WarlockSpellGrant { return r.grant }

// Events returns a new slice holding the InvocationSpellCastWithoutSlot event.
func (r CastInvocationSpellApplied) Events() []Event { return slices.Clone(r.events) }

// CastInvocationSpellRejected is the outcome of a cast a rule refused: the Character is unchanged.
type CastInvocationSpellRejected struct{ Rejection WarlockRejection }

// CastInvocationSpellContentMissing is the outcome when the warlock class is not in the source.
type CastInvocationSpellContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r CastInvocationSpellContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (CastInvocationSpellApplied) isCastInvocationSpellResult()        {}
func (CastInvocationSpellRejected) isCastInvocationSpellResult()       {}
func (CastInvocationSpellContentMissing) isCastInvocationSpellResult() {}

// CastInvocationSpell is the use case that lets a Warlock cast the spell of a spell-granting invocation without a spell slot (SRD 5.2.1,
// "Eldritch Invocation Options", pp. 72-74; class-warlock.md FR-042, FR-059 to FR-063, FR-065, FR-089, FR-165): it loads the Warlock class
// and calls GrantInvocationSpell. Create it with NewCastInvocationSpell; the zero value is not usable.
type CastInvocationSpell struct{ warlockUse }

// NewCastInvocationSpell returns the use case. Options: WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a
// nil source.
func NewCastInvocationSpell(source ContentSource, opts ...Option) (CastInvocationSpell, error) {
	u, err := newWarlockUse("cast an invocation spell", source, true, opts)
	return CastInvocationSpell{u}, err
}

// Run grants the cast. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero Character;
// ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a *srderr.PortError when the source fails. A refusal is a
// CastInvocationSpellRejected result.
func (u CastInvocationSpell) Run(ctx context.Context, cmd CastInvocationSpellCommand) (CastInvocationSpellResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanCastInvocationSpell, ValueFeatureInvocationSpell, cmd.Character, true)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notWarlock(ctx); ok {
		return CastInvocationSpellRejected{Rejection: rej}, nil
	}
	if call.missing(ctx) {
		return CastInvocationSpellContentMissing{missing: warlockMissingContent()}, nil
	}
	next, changed, rej, err := call.commit(ctx, cmd.Character, GrantInvocationSpell(cmd.Character.ID(), call.state, cmd.Option, cmd.Light))
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return CastInvocationSpellRejected{Rejection: rej}, nil
	}
	grant, _ := changed.SpellGrant()
	return CastInvocationSpellApplied{character: next, grant: grant, events: changed.Events()}, nil
}
