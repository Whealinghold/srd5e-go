package content

// Limits of one decode (schema FR-018, plan item P13). A hostile document
// would otherwise yield a problem per bad item and memory far above its size.
const (
	// MaxDocumentBytes is the largest document Decode reads, 16 MiB. A larger
	// one is rejected, before it is parsed, with one outOfRange problem at the
	// path "". Hosts that need more split the content over several documents
	// and merge them with [ValidateSet].
	MaxDocumentBytes = 16 << 20
	// MaxProblems is the most problems Decode reports, 1000. When more were
	// found, or decoding stopped at the cap, the list ends with one further
	// outOfRange problem at the path "" (the truncation problem), so a list of
	// MaxProblems+1 problems is incomplete. Root-level problems that cannot
	// be told from it never accompany it: the size cap is the only other
	// problem at the path "" with that kind, and it stands alone.
	MaxProblems = 1000
)

// SchemaVersion is the one version of the JSON format the decoder reads
// (FR-007).
const SchemaVersion = 1

// SupportedSchemaVersions returns the schema versions the decoder reads, as a
// new slice: exactly {SchemaVersion}.
func SupportedSchemaVersions() []int { return []int{SchemaVersion} }

// JSONDecoder is the [Decoder] for the JSON format of schema FR-001 to
// FR-031: a strict decoder that turns a document into a [Set], or into an
// error that joins every problem found (to [MaxProblems]) in the order of
// FR-016. The zero value is ready to use and, holding no state, is safe for
// concurrent use.
//
// Decoding is a pure function of the bytes: no clock, randomness or I/O, and
// the input slice is neither retained nor modified. It is synchronous and
// takes no context, so cancellation is not honoured while a document decodes;
// bound the document size ([MaxDocumentBytes]) and call it where waiting is
// acceptable. It does not resolve references between entries (FR-035): run
// [ValidateSet] on the result.
type JSONDecoder struct{}

var _ Decoder = JSONDecoder{}

// Decode reads document and returns its entries as a Set and a nil error, or
// the zero Set and an error that is errors.Join of one or more [Problem]
// values, read with [Problems] and matching srderr.ErrInvalidInput. Problems
// are in document order: root fields, then each registered array by index,
// the fields of each entry in the order its kind declares, a repeated key at
// its later occurrence, unknown keys in ordinal order, and last the
// duplicate-id problems grouped by kind. Numbers are read from their decimal
// text, so a whole number is the same on every platform integer width.
//
// It never panics, whatever the bytes (nil and empty included), and returns
// no error other than problems.
func (JSONDecoder) Decode(document []byte) (Set, error) {
	return decodeDocument(document, rootArrays())
}

// decodedEntry is what a kind row returns for one array element: the entry,
// nil when the element had a problem, and the element's id when that was
// valid, which puts the element in the duplicate check even when it has other
// problems (FR-095, FR-098).
type decodedEntry struct {
	entry  Entry
	id     ID
	idOK   bool
	nested []nestedID // valid ids of parts inside the entry, in document order
}

// The id spaces inside an entry that must be unique across the document
// (content-classes FR-004), in the order their duplicates are reported.
const (
	nestedSubclass = iota
	nestedFeature
	nestedGroups
)

// nestedID is a valid id of a part inside an entry (a subclass or a feature):
// its id space, the path of the id and the id text.
type nestedID struct {
	group int
	path  string
	id    string
}

// rootArray registers one root array of the revision table: its JSON key, the
// kind of its entries and the row that decodes one element. The row reads the
// fields of the element object in the order its spec declares and returns;
// the framework reports the unknown and repeated keys afterwards.
type rootArray struct {
	key    string
	kind   Kind
	decode func(o *object) decodedEntry
	// check, when not nil, reports the rules between the rows of one array
	// (content-equipment FR-057, FR-064). It is called after the elements,
	// with the path of the array and the entries in order, only when the array
	// is not empty and every element decoded to an entry.
	check func(d *decoder, path string, entries []Entry)
}

// rootArrays returns the registered root arrays in revision-table order
// (FR-008, FR-016): species, backgrounds, classes, feats, spells, weapons,
// armor, then the arrays of the later rows. A switch or function, never a
// package-level table or init registration (plan item P6). Only the rows of
// steps that have landed are registered (species, backgrounds, classes, feats,
// spells, the twelve equipment arrays, monsters and magicItems);
// every other root array is unknownField until its step lands.
func rootArrays() []rootArray {
	return []rootArray{
		{key: "species", kind: KindSpecies, decode: decodeSpecies},
		{key: "backgrounds", kind: KindBackground, decode: decodeBackground},
		{key: "classes", kind: KindCharacterClass, decode: decodeClass},
		{key: "feats", kind: KindFeat, decode: decodeFeat},
		{key: "spells", kind: KindSpell, decode: decodeSpell},
		{key: "weapons", kind: KindEquipment, decode: decodeWeapon},
		{key: "armor", kind: KindEquipment, decode: decodeArmor},
		{key: "tools", kind: KindEquipment, decode: decodeTool, check: checkFamilyLinks},
		{key: "adventuringGear", kind: KindEquipment, decode: decodeGear, check: checkFamilyLinks},
		{key: "mounts", kind: KindEquipment, decode: decodeMount},
		{key: "tackHarnessAndDrawnVehicles", kind: KindEquipment, decode: decodeTack},
		{key: "airborneAndWaterborneVehicles", kind: KindEquipment, decode: decodeVehicle},
		{key: "lifestyleExpenses", kind: KindEquipment, decode: decodeLifestyle},
		{key: "foodDrinkAndLodging", kind: KindEquipment, decode: decodeFood},
		{key: "hirelings", kind: KindEquipment, decode: decodeHireling},
		{key: "spellcastingServices", kind: KindEquipment, decode: decodeService, check: checkServiceRows},
		{key: "spellScrollCosts", kind: KindEquipment, decode: decodeScrollCost, check: checkScrollCostRows},
		{key: "monsters", kind: KindMonster, decode: decodeMonster},
		{key: "magicItems", kind: KindMagicItem, decode: decodeMagicItem},
	}
}

