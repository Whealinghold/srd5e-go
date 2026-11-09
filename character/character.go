package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Character is the aggregate root of the character context: the identity and
// advancement of one player character. It holds what the rules tie to identity
// (ability scores, Experience Points, the ordered class levels, the stored Hit
// Point maximum and Hit Dice, Heroic Inspiration and the feature states) and
// nothing class-specific. Species and background are held by id only.
//
// A Character is immutable: every transition returns a new one and leaves the
// receiver as it was. It holds slices and a map internally, so compare two with
// Equal, not ==. The zero Character is invalid; see IsZero. Build one with New
// or Restore. Safe for concurrent use.
type Character struct {
	id          Identifier
	xp          ExperiencePoints
	classes     ClassLevels
	species     content.SpeciesID
	background  content.BackgroundID
	scores      ability.Scores
	hitPoints   HitPointMaximum
	pool        HitDicePool
	inspiration HeroicInspiration
	states      FeatureStates
}

// ID returns the Character's id.
func (c Character) ID() Identifier { return c.id }

// ExperiencePoints returns the Experience Points total.
func (c Character) ExperiencePoints() ExperiencePoints { return c.xp }

// ClassLevels returns the classes and levels, the initial class first.
func (c Character) ClassLevels() ClassLevels { return c.classes }

// InitialClass returns the first class the Character took.
func (c Character) InitialClass() content.ClassID { return c.classes.Initial() }

// Species returns the species id.
func (c Character) Species() content.SpeciesID { return c.species }

// Background returns the background id.
func (c Character) Background() content.BackgroundID { return c.background }

// AbilityScores returns the ability scores.
func (c Character) AbilityScores() ability.Scores { return c.scores }

// HitPointMaximum returns the stored Hit Point maximum, without any bonus.
func (c Character) HitPointMaximum() HitPointMaximum { return c.hitPoints }

// HitDicePool returns the Hit Dice the Character has.
func (c Character) HitDicePool() HitDicePool { return c.pool }

// HeroicInspiration returns whether the Character is Inspired.
func (c Character) HeroicInspiration() HeroicInspiration { return c.inspiration }

// FeatureStates returns the stored feature states.
func (c Character) FeatureStates() FeatureStates { return c.states }

// TotalLevel returns the sum of the class levels, 1 to 20.
func (c Character) TotalLevel() Level { return c.classes.TotalLevel() }

// ProficiencyBonus returns the Proficiency Bonus of the total level, not of a
// single class (SRD 5.2.1, "Multiclassing", p. 25).
func (c Character) ProficiencyBonus() ProficiencyBonus { return ProficiencyBonusFor(c.TotalLevel()) }

// IsEligibleForNextLevel reports whether the total level is below 20 and the
// Experience Points reach the minimum for the next level (SRD 5.2.1, "Level
// Advancement", p. 23).
func (c Character) IsEligibleForNextLevel() bool {
	total := c.TotalLevel()
	if total.n >= MaxLevel {
		return false
	}
	return c.xp.n >= c.xp.MinimumForLevel(Level{n: total.n + 1}).n
}

// IsZero reports whether c is the invalid zero Character.
func (c Character) IsZero() bool { return c.id.IsZero() }

// Equal reports whether c and o are equal by value: class levels in order,
// feature states and Heroic Inspiration included.
func (c Character) Equal(o Character) bool {
	return c.id == o.id && c.xp == o.xp && c.classes.Equal(o.classes) && c.species == o.species &&
		c.background == o.background && c.scores == o.scores && c.hitPoints == o.hitPoints &&
		c.pool.Equal(o.pool) && c.inspiration == o.inspiration && c.states.Equal(o.states)
}

// Parts are the stored values of a Character, for hosts that restore one they
// saved. Construct it with a keyed literal.
type Parts struct {
	ID                Identifier
	ExperiencePoints  ExperiencePoints
	ClassLevels       ClassLevels
	Species           content.SpeciesID
	Background        content.BackgroundID
	AbilityScores     ability.Scores
	HitPointMaximum   HitPointMaximum
	HitDicePool       HitDicePool
	HeroicInspiration HeroicInspiration
	FeatureStates     FeatureStates
}

// Restore returns the Character holding parts. It returns an error matching
// srderr.ErrInvalidInput for a zero id, empty class levels, an invalid species,
// background, ability scores or Heroic Inspiration, a Hit Dice pool whose size
// is not the total level, or a Hit Point maximum below the total level (SRD
// 5.2.1, "Gaining a Level", p. 23). It never panics.
func Restore(p Parts) (Character, error) {
	switch {
	case p.ID.IsZero():
		return Character{}, invalidInput("character without an id")
	case p.ClassLevels.IsZero():
		return Character{}, invalidInput("character without a class")
	case !p.Species.Valid(), !p.Background.Valid():
		return Character{}, invalidInput("character with an invalid species or background")
	case p.AbilityScores.IsZero():
		return Character{}, invalidInput("character without ability scores")
	case !p.HeroicInspiration.Valid():
		return Character{}, invalidInput("character with an invalid Heroic Inspiration")
	case p.HitDicePool.Total() != p.ClassLevels.TotalLevel().n:
		return Character{}, invalidInput("Hit Dice pool size is not the total level")
	case p.HitPointMaximum.Value() < p.ClassLevels.TotalLevel().n:
		return Character{}, invalidInput("Hit Point maximum below the total level")
	}
	return Character{
		id: p.ID, xp: p.ExperiencePoints, classes: p.ClassLevels, species: p.Species, background: p.Background,
		scores: p.AbilityScores, hitPoints: p.HitPointMaximum, pool: p.HitDicePool,
		inspiration: p.HeroicInspiration, states: p.FeatureStates,
	}, nil
}
