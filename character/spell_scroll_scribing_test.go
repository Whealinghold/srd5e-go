package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// Requirement: per case (SRD 5.2.1, "Prerequisites for the Scribe", p. 103)
// Path: per case.
func TestAssessScribing(t *testing.T) {
	t.Parallel()
	const all = true
	cases := []struct {
		name, req, path string
		facts           character.ScribeFacts
		want            character.ScribeAssessmentResult
	}{
		{"B4 nothing met", "FR-027", "unhappy (invalid input)", character.ScribeFacts{MaterialsRequired: true},
			nil},
		{"B5 Calligrapher's Supplies only", "FR-027", "happy", character.ScribeFacts{ProficientWithCalligraphersSupplies: all, SpellPrepared: all}, character.ScribeEligible{}},
		{"Arcana only", "FR-027", "happy", character.ScribeFacts{ProficientInArcana: all, SpellPrepared: all}, character.ScribeEligible{}},
		{"materials at hand and consumed", "FR-027", "happy", character.ScribeFacts{ProficientInArcana: all, SpellPrepared: all, MaterialsRequired: all, MaterialsAtHand: all, MaterialsConsumed: all},
			eligibleConsuming(t)},
		{"consumes flag without a Material component", "FR-027", "unhappy (boundary)", character.ScribeFacts{ProficientInArcana: all, SpellPrepared: all, MaterialsConsumed: all}, character.ScribeEligible{}},
	}
	for _, c := range cases[1:] {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := character.AssessScribing(c.facts); !reflect.DeepEqual(got, c.want) {
				t.Errorf("AssessScribing = %#v, want %#v [%s %s]", got, c.want, c.req, c.path)
			}
		})
	}
	// B4: all three problems, in the fixed order.
	got := character.AssessScribing(cases[0].facts).(character.ScribeIneligible).Problems()
	if want := []character.ScribeProblem{character.NoProficiency, character.SpellNotPrepared, character.MissingMaterials}; !reflect.DeepEqual(got, want) {
		t.Errorf("B4 problems = %v, want %v", got, want)
	}
}

func eligibleConsuming(t *testing.T) character.ScribeAssessmentResult {
	t.Helper()
	r := character.AssessScribing(character.ScribeFacts{ProficientInArcana: true, SpellPrepared: true, MaterialsRequired: true, MaterialsAtHand: true, MaterialsConsumed: true})
	if !r.(character.ScribeEligible).ConsumesMaterialsOnCompletion() {
		t.Fatal("an eligible scribe with consumed materials reports none consumed")
	}
	return r
}

// Requirement: spell-scroll-use.md FR-027 - each problem is reported alone, and Problems returns a copy.
// Path: unhappy (boundary).
func TestAssessScribingEachProblemAlone(t *testing.T) {
	t.Parallel()
	ok := character.ScribeFacts{ProficientInArcana: true, SpellPrepared: true, MaterialsRequired: true, MaterialsAtHand: true}
	noProf, notPrepared, noMats := ok, ok, ok
	noProf.ProficientInArcana = false
	notPrepared.SpellPrepared = false
	noMats.MaterialsAtHand = false
	for facts, want := range map[character.ScribeFacts]character.ScribeProblem{noProf: character.NoProficiency, notPrepared: character.SpellNotPrepared, noMats: character.MissingMaterials} {
		r, isIneligible := character.AssessScribing(facts).(character.ScribeIneligible)
		if !isIneligible || !reflect.DeepEqual(r.Problems(), []character.ScribeProblem{want}) {
			t.Errorf("facts %+v: result %#v, want only %v", facts, r, want)
		}
		r.Problems()[0] = 0
		if r.Problems()[0] != want {
			t.Error("Problems() exposes the internal slice")
		}
	}
	if r := character.AssessScribing(ok).(character.ScribeEligible); r.ConsumesMaterialsOnCompletion() {
		t.Error("materials not consumed reported as consumed")
	}
}

// Requirement: spell-scroll-use.md FR-027 - ScribeProblem prints its name.
// Path: unhappy (boundary).
func TestScribeProblemString(t *testing.T) {
	t.Parallel()
	for p, want := range map[character.ScribeProblem]string{character.NoProficiency: "no_proficiency", character.SpellNotPrepared: "spell_not_prepared", character.MissingMaterials: "missing_materials", 0: "unknown", 4: "unknown"} {
		if p.String() != want {
			t.Errorf("%d.String() = %q, want %q", int(p), p.String(), want)
		}
	}
}
