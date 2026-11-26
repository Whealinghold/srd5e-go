package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// StatBlockActionCost is what using a part of a stat block costs the monster, by the section
// the part is printed in (SRD 5.2.1, "Traits", "Actions", p. 256; "Bonus Action", "Reactions",
// "Legendary Actions", p. 257). The zero value is invalid. Comparable and safe for concurrent use.
type StatBlockActionCost int

// The five costs.
const (
	// StatBlockCostPassive: a Trait is active at all times or in certain situations and is never used.
	StatBlockCostPassive StatBlockActionCost = iota + 1
	// StatBlockCostAction: the monster's action.
	StatBlockCostAction
	// StatBlockCostBonusAction: the monster's Bonus Action.
	StatBlockCostBonusAction
	// StatBlockCostReaction: the monster's Reaction.
	StatBlockCostReaction
	// StatBlockCostLegendaryAction: a Legendary Action, which costs none of the three.
	StatBlockCostLegendaryAction
)

// Valid reports whether c is one of the five costs.
func (c StatBlockActionCost) Valid() bool {
	return c >= StatBlockCostPassive && c <= StatBlockCostLegendaryAction
}

// String returns "passive", "action", "bonus_action", "reaction", "legendary_action" or "unknown".
func (c StatBlockActionCost) String() string {
	return enumName([]string{"passive", "action", "bonus_action", "reaction", "legendary_action"}, int(c))
}

// StatBlockActionCostOf returns the cost of using a part of section: passive for Traits, action
// for Actions, bonus action for Bonus Actions, reaction for Reactions and legendary action for
// Legendary Actions. It returns the zero cost for an undefined section.
func StatBlockActionCostOf(section MonsterPartSection) StatBlockActionCost {
	if !section.Valid() {
		return 0
	}
	// The five sections sit in the order of the five costs.
	return StatBlockActionCost(section)
}

// ReactionTriggerReport is the host's report on whether the trigger of a Reaction happened. The
// trigger text stays prose and is never read (content-monsters FR-041). The zero value is invalid.
// Comparable and safe for concurrent use.
type ReactionTriggerReport int

// The two reports.
const (
	// ReactionTriggerOccurred: the trigger happened.
	ReactionTriggerOccurred ReactionTriggerReport = iota + 1
	// ReactionTriggerNotOccurred: the trigger did not happen.
	ReactionTriggerNotOccurred
)

// String returns "occurred", "not_occurred" or "unknown".
func (r ReactionTriggerReport) String() string {
	return enumName([]string{"occurred", "not_occurred"}, int(r))
}

// StatBlockUseRequest asks to use one part of a stat block as an action, a Bonus Action, a
// Reaction or a Legendary Action. Construct it with a keyed literal. Cost and Part are required;
// Granted defaults to NoStandardAction; Trigger is required for a Reaction; Window and
// Availability are required for a Legendary Action. The other fields are ignored.
type StatBlockUseRequest struct {
	// Cost is the kind of use: StatBlockCostAction, BonusAction, Reaction or LegendaryAction.
	Cost StatBlockActionCost
	// Part is the position of the part.
	Part LimitedUsePartRef
	// Granted is the standard action the part names, if any.
	Granted GrantedAction
	// Trigger reports whether the trigger of a Reaction happened.
	Trigger ReactionTriggerReport
	// Window is the window of a Legendary Action.
	Window LegendaryActionWindow
	// Availability says whether the monster can take actions, for a Legendary Action.
	Availability ConditionActionAvailability
}

// StatBlockUseRefusal is why a use was refused, a closed set. Variants: StatBlockPartNotFound,
// StatBlockPartInWrongSection, StatBlockReactionTriggerNotReported, StatBlockLegendaryUseRefused,
// StatBlockPartUsageExhausted.
type StatBlockUseRefusal interface{ isStatBlockUseRefusal() }

// StatBlockPartNotFound: the monster has no part at the position.
type StatBlockPartNotFound struct{}

// StatBlockPartInWrongSection: the part is printed in a section other than the one the use needs.
type StatBlockPartInWrongSection struct{}

// StatBlockReactionTriggerNotReported: the host reported that the trigger did not happen.
type StatBlockReactionTriggerNotReported struct{}

