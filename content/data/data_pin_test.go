package data_test

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// embeddedDocuments returns the embedded documents by file name.
func embeddedDocuments(t *testing.T) map[string]string {
	t.Helper()
	names, err := fs.Glob(data.Files(), "*")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, n := range names {
		b, err := fs.ReadFile(data.Files(), n)
		if err != nil {
			t.Fatal(err)
		}
		out[n] = string(b)
	}
	return out
}

// Requirement: doc/specs/content-thin-slice.md FR-067 and doc/plans/plan-phase-2.md 5.6 - the embedded file
// list is exactly the three slice documents, in lexical order, so a well-formed extra file in content/data fails.
// Path: unhappy (boundary: nothing beyond the slice).
func TestFilesAreExactlyTheSliceDocuments(t *testing.T) {
	t.Parallel()
	entries, err := fs.ReadDir(data.Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if want := []string{"class_wizard.json", "species.json", "spells.json"}; !slices.Equal(names, want) {
		t.Errorf("Files() = %v, want %v", names, want)
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-067 - each embedded document decodes with zero problems into 4, 1 and
// 12 entries, the source holds exactly 17 entries, and whole-set validation passes over the merged documents.
// Path: happy.
func TestEachDocumentDecodesAndTheSetValidates(t *testing.T) {
	t.Parallel()
	want := map[string]int{"species.json": 4, "class_wizard.json": 1, "spells.json": 12}
	var sets []content.Set
	for name, text := range embeddedDocuments(t) {
		set, err := content.JSONDecoder{}.Decode([]byte(text))
		if err != nil {
			t.Errorf("Decode(%s) problems = %v, want none", name, content.Problems(err))
			continue
		}
		if n := len(set.Entries()); n != want[name] {
			t.Errorf("Decode(%s) = %d entries, want %d", name, n, want[name])
		}
		sets = append(sets, set)
	}
	merged, err := content.ValidateSet(sets...)
	if err != nil {
		t.Fatalf("ValidateSet problems = %v", content.ReferenceProblems(err))
	}
	if n := len(merged.Entries()); n != 17 {
		t.Errorf("merged set = %d entries, want 17", n)
	}
	src := newSlice(t)
	total := 0
	for k := content.KindSpecies; k.Valid(); k++ {
		got, err := src.ListKind(context.Background(), k)
		if err != nil {
			t.Fatal(err)
		}
		total += len(got)
	}
	if total != 17 {
		t.Errorf("BundledSource holds %d entries, want 17", total)
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-001, FR-002, FR-003, FR-004, FR-005, FR-006 - the slice holds
// exactly the listed ids of species, characterClass and spell, each found by its key, and nothing of any other kind.
// Path: happy / unhappy (boundary: none missing, none extra; empty case for other kinds).
func TestMembership(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	ctx := context.Background()
	for k := content.KindSpecies; k.Valid(); k++ {
		t.Run(k.String(), func(t *testing.T) {
			t.Parallel()
			got, err := src.ListKind(ctx, k)
			if err != nil {
				t.Fatal(err)
			}
			if want := sliceIDs()[k]; !slices.Equal(idsOf(got), want) { // nil want for the kinds with no entry
				t.Errorf("ListKind(%s) ids = %v, want %v", k, idsOf(got), want)
			}
			for _, id := range sliceIDs()[k] {
				if e := lookup(t, src, k, id); e.Key() != content.NewKey(k, content.MustID(id)) {
					t.Errorf("Lookup(%s %s) key = %v", k, id, e.Key())
				}
			}
		})
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-002, FR-064, FR-065, FR-066 - SRD entries outside the slice and ids
// under another kind are LookupNotFound carrying the key, with a nil error.
// Path: unhappy (missing content; rule interaction: kind mismatch).
func TestLookupOutsideTheSlice(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	for _, c := range []struct {
		kind content.Kind
		id   string
	}{
		{content.KindSpell, "light"}, {content.KindSpell, "ray_of_frost"}, {content.KindSpecies, "gnome"},
		{content.KindCharacterClass, "evoker"}, {content.KindCharacterClass, "fighter"},
		{content.KindSpecies, "fireball"}, {content.KindSpell, "wizard"},
	} {
		key := content.NewKey(c.kind, content.MustID(c.id))
		res, err := src.Lookup(context.Background(), key)
		if nf, ok := res.(content.LookupNotFound); err != nil || !ok || nf.Key != key {
			t.Errorf("Lookup(%s %s) = (%v, %v), want LookupNotFound with the key", c.kind, c.id, res, err)
		}
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-007 - every entry carries the source citation of the section 2 table.
// Path: happy.
func TestCitations(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	kinds := map[string]content.Kind{"species": content.KindSpecies, "characterClass": content.KindCharacterClass, "spell": content.KindSpell}
	o := readOracle[citationOracle](t, "citations")
	if len(o.Entries) != 17 {
		t.Fatalf("oracle has %d citations, want 17", len(o.Entries))
	}
	for _, c := range o.Entries {
		cite := citationOf(t, lookup(t, src, kinds[c.Kind], c.ID))
		if cite.Document() != o.Document || cite.Section() != c.Section || cite.Page() != c.Page || !cite.CitesSRD521() {
			t.Errorf("%s %s source = %q, %q, %d; want %q, %q, %d", c.Kind, c.ID, cite.Document(), cite.Section(), cite.Page(), o.Document, c.Section, c.Page)
		}
	}
}

func citationOf(t *testing.T, e content.Entry) content.SourceCitation {
	t.Helper()
	switch v := e.(type) {
	case content.Species:
		return v.Source()
	case content.Spell:
		return v.Source()
	case content.ClassDefinition:
		return v.Source()
	}
	t.Fatalf("unexpected entry type %T", e)
	return content.SourceCitation{}
}

// Requirement: doc/specs/content-thin-slice.md FR-008 - no entry text contains a banned whole word; "trace" and "grace"
// never match (self-test of the scan).
// Path: unhappy (invalid input: non-SRD terms).
func TestNoBannedTerms(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"Pass without Trace", "by grace", "a race of", "Subraces"} {
		want := s == "a race of" || s == "Subraces"
		got := false
		for _, term := range bannedTerms() {
			got = got || containsWord(s, term)
		}
		if got != want {
			t.Errorf("banned-term scan of %q = %v, want %v", s, got, want)
		}
	}
	src := newSlice(t)
	var texts []string
	for _, id := range sliceIDs()[content.KindSpecies] {
		texts = append(texts, speciesTexts(speciesOf(t, src, id))...)
	}
	for _, id := range sliceIDs()[content.KindSpell] {
		texts = append(texts, spellTexts(t, spellOf(t, src, id))...)
	}
	texts = append(texts, wizardTexts(wizardOf(t, src))...)
	for _, s := range texts {
		for _, term := range bannedTerms() {
			if containsWord(s, term) {
				t.Errorf("text contains the banned term %q: %.60q", term, s)
			}
		}
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-063, FR-084 - the slice is closed under its references: every
// registered reference site is collected from the decoded entries (the nine lineage spellIds), each names a bundled
// spell and equals the T3 spells, and ValidateSet reports no problem.
// Path: unhappy (dangling id: none permitted).
func TestSliceIsClosedUnderItsReferences(t *testing.T) {
	t.Parallel()
	src := newSlice(t)
	spells := map[content.Key]bool{}
	for _, id := range sliceIDs()[content.KindSpell] {
		spells[content.NewKey(content.KindSpell, content.MustID(id))] = true
	}
	var found []string
	for k := content.KindSpecies; k.Valid(); k++ {
		entries, err := src.ListKind(context.Background(), k)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			holder, ok := e.(content.References)
			if !ok {
				continue
			}
			for _, r := range holder.References() {
				if r.Target().Kind() != content.KindSpell || !spells[r.Target()] {
					t.Errorf("%v reference %s targets %v, which the slice does not hold", e.Key(), r.Field(), r.Target())
				}
				found = append(found, r.Target().ID().String())
			}
		}
	}
	o := readOracle[speciesOracleFile](t, "species")
	var want []string
	for _, l := range o.Lineages {
		want = append(want, l.Spells...)
	}
	slices.Sort(found)
	slices.Sort(want)
	if len(found) != 9 || !slices.Equal(found, want) {
		t.Errorf("reference sites target %v, want the nine T3 spells %v", found, want)
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-084, FR-063 - a new reference site whose target is outside the
// slice fails the whole-set validation: the species document with one lineage spell renamed is rejected by the
// bundle with an unresolvedReference problem.
// Path: unhappy (dangling id: a new site with an absent target).
func TestDanglingLineageSpellIsRejected(t *testing.T) {
	t.Parallel()
	docs := embeddedDocuments(t)
	docs["species.json"] = strings.Replace(docs["species.json"], `"faerie_fire"`, `"no_such_spell"`, 1)
	src, err := data.NewBundledSource(data.WithDocuments(docs))
	var bad *data.BundledInvalidError
	if src != nil || !errors.As(err, &bad) || len(bad.DecodeProblems) != 0 {
		t.Fatalf("NewBundledSource(dangling) = (%v, %v), want nil and a *BundledInvalidError with only reference problems", src, err)
	}
	if len(bad.ReferenceProblems) != 1 || bad.ReferenceProblems[0].Kind != content.ProblemUnresolvedReference ||
		bad.ReferenceProblems[0].Owner != content.NewKey(content.KindSpecies, content.MustID("elf")) ||
		bad.ReferenceProblems[0].Target != content.NewKey(content.KindSpell, content.MustID("no_such_spell")) {
		t.Errorf("ReferenceProblems = %+v, want one unresolvedReference from elf to no_such_spell", bad.ReferenceProblems)
	}
}

// Requirement: doc/specs/content-thin-slice.md FR-078 and doc/specs/content-schema-and-decoder.md FR-079 to FR-082 -
// LoadSet once per embedded document with a recording telemetry and a fixed clock adds 4, 1 and 12 to entries_loaded
// by kind, nothing to decode_errors and three loads with outcome success.
// Path: happy.
func TestLoadSetCountsOverTheSlice(t *testing.T) {
	t.Parallel()
	rec := telemetrytest.NewRecorder()
	clock := telemetrytest.NewClock(time.Unix(1000, 0), time.Second)
	loader, err := content.NewLoader(content.JSONDecoder{}, content.WithTelemetry(rec.Telemetry()), content.WithClock(clock.Now))
	if err != nil {
		t.Fatal(err)
	}
	docs := embeddedDocuments(t)
	for _, name := range slices.Sorted(maps.Keys(docs)) {
		res, err := loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(docs[name])})
		if _, ok := res.(content.LoadSetLoaded); err != nil || !ok {
			t.Fatalf("LoadSet(%s) = (%v, %v), want LoadSetLoaded", name, res, err)
		}
	}
	perKind := map[string]float64{}
	for _, m := range rec.Meter.MeasurementsNamed(content.MetricEntriesLoaded) {
		for _, a := range m.Attrs {
			if a.Key() == content.AttrKind {
				perKind[a.Value().(string)] += m.Value
			}
		}
	}
	if want := map[string]float64{content.ValueKindSpecies: 4, content.ValueKindCharacterClass: 1, content.ValueKindSpell: 12}; !maps.Equal(perKind, want) {
		t.Errorf("entries_loaded by kind = %v, want %v", perKind, want)
	}
	if got := rec.Meter.MeasurementsNamed(content.MetricDecodeErrors); len(got) != 0 {
		t.Errorf("decode_errors = %v, want none", got)
	}
	loads := rec.Meter.MeasurementsNamed(content.MetricLoads)
	if len(loads) != 3 {
		t.Fatalf("loads = %d measurements, want 3", len(loads))
	}
	for _, m := range loads {
		if m.Value != 1 || len(m.Attrs) != 1 || m.Attrs[0].Key() != telemetry.AttrOutcome || m.Attrs[0].Value() != telemetry.OutcomeSuccess {
			t.Errorf("loads measurement = %+v, want 1 with outcome success", m)
		}
	}
}

// Requirement: doc/specs/content-schema-and-decoder.md FR-089, FR-090, FR-104, FR-120, FR-121, FR-122 - the one walk
// over content/data: the *.json files of the directory are exactly the names Files() lists, in lexical order, each a
// <stem>.json with a conforming stem; each decodes alone to a nil error with every source citing SRD 5.2.1; the
// directory holds nothing but JSON and test files besides the one embed Go file, which imports neither encoding/json
// nor os (so there is no second JSON path).
// Path: happy / unhappy (boundary: nothing beyond the listed files; invalid input: a malformed name).
func TestDirectoryAndEmbeddedFilesAgree(t *testing.T) {
	t.Parallel()
	dir := os.DirFS(".")
	onDisk, err := fs.Glob(dir, "*.json")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(data.Files(), ".")
	if err != nil {
		t.Fatal(err)
	}
	var embedded []string
	for _, e := range entries {
		embedded = append(embedded, e.Name())
	}
	if !slices.Equal(onDisk, embedded) || !slices.IsSorted(embedded) {
		t.Errorf("directory *.json = %v, Files() = %v; want the same names in lexical order", onDisk, embedded)
	}
	stem := regexp.MustCompile(`^[a-z][a-z0-9_]*\.json$`)
	for _, name := range embedded {
		if !stem.MatchString(name) {
			t.Errorf("file name %q is not <stem>.json with a conforming stem", name)
		}
		b, err := fs.ReadFile(dir, name)
		if err != nil {
			t.Fatal(err)
		}
		set, err := content.JSONDecoder{}.Decode(b)
		if err != nil {
			t.Errorf("Decode(%s) problems = %v, want none", name, content.Problems(err))
			continue
		}
		for _, e := range set.Entries() {
			if !citationOf(t, e).CitesSRD521() {
				t.Errorf("%s %v does not cite SRD 5.2.1", name, e.Key())
			}
		}
	}
	all, err := fs.ReadDir(dir, ".")
	if err != nil {
		t.Fatal(err)
	}
	var goFiles []string
	for _, e := range all {
		switch n := e.Name(); {
		case strings.HasSuffix(n, "_test.go"), strings.HasSuffix(n, ".json"), n == "testdata":
		case strings.HasSuffix(n, ".go"):
			goFiles = append(goFiles, n)
		default:
			t.Errorf("content/data holds %q, want only .json, tests and the embed Go file", n)
		}
	}
	if !slices.Equal(goFiles, []string{"bundled_source.go"}) {
		t.Fatalf("non-test Go files = %v, want only bundled_source.go", goFiles)
	}
	f, err := parser.ParseFile(token.NewFileSet(), "bundled_source.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		if p := imp.Path.Value; p == `"encoding/json"` || p == `"os"` {
			t.Errorf("bundled_source.go imports %s: a second JSON path", p)
		}
	}
}
