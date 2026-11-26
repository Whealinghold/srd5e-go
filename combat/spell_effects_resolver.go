package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// spellRolls is what the resolver asks of the use case: dice and the pure D20 Test
// functions. It is implemented by attackFlow; the resolver never sees a roller.
type spellRolls interface {
	// rollAmount rolls e and returns the face of each die and the total with e's flat modifier.
	rollAmount(e dice.Expression) (faces []int, total int, err error)
	// attackRoll makes one spell attack roll against armorClass.
	attackRoll(circumstances []modifier.RollCircumstance, mods d20.RollModifiers, armorClass d20.ArmorClass, criticalThreshold int) (d20.AttackRollResult, error)
	// rollSpellDamage rolls and composes the damage of spec, returning the faces rolled per component.
	rollSpellDamage(spec DamageSpec, abilityModifier int, critical bool) ([]Damage, [][]int, error)
	// resolveSave makes one saving throw against dc.
	resolveSave(a d20.SaveAttempt, dc d20.DifficultyClass) (UnarmedStrikeSave, error)
}

// spellCast is everything the resolver reads: the command of ResolveSpellEffects with the
// model of the spell.
type spellCast struct {
	spell          content.Spell
	model          content.ModelledEffects
	report         character.CastReport
	caster         CombatantID
	attackMods     d20.RollModifiers
	abilityMod     int
	targets        []SpellTarget
	assignments    []InstanceAssignment
	chosen         content.DamageType // zero: none supplied
	partIndexes    []int
	partIndexesSet bool
	instanceCount  int
}

// spellResolution is the output of resolveSpell: either reasons to reject, or the evidence
// and the plan.
type spellResolution struct {
	rejection []SpellEffectsRejectionKind
	parts     []ResolvedPart
	targets   []TargetEffects
	valid     int
}

// spellPartPlan is a part to resolve with its instance count and the targets of each
// instance, as indexes into the command's targets.
type spellPartPlan struct {
	index     int
	part      content.SpellEffectPart
	instances int
	assign    [][]int
	badAssign bool
}

// spellPlan is the validated cast: the parts to resolve and what each target is.
type spellPlan struct {
	c         spellCast
	targeting content.SpellTargeting
	castAt    int
	charLevel int
	parts     []spellPartPlan
	chosenSet []content.DamageType
	wrong     []bool // the target is of the wrong subject
	invalid   []bool // the spell cannot affect the target (SRD 5.2.1, "Invalid Targets", p. 106)
}

// resolveSpell resolves a cast of a modelled spell (spell-effects.md FR-015 to FR-065). It
// returns an error matching ErrInvalidSpellEffectsCommand for contradictory input, a
// resolution with rejection reasons (before any die is rolled), or the plan. A roller
// failure is returned as the roller returned it.
func resolveSpell(c spellCast, rolls spellRolls) (spellResolution, error) {
	p, err := newSpellPlan(c)
	if err != nil {
		return spellResolution{}, err
	}
	if r := p.rejections(); len(r) > 0 {
		return spellResolution{rejection: r}, nil
	}
	return p.run(rolls)
}

// slotSteps is n of spell-effects.md FR-043: the larger of 0 and castAt minus above, and 0 for
// a cast with no slot level.
func slotSteps(castAt, above int) int {
	if castAt == 0 {
		return 0
	}
	return max(0, castAt-above)
}

func countAbove(s content.CountScaling) int {
	if v, ok := s.(content.PerSlotLevelCount); ok {
		return v.AboveLevel()
	}
	return 0
}

func amountAbove(s content.AmountScaling) int {
	if v, ok := s.(content.PerSlotLevelAmount); ok {
		return v.AboveLevel()
	}
	return 0
}

// resolutionOutcomes returns every outcome of a part resolution.
func resolutionOutcomes(r content.PartResolution) []content.SpellOutcome {
	switch v := r.(type) {
	case content.AttackRollResolution:
		return slices.Concat(v.OnHit(), v.OnMiss())
	case content.SavingThrowResolution:
		return v.OnFailure()
	case content.AutomaticResolution:
		return v.Outcomes()
	}
	return nil
}

