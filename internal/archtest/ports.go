package archtest

import (
	"go/ast"
	"path"
	"slices"
	"strings"
)

// portName reports whether an srderr identifier is confined to use-case files (A6).
func portName(n string) bool {
	switch n {
	case "WrapPort", "Guard", "GuardValue", "ContractViolation", "PortError":
		return true
	}
	return false
}

// importName returns the identifier an import path is bound to in f, or "" when
// f does not import it by name (absent, blank or dot import).
func importName(f File, importPath string) string {
	for _, spec := range f.AST.Imports {
		if strings.Trim(spec.Path.Value, "\"`") != importPath {
			continue
		}
		switch {
		case spec.Name == nil:
			return path.Base(importPath)
		case spec.Name.Name == "_" || spec.Name.Name == ".":
			return ""
		default:
			return spec.Name.Name
		}
	}
	return ""
}

// isUseCaseFile reports whether f is a use-case file (Q-1, by role): it sits
// in a context package, imports the module's telemetry package, and declares an
// exported type with an exported method in the same file.
func isUseCaseFile(m Module, p Policy, pkg Package, f File) bool {
	if p.Entries[pkg.Dir].Kind != KindContext || !slices.ContainsFunc(f.Imports(), func(i Import) bool { return i.Path == m.Path+"/telemetry" }) {
		return false
	}
	types := map[string]bool{}
	for _, decl := range f.AST.Decls {
		if g, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range g.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.IsExported() {
					types[ts.Name.Name] = true
				}
			}
		}
	}
	for _, decl := range f.AST.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && fd.Recv != nil && fd.Name.IsExported() && types[receiverName(fd.Recv)] {
			return true
		}
	}
	return false
}

// ownsPort reports whether the package declares a top-level const Port, i.e.
// it is the owner of a port and may wrap that port's failures itself.
func ownsPort(pkg Package) bool {
	for _, f := range pkg.Production() {
		if len(portConsts(f)) > 0 {
			return true
		}
	}
	return false
}

// checkPortUse covers A6: the srderr port helpers are used only by use-case
// files, srderr itself, test-support packages and the owner package of a port.
func checkPortUse(m Module, p Policy, pkg Package) []Violation {
	if pkg.Dir == "srderr" || p.Entries[pkg.Dir].Kind == KindSupport || ownsPort(pkg) {
		return nil
	}
	var vs []Violation
	for _, f := range pkg.Production() {
		name := importName(f, m.Path+"/srderr")
		if name == "" || isUseCaseFile(m, p, pkg, f) {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && portName(sel.Sel.Name) {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == name {
					vs = append(vs, at(f, sel.Pos(), RulePortUse, "%s.%s used outside a use-case file; domain files return results, use cases wrap ports", name, sel.Sel.Name))
				}
			}
			return true
		})
	}
	return vs
}
