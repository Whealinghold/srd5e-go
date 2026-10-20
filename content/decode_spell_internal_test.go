package content

import (
	"errors"
	"testing"
)

// Requirement: spell-effect-data.md FR-031 - the problem kind of a constructor's
// reason: empty and out of range are outOfRange, a repeat is duplicate, a ragged
// row and a broken rule between parts are constraintViolation, a missing part is
// missingField, and every invalid text, name or member is invalidValue; an error
// that is not a *BlockError is invalidValue.
// Path: per case.
func TestBlockKind(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		p    BlockProblem
		want ProblemKind
	}{
		{BlockEmpty, ProblemOutOfRange}, {BlockOutOfRange, ProblemOutOfRange}, {BlockDuplicate, ProblemDuplicate},
		{BlockRaggedRow, ProblemConstraintViolation}, {BlockConstraint, ProblemConstraintViolation}, {BlockMissing, ProblemMissingField},
		{BlockInvalidName, ProblemInvalidValue}, {BlockBlankText, ProblemInvalidValue}, {BlockInvalidMember, ProblemInvalidValue},
		{BlockInvalidCharacter, ProblemInvalidValue}, {BlockProblem(99), ProblemInvalidValue},
	} {
		if got := blockKind(blockError("X", c.p)); got != c.want {
			t.Errorf("blockKind(%v) = %v, want %v", c.p, got, c.want)
		}
	}
	if got := blockKind(errors.New("not a block error")); got != ProblemInvalidValue {
		t.Errorf("blockKind(other error) = %v, want invalidValue", got)
	}
}

// Requirement: content-classes.md FR-058 - the constructor error of a rule broken between
// parts maps its problem kind back to a block problem: missing, outOfRange,
// invalidValue and duplicate to their own, any other kind to constraint.
// Path: per case.
func TestIssueBlockError(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		k    ProblemKind
		want BlockProblem
	}{
		{ProblemMissingField, BlockMissing}, {ProblemOutOfRange, BlockOutOfRange}, {ProblemInvalidValue, BlockInvalidMember},
		{ProblemDuplicate, BlockDuplicate}, {ProblemConstraintViolation, BlockConstraint}, {ProblemWrongType, BlockConstraint},
	} {
		var be *BlockError
		if err := (issue{"p", c.k}).blockError("X"); !errors.As(err, &be) || be.Problem != c.want || be.Block != "X" {
			t.Errorf("issue kind %v: got %v, want problem %v", c.k, err, c.want)
		}
	}
}

// Requirement: content-spells.md FR-037, FR-051, FR-111 - a range or duration whose private members the
// constructor would reject (a variant built by hand in the package) fails its check with the constructor's reason.
// Path: unhappy (invalid input).
func TestCheckSpellRangeAndDurationRecheckEveryMember(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		err  error
		want BlockProblem
	}{
		{"range unit", checkSpellRange(SpellRangeDistance{amount: 5}), BlockInvalidMember},
		{"range amount", checkSpellRange(SpellRangeDistance{amount: 0, unit: DistanceFeet}), BlockOutOfRange},
		{"duration unit", checkSpellDuration(SpellDurationTimed{amount: 1, limit: LimitFixed}), BlockInvalidMember},
		{"duration limit", checkSpellDuration(SpellDurationTimed{amount: 1, unit: DurationUnitHours}), BlockInvalidMember},
		{"duration amount", checkSpellDuration(SpellDurationTimed{amount: 0, unit: DurationUnitHours, limit: LimitFixed}), BlockOutOfRange},
	} {
		var be *BlockError
		if !errors.As(c.err, &be) || be.Problem != c.want {
			t.Errorf("%s: err = %v, want problem %v", c.name, c.err, c.want)
		}
	}
	if checkSpellRange(SpellRangeDistance{amount: 5, unit: DistanceMiles}) != nil ||
		checkSpellDuration(SpellDurationTimed{amount: 1, unit: DurationUnitHours, limit: LimitFixed}) != nil {
		t.Error("a valid hand-built variant is rejected")
	}
}

// Requirement: content-spells.md FR-043, FR-101, FR-102 - WithMaterial rechecks the costs, returns the zero value
// beside an error, copies the component in and out, and Equal compares the presence and the material.
// Path: unhappy (invalid input).
func TestSpellComponentsMaterialIsRecheckedAndCopied(t *testing.T) {
	t.Parallel()
	base := NewSpellComponents(true, true)
	got, err := base.WithMaterial(MaterialComponent{description: "x", costsGP: []int{0}})
	var be *BlockError
	if !errors.As(err, &be) || be.Problem != BlockOutOfRange || got.Verbal() || got.Somatic() {
		t.Errorf("WithMaterial(bad cost) = (%v, %v), want zero value and outOfRange", got, err)
	}
	src := MaterialComponent{description: "x", costsGP: []int{5}}
	with, err := base.WithMaterial(src)
	if err != nil {
		t.Fatal(err)
	}
	src.costsGP[0] = 9
	m, _ := with.Material()
	if m.costsGP[0] != 5 {
		t.Error("WithMaterial kept the caller's slice")
	}
	m.costsGP[0] = 7
	if again, _ := with.Material(); again.costsGP[0] != 5 {
		t.Error("Material returned the stored slice")
	}
	other, _ := base.WithMaterial(MaterialComponent{description: "y", costsGP: []int{5}})
	if with.Equal(other) || NewSpellComponents(false, false).Equal(SpellComponents{hasMaterial: true}) {
		t.Error("components with a different or an absent material are equal")
	}
}
