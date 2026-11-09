package character_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-cleric.md FR-021, FR-044, FR-098, FR-099, FR-125, FR-134, FR-138, table 4.14.3 AV-01, AV-02, AC-01 to AC-20 (SRD 5.2.1, "Cleric Features", p. 36; "Level 3: Life Domain Spells", p. 40) - every
// advance from the level above is applied with one ClericAdvanced event carrying the features gained, the pending choices, the Channel Divinity uses gained (2 at level 2, 1 at 6 and 18) and the Life Domain spells
// newly granted (the level 5, 7 and 9 rows); the uses rise with the maximum, never above it.
// Path: per case (AV-01, AV-02, AC-01 to AC-20): happy / unhappy (boundary: first and last level).
func TestAdvanceClericPayload(t *testing.T) {
	t.Parallel()
	p := clcProgression(t)
	life := clcLifeDomain(t)
	features := map[int]string{1: "cleric_spellcasting cleric_divine_order", 2: "cleric_channel_divinity", 3: "cleric_cleric_subclass", 4: "cleric_ability_score_improvement",
		5: "cleric_sear_undead", 7: "cleric_blessed_strikes", 10: "cleric_divine_intervention", 14: "cleric_improved_blessed_strikes", 19: "cleric_epic_boon", 20: "cleric_greater_divine_intervention"}
	gained := map[int]int{2: 2, 6: 1, 18: 1}
	spells := map[int][]content.SpellID{5: {"mass_healing_word", "revivify"}, 7: {"aura_of_life", "death_ward"}, 9: {"greater_restoration", "mass_cure_wounds"}}
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			// The character already holds the new Cleric level and the resources of the level before (none at level 1).
			c := clcCleric(t, n)
			if n > 1 {
				prev := p.StatsAt(character.MustClericLevel(n - 1)).ChannelDivinity()
				c = clcWith(t, c, clcUses(prev, prev))
			}
			res, err := character.AdvanceCleric(c, n-1, p, life)
			if err != nil {
				t.Fatal(err)
			}
			r, ok := res.(character.AdvanceClericApplied)
			if !ok {
				t.Fatalf("result = %#v", res)
			}
			ev := r.Events()
			if len(ev) != 1 {
				t.Fatalf("events = %v, want exactly one ClericAdvanced", ev)
			}
			adv := ev[0].(character.ClericAdvanced)
			var wantFeatures []string
			if f, ok := features[n]; ok {
				wantFeatures = strings.Fields(f)
			}
			var gotFeatures []string
			for _, f := range r.FeaturesGained() {
				gotFeatures = append(gotFeatures, string(f))
			}
			wantMax := p.StatsAt(character.MustClericLevel(n)).ChannelDivinity()
			wantPending := 0
			if n == 1 || n == 7 {
				wantPending = 1
			}
			if !slices.Equal(gotFeatures, wantFeatures) || adv.Level() != n || adv.ChannelDivinityUsesGained() != gained[n] || len(r.PendingChoices()) != wantPending ||
				!slices.Equal(r.AlwaysPreparedSpells(), append([]content.SpellID{}, spells[n]...)) || !slices.Equal(adv.AlwaysPreparedSpellsGranted(), r.AlwaysPreparedSpells()) ||
				r.Resources().ChannelDivinityMaximum() != wantMax || r.Resources().ChannelDivinityRemaining() != wantMax {
				t.Errorf("level %d = features %v, gained %d, pending %v, spells %v, resources %+v", n, gotFeatures, adv.ChannelDivinityUsesGained(), r.PendingChoices(), r.AlwaysPreparedSpells(), r.Resources())
			}
			if clcResourcesOf(t, r.Character()) != r.Resources() {
				t.Error("the Character does not hold the new resources")
			}
			if _, held, _ := character.ReadFeatureState(r.Character().FeatureStates(), character.ClericChoicesKey()); !held {
				t.Error("the Character holds no ClericChoices")
			}
		})
	}
}

// Requirement: doc/specs/class-cleric.md FR-044, D-2 - the remaining uses rise by the same number as the maximum, never above it: 1 of 2 becomes 2 of 3 at level 6; a state with a spent use keeps the difference; no Life Domain
// subclass grants no spell.
// Path: per case (unhappy: boundary, rule interaction).
func TestAdvanceClericKeepsExpendedUses(t *testing.T) {
	t.Parallel()
	p := clcProgression(t)
	res := must(character.AdvanceCleric(clcWith(t, clcCleric(t, 6), clcUses(2, 1)), 5, p, nil)).(character.AdvanceClericApplied)
	if got := res.Resources(); got.ChannelDivinityMaximum() != 3 || got.ChannelDivinityRemaining() != 2 || len(res.AlwaysPreparedSpells()) != 0 {
		t.Errorf("1 of 2 at level 6 = %+v, spells %v", got, res.AlwaysPreparedSpells())
	}
	res = must(character.AdvanceCleric(clcWith(t, clcCleric(t, 18), clcUses(3, 0)), 17, p, nil)).(character.AdvanceClericApplied)
	if got := res.Resources(); got.ChannelDivinityMaximum() != 4 || got.ChannelDivinityRemaining() != 1 {
		t.Errorf("0 of 3 at level 18 = %+v", got)
	}
	// A stale state above the new maximum is capped.
	res = must(character.AdvanceCleric(clcWith(t, clcCleric(t, 3), clcUses(2, 2)), 2, p, nil)).(character.AdvanceClericApplied)
	if got := res.Resources(); got.ChannelDivinityMaximum() != 2 || got.ChannelDivinityRemaining() != 2 {
		t.Errorf("level 3 = %+v", got)
	}
}

