package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func mod(v int) modifier.Modifier { return modifier.New(v) }

// trkContrib builds the roll contributions of the conditions on a creature.
func trkContrib(conditions ...combat.Condition) combat.RollContributions {
	return combat.ConditionEffects{}.Initiative(combat.MustConditionSet(conditions, nil))
}

// trkParticipant builds a participant of the ids with the circumstances.
func trkParticipant(tb testing.TB, in combat.InitiativeParticipantInput) combat.InitiativeParticipant {
	tb.Helper()
	p, err := combat.NewInitiativeParticipant(in)
	if err != nil {
		tb.Fatalf("NewInitiativeParticipant: %v", err)
	}
	return p
}

// Requirement: doc/specs/initiative-and-actions.md FR-001, FR-111, FR-139 - the count is the d20 used plus the Initiative modifier plus the roll-only modifier; a natural 20 or 1 is only a 20 and a 1; any integer is a count.
// Path: per case.
func TestInitiativeCountFromRoll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		mode                    modifier.RollMode
		dice                    []int
		mod, rollOnly           int
		want                    int
	}{
		{"typical", "FR-001", "happy", modifier.Normal, []int{10}, 2, 0, 12},
		{"zero modifier", "FR-001", "happy", modifier.Normal, []int{10}, 0, 0, 10},
		{"natural 20 is only a 20", "FR-001", "unhappy (boundary)", modifier.Normal, []int{20}, 0, 0, 20},
		{"natural 1 is only a 1", "FR-001", "unhappy (boundary)", modifier.Normal, []int{1}, 0, 0, 1},
		{"lowest possible", "FR-011", "unhappy (boundary)", modifier.Normal, []int{1}, -5, 0, -4},
		{"highest possible", "FR-011", "unhappy (boundary)", modifier.Normal, []int{20}, 10, 0, 30},
		{"Advantage takes the higher", "FR-002", "happy", modifier.WithAdvantage, []int{3, 18}, 2, 0, 20},
		{"Advantage, reversed", "FR-002", "happy", modifier.WithAdvantage, []int{18, 3}, 2, 0, 20},
		{"Advantage, equal dice", "FR-002", "happy", modifier.WithAdvantage, []int{7, 7}, 0, 0, 7},
		{"Disadvantage takes the lower", "FR-002", "happy", modifier.WithDisadvantage, []int{18, 3}, 2, 0, 5},
		{"non-Dexterity modifier", "FR-001", "unhappy (rule interaction)", modifier.Normal, []int{10}, 7, 0, 17},
		{"roll-only modifier", "FR-111", "unhappy (rule interaction)", modifier.Normal, []int{10}, 2, -4, 8},
		{"roll-only alone drives it negative", "FR-111", "unhappy (rule interaction)", modifier.Normal, []int{1}, -5, -12, -16},
		{"saturates", "FR-011", "unhappy (boundary)", modifier.Normal, []int{20}, math.MaxInt, math.MaxInt, math.MaxInt},
		{"saturates low", "FR-011", "unhappy (boundary)", modifier.Normal, []int{1}, math.MinInt, math.MinInt, math.MinInt},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roll := d20.MustRoll(c.mode, c.dice...)
			if got := combat.InitiativeCount(0).FromRoll(roll, mod(c.mod), mod(c.rollOnly)); got != combat.InitiativeCount(c.want) {
				t.Errorf("FromRoll = %d, want %d [doc/specs/initiative-and-actions.md %s]", got, c.want, c.requirement)
			}
		})
	}
	// Requirement: doc/specs/initiative-and-actions.md FR-002, FR-139 - a die replaced afterwards (Heroic Inspiration) recomputes the count from the roll.
	roll, err := d20.MustRoll(modifier.WithAdvantage, 3, 18).ReplaceDie(1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got := combat.InitiativeCount(0).FromRoll(roll, mod(2), mod(0)); got != 7 {
		t.Errorf("recomputed count = %d, want 7 [FR-139]", got)
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-006, FR-007, FR-003 to FR-005 - the roll mode resolves over surprise, the conditions' circumstances and other circumstances together, never over the pre-resolved mode.
// Path: per case.
func TestInitiativeParticipantMode(t *testing.T) {
	t.Parallel()
	adv, dis := modifier.Advantage{}, modifier.Disadvantage{}
	cases := []struct {
		name, requirement, path string
		surprise                combat.Surprise
		conditions              []combat.Condition
		others                  []modifier.RollCircumstance
		want                    modifier.RollMode
	}{
		{"none", "FR-006", "unhappy (empty case)", combat.NotSurprised, nil, nil, modifier.Normal},
		{"surprised", "FR-003", "happy", combat.Surprised, nil, nil, modifier.WithDisadvantage},
		{"Incapacitated", "FR-004", "happy", combat.NotSurprised, []combat.Condition{combat.Incapacitated{}}, nil, modifier.WithDisadvantage},
		{"Invisible", "FR-005", "happy", combat.NotSurprised, []combat.Condition{combat.Invisible{}}, nil, modifier.WithAdvantage},
		{"two Disadvantage do not stack", "FR-006", "unhappy (rule interaction)", combat.Surprised, []combat.Condition{combat.Incapacitated{}}, nil, modifier.WithDisadvantage},
		{"Advantage cancels Disadvantage", "FR-006", "unhappy (rule interaction)", combat.Surprised, []combat.Condition{combat.Invisible{}}, nil, modifier.Normal},
		{"one Advantage against two Disadvantage", "FR-006", "unhappy (rule interaction)", combat.Surprised, []combat.Condition{combat.Incapacitated{}, combat.Invisible{}}, nil, modifier.Normal},
		{"pre-resolved normal is not merged with other Advantage", "FR-006", "unhappy (rule interaction)", combat.NotSurprised, []combat.Condition{combat.Incapacitated{}, combat.Invisible{}}, []modifier.RollCircumstance{adv}, modifier.Normal},
		{"other Advantage joins Invisible", "FR-007", "unhappy (rule interaction)", combat.NotSurprised, []combat.Condition{combat.Invisible{}}, []modifier.RollCircumstance{adv}, modifier.WithAdvantage},
		{"other Advantage cancels surprise", "FR-007", "unhappy (rule interaction)", combat.Surprised, nil, []modifier.RollCircumstance{adv}, modifier.Normal},
		{"other Disadvantage alone", "FR-007", "happy", combat.NotSurprised, nil, []modifier.RollCircumstance{dis, dis}, modifier.WithDisadvantage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := trkParticipant(t, combat.InitiativeParticipantInput{IDs: []combat.CombatantID{trkA}, Surprise: c.surprise,
				Contributions: trkContrib(c.conditions...), Circumstances: c.others})
			if got := p.Mode(); got != c.want {
				t.Errorf("Mode = %v, want %v [doc/specs/initiative-and-actions.md %s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-013, FR-014 - an entry with no id, a blank id, an id twice or an undefined surprise is an invalid-input error; lists are copied.
// Path: unhappy (invalid input).
func TestNewInitiativeParticipant(t *testing.T) {
	t.Parallel()
	for name, in := range map[string]combat.InitiativeParticipantInput{
		"no ids":            {},
		"blank id":          {IDs: []combat.CombatantID{trkA, ""}},
		"id twice":          {IDs: []combat.CombatantID{trkA, trkB, trkA}},
		"unknown surpise":   {IDs: []combat.CombatantID{trkA}, Surprise: 5},
		"negative surprise": {IDs: []combat.CombatantID{trkA}, Surprise: -1},
	} {
		if p, err := combat.NewInitiativeParticipant(in); !errors.Is(err, srderr.ErrInvalidInput) || !p.IsZero() {
			t.Errorf("%s: NewInitiativeParticipant = %+v, %v; want a zero participant and invalid input", name, p, err)
		}
	}
	ids := []combat.CombatantID{trkA, trkB}
	p := trkParticipant(t, combat.InitiativeParticipantInput{IDs: ids})
	ids[0] = trkC
	got := p.IDs()
	got[1] = trkC
	if want := []combat.CombatantID{trkA, trkB}; p.IDs()[0] != want[0] || p.IDs()[1] != want[1] {
		t.Errorf("IDs = %v, want %v: the participant shares its list [FR-017]", p.IDs(), want)
	}
	if (combat.InitiativeParticipant{}).IsZero() != true || p.IsZero() {
		t.Error("IsZero is wrong")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-013, FR-014 - NewInitiativeParticipant never panics for any ids and surprise.
// Path: unhappy (invalid input).
func FuzzNewInitiativeParticipant(f *testing.F) {
	f.Add("a", "b", 0)
	f.Add("a", "a", 7)
	f.Add("", " ", -1)
	f.Fuzz(func(t *testing.T, a, b string, surprise int) {
		p, err := combat.NewInitiativeParticipant(combat.InitiativeParticipantInput{
			IDs: []combat.CombatantID{combat.CombatantID(a), combat.CombatantID(b)}, Surprise: combat.Surprise(surprise)})
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if len(p.IDs()) != 2 || p.Mode().DiceCount() < 1 {
			t.Fatalf("participant %+v is malformed", p)
		}
	})
}
