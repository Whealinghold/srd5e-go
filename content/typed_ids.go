package content

// TypedID is implemented by every typed id: a string type whose only valid
// values are those [ParseID] accepts.
type TypedID interface {
	// String returns the id text.
	String() string
	// Valid reports whether the value is a valid id.
	Valid() bool
}

// KeyedID is a [TypedID] that addresses a top-level entry, so a source can
// look it up by Key. SubclassID and FeatureID are not KeyedIDs: the entries
// they name are nested and never looked up in a source.
type KeyedID interface {
	TypedID
	// Key returns the key of the entry the id names; the zero Key when the
	// value is not a valid id.
	Key() Key
	// Missing returns the [MissingContent] for the entry the id names; its key
	// is the zero Key when the value is not a valid id.
	Missing() MissingContent
}

// keyOf returns the key of kind k and the id text s, or the zero Key when s is
// not a valid id (a typed id made by conversion instead of by its parser).
func keyOf(k Kind, s string) Key {
	id, err := ParseID(s)
	if err != nil {
		return Key{}
	}
	return NewKey(k, id)
}

func validID(s string) bool {
	_, err := ParseID(s)
	return err == nil
}

// SpeciesID is the id of a species. A conversion such as SpeciesID("x") can bypass
// validation, so only values from ParseSpeciesID or MustSpeciesID are known valid;
// [SpeciesID.Valid] re-checks one. The zero SpeciesID is invalid. Comparable and safe
// for concurrent use.
type SpeciesID string

// ParseSpeciesID validates s exactly as [ParseID] does and returns it as a SpeciesID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseSpeciesID(s string) (SpeciesID, error) {
	id, err := ParseID(s)
	return SpeciesID(id.String()), err
}

// MustSpeciesID is like [ParseSpeciesID] but panics with its error when s is invalid.
func MustSpeciesID(s string) SpeciesID {
	v, err := ParseSpeciesID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i SpeciesID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i SpeciesID) Valid() bool { return validID(string(i)) }

// Key returns the key of the species named i, or the zero Key when i is invalid.
func (i SpeciesID) Key() Key { return keyOf(KindSpecies, string(i)) }

// Missing returns the [MissingContent] for the species named i.
func (i SpeciesID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = SpeciesID("")

// BackgroundID is the id of a background. A conversion such as BackgroundID("x") can bypass
// validation, so only values from ParseBackgroundID or MustBackgroundID are known valid;
// [BackgroundID.Valid] re-checks one. The zero BackgroundID is invalid. Comparable and safe
// for concurrent use.
type BackgroundID string

// ParseBackgroundID validates s exactly as [ParseID] does and returns it as a BackgroundID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseBackgroundID(s string) (BackgroundID, error) {
	id, err := ParseID(s)
	return BackgroundID(id.String()), err
}

// MustBackgroundID is like [ParseBackgroundID] but panics with its error when s is invalid.
func MustBackgroundID(s string) BackgroundID {
	v, err := ParseBackgroundID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i BackgroundID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i BackgroundID) Valid() bool { return validID(string(i)) }

// Key returns the key of the background named i, or the zero Key when i is invalid.
func (i BackgroundID) Key() Key { return keyOf(KindBackground, string(i)) }

// Missing returns the [MissingContent] for the background named i.
func (i BackgroundID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = BackgroundID("")

// ClassID is the id of a character class. A conversion such as ClassID("x") can bypass
// validation, so only values from ParseClassID or MustClassID are known valid;
// [ClassID.Valid] re-checks one. The zero ClassID is invalid. Comparable and safe
// for concurrent use.
type ClassID string

// ParseClassID validates s exactly as [ParseID] does and returns it as a ClassID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseClassID(s string) (ClassID, error) {
	id, err := ParseID(s)
	return ClassID(id.String()), err
}

// MustClassID is like [ParseClassID] but panics with its error when s is invalid.
func MustClassID(s string) ClassID {
	v, err := ParseClassID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i ClassID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i ClassID) Valid() bool { return validID(string(i)) }

// Key returns the key of the character class named i, or the zero Key when i is invalid.
func (i ClassID) Key() Key { return keyOf(KindCharacterClass, string(i)) }

// Missing returns the [MissingContent] for the character class named i.
func (i ClassID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = ClassID("")

// SubclassID is the id of a subclass. A conversion such as SubclassID("x") can bypass
// validation, so only values from ParseSubclassID or MustSubclassID are known valid;
// [SubclassID.Valid] re-checks one. The zero SubclassID is invalid. Comparable and safe
// for concurrent use.
type SubclassID string

// ParseSubclassID validates s exactly as [ParseID] does and returns it as a SubclassID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseSubclassID(s string) (SubclassID, error) {
	id, err := ParseID(s)
	return SubclassID(id.String()), err
}

// MustSubclassID is like [ParseSubclassID] but panics with its error when s is invalid.
func MustSubclassID(s string) SubclassID {
	v, err := ParseSubclassID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i SubclassID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i SubclassID) Valid() bool { return validID(string(i)) }

var _ TypedID = SubclassID("")

// FeatID is the id of a feat. A conversion such as FeatID("x") can bypass
// validation, so only values from ParseFeatID or MustFeatID are known valid;
// [FeatID.Valid] re-checks one. The zero FeatID is invalid. Comparable and safe
// for concurrent use.
type FeatID string

// ParseFeatID validates s exactly as [ParseID] does and returns it as a FeatID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseFeatID(s string) (FeatID, error) {
	id, err := ParseID(s)
	return FeatID(id.String()), err
}

