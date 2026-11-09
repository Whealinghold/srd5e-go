package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ApplySorcererLevelUpCommand is the input of ApplySorcererLevelUp.Run. Character already holds the new Sorcerer level (a
// LevelUpCharacter result) and Gained is the LevelGained event it produced. Metamagic, when not nil, is the full set of options
// the Sorcerer holds after the level, replacing at most one it knew; nil leaves them unchanged. Construct it with a keyed
// literal. Run does not alter it.
type ApplySorcererLevelUpCommand struct {
	Character Character
	Gained    LevelGained
	Metamagic []content.FeatureOptionID
}

// ApplySorcererLevelUpResult is the closed set of outcomes of ApplySorcererLevelUp.Run. Variants: ApplySorcererLevelUpApplied,
// ApplySorcererLevelUpRejected, ApplySorcererLevelUpContentMissing.
type ApplySorcererLevelUpResult interface{ isApplySorcererLevelUpResult() }

// ApplySorcererLevelUpApplied is the outcome of a Sorcerer level applied.
type ApplySorcererLevelUpApplied struct {
	character Character
	owed      int
	grants    SorcererLevelGrants
	events    []Event
}

// Character returns the new Character: the SorcererState seeded or updated with the Sorcery Points of the new level.
func (r ApplySorcererLevelUpApplied) Character() Character { return r.character }

// OwedMetamagicOptions returns how many Metamagic options the level still owes the Sorcerer to choose.
func (r ApplySorcererLevelUpApplied) OwedMetamagicOptions() int { return r.owed }

// Grants returns the feat, Hit Point maximum and always-prepared spell grants of the level for the services that apply them.
func (r ApplySorcererLevelUpApplied) Grants() SorcererLevelGrants { return r.grants }

// Events returns a new slice of the events (a MetamagicOptionsChosen when the options changed); the level itself is the
// LevelGained the command carried.
func (r ApplySorcererLevelUpApplied) Events() []Event { return slices.Clone(r.events) }

// ApplySorcererLevelUpRejected is the outcome of a level a rule refused.
type ApplySorcererLevelUpRejected struct{ Rejection SorcererRejection }

// ApplySorcererLevelUpContentMissing is the outcome when the sorcerer class is not in the source.
type ApplySorcererLevelUpContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the entries not found.
func (r ApplySorcererLevelUpContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (ApplySorcererLevelUpApplied) isApplySorcererLevelUpResult()        {}
func (ApplySorcererLevelUpRejected) isApplySorcererLevelUpResult()       {}
func (ApplySorcererLevelUpContentMissing) isApplySorcererLevelUpResult() {}

// ApplySorcererLevelUp is the use case that applies the Sorcerer side of a gained level (class-sorcerer.md FR-011, FR-035,
// FR-047, FR-058, FR-085 to FR-087, FR-089, FR-127, FR-134): it seeds the SorcererState of a multiclass entrant at Sorcerer level
// 1, raises the Sorcery Points by the rise of the maximum, applies the Metamagic choice, and reports the options still owed and
// the grants of the level. Create it with NewApplySorcererLevelUp; the zero value is not usable.
type ApplySorcererLevelUp struct{ sorcererUse }

// NewApplySorcererLevelUp returns the use case. Options: WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput for a nil source.
func NewApplySorcererLevelUp(source ContentSource, opts ...Option) (ApplySorcererLevelUp, error) {
	u, err := newSorcererUse("apply Sorcerer level up", source, opts)
	return ApplySorcererLevelUp{u}, err
}

// Run applies the level. It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero
// Character or a Gained event of another character; ctx.Err() unwrapped, before any signal, for a cancelled ctx; and a
// *srderr.PortError when the source fails. A Gained event for another class, or for a class level other than 1 with no
// SorcererState, is a rejection (SorcererNotASorcerer), as is a refused Metamagic choice.
func (u ApplySorcererLevelUp) Run(ctx context.Context, cmd ApplySorcererLevelUpCommand) (ApplySorcererLevelUpResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	if cmd.Gained.CharacterID() != cmd.Character.ID() {
		return nil, invalidInput("ApplySorcererLevelUp with a LevelGained event of another character")
	}
	ctx, call, err := u.open(ctx, SpanApplySorcererLevelUp, ValueFeatureSorcererLevelUp, cmd.Character)
	defer call.run.end(ctx)
	if err != nil {
		return nil, err
	}
	reject := func(rej SorcererRejection) ApplySorcererLevelUpResult {
		call.run.rejected(ctx, rej.Kind.String())
		return ApplySorcererLevelUpRejected{Rejection: rej}
	}
	level := call.profile.level.n
	if !call.isLevels || cmd.Gained.Class() != SorcererClassID || cmd.Gained.ClassLevel() != level {
		return reject(refuse(SorcererNotASorcerer)), nil
	}
	if call.missing(ctx) {
		return ApplySorcererLevelUpContentMissing{missing: sorcererMissingContent()}, nil
	}
	state := call.state
	if !call.hasState {
		if level != 1 {
			return reject(refuse(SorcererNotASorcerer)), nil
		}
		state.uses = innateSorceryUses
	}
	previous := SorcererLevel{n: level - 1}
	state.points = min(state.points+call.profile.level.SorceryPointsMaximum()-previous.SorceryPointsMaximum(), call.profile.level.SorceryPointsMaximum())
	var events []Event
	if cmd.Metamagic != nil {
		sel, problems := NewMetamagicSelection(call.profile.level, cmd.Metamagic, metamagicOptions(call.def))
		if len(problems) != 0 {
			return reject(SorcererRejection{Kind: SorcererMetamagicSelectionInvalid, Problems: problems}), nil
		}
		switch t := ChooseMetamagic(cmd.Character.ID(), state, call.profile, sel).(type) {
		case SorcererRefused:
			return reject(t.rejection), nil
		case SorcererChanged:
			state, events = t.state, t.events
		}
	}
	grants, err := SorcererLevelGrantsFor(call.profile, call.def)
	if err != nil {
		call.run.commandError(err)
		return nil, err
	}
	next, err := writeBack(cmd.Character, state)
	if err != nil { // unreachable: the key is valid and a held state of another type was reported by open
		call.run.commandError(err)
		return nil, err
	}
	call.run.succeeded(ctx)
	return ApplySorcererLevelUpApplied{character: next, owed: call.profile.level.MetamagicEntitled() - len(state.Metamagic()), grants: grants, events: events}, nil
}
