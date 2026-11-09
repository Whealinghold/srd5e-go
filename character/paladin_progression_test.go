package character_test

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// Requirement: doc/specs/class-paladin.md FR-057, FR-058 (SRD 5.2.1, "Paladin Features", p. 53; "Proficiency", p. 8) - the Proficiency Bonus column of the bundled Paladin Features table
// equals the general Proficiency Bonus for the same level, it has no Cantrips column, and its Prepared Spells and highest Spell Slot level agree with what spellcasting derives for the Paladin.
// A disagreement fails this consistency test, not a decode.
// Path: per case (the 20 levels): unhappy (rule interaction).
func TestPaladinTableAgreesWithTheRules(t *testing.T) {
	t.Parallel()
	def := pldDef(t)
	for n := 1; n <= 20; n++ {
		t.Run("level "+strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			row := must(def.RowAt(n))
			sc := row.Spellcasting().(content.SlotSpellcastingRow)
			_, hasCantrips := sc.Cantrips()
			highest := 0
			for i, s := range sc.SpellSlots() {
				if s > 0 {
					highest = i + 1
				}
			}
			if row.ProficiencyBonus() != character.ProficiencyBonusFor(character.MustLevel(n)).Value() || hasCantrips ||
				sc.PreparedSpells() != character.PreparedLimit(character.CasterPaladin, n) || highest != character.MaxPreparableLevel(character.CasterPaladin, n) {
				t.Errorf("level %d row = bonus %d cantrips %v prepared %d highest slot %d; the rules say %d no cantrips %d %d", n, row.ProficiencyBonus(), hasCantrips, sc.PreparedSpells(), highest,
					character.ProficiencyBonusFor(character.MustLevel(n)).Value(), character.PreparedLimit(character.CasterPaladin, n), character.MaxPreparableLevel(character.CasterPaladin, n))
			}
		})
	}
}

// Requirement: doc/specs/class-paladin.md FR-065, FR-125 (SRD 5.2.1, "Paladin Features", p. 53) - the Channel Divinity maximum is read from the bundled column: 0 below level 3, 2 at
// levels 3 to 10 and 3 at 11 to 20; the zero progression and level give 0.
// Path: per case (the boundary levels 1, 2, 3, 10, 11, 20): unhappy (boundary).
func TestPaladinChannelDivinityMaximum(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	for _, c := range []struct{ level, want int }{{1, 0}, {2, 0}, {3, 2}, {10, 2}, {11, 3}, {20, 3}} {
		if got := p.ChannelDivinityMaximum(character.MustPaladinLevel(c.level)); got != c.want {
			t.Errorf("ChannelDivinityMaximum(%d) = %d, want %d", c.level, got, c.want)
		}
	}
	if p.IsZero() || !(character.PaladinProgression{}).IsZero() || p.ChannelDivinityMaximum(character.PaladinLevel{}) != 0 || (character.PaladinProgression{}).ChannelDivinityMaximum(character.MustPaladinLevel(5)) != 0 {
		t.Error("zero progression or level gives a non-zero maximum")
	}
}

