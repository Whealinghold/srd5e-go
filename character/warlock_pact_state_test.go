package character_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

var warCreature = character.MustIdentifier("npc-1")

func warPactChanged(t *testing.T, tr character.WarlockPactTransition) character.WarlockPactChanged {
	t.Helper()
	c, ok := tr.(character.WarlockPactChanged)
	if !ok {
		t.Fatalf("transition = %#v, want WarlockPactChanged", tr)
	}
	return c
}

func warPactRefused(t *testing.T, tr character.WarlockPactTransition) character.WarlockRejection {
	t.Helper()
	r, ok := tr.(character.WarlockPactRefused)
	if !ok {
		t.Fatalf("transition = %#v, want WarlockPactRefused", tr)
	}
	return r.Rejection()
}

// warPact restores a pact state from parts.
func warPact(parts character.WarlockPactStateParts) character.WarlockPactState {
	return must(character.RestoreWarlockPactState(parts))
}

// warBook is parts holding a Book of Shadows, with the Gift of the Protectors page and the names when page is true.
func warBook(page bool, names ...character.Identifier) character.WarlockPactStateParts {
	return character.WarlockPactStateParts{BookCantrips: []content.SpellID{"light", "mage_hand", "prestidigitation"}, BookSpells: []content.SpellID{"alarm", "detect_magic"},
		BookPage: page, PageNames: names}
}

// Requirement: doc/specs/class-warlock.md FR-158 - the pact state round-trips through its parts and the FeatureStates key warlock_pact; the zero state holds no bond, familiar, book or link; Equal compares by value (slices included) and is false for another type; Parts returns copies.
// Path: happy.
func TestWarlockPactStateRoundTrips(t *testing.T) {
	t.Parallel()
	parts := warBook(true, warCreature)
	parts.Bond, parts.BondConjured = "longsword", true
	parts.Familiar, parts.FamiliarNormal, parts.FamiliarSpeed = "cat", true, character.FamiliarSwims
	parts.Link, parts.LinkStatus = warCreature, character.GazeLinkActive
	s := warPact(parts)
	if got := s.Parts(); !reflect.DeepEqual(got, parts) {
		t.Errorf("Parts = %+v, want %+v", got, parts)
	}
	if !s.Equal(warPact(parts)) || s.Equal(character.WarlockPactState{}) || s.Equal(character.WarlockState{}) {
		t.Error("Equal must compare by value and by type")
	}
	other := parts
	other.PageNames = nil
	if s.Equal(warPact(other)) {
		t.Error("Equal must see the page names")
	}
	if b := s.Bond(); !b.Bonded() || b.Weapon() != "longsword" || !b.Conjured() || len(b.Benefits()) != 2 {
		t.Errorf("Bond = %+v", b)
	}
	if f, ok := s.Familiar().Form(); !ok || f.Monster() != "cat" || !f.IsNormal() {
		t.Errorf("Form = %+v, %v", f, ok)
	}
	if sp, ok := s.Familiar().Speed(); !ok || sp != character.FamiliarSwims {
		t.Errorf("Speed = %v, %v", sp, ok)
	}
	book := s.Book()
	if !book.Conjured() || !book.HasPage() || len(book.Cantrips()) != 3 || len(book.Spells()) != 2 || len(book.Names()) != 1 {
		t.Errorf("Book = %+v", book)
	}
	book.Cantrips()[0], book.Names()[0] = "changed", character.MustIdentifier("changed")
	if s.Book().Cantrips()[0] != "light" || s.Book().Names()[0] != warCreature {
		t.Error("the book accessors must return copies")
	}
	if s.Link().Creature() != warCreature || s.Link().Status() != character.GazeLinkActive {
		t.Errorf("Link = %+v", s.Link())
	}
	zero := character.WarlockPactState{}
	if _, ok := zero.Familiar().Form(); ok || zero.Bond().Bonded() || zero.Book().Conjured() || zero.Link().Active() || len(zero.Bond().Benefits()) != 0 {
		t.Error("the zero state must hold nothing")
	}
	if _, ok := zero.Familiar().Speed(); ok {
		t.Error("the zero familiar has no Speed")
	}
	if character.WarlockPactStateKey().Name() != "warlock_pact" {
		t.Errorf("key = %q", character.WarlockPactStateKey().Name())
	}
}

