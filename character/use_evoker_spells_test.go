package character_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// evkCastCases drives the four per-cast Evoker use cases in the three modes of wzMode (they load nothing).
func evkCastCases() []wzCase {
	cantrip := wzFact("fire_bolt", 0, evkEvo, evkDmg)
	return []wzCase{
		{name: "ResolvePotentCantrip", span: character.SpanResolvePotentCantrip, feature: "potent_cantrip", hasLevels: true, rejection: "feature_not_available",
			okType: "PotentCantripHalfDamage", badType: "ResolvePotentCantripRejected",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return character.NewResolvePotentCantrip(opts...).Run(t.Context(), character.ResolvePotentCantripCommand{ClassLevels: wzPick(mode, 3, 2, 3), Subclass: evkSub(t),
					Spell: cantrip, Kind: d20.AttackRoll, Outcome: d20.Failure, Damage: 8})
			}},
		{name: "ResolveSculptSpells", span: character.SpanResolveSculptSpells, feature: "sculpt_spells", hasLevels: true, rejection: "feature_not_available",
			okType: "SculptedCreatures", badType: "ResolveSculptSpellsRejected",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return character.NewResolveSculptSpells(opts...).Run(t.Context(), character.ResolveSculptSpellsCommand{ClassLevels: wzPick(mode, 6, 5, 6), Subclass: evkSub(t),
					Spell: evkEvoFact("s", 1), CastLevel: 1, Chosen: evkCreatures(2)})
			}},
		{name: "ResolveEmpoweredEvocation", span: character.SpanResolveEmpoweredEvocation, feature: "empowered_evocation", hasLevels: true, rejection: "feature_not_available",
			okType: "EmpoweredRoll", badType: "ResolveEmpoweredEvocationRejected",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return character.NewResolveEmpoweredEvocation(opts...).Run(t.Context(), character.ResolveEmpoweredEvocationCommand{ClassLevels: wzPick(mode, 10, 9, 10), Subclass: evkSub(t),
					Spell: evkEvoFact("s", 3), Intelligence: evkMod(16), Applies: true})
			}},
		{name: "UseOverchannel", span: character.SpanUseOverchannel, feature: "overchannel", hasLevels: true, rejection: "feature_not_available",
			okType: "OverchannelApplied", badType: "UseOverchannelRejected",
			run: func(t *testing.T, mode wzMode, _ character.ContentSource, opts ...character.Option) (any, error) {
				t.Helper()
				return must(character.NewUseOverchannel(dicetest.NewMaximumRoller(), opts...)).Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzPick(mode, 14, 13, 14),
					Subclass: evkSub(t), Resources: evkResources(t, 1), Request: character.OverchannelRequest{Spell: wzFact("s", 3, evkDmg), Source: character.CastSourceSlot, SlotLevel: 3}})
			}},
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-018 - every per-cast Evoker use case emits one span, counter and duration with srd5e.class wizard and its feature: success with one FINE log, a rule refusal as failure with
// the snake_case rejection, NotAWizard as failure with rejection not_a_wizard; none loads content, so none has a load_content child.
// Path: per case (happy / unhappy: state error).
func TestEvokerCastSignals(t *testing.T) {
	t.Parallel()
	for _, c := range evkCastCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, m := range []struct {
				mode              wzMode
				result, rejection string
			}{
				{wzOK, c.okType, ""},
				{wzBad, c.badType, c.rejection},
				{wzNotWizard, c.name + "NotAWizard", "not_a_wizard"},
			} {
				rec, opts := wzTelemetry()
				res, err := c.run(t, m.mode, nil, opts...)
				if err != nil || typeName(res) != m.result {
					t.Fatalf("mode %d = %T, %v; want %s", m.mode, res, err, m.result)
				}
				want := wzWant{span: c.span, feature: c.feature, outcome: "success", status: telemetry.StatusUnset, logged: true}
				if m.rejection != "" {
					want.outcome, want.rejection = "failure", m.rejection
				}
				wzCheck(t, rec, want)
			}
		})
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-018, class-wizard.md FR-091, FR-096 - the per-cast Evoker use cases give identical results with the no-op and a recording telemetry.
// Path: per case (unhappy: determinism).
func TestEvokerCastUseCasesAreDeterministic(t *testing.T) {
	t.Parallel()
	for _, c := range evkCastCases() {
		for _, mode := range []wzMode{wzOK, wzBad, wzNotWizard} {
			plain, err1 := c.run(t, mode, nil)
			_, opts := wzTelemetry()
			recorded, err2 := c.run(t, mode, nil, opts...)
			if err1 != nil || err2 != nil || !reflect.DeepEqual(plain, recorded) {
				t.Errorf("%s mode %d: %#v (%v) versus %#v (%v)", c.name, mode, plain, err1, recorded, err2)
			}
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-005, FR-006, FR-018, section 4.2.1 rows PCa-01 to PCa-12 - through the use case a level 3 Evoker's missed cantrip does half damage rounded down, a hit or a failed save is accepted
// unchanged (a success, not a rejection), and the refusals of PCa-05 to PCa-08 and PCa-12 come back as ResolvePotentCantripRejected.
// Path: per case (happy / unhappy: boundary, rule interaction, invalid input).
func TestResolvePotentCantripUseCase(t *testing.T) {
	t.Parallel()
	cantrip := wzFact("fire_bolt", 0, evkEvo, evkDmg)
	uc := character.NewResolvePotentCantrip()
	for _, c := range []struct {
		label   string
		levels  character.ClassLevels
		sub     bool
		spell   character.WizardSpellFacts
		kind    d20.Kind
		outcome d20.Outcome
		damage  int
		half    int // -1: unchanged; -2: rejected
		kindRej character.PotentCantripRejectionKind
	}{
		{"PCa-01", wzLevels(3), true, cantrip, d20.AttackRoll, d20.Failure, 8, 4, 0},
		{"PCa-02", wzLevels(3), true, cantrip, d20.SavingThrow, d20.Success, 6, 3, 0},
		{"PCa-03", wzLevels(3), true, cantrip, d20.AttackRoll, d20.Success, 8, -1, 0},
		{"PCa-04", wzLevels(3), true, cantrip, d20.SavingThrow, d20.Failure, 6, -1, 0},
		{"PCa-05", wzLevels(3), true, wzFact("c", 0), d20.SavingThrow, d20.Success, 0, -2, character.PotentCantripSpellDealsNoDamage},
		{"PCa-06", wzLevels(2), true, cantrip, d20.AttackRoll, d20.Failure, 8, -2, character.PotentCantripFeatureNotAvailable},
		{"PCa-07", wzLevels(3), true, wzFact("s", 1, evkDmg), d20.AttackRoll, d20.Failure, 8, -2, character.PotentCantripNotACantrip},
		{"PCa-08", wzLevels(3), false, cantrip, d20.AttackRoll, d20.Failure, 8, -2, character.PotentCantripNotEvoker},
		{"PCa-09", wzLevels(3), true, cantrip, d20.AttackRoll, d20.Failure, 7, 3, 0},
		{"PCa-10", wzLevels(3), true, cantrip, d20.SavingThrow, d20.Success, 7, 3, 0},
		{"PCa-11", wzLevels(3), true, cantrip, d20.AttackRoll, d20.Failure, 1, 0, 0},
		{"PCa-12", wzLevels(3), true, cantrip, d20.AbilityCheck, d20.Failure, 8, -2, character.PotentCantripUnsupportedTestKind},
	} {
		cmd := character.ResolvePotentCantripCommand{ClassLevels: c.levels, Spell: c.spell, Kind: c.kind, Outcome: c.outcome, Damage: c.damage}
		if c.sub {
			cmd.Subclass = evkSub(t)
		}
		res, err := uc.Run(t.Context(), cmd)
		if err != nil {
			t.Fatalf("%s: %v", c.label, err)
		}
		switch got := res.(type) {
		case character.PotentCantripHalfDamage:
			if got.Damage() != c.half {
				t.Errorf("%s: half damage %d, want %d", c.label, got.Damage(), c.half)
			}
		case character.PotentCantripUnchanged:
			if c.half != -1 {
				t.Errorf("%s: unchanged, want %d", c.label, c.half)
			}
		case character.ResolvePotentCantripRejected:
			if c.half != -2 || got.Rejection.Kind != c.kindRej {
				t.Errorf("%s: rejected %+v", c.label, got.Rejection)
			}
		default:
			t.Errorf("%s: %T", c.label, res)
		}
	}
	for name, cmd := range map[string]character.ResolvePotentCantripCommand{
		"invalid kind":    {ClassLevels: wzLevels(3), Spell: cantrip, Kind: 0, Outcome: d20.Failure},
		"invalid outcome": {ClassLevels: wzLevels(3), Spell: cantrip, Kind: d20.AttackRoll},
		"negative damage": {ClassLevels: wzLevels(3), Spell: cantrip, Kind: d20.AttackRoll, Outcome: d20.Failure, Damage: -1},
		"no spell facts":  {ClassLevels: wzLevels(3), Kind: d20.AttackRoll, Outcome: d20.Failure},
	} {
		if _, err := uc.Run(t.Context(), cmd); !srderrIsInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (character.ResolvePotentCantrip{}).Run(t.Context(), character.ResolvePotentCantripCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := uc.Run(cancelled, character.ResolvePotentCantripCommand{ClassLevels: wzLevels(3), Spell: cantrip, Kind: d20.AttackRoll, Outcome: d20.Failure}); err == nil {
		t.Error("a cancelled context was accepted")
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-007, FR-008, FR-015, FR-018, section 4.3.1 - through the use case a level 6 Evoker protects up to 1 plus the cast level of its visible, affected creatures (an empty choice and a
// higher-level slot included) and the refusals come back as ResolveSculptSpellsRejected; a cast level the spell cannot have and a zero or repeated creature id are input errors.
// Path: per case (happy / unhappy: boundary, invalid input).
func TestResolveSculptSpellsUseCase(t *testing.T) {
	t.Parallel()
	uc := character.NewResolveSculptSpells()
	run := func(level int, spell character.WizardSpellFacts, cast int, chosen []character.SculptCandidate) (character.ResolveSculptSpellsResult, error) {
		return uc.Run(t.Context(), character.ResolveSculptSpellsCommand{ClassLevels: wzLevels(level), Subclass: evkSub(t), Spell: spell, CastLevel: cast, Chosen: chosen})
	}
	for _, c := range []struct {
		label  string
		spell  character.WizardSpellFacts
		cast   int
		chosen int
		allow  int
	}{
		{"SC-01", evkEvoFact("s", 1), 1, 2, 2},
		{"SC-04", evkEvoFact("s", 9), 9, 10, 10},
		{"SC-05", evkEvoFact("s", 0), 0, 1, 1},
		{"SC-06", evkEvoFact("s", 3), 3, 0, 4},
		{"SC-11", evkEvoFact("s", 1), 3, 4, 4},
	} {
		res, err := run(6, c.spell, c.cast, evkCreatures(c.chosen))
		if got, ok := res.(character.SculptedCreatures); err != nil || !ok || len(got.Creatures()) != c.chosen || got.Allowed() != c.allow {
			t.Errorf("%s = %#v, %v", c.label, res, err)
		}
	}
	res, _ := run(6, evkEvoFact("s", 1), 1, evkCreatures(3))
	if r, ok := res.(character.ResolveSculptSpellsRejected); !ok || r.Rejection != (character.SculptRejection{Kind: character.SculptTooManyCreatures, Allowed: 2}) {
		t.Errorf("SC-02 = %#v", res)
	}
	for name, c := range map[string]struct {
		spell  character.WizardSpellFacts
		cast   int
		chosen []character.SculptCandidate
	}{
		"cast below the spell level": {evkEvoFact("s", 3), 2, nil},
		"cantrip cast above 0":       {evkEvoFact("s", 0), 1, nil},
		"cast above 9":               {evkEvoFact("s", 3), 10, nil},
		"zero creature id":           {evkEvoFact("s", 3), 3, []character.SculptCandidate{{Visible: true, Affected: true}}},
		"repeated creature":          {evkEvoFact("s", 3), 3, append(evkCreatures(1), evkCreatures(1)...)},
		"no spell facts":             {character.WizardSpellFacts{}, 0, nil},
	} {
		if _, err := run(6, c.spell, c.cast, c.chosen); !srderrIsInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := (character.ResolveSculptSpells{}).Run(t.Context(), character.ResolveSculptSpellsCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := uc.Run(cancelled, character.ResolveSculptSpellsCommand{ClassLevels: wzLevels(6), Spell: evkEvoFact("s", 3), CastLevel: 3}); err == nil {
		t.Error("a cancelled context was accepted")
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-009, FR-010, FR-016, FR-018, section 4.4.1 rows EE-01 to EE-08 - through the use case a level 10 Evoker adds its Intelligence modifier to one damage roll (a negative one
// as it is), the returned cast state refuses a second roll with AlreadyEmpowered, and declining returns no modifier and the state unchanged.
// Path: per case (happy: boundary; unhappy: rule interaction, state error).
func TestResolveEmpoweredEvocationUseCase(t *testing.T) {
	t.Parallel()
	uc := character.NewResolveEmpoweredEvocation()
	run := func(level int, spell character.WizardSpellFacts, score int, applies bool, state character.EmpoweredCastState) character.ResolveEmpoweredEvocationResult {
		res, err := uc.Run(t.Context(), character.ResolveEmpoweredEvocationCommand{ClassLevels: wzLevels(level), Subclass: evkSub(t), Spell: spell, Intelligence: evkMod(score), Applies: applies, State: state})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	for _, c := range []struct {
		label        string
		level, score int
		want         int
	}{{"EE-01", 10, 16, 3}, {"EE-02", 20, 20, 5}, {"EE-03", 10, 10, 0}, {"EE-04", 10, 8, -1}} {
		got, ok := run(c.level, evkEvoFact("s", 3), c.score, true, character.EmpoweredCastState{}).(character.EmpoweredRoll)
		mod, applied := got.Modifier()
		if !ok || !applied || mod.Value() != c.want || !got.State().DamageRollEmpowered() {
			t.Errorf("%s = %#v", c.label, got)
		}
	}
	first := run(10, evkEvoFact("s", 3), 16, true, character.EmpoweredCastState{}).(character.EmpoweredRoll)
	second := run(10, evkEvoFact("s", 3), 16, true, first.State())
	if r, ok := second.(character.ResolveEmpoweredEvocationRejected); !ok || r.Rejection.Kind != character.EmpoweredAlreadyEmpowered {
		t.Errorf("EE-05 = %#v", second)
	}
	if r, ok := run(10, evkEvoFact("s", 3, evkAbj), 16, true, character.EmpoweredCastState{}).(character.ResolveEmpoweredEvocationRejected); !ok || r.Rejection.Kind != character.EmpoweredNotAnEvocationSpell {
		t.Errorf("EE-06 = %#v", r)
	}
	declined := run(10, evkEvoFact("s", 3), 16, false, character.EmpoweredCastState{}).(character.EmpoweredRoll)
	if _, applied := declined.Modifier(); applied || declined.State().DamageRollEmpowered() {
		t.Errorf("EE-08 = %#v", declined)
	}
	if _, err := (character.ResolveEmpoweredEvocation{}).Run(t.Context(), character.ResolveEmpoweredEvocationCommand{}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := uc.Run(t.Context(), character.ResolveEmpoweredEvocationCommand{ClassLevels: wzLevels(10)}); !srderrIsInvalid(err) {
		t.Errorf("no spell facts: %v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := uc.Run(cancelled, character.ResolveEmpoweredEvocationCommand{ClassLevels: wzLevels(10), Spell: evkEvoFact("s", 3)}); err == nil {
		t.Error("a cancelled context was accepted")
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-011 to FR-014, FR-017, FR-018, section 4.5.1 rows OC-01 to OC-12 - through the use case the first use rolls nothing, the n-th rolls n d12 per slot level through the roller (identical
// with the minimum roller, all 1s, and the maximum roller, all 12s) and returns the resources with the count raised; a Long Rest resets the count and a Short Rest does not; a rejection rolls nothing and is not a use.
// Path: per case (happy: boundary, state; unhappy: boundary, determinism).
func TestUseOverchannel(t *testing.T) {
	t.Parallel()
	req := func(slot int, mods ...func(*character.WizardSpellFacts)) character.OverchannelRequest {
		return character.OverchannelRequest{Spell: wzFact("s", max(slot, 1), append([]func(*character.WizardSpellFacts){evkDmg}, mods...)...), Source: character.CastSourceSlot, SlotLevel: slot}
	}
	run := func(roller character.Roller, level, uses int, r character.OverchannelRequest) character.UseOverchannelResult {
		res, err := must(character.NewUseOverchannel(roller)).Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzLevels(level), Subclass: evkSub(t),
			Resources: evkResources(t, uses), Request: r})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	for _, c := range []struct {
		label                   string
		level, slot, uses, dice int
	}{
		{"OC-01", 14, 3, 0, 0}, {"OC-02", 14, 3, 1, 6}, {"OC-03", 14, 1, 2, 3}, {"OC-04", 14, 5, 1, 10}, {"OC-05", 20, 2, 3, 8},
	} {
		for name, roller := range map[string]*dicetest.FixedRoller{"min": dicetest.NewMinimumRoller(), "max": dicetest.NewMaximumRoller()} {
			applied, ok := run(roller, c.level, c.uses, req(c.slot)).(character.OverchannelApplied)
			if !ok || applied.Use() != c.uses+1 || applied.SlotLevel() != c.slot || !applied.MaximumDamage() || applied.Resources().OverchannelUses() != c.uses+1 || len(applied.Events()) != 1 {
				t.Fatalf("%s/%s = %#v", c.label, name, applied)
			}
			necrotic, rolled := applied.Necrotic()
			if c.dice == 0 {
				if rolled || len(roller.Requests()) != 0 {
					t.Errorf("%s/%s: the first use rolled %v", c.label, name, roller.Requests())
				}
				continue
			}
			want := c.dice
			if name == "max" {
				want = c.dice * 12
			}
			if !rolled || necrotic.Expression().Notation() != dice.MustExpression(c.dice, dice.D12(), 0).Notation() || necrotic.Total() != want || len(roller.Requests()) != 1 {
				t.Errorf("%s/%s: Necrotic = %v %v, requests %v; want total %d", c.label, name, necrotic, rolled, roller.Requests(), want)
			}
		}
	}
	// OC-06 and OC-07: a Long Rest resets the count, a Short Rest does not.
	held := evkResources(t, 2)
	if held.AfterRest(character.LongRest).OverchannelUses() != 0 || held.AfterRest(character.ShortRest).OverchannelUses() != 2 {
		t.Error("rest handling of the Overchannel count changed")
	}
	for _, c := range []struct {
		label string
		level int
		r     character.OverchannelRequest
		want  character.OverchannelRejection
	}{
		{"OC-08", 14, req(6), character.OverchannelRejection{Kind: character.OverchannelSlotLevelOutOfRange, Level: 6}},
		{"OC-09", 14, character.OverchannelRequest{Spell: wzFact("s", 3), Source: character.CastSourceSlot, SlotLevel: 3}, character.OverchannelRejection{Kind: character.OverchannelSpellDealsNoDamage}},
		{"OC-10", 14, character.OverchannelRequest{Spell: wzFact("c", 0, evkDmg), Source: character.CastSourceCantrip}, character.OverchannelRejection{Kind: character.OverchannelNoSpellSlotUsed}},
		{"OC-11", 13, req(3), character.OverchannelRejection{Kind: character.OverchannelFeatureNotAvailable}},
		{"OC-12", 14, req(3, wzOffList), character.OverchannelRejection{Kind: character.OverchannelNotAWizardSpell}},
	} {
		roller := dicetest.NewMaximumRoller()
		if r, ok := run(roller, c.level, 2, c.r).(character.UseOverchannelRejected); !ok || r.Rejection != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s = %#v, rolls %v", c.label, r, roller.Requests())
		}
	}
}

// Requirement: doc/specs/class-wizard-evoker.md FR-018, FR-014 - a failing roller is propagated as its *srderr.PortError, recorded once (error status, one errors increment, no log) and the count is not raised; the constructor
// rejects a nil roller, the zero use case and zero CharacterID are input errors, and use numbers so high that the dice pass dice.MaxCount are an input error, not a panic.
// Path: per case (unhappy: collaborator failure, invalid input).
func TestUseOverchannelFailures(t *testing.T) {
	t.Parallel()
	req := character.OverchannelRequest{Spell: wzFact("s", 3, evkDmg), Source: character.CastSourceSlot, SlotLevel: 3}
	failing := dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, errBoom })
	rec, opts := wzTelemetry()
	res, err := must(character.NewUseOverchannel(failing, opts...)).Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzLevels(14), Subclass: evkSub(t), Resources: evkResources(t, 1), Request: req})
	var pe *srderr.PortError
	if res != nil || !asPortError(err, &pe) || pe.Port != dice.Port {
		t.Fatalf("a failing roller = %v, %v; want a dice PortError", res, err)
	}
	wzCheck(t, rec, wzWant{span: character.SpanUseOverchannel, feature: "overchannel", outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err)})
	if _, err := character.NewUseOverchannel(nil); !srderrIsInvalid(err) {
		t.Errorf("nil roller: %v", err)
	}
	uc := must(character.NewUseOverchannel(dicetest.NewMaximumRoller()))
	if _, err := (character.UseOverchannel{}).Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID}); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
	if _, err := uc.Run(t.Context(), character.UseOverchannelCommand{ClassLevels: wzLevels(14), Request: req}); !srderrIsInvalid(err) {
		t.Errorf("zero id: %v", err)
	}
	if _, err := uc.Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzLevels(14), Request: character.OverchannelRequest{}}); !srderrIsInvalid(err) {
		t.Errorf("no spell facts: %v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := uc.Run(cancelled, character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzLevels(14), Request: req}); err == nil {
		t.Error("a cancelled context was accepted")
	}
	rec, opts = wzTelemetry()
	res, err = must(character.NewUseOverchannel(dicetest.NewMaximumRoller(), opts...)).Run(t.Context(), character.UseOverchannelCommand{CharacterID: wzID, ClassLevels: wzLevels(14), Subclass: evkSub(t),
		Resources: evkResources(t, 400), Request: character.OverchannelRequest{Spell: wzFact("s", 5, evkDmg), Source: character.CastSourceSlot, SlotLevel: 5}})
	if res != nil || !srderrIsInvalid(err) {
		t.Errorf("use 401 at slot 5 = %v, %v; want an input error", res, err)
	}
	if spans := rec.Tracer.SpansNamed(character.SpanUseOverchannel); len(spans) != 1 || spans[0].Status != telemetry.StatusError {
		t.Errorf("spans = %+v, want one with error status", spans)
	}
}
