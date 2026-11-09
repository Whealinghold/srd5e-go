package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// RangerFacts is what the pure Ranger rules read: the progression built from the content, the Ranger level, the RangerState and
// the subclass the Ranger holds ("" for none). Construct it with a keyed literal. A zero Progression or Level has no feature, so
// every rule answers "feature not available" rather than panicking.
type RangerFacts struct {
	Progression RangerProgression
	Level       RangerLevel
	State       RangerState
	Subclass    content.SubclassID
}

func (f RangerFacts) has(feat RangerFeature) bool { return f.Progression.has(feat, f.Level) }

func (f RangerFacts) hunter() bool { return f.Subclass == HunterSubclassID }

// rangerHasHunterFeature reports whether the Ranger chose the Hunter and has the Hunter feature.
func (f RangerFacts) hunterHas(feat RangerFeature) bool { return f.hunter() && f.has(feat) }

// AttacksPerAttackAction returns the attacks per Attack action the class gives: 1 below Extra Attack and 2 from it (SRD 5.2.1,
// "Level 5: Extra Attack", p. 59). It is a total, not a bonus, the per-class count ExtraAttackCombination takes the greatest of
// (class-ranger.md FR-065, FR-068).
func (f RangerFacts) AttacksPerAttackAction() int {
	if f.has(RangerFeatureExtraAttack) {
		return 2
	}
	return 1
}

// SpeedInput is a base Speed and whether it can increase: false when a condition such as Grappled sets Speed to 0, which "can't
// increase" (class-ranger.md FR-070). Build it with NewSpeedInput.
type SpeedInput struct {
	base        int
	canIncrease bool
}

// NewSpeedInput returns the input for base feet and canIncrease. It returns an error matching srderr.ErrInvalidInput for a
// negative base. It never panics.
func NewSpeedInput(base int, canIncrease bool) (SpeedInput, error) {
	if base < 0 {
		return SpeedInput{}, invalidInput("base Speed " + strconv.Itoa(base) + " is negative")
	}
	return SpeedInput{base: base, canIncrease: canIncrease}, nil
}

// MustSpeedInput is NewSpeedInput for literals known to be valid. It panics with the error NewSpeedInput returns, and only then.
func MustSpeedInput(base int, canIncrease bool) SpeedInput {
	in, err := NewSpeedInput(base, canIncrease)
	if err != nil {
		panic(err)
	}
	return in
}

// SpeedProfile is the Speeds Roving gives. Immutable, comparable with ==.
type SpeedProfile struct {
	walk, climbSwim int
	hasClimbSwim    bool
}

// Walk returns the walking Speed in feet.
func (p SpeedProfile) Walk() int { return p.walk }

// Climb returns the Climb Speed and false when the Ranger has none from Roving.
func (p SpeedProfile) Climb() (int, bool) { return p.climbSwim, p.hasClimbSwim }

// Swim returns the Swim Speed and false when the Ranger has none from Roving.
func (p SpeedProfile) Swim() (int, bool) { return p.climbSwim, p.hasClimbSwim }

// Speed returns the Speeds with Roving (SRD 5.2.1, "Level 6: Roving", p. 59): from the level of Roving the Speed increases by 10
// feet while no Heavy armor is worn and the Speed can increase, and a Climb Speed and a Swim Speed equal the Speed are had
// (class-ranger.md FR-066, FR-067, FR-069, FR-070). Below it the walking Speed is the base and there is no Climb or Swim Speed.
func (f RangerFacts) Speed(in SpeedInput, heavyArmor bool) SpeedProfile {
	if !f.has(RangerFeatureRoving) {
		return SpeedProfile{walk: in.base}
	}
	walk := in.base
	if in.canIncrease && !heavyArmor {
		walk += 10
	}
	return SpeedProfile{walk: walk, climbSwim: walk, hasClimbSwim: true}
}

// RangerSenseGrant is a special sense of a range in feet that a feature grants. Construct it with a keyed literal.
type RangerSenseGrant struct {
	Sense     content.SpecialSense
	RangeFeet int
}

// Senses returns the senses the Ranger has from its features: Blindsight 30 feet from Feral Senses (SRD 5.2.1, "Level 18: Feral
// Senses", p. 59; class-ranger.md FR-082), as a new slice.
func (f RangerFacts) Senses() []RangerSenseGrant {
	if f.has(RangerFeatureFeralSenses) {
		return []RangerSenseGrant{{Sense: content.SenseBlindsight, RangeFeet: 30}}
	}
	return []RangerSenseGrant{}
}

