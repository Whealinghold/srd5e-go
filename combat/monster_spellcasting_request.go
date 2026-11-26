package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// MonsterCastRequest is the closed set of ways a monster casts a spell: from its
// Spellcasting list or through a stat-block part. Neither has a level parameter that
// could raise the level: a monster never casts at a higher level (SRD 5.2.1,
// "Spellcasting", p. 257). Variants: MonsterListCast, MonsterPartCast. Consumers
// switch over it exhaustively. Comparable.
type MonsterCastRequest interface{ isMonsterCastRequest() }

// MonsterListCast is a cast of one spell of the Spellcasting list, named by its
// SpellUseRef. A spell with several casting time modes needs the mode in force. The
// zero MonsterListCast casts the first spell of the first group with no mode and is
// valid.
type MonsterListCast struct {
	spell SpellUseRef
	mode  content.CastingTimeMode
}

// NewMonsterListCast returns the cast of the list entry at spell. mode is the zero
// content.CastingTimeMode for a spell with one casting time.
func NewMonsterListCast(spell SpellUseRef, mode content.CastingTimeMode) MonsterListCast {
	return MonsterListCast{spell: spell, mode: mode}
}

func (MonsterListCast) isMonsterCastRequest() {}

// Spell returns the list entry.
func (c MonsterListCast) Spell() SpellUseRef { return c.spell }

// CastingTimeMode returns the casting time mode, the zero mode when none.
func (c MonsterListCast) CastingTimeMode() content.CastingTimeMode { return c.mode }

// MonsterPartCast is a cast of a spell by a stat-block part such as "Misty Step
// (3/Day)": the Bonus Action or Reaction of the part, immediate whatever the spell's
// printed casting time. The zero MonsterPartCast is invalid.
type MonsterPartCast struct {
	part        LimitedUsePartRef
	castAtLevel int
	own         PartSpellcasting
}

// NewMonsterPartCast returns the cast through part at the stated version castAtLevel
// (0 for none, else 1 to 9) with the part's own spellcasting values; the zero
// PartSpellcasting means the part prints none and the trait supplies the ability.
//
// It returns an error matching srderr.ErrInvalidInput for a zero part reference and
// a level outside 0 and 1 to 9. It never panics.
func NewMonsterPartCast(part LimitedUsePartRef, castAtLevel int, own PartSpellcasting) (MonsterPartCast, error) {
	if part.IsZero() || (castAtLevel != 0 && (castAtLevel < content.MonsterMinSpellLevel || castAtLevel > content.MonsterMaxSpellLevel)) {
		return MonsterPartCast{}, invalidInput("a part cast needs a part and a stated level of 0 or 1 to 9")
	}
	return MonsterPartCast{part: part, castAtLevel: castAtLevel, own: own}, nil
}

func (MonsterPartCast) isMonsterCastRequest() {}

// Part returns the part that casts.
func (c MonsterPartCast) Part() LimitedUsePartRef { return c.part }

// CastAtLevel returns the stated version level and whether one is stated.
func (c MonsterPartCast) CastAtLevel() (int, bool) { return c.castAtLevel, c.castAtLevel != 0 }

// Spellcasting returns the part's own spellcasting values and whether it prints any.
func (c MonsterPartCast) Spellcasting() (PartSpellcasting, bool) { return c.own, !c.own.IsZero() }

// PartSpellcastingFields holds the members of a PartSpellcasting. Construct it with
// a keyed literal.
type PartSpellcastingFields struct {
	// Ability is the part's spellcasting ability (required).
	Ability ability.Ability
	// SaveDC is the part's printed spell save DC; it counts only when HasSaveDC is set.
	SaveDC int
	// HasSaveDC says the part prints a spell save DC.
	HasSaveDC bool
	// AttackBonus is the part's printed spell attack bonus; it counts only when
	// HasAttackBonus is set.
	AttackBonus int
	// HasAttackBonus says the part prints a spell attack bonus.
	HasAttackBonus bool
	// Components is the part's components statement, such as "requiring no Material
	// components"; "" means none. It is read when the cast is made.
	Components string
}

