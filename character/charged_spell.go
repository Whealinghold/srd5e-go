package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// ItemSpellLevel is the level a fixed-cost charged spell is cast at: the spell's
// lowest level or a named level. Comparable; the zero value is invalid.
type ItemSpellLevel struct {
	kind  int
	level content.SpellLevel
}

// LowestLevel casts the spell at its lowest level.
func LowestLevel() ItemSpellLevel { return ItemSpellLevel{kind: 1} }

// NamedLevel casts the spell at the level the item's description names.
func NamedLevel(l content.SpellLevel) ItemSpellLevel { return ItemSpellLevel{kind: 2, level: l} }

func (l ItemSpellLevel) valid() bool {
	return l.kind == 1 || l.kind == 2 && content.ValidSpellLevel(int(l.level))
}

// ItemSpellStats says whose numbers a charged spell uses. Comparable; the zero
// value is invalid.
type ItemSpellStats struct {
	kind int
	dc   int
}

// UsersSpellStats uses the user's spell save DC, spell attack modifier or
// spellcasting ability modifier, resolved through ItemSpellcastingAbility.
func UsersSpellStats() ItemSpellStats { return ItemSpellStats{kind: 1} }

// FixedSaveDC uses the printed save DC n (for example the Wand of Web, 13).
func FixedSaveDC(n int) ItemSpellStats { return ItemSpellStats{kind: 2, dc: n} }

// SaveDC returns the printed DC and true, or 0 and false for UsersSpellStats.
func (s ItemSpellStats) SaveDC() (int, bool) { return s.dc, s.kind == 2 }

func (s ItemSpellStats) valid() bool { return s.kind == 1 || s.kind == 2 && s.dc >= 1 }

// ChargedSpellOption is the closed set of ways an item casts a spell by expending
// charges. Variants: FixedChargeCost, ChargesPerLevel.
type ChargedSpellOption interface{ isChargedSpellOption() }

// FixedChargeCost is a spell with one printed cost in charges.
type FixedChargeCost struct {
	spell   content.SpellID
	charges int
	level   ItemSpellLevel
	stats   ItemSpellStats
}

// NewFixedChargeCost returns the option. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell, negative charges, or an invalid
// level or stats. Zero charges is a free spell.
func NewFixedChargeCost(spell content.SpellID, charges int, level ItemSpellLevel, stats ItemSpellStats) (FixedChargeCost, error) {
	if !spell.Valid() || charges < 0 || !level.valid() || !stats.valid() {
		return FixedChargeCost{}, invalidInput("fixed charge cost: invalid spell, charges, level or stats")
	}
	return FixedChargeCost{spell, charges, level, stats}, nil
}

// MustFixedChargeCost is NewFixedChargeCost for literals; it panics with the error.
func MustFixedChargeCost(spell content.SpellID, charges int, level ItemSpellLevel, stats ItemSpellStats) FixedChargeCost {
	o, err := NewFixedChargeCost(spell, charges, level, stats)
	if err != nil {
		panic(err)
	}
	return o
}

// SpellID returns the spell.
func (o FixedChargeCost) SpellID() content.SpellID { return o.spell }

// Charges returns the cost.
func (o FixedChargeCost) Charges() int { return o.charges }

// Level returns the level rule.
func (o FixedChargeCost) Level() ItemSpellLevel { return o.level }

// Stats returns whose numbers the spell uses.
func (o FixedChargeCost) Stats() ItemSpellStats { return o.stats }

// ChargesPerLevel is a spell whose level is the lowest level plus the charges
// spent minus one, up to a maximum number of charges.
type ChargesPerLevel struct {
	spell   content.SpellID
	lowest  content.SpellLevel
	maximum int
	stats   ItemSpellStats
}

// NewChargesPerLevel returns the option. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell or stats, a maximum below 1, or a
// lowest level from which the maximum would pass level 9.
func NewChargesPerLevel(spell content.SpellID, lowest content.SpellLevel, maximumCharges int, stats ItemSpellStats) (ChargesPerLevel, error) {
	if !spell.Valid() || !content.ValidSpellLevel(int(lowest)) || maximumCharges < 1 ||
		!content.ValidSpellLevel(int(lowest)+maximumCharges-1) || !stats.valid() {
		return ChargesPerLevel{}, invalidInput("charges per level: invalid spell, level, maximum or stats")
	}
	return ChargesPerLevel{spell, lowest, maximumCharges, stats}, nil
}

