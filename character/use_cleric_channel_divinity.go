package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ClericChannelEffect is the Channel Divinity effect a Cleric creates, a closed set. Variants:
// DivineSparkEffect, TurnUndeadEffect, PreserveLifeEffect.
type ClericChannelEffect interface{ isClericChannelEffect() }

// DivineSparkEffect is the Divine Spark effect: the target, whether it heals or forces a Constitution
// saving throw, and for damage the Cleric's choice of Necrotic or Radiant. Construct it with a keyed literal.
type DivineSparkEffect struct {
	Target     DivineSparkTarget
	Mode       SparkMode
	DamageType content.DamageType
}

// TurnUndeadEffect is the Turn Undead effect: the creatures chosen (possibly none) and whether to roll Sear
// Undead (the default is not to). Construct it with a keyed literal.
type TurnUndeadEffect struct {
	Targets    []UndeadTarget
	SearUndead SearUndeadChoice
}

// PreserveLifeEffect is the Life Domain's Preserve Life effect: the Bloodied creatures chosen (possibly none) with the
// Hit Points allocated to each. Construct it with a keyed literal.
type PreserveLifeEffect struct{ Targets []PreserveLifeTarget }

func (DivineSparkEffect) isClericChannelEffect()  {}
func (TurnUndeadEffect) isClericChannelEffect()   {}
func (PreserveLifeEffect) isClericChannelEffect() {}

// UseClericChannelDivinityCommand is the input of UseClericChannelDivinity.Run. Subclass is the subclass the Cleric
// holds ("" for none). SpellSaveDC is the Cleric's, from the Spellcasting feature; Wisdom is the Wisdom modifier. Construct
// it with a keyed literal. Run does not alter it.
type UseClericChannelDivinityCommand struct {
	Character   Character
	Subclass    content.SubclassID
	Effect      ClericChannelEffect
	SpellSaveDC SpellSaveDC
	Wisdom      ability.Modifier
}

// ClericChannelOutcome is the data of the effect a Channel Divinity use created, a closed set. Variants:
// DivineSparkOutcome, TurnUndeadOutcome, PreserveLifeOutcome.
type ClericChannelOutcome interface{ isClericChannelOutcome() }

// DivineSparkOutcome is the data of a Divine Spark: the resolution, the target and, in damage mode, the Constitution
// saving throw to resolve elsewhere.
type DivineSparkOutcome struct {
	resolution DivineSparkResolution
	target     string
	dc         d20.DifficultyClass
}

// Resolution returns the amounts of the spark.
func (o DivineSparkOutcome) Resolution() DivineSparkResolution { return o.resolution }

// Target returns the host's key of the target.
func (o DivineSparkOutcome) Target() string { return o.target }

// SavingThrow returns the Constitution saving throw and its DC (the Cleric's spell save DC) for damage mode, and false
// in healing mode.
func (o DivineSparkOutcome) SavingThrow() (ability.Ability, d20.DifficultyClass, bool) {
	return ability.Constitution, o.dc, o.resolution.mode == SparkDamage
}

// TurnUndeadOutcome is the data of a Turn Undead: the targets, the Wisdom saving throw each makes, what a failed save
// does and how the effect ends early.
type TurnUndeadOutcome struct {
	targets []string
	dc      d20.DifficultyClass
	sear    SearUndeadDamage
	hasSear bool
}

// Targets returns the host's keys of the targets, in command order, as a new slice.
func (o TurnUndeadOutcome) Targets() []string { return append([]string{}, o.targets...) }

// SavingThrow returns the Wisdom saving throw and its DC (the Cleric's spell save DC).
func (o TurnUndeadOutcome) SavingThrow() (ability.Ability, d20.DifficultyClass) {
	return ability.Wisdom, o.dc
}

// ConditionsOnFailedSave returns Frightened and Incapacitated.
func (TurnUndeadOutcome) ConditionsOnFailedSave() []content.ConditionKind {
	return TurnUndead{}.Conditions()
}

