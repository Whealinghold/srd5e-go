package combat

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// EffectPotency is how potent an effect is, for Combining Spell Effects (SRD 5.2.1, p. 106). The
// SRD gives only examples ("the highest bonus"), so which number is more potent is the template
// author's (FR-019). Zero is the default, so equal-potency castings resolve to the most recent.
type EffectPotency int

// EffectCombiningSubject says what two castings of one spell must share to combine (FR-015).
// The zero value is CombiningOnTarget.
type EffectCombiningSubject int

// The combining subjects.
const (
	// CombiningOnTarget combines castings on the same subject (Bless, Bane).
	CombiningOnTarget EffectCombiningSubject = iota
	// CombiningOnCaster combines castings by the same caster (Hex, Hunter's Mark).
	CombiningOnCaster
)

// Valid reports whether s is one of the two combining subjects.
func (s EffectCombiningSubject) Valid() bool { return s == CombiningOnTarget || s == CombiningOnCaster }

// String returns "target" or "caster", or "unknown".
func (s EffectCombiningSubject) String() string {
	switch s {
	case CombiningOnTarget:
		return "target"
	case CombiningOnCaster:
		return "caster"
	}
	return "unknown"
}

// EffectTurnOwner says whose turn a trigger waits for. The zero value is invalid.
type EffectTurnOwner int

// The turn owners of a trigger.
const (
	// EffectTurnOfCaster is a turn of the caster.
	EffectTurnOfCaster EffectTurnOwner = iota + 1
	// EffectTurnOfSubject is a turn of the creature the effect is on.
	EffectTurnOfSubject
	// EffectTurnOfCreatureInArea is a turn of a creature the host lists as in the effect's area.
	EffectTurnOfCreatureInArea
)

// Valid reports whether o is one of the three owners.
func (o EffectTurnOwner) Valid() bool {
	return o >= EffectTurnOfCaster && o <= EffectTurnOfCreatureInArea
}

// String returns "caster", "subject" or "creature_in_area", or "unknown".
func (o EffectTurnOwner) String() string {
	switch o {
	case EffectTurnOfCaster:
		return "caster"
	case EffectTurnOfSubject:
		return "subject"
	case EffectTurnOfCreatureInArea:
		return "creature_in_area"
	}
	return "unknown"
}

// EffectDamageAudience says who an effect's damage is for. The zero value is invalid.
type EffectDamageAudience int

// The audiences of effect damage.
const (
	// EffectAudienceSubject is the creature the effect is on.
	EffectAudienceSubject EffectDamageAudience = iota + 1
	// EffectAudienceTurnTaker is the combatant of the tick; valid only in a trigger.
	EffectAudienceTurnTaker
	// EffectAudienceAreaMembers is the creatures the host lists; valid only when the effect ends.
	EffectAudienceAreaMembers
)

// Valid reports whether a is one of the three audiences.
func (a EffectDamageAudience) Valid() bool {
	return a >= EffectAudienceSubject && a <= EffectAudienceAreaMembers
}

// String returns "subject", "turn_taker" or "area_members", or "unknown".
func (a EffectDamageAudience) String() string {
	switch a {
	case EffectAudienceSubject:
		return "subject"
	case EffectAudienceTurnTaker:
		return "turn_taker"
	case EffectAudienceAreaMembers:
		return "area_members"
	}
	return "unknown"
}

// EffectSaveSuccess says what a successful saving throw does to effect damage. The zero value is
// invalid.
type EffectSaveSuccess int

// What a success does.
const (
	// SaveSuccessNoDamage is no damage on a success.
	SaveSuccessNoDamage EffectSaveSuccess = iota + 1
	// SaveSuccessHalfDamage is half damage on a success (SRD 5.2.1, "Half Damage", p. 16).
	SaveSuccessHalfDamage
)

// Valid reports whether s is one of the two.
func (s EffectSaveSuccess) Valid() bool {
	return s == SaveSuccessNoDamage || s == SaveSuccessHalfDamage
}

// String returns "none" or "half", or "unknown".
func (s EffectSaveSuccess) String() string {
	switch s {
	case SaveSuccessNoDamage:
		return "none"
	case SaveSuccessHalfDamage:
		return "half"
	}
	return "unknown"
}

