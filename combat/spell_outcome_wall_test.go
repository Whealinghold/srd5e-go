package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-075 - the zero wall is intact, NewPrismaticWallStateAfter accepts 0 to 7 and rejects -1 and 8 with ErrWallDestroyedCount (an invalid input), and Layers lists the colours still present from red; a read copy cannot change the state (SRD 5.2.1, "Prismatic Wall", p. 155).
// Path: per case.
func TestPrismaticWallState(t *testing.T) {
	t.Parallel()
	if got := (combat.PrismaticWallState{}).Layers(); len(got) != 7 || got[0] != combat.PrismaticRed || got[6] != combat.PrismaticViolet {
		t.Errorf("the zero wall holds %v, want the seven layers", got)
	}
	for _, n := range []int{0, 3, 7} {
		w, err := combat.NewPrismaticWallStateAfter(n)
		if err != nil || w.Destroyed() != n || len(w.Layers()) != 7-n {
			t.Errorf("NewPrismaticWallStateAfter(%d) = %v, %v with %d layers", n, w, err, len(w.Layers()))
		}
	}
	for _, n := range []int{-1, 8} {
		if _, err := combat.NewPrismaticWallStateAfter(n); !errors.Is(err, combat.ErrWallDestroyedCount) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewPrismaticWallStateAfter(%d) = %v, want ErrWallDestroyedCount", n, err)
		}
	}
	w := luMust(combat.NewPrismaticWallStateAfter(5))
	w.Layers()[0] = combat.PrismaticRed
	if !slices.Equal(w.Layers(), []combat.PrismaticColor{combat.PrismaticIndigo, combat.PrismaticViolet}) {
		t.Errorf("Layers() = %v, want indigo and violet", w.Layers())
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-076, FR-077, FR-079, FR-080, table W - destroying a layer: the right destroyer at its threshold destroys it (W1, W4, W6, W7, W8, W9, W10), a smaller amount or level gives BelowThreshold (W2, W6, W7), another kind gives WrongDestroyer (W3, W11), a layer that is not the outermost gives NotOutermost (W5) and one already gone gives AlreadyDestroyed (W12); the old state is never changed and Antimagic Field has no value (SRD 5.2.1, "Prismatic Wall", "Prismatic Layers", p. 155).
// Path: per case.
func TestPrismaticWallDestroy(t *testing.T) {
	t.Parallel()
	const (
		red, orange, yellow, green = combat.PrismaticRed, combat.PrismaticOrange, combat.PrismaticYellow, combat.PrismaticGreen
		blue, indigo, violet       = combat.PrismaticBlue, combat.PrismaticIndigo, combat.PrismaticViolet
	)
	cases := []struct {
		name, requirement, path string
		gone                    int
		layer                   combat.PrismaticColor
		destroyer               combat.PrismaticLayerDestroyer
		want                    combat.PrismaticLayerDestruction
	}{
		{"W1 red 25 Cold", "FR-077", "unhappy (boundary)", 0, red, combat.DamageToLayer{Type: content.DamageCold, Amount: 25}, combat.PrismaticLayerDestroyed{Layer: red}},
		{"W1 red 99 Cold", "FR-077", "happy", 0, red, combat.DamageToLayer{Type: content.DamageCold, Amount: 99}, combat.PrismaticLayerDestroyed{Layer: red}},
		{"W2 red 24 Cold", "FR-077", "unhappy (boundary)", 0, red, combat.DamageToLayer{Type: content.DamageCold, Amount: 24}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerBelowThreshold}},
		{"W3 red 99 Fire", "FR-077", "unhappy (rule interaction)", 0, red, combat.DamageToLayer{Type: content.DamageFire, Amount: 99}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"red with a wind", "FR-077", "unhappy (rule interaction)", 0, red, combat.StrongWind{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"W4 orange wind", "FR-077", "happy", 1, orange, combat.StrongWind{}, combat.PrismaticLayerDestroyed{Layer: orange}},
		{"orange with Cold", "FR-077", "unhappy (rule interaction)", 1, orange, combat.DamageToLayer{Type: content.DamageCold, Amount: 99}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"W5 orange wind with all present", "FR-076", "unhappy (rule interaction)", 0, orange, combat.StrongWind{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerNotOutermost}},
		{"W6 yellow 60 Force", "FR-077", "unhappy (boundary)", 2, yellow, combat.DamageToLayer{Type: content.DamageForce, Amount: 60}, combat.PrismaticLayerDestroyed{Layer: yellow}},
		{"W6 yellow 59 Force", "FR-077", "unhappy (boundary)", 2, yellow, combat.DamageToLayer{Type: content.DamageForce, Amount: 59}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerBelowThreshold}},
		{"W7 green portal level 5", "FR-077", "unhappy (boundary)", 3, green, combat.PortalOpened{SpellLevel: 5}, combat.PrismaticLayerDestroyed{Layer: green}},
		{"W7 green portal level 9", "FR-077", "happy", 3, green, combat.PortalOpened{SpellLevel: 9}, combat.PrismaticLayerDestroyed{Layer: green}},
		{"W7 green portal level 4", "FR-077", "unhappy (boundary)", 3, green, combat.PortalOpened{SpellLevel: 4}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerBelowThreshold}},
		{"green with wind", "FR-077", "unhappy (rule interaction)", 3, green, combat.StrongWind{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"W8 blue 25 Fire", "FR-077", "unhappy (boundary)", 4, blue, combat.DamageToLayer{Type: content.DamageFire, Amount: 25}, combat.PrismaticLayerDestroyed{Layer: blue}},
		{"blue 24 Fire", "FR-077", "unhappy (boundary)", 4, blue, combat.DamageToLayer{Type: content.DamageFire, Amount: 24}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerBelowThreshold}},
		{"W9 indigo daylight", "FR-077", "happy", 5, indigo, combat.DaylightBrightLight{}, combat.PrismaticLayerDestroyed{Layer: indigo}},
		{"W10 violet Dispel Magic", "FR-077", "happy", 6, violet, combat.DispelMagicOnLayer{}, combat.PrismaticLayerDestroyed{Layer: violet}},
		{"W11 indigo Dispel Magic", "FR-080", "unhappy (rule interaction)", 5, indigo, combat.DispelMagicOnLayer{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"violet with daylight", "FR-080", "unhappy (rule interaction)", 6, violet, combat.DaylightBrightLight{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"W12 red again", "FR-076", "unhappy (state error)", 1, red, combat.StrongWind{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerAlreadyDestroyed}},
		{"a layer that is not a colour", "FR-076", "unhappy (invalid input)", 0, 0, combat.StrongWind{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerNotOutermost}},
		{"no destroyer", "FR-077", "unhappy (empty and minimal)", 0, red, nil, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerWrongDestroyer}},
		{"an empty wall", "FR-079", "unhappy (empty and minimal)", 7, violet, combat.DispelMagicOnLayer{}, combat.PrismaticLayerNotDestroyed{Reason: combat.LayerAlreadyDestroyed}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := luMust(combat.NewPrismaticWallStateAfter(c.gone))
			after, got := before.Destroy(c.layer, c.destroyer)
			_, destroyed := got.(combat.PrismaticLayerDestroyed)
			wantCount := c.gone
			if destroyed {
				wantCount++
			}
			if got != c.want || after.Destroyed() != wantCount || before.Destroyed() != c.gone {
				t.Errorf("Destroy = %#v, count %d (old %d), want %#v, count %d [%s%s]", got, after.Destroyed(), before.Destroyed(), c.want, wantCount, otSpec, c.requirement)
			}
		})
	}
	reasons := []string{combat.LayerNotOutermost.String(), combat.LayerAlreadyDestroyed.String(), combat.LayerWrongDestroyer.String(), combat.LayerBelowThreshold.String(), combat.PrismaticNotDestroyedReason(0).String()}
	if !slices.Equal(reasons, []string{"not_outermost", "already_destroyed", "wrong_destroyer", "below_threshold", "unknown"}) {
		t.Errorf("reason names = %v", reasons)
	}
}

