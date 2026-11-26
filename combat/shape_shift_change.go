package combat

// ChangeFormCommand is the host's facts for ChangeForm. Construct it with a keyed literal.
type ChangeFormCommand struct {
	// Actor is the creature taking the Magic action; it must be the caster.
	Actor CombatantID
	// State is the transformation to change; the zero value means none.
	State ActiveTransformation
	// Form is the new form, with the size chosen among its options.
	Form AssumedForm
	// Rating is the caster's rating, read for Shapechange (the same TargetRating as Begin).
	Rating TargetRating
	// HasSeenKind is whether the caster has seen the sort of creature (Shapechange).
	HasSeenKind bool
	// Equipment is the fresh choice for Shapechange (DropsToGround or ResizesToFit); it must be
	// the zero value for Animal Shapes, whose outcome is always MeldsUnusable.
	Equipment EquipmentOutcome
	// TargetLife is the life state of the target; nil means Alive.
	TargetLife LifeState
}

// ChangeShapeShiftFormResult is the closed set of answers of ShapeShiftSpells.ChangeForm. Variants:
// ChangeFormApplied, ChangeFormRefused.
type ChangeShapeShiftFormResult interface{ isChangeShapeShiftFormResult() }

// ChangeFormApplied: the form changed. The Hit Points, the Temporary Hit Points and their
// provenance, the caster and MaintainedBy are unchanged. The host spends the ActionCost.
type ChangeFormApplied struct {
	transformation ActiveTransformation
	events         []Event
	equipment      EquipmentOutcome
}

// Transformation returns the new ActiveTransformation.
func (r ChangeFormApplied) Transformation() ActiveTransformation { return r.transformation }

// Events returns a copy of the events: one ShapeShiftFormChanged.
func (r ChangeFormApplied) Events() []Event { return append([]Event(nil), r.events...) }

// Equipment returns what becomes of the equipment in the new form.
func (r ChangeFormApplied) Equipment() EquipmentOutcome { return r.equipment }

// ActionCost returns ShapeShiftMagicAction, the action the change costs, as data.
func (ChangeFormApplied) ActionCost() ShapeShiftActionCost { return ShapeShiftMagicAction }

// ChangeFormRefused: the rules refuse; the state is unchanged. Construct it with a keyed literal.
type ChangeFormRefused struct{ Rejection ChangeFormRejection }

func (ChangeFormApplied) isChangeShapeShiftFormResult() {}
func (ChangeFormRefused) isChangeShapeShiftFormResult() {}

// ChangeForm changes the form of a Shapechange or Animal Shapes transformation (SRD 5.2.1,
// "Shapechange", p. 161; "Animal Shapes", p. 108). The checks run in this order and only the
// first failure is reported: a Dead target (TargetIsDead; the host calls End with TargetDied
// instead), a spell that cannot change form (FormChangeNotAllowed), no transformation
// (NotTransformed), an actor who is not the caster (ActorNotCaster), then the eligibility of
// the new form by the rules of Begin (FormIneligible, every reason). A permanent
// transformation is True Polymorph, so it is refused as FormChangeNotAllowed.
//
// It returns an error matching srderr.ErrInvalidInput, with a nil result, for a zero actor or
// form, an Equipment that is not DropsToGround or ResizesToFit for Shapechange, or one that
// is set for Animal Shapes. It never panics and changes none of its arguments.
func (s ShapeShiftSpells) ChangeForm(c ChangeFormCommand) (ChangeShapeShiftFormResult, error) {
	t := c.State
	if c.Actor.IsZero() || c.Form == nil {
		return nil, invalidInput("ChangeForm needs an actor and a form")
	}
	if t.spell.Profile().CanChangeForm {
		if shapechange := t.spell == ShapechangeSpell; (shapechange && !validShapechangeChoice(c.Equipment)) || (!shapechange && c.Equipment != 0) {
			return nil, invalidInput("only Shapechange takes an equipment choice, DropsToGround or ResizesToFit")
		}
	}
	_, dead := c.TargetLife.(Dead)
	switch {
	case dead:
		return ChangeFormRefused{TargetIsDead{}}, nil
	case !t.IsZero() && !t.spell.Profile().CanChangeForm:
		return ChangeFormRefused{FormChangeNotAllowed{}}, nil
	case t.IsZero():
		return ChangeFormRefused{NotTransformed{}}, nil
	case c.Actor != t.caster:
		return ChangeFormRefused{ActorNotCaster{}}, nil
	}
	eligibility, err := s.Eligibility(FormQuery{Spell: t.spell, Terms: t.terms, Form: c.Form, Rating: c.Rating, HasSeenKind: c.HasSeenKind})
	if err != nil {
		return nil, err
	}
	if ineligible, bad := eligibility.(FormIneligible); bad {
		return ChangeFormRefused{ineligible}, nil
	}
	t.form = c.Form
	return ChangeFormApplied{transformation: t, events: []Event{ShapeShiftFormChanged{shapeShiftBaseOf(t)}},
		equipment: equipmentOf(t.spell, t.mode, c.Equipment)}, nil
}
