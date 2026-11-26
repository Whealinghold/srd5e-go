package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/initiative-and-actions.md FR-092 - the private size comparison compares the six sizes by their order: the gap is the absolute difference and passing needs at least two.
// Path: unhappy (boundary).
func TestSizeGapOverTheSixSizes(t *testing.T) {
	t.Parallel()
	sizes := []content.CreatureSize{content.SizeTiny, content.SizeSmall, content.SizeMedium, content.SizeLarge, content.SizeHuge, content.SizeGargantuan}
	for i, a := range sizes {
		for j, b := range sizes {
			want := i - j
			if want < 0 {
				want = -want
			}
			if got := sizeGap(a, b); got != want {
				t.Errorf("sizeGap(%v, %v) = %d, want %d", a, b, got, want)
			}
		}
	}
	if satMulPositive(1<<20, 3) != 3<<20 {
		t.Error("satMulPositive is not a product")
	}
}