// outcomeAmount returns the scalable amount of an outcome that has one.
func outcomeAmount(o content.SpellOutcome) (content.ScalableAmount, content.ModifierAddition, bool) {
	switch v := o.(type) {
	case content.SpellDealsDamage:
		return v.Amount(), v.AddsModifier(), true
	case content.SpellRestoresHitPoints:
		return v.Amount(), v.AddsModifier(), true
	case content.SpellGrantsTemporaryHitPoints:
		return v.Amount(), v.AddsModifier(), true
	}
	return content.ScalableAmount{}, 0, false
}

// needsCharacterLevel reports whether any count or amount of the model is read from a table of
// the character level.
func needsCharacterLevel(m content.ModelledEffects) bool {
	isTable := func(s content.CountScaling) bool { _, ok := s.(content.ByCharacterLevelCount); return ok }
	if t, ok := m.Targeting().(content.TargetsChosen); ok {
		switch n := t.Count().(type) {
		case content.ExactlyTargets:
			if isTable(n.Scaling()) {
				return true
			}
		case content.UpToTargets:
			if isTable(n.Scaling()) {
				return true
			}
		}
	}
	for _, part := range m.Parts() {
		if isTable(part.Instances().Scaling()) {
			return true
		}
		for _, o := range resolutionOutcomes(part.Resolution()) {
			if a, _, ok := outcomeAmount(o); ok {
				if _, table := a.Scaling().(content.ByCharacterLevelAmount); table {
					return true
				}
			}
		}
	}
	return false
}

// chosenDamageTypes returns the Damage Types the caster may choose from in the model, or none.
func chosenDamageTypes(m content.ModelledEffects) []content.DamageType {
	var out []content.DamageType
	for _, part := range m.Parts() {
		for _, o := range resolutionOutcomes(part.Resolution()) {
			if d, ok := o.(content.SpellDealsDamage); ok {
				if c, ok := d.Type().(content.ChosenDamageType); ok {
					for _, t := range c.Types() {
						if !slices.Contains(out, t) {
							out = append(out, t)
						}
					}
				}
			}
		}
	}
	return out
}

// newSpellPlan validates c and works out the parts to resolve (spell-effects.md FR-015, FR-033,
// FR-050, FR-098). The returned error matches ErrInvalidSpellEffectsCommand.
func newSpellPlan(c spellCast) (*spellPlan, error) {
	model := c.model
	if c.caster.IsZero() {
		return nil, spellCommandError("no caster", nil)
	}
	seen := map[SpellTargetRef]bool{}
	for _, t := range c.targets {
		if !validSpellTargetRef(t.ref) || seen[t.ref] {
			return nil, spellCommandError("a target without a ref or listed twice", nil)
		}
		seen[t.ref] = true
	}
	if c.report.Spell() != c.spell.ID() {
		return nil, spellCommandError("the cast report is of another spell", nil)
	}
	castAt, hasCast := c.report.CastAtLevel()
	if hasCast && castAt < int(c.spell.Level()) {
		return nil, spellCommandError("cast below the spell's level", nil)
	}
	charLevel, hasChar := c.report.ScalingLevel()
	if needsCharacterLevel(model) && !hasChar {
		return nil, spellCommandError("the spell scales with the character level and the report has none", nil)
	}
	p := &spellPlan{c: c, targeting: model.Targeting(), castAt: castAt, charLevel: charLevel, chosenSet: chosenDamageTypes(model)}
	if (c.chosen != 0 && !c.chosen.Valid()) || (c.chosen != 0 && len(p.chosenSet) == 0) {
		return nil, spellCommandError("a Damage Type chosen for a spell that offers none", nil)
	}
	all := model.Parts()
	parts := map[int]bool{}
	for _, a := range c.assignments {
		if a.part >= len(all) || parts[a.part] {
			return nil, spellCommandError("an assignment for a part that does not exist, or two for one part", nil)
		}
		parts[a.part] = true
	}
	run, err := p.selectParts(all)
	if err != nil {
		return nil, err
	}
	for _, i := range run {
		if err := p.addPart(i, all[i]); err != nil {
			return nil, err
		}
	}
	p.classifyTargets()
	return p, nil
}

