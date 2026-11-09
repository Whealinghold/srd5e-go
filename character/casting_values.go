package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// ComponentKind is a kind of spell component (SRD 5.2.1, "Components", pp.
// 105-106). The zero value is invalid. Comparable and safe for concurrent use.
type ComponentKind int

// The component kinds.
const (
	// ComponentVerbal is a Verbal (V) component.
	ComponentVerbal ComponentKind = iota + 1
	// ComponentSomatic is a Somatic (S) component.
	ComponentSomatic
	// ComponentMaterial is a Material (M) component.
	ComponentMaterial
)

// String returns "verbal", "somatic", "material" or "unknown".
func (k ComponentKind) String() string {
	switch k {
	case ComponentVerbal:
		return "verbal"
	case ComponentSomatic:
		return "somatic"
	case ComponentMaterial:
		return "material"
	}
	return "unknown"
}

// CastMode is how the caller says a spell is being cast; CastSpell derives the
// CastSource from it (SRD 5.2.1, "Casting without Slots", p. 104). The zero
// value is invalid. Comparable and safe for concurrent use.
type CastMode int

// The cast modes.
const (
	// CastModeNormal: a prepared spell with a slot, or a known cantrip.
	CastModeNormal CastMode = iota + 1
	// CastModeRitual: the Ritual version of a spell, no slot.
	CastModeRitual
	// CastModeFeatureGranted: a feature lets the caster cast the spell without a slot.
	CastModeFeatureGranted
	// CastModeSpellScroll: the spell is read from a Spell Scroll.
	CastModeSpellScroll
	// CastModeDuplicated: Wish duplicates the spell, which "simply takes effect".
	CastModeDuplicated
)

// String returns "normal", "ritual", "feature_granted", "spell_scroll",
// "duplicated" or "unknown".
func (m CastMode) String() string {
	names := [...]string{"normal", "ritual", "feature_granted", "spell_scroll", "duplicated"}
	if !m.Valid() {
		return "unknown"
	}
	return names[m-1]
}

// Valid reports whether m is one of the five modes.
func (m CastMode) Valid() bool { return m >= CastModeNormal && m <= CastModeDuplicated }

// CastRejectionKind is one reason a cast was refused. The values are in the
// fixed order a CastRejection reports them (FR-080). The zero value is invalid.
// Comparable and safe for concurrent use.
type CastRejectionKind int

// The rejection kinds, in reporting order.
const (
	// CastRejectionCannotCastSpells: the caster cannot cast (for example in Wild Shape).
	CastRejectionCannotCastSpells CastRejectionKind = iota + 1
	// CastRejectionSpellUnavailable: the host lists the spell as one the caster cannot cast.
	CastRejectionSpellUnavailable
	// CastRejectionUntrainedArmor: armor worn without training.
	CastRejectionUntrainedArmor
	// CastRejectionCastingAlreadyInProgress: a longer casting is under way.
	CastRejectionCastingAlreadyInProgress
	// CastRejectionNotKnown: a cantrip the class does not know.
	CastRejectionNotKnown
	// CastRejectionNotPrepared: a spell the class has not prepared.
	CastRejectionNotPrepared
	// CastRejectionNotInSpellbook: a Wizard Ritual not in the spellbook.
	CastRejectionNotInSpellbook
	// CastRejectionSpellbookNotReadable: the spellbook cannot be read for Ritual Adept.
	CastRejectionSpellbookNotReadable
	// CastRejectionNotARitual: the spell lacks the Ritual tag.
	CastRejectionNotARitual
	// CastRejectionTriggerNotMet: the casting time's trigger has not occurred.
	CastRejectionTriggerNotMet
	// CastRejectionMissingVerbal: the caster cannot speak.
	CastRejectionMissingVerbal
	// CastRejectionMissingSomatic: no hand available for gestures.
	CastRejectionMissingSomatic
	// CastRejectionMissingMaterial: no way to provide the Material component.
	CastRejectionMissingMaterial
	// CastRejectionSlotAlreadySpentThisTurn: a slot was already spent this turn.
	CastRejectionSlotAlreadySpentThisTurn
	// CastRejectionSlotLevelTooLow: the chosen slot is below the spell's level.
	CastRejectionSlotLevelTooLow
	// CastRejectionNoSlotAvailable: the chosen slot level has none remaining.
	CastRejectionNoSlotAvailable
	// CastRejectionCastingNotInProgress: CompleteCasting with nothing to complete; never listed with another.
	CastRejectionCastingNotInProgress
)

// String returns the snake_case name or "unknown".
func (k CastRejectionKind) String() string {
	names := [...]string{"cannot_cast_spells", "spell_unavailable", "untrained_armor", "casting_already_in_progress", "not_known", "not_prepared",
		"not_in_spellbook", "spellbook_not_readable", "not_a_ritual", "trigger_not_met", "missing_component_verbal", "missing_component_somatic",
		"missing_component_material", "slot_already_spent_this_turn", "slot_level_too_low", "no_slot_available", "casting_not_in_progress"}
	if k < CastRejectionCannotCastSpells || int(k) > len(names) {
		return "unknown"
	}
	return names[k-1]
}

