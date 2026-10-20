package content_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: spell-effect-data.md FR-001, FR-024, FR-008 - TextOnlyEffects
// is a valid model with no data: every fact unknown, any level accepted, equal
// only to itself.
// Path: happy.
func TestTextOnlyEffects(t *testing.T) {
	t.Parallel()
	var m content.SpellEffectModel = content.TextOnlyEffects{}
	for level := 0; level <= 9; level++ {
		if err := m.CheckSpellLevel(level); err != nil {
			t.Errorf("CheckSpellLevel(%d) = %v", level, err)
		}
	}
	if !m.Equal(content.TextOnlyEffects{}) || m.Equal(fireBolt(content.CoveragePartial)) {
		t.Error("TextOnlyEffects equality is wrong")
	}
	if m != content.SpellEffectModel(content.TextOnlyEffects{}) {
		t.Error("TextOnlyEffects is not comparable with ==")
	}
}

// Requirement: spell-effect-data.md FR-024, table F12 - TextOnlyEffects (and
// the zero facts) answer FactUnknown to every fact.
// Path: unhappy (missing content).
func TestTextOnlyFactsAreUnknown(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]content.SpellEffectFacts{
		"text only": content.TextOnlyEffects{}.Facts(),
		"zero":      {},
	} {
		for fact, got := range map[string]content.EffectFact{
			"ForcesSavingThrow": f.ForcesSavingThrow(), "MakesSpellAttackRoll": f.MakesSpellAttackRoll(),
			"DealsDamage": f.DealsDamage(), "DamageTypes": f.DamageTypes(), "CanTargetMoreWithHigherSlot": f.CanTargetMoreWithHigherSlot(),
		} {
			if got != content.EffectFact(content.FactUnknown{}) {
				t.Errorf("%s: %s = %#v, want FactUnknown", name, fact, got)
			}
		}
	}
}