// MustChargesPerLevel is NewChargesPerLevel for literals; it panics with the error.
func MustChargesPerLevel(spell content.SpellID, lowest content.SpellLevel, maximumCharges int, stats ItemSpellStats) ChargesPerLevel {
	o, err := NewChargesPerLevel(spell, lowest, maximumCharges, stats)
	if err != nil {
		panic(err)
	}
	return o
}

// SpellID returns the spell.
func (o ChargesPerLevel) SpellID() content.SpellID { return o.spell }

// LowestLevel returns the level one charge casts at.
func (o ChargesPerLevel) LowestLevel() content.SpellLevel { return o.lowest }

// MaximumCharges returns the most charges one cast may spend.
func (o ChargesPerLevel) MaximumCharges() int { return o.maximum }

// Stats returns whose numbers the spell uses.
func (o ChargesPerLevel) Stats() ItemSpellStats { return o.stats }

func (FixedChargeCost) isChargedSpellOption() {}
func (ChargesPerLevel) isChargedSpellOption() {}

// ChargedSpellPlanRejection says why a plan was refused. The zero value is not a reason.
type ChargedSpellPlanRejection int

// The reasons.
const (
	ChargesOutOfRange ChargedSpellPlanRejection = iota + 1
	ChargesNotAllowed
	LevelBelowLowest
)

// String returns "charges_out_of_range", "charges_not_allowed", "level_below_lowest" or "unknown".
func (r ChargedSpellPlanRejection) String() string {
	names := []string{"charges_out_of_range", "charges_not_allowed", "level_below_lowest"}
	if r < 1 || int(r) > len(names) {
		return "unknown"
	}
	return names[r-1]
}

// ChargedSpellPlanResult is the closed set of outcomes of ChargedSpellCasting.Plan.
// Variants: ChargedSpellPlan, ChargedSpellPlanRejected.
type ChargedSpellPlanResult interface{ isChargedSpellPlanResult() }

// ChargedSpellPlan is a cast the item can make: the spell, the charges it costs
// and the level it is cast at. It spends nothing and holds no spell slot.
type ChargedSpellPlan struct {
	spell  content.SpellID
	charge int
	lowest content.SpellLevel
	castAt content.SpellLevel
	stats  ItemSpellStats
}

// SpellID returns the spell.
func (p ChargedSpellPlan) SpellID() content.SpellID { return p.spell }

// Charges returns the charges the cast spends.
func (p ChargedSpellPlan) Charges() int { return p.charge }

// CastAtLevel returns the level the spell is cast at.
func (p ChargedSpellPlan) CastAtLevel() content.SpellLevel { return p.castAt }

// Stats returns whose numbers the spell uses.
func (p ChargedSpellPlan) Stats() ItemSpellStats { return p.stats }

// ChargedSpellPlanRejected: the charges or level do not fit the option.
type ChargedSpellPlanRejected struct{ reason ChargedSpellPlanRejection }

// Reason returns why.
func (r ChargedSpellPlanRejected) Reason() ChargedSpellPlanRejection { return r.reason }

func (ChargedSpellPlan) isChargedSpellPlanResult()         {}
func (ChargedSpellPlanRejected) isChargedSpellPlanResult() {}

// ChargedSpellCastResult is the closed set of outcomes of ChargedSpellCasting.Cast.
// Variants: ChargedSpellCastApplied, MagicItemInsufficientCharges.
type ChargedSpellCastResult interface{ isChargedSpellCastResult() }

// ChargedSpellCastApplied: the cast was paid for.
type ChargedSpellCastApplied struct {
	charges ItemCharges
	event   MagicItemChargesSpent
	spent   bool
	spell   SpellFromItem
	stats   ItemSpellStats
}

// Charges returns the charges left.
func (r ChargedSpellCastApplied) Charges() ItemCharges { return r.charges }

// Event returns the MagicItemChargesSpent event and true, or the zero event and
// false when the cast cost nothing.
func (r ChargedSpellCastApplied) Event() (MagicItemChargesSpent, bool) { return r.event, r.spent }

