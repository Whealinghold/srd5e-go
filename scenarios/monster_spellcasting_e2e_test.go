package scenarios_test

import (
	"errors"
	"io/fs"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpCSpells returns the bundled spells by id.
func grpCSpells(t *testing.T) map[content.SpellID]content.Spell {
	t.Helper()
	raw, err := fs.ReadFile(data.Files(), "spells.json")
	if err != nil {
		t.Fatal(err)
	}
	set, err := content.JSONDecoder{}.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[content.SpellID]content.Spell{}
	for _, e := range set.Entries() {
		sp := e.(content.Spell)
		out[sp.ID()] = sp
	}
	return out
}

// The positions of the spells of grpCCaster, as (group, spell).
var (
	grpCFireBolt    = [2]int{0, 0}
	grpCHoldPerson  = [2]int{0, 1}
	grpCFireball    = [2]int{1, 0}
	grpCGeas        = [2]int{1, 1}
	grpCSelfOnly    = [2]int{2, 0}
	grpCStatedFive  = [2]int{3, 0}
	grpCStatedTwo   = [2]int{4, 0}
	grpCMistyStepAt = grpCRef(combat.MonsterPartBonusActions, 0)
)

const grpCCasterID = "caster"

// grpCCaster is a synthetic monster with a Spellcasting trait (Intelligence, printed save DC 14 and spell attack
// bonus +6; Intelligence 16 gives +3, so the formula with a Proficiency Bonus of +2 would give 13) and a Bonus
// Action, Misty Step 3/Day. The spells are the bundled SRD 5.2.1 ones; the entry names are the spell names.
func grpCCaster(t *testing.T, spells map[content.SpellID]content.Spell) content.Monster {
	t.Helper()
	entry := func(id content.SpellID, note string, castAt int) content.MonsterSpellEntry {
		return grpCMust(content.NewMonsterSpellEntry(spells[id].Name(), note, castAt))
	}
	group := func(f content.SpellFrequency, entries ...content.MonsterSpellEntry) content.MonsterSpellGroup {
		return grpCMust(content.NewMonsterSpellGroup(f, entries))
	}
	trait := grpCMust(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
		Ability: ability.Intelligence, SaveDC: 14, HasSaveDC: true, AttackBonus: 6, HasAttackBonus: true,
		Groups: []content.MonsterSpellGroup{
			group(content.SpellFrequencyAtWill{}, entry("fire_bolt", "", 0), entry("hold_person", "", 0)),
			group(grpCMust(content.NewSpellFrequencyPerDayEach(1)), entry("fireball", "", 0), entry("geas", "", 0)),
			group(content.SpellFrequencyAtWill{}, entry("shield_of_faith", "self only", 0)),
			group(content.SpellFrequencyAtWill{}, entry("fireball", "", 5)),
			group(content.SpellFrequencyAtWill{}, entry("fireball", "", 2)),
		},
	}))
	return grpCMonster(func(f *content.MonsterFields) {
		scores := map[ability.Ability]ability.Score{}
		saves := map[ability.Ability]int{}
		for _, a := range ability.All() {
			scores[a], saves[a] = ability.MustScore(10), 0
		}
		scores[ability.Intelligence], saves[ability.Intelligence] = ability.MustScore(16), 3
		f.Abilities = grpCMust(content.NewAbilityBlock(grpCMust(ability.NewScores(scores)), saves))
		f.Traits = []content.StatBlockPart{grpCMust(content.NewStatBlockPart("Spellcasting",
			content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Spellcasting: trait}))}
		f.BonusActions = []content.StatBlockPart{grpCUsage("Misty Step", grpCPerDay(3, 0))}
	})
}

// grpCRig is a caster with its two states and a context in which the monster can speak, gesture and act.
type grpCRig struct {
	t      *testing.T
	spells map[content.SpellID]content.Spell
	caster combat.MonsterSpellcaster
	state  combat.MonsterSpellcastingState
	uses   combat.LimitedUseState
	mc     combat.MonsterCastContext
}

func grpCNewRig(t *testing.T) grpCRig {
	t.Helper()
	spells := grpCSpells(t)
	m := grpCCaster(t, spells)
	id := grpCID(grpCCasterID)
	return grpCRig{
		t: t, spells: spells,
		caster: grpCMust(combat.NewMonsterSpellcaster(m, id)),
		state:  grpCMust(combat.NewMonsterSpellcastingState(id)),
		uses:   grpCMust(combat.StartLimitedUse(m, id, combat.OutsideLair)),
		mc:     grpCMust(combat.NewMonsterCastContext(true, true, combat.ConditionActionAvailabilityAvailable)),
	}
}

