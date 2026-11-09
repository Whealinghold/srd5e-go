package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// MetamagicFact names a fact about a spell a Metamagic option reads (class-sorcerer.md FR-123). The zero value is not a fact.
type MetamagicFact int

// The facts of MetamagicSpellFacts that can be marked unknown.
const (
	// MetamagicFactSavingThrow: whether the spell forces a saving throw, and on how many other creatures.
	MetamagicFactSavingThrow MetamagicFact = iota + 1
	// MetamagicFactRange: the range of the spell.
	MetamagicFactRange
	// MetamagicFactDuration: the duration of the spell.
	MetamagicFactDuration
	// MetamagicFactCastingTime: the casting time of the spell.
	MetamagicFactCastingTime
	// MetamagicFactComponents: the components of the spell.
	MetamagicFactComponents
	// MetamagicFactAttackRoll: the spell attack roll of the casting.
	MetamagicFactAttackRoll
	// MetamagicFactDamage: the damage types and dice of the spell.
	MetamagicFactDamage
	// MetamagicFactHigherSlotTarget: whether a higher-level slot targets an additional creature.
	MetamagicFactHigherSlotTarget
)

// String returns the snake_case name ("saving_throw", ...) or "unknown".
func (f MetamagicFact) String() string {
	names := []string{"saving_throw", "range", "duration", "casting_time", "components", "attack_roll", "damage", "higher_slot_target"}
	if f < 1 || int(f) > len(names) {
		return "unknown"
	}
	return names[f-1]
}

// MetamagicSpellFacts are the facts about a spell and its casting that the Metamagic options read, built by the host from the
// spell's content (Range, Duration, CastingTime, Components) and host-supplied mechanics (class-sorcerer.md section 4.8). A fact
// the host does not supply is listed in Unknown, and an option that needs it refuses with MetamagicSpellFactUnknown rather than
// assume (FR-123). Construct it with a keyed literal; the slices are copied by the functions that read it.
type MetamagicSpellFacts struct {
	Range       content.SpellRange
	Duration    content.SpellDuration
	CastingTime content.CastingTime
	Components  content.SpellComponents
	// ForcesSavingThrow says the spell forces a saving throw; OtherCreaturesSaving is how many creatures other than the caster must save.
	ForcesSavingThrow    bool
	OtherCreaturesSaving int
	// AttackRoll is the spell attack roll made for the casting, nil when the spell has none.
	AttackRoll *d20.AttackRollResult
	// DamageTypes are the damage types the spell deals and DamageDice the number of damage dice rolled.
	DamageTypes []content.DamageType
	DamageDice  int
	// RequiresConcentration says the spell requires Concentration.
	RequiresConcentration bool
	// HigherSlotAdditionalTarget says casting with a higher-level slot targets an additional creature.
	HigherSlotAdditionalTarget bool
	// Unknown lists the facts that are not known.
	Unknown []MetamagicFact
}

// MetamagicCastContext is what a Metamagic request needs to know about the casting so far: the host keeps it because the
// domain holds no per-casting state (class-sorcerer.md FR-110, FR-113). Construct it with a keyed literal.
type MetamagicCastContext struct {
	// Applied are the options already applied to this casting, in order.
	Applied []MetamagicEffect
	// LevelOnePlusSpellCastThisTurn is true after any level 1+ spell cast this turn, slot-free casts included.
	LevelOnePlusSpellCastThisTurn bool
}

// MetamagicParameters are the choices the host makes for an option. Construct it with a keyed literal.
type MetamagicParameters struct {
	// Count is the number of creatures protected (Careful Spell) or damage dice rerolled (Empowered Spell).
	Count int
	// From and To are the damage types of Transmuted Spell: the type the spell deals that changes, and the type it becomes.
	From, To content.DamageType
	// RequestFree asks for the free use of Arcane Apotheosis; it is granted at level 20 with Innate Sorcery active when none
	// was spent this turn, and otherwise the points are spent normally.
	RequestFree bool
}

// MetamagicRequest is one use of a Metamagic option: the effect, its cost from the class definition, the facts, the casting so
// far and the host's choices. Construct it with a keyed literal.
type MetamagicRequest struct {
	Effect     MetamagicEffect
	Cost       int
	Facts      MetamagicSpellFacts
	Context    MetamagicCastContext
	Parameters MetamagicParameters
}

