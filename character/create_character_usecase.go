package character

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// CreateCharacterResult is the closed set of outcomes of CreateCharacter.Run.
// Variants: CreateCharacterSucceeded, CreateCharacterRejected,
// CreateCharacterContentMissing.
type CreateCharacterResult interface{ isCreateCharacterResult() }

// CreateCharacterSucceeded is a character created: the Character at the starting
// level, the CreationRecord of the choices the aggregate does not store and the
// events in order: Created first; then for each step of a plan LevelGained,
// SubclassChosen if any, AbilityScoreChanged and HitPointMaximumChanged; then the
// FeatTaken events of the background's feat and of the species-granted feat. Only
// Run builds one; the zero value holds no character.
type CreateCharacterSucceeded struct {
	character Character
	record    CreationRecord
	events    []Event
}

// Character returns the new Character.
func (r CreateCharacterSucceeded) Character() Character { return r.character }

// Record returns the CreationRecord.
func (r CreateCharacterSucceeded) Record() CreationRecord { return r.record }

// Events returns a new slice of the events, in order.
func (r CreateCharacterSucceeded) Events() []Event { return slices.Clone(r.events) }

// CreateCharacterRejected is a command a rule refused: every problem of the first
// phase that found any, ordered by step and kind. No die was rolled and no
// character built. Build it with NewCreateCharacterRejected; the zero value lists
// nothing and is invalid.
type CreateCharacterRejected struct{ problems []CreateCharacterProblem }

// NewCreateCharacterRejected returns the rejection holding problems, copied. It
// returns an error matching srderr.ErrInvalidInput for an empty list. It never
// panics.
func NewCreateCharacterRejected(problems []CreateCharacterProblem) (CreateCharacterRejected, error) {
	if len(problems) == 0 {
		return CreateCharacterRejected{}, invalidInput("a creation rejection without problems")
	}
	return CreateCharacterRejected{problems: slices.Clone(problems)}, nil
}

// Problems returns a new slice of the problems.
func (r CreateCharacterRejected) Problems() []CreateCharacterProblem { return slices.Clone(r.problems) }

// CreateCharacterContentMissing is the outcome when the source lacks an entry the
// command needs. It names at least one entry. Only Run builds one.
type CreateCharacterContentMissing struct{ missing []content.MissingContent }

// Missing returns a new slice of the absent entries, ordered by content kind and
// then by id, without duplicates.
func (r CreateCharacterContentMissing) Missing() []content.MissingContent {
	return slices.Clone(r.missing)
}

func (CreateCharacterSucceeded) isCreateCharacterResult()      {}
func (CreateCharacterRejected) isCreateCharacterResult()       {}
func (CreateCharacterContentMissing) isCreateCharacterResult() {}

// CreateCharacter is the use case that creates a character, following the five steps
// of "Create Your Character" (SRD 5.2.1, p. 19). It validates every choice, looks up
// the content, rolls the dice a start asks for, builds the Character and its record
// with the pure domain services and emits the srd5e.character.* signals. It calls no
// other use case and holds no rules logic. Create it with NewCreateCharacter; the
// zero value is not usable. It is safe for concurrent use only if its source,
// roller, telemetry and clock are.
type CreateCharacter struct {
	source ContentSource
	roller Roller
	sig    operationSignals
}

// NewCreateCharacter returns the use case. Options: WithSource and WithRoller (both
// required), WithTelemetry, WithClock. It returns an error matching
// srderr.ErrInvalidInput when the source or the roller is missing.
func NewCreateCharacter(opts ...Option) (CreateCharacter, error) {
	o := newOptions(opts)
	if o.source == nil || o.roller == nil {
		return CreateCharacter{}, invalidInput("create a character without a content source and a dice roller")
	}
	return CreateCharacter{source: o.source, roller: o.roller, sig: newOperationSignals(o)}, nil
}

