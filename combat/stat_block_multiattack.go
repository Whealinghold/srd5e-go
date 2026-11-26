package combat

import (
	"maps"
	"math/bits"
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// MultiattackAllowanceRefusal is why a step of a Multiattack was refused for the plan's allowance.
// The zero value is invalid. Comparable and safe for concurrent use.
type MultiattackAllowanceRefusal int

// The seven refusals. For a replacement they are ordered by how far the step got, and the one
// reached furthest in any candidate variant is reported.
const (
	// MultiattackAttackNotAllowed: no variant still open lists the part as an attack.
	MultiattackAttackNotAllowed MultiattackAllowanceRefusal = iota + 1
	// MultiattackAllowanceSpent: a variant lists the part but its attacks are all made.
	MultiattackAllowanceSpent
	// MultiattackAdditionalUseNotListed: no variant still open lists the part as an additional use.
	MultiattackAdditionalUseNotListed
	// MultiattackAdditionalUseAlreadyMade: the entry listing the part already had its one choice.
	MultiattackAdditionalUseAlreadyMade
	// MultiattackReplacementNotListed: no variant still open has a rule listing the option for that attack.
	MultiattackReplacementNotListed
	// MultiattackReplacementLimitReached: the rule applies once and was already used.
	MultiattackReplacementLimitReached
	// MultiattackNotEnoughAttacksToReplace: the attack's group has too few attacks left to replace.
	MultiattackNotEnoughAttacksToReplace
)

// String returns "attack_not_allowed", "allowance_spent", "additional_use_not_listed",
// "additional_use_already_made", "replacement_not_listed", "replacement_limit_reached",
// "not_enough_attacks_to_replace" or "unknown".
func (r MultiattackAllowanceRefusal) String() string {
	return enumName([]string{"attack_not_allowed", "allowance_spent", "additional_use_not_listed",
		"additional_use_already_made", "replacement_not_listed", "replacement_limit_reached",
		"not_enough_attacks_to_replace"}, int(r))
}

// MultiattackPlanProblemKind says what is wrong with a name or reference of a plan. The zero
// value is invalid. Comparable and safe for concurrent use.
type MultiattackPlanProblemKind int

// The three kinds.
const (
	// MultiattackPlanPartMissing: the name is not an Actions part (the Multiattack itself excluded),
	// or, for a part that must have been used, not an Actions, Bonus Actions or Reactions part.
	MultiattackPlanPartMissing MultiattackPlanProblemKind = iota + 1
	// MultiattackPlanPartShared: two parts carry the name.
	MultiattackPlanPartShared
	// MultiattackPlanSpellMissing: a replacement option names a spell the monster's Spellcasting lacks.
	MultiattackPlanSpellMissing
)

// String returns "part_missing", "part_shared", "spell_missing" or "unknown".
func (k MultiattackPlanProblemKind) String() string {
	return enumName([]string{"part_missing", "part_shared", "spell_missing"}, int(k))
}

// MultiattackPlanProblem is a name of a Multiattack plan that does not resolve against the
// monster. Comparable and safe for concurrent use.
type MultiattackPlanProblem struct {
	kind MultiattackPlanProblemKind
	name string
}

// Kind returns what is wrong.
func (p MultiattackPlanProblem) Kind() MultiattackPlanProblemKind { return p.kind }

// Name returns the name as the plan prints it.
func (p MultiattackPlanProblem) Name() string { return p.name }

// planReferences returns the part names the plan uses as attacks, additional uses and
// replacement options, the part names it requires to have been used, and the spell names its
// replacement options cast, each in document order.
func planReferences(plan content.MultiattackPlan) (actions, used, spells []string) {
	for _, v := range plan.Variants() {
		if n, ok := v.RequiresUsed(); ok {
			used = append(used, n)
		}
		for _, g := range v.Groups() {
			actions = append(actions, g.Parts()...)
		}
		for _, u := range v.AdditionalUses() {
			actions = append(actions, u.Options()...)
		}
		for _, r := range v.Replacements() {
			if s, ok := r.Scope().(content.ReplacementScopeOneAttackPart); ok {
				actions = append(actions, s.Part())
			}
			for _, o := range r.Options() {
				actions = append(actions, o.Part())
				if name, _, ok := o.Spell(); ok {
					spells = append(spells, name)
				}
			}
		}
	}
	return actions, used, spells
}

// actionRefs returns the refs of the Actions parts called name, the Multiattack excluded.
func actionRefs(m content.Monster, name string) []LimitedUsePartRef {
	var out []LimitedUsePartRef
	for i, p := range m.Actions() {
		if p.Name() == name && name != content.MultiattackName {
			out = append(out, LimitedUsePartRef{MonsterPartActions, i})
		}
	}
	return out
}

// usedRefs returns the refs of the Actions, Bonus Actions and Reactions parts called name.
func usedRefs(m content.Monster, name string) []LimitedUsePartRef {
	var out []LimitedUsePartRef
	for _, s := range []MonsterPartSection{MonsterPartActions, MonsterPartBonusActions, MonsterPartReactions} {
		for i, p := range sectionParts(m, s) {
			if p.Name() == name {
				out = append(out, LimitedUsePartRef{s, i})
			}
		}
	}
	return out
}

// spellRef returns the position of the spell called name in the first Spellcasting of m.
func spellRef(m content.Monster, name string) (SpellUseRef, bool) {
	sc, ok := m.Spellcasting()
	if !ok {
		return SpellUseRef{}, false
	}
	for gi, g := range sc.Groups() {
		for si, s := range g.Spells() {
			if s.Name() == name {
				return SpellUseRef{gi, si}, true
			}
		}
	}
	return SpellUseRef{}, false
}

// MultiattackPlanProblemsFor returns one problem for each name or reference of the Multiattack
// plan of m that does not resolve (monster-stat-block-actions.md FR-021): a part name that is not
// an Actions part (the Multiattack itself excluded), a part name two parts carry, a part that must
// have been used and is in none of Actions, Bonus Actions and Reactions or in two, and a
// replacement spell absent from the Spellcasting. Each name is reported once. It returns nil when
// m has no plan or the plan is sound. Content's own constructors reject the first two kinds, so
// they appear only for a monster built another way.
func MultiattackPlanProblemsFor(m content.Monster) []MultiattackPlanProblem {
	part, ok := m.Multiattack()
	plan, has := part.Plan()
	if !ok || !has {
		return nil
	}
	return planProblems(plan, planLookups{
		actions: func(name string) int { return len(actionRefs(m, name)) },
		used:    func(name string) int { return len(usedRefs(m, name)) },
		spell:   func(name string) bool { _, found := spellRef(m, name); return found },
	})
}

// planLookups says how many parts and whether a spell a stat block has for a name: the parts of Actions
// other than the Multiattack, the parts of Actions, Bonus Actions and Reactions, and the spells of the
// Spellcasting.
type planLookups struct {
	actions, used func(name string) int
	spell         func(name string) bool
}

// planProblems returns the problems of plan against what l finds, each name once, in document order.
func planProblems(plan content.MultiattackPlan, l planLookups) []MultiattackPlanProblem {
	actions, used, spells := planReferences(plan)
	var out []MultiattackPlanProblem
	add := func(kind MultiattackPlanProblemKind, name string) {
		if p := (MultiattackPlanProblem{kind, name}); !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	check := func(names []string, count func(string) int) {
		for _, n := range names {
			switch count(n) {
			case 0:
				add(MultiattackPlanPartMissing, n)
			case 1:
			default:
				add(MultiattackPlanPartShared, n)
			}
		}
	}
	check(actions, l.actions)
	check(used, l.used)
	for _, n := range spells {
		if !l.spell(n) {
			add(MultiattackPlanSpellMissing, n)
		}
	}
	return out
}

// MultiattackOption configures StartMultiattack.
type MultiattackOption func(*multiattackConfig)

type multiattackConfig struct {
	granted GrantedAction
	used    []LimitedUsePartRef
}

// WithGrantedAction sets the standard action the Multiattack names; the default is
// NoStandardAction.
func WithGrantedAction(g GrantedAction) MultiattackOption {
	return func(c *multiattackConfig) { c.granted = g }
}

// WithUsedThisTurn sets the parts the monster has used this turn, which select the variants that
// require one (SRD 5.2.1, "Clay Golem", p. 274). The refs MUST be valid and unique.
func WithUsedThisTurn(parts []LimitedUsePartRef) MultiattackOption {
	return func(c *multiattackConfig) { c.used = slices.Clone(parts) }
}

// attackReplaced is one replacement made: the option used and the attack part it replaced.
type attackReplaced struct {
	option   content.ReplacementOption
	replaced LimitedUsePartRef
}

// MultiattackProgress is what a monster has done with one Multiattack: the attacks made, the
// additional uses and the replacements, against the variants of its plan (SRD 5.2.1,
// "Multiattack", p. 257). The plan is a ceiling, not an obligation, and the steps may come in
// any order (monster-stat-block-actions.md FR-017, FR-019). Every step returns a new progress and
// leaves the receiver unchanged. It holds slices and maps, so compare it with Equal. The zero
// value is invalid; build one with StartMultiattack. Safe for concurrent use.
type MultiattackProgress struct {
	combatant CombatantID
	part      LimitedUsePartRef
	variants  []content.MultiattackVariant
	refs      map[string]LimitedUsePartRef
	spells    map[string]SpellUseRef
	initial   []int
	attacks   []LimitedUsePartRef
	extras    []LimitedUsePartRef
	replaced  []attackReplaced
}

// Combatants returns a new slice holding the monster combatant.
func (p MultiattackProgress) Combatants() []CombatantID { return []CombatantID{p.combatant} }

// Part returns the position of the Multiattack part.
func (p MultiattackProgress) Part() LimitedUsePartRef { return p.part }

// Attacks returns the parts the attacks made so far were made with, in order, as a new slice.
func (p MultiattackProgress) Attacks() []LimitedUsePartRef { return slices.Clone(p.attacks) }

// IsZero reports whether p is the invalid zero progress.
func (p MultiattackProgress) IsZero() bool { return p.combatant.IsZero() }

// Equal reports whether p and o hold the same plan and the same steps.
func (p MultiattackProgress) Equal(o MultiattackProgress) bool {
	return p.combatant == o.combatant && p.part == o.part && maps.Equal(p.refs, o.refs) && maps.Equal(p.spells, o.spells) &&
		slices.EqualFunc(p.variants, o.variants, content.MultiattackVariant.Equal) && slices.Equal(p.initial, o.initial) &&
		slices.Equal(p.attacks, o.attacks) && slices.Equal(p.extras, o.extras) && slices.Equal(p.replaced, o.replaced)
}

// MultiattackTransition is the change one step of a Multiattack makes: the new progress, the new
// limited-use state and the events. Immutable; every accessor returns a copy.
type MultiattackTransition struct {
	progress   MultiattackProgress
	limitedUse LimitedUseState
	events     []Event
}

// Progress returns the progress after the step.
func (t MultiattackTransition) Progress() MultiattackProgress { return t.progress }

// LimitedUse returns the limited-use state after the step.
func (t MultiattackTransition) LimitedUse() LimitedUseState { return t.limitedUse }

// Events returns the events in the order they happened, as a new slice: the limited-use events of
// the usage spent, then the step's own event.
func (t MultiattackTransition) Events() []Event { return slices.Clone(t.events) }

// MultiattackStartOutcome is the closed set of outcomes of StartMultiattack. Variants:
// MultiattackStarted, MultiattackNotAvailable.
type MultiattackStartOutcome interface{ isMultiattackStartOutcome() }

// MultiattackStarted: the monster takes a Multiattack. The transition is the use of the
// Multiattack part, which consumes the action.
type MultiattackStarted struct {
	progress   MultiattackProgress
	transition StatBlockUseTransition
}

// Progress returns the empty progress of the new Multiattack.
func (s MultiattackStarted) Progress() MultiattackProgress { return s.progress }

// Transition returns the use of the Multiattack part.
func (s MultiattackStarted) Transition() StatBlockUseTransition { return s.transition }

// MultiattackNotAvailable: the monster has no Multiattack part, or the part has no plan.
type MultiattackNotAvailable struct{}

func (MultiattackStarted) isMultiattackStartOutcome()      {}
func (MultiattackNotAvailable) isMultiattackStartOutcome() {}

// MultiattackStepOutcome is the closed set of outcomes of a step of a Multiattack. Variants:
// MultiattackStepRecorded, MultiattackNotInAllowance, MultiattackStepUsageRefused.
type MultiattackStepOutcome interface{ isMultiattackStepOutcome() }

// MultiattackStepRecorded: the step is within the plan and was recorded.
type MultiattackStepRecorded struct{ Transition MultiattackTransition }

// MultiattackNotInAllowance: the plan does not allow the step. Nothing changed.
type MultiattackNotInAllowance struct{ Reason MultiattackAllowanceRefusal }

// MultiattackStepUsageRefused: the plan allows the step but a part it uses has a limit that is
// spent. Nothing changed.
type MultiattackStepUsageRefused struct{ Refusal StatBlockPartUsageExhausted }

func (MultiattackStepRecorded) isMultiattackStepOutcome()     {}
func (MultiattackNotInAllowance) isMultiattackStepOutcome()   {}
func (MultiattackStepUsageRefused) isMultiattackStepOutcome() {}

// StartMultiattack starts a Multiattack of monster m for combatant (monster-stat-block-actions.md
// FR-010 to FR-012). The new progress holds the variants of the plan whose condition holds: one
// that requires a part used this turn only when that part is among the used parts (WithUsedThisTurn).
// The transition is the use of the Multiattack part as an action, naming it by FeatureID with the
// action of WithGrantedAction (NoStandardAction by default). Each Multiattack starts its own
// progress; counts never carry between turns. It returns MultiattackNotAvailable when m has no
// Multiattack part or the part has no plan.
//
// It returns an error matching srderr.ErrInvalidInput for the zero monster, combatant or
// limited-use state, a state of another combatant or monster, a granted action that is not valid,
// a used part that is zero or listed twice, a plan with a count outside the range combat relies on
// (plan P13), a plan with an unresolved name (see MultiattackPlanProblemsFor), a Multiattack part
// that has a usage limit (SRD 5.2.1 prints none), and the *StatBlockFeatureIDRejected of a
// Multiattack whose id cannot be derived, and a LimitExceededError (plan P10) for a stat block of more than 64 parts,
// spells or spell groups, or whose plans hold more than 64 variants, attack groups or replacement rules. It never
// panics.
func StartMultiattack(m content.Monster, combatant CombatantID, limitedUse LimitedUseState, opts ...MultiattackOption) (MultiattackStartOutcome, error) {
	if err := checkStatBlockState(m, combatant, limitedUse); err != nil {
		return nil, err
	}
	if err := monsterPlansInRange(m); err != nil {
		return nil, err
	}
	cfg := multiattackConfig{granted: NoStandardAction{}}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.granted == nil || !cfg.granted.valid() || slices.ContainsFunc(cfg.used, LimitedUsePartRef.IsZero) {
		return nil, invalidInput("a Multiattack needs a valid granted action and valid used parts")
	}
	if hasDuplicateRef(cfg.used) {
		return nil, invalidInput("the parts used this turn must be unique")
	}
	part, ok := m.Multiattack()
	plan, has := part.Plan()
	if !ok || !has {
		return MultiattackNotAvailable{}, nil
	}
	ref := LimitedUsePartRef{MonsterPartActions, slices.IndexFunc(m.Actions(), func(p content.StatBlockPart) bool { return p.Name() == content.MultiattackName })}
	if !monsterPlanOK(plan) || len(MultiattackPlanProblemsFor(m)) > 0 || limitedUse.AvailabilityOf(ref) != UseNotLimited {
		return nil, invalidInput("the Multiattack plan has a number out of range, a name that does not resolve or a usage limit")
	}
	feature, err := StatBlockFeatureID(m, ref)
	if err != nil {
		return nil, err
	}
	p := resolveMultiattack(m, combatant, ref, plan, cfg.used)
	t := StatBlockUseTransition{limitedUse: limitedUse, cost: StatBlockCostAction,
		action: FeatureActionChoice{Feature: feature, Granted: cfg.granted},
		events: []Event{StatBlockPartUsed{statBlockActionEventBase{source: combatant}, ref, StatBlockCostAction}}}
	return MultiattackStarted{p, t}, nil
}

// hasDuplicateRef reports whether refs lists a ref twice.
func hasDuplicateRef(refs []LimitedUsePartRef) bool {
	seen := map[LimitedUsePartRef]struct{}{}
	for _, r := range refs {
		if _, dup := seen[r]; dup {
			return true
		}
		seen[r] = struct{}{}
	}
	return false
}

// resolveMultiattack returns the empty progress of plan, resolving every part and spell name it
// uses against m and selecting the variants whose condition the used parts satisfy.
func resolveMultiattack(m content.Monster, combatant CombatantID, part LimitedUsePartRef, plan content.MultiattackPlan, used []LimitedUsePartRef) MultiattackProgress {
	p := MultiattackProgress{combatant: combatant, part: part, variants: plan.Variants(),
		refs: map[string]LimitedUsePartRef{}, spells: map[string]SpellUseRef{}}
	actions, required, spells := planReferences(plan)
	for _, n := range actions {
		p.refs[n] = actionRefs(m, n)[0]
	}
	for _, n := range required {
		p.refs[n] = usedRefs(m, n)[0]
	}
	for _, n := range spells {
		p.spells[n], _ = spellRef(m, n)
	}
	for i, v := range p.variants {
		if name, ok := v.RequiresUsed(); !ok || slices.Contains(used, p.refs[name]) {
			p.initial = append(p.initial, i)
		}
	}
	return p
}

// maxAssignNodes is the most picks one assign search tries (plan P10). The search backtracks, so it is exponential in
// the steps of a pathological plan; the budget ends it.
const maxAssignNodes = 1 << 14

// assign reports whether each of n items can take one of its options so that ok holds for every
// prefix of the picks made. The items are the steps of one Multiattack, a handful (at most 64). The search tries at
// most maxAssignNodes picks; one that runs out of them answers false, so a plan too tangled to settle refuses the
// step (a fail-closed shortcut: a stat block of SRD 5.2.1 needs a few picks).
func assign(n int, options func(item int) []int, ok func(picks []int) bool) bool {
	picks := make([]int, 0, n)
	nodes := 0
	var rec func(item int) bool
	rec = func(item int) bool {
		if item == n {
			return true
		}
		for _, o := range options(item) {
			if nodes++; nodes > maxAssignNodes {
				return false
			}
			picks = append(picks, o)
			if ok(picks) && rec(item+1) {
				return true
			}
			picks = picks[:len(picks)-1]
		}
		return false
	}
	return rec(0)
}

// maHistory is what has been recorded: the attacks, the additional uses and the replacements.
type maHistory struct {
	attacks  []LimitedUsePartRef
	extras   []LimitedUsePartRef
	replaced []attackReplaced
	// reserved asks for that many more attacks of one part, without listing them.
	reserved struct {
		part LimitedUsePartRef
		n    int64
	}
}

func (p MultiattackProgress) history() maHistory {
	return maHistory{attacks: p.attacks, extras: p.extras, replaced: p.replaced}
}

// groupOf returns the index of the group of groups that holds part, or -1.
func (p MultiattackProgress) groupOf(groups []content.AttackSlotGroup, part LimitedUsePartRef) int {
	return slices.IndexFunc(groups, func(g content.AttackSlotGroup) bool {
		return slices.ContainsFunc(g.Parts(), func(n string) bool {
			r, ok := p.refs[n]
			return ok && r == part
		})
	})
}

// ruleAllows reports whether rule r lists option and its scope allows replacing an attack of
// part in the variant's groups.
func (p MultiattackProgress) ruleAllows(groups []content.AttackSlotGroup, r content.ReplacementRule, o content.ReplacementOption, part LimitedUsePartRef) bool {
	if !slices.Contains(r.Options(), o) || p.groupOf(groups, part) < 0 {
		return false
	}
	if s, ok := r.Scope().(content.ReplacementScopeOneAttackPart); ok {
		return p.refs[s.Part()] == part
	}
	return true
}

// feasible reports whether variant v allows everything in h: every attack in a group with a slot
// for it, every additional use under its own entry, and every replacement under a rule, within the
// rule's limit and the slots of the replaced attack's group. With relaxSlots the slot counts are
// not checked (only the rules' limits), to tell a spent limit from too few attacks.
func (p MultiattackProgress) feasible(v content.MultiattackVariant, h maHistory, relaxSlots bool) bool {
	groups := v.Groups()
	used := make([]int64, len(groups))
	for _, a := range h.attacks {
		gi := p.groupOf(groups, a)
		if gi < 0 {
			return false
		}
		used[gi]++
	}
	if h.reserved.n > 0 { // only AttacksRemaining reserves, and only for a part of one of the variant's groups
		used[p.groupOf(groups, h.reserved.part)] += h.reserved.n
	}
	fits := func(slots []int64) bool {
		for gi, g := range groups {
			if slots[gi] > int64(g.Count()) {
				return false
			}
		}
		return true
	}
	if !relaxSlots && !fits(used) {
		return false
	}
	entries := v.AdditionalUses()
	entriesFor := func(i int) []int {
		var out []int
		for ei, e := range entries {
			if slices.ContainsFunc(e.Options(), func(n string) bool { r, ok := p.refs[n]; return ok && r == h.extras[i] }) {
				out = append(out, ei)
			}
		}
		return out
	}
	if !assign(len(h.extras), entriesFor, func(picks []int) bool { return !slices.Contains(picks[:len(picks)-1], picks[len(picks)-1]) }) {
		return false
	}
	rules := v.Replacements()
	rulesFor := func(i int) []int {
		var out []int
		for ri, r := range rules {
			if p.ruleAllows(groups, r, h.replaced[i].option, h.replaced[i].replaced) {
				out = append(out, ri)
			}
		}
		return out
	}
	return assign(len(h.replaced), rulesFor, func(picks []int) bool {
		last := picks[len(picks)-1]
		if rules[last].Limit() == content.ReplaceOnce && slices.Contains(picks[:len(picks)-1], last) {
			return false
		}
		slots := slices.Clone(used)
		for k, pick := range picks {
			slots[p.groupOf(groups, h.replaced[k].replaced)] += int64(rules[pick].Count())
		}
		return relaxSlots || fits(slots)
	})
}

// open returns the indexes of the variants still open: those of the start that allow all that
// has been recorded.
func (p MultiattackProgress) open(h maHistory) []int {
	return slices.DeleteFunc(slices.Clone(p.initial), func(i int) bool { return !p.feasible(p.variants[i], h, false) })
}

// AttacksRemaining returns the most attacks still allowed with the parts of the group that
// holds part, over the variants still open: the group's count less the attacks made with its
// parts and the slots its replacements used. It is 0 for a part no open variant lists as an attack.
// The plan is a ceiling, so a positive number is not an obligation.
func (p MultiattackProgress) AttacksRemaining(part LimitedUsePartRef) int {
	h := p.history()
	best := 0
	for _, i := range p.open(h) {
		v := p.variants[i]
		gi := p.groupOf(v.Groups(), part)
		if gi < 0 {
			continue
		}
		// Binary search the largest k such that k more attacks of part still fit; feasibility only falls as k grows.
		lo, hi := 0, v.Groups()[gi].Count()
		for range bits.Len(uint(hi)) + 1 { // a halving search ends within this many steps, so a bound, not a hope, ends the loop
			if lo >= hi {
				break
			}
			mid := lo + (hi-lo)/2 + (hi-lo)%2 // the upper middle, written so that hi-lo+1 never overflows a 32-bit int
			more := h
			more.reserved.part, more.reserved.n = part, int64(mid)
			if p.feasible(v, more, false) {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		best = max(best, lo)
	}
	return best
}

// checkStep returns an error matching srderr.ErrInvalidInput unless the step has a started progress and the limited-use state of its monster.
func (p MultiattackProgress) checkStep(limitedUse LimitedUseState) error {
	if p.IsZero() || limitedUse.IsZero() || limitedUse.combatant != p.combatant {
		return invalidInput("a Multiattack step needs a started progress and the limited-use state of its monster")
	}
	return nil
}

// stepRoom returns a LimitExceededError (plan P10) when one more step would make more than 64 steps, or, for a
// replacement, more than 64 replacements.
func (p MultiattackProgress) stepRoom(replacement bool) error {
	if replacement {
		if err := overCap("multiattack replacements", len(p.replaced)+1, maxMultiattackSteps); err != nil {
			return err
		}
	}
	return overCap("multiattack steps", len(p.attacks)+len(p.extras)+len(p.replaced)+1, maxMultiattackSteps)
}

// recordStep builds the outcome of an allowed step: it spends the parts in gates, in order, from
// limitedUse and returns the usage refusal if one is spent, else the new progress with event.
func recordStep(next MultiattackProgress, limitedUse LimitedUseState, gates []func(LimitedUseState) (LimitedUseState, []Event, bool), event Event) MultiattackStepOutcome {
	var events []Event
	for _, gate := range gates {
		var ok bool
		var spent []Event
		if limitedUse, spent, ok = gate(limitedUse); !ok {
			return MultiattackStepUsageRefused{}
		}
		events = append(events, spent...)
	}
	return MultiattackStepRecorded{MultiattackTransition{next, limitedUse, append(events, event)}}
}

func gatePart(part LimitedUsePartRef) func(LimitedUseState) (LimitedUseState, []Event, bool) {
	return func(s LimitedUseState) (LimitedUseState, []Event, bool) { return spendPart(s, part) }
}

// RecordAttack records an attack made with part (monster-stat-block-actions.md FR-013, FR-014,
// FR-015). It succeeds when some variant still open has a group holding part with an attack left;
// the variants still open narrow to those that allow everything recorded, so mixing attacks no
// single variant allows is refused. A part with a usage limit is spent through limitedUse. The
// plan's allowance is checked before the part's usage (FR-020): MultiattackNotInAllowance with
// MultiattackAttackNotAllowed when no open variant lists the part, MultiattackAllowanceSpent when
// one does and its attacks are made, then MultiattackStepUsageRefused. A refusal changes nothing.
//
// It returns an error matching srderr.ErrInvalidInput for the zero progress or limited-use state,
// or a state of another combatant, and a LimitExceededError (plan P10) when the progress already holds 64 steps. It
// never panics.
func (p MultiattackProgress) RecordAttack(part LimitedUsePartRef, limitedUse LimitedUseState) (MultiattackStepOutcome, error) {
	if err := p.checkStep(limitedUse); err != nil {
		return nil, err
	}
	if err := p.stepRoom(false); err != nil {
		return nil, err
	}
	h := p.history()
	listed := false
	for _, i := range p.open(h) {
		listed = listed || p.groupOf(p.variants[i].Groups(), part) >= 0
	}
	next := p
	next.attacks = append(slices.Clone(p.attacks), part)
	if len(p.open(next.history())) == 0 {
		if listed {
			return MultiattackNotInAllowance{MultiattackAllowanceSpent}, nil
		}
		return MultiattackNotInAllowance{MultiattackAttackNotAllowed}, nil
	}
	event := MultiattackAttackRecorded{statBlockActionEventBase{source: p.combatant}, part, len(next.attacks)}
	return recordStep(next, limitedUse, []func(LimitedUseState) (LimitedUseState, []Event, bool){gatePart(part)}, event), nil
}

// RecordAdditionalUse records the use of part as the additional use of an entry of the plan
// (monster-stat-block-actions.md FR-016): it consumes no attack, counts as its entry's one choice
// and spends the part's usage limit. MultiattackNotInAllowance with
// MultiattackAdditionalUseNotListed when no open variant lists part, or
// MultiattackAdditionalUseAlreadyMade when its entry was already used; a refusal changes nothing.
//
// It returns the errors RecordAttack does. It never panics.
func (p MultiattackProgress) RecordAdditionalUse(part LimitedUsePartRef, limitedUse LimitedUseState) (MultiattackStepOutcome, error) {
	if err := p.checkStep(limitedUse); err != nil {
		return nil, err
	}
	if err := p.stepRoom(false); err != nil {
		return nil, err
	}
	listed := false
	for _, i := range p.open(p.history()) {
		for _, e := range p.variants[i].AdditionalUses() {
			listed = listed || slices.ContainsFunc(e.Options(), func(n string) bool { r, ok := p.refs[n]; return ok && r == part })
		}
	}
	next := p
	next.extras = append(slices.Clone(p.extras), part)
	switch {
	case !listed:
		return MultiattackNotInAllowance{MultiattackAdditionalUseNotListed}, nil
	case len(p.open(next.history())) == 0:
		return MultiattackNotInAllowance{MultiattackAdditionalUseAlreadyMade}, nil
	}
	event := MultiattackAdditionalUseRecorded{statBlockActionEventBase{source: p.combatant}, part}
	return recordStep(next, limitedUse, []func(LimitedUseState) (LimitedUseState, []Event, bool){gatePart(part)}, event), nil
}

// RecordReplacement records replacing the attack of the part replaced with a use of option
// (monster-stat-block-actions.md FR-018). It succeeds when some variant still open has a rule that
// lists option, whose scope allows replaced (any attack, or one named part), whose limit is not
// reached (a "once" rule applies once, an "any number" rule always) and whose count of attacks is
// still free in the group of replaced; it consumes that many attacks of the group. It spends the
// usage limit of the option's part and, when the option casts a spell, of that spell. A replaced
// attack is not an attack made. The refusals are MultiattackReplacementNotListed,
// MultiattackReplacementLimitReached and MultiattackNotEnoughAttacksToReplace, in that order of
// progress; with several open variants the furthest is reported. A refusal changes nothing.
//
// It returns the errors RecordAttack does, and a LimitExceededError for a progress that already holds 64
// replacements. It never panics.
func (p MultiattackProgress) RecordReplacement(option content.ReplacementOption, replaced LimitedUsePartRef, limitedUse LimitedUseState) (MultiattackStepOutcome, error) {
	if err := p.checkStep(limitedUse); err != nil {
		return nil, err
	}
	if err := p.stepRoom(true); err != nil {
		return nil, err
	}
	next := p
	next.replaced = append(slices.Clone(p.replaced), attackReplaced{option, replaced})
	reason, allowed := MultiattackReplacementNotListed, false
	for _, i := range p.open(p.history()) {
		v := p.variants[i]
		groups := v.Groups()
		if !slices.ContainsFunc(v.Replacements(), func(r content.ReplacementRule) bool { return p.ruleAllows(groups, r, option, replaced) }) {
			continue
		}
		if allowed = p.feasible(v, next.history(), false); allowed {
			break
		}
		// A rule lists the option: the limit is what stops the step unless the slots do.
		stopped := MultiattackNotEnoughAttacksToReplace
		if !p.feasible(v, next.history(), true) {
			stopped = MultiattackReplacementLimitReached
		}
		reason = max(reason, stopped)
	}
	if !allowed {
		return MultiattackNotInAllowance{reason}, nil
	}
	gates := []func(LimitedUseState) (LimitedUseState, []Event, bool){gatePart(p.refs[option.Part()])}
	if name, _, ok := option.Spell(); ok {
		spell := p.spells[name]
		gates = append(gates, func(s LimitedUseState) (LimitedUseState, []Event, bool) { return spendSpell(s, spell) })
	}
	event := MultiattackAttackReplaced{statBlockActionEventBase{source: p.combatant}, p.refs[option.Part()], replaced}
	return recordStep(next, limitedUse, gates, event), nil
}
