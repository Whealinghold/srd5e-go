package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// CreateSpellSlotCommand is the input of CreateSpellSlot.Run. Construct it with a keyed literal. Run does not alter it.
type CreateSpellSlotCommand struct {
	Character Character
	// SlotLevel is the level of the slot to create, 1 to 9; a level above 5 is a refusal, one outside 1 to 9 an error.
	SlotLevel int
}

// CreateSpellSlotResult is the closed set of outcomes of CreateSpellSlot.Run. Variants: CreateSpellSlotApplied,
// CreateSpellSlotRejected, CreateSpellSlotContentMissing.
type CreateSpellSlotResult interface{ isCreateSpellSlotResult() }

// CreateSpellSlotApplied is the outcome of a slot created.
type CreateSpellSlotApplied struct {
	character Character
	events    []Event
}

// Character returns the new Character: the Sorcery Points spent and the created slot added to its SorcererState.
func (r CreateSpellSlotApplied) Character() Character { return r.character }

// Events returns a new slice holding the SpellSlotCreated event.
func (r CreateSpellSlotApplied) Events() []Event { return slices.Clone(r.events) }

// CreateSpellSlotRejected is the outcome of a creation a rule refused.
type CreateSpellSlotRejected struct{ Rejection SorcererRejection }

// CreateSpellSlotContentMissing is the outcome when the sorcerer class is not in the source.
type CreateSpellSlotContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r CreateSpellSlotContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (CreateSpellSlotApplied) isCreateSpellSlotResult()        {}
func (CreateSpellSlotRejected) isCreateSpellSlotResult()       {}
func (CreateSpellSlotContentMissing) isCreateSpellSlotResult() {}

// CreateSpellSlot is the use case that transforms Sorcery Points into a spell slot (SRD 5.2.1, "Creating Spell Slots", p. 66;
// class-sorcerer.md FR-052 to FR-054, FR-114), reading the cost and minimum level from the Creating Spell Slots table of the
// class definition. Create it with NewCreateSpellSlot; the zero value is not usable.
type CreateSpellSlot struct{ sorcererUse }

// NewCreateSpellSlot returns the use case. Options: WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewCreateSpellSlot(source ContentSource, opts ...Option) (CreateSpellSlot, error) {
	u, err := newSorcererUse("create spell slot", source, opts)
	return CreateSpellSlot{u}, err
}

// Run creates the slot. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character, and, recorded on the span, for a slot level outside 1 to 9; ctx.Err() unwrapped, before any signal, for a cancelled
// ctx; and a *srderr.PortError when the source fails. A refusal (SorcererSlotLevelNotCreatable, SorcererLevelBelowMinimum,
// SorcererSorceryPointsInsufficient, SorcererFeatureNotGained, SorcererNotASorcerer) is a CreateSpellSlotRejected result.
func (u CreateSpellSlot) Run(ctx context.Context, cmd CreateSpellSlotCommand) (CreateSpellSlotResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	ctx, call, err := u.open(ctx, SpanCreateSpellSlot, ValueFeatureFontOfMagic, cmd.Character)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	if rej, ok := call.notSorcerer(ctx); ok {
		return CreateSpellSlotRejected{Rejection: rej}, nil
	}
	if call.missing(ctx) {
		return CreateSpellSlotContentMissing{missing: sorcererMissingContent()}, nil
	}
	var rule SlotCreationRule
	found := false
	if table, ok := creatingSpellSlots(call.def); ok {
		rule, found = SlotCreationRuleFor(table, cmd.SlotLevel)
	}
	t, err := CreateSlot(cmd.Character.ID(), call.state, call.profile, cmd.SlotLevel, rule, found)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	next, events, rej, err := call.commit(ctx, cmd.Character, t)
	switch {
	case err != nil:
		return nil, err
	case rej.Kind != 0:
		return CreateSpellSlotRejected{Rejection: rej}, nil
	}
	return CreateSpellSlotApplied{character: next, events: events}, nil
}
