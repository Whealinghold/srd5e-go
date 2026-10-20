package content_test

import (
	"strconv"
	"strings"
	"testing"
)

// The anchors of the feature entries of the fixture, each unique in it.
const (
	alphaTail = `"name":"Name fixture_alpha","level":1,"text":"Text.\nMore."}`
	betaTail  = `"name":"Name fixture_beta","level":1,"text":"Text.\nMore."}`
	gammaTail = `"name":"Name fixture_gamma","level":2,"text":"Text.\nMore."}`
	eps       = `"name":"Name fixture_epsilon","level":5,"text":"Text.\nMore."}`
)

func withFacts(tail, facts string) string {
	return strings.TrimSuffix(tail, "}") + `,"facts":[` + facts + `]}`
}

func spellFact(grants string) string {
	return `{"kind":"alwaysPreparedSpells","grants":[` + grants + `]}`
}

// Requirement: per case
// Path: per case.
func TestDecodeClassFeatures(t *testing.T) {
	t.Parallel()
	w := wizardClassJSON()
	f := func(old, repl string) string { return classDoc(mutate(w, old, repl)) }
	fp := cp + "features/"
	runDocCases(t, []docCase{
		{"level 0", classSpec + " FR-051", "unhappy (boundary)", f(`"level":1,"text"`, `"level":0,"text"`), []string{fp + "0/level outOfRange"}},
		{"level 21", classSpec + " FR-051", "unhappy (boundary)", f(`"level":1,"text"`, `"level":21,"text"`), []string{fp + "0/level outOfRange"}},
		{"missing text", classSpec + " FR-051", "unhappy (missing content)", f(`,"text":"Text.\nMore."}`, `}`), []string{fp + "0/text missingField"}},
		{"blank text", classSpec + " FR-054", "unhappy (invalid input)", f(`"text":"Text.\nMore."}`, `"text":" "}`), []string{fp + "0/text invalidValue"}},
		{"unknown feature key", classSpec + " FR-051", "unhappy (invalid input)", f(`"text":"Text.\nMore."}`, `"text":"x","more":1}`), []string{fp + "0/more unknownField"}},
		{"wrong id prefix", classSpec + " FR-052", "unhappy (invalid input)", classDoc(strings.ReplaceAll(w, "fixture_alpha", "other_alpha")), []string{fp + "0/id constraintViolation"}},
		{"repeated feature id keeps the first for rows", classSpec + " FR-044", "unhappy (boundary)", classDoc(mutate(w, featureJSON("fixture_theta", 20, ""), featureJSON("fixture_theta", 20, "")+","+featureJSON("fixture_gamma", 20, ""))), []string{fp + "10/level constraintViolation", fp + "10/id duplicate"}},
		{"id prefix without separator", classSpec + " FR-052", "unhappy (boundary)", classDoc(strings.ReplaceAll(w, "fixture_alpha", "fixturealpha")), []string{fp + "0/id constraintViolation"}},
		{"unknown fact kind", classSpec + " FR-055", "unhappy (invalid input)", f(`"grantsFeat"`, `"grantsAll"`), []string{fp + "4/facts/0/kind invalidValue"}},
		{"duplicate fact", classSpec + " FR-055", "unhappy (boundary)", f(`[{"kind":"grantsFeat","category":"General"}]`, `[{"kind":"grantsFeat","category":"General"},{"kind":"grantsFeat","category":"General"}]`), []string{fp + "4/facts/1 duplicate"}},
		{"two different feat facts", classSpec + " FR-055", "happy", f(`[{"kind":"grantsFeat","category":"General"}]`, `[{"kind":"grantsFeat","category":"General"},{"kind":"grantsFeat","category":"Origin"}]`), nil},
		{"unknown category", classSpec + " FR-057", "unhappy (invalid input)", f(`"category":"General"`, `"category":"Epic"`), []string{fp + "4/facts/0/category invalidValue"}},
		{"two subclass facts", classSpec + " FR-056", "unhappy (rule interaction)", f(eps, withFacts(eps, `{"kind":"grantsSubclass"}`)), []string{cp + "features constraintViolation"}},
		{"no subclass fact", classSpec + " FR-056", "unhappy (missing content)", f(`,"facts":[{"kind":"grantsSubclass"}]`, ``), []string{cp + "features constraintViolation"}},
		{"subclass fact on a subclass feature", classSpec + " FR-056", "unhappy (rule interaction)", classDoc(mutate(ownerClassJSON("fixture", subclassJSON("sub_a", 3, 6)), `"id":"sub_a_f1","name":"Name sub_a_f1","level":6,"text":"Text.\nMore."}`, `"id":"sub_a_f1","name":"Name sub_a_f1","level":6,"text":"Text.\nMore.","facts":[{"kind":"grantsSubclass"}]}`)), []string{cp + "subclasses/0/features/1/facts/0 constraintViolation"}},
		{"spell fact", classSpec + " FR-058", "happy", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":1,"spellIds":["spell_a","spell_b"]},{"level":5,"spellIds":["spell_c"]}`))), nil},
		{"grants out of order", classSpec + " FR-058", "unhappy (boundary)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":5,"spellIds":["a"]},{"level":3,"spellIds":["b"]}`))), []string{fp + "0/facts/0/grants/1/level invalidValue"}},
		{"grants at one level twice", classSpec + " FR-058", "unhappy (boundary)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":5,"spellIds":["a"]},{"level":5,"spellIds":["b"]}`))), []string{fp + "0/facts/0/grants/1/level invalidValue"}},
		{"grant below the entry", classSpec + " FR-058", "unhappy (boundary)", f(gammaTail, withFacts(gammaTail, spellFact(`{"level":1,"spellIds":["a"]}`))), []string{fp + "2/facts/0/grants/0/level invalidValue"}},
		{"entry problems wait for field problems", classSpec + " FR-005", "unhappy (rule interaction)", f(gammaTail, `"name":"Name fixture_gamma","level":2,"text":" ","facts":[`+spellFact(`{"level":1,"spellIds":["a"]}`)+`]}`), []string{fp + "2/text invalidValue"}},
		{"grant without a level", classSpec + " FR-058", "unhappy (missing content)", f(alphaTail, withFacts(alphaTail, spellFact(`{"spellIds":["a"]}`))), []string{fp + "0/facts/0/grants/0/level missingField"}},
		{"grant at the entry level", classSpec + " FR-058", "unhappy (boundary)", f(gammaTail, withFacts(gammaTail, spellFact(`{"level":2,"spellIds":["a"]}`))), nil},
		{"empty spell ids", classSpec + " FR-058", "unhappy (empty case)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":1,"spellIds":[]}`))), []string{fp + "0/facts/0/grants/0/spellIds outOfRange"}},
		{"repeated spell id", classSpec + " FR-058", "unhappy (boundary)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":1,"spellIds":["a","a"]}`))), []string{fp + "0/facts/0/grants/0/spellIds/1 duplicate"}},
		{"invalid spell id", classSpec + " FR-058", "unhappy (invalid input)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":1,"spellIds":["Bad"]}`))), []string{fp + "0/facts/0/grants/0/spellIds/0 invalidValue"}},
		{"no grants", classSpec + " FR-058", "unhappy (empty case)", f(alphaTail, withFacts(alphaTail, spellFact(``))), []string{fp + "0/facts/0/grants outOfRange"}},
		{"grant level 21", classSpec + " FR-058", "unhappy (boundary)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":21,"spellIds":["a"]}`))), []string{fp + "0/facts/0/grants/0/level outOfRange"}},
		{"grant level 20", classSpec + " FR-058", "unhappy (boundary)", f(alphaTail, withFacts(alphaTail, spellFact(`{"level":20,"spellIds":["a"]}`))), nil},
	})
}

