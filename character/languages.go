package character

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// Languages are the languages a character knows (SRD 5.2.1, "Choose Languages",
// p. 20). A built value always holds Common and holds each language once. It is
// an immutable value, comparable with ==. The zero Languages is invalid (it lacks
// Common); see IsZero.
type Languages struct{ bits uint32 }

func languageBit(l content.Language) uint32 { return 1 << uint(l) }

// NewLanguages returns the languages Common and extra. It returns an error
// matching srderr.ErrInvalidInput for an invalid language. It never panics.
func NewLanguages(extra ...content.Language) (Languages, error) {
	l := Languages{bits: languageBit(content.LanguageCommon)}
	for _, e := range extra {
		var err error
		if l, err = l.WithAdded(e); err != nil {
			return Languages{}, err
		}
	}
	return l, nil
}

// WithAdded returns the languages with l added, for a language a class or a
// feature grants (a Rare language is allowed). Adding a known language gives an
// equal value. It returns an error matching srderr.ErrInvalidInput for an invalid
// language or the zero Languages.
func (s Languages) WithAdded(l content.Language) (Languages, error) {
	if !l.Valid() || s.IsZero() {
		return Languages{}, invalidInput("add an invalid language, or to the zero Languages")
	}
	return Languages{bits: s.bits | languageBit(l)}, nil
}

// Contains reports whether l is known.
func (s Languages) Contains(l content.Language) bool { return l.Valid() && s.bits&languageBit(l) != 0 }

// All returns a new slice of the known languages, in content.Language order.
func (s Languages) All() []content.Language {
	var out []content.Language
	for l := content.LanguageCommon; l <= content.LanguageUndercommon; l++ {
		if s.Contains(l) {
			out = append(out, l)
		}
	}
	return out
}

// IsZero reports whether s is the invalid zero Languages.
func (s Languages) IsZero() bool { return s.bits == 0 }

// StandardLanguageRoll maps a d12 to a Standard language (SRD 5.2.1, Standard
// Languages table, p. 20). The zero value is ready to use and safe for concurrent
// use.
type StandardLanguageRoll struct{}

// LanguageFor returns the language of a d12 roll: 1 Common Sign Language, 2
// Draconic, 3 and 4 Dwarvish, 5 and 6 Elvish, 7 Giant, 8 Gnomish, 9 Goblin, 10 and
// 11 Halfling, 12 Orc. It returns an error matching srderr.ErrInvalidInput for a
// roll below 1 or above 12. It never panics.
func (StandardLanguageRoll) LanguageFor(roll int) (content.Language, error) {
	switch roll {
	case 1:
		return content.LanguageCommonSignLanguage, nil
	case 2:
		return content.LanguageDraconic, nil
	case 3, 4:
		return content.LanguageDwarvish, nil
	case 5, 6:
		return content.LanguageElvish, nil
	case 7:
		return content.LanguageGiant, nil
	case 8:
		return content.LanguageGnomish, nil
	case 9:
		return content.LanguageGoblin, nil
	case 10, 11:
		return content.LanguageHalfling, nil
	case 12:
		return content.LanguageOrc, nil
	}
	return 0, invalidInput("standard language roll outside 1 to 12")
}
