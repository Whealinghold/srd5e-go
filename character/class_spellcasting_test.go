package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// extraSpells adds spells with narrow class lists to the catalogue.
func extraSpells(t *testing.T) map[content.SpellID]content.Spell {
	t.Helper()
	return withSpells(t, spellCatalogue(t),
		spellSpec{"wizard_only", 1, []string{"wizard"}}, spellSpec{"cleric_only", 1, []string{"cleric"}},
		spellSpec{"cleric_cantrip", 0, []string{"cleric"}}, spellSpec{"druid_cantrip", 0, []string{"druid"}})
}

// problemKinds returns the kinds of the *PreparationProblem values joined in err, in order.
func problemKinds(t *testing.T, err error) []character.PreparationRejectionKind {
	t.Helper()
	j, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("err %v is not a join", err)
	}
	var kinds []character.PreparationRejectionKind
	for _, e := range j.Unwrap() {
		var p *character.PreparationProblem
		if !errors.As(e, &p) {
			t.Fatalf("joined error %v is not a *PreparationProblem", e)
		}
		kinds = append(kinds, p.Kind)
	}
	return kinds
}

// Requirement: per case.
// Path: per case.
func TestNewClassSpellcastingRules(t *testing.T) {
	t.Parallel()
	spells := extraSpells(t)
	book := character.NewSpellbook(spellIDs(1, 6))
	cases := []struct {
		name, requirement, path string
		setup                   character.ClassSpellcastingSetup
		level                   int
		want                    []character.PreparationRejectionKind
	}{
		{"P14 wizard 1 valid", "FR-092 FR-042 Table P14", "happy", character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 3), Prepared: spellIDs(1, 4), Spellbook: book}, 1, nil},
		{"P15 wizard 1 five prepared", "FR-042 Table P15", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 5), Spellbook: book}, 1, []character.PreparationRejectionKind{character.PreparationTooManyPrepared}},
		{"FR-051 empty lists", "FR-051", "unhappy (empty and minimal)", character.ClassSpellcastingSetup{Class: character.CasterCleric}, 1, nil},
		{"FR-040 spell not on the class list", "FR-040", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: []content.SpellID{"wizard_only"}}, 1, []character.PreparationRejectionKind{character.PreparationNotOnClassList}},
		{"FR-040 cantrip not on the class list", "FR-040 FR-052", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: []content.SpellID{"druid_cantrip"}}, 1, []character.PreparationRejectionKind{character.PreparationNotOnClassList}},
		{"FR-041 level at the maximum", "FR-041 Table J2", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(2, 1)}, 3, nil},
		{"FR-041 level above the maximum", "FR-041 Table J2", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(3, 1)}, 3, []character.PreparationRejectionKind{character.PreparationSpellLevelTooHigh}},
		{"FR-016 warlock slot level, not highest ever", "FR-016", "happy", character.ClassSpellcastingSetup{Class: character.CasterWarlock, Prepared: spellIDs(1, 2)}, 2, nil},
		{"FR-016 warlock above its slot level", "FR-016 Table J7", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterWarlock, Prepared: spellIDs(2, 1)}, 2, []character.PreparationRejectionKind{character.PreparationSpellLevelTooHigh}},
		{"FR-042 at the limit", "FR-042", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(1, 4)}, 1, nil},
		{"FR-042 one above the limit", "FR-042", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(1, 5)}, 1, []character.PreparationRejectionKind{character.PreparationTooManyPrepared}},
		{"FR-043 duplicate prepared", "FR-043", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: []content.SpellID{spellID(1, 1), spellID(1, 1)}}, 1, []character.PreparationRejectionKind{character.PreparationDuplicateSpell}},
		{"FR-043 duplicate cantrip", "FR-043 FR-052", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: []content.SpellID{spellID(0, 1), spellID(0, 1)}}, 1, []character.PreparationRejectionKind{character.PreparationDuplicateSpell}},
		{"FR-044 cantrip in prepared list", "FR-044", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(0, 1)}, 1, []character.PreparationRejectionKind{character.PreparationCantripNotPreparable}},
		{"P18 duplicate and cantrip in prepared", "FR-093 FR-054 Table P18", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: []content.SpellID{spellID(1, 1), spellID(1, 1), spellID(0, 1)}}, 1,
			[]character.PreparationRejectionKind{character.PreparationDuplicateSpell, character.PreparationCantripNotPreparable}},
		{"FR-045 always-prepared does not count", "FR-045", "happy", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(1, 4), AlwaysPrepared: spellIDs(2, 3)}, 1, nil},
		{"FR-050 wizard spell outside the spellbook", "FR-050", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 2), Spellbook: character.NewSpellbook(spellIDs(1, 1))}, 1, []character.PreparationRejectionKind{character.PreparationNotInSpellbook}},
		{"FR-050 only a wizard needs a spellbook", "FR-050", "unhappy (rule interaction)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: spellIDs(1, 2)}, 1, nil},
		{"FR-052 cantrip count at the limit", "FR-052 Table F", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 3)}, 1, nil},
		{"FR-052 cantrip count one above", "FR-052 Table F", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 4)}, 1, []character.PreparationRejectionKind{character.PreparationTooManyCantrips}},
		{"FR-052 a cantrip of level 1", "FR-052", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(1, 1)}, 1, []character.PreparationRejectionKind{character.PreparationNotACantrip}},
		{"M1 cleric with Thaumaturge, 4 cantrips", "FR-096 Table M1", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: spellIDs(0, 4), ExtraCantrips: 1}, 1, nil},
		{"M2 cleric with Thaumaturge, 5 cantrips", "FR-052 Table M2", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: spellIDs(0, 5), ExtraCantrips: 1}, 1, []character.PreparationRejectionKind{character.PreparationTooManyCantrips}},
		{"M3 cleric without Thaumaturge, 4 cantrips", "FR-052 Table M3", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: spellIDs(0, 4)}, 1, []character.PreparationRejectionKind{character.PreparationTooManyCantrips}},
		{"M4 druid 4 with Magician, 4 cantrips", "FR-096 Table M4", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterDruid, Cantrips: spellIDs(0, 4), ExtraCantrips: 1}, 4, nil},
		{"M4 druid 4 with Magician, 5 cantrips", "FR-096 Table M4", "unhappy (boundary)", character.ClassSpellcastingSetup{Class: character.CasterDruid, Cantrips: spellIDs(0, 5), ExtraCantrips: 1}, 4, []character.PreparationRejectionKind{character.PreparationTooManyCantrips}},
		{"M5 paladin with Blessed Warrior", "FR-018 FR-097 Table M5", "happy", character.ClassSpellcastingSetup{Class: character.CasterPaladin, Granted: []character.GrantedCantrip{
			mustGranted(t, spellID(0, 1), character.CasterPaladin, "cleric"), mustGranted(t, spellID(0, 2), character.CasterPaladin, "cleric")}}, 2, nil},
		{"M6 wizard 1 with 3 chosen and 2 granted", "FR-097 Table M6", "unhappy (rule interaction)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 3), Granted: []character.GrantedCantrip{
			mustGranted(t, spellID(0, 4), character.CasterWizard, "wizard"), mustGranted(t, spellID(0, 5), character.CasterWizard, "wizard")}}, 1, nil},
		{"M7 granted cantrip off the casting class list", "FR-040 FR-097 Table M7", "unhappy (rule interaction)", character.ClassSpellcastingSetup{Class: character.CasterPaladin, Granted: []character.GrantedCantrip{
			mustGranted(t, "cleric_cantrip", character.CasterPaladin, "cleric")}}, 2, nil},
		{"granted cantrip off the list its feature names", "FR-040 FR-097", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterPaladin, Granted: []character.GrantedCantrip{
			mustGranted(t, "druid_cantrip", character.CasterPaladin, "cleric")}}, 2, []character.PreparationRejectionKind{character.PreparationNotOnClassList}},
		{"granted spell that is not a cantrip", "FR-097", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterPaladin, Granted: []character.GrantedCantrip{
			mustGranted(t, spellID(1, 1), character.CasterPaladin, "cleric")}}, 2, []character.PreparationRejectionKind{character.PreparationNotACantrip}},
		{"M11 granted cantrips are not prepared spells", "FR-097 Table M11", "unhappy (rule interaction)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 4), Spellbook: book, Granted: []character.GrantedCantrip{
			mustGranted(t, spellID(0, 1), character.CasterWizard, "wizard")}}, 1, nil},
		{"FR-054 every problem in the fixed order", "FR-054", "unhappy (invalid input)", character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: []content.SpellID{spellID(1, 2), spellID(0, 1), spellID(0, 1), spellID(0, 2), spellID(0, 3)},
			Prepared: []content.SpellID{"cleric_only", spellID(9, 1), spellID(1, 1), spellID(1, 1), spellID(0, 5), spellID(1, 2)}, Spellbook: character.NewSpellbook([]content.SpellID{spellID(1, 1)})}, 1,
			[]character.PreparationRejectionKind{character.PreparationNotOnClassList, character.PreparationSpellLevelTooHigh, character.PreparationTooManyPrepared, character.PreparationDuplicateSpell, character.PreparationDuplicateSpell,
				character.PreparationCantripNotPreparable, character.PreparationNotInSpellbook, character.PreparationNotInSpellbook, character.PreparationNotInSpellbook, character.PreparationNotInSpellbook,
				character.PreparationTooManyCantrips, character.PreparationNotACantrip}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.NewClassSpellcasting(c.setup, c.level, spells)
			if len(c.want) == 0 {
				if err != nil || got.IsZero() || got.Class() != c.setup.Class || got.Level() != c.level {
					t.Fatalf("err = %v, class = %v, want valid [%s]", err, got.Class(), c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("err = %v, want ErrInvalidInput [%s]", err, c.requirement)
			}
			if kinds := problemKinds(t, err); !reflect.DeepEqual(kinds, c.want) {
				t.Errorf("problems = %v, want %v [%s]", kinds, c.want, c.requirement)
			}
		})
	}
}

