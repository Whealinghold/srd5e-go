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

// gainedFor builds the LevelGained event of the character id reaching total level n, the way the aggregate does.
func gainedFor(t *testing.T, id string, n int) character.LevelGained {
	t.Helper()
	lv := levels(cl("fighter", n-1))
	die := map[content.ClassID]content.HitPointDie{"fighter": content.HitDieD10}
	before, err := character.Restore(character.Parts{
		ID: character.MustIdentifier(id), ExperiencePoints: character.ExperiencePoints{}.MinimumForLevel(character.MustLevel(n)), ClassLevels: lv,
		Species: "human", Background: "soldier", AbilityScores: flatScores(t, 10), HitPointMaximum: character.MustHitPointMaximum(50),
		HitDicePool: character.MustHitDicePoolFromLevels(lv, die), HeroicInspiration: character.NotInspired,
	})
	if err != nil {
		t.Fatal(err)
	}
	res := levelUp(t, before, advancement(t, "fighter", anyOf(ability.Strength, ability.Dexterity)), fixedGain(6), nil)
	for _, e := range changedLevel(t, res).Events() {
		if lg, ok := e.(character.LevelGained); ok {
			return lg
		}
	}
	t.Fatal("no LevelGained")
	return character.LevelGained{}
}

func gainedBy(t *testing.T, c character.Character) character.LevelGained {
	t.Helper()
	return gainedFor(t, c.ID().String(), c.TotalLevel().Value())
}

