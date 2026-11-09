package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

const maxKnownForms = 8

// KnownForms are the Beast forms a Druid knows for Wild Shape, distinct and in the order learned (SRD 5.2.1, "Level 2: Wild
// Shape", p. 42). It is immutable and comparable with ==; the zero value knows none. Safe for concurrent use.
type KnownForms struct {
	ids [maxKnownForms]content.MonsterID
	n   int
}

// Len returns how many forms are known.
func (k KnownForms) Len() int { return k.n }

// IDs returns the known forms in the order learned, as a new slice.
func (k KnownForms) IDs() []content.MonsterID { return slices.Clone(k.ids[:k.n]) }

// Contains reports whether the form is known.
func (k KnownForms) Contains(id content.MonsterID) bool { return slices.Contains(k.ids[:k.n], id) }

// with returns k with the form added; the caller has checked the bounds and that it is new.
func (k KnownForms) with(id content.MonsterID) KnownForms {
	k.ids[k.n] = id
	k.n++
	return k
}

// replace returns k with the form old replaced, in place, by the form new.
func (k KnownForms) replace(old, replacement content.MonsterID) KnownForms {
	if i := slices.Index(k.ids[:k.n], old); i >= 0 {
		k.ids[i] = replacement
	}
	return k
}

// DruidCharacterFacts are the facts about the character that the Druid operations read: its id, its Druid level, the
// subclass it holds (empty for none) and its Proficiency Bonus from the total level (class-druid.md FR-167, FR-152). The
// Druid state stores none of them. Immutable, comparable with ==; the zero value is invalid.
type DruidCharacterFacts struct {
	id       Identifier
	level    DruidLevel
	subclass content.SubclassID
	bonus    ProficiencyBonus
}

// NewDruidCharacterFacts returns the facts. It returns an error matching srderr.ErrInvalidInput for a zero id, a level
// outside 1 to 20 or a zero Proficiency Bonus. It never panics.
func NewDruidCharacterFacts(id Identifier, level int, subclass content.SubclassID, bonus ProficiencyBonus) (DruidCharacterFacts, error) {
	l, err := NewDruidLevel(level)
	if err != nil || id.IsZero() || bonus.IsZero() {
		return DruidCharacterFacts{}, invalidInput("Druid facts with a zero id, a level outside 1 to 20 or a zero Proficiency Bonus")
	}
	return DruidCharacterFacts{id: id, level: l, subclass: subclass, bonus: bonus}, nil
}

// CharacterID returns the id of the character.
func (f DruidCharacterFacts) CharacterID() Identifier { return f.id }

// Level returns the Druid level.
func (f DruidCharacterFacts) Level() DruidLevel { return f.level }

// Subclass returns the subclass the character holds, or "" for none.
func (f DruidCharacterFacts) Subclass() content.SubclassID { return f.subclass }

// ProficiencyBonus returns the Proficiency Bonus of the total character level.
func (f DruidCharacterFacts) ProficiencyBonus() ProficiencyBonus { return f.bonus }

// IsZero reports whether f is the invalid zero value.
func (f DruidCharacterFacts) IsZero() bool { return f.level.IsZero() }

func (f DruidCharacterFacts) lv() int { return f.level.Value() }

// hasCircle reports whether the character holds the Circle of the Land.
func (f DruidCharacterFacts) hasCircle() bool { return f.subclass == CircleOfTheLandSubclassID }

// hasWildShape reports whether Wild Shape is granted (Druid level 2).
func (f DruidCharacterFacts) hasWildShape() bool { return f.lv() >= 2 }

// restWindow is a set of RestWindowPermission, one bit each. The zero value is the empty set.
type restWindow uint8

func (w restWindow) has(p RestWindowPermission) bool           { return p.Valid() && w&(1<<uint(p)) != 0 }
func (w restWindow) with(p RestWindowPermission) restWindow    { return w | 1<<uint(p) }
func (w restWindow) without(p RestWindowPermission) restWindow { return w &^ (1 << uint(p)) }

// DruidStateFields are the fields of a DruidFeatureState, for a host restoring stored state. Construct it with a keyed
// literal; the zero value is the state of a level 1 Druid with no choice made.
type DruidStateFields struct {
	PrimalOrder   PrimalOrder
	ElementalFury ElementalFury
	LandType      LandType
	// WildShapeUsesRemaining is the unexpended Wild Shape uses.
	WildShapeUsesRemaining int
	KnownForms             []content.MonsterID
	// ActiveForm is the Beast form assumed, or "" for none.
	ActiveForm content.MonsterID
	// FormReplacements counts the known forms replaced since the last Long Rest.
	FormReplacements int
	// WildResurgenceSlotForUseThisTurn and PrimalStrikeThisTurn are the once-per-turn flags.
	WildResurgenceSlotForUseThisTurn bool
	PrimalStrikeThisTurn             bool
	// The once-until-Long-Rest flags.
	WildResurgenceUseForSlot bool
	NaturalRecoveryCast      bool
	NaturalRecoveryRecovery  bool
	NatureMagician           bool
	RestWindow               []RestWindowPermission
	// FamiliarPresent says a Wild Companion familiar is present.
	FamiliarPresent bool
	// SanctuaryActive says a Nature's Sanctuary Cube is active.
	SanctuaryActive bool
}