// StatBlockLegendaryUseRefused: the Legendary Action could not be taken, for the reason named.
type StatBlockLegendaryUseRefused struct{ Reason LegendaryActionRefusal }

// StatBlockPartUsageExhausted: the part has a usage limit and no use remains or it is spent.
type StatBlockPartUsageExhausted struct{}

func (StatBlockPartNotFound) isStatBlockUseRefusal()               {}
func (StatBlockPartInWrongSection) isStatBlockUseRefusal()         {}
func (StatBlockReactionTriggerNotReported) isStatBlockUseRefusal() {}
func (StatBlockLegendaryUseRefused) isStatBlockUseRefusal()        {}
func (StatBlockPartUsageExhausted) isStatBlockUseRefusal()         {}

// StatBlockUseTransition is the change one use of a part makes: the new limited-use state, the
// economy of the use (the action, Bonus Action or Reaction it consumes, none for a Legendary
// Action) and the events. Immutable; every accessor returns a copy.
type StatBlockUseTransition struct {
	limitedUse LimitedUseState
	cost       StatBlockActionCost
	action     FeatureActionChoice
	bonus      BonusActionGrant
	reaction   FeatureReaction
	events     []Event
}

// LimitedUse returns the limited-use state after the use.
func (t StatBlockUseTransition) LimitedUse() LimitedUseState { return t.limitedUse }

// Cost returns what the use cost.
func (t StatBlockUseTransition) Cost() StatBlockActionCost { return t.cost }

// ActionChoice returns the choice to record in the turn tracker, and false unless the use cost an action.
func (t StatBlockUseTransition) ActionChoice() (FeatureActionChoice, bool) {
	return t.action, t.cost == StatBlockCostAction
}

// BonusActionGrant returns the grant to record, and false unless the use cost a Bonus Action.
func (t StatBlockUseTransition) BonusActionGrant() (BonusActionGrant, bool) {
	return t.bonus, t.cost == StatBlockCostBonusAction
}

// Reaction returns the Reaction to record, and false unless the use cost a Reaction.
func (t StatBlockUseTransition) Reaction() (FeatureReaction, bool) {
	return t.reaction, t.cost == StatBlockCostReaction
}

// Events returns the events in the order they happened, as a new slice: the limited-use events of
// the Legendary Action and of the usage spent, then StatBlockPartUsed.
func (t StatBlockUseTransition) Events() []Event { return slices.Clone(t.events) }

// StatBlockUseOutcome is the closed set of outcomes of UseStatBlock. Variants:
// StatBlockUseAllowed, StatBlockUseRefused.
type StatBlockUseOutcome interface{ isStatBlockUseOutcome() }

// StatBlockUseAllowed: the part may be used; the transition says what it changes.
type StatBlockUseAllowed struct{ Transition StatBlockUseTransition }

// StatBlockUseRefused: the part may not be used, for the refusal named. Nothing changed.
type StatBlockUseRefused struct{ Refusal StatBlockUseRefusal }

func (StatBlockUseAllowed) isStatBlockUseOutcome() {}
func (StatBlockUseRefused) isStatBlockUseOutcome() {}

// sectionParts returns the parts of section s of m in printed order.
func sectionParts(m content.Monster, s MonsterPartSection) []content.StatBlockPart {
	switch s {
	case MonsterPartTraits:
		return m.Traits()
	case MonsterPartActions:
		return m.Actions()
	case MonsterPartBonusActions:
		return m.BonusActions()
	case MonsterPartReactions:
		return m.Reactions()
	case MonsterPartLegendaryActions:
		l, _ := m.LegendaryActions()
		return l.Actions()
	}
	return nil
}

// hasStatBlockPart reports whether m has a part at ref.
func hasStatBlockPart(m content.Monster, ref LimitedUsePartRef) bool {
	return ref.section.Valid() && ref.index < len(sectionParts(m, ref.section))
}

// limitedEvents widens the events of a limited-use change.
func limitedEvents(es []MonsterLimitedUseEvent) []Event {
	out := make([]Event, len(es))
	for i, e := range es {
		out[i] = e
	}
	return out
}

