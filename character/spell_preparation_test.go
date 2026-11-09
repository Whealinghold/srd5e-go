package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func cat(lists ...[]content.SpellID) []content.SpellID {
	var out []content.SpellID
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

// refused returns the kinds of a PreparationRefused outcome, failing on any other variant.
func refused(t *testing.T, out character.PreparationOutcome, err error) []character.PreparationRejectionKind {
	t.Helper()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	r, ok := out.(character.PreparationRefused)
	if !ok {
		t.Fatalf("outcome = %#v, want PreparationRefused", out)
	}
	var kinds []character.PreparationRejectionKind
	for _, p := range r.Rejection().Problems() {
		kinds = append(kinds, p.Kind)
	}
	return kinds
}

// accepted returns the new state and events of a PreparationAccepted outcome.
func accepted(t *testing.T, out character.PreparationOutcome, err error) (character.SpellcastingState, []character.SpellcastingEvent) {
	t.Helper()
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	a, ok := out.(character.PreparationAccepted)
	if !ok {
		t.Fatalf("outcome = %#v, want PreparationAccepted", out)
	}
	return a.State(), a.Events()
}

func classOfState(t *testing.T, s character.SpellcastingState, c character.SpellcastingClass) character.ClassSpellcasting {
	t.Helper()
	cs, ok := s.Class(c)
	if !ok {
		t.Fatalf("state does not hold %v", c)
	}
	return cs
}

// Requirement: per case.
// Path: per case.
func TestChoosePreparation(t *testing.T) {
	t.Parallel()
	spells := extraSpells(t)
	cleric4 := castingState(t, classLevelsOf(t, "cleric", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: cat(spellIDs(1, 3), spellIDs(2, 3)), Cantrips: spellIDs(0, 3)})
	cases := []struct {
		name, requirement, path string
		prepared, cantrips      []content.SpellID
		want                    []character.PreparationRejectionKind // nil: accepted
		events                  int
	}{
		{"P6 add one within the headroom", "FR-089 Table P6", "happy", spellIDs(2, 4)[3:], nil, nil, 1},
		{"P7 add two over the limit", "FR-089 Table P7", "unhappy (boundary)", spellIDs(2, 5)[3:], nil, []character.PreparationRejectionKind{character.PreparationTooManyPrepared}, 0},
		{"P17 add nothing", "FR-089 Table P17", "unhappy (empty and minimal)", nil, nil, nil, 0},
		{"add an empty non-nil slice", "FR-089", "unhappy (empty and minimal)", []content.SpellID{}, []content.SpellID{}, nil, 0},
		{"add one cantrip within the headroom", "FR-089 FR-052", "happy", nil, spellIDs(0, 4)[3:], nil, 1},
		{"add one cantrip too many", "FR-089 FR-052", "unhappy (boundary)", nil, spellIDs(0, 5)[3:], []character.PreparationRejectionKind{character.PreparationTooManyCantrips}, 0},
		{"add a spell already prepared", "FR-043", "unhappy (invalid input)", spellIDs(1, 1), nil, []character.PreparationRejectionKind{character.PreparationDuplicateSpell}, 0},
		{"add a cantrip as a prepared spell", "FR-044", "unhappy (invalid input)", spellIDs(0, 9)[8:], nil, []character.PreparationRejectionKind{character.PreparationCantripNotPreparable}, 0},
		{"add a level above the maximum", "FR-041", "unhappy (boundary)", spellIDs(3, 1), nil, []character.PreparationRejectionKind{character.PreparationSpellLevelTooHigh}, 0},
		{"add a spell off the class list", "FR-040", "unhappy (invalid input)", []content.SpellID{"wizard_only"}, nil, []character.PreparationRejectionKind{character.PreparationNotOnClassList}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := cleric4
			out, err := character.ChoosePreparation(cleric4, character.CasterCleric, c.prepared, c.cantrips, spells)
			if c.want != nil {
				if got := refused(t, out, err); !reflect.DeepEqual(got, c.want) {
					t.Errorf("kinds = %v, want %v [%s]", got, c.want, c.requirement)
				}
			} else {
				st, ev := accepted(t, out, err)
				if len(ev) != c.events {
					t.Errorf("events = %v, want %d [%s]", ev, c.events, c.requirement)
				}
				if c.events == 0 && !reflect.DeepEqual(st, cleric4) {
					t.Errorf("an empty addition changed the state [%s]", c.requirement)
				}
				if c.events == 1 {
					if e, ok := ev[0].(character.PreparedSpellsChanged); !ok || e.Class() != character.CasterCleric || e.CharacterID() != caster {
						t.Errorf("event = %#v", ev[0])
					}
				}
			}
			if !reflect.DeepEqual(before, cleric4) {
				t.Error("the input state was altered")
			}
		})
	}
	out, err := character.ChoosePreparation(cleric4, character.CasterCleric, spellIDs(2, 4)[3:], nil, spells)
	st, _ := accepted(t, out, err)
	if got := classOfState(t, st, character.CasterCleric).Prepared().Count(); got != 7 {
		t.Errorf("prepared after P6 = %d, want 7", got)
	}
}

