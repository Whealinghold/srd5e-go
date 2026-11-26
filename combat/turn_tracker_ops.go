package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// All operations below are pure, synchronous and silent. They return an error matching
// srderr.ErrInvalidInput only for a malformed argument (a nil or undefined choice, an
// empty list); a refusal is a TrackerStepRejected with the first reason of the
// operation's order (initiative-and-actions.md FR-118), leaving the receiver
// unchanged. An id not in the tracker is TurnUnknownCombatantRejection, never a panic.

func stepRejected(r TurnRejection) (TrackerStep, error) { return TrackerStepRejected{r}, nil }

func stepApplied(t TurnTracker, events ...TurnEvent) (TrackerStep, error) {
	return TrackerStepApplied{t, events}, nil
}

// gate checks the id, the availability and, when active is set, the turn: the first three
// reasons of TakeAction, TakeBonusAction, Move and the rest.
func (t TurnTracker) gate(id CombatantID, av ConditionActionAvailability, needActive bool) TurnRejection {
	switch {
	case t.profiles[id].ID != id || id.IsZero():
		return TurnUnknownCombatantRejection{id}
	case av == ConditionActionAvailabilityNone:
		return TurnIncapacitatedRejection{}
	case needActive && t.order[t.active].Combatant != id:
		return TurnNotActiveCombatantRejection{}
	}
	return nil
}

// startTurn starts id's turn on n: fresh resources, its Reaction regained, its Dodge ended,
// the additional action cleared, the turn-limited feat uses cleared and a readied action
// expired (FR-034, FR-035, FR-037, FR-038, FR-157). It returns the expiry event, if any.
func (t *TurnTracker) startTurn(id CombatantID) []TurnEvent {
	t.res = freshTurnResources()
	t.extra = extraAction{}
	delete(t.spent, id)
	delete(t.dodging, id)
	for k := range t.feats {
		if k.window == featWindowTurn || k.id == id {
			delete(t.feats, k)
		}
	}
	if a, ok := t.readied[id]; ok {
		delete(t.readied, id)
		return []TurnEvent{ReadiedActionExpired{combatant: id, trigger: a.trigger, held: a.Held()}}
	}
	return nil
}

// SpeedUpdate replaces one combatant's profile speeds. Build it with a keyed literal.
type SpeedUpdate struct {
	Combatant CombatantID
	Speeds    MovementSpeeds
}

// TurnAdvance holds what Advance takes: updated speeds for any combatant, in the order
// supplied (a later update of the same combatant wins), and the sizes of the other
// creatures sharing the ending combatant's space.
type TurnAdvance struct {
	Updates              []SpeedUpdate
	SharedSpaceOccupants []content.CreatureSize
}

// advanceOverCap returns the LimitExceededError of an advance with more than 256 updates or shared-space occupants.
func advanceOverCap(a TurnAdvance) error {
	if err := overCap("speed updates", len(a.Updates), maxInputList); err != nil {
		return err
	}
	return overCap("shared-space occupants", len(a.SharedSpaceOccupants), maxInputList)
}

