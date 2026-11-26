package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// SpellEffectsRejectionKind is one reason ResolveSpellEffects refused a command
// (spell-effects.md FR-057). The zero value is invalid.
type SpellEffectsRejectionKind int

// The reasons, in the fixed order a rejection lists them.
const (
	// WrongTargetCount: the number of targets does not match the spell after scaling.
	WrongTargetCount SpellEffectsRejectionKind = iota + 1
	// NoClearPath: a chosen target is behind Total Cover.
	NoClearPath
	// InstanceAssignmentMismatch: the instance assignment does not fit the part.
	InstanceAssignmentMismatch
	// MissingSaveAttempt: a valid target has no saving throw attempt for the ability.
	MissingSaveAttempt
	// MissingAttackFacts: a valid target of an attack part has no attack facts.
	MissingAttackFacts
	// DamageTypeNotChosen: the spell lets the caster choose a Damage Type and none was given.
	DamageTypeNotChosen
	// DamageTypeNotAllowed: the chosen Damage Type is not one the spell offers.
	DamageTypeNotAllowed
	// CasterNotAllowed: the caster is listed as a target of a spell that excludes the caster.
	CasterNotAllowed
	// WrongTargetSubject: a creature was listed for an object spell or an object for a creature spell.
	WrongTargetSubject
	// SpellAttackNotPossible: a spell attack cannot be made against a target (Total Cover, range or reach).
	SpellAttackNotPossible
)

func spellEffectsRejectionNames() []string {
	return []string{"wrong_target_count", "no_clear_path", "instance_assignment_mismatch",
		"missing_save_attempt", "missing_attack_facts", "damage_type_not_chosen", "damage_type_not_allowed",
		"caster_not_allowed", "wrong_target_subject", "attack_not_possible"}
}

// Valid reports whether k is one of the ten reasons.
func (k SpellEffectsRejectionKind) Valid() bool {
	return k >= WrongTargetCount && int(k) <= len(spellEffectsRejectionNames())
}

// String returns the snake_case name of the reason, or "unknown".
func (k SpellEffectsRejectionKind) String() string {
	return enumName(spellEffectsRejectionNames(), int(k))
}

// refusedOnly reports whether k is a reason of legal input the spell's rules do not allow
// (class refused), not host input the engine cannot use (class invalid).
func (k SpellEffectsRejectionKind) refusedOnly() bool {
	return k == NoClearPath || k == CasterNotAllowed || k == WrongTargetSubject || k == SpellAttackNotPossible
}

// InvalidTargetPerception is what the caster perceives when a spell targets
// something it cannot affect (SRD 5.2.1, "Invalid Targets", p. 106). The zero value
// is invalid.
type InvalidTargetPerception int

// The two perceptions.
const (
	// AppearedToSucceed: the target seems to have succeeded on the saving throw.
	AppearedToSucceed InvalidTargetPerception = iota + 1
	// SpellDidNothing: the spell visibly does nothing to the target.
	SpellDidNothing
)

// String returns "appeared_to_succeed", "spell_did_nothing" or "unknown".
func (p InvalidTargetPerception) String() string {
	return enumName([]string{"appeared_to_succeed", "spell_did_nothing"}, int(p))
}

// PlannedDamage is the damage of one damage event of a spell: one part, one
// instance, one target (spell-effects.md FR-065). The host passes one to each
// ApplyDamage command. Amounts are the full rolled amounts; a halved success
// carries the HalfDamage adjustment instead of a halved amount. Immutable and safe
// for concurrent use.
type PlannedDamage struct {
	part, instance int
	damages        []Damage
	half           bool
	kind           HitKind
}

// PartIndex returns the index of the part of the spell the damage came from.
func (d PlannedDamage) PartIndex() int { return d.part }

// InstanceIndex returns the index of the instance of the part.
func (d PlannedDamage) InstanceIndex() int { return d.instance }

// Damages returns the damage of each damaging outcome, in outcome order, as a new slice.
func (d PlannedDamage) Damages() []Damage { return slices.Clone(d.damages) }

// Adjustments returns the adjustments the host passes to ApplyDamage: HalfDamage
// for a halved success, else none, as a new slice.
func (d PlannedDamage) Adjustments() []DamageAdjustment {
	if d.half {
		return []DamageAdjustment{HalfDamage{}}
	}
	return nil
}

// HitKind returns CriticalHit for an attack that scored a Critical Hit, else NormalHit.
func (d PlannedDamage) HitKind() HitKind { return d.kind }

// Equal reports whether d and o are the same plan.
func (d PlannedDamage) Equal(o PlannedDamage) bool {
	return d.part == o.part && d.instance == o.instance && d.half == o.half && d.kind == o.kind && slices.Equal(d.damages, o.damages)
}

// PlannedAmount is an amount of Hit Points a spell restores or grants as Temporary
// Hit Points, with the part and instance it came from. Comparable.
type PlannedAmount struct{ part, instance, amount int }

