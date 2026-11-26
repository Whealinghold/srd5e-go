package combat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// fakeSummonRequest is a request of a kind no constructor accepts, to reach the branches that guard against one.
type fakeSummonRequest struct{}

func (fakeSummonRequest) isSummonRequest() {}

// fakeSummonedChange is a change of a kind NewApplySummonedChangeCommand refuses.
type fakeSummonedChange struct{}

func (fakeSummonedChange) isSummonedChange() {}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-067, FR-070 - a request, change or command that no constructor builds (a request of an unknown variant, a zero command, a value forged inside the package) is handled without panicking: Summon leaves the roster unchanged, Apply leaves it unchanged, validation says no, and the use case turns a summons that made nothing into an error rather than a result.
// Path: unhappy (invalid input, state forgery).
func TestForgedRequestsAndChangesAreHandled(t *testing.T) {
	t.Parallel()
	var roster SummonedRoster
	cmd := SummonCreatureCommand{roster: roster, summoner: "pc", side: SideA, request: fakeSummonRequest{}}
	if got := roster.Summon(cmd); got != (SummonedUnchanged{}) {
		t.Errorf("Summon of an unknown request = %#v, want SummonedUnchanged", got)
	}
	if got := roster.Apply(fakeSummonedChange{}, ""); got != (SummonedUnchanged{}) {
		t.Errorf("Apply of an unknown change = %#v, want SummonedUnchanged", got)
	}
	if got := roster.Apply(nil, ""); got != (SummonedUnchanged{}) {
		t.Errorf("Apply of no change = %#v, want SummonedUnchanged", got)
	}
	if validChange(fakeSummonedChange{}) || validChange(nil) || validRequest(fakeSummonRequest{}) || validRequest(nil) {
		t.Errorf("validation accepted a value that no constructor builds")
	}
	if kindOf(nil) != 0 || kindOf(fakeSummonRequest{}) != 0 || levelOf(nil) != 0 || levelOf(fakeSummonRequest{}) != 0 || len(newIDs(fakeSummonRequest{})) != 0 {
		t.Errorf("an unknown request has a kind, level or ids")
	}
	if cloneRequest(fakeSummonRequest{}) != (fakeSummonRequest{}) || cloneChange(fakeSummonedChange{}) != (fakeSummonedChange{}) {
		t.Errorf("an unknown value was changed by the copy")
	}
	uc, err := NewSummonCreature(contenttest.NewFakeSource())
	if err != nil {
		t.Fatal(err)
	}
	if res, err := uc.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a summons that made nothing = %v, %v, want srderr.ErrInvalidInput [FR-067]", res, err)
	}
	if got := roster.Reassert("pc", ReassertControlRequest{Kind: SummonGiantInsect}); got != (SummonedUnchanged{}) {
		t.Errorf("a reassert of a kind that is not undead = %#v, want SummonedUnchanged", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-035, FR-060 - a command forged inside the package with facts or a mount that the constructors would have refused is refused with FormNotEligibleRejection rather than building a block from nothing: the stat block constructors are checked again at the point of use.
// Path: unhappy (invalid input, state forgery).
func TestAForgedCommandCannotBuildABlock(t *testing.T) {
	t.Parallel()
	facts := SummonerSpellFacts{modifier: 5} // so the weights of the objects pass and only the stat block fails
	for name, req := range map[string]SummonRequest{
		"objects":    AnimateObjectsRequest{Level: 5, Objects: []ObjectTarget{{Creature: "o", Size: content.SizeSmall}}},
		"insect":     GiantInsectRequest{Creature: "c", Level: 5, Form: InsectWasp},
		"spirit":     SummonDragonRequest{Creature: "c", Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageCold},
		"steed":      FindSteedRequest{Level: 2, Type: SteedFey, Choice: SteedChoiceDifferent{Creature: "c"}, Anatomy: MountAnatomyAppropriate, Habitat: MountHabitatOther, Saddle: ExoticSaddleFitted},
		"simulacrum": SimulacrumRequest{Creature: "c", Level: 7, Original: SimulacrumOriginal{}},
	} {
		cmd := SummonCreatureCommand{summoner: "pc", side: SideA, request: req, facts: facts, scene: SummonScene{SpaceAvailable: true}}
		got := (SummonedRoster{}).Summon(cmd)
		if r, ok := got.(SummonedRejected); !ok || r.Rejection != (FormNotEligibleRejection{}) {
			t.Errorf("%s: transition = %#v, want FormNotEligibleRejection", name, got)
		}
	}
	steed := FindSteedRequest{Level: 2, Type: SteedFey, Choice: SteedChoiceDifferent{Creature: "c"}, Saddle: ExoticSaddleFitted}
	cmd := SummonCreatureCommand{summoner: "pc", side: SideA, request: steed, facts: SummonerSpellFacts{proficiencyBonus: 3, saveDC: 1}, scene: SummonScene{SpaceAvailable: true}}
	if r, ok := (SummonedRoster{}).Summon(cmd).(SummonedRejected); !ok || r.Rejection != (FormNotEligibleRejection{}) {
		t.Errorf("a steed with mount facts missing = %#v, want FormNotEligibleRejection", r)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md plan P10 (link cycles walked iteratively with a visited set) - a roster that already holds a ring of summoners (corrupt state that no transition builds) still ends the walk: createsCycle returns false for a creature outside the ring and true for one in it, with no recursion and no endless loop.
// Path: unhappy (boundary, state forgery).
func TestACorruptRingEndsTheCycleWalk(t *testing.T) {
	t.Parallel()
	ring := SummonedRoster{links: []SummonedLink{
		{creature: "a", summoner: "b", kind: SummonGiantInsect, presence: presenceInEncounter},
		{creature: "b", summoner: "a", kind: SummonGiantInsect, presence: presenceInEncounter},
	}}
	done := make(chan [2]bool, 1)
	go func() { done <- [2]bool{ring.createsCycle("z", "a"), ring.createsCycle("a", "b")} }()
	select {
	case got := <-done:
		if got != [2]bool{false, true} {
			t.Errorf("createsCycle = %v, want [false true]", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the cycle walk did not end on a ring [plan P10]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-040 - a window that is not set, or that lapsed, is not changed by time; the lapse is final.
// Path: unhappy (boundary).
func TestAWindowThatIsNotSetOrLapsedDoesNotChange(t *testing.T) {
	t.Parallel()
	lapsed := startedWindow().elapse(ControlWindowStart)
	if got := lapsed.elapse(time.Hour); got != lapsed || !got.Lapsed() || got.Left() != 0 {
		t.Errorf("a lapsed window after an hour = %+v, want it unchanged", got)
	}
	if got := (ControlWindow{}).elapse(time.Hour); !got.IsZero() {
		t.Errorf("a window that is not set after an hour = %+v, want it still unset", got)
	}
	if w := startedWindow().elapse(time.Minute); w.Left() != ControlWindowStart-time.Minute || w.Lapsed() {
		t.Errorf("a window after a minute = %+v", w)
	}
}
