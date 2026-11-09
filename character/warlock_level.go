package character

import (
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	// WarlockClassID is the id of the Warlock class.
	WarlockClassID content.ClassID = "warlock"
	// FiendPatronSubclassID is the id of the Fiend Patron subclass, the only Warlock subclass in SRD 5.2.1.
	FiendPatronSubclassID content.SubclassID = "fiend_patron"
)

// The Warlock levels at which features arrive (SRD 5.2.1, "Warlock Features", p. 71, and "Fiend Patron", p. 76).
const (
	warlockMagicalCunningLevel  = 2
	warlockContactPatronLevel   = 9
	warlockMysticArcanumLevel   = 11
	warlockEldritchMasterLevel  = 20
	fiendBlessingLevel          = 3
	fiendOwnLuckLevel           = 6
	fiendResilienceLevel        = 10
	fiendHurlLevel              = 14
	warlockArcanumFirstSpellLvl = 6
	warlockMaxPactSlotLevel     = 5
)

// WarlockLevel is a level in the Warlock class, 1 to 20 (SRD 5.2.1, "Warlock Features", p. 71). It counts Warlock levels only, not
// the total level. It is immutable and comparable with ==. The zero WarlockLevel is invalid; see IsZero. Safe for concurrent use.
type WarlockLevel struct{ n int }

// NewWarlockLevel returns the Warlock level n. It returns an error matching srderr.ErrInvalidInput for n below 1 or above 20. It
// never panics.
func NewWarlockLevel(n int) (WarlockLevel, error) {
	if n < 1 || n > MaxLevel {
		return WarlockLevel{}, invalidInput("Warlock level " + strconv.Itoa(n) + " outside 1 to 20")
	}
	return WarlockLevel{n: n}, nil
}

// MustWarlockLevel is NewWarlockLevel for literals known to be valid. It panics with the error NewWarlockLevel returns, and only then.
func MustWarlockLevel(n int) WarlockLevel {
	l, err := NewWarlockLevel(n)
	if err != nil {
		panic(err)
	}
	return l
}

// Value returns the level, 1 to 20; 0 for the zero value.
func (l WarlockLevel) Value() int { return l.n }

// IsZero reports whether l is the invalid zero value.
func (l WarlockLevel) IsZero() bool { return l.n == 0 }

// ArcanumCount returns the number of Mystic Arcanum spells at the level: 0 below 11, 1 at 11 and 12, 2 at 13 and 14, 3 at 15 and 16
// and 4 from 17 (SRD 5.2.1, "Level 11: Mystic Arcanum", p. 72; class-warlock.md FR-117). The arcanum spells are of levels 6, 7, 8 and 9
// in that order.
func (l WarlockLevel) ArcanumCount() int {
	if l.n < warlockMysticArcanumLevel {
		return 0
	}
	return min((l.n-warlockMysticArcanumLevel)/2+1, 4)
}

// WarlockProfile is the Warlock level and the chosen subclass a function needs, built by the caller from ClassLevels and the
// character's ChosenSubclass and never stored (class-warlock.md FR-143). The zero WarlockProfile is invalid. It is immutable and
// comparable with ==.
type WarlockProfile struct {
	level    WarlockLevel
	subclass content.SubclassID
}

// NewWarlockProfile returns the profile of a Warlock of level with subclass (empty for none). It returns an error matching
// srderr.ErrInvalidInput for the zero level or a non-empty invalid subclass id. It never panics.
func NewWarlockProfile(level WarlockLevel, subclass content.SubclassID) (WarlockProfile, error) {
	if level.IsZero() || (subclass != "" && !subclass.Valid()) {
		return WarlockProfile{}, invalidInput("Warlock profile with a zero level or an invalid subclass id")
	}
	return WarlockProfile{level: level, subclass: subclass}, nil
}

// WarlockProfileOf returns the profile of c, built from its ClassLevels and the ChosenSubclass held in the Warlock subclass state,
// and false for a character with no Warlock level. It returns an error matching srderr.ErrInvalidInput when another type of state
// sits under the subclass key.
func WarlockProfileOf(c Character) (WarlockProfile, bool, error) {
	n := c.ClassLevels().LevelIn(WarlockClassID)
	if n < 1 {
		return WarlockProfile{}, false, nil
	}
	key, err := SubclassStateKey{}.ForClass(WarlockClassID)
	if err != nil { // unreachable: the class id is valid
		return WarlockProfile{}, false, err
	}
	chosen, held, err := ReadFeatureState(c.FeatureStates(), key)
	if err != nil {
		return WarlockProfile{}, false, err
	}
	p := WarlockProfile{level: WarlockLevel{n: n}}
	if held {
		p.subclass = chosen.Subclass()
	}
	return p, true, nil
}

