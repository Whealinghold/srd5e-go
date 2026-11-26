package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// attackFlow is the unexported helper both attack use cases share: it rolls the
// d20s and the damage dice through the dice port and composes the pure d20
// functions, the CoverResolver, the CircumstanceCollector and the
// AttackDamageComposer. It is not a use case, calls no use case, never calls
// d20.Resolver and records no signal. combat-action-resolution reaches the same
// steps through it.
type attackFlow struct{ roller Roller }

// rollD20 draws the dice of mode through the dice port and builds the d20.Roll. A
// roller failure is a *srderr.PortError; faces that d20 refuses are a contract
// violation of the dice port.
func (f attackFlow) rollD20(mode modifier.RollMode) (d20.Roll, error) {
	res, err := dice.Roll(f.roller, dice.MustExpression(mode.DiceCount(), dice.D20(), 0))
	if err != nil {
		return d20.Roll{}, err
	}
	faces := make([]int, 0, mode.DiceCount())
	for _, r := range res.Rolls() {
		faces = append(faces, r.Value())
	}
	roll, err := d20.NewRoll(mode, faces)
	if err != nil {
		return d20.Roll{}, srderr.ContractViolation(dice.Port, "roll", err.Error())
	}
	return roll, nil
}

// attackRoll resolves the roll mode of circumstances once, rolls that many d20s and
// resolves the attack against armorClass with d20.ResolveAttack.
func (f attackFlow) attackRoll(circumstances []modifier.RollCircumstance, mods d20.RollModifiers,
	armorClass d20.ArmorClass, criticalThreshold int) (d20.AttackRollResult, error) {
	roll, err := f.rollD20(modifier.Resolve(circumstances...))
	if err != nil {
		return d20.AttackRollResult{}, err
	}
	return d20.ResolveAttack(roll, mods, armorClass, criticalRange(criticalThreshold))
}

// rollDamage rolls the dice of every component of spec (doubled on a Critical Hit)
// and composes the Damage with the ability modifier.
func (f attackFlow) rollDamage(spec DamageSpec, abilityModifier int, critical bool) ([]Damage, error) {
	var composer AttackDamageComposer
	rolls := make([][]int, len(spec.components))
	for i, c := range spec.components {
		expr, ok := composer.Dice(c, critical)
		if !ok {
			continue
		}
		res, err := dice.Roll(f.roller, expr)
		if err != nil {
			return nil, err
		}
		for _, r := range res.Rolls() {
			rolls[i] = append(rolls[i], r.Value())
		}
	}
	return composer.Compose(spec, rolls, abilityModifier, critical)
}

// rollTest makes a saving throw or ability check of kind against dc, as d20.Resolver
// does for a RollAttempt: it resolves the mode once, draws that many d20s and compares
// with d20.ResolveCheck.
func (f attackFlow) rollTest(kind d20.Kind, a d20.RollAttempt, dc d20.DifficultyClass) (UnarmedStrikeSave, error) {
	roll, err := f.rollD20(modifier.Resolve(a.Circumstances...))
	if err != nil {
		return nil, err
	}
	res, err := d20.ResolveCheck(kind, roll, a.Modifiers, dc)
	if err != nil {
		return nil, err
	}
	return UnarmedStrikeSaveRolled{res}, nil
}

// forced returns the failure without a die of kind for reason.
func forced(kind d20.Kind, reason d20.ForcedFailureReason) (UnarmedStrikeSave, error) {
	res, err := d20.FailWithoutRoll(kind, reason)
	if err != nil {
		return nil, err
	}
	return UnarmedStrikeSaveForced{res}, nil
}

// resolveSave resolves a saving throw against dc: a RollAttempt rolls, ChooseToFail
// and AutomaticFailure draw no die. It adds no modifier. An attempt of another type
// is an error matching srderr.ErrInvalidInput.
func (f attackFlow) resolveSave(a d20.SaveAttempt, dc d20.DifficultyClass) (UnarmedStrikeSave, error) {
	switch v := a.(type) {
	case d20.RollAttempt:
		return f.rollTest(d20.SavingThrow, v, dc)
	case d20.ChooseToFail:
		return forced(d20.SavingThrow, d20.ChosenToFail)
	case d20.AutomaticFailure:
		return forced(d20.SavingThrow, d20.Automatic)
	}
	return nil, invalidInput("save attempt " + typeName(a))
}

// resolveCheck resolves an ability check against dc: a RollAttempt rolls, an
// AutomaticFailure draws no die. It adds no modifier. An attempt of another type is
// an error matching srderr.ErrInvalidInput.
func (f attackFlow) resolveCheck(a d20.CheckAttempt, dc d20.DifficultyClass) (UnarmedStrikeSave, error) {
	switch v := a.(type) {
	case d20.RollAttempt:
		return f.rollTest(d20.AbilityCheck, v, dc)
	case d20.AutomaticFailure:
		return forced(d20.AbilityCheck, d20.Automatic)
	}
	return nil, invalidInput("check attempt " + typeName(a))
}

// effectiveArmorClass adds the cover bonus to armorClass, saturating at int.
func effectiveArmorClass(armorClass d20.ArmorClass, cover Cover) d20.ArmorClass {
	return d20.ArmorClass(satAdd(int(armorClass), cover.ACBonus()))
}