func mustGranted(t *testing.T, spell content.SpellID, class character.SpellcastingClass, list string) character.GrantedCantrip {
	t.Helper()
	g, err := character.NewGrantedCantrip(spell, class, classOf(list))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// Requirement: doc/specs/spellcasting.md FR-093 - NewClassSpellcasting never panics: an
// invalid class, level, allowance or an absent spell is an ErrInvalidInput error before
// any list check; MustClassSpellcasting panics with that error.
// Path: unhappy (invalid input).
func TestNewClassSpellcastingInvalidInput(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	ok := character.ClassSpellcastingSetup{Class: character.CasterCleric}
	for name, c := range map[string]struct {
		setup character.ClassSpellcastingSetup
		level int
	}{
		"invalid class":   {character.ClassSpellcastingSetup{}, 1},
		"level 0":         {ok, 0},
		"level 21":        {ok, 21},
		"negative extra":  {character.ClassSpellcastingSetup{Class: character.CasterCleric, ExtraCantrips: -1}, 1},
		"absent prepared": {character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: []content.SpellID{"not_there"}}, 1},
		"absent granted":  {character.ClassSpellcastingSetup{Class: character.CasterPaladin, Granted: []character.GrantedCantrip{mustGranted(t, "not_there", character.CasterPaladin, "cleric")}}, 2},
	} {
		if _, err := character.NewClassSpellcasting(c.setup, c.level, spells); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	defer func() {
		if err, isErr := recover().(error); !isErr || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("MustClassSpellcasting panic = %v", err)
		}
	}()
	character.MustClassSpellcasting(ok, 0, spells)
}

