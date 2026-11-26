package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzNewSpellTimeRemaining checks the constructor never panics, accepts exactly the
// positive seconds, reports them back, and refuses others with ErrInvalidInput.
func FuzzNewSpellTimeRemaining(f *testing.F) {
	for _, n := range []int{0, 1, 6, -1, 1 << 30} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		r, err := combat.NewSpellTimeRemaining(n)
		if n < 1 {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("NewSpellTimeRemaining(%d) error = %v, want ErrInvalidInput", n, err)
			}
			return
		}
		if err != nil {
			t.Fatalf("NewSpellTimeRemaining(%d) error = %v", n, err)
		}
		if got, ok := r.Seconds(); !ok || got != n {
			t.Fatalf("Seconds() = %d, %v; want %d", got, ok, n)
		}
	})
}

// FuzzNewPrismaticLayer checks the layer constructor never panics and accepts exactly 1 to 7.
func FuzzNewPrismaticLayer(f *testing.F) {
	for _, n := range []int{0, 1, 7, 8, -1} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		l, err := combat.NewPrismaticLayer(n)
		inRange := n >= 1 && n <= 7
		if inRange != (err == nil) {
			t.Fatalf("NewPrismaticLayer(%d) error = %v, want accepted=%v", n, err, inRange)
		}
		if err == nil && (!l.Valid() || int(l) != n) {
			t.Fatalf("NewPrismaticLayer(%d) = %v", n, l)
		}
	})
}
