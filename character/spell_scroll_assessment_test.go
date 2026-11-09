package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func profile(f character.ScrollReaderFacts) character.ScrollReaderProfile {
	return character.MustScrollReaderProfile(f)
}

func withAny(f character.ScrollReaderFacts) character.ScrollReaderFacts {
	f.AnyRoute = useMagicDevice()
	return f
}

func intArcana() character.ScrollCheckOption {
	return must(character.NewScrollCheckOption(ability.Intelligence, content.SkillArcana))
}

func wisOption() character.ScrollCheckOption {
	return must(character.NewScrollCheckOption(ability.Wisdom, 0))
}

// Requirement: per case (case labels are those of spell-scroll-use.md, "Case tables")
// Path: per case.
func TestAssessScrollReadingReady(t *testing.T) {
	t.Parallel()
	mc := readerFacts(3, classRoute("wizard", ability.Intelligence), classRoute("cleric", ability.Wisdom))
	warlock := readerFacts(5, classRoute("warlock", ability.Charisma))
	warlock.GrantedCastLevel = 6
	cases := []struct {
		name, path string
		facts      character.ScrollReaderFacts
		level      int
		classes    []string
		dc, atk    int
		ability    ability.Ability
	}{
		{"A1 level 3 at W5", "happy", w5(), 3, []string{"wizard"}, 15, 7, ability.Intelligence},
		{"A4 cantrip at W5", "happy", w5(), 0, []string{"wizard"}, 13, 5, ability.Intelligence},
		{"A8 cantrip with no slots", "unhappy (boundary)", readerFacts(0, classRoute("wizard", ability.Intelligence)), 0, []string{"wizard"}, 13, 5, ability.Intelligence},
		{"A9 Use Magic Device, cantrip off the list", "happy", withAny(character.ScrollReaderFacts{UnderstandsWrittenLanguage: true}), 0, []string{"cleric"}, 13, 5, ability.Intelligence},
		{"A10 Use Magic Device, level 1 off the list", "unhappy (boundary)", withAny(character.ScrollReaderFacts{UnderstandsWrittenLanguage: true}), 1, []string{"bard"}, 13, 5, ability.Intelligence},
		{"A13 multiclass uses the other class's list", "unhappy (rule interaction)", mc, 3, []string{"cleric"}, 15, 7, ability.Wisdom},
		{"A15 first reliable route is the Wizard route", "unhappy (rule interaction)", withAny(w5()), 3, []string{"wizard"}, 15, 7, ability.Intelligence},
		{"A21 granted level 6 is castable", "unhappy (boundary)", warlock, 6, []string{"warlock"}, 17, 9, ability.Charisma},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := character.AssessScrollReading(profile(c.facts), scrollOf(t, c.level, character.FoundScroll{}, c.classes...), character.ItemIntact).(character.ScrollReady)
			if !ok {
				t.Fatalf("result is not ScrollReady [%s]", c.path)
			}
			terms := got.Terms()
			if terms.SaveDC() != c.dc || terms.AttackBonus() != c.atk || terms.SpellcastingAbility() != c.ability || int(terms.CastAtLevel()) != c.level {
				t.Errorf("terms = DC %d %+d %v level %d, want DC %d %+d %v level %d", terms.SaveDC(), terms.AttackBonus(), terms.SpellcastingAbility(), terms.CastAtLevel(), c.dc, c.atk, c.ability, c.level)
			}
		})
	}
}

