package archtest

// Kind classifies a package for the rules that depend on its role.
type Kind int

// The package kinds of doc/rules/architecture.md.
const (
	// KindDocs is the module root: documentation only.
	KindDocs Kind = iota + 1
	// KindKernel packages are shared by every context.
	KindKernel
	// KindContext packages are bounded contexts.
	KindContext
	// KindSupport packages hold test doubles and contract suites (*test).
	KindSupport
	// KindAdapter packages adapt a port to a standard-library facility.
	KindAdapter
	// KindTooling packages are checked by go test and never imported.
	KindTooling
	// KindTests packages hold cross-package tests only.
	KindTests
)

// Entry is one row of the allow-list.
type Entry struct {
	Kind Kind
	// Allow lists the module-relative packages that non-test files of this
	// package may import. Test files are not restricted by it.
	Allow []string
}

// Policy is the dependency allow-list, keyed by module-relative directory.
type Policy struct {
	Entries map[string]Entry
}

// DefaultPolicy returns the tables of doc/rules/architecture.md section 1
// (packages) and section 2 (context map). Contexts that do not exist yet are
// listed so that adding one needs no change here; changing an edge is an
// architecture change and belongs in both places.
func DefaultPolicy() Policy {
	kernel := []string{"dice", "ability", "modifier", "srderr", "telemetry"}
	return Policy{Entries: map[string]Entry{
		Root:                      {Kind: KindDocs},
		"srderr":                  {Kind: KindKernel},
		"telemetry":               {Kind: KindKernel},
		"modifier":                {Kind: KindKernel, Allow: []string{"srderr"}},
		"dice":                    {Kind: KindKernel, Allow: []string{"srderr"}},
		"ability":                 {Kind: KindKernel, Allow: []string{"srderr"}},
		"dice/dicetest":           {Kind: KindSupport, Allow: []string{"dice", "srderr"}},
		"telemetry/telemetrytest": {Kind: KindSupport, Allow: []string{"telemetry"}},
		"telemetry/slogtelemetry": {Kind: KindAdapter, Allow: []string{"telemetry"}},
		"dice/randroller":         {Kind: KindAdapter, Allow: []string{"dice", "srderr"}},
		"content/contenttest":     {Kind: KindSupport, Allow: []string{"content", "srderr"}},
		"content/data":            {Kind: KindAdapter, Allow: append([]string{"content"}, kernel...)},
		"d20":                     {Kind: KindContext, Allow: kernel},
		"content":                 {Kind: KindContext, Allow: kernel},
		"character":               {Kind: KindContext, Allow: append([]string{"content", "d20"}, kernel...)},
		"combat":                  {Kind: KindContext, Allow: append([]string{"character", "d20", "content"}, kernel...)},
		"toolbox":                 {Kind: KindContext, Allow: append([]string{"content", "d20", "character", "combat"}, kernel...)},
		"internal/archtest":       {Kind: KindTooling},
		"scenarios":               {Kind: KindTests},
	}}
}

// deniedStdlib lists the standard-library packages that kernel and context
// code must not import (rule A3); exceptions are in stdlibExceptions.
func deniedStdlib(importPath string) bool {
	switch importPath {
	case "os", "os/exec", "net", "io/fs", "io/ioutil", "reflect", "unsafe", "syscall", "log",
		"math/rand", "math/rand/v2", "sync", "sync/atomic":
		return true
	}
	return len(importPath) > 4 && importPath[:4] == "net/"
}

// stdlibExceptions are the packages allowed to import a denied package.
func stdlibExceptions(pkg, importPath string) bool {
	return importPath == "math/rand/v2" && pkg == "dice"
}