// Requirement: doc/specs/class-cleric.md FR-117, FR-122, FR-138 - the Cleric levels count, not the total level (a Fighter 5 / Cleric 2 gains Channel Divinity at its second Cleric level); a character with no Cleric levels is
// AdvanceClericNotACleric; a zero Character or progression, a previous level that is not one below, and a foreign state under a Cleric key are errors matching srderr.ErrInvalidInput; a multiclass first Cleric level
// supplies the initial states.
// Path: per case (unhappy: rule interaction, state error, invalid input).
func TestAdvanceClericMulticlassAndRefusals(t *testing.T) {
	t.Parallel()
	p := clcProgression(t)
	multi := must(character.AdvanceCleric(clcChar(t, cl("fighter", 5), cl("cleric", 2)), 1, p, nil)).(character.AdvanceClericApplied)
	if multi.Resources().ChannelDivinityMaximum() != 2 || multi.Resources().ChannelDivinityRemaining() != 2 {
		t.Errorf("Fighter 5 / Cleric 2 = %+v", multi.Resources())
	}
	first := must(character.AdvanceCleric(clcChar(t, cl("fighter", 5), cl("cleric", 1)), 0, p, nil)).(character.AdvanceClericApplied)
	if !first.Resources().Equal(character.InitialClericResources(0)) || len(first.PendingChoices()) != 1 || first.PendingChoices()[0] != character.ClericPendingDivineOrder {
		t.Errorf("first multiclass level = %+v, %v", first.Resources(), first.PendingChoices())
	}
	res, err := character.AdvanceCleric(fgtFighter(t, 5), 0, p, nil)
	if _, ok := res.(character.AdvanceClericNotACleric); err != nil || !ok {
		t.Errorf("Fighter = %#v, %v", res, err)
	}
	for name, call := range map[string]func() error{
		"zero character": func() error { _, e := character.AdvanceCleric(character.Character{}, 0, p, nil); return e },
		"zero progression": func() error {
			_, e := character.AdvanceCleric(clcCleric(t, 1), 0, character.ClericProgression{}, nil)
			return e
		},
		"previous not one below": func() error { _, e := character.AdvanceCleric(clcCleric(t, 3), 3, p, nil); return e },
		"previous two below":     func() error { _, e := character.AdvanceCleric(clcCleric(t, 3), 1, p, nil); return e },
		"foreign resources": func() error {
			c := must(character.WithFeatureState(clcCleric(t, 2), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_resources"), must(character.NewChosenSubclass("life_domain"))))
			_, e := character.AdvanceCleric(c, 1, p, nil)
			return e
		},
		"foreign choices": func() error {
			c := must(character.WithFeatureState(clcCleric(t, 2), character.MustFeatureStateKey[character.ChosenSubclass]("cleric_choices"), must(character.NewChosenSubclass("life_domain"))))
			_, e := character.AdvanceCleric(c, 1, p, nil)
			return e
		},
	} {
		wantInvalid(t, call(), name)
	}
}

// Requirement: doc/specs/class-cleric.md FR-042, FR-043, FR-091, FR-092, FR-095, FR-125, table 4.11.2 (SRD 5.2.1, "Level 2: Channel Divinity", "Level 20: Greater Divine Intervention", p. 37-38) - ApplyClericRest returns the
// resources after the rest and the events it produced: ClericChannelDivinityRegained with the uses regained and DivineInterventionRecharged when Divine Intervention became available again; nothing when nothing changed.
// Path: per case (happy / unhappy: boundary).
func TestApplyClericRest(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	used := character.DivineInterventionUnavailableUntilLongRest{}
	for _, c := range []struct {
		name       string
		from       character.ClericResources
		kind       character.RestKind
		regained   int
		recharged  bool
		eventsWant int
	}{
		{"short rest with a use spent", must(character.NewClericResources(3, 2, used)), character.ShortRest, 1, false, 1},
		{"short rest with nothing spent", must(character.NewClericResources(3, 3, used)), character.ShortRest, 0, false, 0},
		{"long rest", must(character.NewClericResources(3, 0, used)), character.LongRest, 3, true, 2},
		{"long rest, all rested", character.InitialClericResources(3), character.LongRest, 0, false, 0},
		{"long rest, one Long Rest to go", must(character.NewClericResources(4, 4, character.DivineInterventionRecharging{LongRestsRemaining: 1})), character.LongRest, 0, true, 1},
		{"long rest, two to go", must(character.NewClericResources(4, 4, character.DivineInterventionRecharging{LongRestsRemaining: 2})), character.LongRest, 0, false, 0},
	} {
		got := character.ApplyClericRest(id, c.from, c.kind)
		if got.Resources() != c.from.AfterRest(c.kind) || len(got.Events()) != c.eventsWant {
			t.Errorf("%s: %+v, %d events; want %d events", c.name, got.Resources(), len(got.Events()), c.eventsWant)
			continue
		}
		var regained int
		var recharged bool
		for _, e := range got.Events() {
			switch v := e.(type) {
			case character.ClericChannelDivinityRegained:
				regained = v.Uses()
			case character.DivineInterventionRecharged:
				recharged = v.CharacterID() == id
			}
		}
		if regained != c.regained || recharged != c.recharged {
			t.Errorf("%s: regained %d recharged %v; want %d %v", c.name, regained, recharged, c.regained, c.recharged)
		}
	}
}
