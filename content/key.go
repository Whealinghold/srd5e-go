package content

// Key identifies one entry: its [Kind] and its [ID]. The same id under two
// kinds is two keys. Two keys are equal under == iff both parts are equal, so a
// Key is a map key. The zero Key is invalid (see [Key.IsZero]). Safe for
// concurrent use.
type Key struct {
	kind Kind
	id   ID
}

// NewKey returns the key of the entry of kind k named id. It does not
// validate: a key with an invalid kind or the zero ID is the invalid key
// that [Key.IsZero] and [Kind.Valid] describe.
func NewKey(k Kind, id ID) Key { return Key{kind: k, id: id} }

// Kind returns the kind part of k.
func (k Key) Kind() Kind { return k.kind }

// ID returns the id part of k.
func (k Key) ID() ID { return k.id }

// IsZero reports whether k is the zero Key.
func (k Key) IsZero() bool { return k == Key{} }

// valid reports whether both parts of k are valid.
func (k Key) valid() bool { return k.kind.Valid() && !k.id.IsZero() }
