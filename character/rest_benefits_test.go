package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func spentOf(t *testing.T, counts map[content.HitPointDie]int) character.SpentHitDice {
	t.Helper()
	s, err := character.NewSpentHitDice(counts)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var (
	allHP   = character.AllLostHitPointsRegained{}
	hpMax   = character.HitPointMaximumReturnedToNormal{}
	scores  = character.AbilityScoresReturnedToNormal{}
	tempEnd = character.TemporaryHitPointsEndedByRest{}
)

func benefitsOf(t *testing.T, kind character.RestKind, level int, spent character.SpentHitDice) character.RestBenefits {
	t.Helper()
	b, err := character.NewRestBenefits(kind, level, spent)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// longRecharge is the FeaturesRecharged effect of a Long Rest, as built by the package.
func longRecharge(t *testing.T) character.RestEffect {
	t.Helper()
	effs := benefitsOf(t, character.LongRest, 0, character.SpentHitDice{}).Effects()
	return effs[len(effs)-1]
}

// Requirement: per case.
// Path: per case.
func TestNewRestBenefitsLongRest(t *testing.T) {
	t.Parallel()
	d8x3 := map[content.HitPointDie]int{content.HitDieD8: 3}
	mixed := map[content.HitPointDie]int{content.HitDieD8: 1, content.HitDieD10: 2}
	cases := []struct {
		name, requirement, path string
		level                   int
		spent                   map[content.HitPointDie]int
		want                    []character.RestEffect
	}{
		{"LB-1 healthy, nothing spent", "FR-025 FR-027 FR-028 FR-030 FR-031 (SRD 5.2.1, \"Long Rest\", p. 185)", "happy", 0, nil,
			[]character.RestEffect{allHP, hpMax, scores, tempEnd, nil}},
		{"LB-2 three dice spent", "FR-026", "happy", 0, d8x3,
			[]character.RestEffect{allHP, hitDiceRegained(t, 3), hpMax, scores, tempEnd, nil}},
		{"LB-3 exhaustion 1 ends the condition", "FR-029", "unhappy (boundary)", 1, nil,
			[]character.RestEffect{allHP, hpMax, scores, exhaustion(t, 1), tempEnd, nil}},
		{"LB-4 exhaustion 2 and mixed dice", "FR-029 FR-034", "unhappy (rule interaction)", 2, mixed,
			[]character.RestEffect{allHP, hitDiceRegained(t, 3), hpMax, scores, exhaustion(t, 2), tempEnd, nil}},
		{"LB-5 exhaustion 5", "FR-029", "unhappy (boundary)", 5, nil,
			[]character.RestEffect{allHP, hpMax, scores, exhaustion(t, 5), tempEnd, nil}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := benefitsOf(t, character.LongRest, c.level, spentOf(t, c.spent))
			want := append([]character.RestEffect(nil), c.want...)
			want[len(want)-1] = longRecharge(t)
			if got := b.Effects(); !reflect.DeepEqual(got, want) {
				t.Errorf("effects = %#v, want %#v [%s]", got, want, c.requirement)
			}
			if b.SpentHitDice().Total() != 0 {
				t.Errorf("spent dice after a Long Rest = %v, want none [FR-026]", b.SpentHitDice().Entries())
			}
		})
	}
}

func hitDiceRegained(t *testing.T, n int) character.RestEffect {
	t.Helper()
	for _, e := range benefitsOf(t, character.LongRest, 0, spentOf(t, map[content.HitPointDie]int{content.HitDieD6: n})).Effects() {
		if h, ok := e.(character.HitPointDiceRegained); ok {
			return h
		}
	}
	t.Fatal("no HitPointDiceRegained")
	return nil
}

func exhaustion(t *testing.T, from int) character.RestEffect {
	t.Helper()
	for _, e := range benefitsOf(t, character.LongRest, from, character.SpentHitDice{}).Effects() {
		if x, ok := e.(character.ExhaustionLevelReduced); ok {
			if x.From() != from || x.To() != from-1 {
				t.Fatalf("ExhaustionLevelReduced = %d to %d", x.From(), x.To())
			}
			return x
		}
	}
	t.Fatal("no ExhaustionLevelReduced")
	return nil
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-024, FR-033, table 4.8e SB-1 and
// SB-2 - a Short Rest gives exactly two effects, nothing about Hit Points, the
// maximum, ability scores, Exhaustion or Temporary Hit Points, and leaves the spent
// dice unchanged (SRD 5.2.1, "Short Rest", p. 187).
// Path: unhappy (rule interaction).
func TestNewRestBenefitsShortRest(t *testing.T) {
	t.Parallel()
	spent := spentOf(t, map[content.HitPointDie]int{content.HitDieD8: 2})
	b := benefitsOf(t, character.ShortRest, 3, spent)
	effs := b.Effects()
	if len(effs) != 2 {
		t.Fatalf("effects = %#v", effs)
	}
	if _, ok := effs[0].(character.SpendHitPointDiceAvailable); !ok {
		t.Errorf("first effect = %#v", effs[0])
	}
	if f, ok := effs[1].(character.FeaturesRecharged); !ok || f.Kind() != character.ShortRest {
		t.Errorf("second effect = %#v", effs[1])
	}
	if !b.SpentHitDice().Equal(spent) {
		t.Errorf("spent dice = %v, want unchanged", b.SpentHitDice().Entries())
	}
	if f := longRecharge(t).(character.FeaturesRecharged); f.Kind() != character.LongRest {
		t.Errorf("long rest recharge kind = %v", f.Kind())
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-013 (invalid input), table 4.8e
// LB-6 - an invalid kind or an Exhaustion level outside 0 to 6 is an input error.
// Path: per case.
func TestNewRestBenefitsRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		kind  character.RestKind
		level int
	}{"invalid kind": {0, 0}, "negative level": {character.LongRest, -1}, "level 7": {character.ShortRest, 7}} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := character.NewRestBenefits(c.kind, c.level, character.SpentHitDice{}); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
	if b := benefitsOf(t, character.LongRest, 6, character.SpentHitDice{}); len(b.Effects()) != 6 {
		t.Errorf("level 6 benefits = %#v", b.Effects())
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-064, FR-048 - the benefits of a
// Short Rest are callable without a report, and the returned effects are a copy.
// Path: unhappy (immutability).
func TestRestBenefitsEffectsAreACopy(t *testing.T) {
	t.Parallel()
	b := benefitsOf(t, character.ShortRest, 0, character.SpentHitDice{})
	b.Effects()[0] = allHP
	if _, ok := b.Effects()[0].(character.SpendHitPointDiceAvailable); !ok {
		t.Error("mutating the returned slice changed the benefits")
	}
}
