package content

import "slices"

// PartUsage is the limited usage a part prints in a parenthetical (SRD 5.2.1,
// "Limited Usage", p. 257), a closed family with five variants:
// [PartUnlimited], [PartPerDay], [PartRecharge], [PartRechargeAfterRest] and
// [PartUsageOther]. Consumers switch over it exhaustively. Every variant is
// immutable, comparable and safe for concurrent use.
type PartUsage interface{ isPartUsage() }

// PartUnlimited is a part with no parenthetical. The zero value is valid.
type PartUnlimited struct{}

func (PartUnlimited) isPartUsage() {}

// PartPerDay is "(N/Day)" or "(N/Day, or M/Day in Lair)". The zero value is
// invalid; build it with [NewPartPerDay].
type PartPerDay struct{ count, inLair int }

// NewPartPerDay returns the usage of count a day and, when inLair is not 0,
// inLair a day in the creature's lair.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange when count is below [MonsterMinUses] or either number is
// above [MonsterMaxNumber] or inLair is negative. It never panics.
func NewPartPerDay(count, inLair int) (PartPerDay, error) {
	if count < MonsterMinUses || count > MonsterMaxNumber || inLair < 0 || inLair > MonsterMaxNumber {
		return PartPerDay{}, blockError("PartPerDay", BlockOutOfRange)
	}
	return PartPerDay{count: count, inLair: inLair}, nil
}

// Count returns the uses a day.
func (p PartPerDay) Count() int { return p.count }

// InLairCount returns the uses a day in the lair, and false when none is
// printed.
func (p PartPerDay) InLairCount() (int, bool) { return p.inLair, p.inLair != 0 }

func (PartPerDay) isPartUsage() {}

// PartRecharge is "(Recharge L-H)": the part recharges on a roll of the
// recharge die from low to high. "Recharge 6" is low 6 and high 6. The zero
// value is invalid; build it with [NewPartRecharge].
type PartRecharge struct{ low, high int }

// NewPartRecharge returns the recharge range low to high.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// when either number is outside [MonsterMinRecharge] to [MonsterMaxRecharge],
// then BlockConstraint when low is above high. It never panics.
func NewPartRecharge(low, high int) (PartRecharge, error) {
	switch {
	case low < MonsterMinRecharge || low > MonsterMaxRecharge || high < MonsterMinRecharge || high > MonsterMaxRecharge:
		return PartRecharge{}, blockError("PartRecharge", BlockOutOfRange)
	case low > high:
		return PartRecharge{}, blockError("PartRecharge", BlockConstraint)
	}
	return PartRecharge{low: low, high: high}, nil
}

// Low returns the lowest roll that recharges the part.
func (p PartRecharge) Low() int { return p.low }

// High returns the highest roll that recharges the part.
func (p PartRecharge) High() int { return p.high }

func (PartRecharge) isPartUsage() {}

// PartRechargeAfterRest is "(Recharge after a Short or Long Rest)". The zero
// value is valid.
type PartRechargeAfterRest struct{}

func (PartRechargeAfterRest) isPartUsage() {}

// PartUsageOther is any other parenthetical, kept verbatim, such as
// "1/Day; Requires Soul Bag". The zero value is invalid; build it with
// [NewPartUsageOther].
type PartUsageOther struct{ text string }

// NewPartUsageOther returns the usage that holds text verbatim.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter when text breaks the text rule. It
// never panics.
func NewPartUsageOther(text string) (PartUsageOther, error) {
	if err := checkText("PartUsageOther", text); err != nil {
		return PartUsageOther{}, err
	}
	return PartUsageOther{text: text}, nil
}

// Text returns the verbatim text.
func (p PartUsageOther) Text() string { return p.text }

func (PartUsageOther) isPartUsage() {}

// validPartUsage reports whether u is a variant its constructor would accept.
func validPartUsage(u PartUsage) bool {
	var err error
	switch v := u.(type) {
	case PartUnlimited, PartRechargeAfterRest:
		return true
	case PartPerDay:
		_, err = NewPartPerDay(v.count, v.inLair)
	case PartRecharge:
		_, err = NewPartRecharge(v.low, v.high)
	case PartUsageOther:
		_, err = NewPartUsageOther(v.text)
	default:
		return false
	}
	return err == nil
}

