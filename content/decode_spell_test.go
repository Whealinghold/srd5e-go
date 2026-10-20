package content_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The fixtures are synthetic: made-up names, numbers and texts, never SRD
// spells.

const spellHead = `{"id":"fixture_spell","name":"Fixture Spell","source":{"document":"Fixture","section":"Spells","page":1},` +
	`"level":2,"school":"Evocation","classes":["wizard","fixture_class"],"castingTimes":[{"kind":"magicAction"}],"ritual":false,` +
	`"range":{"kind":"distance","amount":60,"unit":"feet"},"components":{"verbal":true,"somatic":true,` +
	`"material":{"description":"a pinch of dust","costsGp":[],"consumesMaterial":false}},` +
	`"duration":{"kind":"timed","amount":1,"unit":"minutes","limit":"concentrationUpTo"},` +
	`"description":[{"kind":"paragraph","text":"Text."}],"higherLevel":[{"kind":"paragraph","text":"More."}]`

const spellJSON = spellHead + `}`

func spellDoc(entries ...string) string { return wrapDoc("spells", strings.Join(entries, ",")) }

// spellWith is the fixture spell with old replaced by repl, as a document.
func spellWith(old, repl string) string { return spellDoc(mutate(spellJSON, old, repl)) }

// spellFx is the fixture spell at level with the effects object fx, as a document.
func spellFx(level, fx string) string {
	return spellDoc(mutate(spellHead, `"level":2`, `"level":`+level) + `,"effects":` + fx + `}`)
}

func decodeOneSpell(t *testing.T, doc string) content.Spell {
	t.Helper()
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %q", decodeProblems(doc))
	}
	for _, e := range set.Entries() {
		if s, ok := e.(content.Spell); ok {
			return s
		}
	}
	t.Fatal("no spell in the set")
	return content.Spell{}
}

func expectedSpell() content.Spell {
	f := baseSpellFields()
	f.Classes = []content.ClassID{"wizard", "fixture_class"}
	f.Level = 2
	f.Range = must(content.NewSpellRangeDistance(60, content.DistanceFeet))
	f.Components = must(content.NewSpellComponents(true, true).WithMaterial(must(content.NewMaterialComponent("a pinch of dust", nil, false))))
	f.Duration = must(content.NewSpellDurationTimed(1, content.DurationUnitMinutes, content.LimitConcentrationUpTo))
	f.HigherLevel = []content.TextBlock{content.MustTextParagraph("More.")}
	return must(content.NewSpell("fixture_spell", "Fixture Spell", must(content.NewSourceCitation("Fixture", "Spells", 1)), f))
}

// Requirement: content-spells.md FR-092, FR-107, FR-115 and schema FR-008 - the
// spells root array is registered: absent is none, a non-array is wrongType, the
// fixture spell decodes to the Spell built from the same values, the old shape's
// keys are rejected, and a repeated id is duplicate at the later entry.
// Path: happy.
func TestDecodeSpellRoot(t *testing.T) {
	t.Parallel()
	if got := decodeProblems(`{"schemaVersion":1}`); got != nil {
		t.Errorf("no spells array: %v", got)
	}
	if got := decodeProblems(`{"schemaVersion":1,"spells":{}}`); !slices.Equal(got, []string{"/spells wrongType"}) {
		t.Errorf("spells object: %v", got)
	}
	got := decodeOneSpell(t, spellDoc(spellJSON))
	if want := expectedSpell(); !got.Equal(want) {
		t.Errorf("decoded %+v, want %+v", got, want)
	}
	runDocCases(t, []docCase{
		{"two spells, one id", spellsSpec + " FR-107 and schema FR-031", "unhappy (duplicate)", spellDoc(spellJSON, spellJSON), []string{"/spells/1/id duplicate"}},
		{"old shape", "FR-107", "unhappy (invalid input)", spellWith(`"higherLevel":[`, `"higherLevelSlot":"x","higherLevel":[`), []string{"/spells/0/higherLevelSlot unknownField"}},
		{"scaling key", "FR-115", "unhappy (invalid input)", spellWith(`"higherLevel":[`, `"scaling":{},"higherLevel":[`), []string{"/spells/0/scaling unknownField"}},
		{"element not an object", "FR-109", "unhappy (invalid input)", spellDoc(`[]`), []string{"/spells/0 wrongType"}},
	})
}

