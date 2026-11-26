package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// DamageNotationChoice is how one damage instance of a stat block is rolled: by its printed
// number or by its printed die expression, never both (SRD 5.2.1, "Damage Notation", p. 257: "You
// decide whether to use the number or the die expression"). The zero value is invalid. Comparable
// and safe for concurrent use.
type DamageNotationChoice int

// The two choices.
const (
	// DamageNotationNumber: use the printed number; no die is rolled.
	DamageNotationNumber DamageNotationChoice = iota + 1
	// DamageNotationDice: roll the printed die expression, flat modifier included.
	DamageNotationDice
)

// String returns "number", "dice" or "unknown": the values of srd5e.combat.damage_notation.
func (c DamageNotationChoice) String() string {
	return enumName([]string{"number", "dice"}, int(c))
}

// DamageNotationChoices are the choices for the damage instances of one part, fixed before any
// roll (monster-stat-block-actions.md FR-027): one per instance in printed order, or the same
// choice for every instance. It holds a slice, so compare it with Equal. The zero value holds no
// choice and fits a part with no damage. Safe for concurrent use.
type DamageNotationChoices struct {
	each []DamageNotationChoice
	all  DamageNotationChoice
}

// NewDamageNotationChoices returns the choices, one per damage instance in printed order.
//
// It returns an error matching srderr.ErrInvalidInput for no choice or a choice that is not
// DamageNotationNumber or DamageNotationDice. It never panics.
func NewDamageNotationChoices(choices ...DamageNotationChoice) (DamageNotationChoices, error) {
	if len(choices) == 0 || slices.ContainsFunc(choices, func(c DamageNotationChoice) bool { return c != DamageNotationNumber && c != DamageNotationDice }) {
		return DamageNotationChoices{}, invalidInput("damage notation choices need at least one valid choice")
	}
	return DamageNotationChoices{each: slices.Clone(choices)}, nil
}

// AllDamageNotation returns the choice c for every damage instance of a part.
//
// It returns an error matching srderr.ErrInvalidInput for a c that is not DamageNotationNumber
// or DamageNotationDice. It never panics.
func AllDamageNotation(c DamageNotationChoice) (DamageNotationChoices, error) {
	if c != DamageNotationNumber && c != DamageNotationDice {
		return DamageNotationChoices{}, invalidInput("damage notation choice is neither number nor dice")
	}
	return DamageNotationChoices{all: c}, nil
}

// Equal reports whether c and o make the same choices.
func (c DamageNotationChoices) Equal(o DamageNotationChoices) bool {
	return c.all == o.all && slices.Equal(c.each, o.each)
}

// For returns the choice for each of the damage instances, in printed order, as a new slice.
//
// It returns an error matching srderr.ErrInvalidInput when c lists a number of choices other than
// the number of instances (the zero value fits none), and when DamageNotationDice is chosen for an
// instance printed with no die expression ("1 Slashing damage"). It never panics.
func (c DamageNotationChoices) For(damage []content.StatBlockDamage) ([]DamageNotationChoice, error) {
	out := slices.Clone(c.each)
	if c.all != 0 {
		out = slices.Repeat([]DamageNotationChoice{c.all}, len(damage))
	}
	if len(out) != len(damage) {
		return nil, invalidInput("damage notation choices differ in number from the damage instances")
	}
	for i, d := range damage {
		if _, hasDice := d.Dice(); out[i] == DamageNotationDice && !hasDice {
			return nil, invalidInput("dice chosen for a damage instance printed with no die expression")
		}
	}
	return out, nil
}

// statBlockDamageType returns the Damage Type d prints, and false for a type given as text that
// the engine cannot read.
func statBlockDamageType(d content.StatBlockDamage) (content.DamageType, bool) {
	s, ok := d.Type().(content.SpecificDamageType)
	return s.DamageType(), ok
}