// Spell returns the spell to cast.
func (r ChargedSpellCastApplied) Spell() SpellFromItem { return r.spell }

// Stats returns whose numbers the spell uses.
func (r ChargedSpellCastApplied) Stats() ItemSpellStats { return r.stats }

func (ChargedSpellCastApplied) isChargedSpellCastResult()      {}
func (MagicItemInsufficientCharges) isChargedSpellCastResult() {}

// ChargedSpellCasting plans and pays for casts by charges (SRD 5.2.1, "Spells
// Cast from Items", p. 206; "Charges", p. 206). It takes and returns no spell
// slots. The zero value is ready to use.
type ChargedSpellCasting struct{}

// Plan checks charges against the option. For ChargesPerLevel the cast level is
// lowest + charges - 1 and charges run from 1 to the maximum; for FixedChargeCost
// charges must equal the cost and the level is spellLowest (LowestLevel) or the
// named one. spellLowest is the spell's own lowest level, passed by the caller. A
// named level below spellLowest is LevelBelowLowest. It returns an error matching
// srderr.ErrInvalidInput for a nil or foreign option or an invalid spellLowest.
func (ChargedSpellCasting) Plan(option ChargedSpellOption, charges int, spellLowest content.SpellLevel) (ChargedSpellPlanResult, error) {
	if !content.ValidSpellLevel(int(spellLowest)) {
		return nil, invalidInput("plan charged spell: invalid lowest level")
	}
	switch o := option.(type) {
	case ChargesPerLevel:
		if charges < 1 || charges > o.maximum {
			return ChargedSpellPlanRejected{reason: ChargesOutOfRange}, nil
		}
		return ChargedSpellPlan{spell: o.spell, charge: charges, lowest: o.lowest, castAt: o.lowest + content.SpellLevel(charges-1), stats: o.stats}, nil
	case FixedChargeCost:
		if charges != o.charges {
			return ChargedSpellPlanRejected{reason: ChargesNotAllowed}, nil
		}
		at := spellLowest
		if o.level.kind == 2 {
			at = o.level.level
		}
		if at < spellLowest {
			return ChargedSpellPlanRejected{reason: LevelBelowLowest}, nil
		}
		return ChargedSpellPlan{spell: o.spell, charge: o.charges, lowest: spellLowest, castAt: at, stats: o.stats}, nil
	}
	return nil, invalidInput("plan charged spell: nil or foreign option")
}

// Cast spends the plan's charges through ItemCharges.Spend and returns the new
// charges, the MagicItemChargesSpent event and the SpellFromItem (delivery
// CastFromItem, default components and terms, the lowest caster level;
// Concentration per requiresConcentration). Too few charges is
// MagicItemInsufficientCharges, unchanged. A plan of 0 charges spends nothing and
// has no event. It returns an error matching srderr.ErrInvalidInput for a zero
// plan, holder, blank instance or zero charges.
func (ChargedSpellCasting) Cast(charges ItemCharges, plan ChargedSpellPlan, holder Identifier, instance MagicItemInstanceID, requiresConcentration bool) (ChargedSpellCastResult, error) {
	if !plan.spell.Valid() || holder.IsZero() || !instance.Valid() || charges.IsZero() {
		return nil, invalidInput("cast charged spell: zero plan, holder or charges, or blank instance")
	}
	spell, err := NewSpellFromItem(plan.spell, plan.lowest, plan.castAt, NoComponentsRequired, StandardCastingTerms, CastFromItem, requiresConcentration)
	if err != nil {
		return nil, err
	}
	out := ChargedSpellCastApplied{charges: charges, spell: spell, stats: plan.stats}
	if plan.charge == 0 {
		return out, nil
	}
	res, err := charges.Spend(holder, instance, plan.charge)
	if err != nil {
		return nil, err
	}
	switch r := res.(type) {
	case ItemChargesSpendApplied:
		out.charges, out.event, out.spent = r.Charges, r.Event, true
		return out, nil
	case MagicItemInsufficientCharges:
		return r, nil
	}
	return nil, invalidInput("cast charged spell: unknown spend result") // unreachable: Spend has two variants
}
