package content

// enumMask returns the bit set of the members vals of a vocabulary of count
// members, bit v-1 for member v. It returns BlockInvalidMember for a value
// that is not a member, then BlockDuplicate for a repeat.
func enumMask(block string, vals []int, count int) (uint32, error) {
	var mask uint32
	for _, v := range vals {
		if v < 1 || v > count {
			return 0, blockError(block, BlockInvalidMember)
		}
	}
	for _, v := range vals {
		bit := uint32(1) << (v - 1)
		if mask&bit != 0 {
			return 0, blockError(block, BlockDuplicate)
		}
		mask |= bit
	}
	return mask, nil
}

// maskMembers returns the members in the set, in ascending order.
func maskMembers(mask uint32, count int) []int {
	var out []int
	for v := 1; v <= count; v++ {
		if mask&(uint32(1)<<(v-1)) != 0 {
			out = append(out, v)
		}
	}
	return out
}

// CreatureTypeRestriction is the closed set of limits a spell puts on the
// creature types it may target (SRD 5.2.1, "Hold Person", p. 141):
// [AnyCreatureType] and [OnlyCreatureTypes]. Every variant is an immutable,
// comparable value and safe for concurrent use. Consumers switch over it
// exhaustively.
type CreatureTypeRestriction interface{ isCreatureTypeRestriction() }

// AnyCreatureType is the absence of a restriction. The zero value is valid.
type AnyCreatureType struct{}

func (AnyCreatureType) isCreatureTypeRestriction() {}

// OnlyCreatureTypes limits a spell to a non-empty set of creature types. The
// zero value is invalid; build it with [NewOnlyCreatureTypes]. The set has no
// order: two values are equal when they hold the same types.
type OnlyCreatureTypes struct{ types uint32 }

// NewOnlyCreatureTypes returns the restriction to the types, copying them.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// for no type, BlockInvalidMember for a value that is not a creature type and
// BlockDuplicate for a repeat. It never panics.
func NewOnlyCreatureTypes(types []CreatureType) (OnlyCreatureTypes, error) {
	if len(types) == 0 {
		return OnlyCreatureTypes{}, blockError("OnlyCreatureTypes", BlockEmpty)
	}
	vals := make([]int, len(types))
	for i, t := range types {
		vals[i] = int(t)
	}
	mask, err := enumMask("OnlyCreatureTypes", vals, len(creatureTypeNames()))
	if err != nil {
		return OnlyCreatureTypes{}, err
	}
	return OnlyCreatureTypes{types: mask}, nil
}

// Types returns the creature types in vocabulary order, as a new slice.
func (r OnlyCreatureTypes) Types() []CreatureType {
	var out []CreatureType
	for _, v := range maskMembers(r.types, len(creatureTypeNames())) {
		out = append(out, CreatureType(v))
	}
	return out
}

// Contains reports whether t is in the set.
func (r OnlyCreatureTypes) Contains(t CreatureType) bool {
	return t.Valid() && r.types&(uint32(1)<<(t-1)) != 0
}

func (OnlyCreatureTypes) isCreatureTypeRestriction() {}

// SpellTargeting is the closed set of ways a spell picks its targets (SRD
// 5.2.1, "Targets", p. 106): [TargetsSelf], [TargetsChosen] and
// [TargetsInArea]. Every variant is an immutable, comparable value and safe
// for concurrent use. Consumers switch over it exhaustively.
type SpellTargeting interface{ isSpellTargeting() }

// TargetsSelf is a spell whose only target is its caster (SRD 5.2.1, "Range",
// p. 105). The zero value is valid.
type TargetsSelf struct{}

func (TargetsSelf) isSpellTargeting() {}

// TargetsChosen is a spell whose caster chooses its targets. The zero value is
// invalid; build it with [NewTargetsChosen].
type TargetsChosen struct {
	subject     TargetSubject
	count       TargetCount
	restriction CreatureTypeRestriction
	selfPolicy  SelfTargetPolicy
}

