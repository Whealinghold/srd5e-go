package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// bardDieFeatureGate is the gate the two College of Lore die features share: the level and the subclass.
func bardDieFeatureGate(level BardLevel, subclass content.SubclassID, from int) BardReason {
	switch {
	case level.IsZero():
		return BardNotABard
	case level.n < from || subclass != CollegeOfLoreSubclassID:
		return BardFeatureUnavailable
	}
	return 0
}

// CuttingWordsTrigger is what Cutting Words answers, a closed set: CuttingWordsRoll and CuttingWordsDamage. The
// trigger is a character value, never a combat type (class-bard.md FR-072).
type CuttingWordsTrigger interface{ isCuttingWordsTrigger() }

// CuttingWordsRoll is an ability check or attack roll that succeeded. Construct it with a keyed literal.
type CuttingWordsRoll struct{ Result AdjustableD20Result }

// CuttingWordsDamage is a damage roll's total, not negative. Construct it with a keyed literal.
type CuttingWordsDamage struct{ Total int }

func (CuttingWordsRoll) isCuttingWordsTrigger()   {}
func (CuttingWordsDamage) isCuttingWordsTrigger() {}

// CuttingWordsPlan is what a use of Cutting Words needs rolled: the die. Only PlanCuttingWords builds one.
type CuttingWordsPlan struct {
	id      Identifier
	die     BardicDie
	trigger CuttingWordsTrigger
	pool    BardicInspirationPool
}

// Die returns the die to roll.
func (p CuttingWordsPlan) Die() BardicDie { return p.die }

// PlanCuttingWords decides whether the Bard can use Cutting Words (SRD 5.2.1, "Level 3: Cutting Words", p. 35;
// class-bard.md FR-074). The Reaction cost is the host's. The checks run in this order and the first that holds is
// returned with the zero plan: BardNotABard, BardFeatureUnavailable (below Bard level 3, or the subclass is not the
// College of Lore), BardNoUsesRemaining, then BardIneligibleTrigger (a saving throw, or a roll whose outcome is not a
// success). The reason is zero when the die may be rolled. The pool's maximum is first recomputed from charisma.
//
// It returns an error matching srderr.ErrInvalidInput for a nil trigger, a CuttingWordsRoll with a nil or zero result
// or a negative damage total.
func PlanCuttingWords(id Identifier, level BardLevel, subclass content.SubclassID, pool BardicInspirationPool, charisma ability.Modifier, trigger CuttingWordsTrigger) (CuttingWordsPlan, BardReason, error) {
	var eligible bool
	switch t := trigger.(type) {
	case CuttingWordsRoll:
		if err := triggerProblem(t.Result); err != nil {
			return CuttingWordsPlan{}, 0, err
		}
		eligible = !t.Result.SavingThrow() && t.Result.Outcome() == d20.Success
	case CuttingWordsDamage:
		if t.Total < 0 {
			return CuttingWordsPlan{}, 0, invalidInput("Cutting Words on a negative damage total")
		}
		eligible = true
	default:
		return CuttingWordsPlan{}, 0, invalidInput("Cutting Words on a nil trigger")
	}
	pool = pool.WithMaximumFrom(charisma)
	switch reason := bardDieFeatureGate(level, subclass, bardSubclassLevel); {
	case reason != 0:
		return CuttingWordsPlan{}, reason, nil
	case pool.remaining < 1:
		return CuttingWordsPlan{}, BardNoUsesRemaining, nil
	case !eligible:
		return CuttingWordsPlan{}, BardIneligibleTrigger, nil
	}
	return CuttingWordsPlan{id: id, die: BardicDieAt(level), trigger: trigger, pool: pool}, 0, nil
}

// CuttingWordsApplied is the outcome of ApplyCuttingWords.
type CuttingWordsApplied struct {
	result  AdjustableD20Result
	damage  int
	clamped bool
	pool    BardicInspirationPool
	event   BardicInspirationExpended
}