// EffectDamageSave is the saving throw that can reduce effect damage. The zero value is the
// absence of a save. Build one with NewEffectDamageSave.
type EffectDamageSave struct {
	ability   ability.Ability
	dc        d20.DifficultyClass
	onSuccess EffectSaveSuccess
}

// NewEffectDamageSave returns the save of ability against dc. It returns an error matching
// srderr.ErrInvalidInput for an invalid ability or onSuccess. It never panics.
func NewEffectDamageSave(a ability.Ability, dc d20.DifficultyClass, onSuccess EffectSaveSuccess) (EffectDamageSave, error) {
	if !a.Valid() || !onSuccess.Valid() {
		return EffectDamageSave{}, invalidInput("effect damage save needs an ability and what a success does")
	}
	return EffectDamageSave{a, dc, onSuccess}, nil
}

// IsZero reports whether s is the absence of a save.
func (s EffectDamageSave) IsZero() bool { return s == EffectDamageSave{} }

// Ability returns the ability of the save.
func (s EffectDamageSave) Ability() ability.Ability { return s.ability }

// DifficultyClass returns the DC, fixed when the effect started (FR-030).
func (s EffectDamageSave) DifficultyClass() d20.DifficultyClass { return s.dc }

// OnSuccess returns what a success does.
func (s EffectDamageSave) OnSuccess() EffectSaveSuccess { return s.onSuccess }

// EffectAction is what a trigger, a gate or an ending does. Variants: RepeatSave, DealDamage,
// GrowStoredDamage, DealStoredDamage.
type EffectAction interface{ isEffectAction() }

// RepeatSave is a saving throw of the creature the effect is on; a success ends the effect (SRD
// 5.2.1, "Hold Person", p. 141). The DC is fixed in the template (FR-030). Build it with
// NewRepeatSave.
type RepeatSave struct {
	ability ability.Ability
	dc      d20.DifficultyClass
}

// NewRepeatSave returns the repeated save of ability against dc. It returns an error matching
// srderr.ErrInvalidInput for an invalid ability. It never panics.
func NewRepeatSave(a ability.Ability, dc d20.DifficultyClass) (RepeatSave, error) {
	if !a.Valid() {
		return RepeatSave{}, invalidInput("repeat save needs an ability")
	}
	return RepeatSave{a, dc}, nil
}

// Ability returns the ability of the save.
func (s RepeatSave) Ability() ability.Ability { return s.ability }

// DifficultyClass returns the DC.
func (s RepeatSave) DifficultyClass() d20.DifficultyClass { return s.dc }

// DealDamage plans damage of an amount and a type, with an optional save (FR-035). Build it with
// NewDealDamage.
type DealDamage struct {
	amount   dice.Expression
	typ      content.DamageType
	save     EffectDamageSave
	audience EffectDamageAudience
}

// NewDealDamage returns the damage action. It returns an error matching srderr.ErrInvalidInput
// for a zero amount, an invalid Damage Type or an invalid audience. It never panics.
func NewDealDamage(amount dice.Expression, t content.DamageType, audience EffectDamageAudience) (DealDamage, error) {
	if amount.IsZero() || !t.Valid() || !audience.Valid() {
		return DealDamage{}, invalidInput("deal damage needs an amount, a Damage Type and an audience")
	}
	return DealDamage{amount: amount, typ: t, audience: audience}, nil
}

// WithSave returns the action with a save that can reduce the damage.
func (d DealDamage) WithSave(s EffectDamageSave) DealDamage { d.save = s; return d }

// Amount returns the damage to roll.
func (d DealDamage) Amount() dice.Expression { return d.amount }

// DamageType returns the Damage Type.
func (d DealDamage) DamageType() content.DamageType { return d.typ }

// Save returns the save and whether there is one.
func (d DealDamage) Save() (EffectDamageSave, bool) { return d.save, !d.save.IsZero() }

// Audience returns who the damage is for.
func (d DealDamage) Audience() EffectDamageAudience { return d.audience }

// GrowStoredDamage adds dice to the damage an effect stores (SRD 5.2.1, "Delayed Blast Fireball",
// p. 122). Build it with NewGrowStoredDamage.
type GrowStoredDamage struct{ extra dice.Expression }

