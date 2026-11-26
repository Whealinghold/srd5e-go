package combat

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// withCoverBonus appends the cover's Dexterity bonus, once, to a rolled attempt of a
// Dexterity save; any other attempt or ability is returned as it was. The original
// attempt is not altered.
func withCoverBonus(a d20.SaveAttempt, save ability.Ability, cover Cover) d20.SaveAttempt {
	r, ok := a.(d20.RollAttempt)
	if !ok || save != ability.Dexterity || cover.DexteritySaveBonus() == 0 {
		return a
	}
	r.Modifiers = r.Modifiers.WithOthers(modifier.New(cover.DexteritySaveBonus()))
	return r
}

// resolveUnarmedStrike is the work of ResolveUnarmedStrike.Run. extra are further modifiers of the Damage option's
// attack roll, added after the ability modifier and the Proficiency Bonus (combat action resolution adds the Exhaustion
// reduction of the attacker); a Grapple or Shove takes none.
func (f attackFlow) resolveUnarmedStrike(cmd ResolveUnarmedStrikeCommand, extra ...modifier.Modifier) (ResolveUnarmedStrikeResult, error) {
	p := cmd.p
	cover := CoverResolver{}.Applicable(p.Cover)
	if n, no := planUnarmedStrike(p, cover).(AttackPlanNotPossible); no {
		return UnarmedStrikeNotPossible{n.Reason, cover}, nil
	}
	switch o := p.Option.(type) {
	case UnarmedStrikeDamageOption:
		effective := effectiveArmorClass(o.ArmorClass, cover)
		atk, err := f.attackRoll(o.Circumstances, UnarmedStrikeModifiers(p.AbilityModifier, p.ProficiencyBonus).WithOthers(extra...), effective, o.CriticalThreshold)
		if err != nil {
			return nil, err
		}
		critical := atk.Hit() && (atk.IsCriticalHit() || p.AutomaticCritical == AutomaticCriticalAutomatic)
		var damage []Damage
		if atk.Hit() {
			// 1 plus the ability modifier, minimum 0; a Critical Hit has no dice to double.
			spec := MustDamageSpec(MustFixedDamage(1, content.DamageBludgeoning, 0, true))
			if damage, err = f.rollDamage(spec, p.AbilityModifier.Value(), critical); err != nil {
				return nil, err
			}
		}
		return UnarmedStrikeDamageResolved{atk, atk.Hit(), critical, o.ArmorClass, cover, damage}, nil
	case UnarmedStrikeGrappleOption:
		dc := GrappleRules{}.GrappleDC(p.AbilityModifier.Value(), p.ProficiencyBonus.Value())
		save, err := f.resolveSave(withCoverBonus(o.Attempt, o.SaveAbility, cover), dc)
		if err != nil {
			return nil, err
		}
		return UnarmedStrikeGrappleResolved{dc, o.SaveAbility, save, cover}, nil
	case UnarmedStrikeShoveOption:
		dc := unarmedStrikeDC(p.AbilityModifier, p.ProficiencyBonus)
		save, err := f.resolveSave(withCoverBonus(o.Attempt, o.SaveAbility, cover), dc)
		if err != nil {
			return nil, err
		}
		return UnarmedStrikeShoveResolved{dc, o.SaveAbility, save, o.Effect, cover}, nil
	default:
		panic(fmt.Errorf("combat: unreachable unarmed strike option %T", p.Option))
	}
}

// ResolveUnarmedStrike is the use case of one Unarmed Strike with its Damage,
// Grapple or Shove option (SRD 5.2.1, "Unarmed Strike", p. 190). It rolls through
// the dice port and composes the pure d20 functions and GrappleRules; it never calls
// d20.Resolver, ResolveAttack or another use case. It moves no creature and applies
// no condition. Create it with NewResolveUnarmedStrike; the zero value is not usable.
// It is safe for concurrent use only if its roller, telemetry and clock are.
type ResolveUnarmedStrike struct {
	in   instrumenter
	tel  telemetry.Telemetry
	flow attackFlow
}

// NewResolveUnarmedStrike returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for
// a nil roller.
func NewResolveUnarmedStrike(roller Roller, opts ...Option) (ResolveUnarmedStrike, error) {
	if roller == nil {
		return ResolveUnarmedStrike{}, invalidInput("unarmed strike without a dice roller")
	}
	o := newOptions(opts)
	return ResolveUnarmedStrike{newInstrumenter(o.tel, o.clock), o.tel, attackFlow{roller}}, nil
}

