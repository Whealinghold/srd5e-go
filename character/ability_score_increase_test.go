package character_test

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const aiSpec = "doc/specs/level-advancement-and-multiclassing.md"

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-025, FR-026,
// FR-027 - the three Ability Score Improvement and Epic Boon factories hold their
// amounts and maximum; two-by-one needs two distinct abilities.
// Path: per case.
func TestAbilityScoreIncreaseFactories(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		make                    func() (character.AbilityScoreIncrease, error)
		want                    map[ability.Ability]int
		max                     int
		invalid                 bool
	}{
		{"one by two", aiSpec + " FR-025, SRD 5.2.1, Ability Score Improvement, p. 87", "happy", func() (character.AbilityScoreIncrease, error) {
			return character.NewAbilityScoreImprovementOneByTwo(str)
		}, map[ability.Ability]int{str: 2}, 20, false},
		{"two by one", aiSpec + " FR-026, SRD 5.2.1, Ability Score Improvement, p. 87", "happy", func() (character.AbilityScoreIncrease, error) {
			return character.NewAbilityScoreImprovementTwoByOne(str, dex)
		}, map[ability.Ability]int{str: 1, dex: 1}, 20, false},
		{"two by one, the same ability twice", aiSpec + " FR-026", "unhappy (invalid input)", func() (character.AbilityScoreIncrease, error) {
			return character.NewAbilityScoreImprovementTwoByOne(str, str)
		}, nil, 0, true},
		{"epic boon", aiSpec + " FR-027, SRD 5.2.1, Epic Boon Feats, p. 88", "happy", func() (character.AbilityScoreIncrease, error) {
			return character.NewEpicBoonIncrease(cha)
		}, map[ability.Ability]int{cha: 1}, 30, false},
		{"one by two, invalid ability", aiSpec + " FR-029", "unhappy (invalid input)", func() (character.AbilityScoreIncrease, error) {
			return character.NewAbilityScoreImprovementOneByTwo(ability.Ability(0))
		}, nil, 0, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.make()
			if c.invalid {
				if !isInvalid(err) || !got.IsZero() {
					t.Errorf("err = %v, want invalid input [%s]", err, c.requirement)
				}
				return
			}
			if err != nil || !maps.Equal(got.Increases(), c.want) || got.MaximumScore() != c.max || got.IsZero() {
				t.Errorf("got %v max %d err %v, want %v max %d [%s]", got.Increases(), got.MaximumScore(), err, c.want, c.max, c.requirement)
			}
		})
	}
	var e *character.AbilityScoreIncreaseError
	if _, err := character.NewAbilityScoreImprovementTwoByOne(dex, dex); !errors.As(err, &e) || e.Problem != character.IncreaseSameAbilityTwice || e.Parameter != "second" {
		t.Errorf("same ability twice: %v", err)
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-028, FR-029 -
// the general constructor refuses an empty increase, an amount below 1 and a
// maximum outside 20 to 30, reports every problem in the fixed order and names the
// parameter and value; MustAbilityScoreIncrease panics with the same error.
// Path: per case.
func TestNewAbilityScoreIncrease(t *testing.T) {
	t.Parallel()
	type problem struct {
		p     character.AbilityScoreIncreaseProblem
		param string
		value int
	}
	for _, c := range []struct {
		name, requirement, path string
		in                      map[ability.Ability]int
		max                     int
		want                    []problem
	}{
		{"valid, one ability", aiSpec + " FR-028 (Grappler: +1 Strength or Dexterity, SRD 5.2.1, p. 87)", "happy", map[ability.Ability]int{dex: 1}, 20, nil},
		{"valid, maximum 30", aiSpec + " FR-028", "unhappy (boundary)", map[ability.Ability]int{dex: 1}, 30, nil},
		{"maximum 19", aiSpec + " FR-028", "unhappy (boundary)", map[ability.Ability]int{dex: 1}, 19, []problem{{character.IncreaseMaximumBelowDefault, "maximumScore", 19}}},
		{"maximum 31", aiSpec + " FR-028", "unhappy (boundary)", map[ability.Ability]int{dex: 1}, 31, []problem{{character.IncreaseMaximumAboveHighest, "maximumScore", 31}}},
		{"empty", aiSpec + " FR-028", "unhappy (invalid input)", nil, 20, []problem{{character.IncreaseEmpty, "increases", 0}}},
		{"amount zero", aiSpec + " FR-028", "unhappy (boundary)", map[ability.Ability]int{dex: 0}, 20, []problem{{character.IncreaseAmountBelowMinimum, "increases", 0}}},
		{"amount negative", aiSpec + " FR-028", "unhappy (invalid input)", map[ability.Ability]int{dex: -3}, 20, []problem{{character.IncreaseAmountBelowMinimum, "increases", -3}}},
		{"invalid ability", aiSpec + " FR-029", "unhappy (invalid input)", map[ability.Ability]int{ability.Ability(9): 1}, 20, []problem{{character.IncreaseInvalidAbility, "increases", 9}}},
		{"every problem in order", aiSpec + " FR-029", "unhappy (rule interaction)", map[ability.Ability]int{cha: 0, str: -1}, 31, []problem{
			{character.IncreaseAmountBelowMinimum, "increases", -1}, {character.IncreaseAmountBelowMinimum, "increases", 0}, {character.IncreaseMaximumAboveHighest, "maximumScore", 31}}},
		{"empty and low maximum", aiSpec + " FR-029", "unhappy (rule interaction)", map[ability.Ability]int{}, 5, []problem{
			{character.IncreaseEmpty, "increases", 0}, {character.IncreaseMaximumBelowDefault, "maximumScore", 5}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			inc, err := character.NewAbilityScoreIncrease(c.in, c.max)
			if len(c.want) == 0 {
				if err != nil || inc.IsZero() || inc.MaximumScore() != c.max {
					t.Fatalf("err = %v [%s]", err, c.requirement)
				}
				return
			}
			if !isInvalid(err) || !inc.IsZero() {
				t.Fatalf("err = %v, want invalid input [%s]", err, c.requirement)
			}
			var got []problem
			for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
				var pe *character.AbilityScoreIncreaseError
				if !errors.As(e, &pe) || !errors.Is(e, srderr.ErrInvalidInput) {
					t.Fatalf("not a typed problem: %v", e)
				}
				got = append(got, problem{pe.Problem, pe.Parameter, pe.InvalidValue})
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("problems = %v, want %v [%s]", got, c.want, c.requirement)
			}
			defer func() {
				r, _ := recover().(error)
				if r == nil || !isInvalid(r) {
					t.Errorf("MustAbilityScoreIncrease panic = %v, want the NewAbilityScoreIncrease error", r)
				}
			}()
			character.MustAbilityScoreIncrease(c.in, c.max)
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-029 - the
// error matches by problem and parameter and its names are stable.
// Path: unhappy (invalid input).
func TestAbilityScoreIncreaseErrorIdentity(t *testing.T) {
	t.Parallel()
	e := &character.AbilityScoreIncreaseError{Problem: character.IncreaseEmpty, Parameter: "increases"}
	if !errors.Is(e, &character.AbilityScoreIncreaseError{Problem: character.IncreaseEmpty, Parameter: "increases", InvalidValue: 7}) ||
		errors.Is(e, &character.AbilityScoreIncreaseError{Problem: character.IncreaseEmpty, Parameter: "maximumScore"}) ||
		errors.Is(e, &character.AbilityScoreIncreaseError{Problem: character.IncreaseAmountBelowMinimum, Parameter: "increases"}) ||
		errors.Is(e, errors.New("other")) {
		t.Error("Is must compare Problem and Parameter only")
	}
	amount := &character.AbilityScoreIncreaseError{Problem: character.IncreaseAmountBelowMinimum, Parameter: "maximumScore"}
	if errors.Is(amount, &character.AbilityScoreIncreaseError{Problem: character.IncreaseEmpty, Parameter: "maximumScore"}) || errors.Is(amount, &character.AbilityScoreIncreaseError{Problem: character.IncreaseAmountBelowMinimum, Parameter: "increases"}) {
		t.Error("Is must not match a lower Problem or a lesser Parameter")
	}
	var none *character.AbilityScoreIncreaseError
	if e.Is(none) || none.Is(e) {
		t.Error("a nil error matches nothing")
	}
	for p, want := range map[character.AbilityScoreIncreaseProblem]string{
		character.IncreaseEmpty: "empty", character.IncreaseAmountBelowMinimum: "amountBelowMinimum", character.IncreaseInvalidAbility: "invalidAbility",
		character.IncreaseMaximumBelowDefault: "maximumBelowDefault", character.IncreaseMaximumAboveHighest: "maximumAboveHighest",
		character.IncreaseSameAbilityTwice: "sameAbilityTwice", 0: "unknown",
	} {
		if p.String() != want {
			t.Errorf("%d = %q, want %q", int(p), p.String(), want)
		}
	}
	if e.Error() == "" {
		t.Error("Error must render")
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-030, FR-105 -
// an increase copies its map on the way in and out, and compares by value.
// Path: unhappy (immutability).
func TestAbilityScoreIncreaseCopiesAndCompares(t *testing.T) {
	t.Parallel()
	in := map[ability.Ability]int{str: 2}
	inc := character.MustAbilityScoreIncrease(in, 20)
	in[str] = 9
	got := inc.Increases()
	got[dex] = 5
	if !maps.Equal(inc.Increases(), map[ability.Ability]int{str: 2}) {
		t.Errorf("increases = %v", inc.Increases())
	}
	same := character.MustAbilityScoreIncrease(map[ability.Ability]int{str: 2}, 20)
	for name, other := range map[string]character.AbilityScoreIncrease{
		"another amount": character.MustAbilityScoreIncrease(map[ability.Ability]int{str: 1}, 20), "another ability": character.MustAbilityScoreIncrease(map[ability.Ability]int{dex: 2}, 20),
		"another maximum": character.MustAbilityScoreIncrease(map[ability.Ability]int{str: 2}, 21), "zero": {},
	} {
		if inc.Equal(other) {
			t.Errorf("%s must differ", name)
		}
	}
	if character.MustAbilityScoreIncrease(map[ability.Ability]int{str: 2}, 21).Equal(character.MustAbilityScoreIncrease(map[ability.Ability]int{str: 2}, 20)) {
		t.Error("a higher maximum must differ from a lower one")
	}
	if !inc.Equal(same) || !(character.AbilityScoreIncrease{}).IsZero() {
		t.Error("equal values must be Equal and the zero value must be IsZero")
	}
}

type aiCase struct {
	name, requirement, path string
	inc                     character.AbilityScoreIncrease
	scores                  [6]int // STR DEX CON INT WIS CHA
	level                   int
	want                    map[ability.Ability]int // scores after, for the abilities named
	hp                      int                     // Hit Point maximum rise
	events                  []string                // "S:<ability>" for AbilityScoreChanged, "H" for HitPointMaximumChanged
	over                    []ability.Ability       // rejected abilities
	max                     int
}

func fighterAt(t *testing.T, level int, s [6]int) character.Character {
	t.Helper()
	return restored(t, 0, 100, scoresOf(t, s[0], s[1], s[2], s[3], s[4], s[5]), cl("fighter", level))
}

func eventTags(events []character.Event) []string {
	var out []string
	for _, e := range events {
		switch e := e.(type) {
		case character.AbilityScoreChanged:
			out = append(out, "S:"+strings.ToUpper(e.Ability().String()[:1])+e.Ability().String()[1:])
		case character.HitPointMaximumChanged:
			out = append(out, "H")
		default:
			out = append(out, "?")
		}
	}
	return out
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-031 to FR-036,
// FR-038 - Table AI, one row each: amounts added, the maximum honoured with no
// clamp and no partial increase, Constitution Hit Points, event order.
// Path: per case.
func TestAbilityScoreIncreaserTableAI(t *testing.T) {
	t.Parallel()
	one := func(a ability.Ability, n, max int) character.AbilityScoreIncrease {
		return character.MustAbilityScoreIncrease(map[ability.Ability]int{a: n}, max)
	}
	ai := func(m map[ability.Ability]int, max int) character.AbilityScoreIncrease {
		return character.MustAbilityScoreIncrease(m, max)
	}
	two := func(a ability.Ability) character.AbilityScoreIncrease {
		return must(character.NewAbilityScoreImprovementOneByTwo(a))
	}
	boon := func(a ability.Ability) character.AbilityScoreIncrease { return must(character.NewEpicBoonIncrease(a)) }
	s := func(a, b, c, d, e, f int) [6]int { return [6]int{a, b, c, d, e, f} }
	for _, c := range []aiCase{
		{name: "one by two", requirement: aiSpec + " FR-031, FR-034 (p. 87)", path: "happy", inc: two(str), scores: s(15, 10, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{str: 17}, events: []string{"S:Strength"}},
		{name: "one by two, up to the maximum", requirement: aiSpec + " FR-032", path: "unhappy (boundary)", inc: two(str), scores: s(18, 10, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{str: 20}, events: []string{"S:Strength"}},
		{name: "one by two, one above the maximum", requirement: aiSpec + " FR-032, FR-033", path: "unhappy (boundary)", inc: two(str), scores: s(19, 10, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{str: 19}, over: []ability.Ability{str}, max: 20},
		{name: "one by two, at 20", requirement: aiSpec + " FR-032", path: "unhappy (boundary)", inc: two(str), scores: s(20, 10, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{str: 20}, over: []ability.Ability{str}, max: 20},
		{name: "two by one", requirement: aiSpec + " FR-031", path: "happy", inc: must(character.NewAbilityScoreImprovementTwoByOne(str, dex)), scores: s(15, 14, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{str: 16, dex: 15}, events: []string{"S:Strength", "S:Dexterity"}},
		{name: "two by one, named in reverse order", requirement: aiSpec + " FR-031", path: "unhappy (determinism)", inc: must(character.NewAbilityScoreImprovementTwoByOne(dex, str)), scores: s(15, 14, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{str: 16, dex: 15}, events: []string{"S:Strength", "S:Dexterity"}},
		{name: "two by one, both reach the maximum", requirement: aiSpec + " FR-032", path: "unhappy (boundary)", inc: must(character.NewAbilityScoreImprovementTwoByOne(str, dex)), scores: s(19, 19, 10, 10, 10, 10), level: 8, want: map[ability.Ability]int{str: 20, dex: 20}, events: []string{"S:Strength", "S:Dexterity"}},
		{name: "two by one, one capped", requirement: aiSpec + " FR-033", path: "unhappy (rule interaction)", inc: must(character.NewAbilityScoreImprovementTwoByOne(str, dex)), scores: s(20, 14, 10, 10, 10, 10), level: 8, want: map[ability.Ability]int{str: 20, dex: 14}, over: []ability.Ability{str}, max: 20},
		{name: "two by one, both capped", requirement: aiSpec + " FR-033", path: "unhappy (rule interaction)", inc: must(character.NewAbilityScoreImprovementTwoByOne(str, dex)), scores: s(20, 20, 10, 10, 10, 10), level: 8, want: map[ability.Ability]int{str: 20, dex: 20}, over: []ability.Ability{str, dex}, max: 20},
		{name: "score already above 20", requirement: aiSpec + " FR-032, Q-7", path: "unhappy (boundary)", inc: one(str, 1, 20), scores: s(22, 10, 10, 10, 10, 10), level: 12, want: map[ability.Ability]int{str: 22}, over: []ability.Ability{str}, max: 20},
		{name: "single +1 with maximum 20", requirement: aiSpec + " FR-028", path: "happy", inc: one(dex, 1, 20), scores: s(10, 13, 10, 10, 10, 10), level: 4, want: map[ability.Ability]int{dex: 14}, events: []string{"S:Dexterity"}},
		{name: "epic boon", requirement: aiSpec + " FR-027 (p. 88)", path: "happy", inc: boon(str), scores: s(20, 10, 10, 10, 10, 10), level: 19, want: map[ability.Ability]int{str: 21}, events: []string{"S:Strength"}},
		{name: "epic boon, modifier rises", requirement: aiSpec + " FR-027", path: "happy", inc: boon(str), scores: s(21, 10, 10, 10, 10, 10), level: 19, want: map[ability.Ability]int{str: 22}, events: []string{"S:Strength"}},
		{name: "epic boon, up to 30", requirement: aiSpec + " FR-027, FR-032", path: "unhappy (boundary)", inc: boon(str), scores: s(29, 10, 10, 10, 10, 10), level: 19, want: map[ability.Ability]int{str: 30}, events: []string{"S:Strength"}},
		{name: "epic boon, one above 30", requirement: aiSpec + " FR-032", path: "unhappy (boundary)", inc: boon(str), scores: s(30, 10, 10, 10, 10, 10), level: 19, want: map[ability.Ability]int{str: 30}, over: []ability.Ability{str}, max: 30},
		{name: "Constitution, SRD example", requirement: aiSpec + " FR-034 (SRD 5.2.1, Gaining a Level step 5, p. 23)", path: "happy", inc: one(con, 1, 20), scores: s(10, 10, 17, 10, 10, 10), level: 8, want: map[ability.Ability]int{con: 18}, hp: 8, events: []string{"S:Constitution", "H"}},
		{name: "Constitution, one by two, modifier rises by one", requirement: aiSpec + " FR-034", path: "happy", inc: two(con), scores: s(10, 10, 16, 10, 10, 10), level: 8, want: map[ability.Ability]int{con: 18}, hp: 8, events: []string{"S:Constitution", "H"}},
		{name: "Constitution, modifier unchanged", requirement: aiSpec + " FR-035", path: "unhappy (rule interaction)", inc: one(con, 1, 20), scores: s(10, 10, 16, 10, 10, 10), level: 8, want: map[ability.Ability]int{con: 17}, events: []string{"S:Constitution"}},
		{name: "Constitution, odd score to even score", requirement: aiSpec + " FR-034", path: "happy", inc: one(con, 1, 20), scores: s(10, 10, 15, 10, 10, 10), level: 5, want: map[ability.Ability]int{con: 16}, hp: 5, events: []string{"S:Constitution", "H"}},
		{name: "Constitution at level 19", requirement: aiSpec + " FR-034", path: "unhappy (boundary)", inc: boon(con), scores: s(10, 10, 21, 10, 10, 10), level: 19, want: map[ability.Ability]int{con: 22}, hp: 19, events: []string{"S:Constitution", "H"}},
		{name: "other ability, no Hit Point change", requirement: aiSpec + " FR-035", path: "unhappy (rule interaction)", inc: two(wis), scores: s(10, 10, 10, 10, 15, 10), level: 8, want: map[ability.Ability]int{wis: 17}, events: []string{"S:Wisdom"}},
		{name: "event order with a later ability", requirement: aiSpec + " FR-036", path: "unhappy (determinism)", inc: ai(map[ability.Ability]int{con: 1, wis: 1}, 20), scores: s(10, 10, 17, 10, 11, 10), level: 8, want: map[ability.Ability]int{con: 18, wis: 12}, hp: 8, events: []string{"S:Constitution", "S:Wisdom", "H"}},
		{name: "event order with an earlier ability", requirement: aiSpec + " FR-036", path: "unhappy (determinism)", inc: ai(map[ability.Ability]int{str: 1, con: 1}, 20), scores: s(15, 10, 17, 10, 10, 10), level: 8, want: map[ability.Ability]int{str: 16, con: 18}, hp: 8, events: []string{"S:Strength", "S:Constitution", "H"}},
		{name: "level 1 character, direct call", requirement: aiSpec + " FR-037", path: "unhappy (rule interaction)", inc: two(wis), scores: s(10, 10, 10, 10, 10, 10), level: 1, want: map[ability.Ability]int{wis: 12}, events: []string{"S:Wisdom"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := fighterAt(t, c.level, c.scores)
			tr, err := character.AbilityScoreIncreaser{}.Apply(before, c.inc)
			if err != nil {
				t.Fatal(err)
			}
			after := before
			switch r := tr.(type) {
			case character.TransitionChanged[character.AbilityScoreIncreaseRejection]:
				if len(c.over) != 0 {
					t.Fatalf("accepted, want rejected %v [%s]", c.over, c.requirement)
				}
				after = r.Character()
				if got := eventTags(r.Events()); !slices.Equal(got, c.events) {
					t.Errorf("events = %v, want %v [%s]", got, c.events, c.requirement)
				}
				if rise := after.HitPointMaximum().Value() - before.HitPointMaximum().Value(); rise != c.hp {
					t.Errorf("Hit Point maximum rose %d, want %d [%s]", rise, c.hp, c.requirement)
				}
				for _, e := range r.Events() {
					if h, ok := e.(character.HitPointMaximumChanged); ok && (h.Previous() != before.HitPointMaximum() || h.Current() != after.HitPointMaximum()) {
						t.Errorf("HitPointMaximumChanged %d to %d, want from the maximum before to the maximum after", h.Previous().Value(), h.Current().Value())
					}
				}
			case character.TransitionRejected[character.AbilityScoreIncreaseRejection]:
				ex, ok := r.Rejection().(character.IncreaseExceedsMaximum)
				if !ok || !slices.Equal(ex.Abilities(), c.over) || ex.MaximumScore() != c.max {
					t.Errorf("rejection = %#v, want %v max %d [%s]", r.Rejection(), c.over, c.max, c.requirement)
				}
			}
			for a, want := range c.want {
				if got := after.AbilityScores().Score(a).Value(); got != want {
					t.Errorf("%v = %d, want %d [%s]", a, got, want, c.requirement)
				}
			}
			for _, a := range ability.All() {
				if _, named := c.want[a]; !named && after.AbilityScores().Score(a).Value() != c.scores[int(a)-int(ability.Strength)] {
					t.Errorf("%v changed to %d", a, after.AbilityScores().Score(a).Value())
				}
			}
		})
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-038, FR-040 -
// applying the same increase twice is two independent applications, each
// checked against the maximum, and Apply leaves its inputs unchanged and is
// repeatable.
// Path: unhappy (state error, determinism).
func TestAbilityScoreIncreaserRepeats(t *testing.T) {
	t.Parallel()
	inc := must(character.NewAbilityScoreImprovementOneByTwo(str))
	start := fighterAt(t, 12, [6]int{15, 10, 10, 10, 10, 10})
	apply := func(c character.Character) character.Transition[character.AbilityScoreIncreaseRejection] {
		tr, err := character.AbilityScoreIncreaser{}.Apply(c, inc)
		if err != nil {
			t.Fatal(err)
		}
		return tr
	}
	first := apply(start).(character.TransitionChanged[character.AbilityScoreIncreaseRejection]).Character()
	second := apply(first).(character.TransitionChanged[character.AbilityScoreIncreaseRejection]).Character()
	if first.AbilityScores().Score(str).Value() != 17 || second.AbilityScores().Score(str).Value() != 19 {
		t.Errorf("repeat: %d then %d", first.AbilityScores().Score(str).Value(), second.AbilityScores().Score(str).Value())
	}
	if _, ok := apply(second).(character.TransitionRejected[character.AbilityScoreIncreaseRejection]); !ok {
		t.Error("the third application (21 over 20) must be refused")
	}
	if start.AbilityScores().Score(str).Value() != 15 {
		t.Error("the input character must be unchanged")
	}
	again := apply(start).(character.TransitionChanged[character.AbilityScoreIncreaseRejection])
	if !again.Character().Equal(first) {
		t.Error("the same inputs must give an equal result")
	}
	if !inc.Equal(must(character.NewAbilityScoreImprovementOneByTwo(str))) {
		t.Error("the increase must be unchanged")
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-033, FR-105 -
// IncreaseExceedsMaximum returns a copy of its abilities.
// Path: unhappy (immutability).
func TestIncreaseExceedsMaximumCopiesAbilities(t *testing.T) {
	t.Parallel()
	c := fighterAt(t, 4, [6]int{20, 20, 10, 10, 10, 10})
	tr, err := character.AbilityScoreIncreaser{}.Apply(c, must(character.NewAbilityScoreImprovementTwoByOne(str, dex)))
	if err != nil {
		t.Fatal(err)
	}
	r := tr.(character.TransitionRejected[character.AbilityScoreIncreaseRejection]).Rejection().(character.IncreaseExceedsMaximum)
	r.Abilities()[0] = cha
	if !slices.Equal(r.Abilities(), []ability.Ability{str, dex}) {
		t.Errorf("abilities = %v", r.Abilities())
	}
}

// Requirement: doc/specs/level-advancement-and-multiclassing.md FR-028 - a zero
// Character or a zero increase is an invalid-input error, never a result.
// Path: unhappy (invalid input).
func TestAbilityScoreIncreaserInvalidInput(t *testing.T) {
	t.Parallel()
	c := fighterAt(t, 4, [6]int{10, 10, 10, 10, 10, 10})
	if _, err := (character.AbilityScoreIncreaser{}).Apply(character.Character{}, must(character.NewEpicBoonIncrease(str))); !isInvalid(err) {
		t.Errorf("zero character: %v", err)
	}
	if _, err := (character.AbilityScoreIncreaser{}).Apply(c, character.AbilityScoreIncrease{}); !isInvalid(err) {
		t.Errorf("zero increase: %v", err)
	}
}
