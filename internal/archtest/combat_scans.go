package archtest

import (
	"fmt"
	"go/ast"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Rule identifiers of the combat scans (combat-foundations.md FR-022, FR-023, FR-026, FR-038).
const (
	RuleCombatNaming   = "combat-naming"
	RuleCombatImport   = "combat-import-rank"
	RuleCombatEnvelope = "combat-envelope"
	RuleCombatSpecRank = "combat-spec-rank"
)

// combatDir is the module-relative directory of the combat context.
const combatDir = "combat"

// genericNames are the result-variant names no context package exports (FR-022).
func genericNames() []string {
	return []string{"Applied", "Rejected", "Succeeded", "Invalid", "ContentMissing", "Found", "NotFound"}
}

// envelopeNames is the set of transport envelopes encounter-aggregate.md FR-050 and FR-084 name;
// the marked set must equal it (combat-foundations.md FR-038). A new envelope is added here and there.
func envelopeNames() []string {
	return []string{"TurnEventOccurred", "HitPointEventOccurred", "ConditionEventOccurred", "GrappleEventOccurred"}
}

// combatSpecRanks bounds the ranks of the combat specs in the table of shared-types-and-ownership.md 4.11.
const (
	combatRankLow  = 20
	combatRankHigh = 39
)

// checkCombat runs the scans that need the combat package: naming (FR-022, FR-023), the import
// direction (FR-026) and the transport-envelope set (FR-038). It reports nothing when the module has
// no combat package, so fixtures of other contexts stay clean.
func checkCombat(m Module, p Policy) []Violation {
	var pkg *Package
	for i := range m.Packages {
		if m.Packages[i].Dir == combatDir {
			pkg = &m.Packages[i]
		}
	}
	if pkg == nil {
		return nil
	}
	idx := indexPackage(*pkg)
	var vs []Violation
	vs = append(vs, checkCombatNames(idx)...)
	vs = append(vs, checkCombatEnvelopes(idx)...)
	vs = append(vs, checkCombatImports(m, p, *pkg, idx)...)
	vs = append(vs, checkCombatSpecRanks(m)...)
	vs = append(vs, checkCombatFoundations(m, *pkg, idx)...)
	return vs
}

// declInfo is one top-level type declaration with the file that holds it.
type declInfo struct {
	spec *ast.TypeSpec
	file File
}

// pkgIndex is the type and method table of the production files of one package.
type pkgIndex struct {
	files   []File
	types   map[string]declInfo
	methods map[string][]*ast.FuncDecl
	fileOf  map[*ast.FuncDecl]File
}

func indexPackage(pkg Package) pkgIndex {
	idx := pkgIndex{files: pkg.Production(), types: map[string]declInfo{}, methods: map[string][]*ast.FuncDecl{}, fileOf: map[*ast.FuncDecl]File{}}
	for _, f := range idx.files {
		for _, d := range f.AST.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				for _, s := range d.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok {
						idx.types[ts.Name.Name] = declInfo{ts, f}
					}
				}
			case *ast.FuncDecl:
				if d.Recv != nil {
					r := receiverName(d.Recv)
					idx.methods[r] = append(idx.methods[r], d)
					idx.fileOf[d] = f
				}
			}
		}
	}
	return idx
}

func (idx pkgIndex) method(recv, name string) *ast.FuncDecl {
	for _, fd := range idx.methods[recv] {
		if fd.Name.Name == name {
			return fd
		}
	}
	return nil
}

// baseIdent returns the type name of a field or parameter type, looking through pointers, slices and
// arrays; it is "" for a qualified or composite type.
func baseIdent(e ast.Expr) string {
	for {
		switch v := e.(type) {
		case *ast.StarExpr:
			e = v.X
		case *ast.ArrayType:
			e = v.Elt
		case *ast.Ident:
			return v.Name
		default:
			return ""
		}
	}
}

// reasonField reports whether the struct type name holds, directly or through an embedded struct of the
// package, a field whose type name ends in one of the suffixes.
func (idx pkgIndex) reasonField(name string, suffixes []string, seen map[string]bool) bool {
	d, ok := idx.types[name]
	if !ok || seen[name] {
		return false
	}
	seen[name] = true
	st, ok := d.spec.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, fld := range st.Fields.List {
		t := baseIdent(fld.Type)
		if slices.ContainsFunc(suffixes, func(s string) bool { return strings.HasSuffix(t, s) }) {
			return true
		}
		if len(fld.Names) == 0 && idx.reasonField(t, suffixes, seen) {
			return true
		}
	}
	return false
}

