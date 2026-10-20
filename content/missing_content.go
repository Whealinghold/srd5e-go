package content

// MissingContent names content that was needed and absent. Other contexts'
// result variants hold it as a field; there is no shared variant. It is
// immutable, comparable and carries no display text. The zero value names
// nothing. Safe for concurrent use.
type MissingContent struct{ key Key }

// NewMissingContent returns the failure detail for the entry of kind k named
// id. It does not validate.
func NewMissingContent(k Kind, id ID) MissingContent {
	return MissingContent{key: NewKey(k, id)}
}

// Kind returns the kind of the missing entry.
func (m MissingContent) Kind() Kind { return m.key.Kind() }

// ID returns the id of the missing entry.
func (m MissingContent) ID() ID { return m.key.ID() }

// Key returns the key of the missing entry.
func (m MissingContent) Key() Key { return m.key }
