package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// SpellModifierChoice is what the caster chose when casting a spell that offers a choice (spell-granted-modifiers.md
// FR-001). Variants: NoChoice, ChosenSkill, ChosenAbility, ChosenResistanceType, ChosenCurse.
type SpellModifierChoice interface{ isSpellModifierChoice() }

// NoChoice is the choice of a spell that offers none.
type NoChoice struct{}

// ChosenSkill is the Skill chosen for Guidance. Construct it with a keyed literal.
type ChosenSkill struct{ Skill content.Skill }

// ChosenAbility is the ability chosen for Hex. Construct it with a keyed literal.
type ChosenAbility struct{ Ability ability.Ability }

// ChosenResistanceType is the Damage Type chosen for Resistance. Construct it with a keyed literal.
type ChosenResistanceType struct{ Type content.DamageType }

// ChosenCurse is the curse chosen for Bestow Curse. Construct it with a keyed literal.
type ChosenCurse struct{ Effect CurseEffect }

func (NoChoice) isSpellModifierChoice()             {}
func (ChosenSkill) isSpellModifierChoice()          {}
func (ChosenAbility) isSpellModifierChoice()        {}
func (ChosenResistanceType) isSpellModifierChoice() {}
func (ChosenCurse) isSpellModifierChoice()          {}

// CurseEffect is the effect a Bestow Curse caster chose (SRD 5.2.1, "Bestow Curse", pp. 112-113). Variants:
// CurseAbility, CurseAttackRolls, CurseDodge, CurseExtraDamage.
type CurseEffect interface{ isCurseEffect() }

// CurseAbility is Disadvantage on ability checks and saving throws made with one ability. Construct it with a keyed
// literal.
type CurseAbility struct{ Ability ability.Ability }

// CurseAttackRolls is Disadvantage on attack rolls against the caster.
type CurseAttackRolls struct{}

// CurseDodge is the forced Dodge action (a fact, never resolved here).
type CurseDodge struct{}

// CurseExtraDamage is the extra 1d8 Necrotic damage the caster deals to the cursed creature.
type CurseExtraDamage struct{}

func (CurseAbility) isCurseEffect()     {}
func (CurseAttackRolls) isCurseEffect() {}
func (CurseDodge) isCurseEffect()       {}
func (CurseExtraDamage) isCurseEffect() {}

// resistanceTypeOK reports whether t is one of the eleven Damage Types the Resistance spell lists (SRD 5.2.1,
// "Resistance", p. 158): every type but Force and Psychic.
func resistanceTypeOK(t content.DamageType) bool {
	return t.Valid() && t != content.DamageForce && t != content.DamagePsychic
}

// choiceFits reports whether choice is a variant the spell accepts. A spell that is not modelled accepts any valid
// choice, since it contributes nothing.
func choiceFits(spell content.SpellID, choice SpellModifierChoice) bool {
	switch c := choice.(type) {
	case NoChoice:
		return !slices.Contains([]content.SpellID{spellGuidance, spellHex, spellResistance, spellBestowCurse}, spell)
	case ChosenSkill:
		return c.Skill.Valid() && (spell == spellGuidance || !isModelled(spell))
	case ChosenAbility:
		return c.Ability.Valid() && (spell == spellHex || !isModelled(spell))
	case ChosenResistanceType:
		return spell == spellResistance && resistanceTypeOK(c.Type) || !isModelled(spell) && c.Type.Valid()
	case ChosenCurse:
		if a, ok := c.Effect.(CurseAbility); ok && !a.Ability.Valid() {
			return false
		}
		switch c.Effect.(type) {
		case CurseAbility, CurseAttackRolls, CurseDodge, CurseExtraDamage:
			return spell == spellBestowCurse || !isModelled(spell)
		}
	}
	return false
}

// SpellModifierSource is one active spell on one creature, as the host maps it from the effects that
// EffectCombination.Applying returns (spell-granted-modifiers.md FR-001, FR-008): the spell, who cast it, who it is
// on, an order for output and the caster's choice. It holds no lifecycle. No function returns a CombatantID: read the
// creatures with CasterIs and TargetIs. The zero value is invalid. Comparable with ==; immutable; safe for
// concurrent use.
type SpellModifierSource struct {
	spell  content.SpellID
	caster CombatantID
	target CombatantID
	order  int
	choice SpellModifierChoice
}