// MultiattackName is the name of the Actions part that is the Multiattack of a
// stat block (SRD 5.2.1, "Multiattack", p. 257).
const MultiattackName = "Multiattack"

// StatBlockPartFields holds the members of a [StatBlockPart] beyond its name,
// for [NewStatBlockPart]. Construct it with a keyed literal.
type StatBlockPartFields struct {
	// Usage is the limited usage (required); [PartUnlimited] for a part with
	// no parenthetical.
	Usage PartUsage
	// Text is the verbatim text (required).
	Text string
	// Attack is the Attack Roll notation; the zero value means none.
	Attack StatBlockAttack
	// SavingThrow is the Saving Throw notation; the zero value means none.
	SavingThrow StatBlockSavingThrow
	// Clauses are the labelled effect clauses in printed order; a label occurs
	// once.
	Clauses []EffectClause
	// Spellcasting is the structured Spellcasting; the zero value means none.
	Spellcasting MonsterSpellcasting
	// Plan is the Multiattack plan, valid only on the part named
	// [MultiattackName]; the zero value means none.
	Plan MultiattackPlan
}

// StatBlockPart is one trait, action, bonus action, reaction or legendary
// action of a stat block (SRD 5.2.1, "Traits", p. 256): a name, the limited
// usage, the verbatim text and, as data set beside the text, the attack and
// saving throw notation, the effect clauses, the structured Spellcasting and
// the Multiattack plan. A reaction keeps its trigger inside its text. The zero
// value is invalid. It holds slices, so compare it with [StatBlockPart.Equal].
type StatBlockPart struct {
	name string
	f    StatBlockPartFields
}

// partProblems returns the rules the clauses and plan of a part named name
// break, with paths relative to the part: a label that an earlier clause
// already has (invalidValue at the later clause's label) and a plan on a part
// that is not the Multiattack (constraintViolation at plan).
func partProblems(name string, f StatBlockPartFields) []issue {
	var out []issue
	var seen [11]bool
	for i, c := range f.Clauses {
		if !c.label.Valid() {
			out = append(out, issue{idx("clauses", i), ProblemInvalidValue})
			continue
		}
		if seen[c.label] {
			out = append(out, issue{idx("clauses", i, "label"), ProblemInvalidValue})
		}
		seen[c.label] = true
	}
	if !f.Plan.IsZero() && name != MultiattackName {
		out = append(out, issue{"plan", ProblemConstraintViolation})
	}
	return out
}

// NewStatBlockPart returns the part called name with the members of f.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput, for the first
// rule broken: BlockInvalidName for a name that is not a valid name,
// BlockMissing for a nil usage and BlockInvalidMember for an invalid one,
// BlockBlankText or BlockInvalidCharacter for a text that breaks the text
// rule, BlockInvalidMember for an invalid notation or a zero clause,
// BlockInvalidMember for a label that occurs twice among the clauses, and
// BlockConstraint for a plan on a part not named [MultiattackName]. It never
// panics.
func NewStatBlockPart(name string, f StatBlockPartFields) (StatBlockPart, error) {
	const block = "StatBlockPart"
	if err := checkName(block, name); err != nil {
		return StatBlockPart{}, err
	}
	switch {
	case f.Usage == nil:
		return StatBlockPart{}, blockError(block, BlockMissing)
	case !validPartUsage(f.Usage):
		return StatBlockPart{}, blockError(block, BlockInvalidMember)
	}
	if err := checkText(block, f.Text); err != nil {
		return StatBlockPart{}, err
	}
	if is := partProblems(name, f); len(is) > 0 {
		return StatBlockPart{}, is[0].blockError(block)
	}
	f.Clauses = slices.Clone(f.Clauses)
	return StatBlockPart{name: name, f: f}, nil
}

// Name returns the printed name, without the parenthetical.
func (p StatBlockPart) Name() string { return p.name }

// Usage returns the limited usage.
func (p StatBlockPart) Usage() PartUsage { return p.f.Usage }

