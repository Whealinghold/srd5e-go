package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: monSpec FR-034..FR-042 - the numeric bounds of the stat block
// part, notation and plan constructors are inclusive: the lowest and highest
// legal values are accepted.
// Path: unhappy (boundary), happy at the edge.
func TestMonsterPartNotationPlanBoundsInclusive(t *testing.T) {
	t.Parallel()
	hi := content.MonsterMaxNumber
	fire := must(content.NewSpecificDamageType(content.DamageFire))
	checks := map[string]error{
		"PartPerDay min":           second(content.NewPartPerDay(content.MonsterMinUses, 0)),
		"PartPerDay max":           second(content.NewPartPerDay(hi, hi)),
		"LegendaryActions max":     second(content.NewLegendaryActions(hi, hi, []content.StatBlockPart{minimalPart(t, "Move")})),
		"AttackRange min":          second(content.NewStatBlockAttackRange(content.MonsterMinRange, 0)),
		"AttackRange max":          second(content.NewStatBlockAttackRange(hi-1, hi)),
		"AttackRange max normal":   second(content.NewStatBlockAttackRange(hi, 0)),
		"Attack reach max":         second(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Reach: hi, HasReach: true})),
		"SavingThrow min":          second(content.NewStatBlockSavingThrow(ability.Strength, content.MonsterMinSaveDC, "", "")),
		"SavingThrow max":          second(content.NewStatBlockSavingThrow(ability.Strength, hi, "", "")),
		"Immune min":               second(content.NewImmuneToPartForHours(1)),
		"Immune max":               second(content.NewImmuneToPartForHours(hi)),
		"Damage max":               second(content.NewStatBlockDamage(hi, dice.Expression{}, fire, false)),
		"AttackSlotGroup max":      second(content.NewAttackSlotGroup(hi, []string{"Bite"})),
		"AttackSlotGroup min":      second(content.NewAttackSlotGroup(content.MonsterMinAttackCount, []string{"Bite"})),
		"ReplacementOption min":    second(content.NewReplacementOption("Spellcasting", "Fire Bolt", content.MonsterMinSpellLevel)),
		"ReplacementOption max":    second(content.NewReplacementOption("Spellcasting", "Fire Bolt", content.MonsterMaxSpellLevel)),
		"ReplacementOption lowest": second(content.NewReplacementOption("Spellcasting", "Fire Bolt", 0)),
		"ReplacementRule max": second(content.NewReplacementRule(hi, content.ReplacementScopeAnyAttack{},
			[]content.ReplacementOption{must(content.NewReplacementOption("Breath", "", 0))}, content.ReplaceOnce)),
	}
	for name, err := range checks {
		if err != nil {
			t.Errorf("%s: error = %v, want nil [FR-034..FR-042]", name, err)
		}
	}
}

func second[T any](_ T, err error) error { return err }

func minimalPart(t *testing.T, name string) content.StatBlockPart {
	t.Helper()
	return must(content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T."}))
}

// Requirement: monSpec FR-034 - the part keeps its own copy of the clauses, so
// changing the caller's slice afterwards does not change the part.
// Path: unhappy (aliasing).
func TestStatBlockPartCopiesClauses(t *testing.T) {
	t.Parallel()
	hit := must(content.NewEffectClause(content.ClauseHit, "Hit.", nil, nil))
	miss := must(content.NewEffectClause(content.ClauseMiss, "Miss.", nil, nil))
	in := []content.EffectClause{hit}
	p := must(content.NewStatBlockPart("Bite", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Clauses: in}))
	in[0] = miss
	if got := p.Clauses(); len(got) != 1 || got[0].Label() != content.ClauseHit {
		t.Errorf("Clauses() = %v, want the original hit clause [FR-034]", got)
	}
}

