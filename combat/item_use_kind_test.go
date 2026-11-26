package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

const itemUseSpec = "doc/specs/item-use-effects.md "

func dexModifier(score int) ability.Modifier { return ability.MustScore(score).Modifier() }

func proficiency(level int) character.ProficiencyBonus {
	return character.ProficiencyBonusFor(character.MustLevel(level))
}

// Requirement: per case
// Path: per case.
func TestItemSaveDCOf(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		score                   int
		pb                      character.ProficiencyBonus
		want                    d20.DifficultyClass
	}{
		{"modifier 0 and bonus +2", itemUseSpec + "FR-001 table 4.1a", "happy", 10, proficiency(1), 10},
		{"modifier +3 and bonus +2", itemUseSpec + "FR-001 table 4.1a", "happy", 16, proficiency(1), 13},
		{"modifier +5 and bonus +6", itemUseSpec + "FR-001 table 4.1a", "unhappy (boundary)", 20, proficiency(17), 19},
		{"modifier -1 and bonus +2", itemUseSpec + "FR-001 table 4.1a", "unhappy (boundary)", 8, proficiency(1), 9},
		{"modifier -5 is not floored", itemUseSpec + "FR-001 table 4.1a", "unhappy (boundary)", 1, proficiency(1), 5},
		{"modifier 0 and bonus 0", itemUseSpec + "FR-001 FR-002 table 4.1a", "unhappy (empty)", 10, character.ProficiencyBonus{}, 8},
		{"the bonus is always added", itemUseSpec + "FR-002", "unhappy (rule interaction)", 16, proficiency(5), 14},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := combat.ItemSaveDCOf(dexModifier(c.score), c.pb); got != c.want {
				t.Errorf("ItemSaveDCOf = %d, want %d [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/item-use-effects.md FR-060 - Cost: the five thrown kinds replace an attack, Basic Poison, Antitoxin and the Potion of Healing are Bonus Actions, the Healer's Kit is a Utilize action; an invalid kind costs nothing and ItemUseKind has exactly nine members with their content ids.
// Path: happy.
func TestItemUseRulesCostOf(t *testing.T) {
	t.Parallel()
	want := []struct {
		kind combat.ItemUseKind
		id   string
		cost combat.ItemActionCost
	}{
		{combat.ItemUseAcid, "acid", combat.ItemActionCostReplacesOneAttack},
		{combat.ItemUseAlchemistsFire, "alchemists_fire", combat.ItemActionCostReplacesOneAttack},
		{combat.ItemUseHolyWater, "holy_water", combat.ItemActionCostReplacesOneAttack},
		{combat.ItemUseOil, "oil", combat.ItemActionCostReplacesOneAttack},
		{combat.ItemUseNet, "net", combat.ItemActionCostReplacesOneAttack},
		{combat.ItemUseBasicPoison, "poison_basic", combat.ItemActionCostBonusAction},
		{combat.ItemUseAntitoxin, "antitoxin", combat.ItemActionCostBonusAction},
		{combat.ItemUsePotionOfHealing, "potion_of_healing", combat.ItemActionCostBonusAction},
		{combat.ItemUseHealersKit, "healers_kit", combat.ItemActionCostUtilizeAction},
	}
	for _, w := range want {
		if got := (combat.ItemUseRules{}).CostOf(w.kind); got != w.cost || !got.Valid() {
			t.Errorf("CostOf(%v) = %v, want %v", w.kind, got, w.cost)
		}
		if w.kind.String() != w.id || w.kind.ItemID() != content.MustItemID(w.id) || !w.kind.Valid() {
			t.Errorf("kind %d = %q / %q, want %q", w.kind, w.kind.String(), w.kind.ItemID(), w.id)
		}
	}
	for _, bad := range []combat.ItemUseKind{0, -1, 10} {
		if c := (combat.ItemUseRules{}).CostOf(bad); c != 0 || c.Valid() || c.String() != "unknown" || bad.Valid() || bad.ItemID() != "" || bad.String() != "unknown" {
			t.Errorf("invalid kind %d: cost %v, id %q", bad, c, bad.ItemID())
		}
	}
	for cost, name := range map[combat.ItemActionCost]string{combat.ItemActionCostReplacesOneAttack: "replaces_one_attack",
		combat.ItemActionCostBonusAction: "bonus_action", combat.ItemActionCostUtilizeAction: "utilize_action"} {
		if cost.String() != name {
			t.Errorf("cost %d = %q, want %q", cost, cost.String(), name)
		}
	}
}

// Requirement: doc/specs/item-use-effects.md FR-003 - the facts of the five thrown items: target kind, range, sight, Dexterity save, damage dice and type, and the effect of a failed save; exactly five members.
// Path: happy.
func TestThrownItemFactsOf(t *testing.T) {
	t.Parallel()
	type facts struct {
		creaturesOnly bool
		rng           int
		sight         bool
		damage        string
		typ           content.DamageType
		fiendUndead   bool
		burning, oil  bool
		restrains     bool
	}
	want := map[combat.ThrownItemKind]facts{
		combat.ThrownAcid:           {false, 20, true, "2d6", content.DamageAcid, false, false, false, false},
		combat.ThrownAlchemistsFire: {false, 20, true, "1d4", content.DamageFire, false, true, false, false},
		combat.ThrownHolyWater:      {true, 20, true, "2d8", content.DamageRadiant, true, false, false, false},
		combat.ThrownOil:            {false, 20, false, "", 0, false, false, true, false},
		combat.ThrownNet:            {true, 15, true, "", 0, false, false, false, true},
	}
	for kind, w := range want {
		f := combat.ThrownItemFactsOf(kind)
		damage := ""
		if e, ok := f.Damage(); ok {
			damage = e.String()
		}
		got := facts{f.CreaturesOnly(), f.RangeFeet(), f.SightRequired(), damage, f.DamageType(), f.OnlyFiendsAndUndead(), f.StartsBurning(), f.CoatsInOil(), f.RestrainsTarget()}
		if got != w || f.SaveAbility() != ability.Dexterity {
			t.Errorf("%v facts = %+v (save %v), want %+v with a Dexterity save", kind, got, f.SaveAbility(), w)
		}
		if kind.ItemUseKind() != combat.ItemUseKind(kind) || kind.ItemID() != content.MustItemID(kind.String()) {
			t.Errorf("%v: item %v / %q", kind, kind.ItemUseKind(), kind.ItemID())
		}
	}
	zero := combat.ThrownItemFactsOf(0)
	if _, ok := zero.Damage(); ok || zero != (combat.ThrownItemFacts{}) || combat.ThrownItemKind(6).Valid() || combat.ThrownItemKind(0).ItemUseKind() != 0 || combat.ThrownItemKind(6).ItemUseKind() != 0 || combat.ThrownItemKind(-1).ItemUseKind() != 0 || combat.ThrownItemKind(6).ItemID() != "" || combat.ThrownItemKind(6).String() != "unknown" {
		t.Errorf("invalid kind facts = %+v, want the zero facts", zero)
	}
}

// Requirement: doc/specs/item-use-effects.md FR-004 - ThrownItemRules.Replacement returns a replace-by-item AttackReplacement with the kind's ItemID, and records nothing.
// Path: happy.
func TestThrownItemRulesReplacement(t *testing.T) {
	t.Parallel()
	for k := combat.ThrownAcid; k <= combat.ThrownNet; k++ {
		r := (combat.ThrownItemRules{}).Replacement(k)
		item, ok := r.Source().Item()
		if !r.Valid() || r.Kind() != combat.AttackReplacementKindReplace || !ok || item != k.ItemID() {
			t.Errorf("Replacement(%v) = %+v, want a replace by item %q", k, r, k.ItemID())
		}
	}
	if r := (combat.ThrownItemRules{}).Replacement(0); r.Valid() {
		t.Errorf("Replacement(0) = %+v, want the zero value", r)
	}
}
