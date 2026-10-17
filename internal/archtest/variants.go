package archtest

import (
	"go/ast"
	"go/token"
	"regexp"
	"slices"
)

// genericVariantName reports whether n is a variant name too generic to export
// from a context package (A7a); each context prefixes its own variants.
func genericVariantName(n string) bool {
	switch n {
	case "Applied", "Rejected", "Succeeded", "ContentMissing", "Found", "NotFound":
		return true
	}
	return false
}

// sealedSet is an interface with at least one unexported method (a closed
// variant set) and the types of its package that implement it.
type sealedSet struct {
	file         File
	spec         *ast.TypeSpec
	doc          *ast.CommentGroup
	implementers []string
}

// sealedSets finds the sealed interfaces of the package's production files. A
// type implements one when it has a method of every name the interface lists,
// whatever the receiver form; promotion through embedding is not seen.
func sealedSets(pkg Package) []sealedSet {
	methods := map[string][]string{}
	for _, f := range pkg.Production() {
		for _, decl := range f.AST.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv != nil {
				r := receiverName(fd.Recv)
				methods[r] = append(methods[r], fd.Name.Name)
			}
		}
	}
	var out []sealedSet
	for _, f := range pkg.Production() {
		for _, decl := range f.AST.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, spec := range g.Specs {
				ts := spec.(*ast.TypeSpec)
				names, sealed := interfaceMethods(ts)
				if !sealed {
					continue
				}
				doc := ts.Doc
				if doc == nil && !g.Lparen.IsValid() {
					doc = g.Doc
				}
				set := sealedSet{file: f, spec: ts, doc: doc}
				for typ, have := range methods {
					if typ != ts.Name.Name && containsAll(have, names) {
						set.implementers = append(set.implementers, typ)
					}
				}
				slices.Sort(set.implementers)
				out = append(out, set)
			}
		}
	}
	return out
}

// interfaceMethods returns the method names of an interface type and whether
// at least one is unexported.
func interfaceMethods(ts *ast.TypeSpec) (names []string, sealed bool) {
	it, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil, false
	}
	for _, field := range it.Methods.List {
		for _, n := range field.Names {
			names = append(names, n.Name)
			sealed = sealed || !n.IsExported()
		}
	}
	return names, sealed
}

func containsAll(have, want []string) bool {
	for _, w := range want {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

// checkVariants covers A7: generic variant names are not exported from a
// context package, and the doc comment of every sealed interface names every
// implementer in its package.
func checkVariants(p Policy, pkg Package) []Violation {
	var vs []Violation
	if p.Entries[pkg.Dir].Kind == KindContext {
		for _, f := range pkg.Production() {
			for _, decl := range f.AST.Decls {
				g, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range g.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if genericVariantName(s.Name.Name) {
							vs = append(vs, at(f, s.Pos(), RuleVariant, "type %s is too generic for a context package; prefix it with the use case or noun", s.Name.Name))
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if g.Tok == token.CONST && genericVariantName(n.Name) {
								vs = append(vs, at(f, n.Pos(), RuleVariant, "const %s is too generic for a context package; prefix it with the use case or noun", n.Name))
							}
						}
					}
				}
			}
		}
	}
	for _, set := range sealedSets(pkg) {
		text := ""
		if set.doc != nil {
			text = set.doc.Text()
		}
		for _, impl := range set.implementers {
			if !regexp.MustCompile(`\b` + regexp.QuoteMeta(impl) + `\b`).MatchString(text) {
				vs = append(vs, at(set.file, set.spec.Pos(), RuleVariant, "doc comment of sealed interface %s must name its implementer %s", set.spec.Name.Name, impl))
			}
		}
	}
	return vs
}
