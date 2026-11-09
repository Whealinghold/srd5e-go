package character

// HeroicInspiration is whether a character holds Heroic Inspiration (SRD 5.2.1,
// "Heroic Inspiration", p. 8; Rules Glossary, p. 183). A character never holds
// more than one instance, so it is a two-value enum. The zero value is
// invalid. Comparable and safe for concurrent use.
type HeroicInspiration int

// The Heroic Inspiration states.
const (
	// Inspired: the character has Heroic Inspiration.
	Inspired HeroicInspiration = iota + 1
	// NotInspired: the character has none.
	NotInspired
)

// Valid reports whether h is Inspired or NotInspired.
func (h HeroicInspiration) Valid() bool { return h == Inspired || h == NotInspired }

// String returns "inspired", "notInspired" or "unknown".
func (h HeroicInspiration) String() string {
	switch h {
	case Inspired:
		return "inspired"
	case NotInspired:
		return "notInspired"
	}
	return "unknown"
}

// HeroicInspirationRejection is the closed set of reasons a Heroic
// Inspiration transition is refused. Variants: HeroicInspirationNotAvailable.
type HeroicInspirationRejection interface{ isHeroicInspirationRejection() }

// HeroicInspirationNotAvailable is the rejection of expending Heroic
// Inspiration the character does not have.
type HeroicInspirationNotAvailable struct{}

func (HeroicInspirationNotAvailable) isHeroicInspirationRejection() {}

// GrantHeroicInspiration returns the character with Heroic Inspiration and one
// HeroicInspirationGained event. A character already Inspired comes back equal
// with no events: the excess instance is lost (SRD 5.2.1, Rules Glossary, p.
// 183). The receiver is unchanged.
func (c Character) GrantHeroicInspiration() TransitionChanged[NoRejection] {
	if c.inspiration == Inspired {
		return changed[NoRejection](c)
	}
	next := c
	next.inspiration = Inspired
	return changed[NoRejection](next, HeroicInspirationGained{id: c.id})
}

// ExpendHeroicInspiration returns the character without Heroic Inspiration and
// one HeroicInspirationExpended event, or a TransitionRejected holding
// HeroicInspirationNotAvailable when the character is not Inspired (SRD 5.2.1,
// "Heroic Inspiration", p. 8). The receiver is unchanged.
func (c Character) ExpendHeroicInspiration() Transition[HeroicInspirationRejection] {
	if c.inspiration != Inspired {
		return rejected[HeroicInspirationRejection](HeroicInspirationNotAvailable{})
	}
	next := c
	next.inspiration = NotInspired
	return changed[HeroicInspirationRejection](next, HeroicInspirationExpended{id: c.id})
}
