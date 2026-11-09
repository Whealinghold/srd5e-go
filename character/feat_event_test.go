package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/feats-behaviour.md FR-063, FR-080 - the feat events carry the character id and no display text, form one closed family that is a character.Event, and expose what happened.
// Path: happy.
func TestFeatEventsAreCharacterEvents(t *testing.T) {
	t.Parallel()
	c := charAt(t, 5)
	feats := heldFeats(t, initiateFeat(), clericInitiate())
	expended := expend(t, feats, "magic_initiate").(character.FeatsChanged)
	renewed := renew(t, expended.Feats(), character.RenewOnLongRestFinished)
	replaced := replace(t, c, feats, "cleric", "guidance", "mending", gainedBy(t, c)).(character.FeatsChanged)
	var events []character.FeatEvent
	events = append(events, expended.Events()...)
	events = append(events, renewed.Events()...)
	events = append(events, replaced.Events()...)
	if len(events) != 3 {
		t.Fatalf("%d events", len(events))
	}
	var _ character.Event = events[0]
	if e, ok := events[0].(character.FeatUseExpended); !ok || e.Feat() != "magic_initiate" || e.CharacterID() != c.ID() {
		t.Errorf("FeatUseExpended = %#v", events[0])
	}
	if e, ok := events[1].(character.FeatUsesRenewed); !ok || !slices.Equal(e.Feats(), []content.FeatID{"magic_initiate"}) || e.Renewal() != character.RenewOnLongRestFinished || e.CharacterID() != c.ID() {
		t.Errorf("FeatUsesRenewed = %#v", events[1])
	}
	if e, ok := events[2].(character.FeatSpellReplaced); !ok || e.Old() != "guidance" || e.New() != "mending" || e.List() != "cleric" || e.CharacterID() != c.ID() {
		t.Errorf("FeatSpellReplaced = %#v", events[2])
	}
	taken := takeOK(t, charAt(t, 8), character.TakenFeats{}, alertFeat(), customOrigin(), character.NoFeatChoices{}, character.HeldFeatures{}).Events()[0].(character.FeatEvent)
	if e := taken.(character.FeatTaken); e.Feat() != "alert" || e.Category() != content.FeatOrigin || e.Source() != (character.CustomBackgroundFeat{}) || e.CharacterID() != charAt(t, 8).ID() {
		t.Errorf("FeatTaken = %#v", taken)
	}
	repl := replaceFeat(t, charAt(t, 8), heldFeats(t, alertFeat(), character.NoFeatChoices{}), "alert", savageFeat(), anyGrant(), character.NoFeatChoices{}, character.HeldFeatures{}).(character.FeatTakingChanged).Events()[0].(character.FeatEvent)
	if e := repl.(character.FeatReplaced); e.Old() != "alert" || e.New() != "savage_attacker" || e.CharacterID() != charAt(t, 8).ID() {
		t.Errorf("FeatReplaced = %#v", repl)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-057 - FeatUsesRenewed compares the character, the feats and the trigger, whichever way each differs.
// Path: per case.
func TestFeatUsesRenewedEqual(t *testing.T) {
	t.Parallel()
	mk := func(t *testing.T, id string, renewal character.FeatRenewal, fate bool) character.FeatUsesRenewed {
		t.Helper()
		feats := heldFeats(t, boonFeat("boon_of_fate"), singleOf(1))
		if !fate {
			feats = heldFeats(t, initiateFeat(), clericInitiate())
		}
		feat := content.FeatID("boon_of_fate")
		if !fate {
			feat = "magic_initiate"
		}
		c := charWithID(t, id, 5)
		spent := must(character.FeatUses{}.Expend(c, feats, feat)).(character.FeatsChanged).Feats()
		tr := must(character.FeatUses{}.Renew(c, spent, renewal)).(character.FeatsChanged)
		return tr.Events()[0].(character.FeatUsesRenewed)
	}
	base := mk(t, "pc-1", character.RenewOnLongRestFinished, true)
	if !base.Equal(mk(t, "pc-1", character.RenewOnLongRestFinished, true)) {
		t.Error("the same renewal must compare equal")
	}
	for name, other := range map[string]character.FeatUsesRenewed{
		"another trigger":   mk(t, "pc-1", character.RenewOnShortRestFinished, true),
		"another feat":      mk(t, "pc-1", character.RenewOnLongRestFinished, false),
		"another character": mk(t, "pc-2", character.RenewOnLongRestFinished, true),
	} {
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("%s: renewals compared equal", name)
		}
	}
}
