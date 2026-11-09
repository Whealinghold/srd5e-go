package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	divineInterventionLevel = 10
	// divineInterventionMaxSpellLevel is the highest level of a spell on the Cleric list that Divine Intervention may cast.
	divineInterventionMaxSpellLevel = 5
)

// DivineInterventionVerdict is the outcome of DivineIntervention.Check. A zero Reason means the spell may be
// chosen; CastingTime is then the casting time in force and Grant, when Granted, the unlocked grant the spell
// comes from. Construct it only through DivineIntervention.Check.
type DivineInterventionVerdict struct {
	Reason      ClericReason
	CastingTime content.CastingTime
	Grant       ExtraDivineInterventionGrant
	Granted     bool
}

// DivineIntervention is the pure service of Divine Intervention and Greater Divine Intervention (SRD
// 5.2.1, "Level 10: Divine Intervention", "Level 20: Greater Divine Intervention", p. 38). It is stateless;
// the zero value is ready to use. Safe for concurrent use.
type DivineIntervention struct{}

// Check decides whether a Cleric of the level, with Divine Intervention in the state, may choose spell with the
// chosen casting time (nil for the first casting time of the spell). The first reason that holds is reported:
// below_level_10, not_available, not_cleric_spell (the spell is neither on the Cleric list nor an unlocked grant
// of rules), spell_level_too_high (above level 5, unless it is an unlocked grant) and reaction_casting_time.
//
// It returns an error matching srderr.ErrInvalidInput for the zero level, a nil state or a casting time the spell
// does not list. It never panics.
func (DivineIntervention) Check(level ClericLevel, state DivineInterventionState, spell content.Spell, castingTime content.CastingTime, rules ClericRuleData) (DivineInterventionVerdict, error) {
	times := castingTimesOf(spell)
	if level.IsZero() || state == nil || len(times) == 0 || castingTime != nil && !slices.Contains(times, castingTime) {
		return DivineInterventionVerdict{}, invalidInput("Divine Intervention check: zero level, nil state or a casting time the spell does not list")
	}
	if castingTime == nil {
		castingTime = times[0]
	}
	v := DivineInterventionVerdict{CastingTime: castingTime}
	v.Grant, v.Granted = rules.grantFor(spell.ID(), level)
	_, available := state.(DivineInterventionAvailable)
	onList := slices.Contains(spell.Classes(), ClericClassID)
	switch {
	case level.n < divineInterventionLevel:
		v.Reason = ClericBelowLevel10
	case !available:
		v.Reason = ClericNotAvailable
	case !onList && !v.Granted:
		v.Reason = ClericNotClericSpell
	case int(spell.Level()) > divineInterventionMaxSpellLevel && !v.Granted:
		v.Reason = ClericSpellLevelTooHigh
	default:
		if _, reaction := castingTime.(content.CastingTimeReaction); reaction {
			v.Reason = ClericReactionCastingTime
		}
	}
	return v, nil
}

// RechargeAfterGrant returns the state after a granted spell was chosen and the grant's recharge dice rolled: the
// total is the number of Long Rests to finish (2 to 8 for 2d4). It returns an error matching
// srderr.ErrInvalidInput for a number of rolls other than the grant's dice or a roll outside the die's faces.
func (DivineIntervention) RechargeAfterGrant(grant ExtraDivineInterventionGrant, rolls []int) (DivineInterventionRecharging, error) {
	if grant.diceCount < 1 || len(rolls) != grant.diceCount {
		return DivineInterventionRecharging{}, invalidInput("Divine Intervention recharge: " + strconv.Itoa(len(rolls)) + " rolls for the grant's dice")
	}
	total := 0
	for _, r := range rolls {
		if r < 1 || r > grant.die.Sides() {
			return DivineInterventionRecharging{}, invalidInput("Divine Intervention recharge roll " + strconv.Itoa(r) + " outside the die's faces")
		}
		total += r
	}
	return DivineInterventionRecharging{LongRestsRemaining: total}, nil
}

// castingTimesOf lists the casting times of spell: its one time, or the time of each mode.
func castingTimesOf(spell content.Spell) []content.CastingTime {
	var out []content.CastingTime
	switch t := spell.CastingTimes().(type) {
	case content.SpellCastingTimesSingle:
		out = append(out, t.Time())
	case content.SpellCastingTimesModes:
		for _, m := range t.Modes() {
			out = append(out, m.Time())
		}
	}
	return out
}