// Requirement: content-spells.md FR-107 to FR-111, "Cases: spell JSON" - each
// row decodes to the problem (path, kind) of the table, in the order of FR-107,
// with none for the valid rows.
// Path: per case.
func TestDecodeSpellCases(t *testing.T) {
	t.Parallel()
	const p = "/spells/0/"
	s := spellWith
	runDocCases(t, []docCase{
		{"valid minimal", "FR-107, FR-018", "happy", spellDoc(`{"id":"a","name":"A","level":0,"school":"Evocation","classes":[],"castingTimes":[{"kind":"magicAction"}],"ritual":true,"range":{"kind":"self"},"components":{"verbal":false,"somatic":false},"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"x"}]}`), nil},
		{"level too high", "FR-009, FR-109", "unhappy (boundary)", s(`"level":2`, `"level":10`), []string{p + "level outOfRange"}},
		{"level negative", "FR-009", "unhappy (boundary)", s(`"level":2`, `"level":-1`), []string{p + "level outOfRange"}},
		{"level 9", "FR-009", "unhappy (boundary)", s(`"level":2`, `"level":9`), nil},
		{"level 0", "FR-009", "unhappy (boundary)", s(`"level":2`, `"level":0`), nil},
		{"level a string", "FR-109", "unhappy (invalid input)", s(`"level":2`, `"level":"3"`), []string{p + "level wrongType"}},
		{"level fractional", "FR-109", "unhappy (invalid input)", s(`"level":2`, `"level":2.5`), []string{p + "level wrongType"}},
		{"level absent", "FR-107", "unhappy (missing content)", s(`"level":2,`, ``), []string{p + "level missingField"}},
		{"unknown school", "FR-108", "unhappy (invalid input)", s(`"Evocation"`, `"Lore"`), []string{p + "school invalidValue"}},
		{"school wrong case", "FR-108", "unhappy (invalid input)", s(`"Evocation"`, `"evocation"`), []string{p + "school invalidValue"}},
		{"class repeated", "FR-109", "unhappy (invalid input)", s(`["wizard","fixture_class"]`, `["wizard","wizard"]`), []string{p + "classes/1 duplicate"}},
		{"class not an id", "FR-110", "unhappy (invalid input)", s(`["wizard","fixture_class"]`, `["Wizard"]`), []string{p + "classes/0 invalidValue"}},
		{"class unknown to the document", "FR-110", "unhappy (rule interaction)", s(`["wizard","fixture_class"]`, `["unknown_class"]`), nil},
		{"classes empty", "FR-107", "unhappy (empty and minimal)", s(`["wizard","fixture_class"]`, `[]`), nil},
		{"classes not an array", "FR-109", "unhappy (invalid input)", s(`["wizard","fixture_class"]`, `"wizard"`), []string{p + "classes wrongType"}},
		{"no casting time", "FR-030, FR-109", "unhappy (empty and minimal)", s(`[{"kind":"magicAction"}]`, `[]`), []string{p + "castingTimes outOfRange"}},
		{"unknown casting kind", "FR-108", "unhappy (invalid input)", s(`{"kind":"magicAction"}`, `{"kind":"bonus"}`), []string{p + "castingTimes/0/kind invalidValue"}},
		{"unknown kind hides the label rule on a lone time", "FR-030, FR-108", "unhappy (rule interaction)", s(`{"kind":"magicAction"}`, `{"kind":"bonus","label":"Solo"}`), []string{p + "castingTimes/0/kind invalidValue"}},
		{"unknown kind hides the missing label of a mode", "FR-030, FR-108", "unhappy (rule interaction)", s(`[{"kind":"magicAction"}]`, `[{"kind":"magicAction","label":"A"},{"kind":"bonus"}]`), []string{p + "castingTimes/1/kind invalidValue"}},
		{"reaction without trigger", "FR-027", "unhappy (missing content)", s(`{"kind":"magicAction"}`, `{"kind":"reaction"}`), []string{p + "castingTimes/0/trigger missingField"}},
		{"reaction blank trigger", "FR-027", "unhappy (invalid input)", s(`{"kind":"magicAction"}`, `{"kind":"reaction","trigger":" "}`), []string{p + "castingTimes/0/trigger invalidValue"}},
		{"reaction with trigger", "FR-027", "happy", s(`{"kind":"magicAction"}`, `{"kind":"reaction","trigger":"when hit"}`), nil},
		{"bonus action with and without trigger", "FR-028", "unhappy (boundary)", s(`[{"kind":"magicAction"}]`, `[{"kind":"bonusAction","trigger":"t"}]`), nil},
		{"bonus action blank trigger", "FR-028", "unhappy (boundary)", s(`{"kind":"magicAction"}`, `{"kind":"bonusAction","trigger":""}`), []string{p + "castingTimes/0/trigger invalidValue"}},
		{"minutes zero", "FR-029", "unhappy (boundary)", s(`{"kind":"magicAction"}`, `{"kind":"minutes","amount":0}`), []string{p + "castingTimes/0/amount outOfRange"}},
		{"hours one", "FR-029", "unhappy (boundary)", s(`{"kind":"magicAction"}`, `{"kind":"hours","amount":1}`), nil},
		{"hours without amount", "FR-029", "unhappy (missing content)", s(`{"kind":"magicAction"}`, `{"kind":"hours"}`), []string{p + "castingTimes/0/amount missingField"}},
		{"amount on an action", "FR-109", "unhappy (invalid input)", s(`{"kind":"magicAction"}`, `{"kind":"magicAction","amount":5}`), []string{p + "castingTimes/0/amount unknownField"}},
		{"label on the only option", "FR-030", "unhappy (rule interaction)", s(`{"kind":"magicAction"}`, `{"kind":"magicAction","label":"Solo"}`), []string{p + "castingTimes/0/label constraintViolation"}},
		{"two labelled modes", "FR-030, FR-033", "happy", s(`[{"kind":"magicAction"}]`, `[{"kind":"magicAction","label":"A"},{"kind":"hours","amount":8,"label":"B"}]`), nil},
		{"second mode unlabelled", "FR-030", "unhappy (missing content)", s(`[{"kind":"magicAction"}]`, `[{"kind":"magicAction","label":"A"},{"kind":"magicAction"}]`), []string{p + "castingTimes/1/label missingField"}},
		{"blank mode label", "FR-030", "unhappy (invalid input)", s(`[{"kind":"magicAction"}]`, `[{"kind":"magicAction","label":"A"},{"kind":"magicAction","label":""}]`), []string{p + "castingTimes/1/label invalidValue"}},
		{"distance zero", "FR-037", "unhappy (boundary)", s(`"amount":60`, `"amount":0`), []string{p + "range/amount outOfRange"}},
		{"distance miles", "FR-037", "unhappy (boundary)", s(`"unit":"feet"`, `"unit":"miles"`), nil},
		{"distance unit unknown", "FR-108", "unhappy (invalid input)", s(`"unit":"feet"`, `"unit":"furlongs"`), []string{p + "range/unit invalidValue"}},
		{"distance without unit", "FR-109", "unhappy (missing content)", s(`,"unit":"feet"`, ``), []string{p + "range/unit missingField"}},
		{"unit on self", "FR-109", "unhappy (invalid input)", s(`{"kind":"distance","amount":60,"unit":"feet"}`, `{"kind":"self","unit":"feet"}`), []string{p + "range/unit unknownField"}},
		{"each other range", "FR-036", "happy", s(`{"kind":"distance","amount":60,"unit":"feet"}`, `{"kind":"sight"}`), nil},
		{"unknown range kind", "FR-108", "unhappy (invalid input)", s(`{"kind":"distance","amount":60,"unit":"feet"}`, `{"kind":"far"}`), []string{p + "range/kind invalidValue"}},
		{"verbal not a boolean", "FR-109", "unhappy (invalid input)", s(`"verbal":true`, `"verbal":"yes"`), []string{p + "components/verbal wrongType"}},
		{"verbal absent", "FR-109", "unhappy (missing content)", s(`"verbal":true,`, ``), []string{p + "components/verbal missingField"}},
		{"no material", "FR-049", "unhappy (empty and minimal)", s(`,"material":{"description":"a pinch of dust","costsGp":[],"consumesMaterial":false}`, ``), nil},
		{"empty material", "FR-041", "unhappy (invalid input)", s(`"description":"a pinch of dust"`, `"description":""`), []string{p + "components/material/description invalidValue"}},
		{"cost zero", "FR-101", "unhappy (boundary)", s(`"costsGp":[]`, `"costsGp":[0]`), []string{p + "components/material/costsGp/0 outOfRange"}},
		{"cost one and many", "FR-101", "unhappy (boundary)", s(`"costsGp":[]`, `"costsGp":[1,1000]`), nil},
		{"material without consumes", "FR-044", "unhappy (missing content)", s(`,"consumesMaterial":false`, ``), []string{p + "components/material/consumesMaterial missingField"}},
		{"timed without amount", "FR-109", "unhappy (missing content)", s(`"amount":1,"unit":"minutes"`, `"unit":"minutes"`), []string{p + "duration/amount missingField"}},
		{"timed zero", "FR-051", "unhappy (boundary)", s(`"amount":1,"unit":"minutes"`, `"amount":0,"unit":"minutes"`), []string{p + "duration/amount outOfRange"}},
		{"timed unit weeks", "FR-051", "unhappy (invalid input)", s(`"unit":"minutes"`, `"unit":"weeks"`), []string{p + "duration/unit invalidValue"}},
		{"concentration without a cap", "FR-052", "unhappy (rule interaction)", s(`"concentrationUpTo"`, `"concentration"`), []string{p + "duration/limit invalidValue"}},
		{"timed limits", "FR-052, FR-054", "happy", s(`"concentrationUpTo"`, `"upTo"`), nil},
		{"amount on instantaneous", "FR-109", "unhappy (invalid input)", s(`{"kind":"timed","amount":1,"unit":"minutes","limit":"concentrationUpTo"}`, `{"kind":"instantaneous","amount":1}`), []string{p + "duration/amount unknownField"}},
		{"other durations", "FR-050", "happy", s(`{"kind":"timed","amount":1,"unit":"minutes","limit":"concentrationUpTo"}`, `{"kind":"untilDispelledOrTriggered"}`), nil},
		{"description empty", "FR-058", "unhappy (empty and minimal)", s(`"description":[{"kind":"paragraph","text":"Text."}]`, `"description":[]`), []string{p + "description outOfRange"}},
		{"description absent", "FR-107", "unhappy (missing content)", s(`"description":[{"kind":"paragraph","text":"Text."}],`, ``), []string{p + "description missingField"}},
		{"blank paragraph", "FR-059", "unhappy (invalid input)", s(`"text":"Text."`, `"text":"  "`), []string{p + "description/0/text invalidValue"}},
		{"unknown block kind", "FR-059", "unhappy (invalid input)", s(`{"kind":"paragraph","text":"Text."}`, `{"kind":"quote","text":"x"}`), []string{p + "description/0/kind invalidValue"}},
		{"ragged table", "FR-059", "unhappy (rule interaction)", s(`{"kind":"paragraph","text":"Text."}`, `{"kind":"table","header":["a","b"],"rows":[["1","2"],["1","2","3"]]}`), []string{p + "description/0/rows/1 constraintViolation"}},
		{"table without rows", "FR-059", "unhappy (empty and minimal)", s(`{"kind":"paragraph","text":"Text."}`, `{"kind":"table","header":["a"],"rows":[]}`), []string{p + "description/0/rows outOfRange"}},
		{"empty higher-level text", "FR-102", "unhappy (empty and minimal)", s(`"higherLevel":[{"kind":"paragraph","text":"More."}]`, `"higherLevel":[]`), []string{p + "higherLevel outOfRange"}},
		{"empty cantrip upgrade", "FR-102", "unhappy (empty and minimal)", s(`"higherLevel":[{"kind":"paragraph","text":"More."}]`, `"cantripUpgrade":[]`), []string{p + "cantripUpgrade outOfRange"}},
		{"cantrip upgrade present", "FR-102", "happy", s(`"higherLevel":[{"kind":"paragraph","text":"More."}]`, `"cantripUpgrade":[{"kind":"paragraph","text":"Up."}]`), nil},
		{"id not valid", "FR-113", "unhappy (invalid input)", s(`"id":"fixture_spell"`, `"id":"Acid-Arrow"`), []string{p + "id invalidValue"}},
		{"name padded", "FR-098", "unhappy (invalid input)", s(`"name":"Fixture Spell"`, `"name":" Fixture Spell"`), []string{p + "name invalidValue"}},
		{"citation page zero", "schema FR-115", "unhappy (boundary)", s(`"page":1`, `"page":0`), []string{p + "source/page outOfRange"}},
		{"citation section absent", "schema FR-115", "unhappy (missing content)", s(`"section":"Spells",`, ``), []string{p + "source/section missingField"}},
		{"citation blank document", "schema FR-115", "unhappy (invalid input)", s(`"document":"Fixture"`, `"document":" "`), []string{p + "source/document invalidValue"}},
		{"citation absent", "FR-007", "unhappy (empty and minimal)", s(`"source":{"document":"Fixture","section":"Spells","page":1},`, ``), nil},
		{"two problems, one spell", "FR-107, FR-109", "unhappy (rule interaction)", s(`"level":2,"school":"Evocation"`, `"level":10,"school":"Lore"`), []string{p + "level outOfRange", p + "school invalidValue"}},
		{"integer past 2^31", "schema FR-021", "unhappy (boundary)", s(`"amount":60`, `"amount":2147483648`), []string{p + "range/amount outOfRange"}},
		{"integer 2^31 - 1", "schema FR-021", "unhappy (boundary)", s(`"amount":60`, `"amount":2147483647`), nil},
	})
}