// NewGrowStoredDamage returns the growth of extra. It returns an error matching
// srderr.ErrInvalidInput for a zero expression. It never panics.
func NewGrowStoredDamage(extra dice.Expression) (GrowStoredDamage, error) {
	if extra.IsZero() {
		return GrowStoredDamage{}, invalidInput("stored damage growth needs dice")
	}
	return GrowStoredDamage{extra}, nil
}

// Extra returns the dice added each time.
func (g GrowStoredDamage) Extra() dice.Expression { return g.extra }

// DealStoredDamage plans the damage the effect has stored. Build it with NewDealStoredDamage.
type DealStoredDamage struct {
	typ      content.DamageType
	save     EffectDamageSave
	audience EffectDamageAudience
}

// NewDealStoredDamage returns the action. It returns an error matching srderr.ErrInvalidInput for
// an invalid Damage Type or audience. It never panics.
func NewDealStoredDamage(t content.DamageType, audience EffectDamageAudience) (DealStoredDamage, error) {
	if !t.Valid() || !audience.Valid() {
		return DealStoredDamage{}, invalidInput("deal stored damage needs a Damage Type and an audience")
	}
	return DealStoredDamage{typ: t, audience: audience}, nil
}

// WithSave returns the action with a save that can reduce the damage.
func (d DealStoredDamage) WithSave(s EffectDamageSave) DealStoredDamage { d.save = s; return d }

// DamageType returns the Damage Type.
func (d DealStoredDamage) DamageType() content.DamageType { return d.typ }

// Save returns the save and whether there is one.
func (d DealStoredDamage) Save() (EffectDamageSave, bool) { return d.save, !d.save.IsZero() }

// Audience returns who the damage is for.
func (d DealStoredDamage) Audience() EffectDamageAudience { return d.audience }

func (RepeatSave) isEffectAction()       {}
func (DealDamage) isEffectAction()       {}
func (GrowStoredDamage) isEffectAction() {}
func (DealStoredDamage) isEffectAction() {}

// EffectTrigger runs actions at a boundary of a turn while a stage lasts (FR-027). Build it with
// NewEffectTrigger.
type EffectTrigger struct {
	whose    EffectTurnOwner
	boundary content.TurnBoundary
	actions  []EffectAction
}

// NewEffectTrigger returns the trigger of actions at the boundary of a turn of whose. It returns an
// error matching srderr.ErrInvalidInput for an invalid owner or boundary, no action or a nil
// action. It copies actions. It never panics.
func NewEffectTrigger(whose EffectTurnOwner, boundary content.TurnBoundary, actions ...EffectAction) (EffectTrigger, error) {
	if !whose.Valid() || !boundary.Valid() || len(actions) == 0 || slices.Contains(actions, nil) {
		return EffectTrigger{}, invalidInput("trigger needs an owner, a boundary and actions")
	}
	return EffectTrigger{whose, boundary, slices.Clone(actions)}, nil
}

// Whose returns whose turn the trigger waits for.
func (t EffectTrigger) Whose() EffectTurnOwner { return t.whose }

// Boundary returns the start or the end of that turn.
func (t EffectTrigger) Boundary() content.TurnBoundary { return t.boundary }

// Actions returns the actions, as a new slice.
func (t EffectTrigger) Actions() []EffectAction { return slices.Clone(t.actions) }

// Equal reports whether t and o are the same trigger.
func (t EffectTrigger) Equal(o EffectTrigger) bool {
	return t.whose == o.whose && t.boundary == o.boundary && slices.Equal(t.actions, o.actions)
}

// EffectModelProblem is one reason a stage or template was refused. The zero value is not a
// problem.
type EffectModelProblem int

