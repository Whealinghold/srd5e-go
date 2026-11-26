package combat

import (
	"time"

	"github.com/Whealinghold/srd5e-go/content"
)

// RevivalSpell is one of the four spells that return a Dead creature to life
// (SRD 5.2.1, "Revivify", p. 159; "Raise Dead", p. 157; "Resurrection", pp.
// 158-159; "True Resurrection", p. 170). The zero value is not a revival spell.
type RevivalSpell int

// The revival spells.
const (
	// Revivify returns a creature that died within the last minute with 1 Hit Point.
	Revivify RevivalSpell = iota + 1
	// RaiseDead returns a creature dead for up to 10 days with 1 Hit Point and an ordeal.
	RaiseDead
	// Resurrection returns a creature dead for up to a century with all its Hit Points.
	Resurrection
	// TrueResurrection returns a creature dead for up to 200 years with all its Hit Points.
	TrueResurrection
)

// String returns "revivify", "raise_dead", "resurrection", "true_resurrection" or "unknown".
func (s RevivalSpell) String() string {
	return enumName([]string{"revivify", "raise_dead", "resurrection", "true_resurrection"}, int(s))
}

// Valid reports whether s is one of the four revival spells.
func (s RevivalSpell) Valid() bool { return s >= Revivify && s <= TrueResurrection }

// RevivalSpellOf maps a spell id to its revival spell: revivify, raise_dead,
// resurrection and true_resurrection give true; every other id (reincarnate,
// revive, an unknown id) gives the zero value and false. It never panics.
func RevivalSpellOf(id content.SpellID) (RevivalSpell, bool) {
	switch id {
	case "revivify":
		return Revivify, true
	case "raise_dead":
		return RaiseDead, true
	case "resurrection":
		return Resurrection, true
	case "true_resurrection":
		return TrueResurrection, true
	}
	return 0, false
}

// RevivalSpellFacts are the static facts of one revival spell (table 4.1a of
// revival-and-restoration-spells.md). The zero value is the facts of no spell.
// Immutable and comparable.
type RevivalSpellFacts struct {
	window                          time.Duration
	oldAgeBlocks, undeadBlocks      bool
	allHitPoints, neutralisesPoison bool
	liftsCurses, ordeal             bool
	strainAfter                     time.Duration
	integralPartsBlock, nameForBody bool
}

// Facts returns the facts of s; the zero RevivalSpellFacts for an invalid s.
func (s RevivalSpell) Facts() RevivalSpellFacts {
	const day = 24 * time.Hour
	switch s {
	case Revivify:
		return RevivalSpellFacts{window: time.Minute, oldAgeBlocks: true}
	case RaiseDead:
		return RevivalSpellFacts{window: 10 * day, undeadBlocks: true, neutralisesPoison: true, ordeal: true, integralPartsBlock: true}
	case Resurrection:
		return RevivalSpellFacts{window: 36500 * day, oldAgeBlocks: true, undeadBlocks: true, allHitPoints: true,
			neutralisesPoison: true, ordeal: true, strainAfter: 365 * day}
	case TrueResurrection:
		return RevivalSpellFacts{window: 73000 * day, oldAgeBlocks: true, allHitPoints: true, neutralisesPoison: true,
			liftsCurses: true, nameForBody: true}
	}
	return RevivalSpellFacts{}
}

// Window returns how long a creature may have been dead, inclusive (a year is 365 days).
func (f RevivalSpellFacts) Window() time.Duration { return f.window }

// OldAgeBlocks reports whether a creature that died of old age cannot be returned.
func (f RevivalSpellFacts) OldAgeBlocks() bool { return f.oldAgeBlocks }

// UndeadBlocks reports whether a creature that was Undead at death cannot be returned.
func (f RevivalSpellFacts) UndeadBlocks() bool { return f.undeadBlocks }

// ReturnsAllHitPoints reports whether the creature returns with all its Hit Points
// rather than 1.
func (f RevivalSpellFacts) ReturnsAllHitPoints() bool { return f.allHitPoints }

// NeutralisesPoison reports whether the spell neutralises the poisons that affected the creature.
func (f RevivalSpellFacts) NeutralisesPoison() bool { return f.neutralisesPoison }

// LiftsCurses reports whether the spell lifts every curse and magical contagion.
func (f RevivalSpellFacts) LiftsCurses() bool { return f.liftsCurses }

// GivesOrdeal reports whether a successful cast gives the fresh ordeal of -4 to D20 Tests.
func (f RevivalSpellFacts) GivesOrdeal() bool { return f.ordeal }

// StrainAfter returns how long a creature must have been dead for the cast to tax
// the caster (inclusive), or 0 when the spell never does.
func (f RevivalSpellFacts) StrainAfter() time.Duration { return f.strainAfter }

// IntegralPartsBlock reports whether a body missing parts integral to its survival
// makes the spell fail.
func (f RevivalSpellFacts) IntegralPartsBlock() bool { return f.integralPartsBlock }

// NameForBody reports whether the caster can speak the creature's name to give it
// a new body when none remains.
func (f RevivalSpellFacts) NameForBody() bool { return f.nameForBody }
