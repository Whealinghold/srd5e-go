package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FuzzMagicalEffectsConstructors feeds the constructors and rules of the magical-effect values host text and
// numbers. It asserts properties: nothing panics, a rejected value is the zero value with an error matching
// srderr.ErrInvalidInput, an accepted value reads back what was given, a cone width equals the distance inside
// the cone and is an error outside it, and a teleportation with a cost below zero is rejected.
func FuzzMagicalEffectsConstructors(f *testing.F) {
	f.Add("e1", "bless", "visual", "p1", 60, 15, 30, int8(1))
	f.Add("", "", "", "", 0, -1, -1, int8(0))
	f.Add(" ", "Bad Id", " ", " ", math.MaxInt, math.MaxInt, math.MinInt, int8(9))
	f.Add("a b", "ok_id", "hearing", "x", math.MinInt, math.MinInt, math.MaxInt, int8(-1))
	f.Fuzz(func(t *testing.T, effectID, spellID, sense, pending string, coneLength, distance, cost int, cover int8) {
		if id, err := combat.NewOngoingEffectID(effectID); err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !id.IsZero() {
				t.Fatalf("rejected effect id: %q, %v", id, err)
			}
		} else if id.String() != effectID {
			t.Fatalf("accepted effect id reads back %q, want %q", id, effectID)
		}
		if src, err := combat.SpellSource(content.SpellID(spellID)); err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || !src.IsZero() {
				t.Fatalf("rejected source: %v, %v", src, err)
			}
		} else if !src.IsMagical() || src.ID() != spellID {
			t.Fatalf("accepted source reads back %q magical %v", src.ID(), src.IsMagical())
		}
		if s, err := combat.NewIllusionSenseID(sense); err == nil {
			tr, err := combat.NewIllusionTraits([]combat.IllusionSenseID{s}, combat.NotInSpace{})
			if err != nil || tr.DeceivesSight() != (sense == "visual") {
				t.Fatalf("traits for %q: %v, sight %v", sense, err, tr.DeceivesSight())
			}
		}
		if id, err := combat.NewPendingEffectID(pending); err == nil {
			res := (combat.SimultaneousEffects{}).Order(combat.TurnHolderPlayer, []combat.PendingEffectID{id, id}, []combat.PendingEffectID{id})
			if _, ok := res.(combat.OrderSimultaneousEffectsRejected); !ok {
				t.Fatalf("a repeated pending id must be rejected, got %T", res)
			}
		}
		if cone, err := content.NewConeArea(coneLength); err == nil {
			w, err := combat.ConeWidthAt(cone, distance)
			inside := distance >= 0 && distance <= coneLength
			if inside && (err != nil || w != distance) || !inside && !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("ConeWidthAt(%d, %d) = %d, %v", coneLength, distance, w, err)
			}
		}
		cmd, err := combat.NewTeleportCommand(combat.TeleportFields{Teleporter: gid("mage"), Destination: combat.Unoccupied,
			Sight: combat.DestinationSightNotRequired, MovementCostFeet: cost})
		if (cost < 0) != errors.Is(err, srderr.ErrInvalidInput) {
			t.Fatalf("cost %d: error %v", cost, err)
		}
		if err == nil {
			if res, err := (combat.Teleportation{}).Resolve(cmd); err != nil || res.(combat.TeleportArrivedAtDestination).MovementCostFeet() != cost {
				t.Fatalf("cost %d: %v, %v", cost, res, err)
			}
		}
		if facts, err := combat.NewOtherLocationFacts(combat.InsideShape, []combat.Cover{combat.Cover(cover)}); err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) || combat.Cover(cover).Valid() {
				t.Fatalf("cover %d: %v", cover, err)
			}
		} else if combat.Cover(cover).Valid() != (len(facts.Lines()) == 1) {
			t.Fatalf("cover %d accepted but not valid", cover)
		}
	})
}
