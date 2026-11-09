package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func xpFor(level int) int {
	return character.ExperiencePoints{}.MinimumForLevel(character.MustLevel(level)).Value()
}

// agreesWithLevelUp checks FR-012: the preview is available exactly when
// Character.LevelUp changes the character, and a block holds LevelUp's rejection.
func agreesWithLevelUp(t *testing.T, c character.Character, class content.ClassDefinition, current []content.ClassDefinition, p character.LevelAdvancementPreview) {
	t.Helper()
	adv := must(character.NewAdvancementClass(class.ID(), class.HitPointDie(), class.PrimaryAbility()))
	tr, err := c.LevelUp(adv, must(character.NewFixedHitPointGain(class.HitPointDie().Faces()/2+1)), requirementsOf(current...))
	if err != nil {
		t.Fatal(err)
	}
	switch p := p.(type) {
	case character.LevelAdvancementGain:
		ch, ok := tr.(character.TransitionChanged[character.LevelUpRejection])
		if !ok || p.NewTotalLevel() != ch.Character().TotalLevel() || p.ProficiencyBonusAfter() != ch.Character().ProficiencyBonus() {
			t.Errorf("FR-012: available but LevelUp gave %#v", tr)
		}
	case character.LevelAdvancementBlock:
		rj, ok := tr.(character.TransitionRejected[character.LevelUpRejection])
		if !ok || levelUpRejectionTag(p.Rejection()) != levelUpRejectionTag(rj.Rejection()) {
			t.Errorf("FR-012: unavailable but LevelUp gave %#v", tr)
		}
		if s, isShort := p.Rejection().(character.LevelUpPrerequisitesNotMet); isShort {
			if !slices.Equal(s.ShortAbilities(), rj.Rejection().(character.LevelUpPrerequisitesNotMet).ShortAbilities()) {
				t.Error("FR-012: the short abilities differ from LevelUp's")
			}
		}
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-009 to FR-012,
// FR-024, FR-052, FR-053 - Table PR, one row each: the preview is available for a
// held class and for a new class whose prerequisites are met, otherwise it holds the
// one rejection LevelUp returns, in the fixed order; it agrees with LevelUp row by row.
// Path: per case.
func TestLevelAdvancementPreviewTablePR(t *testing.T) {
	t.Parallel()
	barb, druid, monk, fighter, wiz := advBarbarian(t), advDruid(t), advMonk(t), advFighter(t), advWizard(t)
	sc := func(s, d, i, w int) ability.Scores { return scoresOf(t, s, d, 10, i, w, 10) }
	const (
		avail  = "available"
		xp     = "xp"
		maxLvl = "max"
	)
	for _, c := range []struct {
		name, requirement, path string
		held                    content.ClassDefinition
		heldLevel, xp           int
		choose                  content.ClassDefinition
		scores                  ability.Scores
		want                    string
		short                   []ability.Ability
		needed                  int
	}{
		{"existing class, enough XP", aiSpec + " FR-009, SRD 5.2.1, Gaining a Level, p. 23", "happy", barb, 1, 300, barb, sc(10, 10, 10, 10), avail, nil, 0},
		{"new class, SRD example", aiSpec + " FR-010, SRD 5.2.1, Multiclassing, Prerequisites, p. 24", "happy", barb, 1, 300, druid, sc(13, 10, 10, 13), avail, nil, 0},
		{"new class, Strength short (12 fails)", aiSpec + " FR-011, FR-052", "unhappy (boundary)", barb, 1, 300, druid, sc(12, 10, 10, 13), "prereq", []ability.Ability{ability.Strength}, 0},
		{"new class, Wisdom short", aiSpec + " FR-011, FR-052", "unhappy (boundary)", barb, 1, 300, druid, sc(13, 10, 10, 12), "prereq", []ability.Ability{ability.Wisdom}, 0},
		{"new class, both short", aiSpec + " FR-011, FR-052", "unhappy (boundary)", barb, 1, 300, druid, sc(12, 10, 10, 12), "prereq", []ability.Ability{ability.Strength, ability.Wisdom}, 0},
		{"new class, comfortably above", aiSpec + " FR-052", "happy", barb, 1, 300, druid, sc(15, 10, 10, 18), avail, nil, 0},
		{"existing class skips prerequisites", aiSpec + " FR-011", "unhappy (rule interaction)", barb, 1, 300, barb, sc(8, 10, 10, 10), avail, nil, 0},
		{"Fighter new, Strength 13 only", aiSpec + " FR-052", "unhappy (boundary)", wiz, 1, 300, fighter, sc(13, 8, 13, 10), avail, nil, 0},
		{"Fighter new, Dexterity 13 only", aiSpec + " FR-052", "unhappy (boundary)", wiz, 1, 300, fighter, sc(8, 13, 13, 10), avail, nil, 0},
		{"Fighter new, neither at 13", aiSpec + " FR-052", "unhappy (boundary)", wiz, 1, 300, fighter, sc(12, 12, 13, 10), "prereq", []ability.Ability{ability.Strength, ability.Dexterity}, 0},
		{"current class short", aiSpec + " FR-052", "unhappy (rule interaction)", wiz, 1, 300, fighter, sc(13, 8, 12, 10), "prereq", []ability.Ability{ability.Intelligence}, 0},
		{"two-ability class, both at 13", aiSpec + " FR-052", "happy", barb, 1, 300, monk, sc(13, 13, 10, 13), avail, nil, 0},
		{"two-ability class, Dexterity short", aiSpec + " FR-052", "unhappy (boundary)", barb, 1, 300, monk, sc(13, 12, 10, 13), "prereq", []ability.Ability{ability.Dexterity}, 0},
		{"two-ability class, both short", aiSpec + " FR-052", "unhappy (boundary)", barb, 1, 300, monk, sc(13, 12, 10, 12), "prereq", []ability.Ability{ability.Dexterity, ability.Wisdom}, 0},
		{"not enough XP, existing class", aiSpec + " FR-011, FR-013", "unhappy (boundary)", barb, 1, 299, barb, sc(10, 10, 10, 10), xp, nil, 1},
		{"XP beats prerequisites", aiSpec + " FR-011", "unhappy (rule interaction)", barb, 1, 299, druid, sc(12, 10, 10, 12), xp, nil, 1},
		{"maximum level beats everything", aiSpec + " FR-011, FR-024", "unhappy (boundary)", barb, 20, 400000, druid, sc(12, 10, 10, 12), maxLvl, nil, 0},
		{"maximum level, held class", aiSpec + " FR-024", "unhappy (boundary)", barb, 20, 400000, barb, sc(10, 10, 10, 10), maxLvl, nil, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch := restored(t, c.xp, 40, c.scores, cl(c.held.ID().String(), c.heldLevel))
			p := previewOf(t, ch, c.choose, c.held)
			agreesWithLevelUp(t, ch, c.choose, []content.ClassDefinition{c.held}, p)
			if c.want == avail {
				g, ok := p.(character.LevelAdvancementGain)
				if !ok {
					t.Fatalf("preview = %#v, want available [%s]", p, c.requirement)
				}
				_, isNew := g.ClassStatus().(character.AdvancementClassStatusNew)
				if isNew != (c.choose.ID() != c.held.ID()) {
					t.Errorf("new class = %v", isNew)
				}
				return
			}
			b, ok := p.(character.LevelAdvancementBlock)
			if !ok || b.ExperienceStillNeeded() != c.needed {
				t.Fatalf("preview = %#v, want unavailable needing %d [%s]", p, c.needed, c.requirement)
			}
			switch c.want {
			case xp:
				if _, ok := b.Rejection().(character.LevelUpInsufficientExperience); !ok {
					t.Errorf("rejection = %#v", b.Rejection())
				}
			case maxLvl:
				if _, ok := b.Rejection().(character.LevelUpMaximumLevel); !ok {
					t.Errorf("rejection = %#v", b.Rejection())
				}
			default:
				pn, ok := b.Rejection().(character.LevelUpPrerequisitesNotMet)
				if !ok || !slices.Equal(pn.ShortAbilities(), c.short) {
					t.Errorf("rejection = %#v, want short %v", b.Rejection(), c.short)
				}
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-009, FR-010, FR-053 -
// the class status of a level: Existing carries the level before, New carries the
// class's multiclass grants; Table MS walks Fighter 1, Rogue 1, Fighter 2.
// Path: happy.
func TestLevelAdvancementPreviewClassStatusTableMS(t *testing.T) {
	t.Parallel()
	fighter, rogue := advFighter(t), advRogue(t)
	sc := scoresOf(t, 13, 13, 10, 10, 10, 10)
	type step struct {
		xp     int
		levels []character.ClassLevel
		choose content.ClassDefinition
		isNew  bool
		before int
		class  int
		total  int
	}
	for i, s := range []step{
		{300, []character.ClassLevel{cl("fighter", 1)}, rogue, true, 0, 1, 2},
		{900, []character.ClassLevel{cl("fighter", 1), cl("rogue", 1)}, fighter, false, 1, 2, 3},
		{2700, []character.ClassLevel{cl("fighter", 2), cl("rogue", 1)}, rogue, false, 1, 2, 4},
	} {
		ch := restored(t, s.xp, 40, sc, s.levels...)
		g, ok := previewOf(t, ch, s.choose, fighter, rogue).(character.LevelAdvancementGain)
		if !ok {
			t.Fatalf("step %d unavailable", i+1)
		}
		switch st := g.ClassStatus().(type) {
		case character.AdvancementClassStatusExisting:
			if s.isNew || st.CurrentClassLevel != s.before {
				t.Errorf("step %d: status %#v", i+1, st)
			}
		case character.AdvancementClassStatusNew:
			if !s.isNew || len(st.Grants().WeaponProficiencies()) != 0 {
				t.Errorf("step %d: status %#v", i+1, st)
			}
		}
		if g.NewClassLevel() != s.class || g.NewTotalLevel().Value() != s.total || g.HitPointDie() != s.choose.HitPointDie() {
			t.Errorf("step %d: class level %d total %d die %v", i+1, g.NewClassLevel(), g.NewTotalLevel().Value(), g.HitPointDie())
		}
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-108, FR-109 - a new
// class's status holds its MulticlassGrants and a held class holds none; the Hit
// Point Die is the die of the class the level is gained in.
// Path: per case.
func TestLevelAdvancementPreviewNewClassGrants(t *testing.T) {
	t.Parallel()
	defs := grantClasses(t)
	sc := scoresOf(t, 13, 13, 10, 13, 10, 10)
	fighter1 := restored(t, 300, 40, sc, cl("fighter", 1))
	g := previewOf(t, fighter1, defs["barbarian"], defs["fighter"]).(character.LevelAdvancementGain)
	st, ok := g.ClassStatus().(character.AdvancementClassStatusNew)
	w, a := grantsOf(st.Grants())
	if !ok || !slices.Equal(w, []content.WeaponTrainingCategory{content.WeaponTrainingMartial}) || !slices.Equal(a, []content.ArmorTrainingCategory{content.ArmorTrainingShield}) || g.HitPointDie() != content.HitDieD12 {
		t.Errorf("Barbarian as a new class: %#v die %v", g.ClassStatus(), g.HitPointDie())
	}
	if names := entryNames(g.FeaturesGained()); len(names) == 0 || g.NewClassLevel() != 1 {
		t.Errorf("level 1 features missing: %v", names)
	}
	wiz := previewOf(t, fighter1, defs["wizard"], defs["fighter"]).(character.LevelAdvancementGain)
	if st, ok := wiz.ClassStatus().(character.AdvancementClassStatusNew); !ok || len(st.Grants().WeaponProficiencies())+len(st.Grants().ArmorTraining()) != 0 || wiz.HitPointDie() != content.HitDieD6 {
		t.Errorf("Wizard as a new class: %#v", wiz.ClassStatus())
	}
	two := restored(t, 900, 40, sc, cl("fighter", 1), cl("barbarian", 1))
	held := previewOf(t, two, defs["fighter"], defs["fighter"], defs["barbarian"]).(character.LevelAdvancementGain)
	if _, ok := held.ClassStatus().(character.AdvancementClassStatusExisting); !ok || held.HitPointDie() != content.HitDieD10 {
		t.Errorf("existing class: %#v die %v", held.ClassStatus(), held.HitPointDie())
	}
}

func entryNames(es []content.FeatureEntry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-013 - Table XN:
// ExperienceStillNeeded is the minimum XP for the next total level less the XP, 0
// when not positive and at total level 20.
// Path: per case.
func TestLevelAdvancementPreviewExperienceStillNeeded(t *testing.T) {
	t.Parallel()
	barb, druid := advBarbarian(t), advDruid(t)
	weak := scoresOf(t, 10, 10, 10, 10, 10, 10) // blocks a new Druid on prerequisites
	for _, c := range []struct {
		name, requirement, path string
		level, xp, want         int
	}{
		{"level 1, no XP", aiSpec + " FR-013, SRD 5.2.1, Character Advancement table, p. 23", "unhappy (boundary)", 1, 0, 300},
		{"level 1, one XP short", aiSpec + " FR-013", "unhappy (boundary)", 1, 299, 1},
		{"level 1, exactly enough, blocked on prerequisites", aiSpec + " FR-013", "unhappy (boundary)", 1, 300, 0},
		{"level 1, far more than enough", aiSpec + " FR-013", "unhappy (boundary)", 1, 6500, 0},
		{"level 19, one XP short", aiSpec + " FR-013", "unhappy (boundary)", 19, 354999, 1},
		{"level 19, exactly enough", aiSpec + " FR-013", "unhappy (boundary)", 19, 355000, 0},
		{"level 20", aiSpec + " FR-013, FR-024", "unhappy (boundary)", 20, 400000, 0},
		{"level 20, no XP", aiSpec + " FR-013, FR-024", "unhappy (boundary)", 20, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch := restored(t, c.xp, 40, weak, cl("barbarian", c.level))
			b, ok := previewOf(t, ch, druid, barb).(character.LevelAdvancementBlock)
			if !ok || b.ExperienceStillNeeded() != c.want {
				t.Errorf("preview = %#v, want still needed %d [%s]", b, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-014 - Table PB: the
// Proficiency Bonus after the level is LevelUp's, and it increases exactly when it
// is higher than at the current total level; the bonus follows total level, so a level
// in a second class raises it at the same totals.
// Path: per case.
func TestLevelAdvancementPreviewProficiencyBonus(t *testing.T) {
	t.Parallel()
	barb, fighter, rogue := advBarbarian(t), advFighter(t), advRogue(t)
	sc := scoresOf(t, 13, 13, 10, 10, 10, 10)
	for _, c := range []struct {
		name, requirement string
		levels            []character.ClassLevel
		choose            content.ClassDefinition
		after             int
		increases         bool
	}{
		{"1 to 2", aiSpec + " FR-014, SRD 5.2.1, Character Advancement table, p. 23", []character.ClassLevel{cl("barbarian", 1)}, barb, 2, false},
		{"4 to 5", aiSpec + " FR-014", []character.ClassLevel{cl("barbarian", 4)}, barb, 3, true},
		{"8 to 9", aiSpec + " FR-014", []character.ClassLevel{cl("barbarian", 8)}, barb, 4, true},
		{"12 to 13", aiSpec + " FR-014", []character.ClassLevel{cl("barbarian", 12)}, barb, 5, true},
		{"16 to 17", aiSpec + " FR-014", []character.ClassLevel{cl("barbarian", 16)}, barb, 6, true},
		{"19 to 20", aiSpec + " FR-014", []character.ClassLevel{cl("barbarian", 19)}, barb, 6, false},
		{"Fighter 4 to Fighter 4 / Rogue 1", aiSpec + " FR-014 (SRD 5.2.1, Multiclassing, Proficiency Bonus, p. 25)", []character.ClassLevel{cl("fighter", 4)}, rogue, 3, true},
		{"Fighter 3 / Rogue 2 to Rogue 3", aiSpec + " FR-014", []character.ClassLevel{cl("fighter", 3), cl("rogue", 2)}, rogue, 3, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			total := 0
			for _, e := range c.levels {
				total += e.Level
			}
			ch := restored(t, xpFor(total+1), 40, sc, c.levels...)
			g, ok := previewOf(t, ch, c.choose, barb, fighter, rogue).(character.LevelAdvancementGain)
			if !ok {
				t.Fatalf("unavailable [%s]", c.requirement)
			}
			if g.ProficiencyBonusAfter().Value() != c.after || g.ProficiencyBonusIncreases() != c.increases {
				t.Errorf("after %d increases %v, want %d %v [%s]", g.ProficiencyBonusAfter().Value(), g.ProficiencyBonusIncreases(), c.after, c.increases, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-015 to FR-017,
// FR-051 - Table PC over the Wizard Features table (Ability Score Improvement at 4,
// 8, 12 and 16, Epic Boon at 19, the subclass at 3, "Subclass feature" at 6, 10, 14):
// features are the row's resolved entries (a repeated feature is its older entry),
// pending choices are the subclass owed then one feat grant per General or Epic
// Boon entry with the older FeatureID and the new ClassLevel, and a pending choice
// never makes the preview unavailable. The Fighter and Rogue rows need the Phase 3
// class data (step 9) and are deferred; the rule they exercise is the same.
// Path: per case.
func TestLevelAdvancementPreviewTablePC(t *testing.T) {
	t.Parallel()
	wiz := advWizard(t, "evoker")
	sc := flatScores(t, 10)
	const asi, boon = "wizard_ability_score_improvement", "wizard_epic_boon"
	type feat struct {
		id    string
		level int
		cat   content.FeatCategory
	}
	for _, c := range []struct {
		name, requirement, path string
		level                   int  // new class level
		subclass                bool // evoker held
		subclassPending         bool
		feats                   []feat
		subclassRow             bool
		featureIDs              []string
	}{
		{"level 4", aiSpec + " FR-016", "happy", 4, true, false, []feat{{asi, 4, content.FeatGeneral}}, false, []string{asi}},
		{"level 6, subclass row", aiSpec + " FR-015, FR-017", "happy", 6, true, false, nil, true, nil},
		{"level 8, repeated grant", aiSpec + " FR-015, FR-016", "unhappy (rule interaction)", 8, true, false, []feat{{asi, 8, content.FeatGeneral}}, false, []string{asi}},
		{"level 12, repeated grant", aiSpec + " FR-016", "unhappy (rule interaction)", 12, true, false, []feat{{asi, 12, content.FeatGeneral}}, false, []string{asi}},
		{"level 16, repeated grant", aiSpec + " FR-016", "unhappy (rule interaction)", 16, true, false, []feat{{asi, 16, content.FeatGeneral}}, false, []string{asi}},
		{"level 19", aiSpec + " FR-016", "happy", 19, true, false, []feat{{boon, 19, content.FeatEpicBoon}}, false, []string{boon}},
		{"level 3, no subclass", aiSpec + " FR-017", "unhappy (rule interaction)", 3, false, true, nil, false, []string{"wizard_wizard_subclass"}},
		{"level 3, subclass held", aiSpec + " FR-017", "unhappy (rule interaction)", 3, true, false, nil, false, []string{"wizard_wizard_subclass"}},
		{"level 4, subclass still pending", aiSpec + " FR-017, FR-051", "unhappy (rule interaction)", 4, false, true, []feat{{asi, 4, content.FeatGeneral}}, false, []string{asi}},
		{"level 2, below subclass level", aiSpec + " FR-017", "unhappy (boundary)", 2, false, false, nil, false, []string{"wizard_scholar"}},
		{"level 7, no named choice", aiSpec + " FR-015", "unhappy (boundary)", 7, true, false, nil, false, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch := restored(t, xpFor(c.level), 40, sc, cl("wizard", c.level-1))
			if c.subclass {
				ch = holding(t, ch, "wizard", "evoker")
			}
			g, ok := previewOf(t, ch, wiz, wiz).(character.LevelAdvancementGain)
			if !ok {
				t.Fatalf("a pending choice must not make the preview unavailable [%s]", c.requirement)
			}
			var want []character.PendingAdvancementChoice
			if c.subclassPending {
				want = append(want, character.PendingSubclassChoice{Class: "wizard", ClassLevel: c.level})
			}
			for _, f := range c.feats {
				want = append(want, character.PendingFeatGrant{Class: "wizard", ClassLevel: f.level, Feature: content.FeatureID(f.id), Category: f.cat})
			}
			if got := g.PendingChoices(); !slices.Equal(got, want) {
				t.Errorf("pending = %v, want %v [%s]", got, want, c.requirement)
			}
			var ids []string
			for _, e := range g.FeaturesGained() {
				ids = append(ids, e.ID().String())
			}
			if !slices.Equal(ids, c.featureIDs) || g.HasSubclassFeature() != c.subclassRow {
				t.Errorf("features %v subclass row %v, want %v %v [%s]", ids, g.HasSubclassFeature(), c.featureIDs, c.subclassRow, c.requirement)
			}
			if g.NewClassLevel() != c.level {
				t.Errorf("new class level = %d", g.NewClassLevel())
			}
		})
	}
	// A repeated grant reports the older entry, whose own level is lower.
	ch := restored(t, xpFor(8), 40, sc, cl("wizard", 7))
	g := previewOf(t, holding(t, ch, "wizard", "evoker"), wiz, wiz).(character.LevelAdvancementGain)
	if e := g.FeaturesGained(); len(e) != 1 || e[0].Level() != 4 {
		t.Errorf("level 8 features = %v, want the level 4 entry", entryNames(e))
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-015, FR-016 - a
// feat grant is matched by its typed fact and FeatureID, never by display name: a
// renamed Ability Score Improvement is still reported, and a grant of another category
// (Fighting Style) is not.
// Path: unhappy (rule interaction).
func TestLevelAdvancementPreviewMatchesByFactNotName(t *testing.T) {
	t.Parallel()
	sc := flatScores(t, 10)
	spec := func(edits ...[2]string) advSpec {
		return advSpec{id: "wizard", die: "D6", kind: "allOf", abilities: []string{"intelligence"}, subclass: []string{"evoker"}, replace: edits}
	}
	ch := holding(t, restored(t, xpFor(4), 40, sc, cl("wizard", 3)), "wizard", "evoker")
	renamed := advClass(t, spec([2]string{`"name": "Ability Score Improvement"`, `"name": "Something Else"`}))
	if g := previewOf(t, ch, renamed, renamed).(character.LevelAdvancementGain); len(g.PendingChoices()) != 1 {
		t.Errorf("renamed grant: pending = %v", g.PendingChoices())
	}
	style := advClass(t, spec([2]string{`"category": "General"`, `"category": "Fighting Style"`}))
	if g := previewOf(t, ch, style, style).(character.LevelAdvancementGain); len(g.PendingChoices()) != 0 {
		t.Errorf("Fighting Style grant: pending = %v", g.PendingChoices())
	}
	origin := advClass(t, spec([2]string{`"category": "General"`, `"category": "Origin"`}))
	if g := previewOf(t, ch, origin, origin).(character.LevelAdvancementGain); len(g.PendingChoices()) != 0 {
		t.Errorf("Origin grant: pending = %v", g.PendingChoices())
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-018, FR-019 - Table
// SS: SpellSlotsAfter is the full slots of both pools for the class levels after the
// level, from the Multiclass Spellcaster table when two or more classes cast, and the
// class levels of an available preview differ from the current ones by one level.
// Path: per case.
func TestLevelAdvancementPreviewSpellSlotsAfter(t *testing.T) {
	t.Parallel()
	mk := func(id string, abilities ...string) content.ClassDefinition {
		return advClass(t, advSpec{id: id, die: "D8", kind: "allOf", abilities: abilities})
	}
	classes := map[string]content.ClassDefinition{
		"ranger": mk("ranger", "dexterity"), "sorcerer": mk("sorcerer", "charisma"), "fighter": advFighter(t), "wizard": advWizard(t),
		"cleric": mk("cleric", "wisdom"), "warlock": mk("warlock", "charisma"),
	}
	sc := scoresOf(t, 13, 13, 10, 13, 13, 13)
	for _, c := range []struct {
		name, requirement string
		levels            []character.ClassLevel
		choose            string
		slots             [3]int // slots of levels 1 to 3
		pact              int    // Pact Magic slots
		pactLevel         int
	}{
		{"SRD example", aiSpec + " FR-018, SRD 5.2.1, Multiclassing, Spellcasting, p. 26", []character.ClassLevel{cl("ranger", 4), cl("sorcerer", 2)}, "sorcerer", [3]int{4, 3, 2}, 0, 0},
		{"first spellcaster level", aiSpec + " FR-018", []character.ClassLevel{cl("fighter", 3)}, "wizard", [3]int{2, 0, 0}, 0, 0},
		{"second spellcasting class", aiSpec + " FR-018", []character.ClassLevel{cl("wizard", 3)}, "cleric", [3]int{4, 3, 0}, 0, 0},
		{"non-spellcasting class adds nothing", aiSpec + " FR-018", []character.ClassLevel{cl("wizard", 5)}, "fighter", [3]int{4, 3, 2}, 0, 0},
		{"Warlock is a separate pool", aiSpec + " FR-018", []character.ClassLevel{cl("warlock", 5), cl("wizard", 3)}, "wizard", [3]int{4, 3, 0}, 2, 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			total := 1
			for _, e := range c.levels {
				total += e.Level
			}
			ch := restored(t, xpFor(total), 60, sc, c.levels...)
			var current []content.ClassDefinition
			for _, e := range c.levels {
				current = append(current, classes[e.Class.String()])
			}
			g, ok := previewOf(t, ch, classes[c.choose], current...).(character.LevelAdvancementGain)
			if !ok {
				t.Fatalf("unavailable [%s]", c.requirement)
			}
			s := g.SpellSlotsAfter()
			for i, want := range c.slots {
				l := character.MustSlotLevel(i + 1)
				if s.Spellcasting().Maximum(l) != want || s.Spellcasting().Remaining(l) != want {
					t.Errorf("level %d slots = %d/%d, want %d [%s]", i+1, s.Spellcasting().Remaining(l), s.Spellcasting().Maximum(l), want, c.requirement)
				}
			}
			if s.PactMagic().Count() != c.pact || (c.pact > 0 && s.PactMagic().Level() != character.MustSlotLevel(c.pactLevel)) {
				t.Errorf("pact magic = %d slots, want %d", s.PactMagic().Count(), c.pact)
			}
			// FR-019: the new class levels are the current ones plus exactly one level in the class.
			want := character.MustClassLevels(c.levels)
			next := must(want.WithLevelIn(classes[c.choose].ID()))
			if next.TotalLevel().Value() != want.TotalLevel().Value()+1 || g.NewTotalLevel() != next.TotalLevel() {
				t.Errorf("FR-019: total level %d", g.NewTotalLevel().Value())
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-023, FR-105 - the
// previewer leaves the Character unaltered, returns an equal result for the same
// inputs and exposes only copies.
// Path: unhappy (immutability, determinism).
func TestLevelAdvancementPreviewIsPureAndCopies(t *testing.T) {
	t.Parallel()
	wiz := advWizard(t, "evoker")
	ch := restored(t, xpFor(4), 40, flatScores(t, 10), cl("wizard", 3))
	snapshot := ch
	g1 := previewOf(t, ch, wiz, wiz).(character.LevelAdvancementGain)
	g2 := previewOf(t, ch, wiz, wiz).(character.LevelAdvancementGain)
	if !ch.Equal(snapshot) {
		t.Error("the character must be unchanged")
	}
	if !slices.Equal(g1.PendingChoices(), g2.PendingChoices()) || g1.NewTotalLevel() != g2.NewTotalLevel() || g1.SpellSlotsAfter() != g2.SpellSlotsAfter() || g1.HasSubclassFeature() != g2.HasSubclassFeature() {
		t.Error("equal inputs must give equal previews")
	}
	g1.PendingChoices()[0] = nil
	g1.FeaturesGained()[0] = content.FeatureEntry{}
	if g1.PendingChoices()[0] == nil || g1.FeaturesGained()[0].ID() == "" {
		t.Error("PendingChoices and FeaturesGained must return copies")
	}
	// A block holds nothing mutable, but reading it twice is stable.
	b := previewOf(t, restored(t, 0, 40, flatScores(t, 10), cl("wizard", 3)), wiz, wiz).(character.LevelAdvancementBlock)
	if b.ClassStatus() != (character.AdvancementClassStatusExisting{CurrentClassLevel: 3}) {
		t.Errorf("status = %#v", b.ClassStatus())
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-012 - a zero Character,
// a zero class and, for a new class, a requirements map lacking a current class are
// invalid-input errors, not previews.
// Path: unhappy (invalid input).
func TestLevelAdvancementPreviewInvalidInput(t *testing.T) {
	t.Parallel()
	wiz, fighter := advWizard(t), advFighter(t)
	ch := restored(t, 300, 40, flatScores(t, 13), cl("wizard", 1))
	for name, fn := range map[string]func() error{
		"zero character": func() error {
			_, e := character.LevelAdvancementPreviewer{}.Preview(character.Character{}, wiz, nil)
			return e
		},
		"zero class": func() error {
			_, e := character.LevelAdvancementPreviewer{}.Preview(ch, content.ClassDefinition{}, nil)
			return e
		},
		"new class without the current requirements": func() error {
			_, e := character.LevelAdvancementPreviewer{}.Preview(ch, fighter, nil)
			return e
		},
	} {
		if err := fn(); !isInvalid(err) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A held class needs no requirements.
	if _, err := (character.LevelAdvancementPreviewer{}).Preview(ch, wiz, nil); err != nil {
		t.Errorf("held class without requirements: %v", err)
	}
}
