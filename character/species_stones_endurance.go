package character

// StonesEnduranceRule is the damage reduction of a Goliath with the Stone Giant
// ancestry (SRD 5.2.1, "Stone’s Endurance", p. 85). The zero value is ready to
// use. It is stateless and safe for concurrent use.
type StonesEnduranceRule struct{}

// Reduce returns the reduction, max(0, roll + modifier), and the damage left,
// max(0, damage - reduction). Stone's Endurance reduces damage and cannot raise it, so
// a negative total reduces by 0. A negative damage counts as 0. It never panics.
func (StonesEnduranceRule) Reduce(roll, modifier, damage int) (reduction, left int) {
	reduction = max(0, roll+modifier)
	return reduction, max(0, damage-reduction)
}