// Requirement: content-spells.md FR-108 - every spelling of the kinds decodes to
// its variant, with the values given.
// Path: happy.
func TestDecodeSpellVariants(t *testing.T) {
	t.Parallel()
	s := decodeOneSpell(t, spellWith(`[{"kind":"magicAction"}]`, `[{"kind":"minutes","amount":10,"label":"Slow"},{"kind":"reaction","trigger":"when seen","label":"Fast"},{"kind":"bonusAction","label":"Bonus"}]`))
	modes, ok := s.CastingTimes().(content.SpellCastingTimesModes)
	if !ok || len(modes.Modes()) != 3 || modes.Modes()[0].Time() != content.CastingTime(must(content.NewCastingTimeMinutes(10))) ||
		modes.Modes()[1].Time() != content.CastingTime(must(content.NewCastingTimeReaction("when seen"))) || modes.Modes()[2].Label() != "Bonus" {
		t.Errorf("casting times = %#v", s.CastingTimes())
	}
	for kind, want := range map[string]string{
		`{"kind":"self"}`: "self", `{"kind":"touch"}`: "touch", `{"kind":"sight"}`: "sight", `{"kind":"unlimited"}`: "unlimited",
		`{"kind":"special"}`: "special", `{"kind":"distance","amount":1,"unit":"miles"}`: "distance",
	} {
		r := decodeOneSpell(t, spellWith(`{"kind":"distance","amount":60,"unit":"feet"}`, kind)).Range()
		if rangeLabel(r) != want {
			t.Errorf("range %s = %T", kind, r)
		}
		if d, ok := r.(content.SpellRangeDistance); ok && (d.Amount() != 1 || d.Unit() != content.DistanceMiles) {
			t.Errorf("distance = %v", d)
		}
	}
	for kind, want := range map[string]string{
		`{"kind":"instantaneous"}`: "instantaneous", `{"kind":"untilDispelled"}`: "dispelled", `{"kind":"untilDispelledOrTriggered"}`: "triggered",
		`{"kind":"special"}`: "special", `{"kind":"timed","amount":6,"unit":"rounds","limit":"fixed"}`: "timed",
	} {
		d := decodeOneSpell(t, spellWith(`{"kind":"timed","amount":1,"unit":"minutes","limit":"concentrationUpTo"}`, kind)).Duration()
		if durationLabel(d) != want {
			t.Errorf("duration %s = %T", kind, d)
		}
	}
	m, _ := decodeOneSpell(t, spellWith(`"costsGp":[]`, `"costsGp":[50,5]`)).Components().Material()
	if !slices.Equal(m.CostsGP(), []int{50, 5}) {
		t.Errorf("costs = %v", m.CostsGP())
	}
}

// ---- effects ----

const fxTargeting = `{"kind":"targetsChosen","subject":"creature","count":{"kind":"upToTargets","n":2,"scaling":{"kind":"perSlotLevelCount","aboveLevel":2,"extra":1}},` +
	`"restriction":{"kind":"onlyCreatureTypes","types":["Humanoid"]},"selfPolicy":"excludesCaster"}`

const fxDamage = `{"kind":"spellDealsDamage","amount":{"base":{"kind":"rolledSpellAmount","roll":"2d6"},` +
	`"scaling":{"kind":"perSlotLevelAmount","aboveLevel":2,"extraDice":1}},"damageType":{"kind":"fixedDamageType","type":"Fire"}}`

const fxCondition = `{"kind":"spellAppliesCondition","condition":"Frightened","duration":{"kind":"untilNextTurnBoundary","owner":"caster","boundary":"end"}}`

const fxPart = `{"resolution":{"kind":"savingThrowResolution","save":"wisdom","onFailure":[` + fxDamage + `,` + fxCondition + `],"onSuccess":"halfDamageOnSuccess"}}`

func fxOf(targeting, parts string) string {
	return `{"coverage":"partial","targeting":` + targeting + `,"parts":[` + parts + `]}`
}

const fxA = `{"coverage":"partial","targeting":` + fxTargeting + `,"parts":[` + fxPart + `]}`

func expectedFxA() content.ModelledEffects {
	humanoid := must(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
	count := must(content.NewUpToTargets(2, must(content.NewPerSlotLevelCount(2, 1))))
	targeting := must(content.NewTargetsChosen(content.SubjectCreature, count, humanoid, content.ExcludesCaster))
	dmg := damageOf(amount(rolled("2d6"), perSlot(2, 1, 0)), fixedType(content.DamageFire))
	cond := must(content.NewSpellAppliesCondition(content.ConditionFrightened,
		must(content.NewUntilNextTurnBoundary(content.OwnerCaster, content.BoundaryEnd))))
	return modelled(content.CoveragePartial, targeting, part(save(ability.Wisdom, content.HalfDamageOnSuccess, dmg, cond), once()))
}

const fxZone = `{"rules":[{"trigger":"travelsWithin","partIndex":0,"scope":"anyCreature","unitFeet":5},` +
	`{"trigger":"creatureEnters","partIndex":0,"scope":"enemiesOfCaster","region":"edge","casterChoice":true}],` +
	`"mobility":{"kind":"movedByCaster","maxFeet":30,"action":"bonusAction","laterTurnsOnly":true},"oncePerTurn":true,` +
	`"designation":"atCastingOnly","damageBudget":20,"stopsMoverOnContact":true,` +
	`"movement":{"difficultTerrain":true,"speedHalved":true,"extraFeetPerFoot":3}}`

const fxCantripTargeting = `{"kind":"targetsInArea","shape":{"kind":"sphereArea","radiusFeet":10},"subject":"creature"}`

const fxCantripPart = `{"resolution":{"kind":"attackRollResolution","attack":"ranged","onHit":[{"kind":"spellDealsDamage","amount":{"base":{"kind":"rolledSpellAmount","roll":"1d6"},` +
	`"scaling":{"kind":"byCharacterLevelAmount","entries":[{"level":1,"value":{"kind":"rolledSpellAmount","roll":"1d6"}},{"level":5,"value":{"kind":"rolledSpellAmount","roll":"2d6"}}]}},` +
	`"damageType":{"kind":"chosenDamageType","types":["Acid","Cold"]},"addsModifier":"spellcastingAbilityModifier"}],` +
	`"onMiss":[{"kind":"spellGrantsTemporaryHitPoints","amount":{"base":{"kind":"fixedSpellAmount","value":3}}}]},` +
	`"instances":{"base":1,"scaling":{"kind":"byCharacterLevelCount","entries":[{"level":1,"count":1},{"level":5,"count":2}]}}}`

func fxB(zone string) string {
	s := `{"coverage":"complete","targeting":` + fxCantripTargeting + `,"parts":[` + fxCantripPart + `]`
	if zone != "" {
		s += `,"zone":` + zone
	}
	return s + `}`
}

func expectedFxB(withZone bool) content.ModelledEffects {
	tbl := amountTable(content.AmountLevelEntry{Level: 1, Value: rolled("1d6")}, content.AmountLevelEntry{Level: 5, Value: rolled("2d6")})
	chosenType := must(content.NewChosenDamageType([]content.DamageType{content.DamageAcid, content.DamageCold}))
	dmg := must(content.NewSpellDealsDamage(amount(rolled("1d6"), tbl), chosenType, content.AddsSpellcastingAbilityModifier))
	temp := must(content.NewSpellGrantsTemporaryHitPoints(amount(fixed(3), nil), content.AddsNothing))
	res := must(content.NewAttackRollResolution(content.SpellAttackRanged, []content.SpellOutcome{dmg}, []content.SpellOutcome{temp}))
	beams := must(content.NewByCharacterLevelCount(must(content.NewCountLevelTable([]content.CountLevelEntry{{Level: 1, Count: 1}, {Level: 5, Count: 2}}))))
	m := modelled(content.CoverageComplete, area(sphere(10)), part(res, must(content.NewPartInstances(1, beams))))
	if !withZone {
		return m
	}
	rules := []content.ZoneRuleFact{
		must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.TravelsWithin, Scope: content.AnyCreature, UnitFeet: 5})),
		must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, Scope: content.EnemiesOfCaster, Region: "edge", CasterChoice: true})),
	}
	zone := must(content.NewZoneFacts(content.ZoneFactsFields{
		Rules: rules, Mobility: must(content.NewMovedByCaster(30, content.BonusAction, true)), OncePerTurn: true,
		Designation: content.DesignationAtCastingOnly, DamageBudget: 20, StopsMoverOnContact: true,
		Movement: must(content.NewZoneMovementFacts(true, true, 3)),
	}))
	return must(m.WithZoneFacts(zone))
}

// Requirement: spell-effect-data.md FR-030, FR-032 - the effects key decodes to
// the model built by the constructors from the same values (every variant of
// targeting, resolution, outcome, scaling and zone read here), an absent key is
// TextOnlyEffects, and the equal documents decode to equal spells.
// Path: happy.
func TestDecodeSpellEffects(t *testing.T) {
	t.Parallel()
	if _, ok := decodeOneSpell(t, spellDoc(spellJSON)).Effects().(content.TextOnlyEffects); !ok {
		t.Error("an absent effects key is not TextOnlyEffects [FR-030]")
	}
	for _, c := range []struct {
		name  string
		level string
		fx    string
		want  content.ModelledEffects
	}{
		{"saving throw with scaling, restriction and condition", "2", fxA, expectedFxA()},
		{"cantrip with tables, a chosen type, a miss outcome and no zone", "0", fxB(""), expectedFxB(false)},
		{"cantrip with a zone", "0", fxB(fxZone), expectedFxB(true)},
	} {
		got := decodeOneSpell(t, spellFx(c.level, c.fx)).Effects()
		if !got.Equal(c.want) {
			t.Errorf("%s: decoded %#v, want %#v", c.name, got, c.want)
		}
	}
	zone, _ := expectedFxB(true).Zone()
	if zone.DamageBudget() != 20 {
		t.Fatal("fixture")
	}
}

