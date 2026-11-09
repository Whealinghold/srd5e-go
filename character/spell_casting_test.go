package character_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// castCase is one CastSpell call. Defaults: a Wizard 5 casting as a Wizard in CastModeNormal.
type castCase struct {
	name, req, path string
	state           character.SpellcastingState
	class           character.SpellcastingClass
	noClass         bool
	spell           content.Spell
	mode            character.CastMode
	slot            character.SpellSlotChoice
	mods            []func(*character.CastingFacts)
	timeMode        content.CastingTimeMode
	stats           *character.SpellcastingStats
	terms           *character.SpellCastTerms
	featureLevel    int
	counts          bool
}

func (c castCase) command(t *testing.T) character.CastSpellCommand {
	t.Helper()
	if c.state.IsZero() {
		c.state = wizard5(t)
	}
	if c.class == 0 && !c.noClass {
		c.class = character.CasterWizard
	}
	if c.mode == 0 {
		c.mode = character.CastModeNormal
	}
	return character.CastSpellCommand{State: c.state, Class: c.class, Stats: c.stats, Spell: c.spell, Mode: c.mode, Slot: c.slot, CastingTimeMode: c.timeMode,
		Context: castContext(t, c.mods...), Terms: c.terms, FeatureCastLevel: c.featureLevel, CountsAsSlotSpell: c.counts}
}

func (c castCase) run(t *testing.T) (character.CastSpellResult, error) {
	t.Helper()
	return character.NewCastSpell().Do(context.Background(), c.command(t))
}

// eventNames returns the type names of events, without the package.
func eventNames[E any](events []E) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = strings.TrimPrefix(fmt.Sprintf("%T", e), "character.")
	}
	return out
}

func noFree(f *character.CastingFacts)    { f.HasFreeHand = false }
func slotSpent(f *character.CastingFacts) { f.SlotSpentThisTurn = true }
func holdingFocus(k character.SpellcastingFocusKind) func(*character.CastingFacts) {
	return func(f *character.CastingFacts) { f.HeldFocus = k }
}

// expendAll spends n slots of a level from a pool.
func expendAll(t *testing.T, s character.SpellcastingState, pool character.SpellSlotPool, level, n int) character.SpellcastingState {
	t.Helper()
	for range n {
		r, ok := s.ExpendSlot(pool, slotLevel(level)).(character.SlotExpenditureExpended)
		if !ok {
			t.Fatalf("no slot of level %d to expend", level)
		}
		s = r.State()
	}
	return s
}

var (
	kinds = func(k ...character.CastRejectionKind) []character.CastRejectionKind { return k }
	l1    = func(tb testing.TB, opts ...spellOpt) content.Spell {
		tb.Helper()
		return castSpell(tb, preparedL1, 1, opts...)
	}
	focusPlain             = materialOnly(`[]`, false)
	costly                 = materialOnly(`[100]`, false)
	consumed               = materialOnly(`[]`, true)
	arcane, holy, pactWeap = character.FocusArcane, character.FocusHolySymbol, character.FocusPactWeapon
)

// Requirement: doc/specs/casting-and-concentration.md FR-055 to FR-069, FR-080, FR-081, FR-094, FR-095, FR-102, FR-107,
// FR-113, FR-114, FR-129, FR-130, FR-146, Table L - every refused cast returns CastSpellRejected with exactly the reasons
// in the fixed order, leaving the input state unchanged.
// Path: per case.
func TestCastSpellRejections(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	lw := warlockWizard(t, 5)
	spent := expendAll(t, lw, character.PoolPactMagic, 3, 2)
	l1Spent := expendAll(t, wiz, character.PoolSpellcasting, 1, 4)
	going := castingStarted(t, wiz)
	k := kinds
	cases := []struct {
		castCase
		want []character.CastRejectionKind
	}{
		{castCase{name: "L3 level 3 spell, level 2 slot", req: "FR-059", path: "unhappy (boundary)", spell: castSpell(t, preparedL3, 3), slot: choice(2)}, k(character.CastRejectionSlotLevelTooLow)},
		{castCase{name: "L5 chosen level has none remaining", req: "FR-060", path: "unhappy (state error)", state: l1Spent, spell: l1(t), slot: choice(1)}, k(character.CastRejectionNoSlotAvailable)},
		{castCase{name: "L6 spell not prepared", req: "FR-055", path: "unhappy (state error)", spell: castSpell(t, unpreparedSpell, 1), slot: choice(1)}, k(character.CastRejectionNotPrepared)},
		{castCase{name: "L8 cantrip not known", req: "FR-055", path: "unhappy (state error)", spell: castSpell(t, spellID(0, 5), 0)}, k(character.CastRejectionNotKnown)},
		{castCase{name: "L9 slot already spent, slot spell", req: "FR-061", path: "unhappy (state error)", spell: l1(t), slot: choice(1), mods: mods(slotSpent)}, k(character.CastRejectionSlotAlreadySpentThisTurn)},
		{castCase{name: "L11 untrained armor, slot spell", req: "FR-063", path: "unhappy (state error)", spell: l1(t), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.UntrainedArmorWorn = true })}, k(character.CastRejectionUntrainedArmor)},
		{castCase{name: "L11 untrained armor, cantrip", req: "FR-063", path: "unhappy (state error)", spell: castSpell(t, cantripKnown, 0), mods: mods(func(f *character.CastingFacts) { f.UntrainedArmorWorn = true })}, k(character.CastRejectionUntrainedArmor)},
		{castCase{name: "L12 Verbal, cannot speak", req: "FR-064", path: "unhappy (state error)", spell: l1(t), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.CanSpeak = false })}, k(character.CastRejectionMissingVerbal)},
		{castCase{name: "L13 Somatic, no gestures", req: "FR-065", path: "unhappy (state error)", spell: l1(t, withComponents(`{"verbal":false,"somatic":true}`)), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.CanMakeGestures = false })}, k(character.CastRejectionMissingSomatic)},
		{castCase{name: "L14 Material, nothing to provide it", req: "FR-066", path: "unhappy (state error)", spell: l1(t, withComponents(focusPlain)), slot: choice(1)}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L16 costly Material, Arcane Focus, no materials", req: "FR-095", path: "unhappy (rule interaction)", spell: l1(t, withComponents(costly)), slot: choice(1), mods: mods(holdingFocus(arcane))}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L17 Holy Symbol is not a Wizard focus", req: "FR-068", path: "unhappy (invalid input)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(holy))}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L18 Reaction, trigger not met", req: "FR-069", path: "unhappy (state error)", spell: l1(t, withTimes(times(timeReaction))), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.TriggerMet = false })}, k(character.CastRejectionTriggerNotMet)},
		{castCase{name: "L29 triggered Bonus Action, trigger not met", req: "FR-069", path: "unhappy (state error)", spell: l1(t, withTimes(times(`{"kind":"bonusAction","trigger":"which you take when hit"}`))), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.TriggerMet = false })}, k(character.CastRejectionTriggerNotMet)},
		{castCase{name: "L19 untrained armor and slot too low", req: "FR-080", path: "unhappy (rule interaction)", spell: castSpell(t, preparedL3, 3), slot: choice(2), mods: mods(func(f *character.CastingFacts) { f.UntrainedArmorWorn = true })},
			k(character.CastRejectionUntrainedArmor, character.CastRejectionSlotLevelTooLow)},
		{castCase{name: "FR-080 every reason in the fixed order", req: "FR-080", path: "unhappy (rule interaction)", state: going, spell: castSpell(t, unpreparedSpell, 3, withTimes(times(timeReaction)), withComponents(`{"verbal":true,"somatic":true,"material":{"description":"a thing","costsGp":[],"consumesMaterial":false}}`)), slot: choice(1), mods: mods(
			func(f *character.CastingFacts) {
				*f = character.CastingFacts{UntrainedArmorWorn: true, SlotSpentThisTurn: true, UnavailableSpells: []content.SpellID{unpreparedSpell}}
			})},
			k(character.CastRejectionCannotCastSpells, character.CastRejectionSpellUnavailable, character.CastRejectionUntrainedArmor, character.CastRejectionCastingAlreadyInProgress,
				character.CastRejectionNotPrepared, character.CastRejectionTriggerNotMet, character.CastRejectionMissingVerbal, character.CastRejectionMissingSomatic,
				character.CastRejectionMissingMaterial, character.CastRejectionSlotAlreadySpentThisTurn, character.CastRejectionSlotLevelTooLow)},
		{castCase{name: "L22 no Ritual tag, cast as Ritual", req: "FR-074", path: "unhappy (invalid input)", spell: castSpell(t, ritualPrepared, 1), mode: character.CastModeRitual}, k(character.CastRejectionNotARitual)},
		{castCase{name: "L23 pouch, no free hand", req: "FR-067", path: "unhappy (state error)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(noFree, func(f *character.CastingFacts) { f.HasComponentPouch = true })}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L26 costly Material, pouch", req: "FR-067", path: "unhappy (rule interaction)", spell: l1(t, withComponents(costly)), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.HasComponentPouch = true })}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L27 materials, no free hand", req: "FR-066", path: "unhappy (state error)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(noFree, func(f *character.CastingFacts) { f.HasMaterials = true })}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L28 consumed Material, Arcane Focus", req: "FR-095", path: "unhappy (rule interaction)", spell: l1(t, withComponents(consumed)), slot: choice(1), mods: mods(holdingFocus(arcane))}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L31 cannot cast spells", req: "FR-113", path: "unhappy (state error)", spell: l1(t), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.CanCastSpells = false })}, k(character.CastRejectionCannotCastSpells)},
		{castCase{name: "L33 Wizard Ritual not in the spellbook", req: "FR-114", path: "unhappy (state error)", spell: castSpell(t, unpreparedSpell, 1, asRitual()), mode: character.CastModeRitual}, k(character.CastRejectionNotInSpellbook)},
		{castCase{name: "L34 Cleric Ritual not prepared", req: "FR-114", path: "unhappy (state error)", state: cleric5(t), class: character.CasterCleric, spell: castSpell(t, unpreparedSpell, 1, asRitual()), mode: character.CastModeRitual}, k(character.CastRejectionNotPrepared)},
		{castCase{name: "L36 Pact slot too low", req: "FR-107", path: "unhappy (boundary)", state: lw, class: character.CasterWizard, spell: castSpell(t, spellID(4, 1), 4), slot: character.SpellSlotChoicePactMagic{}},
			k(character.CastRejectionNotPrepared, character.CastRejectionSlotLevelTooLow)},
		{castCase{name: "L37 Pact slots spent", req: "FR-107", path: "unhappy (state error)", state: spent, class: character.CasterWarlock, spell: l1(t), slot: character.SpellSlotChoicePactMagic{}},
			k(character.CastRejectionNotPrepared, character.CastRejectionNoSlotAvailable)},
		{castCase{name: "L38 Wizard only, Pact slot", req: "FR-107", path: "unhappy (boundary)", spell: l1(t), slot: character.SpellSlotChoicePactMagic{}}, k(character.CastRejectionNoSlotAvailable)},
		{castCase{name: "L39 Warlock only, Spellcasting slot", req: "FR-107", path: "unhappy (boundary)", state: castingState(t, classLevelsOf(t, "warlock", 5), spellCatalogue(t)), class: character.CasterWarlock, spell: l1(t), slot: choice(1)},
			k(character.CastRejectionNotPrepared, character.CastRejectionNoSlotAvailable)},
		{castCase{name: "L45 casting in progress, any spell", req: "FR-102", path: "unhappy (state error)", state: going, spell: castSpell(t, cantripKnown, 0)}, k(character.CastRejectionCastingAlreadyInProgress)},
		{castCase{name: "L51 pact weapon without the feature kind", req: "FR-068", path: "unhappy (rule interaction)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(pactWeap))}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L53 costly Material, pact weapon", req: "FR-095", path: "unhappy (rule interaction)", spell: l1(t, withComponents(costly)), slot: choice(1), mods: mods(holdingFocus(pactWeap), features(pactWeap))}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L55 Holy Symbol with a pact weapon kind", req: "FR-068", path: "unhappy (rule interaction)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(holy), features(pactWeap))}, k(character.CastRejectionMissingMaterial)},
		{castCase{name: "L56 Ritual Adept, spellbook unreadable", req: "FR-130", path: "unhappy (state error)", spell: castSpell(t, ritualInBook, 1, asRitual()), mode: character.CastModeRitual, mods: mods(func(f *character.CastingFacts) { f.CanReadSpellbook = false })}, k(character.CastRejectionSpellbookNotReadable)},
		{castCase{name: "U8 unavailable, normal", req: "FR-146", path: "unhappy (state error)", spell: l1(t), slot: choice(1), mods: mods(unavailable(preparedL1))}, k(character.CastRejectionSpellUnavailable)},
		{castCase{name: "U8 unavailable, ritual", req: "FR-146", path: "unhappy (state error)", spell: castSpell(t, ritualPrepared, 1, asRitual()), mode: character.CastModeRitual, mods: mods(unavailable(ritualPrepared))}, k(character.CastRejectionSpellUnavailable)},
		{castCase{name: "U8 unavailable, feature-granted", req: "FR-146", path: "unhappy (state error)", spell: l1(t), mode: character.CastModeFeatureGranted, mods: mods(unavailable(preparedL1))}, k(character.CastRejectionSpellUnavailable)},
		{castCase{name: "U8 unavailable, scroll", req: "FR-146", path: "unhappy (state error)", spell: l1(t), mode: character.CastModeSpellScroll, noClass: true, terms: terms13(t, 0), mods: mods(unavailable(preparedL1))}, k(character.CastRejectionSpellUnavailable)},
		{castCase{name: "U9 unavailable and cannot cast", req: "FR-146, FR-080", path: "unhappy (rule interaction)", spell: l1(t), slot: choice(1), mods: mods(unavailable(preparedL1), func(f *character.CastingFacts) { f.CanCastSpells = false })},
			k(character.CastRejectionCannotCastSpells, character.CastRejectionSpellUnavailable)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := c.command(t)
			before := cmd.State
			got, err := character.NewCastSpell().Do(context.Background(), cmd)
			rej, ok := got.(character.CastSpellRejected)
			if err != nil || !ok {
				t.Fatalf("result = %#v, %v, want CastSpellRejected [%s]", got, err, c.req)
			}
			if r := rej.Rejection().Reasons(); !slices.Equal(r, c.want) {
				t.Errorf("reasons = %v, want %v [%s]", r, c.want, c.req)
			}
			if !reflect.DeepEqual(cmd.State, before) {
				t.Errorf("the state changed on a rejection [FR-081]")
			}
		})
	}
}