// The problems of FR-009.
const (
	// EffectModelNoStages: a template has no stage.
	EffectModelNoStages EffectModelProblem = iota + 1
	// EffectModelStageDurationInvalid: a stage has no valid duration.
	EffectModelStageDurationInvalid
	// EffectModelOpenEndedStageNotLast: a stage that never ends by itself is not the last.
	EffectModelOpenEndedStageNotLast
	// EffectModelGatedLastStage: the last stage has a gate.
	EffectModelGatedLastStage
	// EffectModelRemainderWithoutOverall: a remainder stage but no overall clock.
	EffectModelRemainderWithoutOverall
	// EffectModelOverallInvalid: the overall clock is not rounds or indefinite.
	EffectModelOverallInvalid
	// EffectModelConditionInvalid: a held condition is not a condition, or is Exhaustion.
	EffectModelConditionInvalid
	// EffectModelGateNotRepeatSave: a gate is not a RepeatSave.
	EffectModelGateNotRepeatSave
	// EffectModelActionInvalid: an action is nil, or not allowed where it is.
	EffectModelActionInvalid
	// EffectModelAudienceInvalid: an audience is not valid where it is used.
	EffectModelAudienceInvalid
	// EffectModelStoredDamageMissing: an action uses stored damage the template does not set.
	EffectModelStoredDamageMissing
	// EffectModelStoredDamageDieMismatch: a growth uses another die than the stored damage.
	EffectModelStoredDamageDieMismatch
	// EffectModelCombiningSubjectInvalid: the combining subject is not one of the two.
	EffectModelCombiningSubjectInvalid
)

// EffectModelError is one problem of a stage or a template as an error. It matches
// srderr.ErrInvalidInput with errors.Is, and another *EffectModelError with the same Problem. It
// carries no display text. Construct it with a keyed literal.
type EffectModelError struct{ Problem EffectModelProblem }

