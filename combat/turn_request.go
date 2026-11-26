package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// TurnRequest is what TakeCombatantAction does for a combatant: spend one turn resource, or give Help,
// handle a weapon or replace an attack (combat-action-resolution.md FR-051). The effect of the action
// chosen is not resolved here (FR-052). Variants: SpendAction, SpendBonusAction, SpendReaction,
// SpendInteraction, TakeHelpAction, HandleWeapon, ReplaceAttack, UseStatBlockPart, BeginMultiattack,
// RecordMultiattackUse, RecordMultiattackReplacement.
type TurnRequest interface{ isTurnRequest() }

// SpendAction: spend the action on the choice.
type SpendAction struct{ Choice ActionChoice }

// SpendBonusAction: spend the Bonus Action a rule grants; the zero Grant means none is granted.
type SpendBonusAction struct{ Grant BonusActionGrant }

// SpendReaction: spend the Reaction on the choice, on any combatant's turn.
type SpendReaction struct{ Choice ReactionChoice }

// SpendInteraction: spend the one free interaction with an object of the turn.
type SpendInteraction struct{}

// TakeHelpAction: take the Help action and give the benefit the request describes.
type TakeHelpAction struct{ Request HelpRequest }

// HandleWeapon: equip or unequip a weapon within the Attack action in progress, spending no tracker
// resource and not the free interaction (SRD 5.2.1, "Equipping and Unequipping Weapons", p. 177).
type HandleWeapon struct {
	Change AttackActionWeaponChange
	Timing AttackActionWeaponChangeTiming
}

// ReplaceAttack: record an attack of the Attack action that another effect replaces or forgoes.
type ReplaceAttack struct {
	Cost        ReplacementCost
	Replacement AttackReplacement
}

func (SpendAction) isTurnRequest()                  {}
func (SpendBonusAction) isTurnRequest()             {}
func (SpendReaction) isTurnRequest()                {}
func (SpendInteraction) isTurnRequest()             {}
func (TakeHelpAction) isTurnRequest()               {}
func (HandleWeapon) isTurnRequest()                 {}
func (ReplaceAttack) isTurnRequest()                {}
func (UseStatBlockPart) isTurnRequest()             {}
func (BeginMultiattack) isTurnRequest()             {}
func (RecordMultiattackUse) isTurnRequest()         {}
func (RecordMultiattackReplacement) isTurnRequest() {}

// UseStatBlockPart: use one part of the monster's stat block as an action, Bonus Action, Reaction or Legendary Action:
// the use is checked against the monster and its LimitedUseState, then the tracker records the economy it names
// (FR-164). The effect of the part is not resolved (FR-052).
type UseStatBlockPart struct {
	Monster    content.Monster
	Request    StatBlockUseRequest
	LimitedUse LimitedUseState
}

// BeginMultiattack: the Multiattack of the monster begins, spending the action (FR-165). Granted is the standard
// action the part names (nil is NoStandardAction); UsedThisTurn are the parts already used this turn.
type BeginMultiattack struct {
	Monster      content.Monster
	LimitedUse   LimitedUseState
	Granted      GrantedAction
	UsedThisTurn []LimitedUsePartRef
}

// RecordMultiattackUse: record the additional use of a part of the Multiattack in progress (FR-165).
type RecordMultiattackUse struct {
	Part       LimitedUsePartRef
	LimitedUse LimitedUseState
}

// RecordMultiattackReplacement: record replacing an attack of the Multiattack in progress with the option (FR-165).
type RecordMultiattackReplacement struct {
	Option     content.ReplacementOption
	Replaced   LimitedUsePartRef
	LimitedUse LimitedUseState
}

// HelpRequest is the Help the Help action gives (SRD 5.2.1, "Help [Action]", pp. 182-183). Every fact
// the engine lacks (allegiance, distance, proficiency, whether the task allows help) is a field the host
// states (combat-action-resolution.md FR-119). Variants: AbilityCheckHelpRequest, AttackRollHelpRequest.
type HelpRequest interface{ isHelpRequest() }

// AbilityCheckHelpRequest: help the Ally with the next ability check using the proficiency.
type AbilityCheckHelpRequest struct {
	Ally                CombatantID
	Proficiency         AssistedProficiency
	HelperProficiencies HelpGiverProficiencies
	Judgement           AssistanceJudgement
}

// AttackRollHelpRequest: distract the Enemy so an ally's next attack roll against it has Advantage.
type AttackRollHelpRequest struct {
	Enemy  CombatantID
	Target AttackAssistTarget
}

func (AbilityCheckHelpRequest) isHelpRequest() {}
func (AttackRollHelpRequest) isHelpRequest()   {}