func mods(m ...func(*character.CastingFacts)) []func(*character.CastingFacts) { return m }
func features(k ...character.SpellcastingFocusKind) func(*character.CastingFacts) {
	return func(f *character.CastingFacts) { f.FeatureFocusKinds = k }
}
func unavailable(ids ...content.SpellID) func(*character.CastingFacts) {
	return func(f *character.CastingFacts) { f.UnavailableSpells = ids }
}

// terms13 are scroll terms: DC 13, +5, and a cantrip scaling level of cantrip (0 for a level 1+ spell).
func terms13(tb testing.TB, cantrip int) *character.SpellCastTerms {
	tb.Helper()
	v, err := character.NewSpellCastTerms(13, 5, cantrip)
	if err != nil {
		tb.Fatal(err)
	}
	return &v
}

// castingStarted returns the state after starting a one-hour casting of the level 1 spell as a Wizard.
func castingStarted(t *testing.T, s character.SpellcastingState) character.SpellcastingState {
	t.Helper()
	res, err := character.NewCastSpell().Do(context.Background(), castCase{state: s, spell: castSpell(t, preparedL1, 1, withTimes(times(timeHour))), slot: choice(1)}.command(t))
	started, ok := res.(character.CastSpellStarted)
	if err != nil || !ok {
		t.Fatalf("start = %#v, %v", res, err)
	}
	return started.State()
}

// cast expectations of the success rows.
type castWant struct {
	source        character.CastSource
	castAt, scale int // 0: absent
	slotLevel     int // 0: no slot spent
	pool          character.SpellSlotPool
	events        []string
	counts        bool
}

