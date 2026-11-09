package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The Rogue levels of the Thief's features (SRD 5.2.1, "Rogue Subclass: Thief", p. 64).
const (
	thiefHandsLevel       = 3
	useMagicDeviceLevel   = 13
	thiefReflexesLevel    = 17
	thiefAttunementLimit  = 4
	chargesSavedOn        = 6
	reflexesInitiativeCut = 10
)

// FastHandsOption is something Fast Hands lets the Thief do as a Bonus Action (SRD 5.2.1, "Level 3: Fast Hands", p. 64). The
// zero value is invalid.
type FastHandsOption int

// The Fast Hands options.
const (
	// FastHandsSleightOfHandCheck: a Dexterity (Sleight of Hand) check to pick a lock or disarm a trap with Thieves' Tools, or to pick a pocket.
	FastHandsSleightOfHandCheck FastHandsOption = iota + 1
	// FastHandsUtilizeAction: the Utilize action.
	FastHandsUtilizeAction
	// FastHandsMagicActionForMagicItem: the Magic action to use a magic item that requires that action.
	FastHandsMagicActionForMagicItem
)

// SecondStoryWorkResult is the outcome of ThiefFeatures.SecondStoryWork: the closed set SecondStoryWorkGranted and
// SecondStoryWorkNotGained.
//
//sumtype:decl
type SecondStoryWorkResult interface{ isSecondStoryWorkResult() }

// SecondStoryWorkGranted: the Thief has a Climb Speed equal to its Speed and jumps using Dexterity.
type SecondStoryWorkGranted struct{ climb int }

// ClimbSpeedFeet returns the Climb Speed in feet, equal to the Speed.
func (r SecondStoryWorkGranted) ClimbSpeedFeet() int { return r.climb }

// JumpAbility returns the ability that determines the jump distance, Dexterity.
func (SecondStoryWorkGranted) JumpAbility() ability.Ability { return ability.Dexterity }

// SecondStoryWorkNotGained: the Rogue level is below 3.
type SecondStoryWorkNotGained struct{}

func (SecondStoryWorkGranted) isSecondStoryWorkResult()   {}
func (SecondStoryWorkNotGained) isSecondStoryWorkResult() {}

// ThiefFeatures holds the Thief's Fast Hands and Second-Story Work. The zero value is ready to use. The Thief subclass is
// assumed; the caller applies them only to a Thief.
type ThiefFeatures struct{}

// FastHands returns the Bonus Action options of Fast Hands at the Rogue level: three from level 3, none below.
func (ThiefFeatures) FastHands(level RogueLevel) []FastHandsOption {
	if level.n < thiefHandsLevel {
		return []FastHandsOption{}
	}
	return []FastHandsOption{FastHandsSleightOfHandCheck, FastHandsUtilizeAction, FastHandsMagicActionForMagicItem}
}

// SecondStoryWork returns the Second-Story Work benefits at the Rogue level for a creature with Speed speedFeet: granted from
// level 3. It returns an error matching srderr.ErrInvalidInput for a negative Speed.
func (ThiefFeatures) SecondStoryWork(level RogueLevel, speedFeet int) (SecondStoryWorkResult, error) {
	if speedFeet < 0 {
		return nil, invalidInput("negative Speed")
	}
	if level.n < thiefHandsLevel {
		return SecondStoryWorkNotGained{}, nil
	}
	return SecondStoryWorkGranted{climb: speedFeet}, nil
}

// AttunementLimitResult is the outcome of UseMagicDevice.AttunementLimit: the closed set AttunementLimitSet and
// AttunementLimitUnchanged.
//
//sumtype:decl
type AttunementLimitResult interface{ isAttunementLimitResult() }

// AttunementLimitSet: the Thief can attune to Limit magic items at once.
type AttunementLimitSet struct{ limit AttunementLimit }

// Limit returns the attunement limit, 4.
func (r AttunementLimitSet) Limit() AttunementLimit { return r.limit }

// AttunementLimitUnchanged: Use Magic Device does not change the attunement limit.
type AttunementLimitUnchanged struct{}

func (AttunementLimitSet) isAttunementLimitResult()       {}
func (AttunementLimitUnchanged) isAttunementLimitResult() {}

// UseMagicDevice is the Thief's Use Magic Device rule (SRD 5.2.1, "Level 13: Use Magic Device", p. 64). The zero value is ready to use.
type UseMagicDevice struct{}

// AttunementLimit returns AttunementLimitSet(4) for a Thief of Rogue level 13 or higher and AttunementLimitUnchanged below.
func (UseMagicDevice) AttunementLimit(level RogueLevel) AttunementLimitResult {
	if level.n < useMagicDeviceLevel {
		return AttunementLimitUnchanged{}
	}
	return AttunementLimitSet{limit: AttunementLimit(thiefAttunementLimit)}
}

// ChargesExpended reports whether a magic item property that expends charges does expend them: not on a d6 roll of 6, yes on 1
// to 5. It returns an error matching srderr.ErrInvalidInput for a roll outside 1 to 6.
func (UseMagicDevice) ChargesExpended(roll int) (bool, error) {
	if roll < 1 || roll > 6 {
		return false, invalidInput("d6 roll outside 1 to 6")
	}
	return roll != chargesSavedOn, nil
}

// ScrollRoute returns the route that lets a Thief of Rogue level 13 or higher use any Spell Scroll with Intelligence: a cantrip
// or level 1 spell is cast reliably and a higher spell needs an Intelligence (Arcana) check. It is the zero AnyScrollRoute (no
// route) before level 13 and for any other subclass. The DC and the outcome belong to the spell scroll rules.
func (UseMagicDevice) ScrollRoute(level RogueLevel, subclass content.SubclassID) AnyScrollRoute {
	if subclass != ThiefSubclassID || level.n < useMagicDeviceLevel {
		return AnyScrollRoute{}
	}
	route, _ := NewAnyScrollRoute(ability.Intelligence, 1, content.SkillArcana) // constants are valid
	return route
}

// ThiefReflexes is the Thief's Thief's Reflexes rule (SRD 5.2.1, "Level 17: Thief's Reflexes", p. 64). The zero value is ready to use.
type ThiefReflexes struct{}

// InitiativeTurns returns the Initiative counts of the character in the first round of combat: [initiative, initiative - 10]
// for a Thief of Rogue level 17 or higher and [initiative] otherwise. The result is for the combat initiative code.
func (ThiefReflexes) InitiativeTurns(level RogueLevel, subclass content.SubclassID, initiative int) []int {
	if subclass != ThiefSubclassID || level.n < thiefReflexesLevel {
		return []int{initiative}
	}
	return slices.Concat([]int{initiative}, []int{initiative - reflexesInitiativeCut})
}
