package content

import "slices"

// MultiattackPlan is the structured reading of the Multiattack action of a
// stat block (SRD 5.2.1, "Multiattack", p. 257): one or more variants, each
// listing the attacks the monster makes. The plan is data set beside the
// verbatim prose, never read from it. The zero value is invalid and means the
// part has no plan. It holds slices, so compare it with
// [MultiattackPlan.Equal].
type MultiattackPlan struct{ variants []MultiattackVariant }

// MultiattackVariant is one way the Multiattack action goes: an optional part
// the monster must have used this turn, the attack slot groups, the parts it
// may use in addition and the replacement rules (content-monsters FR-098). The
// zero value is invalid. It holds slices, so compare it with
// [MultiattackVariant.Equal].
type MultiattackVariant struct {
	requiresUsed  string
	groups        []AttackSlotGroup
	additionalUse []AdditionalUse
	replacements  []ReplacementRule
}

// AttackSlotGroup is a number of attacks made with any combination of a set of
// parts ("three attacks, using Foreleg or Poison Burst in any combination").
// The zero value is invalid. It holds a slice, so compare it with
// [AttackSlotGroup.Equal].
type AttackSlotGroup struct {
	count int
	parts []string
}

// AdditionalUse is a set of parts one of which the monster may also use ("and
// uses Unsettling Visage if available"). The zero value is invalid. It holds a
// slice, so compare it with [AdditionalUse.Equal].
type AdditionalUse struct{ options []string }

// ReplacementOption is a part an attack may be replaced with, and optionally
// the spell cast with it ("Spellcasting to cast Scorching Ray"). The zero
// value is invalid. Comparable and safe for concurrent use.
type ReplacementOption struct {
	part       string
	spellName  string
	spellLevel int
}

// ReplacementScope says which attacks a [ReplacementRule] replaces, a closed
// family with two variants: [ReplacementScopeAnyAttack] and
// [ReplacementScopeOneAttackPart]. Consumers switch over it exhaustively.
// Every variant is immutable, comparable and safe for concurrent use.
type ReplacementScope interface{ isReplacementScope() }

// ReplacementScopeAnyAttack replaces any attack of the variant. The zero value
// is valid.
type ReplacementScopeAnyAttack struct{}

func (ReplacementScopeAnyAttack) isReplacementScope() {}

// ReplacementScopeOneAttackPart replaces an attack made with one named part,
// which is a part of some group of the variant. The zero value is invalid;
// build it with [NewReplacementScopeOneAttackPart].
type ReplacementScopeOneAttackPart struct{ part string }

// NewReplacementScopeOneAttackPart returns the scope of the part called name.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidName when name is not a valid name. It never panics.
func NewReplacementScopeOneAttackPart(name string) (ReplacementScopeOneAttackPart, error) {
	if err := checkName("ReplacementScopeOneAttackPart", name); err != nil {
		return ReplacementScopeOneAttackPart{}, err
	}
	return ReplacementScopeOneAttackPart{part: name}, nil
}

// Part returns the name of the part.
func (s ReplacementScopeOneAttackPart) Part() string { return s.part }

func (ReplacementScopeOneAttackPart) isReplacementScope() {}

// ReplacementLimit says how often a [ReplacementRule] applies. The wire
// spelling is "once" or "anyNumber". The zero value is invalid. Comparable and
// safe for concurrent use.
type ReplacementLimit int

// The replacement limits.
const (
	ReplaceOnce ReplacementLimit = iota + 1
	ReplaceAnyNumber
)

func replacementLimitNames() []string { return []string{"once", "anyNumber"} }

// Valid reports whether l is one of the two limits.
func (l ReplacementLimit) Valid() bool { return enumValid(replacementLimitNames(), int(l)) }

// String returns the wire spelling of l, or "unknown" for an undefined value.
func (l ReplacementLimit) String() string { return enumString(replacementLimitNames(), int(l)) }

// ParseReplacementLimit returns the limit spelled exactly s. Matching is
// case-sensitive and does not trim.
//
// It returns a *[ParseError] wrapping srderr.ErrInvalidInput for any other
// string. It never panics.
func ParseReplacementLimit(s string) (ReplacementLimit, error) {
	v, err := parseEnum("ReplacementLimit", replacementLimitNames(), s)
	return ReplacementLimit(v), err
}

// ReplacementRule replaces a number of attacks of a variant with a use of one
// of several parts ("replace one attack with a use of Sleep Breath or
// Spellcasting"). The zero value is invalid. It holds a slice, so compare it
// with [ReplacementRule.Equal].
type ReplacementRule struct {
	count   int
	scope   ReplacementScope
	options []ReplacementOption
	limit   ReplacementLimit
}