// Requirement: doc/specs/spellcasting.md FR-028, FR-047 - a ClassSpellcasting copies its
// inputs and its accessors return copies; SpellsToChoose and CantripsToChoose are the
// headroom, never below zero, with granted cantrips and always-prepared spells excluded.
// Path: unhappy (immutability).
func TestClassSpellcastingCopiesAndHeadroom(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	prepared, cantrips := spellIDs(1, 2), spellIDs(0, 2)
	g := mustGranted(t, spellID(0, 9), character.CasterCleric, "cleric")
	cs := character.MustClassSpellcasting(character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: prepared, Cantrips: cantrips, ExtraCantrips: 1,
		AlwaysPrepared: spellIDs(2, 2), Granted: []character.GrantedCantrip{g}}, 1, spells)
	prepared[0], cantrips[0] = spellID(1, 9), spellID(0, 9)
	cs.Prepared().Spells()[0] = spellID(1, 9)
	cs.Cantrips().Spells()[0] = spellID(0, 9)
	cs.GrantedCantrips()[0] = character.GrantedCantrip{}
	if cs.Prepared().Spells()[0] != spellID(1, 1) || cs.Cantrips().Spells()[0] != spellID(0, 1) || cs.GrantedCantrips()[0] != g {
		t.Error("ClassSpellcasting shares memory with its inputs or accessors")
	}
	if cs.SpellsToChoose() != 2 || cs.CantripsToChoose() != 2 || cs.ExtraCantrips() != 1 || cs.Prepared().Count() != 2 || len(cs.Prepared().AlwaysPrepared()) != 2 {
		t.Errorf("headroom spells %d cantrips %d, want 2 and 2 (4 + 1 - 2... Table F 3 + 1 - 2)", cs.SpellsToChoose(), cs.CantripsToChoose())
	}
	if !cs.Prepared().IsPrepared(spellID(1, 1)) || !cs.Prepared().IsPrepared(spellID(2, 1)) || cs.Prepared().IsPrepared(spellID(1, 5)) {
		t.Error("IsPrepared must cover chosen and always-prepared spells only")
	}
	full := character.MustClassSpellcasting(character.ClassSpellcastingSetup{Class: character.CasterPaladin}, 2, spells)
	if full.CantripsToChoose() != 0 || full.SpellsToChoose() != 3 {
		t.Errorf("paladin 2 headroom = %d spells, %d cantrips, want 3 and 0", full.SpellsToChoose(), full.CantripsToChoose())
	}
	if !(character.ClassSpellcasting{}).IsZero() {
		t.Error("zero ClassSpellcasting is not zero")
	}
}

