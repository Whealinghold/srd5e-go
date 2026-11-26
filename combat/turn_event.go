package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// TurnEvent is the closed family of events the tracker operations produce. Each is an
// immutable past-tense value with a constant snake_case Kind and a Combatants list.
// Events that carry a slice (MovementSpent, and an ActionTaken or ReactionTaken that
// records a readied move) compare with their Equal-like accessors, not with ==.
// Each embeds turnEventBase. Variants: TrackerRoundStarted, TrackerTurnStarted, TrackerTurnEnded,
// AdditionalActionGranted, FeatUseRecorded, InitiativeSwapped, ActionTaken,
// BonusActionUsed, ReactionTaken, MovementSpent, ObjectInteracted, DroppedProne,
// StoodUp, FellProneInSharedSpace, ReadiedActionExpired, MovementSpeedsUpdated.
type TurnEvent interface {
	Event
	turnEvent()
}

// turnEventBase puts an event in the Event and TurnEvent families.
type turnEventBase struct{}

func (turnEventBase) event()     {}
func (turnEventBase) turnEvent() {}

// one is the Combatants list of an event about one creature.
func one(id CombatantID) []CombatantID { return []CombatantID{id} }

// TrackerRoundStarted: a round began. Kind tracker_round_started; no combatant.
type TrackerRoundStarted struct {
	turnEventBase
	round int
}

// Kind returns "tracker_round_started".
func (TrackerRoundStarted) Kind() string { return "tracker_round_started" }

// Combatants returns an empty list.
func (TrackerRoundStarted) Combatants() []CombatantID { return []CombatantID{} }

// Round returns the round number, 1 or more.
func (e TrackerRoundStarted) Round() int { return e.round }

// TrackerTurnStarted: a combatant's turn began. Kind tracker_turn_started.
type TrackerTurnStarted struct {
	turnEventBase
	combatant CombatantID
	round     int
}

// Kind returns "tracker_turn_started".
func (TrackerTurnStarted) Kind() string { return "tracker_turn_started" }

// Combatants returns the combatant whose turn began.
func (e TrackerTurnStarted) Combatants() []CombatantID { return one(e.combatant) }

// Round returns the round number.
func (e TrackerTurnStarted) Round() int { return e.round }

// TrackerTurnEnded: a combatant's turn ended. Kind tracker_turn_ended.
type TrackerTurnEnded struct {
	turnEventBase
	combatant CombatantID
	round     int
}

// Kind returns "tracker_turn_ended".
func (TrackerTurnEnded) Kind() string { return "tracker_turn_ended" }

// Combatants returns the combatant whose turn ended.
func (e TrackerTurnEnded) Combatants() []CombatantID { return one(e.combatant) }

// Round returns the round number.
func (e TrackerTurnEnded) Round() int { return e.round }

// AdditionalActionGranted: the active combatant gained an additional action for the
// turn. Kind additional_action_granted.
type AdditionalActionGranted struct {
	turnEventBase
	combatant CombatantID
	grant     AdditionalActionGrant
}

// Kind returns "additional_action_granted".
func (AdditionalActionGranted) Kind() string { return "additional_action_granted" }

// Combatants returns the combatant.
func (e AdditionalActionGranted) Combatants() []CombatantID { return one(e.combatant) }

// Grant returns the grant.
func (e AdditionalActionGranted) Grant() AdditionalActionGrant { return e.grant }

// FeatUseRecorded: a turn-limited feat use was recorded. Kind feat_use_recorded.
type FeatUseRecorded struct {
	turnEventBase
	combatant CombatantID
	feat      content.FeatID
	limit     character.FeatUsageLimit
}

// Kind returns "feat_use_recorded".
func (FeatUseRecorded) Kind() string { return "feat_use_recorded" }

// Combatants returns the combatant.
func (e FeatUseRecorded) Combatants() []CombatantID { return one(e.combatant) }

// Feat returns the feat.
func (e FeatUseRecorded) Feat() content.FeatID { return e.feat }

// Limit returns the limit, OncePerTurn or OnceUntilStartOfYourNextTurn.
func (e FeatUseRecorded) Limit() character.FeatUsageLimit { return e.limit }

// InitiativeSwapped: two combatants exchanged Initiative counts. Kind
// initiative_swapped; Combatants is swapper then ally.
type InitiativeSwapped struct {
	turnEventBase
	first, second           CombatantID
	firstCount, secondCount InitiativeCount
}

