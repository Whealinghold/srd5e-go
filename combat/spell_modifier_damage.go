package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// DamageEvent is the damage a rider is asked for (spell-granted-modifiers.md FR-061). A miss produces no event.
// Variants: AttackRollHitEvent, SpellDamageEvent, OtherDamageEvent.
type DamageEvent interface{ isDamageEvent() }

// AttackRollHitEvent is damage of a weapon or spell attack roll that hit.
type AttackRollHitEvent struct{}

// SpellDamageEvent is damage of a spell that uses a saving throw or no roll.
type SpellDamageEvent struct{}

// OtherDamageEvent is any other damage, such as a trap or an object.
type OtherDamageEvent struct{}

func (AttackRollHitEvent) isDamageEvent() {}
func (SpellDamageEvent) isDamageEvent()   {}
func (OtherDamageEvent) isDamageEvent()   {}

// QueryDamageRiders returns the extra damage the active spells in sources add when attacker deals damage to target,
// in ascending CastOrder: Bestow Curse's 1d8 Necrotic, once per attack roll that hits and once per spell that deals
// damage, and the 1d6 of Hex (Necrotic) and of Hunter's Mark (Force), once per attack roll that hits (SRD 5.2.1, "Hex",
// p. 140; "Hunter's Mark", p. 141); never for other damage, for another attacker or for another target (FR-061,
// FR-063, FR-095). The component is the attack's own damage, so
// a Critical Hit doubles it by the attack's rule (FR-062), and it goes through Resistance, Vulnerability and
// Immunity as any damage (FR-064). The input is not altered.
//
// It returns an error matching ErrDuplicateSource (and srderr.ErrInvalidInput) for two sources with one spell and
// target, and one matching srderr.ErrInvalidInput for a zero source, a zero attacker or target or a nil event. It
// never panics.
func QueryDamageRiders(sources []SpellModifierSource, attacker, target CombatantID, ev DamageEvent) ([]DamageComponent, error) {
	if err := checkSources(sources); err != nil {
		return nil, err
	}
	if attacker.IsZero() || target.IsZero() || ev == nil {
		return nil, invalidInput("damage rider query without attacker, target or event")
	}
	if _, other := ev.(OtherDamageEvent); other {
		return nil, nil
	}
	var out []DamageComponent
	for _, s := range ordered(sources, func(s SpellModifierSource) bool { return s.CasterIs(attacker) && s.TargetIs(target) }) {
		if s.spell == spellBestowCurse {
			if s.choice == (ChosenCurse{CurseExtraDamage{}}) {
				out = append(out, MustDiceDamage(1, dice.D8(), content.DamageNecrotic, 0, false))
			}
			continue
		}
		// Hex and Hunter's Mark add their damage "whenever you hit it with an attack roll" only.
		if _, hit := ev.(AttackRollHitEvent); hit {
			if c, ok := riderOf(s.spell).ExtraDamage(); ok {
				out = append(out, c)
			}
		}
	}
	return out, nil
}

// SpellDamageReduction is what the Resistance spell takes off one instance of damage: a die and the Damage Type it
// covers. The zero value is none. Comparable with ==; immutable.
type SpellDamageReduction struct {
	typ content.DamageType
}

// Dice returns the die to roll, 1d4, and whether there is a reduction. Rolling it is the host's.
func (r SpellDamageReduction) Dice() (dice.Expression, bool) {
	if r.typ == 0 {
		return dice.Expression{}, false
	}
	return dice.MustExpression(1, dice.D4(), 0), true
}

// ToAdjustment returns the flat adjustment of -rolled to damage of the chosen type: it applies in the adjustment
// step, before Resistance and Vulnerability, and the damage is floored at 0 (FR-066). It returns an error matching
// srderr.ErrInvalidInput for a rolled outside 1 to 4 and for no reduction. It never panics.
func (r SpellDamageReduction) ToAdjustment(rolled int) (FlatAdjustment, error) {
	if r.typ == 0 || rolled < 1 || rolled > 4 {
		return FlatAdjustment{}, invalidInput("rolled outside the 1d4 of the Resistance spell, or no reduction")
	}
	return NewFlatAdjustment(-rolled, DamageOfType{Type: r.typ}), nil
}

// QueryDamageReduction returns the reduction of the Resistance spell for creature taking damage of type dmgType: a
// 1d4 when a source on creature chose that type and the reduction was not already made this turn, else none. The
// turn is the host's (FR-065). The input is not altered.
//
// It returns an error matching ErrDuplicateSource (and srderr.ErrInvalidInput) for two sources with one spell and
// target, and one matching srderr.ErrInvalidInput for a zero source, a zero creature or a Damage Type that is not
// valid. It never panics.
func QueryDamageReduction(sources []SpellModifierSource, creature CombatantID, dmgType content.DamageType, alreadyReducedThisTurn bool) (SpellDamageReduction, error) {
	if err := checkSources(sources); err != nil {
		return SpellDamageReduction{}, err
	}
	if creature.IsZero() || !dmgType.Valid() {
		return SpellDamageReduction{}, invalidInput("damage reduction query without a creature or a valid Damage Type")
	}
	if alreadyReducedThisTurn {
		return SpellDamageReduction{}, nil
	}
	for _, s := range sources {
		if s.spell == spellResistance && s.TargetIs(creature) && s.choice == (ChosenResistanceType{dmgType}) {
			return SpellDamageReduction{dmgType}, nil
		}
	}
	return SpellDamageReduction{}, nil
}
