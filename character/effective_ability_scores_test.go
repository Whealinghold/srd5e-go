package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
)

// scoresWith returns ten in every ability except the overrides.
func scoresWith(t *testing.T, overrides map[ability.Ability]int) ability.Scores {
	t.Helper()
	m := map[ability.Ability]ability.Score{}
	for _, a := range ability.All() {
		v := 10
		if o, ok := overrides[a]; ok {
			v = o
		}
		m[a] = score(t, v)
	}
	return must(ability.NewScores(m))
}

type heldItem struct {
	id      string
	choice  character.EnhancementChoice
	usage   character.ItemUsage
	access  character.MagicalPropertiesAccessLevel
	instKey string
}

func activeScores(t *testing.T, items ...heldItem) character.ActiveItemEffects {
	t.Helper()
	var eq []character.EquippedItem
	for _, it := range items {
		access := it.access
		if access == 0 {
			access = character.AccessFullMagical
		}
		usage := it.usage
		if usage == 0 {
			usage = character.UsageWorn
		}
		eq = append(eq, character.EquippedItem{Instance: inst(it.instKey), Effects: effectsOf(t, it.id, it.choice).Effects(), Context: character.ItemEffectContext{Access: access, Usage: usage}})
	}
	return must(character.ActiveEffectsOf(eq))
}

