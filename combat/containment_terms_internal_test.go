package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-004, FR-005 - a term or an entry built by hand with one invalid member (which no exported constructor can produce, since the members are private) is not valid; the guards of the constructors and of valid agree.
// Path: per case.
func TestHandBuiltTermsWithOneInvalidMemberAreInvalid(t *testing.T) {
	t.Parallel()
	good := MustDamageTerm(dice.MustParseExpression("3d6"), content.DamageAcid)
	for name, term := range map[string]DamageTerm{
		"no dice":        {expression: dice.Expression{}, kind: content.DamageAcid},
		"no damage type": {expression: dice.MustParseExpression("3d6")},
		"neither":        {},
		"both":           good,
	} {
		_, ok := newDamageTerms([]DamageTerm{term}, 1)
		if ok != (name == "both") || term.valid() != (name == "both") {
			t.Errorf("%s: newDamageTerms ok = %v, valid = %v", name, ok, term.valid())
		}
	}
	entry := func(mod func(*EntryBySave)) EntryBySave {
		e := MustEntryBySave(ability.Dexterity, 12, []DamageTerm{good}, true, 5, true)
		mod(&e)
		return e
	}
	for name, c := range map[string]struct {
		e  EntryBySave
		ok bool
	}{
		"valid":                 {entry(func(*EntryBySave) {}), true},
		"no ability":            {entry(func(e *EntryBySave) { e.ability = 0 }), false},
		"ability past the last": {entry(func(e *EntryBySave) { e.ability = 7 }), false},
		"dc 0":                  {entry(func(e *EntryBySave) { e.dc = 0 }), false},
		"dc 1":                  {entry(func(e *EntryBySave) { e.dc = 1 }), true},
		"half without damage":   {entry(func(e *EntryBySave) { e.damage = damageTerms{} }), false},
		"no half, no damage":    {entry(func(e *EntryBySave) { e.damage, e.halfOnSuccess = damageTerms{}, false }), true},
		"negative relocation":   {entry(func(e *EntryBySave) { e.relocateFeet = -1 }), false},
		"no relocation":         {entry(func(e *EntryBySave) { e.relocateFeet = 0 }), true},
	} {
		if c.e.valid() != c.ok {
			t.Errorf("%s: valid = %v, want %v", name, c.e.valid(), c.ok)
		}
	}
}

