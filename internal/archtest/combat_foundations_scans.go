package archtest

import (
	"go/ast"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"
)

// Rule identifiers of the foundation scans that sit beside the combat scans: the HitKind rule of
// hit-points-and-damage.md FR-111, the event-file rule of combat-foundations.md FR-016 to FR-020 and the
// result-class rule of combat-foundations.md FR-031.
const (
	RuleCombatHitKind     = "combat-hit-kind"
	RuleCombatEventFiles  = "combat-event-files"
	RuleCombatResultClass = "combat-result-class"
)

// hitKindFile is the one file of package combat that declares HitKind and its two members.
const hitKindFile = "combat/hit_point_inputs.go"

// Rule identifiers of the telemetry and combatant-reference scans of combat-foundations.md: attribute keys
// (FR-041), attribute values (FR-033), span and timer calls (FR-032, FR-035), the variants of CombatantKind
// (FR-012) and the imports of the combatant reference files (FR-014).
const (
	RuleCombatAttrKey       = "combat-attr-key"
	RuleCombatAttrValue     = "combat-attr-value"
	RuleCombatSpanCalls     = "combat-span-calls"
	RuleCombatKindIdentity  = "combat-kind-identity"
	RuleCombatReferenceDeps = "combat-reference-imports"
)

// instrumenterFile is the one file of package combat that starts spans and timers and records errors on them.
const instrumenterFile = "combat/instrumenter.go"

// checkCombatFoundations runs the scans above over the files of package combat.
func checkCombatFoundations(m Module, pkg Package, idx pkgIndex) []Violation {
	files := pkg.Production()
	var vs []Violation
	vs = append(vs, checkHitKind(files)...)
	vs = append(vs, checkEventFiles(files)...)
	vs = append(vs, checkResultClasses(files)...)
	vs = append(vs, checkExamples(pkg.Files)...)
	vs = append(vs, checkAttrKeys(files)...)
	vs = append(vs, checkAttrValues(files)...)
	vs = append(vs, checkSpanCalls(files)...)
	vs = append(vs, checkKindIdentity(idx)...)
	vs = append(vs, checkReferenceImports(m, files)...)
	vs = append(vs, checkOperationPairing(files)...)
	return vs
}

// attrCall returns the name (String, Bool, Int, Float) when c is a telemetry attribute constructor call.
func attrCall(c *ast.CallExpr) string {
	s, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if id, ok := s.X.(*ast.Ident); ok && id.Name == "telemetry" {
		switch s.Sel.Name {
		case "String", "Bool", "Int", "Float":
			return s.Sel.Name
		}
	}
	return ""
}

// signalAttrNames returns the attr... constants declared in combat/signals.go.
func signalAttrNames(files []File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		if f.Path != "combat/signals.go" {
			continue
		}
		for _, d := range f.AST.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.CONST {
				for _, s := range g.Specs {
					for _, n := range s.(*ast.ValueSpec).Names {
						if strings.HasPrefix(n.Name, "attr") {
							out[n.Name] = true
						}
					}
				}
			}
		}
	}
	return out
}

// checkAttrKeys covers combat-foundations.md FR-041: the key of every telemetry.String, Bool, Int or Float call
// in a non-test file (so every extra of a description, however it is built) is an attr... constant of
// signals.go, a d20.Attr... key or a telemetry.Attr... key, never a literal or another expression. The runtime
// check of the Instrumenter only sees the prefix; this scan pins the owner.
func checkAttrKeys(files []File) []Violation {
	consts := signalAttrNames(files)
	var vs []Violation
	for _, f := range files {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || attrCall(c) == "" || len(c.Args) == 0 {
				return true
			}
			switch k := c.Args[0].(type) {
			case *ast.Ident:
				if consts[k.Name] {
					return true
				}
			case *ast.SelectorExpr:
				if id, ok := k.X.(*ast.Ident); ok && (id.Name == "d20" || id.Name == "telemetry") && strings.HasPrefix(k.Sel.Name, "Attr") {
					return true
				}
			}
			vs = append(vs, at(f, c.Args[0].Pos(), RuleCombatAttrKey, "an attribute key is a CombatSignals attr constant or a d20 or telemetry Attr key (FR-041)"))
			return true
		})
	}
	return vs
}

// leaksIdentity reports whether name is one of the expressions FR-033 keeps out of an attribute value: an id, an
// amount or free text.
func leaksIdentity(name string) bool {
	switch name {
	case "CombatantID", "Amount", "Message", "Text", "id":
		return true
	}
	return strings.HasSuffix(name, "ID")
}