// Requirement: spell-effect-data.md FR-010, FR-013, FR-040 - each area shape, each
// target count and the self targeting decode, and a missing selfPolicy is
// MayIncludeCaster.
// Path: happy.
func TestDecodeSpellTargetingForms(t *testing.T) {
	t.Parallel()
	partJSON := `{"resolution":{"kind":"automaticResolution","outcomes":[{"kind":"spellRestoresHitPoints","amount":{"base":{"kind":"fixedSpellAmount","value":3}}}]}}`
	for shape, want := range map[string]content.AreaShape{
		`{"kind":"coneArea","lengthFeet":15}`:                    must(content.NewConeArea(15)),
		`{"kind":"cubeArea","sideFeet":10}`:                      must(content.NewCubeArea(10)),
		`{"kind":"cylinderArea","radiusFeet":5,"heightFeet":20}`: must(content.NewCylinderArea(5, 20)),
		`{"kind":"emanationArea","distanceFeet":30}`:             must(content.NewEmanationArea(30)),
		`{"kind":"lineArea","lengthFeet":60,"widthFeet":5}`:      must(content.NewLineArea(60, 5)),
		`{"kind":"sphereArea","radiusFeet":20}`:                  must(content.NewSphereArea(20)),
	} {
		fx := fxOf(`{"kind":"targetsInArea","shape":`+shape+`,"subject":"object"}`, partJSON)
		got, ok := decodeOneSpell(t, spellFx("2", fx)).Effects().(content.ModelledEffects)
		if !ok {
			t.Fatalf("%s: not modelled", shape)
		}
		if a, ok := got.Targeting().(content.TargetsInArea); !ok || a.Shape() != want || a.Subject() != content.SubjectObject || a.Restriction() != content.CreatureTypeRestriction(content.AnyCreatureType{}) {
			t.Errorf("%s: targeting = %#v", shape, got.Targeting())
		}
	}
	for count, want := range map[string]content.TargetCount{
		`{"kind":"exactlyTargets","n":3}`:                                  must(content.NewExactlyTargets(3, nil)),
		`{"kind":"targetsByInstance"}`:                                     content.TargetsByInstance{},
		`{"kind":"upToTargets","n":1,"scaling":{"kind":"noCountScaling"}}`: must(content.NewUpToTargets(1, content.NoCountScaling{})),
	} {
		fx := fxOf(`{"kind":"targetsChosen","subject":"creatureOrObject","count":`+count+`,"restriction":{"kind":"anyCreatureType"}}`, partJSON)
		got := decodeOneSpell(t, spellFx("2", fx)).Effects().(content.ModelledEffects)
		if c, ok := got.Targeting().(content.TargetsChosen); !ok || c.Count() != want || c.SelfPolicy() != content.MayIncludeCaster || c.Subject() != content.SubjectCreatureOrObject {
			t.Errorf("%s: targeting = %#v", count, got.Targeting())
		}
	}
	self := decodeOneSpell(t, spellFx("2", fxOf(`{"kind":"targetsSelf"}`, partJSON))).Effects().(content.ModelledEffects)
	if self.Targeting() != content.SpellTargeting(content.TargetsSelf{}) {
		t.Errorf("targeting = %#v", self.Targeting())
	}
}

func inFx(old, repl string) string   { return spellFx("2", mutate(fxA, old, repl)) }
func inZone(old, repl string) string { return spellFx("0", fxB(mutate(fxZone, old, repl))) }

// Requirement: spell-effect-data.md FR-031 - every problem of effect data has the
// path and the kind of the rule-to-kind table (outOfRange, constraintViolation,
// duplicate, invalidValue, missingField, wrongType, unknownField), all reported
// together in the order of the declared keys.
// Path: per case.
func TestDecodeSpellEffectsProblems(t *testing.T) {
	t.Parallel()
	const e = "/spells/0/effects/"
	const tg = e + "targeting/"
	const pr = e + "parts/0/resolution/"
	const amt = pr + "onFailure/0/amount/"
	fixedWithStep := fxOf(fxTargeting, `{"resolution":{"kind":"automaticResolution","outcomes":[{"kind":"spellRestoresHitPoints","amount":{"base":{"kind":"fixedSpellAmount","value":5},`+
		`"scaling":{"kind":"perSlotLevelAmount","aboveLevel":2,"extraDice":1}}}]}}`)
	chosenDamage := `{"kind":"spellDealsDamage","amount":{"base":{"kind":"fixedSpellAmount","value":5}},"damageType":{"kind":"chosenDamageType","types":["Acid","Cold"]}}`
	twoChosen := fxOf(fxTargeting, `{"resolution":{"kind":"automaticResolution","outcomes":[`+chosenDamage+`,`+chosenDamage+`]}}`)
	halfNoDamage := fxOf(fxTargeting, `{"resolution":{"kind":"savingThrowResolution","save":"wisdom","onFailure":[`+fxCondition+`],"onSuccess":"halfDamageOnSuccess"}}`)
	runDocCases(t, []docCase{
		{"coverage unknown", "FR-031", "unhappy (invalid input)", inFx(`"coverage":"partial"`, `"coverage":"all"`), []string{e + "coverage invalidValue"}},
		{"coverage absent", "FR-031", "unhappy (missing content)", inFx(`"coverage":"partial",`, ``), []string{e + "coverage missingField"}},
		{"coverage a number", "FR-031", "unhappy (invalid input)", inFx(`"coverage":"partial"`, `"coverage":1`), []string{e + "coverage wrongType"}},
		{"effects not an object", "FR-031", "unhappy (invalid input)", spellFx("2", `[]`), []string{e[:len(e)-1] + " wrongType"}},
		{"effects null", "FR-031", "unhappy (invalid input)", spellFx("2", `null`), []string{e[:len(e)-1] + " wrongType"}},
		{"effects unknown key", "FR-031", "unhappy (invalid input)", inFx(`"coverage"`, `"extra":1,"coverage"`), []string{e + "extra unknownField"}},
		{"targeting kind unknown", "FR-031", "unhappy (invalid input)", inFx(`"kind":"targetsChosen"`, `"kind":"targetsAll"`), []string{tg + "kind invalidValue"}},
		{"targeting kind absent", "FR-031", "unhappy (missing content)", inFx(`"kind":"targetsChosen",`, ``), []string{tg + "kind missingField"}},
		{"targeting absent", "FR-031", "unhappy (missing content)", mutate2(fxA, `"targeting":`+fxTargeting+`,`, ``), []string{e + "targeting missingField"}},
		{"count zero", "FR-012", "unhappy (boundary)", inFx(`"n":2`, `"n":0`), []string{tg + "count/n outOfRange"}},
		{"count negative", "FR-012", "unhappy (boundary)", inFx(`"n":2`, `"n":-1`), []string{tg + "count/n outOfRange"}},
		{"count kind unknown", "FR-012", "unhappy (invalid input)", inFx(`"upToTargets"`, `"allTargets"`), []string{tg + "count/kind invalidValue"}},
		{"extra of zero", "FR-017", "unhappy (boundary)", inFx(`"extra":1`, `"extra":0`), []string{tg + "count/scaling/extra outOfRange"}},
		{"restriction types empty", "FR-015", "unhappy (empty and minimal)", inFx(`["Humanoid"]`, `[]`), []string{tg + "restriction/types outOfRange"}},
		{"restriction type repeated", "FR-015", "unhappy (invalid input)", inFx(`["Humanoid"]`, `["Humanoid","Humanoid"]`), []string{tg + "restriction/types/1 duplicate"}},
		{"restriction type unknown", "FR-015", "unhappy (invalid input)", inFx(`["Humanoid"]`, `["humanoid"]`), []string{tg + "restriction/types/0 invalidValue"}},
		{"restriction on an object", "FR-015", "unhappy (rule interaction)", inFx(`"subject":"creature"`, `"subject":"object"`), []string{tg + "restriction constraintViolation"}},
		{"subject unknown", "FR-011", "unhappy (invalid input)", inFx(`"subject":"creature"`, `"subject":"point"`), []string{tg + "subject invalidValue"}},
		{"self policy unknown", "FR-040", "unhappy (invalid input)", inFx(`"excludesCaster"`, `"never"`), []string{tg + "selfPolicy invalidValue"}},
		{"parts empty", "FR-006", "unhappy (empty and minimal)", spellFx("2", fxOf(fxTargeting, ``)), []string{e + "parts outOfRange"}},
		{"parts not an array", "FR-031", "unhappy (invalid input)", inFx(`"parts":[`+fxPart+`]`, `"parts":{}`), []string{e + "parts wrongType"}},
		{"roll not canonical", "FR-025", "unhappy (invalid input)", inFx(`"roll":"2d6"`, `"roll":"d6"`), []string{amt + "base/roll invalidValue"}},
		{"roll malformed", "FR-025", "unhappy (invalid input)", inFx(`"roll":"2d6"`, `"roll":"2d"`), []string{amt + "base/roll invalidValue"}},
		{"roll a number", "FR-025", "unhappy (invalid input)", inFx(`"roll":"2d6"`, `"roll":6`), []string{amt + "base/roll wrongType"}},
		{"fixed zero", "FR-025", "unhappy (boundary)", inFx(`{"kind":"rolledSpellAmount","roll":"2d6"}`, `{"kind":"fixedSpellAmount","value":0}`), []string{amt + "base/value outOfRange"}},
		{"amount kind unknown", "FR-025", "unhappy (invalid input)", inFx(`"rolledSpellAmount"`, `"fixedDice"`), []string{amt + "base/kind invalidValue"}},
		{"dice on a fixed base", "FR-016", "unhappy (rule interaction)", spellFx("2", fixedWithStep), []string{e + "parts/0/resolution/outcomes/0/amount/scaling constraintViolation"}},
		{"both step parts zero", "FR-016", "unhappy (boundary)", inFx(`"extraDice":1`, `"extraDice":0`), []string{amt + "scaling outOfRange"}},
		{"step part negative", "FR-016", "unhappy (boundary)", inFx(`"extraDice":1`, `"extraDice":-1`), []string{amt + "scaling/extraDice outOfRange"}},
		{"flat step only", "FR-016", "unhappy (boundary)", inFx(`"extraDice":1`, `"extraFlat":2`), nil},
		{"damage type unknown", "FR-044", "unhappy (invalid input)", inFx(`"type":"Fire"`, `"type":"fire"`), []string{amt[:len(amt)-len("amount/")] + "damageType/type invalidValue"}},
		{"chosen set of one", "FR-027", "unhappy (boundary)", inFx(`{"kind":"fixedDamageType","type":"Fire"}`, `{"kind":"chosenDamageType","types":["Fire"]}`), []string{amt[:len(amt)-len("amount/")] + "damageType/types outOfRange"}},
		{"chosen set repeated", "FR-027", "unhappy (invalid input)", inFx(`{"kind":"fixedDamageType","type":"Fire"}`, `{"kind":"chosenDamageType","types":["Fire","Fire"]}`), []string{amt[:len(amt)-len("amount/")] + "damageType/types/1 duplicate"}},
		{"two chosen types", "FR-045", "unhappy (rule interaction)", spellFx("2", twoChosen), []string{e + "parts constraintViolation"}},
		{"half damage with none to halve", "FR-043", "unhappy (rule interaction)", spellFx("2", halfNoDamage), []string{e + "parts/0/resolution/onSuccess constraintViolation"}},
		{"on failure empty", "FR-026", "unhappy (empty and minimal)", inFx(`"onFailure":[`+fxDamage+`,`+fxCondition+`]`, `"onFailure":[]`), []string{pr + "onFailure outOfRange"}},
		{"save unknown", "FR-042", "unhappy (invalid input)", inFx(`"wisdom"`, `"luck"`), []string{pr + "save invalidValue"}},
		{"success effect unknown", "FR-042", "unhappy (invalid input)", inFx(`"halfDamageOnSuccess"`, `"halfDamage"`), []string{pr + "onSuccess invalidValue"}},
		{"resolution kind unknown", "FR-042", "unhappy (invalid input)", inFx(`"savingThrowResolution"`, `"skillCheck"`), []string{pr + "kind invalidValue"}},
		{"outcome kind unknown", "FR-047", "unhappy (invalid input)", inFx(`"spellDealsDamage"`, `"spellDealsHarm"`), []string{pr + "onFailure/0/kind invalidValue"}},
		{"condition unknown", "FR-028", "unhappy (invalid input)", inFx(`"Frightened"`, `"Scared"`), []string{pr + "onFailure/1/condition invalidValue"}},
		{"condition duration amount zero", "FR-046", "unhappy (boundary)", inFx(`{"kind":"untilNextTurnBoundary","owner":"caster","boundary":"end"}`, `{"kind":"timedConditionDuration","amount":0,"unit":"rounds"}`), []string{pr + "onFailure/1/duration/amount outOfRange"}},
		{"condition duration timed", "FR-046", "happy", inFx(`{"kind":"untilNextTurnBoundary","owner":"caster","boundary":"end"}`, `{"kind":"timedConditionDuration","amount":1,"unit":"rounds"}`), nil},
		{"condition duration owner unknown", "FR-046", "unhappy (invalid input)", inFx(`"owner":"caster"`, `"owner":"nobody"`), []string{pr + "onFailure/1/duration/owner invalidValue"}},
		{"instances base zero", "FR-029", "unhappy (boundary)", inFx(`"halfDamageOnSuccess"}}`, `"halfDamageOnSuccess"},"instances":{"base":0}}`), []string{e + "parts/0/instances/base outOfRange"}},
		{"scaling above the spell level", "FR-009", "unhappy (rule interaction)", inFx(`"aboveLevel":2,"extra":1`, `"aboveLevel":1,"extra":1`), []string{e[:len(e)-1] + " constraintViolation"}},
		{"negative amount scaling level", "FR-009", "unhappy (rule interaction)", inFx(`"aboveLevel":2,"extraDice":1`, `"aboveLevel":-1,"extraDice":1`), []string{e[:len(e)-1] + " constraintViolation"}},
		{"negative scaling level", "FR-009", "unhappy (rule interaction)", inFx(`"aboveLevel":2,"extra":1`, `"aboveLevel":-1,"extra":1`), []string{e[:len(e)-1] + " constraintViolation"}},
		{"scaling above level 9", "FR-009", "unhappy (rule interaction)", inFx(`"aboveLevel":2,"extra":1`, `"aboveLevel":10,"extra":1`), []string{e[:len(e)-1] + " constraintViolation"}},
		{"per slot scaling on a cantrip", "FR-009", "unhappy (rule interaction)", spellFx("0", fxA), []string{e[:len(e)-1] + " constraintViolation"}},
		{"table on a level 2 spell", "FR-009", "unhappy (rule interaction)", spellFx("2", fxB("")), []string{e[:len(e)-1] + " constraintViolation"}},
		{"two problems are both reported", "FR-031", "unhappy (rule interaction)", inFx(`"coverage":"partial"`, `"coverage":"x","extra":1`), []string{e + "coverage invalidValue", e + "extra unknownField"}},
	})
}