// validTurnRequest returns an error matching srderr.ErrInvalidInput for a nil request or one with an
// invalid part.
func validTurnRequest(r TurnRequest) error {
	switch v := r.(type) {
	case SpendAction:
		if v.Choice == nil || !validActionChoice(v.Choice) {
			return invalidInput("an action needs a valid choice")
		}
	case SpendBonusAction, SpendInteraction:
	case SpendReaction:
		if v.Choice == nil || !validReactionChoice(v.Choice) {
			return invalidInput("a Reaction needs a valid choice")
		}
	case TakeHelpAction:
		switch h := v.Request.(type) {
		case AbilityCheckHelpRequest:
			if h.Ally.IsZero() {
				return invalidInput("Help needs an ally")
			}
		case AttackRollHelpRequest:
			if h.Enemy.IsZero() {
				return invalidInput("Help needs an enemy")
			}
		default:
			return invalidInput("Help needs a request")
		}
	case HandleWeapon:
		if !v.Change.Valid() || !v.Timing.Valid() {
			return invalidInput("a weapon change needs a change and a timing")
		}
	case ReplaceAttack:
		if !v.Replacement.Valid() {
			return invalidInput("a replacement needs a valid replacement")
		}
		switch c := v.Cost.(type) {
		case BeginsAttackActionByReplacing:
			if !c.Allowance.Valid() {
				return invalidInput("an Attack action needs a valid allowance")
			}
		case ContinuesAttackActionByReplacing:
		default:
			return invalidInput("a replacement needs a cost")
		}
	case UseStatBlockPart:
		if v.Monster.ID() == "" || v.LimitedUse.IsZero() || v.Request.Part.IsZero() {
			return invalidInput("a stat block use needs a monster, a part and a limited-use state")
		}
	case BeginMultiattack:
		if v.Monster.ID() == "" || v.LimitedUse.IsZero() {
			return invalidInput("a Multiattack needs a monster and a limited-use state")
		}
	case RecordMultiattackUse:
		if v.Part.IsZero() || v.LimitedUse.IsZero() {
			return invalidInput("a Multiattack use needs a part and a limited-use state")
		}
	case RecordMultiattackReplacement:
		if v.Replaced.IsZero() || v.LimitedUse.IsZero() {
			return invalidInput("a Multiattack replacement needs the replaced part and a limited-use state")
		}
	default:
		return invalidInput("an action request is needed")
	}
	return nil
}

// MovementRequest is what MoveCombatant does (combat-action-resolution.md FR-056). Variants: MoveAlong,
// DropProne, StandUp. Build MoveAlong and StandUp with their constructors.
type MovementRequest interface{ isMovementRequest() }

// MoveAlong: move along the segments, in order. Build it with NewMoveAlong.
type MoveAlong struct{ segments []MoveSegment }

// DropProne: drop Prone, which uses no action and no movement (SRD 5.2.1, "Dropping Prone", p. 14).
type DropProne struct{}

// StandUp: stand up from Prone, spending half the Speed. Build it with NewStandUp.
type StandUp struct{ baseWalkingSpeedFeet int }

func (MoveAlong) isMovementRequest() {}
func (DropProne) isMovementRequest() {}
func (StandUp) isMovementRequest()   {}

// NewMoveAlong returns the request to move along segments, taking its own copy. It returns an error
// matching srderr.ErrInvalidInput for no segment or a zero segment, and a LimitExceededError for more than 256 segments.
func NewMoveAlong(segments []MoveSegment) (MoveAlong, error) {
	if err := overCap("move segments", len(segments), maxInputList); err != nil {
		return MoveAlong{}, err
	}
	if len(segments) == 0 || slices.ContainsFunc(segments, MoveSegment.IsZero) {
		return MoveAlong{}, invalidInput("a move needs segments")
	}
	return MoveAlong{slices.Clone(segments)}, nil
}

// Segments returns a copy of the segments.
func (m MoveAlong) Segments() []MoveSegment { return slices.Clone(m.segments) }

// NewStandUp returns the request to stand up, for a creature whose Speed without conditions is
// baseWalkingSpeedFeet. It returns an error matching srderr.ErrInvalidInput for a negative Speed.
func NewStandUp(baseWalkingSpeedFeet int) (StandUp, error) {
	if baseWalkingSpeedFeet < 0 {
		return StandUp{}, invalidInput("a stand up needs a walking Speed of 0 or more")
	}
	return StandUp{baseWalkingSpeedFeet}, nil
}

// BaseWalkingSpeedFeet returns the walking Speed the stand-up cost is half of, before conditions.
func (s StandUp) BaseWalkingSpeedFeet() int { return s.baseWalkingSpeedFeet }
