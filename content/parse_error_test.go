package content_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-schema-and-decoder.md - ParseError matches
// srderr.ErrInvalidInput and its rendered text names the vocabulary, not the
// rejected value.
// Path: unhappy (invalid input).
func TestParseErrorRendersVocabulary(t *testing.T) {
	t.Parallel()
	var err error = &content.ParseError{Vocabulary: "DurationUnit", Value: "zzz"}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Error("ParseError does not match ErrInvalidInput")
	}
	if msg := err.Error(); !strings.Contains(msg, "DurationUnit") || strings.Contains(msg, "zzz") {
		t.Errorf("Error() = %q, want it to name the vocabulary and not the value", msg)
	}
}
