package content_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: schema FR-139 (SRD 5.2.1, "Creature Size", Creature Size and
// Space table, p. 14) - the size rows.
// Path: happy.
func TestCreatureSizeSpace(t *testing.T) {
	t.Parallel()
	whole := func(side float64, perSide, squares int) func(content.CreatureSpace) string {
		return func(s content.CreatureSpace) string {
			w, ok := s.(content.WholeSquareSpace)
			if !ok || w.SideFeet() != side || w.SquaresPerSide() != perSide || w.Squares() != squares {
				return fmt.Sprintf("Space() = %#v, want WholeSquareSpace %v ft, %d per side, %d squares", s, side, perSide, squares)
			}
			return ""
		}
	}
	cases := []struct {
		size  content.CreatureSize
		check func(content.CreatureSpace) string
	}{
		{content.SizeTiny, func(s content.CreatureSpace) string {
			sh, ok := s.(content.SharedSquareSpace)
			if !ok || sh.SideFeet() != 2.5 || sh.CreaturesPerSquare() != 4 {
				return fmt.Sprintf("Space() = %#v, want SharedSquareSpace 2.5 ft, 4 per square", s)
			}
			return ""
		}},
		{content.SizeSmall, whole(5, 1, 1)},
		{content.SizeMedium, whole(5, 1, 1)},
		{content.SizeLarge, whole(10, 2, 4)},
		{content.SizeHuge, whole(15, 3, 9)},
		{content.SizeGargantuan, whole(20, 4, 16)},
	}
	for _, c := range cases {
		t.Run(c.size.String(), func(t *testing.T) {
			t.Parallel()
			if msg := c.check(c.size.Space()); msg != "" {
				t.Error(msg)
			}
		})
	}
}

// Requirement: schema FR-139 - values compare with == and an invalid size has no space.
// Path: unhappy (rule interaction).
func TestCreatureSizeSpaceEquality(t *testing.T) {
	t.Parallel()
	if content.SizeSmall.Space() != content.SizeMedium.Space() {
		t.Error("Small and Medium spaces differ")
	}
	if content.SizeLarge.Space() == content.SizeHuge.Space() {
		t.Error("Large and Huge spaces are equal")
	}
	if content.CreatureSize(0).Space() != nil || content.CreatureSize(7).Space() != nil {
		t.Error("an invalid size has a space")
	}
	// Requirement: schema FR-139 (size consistency) - SideFeet is 5 * SquaresPerSide for whole squares.
	for z := content.SizeSmall; z <= content.SizeGargantuan; z++ {
		w := z.Space().(content.WholeSquareSpace)
		if w.SideFeet() != 5*float64(w.SquaresPerSide()) {
			t.Errorf("%v: SideFeet %v != 5 * %d", z, w.SideFeet(), w.SquaresPerSide())
		}
	}
}
