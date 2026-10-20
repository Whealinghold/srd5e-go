package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/plans/plan-phase-2.md P14 and doc/specs/content-schema-and-decoder.md FR-156 and doc/specs/content-schema-and-decoder.md FR-027, FR-028, FR-151 - a
// name with a rejected character is BlockInvalidName; a text is BlockBlankText
// when blank (checked first) and BlockInvalidCharacter when it holds invalid
// UTF-8, a control or a bidirectional character.
// Path: per case.
func TestBlockConstructorsApplyTheCharacterRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  func() error
		want content.BlockProblem
	}{
		{"name with a control character", func() error { _, err := content.NewNamedText("a\x07", "t"); return err }, content.BlockInvalidName},
		{"name with a line feed", func() error { _, err := content.NewTextEntry("a\nb", "t"); return err }, content.BlockInvalidName},
		{"name with a bidi override", func() error { _, err := content.NewFeaturePrerequisite("a\u202eb"); return err }, content.BlockInvalidName},
		{"name with invalid UTF-8", func() error { _, err := content.NewSpecificToolProficiency("a\xff"); return err }, content.BlockInvalidName},
		{"text with a tab", func() error { _, err := content.NewTextParagraph("a\tb"); return err }, content.BlockInvalidCharacter},
		{"text with a bidi isolate", func() error { _, err := content.NewNamedText("n", "a\u2066b"); return err }, content.BlockInvalidCharacter},
		{"text with invalid UTF-8", func() error { _, err := content.NewTextBulletList([]string{"ok", "a\xff"}); return err }, content.BlockInvalidCharacter},
		{"table cell with a NUL", func() error { _, err := content.NewTextTable([]string{"h"}, [][]string{{"a\x00"}}); return err }, content.BlockInvalidCharacter},
		{"blank text outranks a bad character", func() error { _, err := content.NewTextParagraph(" \u202e "); return err }, content.BlockInvalidCharacter},
		{"blank text", func() error { _, err := content.NewTextParagraph(" \n "); return err }, content.BlockBlankText},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := c.err()
			var be *content.BlockError
			if !errors.As(err, &be) || be.Problem != c.want || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("error = %v, want a *BlockError with problem %v wrapping ErrInvalidInput", err, c.want)
			}
		})
	}
	if _, err := content.NewTextParagraph("line one\nline two"); err != nil {
		t.Errorf("a text keeps its line feeds: %v", err)
	}
}
