package d20

// ForcedFailureResult is the failure of an ability check or saving throw
// without a roll ("Saving Throws", SRD 5.2.1, p. 7; "Blinded", p. 177;
// "Object", p. 178). It carries no roll; its outcome is always [Failure], and
// the reason says why. It is an immutable, comparable value built by
// [FailWithoutRoll]. The zero value is not a result.
type ForcedFailureResult struct {
	kind   Kind
	reason ForcedFailureReason
}

// Kind returns [AbilityCheck] or [SavingThrow].
func (f ForcedFailureResult) Kind() Kind { return f.kind }

// Reason returns why the test failed without a roll.
func (f ForcedFailureResult) Reason() ForcedFailureReason { return f.reason }
