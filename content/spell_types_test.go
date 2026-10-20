package content_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const spellsSpec = "doc/specs/content-spells.md"

// The fixtures are synthetic: made-up names and texts, never SRD spells.

// Requirement: content-spells.md FR-009, FR-010, FR-011, FR-111 (SRD 5.2.1,
// "Spell Level", p. 104) - NewSpellLevel accepts 0 to 9 and rejects -1 and 10
// with a *SpellLevelError carrying the integer; IsCantrip is true only for 0;
// MustSpellLevel panics with the same error.
// Path: unhappy (boundary).
func TestSpellLevel(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n      int
		want   bool
		cantip bool
	}{{-1, false, false}, {0, true, true}, {1, true, false}, {9, true, false}, {10, false, false}} {
		l, err := content.NewSpellLevel(c.n)
		if (err == nil) != c.want || content.ValidSpellLevel(c.n) != c.want {
			t.Errorf("NewSpellLevel(%d) error = %v, want accepted=%v", c.n, err, c.want)
		}
		if !c.want {
			var le *content.SpellLevelError
			if !errors.As(err, &le) || le.Value != c.n || !errors.Is(err, srderr.ErrInvalidInput) ||
				!errors.Is(err, &content.SpellLevelError{Value: c.n}) || errors.Is(err, &content.SpellLevelError{Value: c.n + 1}) {
				t.Errorf("NewSpellLevel(%d) error = %v, want *SpellLevelError carrying %d", c.n, err, c.n)
			}
			if !panics(func() { content.MustSpellLevel(c.n) }) {
				t.Errorf("MustSpellLevel(%d) did not panic", c.n)
			}
			continue
		}
		if l.Int() != c.n || l.IsCantrip() != c.cantip || !l.Valid() || content.MustSpellLevel(c.n) != l {
			t.Errorf("level %d = %v (cantrip %v), want cantrip %v", c.n, l, l.IsCantrip(), c.cantip)
		}
	}
	if content.SpellLevel(10).Valid() || content.MinSpellLevel != 0 || content.MaxSpellLevel != 9 {
		t.Error("the limits or Valid are wrong [FR-111]")
	}
}

func panics(f func()) (p bool) {
	defer func() { p = recover() != nil }()
	f()
	return false
}

// Requirement: content-spells.md FR-012, FR-119 and schema FR-025 - each new
// vocabulary has its pinned members, an exact case-sensitive parser and
// String; the effect model enums parse their lower camel case spelling.
// Path: per case.
func TestSpellVocabularies(t *testing.T) {
	t.Parallel()
	type vocab struct {
		name  string
		want  []string
		parse func(string) (string, error)
	}
	for _, v := range []vocab{
		{"SpellSchool", []string{"Abjuration", "Conjuration", "Divination", "Enchantment", "Evocation", "Illusion", "Necromancy", "Transmutation"}, stringer(content.ParseSpellSchool)},
		{"DistanceUnit", []string{"feet", "miles"}, stringer(content.ParseDistanceUnit)},
		{"DurationLimit", []string{"fixed", "upTo", "concentrationUpTo"}, stringer(content.ParseDurationLimit)},
		{"EffectCoverage", []string{"complete", "partial"}, stringer(content.ParseEffectCoverage)},
		{"TargetSubject", []string{"creature", "object", "creatureOrObject"}, stringer(content.ParseTargetSubject)},
		{"SelfTargetPolicy", []string{"mayIncludeCaster", "excludesCaster"}, stringer(content.ParseSelfTargetPolicy)},
		{"SpellAttackKind", []string{"melee", "ranged"}, stringer(content.ParseSpellAttackKind)},
		{"SuccessEffect", []string{"nothingOnSuccess", "halfDamageOnSuccess"}, stringer(content.ParseSuccessEffect)},
		{"ModifierAddition", []string{"none", "spellcastingAbilityModifier"}, stringer(content.ParseModifierAddition)},
		{"TurnOwner", []string{"caster", "target"}, stringer(content.ParseTurnOwner)},
		{"TurnBoundary", []string{"start", "end"}, stringer(content.ParseTurnBoundary)},
		{"ZoneTrigger", []string{"creatureEnters", "creatureFirstEntryOnTurn", "creatureEndsTurnInside", "creatureStartsTurnInside", "zoneMovesIntoSpace", "travelsWithin"}, stringer(content.ParseZoneTrigger)},
		{"ZoneTargetScope", []string{"anyCreature", "enemiesOfCaster", "creaturesCasterCanSee"}, stringer(content.ParseZoneTargetScope)},
		{"ZoneActionKind", []string{"bonusAction", "magicAction", "withCasterMovement"}, stringer(content.ParseZoneActionKind)},
		{"DriftDirectionRule", []string{"awayFromCaster", "awayFromCasterChosen"}, stringer(content.ParseDriftDirectionRule)},
		{"DesignationPolicy", []string{"none", "atCastingOnly"}, stringer(content.ParseDesignationPolicy)},
	} {
		for _, m := range v.want {
			if got, err := v.parse(m); err != nil || got != m {
				t.Errorf("%s: Parse(%q) = (%q, %v), want it back [happy]", v.name, m, got, err)
			}
			for _, bad := range []string{strings.ToUpper(m), m + " ", " " + m} {
				if _, err := v.parse(bad); err == nil {
					t.Errorf("%s: Parse(%q) accepted [unhappy (invalid input)]", v.name, bad)
				}
			}
		}
		if _, err := v.parse(""); err == nil {
			t.Errorf("%s: Parse(\"\") accepted", v.name)
		}
	}
	if content.SpellSchool(0).Valid() || content.SpellSchool(9).Valid() || content.SpellSchool(9).String() != "unknown" {
		t.Error("SpellSchool 0 and 9 are not invalid [boundary]")
	}
	if content.DistanceUnit(3).Valid() || content.DurationLimit(0).Valid() || content.DurationLimit(4).String() != "unknown" {
		t.Error("an undefined DistanceUnit or DurationLimit is valid [boundary]")
	}
}

