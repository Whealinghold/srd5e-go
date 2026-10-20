package archtest

import (
	"bytes"
	"go/ast"
	"io/fs"
	"path"
	"strings"
)

// checkJSONConfined covers schema FR-073 and FR-075: only the files named
// decode*.go in package content import encoding/json (no other production
// file of the module does), so JSON shapes are known to the decoder alone.
func checkJSONConfined(pkg Package) []Violation {
	var vs []Violation
	for _, f := range pkg.Production() {
		base := path.Base(f.Path)
		if pkg.Dir == "content" && strings.HasPrefix(base, "decode") {
			continue
		}
		for _, imp := range f.Imports() {
			if imp.Path == "encoding/json" {
				vs = append(vs, at(f, imp.Pos, RuleJSONConfined, "only content/decode*.go may import encoding/json"))
			}
		}
	}
	return vs
}

// loadUseCaseFile is the one file of package content that refers to the port
// failure helpers of srderr (schema FR-141).
const loadUseCaseFile = "load_usecase.go"

// checkContentPortUse covers schema FR-141: in package content only
// load_usecase.go refers to PortError or to the guard and wrap helpers of
// srderr. Every other file reports through returned errors wrapping
// srderr.ErrInvalidInput. A6 cannot see this: package content owns the Source
// port, so A6 exempts it as the port's owner.
func checkContentPortUse(pkg Package) []Violation {
	if pkg.Dir != "content" {
		return nil
	}
	var vs []Violation
	for _, f := range pkg.Production() {
		if path.Base(f.Path) == loadUseCaseFile {
			continue
		}
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && portName(id.Name) {
				vs = append(vs, at(f, id.Pos(), RuleContentPortUse, "%s refers to %s; only %s may", path.Base(f.Path), id.Name, loadUseCaseFile))
			}
			return true
		})
	}
	return vs
}

// thirtyTwoBitCommand is the local and CI command that runs the decoder tests
// at a 32-bit integer width (schema FR-099).
const thirtyTwoBitCommand = "GOARCH=386 go test ./content/..."

// thirtyTwoBitFiles are the files that must name the 32-bit run: the CI
// workflow, the contributor commands and the release checklist.
func thirtyTwoBitFiles() []string {
	return []string{".github/workflows/ci.yml", "AGENTS.md", "doc/specs/release-0-1-0.md"}
}

// checkThirtyTwoBit covers schema FR-099: once the decoder exists
// (content/decode.go), the CI workflow, AGENTS.md and the release-0-1-0 spec
// each name the 32-bit run. The check is vacuous before the decoder lands.
func checkThirtyTwoBit(m Module) []Violation {
	if _, err := fs.Stat(m.FS, "content/decode.go"); err != nil {
		return nil
	}
	var vs []Violation
	for _, p := range thirtyTwoBitFiles() {
		raw, err := fs.ReadFile(m.FS, p)
		if err != nil {
			vs = append(vs, Violation{Rule: RuleThirtyTwoBit, File: p, Msg: "missing; it must name " + thirtyTwoBitCommand})
			continue
		}
		if !bytes.Contains(raw, []byte(thirtyTwoBitCommand)) {
			vs = append(vs, Violation{Rule: RuleThirtyTwoBit, File: p, Msg: "does not name the 32-bit decoder run " + thirtyTwoBitCommand})
		}
	}
	return vs
}
