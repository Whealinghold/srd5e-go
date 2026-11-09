package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-druid.md FR-167, FR-152 - the facts hold the id, the Druid level 1 to 20, the optional subclass and the Proficiency Bonus of
// the total level; NewDruidCharacterFacts refuses a zero id, a level outside 1 to 20 and a zero bonus; the Druid 3 / Fighter 5 case keeps its +3.
// Path: per case (unhappy: invalid input, boundary, rule interaction).
func TestNewDruidCharacterFacts(t *testing.T) {
	t.Parallel()
	bonus := character.ProficiencyBonusFor(character.MustLevel(8))
	for _, c := range []struct {
		label string
		id    character.Identifier
		level int
		bonus character.ProficiencyBonus
		ok    bool
	}{
		{"level 1", druTestID, 1, bonus, true}, {"level 20", druTestID, 20, bonus, true}, {"level 0", druTestID, 0, bonus, false}, {"level 21", druTestID, 21, bonus, false},
		{"zero id", character.Identifier{}, 3, bonus, false}, {"zero bonus", druTestID, 3, character.ProficiencyBonus{}, false},
	} {
		f, err := character.NewDruidCharacterFacts(c.id, c.level, character.CircleOfTheLandSubclassID, c.bonus)
		if (err == nil) != c.ok {
			t.Errorf("%s: error = %v, want ok %v", c.label, err, c.ok)
		}
		if err != nil {
			wantInvalid(t, err, c.label)
			continue
		}
		if f.CharacterID() != c.id || f.Level().Value() != c.level || f.Subclass() != character.CircleOfTheLandSubclassID || f.ProficiencyBonus() != c.bonus || f.IsZero() {
			t.Errorf("%s: facts = %+v", c.label, f)
		}
	}
	multi := druFactsBonus(t, 3, true, bonus)
	if multi.Level().Value() != 3 || multi.ProficiencyBonus().Value() != 3 {
		t.Errorf("Druid 3 / Fighter 5 facts = level %d, bonus %d; want 3 and +3", multi.Level().Value(), multi.ProficiencyBonus().Value())
	}
	if !(character.DruidCharacterFacts{}).IsZero() {
		t.Error("the zero facts are not zero")
	}
}

// Requirement: doc/specs/class-druid.md FR-162, FR-170, D-2 - DruidFeatureState is stored under the key "druid" and read back, an absent state is the zero
// value (a level 1 Druid with no choice), it compares by value, and is not equal to another feature's state.
// Path: happy.
func TestDruidFeatureStateStoredUnderItsKey(t *testing.T) {
	t.Parallel()
	if character.DruidFeatureStateKey().Name() != "druid" {
		t.Errorf("key = %q, want druid", character.DruidFeatureStateKey().Name())
	}
	var states character.FeatureStates
	if _, ok, err := character.ReadFeatureState(states, character.DruidFeatureStateKey()); ok || err != nil {
		t.Fatalf("an empty set holds a Druid state: %v, %v", ok, err)
	}
	s := druState(t, 3, false, character.DruidStateFields{PrimalOrder: character.PrimalOrderWarden, WildShapeUsesRemaining: 1})
	states = must(character.SetFeatureState(states, character.DruidFeatureStateKey(), s))
	got, ok, err := character.ReadFeatureState(states, character.DruidFeatureStateKey())
	if !ok || err != nil || got != s || !got.Equal(s) || got.Equal(character.DruidFeatureState{}) || got.Equal(character.InitialRogueState()) {
		t.Errorf("read = %v, %v, %v", got, ok, err)
	}
}

