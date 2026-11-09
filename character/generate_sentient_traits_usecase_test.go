package character_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func chosenTraits(t *testing.T) character.GenerateSentientItemTraitsCommand {
	t.Helper()
	return character.GenerateSentientItemTraitsCommand{Intelligence: score(t, 10), Wisdom: score(t, 11), Charisma: score(t, 12), Alignment: content.AlignmentLawfulGood,
		Communication: content.SentientTransmitsEmotion, Senses: must(content.SentientSensesForRoll(1))}
}

func generate(t *testing.T, roller character.Roller, cmd character.GenerateSentientItemTraitsCommand, opts ...character.Option) (character.GenerateSentientItemTraitsResult, error) {
	t.Helper()
	return must(character.NewGenerateSentientItemTraits(roller, opts...)).Do(context.Background(), cmd)
}

// Requirement: doc/specs/magic-item-rules.md FR-097 - requests are made only for the traits asked to roll, in the order Intelligence,
// Wisdom, Charisma (4d6), alignment (1d100), communication (1d10), senses (1d4), special purpose (1d10); none for a chosen trait.
// Path: happy.
func TestGenerateSentientItemTraitsRollOrder(t *testing.T) {
	t.Parallel()
	roller := dicetest.NewFixedRoller(6, 6, 6, 6, 1, 1, 1, 1, 6, 6, 6, 1, 20, 3, 2, 7)
	cmd := character.GenerateSentientItemTraitsCommand{RollIntelligence: true, RollWisdom: true, RollCharisma: true, RollAlignment: true, RollCommunication: true, RollSenses: true, RollPurpose: true,
		PurposeDetail: character.PurposeDetail{Target: "the realm"}}
	res, err := generate(t, roller, cmd)
	want := []dice.Expression{dice.MustExpression(4, dice.D6(), 0), dice.MustExpression(4, dice.D6(), 0), dice.MustExpression(4, dice.D6(), 0), dice.MustExpression(1, dice.D100(), 0),
		dice.MustExpression(1, dice.D10(), 0), dice.MustExpression(1, dice.D4(), 0), dice.MustExpression(1, dice.D10(), 0)}
	if err != nil || !reflect.DeepEqual(roller.Requests(), want) {
		t.Fatalf("err = %v, requests = %v", err, roller.Requests())
	}
	g, ok := res.(character.GenerateSentientItemTraitsGenerated)
	if !ok {
		t.Fatalf("result = %#v", res)
	}
	tr := g.Traits()
	if tr.Intelligence().Value() != 18 || tr.Wisdom().Value() != 3 || tr.Charisma().Value() != 18 || tr.Alignment() != content.AlignmentNeutralGood {
		t.Errorf("traits = %+v", tr)
	}
	if _, ok := tr.SpecialPurpose().Purpose(); !ok {
		t.Error("no special purpose")
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-097 - a chosen trait makes no request; a rolled purpose needing a detail the
// command did not supply is purposeDetailMissing; a chosen unaligned alignment, a Neutral item with an Aligned purpose and an invalid
// Aligned purpose are rejections, not errors).
// Path: per case.
func TestGenerateSentientItemTraitsRejectionsAndChoices(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Sentient Magic Item Traits", pp. 207-208`
	chosen := chosenTraits(t)
	with := func(f func(*character.GenerateSentientItemTraitsCommand)) character.GenerateSentientItemTraitsCommand {
		c := chosen
		f(&c)
		return c
	}
	aligned := func(a, opposed content.Alignment) character.SpecialPurposeSelection {
		return character.SelectPurpose(character.MustAlignedPurpose(a, opposed))
	}
	cases := []struct {
		name, requirement, path string
		cmd                     character.GenerateSentientItemTraitsCommand
		faces                   []int
		requests                int
		want                    character.GenerateRejection // 0 generated
		wantKind                content.SpecialPurposeKind
	}{
		{"all chosen", p, "happy", chosen, nil, 0, 0, 0},
		{"chosen unaligned", p, "unhappy (invalid input)", with(func(c *character.GenerateSentientItemTraitsCommand) { c.Alignment = content.AlignmentUnaligned }), nil, 0, character.GenerateUnalignedNotAllowed, 0},
		{"rolled Aligned purpose needs an opposed alignment", p, "unhappy (missing content)", with(func(c *character.GenerateSentientItemTraitsCommand) { c.RollPurpose = true }), []int{1}, 1, character.GeneratePurposeDetailMissing, content.SpecialPurposeAligned},
		{"rolled Bane purpose needs a creature type", p, "unhappy (missing content)", with(func(c *character.GenerateSentientItemTraitsCommand) { c.RollPurpose = true }), []int{2}, 1, character.GeneratePurposeDetailMissing, content.SpecialPurposeBane},
		{"rolled Protector purpose needs a target", p, "unhappy (missing content)", with(func(c *character.GenerateSentientItemTraitsCommand) { c.RollPurpose = true }), []int{8}, 1, character.GeneratePurposeDetailMissing, content.SpecialPurposeProtector},
		{"rolled Templar purpose needs a target", p, "unhappy (missing content)", with(func(c *character.GenerateSentientItemTraitsCommand) { c.RollPurpose = true }), []int{10}, 1, character.GeneratePurposeDetailMissing, content.SpecialPurposeTemplar},
		{"rolled Aligned purpose with the detail", p, "happy", with(func(c *character.GenerateSentientItemTraitsCommand) {
			c.RollPurpose, c.PurposeDetail = true, character.PurposeDetail{Opposed: content.AlignmentChaoticEvil}
		}), []int{1}, 1, 0, 0},
		{"rolled Aligned purpose, not opposed", p, "unhappy (invalid input)", with(func(c *character.GenerateSentientItemTraitsCommand) {
			c.RollPurpose, c.PurposeDetail = true, character.PurposeDetail{Opposed: content.AlignmentLawfulEvil}
		}), []int{1}, 1, character.GenerateAlignedPurposeInvalid, content.SpecialPurposeAligned},
		{"rolled Neutral with rolled Aligned purpose", p, "unhappy (rule interaction)", with(func(c *character.GenerateSentientItemTraitsCommand) {
			c.RollAlignment, c.RollPurpose, c.PurposeDetail = true, true, character.PurposeDetail{Opposed: content.AlignmentChaoticEvil}
		}), []int{64, 1}, 2, character.GenerateAlignedPurposeNeedsNonNeutralAlignment, content.SpecialPurposeAligned},
		{"rolled Neutral at the top of the band", p, "unhappy (boundary)", with(func(c *character.GenerateSentientItemTraitsCommand) {
			c.RollAlignment, c.RollPurpose, c.PurposeDetail = true, true, character.PurposeDetail{Opposed: content.AlignmentChaoticEvil}
		}), []int{73, 1}, 2, character.GenerateAlignedPurposeNeedsNonNeutralAlignment, content.SpecialPurposeAligned},
		{"chosen Neutral with chosen Aligned purpose", p, "unhappy (rule interaction)", with(func(c *character.GenerateSentientItemTraitsCommand) {
			c.Alignment, c.Purpose = content.AlignmentNeutral, aligned(content.AlignmentLawfulGood, content.AlignmentChaoticEvil)
		}), nil, 0, character.GenerateAlignedPurposeNeedsNonNeutralAlignment, content.SpecialPurposeAligned},
		{"chosen Aligned purpose that is no longer opposed", p, "unhappy (invalid input)", with(func(c *character.GenerateSentientItemTraitsCommand) {
			c.Alignment, c.Purpose = content.AlignmentChaoticGood, aligned(content.AlignmentLawfulGood, content.AlignmentChaoticEvil)
		}), nil, 0, character.GenerateAlignedPurposeInvalid, content.SpecialPurposeAligned},
		{"rolled non-Neutral alignment", p, "happy", with(func(c *character.GenerateSentientItemTraitsCommand) { c.RollAlignment = true }), []int{1}, 1, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roller := dicetest.NewFixedRoller(c.faces...)
			res, err := generate(t, roller, c.cmd)
			if err != nil || len(roller.Requests()) != c.requests {
				t.Fatalf("err = %v, %d requests [%s]", err, len(roller.Requests()), c.requirement)
			}
			if c.want == 0 {
				if _, ok := res.(character.GenerateSentientItemTraitsGenerated); !ok {
					t.Errorf("result = %#v", res)
				}
				return
			}
			rej, ok := res.(character.GenerateSentientItemTraitsRejected)
			if !ok || rej.Reason() != c.want || rej.PurposeKind() != c.wantKind {
				t.Errorf("result = %#v, want %v/%v", res, c.want, c.wantKind)
			}
		})
	}
	rej := must(generate(t, dicetest.NewFixedRoller(64, 1), with(func(c *character.GenerateSentientItemTraitsCommand) {
		c.RollAlignment, c.RollPurpose, c.PurposeDetail = true, true, character.PurposeDetail{Opposed: content.AlignmentChaoticEvil}
	}))).(character.GenerateSentientItemTraitsRejected)
	if rej.Alignment() != content.AlignmentNeutral {
		t.Errorf("rejected alignment = %v", rej.Alignment())
	}
	for r := character.GeneratePurposeDetailMissing; r <= character.GenerateAlignedPurposeNeedsNonNeutralAlignment; r++ {
		if r.String() == "unknown" {
			t.Errorf("reason %d", r)
		}
	}
	if character.GenerateRejection(0).String() != "unknown" {
		t.Error("String")
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-097 - every rolled purpose kind builds its purpose (with detail where needed).
// Path: per case.
func TestGenerateSentientItemTraitsEveryPurposeKind(t *testing.T) {
	t.Parallel()
	detail := character.PurposeDetail{Opposed: content.AlignmentChaoticEvil, Creature: content.CreatureTypeUndead, Target: "the realm"}
	for roll := 1; roll <= 10; roll++ {
		cmd := chosenTraits(t)
		cmd.RollPurpose, cmd.PurposeDetail = true, detail
		res, err := generate(t, dicetest.NewFixedRoller(roll), cmd)
		g, ok := res.(character.GenerateSentientItemTraitsGenerated)
		if err != nil || !ok {
			t.Fatalf("roll %d: %#v, %v", roll, res, err)
		}
		if p, has := g.Traits().SpecialPurpose().Purpose(); !has || p.Kind() != must(content.SpecialPurposeKindForRoll(roll)) {
			t.Errorf("roll %d: purpose %v", roll, p)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-105, FR-106, FR-109 - the no-op bundle returns the same result, signals of both results,
// roller failures are recorded once, invalid input is refused before any signal.
// Path: per case.
func TestGenerateSentientItemTraitsSignalsAndFailures(t *testing.T) {
	t.Parallel()
	good, bad := chosenTraits(t), chosenTraits(t)
	bad.Alignment = content.AlignmentUnaligned
	for _, c := range []struct {
		name string
		cmd  character.GenerateSentientItemTraitsCommand
		want map[string]any
	}{
		{"generated", good, map[string]any{telemetry.AttrOutcome: "success", character.AttrResult: "generated"}},
		{"rejected", bad, map[string]any{telemetry.AttrOutcome: "failure", character.AttrResult: "rejected", character.AttrRejection: "unaligned_not_allowed"}},
	} {
		rec := telemetrytest.NewRecorder()
		res, err := generate(t, dicetest.NewFixedRoller(), c.cmd, character.WithTelemetry(rec.Telemetry()))
		plain, err2 := generate(t, dicetest.NewFixedRoller(), c.cmd)
		if err != nil || err2 != nil || !reflect.DeepEqual(res, plain) {
			t.Fatalf("%s: results differ", c.name)
		}
		checkOneCall(t, rec, character.SpanGenerateSentientTraits, "generate_sentient_traits", telemetry.StatusUnset, c.want)
	}
	cause := errors.New("boom")
	rollAll := character.GenerateSentientItemTraitsCommand{RollIntelligence: true}
	for name, r := range map[string]character.Roller{
		"fails":  dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause }),
		"panics": dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic("boom") }),
	} {
		rec := telemetrytest.NewRecorder()
		res, err := generate(t, r, rollAll, character.WithTelemetry(rec.Telemetry()))
		var pe *srderr.PortError
		if res != nil || !errors.As(err, &pe) || srderr.ErrorType(err) != "dice_roller_error" {
			t.Errorf("%s: %v %v", name, res, err)
		}
		if spans := rec.Tracer.Spans(); len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
			t.Errorf("%s: spans %+v", name, spans)
		}
		noLogsOrViolations(t, rec)
	}
	for stage, cmd := range map[string]character.GenerateSentientItemTraitsCommand{
		"alignment":     {RollAlignment: true},
		"communication": {RollCommunication: true},
		"senses":        {RollSenses: true},
		"purpose":       {RollPurpose: true},
	} {
		fail := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, cause })
		if _, err := generate(t, fail, cmd); !errors.Is(err, cause) {
			t.Errorf("%s: %v", stage, err)
		}
	}
	if _, err := character.NewGenerateSentientItemTraits(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: %v", err)
	}
	if r, err := (character.GenerateSentientItemTraits{}).Do(context.Background(), good); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case: %v %v", r, err)
	}
	if r, err := generate(t, dicetest.NewFixedRoller(), character.GenerateSentientItemTraitsCommand{}); r != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("unset chosen traits: %v %v", r, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := must(character.NewGenerateSentientItemTraits(dicetest.NewFixedRoller())).Do(cancelled, good); r != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v %v", r, err)
	}
}
