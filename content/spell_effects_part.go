package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// PartResolution is the closed set of ways a part of a spell is resolved (SRD
// 5.2.1, "Effects", p. 106): [AttackRollResolution], [SavingThrowResolution]
// and [AutomaticResolution], the last being package design for effects with
// no roll. The variants hold slices and are not comparable with ==; use
// [SpellEffectPart.Equal]. Consumers switch over it exhaustively.
type PartResolution interface{ isPartResolution() }

// checkOutcomes applies FR-026 to an outcome list and checks each outcome.
// A list that must not be empty gets BlockEmpty; a nil outcome gets
// BlockMissing.
func checkOutcomes(block string, outcomes []SpellOutcome, mayBeEmpty bool) error {
	if len(outcomes) == 0 && !mayBeEmpty {
		return blockError(block, BlockEmpty)
	}
	for _, o := range outcomes {
		if err := checkSpellOutcome(o); err != nil {
			return err
		}
	}
	return nil
}

// AttackRollResolution is a part resolved by a spell attack roll (SRD 5.2.1,
// "Attack Rolls", p. 106): the outcomes of a hit and of a miss. The zero value
// is invalid; build it with [NewAttackRollResolution]. Not comparable with ==;
// use [AttackRollResolution.Equal].
type AttackRollResolution struct {
	kind   SpellAttackKind
	onHit  []SpellOutcome
	onMiss []SpellOutcome
}

// NewAttackRollResolution returns the resolution by a spell attack of kind,
// copying the outcome lists. onMiss may be empty.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid kind, BlockEmpty for an empty onHit and
// the reason of an outcome's own check for an invalid outcome. It never panics.
func NewAttackRollResolution(kind SpellAttackKind, onHit, onMiss []SpellOutcome) (AttackRollResolution, error) {
	if !kind.Valid() {
		return AttackRollResolution{}, blockError("AttackRollResolution", BlockInvalidMember)
	}
	if err := checkOutcomes("AttackRollResolution", onHit, false); err != nil {
		return AttackRollResolution{}, err
	}
	if err := checkOutcomes("AttackRollResolution", onMiss, true); err != nil {
		return AttackRollResolution{}, err
	}
	return AttackRollResolution{kind: kind, onHit: slices.Clone(onHit), onMiss: slices.Clone(onMiss)}, nil
}

// Kind returns whether the attack is melee or ranged.
func (r AttackRollResolution) Kind() SpellAttackKind { return r.kind }

// OnHit returns a copy of the outcomes of a hit.
func (r AttackRollResolution) OnHit() []SpellOutcome { return slices.Clone(r.onHit) }

// OnMiss returns a copy of the outcomes of a miss.
func (r AttackRollResolution) OnMiss() []SpellOutcome { return slices.Clone(r.onMiss) }

// Equal reports whether r and o have the same kind and outcome lists.
func (r AttackRollResolution) Equal(o AttackRollResolution) bool {
	return r.kind == o.kind && slices.Equal(r.onHit, o.onHit) && slices.Equal(r.onMiss, o.onMiss)
}

func (AttackRollResolution) isPartResolution() {}

// SavingThrowResolution is a part resolved by a saving throw (SRD 5.2.1,
// "Saving Throws", p. 106): the ability, the outcomes of a failure and what a
// success does. The zero value is invalid; build it with
// [NewSavingThrowResolution]. Not comparable with ==; use
// [SavingThrowResolution.Equal].
type SavingThrowResolution struct {
	save      ability.Ability
	onFailure []SpellOutcome
	onSuccess SuccessEffect
}

// NewSavingThrowResolution returns the resolution by a saving throw of save,
// copying onFailure.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid ability or success effect, BlockEmpty for
// an empty onFailure, the reason of an outcome's own check for an invalid
// outcome, and BlockConstraint for [HalfDamageOnSuccess] with no
// [SpellDealsDamage] in onFailure (FR-043). It never panics.
func NewSavingThrowResolution(save ability.Ability, onFailure []SpellOutcome, onSuccess SuccessEffect) (SavingThrowResolution, error) {
	if !save.Valid() || !onSuccess.Valid() {
		return SavingThrowResolution{}, blockError("SavingThrowResolution", BlockInvalidMember)
	}
	if err := checkOutcomes("SavingThrowResolution", onFailure, false); err != nil {
		return SavingThrowResolution{}, err
	}
	if onSuccess == HalfDamageOnSuccess && !slices.ContainsFunc(onFailure, isDamage) {
		return SavingThrowResolution{}, blockError("SavingThrowResolution", BlockConstraint)
	}
	return SavingThrowResolution{save: save, onFailure: slices.Clone(onFailure), onSuccess: onSuccess}, nil
}

// isDamage reports whether o is a [SpellDealsDamage].
func isDamage(o SpellOutcome) bool {
	_, ok := o.(SpellDealsDamage)
	return ok
}

// Ability returns the ability of the saving throw.
func (r SavingThrowResolution) Ability() ability.Ability { return r.save }

// OnFailure returns a copy of the outcomes of a failed save.
func (r SavingThrowResolution) OnFailure() []SpellOutcome { return slices.Clone(r.onFailure) }