// Requirement: doc/specs/class-paladin.md FR-125 (SRD 5.2.1, "Paladin Class Features", pp. 53-55; "Oath of Devotion", pp. 56-57) - LevelOf gives the level of every gating feature entry of the
// bundled data, and 0 for an invalid feature, the zero progression, or an Oath feature when the progression was built without the Oath.
// Path: per case (the 15 features): happy / unhappy (invalid input, missing subclass).
func TestPaladinLevelOf(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	noOath := must(character.NewPaladinProgression(pldDef(t), nil))
	for _, c := range []struct {
		f     character.PaladinFeature
		level int
		oath  bool
	}{
		{character.PaladinFeatureFightingStyle, 2, false}, {character.PaladinFeatureChannelDivinity, 3, false}, {character.PaladinFeaturePaladinsSmite, 2, false},
		{character.PaladinFeatureFaithfulSteed, 5, false}, {character.PaladinFeatureExtraAttack, 5, false}, {character.PaladinFeatureAuraOfProtection, 6, false},
		{character.PaladinFeatureAbjureFoes, 9, false}, {character.PaladinFeatureAuraOfCourage, 10, false}, {character.PaladinFeatureRadiantStrikes, 11, false},
		{character.PaladinFeatureRestoringTouch, 14, false}, {character.PaladinFeatureAuraExpansion, 18, false}, {character.PaladinFeatureSacredWeapon, 3, true},
		{character.PaladinFeatureAuraOfDevotion, 7, true}, {character.PaladinFeatureSmiteOfProtection, 15, true}, {character.PaladinFeatureHolyNimbus, 20, true},
	} {
		if p.LevelOf(c.f) != c.level || c.f.String() == "unknown" || !c.f.Valid() {
			t.Errorf("LevelOf(%v) = %d, want %d", c.f, p.LevelOf(c.f), c.level)
		}
		want := c.level
		if c.oath {
			want = 0
		}
		if noOath.LevelOf(c.f) != want {
			t.Errorf("without the Oath LevelOf(%v) = %d, want %d", c.f, noOath.LevelOf(c.f), want)
		}
	}
	if p.LevelOf(0) != 0 || p.LevelOf(99) != 0 || (character.PaladinProgression{}).LevelOf(character.PaladinFeatureHolyNimbus) != 0 || character.PaladinFeature(0).String() != "unknown" || character.PaladinFeature(99).Valid() {
		t.Error("invalid feature or zero progression has a level")
	}
}

// Requirement: doc/specs/class-paladin.md FR-014 (SRD 5.2.1, "Paladin Features", p. 53) - NewPaladinProgression rejects a definition that is another class, the zero definition, an Oath of
// Devotion given as another subclass, a missing Channel Divinity column, a dash from level 3, or a missing gating entry or always-prepared fact, and lists every problem found at once, each
// matching srderr.ErrInvalidInput; MustPaladinProgression panics with that error.
// Path: per case (unhappy: invalid input).
func TestNewPaladinProgressionRejects(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		def  content.ClassDefinition
		want []character.PaladinProgressionError
	}{
		{"another class", fgtDef(t), []character.PaladinProgressionError{{Kind: character.PaladinWrongClass}}},
		{"zero definition", content.ClassDefinition{}, []character.PaladinProgressionError{{Kind: character.PaladinWrongClass}}},
		{"no channel_divinity column", pldDefFrom(t, func(s string) string { return strings.ReplaceAll(s, "channel_divinity", "channel_x") }),
			[]character.PaladinProgressionError{{Kind: character.PaladinMissingChannelDivinityColumn}}},
		{"a dash at level 3", pldDefFrom(t, func(s string) string {
			return strings.Replace(s, `"channel_divinity": 2`, `"channel_divinity": "—"`, 1)
		}),
			[]character.PaladinProgressionError{{Kind: character.PaladinMissingChannelDivinityColumn}}},
	} {
		_, err := character.NewPaladinProgression(c.def, pldOath(t))
		wantInvalid(t, err, c.name)
		for _, w := range c.want {
			if !errors.Is(err, &w) {
				t.Errorf("%s: error = %v, want problem %v", c.name, err, w)
			}
		}
	}
	// Several problems at once: two gating entries renamed away.
	multi := pldDefFrom(t, func(s string) string {
		s = strings.ReplaceAll(s, "paladin_abjure_foes", "paladin_x_foes")
		return strings.ReplaceAll(s, "paladin_restoring_touch", "paladin_x_touch")
	})
	_, err := character.NewPaladinProgression(multi, pldOath(t))
	for _, name := range []string{"Abjure Foes", "Restoring Touch"} {
		if !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinMissingFeatureEntry, Feature: name}) {
			t.Errorf("error = %v, want a missing %s", err, name)
		}
	}
	var pe *character.PaladinProgressionError
	if !errors.As(err, &pe) || pe.Error() == "" || pe.Kind.String() != "missing_feature_entry" {
		t.Errorf("errors.As = %v", pe)
	}
	for k, want := range map[character.PaladinProblemKind]string{character.PaladinWrongClass: "wrong_class", character.PaladinMissingChannelDivinityColumn: "missing_channel_divinity_column", 0: "unknown"} {
		if k.String() != want {
			t.Errorf("kind %d = %q, want %q", k, k.String(), want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("MustPaladinProgression did not panic")
		}
	}()
	character.MustPaladinProgression(fgtDef(t), nil)
}