// Result returns the ability check or attack roll with the number rolled subtracted, or nil for a damage roll.
func (a CuttingWordsApplied) Result() AdjustableD20Result { return a.result }

// Damage returns the reduced damage total, never below zero; 0 for an ability check or attack roll.
func (a CuttingWordsApplied) Damage() int { return a.damage }

// Clamped reports whether the number rolled exceeded the damage total, so the damage was held at zero.
func (a CuttingWordsApplied) Clamped() bool { return a.clamped }

// Pool returns the pool with one use expended, whether or not the success became a failure.
func (a CuttingWordsApplied) Pool() BardicInspirationPool { return a.pool }

// Event returns the BardicInspirationExpended event.
func (a CuttingWordsApplied) Event() BardicInspirationExpended { return a.event }

// ApplyCuttingWords subtracts the number rolled from the trigger: through adjustedBy(-rolled) for an ability check or
// attack roll (class-bard.md FR-073, FR-104, so a natural 20 stays a critical hit), and from the damage total for a
// damage roll, held at zero (FR-107). The use is spent either way (FR-075). It returns an error matching
// srderr.ErrInvalidInput for a zero plan or a number outside 1 to the die's sides.
func ApplyCuttingWords(plan CuttingWordsPlan, rolled int) (CuttingWordsApplied, error) {
	if err := rollProblem(plan.die, rolled); err != nil || plan.trigger == nil {
		return CuttingWordsApplied{}, firstError(err, invalidInput("Cutting Words applied from a zero plan"))
	}
	pool := plan.pool.spend()
	out := CuttingWordsApplied{pool: pool, event: BardicInspirationExpended{id: plan.id, feature: bardFeatureCutting, die: plan.die, rolled: rolled}}
	switch t := plan.trigger.(type) {
	case CuttingWordsRoll:
		adjusted, err := t.Result.adjustedBy(-rolled)
		if err != nil {
			return CuttingWordsApplied{}, err
		}
		out.result = adjusted
	case CuttingWordsDamage:
		out.damage = max(t.Total-rolled, 0)
		out.clamped = rolled > t.Total
	}
	return out, nil
}

// PeerlessSkillPlan is what a use of Peerless Skill needs rolled: the die. Only PlanPeerlessSkill builds one.
type PeerlessSkillPlan struct {
	id     Identifier
	die    BardicDie
	failed AdjustableD20Result
	pool   BardicInspirationPool
}

// Die returns the die to roll.
func (p PeerlessSkillPlan) Die() BardicDie { return p.die }

// PlanPeerlessSkill decides whether the Bard can use Peerless Skill on the ability check or attack roll it failed
// (SRD 5.2.1, "Level 14: Peerless Skill", p. 35; class-bard.md FR-083). The checks run in this order: BardNotABard,
// BardFeatureUnavailable (below Bard level 14, or the subclass is not the College of Lore), BardNoUsesRemaining, then
// BardIneligibleTrigger (a saving throw, or a result that is not a failure). The reason is zero when the die may be
// rolled. The pool's maximum is first recomputed from charisma. It returns an error matching srderr.ErrInvalidInput
// for a nil or zero result.
func PlanPeerlessSkill(id Identifier, level BardLevel, subclass content.SubclassID, pool BardicInspirationPool, charisma ability.Modifier, failed AdjustableD20Result) (PeerlessSkillPlan, BardReason, error) {
	if err := triggerProblem(failed); err != nil {
		return PeerlessSkillPlan{}, 0, err
	}
	pool = pool.WithMaximumFrom(charisma)
	switch reason := bardDieFeatureGate(level, subclass, bardPeerlessSkillLevel); {
	case reason != 0:
		return PeerlessSkillPlan{}, reason, nil
	case pool.remaining < 1:
		return PeerlessSkillPlan{}, BardNoUsesRemaining, nil
	case failed.SavingThrow() || failed.Outcome() != d20.Failure:
		return PeerlessSkillPlan{}, BardIneligibleTrigger, nil
	}
	return PeerlessSkillPlan{id: id, die: BardicDieAt(level), failed: failed, pool: pool}, 0, nil
}