// nameList checks a list of part names for block: at least one, each a valid
// name, none repeated.
func nameList(block string, names []string) error {
	if len(names) == 0 {
		return blockError(block, BlockEmpty)
	}
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		if err := checkName(block, n); err != nil {
			return err
		}
		if _, dup := seen[n]; dup {
			return blockError(block, BlockDuplicate)
		}
		seen[n] = struct{}{}
	}
	return nil
}

// NewAttackSlotGroup returns the group of count attacks made with a copy of
// parts.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for a count below [MonsterMinAttackCount] or above [MonsterMaxNumber], then
// BlockEmpty for no part, BlockInvalidName for a part that is not a valid
// name and BlockDuplicate for a repeated part. It never panics.
func NewAttackSlotGroup(count int, parts []string) (AttackSlotGroup, error) {
	if count < MonsterMinAttackCount || count > MonsterMaxNumber {
		return AttackSlotGroup{}, blockError("AttackSlotGroup", BlockOutOfRange)
	}
	if err := nameList("AttackSlotGroup", parts); err != nil {
		return AttackSlotGroup{}, err
	}
	return AttackSlotGroup{count: count, parts: slices.Clone(parts)}, nil
}

// Count returns the number of attacks.
func (g AttackSlotGroup) Count() int { return g.count }

// Parts returns the names of the parts, as a new slice.
func (g AttackSlotGroup) Parts() []string { return slices.Clone(g.parts) }

// Equal reports whether g and o are the same group.
func (g AttackSlotGroup) Equal(o AttackSlotGroup) bool {
	return g.count == o.count && slices.Equal(g.parts, o.parts)
}

// NewAdditionalUse returns the additional use of one of a copy of options.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockEmpty for
// no option, BlockInvalidName for an option that is not a valid name and
// BlockDuplicate for a repeated option. It never panics.
func NewAdditionalUse(options []string) (AdditionalUse, error) {
	if err := nameList("AdditionalUse", options); err != nil {
		return AdditionalUse{}, err
	}
	return AdditionalUse{options: slices.Clone(options)}, nil
}

// Options returns the names of the parts, as a new slice.
func (u AdditionalUse) Options() []string { return slices.Clone(u.options) }

// Equal reports whether u and o are the same additional use.
func (u AdditionalUse) Equal(o AdditionalUse) bool { return slices.Equal(u.options, o.options) }

// NewReplacementOption returns the option of the part called part. A
// non-empty spellName names the spell cast with it, and castAtLevel is then 0
// (the lowest possible level) or a level from [MonsterMinSpellLevel] to
// [MonsterMaxSpellLevel].
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidName for a part or spell name that is not a valid name,
// BlockOutOfRange for a level outside the range and BlockConstraint for a
// level with no spell. It never panics.
func NewReplacementOption(part, spellName string, castAtLevel int) (ReplacementOption, error) {
	const block = "ReplacementOption"
	if err := checkName(block, part); err != nil {
		return ReplacementOption{}, err
	}
	switch {
	case spellName == "" && castAtLevel != 0:
		return ReplacementOption{}, blockError(block, BlockConstraint)
	case spellName != "":
		if err := checkName(block, spellName); err != nil {
			return ReplacementOption{}, err
		}
	}
	if castAtLevel != 0 && (castAtLevel < MonsterMinSpellLevel || castAtLevel > MonsterMaxSpellLevel) {
		return ReplacementOption{}, blockError(block, BlockOutOfRange)
	}
	return ReplacementOption{part: part, spellName: spellName, spellLevel: castAtLevel}, nil
}

// Part returns the name of the part.
func (o ReplacementOption) Part() string { return o.part }

// Spell returns the name of the spell cast with the part and the level it is
// cast at (0 for the lowest possible level), and false when no spell is cast.
func (o ReplacementOption) Spell() (name string, castAtLevel int, ok bool) {
	return o.spellName, o.spellLevel, o.spellName != ""
}

// validReplacementScope reports whether s is a variant its constructor would
// accept.
func validReplacementScope(s ReplacementScope) bool {
	switch v := s.(type) {
	case ReplacementScopeAnyAttack:
		return true
	case ReplacementScopeOneAttackPart:
		_, err := NewReplacementScopeOneAttackPart(v.part)
		return err == nil
	}
	return false
}

