package combat

import "github.com/Whealinghold/srd5e-go/content"

// ShapeShiftSpell is one of the four spells of SRD 5.2.1 that shape-shift a creature:
// Polymorph (p. 153), True Polymorph (p. 170), Shapechange (p. 161) and Animal Shapes
// (p. 108). The zero ShapeShiftSpell is invalid. Comparable and safe for concurrent use.
type ShapeShiftSpell int

// The four shape-shifting spells.
const (
	PolymorphSpell ShapeShiftSpell = iota + 1
	TruePolymorphSpell
	ShapechangeSpell
	AnimalShapesSpell
)

func shapeShiftSpellIDs() []string {
	return []string{"polymorph", "true_polymorph", "shapechange", "animal_shapes"}
}

// Valid reports whether s is one of the four spells.
func (s ShapeShiftSpell) Valid() bool { return s >= PolymorphSpell && s <= AnimalShapesSpell }

// String returns the snake_case spell id, or "unknown".
func (s ShapeShiftSpell) String() string { return enumName(shapeShiftSpellIDs(), int(s)) }

// SpellID returns the content id of the spell, or the empty id for an invalid value.
func (s ShapeShiftSpell) SpellID() content.SpellID {
	if !s.Valid() {
		return ""
	}
	return content.SpellID(s.String())
}

// ShapeShiftSpellOf returns the shape-shifting spell with the content id id and true,
// or the zero ShapeShiftSpell and false for any other id. It never fails or panics.
func ShapeShiftSpellOf(id content.SpellID) (ShapeShiftSpell, bool) {
	for i, name := range shapeShiftSpellIDs() {
		if string(id) == name {
			return ShapeShiftSpell(i + 1), true
		}
	}
	return 0, false
}

// ShapeShiftProfile is the row of a spell in table A of shape-changing-spells.md: what the
// four spells say about Concentration, duration, saving throws and Temporary Hit Points.
// Construct it only through ShapeShiftSpell.Profile; the fields are read-only data.
type ShapeShiftProfile struct {
	// Level is the spell level.
	Level int
	// Concentration is whether the spell needs Concentration.
	Concentration bool
	// MaximumMinutes is the longest duration, in minutes.
	MaximumMinutes int
	// TargetSaves is whether an unwilling target makes a Wisdom saving throw.
	TargetSaves bool
	// GrantsTemporaryHitPoints is whether a form can grant Temporary Hit Points (True
	// Polymorph grants them only when a creature becomes a creature).
	GrantsTemporaryHitPoints bool
	// EndsWhenTemporaryHitPointsGone is whether the spell ends early at 0 Temporary Hit Points.
	EndsWhenTemporaryHitPointsGone bool
	// CanChangeForm is whether the form can change while the spell lasts.
	CanChangeForm bool
	// PermanentIfHeld is whether holding Concentration for the full duration makes the
	// spell last until dispelled.
	PermanentIfHeld bool
}

// Profile returns the table A row of s, and the zero profile for an invalid spell.
func (s ShapeShiftSpell) Profile() ShapeShiftProfile {
	switch s {
	case PolymorphSpell:
		return ShapeShiftProfile{Level: 4, Concentration: true, MaximumMinutes: 60, TargetSaves: true,
			GrantsTemporaryHitPoints: true, EndsWhenTemporaryHitPointsGone: true}
	case TruePolymorphSpell:
		return ShapeShiftProfile{Level: 9, Concentration: true, MaximumMinutes: 60, TargetSaves: true,
			GrantsTemporaryHitPoints: true, PermanentIfHeld: true}
	case ShapechangeSpell:
		return ShapeShiftProfile{Level: 9, Concentration: true, MaximumMinutes: 60,
			GrantsTemporaryHitPoints: true, CanChangeForm: true}
	case AnimalShapesSpell:
		return ShapeShiftProfile{Level: 8, MaximumMinutes: 24 * 60, GrantsTemporaryHitPoints: true, CanChangeForm: true}
	}
	return ShapeShiftProfile{}
}

// TrueTransformation is the mode of True Polymorph (SRD 5.2.1, "True Polymorph", p. 170).
// The zero value is invalid. Comparable and safe for concurrent use.
type TrueTransformation int

// The three modes of True Polymorph.
const (
	CreatureIntoCreature TrueTransformation = iota + 1
	ObjectIntoCreature
	CreatureIntoObject
)

// Valid reports whether m is one of the three modes.
func (m TrueTransformation) Valid() bool { return m >= CreatureIntoCreature && m <= CreatureIntoObject }