// Advance ends the active combatant's turn and starts the next (SRD 5.2.1, "The Order
// of Combat", p. 13), with the events in this order: TrackerTurnEnded, FellProneInSharedSpace
// (unless the ending creature is Tiny or larger than every sharer), MovementSpeedsUpdated
// in turn order, TrackerRoundStarted (when the last combatant ended), TrackerTurnStarted,
// ReadiedActionExpired. An update for an id not in the tracker is refused with the first
// such id and nothing is applied. It returns an error matching srderr.ErrInvalidInput for
// the zero tracker or a size that is not defined, and a
// LimitExceededError for more than 256 updates or shared-space occupants (plan P10).
func (t TurnTracker) Advance(a TurnAdvance) (TrackerStep, error) {
	if err := advanceOverCap(a); err != nil {
		return nil, err
	}
	if t.IsZero() || slices.ContainsFunc(a.SharedSpaceOccupants, func(s content.CreatureSize) bool { return !s.Valid() }) {
		return nil, invalidInput("advance needs a tracker and valid sizes")
	}
	for _, u := range a.Updates {
		if _, ok := t.profiles[u.Combatant]; !ok {
			return stepRejected(TurnUnknownCombatantRejection{u.Combatant})
		}
	}
	n := t.clone()
	ending := t.order[t.active].Combatant
	events := []TurnEvent{TrackerTurnEnded{combatant: ending, round: t.round}}
	if size := t.profiles[ending].Size; len(a.SharedSpaceOccupants) > 0 && size != content.SizeTiny &&
		slices.ContainsFunc(a.SharedSpaceOccupants, func(o content.CreatureSize) bool { return size <= o }) {
		events = append(events, FellProneInSharedSpace{combatant: ending})
	}
	latest := map[CombatantID]MovementSpeeds{}
	for _, u := range a.Updates {
		latest[u.Combatant] = u.Speeds
	}
	for _, s := range t.order {
		if sp, ok := latest[s.Combatant]; ok {
			p := n.profiles[s.Combatant]
			p.Speeds = sp
			n.profiles[s.Combatant] = p
			events = append(events, MovementSpeedsUpdated{combatant: s.Combatant, speeds: sp})
		}
	}
	n.active = (t.active + 1) % len(t.order)
	if n.active == 0 {
		n.round = t.round + 1
		events = append(events, TrackerRoundStarted{round: n.round})
	}
	next := n.order[n.active].Combatant
	events = append(events, TrackerTurnStarted{combatant: next, round: n.round})
	events = append(events, n.startTurn(next)...)
	return stepApplied(n, events...)
}

// UpdateMovementSpeeds replaces id's profile speeds at any time on any turn (SRD 5.2.1,
// "Changes to Your Speeds", p. 188). Moved feet are not refunded and no resource changes;
// the new speeds persist to every later turn until replaced.
func (t TurnTracker) UpdateMovementSpeeds(id CombatantID, speeds MovementSpeeds) (TrackerStep, error) {
	p, ok := t.profiles[id]
	if !ok || id.IsZero() {
		return stepRejected(TurnUnknownCombatantRejection{id})
	}
	n := t.clone()
	p.Speeds = speeds
	n.profiles[id] = p
	return stepApplied(n, MovementSpeedsUpdated{combatant: id, speeds: speeds})
}

// grantedOf is the action a choice carries out: what its effect under FR-051 to FR-054,
// FR-112 and FR-113 is.
func grantedOf(c ActionChoice) GrantedAction {
	switch v := c.(type) {
	case DashChoice:
		return GrantsDash(v)
	case SimpleActionChoice:
		return GrantsSimpleAction(v)
	case FeatureActionChoice:
		return v.Granted
	}
	return NoStandardAction{}
}

// dashRejection refuses a Dash by a speed the combatant lacks.
func (t TurnTracker) dashRejection(id CombatantID, g GrantedAction) TurnRejection {
	if d, ok := g.(GrantsDash); ok {
		if _, has := t.profiles[id].Speeds.Of(d.Speed); !has {
			return TurnSpeedUnavailableRejection(d)
		}
	}
	return nil
}

// applyGranted applies the effect of a granted action to the active combatant: Dash adds
// the speed to the Dash extra, Disengage marks Disengaged, Dodge records dodging.
func (t *TurnTracker) applyGranted(id CombatantID, g GrantedAction) {
	switch v := g.(type) {
	case GrantsDash:
		feet, _ := t.profiles[id].Speeds.Of(v.Speed)
		t.res.dash = satAdd(t.res.dash, feet)
	case GrantsSimpleAction:
		switch v.Action {
		case SimpleStandardActionDisengage:
			t.res.disengage = Disengaged
		case SimpleStandardActionDodge:
			t.dodging[id] = true
		case SimpleStandardActionAttack, SimpleStandardActionHelp, SimpleStandardActionHide, SimpleStandardActionInfluence,
			SimpleStandardActionMagic, SimpleStandardActionSearch, SimpleStandardActionStudy, SimpleStandardActionUtilize:
			// resolved elsewhere
		}
	}
}