func mutate2(base, old, repl string) string { return spellFx("2", mutate(base, old, repl)) }

// Requirement: spell-effect-data.md FR-018, FR-019, FR-031 - a character-level
// table must start at level 1, ascend strictly and stay within 1 to 20: its
// problems are outOfRange for an empty table or a level or count out of range
// and constraintViolation for the order.
// Path: per case.
func TestDecodeSpellEffectsTables(t *testing.T) {
	t.Parallel()
	const entries = "/spells/0/effects/parts/0/instances/scaling/entries"
	const amount = "/spells/0/effects/parts/0/resolution/onHit/0/amount/scaling/entries"
	inB := func(old, repl string) string { return spellFx("0", mutate(fxB(""), old, repl)) }
	runDocCases(t, []docCase{
		{"table valid", "FR-019", "happy", inB(`{"level":5,"count":2}`, `{"level":20,"count":2}`), nil},
		{"first level 2", "FR-019", "unhappy (invalid input)", inB(`{"level":1,"count":1}`, `{"level":2,"count":1}`), []string{entries + " constraintViolation"}},
		{"levels not ascending", "FR-019", "unhappy (invalid input)", inB(`{"level":5,"count":2}`, `{"level":1,"count":2}`), []string{entries + " constraintViolation"}},
		{"level 21", "FR-019", "unhappy (boundary)", inB(`{"level":5,"count":2}`, `{"level":21,"count":2}`), []string{entries + "/1/level outOfRange"}},
		{"level 0", "FR-019", "unhappy (boundary)", inB(`{"level":1,"count":1}`, `{"level":0,"count":1}`), []string{entries + "/0/level outOfRange"}},
		{"count zero", "FR-019", "unhappy (boundary)", inB(`{"level":5,"count":2}`, `{"level":5,"count":0}`), []string{entries + "/1/count outOfRange"}},
		{"entries empty", "FR-019", "unhappy (empty and minimal)", inB(`[{"level":1,"count":1},{"level":5,"count":2}]`, `[]`), []string{entries + " outOfRange"}},
		{"amount table first level 2", "FR-019", "unhappy (invalid input)", inB(`{"level":1,"value"`, `{"level":2,"value"`), []string{amount + " constraintViolation"}},
		{"amount entry a bad roll", "FR-019", "unhappy (invalid input)", inB(`"roll":"2d6"}}]`, `"roll":"2d"}}]`), []string{amount + "/1/value/roll invalidValue"}},
	})
}

