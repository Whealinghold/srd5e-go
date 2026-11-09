package character_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/character-creation.md FR-045 and the d12 mapping line (SRD 5.2.1, Standard Languages table, p. 20) - one test
// per roll; 0 and 13 are validation errors.
// Path: per case.
func TestStandardLanguageRollLanguageFor(t *testing.T) {
	t.Parallel()
	want := map[int]content.Language{
		1: content.LanguageCommonSignLanguage, 2: content.LanguageDraconic, 3: content.LanguageDwarvish, 4: content.LanguageDwarvish,
		5: content.LanguageElvish, 6: content.LanguageElvish, 7: content.LanguageGiant, 8: content.LanguageGnomish, 9: content.LanguageGoblin,
		10: content.LanguageHalfling, 11: content.LanguageHalfling, 12: content.LanguageOrc,
	}
	for roll, lang := range want {
		if got, err := (character.StandardLanguageRoll{}).LanguageFor(roll); err != nil || got != lang {
			t.Errorf("LanguageFor(%d) = %v, %v, want %v", roll, got, err, lang)
		}
	}
	for _, roll := range []int{0, 13, -1} {
		if _, err := (character.StandardLanguageRoll{}).LanguageFor(roll); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("LanguageFor(%d) error = %v", roll, err)
		}
	}
}

// Requirement: doc/specs/character-creation.md FR-043, FR-049, FR-053 and the row "WithAdded" - Languages always holds Common, holds each
// language once, WithAdded accepts a Rare language and gives an equal value for a known one, and the zero value is invalid.
// Path: per case.
func TestLanguages(t *testing.T) {
	t.Parallel()
	l, err := character.NewLanguages(content.LanguageDwarvish, content.LanguageElvish, content.LanguageDwarvish)
	if err != nil || l.IsZero() || !reflect.DeepEqual(l.All(), []content.Language{content.LanguageCommon, content.LanguageDwarvish, content.LanguageElvish}) {
		t.Fatalf("NewLanguages = %v, %v", l.All(), err)
	}
	added, err := l.WithAdded(content.LanguageThievesCant)
	if err != nil || len(added.All()) != 4 || !added.Contains(content.LanguageThievesCant) || l.Contains(content.LanguageThievesCant) {
		t.Errorf("WithAdded(Thieves' Cant) = %v, %v", added.All(), err)
	}
	if same, err := l.WithAdded(content.LanguageDwarvish); err != nil || same != l {
		t.Errorf("WithAdded(known) = %v, %v", same.All(), err)
	}
	if only, err := character.NewLanguages(); err != nil || !reflect.DeepEqual(only.All(), []content.Language{content.LanguageCommon}) {
		t.Errorf("NewLanguages() = %v, %v", only.All(), err)
	}
	if _, err := character.NewLanguages(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewLanguages(0) error = %v", err)
	}
	if _, err := l.WithAdded(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("WithAdded(0) error = %v", err)
	}
	if _, err := (character.Languages{}).WithAdded(content.LanguageOrc); !errors.Is(err, srderr.ErrInvalidInput) || !(character.Languages{}).IsZero() || (character.Languages{}).Contains(content.LanguageCommon) {
		t.Errorf("the zero Languages is not invalid: %v", err)
	}
	if l.Contains(0) || l.Contains(content.LanguageOrc) {
		t.Error("Contains answered true for an unknown language")
	}
	all := l.All()
	all[0] = content.LanguageOrc
	if !l.Contains(content.LanguageCommon) {
		t.Error("All shares its slice")
	}
}

// Requirement: doc/specs/character-creation.md FR-049, FR-053 - All lists every language including the last of the vocabulary (Undercommon), in
// content.Language order.
// Path: unhappy (boundary).
func TestLanguagesIncludeTheLastLanguage(t *testing.T) {
	t.Parallel()
	l := must(character.NewLanguages(content.LanguageUndercommon, content.LanguageAbyssal))
	if want := []content.Language{content.LanguageCommon, content.LanguageAbyssal, content.LanguageUndercommon}; !reflect.DeepEqual(l.All(), want) {
		t.Errorf("All() = %v, want %v", l.All(), want)
	}
}