// CastRejection is the ordered, repeat-free list of reasons a cast was refused,
// in the fixed order of the CastRejectionKind values. It is never empty when it
// comes from a use case. It holds a slice, so compare two with reflect.DeepEqual.
type CastRejection struct{ reasons []CastRejectionKind }

// newCastRejection sorts and de-duplicates kinds.
func newCastRejection(kinds []CastRejectionKind) CastRejection {
	k := slices.Clone(kinds)
	slices.Sort(k)
	return CastRejection{reasons: slices.Compact(k)}
}

// Reasons returns a new slice of the reasons in the fixed order.
func (r CastRejection) Reasons() []CastRejectionKind { return slices.Clone(r.reasons) }

// First returns the first reason, the zero kind for an empty rejection.
func (r CastRejection) First() CastRejectionKind {
	if len(r.reasons) == 0 {
		return 0
	}
	return r.reasons[0]
}

// Has reports whether k is among the reasons.
func (r CastRejection) Has(k CastRejectionKind) bool { return slices.Contains(r.reasons, k) }

// SpellCastTerms are the figures a cast without a casting class needs: the spell
// save DC, the spell attack bonus and, for a cantrip, the level it scales with
// (SRD 5.2.1, "Spell Scroll", p. 244; "Wish", p. 175). Build one with
// NewSpellCastTerms. Comparable and safe for concurrent use. The zero value is
// invalid.
type SpellCastTerms struct{ saveDC, attackBonus, cantripLevel int }

// NewSpellCastTerms returns the terms. cantripScalingLevel is 0 for a level 1+
// spell and the level the cantrip scales with (1 to 20) for a cantrip. It
// returns an error matching srderr.ErrInvalidInput for a save DC below 1 or a
// cantrip scaling level below 0 or above 20. It never panics.
func NewSpellCastTerms(saveDC, attackBonus, cantripScalingLevel int) (SpellCastTerms, error) {
	if saveDC < 1 || cantripScalingLevel < 0 || cantripScalingLevel > MaxLevel {
		return SpellCastTerms{}, invalidInput("spell cast terms: save DC below 1 or cantrip scaling level outside 0 to 20")
	}
	return SpellCastTerms{saveDC: saveDC, attackBonus: attackBonus, cantripLevel: cantripScalingLevel}, nil
}

// SaveDC returns the spell save DC.
func (t SpellCastTerms) SaveDC() int { return t.saveDC }

// AttackBonus returns the spell attack bonus.
func (t SpellCastTerms) AttackBonus() int { return t.attackBonus }

// CantripScalingLevel returns the level a cantrip scales with, 0 when none.
func (t SpellCastTerms) CantripScalingLevel() int { return t.cantripLevel }

// IsZero reports whether t is the invalid zero value.
func (t SpellCastTerms) IsZero() bool { return t.saveDC == 0 }

// CastingFacts are the facts about the caster and the moment that the casting
// rules read; the host reports them. Every flag false is a caster who cannot
// cast. Use NewCastingContext to validate them.
type CastingFacts struct {
	SlotSpentThisTurn  bool // a spell slot was already spent this turn ("One Spell with a Spell Slot per Turn")
	UntrainedArmorWorn bool // Light, Medium or Heavy armor is worn without training
	CanCastSpells      bool // false in Wild Shape or while raging
	CanSpeak           bool
	CanMakeGestures    bool
	HasFreeHand        bool
	HasMaterials       bool
	HasComponentPouch  bool
	CanReadSpellbook   bool
	TriggerMet         bool
	// HeldFocus is the Spellcasting Focus held; the zero kind means none.
	HeldFocus SpellcastingFocusKind
	// FeatureFocusKinds are the feature-granted kinds the caster may use as a
	// focus (FocusPactWeapon, FocusBookOfShadows); never a class kind.
	FeatureFocusKinds []SpellcastingFocusKind
	// UnavailableSpells are spells the caster can never cast now, read as a set.
	UnavailableSpells []content.SpellID
}

// CastingContext is validated CastingFacts (SRD 5.2.1, "Casting Spells", pp.
// 104-106). It is immutable. It holds slices, so compare two with
// reflect.DeepEqual. The zero value is a caster who cannot cast.
type CastingContext struct{ f CastingFacts }

// NewCastingContext validates f and copies its slices. It returns an error
// matching srderr.ErrInvalidInput for an invalid HeldFocus, a FeatureFocusKinds
// entry that is not a feature-granted kind or an invalid spell id. It never panics.
func NewCastingContext(f CastingFacts) (CastingContext, error) {
	if f.HeldFocus != 0 && !f.HeldFocus.Valid() {
		return CastingContext{}, invalidInput("casting context: invalid held focus")
	}
	for _, k := range f.FeatureFocusKinds {
		if !k.Valid() || !k.IsFeatureGranted() {
			return CastingContext{}, invalidInput("casting context: feature focus kinds hold only feature-granted kinds")
		}
	}
	for _, id := range f.UnavailableSpells {
		if !id.Valid() {
			return CastingContext{}, invalidInput("casting context: invalid unavailable spell id")
		}
	}
	f.FeatureFocusKinds = slices.Clone(f.FeatureFocusKinds)
	f.UnavailableSpells = slices.Clone(f.UnavailableSpells)
	return CastingContext{f: f}, nil
}

