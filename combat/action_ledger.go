package combat

import "slices"

// AttackActionState is whether a combatant has an Attack action in progress (SRD 5.2.1, "Attack
// [Action]", p. 177). Variants: NoAttackActionInProgress, AttackActionInProgress.
type AttackActionState interface{ isAttackActionState() }

// NoAttackActionInProgress: no Attack action is in progress.
type NoAttackActionInProgress struct{}

// AttackActionInProgress: an Attack action is in progress; Progress counts its attacks.
type AttackActionInProgress struct{ Progress AttackActionProgress }

func (NoAttackActionInProgress) isAttackActionState() {}
func (AttackActionInProgress) isAttackActionState()   {}

// MultiattackState is whether a monster has a Multiattack in progress (SRD 5.2.1, "Multiattack", p. 257;
// combat-action-resolution.md FR-165). Variants: NoMultiattack, MultiattackInProgress.
type MultiattackState interface{ isMultiattackState() }

// NoMultiattack: no Multiattack is in progress.
type NoMultiattack struct{}

// MultiattackInProgress: a Multiattack is in progress; Progress counts its attacks, additional uses and
// replacements. Not comparable with ==.
type MultiattackInProgress struct{ Progress MultiattackProgress }

func (NoMultiattack) isMultiattackState()         {}
func (MultiattackInProgress) isMultiattackState() {}

// ActionLedger is the state that outlives a combat call and that the Encounter does not hold: the Attack
// action in progress and the pending Help benefits (combat-action-resolution.md FR-112). The host threads
// it from one call's result to the next call's command. The zero value is a new ledger with no Attack
// action, no Multiattack and no Help benefit. It holds a slice, so compare with Equal. Immutable; safe for concurrent use.
type ActionLedger struct {
	state AttackActionState
	helps HelpBenefits
	multi MultiattackState
}

// NewActionLedger returns the ledger holding the Attack action state and the Help benefits, for a host
// restoring saved state. It returns an error matching srderr.ErrInvalidInput for a state that is neither
// variant or holds an invalid progress. A nil state is NoAttackActionInProgress. It never panics.
func NewActionLedger(state AttackActionState, helps HelpBenefits) (ActionLedger, error) {
	switch s := state.(type) {
	case nil, NoAttackActionInProgress:
	case AttackActionInProgress:
		if !s.Progress.Valid() {
			return ActionLedger{}, invalidInput("a ledger needs a valid progress")
		}
	default:
		return ActionLedger{}, invalidInput("a ledger needs an Attack action state")
	}
	return ActionLedger{state, helps, nil}, nil
}

// WithMultiattack returns the ledger holding the Multiattack state, for a host restoring saved state. It returns an
// error matching srderr.ErrInvalidInput for a state that is neither variant or holds a zero progress. A nil
// state is NoMultiattack. It never panics.
func (l ActionLedger) WithMultiattack(state MultiattackState) (ActionLedger, error) {
	switch s := state.(type) {
	case nil, NoMultiattack:
		l.multi = nil
	case MultiattackInProgress:
		if s.Progress.IsZero() {
			return ActionLedger{}, invalidInput("a ledger needs a valid Multiattack progress")
		}
		l.multi = s
	default:
		return ActionLedger{}, invalidInput("a ledger needs a Multiattack state")
	}
	return l, nil
}

// Multiattack returns whether a Multiattack is in progress.
func (l ActionLedger) Multiattack() MultiattackState {
	if l.multi == nil {
		return NoMultiattack{}
	}
	return l.multi
}

// AttackAction returns whether an Attack action is in progress.
func (l ActionLedger) AttackAction() AttackActionState {
	if l.state == nil {
		return NoAttackActionInProgress{}
	}
	return l.state
}

// HelpBenefits returns the pending Help benefits.
func (l ActionLedger) HelpBenefits() HelpBenefits { return l.helps }