// Requirement: doc/specs/class-paladin.md FR-125 (SRD 5.2.1, "Level 5: Extra Attack", p. 55) - the level gates follow the loaded definition: a definition whose Extra Attack entry is at level 6 gives
// 2 attacks from level 6, not 5.
// Path: unhappy (rule interaction: a gate moved by the definition).
func TestPaladinGatesFollowTheDefinition(t *testing.T) {
	t.Parallel()
	moved := pldDefFrom(t, func(s string) string {
		ref := "{\n              \"featureId\": \"paladin_extra_attack\"\n            }"
		s = strings.Replace(s, ref+",\n            ", "", 1)
		aura := "\"featureId\": \"paladin_aura_of_protection\"\n            }"
		s = strings.Replace(s, aura, aura+",\n            "+ref, 1)
		return strings.Replace(s, "\"id\": \"paladin_extra_attack\",\n          \"name\": \"Extra Attack\",\n          \"level\": 5,", "\"id\": \"paladin_extra_attack\",\n          \"name\": \"Extra Attack\",\n          \"level\": 6,", 1)
	})
	p := must(character.NewPaladinProgression(moved, nil))
	if p.LevelOf(character.PaladinFeatureExtraAttack) != 6 {
		t.Fatalf("Extra Attack level = %d, want 6", p.LevelOf(character.PaladinFeatureExtraAttack))
	}
	for _, c := range []struct{ level, want int }{{5, 1}, {6, 2}} {
		if got := character.AttacksPerAttackAction(p, character.MustPaladinLevel(c.level)); got != c.want {
			t.Errorf("level %d = %d attacks, want %d", c.level, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-023, FR-058, FR-059, Table A (SRD 5.2.1, "Level 2: Paladin’s Smite", "Level 5: Faithful Steed", pp. 54-55; "Level 3: Oath of Devotion Spells", p. 56) -
// AlwaysPrepared returns the grants of the Divine Smite and Find Steed facts and, with the Oath chosen, the Oath of Devotion Spells grants at or below the level, unfiltered by the Paladin
// list (Beacon of Hope at level 9).
// Path: per case (Table A): happy / unhappy (boundary, empty case, state error, rule interaction).
func TestPaladinAlwaysPrepared(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	l5 := []string{"divine_smite", "find_steed", "protection_from_evil_and_good", "shield_of_faith", "aid", "zone_of_truth"}
	l13 := append(slices.Clone(l5), "beacon_of_hope", "dispel_magic", "freedom_of_movement", "guardian_of_faith")
	for _, c := range []struct {
		label string
		level int
		oath  bool
		want  []string
	}{
		{"level 1", 1, false, nil},
		{"Paladin’s Smite", 2, false, []string{"divine_smite"}},
		{"level 3, subclass not yet chosen", 3, false, []string{"divine_smite"}},
		{"level 3, Oath", 3, true, []string{"divine_smite", "protection_from_evil_and_good", "shield_of_faith"}},
		{"level 4, Oath", 4, true, []string{"divine_smite", "protection_from_evil_and_good", "shield_of_faith"}},
		{"level 5, no subclass", 5, false, []string{"divine_smite", "find_steed"}},
		{"level 5, Oath", 5, true, l5},
		{"level 9, Oath", 9, true, append(slices.Clone(l5), "beacon_of_hope", "dispel_magic")},
		{"level 13, Oath", 13, true, l13},
		{"level 16, Oath", 16, true, l13},
		{"level 17, Oath", 17, true, append(slices.Clone(l13), "commune", "flame_strike")},
		{"level 20, Oath", 20, true, append(slices.Clone(l13), "commune", "flame_strike")},
	} {
		got := p.AlwaysPrepared(character.MustPaladinLevel(c.level), c.oath)
		ids := make([]string, len(got))
		for i, id := range got {
			ids[i] = string(id)
		}
		if len(ids) != len(c.want) || (len(ids) > 0 && !slices.Equal(ids, c.want)) {
			t.Errorf("%s: AlwaysPrepared = %v, want %v", c.label, ids, c.want)
		}
	}
	if got := p.AlwaysPrepared(character.MustPaladinLevel(20), true); len(got) != 12 {
		t.Errorf("level 20 = %d spells, want 12", len(got))
	}
	// A modified definition (Divine Smite granted at level 3) proves the grants are read from the definition.
	moved := must(character.NewPaladinProgression(pldDefFrom(t, func(s string) string {
		return strings.Replace(s, "\"level\": 2,\n                  \"spellIds\": [\n                    \"divine_smite\"", "\"level\": 3,\n                  \"spellIds\": [\n                    \"divine_smite\"", 1)
	}), nil))
	if got := moved.AlwaysPrepared(character.MustPaladinLevel(2), false); len(got) != 0 {
		t.Errorf("modified definition at level 2 = %v, want none", got)
	}
	if got := moved.AlwaysPrepared(character.MustPaladinLevel(3), false); len(got) != 1 || got[0] != "divine_smite" {
		t.Errorf("modified definition at level 3 = %v, want Divine Smite", got)
	}
	if got := (character.PaladinProgression{}).AlwaysPrepared(character.MustPaladinLevel(9), true); got == nil || len(got) != 0 {
		t.Errorf("zero progression = %v, want an empty slice", got)
	}
	if p.FreeCastSpell(character.FreeCastPaladinsSmite) != "divine_smite" || p.FreeCastSpell(character.FreeCastFaithfulSteed) != "find_steed" || p.FreeCastSpell(0) != "" ||
		(character.PaladinProgression{}).FreeCastSpell(character.FreeCastPaladinsSmite) != "" {
		t.Error("FreeCastSpell wrong")
	}
	again := p.AlwaysPrepared(character.MustPaladinLevel(5), true)
	again[0] = "x"
	if p.AlwaysPrepared(character.MustPaladinLevel(5), true)[0] != "divine_smite" {
		t.Error("AlwaysPrepared returned shared storage")
	}
}

// Requirement: doc/specs/class-paladin.md FR-042, FR-088, Table E (SRD 5.2.1, "Level 4: Ability Score Improvement", "Level 19: Epic Boon", p. 55) - FeatEntitlements gives the feat grants of the
// definition minus the Fighting Style: one General at 4, 8, 12 and 16, one Epic Boon at 19, none otherwise, five by level 20; FeaturesGainedAt lists the entries of the level and
// ChoicesOnGaining the Fighting Style at 2 only.
// Path: per case (Table E): happy / unhappy (boundary, rule interaction).
func TestPaladinFeatEntitlementsAndGains(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	total := 0
	for _, c := range []struct {
		label string
		level int
		want  []content.FeatCategory
	}{
		{"Fighting Style", 2, nil}, {"none yet", 3, nil}, {"first", 4, []content.FeatCategory{content.FeatGeneral}}, {"gap", 7, nil},
		{"second", 8, []content.FeatCategory{content.FeatGeneral}}, {"third", 12, []content.FeatCategory{content.FeatGeneral}}, {"fourth", 16, []content.FeatCategory{content.FeatGeneral}},
		{"Epic Boon", 19, []content.FeatCategory{content.FeatEpicBoon}}, {"last level", 20, nil},
	} {
		got := p.FeatEntitlements(character.MustPaladinLevel(c.level))
		if !slices.Equal(got, c.want) && (len(got) != 0 || len(c.want) != 0) {
			t.Errorf("%s: FeatEntitlements(%d) = %v, want %v", c.label, c.level, got, c.want)
		}
	}
	for n := 1; n <= 20; n++ {
		total += len(p.FeatEntitlements(character.MustPaladinLevel(n)))
	}
	if total != 5 {
		t.Errorf("entitlements by level 20 = %d, want 5", total)
	}
	if got := p.FeaturesGainedAt(character.MustPaladinLevel(1)); len(got) != 3 || got[0] != "paladin_lay_on_hands" {
		t.Errorf("level 1 features = %v", got)
	}
	if got := p.FeaturesGainedAt(character.MustPaladinLevel(13)); got == nil || len(got) != 0 {
		t.Errorf("level 13 features = %v, want an empty slice", got)
	}
	if len(p.ChoicesOnGaining(character.MustPaladinLevel(2))) != 1 || len(p.ChoicesOnGaining(character.MustPaladinLevel(3))) != 0 || len(p.ChoicesOnGaining(character.MustPaladinLevel(1))) != 0 ||
		p.ChoicesOnGaining(character.MustPaladinLevel(2))[0].String() != "fighting_style" || character.PaladinPendingChoice(0).String() != "unknown" {
		t.Error("ChoicesOnGaining wrong")
	}
	z := character.PaladinProgression{}
	if len(z.FeatEntitlements(character.MustPaladinLevel(4))) != 0 || len(z.FeaturesGainedAt(character.MustPaladinLevel(1))) != 0 || len(z.ChoicesOnGaining(character.MustPaladinLevel(2))) != 0 ||
		len(p.FeatEntitlements(character.PaladinLevel{})) != 0 || len(p.FeaturesGainedAt(character.PaladinLevel{})) != 0 {
		t.Error("zero progression or level gains something")
	}
}

// pldDropFact removes the always-prepared fact that names spell from the bundled Paladin file text.
func pldDropFact(spell string) func(string) string {
	re := regexp.MustCompile(`(?s),\n\s*"facts": \[\s*\{\s*"kind": "alwaysPreparedSpells",\s*"grants": \[\s*\{\s*"level": \d+,\s*"spellIds": \[\s*"` + spell + `"\s*\]\s*\}\s*\]\s*\}\s*\]`)
	return func(s string) string { return re.ReplaceAllString(s, "") }
}

// pldDropOathFact removes the always-prepared fact of the Oath of Devotion Spells entry.
func pldDropOathFact(s string) string {
	re := regexp.MustCompile(`(?s)("id": "oath_of_devotion_oath_of_devotion_spells",.*?"text": "[^"]*")(,\n\s*"facts": \[.*?"flame_strike"\s*\]\s*\}\s*\]\s*\}\s*\])`)
	return re.ReplaceAllString(s, "$1")
}

// Requirement: doc/specs/class-paladin.md FR-014 (SRD 5.2.1, "Paladin Features", p. 53; "Level 3: Oath of Devotion Spells", p. 56) - NewPaladinProgression reports a definition of a class that sorts after "paladin", a
// subclass that is not the Oath of Devotion (whichever way its id sorts), a Channel Divinity column that is not a count column, a dash at a level above 3, a gating entry missing without the Oath, an entry without its
// always-prepared fact (Divine Smite, Find Steed, the Oath of Devotion Spells), and MustPaladinProgression returns a valid progression without panicking.
// Path: per case (unhappy: invalid input).
func TestNewPaladinProgressionRejectsMore(t *testing.T) {
	t.Parallel()
	bundle := must(data.NewBundledSource())
	classOf := func(id string) content.ClassDefinition {
		return must(bundle.Lookup(context.Background(), content.NewKey(content.KindCharacterClass, content.MustID(id)))).(content.LookupFound).Entry.(content.ClassDefinition)
	}
	if _, err := character.NewPaladinProgression(classOf("wizard"), nil); !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinWrongClass}) {
		t.Errorf("a Wizard = %v, want wrong_class", err)
	}
	champion, thief := classOf("fighter").Subclasses()[0], classOf("rogue").Subclasses()[0]
	for _, sub := range []content.SubclassDefinition{champion, thief} {
		if _, err := character.NewPaladinProgression(pldDef(t), &sub); !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinWrongClass}) {
			t.Errorf("subclass %s = %v, want wrong_class", sub.ID(), err)
		}
	}
	bonusColumn := pldDefFrom(t, func(s string) string {
		s = strings.Replace(s, "\"header\": \"Channel Divinity\",\n          \"kind\": \"count\"", "\"header\": \"Channel Divinity\",\n          \"kind\": \"bonus\"", 1)
		return s
	})
	lateDash := pldDefFrom(t, func(s string) string {
		return strings.Replace(s, `"channel_divinity": 3`, `"channel_divinity": "—"`, 1)
	})
	for name, def := range map[string]content.ClassDefinition{"a bonus column": bonusColumn, "a dash at level 11": lateDash} {
		if _, err := character.NewPaladinProgression(def, nil); !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinMissingChannelDivinityColumn}) {
			t.Errorf("%s = %v, want missing_channel_divinity_column", name, err)
		}
	}
	noOathAbjure := pldDefFrom(t, func(s string) string { return strings.ReplaceAll(s, "paladin_abjure_foes", "paladin_x_foes") })
	if _, err := character.NewPaladinProgression(noOathAbjure, nil); !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinMissingFeatureEntry, Feature: "Abjure Foes"}) {
		t.Errorf("a missing class entry without the Oath = %v", err)
	}
	for _, c := range []struct{ name, feature, spell string }{{"Divine Smite", "Paladin’s Smite", "divine_smite"}, {"Find Steed", "Faithful Steed", "find_steed"}} {
		def := pldDefFrom(t, pldDropFact(c.spell))
		if _, err := character.NewPaladinProgression(def, nil); !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinMissingFeatureEntry, Feature: c.feature}) {
			t.Errorf("no %s fact = %v, want a missing %s", c.name, err, c.feature)
		}
	}
	oathNoFact := pldDefFrom(t, pldDropOathFact)
	sub := oathNoFact.Subclasses()[0]
	if _, err := character.NewPaladinProgression(oathNoFact, &sub); !errors.Is(err, &character.PaladinProgressionError{Kind: character.PaladinMissingFeatureEntry, Feature: "Oath of Devotion Spells"}) {
		t.Errorf("no Oath spells fact = %v", err)
	}
	if p := character.MustPaladinProgression(pldDef(t), pldOath(t)); p.IsZero() {
		t.Error("MustPaladinProgression returned the zero progression")
	}
	if character.FreeCastFeature(3) != 3 || pldProgression(t).FreeCastSpell(3) != "" || pldProgression(t).FreeCastSpell(-1) != "" {
		t.Error("an undefined free cast has a spell")
	}
	if character.PaladinPendingChoice(2).String() != "unknown" || character.PaladinPendingChoice(-1).String() != "unknown" {
		t.Error("an undefined pending choice is named")
	}
}