// Requirement: monSpec FR-034 - StatBlockPart.Equal compares the attack, the
// saving throw, the plan and every clause, each on its own.
// Path: per case.
func TestStatBlockPartEqualEachMember(t *testing.T) {
	t.Parallel()
	plan := func(n int) content.MultiattackPlan {
		return must(content.NewMultiattackPlan([]content.MultiattackVariant{must(content.NewMultiattackVariant("",
			[]content.AttackSlotGroup{must(content.NewAttackSlotGroup(n, []string{"Bite"}))}, nil, nil))}))
	}
	atk := func(b int) content.StatBlockAttack {
		return must(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Bonus: b}))
	}
	save := func(dc int) content.StatBlockSavingThrow {
		return must(content.NewStatBlockSavingThrow(ability.Dexterity, dc, "", ""))
	}
	clause := func(text string) []content.EffectClause {
		return []content.EffectClause{must(content.NewEffectClause(content.ClauseHit, text, nil, nil))}
	}
	mk := func(name string, f content.StatBlockPartFields) content.StatBlockPart {
		f.Usage, f.Text = content.PartUnlimited{}, "T."
		return must(content.NewStatBlockPart(name, f))
	}
	same := mk("Multiattack", content.StatBlockPartFields{Attack: atk(1), SavingThrow: save(10), Clauses: clause("a"), Plan: plan(2)})
	if !same.Equal(mk("Multiattack", content.StatBlockPartFields{Attack: atk(1), SavingThrow: save(10), Clauses: clause("a"), Plan: plan(2)})) {
		t.Error("identical parts are not Equal [FR-034]")
	}
	for name, o := range map[string]content.StatBlockPart{
		"attack": mk("Multiattack", content.StatBlockPartFields{Attack: atk(2), SavingThrow: save(10), Clauses: clause("a"), Plan: plan(2)}),
		"save":   mk("Multiattack", content.StatBlockPartFields{Attack: atk(1), SavingThrow: save(11), Clauses: clause("a"), Plan: plan(2)}),
		"clause": mk("Multiattack", content.StatBlockPartFields{Attack: atk(1), SavingThrow: save(10), Clauses: clause("b"), Plan: plan(2)}),
		"plan":   mk("Multiattack", content.StatBlockPartFields{Attack: atk(1), SavingThrow: save(10), Clauses: clause("a"), Plan: plan(3)}),
	} {
		if same.Equal(o) || o.Equal(same) {
			t.Errorf("parts differing only in %s are Equal [FR-034]", name)
		}
	}
}

// Requirement: monSpec FR-035 - LegendaryActions.Equal compares the use count.
// Path: unhappy (distinct values).
func TestLegendaryActionsEqualUses(t *testing.T) {
	t.Parallel()
	acts := []content.StatBlockPart{minimalPart(t, "Move")}
	a := must(content.NewLegendaryActions(3, 0, acts))
	if !a.Equal(must(content.NewLegendaryActions(3, 0, acts))) || a.Equal(must(content.NewLegendaryActions(2, 0, acts))) {
		t.Error("LegendaryActions.Equal does not compare uses [FR-035]")
	}
}

// Requirement: monSpec FR-037 - EffectClause.Equal compares label, text, rider
// and damage, each on its own; the clause copies its damage slice.
// Path: per case.
func TestEffectClauseEqualEachMember(t *testing.T) {
	t.Parallel()
	d := must(content.NewStatBlockDamage(1, dice.Expression{}, must(content.NewSpecificDamageType(content.DamageFire)), false))
	d2 := must(content.NewStatBlockDamage(2, dice.Expression{}, must(content.NewSpecificDamageType(content.DamageFire)), false))
	rider := must(content.NewImmuneToPartForHours(24))
	base := must(content.NewEffectClause(content.ClauseSuccess, "T.", rider, []content.StatBlockDamage{d}))
	for name, o := range map[string]content.EffectClause{
		"label":  must(content.NewEffectClause(content.ClauseFailure, "T.", nil, []content.StatBlockDamage{d})),
		"text":   must(content.NewEffectClause(content.ClauseSuccess, "U.", rider, []content.StatBlockDamage{d})),
		"rider":  must(content.NewEffectClause(content.ClauseSuccess, "T.", content.HalfDamage{}, []content.StatBlockDamage{d})),
		"damage": must(content.NewEffectClause(content.ClauseSuccess, "T.", rider, []content.StatBlockDamage{d2})),
	} {
		if base.Equal(o) {
			t.Errorf("clauses differing only in %s are Equal [FR-037]", name)
		}
	}
	in := []content.StatBlockDamage{d}
	c := must(content.NewEffectClause(content.ClauseHit, "T.", nil, in))
	in[0] = d2
	if got := c.Damage(); len(got) != 1 || got[0].Number() != 1 {
		t.Errorf("Damage() = %v, want the original damage [FR-037]", got)
	}
}