// Error renders the error. The text is not a contract.
func (e *EffectModelError) Error() string { return "combat: invalid effect model" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *EffectModelError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *EffectModelError with the same Problem.
func (e *EffectModelError) Is(target error) bool {
	t, ok := target.(*EffectModelError)
	return ok && t != nil && e != nil && *t == *e
}

// EffectStageParams are the parts of an EffectStage. Construct it with a keyed literal.
type EffectStageParams struct {
	// Duration is how long the stage lasts.
	Duration EffectDuration
	// Conditions are the conditions the stage holds on its creature (never Exhaustion).
	Conditions []content.ConditionKind
	// Triggers run at turn boundaries while the stage lasts.
	Triggers []EffectTrigger
	// EndsOnDamage ends the effect when its creature takes damage (FR-033).
	EndsOnDamage bool
	// Gate, a RepeatSave or nil, runs when the stage's duration ends (FR-034).
	Gate EffectAction
	// OnEnd runs once when the effect ends in this stage (FR-038): DealDamage or DealStoredDamage.
	OnEnd []EffectAction
}

// EffectStage is one stage of an effect's lifetime (FR-009). Build it with NewEffectStage; the zero
// value is invalid. Immutable.
type EffectStage struct {
	duration     EffectDuration
	conditions   []content.ConditionKind
	triggers     []EffectTrigger
	endsOnDamage bool
	gate         EffectAction
	onEnd        []EffectAction
}

// NewEffectStage returns the stage of p, copying its slices. It returns every problem found, joined
// as *EffectModelError values matching srderr.ErrInvalidInput: no valid duration, an invalid
// condition, a gate that is not a RepeatSave, an action that is nil or not allowed (RepeatSave and
// GrowStoredDamage cannot run when an effect ends), an audience not valid where it is (turn taker
// only in a trigger, area members only when the effect ends). It never panics.
func NewEffectStage(p EffectStageParams) (EffectStage, error) {
	var problems []error
	add := func(k EffectModelProblem) { problems = append(problems, &EffectModelError{k}) }
	if p.Duration == nil || !validEffectDuration(p.Duration) {
		add(EffectModelStageDurationInvalid)
	}
	for _, c := range p.Conditions {
		if !c.Valid() || c == content.ConditionExhaustion {
			add(EffectModelConditionInvalid)
			break
		}
	}
	if p.Gate != nil {
		if _, ok := p.Gate.(RepeatSave); !ok {
			add(EffectModelGateNotRepeatSave)
		}
	}
	for _, t := range p.Triggers {
		for _, a := range t.actions {
			if k, bad := triggerActionProblem(a); bad {
				add(k)
			}
		}
	}
	for _, a := range p.OnEnd {
		if k, bad := onEndActionProblem(a); bad {
			add(k)
		}
	}
	if len(problems) > 0 {
		return EffectStage{}, errors.Join(problems...)
	}
	return EffectStage{p.Duration, slices.Clone(p.Conditions), slices.Clone(p.Triggers), p.EndsOnDamage, p.Gate, slices.Clone(p.OnEnd)}, nil
}

// triggerActionProblem reports why a cannot be in a trigger: area members are known only when an
// effect ends.
func triggerActionProblem(a EffectAction) (EffectModelProblem, bool) {
	switch v := a.(type) {
	case RepeatSave, GrowStoredDamage:
		return 0, false
	case DealDamage:
		return audienceProblem(v.audience, EffectAudienceAreaMembers)
	case DealStoredDamage:
		return audienceProblem(v.audience, EffectAudienceAreaMembers)
	}
	return EffectModelActionInvalid, true
}

// onEndActionProblem reports why a cannot run when an effect ends: no turn taker is there.
func onEndActionProblem(a EffectAction) (EffectModelProblem, bool) {
	switch v := a.(type) {
	case DealDamage:
		return audienceProblem(v.audience, EffectAudienceTurnTaker)
	case DealStoredDamage:
		return audienceProblem(v.audience, EffectAudienceTurnTaker)
	}
	return EffectModelActionInvalid, true
}

func audienceProblem(got, forbidden EffectDamageAudience) (EffectModelProblem, bool) {
	return EffectModelAudienceInvalid, got == forbidden
}

// Duration returns how long the stage lasts.
func (s EffectStage) Duration() EffectDuration { return s.duration }

// Conditions returns the conditions the stage holds, as a new slice.
func (s EffectStage) Conditions() []content.ConditionKind { return slices.Clone(s.conditions) }

// Triggers returns the triggers, as a new slice.
func (s EffectStage) Triggers() []EffectTrigger { return slices.Clone(s.triggers) }

// EndsOnDamage reports whether damage to the creature ends the effect.
func (s EffectStage) EndsOnDamage() bool { return s.endsOnDamage }

// Gate returns the gate and whether the stage has one.
func (s EffectStage) Gate() (RepeatSave, bool) {
	g, ok := s.gate.(RepeatSave)
	return g, ok
}

// OnEnd returns the actions that run when the effect ends, as a new slice.
func (s EffectStage) OnEnd() []EffectAction { return slices.Clone(s.onEnd) }

// IsZero reports whether s is the invalid zero stage.
func (s EffectStage) IsZero() bool { return s.duration == nil }

// Equal reports whether s and o are the same stage.
func (s EffectStage) Equal(o EffectStage) bool {
	return s.duration == o.duration && s.endsOnDamage == o.endsOnDamage && s.gate == o.gate &&
		slices.Equal(s.conditions, o.conditions) && slices.EqualFunc(s.triggers, o.triggers, EffectTrigger.Equal) &&
		slices.Equal(s.onEnd, o.onEnd)
}

// EffectTemplateParams are the parts of an EffectTemplate. Construct it with a keyed literal.
type EffectTemplateParams struct {
	// Stages are the stages, in order; at least one.
	Stages []EffectStage
	// Potency is the potency for Combining Spell Effects.
	Potency EffectPotency
	// Overall, EffectLastsRounds, EffectLastsIndefinitely or nil, is the clock of the whole effect (FR-061).
	Overall EffectDuration
	// CombiningSubject is what castings must share to combine; the zero value is CombiningOnTarget.
	CombiningSubject EffectCombiningSubject
	// SurvivesSubjectDeath keeps the effect when its subject dies, so it can be moved (FR-043, FR-060).
	SurvivesSubjectDeath bool
	// StoredDamage is the base of the damage the effect stores; the zero Expression is none.
	StoredDamage dice.Expression
}

// EffectTemplate is the shape of a spell's lasting effect: stages, potency, an optional overall
// clock and stored damage (FR-009). Build it with NewEffectTemplate; the zero value is invalid.
// Immutable. It holds slices, so compare with Equal, not ==.
type EffectTemplate struct {
	stages   []EffectStage
	potency  EffectPotency
	overall  EffectDuration
	combine  EffectCombiningSubject
	survives bool
	stored   dice.Expression
}

// NewEffectTemplate returns the template of p, copying its stages. It returns every problem found,
// joined as *EffectModelError values matching srderr.ErrInvalidInput: no stage, a zero stage, an
// invalid combining subject, an overall clock that is not rounds or indefinite, a stage that never
// ends by itself (indefinite or remainder) before the last, a gate on the last stage, a remainder
// stage without an overall clock, an action that stores or grows damage when the template has no
// stored damage, and a growth of another die than the stored damage. It never panics.
func NewEffectTemplate(p EffectTemplateParams) (EffectTemplate, error) {
	var problems []error
	add := func(k EffectModelProblem) { problems = append(problems, &EffectModelError{k}) }
	if len(p.Stages) == 0 {
		add(EffectModelNoStages)
	}
	if !p.CombiningSubject.Valid() {
		add(EffectModelCombiningSubjectInvalid)
	}
	switch p.Overall.(type) {
	case nil, EffectLastsRounds, EffectLastsIndefinitely:
		if p.Overall != nil && !validEffectDuration(p.Overall) {
			add(EffectModelOverallInvalid)
		}
	default:
		add(EffectModelOverallInvalid)
	}
	for i, s := range p.Stages {
		last := i == len(p.Stages)-1
		if s.IsZero() {
			add(EffectModelStageDurationInvalid)
			continue
		}
		_, gated := s.Gate()
		switch s.duration.(type) {
		case EffectLastsIndefinitely, EffectLastsRemainder:
			if !last {
				add(EffectModelOpenEndedStageNotLast)
			}
		}
		if _, rem := s.duration.(EffectLastsRemainder); rem && p.Overall == nil {
			add(EffectModelRemainderWithoutOverall)
		}
		if gated && last {
			add(EffectModelGatedLastStage)
		}
		for _, a := range storedActions(s) {
			if k, bad := storedProblem(a, p.StoredDamage); bad {
				add(k)
			}
		}
	}
	if len(problems) > 0 {
		return EffectTemplate{}, errors.Join(problems...)
	}
	return EffectTemplate{slices.Clone(p.Stages), p.Potency, p.Overall, p.CombiningSubject, p.SurvivesSubjectDeath, p.StoredDamage}, nil
}

// storedActions returns every action of s that reads or grows stored damage.
func storedActions(s EffectStage) []EffectAction {
	var out []EffectAction
	all := slices.Clone(s.onEnd)
	for _, t := range s.triggers {
		all = append(all, t.actions...)
	}
	for _, a := range all {
		switch a.(type) {
		case GrowStoredDamage, DealStoredDamage:
			out = append(out, a)
		}
	}
	return out
}

// storedProblem reports why the stored-damage action a does not fit the base.
func storedProblem(a EffectAction, base dice.Expression) (EffectModelProblem, bool) {
	if base.IsZero() {
		return EffectModelStoredDamageMissing, true
	}
	if g, ok := a.(GrowStoredDamage); ok && g.extra.Die() != base.Die() {
		return EffectModelStoredDamageDieMismatch, true
	}
	return 0, false
}

// IsZero reports whether t is the invalid zero template.
func (t EffectTemplate) IsZero() bool { return len(t.stages) == 0 }

// Stages returns the stages, as a new slice.
func (t EffectTemplate) Stages() []EffectStage { return slices.Clone(t.stages) }

// Potency returns the potency.
func (t EffectTemplate) Potency() EffectPotency { return t.potency }

// Overall returns the overall clock and whether there is one.
func (t EffectTemplate) Overall() (EffectDuration, bool) { return t.overall, t.overall != nil }

// CombiningSubject returns what castings must share to combine.
func (t EffectTemplate) CombiningSubject() EffectCombiningSubject { return t.combine }

// SurvivesSubjectDeath reports whether the effect outlives its dead subject.
func (t EffectTemplate) SurvivesSubjectDeath() bool { return t.survives }

// StoredDamage returns the base stored damage and whether there is one.
func (t EffectTemplate) StoredDamage() (dice.Expression, bool) { return t.stored, !t.stored.IsZero() }

// Equal reports whether t and o are the same template.
func (t EffectTemplate) Equal(o EffectTemplate) bool {
	return t.potency == o.potency && t.overall == o.overall && t.combine == o.combine && t.survives == o.survives &&
		t.stored == o.stored && slices.EqualFunc(t.stages, o.stages, EffectStage.Equal)
}
