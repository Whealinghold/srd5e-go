package combat

import "slices"

// AssistanceJudgement is the Game Master's final say on whether assistance is possible
// (SRD 5.2.1, "Help [Action]", p. 182). It is host input. The zero value is not a
// judgement and counts as not possible.
type AssistanceJudgement int

// The two judgements.
const (
	AssistanceJudgementPossible AssistanceJudgement = iota + 1
	AssistanceJudgementNotPossible
)

// String returns "possible", "not_possible" or "unknown".
func (j AssistanceJudgement) String() string {
	return enumName([]string{"possible", "not_possible"}, int(j))
}

// AttackAssistTarget says what the host found about the creature whose attack roll Help
// would assist against (SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183). It is
// host input: this package has no map or allegiance model. The zero value is not a target
// and counts as not an enemy.
type AttackAssistTarget int

// The three findings.
const (
	AttackAssistTargetEnemyWithinFiveFeet AttackAssistTarget = iota + 1
	AttackAssistTargetEnemyBeyondFiveFeet
	AttackAssistTargetNotAnEnemy
)

// String returns "enemy_within_five_feet", "enemy_beyond_five_feet", "not_an_enemy" or "unknown".
func (t AttackAssistTarget) String() string {
	return enumName([]string{"enemy_within_five_feet", "enemy_beyond_five_feet", "not_an_enemy"}, int(t))
}

// AbilityCheckHelpRefusal says why ability check help was refused, the first that applies
// in this order. The zero value is not a refusal.
type AbilityCheckHelpRefusal int

// The four reasons ability check help is refused. LimitExceeded is the plan P10 cap of 256 pending benefits.
const (
	AbilityCheckHelpRefusalCannotHelpSelf AbilityCheckHelpRefusal = iota + 1
	AbilityCheckHelpRefusalHelperLacksProficiency
	AbilityCheckHelpRefusalAssistanceNotPossible
	AbilityCheckHelpRefusalLimitExceeded
)

// String returns the snake_case name of r, or "unknown".
func (r AbilityCheckHelpRefusal) String() string {
	return enumName([]string{"cannot_help_self", "helper_lacks_proficiency", "assistance_not_possible", valueResultLimitExceeded}, int(r))
}

// AttackRollHelpRefusal says why attack roll help was refused, the first that applies in
// this order. The zero value is not a refusal.
type AttackRollHelpRefusal int

// The three reasons attack roll help is refused. LimitExceeded is the plan P10 cap of 256 pending benefits.
const (
	AttackRollHelpRefusalTargetNotAnEnemy AttackRollHelpRefusal = iota + 1
	AttackRollHelpRefusalEnemyNotWithinFiveFeet
	AttackRollHelpRefusalLimitExceeded
)

// String returns "target_not_an_enemy", "enemy_not_within_five_feet" or "unknown".
func (r AttackRollHelpRefusal) String() string {
	return enumName([]string{"target_not_an_enemy", "enemy_not_within_five_feet", valueResultLimitExceeded}, int(r))
}

// HelpBenefit is what the Help action leaves behind until it is used or expires. Variants:
// AbilityCheckHelp, AttackRollHelp. Both are comparable values. The zero value of either is
// invalid.
type HelpBenefit interface {
	isHelpBenefit()
	// Valid reports whether the benefit was built by its constructor.
	Valid() bool
	// Combatants returns a new slice [helper, ally] for an ability check help and
	// [helper, enemy] for an attack roll help.
	Combatants() []CombatantID
}

// AbilityCheckHelp: the helper gave the ally Advantage on the next ability check the ally
// makes with the proficiency (SRD 5.2.1, "Help [Action]", p. 182). Build it with
// NewAbilityCheckHelp.
type AbilityCheckHelp struct {
	helperID, ally CombatantID
	proficiency    AssistedProficiency
}

// NewAbilityCheckHelp returns the benefit. It returns an error matching
// srderr.ErrInvalidInput for a blank id, a helper equal to the ally or an invalid
// proficiency. It never panics.
func NewAbilityCheckHelp(helper, ally CombatantID, proficiency AssistedProficiency) (AbilityCheckHelp, error) {
	b := AbilityCheckHelp{helper, ally, proficiency}
	if !b.Valid() {
		return AbilityCheckHelp{}, invalidInput("ability check help needs a helper, a different ally and a proficiency")
	}
	return b, nil
}

// Valid reports whether b names a helper and a different ally and a valid proficiency.
func (b AbilityCheckHelp) Valid() bool {
	return !b.helperID.IsZero() && !b.ally.IsZero() && b.helperID != b.ally && b.proficiency.Valid()
}

// Proficiency returns the skill or tool the help is for.
func (b AbilityCheckHelp) Proficiency() AssistedProficiency { return b.proficiency }

func (AbilityCheckHelp) isHelpBenefit() {}

// Combatants returns a new slice [helper, ally].
func (b AbilityCheckHelp) Combatants() []CombatantID { return []CombatantID{b.helperID, b.ally} }

// AttackRollHelp: the helper gave an ally Advantage on the next attack roll against the
// enemy (SRD 5.2.1, "Help [Action]", p. 183). Build it with NewAttackRollHelp.
type AttackRollHelp struct{ helperID, enemy CombatantID }

// NewAttackRollHelp returns the benefit. It returns an error matching
// srderr.ErrInvalidInput for a blank id or a helper equal to the enemy. It never panics.
func NewAttackRollHelp(helper, enemy CombatantID) (AttackRollHelp, error) {
	b := AttackRollHelp{helper, enemy}
	if !b.Valid() {
		return AttackRollHelp{}, invalidInput("attack roll help needs a helper and a different enemy")
	}
	return b, nil
}

// Valid reports whether b names a helper and a different enemy.
func (b AttackRollHelp) Valid() bool {
	return !b.helperID.IsZero() && !b.enemy.IsZero() && b.helperID != b.enemy
}

func (AttackRollHelp) isHelpBenefit() {}

// Combatants returns a new slice [helper, enemy].
func (b AttackRollHelp) Combatants() []CombatantID { return []CombatantID{b.helperID, b.enemy} }

// AlliedHelpers is the set of helpers the host says are allies of an attacker. The zero
// value is empty and valid. It holds a slice, so compare with Equal, not ==. Safe for
// concurrent use.
type AlliedHelpers struct{ ids []CombatantID }

// NewAlliedHelpers returns the set, dropping repeats. It returns an error matching
// srderr.ErrInvalidInput for a blank id, and a LimitExceededError for more than 256 (plan P10). It copies ids and never panics.
func NewAlliedHelpers(ids ...CombatantID) (AlliedHelpers, error) {
	if err := overCap("allied helpers", len(ids), maxInputList); err != nil {
		return AlliedHelpers{}, err
	}
	var out AlliedHelpers
	for _, id := range ids {
		if id.IsZero() {
			return AlliedHelpers{}, invalidInput("allied helper needs an id")
		}
		if !slices.Contains(out.ids, id) {
			out.ids = append(out.ids, id)
		}
	}
	return out, nil
}

// Contains reports whether id is in the set.
func (a AlliedHelpers) Contains(id CombatantID) bool { return slices.Contains(a.ids, id) }

// Equal reports whether a and o hold the same ids, whatever order.
func (a AlliedHelpers) Equal(o AlliedHelpers) bool { return sameSet(a.ids, o.ids) }