// MetamagicApplication says what an option did to the casting, for the host to apply. Only the fields of the Effect are set;
// read it, do not construct it. A range of miles is reported in RangeMiles, a distance in feet in RangeFeet.
type MetamagicApplication struct {
	Effect MetamagicEffect
	// Cost is the Sorcery Points spent, 0 for a free use, and Free says it was the free use of Arcane Apotheosis.
	Cost int
	Free bool
	// ProtectedCreatures is the number of creatures that succeed automatically (Careful Spell).
	ProtectedCreatures int
	// RangeFeet and RangeMiles are the new range (Distant Spell), one of them.
	RangeFeet, RangeMiles int
	// Rerolls is the number of damage dice to reroll, the new rolls to be used (Empowered Spell).
	Rerolls int
	// DurationSeconds is the new duration (Extended Spell) and ConcentrationAdvantage says the saves to maintain Concentration have Advantage.
	DurationSeconds        int
	ConcentrationAdvantage bool
	// DisadvantageOnSave says one target has Disadvantage on its saves (Heightened Spell).
	DisadvantageOnSave bool
	// BonusAction says the casting time becomes a Bonus Action (Quickened Spell).
	BonusAction bool
	// RerollAttack says to reroll the d20 and use the new roll (Seeking Spell).
	RerollAttack bool
	// RemovesVerbal, RemovesSomatic and RemovesMaterial say which components are removed (Subtle Spell).
	RemovesVerbal, RemovesSomatic, RemovesMaterial bool
	// From and To are the damage types changed (Transmuted Spell).
	From, To content.DamageType
	// EffectiveLevelIncrease is 1 for Twinned Spell.
	EffectiveLevelIncrease int
}

// MetamagicTransition is the closed outcome of ApplyMetamagic. Variants: MetamagicApplied, MetamagicRefused.
type MetamagicTransition interface{ isMetamagicTransition() }

// MetamagicApplied is the outcome of a use the rules allowed: the new state, what the option did and the events.
type MetamagicApplied struct {
	state       SorcererState
	application MetamagicApplication
	events      []Event
}

// State returns the new SorcererState.
func (r MetamagicApplied) State() SorcererState { return r.state }

// Application returns what the option did.
func (r MetamagicApplied) Application() MetamagicApplication { return r.application }

// Events returns the events, as a new slice.
func (r MetamagicApplied) Events() []Event { return slices.Clone(r.events) }

// MetamagicRefused is the outcome of a use a rule refused: the state is unchanged.
type MetamagicRefused struct{ rejection SorcererRejection }

// Rejection returns why the use was refused.
func (r MetamagicRefused) Rejection() SorcererRejection { return r.rejection }

func (MetamagicApplied) isMetamagicTransition() {}
func (MetamagicRefused) isMetamagicTransition() {}

func notEligible(why MetamagicIneligibility) SorcererRejection {
	return SorcererRejection{Kind: SorcererMetamagicNotEligible, Ineligibility: why}
}

func unknownFact(f MetamagicFact) SorcererRejection {
	return SorcererRejection{Kind: SorcererMetamagicNotEligible, Ineligibility: MetamagicSpellFactUnknown, Fact: f}
}

// transmutable are the damage types Transmuted Spell works on (SRD 5.2.1, "Transmuted Spell", p. 67).
func transmutable() []content.DamageType {
	return []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire, content.DamageLightning, content.DamagePoison, content.DamageThunder}
}

func secondsOf(d content.SpellDurationTimed) int {
	switch d.Unit() {
	case content.DurationUnitRounds:
		return 6 * d.Amount()
	case content.DurationUnitMinutes:
		return 60 * d.Amount()
	case content.DurationUnitHours:
		return 3600 * d.Amount()
	case content.DurationUnitDays:
		return 86400 * d.Amount()
	}
	return 0
}

const (
	minute     = 60
	maxExtend  = 24 * 3600
	minRangeFt = 5
	touchRange = 30
)

