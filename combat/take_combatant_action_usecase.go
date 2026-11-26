package combat

import (
	"context"
)

// TakeCombatantActionCommand is the immutable input of TakeCombatantAction.Run. Build it with
// NewTakeCombatantActionCommand; the zero value is invalid.
type TakeCombatantActionCommand struct {
	encounter Encounter
	ledger    ActionLedger
	combatant CombatantID
	request   TurnRequest
}

// NewTakeCombatantActionCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a
// zero encounter, a blank id or a nil or invalid request. It never panics.
func NewTakeCombatantActionCommand(encounter Encounter, ledger ActionLedger, combatant CombatantID, request TurnRequest) (TakeCombatantActionCommand, error) {
	if encounter.IsZero() || combatant.IsZero() {
		return TakeCombatantActionCommand{}, invalidInput("an action needs an encounter and a combatant")
	}
	if err := validTurnRequest(request); err != nil {
		return TakeCombatantActionCommand{}, err
	}
	return TakeCombatantActionCommand{encounter, ledger, combatant, request}, nil
}

// TakeCombatantActionResult is the closed set of outcomes of TakeCombatantAction.Run. Variants:
// TakeCombatantActionCompleted, TakeCombatantActionRejected, TakeCombatantActionInvalid.
type TakeCombatantActionResult interface{ isTakeCombatantActionResult() }

// TakeCombatantActionCompleted: the request was carried out. It holds the new encounter and ledger and the
// events in order: the tracker's, then the ledger's.
type TakeCombatantActionCompleted struct {
	ledgerCompleted
	limitedUse LimitedUseState
}

// LimitedUse returns the new LimitedUseState of a stat block use or a Multiattack step and true; the zero state and
// false for any other request (FR-164, FR-165).
func (c TakeCombatantActionCompleted) LimitedUse() (LimitedUseState, bool) {
	return c.limitedUse, !c.limitedUse.IsZero()
}

// TakeCombatantActionRejected: a rule stopped the request; the encounter and ledger are as they came in.
type TakeCombatantActionRejected struct{ ledgerRejected }

// TakeCombatantActionInvalid: the host named a combatant the encounter does not hold; the encounter and
// ledger are as they came in.
type TakeCombatantActionInvalid struct{ ledgerInvalid }

func (TakeCombatantActionCompleted) isTakeCombatantActionResult() {}
func (TakeCombatantActionRejected) isTakeCombatantActionResult()  {}
func (TakeCombatantActionInvalid) isTakeCombatantActionResult()   {}

// TakeCombatantAction is the use case of an action that is not an attack: the action, a Bonus Action, a
// Reaction, the free interaction with an object, the Help action, the handling of a weapon and the
// replacement of an attack (SRD 5.2.1, "Actions", pp. 9-10). It spends the resource and resolves no effect
// (combat-action-resolution.md FR-052). Create it with NewTakeCombatantAction; the zero value is silent and
// works. It rolls no die.
type TakeCombatantAction struct{ resolutionUse }

// NewTakeCombatantAction returns the use case. Options: WithTelemetry, WithClock.
func NewTakeCombatantAction(opts ...Option) TakeCombatantAction {
	return TakeCombatantAction{newResolutionUse(nil, opts)}
}

// Run carries out the request: the stops of the encounter (a Help ally or enemy counts as a target), the
// ledger's refusals, then the tracker's, then the commit. The Incapacitated of the combatant is read from its
// own conditions. It returns the results and errors described for the package.
func (u TakeCombatantAction) Run(ctx context.Context, cmd TakeCombatantActionCommand) (TakeCombatantActionResult, error) {
	if cmd.request == nil {
		return nil, invalidInput("zero command")
	}
	op := combatOperation{valueOperationTakeCombatantAction, spanTakeCombatantAction}
	return execute(ctx, u.resolutionUse, op, func() (TakeCombatantActionResult, error) { return u.take(cmd) })
}

