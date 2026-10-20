package archtest

import (
	"go/ast"
	"go/token"
	"path"
	"strings"
)

// bannedCall reports whether a qualified call hides a time or process effect (A4).
func bannedCall(q string) bool {
	switch q {
	case "time.Now", "time.Since", "time.Sleep", "time.After", "time.Tick", "os.Exit":
		return true
	}
	return false
}

// guardFile is the one production file that may call recover (A5).
const guardFile = "srderr/guard.go"

// checkDecls covers A4 (hidden effects: init, package-level vars, banned
// calls, go statements) and A5 (panic and recover discipline) for the
// production files of one package.
func checkDecls(p Policy, pkg Package) []Violation {
	var vs []Violation
	for _, f := range pkg.Production() {
		vs = append(vs, checkTopLevel(pkg.Dir, f)...)
		vs = append(vs, checkCalls(p, pkg, f)...)
	}
	return vs
}

func checkTopLevel(pkgDir string, f File) []Violation {
	var vs []Violation
	for _, decl := range f.AST.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "init" {
				vs = append(vs, at(f, d.Pos(), RuleInit, "init function; use explicit constructors"))
			}
		case *ast.GenDecl:
			if d.Tok != token.VAR {
				continue
			}
			for _, spec := range d.Specs {
				vspec := spec.(*ast.ValueSpec)
				if embedded(pkgDir, d, vspec) {
					continue
				}
				for _, name := range vspec.Names {
					if n := name.Name; n != "_" && !sentinel(n, vspec) {
						vs = append(vs, at(f, name.Pos(), RuleVar, "package-level var %s; only _ assertions and Err sentinels are allowed", n))
					}
				}
			}
		}
	}
	return vs
}

func checkCalls(p Policy, pkg Package, f File) []Violation {
	var vs []Violation
	// Panics are allowed in Must* functions and in default case clauses.
	var allowed []ast.Node
	ast.Inspect(f.AST, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.FuncDecl:
			if strings.HasPrefix(v.Name.Name, "Must") && v.Body != nil {
				allowed = append(allowed, v.Body)
			}
		case *ast.CaseClause:
			if v.List == nil {
				allowed = append(allowed, v)
			}
		}
		return true
	})
	within := func(pos token.Pos) bool {
		for _, n := range allowed {
			if n.Pos() <= pos && pos < n.End() {
				return true
			}
		}
		return false
	}
	ast.Inspect(f.AST, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.GoStmt:
			vs = append(vs, at(f, v.Pos(), RuleEffect, "go statement; the domain spawns no goroutines"))
		case *ast.CallExpr:
			vs = append(vs, checkCall(p, pkg, f, v, within)...)
		}
		return true
	})
	return vs
}

func checkCall(p Policy, pkg Package, f File, call *ast.CallExpr, within func(token.Pos) bool) []Violation {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		switch fn.Name {
		case "panic":
			var vs []Violation
			if !within(call.Pos()) {
				vs = append(vs, at(f, call.Pos(), RulePanic, "panic outside a Must* function or default case; return an error or result"))
			}
			if len(call.Args) == 1 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					vs = append(vs, at(f, call.Pos(), RulePanic, "panic with string literal %s; panic with an error value", lit.Value))
				}
			}
			return vs
		case "recover":
			if f.Path != guardFile && p.Entries[pkg.Dir].Kind != KindSupport {
				return []Violation{at(f, call.Pos(), RulePanic, "recover outside %s and test-support packages", guardFile)}
			}
		case "print", "println":
			return []Violation{at(f, call.Pos(), RuleEffect, "builtin %s writes to stderr", fn.Name)}
		}
	case *ast.SelectorExpr:
		x, ok := fn.X.(*ast.Ident)
		if !ok {
			return nil
		}
		q := x.Name + "." + fn.Sel.Name
		if bannedCall(q) || (x.Name == "fmt" && strings.HasPrefix(fn.Sel.Name, "Print")) {
			return []Violation{at(f, call.Pos(), RuleEffect, "call to %s hides an effect; inject it as a port", q)}
		}
	}
	return nil
}

