package combat

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Every use case below wraps exactly one TurnTracker operation: it runs it through the
// instrumenter, adds no rule, refusal or event and maps the TrackerStep one to one onto
// its own closed result (initiative-and-actions.md FR-149). Each is built by a
// NewX(opts ...Option) constructor (WithTelemetry, WithClock), its zero value is usable
// and silent, and Run returns an error matching srderr.ErrInvalidInput, before any signal,
// for the zero command, and ctx.Err() unwrapped, before any signal, for a cancelled ctx. A
// refusal is a Rejected result, never an error. They roll no dice and are safe for
// concurrent use if their telemetry and clock are.

// turnTarget checks the tracker and the combatant every tracker-backed command carries.
func turnTarget(t TurnTracker, id CombatantID) error {
	if t.IsZero() || id.IsZero() {
		return invalidInput("a turn command needs a tracker and a combatant id")
	}
	return nil
}

// AdvanceTrackerTurnCommand is the immutable input of AdvanceTrackerTurn.Run. Build it
// with NewAdvanceTrackerTurnCommand; the zero value is invalid.
type AdvanceTrackerTurnCommand struct {
	tracker TurnTracker
	advance TurnAdvance
}

// NewAdvanceTrackerTurnCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank or repeated id among the updates
// or a size that is not defined. An id not in the tracker is not an error: Run refuses it.
// It returns a LimitExceededError for more than 256 updates or shared-space occupants
// (plan P10). The lists are copied. It never panics.
func NewAdvanceTrackerTurnCommand(tracker TurnTracker, advance TurnAdvance) (AdvanceTrackerTurnCommand, error) {
	if err := advanceOverCap(advance); err != nil {
		return AdvanceTrackerTurnCommand{}, err
	}
	seen := map[CombatantID]bool{}
	for _, u := range advance.Updates {
		if u.Combatant.IsZero() || seen[u.Combatant] {
			return AdvanceTrackerTurnCommand{}, invalidInput("an update names a blank or repeated combatant")
		}
		seen[u.Combatant] = true
	}
	if tracker.IsZero() || slices.ContainsFunc(advance.SharedSpaceOccupants, func(s content.CreatureSize) bool { return !s.Valid() }) {
		return AdvanceTrackerTurnCommand{}, invalidInput("advance needs a tracker and valid sizes")
	}
	advance.Updates, advance.SharedSpaceOccupants = slices.Clone(advance.Updates), slices.Clone(advance.SharedSpaceOccupants)
	return AdvanceTrackerTurnCommand{tracker, advance}, nil
}

// AdvanceTrackerTurnResult is the closed set of outcomes of AdvanceTrackerTurn.Run.
// Variants: AdvanceTrackerTurnApplied, AdvanceTrackerTurnRejected.
type AdvanceTrackerTurnResult interface{ isAdvanceTrackerTurnResult() }

// AdvanceTrackerTurnApplied: the turn ended and the next began.
type AdvanceTrackerTurnApplied struct{ appliedTurnStep }

// AdvanceTrackerTurnRejected: an update named a combatant not in the tracker.
type AdvanceTrackerTurnRejected struct{ Reason TurnRejection }

func (AdvanceTrackerTurnApplied) isAdvanceTrackerTurnResult()  {}
func (AdvanceTrackerTurnRejected) isAdvanceTrackerTurnResult() {}

// AdvanceTrackerTurn is the use case of ending the active turn (TurnTracker.Advance).
type AdvanceTrackerTurn struct{ turnUse }

// NewAdvanceTrackerTurn returns the use case.
func NewAdvanceTrackerTurn(opts ...Option) AdvanceTrackerTurn {
	return AdvanceTrackerTurn{newTurnUse(opts)}
}

// Run advances the turn. See TurnTracker.Advance for the events and refusals.
func (u AdvanceTrackerTurn) Run(ctx context.Context, cmd AdvanceTrackerTurnCommand) (AdvanceTrackerTurnResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationAdvanceTrackerTurn, spanAdvanceTrackerTurn}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.Advance(cmd.advance) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) AdvanceTrackerTurnResult { return AdvanceTrackerTurnApplied{a} },
		func(r TurnRejection) AdvanceTrackerTurnResult { return AdvanceTrackerTurnRejected{r} }), nil
}