// Requirement: monSpec FR-098 - ReplacementRule, MultiattackVariant and
// AdditionalUse Equal compare every member, each on its own.
// Path: per case.
func TestMultiattackEqualEachMember(t *testing.T) {
	t.Parallel()
	opt := func(p string) []content.ReplacementOption {
		return []content.ReplacementOption{must(content.NewReplacementOption(p, "", 0))}
	}
	scope := must(content.NewReplacementScopeOneAttackPart("Bite"))
	rule := func(n int, s content.ReplacementScope, o string, l content.ReplacementLimit) content.ReplacementRule {
		return must(content.NewReplacementRule(n, s, opt(o), l))
	}
	base := rule(1, scope, "Breath", content.ReplaceOnce)
	if !base.Equal(rule(1, scope, "Breath", content.ReplaceOnce)) {
		t.Error("identical rules are not Equal [FR-098]")
	}
	for name, o := range map[string]content.ReplacementRule{
		"count":   rule(2, scope, "Breath", content.ReplaceOnce),
		"scope":   rule(1, content.ReplacementScopeAnyAttack{}, "Breath", content.ReplaceOnce),
		"options": rule(1, scope, "Roar", content.ReplaceOnce),
		"limit":   rule(1, scope, "Breath", content.ReplaceAnyNumber),
	} {
		if base.Equal(o) {
			t.Errorf("rules differing only in %s are Equal [FR-098]", name)
		}
	}

	groups := []content.AttackSlotGroup{must(content.NewAttackSlotGroup(2, []string{"Bite"}))}
	uses := func(n string) []content.AdditionalUse {
		return []content.AdditionalUse{must(content.NewAdditionalUse([]string{n}))}
	}
	v := func(req string, u []content.AdditionalUse, r content.ReplacementRule) content.MultiattackVariant {
		return must(content.NewMultiattackVariant(req, groups, u, []content.ReplacementRule{r}))
	}
	vb := v("Reaction", uses("Roar"), base)
	if !vb.Equal(v("Reaction", uses("Roar"), base)) {
		t.Error("identical variants are not Equal [FR-098]")
	}
	for name, o := range map[string]content.MultiattackVariant{
		"requiresUsed":  v("Other", uses("Roar"), base),
		"additionalUse": v("Reaction", uses("Cry"), base),
		"replacements":  v("Reaction", uses("Roar"), rule(2, scope, "Breath", content.ReplaceOnce)),
	} {
		if vb.Equal(o) {
			t.Errorf("variants differing only in %s are Equal [FR-098]", name)
		}
	}
	if must(content.NewAttackSlotGroup(2, []string{"Bite"})).Equal(must(content.NewAttackSlotGroup(3, []string{"Bite"}))) {
		t.Error("groups with different counts are Equal [FR-098]")
	}
	if must(content.NewAdditionalUse([]string{"A"})).Equal(must(content.NewAdditionalUse([]string{"B"}))) {
		t.Error("additional uses with different options are Equal [FR-098]")
	}
}

// Requirement: monSpec FR-098 - the plan types return copies of their slices.
// Path: unhappy (aliasing).
func TestAdditionalUseOptionsCopy(t *testing.T) {
	t.Parallel()
	in := []string{"Roar"}
	u := must(content.NewAdditionalUse(in))
	in[0] = "Cry"
	u.Options()[0] = "Hiss"
	if got := u.Options(); len(got) != 1 || got[0] != "Roar" {
		t.Errorf("Options() = %v, want [Roar] [FR-098]", got)
	}
}

// Requirement: monSpec FR-037 - EffectClause.Equal compares the label alone
// when text, rider and damage are the same.
// Path: unhappy (distinct values).
func TestEffectClauseEqualLabelOnly(t *testing.T) {
	t.Parallel()
	hit := must(content.NewEffectClause(content.ClauseHit, "T.", nil, nil))
	miss := must(content.NewEffectClause(content.ClauseMiss, "T.", nil, nil))
	if hit.Equal(miss) {
		t.Error("clauses differing only in label are Equal [FR-037]")
	}
}