// checkDocsOnly reports declarations in the module root, which is
// documentation only (FR-062), and in the tests-only scenarios
// directory: neither may declare anything outside _test.go files.
func checkDocsOnly(p Policy, pkg Package) []Violation {
	var vs []Violation
	if k := p.Entries[pkg.Dir].Kind; k != KindDocs && k != KindTests {
		return nil
	}
	for _, f := range pkg.Production() {
		for _, decl := range f.AST.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
				continue // import legality is A1's concern
			}
			vs = append(vs, at(f, decl.Pos(), RuleDocsOnly, "%s holds documentation or tests only; move this declaration to a package", pkg.Dir))
		}
	}
	return vs
}

// sentinel reports whether a package-level var named n is an error sentinel:
// an Err-prefixed name whose declared type, if any, is error and whose value
// is not a literal.
func sentinel(n string, spec *ast.ValueSpec) bool {
	if !strings.HasPrefix(n, "Err") {
		return false
	}
	if id, ok := spec.Type.(*ast.Ident); spec.Type != nil && (!ok || id.Name != "error") {
		return false
	}
	for _, v := range spec.Values {
		if _, lit := v.(*ast.BasicLit); lit {
			return false
		}
	}
	return true
}

// embedded reports whether a package-level var is the //go:embed target of the
// embedded data package: the one place a package-level var is allowed besides
// sentinels (architecture.md section 5).
func embedded(pkgDir string, d *ast.GenDecl, spec *ast.ValueSpec) bool {
	if pkgDir != dataDir {
		return false
	}
	for _, doc := range []*ast.CommentGroup{spec.Doc, d.Doc} {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			if strings.HasPrefix(c.Text, "//go:embed ") {
				return true
			}
		}
	}
	return false
}

// VocabularyTypes are the closed vocabulary types, the creature space and the
// typed ids that package content declares exactly once (schema FR-128,
// FR-130). Adding a vocabulary type to the schema spec adds it here.
func VocabularyTypes() []string {
	return []string{
		"Skill", "DamageType", "CreatureType", "CreatureSize", "CreatureSpace",
		"ConditionKind", "Language", "RechargeTrigger", "WeaponProperty",
		"MasteryProperty", "FeatCategory", "HitPointDie", "LightLevel",
		"DarknessKind", "Obscurement", "SpecialSense", "Alignment", "DurationUnit",
		"SpeciesID", "BackgroundID", "ClassID", "SubclassID", "FeatID", "SpellID",
		"ItemID", "MonsterID", "FeatureID", "MagicItemID",
	}
}

// vocabularyDecls returns, for each vocabulary type, the violations-to-be: one
// entry per declaration in a production file, in module order.
func vocabularyDecls(m Module) map[string][]Violation {
	want := map[string]bool{}
	for _, n := range VocabularyTypes() {
		want[n] = true
	}
	found := map[string][]Violation{}
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			for _, decl := range f.AST.Decls {
				g, ok := decl.(*ast.GenDecl)
				if !ok || g.Tok != token.TYPE {
					continue
				}
				for _, spec := range g.Specs {
					ts := spec.(*ast.TypeSpec)
					if want[ts.Name.Name] {
						found[ts.Name.Name] = append(found[ts.Name.Name], at(f, ts.Pos(), RuleVocabulary, "%s is declared in package %s; it is declared once, in package content", ts.Name.Name, pkg.Dir))
					}
				}
			}
		}
	}
	return found
}

// checkVocabulary covers schema FR-128 and FR-130: each vocabulary type is
// declared at most once in the module's production files, and in package
// content. A missing declaration is not reported here (fixtures omit them);
// MissingVocabulary lets a test of the real module require them.
func checkVocabulary(m Module) []Violation {
	var vs []Violation
	found := vocabularyDecls(m)
	for _, name := range VocabularyTypes() {
		for i, v := range found[name] {
			if i > 0 || path.Dir(v.File) != "content" {
				vs = append(vs, v)
			}
		}
	}
	return vs
}

// MissingVocabulary returns the vocabulary types that have no declaration in
// the module's production files. The real module has none missing.
func MissingVocabulary(m Module) []string {
	var missing []string
	found := vocabularyDecls(m)
	for _, n := range VocabularyTypes() {
		if len(found[n]) == 0 {
			missing = append(missing, n)
		}
	}
	return missing
}