// check decides whether the spell suits the option and, if so, fills in the application. cha is the Charisma modifier.
func (r MetamagicRequest) check(cha ability.Modifier) (MetamagicApplication, *SorcererRejection) {
	f, app := r.Facts, MetamagicApplication{Effect: r.Effect}
	need := func(fact MetamagicFact) *SorcererRejection {
		if slices.Contains(f.Unknown, fact) {
			rej := unknownFact(fact)
			return &rej
		}
		return nil
	}
	no := func(why MetamagicIneligibility) (MetamagicApplication, *SorcererRejection) {
		rej := notEligible(why)
		return MetamagicApplication{}, &rej
	}
	limit := max(cha.Value(), 1)
	var rej *SorcererRejection
	switch r.Effect {
	case MetamagicCareful:
		if rej = need(MetamagicFactSavingThrow); rej != nil {
			break
		}
		switch n := r.Parameters.Count; {
		case !f.ForcesSavingThrow:
			return no(MetamagicNoSavingThrow)
		case n < 1 || n > limit || n > f.OtherCreaturesSaving:
			return no(MetamagicCountOutOfRange)
		default:
			app.ProtectedCreatures = n
		}
	case MetamagicDistant:
		if rej = need(MetamagicFactRange); rej != nil {
			break
		}
		switch v := f.Range.(type) {
		case content.SpellRangeTouch:
			app.RangeFeet = touchRange
		case content.SpellRangeDistance:
			switch {
			case v.Unit() == content.DistanceMiles:
				app.RangeMiles = 2 * v.Amount()
			case v.Amount() >= minRangeFt:
				app.RangeFeet = 2 * v.Amount()
			default:
				return no(MetamagicRangeNotEligible)
			}
		default:
			return no(MetamagicRangeNotEligible)
		}
	case MetamagicEmpowered:
		if rej = need(MetamagicFactDamage); rej != nil {
			break
		}
		if n := r.Parameters.Count; n < 1 || n > limit || n > f.DamageDice {
			return no(MetamagicCountOutOfRange)
		}
		app.Rerolls = r.Parameters.Count
	case MetamagicExtended:
		if rej = need(MetamagicFactDuration); rej != nil {
			break
		}
		timed, ok := f.Duration.(content.SpellDurationTimed)
		switch secs := secondsOf(timed); {
		case !ok || secs < minute:
			return no(MetamagicDurationNotEligible)
		case secs >= maxExtend:
			return no(MetamagicDurationAlreadyAtMaximum)
		default:
			app.DurationSeconds = min(2*secs, maxExtend)
			app.ConcentrationAdvantage = f.RequiresConcentration
		}
	case MetamagicHeightened:
		if rej = need(MetamagicFactSavingThrow); rej != nil {
			break
		}
		if !f.ForcesSavingThrow {
			return no(MetamagicNoSavingThrow)
		}
		app.DisadvantageOnSave = true
	case MetamagicQuickened:
		if rej = need(MetamagicFactCastingTime); rej != nil {
			break
		}
		if _, ok := f.CastingTime.(content.CastingTimeMagicAction); !ok {
			return no(MetamagicCastingTimeNotAction)
		}
		if r.Context.LevelOnePlusSpellCastThisTurn {
			rej := refuse(SorcererQuickenedRestrictionApplies)
			return MetamagicApplication{}, &rej
		}
		app.BonusAction = true
	case MetamagicSeeking:
		if rej = need(MetamagicFactAttackRoll); rej != nil {
			break
		}
		if f.AttackRoll == nil || f.AttackRoll.Hit() {
			return no(MetamagicNoMissedAttackRoll)
		}
		app.RerollAttack = true
	case MetamagicSubtle:
		if rej = need(MetamagicFactComponents); rej != nil {
			break
		}
		app.RemovesVerbal, app.RemovesSomatic = f.Components.Verbal(), f.Components.Somatic()
		if m, has := f.Components.Material(); has && !m.ConsumesMaterial() && len(m.CostsGP()) == 0 {
			app.RemovesMaterial = true
		}
		if !app.RemovesVerbal && !app.RemovesSomatic && !app.RemovesMaterial {
			return no(MetamagicNothingToRemove)
		}
	case MetamagicTransmuted:
		if rej = need(MetamagicFactDamage); rej != nil {
			break
		}
		from, to := r.Parameters.From, r.Parameters.To
		if !slices.Contains(f.DamageTypes, from) || !slices.Contains(transmutable(), from) || !slices.Contains(transmutable(), to) || from == to {
			return no(MetamagicDamageTypeNotTransmutable)
		}
		app.From, app.To = from, to
	case MetamagicTwinned:
		if rej = need(MetamagicFactHigherSlotTarget); rej != nil {
			break
		}
		if !f.HigherSlotAdditionalTarget {
			return no(MetamagicNoHigherSlotTarget)
		}
		app.EffectiveLevelIncrease = 1
	}
	if rej != nil {
		return MetamagicApplication{}, rej
	}
	return app, nil
}

