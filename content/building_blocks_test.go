package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// wantBlockError asserts err is a *BlockError for block with problem that
// wraps ErrInvalidInput.
func wantBlockError(t *testing.T, err error, block string, problem content.BlockProblem) {
	t.Helper()
	var be *content.BlockError
	if !errors.As(err, &be) || be.Block != block || be.Problem != problem || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("error = %v, want *BlockError{%s, %v} wrapping ErrInvalidInput", err, block, problem)
		return
	}
	if !errors.Is(err, &content.BlockError{Block: block, Problem: problem}) || errors.Is(err, &content.BlockError{Block: block, Problem: problem + 1}) {
		t.Errorf("errors.Is on %v does not match by Block and Problem", err)
	}
}

// Requirement: schema FR-116 (SRD 5.2.1, "Parts of a Species: Special Traits",
// p. 84) - NewNamedText keeps both parts verbatim and rejects an invalid name
// (FR-027) or blank text (FR-028), name first.
// Path: per case.
func TestNamedText(t *testing.T) {
	t.Parallel()
	ok, err := content.NewNamedText("Darkvision", " Line one.\nIt’s kept. ")
	if err != nil || ok.Name() != "Darkvision" || ok.Text() != " Line one.\nIt’s kept. " {
		t.Errorf("NewNamedText valid = (%+v, %v) [happy]", ok, err)
	}
	if ok != content.MustNamedText("Darkvision", " Line one.\nIt’s kept. ") {
		t.Error("equal NamedTexts differ under == [happy]")
	}
	cases := []struct {
		name, nameArg, text string
		want                content.BlockProblem
	}{
		{"empty name", "", "t", content.BlockInvalidName},
		{"leading space", " n", "t", content.BlockInvalidName},
		{"trailing newline", "n\n", "t", content.BlockInvalidName},
		{"blank text", "n", " \t\n", content.BlockBlankText},
		{"empty text", "n", "", content.BlockBlankText},
		{"both bad: name first", "", "", content.BlockInvalidName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := content.NewNamedText(c.nameArg, c.text)
			wantBlockError(t, err, "NamedText", c.want)
			if got != (content.NamedText{}) {
				t.Errorf("rejected NamedText = %+v, want zero [unhappy (invalid input)]", got)
			}
			assertMustPanics(t, func() { content.MustNamedText(c.nameArg, c.text) })
		})
	}
}

func assertMustPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if err, isErr := r.(error); !isErr || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("Must panicked with %v, want an error wrapping ErrInvalidInput", r)
		}
	}()
	f()
}

// Requirement: schema FR-118 (SRD 5.2.1, "Parts of a Feat: Prerequisite", p. 87) -
// level 1 to 20; ability score minimum 1 to 30, a non-empty set of valid,
// distinct abilities; feature name per FR-027.
// Path: per case.
func TestPrerequisites(t *testing.T) {
	t.Parallel()
	for _, m := range []int{1, 4, 20} {
		if p, err := content.NewLevelPrerequisite(m); err != nil || p.Minimum() != m {
			t.Errorf("NewLevelPrerequisite(%d) = (%+v, %v) [unhappy (boundary)]", m, p, err)
		}
	}
	for _, m := range []int{-1, 0, 21} {
		_, err := content.NewLevelPrerequisite(m)
		wantBlockError(t, err, "LevelPrerequisite", content.BlockOutOfRange)
	}

	str, dex := ability.Strength, ability.Dexterity
	for _, m := range []int{1, 13, 30} {
		p, err := content.NewAbilityScorePrerequisite([]ability.Ability{str, dex}, m)
		if err != nil || p.Minimum() != m || len(p.Abilities()) != 2 {
			t.Errorf("NewAbilityScorePrerequisite(min %d) = (%+v, %v) [unhappy (boundary)]", m, p, err)
		}
	}
	bad := []struct {
		name string
		in   []ability.Ability
		min  int
		want content.BlockProblem
	}{
		{"nil abilities", nil, 13, content.BlockEmpty},
		{"empty abilities", []ability.Ability{}, 13, content.BlockEmpty},
		{"invalid ability", []ability.Ability{str, 0}, 13, content.BlockInvalidMember},
		{"seventh ability", []ability.Ability{7}, 13, content.BlockInvalidMember},
		{"repeat", []ability.Ability{str, dex, str}, 13, content.BlockDuplicate},
		{"minimum 0", []ability.Ability{str}, 0, content.BlockOutOfRange},
		{"minimum 31", []ability.Ability{str}, 31, content.BlockOutOfRange},
		{"empty beats bad minimum", nil, 0, content.BlockEmpty},
		{"member beats bad minimum", []ability.Ability{0}, 0, content.BlockInvalidMember},
	}
	for _, c := range bad {
		_, err := content.NewAbilityScorePrerequisite(c.in, c.min)
		wantBlockError(t, err, "AbilityScorePrerequisite", c.want)
	}

	if p, err := content.NewFeaturePrerequisite("Spellcasting"); err != nil || p.Name() != "Spellcasting" {
		t.Errorf("NewFeaturePrerequisite valid = (%+v, %v)", p, err)
	}
	for _, n := range []string{"", " x", "x "} {
		_, err := content.NewFeaturePrerequisite(n)
		wantBlockError(t, err, "FeaturePrerequisite", content.BlockInvalidName)
	}
}