// TakeAction spends the active combatant's action (or its additional action first, when
// the scope permits the choice) on choice. Refusals, in order: unknown id, incapacitated,
// not active, action already taken (also a second held Ready), speed unavailable. It
// resolves no effect beyond Dash, Disengage and Dodge. It returns an error matching
// srderr.ErrInvalidInput for a nil or undefined choice or availability.
func (t TurnTracker) TakeAction(id CombatantID, av ConditionActionAvailability, choice ActionChoice) (TrackerStep, error) {
	if !validAvailability(av) || !validActionChoice(choice) {
		return nil, invalidInput("action needs an availability and a valid choice")
	}
	if r := t.gate(id, av, true); r != nil {
		return stepRejected(r)
	}
	std, named := choice.Standard()
	useExtra := !t.extra.grant.IsZero() && t.extra.used == ResourceAvailable && t.extra.grant.scope.permits(std, named)
	_, holdsReady := t.readied[id]
	_, isReady := choice.(ReadyChoice)
	if (!useExtra && t.res.action == ResourceSpent) || (isReady && holdsReady) {
		return stepRejected(TurnActionAlreadyTakenRejection{})
	}
	g := grantedOf(choice)
	if r := t.dashRejection(id, g); r != nil {
		return stepRejected(r)
	}
	n := t.clone()
	if useExtra {
		n.extra.used = ResourceSpent
	} else {
		n.res.action = ResourceSpent
	}
	n.applyGranted(id, g)
	if r, ok := choice.(ReadyChoice); ok {
		n.readied[id] = r.Action
	}
	return stepApplied(n, ActionTaken{combatant: id, choice: choice})
}

// TakeBonusAction spends the active combatant's Bonus Action if a rule grants one. The
// action stays available. Refusals, in order: unknown id, incapacitated, not active, no
// Bonus Action granted, Bonus Action already taken, speed unavailable. It returns an
// error matching srderr.ErrInvalidInput for a nil entitlement, an invalid grant or an
// availability that is not defined.
func (t TurnTracker) TakeBonusAction(id CombatantID, av ConditionActionAvailability, e BonusActionEntitlement) (TrackerStep, error) {
	var grant BonusActionGrant
	switch v := e.(type) {
	case EntitledToBonusAction:
		if v.Grant.IsZero() {
			return nil, invalidInput("entitlement needs a grant built by NewBonusActionGrant")
		}
		grant = v.Grant
	case NotEntitledToBonusAction:
	default:
		return nil, invalidInput("Bonus Action needs an entitlement")
	}
	if !validAvailability(av) {
		return nil, invalidInput("availability is not defined")
	}
	if r := t.gate(id, av, true); r != nil {
		return stepRejected(r)
	}
	switch {
	case grant.IsZero():
		return stepRejected(TurnNoBonusActionGrantedRejection{})
	case t.res.bonus == ResourceSpent:
		return stepRejected(TurnBonusActionAlreadyTakenRejection{})
	}
	if r := t.dashRejection(id, grant.granted); r != nil {
		return stepRejected(r)
	}
	n := t.clone()
	n.res.bonus = ResourceSpent
	n.applyGranted(id, grant.granted)
	return stepApplied(n, BonusActionUsed{combatant: id, source: grant.source, granted: grant.granted})
}

