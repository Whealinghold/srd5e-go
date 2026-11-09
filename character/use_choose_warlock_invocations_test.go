package character_test

import (
	"context"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-warlock.md FR-043 to FR-053 - the use case reads the Invocations column of the class row and the options from the class definition, validates the full selection with SelectInvocations and stores it: at level 3 the three entitled invocations are accepted, the new state and the InvocationsChosen event are returned and the command's Character is not altered; the count comes from the row, so two at level 3 or four at level 2 are refused.
// Path: per case (happy; unhappy: boundary count).
func TestChooseWarlockInvocationsStoresTheSelection(t *testing.T) {
	t.Parallel()
	known := []character.CantripFacts{warCantrips()["eldritch_blast"]}
	picks := []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("agonizing_blast", "eldritch_blast", ""), warPick("devils_sight", "", "")}
	run := func(c character.Character, p []character.InvocationPick) character.ChooseWarlockInvocationsResult {
		res, err := must(character.NewChooseWarlockInvocations(warSource(t))).Run(context.Background(), character.ChooseWarlockInvocationsCommand{Character: c, Picks: p, KnownCantrips: known})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	c := warChar(t, 3, 16, "", character.WarlockState{})
	got, ok := run(c, picks).(character.ChooseWarlockInvocationsApplied)
	if !ok {
		t.Fatal("not applied")
	}
	if want := []content.FeatureOptionID{"agonizing_blast", "devils_sight", "pact_of_the_blade"}; !slices.Equal(warStateOf(t, got.Character()).Invocations(), want) {
		t.Errorf("Invocations = %v, want %v", warStateOf(t, got.Character()).Invocations(), want)
	}
	if ev := got.Events(); len(ev) != 1 || ev[0].CharacterID() != c.ID() || len(ev[0].(character.InvocationsChosen).Picks()) != 3 {
		t.Errorf("events = %+v", ev)
	}
	got.Events()[0] = nil
	if got.Events()[0] == nil {
		t.Error("Events must return a copy")
	}
	if len(warStateOf(t, c).Invocations()) != 0 {
		t.Error("the command's Character was altered")
	}
	for name, p := range map[string][]character.InvocationPick{"two at level 3": picks[:2], "four at level 3": append(slices.Clone(picks), warPick("eldritch_mind", "", ""))} {
		rej, ok := run(c, p).(character.ChooseWarlockInvocationsRejected)
		if !ok || rej.Rejection.Kind != character.WarlockInvocationSelectionInvalid || len(rej.Rejection.Problems) != 1 || rej.Rejection.Problems[0].Kind != character.InvocationWrongCount || rej.Rejection.Problems[0].Expected != 3 {
			t.Errorf("%s: = %+v", name, rej)
		}
	}
	// At level 2 the row also entitles three; at level 5 it entitles five, and a replacement at level 5 keeps the rest.
	five := warChar(t, 5, 16, "", warState(warProfile(5, ""), character.WarlockStateParts{Invocations: []content.FeatureOptionID{"pact_of_the_blade", "eldritch_mind", "devils_sight"}}))
	more := []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("eldritch_mind", "", ""), warPick("devils_sight", "", ""), warPick("thirsting_blade", "", ""), warPick("armor_of_shadows", "", "")}
	if _, ok := run(five, more).(character.ChooseWarlockInvocationsApplied); !ok {
		t.Errorf("level 5 selection = %+v", run(five, more))
	}
	// A selection holding an invocation the class does not list names the option in the problem.
	rej := run(c, []character.InvocationPick{warPick("fireball", "", ""), warPick("eldritch_mind", "", ""), warPick("devils_sight", "", "")}).(character.ChooseWarlockInvocationsRejected)
	if len(rej.Rejection.Problems) != 1 || rej.Rejection.Problems[0].Kind != character.InvocationUnknown || rej.Rejection.Problems[0].Option != "fireball" {
		t.Errorf("unknown option = %+v", rej.Rejection)
	}
}
