package combat

import (
	"context"
	"fmt"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// SetUpMonsterCombatantCommand is the input of SetUpMonsterCombatant.Run: the stat block
// of one creature, the size when it has several options, how its Hit Points are decided
// and the options of NewMonsterSetup. Build it with a keyed literal. HitPoints is
// required; a zero ZeroHitPointsRule means DiesInstantly and a zero InitiativeMode means
// MonsterInitiativeRolled.
type SetUpMonsterCombatantCommand struct {
	Monster           content.Monster
	Size              content.CreatureSize
	HitPoints         MonsterHitPointsMethod
	ZeroHitPointsRule ZeroHitPointsRule
	InitiativeMode    MonsterInitiativeMode
	ExtraAffinities   DamageAffinities
}

// SetUpMonsterCombatantResult is the closed set of outcomes of SetUpMonsterCombatant.Run.
// Variants: SetUpMonsterCombatantApplied, SetUpMonsterCombatantInvalid.
type SetUpMonsterCombatantResult interface{ isSetUpMonsterCombatantResult() }

// SetUpMonsterCombatantApplied is the setup of one creature and the Hit Points choice it
// was made with (the roll, for the dice). Only Run builds one.
type SetUpMonsterCombatantApplied struct {
	setup     MonsterSetup
	hitPoints ResolvedMonsterHitPoints
}

// Setup returns the setup.
func (r SetUpMonsterCombatantApplied) Setup() MonsterSetup { return r.setup }

// HitPoints returns the Hit Points choice: PrintedAverageHitPoints or the
// RolledMonsterHitPoints that was rolled.
func (r SetUpMonsterCombatantApplied) HitPoints() ResolvedMonsterHitPoints { return r.hitPoints }

// Equal reports whether r and o hold equal setups made with the same Hit Points choice.
func (r SetUpMonsterCombatantApplied) Equal(o SetUpMonsterCombatantApplied) bool {
	if !r.setup.Equal(o.setup) {
		return false
	}
	a, aRolled := r.hitPoints.(RolledMonsterHitPoints)
	b, bRolled := o.hitPoints.(RolledMonsterHitPoints)
	if aRolled || bRolled {
		return aRolled && bRolled && a.Equal(b)
	}
	return r.hitPoints == o.hitPoints
}

// SetUpMonsterCombatantInvalid: the monster or the size cannot be used, found before any
// roll. It holds the problems in the order of the enum, never none.
type SetUpMonsterCombatantInvalid struct{ problems []MonsterSetupProblem }

// Problems returns a copy of the problems.
func (r SetUpMonsterCombatantInvalid) Problems() []MonsterSetupProblem {
	return slices.Clone(r.problems)
}

func (SetUpMonsterCombatantApplied) isSetUpMonsterCombatantResult() {}
func (SetUpMonsterCombatantInvalid) isSetUpMonsterCombatantResult() {}

// SetUpMonsterCombatant is the use case that turns a stat block into the inputs of one
// creature: it rolls the Hit Point Dice when asked, through the dice port, and composes
// the pure MonsterSetup (SRD 5.2.1, "Stat Block Overview", p. 254). It holds no rule.
// Create it with NewSetUpMonsterCombatant; the zero value is not usable.
//
// The monster is a content.Monster the host fetched: this use case reads no content
// source.
type SetUpMonsterCombatant struct {
	in     instrumenter
	tel    telemetry.Telemetry
	roller Roller
}

// NewSetUpMonsterCombatant returns the use case rolling through roller. Options:
// WithTelemetry, WithClock. It returns an error matching srderr.ErrInvalidInput for a
// nil roller.
func NewSetUpMonsterCombatant(roller Roller, opts ...Option) (SetUpMonsterCombatant, error) {
	if roller == nil {
		return SetUpMonsterCombatant{}, invalidInput("monster setup without a dice roller")
	}
	o := newOptions(opts)
	return SetUpMonsterCombatant{newInstrumenter(o.tel, o.clock), o.tel, roller}, nil
}

// Run sets up one creature. It checks the size and the numbers of the stat block first and
// rolls nothing when they are wrong; with UseAverageHitPoints it makes no request of the
// roller, and with RollHitPointDice exactly one, for the printed Hit Point Dice including
// the flat modifier. It records the shared signals under the operation
// set_up_monster_combatant, the extra srd5e.combat.monster_hit_points_method on an applied
// result, and srd5e.combat.monster_hit_points_clamped when a rolled total was raised to 1.
// srd5e.combat.result is set_up_average or set_up_rolled (class applied), or
// size_choice_required, size_not_offered, content_number_out_of_range or limit_exceeded (more than 64 parts, spells, spell groups, Multiattack variants, groups or replacements; class invalid:
// error status, no errors count). No event is produced.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero
// use case, a command with no Hit Points method or an option value that is not defined,
// and ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller
// fails, panics or answers another expression; the failure is recorded once and never
// logged. A size or content problem, the zero Monster included, is an Invalid result.
func (u SetUpMonsterCombatant) Run(ctx context.Context, cmd SetUpMonsterCombatantCommand) (SetUpMonsterCombatantResult, error) {
	c := setupConfig{size: cmd.Size, rule: cmd.ZeroHitPointsRule, mode: cmd.InitiativeMode, extra: cmd.ExtraAffinities}
	if c.rule == 0 {
		c.rule = DiesInstantly
	}
	if c.mode == 0 {
		c.mode = MonsterInitiativeRolled
	}
	if u.roller == nil || cmd.HitPoints == nil || !c.rule.Valid() || !c.mode.Valid() {
		return nil, invalidInput("zero use case, no Hit Points method or an undefined option")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res, err := run(ctx, u.in, combatOperation{valueOperationSetUpMonsterCombatant, spanSetUpMonsterCombatant},
		func(context.Context) (SetUpMonsterCombatantResult, error) {
			size, set := checkSetup(cmd.Monster, c)
			if problems := set.list(); len(problems) > 0 {
				return SetUpMonsterCombatantInvalid{problems}, nil
			}
			var hp ResolvedMonsterHitPoints = PrintedAverageHitPoints{}
			if _, rolls := cmd.HitPoints.(RollHitPointDice); rolls {
				roll, err := dice.Roll(u.roller, cmd.Monster.HitPoints().Dice())
				if err != nil {
					return nil, err
				}
				hp = RolledMonsterHitPoints{roll}
			}
			maximum, raised, _ := monsterMaximum(cmd.Monster.HitPoints(), hp)
			return SetUpMonsterCombatantApplied{assembleSetup(cmd.Monster, c, size, maximum, raised), hp}, nil
		},
		func(r SetUpMonsterCombatantResult) description {
			switch v := r.(type) {
			case SetUpMonsterCombatantApplied:
				name, method := valueResultSetUpAverage, valueMonsterHitPointsMethodAverage
				if _, rolled := v.hitPoints.(RolledMonsterHitPoints); rolled {
					name, method = valueResultSetUpRolled, valueMonsterHitPointsMethodRolled
				}
				return description{Class: resultApplied, Name: name, Extras: []telemetry.Attr{telemetry.String(attrMonsterHitPointsMethod, method)}}
			case SetUpMonsterCombatantInvalid:
				return description{Class: resultInvalid, Name: invalidResultName(v.problems[0])}
			default:
				panic(fmt.Errorf("combat: unreachable set up monster combatant result %T", r))
			}
		})
	if err != nil {
		return nil, err
	}
	if a, ok := res.(SetUpMonsterCombatantApplied); ok && a.setup.raisedToOne {
		u.tel.WithDefaults().Meter.Counter(telemetry.Descriptor{Name: metricMonsterHitPointsClamped, Unit: "{roll}",
			Description: "Rolled monster Hit Point totals raised to 1."}).Add(ctx, 1)
	}
	return res, nil
}

// invalidResultName returns the srd5e.combat.result of an Invalid result whose first
// problem is p. Run finds only these four before any roll.
func invalidResultName(p MonsterSetupProblem) string {
	switch p {
	case SizeChoiceRequired:
		return valueResultSizeChoiceRequired
	case SizeNotOffered:
		return valueResultSizeNotOffered
	case ContentNumberOutOfRange:
		return valueResultContentNumberOutOfRange
	case SetupLimitExceeded:
		return valueResultLimitExceeded
	case RolledExpressionMismatch, SetupArgumentInvalid:
		fallthrough
	default:
		panic(fmt.Errorf("combat: unreachable monster setup problem %d", int(p)))
	}
}