// TakeReaction spends id's Reaction on choice, on any combatant's turn. Refusals, in
// order: unknown id (reactor, then provoker), incapacitated, Reaction already taken, then
// for an Opportunity Attack provoker disengaged and provoker not seen, for a readied
// action no matching readied action and, for a held move, the move's reasons judged now
// with nothing moved and no Dash extra. It changes nothing of the active turn. A
// Feature or Spell Reaction is only recorded. It returns an error matching
// srderr.ErrInvalidInput for a nil or undefined choice or availability.
func (t TurnTracker) TakeReaction(id CombatantID, av ConditionActionAvailability, choice ReactionChoice) (TrackerStep, error) {
	if !validAvailability(av) || !validReactionChoice(choice) {
		return nil, invalidInput("reaction needs an availability and a valid choice")
	}
	if oa, ok := choice.(OpportunityAttackReaction); ok {
		if _, known := t.profiles[oa.Provoker]; !known && t.profiles[id].ID == id && !id.IsZero() {
			return stepRejected(TurnUnknownCombatantRejection{oa.Provoker})
		}
	}
	if r := t.gate(id, av, false); r != nil {
		return stepRejected(r)
	}
	if t.spent[id] {
		return stepRejected(TurnReactionAlreadyTakenRejection{})
	}
	n := t.clone()
	var record ReactionRecord
	switch v := choice.(type) {
	case OpportunityAttackReaction:
		switch {
		case t.order[t.active].Combatant == v.Provoker && t.res.disengage == Disengaged:
			return stepRejected(TurnProvokerDisengagedRejection{})
		case v.Visibility == VisibilityUnseen:
			return stepRejected(TurnProvokerNotSeenRejection{})
		}
		record = OpportunityAttackRecord{id, v.Provoker, v.Means, AttackTimingRightBeforeLeavingReach}
	case ReadiedActionReaction:
		a, held := t.readied[id]
		if !held || a.trigger != v.Trigger {
			return stepRejected(TurnNoMatchingReadiedActionRejection(v))
		}
		outcome := ReadiedOutcome(nil)
		switch r := a.response.(type) {
		case ReadiedActionResponse:
			outcome = ReadiedChoiceOutcome(r)
		case ReadiedMoveResponse:
			p := t.profiles[id]
			j := judgeMove(p.Speeds, p.Size, 0, 0, r.segments)
			if j.rejection != nil {
				return stepRejected(j.rejection)
			}
			outcome = ReadiedMoveOutcome{j.charges, j.total}
		}
		delete(n.readied, id)
		record = ReadiedActionRecord{a.trigger, outcome, a.Held()}
	case FeatureReaction:
		record = FeatureReactionRecord(v)
	case SpellReaction:
		record = SpellReactionRecord(v)
	}
	n.spent[id] = true
	return stepApplied(n, ReactionTaken{combatant: id, record: record})
}

func validReactionChoice(c ReactionChoice) bool {
	switch v := c.(type) {
	case OpportunityAttackReaction:
		return !v.Provoker.IsZero() && v.Means.Valid() && v.Visibility.Valid()
	case ReadiedActionReaction:
		_, err := ParseReadyTriggerID(string(v.Trigger))
		return err == nil
	case FeatureReaction:
		return v.Feature.Valid() && v.Granted != nil && v.Granted.valid()
	case SpellReaction:
		return v.Spell.Valid()
	}
	return false
}

// Move moves the active combatant along segments, judged segment by segment against the
// speeds, the Dash extra and the feet already moved (initiative-and-actions.md FR-115).
// It is atomic. Refusals, in order: unknown id, not active, speed unavailable, blocked by
// a creature, the first unaffordable segment, ends in an occupied space. It returns an
// error matching srderr.ErrInvalidInput for no segment or a zero segment, and a
// LimitExceededError for more than 256 segments (plan P10).
func (t TurnTracker) Move(id CombatantID, segments []MoveSegment) (TrackerStep, error) {
	if err := overCap("move segments", len(segments), maxInputList); err != nil {
		return nil, err
	}
	if len(segments) == 0 || slices.ContainsFunc(segments, MoveSegment.IsZero) {
		return nil, invalidInput("a move needs segments")
	}
	if r := t.gate(id, ConditionActionAvailabilityAvailable, true); r != nil {
		return stepRejected(r)
	}
	p := t.profiles[id]
	j := judgeMove(p.Speeds, p.Size, t.res.moved, t.res.dash, segments)
	if j.rejection != nil {
		return stepRejected(j.rejection)
	}
	n := t.clone()
	n.res.moved = j.moved
	return stepApplied(n, MovementSpent{combatant: id, charges: j.charges, total: j.total, moved: j.moved})
}

// InteractWithObject spends the active combatant's one free interaction with an object
// (SRD 5.2.1, "Your Turn", p. 13). Refusals, in order: unknown id, not active, needs the
// Utilize action.
func (t TurnTracker) InteractWithObject(id CombatantID) (TrackerStep, error) {
	if r := t.gate(id, ConditionActionAvailabilityAvailable, true); r != nil {
		return stepRejected(r)
	}
	if t.res.interaction == ResourceSpent {
		return stepRejected(TurnNeedsUtilizeActionRejection{})
	}
	n := t.clone()
	n.res.interaction = ResourceSpent
	return stepApplied(n, ObjectInteracted{combatant: id})
}

