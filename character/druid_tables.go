package character

import (
	"errors"
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// DruidInvariantViolation is one invariant a Druid value breaks (class-druid.md FR-141, FR-142, FR-168). The zero value is
// not a violation.
type DruidInvariantViolation int

const (
	// DruidViolationWildShapeRows: the Wild Shape uses are not 20 non-negative rows.
	DruidViolationWildShapeRows DruidInvariantViolation = iota + 1
	// DruidViolationBeastTiers: the Beast Shapes tiers are not three ascending tiers with a positive form count.
	DruidViolationBeastTiers
	// DruidViolationLands: the lands are not the four lands, each with a spell at every one of the levels 3, 5, 7 and 9.
	DruidViolationLands
	// DruidViolationFeatures: the features by level are not empty or 20 lists.
	DruidViolationFeatures
	// DruidViolationUsesOutOfRange: the Wild Shape uses are negative or above the maximum for the level.
	DruidViolationUsesOutOfRange
	// DruidViolationFeatureBeforeLevel: a feature state exists below the level or without the subclass that grants it.
	DruidViolationFeatureBeforeLevel
	// DruidViolationFormWithoutFeature: an active Beast form, Cube or open permission exists without the feature that creates it.
	DruidViolationFormWithoutFeature
	// DruidViolationKnownForms: the known forms are repeated, invalid or more than the level allows.
	DruidViolationKnownForms
	// DruidViolationUndefinedValue: an enum field holds a value outside its set.
	DruidViolationUndefinedValue
)

// String returns the snake_case name of the violation or "unknown".
func (v DruidInvariantViolation) String() string {
	names := []string{"wild_shape_rows", "beast_tiers", "lands", "features", "uses_out_of_range", "feature_before_level", "form_without_feature",
		"known_forms", "undefined_value"}
	if v < 1 || int(v) > len(names) {
		return "unknown"
	}
	return names[v-1]
}

// DruidInvariantError is the error the Druid constructors return: every violated invariant, never only the first. It
// matches srderr.ErrInvalidInput with errors.Is and carries no display text. Build it with a keyed literal.
type DruidInvariantError struct{ Violations []DruidInvariantViolation }

// Error renders the error. The text is not a contract.
func (e *DruidInvariantError) Error() string {
	names := make([]string, len(e.Violations))
	for i, v := range e.Violations {
		names[i] = v.String()
	}
	return "character: invalid Druid value: " + strings.Join(names, ", ")
}

// Unwrap returns srderr.ErrInvalidInput.
func (*DruidInvariantError) Unwrap() error { return srderr.ErrInvalidInput }

// druidViolations returns nil for no violation and the error otherwise; each violation is listed once.
func druidViolations(vs []DruidInvariantViolation) error {
	if len(vs) == 0 {
		return nil
	}
	return &DruidInvariantError{Violations: slices.Compact(slices.Sorted(slices.Values(vs)))}
}

// BeastShapeTier is one row of the Beast Shapes table (SRD 5.2.1, "Level 2: Wild Shape", p. 42): from Druid level MinLevel
// the Druid knows KnownForms forms of at most MaxChallengeRating, with a Fly Speed when FlySpeed. Construct it with a keyed
// literal.
type BeastShapeTier struct {
	MinLevel           int
	KnownForms         int
	MaxChallengeRating content.ChallengeRating
	FlySpeed           bool
}

// DruidLand is one land of the Circle of the Land: the Circle Spells at the Druid levels 3, 5, 7 and 9 (SRD 5.2.1, "Level 3:
// Circle of the Land Spells", p. 46) and the Resistance of Nature's Ward (p. 46). Construct it with a keyed literal.
type DruidLand struct {
	Land LandType
	// Spells holds the spells granted at the levels 3, 5, 7 and 9, in that order.
	Spells [4][]content.SpellID
	// Resistance is the damage type of Nature's Ward for the land.
	Resistance content.DamageType
}

// DruidTablesInput is what NewDruidFeatureTables reads. Construct it with a keyed literal.
type DruidTablesInput struct {
	// WildShapeUses are the maximum uses at the Druid levels 1 to 20 (0 below level 2).
	WildShapeUses []int
	// Tiers are the three rows of the Beast Shapes table in ascending level order.
	Tiers []BeastShapeTier
	// Lands are the four lands of the Circle of the Land.
	Lands []DruidLand
	// FeaturesByLevel are the feature ids granted at exactly each level 1 to 20; empty when unknown.
	FeaturesByLevel [][]content.FeatureID
}

// DruidFeatureTables are the Druid numbers the rules read: Wild Shape uses by level, the Beast Shapes table, the Circle
// Spells and Nature's Ward by land and, when built from class data, the features granted at each level. They hold no
// cantrip, prepared-spell or slot column, which belong to spellcasting. Build them with NewDruidFeatureTables or
// DruidTablesFromClass; the zero value is invalid. Immutable and safe for concurrent use; not comparable with ==.
type DruidFeatureTables struct {
	wild     [MaxLevel]int
	tiers    [3]BeastShapeTier
	lands    [4]DruidLand
	features [MaxLevel][]content.FeatureID
	set      bool
}

// NewDruidFeatureTables copies in into tables. It returns an error matching srderr.ErrInvalidInput that is a
// *DruidInvariantError listing every problem: DruidViolationWildShapeRows for other than 20 non-negative rows,
// DruidViolationBeastTiers for other than three tiers in ascending level order with a positive form count and a valid
// Challenge Rating, DruidViolationLands for other than the four lands once each, each with a valid spell id at all four
// levels and a damage type, DruidViolationFeatures for features that are neither absent nor 20 lists. It never panics.
func NewDruidFeatureTables(in DruidTablesInput) (DruidFeatureTables, error) {
	var vs []DruidInvariantViolation
	var t DruidFeatureTables
	if len(in.WildShapeUses) != MaxLevel || slices.ContainsFunc(in.WildShapeUses, func(n int) bool { return n < 0 || n > content.ClassMaxCount }) {
		vs = append(vs, DruidViolationWildShapeRows)
	} else {
		copy(t.wild[:], in.WildShapeUses)
	}
	if len(in.Tiers) != len(t.tiers) {
		vs = append(vs, DruidViolationBeastTiers)
	} else {
		copy(t.tiers[:], in.Tiers)
		for i, tier := range in.Tiers {
			if tier.KnownForms < 1 || tier.MinLevel < 1 || tier.MinLevel > MaxLevel || i > 0 && tier.MinLevel <= in.Tiers[i-1].MinLevel {
				vs = append(vs, DruidViolationBeastTiers)
				break
			}
		}
	}
	vs = append(vs, t.takeLands(in.Lands)...)
	switch len(in.FeaturesByLevel) {
	case 0:
	case MaxLevel:
		for i, ids := range in.FeaturesByLevel {
			t.features[i] = slices.Clone(ids)
		}
	default:
		vs = append(vs, DruidViolationFeatures)
	}
	if err := druidViolations(vs); err != nil {
		return DruidFeatureTables{}, err
	}
	t.set = true
	return t, nil
}

// takeLands copies lands into t and returns the violation, if any.
func (t *DruidFeatureTables) takeLands(lands []DruidLand) []DruidInvariantViolation {
	seen := map[LandType]bool{}
	if len(lands) != len(t.lands) {
		return []DruidInvariantViolation{DruidViolationLands}
	}
	for i, land := range lands {
		bad := !land.Land.Valid() || seen[land.Land] || !land.Resistance.Valid()
		seen[land.Land] = true
		for j, row := range land.Spells {
			bad = bad || len(row) == 0 || slices.ContainsFunc(row, func(id content.SpellID) bool { return !id.Valid() })
			land.Spells[j] = slices.Clone(row)
		}
		if bad {
			return []DruidInvariantViolation{DruidViolationLands}
		}
		t.lands[i] = land
	}
	return nil
}

// IsZero reports whether t is the invalid zero value.
func (t DruidFeatureTables) IsZero() bool { return !t.set }

// WildShapeUses returns the maximum Wild Shape uses at the Druid level; 0 for a level outside 1 to 20.
func (t DruidFeatureTables) WildShapeUses(level int) int {
	if level < 1 || level > MaxLevel {
		return 0
	}
	return t.wild[level-1]
}

// BeastShapes returns the Beast Shapes tiers in ascending level order, as a new slice.
func (t DruidFeatureTables) BeastShapes() []BeastShapeTier { return slices.Clone(t.tiers[:]) }

// tierAt returns the Beast Shapes row that applies at the Druid level, and false below the first tier.
func (t DruidFeatureTables) tierAt(level int) (BeastShapeTier, bool) {
	for i := len(t.tiers) - 1; i >= 0; i-- {
		if level >= t.tiers[i].MinLevel {
			return t.tiers[i], true
		}
	}
	return BeastShapeTier{}, false
}

// KnownFormsAt returns how many Beast forms the Druid knows at the level; 0 below the first tier.
func (t DruidFeatureTables) KnownFormsAt(level int) int {
	tier, _ := t.tierAt(level)
	return tier.KnownForms
}

// Resistance returns the damage type of Nature's Ward for the land, and false for the zero land.
func (t DruidFeatureTables) Resistance(land LandType) (content.DamageType, bool) {
	if !land.Valid() {
		return 0, false
	}
	for _, l := range t.lands {
		if l.Land == land {
			return l.Resistance, true
		}
	}
	return 0, false
}

// FeaturesAt returns the ids of the features granted at exactly the level, as a new slice; empty when the tables hold no
// features or the level is outside 1 to 20.
func (t DruidFeatureTables) FeaturesAt(level int) []content.FeatureID {
	if level < 1 || level > MaxLevel {
		return []content.FeatureID{}
	}
	return append([]content.FeatureID{}, t.features[level-1]...)
}

const (
	druidWildShapeColumn    content.ClassColumnID = "wild_shape"
	druidWildShapeFeature   content.FeatureID     = "druid_wild_shape"
	circleSpellsFeature     content.FeatureID     = "circle_of_the_land_circle_of_the_land_spells"
	naturesWardFeature      content.FeatureID     = "circle_of_the_land_natures_ward"
	beastShapesTableName                          = "Beast Shapes"
	naturesWardTableName                          = "Nature’s Ward"
	druidCircleSpellLevels                        = 4
	druidFeatureTableCells                        = 4
	druidWardTableCells                           = 2
	druidFeatureTableFlyYes                       = "Yes"
	druidFeatureTableFlyNo                        = "No"
)

// DruidTablesFromClass builds the tables from the Druid class data (class-druid.md FR-172): the Wild Shape uses from the
// wild_shape count column ("—" at level 1 is 0), the Beast Shapes tiers from the table of the Wild Shape feature (the cell
// "1/4" parsed as a Challenge Rating, "Yes" and "No" as true and false), the Circle Spells from the always-prepared facts of
// the four land options of the Circle of the Land, Nature's Ward from its table, and the features granted at each level. It
// returns an error matching srderr.ErrInvalidInput that is a *DruidInvariantError for the definition of another class or for
// any cell it cannot read, never a guess. It never panics.
func DruidTablesFromClass(def content.ClassDefinition) (DruidFeatureTables, error) {
	if def.ID() != DruidClassID {
		return DruidFeatureTables{}, &DruidInvariantError{Violations: []DruidInvariantViolation{DruidViolationWildShapeRows}}
	}
	in := DruidTablesInput{WildShapeUses: make([]int, MaxLevel), FeaturesByLevel: make([][]content.FeatureID, MaxLevel)}
	var vs []DruidInvariantViolation
	for level := 1; level <= MaxLevel; level++ {
		row, err := def.RowAt(level)
		if err != nil {
			return DruidFeatureTables{}, errors.Join(err, &DruidInvariantError{Violations: []DruidInvariantViolation{DruidViolationWildShapeRows}})
		}
		in.WildShapeUses[level-1], _ = countAt(row, druidWildShapeColumn) // a dash is 0
		entries, _ := def.EntriesGainedAt(level)                          // the level is in range
		for _, e := range entries {
			in.FeaturesByLevel[level-1] = append(in.FeaturesByLevel[level-1], e.ID())
		}
	}
	var ok bool
	if in.Tiers, ok = beastShapeTiers(def); !ok {
		vs = append(vs, DruidViolationBeastTiers)
	}
	if in.Lands, ok = druidLands(def); !ok {
		vs = append(vs, DruidViolationLands)
	}
	if len(vs) > 0 {
		return DruidFeatureTables{}, druidViolations(vs)
	}
	return NewDruidFeatureTables(in)
}

// tableOf returns the table called name in the feature, and false when there is none.
func tableOf(entry content.FeatureEntry, name string) (content.FeatureTable, bool) {
	for _, t := range entry.Tables() {
		if t.Name() == name {
			return t, true
		}
	}
	return content.FeatureTable{}, false
}

func textCell(c content.TableCell) (string, bool) {
	v, ok := c.(content.TableCellText)
	return v.Text(), ok
}

func intCell(c content.TableCell) (int, bool) {
	v, ok := c.(content.TableCellInt)
	return v.Value(), ok
}

// beastShapeTiers reads the Beast Shapes table of the Wild Shape feature.
func beastShapeTiers(def content.ClassDefinition) ([]BeastShapeTier, bool) {
	found, isEntry := def.Feature(druidWildShapeFeature).(content.FeatureEntryFound)
	if !isEntry {
		return nil, false
	}
	table, has := tableOf(found.Entry(), beastShapesTableName)
	if !has {
		return nil, false
	}
	var tiers []BeastShapeTier
	for _, row := range table.Rows() {
		if len(row) != druidFeatureTableCells {
			return nil, false
		}
		level, ok1 := intCell(row[0])
		forms, ok2 := intCell(row[1])
		crText, ok3 := textCell(row[2])
		fly, ok4 := textCell(row[3])
		rating, err := content.ParseChallengeRating(crText)
		if !ok1 || !ok2 || !ok3 || !ok4 || err != nil || fly != druidFeatureTableFlyYes && fly != druidFeatureTableFlyNo {
			return nil, false
		}
		tiers = append(tiers, BeastShapeTier{MinLevel: level, KnownForms: forms, MaxChallengeRating: rating, FlySpeed: fly == druidFeatureTableFlyYes})
	}
	return tiers, true
}

// druidLands reads the four land options of the Circle Spells feature and the Nature's Ward table of the Circle of the Land.
func druidLands(def content.ClassDefinition) ([]DruidLand, bool) {
	var sub content.SubclassDefinition
	subs := def.Subclasses()
	i := slices.IndexFunc(subs, func(s content.SubclassDefinition) bool { return s.ID() == CircleOfTheLandSubclassID })
	if i < 0 {
		return nil, false
	}
	sub = subs[i]
	resistance := map[string]content.DamageType{}
	var spells, ward content.FeatureEntry
	for _, f := range sub.Features() {
		switch f.ID() {
		case circleSpellsFeature:
			spells = f
		case naturesWardFeature:
			ward = f
		}
	}
	table, has := tableOf(ward, naturesWardTableName)
	if !has {
		return nil, false
	}
	for _, row := range table.Rows() {
		land, ok1 := "", false
		if len(row) == druidWardTableCells {
			land, ok1 = textCell(row[0])
		}
		name, ok2 := "", false
		if ok1 {
			name, ok2 = textCell(row[1])
		}
		damage, err := content.ParseDamageType(name)
		if !ok1 || !ok2 || err != nil {
			return nil, false
		}
		resistance[strings.ToLower(land)] = damage
	}
	var lands []DruidLand
	for _, opt := range spells.Options() {
		var land DruidLand
		for l := LandArid; l.Valid(); l++ {
			if l.String() == string(opt.ID()) {
				land.Land = l
			}
		}
		dmg, found := resistance[string(opt.ID())]
		land.Resistance = dmg
		for _, fact := range opt.Facts() {
			grants, isGrants := fact.(content.AlwaysPreparedSpells)
			if !isGrants {
				return nil, false
			}
			for _, g := range grants.Grants() {
				slot := (g.Level() - 3) / 2
				if g.Level() < 3 || g.Level() > 9 || g.Level()%2 == 0 || slot >= druidCircleSpellLevels {
					return nil, false
				}
				land.Spells[slot] = g.SpellIDs()
			}
		}
		if !found || !land.Land.Valid() {
			return nil, false
		}
		lands = append(lands, land)
	}
	return lands, true
}

// MustDruidFeatureTables is NewDruidFeatureTables for input known to be valid. It panics with the error
// NewDruidFeatureTables returns, and only then.
func MustDruidFeatureTables(in DruidTablesInput) DruidFeatureTables {
	t, err := NewDruidFeatureTables(in)
	if err != nil {
		panic(err)
	}
	return t
}

// circleSpellsAt returns the Circle Spells of the land at the Druid level: the spells of every row at or below the level,
// in order. The result is a new slice.
func (t DruidFeatureTables) circleSpellsAt(land LandType, level int) []content.SpellID {
	out := []content.SpellID{}
	for _, l := range t.lands {
		if l.Land != land {
			continue
		}
		for i, row := range l.Spells {
			if level >= 3+2*i {
				out = append(out, row...)
			}
		}
	}
	return out
}
