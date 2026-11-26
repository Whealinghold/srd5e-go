package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// concLabel names the variant of a Concentration outcome, with its reason or DC where it has one.
func concLabel(o combat.ConcentrationCheckOutcome) string {
	switch v := o.(type) {
	case combat.ConcentrationNotChecked:
		return "not checked"
	case combat.ConcentrationHeld:
		return "held DC " + string(rune('0'+int(v.DC())/10)) + string(rune('0'+int(v.DC())%10))
	case combat.ConcentrationCheckFailed:
		return "failed DC " + string(rune('0'+int(v.DC())/10)) + string(rune('0'+int(v.DC())%10))
	case combat.ConcentrationEndedWithoutSave:
		return "ended " + v.Reason().String()
	}
	return "unknown"
}

// Requirement: doc/specs/combat-action-resolution.md 4.13.1 (FR-078 to FR-082, FR-131, FR-132, FR-158) - the Concentration of the damaged combatant: not checked (no tracking, none in progress, damage that cannot break it, damage reduced to 0), the DC rows (10, 11, 30), a save that holds and one that fails, an end without a save (0 Hit Points, death), damage fully absorbed by Temporary Hit Points still due, Exhaustion on the save and Advantage on it; one 1d20 (2d20 under Advantage) or none.
// Path: per case.
func TestDealDamageToCombatantConcentration(t *testing.T) {
	t.Parallel()
	const spec = resSpec
	monster := func(hp int) combat.Combatant {
		return encC(t, "b", combat.SideB, status(t, hp, hp, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities()))
	}
	startWith := func(b combat.Combatant) combat.Encounter {
		e, _ := encStart(t, encPC(t, "a", combat.SideA, 20, 20), b)
		return e
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		in                      combat.ConcentrationInput
		damage                  int
		faces                   []int
		want                    string
		dice                    []string
		spellcasting            string // "kept" or "ended" for a TracksConcentration
		kinds                   []string
	}{
		{"no tracking", spec + "FR-078 step 4", "happy", concTarget(t, 20, 20), combat.NoConcentrationTracking{}, 9, nil, "not checked", nil, "", nil},
		{"tracking, none in progress", spec + "FR-078 step 4, FR-079", "happy", concTarget(t, 20, 20), concTracks(t, false, 0, 0), 9, nil, "not checked", nil, "kept", nil},
		{"damage cannot break, 9 damage", spec + "FR-158", "unhappy (rule interaction)", concTarget(t, 20, 20), concTracks(t, true, 0, combat.DamageCannotBreakConcentration), 9, nil, "not checked", nil, "kept", nil},
		{"damage cannot break, to 0 Hit Points", spec + "FR-158, FR-078 step 2", "unhappy (rule interaction)", concTarget(t, 5, 20), concTracks(t, true, 0, combat.DamageCannotBreakConcentration), 9, nil, "ended incapacitated", nil, "ended",
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied", "movement_speeds_review_became_due", "concentration_checked"}},
		{"DC 10, save holds", spec + "FR-131, 4.13.1 DC rows", "unhappy (boundary)", concTarget(t, 20, 20), concTracks(t, true, 0, 0), 9, []int{10}, "held DC 10", []string{"1d20"}, "kept", []string{"damage_taken", "concentration_checked"}},
		{"DC 10, save one below", spec + "FR-131, 4.13.1 DC rows", "unhappy (boundary)", concTarget(t, 20, 20), concTracks(t, true, 0, 0), 9, []int{9}, "failed DC 10", []string{"1d20"}, "ended", []string{"damage_taken", "concentration_checked"}},
		{"DC 11", spec + "FR-131", "unhappy (boundary)", startWith(encC(t, "b", combat.SideB, pcStatus(t, 100, 100, 0))), concTracks(t, true, 0, 0), 22, []int{11}, "held DC 11", []string{"1d20"}, "kept", nil},
		{"DC 30 at the cap", spec + "FR-131", "unhappy (boundary)", startWith(encC(t, "b", combat.SideB, pcStatus(t, 200, 200, 0))), concTracks(t, true, 10, 0), 70, []int{20}, "held DC 30", []string{"1d20"}, "kept", nil},
		{"dead", spec + "FR-078 step 1", "unhappy (rule interaction)", startWith(monster(7)), concTracks(t, true, 0, 0), 9, nil, "ended dead", nil, "ended",
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died", "concentration_checked"}},
		{"absorbed by Temporary Hit Points", spec + "FR-081, FR-132, Q-10", "unhappy (boundary)", startWith(encC(t, "b", combat.SideB, pcStatus(t, 20, 20, 5))), concTracks(t, true, 0, 0), 5, []int{10}, "held DC 10", []string{"1d20"}, "kept", nil},
		{"damage reduced to 0 by Immunity", spec + "FR-081", "unhappy (boundary)", startWith(encC(t, "b", combat.SideB, status(t, 20, 20, 0, combat.Alive{}, combat.FallsUnconscious, combat.ImmuneToAllDamage()))), concTracks(t, true, 0, 0), 9, nil, "not checked", nil, "kept", nil},
		{"Exhaustion level 2 lowers the save", spec + "FR-030, 4.13.1 roll modifiers", "unhappy (rule interaction)", concTarget(t, 20, 20, condLevel(2)), concTracks(t, true, 0, 0), 9, []int{12}, "failed DC 10", []string{"1d20"}, "ended", nil},
		{"Advantage keeps the higher die", spec + "FR-080, 4.13.1 roll modifiers", "unhappy (rule interaction)", concTarget(t, 20, 20), concTracks(t, true, 0, 0, modifier.Advantage{}), 9, []int{4, 12}, "held DC 10", []string{"2d20"}, "kept", nil},
		{"Advantage and Disadvantage together", spec + "FR-080; SRD 5.2.1, \"Advantage/Disadvantage\", p. 8", "unhappy (rule interaction)", concTarget(t, 20, 20), concTracks(t, true, 0, 0, modifier.Advantage{}, modifier.Disadvantage{}), 9, []int{12}, "held DC 10", []string{"1d20"}, "kept", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.e
			res, dice := concDeal(t, c.e, "b", c.damage, c.in, c.faces...)
			got := resAs[combat.DealDamageToCombatantCompleted](t, res)
			if label := concLabel(got.Concentration()); label != c.want {
				t.Errorf("Concentration = %q, want %q [%s]", label, c.want, c.requirement)
			}
			if !slices.Equal(dice, c.dice) {
				t.Errorf("dice = %v, want %v [%s]", dice, c.dice, c.requirement)
			}
			if c.kinds != nil && !slices.Equal(resKinds(got.Events()), c.kinds) {
				t.Errorf("events = %v, want %v [%s]", resKinds(got.Events()), c.kinds, c.requirement)
			}
			state, tracked := got.TargetSpellcasting()
			_, going := state.Concentration()
			switch c.spellcasting {
			case "":
				if tracked {
					t.Errorf("a command without tracking returned a SpellcastingState [FR-079]")
				}
			case "kept":
				if !tracked || going != (c.name != "tracking, none in progress") {
					t.Errorf("SpellcastingState tracked %v concentrating %v, want the state as it came [FR-079]", tracked, going)
				}
			case "ended":
				if !tracked || going {
					t.Errorf("SpellcastingState tracked %v concentrating %v, want Concentration ended [FR-079]", tracked, going)
				}
			}
			if !before.Equal(c.e) {
				t.Errorf("the encounter that came in was altered [FR-004]")
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-079, FR-004 - the outcome carries the break transition's events (ConcentrationEnded with the reason, failed save or Incapacitated), the new state is a new value and the state the host passed in is not altered.
// Path: unhappy (immutability).
func TestConcentrationOutcomeCarriesTheBreakEventsAndLeavesTheInputAlone(t *testing.T) {
	t.Parallel()
	in := concTracks(t, true, 0, 0)
	res, _ := concDeal(t, concTarget(t, 20, 20), "b", 9, in, 3)
	failed, ok := resAs[combat.DealDamageToCombatantCompleted](t, res).Concentration().(combat.ConcentrationCheckFailed)
	if !ok {
		t.Fatal("want ConcentrationCheckFailed")
	}
	ended, ok := failed.Events()[0].(character.ConcentrationEnded)
	if !ok || ended.Reason() != character.EndedFailedSave {
		t.Errorf("events = %#v, want ConcentrationEnded with the failed save reason", failed.Events())
	}
	failed.Events()[0] = nil
	if failed.Events()[0] == nil {
		t.Errorf("Events returned the stored slice")
	}
	if _, going := in.State().Concentration(); !going {
		t.Errorf("the state the host passed in lost its Concentration [FR-004]")
	}
	if t3, ok := failed.Save().(combat.TestRolled); !ok || t3.Outcome() != d20.Failure {
		t.Errorf("save = %#v, want a rolled failure", failed.Save())
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-157 - the constructor of ConcentrationEndedWithoutSave rejects every reason other than Incapacitated and dead; NewTracksConcentration rejects a zero state, a nil circumstance and an undefined rule.
// Path: per case.
func TestConcentrationConstructors(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement, path string
		reason                  character.ConcentrationBreakReason
		ok                      bool
	}{
		{"incapacitated", resSpec + "FR-157", "happy", character.BreakIncapacitated, true},
		{"dead", resSpec + "FR-157", "happy", character.BreakDead, true},
		{"ended", resSpec + "FR-157", "unhappy (invalid input)", character.BreakEnded, false},
		{"failed save", resSpec + "FR-157", "unhappy (invalid input)", character.BreakFailedSave, false},
		{"another effect", resSpec + "FR-157", "unhappy (invalid input)", character.BreakAnotherEffect, false},
		{"zero", resSpec + "FR-157", "unhappy (invalid input)", 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := combat.NewConcentrationEndedWithoutSave(c.reason, nil)
			if (err == nil) != c.ok || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("NewConcentrationEndedWithoutSave(%v) = %v, want ok %v [%s]", c.reason, err, c.ok, c.requirement)
			}
		})
	}
	mods := d20.NewRollModifiers(modifier.New(0))
	for name, build := range map[string]func() (combat.TracksConcentration, error){
		"zero state": func() (combat.TracksConcentration, error) {
			return combat.NewTracksConcentration(character.SpellcastingState{}, mods, nil, 0)
		},
		"nil circumstance": func() (combat.TracksConcentration, error) {
			return combat.NewTracksConcentration(concState(t, true), mods, []modifier.RollCircumstance{nil}, 0)
		},
		"undefined rule": func() (combat.TracksConcentration, error) {
			return combat.NewTracksConcentration(concState(t, true), mods, nil, 9)
		},
	} {
		if _, err := build(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("NewTracksConcentration(%s) = %v, want ErrInvalidInput [FR-158]", name, err)
		}
	}
	cmd, err := combat.NewDealDamageToCombatantCommand(concTarget(t, 20, 20), cid("b"), []combat.Damage{combat.MustDamage(1, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	for name, in := range map[string]combat.ConcentrationInput{"nil": nil, "zero tracking": combat.TracksConcentration{}} {
		if _, err := cmd.WithTargetConcentration(in); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("WithTargetConcentration(%s) = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-079, FR-004 - the outcomes expose their save and their events as copies: a made save is the rolled check, and the events of an end without a save cannot be changed through the accessor.
// Path: unhappy (immutability).
func TestConcentrationOutcomesExposeCopies(t *testing.T) {
	t.Parallel()
	held := resAs[combat.DealDamageToCombatantCompleted](t, func() combat.DealDamageToCombatantResult {
		r, _ := concDeal(t, concTarget(t, 20, 20), "b", 9, concTracks(t, true, 0, 0), 15)
		return r
	}()).Concentration()
	if h, ok := held.(combat.ConcentrationHeld); !ok || h.Save().Outcome() != d20.Success {
		t.Fatalf("outcome = %#v, want a held save", held)
	}
	ended := resAs[combat.DealDamageToCombatantCompleted](t, func() combat.DealDamageToCombatantResult {
		r, _ := concDeal(t, concTarget(t, 5, 20), "b", 9, concTracks(t, true, 0, 0))
		return r
	}()).Concentration()
	e, ok := ended.(combat.ConcentrationEndedWithoutSave)
	if !ok || len(e.Events()) == 0 || e.Reason() != character.BreakIncapacitated {
		t.Fatalf("outcome = %#v, want an end without a save", ended)
	}
	e.Events()[0] = nil
	if e.Events()[0] == nil {
		t.Errorf("Events returned the stored slice")
	}
}
