package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// metaCase is one row of Table M: a Sorcerer, an option and a spell, and the application or refusal expected.
type metaCase struct {
	name, requirement, path string
	level                   int
	active                  bool
	score                   int // Charisma, 16 (+3) when 0
	effect                  character.MetamagicEffect
	cost                    int // 1 when 0
	facts                   func(*character.MetamagicSpellFacts)
	ctx                     character.MetamagicCastContext
	params                  character.MetamagicParameters
	kind                    character.SorcererRejectionKind
	why                     character.MetamagicIneligibility
	want                    character.MetamagicApplication
}

func metaApp(e character.MetamagicEffect, edit func(*character.MetamagicApplication)) character.MetamagicApplication {
	a := character.MetamagicApplication{Effect: e, Cost: 1}
	if edit != nil {
		edit(&a)
	}
	return a
}

func metaRange(n int, u content.DistanceUnit) func(*character.MetamagicSpellFacts) {
	return func(f *character.MetamagicSpellFacts) { f.Range = must(content.NewSpellRangeDistance(n, u)) }
}

func metaDuration(n int, u content.DurationUnit) func(*character.MetamagicSpellFacts) {
	return func(f *character.MetamagicSpellFacts) {
		f.Duration = must(content.NewSpellDurationTimed(n, u, content.LimitFixed))
	}
}

func metaCastTime(t content.CastingTime) func(*character.MetamagicSpellFacts) {
	return func(f *character.MetamagicSpellFacts) { f.CastingTime = t }
}

func metaMaterial(consumed bool, cost []int) func(*character.MetamagicSpellFacts) {
	return func(f *character.MetamagicSpellFacts) {
		m := must(content.NewMaterialComponent("a pinch of dust", cost, consumed))
		f.Components = must(content.NewSpellComponents(true, false).WithMaterial(m))
	}
}

func metaUnknown(fact character.MetamagicFact) func(*character.MetamagicSpellFacts) {
	return func(f *character.MetamagicSpellFacts) { f.Unknown = []character.MetamagicFact{fact} }
}