// HuntersMarkDie returns the damage die of Hunter’s Mark: a d10 from Foe Slayer, else the spell’s own die base (SRD 5.2.1, "Level
// 20: Foe Slayer", p. 59; class-ranger.md FR-083).
func (f RangerFacts) HuntersMarkDie(base dice.Die) dice.Die {
	if f.has(RangerFeatureFoeSlayer) {
		return dice.D10()
	}
	return base
}

// DamageCanBreakConcentration reports whether damage can break the Ranger's Concentration on spell: false for Hunter’s Mark from
// Relentless Hunter, true for any other spell and below it (SRD 5.2.1, "Level 13: Relentless Hunter", p. 59; class-ranger.md
// FR-078). Other reasons Concentration ends are unaffected. It takes only ids and the level; the host passes the answer to combat.
func (f RangerFacts) DamageCanBreakConcentration(spell content.SpellID) bool {
	return !f.has(RangerFeatureRelentlessHunter) || spell != f.Progression.HuntersMark()
}

// MarkStatus is whether a creature is marked by the Ranger's own Hunter’s Mark. The zero value is invalid.
type MarkStatus int

const (
	// MarkedByHuntersMark: the creature is currently marked by the Ranger's Hunter’s Mark.
	MarkedByHuntersMark MarkStatus = iota + 1
	// UnmarkedByHuntersMark: it is not.
	UnmarkedByHuntersMark
)

// PreciseHunter returns the RollCircumstance of an attack roll against a creature: Advantage from Precise Hunter when the Ranger
// has it and the creature is marked, else none (SRD 5.2.1, "Level 17: Precise Hunter", p. 59; class-ranger.md FR-081). It is a
// new slice.
func (f RangerFacts) PreciseHunter(target MarkStatus) []modifier.RollCircumstance {
	if f.has(RangerFeaturePreciseHunter) && target == MarkedByHuntersMark {
		return []modifier.RollCircumstance{modifier.Advantage{}}
	}
	return []modifier.RollCircumstance{}
}

// HuntersLoreResult is the closed outcome of RangerFacts.HuntersLore. Variants: HuntersLoreNoneApply, HuntersLoreReveals,
// HuntersLoreNotMarked, HuntersLoreNotHunter, HuntersLoreFeatureNotAvailable.
type HuntersLoreResult interface{ isHuntersLoreResult() }

// HuntersLoreNoneApply: the creature has no Immunity, Resistance or Vulnerability.
type HuntersLoreNoneApply struct{}

// HuntersLoreReveals reveals the creature's defenses, each entry unchanged so its qualifier is kept. Only HuntersLore builds one.
type HuntersLoreReveals struct{ immunities, resistances, vulnerabilities []content.DefenseEntry }

// Immunities returns the Immunities as a new slice.
func (r HuntersLoreReveals) Immunities() []content.DefenseEntry {
	return append([]content.DefenseEntry{}, r.immunities...)
}

// Resistances returns the Resistances as a new slice.
func (r HuntersLoreReveals) Resistances() []content.DefenseEntry {
	return append([]content.DefenseEntry{}, r.resistances...)
}

// Vulnerabilities returns the Vulnerabilities as a new slice.
func (r HuntersLoreReveals) Vulnerabilities() []content.DefenseEntry {
	return append([]content.DefenseEntry{}, r.vulnerabilities...)
}

// HuntersLoreNotMarked: the creature is not marked, so nothing is revealed.
type HuntersLoreNotMarked struct{}

// HuntersLoreNotHunter: the character has not chosen the Hunter.
type HuntersLoreNotHunter struct{}

// HuntersLoreFeatureNotAvailable: the Ranger level is below the level of Hunter’s Lore.
type HuntersLoreFeatureNotAvailable struct{}

func (HuntersLoreNoneApply) isHuntersLoreResult()           {}
func (HuntersLoreReveals) isHuntersLoreResult()             {}
func (HuntersLoreNotMarked) isHuntersLoreResult()           {}
func (HuntersLoreNotHunter) isHuntersLoreResult()           {}
func (HuntersLoreFeatureNotAvailable) isHuntersLoreResult() {}