// selectParts returns the indexes of the parts to resolve in model order.
func (p *spellPlan) selectParts(all []content.SpellEffectPart) ([]int, error) {
	c := p.c
	if c.instanceCount < 0 || (c.instanceCount > 0 && !c.partIndexesSet) {
		return nil, spellCommandError("an instance count without part indexes, or a negative one", nil)
	}
	var run []int
	if c.partIndexesSet {
		if len(c.partIndexes) == 0 {
			return nil, spellCommandError("an empty set of part indexes", nil)
		}
		for _, i := range c.partIndexes {
			if i < 0 || i >= len(all) || slices.Contains(run, i) {
				return nil, spellCommandError("a part index out of range or listed twice", nil)
			}
			run = append(run, i)
		}
		slices.Sort(run)
		return run, nil
	}
	var skip []int
	if z, ok := c.model.Zone(); ok {
		for _, r := range z.Rules() {
			skip = append(skip, r.PartIndex())
		}
	}
	for i := range all {
		if !slices.Contains(skip, i) {
			run = append(run, i)
		}
	}
	return run, nil
}

// addPart adds part i to the plan with its instance count and the targets of each instance.
func (p *spellPlan) addPart(i int, part content.SpellEffectPart) error {
	c := p.c
	pp := spellPartPlan{index: i, part: part}
	inst := part.Instances()
	if c.instanceCount > 0 {
		pp.instances = c.instanceCount
	} else {
		n, err := inst.At(slotSteps(p.castAt, countAbove(inst.Scaling())), p.charLevel)
		if err != nil {
			return spellCommandError("the number of instances cannot be scaled", err)
		}
		pp.instances = n
	}
	if pp.instances < 1 {
		return spellCommandError("a part with no instance", nil)
	}
	if pp.instances > maxSpellEffectInstances {
		return spellCommandError("a part with more instances than the cap", LimitExceededError{Collection: "spell effect instances", Max: maxSpellEffectInstances})
	}
	switch part.Resolution().(type) {
	case content.SavingThrowResolution:
		if _, ok := c.report.SaveDC(); !ok {
			return spellCommandError("a saving throw part and no spell save DC", nil)
		}
	case content.AttackRollResolution:
		if _, ok := c.report.AttackBonus(); !ok {
			return spellCommandError("a spell attack part and no spell attack modifier", nil)
		}
	}
	p.parts = append(p.parts, pp)
	return nil
}

// isAttack reports whether the part is a spell attack roll.
func (pp spellPartPlan) isAttack() bool {
	_, ok := pp.part.Resolution().(content.AttackRollResolution)
	return ok
}

// classifyTargets decides which targets are of the wrong subject and which the spell cannot
// affect, and assigns the targets to the instances of each part.
func (p *spellPlan) classifyTargets() {
	c := p.c
	var subject content.TargetSubject
	var restriction content.CreatureTypeRestriction = content.AnyCreatureType{}
	switch t := p.targeting.(type) {
	case content.TargetsChosen:
		subject, restriction = t.Subject(), t.Restriction()
	case content.TargetsInArea:
		subject, restriction = t.Subject(), t.Restriction()
	}
	p.wrong = make([]bool, len(c.targets))
	p.invalid = make([]bool, len(c.targets))
	for i, t := range c.targets {
		_, creature := t.ref.(CreatureTargetRef)
		p.wrong[i] = (creature && subject == content.SubjectObject) || (!creature && subject == content.SubjectCreature)
		only, restricted := restriction.(content.OnlyCreatureTypes)
		outside := restricted && (!creature || !only.Contains(t.typ))
		p.invalid[i] = !p.wrong[i] && (t.validity == TargetCannotBeAffected || outside)
	}
	for k := range p.parts {
		p.assignInstances(&p.parts[k])
	}
}

// assignInstances works out the targets of each instance of pp (spell-effects.md FR-031).
func (p *spellPlan) assignInstances(pp *spellPartPlan) {
	c := p.c
	pp.assign = make([][]int, pp.instances)
	var given *InstanceAssignment
	for k := range c.assignments {
		if c.assignments[k].part == pp.index {
			given = &c.assignments[k]
		}
	}
	needs := pp.isAttack() || pp.instances > 1
	switch {
	case len(c.targets) == 0:
	case !needs:
		pp.badAssign = given != nil
		for i := range c.targets {
			pp.assign[0] = append(pp.assign[0], i)
		}
	case len(c.targets) == 1:
		for k := range pp.assign {
			pp.assign[k] = []int{0}
		}
	case given == nil || len(given.targets) != pp.instances:
		pp.badAssign = true
	default:
		covered := make([]bool, len(c.targets))
		for k, ref := range given.targets {
			at := slices.IndexFunc(c.targets, func(t SpellTarget) bool { return t.ref == ref })
			if at < 0 {
				pp.badAssign = true
				return
			}
			covered[at] = true
			pp.assign[k] = []int{at}
		}
		pp.badAssign = slices.Contains(covered, false)
	}
}

