package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// StandardAction is one of the twelve actions of the Actions table (SRD 5.2.1,
// pp. 9-10). The zero StandardAction is invalid.
type StandardAction int

// The twelve standard actions.
const (
	StandardActionAttack StandardAction = iota + 1
	StandardActionDash
	StandardActionDisengage
	StandardActionDodge
	StandardActionHelp
	StandardActionHide
	StandardActionInfluence
	StandardActionMagic
	StandardActionReady
	StandardActionSearch
	StandardActionStudy
	StandardActionUtilize
)

func standardActionNames() []string {
	return []string{"attack", "dash", "disengage", "dodge", "help", "hide", "influence", "magic", "ready", "search", "study", "utilize"}
}

// Valid reports whether a is one of the twelve actions.
func (a StandardAction) Valid() bool { return a >= StandardActionAttack && a <= StandardActionUtilize }

// String returns the lower-case action name or "unknown".
func (a StandardAction) String() string { return enumName(standardActionNames(), int(a)) }

// SimpleStandardAction is one of the ten standard actions other than Dash and
// Ready, which need a speed, or a trigger and a response. The zero value is invalid.
type SimpleStandardAction int

// The ten simple standard actions.
const (
	SimpleStandardActionAttack SimpleStandardAction = iota + 1
	SimpleStandardActionDisengage
	SimpleStandardActionDodge
	SimpleStandardActionHelp
	SimpleStandardActionHide
	SimpleStandardActionInfluence
	SimpleStandardActionMagic
	SimpleStandardActionSearch
	SimpleStandardActionStudy
	SimpleStandardActionUtilize
)

// Valid reports whether a is one of the ten actions.
func (a SimpleStandardAction) Valid() bool {
	return a >= SimpleStandardActionAttack && a <= SimpleStandardActionUtilize
}

// Standard returns the StandardAction of the same name, or the zero StandardAction
// for a value that is not one of the ten.
func (a SimpleStandardAction) Standard() StandardAction {
	if !a.Valid() {
		return 0
	}
	// The ten sit in the order of the twelve with Dash and Ready removed.
	s := StandardAction(a)
	if s >= StandardActionDash {
		s++ // skips Dash
	}
	if s >= StandardActionReady {
		s++ // skips Ready
	}
	return s
}

// String returns the lower-case action name or "unknown".
func (a SimpleStandardAction) String() string { return a.Standard().String() }

// GrantedAction is the standard action a feature grants, a closed set. Variants:
// NoStandardAction, GrantsDash, GrantsSimpleAction. There is no Ready variant.
type GrantedAction interface {
	isGrantedAction()
	// Standard returns the StandardAction the grant names and whether it names one.
	Standard() (StandardAction, bool)
	valid() bool
}

// NoStandardAction: the grant names no standard action.
type NoStandardAction struct{}

// GrantsDash: the grant is a Dash by the named speed.
type GrantsDash struct{ Speed SpeedName }

// GrantsSimpleAction: the grant is one of the ten simple standard actions.
type GrantsSimpleAction struct{ Action SimpleStandardAction }

func (NoStandardAction) isGrantedAction()   {}
func (GrantsDash) isGrantedAction()         {}
func (GrantsSimpleAction) isGrantedAction() {}

// Standard returns no action.
func (NoStandardAction) Standard() (StandardAction, bool) { return 0, false }

// Standard returns Dash.
func (GrantsDash) Standard() (StandardAction, bool) { return StandardActionDash, true }

// Standard returns the action named.
func (g GrantsSimpleAction) Standard() (StandardAction, bool) { return g.Action.Standard(), true }

func (NoStandardAction) valid() bool     { return true }
func (g GrantsDash) valid() bool         { return g.Speed.Valid() }
func (g GrantsSimpleAction) valid() bool { return g.Action.Valid() }

// ActionChoice is what a combatant does with its action, a closed set. Each
// variant reports the StandardAction it records. Variants: ReadyChoice,
// DashChoice, SimpleActionChoice, FeatureActionChoice, ImprovisedActionChoice.
type ActionChoice interface {
	isActionChoice()
	// Standard returns the StandardAction the choice records and whether it
	// records one (a feature action records its GrantedAction's; an improvised
	// action none).
	Standard() (StandardAction, bool)
}