// take is the work of Run.
func (u TakeCombatantAction) take(cmd TakeCombatantActionCommand) (TakeCombatantActionResult, error) {
	rejected := func(r ResolutionRejection) TakeCombatantActionResult {
		return TakeCombatantActionRejected{ledgerRejected{resolutionRejected{r, cmd.encounter}, cmd.ledger}}
	}
	ids := []CombatantID{cmd.combatant}
	if v, ok := cmd.request.(TakeHelpAction); ok {
		switch h := v.Request.(type) {
		case AbilityCheckHelpRequest:
			ids = append(ids, h.Ally)
		case AttackRollHelpRequest:
			ids = append(ids, h.Enemy)
		}
	}
	if s := preconditions(cmd.encounter, ids...); s.stopped() {
		if s.problem != nil {
			return TakeCombatantActionInvalid{ledgerInvalid{resolutionInvalid{s.problem, cmd.encounter}, cmd.ledger}}, nil
		}
		return rejected(s.rejection), nil
	}
	actor, _ := cmd.encounter.Find(cmd.combatant)
	tracker, av := cmd.encounter.tracker, availabilityOf(actor.conditions)

	ledger := cmd.ledger
	var ledgerEvents []Event
	var step TrackerStep
	var limitedUse LimitedUseState
	var err error
	switch v := cmd.request.(type) {
	case SpendAction:
		step, err = tracker.TakeAction(cmd.combatant, av, v.Choice)
	case SpendBonusAction:
		step, err = tracker.TakeBonusAction(cmd.combatant, av, bonusEntitlement(v.Grant))
	case SpendReaction:
		step, err = tracker.TakeReaction(cmd.combatant, av, v.Choice)
	case SpendInteraction:
		step, err = tracker.InteractWithObject(cmd.combatant)
	case TakeHelpAction:
		var refusal LedgerRefusal
		if ledger, ledgerEvents, refusal = giveHelp(cmd.ledger, cmd.combatant, v.Request); refusal != nil {
			if helpOverCap(refusal) {
				return TakeCombatantActionInvalid{ledgerInvalid{resolutionInvalid{LimitExceededProblem{}, cmd.encounter}, cmd.ledger}}, nil
			}
			return rejected(LedgerRejection{refusal}), nil
		}
		step, err = tracker.TakeAction(cmd.combatant, av, SimpleActionChoice{SimpleStandardActionHelp})
	case HandleWeapon:
		l := cmd.ledger.handleWeapon(cmd.combatant, v.Change, v.Timing)
		if l.refusal != nil {
			return rejected(LedgerRejection{l.refusal}), nil
		}
		ledger, ledgerEvents = l.ledger, l.events
	case ReplaceAttack:
		l, lerr := cmd.ledger.recordReplacement(cmd.combatant, v.Cost, v.Replacement)
		if lerr != nil {
			return nil, lerr
		}
		if l.refusal != nil {
			return rejected(LedgerRejection{l.refusal}), nil
		}
		ledger, ledgerEvents = l.ledger, l.events
		if _, begins := v.Cost.(BeginsAttackActionByReplacing); begins {
			step, err = tracker.TakeAction(cmd.combatant, av, SimpleActionChoice{SimpleStandardActionAttack})
		}
	case UseStatBlockPart:
		var refusal ResolutionRejection
		if step, ledgerEvents, limitedUse, refusal, err = u.useStatBlock(tracker, cmd.combatant, av, v); refusal != nil || err != nil {
			if err != nil {
				return nil, err
			}
			return rejected(refusal), nil
		}
	case BeginMultiattack:
		var refusal ResolutionRejection
		if ledger, step, ledgerEvents, limitedUse, refusal, err = u.beginMultiattack(tracker, cmd.ledger, cmd.combatant, av, v); refusal != nil || err != nil {
			if err != nil {
				return nil, err
			}
			return rejected(refusal), nil
		}
	case RecordMultiattackUse, RecordMultiattackReplacement:
		l, lerr := cmd.ledger.recordMultiattackRequest(cmd.combatant, v)
		if lerr != nil {
			return nil, lerr
		}
		if l.refusal != nil {
			return rejected(LedgerRejection{l.refusal}), nil
		}
		ledger, ledgerEvents, limitedUse = l.ledger, l.events, l.limitedUse
	}
	if err != nil {
		return nil, err
	}
	r := &resolution{enc: cmd.encounter}
	switch s := step.(type) {
	case TrackerStepRejected:
		return rejected(TrackerRejection{s.Reason}), nil
	case TrackerStepApplied:
		if err := r.commitTurn(s); err != nil {
			return nil, err
		}
	}
	r.add(ledgerEvents...)
	// No events cap check: the events of this call are bounded by its per-call lists (plan P10), far under 4096.
	if r.exitInvalid() {
		return nil, invalidInput("the encounter is not valid after the call")
	}
	return TakeCombatantActionCompleted{ledgerCompleted{resolutionCompleted{r.enc, r.events}, ledger}, limitedUse}, nil
}

// giveHelp gives the Help the request describes through the ledger: the new ledger and the events, or the
// refusal and the ledger as it was.
func giveHelp(l ActionLedger, helper CombatantID, req HelpRequest) (ActionLedger, []Event, LedgerRefusal) {
	switch v := req.(type) {
	case AbilityCheckHelpRequest:
		o := l.helps.GiveAbilityCheckHelp(helper, v.Ally, v.Proficiency, v.HelperProficiencies, v.Judgement)
		if g, ok := o.(AbilityCheckHelpGiven); ok {
			return l.withHelps(g.Benefits()), helpEventsOf(g.Events()), nil
		}
		return l, nil, LedgerAbilityCheckHelpRefused{o.(AbilityCheckHelpRefused).Reason()}
	case AttackRollHelpRequest:
		o := l.helps.GiveAttackRollHelp(helper, v.Enemy, v.Target)
		if g, ok := o.(AttackRollHelpGiven); ok {
			return l.withHelps(g.Benefits()), helpEventsOf(g.Events()), nil
		}
		return l, nil, LedgerAttackRollHelpRefused{o.(AttackRollHelpRefused).Reason()}
	default:
		panic(unreachableError("help request", req))
	}
}

