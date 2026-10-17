package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Root is the key of the package that sits at the module root.
const Root = "."

// File is one parsed Go source file.
type File struct {
	// Path is slash-separated and relative to the module root.
	Path string
	// Test reports whether the file name ends in _test.go.
	Test bool
	// AST is the parsed file, comments included.
	AST *ast.File
	// Fset resolves positions in AST.
	Fset *token.FileSet
}

// Import is one import declaration of a file.
type Import struct {
	Path string
	Pos  token.Pos
}

// Imports returns the unquoted imports of the file in source order.
func (f File) Imports() []Import {
	out := make([]Import, 0, len(f.AST.Imports))
	for _, spec := range f.AST.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue // the parser already rejected malformed literals
		}
		out = append(out, Import{Path: p, Pos: spec.Pos()})
	}
	return out
}

// Package is the set of Go files in one directory.
type Package struct {
	// Dir is the module-relative directory, or Root.
	Dir   string
	Files []File
}

// Name returns the package clause name of the first non-test file, or of the
// first file when every file is a test.
func (p Package) Name() string {
	for _, f := range p.Files {
		if !f.Test {
			return f.AST.Name.Name
		}
	}
	if len(p.Files) > 0 {
		return strings.TrimSuffix(p.Files[0].AST.Name.Name, "_test")
	}
	return ""
}

// Production returns the non-test files.
func (p Package) Production() []File {
	var out []File
	for _, f := range p.Files {
		if !f.Test {
			out = append(out, f)
		}
	}
	return out
}

// Module is a parsed Go module tree.
type Module struct {
	// Path is the module path from go.mod.
	Path string
	// HasRequire reports whether go.mod has a require directive.
	HasRequire bool
	// Packages are the directories holding Go files, sorted by Dir.
	Packages []Package
	// FS is the tree the module was loaded from.
	FS fs.FS
}

// Local reports whether an import path belongs to the module and, if so,
// returns its module-relative form (Root for the module itself).
func (m Module) Local(importPath string) (string, bool) {
	switch {
	case importPath == m.Path:
		return Root, true
	case strings.HasPrefix(importPath, m.Path+"/"):
		return strings.TrimPrefix(importPath, m.Path+"/"), true
	}
	return "", false
}

// Load parses go.mod and every .go file below the root of fsys. Directories
// named testdata or vendor and those starting with "." or "_" are skipped, as
// the go tool does. It returns an error if go.mod is missing or has no module
// line, or if any Go file does not parse.
func Load(fsys fs.FS) (Module, error) {
	mod, err := fs.ReadFile(fsys, "go.mod")
	if err != nil {
		return Module{}, fmt.Errorf("archtest: read go.mod: %w", err)
	}
	m := Module{FS: fsys}
	for _, line := range strings.Split(string(mod), "\n") {
		fields := strings.Fields(line)
		switch {
		case len(fields) >= 2 && fields[0] == "module":
			m.Path = strings.Trim(fields[1], `"`)
		case len(fields) >= 1 && fields[0] == "require":
			m.HasRequire = true
		}
	}
	if m.Path == "" {
		return Module{}, fmt.Errorf("archtest: go.mod has no module line")
	}

	byDir := map[string]*Package{}
	fset := token.NewFileSet()
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if p != "." && (name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		src, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		af, err := parser.ParseFile(fset, p, src, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("archtest: parse %s: %w", p, err)
		}
		dir := path.Dir(p)
		pkg := byDir[dir]
		if pkg == nil {
			pkg = &Package{Dir: dir}
			byDir[dir] = pkg
		}
		pkg.Files = append(pkg.Files, File{Path: p, Test: strings.HasSuffix(name, "_test.go"), AST: af, Fset: fset})
		return nil
	})
	if err != nil {
		return Module{}, err
	}
	for _, p := range byDir {
		m.Packages = append(m.Packages, *p)
	}
	sort.Slice(m.Packages, func(i, j int) bool { return m.Packages[i].Dir < m.Packages[j].Dir })
	return m, nil
}
