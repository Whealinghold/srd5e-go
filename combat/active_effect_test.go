package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func bless(id, caster, target string, potency int) aeSpec {
	return aeSpec{id: id, spell: "bless", caster: caster, concentration: true, subject: aeCreature2(target), template: aeSimpleNoT(10, potency, combat.CombiningOnTarget)}
}

func aeCreature2(id string) combat.EffectSubject {
	return aeMust(combat.NewCreatureEffectSubject(combat.MustCombatantID(id)))
}

func aeSimpleNoT(rounds, potency int, subject combat.EffectCombiningSubject) combat.EffectTemplate {
	stage := aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeMust(combat.NewEffectLastsRounds(rounds))}))
	return aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Stages: []combat.EffectStage{stage}, Potency: combat.EffectPotency(potency), CombiningSubject: subject}))
}

func mark(id, caster, target string) aeSpec {
	return aeSpec{id: id, spell: "hunters_mark", caster: caster, concentration: true, subject: aeCreature2(target),
		template: aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{
			Stages:           []combat.EffectStage{aeMust(combat.NewEffectStage(combat.EffectStageParams{Duration: aeMust(combat.NewEffectLastsRounds(600))}))},
			CombiningSubject: combat.CombiningOnCaster, SurvivesSubjectDeath: true}))}
}

func applyingIDs(reg combat.ActiveEffects, subject combat.EffectSubject) []string {
	out := []string{}
	for _, a := range (combat.EffectCombination{}).Applying(reg, subject) {
		out = append(out, string(a.Effect().ID()))
	}
	return out
}

// otherSpec starts an effect of an origin that is not a spell.
func startOther(t *testing.T, reg combat.ActiveEffects, id, key, target string) combat.ActiveEffects {
	t.Helper()
	cmd := combat.StartActiveEffectCommand{Effects: reg, ID: aeEffectID(id), Origin: aeMust(combat.NewOtherEffectOrigin(key)), Caster: "item",
		Subject: aeCreature2(target), Template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)}
	return aeStartCmd(t, cmd)
}

func aeStartCmd(t *testing.T, cmd combat.StartActiveEffectCommand) combat.ActiveEffects {
	t.Helper()
	res, err := combat.NewStartActiveEffect().Run(t.Context(), cmd)
	started, ok := res.(combat.StartActiveEffectStarted)
	if err != nil || !ok {
		t.Fatalf("start = %#v, %v", res, err)
	}
	return started.Effects()
}