// Requirement: doc/specs/class-warlock.md FR-098, FR-102, FR-158 - RestoreWarlockPactState refuses with ErrInvalidInput a conjured flag with no bond, a Speed or normal flag with no familiar, a Speed outside Fly and Swim, a book without exactly three cantrips and two spells (or half a book), names without a page or a book, a duplicate or zero name, an invalid id, a link status without a creature, a creature without a status and a status outside the set.
// Path: per case (unhappy: invalid input).
func TestRestoreWarlockPactStateRefusesInvalidParts(t *testing.T) {
	t.Parallel()
	with := func(f func(p *character.WarlockPactStateParts)) character.WarlockPactStateParts {
		p := warBook(true, warCreature)
		f(&p)
		return p
	}
	cases := map[string]character.WarlockPactStateParts{
		"conjured, no bond":    {BondConjured: true},
		"invalid bond":         {Bond: "Bad Id"},
		"speed, no familiar":   {FamiliarSpeed: character.FamiliarFlies},
		"normal, no familiar":  {FamiliarNormal: true},
		"invalid familiar":     {Familiar: "Bad Id"},
		"speed out of the set": {Familiar: "cat", FamiliarSpeed: 3},
		"negative speed":       {Familiar: "cat", FamiliarSpeed: -1},
		"two cantrips":         with(func(p *character.WarlockPactStateParts) { p.BookCantrips = p.BookCantrips[:2] }),
		"one spell":            with(func(p *character.WarlockPactStateParts) { p.BookSpells = p.BookSpells[:1] }),
		"spells, no cantrips":  {BookSpells: []content.SpellID{"alarm", "detect_magic"}},
		"cantrips, no spells":  {BookCantrips: []content.SpellID{"light", "mage_hand", "prestidigitation"}},
		"four cantrips":        with(func(p *character.WarlockPactStateParts) { p.BookCantrips = append(p.BookCantrips, "guidance") }),
		"three spells":         with(func(p *character.WarlockPactStateParts) { p.BookSpells = append(p.BookSpells, "bless") }),
		"invalid spell id":     with(func(p *character.WarlockPactStateParts) { p.BookSpells[0] = "Bad Id" }),
		"page, no book":        {BookPage: true},
		"names, no page":       with(func(p *character.WarlockPactStateParts) { p.BookPage = false }),
		"duplicate name": with(func(p *character.WarlockPactStateParts) {
			p.PageNames = []character.Identifier{warCreature, warCreature}
		}),
		"zero name":             with(func(p *character.WarlockPactStateParts) { p.PageNames = []character.Identifier{{}} }),
		"status, no creature":   {LinkStatus: character.GazeLinkActive},
		"creature, no status":   {Link: warCreature},
		"status out of the set": {Link: warCreature, LinkStatus: 3},
		"negative status":       {Link: warCreature, LinkStatus: -1},
	}
	for name, parts := range cases {
		if _, err := character.RestoreWarlockPactState(parts); !srderrIsInvalid(err) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
	for name, parts := range map[string]character.WarlockPactStateParts{"empty": {}, "book without page": warBook(false), "lapsed link": {Link: warCreature, LinkStatus: character.GazeLinkLapsed},
		"fly speed": {Familiar: "imp", FamiliarSpeed: character.FamiliarFlies}} {
		if _, err := character.RestoreWarlockPactState(parts); err != nil {
			t.Errorf("%s: err = %v, want none", name, err)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-072, FR-151 - an active Gaze of Two Minds link lets the Warlock cast from the linked space within 60 feet (61 feet does not) and passes on the creature's special senses; a lapsed link and no link do neither; GazeLinkOf refuses a zero creature or an invalid status by returning the zero link.
// Path: per case (boundary: 60 and 61 feet).
func TestSensorLinkReach(t *testing.T) {
	t.Parallel()
	senses := []content.SenseEntry{{}}
	active := character.GazeLinkOf(warCreature, character.GazeLinkActive)
	lapsed := character.GazeLinkOf(warCreature, character.GazeLinkLapsed)
	for _, c := range []struct {
		name     string
		link     character.SensorLink
		feet     int
		wantCast bool
		wantSens int
	}{{"active at 60", active, 60, true, 1}, {"active at 61", active, 61, false, 1}, {"active at 0", active, 0, true, 1}, {"lapsed", lapsed, 0, false, 0}, {"none", character.SensorLink{}, 0, false, 0}} {
		if got := c.link.CanCastFromLinkedSpace(c.feet); got != c.wantCast {
			t.Errorf("%s: CanCastFromLinkedSpace = %v, want %v", c.name, got, c.wantCast)
		}
		if got := c.link.LinkedSenses(senses); len(got) != c.wantSens || got == nil {
			t.Errorf("%s: LinkedSenses = %v, want %d", c.name, got, c.wantSens)
		}
	}
	got := active.LinkedSenses(senses)
	got[0] = content.SenseEntry{}
	for _, bad := range []character.SensorLink{character.GazeLinkOf(character.Identifier{}, character.GazeLinkActive), character.GazeLinkOf(warCreature, 0), character.GazeLinkOf(warCreature, 3)} {
		if bad != (character.SensorLink{}) {
			t.Errorf("GazeLinkOf = %+v, want the zero link", bad)
		}
	}
	if !slices.Equal([]character.GazeLinkStatus{active.Status(), lapsed.Status()}, []character.GazeLinkStatus{character.GazeLinkActive, character.GazeLinkLapsed}) {
		t.Error("Status must report the status")
	}
}

// Requirement: doc/specs/class-warlock.md FR-084, FR-158 - the pact weapon benefits keep their numbering (1 and 2, so 0 stays "none"); Equal sees a difference in each field alone (bond, familiar, link, book page, cantrips, spells, names) and is false for another type even from the zero state.
// Path: per case (boundary: one field differs).
func TestWarlockPactStateEqualSeesEachField(t *testing.T) {
	t.Parallel()
	if character.PactWeaponProficiency != 1 || character.PactWeaponSpellcastingFocus != 2 {
		t.Errorf("benefits = %d, %d, want 1, 2", character.PactWeaponProficiency, character.PactWeaponSpellcastingFocus)
	}
	if got := warPact(character.WarlockPactStateParts{Bond: "longsword"}).Bond().Benefits(); !slices.Equal(got, []character.PactWeaponBenefit{character.PactWeaponProficiency, character.PactWeaponSpellcastingFocus}) {
		t.Errorf("Benefits = %v", got)
	}
	base := warBook(true, warCreature)
	base.Bond, base.Familiar, base.Link, base.LinkStatus = "longsword", "cat", warCreature, character.GazeLinkActive
	s := warPact(base)
	other := character.MustIdentifier("npc-2")
	mods := map[string]func(p *character.WarlockPactStateParts){
		"bond":     func(p *character.WarlockPactStateParts) { p.Bond = "dagger" },
		"familiar": func(p *character.WarlockPactStateParts) { p.Familiar = "imp" },
		"link":     func(p *character.WarlockPactStateParts) { p.LinkStatus = character.GazeLinkLapsed },
		"page":     func(p *character.WarlockPactStateParts) { p.BookPage, p.PageNames = false, nil },
		"cantrips": func(p *character.WarlockPactStateParts) {
			p.BookCantrips = []content.SpellID{"light", "mage_hand", "guidance"}
		},
		"spells": func(p *character.WarlockPactStateParts) { p.BookSpells = []content.SpellID{"alarm", "bless"} },
		"names":  func(p *character.WarlockPactStateParts) { p.PageNames = []character.Identifier{other} },
	}
	for name, mod := range mods {
		p := warBook(true, warCreature)
		p.Bond, p.Familiar, p.Link, p.LinkStatus = base.Bond, base.Familiar, base.Link, base.LinkStatus
		mod(&p)
		if s.Equal(warPact(p)) {
			t.Errorf("%s: Equal = true for a state differing only there", name)
		}
	}
	if (character.WarlockPactState{}).Equal(character.WarlockState{}) {
		t.Error("the zero pact state must not equal another type")
	}
}
