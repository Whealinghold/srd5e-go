package content

import "testing"

// Requirement: observability.md signal catalogue (doc/observability/signals.md) -
// each ProblemKind and Kind maps to its own fixed attribute value.
// Path: per case (every member, plus an undefined value).
func TestSignalAttributeValues(t *testing.T) {
	problems := map[ProblemKind]string{
		ProblemSyntax:              "syntax",
		ProblemUnsupportedVersion:  "unsupported_version",
		ProblemMissingField:        "missing_field",
		ProblemUnknownField:        "unknown_field",
		ProblemWrongType:           "wrong_type",
		ProblemOutOfRange:          "out_of_range",
		ProblemInvalidValue:        "invalid_value",
		ProblemDuplicate:           "duplicate",
		ProblemUnresolvedReference: "unresolved_reference",
		ProblemConstraintViolation: "constraint_violation",
		ProblemKind(0):             "unknown",
		ProblemKind(99):            "unknown",
	}
	for k, want := range problems {
		if got := problemKindValue(k); got != want {
			t.Errorf("problemKindValue(%d) = %q, want %q", k, got, want)
		}
	}
	kinds := map[Kind]string{
		KindSpecies:        "species",
		KindBackground:     "background",
		KindCharacterClass: "character_class",
		KindFeat:           "feat",
		KindEquipment:      "equipment",
		KindSpell:          "spell",
		KindMagicItem:      "magic_item",
		KindMonster:        "monster",
		Kind(0):            "unknown",
		Kind(99):           "unknown",
	}
	for k, want := range kinds {
		if got := kindValue(k); got != want {
			t.Errorf("kindValue(%d) = %q, want %q", k, got, want)
		}
	}
}

// Requirement: observability.md signal catalogue - names and attribute keys are
// a public contract.
// Path: happy.
func TestSignalNames(t *testing.T) {
	for got, want := range map[string]string{
		SpanLoad:            "srd5e.content.load",
		SpanDecode:          "srd5e.content.decode",
		MetricLoads:         "srd5e.content.loads",
		MetricLoadDuration:  "srd5e.content.load_duration",
		MetricEntriesLoaded: "srd5e.content.entries_loaded",
		MetricDecodeErrors:  "srd5e.content.decode_errors",
		AttrProblemKind:     "srd5e.content.problem_kind",
		AttrKind:            "srd5e.content.kind",
	} {
		if got != want {
			t.Errorf("signal name %q, want %q", got, want)
		}
	}
}