// idRef is an element with a valid id, for the duplicate check.
type idRef struct {
	kind Kind
	path string // the path of the element
	id   ID
}

// decodeDocument decodes document against the registered root arrays.
func decodeDocument(document []byte, arrays []rootArray) (Set, error) {
	d := &decoder{}
	if len(document) > MaxDocumentBytes {
		d.probs.add("", ProblemOutOfRange)
		return Set{}, d.probs.err()
	}
	root, ok := parseDocument(document)
	if !ok {
		d.probs.add("", ProblemSyntax)
		return Set{}, d.probs.err()
	}
	o, ok := d.object(root, "")
	if !ok {
		return Set{}, d.probs.err()
	}
	if !d.checkVersion(o) {
		return Set{}, d.probs.err()
	}
	var entries []Entry
	var ids []idRef
	var nested []nestedID
	for _, a := range arrays {
		n, p, present := o.field(a.key, false)
		if !present {
			continue
		}
		if n.kind() != nodeArray {
			d.probs.add(p, ProblemWrongType)
			continue
		}
		var built []Entry
		clean := true // every element so far decoded to an entry
		for i, el := range n.elements() {
			if d.probs.full() {
				clean = false
				break
			}
			ep := indexPath(p, i)
			eo, ok := d.object(el, ep)
			if !ok {
				clean = false
				continue
			}
			before := d.probs.count
			res := a.decode(eo)
			eo.finish()
			if d.probs.count > before {
				res.entry = nil // a row may build from what failed; a reported problem drops the entry
			}
			if res.entry == nil && d.probs.count == before { // equivalent mutant: count never falls below before, so == and <= agree
				// The row dropped the element without a problem: a bug in the
				// row, reported rather than hidden.
				d.probs.add(ep, ProblemInvalidValue)
			}
			if res.idOK {
				ids = append(ids, idRef{kind: a.kind, path: ep, id: res.id})
			}
			nested = append(nested, res.nested...)
			entries = append(entries, res.entry) // nil only when a problem was reported, and then no Set is built
			if res.entry == nil {
				clean = false
			} else {
				built = append(built, res.entry)
			}
		}
		if a.check != nil && clean && len(built) > 0 {
			a.check(d, p, built)
		}
	}
	o.finish()
	d.checkDuplicateIDs(ids)
	d.checkNestedDuplicates(nested)
	if err := d.probs.err(); err != nil {
		return Set{}, err
	}
	set, err := NewSet(entries...)
	if err != nil {
		// A row returned no entry, or a bad key, without reporting a problem:
		// a bug in the row, reported rather than panicked on.
		d.probs.add("", ProblemInvalidValue)
		return Set{}, d.probs.err()
	}
	return set, nil
}

// checkVersion applies FR-004 to FR-006 to the root object: the version is
// the only field examined, and a problem with it is the only problem. It
// reports whether decoding may go on.
func (d *decoder) checkVersion(o *object) bool {
	n, p, present := o.field("schemaVersion", true)
	if !present {
		return false
	}
	if n.kind() != nodeNumber {
		d.probs.add(p, ProblemWrongType)
		return false
	}
	v, class := classifyNumber(n.number())
	switch {
	case class == numberFractional:
		d.probs.add(p, ProblemWrongType)
	case class == numberTooLarge || !supportedVersion(v): // equivalent mutant: numberTooLarge is the highest class, so == and >= agree
		d.probs.add(p, ProblemUnsupportedVersion)
	default:
		return true
	}
	return false
}

// supportedVersion reports whether v is a supported schema version, compared in
// int64 so the answer never depends on the platform int width (FR-021).
func supportedVersion(v int64) bool {
	for _, s := range SupportedSchemaVersions() {
		if int64(s) == v {
			return true
		}
	}
	return false
}

// checkDuplicateIDs reports, for the elements with a valid id, each later
// element whose key (kind and id) an earlier one holds, at that element's id
// (FR-031, FR-098). Problems are grouped by kind in declaration order, then
// by array and index.
func (d *decoder) checkDuplicateIDs(ids []idRef) {
	seen := make(map[Key]struct{}, len(ids))
	byKind := map[Kind][]string{}
	for _, r := range ids {
		k := NewKey(r.kind, r.id)
		if _, repeated := seen[k]; repeated {
			byKind[r.kind] = append(byKind[r.kind], r.path+"/id")
			continue
		}
		seen[k] = struct{}{}
	}
	for k := KindSpecies; k.Valid(); k++ {
		for _, p := range byKind[k] {
			d.probs.add(p, ProblemDuplicate)
		}
	}
}

// checkNestedDuplicates reports, in each id space of nestedGroups in turn, each
// later part whose id an earlier one holds, at the later part's id, in document
// order. The same text in two id spaces is accepted.
func (d *decoder) checkNestedDuplicates(nested []nestedID) {
	for g := range nestedGroups {
		seen := map[string]struct{}{}
		for _, n := range nested {
			if n.group != g {
				continue
			}
			if _, repeated := seen[n.id]; repeated {
				d.probs.add(n.path, ProblemDuplicate)
				continue
			}
			seen[n.id] = struct{}{}
		}
	}
}
