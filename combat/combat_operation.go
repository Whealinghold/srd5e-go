package combat

import "strings"

// combatOperation names a use case: the snake_case value of
// srd5e.combat.operation and its span name srd5e.combat.<value>. The zero value
// is not an operation.
type combatOperation struct{ value, spanName string }

// newCombatOperation returns the operation value with span name spanName. It
// returns an error matching srderr.ErrInvalidInput when value is not
// [a-z][a-z0-9_]*, or spanName is not spanPrefix plus ending in "." and value.
func newCombatOperation(value, spanName string) (combatOperation, error) {
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		return combatOperation{}, invalidInput("operation must be snake_case")
	}
	for _, c := range value {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return combatOperation{}, invalidInput("operation must be snake_case")
		}
	}
	if !strings.HasPrefix(spanName, spanPrefix) || !strings.HasSuffix(spanName, "."+value) {
		return combatOperation{}, invalidInput("span name must be srd5e.combat.<operation>")
	}
	return combatOperation{value: value, spanName: spanName}, nil
}

// Value returns the snake_case operation name.
func (o combatOperation) Value() string { return o.value }

// SpanName returns the span name, used as given.
func (o combatOperation) SpanName() string { return o.spanName }