// Requirement: doc/specs/spellcasting.md FR-093 - NewClassSpellcasting never panics for any class, level,
// allowance and list sizes, and a value it returns satisfies the limits of Tables F and G.
// Path: unhappy (invalid input).
func FuzzNewClassSpellcasting(f *testing.F) {
	f.Add(8, 1, 0, 4, 3)
	f.Add(0, 0, -1, 0, 0)
	f.Add(4, 21, 2, 99, 99)
	f.Fuzz(func(t *testing.T, class, level, extra, prepared, cantrips int) {
		spells := spellCatalogue(t)
		prepared, cantrips = max(0, min(prepared, 12)), max(0, min(cantrips, 12))
		cs, err := character.NewClassSpellcasting(character.ClassSpellcastingSetup{Class: character.SpellcastingClass(class), ExtraCantrips: extra,
			Prepared: spellIDs(1, prepared), Cantrips: spellIDs(0, cantrips)}, level, spells)
		if err != nil {
			return
		}
		if cs.Prepared().Count() > character.PreparedLimit(cs.Class(), level) || cs.Cantrips().Count() > character.CantripLimit(cs.Class(), level)+extra {
			t.Fatalf("limits broken: %+v", cs)
		}
	})
}

// Requirement: doc/specs/spellcasting.md FR-054, FR-093 - a PreparationProblem matches only an equal
// *PreparationProblem; a rejection reports its first kind and whether it holds a kind; level 20 is a valid class level.
// Path: unhappy (boundary).
func TestPreparationProblemIdentityAndRejectionReaders(t *testing.T) {
	t.Parallel()
	p := &character.PreparationProblem{Kind: character.PreparationDuplicateSpell, Spell: "a", Class: character.CasterBard}
	var nilProblem *character.PreparationProblem
	for name, target := range map[string]error{"another type": errors.New("x"), "nil problem": nilProblem, "other spell": &character.PreparationProblem{Kind: p.Kind, Spell: "b", Class: p.Class}, "other class": &character.PreparationProblem{Kind: p.Kind, Spell: "a", Class: character.CasterCleric}} {
		if p.Is(target) {
			t.Errorf("Is(%s) = true", name)
		}
	}
	if nilProblem.Is(p) {
		t.Error("a nil problem matches")
	}
	spells := spellCatalogue(t)
	st := castingState(t, classLevelsOf(t, "cleric", 1), spells)
	out, err := character.ChoosePreparation(st, character.CasterCleric, cat(spellIDs(1, 1), spellIDs(1, 1), spellIDs(0, 1)), nil, spells)
	if err != nil {
		t.Fatal(err)
	}
	rej := out.(character.PreparationRefused).Rejection()
	if rej.First() != character.PreparationDuplicateSpell || !rej.Has(character.PreparationCantripNotPreparable) || rej.Has(character.PreparationTooManyPrepared) {
		t.Errorf("rejection = %+v", rej.Problems())
	}
	if _, err := character.NewClassSpellcasting(character.ClassSpellcastingSetup{Class: character.CasterWizard}, 20, spells); err != nil {
		t.Errorf("level 20: %v", err)
	}
	if _, err := character.NewClassSpellcasting(character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: []content.SpellID{"gone"}}, 1, spells); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("absent spell: %v", err)
	} else {
		var pp *character.PreparationProblem
		if errors.As(err, &pp) {
			t.Error("an absent spell must fail before the list checks")
		}
	}
}