// ReadyableChoice is an ActionChoice that can also be the response of a readied
// action, so a Ready cannot be readied. Variants: DashChoice, SimpleActionChoice,
// FeatureActionChoice, ImprovisedActionChoice.
type ReadyableChoice interface {
	ActionChoice
	isReadyableChoice()
}

// ReadyChoice: take the Ready action with a trigger and a response.
type ReadyChoice struct{ Action ReadiedAction }

// DashChoice: take Dash, adding the named speed to the turn's movement.
type DashChoice struct{ Speed SpeedName }

// SimpleActionChoice: take one of the ten simple standard actions.
type SimpleActionChoice struct{ Action SimpleStandardAction }

// FeatureActionChoice: an action a feature provides, optionally naming a standard action.
type FeatureActionChoice struct {
	Feature content.FeatureID
	Granted GrantedAction
}

// ImprovisedActionChoice: an action not covered by the standard ones (SRD 5.2.1, "Actions", p. 10).
type ImprovisedActionChoice struct{}

func (ReadyChoice) isActionChoice()            {}
func (DashChoice) isActionChoice()             {}
func (SimpleActionChoice) isActionChoice()     {}
func (FeatureActionChoice) isActionChoice()    {}
func (ImprovisedActionChoice) isActionChoice() {}

func (DashChoice) isReadyableChoice()             {}
func (SimpleActionChoice) isReadyableChoice()     {}
func (FeatureActionChoice) isReadyableChoice()    {}
func (ImprovisedActionChoice) isReadyableChoice() {}

// Standard returns Ready.
func (ReadyChoice) Standard() (StandardAction, bool) { return StandardActionReady, true }

// Standard returns Dash.
func (DashChoice) Standard() (StandardAction, bool) { return StandardActionDash, true }

// Standard returns the simple action's own.
func (c SimpleActionChoice) Standard() (StandardAction, bool) { return c.Action.Standard(), true }

// Standard returns the granted action's, if any.
func (c FeatureActionChoice) Standard() (StandardAction, bool) {
	if c.Granted == nil {
		return 0, false
	}
	return c.Granted.Standard()
}

// Standard returns no action.
func (ImprovisedActionChoice) Standard() (StandardAction, bool) { return 0, false }

// valid reports whether the choice is well formed: ids valid, speeds and actions defined.
func validActionChoice(c ActionChoice) bool {
	switch v := c.(type) {
	case ReadyChoice:
		return !v.Action.IsZero()
	case DashChoice:
		return v.Speed.Valid()
	case SimpleActionChoice:
		return v.Action.Valid()
	case FeatureActionChoice:
		return v.Feature.Valid() && v.Granted != nil && v.Granted.valid()
	case ImprovisedActionChoice:
		return true
	}
	return false
}

// PermitsMountAction reports whether a rider may take the action while on a mount
// with the given control (SRD 5.2.1, "Controlling a Mount", p. 16): a controlled
// mount permits exactly Dash, Disengage and Dodge as simple choices, an independent
// one everything. A control that is not valid or a nil choice is not permitted.
// Bonus Actions and Reactions are not restricted and are not asked about here.
func PermitsMountAction(control MountControl, choice ActionChoice) bool {
	switch {
	case choice == nil:
		return false
	case control == MountControlIndependent:
		return true
	case control != MountControlControlled:
		return false
	}
	switch c := choice.(type) {
	case DashChoice:
		return true
	case SimpleActionChoice:
		return c.Action == SimpleStandardActionDisengage || c.Action == SimpleStandardActionDodge
	}
	return false
}

// AdditionalActionScope says which choices an additional action permits, a closed
// set. Variants: AnyActionBut, OnlyActions.
type AdditionalActionScope interface {
	isAdditionalActionScope()
	permits(a StandardAction, named bool) bool
	actionList() []StandardAction
}

// AnyActionBut permits every action except those listed (Action Surge: all but
// Magic). A choice that records no standard action is permitted. Build it with
// NewAnyActionBut; the zero value permits everything.
type AnyActionBut struct{ actions []StandardAction }

