package character

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// StartAtHigherLevelCommand is the input of StartAtHigherLevel.Start: the
// arguments of the level 1 Character, the starting level and one step for each
// level from 2 to the starting level (SRD 5.2.1, "Starting at Higher Levels",
// p. 24). Construct it with a keyed literal. Start does not alter it and copies
// Steps.
type StartAtHigherLevelCommand struct {
	CharacterID   Identifier
	ClassID       content.ClassID
	SpeciesID     content.SpeciesID
	BackgroundID  content.BackgroundID
	AbilityScores ability.Scores
	StartingLevel Level
	Steps         []StartingLevelStep
}

// StartAtHigherLevelResult is the closed set of outcomes of
// StartAtHigherLevel.Start. Variants: StartAtHigherLevelStarted,
// StartAtHigherLevelRejected, StartAtHigherLevelContentMissing.
type StartAtHigherLevelResult interface{ isStartAtHigherLevelResult() }

// StartAtHigherLevelStarted is the outcome of a plan applied in full: the
// Character at the starting level, the events in order (Created, then for each
// step LevelGained, SubclassChosen if any, the AbilityScoreChanged events and at
// most one HitPointMaximumChanged) and the feat grants the steps consumed. Only
// Start builds one.
type StartAtHigherLevelStarted struct {
	character Character
	events    []Event
	consumed  []ConsumedFeatGrant
}

// Character returns the Character at the starting level.
func (r StartAtHigherLevelStarted) Character() Character { return r.character }

// Events returns a new slice of the events, in order.
func (r StartAtHigherLevelStarted) Events() []Event { return slices.Clone(r.events) }

// ConsumedGrants returns a new slice of one ConsumedFeatGrant per applied
// increase, in step order; a host keeps them beside the character.
func (r StartAtHigherLevelStarted) ConsumedGrants() []ConsumedFeatGrant {
	return slices.Clone(r.consumed)
}

// StartAtHigherLevelRejected is the outcome of a plan a rule refused; no
// character was built and no die was rolled.
type StartAtHigherLevelRejected struct{ Reason HigherLevelStartRejection }

