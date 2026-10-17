package archtest

import (
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// displayTextName reports whether n names engine-generated display text,
// which error and result types must not carry (A8).
func displayTextName(n string) bool {
	switch n {
	case "Message", "Description", "Text", "DisplayText":
		return true
	}
	return false
}

// errorShapeSuffixes are the type-name endings that mark an error or result type.
func errorShapeSuffixes() []string {
	return []string{"Error", "Result", "Rejection", "Problem", "Reason"}
}

// exemptFromErrorShape lists the declaration sites A8 does not scan: content
// value objects hold verbatim SRD text, telemetry descriptors describe signals.
func exemptFromErrorShape(p Policy, pkg Package) bool {
	return pkg.Dir == "content" || strings.HasPrefix(pkg.Dir, "content/") ||
		pkg.Dir == "telemetry" || p.Entries[pkg.Dir].Kind == KindSupport
}

// portConsts returns the top-level const specs named Port in a file, with the
// value expression (nil when the spec has none).
func portConsts(f File) []portConst {
	var out []portConst
	for _, decl := range f.AST.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, spec := range g.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, n := range vs.Names {
				if n.Name != "Port" {
					continue
				}
				pc := portConst{file: f, pos: n.Pos()}
				if i < len(vs.Values) {
					pc.value = vs.Values[i]
				}
				out = append(out, pc)
			}
		}
	}
	return out
}

type portConst struct {
	file  File
	pos   token.Pos
	value ast.Expr
}

// checkErrorShape covers A8 (a): no exported field or method named like
// display text on an error or result type. A type is an error or result type
// when it has an Error() method, its name ends in Error, Result, Rejection,
// Problem or Reason, or it implements a sealed interface of its package.
func checkErrorShape(p Policy, pkg Package) []Violation {
	if exemptFromErrorShape(p, pkg) {
		return nil
	}
	methods := map[string][]string{}
	for _, f := range pkg.Production() {
		for _, decl := range f.AST.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv != nil {
				methods[receiverName(fd.Recv)] = append(methods[receiverName(fd.Recv)], fd.Name.Name)
			}
		}
	}
	var sealed []string
	for _, set := range sealedSets(pkg) {
		sealed = append(sealed, set.implementers...)
	}
	shaped := func(name string) bool {
		if slices.Contains(methods[name], "Error") || slices.Contains(sealed, name) {
			return true
		}
		for _, s := range errorShapeSuffixes() {
			if strings.HasSuffix(name, s) {
				return true
			}
		}
		return false
	}
	var vs []Violation
	for _, f := range pkg.Production() {
		for _, decl := range f.AST.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				r := receiverName(d.Recv)
				if d.Recv != nil && ast.IsExported(r) && displayTextName(d.Name.Name) && shaped(r) {
					vs = append(vs, at(f, d.Pos(), RuleErrorShape, "method %s.%s: error and result types carry data, not display text", r, d.Name.Name))
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() && shaped(ts.Name.Name) {
						vs = append(vs, displayFields(f, ts)...)
					}
				}
			}
		}
	}
	return vs
}

func displayFields(f File, ts *ast.TypeSpec) []Violation {
	var list []*ast.Field
	switch t := ts.Type.(type) {
	case *ast.StructType:
		list = t.Fields.List
	case *ast.InterfaceType:
		list = t.Methods.List
	}
	var vs []Violation
	for _, field := range list {
		for _, n := range field.Names {
			if n.IsExported() && displayTextName(n.Name) {
				vs = append(vs, at(f, n.Pos(), RuleErrorShape, "%s.%s: error and result types carry data, not display text", ts.Name.Name, n.Name))
			}
		}
	}
	return vs
}

// checkPortNames covers A8 (b): every const Port is a string literal matching
// ^[a-z][a-z0-9_]*$ and unique across the module.
func checkPortNames(m Module) []Violation {
	valid := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	first := map[string]string{}
	var vs []Violation
	for _, pkg := range m.Packages {
		for _, f := range pkg.Production() {
			for _, pc := range portConsts(f) {
				lit, ok := pc.value.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					vs = append(vs, at(f, pc.pos, RuleErrorShape, "const Port must be a string literal so the name can be checked"))
					continue
				}
				v, _ := strconv.Unquote(lit.Value)
				switch owner, dup := first[v]; {
				case !valid.MatchString(v):
					vs = append(vs, at(f, pc.pos, RuleErrorShape, "port name %q must match ^[a-z][a-z0-9_]*$", v))
				case dup:
					vs = append(vs, at(f, pc.pos, RuleErrorShape, "port name %q is already used by %s", v, owner))
				default:
					first[v] = f.Path
				}
			}
		}
	}
	return vs
}