// Requirement: per case (case labels are those of spell-scroll-use.md, "Case tables")
// Path: per case.
func TestAssessScrollReadingCheckRequired(t *testing.T) {
	t.Parallel()
	mc := readerFacts(3, classRoute("wizard", ability.Intelligence), classRoute("cleric", ability.Wisdom))
	warlock := readerFacts(5, classRoute("warlock", ability.Charisma))
	warlock.GrantedCastLevel = 6
	cases := []struct {
		name, path string
		facts      character.ScrollReaderFacts
		level      int
		classes    []string
		dc         int
		options    []character.ScrollCheckOption
	}{
		{"A2 level 4 at W5", "unhappy (boundary)", w5(), 4, []string{"wizard"}, 14, []character.ScrollCheckOption{intOption()}},
		{"A3 level 9 at W5", "unhappy (boundary)", w5(), 9, []string{"wizard"}, 19, []character.ScrollCheckOption{intOption()}},
		{"A7 level 1 with no slots", "unhappy (boundary)", readerFacts(0, classRoute("wizard", ability.Intelligence)), 1, []string{"wizard"}, 11, []character.ScrollCheckOption{intOption()}},
		{"A11 Use Magic Device level 2", "unhappy (boundary)", withAny(character.ScrollReaderFacts{UnderstandsWrittenLanguage: true}), 2, []string{"cleric"}, 12, []character.ScrollCheckOption{intArcana()}},
		{"A12 Use Magic Device level 9", "unhappy (boundary)", withAny(character.ScrollReaderFacts{UnderstandsWrittenLanguage: true}), 9, []string{"wizard"}, 19, []character.ScrollCheckOption{intArcana()}},
		{"A14 spell on two lists", "unhappy (rule interaction)", mc, 4, []string{"wizard", "cleric"}, 14, []character.ScrollCheckOption{intOption(), wisOption()}},
		{"A16 class route then Use Magic Device", "unhappy (rule interaction)", withAny(w5()), 4, []string{"wizard"}, 14, []character.ScrollCheckOption{intOption(), intArcana()}},
		{"A22 granted level 6 is not enough for 7", "unhappy (boundary)", warlock, 7, []string{"warlock"}, 17, []character.ScrollCheckOption{must(character.NewScrollCheckOption(ability.Charisma, 0))}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := character.AssessScrollReading(profile(c.facts), scrollOf(t, c.level, character.FoundScroll{}, c.classes...), character.ItemIntact).(character.ScrollCheckRequired)
			if !ok {
				t.Fatalf("result is not ScrollCheckRequired [%s]", c.path)
			}
			if got.DC() != c.dc || !reflect.DeepEqual(got.Options(), c.options) {
				t.Errorf("DC %d options %v, want DC %d options %v", got.DC(), got.Options(), c.dc, c.options)
			}
			for _, o := range c.options {
				terms, ok := got.TermsFor(o)
				if !ok || terms.SpellcastingAbility() != o.Ability() || terms.CastAtLevel() != content.SpellLevel(c.level) {
					t.Errorf("TermsFor(%v) = %+v, %v", o, terms, ok)
				}
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-008, FR-009 - TermsFor refuses an option that was not offered, and Options returns a copy.
// Path: unhappy (invalid input).
func TestScrollCheckRequiredTermsForUnknownOption(t *testing.T) {
	t.Parallel()
	got := character.AssessScrollReading(profile(w5()), scrollOf(t, 4, character.FoundScroll{}, "wizard"), character.ItemIntact).(character.ScrollCheckRequired)
	if _, ok := got.TermsFor(wisOption()); ok {
		t.Error("TermsFor accepted an option that was not offered")
	}
	got.Options()[0] = wisOption()
	if got.Options()[0] != intOption() {
		t.Error("Options() exposes the internal slice")
	}
}

// Requirement: per case (A5, A6, A17 to A19)
// Path: per case.
func TestAssessScrollReadingRefusals(t *testing.T) {
	t.Parallel()
	noLanguage := w5()
	noLanguage.UnderstandsWrittenLanguage = false
	restricted := w5()
	restricted.DescriptionRestricts = true
	cases := []struct {
		name, req, path string
		facts           character.ScrollReaderFacts
		state           character.MagicItemUseState
		classes         []string
		want            character.ScrollAssessmentResult
	}{
		{"A5 spell on no list of the reader", "FR-004", "unhappy", w5(), character.ItemIntact, []string{"cleric"}, character.ScrollNotOnSpellList{}},
		{"A6 reader with no routes", "FR-005", "unhappy (empty and minimal)", character.ScrollReaderFacts{UnderstandsWrittenLanguage: true}, character.ItemIntact, []string{"wizard"}, character.ScrollNotOnSpellList{}},
		{"A17 cannot read", "FR-003", "unhappy (state error)", noLanguage, character.ItemIntact, []string{"wizard"}, character.ScrollNotReadable{Reason: character.MagicItemScrollNotReadable{Restricted: false}}},
		{"A18 description restricts", "FR-003", "unhappy (state error)", restricted, character.ItemIntact, []string{"wizard"}, character.ScrollNotReadable{Reason: character.MagicItemScrollNotReadable{Restricted: true}}},
		{"A19 consumed", "FR-013", "unhappy (state error)", w5(), character.ItemConsumed, []string{"wizard"}, character.ScrollAlreadyUsed{}},
		{"consumed wins over unreadable", "FR-013", "unhappy (state error)", noLanguage, character.ItemConsumed, []string{"wizard"}, character.ScrollAlreadyUsed{}},
		{"unreadable wins over off the list", "FR-003", "unhappy (state error)", restricted, character.ItemIntact, []string{"cleric"}, character.ScrollNotReadable{Reason: character.MagicItemScrollNotReadable{Restricted: true}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.AssessScrollReading(profile(c.facts), scrollOf(t, 1, character.FoundScroll{}, c.classes...), c.state)
			if got != c.want {
				t.Errorf("result = %#v, want %#v [spell-scroll-use.md %s, %s]", got, c.want, c.req, c.path)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-022 - Use Magic Device waives the spell list only; an unreadable scroll stays unreadable.
// Path: unhappy (rule interaction).
func TestAnyScrollRouteDoesNotWaiveReading(t *testing.T) {
	t.Parallel()
	got := character.AssessScrollReading(profile(character.ScrollReaderFacts{AnyRoute: useMagicDevice()}), scrollOf(t, 1, character.FoundScroll{}, "bard"), character.ItemIntact)
	if _, ok := got.(character.ScrollNotReadable); !ok {
		t.Errorf("result = %#v, want ScrollNotReadable", got)
	}
}

// Requirement: spell-scroll-use.md FR-002, FR-023, FR-025, FR-026, A20, A23, B1, B2 (SRD 5.2.1, "Scribing Spell Scrolls", p. 103)
// - a scribed scroll's cast uses the scribe's numbers, not the table's; only a scribed cantrip has the scribe's caster level.
// Path: per case.
func TestScribedScrollTerms(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		level        int
		dc, atk      int
		casterLevel  character.ScrollCasterLevel
		tableDiffers bool
	}{
		{"A20 level 2 scribed 15 +7, not the table's 13 +5", 2, 15, 7, character.ScrollLowestCasterLevel{}, true},
		{"A23/B1 level 1 scribed 15 +7", 1, 15, 7, character.ScrollLowestCasterLevel{}, true},
		{"B2 scribed cantrip keeps the scribe's level", 0, 15, 7, character.NewScribedCantripCasterLevel(character.MustLevel(11)), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			row := character.SpellScrollTableRow(content.SpellLevel(c.level))
			if c.tableDiffers && row.SaveDC() == c.dc && row.AttackBonus() == c.atk {
				t.Fatal("fixture: the scribed numbers equal the table's")
			}
			got := character.AssessScrollReading(profile(w5()), scrollOf(t, c.level, scribedOrigin(t, 11), "wizard"), character.ItemIntact).(character.ScrollReady).Terms()
			if got.SaveDC() != c.dc || got.AttackBonus() != c.atk || got.CasterLevel() != c.casterLevel {
				t.Errorf("terms = DC %d %+d %v, want DC %d %+d %v", got.SaveDC(), got.AttackBonus(), got.CasterLevel(), c.dc, c.atk, c.casterLevel)
			}
		})
	}
}

// Requirement: spell-scroll-use.md FR-014, FR-025 - a found scroll, cantrip or not, has the lowest caster level; Material components are waived and Verbal and Somatic are not.
// Path: happy.
func TestFoundScrollTermsFields(t *testing.T) {
	t.Parallel()
	for _, level := range []int{0, 3} {
		terms := character.AssessScrollReading(profile(w5()), scrollOf(t, level, character.FoundScroll{}, "wizard"), character.ItemIntact).(character.ScrollReady).Terms()
		if terms.CasterLevel() != (character.ScrollLowestCasterLevel{}) || terms.SpellID() != content.MustSpellID("scroll_spell") {
			t.Errorf("level %d: terms = %+v", level, terms)
		}
		if got := terms.WaivedComponents(); !reflect.DeepEqual(got, []character.ComponentKind{character.ComponentMaterial}) {
			t.Errorf("WaivedComponents() = %v, want exactly material", got)
		}
	}
}

// Requirement: spell-scroll-use.md FR-033 - CastTerms gives the form CastSpell takes: the cantrip scaling level is 0 for level 1+,
// 1 for a found cantrip and the scribe's level for a scribed cantrip; the zero terms are an error.
// Path: unhappy (boundary).
func TestScrollCastTermsConvertForCastSpell(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		scroll  character.SpellScroll
		scaling int
	}{
		{"level 3 found", scrollOf(t, 3, character.FoundScroll{}, "wizard"), 0},
		{"found cantrip", scrollOf(t, 0, character.FoundScroll{}, "wizard"), 1},
		{"scribed cantrip", scrollOf(t, 0, scribedOrigin(t, 11), "wizard"), 11},
	}
	for _, c := range cases {
		terms := character.AssessScrollReading(profile(w5()), c.scroll, character.ItemIntact).(character.ScrollReady).Terms()
		got, err := terms.CastTerms()
		if err != nil || got.CantripScalingLevel() != c.scaling || got.SaveDC() != terms.SaveDC() || got.AttackBonus() != terms.AttackBonus() {
			t.Errorf("%s: CastTerms() = %+v, %v, want scaling %d", c.name, got, err, c.scaling)
		}
	}
	if _, err := (character.ScrollCastTerms{}).CastTerms(); err == nil {
		t.Error("the zero terms converted")
	}
}

// Requirement: spell-scroll-use.md FR-034 - same inputs give equal results and no input is changed.
// Path: unhappy (determinism).
func TestAssessScrollReadingIsDeterministic(t *testing.T) {
	t.Parallel()
	p, s := profile(withAny(w5())), scrollOf(t, 4, character.FoundScroll{}, "wizard")
	a, b := character.AssessScrollReading(p, s, character.ItemIntact), character.AssessScrollReading(p, s, character.ItemIntact)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("results differ: %#v vs %#v", a, b)
	}
}