// NewReplacementRule returns the rule replacing count attacks in scope with a
// use of one of a copy of options, as often as limit says.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for a count below [MonsterMinAttackCount] or above [MonsterMaxNumber],
// BlockMissing for a nil scope, BlockInvalidMember for an invalid scope,
// BlockEmpty for no option, BlockInvalidMember for an invalid option,
// BlockDuplicate for two options naming the same part, BlockInvalidMember for
// an invalid limit. It never panics.
func NewReplacementRule(count int, scope ReplacementScope, options []ReplacementOption, limit ReplacementLimit) (ReplacementRule, error) {
	const block = "ReplacementRule"
	switch {
	case count < MonsterMinAttackCount || count > MonsterMaxNumber:
		return ReplacementRule{}, blockError(block, BlockOutOfRange)
	case scope == nil:
		return ReplacementRule{}, blockError(block, BlockMissing)
	case !validReplacementScope(scope):
		return ReplacementRule{}, blockError(block, BlockInvalidMember)
	case len(options) == 0:
		return ReplacementRule{}, blockError(block, BlockEmpty)
	}
	seen := make(map[string]struct{}, len(options))
	for _, o := range options {
		if _, err := NewReplacementOption(o.part, o.spellName, o.spellLevel); err != nil {
			return ReplacementRule{}, blockError(block, BlockInvalidMember)
		}
		if _, dup := seen[o.part]; dup {
			return ReplacementRule{}, blockError(block, BlockDuplicate)
		}
		seen[o.part] = struct{}{}
	}
	if !limit.Valid() {
		return ReplacementRule{}, blockError(block, BlockInvalidMember)
	}
	return ReplacementRule{count: count, scope: scope, options: slices.Clone(options), limit: limit}, nil
}

// Count returns the number of attacks replaced.
func (r ReplacementRule) Count() int { return r.count }

// Scope returns which attacks the rule replaces.
func (r ReplacementRule) Scope() ReplacementScope { return r.scope }

// Options returns the replacement options, as a new slice.
func (r ReplacementRule) Options() []ReplacementOption { return slices.Clone(r.options) }

// Limit returns how often the rule applies.
func (r ReplacementRule) Limit() ReplacementLimit { return r.limit }

// Equal reports whether r and o are the same rule.
func (r ReplacementRule) Equal(o ReplacementRule) bool {
	return r.count == o.count && r.scope == o.scope && slices.Equal(r.options, o.options) && r.limit == o.limit
}

// NewMultiattackVariant returns the variant of requiresUsed ("" for none) and
// copies of the groups, additional uses and replacements.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput for the first
// problem of [MultiattackVariant]: BlockInvalidName for a requiresUsed that is
// not a valid name, BlockOutOfRange for no group, BlockInvalidMember for a zero
// member, BlockConstraint for groups that share a part and for a
// [ReplacementScopeOneAttackPart] naming a part no group holds. It never
// panics.
func NewMultiattackVariant(requiresUsed string, groups []AttackSlotGroup, additionalUses []AdditionalUse, replacements []ReplacementRule) (MultiattackVariant, error) {
	v := MultiattackVariant{
		requiresUsed: requiresUsed, groups: slices.Clone(groups),
		additionalUse: slices.Clone(additionalUses), replacements: slices.Clone(replacements),
	}
	if requiresUsed != "" {
		if err := checkName("MultiattackVariant", requiresUsed); err != nil {
			return MultiattackVariant{}, err
		}
	}
	if is := v.problems(); len(is) > 0 {
		return MultiattackVariant{}, is[0].blockError("MultiattackVariant")
	}
	return v, nil
}

// problems returns the rules the variant breaks, with paths relative to it:
// no group (outOfRange at groups), a zero member (invalidValue at it), a group
// that shares a part with an earlier group (constraintViolation at the later
// group) and a oneAttackPart scope naming a part no group holds
// (constraintViolation at scope/part). The decoder reports exactly these.
func (v MultiattackVariant) problems() []issue {
	var out []issue
	if len(v.groups) == 0 {
		out = append(out, issue{"groups", ProblemOutOfRange})
	}
	held := map[string]struct{}{}
	for i, g := range v.groups {
		p := idx("groups", i)
		if g.count == 0 {
			out = append(out, issue{p, ProblemInvalidValue})
			continue
		}
		shared := false
		for _, n := range g.parts {
			if _, ok := held[n]; ok {
				shared = true
			}
		}
		if shared {
			out = append(out, issue{p, ProblemConstraintViolation})
		}
		for _, n := range g.parts {
			held[n] = struct{}{}
		}
	}
	for i, u := range v.additionalUse {
		if len(u.options) == 0 {
			out = append(out, issue{idx("additionalUses", i), ProblemInvalidValue})
		}
	}
	for i, r := range v.replacements {
		switch s := r.scope.(type) {
		case nil:
			out = append(out, issue{idx("replacements", i), ProblemInvalidValue})
		case ReplacementScopeOneAttackPart:
			if _, ok := held[s.part]; !ok {
				out = append(out, issue{idx("replacements", i, "scope", "part"), ProblemConstraintViolation})
			}
		}
	}
	return out
}

