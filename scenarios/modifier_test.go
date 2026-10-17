package scenarios_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/modifier"
)

// Advantage and Disadvantage do not stack: any of each cancels to a normal
// roll ("They Don't Stack", SRD 5.2.1, p. 8).
func TestAdvantageAndDisadvantageCancel(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	tests := []struct {
		name string
		in   []modifier.RollCircumstance
		want modifier.RollMode
	}{
		{"none", nil, modifier.Normal},
		{"advantage", []modifier.RollCircumstance{adv}, modifier.WithAdvantage},
		{"disadvantage", []modifier.RollCircumstance{dis}, modifier.WithDisadvantage},
		{"two advantages do not stack", []modifier.RollCircumstance{adv, adv}, modifier.WithAdvantage},
		{"one of each cancels", []modifier.RollCircumstance{adv, dis}, modifier.Normal},
		{"many of each cancel", []modifier.RollCircumstance{adv, adv, adv, dis}, modifier.Normal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := modifier.Resolve(tc.in...); got != tc.want {
				t.Fatalf("Resolve = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRollModeRoundTripsThroughText(t *testing.T) {
	t.Parallel()
	for _, m := range []modifier.RollMode{modifier.Normal, modifier.WithAdvantage, modifier.WithDisadvantage} {
		got, err := modifier.ParseRollMode(m.String())
		if err != nil || got != m {
			t.Errorf("ParseRollMode(%q) = %v, %v; want %v", m.String(), got, err, m)
		}
	}
	if _, err := modifier.ParseRollMode("lucky"); err == nil {
		t.Error("ParseRollMode(lucky) succeeded")
	}
}