// UpdateMovementSpeedsCommand is the immutable input of UpdateMovementSpeeds.Run.
type UpdateMovementSpeedsCommand struct {
	tracker   TurnTracker
	combatant CombatantID
	speeds    MovementSpeeds
}

// NewUpdateMovementSpeedsCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker or a blank id.
func NewUpdateMovementSpeedsCommand(tracker TurnTracker, combatant CombatantID, speeds MovementSpeeds) (UpdateMovementSpeedsCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return UpdateMovementSpeedsCommand{}, err
	}
	return UpdateMovementSpeedsCommand{tracker, combatant, speeds}, nil
}

// UpdateMovementSpeedsResult is the closed set of outcomes of UpdateMovementSpeeds.Run.
// Variants: UpdateMovementSpeedsApplied, UpdateMovementSpeedsRejected.
type UpdateMovementSpeedsResult interface{ isUpdateMovementSpeedsResult() }

// UpdateMovementSpeedsApplied: the profile speeds were replaced.
type UpdateMovementSpeedsApplied struct{ appliedTurnStep }

// UpdateMovementSpeedsRejected: the combatant is not in the tracker.
type UpdateMovementSpeedsRejected struct{ Reason TurnRejection }

func (UpdateMovementSpeedsApplied) isUpdateMovementSpeedsResult()  {}
func (UpdateMovementSpeedsRejected) isUpdateMovementSpeedsResult() {}

// UpdateMovementSpeeds is the use case of replacing a combatant's speeds.
type UpdateMovementSpeeds struct{ turnUse }

// NewUpdateMovementSpeeds returns the use case.
func NewUpdateMovementSpeeds(opts ...Option) UpdateMovementSpeeds {
	return UpdateMovementSpeeds{newTurnUse(opts)}
}

// Run replaces the speeds (TurnTracker.UpdateMovementSpeeds).
func (u UpdateMovementSpeeds) Run(ctx context.Context, cmd UpdateMovementSpeedsCommand) (UpdateMovementSpeedsResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationUpdateMovementSpeeds, spanUpdateMovementSpeeds}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.UpdateMovementSpeeds(cmd.combatant, cmd.speeds) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) UpdateMovementSpeedsResult { return UpdateMovementSpeedsApplied{a} },
		func(r TurnRejection) UpdateMovementSpeedsResult { return UpdateMovementSpeedsRejected{r} }), nil
}

// TakeActionCommand is the immutable input of TakeAction.Run.
type TakeActionCommand struct {
	tracker      TurnTracker
	combatant    CombatantID
	availability ConditionActionAvailability
	choice       ActionChoice
}

// NewTakeActionCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank id, an availability that is not
// defined, a nil or invalid choice, or a Ready that holds a spell whose caster is not the
// acting combatant.
func NewTakeActionCommand(tracker TurnTracker, combatant CombatantID, av ConditionActionAvailability, choice ActionChoice) (TakeActionCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return TakeActionCommand{}, err
	}
	if !validAvailability(av) || !validActionChoice(choice) {
		return TakeActionCommand{}, invalidInput("action needs an availability and a valid choice")
	}
	if r, ok := choice.(ReadyChoice); ok {
		if h, ok := r.Action.Held().(HoldsSpell); ok && h.Spell.Caster != combatant {
			return TakeActionCommand{}, invalidInput("the held spell's caster is not the acting combatant")
		}
	}
	return TakeActionCommand{tracker, combatant, av, choice}, nil
}

// TakeActionResult is the closed set of outcomes of TakeAction.Run. Variants:
// TakeActionApplied, TakeActionRejected.
type TakeActionResult interface{ isTakeActionResult() }

// TakeActionApplied: the action was taken.
type TakeActionApplied struct{ appliedTurnStep }

// TakeActionRejected: the rules refuse the action.
type TakeActionRejected struct{ Reason TurnRejection }

func (TakeActionApplied) isTakeActionResult()  {}
func (TakeActionRejected) isTakeActionResult() {}

// TakeAction is the use case of taking the turn's action.
type TakeAction struct{ turnUse }

// NewTakeAction returns the use case.
func NewTakeAction(opts ...Option) TakeAction { return TakeAction{newTurnUse(opts)} }

