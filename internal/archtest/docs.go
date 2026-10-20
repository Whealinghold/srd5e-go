package archtest

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

// checkDocs covers A9 (documentation coverage) and the no-stutter naming rule
// for the production files of one package.
func checkDocs(pkg Package) []Violation {
	var vs []Violation
	name := pkg.Name()
	hasDoc := false
	for _, f := range pkg.Production() {
		// content/data may hold only one Go file (A12), so that file carries the package comment there.
		if f.Path == pkg.Dir+"/doc.go" || (pkg.Dir == Root && f.Path == "doc.go") || pkg.Dir == dataDir {
			hasDoc = hasDoc || (f.AST.Doc != nil && strings.HasPrefix(f.AST.Doc.Text(), "Package "+name+" "))
		}
		for _, decl := range f.AST.Decls {
			vs = append(vs, checkDeclDoc(f, name, decl)...)
		}
	}
	if len(pkg.Production()) > 0 && !hasDoc {
		vs = append(vs, Violation{Rule: RulePackageDoc, File: pkg.Dir, Msg: "doc.go must carry a package comment starting \"Package " + name + " \""})
	}
	return vs
}

func checkDeclDoc(f File, pkgName string, decl ast.Decl) []Violation {
	var vs []Violation
	need := func(pos token.Pos, name string, doc *ast.CommentGroup, stutters bool) {
		if !ast.IsExported(name) {
			return
		}
		if doc == nil || !strings.HasPrefix(doc.Text(), name) {
			vs = append(vs, at(f, pos, RuleDoc, "exported %s needs a doc comment starting with its name", name))
		}
		if stutters && stutter(pkgName, name) {
			vs = append(vs, at(f, pos, RuleStutter, "%s.%s stutters; drop the package name from the identifier", pkgName, name))
		}
	}
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil {
			need(d.Pos(), d.Name.Name, d.Doc, true)
		} else if ast.IsExported(receiverName(d.Recv)) {
			need(d.Pos(), d.Name.Name, d.Doc, false)
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			vs = append(vs, checkSpecDoc(f, pkgName, d, spec)...)
		}
	}
	return vs
}

func checkSpecDoc(f File, pkgName string, d *ast.GenDecl, spec ast.Spec) []Violation {
	var vs []Violation
	grouped := d.Lparen.IsValid()
	doc := func(own *ast.CommentGroup) *ast.CommentGroup {
		if own == nil && !grouped {
			return d.Doc
		}
		return own
	}
	report := func(pos token.Pos, name string, own *ast.CommentGroup) {
		if !ast.IsExported(name) {
			return
		}
		groupDocumented := grouped && d.Doc != nil && d.Tok != token.TYPE
		if c := doc(own); c == nil && !groupDocumented {
			vs = append(vs, at(f, pos, RuleDoc, "exported %s needs a doc comment starting with its name", name))
		} else if c != nil && !strings.HasPrefix(c.Text(), name) {
			vs = append(vs, at(f, pos, RuleDoc, "doc comment of %s must start with its name", name))
		}
		if stutter(pkgName, name) {
			vs = append(vs, at(f, pos, RuleStutter, "%s.%s stutters; drop the package name from the identifier", pkgName, name))
		}
	}
	switch s := spec.(type) {
	case *ast.TypeSpec:
		report(s.Pos(), s.Name.Name, s.Doc)
	case *ast.ValueSpec:
		for _, n := range s.Names {
			report(n.Pos(), n.Name, s.Doc)
		}
	}
	return vs
}

// receiverName returns the base type name of a method receiver.
func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	for {
		switch v := t.(type) {
		case *ast.StarExpr:
			t = v.X
		case *ast.IndexExpr:
			t = v.X
		case *ast.IndexListExpr:
			t = v.X
		case *ast.Ident:
			return v.Name
		default:
			return ""
		}
	}
}

// stutter reports whether name repeats the package name as a prefix followed
// by a new word (dice.DiceRoll), as golint does. An exact match (ability.Ability)
// is the accepted idiom for a package's principal type.
func stutter(pkg, name string) bool {
	if len(name) <= len(pkg) || !strings.EqualFold(name[:len(pkg)], pkg) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name[len(pkg):])
	return unicode.IsUpper(r) || unicode.IsDigit(r)
}
