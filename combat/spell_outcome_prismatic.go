package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
)

// PrismaticColor is a ray of Prismatic Spray and a layer of Prismatic Wall, red to violet; the number of the ray is
// the order of the layer (SRD 5.2.1, "Prismatic Rays", pp. 154-155; "Prismatic Layers", p. 155). The zero value is
// invalid.
type PrismaticColor int

// The seven colours, in order.
const (
	// PrismaticRed is ray and layer 1: Fire damage.
	PrismaticRed PrismaticColor = iota + 1
	// PrismaticOrange is ray and layer 2: Acid damage.
	PrismaticOrange
	// PrismaticYellow is ray and layer 3: Lightning damage.
	PrismaticYellow
	// PrismaticGreen is ray and layer 4: Poison damage.
	PrismaticGreen
	// PrismaticBlue is ray and layer 5: Cold damage.
	PrismaticBlue
	// PrismaticIndigo is ray and layer 6: Restrained.
	PrismaticIndigo
	// PrismaticViolet is ray and layer 7: Blinded.
	PrismaticViolet
)

// Valid reports whether c is one of the seven colours.
func (c PrismaticColor) Valid() bool { return c >= PrismaticRed && c <= PrismaticViolet }

// Order returns 1 for red to 7 for violet, 0 for an invalid colour.
func (c PrismaticColor) Order() int {
	if !c.Valid() {
		return 0
	}
	return int(c)
}

// String returns the value of srd5e.combat.table_outcome for the ray: red to violet, or "unknown".
func (c PrismaticColor) String() string {
	return enumName([]string{"red", "orange", "yellow", "green", "blue", "indigo", "violet"}, int(c))
}

// DamageType returns the Damage Type of a damage colour (red to blue) and true, or false for indigo, violet and an
// invalid colour.
func (c PrismaticColor) DamageType() (content.DamageType, bool) {
	switch c {
	case PrismaticRed:
		return content.DamageFire, true
	case PrismaticOrange:
		return content.DamageAcid, true
	case PrismaticYellow:
		return content.DamageLightning, true
	case PrismaticGreen:
		return content.DamagePoison, true
	case PrismaticBlue:
		return content.DamageCold, true
	case PrismaticIndigo, PrismaticViolet:
		return 0, false
	}
	return 0, false
}

// prismaticRay is a row of table P: a colour, or the special 8.
type prismaticRay struct {
	color   PrismaticColor
	special bool
}

// prismaticTable returns the 1d8 table P of spell-outcome-tables.md.
func prismaticTable() outcomeTable[prismaticRay] {
	bands := make([]outcomeBand[prismaticRay], 0, 8)
	for c := PrismaticRed; c <= PrismaticViolet; c++ {
		bands = append(bands, outcomeBand[prismaticRay]{int(c), int(c), prismaticRay{color: c}})
	}
	return newOutcomeTable(dice.D8(), append(bands, outcomeBand[prismaticRay]{8, 8, prismaticRay{special: true}}))
}

// PrismaticRayFor returns the ray of the 1d8 face: the colour for 1 to 7, special true for the 8 (SRD 5.2.1,
// "Prismatic Rays", pp. 154-155). It returns an error matching ErrFaceOutOfRange for a face outside 1 to 8. It never
// panics.
func PrismaticRayFor(face int) (color PrismaticColor, special bool, err error) {
	r, err := prismaticTable().lookup(face)
	return r.color, r.special, err
}

// PrismaticStrike is the closed set of what a ray does to a target. Variants: PrismaticDamageStrike,
// PrismaticConditionStrike.
type PrismaticStrike interface{ isPrismaticStrike() }

// PrismaticDamageStrike: a damage ray. The damage is the full roll; Halved is true after a successful save, and the
// host applies HalfDamage then (SRD 5.2.1, "Half Damage", p. 16).
type PrismaticDamageStrike struct {
	damage Damage
	halved bool
}

// Damage returns the full damage of the ray.
func (s PrismaticDamageStrike) Damage() Damage { return s.damage }

// Halved reports whether the target saved, so the damage is halved.
func (s PrismaticDamageStrike) Halved() bool { return s.halved }

// PrismaticConditionStrike: indigo or violet on a failed save. The follow-up is the saves the host rolls later.
type PrismaticConditionStrike struct {
	condition content.ConditionKind
	followUp  PrismaticFollowUp
}

// Condition returns Restrained (indigo) or Blinded (violet).
func (s PrismaticConditionStrike) Condition() content.ConditionKind { return s.condition }

// FollowUp returns the later saves.
func (s PrismaticConditionStrike) FollowUp() PrismaticFollowUp { return s.followUp }

func (PrismaticDamageStrike) isPrismaticStrike()    {}
func (PrismaticConditionStrike) isPrismaticStrike() {}

// PrismaticFollowUp is the closed set of later saves of a condition ray. Variants: IndigoFollowUp,
// VioletFollowUpPending.
type PrismaticFollowUp interface{ isPrismaticFollowUp() }

// IndigoFollowUp: a Constitution save at the end of each of the target's turns, tracked by Track, which starts at
// 0 and 0. The host rolls the saves.
type IndigoFollowUp struct{ Track IndigoSaveTrack }

// VioletFollowUpPending: a Wisdom save at the start of the caster's next turn, resolved with ResolveVioletFollowUp.
type VioletFollowUpPending struct{}

func (IndigoFollowUp) isPrismaticFollowUp()        {}
func (VioletFollowUpPending) isPrismaticFollowUp() {}

