package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// mscStartedRig returns a rig whose monster has begun to cast the spell at ref.
func mscStartedRig(tb testing.TB, ref [2]int, name string) mscRig {
	tb.Helper()
	rig := mscNewRig(tb, mscMonster(tb, nil))
	res := rig.cast(mscList(tb, ref), mscSpells()[name])
	mscStarted(tb, res)
	return rig.then(tb, res.Out)
}

// Requirement: doc/specs/monster-spellcasting.md FR-022, FR-023, LC-1 - a ListCast of a spell with a casting time of 1 minute or more only starts: it returns the started variant with Concentration of kind casting and the minutes, spends no use and emits began then started; an action casting time finishes at once; an earlier Concentration ends first with the reason another effect (SRD 5.2.1, "Spellcasting", p. 257; "Longer Casting Times", p. 105; "Geas", p. 135; "Scrying", p. 160).
// Path: per case.
func TestLongerCastingStarts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		ref                     [2]int
		spell                   string
		holding                 bool
		minutes                 int
		events                  []string
	}{
		{"geas starts", "FR-022 LC-1", "happy", mscGeas, "geas", false, 1, []string{"monster_concentration_began", "monster_casting_started"}},
		{"scrying starts, 1/Day Each", "FR-022 LC-1", "happy", mscScrying, "scrying", false, 10, []string{"monster_concentration_began", "monster_casting_started"}},
		{"replaces Concentration", "FR-023 LC-1", "unhappy (rule interaction)", mscGeas, "geas", true, 1,
			[]string{"monster_concentration_ended", "monster_concentration_began", "monster_casting_started"}},
	}
	spells := mscSpells()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig := mscNewRig(t, mscMonster(t, nil))
			if c.holding {
				rig.state = mscConcentrating(t, "other_spell")
			}
			before := rig
			res := rig.cast(mscList(t, c.ref), spells[c.spell])
			tr := mscStarted(t, res).Transition
			mscUnchanged(t, before, rig)
			if got := mscKinds(tr.Events()); !equalStrings(got, c.events) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, monsterSpellcastingSpec+c.requirement)
			}
			casting, ref, going := tr.State().CastingInProgress()
			conc, held := tr.State().Concentration()
			if !going || casting.TotalMinutes() != c.minutes || casting.Spell() != content.SpellID(c.spell) || ref != mscRef(t, c.ref) ||
				casting.Source() != character.CastSourceFeatureGranted || !held || conc.Kind() != character.ConcentrationCasting ||
				conc.Subject() != character.NewConcentrationSubjectSpell(content.SpellID(c.spell)) {
				t.Errorf("state = casting %v (%v) %d minutes ref %v, Concentration %v %v; want the casting of %s", casting.Spell(), going, casting.TotalMinutes(), ref, conc.Kind(), held, c.spell)
			}
			if !tr.LimitedUse().Equal(rig.uses) || len(tr.LimitedUseEvents()) != 0 {
				t.Errorf("starting a casting spent a use [FR-022]")
			}
			if c.holding {
				if ended := tr.Events()[0].(combat.MonsterConcentrationEnded); ended.Reason() != character.EndedAnotherEffect {
					t.Errorf("earlier Concentration ended for %v, want another effect", ended.Reason())
				}
			}
			started := tr.Events()[len(tr.Events())-1].(combat.MonsterCastingStarted)
			if started.TotalMinutes() != c.minutes || started.Spell() != content.SpellID(c.spell) {
				t.Errorf("MonsterCastingStarted = %v, %d, want %s, %d", started.Spell(), started.TotalMinutes(), c.spell, c.minutes)
			}
		})
	}
	rig := mscNewRig(t, mscMonster(t, nil))
	mscPerformed(t, rig.cast(mscList(t, mscMindBlank), spells["mind_blank"]))
	rig.uses = luSpellSpent(t, rig.uses, mscRef(t, mscGeas))
	if got := mscRefused(t, rig.cast(mscList(t, mscGeas), spells["geas"])); got != combat.MonsterCastRefusalUsesExhausted {
		t.Errorf("starting a spent-out casting refusal = %v, want uses_exhausted", got)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-024 - while a casting is in progress any further cast is refused with casting_already_in_progress, a cantrip and a part cast included.
// Path: per case.
func TestCastDuringACasting(t *testing.T) {
	t.Parallel()
	rig := mscStartedRig(t, mscGeas, "geas")
	spells := mscSpells()
	before := rig
	for _, c := range []struct {
		name, requirement, path string
		res                     mscResult
	}{
		{"a cantrip", "FR-024", "unhappy (state error)", rig.cast(mscList(t, mscLight), spells["light"])},
		{"a part cast", "FR-024", "unhappy (state error)", rig.cast(mscPartCast(t, mscMistyStep, 0, combat.PartSpellcasting{}), spells["misty_step"])},
		{"a second longer casting", "FR-024", "unhappy (state error)", rig.cast(mscList(t, mscScrying), spells["scrying"])},
	} {
		if got := mscRefused(t, c.res); got != combat.MonsterCastRefusalAlreadyInProgress {
			t.Errorf("%s: refusal = %v, want casting_already_in_progress [%s]", c.name, got, c.requirement)
		}
	}
	mscUnchanged(t, before, rig)
}

// Requirement: doc/specs/monster-spellcasting.md FR-025, LC-1 - at the end of a turn with a casting in progress and no Magic action the casting fails (Concentration ends with the reason ended, no use is spent); with the Magic action, or no casting, the states come back equal with no event (SRD 5.2.1, "Spellcasting", p. 257).
// Path: per case.
func TestEndMagicActionTurn(t *testing.T) {
	t.Parallel()
	started := mscStartedRig(t, mscGeas, "geas")
	idle := mscNewRig(t, mscMonster(t, nil))
	effect := idle
	effect.state = mscConcentrating(t, "other_spell")
	cases := []struct {
		name, requirement, path string
		rig                     mscRig
		took                    bool
		events                  []string
		cleared                 bool
	}{
		{"action each turn", "FR-025 LC-1", "happy", started, true, nil, false},
		{"turn skipped", "FR-025 LC-1", "unhappy (state error)", started, false, []string{"monster_concentration_ended", "monster_casting_failed"}, true},
		{"no casting", "FR-025 LC-1", "unhappy (empty and minimal)", idle, false, nil, false},
		{"Concentration on an effect is not a casting", "FR-025", "unhappy (rule interaction)", effect, false, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := combat.MonsterSpellcastingRules{}.EndMagicActionTurn(c.rig.state, c.rig.uses, c.took)
			if got := mscKinds(tr.Events()); !equalStrings(got, c.events) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, monsterSpellcastingSpec+c.requirement)
			}
			if !tr.LimitedUse().Equal(c.rig.uses) || len(tr.LimitedUseEvents()) != 0 {
				t.Errorf("a turn change spent a use")
			}
			if !c.cleared {
				if tr.State() != c.rig.state {
					t.Errorf("state changed, want it equal")
				}
				return
			}
			if _, _, going := tr.State().CastingInProgress(); going || tr.State() == c.rig.state {
				t.Errorf("casting still in progress after a skipped turn")
			}
			if _, held := tr.State().Concentration(); held {
				t.Errorf("Concentration still held after a failed casting")
			}
			ended, failed := tr.Events()[0].(combat.MonsterConcentrationEnded), tr.Events()[1].(combat.MonsterCastingFailed)
			if ended.Reason() != character.EndedByCaster || failed.Reason() != character.EndedByCaster || failed.Spell() != "geas" ||
				ended.Subject() != character.NewConcentrationSubjectSpell("geas") {
				t.Errorf("ended %v %v, failed %v %v, want geas ended for reason ended", ended.Subject(), ended.Reason(), failed.Spell(), failed.Reason())
			}
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-026, FR-027, LC-1 - completing a casting re-judges the refusals, spends the use once, clears the casting and emits the cast; a spell that needs Concentration (and is not waived) continues as Concentration of kind effect with no event, any other ends the casting Concentration with the reason casting completed (SRD 5.2.1, "Longer Casting Times", p. 105; "Concentration", p. 179).
// Path: per case.
func TestCompleteCasting(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	spells["waived_long"] = mscSpellOf("waived_long", 3, mscConcentration, mscCastingMinutes(10))
	waivedRig := func(t *testing.T) mscRig {
		t.Helper()
		rig := mscSingle(t, luPerDayEach(1), mscEntry("waived_long", "no Concentration", 0), nil)
		return rig.then(t, mustStart(t, rig, mscFirst, spells["waived_long"]))
	}
	cases := []struct {
		name, requirement, path string
		rig                     func(*testing.T) mscRig
		ref                     [2]int
		spell                   string
		events                  []string
		holds                   bool // Concentration of kind effect afterwards
	}{
		{"complete", "FR-026 LC-1", "happy", mscGeasStarted, mscGeas, "geas",
			[]string{"monster_concentration_ended", "monster_spell_cast"}, false},
		{"Concentration spell done", "FR-027 LC-1", "happy", mscScryingStarted, mscScrying, "scrying",
			[]string{"monster_spell_cast"}, true},
		{"waived done", "FR-027 LC-1", "unhappy (rule interaction)", waivedRig, mscFirst, "waived_long",
			[]string{"monster_concentration_ended", "monster_spell_cast"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rig := c.rig(t)
			before := rig
			p := mscPerformed(t, rig.complete(mscList(t, c.ref), spells[c.spell]))
			mscUnchanged(t, before, rig)
			if got := mscKinds(p.Transition.Events()); !equalStrings(got, c.events) {
				t.Errorf("events = %v, want %v [%s]", got, c.events, monsterSpellcastingSpec+c.requirement)
			}
			if _, _, going := p.Transition.State().CastingInProgress(); going {
				t.Errorf("casting still in progress after completion")
			}
			conc, held := p.Transition.State().Concentration()
			if held != c.holds || (held && conc.Kind() != character.ConcentrationEffect) {
				t.Errorf("Concentration = %v %v, want held %v as effect", conc.Kind(), held, c.holds)
			}
			if c.events[0] == "monster_concentration_ended" {
				if r := p.Transition.Events()[0].(combat.MonsterConcentrationEnded).Reason(); r != character.EndedCastingCompleted {
					t.Errorf("Concentration ended for %v, want casting completed", r)
				}
			}
			if len(p.Transition.LimitedUseEvents()) != 1 || p.Transition.LimitedUse().AvailabilityOfSpell(mscRef(t, c.ref)) != combat.UseExhausted {
				t.Errorf("the use was not spent exactly once at completion")
			}
			if at, _ := p.Report.CastAtLevel(); at != 5 && c.spell != "waived_long" {
				t.Errorf("report cast at %d, want the spell's level 5", at)
			}
		})
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-026, FR-040, LC-1 - completion with nothing in progress is refused with casting_not_in_progress; a spell other than the one being cast is an error; a spent-out use, an incapacitated monster, a request of another entry or a part cast is refused and leaves the casting in progress and both states unchanged (SRD 5.2.1, "Longer Casting Times", p. 105).
// Path: per case.
func TestCompleteCastingRefusals(t *testing.T) {
	t.Parallel()
	spells := mscSpells()
	idle := mscNewRig(t, mscMonster(t, nil))
	started := mscStartedRig(t, mscScrying, "scrying")
	exhausted := started
	exhausted.uses = luSpellSpent(t, exhausted.uses, mscRef(t, mscScrying))
	blocked := started
	blocked.mc = mscContext(true, true, combat.ConditionActionAvailabilityNone)
	silenced := mscStartedRig(t, mscScrying, "scrying")
	silenced.mc = mscContext(false, true, combat.ConditionActionAvailabilityAvailable)
	cases := []struct {
		name, requirement, path string
		rig                     mscRig
		req                     combat.MonsterCastRequest
		spell                   string
		want                    combat.MonsterCastRefusal
	}{
		{"none in progress", "FR-026 LC-1", "unhappy (state error)", idle, mscList(t, mscGeas), "geas", combat.MonsterCastRefusalNotInProgress},
		{"exhausted at completion", "FR-026 LC-1", "unhappy (boundary)", exhausted, mscList(t, mscScrying), "scrying", combat.MonsterCastRefusalUsesExhausted},
		{"incapacitated at completion", "FR-026", "unhappy (state error)", blocked, mscList(t, mscScrying), "scrying", combat.MonsterCastRefusalCannotTakeActions},
		{"another group's entry for the same spell", "FR-040 LC-1", "unhappy (state error)", started, mscList(t, mscScryingToo), "scrying", combat.MonsterCastRefusalRefMismatch},
		{"a part cast", "FR-040 LC-1", "unhappy (state error)", started, mscPartCast(t, mscPhantasms, 0, combat.PartSpellcasting{}), "scrying", combat.MonsterCastRefusalRefMismatch},
		{"a reference outside the list", "FR-031", "unhappy (rule interaction)", started, mscList(t, [2]int{9, 9}), "scrying", combat.MonsterCastRefusalNotInSpellList},
		{"silenced at completion", "FR-026", "unhappy (state error)", silenced, mscList(t, mscScrying), "scrying", combat.MonsterCastRefusalVerbalMissing},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.rig
			res := c.rig.complete(c.req, spells[c.spell])
			if got := mscRefused(t, res); got != c.want {
				t.Errorf("refusal = %v, want %v [%s]", got, c.want, monsterSpellcastingSpec+c.requirement)
			}
			mscUnchanged(t, before, c.rig)
			if _, _, going := c.rig.state.CastingInProgress(); c.name != "none in progress" && !going {
				t.Errorf("the casting is no longer in progress after a refusal")
			}
		})
	}
	t.Run("a spell other than the one being cast", func(t *testing.T) {
		t.Parallel()
		geasStarted := mscStartedRig(t, mscGeas, "geas")
		for name, res := range map[string]mscResult{
			"a spell that sorts before it":                      started.complete(mscList(t, mscScrying), spells["geas"]),
			"a spell that sorts before it, entry of that spell": started.complete(mscList(t, mscGeas), spells["geas"]),
			"a spell that sorts after it":                       geasStarted.complete(mscList(t, mscScrying), spells["scrying"]),
		} {
			if !errors.Is(res.Err, srderr.ErrInvalidInput) {
				t.Errorf("%s: error = %v, want ErrInvalidInput", name, res.Err)
			}
		}
	})
}

// Requirement: doc/specs/monster-spellcasting.md FR-028, FR-016 - breaking Concentration for any of the five reasons ends it with that reason; with a casting in progress it also fails the casting and spends no use; with no Concentration, or a reason that is not one of the five, it returns the states equal and no events (SRD 5.2.1, "Concentration", p. 179).
// Path: per case.
func TestBreakConcentration(t *testing.T) {
	t.Parallel()
	started := mscStartedRig(t, mscGeas, "geas")
	effect := mscNewRig(t, mscMonster(t, nil))
	effect.state = mscConcentrating(t, "other_spell")
	idle := mscNewRig(t, mscMonster(t, nil))
	reasons := []struct {
		reason character.ConcentrationBreakReason
		end    character.ConcentrationEndReason
	}{
		{character.BreakIncapacitated, character.EndedIncapacitated}, {character.BreakDead, character.EndedDead}, {character.BreakEnded, character.EndedByCaster},
		{character.BreakFailedSave, character.EndedFailedSave}, {character.BreakAnotherEffect, character.EndedAnotherEffect},
	}
	for _, r := range reasons {
		t.Run("casting, "+r.reason.String(), func(t *testing.T) {
			t.Parallel()
			tr := combat.MonsterSpellcastingRules{}.BreakConcentration(started.state, started.uses, r.reason)
			if got := mscKinds(tr.Events()); !equalStrings(got, []string{"monster_concentration_ended", "monster_casting_failed"}) {
				t.Fatalf("events = %v, want ended then failed [FR-028]", got)
			}
			ended, failed := tr.Events()[0].(combat.MonsterConcentrationEnded), tr.Events()[1].(combat.MonsterCastingFailed)
			if ended.Reason() != r.end || failed.Reason() != r.end || failed.Spell() != "geas" {
				t.Errorf("reasons = %v, %v, want %v", ended.Reason(), failed.Reason(), r.end)
			}
			if _, _, going := tr.State().CastingInProgress(); going || !tr.LimitedUse().Equal(started.uses) {
				t.Errorf("casting in progress after the break, or a use spent")
			}
			if !slices.Equal(tr.State().Combatants(), []combat.CombatantID{mscID()}) {
				t.Errorf("the state is no longer the monster's")
			}
		})
	}
	t.Run("Concentration on an effect", func(t *testing.T) {
		t.Parallel()
		tr := combat.MonsterSpellcastingRules{}.BreakConcentration(effect.state, effect.uses, character.BreakFailedSave)
		if got := mscKinds(tr.Events()); !equalStrings(got, []string{"monster_concentration_ended"}) {
			t.Errorf("events = %v, want ended only", got)
		}
		if _, held := tr.State().Concentration(); held {
			t.Errorf("Concentration still held")
		}
	})
	for _, c := range []struct {
		name, requirement, path string
		rig                     mscRig
		reason                  character.ConcentrationBreakReason
	}{
		{"no Concentration", "FR-028", "unhappy (empty and minimal)", idle, character.BreakDead},
		{"reason zero", "FR-028 FR-016", "unhappy (invalid input)", effect, 0},
		{"reason past the five", "FR-028 FR-016", "unhappy (boundary)", effect, character.BreakAnotherEffect + 1},
		{"reason negative", "FR-028 FR-016", "unhappy (invalid input)", started, -1},
	} {
		tr := combat.MonsterSpellcastingRules{}.BreakConcentration(c.rig.state, c.rig.uses, c.reason)
		if tr.State() != c.rig.state || len(tr.Events()) != 0 || !tr.LimitedUse().Equal(c.rig.uses) {
			t.Errorf("%s: the states changed or events were emitted [%s]", c.name, c.requirement)
		}
	}
}

func mscGeasStarted(t *testing.T) mscRig {
	t.Helper()
	return mscStartedRig(t, mscGeas, "geas")
}

func mscScryingStarted(t *testing.T) mscRig {
	t.Helper()
	return mscStartedRig(t, mscScrying, "scrying")
}
