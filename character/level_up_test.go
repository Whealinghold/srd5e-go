package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

var (
	str, dex, con, intl, wis, cha = ability.Strength, ability.Dexterity, ability.Constitution, ability.Intelligence, ability.Wisdom, ability.Charisma
)

// Requirement: doc/specs/character-aggregate.md FR-025, FR-036, FR-037, FR-040,
// FR-042, FR-054, FR-097 - a level in a held class raises it in place by one,
// adds the gain and one die, and emits LevelGained; the input is unchanged.
// Path: happy.
func TestLevelUpHeldClass(t *testing.T) {
	t.Parallel()
	c := restored(t, 2700, 20, scoresOf(t, 10, 10, 14, 10, 10, 10), cl("fighter", 2), cl("rogue", 1))
	before := c
	got := changedLevel(t, levelUp(t, c, advancement(t, "fighter", allOf(str)), fixedGain(6), nil))
	n := got.Character()
	if n.ClassLevels().LevelIn("fighter") != 3 || n.TotalLevel().Value() != 4 || n.ClassLevels().Entries()[0].Class != "fighter" || n.ClassLevels().Len() != 2 {
		t.Errorf("classes = %v", n.ClassLevels().Entries())
	}
	if n.HitPointMaximum().Value() != 20+8 || n.HitDicePool().Count(content.HitDieD10) != 3 || n.HitDicePool().Total() != 4 {
		t.Errorf("maximum %d, d10 %d", n.HitPointMaximum().Value(), n.HitDicePool().Count(content.HitDieD10))
	}
	evs := got.Events()
	lg, ok := evs[0].(character.LevelGained)
	if len(evs) != 1 || !ok || lg.CharacterID() != c.ID() || lg.Class() != "fighter" || lg.ClassLevel() != 3 || lg.TotalLevel().Value() != 4 ||
		lg.HitPointsGained() != 8 || lg.HitPointMaximum().Value() != 28 || lg.ProficiencyBonus().Value() != 2 {
		t.Errorf("event = %#v", evs)
	}
	if !c.Equal(before) {
		t.Error("the input character must be unchanged")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-033, FR-034, FR-036, FR-039 -
// the maximum grows by the gain table's value (roll or fixed, plus the
// Constitution modifier, at least 1), never by a level 1 maximum-face value.
// Path: per case.
func TestLevelUpHitPointGain(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, class string
		fixed       bool
		value, con  int
		want        int
	}{
		{"roll average", "cleric", false, 5, 14, 7}, {"roll maximum", "cleric", false, 8, 10, 8}, {"roll minimum", "cleric", false, 1, 10, 1},
		{"roll one, Con -1", "cleric", false, 1, 8, 1}, {"roll two, Con -5", "wizard", false, 2, 1, 1}, {"roll exactly clamp edge", "wizard", false, 3, 6, 1},
		{"roll one above clamp edge", "wizard", false, 4, 6, 2}, {"fixed Barbarian", "barbarian", true, 7, 14, 9}, {"fixed Fighter", "fighter", true, 6, 14, 8},
		{"fixed Bard", "bard", true, 5, 10, 5}, {"fixed Wizard", "wizard", true, 4, 10, 4}, {"fixed Wizard, Con -5", "wizard", true, 4, 1, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ch := restored(t, 14000, 30, scoresOf(t, 10, 10, c.con, 10, 10, 10), cl(c.class, 5))
			var gain character.HitPointGain = rolledGain(c.value, dieOf(c.class))
			if c.fixed {
				gain = fixedGain(c.value)
			}
			got := changedLevel(t, levelUp(t, ch, advancement(t, c.class, allOf(str)), gain, nil))
			if d := got.Character().HitPointMaximum().Value() - 30; d != c.want {
				t.Errorf("gain %d, want %d", d, c.want)
			}
			if lg := got.Events()[0].(character.LevelGained); lg.HitPointsGained() != c.want {
				t.Errorf("event gain %d", lg.HitPointsGained())
			}
		})
	}
}