// Equal reports whether l and o hold the same Attack action state and Help benefits.
func (l ActionLedger) Equal(o ActionLedger) bool {
	a, aok := l.AttackAction().(AttackActionInProgress)
	b, bok := o.AttackAction().(AttackActionInProgress)
	ma, maok := l.Multiattack().(MultiattackInProgress)
	mb, mbok := o.Multiattack().(MultiattackInProgress)
	return aok == bok && (!aok || a.Progress.Equal(b.Progress)) && l.helps.Equal(o.helps) &&
		maok == mbok && (!maok || ma.Progress.Equal(mb.Progress))
}

// progressFor returns the progress held for the attacker, if there is one.
func (l ActionLedger) progressFor(attacker CombatantID) (AttackActionProgress, bool) {
	p, ok := l.AttackAction().(AttackActionInProgress)
	return p.Progress, ok && p.Progress.IsFor(attacker)
}

// withProgress returns the ledger holding p as its Attack action.
func (l ActionLedger) withProgress(p AttackActionProgress) ActionLedger {
	l.state = AttackActionInProgress{p}
	return l
}

// withHelps returns the ledger holding h.
func (l ActionLedger) withHelps(h HelpBenefits) ActionLedger {
	l.helps = h
	return l
}

// ledgerStep is the outcome of a ledger operation: the new ledger and the events, or the refusal.
type ledgerStep struct {
	ledger  ActionLedger
	events  []Event
	refusal LedgerRefusal
	// limitedUse is the new LimitedUseState of a Multiattack step; it is zero for any other step.
	limitedUse LimitedUseState
}

// attackEvents widens the Attack action events to Events.
func attackEvents(in []AttackActionEvent) []Event {
	out := make([]Event, len(in))
	for i, e := range in {
		out[i] = e
	}
	return out
}

// recordAttack applies the ledger part of an attack's cost (FR-113, FR-116): an Attack action begins or
// continues and the attack is recorded; any other cost changes nothing. The record is a returned value;
// the host's ledger is never altered.
func (l ActionLedger) recordAttack(attacker CombatantID, c AttackCost) (ledgerStep, error) {
	switch v := c.(type) {
	case BeginsAttackAction:
		p, err := StartAttackActionProgress(attacker, v.Allowance)
		if err != nil {
			return ledgerStep{}, err
		}
		return l.record(p.RecordAttack(v.Means)), nil
	case ContinuesAttackAction:
		p, ok := l.progressFor(attacker)
		if !ok {
			return ledgerStep{ledger: l, refusal: LedgerNoAttackActionInProgress{}}, nil
		}
		return l.record(p.RecordAttack(v.Means)), nil
	case RecordsMultiattack:
		p, ok := l.multiattackFor(attacker)
		if !ok {
			return ledgerStep{ledger: l, refusal: LedgerNoAttackActionInProgress{}}, nil
		}
		out, err := p.RecordAttack(v.Part, v.LimitedUse)
		return l.recordMultiattack(out, err)
	}
	return ledgerStep{ledger: l}, nil
}

// recordReplacement applies the ledger part of a replaced attack (FR-122).
func (l ActionLedger) recordReplacement(attacker CombatantID, c ReplacementCost, r AttackReplacement) (ledgerStep, error) {
	switch v := c.(type) {
	case BeginsAttackActionByReplacing:
		p, err := StartAttackActionProgress(attacker, v.Allowance)
		if err != nil {
			return ledgerStep{}, err
		}
		return l.record(p.RecordReplacement(r)), nil
	case ContinuesAttackActionByReplacing:
		p, ok := l.progressFor(attacker)
		if !ok {
			return ledgerStep{ledger: l, refusal: LedgerNoAttackActionInProgress{}}, nil
		}
		return l.record(p.RecordReplacement(r)), nil
	}
	return ledgerStep{}, invalidInput("a replacement needs a cost")
}