// Requirement: doc/specs/magic-item-effects.md FR-014 to FR-020 (SRD 5.2.1, "Belt of Giant Strength", p. 213; "Gauntlets of Ogre Power", p. 223;
// "Belt of Dwarvenkind", p. 213; "Hammer of Thunderbolts", p. 224) - every row of table "Score cases": set is max(base, score), sets never add and the
// highest wins, raises apply after sets and never lower, the Hammer boosts only a bestowed Strength from a worn, attuned Belt or Gauntlets, an inactive
// item does nothing, and the source says what decided each score.
// Path: per case.
func TestEffectiveScoresOfTableScoreCases(t *testing.T) {
	t.Parallel()
	hill := character.EnhancementChoice{Giant: character.GiantHill}
	storm := character.EnhancementChoice{Giant: character.GiantStorm}
	gauntlets := heldItem{id: "gauntlets_of_ogre_power", instKey: "g-1"}
	hammer := heldItem{id: "hammer_of_thunderbolts", instKey: "h-1", usage: character.UsageCarried}
	dwarvenkind := heldItem{id: "belt_of_dwarvenkind", instKey: "d-1"}
	amulet := heldItem{id: "amulet_of_health", instKey: "a-1"}
	cases := []struct {
		name       string
		base       map[ability.Ability]int
		items      []heldItem
		a          ability.Ability
		want       int
		kind       character.AbilityScoreSourceKind
		sourceInst string
	}{
		{"gauntlets raise", map[ability.Ability]int{ability.Strength: 10}, []heldItem{gauntlets}, ability.Strength, 19, character.SetByItem, "g-1"},
		{"gauntlets, base equal", map[ability.Ability]int{ability.Strength: 19}, []heldItem{gauntlets}, ability.Strength, 19, character.BaseScore, ""},
		{"gauntlets, base one below", map[ability.Ability]int{ability.Strength: 18}, []heldItem{gauntlets}, ability.Strength, 19, character.SetByItem, "g-1"},
		{"gauntlets, base higher", map[ability.Ability]int{ability.Strength: 20}, []heldItem{gauntlets}, ability.Strength, 20, character.BaseScore, ""},
		{"belt and gauntlets", map[ability.Ability]int{ability.Strength: 10}, []heldItem{{id: "belt_of_giant_strength", choice: hill, instKey: "b-1"}, gauntlets}, ability.Strength, 21, character.SetByItem, "b-1"},
		{"gauntlets then belt, order does not matter", map[ability.Ability]int{ability.Strength: 10}, []heldItem{gauntlets, {id: "belt_of_giant_strength", choice: hill, instKey: "b-1"}}, ability.Strength, 21, character.SetByItem, "b-1"},
		{"storm belt", map[ability.Ability]int{ability.Strength: 8}, []heldItem{{id: "belt_of_giant_strength", choice: storm, instKey: "b-1"}}, ability.Strength, 29, character.SetByItem, "b-1"},
		{"amulet on 18", map[ability.Ability]int{ability.Constitution: 18}, []heldItem{amulet}, ability.Constitution, 19, character.SetByItem, "a-1"},
		{"dwarvenkind 17", map[ability.Ability]int{ability.Constitution: 17}, []heldItem{dwarvenkind}, ability.Constitution, 19, character.RaisedByItem, "d-1"},
		{"dwarvenkind 19", map[ability.Ability]int{ability.Constitution: 19}, []heldItem{dwarvenkind}, ability.Constitution, 20, character.RaisedByItem, "d-1"},
		{"dwarvenkind 20", map[ability.Ability]int{ability.Constitution: 20}, []heldItem{dwarvenkind}, ability.Constitution, 20, character.BaseScore, ""},
		{"dwarvenkind 22", map[ability.Ability]int{ability.Constitution: 22}, []heldItem{dwarvenkind}, ability.Constitution, 22, character.BaseScore, ""},
		{"amulet then dwarvenkind", map[ability.Ability]int{ability.Constitution: 12}, []heldItem{amulet, dwarvenkind}, ability.Constitution, 20, character.RaisedByItem, "d-1"},
		{"dwarvenkind listed before amulet", map[ability.Ability]int{ability.Constitution: 12}, []heldItem{dwarvenkind, amulet}, ability.Constitution, 20, character.RaisedByItem, "d-1"},
		{"might of giants on storm belt", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer, {id: "belt_of_giant_strength", choice: storm, instKey: "b-1"}}, ability.Strength, 30, character.SetByItem, "b-1"},
		{"might of giants on gauntlets", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer, gauntlets}, ability.Strength, 23, character.SetByItem, "g-1"},
		{"hammer, gauntlets without effect", map[ability.Ability]int{ability.Strength: 19}, []heldItem{hammer, gauntlets}, ability.Strength, 19, character.BaseScore, ""},
		{"belt and gauntlets with hammer", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer, {id: "belt_of_giant_strength", choice: hill, instKey: "b-1"}, gauntlets}, ability.Strength, 25, character.SetByItem, "b-1"},
		{"hammer, belt carried only", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer, {id: "belt_of_giant_strength", choice: hill, instKey: "b-1", usage: character.UsageCarried}}, ability.Strength, 10, character.BaseScore, ""},
		{"hammer alone", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer}, ability.Strength, 10, character.BaseScore, ""},
		{"inactive item", map[ability.Ability]int{ability.Strength: 10}, []heldItem{{id: "gauntlets_of_ogre_power", instKey: "g-1", access: character.AccessNoMagic}}, ability.Strength, 10, character.BaseScore, ""},
		{"not attuned item", map[ability.Ability]int{ability.Strength: 10}, []heldItem{{id: "gauntlets_of_ogre_power", instKey: "g-1", access: character.AccessNonmagicalBenefitsOnly}}, ability.Strength, 10, character.BaseScore, ""},
		{"hammer not attuned", map[ability.Ability]int{ability.Strength: 10}, []heldItem{{id: "hammer_of_thunderbolts", instKey: "h-1", usage: character.UsageCarried, access: character.AccessNonmagicalBenefitsOnly}, gauntlets}, ability.Strength, 19, character.SetByItem, "g-1"},
		{"potion takes no boost", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer, {id: "potion_of_giant_strength", choice: storm, instKey: "p-1", usage: character.UsageCarried}}, ability.Strength, 29, character.SetByItem, "p-1"},
		{"potion beats the belt, so the belt bestows nothing", map[ability.Ability]int{ability.Strength: 10}, []heldItem{hammer, {id: "belt_of_giant_strength", choice: hill, instKey: "b-1"}, {id: "potion_of_giant_strength", choice: storm, instKey: "p-1", usage: character.UsageCarried}}, ability.Strength, 29, character.SetByItem, "p-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			base := scoresWith(t, c.base)
			before := base
			eff := must(character.EffectiveScoresOf(base, activeScores(t, c.items...)))
			if got := eff.Scores().Score(c.a).Value(); got != c.want {
				t.Errorf("%s = %d, want %d", c.a, got, c.want)
			}
			if s := eff.Source(c.a); s.Kind() != c.kind || string(s.InstanceID()) != c.sourceInst {
				t.Errorf("source = %v %q, want %v %q", s.Kind(), s.InstanceID(), c.kind, c.sourceInst)
			}
			if base != before {
				t.Error("the input scores changed")
			}
		})
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-017, FR-020 - effects on different abilities are independent, the other abilities keep their base and
// the base source, and an invalid ability has no source.
// Path: unhappy (rule interaction).
func TestEffectiveScoresOfAbilitiesAreIndependent(t *testing.T) {
	t.Parallel()
	base := scoresWith(t, map[ability.Ability]int{ability.Strength: 12, ability.Constitution: 14, ability.Intelligence: 8})
	eff := must(character.EffectiveScoresOf(base, activeScores(t, heldItem{id: "gauntlets_of_ogre_power", instKey: "g-1"}, heldItem{id: "headband_of_intellect", instKey: "i-1"})))
	for a, want := range map[ability.Ability]int{ability.Strength: 19, ability.Dexterity: 10, ability.Constitution: 14, ability.Intelligence: 19, ability.Wisdom: 10, ability.Charisma: 10} {
		if got := eff.Scores().Score(a).Value(); got != want {
			t.Errorf("%s = %d, want %d", a, got, want)
		}
	}
	if eff.Source(ability.Constitution).Kind() != character.BaseScore || eff.Source(ability.Intelligence).Kind() != character.SetByItem {
		t.Error("sources")
	}
	if (eff.Source(0) != character.AbilityScoreSource{}) || (eff.Source(ability.Charisma+1) != character.AbilityScoreSource{}) {
		t.Error("invalid ability has a source")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-020 - a zero base is invalid input.
// Path: unhappy (invalid input).
func TestEffectiveScoresOfZeroBase(t *testing.T) {
	t.Parallel()
	if _, err := character.EffectiveScoresOf(ability.Scores{}, character.ActiveItemEffects{}); err == nil {
		t.Error("zero base accepted")
	} else {
		wantInvalid(t, err, "zero base")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-015 - GiantStrengthType has exactly five values with the scores 21, 23, 25, 27, 29.
// Path: unhappy (boundary).
func TestGiantStrengthType(t *testing.T) {
	t.Parallel()
	want := map[character.GiantStrengthType]struct {
		score int
		name  string
	}{character.GiantHill: {21, "hill"}, character.GiantFrostOrStone: {23, "frost_or_stone"}, character.GiantFire: {25, "fire"},
		character.GiantCloud: {27, "cloud"}, character.GiantStorm: {29, "storm"}, 0: {0, "unknown"}, character.GiantStorm + 1: {0, "unknown"}}
	for g, w := range want {
		if g.StrengthScore() != w.score || g.String() != w.name {
			t.Errorf("%d = %d %s", g, g.StrengthScore(), g.String())
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-019 - the Hammer's boost goes only on a bestowing set whose item is worn: a host-defined bestowing set that
// is merely carried gets none.
// Path: unhappy (state error).
func TestEffectiveScoresOfBoostNeedsAWornItem(t *testing.T) {
	t.Parallel()
	set := must(character.NewSetAbilityScore(ability.Strength, 21, character.WhileOnPerson)).ForBestowing()
	hammer := effectsOf(t, "hammer_of_thunderbolts", character.EnhancementChoice{}).Effects()
	full := func(u character.ItemUsage) character.ItemEffectContext {
		return character.ItemEffectContext{Access: character.AccessFullMagical, Usage: u}
	}
	for usage, want := range map[character.ItemUsage]int{character.UsageWorn: 25, character.UsageCarried: 21} {
		act := must(character.ActiveEffectsOf([]character.EquippedItem{
			{Instance: inst("h-1"), Effects: hammer, Context: full(character.UsageCarried)},
			{Instance: inst("x-1"), Effects: []character.MagicItemEffect{set}, Context: full(usage)}}))
		if got := must(character.EffectiveScoresOf(scoresWith(t, nil), act)).Scores().Score(ability.Strength).Value(); got != want {
			t.Errorf("usage %d = %d, want %d", usage, got, want)
		}
	}
}