// PartSpellcasting is the ability, save DC, spell attack bonus and components
// statement a stat-block part prints for a spell it casts (SRD 5.2.1, "Pit Fiend",
// p. 315; "Cloaker", p. 274). The zero value is invalid and means the part prints
// none. Comparable and safe for concurrent use.
type PartSpellcasting struct{ f PartSpellcastingFields }

// NewPartSpellcasting returns the values of f.
//
// It returns an error matching srderr.ErrInvalidInput for an invalid ability, a DC
// outside content.MonsterMinSaveDC to content.MonsterMaxNumber and an attack bonus
// outside plus or minus content.MonsterMaxNumber. It never panics.
func NewPartSpellcasting(f PartSpellcastingFields) (PartSpellcasting, error) {
	if !f.Ability.Valid() || (f.HasSaveDC && !saveDCInRange(f.SaveDC)) || (f.HasAttackBonus && !bonusInRange(f.AttackBonus)) {
		return PartSpellcasting{}, invalidInput("part spellcasting needs an ability and numbers in range")
	}
	if !f.HasSaveDC {
		f.SaveDC = 0
	}
	if !f.HasAttackBonus {
		f.AttackBonus = 0
	}
	return PartSpellcasting{f: f}, nil
}

// Ability returns the spellcasting ability.
func (p PartSpellcasting) Ability() ability.Ability { return p.f.Ability }

// SaveDC returns the printed spell save DC and whether there is one.
func (p PartSpellcasting) SaveDC() (int, bool) { return p.f.SaveDC, p.f.HasSaveDC }

// AttackBonus returns the printed spell attack bonus and whether there is one.
func (p PartSpellcasting) AttackBonus() (int, bool) { return p.f.AttackBonus, p.f.HasAttackBonus }

// Components returns the components statement, "" when none.
func (p PartSpellcasting) Components() string { return p.f.Components }

// IsZero reports whether p is the invalid zero value.
func (p PartSpellcasting) IsZero() bool { return p.f.Ability == 0 }

// MonsterCastContext is what the host says about the monster at the moment of a cast:
// whether it can speak, whether it can make gestures and whether it can take actions
// (SRD 5.2.1, "Verbal (V)", "Somatic (S)", pp. 105-106; "Incapacitated", p. 184). The
// zero value is invalid. Comparable and safe for concurrent use.
type MonsterCastContext struct {
	canSpeak, canMakeGestures bool
	actions                   ConditionActionAvailability
}

// NewMonsterCastContext returns the context.
//
// It returns an error matching srderr.ErrInvalidInput for an undefined actions
// value. It never panics.
func NewMonsterCastContext(canSpeak, canMakeGestures bool, actions ConditionActionAvailability) (MonsterCastContext, error) {
	if actions != ConditionActionAvailabilityAvailable && actions != ConditionActionAvailabilityNone {
		return MonsterCastContext{}, invalidInput("a cast context needs a defined action availability")
	}
	return MonsterCastContext{canSpeak: canSpeak, canMakeGestures: canMakeGestures, actions: actions}, nil
}

// CanSpeak reports whether the monster can speak.
func (c MonsterCastContext) CanSpeak() bool { return c.canSpeak }

// CanMakeGestures reports whether the monster can make the gestures of a Somatic
// component.
func (c MonsterCastContext) CanMakeGestures() bool { return c.canMakeGestures }

// Actions returns whether the monster can take actions.
func (c MonsterCastContext) Actions() ConditionActionAvailability { return c.actions }

// IsZero reports whether c is the invalid zero value.
func (c MonsterCastContext) IsZero() bool { return c.actions == 0 }

// TargetLimit says whom a cast may affect: only the caster, or anyone.
// The zero value is invalid. Comparable and safe for concurrent use.
type TargetLimit int

// The two limits.
const (
	// TargetLimitSelfOnly: the spell affects only the monster ("self only").
	TargetLimitSelfOnly TargetLimit = iota + 1
	// TargetLimitUnrestricted: this spec does not limit the targets.
	TargetLimitUnrestricted
)

// String returns "self_only", "unrestricted" or "unknown".
func (l TargetLimit) String() string { return enumName([]string{"self_only", "unrestricted"}, int(l)) }