// Requirement: spell-effect-data.md FR-022, FR-023, table F1 to F11 - a fact is
// known true when the parts show it in any coverage; known false and the
// damage types only when the coverage is complete.
// Path: per case.
func TestDerivedFacts(t *testing.T) {
	t.Parallel()
	yes, no, unknown := content.EffectFact(content.FactKnown{Value: true}), content.EffectFact(content.FactKnown{Value: false}), content.EffectFact(content.FactUnknown{})
	damageTypes := func(m content.ModelledEffects) []content.DamageType {
		k, ok := m.Facts().DamageTypes().(content.FactKnownDamageTypes)
		if !ok {
			return nil
		}
		return k.Types()
	}
	cases := []struct {
		name, path string
		m          content.ModelledEffects
		fact       func(content.SpellEffectFacts) content.EffectFact
		want       content.EffectFact
	}{
		{"F1 Eldritch Blast forces no save", "happy", eldritchBlast(), content.SpellEffectFacts.ForcesSavingThrow, no},
		{"F2 Eldritch Blast attacks", "happy", eldritchBlast(), content.SpellEffectFacts.MakesSpellAttackRoll, yes},
		{"F4 Eldritch Blast: no extra target by slot", "unhappy (boundary)", eldritchBlast(), content.SpellEffectFacts.CanTargetMoreWithHigherSlot, no},
		{"F5 partial Fire Bolt attacks", "happy", fireBolt(content.CoveragePartial), content.SpellEffectFacts.MakesSpellAttackRoll, yes},
		{"F6 partial Fire Bolt: save unknown", "unhappy (rule interaction)", fireBolt(content.CoveragePartial), content.SpellEffectFacts.ForcesSavingThrow, unknown},
		{"F8 partial Fireball forces a save", "happy", fireball(content.CoveragePartial), content.SpellEffectFacts.ForcesSavingThrow, yes},
		{"F9 partial Hold Person targets more by slot", "happy", holdPerson(), content.SpellEffectFacts.CanTargetMoreWithHigherSlot, yes},
		{"F10 Magic Missile: more darts, not more targets", "unhappy (boundary)", magicMissile(), content.SpellEffectFacts.CanTargetMoreWithHigherSlot, no},
		{"complete Fireball deals damage", "happy", fireball(content.CoverageComplete), content.SpellEffectFacts.DealsDamage, yes},
		{"complete Cure Wounds deals no damage", "happy", cureWounds(), content.SpellEffectFacts.DealsDamage, no},
		{"partial Heal: damage unknown", "unhappy (rule interaction)", heal(content.CoveragePartial), content.SpellEffectFacts.DealsDamage, unknown},
		{"partial Heal: spell attack unknown", "unhappy (rule interaction)", heal(content.CoveragePartial), content.SpellEffectFacts.MakesSpellAttackRoll, unknown},
		{"partial Heal: more targets unknown", "unhappy (rule interaction)", heal(content.CoveragePartial), content.SpellEffectFacts.CanTargetMoreWithHigherSlot, unknown},
		{"self targeting adds no target", "unhappy (boundary)", falseLife(), content.SpellEffectFacts.CanTargetMoreWithHigherSlot, no},
		{"area targeting adds no target", "unhappy (boundary)", fireball(content.CoverageComplete), content.SpellEffectFacts.CanTargetMoreWithHigherSlot, no},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := c.fact(c.m.Facts()); got != c.want {
				t.Errorf("fact = %#v, want %#v [FR-022, FR-023]", got, c.want)
			}
		})
	}
	// Damage types: F3, F7, F11 and the union over parts.
	if got := damageTypes(eldritchBlast()); !slices.Equal(got, []content.DamageType{content.DamageForce}) { // F3
		t.Errorf("Eldritch Blast DamageTypes = %v, want [Force]", got)
	}
	if got := fireBolt(content.CoveragePartial).Facts().DamageTypes(); got != unknown { // F7
		t.Errorf("partial Fire Bolt DamageTypes = %#v, want FactUnknown", got)
	}
	orb := chromaticOrb()
	complete := modelled(content.CoverageComplete, orb.Targeting(), orb.Parts()...) // F11, synthetic complete
	if got := damageTypes(complete); len(got) != 6 || got[0] != content.DamageAcid || got[5] != content.DamageThunder {
		t.Errorf("a ChosenDamageType of six types: DamageTypes = %v", got)
	}
	if got := damageTypes(flameStrike()); !slices.Equal(got, []content.DamageType{content.DamageFire, content.DamageRadiant}) {
		t.Errorf("Flame Strike DamageTypes = %v, want [Fire Radiant]", got)
	}
	if got := damageTypes(cureWounds()); got != nil || cureWounds().Facts().DamageTypes() != content.EffectFact(content.FactKnownDamageTypes{}) {
		t.Errorf("Cure Wounds DamageTypes = %v, want the known empty set", got)
	}
}