// OnSuccess returns what a successful save does.
func (r SavingThrowResolution) OnSuccess() SuccessEffect { return r.onSuccess }

// Equal reports whether r and o have the same ability, outcomes and success
// effect.
func (r SavingThrowResolution) Equal(o SavingThrowResolution) bool {
	return r.save == o.save && r.onSuccess == o.onSuccess && slices.Equal(r.onFailure, o.onFailure)
}

func (SavingThrowResolution) isPartResolution() {}

// AutomaticResolution is a part with no roll, such as the darts of Magic
// Missile (SRD 5.2.1, p. 146). The zero value is invalid; build it with
// [NewAutomaticResolution]. Not comparable with ==; use
// [AutomaticResolution.Equal].
type AutomaticResolution struct{ outcomes []SpellOutcome }

// NewAutomaticResolution returns the resolution with outcomes, copying them.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with BlockEmpty
// for no outcome and the reason of an outcome's own check for an invalid
// outcome. It never panics.
func NewAutomaticResolution(outcomes []SpellOutcome) (AutomaticResolution, error) {
	if err := checkOutcomes("AutomaticResolution", outcomes, false); err != nil {
		return AutomaticResolution{}, err
	}
	return AutomaticResolution{outcomes: slices.Clone(outcomes)}, nil
}

// Outcomes returns a copy of the outcomes.
func (r AutomaticResolution) Outcomes() []SpellOutcome { return slices.Clone(r.outcomes) }

// Equal reports whether r and o have the same outcomes.
func (r AutomaticResolution) Equal(o AutomaticResolution) bool {
	return slices.Equal(r.outcomes, o.outcomes)
}

func (AutomaticResolution) isPartResolution() {}

// checkPartResolution reports why r is not a resolution its constructor would
// accept: BlockMissing for nil.
func checkPartResolution(r PartResolution) error {
	var err error
	switch v := r.(type) {
	case AttackRollResolution:
		_, err = NewAttackRollResolution(v.kind, v.onHit, v.onMiss)
	case SavingThrowResolution:
		_, err = NewSavingThrowResolution(v.save, v.onFailure, v.onSuccess)
	case AutomaticResolution:
		_, err = NewAutomaticResolution(v.outcomes)
	default:
		err = blockError("PartResolution", BlockMissing)
	}
	return err
}

// SpellEffectPart is one resolution of a spell with the number of times it is
// made (spell-effect-data.md FR-042). The zero value is invalid; build it with
// [NewSpellEffectPart]. Not comparable with ==; use [SpellEffectPart.Equal].
type SpellEffectPart struct {
	resolution PartResolution
	instances  PartInstances
}

// NewSpellEffectPart returns the part resolved by resolution, made instances
// times.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the resolution's or the instances' own check for an invalid one
// (BlockMissing for a nil resolution). It never panics.
func NewSpellEffectPart(resolution PartResolution, instances PartInstances) (SpellEffectPart, error) {
	if err := checkPartResolution(resolution); err != nil {
		return SpellEffectPart{}, err
	}
	if _, err := NewPartInstances(instances.base, instances.scaling); err != nil {
		return SpellEffectPart{}, err
	}
	return SpellEffectPart{resolution: cloneResolution(resolution), instances: instances}, nil
}

// cloneResolution copies the slices of r so the part owns them.
func cloneResolution(r PartResolution) PartResolution {
	switch v := r.(type) {
	case AttackRollResolution:
		return AttackRollResolution{kind: v.kind, onHit: slices.Clone(v.onHit), onMiss: slices.Clone(v.onMiss)}
	case SavingThrowResolution:
		return SavingThrowResolution{save: v.save, onFailure: slices.Clone(v.onFailure), onSuccess: v.onSuccess}
	case AutomaticResolution:
		return AutomaticResolution{outcomes: slices.Clone(v.outcomes)}
	}
	return r
}

// Resolution returns the resolution of the part. Its slices are copies.
func (p SpellEffectPart) Resolution() PartResolution { return cloneResolution(p.resolution) }

// Instances returns how many times the resolution is made.
func (p SpellEffectPart) Instances() PartInstances { return p.instances }

// Equal reports whether p and o have equal resolutions and instances.
func (p SpellEffectPart) Equal(o SpellEffectPart) bool {
	if p.instances != o.instances {
		return false
	}
	switch a := p.resolution.(type) {
	case AttackRollResolution:
		b, ok := o.resolution.(AttackRollResolution)
		return ok && a.Equal(b)
	case SavingThrowResolution:
		b, ok := o.resolution.(SavingThrowResolution)
		return ok && a.Equal(b)
	case AutomaticResolution:
		b, ok := o.resolution.(AutomaticResolution)
		return ok && a.Equal(b)
	}
	return o.resolution == nil
}

// outcomes returns every outcome list entry of the part, in order.
func (p SpellEffectPart) outcomes() []SpellOutcome {
	switch v := p.resolution.(type) {
	case AttackRollResolution:
		return append(slices.Clone(v.onHit), v.onMiss...)
	case SavingThrowResolution:
		return v.onFailure
	case AutomaticResolution:
		return v.outcomes
	}
	return nil
}