// Requirement: doc/specs/casting-and-concentration.md FR-056 to FR-058, FR-062, FR-066, FR-067, FR-082, FR-083, FR-094, FR-095,
// FR-107, FR-113, FR-115, FR-129, Table L - an accepted cast reports its source, levels and slot, spends exactly the slot
// named and emits its events in order.
// Path: per case.
func TestCastSpellSucceeds(t *testing.T) {
	t.Parallel()
	lw := warlockWizard(t, 5)
	lwPrepared := castingState(t, classLevelsOf(t, "warlock", 5, "wizard", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(3, 1), Spellbook: character.NewSpellbook(spellIDs(3, 1))})
	druid := castingState(t, classLevelsOf(t, "druid", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterDruid, Prepared: spellIDs(3, 1)})
	twoModes := times(labelled("Overgrowth", timeAction), labelled("Enrichment", timeEight))
	slotSpell := []string{"SpellSlotExpended", "SpellCast"}
	cases := []struct {
		castCase
		want castWant
	}{
		{castCase{name: "L1 level 1 spell, level 1 slot", req: "FR-057", path: "happy", spell: l1(t), slot: choice(1)}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L2 level 1 spell, level 3 slot", req: "FR-058", path: "happy", spell: l1(t), slot: choice(3)}, castWant{source: character.CastSourceSlot, castAt: 3, scale: 3, slotLevel: 3, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L4 level 3 spell, level 3 slot", req: "FR-057", path: "unhappy (boundary)", spell: castSpell(t, preparedL3, 3), slot: choice(3)}, castWant{source: character.CastSourceSlot, castAt: 3, scale: 3, slotLevel: 3, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L7 cantrip known", req: "FR-056 FR-038", path: "happy", spell: castSpell(t, cantripKnown, 0)}, castWant{source: character.CastSourceCantrip, scale: 5, events: []string{"SpellCast"}}},
		{castCase{name: "L10 cantrip after a slot this turn", req: "FR-062", path: "unhappy (rule interaction)", spell: castSpell(t, cantripKnown, 0), mods: mods(slotSpent)}, castWant{source: character.CastSourceCantrip, scale: 5, events: []string{"SpellCast"}}},
		{castCase{name: "L15 Material, Arcane Focus held", req: "FR-095", path: "happy", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(arcane))}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L24 Material, pouch and a free hand", req: "FR-067 FR-094", path: "happy", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.HasComponentPouch = true })}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L25 Material, Arcane Focus held, no free hand", req: "FR-095", path: "unhappy (rule interaction)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(arcane), noFree)}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "FR-066 actual materials and a free hand, costly", req: "FR-066", path: "happy", spell: l1(t, withComponents(costly)), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.HasMaterials = true })}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L30 Bonus Action without a trigger, trigger not met", req: "FR-069", path: "unhappy (rule interaction)", spell: l1(t, withTimes(times(timeBonus))), slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.TriggerMet = false })}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L35 Pact slot, level 3 spell", req: "FR-107", path: "happy", state: lwPrepared, class: character.CasterWizard, spell: castSpell(t, preparedL3, 3), slot: character.SpellSlotChoicePactMagic{}}, castWant{source: character.CastSourceSlot, castAt: 3, scale: 3, slotLevel: 3, pool: character.PoolPactMagic, events: slotSpell, counts: true}},
		{castCase{name: "L46 two casting modes, Overgrowth", req: "FR-115 FR-069", path: "unhappy (rule interaction)", state: druid, class: character.CasterDruid, spell: castSpell(t, preparedL3, 3, withTimes(twoModes)), slot: choice(3)}, castWant{source: character.CastSourceSlot, castAt: 3, scale: 3, slotLevel: 3, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L48 no mode given, the first is used", req: "FR-115", path: "unhappy (empty and minimal)", state: druid, class: character.CasterDruid, spell: castSpell(t, preparedL3, 3, withTimes(twoModes)), slot: choice(3)}, castWant{source: character.CastSourceSlot, castAt: 3, scale: 3, slotLevel: 3, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L50 pact weapon as focus", req: "FR-068 FR-129", path: "happy", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(pactWeap), features(pactWeap), noFree)}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L52 Book of Shadows, no free hand", req: "FR-095 FR-129", path: "unhappy (rule interaction)", spell: l1(t, withComponents(focusPlain)), slot: choice(1), mods: mods(holdingFocus(character.FocusBookOfShadows), features(character.FocusBookOfShadows), noFree)}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: slotSpell, counts: true}},
		{castCase{name: "L57 prepared Ritual spell, book unreadable, Ritual", req: "FR-130", path: "unhappy (rule interaction)", spell: castSpell(t, ritualPrepared, 1, asRitual(), withTimes(times(timeAction))), mode: character.CastModeRitual, mods: mods(func(f *character.CastingFacts) { f.CanReadSpellbook = false })}, castWant{source: character.CastSourceRitual, events: []string{"ConcentrationBegan"}}},
		{castCase{name: "FR-070 a Concentration slot spell", req: "FR-075 FR-083", path: "happy", spell: l1(t, concentrating()), slot: choice(1)}, castWant{source: character.CastSourceSlot, castAt: 1, scale: 1, slotLevel: 1, pool: character.PoolSpellcasting, events: []string{"SpellSlotExpended", "ConcentrationBegan", "SpellCast"}, counts: true}},
		{castCase{name: "L58 Cleric prepared Ritual, book unreadable", req: "FR-130", path: "unhappy (rule interaction)", state: cleric5(t), class: character.CasterCleric, spell: castSpell(t, ritualPrepared, 1, asRitual()), mode: character.CastModeRitual, mods: mods(func(f *character.CastingFacts) { f.CanReadSpellbook = false })}, castWant{source: character.CastSourceRitual, events: []string{"ConcentrationBegan"}}},
	}
	_ = lw
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := c.command(t)
			got, err := character.NewCastSpell().Do(context.Background(), cmd)
			switch r := got.(type) {
			case character.CastSpellCast:
				checkCast(t, c.want, r.Report(), r.State(), r.Events(), cmd.State, c.req)
			case character.CastSpellStarted:
				if c.want.source != character.CastSourceRitual || !slices.Equal(eventNames(r.Events()), c.want.events) {
					t.Errorf("started with events %v, want %+v [%s]", eventNames(r.Events()), c.want, c.req)
				}
			default:
				t.Fatalf("result = %#v, %v [%s]", got, err, c.req)
			}
		})
	}
}

func checkCast(t *testing.T, want castWant, r character.CastReport, after character.SpellcastingState, events []character.Event, before character.SpellcastingState, req string) {
	t.Helper()
	at, hasAt := r.CastAtLevel()
	sc, hasSc := r.ScalingLevel()
	if r.Source() != want.source || at != want.castAt || hasAt != (want.castAt > 0) || sc != want.scale || hasSc != (want.scale > 0) || r.CountsAsSlotSpell() != want.counts {
		t.Errorf("report = source %v castAt %d (%v) scaling %d (%v) counts %v, want %+v [%s]", r.Source(), at, hasAt, sc, hasSc, r.CountsAsSlotSpell(), want, req)
	}
	pool, lvl, spent := r.Slot()
	if spent != (want.slotLevel > 0) || (spent && (lvl != slotLevel(want.slotLevel) || pool != want.pool)) {
		t.Errorf("slot = %v %v %v, want level %d of %v [%s]", pool, lvl, spent, want.slotLevel, want.pool, req)
	}
	if got := eventNames(events); !slices.Equal(got, want.events) {
		t.Errorf("events = %v, want %v [FR-083]", got, want.events)
	}
	for _, p := range []character.SpellSlotPool{character.PoolSpellcasting, character.PoolPactMagic} {
		for l := 1; l <= 9; l++ {
			wantSpent := 0
			if spent && p == pool && l == want.slotLevel {
				wantSpent = 1
			}
			if d := before.Slots(p).Remaining(slotLevel(l)) - after.Slots(p).Remaining(slotLevel(l)); d != wantSpent {
				t.Errorf("slots of %v level %d dropped by %d, want %d [%s]", p, l, d, wantSpent, req)
			}
		}
	}
}

// Requirement: doc/specs/casting-and-concentration.md FR-106, FR-108, FR-115, FR-124, FR-143, FR-144, FR-149, FR-150,
// Tables L, S, U - contradictory command input is an error matching srderr.ErrInvalidInput, before any state change.
// Path: unhappy (invalid input) per case.
func TestCastSpellRejectsContradictoryInput(t *testing.T) {
	t.Parallel()
	wiz := wizard5(t)
	stats := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	wisStats := character.StatsForAbility(mustFeatureAbility(t, ability.Wisdom), ability.MustScore(14).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	twoModes := times(labelled("Overgrowth", timeAction), labelled("Enrichment", timeEight))
	other, err := content.NewCastingTimeMode("Other", content.NewCastingTimeMagicAction())
	if err != nil {
		t.Fatal(err)
	}
	cantrip, lvl3, lvl1 := castSpell(t, cantripKnown, 0), castSpell(t, preparedL3, 3), l1(t)
	cases := []castCase{
		{name: "L40 casting class not in the state", req: "FR-108", class: character.CasterCleric, spell: lvl1, slot: choice(1)},
		{name: "L41 cantrip with a slot choice", req: "FR-106", spell: cantrip, slot: choice(1)},
		{name: "L42 level 1 spell, normal, no slot choice", req: "FR-106", spell: lvl1},
		{name: "L43 Ritual with a slot choice", req: "FR-106", spell: castSpell(t, ritualPrepared, 1, asRitual()), mode: character.CastModeRitual, slot: choice(1)},
		{name: "FR-106 Ritual cantrip", req: "FR-106", spell: castSpell(t, cantripKnown, 0, asRitual()), mode: character.CastModeRitual},
		{name: "FR-106 zero slot level", req: "FR-106", spell: lvl1, slot: character.NewSpellSlotChoiceSpellcasting(character.SlotLevel{})},
		{name: "FR-106 zero mode and spell", req: "FR-106", mode: 0, spell: content.Spell{}},
		{name: "L44 a mode for a single casting time", req: "FR-115", spell: lvl1, slot: choice(1), timeMode: other},
		{name: "L49 a mode not among the spell's modes", req: "FR-115", spell: castSpell(t, preparedL3, 3, withTimes(twoModes)), slot: choice(3), timeMode: other},
		{name: "S4 feature-granted with a slot choice", req: "FR-124", spell: lvl1, mode: character.CastModeFeatureGranted, slot: choice(1)},
		{name: "S19 flag with a cantrip", req: "FR-144", spell: cantrip, mode: character.CastModeFeatureGranted, counts: true},
		{name: "S19 level on a cantrip", req: "FR-144", spell: cantrip, mode: character.CastModeFeatureGranted, featureLevel: 3},
		{name: "S19 level below the spell's", req: "FR-144", spell: lvl3, mode: character.CastModeFeatureGranted, featureLevel: 2},
		{name: "S19 level 10", req: "FR-144", spell: lvl1, mode: character.CastModeFeatureGranted, featureLevel: 10},
		{name: "S19 negative level", req: "FR-144", spell: lvl1, mode: character.CastModeFeatureGranted, featureLevel: -1},
		{name: "S20 level with a normal cast", req: "FR-144", spell: lvl1, slot: choice(1), featureLevel: 3},
		{name: "S20 flag with a normal cast", req: "FR-144", spell: lvl1, slot: choice(1), counts: true},
		{name: "S24 class-less feature-granted, no stats", req: "FR-149", spell: lvl1, mode: character.CastModeFeatureGranted, noClass: true},
		{name: "FR-149 zero stats", req: "FR-149", spell: lvl1, mode: character.CastModeFeatureGranted, noClass: true, stats: &character.SpellcastingStats{}},
		{name: "FR-149 stats of another ability than the class", req: "FR-149", spell: lvl1, mode: character.CastModeFeatureGranted, stats: &wisStats},
		{name: "FR-082 stats with a normal cast of another class ability", req: "FR-082", spell: lvl1, slot: choice(1), stats: &wisStats},
		{name: "FR-082 a normal cast without a class", req: "FR-108", spell: lvl1, slot: choice(1), noClass: true},
		{name: "U6 scroll without terms", req: "FR-143", spell: lvl1, mode: character.CastModeSpellScroll, noClass: true},
		{name: "U6 terms with a normal cast", req: "FR-143", spell: lvl1, slot: choice(1), terms: terms13(t, 0)},
		{name: "U6 scroll with a casting class", req: "FR-143", spell: lvl1, mode: character.CastModeSpellScroll, terms: terms13(t, 0)},
		{name: "U6 scroll with a slot choice", req: "FR-143", spell: lvl1, mode: character.CastModeSpellScroll, noClass: true, terms: terms13(t, 0), slot: choice(1)},
		{name: "U6 scroll with stats", req: "FR-143", spell: lvl1, mode: character.CastModeSpellScroll, noClass: true, terms: terms13(t, 0), stats: &stats},
		{name: "U6 cantrip scaling level on a level 3 spell", req: "FR-143", spell: lvl3, mode: character.CastModeSpellScroll, noClass: true, terms: terms13(t, 5)},
		{name: "U6 cantrip without the scaling level", req: "FR-143", spell: cantrip, mode: character.CastModeSpellScroll, noClass: true, terms: terms13(t, 0)},
		{name: "U21 duplicated without terms", req: "FR-150", spell: lvl1, mode: character.CastModeDuplicated, noClass: true},
		{name: "U21 duplicated with a class", req: "FR-150", spell: lvl1, mode: character.CastModeDuplicated, terms: terms13(t, 0)},
		{name: "U21 duplicated with a slot choice", req: "FR-150", spell: lvl1, mode: character.CastModeDuplicated, noClass: true, terms: terms13(t, 0), slot: choice(1)},
		{name: "U21 duplicated with a feature level", req: "FR-150", spell: lvl1, mode: character.CastModeDuplicated, noClass: true, terms: terms13(t, 0), featureLevel: 3},
		{name: "U21 duplicated with the slot flag", req: "FR-150", spell: lvl1, mode: character.CastModeDuplicated, noClass: true, terms: terms13(t, 0), counts: true},
		{name: "U21 duplicated with a casting time mode", req: "FR-150", spell: lvl1, mode: character.CastModeDuplicated, noClass: true, terms: terms13(t, 0), timeMode: other},
		{name: "U21 cantrip scaling level on a level 3 spell", req: "FR-150", spell: lvl3, mode: character.CastModeDuplicated, noClass: true, terms: terms13(t, 5)},
		{name: "invalid class value", req: "FR-108", class: character.SpellcastingClass(99), spell: lvl1, slot: choice(1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			c.state = wiz
			got, err := c.run(t)
			if got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("Do = %#v, %v, want an error matching ErrInvalidInput [%s]", got, err, c.req)
			}
		})
	}
	t.Run("L40 matches ErrClassNotHeld", func(t *testing.T) {
		t.Parallel()
		_, err := castCase{state: wiz, class: character.CasterCleric, spell: lvl1, slot: choice(1)}.run(t)
		if !errors.Is(err, character.ErrClassNotHeld) {
			t.Errorf("err = %v, want ErrClassNotHeld [FR-108]", err)
		}
	})
	t.Run("zero use case and cancelled context", func(t *testing.T) {
		t.Parallel()
		if _, err := (character.CastSpell{}).Do(context.Background(), castCase{state: wiz, spell: lvl1, slot: choice(1)}.command(t)); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero use case err = %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := character.NewCastSpell().Do(ctx, castCase{state: wiz, spell: lvl1, slot: choice(1)}.command(t)); !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled err = %v, want context.Canceled", err)
		}
	})
}