// castingLabel returns a distinct label for each variant of the closed sets, with no
// default: the compiler-visible exhaustive switches of FR-114.
func castingLabel(c content.CastingTime) string {
	switch c.(type) {
	case content.CastingTimeMagicAction:
		return "action"
	case content.CastingTimeBonusAction:
		return "bonus"
	case content.CastingTimeReaction:
		return "reaction"
	case content.CastingTimeMinutes:
		return "minutes"
	case content.CastingTimeHours:
		return "hours"
	}
	return "?"
}

func rangeLabel(r content.SpellRange) string {
	switch r.(type) {
	case content.SpellRangeSelf:
		return "self"
	case content.SpellRangeTouch:
		return "touch"
	case content.SpellRangeDistance:
		return "distance"
	case content.SpellRangeSight:
		return "sight"
	case content.SpellRangeUnlimited:
		return "unlimited"
	case content.SpellRangeSpecial:
		return "special"
	}
	return "?"
}

func durationLabel(d content.SpellDuration) string {
	switch d.(type) {
	case content.SpellDurationInstantaneous:
		return "instantaneous"
	case content.SpellDurationTimed:
		return "timed"
	case content.SpellDurationUntilDispelled:
		return "dispelled"
	case content.SpellDurationUntilDispelledOrTriggered:
		return "triggered"
	case content.SpellDurationSpecial:
		return "special"
	}
	return "?"
}

// Requirement: content-spells.md FR-026 to FR-029, FR-114 (SRD 5.2.1, "Casting
// Time", p. 105) - the five casting times have distinct labels, the factories
// return the variants, and reaction, bonus action trigger, minutes and hours
// reject blank triggers and amounts below 1.
// Path: unhappy (boundary).
func TestCastingTimes(t *testing.T) {
	t.Parallel()
	trig := must(content.NewCastingTimeBonusActionTriggered("when hit"))
	rea := must(content.NewCastingTimeReaction("when seen"))
	min1 := must(content.NewCastingTimeMinutes(1))
	hr := must(content.NewCastingTimeHours(8))
	for want, c := range map[string]content.CastingTime{
		"action": content.NewCastingTimeMagicAction(), "bonus": content.NewCastingTimeBonusAction(), "reaction": rea, "minutes": min1, "hours": hr,
	} {
		if got := castingLabel(c); got != want {
			t.Errorf("label = %q, want %q", got, want)
		}
	}
	if s, ok := trig.Trigger(); !ok || s != "when hit" {
		t.Errorf("Trigger() = (%q, %v)", s, ok)
	}
	if _, ok := content.NewCastingTimeBonusAction().Trigger(); ok {
		t.Error("a bonus action with no trigger has one")
	}
	if rea.Trigger() != "when seen" || min1.Amount() != 1 || hr.Amount() != 8 {
		t.Error("accessors")
	}
	for _, s := range []string{"", " ", "\t", "a\x00"} {
		_, e1 := content.NewCastingTimeBonusActionTriggered(s)
		_, e2 := content.NewCastingTimeReaction(s)
		if !errors.Is(e1, srderr.ErrInvalidInput) || !errors.Is(e2, srderr.ErrInvalidInput) {
			t.Errorf("trigger %q accepted: %v, %v [FR-027, FR-028]", s, e1, e2)
		}
	}
	for _, n := range []int{-1, 0} {
		_, e1 := content.NewCastingTimeMinutes(n)
		_, e2 := content.NewCastingTimeHours(n)
		wantBlockError(t, e1, "CastingTimeMinutes", content.BlockOutOfRange)
		wantBlockError(t, e2, "CastingTimeHours", content.BlockOutOfRange)
	}
	if content.ValidCastingTimeAmount(0) || !content.ValidCastingTimeAmount(1) {
		t.Error("ValidCastingTimeAmount boundary [FR-029]")
	}
}

