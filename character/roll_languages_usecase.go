package character

import (
	"context"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// RollLanguagesCommand is the input of RollLanguages.Run. It has no fields; it keeps
// the call shape of the other use cases.
type RollLanguagesCommand struct{}

// RollLanguagesResult is the closed set of outcomes of RollLanguages.Run. Variants:
// RollLanguagesSucceeded, RollLanguagesDuplicate.
type RollLanguagesResult interface{ isRollLanguagesResult() }

// RollLanguagesSucceeded is two different Standard languages and the two d12 rolls
// that gave them. Only Run builds one.
type RollLanguagesSucceeded struct {
	languages [2]content.Language
	rolls     [2]int
}

// Languages returns a new slice of the two languages, in roll order. They are valid
// as the languages of a CreateCharacterCommand.
func (r RollLanguagesSucceeded) Languages() []content.Language {
	return append([]content.Language(nil), r.languages[:]...)
}

// Rolls returns a new slice of the two d12 rolls, in roll order.
func (r RollLanguagesSucceeded) Rolls() []int { return append([]int(nil), r.rolls[:]...) }

// RollLanguagesDuplicate is two rolls that gave the same language; the SRD has no
// rule for it, so the host rolls again. Only Run builds one.
type RollLanguagesDuplicate struct {
	language content.Language
	rolls    [2]int
}

// Language returns the language both rolls gave.
func (r RollLanguagesDuplicate) Language() content.Language { return r.language }

// Rolls returns a new slice of the two d12 rolls, in roll order.
func (r RollLanguagesDuplicate) Rolls() []int { return append([]int(nil), r.rolls[:]...) }

func (RollLanguagesSucceeded) isRollLanguagesResult() {}
func (RollLanguagesDuplicate) isRollLanguagesResult() {}

// RollLanguages is the use case that rolls two languages on the Standard Languages
// table (SRD 5.2.1, "Choose Languages", p. 20). It reads no content and keeps no
// state. Create it with NewRollLanguages; the zero value is not usable. It is safe
// for concurrent use only if its roller, telemetry and clock are.
type RollLanguages struct {
	roller Roller
	sig    operationSignals
}

// NewRollLanguages returns the use case rolling through roller. Options:
// WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for a nil roller.
func NewRollLanguages(roller Roller, opts ...Option) (RollLanguages, error) {
	if roller == nil {
		return RollLanguages{}, invalidInput("roll languages without a dice roller")
	}
	return RollLanguages{roller: roller, sig: newOperationSignals(newOptions(opts))}, nil
}

// Run rolls two separate 1d12 requests in order and maps each to a language. When both
// give the same language it returns RollLanguagesDuplicate and does not roll again.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
// It returns a *srderr.PortError (error type "dice_roller_error") with a nil result
// when the roller fails, panics or answers another expression; the failure is recorded
// once and never logged.
func (u RollLanguages) Run(ctx context.Context, _ RollLanguagesCommand) (RollLanguagesResult, error) {
	if u.roller == nil {
		return nil, invalidInput("zero RollLanguages; use NewRollLanguages")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanRollLanguages, ValueOperationRollLanguages)
	defer run.end(ctx)
	var rolls [2]int
	var langs [2]content.Language
	for i := range rolls {
		res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.MustDie(12), 0))
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		rolls[i] = res.Total()
		if langs[i], err = (StandardLanguageRoll{}).LanguageFor(rolls[i]); err != nil { // the roller breaks its contract: a face outside 1 to 12
			run.commandError(err)
			return nil, err
		}
	}
	if langs[0] == langs[1] {
		run.finish(ctx, ValueResultDuplicate, "")
		return RollLanguagesDuplicate{language: langs[0], rolls: rolls}, nil
	}
	run.finish(ctx, ValueResultRolled, "")
	return RollLanguagesSucceeded{languages: langs, rolls: rolls}, nil
}