// HuntersLore decides what Hunter’s Lore reveals about a creature with the three defense lists the caller supplies (SRD 5.2.1,
// "Level 3: Hunter’s Lore", p. 61; class-ranger.md FR-084, FR-085): nothing for a character that has not chosen the Hunter, below
// the level of the feature or for a creature not marked, never an empty reveal; HuntersLoreNoneApply when all three lists are
// empty. The lists are copied.
func (f RangerFacts) HuntersLore(target MarkStatus, immunities, resistances, vulnerabilities []content.DefenseEntry) HuntersLoreResult {
	switch {
	case !f.hunter():
		return HuntersLoreNotHunter{}
	case !f.has(RangerFeatureHuntersLore):
		return HuntersLoreFeatureNotAvailable{}
	case target != MarkedByHuntersMark:
		return HuntersLoreNotMarked{}
	case len(immunities)+len(resistances)+len(vulnerabilities) == 0:
		return HuntersLoreNoneApply{}
	}
	return HuntersLoreReveals{immunities: append([]content.DefenseEntry{}, immunities...),
		resistances: append([]content.DefenseEntry{}, resistances...), vulnerabilities: append([]content.DefenseEntry{}, vulnerabilities...)}
}

// RiderDamageType is the damage type of an AttackRider, a closed set. Variants: UnstatedDamageType, StatedDamageType.
type RiderDamageType interface{ isRiderDamageType() }

// UnstatedDamageType: the SRD states no damage type for the rider, so none is inferred.
type UnstatedDamageType struct{}

// StatedDamageType is a rider of a stated damage type. Construct it with a keyed literal.
type StatedDamageType struct{ Type content.DamageType }

func (UnstatedDamageType) isRiderDamageType() {}
func (StatedDamageType) isRiderDamageType()   {}

// AttackRider is extra damage a Ranger feature adds, as dice for the host to roll: Count dice of Die of Damage. Construct it with
// a keyed literal.
type AttackRider struct {
	Count  int
	Die    dice.Die
	Damage RiderDamageType
}

// ColossusSlayerFacts are the facts of one hit for Colossus Slayer. Construct it with a keyed literal.
type ColossusSlayerFacts struct {
	WeaponHit       bool // the hit was made with a weapon
	TargetHitPoints int
	TargetMaximum   int
	DealtThisTurn   bool // the extra damage was already dealt this turn
}

// optionHeld reports whether the Hunter has the feature at its level and holds the option.
func (f RangerFacts) preyHeld(option string) (RangerRiderReason, bool) {
	if !f.hunterHas(RangerFeatureHuntersPrey) {
		return RangerRiderFeatureNotAvailable, false
	}
	if held, _ := f.State.HuntersPrey(); held != option {
		return RangerRiderOptionNotHeld, false
	}
	return 0, true
}

// ColossusSlayer returns the rider of a hit that deals the extra 1d8 damage and true, or false when it does not apply: below
// Ranger level 3, without the Hunter, without the option, when the creature is at its Hit Point maximum, when the extra damage
// was already dealt this turn or when the hit was not made with a weapon (SRD 5.2.1, "Level 3: Hunter’s Prey", p. 61;
// class-ranger.md FR-088, FR-089). The rider carries UnstatedDamageType.
func (f RangerFacts) ColossusSlayer(h ColossusSlayerFacts) (AttackRider, bool) {
	if _, ok := f.preyHeld("colossus_slayer"); !ok || !h.WeaponHit || h.DealtThisTurn || h.TargetHitPoints >= h.TargetMaximum {
		return AttackRider{}, false
	}
	return AttackRider{Count: 1, Die: dice.D8(), Damage: UnstatedDamageType{}}, true
}

// RangerRiderEligibility is the closed outcome of the rider eligibility rules. Variants: RangerRiderEligible,
// RangerRiderIneligible.
type RangerRiderEligibility interface{ isRangerRiderEligibility() }

// RangerRiderEligible: the extra attack or rider is allowed. Die is the extra damage die of Superior Hunter’s Prey and the zero
// Die for Horde Breaker, which deals its weapon's damage.
type RangerRiderEligible struct{ Die dice.Die }

// RangerRiderIneligible: it is not, with the first reason that holds. Construct it with a keyed literal.
type RangerRiderIneligible struct{ Reason RangerRiderReason }

func (RangerRiderEligible) isRangerRiderEligibility()   {}
func (RangerRiderIneligible) isRangerRiderEligibility() {}