// StatBlockAttackPlan is what the Attack Notation of a part gives an attack roll (SRD 5.2.1,
// "Attack Notation", p. 256): the attack form, the roll's modifiers (the printed bonus alone), the
// reach or range, the printed qualifier and the damage of a hit. Pass the pieces to ResolveAttack.
// Immutable; the zero value is invalid. Safe for concurrent use.
type StatBlockAttackPlan struct {
	form      AttackForm
	modifiers d20.RollModifiers
	reach     Reach
	rng       AttackRange
	qualifier string
	damage    DamageSpec
}

// Form returns MeleeWeaponAttack or RangedWeaponAttack.
func (p StatBlockAttackPlan) Form() AttackForm { return p.form }

// Modifiers returns the attack roll's modifiers: the printed bonus as the ability modifier and
// nothing else, since the bonus is the whole attack modifier (FR-022).
func (p StatBlockAttackPlan) Modifiers() d20.RollModifiers { return p.modifiers }

// Reach returns the printed reach of a melee attack, and false for a ranged one.
func (p StatBlockAttackPlan) Reach() (Reach, bool) {
	return p.reach, p.form.Delivery() == AttackDeliveryMelee
}

// Range returns the printed range of a ranged attack, and false for a melee one.
func (p StatBlockAttackPlan) Range() (AttackRange, bool) {
	return p.rng, p.form.Delivery() == AttackDeliveryRanged
}

// Qualifier returns the printed qualifier after the bonus, verbatim and never interpreted.
func (p StatBlockAttackPlan) Qualifier() string { return p.qualifier }

// Damage returns the damage of a hit, and false when the Hit clause prints none.
func (p StatBlockAttackPlan) Damage() (DamageSpec, bool) { return p.damage, !p.damage.IsZero() }

// IsZero reports whether p is the invalid zero plan.
func (p StatBlockAttackPlan) IsZero() bool { return !p.form.Valid() }

// AttackPlanOption configures PlanStatBlockAttack.
type AttackPlanOption func(*attackPlanConfig)

type attackPlanConfig struct{ delivery AttackDelivery }

// WithDelivery chooses melee or ranged for an attack printed "Melee or Ranged Attack Roll"; it is
// required for that attack and, for any other, must agree with it.
func WithDelivery(d AttackDelivery) AttackPlanOption {
	return func(c *attackPlanConfig) { c.delivery = d }
}

// planDelivery returns the delivery the part's attack takes, given the choice opts made.
func planDelivery(kind content.StatBlockAttackKind, chosen AttackDelivery) (AttackDelivery, bool) {
	fixed := map[content.StatBlockAttackKind]AttackDelivery{
		content.AttackMelee: AttackDeliveryMelee, content.AttackRanged: AttackDeliveryRanged}[kind]
	switch {
	case fixed != 0:
		return fixed, chosen == 0 || chosen == fixed
	default:
		return chosen, chosen == AttackDeliveryMelee || chosen == AttackDeliveryRanged
	}
}