func mustFeatureAbility(tb testing.TB, a ability.Ability) character.FeatureSpellAbility {
	tb.Helper()
	f, err := character.NewFeatureSpellAbility(a)
	if err != nil {
		tb.Fatal(err)
	}
	return f
}

func mustCast(t *testing.T, c castCase) character.CastSpellCast {
	t.Helper()
	got, err := c.run(t)
	r, ok := got.(character.CastSpellCast)
	if err != nil || !ok {
		t.Fatalf("cast = %#v, %v, want CastSpellCast", got, err)
	}
	return r
}

func mustStart(t *testing.T, c castCase) character.CastSpellStarted {
	t.Helper()
	got, err := c.run(t)
	r, ok := got.(character.CastSpellStarted)
	if err != nil || !ok {
		t.Fatalf("cast = %#v, %v, want CastSpellStarted", got, err)
	}
	return r
}

// concentratingWizard is a Wizard 5 concentrating on the level 1 spell s1_03 (cast with a level 1 slot).
func concentratingWizard(t *testing.T) character.SpellcastingState {
	t.Helper()
	return mustCast(t, castCase{spell: castSpell(t, spellID(1, 3), 1, concentrating()), slot: choice(1)}).State()
}

// Requirement: doc/specs/casting-and-concentration.md FR-070, FR-073, FR-115, FR-131, Table T - a casting of a minute or
// more, or a Ritual, starts a CastingInProgress with the whole minutes of the chosen casting time (a Ritual adds 10) and
// spends no slot.
// Path: per case.
func TestLongerCastingTotals(t *testing.T) {
	t.Parallel()
	twoModes := times(labelled("Overgrowth", timeAction), labelled("Enrichment", timeEight))
	enrichment, err := content.NewCastingTimeMode("Enrichment", content.NewCastingTimeMagicAction())
	_ = enrichment
	if err != nil {
		t.Fatal(err)
	}
	mode8h, err := content.NewCastingTimeHours(8)
	if err != nil {
		t.Fatal(err)
	}
	enrichmentMode, err := content.NewCastingTimeMode("Enrichment", mode8h)
	if err != nil {
		t.Fatal(err)
	}
	druid := castingState(t, classLevelsOf(t, "druid", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterDruid, Prepared: spellIDs(3, 1)})
	ritual := func(time string) castCase {
		return castCase{spell: castSpell(t, ritualPrepared, 1, asRitual(), withTimes(times(time))), mode: character.CastModeRitual}
	}
	normal := func(time string) castCase {
		return castCase{spell: l1(t, withTimes(times(time))), slot: choice(1)}
	}
	cases := []struct {
		name, req, path string
		c               castCase
		minutes         int // 0: an immediate SpellCast
		source          character.CastSource
		mode            content.CastingTimeMode
	}{
		{"T1 Magic action Ritual", "FR-131", "unhappy (boundary)", ritual(timeAction), 10, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T2 Bonus Action Ritual", "FR-131", "unhappy (boundary)", ritual(timeBonus), 10, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T3 Reaction Ritual, trigger met", "FR-131 FR-069", "unhappy (boundary)", ritual(timeReaction), 10, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T4 1 minute Ritual", "FR-073", "unhappy (boundary)", ritual(timeMinute), 11, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T5 10 minutes Ritual", "FR-073", "happy", ritual(timeTenMin), 20, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T6 1 hour Ritual", "FR-073", "unhappy (boundary)", ritual(timeHour), 70, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T7 8 hours Ritual", "FR-073", "happy", ritual(timeEight), 490, character.CastSourceRitual, content.CastingTimeMode{}},
		{"T8 1 minute normal", "FR-070", "unhappy (boundary)", normal(timeMinute), 1, character.CastSourceSlot, content.CastingTimeMode{}},
		{"T9 8 hours normal", "FR-070", "happy", normal(timeEight), 480, character.CastSourceSlot, content.CastingTimeMode{}},
		{"T10 Magic action normal", "FR-070", "happy", normal(timeAction), 0, character.CastSourceSlot, content.CastingTimeMode{}},
		{"T11 Plant Growth Enrichment", "FR-115 FR-070", "unhappy (rule interaction)", castCase{state: druid, class: character.CasterDruid, spell: castSpell(t, preparedL3, 3, withTimes(twoModes)), slot: choice(3), timeMode: enrichmentMode}, 480, character.CastSourceSlot, enrichmentMode},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.minutes == 0 {
				mustCast(t, c.c)
				return
			}
			before := c.c.command(t).State
			r := mustStart(t, c.c)
			cip, ok := r.State().CastingInProgress()
			if !ok || cip.TotalMinutes() != c.minutes || cip.Source() != c.source || cip.Mode() != c.mode {
				t.Errorf("CastingInProgress = %+v %v, want %d minutes, %v, mode %v [%s]", cip, ok, c.minutes, c.source, c.mode, c.req)
			}
			for l := 1; l <= 9; l++ {
				if r.State().Slots(character.PoolSpellcasting).Remaining(slotLevel(l)) != before.Slots(character.PoolSpellcasting).Remaining(slotLevel(l)) {
					t.Errorf("a slot of level %d was spent at the start [FR-070]", l)
				}
			}
			if conc, ok := r.State().Concentration(); !ok || conc.Kind() != character.ConcentrationCasting {
				t.Errorf("Concentration = %+v %v, want kind casting [FR-101]", conc, ok)
			}
			if got, want := eventNames(r.Events()), []string{"ConcentrationBegan"}; !slices.Equal(got, want) {
				t.Errorf("events = %v, want %v", got, want)
			}
		})
	}
	t.Run("T11 the slot choice is held", func(t *testing.T) {
		t.Parallel()
		r := mustStart(t, castCase{spell: l1(t, withTimes(times(timeMinute))), slot: choice(2)})
		cip, _ := r.State().CastingInProgress()
		if cip.Slot() != choice(2) || cip.Spell() != preparedL1 || cip.Class() != character.CasterWizard {
			t.Errorf("CastingInProgress = %+v, want slot level 2 held", cip)
		}
	})
}