// Requirement: doc/specs/spellcasting.md FR-117, FR-126, Tables P20-P22, R4, R5 - a spell
// absent from the map is a content-missing outcome that takes precedence over rule
// checks, in command order without repeats; a class not held is an ErrClassNotHeld error.
// Path: unhappy (invalid input).
func TestPreparationContentMissingAndClassNotHeld(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	st := castingState(t, classLevelsOf(t, "cleric", 1), spells)
	out, err := character.ChoosePreparation(st, character.CasterCleric, []content.SpellID{"gone_a", spellID(1, 1), "gone_a"}, []content.SpellID{"gone_b"}, spells)
	m, ok := out.(character.PreparationContentMissing)
	if err != nil || !ok || len(m.Missing()) != 2 || m.Missing()[0] != content.SpellID("gone_a").Missing() || m.Missing()[1] != content.SpellID("gone_b").Missing() {
		t.Fatalf("R5: outcome = %#v, err %v", out, err)
	}
	// R4: the missing id is reported alone although the list is also too long.
	out, err = character.ChangePreparation(st, character.CasterCleric, character.TriggerLongRest, character.PreparationChange{Prepared: cat(spellIDs(1, 6), []content.SpellID{"gone_a"})}, spells)
	m, ok = out.(character.PreparationContentMissing)
	if err != nil || !ok || len(m.Missing()) != 1 {
		t.Fatalf("R4: outcome = %#v, err %v", out, err)
	}
	for name, f := range map[string]func() (character.PreparationOutcome, error){
		"P20 choose": func() (character.PreparationOutcome, error) {
			return character.ChoosePreparation(st, character.CasterWizard, nil, nil, spells)
		},
		"P21 change": func() (character.PreparationOutcome, error) {
			return character.ChangePreparation(st, character.CasterWizard, character.TriggerLongRest, character.PreparationChange{}, spells)
		},
		"P22 granted class not held": func() (character.PreparationOutcome, error) {
			g := mustGranted(t, spellID(0, 1), character.CasterWizard, "wizard")
			return character.ChangePreparation(st, character.CasterCleric, character.TriggerGainedLevel, character.PreparationChange{GrantedReplacements: []character.GrantedCantripReplacement{{Replaced: "x", With: g}}}, spells)
		},
	} {
		if out, err := f(); out != nil || !errors.Is(err, character.ErrClassNotHeld) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: outcome %v err %v, want ErrClassNotHeld", name, out, err)
		}
	}
}

