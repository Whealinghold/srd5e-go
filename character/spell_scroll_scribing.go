package character

import "slices"

// ScribeProblem is a prerequisite a scribe of a Spell Scroll lacks (SRD 5.2.1,
// "Prerequisites for the Scribe", p. 103). The zero value is invalid.
type ScribeProblem int

// The problems, in the fixed order AssessScribing lists them.
const (
	// NoProficiency: no proficiency in the Arcana skill or with Calligrapher's Supplies.
	NoProficiency ScribeProblem = iota + 1
	// SpellNotPrepared: the spell is not prepared (a known cantrip counts as prepared).
	SpellNotPrepared
	// MissingMaterials: a required Material component is not at hand.
	MissingMaterials
)

// String returns "no_proficiency", "spell_not_prepared", "missing_materials" or "unknown".
func (p ScribeProblem) String() string {
	names := [...]string{"no_proficiency", "spell_not_prepared", "missing_materials"}
	if p < NoProficiency || p > MissingMaterials {
		return "unknown"
	}
	return names[p-1]
}

// ScribeFacts are the facts AssessScribing reads. Construct it with a keyed literal.
// The host reports them; the zero value is a scribe who meets nothing.
type ScribeFacts struct {
	// ProficientInArcana is whether the scribe has proficiency in the Arcana skill.
	ProficientInArcana bool
	// ProficientWithCalligraphersSupplies is whether the scribe has proficiency with Calligrapher's Supplies.
	ProficientWithCalligraphersSupplies bool
	// SpellPrepared is whether the scribe has the spell prepared on each day of the
	// inscription; the host passes true for a known cantrip.
	SpellPrepared bool
	// MaterialsRequired is whether the spell has a Material component.
	MaterialsRequired bool
	// MaterialsAtHand is whether the scribe has those components at hand.
	MaterialsAtHand bool
	// MaterialsConsumed is whether the spell consumes its Material components.
	MaterialsConsumed bool
}

// ScribeAssessmentResult is the closed set of outcomes of AssessScribing. Variants:
// ScribeEligible, ScribeIneligible.
type ScribeAssessmentResult interface{ isScribeAssessmentResult() }

// ScribeEligible: the scribe meets every prerequisite.
type ScribeEligible struct{ consumes bool }

// ConsumesMaterialsOnCompletion reports whether the spell's Material components are
// consumed, which happens only on completion of the scroll.
func (r ScribeEligible) ConsumesMaterialsOnCompletion() bool { return r.consumes }

// ScribeIneligible: the scribe lacks at least one prerequisite.
type ScribeIneligible struct{ problems []ScribeProblem }

// Problems returns a new slice of every problem, in the order NoProficiency,
// SpellNotPrepared, MissingMaterials.
func (r ScribeIneligible) Problems() []ScribeProblem { return slices.Clone(r.problems) }

func (ScribeEligible) isScribeAssessmentResult()   {}
func (ScribeIneligible) isScribeAssessmentResult() {}

// AssessScribing decides whether a character may scribe a Spell Scroll (SRD 5.2.1,
// "Prerequisites for the Scribe", p. 103): proficiency in Arcana or with
// Calligrapher's Supplies, the spell prepared, and any Material components at hand.
// It lists every problem. It mutates nothing and never panics.
func AssessScribing(f ScribeFacts) ScribeAssessmentResult {
	var problems []ScribeProblem
	if !f.ProficientInArcana && !f.ProficientWithCalligraphersSupplies {
		problems = append(problems, NoProficiency)
	}
	if !f.SpellPrepared {
		problems = append(problems, SpellNotPrepared)
	}
	if f.MaterialsRequired && !f.MaterialsAtHand {
		problems = append(problems, MissingMaterials)
	}
	if len(problems) > 0 {
		return ScribeIneligible{problems: problems}
	}
	return ScribeEligible{consumes: f.MaterialsRequired && f.MaterialsConsumed}
}
