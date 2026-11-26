package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// circs returns n Advantage circumstances.
func circs(n int) []modifier.RollCircumstance {
	out := make([]modifier.RollCircumstance, n)
	for i := range out {
		out[i] = modifier.Advantage{}
	}
	return out
}

// Requirement: doc/specs/mounted-and-underwater-combat.md Limits (plan P10) - a command carrying a d20.RollAttempt holds at most 256 circumstances; the gear commands answer GearProblemLimitExceeded, the others a LimitExceededError.
// Path: per case.
func TestRollAttemptCircumstancesCap(t *testing.T) {
	t.Parallel()
	save := func(n int) d20.RollAttempt { return rollOf(0, circs(n)...) }
	limit := func(collection string) func(testing.TB, error) {
		return func(tb testing.TB, err error) { tb.Helper(); wantLimit(tb, err, collection, 256) }
	}
	gear := func(tb testing.TB, err error) { tb.Helper(); wantGearLimit(tb, err) }
	cases := []struct {
		name  string
		build func(n int) error
		want  func(testing.TB, error)
	}{
		{"resolve falling off", func(n int) error {
			_, err := combat.NewResolveFallingOffCommand(gid("rider"), mntMounted(t, combat.MountControlControlled), combat.FallingOffTriggerRiderKnockedProne, save(n), combat.LandingSpaceUnoccupiedWithin5Feet)
			return err
		}, limit("attempt circumstances")},
		{"enter container", func(n int) error {
			in := ctEnterInput(t, "cube")
			in.Attempt = save(n)
			_, err := combat.NewEnterContainerCommand(in)
			return err
		}, limit("attempt circumstances")},
		{"exit container by self", func(n int) error {
			_, err := combat.NewExitContainerCommand(ctExitInput(t, "cube", ctSelf(save(n))))
			return err
		}, limit("attempt circumstances")},
		{"exit container by pull", func(n int) error {
			_, err := combat.NewExitContainerCommand(ctExitInput(t, "cube", combat.ExitByPull{Puller: gid("q"), PullerWithin5Feet: true, Attempt: save(n), Actions: combat.ConditionActionAvailabilityAvailable}))
			return err
		}, limit("attempt circumstances")},
		{"resolve regurgitation", func(n int) error {
			_, err := combat.NewResolveRegurgitationCommand(combat.ResolveRegurgitationInput{Container: gid("m"), Attempt: save(n)})
			return err
		}, limit("attempt circumstances")},
		{"resolve thrown item", func(n int) error {
			in := thrownIn()
			in.Attempt = save(n)
			_, err := combat.NewResolveThrownItemCommand(in)
			return err
		}, limit("attempt circumstances")},
		{"bind creature", func(n int) error {
			in := bindIn(t)
			in.Attempt = save(n)
			_, err := combat.NewBindCreatureCommand(in)
			return err
		}, gear},
		{"throw net", func(n int) error {
			in := netIn()
			in.Attempt = save(n)
			_, err := combat.NewThrowNetCommand(in)
			return err
		}, gear},
		{"release from hold", func(n int) error {
			in := releaseIn(gearHolds(t, gearHold(t, holdIn("h", combat.HoldKindNet))))
			in.Attempt = save(n)
			_, err := combat.NewReleaseFromHoldCommand(in)
			return err
		}, gear},
		{"resolve gear entry", func(n int) error {
			in := gearEntryIn(combat.DeployedGears{})
			in.Attempt = save(n)
			_, err := combat.NewResolveGearEntryCommand(in)
			return err
		}, gear},
		{"counter cast spell", func(n int) error {
			cmd := ccsCommand(t)
			cmd.CasterSave = ccsRoll(circs(n)...)
			_, err := ccsUse(t, dicetest.NewFixedRoller(10, 10)).Run(context.Background(), cmd)
			return err
		}, limit("attempt circumstances")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			t.Run("at the cap", func(t *testing.T) {
				t.Parallel()
				if err := c.build(256); err != nil {
					t.Errorf("256 circumstances: error = %v, want none", err)
				}
			})
			t.Run("cap plus one", func(t *testing.T) {
				t.Parallel()
				c.want(t, c.build(257))
			})
		})
	}
}
