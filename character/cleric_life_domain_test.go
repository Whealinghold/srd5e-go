package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// clcSubclass builds a Life Domain-shaped subclass with the grants as its "Life Domain Spells" entry (no fact when grants is empty).
func clcSubclass(tb testing.TB, id content.SubclassID, grants ...content.SpellGrant) *content.SubclassDefinition {
	tb.Helper()
	var facts []content.FeatureFact
	if len(grants) > 0 {
		facts = append(facts, must(content.NewAlwaysPreparedSpells(grants)))
	}
	entry := must(content.NewFeatureEntry("life_domain_life_domain_spells", "Life Domain Spells", content.FeatureEntryFields{Level: 3, Text: "Fixture.", Facts: facts}))
	sub := must(content.NewSubclassDefinition(id, "Fixture", content.MustSourceCitation("SRD 5.2.1", "Fixture", 40), []content.FeatureEntry{entry}))
	return &sub
}

func clcGrant(level int, ids ...content.SpellID) content.SpellGrant {
	return must(content.NewSpellGrant(level, ids))
}

// Requirement: doc/specs/class-cleric.md FR-098, FR-099, FR-100, FR-134, table 4.12.1 LS-01 to LS-11 (SRD 5.2.1, "Level 3: Life Domain Spells", p. 40) - the always-prepared spells of a Life Domain Cleric are the union of the grants
// at or below the level, read from the held subclass (the bundled one, or a fixture with other rows); they are empty for no subclass, another subclass, below level 3 and an entry with no fact; NewlyGrantedAt gives only the
// new grants of a level.
// Path: per case (LS-01 to LS-11): happy / unhappy (boundary, state error, missing content).
func TestLifeDomainSpellsGrantedAt(t *testing.T) {
	t.Parallel()
	life := clcLifeDomain(t)
	l3 := []content.SpellID{"aid", "bless", "cure_wounds", "lesser_restoration"}
	l5 := []content.SpellID{"mass_healing_word", "revivify"}
	l7 := []content.SpellID{"aura_of_life", "death_ward"}
	l9 := []content.SpellID{"greater_restoration", "mass_cure_wounds"}
	fixture := clcSubclass(t, "life_domain", clcGrant(3, "fix_a"), clcGrant(5, "fix_b", "fix_c"))
	for _, c := range []struct {
		label string
		held  *content.SubclassDefinition
		level int
		want  []content.SpellID
	}{
		{"LS-01", life, 2, nil}, {"LS-02", life, 3, l3}, {"LS-03", life, 4, l3}, {"LS-04", life, 5, slices.Concat(l3, l5)}, {"LS-05", life, 7, slices.Concat(l3, l5, l7)},
		{"LS-06", life, 9, slices.Concat(l3, l5, l7, l9)}, {"LS-07", life, 20, slices.Concat(l3, l5, l7, l9)}, {"LS-08", nil, 9, nil},
		{"LS-09", clcSubclass(t, "other", clcGrant(3, "fix_a")), 9, nil},
		{"LS-10", fixture, 5, []content.SpellID{"fix_a", "fix_b", "fix_c"}}, {"LS-10 below the second row", fixture, 4, []content.SpellID{"fix_a"}},
		{"LS-11", clcSubclass(t, "life_domain"), 3, nil},
	} {
		got := character.LifeDomainSpells{}.GrantedAt(c.held, c.level)
		if got == nil || !slices.Equal(got, append([]content.SpellID{}, c.want...)) {
			t.Errorf("%s: GrantedAt(%d) = %#v, want %v", c.label, c.level, got, c.want)
		}
	}
	for level, want := range map[int][]content.SpellID{2: nil, 3: l3, 4: nil, 5: l5, 6: nil, 7: l7, 9: l9, 10: nil} {
		if got := (character.LifeDomainSpells{}).NewlyGrantedAt(life, level); !slices.Equal(got, append([]content.SpellID{}, want...)) {
			t.Errorf("NewlyGrantedAt(%d) = %v, want %v", level, got, want)
		}
	}
	if got := (character.LifeDomainSpells{}).NewlyGrantedAt(nil, 5); len(got) != 0 {
		t.Errorf("NewlyGrantedAt without a subclass = %v", got)
	}
}