func (r grpCRig) list(at [2]int) combat.MonsterListCast {
	return combat.NewMonsterListCast(grpCMust(combat.NewSpellUseRef(at[0], at[1])), content.CastingTimeMode{})
}

func (r grpCRig) cast(req combat.MonsterCastRequest, spell content.SpellID, targets ...combat.CombatantID) (combat.MonsterCastOutcome, error) {
	return combat.MonsterSpellcastingRules{}.Cast(r.caster, r.state, r.uses, req, r.spells[spell], targets, r.mc)
}

func (r grpCRig) complete(req combat.MonsterCastRequest, spell content.SpellID, targets ...combat.CombatantID) (combat.MonsterCastOutcome, error) {
	return combat.MonsterSpellcastingRules{}.CompleteCasting(r.caster, r.state, r.uses, req, r.spells[spell], targets, r.mc)
}

func (r grpCRig) performed(out combat.MonsterCastOutcome, err error) combat.MonsterCastPerformed {
	r.t.Helper()
	if err != nil {
		r.t.Fatal(err)
	}
	p, ok := out.(combat.MonsterCastPerformed)
	if !ok {
		r.t.Fatalf("outcome = %#v, want a performed cast", out)
	}
	return p
}

func (r grpCRig) refusal(out combat.MonsterCastOutcome, err error) combat.MonsterCastRefusal {
	r.t.Helper()
	if err != nil {
		r.t.Fatal(err)
	}
	ref, ok := out.(combat.MonsterCastRefused)
	if !ok {
		r.t.Fatalf("outcome = %#v, want a refusal", out)
	}
	return ref.Reason
}

func (r grpCRig) then(p combat.MonsterSpellcastingTransition) grpCRig {
	r.state, r.uses = p.State(), p.LimitedUse()
	return r
}