// checkAttrValues covers combat-foundations.md FR-033: the value of a telemetry.String, Int or Float call in a
// non-test file names no CombatantID or other id, no Amount, Message or Text and calls no Error method. A Bool
// carries no identity and is not scanned. Names are read, not types, so a differently named alias is not seen.
func checkAttrValues(files []File) []Violation {
	var vs []Violation
	for _, f := range files {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || len(c.Args) < 2 || attrCall(c) == "" || attrCall(c) == "Bool" {
				return true
			}
			ast.Inspect(c.Args[1], func(m ast.Node) bool {
				name := ""
				switch v := m.(type) {
				case *ast.Ident:
					name = v.Name
				case *ast.SelectorExpr:
					name = v.Sel.Name
				case *ast.CallExpr:
					if calleeName(v) == "Error" {
						name = "Error"
					}
				}
				if name == "Error" || leaksIdentity(name) {
					vs = append(vs, at(f, m.Pos(), RuleCombatAttrValue, "an attribute value names %s: no id, amount or free text (FR-033)", name))
				}
				return true
			})
			return true
		})
	}
	return vs
}

// checkSpanCalls covers combat-foundations.md FR-032 and FR-035: only instrumenterFile calls RecordError or
// SetStatus on a span, calls Tracer.Start or calls telemetry.StartTimer. Selector calls only, so a type named
// HazardRecordError is no hit. A tracer or timer held in a variable first is not seen.
func checkSpanCalls(files []File) []Violation {
	var vs []Violation
	for _, f := range files {
		if f.Path == instrumenterFile {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			s, ok := c.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch {
			case s.Sel.Name == "RecordError" || s.Sel.Name == "SetStatus":
				vs = append(vs, at(f, c.Pos(), RuleCombatSpanCalls, "%s is called in %s only (FR-032, FR-035)", s.Sel.Name, instrumenterFile))
			case s.Sel.Name == "StartTimer" && isIdent(s.X, "telemetry"):
				vs = append(vs, at(f, c.Pos(), RuleCombatSpanCalls, "StartTimer is called in %s only (FR-035)", instrumenterFile))
			case s.Sel.Name == "Start":
				if t, ok := s.X.(*ast.SelectorExpr); ok && t.Sel.Name == "Tracer" {
					vs = append(vs, at(f, c.Pos(), RuleCombatSpanCalls, "Tracer.Start is called in %s only (FR-035)", instrumenterFile))
				}
			}
			return true
		})
	}
	return vs
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

// checkKindIdentity covers combat-foundations.md FR-012: a variant of CombatantKind (a type with the combatantKind
// marker) has the exported field ID and no other, and no exported method.
func checkKindIdentity(idx pkgIndex) []Violation {
	var vs []Violation
	for name, ms := range idx.methods {
		marked := false
		for _, m := range ms {
			marked = marked || m.Name.Name == "combatantKind"
		}
		d, ok := idx.types[name]
		if !marked || !ok {
			continue
		}
		var exported []string
		if st, ok := d.spec.Type.(*ast.StructType); ok {
			for _, fld := range st.Fields.List {
				if len(fld.Names) == 0 {
					if b := baseIdent(fld.Type); ast.IsExported(b) {
						exported = append(exported, b)
					}
				}
				for _, n := range fld.Names {
					if n.IsExported() {
						exported = append(exported, n.Name)
					}
				}
			}
		}
		if !slices.Equal(exported, []string{"ID"}) {
			vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatKindIdentity, "%s has exported fields %v, want exactly ID (FR-012)", name, exported))
		}
		for _, m := range ms {
			if m.Name.IsExported() {
				vs = append(vs, at(idx.fileOf[m], m.Pos(), RuleCombatKindIdentity, "%s has the exported method %s; a kind carries identity only (FR-012)", name, m.Name.Name))
			}
		}
	}
	return vs
}

// referenceFiles are the files of the combatant reference types (FR-014).
func referenceFiles() []string {
	return []string{"combat/combatant_kind.go", "combat/combatant_ref.go", "combat/combatant_id.go"}
}

// checkReferenceImports covers combat-foundations.md FR-014: the reference files import the standard library and
// the character, content and srderr packages of this module, nothing else.
func checkReferenceImports(m Module, files []File) []Violation {
	var vs []Violation
	for _, f := range files {
		if !slices.Contains(referenceFiles(), f.Path) {
			continue
		}
		for _, im := range f.Imports() {
			local, isLocal := m.Local(im.Path)
			switch {
			case isLocal && (local == "character" || local == "content" || local == "srderr"):
			case isLocal, strings.Contains(strings.SplitN(im.Path, "/", 2)[0], "."):
				vs = append(vs, at(f, im.Pos, RuleCombatReferenceDeps, "%s imports %s; the reference types import the standard library, character, content and srderr only (FR-014)", f.Path, im.Path))
			}
		}
	}
	return vs
}

