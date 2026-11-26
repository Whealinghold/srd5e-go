package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// dspDC returns the spell save DC of a Wizard with a +3 modifier, a Proficiency Bonus of +2.
func dspDC(t *testing.T) character.SpellSaveDC {
	t.Helper()
	lvl, err := character.NewLevel(3)
	if err != nil {
		t.Fatal(err)
	}
	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(lvl))
	return stats.SaveDC()
}

// dspObservation builds a cast of fireball by wizard from source, with the given
// components and casting time.
func dspObservation(t *testing.T, source character.CastSource, used combat.UsedComponents, concentration bool, ct content.CastingTime) combat.CastObservation {
	t.Helper()
	obs, err := combat.NewCastObservation(content.SpellID("fireball"), dspCreature(t, "wizard"), source, used, concentration, ct)
	if err != nil {
		t.Fatal(err)
	}
	return obs
}

// dspEligible assesses a cast of fireball by wizard that a goblin can counter, and
// returns the eligible assessment.
func dspEligible(t *testing.T, source character.CastSource, concentration bool, ct content.CastingTime) combat.CounterspellEligible {
	t.Helper()
	obs := dspObservation(t, source, combat.NewUsedComponents(true, true, false), concentration, ct)
	a, err := combat.Counterspell{}.Assess(obs, dspCreature(t, "goblin"), 30, true, dspDC(t))
	if err != nil {
		t.Fatal(err)
	}
	e, ok := a.(combat.CounterspellEligible)
	if !ok {
		t.Fatalf("Assess = %T, want CounterspellEligible", a)
	}
	return e
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031 and table T-C - Assess returns eligible or ineligible with the first refusal, one row per case (SRD 5.2.1, "Counterspell" (Casting Time, Range), p. 120).
// Path: per case.
func TestCounterspellAssessTableT_C(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		sees                    bool
		distance                int
		v, s, m                 bool
		want                    string
	}{
		{"all three present", "FR-031", "happy", true, 30, true, true, true, "eligible"},
		{"only verbal at 60 feet", "FR-031", "unhappy (boundary)", true, 60, true, false, false, "eligible"},
		{"only somatic", "FR-031", "happy", true, 10, false, true, false, "eligible"},
		{"only material", "FR-031", "happy", true, 10, false, false, true, "eligible"},
		{"61 feet", "FR-031", "unhappy (boundary)", true, 61, true, false, false, "out_of_range"},
		{"caster not seen", "FR-031", "unhappy (rule interaction)", false, 10, true, false, false, "caster_not_seen"},
		{"no components (Subtle Spell)", "FR-032", "unhappy (rule interaction)", true, 10, false, false, false, "no_components"},
		{"not seen, far and no components", "FR-031", "unhappy (rule interaction)", false, 90, false, false, false, "caster_not_seen"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			obs := dspObservation(t, character.CastSourceSlot, combat.NewUsedComponents(c.v, c.s, c.m), false, content.NewCastingTimeMagicAction())
			a, err := combat.Counterspell{}.Assess(obs, dspCreature(t, "goblin"), c.distance, c.sees, dspDC(t))
			if err != nil {
				t.Fatalf("Assess error = %v [%s]", err, c.requirement)
			}
			if got := assessName(t, a); got != c.want {
				t.Errorf("Assess = %s, want %s [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// assessName names an assessment: eligible, or the refusal of an ineligible one.
func assessName(t *testing.T, a combat.CounterspellAssessment) string {
	t.Helper()
	switch v := a.(type) {
	case combat.CounterspellEligible:
		return "eligible"
	case combat.CounterspellIneligible:
		return v.Refusal().String()
	default:
		t.Fatalf("assessment %T is not a known variant", a)
		return ""
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031 - an eligible assessment carries the counterspeller's spell save DC, which the caster's save targets (FR-041).
// Path: happy.
func TestCounterspellEligibleCarriesTheDC(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	if e.Dc() != dspDC(t) {
		t.Errorf("Dc() = %v, want the counterspeller's save DC [FR-031]", e.Dc())
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031 - a negative distance and a zero counterspeller are refused (invalid input, distanceFeet at least 0).
// Path: unhappy (invalid input).
func TestCounterspellAssessRefusesInvalidInput(t *testing.T) {
	t.Parallel()
	obs := dspObservation(t, character.CastSourceSlot, combat.NewUsedComponents(true, false, false), false, content.NewCastingTimeMagicAction())
	if _, err := (combat.Counterspell{}).Assess(obs, dspCreature(t, "goblin"), -1, true, dspDC(t)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("negative distance error = %v, want ErrInvalidInput [FR-031]", err)
	}
	if _, err := (combat.Counterspell{}).Assess(obs, combat.CombatantID(""), 10, true, dspDC(t)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero counterspeller error = %v, want ErrInvalidInput [FR-031]", err)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-034 - a failed save counters the cast and a success resists it; a save at the DC is a success (SRD 5.2.1, "Counterspell", p. 120; "Compare the Total to a Target Number", p. 6).
// Path: per case.
func TestCounterspellConcludeRolledSaveCountersOrResists(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	cases := []struct {
		name, requirement, path string
		delta                   int
		want                    string
	}{
		{"total one below the DC is countered", "FR-034", "unhappy (boundary)", -1, "countered"},
		{"total equal to the DC resists", "FR-033", "unhappy (boundary)", 0, "resisted"},
		{"high total resists", "FR-033", "happy", 7, "resisted"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			check := dspSavingThrow(t, e.Dc().Value()+c.delta, e.Dc().DifficultyClass())
			got := counterspellName(t, combat.Counterspell{}.Conclude(e, combat.NewCounterspellRolledSave(check)))
			if got != c.want {
				t.Errorf("Conclude = %s, want %s [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// counterspellName names a Conclude result for comparison.
func counterspellName(t *testing.T, r combat.CounterspellResult) string {
	t.Helper()
	switch v := r.(type) {
	case combat.CounterspellCountered:
		return "countered"
	case combat.CounterspellResistedResult:
		return "resisted"
	case combat.CounterspellInvalid:
		return "invalid:" + v.Problem().String()
	default:
		t.Fatalf("result %T is not a known variant", r)
		return ""
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-034 and FR-037 - only the slot is restored: a cast from a slot reports no expended slot, every other source reports one (SRD 5.2.1, "Counterspell", p. 120; interpretation Q-4).
// Path: per case.
func TestCounterspellCounteredSlotRuleBySource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		source                  character.CastSource
		want                    bool
	}{
		{"a slot is restored", "FR-034", "happy", character.CastSourceSlot, true},
		{"a cantrip expends nothing", "FR-037", "unhappy (rule interaction)", character.CastSourceCantrip, false},
		{"a ritual expends nothing", "FR-037", "unhappy (rule interaction)", character.CastSourceRitual, false},
		{"a feature-granted cast leaves the feature's use", "FR-037", "unhappy (rule interaction)", character.CastSourceFeatureGranted, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := dspEligible(t, c.source, false, content.NewCastingTimeMagicAction())
			countered := combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t))).(combat.CounterspellCountered)
			if countered.SlotNotExpended() != c.want {
				t.Errorf("SlotNotExpended = %v, want %v [%s]", countered.SlotNotExpended(), c.want, c.requirement)
			}
		})
	}
}

// dspForcedFailure returns a saving throw that fails by choice.
func dspForcedFailure(t *testing.T) d20.ForcedFailureResult {
	t.Helper()
	f, err := d20.FailWithoutRoll(d20.SavingThrow, d20.ChosenToFail)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-035 - the wasted action is the cast's action; a cast of a minute or longer wastes none (SRD 5.2.1, "Counterspell", p. 120).
// Path: per case.
func TestCounterspellWastedActionFollowsCastingTime(t *testing.T) {
	t.Parallel()
	bonus, err := content.NewCastingTimeBonusActionTriggered("a trigger")
	if err != nil {
		t.Fatal(err)
	}
	reaction, err := content.NewCastingTimeReaction("a trigger")
	if err != nil {
		t.Fatal(err)
	}
	minutes, err := content.NewCastingTimeMinutes(1)
	if err != nil {
		t.Fatal(err)
	}
	hours, err := content.NewCastingTimeHours(1)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, requirement, path string
		ct                      content.CastingTime
		want                    combat.WastedActionCost
	}{
		{"magic action", "FR-035", "happy", content.NewCastingTimeMagicAction(), combat.WastedMagicAction},
		{"bonus action", "FR-035", "happy", bonus, combat.WastedBonusAction},
		{"reaction", "FR-035", "happy", reaction, combat.WastedReaction},
		{"one minute wastes none", "FR-035", "unhappy (boundary)", minutes, combat.WastedNone},
		{"one hour wastes none", "FR-035", "unhappy (boundary)", hours, combat.WastedNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := dspEligible(t, character.CastSourceSlot, false, c.ct)
			got := combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t))).(combat.CounterspellCountered).Wasted()
			if got != c.want {
				t.Errorf("Wasted = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-036 - a countered Concentration spell still ends the caster's prior Concentration; any other spell does not (SRD 5.2.1, "Concentration" (Another Concentration Effect), p. 179).
// Path: per case.
func TestCounterspellPriorConcentrationOnlyForConcentrationSpells(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		concentration           bool
	}{
		{"a Concentration spell ends the prior Concentration", "FR-036", "unhappy (rule interaction)", true},
		{"another spell does not", "FR-036", "happy", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := dspEligible(t, character.CastSourceSlot, c.concentration, content.NewCastingTimeMagicAction())
			got := combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t))).(combat.CounterspellCountered).PriorConcentrationEnds()
			if got != c.concentration {
				t.Errorf("PriorConcentrationEnds = %v, want %v [%s]", got, c.concentration, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-041 - a rolled save that is not a saving throw, or that targets another DC, is invalid (invalid input; architecture.md section 6).
// Path: per case.
func TestCounterspellConcludeRefusesAWrongSave(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	cases := []struct {
		name, requirement, path string
		check                   d20.CheckResult
		want                    string
	}{
		{"an ability check, not a saving throw", "FR-041", "unhappy (invalid input)", dspCheck(t, 5, 0, e.Dc().DifficultyClass()), "invalid:not_a_saving_throw"},
		{"a save against another DC", "FR-041", "unhappy (invalid input)", dspSavingThrow(t, 5, e.Dc().DifficultyClass()+1), "invalid:save_target_mismatch"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := counterspellName(t, combat.Counterspell{}.Conclude(e, combat.NewCounterspellRolledSave(c.check)))
			if got != c.want {
				t.Errorf("Conclude = %s, want %s [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-041 - a forced failure that is not a saving throw is invalid (invalid input).
// Path: unhappy (invalid input).
func TestCounterspellConcludeRefusesAForcedNonSave(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	f, err := d20.FailWithoutRoll(d20.AbilityCheck, d20.Automatic)
	if err != nil {
		t.Fatal(err)
	}
	if got := counterspellName(t, combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(f))); got != "invalid:not_a_saving_throw" {
		t.Errorf("Conclude = %s, want invalid:not_a_saving_throw [FR-041]", got)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-042 - a countered cast gives one SpellCountered event, a resisted one a CounterspellResisted, both naming the counterspeller, the caster and the spell; Conclude is deterministic (SRD 5.2.1, "Counterspell", p. 120).
// Path: happy.
func TestCounterspellConcludeEventsNameTheParticipants(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	countered := combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t))).(combat.CounterspellCountered)
	again := combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t))).(combat.CounterspellCountered)
	ev := countered.Event()
	if ev.Kind() != "spell_countered" || !slices.Equal(ev.Combatants(), []combat.CombatantID{dspCreature(t, "goblin"), dspCreature(t, "wizard")}) || ev.SpellID() != "fireball" {
		t.Errorf("SpellCountered = %v %v %v [FR-042]", ev.Kind(), ev.Combatants(), ev.SpellID())
	}
	if again.Event() != ev {
		t.Error("Conclude is not deterministic [FR-042]")
	}
	check := dspSavingThrow(t, 20, e.Dc().DifficultyClass())
	resisted := combat.Counterspell{}.Conclude(e, combat.NewCounterspellRolledSave(check)).(combat.CounterspellResistedResult)
	rev := resisted.Event()
	if rev.Kind() != "counterspell_resisted" || !slices.Equal(rev.Combatants(), []combat.CombatantID{dspCreature(t, "goblin"), dspCreature(t, "wizard")}) || rev.SpellID() != "fireball" {
		t.Errorf("CounterspellResisted = %v %v %v [FR-042]", rev.Kind(), rev.Combatants(), rev.SpellID())
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031, FR-034 and FR-035 - a CastObservation refuses an invalid spell, caster, source or casting time (invalid input; the parts the assessment and the result read).
// Path: per case.
func TestNewCastObservationRefusesInvalidParts(t *testing.T) {
	t.Parallel()
	mag := content.NewCastingTimeMagicAction()
	cases := []struct {
		name, requirement, path string
		spell                   content.SpellID
		caster                  combat.CombatantID
		source                  character.CastSource
		ct                      content.CastingTime
	}{
		{"spell id not valid", "FR-031", "unhappy (invalid input)", "Not Valid", dspCreature(t, "w"), character.CastSourceSlot, mag},
		{"zero caster", "FR-031", "unhappy (invalid input)", "fireball", "", character.CastSourceSlot, mag},
		{"undefined cast source", "FR-034", "unhappy (invalid input)", "fireball", dspCreature(t, "w"), character.CastSource(0), mag},
		{"nil casting time", "FR-035", "unhappy (invalid input)", "fireball", dspCreature(t, "w"), character.CastSourceSlot, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewCastObservation(c.spell, c.caster, c.source, combat.NewUsedComponents(true, false, false), false, c.ct)
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewCastObservation error = %v, want ErrInvalidInput [%s]", err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-033 - a forced failure counters the cast without a roll.
// Path: happy.
func TestCounterspellConcludeForcedFailureCounters(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	if got := counterspellName(t, combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t)))); got != "countered" {
		t.Errorf("Conclude = %s, want countered [FR-033]", got)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-039 - a Counterspell cast in response resolves first: its conclusion changes neither the original cast's eligibility nor its DC, so the host's second call decides the original (SRD 5.2.1, "Counterspell", p. 120; interpretation Q-5).
// Path: unhappy (rule interaction).
func TestCounterspellInResponseResolvesFirst(t *testing.T) {
	t.Parallel()
	original := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	reaction, err := content.NewCastingTimeReaction("a trigger")
	if err != nil {
		t.Fatal(err)
	}
	obs, err := combat.NewCastObservation(content.SpellID("counterspell"), dspCreature(t, "goblin"), character.CastSourceSlot, combat.NewUsedComponents(true, true, false), false, reaction)
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := combat.Counterspell{}.Assess(obs, dspCreature(t, "priest"), 30, true, dspDC(t))
	if err != nil {
		t.Fatal(err)
	}
	inResponse, ok := assessed.(combat.CounterspellEligible)
	if !ok {
		t.Fatalf("Assess = %T, want CounterspellEligible [FR-039]", assessed)
	}
	// The first call: the goblin's Counterspell fails its save, so it is not cast.
	if got := counterspellName(t, combat.Counterspell{}.Conclude(inResponse, combat.NewCounterspellForcedFailure(dspForcedFailure(t)))); got != "countered" {
		t.Fatalf("first Conclude = %s, want countered [FR-039]", got)
	}
	// The second call: the original cast resolves on its own save, which the first call did not change.
	check := dspSavingThrow(t, original.Dc().Value()+7, original.Dc().DifficultyClass())
	if got := counterspellName(t, combat.Counterspell{}.Conclude(original, combat.NewCounterspellRolledSave(check))); got != "resisted" {
		t.Errorf("second Conclude = %s, want resisted [FR-039]", got)
	}
	if original.Observation().Spell() != "fireball" || original.Dc() != dspDC(t) {
		t.Error("the first call changed the original cast's observation or DC [FR-039]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-041 - a nil save, and a zero eligible assessment, conclude as CounterspellInvalid with CounterspellMissingInput and never panic.
// Path: unhappy (invalid input).
func TestCounterspellConcludeRefusesMissingInput(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	forced := combat.NewCounterspellForcedFailure(dspForcedFailure(t))
	for name, got := range map[string]combat.CounterspellResult{
		"nil save":        combat.Counterspell{}.Conclude(e, nil),
		"zero eligible":   combat.Counterspell{}.Conclude(combat.CounterspellEligible{}, forced),
		"both are absent": combat.Counterspell{}.Conclude(combat.CounterspellEligible{}, nil),
	} {
		if inv, ok := got.(combat.CounterspellInvalid); !ok || inv.Problem() != combat.CounterspellMissingInput || inv.Problem().String() != "missing_input" {
			t.Errorf("%s: Conclude = %#v, want CounterspellInvalid{CounterspellMissingInput}", name, got)
		}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031 - a CastObservation returns exactly the components, concentration flag and source it was built with, so the assessment reads what the host reported (and a monster source is accepted).
// Path: happy.
func TestCastObservationReturnsWhatItWasBuiltWith(t *testing.T) {
	t.Parallel()
	for _, source := range []character.CastSource{character.CastSourceSlot, character.CastSourceMonster} {
		used := combat.NewUsedComponents(true, false, true)
		obs, err := combat.NewCastObservation("fireball", dspCreature(t, "wizard"), source, used, true, content.NewCastingTimeMagicAction())
		if err != nil {
			t.Fatal(err)
		}
		if obs.Used() != used || !obs.RequiresConcentration() || obs.Source() != source || obs.Spell() != "fireball" {
			t.Errorf("observation = %+v, want the parts it was built with (source %v)", obs, source)
		}
	}
}
