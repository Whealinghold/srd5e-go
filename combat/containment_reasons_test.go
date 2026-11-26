package combat_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// containmentVariant names the variant of a result of the five containment use cases with an exhaustive type
// switch, so a variant added to a closed set without a case here fails the test.
func containmentVariant(v any) string {
	switch r := v.(type) {
	case combat.EnterContainerResult:
		switch r.(type) {
		case combat.EnterContainerContained, combat.EnterContainerResisted, combat.EnterContainerDamagedOnly, combat.EnterContainerRejected, combat.EnterContainerInvalid:
			return fmt.Sprintf("%T", r)
		}
	case combat.ExitContainerResult:
		switch r.(type) {
		case combat.ExitContainerExited, combat.ExitContainerHeld, combat.ExitContainerRejected:
			return fmt.Sprintf("%T", r)
		}
	case combat.AdvanceContainerTurnResult:
		switch r.(type) {
		case combat.AdvanceContainerTurnAdvanced, combat.AdvanceContainerTurnInvalid:
			return fmt.Sprintf("%T", r)
		}
	case combat.ResolveRegurgitationResult:
		switch r.(type) {
		case combat.ResolveRegurgitationRegurgitated, combat.ResolveRegurgitationHeld, combat.ResolveRegurgitationNotDue, combat.ResolveRegurgitationInvalid:
			return fmt.Sprintf("%T", r)
		}
	case combat.ReleaseOccupantsResult:
		switch r.(type) {
		case combat.ReleaseOccupantsReleased, combat.ReleaseOccupantsRejected, combat.ReleaseOccupantsInvalid:
			return fmt.Sprintf("%T", r)
		}
	}
	return "unknown"
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-066, FR-076, FR-077 - a consumer in package combat_test builds each command and type-switches exhaustively over every result variant (no generic name), the terms' closed sets and the two reason enums, with the snake_case names of section 7.
// Path: happy.
func TestContainmentPublicAPI(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, c := range containmentRows(t) {
		res, err := c.run()
		if err != nil {
			t.Fatal(err)
		}
		if name := containmentVariant(res); name == "unknown" {
			t.Errorf("%s: %T is not a variant of a closed set", c.name, res)
		} else {
			seen[name] = true
		}
	}
	if len(seen) != 17 {
		t.Errorf("the rows reach %d result variants, want all 17: %v", len(seen), seen)
	}
	var rejections []string
	for r := combat.ContainmentRejectionContainerIsContained; r <= combat.ContainmentRejectionInsufficientMovement; r++ {
		rejections = append(rejections, r.String())
	}
	var problems []string
	for p := combat.ContainmentProblemSameContainerAndOccupant; p <= combat.ContainmentProblemUnknownOccupant; p++ {
		problems = append(problems, p.String())
	}
	if fmt.Sprint(rejections) != "[container_is_contained occupant_is_container occupant_already_contained not_grappled_by_container occupant_grappled_elsewhere too_large no_room not_contained no_way_out puller_is_contained puller_out_of_reach no_action not_trapped insufficient_movement]" ||
		fmt.Sprint(problems) != "[same_container_and_occupant circumstance_not_applicable invalid_terms missing_occupant_conditions unknown_occupant]" ||
		combat.ContainmentRejection(0).String() != "unknown" || combat.ContainmentProblem(99).String() != "unknown" {
		t.Errorf("reason names changed: %v %v", rejections, problems)
	}
	for _, e := range []struct {
		got, want string
	}{{combat.EntryCircumstanceNone.String(), "none"}, {combat.EntryCircumstanceEnteredContainerSpace.String(), "entered_container_space"}, {combat.EntryCircumstance(0).String(), "unknown"},
		{combat.DamagedOnlyReasonTooLarge.String(), "too_large"}, {combat.DamagedOnlyReasonNoRoom.String(), "no_room"}, {combat.DamagedOnlyReasonGrappleImmune.String(), "grapple_immune"}, {combat.DamagedOnlyReason(0).String(), "unknown"},
		{combat.ContainmentCoverNone.String(), "none"}, {combat.ContainmentCoverTotal.String(), "total"},
		{combat.ConditionImpositionAdded.String(), "added"}, {combat.ConditionImpositionAlreadyHeld.String(), "already_held"}, {combat.ConditionImpositionImmune.String(), "immune"},
		{combat.ContainmentDamageCauseEntry.String(), "entry"}, {combat.ContainmentDamageCausePeriodic.String(), "periodic"}, {combat.ContainmentDamageCausePull.String(), "pull"},
		{combat.ContainmentActionKindBonusAction.String(), "bonus_action"}, {combat.ContainmentActionKindAction.String(), "action"},
		{combat.SuffocationRuleNone.String(), "none"}, {combat.SuffocationRuleAlways.String(), "always"}, {combat.SuffocationRuleUnlessBreathesWater.String(), "unless_breathes_water"},
		{combat.ContainerTurnBoundaryStart.String(), "start"}, {combat.ContainerTurnBoundaryEnd.String(), "end"}, {combat.ContainerTurnBoundary(0).String(), "unknown"}} {
		if e.got != e.want {
			t.Errorf("name %q, want %q", e.got, e.want)
		}
	}
	for _, valid := range []bool{combat.SuffocationRule(0).Valid(), combat.SuffocationRule(4).Valid(), combat.ContainmentTiming(0).Valid(), combat.ContainmentActionKind(3).Valid(),
		combat.ContainmentActionKind(0).Valid(), combat.ContainerTurnBoundary(3).Valid(), combat.ContainerTurnBoundary(0).Valid()} {
		if valid {
			t.Error("an out-of-range enum value must not be valid")
		}
	}
	var (
		_ combat.ExitPlacement = combat.NearestUnoccupiedSpace{}
		_ combat.ExitPlacement = combat.LandingWithin{}
	)
}

// Example of a Behir swallowing a grappled creature that fails its Dexterity saving throw.
func ExampleEnterContainer() {
	terms := combat.MustContainmentTerms(combat.ContainmentTermsInput{
		Mode: combat.ContainmentModeSwallowed, MaxSize: 4, Capacity: combat.MustContainmentCapacity(1, nil),
		Entry:   combat.MustEntryBySave(1, 18, nil, false, 0, false),
		Imposed: ctBR(), TotalCover: true, Suffocation: combat.SuffocationRuleNone, DeathRule: combat.MustEscapeFromCorpse(15),
		ActionKind: combat.ContainmentActionKindBonusAction,
	})
	grapple := combat.MustGrapple("behir", "knight", combat.MustGraspingHand(0), 14, 5)
	cmd, err := combat.NewEnterContainerCommand(combat.EnterContainerInput{
		Grapples: combat.MustGrapples([]combat.Grapple{grapple}), Container: "behir", Occupant: "knight", OccupantSize: 3, Terms: terms,
		OccupantConditions: combat.MustConditionSet([]combat.Condition{grapple.Condition()}, nil),
		Circumstance:       combat.EntryCircumstanceNone, Attempt: rollOf(2),
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	uc, _ := combat.NewEnterContainer(dicetest.NewFixedRoller(5))
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	if held, ok := res.(combat.EnterContainerContained); ok {
		fmt.Println(held.Containments().Len(), held.Grapples().Len(), held.Conditions().Effective())
	}
	// Output: 1 0 [Blinded Restrained]
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-012, FR-015, FR-022 - the Must forms of the state panic with the constructor's error, the typed error renders and matches by problem, and a forced test carries its d20 result.
// Path: unhappy (invalid input).
func TestContainmentErrorsAndForcedChecks(t *testing.T) {
	t.Parallel()
	func() {
		defer func() {
			if err, _ := recover().(error); err == nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("MustContainments panicked with %v, want an ErrInvalidInput error", err)
			}
		}()
		combat.MustContainments([]combat.Containment{{}})
	}()
	e := &combat.ContainmentError{Problem: combat.ContainmentProblemUnknownOccupant}
	if e.Error() == "" || !errors.Is(e, &combat.ContainmentError{Problem: combat.ContainmentProblemUnknownOccupant}) || errors.Is(e, &combat.ContainmentError{Problem: combat.ContainmentProblemInvalidTerms}) ||
		errors.Is(e, errors.New("other")) {
		t.Error("ContainmentError must match by problem only")
	}
	in := combat.ResolveRegurgitationInput{Containments: ctDamaged(t, ctHolding(t, "worm", "m", "p"), map[string]int{"p": 31}), Container: gid("m"), OccupantConditions: ctConds(t, "p"), Attempt: d20.ChooseToFail{}}
	res, _ := ctRegurgitate(t, in)
	forced, ok := res.(combat.ResolveRegurgitationRegurgitated).Check().(combat.ContainmentTestForced)
	if !ok || forced.Result().Reason() != d20.ChosenToFail {
		t.Errorf("check = %#v, want the forced failure chosen to fail", res)
	}
	in.Attempt = rollOf(1)
	res, _ = ctRegurgitate(t, in, 20)
	if held := res.(combat.ResolveRegurgitationHeld); held.Check().(combat.ContainmentTestRolled).Result().Outcome() != d20.Success {
		t.Error("the held result carries the success")
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-012 - ContainmentError.Is never matches a nil error value on either side, nor an error of another type.
// Path: unhappy (empty and minimal).
func TestContainmentErrorIsToleratesNil(t *testing.T) {
	t.Parallel()
	e := &combat.ContainmentError{Problem: combat.ContainmentProblemInvalidTerms}
	var none *combat.ContainmentError
	if none.Is(e) || e.Is(none) || none.Is(none) || e.Is(nil) || e.Is(errors.New("other")) || !e.Is(&combat.ContainmentError{Problem: combat.ContainmentProblemInvalidTerms}) {
		t.Error("Is must match a ContainmentError of the same problem and nothing else, nil included")
	}
}
