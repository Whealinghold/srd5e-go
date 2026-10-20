package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: monSpec FR-027, FR-028 - a sense range accepts 0 and
// MonsterMaxNumber and rejects one above.
// Path: unhappy (boundary).
func TestSenseEntryBounds(t *testing.T) {
	t.Parallel()
	for _, feet := range []int{0, content.MonsterMaxNumber} {
		if _, err := content.NewSenseEntry(content.SenseDarkvision, feet, ""); err != nil {
			t.Errorf("NewSenseEntry(%d) = %v, want nil [happy]", feet, err)
		}
	}
	if over, ok := aboveMaxNumber(); ok {
		_, err := content.NewSenseEntry(content.SenseDarkvision, over, "")
		wantBlockError(t, err, "SenseEntry", content.BlockOutOfRange)
	}
}

// Requirement: monSpec FR-029 - telepathy accepts 0 and MonsterMaxNumber feet,
// rejects one above, and Equal tells "no telepathy" from "telepathy 0 ft.",
// and compares phrases, range and qualifier.
// Path: per case.
func TestLanguagesEntryTelepathy(t *testing.T) {
	t.Parallel()
	base, err := content.NewLanguagesEntry([]string{"Common"})
	if err != nil {
		t.Fatal(err)
	}
	for _, feet := range []int{0, content.MonsterMaxNumber} {
		if _, err := base.WithTelepathy(feet, ""); err != nil {
			t.Errorf("WithTelepathy(%d) = %v, want nil [happy]", feet, err)
		}
	}
	if over, ok := aboveMaxNumber(); ok {
		_, err = base.WithTelepathy(over, "")
		wantBlockError(t, err, "LanguagesEntry", content.BlockOutOfRange)
	}
	zero, _ := base.WithTelepathy(0, "")
	if base.Equal(zero) || zero.Equal(base) {
		t.Error("no telepathy equals telepathy 0 ft. [unhappy]")
	}
	a, _ := base.WithTelepathy(60, "q")
	if !a.Equal(must(base.WithTelepathy(60, "q"))) {
		t.Error("equal entries differ [happy]")
	}
	other, _ := content.NewLanguagesEntry([]string{"Elvish"})
	for name, o := range map[string]content.LanguagesEntry{
		"phrases":   must(other.WithTelepathy(60, "q")),
		"feet":      must(base.WithTelepathy(30, "q")),
		"qualifier": must(base.WithTelepathy(60, "r")),
	} {
		if a.Equal(o) {
			t.Errorf("entries differing in %s are Equal [unhappy]", name)
		}
	}
}

// Requirement: monSpec FR-044 - a spell group copies its spells on the way in
// and out, and MonsterSpellcasting copies its groups on the way out, so a
// caller's slice never aliases the value.
// Path: per case.
func TestSpellcastingCopiesSlices(t *testing.T) {
	t.Parallel()
	e1 := must(content.NewMonsterSpellEntry("Fire Bolt", "", 0))
	e2 := must(content.NewMonsterSpellEntry("Shield", "", 0))
	in := []content.MonsterSpellEntry{e1}
	g := must(content.NewMonsterSpellGroup(content.SpellFrequencyAtWill{}, in))
	in[0] = e2
	if got := g.Spells(); got[0] != e1 {
		t.Errorf("group aliases its input slice: %v", got)
	}
	g.Spells()[0] = e2
	if got := g.Spells(); got[0] != e1 {
		t.Errorf("Spells() returns the internal slice: %v", got)
	}
	groups := []content.MonsterSpellGroup{g}
	sc := must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{Ability: ability.Intelligence, Groups: groups}))
	g2 := must(content.NewMonsterSpellGroup(content.SpellFrequencyAtWill{}, []content.MonsterSpellEntry{e2}))
	groups[0] = g2
	if !sc.Groups()[0].Equal(g) {
		t.Error("spellcasting aliases its input groups [unhappy]")
	}
	sc.Groups()[0] = g2
	if !sc.Groups()[0].Equal(g) {
		t.Error("Groups() returns the internal slice [unhappy]")
	}
}

// Requirement: monSpec FR-044 - MonsterSpellcasting.Equal distinguishes "attack
// bonus 0 printed" from "none printed".
// Path: per case.
func TestSpellcastingEqualAttackBonusPrinted(t *testing.T) {
	t.Parallel()
	g := must(content.NewMonsterSpellGroup(content.SpellFrequencyAtWill{},
		[]content.MonsterSpellEntry{must(content.NewMonsterSpellEntry("Fire Bolt", "", 0))}))
	with := must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
		Ability: ability.Intelligence, HasAttackBonus: true, Groups: []content.MonsterSpellGroup{g}}))
	without := must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
		Ability: ability.Intelligence, AttackBonus: 7, Groups: []content.MonsterSpellGroup{g}}))
	if with.Equal(without) || without.Equal(with) {
		t.Error("printed attack bonus 0 equals none printed [unhappy]")
	}
	if _, ok := without.AttackBonus(); ok {
		t.Error("unprinted attack bonus reported printed [unhappy]")
	}
}

// Requirement: monSpec FR-020 - an invalid ability has save 0 and is never
// proficient.
// Path: unhappy.
func TestAbilityBlockInvalidAbility(t *testing.T) {
	t.Parallel()
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		saves[a] = 0
	}
	b := must(content.NewAbilityBlock(monScores(10), saves))
	for _, a := range []ability.Ability{0, ability.Ability(99)} {
		if b.Save(a) != 0 || b.ProficientInSave(a) {
			t.Errorf("invalid ability %d: Save=%d proficient=%v, want 0 and false", a, b.Save(a), b.ProficientInSave(a))
		}
	}
}

// aboveMaxNumber returns MonsterMaxNumber + 1 and true, or false where int is
// 32 bits wide and the value does not exist.
func aboveMaxNumber() (int, bool) {
	v := int64(content.MonsterMaxNumber) + 1
	return int(v), int64(int(v)) == v
}