// Run resolves the strike of cmd. It refuses, before any die and in this order, a
// target behind Total Cover, beyond 5 feet, more than one size larger (Grapple and
// Shove) and an attacker with no free hand (Grapple). Otherwise the Damage option
// rolls an attack and Bludgeoning damage, and the Grapple and Shove options resolve
// the target's save, a Dexterity save gaining the cover bonus once. A miss, a saved
// target and AttackNotPossible-like refusals are results, not errors.
//
// It records the shared signals with the Unarmed Strike extras and at most one debug
// log. It returns an error matching srderr.ErrInvalidInput, before any signal, for
// the zero use case or command, and ctx.Err() unwrapped, before any signal, for a
// cancelled ctx. It returns a *srderr.PortError (error type "dice_roller_error") with
// a nil result when the roller fails, panics or answers another expression; the
// failure is recorded once and never logged.
func (u ResolveUnarmedStrike) Run(ctx context.Context, cmd ResolveUnarmedStrikeCommand) (ResolveUnarmedStrikeResult, error) {
	if u.flow.roller == nil || cmd.p.Option == nil {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationResolveUnarmedStrike, spanResolveUnarmedStrike}
	res, err := run(ctx, u.in, op, func(context.Context) (ResolveUnarmedStrikeResult, error) { return u.flow.resolveUnarmedStrike(cmd) },
		func(r ResolveUnarmedStrikeResult) description { return describeUnarmedStrike(cmd, r) })
	if err != nil {
		return nil, err
	}
	u.tel.WithDefaults().Logger.Log(ctx, telemetry.LevelDebug, msgUnarmedStrikeResolved, func() []telemetry.Attr {
		return []telemetry.Attr{telemetry.String(attrResult, describeUnarmedStrike(cmd, res).Name)}
	})
	return res, nil
}

// describeUnarmedStrike classifies a result: its class, srd5e.combat.result and the
// extras of attacks.md section 7. The d20 extras are left out when no d20 was rolled.
func describeUnarmedStrike(cmd ResolveUnarmedStrikeCommand, r ResolveUnarmedStrikeResult) description {
	option := valueUnarmedStrikeDamage
	switch cmd.p.Option.(type) {
	case UnarmedStrikeGrappleOption:
		option = valueUnarmedStrikeGrapple
	case UnarmedStrikeShoveOption:
		option = valueUnarmedStrikeShove
	}
	extras := []telemetry.Attr{
		telemetry.String(attrUnarmedStrikeOption, option),
		telemetry.Bool(attrAttackAutomaticCritical, cmd.p.AutomaticCritical == AutomaticCriticalAutomatic),
	}
	saveMode := func(s UnarmedStrikeSave) {
		if v, ok := s.(UnarmedStrikeSaveRolled); ok {
			extras = append(extras, telemetry.String(d20.AttrRollMode, v.result.Roll().Mode().String()))
		}
	}
	name := func(applied, resisted string, s UnarmedStrikeSave) string {
		if s.Outcome() == d20.Failure {
			return applied
		}
		return resisted
	}
	switch v := r.(type) {
	case UnarmedStrikeDamageResolved:
		n := valueResultDamageMiss
		switch {
		case v.critical:
			n = valueResultDamageCriticalHit
		case v.hit:
			n = valueResultDamageHit
		}
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()),
			telemetry.String(d20.AttrRollMode, v.attack.Roll().Mode().String()),
			telemetry.Bool(d20.AttrCriticalHit, v.attack.IsCriticalHit()))
		return description{Class: resultApplied, Name: n, Extras: extras}
	case UnarmedStrikeGrappleResolved:
		saveMode(v.save)
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()))
		return description{Class: resultApplied, Name: name(valueResultGrappleApplied, valueResultGrappleResisted, v.save), Extras: extras}
	case UnarmedStrikeShoveResolved:
		saveMode(v.save)
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()))
		return description{Class: resultApplied, Name: name(valueResultShoveApplied, valueResultShoveResisted, v.save), Extras: extras}
	case UnarmedStrikeNotPossible:
		extras = append(extras, telemetry.String(attrAttackCover, v.cover.String()),
			telemetry.String(attrAttackReason, v.reason.String()))
		return description{Class: resultRefused, Name: valueResultNotPossible, Extras: extras}
	default:
		panic(fmt.Errorf("combat: unreachable unarmed strike result %T", r))
	}
}