// resolveAttack is the work of ResolveAttack.Run.
func (f attackFlow) resolveAttack(cmd ResolveAttackCommand) (ResolveAttackResult, error) {
	p := cmd.p
	cover := CoverResolver{}.Applicable(p.Cover)
	switch plan := (CircumstanceCollector{}).Collect(p.Form, p.Distance, p.Range, p.Reach, cover, p.NearbyEnemies, p.Situation, p.Damage).(type) {
	case AttackPlanNotPossible:
		return AttackNotPossible{plan.Reason, cover}, nil
	case AttackPlanAutomaticMiss:
		return AttackAutomaticallyMissed{plan.Reason, cover, plan.RevealsAttackerLocation}, nil
	case AttackPlanProceeds:
		circumstances := slices.Concat(plan.Circumstances, p.Circumstances)
		effective := effectiveArmorClass(p.ArmorClass, cover)
		atk, err := f.attackRoll(circumstances, p.Modifiers, effective, p.CriticalThreshold)
		if err != nil {
			return nil, err
		}
		hit := atk.Hit() && plan.MissOverride == MissOverrideNone
		critical := hit && (atk.IsCriticalHit() || p.AutomaticCritical == AutomaticCriticalAutomatic)
		var damage []Damage
		if hit && !p.Damage.IsZero() {
			if damage, err = f.rollDamage(p.Damage, p.Modifiers.Ability().Value(), critical); err != nil {
				return nil, err
			}
		}
		return AttackResolved{atk, plan.MissOverride, hit, critical, p.ArmorClass, effective, cover, circumstances,
			p.Form.Delivery(), damage, plan.RevealsAttackerLocation}, nil
	default:
		panic(fmt.Errorf("combat: unreachable attack plan %T", plan))
	}
}

// ResolveAttack is the use case of one attack roll and its damage (SRD 5.2.1,
// "Making an Attack", pp. 14-15). It rolls through the dice port and composes the
// pure d20 functions; it never calls d20.Resolver or another use case, so no
// srd5e.d20_test signal fires. It holds no rules logic. Create it with
// NewResolveAttack; the zero value is not usable. It is safe for concurrent use
// only if its roller, telemetry and clock are.
type ResolveAttack struct {
	in   instrumenter
	tel  telemetry.Telemetry
	flow attackFlow
}

// NewResolveAttack returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput
// for a nil roller.
func NewResolveAttack(roller Roller, opts ...Option) (ResolveAttack, error) {
	if roller == nil {
		return ResolveAttack{}, invalidInput("attack without a dice roller")
	}
	o := newOptions(opts)
	return ResolveAttack{newInstrumenter(o.tel, o.clock), o.tel, attackFlow{roller}}, nil
}

// Run resolves the attack of cmd. It decides what cannot be done before any die
// (Total Cover, beyond reach or range), then a miss without a roll, otherwise
// rolls the d20s the circumstances call for, then the damage dice on a hit. A miss,
// an automatic miss and AttackNotPossible are results, not errors.
//
// It records the shared signals with the attack extras, one sample of
// srd5e.combat.attack.modifiers per returned result and at most one debug log. It
// returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case or command, and ctx.Err() unwrapped, before any signal, for a cancelled
// ctx. It returns a *srderr.PortError (error type "dice_roller_error") with a nil
// result when the roller fails, panics or answers another expression; the failure is
// recorded once and never logged.
func (u ResolveAttack) Run(ctx context.Context, cmd ResolveAttackCommand) (ResolveAttackResult, error) {
	if u.flow.roller == nil || !cmd.p.Form.Valid() {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveAttack, spanResolveAttack}
	res, err := run(ctx, u.in, op, func(context.Context) (ResolveAttackResult, error) { return u.flow.resolveAttack(cmd) },
		func(r ResolveAttackResult) description { return describeAttack(cmd, r) })
	if err != nil {
		return nil, err
	}
	tel := u.tel.WithDefaults()
	tel.Meter.Histogram(telemetry.Descriptor{Name: metricAttackModifiers, Unit: "{modifier}",
		Description: "Modifiers in the RollModifiers of a returned attack."}).
		Record(ctx, float64(cmd.p.Modifiers.Count()), telemetry.String(attrAttackForm, cmd.p.Form.String()))
	tel.Logger.Log(ctx, telemetry.LevelDebug, msgAttackResolved, func() []telemetry.Attr {
		return []telemetry.Attr{telemetry.String(attrResult, describeAttack(cmd, res).Name)}
	})
	return res, nil
}

// describeAttack classifies a result: its class, srd5e.combat.result and the extras
// of attacks.md section 7. The d20 extras are left out when no d20 was rolled.
func describeAttack(cmd ResolveAttackCommand, r ResolveAttackResult) description {
	extras := []telemetry.Attr{
		telemetry.String(attrAttackForm, cmd.p.Form.String()),
		telemetry.Bool(attrAttackAutomaticCritical, cmd.p.AutomaticCritical == AutomaticCriticalAutomatic),
	}
	switch v := r.(type) {
	case AttackResolved:
		name := valueResultMiss
		switch {
		case v.critical:
			name = valueResultCriticalHit
		case v.hit:
			name = valueResultHit
		}
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()),
			telemetry.String(d20.AttrRollMode, v.attack.Roll().Mode().String()),
			telemetry.Bool(d20.AttrCriticalHit, v.attack.IsCriticalHit()))
		return description{Class: resultApplied, Name: name, Extras: extras}
	case AttackAutomaticallyMissed:
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()))
		return description{Class: resultApplied, Name: valueResultAutomaticMiss, Extras: extras}
	case AttackNotPossible:
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()),
			telemetry.String(attrAttackReason, v.reason.String()))
		return description{Class: resultRefused, Name: valueResultNotPossible, Extras: extras}
	default:
		panic(fmt.Errorf("combat: unreachable attack result %T", r))
	}
}
