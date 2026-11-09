package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-warlock.md section 4.1 and shared-types-and-ownership.md - a Warlock level is 1 to 20; 0, 21 and a negative number are refused with ErrInvalidInput; the zero value is invalid; MustWarlockLevel panics with that error.
// Path: unhappy (boundary).
func TestWarlockLevelBounds(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, -1, 21} {
		if _, err := character.NewWarlockLevel(n); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewWarlockLevel(%d) = %v, want ErrInvalidInput", n, err)
		}
	}
	for _, n := range []int{1, 20} {
		l, err := character.NewWarlockLevel(n)
		if err != nil || l.Value() != n || l.IsZero() {
			t.Errorf("NewWarlockLevel(%d) = %v, %v", n, l, err)
		}
	}
	if !(character.WarlockLevel{}).IsZero() || (character.WarlockLevel{}).Value() != 0 {
		t.Error("the zero WarlockLevel must be zero")
	}
	defer func() {
		if r := recover(); r == nil || !errors.Is(r.(error), srderr.ErrInvalidInput) {
			t.Errorf("MustWarlockLevel(0) panicked with %v, want an ErrInvalidInput error", r)
		}
	}()
	character.MustWarlockLevel(0)
}

// Requirement: doc/specs/class-warlock.md FR-117, table 4.7.2 (SRD 5.2.1, "Level 11: Mystic Arcanum", p. 72) - the arcanum spells are 0 below level 11, 1 at 11-12, 2 at 13-14, 3 at 15-16 and 4 from 17.
// Path: per case (the boundaries of the table).
func TestWarlockLevelArcanumCount(t *testing.T) {
	t.Parallel()
	for level, want := range map[int]int{1: 0, 10: 0, 11: 1, 12: 1, 13: 2, 14: 2, 15: 3, 16: 3, 17: 4, 20: 4} {
		if got := character.MustWarlockLevel(level).ArcanumCount(); got != want {
			t.Errorf("ArcanumCount at level %d = %d, want %d", level, got, want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-107, FR-109, table 4.7.1 (SRD 5.2.1, "Level 2: Magical Cunning" and "Level 20: Eldritch Master", p. 72) - Magical Cunning regains the lesser of the slots expended and half the maximum rounded up, and every expended slot at level 20.
// Path: per case (MC-01 to MC-10).
func TestMagicalCunningRegain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                     string
		level, maximum, expended int
		want                     int
	}{
		{"MC-01 all expended of 2", 2, 2, 2, 1}, {"MC-02 one of 2", 2, 2, 1, 1}, {"MC-03 none expended", 2, 2, 0, 0}, {"MC-04 level 10", 10, 2, 2, 1},
		{"MC-05 half of 3 rounded up", 11, 3, 3, 2}, {"MC-06 fewer expended than half", 11, 3, 1, 1}, {"MC-07 half of 4", 17, 4, 4, 2},
		{"MC-08 level 19 is capped", 19, 4, 4, 2}, {"MC-09 level 20 regains all", 20, 4, 4, 4}, {"MC-10 level 20 regains the expended", 20, 4, 3, 3},
		{"negative expended is read as 0", 5, 2, -1, 0}, {"expended above the maximum is read as the maximum", 20, 2, 9, 2}, {"negative maximum", 2, -3, 1, 0},
	}
	for _, c := range cases {
		if got := character.MagicalCunningRegain(character.MustWarlockLevel(c.level), c.maximum, c.expended); got != c.want {
			t.Errorf("%s: MagicalCunningRegain(level %d, max %d, expended %d) = %d, want %d", c.name, c.level, c.maximum, c.expended, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-056, table 4.6.2 (SRD 5.2.1, "Eldritch Spear", p. 73) - the range grows by 30 feet times the Warlock level.
// Path: per case (ES-01 to ES-05).
func TestEldritchSpearRangeIncrease(t *testing.T) {
	t.Parallel()
	for level, want := range map[int]int{2: 60, 5: 150, 11: 330, 17: 510, 20: 600, 1: 30} {
		if got := character.EldritchSpearRangeIncrease(character.MustWarlockLevel(level)); got != want {
			t.Errorf("EldritchSpearRangeIncrease(%d) = %d, want %d", level, got, want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-073, table 4.6.4 (SRD 5.2.1, "Eldritch Smite", pp. 72-73) - the extra Force damage is 1d8 plus 1d8 per slot level, (slot level + 1)d8; a slot level outside 1 to 5 has none.
// Path: per case (ESM-01 to ESM-05; boundary 0 and 6).
func TestEldritchSmiteDamage(t *testing.T) {
	t.Parallel()
	for level := 1; level <= 5; level++ {
		got, ok := character.EldritchSmiteDamage(level)
		if !ok || got != dice.MustExpression(level+1, dice.D8(), 0) {
			t.Errorf("EldritchSmiteDamage(%d) = %v, %v; want %dd8", level, got, ok, level+1)
		}
	}
	for _, level := range []int{0, 6, -1} {
		if _, ok := character.EldritchSmiteDamage(level); ok {
			t.Errorf("EldritchSmiteDamage(%d) must have no damage", level)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-123, FR-128, FR-102, FR-078, tables 4.8.1, 4.8.3, 4.6.7 and 4.6.5 (SRD 5.2.1, pp. 73 and 76) - Dark One's Blessing is the Charisma modifier plus the level, at least 1; Dark One's Own Luck and the Gift of the Protectors page are the Charisma modifier, at least 1; Lifedrinker heals the roll plus the Constitution modifier, at least 1.
// Path: per case (DB-01 to DB-07, DL-01 to DL-05, GP-01 to GP-05, LD-01 to LD-05).
func TestWarlockCharismaFormulas(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ level, mod, want int }{{3, 3, 6}, {3, 0, 3}, {3, -1, 2}, {3, -2, 1}, {3, -3, 1}, {3, -5, 1}, {20, 5, 25}} {
		if got := character.DarkOnesBlessingTemporaryHitPoints(character.MustWarlockLevel(c.level), warModifier(c.mod)); got != c.want {
			t.Errorf("DarkOnesBlessingTemporaryHitPoints(level %d, %d) = %d, want %d", c.level, c.mod, got, c.want)
		}
	}
	for _, c := range []struct{ mod, want int }{{-1, 1}, {0, 1}, {1, 1}, {2, 2}, {3, 3}, {5, 5}} {
		if got := character.DarkOnesOwnLuckUses(warModifier(c.mod)); got != c.want {
			t.Errorf("DarkOnesOwnLuckUses(%d) = %d, want %d", c.mod, got, c.want)
		}
		if got := character.GiftOfTheProtectorsCapacity(warModifier(c.mod)); got != c.want {
			t.Errorf("GiftOfTheProtectorsCapacity(%d) = %d, want %d", c.mod, got, c.want)
		}
	}
	for _, c := range []struct{ roll, mod, want int }{{1, -2, 1}, {1, -1, 1}, {1, 0, 1}, {4, 2, 6}, {8, 3, 11}} {
		if got := character.LifedrinkerHealing(c.roll, warModifier(c.mod)); got != c.want {
			t.Errorf("LifedrinkerHealing(%d, %d) = %d, want %d", c.roll, c.mod, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-079, FR-155, table 4.6.6 (SRD 5.2.1, "Thirsting Blade", p. 74; "Devouring Blade", p. 72; "Extra Attack" under "Multiclassing", p. 25) - Thirsting Blade raises the pact weapon count to 2 and with Devouring Blade to 3, never lowering a higher baseline; Devouring Blade alone gives nothing.
// Path: per case (TB-01 to TB-08).
func TestPactWeaponAttackCount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                 string
		baseline             int
		thirsting, devouring bool
		want                 int
	}{
		{"TB-01 Pact of the Blade only", 1, false, false, 1}, {"TB-02 Thirsting Blade", 1, true, false, 2}, {"TB-03 with Devouring Blade", 1, true, true, 3},
		{"TB-04 Devouring Blade without Thirsting Blade", 1, false, true, 1}, {"TB-05 baseline 2", 2, true, false, 2}, {"TB-06 baseline 3", 3, true, false, 3},
		{"TB-07 Devouring Blade over baseline 2", 2, true, true, 3}, {"TB-08 baseline 4 is never lowered", 4, true, true, 4}, {"a baseline below 1 is read as 1", 0, false, false, 1},
	}
	for _, c := range cases {
		if got := character.PactWeaponAttackCount(c.baseline, c.thirsting, c.devouring); got != c.want {
			t.Errorf("%s: PactWeaponAttackCount(%d, %v, %v) = %d, want %d", c.name, c.baseline, c.thirsting, c.devouring, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-143 (profile) - a profile needs a level and a valid subclass id; the zero profile is refused; Level and Subclass read back.
// Path: unhappy (invalid input).
func TestWarlockProfile(t *testing.T) {
	t.Parallel()
	if _, err := character.NewWarlockProfile(character.WarlockLevel{}, ""); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero level: %v", err)
	}
	if _, err := character.NewWarlockProfile(character.MustWarlockLevel(3), "Not Valid"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("invalid subclass: %v", err)
	}
	p := warFiend(7)
	if p.Level().Value() != 7 || p.Subclass() != character.FiendPatronSubclassID || character.WarlockClassID != "warlock" {
		t.Errorf("profile = %v %v", p.Level(), p.Subclass())
	}
}

// Requirement: doc/specs/class-warlock.md FR-143 - WarlockProfileOf builds the profile from the class levels and the chosen subclass; a character with no Warlock level has none.
// Path: per case (happy, no subclass, not a Warlock).
func TestWarlockProfileOf(t *testing.T) {
	t.Parallel()
	c := warChar(t, 6, 16, "fiend_patron", character.WarlockState{})
	p, ok, err := character.WarlockProfileOf(c)
	if err != nil || !ok || p.Level().Value() != 6 || p.Subclass() != "fiend_patron" {
		t.Errorf("WarlockProfileOf = %v, %v, %v", p, ok, err)
	}
	p, ok, err = character.WarlockProfileOf(warChar(t, 2, 16, "", character.WarlockState{}))
	if err != nil || !ok || p.Subclass() != "" {
		t.Errorf("no subclass: %v, %v, %v", p, ok, err)
	}
	if _, ok, err := character.WarlockProfileOf(restored(t, 355000, 200, flatScores(t, 10), cl("fighter", 4))); ok || err != nil {
		t.Errorf("a Fighter = %v, %v", ok, err)
	}
}