// HordeBreakerFacts are the facts of a possible extra attack for Horde Breaker. DistanceFeet is from the original target.
// Construct it with a keyed literal.
type HordeBreakerFacts struct {
	WeaponAttack      bool
	DifferentCreature bool
	DistanceFeet      int
	InWeaponRange     bool
	AlreadyAttacked   bool // the creature was attacked this turn
	UsedThisTurn      bool // Horde Breaker was used this turn
}

// HordeBreaker decides the extra attack of Horde Breaker: once on each of the Ranger's turns, with the same weapon, against a
// different creature within 5 feet of the original target, within the weapon's range and not yet attacked this turn (SRD 5.2.1,
// "Level 3: Hunter’s Prey", p. 61; class-ranger.md FR-090). The reasons are checked in the order of RangerRiderReason.
func (f RangerFacts) HordeBreaker(h HordeBreakerFacts) RangerRiderEligibility {
	if reason, ok := f.preyHeld("horde_breaker"); !ok {
		return RangerRiderIneligible{Reason: reason}
	}
	switch {
	case !h.WeaponAttack:
		return RangerRiderIneligible{Reason: RangerRiderNotWeaponAttack}
	case h.UsedThisTurn:
		return RangerRiderIneligible{Reason: RangerRiderUsedThisTurn}
	case !h.DifferentCreature:
		return RangerRiderIneligible{Reason: RangerRiderSameCreature}
	case h.DistanceFeet > 5:
		return RangerRiderIneligible{Reason: RangerRiderTooFar}
	case !h.InWeaponRange:
		return RangerRiderIneligible{Reason: RangerRiderOutOfRange}
	case h.AlreadyAttacked:
		return RangerRiderIneligible{Reason: RangerRiderAlreadyAttacked}
	}
	return RangerRiderEligible{}
}

// SuperiorHuntersPreyFacts are the facts of a possible second target for Superior Hunter’s Prey: the first target's mark, the
// spell's own die and the second target. Construct it with a keyed literal.
type SuperiorHuntersPreyFacts struct {
	FirstTarget       MarkStatus
	SpellDie          dice.Die
	DifferentCreature bool
	DistanceFeet      int // from the first creature
	Visible           bool
	UsedThisTurn      bool
}

// SuperiorHuntersPrey decides the second target of Superior Hunter’s Prey: once per turn, when the Ranger deals damage to a
// creature marked by its Hunter’s Mark, the spell's extra damage to a different creature it can see within 30 feet of the first
// (SRD 5.2.1, "Level 11: Superior Hunter’s Prey", p. 61; class-ranger.md FR-094). The eligible result carries the die of
// HuntersMarkDie. The reasons are checked in the order of RangerRiderReason.
func (f RangerFacts) SuperiorHuntersPrey(h SuperiorHuntersPreyFacts) RangerRiderEligibility {
	switch {
	case !f.hunterHas(RangerFeatureSuperiorHuntersPrey):
		return RangerRiderIneligible{Reason: RangerRiderFeatureNotAvailable}
	case h.FirstTarget != MarkedByHuntersMark:
		return RangerRiderIneligible{Reason: RangerRiderNotMarked}
	case h.UsedThisTurn:
		return RangerRiderIneligible{Reason: RangerRiderUsedThisTurn}
	case !h.DifferentCreature:
		return RangerRiderIneligible{Reason: RangerRiderSameCreature}
	case h.DistanceFeet > 30:
		return RangerRiderIneligible{Reason: RangerRiderTooFar}
	case !h.Visible:
		return RangerRiderIneligible{Reason: RangerRiderNotVisible}
	}
	return RangerRiderEligible{Die: f.HuntersMarkDie(h.SpellDie)}
}

// defensiveTacticsHeld reports whether the Hunter has Defensive Tactics at its level and holds the option.
func (f RangerFacts) defensiveTacticsHeld(option string) bool {
	held, _ := f.State.DefensiveTactics()
	return f.hunterHas(RangerFeatureDefensiveTactics) && held == option
}

// EscapeTheHorde returns the RollCircumstance of an attack roll against the Ranger: Disadvantage for an Opportunity Attack when
// Escape the Horde is held, else none (SRD 5.2.1, "Level 7: Defensive Tactics", p. 61; class-ranger.md FR-092). It is a new
// slice.
func (f RangerFacts) EscapeTheHorde(opportunityAttack bool) []modifier.RollCircumstance {
	if opportunityAttack && f.defensiveTacticsHeld("escape_the_horde") {
		return []modifier.RollCircumstance{modifier.Disadvantage{}}
	}
	return []modifier.RollCircumstance{}
}