// RequiresUsed returns the name of the part the monster must have used this
// turn for the variant to apply, and false when there is none.
func (v MultiattackVariant) RequiresUsed() (string, bool) {
	return v.requiresUsed, v.requiresUsed != ""
}

// Groups returns the attack slot groups, as a new slice.
func (v MultiattackVariant) Groups() []AttackSlotGroup { return slices.Clone(v.groups) }

// AdditionalUses returns the additional uses, as a new slice.
func (v MultiattackVariant) AdditionalUses() []AdditionalUse { return slices.Clone(v.additionalUse) }

// Replacements returns the replacement rules, as a new slice.
func (v MultiattackVariant) Replacements() []ReplacementRule { return slices.Clone(v.replacements) }

// Equal reports whether v and o are the same variant.
func (v MultiattackVariant) Equal(o MultiattackVariant) bool {
	return v.requiresUsed == o.requiresUsed && slices.EqualFunc(v.groups, o.groups, AttackSlotGroup.Equal) &&
		slices.EqualFunc(v.additionalUse, o.additionalUse, AdditionalUse.Equal) &&
		slices.EqualFunc(v.replacements, o.replacements, ReplacementRule.Equal)
}

// NewMultiattackPlan returns the plan of a copy of variants.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput for the first
// problem of [MultiattackPlan]: BlockConstraint for no variant or for a
// variant with requiresUsed when every variant has one (a Multiattack must
// always be available), BlockInvalidMember for a zero variant. It never
// panics.
func NewMultiattackPlan(variants []MultiattackVariant) (MultiattackPlan, error) {
	p := MultiattackPlan{variants: slices.Clone(variants)}
	if is := p.problems(); len(is) > 0 {
		return MultiattackPlan{}, is[0].blockError("MultiattackPlan")
	}
	return p, nil
}

// problems returns the rules the plan breaks, with paths relative to it: no
// variant (constraintViolation at variants), a zero variant (invalidValue at
// it) and a variant with requiresUsed while no variant lacks one
// (constraintViolation at its requiresUsed).
func (p MultiattackPlan) problems() []issue {
	if len(p.variants) == 0 {
		return []issue{{"variants", ProblemConstraintViolation}}
	}
	var out []issue
	unconditional := false
	for i, v := range p.variants {
		switch {
		case len(v.groups) == 0:
			out = append(out, issue{idx("variants", i), ProblemInvalidValue})
		case v.requiresUsed == "":
			unconditional = true
		}
	}
	if !unconditional {
		for i, v := range p.variants {
			if v.requiresUsed != "" {
				out = append(out, issue{idx("variants", i, "requiresUsed"), ProblemConstraintViolation})
			}
		}
	}
	return out
}

// Variants returns the variants, as a new slice.
func (p MultiattackPlan) Variants() []MultiattackVariant { return slices.Clone(p.variants) }

// IsZero reports whether p is the zero value, meaning no plan.
func (p MultiattackPlan) IsZero() bool { return len(p.variants) == 0 }

// Equal reports whether p and o hold the same variants in order.
func (p MultiattackPlan) Equal(o MultiattackPlan) bool {
	return slices.EqualFunc(p.variants, o.variants, MultiattackVariant.Equal)
}

// planName is a part name a plan refers to, with its path relative to the
// plan and whether it must be a part of Actions, Bonus Actions or Reactions
// (requiresUsed) rather than one Actions part.
type planName struct {
	path     string
	name     string
	requires bool
}

// names returns every part name the plan refers to, in document order.
func (p MultiattackPlan) names() []planName {
	var out []planName
	for v, variant := range p.variants {
		base := idx("variants", v)
		if variant.requiresUsed != "" {
			out = append(out, planName{idx(base, "requiresUsed"), variant.requiresUsed, true})
		}
		for g, group := range variant.groups {
			for k, n := range group.parts {
				out = append(out, planName{idx(base, "groups", g, "parts", k), n, false})
			}
		}
		for a, use := range variant.additionalUse {
			for k, n := range use.options {
				out = append(out, planName{idx(base, "additionalUses", a, "options", k), n, false})
			}
		}
		for r, rule := range variant.replacements {
			if s, ok := rule.scope.(ReplacementScopeOneAttackPart); ok {
				out = append(out, planName{idx(base, "replacements", r, "scope", "part"), s.part, false})
			}
			for k, o := range rule.options {
				out = append(out, planName{idx(base, "replacements", r, "options", k, "part"), o.part, false})
			}
		}
	}
	return out
}