// Requirement: doc/specs/active-effects.md FR-015 to FR-019, table C - Combining Spell Effects: castings of one spell on one combining subject do not combine and the most potent (then the latest) applies; different spells, other origins and different subjects all apply; a suppressed effect stays and applies again when the applying one ends.
// Path: per case.
func TestCombiningSpellEffects(t *testing.T) {
	t.Parallel()
	target := aeCreature2("t")
	end := func(reg combat.ActiveEffects, ids ...string) combat.ActiveEffects {
		var effectIDs []combat.ActiveEffectID
		for _, id := range ids {
			effectIDs = append(effectIDs, aeEffectID(id))
		}
		res, err := combat.NewEndActiveEffects().Run(t.Context(), combat.EndActiveEffectsCommand{Effects: reg, Cause: combat.NewRemovedByEffect(effectIDs...)})
		ended, ok := res.(combat.EndActiveEffectsEnded)
		if err != nil || !ok {
			t.Fatalf("end = %#v, %v", res, err)
		}
		return ended.Effects()
	}
	c2 := aeSeq(t, bless("a", "cleric_a", "t", 0), bless("b", "cleric_b", "t", 0))
	bane := aeSpec{id: "bane", spell: "bane", caster: "cleric_b", subject: target, template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)}
	movedMarks := func() combat.ActiveEffects {
		reg := aeSeq(t, mark("ma", "ranger_a", "x"), mark("mb", "ranger_b", "y"))
		cause := aeMust(combat.NewMoveEffectSubject(aeEffectID("ma"), aeCreature2("y")))
		res, err := combat.NewEndActiveEffects().Run(t.Context(), combat.EndActiveEffectsCommand{Effects: reg, Cause: cause})
		moved, ok := res.(combat.EndActiveEffectsMoved)
		if err != nil || !ok {
			t.Fatalf("move = %#v, %v", res, err)
		}
		return moved.Effects()
	}
	otherPair := startOther(t, startOther(t, combat.ActiveEffects{}, "o1", "ring", "t"), "o2", "ring", "t")
	cases := []struct {
		name, path string
		reg        combat.ActiveEffects
		subject    combat.EffectSubject
		want       []string
	}{
		{"C1 one Bless", "happy", aeSeq(t, bless("a", "cleric_a", "t", 0)), target, []string{"a"}},
		{"C2 two Blesses, equal potency, the latest applies", "unhappy (rule interaction)", c2, target, []string{"b"}},
		{"C3 the latest ends, the earlier applies again", "unhappy (rule interaction)", end(c2, "b"), target, []string{"a"}},
		{"C4 Bless and Bane both apply", "happy", aeSeq(t, bless("a", "cleric_a", "t", 0), bane), target, []string{"a", "bane"}},
		{"C5 the higher potency applies though earlier", "unhappy (boundary)", aeSeq(t, bless("a", "cleric_a", "t", 2), bless("b", "cleric_b", "t", 1)), target, []string{"a"}},
		{"C6 two other origins with one key both apply", "unhappy (rule interaction)", otherPair, target, []string{"o1", "o2"}},
		{"C7 one caster, two targets, both apply", "happy", aeSeq(t, bless("a", "cleric_a", "t", 0), bless("b", "cleric_a", "u", 0)), aeCreature2("u"), []string{"b"}},
		{"C8 Hunter's Mark by two Rangers both apply", "unhappy (rule interaction)", aeSeq(t, mark("ma", "ranger_a", "t"), mark("mb", "ranger_b", "t")), target, []string{"ma", "mb"}},
		{"C9 a moved mark and another Ranger's mark both apply", "unhappy (rule interaction)", movedMarks(), aeCreature2("y"), []string{"ma", "mb"}},
		{"a place subject has none of a creature's effects", "unhappy (empty and minimal)", c2, aePlace(t, "p"), []string{}},
		{"an empty registry", "unhappy (empty and minimal)", combat.ActiveEffects{}, target, []string{}},
	}
	for _, c := range cases {
		if got := applyingIDs(c.reg, c.subject); !slices.Equal(got, c.want) {
			t.Errorf("%s: applying = %v, want %v [FR-015]", c.name, got, c.want)
		}
	}
	if c2.Len() != 2 {
		t.Errorf("the suppressed Bless left the registry: %v [FR-017]", aeIDs(c2))
	}
}

// Requirement: doc/specs/active-effects.md FR-015 - one caster casting one spell at a creature twice cannot overlap (the second Concentration spell ends the first), so for a spell that combines per caster two marks by one caster on two creatures give one applying effect, the latest.
// Path: unhappy (rule interaction).
func TestCombiningPerCasterAcrossSubjects(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, mark("m1", "ranger", "x"), mark("m2", "ranger", "y"))
	if got := applyingIDs(reg, aeCreature2("x")); len(got) != 0 {
		t.Errorf("the earlier mark applies on x: %v", got)
	}
	if got := applyingIDs(reg, aeCreature2("y")); !slices.Equal(got, []string{"m2"}) {
		t.Errorf("applying on y = %v, want [m2]", got)
	}
}