// PartIndex returns the index of the part of the spell the amount came from.
func (a PlannedAmount) PartIndex() int { return a.part }

// InstanceIndex returns the index of the instance of the part.
func (a PlannedAmount) InstanceIndex() int { return a.instance }

// Amount returns the amount, at least 1.
func (a PlannedAmount) Amount() int { return a.amount }

// ConditionToApply is a condition a spell puts on a target: the host passes
// ToCondition to ApplyConditionToSet. The duration is data for the host; no
// duration is tracked here (spell-effects.md FR-038, FR-040). Immutable and safe
// for concurrent use.
type ConditionToApply struct {
	target   SpellTargetRef
	kind     content.ConditionKind
	source   CombatantID
	duration content.ConditionDuration
	part     int
}

// Target returns the target of the condition.
func (c ConditionToApply) Target() SpellTargetRef { return c.target }

// Kind returns the kind of the condition.
func (c ConditionToApply) Kind() content.ConditionKind { return c.kind }

// Source returns the caster as the source of the condition.
func (c ConditionToApply) Source() SpellTargetRef { return CreatureTargetRef{c.source} }

// Duration returns how long the condition lasts, as the model states it.
func (c ConditionToApply) Duration() content.ConditionDuration { return c.duration }

// PartIndex returns the index of the part of the spell the condition came from.
func (c ConditionToApply) PartIndex() int { return c.part }

// ToCondition returns the Condition for ApplyConditionToSet: Charmed, Frightened
// and Grappled with the caster as their source, Exhaustion at level 1 (applying
// Exhaustion adds one level), any other kind as itself.
func (c ConditionToApply) ToCondition() Condition {
	switch c.kind {
	case content.ConditionBlinded:
		return Blinded{}
	case content.ConditionCharmed:
		return Charmed{CharmedByCombatant{c.source}}
	case content.ConditionDeafened:
		return Deafened{}
	case content.ConditionExhaustion:
		return ExhaustionCondition{MustExhaustionLevel(1)}
	case content.ConditionFrightened:
		return Frightened{c.source}
	case content.ConditionGrappled:
		return Grappled{c.source}
	case content.ConditionIncapacitated:
		return Incapacitated{}
	case content.ConditionInvisible:
		return Invisible{}
	case content.ConditionParalyzed:
		return Paralyzed{}
	case content.ConditionPetrified:
		return Petrified{}
	case content.ConditionPoisoned:
		return Poisoned{}
	case content.ConditionProne:
		return Prone{}
	case content.ConditionRestrained:
		return Restrained{}
	case content.ConditionStunned:
		return Stunned{}
	case content.ConditionUnconscious:
		return Unconscious{}
	default:
		panic(unreachableError("condition kind", c.kind))
	}
}

// TargetEffects is what a spell does to one target: the plans for ApplyDamage,
// RestoreHitPoints, GainTemporaryHitPoints and ApplyConditionToSet, or, for a
// target the spell cannot affect, what the caster perceives (spell-effects.md
// FR-064). Immutable and safe for concurrent use.
type TargetEffects struct {
	target     SpellTargetRef
	damage     []PlannedDamage
	healing    []PlannedAmount
	temporary  []PlannedAmount
	conditions []ConditionToApply
	perception InvalidTargetPerception
}

// Target returns the target.
func (e TargetEffects) Target() SpellTargetRef { return e.target }

// Damage returns the planned damage in part and instance order, as a new slice.
func (e TargetEffects) Damage() []PlannedDamage { return slices.Clone(e.damage) }

// Healing returns the Hit Points to restore, as a new slice.
func (e TargetEffects) Healing() []PlannedAmount { return slices.Clone(e.healing) }

// TemporaryHitPoints returns the Temporary Hit Points to grant, as a new slice.
func (e TargetEffects) TemporaryHitPoints() []PlannedAmount { return slices.Clone(e.temporary) }

// Conditions returns the conditions to apply, as a new slice.
func (e TargetEffects) Conditions() []ConditionToApply { return slices.Clone(e.conditions) }

// Perception returns what the caster perceives and true when the target is one
// the spell cannot affect.
func (e TargetEffects) Perception() (InvalidTargetPerception, bool) {
	return e.perception, e.perception != 0
}

// Equal reports whether e and o are the same effects.
func (e TargetEffects) Equal(o TargetEffects) bool {
	return e.target == o.target && e.perception == o.perception && slices.EqualFunc(e.damage, o.damage, PlannedDamage.Equal) &&
		slices.Equal(e.healing, o.healing) && slices.Equal(e.temporary, o.temporary) && slices.Equal(e.conditions, o.conditions)
}

// RolledAmount is the evidence of one amount: the faces of the dice rolled for the
// outcome at Outcome and the amount that resulted after the modifier and the floor
// at 0. Immutable.
type RolledAmount struct {
	outcome int
	faces   []int
	amount  int
}

