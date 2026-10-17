package archtest_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/Whealinghold/srd5e-go/internal/archtest"
)

// TestModuleArchitecture runs every rule against the real module tree. Each
// violation is its own failure line so a breach names the file and rule.
func TestModuleArchitecture(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.Packages) < 5 {
		t.Fatalf("loaded %d packages; the scan is not seeing the module", len(m.Packages))
	}
	for _, v := range archtest.Check(m, archtest.DefaultPolicy()) {
		t.Error(v)
	}
}

// TestPolicyCoversEveryPackage fails when a package directory exists that the
// policy table does not list, so a new package cannot slip in unchecked.
func TestPolicyCoversEveryPackage(t *testing.T) {
	t.Parallel()
	m, err := archtest.Load(os.DirFS("../.."))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	policy := archtest.DefaultPolicy()
	for _, pkg := range m.Packages {
		if _, ok := policy.Entries[pkg.Dir]; !ok {
			t.Errorf("package %q is not in the policy table", pkg.Dir)
		}
	}
}

// TestRealModuleHasRuleFiles pins the files A13 and A12 depend on, so deleting
// doc/rules/ cannot make the vacuous checks pass.
func TestRealModuleHasRuleFiles(t *testing.T) {
	t.Parallel()
	rules, err := fs.Glob(os.DirFS("../.."), "doc/rules/*.md")
	if err != nil || len(rules) != 4 {
		t.Fatalf("doc/rules/*.md = %v (err %v); want architecture, go, observability and testing", rules, err)
	}
}