// preparedWizard5 is a Wizard 5 with 9 prepared spells, a spellbook of 17 and 4 cantrips.
func preparedWizard5(t *testing.T, spells map[content.SpellID]content.Spell) character.SpellcastingState {
	t.Helper()
	return castingState(t, classLevelsOf(t, "wizard", 5), spells, character.ClassSpellcastingSetup{
		Class: character.CasterWizard, Prepared: cat(spellIDs(1, 5), spellIDs(2, 4)), Cantrips: spellIDs(0, 4),
		Spellbook: character.NewSpellbook(cat(spellIDs(1, 8), spellIDs(2, 6), spellIDs(3, 3)))})
}

// Requirement: per case.
// Path: per case.
func TestChangePreparationTriggers(t *testing.T) {
	t.Parallel()
	spells := extraSpells(t)
	wiz5 := preparedWizard5(t, spells)
	swap := func(old, repl content.SpellID) []content.SpellID {
		l := cat(spellIDs(1, 5), spellIDs(2, 4))
		for i := range l {
			if l[i] == old {
				l[i] = repl
			}
		}
		return l
	}
	cleric6 := func(always ...content.SpellID) character.SpellcastingState {
		return castingState(t, classLevelsOf(t, "cleric", 3), spells, character.ClassSpellcastingSetup{Class: character.CasterCleric, Prepared: cat(spellIDs(1, 3), spellIDs(2, 3)), AlwaysPrepared: always})
	}
	paladin := castingState(t, classLevelsOf(t, "paladin", 5), spells, character.ClassSpellcastingSetup{Class: character.CasterPaladin, Prepared: cat(spellIDs(1, 3), spellIDs(2, 3))})
	bard3 := castingState(t, classLevelsOf(t, "bard", 3), spells, character.ClassSpellcastingSetup{Class: character.CasterBard, Prepared: spellIDs(1, 5)})
	ranger5 := castingState(t, classLevelsOf(t, "ranger", 5), spells, character.ClassSpellcastingSetup{Class: character.CasterRanger, Prepared: cat(spellIDs(1, 3), spellIDs(2, 3))})
	wiz4 := castingState(t, classLevelsOf(t, "wizard", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 7), Spellbook: character.NewSpellbook(spellIDs(1, 9))})
	NotNow, TooMany := character.PreparationChangeNotAllowedNow, character.PreparationTooManyReplacements
	cases := []struct {
		name, requirement, path string
		state                   character.SpellcastingState
		class                   character.SpellcastingClass
		trigger                 character.PreparationTrigger
		prepared                []content.SpellID
		want                    []character.PreparationRejectionKind // nil: accepted
	}{
		{"P1 wizard 5 swaps one on a Short Rest", "FR-091 Table P1", "happy", wiz5, character.CasterWizard, character.TriggerShortRest, swap(spellID(1, 1), spellID(1, 6)), nil},
		{"P2 wizard 4 on a Short Rest", "FR-091 Table P2", "unhappy (state error)", wiz4, character.CasterWizard, character.TriggerShortRest, cat(spellIDs(1, 6), spellIDs(1, 9)[8:]), []character.PreparationRejectionKind{NotNow}},
		{"P3 wizard 5 swaps two on a Short Rest", "FR-049 Table P3", "unhappy (boundary)", wiz5, character.CasterWizard, character.TriggerShortRest, swap(spellID(1, 2), spellID(1, 7))[:0:0], nil},
		{"P4 swap in a spell outside the spellbook", "FR-050 Table P4", "unhappy (invalid input)", wiz5, character.CasterWizard, character.TriggerShortRest, swap(spellID(1, 1), spellID(1, 9)), []character.PreparationRejectionKind{character.PreparationNotInSpellbook}},
		{"P5 ranger on a Short Rest", "FR-091 Table P5", "unhappy (state error)", ranger5, character.CasterRanger, character.TriggerShortRest, cat(spellIDs(1, 2), spellIDs(1, 7)[6:], spellIDs(2, 3)), []character.PreparationRejectionKind{NotNow}},
		{"P8 cleric on gaining a level", "FR-048 Table P8", "unhappy (state error)", cleric6(), character.CasterCleric, character.TriggerGainedLevel, cat(spellIDs(1, 2), spellIDs(1, 7)[6:], spellIDs(2, 3)), []character.PreparationRejectionKind{NotNow}},
		{"P11 paladin replaces one on a Long Rest", "FR-049 Table P11", "happy", paladin, character.CasterPaladin, character.TriggerLongRest, cat(spellIDs(1, 2), spellIDs(1, 7)[6:], spellIDs(2, 3)), nil},
		{"P12 paladin replaces two on a Long Rest", "FR-049 Table P12", "unhappy (boundary)", paladin, character.CasterPaladin, character.TriggerLongRest, cat(spellIDs(1, 1), spellIDs(1, 8)[6:], spellIDs(2, 3)), []character.PreparationRejectionKind{TooMany}},
		{"cleric replaces every spell on a Long Rest", "FR-049", "happy", cleric6(), character.CasterCleric, character.TriggerLongRest, cat(spellIDs(1, 6)[3:], spellIDs(2, 6)[3:]), nil},
		{"P9 bard replaces one and adds one on gaining a level", "FR-090 Table P9", "unhappy (rule interaction)", bard3, character.CasterBard, character.TriggerGainedLevel, cat(spellIDs(1, 5)[1:], spellIDs(1, 7)[5:]), nil},
		{"P10 bard replaces two on gaining a level", "FR-049 Table P10", "unhappy (boundary)", bard3, character.CasterBard, character.TriggerGainedLevel, cat(spellIDs(1, 3)[2:], spellIDs(1, 8)[5:]), []character.PreparationRejectionKind{TooMany}},
		{"bard on a Long Rest", "FR-048", "unhappy (state error)", bard3, character.CasterBard, character.TriggerLongRest, cat(spellIDs(1, 5)[1:], spellIDs(1, 6)[5:]), []character.PreparationRejectionKind{NotNow}},
		{"P13 always-prepared dropped", "FR-046 Table P13", "unhappy (state error)", cleric6(spellID(3, 1)), character.CasterCleric, character.TriggerLongRest, cat(spellIDs(1, 3), spellIDs(2, 3)), []character.PreparationRejectionKind{character.PreparationAlwaysPreparedNotReplaceable}},
		{"always-prepared kept and not counted", "FR-045 FR-046", "happy", cleric6(spellID(3, 1)), character.CasterCleric, character.TriggerLongRest, cat(spellIDs(1, 3), spellIDs(2, 3), []content.SpellID{spellID(3, 1)}), nil},
		{"same list again", "FR-090", "unhappy (empty and minimal)", wiz5, character.CasterWizard, character.TriggerLongRest, cat(spellIDs(1, 5), spellIDs(2, 4)), nil},
		{"empty list is valid", "FR-051", "unhappy (empty and minimal)", wiz5, character.CasterWizard, character.TriggerLongRest, []content.SpellID{}, nil},
	}
	cases[2].prepared = cat(spellIDs(1, 5)[2:], spellIDs(1, 8)[6:], spellIDs(2, 4))
	cases[2].want = []character.PreparationRejectionKind{TooMany}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.state
			out, err := character.ChangePreparation(c.state, c.class, c.trigger, character.PreparationChange{Prepared: c.prepared}, spells)
			if c.want != nil {
				if got := refused(t, out, err); !reflect.DeepEqual(got, c.want) {
					t.Errorf("kinds = %v, want %v [%s]", got, c.want, c.requirement)
				}
			} else {
				st, _ := accepted(t, out, err)
				if got := classOfState(t, st, c.class).Prepared().Spells(); !reflect.DeepEqual(got, withoutAlways(c.prepared, classOfState(t, c.state, c.class).Prepared().AlwaysPrepared())) {
					t.Errorf("prepared = %v, want %v [%s]", got, c.prepared, c.requirement)
				}
			}
			if !reflect.DeepEqual(before, c.state) {
				t.Error("the input state was altered")
			}
		})
	}
}