// hitSpec builds the DamageSpec of the damage outcomes of outcomes, in outcome order, and the
// index of each component's outcome. It returns the zero spec when there is no damage.
func (p *spellPlan) hitSpec(outcomes []content.SpellOutcome) (DamageSpec, []int, error) {
	var comps []DamageComponent
	var which []int
	for i, o := range outcomes {
		d, ok := o.(content.SpellDealsDamage)
		if !ok {
			continue
		}
		amt, err := d.Amount().At(slotSteps(p.castAt, amountAbove(d.Amount().Scaling())), p.charLevel)
		if err != nil {
			return DamageSpec{}, nil, spellCommandError("an amount cannot be scaled", err)
		}
		adds := d.AddsModifier() == content.AddsSpellcastingAbilityModifier
		t := p.damageType(d.Type())
		var comp DamageComponent
		switch a := amt.(type) {
		case content.RolledSpellAmount:
			comp, err = NewDiceDamage(a.Roll().Count(), a.Roll().Die(), t, a.Roll().Modifier(), adds)
		case content.FixedSpellAmount:
			comp, err = NewFixedDamage(a.Value(), t, 0, adds)
		}
		if err != nil {
			return DamageSpec{}, nil, spellCommandError("damage cannot be composed", err)
		}
		comps = append(comps, comp)
		which = append(which, i)
	}
	if len(comps) == 0 {
		return DamageSpec{}, nil, nil
	}
	spec, err := NewDamageSpec(comps...)
	if err != nil {
		return DamageSpec{}, nil, spellCommandError("damage cannot be composed", err)
	}
	return spec, which, nil
}

// damageType returns the Damage Type of a selection: the fixed one or the caster's choice.
func (p *spellPlan) damageType(s content.DamageTypeSelection) content.DamageType {
	if f, ok := s.(content.FixedDamageType); ok {
		return f.Type()
	}
	return p.c.chosen
}

// attackForm returns the spell attack form of kind.
func attackFormOf(k content.SpellAttackKind) AttackForm {
	if k == content.SpellAttackMelee {
		return MeleeSpellAttack
	}
	return RangedSpellAttack
}

// attackPlan returns what CircumstanceCollector decides about the attack of res against facts.
func (p *spellPlan) attackPlan(res content.AttackRollResolution, f SpellAttackFactsParams) (AttackPlan, Cover, error) {
	spec, _, err := p.hitSpec(res.OnHit())
	if err != nil {
		return nil, 0, err
	}
	cover := CoverResolver{}.Applicable(f.CoverSources)
	return CircumstanceCollector{}.Collect(attackFormOf(res.Kind()), f.Distance, f.Range, f.Reach, cover, f.NearbyEnemies, f.Situation, spec), cover, nil
}

// rejections returns every reason to refuse the command, in the fixed order (FR-057).
func (p *spellPlan) rejections() []SpellEffectsRejectionKind {
	c := p.c
	var set [SpellAttackNotPossible + 1]bool
	creatures := 0
	for i, t := range c.targets {
		if _, ok := t.ref.(CreatureTargetRef); ok {
			creatures++
		}
		if p.wrong[i] {
			set[WrongTargetSubject] = true
		}
	}
	if c.partIndexesSet {
		set[WrongTargetCount] = creatures == 0
	} else {
		p.checkTargeting(&set)
	}
	if _, ok := p.targeting.(content.TargetsChosen); ok {
		for _, tg := range c.targets {
			if tg.path == PathBlockedByTotalCover {
				set[NoClearPath] = true
			}
		}
	}
	if len(p.chosenSet) > 0 {
		switch {
		case c.chosen == 0:
			set[DamageTypeNotChosen] = true
		case !slices.Contains(p.chosenSet, c.chosen):
			set[DamageTypeNotAllowed] = true
		}
	}
	for _, pp := range p.parts {
		p.checkPart(pp, &set)
	}
	var out []SpellEffectsRejectionKind
	for k := WrongTargetCount; k <= SpellAttackNotPossible; k++ {
		if set[k] {
			out = append(out, k)
		}
	}
	return out
}