// Kind returns "initiative_swapped".
func (InitiativeSwapped) Kind() string { return "initiative_swapped" }

// Combatants returns the swapper then the ally.
func (e InitiativeSwapped) Combatants() []CombatantID { return []CombatantID{e.first, e.second} }

// FirstCount returns the swapper's count after the swap.
func (e InitiativeSwapped) FirstCount() InitiativeCount { return e.firstCount }

// SecondCount returns the ally's count after the swap.
func (e InitiativeSwapped) SecondCount() InitiativeCount { return e.secondCount }

// ActionTaken: the combatant took the choice as its action. Kind action_taken.
type ActionTaken struct {
	turnEventBase
	combatant CombatantID
	choice    ActionChoice
}

// Kind returns "action_taken".
func (ActionTaken) Kind() string { return "action_taken" }

// Combatants returns the combatant.
func (e ActionTaken) Combatants() []CombatantID { return one(e.combatant) }

// Choice returns the choice as taken.
func (e ActionTaken) Choice() ActionChoice { return e.choice }

// BonusActionUsed: the combatant used a Bonus Action. Kind bonus_action_used.
type BonusActionUsed struct {
	turnEventBase
	combatant CombatantID
	source    BonusActionSource
	granted   GrantedAction
}

// Kind returns "bonus_action_used".
func (BonusActionUsed) Kind() string { return "bonus_action_used" }

// Combatants returns the combatant.
func (e BonusActionUsed) Combatants() []CombatantID { return one(e.combatant) }

// Source returns where the Bonus Action came from.
func (e BonusActionUsed) Source() BonusActionSource { return e.source }

// Granted returns the standard action the grant named.
func (e BonusActionUsed) Granted() GrantedAction { return e.granted }

// ReactionRecord is what a Reaction was spent on, a closed set. Variants:
// OpportunityAttackRecord, ReadiedActionRecord, FeatureReactionRecord, SpellReactionRecord.
type ReactionRecord interface{ isReactionRecord() }

// OpportunityAttackRecord: an Opportunity Attack, not resolved here.
type OpportunityAttackRecord struct {
	Observer, Provoker CombatantID
	Means              DepartureMeans
	Timing             AttackTiming
}

// ReadiedActionRecord: a readied action fired.
type ReadiedActionRecord struct {
	Trigger ReadyTriggerID
	Outcome ReadiedOutcome
	Held    HeldSpellInput
}

// FeatureReactionRecord: a feature's Reaction, recorded only.
type FeatureReactionRecord struct {
	Feature content.FeatureID
	Granted GrantedAction
}

// SpellReactionRecord: a spell cast with the Reaction, recorded only.
type SpellReactionRecord struct{ Spell content.SpellID }

func (OpportunityAttackRecord) isReactionRecord() {}
func (ReadiedActionRecord) isReactionRecord()     {}
func (FeatureReactionRecord) isReactionRecord()   {}
func (SpellReactionRecord) isReactionRecord()     {}

// ReadiedOutcome is what a fired readied action did, a closed set. Variants:
// ReadiedChoiceOutcome, ReadiedMoveOutcome.
type ReadiedOutcome interface{ isReadiedOutcome() }

// ReadiedChoiceOutcome: the readied choice, recorded only.
type ReadiedChoiceOutcome struct{ Choice ReadyableChoice }

// ReadiedMoveOutcome: the readied move, with its charges and total.
type ReadiedMoveOutcome struct {
	charges []MovementCharge
	total   int
}

func (ReadiedChoiceOutcome) isReadiedOutcome() {}
func (ReadiedMoveOutcome) isReadiedOutcome()   {}

// Charges returns a copy of the movement charged, one per segment.
func (o ReadiedMoveOutcome) Charges() []MovementCharge { return slices.Clone(o.charges) }

// TotalFeet returns the sum of the charges.
func (o ReadiedMoveOutcome) TotalFeet() int { return o.total }

// ReactionTaken: the combatant spent its Reaction. Kind reaction_taken; Combatants is
// observer then provoker for an Opportunity Attack, else the reactor.
type ReactionTaken struct {
	turnEventBase
	combatant CombatantID
	record    ReactionRecord
}

// Kind returns "reaction_taken".
func (ReactionTaken) Kind() string { return "reaction_taken" }

