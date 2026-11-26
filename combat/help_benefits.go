package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/modifier"
)

// HelpBenefits is the benefits the Help action has left and nobody has used, in the order
// they were given (SRD 5.2.1, "Help [Action]", pp. 182-183). Every operation returns a new
// value. The zero value is empty and valid. It holds a slice, so compare with Equal, not ==;
// accessors return copies. Safe for concurrent use.
type HelpBenefits struct{ list []HelpBenefit }

// validBenefit reports whether b is a non-nil, valid benefit.
func validBenefit(b HelpBenefit) bool { return b != nil && b.Valid() }

// NewHelpBenefits returns the benefits holding list, for a host restoring saved state. It
// returns an error matching srderr.ErrInvalidInput for a nil or invalid benefit, and a
// LimitExceededError for more than 256 (plan P10). It copies list and never panics.
func NewHelpBenefits(list []HelpBenefit) (HelpBenefits, error) {
	if err := overCap("help benefits", len(list), maxInputList); err != nil {
		return HelpBenefits{}, err
	}
	for _, b := range list {
		if !validBenefit(b) {
			return HelpBenefits{}, invalidInput("help benefit is invalid")
		}
	}
	return HelpBenefits{slices.Clone(list)}, nil
}

// Benefits returns a copy of the pending benefits, in the order they were given.
func (h HelpBenefits) Benefits() []HelpBenefit { return append([]HelpBenefit{}, h.list...) }

// Len returns the number of pending benefits.
func (h HelpBenefits) Len() int { return len(h.list) }

// Equal reports whether h and o hold the same benefits in the same order.
func (h HelpBenefits) Equal(o HelpBenefits) bool { return slices.Equal(h.list, o.list) }

// add returns h with b after the existing benefits.
func (h HelpBenefits) add(b HelpBenefit) HelpBenefits {
	return HelpBenefits{append(slices.Clone(h.list), b)}
}

// split returns the benefits for which take is true (removed, in order) and the others.
func (h HelpBenefits) split(take func(HelpBenefit) bool) (taken []HelpBenefit, kept HelpBenefits) {
	var left []HelpBenefit
	for _, b := range h.list {
		if take(b) {
			taken = append(taken, b)
		} else {
			left = append(left, b)
		}
	}
	return taken, HelpBenefits{left}
}

// AbilityCheckHelpOutcome is the closed set of outcomes of GiveAbilityCheckHelp. Variants:
// AbilityCheckHelpGiven, AbilityCheckHelpRefused.
type AbilityCheckHelpOutcome interface{ isAbilityCheckHelpOutcome() }

// AbilityCheckHelpGiven: the benefit was added. Read the new benefits and the one event.
type AbilityCheckHelpGiven struct {
	benefits HelpBenefits
	events   []HelpEvent
}

// AbilityCheckHelpRefused: no benefit was added and the benefits are unchanged.
type AbilityCheckHelpRefused struct {
	benefits HelpBenefits
	reason   AbilityCheckHelpRefusal
}

func (AbilityCheckHelpGiven) isAbilityCheckHelpOutcome()   {}
func (AbilityCheckHelpRefused) isAbilityCheckHelpOutcome() {}

// Benefits returns the new benefits.
func (r AbilityCheckHelpGiven) Benefits() HelpBenefits { return r.benefits }

// Events returns a copy of the events: exactly one HelpGiven.
func (r AbilityCheckHelpGiven) Events() []HelpEvent { return slices.Clone(r.events) }

// Benefits returns the benefits, unchanged.
func (r AbilityCheckHelpRefused) Benefits() HelpBenefits { return r.benefits }

// Reason returns why the help was refused.
func (r AbilityCheckHelpRefused) Reason() AbilityCheckHelpRefusal { return r.reason }

// Events returns a new, empty slice.
func (AbilityCheckHelpRefused) Events() []HelpEvent { return []HelpEvent{} }

// AttackRollHelpOutcome is the closed set of outcomes of GiveAttackRollHelp. Variants:
// AttackRollHelpGiven, AttackRollHelpRefused.
type AttackRollHelpOutcome interface{ isAttackRollHelpOutcome() }