// checkTargeting checks the number of targets and the caster against the targeting.
func (p *spellPlan) checkTargeting(set *[SpellAttackNotPossible + 1]bool) {
	c := p.c
	switch t := p.targeting.(type) {
	case content.TargetsSelf:
		if len(c.targets) != 1 || c.targets[0].ref != (CreatureTargetRef{c.caster}) {
			set[WrongTargetCount] = true
		}
	case content.TargetsChosen:
		switch n := t.Count().(type) {
		case content.ExactlyTargets:
			want, _ := n.At(slotSteps(p.castAt, countAbove(n.Scaling())), p.charLevel)
			set[WrongTargetCount] = len(c.targets) != want
		case content.UpToTargets:
			most, _ := n.At(slotSteps(p.castAt, countAbove(n.Scaling())), p.charLevel)
			set[WrongTargetCount] = len(c.targets) < 1 || len(c.targets) > most
		}
		if t.SelfPolicy() == content.ExcludesCaster && slices.ContainsFunc(c.targets, func(s SpellTarget) bool { return s.ref == CreatureTargetRef{c.caster} }) {
			set[CasterNotAllowed] = true
		}
	}
}

// checkPart checks the assignment and the facts of the valid targets of one part.
func (p *spellPlan) checkPart(pp spellPartPlan, set *[SpellAttackNotPossible + 1]bool) {
	if pp.badAssign {
		set[InstanceAssignmentMismatch] = true
	}
	for _, inst := range pp.assign {
		for _, at := range inst {
			if p.wrong[at] || p.invalid[at] {
				continue
			}
			t := p.c.targets[at]
			switch r := pp.part.Resolution().(type) {
			case content.SavingThrowResolution:
				if _, ok := t.ref.(ObjectTargetRef); ok {
					set[MissingSaveAttempt] = true
				} else if _, ok := t.Save(r.Ability()); !ok {
					set[MissingSaveAttempt] = true
				}
			case content.AttackRollResolution:
				f, ok := t.AttackFacts()
				if !ok {
					set[MissingAttackFacts] = true
					continue
				}
				plan, _, err := p.attackPlan(r, f.p)
				if _, notPossible := plan.(AttackPlanNotPossible); err == nil && notPossible {
					set[SpellAttackNotPossible] = true
				}
			}
		}
	}
}

// run rolls and resolves every part in order (FR-041) and assembles the plan.
func (p *spellPlan) run(rolls spellRolls) (spellResolution, error) {
	out := spellResolution{targets: make([]TargetEffects, len(p.c.targets))}
	for i, t := range p.c.targets {
		out.targets[i].target = t.ref
		if !p.invalid[i] {
			out.valid++
		}
	}
	for _, pp := range p.parts {
		var part ResolvedPart
		var err error
		switch r := pp.part.Resolution().(type) {
		case content.SavingThrowResolution:
			part, err = p.runSave(rolls, pp, r, out.targets)
		case content.AttackRollResolution:
			part, err = p.runAttack(rolls, pp, r, out.targets)
		case content.AutomaticResolution:
			part, err = p.runAutomatic(rolls, pp, r, out.targets)
		}
		if err != nil {
			return spellResolution{}, err
		}
		out.parts = append(out.parts, part)
	}
	return out, nil
}

// validOf returns the indexes of the valid targets among at, and records the perception of the
// others (the first part to meet an invalid target decides it).
func (p *spellPlan) validOf(at []int, perception InvalidTargetPerception, effects []TargetEffects) []int {
	var valid []int
	for _, i := range at {
		switch {
		case !p.invalid[i]:
			valid = append(valid, i)
		case effects[i].perception == 0:
			effects[i].perception = perception
		}
	}
	return valid
}

// refsOf returns the refs of the targets at.
func (p *spellPlan) refsOf(at []int) []SpellTargetRef {
	var refs []SpellTargetRef
	for _, i := range at {
		refs = append(refs, p.c.targets[i].ref)
	}
	return refs
}

// evaluated is an outcome with the amount rolled for it, when it has one.
type evaluated struct {
	outcome content.SpellOutcome
	rolled  RolledAmount
	amount  bool
}