// Requirement: doc/specs/casting-and-concentration.md FR-070 to FR-072, FR-075, FR-100 to FR-105, FR-125, FR-130, FR-138,
// FR-139, Table N (N1 to N9, N14, N23, N31) and L59 - a longer casting starts without a slot, ends the earlier
// Concentration, and CompleteCasting spends the slot, re-checks, and settles the Concentration.
// Path: per case.
func TestCompleteCasting(t *testing.T) {
	t.Parallel()
	hourSpell := func(opts ...spellOpt) content.Spell {
		return castSpell(t, preparedL1, 1, append([]spellOpt{withTimes(times(timeHour))}, opts...)...)
	}
	start := func(t *testing.T, c castCase) character.SpellcastingState {
		t.Helper()
		return mustStart(t, c).State()
	}
	complete := func(t *testing.T, s character.SpellcastingState, spell content.Spell, mod ...func(*character.CastingFacts)) (character.CompleteCastingResult, error) {
		t.Helper()
		return character.NewCompleteCasting().Do(context.Background(), character.CompleteCastingCommand{State: s, Spell: spell, Context: castContext(t, mod...)})
	}
	completed := func(t *testing.T, s character.SpellcastingState, spell content.Spell, mod ...func(*character.CastingFacts)) character.CompleteCastingCompleted {
		t.Helper()
		got, err := complete(t, s, spell, mod...)
		r, ok := got.(character.CompleteCastingCompleted)
		if err != nil || !ok {
			t.Fatalf("complete = %#v, %v", got, err)
		}
		return r
	}
	rejected := func(t *testing.T, s character.SpellcastingState, spell content.Spell, mod ...func(*character.CastingFacts)) []character.CastRejectionKind {
		t.Helper()
		got, err := complete(t, s, spell, mod...)
		r, ok := got.(character.CompleteCastingRejected)
		if err != nil || !ok {
			t.Fatalf("complete = %#v, %v", got, err)
		}
		return r.Rejection().Reasons()
	}
	t.Run("N1 start", func(t *testing.T) {
		t.Parallel()
		s := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		if _, ok := s.CastingInProgress(); !ok {
			t.Error("no CastingInProgress [FR-070]")
		}
	})
	t.Run("N2 start while concentrating on another spell", func(t *testing.T) {
		t.Parallel()
		r := mustStart(t, castCase{state: concentratingWizard(t), spell: hourSpell(), slot: choice(1)})
		if got, want := eventNames(r.Events()), []string{"ConcentrationEnded", "ConcentrationBegan"}; !slices.Equal(got, want) {
			t.Fatalf("events = %v, want %v [FR-101]", got, want)
		}
		if e := r.Events()[0].(character.ConcentrationEnded); e.Reason() != character.EndedAnotherEffect || e.Subject() != character.NewConcentrationSubjectSpell(spellID(1, 3)) {
			t.Errorf("ended = %+v", e)
		}
	})
	t.Run("N3 start with no slot of the chosen level", func(t *testing.T) {
		t.Parallel()
		s := expendAll(t, wizard5(t), character.PoolSpellcasting, 1, 4)
		got, err := castCase{state: s, spell: hourSpell(), slot: choice(1)}.run(t)
		if r, ok := got.(character.CastSpellRejected); err != nil || !ok || !slices.Equal(r.Rejection().Reasons(), kinds(character.CastRejectionNoSlotAvailable)) {
			t.Errorf("cast = %#v, %v [FR-103]", got, err)
		}
	})
	t.Run("N4 complete a non-Concentration spell", func(t *testing.T) {
		t.Parallel()
		before := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		r := completed(t, before, hourSpell())
		if got, want := eventNames(r.Events()), []string{"SpellSlotExpended", "ConcentrationEnded", "SpellCast"}; !slices.Equal(got, want) {
			t.Fatalf("events = %v, want %v [FR-083]", got, want)
		}
		if e := r.Events()[1].(character.ConcentrationEnded); e.Reason() != character.EndedCastingCompleted {
			t.Errorf("reason = %v, want casting_completed", e.Reason())
		}
		if _, ok := r.State().CastingInProgress(); ok {
			t.Error("the casting is still in progress")
		}
		if _, ok := r.State().Concentration(); ok {
			t.Error("Concentration is still held [FR-138]")
		}
		if got := r.State().Slots(character.PoolSpellcasting).Remaining(slotLevel(1)); got != 3 {
			t.Errorf("level 1 slots = %d, want 3 [FR-071]", got)
		}
		if at, _ := r.Report().CastAtLevel(); at != 1 || r.Report().Source() != character.CastSourceSlot {
			t.Errorf("report = %+v", r.Report())
		}
	})
	t.Run("N5 complete a Concentration spell", func(t *testing.T) {
		t.Parallel()
		before := start(t, castCase{spell: hourSpell(concentrating()), slot: choice(1)})
		r := completed(t, before, hourSpell(concentrating()))
		if got, want := eventNames(r.Events()), []string{"SpellSlotExpended", "SpellCast"}; !slices.Equal(got, want) {
			t.Fatalf("events = %v, want %v [FR-138]", got, want)
		}
		if conc, ok := r.State().Concentration(); !ok || conc.Kind() != character.ConcentrationEffect || conc.Subject() != character.NewConcentrationSubjectSpell(preparedL1) {
			t.Errorf("Concentration = %+v %v, want effect on the spell", conc, ok)
		}
	})
	t.Run("N6 complete after the slot was spent elsewhere", func(t *testing.T) {
		t.Parallel()
		started := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		s := expendAll(t, started, character.PoolSpellcasting, 1, 4)
		if got := rejected(t, s, hourSpell()); !slices.Equal(got, kinds(character.CastRejectionNoSlotAvailable)) {
			t.Errorf("reasons = %v [FR-104]", got)
		}
		if _, ok := s.CastingInProgress(); !ok {
			t.Error("the casting left progress [FR-104]")
		}
	})
	t.Run("N7 complete with a slot already spent this turn", func(t *testing.T) {
		t.Parallel()
		started := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		if got := rejected(t, started, hourSpell(), slotSpent); !slices.Equal(got, kinds(character.CastRejectionSlotAlreadySpentThisTurn)) {
			t.Errorf("reasons = %v [FR-104]", got)
		}
	})
	t.Run("N8 complete with a replacement slot", func(t *testing.T) {
		t.Parallel()
		started := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		got, err := character.NewCompleteCasting().Do(context.Background(), character.CompleteCastingCommand{State: started, Spell: hourSpell(), Context: castContext(t), Slot: choice(2)})
		r, ok := got.(character.CompleteCastingCompleted)
		if err != nil || !ok {
			t.Fatalf("complete = %#v, %v", got, err)
		}
		if at, _ := r.Report().CastAtLevel(); at != 2 || r.State().Slots(character.PoolSpellcasting).Remaining(slotLevel(2)) != 2 {
			t.Errorf("castAt = %d, level 2 slots = %d, want 2 and 2 [FR-103]", at, r.State().Slots(character.PoolSpellcasting).Remaining(slotLevel(2)))
		}
	})
	t.Run("N9 complete a Ritual with every slot spent", func(t *testing.T) {
		t.Parallel()
		spell := castSpell(t, ritualPrepared, 1, asRitual())
		started := start(t, castCase{spell: spell, mode: character.CastModeRitual})
		s := started
		for l, n := range map[int]int{1: 4, 2: 3, 3: 2} {
			s = expendAll(t, s, character.PoolSpellcasting, l, n)
		}
		r := completed(t, s, spell, slotSpent)
		_, _, spent := r.Report().Slot()
		_, hasAt := r.Report().CastAtLevel()
		if r.Report().Source() != character.CastSourceRitual || spent || hasAt || r.Report().CountsAsSlotSpell() || !slices.Equal(eventNames(r.Events()), []string{"ConcentrationEnded", "SpellCast"}) {
			t.Errorf("report = %+v events %v [FR-103]", r.Report(), eventNames(r.Events()))
		}
	})
	t.Run("N14 complete with nothing in progress", func(t *testing.T) {
		t.Parallel()
		if got := rejected(t, wizard5(t), hourSpell()); !slices.Equal(got, kinds(character.CastRejectionCastingNotInProgress)) {
			t.Errorf("reasons = %v [FR-105]", got)
		}
	})
	t.Run("N23 complete a different spell", func(t *testing.T) {
		t.Parallel()
		started := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		got, err := complete(t, started, castSpell(t, spellID(1, 2), 1))
		if got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("complete = %#v, %v, want ErrInvalidInput [FR-125]", got, err)
		}
	})
	t.Run("N31 a Concentration spell then a non-Concentration spell", func(t *testing.T) {
		t.Parallel()
		first := completed(t, start(t, castCase{spell: hourSpell(concentrating()), slot: choice(1)}), hourSpell(concentrating()))
		second := castSpell(t, spellID(1, 2), 1, withTimes(times(timeHour)))
		r := mustStart(t, castCase{state: first.State(), spell: second, slot: choice(1)})
		if got, want := eventNames(r.Events()), []string{"ConcentrationEnded", "ConcentrationBegan"}; !slices.Equal(got, want) {
			t.Fatalf("events = %v, want %v", got, want)
		}
		done := completed(t, r.State(), second)
		if got, want := eventNames(done.Events()), []string{"SpellSlotExpended", "ConcentrationEnded", "SpellCast"}; !slices.Equal(got, want) {
			t.Errorf("events = %v, want %v [FR-138]", got, want)
		}
	})
	t.Run("L59 complete the Ritual Adept casting with the book unreadable", func(t *testing.T) {
		t.Parallel()
		spell := castSpell(t, ritualInBook, 1, asRitual())
		started := start(t, castCase{spell: spell, mode: character.CastModeRitual})
		if got := rejected(t, started, spell, func(f *character.CastingFacts) { f.CanReadSpellbook = false }); !slices.Equal(got, kinds(character.CastRejectionSpellbookNotReadable)) {
			t.Errorf("reasons = %v [FR-139]", got)
		}
		completed(t, started, spell)
	})
	t.Run("FR-139 untrained armor and components are re-checked", func(t *testing.T) {
		t.Parallel()
		spell := hourSpell(withComponents(`{"verbal":true,"somatic":true,"material":{"description":"a thing","costsGp":[],"consumesMaterial":false}}`))
		started := start(t, castCase{spell: spell, slot: choice(1), mods: mods(func(f *character.CastingFacts) { f.HasMaterials = true })})
		got := rejected(t, started, spell, func(f *character.CastingFacts) {
			f.UntrainedArmorWorn, f.CanSpeak, f.CanMakeGestures, f.HasFreeHand = true, false, false, false
		})
		want := kinds(character.CastRejectionUntrainedArmor, character.CastRejectionMissingVerbal, character.CastRejectionMissingSomatic, character.CastRejectionMissingMaterial)
		if !slices.Equal(got, want) {
			t.Errorf("reasons = %v, want %v", got, want)
		}
	})
	t.Run("U11 the spell became unavailable", func(t *testing.T) {
		t.Parallel()
		started := start(t, castCase{spell: hourSpell(), slot: choice(1)})
		if got := rejected(t, started, hourSpell(), unavailable(preparedL1)); !slices.Equal(got, kinds(character.CastRejectionSpellUnavailable)) {
			t.Errorf("reasons = %v [FR-146]", got)
		}
	})
	t.Run("a Pact slot casting spends the Pact slot", func(t *testing.T) {
		t.Parallel()
		lw := castingState(t, classLevelsOf(t, "warlock", 5, "wizard", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 1), Spellbook: character.NewSpellbook(spellIDs(1, 1))})
		started := start(t, castCase{state: lw, spell: hourSpell(), slot: character.SpellSlotChoicePactMagic{}})
		r := completed(t, started, hourSpell())
		if p, l, ok := r.Report().Slot(); !ok || p != character.PoolPactMagic || l != slotLevel(3) || r.State().Slots(character.PoolPactMagic).Remaining(slotLevel(3)) != 1 {
			t.Errorf("slot = %v %v %v", p, l, ok)
		}
	})
	t.Run("a replacement slot for a casting with no slot, and a missing spell, are errors", func(t *testing.T) {
		t.Parallel()
		spell := castSpell(t, ritualPrepared, 1, asRitual())
		started := start(t, castCase{spell: spell, mode: character.CastModeRitual})
		for name, cmd := range map[string]character.CompleteCastingCommand{
			"replacement slot":       {State: started, Spell: spell, Context: castContext(t), Slot: choice(1)},
			"zero spell":             {State: started, Context: castContext(t)},
			"zero state":             {Spell: spell, Context: castContext(t)},
			"level on a non-feature": {State: started, Spell: spell, Context: castContext(t), FeatureCastLevel: 3},
			"stats with a Ritual":    {State: started, Spell: spell, Context: castContext(t), Terms: terms13(t, 0)},
		} {
			if got, err := character.NewCompleteCasting().Do(context.Background(), cmd); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("%s: Do = %#v, %v, want ErrInvalidInput", name, got, err)
			}
		}
		if _, err := (character.CompleteCasting{}).Do(context.Background(), character.CompleteCastingCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("zero use case err = %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := character.NewCompleteCasting().Do(ctx, character.CompleteCastingCommand{State: started, Spell: spell, Context: castContext(t)}); !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled err = %v", err)
		}
	})
}