// Requirement: spell-effect-data.md FR-006, FR-045, FR-004 - a model has at
// least one part and at most one ChosenDamageType, with a valid coverage and
// targeting.
// Path: unhappy (empty and minimal, invalid input).
func TestModelledEffectsRules(t *testing.T) {
	t.Parallel()
	targeting := content.SpellTargeting(content.TargetsSelf{})
	p := part(automatic(damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageFire))), once())
	_, err := content.NewModelledEffects(content.CoverageComplete, targeting, nil)
	wantProblem(t, err, "ModelledEffects", content.BlockEmpty)
	_, err = content.NewModelledEffects(content.CoverageComplete, targeting, []content.SpellEffectPart{})
	wantProblem(t, err, "ModelledEffects", content.BlockEmpty)
	_, err = content.NewModelledEffects(0, targeting, []content.SpellEffectPart{p})
	wantProblem(t, err, "ModelledEffects", content.BlockInvalidMember)
	_, err = content.NewModelledEffects(content.CoverageComplete, nil, []content.SpellEffectPart{p})
	wantProblem(t, err, "SpellTargeting", content.BlockMissing)
	_, err = content.NewModelledEffects(content.CoverageComplete, content.TargetsChosen{}, []content.SpellEffectPart{p})
	wantProblem(t, err, "TargetsChosen", content.BlockInvalidMember)
	_, err = content.NewModelledEffects(content.CoverageComplete, targeting, []content.SpellEffectPart{{}})
	wantProblem(t, err, "PartResolution", content.BlockMissing)

	pick := must(content.NewChosenDamageType([]content.DamageType{content.DamageFire, content.DamageCold}))
	one := damageOf(amount(rolled("1d6"), nil), pick)
	two := part(attack(one, one), once()) // two choices in one part
	_, err = content.NewModelledEffects(content.CoveragePartial, targeting, []content.SpellEffectPart{two})
	wantProblem(t, err, "ModelledEffects", content.BlockConstraint)
	apart := part(attack(one), once())
	spart := part(save(ability.Dexterity, content.NothingOnSuccess, one), once())
	_, err = content.NewModelledEffects(content.CoveragePartial, targeting, []content.SpellEffectPart{apart, spart}) // across parts
	wantProblem(t, err, "ModelledEffects", content.BlockConstraint)
	missOne := part(must(content.NewAttackRollResolution(content.SpellAttackRanged,
		[]content.SpellOutcome{damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageFire))}, []content.SpellOutcome{one})), once())
	_, err = content.NewModelledEffects(content.CoveragePartial, targeting, []content.SpellEffectPart{missOne, apart}) // OnMiss counts
	wantProblem(t, err, "ModelledEffects", content.BlockConstraint)
	if _, err := content.NewModelledEffects(content.CoveragePartial, targeting, []content.SpellEffectPart{apart}); err != nil {
		t.Errorf("one ChosenDamageType: %v", err)
	}
	m := modelled(content.CoveragePartial, targeting, apart)
	if m.Coverage() != content.CoveragePartial || m.Targeting() != targeting || len(m.Parts()) != 1 {
		t.Errorf("accessors: %v", m)
	}
	if _, has := m.Zone(); has {
		t.Error("a model has zone facts by default")
	}
}

// Requirement: spell-effect-data.md FR-007 - the parts are copied on
// construction and on read.
// Path: unhappy (immutability).
func TestModelledEffectsPartsAreCopied(t *testing.T) {
	t.Parallel()
	a := part(automatic(damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageFire))), once())
	b := part(automatic(damageOf(amount(rolled("2d6"), nil), fixedType(content.DamageFire))), once())
	in := []content.SpellEffectPart{a}
	m := must(content.NewModelledEffects(content.CoverageComplete, content.TargetsSelf{}, in))
	in[0] = b
	m.Parts()[0] = b
	if got := m.Parts(); len(got) != 1 || !got[0].Equal(a) {
		t.Errorf("Parts() = %v after mutation, want the original part [FR-007]", got)
	}
}

// Requirement: spell-effect-data.md FR-008 - a model equals another iff every
// field is equal; the variants differ; parts are ordered.
// Path: happy.
func TestModelledEffectsEquality(t *testing.T) {
	t.Parallel()
	if !fireBolt(content.CoveragePartial).Equal(fireBolt(content.CoveragePartial)) {
		t.Error("equal models are not equal")
	}
	for name, o := range map[string]content.SpellEffectModel{
		"coverage":    fireBolt(content.CoverageComplete),
		"spell":       eldritchBlast(),
		"text only":   content.TextOnlyEffects{},
		"targeting":   modelled(content.CoveragePartial, content.TargetsSelf{}, fireBolt(content.CoveragePartial).Parts()...),
		"parts count": modelled(content.CoveragePartial, fireBolt(content.CoveragePartial).Targeting(), append(fireBolt(content.CoveragePartial).Parts(), fireBolt(content.CoveragePartial).Parts()...)...),
	} {
		if fireBolt(content.CoveragePartial).Equal(o) {
			t.Errorf("models differing in %s are equal", name)
		}
	}
	m := fireBolt(content.CoveragePartial)
	if m.Equal(content.TextOnlyEffects{}) {
		t.Error("a modelled spell equals the text-only model")
	}
	a, b := part(automatic(damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageFire))), once()), part(attack(damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageFire))), once())
	ab := modelled(content.CoverageComplete, content.TargetsSelf{}, a, b)
	ba := modelled(content.CoverageComplete, content.TargetsSelf{}, b, a)
	if ab.Equal(ba) {
		t.Error("models with parts in another order are equal")
	}
}