// AttackRollHelpGiven: the benefit was added. Read the new benefits and the one event.
type AttackRollHelpGiven struct {
	benefits HelpBenefits
	events   []HelpEvent
}

// AttackRollHelpRefused: no benefit was added and the benefits are unchanged.
type AttackRollHelpRefused struct {
	benefits HelpBenefits
	reason   AttackRollHelpRefusal
}

func (AttackRollHelpGiven) isAttackRollHelpOutcome()   {}
func (AttackRollHelpRefused) isAttackRollHelpOutcome() {}

// Benefits returns the new benefits.
func (r AttackRollHelpGiven) Benefits() HelpBenefits { return r.benefits }

// Events returns a copy of the events: exactly one HelpGiven.
func (r AttackRollHelpGiven) Events() []HelpEvent { return slices.Clone(r.events) }

// Benefits returns the benefits, unchanged.
func (r AttackRollHelpRefused) Benefits() HelpBenefits { return r.benefits }

// Reason returns why the help was refused.
func (r AttackRollHelpRefused) Reason() AttackRollHelpRefusal { return r.reason }

// Events returns a new, empty slice.
func (AttackRollHelpRefused) Events() []HelpEvent { return []HelpEvent{} }

// GiveAbilityCheckHelp adds a benefit for the ally's next ability check with the
// proficiency (SRD 5.2.1, "Help [Action]", "Assist an Ability Check", p. 182). It returns
// AbilityCheckHelpGiven with a new HelpBenefits holding the benefit after the existing ones
// and one HelpGiven event, or AbilityCheckHelpRefused with the benefits unchanged. When
// several reasons apply the first of these wins: CannotHelpSelf (the helper is the ally, or
// either id is blank), HelperLacksProficiency (the helper's proficiencies do not hold the
// proficiency, an invalid one included), AssistanceNotPossible (the judgement is not
// possible, the zero judgement included). LimitExceeded (plan P10) comes last: the benefits already hold 256. It needs no enemy and never panics.
func (h HelpBenefits) GiveAbilityCheckHelp(helper, ally CombatantID, proficiency AssistedProficiency,
	helperProficiencies HelpGiverProficiencies, judgement AssistanceJudgement) AbilityCheckHelpOutcome {
	benefit, err := NewAbilityCheckHelp(helper, ally, proficiency)
	switch {
	case helper.IsZero() || ally.IsZero() || helper == ally:
		return AbilityCheckHelpRefused{h, AbilityCheckHelpRefusalCannotHelpSelf}
	case err != nil || !helperProficiencies.Has(proficiency):
		return AbilityCheckHelpRefused{h, AbilityCheckHelpRefusalHelperLacksProficiency}
	case judgement != AssistanceJudgementPossible:
		return AbilityCheckHelpRefused{h, AbilityCheckHelpRefusalAssistanceNotPossible}
	case len(h.list) >= maxInputList:
		return AbilityCheckHelpRefused{h, AbilityCheckHelpRefusalLimitExceeded}
	}
	return AbilityCheckHelpGiven{h.add(benefit), []HelpEvent{HelpGiven{mustHelpBase(benefit)}}}
}

// GiveAttackRollHelp adds a benefit for the next attack roll against the enemy by one of the
// helper's allies (SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183). It returns
// AttackRollHelpGiven with a new HelpBenefits and one HelpGiven event for
// AttackAssistTargetEnemyWithinFiveFeet, or AttackRollHelpRefused with the benefits
// unchanged. TargetNotAnEnemy wins when the enemy is the helper itself or either id is blank,
// or the target is not an enemy (the zero target included); otherwise EnemyNotWithinFiveFeet. LimitExceeded (plan P10) comes last: the benefits already hold 256.
// It needs no proficiencies and never panics.
func (h HelpBenefits) GiveAttackRollHelp(helper, enemy CombatantID, target AttackAssistTarget) AttackRollHelpOutcome {
	benefit, err := NewAttackRollHelp(helper, enemy)
	switch {
	case err != nil || (target != AttackAssistTargetEnemyWithinFiveFeet && target != AttackAssistTargetEnemyBeyondFiveFeet):
		return AttackRollHelpRefused{h, AttackRollHelpRefusalTargetNotAnEnemy}
	case target == AttackAssistTargetEnemyBeyondFiveFeet:
		return AttackRollHelpRefused{h, AttackRollHelpRefusalEnemyNotWithinFiveFeet}
	case len(h.list) >= maxInputList:
		return AttackRollHelpRefused{h, AttackRollHelpRefusalLimitExceeded}
	}
	return AttackRollHelpGiven{h.add(benefit), []HelpEvent{HelpGiven{mustHelpBase(benefit)}}}
}

