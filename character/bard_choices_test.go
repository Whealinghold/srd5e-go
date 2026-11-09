package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

type bp = character.BardChoiceProblem

func brdSpell(t *testing.T, id string, level int, classes ...string) content.Spell {
	t.Helper()
	return clcSpell(t, id, level, classes, "")
}

// Requirement: doc/specs/class-bard.md FR-060 to FR-062 (SRD 5.2.1, "Level 2: Expertise", p. 32; "Expertise", p. 182) - Expertise is two distinct skills the Bard is proficient in and does not yet have Expertise in, at level 2 and again at 9; every
// problem (count, repeat, no proficiency, already held) is listed together, in that order; an accepted choice adds Expertise and the BardExpertiseGained event and leaves the argument alone.
// Path: per case (happy / unhappy: invalid input).
func TestChooseBardExpertise(t *testing.T) {
	t.Parallel()
	ath, acr, arc, hist := content.SkillAthletics, content.SkillAcrobatics, content.SkillArcana, content.SkillHistory
	profs := must(character.NewSkillProficiencies([]content.Skill{ath, acr, arc}, []content.Skill{arc}))
	for _, c := range []struct {
		name   string
		chosen []content.Skill
		want   []bp
	}{
		{"two proficient skills", []content.Skill{ath, acr}, nil},
		{"none", nil, []bp{character.BardExpertiseCountWrong}},
		{"one", []content.Skill{ath}, []bp{character.BardExpertiseCountWrong}},
		{"three", []content.Skill{ath, acr, hist}, []bp{character.BardExpertiseCountWrong, character.BardExpertiseNotProficient}},
		{"a repeat", []content.Skill{ath, ath}, []bp{character.BardExpertiseDuplicate}},
		{"no proficiency", []content.Skill{ath, hist}, []bp{character.BardExpertiseNotProficient}},
		{"already held", []content.Skill{ath, arc}, []bp{character.BardExpertiseAlreadyHeld}},
		{"everything", []content.Skill{hist, hist, arc}, []bp{character.BardExpertiseCountWrong, character.BardExpertiseNotProficient, character.BardExpertiseDuplicate, character.BardExpertiseAlreadyHeld}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, problems := character.ChooseBardExpertise(brdID, c.chosen, profs)
			if c.want != nil {
				if !slices.Equal(problems, c.want) || got.Event().CharacterID().String() != "" {
					t.Errorf("problems = %v, want %v", problems, c.want)
				}
				return
			}
			if len(problems) != 0 || problems == nil {
				t.Fatalf("problems = %#v, want an empty non-nil slice", problems)
			}
			for _, s := range c.chosen {
				if !got.Proficiencies().HasExpertise(s) || !got.Proficiencies().Proficient(s) || profs.HasExpertise(s) {
					t.Errorf("%v: new Expertise %v, argument Expertise %v", s, got.Proficiencies().HasExpertise(s), profs.HasExpertise(s))
				}
			}
			if !got.Proficiencies().HasExpertise(arc) || !slices.Equal(got.Event().Skills(), c.chosen) || got.Event().CharacterID() != brdID {
				t.Errorf("event = %v, Arcana kept = %v", got.Event().Skills(), got.Proficiencies().HasExpertise(arc))
			}
		})
	}
}