// ChangePosture drops the active combatant Prone or stands it up (SRD 5.2.1, "Dropping
// Prone", p. 14; "Prone [Condition]", p. 186). Dropping needs a walking Speed above 0. A
// stand-up takes its cost and refusal from the supplied ProneMovement and the walking
// movement left (profile Speed plus Dash extra minus feet moved), and adds the cost to the
// feet moved. Refusals, in order: unknown id, not active, speed is zero, insufficient
// movement. It returns an error matching srderr.ErrInvalidInput for a nil or undefined
// change.
func (t TurnTracker) ChangePosture(id CombatantID, change PostureChange) (TrackerStep, error) {
	var cost int
	stand, crawlOnly := false, false
	switch v := change.(type) {
	case DropProneChange:
	case StandUpChange:
		switch m := v.Movement.(type) {
		case CrawlOnly:
			crawlOnly = true
		case CrawlOrRightItself:
			if m.CostFeet < 0 {
				return nil, invalidInput("negative stand-up cost")
			}
			cost = m.CostFeet
		default:
			return nil, invalidInput("stand up needs a ProneMovement")
		}
		stand = true
	default:
		return nil, invalidInput("posture needs a change")
	}
	if r := t.gate(id, ConditionActionAvailabilityAvailable, true); r != nil {
		return stepRejected(r)
	}
	walk := t.profiles[id].Speeds.Walk()
	if !stand {
		if walk == 0 {
			return stepRejected(TurnSpeedIsZeroRejection{})
		}
		return stepApplied(t.clone(), DroppedProne{combatant: id})
	}
	if crawlOnly {
		return stepRejected(TurnSpeedIsZeroRejection{})
	}
	if left := satAdd(satAdd(walk, t.res.dash), -t.res.moved); left < cost {
		return stepRejected(TurnInsufficientMovementRejection{ShortfallFeet: cost - left})
	}
	n := t.clone()
	n.res.moved = satAdd(n.res.moved, cost)
	return stepApplied(n, StoodUp{combatant: id, cost: cost, moved: n.res.moved})
}

// GrantAdditionalAction records one additional action for the active combatant's
// current turn; TakeAction spends it first when its scope permits the choice. Refusals,
// in order: unknown id, not active, additional action already granted. It returns an error
// matching srderr.ErrInvalidInput for the zero grant.
func (t TurnTracker) GrantAdditionalAction(id CombatantID, grant AdditionalActionGrant) (TrackerStep, error) {
	if grant.IsZero() {
		return nil, invalidInput("additional action needs a grant built by NewAdditionalActionGrant")
	}
	if r := t.gate(id, ConditionActionAvailabilityAvailable, true); r != nil {
		return stepRejected(r)
	}
	if !t.extra.grant.IsZero() {
		return stepRejected(TurnAdditionalActionAlreadyGrantedRejection{})
	}
	n := t.clone()
	n.extra = extraAction{grant, ResourceAvailable}
	return stepApplied(n, AdditionalActionGranted{combatant: id, grant: grant})
}

// RecordFeatUse records a use of a feat limited by OncePerTurn (until the current turn
// ends) or OnceUntilStartOfYourNextTurn (until id's next turn starts). It checks neither
// the active combatant nor availability. Refusals, in order: unknown id, feat use limited. It
// returns an error matching srderr.ErrInvalidInput for an invalid feat id or any other
// limit (those are not turn state).
func (t TurnTracker) RecordFeatUse(id CombatantID, feat content.FeatID, limit character.FeatUsageLimit) (TrackerStep, error) {
	window := 0
	switch limit.(type) {
	case character.OncePerTurn:
		window = featWindowTurn
	case character.OnceUntilStartOfYourNextTurn:
		window = featWindowNextStart
	}
	if window == 0 || !feat.Valid() {
		return nil, invalidInput("a feat use needs a feat and a once per turn or once until start of next turn limit")
	}
	if t.profiles[id].ID != id || id.IsZero() {
		return stepRejected(TurnUnknownCombatantRejection{id})
	}
	key := featUseKey{id, feat, window}
	if t.feats[key] {
		return stepRejected(TurnFeatUseLimitedRejection{feat})
	}
	n := t.clone()
	n.feats[key] = true
	return stepApplied(n, FeatUseRecorded{combatant: id, feat: feat, limit: limit})
}
