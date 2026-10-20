package content_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/content-monsters.md FR-095 - MonsterMaxNumber is 2^31 - 1, so the decoder reads a
// number the same on every platform integer width.
// Path: unhappy (boundary).
func TestMonsterMaxNumberValue(t *testing.T) {
	t.Parallel()
	if int64(content.MonsterMaxNumber) != 2147483647 {
		t.Errorf("MonsterMaxNumber = %d, want 2147483647", int64(content.MonsterMaxNumber))
	}
}
