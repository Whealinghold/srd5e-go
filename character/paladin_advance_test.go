package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// pldAdvanced applies AdvancePaladin to c (already at the new level) from previous.
func pldAdvanced(t *testing.T, c character.Character, previous int, subclass content.SubclassID) character.AdvancePaladinApplied {
	t.Helper()
	res, err := character.AdvancePaladin(c, previous, pldProgression(t), subclass)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := res.(character.AdvancePaladinApplied)
	if !ok {
		t.Fatalf("AdvancePaladin = %#v", res)
	}
	return a
}

// Requirement: doc/specs/class-paladin.md FR-066, FR-098, Table G (SRD 5.2.1, "Level 1: Lay On Hands", p. 53; "Level 2: Paladin’s Smite", "Level 3: Channel Divinity", p. 54; "Level 5: Faithful
// Steed", p. 55; "Level 20: Holy Nimbus", p. 57) - AdvancePaladin raises Lay On Hands by 5, gives Channel Divinity its 2 uses at level 3 and the third at 11, makes the feature gained at the level
// available, reports the features gained, the pending choice and the spells newly granted, and refills nothing else (a spent Faithful Steed stays spent at level 6).
// Path: per case (Table G): happy / unhappy (boundary, empty case, state error).
func TestAdvancePaladinTableG(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label       string
		from, to    int
		subclass    content.SubclassID
		state       character.PaladinStateParts
		wantLay     int
		wantChannel int
		wantGained  int
		features    []content.FeatureID
		spells      []content.SpellID
		pending     int
	}{
		{"level 2", 1, 2, "", character.PaladinStateParts{LayOnHandsRemaining: 5}, 10, 0, 0, []content.FeatureID{"paladin_fighting_style", "paladin_paladins_smite"}, []content.SpellID{"divine_smite"}, 1},
		{"level 3, no subclass yet", 2, 3, "", character.PaladinStateParts{LayOnHandsRemaining: 10}, 15, 2, 2, []content.FeatureID{"paladin_channel_divinity", "paladin_paladin_subclass"}, nil, 0},
		{"level 3, subclass chosen at once", 2, 3, character.OathOfDevotionSubclassID, character.PaladinStateParts{LayOnHandsRemaining: 10}, 15, 2, 2, []content.FeatureID{"paladin_channel_divinity", "paladin_paladin_subclass"},
			[]content.SpellID{"protection_from_evil_and_good", "shield_of_faith"}, 0},
		{"level 5", 4, 5, "", character.PaladinStateParts{LayOnHandsRemaining: 20, ChannelDivinityRemaining: 1}, 25, 1, 0, []content.FeatureID{"paladin_extra_attack", "paladin_faithful_steed"}, []content.SpellID{"find_steed"}, 0},
		{"level 7", 6, 7, character.OathOfDevotionSubclassID, character.PaladinStateParts{LayOnHandsRemaining: 30}, 35, 0, 0, []content.FeatureID(nil), nil, 0},
		{"level 11", 10, 11, "", character.PaladinStateParts{LayOnHandsRemaining: 50, ChannelDivinityRemaining: 0}, 55, 1, 1, []content.FeatureID{"paladin_radiant_strikes"}, nil, 0},
		{"level 13", 12, 13, "", character.PaladinStateParts{LayOnHandsRemaining: 60}, 65, 0, 0, []content.FeatureID(nil), nil, 0},
		{"level 20", 19, 20, character.OathOfDevotionSubclassID, character.PaladinStateParts{LayOnHandsRemaining: 95}, 100, 0, 0, []content.FeatureID(nil), nil, 0},
	} {
		prev := pldProgression(t).ChannelDivinityMaximum(character.MustPaladinLevel(c.from))
		parts := c.state
		parts.ChannelDivinityRemaining = min(parts.ChannelDivinityRemaining, prev)
		if c.from == 2 || c.from == 1 {
			parts.ChannelDivinityRemaining = 0
		}
		start := pldWith(t, pldPaladin(t, c.to), pldState(t, c.from, parts))
		before := start
		res := pldAdvanced(t, start, c.from, c.subclass)
		s := res.State()
		if s.LayOnHandsRemaining() != c.wantLay || s.ChannelDivinityRemaining() != c.wantChannel || !pldStateOf(t, res.Character()).Equal(s) {
			t.Errorf("%s: lay %d, channel %d; want %d, %d", c.label, s.LayOnHandsRemaining(), s.ChannelDivinityRemaining(), c.wantLay, c.wantChannel)
		}
		ev, ok := res.Events()[0].(character.PaladinAdvanced)
		if !ok || len(res.Events()) != 1 || ev.Level() != c.to || ev.ChannelDivinityUsesGained() != c.wantGained || !slices.Equal(res.FeaturesGained(), c.features) && (len(c.features) != 0 || len(res.FeaturesGained()) != 0) ||
			!slices.Equal(res.AlwaysPreparedSpells(), c.spells) && (len(c.spells) != 0 || len(res.AlwaysPreparedSpells()) != 0) || len(res.PendingChoices()) != c.pending || ev.CharacterID() != start.ID() {
			t.Errorf("%s: event = %+v, features %v, spells %v, pending %v", c.label, ev, res.FeaturesGained(), res.AlwaysPreparedSpells(), res.PendingChoices())
		}
		if !start.Equal(before) {
			t.Errorf("%s: the input changed", c.label)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-098 - a newly gained free cast starts available, a spent use stays spent when an unrelated level is gained, nothing else is refilled, and a
// Character holding no state gets the full pool of the new level.
// Path: unhappy (rule interaction: no refill beyond the gain).
func TestAdvancePaladinKeepsSpentUses(t *testing.T) {
	t.Parallel()
	spent := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 3, ChannelDivinityRemaining: 0, PaladinsSmite: character.LongRestUseSpent, FaithfulSteed: character.LongRestUseSpent, HolyNimbus: character.LongRestUseSpent})
	res := pldAdvanced(t, pldWith(t, pldPaladin(t, 6), spent), 5, "")
	s := res.State()
	if s.FreeCast(character.FreeCastPaladinsSmite) != character.LongRestUseSpent || s.FreeCast(character.FreeCastFaithfulSteed) != character.LongRestUseSpent || s.HolyNimbusUse() != character.LongRestUseSpent ||
		s.LayOnHandsRemaining() != 8 || s.ChannelDivinityRemaining() != 0 {
		t.Errorf("level 6 = %+v", s)
	}
	// Reaching the level of the feature makes its use available again (the "not yet gained" use is available by the invariant).
	atTwo := pldAdvanced(t, pldWith(t, pldPaladin(t, 2), pldState(t, 1, character.PaladinStateParts{LayOnHandsRemaining: 5, PaladinsSmite: character.LongRestUseSpent})), 1, "")
	if atTwo.State().FreeCast(character.FreeCastPaladinsSmite) != character.LongRestUseAvailable {
		t.Error("Divine Smite is not usable at once at level 2")
	}
	atFive := pldAdvanced(t, pldWith(t, pldPaladin(t, 5), pldState(t, 4, character.PaladinStateParts{LayOnHandsRemaining: 20, FaithfulSteed: character.LongRestUseSpent})), 4, "")
	if atFive.State().FreeCast(character.FreeCastFaithfulSteed) != character.LongRestUseAvailable {
		t.Error("Find Steed is not usable at once at level 5")
	}
	atTwenty := pldAdvanced(t, pldWith(t, pldPaladin(t, 20), pldState(t, 19, character.PaladinStateParts{LayOnHandsRemaining: 95, HolyNimbus: character.LongRestUseSpent})), 19, character.OathOfDevotionSubclassID)
	if atTwenty.State().HolyNimbusUse() != character.LongRestUseAvailable {
		t.Error("Holy Nimbus is not available at once at level 20")
	}
	bare := pldAdvanced(t, pldPaladin(t, 3), 2, "")
	if ev := bare.Events()[0].(character.PaladinAdvanced); ev.ChannelDivinityUsesGained() != 2 {
		t.Errorf("uses gained with no stored state = %d, want 2", ev.ChannelDivinityUsesGained())
	}
	if ev := pldAdvanced(t, pldPaladin(t, 11), 10, "").Events()[0].(character.PaladinAdvanced); ev.ChannelDivinityUsesGained() != 1 {
		t.Errorf("uses gained at level 11 with no stored state = %d, want 1", ev.ChannelDivinityUsesGained())
	}
	// A subclass that is not the Oath of Devotion, whichever way its id sorts, brings none of its spells.
	for _, other := range []content.SubclassID{"champion", "zealot"} {
		if got := pldAdvanced(t, pldPaladin(t, 3), 2, other).AlwaysPreparedSpells(); len(got) != 0 {
			t.Errorf("subclass %s grants %v at level 3", other, got)
		}
	}
	// A held state advanced from before the first level keeps its pool and gains the level 1 amount.
	if first := pldAdvanced(t, pldWith(t, pldPaladin(t, 1), pldFull(t, 1)), 0, ""); first.State().LayOnHandsRemaining() != 10 {
		t.Errorf("held state from level 0 = %+v", first.State())
	}
	if bare.State().LayOnHandsRemaining() != 15 || bare.State().ChannelDivinityRemaining() != 2 || !pldStateOf(t, bare.Character()).Equal(bare.State()) {
		t.Errorf("no stored state = %+v", bare.State())
	}
	first := pldAdvanced(t, pldPaladin(t, 1), 0, "")
	if first.State().LayOnHandsRemaining() != 5 {
		t.Errorf("level 1 = %+v", first.State())
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-105 - AdvancePaladin returns NotAPaladin for a character with no Paladin level, and an error matching srderr.ErrInvalidInput for a zero Character
// or progression, a previous level that is not one below the new level, or a foreign state under the Paladin key.
// Path: per case (unhappy: state error, invalid input).
func TestAdvancePaladinRejects(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	if res, err := character.AdvancePaladin(fgtFighter(t, 5), 4, p, ""); err != nil {
		t.Fatal(err)
	} else if _, ok := res.(character.AdvancePaladinNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
	foreign := must(character.WithFeatureState(pldPaladin(t, 3), character.MustFeatureStateKey[character.ChosenSubclass]("paladin_state"), must(character.NewChosenSubclass("oath_of_devotion"))))
	for name, args := range map[string]func() (character.AdvancePaladinResult, error){
		"zero character": func() (character.AdvancePaladinResult, error) {
			return character.AdvancePaladin(character.Character{}, 2, p, "")
		},
		"zero progression": func() (character.AdvancePaladinResult, error) {
			return character.AdvancePaladin(pldPaladin(t, 3), 2, character.PaladinProgression{}, "")
		},
		"previous equal": func() (character.AdvancePaladinResult, error) {
			return character.AdvancePaladin(pldPaladin(t, 3), 3, p, "")
		},
		"previous skipped": func() (character.AdvancePaladinResult, error) {
			return character.AdvancePaladin(pldPaladin(t, 3), 1, p, "")
		},
		"foreign state": func() (character.AdvancePaladinResult, error) { return character.AdvancePaladin(foreign, 2, p, "") },
	} {
		if res, err := args(); res != nil {
			t.Errorf("%s: result %#v", name, res)
		} else {
			wantInvalid(t, err, name)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-046, FR-064, FR-067, FR-096, Table R (SRD 5.2.1, "Level 1: Lay On Hands", p. 53; "Level 3: Channel Divinity", "Level 2: Paladin’s Smite", p. 54; "Level 5: Faithful Steed", p.
// 55; "Level 20: Holy Nimbus", p. 57) - ApplyPaladinRest follows each of the eight rows of Table R: a Long Rest refills the pool, the uses and the free casts, a Short Rest regains one Channel Divinity use
// (never above the maximum, nothing at the maximum) and restores no free cast, and active effects are left unchanged; an event is emitted only when something changed.
// Path: per case (Table R): happy / unhappy (boundary, empty and minimal, rule interaction).
func TestApplyPaladinRestTableR(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	id := character.MustIdentifier("pc-1")
	spentAll := character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 0, PaladinsSmite: character.LongRestUseSpent, FaithfulSteed: character.LongRestUseSpent, HolyNimbus: character.LongRestUseSpent}
	for _, c := range []struct {
		label string
		level int
		kind  character.RestKind
		in    character.PaladinStateParts
		out   character.PaladinStateParts
		event bool
	}{
		{"Long Rest, all spent", 20, character.LongRest, spentAll, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3}, true},
		{"Long Rest, nothing spent", 20, character.LongRest, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3}, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3}, false},
		{"Long Rest, effect active", 20, character.LongRest, func() character.PaladinStateParts {
			s := spentAll
			s.SacredWeaponActive, s.DivineSenseActive, s.HolyNimbusActive = "sword", true, true
			return s
		}(), character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3, SacredWeaponActive: "sword", DivineSenseActive: true, HolyNimbusActive: true}, true},
		{"Short Rest, none remaining", 5, character.ShortRest, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 0}, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 1}, true},
		{"Short Rest, 1 of 2 remaining", 5, character.ShortRest, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 1}, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 2}, true},
		{"Short Rest, at maximum", 5, character.ShortRest, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 2}, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 2}, false},
		{"Short Rest, level 11, 0 of 3", 11, character.ShortRest, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 0}, character.PaladinStateParts{LayOnHandsRemaining: 10, ChannelDivinityRemaining: 1}, true},
		{"Long Rest, only Lay On Hands used", 5, character.LongRest, character.PaladinStateParts{LayOnHandsRemaining: 20, ChannelDivinityRemaining: 2}, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2}, true},
		{"Long Rest, only Paladin’s Smite spent", 5, character.LongRest, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, PaladinsSmite: character.LongRestUseSpent}, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2}, true},
		{"Long Rest, only Faithful Steed spent", 5, character.LongRest, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, FaithfulSteed: character.LongRestUseSpent}, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2}, true},
		{"Long Rest, only Holy Nimbus spent", 20, character.LongRest, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3, HolyNimbus: character.LongRestUseSpent}, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 3}, true},
		{"Short Rest, spent free casts", 20, character.ShortRest, spentAll, func() character.PaladinStateParts {
			s := spentAll
			s.ChannelDivinityRemaining = 1
			return s
		}(), true},
	} {
		in := pldState(t, c.level, c.in)
		res := character.ApplyPaladinRest(id, p, character.MustPaladinLevel(c.level), in, c.kind)
		want := pldState(t, c.level, c.out)
		if !res.State().Equal(want) || (len(res.Events()) == 1) != c.event || len(res.Events()) > 1 {
			t.Errorf("%s: state = %+v (events %v), want %+v (event %v)", c.label, res.State(), res.Events(), want, c.event)
		}
	}
	long := character.ApplyPaladinRest(id, p, character.MustPaladinLevel(20), pldState(t, 20, spentAll), character.LongRest)
	ev := long.Events()[0].(character.PaladinResourcesRestored)
	if ev.LayOnHands() != 90 || ev.ChannelDivinityUses() != 3 || !ev.HolyNimbus() || !slices.Equal(ev.FreeCasts(), []character.FreeCastFeature{character.FreeCastPaladinsSmite, character.FreeCastFaithfulSteed}) || ev.CharacterID() != id {
		t.Errorf("event = %+v", ev)
	}
	if got := character.ApplyPaladinRest(id, p, character.MustPaladinLevel(5), pldFull(t, 5), 0); len(got.Events()) != 0 || got.Events() == nil {
		t.Errorf("an undefined rest kind = %v", got.Events())
	}
	ev.FreeCasts()[0] = character.FreeCastFaithfulSteed
	if ev.FreeCasts()[0] != character.FreeCastPaladinsSmite {
		t.Error("FreeCasts returned shared storage")
	}
}

