package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestCurseRemovalMeansFromSpell(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		id         string
		want       combat.CurseRemovalMeans
		ok         bool
	}{
		{"Remove Curse", "happy", "remove_curse", combat.RemoveCurseSpell, true},
		{"Greater Restoration", "happy", "greater_restoration", combat.GreaterRestorationSpell, true},
		{"a different spell", "unhappy (boundary)", "dispel_magic", 0, false},
		{"id of the possession ending spell", "unhappy (rule interaction)", "dispel_evil_and_good", 0, false},
		{"unknown id", "unhappy (invalid input)", "no_such_spell", 0, false},
		{"the empty id", "unhappy (invalid input)", "", 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Requirement: FR-028, table 4.5a; SRD 5.2.1, "Curses", p. 179
			got, ok := combat.CurseRemovalMeansFromSpell(content.SpellID(c.id))
			if got != c.want || ok != c.ok {
				t.Errorf("CurseRemovalMeansFromSpell(%q) = %v, %v, want %v, %v [%s]", c.id, got, ok, c.want, c.ok, c.path)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-029 - MagicThatEndsCurses accepts every magical source and returns an ErrInvalidInput error for a non-magical (or zero) source (SRD 5.2.1, "Curses", p. 179; "Magical Effect", p. 185).
// Path: unhappy (invalid input).
func TestMagicThatEndsCurses(t *testing.T) {
	t.Parallel()
	for name, s := range map[string]combat.EffectSource{"spell": meSpell("x"), "item": meItem("x"), "labeled": meLabeled("x")} {
		if m, err := combat.MagicThatEndsCurses(s); err != nil || m != combat.OtherCurseEndingMagic {
			t.Errorf("%s: got %v, %v", name, m, err)
		}
	}
	for name, s := range map[string]combat.EffectSource{"non-magical": meNonMagical("x"), "zero": {}} {
		if m, err := combat.MagicThatEndsCurses(s); !errors.Is(err, srderr.ErrInvalidInput) || m != 0 {
			t.Errorf("%s: got %v, %v", name, m, err)
		}
	}
	for m, want := range map[combat.CurseRemovalMeans]string{combat.RemoveCurseSpell: "remove_curse", combat.GreaterRestorationSpell: "greater_restoration",
		combat.OtherCurseEndingMagic: "other_magic", 0: "unknown"} {
		if m.String() != want {
			t.Errorf("%d.String() = %q", int(m), m.String())
		}
	}
}

// Requirement: doc/specs/magical-effects.md FR-027 - a curse is identified only by its id, category and source: two curses differing only in source are not equal, and RemoveCurse reads nothing else.
// Path: happy.
func TestCurseIsIdentifiedByIDCategoryAndSource(t *testing.T) {
	t.Parallel()
	a := meEffect("c1", combat.OngoingEffectCurse, meLabeled("hag_curse"))
	if a == meEffect("c1", combat.OngoingEffectCurse, meItem("cursed_ring")) || a != meEffect("c1", combat.OngoingEffectCurse, meLabeled("hag_curse")) {
		t.Error("two curses must be equal exactly when id, category and source are equal")
	}
}

// Requirement: doc/specs/magical-effects.md FR-030, FR-033 and table 4.15a - removing a curse present in the list returns Applied whose effects lack it and keep every other entry in order, with one CurseRemoved carrying the creature, the effect id and the means, equal by value.
// Path: happy.
func TestRemoveCurseApplied(t *testing.T) {
	t.Parallel()
	bless, c1, other := meBless(), meCurse("c1"), meEffect("m", combat.OngoingEffectMagicalEffect, meItem("ring"))
	effects := meEffects(bless, c1, other)
	res := (combat.CurseRemoval{}).Remove(gid("hero"), effects, meEID("c1"), combat.RemoveCurseSpell)
	applied, ok := res.(combat.RemoveCurseApplied)
	if !ok {
		t.Fatalf("Remove = %#v, want RemoveCurseApplied", res)
	}
	if !slices.Equal(applied.Effects().All(), []combat.OngoingEffect{bless, other}) {
		t.Errorf("effects after = %v", applied.Effects().All())
	}
	evs := applied.Events()
	if len(evs) != 1 {
		t.Fatalf("events = %v", evs)
	}
	ev := evs[0].(combat.CurseRemoved)
	combattest.CheckEvent(t, ev, "curse_removed", gid("hero"))
	if ev.Effect() != meEID("c1") || ev.Means() != combat.RemoveCurseSpell {
		t.Errorf("event carries %v, %v", ev.Effect(), ev.Means())
	}
	again := (combat.CurseRemoval{}).Remove(gid("hero"), effects, meEID("c1"), combat.RemoveCurseSpell).(combat.RemoveCurseApplied)
	if again.Events()[0] != evs[0] {
		t.Error("equal inputs must give equal events")
	}
	if other := (combat.CurseRemoval{}).Remove(gid("hero"), effects, meEID("c1"), combat.GreaterRestorationSpell).(combat.RemoveCurseApplied); other.Events()[0] == evs[0] {
		t.Error("a different means must give a different event")
	}
	combattest.CheckEventsCopy(t, func() []combat.Event { return []combat.Event{applied.Events()[0]} })
	if !effects.Equal(meEffects(bless, c1, other)) {
		t.Error("the input list changed")
	}
}

// Requirement: doc/specs/magical-effects.md FR-031, FR-032 - an id not in the list is NotPresent and an effect that is not a curse is NotACurse carrying its category; each leaves the effects unchanged and emits nothing.
// Path: unhappy (state error).
func TestRemoveCurseRefusals(t *testing.T) {
	t.Parallel()
	effects := meEffects(meBless(), meCurse("c1"))
	cases := []struct {
		name     string
		creature combat.CombatantID
		id       string
		means    combat.CurseRemovalMeans
		want     combat.RemoveCombatantCurseResult
	}{
		{"id not in the list", gid("hero"), "nope", combat.RemoveCurseSpell, combat.RemoveCurseNotPresent{}},
		{"the effect is a spell", gid("hero"), "bless", combat.RemoveCurseSpell, combat.RemoveCurseNotACurse{}},
		{"a zero creature changes nothing", "", "c1", combat.RemoveCurseSpell, combat.RemoveCurseNotPresent{}},
		{"an undefined means changes nothing", gid("hero"), "c1", 0, combat.RemoveCurseNotPresent{}},
		{"a means past the last changes nothing", gid("hero"), "c1", combat.OtherCurseEndingMagic + 1, combat.RemoveCurseNotPresent{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := (combat.CurseRemoval{}).Remove(c.creature, effects, combat.OngoingEffectID(c.id), c.means)
			if !res.Effects().Equal(effects) {
				t.Errorf("effects changed to %v", res.Effects().All())
			}
			switch r := res.(type) {
			case combat.RemoveCurseNotPresent:
				if _, ok := c.want.(combat.RemoveCurseNotPresent); !ok {
					t.Errorf("got NotPresent, want %T", c.want)
				}
			case combat.RemoveCurseNotACurse:
				if _, ok := c.want.(combat.RemoveCurseNotACurse); !ok || r.Category() != combat.OngoingEffectSpell {
					t.Errorf("got NotACurse(%v), want %T", r.Category(), c.want)
				}
			default:
				t.Errorf("got %T, want %T", res, c.want)
			}
		})
	}
}

// Requirement: doc/specs/magical-effects.md FR-145 - RemoveCurse acts on a creature's OngoingEffects only and is independent of any possession state, which this package does not hold (checked by the export scan in internal/archtest); a creature with a curse and other effects loses only the curse.
// Path: unhappy (rule interaction).
func TestRemoveCurseLeavesOtherEffectsAlone(t *testing.T) {
	t.Parallel()
	effects := meEffects(meCurse("c1"), meEffect("p", combat.OngoingEffectMagicalEffect, meLabeled("possession_magic")))
	res := (combat.CurseRemoval{}).Remove(gid("hero"), effects, meEID("c1"), combat.OtherCurseEndingMagic).(combat.RemoveCurseApplied)
	if res.Effects().Len() != 1 {
		t.Errorf("effects = %v", res.Effects().All())
	}
	if _, ok := res.Effects().Find(meEID("p")); !ok {
		t.Error("the unrelated effect must stay")
	}
}

// Requirement: doc/specs/magical-effects.md FR-032 - every category other than curse (spell, magical contagion, other Magical Effect) is NotACurse and carries its own category (SRD 5.2.1, "Curses", p. 179).
// Path: unhappy (state error).
func TestRemoveCurseRefusesEveryOtherCategory(t *testing.T) {
	t.Parallel()
	for cat, src := range map[combat.OngoingEffectCategory]combat.EffectSource{
		combat.OngoingEffectSpell: meSpell("bless"), combat.OngoingEffectMagicalContagion: meLabeled("rot"), combat.OngoingEffectMagicalEffect: meItem("ring"),
	} {
		effects := meEffects(meEffect("x", cat, src))
		res, ok := (combat.CurseRemoval{}).Remove(gid("hero"), effects, meEID("x"), combat.RemoveCurseSpell).(combat.RemoveCurseNotACurse)
		if !ok || res.Category() != cat || !res.Effects().Equal(effects) {
			t.Errorf("%v: got %#v", cat, res)
		}
	}
}
