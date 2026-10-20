package content

import (
	"strconv"
	"strings"
)

// ClassLevels is the number of levels of a class and so of rows in its
// progression: 20 (SRD 5.2.1, the Features tables of the classes).
const ClassLevels = 20

// ClassMaxCount bounds every count a class stores (picks, quantities, gold,
// cantrips, spells, slots, point costs, table integers): the SRD sets no
// maximum, and the bound keeps a decoded value the same on every platform
// integer width.
const ClassMaxCount = 1<<31 - 1

// ClassColumnID is the id of a class-specific column of the Features table.
// A conversion such as ClassColumnID("x") can bypass validation, so only
// values from ParseClassColumnID or MustClassColumnID are known valid;
// [ClassColumnID.Valid] re-checks one. The zero ClassColumnID is invalid.
// Comparable and safe for concurrent use.
type ClassColumnID string

// ParseClassColumnID validates s exactly as [ParseID] does and returns it as
// a ClassColumnID.
//
// It returns the same *[IDError] as ParseID for an invalid s. It never
// panics.
func ParseClassColumnID(s string) (ClassColumnID, error) {
	id, err := ParseID(s)
	return ClassColumnID(id.String()), err
}

// MustClassColumnID is like [ParseClassColumnID] but panics with its error
// when s is invalid.
func MustClassColumnID(s string) ClassColumnID {
	v, err := ParseClassColumnID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i ClassColumnID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i ClassColumnID) Valid() bool { return validID(string(i)) }

var _ TypedID = ClassColumnID("")

// FeatureOptionID is the id of an option of a class feature, unique within
// that feature. A conversion such as FeatureOptionID("x") can bypass
// validation, so only values from ParseFeatureOptionID or
// MustFeatureOptionID are known valid; [FeatureOptionID.Valid] re-checks one.
// The zero FeatureOptionID is invalid. Comparable and safe for concurrent
// use.
type FeatureOptionID string

// ParseFeatureOptionID validates s exactly as [ParseID] does and returns it
// as a FeatureOptionID.
//
// It returns the same *[IDError] as ParseID for an invalid s. It never
// panics.
func ParseFeatureOptionID(s string) (FeatureOptionID, error) {
	id, err := ParseID(s)
	return FeatureOptionID(id.String()), err
}

// MustFeatureOptionID is like [ParseFeatureOptionID] but panics with its
// error when s is invalid.
func MustFeatureOptionID(s string) FeatureOptionID {
	v, err := ParseFeatureOptionID(s)
	if err != nil {
		panic(err)
	}
	return v
}

// String returns the id text.
func (i FeatureOptionID) String() string { return string(i) }

// Valid reports whether i is a valid id.
func (i FeatureOptionID) Valid() bool { return validID(string(i)) }

var _ TypedID = FeatureOptionID("")

// DeriveFeatureID returns the id of the feature called name owned by the
// class or subclass ownerID: the owner id, an underscore, then the slug of
// the name (lower case, U+0027 and U+2019 deleted, each run of other
// characters one underscore, edge underscores trimmed). A non-zero level is
// appended as "_level_<n>", for an owner that prints two features with one
// name (FR-052, FR-053).
//
// It returns the *[IDError] of [ParseID] for an invalid owner or a result
// longer than 64 characters, and a *[BlockError] wrapping
// srderr.ErrInvalidInput with BlockInvalidName for a name with no letter or
// digit, or BlockOutOfRange for a level that is neither 0 nor from 1 to 20.
// It never panics.
func DeriveFeatureID(ownerID, name string, level int) (FeatureID, error) {
	const block = "DeriveFeatureID"
	if _, err := ParseID(ownerID); err != nil { // equivalent mutant: ParseFeatureID below rejects an invalid owner with the same error
		return "", err // equivalent mutant: replaces the literal "" with ""
	}
	if level != 0 && (level < 1 || level > ClassLevels) { // equivalent mutant: 0 is excluded above, so < and <= agree
		return "", blockError(block, BlockOutOfRange) // equivalent mutant: replaces the literal "" with ""
	}
	slug := featureSlug(name)
	if slug == "" { // equivalent mutant: a length is never negative, so == and <= agree
		return "", blockError(block, BlockInvalidName) // equivalent mutant: replaces the literal "" with ""
	}
	s := ownerID + "_" + slug
	if level != 0 { // equivalent mutant: a negative level was rejected above, so != and > agree
		s += "_level_" + strconv.Itoa(level)
	}
	return ParseFeatureID(s)
}

// featureSlug lower-cases name, deletes apostrophes and turns each run of
// other characters into one underscore, trimming the edges.
func featureSlug(name string) string {
	var b strings.Builder
	pending := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r == '\'' || r == '’':
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if pending && b.Len() > 0 { // equivalent mutant: a length is never negative, so > and != agree
				b.WriteByte('_')
			}
			pending = false
			b.WriteRune(r)
		default:
			pending = true
		}
	}
	return b.String()
}
