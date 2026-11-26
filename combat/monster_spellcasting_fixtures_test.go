package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

const monsterSpellcastingSpec = "doc/specs/monster-spellcasting.md "

// The monsters and spells of these tests are synthetic: made-up names carrying the numbers the spec's case
// tables quote from SRD 5.2.1, pp. 257 to 316, never a stat block or spell text.

// The spell references of mscMonster, as (group, spell).
var (
	mscLight          = [2]int{0, 0}
	mscInvisibility   = [2]int{0, 1}
	mscShapechange    = [2]int{0, 2}
	mscMindBlank      = [2]int{0, 3}
	mscGuardians      = [2]int{0, 4}
	mscPlantGrowth    = [2]int{0, 6}
	mscScrying        = [2]int{1, 0} // 1/Day Each
	mscGeas           = [2]int{1, 1}
	mscFireball       = [2]int{1, 2}
	mscScryingToo     = [2]int{2, 0} // 1/Day, a second entry for the same spell
	mscLightningBolt7 = [2]int{2, 1}
	mscConeOfCold9    = [2]int{2, 2}
	mscFireball3      = [2]int{3, 0}
	mscFireball2      = [2]int{3, 1}
	mscLightOne       = [2]int{3, 2}
)

func mscRef(tb testing.TB, r [2]int) combat.SpellUseRef {
	tb.Helper()
	return luMust(combat.NewSpellUseRef(r[0], r[1]))
}

func mscEntry(name, note string, castAt int) content.MonsterSpellEntry {
	return luMust(content.NewMonsterSpellEntry(name, note, castAt))
}

func mscGroup(f content.SpellFrequency, entries ...content.MonsterSpellEntry) content.MonsterSpellGroup {
	return luMust(content.NewMonsterSpellGroup(f, entries))
}

func mscTraitPart(name string, f content.MonsterSpellcastingFields) content.StatBlockPart {
	sc := luMust(content.NewMonsterSpellcasting(f))
	return luMust(content.NewStatBlockPart(name, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Spellcasting: sc}))
}

func mscScores(tb testing.TB, f *content.MonsterFields, scores map[ability.Ability]int) {
	tb.Helper()
	all, saves := map[ability.Ability]ability.Score{}, map[ability.Ability]int{}
	for _, a := range ability.All() {
		all[a], saves[a] = ability.MustScore(10), 0
		if v, ok := scores[a]; ok {
			all[a] = ability.MustScore(v)
			saves[a] = ability.MustScore(v).Modifier().Value()
		}
	}
	f.Abilities = luMust(content.NewAbilityBlock(luMust(ability.NewScores(all)), saves))
}

// mscTraitFields is the trait of mscMonster: Intelligence, DC 14, bonus +6, and the groups listed above.
func mscTraitFields() content.MonsterSpellcastingFields {
	return content.MonsterSpellcastingFields{
		Ability: ability.Intelligence, SaveDC: 14, HasSaveDC: true, AttackBonus: 6, HasAttackBonus: true,
		Groups: []content.MonsterSpellGroup{
			mscGroup(content.SpellFrequencyAtWill{}, mscEntry("light", "", 0), mscEntry("invisibility", "self only, and the hag leaves no tracks while Invisible", 0),
				mscEntry("shapechange", "Beast or Humanoid form only, and no Concentration or Temporary Hit Points required", 0), mscEntry("mind_blank", "included in AC", 0),
				mscEntry("spirit_guardians", "", 0), mscEntry("fire_bolt", "", 0), mscEntry("plant_growth", "", 0)),
			mscGroup(luPerDayEach(1), mscEntry("scrying", "", 0), mscEntry("geas", "", 0), mscEntry("fireball", "", 0)),
			mscGroup(luPerDaySpells(1), mscEntry("scrying", "", 0), mscEntry("lightning_bolt", "", 7), mscEntry("cone_of_cold", "", 9)),
			mscGroup(content.SpellFrequencyAtWill{}, mscEntry("fireball", "", 3), mscEntry("fireball", "", 2), mscEntry("light", "", 1)),
		},
	}
}

// mscMonster has the trait of f (mscTraitFields when f is nil), Intelligence 16, Wisdom 12 and Charisma 20, a Bonus Action "Misty Step"
// 3/Day (BonusActions 0) and an unlimited Action "Phantasms" (Actions 0).
func mscMonster(tb testing.TB, edit func(*content.MonsterSpellcastingFields)) content.Monster {
	tb.Helper()
	f := mscTraitFields()
	if edit != nil {
		edit(&f)
	}
	return luMonster(tb, func(m *content.MonsterFields) {
		mscScores(tb, m, map[ability.Ability]int{ability.Intelligence: 16, ability.Wisdom: 12, ability.Charisma: 20})
		m.Traits = []content.StatBlockPart{mscTraitPart("Spellcasting", f)}
		m.Actions = []content.StatBlockPart{luPart("Phantasms", content.PartUnlimited{})}
		m.BonusActions = []content.StatBlockPart{luPart("Misty Step", luPerDay(3, 0))}
	})
}

