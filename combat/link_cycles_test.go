package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: plan-phase-4.md P10, doc/specs/combat-action-resolution.md section 1 (Limits) - a mount link cannot form a cycle: a mount must be at least one size larger than its rider, so with the six sizes no chain of mounts has more than six links and no link leads back (A on B, then B on A, is refused), and MountCreature reads only the one rider's own status, so nothing is walked and nothing recurses. Containment links are the same rule in TestNewContainmentsInvariantsAndCycles (2, 3 and 2001 links, with sets and no recursion); summon links wait for step 12.
// Path: per case.
func TestMountLinksCannotCycle(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		rider, mount            content.CreatureSize
		want                    string
	}{
		{"Medium on Large", "P10 mount chains", "happy", content.SizeMedium, content.SizeLarge, "applied"},
		{"the mount then mounts the rider", "P10 link cycles are rejected", "unhappy (rule interaction)", content.SizeLarge, content.SizeMedium, "mount_not_larger"},
		{"a creature on itself", "P10 link cycles are rejected", "unhappy (rule interaction)", content.SizeMedium, content.SizeMedium, "mount_not_larger"},
		{"the longest chain: Tiny on Small", "P10 mount chains", "unhappy (boundary)", content.SizeTiny, content.SizeSmall, "applied"},
		{"the longest chain: Huge on Gargantuan", "P10 mount chains", "unhappy (boundary)", content.SizeHuge, content.SizeGargantuan, "applied"},
		{"past the longest chain: Gargantuan on Gargantuan", "P10 mount chains", "unhappy (boundary)", content.SizeGargantuan, content.SizeGargantuan, "mount_not_larger"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			in := mntInput(t)
			in.RiderSize, in.MountSize = c.rider, c.mount
			res, err := combat.NewMountCreature().Run(context.Background(), mntCommand(t, in))
			if err != nil {
				t.Fatal(err)
			}
			got := "applied"
			if r, rejected := res.(combat.MountCreatureRejected); rejected {
				got = r.Reason().String()
			}
			if got != c.want {
				t.Errorf("mounting %v on %v = %q, want %q [%s]", c.rider, c.mount, got, c.want, c.requirement)
			}
		})
	}
}