// mustHelpBase builds the base of an event for a benefit already known valid.
func mustHelpBase(b HelpBenefit) helpEventBase {
	ids := b.Combatants()
	return helpEventBase{benefit: b, first: ids[0], second: ids[1]}
}

// AbilityCheckHelpUseOutcome is the closed set of outcomes of ConsumeForAbilityCheck and
// ConsumeForAbilityCheckUsing. Variants: AbilityCheckHelpUsed, NoAbilityCheckHelp.
type AbilityCheckHelpUseOutcome interface{ isAbilityCheckHelpUseOutcome() }

// AbilityCheckHelpUsed: at least one benefit was used up. Read the new benefits, the one
// Advantage circumstance and one HelpBenefitUsed event for each benefit used.
type AbilityCheckHelpUsed struct {
	benefits      HelpBenefits
	circumstances []modifier.RollCircumstance
	events        []HelpEvent
}

// NoAbilityCheckHelp: no pending benefit applies and the benefits are unchanged.
type NoAbilityCheckHelp struct{ benefits HelpBenefits }

func (AbilityCheckHelpUsed) isAbilityCheckHelpUseOutcome() {}
func (NoAbilityCheckHelp) isAbilityCheckHelpUseOutcome()   {}

// Benefits returns the benefits left.
func (r AbilityCheckHelpUsed) Benefits() HelpBenefits { return r.benefits }

// Circumstances returns a copy of the circumstances: exactly one Advantage.
func (r AbilityCheckHelpUsed) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(r.circumstances)
}

// Events returns a copy of the events, in the order the benefits were given.
func (r AbilityCheckHelpUsed) Events() []HelpEvent { return slices.Clone(r.events) }

// Benefits returns the benefits, unchanged.
func (r NoAbilityCheckHelp) Benefits() HelpBenefits { return r.benefits }

// Circumstances returns a new, empty slice.
func (NoAbilityCheckHelp) Circumstances() []modifier.RollCircumstance {
	return []modifier.RollCircumstance{}
}

// Events returns a new, empty slice.
func (NoAbilityCheckHelp) Events() []HelpEvent { return []HelpEvent{} }

// AttackRollHelpUseOutcome is the closed set of outcomes of ConsumeForAttackRoll. Variants:
// AttackRollHelpUsed, NoAttackRollHelp.
type AttackRollHelpUseOutcome interface{ isAttackRollHelpUseOutcome() }

// AttackRollHelpUsed: at least one benefit was used up. Read the new benefits, the one
// Advantage circumstance and one HelpBenefitUsed event for each benefit used.
type AttackRollHelpUsed struct {
	benefits      HelpBenefits
	circumstances []modifier.RollCircumstance
	events        []HelpEvent
}

// NoAttackRollHelp: no pending benefit applies and the benefits are unchanged.
type NoAttackRollHelp struct{ benefits HelpBenefits }

func (AttackRollHelpUsed) isAttackRollHelpUseOutcome() {}
func (NoAttackRollHelp) isAttackRollHelpUseOutcome()   {}

// Benefits returns the benefits left.
func (r AttackRollHelpUsed) Benefits() HelpBenefits { return r.benefits }

// Circumstances returns a copy of the circumstances: exactly one Advantage.
func (r AttackRollHelpUsed) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(r.circumstances)
}

// Events returns a copy of the events, in the order the benefits were given.
func (r AttackRollHelpUsed) Events() []HelpEvent { return slices.Clone(r.events) }

// Benefits returns the benefits, unchanged.
func (r NoAttackRollHelp) Benefits() HelpBenefits { return r.benefits }

