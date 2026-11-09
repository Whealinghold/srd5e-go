package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func expend(t *testing.T, feats character.TakenFeats, id content.FeatID) character.FeatsTransition {
	t.Helper()
	tr, err := character.FeatUses{}.Expend(charAt(t, 20), feats, id)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func renew(t *testing.T, feats character.TakenFeats, r character.FeatRenewal) character.FeatsChanged {
	t.Helper()
	tr, err := character.FeatUses{}.Renew(charAt(t, 20), feats, r)
	if err != nil {
		t.Fatal(err)
	}
	return tr.(character.FeatsChanged)
}

func fateAndInitiate(t *testing.T) character.TakenFeats {
	t.Helper()
	return heldFeats(t, boonFeat("boon_of_fate"), singleOf(ability.Wisdom), initiateFeat(), clericInitiate())
}

// Requirement: doc/specs/feats-behaviour.md FR-032, FR-050, FR-051, Table U (SRD 5.2.1, "Magic Initiate", p. 87; "Boon of Fate", p. 88) - a feat with a limited use starts available, expending it gives FeatUseIsExpended and one FeatUseExpended, expending it again is rejected; a feat not held and a held feat with no limited use are rejected.
// Path: per case.
func TestFeatUsesExpend(t *testing.T) {
	t.Parallel()
	feats := fateAndInitiate(t)
	if _, ok := feats.UseStateOf("boon_of_fate").(character.FeatUseIsAvailable); !ok {
		t.Fatalf("a new Improve Fate is available: %v", feats.UseStateOf("boon_of_fate"))
	}
	if _, ok := feats.UseStateOf("magic_initiate").(character.FeatUseIsAvailable); !ok {
		t.Fatalf("a new free cast is available: %v", feats.UseStateOf("magic_initiate"))
	}
	one := expend(t, feats, "boon_of_fate").(character.FeatsChanged)
	if _, ok := one.Feats().UseStateOf("boon_of_fate").(character.FeatUseIsExpended); !ok {
		t.Errorf("state = %v", one.Feats().UseStateOf("boon_of_fate"))
	}
	if _, ok := one.Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable); !ok {
		t.Error("expending one feat's use must leave another's alone")
	}
	if ev := one.Events(); len(ev) != 1 || ev[0] != (character.FeatUseExpended{}) && ev[0].(character.FeatUseExpended).Feat() != "boon_of_fate" || ev[0].CharacterID() != charAt(t, 20).ID() {
		t.Errorf("events = %#v", ev)
	}
	if _, ok := feats.UseStateOf("boon_of_fate").(character.FeatUseIsAvailable); !ok {
		t.Error("Expend altered its input")
	}
	for _, c := range []struct {
		name  string
		feats character.TakenFeats
		id    content.FeatID
		want  character.FeatRejection
	}{
		{"expended twice", one.Feats(), "boon_of_fate", character.FeatUseAlreadyExpended{Feat: "boon_of_fate"}},
		{"not held", feats, "boon_of_truesight", character.FeatNotHeld{Feat: "boon_of_truesight"}},
		{"held with no limited use", heldFeats(t, alertFeat(), character.NoFeatChoices{}), "alert", character.FeatHasNoLimitedUse{Feat: "alert"}},
	} {
		rj, ok := expend(t, c.feats, c.id).(character.FeatsRejected)
		if !ok || rj.Rejection != c.want {
			t.Errorf("%s: %#v, want %#v", c.name, rj, c.want)
		}
	}
	if _, err := (character.FeatUses{}).Expend(character.Character{}, feats, "alert"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v", err)
	}
	if _, err := (character.FeatUses{}).Expend(charAt(t, 1), feats, ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid id: %v", err)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-032, FR-052, Table M "free cast, rests", Table U "renewal triggers" - the free cast of Magic Initiate renews only on a finished Long Rest; Improve Fate on Initiative, a Short Rest and a Long Rest; a renewal lists the feats renewed in one event.
// Path: per case.
func TestFeatUsesRenew(t *testing.T) {
	t.Parallel()
	feats := fateAndInitiate(t)
	spent := expend(t, expend(t, feats, "boon_of_fate").(character.FeatsChanged).Feats(), "magic_initiate").(character.FeatsChanged).Feats()
	for _, c := range []struct {
		renewal      character.FeatRenewal
		fate, intiat bool // available afterwards
	}{
		{character.RenewOnInitiativeRolled, true, false},
		{character.RenewOnShortRestFinished, true, false},
		{character.RenewOnLongRestFinished, true, true},
	} {
		ch := renew(t, spent, c.renewal)
		_, fate := ch.Feats().UseStateOf("boon_of_fate").(character.FeatUseIsAvailable)
		_, init := ch.Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable)
		if fate != c.fate || init != c.intiat {
			t.Errorf("%v: fate %v, initiate %v", c.renewal, fate, init)
		}
		ev := ch.Events()
		want := []content.FeatID{"boon_of_fate"}
		if c.intiat {
			want = []content.FeatID{"boon_of_fate", "magic_initiate"}
		}
		got, ok := ev[0].(character.FeatUsesRenewed)
		if len(ev) != 1 || !ok || !slices.Equal(got.Feats(), want) || got.Renewal() != c.renewal || got.CharacterID() != charAt(t, 20).ID() {
			t.Errorf("%v: events %#v", c.renewal, ev)
		}
		got.Feats()[0] = ""
		if got.Feats()[0] == "" || !got.Equal(got) || got.Equal(character.FeatUsesRenewed{}) {
			t.Error("Feats must return a copy and Equal must compare the renewal")
		}
	}
	if _, ok := spent.UseStateOf("boon_of_fate").(character.FeatUseIsExpended); !ok {
		t.Error("Renew altered its input")
	}
	// nothing expended, or nothing listing the trigger: the same state and no events
	for name, ch := range map[string]character.FeatsChanged{
		"nothing expended":      renew(t, feats, character.RenewOnLongRestFinished),
		"no feats":              renew(t, character.TakenFeats{}, character.RenewOnLongRestFinished),
		"a trigger not listed":  renew(t, expend(t, feats, "magic_initiate").(character.FeatsChanged).Feats(), character.RenewOnShortRestFinished),
		"an initiative trigger": renew(t, expend(t, feats, "magic_initiate").(character.FeatsChanged).Feats(), character.RenewOnInitiativeRolled),
	} {
		if len(ch.Events()) != 0 {
			t.Errorf("%s: events %#v", name, ch.Events())
		}
	}
	if _, ok := renew(t, expend(t, feats, "magic_initiate").(character.FeatsChanged).Feats(), character.RenewOnShortRestFinished).Feats().UseStateOf("magic_initiate").(character.FeatUseIsExpended); !ok {
		t.Error("a Short Rest must not renew a free cast")
	}
	if _, err := (character.FeatUses{}).Renew(character.Character{}, feats, character.RenewOnLongRestFinished); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v", err)
	}
	if _, err := (character.FeatUses{}).Renew(charAt(t, 1), feats, 0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid renewal: %v", err)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-032 (SRD 5.2.1, "Magic Initiate", p. 87) - each Magic Initiate instance has its own free cast: expending twice spends both in turn, and a third is rejected.
// Path: unhappy (rule interaction).
func TestFeatUsesExpendTwoInstances(t *testing.T) {
	t.Parallel()
	feats := heldFeats(t, initiateFeat(), clericInitiate(), initiateFeat(), wizardInitiate())
	one := expend(t, feats, "magic_initiate").(character.FeatsChanged).Feats()
	if _, ok := one.UseStateOf("magic_initiate").(character.FeatUseIsAvailable); !ok {
		t.Error("one instance is still available")
	}
	two := expend(t, one, "magic_initiate").(character.FeatsChanged).Feats()
	if _, ok := two.UseStateOf("magic_initiate").(character.FeatUseIsExpended); !ok {
		t.Error("both are expended")
	}
	if _, ok := expend(t, two, "magic_initiate").(character.FeatsRejected); !ok {
		t.Error("a third expenditure must be rejected")
	}
	if character.TakenFeats.UseStateOf(character.TakenFeats{}, "magic_initiate") != nil {
		t.Error("a feat not held has no use state")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-032 - with several Magic Initiate instances the first available free cast is expended, in the order the feats were taken.
// Path: unhappy (rule interaction).
func TestFeatUsesExpendTakesTheFirstAvailableInstance(t *testing.T) {
	t.Parallel()
	druid := miOf("druid", ability.Wisdom, []string{"guidance", "shillelagh"}, "cure_wounds")
	feats := heldFeats(t, initiateFeat(), clericInitiate(), initiateFeat(), druid, initiateFeat(), wizardInitiate())
	states := func(f character.TakenFeats) [3]bool {
		var out [3]bool
		for i, t := range f.All() {
			_, out[i] = t.Use().(character.FeatUseIsExpended)
		}
		return out
	}
	step := feats
	for i, want := range [][3]bool{{true, false, false}, {true, true, false}, {true, true, true}} {
		step = expend(t, step, "magic_initiate").(character.FeatsChanged).Feats()
		if got := states(step); got != want {
			t.Fatalf("after %d expenditures: %v, want %v", i+1, got, want)
		}
	}
}
