package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func replaceFeat(t *testing.T, c character.Character, feats character.TakenFeats, old string, feat content.Feat, grant character.FeatGrant, choices character.FeatChoices, held character.HeldFeatures) character.FeatTaking {
	t.Helper()
	tk, err := character.FeatReplacement{}.Replace(c, feats, content.FeatID(old), feat, grant, choices, held, miSpells(t))
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	return tk
}

// Requirement: doc/specs/feats-behaviour.md FR-082 (SRD 5.2.1, "Fighting Style" (Fighter), p. 47; "Wish", p. 175) - replacing a Fighting Style feat by another removes the old effects, emits FeatReplaced then FeatTaken, and sees repeatability without the old feat.
// Path: happy.
func TestFeatReplacementOfAFightingStyle(t *testing.T) {
	t.Parallel()
	c := charAt(t, 8, 14, 14, 10, 10, 10, 10)
	held := fightingStyleHeld(t)
	style := grantOf(character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}, character.FeatFromCategory{Category: content.FeatFightingStyle})
	feats := takeOK(t, c, character.TakenFeats{}, archeryFeat(), style, character.NoFeatChoices{}, held).Feats()
	tk := replaceFeat(t, c, feats, "archery", defenseFeat(), style, character.NoFeatChoices{}, held).(character.FeatTakingChanged)
	ev := tk.Events()
	if len(ev) != 2 || ev[0] != (character.FeatReplaced{}) && ev[0].(character.FeatReplaced).Old() != "archery" || ev[0].(character.FeatReplaced).New() != "defense" ||
		ev[0].CharacterID() != c.ID() || ev[1].(character.FeatTaken).Feat() != "defense" {
		t.Errorf("events = %#v", ev)
	}
	if tk.Feats().Holds("archery") || !tk.Feats().Holds("defense") || tk.Feats().Len() != 1 {
		t.Errorf("feats = %v", tk.Feats().All())
	}
	if !feats.Holds("archery") {
		t.Error("Replace altered its input")
	}
	// the new feat repeating the old one is accepted, because the old one is removed first
	same := replaceFeat(t, c, feats, "archery", archeryFeat(), style, character.NoFeatChoices{}, held)
	if _, ok := same.(character.FeatTakingChanged); !ok {
		t.Errorf("a feat replaced by itself: %#v", same)
	}
}

// Requirement: per case
// Path: per case.
func TestFeatReplacementRejections(t *testing.T) {
	t.Parallel()
	c := charAt(t, 20, 14, 14, 10, 10, 10, 10)
	held := fightingStyleHeld(t)
	style := grantOf(character.ClassFeatureFeat{Class: "fighter", Feature: "fighter_fighting_style"}, character.FeatFromCategory{Category: content.FeatFightingStyle})
	asi := must(character.NewAbilityScoreImprovementChoices(character.ImprovementOneByTwo, ability.Strength, 0))
	feats := heldFeats(t, archeryFeat(), character.NoFeatChoices{}, asiFeat(), asi, initiateFeat(), clericInitiate(), alertFeat(), character.NoFeatChoices{})
	for _, tc := range []struct {
		name, requirement, path string
		old                     string
		feat                    content.Feat
		grant                   character.FeatGrant
		choices                 character.FeatChoices
		want                    character.FeatRejection
	}{
		{"the old feat is not held", featsSpec + " FR-082", "unhappy (state error)", "defense", defenseFeat(), style, character.NoFeatChoices{}, character.FeatNotHeld{Feat: "defense"}},
		{"the old feat gave an increase", featsSpec + " FR-082, Q20", "unhappy (state error)", "ability_score_improvement", archeryFeat(), anyGrant(), character.NoFeatChoices{}, character.FeatNotReplaceable{Feat: "ability_score_improvement"}},
		{"the new feat is outside the grant", featsSpec + " FR-082, FR-083", "unhappy (state error)", "archery", alertFeat(), style, character.NoFeatChoices{}, character.FeatOutsideGrant{Feat: "alert"}},
		{"the new feat is held and not repeatable", featsSpec + " FR-082, FR-083", "unhappy (state error)", "archery", alertFeat(), anyGrant(), character.NoFeatChoices{}, character.FeatNotRepeatable{Feat: "alert"}},
		{"the new feat has no behaviour", featsSpec + " FR-082, FR-018", "unhappy (missing content)", "archery", featOf("house_feat", content.FeatGeneral, false), anyGrant(), character.NoFeatChoices{}, character.FeatBehaviourUnknown{Feat: "house_feat"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tk := replaceFeat(t, c, feats, tc.old, tc.feat, tc.grant, tc.choices, held)
			if got := tk.(character.FeatTakingRejected).Rejection; got != tc.want {
				t.Errorf("= %#v, want %#v [%s]", got, tc.want, tc.requirement)
			}
		})
	}
	// a Magic Initiate replaced loses its use state; Wish replaces by any feat
	expended := (character.FeatUses{})
	tr, err := expended.Expend(c, feats, "magic_initiate")
	if err != nil {
		t.Fatal(err)
	}
	used := tr.(character.FeatsChanged).Feats()
	got := replaceFeat(t, c, used, "magic_initiate", savageFeat(), anyGrant(), character.NoFeatChoices{}, held).(character.FeatTakingChanged)
	if got.Feats().Holds("magic_initiate") || got.Feats().UseStateOf("magic_initiate") != nil || !got.Feats().Holds("savage_attacker") {
		t.Errorf("feats = %v", got.Feats().All())
	}
	// a new feat that fails a prerequisite gives that rejection and no change
	if rj := replaceFeat(t, charAt(t, 3), heldFeats(t, alertFeat(), character.NoFeatChoices{}), "alert", asiFeat(), anyGrant(), asi, held).(character.FeatTakingRejected).Rejection; rj.String() != "feat_prerequisite_not_met" {
		t.Errorf("rejection = %#v", rj)
	}
	if _, err := (character.FeatReplacement{}).Replace(c, feats, "", alertFeat(), anyGrant(), character.NoFeatChoices{}, held, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid old id: %v", err)
	}
	if _, err := (character.FeatReplacement{}).Replace(character.Character{}, feats, "alert", alertFeat(), anyGrant(), character.NoFeatChoices{}, held, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v", err)
	}
}
