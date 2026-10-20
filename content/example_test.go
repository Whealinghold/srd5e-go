package content_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/content"
)

// Decode a host document: every problem is reported with its path and kind.
func ExampleJSONDecoder() {
	_, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion": 2}`))
	for _, p := range content.Problems(err) {
		fmt.Println(p.Path(), p.Kind())
	}
	set, err := content.JSONDecoder{}.Decode([]byte(`{"schemaVersion": 1}`))
	fmt.Println(len(set.Entries()), err)
	// Output:
	// /schemaVersion unsupportedVersion
	// 0 <nil>
}

// Load a document through the use case; with no options it emits nothing.
func ExampleLoader() {
	loader, err := content.NewLoader(content.JSONDecoder{})
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := loader.LoadSet(context.Background(), content.LoadSetCommand{Document: []byte(`{"schemaVersion": 1}`)})
	if err != nil {
		fmt.Println(err)
		return
	}
	switch r := res.(type) {
	case content.LoadSetLoaded:
		fmt.Println("loaded", len(r.Set().Entries()))
	case content.LoadSetRejected:
		fmt.Println("rejected", len(r.Problems()))
	}
	// Output: loaded 0
}

// Serve entries from memory and switch over the closed lookup result.
func ExampleInMemorySource() {
	src, err := content.NewInMemorySource(nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	key := content.NewKey(content.KindSpell, content.MustID("absent"))
	res, err := src.Lookup(context.Background(), key)
	if err != nil {
		fmt.Println(err)
		return
	}
	switch r := res.(type) {
	case content.LookupFound:
		fmt.Println("found", r.Entry.Key())
	case content.LookupNotFound:
		fmt.Println("not found", r.Key == key)
	}
	// Output: not found true
}