// checkCombatNames covers FR-022 and FR-023 for package combat.
func checkCombatNames(idx pkgIndex) []Violation {
	var vs []Violation
	generic := genericNames()
	for _, f := range idx.files {
		for _, d := range f.AST.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && slices.Contains(generic, d.Name.Name) {
					vs = append(vs, at(f, d.Pos(), RuleCombatNaming, "%s is a generic name; prefix it with the operation noun (FR-022)", d.Name.Name))
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					for _, n := range specNames(s) {
						if slices.Contains(generic, n.Name) {
							vs = append(vs, at(f, n.Pos(), RuleCombatNaming, "%s is a generic name; prefix it with the operation noun (FR-022)", n.Name))
						}
					}
				}
			}
		}
	}

	// The results of the use cases: the types the exported Run methods return.
	markers := map[string]string{} // unexported marker method -> use-case result interface
	for _, recv := range sortedKeys(idx.methods) {
		run := idx.method(recv, "Run")
		if !ast.IsExported(recv) || run == nil || run.Type.Results == nil || len(run.Type.Results.List) != 2 {
			continue
		}
		name := baseIdent(run.Type.Results.List[0].Type)
		d, found := idx.types[name]
		if !found {
			continue
		}
		if it, ok := d.spec.Type.(*ast.InterfaceType); ok {
			for _, m := range it.Methods.List {
				for _, mn := range m.Names {
					if !mn.IsExported() {
						markers[mn.Name] = name
					}
				}
			}
		}
	}
	variants := map[string]bool{}
	for _, name := range sortedKeys(idx.methods) {
		if !ast.IsExported(name) {
			continue
		}
		vs = append(vs, checkUseCaseShape(idx, name)...)
		ifaces := map[string]bool{}
		for _, fd := range idx.methods[name] {
			if r, ok := markers[fd.Name.Name]; ok {
				ifaces[r] = true
			}
		}
		variants[name] = len(ifaces) > 0
		if len(ifaces) > 1 {
			d := idx.types[name]
			vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatNaming, "%s is a variant of %d use-case results; a variant belongs to one use case (FR-023)", name, len(ifaces)))
		}
	}

	for _, name := range sortedKeys(idx.types) {
		d := idx.types[name]
		st, isStruct := d.spec.Type.(*ast.StructType)
		if !ast.IsExported(name) || !isStruct || !variants[name] {
			continue
		}
		switch {
		case strings.HasSuffix(name, "Rejected"):
			if !idx.reasonField(name, []string{"Rejection", "Problem"}, map[string]bool{}) {
				vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatNaming, "%s holds no field of a *Rejection or *Problem type (FR-023)", name))
			}
		case strings.HasSuffix(name, "Invalid"):
			// A parser's failure is held as the error that joins its problems; a variant with no
			// field at all has one reason, the command itself, and nothing to hold.
			if len(st.Fields.List) > 0 && !idx.reasonField(name, []string{"Rejection", "Problem", "error"}, map[string]bool{}) {
				vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatNaming, "%s holds no field of a *Rejection or *Problem type, or the error that joins them (FR-023)", name))
			}
		case strings.HasSuffix(name, "NotPossible"):
			if !idx.reasonField(name, []string{"Reason"}, map[string]bool{}) {
				vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatNaming, "%s holds no field of a *Reason type (FR-023)", name))
			}
		}
	}
	return vs
}

func specNames(s ast.Spec) []*ast.Ident {
	switch s := s.(type) {
	case *ast.TypeSpec:
		return []*ast.Ident{s.Name}
	case *ast.ValueSpec:
		return s.Names
	}
	return nil
}