// Requirement: spell-effect-data.md FR-009 - CheckSpellLevel rejects a model
// that contradicts the spell's level.
// Path: per case.
func TestCheckSpellLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		m     content.SpellEffectModel
		level int
		bad   bool
	}{
		{"Fire Bolt (table) as a cantrip", fireBolt(content.CoveragePartial), 0, false},
		{"table on a level 1 spell", fireBolt(content.CoveragePartial), 1, true},
		{"table on a level 9 spell", fireBolt(content.CoveragePartial), 9, true},
		{"Eldritch Blast count table as a cantrip", eldritchBlast(), 0, false},
		{"count table on a level 2 spell", eldritchBlast(), 2, true},
		{"Fireball (above 3) at level 3", fireball(content.CoverageComplete), 3, false},
		{"Fireball at level 2: above is higher than the spell", fireball(content.CoverageComplete), 2, false},
		{"Fireball at level 4: above is below the spell", fireball(content.CoverageComplete), 4, true},
		{"Fireball as a cantrip", fireball(content.CoverageComplete), 0, true},
		{"Cure Wounds (above 1) at level 1", cureWounds(), 1, false},
		{"Cure Wounds as a cantrip", cureWounds(), 0, true},
		{"Hold Person count (above 2) at level 2", holdPerson(), 2, false},
		{"Hold Person at level 3", holdPerson(), 3, true},
		{"Magic Missile instances (above 1) as a cantrip", magicMissile(), 0, true},
		{"level -1", cureWounds(), -1, true},
		{"level 10", cureWounds(), 10, true},
		{"unscaled model at every level", acidSplashNoScaling(), 5, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := c.m.CheckSpellLevel(c.level)
			if (err != nil) != c.bad {
				t.Errorf("CheckSpellLevel(%d) = %v, want error: %v [FR-009]", c.level, err, c.bad)
			}
		})
	}
	err := cureWounds().CheckSpellLevel(10)
	wantProblem(t, err, "ModelledEffects", content.BlockOutOfRange)
	err = fireBolt(content.CoveragePartial).CheckSpellLevel(2)
	wantProblem(t, err, "ModelledEffects", content.BlockConstraint)
	// AboveLevel 10 is a contradiction found only here, never by the constructor.
	above10 := modelled(content.CoverageComplete, content.TargetsSelf{}, part(automatic(
		must(content.NewSpellGrantsTemporaryHitPoints(amount(rolled("1d4"), perSlot(10, 1, 0)), content.AddsNothing))), once()))
	wantProblem(t, above10.CheckSpellLevel(9), "ModelledEffects", content.BlockConstraint)
	wantProblem(t, content.ModelledEffects{}.CheckSpellLevel(10), "ModelledEffects", content.BlockOutOfRange)
	// Scaling in a target count, in instances and in each of the three amount outcomes is found.
	for name, o := range map[string]content.SpellOutcome{
		"heal": must(content.NewSpellRestoresHitPoints(amount(rolled("1d4"), perSlot(1, 1, 0)), content.AddsNothing)),
		"thp":  must(content.NewSpellGrantsTemporaryHitPoints(amount(rolled("1d4"), perSlot(1, 1, 0)), content.AddsNothing)),
		"dmg":  damageOf(amount(rolled("1d4"), perSlot(1, 1, 0)), fixedType(content.DamageFire)),
	} {
		m := modelled(content.CoverageComplete, content.TargetsSelf{}, part(automatic(o), once()))
		if m.CheckSpellLevel(0) == nil {
			t.Errorf("per-slot scaling on a %s outcome of a cantrip was accepted", name)
		}
	}
	upTo := must(content.NewUpToTargets(2, must(content.NewPerSlotLevelCount(1, 1))))
	m := modelled(content.CoverageComplete, chosen(content.SubjectCreature, upTo, nil), part(automatic(damageOf(amount(rolled("1d4"), nil), fixedType(content.DamageFire))), once()))
	if m.CheckSpellLevel(0) == nil || m.CheckSpellLevel(1) != nil {
		t.Error("per-slot scaling of an UpToTargets count is not checked")
	}
}