// calleeName is the name a call expression calls: the identifier or the selected name.
func calleeName(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// checkHitKind covers hit-points-and-damage.md FR-111: HitKind has exactly two members and is declared only
// in hitKindFile, and no call that applies damage takes an IsCriticalHit selector in its arguments (the
// attack resolvers map the roll to a HitKind, once, in their result).
func checkHitKind(files []File) []Violation {
	var vs []Violation
	for _, f := range files {
		for _, d := range f.AST.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			members := 0
			typed := false
			for _, s := range g.Specs {
				vsp := s.(*ast.ValueSpec)
				if id, ok := vsp.Type.(*ast.Ident); ok && id.Name == "HitKind" {
					typed = true
				}
				if typed {
					members += len(vsp.Names)
				}
			}
			switch {
			case typed && f.Path != hitKindFile:
				vs = append(vs, at(f, g.Pos(), RuleCombatHitKind, "HitKind members are declared in %s only (FR-111)", hitKindFile))
			case typed && members != 2:
				vs = append(vs, at(f, g.Pos(), RuleCombatHitKind, "HitKind has %d members, want exactly NormalHit and CriticalHit (FR-111)", members))
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if name := calleeName(c); !strings.Contains(name, "ApplyDamage") && !strings.Contains(name, "DealDamage") {
				return true
			}
			for _, a := range c.Args {
				ast.Inspect(a, func(m ast.Node) bool {
					if s, ok := m.(*ast.SelectorExpr); ok && s.Sel.Name == "IsCriticalHit" {
						vs = append(vs, at(f, s.Pos(), RuleCombatHitKind, "%s takes IsCriticalHit as a damage input; hand it a HitKind (FR-111)", calleeName(c)))
					}
					return true
				})
			}
			return true
		})
	}
	return vs
}

// isEventFile reports whether f is an event file: its name ends in _event.go or it declares a Kind.
func isEventFile(f File) bool {
	if strings.HasSuffix(path.Base(f.Path), "_event.go") {
		return true
	}
	for _, d := range f.AST.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "Kind" {
			return true
		}
	}
	return false
}

// checkEventFiles fails an event file that imports reflect or holds a %T verb: an event kind is a declared
// constant, never derived from a type at run time (FR-016, FR-017).
func checkEventFiles(files []File) []Violation {
	var vs []Violation
	for _, f := range files {
		if !isEventFile(f) {
			continue
		}
		for _, im := range f.AST.Imports {
			if p, err := strconv.Unquote(im.Path.Value); err == nil && p == "reflect" {
				vs = append(vs, at(f, im.Pos(), RuleCombatEventFiles, "an event file imports reflect; declare the Kind as a constant"))
			}
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if l, ok := n.(*ast.BasicLit); ok && l.Kind == token.STRING && strings.Contains(l.Value, "%T") {
				vs = append(vs, at(f, l.Pos(), RuleCombatEventFiles, "an event file formats a type name with %%T; declare the Kind as a constant"))
			}
			return true
		})
	}
	return vs
}

// checkResultClasses covers combat-foundations.md FR-031: a srd5e.combat.result constant (valueResult...,
// not the valueResultClass... values) named by two description literals with different classes has more than
// one class, and one that no file outside signals.go names reaches no use case, so it has none.
func checkResultClasses(files []File) []Violation {
	var signals *File
	for i := range files {
		if path.Base(files[i].Path) == "signals.go" {
			signals = &files[i]
		}
	}
	if signals == nil {
		return nil
	}
	declared := map[string]token.Pos{}
	for _, d := range signals.AST.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.CONST {
			for _, s := range g.Specs {
				for _, n := range s.(*ast.ValueSpec).Names {
					if isResultValue(n.Name) {
						declared[n.Name] = n.Pos()
					}
				}
			}
		}
	}
	var vs []Violation
	classes := map[string][]string{}
	used := map[string]bool{}
	for _, f := range files {
		if f.Path == signals.Path {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.Ident:
				used[v.Name] = true
			case *ast.CompositeLit:
				if t, ok := v.Type.(*ast.Ident); !ok || t.Name != "description" {
					return true
				}
				var class, name string
				for _, e := range v.Elts {
					kv, ok := e.(*ast.KeyValueExpr)
					id, isID := ast.Expr(nil), false
					if ok {
						id = kv.Value
						_, isID = kv.Value.(*ast.Ident)
					}
					if !isID {
						continue
					}
					switch kv.Key.(*ast.Ident).Name {
					case "Class":
						class = id.(*ast.Ident).Name
					case "Name":
						name = id.(*ast.Ident).Name
					}
				}
				if class != "" && isResultValue(name) && !slices.Contains(classes[name], class) {
					classes[name] = append(classes[name], class)
				}
			}
			return true
		})
	}
	for _, name := range sortedKeys(classes) {
		if len(classes[name]) > 1 {
			vs = append(vs, at(*signals, declared[name], RuleCombatResultClass, "%s is classed %s; a result value has exactly one class (FR-031)", name, strings.Join(classes[name], " and ")))
		}
	}
	for _, name := range sortedKeys(declared) {
		if !used[name] {
			vs = append(vs, at(*signals, declared[name], RuleCombatResultClass, "%s is named by no use case, so it has no result class (FR-031)", name))
		}
	}
	return vs
}