// Requirement: content-spells.md FR-030, FR-033 - SpellCastingTimes is a single
// unlabelled time or at least two labelled modes; fewer, a blank label and a
// nil time are rejected; the modes are copied on construction and on read.
// Path: unhappy (invalid input).
func TestSpellCastingTimes(t *testing.T) {
	t.Parallel()
	act, hrs := content.CastingTime(content.NewCastingTimeMagicAction()), content.CastingTime(must(content.NewCastingTimeHours(8)))
	a, b := must(content.NewCastingTimeMode("One", act)), must(content.NewCastingTimeMode("Two", hrs))
	_, err := content.NewSpellCastingTimesModes(nil)
	wantBlockError(t, err, "SpellCastingTimesModes", content.BlockOutOfRange)
	_, err = content.NewSpellCastingTimesModes([]content.CastingTimeMode{a})
	wantBlockError(t, err, "SpellCastingTimesModes", content.BlockOutOfRange)
	_, err = content.NewSpellCastingTimesModes([]content.CastingTimeMode{a, {}})
	wantBlockError(t, err, "CastingTimeMode", content.BlockInvalidName)
	_, err = content.NewCastingTimeMode("", act)
	wantBlockError(t, err, "CastingTimeMode", content.BlockInvalidName)
	_, err = content.NewCastingTimeMode("x", nil)
	wantBlockError(t, err, "CastingTime", content.BlockMissing)
	_, err = content.NewSpellCastingTimesSingle(nil)
	wantBlockError(t, err, "CastingTime", content.BlockMissing)
	in := []content.CastingTimeMode{a, b}
	m := must(content.NewSpellCastingTimesModes(in))
	in[0] = b
	m.Modes()[1] = a
	if got := m.Modes(); !slices.Equal(got, []content.CastingTimeMode{a, b}) || got[0].Label() != "One" || got[1].Time() != hrs {
		t.Errorf("Modes() = %v after mutation, want the original [FR-086]", got)
	}
	single := must(content.NewSpellCastingTimesSingle(act))
	if single.Time() != act || !content.SpellCastingTimesEqual(single, must(content.NewSpellCastingTimesSingle(act))) ||
		content.SpellCastingTimesEqual(single, m) || content.SpellCastingTimesEqual(m, single) || !content.SpellCastingTimesEqual(m, must(content.NewSpellCastingTimesModes([]content.CastingTimeMode{a, b}))) ||
		content.SpellCastingTimesEqual(m, must(content.NewSpellCastingTimesModes([]content.CastingTimeMode{b, a}))) || content.SpellCastingTimesEqual(single, must(content.NewSpellCastingTimesSingle(hrs))) {
		t.Error("equality of casting times")
	}
}

// Requirement: content-spells.md FR-036, FR-037, FR-050, FR-051, FR-114 - the six
// ranges and five durations have distinct labels; a distance and a timed
// duration reject an amount below 1 and an invalid unit or limit.
// Path: unhappy (boundary).
func TestSpellRangeAndDuration(t *testing.T) {
	t.Parallel()
	dist := must(content.NewSpellRangeDistance(5, content.DistanceFeet))
	for want, r := range map[string]content.SpellRange{
		"self": content.NewSpellRangeSelf(), "touch": content.NewSpellRangeTouch(), "distance": dist,
		"sight": content.NewSpellRangeSight(), "unlimited": content.NewSpellRangeUnlimited(), "special": content.NewSpellRangeSpecial(),
	} {
		if got := rangeLabel(r); got != want {
			t.Errorf("range label = %q, want %q", got, want)
		}
	}
	if dist.Amount() != 5 || dist.Unit() != content.DistanceFeet {
		t.Error("distance accessors")
	}
	for _, n := range []int{-1, 0} {
		_, err := content.NewSpellRangeDistance(n, content.DistanceFeet)
		wantBlockError(t, err, "SpellRangeDistance", content.BlockOutOfRange)
	}
	_, err := content.NewSpellRangeDistance(1, 0)
	wantBlockError(t, err, "SpellRangeDistance", content.BlockInvalidMember)
	timed := must(content.NewSpellDurationTimed(1, content.DurationUnitDays, content.LimitConcentrationUpTo))
	for want, d := range map[string]content.SpellDuration{
		"instantaneous": content.NewSpellDurationInstantaneous(), "timed": timed, "dispelled": content.NewSpellDurationUntilDispelled(),
		"triggered": content.NewSpellDurationUntilDispelledOrTriggered(), "special": content.NewSpellDurationSpecial(),
	} {
		if got := durationLabel(d); got != want {
			t.Errorf("duration label = %q, want %q", got, want)
		}
	}
	if timed.Amount() != 1 || timed.Unit() != content.DurationUnitDays || timed.Limit() != content.LimitConcentrationUpTo {
		t.Error("timed accessors")
	}
	for _, n := range []int{-1, 0} {
		_, err := content.NewSpellDurationTimed(n, content.DurationUnitRounds, content.LimitFixed)
		wantBlockError(t, err, "SpellDurationTimed", content.BlockOutOfRange)
	}
	for _, c := range []struct {
		u content.DurationUnit
		l content.DurationLimit
	}{{0, content.LimitFixed}, {content.DurationUnitHours, 0}} {
		_, err := content.NewSpellDurationTimed(1, c.u, c.l)
		wantBlockError(t, err, "SpellDurationTimed", content.BlockInvalidMember)
	}
	if content.ValidSpellRangeDistance(0) || !content.ValidSpellRangeDistance(1) || content.ValidSpellDurationAmount(0) || !content.ValidSpellDurationAmount(1) {
		t.Error("amount predicates [FR-111]")
	}
}