// DurationMinutes returns the duration of the conditions: 1 minute.
func (TurnUndeadOutcome) DurationMinutes() int { return 1 }

// MovesAwayFromCleric reports whether the creature tries to move as far from the Cleric as it can on its turns.
func (TurnUndeadOutcome) MovesAwayFromCleric() bool { return true }

// EarlyEndings returns the ways the effect ends early on a creature.
func (TurnUndeadOutcome) EarlyEndings() []TurnUndeadEnding { return TurnUndead{}.EarlyEndings() }

// SearUndead returns the Sear Undead damage applied to each Undead that fails its save, and false when it was not rolled.
func (o TurnUndeadOutcome) SearUndead() (SearUndeadDamage, bool) { return o.sear, o.hasSear }

// PreserveLifeAllocation is the Hit Points one target regains from Preserve Life.
type PreserveLifeAllocation struct {
	Key       string
	HitPoints int
}

// PreserveLifeOutcome is the data of a Preserve Life: the pool and the allocation to each target.
type PreserveLifeOutcome struct {
	pool        int
	allocations []PreserveLifeAllocation
}

// Pool returns the Hit Points that could be divided: five times the Cleric level.
func (o PreserveLifeOutcome) Pool() int { return o.pool }

// Allocations returns the Hit Points each target regains, in command order, as a new slice; unspent Hit Points are lost.
func (o PreserveLifeOutcome) Allocations() []PreserveLifeAllocation {
	return slices.Clone(o.allocations)
}

func (DivineSparkOutcome) isClericChannelOutcome()  {}
func (TurnUndeadOutcome) isClericChannelOutcome()   {}
func (PreserveLifeOutcome) isClericChannelOutcome() {}

// UseClericChannelDivinityResult is the closed set of outcomes of UseClericChannelDivinity.Run. Variants:
// UseClericChannelDivinityApplied, UseClericChannelDivinityRejected, UseClericChannelDivinityNotACleric.
type UseClericChannelDivinityResult interface{ isUseClericChannelDivinityResult() }

// UseClericChannelDivinityApplied is the outcome of a use. Only Run builds one.
type UseClericChannelDivinityApplied struct {
	character Character
	outcome   ClericChannelOutcome
	resources ClericResources
	events    []Event
}

// Character returns the new Character, with one Channel Divinity use expended.
func (r UseClericChannelDivinityApplied) Character() Character { return r.character }

// Outcome returns the data of the effect.
func (r UseClericChannelDivinityApplied) Outcome() ClericChannelOutcome { return r.outcome }

// Resources returns the new resources.
func (r UseClericChannelDivinityApplied) Resources() ClericResources { return r.resources }

// Events returns a new slice holding the ClericChannelDivinityUsed event.
func (r UseClericChannelDivinityApplied) Events() []Event { return slices.Clone(r.events) }

// UseClericChannelDivinityRejected is the outcome of a use a rule refused: the Character is unchanged, there are no events
// and no use is expended. Reason is the first reason that holds; Effect, Target, Pool, Sum and Room name what the reason is
// about when it has one. Construct it with a keyed literal.
type UseClericChannelDivinityRejected struct {
	Reason ClericReason
	// Effect is the effect asked for ("divine_spark", "turn_undead", "preserve_life").
	Effect string
	// Target is the host's key of the target the reason names, if any.
	Target string
	// Room is the target's room, for allocation_exceeds_room.
	Room int
	// Pool and Sum are the pool and the allocated total, for allocation_exceeds_pool.
	Pool, Sum int
}

// UseClericChannelDivinityNotACleric is the outcome for a character with no Cleric levels.
type UseClericChannelDivinityNotACleric struct{}

func (UseClericChannelDivinityApplied) isUseClericChannelDivinityResult()    {}
func (UseClericChannelDivinityRejected) isUseClericChannelDivinityResult()   {}
func (UseClericChannelDivinityNotACleric) isUseClericChannelDivinityResult() {}

