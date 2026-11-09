package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type dieCounts = map[content.HitPointDie]int

func poolOfDice(t *testing.T, counts dieCounts) character.HitDicePool {
	t.Helper()
	p, err := character.NewHitDicePool(counts)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

const (
	d6, d8, d10, d12 = content.HitDieD6, content.HitDieD8, content.HitDieD10, content.HitDieD12
)

func allowed(t *testing.T, r character.HitDieSpendResult) character.SpentHitDice {
	t.Helper()
	a, ok := r.(character.HitDieSpendAllowed)
	if !ok {
		t.Fatalf("spend = %#v, want allowed", r)
	}
	return a.Spent()
}

// Requirement: per case.
// Path: per case.
func TestSpentHitDiceSpend(t *testing.T) {
	t.Parallel()
	d10x5d8x5 := dieCounts{d8: 5, d10: 5}
	cases := []struct {
		name, requirement, path string
		pool, spent             dieCounts
		die                     content.HitPointDie
		benefitsOf              character.RestKind
		wantReason              character.HitDieSpendRejection // zero when allowed
		wantSpent               dieCounts
	}{
		{"SP-1 level 1 Fighter", "FR-039 (SRD 5.2.1, \"Hit Point Dice\", p. 22)", "happy", dieCounts{d10: 1}, nil, d10, character.ShortRest, 0, dieCounts{d10: 1}},
		{"SP-2 the last die is spent", "FR-040", "unhappy (boundary)", dieCounts{d10: 1}, dieCounts{d10: 1}, d10, character.ShortRest, character.NoneRemaining, nil},
		{"SP-3 sizes are tracked separately", "FR-041 (SRD 5.2.1, \"Multiclassing\", p. 25)", "happy", d10x5d8x5, nil, d10, character.ShortRest, 0, dieCounts{d10: 1}},
		{"SP-4 d10 exhausted", "FR-041", "unhappy (boundary, rule interaction)", d10x5d8x5, dieCounts{d10: 5}, d10, character.ShortRest, character.NoneRemaining, nil},
		{"SP-4 d8 still available", "FR-041", "unhappy (boundary, rule interaction)", d10x5d8x5, dieCounts{d10: 5}, d8, character.ShortRest, 0, dieCounts{d8: 1, d10: 5}},
		{"SP-5 a size the pool lacks", "FR-040", "unhappy (invalid input)", d10x5d8x5, nil, d12, character.ShortRest, character.NotInPool, nil},
		{"invalid die is not in the pool", "FR-040", "unhappy (invalid input)", d10x5d8x5, nil, 0, character.ShortRest, character.NotInPool, nil},
		{"SP-8 Long Rest benefits cannot spend", "FR-045 (SRD 5.2.1, \"Short Rest\", p. 187)", "unhappy (state error)", dieCounts{d10: 1}, nil, d10, character.LongRest, character.NotAShortRestBenefit, nil},
		{"SP-9 spent above the pool", "FR-038", "unhappy (invalid input)", dieCounts{d8: 2}, dieCounts{d8: 3}, d8, character.ShortRest, character.SpentExceedsPool, nil},
		{"SP-9 a foreign size above the pool", "FR-038", "unhappy (invalid input)", dieCounts{d8: 2, d10: 1}, dieCounts{d10: 2}, d8, character.ShortRest, character.SpentExceedsPool, nil},
		{"reason order: benefit before exceeds", "FR-040", "unhappy (rule interaction)", dieCounts{d8: 2}, dieCounts{d8: 3}, d8, character.LongRest, character.NotAShortRestBenefit, nil},
		{"reason order: exceeds before not in pool", "FR-040", "unhappy (rule interaction)", dieCounts{d8: 2}, dieCounts{d8: 3}, d12, character.ShortRest, character.SpentExceedsPool, nil},
		{"reason order: not in pool before none remaining", "FR-040", "unhappy (rule interaction)", dieCounts{d8: 2}, dieCounts{d8: 1}, d6, character.ShortRest, character.NotInPool, nil},
		{"invalid benefit kind", "FR-045", "unhappy (invalid input)", dieCounts{d8: 2}, nil, d8, 0, character.NotAShortRestBenefit, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			pool, before := poolOfDice(t, c.pool), spentOf(t, c.spent)
			res := before.Spend(pool, c.die, c.benefitsOf)
			if !before.Equal(spentOf(t, c.spent)) {
				t.Error("Spend changed its receiver")
			}
			if c.wantReason != 0 {
				if r, ok := res.(character.HitDieSpendRejected); !ok || r.Reason() != c.wantReason {
					t.Errorf("spend = %#v, want rejected %v [%s]", res, c.wantReason, c.requirement)
				}
				return
			}
			if got := allowed(t, res); !got.Equal(spentOf(t, c.wantSpent)) {
				t.Errorf("spent = %v, want %v [%s]", got.Entries(), c.wantSpent, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md table 4.8d SP-6, FR-046 - ten d10 for
// a level 5 Fighter and level 5 Paladin are spendable, one at a time, and the
// eleventh is rejected (SRD 5.2.1, "Multiclassing", p. 25).
// Path: unhappy (boundary).
func TestSpentHitDiceSpendTenThenNone(t *testing.T) {
	t.Parallel()
	pool, spent := poolOfDice(t, dieCounts{d10: 10}), character.SpentHitDice{}
	for i := 1; i <= 10; i++ {
		spent = allowed(t, spent.Spend(pool, d10, character.ShortRest))
		if spent.Count(d10) != i || spent.Total() != i {
			t.Fatalf("after %d spends: %v", i, spent.Entries())
		}
	}
	if r, ok := spent.Spend(pool, d10, character.ShortRest).(character.HitDieSpendRejected); !ok || r.Reason() != character.NoneRemaining {
		t.Errorf("eleventh spend = %#v", r)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md table 4.8d SP-7, FR-042 - gaining a
// level adds one die to the pool and leaves the spent dice, so one more is left
// (SRD 5.2.1, "Gaining a Level", step 2, p. 23).
// Path: unhappy (boundary).
func TestSpentHitDiceLevelUpLeavesOneMoreToSpend(t *testing.T) {
	t.Parallel()
	pool := poolOfDice(t, dieCounts{d10: 1})
	spent := allowed(t, character.SpentHitDice{}.Spend(pool, d10, character.ShortRest))
	if _, ok := spent.Spend(pool, d10, character.ShortRest).(character.HitDieSpendRejected); !ok {
		t.Fatal("a second die at level 1 was allowed")
	}
	if got := allowed(t, spent.Spend(pool.Plus(d10), d10, character.ShortRest)); got.Count(d10) != 2 {
		t.Errorf("spent after level up = %v", got.Entries())
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md table 4.8d SP-13 to SP-15, FR-069 - a
// Lifedrinker spend needs no rest, is recorded in the same set, and so is not
// spendable again at the next Short Rest (SRD 5.2.1, "Lifedrinker", p. 73).
// Path: per case.
func TestSpentHitDiceSpendOutsideRest(t *testing.T) {
	t.Parallel()
	pool := poolOfDice(t, dieCounts{d8: 2})
	one := allowed(t, character.SpentHitDice{}.SpendOutsideRest(pool, d8))
	if one.Count(d8) != 1 {
		t.Fatalf("SP-13: spent = %v", one.Entries())
	}
	two := allowed(t, one.Spend(pool, d8, character.ShortRest))
	if two.Count(d8) != 2 {
		t.Fatalf("SP-14 first Short Rest spend: spent = %v", two.Entries())
	}
	if r, ok := two.Spend(pool, d8, character.ShortRest).(character.HitDieSpendRejected); !ok || r.Reason() != character.NoneRemaining {
		t.Errorf("SP-14 second Short Rest spend = %#v", r)
	}
	if r, ok := two.SpendOutsideRest(pool, d8).(character.HitDieSpendRejected); !ok || r.Reason() != character.NoneRemaining {
		t.Errorf("SP-15 = %#v", r)
	}
	if r, ok := character.MustSpentHitDice(dieCounts{d8: 3}).SpendOutsideRest(pool, d8).(character.HitDieSpendRejected); !ok || r.Reason() != character.SpentExceedsPool {
		t.Errorf("exceeding pool = %#v", r)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-037, table 4.8d SP-10, SP-11 - a
// count below 1 is an error joining CountBelowOne; zero entries are not representable;
// the zero value is the empty set; entries are ordered by faces.
// Path: per case.
func TestNewSpentHitDice(t *testing.T) {
	t.Parallel()
	for name, counts := range map[string]dieCounts{"SP-10 zero count": {d8: 0}, "SP-11 zero among valid": {d8: 0, d10: 1}, "negative": {d6: -1}} {
		_, err := character.NewSpentHitDice(counts)
		if !errors.Is(err, character.CountBelowOne) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := character.NewSpentHitDice(dieCounts{0: 1}); !errors.Is(err, srderr.ErrInvalidInput) || errors.Is(err, character.CountBelowOne) {
		t.Errorf("invalid die: err = %v", err)
	}
	s := spentOf(t, dieCounts{d12: 1, d6: 2, d8: 3})
	got := s.Entries()
	if len(got) != 3 || got[0].Die != d6 || got[1].Die != d8 || got[2].Die != d12 || s.Count(d8) != 3 || s.Count(d10) != 0 || s.Total() != 6 {
		t.Errorf("entries = %v", got)
	}
	if e := (character.SpentHitDice{}); e.Total() != 0 || !e.Equal(spentOf(t, nil)) || e.Count(d8) != 0 {
		t.Error("the zero SpentHitDice is not the empty set")
	}
	if character.CountBelowOne.String() != "count_below_one" || character.SpentHitDiceProblem(0).String() != "unknown" || character.SpentHitDiceProblem(2).String() != "unknown" || character.CountBelowOne.Error() == "" {
		t.Error("problem names are wrong")
	}
	defer func() {
		if e, ok := recover().(error); !ok || !errors.Is(e, character.CountBelowOne) {
			t.Error("MustSpentHitDice did not panic with CountBelowOne")
		}
	}()
	character.MustSpentHitDice(dieCounts{d8: 0})
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-048 - SpentHitDice is immutable:
// the map given is copied, Entries is a copy, and Equal compares by value and
// only with another SpentHitDice.
// Path: unhappy (immutability).
func TestSpentHitDiceIsImmutableAndComparable(t *testing.T) {
	t.Parallel()
	in := dieCounts{d8: 2}
	s := spentOf(t, in)
	in[d8] = 9
	s.Entries()[0].Count = 9
	if s.Count(d8) != 2 {
		t.Errorf("count = %d, want 2", s.Count(d8))
	}
	var fs character.FeatureState = s
	if !fs.Equal(spentOf(t, dieCounts{d8: 2})) || fs.Equal(spentOf(t, dieCounts{d8: 3})) || fs.Equal(spentOf(t, dieCounts{d10: 2})) || fs.Equal(spentOf(t, nil)) {
		t.Error("Equal compares the wrong thing")
	}
	if fs.Equal(heroicStateStub{}) {
		t.Error("Equal matched a state of another type")
	}
}

type heroicStateStub struct{}

func (heroicStateStub) Equal(character.FeatureState) bool { return true }

// Requirement: doc/specs/rests-and-hit-dice.md Q-6, shared-types-and-ownership.md
// FR-022 - SpentHitDice is stored and read back under SpentHitDiceKey through
// FeatureStates, beside a foreign key, with no type assertion by the caller.
// Path: happy.
func TestSpentHitDiceStoredUnderItsKey(t *testing.T) {
	t.Parallel()
	key := character.SpentHitDiceKey()
	if key.Name() != "spent_hit_dice" || key != character.SpentHitDiceKey() {
		t.Errorf("key = %q", key.Name())
	}
	s := spentOf(t, dieCounts{d8: 1})
	states, err := character.SetFeatureState(character.FeatureStates{}, key, s)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := character.ReadFeatureState(states, key)
	if err != nil || !ok || !got.Equal(s) {
		t.Errorf("read back = %v, %v, %v", got, ok, err)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-040 - rejection reasons have
// stable snake_case names, used as the srd5e.character.rejection value.
// Path: happy.
func TestHitDieSpendRejectionNames(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.HitDieSpendRejection]string{
		character.NoneRemaining: "none_remaining", character.NotInPool: "not_in_pool",
		character.NotAShortRestBenefit: "not_a_short_rest_benefit", character.SpentExceedsPool: "spent_exceeds_pool", 0: "unknown",
	} {
		if r.String() != want {
			t.Errorf("%d = %q, want %q", int(r), r.String(), want)
		}
	}
}
