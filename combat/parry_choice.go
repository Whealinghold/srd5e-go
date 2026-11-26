package combat

// ParryChoice is the host's standing choice of whether the target of a melee attack uses its Parry
// (combat-action-resolution.md FR-161; SRD 5.2.1, "Bandit", p. 262). Variants: NoParry, ParryIfAble.
type ParryChoice interface{ isParryChoice() }

// NoParry: the target does not use Parry. The default; a nil ParryChoice reads as this.
type NoParry struct{}

// ParryIfAble: the target uses Parry when it can. Construct with a keyed literal; the Trait must not be the
// zero value. HoldingWeapon is the host's fact that the target holds a weapon.
type ParryIfAble struct {
	Trait         ParryTrait
	Reaction      FeatureReaction
	HoldingWeapon bool
}

func (NoParry) isParryChoice()     {}
func (ParryIfAble) isParryChoice() {}

// ParryOutcome is what became of the target's Parry for one attack (FR-161). Variants: ParryNotOffered,
// ParryNotUsed, ParryApplied.
type ParryOutcome interface{ isParryOutcome() }

// ParryNotOffered: the host chose NoParry.
type ParryNotOffered struct{}

// ParryNotUsed: Parry was wanted but not used, and the hit stands. Exactly one field is set: Unavailable
// (the zero value when the tracker refused) says why Parry could not answer the attack, Refused (nil when
// Parry was unavailable) is the tracker's refusal of the target's Reaction. Not comparable with ==.
type ParryNotUsed struct {
	Unavailable ParryUnavailableReason
	Refused     TurnRejection
}

// ParryApplied: the target took its Reaction and Parried; the response holds the attack roll resolved again
// and whether the hit became a miss. Not comparable with ==.
type ParryApplied struct{ Response ParryResponse }

func (ParryNotOffered) isParryOutcome() {}
func (ParryNotUsed) isParryOutcome()    {}
func (ParryApplied) isParryOutcome()    {}

// validParryChoice returns an error matching srderr.ErrInvalidInput for a ParryIfAble with the zero trait or
// a choice that is not defined; nil and NoParry are valid.
func validParryChoice(c ParryChoice) error {
	switch v := c.(type) {
	case nil, NoParry:
		return nil
	case ParryIfAble:
		if v.Trait.IsZero() {
			return invalidInput("Parry needs its trait")
		}
		return nil
	}
	return invalidInput("a Parry choice is NoParry or ParryIfAble")
}
