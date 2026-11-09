package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewScrollReaderProfileRejectsInvalidFacts(t *testing.T) {
	t.Parallel()
	wiz := classRoute("wizard", ability.Intelligence)
	cases := []struct {
		name  string
		facts character.ScrollReaderFacts
	}{
		{"slot level above 9", character.ScrollReaderFacts{HighestSlotLevel: 10}},
		{"slot level below 0", character.ScrollReaderFacts{HighestSlotLevel: -1}},
		{"granted level above 9", character.ScrollReaderFacts{GrantedCastLevel: 10}},
		{"granted level below 0", character.ScrollReaderFacts{GrantedCastLevel: -1}},
		{"two routes of one class", character.ScrollReaderFacts{ClassRoutes: []character.ScrollClassRoute{wiz, wiz}}},
		{"a zero class route", character.ScrollReaderFacts{ClassRoutes: []character.ScrollClassRoute{{}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: spell-scroll-use.md FR-032 (architecture.md section 3, 6)
			// Path: unhappy (invalid input)
			if _, err := character.NewScrollReaderProfile(c.facts); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-032 - the boundary levels 0 and 9 are accepted.
// Path: unhappy (boundary).
func TestNewScrollReaderProfileAcceptsLevelBounds(t *testing.T) {
	t.Parallel()
	for _, l := range []content.SpellLevel{0, 9} {
		p, err := character.NewScrollReaderProfile(character.ScrollReaderFacts{HighestSlotLevel: l, GrantedCastLevel: l})
		if err != nil || p.CastableLevel() != l {
			t.Errorf("level %d: profile castable %d, err %v", l, p.CastableLevel(), err)
		}
	}
}

// Requirement: spell-scroll-use.md FR-032 - MustScrollReaderProfile panics with the constructor's error.
// Path: unhappy (invalid input).
func TestMustScrollReaderProfilePanicsWithTheError(t *testing.T) {
	t.Parallel()
	defer func() {
		err, ok := recover().(error)
		if !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want an ErrInvalidInput error", err)
		}
	}()
	character.MustScrollReaderProfile(character.ScrollReaderFacts{HighestSlotLevel: 10})
}

// Requirement: spell-scroll-use.md FR-007, FR-038 - the castable level is the greater of the slot level and the granted level.
// Path: unhappy (boundary).
func TestScrollReaderCastableLevelIsTheGreater(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ slots, granted, want content.SpellLevel }{{5, 6, 6}, {6, 5, 6}, {4, 4, 4}} {
		p := character.MustScrollReaderProfile(character.ScrollReaderFacts{HighestSlotLevel: c.slots, GrantedCastLevel: c.granted})
		if p.CastableLevel() != c.want {
			t.Errorf("slots %d granted %d: castable %d, want %d", c.slots, c.granted, p.CastableLevel(), c.want)
		}
	}
}

// Requirement: spell-scroll-use.md FR-032 - routes are copied on construction and on read.
// Path: unhappy (immutability).
func TestScrollReaderProfileRoutesAreCopied(t *testing.T) {
	t.Parallel()
	wiz, cleric := classRoute("wizard", ability.Intelligence), classRoute("cleric", ability.Wisdom)
	in := []character.ScrollClassRoute{wiz}
	p := character.MustScrollReaderProfile(character.ScrollReaderFacts{ClassRoutes: in, AnyRoute: useMagicDevice()})
	in[0] = cleric
	p.ClassRoutes()[0] = cleric
	if got := p.ClassRoutes(); len(got) != 1 || got[0] != wiz {
		t.Errorf("ClassRoutes() = %v, want the original wizard route", got)
	}
	if a, ok := p.AnyRoute(); !ok || a != useMagicDevice() {
		t.Errorf("AnyRoute() = %v, %v", a, ok)
	}
	if _, ok := (character.ScrollReaderProfile{}).AnyRoute(); ok {
		t.Error("the zero profile has an any-scroll route")
	}
}

// Requirement: per case
// Path: per case.
func TestRouteConstructorsRejectInvalidInput(t *testing.T) {
	t.Parallel()
	_, errOpt := character.NewScrollCheckOption(0, 0)
	_, errSkill := character.NewScrollCheckOption(ability.Wisdom, content.Skill(99))
	_, errClass := character.NewScrollClassRoute("", ability.Wisdom)
	_, errClassAbility := character.NewScrollClassRoute(content.MustClassID("cleric"), 0)
	_, errAnyAbility := character.NewAnyScrollRoute(0, 1, 0)
	_, errAnyLevel := character.NewAnyScrollRoute(ability.Intelligence, 10, 0)
	_, errAnySkill := character.NewAnyScrollRoute(ability.Intelligence, 1, content.Skill(99))
	for name, err := range map[string]error{"option ability": errOpt, "option skill": errSkill, "route class": errClass,
		"route ability": errClassAbility, "any ability": errAnyAbility, "any level": errAnyLevel, "any skill": errAnySkill} {
		// Requirement: spell-scroll-use.md FR-008, FR-019, FR-032 (architecture.md section 6)
		// Path: unhappy (invalid input)
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: spell-scroll-use.md FR-008, FR-019 - a bare ability check has no skill; Use Magic Device carries Arcana.
// Path: happy.
func TestRouteAccessors(t *testing.T) {
	t.Parallel()
	if _, has := intOption().Skill(); has || intOption().Ability() != ability.Intelligence {
		t.Error("a bare option reports a skill")
	}
	a := useMagicDevice()
	if s, has := must(character.NewScrollCheckOption(a.Ability(), a.CheckSkill())).Skill(); !has || s != content.SkillArcana || a.ReliableUpToLevel() != 1 || a.IsZero() {
		t.Errorf("any route = %+v", a)
	}
	r := classRoute("cleric", ability.Wisdom)
	if r.ClassID() != content.MustClassID("cleric") || r.Ability() != ability.Wisdom {
		t.Errorf("route = %+v", r)
	}
	if !(character.AnyScrollRoute{}).IsZero() {
		t.Error("the zero any-scroll route is not zero")
	}
}