// UseClericChannelDivinity is the use case that uses a Cleric's Channel Divinity (SRD 5.2.1, "Level 2: Channel
// Divinity", p. 37): Divine Spark, Turn Undead (with Sear Undead) or the Life Domain's Preserve Life. It checks the use, rolls
// the d8s through the dice port only for a use that is allowed and emits the srd5e.character.feature_used signals. It holds no
// rules logic. Create it with NewUseClericChannelDivinity; the zero value is not usable. It is safe for concurrent use only if
// its roller, telemetry and clock are.
type UseClericChannelDivinity struct {
	progression ClericProgression
	roller      Roller
	sig         featureSignals
	set         bool
}

// NewUseClericChannelDivinity returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an
// error matching srderr.ErrInvalidInput for the zero progression or a nil roller.
func NewUseClericChannelDivinity(progression ClericProgression, roller Roller, opts ...Option) (UseClericChannelDivinity, error) {
	if progression.IsZero() || roller == nil {
		return UseClericChannelDivinity{}, invalidInput("use Channel Divinity without a Cleric progression or a dice roller")
	}
	return UseClericChannelDivinity{progression: progression, roller: roller, sig: newFeatureSignals(newOptions(opts)), set: true}, nil
}

func clericEffectName(e ClericChannelEffect) string {
	switch e.(type) {
	case DivineSparkEffect:
		return ValueFeatureDivineSpark
	case TurnUndeadEffect:
		return ValueFeatureTurnUndead
	case PreserveLifeEffect:
		return ValueFeaturePreserveLife
	}
	return ""
}

// Run uses Channel Divinity for cmd.Character. The first reason that holds is reported, in the order of class-cleric.md table
// 4.14.1: below_level_2, effect_not_available, no_uses_remaining, then by effect.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a nil
// effect, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. It returns an error matching
// srderr.ErrInvalidInput for a negative distance, a Hit Point value or an effect the rules cannot take, and a
// *srderr.PortError (error type "dice_roller_error") with a nil result when the roller fails, panics or breaks its contract; the
// Character is unchanged and the failure is recorded once and never logged. A refused use is a result.
func (u UseClericChannelDivinity) Run(ctx context.Context, cmd UseClericChannelDivinityCommand) (UseClericChannelDivinityResult, error) {
	name := clericEffectName(cmd.Effect)
	if !u.set || cmd.Character.IsZero() || name == "" {
		return nil, invalidInput("use Channel Divinity: zero use case or character, or an unknown effect")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, run := u.sig.start(ctx, SpanUseClericChannelDivinity, ValueClassCleric, name)
	defer run.end(ctx)
	level, ok := clericLevelOf(cmd.Character)
	if !ok {
		run.rejected(ctx, ClericNotACleric.String())
		return UseClericChannelDivinityNotACleric{}, nil
	}
	stats := u.progression.StatsAt(level)
	resources, err := readClericResources(cmd.Character, stats.channel)
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	reject := func(rj UseClericChannelDivinityRejected) (UseClericChannelDivinityResult, error) {
		rj.Effect = name
		run.rejected(ctx, rj.Reason.String())
		return rj, nil
	}
	switch {
	case level.n < 2:
		return reject(UseClericChannelDivinityRejected{Reason: ClericBelowLevel2})
	case name == ValueFeaturePreserveLife && (cmd.Subclass != LifeDomainSubclassID || level.n < discipleOfLifeLevel):
		return reject(UseClericChannelDivinityRejected{Reason: ClericEffectNotAvailable})
	case resources.channelRemaining < 1:
		return reject(UseClericChannelDivinityRejected{Reason: ClericNoUsesRemaining})
	}
	outcome, rj, err := u.createEffect(cmd, level, stats)
	if err != nil {
		if et := srderr.ErrorType(err); et != "" {
			run.portFailure(ctx, err, et)
		} else {
			run.commandError(err)
		}
		return nil, err
	}
	if rj.Reason != 0 {
		return reject(rj)
	}
	resources.channelRemaining--
	next, err := WithFeatureState(cmd.Character, ClericResourcesKey(), resources)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		run.commandError(err)
		return nil, err
	}
	run.succeeded(ctx)
	return UseClericChannelDivinityApplied{character: next, outcome: outcome, resources: resources,
		events: []Event{ClericChannelDivinityUsed{id: next.ID(), effect: name, remaining: resources.channelRemaining}}}, nil
}