// Facts returns the facts with new slices.
func (c CastingContext) Facts() CastingFacts {
	f := c.f
	f.FeatureFocusKinds = slices.Clone(f.FeatureFocusKinds)
	f.UnavailableSpells = slices.Clone(f.UnavailableSpells)
	return f
}

// CastReportParts are the fields of a CastReport; the zero value of an optional
// field means it is absent.
type CastReportParts struct {
	Spell        content.SpellID
	Class        SpellcastingClass // absent: no casting class (scroll, monster, class-less cast)
	Source       CastSource
	SlotPool     SpellSlotPool // invalid pool: no slot spent
	SlotLevel    SlotLevel
	CastAtLevel  int // 0: absent
	ScalingLevel int // 0: absent
	SaveDC       *SpellSaveDC
	AttackBonus  *SpellAttackBonus
	CountsAsSlot bool
	HasSlotSpent bool // set with SlotPool and SlotLevel
}

// CastReport is what a successful cast reports: the spell, the casting class,
// the source, the slot spent, the level it was cast at and scaled with, the spell
// save DC and attack bonus, and whether it counts as a slot spell (SRD 5.2.1,
// "Spells", p. 106). Class, slot, levels, DC and bonus may be absent; read each
// absent field as not applicable. Comparable and safe for concurrent use.
type CastReport struct {
	spell      content.SpellID
	class      SpellcastingClass
	source     CastSource
	pool       SpellSlotPool
	slotLevel  SlotLevel
	castAt     int
	scaling    int
	saveDC     SpellSaveDC
	hasAttack  bool
	attack     SpellAttackBonus
	countsSlot bool
}

// NewCastReport returns the report for parts. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell id or source, a class that is set
// but invalid, a slot set without a valid pool and level, or a level outside 0
// to 20. It never panics.
func NewCastReport(p CastReportParts) (CastReport, error) {
	if !p.Spell.Valid() || p.Source < CastSourceSlot || p.Source > CastSourceDuplicated || (p.Class != 0 && !p.Class.Valid()) ||
		p.CastAtLevel < 0 || p.CastAtLevel > MaxLevel || p.ScalingLevel < 0 || p.ScalingLevel > MaxLevel ||
		(p.HasSlotSpent && (!p.SlotPool.Valid() || p.SlotLevel.IsZero())) {
		return CastReport{}, invalidInput("cast report: invalid spell, source, class, slot or level")
	}
	r := CastReport{spell: p.Spell, class: p.Class, source: p.Source, castAt: p.CastAtLevel, scaling: p.ScalingLevel, countsSlot: p.CountsAsSlot}
	if p.HasSlotSpent {
		r.pool, r.slotLevel = p.SlotPool, p.SlotLevel
	}
	if p.SaveDC != nil {
		r.saveDC = *p.SaveDC
	}
	if p.AttackBonus != nil {
		r.hasAttack, r.attack = true, *p.AttackBonus
	}
	return r, nil
}

// Spell returns the spell id.
func (r CastReport) Spell() content.SpellID { return r.spell }

// Class returns the casting class and whether there is one.
func (r CastReport) Class() (SpellcastingClass, bool) { return r.class, r.class.Valid() }

// Source returns the source.
func (r CastReport) Source() CastSource { return r.source }

// Slot returns the pool and level of the slot spent and whether one was.
func (r CastReport) Slot() (SpellSlotPool, SlotLevel, bool) {
	return r.pool, r.slotLevel, r.pool.Valid()
}

// CastAtLevel returns the level the spell was cast at and whether there is one.
func (r CastReport) CastAtLevel() (int, bool) { return r.castAt, r.castAt > 0 }

// ScalingLevel returns the level the spell scales with and whether there is one.
func (r CastReport) ScalingLevel() (int, bool) { return r.scaling, r.scaling > 0 }

// SaveDC returns the spell save DC and whether there is one.
func (r CastReport) SaveDC() (SpellSaveDC, bool) { return r.saveDC, r.saveDC.dc != 0 }

// AttackBonus returns the spell attack bonus and whether there is one.
func (r CastReport) AttackBonus() (SpellAttackBonus, bool) { return r.attack, r.hasAttack }

// CountsAsSlotSpell reports whether the host sets SlotSpentThisTurn: a slot cast
// and a flagged feature-granted cast.
func (r CastReport) CountsAsSlotSpell() bool { return r.countsSlot }

// dcOf returns the spell save DC with value n.
func dcOf(n int) SpellSaveDC { return SpellSaveDC{dc: d20.DifficultyClass(n)} }
