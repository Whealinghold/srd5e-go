package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-001 - an OngoingSpell is an OngoingEffect of category spell with a spell source; anything else is refused (SRD 5.2.1, "Spell", p. 188).
// Path: per case.
func TestNewOngoingSpellNeedsASpellEffect(t *testing.T) {
	t.Parallel()
	spell := dspSpell(t, dspSpec{effect: "bless-1", spell: "bless", level: 1})
	if spell.SpellID() != "bless" || spell.IsZero() {
		t.Fatalf("spell built from a spell effect has id %q, zero %v [FR-001]", spell.SpellID(), spell.IsZero())
	}
	_, err := combat.NewOngoingSpell(combat.OngoingSpellParams{})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewOngoingSpell(zero effect) error = %v, want ErrInvalidInput [FR-001]", err)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-002 - CastAtLevel is the level the spell was cast at, a cantrip is 0 and an upcast keeps its higher level (SRD 5.2.1, "Using a Higher-Level Spell Slot", p. 104).
// Path: per case.
func TestOngoingSpellKeepsCastAtLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		level                   int
	}{
		{"cantrip", "FR-002", "happy", 0},
		{"ritual at its own level", "FR-002", "happy", 1},
		{"upcast to level 5", "FR-002", "happy", 5},
		{"ninth level", "FR-002", "unhappy (boundary)", 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := dspSpell(t, dspSpec{effect: "e", spell: "bless", level: c.level})
			if got := s.CastAtLevel().Int(); got != c.level {
				t.Errorf("CastAtLevel() = %d, want %d [%s]", got, c.level, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-002 - a cast level outside 0 to 9 is refused (boundary, invalid input).
// Path: per case.
func TestNewOngoingSpellRefusesLevelOutsideZeroToNine(t *testing.T) {
	t.Parallel()
	for _, level := range []int{-1, 10} {
		eff := dspSpell(t, dspSpec{effect: "e", spell: "bless"}).Effect()
		_, err := combat.NewOngoingSpell(combat.OngoingSpellParams{
			Effect:      eff,
			CastAtLevel: content.SpellLevel(level),
			Origin:      combat.SpellOriginOrdinary,
			Remaining:   dspTimed(t, 60),
			Restriction: combat.DispelUnrestricted{},
		})
		if !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewOngoingSpell(level %d) error = %v, want ErrInvalidInput [FR-002]", level, err)
		}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-003 - an ongoing spell needs its remaining time, a restriction and a defined origin (invalid input).
// Path: unhappy (invalid input).
func TestNewOngoingSpellRefusesMissingFields(t *testing.T) {
	t.Parallel()
	eff := dspSpell(t, dspSpec{effect: "e", spell: "bless"}).Effect()
	base := combat.OngoingSpellParams{
		Effect:      eff,
		Origin:      combat.SpellOriginOrdinary,
		Remaining:   dspTimed(t, 60),
		Restriction: combat.DispelUnrestricted{},
	}
	noTime := base
	noTime.Remaining = combat.SpellTimeRemaining{}
	noRestriction := base
	noRestriction.Restriction = nil
	noOrigin := base
	noOrigin.Origin = 0
	for name, p := range map[string]combat.OngoingSpellParams{"no time": noTime, "no restriction": noRestriction, "no origin": noOrigin} {
		if _, err := combat.NewOngoingSpell(p); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: NewOngoingSpell error = %v, want ErrInvalidInput [FR-003]", name, err)
		}
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-004 - a spell MUST NOT name itself as its parent (invalid input).
// Path: unhappy (invalid input).
func TestNewOngoingSpellRefusesItselfAsParent(t *testing.T) {
	t.Parallel()
	eff := dspSpell(t, dspSpec{effect: "self", spell: "bless"}).Effect()
	_, err := combat.NewOngoingSpell(combat.OngoingSpellParams{
		Effect:      eff,
		Origin:      combat.SpellOriginOrdinary,
		Remaining:   dspTimed(t, 60),
		Restriction: combat.DispelUnrestricted{},
		PartOf:      dspEffectID(t, "self"),
	})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewOngoingSpell(part of itself) error = %v, want ErrInvalidInput [FR-004]", err)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-004 - equality covers every field, including RequiresConcentration; conditions are copied and unmodifiable (immutability).
// Path: per case.
func TestOngoingSpellEqualityCoversEveryField(t *testing.T) {
	t.Parallel()
	base := dspSpec{effect: "e", spell: "bless", level: 1, caster: "cleric", conditions: []content.ConditionKind{content.ConditionKind(1)}}
	same := dspSpell(t, base)
	cases := []struct {
		name, requirement, path string
		other                   dspSpec
		wantEqual               bool
	}{
		{"same fields", "FR-004", "happy", base, true},
		{"requires concentration differs", "FR-004", "unhappy (rule interaction)", dspSpecWith(base, func(s *dspSpec) { s.concentration = true }), false},
		{"caster differs", "FR-004", "unhappy (boundary)", dspSpecWith(base, func(s *dspSpec) { s.caster = "other" }), false},
		{"level differs", "FR-004", "unhappy (boundary)", dspSpecWith(base, func(s *dspSpec) { s.level = 2 }), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := same.Equal(dspSpell(t, c.other))
			if got != c.wantEqual {
				t.Errorf("Equal = %v, want %v [%s]", got, c.wantEqual, c.requirement)
			}
		})
	}
	conds := same.Conditions()
	if len(conds) != 1 {
		t.Fatalf("Conditions() length = %d, want 1 [FR-004]", len(conds))
	}
	conds[0] = content.ConditionKind(99)
	if got := same.Conditions()[0]; got != content.ConditionKind(1) {
		t.Errorf("Conditions() changed through a returned slice: %v [FR-004 immutability]", got)
	}
}

// dspSpecWith returns a copy of s with change applied.
func dspSpecWith(s dspSpec, change func(*dspSpec)) dspSpec {
	change(&s)
	return s
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-005 - two spells sharing an effect id in one registry are refused (invalid input).
// Path: unhappy (invalid input).
func TestOngoingSpellRegistryRefusesSharedEffectID(t *testing.T) {
	t.Parallel()
	target := dspTarget(t, "goblin")
	a := dspSpell(t, dspSpec{effect: "same", spell: "bless"})
	b := dspSpell(t, dspSpec{effect: "same", spell: "shield_of_faith"})
	_, err := combat.NewOngoingSpellRegistry([]combat.HeldSpell{dspHold(target, a), dspHold(target, b)})
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("NewOngoingSpellRegistry(shared id) error = %v, want ErrInvalidInput [FR-005]", err)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-005 - the registry rejects an entry with no target or no spell (invalid input; a readied spell has no target, FR-030).
// Path: unhappy (invalid input).
func TestOngoingSpellRegistryRefusesEntryWithoutTarget(t *testing.T) {
	t.Parallel()
	sp := dspSpell(t, dspSpec{effect: "held", spell: "bless"})
	if _, err := combat.NewOngoingSpellRegistry([]combat.HeldSpell{{Target: nil, Spell: sp}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("entry with no target error = %v, want ErrInvalidInput [FR-030]", err)
	}
	if _, err := combat.NewOngoingSpellRegistry([]combat.HeldSpell{{Target: dspTarget(t, "a")}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Fatalf("entry with no spell error = %v, want ErrInvalidInput [FR-030]", err)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-006 - SpellsOn returns the target's spells in registry order and an empty list for an unknown target (empty and minimal).
// Path: per case.
func TestOngoingSpellRegistrySpellsOnInRegistryOrder(t *testing.T) {
	t.Parallel()
	goblin, wolf := dspTarget(t, "goblin"), dspTarget(t, "wolf")
	first := dspSpell(t, dspSpec{effect: "first", spell: "bless"})
	second := dspSpell(t, dspSpec{effect: "second", spell: "shield_of_faith"})
	other := dspSpell(t, dspSpec{effect: "other", spell: "haste"})
	r := dspRegistry(t, dspHold(goblin, first), dspHold(wolf, other), dspHold(goblin, second))
	cases := []struct {
		name, requirement, path string
		target                  combat.DispelTarget
		want                    []combat.OngoingSpell
	}{
		{"two spells keep registry order", "FR-006", "happy", goblin, []combat.OngoingSpell{first, second}},
		{"a target with one spell", "FR-006", "happy", wolf, []combat.OngoingSpell{other}},
		{"a target the registry does not hold", "FR-006", "unhappy (empty and minimal)", dspTarget(t, "nobody"), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := r.SpellsOn(c.target)
			if len(got) != len(c.want) {
				t.Fatalf("SpellsOn length = %d, want %d [%s]", len(got), len(c.want), c.requirement)
			}
			for i := range got {
				if !got[i].Equal(c.want[i]) {
					t.Errorf("SpellsOn[%d] differs [%s]", i, c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-005 - the registry copies its input and returns copies (immutability); the zero registry is empty.
// Path: unhappy (immutability).
func TestOngoingSpellRegistryIsImmutable(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	held := []combat.HeldSpell{dspHold(goblin, dspSpell(t, dspSpec{effect: "a", spell: "bless"}))}
	r := dspRegistry(t, held...)
	held[0] = dspHold(dspTarget(t, "wolf"), dspSpell(t, dspSpec{effect: "b", spell: "haste"}))
	if got := r.SpellsOn(goblin); len(got) != 1 {
		t.Fatalf("changing the input changed the registry: %d spells on goblin, want 1 [FR-005]", len(got))
	}
	got := r.SpellsOn(goblin)
	got[0] = dspSpell(t, dspSpec{effect: "c", spell: "haste"})
	if again := r.SpellsOn(goblin); len(again) != 1 || again[0].Effect().ID() != dspEffectID(t, "a") {
		t.Errorf("writing a returned list changed the registry [FR-005]")
	}
	var zero combat.OngoingSpellRegistry
	if !zero.Equal(dspRegistry(t)) || len(zero.SpellsOn(goblin)) != 0 {
		t.Error("zero registry is not the empty registry [FR-005]")
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-010 - two castings of one spell are two entries, each with its own effect id (combining is the host's, SRD 5.2.1, "Combining Spell Effects", p. 106).
// Path: happy.
func TestOngoingSpellRegistryKeepsTwoCastingsOfOneSpell(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t,
		dspHold(goblin, dspSpell(t, dspSpec{effect: "bless-a", spell: "bless", caster: "cleric"})),
		dspHold(goblin, dspSpell(t, dspSpec{effect: "bless-b", spell: "bless", caster: "priest"})),
	)
	if got := len(r.SpellsOn(goblin)); got != 2 {
		t.Fatalf("SpellsOn length = %d, want 2 castings kept [FR-010]", got)
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-008 - a spell Antimagic Field suppresses keeps its remaining time: suppression reads no clock and changes no entry, and the host advances the time through AdvanceActiveEffects (SRD 5.2.1, "Antimagic Field", p. 109).
// Path: unhappy (rule interaction).
func TestSuppressedSpellKeepsItsRemainingTime(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	bless := dspSpell(t, dspSpec{effect: "bless", spell: "bless", level: 1, remaining: dspTimed(t, 60)})
	r := dspRegistry(t, dspHold(goblin, bless))
	if got := (combat.AntimagicField{}).Status(bless, true); got != combat.Suppressed {
		t.Fatalf("Status inside the aura = %v, want Suppressed [FR-008]", got)
	}
	held := r.SpellsOn(goblin)
	if len(held) != 1 {
		t.Fatalf("registry holds %d spells on the target, want 1 [FR-008]", len(held))
	}
	if secs, timed := held[0].Remaining().Seconds(); !timed || secs != 60 {
		t.Errorf("remaining after suppression = %d seconds (timed %v), want 60 unchanged [FR-008]", secs, timed)
	}
}
