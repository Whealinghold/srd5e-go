package character

import (
	"context"
	"slices"
)

// AwardExperienceCommand is the input of AwardExperience.Run. Construct it with
// a keyed literal. Run does not alter it.
type AwardExperienceCommand struct {
	Character Character
	Amount    ExperienceAmount
}

// AwardExperienceResult is the closed set of outcomes of AwardExperience.Run.
// Variants: AwardExperienceSucceeded.
type AwardExperienceResult interface{ isAwardExperienceResult() }

// AwardExperienceSucceeded is the outcome of an award: the new Character and the
// events. Only Run builds one.
type AwardExperienceSucceeded struct {
	character Character
	events    []Event
}

// Character returns the Character after the award.
func (r AwardExperienceSucceeded) Character() Character { return r.character }

// Events returns a new slice of the events, in order.
func (r AwardExperienceSucceeded) Events() []Event { return slices.Clone(r.events) }

func (AwardExperienceSucceeded) isAwardExperienceResult() {}

// AwardExperience is the use case that awards Experience Points to a Character
// and emits the srd5e.character.* signals. It holds no rules logic and needs no
// port beyond telemetry. Create it with NewAwardExperience; the zero value is
// not usable. It is safe for concurrent use only if its telemetry and clock are.
type AwardExperience struct {
	sig operationSignals
	set bool
}

// NewAwardExperience returns the use case. Options: WithTelemetry, WithClock;
// the others are ignored.
func NewAwardExperience(opts ...Option) AwardExperience {
	return AwardExperience{sig: newOperationSignals(newOptions(opts)), set: true}
}

// Run awards cmd.Amount to cmd.Character. It returns an error matching
// srderr.ErrInvalidInput, before any signal, for the zero use case or a zero
// Character or amount, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. A valid command always succeeds; it never returns
// srderr.PortError.
func (u AwardExperience) Run(ctx context.Context, cmd AwardExperienceCommand) (AwardExperienceResult, error) {
	if !u.set || cmd.Character.IsZero() || cmd.Amount.IsZero() {
		return nil, invalidInput("award experience: zero use case, character or amount")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAwardExperience, ValueOperationAwardExperience)
	defer run.end(ctx)
	t := cmd.Character.AwardExperience(cmd.Amount)
	run.finish(ctx, ValueResultChanged, "")
	return AwardExperienceSucceeded{character: t.Character(), events: t.events}, nil
}
