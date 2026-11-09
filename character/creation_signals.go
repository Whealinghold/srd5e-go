package character

import "github.com/Whealinghold/srd5e-go/telemetry"

// Spans and metrics of the character creation use cases CreateCharacter,
// RollAbilityScores and RollLanguages. All three record on the shared
// MetricOperations counter and MetricDuration histogram; CreateCharacter looks
// content up in the child span SpanLoadContent and RollAbilityScores records
// MetricAbilityScoreRoll.
const (
	// SpanCreate wraps one CreateCharacter.Run call.
	SpanCreate = "srd5e.character.create"
	// SpanRollAbilityScores wraps one RollAbilityScores.Run call.
	SpanRollAbilityScores = "srd5e.character.roll_ability_scores"
	// SpanRollLanguages wraps one RollLanguages.Run call.
	SpanRollLanguages = "srd5e.character.roll_languages"
	// MetricAbilityScoreRoll records each of the six totals of a Random Generation roll ({score}).
	MetricAbilityScoreRoll = "srd5e.character.ability_score_roll"
	// AttrAbilityScoreMethod is the ability score generation method of a creation: a
	// ValueAbilityScoreMethod* value; the CreateCharacter span and counter only.
	AttrAbilityScoreMethod = "srd5e.character.ability_score_method"
)

// Attribute values of the creation use cases. The rejection values are the String
// of CreateCharacterProblemKind. ValueResultCreated, ValueResultRolled,
// ValueResultRejected and ValueResultContentMissing are shared.
const (
	ValueOperationCreate            = "create"
	ValueOperationRollAbilityScores = "roll_ability_scores"
	ValueOperationRollLanguages     = "roll_languages"

	ValueResultDuplicate = "duplicate"

	ValueAbilityScoreMethodStandardArray    = "standard_array"
	ValueAbilityScoreMethodRandomGeneration = "random_generation"
	ValueAbilityScoreMethodPointCost        = "point_cost"
)

// abilityScoreMethodValue maps a method to its ValueAbilityScoreMethod* value.
func abilityScoreMethodValue(m AbilityScoreGenerationMethod) string {
	switch m {
	case StandardArray:
		return ValueAbilityScoreMethodStandardArray
	case RandomGeneration:
		return ValueAbilityScoreMethodRandomGeneration
	case PointCost:
		return ValueAbilityScoreMethodPointCost
	}
	return "unknown"
}

// rollSignals are the instruments RollAbilityScores shares with the other
// character use cases, plus its totals histogram.
type rollSignals struct {
	operationSignals
	totals telemetry.Histogram
}

func newRollSignals(o options) rollSignals {
	return rollSignals{
		operationSignals: newOperationSignals(o),
		totals:           o.tel.Meter.Histogram(telemetry.Descriptor{Name: MetricAbilityScoreRoll, Unit: "{score}", Description: "Totals of the six ability score rolls of Random Generation."}),
	}
}
