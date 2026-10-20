package archtest

import (
	"fmt"
	"go/token"
	"sort"
)

// Rule identifiers carried by Violation.Rule.
const (
	RuleUnlisted       = "unlisted-package"
	RuleImport         = "import-not-allowed"
	RuleIsolation      = "test-support-isolation"
	RuleInternal       = "internal-import"
	RuleStdlib         = "stdlib-denied"
	RuleThirdParty     = "third-party-import"
	RuleGoMod          = "go-mod-require"
	RuleInit           = "init-function"
	RuleVar            = "package-level-var"
	RuleEffect         = "hidden-effect"
	RulePanic          = "panic-discipline"
	RuleDoc            = "missing-doc"
	RulePackageDoc     = "missing-package-doc"
	RuleStutter        = "stutter"
	RuleConsumer       = "consumer-name"
	RuleKernelUp       = "kernel-imports-context"
	RuleCycle          = "import-cycle"
	RulePortUse        = "port-use"
	RuleVariant        = "variant-set"
	RuleErrorShape     = "error-shape"
	RuleNoticePaths    = "notice-paths"
	RuleEmbed          = "embedded-data"
	RuleRuleTag        = "rule-tag"
	RuleDocsOnly       = "docs-only"
	RuleIdentity       = "identity-files"
	RuleVocabulary     = "vocabulary-declaration"
	RuleSourceFiles    = "source-files"
	RuleJSONConfined   = "json-confined"
	RuleContentPortUse = "content-port-use"
	RuleThirtyTwoBit   = "thirty-two-bit"
)

// Violation is one breach of an architecture rule.
type Violation struct {
	Rule string
	// File is module-relative; it is empty for module-wide rules.
	File string
	Line int
	Msg  string
}

// String renders the violation as file:line: rule: message.
func (v Violation) String() string {
	loc := v.File
	if v.File != "" && v.Line > 0 {
		loc = fmt.Sprintf("%s:%d", v.File, v.Line)
	}
	if loc == "" {
		loc = "module"
	}
	return fmt.Sprintf("%s: %s: %s", loc, v.Rule, v.Msg)
}

// Check runs every rule against the module and returns the violations sorted
// by file, line and rule. An empty result means the architecture holds.
func Check(m Module, p Policy) []Violation {
	var vs []Violation
	vs = append(vs, checkModule(m)...)
	vs = append(vs, checkCycles(m)...)
	for _, pkg := range m.Packages {
		entry, listed := p.Entries[pkg.Dir]
		if !listed {
			vs = append(vs, Violation{Rule: RuleUnlisted, File: pkg.Dir, Msg: "package is in neither the allow-list nor the context map; add it to the policy and doc/rules/architecture.md"})
		}
		vs = append(vs, checkImports(m, p, pkg, entry, listed)...)
		vs = append(vs, checkIdentityFiles(m, pkg)...)
		vs = append(vs, checkSourceFiles(m, pkg)...)
		vs = append(vs, checkDecls(p, pkg)...)
		vs = append(vs, checkDocs(pkg)...)
		vs = append(vs, checkPortUse(m, p, pkg)...)
		vs = append(vs, checkVariants(p, pkg)...)
		vs = append(vs, checkErrorShape(p, pkg)...)
		vs = append(vs, checkDocsOnly(p, pkg)...)
		vs = append(vs, checkJSONConfined(pkg)...)
		vs = append(vs, checkContentPortUse(pkg)...)
	}
	vs = append(vs, checkConsumerNames(m, consumerNames())...)
	vs = append(vs, checkPortNames(m)...)
	vs = append(vs, checkEmbedded(m)...)
	vs = append(vs, checkVocabulary(m)...)
	vs = append(vs, checkThirtyTwoBit(m)...)
	vs = append(vs, checkRuleTags(m)...)
	sort.SliceStable(vs, func(i, j int) bool {
		a, b := vs[i], vs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Rule < b.Rule
	})
	return vs
}

func at(f File, pos token.Pos, rule, format string, args ...any) Violation {
	return Violation{Rule: rule, File: f.Path, Line: f.Fset.Position(pos).Line, Msg: fmt.Sprintf(format, args...)}
}

func checkModule(m Module) []Violation {
	if m.HasRequire {
		return []Violation{{Rule: RuleGoMod, File: "go.mod", Msg: "go.mod has a require directive; the module is standard-library only"}}
	}
	return nil
}