// Run creates the character of cmd. The phases run in a fixed order and a phase that
// finds problems ends the flow: phase 1 checks the command with no lookup; phase 2
// looks up all content; phase 3 checks the choices that need content and dry-runs the
// character and both feats with fixed Hit Point gains; phase 4 rolls the dice (the
// allowance die, the trinket die, then one Hit Point Die for each RollHitDie step);
// phase 5 builds the character and record. No die is rolled for a command that will
// be rejected, and a command that asks for no roll never calls the roller.
//
// It returns an error matching srderr.ErrInvalidInput, before any lookup or signal,
// for the zero use case or a command with a zero or invalid required field, and
// ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns a
// *srderr.PortError (error type "content_error" or "dice_roller_error") with a nil
// result when the source or the roller fails, panics or breaks its contract; the
// failure is recorded once and never logged, and the rolls made so far stay made.
// A choice the rules refuse is a CreateCharacterRejected result.
func (u CreateCharacter) Run(ctx context.Context, cmd CreateCharacterCommand) (CreateCharacterResult, error) {
	if u.source == nil || u.roller == nil {
		return nil, invalidInput("zero CreateCharacter; use NewCreateCharacter")
	}
	norm, err := cmd.normalized()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	method := telemetry.String(AttrAbilityScoreMethod, abilityScoreMethodValue(norm.AbilityScores.Method()))
	ctx, run := u.sig.start(ctx, SpanCreate, ValueOperationCreate, method)
	defer run.end(ctx)

	if ps := commandProblems(norm); len(ps) > 0 {
		return u.rejected(ctx, run, ps), nil
	}
	in, missing, err := u.load(ctx, norm)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	if len(missing) > 0 {
		run.finish(ctx, ValueResultContentMissing, "")
		return CreateCharacterContentMissing{missing: missing}, nil
	}
	ps, err := in.contentProblems()
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	if len(ps) > 0 {
		return u.rejected(ctx, run, ps), nil
	}
	rolls, err := u.roll(in)
	if err != nil {
		run.portFailure(ctx, err, srderr.ErrorType(err))
		return nil, err
	}
	built, record, err := in.assemble(rolls)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	run.finish(ctx, ValueResultCreated, "")
	return CreateCharacterSucceeded{character: built.character, record: record, events: built.events}, nil
}

// rejected records a rejection once, with the kind of its first problem, and builds
// the result.
func (u CreateCharacter) rejected(ctx context.Context, run operationRun, problems []CreateCharacterProblem) CreateCharacterResult {
	run.finish(ctx, ValueResultRejected, problems[0].Kind().String())
	return CreateCharacterRejected{problems: problems}
}