// Circumstances returns a new, empty slice.
func (NoAttackRollHelp) Circumstances() []modifier.RollCircumstance {
	return []modifier.RollCircumstance{}
}

// Events returns a new, empty slice.
func (NoAttackRollHelp) Events() []HelpEvent { return []HelpEvent{} }

// usedEvents builds one HelpBenefitUsed for each benefit, rolled by the roller.
func usedEvents(taken []HelpBenefit, roller CombatantID) []HelpEvent {
	events := make([]HelpEvent, 0, len(taken))
	for _, b := range taken {
		e := HelpBenefitUsed{mustHelpBase(b)}
		if _, isRoll := b.(AttackRollHelp); isRoll {
			e.second, e.third = roller, b.Combatants()[1]
		}
		events = append(events, e)
	}
	return events
}

// ConsumeForAbilityCheck uses the pending benefits for the checker's ability check with one
// proficiency. It gives the same result as ConsumeForAbilityCheckUsing with that one
// proficiency.
func (h HelpBenefits) ConsumeForAbilityCheck(checker CombatantID, proficiency AssistedProficiency) AbilityCheckHelpUseOutcome {
	return h.ConsumeForAbilityCheckUsing(checker, []AssistedProficiency{proficiency})
}

// ConsumeForAbilityCheckUsing uses the pending benefits for the checker's ability check that
// uses every proficiency in proficiencies, a skill, a tool or both (SRD 5.2.1, "Equipment
// Proficiencies", Tools, p. 9; "Help [Action]", p. 182). A benefit matches when it names the
// checker as ally and its proficiency is in the slice, read as a set. Every matching benefit
// is used up (SRD 5.2.1, "They Don't Stack", p. 8) and the result holds exactly one
// Advantage and one HelpBenefitUsed for each benefit, in the order they were given. It
// returns NoAbilityCheckHelp with the benefits unchanged when nothing matches, an empty or
// nil slice included. The receiver is not changed. It never panics.
func (h HelpBenefits) ConsumeForAbilityCheckUsing(checker CombatantID, proficiencies []AssistedProficiency) AbilityCheckHelpUseOutcome {
	taken, kept := h.split(func(b HelpBenefit) bool {
		a, ok := b.(AbilityCheckHelp)
		return ok && a.ally == checker && slices.Contains(proficiencies, a.proficiency)
	})
	if len(taken) == 0 {
		return NoAbilityCheckHelp{h}
	}
	return AbilityCheckHelpUsed{kept, []modifier.RollCircumstance{modifier.Advantage{}}, usedEvents(taken, checker)}
}

// ConsumeForAttackRoll uses the pending benefits for the attacker's next attack roll against
// the target (SRD 5.2.1, "Help [Action]", "Assist an Attack Roll", p. 183). A benefit matches
// when it names the target as enemy, its helper is in allied (the helpers the host says are
// the attacker's allies) and its helper is not the attacker. An attacker that is the target
// uses nothing. Every matching benefit is used up, whether or not the roll ends with
// Advantage, Disadvantage or neither, and the result holds exactly one Advantage circumstance
// and does not resolve a roll mode. It returns NoAttackRollHelp with the benefits unchanged
// when nothing matches. The receiver is not changed. It never panics.
func (h HelpBenefits) ConsumeForAttackRoll(attacker, target CombatantID, allied AlliedHelpers) AttackRollHelpUseOutcome {
	if attacker == target {
		return NoAttackRollHelp{h}
	}
	taken, kept := h.split(func(b HelpBenefit) bool {
		a, ok := b.(AttackRollHelp)
		return ok && a.enemy == target && a.helperID != attacker && allied.Contains(a.helperID)
	})
	if len(taken) == 0 {
		return NoAttackRollHelp{h}
	}
	return AttackRollHelpUsed{kept, []modifier.RollCircumstance{modifier.Advantage{}}, usedEvents(taken, attacker)}
}

// HelpBenefitsExpiry is the result of ExpireAtStartOfTurn: the benefits left and a
// HelpBenefitExpired event for each benefit removed, in the order they were given. It is
// always a record, so a caller never branches on a nil value. The zero value is an empty
// result. It holds a slice, so compare with Equal, not ==; accessors return copies.
type HelpBenefitsExpiry struct {
	benefits HelpBenefits
	events   []HelpEvent
}