// Text returns the verbatim text.
func (p StatBlockPart) Text() string { return p.f.Text }

// Attack returns the Attack Roll notation, and false when the part has none.
func (p StatBlockPart) Attack() (StatBlockAttack, bool) { return p.f.Attack, !p.f.Attack.IsZero() }

// SavingThrow returns the Saving Throw notation, and false when the part has
// none.
func (p StatBlockPart) SavingThrow() (StatBlockSavingThrow, bool) {
	return p.f.SavingThrow, !p.f.SavingThrow.IsZero()
}

// Clauses returns the effect clauses in printed order, as a new slice.
func (p StatBlockPart) Clauses() []EffectClause { return slices.Clone(p.f.Clauses) }

// Spellcasting returns the structured Spellcasting, and false when the part
// has none.
func (p StatBlockPart) Spellcasting() (MonsterSpellcasting, bool) {
	return p.f.Spellcasting, !p.f.Spellcasting.IsZero()
}

// Plan returns the Multiattack plan, and false when the part has none.
func (p StatBlockPart) Plan() (MultiattackPlan, bool) { return p.f.Plan, !p.f.Plan.IsZero() }

// IsZero reports whether p is the invalid zero part.
func (p StatBlockPart) IsZero() bool { return p.name == "" }

// Equal reports whether p and o are the same part.
func (p StatBlockPart) Equal(o StatBlockPart) bool {
	a, b := p.f, o.f
	return p.name == o.name && a.Usage == b.Usage && a.Text == b.Text && a.Attack == b.Attack &&
		a.SavingThrow == b.SavingThrow && a.Spellcasting.Equal(b.Spellcasting) && a.Plan.Equal(b.Plan) &&
		slices.EqualFunc(a.Clauses, b.Clauses, EffectClause.Equal)
}

// LegendaryActions is the Legendary Actions section of a stat block: the
// printed use count, the count "in Lair" when printed and the actions (SRD
// 5.2.1, "Legendary Actions", p. 257). The zero value means the monster has
// none. It holds a slice, so compare it with [LegendaryActions.Equal].
type LegendaryActions struct {
	uses    int
	inLair  int
	actions []StatBlockPart
}

// NewLegendaryActions returns the section of uses legendary actions a round
// ("3 (4 in Lair)" is 3 and 4; inLair 0 means none printed) and a copy of
// actions.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for uses below [MonsterMinUses], an inLair below 0 or a
// number above [MonsterMaxNumber], BlockConstraint for no action and
// BlockInvalidMember for a zero action. It never panics.
func NewLegendaryActions(uses, inLair int, actions []StatBlockPart) (LegendaryActions, error) {
	const block = "LegendaryActions"
	switch {
	case uses < MonsterMinUses || uses > MonsterMaxNumber || inLair < 0 || inLair > MonsterMaxNumber:
		return LegendaryActions{}, blockError(block, BlockOutOfRange)
	case len(actions) == 0:
		return LegendaryActions{}, blockError(block, BlockConstraint)
	case slices.ContainsFunc(actions, StatBlockPart.IsZero):
		return LegendaryActions{}, blockError(block, BlockInvalidMember)
	}
	return LegendaryActions{uses: uses, inLair: inLair, actions: slices.Clone(actions)}, nil
}

// Uses returns the printed number of uses.
func (l LegendaryActions) Uses() int { return l.uses }

// UsesInLair returns the number of uses in the lair, and false when none is
// printed.
func (l LegendaryActions) UsesInLair() (int, bool) { return l.inLair, l.inLair != 0 }

// Actions returns the legendary actions in printed order, as a new slice.
func (l LegendaryActions) Actions() []StatBlockPart { return slices.Clone(l.actions) }

// IsZero reports whether l is the zero value, meaning no legendary actions.
func (l LegendaryActions) IsZero() bool { return l.uses == 0 }

// Equal reports whether l and o are the same section.
func (l LegendaryActions) Equal(o LegendaryActions) bool {
	return l.uses == o.uses && l.inLair == o.inLair && slices.EqualFunc(l.actions, o.actions, StatBlockPart.Equal)
}
