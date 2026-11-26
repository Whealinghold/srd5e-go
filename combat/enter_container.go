package combat

import (
	"context"
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// EntryCircumstance is a circumstance of an entry that changes its saving throw. The zero value is
// invalid.
type EntryCircumstance int

// The two circumstances.
const (
	// EntryCircumstanceNone: no circumstance.
	EntryCircumstanceNone EntryCircumstance = iota + 1
	// EntryCircumstanceEnteredContainerSpace: the creature entered the container's space, which gives the
	// save Disadvantage (SRD 5.2.1, "Gelatinous Cube", p. 287). Valid for the Engulfed mode only.
	EntryCircumstanceEnteredContainerSpace
)

// String returns "none", "entered_container_space" or "unknown".
func (c EntryCircumstance) String() string {
	return enumName([]string{"none", "entered_container_space"}, int(c))
}

// EnterContainerInput is the raw values of an EnterContainerCommand. Build it with a keyed literal.
type EnterContainerInput struct {
	// Containments is the creatures held, which the host stores.
	Containments Containments
	// Grapples is the grapples in force.
	Grapples Grapples
	// Container is the monster whose Swallow, Engulf or Whelm resolves.
	Container CombatantID
	// Occupant is the target.
	Occupant CombatantID
	// OccupantSize is the target's size.
	OccupantSize content.CreatureSize
	// Terms are the monster's terms.
	Terms ContainmentTerms
	// OccupantConditions are the target's conditions.
	OccupantConditions ConditionSet
	// Circumstance is EntryCircumstanceNone or EntryCircumstanceEnteredContainerSpace.
	Circumstance EntryCircumstance
	// Attempt is the target's saving throw: a d20.RollAttempt carrying the ability's modifier,
	// d20.ChooseToFail or d20.AutomaticFailure. It is required for EntryBySave and ignored otherwise.
	Attempt d20.SaveAttempt
	// RelocationSpaceAvailable says whether an unoccupied space within the entry's relocation distance is free.
	RelocationSpaceAvailable bool
	// DuringContainerTurn says whether the entry happens during the container's own turn.
	DuringContainerTurn bool
}

// EnterContainerCommand is the input of EnterContainer.Run. Build it with NewEnterContainerCommand; the
// zero value is invalid.
type EnterContainerCommand struct{ in EnterContainerInput }

// NewEnterContainerCommand returns the command. It returns an error matching srderr.ErrInvalidInput for a
// blank id, an invalid size, circumstance or terms, an EntryBySave without an attempt of the three kinds,
// or a container that already holds creatures under other terms. Two equal ids are a result of Run, not
// an error. It copies the attempt and never panics. It returns a LimitExceededError for a d20.RollAttempt of more than 256 circumstances (plan P10).
func NewEnterContainerCommand(in EnterContainerInput) (EnterContainerCommand, error) {
	if in.Container.IsZero() || in.Occupant.IsZero() || !in.OccupantSize.Valid() || !in.Terms.valid() ||
		in.Circumstance < EntryCircumstanceNone || in.Circumstance > EntryCircumstanceEnteredContainerSpace {
		return EnterContainerCommand{}, invalidInput("enter needs ids, a size, terms and a circumstance")
	}
	if _, saves := in.Terms.entry.(EntryBySave); saves {
		if err := attemptOverCap(in.Attempt); err != nil {
			return EnterContainerCommand{}, err
		}
		a, ok := checkSave(in.Attempt)
		if !ok {
			return EnterContainerCommand{}, invalidInput("enter by save needs a roll, a choice to fail or an automatic failure")
		}
		in.Attempt = a
	}
	for _, r := range in.Containments.OccupantsOf(in.Container) {
		if r.terms != in.Terms {
			return EnterContainerCommand{}, invalidInput("a container has one set of terms")
		}
	}
	return EnterContainerCommand{in}, nil
}

// enterState is what the three applied results of EnterContainer share.
type enterState struct {
	containments Containments
	grapples     Grapples
	conditions   ConditionSet
	check        ContainmentTest
	damage       []ContainmentDamageDue
	actionKind   ContainmentActionKind
}

// Containments returns the set after the call.
func (s enterState) Containments() Containments { return s.containments }

// Grapples returns the grapples after the call.
func (s enterState) Grapples() Grapples { return s.grapples }

// Conditions returns the occupant's conditions after the call.
func (s enterState) Conditions() ConditionSet { return s.conditions }

// Check returns the saving throw, or ContainmentTestNotRequired.
func (s enterState) Check() ContainmentTest { return s.check }

// DamageDue returns a new slice of the damage the host rolls and applies; empty for none.
func (s enterState) DamageDue() []ContainmentDamageDue { return slices.Clone(s.damage) }

// ActionKind returns the action the monster spends.
func (s enterState) ActionKind() ContainmentActionKind { return s.actionKind }

// EnterContainerResult is the closed set of outcomes of EnterContainer.Run. Variants:
// EnterContainerContained, EnterContainerResisted, EnterContainerDamagedOnly, EnterContainerRejected,
// EnterContainerInvalid.
type EnterContainerResult interface{ isEnterContainerResult() }

// EnterContainerContained: the creature is held. The result carries the new set, grapples and conditions.
type EnterContainerContained struct {
	enterState
	effects   ContainmentEffects
	placement ContainedPlacement
	imposed   []ImposedCondition
	events    []Event
}

// Effects returns what the host does: choking, forbidden spells and teleport, Total Cover.
func (r EnterContainerContained) Effects() ContainmentEffects { return r.effects }

// Placement says the occupant is in the container's space and moves with it.
func (r EnterContainerContained) Placement() ContainedPlacement { return r.placement }

// Imposed returns a new slice of what applying each condition did, the Grappled of an Engulfed-grappled
// creature first; a condition the creature was immune to or already had is reported here and never removed.
func (r EnterContainerContained) Imposed() []ImposedCondition { return slices.Clone(r.imposed) }

// Events returns a copy of the events, in the order they happened.
func (r EnterContainerContained) Events() []Event { return slices.Clone(r.events) }

// EnterContainerResisted: the save succeeded and the creature is not held.
type EnterContainerResisted struct {
	enterState
	relocation RelocationDue
	relocates  bool
}

// Relocation returns the move the creature makes and whether it makes one.
func (r EnterContainerResisted) Relocation() (RelocationDue, bool) { return r.relocation, r.relocates }

// EnterContainerDamagedOnly: the save failed but the creature is not held, for the reason.
type EnterContainerDamagedOnly struct {
	enterState
	reason  DamagedOnlyReason
	imposed []ImposedCondition
}

// Reason says why the creature is not held.
func (r EnterContainerDamagedOnly) Reason() DamagedOnlyReason { return r.reason }

// Imposed returns a new slice of the condition that could not be applied (the Grappled of
// DamagedOnlyReasonGrappleImmune); empty otherwise.
func (r EnterContainerDamagedOnly) Imposed() []ImposedCondition { return slices.Clone(r.imposed) }

// EnterContainerRejected: the rules refuse the entry in this state. No die was rolled. Construct it with a
// keyed literal.
type EnterContainerRejected struct{ Rejection ContainmentRejection }

// EnterContainerInvalid: a value of the command is unusable. No die was rolled. Construct it with a keyed
// literal.
type EnterContainerInvalid struct{ Problem ContainmentProblem }

func (EnterContainerContained) isEnterContainerResult()   {}
func (EnterContainerResisted) isEnterContainerResult()    {}
func (EnterContainerDamagedOnly) isEnterContainerResult() {}
func (EnterContainerRejected) isEnterContainerResult()    {}
func (EnterContainerInvalid) isEnterContainerResult()     {}

// EnterContainer is the use case of a Swallow, Engulf or Whelm landing on a creature (SRD 5.2.1, "Behir",
// p. 263; "Gelatinous Cube", p. 287; "Shambling Mound", p. 323). It composes the pure d20 functions with
// the dice port and never calls d20.Resolver or another use case, so no srd5e.d20_test signal fires. It
// rolls no damage. Create it with NewEnterContainer; the zero value is not usable.
type EnterContainer struct {
	containmentUse
	flow attackFlow
}

// NewEnterContainer returns the use case rolling through roller. Options: WithTelemetry, WithClock. It
// returns an error matching srderr.ErrInvalidInput for a nil roller.
func NewEnterContainer(roller Roller, opts ...Option) (EnterContainer, error) {
	if roller == nil {
		return EnterContainer{}, invalidInput("enter without a dice roller")
	}
	return EnterContainer{newContainmentUse(opts), attackFlow{roller}}, nil
}

// Run resolves the entry. It checks before any die, in this order: two equal ids and a circumstance the mode
// does not allow (EnterContainerInvalid), then a set that already holds 256 records (EnterContainerInvalid with the shared limit_exceeded value, plan P10), then the refusals container is contained, occupant is a container,
// occupant already contained, a Swallow of a creature the container does not grapple, an Engulf-grappled of a
// creature that is grappled, and a creature too large or with no room, unless the entry still needs its save
// for the damage (EntryBySave with TargetAnySize, or with damage). A save is rolled with Disadvantage when
// the creature entered the container's space. A success is EnterContainerResisted with half the damage when
// the entry says so; with a relocation and no free space it fails instead. A failure holds the creature
// (EnterContainerContained: the Grappled of a Swallow ends first, then the imposed conditions apply, and an
// Engulf-grappled begins its grapple first), or deals its damage only (EnterContainerDamagedOnly) when the
// creature is too large, there is no room or the creature cannot be Grappled.
//
// It records the shared signals and srd5e.combat.containment.occupants for a result with a set. It returns
// an error matching srderr.ErrInvalidInput, before any signal, for the zero use case or command, and
// ctx.Err() unwrapped, before any signal, for a cancelled ctx. It returns a *srderr.PortError (error type
// "dice_roller_error") with a nil result when the roller fails, panics or answers another expression; the
// failure is recorded once and never logged.
func (u EnterContainer) Run(ctx context.Context, cmd EnterContainerCommand) (EnterContainerResult, error) {
	if u.flow.roller == nil || cmd.in.Container.IsZero() {
		return nil, invalidInput("zero use case or command")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	op := combatOperation{valueOperationEnterContainer, spanEnterContainer}
	res, err := runContainment(ctx, u.containmentUse, op, func(context.Context) (EnterContainerResult, error) { return u.enter(cmd.in) },
		describeEnterContainer)
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case EnterContainerContained:
		u.recordSet(ctx, op, v.containments)
	case EnterContainerResisted:
		u.recordSet(ctx, op, v.containments)
	case EnterContainerDamagedOnly:
		u.recordSet(ctx, op, v.containments)
	}
	return res, nil
}

// describeEnterContainer classifies a result of EnterContainer.
func describeEnterContainer(r EnterContainerResult) description {
	switch v := r.(type) {
	case EnterContainerContained:
		return description{Class: resultApplied, Name: valueResultContained, Events: v.events}
	case EnterContainerResisted:
		return description{Class: resultApplied, Name: valueResultResisted}
	case EnterContainerDamagedOnly:
		return description{Class: resultApplied, Name: valueResultDamagedOnly}
	case EnterContainerRejected:
		return description{Class: resultRefused, Name: v.Rejection.String()}
	case EnterContainerInvalid:
		return description{Class: resultInvalid, Name: v.Problem.String()}
	default:
		panic(unreachableError("enter container result", r))
	}
}

// dueOf returns the damage due of terms, or nil when terms is empty.
func dueOf(container, target CombatantID, terms damageTerms, halved bool, cause ContainmentDamageCause) []ContainmentDamageDue {
	if terms.n == 0 {
		return nil
	}
	return []ContainmentDamageDue{{Container: container, Target: target, terms: terms, halved: halved, cause: cause}}
}

// enter is the work of Run.
func (u EnterContainer) enter(in EnterContainerInput) (EnterContainerResult, error) {
	t := in.Terms
	switch {
	case in.Container == in.Occupant:
		return EnterContainerInvalid{ContainmentProblemSameContainerAndOccupant}, nil
	case in.Circumstance == EntryCircumstanceEnteredContainerSpace && t.mode != ContainmentModeEngulfed:
		return EnterContainerInvalid{ContainmentProblemCircumstanceNotApplicable}, nil
	}
	if in.Containments.Len() >= maxInputList {
		return EnterContainerInvalid{ContainmentProblemLimitExceeded}, nil
	}
	fit := in.Containments.check(in.Container, in.Occupant, in.OccupantSize, t)
	if fit != 0 && fit != ContainmentRejectionTooLarge && fit != ContainmentRejectionNoRoom {
		return EnterContainerRejected{fit}, nil
	}
	held, grappled := in.Grapples.GrappleOf(in.Occupant).(InGrapple)
	switch {
	case t.mode == ContainmentModeSwallowed && (!grappled || held.grapple.Grappler != in.Container):
		return EnterContainerRejected{ContainmentRejectionNotGrappledByContainer}, nil
	case t.mode == ContainmentModeEngulfedGrappled && grappled:
		return EnterContainerRejected{ContainmentRejectionOccupantGrappledElsewhere}, nil
	}
	by, saves := t.entry.(EntryBySave)
	switch {
	case fit == ContainmentRejectionTooLarge && (!saves || !by.targetAnySize):
		return EnterContainerRejected{fit}, nil
	case fit == ContainmentRejectionNoRoom && (!saves || by.damage.n == 0):
		return EnterContainerRejected{fit}, nil
	}

	state := enterState{in.Containments, in.Grapples, in.OccupantConditions, ContainmentTestNotRequired{}, nil, t.actionKind}
	if saves {
		attempt := in.Attempt
		if ra, ok := attempt.(d20.RollAttempt); ok && in.Circumstance == EntryCircumstanceEnteredContainerSpace {
			ra.Circumstances = append(slices.Clone(ra.Circumstances), modifier.Disadvantage{})
			attempt = ra
		}
		test, err := u.flow.containmentTest(d20.SavingThrow, attempt, by.dc)
		if err != nil {
			return nil, err
		}
		state.check = test
		// A success that has nowhere to relocate to fails instead (SRD 5.2.1, "Gelatinous Cube", p. 287).
		if test.passed() && (by.relocateFeet == 0 || in.RelocationSpaceAvailable) {
			state.damage = dueOf(in.Container, in.Occupant, halfOf(by), by.halfOnSuccess, ContainmentDamageCauseEntry)
			res := EnterContainerResisted{enterState: state}
			if by.relocateFeet > 0 {
				res.relocation, res.relocates = RelocationDue{in.Occupant, by.relocateFeet}, true
			}
			return res, nil
		}
		state.damage = dueOf(in.Container, in.Occupant, by.damage, false, ContainmentDamageCauseEntry)
	}
	if fit == ContainmentRejectionTooLarge {
		return EnterContainerDamagedOnly{state, DamagedOnlyReasonTooLarge, nil}, nil
	}
	if fit == ContainmentRejectionNoRoom {
		return EnterContainerDamagedOnly{state, DamagedOnlyReasonNoRoom, nil}, nil
	}
	return u.hold(in, state), nil
}

// halfOf returns the damage terms a success takes: the entry's damage when it halves them, otherwise none.
func halfOf(by EntryBySave) damageTerms {
	if by.halfOnSuccess {
		return by.damage
	}
	return damageTerms{}
}

// hold holds the creature after a failed (or no) save, as far as the Grappled condition allows.
func (u EnterContainer) hold(in EnterContainerInput, state enterState) EnterContainerResult {
	t := in.Terms
	grapples, conds := in.Grapples, in.OccupantConditions
	var events []Event
	var imposed []ImposedCondition
	ordinal := in.Containments.nextPartOrdinal(in.Container)
	switch t.mode {
	case ContainmentModeSwallowed:
		var ended []Event
		grapples, conds, ended = endHold(grapples, conds, in.Container, in.Occupant, false)
		events = append(events, ended...)
	case ContainmentModeEngulfedGrappled:
		hold, _ := t.Grapple()
		part := func(n int) GraspingBodyPart { return MustGraspingBodyPart(hold.partName + " " + strconv.Itoa(n)) }
		for grapples.IsHolding(in.Container, part(ordinal)) == PartStateHolding {
			ordinal++
		}
		g := MustGrapple(in.Container, in.Occupant, part(ordinal), hold.escapeDC, 0)
		set, applied, result := impose(conds, in.Occupant, g.Condition())
		imposed = append(imposed, ImposedCondition{content.ConditionGrappled, result})
		if result != ConditionImpositionAdded {
			return EnterContainerDamagedOnly{state, DamagedOnlyReasonGrappleImmune, imposed}
		}
		begun := grapples.Begin(g).(GrappleTransitionApplied)
		grapples, conds = begun.grapples, set
		events = append(events, applied...)
		events = append(events, begun.events...)
	case ContainmentModeEngulfed:
	}
	var added kindMask
	for _, k := range kindsOf(t.imposed) {
		var c Condition = Blinded{}
		if k == content.ConditionRestrained {
			c = Restrained{}
		}
		var applied []Event
		var result ConditionImposition
		conds, applied, result = impose(conds, in.Occupant, c)
		imposed = append(imposed, ImposedCondition{k, result})
		if result == ConditionImpositionAdded {
			added |= bit(k)
		}
		events = append(events, applied...)
	}
	p, periodic := t.Periodic()
	tickDeferred := periodic && p.firstTickNextTurn && in.DuringContainerTurn
	entered := in.Containments.enter(in.Container, in.Occupant, in.OccupantSize, t, added, ordinal, tickDeferred).(ContainmentTransitionApplied)
	events = append(events, entered.events...)
	state.containments, state.grapples, state.conditions = entered.containments, grapples, conds
	return EnterContainerContained{state,
		ContainmentEffects{t.suffocation, t.noVerbalSpells, t.noTeleport, t.totalCover},
		ContainedPlacement{in.Container, in.Occupant}, imposed, events}
}
