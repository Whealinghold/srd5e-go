package combat

import "errors"

// AttackAllowance is the number of attacks one Attack action may make (SRD 5.2.1,
// "Attack [Action]", p. 177): GeneralAttacks for any means, and PactWeaponAttacks for
// an action made entirely with the pact weapon (replacements allowed). The host
// builds it from the owners of the counts (the Extra Attack combination and the pact
// weapon count) and passes plain integers; this package computes neither. The zero
// value is invalid. Comparable and safe for concurrent use.
type AttackAllowance struct{ general, pact int }

// BaselineAttackAllowance returns the allowance with no Extra Attack: one attack for
// every means (SRD 5.2.1, "Attack [Action]", p. 177).
func BaselineAttackAllowance() AttackAllowance { return AttackAllowance{1, 1} }

// NewAttackAllowance returns the allowance. It returns an error matching
// srderr.ErrInvalidInput unless generalAttacks is at least 1 and pactWeaponAttacks is
// at least generalAttacks, and a LimitExceededError for more than 256 of either (plan P10). It never panics.
func NewAttackAllowance(generalAttacks, pactWeaponAttacks int) (AttackAllowance, error) {
	if generalAttacks < 1 || pactWeaponAttacks < generalAttacks {
		return AttackAllowance{}, invalidInput("attack allowance needs general >= 1 and pact >= general")
	}
	if err := errors.Join(overCap("general attacks", generalAttacks, maxInputList), overCap("pact weapon attacks", pactWeaponAttacks, maxInputList)); err != nil {
		return AttackAllowance{}, err
	}
	return AttackAllowance{generalAttacks, pactWeaponAttacks}, nil
}

// GeneralAttacks returns the attacks allowed with any means.
func (a AttackAllowance) GeneralAttacks() int { return a.general }

// PactWeaponAttacks returns the attacks allowed when every attack is a pact weapon
// attack or a replacement.
func (a AttackAllowance) PactWeaponAttacks() int { return a.pact }

// Valid reports whether a was built by NewAttackAllowance.
func (a AttackAllowance) Valid() bool { return a.general >= 1 && a.pact >= a.general }
