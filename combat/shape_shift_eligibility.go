package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FormIneligibility is one reason a form is not allowed, in the fixed order of the enum.
// The zero value is invalid.
type FormIneligibility int

// The reasons a form is ineligible, in the order they are reported.
const (
	NotABeast FormIneligibility = iota + 1
	ConstructOrUndead
	NotBeastOrHumanoid
	SizeTooLarge
	ChallengeRatingTooHigh
	KindNotSeen
	ObjectIsMagical
	ObjectWornOrCarried
)

// String returns the snake_case reason, or "unknown".
func (r FormIneligibility) String() string {
	return enumName([]string{"not_a_beast", "construct_or_undead", "not_beast_or_humanoid", "size_too_large",
		"challenge_rating_too_high", "kind_not_seen", "object_is_magical", "object_worn_or_carried"}, int(r))
}

// FormEligibility is the closed set of answers of ShapeShiftSpells.Eligibility. Variants:
// FormEligible, FormIneligible.
type FormEligibility interface{ isFormEligibility() }

// FormEligible: the form is allowed.
type FormEligible struct{}

// FormIneligible: the form is not allowed, for every reason that applies in the fixed order of
// FormIneligibility. The list is never empty. It is also a rejection of Begin and ChangeForm.
// Not comparable with ==.
type FormIneligible struct{ reasons []FormIneligibility }

// Reasons returns every violated rule in the fixed order, as a new slice.
func (f FormIneligible) Reasons() []FormIneligibility { return slices.Clone(f.reasons) }

// String returns "form_ineligible".
func (FormIneligible) String() string { return "form_ineligible" }

func (FormEligible) isFormEligibility()   {}
func (FormIneligible) isFormEligibility() {}

// FormQuery is what Eligibility judges: the spell and the facts it needs. Construct it with
// a keyed literal.
type FormQuery struct {
	// Spell is the shape-shifting spell.
	Spell ShapeShiftSpell
	// Mode is the mode of True Polymorph; ignored for the other spells.
	Mode TrueTransformation
	// Terms are the terms of Shapechange; ignored for the other spells.
	Terms ShapechangeTerms
	// Form is the form to judge, judged by the size the host chose.
	Form AssumedForm
	// Rating is the target's rating (Polymorph, True Polymorph creature into creature) or the
	// caster's (Shapechange); unused by Animal Shapes and the object modes.
	Rating TargetRating
	// HasSeenKind is whether the caster has seen the sort of creature (Shapechange).
	HasSeenKind bool
	// Object describes the object of True Polymorph object into creature.
	Object ObjectFacts
	// TargetSize is the size of the creature of True Polymorph creature into object.
	TargetSize content.CreatureSize
}

// Eligibility judges the form of q (SRD 5.2.1, "Polymorph", p. 153; "True Polymorph", p. 170;
// "Shapechange", p. 161; "Animal Shapes", p. 108). A form is eligible when it breaks no rule;
// a Challenge Rating equal to the rating is allowed, a level n compares as the Challenge
// Rating n. Otherwise every violated rule is listed once in the fixed order of
// FormIneligibility.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil result, for an invalid
// spell or mode, a nil or mismatched form (a creature form for an object mode and the
// reverse, an object form for any other spell), a missing rating where one is read, or an
// invalid object or size fact. It never panics.
func (ShapeShiftSpells) Eligibility(q FormQuery) (FormEligibility, error) {
	if err := q.validate(); err != nil {
		return nil, err
	}
	var hit [ObjectWornOrCarried + 1]bool
	size := q.Form.Size()
	cf, _ := q.Form.(CreatureForm)
	tooHigh := func(limit content.ChallengeRating) {
		hit[ChallengeRatingTooHigh] = content.CompareChallengeRating(cf.facts.rating, limit) > 0
	}
	switch q.Spell {
	case PolymorphSpell:
		hit[NotABeast] = cf.facts.kind != content.CreatureTypeBeast
		tooHigh(q.Rating.Challenge())
	case AnimalShapesSpell:
		hit[NotABeast] = cf.facts.kind != content.CreatureTypeBeast
		hit[SizeTooLarge] = size > content.SizeLarge
		tooHigh(content.MustChallengeRating("4"))
	case ShapechangeSpell:
		hit[ConstructOrUndead] = cf.facts.kind == content.CreatureTypeConstruct || cf.facts.kind == content.CreatureTypeUndead
		hit[NotBeastOrHumanoid] = q.Terms == StatBlockShapechange &&
			cf.facts.kind != content.CreatureTypeBeast && cf.facts.kind != content.CreatureTypeHumanoid
		tooHigh(q.Rating.Challenge())
		hit[KindNotSeen] = !q.HasSeenKind
	case TruePolymorphSpell:
		switch q.Mode {
		case CreatureIntoCreature:
			tooHigh(q.Rating.Challenge())
		case ObjectIntoCreature:
			hit[ObjectIsMagical] = q.Object.IsMagical
			hit[ObjectWornOrCarried] = q.Object.IsWornOrCarried
			hit[SizeTooLarge] = size > q.Object.Size
			tooHigh(content.MustChallengeRating("9"))
		case CreatureIntoObject:
			hit[SizeTooLarge] = size > q.TargetSize
		}
	}
	var reasons []FormIneligibility
	for r := NotABeast; r <= ObjectWornOrCarried; r++ {
		if hit[r] {
			reasons = append(reasons, r)
		}
	}
	if len(reasons) == 0 {
		return FormEligible{}, nil
	}
	return FormIneligible{reasons}, nil
}

// validate rejects a query whose parts do not fit the spell.
func (q FormQuery) validate() error {
	if !q.Spell.Valid() || q.Form == nil || !q.Terms.Valid() {
		return invalidInput("a form query needs a spell, terms and a form")
	}
	_, isCreature := q.Form.(CreatureForm)
	wantCreature, needRating := true, true
	switch q.Spell {
	case PolymorphSpell, ShapechangeSpell:
		// A creature form judged against a rating: the defaults.
	case AnimalShapesSpell:
		needRating = false
	case TruePolymorphSpell:
		if !q.Mode.Valid() {
			return invalidInput("True Polymorph needs a mode")
		}
		wantCreature = q.Mode != CreatureIntoObject
		needRating = q.Mode == CreatureIntoCreature
		if q.Mode == ObjectIntoCreature && !q.Object.Size.Valid() {
			return invalidInput("an object needs a size")
		}
		if q.Mode == CreatureIntoObject && !q.TargetSize.Valid() {
			return invalidInput("a creature needs a size")
		}
	}
	if isCreature != wantCreature || (isCreature && q.Form.(CreatureForm).facts.IsZero()) || !q.Form.Size().Valid() {
		return invalidInput("the form does not fit the spell")
	}
	if needRating && !ratingValid(q.Rating) {
		return invalidInput("a rating is needed")
	}
	return nil
}