// evaluate rolls the amount of every outcome that has one, in outcome order (FR-032).
func (p *spellPlan) evaluate(rolls spellRolls, outcomes []content.SpellOutcome) ([]evaluated, error) {
	out := make([]evaluated, len(outcomes))
	for i, o := range outcomes {
		out[i].outcome = o
		a, adds, ok := outcomeAmount(o)
		if !ok {
			continue
		}
		amt, err := a.At(slotSteps(p.castAt, amountAbove(a.Scaling())), p.charLevel)
		if err != nil {
			return nil, spellCommandError("an amount cannot be scaled", err)
		}
		var faces []int
		var total int
		switch v := amt.(type) {
		case content.RolledSpellAmount:
			if faces, total, err = rolls.rollAmount(v.Roll()); err != nil {
				return nil, err
			}
		case content.FixedSpellAmount:
			total = v.Value()
		}
		if adds == content.AddsSpellcastingAbilityModifier {
			total = satAdd(total, p.c.abilityMod)
		}
		out[i].rolled = RolledAmount{outcome: i, faces: faces, amount: max(total, 0)}
		out[i].amount = true
	}
	return out, nil
}

// amountsOf returns the evidence of the amounts of evs.
func amountsOf(evs []evaluated) []RolledAmount {
	var out []RolledAmount
	for _, e := range evs {
		if e.amount {
			out = append(out, e.rolled)
		}
	}
	return out
}

// apply adds the plan of evs for one target: the damage as one PlannedDamage (halved when half),
// and, unless damageOnly, the healing, Temporary Hit Points and conditions.
func (p *spellPlan) apply(t *TargetEffects, part, instance int, evs []evaluated, half, damageOnly bool, kind HitKind) error {
	var damages []Damage
	for _, e := range evs {
		switch o := e.outcome.(type) {
		case content.SpellDealsDamage:
			d, err := NewDamage(e.rolled.amount, p.damageType(o.Type()))
			if err != nil {
				return err
			}
			damages = append(damages, d)
		case content.SpellRestoresHitPoints:
			if !damageOnly && e.rolled.amount > 0 {
				t.healing = append(t.healing, PlannedAmount{part, instance, e.rolled.amount})
			}
		case content.SpellGrantsTemporaryHitPoints:
			if !damageOnly && e.rolled.amount > 0 {
				t.temporary = append(t.temporary, PlannedAmount{part, instance, e.rolled.amount})
			}
		case content.SpellAppliesCondition:
			if !damageOnly {
				t.conditions = append(t.conditions, ConditionToApply{t.target, o.Kind(), p.c.caster, o.Duration(), part})
			}
		}
	}
	if len(damages) > 0 {
		t.damage = append(t.damage, PlannedDamage{part, instance, damages, half, kind})
	}
	return nil
}

// runSave resolves a saving throw part: per instance, the amounts are rolled once for all its
// targets, then each valid target saves (FR-024, FR-025, FR-028).
func (p *spellPlan) runSave(rolls spellRolls, pp spellPartPlan, r content.SavingThrowResolution, effects []TargetEffects) (ResolvedPart, error) {
	perception := SpellDidNothing
	if r.OnSuccess() == content.NothingOnSuccess {
		perception = AppearedToSucceed
	}
	part := ResolvedPart{index: pp.index, save: r.Ability()}
	dc, _ := p.c.report.SaveDC()
	for k, all := range pp.assign {
		valid := p.validOf(all, perception, effects)
		inst := ResolvedInstance{index: k, targets: p.refsOf(all)}
		if len(valid) > 0 {
			evs, err := p.evaluate(rolls, r.OnFailure())
			if err != nil {
				return ResolvedPart{}, err
			}
			inst.amounts = amountsOf(evs)
			for _, at := range valid {
				attempt, _ := p.c.targets[at].Save(r.Ability())
				save, err := rolls.resolveSave(attempt, dc.DifficultyClass())
				if err != nil {
					return ResolvedPart{}, err
				}
				inst.saves = append(inst.saves, ResolvedSave{p.c.targets[at].ref, save})
				saved := save.Outcome() == d20.Success
				if saved && r.OnSuccess() == content.NothingOnSuccess {
					continue
				}
				if err := p.apply(&effects[at], pp.index, k, evs, saved, saved, NormalHit); err != nil {
					return ResolvedPart{}, err
				}
			}
		}
		part.instances = append(part.instances, inst)
	}
	return part, nil
}