// Requirement: doc/specs/active-effects.md FR-064 - Applying returns only applying effects, each with its distinct, increasing position in registry order, and changes nothing.
// Path: happy.
func TestApplyingEffectPositions(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, bless("a", "cleric_a", "t", 0), bless("b", "cleric_b", "t", 0), bane2("bane", "t"))
	before := aeIDs(reg)
	got := (combat.EffectCombination{}).Applying(reg, aeCreature2("t"))
	if len(got) != 2 || got[0].Position() != 1 || got[1].Position() != 2 || got[0].Effect().ID() != "b" {
		t.Fatalf("applying = %v, want b at 1 and bane at 2 [FR-064]", got)
	}
	if !slices.Equal(aeIDs(reg), before) {
		t.Error("Applying changed the registry")
	}
	if p, ok := reg.Position("bane"); !ok || p != 2 {
		t.Errorf("Position(bane) = %d, %v", p, ok)
	}
	if _, ok := reg.Position("nobody"); ok {
		t.Error("Position finds an unknown id")
	}
}

func bane2(id, target string) aeSpec {
	return aeSpec{id: id, spell: "bane", caster: "cleric_c", subject: aeCreature2(target), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget)}
}

// Requirement: doc/specs/active-effects.md FR-002, FR-064 - an ActiveEffect exposes its id, origin, subject, spell, level, potency, binding, stage and clocks, and answers who cast it and what it is on.
// Path: happy.
func TestActiveEffectAccessors(t *testing.T) {
	t.Parallel()
	reg := aeSeq(t, aeSpec{id: "hp", spell: "hold_person", caster: "wiz", concentration: true, subject: aeCreature2("orc"), template: aeHoldPerson(t, 14),
		conditions: []content.ConditionKind{content.ConditionParalyzed}, active: "wiz"})
	e := aeEffect(t, reg, "hp")
	spell, isSpell := e.SpellID()
	level, hasLevel := e.CastAtLevel()
	b, bound := e.Binding()
	left, countsRounds := e.OverallRemainingRounds()
	if e.IsZero() || e.ID() != "hp" || spell != "hold_person" || !isSpell || level != 1 || !hasLevel || !bound || !b.CasterIs("wiz") ||
		!e.CasterIs("wiz") || e.CasterIs("orc") || !e.OnCreature("orc") || e.OnCreature("wiz") || e.Potency() != 0 || e.Stage() != 0 ||
		e.RemainingRounds() != 10 || e.Duration() != aeRounds(t, 10) || countsRounds || left != 0 || !e.Anchored() || !e.AnchoredTo("wiz") || e.AnchoredTo("orc") ||
		e.SkipsCurrentTurnEnd() || !e.Subject().(combat.CreatureEffectSubject).Is("orc") || e.Template().IsZero() {
		t.Errorf("effect = %#v", e)
	}
	if !slices.Equal(e.Conditions(), []content.ConditionKind{content.ConditionParalyzed}) {
		t.Errorf("conditions = %v", e.Conditions())
	}
	e.Conditions()[0] = content.ConditionBlinded
	if aeEffect(t, reg, "hp").Conditions()[0] != content.ConditionParalyzed {
		t.Error("a caller's slice changed the effect")
	}
	if _, has := e.StoredDamage(); has {
		t.Error("hold person stores damage")
	}
	if !e.Equal(aeEffect(t, reg, "hp")) || e.Equal(combat.ActiveEffect{}) || !(combat.ActiveEffect{}).IsZero() {
		t.Error("Equal or IsZero")
	}
	if o, _ := e.Origin().(combat.SpellEffectOrigin); o.SpellID() != "hold_person" {
		t.Error("origin")
	}
	var anchorless combat.ActiveEffect
	if anchorless.Anchored() || anchorless.AnchoredTo("") {
		t.Error("the zero effect is anchored")
	}
	other := startOther(t, combat.ActiveEffects{}, "o", "ring", "t")
	if _, isSpell := aeEffect(t, other, "o").SpellID(); isSpell {
		t.Error("an other-origin effect has a spell")
	}
	if _, has := aeEffect(t, other, "o").CastAtLevel(); has {
		t.Error("an other-origin effect has a cast level")
	}
	if _, bound := aeEffect(t, other, "o").Binding(); bound {
		t.Error("an other-origin effect is bound")
	}
	blast := aeSeq(t, aeSpec{id: "bf", spell: "delayed_blast_fireball", caster: "wiz", concentration: true, subject: aePlace(t, "bead"), template: aeDelayedBlast(t)})
	if s, has := aeEffect(t, blast, "bf").StoredDamage(); !has || s != dice.MustParseExpression("12d6") {
		t.Errorf("stored = %v, %v", s, has)
	}
	if o, has := aeEffect(t, aeSeq(t, aeSpec{id: "sl", spell: "sleep", caster: "wiz", subject: aeCreature2("t"), template: aeSleep(t, "t", 12)}), "sl").OverallRemainingRounds(); !has || o != 10 {
		t.Errorf("overall remaining = %d, %v, want 10 true", o, has)
	}
}