// Requirement: schema FR-118, testing.md section 4 (immutability) - the
// constructor and the accessor copy the abilities; Equal compares abilities in
// order and the minimum.
// Path: unhappy (rule interaction).
func TestAbilityScorePrerequisiteCopiesAndEquals(t *testing.T) {
	t.Parallel()
	in := []ability.Ability{ability.Strength, ability.Dexterity}
	p, err := content.NewAbilityScorePrerequisite(in, 13)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = ability.Wisdom
	out := p.Abilities()
	out[1] = ability.Wisdom
	if got := p.Abilities(); got[0] != ability.Strength || got[1] != ability.Dexterity {
		t.Errorf("Abilities() = %v after the input and a result were mutated, want [strength dexterity]", got)
	}
	same := mustAbilityPrereq(t, []ability.Ability{ability.Strength, ability.Dexterity}, 13)
	other := mustAbilityPrereq(t, []ability.Ability{ability.Dexterity, ability.Strength}, 13)
	higher := mustAbilityPrereq(t, []ability.Ability{ability.Strength, ability.Dexterity}, 14)
	if !p.Equal(same) || p.Equal(other) || p.Equal(higher) || higher.Equal(p) {
		t.Error("Equal: want equal to same, unequal to reordered and to a higher minimum")
	}
}

// Requirement: schema FR-119 (SRD 5.2.1, "Parts of a Background: Tool
// Proficiency", p. 83) - specific name per FR-027; choice category non-blank.
// Path: per case.
func TestToolProficiencies(t *testing.T) {
	t.Parallel()
	if s, err := content.NewSpecificToolProficiency("Thieves’ Tools"); err != nil || s.Name() != "Thieves’ Tools" {
		t.Errorf("NewSpecificToolProficiency valid = (%+v, %v) [happy]", s, err)
	}
	for _, n := range []string{"", " x", "x\t"} {
		_, err := content.NewSpecificToolProficiency(n)
		wantBlockError(t, err, "SpecificToolProficiency", content.BlockInvalidName)
	}
	if c, err := content.NewChoiceToolProficiency("Gaming Set"); err != nil || c.Category() != "Gaming Set" {
		t.Errorf("NewChoiceToolProficiency valid = (%+v, %v) [happy]", c, err)
	}
	for _, n := range []string{"", "  "} {
		_, err := content.NewChoiceToolProficiency(n)
		wantBlockError(t, err, "ChoiceToolProficiency", content.BlockBlankText)
	}
}

// Requirement: schema FR-118, FR-119 - closed families switched exhaustively
// (the doc comments name every variant; archtest A7 checks it).
// Path: happy.
func TestFamiliesAreSwitchable(t *testing.T) {
	t.Parallel()
	level, _ := content.NewLevelPrerequisite(4)
	feature, _ := content.NewFeaturePrerequisite("Spellcasting")
	abil, _ := content.NewAbilityScorePrerequisite([]ability.Ability{ability.Wisdom}, 13)
	for want, p := range map[string]content.Prerequisite{"level": level, "feature": feature, "abilityScore": abil} {
		var got string
		switch p.(type) {
		case content.LevelPrerequisite:
			got = "level"
		case content.AbilityScorePrerequisite:
			got = "abilityScore"
		case content.FeaturePrerequisite:
			got = "feature"
		}
		if got != want {
			t.Errorf("%T classified as %q, want %q", p, got, want)
		}
	}
	spec, _ := content.NewSpecificToolProficiency("Smith’s Tools")
	choice, _ := content.NewChoiceToolProficiency("Gaming Set")
	for want, p := range map[string]content.ToolProficiency{"specific": spec, "choice": choice} {
		var got string
		switch p.(type) {
		case content.SpecificToolProficiency:
			got = "specific"
		case content.ChoiceToolProficiency:
			got = "choice"
		}
		if got != want {
			t.Errorf("%T classified as %q, want %q", p, got, want)
		}
	}
}

// Requirement: schema FR-114 style check of the error strings - every
// BlockProblem has a distinct name and an undefined one is "unknown".
// Path: unhappy (boundary).
func TestBlockProblemString(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for p := content.BlockInvalidName; p <= content.BlockConstraint; p++ {
		s := p.String()
		if s == "unknown" || seen[s] {
			t.Errorf("BlockProblem(%d).String() = %q, want a distinct name", int(p), s)
		}
		seen[s] = true
	}
	for p, want := range map[content.BlockProblem]string{
		content.BlockInvalidName: "invalidName", content.BlockBlankText: "blankText", content.BlockEmpty: "empty",
		content.BlockOutOfRange: "outOfRange", content.BlockInvalidMember: "invalidMember",
		content.BlockDuplicate: "duplicate", content.BlockRaggedRow: "raggedRow",
		content.BlockInvalidCharacter: "invalidCharacter", content.BlockMissing: "missing", content.BlockConstraint: "constraint",
	} {
		if p.String() != want {
			t.Errorf("BlockProblem(%d).String() = %q, want %q", int(p), p.String(), want)
		}
	}
	if content.BlockProblem(0).String() != "unknown" || (content.BlockConstraint+1).String() != "unknown" {
		t.Error("an undefined BlockProblem is not unknown")
	}
	if (&content.BlockError{Block: "X", Problem: content.BlockEmpty}).Error() == "" {
		t.Error("BlockError.Error() is empty")
	}
	// Requirement: schema FR-151 - the constructors never panic; Is with a nil target or receiver is false.
	// Path: unhappy (invalid input).
	_, err := content.NewNamedText("", "x")
	var nilTarget *content.BlockError
	if errors.Is(err, nilTarget) || nilTarget.Is(err) {
		t.Error("errors.Is against a nil *BlockError is true, want false")
	}
}

func mustAbilityPrereq(t *testing.T, abilities []ability.Ability, minimum int) content.AbilityScorePrerequisite {
	t.Helper()
	p, err := content.NewAbilityScorePrerequisite(abilities, minimum)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
