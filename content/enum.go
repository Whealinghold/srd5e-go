package content

// The closed vocabularies of this package share one shape: members are
// numbered from 1 in declaration order, 0 is invalid, and the wire spelling of
// member v is names[v-1]. The three helpers below hold that logic once; each
// vocabulary file keeps its own names list in a function (a package-level var
// is not allowed, architecture.md A4) and its own typed API.

// unknownName is what String returns for a value that is not a member.
const unknownName = "unknown"

// enumValid reports whether v is a member ordinal of a vocabulary with the
// given names.
func enumValid(names []string, v int) bool { return v >= 1 && v <= len(names) }

// enumString returns the wire spelling of member v, or unknownName.
func enumString(names []string, v int) string {
	if !enumValid(names, v) {
		return unknownName
	}
	return names[v-1]
}

// parseEnum returns the ordinal of the member spelled exactly s. Matching is
// case-sensitive and does not trim. It returns a *ParseError naming vocabulary
// when s is not a member.
func parseEnum(vocabulary string, names []string, s string) (int, error) {
	for i, n := range names {
		if n == s {
			return i + 1, nil
		}
	}
	return 0, &ParseError{Vocabulary: vocabulary, Value: s} // equivalent mutant: replaces the literal 0 with 0
}