// Requirement: content-spells.md FR-041, FR-042, FR-043, FR-049, FR-101, FR-111 -
// a material component keeps its description verbatim, copies its costs, rejects
// a blank description and a cost below 1; SpellComponents has a material only
// when given one.
// Path: unhappy (invalid input).
func TestMaterialAndComponents(t *testing.T) {
	t.Parallel()
	costs := []int{1000, 100}
	m := must(content.NewMaterialComponent(" a gem worth 1,000+ GP ", costs, true))
	costs[0] = 5
	m.CostsGP()[1] = 7
	if m.Description() != " a gem worth 1,000+ GP " || !slices.Equal(m.CostsGP(), []int{1000, 100}) || !m.ConsumesMaterial() {
		t.Errorf("material = %q %v %v, want verbatim and copied [FR-086]", m.Description(), m.CostsGP(), m.ConsumesMaterial())
	}
	empty := must(content.NewMaterialComponent("dust", []int{}, false))
	if empty.CostsGP() != nil || !empty.Equal(must(content.NewMaterialComponent("dust", nil, false))) {
		t.Error("an empty cost list is not stored as none")
	}
	for _, s := range []string{"", " ", "\n", "a\x00"} {
		if _, err := content.NewMaterialComponent(s, nil, false); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("description %q accepted", s)
		}
	}
	for _, n := range []int{-1, 0} {
		_, err := content.NewMaterialComponent("x", []int{1, n}, false)
		wantBlockError(t, err, "MaterialComponent", content.BlockOutOfRange)
	}
	if content.ValidMaterialCostGP(0) || !content.ValidMaterialCostGP(1) {
		t.Error("cost predicate [FR-101]")
	}
	for _, o := range []content.MaterialComponent{
		must(content.NewMaterialComponent(" a gem worth 1,000+ GP ", []int{1000}, true)), must(content.NewMaterialComponent("a gem", []int{1000, 100}, true)),
		must(content.NewMaterialComponent(" a gem worth 1,000+ GP ", []int{1000, 100}, false)),
	} {
		if m.Equal(o) {
			t.Error("different materials are equal")
		}
	}
	vs := content.NewSpellComponents(true, false)
	if _, has := vs.Material(); has || !vs.Verbal() || vs.Somatic() {
		t.Error("components without a material")
	}
	with, err := vs.WithMaterial(m)
	got, has := with.Material()
	if err != nil || !has || !got.Equal(m) {
		t.Fatalf("WithMaterial = (%v, %v)", with, err)
	}
	if _, has := vs.Material(); has {
		t.Error("WithMaterial changed its receiver")
	}
	if _, err := vs.WithMaterial(content.MaterialComponent{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero material accepted: %v", err)
	}
	if vs.Equal(with) || !with.Equal(with) || vs.Equal(content.NewSpellComponents(true, true)) || !vs.Equal(content.NewSpellComponents(true, false)) || vs.Equal(content.NewSpellComponents(false, false)) {
		t.Error("components equality")
	}
}

// baseSpellFields are the members of the smallest valid spell.
func baseSpellFields() content.SpellFields {
	return content.SpellFields{
		Level: 1, School: content.SchoolEvocation, Classes: []content.ClassID{"wizard"},
		CastingTimes: must(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction())),
		Range:        content.NewSpellRangeSelf(), Components: content.NewSpellComponents(true, true),
		Duration:    content.NewSpellDurationInstantaneous(),
		Description: []content.TextBlock{content.MustTextParagraph("Text.")},
	}
}

func baseSpell(edit func(*content.SpellFields)) (content.Spell, error) {
	f := baseSpellFields()
	if edit != nil {
		edit(&f)
	}
	return content.NewSpell("fixture_spell", "Fixture Spell", content.SourceCitation{}, f)
}