// StartAtHigherLevelContentMissing is the outcome when the initial class, the
// species, the background or a class of a step is unknown to the content source.
// It names at least one entry. Only Start builds one; the zero value names
// nothing.
type StartAtHigherLevelContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the unknown entries, ordered by content kind and
// then by id, without duplicates.
func (r StartAtHigherLevelContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (StartAtHigherLevelStarted) isStartAtHigherLevelResult()        {}
func (StartAtHigherLevelRejected) isStartAtHigherLevelResult()       {}
func (StartAtHigherLevelContentMissing) isStartAtHigherLevelResult() {}

// StartAtHigherLevel is the use case that builds a character at a starting level
// above 1: it looks up the classes, species and background, validates the plan
// with a dry run of HigherLevelStart, rolls one Hit Point Die for each RollHitDie
// step and emits the srd5e.character.* signals. It holds no rules logic. Create
// it with NewStartAtHigherLevel; the zero value is not usable. It is safe for
// concurrent use only if its source, roller, telemetry and clock are.
type StartAtHigherLevel struct {
	source ContentSource
	roller Roller
	sig    startSignals
}

// NewStartAtHigherLevel returns the use case. Options: WithSource and WithRoller
// (both required), WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput when the source or the roller is missing.
func NewStartAtHigherLevel(opts ...Option) (StartAtHigherLevel, error) {
	o := newOptions(opts)
	if o.source == nil || o.roller == nil {
		return StartAtHigherLevel{}, invalidInput("start at a higher level without a content source and a dice roller")
	}
	return StartAtHigherLevel{source: o.source, roller: o.roller, sig: newStartSignals(o)}, nil
}

// Start builds the character of cmd. The order of the checks is fixed: the plan
// length (before any lookup), missing content, invalid starting arguments, then
// the steps in order, then the subclasses owed. Every rule rejection and the
// missing-content result are found before any die is rolled; the dice are then
// rolled in step order, one die of the class's size for each RollHitDie step,
// and a TakeFixedValue step never calls the roller.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or
// signal, for the zero use case, a zero id or scores or starting level, an
// invalid class, species, background or subclass id, or a step with a nil
// HitPointChoice, and ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled. It returns a *srderr.PortError (error type "content_error" or
// "dice_roller_error") with a nil result when the source or the roller fails,
// panics or breaks its contract (a roll outside the die included); the failure is
// recorded once and never logged, no character is built, and the rolls made so
// far stay made.
func (u StartAtHigherLevel) Start(ctx context.Context, cmd StartAtHigherLevelCommand) (StartAtHigherLevelResult, error) {
	if u.source == nil || u.roller == nil {
		return nil, invalidInput("zero StartAtHigherLevel; use NewStartAtHigherLevel")
	}
	if err := checkStartCommand(cmd); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanAdvancementStartAtHigherLevel, ValueOperationStartAtHigherLevel)
	defer run.end(ctx)
	u.sig.recordSteps(ctx, len(cmd.Steps))

	rules := HigherLevelStart{}
	if rej := rules.CheckPlanLength(cmd.StartingLevel, len(cmd.Steps)); rej != nil {
		return u.rejected(ctx, run, rej), nil
	}
	loaded, missing, err := u.load(ctx, cmd)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return StartAtHigherLevelContentMissing{missing: missing}, nil
	}
	initial := loaded[cmd.ClassID]
	created, err := New(NewParams{
		ID: cmd.CharacterID, Class: cmd.ClassID, HitPointDie: initial.HitPointDie(), Species: cmd.SpeciesID, Background: cmd.BackgroundID,
		AbilityScores: cmd.AbilityScores, ExperiencePoints: rules.StartingExperience(cmd.StartingLevel),
	})
	if err != nil {
		return u.rejected(ctx, run, creationInvalidFrom(err)), nil
	}
	plan, rolled, err := planSteps(cmd.Steps, loaded, nil)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	outcome, err := rules.Advance(created.Character(), cmd.StartingLevel, loaded, plan)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if refused, ok := outcome.(HigherLevelStartRefused); ok {
		return u.rejected(ctx, run, refused.Reason), nil
	}
	if len(rolled) > 0 {
		gains, err := u.roll(rolled, cmd.Steps, loaded)
		if err != nil {
			run.portFailure(ctx, err, srderr.ErrorType(err))
			return nil, err
		}
		if plan, _, err = planSteps(cmd.Steps, loaded, gains); err == nil {
			outcome, err = rules.Advance(created.Character(), cmd.StartingLevel, loaded, plan)
		}
		if err != nil { // unreachable: the plan was accepted and the gains fit the classes
			run.commandError(err)
			return nil, err
		}
	}
	built, ok := outcome.(HigherLevelStartBuilt)
	if !ok { // unreachable: no rule depends on the Hit Point gain
		err := invalidInput("higher-level start refused after the dice were rolled")
		run.commandError(err)
		return nil, err
	}
	run.finish(ctx, ValueResultStarted, "")
	return StartAtHigherLevelStarted{
		character: built.character, events: slices.Concat(created.events, built.events), consumed: built.consumed,
	}, nil
}

// rejected records a rule rejection once and builds the result.
func (u StartAtHigherLevel) rejected(ctx context.Context, run operationRun, reason HigherLevelStartRejection) StartAtHigherLevelResult {
	run.finish(ctx, ValueResultRejected, higherLevelStartRejectionValue(reason))
	return StartAtHigherLevelRejected{Reason: reason}
}

// checkStartCommand returns an error matching srderr.ErrInvalidInput for a
// command that cannot be evaluated at all.
func checkStartCommand(cmd StartAtHigherLevelCommand) error {
	if cmd.CharacterID.IsZero() || !cmd.ClassID.Valid() || !cmd.SpeciesID.Valid() || !cmd.BackgroundID.Valid() ||
		cmd.AbilityScores.IsZero() || cmd.StartingLevel.IsZero() {
		return invalidInput("start at a higher level: zero id, scores or level, or an invalid class, species or background id")
	}
	for _, s := range cmd.Steps {
		switch s.HitPointChoice.(type) {
		case RollHitDie, TakeFixedValue:
		default:
			return invalidInput("start at a higher level: a step without a Hit Point choice")
		}
		if !s.ClassID.Valid() || (s.SubclassID != "" && !s.SubclassID.Valid()) {
			return invalidInput("start at a higher level: a step with an invalid class or subclass id")
		}
	}
	return nil
}