// Requirement: doc/specs/class-cleric.md FR-101 to FR-104, FR-112, FR-113, FR-117, table 4.12.2 DL-01 to DL-15 (SRD 5.2.1, "Level 3: Disciple of Life", "Level 6: Blessed Healer", p. 40) - a Life Domain Cleric of
// level 3 gives each creature healed by a spell cast with a slot 2 plus the slot's level; from level 6 the Cleric regains 2 plus the slot's level once per cast when it healed a creature other than itself; a cast with no
// slot, below the levels, with no subclass or another subclass gives nothing.
// Path: per case (DL-01 to DL-15): happy / unhappy (boundary, rule interaction, state error).
func TestResolveClericHealingLifeDomain(t *testing.T) {
	t.Parallel()
	other := func(keys ...string) []character.HealedCreature {
		var out []character.HealedCreature
		for _, k := range keys {
			out = append(out, character.HealedCreature{Key: k})
		}
		return out
	}
	self := character.HealedCreature{Key: "cleric", IsCleric: true}
	for _, c := range []struct {
		label    string
		level    int
		subclass content.SubclassID
		slot     int
		healed   []character.HealedCreature
		disciple int // bonus to each healed creature, 0 for none
		healer   int
	}{
		{"DL-01", 3, "life_domain", 1, other("a"), 3, 0}, {"DL-02", 6, "life_domain", 1, other("a"), 3, 3}, {"DL-03", 6, "life_domain", 3, other("a"), 5, 5},
		{"DL-04", 17, "life_domain", 9, other("a"), 11, 11}, {"DL-05", 6, "life_domain", 3, other("a", "b", "c"), 5, 5},
		{"DL-06", 6, "life_domain", 2, []character.HealedCreature{self}, 4, 0}, {"DL-07", 6, "life_domain", 2, append(other("a"), self), 4, 4},
		{"DL-08 to DL-10 no slot", 6, "life_domain", 0, other("a"), 0, 0}, {"DL-11", 2, "life_domain", 1, other("a"), 0, 0}, {"DL-12", 5, "life_domain", 1, other("a"), 3, 0},
		{"DL-13", 3, "", 1, other("a"), 0, 0}, {"DL-14", 6, "other", 3, other("a"), 0, 0}, {"DL-15", 6, "", 1, other("a"), 0, 0},
	} {
		t.Run(c.label, func(t *testing.T) {
			t.Parallel()
			res, err := character.ResolveClericHealing(clcCleric(t, c.level), c.subclass, character.HealingCast{SlotLevel: c.slot, Healed: c.healed}, nil)
			if err != nil {
				t.Fatal(err)
			}
			r := res.(character.ResolveClericHealingResolved)
			bonuses := r.DiscipleOfLife()
			if c.disciple == 0 && len(bonuses) != 0 || c.disciple != 0 && len(bonuses) != len(c.healed) || r.BlessedHealer() != c.healer {
				t.Fatalf("Disciple of Life %v, Blessed Healer %d; want %d each and %d", bonuses, r.BlessedHealer(), c.disciple, c.healer)
			}
			for i, b := range bonuses {
				if b.HitPoints != c.disciple || b.Key != c.healed[i].Key {
					t.Errorf("bonus %d = %+v, want %d to %q", i, b, c.disciple, c.healed[i].Key)
				}
			}
			if _, has := r.Dice(); has {
				t.Error("dice reported although none were supplied")
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-114, FR-115, FR-116, FR-117, table 4.12.4 SH-01 to SH-12 (SRD 5.2.1, "Level 17: Supreme Healing", p. 40) - the healing dice of a Life Domain Cleric of level 17 are not rolled
// (the highest number on each die, 2d6 gives 12); at level 16, with no subclass or another subclass they are rolled; dice with fewer than 1 die or 2 sides, a slot level outside 0 to 9 and a zero Character are invalid input.
// Path: per case (SH rows): happy / unhappy (boundary, state error, invalid input).
func TestResolveClericHealingSupremeHealing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label    string
		level    int
		subclass content.SubclassID
		dice     character.HealingDice
		maximum  bool
		total    int
	}{
		{"SH-01", 17, "life_domain", character.HealingDice{Count: 2, Sides: 6}, true, 12}, {"SH-02", 17, "life_domain", character.HealingDice{Count: 1, Sides: 4}, true, 4},
		{"SH-03", 16, "life_domain", character.HealingDice{Count: 2, Sides: 6}, false, 0}, {"SH-06", 20, "life_domain", character.HealingDice{Count: 4, Sides: 8}, true, 32},
		{"SH-09", 17, "", character.HealingDice{Count: 2, Sides: 6}, false, 0}, {"SH-10", 17, "other", character.HealingDice{Count: 3, Sides: 8}, false, 0},
	} {
		d := c.dice
		res, err := character.ResolveClericHealing(clcCleric(t, c.level), c.subclass, character.HealingCast{}, &d)
		got, has := res.(character.ResolveClericHealingResolved).Dice()
		if err != nil || !has || got.Maximised != c.maximum || got.Total != c.total || got.Count != c.dice.Count || got.Sides != c.dice.Sides {
			t.Errorf("%s: = %+v, %v", c.label, got, err)
		}
	}
	for name, d := range map[string]character.HealingDice{"SH-07": {Count: 0, Sides: 6}, "SH-08": {Count: 1, Sides: 1}} {
		_, err := character.ResolveClericHealing(clcCleric(t, 17), "life_domain", character.HealingCast{}, &d)
		wantInvalid(t, err, name)
	}
	for _, slot := range []int{-1, 10} {
		_, err := character.ResolveClericHealing(clcCleric(t, 17), "life_domain", character.HealingCast{SlotLevel: slot}, nil)
		wantInvalid(t, err, "slot level "+strconv.Itoa(slot))
	}
	_, err := character.ResolveClericHealing(character.Character{}, "life_domain", character.HealingCast{}, nil)
	wantInvalid(t, err, "zero character")
}

// Requirement: doc/specs/class-cleric.md FR-117, FR-122 - the levels are Cleric levels: a Fighter 5 / Cleric 3 gets Disciple of Life but not Blessed Healer; a character with no Cleric levels is
// ResolveClericHealingNotACleric.
// Path: per case (unhappy: rule interaction, state error).
func TestResolveClericHealingMulticlassAndNotACleric(t *testing.T) {
	t.Parallel()
	res, err := character.ResolveClericHealing(clcChar(t, cl("fighter", 5), cl("cleric", 3)), "life_domain", character.HealingCast{SlotLevel: 2, Healed: []character.HealedCreature{{Key: "a"}}}, nil)
	r, ok := res.(character.ResolveClericHealingResolved)
	if err != nil || !ok || len(r.DiscipleOfLife()) != 1 || r.BlessedHealer() != 0 {
		t.Errorf("Fighter 5 / Cleric 3 = %#v, %v", res, err)
	}
	res, err = character.ResolveClericHealing(fgtFighter(t, 17), "life_domain", character.HealingCast{SlotLevel: 2, Healed: []character.HealedCreature{{Key: "a"}}}, nil)
	if _, ok := res.(character.ResolveClericHealingNotACleric); err != nil || !ok {
		t.Errorf("Fighter = %#v, %v", res, err)
	}
	r.DiscipleOfLife()[0].HitPoints = 99
	if r.DiscipleOfLife()[0].HitPoints == 99 {
		t.Error("DiscipleOfLife returned a shared slice")
	}
}