func withOptions(tail, options string) string {
	return strings.TrimSuffix(tail, "}") + `,"options":[` + options + `]}`
}

func option(id, extra string) string {
	return `{"id":"` + id + `","name":"Opt ` + id + `","text":"Option text."` + extra + `}`
}

// Requirement: per case
// Path: per case.
func TestDecodeClassOptionsAndTables(t *testing.T) {
	t.Parallel()
	w := wizardClassJSON()
	o := func(options ...string) string {
		return classDoc(mutate(w, betaTail, withOptions(betaTail, strings.Join(options, ","))))
	}
	t2 := func(tables string) string {
		return classDoc(mutate(w, gammaTail, strings.TrimSuffix(gammaTail, "}")+`,"tables":[`+tables+`]}`))
	}
	op := cp + "features/1/options/"
	tp := cp + "features/2/tables/0/"
	runDocCases(t, []docCase{
		{"options", classSpec + " FR-061", "happy", o(option("a", ""), option("b", `,"pointCost":2,"repeatable":true`)), nil},
		{"empty option list", classSpec + " FR-061", "unhappy (empty case)", o(), nil},
		{"duplicate option id", classSpec + " FR-061", "unhappy (invalid input)", o(option("a", ""), option("a", "")), []string{op + "1/id duplicate"}},
		{"blank option text", classSpec + " FR-061", "unhappy (invalid input)", o(`{"id":"a","name":"A","text":" "}`), []string{op + "0/text invalidValue"}},
		{"empty prerequisite", classSpec + " FR-062", "unhappy (empty case)", o(option("a", `,"prerequisite":{}`)), []string{op + "0/prerequisite constraintViolation"}},
		{"prerequisite parts", classSpec + " FR-062", "happy", o(option("a", ""), option("b", `,"prerequisite":{"minClassLevel":5,"requiresOptionIds":["a"],"note":"A cantrip."}`)), nil},
		{"prerequisite level 0", classSpec + " FR-062", "unhappy (boundary)", o(option("a", `,"prerequisite":{"minClassLevel":0}`)), []string{op + "0/prerequisite/minClassLevel outOfRange"}},
		{"prerequisite level 21", classSpec + " FR-062", "unhappy (boundary)", o(option("a", `,"prerequisite":{"minClassLevel":21}`)), []string{op + "0/prerequisite/minClassLevel outOfRange"}},
		{"prerequisite level 20", classSpec + " FR-062", "unhappy (boundary)", o(option("a", `,"prerequisite":{"minClassLevel":20}`)), nil},
		{"prerequisite on a missing option", classSpec + " FR-062", "unhappy (missing content)", o(option("a", `,"prerequisite":{"requiresOptionIds":["zzz"]}`)), []string{op + "0/prerequisite/requiresOptionIds/0 unresolvedReference"}},
		{"self prerequisite", classSpec + " FR-062", "unhappy (rule interaction)", o(option("a", `,"prerequisite":{"requiresOptionIds":["a"]}`)), []string{op + "0/prerequisite/requiresOptionIds/0 constraintViolation"}},
		{"prerequisite cycle", classSpec + " FR-062", "unhappy (rule interaction)", o(option("a", `,"prerequisite":{"requiresOptionIds":["b"]}`), option("b", `,"prerequisite":{"requiresOptionIds":["a"]}`)), []string{op + "0/prerequisite/requiresOptionIds/0 constraintViolation", op + "1/prerequisite/requiresOptionIds/0 constraintViolation"}},
		{"prerequisite chain", classSpec + " FR-062", "happy", o(option("a", `,"prerequisite":{"requiresOptionIds":["b"]}`), option("b", `,"prerequisite":{"requiresOptionIds":["c"]}`), option("c", "")), nil},
		{"longer cycle", classSpec + " FR-062", "unhappy (rule interaction)", o(option("a", `,"prerequisite":{"requiresOptionIds":["b"]}`), option("b", `,"prerequisite":{"requiresOptionIds":["c"]}`), option("c", `,"prerequisite":{"requiresOptionIds":["a"]}`)), []string{op + "0/prerequisite/requiresOptionIds/0 constraintViolation", op + "1/prerequisite/requiresOptionIds/0 constraintViolation", op + "2/prerequisite/requiresOptionIds/0 constraintViolation"}},
		{"cycle that does not reach the first option", classSpec + " FR-062", "unhappy (rule interaction)", o(option("a", `,"prerequisite":{"requiresOptionIds":["b"]}`), option("b", `,"prerequisite":{"requiresOptionIds":["c"]}`), option("c", `,"prerequisite":{"requiresOptionIds":["b"]}`)), []string{op + "1/prerequisite/requiresOptionIds/0 constraintViolation", op + "2/prerequisite/requiresOptionIds/0 constraintViolation"}},
		{"chain of earlier options", classSpec + " FR-062", "happy", o(option("a", ""), option("b", `,"prerequisite":{"requiresOptionIds":["a"]}`), option("c", `,"prerequisite":{"requiresOptionIds":["b"]}`)), nil},
		{"repeated required id", classSpec + " FR-062", "unhappy (boundary)", o(option("a", ""), option("b", `,"prerequisite":{"requiresOptionIds":["a","a"]}`)), []string{op + "1/prerequisite/requiresOptionIds/1 duplicate"}},
		{"empty required ids", classSpec + " FR-062", "unhappy (empty case)", o(option("a", `,"prerequisite":{"requiresOptionIds":[]}`)), []string{op + "0/prerequisite/requiresOptionIds outOfRange"}},
		{"zero point cost", classSpec + " FR-063", "unhappy (boundary)", o(option("a", `,"pointCost":0`)), []string{op + "0/pointCost outOfRange"}},
		{"point cost 1", classSpec + " FR-063", "unhappy (boundary)", o(option("a", `,"pointCost":1`)), nil},
		{"false repeatable", classSpec + " FR-063", "unhappy (invalid input)", o(option("a", `,"repeatable":false`)), []string{op + "0/repeatable invalidValue"}},
		{"repeatable as a string", classSpec + " FR-063", "unhappy (invalid input)", o(option("a", `,"repeatable":"yes"`)), []string{op + "0/repeatable wrongType"}},
		{"subclass fact on an option", classSpec + " FR-064", "unhappy (invalid input)", o(option("a", `,"facts":[{"kind":"grantsSubclass"}]`)), []string{op + "0/facts/0 constraintViolation"}},
		{"spell fact on an option", classSpec + " FR-064", "happy", o(option("a", `,"facts":[`+spellFact(`{"level":1,"spellIds":["a"]}`)+`]`)), nil},
		{"option spell grant below the feature", classSpec + " FR-058", "unhappy (boundary)", classDoc(mutate(w, gammaTail, withOptions(gammaTail, option("a", `,"facts":[`+spellFact(`{"level":1,"spellIds":["a"]}`)+`]`)))), []string{cp + "features/2/options/0/facts/0/grants/0/level invalidValue"}},
		{"table", classSpec + " FR-066", "happy", t2(`{"name":"T","columns":["A","B"],"rows":[["x",1],["y",2]]}`), nil},
		{"ragged table", classSpec + " FR-066", "unhappy (boundary)", t2(`{"name":"T","columns":["A","B"],"rows":[["x",1],["y"]]}`), []string{tp + "rows/1 outOfRange"}},
		{"mixed column", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":["A","B"],"rows":[["x",1],[2,2]]}`), []string{tp + "rows/1/0 wrongType"}},
		{"table with no rows", classSpec + " FR-066", "unhappy (empty case)", t2(`{"name":"T","columns":["A"],"rows":[]}`), []string{tp + "rows outOfRange"}},
		{"table with no columns", classSpec + " FR-066", "unhappy (empty case)", t2(`{"name":"T","columns":[],"rows":[]}`), []string{tp + "columns outOfRange", tp + "rows outOfRange"}},
		{"table cell of another type", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":["A"],"rows":[[true]]}`), []string{tp + "rows/0/0 wrongType"}},
		{"table cell fraction", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":["A"],"rows":[[1.5]]}`), []string{tp + "rows/0/0 wrongType"}},
		{"table blank header", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":[" "],"rows":[["x"]]}`), []string{tp + "columns/0 invalidValue"}},
		{"table blank cell", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":["A"],"rows":[[" "]]}`), []string{tp + "rows/0/0 invalidValue"}},
		{"table first cell bad then others", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":["A","B"],"rows":[[true,"x"],["y",1]]}`), []string{tp + "rows/0/0 wrongType", tp + "rows/1/1 wrongType"}},
		{"table cells with unknown header", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":[" "],"rows":[["x","y"]]}`), []string{tp + "columns/0 invalidValue"}},
		{"table row not an array", classSpec + " FR-066", "unhappy (invalid input)", t2(`{"name":"T","columns":["A"],"rows":[3]}`), []string{tp + "rows/0 wrongType"}},
	})
}

// repeatFeatureIDs lists the duplicate problem of every feature of the second
// of two identical classes.
func repeatFeatureIDs() []string {
	var out []string
	for i := range fixtureFeatures() {
		out = append(out, "/classes/1/features/"+strconv.Itoa(i)+"/id duplicate")
	}
	return out
}

// Requirement: per case
// Path: per case.
func TestDecodeClassSubclassesAndIDs(t *testing.T) {
	t.Parallel()
	own := func(subs string) string { return classDoc(ownerClassJSON("fixture", subs)) }
	sp := cp + "subclasses/"
	noFact := strings.Replace(ownerClassJSON("fixture", subclassJSON("sub_a", 3)), `,"facts":[{"kind":"grantsSubclass"}]`, ``, 1)
	emptyFeatures := `{"id":"sub_a","name":"Sub","features":[]}`
	twoClasses := `{"schemaVersion":1,"classes":[` + ownerClassJSON("fixture", subclassJSON("shared", 3)) + `,` + ownerClassJSON("fixture2", subclassJSON("shared", 3)) + `]}`
	sameClass := `{"schemaVersion":1,"classes":[` + wizardClassJSON() + `,` + wizardClassJSON() + `]}`
	featureClash := strings.ReplaceAll(ownerClassJSON("fixture", subclassJSON("fixture_sub", 3)), "fixture_theta", "fixture_sub_f0")
	bothSpaces := ownerClassJSON("fixture", subclassJSON("fixture_alpha", 3))
	badIDs := mutate(ownerClassJSON("fixture", subclassJSON("sub_a", 3)), `"id":"sub_a_f0"`, `"id":"Bad"`)
	runDocCases(t, []docCase{
		{"subclass", classSpec + " FR-070", "happy", own(subclassJSON("sub_a", 3, 6)), nil},
		{"two subclasses", classSpec + " FR-070", "happy", own(subclassJSON("sub_a", 3) + "," + subclassJSON("sub_b", 3, 10)), nil},
		{"no subclasses", classSpec + " FR-070", "unhappy (empty case)", own(""), nil},
		{"empty feature list", classSpec + " FR-070", "unhappy (empty case)", own(emptyFeatures), []string{sp + "0/features outOfRange"}},
		{"subclass feature below the subclass level", classSpec + " FR-071", "unhappy (boundary)", own(subclassJSON("sub_a", 2, 3)), []string{sp + "0/features/0 constraintViolation"}},
		{"two invalid subclass ids", classSpec + " FR-004", "unhappy (boundary)", own(`{"id":"Bad","name":"S","features":[` + featureJSON("sub_a_f0", 3, "") + `]},{"id":"Bad","name":"S","features":[` + featureJSON("sub_b_f0", 3, "") + `]}`), []string{sp + "0/id invalidValue", sp + "1/id invalidValue"}},
		{"no subclass feature at the subclass level", classSpec + " FR-071", "unhappy (boundary)", own(subclassJSON("sub_a", 6)), []string{sp + "0/features constraintViolation"}},
		{"subclass feature at level 20", classSpec + " FR-071", "unhappy (boundary)", own(subclassJSON("sub_a", 3, 20)), nil},
		{"subclass feature at level 21", classSpec + " FR-051", "unhappy (boundary)", own(subclassJSON("sub_a", 3, 21)), []string{sp + "0/features/1/level outOfRange"}},
		{"subclass features only below the subclass level", classSpec + " FR-071", "unhappy (boundary)", own(subclassJSON("sub_a", 2)), []string{sp + "0/features/0 constraintViolation", sp + "0/features constraintViolation"}},
		{"subclass fact as a later fact", classSpec + " FR-056", "unhappy (rule interaction)", own(mutate(subclassJSON("sub_a", 3), `"text":"Text.\nMore."}`, `"text":"Text.\nMore.","facts":[{"kind":"grantsFeat","category":"General"},{"kind":"grantsSubclass"}]}`)), []string{sp + "0/features/0/facts/1 constraintViolation"}},
		{"two subclass facts with a subclass", classSpec + " FR-073", "unhappy (rule interaction)", classDoc(mutate(ownerClassJSON("fixture", subclassJSON("sub_a", 3)), eps, withFacts(eps, `{"kind":"grantsSubclass"}`))), []string{cp + "features constraintViolation", cp + "subclasses constraintViolation"}},
		{"subclass feature prefix", classSpec + " FR-072", "unhappy (invalid input)", own(mutate(subclassJSON("sub_a", 3), `"id":"sub_a_f0"`, `"id":"other_f0"`)), []string{sp + "0/features/0/id constraintViolation"}},
		{"subclass feature prefix without separator", classSpec + " FR-072", "unhappy (boundary)", own(mutate(subclassJSON("sub_a", 3), `"id":"sub_a_f0"`, `"id":"sub_af0"`)), []string{sp + "0/features/0/id constraintViolation"}},
		{"subclasses with no subclass fact", classSpec + " FR-073", "unhappy (rule interaction)", classDoc(noFact), []string{cp + "features constraintViolation", cp + "subclasses constraintViolation"}},
		{"subclass missing name", classSpec + " FR-070", "unhappy (missing content)", own(`{"id":"sub_a","features":[` + featureJSON("sub_a_f0", 3, "") + `]}`), []string{sp + "0/name missingField"}},
		{"subclass source", classSpec + " FR-009", "happy", own(`{"id":"sub_a","name":"S","source":{"document":"D","section":"S","page":4},"features":[` + featureJSON("sub_a_f0", 3, "") + `]}`), nil},
		{"duplicate class id", classSpec + " FR-004", "unhappy (invalid input)", sameClass, append([]string{"/classes/1/id duplicate"}, repeatFeatureIDs()...)},
		{"duplicate subclass id across classes", classSpec + " FR-004", "unhappy (invalid input)", twoClasses, []string{"/classes/1/subclasses/0/id duplicate", "/classes/1/subclasses/0/features/0/id duplicate"}},
		{"feature id clash with a subclass feature", classSpec + " FR-004", "unhappy (invalid input)", classDoc(featureClash), []string{sp + "0/features/0/id duplicate"}},
		{"same string in two id spaces", classSpec + " FR-004", "unhappy (boundary)", classDoc(bothSpaces), nil},
		{"invalid id takes no part", classSpec + " FR-004", "unhappy (boundary)", classDoc(badIDs), []string{sp + "0/features/0/id invalidValue"}},
	})
	// A document with problems in several parts lists them in field order,
	// unknown keys last.
	doc := classDoc(strings.Replace(mutate(mutate(mutate(wizardClassJSON(), `"D6"`, `"d8"`), `"armorTraining":[]`, `"armorTraining":["Light","Light"]`), `"level":1,"text"`, `"level":0,"text"`), `"name":"Fixture",`, `"name":"Fixture","zz":1,`, 1))
	want := []string{cp + "hitPointDie invalidValue", cp + "armorTraining/1 duplicate", cp + "features/0/level outOfRange", cp + "zz unknownField"}
	runDocCases(t, []docCase{{"several parts", classSpec + " FR-005", "unhappy (rule interaction)", doc, want}, {"several parts again", classSpec + " FR-005", "unhappy (rule interaction)", doc, want}})
}
