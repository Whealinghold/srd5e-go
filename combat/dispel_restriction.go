package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// DispelRestriction says whether and how Dispel Magic can end one ongoing spell
// (SRD 5.2.1, "Exceptions Supersede General Rules", p. 5). It is a closed set:
// DispelUnrestricted, DispelNeverDispellable, DispelRequiresSlotLevel and
// DispelAfterPartsGone. Build the values with DispelRestrictions.ForSpell or the
// constructors of the variants.
type DispelRestriction interface {
	isDispelRestriction()
}

// DispelUnrestricted is the restriction of a spell Dispel Magic can end by the
// general rule (SRD 5.2.1, "Dispel Magic", p. 124).
type DispelUnrestricted struct{}

// DispelNeverDispellable is the restriction of a spell that "can't be dispelled"
// (for example Forcecage, SRD 5.2.1, p. 134).
type DispelNeverDispellable struct{}

// DispelRequiresSlotLevel is the restriction of a spell that Dispel Magic ends
// only from a slot of at least Level (Imprisonment, SRD 5.2.1, p. 142).
type DispelRequiresSlotLevel struct{ level character.SlotLevel }

// NewDispelRequiresSlotLevel returns the restriction that needs a slot of at
// least level. It never panics.
func NewDispelRequiresSlotLevel(level character.SlotLevel) DispelRequiresSlotLevel {
	return DispelRequiresSlotLevel{level: level}
}

// Level returns the slot level Dispel Magic needs.
func (r DispelRequiresSlotLevel) Level() character.SlotLevel { return r.level }

// DispelAfterPartsGone is the restriction of a part that Dispel Magic can end
// only after every listed part is gone from the same target (the violet layer of
// a Prismatic Wall, SRD 5.2.1, p. 155).
type DispelAfterPartsGone struct{ ids []OngoingEffectID }

// NewDispelAfterPartsGone returns the restriction that waits for ids to be gone.
// It copies ids and never panics.
func NewDispelAfterPartsGone(ids []OngoingEffectID) DispelAfterPartsGone {
	return DispelAfterPartsGone{ids: slices.Clone(ids)}
}

// IDs returns a copy of the ids that must be gone first.
func (r DispelAfterPartsGone) IDs() []OngoingEffectID { return slices.Clone(r.ids) }

func (DispelUnrestricted) isDispelRestriction()      {}
func (DispelNeverDispellable) isDispelRestriction()  {}
func (DispelRequiresSlotLevel) isDispelRestriction() {}
func (DispelAfterPartsGone) isDispelRestriction()    {}

// dispelRestrictionEqual reports whether a and b are the same restriction with
// equal fields. A nil restriction equals only nil.
func dispelRestrictionEqual(a, b DispelRestriction) bool {
	switch x := a.(type) {
	case DispelUnrestricted, DispelNeverDispellable:
		return a == b
	case DispelRequiresSlotLevel:
		y, ok := b.(DispelRequiresSlotLevel)
		return ok && x.level == y.level
	case DispelAfterPartsGone:
		y, ok := b.(DispelAfterPartsGone)
		return ok && slices.Equal(x.ids, y.ids)
	default:
		return a == nil && b == nil
	}
}

// ForSpellOptions carries what DispelRestrictions.ForSpell needs beyond the spell
// id. Permanent is whether a Wall of Stone is permanent (SRD 5.2.1, p. 173). Part
// names which part of a multi-part spell the entry is; the zero SpellPart is the
// whole spell or its parent. Siblings are the ids of the other parts of the same
// parent, which the host reads from the registry; only the violet layer of a
// Prismatic Wall uses them.
type ForSpellOptions struct {
	Permanent bool
	Part      SpellPart
	Siblings  []OngoingEffectID
}

// DispelRestrictions gives the restriction of a spell (table T-R of
// dispel-and-counterspell.md). It is stateless: use the zero value.
type DispelRestrictions struct{}

// ForSpell returns the restriction of the spell id given the options. It returns
// Unrestricted for any spell the table does not name. It returns an error
// matching srderr.ErrInvalidInput for a Prismatic Wall layer outside 1 to 7 (a
// layer 8 is not a part), and for a part that the named spell does not have
// (a Prismatic Wall layer on Guards and Wards, or a Guards and Wards effect on a
// Prismatic Wall). It never panics.
func (DispelRestrictions) ForSpell(id content.SpellID, opts ForSpellOptions) (DispelRestriction, error) {
	switch id {
	case "forcecage", "wall_of_force", "antimagic_field":
		return DispelNeverDispellable{}, nil
	case "wall_of_stone":
		if opts.Permanent {
			return DispelNeverDispellable{}, nil
		}
		return DispelUnrestricted{}, nil
	case "guards_and_wards":
		switch {
		case opts.Part.IsWhole():
			return DispelNeverDispellable{}, nil
		case opts.Part.kind == spellPartGuardsAndWardsEffect:
			return DispelUnrestricted{}, nil
		default:
			return nil, invalidInput("guards and wards has no prismatic layer part")
		}
	case "imprisonment":
		return NewDispelRequiresSlotLevel(character.MustSlotLevel(9)), nil
	case "prismatic_wall":
		return prismaticRestriction(opts)
	default:
		return DispelUnrestricted{}, nil
	}
}

// prismaticRestriction is the restriction of a Prismatic Wall part (table T-R).
func prismaticRestriction(opts ForSpellOptions) (DispelRestriction, error) {
	switch {
	case opts.Part.IsWhole():
		return DispelNeverDispellable{}, nil
	case opts.Part.kind != spellPartPrismaticLayer:
		return nil, invalidInput("prismatic wall has no guards and wards part")
	case !opts.Part.layer.Valid():
		return nil, invalidInput("prismatic layer must be 1 to 7")
	case opts.Part.layer == 7:
		return NewDispelAfterPartsGone(opts.Siblings), nil
	default:
		return DispelNeverDispellable{}, nil
	}
}