// Combatants returns observer then provoker for an Opportunity Attack, else the reactor.
func (e ReactionTaken) Combatants() []CombatantID {
	if r, ok := e.record.(OpportunityAttackRecord); ok {
		return []CombatantID{r.Observer, r.Provoker}
	}
	return one(e.combatant)
}

// Record returns what the Reaction was spent on.
func (e ReactionTaken) Record() ReactionRecord { return e.record }

// MovementSpent: an accepted Move charged its movement. Kind movement_spent.
type MovementSpent struct {
	turnEventBase
	combatant CombatantID
	charges   []MovementCharge
	total     int
	moved     int
}

// Kind returns "movement_spent".
func (MovementSpent) Kind() string { return "movement_spent" }

// Combatants returns the mover.
func (e MovementSpent) Combatants() []CombatantID { return one(e.combatant) }

// Charges returns a copy of the charges, one per segment in order.
func (e MovementSpent) Charges() []MovementCharge { return slices.Clone(e.charges) }

// TotalFeet returns the sum of the charges.
func (e MovementSpent) TotalFeet() int { return e.total }

// MovedFeet returns the feet moved this turn after the move.
func (e MovementSpent) MovedFeet() int { return e.moved }

// ObjectInteracted: the combatant used its free object interaction. Kind object_interacted.
type ObjectInteracted struct {
	turnEventBase
	combatant CombatantID
}

// Kind returns "object_interacted".
func (ObjectInteracted) Kind() string { return "object_interacted" }

// Combatants returns the combatant.
func (e ObjectInteracted) Combatants() []CombatantID { return one(e.combatant) }

// DroppedProne: the combatant dropped Prone. Kind dropped_prone; applies no condition.
type DroppedProne struct {
	turnEventBase
	combatant CombatantID
}

// Kind returns "dropped_prone".
func (DroppedProne) Kind() string { return "dropped_prone" }

// Combatants returns the combatant.
func (e DroppedProne) Combatants() []CombatantID { return one(e.combatant) }

// StoodUp: the combatant stood up at a cost. Kind stood_up; removes no condition.
type StoodUp struct {
	turnEventBase
	combatant CombatantID
	cost      int
	moved     int
}

// Kind returns "stood_up".
func (StoodUp) Kind() string { return "stood_up" }

// Combatants returns the combatant.
func (e StoodUp) Combatants() []CombatantID { return one(e.combatant) }

// CostFeet returns the movement the stand-up cost.
func (e StoodUp) CostFeet() int { return e.cost }

// MovedFeet returns the feet moved this turn after standing up.
func (e StoodUp) MovedFeet() int { return e.moved }

// FellProneInSharedSpace: the combatant ended its turn in a space it shares and falls
// Prone. Kind fell_prone_in_shared_space; applies no condition.
type FellProneInSharedSpace struct {
	turnEventBase
	combatant CombatantID
}

// Kind returns "fell_prone_in_shared_space".
func (FellProneInSharedSpace) Kind() string { return "fell_prone_in_shared_space" }

// Combatants returns the combatant.
func (e FellProneInSharedSpace) Combatants() []CombatantID { return one(e.combatant) }

// ReadiedActionExpired: an unused readied action was discarded at the start of its
// owner's turn. Kind readied_action_expired.
type ReadiedActionExpired struct {
	turnEventBase
	combatant CombatantID
	trigger   ReadyTriggerID
	held      HeldSpellInput
}

// Kind returns "readied_action_expired".
func (ReadiedActionExpired) Kind() string { return "readied_action_expired" }

// Combatants returns the owner.
func (e ReadiedActionExpired) Combatants() []CombatantID { return one(e.combatant) }

// Trigger returns the trigger of the discarded action.
func (e ReadiedActionExpired) Trigger() ReadyTriggerID { return e.trigger }

// Held returns the held spell input.
func (e ReadiedActionExpired) Held() HeldSpellInput { return e.held }

// MovementSpeedsUpdated: a combatant's profile speeds were replaced. Kind
// movement_speeds_updated.
type MovementSpeedsUpdated struct {
	turnEventBase
	combatant CombatantID
	speeds    MovementSpeeds
}

// Kind returns "movement_speeds_updated".
func (MovementSpeedsUpdated) Kind() string { return "movement_speeds_updated" }

// Combatants returns the combatant.
func (e MovementSpeedsUpdated) Combatants() []CombatantID { return one(e.combatant) }

// Speeds returns the new speeds.
func (e MovementSpeedsUpdated) Speeds() MovementSpeeds { return e.speeds }