// DruidFeatureState is the Druid's class feature state: the Primal Order, Elemental Fury and land choices, the Wild Shape
// uses, known forms and active form, the once-per-turn and once-until-Long-Rest flags, the open rest window, the Wild Companion
// familiar and the Nature's Sanctuary Cube (class-druid.md FR-162). It holds no level and no subclass (FR-167) and no spell
// slot (FR-160). It is stored under DruidFeatureStateKey; an absent state is the zero value. Immutable, comparable with ==;
// the zero value is the state of a level 1 Druid with no choice made. Safe for concurrent use.
type DruidFeatureState struct {
	order      PrimalOrder
	fury       ElementalFury
	land       LandType
	uses       int
	known      KnownForms
	active     content.MonsterID
	swaps      int
	wrTurn     bool
	psTurn     bool
	wrLongRest bool
	nrCast     bool
	nrRecover  bool
	nature     bool
	window     restWindow
	familiar   bool
	sanctuary  bool
}

// DruidFeatureStateKey returns the FeatureStateKey named druid under which a character holds its DruidFeatureState.
func DruidFeatureStateKey() FeatureStateKey[DruidFeatureState] {
	return MustFeatureStateKey[DruidFeatureState]("druid")
}

// NewDruidFeatureState returns the state with the given fields, validated against the facts and the tables
// (class-druid.md FR-141, FR-142). It returns an error matching srderr.ErrInvalidInput that is a *DruidInvariantError
// listing every violation: remaining uses negative or above the maximum of the level, a feature state below the level or
// without the subclass that grants it, an active Beast form, Cube or open permission without its feature, known forms that
// are repeated, invalid or too many, an enum outside its set. It never panics.
func NewDruidFeatureState(f DruidStateFields, facts DruidCharacterFacts, tables DruidFeatureTables) (DruidFeatureState, error) {
	s := DruidFeatureState{order: f.PrimalOrder, fury: f.ElementalFury, land: f.LandType, uses: f.WildShapeUsesRemaining, active: f.ActiveForm,
		swaps: f.FormReplacements, wrTurn: f.WildResurgenceSlotForUseThisTurn, psTurn: f.PrimalStrikeThisTurn, wrLongRest: f.WildResurgenceUseForSlot,
		nrCast: f.NaturalRecoveryCast, nrRecover: f.NaturalRecoveryRecovery, nature: f.NatureMagician, familiar: f.FamiliarPresent, sanctuary: f.SanctuaryActive}
	var vs []DruidInvariantViolation
	if len(f.KnownForms) > maxKnownForms {
		vs = append(vs, DruidViolationKnownForms)
	} else {
		for _, id := range f.KnownForms {
			if !id.Valid() || s.known.Contains(id) {
				vs = append(vs, DruidViolationKnownForms)
			} else {
				s.known = s.known.with(id)
			}
		}
	}
	for _, p := range f.RestWindow {
		if !p.Valid() {
			vs = append(vs, DruidViolationUndefinedValue)
			continue
		}
		s.window = s.window.with(p)
	}
	vs = append(vs, s.violations(facts.lv(), facts.hasCircle(), tables)...)
	if err := druidViolations(vs); err != nil {
		return DruidFeatureState{}, err
	}
	return s, nil
}

// MustDruidFeatureState is NewDruidFeatureState for fields known to be valid. It panics with the error
// NewDruidFeatureState returns, and only then.
func MustDruidFeatureState(f DruidStateFields, facts DruidCharacterFacts, tables DruidFeatureTables) DruidFeatureState {
	s, err := NewDruidFeatureState(f, facts, tables)
	if err != nil {
		panic(err)
	}
	return s
}