// Requirement: doc/specs/active-effects.md FR-007, FR-008 - a registry keeps the order started, accepts an empty list, refuses two effects with one id and a zero effect, copies its input and its output, and answers the four queries in registry order without changing.
// Path: unhappy (empty and minimal, immutability).
func TestActiveEffectsRegistry(t *testing.T) {
	t.Parallel()
	var empty combat.ActiveEffects
	if reg, err := combat.NewActiveEffects(); err != nil || reg.Len() != 0 || !reg.Equal(empty) {
		t.Errorf("NewActiveEffects() = %v, %v", reg, err)
	}
	reg := aeSeq(t, bless("a", "c1", "t", 0), mark("m", "r", "t"), bless("b", "c1", "u", 0))
	effects := reg.Effects()
	if _, err := combat.NewActiveEffects(effects[0], effects[0]); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a duplicate id err = %v", err)
	}
	if _, err := combat.NewActiveEffects(combat.ActiveEffect{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a zero effect err = %v", err)
	}
	again, err := combat.NewActiveEffects(effects...)
	if err != nil || !again.Equal(reg) {
		t.Fatalf("NewActiveEffects = %v, %v", again, err)
	}
	effects[0] = combat.ActiveEffect{}
	if !again.Equal(reg) || reg.Effects()[0].IsZero() {
		t.Error("a caller's slice changed a registry")
	}
	ids := func(es []combat.ActiveEffect) []string {
		out := []string{}
		for _, e := range es {
			out = append(out, string(e.ID()))
		}
		return out
	}
	place := aeSeq(t, aeSpec{id: "sp", spell: "flaming_sphere", caster: "c1", subject: aePlace(t, "sphere"), template: aeFlamingSphere(t, 13)})
	cases := []struct {
		name string
		got  []combat.ActiveEffect
		want []string
	}{
		{"for creature t", reg.ForCreature("t"), []string{"a", "m"}},
		{"for creature nobody", reg.ForCreature("nobody"), []string{}},
		{"for place", place.ForPlace("sphere"), []string{"sp"}},
		{"for another place", place.ForPlace("other"), []string{}},
		{"a creature's effects are not on a place", reg.ForPlace("t"), []string{}},
		{"cast by c1", reg.CastBy("c1"), []string{"a", "b"}},
		{"bound to c1's bless", reg.BoundTo(aeBinding(t, "c1", "bless")), []string{"a", "b"}},
		{"bound to c1's other spell", reg.BoundTo(aeBinding(t, "c1", "bane")), []string{}},
		{"the zero binding matches nothing", reg.BoundTo(combat.ConcentrationBinding{}), []string{}},
	}
	for _, c := range cases {
		if got := ids(c.got); !slices.Equal(got, c.want) {
			t.Errorf("%s = %v, want %v [FR-008]", c.name, got, c.want)
		}
	}
	if e, ok := reg.Find("m"); !ok || e.ID() != "m" {
		t.Error("Find(m)")
	}
	if _, ok := reg.Find("zzz"); ok {
		t.Error("Find finds an unknown id")
	}
	if !slices.Equal(aeIDs(reg), []string{"a", "m", "b"}) {
		t.Errorf("order = %v, want the order started", aeIDs(reg))
	}
	if reg.Equal(empty) || empty.Equal(reg) {
		t.Error("a registry equals the empty one")
	}
}

