package character

// ExcessInspirationChoice is what the host decides when Heroic Inspiration is
// gained by a character who already has it: lose the new one, or give it to a
// player character in the group who lacks it (SRD 5.2.1, "Rules Glossary",
// "Heroic Inspiration", p. 183). It is a closed set. Variants:
// LoseExcessInspiration, GiveExcessInspirationTo. The choice is read only when
// the holder is already Inspired; any value other than GiveExcessInspirationTo,
// including nil, is a loss.
type ExcessInspirationChoice interface{ isExcessInspirationChoice() }

// LoseExcessInspiration is the choice to lose the excess instance.
type LoseExcessInspiration struct{}

// GiveExcessInspirationTo is the choice to give the excess instance to a
// player character in the group. Build it with a keyed literal; the zero
// Recipient is refused by GainHeroicInspiration as invalid input.
type GiveExcessInspirationTo struct{ Recipient Character }

func (LoseExcessInspiration) isExcessInspirationChoice()   {}
func (GiveExcessInspirationTo) isExcessInspirationChoice() {}
