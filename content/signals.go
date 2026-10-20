package content

// The signals Loader emits, defined once. The names and attribute keys are a
// public contract: renaming one breaks consumers' dashboards and alerts. They
// are catalogued in doc/observability/signals.md. The shared keys
// telemetry.AttrOutcome and telemetry.AttrErrorType are used as they are.
const (
	// SpanLoad wraps one LoadSet call.
	SpanLoad = "srd5e.content.load"
	// SpanDecode wraps the call to the Decoder port, inside SpanLoad.
	SpanDecode = "srd5e.content.decode"
	// MetricLoads counts LoadSet calls that return a result or a port failure
	// ({load}), by outcome.
	MetricLoads = "srd5e.content.loads"
	// MetricLoadDuration records LoadSet durations (s), port failures
	// included, with no attributes.
	MetricLoadDuration = "srd5e.content.load_duration"
	// MetricEntriesLoaded counts the entries of a loaded set ({entry}), by
	// kind; a kind with no entries is not recorded.
	MetricEntriesLoaded = "srd5e.content.entries_loaded"
	// MetricDecodeErrors counts the problems of a rejected document
	// ({problem}), by problem kind.
	MetricDecodeErrors = "srd5e.content.decode_errors"
)

// Attribute keys.
const (
	// AttrProblemKind is the kind of a problem: a ValueProblem* value. It is
	// never error.type, which is reserved for port failures.
	AttrProblemKind = "srd5e.content.problem_kind"
	// AttrKind is the kind of a content entry: a ValueKind* value.
	AttrKind = "srd5e.content.kind"
)

// Values of AttrProblemKind, the snake_case of each ProblemKind.
const (
	ValueProblemSyntax              = "syntax"
	ValueProblemUnsupportedVersion  = "unsupported_version"
	ValueProblemMissingField        = "missing_field"
	ValueProblemUnknownField        = "unknown_field"
	ValueProblemWrongType           = "wrong_type"
	ValueProblemOutOfRange          = "out_of_range"
	ValueProblemInvalidValue        = "invalid_value"
	ValueProblemDuplicate           = "duplicate"
	ValueProblemUnresolvedReference = "unresolved_reference"
	ValueProblemConstraintViolation = "constraint_violation"
)

// Values of AttrKind, the snake_case of each Kind.
const (
	ValueKindSpecies        = "species"
	ValueKindBackground     = "background"
	ValueKindCharacterClass = "character_class"
	ValueKindFeat           = "feat"
	ValueKindEquipment      = "equipment"
	ValueKindSpell          = "spell"
	ValueKindMagicItem      = "magic_item"
	ValueKindMonster        = "monster"
)

// problemKindValue returns the AttrProblemKind value of k.
func problemKindValue(k ProblemKind) string {
	switch k {
	case ProblemSyntax:
		return ValueProblemSyntax
	case ProblemUnsupportedVersion:
		return ValueProblemUnsupportedVersion
	case ProblemMissingField:
		return ValueProblemMissingField
	case ProblemUnknownField:
		return ValueProblemUnknownField
	case ProblemWrongType:
		return ValueProblemWrongType
	case ProblemOutOfRange:
		return ValueProblemOutOfRange
	case ProblemInvalidValue:
		return ValueProblemInvalidValue
	case ProblemDuplicate:
		return ValueProblemDuplicate
	case ProblemUnresolvedReference:
		return ValueProblemUnresolvedReference
	case ProblemConstraintViolation:
		return ValueProblemConstraintViolation
	}
	return unknownName
}

// kindValue returns the AttrKind value of k.
func kindValue(k Kind) string {
	switch k {
	case KindSpecies:
		return ValueKindSpecies
	case KindBackground:
		return ValueKindBackground
	case KindCharacterClass:
		return ValueKindCharacterClass
	case KindFeat:
		return ValueKindFeat
	case KindEquipment:
		return ValueKindEquipment
	case KindSpell:
		return ValueKindSpell
	case KindMagicItem:
		return ValueKindMagicItem
	case KindMonster:
		return ValueKindMonster
	}
	return unknownName
}