func acidSplashNoScaling() content.ModelledEffects {
	return modelled(content.CoverageComplete, content.TargetsSelf{}, part(automatic(damageOf(amount(rolled("1d6"), nil), fixedType(content.DamageAcid))), once()))
}

// Requirement: spell-effect-data.md FR-048 - a model may hold one optional
// ZoneFacts; the facts are copied and compared.
// Path: happy.
func TestModelWithZoneFacts(t *testing.T) {
	t.Parallel()
	rule := must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, PartIndex: 0, Scope: content.AnyCreature}))
	z := must(content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{rule}}))
	base := fireball(content.CoveragePartial)
	m, err := base.WithZoneFacts(z)
	if err != nil {
		t.Fatal(err)
	}
	if got, has := m.Zone(); !has || !got.Equal(z) {
		t.Errorf("Zone() = %v, %v", got, has)
	}
	if _, has := base.Zone(); has {
		t.Error("WithZoneFacts changed the receiver")
	}
	if m.Equal(base) || base.Equal(m) || !m.Equal(must(base.WithZoneFacts(z))) {
		t.Error("equality ignores the zone facts")
	}
	other := must(content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{rule}, OncePerTurn: true}))
	if m.Equal(must(base.WithZoneFacts(other))) {
		t.Error("models with different zone facts are equal")
	}
	_, err = base.WithZoneFacts(content.ZoneFacts{})
	wantProblem(t, err, "ZoneFacts", content.BlockEmpty)
	// A PartIndex past the parts is not a constructor rule (FR-048).
	far := must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, PartIndex: 99, Scope: content.AnyCreature}))
	if _, err := base.WithZoneFacts(must(content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{far}}))); err != nil {
		t.Errorf("a PartIndex past the parts: %v, want it accepted by the constructor", err)
	}
}