// ApplyMetamagic uses a Metamagic option on a casting (class-sorcerer.md FR-045, FR-046, FR-059 to FR-082, FR-109 to FR-118,
// FR-120, FR-123): it spends the cost from the Sorcery Points and reports what the option did. cha is the Sorcerer's Charisma
// modifier. The checks run in this order and the first that fails refuses: SorcererFeatureNotGained below level 2,
// SorcererMetamagicOptionNotKnown, SorcererMetamagicCombinationRefused, the eligibility of the spell (SorcererMetamagicNotEligible
// with its Ineligibility, or SorcererQuickenedRestrictionApplies), and SorcererSorceryPointsInsufficient. A refusal leaves the
// state unchanged. The free use of Arcane Apotheosis is taken when requested and available, and otherwise the points are spent.
// It never panics and rolls no dice: the rerolls arrive as host values.
//
// A casting takes one option, or two with Innate Sorcery active from level 7; Empowered Spell and Seeking Spell may be added on
// top of those, after them, and the same option may not be applied twice.
func ApplyMetamagic(id Identifier, s SorcererState, p SorcererProfile, cha ability.Modifier, r MetamagicRequest) MetamagicTransition {
	applied := r.Context.Applied
	capacity := 1
	if s.active && p.level.n >= sorcererIncarnateLevel {
		capacity = 2
	}
	switch {
	case p.level.n < sorcererFontOfMagicLevel:
		return MetamagicRefused{rejection: notGained("metamagic")}
	case !s.knows(r.Effect):
		return MetamagicRefused{rejection: refuse(SorcererMetamagicOptionNotKnown)}
	case slices.Contains(applied, r.Effect), !r.Effect.exempt() && len(applied) >= capacity:
		return MetamagicRefused{rejection: refuse(SorcererMetamagicCombinationRefused)}
	}
	app, rej := r.check(cha)
	if rej != nil {
		return MetamagicRefused{rejection: *rej}
	}
	app.Cost = r.Cost
	if r.Parameters.RequestFree && p.level.n >= sorcererApotheosisLevel && s.active && !s.freeSpent {
		app.Cost, app.Free = 0, true
		s.freeSpent = true
	}
	if s.points < app.Cost {
		return MetamagicRefused{rejection: SorcererRejection{Kind: SorcererSorceryPointsInsufficient, Required: app.Cost, Available: s.points}}
	}
	s.points -= app.Cost
	if r.Effect == MetamagicQuickened {
		s.quickened = true
	}
	return MetamagicApplied{state: s, application: app, events: []Event{MetamagicUsed{id: id, effect: r.Effect, cost: app.Cost, free: app.Free}}}
}

// QuickenedRestriction is the closed outcome of CheckQuickenedRestriction. Variants: QuickenedCastAllowed, QuickenedCastRefused.
type QuickenedRestriction interface{ isQuickenedRestriction() }

// QuickenedCastAllowed: the casting may go ahead.
type QuickenedCastAllowed struct{}

// QuickenedCastRefused: a level 1+ spell cannot be cast on this turn after Quickened Spell.
type QuickenedCastRefused struct{ rejection SorcererRejection }

// Rejection returns the SorcererQuickenedRestrictionApplies rejection.
func (r QuickenedCastRefused) Rejection() SorcererRejection { return r.rejection }

func (QuickenedCastAllowed) isQuickenedRestriction() {}
func (QuickenedCastRefused) isQuickenedRestriction() {}

// CheckQuickenedRestriction decides whether a spell of spellLevel may be cast now: a level 1+ spell may not be cast on a turn
// after Quickened Spell was used, and a cantrip may (class-sorcerer.md FR-075; SRD 5.2.1, "Quickened Spell", p. 67).
func CheckQuickenedRestriction(s SorcererState, spellLevel int) QuickenedRestriction {
	if s.quickened && spellLevel >= 1 {
		return QuickenedCastRefused{rejection: refuse(SorcererQuickenedRestrictionApplies)}
	}
	return QuickenedCastAllowed{}
}
