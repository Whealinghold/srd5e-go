package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// DruidChoice is a choice a Druid makes, a closed set. Variants: DruidChoicePrimalOrder, DruidChoiceElementalFury,
// DruidChoiceLandType, DruidChoiceKnownForms, DruidChoiceReplaceForm. Cantrips and prepared spells are not a DruidChoice: the
// host calls ChangePreparedSpells.
type DruidChoice interface{ isDruidChoice() }

// DruidChoicePrimalOrder chooses the Primal Order (SRD 5.2.1, "Level 1: Primal Order", p. 42).
type DruidChoicePrimalOrder struct{ Order PrimalOrder }

// DruidChoiceElementalFury chooses the Elemental Fury option (SRD 5.2.1, "Level 7: Elemental Fury", p. 43).
type DruidChoiceElementalFury struct{ Option ElementalFury }

// DruidChoiceLandType chooses the land type of the Circle of the Land: the first time, or after a Long Rest (SRD 5.2.1,
// "Level 3: Circle of the Land Spells", p. 46).
type DruidChoiceLandType struct{ Land LandType }

// DruidChoiceKnownForms adds Beast forms to the known forms (SRD 5.2.1, "Level 2: Wild Shape", p. 42).
type DruidChoiceKnownForms struct{ Forms []BeastFormFacts }

// DruidChoiceReplaceForm replaces the known form Replace by the form With, once after a Long Rest.
type DruidChoiceReplaceForm struct {
	Replace content.MonsterID
	With    BeastFormFacts
}

func (DruidChoicePrimalOrder) isDruidChoice()   {}
func (DruidChoiceElementalFury) isDruidChoice() {}
func (DruidChoiceLandType) isDruidChoice()      {}
func (DruidChoiceKnownForms) isDruidChoice()    {}
func (DruidChoiceReplaceForm) isDruidChoice()   {}

// ApplyDruidChoiceInput is what ApplyDruidChoice reads. Construct it with a keyed literal.
type ApplyDruidChoiceInput struct {
	State  DruidFeatureState
	Tables DruidFeatureTables
	Facts  DruidCharacterFacts
	Choice DruidChoice
}

// ApplyDruidChoiceResult is the outcome of ApplyDruidChoice, a closed set. Variants: ApplyDruidChoiceApplied,
// ApplyDruidChoiceRejected.
type ApplyDruidChoiceResult interface{ isApplyDruidChoiceResult() }

// ApplyDruidChoiceApplied: the choice was applied. Hold State, publish Events.
type ApplyDruidChoiceApplied struct {
	state  DruidFeatureState
	events []Event
}

// State returns the new Druid state.
func (r ApplyDruidChoiceApplied) State() DruidFeatureState { return r.state }

// Events returns the events, as a new slice.
func (r ApplyDruidChoiceApplied) Events() []Event { return append([]Event{}, r.events...) }

// ApplyDruidChoiceRejected: the choice was refused and the state is unchanged; Reason is the first of the D-5 order.
type ApplyDruidChoiceRejected struct{ Reason DruidRejection }

func (ApplyDruidChoiceApplied) isApplyDruidChoiceResult()  {}
func (ApplyDruidChoiceRejected) isApplyDruidChoiceResult() {}

// when returns reason when cond holds and 0 otherwise.
func when(cond bool, reason DruidRejection) DruidRejection {
	if cond {
		return reason
	}
	return 0
}

