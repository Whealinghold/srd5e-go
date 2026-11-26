package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// MonsterInitiativeMode says whether a monster's Initiative is rolled or fixed. The
// zero value is invalid; the setup default is MonsterInitiativeRolled.
type MonsterInitiativeMode int

// The two modes.
const (
	// MonsterInitiativeRolled: the host rolls Initiative with the printed modifier.
	MonsterInitiativeRolled MonsterInitiativeMode = iota + 1
	// MonsterInitiativeFixed: the host uses the printed Initiative score as the count.
	MonsterInitiativeFixed
)

// Valid reports whether m is one of the two modes.
func (m MonsterInitiativeMode) Valid() bool {
	return m == MonsterInitiativeRolled || m == MonsterInitiativeFixed
}

// String returns "rolled", "fixed" or "unknown".
func (m MonsterInitiativeMode) String() string { return enumName([]string{"rolled", "fixed"}, int(m)) }

// MonsterInitiativePlan is the Initiative data a setup hands to the host: it rolls no
// die (SRD 5.2.1, "Initiative", p. 254). Variants: RolledMonsterInitiative,
// FixedMonsterInitiative.
type MonsterInitiativePlan interface{ isMonsterInitiativePlan() }

// RolledMonsterInitiative is the printed Initiative modifier, not recomputed from
// Dexterity, for the host to pass to RollInitiative. Construct it with a keyed literal.
type RolledMonsterInitiative struct{ Modifier modifier.Modifier }

// FixedMonsterInitiative is the printed Initiative score as the count, never "10 plus
// the modifier", for the host to pass to EstablishInitiativeOrder. Construct it with a
// keyed literal.
type FixedMonsterInitiative struct{ Count InitiativeCount }

func (RolledMonsterInitiative) isMonsterInitiativePlan() {}
func (FixedMonsterInitiative) isMonsterInitiativePlan()  {}

// monsterInitiative returns the plan of m for mode.
func monsterInitiative(m content.Monster, mode MonsterInitiativeMode) MonsterInitiativePlan {
	if mode == MonsterInitiativeFixed {
		return FixedMonsterInitiative{InitiativeCount(m.Initiative().Score())}
	}
	return RolledMonsterInitiative{modifier.New(m.Initiative().Modifier())}
}

// UnappliedSpeed is a printed speed a MonsterSetup did not apply because it has a
// qualifier or several modes (SRD 5.2.1, "Speed", p. 255). It carries the content entry.
// Not comparable with ==; use Equal. Immutable.
type UnappliedSpeed struct{ entry content.SpeedEntry }

// Entry returns the printed speed.
func (s UnappliedSpeed) Entry() content.SpeedEntry { return s.entry }

// Equal reports whether s and o are the same printed speed.
func (s UnappliedSpeed) Equal(o UnappliedSpeed) bool { return s.entry.Equal(o.entry) }

// monsterSpeeds returns the speeds of m that apply (unqualified entries with one mode;
// the walking Speed is 0 when there is none) and the entries that do not, in printed
// order. The hover flag has no slot in MovementSpeeds and stays readable on the
// monster. content.SpeedMode and SpeedName list the modes in the same order.
func monsterSpeeds(m content.Monster) (MovementSpeeds, []UnappliedSpeed) {
	var s MovementSpeeds
	var unapplied []UnappliedSpeed
	for _, e := range m.Speeds() {
		modes := e.Modes()
		if e.Qualifier() != "" || len(modes) != 1 {
			unapplied = append(unapplied, UnappliedSpeed{e})
			continue
		}
		i := SpeedName(modes[0]) - 1
		s.feet[i], s.has[i] = e.Feet(), true
	}
	return s, unapplied
}

// printedModifiers returns the modifiers that make up a printed modifier: the ability
// modifier alone when they are equal, with the Proficiency Bonus when the difference is
// the bonus (or, when doubled is set, twice the bonus, the Expertise form), and the
// difference as another modifier otherwise. The printed value is authoritative ("Other
// modifiers might apply", SRD 5.2.1, "Ability Scores", p. 255), so the slots always sum
// to it. The difference must fit an int (see differenceFits).
func printedModifiers(abilityModifier, printed int, pb modifier.Modifier, doubled bool) d20.RollModifiers {
	r := d20.NewRollModifiers(modifier.New(abilityModifier))
	d := printed - abilityModifier
	switch {
	case d == 0:
		return r
	case d == pb.Value():
		return r.WithProficiencyBonus(pb)
	case doubled && d == 2*pb.Value():
		return r.WithProficiencyBonus(modifier.New(d))
	}
	return r.WithOthers(modifier.New(d))
}

// SavingThrow returns the modifiers of a saving throw of a: the ability modifier, the
// Proficiency Bonus when the printed save differs from the ability modifier by exactly
// that bonus, and any other difference as another modifier, so that the total is the
// printed save (SRD 5.2.1, "Ability Scores", pp. 255-256). It returns the zero
// RollModifiers for an invalid ability.
func (s MonsterSetup) SavingThrow(a ability.Ability) d20.RollModifiers {
	if !a.Valid() {
		return d20.RollModifiers{}
	}
	return printedModifiers(s.monster.AbilityModifier(a).Value(), s.monster.Abilities().Save(a), s.pb, false)
}

// SkillCheck returns the modifiers of a check of sk: for a skill the stat block lists,
// split as SavingThrow does, with twice the Proficiency Bonus put in its slot when the
// stat block prints that (Expertise, p. 182); for a skill it does not list, the ability
// modifier alone (SRD 5.2.1, "Skills", p. 255). It returns the zero RollModifiers for an
// invalid skill.
func (s MonsterSetup) SkillCheck(sk content.Skill) d20.RollModifiers {
	if !sk.Valid() {
		return d20.RollModifiers{}
	}
	am := s.monster.AbilityModifier(sk.Ability()).Value()
	if printed, listed := s.monster.Skills()[sk]; listed {
		return printedModifiers(am, printed, s.pb, true)
	}
	return d20.NewRollModifiers(modifier.New(am))
}

// AbilityCheck returns the ability modifier alone: a stat block has no ability-check
// proficiency except through its Skills (SRD 5.2.1, "Skills", p. 255). It returns the
// zero RollModifiers for an invalid ability.
func (s MonsterSetup) AbilityCheck(a ability.Ability) d20.RollModifiers {
	if !a.Valid() {
		return d20.RollModifiers{}
	}
	return d20.NewRollModifiers(modifier.New(s.monster.AbilityModifier(a).Value()))
}