// Requirement: doc/specs/character-aggregate.md FR-035 - a rolled gain of another
// die, or a fixed value other than the die's fixed value, is a validation error.
// Path: per case.
func TestLevelUpGainValidation(t *testing.T) {
	t.Parallel()
	ch := restored(t, 14000, 30, flatScores(t, 10), cl("fighter", 5))
	for _, c := range []struct {
		name, class string
		gain        character.HitPointGain
		ok          bool
	}{
		{"roll of another die", "cleric", rolledGain(5, content.HitDieD10), false}, {"roll of the class die", "cleric", rolledGain(5, content.HitDieD8), true},
		{"fixed 100", "fighter", fixedGain(100), false}, {"fixed value of the d8", "fighter", fixedGain(5), false}, {"fixed value of the d10", "fighter", fixedGain(6), true},
		{"fixed one above d6", "wizard", fixedGain(5), false}, {"nil gain", "fighter", nil, false},
		{"zero rolled gain", "fighter", character.RolledHitPointGain{}, false}, {"zero fixed gain", "fighter", character.FixedHitPointGain{}, false},
	} {
		reqs := map[content.ClassID]content.AbilityRequirement{"fighter": allOf(str)}
		ch2 := ch
		_, err := ch2.LevelUp(advancement(t, c.class, allOf(str)), c.gain, reqs)
		if (err == nil) != c.ok {
			t.Errorf("%s: %v", c.name, err)
		} else if !c.ok {
			wantInvalid(t, err, c.name)
		}
	}
	if _, err := ch.LevelUp(character.AdvancementClass{}, fixedGain(6), nil); !isInvalid(err) {
		t.Errorf("zero class: %v", err)
	}
	if _, err := (character.Character{}).LevelUp(advancement(t, "fighter", allOf(str)), fixedGain(6), nil); !isInvalid(err) {
		t.Errorf("zero character: %v", err)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-027, FR-028, FR-029, FR-030,
// FR-043 - the rejection reason follows the order maximum level, insufficient
// experience, prerequisites; eligibility uses the total level.
// Path: per case.
func TestLevelUpRejectionOrder(t *testing.T) {
	t.Parallel()
	weak := flatScores(t, 10)
	for _, c := range []struct {
		name      string
		level, xp int
		newClass  bool
		scores    ability.Scores
		want      string
	}{
		{"maximum beats everything", 20, 400000, true, weak, "max"},
		{"maximum at the threshold", 20, 355000, false, weak, "max"},
		{"experience beats prerequisites", 5, 6500, true, weak, "xp"},
		{"experience, one short", 5, 13999, false, weak, "xp"},
		{"exactly the minimum succeeds", 5, 14000, false, weak, ""},
		{"prerequisites unmet", 5, 14000, true, weak, "prereq"},
		{"prerequisites met", 5, 14000, true, flatScores(t, 13), ""},
	} {
		ch := restored(t, c.xp, 200, c.scores, cl("fighter", c.level))
		target := "fighter"
		if c.newClass {
			target = "wizard"
		}
		tr := levelUp(t, ch, advancement(t, target, allOf(intl)), fixedGain(dieOf(target).Faces()/2+1), map[content.ClassID]content.AbilityRequirement{"fighter": allOf(str)})
		if c.want == "" {
			changedLevel(t, tr)
			continue
		}
		got := levelUpRejectionTag(rejectedLevel(t, tr))
		if got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	// FR-028: a level 6 Cleric / level 1 Fighter needs the XP of level 8.
	ch := restored(t, 33999, 80, flatScores(t, 15), cl("cleric", 6), cl("fighter", 1))
	for _, class := range []string{"cleric", "fighter"} {
		got := rejectedLevel(t, levelUp(t, ch, advancement(t, class, allOf(str)), fixedGain(dieOf(class).Faces()/2+1), nil))
		if got != (character.LevelUpInsufficientExperience{}) {
			t.Errorf("%s one XP short of level 8: %#v", class, got)
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-026, FR-031, FR-032, FR-038,
// FR-039, FR-095 - a new class is appended at level 1 when the Primary Ability of
// the new and every current class is 13 or more; unmet abilities are listed once
// in declaration order; a held class skips the check (SRD 5.2.1, "Multiclassing",
// p. 24).
// Path: per case.
func TestLevelUpPrerequisites(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		existing  content.AbilityRequirement
		next      content.AbilityRequirement
		scores    [6]int // str dex con int wis cha
		sameClass bool
		short     []ability.Ability // nil = allowed
	}{
		{"example from the SRD", allOf(str), allOf(wis), [6]int{13, 10, 10, 10, 13, 10}, false, nil},
		{"existing class ability short", allOf(str), allOf(wis), [6]int{12, 10, 10, 10, 13, 10}, false, []ability.Ability{str}},
		{"new class ability short", allOf(str), allOf(wis), [6]int{13, 10, 10, 10, 12, 10}, false, []ability.Ability{wis}},
		{"both short", allOf(str), allOf(wis), [6]int{12, 10, 10, 10, 12, 10}, false, []ability.Ability{str, wis}},
		{"comfortably above", allOf(str), allOf(wis), [6]int{15, 10, 10, 10, 18, 10}, false, nil},
		{"same class, scores low", allOf(str), allOf(str), [6]int{8, 10, 10, 10, 10, 10}, true, nil},
		{"two primary abilities", allOf(dex, wis), allOf(cha), [6]int{10, 13, 10, 10, 13, 13}, false, nil},
		{"two primary abilities, one short", allOf(dex, wis), allOf(cha), [6]int{10, 13, 10, 10, 12, 13}, false, []ability.Ability{wis}},
		{"Fighter as new class, Dexterity only", allOf(wis), anyOf(str, dex), [6]int{8, 13, 10, 10, 13, 10}, false, nil},
		{"Fighter as new class, Strength only", allOf(wis), anyOf(str, dex), [6]int{13, 8, 10, 10, 13, 10}, false, nil},
		{"Fighter as new class, both short", allOf(wis), anyOf(str, dex), [6]int{12, 12, 10, 10, 13, 10}, false, []ability.Ability{str, dex}},
		{"Fighter as existing class, satisfied by Dexterity", anyOf(str, dex), allOf(intl), [6]int{8, 13, 10, 13, 10, 10}, false, nil},
		{"Fighter as existing class, new ability short", anyOf(str, dex), allOf(intl), [6]int{8, 13, 10, 12, 10, 10}, false, []ability.Ability{intl}},
		{"two classes unmet, deduplicated", allOf(str), allOf(str, wis), [6]int{10, 10, 10, 10, 10, 10}, false, []ability.Ability{str, wis}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := scoresOf(t, c.scores[0], c.scores[1], c.scores[2], c.scores[3], c.scores[4], c.scores[5])
			ch := restored(t, 14000, 50, s, cl("fighter", 5))
			target := "wizard"
			if c.sameClass {
				target = "fighter"
			}
			tr := levelUp(t, ch, advancement(t, target, c.next), fixedGain(dieOf(target).Faces()/2+1), map[content.ClassID]content.AbilityRequirement{"fighter": c.existing})
			if c.short == nil {
				n := changedLevel(t, tr).Character()
				if c.sameClass {
					return
				}
				entries := n.ClassLevels().Entries()
				if last := entries[len(entries)-1]; last.Class != "wizard" || last.Level != 1 || n.TotalLevel().Value() != 6 {
					t.Errorf("classes = %v", entries)
				}
				return
			}
			rej, ok := rejectedLevel(t, tr).(character.LevelUpPrerequisitesNotMet)
			if !ok || !slices.Equal(rej.ShortAbilities(), c.short) {
				t.Errorf("rejection = %#v, want short %v", rej, c.short)
			}
		})
	}
	short := character.LevelUpPrerequisitesNotMet{}
	if short.ShortAbilities() != nil {
		t.Error("the zero rejection lists nothing")
	}
}

// Requirement: doc/specs/character-aggregate.md FR-038, FR-039, FR-037 - pooling:
// repeated level-ups hold ten d10s for Fighter 5 / Paladin 5 and five d8s plus
// five d10s for Cleric 5 / Paladin 5; a new class adds only the gain.
// Path: unhappy (rule interaction).
func TestLevelUpPooling(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		first     string
		d10s, d8s int
	}{{"fighter", 10, 0}, {"cleric", 5, 5}} {
		xp := (character.ExperiencePoints{}).MinimumForLevel(character.MustLevel(10)).Value()
		ch := restored(t, xp, 100, flatScores(t, 14), cl(c.first, 5))
		reqs := map[content.ClassID]content.AbilityRequirement{content.ClassID(c.first): allOf(str)}
		for i := 0; i < 5; i++ {
			ch = changedLevel(t, levelUp(t, ch, advancement(t, "paladin", allOf(str, cha)), fixedGain(6), reqs)).Character()
		}
		if ch.HitDicePool().Count(content.HitDieD10) != c.d10s || ch.HitDicePool().Count(content.HitDieD8) != c.d8s || ch.HitDicePool().Total() != 10 {
			t.Errorf("%s/paladin pool = %v", c.first, ch.HitDicePool().Entries())
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-102 - a new class with a
// requirements map lacking a current class is a validation error, even at level
// 20; a held class ignores the map.
// Path: unhappy (invalid input).
func TestLevelUpRequirementsMap(t *testing.T) {
	t.Parallel()
	for _, level := range []int{5, 20} {
		ch := restored(t, 400000, 300, flatScores(t, 15), cl("fighter", level))
		if _, err := ch.LevelUp(advancement(t, "wizard", allOf(intl)), fixedGain(4), nil); !isInvalid(err) {
			t.Errorf("level %d: %v", level, err)
		}
		if _, err := ch.LevelUp(advancement(t, "wizard", allOf(intl)), fixedGain(4), map[content.ClassID]content.AbilityRequirement{"fighter": nil}); !isInvalid(err) {
			t.Errorf("level %d nil entry: %v", level, err)
		}
	}
	ch := restored(t, 14000, 50, flatScores(t, 10), cl("fighter", 5))
	if _, err := ch.LevelUp(advancement(t, "fighter", allOf(str)), fixedGain(6), nil); err != nil {
		t.Errorf("held class with no map: %v", err)
	}
	extra := map[content.ClassID]content.AbilityRequirement{"rogue": allOf(dex)}
	if _, err := ch.LevelUp(advancement(t, "fighter", allOf(str)), fixedGain(6), extra); err != nil {
		t.Errorf("held class with an unrelated entry: %v", err)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-041, FR-040 - the event's
// Proficiency Bonus is the new total level's; it rises at 4 to 5, 8 to 9, 12 to 13
// and 16 to 17.
// Path: per case.
func TestLevelUpProficiencyBoundaries(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ from, before, after int }{{4, 2, 3}, {8, 3, 4}, {12, 4, 5}, {16, 5, 6}, {1, 2, 2}, {19, 6, 6}} {
		xp := (character.ExperiencePoints{}).MinimumForLevel(character.MustLevel(c.from + 1)).Value()
		ch := restored(t, xp, 300, flatScores(t, 10), cl("fighter", c.from))
		got := changedLevel(t, levelUp(t, ch, advancement(t, "fighter", allOf(str)), fixedGain(6), nil))
		if ch.ProficiencyBonus().Value() != c.before || got.Events()[0].(character.LevelGained).ProficiencyBonus().Value() != c.after {
			t.Errorf("level %d to %d: %d to %d", c.from, c.from+1, ch.ProficiencyBonus().Value(), got.Events()[0].(character.LevelGained).ProficiencyBonus().Value())
		}
	}
}

// Requirement: doc/specs/character-aggregate.md FR-042, FR-057, FR-084, FR-096 -
// a rejection leaves the input equal; the same inputs give equal outcomes; level
// 20 with every roll 1 at Constitution 1 keeps a maximum of at least the level.
// Path: unhappy (determinism).
func TestLevelUpDeterminismAndFloor(t *testing.T) {
	t.Parallel()
	ch := restored(t, 0, 10, flatScores(t, 10), cl("fighter", 1))
	before := ch
	rej := rejectedLevel(t, levelUp(t, ch, advancement(t, "fighter", allOf(str)), fixedGain(6), nil))
	if rej != (character.LevelUpInsufficientExperience{}) || !ch.Equal(before) {
		t.Error("a rejection leaves the input unchanged")
	}
	low := scoresOf(t, 10, 10, 1, 10, 10, 10)
	c := character.MustNew(newParams(t, "wizard", low)).Character()
	c2 := character.MustNew(newParams(t, "wizard", low))
	if !c.Equal(c2.Character()) {
		t.Error("New must be deterministic")
	}
	xp := character.MustExperiencePoints(355000)
	p := newParams(t, "wizard", low)
	p.ExperiencePoints = xp
	c = character.MustNew(p).Character()
	for c.TotalLevel().Value() < 20 {
		c = changedLevel(t, levelUp(t, c, advancement(t, "wizard", allOf(intl)), rolledGain(1, content.HitDieD6), nil)).Character()
	}
	if c.HitPointMaximum().Value() != 20 {
		t.Errorf("maximum at level 20 = %d, want 20", c.HitPointMaximum().Value())
	}
}

// Requirement: doc/specs/character-aggregate.md FR-085 - every event carries the
// id of its character and the seven are RecordEvents.
// Path: happy.
func TestEventsCarryTheCharacterID(t *testing.T) {
	t.Parallel()
	n := character.MustNew(newParams(t, "fighter", scoresOf(t, 10, 10, 17, 10, 10, 10)))
	c := n.Character()
	events := n.Events()
	events = append(events, c.AwardExperience(character.MustExperienceAmount(300)).Events()...)
	events = append(events, c.GrantHeroicInspiration().Events()...)
	events = append(events, c.GrantHeroicInspiration().Character().ExpendHeroicInspiration().(character.TransitionChanged[character.HeroicInspirationRejection]).Events()...)
	xp := restored(t, 300, 12, scoresOf(t, 10, 10, 17, 10, 10, 10), cl("fighter", 1))
	events = append(events, changedLevel(t, levelUp(t, xp, advancement(t, "fighter", allOf(str)), fixedGain(6), nil)).Events()...)
	ab, _ := c.ChangeAbilityScore(con, ability.MustScore(18), 20)
	events = append(events, ab.(character.TransitionChanged[character.AbilityScoreChangeRejection]).Events()...)
	if len(events) != 7 {
		t.Fatalf("%d events, want one of each of the seven", len(events))
	}
	for _, e := range events {
		rec, ok := e.(character.RecordEvent)
		if !ok || e.CharacterID() != c.ID() {
			t.Errorf("%T: record event %v, id %v", e, ok, e.CharacterID())
		}
		switch rec.(type) {
		case character.Created, character.ExperiencePointsGained, character.LevelGained, character.AbilityScoreChanged,
			character.HitPointMaximumChanged, character.HeroicInspirationGained, character.HeroicInspirationExpended:
		default:
			t.Errorf("%T is not one of the seven", rec)
		}
	}
}

func levelUpRejectionTag(r character.LevelUpRejection) string {
	switch r.(type) {
	case character.LevelUpMaximumLevel:
		return "max"
	case character.LevelUpInsufficientExperience:
		return "xp"
	case character.LevelUpPrerequisitesNotMet:
		return "prereq"
	}
	return "?"
}
