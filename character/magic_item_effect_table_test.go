package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func effectsOf(t *testing.T, id string, choice character.EnhancementChoice) character.MagicItemEffectsFound {
	t.Helper()
	f, ok := (character.MagicItemEffectTable{}).Lookup(mid(id), choice).(character.MagicItemEffectsFound)
	if !ok {
		t.Fatalf("%s: not modelled", id)
	}
	return f
}

// activeOf puts one item with its effects through ActiveEffectsOf in ctx.
func activeOf(t *testing.T, effects []character.MagicItemEffect, ctx character.ItemEffectContext, instance string) character.ActiveItemEffects {
	t.Helper()
	return must(character.ActiveEffectsOf([]character.EquippedItem{{Instance: inst(instance), Effects: effects, Context: ctx}}))
}

func fullWorn() character.ItemEffectContext {
	return character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageWorn}
}

func bonusValues(bs []character.MagicItemBonus) []int {
	out := []int{}
	for _, b := range bs {
		out = append(out, b.Modifier().Value())
	}
	return out
}

func eqInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Requirement: doc/specs/magic-item-effects.md FR-003, FR-004, FR-005, FR-007, FR-008, FR-013 (SRD 5.2.1, as cited per row, pp. 209-253) - every row of
// table "Fixed bonus effects" and the +N items yield exactly the listed effects under their condition, each with its Attunement fact.
// Path: per case.
func TestMagicItemEffectTableBonusRows(t *testing.T) {
	t.Parallel()
	worn := fullWorn()
	held := character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageHeld}
	carried := character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageCarried}
	bare := fullWorn()
	type want struct{ ac, saves, spell, hit []int }
	cases := []struct {
		id         string
		rarity     content.ItemRarity
		attune     bool
		ctx        character.ItemEffectContext
		want       want
		halfCover  bool
		wrongState character.ItemEffectContext
	}{
		{"ring_of_protection", 0, true, worn, want{ac: []int{1}, saves: []int{1}}, false, held},
		{"cloak_of_protection", 0, true, worn, want{ac: []int{1}, saves: []int{1}}, false, held},
		{"bracers_of_defense", 0, true, bare, want{ac: []int{2}}, false, character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageWorn, WearsArmor: true}},
		{"scarab_of_protection", 0, true, carried, want{ac: []int{1}}, false, character.ItemEffectContext{Access: character.AccessFullMagical}},
		{"robe_of_stars", 0, true, worn, want{saves: []int{1}}, false, held},
		{"dwarven_plate", 0, false, worn, want{ac: []int{2}}, false, held},
		{"elven_chain", 0, false, worn, want{ac: []int{1}}, false, held},
		{"glamoured_studded_leather", 0, false, worn, want{ac: []int{1}}, false, held},
		{"staff_of_power", 0, true, held, want{ac: []int{2}, saves: []int{2}, spell: []int{2}, hit: []int{2}}, false, worn},
		{"staff_of_the_magi", 0, true, held, want{spell: []int{2}, hit: []int{2}}, false, worn},
		{"staff_of_the_woodlands", 0, true, held, want{spell: []int{2}, hit: []int{2}}, false, worn},
		{"armor_plus", content.RarityVeryRare, false, worn, want{ac: []int{2}}, false, held},
		{"shield_plus", content.RarityUncommon, false, held, want{ac: []int{1}}, false, worn},
		{"weapon_plus", content.RarityRare, false, held, want{hit: []int{2}}, false, carried},
		{"ammunition_plus", content.RarityVeryRare, false, carried, want{hit: []int{3}}, false, character.ItemEffectContext{Access: character.AccessFullMagical}},
		{"wand_of_the_war_mage_plus", content.RarityRare, true, held, want{spell: []int{2}}, true, carried},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			t.Parallel()
			f := effectsOf(t, c.id, character.EnhancementChoice{Rarity: c.rarity})
			if f.RequiresAttunement() != c.attune {
				t.Errorf("attunement = %v", f.RequiresAttunement())
			}
			got := activeOf(t, f.Effects(), c.ctx, "i-1")
			if !eqInts(bonusValues(got.ArmorClass()), c.want.ac) || !eqInts(bonusValues(got.SavingThrows()), c.want.saves) ||
				!eqInts(bonusValues(got.SpellAttackRolls()), c.want.spell) || !eqInts(bonusValues(got.AttackAndDamage()), c.want.hit) || got.IgnoresHalfCoverOnSpellAttacks() != c.halfCover {
				t.Errorf("active = ac %v saves %v spell %v hit %v cover %v", bonusValues(got.ArmorClass()), bonusValues(got.SavingThrows()),
					bonusValues(got.SpellAttackRolls()), bonusValues(got.AttackAndDamage()), got.IgnoresHalfCoverOnSpellAttacks())
			}
			off := activeOf(t, f.Effects(), c.wrongState, "i-1")
			if len(off.ArmorClass())+len(off.SavingThrows())+len(off.SpellAttackRolls())+len(off.AttackAndDamage()) != 0 || off.IgnoresHalfCoverOnSpellAttacks() {
				t.Errorf("wrong state still gives effects: %+v", off)
			}
		})
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-007 - the Wand of the War Mage yields a spell attack roll bonus and the Half Cover flag, and no save DC
// or other bonus.
// Path: unhappy (rule interaction).
func TestWarMageWandHasNoSaveDCBonus(t *testing.T) {
	t.Parallel()
	f := effectsOf(t, "wand_of_the_war_mage_plus", character.EnhancementChoice{Rarity: content.RarityVeryRare})
	effects := f.Effects()
	if len(effects) != 2 {
		t.Fatalf("effects = %+v", effects)
	}
	if b, ok := effects[0].(character.SpellAttackRollBonus); !ok || b.Modifier().Value() != 3 || b.Condition() != character.WhileHeld {
		t.Errorf("first = %+v", effects[0])
	}
	if h, ok := effects[1].(character.IgnoresHalfCoverOnSpellAttacks); !ok || h.Condition() != character.WhileHeld {
		t.Errorf("second = %+v", effects[1])
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-004 - a +N Shield is its own AC term held, separate from the Shield's own bonus; FR-003 a +N armor is a
// bonus worn, not a base.
// Path: unhappy (rule interaction).
func TestPlusArmorAndShieldConditions(t *testing.T) {
	t.Parallel()
	a := effectsOf(t, "armor_plus", character.EnhancementChoice{Rarity: content.RarityRare}).Effects()
	s := effectsOf(t, "shield_plus", character.EnhancementChoice{Rarity: content.RarityRare}).Effects()
	if len(a) != 1 || len(s) != 1 || a[0].(character.ArmorClassBonus).Condition() != character.WhileWorn || s[0].(character.ArmorClassBonus).Condition() != character.WhileHeld ||
		a[0].(character.ArmorClassBonus).Modifier().Value() != 1 || s[0].(character.ArmorClassBonus).Modifier().Value() != 2 {
		t.Errorf("armor %+v shield %+v", a, s)
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-013 - an id the table lacks, and a +N or giant item whose choice does not resolve, is
// MagicItemEffectsNotModelled carrying the id; the returned effects are a copy.
// Path: unhappy (missing content).
func TestMagicItemEffectTableNotModelled(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		id     string
		choice character.EnhancementChoice
	}{
		{"flame_tongue", character.EnhancementChoice{}},
		{"armor_plus", character.EnhancementChoice{}},
		{"armor_plus", character.EnhancementChoice{Rarity: content.RarityUncommon}},
		{"shield_plus", character.EnhancementChoice{Rarity: content.RarityLegendary}},
		{"weapon_plus", character.EnhancementChoice{Rarity: content.RarityCommon}},
		{"ammunition_plus", character.EnhancementChoice{Rarity: content.RarityArtifact}},
		{"wand_of_the_war_mage_plus", character.EnhancementChoice{}},
		{"belt_of_giant_strength", character.EnhancementChoice{}},
		{"potion_of_giant_strength", character.EnhancementChoice{Giant: character.GiantStorm + 1}},
	} {
		nm, ok := (character.MagicItemEffectTable{}).Lookup(mid(c.id), c.choice).(character.MagicItemEffectsNotModelled)
		if !ok || nm.ItemID() != mid(c.id) {
			t.Errorf("%s %+v = %+v", c.id, c.choice, nm)
		}
	}
	f := effectsOf(t, "ring_of_protection", character.EnhancementChoice{})
	f.Effects()[0] = nil
	if f.Effects()[0] == nil {
		t.Error("Effects returned the internal slice")
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-015, FR-021 (SRD 5.2.1, "Belt of Giant Strength", p. 213; "Potion of Giant Strength", p. 235;
// "Amulet of Health", p. 209; "Gauntlets of Ogre Power", p. 223; "Headband of Intellect", p. 225) - the set-score rows hold exactly their scores, the
// giant type picks the score, and a potion carries 60 minutes.
// Path: per case.
func TestMagicItemEffectTableSetScores(t *testing.T) {
	t.Parallel()
	type row struct {
		id        string
		giant     character.GiantStrengthType
		a         ability.Ability
		score     int
		duration  bool
		attune    bool
		bestowing bool
	}
	rows := []row{
		{"amulet_of_health", 0, ability.Constitution, 19, false, true, false},
		{"gauntlets_of_ogre_power", 0, ability.Strength, 19, false, true, true},
		{"headband_of_intellect", 0, ability.Intelligence, 19, false, true, false},
	}
	giants := []struct {
		g     character.GiantStrengthType
		score int
	}{{character.GiantHill, 21}, {character.GiantFrostOrStone, 23}, {character.GiantFire, 25}, {character.GiantCloud, 27}, {character.GiantStorm, 29}}
	for _, g := range giants {
		rows = append(rows, row{"belt_of_giant_strength", g.g, ability.Strength, g.score, false, true, true}, row{"potion_of_giant_strength", g.g, ability.Strength, g.score, true, false, false})
	}
	for _, r := range rows {
		f := effectsOf(t, r.id, character.EnhancementChoice{Giant: r.giant})
		set, ok := f.Effects()[0].(character.SetAbilityScore)
		if len(f.Effects()) != 1 || !ok || set.Ability() != r.a || set.Score().Value() != r.score || f.RequiresAttunement() != r.attune {
			t.Errorf("%s %v = %+v", r.id, r.giant, f.Effects())
			continue
		}
		if (set.Duration() == character.GiantPotionDuration) != r.duration || (set.Duration() != 0) != r.duration {
			t.Errorf("%s duration = %v", r.id, set.Duration())
		}
		// bestowing is observable through the Hammer: it adds only to a bestowing set.
		hammer := effectsOf(t, "hammer_of_thunderbolts", character.EnhancementChoice{}).Effects()
		base := scoresWith(t, map[ability.Ability]int{})
		act := must(character.ActiveEffectsOf([]character.EquippedItem{
			{Instance: inst("h-1"), Effects: hammer, Context: character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageCarried}},
			{Instance: inst("x-1"), Effects: f.Effects(), Context: character.ItemEffectContext{Access: character.AccessFullMagical, Usage: character.UsageWorn}},
		}))
		eff := must(character.EffectiveScoresOf(base, act))
		if r.a == ability.Strength {
			want := r.score
			if r.bestowing {
				want = min(r.score+4, 30)
			}
			if got := eff.Scores().Score(ability.Strength).Value(); got != want {
				t.Errorf("%s %v with hammer = %d, want %d", r.id, r.giant, got, want)
			}
		}
	}
}

// Requirement: doc/specs/magic-item-effects.md FR-018, FR-019 (SRD 5.2.1, "Belt of Dwarvenkind", p. 213; "Hammer of Thunderbolts", p. 224) - the Belt of
// Dwarvenkind raises Constitution by 2 to 20 and the Hammer boosts a bestowed Strength by 4 to 30; both need Attunement.
// Path: happy.
func TestMagicItemEffectTableRaiseAndBoost(t *testing.T) {
	t.Parallel()
	belt := effectsOf(t, "belt_of_dwarvenkind", character.EnhancementChoice{})
	r, ok := belt.Effects()[0].(character.RaiseAbilityScore)
	if !ok || !belt.RequiresAttunement() || r.Ability() != ability.Constitution || r.Amount() != 2 || r.Cap() != 20 || r.Condition() != character.WhileWorn {
		t.Errorf("belt = %+v", belt.Effects())
	}
	hammer := effectsOf(t, "hammer_of_thunderbolts", character.EnhancementChoice{})
	b, ok := hammer.Effects()[0].(character.BoostBestowedScore)
	if !ok || !hammer.RequiresAttunement() || b.Amount() != 4 || b.Cap() != 30 {
		t.Errorf("hammer = %+v", hammer.Effects())
	}
}
