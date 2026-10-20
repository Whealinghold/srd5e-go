package content

// Entry is one piece of content that a [Source] can supply and a [Set] can
// hold. Every implementation MUST be an immutable value: unexported fields,
// value receivers, and collections copied on construction and on read. Go
// cannot enforce this, so it is a rule of this interface.
type Entry interface {
	// Key returns the kind and id of the entry.
	Key() Key
}