// Benefits returns the benefits left.
func (r HelpBenefitsExpiry) Benefits() HelpBenefits { return r.benefits }

// Events returns a copy of the HelpBenefitExpired events.
func (r HelpBenefitsExpiry) Events() []HelpEvent { return append([]HelpEvent{}, r.events...) }

// Equal reports whether r and o hold equal benefits and the same events in the same order.
func (r HelpBenefitsExpiry) Equal(o HelpBenefitsExpiry) bool {
	return r.benefits.Equal(o.benefits) && slices.Equal(r.events, o.events)
}

// HelpBenefitsDrop is the result of DropFor and DropAll: the benefits left and a
// HelpBenefitDropped event for each benefit removed, in the order they were given. It is
// always a record. The zero value is an empty result. It holds a slice, so compare with
// Equal, not ==; accessors return copies.
type HelpBenefitsDrop struct {
	benefits HelpBenefits
	events   []HelpEvent
}

// Benefits returns the benefits left.
func (r HelpBenefitsDrop) Benefits() HelpBenefits { return r.benefits }

// Events returns a copy of the HelpBenefitDropped events.
func (r HelpBenefitsDrop) Events() []HelpEvent { return append([]HelpEvent{}, r.events...) }

// Equal reports whether r and o hold equal benefits and the same events in the same order.
func (r HelpBenefitsDrop) Equal(o HelpBenefitsDrop) bool {
	return r.benefits.Equal(o.benefits) && slices.Equal(r.events, o.events)
}

// remove removes the benefits for which take is true, with event building one event each.
func (h HelpBenefits) remove(take func(HelpBenefit) bool, event func(helpEventBase) HelpEvent) (HelpBenefits, []HelpEvent) {
	taken, kept := h.split(take)
	if len(taken) == 0 {
		return h, nil
	}
	events := make([]HelpEvent, 0, len(taken))
	for _, b := range taken {
		events = append(events, event(mustHelpBase(b)))
	}
	return kept, events
}

// ExpireAtStartOfTurn removes every benefit whose helper is the combatant, whose turn is
// starting, and returns the benefits left with one HelpBenefitExpired for each (SRD 5.2.1,
// "Help [Action]", pp. 182-183). The start of any other combatant's turn, the ally's included,
// expires nothing. It always returns a HelpBenefitsExpiry: with nothing removed, its benefits
// equal the receiver and its events are empty. The receiver is not changed.
func (h HelpBenefits) ExpireAtStartOfTurn(combatant CombatantID) HelpBenefitsExpiry {
	kept, events := h.remove(func(b HelpBenefit) bool { return b.Combatants()[0] == combatant },
		func(base helpEventBase) HelpEvent { return HelpBenefitExpired{base} })
	return HelpBenefitsExpiry{kept, events}
}

// DropFor removes every benefit that names the combatant as helper, ally or enemy, for a
// combatant that left the encounter, and returns the benefits left with one
// HelpBenefitDropped for each. It always returns a HelpBenefitsDrop: with nothing removed, its
// benefits equal the receiver and its events are empty. The receiver is not changed.
func (h HelpBenefits) DropFor(combatant CombatantID) HelpBenefitsDrop {
	kept, events := h.remove(func(b HelpBenefit) bool { return slices.Contains(b.Combatants(), combatant) },
		func(base helpEventBase) HelpEvent { return HelpBenefitDropped{base} })
	return HelpBenefitsDrop{kept, events}
}

// DropAll removes every benefit, for the end of the encounter or the exploration, and returns
// no benefits with one HelpBenefitDropped for each, in the order they were given. It always
// returns a HelpBenefitsDrop; with nothing pending its events are empty. The receiver is not
// changed.
func (h HelpBenefits) DropAll() HelpBenefitsDrop {
	kept, events := h.remove(func(HelpBenefit) bool { return true },
		func(base helpEventBase) HelpEvent { return HelpBenefitDropped{base} })
	return HelpBenefitsDrop{kept, events}
}
