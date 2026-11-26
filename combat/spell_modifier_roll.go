package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// RollContext is the roll a spell's contribution is asked for (spell-granted-modifiers.md section 4.2). Variants:
// AttackRollContext, SavingThrowContext, DeathSavingThrowContext, AbilityCheckContext.
type RollContext interface{ isRollContext() }

// AttackRollContext is an attack roll by Attacker against Target. Construct it with a keyed literal.
type AttackRollContext struct{ Attacker, Target CombatantID }

// SavingThrowContext is a saving throw of Creature with Ability. Construct it with a keyed literal.
type SavingThrowContext struct {
	Creature CombatantID
	Ability  ability.Ability
}

// DeathSavingThrowContext is a Death Saving Throw of Creature, which has no ability. Construct it with a keyed
// literal.
type DeathSavingThrowContext struct{ Creature CombatantID }

// AbilityCheckContext is an ability check of Creature with Ability. Skill is the Skill used with it, zero for a raw
// ability check, and Seeking the creature the check is made to find, zero for none. Construct it with a keyed
// literal.
type AbilityCheckContext struct {
	Creature CombatantID
	Ability  ability.Ability
	Skill    content.Skill
	Seeking  CombatantID
}

func (AttackRollContext) isRollContext()       {}
func (SavingThrowContext) isRollContext()      {}
func (DeathSavingThrowContext) isRollContext() {}
func (AbilityCheckContext) isRollContext()     {}

// valid reports whether the context names its creatures and has a valid ability and skill where it has them.
func rollContextValid(c RollContext) bool {
	switch v := c.(type) {
	case AttackRollContext:
		return !v.Attacker.IsZero() && !v.Target.IsZero()
	case SavingThrowContext:
		return !v.Creature.IsZero() && v.Ability.Valid()
	case DeathSavingThrowContext:
		return !v.Creature.IsZero()
	case AbilityCheckContext:
		return !v.Creature.IsZero() && v.Ability.Valid() && (v.Skill == 0 || v.Skill.Valid())
	}
	return false
}

// BonusDie is one die added to (or, as a penalty, subtracted from) a roll's total: Guidance's 1d4, or a signed die
// from a rider. Build it with NewBonusDie; the zero value is invalid. Comparable with ==; immutable.
type BonusDie struct {
	expr     dice.Expression
	negative bool
}

// NewBonusDie returns the die of expr, a penalty when negative. It returns an error matching
// srderr.ErrInvalidInput for the zero Expression. It never panics.
func NewBonusDie(expr dice.Expression, negative bool) (BonusDie, error) {
	if expr.IsZero() {
		return BonusDie{}, invalidInput("bonus die without dice")
	}
	return BonusDie{expr, negative}, nil
}

// Expression returns what is rolled.
func (b BonusDie) Expression() dice.Expression { return b.expr }

// Negative reports whether the rolled total is subtracted (a penalty die such as Bane's).
func (b BonusDie) Negative() bool { return b.negative }

// IsZero reports whether b is the invalid zero BonusDie.
func (b BonusDie) IsZero() bool { return b.expr.IsZero() }

// SpellRollContributions is what the active spells add to one roll, in ascending CastOrder of their sources
// (spell-granted-modifiers.md FR-023): bonus dice to roll, flat modifiers, and Advantage or Disadvantage for
// modifier.Resolve together with the circumstances of conditions. It resolves no mode. The zero value is no
// contribution. Not comparable with ==; compare with Equal. Slices are copied on the way out.
type SpellRollContributions struct {
	dice          []BonusDie
	modifiers     []modifier.Modifier
	circumstances []modifier.RollCircumstance
}

// BonusDice returns the dice to roll, as a new slice.
func (c SpellRollContributions) BonusDice() []BonusDie { return slices.Clone(c.dice) }

// Modifiers returns the flat modifiers, as a new slice.
func (c SpellRollContributions) Modifiers() []modifier.Modifier { return slices.Clone(c.modifiers) }

// Circumstances returns the Advantages and Disadvantages, as a new slice.
func (c SpellRollContributions) Circumstances() []modifier.RollCircumstance {
	return slices.Clone(c.circumstances)
}

// Equal reports whether c and o hold the same contributions in the same order.
func (c SpellRollContributions) Equal(o SpellRollContributions) bool {
	return slices.Equal(c.dice, o.dice) && slices.Equal(c.modifiers, o.modifiers) && slices.Equal(c.circumstances, o.circumstances)
}