// OnlyActions permits only the listed actions (Haste). A choice that records no
// standard action is not permitted. Build it with NewOnlyActions.
type OnlyActions struct{ actions []StandardAction }

func (AnyActionBut) isAdditionalActionScope() {}
func (OnlyActions) isAdditionalActionScope()  {}

// sortedActions validates, sorts and deduplicates actions.
func sortedActions(actions []StandardAction) ([]StandardAction, error) {
	out := slices.Clone(actions)
	for _, a := range out {
		if !a.Valid() {
			return nil, invalidInput("scope names an action that is not a standard action")
		}
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

// NewAnyActionBut returns the scope that permits every action but those listed (an
// empty list permits all). It returns an error matching srderr.ErrInvalidInput for
// an action that is not a StandardAction. It never panics.
func NewAnyActionBut(actions ...StandardAction) (AnyActionBut, error) {
	list, err := sortedActions(actions)
	return AnyActionBut{list}, err
}

// NewOnlyActions returns the scope that permits only the listed actions. It returns
// an error matching srderr.ErrInvalidInput for an empty list or an action that is
// not a StandardAction. It never panics.
func NewOnlyActions(actions ...StandardAction) (OnlyActions, error) {
	if len(actions) == 0 {
		return OnlyActions{}, invalidInput("OnlyActions needs at least one action")
	}
	list, err := sortedActions(actions)
	return OnlyActions{list}, err
}

func (s AnyActionBut) permits(a StandardAction, named bool) bool {
	return !named || !slices.Contains(s.actions, a)
}
func (s OnlyActions) permits(a StandardAction, named bool) bool {
	return named && slices.Contains(s.actions, a)
}
func (s AnyActionBut) actionList() []StandardAction { return s.actions }
func (s OnlyActions) actionList() []StandardAction  { return s.actions }

// Actions returns a copy of the listed actions.
func (s AnyActionBut) Actions() []StandardAction { return slices.Clone(s.actions) }

// Actions returns a copy of the listed actions.
func (s OnlyActions) Actions() []StandardAction { return slices.Clone(s.actions) }

// AdditionalActionGrant is one additional action a feature gives for a turn (Action
// Surge, Haste). Build it with NewAdditionalActionGrant; the zero value is invalid.
type AdditionalActionGrant struct {
	feature content.FeatureID
	scope   AdditionalActionScope
}

// NewAdditionalActionGrant returns the grant. It returns an error matching
// srderr.ErrInvalidInput for an invalid feature id or a nil scope. It never panics.
func NewAdditionalActionGrant(feature content.FeatureID, scope AdditionalActionScope) (AdditionalActionGrant, error) {
	if !feature.Valid() || scope == nil {
		return AdditionalActionGrant{}, invalidInput("additional action needs a feature and a scope")
	}
	return AdditionalActionGrant{feature, scope}, nil
}

// Feature returns the feature that grants the action.
func (g AdditionalActionGrant) Feature() content.FeatureID { return g.feature }

// Scope returns the scope.
func (g AdditionalActionGrant) Scope() AdditionalActionScope { return g.scope }

// IsZero reports whether g is the invalid zero grant.
func (g AdditionalActionGrant) IsZero() bool { return g.scope == nil }

// Equal reports whether g and o grant the same feature and scope.
func (g AdditionalActionGrant) Equal(o AdditionalActionGrant) bool {
	if g.scope == nil || o.scope == nil {
		return g.scope == nil && o.scope == nil
	}
	_, ga := g.scope.(AnyActionBut)
	_, oa := o.scope.(AnyActionBut)
	return g.feature == o.feature && ga == oa && slices.Equal(g.scope.actionList(), o.scope.actionList())
}

// BonusActionSource is where a Bonus Action comes from, a closed set. Variants:
// BonusActionFromFeature, BonusActionFromItem.
type BonusActionSource interface{ isBonusActionSource() }

// BonusActionFromFeature: a class feature or other feature grants the Bonus Action.
type BonusActionFromFeature struct{ Feature content.FeatureID }

// BonusActionFromItem: an item whose text says "As a Bonus Action" grants it.
type BonusActionFromItem struct{ Item content.ItemID }

func (BonusActionFromFeature) isBonusActionSource() {}
func (BonusActionFromItem) isBonusActionSource()    {}

// BonusActionGrant is a rule that gives a combatant a Bonus Action: its source and the
// standard action it names. Build it with NewBonusActionGrant. Comparable with ==.
type BonusActionGrant struct {
	source  BonusActionSource
	granted GrantedAction
}

// NewBonusActionGrant returns the grant. It returns an error matching
// srderr.ErrInvalidInput for a nil or invalid source or granted action. It never panics.
func NewBonusActionGrant(source BonusActionSource, granted GrantedAction) (BonusActionGrant, error) {
	switch s := source.(type) {
	case BonusActionFromFeature:
		if !s.Feature.Valid() {
			return BonusActionGrant{}, invalidInput("Bonus Action feature id")
		}
	case BonusActionFromItem:
		if !s.Item.Valid() {
			return BonusActionGrant{}, invalidInput("Bonus Action item id")
		}
	default:
		return BonusActionGrant{}, invalidInput("Bonus Action needs a source")
	}
	if granted == nil || !granted.valid() {
		return BonusActionGrant{}, invalidInput("Bonus Action needs a valid granted action")
	}
	return BonusActionGrant{source, granted}, nil
}

// Source returns where the Bonus Action comes from.
func (g BonusActionGrant) Source() BonusActionSource { return g.source }

// Granted returns the standard action the grant names.
func (g BonusActionGrant) Granted() GrantedAction { return g.granted }

// IsZero reports whether g is the invalid zero grant.
func (g BonusActionGrant) IsZero() bool { return g.source == nil }

// BonusActionEntitlement says whether a rule grants a Bonus Action now, a closed set.
// Variants: EntitledToBonusAction, NotEntitledToBonusAction.
type BonusActionEntitlement interface{ isBonusActionEntitlement() }

// EntitledToBonusAction: a rule grants the Bonus Action.
type EntitledToBonusAction struct{ Grant BonusActionGrant }

// NotEntitledToBonusAction: no rule grants a Bonus Action.
type NotEntitledToBonusAction struct{}

func (EntitledToBonusAction) isBonusActionEntitlement()    {}
func (NotEntitledToBonusAction) isBonusActionEntitlement() {}

// DepartureMeans is how a provoker left an observer's reach (SRD 5.2.1,
// "Opportunity Attacks", p. 185). The zero value is invalid.
type DepartureMeans int

// The four means.
const (
	DepartureByAction DepartureMeans = iota + 1
	DepartureByBonusAction
	DepartureByReaction
	DepartureBySpeed
)

// Valid reports whether m is one of the four means.
func (m DepartureMeans) Valid() bool { return m >= DepartureByAction && m <= DepartureBySpeed }

// String returns "action", "bonus_action", "reaction", "speed" or "unknown".
func (m DepartureMeans) String() string {
	return enumName([]string{"action", "bonus_action", "reaction", "speed"}, int(m))
}

// Visibility says whether the observer can see the provoker. The zero value is invalid.
type Visibility int

// The two values.
const (
	VisibilitySeen Visibility = iota + 1
	VisibilityUnseen
)

// Valid reports whether v is one of the two.
func (v Visibility) Valid() bool { return v == VisibilitySeen || v == VisibilityUnseen }

// String returns "seen", "unseen" or "unknown".
func (v Visibility) String() string { return enumName([]string{"seen", "unseen"}, int(v)) }

// ReactionChoice is what a combatant spends its Reaction on, a closed set. Variants:
// OpportunityAttackReaction, ReadiedActionReaction, FeatureReaction, SpellReaction.
type ReactionChoice interface{ isReactionChoice() }

// OpportunityAttackReaction: an Opportunity Attack against a provoker that left reach.
type OpportunityAttackReaction struct {
	Provoker   CombatantID
	Means      DepartureMeans
	Visibility Visibility
}

// ReadiedActionReaction: the trigger of a readied action fired.
type ReadiedActionReaction struct{ Trigger ReadyTriggerID }

// FeatureReaction: a Reaction a feature provides; only recorded.
type FeatureReaction struct {
	Feature content.FeatureID
	Granted GrantedAction
}

// SpellReaction: a spell cast with the Reaction; only recorded.
type SpellReaction struct{ Spell content.SpellID }

func (OpportunityAttackReaction) isReactionChoice() {}
func (ReadiedActionReaction) isReactionChoice()     {}
func (FeatureReaction) isReactionChoice()           {}
func (SpellReaction) isReactionChoice()             {}

// AttackTiming says when an Opportunity Attack happens; its only value is
// AttackTimingRightBeforeLeavingReach.
type AttackTiming int

// AttackTimingRightBeforeLeavingReach: right before the provoker leaves reach.
const AttackTimingRightBeforeLeavingReach AttackTiming = 1

// String returns "right_before_leaving_reach" or "unknown".
func (t AttackTiming) String() string {
	return enumName([]string{"right_before_leaving_reach"}, int(t))
}

// PostureChange is a change of posture, a closed set. Variants: DropProneChange,
// StandUpChange.
type PostureChange interface{ isPostureChange() }

// DropProneChange: drop Prone, which costs nothing (SRD 5.2.1, "Dropping Prone", p. 14).
type DropProneChange struct{}

// StandUpChange: stand up from Prone at the cost and refusal the supplied ProneMovement gives.
type StandUpChange struct{ Movement ProneMovement }

func (DropProneChange) isPostureChange() {}
func (StandUpChange) isPostureChange()   {}

// DodgeBenefits says whether the Dodge benefits apply to a combatant.
type DodgeBenefits int

// The two values.
const (
	// DodgeBenefitsActive: attack rolls against it have Disadvantage if the attacker is seen, and its Dexterity saves have Advantage.
	DodgeBenefitsActive DodgeBenefits = iota + 1
	// DodgeBenefitsNone: neither benefit applies.
	DodgeBenefitsNone
)

// String returns "active", "none" or "unknown".
func (b DodgeBenefits) String() string { return enumName([]string{"active", "none"}, int(b)) }

// AllyConsent says whether the ally consents to an Initiative swap. The zero value is invalid.
type AllyConsent int

// The two values.
const (
	AllyConsentWilling AllyConsent = iota + 1
	AllyConsentUnwilling
)

// Valid reports whether c is one of the two.
func (c AllyConsent) Valid() bool { return c == AllyConsentWilling || c == AllyConsentUnwilling }

// String returns "willing", "unwilling" or "unknown".
func (c AllyConsent) String() string { return enumName([]string{"willing", "unwilling"}, int(c)) }

// ActivationCost is what activating a magic item costs a combatant, a closed set.
// Variants: ActivatesByAction, ActivatesByBonusAction, ActivationCostDescribed.
type ActivationCost interface{ isActivationCost() }

// ActivatesByAction: the activation takes the action in Choice.
type ActivatesByAction struct{ Choice ActionChoice }

// ActivatesByBonusAction: the activation takes the Bonus Action the grant gives.
type ActivatesByBonusAction struct{ Grant BonusActionGrant }

// ActivationCostDescribed: the item's own text decides; the host picks a choice.
type ActivationCostDescribed struct{}

func (ActivatesByAction) isActivationCost()       {}
func (ActivatesByBonusAction) isActivationCost()  {}
func (ActivationCostDescribed) isActivationCost() {}

// ActivationCosts maps an ActivationTime to the cost it implies. The zero value is
// ready to use.
type ActivationCosts struct{}

// Of returns the cost of activating item when its ActivationTime is t: the Magic
// action for MagicAction, the Utilize action for Action (an interpretation of
// "Utilize [Action]", p. 191), the item's Bonus Action for BonusAction, and
// ActivationCostDescribed for Reaction, DescribedElsewhere and any other value. It
// is total and never panics.
func (ActivationCosts) Of(t content.ActivationTime, item content.ItemID) ActivationCost {
	switch t {
	case content.ActivationMagicAction:
		return ActivatesByAction{SimpleActionChoice{SimpleStandardActionMagic}}
	case content.ActivationAction:
		return ActivatesByAction{SimpleActionChoice{SimpleStandardActionUtilize}}
	case content.ActivationBonusAction:
		return ActivatesByBonusAction{BonusActionGrant{BonusActionFromItem{item}, NoStandardAction{}}}
	case content.ActivationReaction, content.ActivationDescribedElsewhere:
	}
	return ActivationCostDescribed{}
}