// Requirement: content-spells.md FR-007, FR-017, FR-018, FR-058, FR-086, FR-098,
// FR-102, FR-122 - NewSpell accepts the minimal spell (no citation, no class,
// no higher-level text, text-only effects by default) and rejects each broken
// member with the reason of its own check.
// Path: per case.
func TestNewSpell(t *testing.T) {
	t.Parallel()
	s, err := baseSpell(nil)
	if err != nil || s.Key() != content.NewKey(content.KindSpell, content.MustID("fixture_spell")) || s.ID() != "fixture_spell" ||
		s.Name() != "Fixture Spell" || s.Source() != (content.SourceCitation{}) || s.Level() != 1 || s.School() != content.SchoolEvocation ||
		s.Ritual() || s.HigherLevel() != nil || s.CantripUpgrade() != nil || s.RequiresConcentration() {
		t.Fatalf("NewSpell(minimal) = (%+v, %v) [happy]", s, err)
	}
	if _, ok := s.Effects().(content.TextOnlyEffects); !ok {
		t.Errorf("Effects() = %T, want TextOnlyEffects by default [FR-122]", s.Effects())
	}
	if m, err := baseSpell(func(f *content.SpellFields) { f.Classes = nil }); err != nil || m.Classes() != nil {
		t.Errorf("a spell with no class: (%v, %v) [FR-018]", m.Classes(), err)
	}
	if _, err := content.NewSpell("Bad-Id", "N", content.SourceCitation{}, baseSpellFields()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a bad id accepted: %v", err)
	}
	for _, name := range []string{"", " N", "N ", "a\x00"} {
		_, err := content.NewSpell("a", name, content.SourceCitation{}, baseSpellFields())
		wantBlockError(t, err, "Spell", content.BlockInvalidName)
		if content.ValidSpellName(name) {
			t.Errorf("ValidSpellName(%q) = true", name)
		}
	}
	para := content.MustTextParagraph("x")
	cases := []struct {
		name string
		edit func(*content.SpellFields)
		want content.BlockProblem
	}{
		{"level 10", func(f *content.SpellFields) { f.Level = 10 }, content.BlockOutOfRange},
		{"level -1", func(f *content.SpellFields) { f.Level = -1 }, content.BlockOutOfRange},
		{"school zero", func(f *content.SpellFields) { f.School = 0 }, content.BlockInvalidMember},
		{"class not an id", func(f *content.SpellFields) { f.Classes = []content.ClassID{"Wizard"} }, content.BlockInvalidMember},
		{"class repeated", func(f *content.SpellFields) { f.Classes = []content.ClassID{"a", "b", "a"} }, content.BlockDuplicate},
		{"casting nil", func(f *content.SpellFields) { f.CastingTimes = nil }, content.BlockMissing},
		{"casting pointer", func(f *content.SpellFields) {
			c := must(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction()))
			f.CastingTimes = &c
		}, content.BlockMissing},
		{"range nil", func(f *content.SpellFields) { f.Range = nil }, content.BlockMissing},
		{"range pointer", func(f *content.SpellFields) { r := content.NewSpellRangeSelf(); f.Range = &r }, content.BlockMissing},
		{"range distance zero", func(f *content.SpellFields) { f.Range = content.SpellRangeDistance{} }, content.BlockOutOfRange},
		{"duration nil", func(f *content.SpellFields) { f.Duration = nil }, content.BlockMissing},
		{"duration pointer", func(f *content.SpellFields) { d := content.NewSpellDurationSpecial(); f.Duration = &d }, content.BlockMissing},
		{"duration timed zero", func(f *content.SpellFields) { f.Duration = content.SpellDurationTimed{} }, content.BlockOutOfRange},
		{"description nil", func(f *content.SpellFields) { f.Description = nil }, content.BlockEmpty},
		{"description empty", func(f *content.SpellFields) { f.Description = []content.TextBlock{} }, content.BlockEmpty},
		{"description zero block", func(f *content.SpellFields) { f.Description = []content.TextBlock{content.TextParagraph{}} }, content.BlockInvalidMember},
		{"higher level empty", func(f *content.SpellFields) { f.HigherLevel = []content.TextBlock{} }, content.BlockEmpty},
		{"higher level bad block", func(f *content.SpellFields) { f.HigherLevel = []content.TextBlock{&para} }, content.BlockInvalidMember},
		{"cantrip upgrade empty", func(f *content.SpellFields) { f.CantripUpgrade = []content.TextBlock{} }, content.BlockEmpty},
		{"effects pointer", func(f *content.SpellFields) { e := content.TextOnlyEffects{}; f.Effects = &e }, content.BlockInvalidMember},
		{"effects zero modelled", func(f *content.SpellFields) { f.Effects = content.ModelledEffects{} }, content.BlockInvalidMember},
	}
	for _, c := range cases {
		_, err := baseSpell(c.edit)
		var be *content.BlockError
		if !errors.As(err, &be) || be.Problem != c.want || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want a *BlockError with problem %v [unhappy (invalid input)]", c.name, err, c.want)
		}
	}
}

// Requirement: spell-effect-data.md FR-009 and content-spells.md FR-122 - the
// Spell constructor calls the level check of the model: a per-slot scaling on a
// cantrip, a table on a level 1 spell and an AboveLevel below the spell's level
// are rejected; a model that agrees is kept.
// Path: unhappy (rule interaction).
func TestNewSpellChecksEffectsAgainstLevel(t *testing.T) {
	t.Parallel()
	perSlotModel := fireball(content.CoverageComplete) // per slot above level 3
	tableModel := fireBolt(content.CoverageComplete)   // a character level table
	for _, c := range []struct {
		name  string
		level content.SpellLevel
		fx    content.SpellEffectModel
		ok    bool
	}{
		{"per slot, AboveLevel equal to the level", 3, perSlotModel, true},
		{"per slot, AboveLevel above the level", 2, perSlotModel, true},
		{"per slot, AboveLevel below the level", 4, perSlotModel, false},
		{"per slot on a cantrip", 0, perSlotModel, false},
		{"table on a level 1 spell", 1, tableModel, false},
		{"table on a cantrip", 0, tableModel, true},
	} {
		s, err := baseSpell(func(f *content.SpellFields) { f.Level, f.Effects = c.level, c.fx })
		switch {
		case c.ok && err != nil:
			t.Errorf("%s: error = %v, want accepted", c.name, err)
		case c.ok && !s.Effects().Equal(c.fx):
			t.Errorf("%s: Effects() differs from the given model", c.name)
		case !c.ok:
			wantBlockError(t, err, "ModelledEffects", content.BlockConstraint)
		}
	}
}