// Requirement: spell-effect-data.md FR-048, FR-049 - the zone object decodes the
// problems of FR-049 with the kinds of FR-031: outOfRange for an empty rule list
// and for a unitFeet, maxFeet, feet or damageBudget below 1 and an
// extraFeetPerFoot below 0, constraintViolation for unitFeet missing on
// travelsWithin or present on another trigger and for a partIndex outside the
// parts, and duplicate for a repeated identical rule. A damageBudget or unitFeet
// key that is present is at least 1; an absent one is 0 (the decision recorded in
// the spec).
// Path: per case.
func TestDecodeSpellZone(t *testing.T) {
	t.Parallel()
	const z = "/spells/0/effects/zone/"
	const r0, r1 = z + "rules/0/", z + "rules/1/"
	rule0 := `{"trigger":"travelsWithin","partIndex":0,"scope":"anyCreature","unitFeet":5}`
	badPart := `{"resolution":{"kind":"automaticResolution","outcomes":[]}}`
	runDocCases(t, []docCase{
		{"valid", "FR-048", "happy", inZone(`"casterChoice":true`, `"casterChoice":false`), nil},
		{"rules empty", "FR-049", "unhappy (empty and minimal)", spellFx("0", fxB(`{"rules":[]}`)), []string{z + "rules outOfRange"}},
		{"unitFeet missing on travelsWithin", "FR-049", "unhappy (rule interaction)", inZone(`,"unitFeet":5`, ``), []string{r0 + "unitFeet constraintViolation"}},
		{"unitFeet present on another trigger", "FR-049", "unhappy (rule interaction)", inZone(`"region":"edge"`, `"region":"edge","unitFeet":5`), []string{r1 + "unitFeet constraintViolation"}},
		{"unitFeet zero", "FR-049", "unhappy (boundary)", inZone(`"unitFeet":5`, `"unitFeet":0`), []string{r0 + "unitFeet outOfRange"}},
		{"unitFeet zero on another trigger", "FR-049", "unhappy (boundary)", inZone(`"region":"edge"`, `"region":"edge","unitFeet":0`), []string{r1 + "unitFeet outOfRange"}},
		{"unitFeet one", "FR-049", "unhappy (boundary)", inZone(`"unitFeet":5`, `"unitFeet":1`), nil},
		{"partIndex past the parts", "FR-049", "unhappy (boundary)", inZone(`"partIndex":0,"scope":"anyCreature"`, `"partIndex":1,"scope":"anyCreature"`), []string{r0 + "partIndex constraintViolation"}},
		{"partIndex negative", "FR-049", "unhappy (boundary)", inZone(`"partIndex":0,"scope":"anyCreature"`, `"partIndex":-1,"scope":"anyCreature"`), []string{r0 + "partIndex constraintViolation"}},
		{"partIndex checked only for a clean model", "FR-049", "unhappy (rule interaction)", spellFx("0", mutate(mutate(fxB(fxZone), `"partIndex":0,"scope":"anyCreature"`, `"partIndex":7,"scope":"anyCreature"`), fxCantripPart, badPart)), []string{"/spells/0/effects/parts/0/resolution/outcomes outOfRange"}},
		{"duplicate rule", "FR-049", "unhappy (invalid input)", inZone(`"rules":[`+rule0, `"rules":[`+rule0+`,`+rule0), []string{z + "rules/1 duplicate"}},
		{"region blank", "FR-048", "unhappy (invalid input)", inZone(`"region":"edge"`, `"region":""`), []string{r1 + "region invalidValue"}},
		{"region absent", "FR-048", "unhappy (boundary)", inZone(`"region":"edge",`, ``), nil},
		{"trigger unknown", "FR-048", "unhappy (invalid input)", inZone(`"travelsWithin"`, `"always"`), []string{r0 + "trigger invalidValue"}},
		{"damageBudget zero", "FR-049", "unhappy (boundary)", inZone(`"damageBudget":20`, `"damageBudget":0`), []string{z + "damageBudget outOfRange"}},
		{"damageBudget one", "FR-049", "unhappy (boundary)", inZone(`"damageBudget":20`, `"damageBudget":1`), nil},
		{"damageBudget absent", "FR-049", "unhappy (boundary)", inZone(`"damageBudget":20,`, ``), nil},
		{"extraFeetPerFoot negative", "FR-049", "unhappy (boundary)", inZone(`"extraFeetPerFoot":3`, `"extraFeetPerFoot":-1`), []string{z + "movement/extraFeetPerFoot outOfRange"}},
		{"extraFeetPerFoot zero", "FR-049", "unhappy (boundary)", inZone(`"extraFeetPerFoot":3`, `"extraFeetPerFoot":0`), nil},
		{"movement absent", "FR-048", "unhappy (empty and minimal)", inZone(`,"movement":{"difficultTerrain":true,"speedHalved":true,"extraFeetPerFoot":3}`, ``), nil},
		{"maxFeet zero", "FR-049", "unhappy (boundary)", inZone(`"maxFeet":30`, `"maxFeet":0`), []string{z + "mobility/maxFeet outOfRange"}},
		{"action unknown", "FR-048", "unhappy (invalid input)", inZone(`"bonusAction"`, `"freeAction"`), []string{z + "mobility/action invalidValue"}},
		{"drift", "FR-048", "happy", inZone(`{"kind":"movedByCaster","maxFeet":30,"action":"bonusAction","laterTurnsOnly":true}`, `{"kind":"driftsAtTurnStart","feet":10,"direction":"awayFromCasterChosen"}`), nil},
		{"drift feet zero", "FR-049", "unhappy (boundary)", inZone(`{"kind":"movedByCaster","maxFeet":30,"action":"bonusAction","laterTurnsOnly":true}`, `{"kind":"driftsAtTurnStart","feet":0,"direction":"awayFromCaster"}`), []string{z + "mobility/feet outOfRange"}},
		{"stationary and moves with origin", "FR-048", "happy", inZone(`{"kind":"movedByCaster","maxFeet":30,"action":"bonusAction","laterTurnsOnly":true}`, `{"kind":"movesWithOrigin"}`), nil},
		{"designation unknown", "FR-048", "unhappy (invalid input)", inZone(`"atCastingOnly"`, `"always"`), []string{z + "designation invalidValue"}},
		{"zone not an object", "FR-048", "unhappy (invalid input)", spellFx("0", fxB(`[]`)), []string{"/spells/0/effects/zone wrongType"}},
	})
	set := decodeOneSpell(t, inZone(`"damageBudget":20,`, ``)).Effects().(content.ModelledEffects)
	if zone, ok := set.Zone(); !ok || zone.DamageBudget() != 0 || zone.Movement().ExtraFeetPerFoot() != 3 {
		t.Errorf("an absent damageBudget decodes to %d, want 0 (no budget)", zone.DamageBudget())
	}
}

// agree fails unless the decoder and the constructor give the same verdict for
// one row: the document decodes with no problem exactly when the constructor
// returned no error (spell-effect-data.md FR-032, content-spells.md FR-111).
func agree(t *testing.T, label, doc string, ctor error) {
	t.Helper()
	problems := decodeProblems(doc)
	if (len(problems) == 0) != (ctor == nil) {
		t.Errorf("%s: decoder problems = %q, constructor error = %v; the verdicts differ", label, problems, ctor)
	}
}

