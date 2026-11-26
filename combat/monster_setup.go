package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// MonsterSetupProblem is one reason a MonsterSetup could not be built. The zero value is
// not a problem.
type MonsterSetupProblem int

// The problems NewMonsterSetup reports, in the order they are reported.
const (
	// SizeChoiceRequired: the monster has several size options and none was chosen.
	SizeChoiceRequired MonsterSetupProblem = iota + 1
	// SizeNotOffered: the chosen size is not among the monster's size options.
	SizeNotOffered
	// RolledExpressionMismatch: the roll is not a roll of the monster's Hit Point Dice.
	RolledExpressionMismatch
	// ContentNumberOutOfRange: a number of the stat block is outside the range combat
	// relies on, or the monster is the zero Monster (plan P13).
	ContentNumberOutOfRange
	// SetupArgumentInvalid: no Hit Points choice, or an option value that is not
	// defined.
	SetupArgumentInvalid
	// SetupLimitExceeded: the stat block holds more than 64 parts, spells, spell groups,
	// Multiattack variants, attack groups or replacement rules (plan P10). Its value is
	// limit_exceeded.
	SetupLimitExceeded
)

// String returns the snake_case name of p, or "unknown".
func (p MonsterSetupProblem) String() string {
	return enumName([]string{"size_choice_required", "size_not_offered", "rolled_expression_mismatch",
		"content_number_out_of_range", "setup_argument_invalid", "limit_exceeded"}, int(p))
}

// MonsterSetupError is the error NewMonsterSetup returns: every problem found, once
// each, in the order of the enum. It matches srderr.ErrInvalidInput with errors.Is, is
// never built with no problems and carries no display text.
type MonsterSetupError struct{ problems []MonsterSetupProblem }

// Problems returns a copy of the problems.
func (e *MonsterSetupError) Problems() []MonsterSetupProblem { return slices.Clone(e.problems) }

