package combat_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Requirement: doc/plans/plan-phase-4.md 5.12 audit, P11 and doc/specs/combat-foundations.md FR-037 - the signal catalogue and
// combat/signals.go name the same srd5e.combat signals.
// Path: unhappy (drift between code and catalogue).
func TestSignalCatalogueEqualsTheConstants(t *testing.T) {
	t.Parallel()
	code, err := os.ReadFile("signals.go")
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := os.ReadFile("../doc/observability/signals.md")
	if err != nil {
		t.Fatal(err)
	}
	literal := regexp.MustCompile(`"(srd5e\.combat[a-z0-9_.]*)"`)
	inCode := map[string]bool{}
	for _, m := range literal.FindAllStringSubmatch(string(code), -1) {
		inCode[m[1]] = true
	}
	if len(inCode) < 100 {
		t.Fatalf("found only %d srd5e.combat names in signals.go; the scan is broken", len(inCode))
	}
	inCatalogue := map[string]bool{}
	for _, m := range regexp.MustCompile("`(srd5e\\.combat[a-z0-9_.]*)`").FindAllStringSubmatch(string(catalogue), -1) {
		inCatalogue[m[1]] = true
	}
	for name := range inCode {
		if !inCatalogue[name] && !strings.HasSuffix(name, ".") { // "srd5e.combat." is the span prefix
			t.Errorf("%s is a constant of signals.go with no catalogue row", name)
		}
	}
	for name := range inCatalogue {
		if !inCode[name] && !strings.HasSuffix(name, ".") {
			t.Errorf("%s is in the catalogue and is no constant of signals.go", name)
		}
	}
}
