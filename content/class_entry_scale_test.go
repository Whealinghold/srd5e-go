package content_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-classes.md FR-062 and FR-096 - the option
// prerequisite check costs time linear in the options, so a hostile chain
// within the document size cap cannot make a feature entry expensive.
// Path: unhappy (hostile size; completing at all, not a wall-clock bound, is
// the check, since a quadratic scan would hit the test timeout), then the cycle closed at the far end.
func TestFeatureEntryLongPrerequisiteChain(t *testing.T) {
	t.Parallel()
	const n = 50000
	build := func(closeCycle bool) error {
		opts := make([]content.FeatureOption, n)
		for i := range opts {
			var pre content.OptionPrerequisite
			var req []content.FeatureOptionID
			if i > 0 {
				req = []content.FeatureOptionID{content.FeatureOptionID("o" + strconv.Itoa(i-1))}
			} else if closeCycle {
				req = []content.FeatureOptionID{content.FeatureOptionID("o" + strconv.Itoa(n-1))}
			}
			if req != nil {
				pre = mustValue(content.NewOptionPrerequisite(0, req, ""))
			}
			opts[i] = mustValue(content.NewFeatureOption(content.FeatureOptionID("o"+strconv.Itoa(i)), "Opt", content.FeatureOptionFields{Text: "t", Prerequisite: pre}))
		}
		_, err := content.NewFeatureEntry("a_b", "Name", content.FeatureEntryFields{Level: 1, Text: "t", Options: opts})
		return err
	}
	if err := build(false); err != nil {
		t.Fatalf("chain: %v", err)
	}
	err := build(true)
	var be *content.BlockError
	if !errors.As(err, &be) || be.Problem != content.BlockConstraint {
		t.Fatalf("cycle: got %v, want BlockConstraint", err)
	}
}

// Requirement: doc/specs/content-classes.md FR-045 and FR-096 - the check that
// a feature is named by the row of its own level costs time linear in the
// document, so many level 1 features all named by row 1 cannot make a valid
// class expensive to decode.
// Path: unhappy (hostile size), decoding a valid document with no problems.
func TestDecodeClassManyFeaturesInOneRow(t *testing.T) {
	t.Parallel()
	const n = 30000
	var feats, refs []string
	for i := range n {
		id := "fixture_x" + strconv.Itoa(i)
		feats = append(feats, featureJSON(id, 1, ""))
		refs = append(refs, `{"featureId":"`+id+`"}`)
	}
	doc := wizardClassJSON()
	doc = strings.Replace(doc, `"features":[{"id":"fixture_alpha"`, `"features":[`+strings.Join(feats, ",")+`,{"id":"fixture_alpha"`, 1)
	doc = strings.Replace(doc, `{"level":1,"proficiencyBonus":2,"features":[`, `{"level":1,"proficiencyBonus":2,"features":[`+strings.Join(refs, ",")+`,`, 1)
	if got := decodeProblems(classDoc(doc)); len(got) != 0 {
		t.Fatalf("problems = %q, want none", got[:min(len(got), 5)])
	}
}
