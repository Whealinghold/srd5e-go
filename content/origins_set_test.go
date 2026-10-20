package content_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// stub is an entry of any key, for the targets of references (no spell type
// exists yet in this step).
type stub struct{ key content.Key }

func (s stub) Key() content.Key { return s.key }

func stubSet(t *testing.T, kind content.Kind, ids ...string) content.Set {
	t.Helper()
	var es []content.Entry
	for _, id := range ids {
		es = append(es, stub{content.NewKey(kind, content.MustID(id))})
	}
	s, err := content.NewSet(es...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func decodeSet(t *testing.T, doc string) content.Set {
	t.Helper()
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode(%s) = %v, want a Set", doc, err)
	}
	return set
}

func speciesWithSpells(t *testing.T) content.Set {
	t.Helper()
	return decodeSet(t, wrapDoc("species", speciesJSON(choiceJSON(lineageOpt("L", 1, 3, 5)))))
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-023, FR-021 - a decoded species lists one site
// per spell id; ValidateSet reports an unresolvedReference naming the owner key, field and target when a spell
// is missing, accepts a complete set, and a single-document decode never resolves a spell.
// Path: unhappy (missing content) and happy.
func TestValidateSetSpeciesSpells(t *testing.T) {
	t.Parallel()
	species := speciesWithSpells(t) // spells spell_0, spell_1, spell_2 (levels 1, 3, 5)
	complete := stubSet(t, content.KindSpell, "spell_0", "spell_1", "spell_2")
	if _, err := content.ValidateSet(species, complete); err != nil {
		t.Errorf("ValidateSet over a complete set = %v, want nil", err)
	}
	missing := stubSet(t, content.KindSpell, "spell_0", "spell_2")
	_, err := content.ValidateSet(species, missing)
	got := content.ReferenceProblems(err)
	want := []content.ReferenceProblem{{
		Owner:  content.NewKey(content.KindSpecies, content.MustID("fixture")),
		Field:  "choices/0/options/0/levelBenefits/1/spellId",
		Target: content.NewKey(content.KindSpell, content.MustID("spell_1")),
		Kind:   content.ProblemUnresolvedReference,
	}}
	if !slices.Equal(got, want) {
		t.Errorf("reference problems = %+v, want %+v", got, want)
	}
	if _, err := content.ValidateSet(species); len(content.ReferenceProblems(err)) != 3 {
		t.Errorf("ValidateSet with no spells = %v, want one unresolvedReference per spell id", err)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-025, FR-072 - a background's feat must name a
// feat of category Origin: an unknown id is unresolvedReference and another category constraintViolation, both
// from ValidateSet, never from a single-document decode.
// Path: per case.
func TestValidateSetBackgroundFeat(t *testing.T) {
	t.Parallel()
	bg := decodeSet(t, wrapDoc("backgrounds", backgroundJSON))
	owner := content.NewKey(content.KindBackground, content.MustID("fixture"))
	target := content.NewKey(content.KindFeat, content.MustID("feat_a"))
	cases := []struct {
		name  string
		feats string
		want  []content.ReferenceProblem
	}{
		{"Origin feat", featJSON, nil},
		{"unresolved feat", mutate(featJSON, `"feat_a"`, `"other"`), []content.ReferenceProblem{{Owner: owner, Field: "feat/featId", Target: target, Kind: content.ProblemUnresolvedReference}}},
		{"General feat", mutate(featJSON, `"Origin"`, `"General"`), []content.ReferenceProblem{{Owner: owner, Field: "feat/featId", Target: target, Kind: content.ProblemConstraintViolation}}},
		{"Fighting Style feat", mutate(featJSON, `"Origin"`, `"Fighting Style"`), []content.ReferenceProblem{{Owner: owner, Field: "feat/featId", Target: target, Kind: content.ProblemConstraintViolation}}},
		{"Epic Boon feat", mutate(featJSON, `"Origin"`, `"Epic Boon"`), []content.ReferenceProblem{{Owner: owner, Field: "feat/featId", Target: target, Kind: content.ProblemConstraintViolation}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			feats := decodeSet(t, wrapDoc("feats", c.feats))
			_, err := content.ValidateSet(bg, feats)
			if got := content.ReferenceProblems(err); !slices.Equal(got, c.want) {
				t.Errorf("reference problems = %+v, want %+v [FR-025]", got, c.want)
			}
		})
	}
	if _, err := content.ValidateSet(bg); len(content.ReferenceProblems(err)) != 1 {
		t.Errorf("ValidateSet over backgrounds alone = %v, want one unresolvedReference [FR-025]", err)
	}
	// A single document holding both arrays decodes with no problem: the site is whole-set.
	_, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion":1,"backgrounds":[` + backgroundJSON + `]}`))
	if err != nil {
		t.Errorf("Decode of backgrounds alone = %v, want no problem (references are not resolved by a decode)", err)
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-056 - a decoded entry is equal by value to the
// same text decoded again, and exposes what the text said, with typed spell ids and a verbatim text.
// Path: happy.
func TestDecodedEntriesAreEqualByValueAndFaithful(t *testing.T) {
	t.Parallel()
	doc := `{"schemaVersion":1,"species":[` + mutate(speciesJSON(choiceJSON(textOpt("A", "spell_a"))), `"text":"x"`, `"text":"It’s kept.\nTwo lines."`) + `],"backgrounds":[` + backgroundJSON + `],"feats":[` + featJSON + `]}`
	a, b := decodeSet(t, doc), decodeSet(t, doc)
	if !a.Equal(b) || len(a.Entries()) != 3 {
		t.Fatalf("two decodes differ or hold %d entries, want 3 equal ones", len(a.Entries()))
	}
	var sp content.Species
	for _, e := range a.Entries() {
		if s, ok := e.(content.Species); ok {
			sp = s
		}
	}
	if sp.Traits()[0].Text() != "It’s kept.\nTwo lines." {
		t.Errorf("trait text = %q, want it kept character for character [schema FR-029]", sp.Traits()[0].Text())
	}
	opt := sp.Choices()[0].Options()[0].(content.TextOption)
	if !slices.Equal(opt.SpellIDs(), []content.SpellID{"spell_a"}) || opt.SpellIDs()[0].Key().Kind() != content.KindSpell {
		t.Errorf("spell ids = %v, want the typed id spell_a", opt.SpellIDs())
	}
}

// Requirement: doc/specs/content-species-backgrounds-feats.md FR-069 and schema FR-094 - for each row the decoder
// and the constructor agree: an invalid row is the stated problem and an error wrapping ErrInvalidInput, a valid
// row is accepted by both.
// Path: per case.
func TestAgreementOrigins(t *testing.T) {
	t.Parallel()
	level := func(n int) content.LevelBenefit { return content.MustLevelBenefit(n, "b", "spell_a") }
	lineageErr := func(levels ...int) error {
		var bs []content.LevelBenefit
		for _, l := range levels {
			bs = append(bs, level(l))
		}
		_, err := content.NewLineageOption("L", bs)
		return err
	}
	speciesErr := func(edit func(*content.SpeciesFields)) error {
		f := baseSpeciesFields(t)
		edit(&f)
		_, err := content.NewSpecies("fixture", "Fixture", content.SourceCitation{}, f)
		return err
	}
	featErr := func(edit func(*content.FeatFields)) error {
		f := baseFeatFields()
		edit(&f)
		_, err := content.NewFeat("feat_a", "Feat", content.SourceCitation{}, f)
		return err
	}
	bgErr := func(edit func(*content.BackgroundFields)) error {
		f := baseBackgroundFields()
		edit(&f)
		_, err := content.NewBackground("fixture", "Fixture", content.SourceCitation{}, f)
		return err
	}
	sp := func(old, repl string) string { return wrapDoc("species", mutate(speciesJSON(""), old, repl)) }
	ft := func(old, repl string) string { return wrapDoc("feats", mutate(featJSON, old, repl)) }
	bg := func(old, repl string) string { return wrapDoc("backgrounds", mutate(backgroundJSON, old, repl)) }
	rows := []struct {
		name, requirement, path string
		doc                     string
		want                    []string // decoder problems; nil accepted
		ctor                    error    // the constructor's verdict on the same row
	}{
		{"sizes: none", "FR-012", "unhappy (boundary)", sp(`["Medium"]`, `[]`), []string{"/species/0/sizes outOfRange"}, speciesErr(func(f *content.SpeciesFields) { f.Sizes = nil })},
		{"sizes: three", "FR-012", "unhappy (boundary)", sp(`["Medium"]`, `["Small","Small","Small"]`), []string{"/species/0/sizes outOfRange"}, speciesErr(func(f *content.SpeciesFields) { f.Sizes = []content.CreatureSize{1, 1, 1} })},
		{"sizes: repeat", "FR-012", "unhappy (boundary)", sp(`["Medium"]`, `["Small","Small"]`), []string{"/species/0/sizes/1 duplicate"}, speciesErr(func(f *content.SpeciesFields) { f.Sizes = []content.CreatureSize{content.SizeSmall, content.SizeSmall} })},
		{"sizes: Large", "FR-012", "unhappy (boundary)", sp(`["Medium"]`, `["Large"]`), []string{"/species/0/sizes/0 invalidValue"}, speciesErr(func(f *content.SpeciesFields) { f.Sizes = []content.CreatureSize{content.SizeLarge} })},
		{"sizes: Small and Medium", "FR-012", "happy", sp(`["Medium"]`, `["Small","Medium"]`), nil, speciesErr(func(f *content.SpeciesFields) {
			f.Sizes = []content.CreatureSize{content.SizeSmall, content.SizeMedium}
		})},
		{"speed: negative", "FR-013", "unhappy (boundary)", sp(`30`, `-1`), []string{"/species/0/speed outOfRange"}, speciesErr(func(f *content.SpeciesFields) { f.Speed = -1 })},
		{"speed: zero", "FR-013", "unhappy (boundary)", sp(`30`, `0`), nil, speciesErr(func(f *content.SpeciesFields) { f.Speed = 0 })},
		{"traits: empty", "FR-014", "unhappy (boundary)", sp(`[{"name":"T","text":"x"},{"name":"U","text":"y"}]`, `[]`), []string{"/species/0/traits outOfRange"}, speciesErr(func(f *content.SpeciesFields) { f.Traits = nil })},
		{"traits: repeated name", "FR-014", "unhappy (invalid input)", sp(`"name":"U"`, `"name":"T"`), []string{"/species/0/traits/1/name duplicate"}, speciesErr(func(f *content.SpeciesFields) {
			f.Traits = []content.NamedText{content.MustNamedText("T", "x"), content.MustNamedText("T", "y")}
		})},
		{"lineage: two levels", "FR-061", "unhappy (boundary)", sp(`"traits"`, `"choices":[{"traitName":"T","options":[`+lineageOpt("L", 1, 3)+`]}],"traits"`), []string{"/species/0/choices/0/options/0/levelBenefits outOfRange"}, lineageErr(1, 3)},
		{"lineage: repeat", "FR-061", "unhappy (boundary)", sp(`"traits"`, `"choices":[{"traitName":"T","options":[`+lineageOpt("L", 1, 3, 3)+`]}],"traits"`), []string{"/species/0/choices/0/options/0/levelBenefits/2/level duplicate"}, lineageErr(1, 3, 3)},
		{"lineage: order", "FR-061", "unhappy (boundary)", sp(`"traits"`, `"choices":[{"traitName":"T","options":[`+lineageOpt("L", 1, 5, 3)+`]}],"traits"`), []string{"/species/0/choices/0/options/0/levelBenefits constraintViolation"}, lineageErr(1, 5, 3)},
		{"lineage: 1, 3, 5", "FR-061", "happy", sp(`"traits"`, `"choices":[{"traitName":"T","options":[`+lineageOpt("L", 1, 3, 5)+`]}],"traits"`), nil, lineageErr(1, 3, 5)},
		{"feat: empty prerequisites", "FR-035", "unhappy (boundary)", ft(`"benefits"`, `"prerequisites":[],"benefits"`), []string{"/feats/0/prerequisites outOfRange"}, featErr(func(f *content.FeatFields) { f.Prerequisites = []content.Prerequisite{} })},
		{"feat: no benefits", "FR-033", "unhappy (boundary)", ft(`[{"text":"x"}]`, `[]`), []string{"/feats/0/benefits outOfRange"}, featErr(func(f *content.FeatFields) { f.Benefits = nil })},
		{"feat: repeatable text without repeatable", "FR-039", "unhappy (rule interaction)", ft(`false`, `false,"repeatableText":"r"`), []string{"/feats/0/repeatableText constraintViolation"}, featErr(func(f *content.FeatFields) { f.RepeatableText = "r" })},
		{"feat: repeatable without text", "FR-039", "unhappy (missing content)", ft(`false`, `true`), []string{"/feats/0/repeatableText missingField"}, featErr(func(f *content.FeatFields) { f.Repeatable = true })},
		{"feat: repeatable with text", "FR-039", "happy", ft(`false`, `true,"repeatableText":"r"`), nil, featErr(func(f *content.FeatFields) { f.Repeatable, f.RepeatableText = true, "r" })},
		{"feat: blank benefit text", "FR-063", "unhappy (invalid input)", ft(`"x"`, `" "`), []string{"/feats/0/benefits/0/text invalidValue"}, featErr(func(f *content.FeatFields) { f.Benefits = []content.FeatBenefit{{}} })},
		{"background: two abilities", "FR-024", "unhappy (boundary)", bg(`["strength","dexterity","constitution"]`, `["strength","dexterity"]`), []string{"/backgrounds/0/abilityScores outOfRange"}, bgErr(func(f *content.BackgroundFields) { f.AbilityScores = f.AbilityScores[:2] })},
		{"background: repeated skill", "FR-027", "unhappy (boundary)", bg(`"Stealth"`, `"Athletics"`), []string{"/backgrounds/0/skillProficiencies/1 duplicate"}, bgErr(func(f *content.BackgroundFields) { f.SkillProficiencies[1] = content.SkillAthletics })},
		{"background: valid", "FR-031", "happy", wrapDoc("backgrounds", backgroundJSON), nil, bgErr(func(*content.BackgroundFields) {})},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			if got := decodeProblems(r.doc); !slices.Equal(got, r.want) {
				t.Errorf("decoder problems = %q, want %q [%s, %s]", got, r.want, r.requirement, r.path)
			}
			if accepted := r.want == nil; (r.ctor == nil) != accepted || (r.ctor != nil && !isInvalidInput(r.ctor)) {
				t.Errorf("constructor error = %v, want accepted=%v and ErrInvalidInput otherwise [%s, %s]", r.ctor, accepted, r.requirement, r.path)
			}
		})
	}
}

func isInvalidInput(err error) bool { return err != nil && errors.Is(err, srderr.ErrInvalidInput) }

// Requirement: doc/specs/content-species-backgrounds-feats.md section 7 and schema FR-079 to FR-083, FR-101 - the
// Loader over the real decoder counts one entry of each of the three kinds, and counts the problems of a
// rejected document by problem kind, with no new signal.
// Path: happy and unhappy (invalid input).
func TestLoaderCountsSpeciesBackgroundsFeats(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	loader := mustLoader(t, content.JSONDecoder{}, content.WithTelemetry(rec.Telemetry()))
	doc := `{"schemaVersion":1,"species":[` + speciesJSON("") + `],"backgrounds":[` + backgroundJSON + `],"feats":[` + featJSON + `]}`
	res, err := loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(doc)})
	if _, ok := res.(content.LoadSetLoaded); !ok || err != nil {
		t.Fatalf("LoadSet = (%T, %v), want LoadSetLoaded", res, err)
	}
	var kinds []string
	for _, m := range rec.Meter.MeasurementsNamed(content.MetricEntriesLoaded) {
		kinds = append(kinds, attrsOf(m.Attrs)[content.AttrKind].(string))
	}
	if want := []string{content.ValueKindSpecies, content.ValueKindBackground, content.ValueKindFeat}; !slices.Equal(kinds, want) {
		t.Errorf("entries_loaded kinds = %v, want %v", kinds, want)
	}
	bad := wrapDoc("feats", mutate(featJSON, `"Origin"`, `"origin"`))
	res, err = loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(bad)})
	if rej, ok := res.(content.LoadSetRejected); !ok || err != nil || len(rej.Problems()) != 1 {
		t.Fatalf("LoadSet(bad) = (%T, %v), want one rejected problem", res, err)
	}
	var counted []string
	for _, m := range rec.Meter.MeasurementsNamed(content.MetricDecodeErrors) {
		counted = append(counted, attrsOf(m.Attrs)[content.AttrProblemKind].(string))
	}
	if want := []string{content.ValueProblemInvalidValue}; !slices.Equal(counted, want) {
		t.Errorf("decode_errors kinds = %v, want %v", counted, want)
	}
}