// helpOverCap reports whether the refusal is a Help that would pass the cap of pending benefits (plan P10), an invalid
// call rather than a refusal of the rules.
func helpOverCap(r LedgerRefusal) bool {
	switch v := r.(type) {
	case LedgerAbilityCheckHelpRefused:
		return v.Reason == AbilityCheckHelpRefusalLimitExceeded
	case LedgerAttackRollHelpRefused:
		return v.Reason == AttackRollHelpRefusalLimitExceeded
	}
	return false
}

// helpEventsOf widens Help events to Events.
func helpEventsOf(in []HelpEvent) []Event {
	out := make([]Event, len(in))
	for i, e := range in {
		out[i] = e
	}
	return out
}

// useStatBlock decides the use of a stat block part (FR-164): UseStatBlock first, so that a refusal costs no
// resource, then the tracker step of the economy it names (none for a Legendary Action or a Trait). It returns
// the step to commit, the events of the transition, the new limited-use state, the rejection of a refused use or
// an error.
func (TakeCombatantAction) useStatBlock(t TurnTracker, id CombatantID, av ConditionActionAvailability, v UseStatBlockPart) (TrackerStep, []Event, LimitedUseState, ResolutionRejection, error) {
	out, err := UseStatBlock(v.Monster, id, v.Request, v.LimitedUse)
	if err != nil {
		return nil, nil, LimitedUseState{}, nil, err
	}
	allowed, ok := out.(StatBlockUseAllowed)
	if !ok {
		return nil, nil, LimitedUseState{}, StatBlockRejection{Refusal: out.(StatBlockUseRefused).Refusal}, nil
	}
	var step TrackerStep
	if spendsEconomy(allowed.Transition) {
		var err error
		if step, err = economyStep(t, id, av, allowed.Transition); err != nil {
			return nil, nil, LimitedUseState{}, nil, err
		}
	}
	return step, allowed.Transition.Events(), allowed.Transition.LimitedUse(), nil, nil
}

// beginMultiattack starts a Multiattack (FR-165): the plan's start, then the action as FR-164 spends it.
func (TakeCombatantAction) beginMultiattack(t TurnTracker, l ActionLedger, id CombatantID, av ConditionActionAvailability, v BeginMultiattack) (ActionLedger, TrackerStep, []Event, LimitedUseState, ResolutionRejection, error) {
	var opts []MultiattackOption
	if v.Granted != nil {
		opts = append(opts, WithGrantedAction(v.Granted))
	}
	opts = append(opts, WithUsedThisTurn(v.UsedThisTurn))
	out, err := StartMultiattack(v.Monster, id, v.LimitedUse, opts...)
	if err != nil {
		return l, nil, nil, LimitedUseState{}, nil, err
	}
	started, ok := out.(MultiattackStarted)
	if !ok {
		return l, nil, nil, LimitedUseState{}, StatBlockRejection{MultiattackNotAvailable: true}, nil
	}
	tr := started.Transition()
	step, err := economyStep(t, id, av, tr)
	return l.withMultiattack(started.Progress()), step, tr.Events(), tr.LimitedUse(), nil, err
}

// economyStep is the tracker step of the economy a stat block use costs: the action, the Bonus Action or the
// Reaction; none for a Legendary Action or a Trait.
func economyStep(t TurnTracker, id CombatantID, av ConditionActionAvailability, tr StatBlockUseTransition) (step TrackerStep, err error) {
	switch {
	case tr.Cost() == StatBlockCostAction:
		choice, _ := tr.ActionChoice()
		step, err = t.TakeAction(id, av, choice)
	case tr.Cost() == StatBlockCostBonusAction:
		grant, _ := tr.BonusActionGrant()
		step, err = t.TakeBonusAction(id, av, EntitledToBonusAction{grant})
	case tr.Cost() == StatBlockCostReaction:
		reaction, _ := tr.Reaction()
		step, err = t.TakeReaction(id, av, reaction)
	default:
		panic(unreachableError("stat block economy", tr.Cost()))
	}
	return step, err
}

// spendsEconomy reports whether a use costs the action, a Bonus Action or a Reaction; a Legendary Action and a Trait cost
// none, so no tracker step is asked for them.
func spendsEconomy(tr StatBlockUseTransition) bool {
	return tr.Cost() == StatBlockCostAction || tr.Cost() == StatBlockCostBonusAction || tr.Cost() == StatBlockCostReaction
}
