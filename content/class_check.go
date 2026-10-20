package content

import (
	"maps"
	"slices"
	"strings"
)

// problems returns the rules broken between the parts of the class (the rules
// no single part can see), with paths relative to the entry, in a fixed order:
// multiclass grants beyond the core traits (FR-026), feature ids without their
// owner's prefix (FR-052, FR-072), each progression row's level and values
// against the declared columns (FR-037, FR-039, FR-040), its spellcasting part
// against the class (FR-030, FR-031), the references of the rows to features
// and of features to rows (FR-044, FR-045), then the subclass rules (FR-056,
// FR-071, FR-073). It expects every part to be valid on its own. The decoder
// reports exactly these, so each rule lives once.
func (c ClassDefinition) problems() []issue {
	f := c.f
	out := multiclassIssues(f)
	for i, e := range f.Features {
		if !strings.HasPrefix(string(e.id), c.id.String()+"_") {
			out = append(out, issue{idx("features", i, "id"), ProblemConstraintViolation})
		}
	}
	for s, sub := range f.Subclasses {
		for i, e := range sub.features {
			if !strings.HasPrefix(string(e.id), string(sub.id)+"_") {
				out = append(out, issue{idx("subclasses", s, "features", i, "id"), ProblemConstraintViolation})
			}
		}
	}
	for i, r := range f.Progression {
		if r.level != i+1 {
			out = append(out, issue{idx("progression", i, "level"), ProblemInvalidValue})
		}
		out = append(out, valueIssues(f.Columns, i, r)...)
	}
	for i, r := range f.Progression {
		out = append(out, spellcastingIssues(f.Spellcasting, i, r.spellcasting)...)
	}
	out = append(out, referenceIssues(f)...)
	return append(out, subclassIssues(f)...)
}

// valueIssues compares the values of row i with the declared columns: a
// declared column with no value is missingField, a value of the wrong kind
// wrongType and a value with no column unknownField.
func valueIssues(columns []ClassColumn, i int, r ProgressionRow) []issue {
	var out []issue
	declared := make(map[ClassColumnID]struct{}, len(columns))
	for _, col := range columns {
		declared[col.id] = struct{}{}
		p := idx("progression", i, "values", string(col.id))
		switch v, ok := r.values[col.id]; {
		case !ok:
			out = append(out, issue{p, ProblemMissingField})
		case !v.fits(col.kind):
			out = append(out, issue{p, ProblemWrongType})
		}
	}
	for _, id := range slices.Sorted(maps.Keys(r.values)) {
		if _, ok := declared[id]; !ok {
			out = append(out, issue{idx("progression", i, "values", string(id)), ProblemUnknownField})
		}
	}
	return out
}

// spellcastingIssues checks the spellcasting part of row i against the class
// trait: the variant must match (constraintViolation), a caster's row must
// have the part (missingField), and the cantrips column must be present
// exactly when the class has the flag.
func spellcastingIssues(trait ClassSpellcastingTrait, i int, row SpellcastingRow) []issue {
	p := idx("progression", i, "spellcasting")
	bad := []issue{{p, ProblemConstraintViolation}}
	switch t := trait.(type) {
	case SlotSpellcastingTrait:
		switch r := row.(type) {
		case nil:
			return []issue{{p, ProblemMissingField}}
		case SlotSpellcastingRow:
			switch {
			case t.cantrips && !r.hasCantrips:
				return []issue{{p + "/cantrips", ProblemMissingField}}
			case !t.cantrips && r.hasCantrips:
				return []issue{{p + "/cantrips", ProblemConstraintViolation}}
			}
			return nil
		}
		return bad
	case PactSpellcastingTrait:
		switch row.(type) {
		case nil:
			return []issue{{p, ProblemMissingField}}
		case PactSpellcastingRow:
			return nil
		}
		return bad
	}
	if row != nil {
		return bad
	}
	return nil
}

// levelFeature keys the set of class features a row names, so FR-045 is one
// map lookup per feature rather than a scan of the row.
type levelFeature struct {
	level int
	id    FeatureID
}

// referenceIssues applies FR-044 (a row names a feature of the class, gained
// at or before the row's level) and FR-045 (a feature is named by the row of
// its own level).
func referenceIssues(f ClassFields) []issue {
	var out []issue
	byID := make(map[FeatureID]FeatureEntry, len(f.Features))
	for _, e := range slices.Backward(f.Features) {
		byID[e.id] = e // the first of a repeated id wins
	}
	for i, r := range f.Progression {
		for j, rf := range r.features {
			rc, ok := rf.(RowClassFeature)
			if !ok {
				continue
			}
			p := idx("progression", i, "features", j, "featureId")
			switch e, found := byID[rc.featureID]; {
			case !found:
				out = append(out, issue{p, ProblemUnresolvedReference})
			case e.level > r.level:
				out = append(out, issue{p, ProblemConstraintViolation})
			}
		}
	}
	listed := make(map[levelFeature]struct{})
	for _, r := range f.Progression {
		for _, rf := range r.features {
			if rc, ok := rf.(RowClassFeature); ok {
				listed[levelFeature{r.level, rc.featureID}] = struct{}{}
			}
		}
	}
	for i, e := range f.Features {
		if _, ok := listed[levelFeature{f.Progression[e.level-1].level, e.id}]; !ok {
			out = append(out, issue{idx("features", i, "level"), ProblemConstraintViolation})
		}
	}
	return out
}