// Requirement: spell-effect-data.md FR-004, FR-010, FR-011, FR-040, FR-042,
// FR-044, FR-046, FR-047 - the closed enums have a name for each member and
// reject 0 and values past the last.
// Path: unhappy (boundary).
func TestEnumsOfTheModel(t *testing.T) {
	t.Parallel()
	type enum interface {
		Valid() bool
		String() string
	}
	groups := map[string][]enum{
		"EffectCoverage":   {content.CoverageComplete, content.CoveragePartial},
		"TargetSubject":    {content.SubjectCreature, content.SubjectObject, content.SubjectCreatureOrObject},
		"SelfTargetPolicy": {content.MayIncludeCaster, content.ExcludesCaster},
		"SpellAttackKind":  {content.SpellAttackMelee, content.SpellAttackRanged},
		"SuccessEffect":    {content.NothingOnSuccess, content.HalfDamageOnSuccess},
		"ModifierAddition": {content.AddsNothing, content.AddsSpellcastingAbilityModifier},
		"TurnOwner":        {content.OwnerCaster, content.OwnerTarget},
		"TurnBoundary":     {content.BoundaryStart, content.BoundaryEnd},
		"ZoneTrigger": {content.CreatureEnters, content.CreatureFirstEntryOnTurn, content.CreatureEndsTurnInside,
			content.CreatureStartsTurnInside, content.ZoneMovesIntoSpace, content.TravelsWithin},
		"ZoneTargetScope":    {content.AnyCreature, content.EnemiesOfCaster, content.CreaturesCasterCanSee},
		"ZoneActionKind":     {content.BonusAction, content.MagicAction, content.WithCasterMovement},
		"DriftDirectionRule": {content.AwayFromCaster, content.AwayFromCasterChosen},
		"DesignationPolicy":  {content.NoDesignation, content.DesignationAtCastingOnly},
	}
	invalid := map[string][]enum{
		"EffectCoverage": {content.EffectCoverage(0), content.EffectCoverage(3)}, "TargetSubject": {content.TargetSubject(0), content.TargetSubject(4)},
		"SelfTargetPolicy": {content.SelfTargetPolicy(0), content.SelfTargetPolicy(3)}, "SpellAttackKind": {content.SpellAttackKind(0), content.SpellAttackKind(3)},
		"SuccessEffect": {content.SuccessEffect(0), content.SuccessEffect(3)}, "ModifierAddition": {content.ModifierAddition(0), content.ModifierAddition(3)},
		"TurnOwner": {content.TurnOwner(0), content.TurnOwner(3)}, "TurnBoundary": {content.TurnBoundary(0), content.TurnBoundary(3)},
		"ZoneTrigger": {content.ZoneTrigger(0), content.ZoneTrigger(7)}, "ZoneTargetScope": {content.ZoneTargetScope(0), content.ZoneTargetScope(4)},
		"ZoneActionKind": {content.ZoneActionKind(0), content.ZoneActionKind(4)}, "DriftDirectionRule": {content.DriftDirectionRule(0), content.DriftDirectionRule(3)},
		"DesignationPolicy": {content.DesignationPolicy(0), content.DesignationPolicy(3)},
	}
	for name, members := range groups {
		seen := map[string]bool{}
		for _, m := range members {
			if !m.Valid() || m.String() == "unknown" || seen[m.String()] {
				t.Errorf("%s member %v: valid %v, name %q (duplicate or unknown)", name, m, m.Valid(), m.String())
			}
			seen[m.String()] = true
		}
		for _, m := range invalid[name] {
			if m.Valid() || m.String() != "unknown" {
				t.Errorf("%s value %v: valid %v, name %q, want invalid and unknown", name, m, m.Valid(), m.String())
			}
		}
	}
}

// Requirement: spell-effect-data.md FR-022, FR-023 - the damage types are the
// union over every part, a chosen set adds to a fixed type, and an
// up-to-N count that grows per slot level can target more.
// Path: happy (rule interaction).
func TestDerivedFactsUnionAndUpToTargets(t *testing.T) {
	t.Parallel()
	pick := must(content.NewChosenDamageType([]content.DamageType{content.DamageAcid, content.DamageCold}))
	fireThenPick := modelled(content.CoverageComplete, chosen(content.SubjectCreature, exactly(1, nil), nil),
		part(attack(damageOf(amount(rolled("1d6"), perSlot(1, 1, 0)), fixedType(content.DamageFire))), once()),
		part(attack(damageOf(amount(rolled("1d6"), perSlot(1, 1, 0)), pick)), once()))
	k, ok := fireThenPick.Facts().DamageTypes().(content.FactKnownDamageTypes)
	if want := []content.DamageType{content.DamageAcid, content.DamageCold, content.DamageFire}; !ok || !slices.Equal(k.Types(), want) {
		t.Errorf("DamageTypes = %v, want %v", k.Types(), want)
	}
	upTo := must(content.NewUpToTargets(3, must(content.NewPerSlotLevelCount(2, 1))))
	m := modelled(content.CoveragePartial, chosen(content.SubjectCreature, upTo, nil), part(automatic(must(content.NewSpellAppliesCondition(content.ConditionParalyzed, content.ForSpellDuration{}))), once()))
	if got := m.Facts().CanTargetMoreWithHigherSlot(); got != content.EffectFact(content.FactKnown{Value: true}) {
		t.Errorf("CanTargetMoreWithHigherSlot = %#v, want known true", got)
	}
}