// checkUseCaseShape checks the use case U with a method Run(ctx, cmd) (result, error): the command is
// U+"Command" and the result is U+"Result", a closed interface (FR-022).
func checkUseCaseShape(idx pkgIndex, u string) []Violation {
	run := idx.method(u, "Run")
	if run == nil || !run.Name.IsExported() {
		return nil
	}
	f := idx.fileOf[run]
	params := run.Type.Params.List
	results := run.Type.Results
	if len(params) != 2 || results == nil || len(results.List) != 2 {
		return []Violation{at(f, run.Pos(), RuleCombatNaming, "%s.Run must be Run(ctx, %sCommand) (%sResult, error) (FR-022)", u, u, u)}
	}
	var vs []Violation
	if got := baseIdent(params[1].Type); got != u+"Command" {
		vs = append(vs, at(f, run.Pos(), RuleCombatNaming, "%s.Run takes %q, want %sCommand (FR-022)", u, got, u))
	}
	got := baseIdent(results.List[0].Type)
	if got != u+"Result" {
		vs = append(vs, at(f, run.Pos(), RuleCombatNaming, "%s.Run returns %q, want %sResult (FR-022)", u, got, u))
		return vs
	}
	d, ok := idx.types[got]
	it, isIface := d.spec.Type.(*ast.InterfaceType)
	if !ok || !isIface || !slices.ContainsFunc(it.Methods.List, func(m *ast.Field) bool {
		return len(m.Names) > 0 && !m.Names[0].IsExported()
	}) {
		vs = append(vs, at(f, run.Pos(), RuleCombatNaming, "%s must be an interface with an unexported marker method (FR-022)", got))
	}
	return vs
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// checkCombatEnvelopes covers FR-038: the types implementing transportEnvelope are exactly the named
// envelopes, each delegates Kind and Combatants to the wrapped event, and a type named ...Occurred is
// marked.
func checkCombatEnvelopes(idx pkgIndex) []Violation {
	var vs []Violation
	want := envelopeNames()
	marked := map[string]bool{}
	for recv, fds := range idx.methods {
		for _, fd := range fds {
			if fd.Name.Name == "transportEnvelope" {
				marked[recv] = true
			}
		}
	}
	for _, name := range want {
		if _, shipped := idx.types["transportEnvelope"]; !shipped {
			break // the package has no envelope machinery yet
		}
		if !marked[name] {
			vs = append(vs, Violation{Rule: RuleCombatEnvelope, File: combatDir, Msg: name + " is a named envelope but does not implement transportEnvelope (FR-038)"})
		}
	}
	for _, name := range sortedKeys(marked) {
		d, ok := idx.types[name]
		if !ok {
			continue
		}
		if !slices.Contains(want, name) {
			vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatEnvelope, "%s implements transportEnvelope but is not a named envelope (FR-038)", name))
		}
		for _, method := range []string{"Kind", "Combatants"} {
			fd := idx.method(name, method)
			switch {
			case fd == nil:
				vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatEnvelope, "envelope %s has no %s method (FR-038)", name, method))
			case returnsLiteral(fd):
				vs = append(vs, at(idx.fileOf[fd], fd.Pos(), RuleCombatEnvelope, "envelope %s.%s returns a literal; it reports the wrapped event's value (FR-038)", name, method))
			}
		}
	}
	for _, name := range sortedKeys(idx.types) {
		d := idx.types[name]
		if ast.IsExported(name) && strings.HasSuffix(name, "Occurred") && !marked[name] {
			vs = append(vs, at(d.file, d.spec.Pos(), RuleCombatEnvelope, "%s is named like an envelope but does not implement transportEnvelope (FR-038)", name))
		}
	}
	return vs
}

// returnsLiteral reports whether fd's body is one return of a literal (a string, number or nil).
func returnsLiteral(fd *ast.FuncDecl) bool {
	if fd.Body == nil || len(fd.Body.List) != 1 {
		return false
	}
	ret, ok := fd.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	switch r := ret.Results[0].(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		return r.Name == "nil"
	}
	return false
}

// serviceTypes returns the exported struct types of a package with an exported method whose first
// parameter is a context.Context: the use cases and domain services it publishes.
func serviceTypes(pkg Package) map[string]bool {
	out := map[string]bool{}
	idx := indexPackage(pkg)
	for name, d := range idx.types {
		if _, ok := d.spec.Type.(*ast.StructType); !ok || !ast.IsExported(name) {
			continue
		}
		for _, fd := range idx.methods[name] {
			if fd.Name.IsExported() && firstParamIsContext(fd) {
				out[name] = true
			}
		}
	}
	return out
}

func firstParamIsContext(fd *ast.FuncDecl) bool {
	if len(fd.Type.Params.List) == 0 {
		return false
	}
	sel, ok := fd.Type.Params.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "context" && sel.Sel.Name == "Context"
}