func withoutAlways(l, always []content.SpellID) []content.SpellID {
	out := []content.SpellID{}
	for _, id := range l {
		keep := true
		for _, a := range always {
			keep = keep && a != id
		}
		if keep {
			out = append(out, id)
		}
	}
	return out
}

// Requirement: doc/specs/spellcasting.md FR-089 to FR-091, FR-098 - a change that
// replaces nothing reports no event and leaves the state equal; an accepted change
// reports one PreparedSpellsChanged event.
// Path: unhappy (empty and minimal).
func TestChangePreparationEvents(t *testing.T) {
	t.Parallel()
	spells := extraSpells(t)
	wiz5 := preparedWizard5(t, spells)
	out, err := character.ChangePreparation(wiz5, character.CasterWizard, character.TriggerLongRest, character.PreparationChange{}, spells)
	if st, ev := accepted(t, out, err); len(ev) != 0 || !reflect.DeepEqual(st, wiz5) {
		t.Errorf("empty change: events %v", ev)
	}
	out, err = character.ChangePreparation(wiz5, character.CasterWizard, character.TriggerShortRest, character.PreparationChange{Prepared: cat(spellIDs(1, 4), spellIDs(1, 6)[5:], spellIDs(2, 4))}, spells)
	_, ev := accepted(t, out, err)
	if len(ev) != 1 {
		t.Fatalf("events = %v, want one", ev)
	}
	if e, ok := ev[0].(character.PreparedSpellsChanged); !ok || e.Class() != character.CasterWizard {
		t.Errorf("event = %#v", ev[0])
	}
}

