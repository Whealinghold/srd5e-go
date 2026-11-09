package character_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/class-bard.md section 5.2 (events), FR-037, FR-116 - every Bard event is an Event and a BardEvent carrying the Identifier it concerns, the family is closed (an exhaustive switch names all five), and each exposes what it recorded.
// Path: per case (happy).
func TestBardEventsAreOneClosedFamily(t *testing.T) {
	t.Parallel()
	granted, _ := character.GrantBardicInspiration(brdID, character.MustBardLevel(5), brdPool(t, 3, 3), brdMod(t, 16), character.InspirationRecipientOther{})
	plan, _, _ := character.PlanBardicInspirationUse(brdID, granted.Grant(), 0, brdAbility(t, 5, 0, 15))
	applied := must(character.ApplyBardicInspirationDie(plan, 3))
	restored, _ := character.RecoverBardicInspiration(brdID, character.MustBardLevel(5), brdPool(t, 3, 1), brdMod(t, 16), character.BardRecoveryRest{Kind: character.LongRest})
	skills := []content.Skill{content.SkillArcana, content.SkillHistory}
	gain, _ := character.ChooseBardExpertise(brdID, skills, must(character.NewSkillProficiencies(skills, nil)))
	events := []character.BardEvent{granted.Event(), applied.Event(), restored.Event(), gain.Event(), mustExpired(t)}
	seen := map[string]bool{}
	for _, e := range events {
		var _ character.Event = e
		if e.CharacterID() != brdID {
			t.Errorf("%T concerns %v, want %v", e, e.CharacterID(), brdID)
		}
		switch v := e.(type) {
		case character.BardicInspirationGranted:
			seen["granted"] = v.Die() == character.BardicD8 && v.Remaining() == 2
		case character.BardicInspirationExpended:
			seen["expended"] = v.Die() == character.BardicD8 && v.Rolled() == 3 && v.Feature() == "bardic_inspiration"
		case character.BardicInspirationExpired:
			seen["expired"] = v.Die() == character.BardicD6
		case character.BardicInspirationRestored:
			seen["restored"] = v.Uses() == 2 && v.Remaining() == 3
		case character.BardExpertiseGained:
			seen["expertise"] = slices.Equal(v.Skills(), skills)
		}
	}
	for _, name := range []string{"granted", "expended", "expired", "restored", "expertise"} {
		if !seen[name] {
			t.Errorf("event %q missing or wrong", name)
		}
	}
	got := gain.Event().Skills()
	got[0] = content.SkillStealth
	if gain.Event().Skills()[0] != content.SkillArcana {
		t.Error("Skills returns the event's own slice")
	}
}

// mustExpired builds the BardicInspirationExpired event through the use case that produces it.
func mustExpired(t *testing.T) character.BardicInspirationExpired {
	t.Helper()
	uc := must(character.NewUseBardicInspiration(dicetest.NewFixedRoller()))
	res := must(uc.Run(t.Context(), character.UseBardicInspirationCommand{ID: brdID, Grant: must(character.NewBardicInspirationGrant(character.BardicD6)), Elapsed: 2 * time.Hour, Failed: brdAbility(t, 5, 0, 15)}))
	return *res.(character.UseBardicInspirationRejected).Expired
}