// record turns the outcome of recording on p into a ledger step; l is what a refusal leaves.
func (l ActionLedger) record(out AttackRecordOutcome) ledgerStep {
	switch o := out.(type) {
	case AttackActionAttackRecorded:
		return ledgerStep{ledger: l.withProgress(o.Progress()), events: attackEvents(o.Events())}
	case AttackActionAttackRefused:
		return ledgerStep{ledger: l, refusal: LedgerAttackRefused{o.Reason()}}
	default:
		panic(unreachableError("attack record outcome", out))
	}
}

// handleWeapon applies a weapon change to the Attack action in progress (FR-118).
func (l ActionLedger) handleWeapon(attacker CombatantID, change AttackActionWeaponChange, timing AttackActionWeaponChangeTiming) ledgerStep {
	p, ok := l.progressFor(attacker)
	if !ok {
		return ledgerStep{ledger: l, refusal: LedgerNoAttackActionInProgress{}}
	}
	switch o := p.HandleWeapon(change, timing).(type) {
	case AttackActionWeaponChangeRecorded:
		return ledgerStep{ledger: l.withProgress(o.Progress()), events: attackEvents(o.Events())}
	case AttackActionWeaponChangeRefused:
		return ledgerStep{ledger: l, refusal: LedgerWeaponChangeRefused{o.Reason()}}
	default:
		panic(unreachableError("weapon change outcome", o))
	}
}

// ended returns the ledger after a turn ends for the combatant whose turn starts: the Attack action ends
// with the turn (FR-115) and the Help benefits the new active combatant gave expire (FR-118).
func (l ActionLedger) ended(newActive CombatantID) (ActionLedger, []Event) {
	exp := l.helps.ExpireAtStartOfTurn(newActive)
	events := make([]Event, 0, len(exp.Events()))
	for _, e := range exp.Events() {
		events = append(events, e)
	}
	return ActionLedger{state: NoAttackActionInProgress{}, helps: exp.Benefits(), multi: NoMultiattack{}}, events
}

// multiattackFor returns the Multiattack in progress if it is the combatant's.
func (l ActionLedger) multiattackFor(combatant CombatantID) (MultiattackProgress, bool) {
	m, ok := l.Multiattack().(MultiattackInProgress)
	return m.Progress, ok && slices.Contains(m.Progress.Combatants(), combatant)
}

// withMultiattack returns the ledger holding p as its Multiattack.
func (l ActionLedger) withMultiattack(p MultiattackProgress) ActionLedger {
	l.multi = MultiattackInProgress{p}
	return l
}

// recordMultiattack turns the outcome of a Multiattack step on p into a ledger step (FR-165): the new progress,
// the new limited-use state and the events, or the refusal with the ledger as it was.
func (l ActionLedger) recordMultiattack(out MultiattackStepOutcome, err error) (ledgerStep, error) {
	if err != nil {
		return ledgerStep{}, err
	}
	if rec, ok := out.(MultiattackStepRecorded); ok {
		t := rec.Transition
		return ledgerStep{ledger: l.withMultiattack(t.Progress()), events: t.Events(), limitedUse: t.LimitedUse()}, nil
	}
	return ledgerStep{ledger: l, refusal: LedgerMultiattackRefused{out}}, nil
}

// recordMultiattackRequest applies the additional use or the replacement of a Multiattack request (FR-165).
func (l ActionLedger) recordMultiattackRequest(attacker CombatantID, r TurnRequest) (ledgerStep, error) {
	p, ok := l.multiattackFor(attacker)
	if !ok {
		return ledgerStep{ledger: l, refusal: LedgerNoAttackActionInProgress{}}, nil
	}
	switch v := r.(type) {
	case RecordMultiattackUse:
		out, err := p.RecordAdditionalUse(v.Part, v.LimitedUse)
		return l.recordMultiattack(out, err)
	case RecordMultiattackReplacement:
		out, err := p.RecordReplacement(v.Option, v.Replaced, v.LimitedUse)
		return l.recordMultiattack(out, err)
	}
	return ledgerStep{}, invalidInput("a Multiattack request is a use or a replacement")
}
