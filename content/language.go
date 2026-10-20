package content

// Language is one of the 19 languages of the SRD: the 10 of the Standard
// Languages table, then the 9 of the Rare Languages table (SRD 5.2.1,
// "Standard Languages", "Rare Languages", p. 20). Primordial is one member;
// its dialects are not. The zero Language is invalid. Comparable and safe for
// concurrent use.
type Language int

// The languages. The apostrophe of Thieves' Cant is the typographic one
// (U+2019), as printed.
const (
	LanguageCommon Language = iota + 1
	LanguageCommonSignLanguage
	LanguageDraconic
	LanguageDwarvish
	LanguageElvish
	LanguageGiant
	LanguageGnomish
	LanguageGoblin
	LanguageHalfling
	LanguageOrc
	LanguageAbyssal
	LanguageCelestial
	LanguageDeepSpeech
	LanguageDruidic
	LanguageInfernal
	LanguagePrimordial
	LanguageSylvan
	LanguageThievesCant
	LanguageUndercommon
)

// standardLanguages is how many members are in the Standard Languages table.
const standardLanguages = 10

func languageNames() []string {
	return []string{
		"Common", "Common Sign Language", "Draconic", "Dwarvish", "Elvish", "Giant",
		"Gnomish", "Goblin", "Halfling", "Orc",
		"Abyssal", "Celestial", "Deep Speech", "Druidic", "Infernal", "Primordial",
		"Sylvan", "Thieves’ Cant", "Undercommon",
	}
}

// Valid reports whether l is one of the 19 languages.
func (l Language) Valid() bool { return enumValid(languageNames(), int(l)) }

// String returns the printed name of l, or "unknown" for an undefined value.
func (l Language) String() string { return enumString(languageNames(), int(l)) }

// IsStandard reports whether l is a Standard Language (the first 10 members).
// It is false for a Rare Language and for an invalid value.
func (l Language) IsStandard() bool { return l >= 1 && l <= standardLanguages }

// ParseLanguage returns the language spelled exactly s. Matching is
// case-sensitive and does not trim, and the apostrophe of "Thieves’ Cant"
// must be U+2019: the ASCII apostrophe is rejected.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseLanguage(s string) (Language, error) {
	v, err := parseEnum("Language", languageNames(), s)
	return Language(v), err
}