// PeerlessSkillApplied is the outcome of ApplyPeerlessSkill.
type PeerlessSkillApplied struct {
	result AdjustableD20Result
	spent  bool
	pool   BardicInspirationPool
	events []Event
}

// Result returns the ability check or attack roll with the number rolled added to its total.
func (a PeerlessSkillApplied) Result() AdjustableD20Result { return a.result }

// Spent reports whether the use was expended, which it is only when the result became a success.
func (a PeerlessSkillApplied) Spent() bool { return a.spent }

// Pool returns the pool: one use expended on a success, unchanged on a failure.
func (a PeerlessSkillApplied) Pool() BardicInspirationPool { return a.pool }

// Events returns a new slice holding the BardicInspirationExpended event when the use was spent, and none otherwise.
func (a PeerlessSkillApplied) Events() []Event { return slices.Clone(a.events) }

// ApplyPeerlessSkill adds the number rolled to the failed result through adjustedBy(rolled), so the natural 1 rule of
// an attack roll holds (class-bard.md FR-081, FR-082, FR-104). The use is expended only when the new result is a
// success. It returns an error matching srderr.ErrInvalidInput for a zero plan or a number outside 1 to the die's sides.
func ApplyPeerlessSkill(plan PeerlessSkillPlan, rolled int) (PeerlessSkillApplied, error) {
	if err := rollProblem(plan.die, rolled); err != nil || plan.failed == nil {
		return PeerlessSkillApplied{}, firstError(err, invalidInput("Peerless Skill applied from a zero plan"))
	}
	adjusted, err := plan.failed.adjustedBy(rolled)
	if err != nil {
		return PeerlessSkillApplied{}, err
	}
	out := PeerlessSkillApplied{result: adjusted, pool: plan.pool, events: []Event{}}
	if adjusted.Outcome() == d20.Success {
		out.spent = true
		out.pool = plan.pool.spend()
		out.events = []Event{BardicInspirationExpended{id: plan.id, feature: bardFeaturePeerless, die: plan.die, rolled: rolled}}
	}
	return out, nil
}

// CountercharmPlan is what a use of Countercharm needs rolled: the roll mode of the new save and so the number of d20s.
// Only PlanCountercharm builds one.
type CountercharmPlan struct {
	mode   modifier.RollMode
	failed d20.CheckResult
}

// Mode returns the roll mode of the rerolled save.
func (p CountercharmPlan) Mode() modifier.RollMode { return p.mode }

// DiceCount returns how many d20s to roll: one when the circumstances cancel, otherwise two.
func (p CountercharmPlan) DiceCount() int { return p.mode.DiceCount() }

// PlanCountercharm decides whether Countercharm can answer the failed saving throw and how many d20s the reroll takes
// (SRD 5.2.1, "Level 7: Countercharm", p. 33; class-bard.md FR-084, FR-086, FR-105). The Reaction cost is the host's.
// The circumstances of the original save carry over and one Advantage is added to them, so any Disadvantage cancels it
// (the rule that Advantage and Disadvantage cancel, p. 176). The checks run in this order: BardNotABard,
// BardBelowRequiredLevel (below Bard level 7), BardIneligibleEffect (conditions holds neither Charmed nor Frightened),
// then BardSaveNotFailed (the result is not a failed saving throw). The reason is zero when the d20s may be rolled.
// It returns an error matching srderr.ErrInvalidInput for a zero result.
func PlanCountercharm(level BardLevel, failed d20.CheckResult, circumstances []modifier.RollCircumstance, conditions []content.ConditionKind) (CountercharmPlan, BardReason, error) {
	if failed.Roll().IsZero() {
		return CountercharmPlan{}, 0, invalidInput("Countercharm on a zero saving throw result")
	}
	switch {
	case level.IsZero():
		return CountercharmPlan{}, BardNotABard, nil
	case level.n < bardCountercharmLevel:
		return CountercharmPlan{}, BardBelowRequiredLevel, nil
	case !slices.Contains(conditions, content.ConditionCharmed) && !slices.Contains(conditions, content.ConditionFrightened):
		return CountercharmPlan{}, BardIneligibleEffect, nil
	case failed.Kind() != d20.SavingThrow || failed.Outcome() != d20.Failure:
		return CountercharmPlan{}, BardSaveNotFailed, nil
	}
	all := append(slices.Clone(circumstances), modifier.Advantage{})
	return CountercharmPlan{mode: modifier.Resolve(all...), failed: failed}, 0, nil
}