// Requirement: spell-effect-data.md FR-032 and schema FR-094 - for every rule of
// the effect constructors, the decoder and the constructor accept and reject the
// same values: counts, area dimensions, steps, fixed amounts, chosen sets, condition
// durations, zone numbers, level tables and the half damage rule. The one
// documented divergence is the zero of unitFeet and damageBudget, which the
// constructors read as absent and the decoder rejects when the key is present
// (spell-effect-data.md FR-049).
// Path: per case.
func TestAgreementSpellEffects(t *testing.T) {
	t.Parallel()
	str := itoa
	automatic := func(outcome string) string {
		return `{"resolution":{"kind":"automaticResolution","outcomes":[` + outcome + `]}}`
	}
	healing := func(amount string) string {
		return automatic(`{"kind":"spellRestoresHitPoints","amount":{"base":` + amount + `}}`)
	}
	withPart := func(part string) string { return spellFx("2", fxOf(`{"kind":"targetsSelf"}`, part)) }
	withTarget := func(targeting string) string {
		return spellFx("2", fxOf(targeting, healing(`{"kind":"fixedSpellAmount","value":1}`)))
	}
	for _, n := range []int{-1, 0, 1, 2} {
		agree(t, "exactlyTargets "+str(n), withTarget(`{"kind":"targetsChosen","subject":"creature","count":{"kind":"exactlyTargets","n":`+str(n)+`}}`), errOf(content.NewExactlyTargets(n, nil)))
		agree(t, "upToTargets "+str(n), withTarget(`{"kind":"targetsChosen","subject":"creature","count":{"kind":"upToTargets","n":`+str(n)+`}}`), errOf(content.NewUpToTargets(n, nil)))
		agree(t, "perSlotLevelCount extra "+str(n), withTarget(`{"kind":"targetsChosen","subject":"creature","count":{"kind":"exactlyTargets","n":1,"scaling":{"kind":"perSlotLevelCount","aboveLevel":2,"extra":`+str(n)+`}}}`), errOf(content.NewPerSlotLevelCount(2, n)))
		agree(t, "fixedSpellAmount "+str(n), withPart(healing(`{"kind":"fixedSpellAmount","value":`+str(n)+`}`)), errOf(content.NewFixedSpellAmount(n)))
		agree(t, "instances base "+str(n), withPart(mutate(healing(`{"kind":"fixedSpellAmount","value":1}`), `}}]}}`, `}}]},"instances":{"base":`+str(n)+`}}`)), errOf(content.NewPartInstances(n, nil)))
		agree(t, "timedConditionDuration "+str(n), withPart(automatic(`{"kind":"spellAppliesCondition","condition":"Prone","duration":{"kind":"timedConditionDuration","amount":`+str(n)+`,"unit":"hours"}}`)), errOf(content.NewTimedConditionDuration(n, content.DurationUnitHours)))
		for kind, dims := range map[string][]string{
			`coneArea`: {"lengthFeet"}, `cubeArea`: {"sideFeet"}, `emanationArea`: {"distanceFeet"}, `sphereArea`: {"radiusFeet"},
		} {
			ctor := map[string]error{
				"coneArea": errOf(content.NewConeArea(n)), "cubeArea": errOf(content.NewCubeArea(n)),
				"emanationArea": errOf(content.NewEmanationArea(n)), "sphereArea": errOf(content.NewSphereArea(n)),
			}[kind]
			agree(t, kind+" "+str(n), withTarget(`{"kind":"targetsInArea","subject":"creature","shape":{"kind":"`+kind+`","`+dims[0]+`":`+str(n)+`}}`), ctor)
		}
		agree(t, "cylinderArea radius "+str(n), withTarget(`{"kind":"targetsInArea","subject":"creature","shape":{"kind":"cylinderArea","radiusFeet":`+str(n)+`,"heightFeet":5}}`), errOf(content.NewCylinderArea(n, 5)))
		agree(t, "cylinderArea height "+str(n), withTarget(`{"kind":"targetsInArea","subject":"creature","shape":{"kind":"cylinderArea","radiusFeet":5,"heightFeet":`+str(n)+`}}`), errOf(content.NewCylinderArea(5, n)))
		agree(t, "lineArea length "+str(n), withTarget(`{"kind":"targetsInArea","subject":"creature","shape":{"kind":"lineArea","lengthFeet":`+str(n)+`,"widthFeet":5}}`), errOf(content.NewLineArea(n, 5)))
		agree(t, "lineArea width "+str(n), withTarget(`{"kind":"targetsInArea","subject":"creature","shape":{"kind":"lineArea","lengthFeet":5,"widthFeet":`+str(n)+`}}`), errOf(content.NewLineArea(5, n)))
	}
	for _, s := range [][2]int{{0, 0}, {-1, 0}, {0, -1}, {1, 0}, {0, 1}, {1, 1}} {
		step := `{"kind":"perSlotLevelAmount","aboveLevel":2,"extraDice":` + str(s[0]) + `,"extraFlat":` + str(s[1]) + `}`
		agree(t, "step "+str(s[0])+","+str(s[1]), withPart(healing(`{"kind":"rolledSpellAmount","roll":"1d4"},"scaling":`+step)), errOf(content.NewAmountStep(s[0], s[1])))
	}
	for _, c := range []struct {
		label string
		types []content.DamageType
		json  string
	}{
		{"one", []content.DamageType{content.DamageFire}, `"Fire"`},
		{"two", []content.DamageType{content.DamageFire, content.DamageCold}, `"Fire","Cold"`},
		{"repeated", []content.DamageType{content.DamageFire, content.DamageFire}, `"Fire","Fire"`},
		{"none", nil, ``},
	} {
		agree(t, "chosen types "+c.label, withPart(automatic(`{"kind":"spellDealsDamage","amount":{"base":{"kind":"fixedSpellAmount","value":1}},"damageType":{"kind":"chosenDamageType","types":[`+c.json+`]}}`)), errOf(content.NewChosenDamageType(c.types)))
	}
	for _, c := range []struct {
		label string
		types []content.CreatureType
		json  string
	}{
		{"none", nil, ``}, {"one", []content.CreatureType{content.CreatureTypeHumanoid}, `"Humanoid"`},
		{"repeated", []content.CreatureType{content.CreatureTypeHumanoid, content.CreatureTypeHumanoid}, `"Humanoid","Humanoid"`},
	} {
		agree(t, "creature types "+c.label, withTarget(`{"kind":"targetsChosen","subject":"creature","count":{"kind":"targetsByInstance"},"restriction":{"kind":"onlyCreatureTypes","types":[`+c.json+`]}}`), errOf(content.NewOnlyCreatureTypes(c.types)))
	}
	for _, c := range []struct {
		label   string
		subject string
		want    content.TargetSubject
		wantErr bool
	}{{"creature", "creature", content.SubjectCreature, false}, {"object", "object", content.SubjectObject, true}} {
		humanoid := must(content.NewOnlyCreatureTypes([]content.CreatureType{content.CreatureTypeHumanoid}))
		_, ctor := content.NewTargetsChosen(c.want, must(content.NewExactlyTargets(1, nil)), humanoid, content.MayIncludeCaster)
		agree(t, "restriction on "+c.label, withTarget(`{"kind":"targetsChosen","subject":"`+c.subject+`","count":{"kind":"targetsByInstance"},"restriction":{"kind":"onlyCreatureTypes","types":["Humanoid"]}}`), ctor)
	}
	// Half damage needs damage to halve.
	cond := must(content.NewSpellAppliesCondition(content.ConditionProne, content.ForSpellDuration{}))
	dmg := damageOf(amount(fixed(1), nil), fixedType(content.DamageFire))
	for _, c := range []struct {
		label   string
		out     []content.SpellOutcome
		outJSON string
		success content.SuccessEffect
		name    string
	}{
		{"condition, half", []content.SpellOutcome{cond}, `{"kind":"spellAppliesCondition","condition":"Prone","duration":{"kind":"forSpellDuration"}}`, content.HalfDamageOnSuccess, "halfDamageOnSuccess"},
		{"condition, nothing", []content.SpellOutcome{cond}, `{"kind":"spellAppliesCondition","condition":"Prone","duration":{"kind":"forSpellDuration"}}`, content.NothingOnSuccess, "nothingOnSuccess"},
		{"damage, half", []content.SpellOutcome{dmg}, `{"kind":"spellDealsDamage","amount":{"base":{"kind":"fixedSpellAmount","value":1}},"damageType":{"kind":"fixedDamageType","type":"Fire"}}`, content.HalfDamageOnSuccess, "halfDamageOnSuccess"},
	} {
		agree(t, "save "+c.label, withPart(`{"resolution":{"kind":"savingThrowResolution","save":"dexterity","onFailure":[`+c.outJSON+`],"onSuccess":"`+c.name+`"}}`),
			errOf(content.NewSavingThrowResolution(ability.Dexterity, c.out, c.success)))
	}
	// Tables: first level, order, bounds.
	for _, c := range []struct {
		label  string
		levels []int
	}{{"1", []int{1}}, {"2", []int{2}}, {"1,5,17", []int{1, 5, 17}}, {"1,1", []int{1, 1}}, {"1,5,5", []int{1, 5, 5}}, {"5,1", []int{5, 1}}, {"1,21", []int{1, 21}}, {"0,1", []int{0, 1}}} {
		var entries []content.CountLevelEntry
		var js []string
		for _, l := range c.levels {
			entries = append(entries, content.CountLevelEntry{Level: l, Count: 1})
			js = append(js, `{"level":`+str(l)+`,"count":1}`)
		}
		doc := spellFx("0", fxOf(`{"kind":"targetsSelf"}`, mutate(healing(`{"kind":"fixedSpellAmount","value":1}`), `}}]}}`, `}}]},"instances":{"base":1,"scaling":{"kind":"byCharacterLevelCount","entries":[`+strings.Join(js, ",")+`]}}}`)))
		agree(t, "count table "+c.label, doc, errOf(content.NewCountLevelTable(entries)))
	}
	// Zone numbers.
	zoneOf := func(zone string) string { return spellFx("0", fxB(zone)) }
	rule := `{"trigger":"creatureEnters","partIndex":0,"scope":"anyCreature"}`
	for _, n := range []int{-1, 0, 1} {
		agree(t, "movedByCaster "+str(n), zoneOf(`{"rules":[`+rule+`],"mobility":{"kind":"movedByCaster","maxFeet":`+str(n)+`,"action":"magicAction"}}`), errOf(content.NewMovedByCaster(n, content.MagicAction, false)))
		agree(t, "driftsAtTurnStart "+str(n), zoneOf(`{"rules":[`+rule+`],"mobility":{"kind":"driftsAtTurnStart","feet":`+str(n)+`,"direction":"awayFromCaster"}}`), errOf(content.NewDriftsAtTurnStart(n, content.AwayFromCaster)))
		agree(t, "extraFeetPerFoot "+str(n), zoneOf(`{"rules":[`+rule+`],"movement":{"extraFeetPerFoot":`+str(n)+`}}`), errOf(content.NewZoneMovementFacts(false, false, n)))
	}
	for _, c := range []struct {
		label   string
		trigger string
		fields  content.ZoneRuleFields
		unit    string
	}{
		{"travelsWithin with a unit", "travelsWithin", content.ZoneRuleFields{Trigger: content.TravelsWithin, Scope: content.AnyCreature, UnitFeet: 5}, `,"unitFeet":5`},
		{"travelsWithin without a unit", "travelsWithin", content.ZoneRuleFields{Trigger: content.TravelsWithin, Scope: content.AnyCreature}, ``},
		{"creatureEnters without a unit", "creatureEnters", content.ZoneRuleFields{Trigger: content.CreatureEnters, Scope: content.AnyCreature}, ``},
		{"creatureEnters with a unit", "creatureEnters", content.ZoneRuleFields{Trigger: content.CreatureEnters, Scope: content.AnyCreature, UnitFeet: 5}, `,"unitFeet":5`},
	} {
		doc := zoneOf(`{"rules":[{"trigger":"` + c.trigger + `","partIndex":0,"scope":"anyCreature"` + c.unit + `}]}`)
		agree(t, "zone rule "+c.label, doc, errOf(content.NewZoneRuleFact(c.fields)))
	}
	// The documented divergence: a present zero is outOfRange, an absent key is 0.
	if got := decodeProblems(zoneOf(`{"rules":[` + rule + `],"damageBudget":0}`)); !slices.Equal(got, []string{"/spells/0/effects/zone/damageBudget outOfRange"}) {
		t.Errorf("damageBudget 0 present: %q", got)
	}
	if _, err := content.NewZoneFacts(content.ZoneFactsFields{Rules: []content.ZoneRuleFact{must(content.NewZoneRuleFact(content.ZoneRuleFields{Trigger: content.CreatureEnters, Scope: content.AnyCreature}))}}); err != nil {
		t.Errorf("the constructor rejects a DamageBudget of 0 (absent): %v", err)
	}
}