// Requirement: doc/specs/active-effects.md FR-002 - an ActiveEffect is equal by value over all its fields: two effects that differ in one field only are unequal, whichever field it is.
// Path: per case.
func TestActiveEffectEqualComparesEveryField(t *testing.T) {
	t.Parallel()
	base := aeSpec{id: "e", spell: "bless", caster: "c", subject: aeCreature2("t"), template: aeSimpleNoT(10, 0, combat.CombiningOnTarget), active: "a"}
	variant := func(edit func(*aeSpec)) combat.ActiveEffect {
		s := base
		edit(&s)
		return aeEffect(t, aeSeq(t, s), s.id)
	}
	want := variant(func(*aeSpec) {})
	cases := map[string]combat.ActiveEffect{
		"id":         variant(func(s *aeSpec) { s.id = "f" }),
		"origin":     variant(func(s *aeSpec) { s.spell = "bane" }),
		"caster":     variant(func(s *aeSpec) { s.caster = "d" }),
		"subject":    variant(func(s *aeSpec) { s.subject = aeCreature2("u") }),
		"binding":    variant(func(s *aeSpec) { s.concentration = true }),
		"template":   variant(func(s *aeSpec) { s.template = aeSimpleNoT(10, 1, combat.CombiningOnTarget) }),
		"anchor":     variant(func(s *aeSpec) { s.active = "b" }),
		"conditions": variant(func(s *aeSpec) { s.conditions = []content.ConditionKind{content.ConditionPoisoned} }),
	}
	if !want.Equal(variant(func(*aeSpec) {})) {
		t.Fatal("two equal effects are unequal")
	}
	for name, other := range cases {
		if want.Equal(other) || other.Equal(want) {
			t.Errorf("effects that differ in %s are equal [FR-002]", name)
		}
	}
	reg := aeSeq(t, base)
	ticked := aeAdvance(t, reg, aeStartTick(t, "a"), "a").Effects()
	if aeEffect(t, ticked, "e").Equal(aeEffect(t, reg, "e")) {
		t.Error("effects that differ in the stage clock are equal")
	}
	until := aeSeq(t, untilSpec("u", "a", content.BoundaryEnd, "a"))
	outside := aeSeq(t, untilSpec("u", "a", content.BoundaryEnd, ""))
	if aeEffect(t, until, "u").Equal(aeEffect(t, outside, "u")) {
		t.Error("effects that differ in the skipped turn end are equal")
	}
	overall := aeMust(combat.NewEffectTemplate(combat.EffectTemplateParams{Overall: aeRounds(t, 5),
		Stages: []combat.EffectStage{aeStage(t, combat.EffectStageParams{Duration: combat.EffectLastsIndefinitely{}})}}))
	long := aeSeq(t, aeSpec{id: "o", spell: "bless", caster: "c", subject: aeCreature2("t"), template: overall})
	if aeEffect(t, aeAdvance(t, long, aeElapsed(t, 1), "").Effects(), "o").Equal(aeEffect(t, long, "o")) {
		t.Error("effects that differ in the overall clock are equal")
	}
	blast := aeSeq(t, aeSpec{id: "b", spell: "bless", caster: "c", subject: aePlace(t, "p"), template: aeDelayedBlast(t), active: "c"})
	if aeEffect(t, aeAdvance(t, blast, aeEndTick(t, "c"), "c").Effects(), "b").Equal(aeEffect(t, blast, "b")) {
		t.Error("effects that differ in the stored damage are equal")
	}
}
