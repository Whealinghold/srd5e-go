package combat

// BeginShapeShiftRejection is the closed set of reasons ShapeShiftSpells.Begin refuses. Each
// has a snake_case String that is the value of srd5e.combat.result. Variants: TargetIsDead,
// TargetNotWilling, ShapechangeNotSelf, FormIneligible.
type BeginShapeShiftRejection interface {
	// String returns the snake_case name of the rejection.
	String() string
	isBeginShapeShiftRejection()
}

// ChangeFormRejection is the closed set of reasons ShapeShiftSpells.ChangeForm refuses. Each
// has a snake_case String that is the value of srd5e.combat.result. Variants: TargetIsDead,
// FormChangeNotAllowed, NotTransformed, ActorNotCaster, FormIneligible.
type ChangeFormRejection interface {
	// String returns the snake_case name of the rejection.
	String() string
	isChangeFormRejection()
}

// TargetIsDead: the target is Dead (SRD 5.2.1, "Dead", p. 180). It refuses Begin and
// ChangeForm.
type TargetIsDead struct{}

// TargetNotWilling: Animal Shapes and Shapechange choose only willing creatures.
type TargetNotWilling struct{}

// ShapechangeNotSelf: Shapechange has a range of Self.
type ShapechangeNotSelf struct{}

// FormChangeNotAllowed: Polymorph and True Polymorph never change form.
type FormChangeNotAllowed struct{}

// NotTransformed: there is no transformation to change.
type NotTransformed struct{}

// ActorNotCaster: only the caster changes the form.
type ActorNotCaster struct{}

// String returns "target_is_dead".
func (TargetIsDead) String() string { return "target_is_dead" }

// String returns "target_not_willing".
func (TargetNotWilling) String() string { return "target_not_willing" }

// String returns "shapechange_not_self".
func (ShapechangeNotSelf) String() string { return "shapechange_not_self" }

// String returns "form_change_not_allowed".
func (FormChangeNotAllowed) String() string { return "form_change_not_allowed" }

// String returns "not_transformed".
func (NotTransformed) String() string { return "not_transformed" }

// String returns "actor_not_caster".
func (ActorNotCaster) String() string { return "actor_not_caster" }

func (TargetIsDead) isBeginShapeShiftRejection()       {}
func (TargetNotWilling) isBeginShapeShiftRejection()   {}
func (ShapechangeNotSelf) isBeginShapeShiftRejection() {}
func (FormIneligible) isBeginShapeShiftRejection()     {}
func (TargetIsDead) isChangeFormRejection()            {}
func (FormChangeNotAllowed) isChangeFormRejection()    {}
func (NotTransformed) isChangeFormRejection()          {}
func (ActorNotCaster) isChangeFormRejection()          {}
func (FormIneligible) isChangeFormRejection()          {}

// EndShapeShiftRejection is why ShapeShiftSpells.End refuses. The zero value is invalid.
type EndShapeShiftRejection int

// The two reasons End refuses.
const (
	// CannotEndAtWill: only Animal Shapes lets the target end its transformation.
	CannotEndAtWill EndShapeShiftRejection = iota + 1
	// EndTargetIsDead: the target is Dead and only TargetDied and TargetDestroyed apply.
	EndTargetIsDead
)

// String returns "cannot_end_at_will", "target_is_dead" or "unknown".
func (r EndShapeShiftRejection) String() string {
	return enumName([]string{"cannot_end_at_will", "target_is_dead"}, int(r))
}
