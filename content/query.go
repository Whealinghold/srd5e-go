package content

// Query is what was asked of a [Source]. Variants: [LookupQuery],
// [ListKindQuery]. Switch over it with no default. Variants compare by value.
type Query interface{ query() }

// LookupQuery is a request for the entry with Key.
type LookupQuery struct{ Key Key }

// ListKindQuery is a request for every entry of Kind.
type ListKindQuery struct{ Kind Kind }

func (LookupQuery) query()   {}
func (ListKindQuery) query() {}
