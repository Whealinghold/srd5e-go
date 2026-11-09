package character_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/class-rogue.md FR-093 (SRD 5.2.1, "Level 3: Fast Hands", p. 64) - Fast Hands gives three Bonus Action
// options from Rogue level 3 and none below.
// Path: per case (levels 2, 3, 20): unhappy (boundary).
func TestThiefFastHands(t *testing.T) {
	t.Parallel()
	want := []character.FastHandsOption{character.FastHandsSleightOfHandCheck, character.FastHandsUtilizeAction, character.FastHandsMagicActionForMagicItem}
	for _, c := range []struct {
		level int
		want  []character.FastHandsOption
	}{{2, []character.FastHandsOption{}}, {3, want}, {20, want}} {
		if got := (character.ThiefFeatures{}).FastHands(rl(c.level)); !slices.Equal(got, c.want) {
			t.Errorf("level %d: %v, want %v", c.level, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-094 (SRD 5.2.1, "Level 3: Second-Story Work", p. 64) - from Rogue level 3 the Climb Speed
// equals the Speed and jumps use Dexterity; below level 3 the feature is not gained; a negative Speed is refused.
// Path: per case (levels 2 and 3; Speed 0, 30): happy / unhappy (boundary).
func TestThiefSecondStoryWork(t *testing.T) {
	t.Parallel()
	got, err := character.ThiefFeatures{}.SecondStoryWork(rl(3), 30)
	if g, ok := got.(character.SecondStoryWorkGranted); err != nil || !ok || g.ClimbSpeedFeet() != 30 || g.JumpAbility() != ability.Dexterity {
		t.Errorf("level 3 = %#v, %v", got, err)
	}
	if got, err = (character.ThiefFeatures{}).SecondStoryWork(rl(20), 0); err != nil || got.(character.SecondStoryWorkGranted).ClimbSpeedFeet() != 0 {
		t.Errorf("Speed 0 = %#v, %v", got, err)
	}
	if got, err = (character.ThiefFeatures{}).SecondStoryWork(rl(2), 30); err != nil || got != (character.SecondStoryWorkNotGained{}) {
		t.Errorf("level 2 = %#v, %v", got, err)
	}
	_, err = character.ThiefFeatures{}.SecondStoryWork(rl(3), -1)
	wantInvalid(t, err, "negative Speed")
}

// Requirement: doc/specs/class-rogue.md FR-095, FR-096, FR-113 (SRD 5.2.1, "Level 13: Use Magic Device", p. 64) - the attunement limit is
// 4 from Rogue level 13 and unchanged before; no charges are expended on a d6 roll of 6 and charges are expended on 1 to 5; the scroll
// route is Intelligence, reliable up to level 1, Arcana, for a Thief of level 13 or higher and none otherwise.
// Path: per case (the Use Magic Device cases): happy / unhappy (boundary, rule interaction).
func TestUseMagicDevice(t *testing.T) {
	t.Parallel()
	umd := character.UseMagicDevice{}
	if got, ok := umd.AttunementLimit(rl(13)).(character.AttunementLimitSet); !ok || got.Limit() != 4 {
		t.Errorf("level 13 limit = %#v", umd.AttunementLimit(rl(13)))
	}
	if _, ok := umd.AttunementLimit(rl(12)).(character.AttunementLimitUnchanged); !ok {
		t.Errorf("level 12 limit = %#v", umd.AttunementLimit(rl(12)))
	}
	for roll, want := range map[int]bool{1: true, 5: true, 6: false} {
		t.Run("d6 "+strconv.Itoa(roll), func(t *testing.T) {
			t.Parallel()
			if got, err := umd.ChargesExpended(roll); err != nil || got != want {
				t.Errorf("ChargesExpended(%d) = %v, %v; want %v", roll, got, err, want)
			}
		})
	}
	for _, roll := range []int{0, 7, -1} {
		_, err := umd.ChargesExpended(roll)
		wantInvalid(t, err, "roll "+strconv.Itoa(roll))
	}
	route := umd.ScrollRoute(rl(13), character.ThiefSubclassID)
	if route.Ability() != ability.Intelligence || route.ReliableUpToLevel() != 1 || route.CheckSkill() != content.SkillArcana || route.IsZero() {
		t.Errorf("Thief 13 route = %#v", route)
	}
	for name, got := range map[string]character.AnyScrollRoute{
		"Thief 12": umd.ScrollRoute(rl(12), character.ThiefSubclassID), "another subclass 20": umd.ScrollRoute(rl(20), "other"), "no subclass 20": umd.ScrollRoute(rl(20), ""),
	} {
		if !got.IsZero() {
			t.Errorf("%s has a route: %#v", name, got)
		}
	}
}

// Requirement: doc/specs/class-rogue.md FR-100 (SRD 5.2.1, "Level 17: Thief's Reflexes", p. 64) - a Thief of Rogue level 17 or higher
// has two Initiative counts, the Initiative and the Initiative minus 10; everyone else has one.
// Path: per case (Initiative 15 and 5; levels 16, 17; non-Thief): happy / unhappy (boundary).
func TestThiefReflexesInitiativeTurns(t *testing.T) {
	t.Parallel()
	thief := character.ThiefSubclassID
	for _, c := range []struct {
		name       string
		level      int
		subclass   content.SubclassID
		initiative int
		want       []int
	}{
		{"Thief 17, 15", 17, thief, 15, []int{15, 5}}, {"Thief 17, 5", 17, thief, 5, []int{5, -5}}, {"Thief 20", 20, thief, 12, []int{12, 2}},
		{"Thief 16", 16, thief, 15, []int{15}}, {"another subclass 17", 17, "other", 15, []int{15}}, {"no subclass 20", 20, "", 15, []int{15}},
	} {
		if got := (character.ThiefReflexes{}).InitiativeTurns(rl(c.level), c.subclass, c.initiative); !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