// spendPart spends one use of part from s. It returns the state and events after, and false when
// the part has a limit and none remains. A part with no limit has nothing to spend and is not refused.
func spendPart(s LimitedUseState, part LimitedUsePartRef) (LimitedUseState, []Event, bool) {
	switch o := s.Spend(part).(type) {
	case LimitedUseSpent:
		return o.Transition.State(), limitedEvents(o.Transition.Events()), true
	case LimitedUseSpendRefused:
		return s, nil, o.Reason == LimitedUseRefusedNotLimited
	default:
		panic(unreachableError("spend outcome", o))
	}
}

// spendSpell is spendPart for a spell of the Spellcasting.
func spendSpell(s LimitedUseState, spell SpellUseRef) (LimitedUseState, []Event, bool) {
	switch o := s.SpendSpell(spell).(type) {
	case LimitedUseSpent:
		return o.Transition.State(), limitedEvents(o.Transition.Events()), true
	case LimitedUseSpendRefused:
		return s, nil, o.Reason == LimitedUseRefusedNotLimited
	default:
		panic(unreachableError("spend outcome", o))
	}
}

// checkStatBlockState returns an error matching srderr.ErrInvalidInput unless m, combatant and the
// limited-use state describe one monster combatant. A LimitExceededError (plan P10)
// is returned for a stat block of more than 64 parts, spells or spell groups.
func checkStatBlockState(m content.Monster, combatant CombatantID, limitedUse LimitedUseState) error {
	switch {
	case m.ID() == "" || combatant.IsZero() || limitedUse.IsZero():
		return invalidInput("a stat-block use needs a monster, a combatant and a limited-use state")
	case limitedUse.combatant != combatant || limitedUse.monsterID != m.ID():
		return invalidInput("the limited-use state belongs to another monster combatant")
	}
	return monsterCollectionsInRange(m)
}

// UseStatBlock uses one part of a stat block (monster-stat-block-actions.md FR-001 to FR-009;
// SRD 5.2.1, "Actions", p. 256; "Bonus Action", "Reactions", "Legendary Actions", "Limited Usage",
// p. 257). It checks, in this order, that the part exists, that it is printed in the section the
// cost needs, for a Reaction that the host reported the trigger, for a Legendary Action that one can
// be taken, and that a part with a limit has a use left. The first that fails is returned as
// StatBlockUseRefused and nothing changes. Otherwise the part's limit is spent (a part without one
// is not), a Legendary Action spends a Legendary Action use, and the transition carries the
// economy: a FeatureActionChoice, a BonusActionGrant or a FeatureReaction naming the part's
// FeatureID, and none for a Legendary Action. The Multiattack part is used like any other here;
// StartMultiattack is the entry point that also tracks its attacks.
//
// It returns an error matching srderr.ErrInvalidInput for the zero monster, combatant or
// limited-use state, a state of another combatant or monster, a Cost that is not an action, Bonus
// Action, Reaction or Legendary Action, a Granted action that is not valid, a Reaction without a
// defined Trigger report, a Legendary Action without a Window or Availability, and, for a use that
// records the part by FeatureID, the *StatBlockFeatureIDRejected of a part whose id cannot be
// derived. It never panics.
func UseStatBlock(m content.Monster, combatant CombatantID, req StatBlockUseRequest, limitedUse LimitedUseState) (StatBlockUseOutcome, error) {
	if err := checkStatBlockState(m, combatant, limitedUse); err != nil {
		return nil, err
	}
	granted := req.Granted
	if granted == nil {
		granted = NoStandardAction{}
	}
	switch {
	case req.Cost < StatBlockCostAction || req.Cost > StatBlockCostLegendaryAction || req.Part.IsZero() || !granted.valid():
		return nil, invalidInput("a stat-block use needs an action, Bonus Action, Reaction or Legendary Action cost, a part and a valid granted action")
	case req.Cost == StatBlockCostReaction && (req.Trigger < ReactionTriggerOccurred || req.Trigger > ReactionTriggerNotOccurred):
		return nil, invalidInput("a Reaction needs a trigger report")
	case req.Cost == StatBlockCostLegendaryAction && (req.Window.IsZero() || !validAvailability(req.Availability)):
		return nil, invalidInput("a Legendary Action needs a window and an action availability")
	}
	if !hasStatBlockPart(m, req.Part) {
		return StatBlockUseRefused{StatBlockPartNotFound{}}, nil
	}
	if StatBlockActionCostOf(req.Part.section) != req.Cost {
		return StatBlockUseRefused{StatBlockPartInWrongSection{}}, nil
	}
	var feature content.FeatureID
	if req.Cost != StatBlockCostLegendaryAction {
		var err error
		if feature, err = StatBlockFeatureID(m, req.Part); err != nil {
			return nil, err
		}
	}
	if req.Cost == StatBlockCostReaction && req.Trigger != ReactionTriggerOccurred {
		return StatBlockUseRefused{StatBlockReactionTriggerNotReported{}}, nil
	}
	state, events := limitedUse, []Event(nil)
	if req.Cost == StatBlockCostLegendaryAction {
		out, err := state.TakeLegendaryAction(req.Window, req.Availability)
		if err != nil {
			return nil, err
		}
		switch o := out.(type) {
		case LegendaryActionRefused:
			return StatBlockUseRefused{StatBlockLegendaryUseRefused(o)}, nil
		case LegendaryActionAllowed:
			state, events = o.Transition.State(), limitedEvents(o.Transition.Events())
		default:
			panic(unreachableError("legendary action outcome", o))
		}
	}
	state, spent, ok := spendPart(state, req.Part)
	if !ok {
		return StatBlockUseRefused{StatBlockPartUsageExhausted{}}, nil
	}
	t := StatBlockUseTransition{limitedUse: state, cost: req.Cost,
		events: append(slices.Concat(events, spent), StatBlockPartUsed{statBlockActionEventBase{source: combatant}, req.Part, req.Cost})}
	switch req.Cost {
	case StatBlockCostPassive, StatBlockCostLegendaryAction:
		// A Legendary Action records no action, Bonus Action or Reaction; a passive cost was rejected above.
	case StatBlockCostAction:
		t.action = FeatureActionChoice{Feature: feature, Granted: granted}
	case StatBlockCostBonusAction:
		grant, err := NewBonusActionGrant(BonusActionFromFeature{feature}, granted)
		if err != nil {
			return nil, err
		}
		t.bonus = grant
	case StatBlockCostReaction:
		t.reaction = FeatureReaction{Feature: feature, Granted: granted}
	}
	return StatBlockUseAllowed{t}, nil
}

