package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: monSpec FR-015, FR-016, FR-095 - the Hit Points average and a speed's feet accept
// MonsterMaxNumber itself (the bound is inclusive) and reject one above it.
// Path: unhappy (boundary).
func TestMonsterHeaderUpperBounds(t *testing.T) {
	t.Parallel()
	d := dice.MustExpression(1, dice.MustDie(8), 0)
	if _, err := content.NewHitPointsEntry(content.MonsterMaxNumber, d); err != nil {
		t.Errorf("NewHitPointsEntry(max) = %v, want nil", err)
	}
	if _, err := content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, content.MonsterMaxNumber, false, ""); err != nil {
		t.Errorf("NewSpeedEntry(max feet) = %v, want nil", err)
	}
	over, ok := aboveMaxNumber()
	if !ok {
		return
	}
	_, err := content.NewHitPointsEntry(over, d)
	wantBlockError(t, err, "HitPointsEntry", content.BlockOutOfRange)
	_, err = content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, over, false, "")
	wantBlockError(t, err, "SpeedEntry", content.BlockOutOfRange)
}

// Requirement: monSpec FR-016 - SpeedEntry.Equal compares hover and qualifier as
// well as modes and feet.
// Path: per case.
func TestSpeedEntryEqualMembers(t *testing.T) {
	t.Parallel()
	fly := []content.SpeedMode{content.SpeedFly}
	base, err := content.NewSpeedEntry(fly, 30, false, "")
	if err != nil {
		t.Fatal(err)
	}
	hover, _ := content.NewSpeedEntry(fly, 30, true, "")
	qual, _ := content.NewSpeedEntry(fly, 30, false, "wolf form only")
	feet, _ := content.NewSpeedEntry(fly, 40, false, "")
	same, _ := content.NewSpeedEntry(fly, 30, false, "")
	if !base.Equal(same) {
		t.Error("equal speeds differ [happy]")
	}
	for name, o := range map[string]content.SpeedEntry{"hover": hover, "qualifier": qual, "feet": feet} {
		if base.Equal(o) {
			t.Errorf("speeds differing in %s are Equal [unhappy]", name)
		}
	}
}
