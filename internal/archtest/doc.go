// Package archtest enforces the architecture rules of doc/rules/architecture.md
// mechanically. It parses the module's Go files with go/parser (standard
// library only, no type checking), then reports every breach of the dependency
// allow-list and of the hygiene rules as a Violation.
//
// The scanner reads from an fs.FS, so the same checks run against the real
// module tree (see module_test.go) and against small in-memory fixtures that
// prove each rule fires. Package archtest is executed by go test and is never
// imported by other packages.
//
// Every check is lexical or AST-level and so has blind spots (for example a
// banned call reached through a variable). Review covers those.
package archtest
