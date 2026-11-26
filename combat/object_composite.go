package combat

// CollapseRuling is the GM's ruling on whether destroying a section makes the whole
// composite object collapse (SRD 5.2.1, "Breaking Objects", p. 178). The zero value
// is invalid.
type CollapseRuling int

// The two rulings.
const (
	// DoesNotCollapse: destroying the section leaves the rest standing.
	DoesNotCollapse CollapseRuling = iota + 1
	// CollapsesWhole: destroying the section brings the whole object down.
	CollapsesWhole
)

// Valid reports whether r is one of the two rulings.
func (r CollapseRuling) Valid() bool { return r == DoesNotCollapse || r == CollapsesWhole }

// String returns "does_not_collapse", "collapses_whole" or "unknown".
func (r CollapseRuling) String() string {
	return enumName([]string{"does_not_collapse", "collapses_whole"}, int(r))
}

// CompositeStatus is the state of a composite object. The zero value is invalid.
type CompositeStatus int

// The states.
const (
	// CompositeStanding: the object stands.
	CompositeStanding CompositeStatus = iota + 1
	// CompositeCollapsed: a section whose ruling is CollapsesWhole was destroyed.
	CompositeCollapsed
	// CompositeAllSectionsDestroyed: every section is destroyed and none collapses it.
	CompositeAllSectionsDestroyed
)

// String returns "standing", "collapsed", "all_sections_destroyed" or "unknown".
func (s CompositeStatus) String() string {
	return enumName([]string{"standing", "collapsed", "all_sections_destroyed"}, int(s))
}

// CompositeSection is one section of a composite object with the GM's collapse
// ruling for it. Construct it with a keyed literal.
type CompositeSection struct {
	Status ObjectStatus
	Ruling CollapseRuling
}

// CompositeObject is a thing made of many objects, such as a building or a vehicle
// (SRD 5.2.1, "Object", p. 185): the host models it as sections, each an
// ObjectStatus tracked separately, and there is no single-object building type. It
// is immutable. The zero value is invalid; build it with NewCompositeObject.
type CompositeObject struct{ sections []CompositeSection }

// NewCompositeObject returns the composite of the sections, which are copied. It
// returns an error matching srderr.ErrInvalidInput for fewer than two sections, a
// zero ObjectStatus or an invalid ruling, and a LimitExceededError for more than 256 sections
// (plan P10). It never panics.
func NewCompositeObject(sections []CompositeSection) (CompositeObject, error) {
	if err := overCap("composite sections", len(sections), maxInputList); err != nil {
		return CompositeObject{}, err
	}
	if len(sections) < 2 {
		return CompositeObject{}, invalidInput("composite object needs two or more sections")
	}
	for _, s := range sections {
		if s.Status.IsZero() || !s.Ruling.Valid() {
			return CompositeObject{}, invalidInput("composite section needs a status and a ruling")
		}
	}
	return CompositeObject{append([]CompositeSection(nil), sections...)}, nil
}

// Sections returns a copy of the sections.
func (c CompositeObject) Sections() []CompositeSection {
	return append([]CompositeSection(nil), c.sections...)
}

// Status returns the state of the composite: collapsed when a CollapsesWhole section
// is destroyed (this takes precedence), all sections destroyed when every section is
// and none collapses it, otherwise standing. The zero composite is standing.
func (c CompositeObject) Status() CompositeStatus {
	destroyed := 0
	for _, s := range c.sections {
		if s.Status.IsDestroyed() {
			destroyed++
			if s.Ruling == CollapsesWhole {
				return CompositeCollapsed
			}
		}
	}
	if len(c.sections) > 0 && destroyed == len(c.sections) {
		return CompositeAllSectionsDestroyed
	}
	return CompositeStanding
}

// ReplaceSection returns the composite with the status of section index replaced,
// and the events of the change: ObjectCollapsed once, when the composite collapses
// with this change and not before.
//
// It returns an error matching srderr.ErrInvalidInput for an index outside the
// sections or a zero status. The receiver is not changed.
func (c CompositeObject) ReplaceSection(index int, status ObjectStatus) (CompositeObject, []EnvironmentEvent, error) {
	if index < 0 || index >= len(c.sections) || status.IsZero() {
		return CompositeObject{}, nil, invalidInput("replace section needs a section index and a status")
	}
	next := CompositeObject{append([]CompositeSection(nil), c.sections...)}
	next.sections[index].Status = status
	events := []EnvironmentEvent{}
	if c.Status() != CompositeCollapsed && next.Status() == CompositeCollapsed {
		events = append(events, ObjectCollapsed{})
	}
	return next, events, nil
}