// CountercharmResolved is the outcome of ResolveCountercharm: the failed save and the rerolled one. The Bardic
// Inspiration pool and the spell slots are untouched (class-bard.md FR-087).
type CountercharmResolved struct{ failed, rerolled d20.CheckResult }

// Failed returns the original, failed saving throw.
func (r CountercharmResolved) Failed() d20.CheckResult { return r.failed }

// Rerolled returns the new saving throw, made with the original modifiers against the original DC.
func (r CountercharmResolved) Rerolled() d20.CheckResult { return r.rerolled }

// ResolveCountercharm builds the d20 Roll of the plan's mode from the faces rolled and resolves it with the original
// modifiers and DC (class-bard.md FR-085, FR-106); the higher or lower die is chosen by the d20 package, never here.
// It returns an error matching srderr.ErrInvalidInput for a zero plan or faces that are not DiceCount values from 1 to 20.
func ResolveCountercharm(plan CountercharmPlan, faces []int) (CountercharmResolved, error) {
	if plan.failed.Roll().IsZero() {
		return CountercharmResolved{}, invalidInput("Countercharm resolved from a zero plan")
	}
	roll, err := d20.NewRoll(plan.mode, faces)
	if err != nil {
		return CountercharmResolved{}, err
	}
	rerolled, err := d20.ResolveCheck(d20.SavingThrow, roll, plan.failed.Modifiers(), plan.failed.Target())
	if err != nil { // unreachable: the roll is built and the modifiers and DC come from a resolved check
		return CountercharmResolved{}, err
	}
	return CountercharmResolved{failed: plan.failed, rerolled: rerolled}, nil
}

// JackOfAllTrades returns half the Proficiency Bonus, rounded down, for an ability check that uses a skill the
// character lacks and that does not otherwise use the Proficiency Bonus (SRD 5.2.1, "Level 2: Jack of All Trades",
// p. 32; class-bard.md FR-064, FR-065), and false otherwise. level is the Bard level (the zero value for none);
// skill is the skill of the check, or nil for a check that uses none; usesProficiency says the check already uses the
// Proficiency Bonus through another source such as a proficient tool. A skill with Expertise counts as proficient.
// The check composer places the modifier in RollModifiers.others.
func JackOfAllTrades(level BardLevel, bonus ProficiencyBonus, skill *content.Skill, proficiencies SkillProficiencies, usesProficiency bool) (modifier.Modifier, bool) {
	if level.n < 2 || skill == nil || usesProficiency || proficiencies.Proficient(*skill) {
		return modifier.Modifier{}, false
	}
	return modifier.New(bonus.Value() / 2), true
}

// AllowsSecondTarget reports whether the Bard can target a second creature with the spell: from Bard level 20 and
// only for Power Word Heal and Power Word Kill (SRD 5.2.1, "Level 20: Words of Creation", p. 33; class-bard.md
// FR-096). The second creature must be within 10 feet of the first target; geometry is the host's.
func AllowsSecondTarget(level BardLevel, spell content.SpellID) bool {
	return level.n >= bardWordsOfCreationLevel && slices.Contains(wordsOfCreationSpells(), spell)
}

// wordsOfCreationSpells returns the two spells a level 20 Bard always has prepared.
func wordsOfCreationSpells() []content.SpellID {
	return []content.SpellID{"power_word_heal", "power_word_kill"}
}