// Requirement: content-spells.md FR-017, FR-086, FR-087 - a Spell copies every
// list on construction and on read, two spells whose classes differ only in
// listing order are equal, and Equal covers each field including effects.
// Path: unhappy (immutability).
func TestSpellImmutabilityAndEquality(t *testing.T) {
	t.Parallel()
	classes := []content.ClassID{"wizard", "bard"}
	desc := []content.TextBlock{content.MustTextParagraph("One.")}
	higher := []content.TextBlock{content.MustTextParagraph("Two.")}
	s, err := baseSpell(func(f *content.SpellFields) { f.Classes, f.Description, f.HigherLevel = classes, desc, higher })
	if err != nil {
		t.Fatal(err)
	}
	classes[0], desc[0], higher[0] = "x", content.MustTextParagraph("changed"), content.MustTextParagraph("changed")
	s.Classes()[0] = "x"
	s.Description()[0] = content.MustTextParagraph("changed")
	s.HigherLevel()[0] = content.MustTextParagraph("changed")
	if got := s.Classes(); !slices.Equal(got, []content.ClassID{"bard", "wizard"}) {
		t.Errorf("Classes() = %v, want the sorted original [FR-017]", got)
	}
	if got := s.Description(); len(got) != 1 || got[0] != content.TextBlock(content.MustTextParagraph("One.")) {
		t.Errorf("Description() = %v, want the original", got)
	}
	if got := s.HigherLevel(); len(got) != 1 || got[0] != content.TextBlock(content.MustTextParagraph("Two.")) {
		t.Errorf("HigherLevel() = %v, want the original", got)
	}
	reordered, _ := baseSpell(func(f *content.SpellFields) {
		f.Classes, f.HigherLevel = []content.ClassID{"bard", "wizard"}, []content.TextBlock{content.MustTextParagraph("Two.")}
	})
	if !s.Equal(s) || s.Equal(reordered) { // description differs: One. versus Text.
		t.Error("Equal on the same spell or on a different description")
	}
	same, _ := baseSpell(func(f *content.SpellFields) {
		f.Classes, f.Description, f.HigherLevel = []content.ClassID{"bard", "wizard"}, []content.TextBlock{content.MustTextParagraph("One.")}, []content.TextBlock{content.MustTextParagraph("Two.")}
	})
	if !s.Equal(same) || same.Equal(content.Spell{}) || (content.Spell{}).Equal(same) || !(content.Spell{}).Equal(content.Spell{}) {
		t.Error("spells differing only in the listing order of classes are not equal [FR-017]")
	}
	for name, edit := range map[string]func(*content.SpellFields){
		"level":   func(f *content.SpellFields) { f.Level = 2 },
		"school":  func(f *content.SpellFields) { f.School = content.SchoolIllusion },
		"classes": func(f *content.SpellFields) { f.Classes = nil },
		"casting times": func(f *content.SpellFields) {
			f.CastingTimes = must(content.NewSpellCastingTimesSingle(must(content.NewCastingTimeHours(1))))
		},
		"ritual":          func(f *content.SpellFields) { f.Ritual = true },
		"range":           func(f *content.SpellFields) { f.Range = content.NewSpellRangeTouch() },
		"components":      func(f *content.SpellFields) { f.Components = content.NewSpellComponents(true, false) },
		"duration":        func(f *content.SpellFields) { f.Duration = content.NewSpellDurationSpecial() },
		"description":     func(f *content.SpellFields) { f.Description = []content.TextBlock{content.MustTextParagraph("Other.")} },
		"cantrip upgrade": func(f *content.SpellFields) { f.CantripUpgrade = []content.TextBlock{content.MustTextParagraph("Up.")} },
		"effects":         func(f *content.SpellFields) { f.Effects = magicMissile() },
	} {
		o, err := baseSpell(edit)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if b, _ := baseSpell(nil); b.Equal(o) || o.Equal(b) {
			t.Errorf("spells differing in %s are equal [FR-087]", name)
		}
	}
	a, _ := baseSpell(func(f *content.SpellFields) { f.Effects = magicMissile() })
	b, _ := baseSpell(func(f *content.SpellFields) { f.Effects = magicMissile() })
	if !a.Equal(b) {
		t.Error("equal effects, unequal spells")
	}
}

