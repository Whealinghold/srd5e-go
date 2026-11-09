package character_test

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
)

// srdTrinkets reads the independent transcription of the Trinkets table, SRD 5.2.1
// pp. 26-27, from testdata/srd/trinkets.txt: the printed number (00 for 100) and
// the text of each of the 100 entries, in order.
func srdTrinkets(tb testing.TB) (numbers []string, texts []string) {
	tb.Helper()
	f, err := os.Open("testdata/srd/trinkets.txt")
	if err != nil {
		tb.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		number, text, ok := strings.Cut(line, " ")
		if !ok {
			tb.Fatalf("transcription line %q has no text", line)
		}
		numbers, texts = append(numbers, number), append(texts, text)
	}
	if err := sc.Err(); err != nil {
		tb.Fatal(err)
	}
	return numbers, texts
}

// srdTrinketTable builds the table from the transcription.
func srdTrinketTable(tb testing.TB) character.TrinketTable {
	tb.Helper()
	_, texts := srdTrinkets(tb)
	entries := make([]character.Trinket, len(texts))
	for i, s := range texts {
		entries[i] = must(character.NewTrinket(s))
	}
	return must(character.NewTrinketTable(entries))
}

func trinkets(n int) []character.Trinket {
	out := make([]character.Trinket, n)
	for i := range out {
		out[i] = must(character.NewTrinket("Entry " + strconv.Itoa(i+1)))
	}
	return out
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-093 - the
// transcription of SRD 5.2.1, "Trinkets", pp. 26-27, holds exactly 100 entries
// numbered 01 to 99 and 00, and a table built from it returns the transcribed text
// for every roll 1 to 100 (the 00 entry for a roll of 100).
// Path: happy.
func TestTrinketTableReturnsTheTranscribedTextForEveryRoll(t *testing.T) {
	t.Parallel()
	numbers, texts := srdTrinkets(t)
	if len(numbers) != 100 {
		t.Fatalf("transcription has %d entries, want 100", len(numbers))
	}
	table := srdTrinketTable(t)
	for roll := 1; roll <= 100; roll++ {
		want := strconv.Itoa(roll)
		if roll < 10 {
			want = "0" + want
		}
		if roll == 100 {
			want = "00"
		}
		if numbers[roll-1] != want {
			t.Fatalf("transcription entry %d is numbered %s, want %s", roll, numbers[roll-1], want)
		}
		got, err := table.EntryFor(roll)
		if err != nil || got.Description() != texts[roll-1] {
			t.Errorf("EntryFor(%d) = %q, %v, want %q", roll, got.Description(), err, texts[roll-1])
		}
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-091, FR-095
// (Table TR) - the rolls 1, 34, 35, 47, 72, 73, 99 and 100 return the printed entry,
// so the page break between 34 and 35 and the 00 entry are covered, and a quoted
// label (93) and an entry that leaves a choice to the reader (47) stay verbatim.
// Path: per case.
func TestTrinketTableEntryForTableTR(t *testing.T) {
	t.Parallel()
	table := srdTrinketTable(t)
	for _, c := range []struct {
		roll int
		want string
	}{
		{1, "A mummified goblin hand"},
		{34, "A rectangular metal device with two tiny metal cups on one end that throws sparks when wet"},
		{35, "A white, sequined glove sized for a human"},
		{47, "A metal can that has no opening but sounds as if it is filled with liquid, sand, spiders, or broken glass (your choice)"},
		{72, "A silver teardrop earring containing a real teardrop"},
		{73, "An eggshell painted with scenes of misery in disturbing detail"},
		{93, "An empty wine bottle bearing a pretty label that says, “The Wizard of Wines Winery, Red Dragon Crush, 331422-W”"},
		{99, "A wooden box with a ceramic bottom that holds a living worm with a head on each end of its body"},
		{100, "A metal urn containing the ashes of a hero"},
	} {
		t.Run(strconv.Itoa(c.roll), func(t *testing.T) {
			t.Parallel()
			got, err := table.EntryFor(c.roll)
			if err != nil || got.Description() != c.want {
				t.Errorf("EntryFor(%d) = %q, %v, want %q", c.roll, got.Description(), err, c.want)
			}
		})
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-091 (Table TR,
// rows 0 and 101) - a roll outside 1 to 100 is an invalid-input error carrying the
// roll, and the zero table refuses every roll.
// Path: unhappy (boundary, invalid input).
func TestTrinketTableEntryForRefusesRollsOutsideTheTable(t *testing.T) {
	t.Parallel()
	table := srdTrinketTable(t)
	for _, roll := range []int{0, 101, -1} {
		_, err := table.EntryFor(roll)
		if !errors.Is(err, &character.TrinketError{Problem: character.TrinketRollOutOfRange, InvalidValue: roll}) || !isInvalid(err) {
			t.Errorf("EntryFor(%d) error = %v, want TrinketRollOutOfRange carrying the roll", roll, err)
		}
	}
	if _, err := (character.TrinketTable{}).EntryFor(50); !errors.Is(err, &character.TrinketError{Problem: character.TrinketTableSize}) {
		t.Errorf("zero table error = %v, want TrinketTableSize", err)
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-090 - the table
// holds exactly 100 entries: 99, 100 and 101 entries give an invalid-input error
// carrying the count (only 100 is accepted), a blank entry is refused with its roll,
// MustTrinketTable panics with that error, and the table copies its entries.
// Path: per case.
func TestNewTrinketTable(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		n    int
		want *character.TrinketError
	}{
		{"99 entries", 99, &character.TrinketError{Problem: character.TrinketTableSize, InvalidValue: 99}},
		{"100 entries", 100, nil},
		{"101 entries", 101, &character.TrinketError{Problem: character.TrinketTableSize, InvalidValue: 101}},
		{"no entries", 0, &character.TrinketError{Problem: character.TrinketTableSize}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			table, err := character.NewTrinketTable(trinkets(c.n))
			if c.want == nil {
				if err != nil || table.IsZero() {
					t.Fatalf("NewTrinketTable = %v, %v, want a table", table, err)
				}
				return
			}
			if !errors.Is(err, c.want) || !isInvalid(err) || !table.IsZero() {
				t.Errorf("NewTrinketTable error = %v, want %v", err, c.want)
			}
		})
	}
	blank := trinkets(100)
	blank[41] = character.Trinket{}
	if _, err := character.NewTrinketTable(blank); !errors.Is(err, &character.TrinketError{Problem: character.TrinketBlankDescription, InvalidValue: 42}) {
		t.Errorf("blank entry error = %v, want TrinketBlankDescription at roll 42", err)
	}
	firstBlank := trinkets(100)
	firstBlank[0] = character.Trinket{}
	if _, err := character.NewTrinketTable(firstBlank); !errors.Is(err, &character.TrinketError{Problem: character.TrinketBlankDescription, InvalidValue: 1}) {
		t.Errorf("blank first entry error = %v, want TrinketBlankDescription at roll 1", err)
	}
	func() {
		defer func() {
			err, _ := recover().(error)
			if !errors.Is(err, &character.TrinketError{Problem: character.TrinketTableSize, InvalidValue: 3}) {
				t.Errorf("MustTrinketTable panic = %v, want the NewTrinketTable error", err)
			}
		}()
		character.MustTrinketTable(trinkets(3))
	}()
	entries := trinkets(100)
	table := character.MustTrinketTable(entries)
	entries[0] = must(character.NewTrinket("changed"))
	if got, _ := table.EntryFor(1); got.Description() != "Entry 1" {
		t.Errorf("table entry 1 = %q after the input slice changed, want it copied", got.Description())
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-092 - a Trinket
// description must be non-blank and is kept verbatim (inner and outer whitespace, quoted
// labels), so a blank one is an invalid-input error.
// Path: per case.
func TestNewTrinket(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"A mummified goblin hand", "  padded  ", "An empty bottle that says, “Red Dragon Crush”"} {
		got, err := character.NewTrinket(s)
		if err != nil || got.Description() != s || got.IsZero() {
			t.Errorf("NewTrinket(%q) = %q, %v, want it verbatim", s, got.Description(), err)
		}
	}
	for _, s := range []string{"", " ", "\t\n"} {
		_, err := character.NewTrinket(s)
		if !errors.Is(err, &character.TrinketError{Problem: character.TrinketBlankDescription}) || !isInvalid(err) {
			t.Errorf("NewTrinket(%q) error = %v, want TrinketBlankDescription", s, err)
		}
	}
	if !(character.Trinket{}).IsZero() {
		t.Error("the zero Trinket must be zero")
	}
}

// Requirement: doc/specs/higher-level-start-equipment-and-trinkets.md FR-089 and FR-123 - the
// table is an immutable value that compares by value, and the problem names are stable.
// Path: unhappy (immutability).
func TestTrinketTableComparesByValueAndErrorsDescribeThemselves(t *testing.T) {
	t.Parallel()
	srd, other := srdTrinketTable(t), character.MustTrinketTable(trinkets(100))
	if srd != srdTrinketTable(t) || other == srd {
		t.Error("tables must compare by value")
	}
	for p, want := range map[character.TrinketProblem]string{
		character.TrinketBlankDescription: "blankDescription", character.TrinketTableSize: "tableSize",
		character.TrinketRollOutOfRange: "rollOutOfRange", 0: "unknown",
	} {
		if p.String() != want {
			t.Errorf("TrinketProblem(%d).String() = %q, want %q", int(p), p.String(), want)
		}
	}
	e := &character.TrinketError{Problem: character.TrinketTableSize, InvalidValue: 3}
	var nilTarget *character.TrinketError
	if e.Error() == "" || errors.Is(e, &character.TrinketError{Problem: character.TrinketTableSize, InvalidValue: 4}) || errors.Is(e, errors.New("x")) || errors.Is(e, nilTarget) {
		t.Error("TrinketError must match only an equal TrinketError and srderr.ErrInvalidInput")
	}
}

// FuzzNewTrinketTable: NewTrinketTable never panics for any number of entries and
// accepts exactly 100.
func FuzzNewTrinketTable(f *testing.F) {
	for _, n := range []int{0, 1, 99, 100, 101, 1000} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, n int) {
		n = n % 2000
		if n < 0 {
			n = -n
		}
		table, err := character.NewTrinketTable(trinkets(n))
		if (err == nil) != (n == 100) || (err == nil) == table.IsZero() {
			t.Fatalf("NewTrinketTable(%d entries) = %v, %v", n, table.IsZero(), err)
		}
		if err != nil && !isInvalid(err) {
			t.Fatalf("error %v does not match ErrInvalidInput", err)
		}
	})
}