// Requirement: content-spells.md FR-111 and schema FR-094 - the decoder and the
// constructors of the spell parts give the same verdict for the limits: level,
// casting time amount and triggers, range distance, duration amount, cost
// figure, description, name and the id.
// Path: per case.
func TestAgreementSpellParts(t *testing.T) {
	t.Parallel()
	base := baseSpellFields()
	build := func(edit func(*content.SpellFields)) error {
		f := base
		edit(&f)
		return errOf(content.NewSpell("fixture_spell", "Fixture Spell", content.SourceCitation{}, f))
	}
	for _, n := range []int{-1, 0, 1, 9, 10} {
		doc := spellWith(`"level":2`, `"level":`+istr(n))
		agree(t, "level "+istr(n), doc, errOf(content.NewSpellLevel(n)))
		agree(t, "spell level "+istr(n), doc, build(func(f *content.SpellFields) { f.Level = content.SpellLevel(n) }))
	}
	for _, n := range []int{-1, 0, 1, 2} {
		agree(t, "minutes "+istr(n), spellWith(`{"kind":"magicAction"}`, `{"kind":"minutes","amount":`+istr(n)+`}`), errOf(content.NewCastingTimeMinutes(n)))
		agree(t, "hours "+istr(n), spellWith(`{"kind":"magicAction"}`, `{"kind":"hours","amount":`+istr(n)+`}`), errOf(content.NewCastingTimeHours(n)))
		agree(t, "distance "+istr(n), spellWith(`"amount":60`, `"amount":`+istr(n)), errOf(content.NewSpellRangeDistance(n, content.DistanceFeet)))
		agree(t, "timed "+istr(n), spellWith(`"amount":1,"unit":"minutes"`, `"amount":`+istr(n)+`,"unit":"minutes"`), errOf(content.NewSpellDurationTimed(n, content.DurationUnitMinutes, content.LimitConcentrationUpTo)))
		agree(t, "cost "+istr(n), spellWith(`"costsGp":[]`, `"costsGp":[`+istr(n)+`]`), errOf(content.NewMaterialComponent("a pinch of dust", []int{n}, false)))
	}
	for _, s := range []string{"", " ", "when hit", "a\x00"} {
		agree(t, "reaction trigger "+strings.TrimSpace(strings.ToValidUTF8(s, "?")), spellWith(`{"kind":"magicAction"}`, `{"kind":"reaction","trigger":`+jsonString(s)+`}`), errOf(content.NewCastingTimeReaction(s)))
		agree(t, "mode label "+strings.TrimSpace(s), spellWith(`[{"kind":"magicAction"}]`, `[{"kind":"magicAction","label":"A"},{"kind":"magicAction","label":`+jsonString(s)+`}]`), errOf(content.NewCastingTimeMode(s, content.NewCastingTimeMagicAction())))
		agree(t, "material "+strings.TrimSpace(s), spellWith(`"a pinch of dust"`, jsonString(s)), errOf(content.NewMaterialComponent(s, nil, false)))
	}
	for _, s := range []string{"", " N", "Fixture Spell", "N ", "a\u202eb"} {
		agree(t, "name "+strings.TrimSpace(s), spellWith(`"name":"Fixture Spell"`, `"name":`+jsonString(s)), errOf(content.NewSpell("fixture_spell", s, content.SourceCitation{}, base)))
	}
	for _, id := range []string{"a", "Acid-Arrow", "", "a_b2", "2a"} {
		_, perr := content.ParseSpellID(id)
		agree(t, "id "+id, spellWith(`"id":"fixture_spell"`, `"id":`+jsonString(id)), perr)
	}
	agree(t, "description empty", spellWith(`"description":[{"kind":"paragraph","text":"Text."}]`, `"description":[]`), build(func(f *content.SpellFields) { f.Description = nil }))
	agree(t, "classes repeated", spellWith(`["wizard","fixture_class"]`, `["wizard","wizard"]`), build(func(f *content.SpellFields) { f.Classes = []content.ClassID{"wizard", "wizard"} }))
	agree(t, "higher level empty", spellWith(`"higherLevel":[{"kind":"paragraph","text":"More."}]`, `"higherLevel":[]`), build(func(f *content.SpellFields) { f.HigherLevel = []content.TextBlock{} }))
}

// FuzzDecodeSpellEffects feeds arbitrary JSON as the effects of a spell of any
// level. The decoder never panics, is deterministic, reports typed problems for
// what it rejects (matching ErrInvalidInput, with the zero Set), and a spell it
// accepts holds effects that agree with its level and that its constructor
// takes again.
//
// Requirement: doc/specs/spell-effect-data.md FR-031 and doc/rules/testing.md section 10
// Path: unhappy (invalid input).
func FuzzDecodeSpellEffects(f *testing.F) {
	for _, s := range []string{
		fxA, fxB(""), fxB(fxZone), `{}`, `null`, `[]`, `""`, `1`, `{"coverage":"complete"}`, `{"coverage":"complete","targeting":{"kind":"targetsSelf"},"parts":[]}`,
		fxOf(`{"kind":"targetsSelf"}`, `{}`), fxOf(`{"kind":"targetsSelf"}`, `{"resolution":{"kind":"automaticResolution","outcomes":[null,[],{"kind":1}]}}`),
		fxOf(fxTargeting, `{"resolution":{"kind":"attackRollResolution","attack":"melee","onHit":[{"kind":"spellDealsDamage","amount":{"base":{"kind":"rolledSpellAmount","roll":"99999999999d6"}}}]}}`),
		fxB(`{"rules":[{"trigger":"travelsWithin","partIndex":1e999999999,"scope":"anyCreature","unitFeet":-0}]}`),
		fxB(`{"rules":[{"trigger":"creatureEnters","partIndex":9,"scope":"anyCreature"},{"trigger":"creatureEnters","partIndex":9,"scope":"anyCreature"}],"damageBudget":0,"movement":[]}`),
		strings.Repeat("[", 10000) + strings.Repeat("]", 10000), `{"coverage":"complete","targeting":{"kind":"targetsInArea","shape":{"kind":"coneArea","lengthFeet":2147483648},"subject":"creature"},"parts":[` + strings.Repeat(`{},`, 5000) + `{}]}`,
	} {
		f.Add(byte(2), []byte(s))
		f.Add(byte(0), []byte(s))
	}
	f.Fuzz(func(t *testing.T, level byte, effects []byte) {
		doc := []byte(spellFx(istr(int(level%10)), string(effects)))
		set1, err1 := content.JSONDecoder{}.Decode(doc)
		set2, err2 := content.JSONDecoder{}.Decode(doc)
		if (err1 == nil) != (err2 == nil) || !set1.Equal(set2) || !slices.Equal(content.Problems(err1), content.Problems(err2)) {
			t.Fatal("Decode is not deterministic")
		}
		if err1 != nil {
			if len(content.Problems(err1)) == 0 || !set1.Equal(content.Set{}) {
				t.Fatalf("a rejection must hold problems and the zero Set: %v", err1)
			}
			return
		}
		for _, e := range set1.Entries() {
			s, ok := e.(content.Spell)
			if !ok || s.Effects().CheckSpellLevel(s.Level().Int()) != nil {
				t.Fatalf("accepted effects that disagree with the level: %#v", e)
			}
			again, err := content.NewSpell(s.ID(), s.Name(), s.Source(), content.SpellFields{
				Level: s.Level(), School: s.School(), Classes: s.Classes(), CastingTimes: s.CastingTimes(), Ritual: s.Ritual(), Range: s.Range(),
				Components: s.Components(), Duration: s.Duration(), Description: s.Description(), HigherLevel: s.HigherLevel(),
				CantripUpgrade: s.CantripUpgrade(), Effects: s.Effects(),
			})
			if err != nil || !again.Equal(s) {
				t.Fatalf("a decoded spell is not accepted again: %v", err)
			}
		}
	})
}

func istr(n int) string { return strconv.Itoa(n) }

// Requirement: spell-effect-data.md FR-030, FR-031 - an attackRollResolution
// without onMiss decodes with an empty miss list (onMiss is optional, onHit is
// required).
// Path: happy.
func TestDecodeSpellAttackWithoutOnMiss(t *testing.T) {
	t.Parallel()
	part := `{"resolution":{"kind":"attackRollResolution","attack":"ranged","onHit":[{"kind":"spellRestoresHitPoints","amount":{"base":{"kind":"fixedSpellAmount","value":3}}}]}}`
	got := decodeOneSpell(t, spellFx("2", fxOf(`{"kind":"targetsSelf"}`, part))).Effects().(content.ModelledEffects)
	r, ok := got.Parts()[0].Resolution().(content.AttackRollResolution)
	if !ok || r.Kind() != content.SpellAttackRanged || len(r.OnHit()) != 1 || len(r.OnMiss()) != 0 {
		t.Errorf("resolution = %#v", got.Parts()[0].Resolution())
	}
}
