package combat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-022, FR-043, FR-046, FR-057 (SRD 5.2.1, "Purple Worm", p. 317) - a swallow resolved with untrusted capacity, size, DC, die and modifier never panics; a command is an ErrInvalidInput error or a result; a held set never exceeds the capacity; damage from inside refuses a non-positive amount; the container's death always leaves the held set empty or trapping each occupant.
// Path: unhappy (boundary, state error).
func FuzzEnterContainerThenContainerDies(f *testing.F) {
	f.Add(1, 1, 3, 18, 3, 0, 5)
	f.Add(3, 2, 4, 19, 10, 2, 31)
	f.Add(0, 0, 0, 0, 0, 0, 0)
	f.Add(-1, 99, -4, -18, 21, -9, -5)
	f.Fuzz(func(t *testing.T, slots, size, occSize, dc, face, mod, damage int) {
		capacity, err := combat.NewContainmentCapacity(slots, map[content.CreatureSize]int{content.CreatureSize(size): 1})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("capacity error %v does not match ErrInvalidInput", err)
			}
			return
		}
		entry, err := combat.NewEntryBySave(ability.Dexterity, d20.DifficultyClass(dc), nil, false, 0, false)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("entry error %v does not match ErrInvalidInput", err)
			}
			return
		}
		terms, err := combat.NewContainmentTerms(combat.ContainmentTermsInput{Mode: combat.ContainmentModeSwallowed, MaxSize: content.SizeLarge, Capacity: capacity,
			Entry: entry, Imposed: []content.ConditionKind{content.ConditionBlinded, content.ConditionRestrained}, TotalCover: true,
			Suffocation: combat.SuffocationRuleNone, ActionKind: combat.ContainmentActionKindBonusAction,
			Regurgitation: combat.MustRegurgitation(30, 21, 5), DeathRule: combat.MustEscapeFromCorpse(20)})
		if err != nil {
			return
		}
		grapple := combat.MustGrapple(combat.MustCombatantID("worm"), combat.MustCombatantID("victim"), combat.MustGraspingHand(0), 15, 5)
		cmd, err := combat.NewEnterContainerCommand(combat.EnterContainerInput{Grapples: combat.MustGrapples([]combat.Grapple{grapple}),
			Container: combat.MustCombatantID("worm"), Occupant: combat.MustCombatantID("victim"), OccupantSize: content.CreatureSize(occSize), Terms: terms,
			OccupantConditions: combat.MustConditionSet([]combat.Condition{grapple.Condition()}, nil), Circumstance: combat.EntryCircumstanceNone,
			Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(mod))}})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("command error %v does not match ErrInvalidInput", err)
			}
			return
		}
		use := gapMust(combat.NewEnterContainer(dicetest.NewFixedRoller(dieFace(face))))
		res, err := use.Run(context.Background(), cmd)
		if err != nil {
			return
		}
		held, ok := res.(combat.EnterContainerContained)
		if !ok {
			return // Resisted, DamagedOnly, Rejected and Invalid are results
		}
		if held.Containments().Len() < 1 || held.Containments().Len() > slots {
			t.Fatalf("held %d creatures with %d slots", held.Containments().Len(), slots)
		}
		if _, err := held.Containments().RecordDamageFromInside(combat.MustCombatantID("victim"), damage); damage < 0 && err == nil {
			t.Fatalf("a damage amount of %d was accepted", damage)
		}
		rel, err := combat.NewReleaseOccupantsCommand(combat.ReleaseOccupantsInput{Containments: held.Containments(),
			OccupantConditions: map[combat.CombatantID]combat.ConditionSet{combat.MustCombatantID("victim"): held.Conditions()},
			Trigger:            combat.ContainerDied{Container: combat.MustCombatantID("worm")}})
		if err != nil {
			t.Fatalf("a container's death command was refused: %v", err)
		}
		out, err := combat.NewReleaseOccupants().Run(context.Background(), rel)
		if err != nil {
			t.Fatalf("a container's death errored: %v", err)
		}
		if r, ok := out.(combat.ReleaseOccupantsReleased); ok && r.Containments().Len() > 0 && len(r.Trapped()) == 0 {
			t.Fatalf("after the death %d creatures are still held and none is trapped", r.Containments().Len())
		}
	})
}
