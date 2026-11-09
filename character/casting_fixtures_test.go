package character_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// spellOpts describes a synthetic spell for the casting tests: its casting
// times, components and duration are raw JSON of the content schema.
type spellOpts struct {
	times, components, duration string
	ritual                      bool
}

type spellOpt func(*spellOpts)

func withTimes(json string) spellOpt      { return func(o *spellOpts) { o.times = json } }
func withComponents(json string) spellOpt { return func(o *spellOpts) { o.components = json } }
func asRitual() spellOpt                  { return func(o *spellOpts) { o.ritual = true } }
func concentrating() spellOpt {
	return func(o *spellOpts) {
		o.duration = `{"kind":"timed","amount":1,"unit":"minutes","limit":"concentrationUpTo"}`
	}
}

const (
	timeAction   = `{"kind":"magicAction"}`
	timeBonus    = `{"kind":"bonusAction"}`
	timeReaction = `{"kind":"reaction","trigger":"which you take when hit"}`
	timeMinute   = `{"kind":"minutes","amount":1}`
	timeTenMin   = `{"kind":"minutes","amount":10}`
	timeHour     = `{"kind":"hours","amount":1}`
	timeEight    = `{"kind":"hours","amount":8}`
)

func verbalOnly() string { return `{"verbal":true,"somatic":false}` }
func materialOnly(costs string, consumes bool) string {
	return fmt.Sprintf(`{"verbal":false,"somatic":false,"material":{"description":"a pinch of dust","costsGp":%s,"consumesMaterial":%t}}`, costs, consumes)
}

// castSpell decodes a synthetic spell with id and level; by default a Verbal
// spell with a Magic action casting time and an instantaneous duration.
func castSpell(tb testing.TB, id content.SpellID, level int, opts ...spellOpt) content.Spell {
	tb.Helper()
	o := spellOpts{times: "[" + timeAction + "]", components: verbalOnly(), duration: `{"kind":"instantaneous"}`}
	for _, f := range opts {
		f(&o)
	}
	doc := fmt.Sprintf(`{"schemaVersion":1,"spells":[{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":["wizard"],"castingTimes":%s,"ritual":%t,"range":{"kind":"self"},"components":%s,"duration":%s,"description":[{"kind":"paragraph","text":"Synthetic."}]}]}`,
		id, level, o.times, o.ritual, o.components, o.duration)
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		tb.Fatalf("fixture spell: %v\n%s", err, strings.TrimSpace(doc))
	}
	return set.Entries()[0].(content.Spell)
}

// times builds a casting time list from raw JSON objects.
func times(objs ...string) string { return "[" + strings.Join(objs, ",") + "]" }

// labelled adds a label to a raw casting time object.
func labelled(label, obj string) string {
	return strings.TrimSuffix(obj, "}") + fmt.Sprintf(`,"label":%q}`, label)
}

func permissive() character.CastingFacts {
	return character.CastingFacts{CanCastSpells: true, CanSpeak: true, CanMakeGestures: true, HasFreeHand: true, TriggerMet: true, CanReadSpellbook: true}
}

// castContext builds a CastingContext from the permissive facts changed by mods.
func castContext(tb testing.TB, mods ...func(*character.CastingFacts)) character.CastingContext {
	tb.Helper()
	f := permissive()
	for _, m := range mods {
		m(&f)
	}
	c, err := character.NewCastingContext(f)
	if err != nil {
		tb.Fatal(err)
	}
	return c
}

// ids used by the casting states.
var (
	cantripKnown    = spellID(0, 1)
	preparedL1      = spellID(1, 1)
	preparedL3      = spellID(3, 1)
	ritualInBook    = spellID(1, 9) // in the Wizard's spellbook, not prepared
	ritualPrepared  = spellID(1, 2) // prepared
	unpreparedSpell = spellID(1, 11)
	level6Spell     = spellID(6, 1)
)

// wizard5 is a Wizard 5 with three cantrips, nine prepared spells and a spellbook.
func wizard5(t *testing.T) character.SpellcastingState {
	t.Helper()
	prepared := append(append(spellIDs(1, 4), spellID(2, 1), spellID(2, 2)), spellID(3, 1), spellID(3, 2), spellID(3, 3))
	return castingState(t, classLevelsOf(t, "wizard", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterWizard, Cantrips: spellIDs(0, 3),
		Prepared: prepared, Spellbook: character.NewSpellbook(append(cloneIDs(prepared), ritualInBook))})
}

func cloneIDs(in []content.SpellID) []content.SpellID {
	return append([]content.SpellID(nil), in...)
}

// cleric5 is a Cleric 5 with three cantrips and the level 1 spells s1_01 to s1_04 prepared.
func cleric5(t *testing.T) character.SpellcastingState {
	t.Helper()
	return castingState(t, classLevelsOf(t, "cleric", 5), spellCatalogue(t), character.ClassSpellcastingSetup{Class: character.CasterCleric, Cantrips: spellIDs(0, 3), Prepared: spellIDs(1, 4)})
}

// warlockWizard builds a Warlock w / Wizard 5 state with empty lists.
func warlockWizard(t *testing.T, warlock int) character.SpellcastingState {
	t.Helper()
	return castingState(t, classLevelsOf(t, "warlock", warlock, "wizard", 5), spellCatalogue(t))
}

func choice(n int) character.SpellSlotChoice {
	return character.NewSpellSlotChoiceSpellcasting(slotLevel(n))
}
