package combat

// DispelTarget is what an ongoing spell is on: a creature, an object or a
// magical effect such as a wall (SRD 5.2.1, "Dispel Magic", p. 124: "Choose one
// creature, object, or magical effect"). It is a closed set: DispelCreature,
// DispelObject and DispelEffect. Each is comparable, so it can be compared with
// ==. Build one with the constructor of its variant; the zero value of the
// interface is not a target.
type DispelTarget interface {
	isDispelTarget()
}

// DispelCreature is a creature the spells are on.
type DispelCreature struct{ id CombatantID }

// DispelObject is an object the spells are on.
type DispelObject struct{ id SpellObjectID }

// DispelEffect is a magical effect that is not a creature or object (a wall, a
// zone), keyed by its ongoing effect id (FR-058).
type DispelEffect struct{ id OngoingEffectID }

// NewDispelCreature returns the target that is the creature id. It returns an
// error matching srderr.ErrInvalidInput for the zero CombatantID. It never panics.
func NewDispelCreature(id CombatantID) (DispelCreature, error) {
	if id.IsZero() {
		return DispelCreature{}, invalidInput("a dispel creature target needs an id")
	}
	return DispelCreature{id: id}, nil
}

// NewDispelObject returns the target that is the object id. It returns an error
// matching srderr.ErrInvalidInput for the zero SpellObjectID. It never panics.
func NewDispelObject(id SpellObjectID) (DispelObject, error) {
	if id == "" {
		return DispelObject{}, invalidInput("a dispel object target needs an id")
	}
	return DispelObject{id: id}, nil
}

// NewDispelEffect returns the target that is the magical effect id. It returns an
// error matching srderr.ErrInvalidInput for the zero OngoingEffectID. It never panics.
func NewDispelEffect(id OngoingEffectID) (DispelEffect, error) {
	if id.IsZero() {
		return DispelEffect{}, invalidInput("a dispel effect target needs an id")
	}
	return DispelEffect{id: id}, nil
}

// ID returns the object id.
func (t DispelObject) ID() SpellObjectID { return t.id }

// ID returns the effect id.
func (t DispelEffect) ID() OngoingEffectID { return t.id }

func (DispelCreature) isDispelTarget() {}
func (DispelObject) isDispelTarget()   {}
func (DispelEffect) isDispelTarget()   {}

// validDispelTarget reports whether t is one of the variants with an id.
func validDispelTarget(t DispelTarget) bool {
	switch v := t.(type) {
	case DispelCreature:
		return !v.id.IsZero()
	case DispelObject:
		return v.id != ""
	case DispelEffect:
		return !v.id.IsZero()
	default:
		return false
	}
}