// Run takes the action (TurnTracker.TakeAction).
func (u TakeAction) Run(ctx context.Context, cmd TakeActionCommand) (TakeActionResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationTakeAction, spanTakeAction}, cmd.tracker,
		func() (TrackerStep, error) {
			return cmd.tracker.TakeAction(cmd.combatant, cmd.availability, cmd.choice)
		})
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) TakeActionResult { return TakeActionApplied{a} },
		func(r TurnRejection) TakeActionResult { return TakeActionRejected{r} }), nil
}

// TakeBonusActionCommand is the immutable input of TakeBonusAction.Run.
type TakeBonusActionCommand struct {
	tracker      TurnTracker
	combatant    CombatantID
	availability ConditionActionAvailability
	entitlement  BonusActionEntitlement
}

// NewTakeBonusActionCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank id, an availability that is not
// defined, a nil entitlement or an entitled zero grant.
func NewTakeBonusActionCommand(tracker TurnTracker, combatant CombatantID, av ConditionActionAvailability, e BonusActionEntitlement) (TakeBonusActionCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return TakeBonusActionCommand{}, err
	}
	if g, ok := e.(EntitledToBonusAction); !validAvailability(av) || (e == nil) || (ok && g.Grant.IsZero()) {
		return TakeBonusActionCommand{}, invalidInput("Bonus Action needs an availability and an entitlement")
	}
	return TakeBonusActionCommand{tracker, combatant, av, e}, nil
}

// TakeBonusActionResult is the closed set of outcomes of TakeBonusAction.Run. Variants:
// TakeBonusActionApplied, TakeBonusActionRejected.
type TakeBonusActionResult interface{ isTakeBonusActionResult() }

// TakeBonusActionApplied: the Bonus Action was used.
type TakeBonusActionApplied struct{ appliedTurnStep }

// TakeBonusActionRejected: the rules refuse the Bonus Action.
type TakeBonusActionRejected struct{ Reason TurnRejection }

func (TakeBonusActionApplied) isTakeBonusActionResult()  {}
func (TakeBonusActionRejected) isTakeBonusActionResult() {}

// TakeBonusAction is the use case of taking a Bonus Action.
type TakeBonusAction struct{ turnUse }

// NewTakeBonusAction returns the use case.
func NewTakeBonusAction(opts ...Option) TakeBonusAction { return TakeBonusAction{newTurnUse(opts)} }

// Run takes the Bonus Action (TurnTracker.TakeBonusAction).
func (u TakeBonusAction) Run(ctx context.Context, cmd TakeBonusActionCommand) (TakeBonusActionResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationTakeBonusAction, spanTakeBonusAction}, cmd.tracker,
		func() (TrackerStep, error) {
			return cmd.tracker.TakeBonusAction(cmd.combatant, cmd.availability, cmd.entitlement)
		})
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) TakeBonusActionResult { return TakeBonusActionApplied{a} },
		func(r TurnRejection) TakeBonusActionResult { return TakeBonusActionRejected{r} }), nil
}

// TakeReactionCommand is the immutable input of TakeReaction.Run.
type TakeReactionCommand struct {
	tracker      TurnTracker
	combatant    CombatantID
	availability ConditionActionAvailability
	choice       ReactionChoice
}

// NewTakeReactionCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank id, an availability that is not
// defined, a nil or invalid choice, or an Opportunity Attack that names the reactor as the
// provoker.
func NewTakeReactionCommand(tracker TurnTracker, combatant CombatantID, av ConditionActionAvailability, choice ReactionChoice) (TakeReactionCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return TakeReactionCommand{}, err
	}
	if !validAvailability(av) || !validReactionChoice(choice) {
		return TakeReactionCommand{}, invalidInput("reaction needs an availability and a valid choice")
	}
	if oa, ok := choice.(OpportunityAttackReaction); ok && oa.Provoker == combatant {
		return TakeReactionCommand{}, invalidInput("an Opportunity Attack cannot name its observer as the provoker")
	}
	return TakeReactionCommand{tracker, combatant, av, choice}, nil
}

// TakeReactionResult is the closed set of outcomes of TakeReaction.Run. Variants:
// TakeReactionApplied, TakeReactionRejected.
type TakeReactionResult interface{ isTakeReactionResult() }

