package character_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md section 7 - the signal names, attribute values and
// the new rejection values are a public contract: their strings are pinned here and each appears in
// doc/observability/signals.md and, for the attribute values, in doc/rules/observability.md.
// Path: happy.
func TestHigherLevelStartSignalNamesArePinnedAndCatalogued(t *testing.T) {
	t.Parallel()
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := os.ReadFile("../doc/rules/observability.md")
	if err != nil {
		t.Fatal(err)
	}
	pins := []struct{ got, want string }{
		{character.SpanAdvancementStartAtHigherLevel, "srd5e.character.advancement.start_at_higher_level"},
		{character.SpanAdvancementStartingMoney, "srd5e.character.advancement.starting_money"},
		{character.SpanAdvancementTrinket, "srd5e.character.advancement.trinket"},
		{character.MetricStartSteps, "srd5e.character.start_steps"},
		{character.ValueOperationStartAtHigherLevel, "start_at_higher_level"},
		{character.ValueOperationStartingMoney, "starting_money"},
		{character.ValueOperationTrinket, "trinket"},
		{character.ValueRejectionPlanLengthMismatch, "plan_length_mismatch"},
		{character.ValueRejectionCreationInvalid, "creation_invalid"},
		{character.ValueRejectionLevelUpRejected, "level_up_rejected"},
		{character.ValueRejectionSubclassRejected, "subclass_rejected"},
		{character.ValueRejectionIncreaseRejected, "increase_rejected"},
		{character.ValueRejectionIncreaseNotGranted, "increase_not_granted"},
		{character.ValueRejectionSubclassRequired, "subclass_required"},
		{character.ValueResultStarted, "started"},
		{character.ValueResultRolled, "rolled"},
		{character.ValueResultNotApplicable, "not_applicable"},
	}
	for _, p := range pins {
		if p.got != p.want {
			t.Errorf("constant = %q, want %q", p.got, p.want)
		}
		if !strings.HasPrefix(p.want, "srd5e.") && !strings.Contains(string(rules), "`"+p.want+"`") {
			t.Errorf("observability.md does not list %q", p.want)
		}
		if strings.HasPrefix(p.want, "srd5e.") && !strings.Contains(string(catalogue), "`"+p.want+"`") {
			t.Errorf("signals.md does not list %q", p.want)
		}
		if !strings.HasPrefix(p.want, "srd5e.") && !strings.Contains(string(catalogue), "`"+p.want+"`") {
			t.Errorf("signals.md does not list %q", p.want)
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-122, section 7 - each rule rejection is recorded as a
// normal outcome (failure, result rejected, the rejection value of its variant, no error status), once, with
// the right rejection value.
// Path: per case.
func TestStartAtHigherLevelRejectionValues(t *testing.T) {
	t.Parallel()
	f := hlsFighter(t)
	for _, c := range []struct {
		name string
		cmd  character.StartAtHigherLevelCommand
		want string
	}{
		{"level up rejected", func() character.StartAtHigherLevelCommand {
			cmd := startCmd(t, 3, 15, []character.StartingLevelStep{fixedStep("fighter", "", character.AbilityScoreIncrease{}), fixedStep("wizard", "", character.AbilityScoreIncrease{})})
			cmd.AbilityScores = scoresOf(t, 15, 10, 14, 10, 10, 10)
			return cmd
		}(), "level_up_rejected"},
		{"subclass rejected", startCmd(t, 2, 15, []character.StartingLevelStep{fixedStep("fighter", "champion", character.AbilityScoreIncrease{})}), "subclass_rejected"},
		{"increase not granted", startCmd(t, 2, 15, []character.StartingLevelStep{fixedStep("fighter", "", plusTwo(ability.Strength))}), "increase_not_granted"},
		{"increase rejected", startCmd(t, 4, 19, fighterSteps(3, false, map[int]character.AbilityScoreIncrease{4: plusTwo(ability.Strength)})), "increase_rejected"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			uc, rec := recordedStart(t, hlsSource(t, f, advWizard(t)), dicetest.NewMinimumRoller())
			if _, ok := startOf(t, uc, c.cmd).(character.StartAtHigherLevelRejected); !ok {
				t.Fatal("want a rejected result")
			}
			spans := rec.Tracer.Spans()
			if len(spans) != 2 || spans[0].Status != telemetry.StatusUnset || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v", spans)
			}
			sameAttrs(t, "operations", onlyOne(t, rec, character.MetricOperations).Attrs, opAttrs("start_at_higher_level",
				map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: c.want}))
			noLogsOrViolations(t, rec)
		})
	}
}
