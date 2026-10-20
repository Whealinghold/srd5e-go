package content_test

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/content-source-port.md FR-038, FR-017 - Port is "content"
// in the allowed form, and a use-case-style GuardValue call over a failing
// source yields a *srderr.PortError{Port: "content"} whose ErrorType is
// content_error and which still reaches the *SourceError and the backend error;
// a panicking third-party source is converted the same way.
// Path: unhappy (collaborator failure).
func TestSourceFailureIsAContentPortError(t *testing.T) {
	t.Parallel()
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(content.Port) || content.Port != "content" {
		t.Errorf("Port = %q, want \"content\"", content.Port)
	}
	backend := errors.New("backend down")
	key := plain(content.KindSpell, "a").Key()
	var src content.Source = contenttest.NewFakeSource().FailAll(backend)

	_, err := srderr.GuardValue(content.Port, "lookup", func() (content.LookupResult, error) {
		return src.Lookup(context.Background(), key)
	})
	var pe *srderr.PortError
	var se *content.SourceError
	switch {
	case !errors.As(err, &pe) || pe.Port != content.Port || pe.Op != "lookup":
		t.Fatalf("error = %v, want *srderr.PortError{Port: content, Op: lookup}", err)
	case srderr.ErrorType(err) != "content_error":
		t.Errorf("ErrorType = %q, want content_error", srderr.ErrorType(err))
	case !errors.As(err, &se) || se.Query != (content.LookupQuery{Key: key}) || !errors.Is(err, backend):
		t.Errorf("error %v does not reach the SourceError and the backend error", err)
	}

	_, err = srderr.GuardValue(content.Port, "list_kind", func() ([]content.Entry, error) {
		panic(backend)
	})
	if !errors.As(err, &pe) || pe.Port != content.Port || pe.Op != "list_kind" {
		t.Errorf("recovered panic = %v, want *srderr.PortError{Port: content, Op: list_kind}", err)
	}
}

// Requirement: doc/specs/content-source-port.md FR-019 - SourceError,
// LookupResult and its variants, and the other result and error types of the
// port carry structured data only: no exported field or method named Message,
// Description, Text or DisplayText. (The A8 scan skips package content, so this
// test is the check.)
// Path: unhappy (invalid design).
func TestSourceTypesCarryNoDisplayText(t *testing.T) {
	t.Parallel()
	banned := map[string]bool{"Message": true, "Description": true, "Text": true, "DisplayText": true}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[content.SourceError](), reflect.TypeFor[*content.SourceError](),
		reflect.TypeFor[content.DuplicateKeyError](), reflect.TypeFor[*content.DuplicateKeyError](),
		reflect.TypeFor[content.LookupFound](), reflect.TypeFor[content.LookupNotFound](),
		reflect.TypeFor[content.LookupQuery](), reflect.TypeFor[content.ListKindQuery](),
		reflect.TypeFor[content.Catalogue](), reflect.TypeFor[content.LookupResult](),
	} {
		for i := range typ.NumMethod() {
			if banned[typ.Method(i).Name] {
				t.Errorf("%v has method %s", typ, typ.Method(i).Name)
			}
		}
		if typ.Kind() == reflect.Struct {
			for i := range typ.NumField() {
				if banned[typ.Field(i).Name] {
					t.Errorf("%v has field %s", typ, typ.Field(i).Name)
				}
			}
		}
	}
}