// Requirement: doc/specs/spell-outcome-tables.md FR-076 - the layers go one at a time from red to violet: seven destroyers in order empty the wall.
// Path: happy.
func TestPrismaticWallDestroyedInOrder(t *testing.T) {
	t.Parallel()
	steps := []struct {
		layer     combat.PrismaticColor
		destroyer combat.PrismaticLayerDestroyer
	}{
		{combat.PrismaticRed, combat.DamageToLayer{Type: content.DamageCold, Amount: 30}}, {combat.PrismaticOrange, combat.StrongWind{}},
		{combat.PrismaticYellow, combat.DamageToLayer{Type: content.DamageForce, Amount: 60}}, {combat.PrismaticGreen, combat.PortalOpened{SpellLevel: 5}},
		{combat.PrismaticBlue, combat.DamageToLayer{Type: content.DamageFire, Amount: 25}}, {combat.PrismaticIndigo, combat.DaylightBrightLight{}},
		{combat.PrismaticViolet, combat.DispelMagicOnLayer{}},
	}
	var wall combat.PrismaticWallState
	for i, s := range steps {
		var got combat.PrismaticLayerDestruction
		if wall, got = wall.Destroy(s.layer, s.destroyer); got != (combat.PrismaticLayerDestroyed{Layer: s.layer}) || wall.Destroyed() != i+1 {
			t.Fatalf("step %d: %#v, %d destroyed", i, got, wall.Destroyed())
		}
	}
	if len(wall.Layers()) != 0 {
		t.Errorf("layers left = %v, want none", wall.Layers())
	}
}
