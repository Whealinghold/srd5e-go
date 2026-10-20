package data_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
)

// Build the bundled source once and look entries up by key.
func ExampleBundledSource() {
	src, err := data.NewBundledSource()
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx := context.Background()
	for _, key := range []content.Key{
		content.NewKey(content.KindSpecies, content.MustID("dwarf")),
		content.NewKey(content.KindSpecies, content.MustID("gnome")),
	} {
		res, err := src.Lookup(ctx, key)
		if err != nil {
			fmt.Println(err)
			return
		}
		switch r := res.(type) {
		case content.LookupFound:
			fmt.Println("found", r.Entry.(content.Species).Name())
		case content.LookupNotFound:
			fmt.Println("not found")
		}
	}
	spells, _ := src.ListKind(ctx, content.KindSpell)
	fmt.Println(len(spells), "spells")
	// Output:
	// found Dwarf
	// not found
	// 12 spells
}
