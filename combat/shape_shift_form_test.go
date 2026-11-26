package combat_test

import (
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/shape-changing-spells.md FR-006 - NewFormFacts maps the id, type, size options, Challenge Rating and printed average Hit Points of a monster (a Beast, a multi-size block, Challenge Rating 0).
// Path: happy.
func TestNewFormFactsMapsTheStatBlock(t *testing.T) {
	t.Parallel()
	f := ssFacts(t, content.CreatureTypeBeast, "0", 4, content.SizeMedium, content.SizeSmall)
	if f.ID() != "fixture_beast" || f.Type() != content.CreatureTypeBeast || f.HitPoints() != 4 || f.IsZero() ||
		f.ChallengeRating() != content.MustChallengeRating("0") {
		t.Errorf("facts = %+v", f)
	}
	if got := f.Sizes(); !slices.Equal(got, []content.CreatureSize{content.SizeSmall, content.SizeMedium}) {
		t.Errorf("Sizes() = %v, want Small, Medium in ascending order", got)
	}
	got := f.Sizes()
	got[0] = content.SizeHuge
	if f.Sizes()[0] != content.SizeSmall {
		t.Error("Sizes() returned a view of the facts")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-006 - NewFormFacts rejects the zero Monster and an average below 1 with ErrInvalidInput (content never builds an entry below 1, so the zero Monster is the one input that shows it).
// Path: unhappy (invalid input).
func TestNewFormFactsRejectsAMonsterWithNoStatBlock(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewFormFacts(content.Monster{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewFormFacts(zero Monster) error = %v, want ErrInvalidInput", err)
	}
	if !(combat.FormFacts{}).IsZero() {
		t.Error("the zero FormFacts is not zero")
	}
	if _, err := content.NewHitPointsEntry(0, dice.MustExpression(1, dice.D8(), 0)); err == nil {
		t.Error("content accepted an average below 1, so NewFormFacts has a case the content type forbids")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-007 - NewCreatureForm returns a *SizeError (ErrInvalidInput) for a size not among the printed options, accepts each printed option, and the zero AssumedForm is invalid.
// Path: per case.
func TestNewCreatureFormJudgesTheSize(t *testing.T) {
	t.Parallel()
	facts := ssFacts(t, content.CreatureTypeBeast, "1", 10, content.SizeLarge, content.SizeHuge)
	cases := []struct {
		name, requirement, path string
		size                    content.CreatureSize
		ok                      bool
	}{
		{"first printed size", "FR-007", "happy", content.SizeLarge, true},
		{"second printed size", "FR-007", "happy", content.SizeHuge, true},
		{"smaller than printed", "FR-007", "unhappy (boundary)", content.SizeMedium, false},
		{"larger than printed", "FR-007", "unhappy (boundary)", content.SizeGargantuan, false},
		{"the zero size", "FR-007", "unhappy (invalid input)", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			form, err := combat.NewCreatureForm(facts, c.size)
			if c.ok {
				if err != nil || form.Size() != c.size || form.Facts() != facts {
					t.Fatalf("NewCreatureForm(%v) = %+v, %v", c.size, form, err)
				}
				return
			}
			var se *combat.SizeError
			if !errors.As(err, &se) || !errors.Is(err, srderr.ErrInvalidInput) || se.Size() != c.size ||
				!slices.Equal(se.Allowed(), []content.CreatureSize{content.SizeLarge, content.SizeHuge}) {
				t.Errorf("NewCreatureForm(%v) error = %v [%s]", c.size, err, c.requirement)
			}
		})
	}
	if _, err := combat.NewCreatureForm(combat.FormFacts{}, content.SizeLarge); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewCreatureForm(zero facts) error = %v", err)
	}
	if _, err := combat.NewObjectForm(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewObjectForm(0) error = %v", err)
	}
	se := &combat.SizeError{}
	if got := se.Allowed(); got != nil {
		t.Errorf("the zero SizeError allows %v", got)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-007 - FormFacts and AssumedForm are immutable and equal by value.
// Path: unhappy (immutability).
func TestFormsAreEqualByValue(t *testing.T) {
	t.Parallel()
	a, b := ssBeast(t, "1", 11), ssBeast(t, "1", 11)
	var x, y combat.AssumedForm = a, b
	if x != y || a.Facts() != b.Facts() {
		t.Error("two forms built from equal facts differ")
	}
	var o1, o2 combat.AssumedForm = ssObject(t, content.SizeLarge), ssObject(t, content.SizeLarge)
	if o1 != o2 || o1 == x {
		t.Error("object forms compare wrongly")
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-008 - NewRatedByLevel returns a typed range error (ErrInvalidInput, Min 1, Max 20) for n below 1 or above 20 and accepts 1 and 20; a level n compares as the Challenge Rating n.
// Path: per case.
func TestNewRatedByLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		n                       int
		ok                      bool
	}{
		{"level 0", "FR-008", "unhappy (boundary)", 0, false},
		{"level 1", "FR-008", "unhappy (boundary)", 1, true},
		{"level 20", "FR-008", "unhappy (boundary)", 20, true},
		{"level 21", "FR-008", "unhappy (boundary)", 21, false},
		{"negative level", "FR-008", "unhappy (invalid input)", -3, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r, err := combat.NewRatedByLevel(c.n)
			if c.ok {
				if err != nil || r.Level() != c.n || r.Challenge() != content.MustChallengeRating(strconv.Itoa(c.n)) {
					t.Errorf("NewRatedByLevel(%d) = %+v, %v", c.n, r, err)
				}
				return
			}
			var re *combat.LevelRangeError
			if !errors.As(err, &re) || !errors.Is(err, srderr.ErrInvalidInput) || re.Level != c.n || re.Min != 1 || re.Max != 20 {
				t.Errorf("NewRatedByLevel(%d) error = %v [%s]", c.n, err, c.requirement)
			}
		})
	}
	if got := (combat.RatedByLevel{}).Challenge(); got != content.MustChallengeRating("0") {
		t.Errorf("the zero RatedByLevel compares as %v", got)
	}
}

// Requirement: doc/specs/shape-changing-spells.md FR-007, FR-008 - the typed errors render some text, never branched on (errors-and-results.md), and a retained statistic has a snake_case name.
// Path: unhappy (invalid input).
func TestShapeShiftErrorsAndStatisticsRender(t *testing.T) {
	t.Parallel()
	_, sizeErr := combat.NewObjectForm(0)
	_, levelErr := combat.NewRatedByLevel(0)
	_, transformationErr := combat.NewActiveTransformation(combat.ActiveTransformationFields{})
	for name, err := range map[string]error{"size": sizeErr, "level": levelErr, "transformation": transformationErr} {
		if err == nil || err.Error() == "" {
			t.Errorf("%s error = %v, want some text", name, err)
		}
	}
	for s := combat.RetainedAlignment; s <= combat.RetainedSpellcastingFeature; s++ {
		if s.String() == "unknown" {
			t.Errorf("statistic %d has no name", s)
		}
	}
	if combat.RetainedStatistic(0).String() != "unknown" {
		t.Error("the zero statistic has a name")
	}
}