func replace(t *testing.T, c character.Character, feats character.TakenFeats, list string, old string, repl string, ev character.LevelGained) character.FeatsTransition {
	t.Helper()
	tr, err := character.MagicInitiateSpellChange{}.Replace(c, feats, content.ClassID(list), content.SpellID(old), miSpells(t)[content.SpellID(repl)], ev)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

// Requirement: per case
// Path: per case.
func TestMagicInitiateSpellChange(t *testing.T) {
	t.Parallel()
	c := charAt(t, 5)
	feats := heldFeats(t, initiateFeat(), clericInitiate())
	ev := gainedBy(t, c)
	if ev.CharacterID() != c.ID() || ev.TotalLevel().Value() != 5 {
		t.Fatalf("fixture event = %#v", ev)
	}
	otherEvent := gainedFor(t, "pc-2", 5)
	for _, tc := range []struct {
		name, requirement, path string
		old, repl               string
		ev                      character.LevelGained
		char                    character.Character
		want                    character.FeatRejection // nil: accepted
		cantrips                []content.SpellID
		levelOne                content.SpellID
	}{
		{"a cantrip for a cantrip", featsSpec + ` FR-034, Table M; SRD 5.2.1, "Magic Initiate", Spell Change, p. 87`, "happy", "guidance", "mending", ev, c, nil, spellIDsOf("mending", "light"), "bless"},
		{"the level 1 spell for a level 1 spell", featsSpec + " FR-035, Table M", "happy", "bless", "cure_wounds", ev, c, nil, spellIDsOf("guidance", "light"), "cure_wounds"},
		{"a cantrip for a level 1 spell", featsSpec + " FR-035, Table M", "unhappy (invalid input)", "guidance", "cure_wounds", ev, c, character.FeatChoicesInvalid{}, nil, ""},
		{"a spell by itself", featsSpec + " FR-035, Table M", "unhappy (invalid input)", "guidance", "guidance", ev, c, character.FeatChoicesInvalid{}, nil, ""},
		{"a spell by the feat's other spell", featsSpec + " FR-035, Table M", "unhappy (invalid input)", "guidance", "light", ev, c, character.FeatChoicesInvalid{}, nil, ""},
		{"a spell by an off-list spell", featsSpec + " FR-035, Table M", "unhappy (invalid input)", "guidance", "fire_bolt", ev, c, character.FeatChoicesInvalid{}, nil, ""},
		{"a spell the feat does not hold", featsSpec + " FR-035", "unhappy (invalid input)", "mending", "fire_bolt", ev, c, character.FeatChoicesInvalid{}, nil, ""},
		{"no LevelGained", featsSpec + " FR-036, Table M", "unhappy (state error)", "guidance", "mending", character.LevelGained{}, c, character.SpellChangeNotAllowedNow{}, nil, ""},
		{"a level other than the current total", featsSpec + " FR-036, Table M", "unhappy (state error)", "guidance", "mending", ev, restored(t, 0, 50, flatScores(t, 10), cl("fighter", 6)), character.SpellChangeNotAllowedNow{}, nil, ""},
		{"the event of another character", featsSpec + " FR-036, Table M", "unhappy (state error)", "guidance", "mending", otherEvent, c, character.SpellChangeNotAllowedNow{}, nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := replace(t, tc.char, feats, "cleric", tc.old, tc.repl, tc.ev)
			if tc.want == nil {
				ch, ok := tr.(character.FeatsChanged)
				if !ok {
					t.Fatalf("= %#v [%s]", tr, tc.requirement)
				}
				mi := ch.Feats().All()[0].Choices().(character.MagicInitiateChoices)
				if !slices.Equal(mi.Cantrips(), tc.cantrips) || mi.LevelOneSpell() != tc.levelOne || mi.List() != "cleric" || mi.Ability() != ability.Wisdom || !mi.SpellChangedAt(tc.char.TotalLevel()) {
					t.Errorf("choices = %+v", mi)
				}
				ev, ok := ch.Events()[0].(character.FeatSpellReplaced)
				if !ok || ev.Old() != content.SpellID(tc.old) || ev.New() != content.SpellID(tc.repl) || ev.List() != "cleric" || ev.CharacterID() != tc.char.ID() {
					t.Errorf("event = %#v", ch.Events())
				}
				if _, ok := ch.Feats().UseStateOf("magic_initiate").(character.FeatUseIsAvailable); !ok {
					t.Error("Spell Change must not touch the free cast (FR-037)")
				}
				if !feats.All()[0].Choices().(character.MagicInitiateChoices).Equal(clericInitiate()) {
					t.Error("Replace altered its input")
				}
				return
			}
			rj := tr.(character.FeatsRejected).Rejection
			if _, isInvalid := tc.want.(character.FeatChoicesInvalid); isInvalid {
				if _, ok := rj.(character.FeatChoicesInvalid); !ok {
					t.Errorf("rejection = %#v [%s]", rj, tc.requirement)
				}
			} else if rj != tc.want {
				t.Errorf("rejection = %#v, want %#v [%s]", rj, tc.want, tc.requirement)
			}
		})
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-035, Table M "change, invalid" - every problem of a replacement is reported, with the kind of each.
// Path: unhappy (invalid input).
func TestMagicInitiateSpellChangeProblems(t *testing.T) {
	t.Parallel()
	c := charAt(t, 5)
	feats := heldFeats(t, initiateFeat(), clericInitiate())
	ev := gainedBy(t, c)
	p := func(k character.FeatChoiceProblemKind) character.FeatChoiceProblem {
		return character.FeatChoiceProblem{Name: character.FeatChoiceReplacementSpell, Kind: k}
	}
	for _, tc := range []struct {
		old, repl string
		want      []character.FeatChoiceProblem
	}{
		{"guidance", "cure_wounds", []character.FeatChoiceProblem{p(character.FeatProblemLevelMismatch)}},
		{"bless", "mending", []character.FeatChoiceProblem{p(character.FeatProblemLevelMismatch)}},
		{"guidance", "light", []character.FeatChoiceProblem{p(character.FeatProblemDuplicate)}},
		{"bless", "bless", []character.FeatChoiceProblem{p(character.FeatProblemDuplicate)}},
		{"guidance", "fire_bolt", []character.FeatChoiceProblem{p(character.FeatProblemNotOnSpellList)}},
		{"bless", "shield", []character.FeatChoiceProblem{p(character.FeatProblemNotOnSpellList)}},
		{"mending", "fire_bolt", []character.FeatChoiceProblem{p(character.FeatProblemMissing), p(character.FeatProblemNotOnSpellList)}},
		{"guidance", "magic_missile", []character.FeatChoiceProblem{p(character.FeatProblemLevelMismatch), p(character.FeatProblemNotOnSpellList)}},
	} {
		got := replace(t, c, feats, "cleric", tc.old, tc.repl, ev).(character.FeatsRejected).Rejection.(character.FeatChoicesInvalid).Problems()
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s -> %s: %v, want %v", tc.old, tc.repl, got, tc.want)
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-036, Table M "change twice", "change at two levels" - a second replacement of the same instance at the same total level is SpellChangeAlreadyUsed; one at another level is accepted; a list not held is FeatNotHeld.
// Path: unhappy (state error).
func TestMagicInitiateSpellChangeOncePerLevel(t *testing.T) {
	t.Parallel()
	c5 := charAt(t, 5)
	feats := heldFeats(t, initiateFeat(), clericInitiate())
	first := replace(t, c5, feats, "cleric", "guidance", "mending", gainedBy(t, c5)).(character.FeatsChanged).Feats()
	if rj := replace(t, c5, first, "cleric", "light", "guidance", gainedBy(t, c5)).(character.FeatsRejected).Rejection; rj != (character.SpellChangeAlreadyUsed{}) {
		t.Errorf("second at the same level: %#v", rj)
	}
	c6 := charAt(t, 6)
	if _, ok := replace(t, c6, first, "cleric", "light", "guidance", gainedBy(t, c6)).(character.FeatsChanged); !ok {
		t.Error("a replacement at the next level must be accepted")
	}
	if rj := replace(t, c5, feats, "wizard", "light", "fire_bolt", gainedBy(t, c5)).(character.FeatsRejected).Rejection; rj != (character.FeatNotHeld{Feat: "magic_initiate"}) {
		t.Errorf("another list: %#v", rj)
	}
	if rj := replace(t, c5, character.TakenFeats{}, "cleric", "light", "guidance", gainedBy(t, c5)).(character.FeatsRejected).Rejection; rj != (character.FeatNotHeld{Feat: "magic_initiate"}) {
		t.Errorf("no feats: %#v", rj)
	}
	other := heldFeats(t, initiateFeat(), clericInitiate(), initiateFeat(), wizardInitiate())
	one := replace(t, c5, other, "wizard", "light", "magic_missile", gainedBy(t, c5))
	if _, ok := one.(character.FeatsRejected); !ok { // light is a cantrip, magic_missile level 1
		t.Error("a wrong level is rejected")
	}
	ok := replace(t, c5, other, "wizard", "light", "mending", gainedBy(t, c5)).(character.FeatsChanged)
	if !ok.Feats().All()[0].Equal(other.All()[0]) {
		t.Error("another instance must be unchanged")
	}
	if _, err := (character.MagicInitiateSpellChange{}).Replace(character.Character{}, feats, "cleric", "light", miSpells(t)["mending"], character.LevelGained{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero character: %v", err)
	}
	if _, err := (character.MagicInitiateSpellChange{}).Replace(c5, feats, "cleric", "light", content.Spell{}, character.LevelGained{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero spell: %v", err)
	}
	if _, err := (character.MagicInitiateSpellChange{}).Replace(c5, feats, "", "light", miSpells(t)["mending"], character.LevelGained{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid list: %v", err)
	}
	if _, err := (character.MagicInitiateSpellChange{}).Replace(c5, feats, "cleric", "", miSpells(t)["mending"], character.LevelGained{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid old spell: %v", err)
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-031, Table U "granted spells" (SRD 5.2.1, "Magic Initiate", p. 87; "Always-Prepared Spells", p. 104) - a held Magic Initiate exposes the two cantrips, the always-prepared level 1 spell that does not count against any limit, and the chosen ability, per instance and as copies.
// Path: happy.
func TestFeatGrantedSpells(t *testing.T) {
	t.Parallel()
	feats := heldFeats(t, initiateFeat(), clericInitiate(), alertFeat(), character.NoFeatChoices{}, initiateFeat(), wizardInitiate())
	got := feats.GrantedSpells()
	if len(got) != 2 {
		t.Fatalf("granted = %+v", got)
	}
	g := got[0]
	if g.List() != "cleric" || g.Ability() != ability.Wisdom || g.LevelOneSpell() != "bless" || !slices.Equal(g.Cantrips(), spellIDsOf("guidance", "light")) ||
		!g.LevelOneAlwaysPrepared() || g.LevelOneCountsAgainstPreparedLimit() || g.IsZero() {
		t.Errorf("granted = %+v", g)
	}
	g.Cantrips()[0] = ""
	if g.Cantrips()[0] == "" || !g.Equal(got[0]) || g.Equal(got[1]) || got[1].List() != "wizard" {
		t.Error("the cantrips must be copies and Equal must compare the grant")
	}
	if !(character.FeatGrantedSpells{}).IsZero() || len((character.TakenFeats{}).GrantedSpells()) != 0 {
		t.Error("the zero value grants nothing")
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-034, FR-035 (SRD 5.2.1, "Magic Initiate", Spell Change, p. 87) - the instance is found by its spell list wherever it stands among the held feats, the second cantrip can be replaced, a spell the feat does not hold is a missing replacementSpell whatever its name, and a level mismatch is not reported on top of it.
// Path: per case.
func TestMagicInitiateSpellChangeLocatesTheInstanceAndTheSpell(t *testing.T) {
	t.Parallel()
	c := charAt(t, 5)
	ev := gainedBy(t, c)
	feats := heldFeats(t, initiateFeat(), wizardInitiate(), initiateFeat(), clericInitiate())
	ch := replace(t, c, feats, "cleric", "light", "mending", ev).(character.FeatsChanged)
	wizard, cleric := ch.Feats().All()[0].Choices().(character.MagicInitiateChoices), ch.Feats().All()[1].Choices().(character.MagicInitiateChoices)
	if !wizard.Equal(wizardInitiate()) || !slices.Equal(cleric.Cantrips(), spellIDsOf("guidance", "mending")) {
		t.Errorf("wizard %+v, cleric %+v", wizard, cleric)
	}
	one := heldFeats(t, initiateFeat(), clericInitiate())
	missing := []character.FeatChoiceProblem{{Name: character.FeatChoiceReplacementSpell, Kind: character.FeatProblemMissing}}
	for _, tc := range []struct {
		old, repl string
		want      []character.FeatChoiceProblem
	}{
		{"aid", "cure_wounds", missing},
		{"mending", "cure_wounds", missing},
		{"aid", "mending", missing},
	} {
		got := replace(t, c, one, "cleric", tc.old, tc.repl, ev).(character.FeatsRejected).Rejection.(character.FeatChoicesInvalid).Problems()
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s -> %s: %v, want %v", tc.old, tc.repl, got, tc.want)
		}
	}
}

// Requirement: doc/specs/feats-behaviour.md FR-031 - FeatGrantedSpells compares every field: the list, the ability, the level 1 spell and each cantrip, whichever way they differ.
// Path: per case.
func TestFeatGrantedSpellsEqualComparesEveryField(t *testing.T) {
	t.Parallel()
	of := func(c character.MagicInitiateChoices) character.FeatGrantedSpells {
		return heldFeats(t, initiateFeat(), c).GrantedSpells()[0]
	}
	mi := func(list string, ab ability.Ability, cantrips []string, levelOne string) character.FeatGrantedSpells {
		return of(miOf(list, ab, cantrips, levelOne))
	}
	base := mi("druid", ability.Wisdom, []string{"cantrip_a", "cantrip_b"}, "lvl1_b")
	for name, other := range map[string]character.FeatGrantedSpells{
		"a greater list":      mi("wizard", ability.Wisdom, []string{"cantrip_a", "cantrip_b"}, "lvl1_b"),
		"a lesser list":       mi("cleric", ability.Wisdom, []string{"cantrip_a", "cantrip_b"}, "lvl1_b"),
		"a greater ability":   mi("druid", ability.Charisma, []string{"cantrip_a", "cantrip_b"}, "lvl1_b"),
		"a lesser ability":    mi("druid", ability.Intelligence, []string{"cantrip_a", "cantrip_b"}, "lvl1_b"),
		"a greater level one": mi("druid", ability.Wisdom, []string{"cantrip_a", "cantrip_b"}, "lvl1_c"),
		"a lesser level one":  mi("druid", ability.Wisdom, []string{"cantrip_a", "cantrip_b"}, "lvl1_a"),
		"another cantrip":     mi("druid", ability.Wisdom, []string{"cantrip_a", "cantrip_c"}, "lvl1_b"),
	} {
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("%s: grants compared equal", name)
		}
	}
	if !base.Equal(mi("druid", ability.Wisdom, []string{"cantrip_a", "cantrip_b"}, "lvl1_b")) {
		t.Error("the same grants must compare equal")
	}
}