// PlanStatBlockAttack plans the attack of part with the damage notation choices (monster-stat-block-actions.md
// FR-022 to FR-030; SRD 5.2.1, "Attack Notation", "Damage Notation", pp. 256-257). The attack form is
// MeleeWeaponAttack or RangedWeaponAttack by delivery; a melee-only or ranged-only attack takes its own
// delivery when WithDelivery is not given, and a "Melee or Ranged" one needs WithDelivery. A melee plan carries
// the printed reach; a ranged plan the printed range, a single range when no long range is printed. The roll's
// modifiers are the printed bonus alone, with no Proficiency Bonus and no ability modifier added again, and the
// planned damage adds no ability modifier either, since the printed numbers include it. The damage is that of
// the Hit clause, instance by instance as the choices say; a Critical Hit adds no stat-block rule, so ResolveAttack
// doubles the dice of a dice instance and leaves a number as it is.
//
// It returns an error matching srderr.ErrInvalidInput for a part with no Attack Notation, a missing, invalid or
// disagreeing delivery, a melee plan of an attack with no reach or a ranged plan of one with no range, a bonus,
// reach or range outside the range combat relies on (plan P13), choices that do not fit the Hit damage, a damage
// type printed as text, and dice the damage rules refuse. It never panics.
func PlanStatBlockAttack(part content.StatBlockPart, choices DamageNotationChoices, opts ...AttackPlanOption) (StatBlockAttackPlan, error) {
	atk, ok := part.Attack()
	if !ok {
		return StatBlockAttackPlan{}, invalidInput("the part has no Attack Notation")
	}
	var cfg attackPlanConfig
	for _, o := range opts {
		o(&cfg)
	}
	delivery, ok := planDelivery(atk.Kind(), cfg.delivery)
	if !ok || !monsterNumberOK(atk.Bonus()) {
		return StatBlockAttackPlan{}, invalidInput("the attack needs a delivery it can take and a bonus in range")
	}
	plan := StatBlockAttackPlan{modifiers: d20.NewRollModifiers(modifier.New(atk.Bonus())), qualifier: atk.Qualifier()}
	var err error
	if delivery == AttackDeliveryMelee {
		feet, hasReach := atk.Reach()
		if !hasReach {
			return StatBlockAttackPlan{}, invalidInput("a melee plan needs the printed reach")
		}
		plan.form = MeleeWeaponAttack
		if plan.reach, err = NewReach(feet); err != nil {
			return StatBlockAttackPlan{}, err
		}
	} else {
		r, hasRange := atk.Range()
		if !hasRange {
			return StatBlockAttackPlan{}, invalidInput("a ranged plan needs the printed range")
		}
		long, _ := r.Long()
		plan.form = RangedWeaponAttack
		if plan.rng, err = NewAttackRange(r.Normal(), long); err != nil {
			return StatBlockAttackPlan{}, err
		}
	}
	for _, c := range part.Clauses() {
		if c.Label() != content.ClauseHit || len(c.Damage()) == 0 {
			continue
		}
		if plan.damage, err = planDamage(c.Damage(), choices); err != nil {
			return StatBlockAttackPlan{}, err
		}
	}
	if len(choices.each) > 0 && plan.damage.IsZero() {
		return StatBlockAttackPlan{}, invalidInput("damage notation choices for a hit that prints no damage")
	}
	return plan, nil
}

// planDamage returns the DamageSpec of the damage instances rolled as choices say.
func planDamage(damage []content.StatBlockDamage, choices DamageNotationChoices) (DamageSpec, error) {
	picks, err := choices.For(damage)
	if err != nil {
		return DamageSpec{}, err
	}
	components := make([]DamageComponent, len(damage))
	for i, d := range damage {
		t, readable := statBlockDamageType(d)
		if !readable {
			return DamageSpec{}, invalidInput("a damage type printed as text cannot be planned")
		}
		if picks[i] == DamageNotationNumber {
			components[i], err = NewFixedDamage(d.Number(), t, 0, false)
		} else {
			e, _ := d.Dice()
			components[i], err = NewDiceDamage(e.Count(), e.Die(), t, e.Modifier(), false)
		}
		if err != nil {
			return DamageSpec{}, err
		}
	}
	return NewDamageSpec(components...)
}

// AppliedClausesForAttack returns the clauses of the part's Attack Notation that apply to an attack
// that was made, in printed order (monster-stat-block-actions.md FR-031; SRD 5.2.1, "Attack Notation",
// p. 256): for a hit, a Critical Hit included, the Hit and Hit or Miss clauses; for a miss, a natural 1,
// an automatic miss and a forced miss included, the Miss and Hit or Miss clauses. Hit and Miss never
// both apply. An attack that was not possible (possible false) applies no clause. The damage of a clause
// other than Hit is for RollStatBlockDamage to roll; their text is the host's to read.
//
// It returns an error matching srderr.ErrInvalidInput for a part with no Attack Notation. It never panics.
func AppliedClausesForAttack(part content.StatBlockPart, hit, possible bool) ([]content.EffectClause, error) {
	if _, ok := part.Attack(); !ok {
		return nil, invalidInput("the part has no Attack Notation")
	}
	var out []content.EffectClause
	for _, c := range part.Clauses() {
		l := c.Label()
		if possible && (l == content.ClauseHitOrMiss || (hit && l == content.ClauseHit) || (!hit && l == content.ClauseMiss)) {
			out = append(out, c)
		}
	}
	return out, nil
}
