package combat_test

import (
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

const meSpec = "doc/specs/magical-effects.md "

func meMust[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func meSpell(id string) combat.EffectSource {
	return meMust(combat.SpellSource(content.MustSpellID(id)))
}

func meItem(id string) combat.EffectSource {
	return meMust(combat.MagicItemSource(content.MustMagicItemID(id)))
}

func meLabeled(id string) combat.EffectSource {
	return meMust(combat.LabeledMagicalSource(content.MustFeatureID(id)))
}

func meNonMagical(id string) combat.EffectSource {
	return meMust(combat.NonMagicalSource(content.MustFeatureID(id)))
}

func meEID(s string) combat.OngoingEffectID { return meMust(combat.NewOngoingEffectID(s)) }

func meEffect(id string, c combat.OngoingEffectCategory, src combat.EffectSource) combat.OngoingEffect {
	return meMust(combat.NewOngoingEffect(meEID(id), c, src))
}

// meBless and meCurse are the baseline effects of table 4.8a: the spell bless and the curse c1.
func meBless() combat.OngoingEffect {
	return meEffect("bless", combat.OngoingEffectSpell, meSpell("bless"))
}

func meCurse(id string) combat.OngoingEffect {
	return meEffect(id, combat.OngoingEffectCurse, meLabeled("hag_curse"))
}

func meEffects(es ...combat.OngoingEffect) combat.OngoingEffects {
	return meMust(combat.NewOngoingEffects(es))
}

// meConditions is the baseline set of table 4.8a: Poisoned and Frightened of a source.
func meConditions(cs ...combat.Condition) combat.ConditionSet {
	return combat.MustConditionSet(cs, nil)
}

func meBaseline() combat.ConditionSet {
	return meConditions(combat.Poisoned{}, combat.Frightened{Source: gid("scary")})
}

func meKinds(ks ...content.ConditionKind) []content.ConditionKind { return ks }

func meExcept(conds []content.ConditionKind, effs ...string) combat.CarryOverExcept {
	var ids []combat.OngoingEffectID
	for _, e := range effs {
		ids = append(ids, meEID(e))
	}
	return meMust(combat.NewCarryOverExcept(conds, ids))
}