// isResultValue reports whether name is a srd5e.combat.result constant of signals.go.
func isResultValue(name string) bool {
	return strings.HasPrefix(name, "valueResult") && !strings.HasPrefix(name, "valueResultClass")
}

// RuleCombatExamples is the rule of the Example functions of package combat (combat-action-resolution.md FR-153 to FR-155).
const RuleCombatExamples = "combat-examples"

// bannedExampleImport reports whether an Example function may not use the package: randomness, the environment, the
// network and the file system (FR-154). time is judged by what is called, in clockCall.
func bannedExampleImport(p string) bool {
	switch p {
	case "os", "os/exec", "math/rand", "math/rand/v2", "crypto/rand", "net", "net/http", "io/fs", "io/ioutil", "time":
		return true
	}
	return false
}

// clockCall reports whether name, selected from the time package, reads or waits on the clock; a Duration is no clock.
func clockCall(name string) bool {
	switch name {
	case "Now", "Since", "Until", "Sleep", "After", "AfterFunc", "Tick", "NewTimer", "NewTicker":
		return true
	}
	return false
}

// checkExamples covers combat-action-resolution.md FR-153 to FR-155 over the Example functions of the test files of
// package combat: each has an Output comment, so go test runs it; no switch has a default arm (FR-153); it calls no
// print or println builtin, which write to stderr (FR-155); and it touches no clock, randomness, environment, network
// or file system (FR-154). A token scan, so a helper an example calls is not seen.
func checkExamples(files []File) []Violation {
	var vs []Violation
	for _, f := range files {
		if !f.Test {
			continue
		}
		banned := map[string]string{}
		for _, im := range f.Imports() {
			if bannedExampleImport(im.Path) {
				banned[path.Base(strings.TrimSuffix(im.Path, "/v2"))] = im.Path
			}
		}
		for _, d := range f.AST.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || !strings.HasPrefix(fd.Name.Name, "Example") || fd.Body == nil {
				continue
			}
			if !hasOutputComment(f, fd) {
				vs = append(vs, at(f, fd.Pos(), RuleCombatExamples, "%s has no Output comment, so go test does not run it (FR-152)", fd.Name.Name))
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.CaseClause:
					if v.List == nil {
						vs = append(vs, at(f, v.Pos(), RuleCombatExamples, "%s has a default arm; switch over every variant (FR-153)", fd.Name.Name))
					}
				case *ast.CallExpr:
					if id, ok := v.Fun.(*ast.Ident); ok && (id.Name == "print" || id.Name == "println") {
						vs = append(vs, at(f, v.Pos(), RuleCombatExamples, "%s calls the builtin %s; write to stdout with fmt (FR-155)", fd.Name.Name, id.Name))
					}
				case *ast.SelectorExpr:
					if id, ok := v.X.(*ast.Ident); ok && banned[id.Name] != "" && (banned[id.Name] != "time" || clockCall(v.Sel.Name)) {
						vs = append(vs, at(f, v.Pos(), RuleCombatExamples, "%s uses %s; an example has no clock, randomness, environment, network or file system (FR-154)", fd.Name.Name, banned[id.Name]))
					}
				}
				return true
			})
		}
	}
	return vs
}

// hasOutputComment reports whether the body of fd ends in an Output or Unordered output comment.
func hasOutputComment(f File, fd *ast.FuncDecl) bool {
	for _, g := range f.AST.Comments {
		if g.Pos() < fd.Body.Lbrace || g.End() > fd.Body.Rbrace {
			continue
		}
		for _, c := range g.List {
			t := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(c.Text, "//")))
			if strings.HasPrefix(t, "output:") || strings.HasPrefix(t, "unordered output:") {
				return true
			}
		}
	}
	return false
}
