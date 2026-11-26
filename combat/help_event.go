package combat

// HelpEvent is the closed family of events the Help action records. Each is an immutable
// value comparable with ==. Variants: HelpGiven, HelpBenefitUsed, HelpBenefitExpired,
// HelpBenefitDropped. Each embeds helpEventBase.
type HelpEvent interface {
	Event
	helpEvent()
}

// helpEventBase carries the benefit an event is about and the creatures it names, and the
// markers that put it in the Event and HelpEvent families. Its combatants are the helper,
// then the second creature, then the third when there is one.
type helpEventBase struct {
	benefit              HelpBenefit
	first, second, third CombatantID
}

func (helpEventBase) event()     {}
func (helpEventBase) helpEvent() {}

// Combatants returns a new slice: the helper, then the ally or enemy, and for a used attack
// roll benefit the helper, the attacker, then the enemy.
func (b helpEventBase) Combatants() []CombatantID {
	ids := []CombatantID{b.first, b.second}
	if !b.third.IsZero() {
		ids = append(ids, b.third)
	}
	return ids
}

// Benefit returns the benefit the event is about.
func (b helpEventBase) Benefit() HelpBenefit { return b.benefit }

func newHelpBase(b HelpBenefit) (helpEventBase, error) {
	if !validBenefit(b) {
		return helpEventBase{}, invalidInput("help event needs a valid benefit")
	}
	ids := b.Combatants()
	return helpEventBase{benefit: b, first: ids[0], second: ids[1]}, nil
}

// HelpGiven: a creature took the Help action and left a benefit. Its Kind is help_given.
type HelpGiven struct{ helpEventBase }

// NewHelpGiven returns the event. It returns an error matching srderr.ErrInvalidInput for a
// nil or invalid benefit. It never panics.
func NewHelpGiven(b HelpBenefit) (HelpGiven, error) {
	base, err := newHelpBase(b)
	if err != nil {
		return HelpGiven{}, err
	}
	return HelpGiven{base}, nil
}

// Kind returns "help_given".
func (HelpGiven) Kind() string { return "help_given" }

// HelpBenefitUsed: a pending benefit was used up by a roll. Its Kind is help_benefit_used.
type HelpBenefitUsed struct{ helpEventBase }

// NewHelpBenefitUsed returns the event for the benefit used by the creature that rolled. For
// ability check help the roller must be the ally; for attack roll help the roller is the
// attacker and must be neither the helper nor the enemy. It returns an error matching
// srderr.ErrInvalidInput for a nil or invalid benefit or a roller that does not fit. It never
// panics.
func NewHelpBenefitUsed(b HelpBenefit, roller CombatantID) (HelpBenefitUsed, error) {
	base, err := newHelpBase(b)
	if err != nil {
		return HelpBenefitUsed{}, err
	}
	switch b := b.(type) {
	case AbilityCheckHelp:
		if roller != b.ally {
			return HelpBenefitUsed{}, invalidInput("ability check help is used by the ally")
		}
	case AttackRollHelp:
		if roller.IsZero() || roller == b.helperID || roller == b.enemy {
			return HelpBenefitUsed{}, invalidInput("attack roll help is used by an attacker other than the helper and the enemy")
		}
		base.second, base.third = roller, b.enemy
	}
	return HelpBenefitUsed{base}, nil
}

// Kind returns "help_benefit_used".
func (HelpBenefitUsed) Kind() string { return "help_benefit_used" }

// HelpBenefitExpired: a pending benefit expired at the start of the helper's next turn. Its
// Kind is help_benefit_expired.
type HelpBenefitExpired struct{ helpEventBase }

// NewHelpBenefitExpired returns the event. It returns an error matching
// srderr.ErrInvalidInput for a nil or invalid benefit. It never panics.
func NewHelpBenefitExpired(b HelpBenefit) (HelpBenefitExpired, error) {
	base, err := newHelpBase(b)
	if err != nil {
		return HelpBenefitExpired{}, err
	}
	return HelpBenefitExpired{base}, nil
}

// Kind returns "help_benefit_expired".
func (HelpBenefitExpired) Kind() string { return "help_benefit_expired" }

// HelpBenefitDropped: a pending benefit was dropped because a creature it names left the
// encounter or the encounter ended. Its Kind is help_benefit_dropped.
type HelpBenefitDropped struct{ helpEventBase }

// NewHelpBenefitDropped returns the event. It returns an error matching
// srderr.ErrInvalidInput for a nil or invalid benefit. It never panics.
func NewHelpBenefitDropped(b HelpBenefit) (HelpBenefitDropped, error) {
	base, err := newHelpBase(b)
	if err != nil {
		return HelpBenefitDropped{}, err
	}
	return HelpBenefitDropped{base}, nil
}

// Kind returns "help_benefit_dropped".
func (HelpBenefitDropped) Kind() string { return "help_benefit_dropped" }
