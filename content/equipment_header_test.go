package content_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: equipSpec FR-007 - an equipment entry keeps the citation, id and
// name it was built with, and an invalid id is reported before an invalid name.
// Path: happy and unhappy.
func TestEquipmentHeaderKeepsSource(t *testing.T) {
	t.Parallel()
	src, err := content.NewSourceCitation("Doc", "Section", 7)
	if err != nil {
		t.Fatal(err)
	}
	v, err := content.NewToolVariant("v_a", "Variant A", src, "family_x", fixedCost(5, content.CoinGP), content.NoWeight{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := content.ParseID("v_a")
	if err != nil {
		t.Fatal(err)
	}
	if v.Source() != src || v.ID() != "v_a" || v.Name() != "Variant A" || v.Key() != content.NewKey(content.KindEquipment, id) {
		t.Errorf("header = %+v, %q, %q, %v; want the built source, id, name and key", v.Source(), v.ID(), v.Name(), v.Key())
	}
	_, err = content.NewToolVariant("Bad", "", src, "family_x", fixedCost(5, content.CoinGP), content.NoWeight{})
	var idErr *content.IDError
	if !errors.As(err, &idErr) {
		t.Errorf("invalid id and name error = %v, want *IDError first", err)
	}
}