// warlock11Wizard5 is a Warlock 11 / Wizard 5 with the Wizard's level 1 spell s1_01 prepared (Table S baseline).
func warlock11Wizard5(t *testing.T) character.SpellcastingState {
	t.Helper()
	return castingState(t, classLevelsOf(t, "warlock", 11, "wizard", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterWizard, Prepared: spellIDs(1, 1), Spellbook: character.NewSpellbook(spellIDs(1, 1))})
}

// Requirement: doc/specs/casting-and-concentration.md FR-121 to FR-124, FR-140, FR-144, FR-149, Table S - a feature-granted
// cast spends no slot, skips the list and slot checks, keeps every other check, and reports its levels, source and figures.
// Path: per case.
func TestFeatureGrantedCasts(t *testing.T) {
	t.Parallel()
	s := warlock11Wizard5(t)
	l6 := castSpell(t, level6Spell, 6)
	base := func(c castCase) castCase {
		if c.state.IsZero() {
			c.state = s
		}
		c.class, c.mode = character.CasterWarlock, character.CastModeFeatureGranted
		if c.spell.ID() == "" {
			c.spell = l6
		}
		return c
	}
	casting := func(f *character.CastingFacts) {}
	_ = casting
	wisStats := character.StatsForAbility(mustFeatureAbility(t, ability.Wisdom), ability.MustScore(14).Modifier(), character.ProficiencyBonusFor(character.MustLevel(16)))
	cases := []struct {
		castCase
		want castWant
	}{
		{base(castCase{name: "S1 level 6 spell", req: "FR-121 FR-123", path: "happy"}), castWant{source: character.CastSourceFeatureGranted, castAt: 6, scale: 6, events: []string{"SpellCast"}}},
		{base(castCase{name: "S2 slot already spent this turn", req: "FR-123", path: "unhappy (rule interaction)", mods: mods(slotSpent)}), castWant{source: character.CastSourceFeatureGranted, castAt: 6, scale: 6, events: []string{"SpellCast"}}},
		{base(castCase{name: "S10 Concentration spell, concentrating on another", req: "FR-122 FR-075 FR-083", path: "unhappy (rule interaction)", state: concentratingFeatureState(t, s), spell: castSpell(t, level6Spell, 6, concentrating())}),
			castWant{source: character.CastSourceFeatureGranted, castAt: 6, scale: 6, events: []string{"ConcentrationEnded", "ConcentrationBegan", "SpellCast"}}},
		{base(castCase{name: "S12 cantrip", req: "FR-124 FR-038", path: "unhappy (boundary)", spell: castSpell(t, spellID(0, 7), 0)}), castWant{source: character.CastSourceFeatureGranted, scale: 16, events: []string{"SpellCast"}}},
		{base(castCase{name: "S13 not on the class list, not prepared", req: "FR-122", path: "unhappy (rule interaction)", spell: castSpell(t, spellID(9, 3), 9)}), castWant{source: character.CastSourceFeatureGranted, castAt: 9, scale: 9, events: []string{"SpellCast"}}},
		{base(castCase{name: "S15 Ritual spell not in the spellbook, Wizard, book unreadable", req: "FR-140 FR-130", path: "unhappy (rule interaction)", class: character.CasterWizard, spell: castSpell(t, unpreparedSpell, 1, asRitual()), mods: mods(func(f *character.CastingFacts) { f.CanReadSpellbook = false })}),
			castWant{source: character.CastSourceFeatureGranted, castAt: 1, scale: 1, events: []string{"SpellCast"}}},
		{base(castCase{name: "S16 level 1 spell cast at level 3", req: "FR-144", path: "happy", spell: castSpell(t, preparedL1, 1), featureLevel: 3}), castWant{source: character.CastSourceFeatureGranted, castAt: 3, scale: 3, events: []string{"SpellCast"}}},
		{base(castCase{name: "S17 equal level and the flag, no slot spent this turn", req: "FR-144", path: "unhappy (boundary)", spell: castSpell(t, preparedL3, 3), featureLevel: 3, counts: true}), castWant{source: character.CastSourceFeatureGranted, castAt: 3, scale: 3, events: []string{"SpellCast"}, counts: true}},
		{base(castCase{name: "S21 no level, flag false, slot spent", req: "FR-144 FR-123", path: "unhappy (rule interaction)", spell: castSpell(t, preparedL3, 3), mods: mods(slotSpent)}), castWant{source: character.CastSourceFeatureGranted, castAt: 3, scale: 3, events: []string{"SpellCast"}}},
		{castCase{name: "S23 class-less cast with Wisdom stats", req: "FR-149", path: "happy", state: s, noClass: true, spell: l6, mode: character.CastModeFeatureGranted, stats: &wisStats}, castWant{source: character.CastSourceFeatureGranted, castAt: 6, scale: 6, events: []string{"SpellCast"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd := c.command(t)
			r := mustCast(t, c.castCase)
			checkCast(t, c.want, r.Report(), r.State(), r.Events(), cmd.State, c.req)
			if cls, has := r.Report().Class(); has != !c.noClass || (has && cls != c.class) {
				t.Errorf("class = %v %v, want %v (absent %v) [FR-147]", cls, has, c.class, c.noClass)
			}
		})
	}
	t.Run("S23 the DC and attack bonus come from the stats", func(t *testing.T) {
		t.Parallel()
		r := mustCast(t, castCase{state: s, noClass: true, spell: l6, mode: character.CastModeFeatureGranted, stats: &wisStats})
		dc, hasDC := r.Report().SaveDC()
		bonus, hasBonus := r.Report().AttackBonus()
		if !hasDC || !hasBonus || dc.Value() != 8+2+5 || bonus.Value() != 2+5 {
			t.Errorf("DC %v %v bonus %v %v, want 15 and +7", dc.Value(), hasDC, bonus.Value(), hasBonus)
		}
	})
	t.Run("FR-082 a class cast with stats reports the class figures", func(t *testing.T) {
		t.Parallel()
		stats := character.Stats(character.CasterWarlock, ability.MustScore(18).Modifier(), character.ProficiencyBonusFor(character.MustLevel(16)))
		r := mustCast(t, base(castCase{stats: &stats}))
		dc, _ := r.Report().SaveDC()
		bonus, _ := r.Report().AttackBonus()
		cls, _ := r.Report().Class()
		if dc.Value() != 8+4+5 || bonus.Value() != 4+5 || cls != character.CasterWarlock {
			t.Errorf("DC %d bonus %d class %v, want 17, +9, warlock", dc.Value(), bonus.Value(), cls)
		}
		plain := mustCast(t, base(castCase{}))
		if _, ok := plain.Report().SaveDC(); ok {
			t.Error("a report without stats carries a DC")
		}
		if _, ok := plain.Report().AttackBonus(); ok {
			t.Error("a report without stats carries an attack bonus")
		}
	})
	rejections := []struct {
		castCase
		want []character.CastRejectionKind
	}{
		{base(castCase{name: "S5 cannot cast spells", req: "FR-113 FR-122", path: "unhappy (state error)", mods: mods(func(f *character.CastingFacts) { f.CanCastSpells = false })}), kinds(character.CastRejectionCannotCastSpells)},
		{base(castCase{name: "S6 untrained armor", req: "FR-122 FR-063", path: "unhappy (state error)", mods: mods(func(f *character.CastingFacts) { f.UntrainedArmorWorn = true })}), kinds(character.CastRejectionUntrainedArmor)},
		{base(castCase{name: "S7 cannot speak", req: "FR-122 FR-064", path: "unhappy (state error)", spell: castSpell(t, level6Spell, 6), mods: mods(func(f *character.CastingFacts) { f.CanSpeak = false })}), kinds(character.CastRejectionMissingVerbal)},
		{base(castCase{name: "S8 Reaction, trigger not met", req: "FR-122 FR-069", path: "unhappy (state error)", spell: castSpell(t, level6Spell, 6, withTimes(times(timeReaction))), mods: mods(func(f *character.CastingFacts) { f.TriggerMet = false })}), kinds(character.CastRejectionTriggerNotMet)},
		{base(castCase{name: "S9 casting in progress", req: "FR-122 FR-102", path: "unhappy (state error)", state: castingStarted(t, s)}), kinds(character.CastRejectionCastingAlreadyInProgress)},
		{base(castCase{name: "S18 flagged, slot already spent", req: "FR-144 FR-061", path: "unhappy (state error)", spell: castSpell(t, preparedL3, 3), featureLevel: 3, counts: true, mods: mods(slotSpent)}), kinds(character.CastRejectionSlotAlreadySpentThisTurn)},
	}
	for _, c := range rejections {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.name[:2] == "S9" {
				c.state = castingStartedFor(t, s)
			}
			got, err := c.run(t)
			if r, ok := got.(character.CastSpellRejected); err != nil || !ok || !slices.Equal(r.Rejection().Reasons(), c.want) {
				t.Errorf("cast = %#v, %v, want rejected %v [%s]", got, err, c.want, c.req)
			}
		})
	}
	t.Run("S3 a feature-granted cast then a slot spell the same turn", func(t *testing.T) {
		t.Parallel()
		first := mustCast(t, base(castCase{}))
		second := mustCast(t, castCase{state: first.State(), spell: castSpell(t, preparedL1, 1), slot: choice(1)})
		if second.Report().Source() != character.CastSourceSlot {
			t.Errorf("source = %v", second.Report().Source())
		}
	})
	t.Run("S11 a one-hour feature-granted casting completes with every slot spent", func(t *testing.T) {
		t.Parallel()
		hour := castSpell(t, level6Spell, 6, withTimes(times(timeHour)))
		started := mustStart(t, base(castCase{spell: hour}))
		cip, _ := started.State().CastingInProgress()
		if cip.Source() != character.CastSourceFeatureGranted || cip.TotalMinutes() != 60 || cip.Class() != character.CasterWarlock {
			t.Fatalf("CastingInProgress = %+v", cip)
		}
		st := expendAll(t, started.State(), character.PoolPactMagic, 5, 3)
		got, err := character.NewCompleteCasting().Do(context.Background(), character.CompleteCastingCommand{State: st, Spell: hour, Context: castContext(t, slotSpent)})
		r, ok := got.(character.CompleteCastingCompleted)
		if err != nil || !ok {
			t.Fatalf("complete = %#v, %v", got, err)
		}
		if _, _, spent := r.Report().Slot(); spent || r.Report().Source() != character.CastSourceFeatureGranted {
			t.Errorf("report = %+v", r.Report())
		}
		if at, _ := r.Report().CastAtLevel(); at != 6 {
			t.Errorf("castAt = %d, want 6", at)
		}
	})
	t.Run("S22 a flagged completion re-checks the slot rule and reports the level", func(t *testing.T) {
		t.Parallel()
		hour := castSpell(t, preparedL3, 3, withTimes(times(timeHour)))
		started := mustStart(t, base(castCase{spell: hour, featureLevel: 4, counts: true}))
		cmd := character.CompleteCastingCommand{State: started.State(), Spell: hour, Context: castContext(t, slotSpent), FeatureCastLevel: 4, CountsAsSlotSpell: true}
		got, err := character.NewCompleteCasting().Do(context.Background(), cmd)
		if r, ok := got.(character.CompleteCastingRejected); err != nil || !ok || !slices.Equal(r.Rejection().Reasons(), kinds(character.CastRejectionSlotAlreadySpentThisTurn)) {
			t.Fatalf("complete = %#v, %v [FR-139]", got, err)
		}
		cmd.Context = castContext(t)
		got, err = character.NewCompleteCasting().Do(context.Background(), cmd)
		r, ok := got.(character.CompleteCastingCompleted)
		if at, _ := r.Report().CastAtLevel(); err != nil || !ok || at != 4 || !r.Report().CountsAsSlotSpell() {
			t.Errorf("complete = %#v, %v, want castAt 4 and CountsAsSlotSpell", got, err)
		}
	})
	t.Run("S14 events carry the character id", func(t *testing.T) {
		t.Parallel()
		other := warlockWizardFor(t, character.MustIdentifier("other"))
		a := mustCast(t, base(castCase{spell: castSpell(t, level6Spell, 6, concentrating())}))
		b := mustCast(t, castCase{state: other, class: character.CasterWarlock, mode: character.CastModeFeatureGranted, spell: castSpell(t, level6Spell, 6, concentrating())})
		for _, e := range a.Events() {
			if e.CharacterID() != caster {
				t.Errorf("%T id = %v, want %v [FR-141]", e, e.CharacterID(), caster)
			}
		}
		for _, e := range b.Events() {
			if e.CharacterID() != character.MustIdentifier("other") {
				t.Errorf("%T id = %v, want other", e, e.CharacterID())
			}
		}
	})
}