// Requirement: per case (doc/specs/class-sorcerer.md Table M, FR-045, FR-062 to FR-082, FR-111 to FR-118, FR-120, FR-123; SRD 5.2.1, "Metamagic Options", pp. 66-67)
// Path: per case.
func TestApplyMetamagicTableM(t *testing.T) {
	t.Parallel()
	const (
		ca, di, em, ex, he, qu, se, su, tr, tw = character.MetamagicCareful, character.MetamagicDistant, character.MetamagicEmpowered, character.MetamagicExtended,
			character.MetamagicHeightened, character.MetamagicQuickened, character.MetamagicSeeking, character.MetamagicSubtle, character.MetamagicTransmuted, character.MetamagicTwinned
	)
	applied := func(e ...character.MetamagicEffect) character.MetamagicCastContext {
		return character.MetamagicCastContext{Applied: e}
	}
	seconds := func(n int) func(*character.MetamagicApplication) {
		return func(a *character.MetamagicApplication) { a.DurationSeconds = n }
	}
	notEligible := character.SorcererMetamagicNotEligible
	cases := []metaCase{
		{name: "M-1", requirement: "FR-064", path: "happy", level: 5, effect: ca, params: character.MetamagicParameters{Count: 3},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 3 })},
		{name: "M-2", requirement: "FR-065", path: "unhappy (boundary)", level: 5, effect: ca, params: character.MetamagicParameters{Count: 4}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-3", requirement: "FR-065", path: "unhappy (boundary)", level: 5, score: 10, effect: ca, params: character.MetamagicParameters{Count: 1},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 1 })},
		{name: "M-4", requirement: "FR-065", path: "unhappy (boundary)", level: 5, score: 10, effect: ca, params: character.MetamagicParameters{Count: 2}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-5", requirement: "FR-065", path: "unhappy (boundary)", level: 5, score: 8, effect: ca, params: character.MetamagicParameters{Count: 1},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 1 })},
		{name: "M-6", requirement: "FR-068", path: "happy", level: 5, score: 14, effect: em, params: character.MetamagicParameters{Count: 2}, want: metaApp(em, func(a *character.MetamagicApplication) { a.Rerolls = 2 })},
		{name: "M-7", requirement: "FR-069", path: "unhappy (boundary)", level: 5, score: 14, effect: em, params: character.MetamagicParameters{Count: 3}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-8", requirement: "FR-068", path: "unhappy (boundary)", level: 5, score: 10, effect: em, params: character.MetamagicParameters{Count: 1}, want: metaApp(em, func(a *character.MetamagicApplication) { a.Rerolls = 1 })},
		{name: "M-9", requirement: "FR-069", path: "unhappy (boundary)", level: 5, score: 14, effect: em, params: character.MetamagicParameters{Count: 2},
			facts: func(f *character.MetamagicSpellFacts) { f.DamageDice = 1 }, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-10", requirement: "FR-066", path: "unhappy (boundary)", level: 5, effect: di, facts: metaRange(5, content.DistanceFeet), want: metaApp(di, func(a *character.MetamagicApplication) { a.RangeFeet = 10 })},
		{name: "M-11", requirement: "FR-067", path: "unhappy (boundary)", level: 5, effect: di, facts: metaRange(4, content.DistanceFeet), kind: notEligible, why: character.MetamagicRangeNotEligible},
		{name: "M-12", requirement: "FR-066", path: "happy", level: 5, effect: di, facts: func(f *character.MetamagicSpellFacts) { f.Range = content.NewSpellRangeTouch() },
			want: metaApp(di, func(a *character.MetamagicApplication) { a.RangeFeet = 30 })},
		{name: "M-13", requirement: "FR-067", path: "unhappy (invalid input)", level: 5, effect: di, facts: func(f *character.MetamagicSpellFacts) { f.Range = content.NewSpellRangeSelf() },
			kind: notEligible, why: character.MetamagicRangeNotEligible},
		{name: "M-14", requirement: "FR-070", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(1, content.DurationUnitMinutes), want: metaApp(ex, seconds(120))},
		{name: "M-15", requirement: "FR-071", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(1, content.DurationUnitRounds), kind: notEligible, why: character.MetamagicDurationNotEligible},
		{name: "M-16", requirement: "FR-071", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(12, content.DurationUnitHours), want: metaApp(ex, seconds(86400))},
		{name: "M-17", requirement: "FR-071", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(16, content.DurationUnitHours), want: metaApp(ex, seconds(86400))},
		{name: "M-18", requirement: "FR-075", path: "unhappy (rule interaction)", level: 5, effect: qu, ctx: character.MetamagicCastContext{LevelOnePlusSpellCastThisTurn: false},
			cost: 2, want: character.MetamagicApplication{Effect: qu, Cost: 2, BonusAction: true}},
		{name: "M-19", requirement: "FR-075, FR-113", path: "unhappy (rule interaction)", level: 5, effect: qu, ctx: character.MetamagicCastContext{LevelOnePlusSpellCastThisTurn: true},
			cost: 2, kind: character.SorcererQuickenedRestrictionApplies},
		{name: "M-21", requirement: "FR-079", path: "happy", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageFire, To: content.DamageThunder},
			want: metaApp(tr, func(a *character.MetamagicApplication) { a.From, a.To = content.DamageFire, content.DamageThunder })},
		{name: "M-22", requirement: "FR-080", path: "unhappy (invalid input)", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageFire, To: content.DamageFire}, kind: notEligible, why: character.MetamagicDamageTypeNotTransmutable},
		{name: "M-23", requirement: "FR-080", path: "unhappy (invalid input)", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageForce, To: content.DamageFire},
			facts: func(f *character.MetamagicSpellFacts) { f.DamageTypes = []content.DamageType{content.DamageForce} }, kind: notEligible, why: character.MetamagicDamageTypeNotTransmutable},
		{name: "M-24", requirement: "FR-120", path: "unhappy (boundary)", level: 5, effect: ca, params: character.MetamagicParameters{Count: 3},
			facts: func(f *character.MetamagicSpellFacts) { f.OtherCreaturesSaving = 2 }, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-25", requirement: "FR-115", path: "unhappy (boundary)", level: 5, effect: ca, params: character.MetamagicParameters{Count: 0}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-26", requirement: "FR-115", path: "unhappy (invalid input)", level: 5, effect: ca, params: character.MetamagicParameters{Count: -1}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-27", requirement: "FR-065", path: "unhappy (invalid input)", level: 5, effect: ca, params: character.MetamagicParameters{Count: 1},
			facts: func(f *character.MetamagicSpellFacts) { f.ForcesSavingThrow = false }, kind: notEligible, why: character.MetamagicNoSavingThrow},
		{name: "M-28", requirement: "FR-115", path: "unhappy (boundary)", level: 5, effect: em, params: character.MetamagicParameters{Count: 0}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-29", requirement: "FR-115", path: "unhappy (invalid input)", level: 5, effect: em, params: character.MetamagicParameters{Count: -1}, kind: notEligible, why: character.MetamagicCountOutOfRange},
		{name: "M-30", requirement: "FR-070", path: "happy", level: 5, effect: ex, facts: metaDuration(10, content.DurationUnitMinutes), want: metaApp(ex, seconds(1200))},
		{name: "M-31", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(10, content.DurationUnitRounds), want: metaApp(ex, seconds(120))},
		{name: "M-32", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: func(f *character.MetamagicSpellFacts) { f.Duration = content.NewSpellDurationInstantaneous() },
			kind: notEligible, why: character.MetamagicDurationNotEligible},
		{name: "M-33", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: func(f *character.MetamagicSpellFacts) { f.Duration = content.NewSpellDurationUntilDispelled() },
			kind: notEligible, why: character.MetamagicDurationNotEligible},
		{name: "M-34", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: func(f *character.MetamagicSpellFacts) {
			f.Duration = content.NewSpellDurationUntilDispelledOrTriggered()
		}, kind: notEligible, why: character.MetamagicDurationNotEligible},
		{name: "M-35", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: func(f *character.MetamagicSpellFacts) { f.Duration = content.NewSpellDurationSpecial() },
			kind: notEligible, why: character.MetamagicDurationNotEligible},
		{name: "M-36", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(24, content.DurationUnitHours), kind: notEligible, why: character.MetamagicDurationAlreadyAtMaximum},
		{name: "M-37", requirement: "FR-116", path: "unhappy (boundary)", level: 5, effect: ex, facts: metaDuration(10, content.DurationUnitDays), kind: notEligible, why: character.MetamagicDurationAlreadyAtMaximum},
		{name: "M-38", requirement: "FR-070", path: "happy", level: 5, effect: ex, facts: func(f *character.MetamagicSpellFacts) { f.RequiresConcentration = true },
			want: metaApp(ex, func(a *character.MetamagicApplication) { a.DurationSeconds, a.ConcentrationAdvantage = 120, true })},
		{name: "M-39", requirement: "FR-117", path: "unhappy (boundary)", level: 5, effect: di, facts: func(f *character.MetamagicSpellFacts) { f.Range = content.NewSpellRangeSight() },
			kind: notEligible, why: character.MetamagicRangeNotEligible},
		{name: "M-40", requirement: "FR-117", path: "unhappy (boundary)", level: 5, effect: di, facts: func(f *character.MetamagicSpellFacts) { f.Range = content.NewSpellRangeUnlimited() },
			kind: notEligible, why: character.MetamagicRangeNotEligible},
		{name: "M-41", requirement: "FR-117", path: "unhappy (boundary)", level: 5, effect: di, facts: func(f *character.MetamagicSpellFacts) { f.Range = content.NewSpellRangeSpecial() },
			kind: notEligible, why: character.MetamagicRangeNotEligible},
		{name: "M-42", requirement: "FR-117", path: "unhappy (boundary)", level: 5, effect: di, facts: metaRange(1, content.DistanceMiles), want: metaApp(di, func(a *character.MetamagicApplication) { a.RangeMiles = 2 })},
		{name: "M-43", requirement: "FR-072", path: "happy", level: 5, effect: he, cost: 2, want: character.MetamagicApplication{Effect: he, Cost: 2, DisadvantageOnSave: true}},
		{name: "M-44", requirement: "FR-072", path: "unhappy (invalid input)", level: 5, effect: he, cost: 2, facts: func(f *character.MetamagicSpellFacts) { f.ForcesSavingThrow = false },
			kind: notEligible, why: character.MetamagicNoSavingThrow},
		{name: "M-45", requirement: "FR-078", path: "happy", level: 5, effect: su, want: metaApp(su, func(a *character.MetamagicApplication) { a.RemovesVerbal, a.RemovesSomatic = true, true })},
		{name: "M-46", requirement: "FR-078", path: "unhappy (boundary)", level: 5, effect: su, facts: metaMaterial(false, []int{50}), want: metaApp(su, func(a *character.MetamagicApplication) { a.RemovesVerbal = true })},
		{name: "M-47", requirement: "FR-078", path: "unhappy (boundary)", level: 5, effect: su, facts: func(f *character.MetamagicSpellFacts) {
			metaMaterial(true, nil)(f)
			f.Components = must(content.NewSpellComponents(false, false).WithMaterial(must(content.NewMaterialComponent("a pearl", nil, true))))
		}, kind: notEligible, why: character.MetamagicNothingToRemove},
		{name: "M-48", requirement: "FR-078", path: "unhappy (empty/minimal)", level: 5, effect: su, facts: func(f *character.MetamagicSpellFacts) { f.Components = content.NewSpellComponents(false, false) },
			kind: notEligible, why: character.MetamagicNothingToRemove},
		{name: "M-48b uncosted material", requirement: "FR-078", path: "happy", level: 5, effect: su, facts: func(f *character.MetamagicSpellFacts) {
			f.Components = must(content.NewSpellComponents(false, false).WithMaterial(must(content.NewMaterialComponent("a twig", nil, false))))
		}, want: metaApp(su, func(a *character.MetamagicApplication) { a.RemovesMaterial = true })},
		{name: "M-49", requirement: "FR-074", path: "unhappy (invalid input)", level: 5, effect: qu, cost: 2, facts: metaCastTime(content.NewCastingTimeBonusAction()), kind: notEligible, why: character.MetamagicCastingTimeNotAction},
		{name: "M-50", requirement: "FR-074", path: "unhappy (invalid input)", level: 5, effect: qu, cost: 2, facts: metaCastTime(must(content.NewCastingTimeReaction("when hit"))), kind: notEligible, why: character.MetamagicCastingTimeNotAction},
		{name: "M-51", requirement: "FR-074", path: "unhappy (invalid input)", level: 5, effect: qu, cost: 2, facts: metaCastTime(must(content.NewCastingTimeMinutes(1))), kind: notEligible, why: character.MetamagicCastingTimeNotAction},
		{name: "M-52", requirement: "FR-073", path: "happy", level: 5, effect: qu, cost: 2, want: character.MetamagicApplication{Effect: qu, Cost: 2, BonusAction: true}},
		{name: "M-53", requirement: "FR-111", path: "unhappy (rule interaction)", level: 5, effect: em, ctx: applied(di), params: character.MetamagicParameters{Count: 1}, want: metaApp(em, func(a *character.MetamagicApplication) { a.Rerolls = 1 })},
		{name: "M-54", requirement: "FR-111", path: "unhappy (rule interaction)", level: 5, effect: di, ctx: applied(em), kind: character.SorcererMetamagicCombinationRefused},
		{name: "M-55", requirement: "FR-062", path: "unhappy (rule interaction)", level: 5, effect: ex, ctx: applied(di), kind: character.SorcererMetamagicCombinationRefused},
		{name: "M-56", requirement: "FR-045", path: "happy", level: 7, active: true, effect: ex, ctx: applied(di), want: metaApp(ex, seconds(120))},
		{name: "M-57", requirement: "FR-112", path: "unhappy (rule interaction)", level: 7, active: true, effect: em, ctx: applied(di, ex), params: character.MetamagicParameters{Count: 1},
			want: metaApp(em, func(a *character.MetamagicApplication) { a.Rerolls = 1 })},
		{name: "M-58", requirement: "FR-112", path: "unhappy (rule interaction)", level: 7, active: true, effect: qu, cost: 2, ctx: applied(di, ex), kind: character.SorcererMetamagicCombinationRefused},
		{name: "M-59", requirement: "FR-112", path: "unhappy (rule interaction)", level: 5, effect: se, ctx: applied(di, em), want: metaApp(se, func(a *character.MetamagicApplication) { a.RerollAttack = true })},
		{name: "M-60", requirement: "FR-112", path: "unhappy (rule interaction)", level: 5, effect: em, ctx: applied(em), params: character.MetamagicParameters{Count: 1}, kind: character.SorcererMetamagicCombinationRefused},
		{name: "M-61", requirement: "FR-103", path: "unhappy (rule interaction)", level: 5, effect: ca, params: character.MetamagicParameters{Count: 1},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 1 })},
		{name: "M-62 Careful", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: ca, params: character.MetamagicParameters{Count: 1}, facts: metaUnknown(character.MetamagicFactSavingThrow),
			kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Distant", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: di, facts: metaUnknown(character.MetamagicFactRange), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Empowered", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: em, params: character.MetamagicParameters{Count: 1}, facts: metaUnknown(character.MetamagicFactDamage),
			kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Extended", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: ex, facts: metaUnknown(character.MetamagicFactDuration), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Heightened", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: he, cost: 2, facts: metaUnknown(character.MetamagicFactSavingThrow), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Quickened", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: qu, cost: 2, facts: metaUnknown(character.MetamagicFactCastingTime), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Seeking", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: se, facts: metaUnknown(character.MetamagicFactAttackRoll), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Subtle", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: su, facts: metaUnknown(character.MetamagicFactComponents), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Transmuted", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageFire, To: content.DamageCold},
			facts: metaUnknown(character.MetamagicFactDamage), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-62 Twinned", requirement: "FR-123", path: "unhappy (missing content)", level: 5, effect: tw, facts: metaUnknown(character.MetamagicFactHigherSlotTarget), kind: notEligible, why: character.MetamagicSpellFactUnknown},
		{name: "M-63", requirement: "FR-118", path: "unhappy (rule interaction)", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageFire, To: content.DamageLightning},
			facts: func(f *character.MetamagicSpellFacts) {
				f.DamageTypes = []content.DamageType{content.DamageFire, content.DamageCold}
			},
			want: metaApp(tr, func(a *character.MetamagicApplication) { a.From, a.To = content.DamageFire, content.DamageLightning })},
		{name: "M-64", requirement: "FR-118", path: "unhappy (invalid input)", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageAcid, To: content.DamageCold}, kind: notEligible, why: character.MetamagicDamageTypeNotTransmutable},
		{name: "a target outside the list", requirement: "FR-080", path: "unhappy (invalid input)", level: 5, effect: tr, params: character.MetamagicParameters{From: content.DamageFire, To: content.DamageForce}, kind: notEligible, why: character.MetamagicDamageTypeNotTransmutable},
		{name: "M-65 free use while inactive", requirement: "FR-046", path: "unhappy (state)", level: 20, effect: ca, params: character.MetamagicParameters{Count: 1, RequestFree: true},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 1 })},
		{name: "free use below level 20", requirement: "FR-046", path: "unhappy (boundary)", level: 19, active: true, effect: ca, params: character.MetamagicParameters{Count: 1, RequestFree: true},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 1 })},
		{name: "free use at level 20 while active", requirement: "FR-046", path: "happy", level: 20, active: true, effect: ca, params: character.MetamagicParameters{Count: 1, RequestFree: true},
			want: character.MetamagicApplication{Effect: ca, Cost: 0, Free: true, ProtectedCreatures: 1}},
		{name: "free use not asked for", requirement: "FR-046", path: "happy", level: 20, active: true, effect: ca, params: character.MetamagicParameters{Count: 1},
			want: metaApp(ca, func(a *character.MetamagicApplication) { a.ProtectedCreatures = 1 })},
		{name: "Twinned", requirement: "FR-081", path: "happy", level: 5, effect: tw, want: metaApp(tw, func(a *character.MetamagicApplication) { a.EffectiveLevelIncrease = 1 })},
		{name: "Twinned with no extra target", requirement: "FR-082", path: "unhappy (invalid input)", level: 5, effect: tw, facts: func(f *character.MetamagicSpellFacts) { f.HigherSlotAdditionalTarget = false },
			kind: notEligible, why: character.MetamagicNoHigherSlotTarget},
		{name: "Seeking after a miss", requirement: "FR-076", path: "happy", level: 5, effect: se, want: metaApp(se, func(a *character.MetamagicApplication) { a.RerollAttack = true })},
		{name: "Seeking after a hit", requirement: "FR-077", path: "unhappy (state)", level: 5, effect: se, facts: func(f *character.MetamagicSpellFacts) { f.AttackRoll = sorAttack(true) },
			kind: notEligible, why: character.MetamagicNoMissedAttackRoll},
		{name: "Seeking with no attack roll", requirement: "FR-077", path: "unhappy (state)", level: 5, effect: se, facts: func(f *character.MetamagicSpellFacts) { f.AttackRoll = nil },
			kind: notEligible, why: character.MetamagicNoMissedAttackRoll},
		{name: "the option is not known", requirement: "FR-061", path: "unhappy (state)", level: 5, effect: character.MetamagicEffect(0), kind: character.SorcererMetamagicOptionNotKnown},
		{name: "level 1 has no Metamagic", requirement: "FR-019", path: "unhappy (boundary)", level: 1, effect: ca, params: character.MetamagicParameters{Count: 1}, kind: character.SorcererFeatureNotGained},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := sorProfile(c.level, "")
			score := c.score
			if score == 0 {
				score = 16
			}
			known := []character.MetamagicEffect{ca, di, em, ex, he, qu, se, su, tr, tw}
			s := sorWith(p, p.Level().SorceryPointsMaximum(), known, func(s *character.SorcererStateParts) { s.InnateSorceryActive = c.active })
			cost := c.cost
			if cost == 0 {
				cost = 1
			}
			req := character.MetamagicRequest{Effect: c.effect, Cost: cost, Facts: sorFacts(c.facts), Context: c.ctx, Parameters: c.params}
			res := character.ApplyMetamagic(sorID, s, p, sorMod(score), req)
			if c.kind != 0 {
				r, ok := res.(character.MetamagicRefused)
				if !ok || r.Rejection().Kind != c.kind || r.Rejection().Ineligibility != c.why {
					t.Fatalf("result = %#v, want refused %v %v [%s %s]", res, c.kind, c.why, c.requirement, c.path)
				}
				return
			}
			a, ok := res.(character.MetamagicApplied)
			if !ok {
				t.Fatalf("result = %#v, want applied [%s %s]", res, c.requirement, c.path)
			}
			if got := a.Application(); !reflect.DeepEqual(got, c.want) {
				t.Errorf("application = %+v, want %+v [%s %s]", got, c.want, c.requirement, c.path)
			}
			spent := s.SorceryPoints() - a.State().SorceryPoints()
			ev, isUsed := a.Events()[0].(character.MetamagicUsed)
			if spent != c.want.Cost || len(a.Events()) != 1 || !isUsed || ev.CharacterID() != sorID || ev.Effect() != c.effect || ev.Cost() != c.want.Cost || ev.Free() != c.want.Free {
				t.Errorf("spent %d, events %#v; want %d points and a MetamagicUsed of %v [%s %s]", spent, a.Events(), c.want.Cost, c.effect, c.requirement, c.path)
			}
			if a.State().QuickenedUsedThisTurn() != (c.effect == qu) || a.State().FreeMetamagicSpent() != c.want.Free {
				t.Errorf("per-turn flags = %+v [%s]", a.State().Parts(), c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/class-sorcerer.md FR-059, FR-060 (SRD 5.2.1, "Level 2: Metamagic", p. 66) - using an option spends its cost and reports the new total; the points one below the cost refuse, exactly the cost is accepted, and a refusal
// leaves the state unchanged.
// Path: unhappy (boundary).
func TestApplyMetamagicCost(t *testing.T) {
	t.Parallel()
	p := sorProfile(5, "")
	known := []character.MetamagicEffect{character.MetamagicHeightened}
	req := character.MetamagicRequest{Effect: character.MetamagicHeightened, Cost: 2, Facts: sorFacts(nil)}
	one := sorWith(p, 1, known, nil)
	r, ok := character.ApplyMetamagic(sorID, one, p, sorMod(16), req).(character.MetamagicRefused)
	if !ok || r.Rejection().Kind != character.SorcererSorceryPointsInsufficient || r.Rejection().Required != 2 || r.Rejection().Available != 1 {
		t.Fatalf("one below the cost: %#v", r)
	}
	two := sorWith(p, 2, known, nil)
	a, ok := character.ApplyMetamagic(sorID, two, p, sorMod(16), req).(character.MetamagicApplied)
	if !ok || a.State().SorceryPoints() != 0 || two.SorceryPoints() != 2 {
		t.Fatalf("exactly the cost: %#v (input points %d)", a, two.SorceryPoints())
	}
	if got := a.Events(); len(got) != 1 {
		t.Fatal("events")
	} else {
		got[0] = nil
	}
	if a.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
}

// Requirement: doc/specs/class-sorcerer.md M-20, FR-075 (SRD 5.2.1, "Quickened Spell", p. 67) - after Quickened Spell this turn a level 1+ spell may not be cast, a cantrip may, and starting the next turn lifts the restriction.
// Path: unhappy (rule interaction).
func TestCheckQuickenedRestriction(t *testing.T) {
	t.Parallel()
	p := sorProfile(5, "")
	after := character.MetamagicRequest{Effect: character.MetamagicQuickened, Cost: 2, Facts: sorFacts(nil)}
	used := character.ApplyMetamagic(sorID, sorWith(p, 5, []character.MetamagicEffect{character.MetamagicQuickened}, nil), p, sorMod(16), after).(character.MetamagicApplied).State()
	for _, c := range []struct {
		level   int
		allowed bool
	}{{0, true}, {1, false}, {3, false}} {
		got := character.CheckQuickenedRestriction(used, c.level)
		_, allowed := got.(character.QuickenedCastAllowed)
		if allowed != c.allowed {
			t.Errorf("level %d: %#v, want allowed %v", c.level, got, c.allowed)
		}
		if r, refused := got.(character.QuickenedCastRefused); refused && r.Rejection().Kind != character.SorcererQuickenedRestrictionApplies {
			t.Errorf("level %d: kind = %v", c.level, r.Rejection().Kind)
		}
	}
	if _, ok := character.CheckQuickenedRestriction(character.StartSorcererTurn(used), 1).(character.QuickenedCastAllowed); !ok {
		t.Error("a new turn must lift the restriction")
	}
	if _, ok := character.CheckQuickenedRestriction(sorFresh(p), 1).(character.QuickenedCastAllowed); !ok {
		t.Error("with no Quickened Spell used a level 1 spell is allowed")
	}
}