// Requirement: doc/specs/spellcasting.md FR-053 - one cantrip may be replaced on the
// class's trigger (gaining a level; a Wizard's Long Rest), two or a wrong trigger are
// refused, and Paladin and Ranger have none to replace.
// Path: unhappy (state error).
func TestChangeCantrips(t *testing.T) {
	t.Parallel()
	spells := extraSpells(t)
	bard := castingState(t, classLevelsOf(t, "bard", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterBard, Cantrips: spellIDs(0, 2)})
	wiz := castingState(t, classLevelsOf(t, "wizard", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 4)})
	pal := castingState(t, classLevelsOf(t, "paladin", 4), spells, character.ClassSpellcastingSetup{Class: character.CasterPaladin})
	cases := []struct {
		name    string
		state   character.SpellcastingState
		class   character.SpellcastingClass
		trigger character.PreparationTrigger
		new     []content.SpellID
		want    []character.PreparationRejectionKind
	}{
		{"bard replaces one on gaining a level", bard, character.CasterBard, character.TriggerGainedLevel, cat(spellIDs(0, 2)[1:], spellIDs(0, 8)[7:]), nil},
		{"bard replaces two", bard, character.CasterBard, character.TriggerGainedLevel, spellIDs(0, 9)[7:], []character.PreparationRejectionKind{character.PreparationTooManyReplacements}},
		{"bard replaces one on a Long Rest", bard, character.CasterBard, character.TriggerLongRest, cat(spellIDs(0, 2)[1:], spellIDs(0, 8)[7:]), []character.PreparationRejectionKind{character.PreparationChangeNotAllowedNow}},
		{"wizard replaces one on a Long Rest", wiz, character.CasterWizard, character.TriggerLongRest, cat(spellIDs(0, 3), spellIDs(0, 9)[8:]), nil},
		{"wizard replaces one on gaining a level", wiz, character.CasterWizard, character.TriggerGainedLevel, cat(spellIDs(0, 3), spellIDs(0, 9)[8:]), []character.PreparationRejectionKind{character.PreparationChangeNotAllowedNow}},
		{"paladin has no cantrip to replace", pal, character.CasterPaladin, character.TriggerGainedLevel, spellIDs(0, 1), []character.PreparationRejectionKind{character.PreparationTooManyCantrips}},
		{"adding a cantrip needs no trigger", bard, character.CasterBard, character.TriggerLongRest, cat(spellIDs(0, 2), spellIDs(0, 3)[2:]), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := character.ChangePreparation(c.state, c.class, c.trigger, character.PreparationChange{Cantrips: c.new}, spells)
			if c.want != nil {
				if got := refused(t, out, err); !reflect.DeepEqual(got, c.want) {
					t.Errorf("kinds = %v, want %v [FR-053]", got, c.want)
				}
				return
			}
			st, _ := accepted(t, out, err)
			if got := classOfState(t, st, c.class).Cantrips().Spells(); !reflect.DeepEqual(got, c.new) {
				t.Errorf("cantrips = %v, want %v", got, c.new)
			}
		})
	}
}