// load looks up the initial class, every other class of the steps, the species
// and the background, all before reporting what is missing, inside the
// load_content span. The missing entries are ordered by kind and then by id,
// without duplicates.
func (u StartAtHigherLevel) load(ctx context.Context, cmd StartAtHigherLevelCommand) (map[content.ClassID]content.ClassDefinition, []content.MissingContent, error) {
	ctx, span := u.sig.tel.Tracer.Start(ctx, SpanAdvancementLoadContent)
	defer span.End()
	keys := []content.Key{cmd.ClassID.Key(), cmd.SpeciesID.Key(), cmd.BackgroundID.Key()}
	for _, s := range cmd.Steps {
		keys = append(keys, s.ClassID.Key())
	}
	classes := map[content.ClassID]content.ClassDefinition{}
	var missing []content.MissingContent
	seen := map[content.Key]bool{}
	for _, k := range keys {
		if seen[k] {
			continue
		}
		seen[k] = true
		entry, found, err := lookupEntry(ctx, u.source, k)
		if err == nil && found {
			if k.Kind() == content.KindCharacterClass {
				def, ok := entry.(content.ClassDefinition)
				if !ok {
					err = srderr.ContractViolation(content.Port, "lookup", "a class key answered with another entry type")
				}
				classes[content.ClassID(k.ID().String())] = def
			}
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(telemetry.StatusError)
			span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
			return nil, nil, err
		}
		if !found {
			missing = append(missing, content.NewMissingContent(k.Kind(), k.ID()))
		}
	}
	slices.SortFunc(missing, func(a, b content.MissingContent) int {
		return cmp.Or(cmp.Compare(a.Kind(), b.Kind()), strings.Compare(a.ID().String(), b.ID().String()))
	})
	outcome := telemetry.OutcomeSuccess
	if len(missing) > 0 {
		outcome = telemetry.OutcomeFailure
		span.SetStatus(telemetry.StatusError)
	}
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, outcome))
	return classes, missing, nil
}

// plan turns the command steps into HigherLevelSteps (FR-119): the
// AdvancementClass comes from the looked-up class and the Hit Point gain from the
// choice, the fixed value for TakeFixedValue and for a RollHitDie step whose
// rolled value is not yet in gains, the rolled value otherwise. It returns the
// indexes of the RollHitDie steps.
func planSteps(steps []StartingLevelStep, classes map[content.ClassID]content.ClassDefinition, gains map[int]int) ([]HigherLevelStep, []int, error) {
	out := make([]HigherLevelStep, 0, len(steps))
	var rolled []int
	for i, s := range steps {
		def := classes[s.ClassID]
		class, err := NewAdvancementClass(s.ClassID, def.HitPointDie(), def.PrimaryAbility())
		if err != nil {
			return nil, nil, err
		}
		var gain HitPointGain
		if _, isRoll := s.HitPointChoice.(RollHitDie); isRoll {
			rolled = append(rolled, i)
		}
		if value, ok := gains[i]; ok {
			gain, err = NewRolledHitPointGain(value, def.HitPointDie())
		} else {
			gain, err = NewFixedHitPointGain(def.HitPointDie().Faces()/2 + 1)
		}
		if err != nil {
			return nil, nil, err
		}
		step, err := NewHigherLevelStep(class, gain, s.SubclassID, s.AbilityScoreIncrease)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, step)
	}
	return out, rolled, nil
}

// roll rolls one Hit Point Die of the step's class for each of the rolled steps,
// in step order, and returns the values by step index. A failure comes back as a
// *srderr.PortError.
func (u StartAtHigherLevel) roll(rolled []int, steps []StartingLevelStep, classes map[content.ClassID]content.ClassDefinition) (map[int]int, error) {
	gains := map[int]int{}
	for _, i := range rolled {
		die, err := dice.NewDie(classes[steps[i].ClassID].HitPointDie().Faces())
		if err != nil { // unreachable: the class die was checked
			return nil, err
		}
		expr, err := dice.NewExpression(1, die, 0)
		if err != nil { // unreachable: one die, no modifier
			return nil, err
		}
		res, err := dice.Roll(u.roller, expr)
		if err != nil {
			return nil, err
		}
		gains[i] = res.Total()
	}
	return gains, nil
}
