package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellScrollOrigin is the closed set of origins of a Spell Scroll, which decide
// the save DC and attack bonus of its cast (SRD 5.2.1, "Spell Scroll", p. 244;
// "Scribing Spell Scrolls", p. 103). Variants: FoundScroll, ScribedScrollOrigin.
type SpellScrollOrigin interface{ isSpellScrollOrigin() }

// FoundScroll is a scroll whose scribe is unknown: its cast uses the Spell Scroll
// table for its spell level. The zero value is ready to use.
type FoundScroll struct{}

// ScribedScrollOrigin is a scroll with a known scribe: its cast uses the scribe's
// numbers, fixed when the scroll was completed. Build it with NewScribedScrollOrigin.
// The zero ScribedScrollOrigin is invalid. Comparable.
type ScribedScrollOrigin struct{ stats ScribedScrollStats }

func (FoundScroll) isSpellScrollOrigin()         {}
func (ScribedScrollOrigin) isSpellScrollOrigin() {}

// NewScribedScrollOrigin returns the origin of a scroll scribed with stats. It returns
// an error matching srderr.ErrInvalidInput for the zero stats. It never panics.
func NewScribedScrollOrigin(stats ScribedScrollStats) (ScribedScrollOrigin, error) {
	if stats.level.IsZero() {
		return ScribedScrollOrigin{}, invalidInput("scribed scroll: zero stats")
	}
	return ScribedScrollOrigin{stats: stats}, nil
}

// Stats returns the scribe's numbers.
func (s ScribedScrollOrigin) Stats() ScribedScrollStats { return s.stats }

// ScribedScrollStats are the spell save DC, spell attack bonus and character
// level of the scribe of a scroll, as they were when the scroll was completed
// (SRD 5.2.1, "Scribing Spell Scrolls", p. 103). Comparable. The zero value is
// invalid.
type ScribedScrollStats struct {
	saveDC      int
	attackBonus int
	level       Level
}

// NewScribedScrollStats takes the DC and attack bonus of stats, the casting
// class that prepared the spell, and the scribe's character level, and keeps
// them fixed on the scroll: later level-ups of the scribe do not change them.
// It returns an error matching srderr.ErrInvalidInput for the zero stats or the
// zero level. It never panics.
func NewScribedScrollStats(stats SpellcastingStats, level Level) (ScribedScrollStats, error) {
	if !stats.Ability().Valid() || level.IsZero() {
		return ScribedScrollStats{}, invalidInput("scribed scroll stats: zero stats or level")
	}
	return ScribedScrollStats{saveDC: stats.SaveDC().Value(), attackBonus: stats.AttackBonus().Value(), level: level}, nil
}

// SaveDC returns the scribe's spell save DC.
func (s ScribedScrollStats) SaveDC() int { return s.saveDC }

// AttackBonus returns the scribe's spell attack bonus.
func (s ScribedScrollStats) AttackBonus() int { return s.attackBonus }

// CharacterLevel returns the scribe's character level when the scroll was completed.
func (s ScribedScrollStats) CharacterLevel() Level { return s.level }

// SpellScroll is a Spell Scroll holding one spell of level 0 to 9 (SRD 5.2.1,
// "Spell Scroll", p. 244). Build it with NewSpellScroll. Immutable; the zero
// SpellScroll is invalid.
type SpellScroll struct {
	spell   content.SpellID
	level   content.SpellLevel
	classes []content.ClassID
	origin  SpellScrollOrigin
}

// NewSpellScroll returns the scroll of spell with origin. It returns an error
// matching srderr.ErrInvalidInput for a nil origin or a zero ScribedScrollOrigin. It
// never panics.
func NewSpellScroll(spell content.Spell, origin SpellScrollOrigin) (SpellScroll, error) {
	switch o := origin.(type) {
	case FoundScroll:
	case ScribedScrollOrigin:
		if o.stats.level.IsZero() {
			return SpellScroll{}, invalidInput("spell scroll: zero scribed origin")
		}
	default:
		return SpellScroll{}, invalidInput("spell scroll: nil origin")
	}
	return SpellScroll{spell: spell.ID(), level: spell.Level(), classes: spell.Classes(), origin: origin}, nil
}

// SpellID returns the spell on the scroll.
func (s SpellScroll) SpellID() content.SpellID { return s.spell }

// SpellLevel returns the level of the spell, 0 for a cantrip.
func (s SpellScroll) SpellLevel() content.SpellLevel { return s.level }

// Classes returns a new slice of the classes whose spell lists hold the spell.
func (s SpellScroll) Classes() []content.ClassID { return slices.Clone(s.classes) }

// Origin returns whether the scroll was found or scribed.
func (s SpellScroll) Origin() SpellScrollOrigin { return s.origin }

// ScrollCasterLevel is the closed set of caster levels of a scroll cast. Variants:
// ScrollLowestCasterLevel, ScribedCantripCasterLevel.
type ScrollCasterLevel interface{ isScrollCasterLevel() }

// ScrollLowestCasterLevel is the lowest possible spell and caster level of a spell
// cast from an item (SRD 5.2.1, "Spells Cast from Items", p. 206). The zero value
// is ready to use.
type ScrollLowestCasterLevel struct{}

// Level returns ItemCasterLevelLowest.
func (ScrollLowestCasterLevel) Level() ItemCasterLevel { return ItemCasterLevelLowest }

// ScribedCantripCasterLevel is the caster level of a scribed cantrip: the scribe's
// character level when the scroll was completed (SRD 5.2.1, "Scribing Spell
// Scrolls", "Cantrips", p. 103). Build it with NewScribedCantripCasterLevel. The
// zero value is invalid. Comparable.
type ScribedCantripCasterLevel struct{ level Level }

// NewScribedCantripCasterLevel returns the caster level of a cantrip scribed at level.
func NewScribedCantripCasterLevel(level Level) ScribedCantripCasterLevel {
	return ScribedCantripCasterLevel{level: level}
}

// Level returns the scribe's character level.
func (c ScribedCantripCasterLevel) Level() Level { return c.level }

func (ScrollLowestCasterLevel) isScrollCasterLevel()   {}
func (ScribedCantripCasterLevel) isScrollCasterLevel() {}