// Requirement: per case.
// Path: per case.
func TestChangeGrantedCantrips(t *testing.T) {
	t.Parallel()
	spells := extraSpells(t)
	g1, g2 := mustGranted(t, spellID(0, 1), character.CasterRanger, "druid"), mustGranted(t, spellID(0, 2), character.CasterRanger, "druid")
	ranger := castingState(t, classLevelsOf(t, "ranger", 2), spells, character.ClassSpellcastingSetup{Class: character.CasterRanger, Granted: []character.GrantedCantrip{g1, g2}})
	repl := func(old, new content.SpellID) character.GrantedCantripReplacement {
		return character.GrantedCantripReplacement{Replaced: old, With: mustGranted(t, new, character.CasterRanger, "druid")}
	}
	cases := []struct {
		name, requirement, path string
		trigger                 character.PreparationTrigger
		repl                    []character.GrantedCantripReplacement
		want                    []character.PreparationRejectionKind
	}{
		{"M8 one on gaining a level", "FR-098 Table M8", "happy", character.TriggerGainedLevel, []character.GrantedCantripReplacement{repl(spellID(0, 1), spellID(0, 7))}, nil},
		{"M9 two on gaining a level", "FR-098 Table M9", "unhappy (boundary)", character.TriggerGainedLevel, []character.GrantedCantripReplacement{repl(spellID(0, 1), spellID(0, 7)), repl(spellID(0, 2), spellID(0, 8))}, []character.PreparationRejectionKind{character.PreparationTooManyReplacements}},
		{"M10 one on a Long Rest", "FR-098 Table M10", "unhappy (state error)", character.TriggerLongRest, []character.GrantedCantripReplacement{repl(spellID(0, 1), spellID(0, 7))}, []character.PreparationRejectionKind{character.PreparationChangeNotAllowedNow}},
		{"replacement off the feature list", "FR-040 FR-098", "unhappy (invalid input)", character.TriggerGainedLevel, []character.GrantedCantripReplacement{{Replaced: spellID(0, 1), With: mustGranted(t, "cleric_cantrip", character.CasterRanger, "druid")}}, []character.PreparationRejectionKind{character.PreparationNotOnClassList}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, err := character.ChangePreparation(ranger, character.CasterRanger, c.trigger, character.PreparationChange{GrantedReplacements: c.repl}, spells)
			if c.want != nil {
				if got := refused(t, out, err); !reflect.DeepEqual(got, c.want) {
					t.Errorf("kinds = %v, want %v [%s]", got, c.want, c.requirement)
				}
				return
			}
			st, ev := accepted(t, out, err)
			got := classOfState(t, st, character.CasterRanger).GrantedCantrips()
			if len(got) != 2 || got[0].Spell() != spellID(0, 7) || got[1] != g2 || len(ev) != 1 {
				t.Errorf("granted = %v events %v [%s]", got, ev, c.requirement)
			}
		})
	}
	// Invalid commands are errors: a replaced cantrip never granted, another class, an invalid trigger.
	for name, f := range map[string]func() error{
		"ungranted": func() error {
			_, err := character.ChangePreparation(ranger, character.CasterRanger, character.TriggerGainedLevel, character.PreparationChange{GrantedReplacements: []character.GrantedCantripReplacement{repl(spellID(0, 9), spellID(0, 7))}}, spells)
			return err
		},
		"another class": func() error {
			st := castingState(t, classLevelsOf(t, "ranger", 2, "wizard", 1), spells)
			w := character.GrantedCantripReplacement{Replaced: spellID(0, 1), With: mustGranted(t, spellID(0, 7), character.CasterWizard, "wizard")}
			_, err := character.ChangePreparation(st, character.CasterRanger, character.TriggerGainedLevel, character.PreparationChange{GrantedReplacements: []character.GrantedCantripReplacement{w}}, spells)
			return err
		},
		"invalid trigger": func() error {
			_, err := character.ChangePreparation(ranger, character.CasterRanger, 0, character.PreparationChange{}, spells)
			return err
		},
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) || errors.Is(err, character.ErrClassNotHeld) {
			t.Errorf("%s: err = %v, want a plain ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/spellcasting.md FR-048, FR-143 - triggers and rejection kinds have
// stable names.
// Path: happy.
func TestPreparationNames(t *testing.T) {
	t.Parallel()
	want := []string{"not_on_class_list", "spell_level_too_high", "too_many_prepared", "duplicate_spell", "cantrip_not_preparable",
		"always_prepared_not_replaceable", "change_not_allowed_now", "too_many_replacements", "not_in_spellbook", "too_many_cantrips", "not_a_cantrip"}
	for i, w := range want {
		if got := character.PreparationRejectionKind(i + 1).String(); got != w {
			t.Errorf("kind %d = %q, want %q", i+1, got, w)
		}
	}
	if character.PreparationRejectionKind(0).String() != "unknown" || character.PreparationRejectionKind(12).String() != "unknown" {
		t.Error("out-of-range kind is not unknown")
	}
	for tr, w := range map[character.PreparationTrigger]string{character.TriggerLongRest: "long_rest", character.TriggerShortRest: "short_rest", character.TriggerGainedLevel: "gained_level", 0: "unknown"} {
		if tr.String() != w || tr.Valid() != (w != "unknown") {
			t.Errorf("trigger %d = %q", tr, tr.String())
		}
	}
	p := &character.PreparationProblem{Kind: character.PreparationDuplicateSpell, Spell: "a", Class: character.CasterBard}
	if !errors.Is(p, &character.PreparationProblem{Kind: character.PreparationDuplicateSpell, Spell: "a", Class: character.CasterBard}) || errors.Is(p, &character.PreparationProblem{Kind: character.PreparationDuplicateSpell}) || !errors.Is(p, srderr.ErrInvalidInput) || p.Error() == "" {
		t.Error("PreparationProblem identity")
	}
	r := character.PreparationRejection{}
	if r.First() != 0 || r.Has(character.PreparationDuplicateSpell) || len(r.Problems()) != 0 {
		t.Error("zero PreparationRejection is not empty")
	}
}

// Requirement: per case.
// Path: per case.
func TestReplacementTriggersForEveryClass(t *testing.T) {
	t.Parallel()
	spells := spellCatalogue(t)
	const any, none = -1, 0                              // replacements allowed: any number, one (1) or no trigger
	prepTable := map[character.SpellcastingClass][3]int{ // long rest, short rest, gained level
		character.CasterBard: {none, none, 1}, character.CasterCleric: {any, none, none}, character.CasterDruid: {any, none, none}, character.CasterPaladin: {1, none, none},
		character.CasterRanger: {1, none, none}, character.CasterSorcerer: {none, none, 1}, character.CasterWarlock: {none, none, 1}, character.CasterWizard: {any, 1, none},
	}
	triggers := [3]character.PreparationTrigger{character.TriggerLongRest, character.TriggerShortRest, character.TriggerGainedLevel}
	for class, allowed := range prepTable {
		for ti, trig := range triggers {
			level := 6 // Wizard level 5 or higher has Memorize Spell
			st := castingState(t, classLevelsOf(t, class.String(), level), spells, character.ClassSpellcastingSetup{Class: class, Prepared: spellIDs(1, 2), Spellbook: character.NewSpellbook(spellIDs(1, 8))})
			for replaced := 1; replaced <= 2; replaced++ {
				next := append(spellIDs(1, 2)[replaced:], spellIDs(1, 8)[8-replaced:]...)
				out, err := character.ChangePreparation(st, class, trig, character.PreparationChange{Prepared: next}, spells)
				a := allowed[ti]
				switch {
				case a == none:
					if k := refused(t, out, err); len(k) != 1 || k[0] != character.PreparationChangeNotAllowedNow {
						t.Errorf("%v %v x%d: kinds = %v, want not allowed now [FR-048 Table I]", class, trig, replaced, k)
					}
				case a == 1 && replaced == 2:
					if k := refused(t, out, err); len(k) != 1 || k[0] != character.PreparationTooManyReplacements {
						t.Errorf("%v %v x%d: kinds = %v, want too many replacements [FR-049]", class, trig, replaced, k)
					}
				default:
					accepted(t, out, err)
				}
			}
		}
	}
	// Memorize Spell needs Wizard level 5: level 4 is refused, 5 and 6 accepted.
	for level, ok := range map[int]bool{4: false, 5: true, 6: true} {
		st := castingState(t, classLevelsOf(t, "wizard", level), spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 2), Spellbook: character.NewSpellbook(spellIDs(1, 8))})
		out, err := character.ChangePreparation(st, character.CasterWizard, character.TriggerShortRest, character.PreparationChange{Prepared: append(spellIDs(1, 2)[1:], spellID(1, 8))}, spells)
		if ok {
			accepted(t, out, err)
		} else if k := refused(t, out, err); k[0] != character.PreparationChangeNotAllowedNow {
			t.Errorf("wizard %d: %v", level, k)
		}
	}
	// A failing swap by the same call reports the problems in the fixed order across both sources.
	st := castingState(t, classLevelsOf(t, "wizard", 5), spells, character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 3), Spellbook: character.NewSpellbook(spellIDs(1, 3))})
	out, err := character.ChangePreparation(st, character.CasterWizard, character.TriggerShortRest, character.PreparationChange{Prepared: []content.SpellID{spellID(1, 3), spellID(1, 3), spellID(1, 9), spellID(1, 10)}}, spells)
	want := []character.PreparationRejectionKind{character.PreparationDuplicateSpell, character.PreparationTooManyReplacements, character.PreparationNotInSpellbook, character.PreparationNotInSpellbook}
	if got := refused(t, out, err); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds = %v, want %v", got, want)
	}

	cantrips := map[character.SpellcastingClass][3]bool{ // long rest, short rest, gained level
		character.CasterBard: {false, false, true}, character.CasterCleric: {false, false, true}, character.CasterDruid: {false, false, true},
		character.CasterSorcerer: {false, false, true}, character.CasterWarlock: {false, false, true}, character.CasterWizard: {true, false, false},
	}
	for class, ok := range cantrips {
		for ti, trig := range triggers {
			st := castingState(t, classLevelsOf(t, class.String(), 4), spells, character.ClassSpellcastingSetup{Class: class, Cantrips: spellIDs(0, 2)})
			out, err := character.ChangePreparation(st, class, trig, character.PreparationChange{Cantrips: []content.SpellID{spellID(0, 2), spellID(0, 9)}}, spells)
			if ok[ti] {
				accepted(t, out, err)
			} else if k := refused(t, out, err); k[0] != character.PreparationChangeNotAllowedNow {
				t.Errorf("%v %v: cantrip kinds = %v [FR-053]", class, trig, k)
			}
		}
	}
}