// TakeReactionApplied: the Reaction was spent.
type TakeReactionApplied struct{ appliedTurnStep }

// TakeReactionRejected: the rules refuse the Reaction.
type TakeReactionRejected struct{ Reason TurnRejection }

func (TakeReactionApplied) isTakeReactionResult()  {}
func (TakeReactionRejected) isTakeReactionResult() {}

// TakeReaction is the use case of taking a Reaction.
type TakeReaction struct{ turnUse }

// NewTakeReaction returns the use case.
func NewTakeReaction(opts ...Option) TakeReaction { return TakeReaction{newTurnUse(opts)} }

// Run takes the Reaction (TurnTracker.TakeReaction); an accepted readied move records
// srd5e.combat.turn_movement_spent.
func (u TakeReaction) Run(ctx context.Context, cmd TakeReactionCommand) (TakeReactionResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationTakeReaction, spanTakeReaction}, cmd.tracker,
		func() (TrackerStep, error) {
			return cmd.tracker.TakeReaction(cmd.combatant, cmd.availability, cmd.choice)
		})
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) TakeReactionResult { return TakeReactionApplied{a} },
		func(r TurnRejection) TakeReactionResult { return TakeReactionRejected{r} }), nil
}

// MoveCommand is the immutable input of Move.Run.
type MoveCommand struct {
	tracker   TurnTracker
	combatant CombatantID
	segments  []MoveSegment
}

// NewMoveCommand returns the command. It returns an error matching srderr.ErrInvalidInput
// for the zero tracker, a blank id, no segment or a zero segment. It returns a
// LimitExceededError for more than 256 segments (plan P10). The segments are copied.
func NewMoveCommand(tracker TurnTracker, combatant CombatantID, segments []MoveSegment) (MoveCommand, error) {
	if err := overCap("move segments", len(segments), maxInputList); err != nil {
		return MoveCommand{}, err
	}
	if err := turnTarget(tracker, combatant); err != nil {
		return MoveCommand{}, err
	}
	if len(segments) == 0 || slices.ContainsFunc(segments, MoveSegment.IsZero) {
		return MoveCommand{}, invalidInput("a move needs segments")
	}
	return MoveCommand{tracker, combatant, slices.Clone(segments)}, nil
}

// MoveResult is the closed set of outcomes of Move.Run. Variants: MoveApplied, MoveRejected.
type MoveResult interface{ isMoveResult() }

// MoveApplied: the combatant moved.
type MoveApplied struct{ appliedTurnStep }

// MoveRejected: the rules refuse the move; nothing was applied.
type MoveRejected struct{ Reason TurnRejection }

func (MoveApplied) isMoveResult()  {}
func (MoveRejected) isMoveResult() {}

// Move is the use case of moving the active combatant.
type Move struct{ turnUse }

// NewMove returns the use case.
func NewMove(opts ...Option) Move { return Move{newTurnUse(opts)} }

// Run moves (TurnTracker.Move); an accepted move records srd5e.combat.turn_movement_spent.
func (u Move) Run(ctx context.Context, cmd MoveCommand) (MoveResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationMove, spanMove}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.Move(cmd.combatant, cmd.segments) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) MoveResult { return MoveApplied{a} },
		func(r TurnRejection) MoveResult { return MoveRejected{r} }), nil
}

// InteractWithObjectCommand is the immutable input of InteractWithObject.Run.
type InteractWithObjectCommand struct {
	tracker   TurnTracker
	combatant CombatantID
}

// NewInteractWithObjectCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker or a blank id.
func NewInteractWithObjectCommand(tracker TurnTracker, combatant CombatantID) (InteractWithObjectCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return InteractWithObjectCommand{}, err
	}
	return InteractWithObjectCommand{tracker, combatant}, nil
}

// InteractWithObjectResult is the closed set of outcomes of InteractWithObject.Run.
// Variants: InteractWithObjectApplied, InteractWithObjectRejected.
type InteractWithObjectResult interface{ isInteractWithObjectResult() }

// InteractWithObjectApplied: the free interaction was used.
type InteractWithObjectApplied struct{ appliedTurnStep }

// InteractWithObjectRejected: the rules refuse the interaction.
type InteractWithObjectRejected struct{ Reason TurnRejection }