// Outcome returns the index of the outcome in its list.
func (a RolledAmount) Outcome() int { return a.outcome }

// Faces returns the face of each die rolled, as a new slice; none for a fixed amount.
func (a RolledAmount) Faces() []int { return slices.Clone(a.faces) }

// Amount returns the amount that resulted.
func (a RolledAmount) Amount() int { return a.amount }

func (a RolledAmount) equal(o RolledAmount) bool {
	return a.outcome == o.outcome && a.amount == o.amount && slices.Equal(a.faces, o.faces)
}

// ResolvedSave is the saving throw of one target: the d20 result unchanged.
type ResolvedSave struct {
	target SpellTargetRef
	save   UnarmedStrikeSave
}

// Target returns the target that saved.
func (s ResolvedSave) Target() SpellTargetRef { return s.target }

// Save returns the saving throw as d20 resolved it: rolled, or failed without a die.
func (s ResolvedSave) Save() UnarmedStrikeSave { return s.save }

func (s ResolvedSave) equal(o ResolvedSave) bool {
	switch a := s.save.(type) {
	case UnarmedStrikeSaveRolled:
		b, ok := o.save.(UnarmedStrikeSaveRolled)
		return ok && a.result.Equal(b.result)
	case UnarmedStrikeSaveForced:
		b, ok := o.save.(UnarmedStrikeSaveForced)
		return ok && a.result == b.result
	}
	return false
}

// ResolvedInstance is the evidence of one instance of a part (spell-effects.md
// FR-042): its targets, the attack roll or the saves, and the amounts rolled.
// Immutable and safe for concurrent use.
type ResolvedInstance struct {
	index    int
	targets  []SpellTargetRef
	attack   d20.AttackRollResult
	rolled   bool
	miss     AutomaticMissReason
	saves    []ResolvedSave
	amounts  []RolledAmount
	hit      bool
	critical bool
}

// Index returns the index of the instance in its part.
func (i ResolvedInstance) Index() int { return i.index }

// Targets returns the targets of the instance, as a new slice.
func (i ResolvedInstance) Targets() []SpellTargetRef { return slices.Clone(i.targets) }

// Attack returns the attack roll exactly as d20.ResolveAttack returned it, with the
// effective Armor Class as its target, and whether one was rolled.
func (i ResolvedInstance) Attack() (d20.AttackRollResult, bool) { return i.attack, i.rolled }

// AutomaticMiss returns why the attack missed without a roll and whether it did.
func (i ResolvedInstance) AutomaticMiss() (AutomaticMissReason, bool) { return i.miss, i.miss != 0 }

// Hit reports whether an attack instance hit, a Critical Hit included.
func (i ResolvedInstance) Hit() bool { return i.hit }

// CriticalHit reports whether an attack instance scored a Critical Hit.
func (i ResolvedInstance) CriticalHit() bool { return i.critical }

// Saves returns the saving throws of the instance, in target order, as a new slice.
func (i ResolvedInstance) Saves() []ResolvedSave { return slices.Clone(i.saves) }

// Amounts returns the amounts rolled for the instance, as a new slice.
func (i ResolvedInstance) Amounts() []RolledAmount {
	out := slices.Clone(i.amounts)
	for k := range out {
		out[k].faces = slices.Clone(out[k].faces)
	}
	return out
}

// Equal reports whether i and o are the same evidence.
func (i ResolvedInstance) Equal(o ResolvedInstance) bool {
	return i.index == o.index && i.rolled == o.rolled && i.miss == o.miss && i.hit == o.hit && i.critical == o.critical &&
		slices.Equal(i.targets, o.targets) && (!i.rolled || i.attack.Equal(o.attack)) &&
		slices.EqualFunc(i.saves, o.saves, ResolvedSave.equal) && slices.EqualFunc(i.amounts, o.amounts, RolledAmount.equal)
}

// ResolvedPart is the evidence of one resolved part of a spell.
// Immutable and safe for concurrent use.
type ResolvedPart struct {
	index     int
	save      ability.Ability
	instances []ResolvedInstance
}

// Index returns the index of the part in the model.
func (p ResolvedPart) Index() int { return p.index }

// SaveAbility returns the ability of the saving throw and whether the part is a saving throw part.
func (p ResolvedPart) SaveAbility() (ability.Ability, bool) { return p.save, p.save.Valid() }

// Instances returns the evidence of each instance, as a new slice.
func (p ResolvedPart) Instances() []ResolvedInstance { return slices.Clone(p.instances) }

// Equal reports whether p and o are the same evidence.
func (p ResolvedPart) Equal(o ResolvedPart) bool {
	return p.index == o.index && p.save == o.save && slices.EqualFunc(p.instances, o.instances, ResolvedInstance.Equal)
}