// violations lists the invariants s breaks at the Druid level, with or without the Circle of the Land, for the tables.
func (s DruidFeatureState) violations(level int, circle bool, tables DruidFeatureTables) []DruidInvariantViolation {
	var vs []DruidInvariantViolation
	wildShape := level >= 2
	if s.uses < 0 || s.uses > tables.WildShapeUses(level) {
		vs = append(vs, DruidViolationUsesOutOfRange)
	}
	if s.order != 0 && !s.order.Valid() || s.fury != 0 && !s.fury.Valid() || s.land != 0 && !s.land.Valid() {
		vs = append(vs, DruidViolationUndefinedValue)
	}
	if s.fury != 0 && level < 7 || s.nature && level < 20 || (s.wrTurn || s.wrLongRest) && level < 5 || s.psTurn && (level < 7 || s.fury != ElementalFuryPrimalStrike) ||
		(s.land != 0 || s.nrCast || s.nrRecover || s.window.has(RestWindowLandTypeChange)) && (!circle || level < 3) ||
		(s.nrCast || s.nrRecover || s.window.has(RestWindowNaturalRecoveryRecovery)) && (!circle || level < 6) ||
		(s.known.Len() > 0 || s.swaps > 0 || s.familiar || s.window.has(RestWindowKnownFormReplacement)) && !wildShape ||
		s.sanctuary && (!circle || level < 14) {
		vs = append(vs, DruidViolationFeatureBeforeLevel)
	}
	if s.active != "" && (!wildShape) {
		vs = append(vs, DruidViolationFormWithoutFeature)
	}
	if s.known.Len() > tables.KnownFormsAt(level) || s.swaps < 0 {
		vs = append(vs, DruidViolationKnownForms)
	}
	return vs
}

// check returns an error matching srderr.ErrInvalidInput when s breaks an invariant for the facts and tables, or when the
// facts or tables are the invalid zero values.
func (s DruidFeatureState) check(facts DruidCharacterFacts, tables DruidFeatureTables) error {
	if facts.IsZero() || tables.IsZero() {
		return invalidInput("Druid operation with zero facts or zero tables")
	}
	return druidViolations(s.violations(facts.lv(), facts.hasCircle(), tables))
}

// PrimalOrder returns the chosen Primal Order, or the zero value for none.
func (s DruidFeatureState) PrimalOrder() PrimalOrder { return s.order }

// ExtraCantrips returns the extra-cantrip allowance for spellcasting (class-druid.md FR-051): 1 for a Magician, else 0.
func (s DruidFeatureState) ExtraCantrips() int {
	if s.order == PrimalOrderMagician {
		return 1
	}
	return 0
}

// ElementalFury returns the chosen Elemental Fury option, or the zero value for none.
func (s DruidFeatureState) ElementalFury() ElementalFury { return s.fury }

// LandType returns the current land type, or the zero value for none.
func (s DruidFeatureState) LandType() LandType { return s.land }

// WildShapeUsesRemaining returns the unexpended Wild Shape uses.
func (s DruidFeatureState) WildShapeUsesRemaining() int { return s.uses }

// KnownForms returns the known Beast forms.
func (s DruidFeatureState) KnownForms() KnownForms { return s.known }

// ActiveForm returns the Beast form assumed, and false when the Druid is not in a Beast form.
func (s DruidFeatureState) ActiveForm() (content.MonsterID, bool) { return s.active, s.active != "" }

// FormReplacements returns the known forms replaced since the last Long Rest.
func (s DruidFeatureState) FormReplacements() int { return s.swaps }

// WildResurgenceSlotForUseThisTurn reports whether the slot-for-use exchange was made this turn.
func (s DruidFeatureState) WildResurgenceSlotForUseThisTurn() bool { return s.wrTurn }

// PrimalStrikeThisTurn reports whether Primal Strike was used this turn.
func (s DruidFeatureState) PrimalStrikeThisTurn() bool { return s.psTurn }

// WildResurgenceUseForSlot reports whether the use-for-slot exchange was made since the last Long Rest.
func (s DruidFeatureState) WildResurgenceUseForSlot() bool { return s.wrLongRest }

// NaturalRecoveryCast reports whether the Natural Recovery casting was used since the last Long Rest.
func (s DruidFeatureState) NaturalRecoveryCast() bool { return s.nrCast }

// NaturalRecoveryRecovery reports whether Natural Recovery slot recovery was used since the last Long Rest.
func (s DruidFeatureState) NaturalRecoveryRecovery() bool { return s.nrRecover }

// NatureMagician reports whether Nature Magician was used since the last Long Rest.
func (s DruidFeatureState) NatureMagician() bool { return s.nature }

// RestWindow returns the open permissions, in the order land type, known form, recovery, as a new slice.
func (s DruidFeatureState) RestWindow() []RestWindowPermission {
	out := []RestWindowPermission{}
	for p := RestWindowLandTypeChange; p.Valid(); p++ {
		if s.window.has(p) {
			out = append(out, p)
		}
	}
	return out
}

// FamiliarPresent reports whether a Wild Companion familiar is present.
func (s DruidFeatureState) FamiliarPresent() bool { return s.familiar }

// SanctuaryActive reports whether a Nature's Sanctuary Cube is active.
func (s DruidFeatureState) SanctuaryActive() bool { return s.sanctuary }

// Equal reports whether other is a DruidFeatureState with the same fields.
func (s DruidFeatureState) Equal(other FeatureState) bool {
	o, ok := other.(DruidFeatureState)
	return ok && o == s
}

var _ FeatureState = DruidFeatureState{}