// Requirement: doc/specs/monster-engulf-and-swallow.md FR-010, FR-011 - ContainmentTerms.valid, the guard every record and set relies on, rejects hand-built terms that break one rule at a time: an imposed condition other than Blinded and Restrained, a locked action with surrounding space, a regurgitation unless swallowed, a grapple mismatch, a periodic damage with an invalid or misplaced deferral, a corpse escape of no feet.
// Path: per case.
func TestHandBuiltContainmentTermsBreakingOneRuleAreInvalid(t *testing.T) {
	t.Parallel()
	acid := []DamageTerm{MustDamageTerm(dice.MustParseExpression("2d4"), content.DamageAcid)}
	base := MustContainmentTerms(ContainmentTermsInput{Mode: ContainmentModeSwallowed, MaxSize: content.SizeLarge, Capacity: MustContainmentCapacity(1, nil),
		Entry: EntryAutomatic{}, Imposed: []content.ConditionKind{content.ConditionBlinded, content.ConditionRestrained}, Suffocation: SuffocationRuleNone,
		DeathRule: ReleaseOnDeath{}, ActionKind: ContainmentActionKindAction})
	cases := []struct {
		name string
		mod  func(*ContainmentTerms)
		ok   bool
	}{
		{"the base", func(*ContainmentTerms) {}, true},
		{"no imposed conditions", func(t *ContainmentTerms) { t.imposed = 0 }, true},
		{"Restrained only", func(t *ContainmentTerms) { t.imposed = bit(content.ConditionRestrained) }, true},
		{"Poisoned imposed", func(t *ContainmentTerms) { t.imposed |= bit(content.ConditionPoisoned) }, false},
		{"Grappled imposed", func(t *ContainmentTerms) { t.imposed = bit(content.ConditionGrappled) }, false},
		{"locked action", func(t *ContainmentTerms) { t.lockedAction = "Bite" }, true},
		{"locked action with a leading space", func(t *ContainmentTerms) { t.lockedAction = " Bite" }, false},
		{"locked action with a trailing space", func(t *ContainmentTerms) { t.lockedAction = "Bite " }, false},
		{"locked action blank", func(t *ContainmentTerms) { t.lockedAction = " " }, false},
		{"regurgitation with a swallow", func(t *ContainmentTerms) { t.regurgitation = MustRegurgitation(30, 14, 10) }, true},
		{"regurgitation with an engulf", func(t *ContainmentTerms) {
			t.mode, t.regurgitation = ContainmentModeEngulfed, MustRegurgitation(30, 14, 10)
		}, false},
		{"engulf with no regurgitation", func(t *ContainmentTerms) { t.mode = ContainmentModeEngulfed }, true},
		{"grapple with a swallow", func(t *ContainmentTerms) { t.grapple = MustGrappleHoldTerms(14, "Maw") }, false},
		{"engulfed-grappled without a grapple", func(t *ContainmentTerms) { t.mode = ContainmentModeEngulfedGrappled }, false},
		{"engulfed-grappled with a grapple", func(t *ContainmentTerms) {
			t.mode, t.grapple = ContainmentModeEngulfedGrappled, MustGrappleHoldTerms(14, "Maw")
		}, true},
		{"periodic at the end", func(t *ContainmentTerms) {
			t.periodic = MustPeriodicDamage(acid, ContainmentTimingEndOfContainerTurn, true)
		}, true},
		{"periodic at the start", func(t *ContainmentTerms) {
			t.periodic = MustPeriodicDamage(acid, ContainmentTimingStartOfContainerTurn, false)
		}, true},
		{"periodic with no timing", func(t *ContainmentTerms) {
			t.periodic = PeriodicDamage{damage: damageTerms{n: 1, items: [maxDamageTerms]DamageTerm{acid[0]}}}
		}, false},
		{"periodic with a timing past the last", func(t *ContainmentTerms) {
			t.periodic = PeriodicDamage{damage: damageTerms{n: 1, items: [maxDamageTerms]DamageTerm{acid[0]}}, timing: 3}
		}, false},
		{"deferral at the start of the turn", func(t *ContainmentTerms) {
			t.periodic = PeriodicDamage{damage: damageTerms{n: 1, items: [maxDamageTerms]DamageTerm{acid[0]}}, timing: ContainmentTimingStartOfContainerTurn, firstTickNextTurn: true}
		}, false},
		{"disgorge with no periodic damage", func(t *ContainmentTerms) { t.disgorgeAfterFirstDamage = true }, false},
		{"a corpse escape of 1 foot", func(t *ContainmentTerms) { t.deathRule = EscapeFromCorpse{feet: 1} }, true},
		{"a corpse escape of 0 feet", func(t *ContainmentTerms) { t.deathRule = EscapeFromCorpse{} }, false},
		{"a corpse escape of -1 feet", func(t *ContainmentTerms) { t.deathRule = EscapeFromCorpse{feet: -1} }, false},
		{"no death rule", func(t *ContainmentTerms) { t.deathRule = nil }, false},
		{"no entry", func(t *ContainmentTerms) { t.entry = nil }, false},
		{"zero capacity", func(t *ContainmentTerms) { t.capacity = ContainmentCapacity{} }, false},
	}
	for _, c := range cases {
		terms := base
		c.mod(&terms)
		if terms.valid() != c.ok {
			t.Errorf("%s: valid = %v, want %v", c.name, terms.valid(), c.ok)
		}
	}
	if (ContainmentTerms{}).valid() {
		t.Error("the zero terms are invalid")
	}
}