// Level returns the Warlock level.
func (p WarlockProfile) Level() WarlockLevel { return p.level }

// Subclass returns the chosen subclass, empty for none.
func (p WarlockProfile) Subclass() content.SubclassID { return p.subclass }

// MagicalCunningRegain returns the Pact Magic slots Magical Cunning regains: the lesser of the slots expended and half the maximum
// rounded up, or all the expended slots at Warlock level 20 (SRD 5.2.1, "Level 2: Magical Cunning" and "Level 20: Eldritch Master",
// p. 72; class-warlock.md FR-107, FR-109, table 4.7.1). A negative count is read as 0.
func MagicalCunningRegain(level WarlockLevel, maximum, expended int) int {
	expended = min(max(expended, 0), max(maximum, 0))
	if level.n >= warlockEldritchMasterLevel {
		return expended
	}
	return min(expended, (max(maximum, 0)+1)/2)
}

// EldritchSpearRangeIncrease returns the feet Eldritch Spear adds to the range of the chosen cantrip, 30 times the Warlock level
// (SRD 5.2.1, "Eldritch Spear", p. 73; class-warlock.md FR-056, table 4.6.2).
func EldritchSpearRangeIncrease(level WarlockLevel) int { return 30 * level.n }

// EldritchSmiteDamage returns the extra Force damage of Eldritch Smite for a Pact Magic slot of slotLevel, 1d8 plus 1d8 per slot
// level (SRD 5.2.1, "Eldritch Smite", pp. 72-73; class-warlock.md FR-073, table 4.6.4), and false for a level outside 1 to 5.
func EldritchSmiteDamage(slotLevel int) (dice.Expression, bool) {
	if slotLevel < 1 || slotLevel > warlockMaxPactSlotLevel {
		return dice.Expression{}, false
	}
	return dice.MustExpression(slotLevel+1, dice.D8(), 0), true
}

// DarkOnesBlessingTemporaryHitPoints returns the Temporary Hit Points of Dark One's Blessing: the Charisma modifier plus the Warlock
// level, at least 1 (SRD 5.2.1, "Level 3: Dark One's Blessing", p. 76; class-warlock.md FR-123, table 4.8.1).
func DarkOnesBlessingTemporaryHitPoints(level WarlockLevel, charisma ability.Modifier) int {
	return max(1, charisma.Value()+level.n)
}

// DarkOnesOwnLuckUses returns the uses of Dark One's Own Luck between Long Rests: the Charisma modifier, at least 1 (SRD 5.2.1,
// "Level 6: Dark One's Own Luck", p. 76; class-warlock.md FR-128, table 4.8.3).
func DarkOnesOwnLuckUses(charisma ability.Modifier) int { return max(1, charisma.Value()) }

// GiftOfTheProtectorsCapacity returns how many names the page of Gift of the Protectors holds: the Charisma modifier, at least 1
// (SRD 5.2.1, "Gift of the Protectors", p. 73; class-warlock.md FR-102, table 4.6.7).
func GiftOfTheProtectorsCapacity(charisma ability.Modifier) int { return max(1, charisma.Value()) }

// LifedrinkerHealing returns the Hit Points Lifedrinker regains for a Hit Point Die roll: the roll plus the Constitution modifier, at
// least 1 (SRD 5.2.1, "Lifedrinker", p. 73; class-warlock.md FR-078, table 4.6.5).
func LifedrinkerHealing(roll int, constitution ability.Modifier) int {
	return max(1, roll+constitution.Value())
}

// PactWeaponAttackCount returns the attacks made with the pact weapon on the Attack action: the baseline the character's other
// features give, raised to 2 by Thirsting Blade and to 3 by Thirsting Blade with Devouring Blade, never lowered (SRD 5.2.1,
// "Thirsting Blade", p. 74, and "Devouring Blade", p. 72; class-warlock.md FR-079, FR-155, table 4.6.6). Devouring Blade without
// Thirsting Blade gives nothing.
func PactWeaponAttackCount(baseline int, thirstingBlade, devouringBlade bool) int {
	baseline = max(baseline, 1)
	switch {
	case thirstingBlade && devouringBlade:
		return max(baseline, 3)
	case thirstingBlade:
		return max(baseline, 2)
	}
	return baseline
}