// Requirement: doc/specs/class-paladin.md FR-088, FR-125 - the entries and feat grants of level 20 are read too: a definition whose Epic Boon entry is at level 20 gains it, and its feat grant, at 20 and not at 19.
// Path: unhappy (boundary: the last level, rule interaction: a gate moved by the definition).
func TestPaladinProgressionReadsLevel20(t *testing.T) {
	t.Parallel()
	moved := pldDefFrom(t, func(s string) string {
		s = regexp.MustCompile(`"features": \[\s*\{\s*"featureId": "paladin_epic_boon"\s*\}\s*\]`).ReplaceAllString(s, `"features": []`)
		s = regexp.MustCompile(`("level": 20,\s*"proficiencyBonus": 6,\s*"features": \[\s*\{\s*"kind": "subclassFeature"\s*\})`).ReplaceAllString(s, `$1, {"featureId": "paladin_epic_boon"}`)
		return strings.Replace(s, "\"id\": \"paladin_epic_boon\",\n          \"name\": \"Epic Boon\",\n          \"level\": 19,", "\"id\": \"paladin_epic_boon\",\n          \"name\": \"Epic Boon\",\n          \"level\": 20,", 1)
	})
	p := must(character.NewPaladinProgression(moved, nil))
	if got := p.FeatEntitlements(character.MustPaladinLevel(20)); !slices.Equal(got, []content.FeatCategory{content.FeatEpicBoon}) {
		t.Errorf("level 20 entitlements = %v, want Epic Boon", got)
	}
	if got := p.FeaturesGainedAt(character.MustPaladinLevel(20)); !slices.Equal(got, []content.FeatureID{"paladin_epic_boon"}) {
		t.Errorf("level 20 features = %v", got)
	}
	if got := p.FeatEntitlements(character.MustPaladinLevel(19)); len(got) != 0 {
		t.Errorf("level 19 entitlements = %v, want none", got)
	}
}