// Requirement: doc/specs/class-paladin.md FR-111 - StartPaladin stores the level 1 state (pool 5, uses available, no choices) with PaladinLevelOneGained(multiclass) only on the multiclass path, rejects an
// existing state with AlreadyAPaladin, and returns an error matching srderr.ErrInvalidInput for a zero Character or progression, an undefined entry, a Character that is not exactly Paladin level 1 and a foreign state.
// Path: per case (happy / unhappy: state error, invalid input).
func TestStartPaladin(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	for _, c := range []struct {
		entry  character.ClassEntry
		events int
	}{{character.ClassEntryFirstLevel, 0}, {character.ClassEntryMulticlass, 1}} {
		res, err := character.StartPaladin(pldPaladin(t, 1), p, c.entry)
		if err != nil {
			t.Fatal(err)
		}
		st, ok := res.(character.StartPaladinStarted)
		if !ok || len(st.Events()) != c.events || st.Events() == nil || !pldStateOf(t, st.Character()).Equal(pldFull(t, 1)) || pldStateOf(t, st.Character()).LayOnHandsRemaining() != 5 {
			t.Fatalf("%v: result = %#v", c.entry, res)
		}
		if c.events == 1 {
			ev, ok := st.Events()[0].(character.PaladinLevelOneGained)
			if !ok || ev.Entry() != character.ClassEntryMulticlass || ev.CharacterID() != st.Character().ID() {
				t.Errorf("event = %#v", st.Events()[0])
			}
		}
		again, err := character.StartPaladin(st.Character(), p, c.entry)
		if _, ok := again.(character.StartPaladinAlreadyAPaladin); !ok || err != nil {
			t.Errorf("a second start = %#v, %v", again, err)
		}
	}
	foreign := must(character.WithFeatureState(pldPaladin(t, 1), character.MustFeatureStateKey[character.ChosenSubclass]("paladin_state"), must(character.NewChosenSubclass("oath_of_devotion"))))
	for name, call := range map[string]func() (character.StartPaladinResult, error){
		"zero character": func() (character.StartPaladinResult, error) {
			return character.StartPaladin(character.Character{}, p, 1)
		},
		"zero progression": func() (character.StartPaladinResult, error) {
			return character.StartPaladin(pldPaladin(t, 1), character.PaladinProgression{}, 1)
		},
		"undefined entry": func() (character.StartPaladinResult, error) { return character.StartPaladin(pldPaladin(t, 1), p, 0) },
		"not level 1":     func() (character.StartPaladinResult, error) { return character.StartPaladin(pldPaladin(t, 2), p, 1) },
		"not a Paladin":   func() (character.StartPaladinResult, error) { return character.StartPaladin(fgtFighter(t, 1), p, 1) },
		"foreign state":   func() (character.StartPaladinResult, error) { return character.StartPaladin(foreign, p, 1) },
	} {
		res, err := call()
		if res != nil {
			t.Errorf("%s: result %#v", name, res)
		}
		wantInvalid(t, err, name)
	}
}