// Requirement: content-spells.md FR-057 (SRD 5.2.1, "Duration", p. 106) -
// RequiresConcentration is true only for a timed duration limited by
// Concentration.
// Path: unhappy (boundary).
func TestSpellRequiresConcentration(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		d    content.SpellDuration
		want bool
	}{
		{must(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitConcentrationUpTo)), true},
		{must(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitUpTo)), false},
		{must(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitFixed)), false},
		{content.NewSpellDurationInstantaneous(), false},
	} {
		s, err := baseSpell(func(f *content.SpellFields) { f.Duration = c.d })
		if err != nil || s.RequiresConcentration() != c.want {
			t.Errorf("%v: RequiresConcentration = %v (%v), want %v", c.d, s.RequiresConcentration(), err, c.want)
		}
	}
}

// Requirement: content-spells.md FR-120, FR-121 (SRD 5.2.1, "Class Spell
// Lists", p. 105) and plan item P7 - DeriveClassSpellList groups the ids of the
// spells that name the class by level, in ordinal order, whatever the input
// order, copies on read, leaves its input alone and gives empty groups for a
// class no spell names.
// Path: unhappy (determinism).
func TestDeriveClassSpellList(t *testing.T) {
	t.Parallel()
	mk := func(id string, level content.SpellLevel, classes ...content.ClassID) content.Spell {
		return must(content.NewSpell(content.SpellID(id), strings.ToUpper(id), content.SourceCitation{}, func() content.SpellFields {
			f := baseSpellFields()
			f.Level, f.Classes = level, classes
			return f
		}()))
	}
	spells := []content.Spell{
		mk("zeta", 1, "wizard"), mk("alpha", 1, "wizard", "bard"), mk("mid", 0, "wizard"), mk("only_bard", 9, "bard"),
		mk("beta", 1, "wizard"), mk("no_class", 3),
	}
	snapshot := slices.Clone(spells)
	list := content.DeriveClassSpellList("wizard", spells)
	if list.ClassID() != "wizard" || !slices.Equal(list.IDsAt(1), []content.SpellID{"alpha", "beta", "zeta"}) || !slices.Equal(list.IDsAt(0), []content.SpellID{"mid"}) {
		t.Errorf("lists = %v %v, want ordinal order [FR-121]", list.IDsAt(1), list.IDsAt(0))
	}
	for l := content.SpellLevel(2); l <= 9; l++ {
		if got := list.IDsAt(l); got != nil {
			t.Errorf("level %d = %v, want none", l, got)
		}
	}
	if list.IDsAt(-1) != nil || list.IDsAt(10) != nil {
		t.Error("a level outside 0 to 9 has ids")
	}
	list.IDsAt(1)[0] = "changed"
	if got := list.IDsAt(1)[0]; got != "alpha" {
		t.Errorf("IDsAt returned the stored slice: %v", got)
	}
	shuffled := []content.Spell{spells[4], spells[2], spells[0], spells[1], spells[3], spells[5], spells[0]}
	if !content.DeriveClassSpellList("wizard", shuffled).Equal(list) {
		t.Error("input order or a repeated spell changed the list [FR-121]")
	}
	for i := range spells {
		if !spells[i].Equal(snapshot[i]) {
			t.Fatal("the input was altered")
		}
	}
	bard := content.DeriveClassSpellList("bard", spells)
	if !slices.Equal(bard.IDsAt(9), []content.SpellID{"only_bard"}) || bard.Equal(list) {
		t.Errorf("bard list = %v", bard.IDsAt(9))
	}
	house := content.DeriveClassSpellList("house_class", spells)
	none := content.DeriveClassSpellList("wizard", nil)
	if house.ClassID() != "house_class" || house.IDsAt(1) != nil || none.IDsAt(0) != nil || none.Equal(content.ClassSpellList{}) || !(content.ClassSpellList{}).Equal(content.ClassSpellList{}) {
		t.Error("an unlisted class or a non-caster has ids, or the zero list is not empty [FR-120]")
	}
}

// Requirement: content-spells.md FR-009, FR-010 - a rejected level returns the cantrip level beside the error;
// MustSpellLevel panics with the *SpellLevelError itself; Is matches no nil pointer, no nil receiver and no
// other error type.
// Path: unhappy (boundary).
func TestSpellLevelErrorContract(t *testing.T) {
	t.Parallel()
	l, err := content.NewSpellLevel(10)
	if err == nil || l != 0 {
		t.Errorf("NewSpellLevel(10) = %v, %v, want 0 and an error", l, err)
	}
	func() {
		defer func() {
			le, ok := recover().(*content.SpellLevelError)
			if !ok || le.Value != 10 {
				t.Errorf("MustSpellLevel(10) panic value = %v, want *SpellLevelError{10}", le)
			}
		}()
		content.MustSpellLevel(10)
	}()
	var nilErr *content.SpellLevelError
	if errors.Is(err, nilErr) {
		t.Error("Is matched a nil *SpellLevelError target")
	}
	if nilErr.Is(&content.SpellLevelError{}) {
		t.Error("a nil receiver matched a target")
	}
	if (&content.SpellLevelError{}).Is(srderr.ErrInvalidInput) {
		t.Error("Is matched an error of another type")
	}
}