// Requirement: doc/specs/class-druid.md FR-141, FR-142 - NewDruidFeatureState accepts a consistent state and lists every violated invariant otherwise as a
// *DruidInvariantError matching ErrInvalidInput: remaining uses negative or above the maximum, a feature state below the level or without the subclass
// that grants it (Wild Shape state, flags, Elemental Fury, land type, Natural Recovery, Nature Magician, a Cube, an open permission), an active form without
// Wild Shape, known forms repeated, invalid or too many, and an undefined enum; MustDruidFeatureState panics for them.
// Path: per case (happy / unhappy: invalid input, boundary at each granting level).
func TestNewDruidFeatureStateInvariants(t *testing.T) {
	t.Parallel()
	tables := druTables(t)
	forms := druIDs("wolf", "rat", "spider", "riding_horse")
	full := character.DruidStateFields{PrimalOrder: character.PrimalOrderMagician, ElementalFury: character.ElementalFuryPrimalStrike, LandType: character.LandPolar,
		WildShapeUsesRemaining: 3, KnownForms: forms, ActiveForm: "wolf", FormReplacements: 1, WildResurgenceSlotForUseThisTurn: true, PrimalStrikeThisTurn: true,
		WildResurgenceUseForSlot: true, NaturalRecoveryCast: true, NaturalRecoveryRecovery: true, FamiliarPresent: true, SanctuaryActive: true,
		RestWindow: []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowKnownFormReplacement, character.RestWindowNaturalRecoveryRecovery}}
	with := func(f character.DruidStateFields, mutate func(*character.DruidStateFields)) character.DruidStateFields {
		mutate(&f)
		return f
	}
	type problem = character.DruidInvariantViolation
	for _, c := range []struct {
		label  string
		level  int
		circle bool
		f      character.DruidStateFields
		want   []problem
	}{
		{"everything at level 14", 14, true, full, nil},
		{"zero state at level 1", 1, false, character.DruidStateFields{}, nil},
		{"level 20", 20, true, with(full, func(f *character.DruidStateFields) { f.NatureMagician = true; f.WildShapeUsesRemaining = 0 }), nil},
		{"uses above the maximum", 5, false, character.DruidStateFields{WildShapeUsesRemaining: 3}, []problem{character.DruidViolationUsesOutOfRange}},
		{"uses at the maximum", 5, false, character.DruidStateFields{WildShapeUsesRemaining: 2}, nil},
		{"negative uses", 5, false, character.DruidStateFields{WildShapeUsesRemaining: -1}, []problem{character.DruidViolationUsesOutOfRange}},
		{"uses at level 1", 1, false, character.DruidStateFields{WildShapeUsesRemaining: 1}, []problem{character.DruidViolationUsesOutOfRange}},
		{"known form at level 1", 1, false, character.DruidStateFields{KnownForms: druIDs("wolf")}, []problem{character.DruidViolationFeatureBeforeLevel, character.DruidViolationKnownForms}},
		{"known form at level 2", 2, false, character.DruidStateFields{KnownForms: druIDs("wolf")}, nil},
		{"familiar at level 1", 1, false, character.DruidStateFields{FamiliarPresent: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"replacement at level 1", 1, false, character.DruidStateFields{FormReplacements: 1}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"negative replacements", 2, false, character.DruidStateFields{FormReplacements: -1}, []problem{character.DruidViolationKnownForms}},
		{"form window at level 1", 1, false, character.DruidStateFields{RestWindow: []character.RestWindowPermission{character.RestWindowKnownFormReplacement}}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"active form at level 1", 1, false, character.DruidStateFields{ActiveForm: "wolf"}, []problem{character.DruidViolationFormWithoutFeature}},
		{"active form at level 2", 2, false, character.DruidStateFields{ActiveForm: "wolf", KnownForms: druIDs("wolf")}, nil},
		{"active form not known", 2, false, character.DruidStateFields{ActiveForm: "wolf"}, nil},
		{"Wild Resurgence flag at 4", 4, false, character.DruidStateFields{WildResurgenceUseForSlot: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Wild Resurgence flag at 5", 5, false, character.DruidStateFields{WildResurgenceUseForSlot: true, WildResurgenceSlotForUseThisTurn: true}, nil},
		{"turn flag at 4", 4, false, character.DruidStateFields{WildResurgenceSlotForUseThisTurn: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Elemental Fury at 6", 6, false, character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Elemental Fury at 7", 7, false, character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike, PrimalStrikeThisTurn: true}, nil},
		{"Primal Strike flag without the option", 7, false, character.DruidStateFields{ElementalFury: character.ElementalFuryPotentSpellcasting, PrimalStrikeThisTurn: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Primal Strike flag at 6", 6, false, character.DruidStateFields{PrimalStrikeThisTurn: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Nature Magician at 19", 19, false, character.DruidStateFields{NatureMagician: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"land without the subclass", 3, false, character.DruidStateFields{LandType: character.LandArid}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"land at 3", 3, true, character.DruidStateFields{LandType: character.LandArid}, nil},
		{"land window without the subclass", 6, false, character.DruidStateFields{RestWindow: []character.RestWindowPermission{character.RestWindowLandTypeChange}}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Natural Recovery flag at 5", 5, true, character.DruidStateFields{NaturalRecoveryCast: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Natural Recovery flag at 6", 6, true, character.DruidStateFields{NaturalRecoveryCast: true, NaturalRecoveryRecovery: true}, nil},
		{"recovery window at 5", 5, true, character.DruidStateFields{RestWindow: []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Cube at 13", 13, true, character.DruidStateFields{SanctuaryActive: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Cube without the subclass", 14, false, character.DruidStateFields{SanctuaryActive: true}, []problem{character.DruidViolationFeatureBeforeLevel}},
		{"Cube at 14", 14, true, character.DruidStateFields{SanctuaryActive: true}, nil},
		{"too many known forms", 2, false, character.DruidStateFields{KnownForms: druIDs("a", "b", "c", "d", "e")}, []problem{character.DruidViolationKnownForms}},
		{"nine known forms", 20, false, character.DruidStateFields{KnownForms: druIDs("a", "b", "c", "d", "e", "f", "g", "h", "i")}, []problem{character.DruidViolationKnownForms}},
		{"repeated known form", 4, false, character.DruidStateFields{KnownForms: druIDs("wolf", "wolf")}, []problem{character.DruidViolationKnownForms}},
		{"invalid form id", 4, false, character.DruidStateFields{KnownForms: druIDs("Not A Form")}, []problem{character.DruidViolationKnownForms}},
		{"undefined order", 1, false, character.DruidStateFields{PrimalOrder: 9}, []problem{character.DruidViolationUndefinedValue}},
		{"undefined fury", 7, false, character.DruidStateFields{ElementalFury: 9}, []problem{character.DruidViolationUndefinedValue}},
		{"undefined land", 3, true, character.DruidStateFields{LandType: 9}, []problem{character.DruidViolationUndefinedValue}},
		{"undefined permission", 3, true, character.DruidStateFields{RestWindow: []character.RestWindowPermission{9}}, []problem{character.DruidViolationUndefinedValue}},
		{"several problems", 1, false, character.DruidStateFields{WildShapeUsesRemaining: 9, PrimalOrder: 9, SanctuaryActive: true},
			[]problem{character.DruidViolationUsesOutOfRange, character.DruidViolationFeatureBeforeLevel, character.DruidViolationUndefinedValue}},
	} {
		s, err := character.NewDruidFeatureState(c.f, druFacts(t, c.level, c.circle), tables)
		if len(c.want) == 0 {
			if err != nil {
				t.Errorf("%s: error = %v, want none", c.label, err)
			} else if s.WildShapeUsesRemaining() != c.f.WildShapeUsesRemaining {
				t.Errorf("%s: uses = %d", c.label, s.WildShapeUsesRemaining())
			}
			continue
		}
		var ie *character.DruidInvariantError
		if !errors.Is(err, srderr.ErrInvalidInput) || !errors.As(err, &ie) || !slices.Equal(ie.Violations, c.want) {
			t.Errorf("%s: error = %v, want violations %v", c.label, err, c.want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustDruidFeatureState(invalid) did not panic")
		}
	}()
	character.MustDruidFeatureState(character.DruidStateFields{PrimalOrder: 9}, druFacts(t, 1, false), tables)
}

// Requirement: doc/specs/class-druid.md FR-148 - the accessors of a restored state return the fields, the known forms and the window as copies, the
// known forms keep their order and the window is listed in the order land type, known form, recovery.
// Path: happy and unhappy (immutability).
func TestDruidFeatureStateAccessors(t *testing.T) {
	t.Parallel()
	f := character.DruidStateFields{PrimalOrder: character.PrimalOrderMagician, ElementalFury: character.ElementalFuryPotentSpellcasting, LandType: character.LandTropical,
		WildShapeUsesRemaining: 2, KnownForms: druIDs("wolf", "rat"), ActiveForm: "rat", FormReplacements: 1, WildResurgenceSlotForUseThisTurn: true, PrimalStrikeThisTurn: false,
		WildResurgenceUseForSlot: true, NaturalRecoveryCast: true, NaturalRecoveryRecovery: true, NatureMagician: false, FamiliarPresent: true, SanctuaryActive: true,
		RestWindow: []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery, character.RestWindowLandTypeChange}}
	s := druState(t, 14, true, f)
	form, inForm := s.ActiveForm()
	if s.PrimalOrder() != f.PrimalOrder || s.ElementalFury() != f.ElementalFury || s.LandType() != f.LandType || s.WildShapeUsesRemaining() != 2 || form != "rat" || !inForm ||
		s.FormReplacements() != 1 || !s.WildResurgenceSlotForUseThisTurn() || s.PrimalStrikeThisTurn() || !s.WildResurgenceUseForSlot() || !s.NaturalRecoveryCast() ||
		!s.NaturalRecoveryRecovery() || s.NatureMagician() || !s.FamiliarPresent() || !s.SanctuaryActive() {
		t.Errorf("accessors disagree with the fields: %+v", s)
	}
	if !slices.Equal(s.RestWindow(), []character.RestWindowPermission{character.RestWindowLandTypeChange, character.RestWindowNaturalRecoveryRecovery}) {
		t.Errorf("RestWindow = %v", s.RestWindow())
	}
	k := s.KnownForms()
	ids := k.IDs()
	ids[0] = "changed"
	if k.Len() != 2 || !k.Contains("wolf") || k.Contains("changed") || k.Contains("bear") || !slices.Equal(k.IDs(), druIDs("wolf", "rat")) {
		t.Errorf("known forms = %v", k.IDs())
	}
	w := s.RestWindow()
	w[0] = 0
	if len(s.RestWindow()) != 2 || s.RestWindow()[0] != character.RestWindowLandTypeChange {
		t.Error("RestWindow shares storage")
	}
	if _, in := (character.DruidFeatureState{}).ActiveForm(); in || len((character.DruidFeatureState{}).RestWindow()) != 0 {
		t.Error("the zero state has an active form or an open window")
	}
}

// Requirement: doc/specs/class-druid.md FR-015, FR-038, FR-039, FR-042, FR-050, FR-051, FR-058 and table D (SRD 5.2.1, "Druid Features", p. 41; "Level 1: Primal Order", p. 42) - the Proficiency Bonus, Cantrips, Prepared Spells,
// highest Spell Slot level and Wild Shape columns of the bundled Features table agree with what spellcasting derives for the Druid at each of the 20 levels (a Druid 3 sees slots to level 2, 2 cantrips and 6 prepared spells
// whatever its other classes); the Magician supplies an extra cantrip allowance of 1 (3 cantrips at levels 1 to 3, 4 at 4 to 9, 5 from 10) and the Warden and no order none (2, 3, 4).
// Path: per case (the 20 levels): unhappy (rule interaction, boundary).
func TestDruidTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	def := druDef(t)
	tables := druTables(t)
	for n := 1; n <= 20; n++ {
		row := must(def.RowAt(n))
		sc := row.Spellcasting().(content.SlotSpellcastingRow)
		cantrips, _ := sc.Cantrips()
		highest := 0
		for i, s := range sc.SpellSlots() {
			if s > 0 {
				highest = i + 1
			}
		}
		if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() ||
			cantrips != character.CantripLimit(character.CasterDruid, n) || sc.PreparedSpells() != character.PreparedLimit(character.CasterDruid, n) ||
			highest != character.MaxPreparableLevel(character.CasterDruid, n) {
			t.Errorf("level %d row = bonus %d cantrips %d prepared %d highest slot %d; the rules say %d %d %d %d", n, row.ProficiencyBonus(), cantrips, sc.PreparedSpells(), highest,
				character.ProficiencyBonusFor(character.MustLevel(n)).Value(), character.CantripLimit(character.CasterDruid, n), character.PreparedLimit(character.CasterDruid, n),
				character.MaxPreparableLevel(character.CasterDruid, n))
		}
		magician := druState(t, n, false, character.DruidStateFields{PrimalOrder: character.PrimalOrderMagician})
		warden := druState(t, n, false, character.DruidStateFields{PrimalOrder: character.PrimalOrderWarden})
		want := 2
		switch {
		case n >= 10:
			want = 4
		case n >= 4:
			want = 3
		}
		if magician.ExtraCantrips() != 1 || warden.ExtraCantrips() != 0 || (character.DruidFeatureState{}).ExtraCantrips() != 0 ||
			character.CantripLimit(character.CasterDruid, n)+magician.ExtraCantrips() != want+1 || character.CantripLimit(character.CasterDruid, n)+warden.ExtraCantrips() != want {
			t.Errorf("level %d: cantrips with the Magician %d, with the Warden %d; want %d and %d", n, character.CantripLimit(character.CasterDruid, n)+magician.ExtraCantrips(),
				character.CantripLimit(character.CasterDruid, n)+warden.ExtraCantrips(), want+1, want)
		}
		if got := tables.WildShapeUses(n); got != wildShapeAt(n) {
			t.Errorf("level %d Wild Shape uses = %d", n, got)
		}
	}
	// FR-042: a Druid 3 / Fighter 5 reads its own Druid level
	if character.CantripLimit(character.CasterDruid, 3) != 2 || character.PreparedLimit(character.CasterDruid, 3) != 6 || character.MaxPreparableLevel(character.CasterDruid, 3) != 2 {
		t.Error("a level 3 Druid does not see 2 cantrips, 6 prepared spells and spells to level 2")
	}
}

func wildShapeAt(level int) int {
	switch {
	case level < 2:
		return 0
	case level < 6:
		return 2
	case level < 17:
		return 3
	}
	return 4
}