func (InteractWithObjectApplied) isInteractWithObjectResult()  {}
func (InteractWithObjectRejected) isInteractWithObjectResult() {}

// InteractWithObject is the use case of the free interaction with an object.
type InteractWithObject struct{ turnUse }

// NewInteractWithObject returns the use case.
func NewInteractWithObject(opts ...Option) InteractWithObject {
	return InteractWithObject{newTurnUse(opts)}
}

// Run interacts (TurnTracker.InteractWithObject).
func (u InteractWithObject) Run(ctx context.Context, cmd InteractWithObjectCommand) (InteractWithObjectResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationInteractWithObject, spanInteractWithObject}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.InteractWithObject(cmd.combatant) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) InteractWithObjectResult { return InteractWithObjectApplied{a} },
		func(r TurnRejection) InteractWithObjectResult { return InteractWithObjectRejected{r} }), nil
}

// ChangePostureCommand is the immutable input of ChangePosture.Run.
type ChangePostureCommand struct {
	tracker   TurnTracker
	combatant CombatantID
	change    PostureChange
}

// NewChangePostureCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank id, a nil change or a stand-up
// without a ProneMovement.
func NewChangePostureCommand(tracker TurnTracker, combatant CombatantID, change PostureChange) (ChangePostureCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return ChangePostureCommand{}, err
	}
	switch v := change.(type) {
	case DropProneChange:
	case StandUpChange:
		switch m := v.Movement.(type) {
		case CrawlOnly:
		case CrawlOrRightItself:
			if m.CostFeet < 0 {
				return ChangePostureCommand{}, invalidInput("negative stand-up cost")
			}
		default:
			return ChangePostureCommand{}, invalidInput("stand up needs a ProneMovement")
		}
	default:
		return ChangePostureCommand{}, invalidInput("posture needs a change")
	}
	return ChangePostureCommand{tracker, combatant, change}, nil
}

// ChangePostureResult is the closed set of outcomes of ChangePosture.Run. Variants:
// ChangePostureApplied, ChangePostureRejected.
type ChangePostureResult interface{ isChangePostureResult() }

// ChangePostureApplied: the combatant dropped Prone or stood up.
type ChangePostureApplied struct{ appliedTurnStep }

// ChangePostureRejected: the rules refuse the change.
type ChangePostureRejected struct{ Reason TurnRejection }

func (ChangePostureApplied) isChangePostureResult()  {}
func (ChangePostureRejected) isChangePostureResult() {}

// ChangePosture is the use case of dropping Prone or standing up.
type ChangePosture struct{ turnUse }

// NewChangePosture returns the use case.
func NewChangePosture(opts ...Option) ChangePosture { return ChangePosture{newTurnUse(opts)} }

// Run changes posture (TurnTracker.ChangePosture); an accepted stand-up records
// srd5e.combat.turn_movement_spent with the mode posture.
func (u ChangePosture) Run(ctx context.Context, cmd ChangePostureCommand) (ChangePostureResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationChangePosture, spanChangePosture}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.ChangePosture(cmd.combatant, cmd.change) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) ChangePostureResult { return ChangePostureApplied{a} },
		func(r TurnRejection) ChangePostureResult { return ChangePostureRejected{r} }), nil
}

// GrantAdditionalActionCommand is the immutable input of GrantAdditionalAction.Run.
type GrantAdditionalActionCommand struct {
	tracker   TurnTracker
	combatant CombatantID
	grant     AdditionalActionGrant
}

// NewGrantAdditionalActionCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank id or the zero grant.
func NewGrantAdditionalActionCommand(tracker TurnTracker, combatant CombatantID, grant AdditionalActionGrant) (GrantAdditionalActionCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return GrantAdditionalActionCommand{}, err
	}
	if grant.IsZero() {
		return GrantAdditionalActionCommand{}, invalidInput("additional action needs a grant")
	}
	return GrantAdditionalActionCommand{tracker, combatant, grant}, nil
}

// GrantAdditionalActionResult is the closed set of outcomes of GrantAdditionalAction.Run.
// Variants: GrantAdditionalActionApplied, GrantAdditionalActionRejected.
type GrantAdditionalActionResult interface{ isGrantAdditionalActionResult() }