// Error renders the error. The text is not a contract.
func (e *MonsterSetupError) Error() string {
	names := make([]string, len(e.problems))
	for i, p := range e.problems {
		names[i] = p.String()
	}
	return "combat: invalid monster setup: " + strings.Join(names, ", ")
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *MonsterSetupError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *MonsterSetupError listing the same problems.
func (e *MonsterSetupError) Is(target error) bool {
	t, ok := target.(*MonsterSetupError)
	return ok && t != nil && e != nil && slices.Equal(t.problems, e.problems)
}

// setupConfig is what the options of NewMonsterSetup set.
type setupConfig struct {
	size  content.CreatureSize
	rule  ZeroHitPointsRule
	mode  MonsterInitiativeMode
	extra DamageAffinities
}

// SetupOption sets an optional value of NewMonsterSetup. A nil option is ignored.
type SetupOption func(*setupConfig)

// WithSize chooses the size of a monster with several size options; the zero size
// means none chosen. A size equal to the only option is accepted.
func WithSize(s content.CreatureSize) SetupOption { return func(c *setupConfig) { c.size = s } }

// WithZeroHitPointsRule sets what dropping to 0 Hit Points does to the creature; the
// default is DiesInstantly (SRD 5.2.1, "Monster Death", p. 17).
func WithZeroHitPointsRule(r ZeroHitPointsRule) SetupOption {
	return func(c *setupConfig) { c.rule = r }
}

// WithInitiativeMode sets whether the Initiative is rolled or fixed; the default is
// MonsterInitiativeRolled.
func WithInitiativeMode(m MonsterInitiativeMode) SetupOption {
	return func(c *setupConfig) { c.mode = m }
}

// WithExtraAffinities adds affinities to those of the stat block, by union. It is how a
// host applies a described or qualified defence it has decided holds.
func WithExtraAffinities(a DamageAffinities) SetupOption {
	return func(c *setupConfig) { c.extra = a }
}

// newSetupConfig returns the defaults changed by opts.
func newSetupConfig(opts []SetupOption) setupConfig {
	c := setupConfig{rule: DiesInstantly, mode: MonsterInitiativeRolled}
	for _, o := range opts {
		if o != nil {
			o(&c)
		}
	}
	return c
}

// problemSet is a set of MonsterSetupProblem, one bit each.
type problemSet uint8

func (s *problemSet) add(p MonsterSetupProblem) {
	if p != 0 {
		*s |= 1 << p
	}
}

// list returns the problems in the order of the enum.
func (s problemSet) list() []MonsterSetupProblem {
	var out []MonsterSetupProblem
	for p := SizeChoiceRequired; p <= SetupLimitExceeded; p++ {
		if s&(1<<p) != 0 {
			out = append(out, p)
		}
	}
	return out
}

// checkSetup returns the size the setup of m takes and the problems found before any
// roll: the size, the numbers of the stat block and the option values. A monster with
// one size option takes it when none is chosen; with several it needs a choice, and the
// engine never picks (SRD 5.2.1, "Size", p. 254).
func checkSetup(m content.Monster, c setupConfig) (content.CreatureSize, problemSet) {
	var set problemSet
	options := m.Sizes()
	size := c.size
	switch {
	case size != 0:
		if !slices.Contains(options, size) {
			set.add(SizeNotOffered)
		}
	case len(options) == 1:
		size = options[0]
	case len(options) > 1:
		set.add(SizeChoiceRequired)
	}
	if monsterCollectionsInRange(m) != nil || monsterPlansInRange(m) != nil {
		set.add(SetupLimitExceeded)
	} else if !monsterNumbersOK(m) {
		set.add(ContentNumberOutOfRange)
	}
	if !c.rule.Valid() || !c.mode.Valid() {
		set.add(SetupArgumentInvalid)
	}
	return size, set
}

// MonsterSetup is one monster prepared for a fight: its Hit Points as a HitPointStatus,
// the conditions it is immune to, the Initiative plan, the Proficiency Bonus of its
// Challenge Rating, its speeds and size, and the defences and speeds it did not apply
// (SRD 5.2.1, "Stat Block Overview", p. 254). It is one creature: it holds no
// CombatantID, and several creatures of one stat block are several setups. It is
// immutable, holds the Monster by value, rolls nothing and reads no clock, so it is
// silent. The zero value is invalid (see IsZero). Not comparable with ==; use Equal.
// Safe for concurrent use.
type MonsterSetup struct {
	monster           content.Monster
	size              content.CreatureSize
	hitPoints         HitPointStatus
	raisedToOne       bool
	conditions        ConditionSet
	initiative        MonsterInitiativePlan
	pb                modifier.Modifier
	speeds            MovementSpeeds
	unappliedDefenses []UnappliedDefense
	unappliedSpeeds   []UnappliedSpeed
}

// NewMonsterSetup returns the setup of m with the Hit Points hp, the choice of exactly
// one of the printed average and a roll of the Hit Point Dice (SRD 5.2.1, "Hit Points",
// p. 255). The options default to a size taken from the only option, DiesInstantly, a
// rolled Initiative and no extra affinities.
//
// It returns a *MonsterSetupError (matching srderr.ErrInvalidInput) listing every
// problem found, once each, in the order of the enum, and no setup. It never panics.
func NewMonsterSetup(m content.Monster, hp ResolvedMonsterHitPoints, opts ...SetupOption) (MonsterSetup, error) {
	c := newSetupConfig(opts)
	size, set := checkSetup(m, c)
	maximum, raised, p := monsterMaximum(m.HitPoints(), hp)
	set.add(p)
	if problems := set.list(); len(problems) > 0 {
		return MonsterSetup{}, &MonsterSetupError{problems}
	}
	return assembleSetup(m, c, size, maximum, raised), nil
}

// MustMonsterSetup is NewMonsterSetup for literals and tests. It panics with the error
// NewMonsterSetup returns, and only then.
func MustMonsterSetup(m content.Monster, hp ResolvedMonsterHitPoints, opts ...SetupOption) MonsterSetup {
	s, err := NewMonsterSetup(m, hp, opts...)
	if err != nil {
		panic(err)
	}
	return s
}

// assembleSetup builds the setup from values checkSetup and monsterMaximum accepted:
// the maximum is at least 1 and the rule valid, so the status cannot fail.
func assembleSetup(m content.Monster, c setupConfig, size content.CreatureSize, maximum int, raised bool) MonsterSetup {
	aff, conditions, unappliedDefenses := monsterDefenses(m)
	speeds, unappliedSpeeds := monsterSpeeds(m)
	return MonsterSetup{
		monster:           m,
		size:              size,
		hitPoints:         MustHitPointStatus(maximum, c.rule, unionAffinities(aff, c.extra)),
		raisedToOne:       raised,
		conditions:        conditions,
		initiative:        monsterInitiative(m, c.mode),
		pb:                modifier.New(m.ProficiencyBonus()),
		speeds:            speeds,
		unappliedDefenses: unappliedDefenses,
		unappliedSpeeds:   unappliedSpeeds,
	}
}

// IsZero reports whether s is the invalid zero MonsterSetup.
func (s MonsterSetup) IsZero() bool { return s.monster.ID() == "" }

// Monster returns the stat block the setup was made from.
func (s MonsterSetup) Monster() content.Monster { return s.monster }

// Size returns the chosen size.
func (s MonsterSetup) Size() content.CreatureSize { return s.size }

// HitPointStatus returns the creature at full Hit Points: current equal to the maximum,
// no Temporary Hit Points and Alive.
func (s MonsterSetup) HitPointStatus() HitPointStatus { return s.hitPoints }

// HitPointsRaisedToOne reports whether a rolled total of 0 or less was raised to 1 (a
// "minus" Hit Point Dice block). It is always false for the printed average.
func (s MonsterSetup) HitPointsRaisedToOne() bool { return s.raisedToOne }

// Conditions returns the set holding the condition Immunities and no explicit condition.
func (s MonsterSetup) Conditions() ConditionSet { return s.conditions }

// Initiative returns the Initiative plan, the printed modifier or the printed score.
func (s MonsterSetup) Initiative() MonsterInitiativePlan { return s.initiative }

// ProficiencyBonus returns the Proficiency Bonus of the monster's Challenge Rating, +2
// to +9 (SRD 5.2.1, "Proficiency Bonus", p. 256).
func (s MonsterSetup) ProficiencyBonus() modifier.Modifier { return s.pb }

// Speeds returns the speeds that apply: the unqualified entries with one mode.
func (s MonsterSetup) Speeds() MovementSpeeds { return s.speeds }

// UnappliedDefenses returns a copy of the defences that were not applied, ordered by
// list and then by printed order.
func (s MonsterSetup) UnappliedDefenses() []UnappliedDefense {
	return slices.Clone(s.unappliedDefenses)
}

// UnappliedSpeeds returns a copy of the speeds that were not applied, in printed order.
func (s MonsterSetup) UnappliedSpeeds() []UnappliedSpeed { return slices.Clone(s.unappliedSpeeds) }

// Ref returns the reference to a monster combatant with id and this stat block. The
// setup holds no id, so one setup may be reused only where several creatures should
// have identical Hit Points.
func (s MonsterSetup) Ref(id CombatantID) CombatantRef {
	return CombatantRef{ID: id, Kind: MonsterCombatant{ID: s.monster.ID()}}
}

// Profile returns the profile of id for EstablishInitiativeOrder: the chosen size and
// the speeds.
func (s MonsterSetup) Profile(id CombatantID) CombatantProfile {
	return CombatantProfile{ID: id, Size: s.size, Speeds: s.speeds}
}

// Equal reports whether s and o hold the same monster, size, Hit Points, conditions,
// Initiative plan, Proficiency Bonus, speeds and unapplied lists.
func (s MonsterSetup) Equal(o MonsterSetup) bool {
	return s.monster.Equal(o.monster) && s.size == o.size && s.hitPoints == o.hitPoints &&
		s.raisedToOne == o.raisedToOne && s.conditions == o.conditions && s.initiative == o.initiative &&
		s.pb == o.pb && s.speeds == o.speeds && slices.Equal(s.unappliedDefenses, o.unappliedDefenses) &&
		slices.EqualFunc(s.unappliedSpeeds, o.unappliedSpeeds, UnappliedSpeed.Equal)
}
