package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// sorEntrant is a Fighter 3 who gains a first Sorcerer level (LevelUp through the aggregate), holding no SorcererState.
func sorEntrant(t *testing.T) (character.Character, character.LevelGained) {
	t.Helper()
	return sorLevelUp(t, restored(t, 355000, 200, flatScores(t, 16), cl("fighter", 3)))
}

// Requirement: doc/specs/class-sorcerer.md FR-011, FR-127 - a multiclass Sorcerer's first level seeds the level 1 SorcererState (0 Sorcery Points, 2 Innate Sorcery uses); at any other class level with no state it is NotASorcerer, and so is a
// LevelGained event for another class.
// Path: per case (happy / unhappy: state).
func TestApplySorcererLevelUpMulticlassEntry(t *testing.T) {
	t.Parallel()
	uc := must(character.NewApplySorcererLevelUp(sorSource(t)))
	c, gained := sorEntrant(t)
	res := must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained})).(character.ApplySorcererLevelUpApplied)
	if got := sorStateOf(t, res.Character()); got != sorFresh(sorProfile(1, "")) || res.OwedMetamagicOptions() != 0 || len(res.Events()) != 0 {
		t.Errorf("entrant = %+v owing %d with %d events, want the level 1 state", got.Parts(), res.OwedMetamagicOptions(), len(res.Events()))
	}
	if _, held, _ := character.ReadFeatureState(c.FeatureStates(), character.SorcererStateKey()); held {
		t.Error("the command's Character was altered")
	}
	// A Sorcerer level 2 with no state, as if the entrant skipped its first level: NotASorcerer.
	twoNoState := restored(t, 355000, 200, flatScores(t, 16), cl("fighter", 3), cl("sorcerer", 1))
	two, gained2 := sorLevelUp(t, twoNoState)
	if r, ok := must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: two, Gained: gained2})).(character.ApplySorcererLevelUpRejected); !ok || r.Rejection.Kind != character.SorcererNotASorcerer {
		t.Errorf("level 2 with no state = %#v, want NotASorcerer", r)
	}
	fighter, fighterGain := fgtLevelUp(t, restored(t, 355000, 200, flatScores(t, 16), cl("fighter", 3), cl("sorcerer", 1)), "fighter")
	withState := must(character.WithFeatureState(fighter, character.SorcererStateKey(), sorFresh(sorProfile(1, ""))))
	if r, ok := must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: withState, Gained: fighterGain})).(character.ApplySorcererLevelUpRejected); !ok || r.Rejection.Kind != character.SorcererNotASorcerer {
		t.Errorf("a Fighter level = %#v, want NotASorcerer", r)
	}
	other := sorRenamed(t, c, "pc-2")
	if _, err := uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: other, Gained: gained}); !srderrIsInvalid(err) {
		t.Errorf("another character's event: %v", err)
	}
	if _, err := uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: other, Gained: character.LevelGained{}}); !srderrIsInvalid(err) {
		t.Errorf("the zero event: %v", err)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-035, FR-047, FR-048, FR-127 (SRD 5.2.1, "Level 2: Font of Magic", "Level 2: Metamagic", p. 66) - each Sorcerer level raises the Sorcery Points by the rise of the maximum and never above it
// (2 on reaching level 2, then 1 a level), and reports the Metamagic options still owed (2 at level 2, 2 more at 10 and at 17).
// Path: per level (1 to 20).
func TestApplySorcererLevelUpPointsAndOwedOptions(t *testing.T) {
	t.Parallel()
	owed := []int{0, 2, 2, 2, 2, 2, 2, 2, 2, 4, 4, 4, 4, 4, 4, 4, 6, 6, 6, 6}
	uc := must(character.NewApplySorcererLevelUp(sorSource(t)))
	for n := 2; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			prev := sorProfile(n-1, "")
			before := sorChar(t, n-1, 16, "", sorWith(prev, prev.Level().SorceryPointsMaximum(), nil, nil))
			c, gained := sorLevelUp(t, before)
			res := must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained})).(character.ApplySorcererLevelUpApplied)
			got := sorStateOf(t, res.Character())
			if got.SorceryPoints() != n || res.OwedMetamagicOptions() != owed[n-1] {
				t.Errorf("level %d = %d points owing %d, want %d and %d", n, got.SorceryPoints(), res.OwedMetamagicOptions(), n, owed[n-1])
			}
		})
	}
	// Points spent are not refilled by a level: only the rise of the maximum is added.
	p := sorProfile(4, "")
	spent := sorChar(t, 4, 16, "", sorWith(p, 1, nil, nil))
	c, gained := sorLevelUp(t, spent)
	if got := sorStateOf(t, must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained})).(character.ApplySorcererLevelUpApplied).Character()).SorceryPoints(); got != 2 {
		t.Errorf("1 point at level 4 → %d at level 5, want 2", got)
	}
	// A state above the new maximum (restored by a host) is clamped.
	full := sorChar(t, 4, 16, "", sorWith(p, 4, nil, nil))
	full = must(character.WithFeatureState(full, character.SorcererStateKey(), sorWith(p, 4, nil, nil)))
	c, gained = sorLevelUp(t, full)
	if got := sorStateOf(t, must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained})).(character.ApplySorcererLevelUpApplied).Character()).SorceryPoints(); got != 5 {
		t.Errorf("4 of 4 points → %d at level 5, want 5", got)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-035, FR-056 to FR-058 - the Metamagic choice of a level is the full set the Sorcerer holds, replacing at most one option it knew, in which case none is owed; a wrong size, a repeat, an unlisted
// option and a second replacement are refused.
// Path: per case (happy / unhappy: invalid input, state).
func TestApplySorcererLevelUpMetamagicChoice(t *testing.T) {
	t.Parallel()
	uc := must(character.NewApplySorcererLevelUp(sorSource(t)))
	knows := []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicTwinned}
	level3 := func(metamagic []content.FeatureOptionID) character.ApplySorcererLevelUpResult {
		c, gained := sorLevelUp(t, sorChar(t, 2, 16, "", sorWith(sorProfile(2, ""), 2, knows, nil)))
		return must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained, Metamagic: metamagic}))
	}
	kept := level3(nil).(character.ApplySorcererLevelUpApplied)
	if !slices.Equal(sorStateOf(t, kept.Character()).Metamagic(), knows) || len(kept.Events()) != 0 || kept.OwedMetamagicOptions() != 0 {
		t.Errorf("nil choice = %+v owing %d with %d events, want the options kept", sorStateOf(t, kept.Character()).Parts(), kept.OwedMetamagicOptions(), len(kept.Events()))
	}
	one := level3(sorOptionIDs(character.MetamagicCareful, character.MetamagicDistant)).(character.ApplySorcererLevelUpApplied)
	ev, ok := one.Events()[0].(character.MetamagicOptionsChosen)
	if !slices.Equal(sorStateOf(t, one.Character()).Metamagic(), []character.MetamagicEffect{character.MetamagicCareful, character.MetamagicDistant}) || !ok || len(one.Events()) != 1 || ev.CharacterID() != one.Character().ID() {
		t.Errorf("one replacement = %+v events %#v", sorStateOf(t, one.Character()).Parts(), one.Events())
	}
	if r, ok := level3(sorOptionIDs(character.MetamagicDistant, character.MetamagicEmpowered)).(character.ApplySorcererLevelUpRejected); !ok || r.Rejection.Kind != character.SorcererMetamagicReplacementInvalid {
		t.Errorf("two replacements = %#v", r)
	}
	for name, chosen := range map[string][]content.FeatureOptionID{"wrong size": sorOptionIDs(1), "a repeat": sorOptionIDs(1, 1), "unlisted": {"careful_spell", "lucky_spell"}, "empty": {}} {
		if r, ok := level3(chosen).(character.ApplySorcererLevelUpRejected); !ok || r.Rejection.Kind != character.SorcererMetamagicSelectionInvalid || len(r.Rejection.Problems) == 0 {
			t.Errorf("%s = %#v", name, r)
		}
	}
	// A choice at level 1 (no Metamagic yet) is refused by the rule, not by the selection.
	c, gained := sorEntrant(t)
	if r, ok := must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: c, Gained: gained, Metamagic: []content.FeatureOptionID{}})).(character.ApplySorcererLevelUpRejected); !ok || r.Rejection.Kind != character.SorcererFeatureNotGained {
		t.Errorf("a choice at level 1 = %#v, want SorcererFeatureNotGained", r)
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-085 to FR-087, FR-089, FR-127, FR-134 - the result reports the feat grants, the Draconic Resilience bonus and the Draconic Spells newly granted at the level; no second level event is emitted.
// Path: per case (happy).
func TestApplySorcererLevelUpGrants(t *testing.T) {
	t.Parallel()
	uc := must(character.NewApplySorcererLevelUp(sorSource(t)))
	for _, c := range []struct {
		level  int
		feats  []content.FeatCategory
		spells []content.SpellID
		bonus  bool
	}{
		{3, []content.FeatCategory{}, []content.SpellID{"alter_self", "chromatic_orb", "command", "dragons_breath"}, true},
		{4, []content.FeatCategory{content.FeatGeneral}, []content.SpellID{}, true},
		{5, []content.FeatCategory{}, []content.SpellID{"fear", "fly"}, true},
		{19, []content.FeatCategory{content.FeatEpicBoon}, []content.SpellID{}, true},
	} {
		prev := sorProfile(c.level-1, "draconic_sorcery")
		before := sorChar(t, c.level-1, 16, "draconic_sorcery", sorWith(prev, prev.Level().SorceryPointsMaximum(), nil, nil))
		ch, gained := sorLevelUp(t, before)
		res := must(uc.Run(t.Context(), character.ApplySorcererLevelUpCommand{Character: ch, Gained: gained})).(character.ApplySorcererLevelUpApplied)
		_, has := res.Grants().HitPointBonus()
		if !slices.Equal(res.Grants().FeatGrants(), c.feats) || !slices.Equal(res.Grants().AlwaysPreparedSpells(), c.spells) || has != c.bonus || len(res.Events()) != 0 {
			t.Errorf("level %d grants = feats %v spells %v bonus %v events %d", c.level, res.Grants().FeatGrants(), res.Grants().AlwaysPreparedSpells(), has, len(res.Events()))
		}
	}
}