// checkCombatImports covers the Go half of FR-026: a combat file names no use case of character or
// content, and names a d20 domain service only in a use-case file. Imports of toolbox and of internal/
// are A1 and A2 and stay there. The rank half is checkCombatSpecRanks.
func checkCombatImports(m Module, p Policy, pkg Package, idx pkgIndex) []Violation {
	services := map[string]map[string]bool{}
	for _, up := range m.Packages {
		if up.Dir == "character" || up.Dir == "d20" || up.Dir == "content" {
			services[up.Dir] = serviceTypes(up)
		}
	}
	var vs []Violation
	for _, f := range idx.files {
		useCaseFile := isUseCaseFile(m, p, pkg, f)
		for upstream, set := range services {
			name := importName(f, m.Path+"/"+upstream)
			if name == "" {
				continue
			}
			ast.Inspect(f.AST, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if x, ok := sel.X.(*ast.Ident); !ok || x.Name != name || !set[sel.Sel.Name] {
					return true
				}
				if upstream == "d20" && useCaseFile {
					return true // a d20 domain service, such as d20.Resolver, is allowed in a use-case file
				}
				vs = append(vs, at(f, sel.Pos(), RuleCombatImport, "%s.%s is another context's use case or service; combat composes published language only, and d20 services only in use-case files (FR-026)", upstream, sel.Sel.Name))
				return true
			})
		}
	}
	return vs
}

// The patterns of the rank table and of the "Depends on" row, compiled on use (a package-level var is
// not allowed, A4).
func rankRow() *regexp.Regexp    { return regexp.MustCompile(`^\| (\d+) \| (.*) \|$`) }
func specToken() *regexp.Regexp  { return regexp.MustCompile("`([a-z0-9-]+)\\.md`") }
func specTicked() *regexp.Regexp { return regexp.MustCompile("`([a-z0-9-]+)`") }

const rankDoc = "doc/specs/shared-types-and-ownership.md"

// specRanks parses the rank table of shared-types-and-ownership.md section 4.11; it is nil when the
// document or the table is absent.
func specRanks(m Module) map[string]int {
	raw, err := fs.ReadFile(m.FS, rankDoc)
	if err != nil {
		return nil
	}
	var ranks map[string]int
	in := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "### 4.11"):
			in, ranks = true, map[string]int{}
		case in && strings.HasPrefix(line, "## "):
			in = false
		case in:
			if r := rankRow().FindStringSubmatch(line); r != nil {
				n, _ := strconv.Atoi(r[1])
				for _, s := range specTicked().FindAllStringSubmatch(r[2], -1) {
					ranks[s[1]] = n
				}
			}
		}
	}
	return ranks
}

// checkCombatSpecRanks covers the rank half of FR-026 (and FR-024, shared-types-and-ownership.md
// FR-056): the "Depends on" row of every combat spec lists only specs of strictly lower rank, so no
// combat spec lists a spec of equal or higher rank or a toolbox spec. A Go import cannot be mapped to
// a spec, so the rank is checked on the spec headers.
func checkCombatSpecRanks(m Module) []Violation {
	ranks := specRanks(m)
	if ranks == nil {
		return nil
	}
	var vs []Violation
	for _, name := range sortedKeys(ranks) {
		rank := ranks[name]
		if rank < combatRankLow || rank > combatRankHigh {
			continue
		}
		doc := "doc/specs/" + name + ".md"
		raw, err := fs.ReadFile(m.FS, doc)
		if err != nil {
			vs = append(vs, Violation{Rule: RuleCombatSpecRank, File: doc, Msg: fmt.Sprintf("spec %s has rank %d in the table but no file", name, rank)})
			continue
		}
		found := false
		for i, line := range strings.Split(string(raw), "\n") {
			if !strings.HasPrefix(line, "| Depends on |") {
				continue
			}
			found = true
			for _, tok := range specToken().FindAllStringSubmatch(line, -1) {
				dep := tok[1]
				r, ok := ranks[dep]
				switch {
				case !ok:
					vs = append(vs, Violation{Rule: RuleCombatSpecRank, File: doc, Line: i + 1, Msg: dep + " is not in the rank table (shared-types-and-ownership.md 4.11)"})
				case r >= rank:
					vs = append(vs, Violation{Rule: RuleCombatSpecRank, File: doc, Line: i + 1, Msg: fmt.Sprintf("%s (rank %d) depends on %s (rank %d); a spec may depend only on a strictly lower rank (FR-026)", name, rank, dep, r)})
				}
			}
			break
		}
		if !found {
			vs = append(vs, Violation{Rule: RuleCombatSpecRank, File: doc, Msg: "no \"Depends on\" row in the header"})
		}
	}
	return vs
}