// StatBlockFeatureIDReason is why the FeatureID of a part could not be derived. The zero value is
// invalid. Comparable and safe for concurrent use.
type StatBlockFeatureIDReason int

// The three reasons.
const (
	// FeatureIDInvalidName: the part's name has no letter or digit.
	FeatureIDInvalidName StatBlockFeatureIDReason = iota + 1
	// FeatureIDTooLong: the id is longer than content.MaxIDLength.
	FeatureIDTooLong
	// FeatureIDDuplicate: another part of the monster derives the same id.
	FeatureIDDuplicate
)

// String returns "invalid_name", "too_long", "duplicate" or "unknown".
func (r StatBlockFeatureIDReason) String() string {
	return enumName([]string{"invalid_name", "too_long", "duplicate"}, int(r))
}

// StatBlockFeatureIDRejected is the error of a part whose FeatureID cannot be derived, and the
// element of StatBlockFeatureIDProblems. It matches srderr.ErrInvalidInput with errors.Is, and
// another *StatBlockFeatureIDRejected with the same Reason. Comparable.
//
//nolint:errname // the typed error named by the stat-block actions spec
type StatBlockFeatureIDRejected struct {
	part   LimitedUsePartRef
	reason StatBlockFeatureIDReason
}

// Part returns the part whose id was rejected.
func (e StatBlockFeatureIDRejected) Part() LimitedUsePartRef { return e.part }

// Reason returns why.
func (e StatBlockFeatureIDRejected) Reason() StatBlockFeatureIDReason { return e.reason }

// Error renders the rejection. The text is not a contract.
func (e StatBlockFeatureIDRejected) Error() string {
	return problemText("stat-block feature id", e.reason.String())
}

// Unwrap returns srderr.ErrInvalidInput.
func (e StatBlockFeatureIDRejected) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *StatBlockFeatureIDRejected with the same reason.
func (e StatBlockFeatureIDRejected) Is(target error) bool {
	t, ok := target.(*StatBlockFeatureIDRejected)
	return ok && t.reason == e.reason
}