// AttackerKey identifies an attacker for Multiattack Defense. It is opaque: the host maps its own creature id to it.
type AttackerKey string

// MultiattackDefenseRecord is the attackers that hit the Ranger this turn. The zero value is the empty record. Immutable; not
// comparable with ==.
type MultiattackDefenseRecord struct{ hit []AttackerKey }

// HitBy returns a record that also holds attacker.
func (r MultiattackDefenseRecord) HitBy(attacker AttackerKey) MultiattackDefenseRecord {
	for _, a := range r.hit {
		if a == attacker {
			return r
		}
	}
	return MultiattackDefenseRecord{hit: append(append([]AttackerKey{}, r.hit...), attacker)}
}

// Holds reports whether attacker is in the record.
func (r MultiattackDefenseRecord) Holds(attacker AttackerKey) bool {
	for _, a := range r.hit {
		if a == attacker {
			return true
		}
	}
	return false
}

// EndOfTurn returns the empty record: the record is empty at the start of the next turn.
func (MultiattackDefenseRecord) EndOfTurn() MultiattackDefenseRecord {
	return MultiattackDefenseRecord{}
}

// RecordHit returns the record after attacker hit the Ranger with an attack roll: with attacker added when Multiattack Defense is
// held, and unchanged otherwise (class-ranger.md FR-093). Call it after resolving the attack roll that hit, which gets no
// Disadvantage from it.
func (f RangerFacts) RecordHit(r MultiattackDefenseRecord, attacker AttackerKey) MultiattackDefenseRecord {
	if f.defensiveTacticsHeld("multiattack_defense") {
		return r.HitBy(attacker)
	}
	return r
}

// MultiattackDefense returns the RollCircumstance of an attack roll by attacker against the Ranger: Disadvantage when
// Multiattack Defense is held and attacker already hit the Ranger this turn, else none (SRD 5.2.1, "Level 7: Defensive Tactics",
// p. 61; class-ranger.md FR-093). It is a new slice.
func (f RangerFacts) MultiattackDefense(r MultiattackDefenseRecord, attacker AttackerKey) []modifier.RollCircumstance {
	if f.defensiveTacticsHeld("multiattack_defense") && r.Holds(attacker) {
		return []modifier.RollCircumstance{modifier.Disadvantage{}}
	}
	return []modifier.RollCircumstance{}
}

// RangerResistanceGrant is a Resistance to a damage type until the grant expires. Construct it with a keyed literal.
type RangerResistanceGrant struct {
	Damage content.DamageType
	Expiry RangerGrantExpiry
}

// RangerConditionGrant is a condition the Ranger has until the grant expires. Construct it with a keyed literal.
type RangerConditionGrant struct {
	Condition content.ConditionKind
	Expiry    RangerGrantExpiry
}

// RangerGrantExpiry is when a Ranger grant ends, a closed set. The zero value is invalid.
type RangerGrantExpiry int

const (
	// GrantEndsAtEndOfCurrentTurn: the grant ends at the end of the current turn.
	GrantEndsAtEndOfCurrentTurn RangerGrantExpiry = iota + 1
	// GrantEndsAtEndOfNextTurn: the grant ends at the end of the Ranger's next turn.
	GrantEndsAtEndOfNextTurn
)

// FavoredEnemyUseResult is the closed outcome of UseFavoredEnemy. Variants: FavoredEnemyUseAccepted, FavoredEnemyUseRejected.
type FavoredEnemyUseResult interface{ isFavoredEnemyUseResult() }

// FavoredEnemyUseAccepted is a free cast of Hunter’s Mark spent: the new state and the FavoredEnemyCastSpent event. The host
// casts with CastMode.featureGranted. Only UseFavoredEnemy builds one.
type FavoredEnemyUseAccepted struct {
	state  RangerState
	events []Event
}

// State returns the new state.
func (r FavoredEnemyUseAccepted) State() RangerState { return r.state }

// Events returns a new slice holding the FavoredEnemyCastSpent event.
func (r FavoredEnemyUseAccepted) Events() []Event { return append([]Event{}, r.events...) }