// NewTargetsChosen returns the targeting of count targets of subject, limited
// by restriction (nil is [AnyCreatureType]) and by selfPolicy (SRD 5.2.1,
// "Targeting Yourself", p. 106).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid subject or self policy, the reason of the
// count's or restriction's own check for an invalid one, and BlockConstraint
// for a restriction other than [AnyCreatureType] on an object subject (FR-015).
// It never panics.
func NewTargetsChosen(subject TargetSubject, count TargetCount, restriction CreatureTypeRestriction, selfPolicy SelfTargetPolicy) (TargetsChosen, error) {
	if !selfPolicy.Valid() {
		return TargetsChosen{}, blockError("TargetsChosen", BlockInvalidMember)
	}
	if err := checkTargetCount(count); err != nil {
		return TargetsChosen{}, err
	}
	restriction, err := checkSubject("TargetsChosen", subject, restriction)
	if err != nil {
		return TargetsChosen{}, err
	}
	return TargetsChosen{subject: subject, count: count, restriction: restriction, selfPolicy: selfPolicy}, nil
}

// checkSubject applies the subject and restriction rules shared by the chosen
// and area targetings, and returns the restriction with nil replaced by
// [AnyCreatureType].
func checkSubject(block string, subject TargetSubject, r CreatureTypeRestriction) (CreatureTypeRestriction, error) {
	if !subject.Valid() {
		return nil, blockError(block, BlockInvalidMember)
	}
	switch v := r.(type) {
	case nil:
		return AnyCreatureType{}, nil
	case AnyCreatureType:
		return r, nil
	case OnlyCreatureTypes:
		if _, err := NewOnlyCreatureTypes(v.Types()); err != nil {
			return nil, err
		}
		if subject == SubjectObject {
			return nil, blockError(block, BlockConstraint)
		}
		return r, nil
	}
	return nil, blockError(block, BlockInvalidMember)
}

// Subject returns what the spell may target.
func (t TargetsChosen) Subject() TargetSubject { return t.subject }

// Count returns how many targets the spell has.
func (t TargetsChosen) Count() TargetCount { return t.count }

// Restriction returns the limit on the creature types.
func (t TargetsChosen) Restriction() CreatureTypeRestriction { return t.restriction }

// SelfPolicy returns whether the caster may be a target.
func (t TargetsChosen) SelfPolicy() SelfTargetPolicy { return t.selfPolicy }

func (TargetsChosen) isSpellTargeting() {}

// TargetsInArea is a spell that affects an area of effect (SRD 5.2.1, "Area
// of Effect", p. 177). The zero value is invalid; build it with
// [NewTargetsInArea].
type TargetsInArea struct {
	shape       AreaShape
	subject     TargetSubject
	restriction CreatureTypeRestriction
}

// NewTargetsInArea returns the targeting of subjects in shape, limited by
// restriction (nil is [AnyCreatureType]).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the shape's own check for an invalid shape (BlockMissing for nil),
// BlockInvalidMember for an invalid subject or restriction, and
// BlockConstraint for a restriction other than [AnyCreatureType] on an object
// subject (FR-015). It never panics.
func NewTargetsInArea(shape AreaShape, subject TargetSubject, restriction CreatureTypeRestriction) (TargetsInArea, error) {
	if err := checkAreaShape(shape); err != nil {
		return TargetsInArea{}, err
	}
	restriction, err := checkSubject("TargetsInArea", subject, restriction)
	if err != nil {
		return TargetsInArea{}, err
	}
	return TargetsInArea{shape: shape, subject: subject, restriction: restriction}, nil
}

// Shape returns the area of effect.
func (t TargetsInArea) Shape() AreaShape { return t.shape }

// Subject returns what the spell may target in the area.
func (t TargetsInArea) Subject() TargetSubject { return t.subject }

// Restriction returns the limit on the creature types.
func (t TargetsInArea) Restriction() CreatureTypeRestriction { return t.restriction }

func (TargetsInArea) isSpellTargeting() {}

// checkSpellTargeting reports why t is not a targeting its constructor would
// accept: BlockMissing for nil.
func checkSpellTargeting(t SpellTargeting) error {
	var err error
	switch v := t.(type) {
	case TargetsSelf:
	case TargetsChosen:
		_, err = NewTargetsChosen(v.subject, v.count, v.restriction, v.selfPolicy)
	case TargetsInArea:
		_, err = NewTargetsInArea(v.shape, v.subject, v.restriction)
	default:
		err = blockError("SpellTargeting", BlockMissing)
	}
	return err
}