// load looks up everything the command needs inside the load_content span: the
// class, background and species, every class of the plan, the tools (and the family
// of a variant), the feats and the spells of their choices, and the content the class
// choices name. The missing entries are ordered by kind and then by id, without
// duplicates.
func (u CreateCharacter) load(ctx context.Context, cmd CreateCharacterCommand) (creationInput, []content.MissingContent, error) {
	ctx, span := u.sig.tel.Tracer.Start(ctx, SpanLoadContent)
	defer span.End()
	level, _ := NewLevel(cmd.Options.StartingLevel()) // checked by normalized
	in := creationInput{
		cmd: cmd, level: level, tools: map[content.ItemID]content.Tool{},
		classes: map[content.ClassID]content.ClassDefinition{}, spells: map[content.SpellID]content.Spell{},
	}
	var missing []content.MissingContent
	queue := cmd.keys()
	seen := map[content.Key]bool{}
	for i := 0; i < len(queue); i++ {
		k := queue[i]
		if seen[k] {
			continue
		}
		seen[k] = true
		entry, found, err := lookupEntry(ctx, u.source, k)
		var more []content.Key
		if err == nil && found {
			var ok bool
			if more, ok = in.admit(k, entry); !ok {
				if k.Kind() == content.KindEquipment { // an equipment entry that is not a tool counts as not found
					found = false
				} else {
					err = srderr.ContractViolation(content.Port, "lookup", "a key answered with an entry of another type")
				}
			}
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(telemetry.StatusError)
			span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
			return creationInput{}, nil, err
		}
		if !found {
			missing = append(missing, content.NewMissingContent(k.Kind(), k.ID()))
			continue
		}
		queue = append(queue, more...)
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
	return in, missing, nil
}

// admit files a found entry under the key it was looked up by. It returns the keys
// the entry makes necessary (a background's feat, a tool variant's family) and false
// when the entry is not of the type its key's kind promises.
func (in *creationInput) admit(k content.Key, entry content.Entry) ([]content.Key, bool) {
	switch k.Kind() {
	case content.KindCharacterClass:
		def, ok := entry.(content.ClassDefinition)
		if ok {
			in.classes[def.ID()] = def
			if def.ID() == in.cmd.ClassID {
				in.class = def
			}
		}
		return nil, ok
	case content.KindBackground:
		b, ok := entry.(content.Background)
		if ok {
			in.background = b
			return []content.Key{b.Feat().FeatID().Key()}, true
		}
		return nil, false
	case content.KindSpecies:
		s, ok := entry.(content.Species)
		if ok {
			in.species = s
		}
		return nil, ok
	case content.KindFeat:
		f, ok := entry.(content.Feat)
		if !ok {
			return nil, false
		}
		if id, has := in.cmd.SpeciesGrants.Feat(); has && id == f.ID() {
			in.speciesFeat = &f
		}
		if in.background.Feat().FeatID() == f.ID() {
			in.bgFeat = f
		}
		return nil, true
	case content.KindSpell:
		s, ok := entry.(content.Spell)
		if ok {
			in.spells[s.ID()] = s
		}
		return nil, ok
	case content.KindEquipment:
		t, ok := entry.(content.Tool)
		if !ok {
			return nil, false
		}
		in.tools[t.ID()] = t
		if v, isVariant := t.(content.ToolVariant); isVariant {
			return []content.Key{v.VariantOf().Key()}, true
		}
		return nil, true
	case content.KindMagicItem, content.KindMonster: // nothing to file; a class choice may name them
		return nil, true
	}
	return nil, true
}

// roll rolls the dice of phase 4 in order: the allowance die, the trinket die, then
// one Hit Point Die of the step's class for each RollHitDie step, in step order.
func (u CreateCharacter) roll(in creationInput) (creationRolls, error) {
	var r creationRolls
	if _, expr := in.allowance(); !expr.IsZero() {
		res, err := dice.Roll(u.roller, expr)
		if err != nil {
			return creationRolls{}, err
		}
		r.allowance = res.DiceTotal()
	}
	if _, ok := in.cmd.Trinket.(RollForTrinket); ok {
		res, err := dice.Roll(u.roller, dice.MustExpression(1, dice.D100(), 0))
		if err != nil {
			return creationRolls{}, err
		}
		if r.trinket, err = NewTrinketRoll(res.Total()); err != nil {
			return creationRolls{}, err
		}
	}
	_, rolled, err := planSteps(in.cmd.Plan, in.classes, nil)
	if err != nil {
		return creationRolls{}, err
	}
	r.gains = map[int]int{}
	for _, i := range rolled {
		die, err := dice.NewDie(in.classes[in.cmd.Plan[i].ClassID].HitPointDie().Faces())
		if err != nil { // unreachable: the class die was checked
			return creationRolls{}, err
		}
		expr, err := dice.NewExpression(1, die, 0)
		if err != nil { // unreachable: one die, no modifier
			return creationRolls{}, err
		}
		res, err := dice.Roll(u.roller, expr)
		if err != nil {
			return creationRolls{}, err
		}
		r.gains[i] = res.Total()
	}
	return r, nil
}
