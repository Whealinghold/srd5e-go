package content

// ConditionKind is one of the 15 conditions, in Glossary order (SRD 5.2.1,
// "Condition", p. 179). The declaration order is load-bearing: other contexts
// order by it. It carries no effect. The zero ConditionKind is invalid.
// Comparable and safe for concurrent use.
type ConditionKind int

// The conditions.
const (
	ConditionBlinded ConditionKind = iota + 1
	ConditionCharmed
	ConditionDeafened
	ConditionExhaustion
	ConditionFrightened
	ConditionGrappled
	ConditionIncapacitated
	ConditionInvisible
	ConditionParalyzed
	ConditionPetrified
	ConditionPoisoned
	ConditionProne
	ConditionRestrained
	ConditionStunned
	ConditionUnconscious
)

func conditionKindNames() []string {
	return []string{
		"Blinded", "Charmed", "Deafened", "Exhaustion", "Frightened", "Grappled",
		"Incapacitated", "Invisible", "Paralyzed", "Petrified", "Poisoned", "Prone",
		"Restrained", "Stunned", "Unconscious",
	}
}

// Valid reports whether c is one of the 15 conditions.
func (c ConditionKind) Valid() bool { return enumValid(conditionKindNames(), int(c)) }

// String returns the printed name of c, or "unknown" for an undefined value.
func (c ConditionKind) String() string { return enumString(conditionKindNames(), int(c)) }

// ParseConditionKind returns the condition spelled exactly s, such as
// "Frightened". Matching is case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseConditionKind(s string) (ConditionKind, error) {
	v, err := parseEnum("ConditionKind", conditionKindNames(), s)
	return ConditionKind(v), err
}