// MustFeatID is like [ParseFeatID] but panics with its error when s is invalid.
func MustFeatID(s string) FeatID {
	v, err := ParseFeatID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i FeatID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i FeatID) Valid() bool { return validID(string(i)) }

// Key returns the key of the feat named i, or the zero Key when i is invalid.
func (i FeatID) Key() Key { return keyOf(KindFeat, string(i)) }

// Missing returns the [MissingContent] for the feat named i.
func (i FeatID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = FeatID("")

// SpellID is the id of a spell. A conversion such as SpellID("x") can bypass
// validation, so only values from ParseSpellID or MustSpellID are known valid;
// [SpellID.Valid] re-checks one. The zero SpellID is invalid. Comparable and safe
// for concurrent use.
type SpellID string

// ParseSpellID validates s exactly as [ParseID] does and returns it as a SpellID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseSpellID(s string) (SpellID, error) {
	id, err := ParseID(s)
	return SpellID(id.String()), err
}

// MustSpellID is like [ParseSpellID] but panics with its error when s is invalid.
func MustSpellID(s string) SpellID {
	v, err := ParseSpellID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i SpellID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i SpellID) Valid() bool { return validID(string(i)) }

// Key returns the key of the spell named i, or the zero Key when i is invalid.
func (i SpellID) Key() Key { return keyOf(KindSpell, string(i)) }

// Missing returns the [MissingContent] for the spell named i.
func (i SpellID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = SpellID("")

// ItemID is the id of a piece of equipment. A conversion such as ItemID("x") can bypass
// validation, so only values from ParseItemID or MustItemID are known valid;
// [ItemID.Valid] re-checks one. The zero ItemID is invalid. Comparable and safe
// for concurrent use.
type ItemID string

// ParseItemID validates s exactly as [ParseID] does and returns it as a ItemID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseItemID(s string) (ItemID, error) {
	id, err := ParseID(s)
	return ItemID(id.String()), err
}

// MustItemID is like [ParseItemID] but panics with its error when s is invalid.
func MustItemID(s string) ItemID {
	v, err := ParseItemID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i ItemID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i ItemID) Valid() bool { return validID(string(i)) }

// Key returns the key of the piece of equipment named i, or the zero Key when i is invalid.
func (i ItemID) Key() Key { return keyOf(KindEquipment, string(i)) }

// Missing returns the [MissingContent] for the piece of equipment named i.
func (i ItemID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = ItemID("")

// MonsterID is the id of a monster or animal stat block. A conversion such as MonsterID("x") can bypass
// validation, so only values from ParseMonsterID or MustMonsterID are known valid;
// [MonsterID.Valid] re-checks one. The zero MonsterID is invalid. Comparable and safe
// for concurrent use.
type MonsterID string

// ParseMonsterID validates s exactly as [ParseID] does and returns it as a MonsterID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseMonsterID(s string) (MonsterID, error) {
	id, err := ParseID(s)
	return MonsterID(id.String()), err
}

// MustMonsterID is like [ParseMonsterID] but panics with its error when s is invalid.
func MustMonsterID(s string) MonsterID {
	v, err := ParseMonsterID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i MonsterID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i MonsterID) Valid() bool { return validID(string(i)) }

// Key returns the key of the monster or animal stat block named i, or the zero Key when i is invalid.
func (i MonsterID) Key() Key { return keyOf(KindMonster, string(i)) }

// Missing returns the [MissingContent] for the monster or animal stat block named i.
func (i MonsterID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = MonsterID("")

// FeatureID is the id of a class, subclass, species or feat feature. A conversion such as FeatureID("x") can bypass
// validation, so only values from ParseFeatureID or MustFeatureID are known valid;
// [FeatureID.Valid] re-checks one. The zero FeatureID is invalid. Comparable and safe
// for concurrent use.
type FeatureID string

// ParseFeatureID validates s exactly as [ParseID] does and returns it as a FeatureID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseFeatureID(s string) (FeatureID, error) {
	id, err := ParseID(s)
	return FeatureID(id.String()), err
}

// MustFeatureID is like [ParseFeatureID] but panics with its error when s is invalid.
func MustFeatureID(s string) FeatureID {
	v, err := ParseFeatureID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i FeatureID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i FeatureID) Valid() bool { return validID(string(i)) }

var _ TypedID = FeatureID("")

// MagicItemID is the id of a magic item. A conversion such as MagicItemID("x") can bypass
// validation, so only values from ParseMagicItemID or MustMagicItemID are known valid;
// [MagicItemID.Valid] re-checks one. The zero MagicItemID is invalid. Comparable and safe
// for concurrent use.
type MagicItemID string

// ParseMagicItemID validates s exactly as [ParseID] does and returns it as a MagicItemID.
//
// It returns the same *[IDError] as ParseID, with the same precedence, for an
// invalid s. It never panics.
func ParseMagicItemID(s string) (MagicItemID, error) {
	id, err := ParseID(s)
	return MagicItemID(id.String()), err
}

// MustMagicItemID is like [ParseMagicItemID] but panics with its error when s is invalid.
func MustMagicItemID(s string) MagicItemID {
	v, err := ParseMagicItemID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i MagicItemID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i MagicItemID) Valid() bool { return validID(string(i)) }

// Key returns the key of the magic item named i, or the zero Key when i is invalid.
func (i MagicItemID) Key() Key { return keyOf(KindMagicItem, string(i)) }

// Missing returns the [MissingContent] for the magic item named i.
func (i MagicItemID) Missing() MissingContent { return MissingContent{key: i.Key()} }

var _ KeyedID = MagicItemID("")
