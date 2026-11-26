package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// SpellFunctioning says whether an ongoing spell works in the place it is. An
// Antimagic Field suppresses it while it is inside the aura (SRD 5.2.1, "Antimagic
// Field", p. 109). The zero value is not a state.
type SpellFunctioning int

const (
	// Functioning: the spell works.
	Functioning SpellFunctioning = iota + 1
	// Suppressed: an Antimagic Field holds the spell still; it keeps losing time.
	Suppressed
)

// String returns "functioning", "suppressed" or "unknown".
func (f SpellFunctioning) String() string {
	return enumName([]string{"functioning", "suppressed"}, int(f))
}

// AntimagicActivity is what a creature tries to do that an Antimagic Field may block
// (SRD 5.2.1, "Antimagic Field", p. 109). The zero value is not an activity.
type AntimagicActivity int

const (
	// ActivityCastSpell: casting a spell.
	ActivityCastSpell AntimagicActivity = iota + 1
	// ActivityTakeMagicAction: taking a Magic action.
	ActivityTakeMagicAction
	// ActivityCreateMagicalEffect: creating another magical effect.
	ActivityCreateMagicalEffect
	// ActivityUseMagicItemProperty: a magic item's property working (targets are not
	// consulted for it).
	ActivityUseMagicItemProperty
	// ActivityTeleport: teleporting, or using planar travel, into or out of the aura.
	ActivityTeleport
)

// Valid reports whether a is an activity.
func (a AntimagicActivity) Valid() bool {
	return a >= ActivityCastSpell && a <= ActivityTeleport
}

// String returns the snake_case name of the activity, or "unknown".
func (a AntimagicActivity) String() string {
	return enumName([]string{"cast_spell", "take_magic_action", "create_magical_effect", "use_magic_item_property", "teleport"}, int(a))
}

// AntimagicBlock is why an Antimagic Field blocks an activity. The zero value is not a reason.
type AntimagicBlock int

const (
	// AntimagicBlockActorInsideAura: the actor is inside the aura (or, for a magic
	// item, the item is, or for teleport the origin or destination is).
	AntimagicBlockActorInsideAura AntimagicBlock = iota + 1
	// AntimagicBlockTargetInsideAura: a target of the activity is inside the aura.
	AntimagicBlockTargetInsideAura
)

// String returns "actor_inside_aura", "target_inside_aura" or "unknown".
func (b AntimagicBlock) String() string {
	return enumName([]string{"actor_inside_aura", "target_inside_aura"}, int(b))
}

// AntimagicVerdict is whether an activity is allowed inside or against an Antimagic
// Field. It is a closed set: AntimagicPermits and AntimagicBlocks.
type AntimagicVerdict interface {
	isAntimagicVerdict()
}

// AntimagicPermits is the verdict that the activity may go ahead.
type AntimagicPermits struct{}

// AntimagicBlocks is the verdict that the activity is blocked: Block says why, and
// Targets lists the targets inside the aura when the reason is a target.
type AntimagicBlocks struct {
	block   AntimagicBlock
	targets []SpellTargetRef
}

// Block returns why the activity is blocked.
func (b AntimagicBlocks) Block() AntimagicBlock { return b.block }

// Targets returns a copy of the targets inside the aura that blocked the activity.
func (b AntimagicBlocks) Targets() []SpellTargetRef { return slices.Clone(b.targets) }

func (AntimagicPermits) isAntimagicVerdict() {}
func (AntimagicBlocks) isAntimagicVerdict()  {}

// AntimagicField is the Antimagic Field spell's rules (SRD 5.2.1, p. 109), stateless:
// use the zero value. Status says whether a spell works; Blocks says whether an activity
// is allowed. Suppression is derived from the aura each time and is never stored.
type AntimagicField struct{}

// antimagicExemptSpells are the spells the field does not suppress: another field "don't
// nullify each other", and the field "has no effect on the wall" (SRD 5.2.1, pp. 109, 155).
func antimagicExempt(id content.SpellID) bool {
	return id == "antimagic_field" || id == "prismatic_wall"
}

// Status returns Suppressed exactly when insideAura is true, the spell's origin is
// ordinary and the spell is not exempt (FR-043 to FR-051): an Artifact or deity's
// spell, another Antimagic Field and a Prismatic Wall keep working. It holds no state.
func (AntimagicField) Status(spell OngoingSpell, insideAura bool) SpellFunctioning {
	switch {
	case !insideAura, spell.Origin() != SpellOriginOrdinary, antimagicExempt(spell.SpellID()):
		return Functioning
	default:
		return Suppressed
	}
}

// Blocks decides an activity (FR-045 to FR-047). actorInside is whether the actor is
// inside the aura, which for a magic item property means the item or anything it is on,
// and for teleport means the origin or destination. targetsInside are the targets inside the
// aura; they are ignored for ActivityUseMagicItemProperty. The verdict is blocked by the
// actor first, then by a target. It returns an error matching srderr.ErrInvalidInput for an
// activity that is not defined, or for targetsInside that names the same target twice
// (not checked for ActivityUseMagicItemProperty), and a LimitExceededError for more than 256 targetsInside (checked for every
// activity, plan P10). It never panics.
func (AntimagicField) Blocks(activity AntimagicActivity, actorInside bool, targetsInside []SpellTargetRef) (AntimagicVerdict, error) {
	if !activity.Valid() {
		return nil, invalidInput("an antimagic activity must be defined")
	}
	if err := overCap("antimagic targets inside", len(targetsInside), maxInputList); err != nil {
		return nil, err
	}
	if activity == ActivityUseMagicItemProperty {
		if actorInside {
			return AntimagicBlocks{block: AntimagicBlockActorInsideAura}, nil
		}
		return AntimagicPermits{}, nil
	}
	seen := make(map[SpellTargetRef]bool, len(targetsInside))
	for _, t := range targetsInside {
		if seen[t] {
			return nil, invalidInput("targetsInside must not name a target twice")
		}
		seen[t] = true
	}
	switch {
	case actorInside:
		return AntimagicBlocks{block: AntimagicBlockActorInsideAura}, nil
	case len(targetsInside) > 0:
		return AntimagicBlocks{block: AntimagicBlockTargetInsideAura, targets: slices.Clone(targetsInside)}, nil
	default:
		return AntimagicPermits{}, nil
	}
}