// NewSpellModifierSource returns the source. castOrder only orders output and need not be unique.
//
// It returns an error matching srderr.ErrInvalidInput for a spell id that is not valid, a zero caster or target, a
// negative castOrder, and a choice that is nil or of the wrong variant for the spell (Guidance needs ChosenSkill,
// Hex ChosenAbility, Resistance a ChosenResistanceType of its eleven types, Bestow Curse ChosenCurse, every other
// modelled spell NoChoice). A spell that is not modelled takes any valid choice. It never panics.
func NewSpellModifierSource(spell content.SpellID, caster, target CombatantID, castOrder int, choice SpellModifierChoice) (SpellModifierSource, error) {
	switch {
	case !spell.Valid():
		return SpellModifierSource{}, invalidInput("spell modifier source without a valid spell id")
	case caster.IsZero() || target.IsZero():
		return SpellModifierSource{}, invalidInput("spell modifier source without a caster or target")
	case castOrder < 0:
		return SpellModifierSource{}, invalidInput("negative cast order")
	case !choiceFits(spell, choice):
		return SpellModifierSource{}, invalidInput("choice does not fit the spell")
	}
	return SpellModifierSource{spell, caster, target, castOrder, choice}, nil
}

// MustSpellModifierSource is NewSpellModifierSource for values known to be valid. It panics with the error
// NewSpellModifierSource returns, and only then.
func MustSpellModifierSource(spell content.SpellID, caster, target CombatantID, castOrder int, choice SpellModifierChoice) SpellModifierSource {
	s, err := NewSpellModifierSource(spell, caster, target, castOrder, choice)
	if err != nil {
		panic(err)
	}
	return s
}

// IsZero reports whether s is the invalid zero source.
func (s SpellModifierSource) IsZero() bool { return s.spell == "" }

// SpellID returns the spell.
func (s SpellModifierSource) SpellID() content.SpellID { return s.spell }

// CastOrder returns the order used to order output.
func (s SpellModifierSource) CastOrder() int { return s.order }

// Choice returns the caster's choice.
func (s SpellModifierSource) Choice() SpellModifierChoice { return s.choice }

// CasterIs reports whether c cast the spell.
func (s SpellModifierSource) CasterIs(c CombatantID) bool { return !c.IsZero() && s.caster == c }

// TargetIs reports whether the spell is on the creature c.
func (s SpellModifierSource) TargetIs(c CombatantID) bool { return !c.IsZero() && s.target == c }

// maxSourcesPerTarget is the cap of plan P10 on the sources one creature holds: it is the cap of active effects per creature.
const maxSourcesPerTarget = maxEffectsPerCreature

// checkSources returns an error matching ErrDuplicateSource when two sources hold one spell and one target, and one
// matching srderr.ErrInvalidInput for a zero source (FR-008, FR-084). A creature holding more than 64 sources is a LimitExceededError (FR-097, plan P10). It does not alter sources.
func checkSources(sources []SpellModifierSource) error {
	type key struct {
		spell  content.SpellID
		target CombatantID
	}
	seen := make(map[key]struct{}, len(sources))
	per := make(map[CombatantID]int, len(sources))
	for _, s := range sources {
		if s.IsZero() {
			return invalidInput("zero spell modifier source")
		}
		k := key{s.spell, s.target}
		if _, dup := seen[k]; dup {
			return ErrDuplicateSource
		}
		seen[k] = struct{}{}
		per[s.target]++
		if err := overCap("spell modifier sources per target", per[s.target], maxSourcesPerTarget); err != nil {
			return err
		}
	}
	return nil
}

// ordered returns the sources that satisfy keep, in ascending CastOrder; equal orders keep the
// input order (FR-023).
func ordered(sources []SpellModifierSource, keep func(SpellModifierSource) bool) []SpellModifierSource {
	var out []SpellModifierSource
	for _, s := range sources {
		if keep(s) {
			out = append(out, s)
		}
	}
	slices.SortStableFunc(out, func(a, b SpellModifierSource) int { return a.order - b.order })
	return out
}