// subclassIssues applies FR-056 (exactly one class feature grants the
// subclass, no subclass feature does), FR-073 (subclasses only when it does)
// and FR-071 (each subclass has features from the subclass level on, and one
// at it).
func subclassIssues(f ClassFields) []issue {
	var out []issue
	level, grants := 0, 0
	for _, e := range f.Features {
		if slices.ContainsFunc(e.facts, isGrantsSubclass) {
			level = e.level
			grants++
		}
	}
	if grants != 1 {
		out = append(out, issue{"features", ProblemConstraintViolation})
		if len(f.Subclasses) > 0 { // equivalent mutant: a length is never negative, so > and != agree
			out = append(out, issue{"subclasses", ProblemConstraintViolation})
		}
	}
	for s, sub := range f.Subclasses {
		atLevel := false
		for i, e := range sub.features {
			if j := slices.IndexFunc(e.facts, isGrantsSubclass); j >= 0 {
				out = append(out, issue{idx("subclasses", s, "features", i, "facts", j), ProblemConstraintViolation})
			}
			if grants != 1 { // equivalent mutant when != becomes >: with no subclass fact the level checks below flag nothing
				continue
			}
			atLevel = atLevel || e.level == level
			if e.level < level {
				out = append(out, issue{idx("subclasses", s, "features", i), ProblemConstraintViolation})
			}
		}
		if grants == 1 && !atLevel {
			out = append(out, issue{idx("subclasses", s, "features"), ProblemConstraintViolation})
		}
	}
	return out
}

// multiclassIssues applies FR-026: the multiclass grants never exceed the core
// traits. Each violation is constraintViolation at the offending element.
func multiclassIssues(f ClassFields) []issue {
	var out []issue
	g := f.MulticlassGrants
	for i, w := range g.weapons {
		if !weaponCovered(f.WeaponProficiencies, w) {
			out = append(out, issue{idx("multiclassGrants", "weaponProficiencies", i), ProblemConstraintViolation})
		}
	}
	for i, a := range g.armor {
		if !slices.Contains(f.ArmorTraining, a) {
			out = append(out, issue{idx("multiclassGrants", "armorTraining", i), ProblemConstraintViolation})
		}
	}
	for i, t := range g.tools {
		if !toolCovered(f.ToolProficiencies, t) {
			out = append(out, issue{idx("multiclassGrants", "toolProficiencies", i), ProblemConstraintViolation})
		}
	}
	if g.skills != nil && !skillsCovered(f.SkillChoice, g.skills) {
		out = append(out, issue{idx("multiclassGrants", "skillChoice"), ProblemConstraintViolation})
	}
	return out
}

// weaponCovered reports whether the core trainings include everything w grants.
func weaponCovered(core []WeaponProficiency, w WeaponProficiency) bool {
	return slices.ContainsFunc(core, func(c WeaponProficiency) bool {
		if c.category != w.category {
			return false
		}
		if c.anyOf == nil {
			return true
		}
		return w.anyOf != nil && !slices.ContainsFunc(w.anyOf, func(p WeaponProperty) bool { return !slices.Contains(c.anyOf, p) })
	})
}

// toolCovered reports whether the core tool proficiencies include everything
// t grants: a named tool the core names, or categories of a core choice (or a
// tool the core names) within the most picks the core offers.
func toolCovered(core []ClassToolProficiency, t ClassToolProficiency) bool {
	maxPicks := 0
	for _, c := range core {
		switch v := c.(type) {
		case ClassToolProficiencySpecific:
			maxPicks = max(maxPicks, 1)
		case ClassToolProficiencyChoice:
			maxPicks = max(maxPicks, v.picks)
		}
	}
	named := func(name string) bool {
		return slices.ContainsFunc(core, func(c ClassToolProficiency) bool {
			switch v := c.(type) {
			case ClassToolProficiencySpecific:
				return v.name == name
			case ClassToolProficiencyChoice:
				return slices.Contains(v.categories, name)
			}
			return false // callers pass validated values, so this is unreachable
		})
	}
	switch v := t.(type) {
	case ClassToolProficiencySpecific:
		return named(v.name)
	case ClassToolProficiencyChoice:
		return v.picks <= maxPicks && !slices.ContainsFunc(v.categories, func(cat string) bool { return !named(cat) })
	}
	return false // callers pass validated values, so this is unreachable
}

// skillsCovered reports whether the multiclass skill grant stays within the
// core choice: no more picks, listed skills among the core's, and an any-skill
// grant only on an any-skill core.
func skillsCovered(core, grant SkillChoice) bool {
	if grant.Picks() > core.Picks() {
		return false
	}
	switch g := grant.(type) {
	case SkillChoiceAny:
		_, ok := core.(SkillChoiceAny)
		return ok
	case SkillChoiceListed:
		c, ok := core.(SkillChoiceListed)
		return !ok || !slices.ContainsFunc(g.skills, func(s Skill) bool { return !slices.Contains(c.skills, s) })
	}
	return false // callers pass validated values, so this is unreachable
}