func warlockWizardFor(t *testing.T, id character.Identifier) character.SpellcastingState {
	t.Helper()
	res, err := character.NewCreateSpellcasting().Do(t.Context(), character.CreateSpellcastingCommand{Character: id, Levels: classLevelsOf(t, "warlock", 11), Spells: spellCatalogue(t)})
	if err != nil {
		t.Fatal(err)
	}
	return res.(character.CreateSpellcastingCreated).State()
}

// concentratingFeatureState is s after a Concentration spell was cast (feature-granted, level 6).
func concentratingFeatureState(t *testing.T, s character.SpellcastingState) character.SpellcastingState {
	t.Helper()
	return mustCast(t, castCase{state: s, class: character.CasterWarlock, mode: character.CastModeFeatureGranted, spell: castSpell(t, spellID(6, 2), 6, concentrating())}).State()
}

// castingStartedFor starts a one-hour feature-granted casting in s.
func castingStartedFor(t *testing.T, s character.SpellcastingState) character.SpellcastingState {
	t.Helper()
	return mustStart(t, castCase{state: s, class: character.CasterWarlock, mode: character.CastModeFeatureGranted, spell: castSpell(t, level6Spell, 6, withTimes(times(timeHour)))}).State()
}

// Requirement: doc/specs/casting-and-concentration.md FR-143, FR-146, FR-147, FR-150, Table U (U1 to U7, U10, U19, U20) -
// a scroll cast and a duplicated cast spend no slot, need no class, report the terms' figures, and skip the checks the
// spec lists.
// Path: per case.
func TestScrollAndDuplicatedCasts(t *testing.T) {
	t.Parallel()
	s := warlock11Wizard5(t)
	spent := s
	for _, l := range []int{1, 2, 3, 4, 5} {
		spent = expendAll(t, spent, character.PoolSpellcasting, l, spent.Slots(character.PoolSpellcasting).Remaining(slotLevel(l)))
	}
	rich := componentsOf(`{"verbal":true,"somatic":false,"material":{"description":"a diamond","costsGp":[300],"consumesMaterial":true}}`)
	l3 := func(opts ...spellOpt) content.Spell {
		return castSpell(t, preparedL3, 3, append([]spellOpt{withComponents(rich.json)}, opts...)...)
	}
	scroll := func(c castCase) castCase {
		c.state, c.noClass, c.mode = s, true, character.CastModeSpellScroll
		if c.terms == nil {
			c.terms = terms13(t, 0)
		}
		return c
	}
	dup := func(c castCase) castCase {
		c.state, c.noClass, c.mode = s, true, character.CastModeDuplicated
		if c.terms == nil {
			c.terms = terms13(t, 0)
		}
		return c
	}
	t.Run("U1 scroll with every slot spent, nothing known, no materials", func(t *testing.T) {
		t.Parallel()
		c := scroll(castCase{spell: l3()})
		c.state = spent
		r := mustCast(t, c)
		rep := r.Report()
		dc, hasDC := rep.SaveDC()
		bonus, hasBonus := rep.AttackBonus()
		at, _ := rep.CastAtLevel()
		sc, _ := rep.ScalingLevel()
		_, _, slot := rep.Slot()
		_, hasClass := rep.Class()
		if rep.Source() != character.CastSourceScroll || slot || hasClass || at != 3 || sc != 3 || !hasDC || dc.Value() != 13 || !hasBonus || bonus.Value() != 5 || rep.CountsAsSlotSpell() {
			t.Errorf("report = %+v [FR-143 FR-147]", rep)
		}
		if !reflect.DeepEqual(spent.Slots(character.PoolSpellcasting), r.State().Slots(character.PoolSpellcasting)) {
			t.Error("the slots changed")
		}
		if got := eventNames(r.Events()); !slices.Equal(got, []string{"SpellCast"}) {
			t.Errorf("events = %v", got)
		}
		if e := r.Events()[0].(character.SpellCast); e.Source() != character.CastSourceScroll {
			t.Errorf("event = %+v", e)
		}
	})
	t.Run("U2 scroll with a slot spent this turn", func(t *testing.T) {
		t.Parallel()
		r := mustCast(t, scroll(castCase{spell: l3(), mods: mods(slotSpent)}))
		if r.Report().CountsAsSlotSpell() {
			t.Error("CountsAsSlotSpell is true")
		}
	})
	scrollRej := []struct {
		name string
		c    castCase
		want []character.CastRejectionKind
	}{
		{"U3 cannot speak", scroll(castCase{spell: l3(), mods: mods(func(f *character.CastingFacts) { f.CanSpeak = false })}), kinds(character.CastRejectionMissingVerbal)},
		{"U4 cannot cast", scroll(castCase{spell: l3(), mods: mods(func(f *character.CastingFacts) { f.CanCastSpells = false })}), kinds(character.CastRejectionCannotCastSpells)},
		{"U4 untrained armor", scroll(castCase{spell: l3(), mods: mods(func(f *character.CastingFacts) { f.UntrainedArmorWorn = true })}), kinds(character.CastRejectionUntrainedArmor)},
		{"U4 trigger not met", scroll(castCase{spell: castSpell(t, preparedL3, 3, withTimes(times(timeReaction))), mods: mods(func(f *character.CastingFacts) { f.TriggerMet = false })}), kinds(character.CastRejectionTriggerNotMet)},
	}
	for _, c := range scrollRej {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.c.run(t)
			if r, ok := got.(character.CastSpellRejected); err != nil || !ok || !slices.Equal(r.Rejection().Reasons(), c.want) {
				t.Errorf("cast = %#v, %v, want %v [FR-143]", got, err, c.want)
			}
		})
	}
	t.Run("U4 casting in progress", func(t *testing.T) {
		t.Parallel()
		c := scroll(castCase{spell: l3()})
		c.state = castingStartedFor(t, s)
		got, err := c.run(t)
		if r, ok := got.(character.CastSpellRejected); err != nil || !ok || !slices.Equal(r.Rejection().Reasons(), kinds(character.CastRejectionCastingAlreadyInProgress)) {
			t.Errorf("cast = %#v, %v", got, err)
		}
	})
	t.Run("U5 scroll of a cantrip", func(t *testing.T) {
		t.Parallel()
		r := mustCast(t, scroll(castCase{spell: castSpell(t, spellID(0, 9), 0), terms: terms13(t, 5)}))
		_, hasAt := r.Report().CastAtLevel()
		if sc, _ := r.Report().ScalingLevel(); hasAt || sc != 5 {
			t.Errorf("report = %+v, want no castAt and scaling 5", r.Report())
		}
	})
	t.Run("U7 scroll Concentration spell while concentrating", func(t *testing.T) {
		t.Parallel()
		c := scroll(castCase{spell: castSpell(t, spellID(3, 4), 3, concentrating())})
		c.state = concentratingFeatureState(t, s)
		r := mustCast(t, c)
		if got, want := eventNames(r.Events()), []string{"ConcentrationEnded", "ConcentrationBegan", "SpellCast"}; !slices.Equal(got, want) {
			t.Errorf("events = %v, want %v", got, want)
		}
	})
	t.Run("U7 one-hour scroll spell, completed with every slot spent", func(t *testing.T) {
		t.Parallel()
		hour := castSpell(t, preparedL3, 3, withTimes(times(timeHour)))
		c := scroll(castCase{spell: hour})
		started := mustStart(t, c)
		cip, _ := started.State().CastingInProgress()
		if cip.Source() != character.CastSourceScroll || cip.Class() != 0 || cip.TotalMinutes() != 60 {
			t.Fatalf("CastingInProgress = %+v", cip)
		}
		cmd := character.CompleteCastingCommand{State: started.State(), Spell: hour, Context: castContext(t), Terms: terms13(t, 0)}
		got, err := character.NewCompleteCasting().Do(context.Background(), cmd)
		r, ok := got.(character.CompleteCastingCompleted)
		dc, _ := r.Report().SaveDC()
		if err != nil || !ok || r.Report().Source() != character.CastSourceScroll || dc.Value() != 13 {
			t.Fatalf("complete = %#v, %v", got, err)
		}
		cmd.Terms = nil
		if _, err := character.NewCompleteCasting().Do(context.Background(), cmd); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("completion of a scroll without terms: err = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("U10 another spell unavailable leaves the cast alone", func(t *testing.T) {
		t.Parallel()
		mustCast(t, castCase{spell: l1(t), slot: choice(1), mods: mods(unavailable(spellID(1, 7)))})
	})
	t.Run("U19 duplicated level 6 spell, every requirement unmet", func(t *testing.T) {
		t.Parallel()
		c := dup(castCase{spell: castSpell(t, level6Spell, 6, withTimes(times(timeReaction)), withComponents(rich.json)), mods: mods(func(f *character.CastingFacts) {
			*f = character.CastingFacts{UntrainedArmorWorn: true, SlotSpentThisTurn: true}
		})})
		r := mustCast(t, c)
		at, _ := r.Report().CastAtLevel()
		_, _, slot := r.Report().Slot()
		if r.Report().Source() != character.CastSourceDuplicated || at != 6 || slot || !slices.Equal(eventNames(r.Events()), []string{"SpellCast"}) {
			t.Errorf("report = %+v events %v [FR-150]", r.Report(), eventNames(r.Events()))
		}
		if r.Report().CountsAsSlotSpell() {
			t.Error("CountsAsSlotSpell is true")
		}
	})
	for _, k := range []struct {
		name string
		opts []spellOpt
		want []string
		left bool
	}{
		{"concentrating", []spellOpt{concentrating()}, []string{"ConcentrationEnded", "CastingFailed", "ConcentrationBegan", "SpellCast"}, false},
		{"plain", nil, []string{"SpellCast"}, true},
	} {
		t.Run("U20 duplicated cast while a casting is in progress, "+k.name+" spell", func(t *testing.T) {
			t.Parallel()
			c := dup(castCase{spell: castSpell(t, spellID(6, 6), 6, k.opts...)})
			c.state = castingStartedFor(t, s)
			r := mustCast(t, c)
			if got := eventNames(r.Events()); !slices.Equal(got, k.want) {
				t.Errorf("events = %v, want %v [FR-072 FR-083 FR-150]", got, k.want)
			}
			if _, going := r.State().CastingInProgress(); going != k.left {
				t.Errorf("casting in progress = %v, want %v [FR-072]", going, k.left)
			}
		})
	}
	t.Run("U20 duplicated Concentration spell while concentrating, a 10-minute spell", func(t *testing.T) {
		t.Parallel()
		c := dup(castCase{spell: castSpell(t, spellID(6, 5), 6, concentrating(), withTimes(times(timeTenMin)))})
		c.state = concentratingFeatureState(t, s)
		r := mustCast(t, c)
		if got, want := eventNames(r.Events()), []string{"ConcentrationEnded", "ConcentrationBegan", "SpellCast"}; !slices.Equal(got, want) {
			t.Errorf("events = %v, want %v", got, want)
		}
		if _, going := r.State().CastingInProgress(); going {
			t.Error("a casting began for a duplicated cast [FR-150]")
		}
	})
	t.Run("U20 duplicated cantrip uses the scaling level", func(t *testing.T) {
		t.Parallel()
		r := mustCast(t, dup(castCase{spell: castSpell(t, spellID(0, 8), 0), terms: terms13(t, 11)}))
		if sc, _ := r.Report().ScalingLevel(); sc != 11 {
			t.Errorf("scaling = %d, want 11", sc)
		}
	})
}

// componentsJSON is one components JSON.
type componentsJSON struct{ json string }

func componentsOf(json string) componentsJSON { return componentsJSON{json: json} }