// FavoredEnemyUseRejected is a use the rules refused: the state stands, there are no events, and a cast with a spell slot is
// spellcasting's concern. Construct it with a keyed literal.
type FavoredEnemyUseRejected struct{ Reason RangerReason }

func (FavoredEnemyUseAccepted) isFavoredEnemyUseResult() {}
func (FavoredEnemyUseRejected) isFavoredEnemyUseResult() {}

// UseFavoredEnemy spends one free cast of Hunter’s Mark (SRD 5.2.1, "Level 1: Favored Enemy", p. 58; class-ranger.md FR-033 to
// FR-035). The last remaining use is accepted; with none remaining it is rejected with RangerNoUsesRemaining and the state stands.
// id is the Character the event concerns.
func UseFavoredEnemy(id Identifier, f RangerFacts) FavoredEnemyUseResult {
	if !f.has(RangerFeatureFavoredEnemy) {
		return FavoredEnemyUseRejected{Reason: RangerFeatureNotAvailable}
	}
	left := f.State.FavoredEnemyRemaining(f.Progression, f.Level)
	if left == 0 {
		return FavoredEnemyUseRejected{Reason: RangerNoUsesRemaining}
	}
	next := f.State
	next.favoredSpent++
	return FavoredEnemyUseAccepted{state: next, events: []Event{FavoredEnemyCastSpent{id: id, remaining: left - 1}}}
}

// NaturesVeilResult is the closed outcome of UseNaturesVeil. Variants: NaturesVeilAccepted, NaturesVeilRejected.
type NaturesVeilResult interface{ isNaturesVeilResult() }

// NaturesVeilAccepted is a use of Nature’s Veil: the new state, the Invisible grant, the action cost and the event. Only
// UseNaturesVeil builds one.
type NaturesVeilAccepted struct {
	state  RangerState
	events []Event
}

// State returns the new state.
func (r NaturesVeilAccepted) State() RangerState { return r.state }

// Invisible returns the Invisible grant, until the end of the Ranger's next turn.
func (NaturesVeilAccepted) Invisible() RangerConditionGrant {
	return RangerConditionGrant{Condition: content.ConditionInvisible, Expiry: GrantEndsAtEndOfNextTurn}
}

// Cost returns RangerBonusAction.
func (NaturesVeilAccepted) Cost() RangerActionCost { return RangerBonusAction }

// Events returns a new slice holding the NaturesVeilUsed event.
func (r NaturesVeilAccepted) Events() []Event { return append([]Event{}, r.events...) }

// NaturesVeilRejected is a use the rules refused. Construct it with a keyed literal.
type NaturesVeilRejected struct{ Reason RangerReason }

func (NaturesVeilAccepted) isNaturesVeilResult() {}
func (NaturesVeilRejected) isNaturesVeilResult() {}

// UseNaturesVeil uses Nature’s Veil (SRD 5.2.1, "Level 14: Nature’s Veil", p. 59; class-ranger.md FR-079, FR-080): rejected with
// RangerFeatureNotAvailable below its level and RangerNoUsesRemaining with none left (the uses are the Wisdom modifier, minimum
// one).
func UseNaturesVeil(id Identifier, f RangerFacts, wisdom ability.Modifier) NaturesVeilResult {
	if !f.has(RangerFeatureNaturesVeil) {
		return NaturesVeilRejected{Reason: RangerFeatureNotAvailable}
	}
	left := f.State.NaturesVeilRemaining(wisdom)
	if left == 0 {
		return NaturesVeilRejected{Reason: RangerNoUsesRemaining}
	}
	next := f.State
	next.veilSpent++
	return NaturesVeilAccepted{state: next, events: []Event{NaturesVeilUsed{id: id, remaining: left - 1}}}
}

// SuperiorHuntersDefenseResult is the closed outcome of UseSuperiorHuntersDefense. Variants: SuperiorHuntersDefenseAccepted,
// SuperiorHuntersDefenseRejected.
type SuperiorHuntersDefenseResult interface{ isSuperiorHuntersDefenseResult() }

// SuperiorHuntersDefenseAccepted is a use of the Reaction: Resistance to the damage type until the end of the current turn.
// Construct it with a keyed literal.
type SuperiorHuntersDefenseAccepted struct{ Grant RangerResistanceGrant }