// runAutomatic resolves a part with no roll to hit: the amounts are rolled once per instance
// and shared by its targets (FR-029).
func (p *spellPlan) runAutomatic(rolls spellRolls, pp spellPartPlan, r content.AutomaticResolution, effects []TargetEffects) (ResolvedPart, error) {
	part := ResolvedPart{index: pp.index}
	for k, all := range pp.assign {
		valid := p.validOf(all, SpellDidNothing, effects)
		inst := ResolvedInstance{index: k, targets: p.refsOf(all)}
		if len(valid) > 0 {
			evs, err := p.evaluate(rolls, r.Outcomes())
			if err != nil {
				return ResolvedPart{}, err
			}
			inst.amounts = amountsOf(evs)
			for _, at := range valid {
				if err := p.apply(&effects[at], pp.index, k, evs, false, false, NormalHit); err != nil {
					return ResolvedPart{}, err
				}
			}
		}
		part.instances = append(part.instances, inst)
	}
	return part, nil
}

// runAttack resolves a spell attack part: each instance makes its own attack roll and, on a
// hit, rolls its own damage (FR-022, FR-023, FR-030).
func (p *spellPlan) runAttack(rolls spellRolls, pp spellPartPlan, r content.AttackRollResolution, effects []TargetEffects) (ResolvedPart, error) {
	part := ResolvedPart{index: pp.index}
	for k, all := range pp.assign {
		inst := ResolvedInstance{index: k, targets: p.refsOf(all)}
		if valid := p.validOf(all, SpellDidNothing, effects); len(valid) > 0 {
			at := valid[0]
			facts, _ := p.c.targets[at].AttackFacts()
			plan, cover, err := p.attackPlan(r, facts.p)
			if err == nil {
				err = p.attackInstance(rolls, pp, r, facts.p, plan, cover, &effects[at], &inst)
			}
			if err != nil {
				return ResolvedPart{}, err
			}
		}
		part.instances = append(part.instances, inst)
	}
	return part, nil
}

// attackInstance makes one attack roll against f, as plan decides, and applies the outcomes of the hit or the
// miss. plan is never AttackPlanNotPossible here: that was refused before any die.
func (p *spellPlan) attackInstance(rolls spellRolls, pp spellPartPlan, r content.AttackRollResolution, f SpellAttackFactsParams,
	plan AttackPlan, cover Cover, t *TargetEffects, inst *ResolvedInstance) error {
	outcomes, critical := r.OnMiss(), false
	switch v := plan.(type) {
	case AttackPlanAutomaticMiss:
		inst.miss = v.Reason
	case AttackPlanProceeds:
		circumstances := slices.Concat(v.Circumstances, f.Circumstances)
		atk, err := rolls.attackRoll(circumstances, p.c.attackMods, effectiveArmorClass(f.ArmorClass, cover), 0)
		if err != nil {
			return err
		}
		inst.attack, inst.rolled = atk, true
		inst.hit = atk.Hit() && v.MissOverride == MissOverrideNone
		inst.critical = inst.hit && (atk.IsCriticalHit() || f.AutomaticCritical == AutomaticCriticalAutomatic)
		if inst.hit {
			outcomes, critical = r.OnHit(), inst.critical
		}
	default:
		panic(unreachableError("attack plan", plan))
	}
	spec, which, err := p.hitSpec(outcomes)
	if err != nil {
		return err
	}
	evs := make([]evaluated, len(outcomes))
	if len(which) > 0 {
		damages, faces, err := rolls.rollSpellDamage(spec, p.c.abilityMod, critical)
		if err != nil {
			return err
		}
		for c, i := range which {
			evs[i] = evaluated{outcomes[i], RolledAmount{outcome: i, faces: faces[c], amount: damages[c].Amount()}, true}
		}
	}
	var others []content.SpellOutcome
	for _, o := range outcomes {
		if _, damage := o.(content.SpellDealsDamage); !damage {
			others = append(others, o)
		}
	}
	rest, err := p.evaluate(rolls, others)
	if err != nil {
		return err
	}
	next := 0
	for i, o := range outcomes {
		if _, damage := o.(content.SpellDealsDamage); damage {
			evs[i].outcome = o
			continue
		}
		evs[i] = rest[next]
		evs[i].rolled.outcome = i
		next++
	}
	inst.amounts = amountsOf(evs)
	kind := NormalHit
	if critical {
		kind = CriticalHit
	}
	return p.apply(t, pp.index, inst.index, evs, false, false, kind)
}
