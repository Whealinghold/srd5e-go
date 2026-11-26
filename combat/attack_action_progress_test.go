package combat_test

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/attack-and-help-actions.md FR-015, FR-016, FR-017, FR-062, Table B - each row records the next attack
// (attack number n, with the event naming the means or the replacement) or refuses it with AttackLimitReached and the progress unchanged.
// Path: per case.
func TestRecordAttackTableB(t *testing.T) {
	t.Parallel()
	cases := []struct {
		row               string
		general, pact     int
		history, next     string
		want              int // attack number recorded, or 0 for a refusal
		requirement, path string
	}{
		{"B01", 1, 1, "", "O", 1, "FR-015", "happy"},
		{"B02", 1, 1, "O", "O", 0, "FR-015", "unhappy (boundary)"},
		{"B03", 2, 2, "O", "U", 2, "FR-017", "happy"},
		{"B04", 2, 2, "O U", "O", 0, "FR-015", "unhappy (boundary)"},
		{"B05", 1, 2, "", "P", 1, "FR-016", "happy"},
		{"B06", 1, 2, "P", "P", 2, "FR-016", "happy"},
		{"B07", 1, 2, "P P", "P", 0, "FR-016", "unhappy (boundary)"},
		{"B08", 1, 2, "P", "O", 0, "FR-016", "unhappy (rule interaction)"},
		{"B09", 1, 2, "O", "P", 0, "FR-016", "unhappy (rule interaction)"},
		{"B10", 1, 2, "", "O", 1, "FR-016", "happy"},
		{"B11", 1, 2, "P", "U", 0, "FR-016", "unhappy (rule interaction)"},
		{"B12", 3, 3, "O O", "P", 3, "FR-015", "happy"},
		{"B13", 3, 3, "O O P", "O", 0, "FR-015", "unhappy (boundary)"},
		{"B14", 4, 4, "U O O O", "O", 0, "FR-015", "unhappy (boundary)"},
		{"B15", 1, 3, "", "P", 1, "FR-016", "happy"},
		{"B16", 1, 3, "P P", "P", 3, "FR-016", "happy"},
		{"B17", 1, 3, "P P P", "P", 0, "FR-016", "unhappy (boundary)"},
		{"B18", 1, 3, "P P", "O", 0, "FR-016", "unhappy (rule interaction)"},
		{"B19", 2, 2, "R", "O", 2, "FR-062", "happy"},
		{"B20", 2, 2, "R O", "O", 0, "FR-062", "unhappy (boundary)"},
		{"B21", 1, 1, "R", "O", 0, "FR-062", "unhappy (boundary)"},
		{"B22", 1, 2, "R", "P", 2, "FR-016", "unhappy (rule interaction)"},
		{"B23", 1, 2, "R", "R", 0, "FR-016", "unhappy (rule interaction)"},
		{"B24", 1, 2, "P", "R", 2, "FR-016", "unhappy (rule interaction)"},
		{"B25", 1, 2, "P R", "P", 0, "FR-016", "unhappy (boundary)"},
		{"B26", 1, 2, "O", "R", 0, "FR-016", "unhappy (rule interaction)"},
		{"B27", 2, 2, "", "R", 1, "FR-062", "happy"},
		{"B28", 2, 3, "", "P", 1, "FR-016", "happy"},
		{"B29", 2, 3, "P P", "P", 3, "FR-016", "happy"},
		{"B30", 2, 3, "P P P", "P", 0, "FR-016", "unhappy (boundary)"},
		{"B31", 2, 3, "P P", "O", 0, "FR-016", "unhappy (rule interaction)"},
		{"B32", 2, 3, "P", "O", 2, "FR-016", "unhappy (rule interaction)"},
		{"B33", 2, 3, "P O", "O", 0, "FR-016", "unhappy (boundary)"},
		{"B34", 2, 3, "O", "P", 2, "FR-016", "unhappy (rule interaction)"},
		{"B35", 2, 3, "O P", "P", 0, "FR-016", "unhappy (boundary)"},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			p := progressOf(t, c.general, c.pact, c.history)
			var out combat.AttackRecordOutcome
			if c.next == "R" {
				out = p.RecordReplacement(replacementOf(t))
			} else {
				out = p.RecordAttack(entryOf(t, c.next).Means())
			}
			if c.want == 0 {
				r, ok := out.(combat.AttackActionAttackRefused)
				if !ok || r.Reason() != combat.AttackRefusalLimitReached || !r.Progress().Equal(p) || len(r.Events()) != 0 {
					t.Fatalf("%s: outcome = %#v, want refused with the progress unchanged [%s]", c.row, out, c.requirement)
				}
				return
			}
			next, events := recorded(t, out)
			if got := len(next.Attacks()); got != c.want || len(events) != 1 {
				t.Fatalf("%s: attacks = %d, events = %d, want %d, 1 [%s]", c.row, got, len(events), c.want, c.requirement)
			}
			if c.next == "R" {
				ev, ok := events[0].(combat.AttackReplacedInAttackAction)
				if !ok || ev.Number() != c.want || ev.Replacement() != replacementOf(t) {
					t.Errorf("%s: event = %#v", c.row, events[0])
				}
				combattest.CheckEvent(t, ev, "attack_replaced_in_attack_action", gid("hero"))
				return
			}
			ev, ok := events[0].(combat.AttackMadeInAttackAction)
			if !ok || ev.Number() != c.want || ev.Means() != entryOf(t, c.next).Means() {
				t.Errorf("%s: event = %#v", c.row, events[0])
			}
			combattest.CheckEvent(t, ev, "attack_made_in_attack_action", gid("hero"))
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-014 - a started progress has no attack, no slot used and AttacksRemaining equal to PactWeaponAttacks, for each Table A allowance.
// Path: happy.
func TestStartAttackActionProgress(t *testing.T) {
	t.Parallel()
	for _, a := range [][2]int{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {1, 2}, {1, 3}, {2, 3}} {
		p, err := combat.StartAttackActionProgress(gid("hero"), allowanceOf(t, a[0], a[1]))
		if err != nil || len(p.Attacks()) != 0 || len(p.UsedWeaponChangeSlots()) != 0 || p.AttacksRemaining() != a[1] || !p.Valid() || p.Allowance() != allowanceOf(t, a[0], a[1]) {
			t.Errorf("Start(%v) = %v, %v, remaining %d", a, p, err, p.AttacksRemaining())
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-014, FR-019 - a blank attacker or an invalid allowance is a validation error, and the zero progress is invalid and refuses everything.
// Path: unhappy (invalid input).
func TestStartAttackActionProgressRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	if _, err := combat.StartAttackActionProgress("", combat.BaselineAttackAllowance()); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("blank attacker: %v", err)
	}
	if _, err := combat.StartAttackActionProgress(gid("hero"), combat.AttackAllowance{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero allowance: %v", err)
	}
	var zero combat.AttackActionProgress
	if zero.Valid() {
		t.Error("the zero progress is valid")
	}
	if _, ok := zero.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused); !ok {
		t.Error("the zero progress records an attack")
	}
	if r, ok := zero.HandleWeapon(combat.WeaponChangeDrawFromSheath, combat.WeaponChangeBeforeAttack).(combat.AttackActionWeaponChangeRefused); !ok || r.Reason() != combat.WeaponChangeRefusalNoAttackRemaining {
		t.Errorf("the zero progress records a weapon change: %v", r)
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-001, FR-062 - a means or replacement that is not valid is refused, not counted and not panicked on.
// Path: unhappy (invalid input).
func TestRecordInvalidAttackIsRefused(t *testing.T) {
	t.Parallel()
	p := progressOf(t, 2, 2, "")
	for _, out := range []combat.AttackRecordOutcome{p.RecordAttack(0), p.RecordAttack(9), p.RecordReplacement(combat.AttackReplacement{})} {
		if r, ok := out.(combat.AttackActionAttackRefused); !ok || r.Reason() != combat.AttackRefusalLimitReached || !r.Progress().Equal(p) {
			t.Errorf("outcome = %#v, want a refusal", out)
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-018 - AttacksRemaining is the pact limit minus the attacks for a history of pact attacks and replacements (empty included) and the general limit minus them otherwise; never negative.
// Path: per case.
func TestAttacksRemaining(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		general, pact int
		history       string
		want          int
	}{
		{"empty (2,3)", 2, 3, "", 3},
		{"after P (2,3)", 2, 3, "P", 2},
		{"after P P (2,3)", 2, 3, "P P", 1},
		{"after P P P (2,3)", 2, 3, "P P P", 0},
		{"after P O (2,3)", 2, 3, "P O", 0},
		{"after O (2,3)", 2, 3, "O", 1},
		{"after R (1,2)", 1, 2, "R", 1},
		{"after P (1,2)", 1, 2, "P", 1},
		{"after O (1,2)", 1, 2, "O", 0},
		{"after U (1,1)", 1, 1, "U", 0},
		{"after O O (3,3)", 3, 3, "O O", 1},
		{"after P P (1,3)", 1, 3, "P P", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := progressOf(t, c.general, c.pact, c.history).AttacksRemaining(); got != c.want {
				t.Errorf("AttacksRemaining = %d, want %d [FR-018]", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-019, FR-064 - the constructor enforces FR-016 on every prefix and validates the slots: (1,3) with R R P is an error and R P P builds; a history longer than the limit is an error and one exactly at it builds; slot 0, a repeat, made + 2 and made + 1 with nothing remaining are errors.
// Path: per case.
func TestNewAttackActionProgress(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		general, pact           int
		history                 string
		slots                   []int
		ok                      bool
	}{
		{"R R P at (1,3)", "FR-019", "unhappy (invalid input)", 1, 3, "R R P", nil, false},
		{"R P P at (1,3)", "FR-019", "happy", 1, 3, "R P P", nil, true},
		{"one past the limit", "FR-019", "unhappy (boundary)", 2, 2, "O O O", nil, false},
		{"exactly the limit", "FR-019", "unhappy (boundary)", 2, 2, "O U", nil, true},
		{"slot 0", "FR-064", "unhappy (boundary)", 2, 2, "O", []int{0}, false},
		{"negative slot", "FR-064", "unhappy (invalid input)", 2, 2, "O", []int{-1}, false},
		{"duplicate slot", "FR-064", "unhappy (invalid input)", 2, 2, "O", []int{1, 1}, false},
		{"made + 2", "FR-064", "unhappy (boundary)", 2, 2, "O", []int{3}, false},
		{"slot 1 with one attack", "FR-064", "happy", 2, 2, "O", []int{1}, true},
		{"made + 1 with one remaining", "FR-064", "happy", 2, 2, "O", []int{2}, true},
		{"made + 1 with none remaining", "FR-064", "unhappy (boundary)", 2, 2, "O U", []int{3}, false},
		{"slot 1 with none made", "FR-064", "happy", 2, 2, "", []int{1}, true},
		{"slot 2 with none made", "FR-064", "unhappy (boundary)", 2, 2, "", []int{2}, false},
		{"slot 2 after a full action", "FR-064", "happy", 2, 2, "O U", []int{2}, true},
		{"slots 1 and 2 after a full action", "FR-064", "happy", 2, 2, "O U", []int{1, 2}, true},
		{"largest slot", "FR-064", "unhappy (boundary)", 1, 1, "", []int{math.MaxInt}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p, err := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, c.general, c.pact), entriesOf(t, c.history), c.slots)
			if (err == nil) != c.ok || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Fatalf("NewAttackActionProgress = %v [%s]", err, c.requirement)
			}
			if c.ok && !slices.Equal(p.UsedWeaponChangeSlots(), c.slots) && len(c.slots) > 0 {
				t.Errorf("slots = %v, want %v", p.UsedWeaponChangeSlots(), c.slots)
			}
		})
	}
	if _, err := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 2, 2), []combat.AttackEntry{{}}, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an invalid entry: %v", err)
	}
	if _, err := combat.NewAttackActionProgress("", allowanceOf(t, 2, 2), nil, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a blank attacker: %v", err)
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-019, FR-064 - the constructor never panics for any history and slots built from bytes, and a progress it builds never holds more attacks than the limit allows.
// Path: unhappy (invalid input).
func FuzzNewAttackActionProgress(f *testing.F) {
	f.Add(1, 3, "RRP", []byte{1, 2})
	f.Add(2, 3, "PPP", []byte{0})
	f.Add(1, 1, "", []byte{1})
	f.Add(0, 0, "OU", []byte{255})
	f.Fuzz(func(t *testing.T, general, pact int, history string, slots []byte) {
		a, err := combat.NewAttackAllowance(general, pact)
		if err != nil {
			return
		}
		var entries []combat.AttackEntry
		for _, r := range history {
			entries = append(entries, entryOf(t, string("POUR"[int(r)&3])))
		}
		var ints []int
		for _, s := range slots {
			ints = append(ints, int(s)-1)
		}
		p, err := combat.NewAttackActionProgress(gid("hero"), a, entries, ints)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if len(p.Attacks()) > a.PactWeaponAttacks() || p.AttacksRemaining() < 0 || !p.Valid() {
			t.Fatalf("built %d attacks against %d, remaining %d", len(p.Attacks()), a.PactWeaponAttacks(), p.AttacksRemaining())
		}
	})
}

// Requirement: doc/specs/attack-and-help-actions.md FR-020, FR-021, FR-077 - the progress is immutable: IsFor names the attacker only; an operation leaves the original equal to what it was; a recorded result differs and a refusal equals; accessors return copies; equality follows every field.
// Path: unhappy (immutability).
func TestAttackActionProgressIsImmutableAndComparable(t *testing.T) {
	t.Parallel()
	p := progressOf(t, 2, 2, "")
	control := progressOf(t, 2, 2, "")
	if p.IsFor(gid("aaa")) {
		t.Error("IsFor must be false for an id that sorts before the attacker [FR-077]")
	}
	if !p.IsFor(gid("hero")) || p.IsFor(gid("villain")) || p.IsFor("") {
		t.Error("IsFor must be true for the attacker only [FR-077]")
	}
	next, _ := recorded(t, p.RecordAttack(combat.AttackMeansOtherWeapon))
	if !p.Equal(control) || next.Equal(p) || !next.IsFor(gid("hero")) {
		t.Error("RecordAttack changed the progress it was called on, or the result equals it [FR-020]")
	}
	changed, _ := changeRecorded(t, next.HandleWeapon(combat.WeaponChangeStow, combat.WeaponChangeAfterAttack))
	if changed.Equal(next) || !next.Equal(progressOf(t, 2, 2, "O")) {
		t.Error("HandleWeapon changed the progress it was called on, or the result equals it")
	}
	refused := next.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRecorded).Progress().RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused)
	if !refused.Progress().Equal(progressOf(t, 2, 2, "O O")) {
		t.Error("a refusal changed the progress")
	}
	attacks := next.Attacks()
	attacks[0] = combat.AttackEntry{}
	slots := changed.UsedWeaponChangeSlots()
	slots[0] = 99
	if !next.Equal(progressOf(t, 2, 2, "O")) || !slices.Equal(changed.UsedWeaponChangeSlots(), []int{1}) {
		t.Error("a returned slice is not a copy [FR-020]")
	}
	other, _ := combat.StartAttackActionProgress(gid("other"), allowanceOf(t, 2, 2))
	diff := []combat.AttackActionProgress{progressOf(t, 2, 3, ""), progressOf(t, 1, 1, ""), other, progressOf(t, 2, 2, "O"), progressOf(t, 2, 2, "U")}
	for i, d := range diff {
		if d.Equal(p) || p.Equal(d) {
			t.Errorf("progress %d equals the empty (2,2) progress [FR-021]", i)
		}
	}
	a, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 2, 2), entriesOf(t, "O"), []int{1})
	b, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 2, 2), entriesOf(t, "O"), nil)
	if a.Equal(b) || !a.Equal(a) {
		t.Error("slots must take part in equality")
	}
	if two, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 2, 2), nil, []int{1}); two.Equal(progressOf(t, 2, 2, "")) {
		t.Error("a used slot must differ from none")
	}
	x, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 3, 3), entriesOf(t, "O"), []int{1, 2})
	y, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 3, 3), entriesOf(t, "O"), []int{2, 1})
	z, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 3, 3), entriesOf(t, "O"), []int{1})
	if !x.Equal(y) || x.Equal(z) || z.Equal(x) {
		t.Error("slots compare as a set")
	}
	w, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 3, 3), entriesOf(t, "O"), []int{1, 2})
	v, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 3, 3), entriesOf(t, "O"), []int{1, 2})
	if !w.Equal(v) {
		t.Error("equal fields must compare equal")
	}
	third, _ := combat.NewAttackActionProgress(gid("hero"), allowanceOf(t, 3, 3), entriesOf(t, "O"), []int{2})
	if z.Equal(third) {
		t.Error("slot 1 and slot 2 must differ")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-023 - each Attack action has its own progress: a second one for the same combatant starts with no attack and no slot used and is unaffected by the first.
// Path: unhappy (state error).
func TestSecondAttackActionHasItsOwnProgress(t *testing.T) {
	t.Parallel()
	first := progressOf(t, 2, 2, "")
	spent, _ := recorded(t, first.RecordAttack(combat.AttackMeansOtherWeapon))
	spent, _ = changeRecorded(t, spent.HandleWeapon(combat.WeaponChangeStow, combat.WeaponChangeAfterAttack))
	second, err := combat.StartAttackActionProgress(gid("hero"), allowanceOf(t, 2, 2))
	if err != nil || len(second.Attacks()) != 0 || len(second.UsedWeaponChangeSlots()) != 0 || second.AttacksRemaining() != 2 {
		t.Fatalf("second progress = %v, %v", second, err)
	}
	if len(spent.Attacks()) != 1 || !slices.Equal(spent.UsedWeaponChangeSlots(), []int{1}) {
		t.Error("the first progress was affected by the second")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-025 to FR-029, Table C - each row records a weapon change in the slot of an attack (before: made + 1, after: made) or refuses it with the reason, the progress unchanged. B is a change before the next attack, A a change after the last, a letter P or O an attack.
// Path: per case.
func TestHandleWeaponTableC(t *testing.T) {
	t.Parallel()
	const none = combat.WeaponChangeRefusal(0)
	cases := []struct {
		row           string
		general, pact int
		seq           string
		timing        string // the request, B or A
		slot          int    // recorded slot, or 0 for a refusal
		refusal       combat.WeaponChangeRefusal
		path          string
	}{
		{"C01", 1, 1, "", "B", 1, none, "happy"},
		{"C03", 1, 1, "", "A", 0, combat.WeaponChangeRefusalNoAttackMade, "unhappy (state error)"},
		{"C04", 1, 1, "O", "A", 1, none, "happy"},
		{"C05", 1, 1, "O", "B", 0, combat.WeaponChangeRefusalNoAttackRemaining, "unhappy (boundary)"},
		{"C06", 1, 1, "B", "B", 0, combat.WeaponChangeRefusalAlreadyUsedForThisAttack, "unhappy (boundary)"},
		{"C07", 1, 1, "B O", "A", 0, combat.WeaponChangeRefusalAlreadyUsedForThisAttack, "unhappy (boundary)"},
		{"C08", 1, 1, "O A", "A", 0, combat.WeaponChangeRefusalAlreadyUsedForThisAttack, "unhappy (boundary)"},
		{"C09", 2, 2, "O A", "B", 2, none, "happy"},
		{"C10", 2, 2, "O A B", "B", 0, combat.WeaponChangeRefusalAlreadyUsedForThisAttack, "unhappy (boundary)"},
		{"C11", 2, 2, "O O", "A", 2, none, "happy"},
		{"C12", 2, 2, "O O A", "A", 0, combat.WeaponChangeRefusalAlreadyUsedForThisAttack, "unhappy (boundary)"},
		{"C13", 2, 2, "B O O", "A", 2, none, "happy"},
		{"C14", 2, 2, "O O", "B", 0, combat.WeaponChangeRefusalNoAttackRemaining, "unhappy (boundary)"},
		{"C15", 1, 2, "P", "B", 2, none, "happy"},
		{"C16", 1, 2, "O", "B", 0, combat.WeaponChangeRefusalNoAttackRemaining, "unhappy (rule interaction)"},
	}
	for _, c := range cases {
		t.Run(c.row, func(t *testing.T) {
			t.Parallel()
			p, err := combat.StartAttackActionProgress(gid("hero"), allowanceOf(t, c.general, c.pact))
			if err != nil {
				t.Fatal(err)
			}
			for _, tok := range strings.Fields(c.seq) {
				switch tok {
				case "B":
					p, _ = changeRecorded(t, p.HandleWeapon(combat.WeaponChangeDrawFromSheath, combat.WeaponChangeBeforeAttack))
				case "A":
					p, _ = changeRecorded(t, p.HandleWeapon(combat.WeaponChangeStow, combat.WeaponChangeAfterAttack))
				default:
					p, _ = recorded(t, p.RecordAttack(entryOf(t, tok).Means()))
				}
			}
			change, timing := combat.WeaponChangePickUp, combat.WeaponChangeBeforeAttack
			if c.timing == "A" {
				change, timing = combat.WeaponChangeDrop, combat.WeaponChangeAfterAttack
			}
			out := p.HandleWeapon(change, timing)
			if c.slot == 0 {
				r, ok := out.(combat.AttackActionWeaponChangeRefused)
				if !ok || r.Reason() != c.refusal || !r.Progress().Equal(p) || len(r.Events()) != 0 {
					t.Fatalf("%s: outcome = %#v, want refused %v", c.row, out, c.refusal)
				}
				return
			}
			next, events := changeRecorded(t, out)
			if !slices.Contains(next.UsedWeaponChangeSlots(), c.slot) || len(next.UsedWeaponChangeSlots()) != len(p.UsedWeaponChangeSlots())+1 || len(events) != 1 {
				t.Fatalf("%s: slots = %v, want slot %d added", c.row, next.UsedWeaponChangeSlots(), c.slot)
			}
			ev := events[0].(combat.WeaponChangedInAttackAction)
			if ev.Slot() != c.slot || ev.Change() != change || ev.Timing() != timing || ev.ChangeKind() != change.Change() {
				t.Errorf("%s: event = %#v", c.row, ev)
			}
			combattest.CheckEvent(t, ev, "weapon_changed_in_attack_action", gid("hero"))
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-025 - an invalid change or timing is refused (fail closed), never recorded and never a panic.
// Path: unhappy (invalid input).
func TestHandleWeaponRejectsInvalidChangeOrTiming(t *testing.T) {
	t.Parallel()
	p := progressOf(t, 2, 2, "O")
	for _, out := range []combat.WeaponChangeOutcome{
		p.HandleWeapon(0, combat.WeaponChangeAfterAttack), p.HandleWeapon(combat.WeaponChangeDrop, 0), p.HandleWeapon(99, 99),
	} {
		r, ok := out.(combat.AttackActionWeaponChangeRefused)
		if !ok || !r.Progress().Equal(p) {
			t.Errorf("outcome = %#v, want a refusal with the progress unchanged", out)
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-030 - an equipped weapon restricts nothing: after a draw before attack 1, RecordAttack accepts every means the limit allows, and the progress records no weapon held.
// Path: happy.
func TestWeaponChangeDoesNotRestrictTheNextAttack(t *testing.T) {
	t.Parallel()
	for _, m := range []combat.AttackMeans{combat.AttackMeansUnarmedStrike, combat.AttackMeansOtherWeapon, combat.AttackMeansPactWeapon} {
		p := progressOf(t, 1, 2, "")
		p, _ = changeRecorded(t, p.HandleWeapon(combat.WeaponChangeDrawFromSheath, combat.WeaponChangeBeforeAttack))
		if _, ok := p.RecordAttack(m).(combat.AttackActionAttackRecorded); !ok {
			t.Errorf("RecordAttack(%v) refused after a draw", m)
		}
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-032, FR-033 - IsBetweenAttacks is true exactly when an attack is recorded and attacks remain: false at 0 attacks, true at 1 of 2, false at 2 of 2, always false with one attack allowed, false after an other-weapon attack at (1,2).
// Path: per case.
func TestIsBetweenAttacks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		general, pact int
		history       string
		want          bool
		path          string
	}{
		{"(2,2) no attack", 2, 2, "", false, "unhappy (boundary)"},
		{"(2,2) one attack", 2, 2, "O", true, "happy"},
		{"(2,2) two attacks", 2, 2, "O O", false, "unhappy (boundary)"},
		{"(1,1) no attack", 1, 1, "", false, "unhappy (boundary)"},
		{"(1,1) one attack", 1, 1, "O", false, "unhappy (boundary)"},
		{"(1,2) other weapon", 1, 2, "O", false, "unhappy (boundary)"},
		{"(1,2) pact weapon", 1, 2, "P", true, "happy"},
		{"(1,2) replacement only", 1, 2, "R", true, "happy"},
		{"(3,3) one of three", 3, 3, "O", true, "happy"},
		{"(4,4) two of four", 4, 4, "O U", true, "happy"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := progressOf(t, c.general, c.pact, c.history).IsBetweenAttacks(); got != c.want {
				t.Errorf("IsBetweenAttacks = %v, want %v", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-034, FR-063 - the queries are pure: repeating them leaves the progress equal, the next operations give the same result as on a control that was never queried, and AttacksRemaining, IsBetweenAttacks and the slots are the same before and after a Nick attack is recorded elsewhere (the progress has no operation for it).
// Path: unhappy (rule interaction).
func TestQueriesAreSideEffectFree(t *testing.T) {
	t.Parallel()
	p, control := progressOf(t, 2, 2, "O"), progressOf(t, 2, 2, "O")
	for range 3 {
		_ = p.IsBetweenAttacks()
		_ = p.AttacksRemaining()
	}
	a, _ := recorded(t, p.RecordAttack(combat.AttackMeansUnarmedStrike))
	b, _ := recorded(t, control.RecordAttack(combat.AttackMeansUnarmedStrike))
	c, _ := changeRecorded(t, p.HandleWeapon(combat.WeaponChangeStow, combat.WeaponChangeAfterAttack))
	d, _ := changeRecorded(t, control.HandleWeapon(combat.WeaponChangeStow, combat.WeaponChangeAfterAttack))
	if !p.Equal(control) || !a.Equal(b) || !c.Equal(d) {
		t.Error("a query changed what the next operation returns")
	}
}

// Requirement: doc/specs/attack-and-help-actions.md FR-062 - the Attack action example: Fighter 5 / Barbarian 5 has the allowance (2,2); the host draws a weapon before attack 1, attacks, is between attacks, attacks again and the allowance is spent.
// Path: happy.
func TestWorkedExample(t *testing.T) {
	t.Parallel()
	p := progressOf(t, 2, 2, "")
	p, _ = changeRecorded(t, p.HandleWeapon(combat.WeaponChangeDrawFromSheath, combat.WeaponChangeBeforeAttack))
	p, _ = recorded(t, p.RecordAttack(combat.AttackMeansOtherWeapon))
	if !p.IsBetweenAttacks() {
		t.Error("not between attacks after attack 1 of 2")
	}
	p, _ = recorded(t, p.RecordAttack(combat.AttackMeansOtherWeapon))
	if p.IsBetweenAttacks() || p.AttacksRemaining() != 0 {
		t.Error("allowance not spent")
	}
	if _, ok := p.RecordAttack(combat.AttackMeansOtherWeapon).(combat.AttackActionAttackRefused); !ok {
		t.Error("a third attack was recorded")
	}
}
