package character

// IsBloodied reports whether a creature is Bloodied: it has half its Hit
// Points or fewer remaining (SRD 5.2.1, Rules Glossary, "Bloodied", p. 177).
// "Half its Hit Points" is read as half the Hit Point maximum, in whole-number
// arithmetic: twice currentHitPoints is at most the maximum. Current above the
// maximum is not Bloodied. Temporary Hit Points never count: pass current Hit
// Points only. It is the only Bloodied predicate in the module.
//
// It returns false and an error matching srderr.ErrInvalidInput for a negative
// currentHitPoints or the zero maximum. It never panics.
func IsBloodied(currentHitPoints int, maximum HitPointMaximum) (bool, error) {
	if currentHitPoints < 0 {
		return false, invalidInput("negative current Hit Points")
	}
	if maximum.IsZero() {
		return false, invalidInput("zero Hit Point maximum")
	}
	return 2*currentHitPoints <= maximum.n, nil
}