// String returns the snake_case mode name, or "unknown".
func (m TrueTransformation) String() string {
	return enumName([]string{"creature_into_creature", "object_into_creature", "creature_into_object"}, int(m))
}

// ShapechangeTerms says whether a Shapechange is the spell or the monster Spellcasting
// entry of a stat block (SRD 5.2.1, "Adult Brass Dragon", p. 269: Beast or Humanoid form
// only, no Temporary Hit Points, no Concentration). The zero value is StandardShapechange.
type ShapechangeTerms int

// The two terms of Shapechange.
const (
	StandardShapechange ShapechangeTerms = iota
	StatBlockShapechange
)

// Valid reports whether t is one of the two terms.
func (t ShapechangeTerms) Valid() bool { return t == StandardShapechange || t == StatBlockShapechange }

// EquipmentOutcome is what becomes of a creature's equipment in a new form (SRD 5.2.1,
// "Polymorph", p. 154; "Shapechange", p. 161). The zero value is invalid.
type EquipmentOutcome int

// The four equipment outcomes.
const (
	MeldsUnusable EquipmentOutcome = iota + 1
	DropsToGround
	ResizesToFit
	TransformsWithCreature
)

// Valid reports whether o is one of the four outcomes.
func (o EquipmentOutcome) Valid() bool { return o >= MeldsUnusable && o <= TransformsWithCreature }

// String returns the snake_case outcome name, or "unknown".
func (o EquipmentOutcome) String() string {
	return enumName([]string{"melds_unusable", "drops_to_ground", "resizes_to_fit", "transforms_with_creature"}, int(o))
}

// FormTemporaryHitPoints says where the Temporary Hit Points of a transformation came from
// (SRD 5.2.1, "They Don't Stack", p. 18). The zero value is invalid.
type FormTemporaryHitPoints int

// The provenances of the Temporary Hit Points of a form.
const (
	// TemporaryFromForm: the form's Hit Points are held as Temporary Hit Points.
	TemporaryFromForm FormTemporaryHitPoints = iota + 1
	// TemporaryNotGranted: the spell or mode grants none.
	TemporaryNotGranted
	// TemporaryDeclined: the creature kept the Temporary Hit Points it held, or the offer
	// equalled them; the held points are not the form's.
	TemporaryDeclined
	// TemporaryReplaced: another effect's Temporary Hit Points took the place of the form's.
	TemporaryReplaced
)

// Valid reports whether p is one of the four provenances.
func (p FormTemporaryHitPoints) Valid() bool { return p >= TemporaryFromForm && p <= TemporaryReplaced }

// String returns the snake_case provenance, or "unknown".
func (p FormTemporaryHitPoints) String() string {
	return enumName([]string{"from_form", "not_granted", "declined", "replaced"}, int(p))
}

// MaintainedBy says whether a transformation depends on its caster's Concentration. The
// zero value is invalid.
type MaintainedBy int

// The two ways a transformation is maintained.
const (
	MaintainedByConcentration MaintainedBy = iota + 1
	NotMaintained
)

// Valid reports whether m is one of the two values.
func (m MaintainedBy) Valid() bool { return m == MaintainedByConcentration || m == NotMaintained }

// ShapeShiftEndCause is why a transformation ends. The zero value is invalid.
type ShapeShiftEndCause int

// The eight causes (snake_case names: duration_elapsed, concentration_ended,
// form_hit_points_depleted, target_died, target_destroyed, ended_by_target, dispelled,
// replaced).
const (
	CauseDurationElapsed ShapeShiftEndCause = iota + 1
	CauseConcentrationEnded
	CauseFormHitPointsDepleted
	CauseTargetDied
	CauseTargetDestroyed
	CauseEndedByTarget
	CauseDispelled
	CauseReplaced
)

// Valid reports whether c is one of the eight causes.
func (c ShapeShiftEndCause) Valid() bool { return c >= CauseDurationElapsed && c <= CauseReplaced }

// String returns the snake_case cause, or "unknown".
func (c ShapeShiftEndCause) String() string {
	return enumName([]string{"duration_elapsed", "concentration_ended", "form_hit_points_depleted", "target_died",
		"target_destroyed", "ended_by_target", "dispelled", "replaced"}, int(c))
}

// ShapeShiftActionCost is the action a shape-shifting operation costs, as data the host
// spends. The zero value is invalid.
type ShapeShiftActionCost int

// The two costs.
const (
	ShapeShiftMagicAction ShapeShiftActionCost = iota + 1
	ShapeShiftBonusAction
)

// String returns "magic_action", "bonus_action" or "unknown".
func (c ShapeShiftActionCost) String() string {
	return enumName([]string{"magic_action", "bonus_action"}, int(c))
}