// createEffect checks the effect-specific rules and, when they hold, rolls and builds the effect data. A zero Reason in the
// returned rejection means the effect was created.
func (u UseClericChannelDivinity) createEffect(cmd UseClericChannelDivinityCommand, level ClericLevel, stats ClericLevelStats) (ClericChannelOutcome, UseClericChannelDivinityRejected, error) {
	dc := cmd.SpellSaveDC.DifficultyClass()
	switch e := cmd.Effect.(type) {
	case DivineSparkEffect:
		spark := DivineSpark{}
		reason, err := spark.TargetProblem(e.Target)
		if err != nil {
			return nil, UseClericChannelDivinityRejected{}, err
		}
		if reason == 0 && e.Mode == SparkDamage && !spark.AllowsDamageType(e.DamageType) {
			reason = ClericDamageTypeNotAllowed
		}
		if reason != 0 {
			return nil, UseClericChannelDivinityRejected{Reason: reason, Target: e.Target.Key}, nil
		}
		var rolls []int
		if e.Mode == SparkHealing && (SupremeHealing{}).Applies(level, cmd.Subclass) {
			rolls = slices.Repeat([]int{8}, stats.sparkD8)
		} else if rolls, err = rollDiceValues(u.roller, stats.sparkD8, dice.D8()); err != nil {
			return nil, UseClericChannelDivinityRejected{}, err
		}
		res, err := spark.Resolve(level, rolls, cmd.Wisdom, e.Mode, e.DamageType)
		return DivineSparkOutcome{resolution: res, target: e.Target.Key, dc: dc}, UseClericChannelDivinityRejected{}, err
	case TurnUndeadEffect:
		sear := SearUndead{}
		n, gained := sear.Dice(level, cmd.Wisdom)
		if e.SearUndead == SearUndeadUse && !gained {
			return nil, UseClericChannelDivinityRejected{Reason: ClericSearUndeadNotGained}, nil
		}
		reason, key, err := TurnUndead{}.TargetProblem(e.Targets)
		if err != nil || reason != 0 {
			return nil, UseClericChannelDivinityRejected{Reason: reason, Target: key}, err
		}
		out := TurnUndeadOutcome{targets: []string{}, dc: dc}
		for _, t := range e.Targets {
			out.targets = append(out.targets, t.Key)
		}
		if e.SearUndead == SearUndeadUse {
			rolls, err := rollDiceValues(u.roller, n, dice.D8())
			if err != nil {
				return nil, UseClericChannelDivinityRejected{}, err
			}
			if out.sear, err = sear.Resolve(level, cmd.Wisdom, rolls); err != nil {
				return nil, UseClericChannelDivinityRejected{}, err
			}
			out.hasSear = true
		}
		return out, UseClericChannelDivinityRejected{}, nil
	default: // PreserveLifeEffect: clericEffectName admitted only the three effects
		targets := cmd.Effect.(PreserveLifeEffect).Targets
		problem, err := PreserveLife{}.Validate(level, targets)
		if err != nil || problem.Reason != 0 {
			return nil, UseClericChannelDivinityRejected{Reason: problem.Reason, Target: problem.Target, Room: problem.Room, Pool: problem.Pool, Sum: problem.Sum}, err
		}
		out := PreserveLifeOutcome{pool: PreserveLife{}.Pool(level), allocations: []PreserveLifeAllocation{}}
		for _, t := range targets {
			out.allocations = append(out.allocations, PreserveLifeAllocation{Key: t.Key, HitPoints: t.Allocation})
		}
		return out, UseClericChannelDivinityRejected{}, nil
	}
}
