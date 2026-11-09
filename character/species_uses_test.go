package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func spExpend(t *testing.T, c character.Character, tr character.SpeciesTraits, req character.SpeciesUseRequest) character.SpeciesTraitsTransition {
	t.Helper()
	return must(character.SpeciesUses{}.Expend(c, tr, req))
}

func spRejection(t *testing.T, tr character.SpeciesTraitsTransition) character.SpeciesTraitRejection {
	t.Helper()
	r, ok := tr.(character.SpeciesTraitsRejected)
	if !ok {
		t.Fatalf("transition = %#v, want a rejection", tr)
	}
	return r.Rejection
}

// Requirement: doc/specs/species-trait-behaviour.md FR-030, FR-031, Table U "expend then exhausted" (SRD 5.2.1, "Breath Weapon", p. 84) - a use is available while the expended count is below the limit read at the level now: the third Breath Weapon at level 1 is exhausted, the same request at level 5 is accepted, and the event names the character and the use.
// Path: unhappy (boundary).
func TestSpeciesUsesExpendUntilExhausted(t *testing.T) {
	t.Parallel()
	req := character.BreathWeaponRequest{Shape: character.BreathCone15}
	tr := spDragonborn(t, character.DraconicRed)
	low := spChar(t, 1, 14)
	for i := 1; i <= 2; i++ {
		ch, ok := spExpend(t, low, tr, req).(character.SpeciesTraitsChanged)
		if !ok || ch.Traits().ExpendedOf(character.BreathWeaponUse) != i || len(ch.Events()) != 1 {
			t.Fatalf("expend %d = %#v", i, ch)
		}
		if ev := ch.Events()[0].(character.SpeciesUseExpended); ev.CharacterID() != low.ID() || ev.Use() != character.BreathWeaponUse {
			t.Errorf("event = %#v", ev)
		}
		tr = ch.Traits()
	}
	if rj := spRejection(t, spExpend(t, low, tr, req)); rj != (character.SpeciesUseExhausted{}) {
		t.Errorf("third use at level 1 = %v", rj)
	}
	if ch, ok := spExpend(t, spChar(t, 5, 14), tr, req).(character.SpeciesTraitsChanged); !ok || ch.Traits().ExpendedOf(character.BreathWeaponUse) != 3 {
		t.Errorf("third use at level 5 = %#v", ch)
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-031, FR-035, FR-039, Table U (SRD 5.2.1, pp. 84-86) - each use is accepted at its minimum level and refused one level below it with SpeciesUseNotYetAvailable carrying the minimum; a Forest Gnome, Elf, Tiefling, Orc, Goliath and Dwarf can spend their uses.
// Path: per case.
func TestSpeciesUsesExpendAccepted(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		traits character.SpeciesTraits
		req    character.SpeciesUseRequest
		use    character.SpeciesUse
		level  int
	}{
		{"Breath Weapon", spDragonborn(t, character.DraconicBlue), character.BreathWeaponRequest{Shape: character.BreathLine30}, character.BreathWeaponUse, 1},
		{"Draconic Flight", spDragonborn(t, character.DraconicBlue), character.DraconicFlightRequest{}, character.DraconicFlightUse, 5},
		{"Stonecunning", spDwarf(t), character.StonecunningRequest{}, character.StonecunningUse, 1},
		{"Giant Ancestry", spGoliath(t, character.AncestryFire), character.GiantAncestryRequest{Benefit: character.AncestryFire}, character.GiantAncestryUse, 1},
		{"Large Form", spGoliath(t, character.AncestryFire), character.LargeFormRequest{HasEnoughSpace: true}, character.LargeFormUse, 5},
		{"Adrenaline Rush", spOrc(t), character.AdrenalineRushRequest{}, character.AdrenalineRushUse, 1},
		{"Drow level 3 spell", spElf(t, character.ElvenDrow), character.LineageSpellRequest{Level: 3}, character.LineageSpellThreeUse, 3},
		{"Wood Elf level 5 spell", spElf(t, character.ElvenWoodElf), character.LineageSpellRequest{Level: 5}, character.LineageSpellFiveUse, 5},
		{"Tiefling level 3 spell", spTiefling(t, character.LegacyChthonic), character.LineageSpellRequest{Level: 3}, character.LineageSpellThreeUse, 3},
		{"Forest Gnome", spGnome(t, character.GnomishForest), character.GnomeSpellRequest{}, character.GnomeSpellUse, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch, ok := spExpend(t, spChar(t, c.level, 14), c.traits, c.req).(character.SpeciesTraitsChanged)
			if !ok || ch.Traits().ExpendedOf(c.use) != 1 || ch.Effect() == nil {
				t.Fatalf("at level %d = %#v", c.level, ch)
			}
			if c.level > 1 {
				rj := spRejection(t, spExpend(t, spChar(t, c.level-1, 14), c.traits, c.req))
				if rj != (character.SpeciesUseNotYetAvailable{MinimumLevel: c.level}) {
					t.Errorf("at level %d = %v", c.level-1, rj)
				}
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-031, FR-037, FR-038, FR-053, Table U "not offered", "own use case", Table O (SRD 5.2.1, pp. 84-86) - the first reason that holds is reported, in the order unknown species, not offered, own use case, not yet available, no space, exhausted; one case per adjacent pair with both conditions holding.
// Path: per case.
func TestSpeciesUsesExpendRejectionOrder(t *testing.T) {
	t.Parallel()
	large := character.LargeFormRequest{HasEnoughSpace: false}
	stoneSpent := spExpended(t, spGoliath(t, character.AncestryStone), character.GiantAncestryUse, 2)
	for _, c := range []struct {
		name   string
		level  int
		traits character.SpeciesTraits
		req    character.SpeciesUseRequest
		want   character.SpeciesTraitRejection
	}{
		{"unknown species and a use", 20, character.SpeciesTraits{}, character.StonecunningRequest{}, character.SpeciesBehaviourUnknown{}},
		{"Dwarf asks Breath Weapon", 5, spDwarf(t), character.BreathWeaponRequest{Shape: character.BreathCone15}, character.SpeciesUseNotOffered{}},
		{"Dwarf asks Relentless Endurance: not offered beats own use case", 5, spDwarf(t), character.RelentlessEnduranceRequest{}, character.SpeciesUseNotOffered{}},
		{"Rock Gnome asks Speak with Animals", 5, spGnome(t, character.GnomishRock), character.GnomeSpellRequest{}, character.SpeciesUseNotOffered{}},
		{"Hill Goliath asks a benefit not chosen", 5, spGoliath(t, character.AncestryHill), character.GiantAncestryRequest{Benefit: character.AncestryCloud}, character.SpeciesUseNotOffered{}},
		{"Hill Goliath asks Stone's Endurance", 5, spGoliath(t, character.AncestryHill), character.StonesEnduranceRequest{}, character.SpeciesUseNotOffered{}},
		{"Orc asks Relentless Endurance", 1, spOrc(t), character.RelentlessEnduranceRequest{}, character.SpeciesUseHasOwnUseCase{}},
		{"Stone Goliath asks Stone's Endurance", 1, spGoliath(t, character.AncestryStone), character.StonesEnduranceRequest{}, character.SpeciesUseHasOwnUseCase{}},
		{"Stone Goliath asks the Stone benefit", 1, spGoliath(t, character.AncestryStone), character.GiantAncestryRequest{Benefit: character.AncestryStone}, character.SpeciesUseHasOwnUseCase{}},
		{"own use case beats a spent pool", 1, stoneSpent, character.StonesEnduranceRequest{}, character.SpeciesUseHasOwnUseCase{}},
		{"Large Form at level 4 without space", 4, spGoliath(t, character.AncestryHill), large, character.SpeciesUseNotYetAvailable{MinimumLevel: 5}},
		{"Large Form without space", 5, spGoliath(t, character.AncestryHill), large, character.NotEnoughSpaceForLargeForm{}},
		{"Large Form without space and spent", 5, spExpended(t, spGoliath(t, character.AncestryHill), character.LargeFormUse, 1), large, character.NotEnoughSpaceForLargeForm{}},
		{"Large Form spent, with space", 5, spExpended(t, spGoliath(t, character.AncestryHill), character.LargeFormUse, 1), character.LargeFormRequest{HasEnoughSpace: true}, character.SpeciesUseExhausted{}},
		{"level 5 lineage spell already spent", 5, spExpended(t, spElf(t, character.ElvenDrow), character.LineageSpellFiveUse, 1), character.LineageSpellRequest{Level: 5}, character.SpeciesUseExhausted{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.traits
			if got := spRejection(t, spExpend(t, spChar(t, c.level, 14), c.traits, c.req)); got != c.want {
				t.Errorf("rejection = %#v, want %#v", got, c.want)
			}
			if c.traits != before {
				t.Error("the traits were altered")
			}
		})
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-031, FR-052 - a zero character, a nil request and a request with an invalid shape, benefit or lineage level are errors matching srderr.ErrInvalidInput, not rejections.
// Path: unhappy (invalid input).
func TestSpeciesUsesExpendInvalidInput(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		ch  character.Character
		req character.SpeciesUseRequest
	}{
		"zero character":        {character.Character{}, character.StonecunningRequest{}},
		"nil request":           {spChar(t, 5, 10), nil},
		"zero shape":            {spChar(t, 5, 10), character.BreathWeaponRequest{}},
		"zero benefit":          {spChar(t, 5, 10), character.GiantAncestryRequest{}},
		"lineage level 4":       {spChar(t, 5, 10), character.LineageSpellRequest{Level: 4}},
		"lineage level 0":       {spChar(t, 5, 10), character.LineageSpellRequest{}},
		"Breath Weapon level 9": {spChar(t, 5, 10), character.BreathWeaponRequest{Shape: 9}},
	} {
		if got, err := (character.SpeciesUses{}).Expend(c.ch, spDwarf(t), c.req); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, %v", name, got, err)
		}
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-033, Table U "renewal", "windows", "other character", "nothing spent" (SRD 5.2.1, "Long Rest" p. 185, "Short Rest" p. 187, "Resourceful" p. 86, "High Elf" p. 85) - a Long Rest clears every use, a Short Rest and the Short Rest benefits of an interrupted Long Rest clear only Adrenaline Rush; a Long Rest opens the Human and High Elf windows only; an event for another character is refused; nothing to clear and no window is a successful no-op.
// Path: per case.
func TestSpeciesUsesRenew(t *testing.T) {
	t.Parallel()
	orcBoth := spExpended(t, spExpended(t, spOrc(t), character.AdrenalineRushUse, 2), character.RelentlessEnduranceUse, 1)
	dragon := spExpended(t, spDragonborn(t, character.DraconicRed), character.BreathWeaponUse, 2)
	for _, c := range []struct {
		name     string
		traits   character.SpeciesTraits
		rest     character.RestApplied
		cleared  []character.SpeciesUse
		windows  []character.SpeciesWindow
		noEvents bool
		after    func(character.SpeciesTraits) bool
	}{
		{"Long Rest clears every use", orcBoth, spRest(t, character.LongRest, false), []character.SpeciesUse{character.AdrenalineRushUse, character.RelentlessEnduranceUse}, nil, false,
			func(tr character.SpeciesTraits) bool {
				return tr.ExpendedOf(character.AdrenalineRushUse) == 0 && tr.ExpendedOf(character.RelentlessEnduranceUse) == 0
			}},
		{"Short Rest clears Adrenaline Rush only", orcBoth, spRest(t, character.ShortRest, false), []character.SpeciesUse{character.AdrenalineRushUse}, nil, false,
			func(tr character.SpeciesTraits) bool {
				return tr.ExpendedOf(character.AdrenalineRushUse) == 0 && tr.ExpendedOf(character.RelentlessEnduranceUse) == 1
			}},
		{"interrupted Long Rest clears Adrenaline Rush only", orcBoth, spRest(t, character.LongRest, true), []character.SpeciesUse{character.AdrenalineRushUse}, nil, false,
			func(tr character.SpeciesTraits) bool {
				return tr.ExpendedOf(character.AdrenalineRushUse) == 0 && tr.ExpendedOf(character.RelentlessEnduranceUse) == 1
			}},
		{"Short Rest leaves Breath Weapon", dragon, spRest(t, character.ShortRest, false), nil, nil, true,
			func(tr character.SpeciesTraits) bool { return tr == dragon }},
		{"interrupted Long Rest leaves Breath Weapon", dragon, spRest(t, character.LongRest, true), nil, nil, true,
			func(tr character.SpeciesTraits) bool { return tr == dragon }},
		{"Long Rest opens Resourceful", spHuman(t), spRest(t, character.LongRest, false), nil, []character.SpeciesWindow{character.WindowResourceful}, false,
			func(tr character.SpeciesTraits) bool { return tr == spHuman(t) }},
		{"Short Rest opens nothing for a Human", spHuman(t), spRest(t, character.ShortRest, false), nil, nil, true,
			func(tr character.SpeciesTraits) bool { return tr == spHuman(t) }},
		{"Long Rest opens the High Elf swap", spElf(t, character.ElvenHighElf), spRest(t, character.LongRest, false), nil, []character.SpeciesWindow{character.WindowHighElfCantripReplacement}, false,
			func(tr character.SpeciesTraits) bool { return tr.CantripWindowOpen() }},
		{"interrupted Long Rest does not open the swap", spElf(t, character.ElvenHighElf), spRest(t, character.LongRest, true), nil, nil, true,
			func(tr character.SpeciesTraits) bool { return !tr.CantripWindowOpen() }},
		{"Long Rest for a Drow opens nothing", spElf(t, character.ElvenDrow), spRest(t, character.LongRest, false), nil, nil, true,
			func(tr character.SpeciesTraits) bool { return !tr.CantripWindowOpen() }},
		{"nothing spent, Dwarf", spDwarf(t), spRest(t, character.LongRest, false), nil, nil, true,
			func(tr character.SpeciesTraits) bool { return tr == spDwarf(t) }},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.traits
			ch, ok := must(character.SpeciesUses{}.Renew(spChar(t, 5, 10), c.traits, c.rest)).(character.SpeciesTraitsChanged)
			if !ok || !c.after(ch.Traits()) || c.traits != before {
				t.Fatalf("Renew = %#v", ch)
			}
			if c.noEvents {
				if len(ch.Events()) != 0 || len(ch.Windows()) != 0 {
					t.Errorf("events %v, windows %v; want none", ch.Events(), ch.Windows())
				}
				return
			}
			ev := ch.Events()
			if len(ev) != 1 {
				t.Fatalf("events = %v", ev)
			}
			got := ev[0].(character.SpeciesUsesRenewed)
			if !slices.Equal(got.Uses(), c.cleared) || !slices.Equal(got.Windows(), c.windows) || got.CharacterID() != character.MustIdentifier("pc-1") || len(ch.Windows()) != len(c.windows) {
				t.Errorf("event = %#v windows %v, want uses %v windows %v", got, ch.Windows(), c.cleared, c.windows)
			}
		})
	}
	if rj := spRejection(t, must(character.SpeciesUses{}.Renew(spChar(t, 5, 10), spDwarf(t), fgtRest(t, character.MustIdentifier("someone-else"), character.LongRest)))); rj != (character.RestNotForCharacter{}) {
		t.Errorf("another character = %v", rj)
	}
	if rj := spRejection(t, must(character.SpeciesUses{}.Renew(spChar(t, 5, 10), character.SpeciesTraits{}, spRest(t, character.LongRest, false)))); rj != (character.SpeciesBehaviourUnknown{}) {
		t.Errorf("zero traits = %v", rj)
	}
	if rj := spRejection(t, must(character.SpeciesUses{}.Renew(spChar(t, 5, 10), spDwarf(t), character.RestApplied{}))); rj != (character.RestNotForCharacter{}) {
		t.Errorf("zero event = %v", rj)
	}
	if got, err := (character.SpeciesUses{}).Renew(character.Character{}, spDwarf(t), spRest(t, character.LongRest, false)); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v, %v", got, err)
	}
}

// Requirement: doc/specs/species-trait-behaviour.md FR-033, Table U "renewal" (SRD 5.2.1, "Long Rest" p. 185, "Short Rest" p. 187) - every one of the ten uses is cleared by a Long Rest, and only Adrenaline Rush by a Short Rest.
// Path: per case.
func TestSpeciesUsesRenewEveryUse(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		use    character.SpeciesUse
		traits character.SpeciesTraits
	}{
		{character.BreathWeaponUse, spDragonborn(t, character.DraconicRed)}, {character.DraconicFlightUse, spDragonborn(t, character.DraconicRed)},
		{character.StonecunningUse, spDwarf(t)}, {character.GiantAncestryUse, spGoliath(t, character.AncestryStone)}, {character.LargeFormUse, spGoliath(t, character.AncestryStone)},
		{character.AdrenalineRushUse, spOrc(t)}, {character.RelentlessEnduranceUse, spOrc(t)},
		{character.LineageSpellThreeUse, spElf(t, character.ElvenDrow)}, {character.LineageSpellFiveUse, spTiefling(t, character.LegacyInfernal)},
		{character.GnomeSpellUse, spGnome(t, character.GnomishForest)},
	} {
		t.Run(c.use.String(), func(t *testing.T) {
			t.Parallel()
			spent := spExpended(t, c.traits, c.use, 1)
			for _, long := range []bool{true, false} {
				kind := character.ShortRest
				if long {
					kind = character.LongRest
				}
				ch := must(character.SpeciesUses{}.Renew(spChar(t, 5, 10), spent, spRest(t, kind, false))).(character.SpeciesTraitsChanged)
				cleared := ch.Traits().ExpendedOf(c.use) == 0
				if want := long || c.use == character.AdrenalineRushUse; cleared != want {
					t.Errorf("%v after a rest (long %v): cleared %v, want %v", c.use, long, cleared, want)
				}
			}
		})
	}
}
