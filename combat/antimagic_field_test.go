package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// dspTargetRef returns a creature target reference for id.
func dspTargetRef(t *testing.T, id string) combat.SpellTargetRef {
	t.Helper()
	return combat.CreatureTargetRef{ID: dspCreature(t, id)}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-043 and table T-A - Status is suppressed exactly for an ordinary spell inside the aura that is not exempt, one row per case (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: per case.
func TestAntimagicFieldStatusTableT_A(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		spec                    dspSpec
		inside                  bool
		want                    combat.SpellFunctioning
	}{
		{"ordinary spell inside", "FR-043", "happy", dspSpec{effect: "e", spell: "bless", level: 1}, true, combat.Suppressed},
		{"ordinary spell outside", "FR-043", "happy", dspSpec{effect: "e", spell: "bless", level: 1}, false, combat.Functioning},
		{"concentration spell inside", "FR-043", "happy", dspSpec{effect: "e", spell: "haste", level: 3, concentration: true}, true, combat.Suppressed},
		{"artifact or deity", "FR-051", "unhappy (rule interaction)", dspSpec{effect: "e", spell: "bless", level: 1, origin: combat.SpellOriginArtifactOrDeity}, true, combat.Functioning},
		{"a second field's aura", "FR-044", "unhappy (rule interaction)", dspSpec{effect: "e", spell: "antimagic_field", level: 8}, true, combat.Functioning},
		{"prismatic wall", "FR-044", "unhappy (rule interaction)", dspSpec{effect: "e", spell: "prismatic_wall", level: 8}, true, combat.Functioning},
		{"cantrip with a duration", "FR-043", "unhappy (boundary)", dspSpec{effect: "e", spell: "light", level: 0}, true, combat.Suppressed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := combat.AntimagicField{}.Status(dspSpell(t, c.spec), c.inside)
			if got != c.want {
				t.Errorf("Status = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-045 to FR-047 and table T-B - Blocks gives the verdict of each activity, one row per case (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: per case.
func TestAntimagicFieldBlocksTableT_B(t *testing.T) {
	t.Parallel()
	goblin := []combat.SpellTargetRef{dspTargetRef(t, "goblin")}
	cases := []struct {
		name, requirement, path string
		activity                combat.AntimagicActivity
		actor                   bool
		targets                 []combat.SpellTargetRef
		want                    string
	}{
		{"cast from inside", "FR-045", "happy", combat.ActivityCastSpell, true, nil, "blocked:actor_inside_aura"},
		{"cast at a target inside", "FR-045", "happy", combat.ActivityCastSpell, false, goblin, "blocked:target_inside_aura"},
		{"both inside, the actor first", "FR-045", "unhappy (rule interaction)", combat.ActivityCastSpell, true, goblin, "blocked:actor_inside_aura"},
		{"cast, nothing inside", "FR-045", "happy", combat.ActivityCastSpell, false, nil, "permits"},
		{"magic action inside", "FR-045", "happy", combat.ActivityTakeMagicAction, true, nil, "blocked:actor_inside_aura"},
		{"item property on something inside", "FR-046", "unhappy (rule interaction)", combat.ActivityUseMagicItemProperty, true, goblin, "blocked:actor_inside_aura"},
		{"item outside, target inside", "FR-046", "unhappy (boundary)", combat.ActivityUseMagicItemProperty, false, goblin, "permits"},
		{"teleport out of the aura", "FR-047", "happy", combat.ActivityTeleport, true, nil, "blocked:actor_inside_aura"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			v, err := combat.AntimagicField{}.Blocks(c.activity, c.actor, c.targets)
			if err != nil {
				t.Fatalf("Blocks error = %v [%s]", err, c.requirement)
			}
			if got := verdictName(t, v); got != c.want {
				t.Errorf("Blocks = %s, want %s [%s]", got, c.want, c.requirement)
			}
		})
	}
}

// verdictName names a verdict: permits, or blocked:<reason>.
func verdictName(t *testing.T, v combat.AntimagicVerdict) string {
	t.Helper()
	switch b := v.(type) {
	case combat.AntimagicPermits:
		return "permits"
	case combat.AntimagicBlocks:
		return "blocked:" + b.Block().String()
	default:
		t.Fatalf("verdict %T is not a known variant", v)
		return ""
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-045 - a duplicate target is invalid input, and a valid verdict copies its targets (invalid input; immutability).
// Path: per case.
func TestAntimagicFieldBlocksRefusesDuplicateTargets(t *testing.T) {
	t.Parallel()
	goblin := dspTargetRef(t, "goblin")
	_, err := combat.AntimagicField{}.Blocks(combat.ActivityCastSpell, false, []combat.SpellTargetRef{goblin, goblin})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("Blocks(duplicate targets) error = %v, want ErrInvalidInput [FR-045]", err)
	}
	if _, err := (combat.AntimagicField{}).Blocks(combat.AntimagicActivity(0), false, nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("Blocks(undefined activity) error = %v, want ErrInvalidInput [FR-045]", err)
	}
	v, err := combat.AntimagicField{}.Blocks(combat.ActivityCastSpell, false, []combat.SpellTargetRef{goblin})
	if err != nil {
		t.Fatal(err)
	}
	blocked := v.(combat.AntimagicBlocks)
	out := blocked.Targets()
	out[0] = dspTargetRef(t, "changed")
	if blocked.Targets()[0] != goblin {
		t.Error("changing a returned target list changed the verdict [FR-045 immutability]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-046 - for an item property the target list is ignored, so duplicates do not refuse it (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: unhappy (rule interaction).
func TestAntimagicFieldItemPropertyIgnoresTargets(t *testing.T) {
	t.Parallel()
	goblin := dspTargetRef(t, "goblin")
	v, err := combat.AntimagicField{}.Blocks(combat.ActivityUseMagicItemProperty, false, []combat.SpellTargetRef{goblin, goblin})
	if err != nil {
		t.Fatalf("Blocks error = %v, want the target list ignored [FR-046]", err)
	}
	if verdictName(t, v) != "permits" {
		t.Errorf("verdict = %s, want permits [FR-046]", verdictName(t, v))
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-048 - a Counterspell cast from inside the aura is a spell cast and is blocked like any other (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: unhappy (rule interaction).
func TestAntimagicFieldBlocksCounterspellCastInside(t *testing.T) {
	t.Parallel()
	v, err := combat.AntimagicField{}.Blocks(combat.ActivityCastSpell, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if verdictName(t, v) != "blocked:actor_inside_aura" {
		t.Errorf("verdict = %s, want the cast blocked [FR-048]", verdictName(t, v))
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-049 and FR-052 - suppression holds no state: Status returns functioning again once the aura is gone, and the spell is unchanged (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: unhappy (state error).
func TestAntimagicFieldSuppressionHoldsNoState(t *testing.T) {
	t.Parallel()
	spell := dspSpell(t, dspSpec{effect: "e", spell: "haste", level: 3, concentration: true, caster: "wizard"})
	before := spell
	if got := (combat.AntimagicField{}).Status(spell, true); got != combat.Suppressed {
		t.Fatalf("inside the aura Status = %v, want suppressed [FR-049]", got)
	}
	if got := (combat.AntimagicField{}).Status(spell, false); got != combat.Functioning {
		t.Errorf("after the aura Status = %v, want functioning [FR-049]", got)
	}
	if !spell.Equal(before) {
		t.Error("Status changed the spell [FR-049]")
	}
}