// PrismaticStrikeFor returns what a ray of colour does to a target that made a Dexterity save with outcome. A damage
// colour gives a PrismaticDamageStrike of damageTotal (halved on Success). Indigo and violet give a
// PrismaticConditionStrike on Failure and nothing (false) on Success (SRD 5.2.1, "Prismatic Rays", pp. 154-155).
// It returns an error matching srderr.ErrInvalidInput for an invalid colour or a negative damageTotal. It never
// panics.
func PrismaticStrikeFor(color PrismaticColor, damageTotal int, outcome d20.Outcome) (PrismaticStrike, bool, error) {
	if t, ok := color.DamageType(); ok {
		d, err := NewDamage(damageTotal, t)
		if err != nil {
			return nil, false, err
		}
		return PrismaticDamageStrike{d, outcome == d20.Success}, true, nil
	}
	switch {
	case !color.Valid():
		return nil, false, invalidInput("prismatic color is not a ray")
	case outcome == d20.Success:
		return nil, false, nil
	case color == PrismaticIndigo:
		return PrismaticConditionStrike{content.ConditionRestrained, IndigoFollowUp{}}, true, nil
	}
	return PrismaticConditionStrike{content.ConditionBlinded, VioletFollowUpPending{}}, true, nil
}

// IndigoSaveTrack counts the successes and failures of the Constitution saves of an Indigo ray, which need not be
// consecutive. The zero value is 0 and 0. Immutable and comparable.
type IndigoSaveTrack struct{ successes, failures int }

// NewIndigoSaveTrack returns the track with successes and failures. It returns an error matching
// srderr.ErrInvalidInput for a count below 0 or 3 or more (the track would have ended). It never panics.
func NewIndigoSaveTrack(successes, failures int) (IndigoSaveTrack, error) {
	if successes < 0 || successes > 2 || failures < 0 || failures > 2 {
		return IndigoSaveTrack{}, invalidInput("indigo save counts are 0 to 2")
	}
	return IndigoSaveTrack{successes, failures}, nil
}

// Successes returns the successful saves so far.
func (t IndigoSaveTrack) Successes() int { return t.successes }

// Failures returns the failed saves so far.
func (t IndigoSaveTrack) Failures() int { return t.failures }

// IndigoStep is the closed set of results of recording a save. Variants: IndigoContinues, IndigoFreed,
// IndigoPetrified.
type IndigoStep interface{ isIndigoStep() }

// IndigoContinues: neither count reached 3; Track is the raised track.
type IndigoContinues struct{ Track IndigoSaveTrack }

// IndigoFreed: the third success; the Restrained condition ends.
type IndigoFreed struct{}

// IndigoPetrified: the third failure; the target has the Petrified condition until it is ended by an effect such as
// Greater Restoration.
type IndigoPetrified struct{}

func (IndigoContinues) isIndigoStep() {}
func (IndigoFreed) isIndigoStep()     {}
func (IndigoPetrified) isIndigoStep() {}

// Record adds a save: d20.Success counts a success and any other outcome a failure, since d20.Outcome has no third
// valid value (SRD 5.2.1, "Prismatic Rays", Indigo, p. 155).
func (t IndigoSaveTrack) Record(o d20.Outcome) IndigoStep {
	if o == d20.Success {
		t.successes++
	} else {
		t.failures++
	}
	switch {
	case t.successes == 3:
		return IndigoFreed{}
	case t.failures == 3:
		return IndigoPetrified{}
	}
	return IndigoContinues{t}
}

// VioletResult is the closed set of results of the Violet Wisdom save. Variants: VioletBlindnessEnds, VioletBanished.
type VioletResult interface{ isVioletResult() }

// VioletBlindnessEnds: the save succeeded; the Blinded condition ends.
type VioletBlindnessEnds struct{}

// VioletBanished: the save failed; the Blinded condition ends and the target is teleported to another plane of
// existence, which the host chooses.
type VioletBanished struct{}

func (VioletBlindnessEnds) isVioletResult() {}
func (VioletBanished) isVioletResult()      {}

// ResolveVioletFollowUp returns the result of the Wisdom save at the start of the caster's next turn: Success ends
// the blindness, any other outcome banishes (SRD 5.2.1, "Prismatic Rays", Violet, p. 155).
func ResolveVioletFollowUp(o d20.Outcome) VioletResult {
	if o == d20.Success {
		return VioletBlindnessEnds{}
	}
	return VioletBanished{}
}

// PrismaticSaveTarget is a creature that makes the Dexterity save of a Prismatic Spray or Wall: its id and how it
// attempts the save. Build it with a keyed literal; the zero value is invalid.
type PrismaticSaveTarget struct {
	ID      CombatantID
	Attempt d20.SaveAttempt
}

// validPrismaticTarget reports whether t has an id and a save attempt the d20 package knows.
func validPrismaticTarget(t PrismaticSaveTarget) bool {
	switch t.Attempt.(type) {
	case d20.RollAttempt, d20.ChooseToFail, d20.AutomaticFailure:
		return !t.ID.IsZero()
	}
	return false
}

// PrismaticTargetResult is what a Prismatic Spray did to one target: its rays, its one Dexterity save and a strike
// for each ray that has one. Immutable.
type PrismaticTargetResult struct {
	// ID is the target.
	ID      CombatantID
	rays    []PrismaticColor
	save    UnarmedStrikeSave
	strikes []PrismaticStrike
}

// Rays returns the rays that struck the target, in order (one or two), as a new slice.
func (r PrismaticTargetResult) Rays() []PrismaticColor { return slices.Clone(r.rays) }

// Save returns the Dexterity save, with its evidence; it applies to every ray.
func (r PrismaticTargetResult) Save() UnarmedStrikeSave { return r.save }

// Strikes returns the strike of each ray that has one, in ray order, as a new slice.
func (r PrismaticTargetResult) Strikes() []PrismaticStrike { return slices.Clone(r.strikes) }