func grpCSpellKinds(events []combat.MonsterSpellcastingEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// Requirement: doc/specs/monster-spellcasting.md FR-001, FR-002, FR-003, FR-006, FR-007, FR-008, FR-009, FR-010, FR-011, FR-021, FR-029, FR-030, FR-047, and doc/specs/monster-limited-use.md FR-043, FR-045, FR-046 - a monster casts a 1/Day Each spell at the spell's own level with the printed DC (not the formula's) and spell attack bonus, spending that spell's use and no slot; the report reaches the character side as a cast with the monster source, no class and no slot; At Will cantrips are cast twice in one turn; the next Long Rest returns the use.
// Path: happy.
func TestGrpCMonsterCastsFromItsSpellList(t *testing.T) {
	t.Parallel()
	rig := grpCNewRig(t)

	fireball := rig.performed(rig.cast(rig.list(grpCFireball), "fireball", grpCID("hero")))
	rep := fireball.Report
	if rep.Spell() != "fireball" || rep.ScalingLevel() != 3 {
		t.Errorf("report = spell %q scaling %d, want fireball at its own level 3", rep.Spell(), rep.ScalingLevel())
	}
	if at, ok := rep.CastAtLevel(); !ok || at != 3 {
		t.Errorf("cast at level = %d, %v, want 3", at, ok)
	}
	if dc, ok := rep.SaveDC(); !ok || int(dc) != 14 {
		t.Errorf("save DC = %v, %v, want the printed 14 (the formula gives 13)", dc, ok)
	}
	if _, ok := rep.AttackModifiers(); !ok {
		t.Errorf("the printed spell attack bonus is missing from the report")
	}
	if rep.Ability() != ability.Intelligence || rep.AbilityModifier().Value() != 3 {
		t.Errorf("ability = %v %v, want Intelligence +3", rep.Ability(), rep.AbilityModifier().Value())
	}
	if rep.TargetLimit() != combat.TargetLimit(2) { // unrestricted
		t.Errorf("target limit = %v, want unrestricted", rep.TargetLimit())
	}
	if !slices.Equal(grpCSpellKinds(fireball.Transition.Events()), []string{"monster_spell_cast"}) {
		t.Errorf("events = %v, want monster_spell_cast", grpCSpellKinds(fireball.Transition.Events()))
	}

	// The report reaches the character side with the monster source, no class and no slot.
	cr, err := rep.ToCastReport()
	if err != nil {
		t.Fatal(err)
	}
	_, hasClass := cr.Class()
	_, _, hasSlot := cr.Slot()
	dc, hasDC := cr.SaveDC()
	if cr.Source() != character.CastSourceMonster || hasClass || hasSlot || !hasDC || dc.Value() != 14 || cr.Spell() != "fireball" {
		t.Errorf("character report = source %v class %v slot %v DC %v/%v, want a monster cast with the printed DC 14 and nothing else", cr.Source(), hasClass, hasSlot, dc, hasDC)
	}

	// The 1/Day Each use is spent: a second Fireball is refused, a Geas (its own use) is not, and a Long Rest returns the use.
	rig = rig.then(fireball.Transition)
	if got := rig.refusal(rig.cast(rig.list(grpCFireball), "fireball")); got != combat.MonsterCastRefusalUsesExhausted {
		t.Errorf("a second Fireball = %v, want uses_exhausted", got)
	}
	if rig.uses.AvailabilityOfSpell(grpCMust(combat.NewSpellUseRef(1, 1))) != combat.UseAvailable {
		t.Errorf("Geas lost a use when Fireball was cast: each spell has its own uses")
	}
	rested := rig
	rested.uses = rig.uses.AfterRest(character.LongRest).State()
	rested.performed(rested.cast(rested.list(grpCFireball), "fireball"))

	// An At Will cantrip is cast twice in one turn: no slot, no "one spell with a slot per turn".
	for range 2 {
		bolt := rig.performed(rig.cast(rig.list(grpCFireBolt), "fire_bolt", grpCID("hero")))
		if at, ok := bolt.Report.CastAtLevel(); ok || bolt.Report.ScalingLevel() != 1 {
			t.Errorf("cantrip report = cast at %d (%v) scaling %d, want no cast level and scaling 1", at, ok, bolt.Report.ScalingLevel())
		}
		rig = rig.then(bolt.Transition)
	}

	// The entry that states a level 5 version is cast at level 5; an entry that states a level below the spell's own (2 for a level 3 spell) is a content defect, an error.
	stated := rig.performed(rig.cast(rig.list(grpCStatedFive), "fireball"))
	if at, _ := stated.Report.CastAtLevel(); at != 5 || stated.Report.ScalingLevel() != 5 {
		t.Errorf("the level 5 version cast at level %d scaling %d, want 5 and 5", at, stated.Report.ScalingLevel())
	}
	if _, err := rig.cast(rig.list(grpCStatedTwo), "fireball"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a stated level below the spell's error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-016, FR-020, FR-018, FR-019, FR-023, FR-028, FR-029, FR-014, FR-013, FR-015 - a Concentration spell begins Concentration on the monster; a second one ends it by another effect; Concentration breaks for a stated reason; a "self only" spell is refused when it affects another creature; an Incapacitated monster, one that cannot speak and one that cannot gesture cast nothing, and a Material component never refuses.
// Path: happy and unhappy (rule interaction).
func TestGrpCMonsterConcentrationRestrictionsAndComponents(t *testing.T) {
	t.Parallel()
	rig := grpCNewRig(t)
	hero := grpCID("hero")

	hold := rig.performed(rig.cast(rig.list(grpCHoldPerson), "hold_person", hero))
	if !slices.Contains(grpCSpellKinds(hold.Transition.Events()), "monster_concentration_began") {
		t.Fatalf("events = %v, want monster_concentration_began", grpCSpellKinds(hold.Transition.Events()))
	}
	rig = rig.then(hold.Transition)
	if _, held := rig.state.Concentration(); !held {
		t.Fatalf("the monster holds no Concentration after Hold Person")
	}
	broken := combat.MonsterSpellcastingRules{}.BreakConcentration(rig.state, rig.uses, character.BreakFailedSave)
	if _, held := broken.State().Concentration(); held || !slices.Contains(grpCSpellKinds(broken.Events()), "monster_concentration_ended") {
		t.Errorf("after a failed save: held %v events %v, want Concentration ended", held, grpCSpellKinds(broken.Events()))
	}
	if again := (combat.MonsterSpellcastingRules{}).BreakConcentration(broken.State(), broken.LimitedUse(), character.BreakFailedSave); len(again.Events()) != 0 {
		t.Errorf("breaking Concentration that is not held produced %v, want no event", grpCSpellKinds(again.Events()))
	}

	// "Self only": another creature as a target is refused; the monster itself is not.
	if got := rig.refusal(rig.cast(rig.list(grpCSelfOnly), "shield_of_faith", hero)); got != combat.MonsterCastRefusalSelfOnlyViolated {
		t.Errorf("a self-only spell on another creature = %v, want self_only_violated", got)
	}
	self := rig.performed(rig.cast(rig.list(grpCSelfOnly), "shield_of_faith", grpCID(grpCCasterID)))
	if self.Report.TargetLimit() != combat.TargetLimit(1) { // self only
		t.Errorf("target limit = %v, want self only", self.Report.TargetLimit())
	}

	// Components and actions: a monster that cannot speak, gesture or act casts nothing; a Material component never refuses.
	for _, c := range []struct {
		name    string
		speak   bool
		gesture bool
		actions combat.ConditionActionAvailability
		want    combat.MonsterCastRefusal
	}{
		{"cannot speak", false, true, combat.ConditionActionAvailabilityAvailable, combat.MonsterCastRefusalVerbalMissing},
		{"cannot gesture", true, false, combat.ConditionActionAvailabilityAvailable, combat.MonsterCastRefusalSomaticMissing},
		{"cannot act", true, true, combat.ConditionActionAvailabilityNone, combat.MonsterCastRefusalCannotTakeActions},
	} {
		r := rig
		r.state, r.uses = grpCMust(combat.NewMonsterSpellcastingState(grpCID(grpCCasterID))), grpCMust(combat.StartLimitedUse(grpCCaster(t, rig.spells), grpCID(grpCCasterID), combat.OutsideLair))
		r.mc = grpCMust(combat.NewMonsterCastContext(c.speak, c.gesture, c.actions))
		if got := r.refusal(r.cast(r.list(grpCFireball), "fireball")); got != c.want {
			t.Errorf("%s: refusal = %v, want %v", c.name, got, c.want)
		}
	}
	// Fireball has a Material component (a ball of bat guano and sulfur) and was cast above with the context that allows it.
}

// Requirement: doc/specs/monster-spellcasting.md FR-022, FR-023, FR-024, FR-025, FR-026, FR-027, FR-040 - a spell with a casting time of 1 minute only starts when it is cast: no use is spent, Concentration of the casting begins, and every other cast is refused until it ends; if the Magic action is not taken on the next turn the casting fails; if it is, the casting completes, spends the use and returns the performed report; the completion must be the same spell, from the same list entry.
// Path: happy and unhappy (state).
func TestGrpCMonsterCastsALongerCasting(t *testing.T) {
	t.Parallel()
	rig := grpCNewRig(t)
	geasRef := grpCMust(combat.NewSpellUseRef(grpCGeas[0], grpCGeas[1]))

	started, err := rig.cast(rig.list(grpCGeas), "geas", grpCID("hero"))
	if err != nil {
		t.Fatal(err)
	}
	st, ok := started.(combat.MonsterCastStarted)
	if !ok {
		t.Fatalf("outcome = %#v, want a started casting", started)
	}
	if !slices.Equal(grpCSpellKinds(st.Transition.Events()), []string{"monster_concentration_began", "monster_casting_started"}) {
		t.Errorf("events = %v, want concentration then casting started", grpCSpellKinds(st.Transition.Events()))
	}
	if rig.uses.AvailabilityOfSpell(geasRef) != combat.UseAvailable || st.Transition.LimitedUse().AvailabilityOfSpell(geasRef) != combat.UseAvailable {
		t.Errorf("starting the casting spent the use")
	}
	going := rig.then(st.Transition)
	if _, _, in := going.state.CastingInProgress(); !in {
		t.Fatalf("no casting in progress after the start")
	}
	if got := going.refusal(going.cast(going.list(grpCFireBolt), "fire_bolt")); got != combat.MonsterCastRefusalAlreadyInProgress {
		t.Errorf("a cantrip during a casting = %v, want casting_already_in_progress", got)
	}

	// The turn passes without the Magic action: the casting fails and no use is spent.
	failed := combat.MonsterSpellcastingRules{}.EndMagicActionTurn(going.state, going.uses, false)
	if _, _, in := failed.State().CastingInProgress(); in || !slices.Equal(grpCSpellKinds(failed.Events()), []string{"monster_concentration_ended", "monster_casting_failed"}) {
		t.Errorf("failed casting: events %v, want concentration ended then casting failed", grpCSpellKinds(failed.Events()))
	}
	if failed.LimitedUse().AvailabilityOfSpell(geasRef) != combat.UseAvailable {
		t.Errorf("a failed casting spent the use")
	}
	if kept := (combat.MonsterSpellcastingRules{}).EndMagicActionTurn(going.state, going.uses, true); len(kept.Events()) != 0 {
		t.Errorf("a turn with the Magic action produced %v, want no event", grpCSpellKinds(kept.Events()))
	}

	// The Magic action is taken: the casting completes for the same spell and entry, spends the use and reports the cast.
	partCast := grpCMust(combat.NewMonsterPartCast(grpCMistyStepAt, 0, combat.PartSpellcasting{}))
	if got := going.refusal(going.complete(partCast, "geas")); got != combat.MonsterCastRefusalRefMismatch {
		t.Errorf("completing by a part cast = %v, want casting_ref_mismatch", got)
	}
	if _, err := going.complete(going.list(grpCFireball), "geas"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("completing from an entry of another spell error = %v, want ErrInvalidInput", err)
	}
	if _, err := going.complete(going.list(grpCGeas), "fireball"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("completing with another spell error = %v, want ErrInvalidInput", err)
	}
	done := going.performed(going.complete(going.list(grpCGeas), "geas", grpCID("hero")))
	if done.Transition.LimitedUse().AvailabilityOfSpell(geasRef) != combat.UseExhausted {
		t.Errorf("completing the casting left the use unspent")
	}
	if _, _, in := done.Transition.State().CastingInProgress(); in {
		t.Errorf("a casting is still in progress after completion")
	}
	if got := rig.refusal(rig.complete(rig.list(grpCGeas), "geas")); got != combat.MonsterCastRefusalNotInProgress {
		t.Errorf("completing with none in progress = %v, want casting_not_in_progress", got)
	}
}

// Requirement: doc/specs/monster-spellcasting.md FR-039, FR-021, FR-005, FR-004, FR-002, FR-001, FR-032, FR-035 - a Bonus Action part casts a spell by its own part, immediately, spending the part's use; a spell with the wrong name for its list entry, a monster with no Spellcasting trait, the zero caster or state, an entry level below the spell's and a part cast with no values of its own on a monster with no trait are errors, not refusals; the character side rejects a monster report that names a class or a slot.
// Path: unhappy (invalid input).
func TestGrpCMonsterSpellcastingInvalidInput(t *testing.T) {
	t.Parallel()
	rig := grpCNewRig(t)

	part := grpCMust(combat.NewMonsterPartCast(grpCMistyStepAt, 0, combat.PartSpellcasting{}))
	out := rig.performed(rig.cast(part, "misty_step"))
	if out.Transition.LimitedUse().AvailabilityOf(grpCMistyStepAt) != combat.UseAvailable { // 3/Day, one spent
		t.Errorf("Misty Step after one cast = %v, want uses left", out.Transition.LimitedUse().AvailabilityOf(grpCMistyStepAt))
	}
	spent := rig.uses
	for range 3 {
		spent = grpCSpent(t, spent, grpCMistyStepAt)
	}
	exhausted := rig
	exhausted.uses = spent
	if got := exhausted.refusal(exhausted.cast(part, "misty_step")); got != combat.MonsterCastRefusalUsesExhausted {
		t.Errorf("a fourth Misty Step = %v, want uses_exhausted", got)
	}

	if _, err := rig.cast(rig.list(grpCFireball), "hold_person"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a spell with the wrong name for its entry error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewMonsterSpellcaster(grpCMonster(nil), grpCID("plain")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a monster with no Spellcasting trait error = %v, want ErrInvalidInput", err)
	}
	zero := rig
	zero.caster = combat.MonsterSpellcaster{}
	if _, err := zero.cast(rig.list(grpCFireBolt), "fire_bolt"); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero caster error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewMonsterSpellcastingState(combat.CombatantID("")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a state of no combatant error = %v, want ErrInvalidInput", err)
	}

	// Character side: a monster cast carries no class and no slot, and the zero MonsterCastReport has no character report.
	for name, p := range map[string]character.CastReportParts{
		"class":      {Spell: "fireball", Source: character.CastSourceMonster, Class: character.CasterWizard},
		"slot spent": {Spell: "fireball", Source: character.CastSourceMonster, HasSlotSpent: true},
	} {
		if _, err := character.NewCastReport(p); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: NewCastReport error = %v, want ErrInvalidInput", name, err)
		}
	}
	if _, err := (combat.MonsterCastReport{}).ToCastReport(); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero report error = %v, want ErrInvalidInput", err)
	}
	if _, err := character.NewCastingInProgress("fireball", character.CastSourceMonster, 0, content.CastingTimeMode{}, nil, 1); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a casting in progress with the monster source error = %v, want ErrInvalidInput", err)
	}
}