var (
	mscMistyStep = [2]int{int(combat.MonsterPartBonusActions), 0}
	mscPhantasms = [2]int{int(combat.MonsterPartActions), 0}
)

func mscPart(tb testing.TB, p [2]int) combat.LimitedUsePartRef {
	tb.Helper()
	return luRef(tb, combat.MonsterPartSection(p[0]), p[1])
}

func mscID() combat.CombatantID { return combat.MustCombatantID(luSelf) }

// mscRig is a caster with its two states and a context in which the monster can speak, gesture and act.
type mscRig struct {
	caster combat.MonsterSpellcaster
	state  combat.MonsterSpellcastingState
	uses   combat.LimitedUseState
	mc     combat.MonsterCastContext
}

func mscNewRig(tb testing.TB, m content.Monster) mscRig {
	tb.Helper()
	return mscRig{
		caster: luMust(combat.NewMonsterSpellcaster(m, mscID())),
		state:  luMust(combat.NewMonsterSpellcastingState(mscID())),
		uses:   luStart(tb, m, combat.OutsideLair),
		mc:     mscContext(true, true, combat.ConditionActionAvailabilityAvailable),
	}
}

func mscContext(speak, gesture bool, a combat.ConditionActionAvailability) combat.MonsterCastContext {
	return luMust(combat.NewMonsterCastContext(speak, gesture, a))
}

// mscResult is what Cast and CompleteCasting return, as one value so the helpers can take it whole.
type mscResult struct {
	Out combat.MonsterCastOutcome
	Err error
}

func (r mscRig) cast(req combat.MonsterCastRequest, spell content.Spell, targets ...combat.CombatantID) mscResult {
	out, err := combat.MonsterSpellcastingRules{}.Cast(r.caster, r.state, r.uses, req, spell, targets, r.mc)
	return mscResult{out, err}
}

func (r mscRig) complete(req combat.MonsterCastRequest, spell content.Spell, targets ...combat.CombatantID) mscResult {
	out, err := combat.MonsterSpellcastingRules{}.CompleteCasting(r.caster, r.state, r.uses, req, spell, targets, r.mc)
	return mscResult{out, err}
}

// list is a ListCast of the entry at ref with no casting time mode.
func mscList(tb testing.TB, ref [2]int) combat.MonsterListCast {
	tb.Helper()
	return combat.NewMonsterListCast(mscRef(tb, ref), content.CastingTimeMode{})
}

func mscPartCast(tb testing.TB, p [2]int, castAt int, own combat.PartSpellcasting) combat.MonsterPartCast {
	tb.Helper()
	return luMust(combat.NewMonsterPartCast(mscPart(tb, p), castAt, own))
}

// then applies a performed or started outcome to the rig.
func (r mscRig) then(tb testing.TB, out combat.MonsterCastOutcome) mscRig {
	tb.Helper()
	switch v := out.(type) {
	case combat.MonsterCastPerformed:
		r.state, r.uses = v.Transition.State(), v.Transition.LimitedUse()
	case combat.MonsterCastStarted:
		r.state, r.uses = v.Transition.State(), v.Transition.LimitedUse()
	default:
		tb.Fatalf("outcome %#v is not a cast that changed the states", out)
	}
	return r
}

// mscPerformed returns the performed outcome of a cast that must have been performed.
func mscPerformed(tb testing.TB, res mscResult) combat.MonsterCastPerformed {
	tb.Helper()
	out := res.Out
	if res.Err != nil {
		tb.Fatalf("cast returned error %v", res.Err)
	}
	p, ok := out.(combat.MonsterCastPerformed)
	if !ok {
		tb.Fatalf("outcome = %#v, want a performed cast", out)
	}
	return p
}

func mscStarted(tb testing.TB, res mscResult) combat.MonsterCastStarted {
	tb.Helper()
	out := res.Out
	if res.Err != nil {
		tb.Fatalf("cast returned error %v", res.Err)
	}
	s, ok := out.(combat.MonsterCastStarted)
	if !ok {
		tb.Fatalf("outcome = %#v, want a started casting", out)
	}
	return s
}

