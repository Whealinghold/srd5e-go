package character

import (
	"context"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// HitPointChoice is how a level's Hit Points are decided (SRD 5.2.1, "Gaining a
// Level" step 2, p. 23). It is a closed set. Variants: RollHitDie,
// TakeFixedValue.
type HitPointChoice interface{ isHitPointChoice() }

// RollHitDie rolls one Hit Point Die of the class's size through the dice port.
type RollHitDie struct{}

// TakeFixedValue takes the class's fixed value, the die's faces halved plus 1,
// without rolling.
type TakeFixedValue struct{}

func (RollHitDie) isHitPointChoice()     {}
func (TakeFixedValue) isHitPointChoice() {}

// LevelUpCharacterCommand is the input of LevelUpCharacter.Run. Construct it
// with a keyed literal. Run does not alter it.
type LevelUpCharacterCommand struct {
	Character      Character
	Class          content.ClassID
	HitPointChoice HitPointChoice
}

// LevelUpCharacterResult is the closed set of outcomes of
// LevelUpCharacter.Run. Variants: LevelUpCharacterSucceeded,
// LevelUpCharacterRejected, LevelUpCharacterContentMissing.
type LevelUpCharacterResult interface{ isLevelUpCharacterResult() }

// LevelUpCharacterSucceeded is the outcome of a level gained: the new Character
// and the events. Only Run builds one.
type LevelUpCharacterSucceeded struct {
	character Character
	events    []Event
}

// Character returns the Character after the level.
func (r LevelUpCharacterSucceeded) Character() Character { return r.character }

// Events returns a new slice of the events, in order.
func (r LevelUpCharacterSucceeded) Events() []Event { return slices.Clone(r.events) }

// LevelUpCharacterRejected is the outcome of a level a rule refused.
type LevelUpCharacterRejected struct{ Rejection LevelUpRejection }

// LevelUpCharacterContentMissing is the outcome when the chosen class, or for a
// new class a current class, is unknown to the content source. It names at
// least one entry. Only Run builds one; the zero value names nothing.
type LevelUpCharacterContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the unknown entries, ordered by id, without
// duplicates.
func (r LevelUpCharacterContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (LevelUpCharacterSucceeded) isLevelUpCharacterResult()      {}
func (LevelUpCharacterRejected) isLevelUpCharacterResult()       {}
func (LevelUpCharacterContentMissing) isLevelUpCharacterResult() {}

// LevelUpCharacter is the use case that gains a level for a Character: it
// looks up the chosen class (and, for a new class, every current class), rolls
// the Hit Point Die when asked, calls Character.LevelUp and emits the
// srd5e.character.* signals. It holds no rules logic. Create it with
// NewLevelUpCharacter; the zero value is not usable. It is safe for concurrent
// use only if its source, roller, telemetry and clock are.
type LevelUpCharacter struct {
	source ContentSource
	roller Roller
	sig    operationSignals
}

// NewLevelUpCharacter returns the use case. Options: WithSource and WithRoller
// (both required), WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput when the source or the roller is missing.
func NewLevelUpCharacter(opts ...Option) (LevelUpCharacter, error) {
	o := newOptions(opts)
	if o.source == nil || o.roller == nil {
		return LevelUpCharacter{}, invalidInput("level up without a content source and a dice roller")
	}
	return LevelUpCharacter{source: o.source, roller: o.roller, sig: newOperationSignals(o)}, nil
}

// Run gains a level for cmd.Character in cmd.Class. Every lookup completes
// before any rule is evaluated, so a missing class wins over every rejection.
// The die is rolled, when asked, before the rules run, so a refused level still
// consumes the roll.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case, a zero Character, an invalid class or a nil
// HitPointChoice, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. It returns a *srderr.PortError (error type "content_error"
// or "dice_roller_error") with a nil result when the source or the roller
// fails, panics or breaks its contract; the failure is recorded once and never
// logged.
func (u LevelUpCharacter) Run(ctx context.Context, cmd LevelUpCharacterCommand) (LevelUpCharacterResult, error) {
	if u.source == nil {
		return nil, invalidInput("zero LevelUpCharacter; use NewLevelUpCharacter")
	}
	var choice string
	switch cmd.HitPointChoice.(type) {
	case RollHitDie:
		choice = ValueHitPointChoiceRolled
	case TakeFixedValue:
		choice = ValueHitPointChoiceFixed
	default:
		return nil, invalidInput("level up without a Hit Point choice")
	}
	if cmd.Character.IsZero() || !cmd.Class.Valid() {
		return nil, invalidInput("level up of a zero character or an invalid class")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanLevelUp, ValueOperationLevelUp, telemetry.String(AttrHitPointChoice, choice))
	defer run.end(ctx)

	loaded, err := u.load(ctx, cmd)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(loaded.missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return LevelUpCharacterContentMissing{missing: loaded.missing}, nil
	}
	class, err := NewAdvancementClass(cmd.Class, loaded.class.HitPointDie(), loaded.class.PrimaryAbility())
	if err != nil { // unreachable for a decoded class
		run.portFailure(ctx, err, "")
		return nil, err
	}
	gain, err := u.gain(cmd.HitPointChoice, class)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	tr, err := cmd.Character.LevelUp(class, gain, loaded.requirements)
	if err != nil { // unreachable: the gain was built for the class
		run.portFailure(ctx, err, "")
		return nil, err
	}
	switch t := tr.(type) {
	case TransitionChanged[LevelUpRejection]:
		run.finish(ctx, ValueResultChanged, "")
		return LevelUpCharacterSucceeded(t), nil
	case TransitionRejected[LevelUpRejection]:
		run.finish(ctx, ValueResultRejected, levelUpRejectionValue(t.rejection))
		return LevelUpCharacterRejected{Rejection: t.rejection}, nil
	default:
		panic(invalidInput("unknown transition")) // unreachable: Transition is closed
	}
}

// gain decides the level's Hit Point gain: a one-die roll through the port, or
// the fixed value.
func (u LevelUpCharacter) gain(choice HitPointChoice, class AdvancementClass) (HitPointGain, error) {
	if _, ok := choice.(TakeFixedValue); ok {
		return NewFixedHitPointGain(class.die.Faces()/2 + 1)
	}
	die, err := dice.NewDie(class.die.Faces())
	if err != nil { // unreachable: the class die was checked
		return nil, err
	}
	expr, err := dice.NewExpression(1, die, 0)
	if err != nil { // unreachable: one die, no modifier
		return nil, err
	}
	res, err := dice.Roll(u.roller, expr) // a *srderr.PortError; a result outside the die cannot be built
	if err != nil {
		return nil, err
	}
	return NewRolledHitPointGain(res.Total(), class.die)
}

// load looks up the chosen class and, when it is new, every current class, all
// before reporting what is missing, inside the load_content span.
func (u LevelUpCharacter) load(ctx context.Context, cmd LevelUpCharacterCommand) (loadedClasses, error) {
	return loadClasses(ctx, u.source, u.sig.tel.Tracer, SpanLoadContent, cmd.Character, cmd.Class, cmd.Character.classes.LevelIn(cmd.Class) == 0)
}

// loadedClasses is what the lookups of one call found.
type loadedClasses struct {
	class        content.ClassDefinition
	requirements map[content.ClassID]content.AbilityRequirement
	missing      []content.MissingContent
}

// loadClasses looks up the chosen class and, when withCurrent is set (the
// character does not hold it), every current class, all before reporting what is missing, inside a span
// called spanName. The missing entries are ordered by id without duplicates.
func loadClasses(ctx context.Context, source ContentSource, tracer telemetry.Tracer, spanName string, c Character, chosen content.ClassID, withCurrent bool) (loadedClasses, error) {
	ctx, span := tracer.Start(ctx, spanName)
	defer span.End()
	out := loadedClasses{requirements: map[content.ClassID]content.AbilityRequirement{}}
	classes := []content.ClassID{chosen}
	if withCurrent {
		for _, e := range c.classes.entries {
			classes = append(classes, e.Class)
		}
	}
	for _, id := range classes {
		entry, found, err := lookupEntry(ctx, source, id.Key())
		if err == nil && found {
			def, ok := entry.(content.ClassDefinition)
			switch {
			case !ok:
				err = srderr.ContractViolation(content.Port, "lookup", "a class key answered with another entry type")
			case id == chosen:
				out.class = def
			default:
				out.requirements[id] = def.PrimaryAbility()
			}
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(telemetry.StatusError)
			span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
			return loadedClasses{}, err
		}
		if !found {
			out.missing = append(out.missing, id.Missing())
		}
	}
	slices.SortFunc(out.missing, func(a, b content.MissingContent) int { return strings.Compare(a.ID().String(), b.ID().String()) })
	outcome := telemetry.OutcomeSuccess
	if len(out.missing) > 0 {
		outcome = telemetry.OutcomeFailure
		span.SetStatus(telemetry.StatusError)
	}
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, outcome))
	return out, nil
}

// levelUpRejectionValue maps a rejection to its ValueRejection* value.
func levelUpRejectionValue(r LevelUpRejection) string {
	switch r.(type) {
	case LevelUpMaximumLevel:
		return ValueRejectionMaximumLevel
	case LevelUpInsufficientExperience:
		return ValueRejectionInsufficientExperience
	}
	return ValueRejectionMulticlassPrerequisitesNotMet
}

// lookupEntry asks source for key and returns the entry and true, or false when
// it is not found. A source failure, a panic and a broken contract come back as
// a *srderr.PortError.
func lookupEntry(ctx context.Context, source ContentSource, key content.Key) (content.Entry, bool, error) {
	res, err := srderr.GuardValue(content.Port, "lookup", func() (content.LookupResult, error) {
		return source.Lookup(ctx, key)
	})
	if err != nil {
		return nil, false, err
	}
	switch r := res.(type) {
	case content.LookupFound:
		if r.Entry == nil {
			return nil, false, srderr.ContractViolation(content.Port, "lookup", "found result without an entry")
		}
		return r.Entry, true, nil
	case content.LookupNotFound:
		return nil, false, nil
	}
	return nil, false, srderr.ContractViolation(content.Port, "lookup", "result is neither found nor not found")
}
