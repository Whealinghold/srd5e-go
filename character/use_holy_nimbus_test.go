package character_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func useHolyNimbus(t *testing.T, opts ...character.Option) character.UseHolyNimbus {
	t.Helper()
	return must(character.NewUseHolyNimbus(pldProgression(t), opts...))
}

func nimbusSpent(t *testing.T) character.Character {
	t.Helper()
	return pldWith(t, pldPaladin(t, 20), pldState(t, 20, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 2, HolyNimbus: character.LongRestUseSpent}))
}

// Requirement: doc/specs/class-paladin.md FR-089 to FR-095, Table N (SRD 5.2.1, "Level 20: Holy Nimbus", p. 57) - activating Holy Nimbus returns a Bonus Action effect lasting 10 minutes with Holy Ward (Fiend, Undead),
// Radiant damage equal to the Charisma modifier plus the Proficiency Bonus (the four rows of Table N) and Sunlight, sets the use to spent, makes the effect active, spends no Channel Divinity use and emits HolyNimbusActivated.
// Path: per case (Table N): happy / unhappy (boundary).
func TestUseHolyNimbusActivate(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label     string
		cha, want int
	}{{"lowest modifier", -5, 1}, {"zero", 0, 6}, {"three", 3, 9}, {"five", 5, 11}} {
		ch := pldWith(t, pldPaladin(t, 20), pldState(t, 20, character.PaladinStateParts{LayOnHandsRemaining: 100, ChannelDivinityRemaining: 0}))
		res, err := useHolyNimbus(t).Run(context.Background(), character.UseHolyNimbusCommand{Character: ch, Subclass: oath, Action: character.HolyNimbusActivate,
			Charisma: modifierOf(t, c.cha), Proficiency: character.ProficiencyBonusFor(character.MustLevel(20))})
		a, ok := res.(character.UseHolyNimbusApplied)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.label, res, err)
		}
		eff, hasEffect := a.Effect()
		damage, dmgActive := eff.RadiantDamage()
		ward, wardActive := eff.HolyWard()
		s := pldStateOf(t, a.Character())
		if !hasEffect || damage != c.want || !dmgActive || !wardActive || !eff.AuraActive() || !eff.Sunlight() || eff.ActionCost() != character.PaladinBonusAction || eff.DurationMinutes() != 10 ||
			!slices.Equal(ward, []content.CreatureType{content.CreatureTypeFiend, content.CreatureTypeUndead}) {
			t.Errorf("%s: effect = %+v (damage %d)", c.label, eff, damage)
		}
		if s.HolyNimbusUse() != character.LongRestUseSpent || !s.HolyNimbusActive() || s.ChannelDivinityRemaining() != 0 || !s.Equal(a.State()) {
			t.Errorf("%s: state = %+v", c.label, s)
		}
		if _, hasSpellcasting := a.Spellcasting(); hasSpellcasting {
			t.Errorf("%s: an activation returned a SpellcastingState", c.label)
		}
		ev, ok := a.Events()[0].(character.HolyNimbusActivated)
		if len(a.Events()) != 1 || !ok || ev.CharacterID() != ch.ID() {
			t.Errorf("%s: events = %#v", c.label, a.Events())
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-095, D4 - while the Paladin is Incapacitated the Holy Nimbus effect reports its aura-based effects (Radiant damage, Sunlight, Holy Ward) as inactive.
// Path: unhappy (rule interaction).
func TestUseHolyNimbusIncapacitated(t *testing.T) {
	t.Parallel()
	res := must(useHolyNimbus(t).Run(context.Background(), character.UseHolyNimbusCommand{Character: pldPaladin(t, 20), Subclass: oath, Action: character.HolyNimbusActivate, Charisma: modifierOf(t, 3),
		Proficiency: character.ProficiencyBonusFor(character.MustLevel(20)), Incapacitated: true}))
	eff, _ := res.(character.UseHolyNimbusApplied).Effect()
	if _, active := eff.RadiantDamage(); active || eff.AuraActive() || eff.Sunlight() {
		t.Errorf("effect = %+v, want inactive", eff)
	}
	if _, active := eff.HolyWard(); active {
		t.Error("Holy Ward is active")
	}
}

// Requirement: doc/specs/class-paladin.md FR-089, FR-091, FR-092 - Holy Nimbus is rejected with feature_not_gained at level 19 or without the Oath, a second activation with already_used_since_long_rest, a restore while the use is available
// with not_expended, and a restore from a pool without a level 5 slot with no_level_5_slot_available; nothing changes and no event is emitted.
// Path: per case (unhappy: boundary levels 19/20, state error).
func TestUseHolyNimbusRejected(t *testing.T) {
	t.Parallel()
	slots := castingState(t, classLevelsOf(t, "paladin", 20), nil)
	empty := castingState(t, classLevelsOf(t, "paladin", 5), nil)
	for _, c := range []struct {
		name string
		char character.Character
		cmd  character.UseHolyNimbusCommand
		want []string
	}{
		{"level 19", pldPaladin(t, 19), character.UseHolyNimbusCommand{Subclass: oath, Action: character.HolyNimbusActivate}, []string{"feature_not_gained"}},
		{"no Oath", pldPaladin(t, 20), character.UseHolyNimbusCommand{Action: character.HolyNimbusActivate}, []string{"feature_not_gained"}},
		{"second activation", nimbusSpent(t), character.UseHolyNimbusCommand{Subclass: oath, Action: character.HolyNimbusActivate}, []string{"already_used_since_long_rest"}},
		{"restore while available", pldPaladin(t, 20), character.UseHolyNimbusCommand{Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: slots, Pool: character.PoolSpellcasting}, []string{"not_expended"}},
		{"restore at level 19", pldPaladin(t, 19), character.UseHolyNimbusCommand{Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: slots, Pool: character.PoolSpellcasting}, []string{"feature_not_gained", "not_expended"}},
		{"no level 5 slot in the pool", nimbusSpent(t), character.UseHolyNimbusCommand{Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: empty, Pool: character.PoolSpellcasting}, []string{"no_level_5_slot_available"}},
		{"the wrong pool", nimbusSpent(t), character.UseHolyNimbusCommand{Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: slots, Pool: character.PoolPactMagic}, []string{"no_level_5_slot_available"}},
	} {
		cmd, before := c.cmd, c.char
		cmd.Character = c.char
		res, err := useHolyNimbus(t).Run(context.Background(), cmd)
		rj, ok := res.(character.UseHolyNimbusRejected)
		if err != nil || !ok || !slices.Equal(pldReasons(rj.Reasons()), c.want) || !c.char.Equal(before) {
			t.Errorf("%s: result = %#v, %v; want reasons %v", c.name, res, err, c.want)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-092 - restoring the use expends exactly one level 5 slot of the chosen pool through SpellcastingState.ExpendSlot (the spellcasting pool, and a level 5 slot held only in Pact Magic), returns
// the new SpellcastingState, sets the use to available, keeps the effect state and emits SpellSlotExpended then HolyNimbusRestored.
// Path: per case (happy / unhappy: collaborator).
func TestUseHolyNimbusRestoreWithSlot(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		state character.SpellcastingState
		pool  character.SpellSlotPool
	}{
		{"spellcasting pool", castingState(t, classLevelsOf(t, "paladin", 20), nil), character.PoolSpellcasting},
		{"pact magic only", castingState(t, classLevelsOf(t, "warlock", 9), nil), character.PoolPactMagic},
	} {
		ch := nimbusSpent(t)
		res, err := useHolyNimbus(t).Run(context.Background(), character.UseHolyNimbusCommand{Character: ch, Subclass: oath, Action: character.HolyNimbusRestoreWithSlot, Spellcasting: c.state, Pool: c.pool})
		a, ok := res.(character.UseHolyNimbusApplied)
		if err != nil || !ok {
			t.Fatalf("%s: %#v, %v", c.name, res, err)
		}
		next, hasState := a.Spellcasting()
		if _, hasEffect := a.Effect(); hasEffect || !hasState {
			t.Errorf("%s: effect/state flags wrong", c.name)
		}
		five := character.MustSlotLevel(5)
		if next.Slots(c.pool).Remaining(five) != c.state.Slots(c.pool).Remaining(five)-1 {
			t.Errorf("%s: level 5 slots %d -> %d", c.name, c.state.Slots(c.pool).Remaining(five), next.Slots(c.pool).Remaining(five))
		}
		s := pldStateOf(t, a.Character())
		if s.HolyNimbusUse() != character.LongRestUseAvailable || s.ChannelDivinityRemaining() != 2 || !s.Equal(a.State()) {
			t.Errorf("%s: state = %+v", c.name, s)
		}
		_, expended := a.Events()[0].(character.SpellSlotExpended)
		restored, ok := a.Events()[1].(character.HolyNimbusRestored)
		if len(a.Events()) != 2 || !expended || !ok || restored.CharacterID() != ch.ID() {
			t.Errorf("%s: events = %#v", c.name, a.Events())
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-101, FR-105, FR-109, FR-115 - NotAPaladin for a Fighter; an undefined action, a restore without a SpellcastingState or with an invalid pool, a zero Character or use case and a cancelled
// context are errors before any signal; recording telemetry gives the same result and one holy_nimbus span.
// Path: per case (unhappy: invalid input).
func TestUseHolyNimbusEdges(t *testing.T) {
	t.Parallel()
	rec, opts := fgtTelemetry()
	if res, _ := useHolyNimbus(t, opts...).Run(context.Background(), character.UseHolyNimbusCommand{Character: fgtFighter(t, 20), Action: character.HolyNimbusActivate}); res == nil {
		t.Fatal("no result")
	} else if _, ok := res.(character.UseHolyNimbusNotAPaladin); !ok {
		t.Errorf("a Fighter = %#v", res)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUseHolyNimbus, "holy_nimbus", "not_a_paladin"))
	rec, opts = fgtTelemetry()
	if _, err := useHolyNimbus(t, opts...).Run(context.Background(), character.UseHolyNimbusCommand{Character: pldPaladin(t, 20), Subclass: oath, Action: character.HolyNimbusActivate, Charisma: modifierOf(t, 3)}); err != nil {
		t.Fatal(err)
	}
	checkFeatureCall(t, rec, pldWant(character.SpanUseHolyNimbus, "holy_nimbus", ""))
	slots := castingState(t, classLevelsOf(t, "paladin", 20), nil)
	for name, cmd := range map[string]character.UseHolyNimbusCommand{
		"undefined action":      {Character: pldPaladin(t, 20)},
		"restore without state": {Character: nimbusSpent(t), Action: character.HolyNimbusRestoreWithSlot, Pool: character.PoolSpellcasting},
		"restore invalid pool":  {Character: nimbusSpent(t), Action: character.HolyNimbusRestoreWithSlot, Spellcasting: slots},
		"zero character":        {Action: character.HolyNimbusActivate},
		"foreign state":         {Character: pldForeign(t, pldPaladin(t, 20)), Action: character.HolyNimbusActivate},
	} {
		if res, err := useHolyNimbus(t).Run(context.Background(), cmd); res != nil || err == nil {
			t.Errorf("%s = %v, %v", name, res, err)
		} else {
			wantInvalid(t, err, name)
		}
	}
	if _, err := (character.UseHolyNimbus{}).Run(context.Background(), character.UseHolyNimbusCommand{Character: pldPaladin(t, 20), Action: character.HolyNimbusActivate}); err == nil {
		t.Error("the zero use case ran")
	}
	if _, err := character.NewUseHolyNimbus(character.PaladinProgression{}); err == nil {
		t.Error("the zero progression was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := useHolyNimbus(t).Run(ctx, character.UseHolyNimbusCommand{Character: pldPaladin(t, 20), Action: character.HolyNimbusActivate}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled = %v", err)
	}
}