// QueryRollContributions returns what the active spells in sources add to the roll in rc: Guidance's die, Bless's 1d4 and Bane's -1d4 on attack rolls,
// saving throws and Death Saving Throws, Hex's Disadvantage on checks with its chosen ability, Haste's
// Advantage and Slow's -2 on Dexterity saves, the Disadvantage of Bestow Curse and the Advantage of Hunter's Mark
// to find its mark. A source applies only to the creature it is on, except Hunter's Mark, which applies to its
// caster (FR-010). A source for a spell that is not modelled is skipped. No die is rolled and no mode resolved. The
// input is not altered.
//
// It returns an error matching ErrDuplicateSource (and srderr.ErrInvalidInput) for two sources with one spell and
// target, and one matching srderr.ErrInvalidInput for a zero source or an invalid context (nil, a zero creature, an
// invalid ability or skill). It never panics.
func QueryRollContributions(sources []SpellModifierSource, rc RollContext) (SpellRollContributions, error) {
	if err := checkSources(sources); err != nil {
		return SpellRollContributions{}, err
	}
	if !rollContextValid(rc) {
		return SpellRollContributions{}, invalidInput("invalid roll context")
	}
	var out SpellRollContributions
	for _, s := range ordered(sources, func(SpellModifierSource) bool { return true }) {
		d, m, c := sourceRollContribution(s, rc)
		if !d.IsZero() {
			out.dice = append(out.dice, d)
		}
		out.modifiers = append(out.modifiers, m...)
		out.circumstances = append(out.circumstances, c...)
	}
	return out, nil
}

// sourceRollContribution is the contribution of one source to a roll (FR-016 to FR-021).
func sourceRollContribution(s SpellModifierSource, rc RollContext) (BonusDie, []modifier.Modifier, []modifier.RollCircumstance) {
	adv, dis := []modifier.RollCircumstance{modifier.Advantage{}}, []modifier.RollCircumstance{modifier.Disadvantage{}}
	switch s.spell {
	case spellGuidance:
		if c, ok := rc.(AbilityCheckContext); ok && s.TargetIs(c.Creature) && c.Skill != 0 && s.choice == (ChosenSkill{c.Skill}) {
			return MustBonusDie(dice.MustExpression(1, dice.D4(), 0), false), nil, nil
		}
	case spellHaste:
		if c, ok := rc.(SavingThrowContext); ok && s.TargetIs(c.Creature) && c.Ability == ability.Dexterity {
			return BonusDie{}, nil, adv
		}
	case spellSlow:
		if c, ok := rc.(SavingThrowContext); ok && s.TargetIs(c.Creature) && c.Ability == ability.Dexterity {
			return BonusDie{}, []modifier.Modifier{modifier.New(-2)}, nil
		}
	case spellBestowCurse:
		return BonusDie{}, nil, curseCircumstances(s, rc, dis)
	case spellHuntersMark:
		if c, ok := rc.(AbilityCheckContext); ok && s.CasterIs(c.Creature) && c.Ability == ability.Wisdom &&
			(c.Skill == content.SkillPerception || c.Skill == content.SkillSurvival) && s.TargetIs(c.Seeking) {
			return BonusDie{}, nil, adv
		}
	}
	return riderRollContribution(s, rc, dis)
}

// riderRollContribution is the contribution of the rider of a source: the die of Bless and Bane to the attack rolls, saving
// throws and Death Saving Throws of the creature it is on, and the Disadvantage of Hex on its ability checks with the
// chosen ability (the spell-granted-modifiers.md riders).
func riderRollContribution(s SpellModifierSource, rc RollContext, dis []modifier.RollCircumstance) (BonusDie, []modifier.Modifier, []modifier.RollCircumstance) {
	rider := riderOf(s.spell)
	if die, ok := rider.RollDie(); ok {
		switch c := rc.(type) {
		case AttackRollContext:
			if s.TargetIs(c.Attacker) {
				return die, nil, nil
			}
		case SavingThrowContext:
			if s.TargetIs(c.Creature) {
				return die, nil, nil
			}
		case DeathSavingThrowContext:
			if s.TargetIs(c.Creature) {
				return die, nil, nil
			}
		}
	}
	if c, ok := rc.(AbilityCheckContext); ok && rider.CheckDisadvantage() && s.TargetIs(c.Creature) && s.choice == (ChosenAbility{c.Ability}) {
		return BonusDie{}, nil, dis
	}
	return BonusDie{}, nil, nil
}

// curseCircumstances is the Disadvantage of a Bestow Curse source on the roll, if any (FR-020).
func curseCircumstances(s SpellModifierSource, rc RollContext, dis []modifier.RollCircumstance) []modifier.RollCircumstance {
	curse, _ := s.choice.(ChosenCurse)
	switch e := curse.Effect.(type) {
	case CurseAbility:
		switch c := rc.(type) {
		case AbilityCheckContext:
			if s.TargetIs(c.Creature) && c.Ability == e.Ability {
				return dis
			}
		case SavingThrowContext:
			if s.TargetIs(c.Creature) && c.Ability == e.Ability {
				return dis
			}
		}
	case CurseAttackRolls:
		if c, ok := rc.(AttackRollContext); ok && s.TargetIs(c.Attacker) && s.CasterIs(c.Target) {
			return dis
		}
	}
	return nil
}

// MustBonusDie is NewBonusDie for values known to be valid. It panics with the error NewBonusDie returns, and only
// then.
func MustBonusDie(expr dice.Expression, negative bool) BonusDie {
	b, err := NewBonusDie(expr, negative)
	if err != nil {
		panic(err)
	}
	return b
}
