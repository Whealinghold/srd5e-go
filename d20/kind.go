package d20

// Kind is the kind of a D20 Test ("D20 Test", SRD 5.2.1, p. 180). The zero
// value is not a valid kind.
type Kind int

// The kinds of D20 Test.
const (
	// AbilityCheck is a total compared with a Difficulty Class ("Ability
	// Checks", SRD 5.2.1, p. 6).
	AbilityCheck Kind = iota + 1
	// SavingThrow is a total compared with a Difficulty Class ("Saving
	// Throws", p. 7).
	SavingThrow
	// AttackRoll is a total compared with an Armor Class, with the natural
	// 20 and natural 1 rules of p. 7.
	AttackRoll
)

// Valid reports whether k is one of the three kinds.
func (k Kind) Valid() bool { return k >= AbilityCheck && k <= AttackRoll }

// String returns the value of the srd5e.d20_test.kind telemetry attribute for
// k, or "unknown" when k is not valid.
func (k Kind) String() string {
	switch k {
	case AbilityCheck:
		return ValueKindAbilityCheck
	case SavingThrow:
		return ValueKindSavingThrow
	case AttackRoll:
		return ValueKindAttackRoll
	default:
		return "unknown"
	}
}

// Outcome is the outcome of a D20 Test of every kind ("Compare the Total to a
// Target Number", SRD 5.2.1, p. 6). A Critical Hit is reported by
// [AttackRollResult.IsCriticalHit], not by a third outcome. The zero value is
// not a valid outcome.
type Outcome int

// The outcomes of a D20 Test.
const (
	// Success means the total equals or exceeds the target number.
	Success Outcome = iota + 1
	// Failure means the total is below the target number, or the test
	// failed without a roll.
	Failure
)

// String returns "success", "failure", or "unknown" for an invalid outcome.
// These are the srd5e.outcome telemetry values.
func (o Outcome) String() string {
	switch o {
	case Success:
		return "success"
	case Failure:
		return "failure"
	default:
		return "unknown"
	}
}

// ForcedFailureReason says why a check or saving throw fails without a roll.
// There is no reason for an attack roll: the SRD gives no rule for failing
// one without rolling. The zero value is not a valid reason.
type ForcedFailureReason int

// The reasons for a failure without a roll.
const (
	// ChosenToFail is a saving throw the creature chooses to fail ("Saving
	// Throws", SRD 5.2.1, p. 7).
	ChosenToFail ForcedFailureReason = iota + 1
	// Automatic is a failure the caller declares from the situation, such
	// as a sight-based check while Blinded (p. 177) or an object's saving
	// throw (p. 178).
	Automatic
)

// String returns the srd5e.d20_test.forced_failure_reason telemetry value, or
// "unknown" for an invalid reason.
func (r ForcedFailureReason) String() string {
	switch r {
	case ChosenToFail:
		return ValueForcedChosenToFail
	case Automatic:
		return ValueForcedAutomatic
	default:
		return "unknown"
	}
}

// DifficultyClass is the target number of an ability check or saving throw
// ("Difficulty Class", SRD 5.2.1, p. 6). The SRD sets no minimum or maximum,
// so every int is valid. It never equals an [ArmorClass].
type DifficultyClass int

// The Typical Difficulty Classes table (SRD 5.2.1, p. 6).
const (
	DCVeryEasy         DifficultyClass = 5
	DCEasy             DifficultyClass = 10
	DCMedium           DifficultyClass = 15
	DCHard             DifficultyClass = 20
	DCVeryHard         DifficultyClass = 25
	DCNearlyImpossible DifficultyClass = 30
)

// ArmorClass is the target number of an attack roll ("Armor Class", SRD 5.2.1,
// p. 7). The SRD sets no minimum or maximum, so every int is valid.
type ArmorClass int