// Requirement: doc/specs/class-bard.md FR-070, FR-071, FR-110 (SRD 5.2.1, "Level 3: Bonus Proficiencies", p. 35) - Bonus Proficiencies is exactly three distinct skills the Bard is not already proficient in; every problem is listed together.
// Path: per case (happy / unhappy: invalid input).
func TestBonusProficienciesProblems(t *testing.T) {
	t.Parallel()
	ath, acr, arc, hist, stealth := content.SkillAthletics, content.SkillAcrobatics, content.SkillArcana, content.SkillHistory, content.SkillStealth
	profs := must(character.NewSkillProficiencies([]content.Skill{stealth}, nil))
	for _, c := range []struct {
		name   string
		chosen []content.Skill
		want   []bp
	}{
		{"three new skills", []content.Skill{ath, acr, arc}, []bp{}},
		{"two", []content.Skill{ath, acr}, []bp{character.BardSkillCountWrong}},
		{"four", []content.Skill{ath, acr, arc, hist}, []bp{character.BardSkillCountWrong}},
		{"a repeat", []content.Skill{ath, ath, acr}, []bp{character.BardSkillDuplicate}},
		{"a repeat twice is listed once", []content.Skill{ath, ath, ath}, []bp{character.BardSkillDuplicate}},
		{"already proficient", []content.Skill{ath, acr, stealth}, []bp{character.BardSkillAlreadyProficient}},
		{"everything", []content.Skill{stealth, stealth}, []bp{character.BardSkillCountWrong, character.BardSkillDuplicate, character.BardSkillAlreadyProficient}},
		{"invalid skills", []content.Skill{ath, acr, content.Skill(99)}, []bp{character.BardSkillInvalid}},
		{"zero skills", []content.Skill{0, 0, 0}, []bp{character.BardSkillDuplicate, character.BardSkillInvalid}},
		{"just past the last skill", []content.Skill{ath, acr, content.SkillStealth + 20}, []bp{character.BardSkillInvalid}},
	} {
		if got := character.BonusProficienciesProblems(c.chosen, profs); !slices.Equal(got, c.want) || got == nil {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-066, FR-068 (SRD 5.2.1, "Level 3: Bard Subclass", p. 32) - a subclass is chosen from Bard level 3 (level 2 refused) and only once.
// Path: per case (unhappy: boundary, state error).
func TestBardSubclassProblems(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		level int
		held  content.SubclassID
		want  []bp
	}{{3, "", []bp{}}, {2, "", []bp{character.BardSubclassTooEarly}}, {3, lore, []bp{character.BardSubclassAlreadyChosen}}, {1, lore, []bp{character.BardSubclassTooEarly, character.BardSubclassAlreadyChosen}}, {20, "", []bp{}}} {
		if got := character.BardSubclassProblems(character.MustBardLevel(c.level), c.held); !slices.Equal(got, c.want) || got == nil {
			t.Errorf("level %d, held %q: %v, want %v", c.level, c.held, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-076, FR-077, FR-078 (SRD 5.2.1, "Level 6: Magical Discoveries", p. 35) - the Magical Discoveries table at Bard level 6 (slots to level 3): a Cleric-only and a Druid-only spell are accepted; a Bard-only spell is
// refused; a spell on the Bard and the Wizard list is accepted; a level 4 spell is refused; a cantrip is accepted; the count must be two, a repeat is refused, every problem is listed together.
// Path: per case (the table of the spec): happy / unhappy (invalid input, boundary).
func TestMagicalDiscoveriesProblems(t *testing.T) {
	t.Parallel()
	clericOnly, druidOnly := brdSpell(t, "cleric_only", 1, "cleric"), brdSpell(t, "druid_only", 2, "druid")
	bardOnly, wizardOnly := brdSpell(t, "bard_only", 1, "bard"), brdSpell(t, "wizard_only", 3, "wizard")
	bardWizard, level4, cantrip := brdSpell(t, "bard_wizard", 2, "bard", "wizard"), brdSpell(t, "level_four", 4, "wizard"), brdSpell(t, "a_cantrip", 0, "druid")
	for _, c := range []struct {
		name   string
		level  int
		chosen []content.Spell
		want   []bp
	}{
		{"one each from the lists", 6, []content.Spell{clericOnly, druidOnly}, []bp{}},
		{"Bard-only spell", 6, []content.Spell{bardOnly, wizardOnly}, []bp{character.BardSpellNotOnList}},
		{"Bard and another list", 6, []content.Spell{bardWizard, druidOnly}, []bp{}},
		{"level 4 spell", 6, []content.Spell{level4, clericOnly}, []bp{character.BardSpellWithoutSlots}},
		{"level 4 spell at level 7 has slots", 7, []content.Spell{level4, clericOnly}, []bp{}},
		{"a cantrip", 6, []content.Spell{cantrip, wizardOnly}, []bp{}},
		{"one spell", 6, []content.Spell{clericOnly}, []bp{character.BardSpellCountWrong}},
		{"three spells", 6, []content.Spell{clericOnly, druidOnly, wizardOnly}, []bp{character.BardSpellCountWrong}},
		{"a repeat", 6, []content.Spell{clericOnly, clericOnly}, []bp{character.BardSpellDuplicate}},
		{"everything", 6, []content.Spell{bardOnly, bardOnly, level4}, []bp{character.BardSpellCountWrong, character.BardSpellDuplicate, character.BardSpellNotOnList, character.BardSpellWithoutSlots}},
	} {
		if got := character.MagicalDiscoveriesProblems(character.MustBardLevel(c.level), c.chosen); !slices.Equal(got, c.want) || got == nil {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-088, FR-090, FR-091, FR-111 (SRD 5.2.1, "Level 10: Magical Secrets", p. 33) - the lists a prepared spell may come from: the Bard list below level 10 (levels 9 and 10 tested); from level 10 the Bard, Cleric, Druid
// and Wizard lists for a replacement, and for an addition only where the Prepared Spells number rises (levels 10 and 11 yes, 12 no).
// Path: per case (unhappy: boundary).
func TestBardPreparedSpellLists(t *testing.T) {
	t.Parallel()
	bard, four := []content.ClassID{"bard"}, []content.ClassID{"bard", "cleric", "druid", "wizard"}
	for _, c := range []struct {
		level     int
		replacing bool
		want      []content.ClassID
	}{
		{9, false, bard}, {9, true, bard}, {10, false, four}, {10, true, four}, {11, false, four}, {12, false, bard}, {12, true, four},
		{13, false, four}, {14, false, bard}, {14, true, four}, {16, false, bard}, {17, false, four}, {20, false, four},
	} {
		if got := character.BardPreparedSpellLists(character.MustBardLevel(c.level), c.replacing); !slices.Equal(got, c.want) {
			t.Errorf("level %d, replacing %v: %v, want %v", c.level, c.replacing, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-055, FR-089, FR-091 - prepared spells are checked against the allowed lists and the level ceiling (level 3 has slots to level 2; level 5 to 3): a Cleric spell is refused at level 9 and accepted at level 10 when the
// number rises, a spell above the ceiling and a repeat are refused, every problem is listed together; a spell on the Bard list is always accepted.
// Path: per case (happy / unhappy: boundary, invalid input).
func TestPreparedSpellProblems(t *testing.T) {
	t.Parallel()
	bardL1, bardL3 := brdSpell(t, "bard_one", 1, "bard", "wizard"), brdSpell(t, "bard_three", 3, "bard")
	cleric2 := brdSpell(t, "cleric_two", 2, "cleric")
	for _, c := range []struct {
		name      string
		level     int
		chosen    []content.Spell
		replacing bool
		want      []bp
	}{
		{"a Bard spell", 1, []content.Spell{bardL1}, false, []bp{}},
		{"level 3 spell at level 3", 3, []content.Spell{bardL3}, false, []bp{character.BardSpellWithoutSlots}},
		{"level 3 spell at level 5", 5, []content.Spell{bardL3}, false, []bp{}},
		{"a Cleric spell at level 9", 9, []content.Spell{cleric2}, false, []bp{character.BardSpellNotOnList}},
		{"a Cleric spell at level 10", 10, []content.Spell{cleric2}, false, []bp{}},
		{"a Cleric spell added at level 12", 12, []content.Spell{cleric2}, false, []bp{character.BardSpellNotOnList}},
		{"a Cleric spell replacing at level 12", 12, []content.Spell{cleric2}, true, []bp{}},
		{"a repeat", 5, []content.Spell{bardL1, bardL1}, false, []bp{character.BardSpellDuplicate}},
		{"everything", 3, []content.Spell{cleric2, bardL3, bardL3}, false, []bp{character.BardSpellDuplicate, character.BardSpellNotOnList, character.BardSpellWithoutSlots}},
	} {
		if got := character.PreparedSpellProblems(character.MustBardLevel(c.level), c.chosen, c.replacing); !slices.Equal(got, c.want) || got == nil {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-bard.md FR-057, FR-078, FR-095 (SRD 5.2.1, "Level 20: Words of Creation", p. 33) - the spells always prepared are the Magical Discoveries spells from level 6 and Power Word Heal and Power Word Kill from level 20, each once
// (a discovery that is one of the two is not repeated); nothing below level 6.
// Path: per case (happy / unhappy: boundary, rule interaction).
func TestBardAlwaysPreparedSpells(t *testing.T) {
	t.Parallel()
	picks := []content.SpellID{"bless", "power_word_kill"}
	for _, c := range []struct {
		level int
		want  []content.SpellID
	}{
		{5, []content.SpellID{}}, {6, picks}, {19, picks}, {20, []content.SpellID{"bless", "power_word_kill", "power_word_heal"}},
	} {
		if got := character.BardAlwaysPreparedSpells(character.MustBardLevel(c.level), picks); !slices.Equal(got, c.want) {
			t.Errorf("level %d: %v, want %v", c.level, got, c.want)
		}
	}
	if got := character.BardAlwaysPreparedSpells(character.MustBardLevel(20), nil); !slices.Equal(got, []content.SpellID{"power_word_heal", "power_word_kill"}) {
		t.Errorf("level 20 without discoveries: %v", got)
	}
	if got := character.BardAlwaysPreparedSpells(character.MustBardLevel(6), nil); got == nil || len(got) != 0 {
		t.Errorf("level 6 without discoveries: %#v, want an empty non-nil slice", got)
	}
}

// Requirement: doc/specs/class-bard.md FR-046, FR-047 (SRD 5.2.1, "Spellcasting Ability", "Spellcasting Focus", p. 32) - the Bard's Spellcasting Ability is Charisma and its Spellcasting Focus is a Musical Instrument and nothing else; the class id is bard.
// Path: happy / unhappy (invalid input: no other focus).
func TestBardSpellcastingAbilityAndFocus(t *testing.T) {
	t.Parallel()
	if character.CasterBard.Ability() != abilityCharisma || character.CasterBard.ClassID() != character.BardClassID || !slices.Equal(character.CasterBard.FocusKinds(), []character.SpellcastingFocusKind{character.FocusMusicalInstrument}) {
		t.Errorf("Bard row = %v, %v, %v", character.CasterBard.Ability(), character.CasterBard.ClassID(), character.CasterBard.FocusKinds())
	}
}

// Requirement: doc/specs/class-bard.md FR-051, FR-054, FR-058, FR-059 (SRD 5.2.1, "Cantrips", "Spell Slots", "Prepared Spells of Level 1+", p. 32) - the Bard learns one more cantrip only at levels 4 and 10 (2, 3, 4 cantrips), the highest spell level it can prepare follows Table 1
// (1 at levels 1 and 2, 2 at 3, 3 at 5, 5 at 9, 9 at 17), and a Bard's expended slots come back through the generic rest recovery on a Long Rest and not on a Short Rest.
// Path: per case (unhappy: boundary).
func TestBardCountsCeilingAndSlotRecovery(t *testing.T) {
	t.Parallel()
	for n := 1; n <= 20; n++ {
		want := 2
		if n >= 4 {
			want = 3
		}
		if n >= 10 {
			want = 4
		}
		if got := character.CantripLimit(character.CasterBard, n); got != want {
			t.Errorf("cantrips at level %d = %d, want %d", n, got, want)
		}
	}
	for level, want := range map[int]int{1: 1, 2: 1, 3: 2, 5: 3, 9: 5, 17: 9} {
		if got := character.MaxPreparableLevel(character.CasterBard, level); got != want {
			t.Errorf("highest spell level at Bard level %d = %d, want %d", level, got, want)
		}
	}
	state := castingState(t, classLevelsOf(t, "bard", 5), spellCatalogue(t))
	spent := state.ExpendSlot(character.PoolSpellcasting, slotLevel(1)).(character.SlotExpenditureExpended).State()
	full := remaining(state, character.PoolSpellcasting, 1)
	if short, _ := spent.RestoreSlots(character.ShortRest); remaining(short, character.PoolSpellcasting, 1) != full-1 {
		t.Error("a Short Rest restored a Bard spell slot")
	}
	if long, _ := spent.RestoreSlots(character.LongRest); remaining(long, character.PoolSpellcasting, 1) != full {
		t.Error("a Long Rest did not restore the Bard spell slots")
	}
}
