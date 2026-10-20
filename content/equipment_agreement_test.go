package content_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-equipment.md FR-004, FR-008 and doc/specs/content-schema-and-decoder.md FR-094 -
// the id rule of a weapon row in the decoder and ParseItemID give the same
// verdict, and the weapons array accepts a row exactly when the id is valid.
// Path: per case.
func TestAgreementEquipmentWeaponID(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"club_x", "a", "a" + strings.Repeat("b", 63), "z9_", "", "A", "2a", "a-b", " a", "a\nb", "a" + strings.Repeat("b", 64), "a\xff", "a\u202eb"} {
		_, perr := content.ParseItemID(id)
		doc := eqDoc("weapons", strings.Replace(weaponJSON, `"club_x"`, jsonString(id), 1))
		_, derr := content.JSONDecoder{}.Decode([]byte(doc))
		if (perr == nil) != (derr == nil) {
			t.Errorf("id %q: ParseItemID error = %v, decoder error = %v", id, perr, derr)
		}
	}
}

// Requirement: doc/specs/content-equipment.md FR-007 and doc/specs/content-schema-and-decoder.md FR-021, FR-094 -
// the integer rule of a cost amount in a weapon row and NewCost agree for
// every spelling and bound.
// Path: per case.
func TestAgreementEquipmentCostAmount(t *testing.T) {
	t.Parallel()
	max := strconv.Itoa(content.MaxCostAmount)
	for _, r := range []struct {
		lit  string
		want bool
	}{
		{"0", true}, {"1", true}, {"2.0", true}, {"2e0", true}, {max, true}, {strconv.FormatInt(int64(content.MaxCostAmount)+1, 10), false},
		{"-1", false}, {"2.5", false}, {`"2"`, false}, {"null", false}, {"1e999999999", false},
	} {
		doc := eqDoc("weapons", strings.Replace(weaponJSON, `"amount":1`, `"amount":`+r.lit, 1))
		_, err := content.JSONDecoder{}.Decode([]byte(doc))
		if (err == nil) != r.want {
			t.Errorf("amount %s: decoder error = %v, want accepted=%v", r.lit, err, r.want)
		}
		if n, perr := strconv.Atoi(r.lit); perr == nil {
			if _, cerr := content.NewCost(n, content.CoinSP, 0); (cerr == nil) != r.want {
				t.Errorf("NewCost(%d) error = %v, want accepted=%v", n, cerr, r.want)
			}
		}
	}
}