// partSlug is the part id of a part name: the name lower-cased, apostrophes removed and each
// other run of characters outside a-z and 0-9 one underscore, the ends trimmed. It is the rule
// content.DeriveMonsterID applies, without that function's requirement of a leading letter.
func partSlug(name string) string {
	var b strings.Builder
	pending := false
	for _, r := range name {
		switch {
		case r == '\'' || r == '’':
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			if pending && b.Len() > 0 {
				b.WriteByte('_')
			}
			pending = false
			b.WriteRune(r | 0x20)
		default:
			pending = true
		}
	}
	return b.String()
}

// derivePartFeatureID returns the id "<monster id>__<part id>" of the part called name, and
// the reason it cannot be derived.
func derivePartFeatureID(monster content.MonsterID, name string) (content.FeatureID, StatBlockFeatureIDReason) {
	slug := partSlug(name)
	if slug == "" {
		return "", FeatureIDInvalidName
	}
	full := monster.String() + "__" + slug
	if len(full) > content.MaxIDLength {
		return "", FeatureIDTooLong
	}
	id, err := content.ParseFeatureID(full)
	if err != nil {
		return "", FeatureIDInvalidName
	}
	return id, 0
}

// statBlockPartIDs returns the derived id of every part of m in printed order, the empty id for a
// part that has none, and the reason it has none.
func statBlockPartIDs(m content.Monster) ([]LimitedUsePartRef, []content.FeatureID, []StatBlockFeatureIDReason) {
	var refs []LimitedUsePartRef
	var ids []content.FeatureID
	var reasons []StatBlockFeatureIDReason
	for s := MonsterPartTraits; s <= MonsterPartLegendaryActions; s++ {
		for i, p := range sectionParts(m, s) {
			id, why := derivePartFeatureID(m.ID(), p.Name())
			refs, ids, reasons = append(refs, LimitedUsePartRef{s, i}), append(ids, id), append(reasons, why)
		}
	}
	return refs, ids, reasons
}

// StatBlockFeatureID returns the FeatureID of a part of m: "<monster id>__<part id>", the part id
// being the part name lower-cased, apostrophes removed and each other run of characters outside
// a-z and 0-9 one underscore. It is not qualified by section, to stay inside the 64 characters of
// content.MaxIDLength (monster-stat-block-actions.md FR-005).
//
// It returns an error matching srderr.ErrInvalidInput for the zero monster or a part m does not
// have, and a *StatBlockFeatureIDRejected for a name with no letter or digit, an id over the limit
// and a part another part of m derives the same id as. It never panics.
func StatBlockFeatureID(m content.Monster, part LimitedUsePartRef) (content.FeatureID, error) {
	if !hasStatBlockPart(m, part) {
		return "", invalidInput("the zero monster, or a monster with no such part")
	}
	refs, ids, reasons := statBlockPartIDs(m)
	i := slices.Index(refs, part)
	if reasons[i] != 0 {
		return "", &StatBlockFeatureIDRejected{part, reasons[i]}
	}
	if slices.Index(ids, ids[i]) != i || slices.Index(ids[i+1:], ids[i]) >= 0 {
		return "", &StatBlockFeatureIDRejected{part, FeatureIDDuplicate}
	}
	return ids[i], nil
}

// StatBlockFeatureIDProblems returns, in printed order, the parts of m whose FeatureID cannot be
// derived: one entry per part with a name that has no letter or digit, an id over the limit, or an
// id another part derives too (both parts are listed). Hosts check it once per monster. It returns
// nil for the zero monster or when every part has an id.
func StatBlockFeatureIDProblems(m content.Monster) []StatBlockFeatureIDRejected {
	refs, ids, reasons := statBlockPartIDs(m)
	var out []StatBlockFeatureIDRejected
	for i, ref := range refs {
		switch {
		case reasons[i] != 0:
			out = append(out, StatBlockFeatureIDRejected{ref, reasons[i]})
		case slices.Index(ids, ids[i]) != i || slices.Index(ids[i+1:], ids[i]) >= 0:
			out = append(out, StatBlockFeatureIDRejected{ref, FeatureIDDuplicate})
		}
	}
	return out
}