// Requirement: content-spells.md FR-120, FR-121 - two lists for the same class are equal only when
// every level holds the same ids, whichever level differs.
// Path: unhappy (boundary).
func TestClassSpellListEqualComparesEveryLevel(t *testing.T) {
	t.Parallel()
	mk := func(id string, level content.SpellLevel) content.Spell {
		f := baseSpellFields()
		f.Level, f.Classes = level, []content.ClassID{"wizard"}
		return must(content.NewSpell(content.SpellID(id), strings.ToUpper(id), content.SourceCitation{}, f))
	}
	base := content.DeriveClassSpellList("wizard", []content.Spell{mk("a", 1), mk("b", 9)})
	for _, other := range [][]content.Spell{
		{mk("a", 1)}, {mk("a", 1), mk("c", 9)}, {mk("a", 1), mk("b", 9), mk("d", 5)}, {mk("a", 2), mk("b", 9)},
	} {
		if base.Equal(content.DeriveClassSpellList("wizard", other)) {
			t.Errorf("lists differing from %v compare equal", other)
		}
	}
}

// Requirement: content-spells.md FR-121 - the top level (9) is also sorted and listed once per spell.
// Path: unhappy (boundary).
func TestDeriveClassSpellListSortsTopLevel(t *testing.T) {
	t.Parallel()
	mk := func(id string) content.Spell {
		f := baseSpellFields()
		f.Level, f.Classes = 9, []content.ClassID{"wizard"}
		return must(content.NewSpell(content.SpellID(id), strings.ToUpper(id), content.SourceCitation{}, f))
	}
	got := content.DeriveClassSpellList("wizard", []content.Spell{mk("z"), mk("y"), mk("z")}).IDsAt(9)
	if !slices.Equal(got, []content.SpellID{"y", "z"}) {
		t.Errorf("level 9 = %v, want [y z]", got)
	}
}

// Requirement: content-spells.md FR-012, FR-119 - every member of the school,
// distance unit and duration limit vocabularies prints and parses as its
// own spelling, in declaration order.
// Path: happy.
func TestSpellVocabularySpellings(t *testing.T) {
	t.Parallel()
	for i, name := range []string{"Abjuration", "Conjuration", "Divination", "Enchantment", "Evocation", "Illusion", "Necromancy", "Transmutation"} {
		s := content.SpellSchool(i + 1)
		if got, err := content.ParseSpellSchool(name); err != nil || got != s || s.String() != name || !s.Valid() {
			t.Errorf("school %q: parsed %v (%v), String %q", name, got, err, s.String())
		}
	}
	for i, name := range []string{"feet", "miles"} {
		u := content.DistanceUnit(i + 1)
		if got, err := content.ParseDistanceUnit(name); err != nil || got != u || u.String() != name || !u.Valid() {
			t.Errorf("unit %q: parsed %v (%v), String %q", name, got, err, u.String())
		}
	}
	for i, name := range []string{"fixed", "upTo", "concentrationUpTo"} {
		l := content.DurationLimit(i + 1)
		if got, err := content.ParseDurationLimit(name); err != nil || got != l || l.String() != name || !l.Valid() {
			t.Errorf("limit %q: parsed %v (%v), String %q", name, got, err, l.String())
		}
	}
}

// Requirement: content-spells.md FR-029, FR-030, FR-087 - a zero-valued casting
// time variant is rejected where a casting time is checked, nil casting times
// equal only nil, and spells differing only in name or higher-level text are
// not equal.
// Path: unhappy (invalid input).
func TestSpellCastingTimeZeroVariantsAndEqualityEdges(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]content.CastingTime{
		"minutes": content.CastingTimeMinutes{}, "hours": content.CastingTimeHours{}, "reaction": content.CastingTimeReaction{},
	} {
		if _, err := content.NewSpellCastingTimesSingle(c); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero %s accepted, error = %v", name, err)
		}
	}
	single := must(content.NewSpellCastingTimesSingle(content.NewCastingTimeMagicAction()))
	if !content.SpellCastingTimesEqual(nil, nil) || content.SpellCastingTimesEqual(nil, single) || content.SpellCastingTimesEqual(single, nil) {
		t.Error("nil casting times equal only nil")
	}
	base, _ := baseSpell(func(f *content.SpellFields) { f.HigherLevel = []content.TextBlock{content.MustTextParagraph("More.")} })
	other, _ := baseSpell(func(f *content.SpellFields) { f.HigherLevel = []content.TextBlock{content.MustTextParagraph("Less.")} })
	renamed, err := content.NewSpell("fixture_spell", "Other Name", content.SourceCitation{}, func() content.SpellFields {
		f := baseSpellFields()
		f.HigherLevel = []content.TextBlock{content.MustTextParagraph("More.")}
		return f
	}())
	if err != nil {
		t.Fatal(err)
	}
	if base.Equal(other) || other.Equal(base) || base.Equal(renamed) || renamed.Equal(base) {
		t.Error("spells differing in higher-level text or name are equal")
	}
}