// GrantAdditionalActionApplied: the additional action is granted for the turn.
type GrantAdditionalActionApplied struct{ appliedTurnStep }

// GrantAdditionalActionRejected: the rules refuse the grant.
type GrantAdditionalActionRejected struct{ Reason TurnRejection }

func (GrantAdditionalActionApplied) isGrantAdditionalActionResult()  {}
func (GrantAdditionalActionRejected) isGrantAdditionalActionResult() {}

// GrantAdditionalAction is the use case of granting an additional action for a turn.
type GrantAdditionalAction struct{ turnUse }

// NewGrantAdditionalAction returns the use case.
func NewGrantAdditionalAction(opts ...Option) GrantAdditionalAction {
	return GrantAdditionalAction{newTurnUse(opts)}
}

// Run grants the action (TurnTracker.GrantAdditionalAction).
func (u GrantAdditionalAction) Run(ctx context.Context, cmd GrantAdditionalActionCommand) (GrantAdditionalActionResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationGrantAdditionalAction, spanGrantAdditionalAction}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.GrantAdditionalAction(cmd.combatant, cmd.grant) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) GrantAdditionalActionResult { return GrantAdditionalActionApplied{a} },
		func(r TurnRejection) GrantAdditionalActionResult { return GrantAdditionalActionRejected{r} }), nil
}

// RecordFeatUseCommand is the immutable input of RecordFeatUse.Run.
type RecordFeatUseCommand struct {
	tracker   TurnTracker
	combatant CombatantID
	feat      content.FeatID
	limit     character.FeatUsageLimit
}

// NewRecordFeatUseCommand returns the command. It returns an error matching
// srderr.ErrInvalidInput for the zero tracker, a blank id, an invalid feat id or a limit
// other than OncePerTurn and OnceUntilStartOfYourNextTurn (the others are not turn state).
func NewRecordFeatUseCommand(tracker TurnTracker, combatant CombatantID, feat content.FeatID, limit character.FeatUsageLimit) (RecordFeatUseCommand, error) {
	if err := turnTarget(tracker, combatant); err != nil {
		return RecordFeatUseCommand{}, err
	}
	switch limit.(type) {
	case character.OncePerTurn, character.OnceUntilStartOfYourNextTurn:
	default:
		return RecordFeatUseCommand{}, invalidInput("only a once per turn or once until start of next turn limit is turn state")
	}
	if !feat.Valid() {
		return RecordFeatUseCommand{}, invalidInput("feat id")
	}
	return RecordFeatUseCommand{tracker, combatant, feat, limit}, nil
}

// RecordFeatUseResult is the closed set of outcomes of RecordFeatUse.Run. Variants:
// RecordFeatUseApplied, RecordFeatUseRejected.
type RecordFeatUseResult interface{ isRecordFeatUseResult() }

// RecordFeatUseApplied: the use was recorded.
type RecordFeatUseApplied struct{ appliedTurnStep }

// RecordFeatUseRejected: the use is already recorded in its window, or the combatant is unknown.
type RecordFeatUseRejected struct{ Reason TurnRejection }

func (RecordFeatUseApplied) isRecordFeatUseResult()  {}
func (RecordFeatUseRejected) isRecordFeatUseResult() {}

// RecordFeatUse is the use case of recording a turn-limited feat use.
type RecordFeatUse struct{ turnUse }

// NewRecordFeatUse returns the use case.
func NewRecordFeatUse(opts ...Option) RecordFeatUse { return RecordFeatUse{newTurnUse(opts)} }

// Run records the use (TurnTracker.RecordFeatUse).
func (u RecordFeatUse) Run(ctx context.Context, cmd RecordFeatUseCommand) (RecordFeatUseResult, error) {
	step, err := u.step(ctx, combatOperation{valueOperationRecordFeatUse, spanRecordFeatUse}, cmd.tracker,
		func() (TrackerStep, error) { return cmd.tracker.RecordFeatUse(cmd.combatant, cmd.feat, cmd.limit) })
	if err != nil {
		return nil, err
	}
	return convertStep(step, func(a appliedTurnStep) RecordFeatUseResult { return RecordFeatUseApplied{a} },
		func(r TurnRejection) RecordFeatUseResult { return RecordFeatUseRejected{r} }), nil
}