// ApplyDruidChoice applies a Druid choice (class-druid.md FR-048 to FR-051, FR-075 to FR-078, FR-104, FR-118, FR-121). When
// several reasons hold the rejection is the first of the D-5 order.
//
//   - Primal Order: DruidPrimalOrderAlreadyChosen when one is chosen.
//   - Elemental Fury: DruidFeatureNotGranted below level 7, DruidElementalFuryAlreadyChosen when one is chosen.
//   - Land type: DruidFeatureNotGranted below level 3 or without the Circle of the Land; the first choice is always open and a
//     later one needs the land type window opened by a Long Rest (DruidChangeNotAllowedNow); a choice closes the window.
//   - Known forms: DruidFeatureNotGranted below level 2; each form must be eligible (DruidNotABeast, DruidChallengeRatingTooHigh,
//     DruidFlySpeedNotAllowed), distinct from the others and from the known forms (DruidDuplicateForm), and no more than the
//     level allows in all (DruidTooManyKnownForms). Fewer than allowed is accepted and the rest stay pending.
//   - Replace a form: the known forms window opened by a Long Rest (DruidChangeNotAllowedNow), a known form to replace
//     (DruidFormNotKnown), an eligible form not already known (the eligibility reasons, DruidDuplicateForm), and at most one
//     replacement per Long Rest (DruidTooManyReplacements). The window stays open.
//
// It returns an error matching srderr.ErrInvalidInput for zero facts or tables, a State that breaks an invariant, a nil choice
// or a value outside its set. It never panics.
func ApplyDruidChoice(in ApplyDruidChoiceInput) (ApplyDruidChoiceResult, error) {
	if err := in.State.check(in.Facts, in.Tables); err != nil {
		return nil, err
	}
	level := in.Facts.lv()
	s, id := in.State, in.Facts.id
	var event Event
	var reason DruidRejection
	switch c := in.Choice.(type) {
	case DruidChoicePrimalOrder:
		if !c.Order.Valid() {
			return nil, invalidInput("Primal Order outside the closed set")
		}
		reason = when(s.order != 0, DruidPrimalOrderAlreadyChosen)
		s.order, event = c.Order, PrimalOrderChosen{id: id, order: c.Order}
	case DruidChoiceElementalFury:
		if !c.Option.Valid() {
			return nil, invalidInput("Elemental Fury option outside the closed set")
		}
		reason, _ = lowestDruidRejection(when(level < 7, DruidFeatureNotGranted), when(s.fury != 0, DruidElementalFuryAlreadyChosen))
		s.fury, event = c.Option, ElementalFuryChosen{id: id, option: c.Option}
	case DruidChoiceLandType:
		if !c.Land.Valid() {
			return nil, invalidInput("land type outside the closed set")
		}
		reason, _ = lowestDruidRejection(when(level < 3 || !in.Facts.hasCircle(), DruidFeatureNotGranted),
			when(s.land != 0 && !s.window.has(RestWindowLandTypeChange), DruidChangeNotAllowedNow))
		s.land, s.window, event = c.Land, s.window.without(RestWindowLandTypeChange), LandTypeChosen{id: id, land: c.Land}
	case DruidChoiceKnownForms:
		reason, s = chooseKnownForms(s, in, c)
		event = KnownFormsChosen{id: id, forms: formIDs(c.Forms)}
	case DruidChoiceReplaceForm:
		reason, s = replaceKnownForm(s, in, c)
		event = KnownFormReplaced{id: id, replaced: c.Replace, replacing: c.With.id}
	default:
		return nil, invalidInput("Druid choice is nil or not one of the closed set")
	}
	if reason != 0 {
		return ApplyDruidChoiceRejected{Reason: reason}, nil
	}
	if kf, ok := in.Choice.(DruidChoiceKnownForms); ok && len(kf.Forms) == 0 {
		return ApplyDruidChoiceApplied{state: s, events: []Event{}}, nil
	}
	return ApplyDruidChoiceApplied{state: s, events: []Event{event}}, nil
}

func formIDs(forms []BeastFormFacts) []content.MonsterID {
	out := make([]content.MonsterID, len(forms))
	for i, f := range forms {
		out[i] = f.id
	}
	return out
}

// chooseKnownForms returns the rejection of adding the forms, or the state with them added.
func chooseKnownForms(s DruidFeatureState, in ApplyDruidChoiceInput, c DruidChoiceKnownForms) (DruidRejection, DruidFeatureState) {
	level := in.Facts.lv()
	reasons := []DruidRejection{when(!in.Facts.hasWildShape(), DruidFeatureNotGranted)}
	seen := s.known
	next := s
	for _, f := range c.Forms {
		reasons = append(reasons, formIneligibility(f, in.Tables, level), when(seen.Contains(f.id), DruidDuplicateForm))
		if !seen.Contains(f.id) && seen.Len() < maxKnownForms {
			seen = seen.with(f.id)
		}
	}
	reasons = append(reasons, when(s.known.Len()+len(c.Forms) > in.Tables.KnownFormsAt(level), DruidTooManyKnownForms))
	if r, bad := lowestDruidRejection(reasons...); bad {
		return r, s
	}
	next.known = seen
	return 0, next
}

// replaceKnownForm returns the rejection of replacing a known form, or the state with the form replaced.
func replaceKnownForm(s DruidFeatureState, in ApplyDruidChoiceInput, c DruidChoiceReplaceForm) (DruidRejection, DruidFeatureState) {
	reasons := []DruidRejection{
		when(!in.Facts.hasWildShape(), DruidFeatureNotGranted),
		when(!s.window.has(RestWindowKnownFormReplacement), DruidChangeNotAllowedNow),
		when(!s.known.Contains(c.Replace), DruidFormNotKnown),
		formIneligibility(c.With, in.Tables, in.Facts.lv()),
		when(s.known.Contains(c.With.id), DruidDuplicateForm),
		when(s.swaps >= 1, DruidTooManyReplacements),
	}
	if r, bad := lowestDruidRejection(reasons...); bad {
		return r, s
	}
	s.known = s.known.replace(c.Replace, c.With.id)
	s.swaps++
	return 0, s
}