func mscRefused(tb testing.TB, res mscResult) combat.MonsterCastRefusal {
	tb.Helper()
	out := res.Out
	if res.Err != nil {
		tb.Fatalf("cast returned error %v", res.Err)
	}
	r, ok := out.(combat.MonsterCastRefused)
	if !ok {
		tb.Fatalf("outcome = %#v, want a refusal", out)
	}
	return r.Reason
}

func mscKinds(events []combat.MonsterSpellcastingEvent) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// mscConcentrating returns a state of the monster that concentrates on spell as an effect.
func mscConcentrating(tb testing.TB, spell string) combat.MonsterSpellcastingState {
	tb.Helper()
	c := luMust(character.NewConcentration(character.NewConcentrationSubjectSpell(content.SpellID(spell)), character.ConcentrationEffect))
	return luMust(combat.NewMonsterSpellcastingState(mscID(), combat.WithMonsterConcentration(c)))
}

// The spells. Ids are names, so a list entry named like a spell matches it.

func mscSpellOf(id string, level int, edit ...func(*content.SpellFields)) content.Spell {
	return seSpell(id, level, content.TextOnlyEffects{}, func(f *content.SpellFields) {
		for _, e := range edit {
			e(f)
		}
	})
}

func mscConcentration(f *content.SpellFields) {
	f.Duration = luMust(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitConcentrationUpTo))
}

func mscCastingMinutes(n int) func(*content.SpellFields) {
	return func(f *content.SpellFields) {
		f.CastingTimes = luMust(content.NewSpellCastingTimesSingle(luMust(content.NewCastingTimeMinutes(n))))
	}
}

func mscCastingHours(n int) func(*content.SpellFields) {
	return func(f *content.SpellFields) {
		f.CastingTimes = luMust(content.NewSpellCastingTimesSingle(luMust(content.NewCastingTimeHours(n))))
	}
}

func mscComponents(v, s bool) func(*content.SpellFields) {
	return func(f *content.SpellFields) { f.Components = content.NewSpellComponents(v, s) }
}

func mscSpells() map[string]content.Spell {
	out := map[string]content.Spell{
		"light":             mscSpellOf("light", 0),
		"invisibility":      mscSpellOf("invisibility", 2),
		"shapechange":       mscSpellOf("shapechange", 9, mscConcentration),
		"mind_blank":        mscSpellOf("mind_blank", 8),
		"spirit_guardians":  mscSpellOf("spirit_guardians", 3, mscConcentration),
		"fire_bolt":         seFireBolt(),
		"fireball":          seFireball(),
		"scrying":           mscSpellOf("scrying", 5, mscConcentration, mscCastingMinutes(10)),
		"geas":              mscSpellOf("geas", 5, mscCastingMinutes(1)),
		"lightning_bolt":    mscSpellOf("lightning_bolt", 3),
		"cone_of_cold":      mscSpellOf("cone_of_cold", 5),
		"misty_step":        mscSpellOf("misty_step", 2),
		"mirror_image":      mscSpellOf("mirror_image", 2),
		"long_ritual":       mscSpellOf("long_ritual", 4, mscCastingHours(8)),
		"plant_growth":      mscPlantGrowthSpell(),
		"silent_gesture":    mscSpellOf("silent_gesture", 1, mscComponents(false, true)),
		"spoken_word":       mscSpellOf("spoken_word", 1, mscComponents(true, false)),
		"costly_material":   mscCostlyMaterial(),
		"text_only_save":    mscSpellOf("text_only_save", 1),
		"concentration_one": mscSpellOf("concentration_one", 1, mscConcentration),
	}
	return out
}

// mscPlantGrowth has two casting time modes: a Magic action and 8 hours.
func mscPlantGrowthSpell() content.Spell {
	modes := []content.CastingTimeMode{
		luMust(content.NewCastingTimeMode("Action", content.NewCastingTimeMagicAction())),
		luMust(content.NewCastingTimeMode("Long", luMust(content.NewCastingTimeHours(8)))),
	}
	return mscSpellOf("plant_growth", 3, func(f *content.SpellFields) {
		f.CastingTimes = luMust(content.NewSpellCastingTimesModes(modes))
	})
}

func mscCostlyMaterial() content.Spell {
	material := luMust(content.NewMaterialComponent("a pearl worth 500+ GP", []int{500}, false))
	return mscSpellOf("costly_material", 4, func(f *content.SpellFields) {
		f.Components = luMust(content.NewSpellComponents(true, true).WithMaterial(material))
	})
}

// mscWrap turns the two results of a direct call of the rules into an mscResult.
func mscWrap(out combat.MonsterCastOutcome, err error) mscResult { return mscResult{out, err} }
