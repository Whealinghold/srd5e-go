package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case (doc/specs/magic-item-rules.md FR-110 and table "Prerequisite reading").
// Path: per case.
func TestAttunementPrerequisiteReader(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Rules" (Attunement Prerequisites), p. 205`
	classes := func(names ...string) content.AttunementPrerequisite {
		ids := make([]content.ClassID, len(names))
		for i, n := range names {
			ids[i] = classOf(n)
		}
		return must(content.NewClassPrerequisite(ids))
	}
	cases := []struct {
		name, requirement, path string
		in                      content.Attunement
		want                    content.AttunementPrerequisite
	}{
		{"none", p, "happy", content.AttunementRequired{}, content.NoPrerequisite{}},
		{"spellcaster", p, "happy", requiredBy(t, "a Spellcaster"), content.SpellcasterPrerequisite{}},
		{"one class", p, "happy", requiredBy(t, "a Wizard"), classes("wizard")},
		{"list of classes", p, "unhappy (rule interaction)", requiredBy(t, "a Cleric, Druid, or Paladin"), classes("cleric", "druid", "paladin")},
		{"two classes with or", p, "unhappy (rule interaction)", requiredBy(t, "a Bard or Warlock"), classes("bard", "warlock")},
		{"not a class", p, "unhappy (rule interaction)", requiredBy(t, "a Dwarf or a Creature Attuned to a Belt of Dwarvenkind"), must(content.NewDescribedPrerequisite("a Dwarf or a Creature Attuned to a Belt of Dwarvenkind"))},
		{"one non-class name in a list", p, "unhappy (rule interaction)", requiredBy(t, "a Cleric or Dwarf"), must(content.NewDescribedPrerequisite("a Cleric or Dwarf"))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.AttunementPrerequisiteReader{}.Read(c.in)
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Errorf("Read = %#v, %v, want %#v [%s]", got, err, c.want, c.requirement)
			}
		})
	}
	for _, in := range []content.Attunement{content.NoAttunement{}, nil} {
		if got, err := (character.AttunementPrerequisiteReader{}).Read(in); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Read(%#v) = %v, %v", in, got, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-044 to FR-046 and table "Attunement prerequisite").
// Path: per case.
func TestAttunementEligibility(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Magic Item Rules" (Attunement Prerequisites), p. 205`
	cl := func(names ...string) content.AttunementPrerequisite {
		ids := make([]content.ClassID, len(names))
		for i, n := range names {
			ids[i] = classOf(n)
		}
		return must(content.NewClassPrerequisite(ids))
	}
	described := must(content.NewDescribedPrerequisite("a Dwarf"))
	cases := []struct {
		name, requirement, path string
		prereq                  content.AttunementPrerequisite
		who                     character.AttuneCandidate
		judgement               character.PrerequisiteJudgement
		want, missing           bool
	}{
		{"class member, one of several", p, "happy", cl("bard", "cleric", "druid"), candidate(t, character.CannotCast, "cleric"), 0, true, false},
		{"multiclass member", p, "happy", cl("cleric"), candidate(t, character.CannotCast, "druid", "cleric"), 0, true, false},
		{"other class", p, "unhappy (rule interaction)", cl("cleric"), candidate(t, character.CannotCast, "fighter"), 0, false, false},
		{"no classes", p, "unhappy (empty case)", cl("cleric"), candidate(t, character.CannotCast), 0, false, false},
		{"spellcaster can cast", p, "unhappy (rule interaction)", content.SpellcasterPrerequisite{}, candidate(t, character.CanCast), 0, true, false},
		{"spellcaster casts only via items", p, "unhappy (rule interaction)", content.SpellcasterPrerequisite{}, candidate(t, character.CannotCast), 0, false, false},
		{"described, met", p, "unhappy (missing content)", described, candidate(t, character.CannotCast), character.PrerequisiteMet, true, false},
		{"described, not met", p, "unhappy (missing content)", described, candidate(t, character.CannotCast), character.PrerequisiteNotMet, false, false},
		{"described, not supplied", p, "unhappy (missing content)", described, candidate(t, character.CannotCast), character.PrerequisiteNotSupplied, false, true},
		{"no prerequisite", p, "unhappy (empty case)", content.NoPrerequisite{}, candidate(t, character.CannotCast), 0, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := character.AttunementEligibility{}
			got, err := e.IsMet(c.prereq, c.who, c.judgement)
			if err != nil || got != c.want || e.JudgementMissing(c.prereq, c.judgement) != c.missing {
				t.Errorf("IsMet = %v, %v, missing %v; want %v, %v [%s]", got, err, e.JudgementMissing(c.prereq, c.judgement), c.want, c.missing, c.requirement)
			}
		})
	}
	for name, p := range map[string]content.AttunementPrerequisite{"empty class set": content.ClassPrerequisite{}, "nil": nil} {
		if _, err := (character.AttunementEligibility{}).IsMet(p, candidate(t, character.CanCast), 0); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := character.NewAttuneCandidate(nil, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero spellcasting: %v", err)
	}
	if _, err := character.NewAttuneCandidate([]content.ClassID{""}, character.CanCast); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid class: %v", err)
	}
	c := candidate(t, character.CanCast, "wizard")
	if cls := c.Classes(); len(cls) != 1 || c.Spellcasting() != character.CanCast {
		t.Errorf("accessors: %v", cls)
	} else {
		cls[0] = "x"
		if c.Classes()[0] != "wizard" {
			t.Error("Classes is not a copy")
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-052, FR-053 and table "Attunement ending").
// Path: per case.
func TestAttunementEndingChecks(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Ending Attunement", p. 102`
	cases := []struct {
		name, requirement, path string
		feet, hours             int
		ends                    bool
	}{
		{"exactly 100 feet for 24 hours", p, "unhappy (boundary)", 100, 24, false},
		{"101 feet for 24 hours", p, "unhappy (boundary)", 101, 24, true},
		{"101 feet for 23 hours", p, "unhappy (boundary)", 101, 23, false},
		{"far for a long time", p, "happy", 5000, 240, true},
		{"close for a long time", p, "unhappy (boundary)", 0, 500, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			v, err := character.AttunementEnding{}.CheckDistance(c.feet, c.hours)
			want := character.AttunementEndingVerdict(character.AttunementContinues{})
			if c.ends {
				want = character.AttunementEnds{Reason: character.AttunementEndedItemTooFarAway}
			}
			if err != nil || v != want {
				t.Errorf("CheckDistance(%d, %d) = %v, %v, want %v [%s]", c.feet, c.hours, v, err, want, c.requirement)
			}
		})
	}
	for _, in := range [][2]int{{-1, 5}, {101, -1}} {
		if _, err := (character.AttunementEnding{}).CheckDistance(in[0], in[1]); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("CheckDistance(%v): %v", in, err)
		}
	}
	prereq := content.AttunementPrerequisite(content.SpellcasterPrerequisite{})
	described := must(content.NewDescribedPrerequisite("a Dwarf"))
	for name, c := range map[string]struct {
		pre  content.AttunementPrerequisite
		who  character.AttuneCandidate
		j    character.PrerequisiteJudgement
		ends bool
	}{
		"prerequisite lost":                    {prereq, candidate(t, character.CannotCast), 0, true},
		"prerequisite still met":               {prereq, candidate(t, character.CanCast), 0, false},
		"described and not supplied continues": {described, candidate(t, character.CanCast), character.PrerequisiteNotSupplied, false},
		"described and not met ends":           {described, candidate(t, character.CanCast), character.PrerequisiteNotMet, true},
	} {
		v, err := character.AttunementEnding{}.CheckPrerequisite(c.pre, c.who, c.j)
		_, ends := v.(character.AttunementEnds)
		if err != nil || ends != c.ends {
			t.Errorf("%s: %v, %v", name, v, err)
		}
	}
	if v, _ := (character.AttunementEnding{}).CheckPrerequisite(prereq, candidate(t, character.CannotCast), 0); v != (character.AttunementEnds{Reason: character.AttunementEndedPrerequisiteNoLongerMet}) {
		t.Errorf("reason = %v", v)
	}
	if _, err := (character.AttunementEnding{}).CheckPrerequisite(content.ClassPrerequisite{}, candidate(t, character.CanCast), 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid prerequisite: %v", err)
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-039 - properties give full magic to an item needing no Attunement or an attuned
// creature, only nonmagical benefits to a not attuned creature unless the description says otherwise, and nothing once consumed.
// Path: per case.
func TestMagicalPropertiesAccess(t *testing.T) {
	t.Parallel()
	req := content.AttunementRequired{}
	cases := []struct {
		name     string
		a        content.Attunement
		standing character.AttunementStanding
		without  bool
		use      character.MagicItemUseState
		want     character.MagicalPropertiesAccessLevel
	}{
		{"no attunement", content.NoAttunement{}, character.StandingNotAttuned, false, character.ItemIntact, character.AccessFullMagical},
		{"attuned", req, character.StandingAttuned, false, character.ItemIntact, character.AccessFullMagical},
		{"not attuned", req, character.StandingNotAttuned, false, character.ItemIntact, character.AccessNonmagicalBenefitsOnly},
		{"property works without attunement", req, character.StandingNotAttuned, true, character.ItemIntact, character.AccessFullMagical},
		{"consumed, attuned", req, character.StandingAttuned, false, character.ItemConsumed, character.AccessNoMagic},
		{"consumed, no attunement", content.NoAttunement{}, character.StandingNotAttuned, true, character.ItemConsumed, character.AccessNoMagic},
	}
	for _, c := range cases {
		if got := (character.MagicalPropertiesAccess{}).Of(c.a, c.standing, c.without, c.use); got != c.want {
			t.Errorf("%s: %v, want %v (SRD 5.2.1, \"Attunement\", p. 102)", c.name, got, c.want)
		}
	}
}