// Cost returns RangerReaction.
func (SuperiorHuntersDefenseAccepted) Cost() RangerActionCost { return RangerReaction }

// SuperiorHuntersDefenseRejected is a use the rules refused. Construct it with a keyed literal.
type SuperiorHuntersDefenseRejected struct{ Reason RangerReason }

func (SuperiorHuntersDefenseAccepted) isSuperiorHuntersDefenseResult() {}
func (SuperiorHuntersDefenseRejected) isSuperiorHuntersDefenseResult() {}

// UseSuperiorHuntersDefense decides the Reaction of Superior Hunter’s Defense when the Ranger takes damage of the type (SRD 5.2.1,
// "Level 15: Superior Hunter’s Defense", p. 61; class-ranger.md FR-095): rejected with RangerNotHunter without the Hunter,
// RangerFeatureNotAvailable below its level and RangerNoReaction without a Reaction, in that order. The Resistance applies to the
// triggering damage and any other damage of the same type, and expires at the end of the current turn.
func UseSuperiorHuntersDefense(f RangerFacts, damage content.DamageType, reactionAvailable bool) SuperiorHuntersDefenseResult {
	switch {
	case !f.hunter():
		return SuperiorHuntersDefenseRejected{Reason: RangerNotHunter}
	case !f.has(RangerFeatureSuperiorHuntersDefense):
		return SuperiorHuntersDefenseRejected{Reason: RangerFeatureNotAvailable}
	case !reactionAvailable:
		return SuperiorHuntersDefenseRejected{Reason: RangerNoReaction}
	}
	return SuperiorHuntersDefenseAccepted{Grant: RangerResistanceGrant{Damage: damage, Expiry: GrantEndsAtEndOfCurrentTurn}}
}

// TirelessTemporaryHitPoints returns the Temporary Hit Points of Tireless: the d8 roll plus the greater of 1 and the Wisdom
// modifier, the minimum applying to the modifier and not the total (SRD 5.2.1, "Level 10: Tireless", p. 59; class-ranger.md
// FR-072, FR-077). It returns an error matching srderr.ErrInvalidInput for a roll outside 1 to 8. It never panics.
func TirelessTemporaryHitPoints(roll int, wisdom ability.Modifier) (int, error) {
	if roll < 1 || roll > 8 {
		return 0, invalidInput("Tireless roll " + strconv.Itoa(roll) + " outside 1 to 8")
	}
	return roll + TirelessUses(wisdom), nil
}

// ApplyRangerRestApplied is the outcome of ApplyRangerRest: the state after the rest and the events it produced.
type ApplyRangerRestApplied struct {
	state  RangerState
	events []Event
}

// State returns the state after the rest.
func (r ApplyRangerRestApplied) State() RangerState { return r.state }

// Events returns a new slice holding the ExhaustionReductionGranted event, and none when the rest grants none.
func (r ApplyRangerRestApplied) Events() []Event { return append([]Event{}, r.events...) }

// ApplyRangerRest applies the benefits of a finished rest (RestApplied.BenefitsOf) to a Ranger's resources (SRD 5.2.1, "Level 1:
// Favored Enemy", "Level 10: Tireless", "Level 14: Nature’s Veil", p. 58-59; class-ranger.md FR-036, FR-075, FR-076, FR-080): a
// Long Rest resets the expended uses of Favored Enemy, Tireless and Nature’s Veil; a Short Rest leaves them and, from the level
// of Tireless with Exhaustion level 1 or more, returns an ExhaustionReductionGranted event of 1 (applying it is the rest rules').
// It returns an error matching srderr.ErrInvalidInput for a negative Exhaustion level. The inputs are not altered.
func ApplyRangerRest(id Identifier, f RangerFacts, kind RestKind, exhaustion int) (ApplyRangerRestApplied, error) {
	if exhaustion < 0 {
		return ApplyRangerRestApplied{}, invalidInput("Exhaustion level " + strconv.Itoa(exhaustion) + " is negative")
	}
	out := ApplyRangerRestApplied{state: f.State, events: []Event{}}
	switch kind {
	case LongRest:
		out.state.favoredSpent, out.state.tirelessSpent, out.state.veilSpent = 0, 0, 0
	case ShortRest:
		if f.has(RangerFeatureTireless) && exhaustion >= 1 {
			out.events = append(out.events, ExhaustionReductionGranted{id: id, amount: 1})
		}
	}
	return out, nil
}
