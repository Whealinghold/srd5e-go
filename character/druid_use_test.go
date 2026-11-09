package character_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// druCase is a Druid at a level with a state and a spell slot state, about to use a feature.
type druCase struct {
	level   int
	circle  bool
	f       character.DruidStateFields
	slots   map[int]int // expended slots per slot level
	use     character.DruidFeatureUse
	slotsAt *character.SpellcastingState
}

func (c druCase) input(t *testing.T) character.UseDruidFeatureInput {
	t.Helper()
	spell := druSlots(t, max(c.level, 1), c.slots)
	if c.slotsAt != nil {
		spell = *c.slotsAt
	}
	return character.UseDruidFeatureInput{State: druState(t, c.level, c.circle, c.f), Tables: druTables(t), Facts: druFacts(t, c.level, c.circle), Spellcasting: spell, Use: c.use}
}

// run applies the use and returns the used result, or the rejection reason.
func (c druCase) run(t *testing.T) (character.UseDruidFeatureUsed, character.DruidRejection) {
	t.Helper()
	r, err := character.UseDruidFeature(c.input(t))
	if err != nil {
		t.Fatal(err)
	}
	switch got := r.(type) {
	case character.UseDruidFeatureUsed:
		return got, 0
	case character.UseDruidFeatureRejected:
		return character.UseDruidFeatureUsed{}, got.Reason
	}
	t.Fatalf("result %#v", r)
	return character.UseDruidFeatureUsed{}, 0
}

var wolfKnown = druIDs("wolf", "rat", "spider", "riding_horse")

// Requirement: doc/specs/class-druid.md FR-057 to FR-065, FR-070, FR-071, FR-084, FR-085, FR-139, FR-064 and tables C, O (SRD 5.2.1, "Level 2: Wild Shape", "Rules While Shape-Shifted", pp. 42-43) -
// assuming a known form spends exactly one use, sets the active form and emits WildShapeAssumed with the form, a duration of half the Druid level in hours (exact), Temporary Hit Points equal to the Druid
// level (a multiclass Druid 3 / Fighter 5 gets 1 hour 30 minutes and 3), a Bonus Action cost and the worn and merged item keys as two lists; from a form it is a switch that emits WildShapeEnded
// (usedWildShapeAgain) then WildShapeAssumed, also to the same form; the rejection is the first of the order and leaves the state unchanged.
// Path: per case (happy / unhappy: boundary at the levels of table C, state error, invalid input, rule interaction).
func TestUseDruidFeatureAssumeWildShape(t *testing.T) {
	t.Parallel()
	items := []character.DruidItemPlan{{Key: "cloak", Disposition: character.EquipmentDispositionWorn, FormCanWear: true}, {Key: "staff", Disposition: character.EquipmentDispositionMergesIntoForm},
		{Key: "coin", Disposition: character.EquipmentDispositionFallsInSpace}, {Key: "ring", Disposition: character.EquipmentDispositionWorn, FormCanWear: true}}
	assume := func(form string, items ...character.DruidItemPlan) character.DruidUseAssumeWildShape {
		return character.DruidUseAssumeWildShape{Form: content.MonsterID(form), Items: items}
	}
	for _, c := range []struct {
		level            int
		duration         time.Duration
		hitPointsAndUses int
	}{{2, time.Hour, 2}, {3, 90 * time.Minute, 3}, {10, 5 * time.Hour, 10}, {19, 9*time.Hour + 30*time.Minute, 19}, {20, 10 * time.Hour, 20}} {
		u, reason := druCase{level: c.level, f: character.DruidStateFields{WildShapeUsesRemaining: min(2, c.level), KnownForms: wolfKnown[:2]}, use: assume("wolf", items...)}.run(t)
		if reason != 0 {
			t.Fatalf("level %d: %v", c.level, reason)
		}
		ev, ok := u.Events()[0].(character.WildShapeAssumed)
		form, inForm := u.State().ActiveForm()
		if len(u.Events()) != 1 || !ok || ev.Form() != "wolf" || ev.Duration() != c.duration || ev.TemporaryHitPoints() != c.hitPointsAndUses || ev.ActionCost() != character.DruidActionCostBonus ||
			!slices.Equal(ev.WornItems(), []character.DruidItemKey{"cloak", "ring"}) || !slices.Equal(ev.MergedItems(), []character.DruidItemKey{"staff"}) ||
			u.State().WildShapeUsesRemaining() != min(2, c.level)-1 || form != "wolf" || !inForm || len(u.SlotChanges()) != 0 || ev.CharacterID() != druTestID {
			t.Errorf("level %d: events %#v state %+v", c.level, u.Events(), u.State())
		}
	}
	ev := mustUsed(t, druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown}, use: assume("rat", items...)}).Events()[0].(character.WildShapeAssumed)
	ev.WornItems()[0] = "changed"
	ev.MergedItems()[0] = "changed"
	if ev.WornItems()[0] != "cloak" || ev.MergedItems()[0] != "staff" {
		t.Error("WildShapeAssumed shares storage")
	}
	// table O: switch
	sw, _ := druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown, ActiveForm: "wolf"}, use: assume("riding_horse")}.run(t)
	ended, ok1 := sw.Events()[0].(character.WildShapeEnded)
	assumed, ok2 := sw.Events()[1].(character.WildShapeAssumed)
	if len(sw.Events()) != 2 || !ok1 || !ok2 || ended.Reason() != character.WildShapeEndUsedWildShapeAgain || ended.Form() != "wolf" || ended.ActionCost() != character.DruidActionCostNone ||
		assumed.TemporaryHitPoints() != 5 || sw.State().WildShapeUsesRemaining() != 1 {
		t.Errorf("switch: %#v", sw.Events())
	}
	same, _ := druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown, ActiveForm: "wolf"}, use: assume("wolf")}.run(t)
	if len(same.Events()) != 2 || same.State().WildShapeUsesRemaining() != 1 {
		t.Errorf("switch to the same form: %#v", same.Events())
	}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"level 1", druCase{level: 1, use: assume("wolf")}, character.DruidFeatureNotGranted},
		{"form not known", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown}, use: assume("bear")}, character.DruidFormNotKnown},
		{"no uses", druCase{level: 5, f: character.DruidStateFields{KnownForms: wolfKnown}, use: assume("wolf")}, character.DruidNoUsesRemaining},
		{"switch with none left", druCase{level: 5, f: character.DruidStateFields{KnownForms: wolfKnown, ActiveForm: "wolf"}, use: assume("rat")}, character.DruidNoUsesRemaining},
		{"switch to an unknown form", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown, ActiveForm: "wolf"}, use: assume("bear")}, character.DruidFormNotKnown},
		{"item unassigned", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown}, use: assume("wolf", character.DruidItemPlan{Key: "cloak"})}, character.DruidItemUnassigned},
		{"cannot be worn", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown}, use: assume("wolf", character.DruidItemPlan{Key: "plate", Disposition: character.EquipmentDispositionWorn})}, character.DruidItemCannotBeWorn},
		{"unknown form and unassigned item and no uses", druCase{level: 5, f: character.DruidStateFields{KnownForms: wolfKnown}, use: assume("bear", character.DruidItemPlan{Key: "cloak"})}, character.DruidFormNotKnown},
		{"unassigned item and cannot be worn", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown},
			use: assume("wolf", character.DruidItemPlan{Key: "a"}, character.DruidItemPlan{Key: "b", Disposition: character.EquipmentDispositionWorn})}, character.DruidItemUnassigned},
		{"item problem and no uses", druCase{level: 5, f: character.DruidStateFields{KnownForms: wolfKnown}, use: assume("wolf", character.DruidItemPlan{Key: "a"})}, character.DruidItemUnassigned},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	// a rejection leaves the state as it was
	in := druCase{level: 5, f: character.DruidStateFields{KnownForms: wolfKnown, ActiveForm: "wolf"}, use: assume("rat")}.input(t)
	before := in.State
	if r, err := character.UseDruidFeature(in); err != nil || in.State != before {
		t.Errorf("rejected use: %#v, %v", r, err)
	}
	if _, err := character.UseDruidFeature(druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown},
		use: assume("wolf", character.DruidItemPlan{Key: "a", Disposition: 9})}.input(t)); err == nil {
		t.Error("a disposition outside the set was accepted")
	} else {
		wantInvalid(t, err, "disposition")
	}
}

func mustUsed(t *testing.T, c druCase) character.UseDruidFeatureUsed {
	t.Helper()
	u, reason := c.run(t)
	if reason != 0 {
		t.Fatalf("use rejected with %v", reason)
	}
	return u
}

// Requirement: doc/specs/class-druid.md FR-064, FR-066, FR-067 (SRD 5.2.1, "Level 2: Wild Shape", p. 42) - ending Wild Shape clears the active form and emits WildShapeEnded with the reason (the Bonus
// Action cost only for leaving early), refunds no use, and is NotInBeastForm out of a form and FeatureNotGranted below level 2; the end reason used-again belongs to a switch and is refused.
// Path: per case (happy / unhappy: state error, boundary, invalid input).
func TestUseDruidFeatureEndWildShape(t *testing.T) {
	t.Parallel()
	in := character.DruidStateFields{WildShapeUsesRemaining: 1, KnownForms: wolfKnown, ActiveForm: "wolf"}
	for _, r := range []character.WildShapeEndReason{character.WildShapeEndDurationElapsed, character.WildShapeEndIncapacitated, character.WildShapeEndDied, character.WildShapeEndLeftEarly} {
		u := mustUsed(t, druCase{level: 5, f: in, use: character.DruidUseEndWildShape{Reason: r}})
		ev, ok := u.Events()[0].(character.WildShapeEnded)
		_, inForm := u.State().ActiveForm()
		wantCost := character.DruidActionCostNone
		if r == character.WildShapeEndLeftEarly {
			wantCost = character.DruidActionCostBonus
		}
		if !ok || ev.Reason() != r || ev.Form() != "wolf" || ev.ActionCost() != wantCost || inForm || u.State().WildShapeUsesRemaining() != 1 {
			t.Errorf("%v: events %#v state %+v", r, u.Events(), u.State())
		}
	}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"not in a form", druCase{level: 5, f: character.DruidStateFields{KnownForms: wolfKnown}, use: character.DruidUseEndWildShape{Reason: character.WildShapeEndLeftEarly}}, character.DruidNotInBeastForm},
		{"level 1", druCase{level: 1, use: character.DruidUseEndWildShape{Reason: character.WildShapeEndLeftEarly}}, character.DruidFeatureNotGranted},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	for _, r := range []character.WildShapeEndReason{0, character.WildShapeEndUsedWildShapeAgain, 9} {
		if _, err := character.UseDruidFeature(druCase{level: 5, f: in, use: character.DruidUseEndWildShape{Reason: r}}.input(t)); err == nil {
			t.Errorf("end reason %v was accepted", r)
		} else {
			wantInvalid(t, err, "end reason")
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-086 to FR-091, FR-140, FR-155, FR-160 and table P (SRD 5.2.1, "Level 2: Wild Companion", p. 43) - Wild Companion pays one spell slot (a DruidSlotChangeSpend
// for the level, the Wild Shape state untouched) or one Wild Shape use, as a Magic action, marks the familiar, and does not need Find Familiar prepared; it is FeatureNotGranted at level 1,
// CannotCastInBeastForm in a form below level 18 whatever the resources, and from level 18 only the resources matter; no slot is NoSlotAvailable, no use NoUsesRemaining; the slot state is never changed.
// Path: per case (happy / unhappy: boundary at 17 and 18, state error, rule interaction).
func TestUseDruidFeatureWildCompanion(t *testing.T) {
	t.Parallel()
	slot := character.DruidUseWildCompanion{Payment: character.DruidPaymentSlot{Pool: character.PoolSpellcasting, Level: 1}}
	byUse := character.DruidUseWildCompanion{Payment: character.DruidPaymentWildShape{}}
	allGone := map[int]int{1: 4}
	inForm := character.DruidStateFields{WildShapeUsesRemaining: 1, KnownForms: wolfKnown, ActiveForm: "wolf"}
	u := mustUsed(t, druCase{level: 2, f: character.DruidStateFields{WildShapeUsesRemaining: 2}, use: slot})
	ev, ok := u.Events()[0].(character.WildCompanionCast)
	wantChange := []character.DruidSlotChange{character.DruidSlotChangeSpend{Pool: character.PoolSpellcasting, Level: 1}}
	if !ok || ev.ActionCost() != character.DruidActionCostMagic || ev.Payment() != slot.Payment || !slices.Equal(u.SlotChanges(), wantChange) || u.State().WildShapeUsesRemaining() != 2 || !u.State().FamiliarPresent() {
		t.Errorf("slot payment: %#v, %+v", u.Events(), u.State())
	}
	u = mustUsed(t, druCase{level: 2, f: character.DruidStateFields{WildShapeUsesRemaining: 2}, use: byUse})
	if len(u.SlotChanges()) != 0 || u.State().WildShapeUsesRemaining() != 1 || !u.State().FamiliarPresent() || u.Events()[0].(character.WildCompanionCast).Payment() != byUse.Payment {
		t.Errorf("use payment: %#v, %+v", u.Events(), u.State())
	}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"not granted", druCase{level: 1, use: slot}, character.DruidFeatureNotGranted},
		{"form below 18 and no slot", druCase{level: 10, f: inForm, slots: map[int]int{1: 4}, use: slot}, character.DruidCannotCastInBeastForm},
		{"form below 18 and no uses", druCase{level: 10, f: character.DruidStateFields{KnownForms: wolfKnown, ActiveForm: "wolf"}, use: byUse}, character.DruidCannotCastInBeastForm},
		{"form at 17", druCase{level: 17, f: inForm, use: slot}, character.DruidCannotCastInBeastForm},
		{"form at 18, no slot", druCase{level: 18, f: inForm, slots: map[int]int{1: 4}, use: slot}, character.DruidNoSlotAvailable},
		{"no form, no uses", druCase{level: 10, use: byUse}, character.DruidNoUsesRemaining},
		{"no slot", druCase{level: 5, slots: allGone, use: slot}, character.DruidNoSlotAvailable},
		{"not granted and no slot", druCase{level: 1, slots: allGone, use: slot}, character.DruidFeatureNotGranted},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	if _, reason := (druCase{level: 18, f: inForm, use: byUse}).run(t); reason != 0 {
		t.Errorf("form at 18, paid by use: %v, want accepted", reason)
	}
	// the slot state is read only
	spells := druSlots(t, 5, nil)
	before := spells.Slots(character.PoolSpellcasting)
	if _, reason := (druCase{level: 5, slotsAt: &spells, use: slot}).run(t); reason != 0 || spells.Slots(character.PoolSpellcasting) != before {
		t.Errorf("the slot state changed or the use was rejected: %v", reason)
	}
	// the Pact Magic pool is a pool too, a slot of level 1 to 9 only
	for _, p := range []character.DruidPayment{character.DruidPaymentSlot{Pool: 0, Level: 1}, character.DruidPaymentSlot{Pool: character.PoolSpellcasting, Level: 0},
		character.DruidPaymentSlot{Pool: character.PoolSpellcasting, Level: 10}, nil} {
		if _, err := character.UseDruidFeature(druCase{level: 5, use: character.DruidUseWildCompanion{Payment: p}}.input(t)); err == nil {
			t.Errorf("payment %#v was accepted", p)
		} else {
			wantInvalid(t, err, "payment")
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-096 to FR-103, FR-088, FR-160 (SRD 5.2.1, "Level 5: Wild Resurgence", p. 43) - the slot-for-use exchange needs level 5, no use left (WildShapeUsesRemain), a slot of the level
// (NoSlotAvailable) and no earlier exchange this turn (AlreadyUsedThisTurn); it adds the use, spends the slot (a DruidSlotChangeSpend) and emits SpellSlotsExchanged; the use-for-slot exchange needs a use
// (NoUsesRemaining), an expended level 1 slot (NoExpendedSlot) and a Long Rest since the last (AlreadyUsedSinceLongRest), spends the use and returns a DruidSlotChangeRegain of level 1; the two limits are independent.
// Path: per case (happy / unhappy: boundary, state error, rule interaction).
func TestUseDruidFeatureWildResurgence(t *testing.T) {
	t.Parallel()
	forSlot := character.DruidUseSlotForWildShapeUse{Pool: character.PoolSpellcasting, Level: 2}
	forUse := character.DruidUseWildShapeUseForSlot{Pool: character.PoolSpellcasting}
	u := mustUsed(t, druCase{level: 5, use: forSlot})
	ev, ok := u.Events()[0].(character.SpellSlotsExchanged)
	if !ok || ev.Kind() != character.SlotExchangeSlotForUse || ev.SlotLevel() != 2 || ev.Uses() != 1 || u.State().WildShapeUsesRemaining() != 1 || !u.State().WildResurgenceSlotForUseThisTurn() ||
		!slices.Equal(u.SlotChanges(), []character.DruidSlotChange{character.DruidSlotChangeSpend{Pool: character.PoolSpellcasting, Level: 2}}) {
		t.Errorf("slot for use: %#v %+v", u.Events(), u.State())
	}
	u = mustUsed(t, druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2}, slots: map[int]int{1: 1}, use: forUse})
	ev, ok = u.Events()[0].(character.SpellSlotsExchanged)
	if !ok || ev.Kind() != character.SlotExchangeUseForSlot || ev.SlotLevel() != 1 || u.State().WildShapeUsesRemaining() != 1 || !u.State().WildResurgenceUseForSlot() ||
		!slices.Equal(u.SlotChanges(), []character.DruidSlotChange{character.DruidSlotChangeRegain{Pool: character.PoolSpellcasting, Level: 1}}) {
		t.Errorf("use for slot: %#v %+v", u.Events(), u.State())
	}
	// FR-103: neither limit blocks the other
	both := druCase{level: 5, f: character.DruidStateFields{WildResurgenceUseForSlot: true}, use: forSlot}
	if _, reason := both.run(t); reason != 0 {
		t.Errorf("slot for use after the use-for-slot limit: %v", reason)
	}
	both = druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 1, WildResurgenceSlotForUseThisTurn: true}, slots: map[int]int{1: 1}, use: forUse}
	if _, reason := both.run(t); reason != 0 {
		t.Errorf("use for slot after the once-per-turn limit: %v", reason)
	}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"slot for use at level 4", druCase{level: 4, use: forSlot}, character.DruidFeatureNotGranted},
		{"use for slot at level 4", druCase{level: 4, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, slots: map[int]int{1: 1}, use: forUse}, character.DruidFeatureNotGranted},
		{"uses remain", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, use: forSlot}, character.DruidWildShapeUsesRemain},
		{"no slot", druCase{level: 5, slots: map[int]int{2: 3}, use: forSlot}, character.DruidNoSlotAvailable},
		{"second this turn", druCase{level: 5, f: character.DruidStateFields{WildResurgenceSlotForUseThisTurn: true}, use: forSlot}, character.DruidAlreadyUsedThisTurn},
		{"second this turn and uses remain", druCase{level: 5, f: character.DruidStateFields{WildResurgenceSlotForUseThisTurn: true, WildShapeUsesRemaining: 1}, use: forSlot}, character.DruidAlreadyUsedThisTurn},
		{"uses remain and no slot", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, slots: map[int]int{2: 3}, use: forSlot}, character.DruidWildShapeUsesRemain},
		{"no use to spend", druCase{level: 5, slots: map[int]int{1: 1}, use: forUse}, character.DruidNoUsesRemaining},
		{"no expended slot", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, use: forUse}, character.DruidNoExpendedSlot},
		{"used since the Long Rest", druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 1, WildResurgenceUseForSlot: true}, slots: map[int]int{1: 1}, use: forUse}, character.DruidAlreadyUsedSinceLongRest},
		{"used and no use left", druCase{level: 5, f: character.DruidStateFields{WildResurgenceUseForSlot: true}, slots: map[int]int{1: 1}, use: forUse}, character.DruidAlreadyUsedSinceLongRest},
		{"no use and no expended slot", druCase{level: 5, use: forUse}, character.DruidNoUsesRemaining},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	for _, bad := range []character.DruidFeatureUse{character.DruidUseSlotForWildShapeUse{Pool: character.PoolSpellcasting, Level: 0}, character.DruidUseSlotForWildShapeUse{Level: 1}, character.DruidUseWildShapeUseForSlot{}} {
		if _, err := character.UseDruidFeature(druCase{level: 5, use: bad}.input(t)); err == nil {
			t.Errorf("%#v was accepted", bad)
		} else {
			wantInvalid(t, err, "slot")
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-122 to FR-124, FR-157, FR-123, FR-152 and tables C, L (SRD 5.2.1, "Level 3: Land's Aid", p. 46) - Land's Aid needs the Circle of the Land from level 3, spends one Wild Shape
// use (NoUsesRemaining, state unchanged) as a Magic action and emits LandsAidUsed with the caller's spell save DC, a Sphere of radius 10 feet within 60 feet and the damage and healing dice 2d6, 3d6 from 10, 4d6 from 14.
// Path: per case (happy / unhappy: boundary at 2/3, 9/10, 13/14, state error, invalid input).
func TestUseDruidFeatureLandsAid(t *testing.T) {
	t.Parallel()
	dc := dcOf(t, 14)
	for _, c := range []struct{ level, dice int }{{3, 2}, {9, 2}, {10, 3}, {13, 3}, {14, 4}, {20, 4}} {
		u := mustUsed(t, druCase{level: c.level, circle: true, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, use: character.DruidUseLandsAid{DC: dc}})
		ev, ok := u.Events()[0].(character.LandsAidUsed)
		d := dice.MustExpression(c.dice, dice.D6(), 0)
		if !ok || ev.SaveDC() != dc || ev.Damage() != d || ev.Healing() != d || ev.RangeFeet() != 60 || ev.RadiusFeet() != 10 || ev.ActionCost() != character.DruidActionCostMagic ||
			u.State().WildShapeUsesRemaining() != 0 || len(u.SlotChanges()) != 0 {
			t.Errorf("level %d: %#v %+v", c.level, u.Events(), u.State())
		}
	}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"level 2 with the subclass", druCase{level: 2, circle: true, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, use: character.DruidUseLandsAid{DC: dc}}, character.DruidFeatureNotGranted},
		{"level 3 without the subclass", druCase{level: 3, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, use: character.DruidUseLandsAid{DC: dc}}, character.DruidFeatureNotGranted},
		{"no uses", druCase{level: 3, circle: true, use: character.DruidUseLandsAid{DC: dc}}, character.DruidNoUsesRemaining},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	if _, err := character.UseDruidFeature(druCase{level: 3, circle: true, f: character.DruidStateFields{WildShapeUsesRemaining: 1}, use: character.DruidUseLandsAid{}}.input(t)); err == nil {
		t.Error("a zero spell save DC was accepted")
	} else {
		wantInvalid(t, err, "DC")
	}
}

// Requirement: doc/specs/class-druid.md FR-125 to FR-130, FR-138, FR-157 and tables K, L, P (SRD 5.2.1, "Level 6: Natural Recovery", p. 46) - the casting takes a level 1+ Circle Spell of the current land without a slot, once
// until a Long Rest, and consults the Beast form rule with the spell; NotACircleSpell for a spell of another land, another source or a cantrip, LandTypeNotChosen without a land, AlreadyUsedSinceLongRest after use,
// CannotCastInBeastForm in a form below 18 and MaterialComponentRestricted from 18 for a costed spell; the first reason of the order wins.
// Path: per case (happy / unhappy: boundary, state error, rule interaction).
func TestUseDruidFeatureNaturalRecoveryCast(t *testing.T) {
	t.Parallel()
	polar := func(extra func(*character.DruidStateFields)) character.DruidStateFields {
		f := character.DruidStateFields{LandType: character.LandPolar}
		if extra != nil {
			extra(&f)
		}
		return f
	}
	cast := func(id string, level int, material string) character.DruidUseNaturalRecoveryCast {
		return character.DruidUseNaturalRecoveryCast{Spell: druSpell(t, id, level, []string{"druid", "wizard"}, "", material)}
	}
	u := mustUsed(t, druCase{level: 6, circle: true, f: polar(nil), use: cast("hold_person", 2, "")})
	ev, ok := u.Events()[0].(character.NaturalRecoveryUsed)
	if !ok || ev.Spell() != "hold_person" || len(ev.RecoveredSlots()) != 0 || !u.State().NaturalRecoveryCast() || len(u.SlotChanges()) != 0 {
		t.Errorf("cast: %#v %+v", u.Events(), u.State())
	}
	inForm := func(f *character.DruidStateFields) { f.KnownForms, f.ActiveForm = wolfKnown, "wolf" }
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"level 5", druCase{level: 5, circle: true, f: polar(nil), use: cast("fog_cloud", 1, "")}, character.DruidFeatureNotGranted},
		{"without the subclass", druCase{level: 6, use: cast("fog_cloud", 1, "")}, character.DruidFeatureNotGranted},
		{"no land", druCase{level: 6, circle: true, use: cast("fog_cloud", 1, "")}, character.DruidLandTypeNotChosen},
		{"spell of another land", druCase{level: 6, circle: true, f: polar(nil), use: cast("blur", 2, "")}, character.DruidNotACircleSpell},
		{"a cantrip of the land", druCase{level: 6, circle: true, f: polar(nil), use: cast("ray_of_frost", 0, "")}, character.DruidNotACircleSpell},
		{"a level 5 row at level 6", druCase{level: 6, circle: true, f: polar(nil), use: cast("cone_of_cold", 5, "")}, character.DruidNotACircleSpell},
		{"used", druCase{level: 6, circle: true, f: polar(func(f *character.DruidStateFields) { f.NaturalRecoveryCast = true }), use: cast("fog_cloud", 1, "")}, character.DruidAlreadyUsedSinceLongRest},
		{"in a form at 17", druCase{level: 17, circle: true, f: polar(inForm), use: cast("fog_cloud", 1, "")}, character.DruidCannotCastInBeastForm},
		{"in a form at 18, costed", druCase{level: 18, circle: true, f: polar(inForm), use: cast("fog_cloud", 1, materialOnly("[100]", false))}, character.DruidMaterialComponentRestricted},
		{"in a form at 18, plain", druCase{level: 18, circle: true, f: polar(inForm), use: cast("fog_cloud", 1, "")}, 0},
		{"in a form, used and not a Circle Spell", druCase{level: 10, circle: true, f: polar(func(f *character.DruidStateFields) { inForm(f); f.NaturalRecoveryCast = true }), use: cast("blur", 2, "")}, character.DruidCannotCastInBeastForm},
		{"used and not a Circle Spell", druCase{level: 10, circle: true, f: polar(func(f *character.DruidStateFields) { f.NaturalRecoveryCast = true }), use: cast("blur", 2, "")}, character.DruidAlreadyUsedSinceLongRest},
		{"no land and used", druCase{level: 6, circle: true, f: character.DruidStateFields{NaturalRecoveryCast: true}, use: cast("fog_cloud", 1, "")}, character.DruidLandTypeNotChosen},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	if _, err := character.UseDruidFeature(druCase{level: 6, circle: true, f: polar(nil), use: character.DruidUseNaturalRecoveryCast{}}.input(t)); err == nil {
		t.Error("a zero spell was accepted")
	} else {
		wantInvalid(t, err, "spell")
	}
}

// Requirement: doc/specs/class-druid.md FR-127 to FR-130, FR-154, FR-158, FR-160 and tables K, L (SRD 5.2.1, "Level 6: Natural Recovery", p. 46) - the recovery needs the window a Short Rest opens
// (RecoveryNotAllowedNow), once until a Long Rest (AlreadyUsedSinceLongRest), recovers slots of combined level at most half the Druid level rounded up (RecoveryBudgetExceeded) and none of level 6 or higher
// (SlotLevelTooHigh), only expended slots (NoExpendedSlot), returns one DruidSlotChangeRegain per slot and closes the window; no slot is accepted and changes nothing; the casting and the recovery are independent.
// Path: per case (happy / unhappy: boundary, state error, rule interaction).
func TestUseDruidFeatureNaturalRecoveryRecover(t *testing.T) {
	t.Parallel()
	open := []character.RestWindowPermission{character.RestWindowNaturalRecoveryRecovery}
	recover := func(levels ...int) character.DruidUseNaturalRecoveryRecover {
		return character.DruidUseNaturalRecoveryRecover{Pool: character.PoolSpellcasting, Levels: levels}
	}
	base := character.DruidStateFields{RestWindow: open}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"SRD example: a level 3 slot", druCase{level: 6, circle: true, f: base, slots: map[int]int{3: 1}, use: recover(3)}, 0},
		{"SRD example: a level 2 and a level 1", druCase{level: 6, circle: true, f: base, slots: map[int]int{1: 1, 2: 1}, use: recover(2, 1)}, 0},
		{"SRD example: three level 1", druCase{level: 6, circle: true, f: base, slots: map[int]int{1: 3}, use: recover(1, 1, 1)}, 0},
		{"one over the budget", druCase{level: 6, circle: true, f: base, slots: map[int]int{2: 2}, use: recover(2, 2)}, character.DruidRecoveryBudgetExceeded},
		{"odd level rounds up", druCase{level: 7, circle: true, f: base, slots: map[int]int{1: 1, 3: 1}, use: recover(3, 1)}, 0},
		{"odd level, one over", druCase{level: 7, circle: true, f: base, slots: map[int]int{2: 1, 3: 1}, use: recover(3, 2)}, character.DruidRecoveryBudgetExceeded},
		{"a level 5 slot at 10", druCase{level: 10, circle: true, f: base, slots: map[int]int{5: 1}, use: recover(5)}, 0},
		{"a level 6 slot at 20", druCase{level: 20, circle: true, f: base, slots: map[int]int{6: 1}, use: recover(6)}, character.DruidSlotLevelTooHigh},
		{"level 9 slot at 20", druCase{level: 20, circle: true, f: base, slots: map[int]int{9: 1}, use: recover(9)}, character.DruidSlotLevelTooHigh},
		{"not expended", druCase{level: 6, circle: true, f: base, slots: map[int]int{1: 1}, use: recover(1, 1)}, character.DruidNoExpendedSlot},
		{"nothing expended", druCase{level: 6, circle: true, f: base, use: recover(1)}, character.DruidNoExpendedSlot},
		{"no slots chosen", druCase{level: 6, circle: true, f: base, use: recover()}, 0},
		{"over budget and level 6", druCase{level: 20, circle: true, f: base, slots: map[int]int{5: 2, 6: 1}, use: recover(6, 5, 5)}, character.DruidSlotLevelTooHigh},
		{"over budget and not expended", druCase{level: 6, circle: true, f: base, slots: map[int]int{2: 1}, use: recover(2, 2)}, character.DruidRecoveryBudgetExceeded},
		{"closed and not expended", druCase{level: 6, circle: true, use: recover(1)}, character.DruidRecoveryNotAllowedNow},
		{"used", druCase{level: 6, circle: true, f: character.DruidStateFields{RestWindow: open, NaturalRecoveryRecovery: true}, slots: map[int]int{1: 1}, use: recover(1)}, character.DruidAlreadyUsedSinceLongRest},
		{"closed and used", druCase{level: 6, circle: true, f: character.DruidStateFields{NaturalRecoveryRecovery: true}, slots: map[int]int{1: 1}, use: recover(1)}, character.DruidRecoveryNotAllowedNow},
		{"level 5", druCase{level: 5, circle: true, use: recover(1)}, character.DruidFeatureNotGranted},
		{"no subclass", druCase{level: 6, use: recover(1)}, character.DruidFeatureNotGranted},
		{"independent of the casting", druCase{level: 6, circle: true, f: character.DruidStateFields{RestWindow: open, NaturalRecoveryCast: true, LandType: character.LandArid}, slots: map[int]int{1: 1}, use: recover(1)}, 0},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	u := mustUsed(t, druCase{level: 6, circle: true, f: base, slots: map[int]int{1: 1, 2: 1}, use: recover(2, 1)})
	ev, ok := u.Events()[0].(character.NaturalRecoveryUsed)
	if !ok || !slices.Equal(ev.RecoveredSlots(), []int{2, 1}) || ev.Spell() != "" || !u.State().NaturalRecoveryRecovery() || len(u.State().RestWindow()) != 0 ||
		!slices.Equal(u.SlotChanges(), []character.DruidSlotChange{character.DruidSlotChangeRegain{Pool: character.PoolSpellcasting, Level: 2}, character.DruidSlotChangeRegain{Pool: character.PoolSpellcasting, Level: 1}}) {
		t.Errorf("recovery: %#v %+v", u.Events(), u.State())
	}
	none := mustUsed(t, druCase{level: 6, circle: true, f: base, use: recover()})
	if none.State() != druState(t, 6, true, base) || len(none.Events()) != 0 || len(none.SlotChanges()) != 0 {
		t.Errorf("recovering no slot changed something: %#v %+v", none.Events(), none.State())
	}
	ev.RecoveredSlots()[0] = 9
	if ev.RecoveredSlots()[0] != 2 {
		t.Error("NaturalRecoveryUsed shares storage")
	}
	for _, bad := range []character.DruidUseNaturalRecoveryRecover{recover(0), recover(10), recover(1, -1), {Levels: []int{1}}} {
		if _, err := character.UseDruidFeature(druCase{level: 6, circle: true, f: base, use: bad}.input(t)); err == nil {
			t.Errorf("%#v was accepted", bad)
		} else {
			wantInvalid(t, err, "slot level")
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-133 to FR-137, FR-157, FR-134, FR-131 and tables L, P (SRD 5.2.1, "Level 14: Nature's Sanctuary", p. 46) - the Cube is created for one Wild Shape use as a Magic action within
// 120 feet (MoveTooFar beyond), reporting the Resistance of the current land when one is chosen; a second is SanctuaryAlreadyActive, none left NoUsesRemaining, below 14 or without the subclass FeatureNotGranted; moving
// is a Bonus Action up to 60 feet to ground within 120 feet (61 and 121 are MoveTooFar; no Cube is NoActiveSanctuary); it ends with a reason and clears the Cube.
// Path: per case (happy / unhappy: boundary at 60/61 and 120/121, state error, rule interaction).
func TestUseDruidFeatureNaturesSanctuary(t *testing.T) {
	t.Parallel()
	create := func(feet int) character.DruidUseCreateSanctuary {
		return character.DruidUseCreateSanctuary{DistanceFeet: feet}
	}
	move := func(moveFeet, feet int) character.DruidUseMoveSanctuary {
		return character.DruidUseMoveSanctuary{MoveFeet: moveFeet, DistanceFeet: feet}
	}
	has := character.DruidStateFields{WildShapeUsesRemaining: 1}
	active := character.DruidStateFields{WildShapeUsesRemaining: 1, SanctuaryActive: true}
	for _, c := range []struct {
		land character.LandType
		want content.DamageType
	}{{character.LandArid, content.DamageFire}, {character.LandPolar, content.DamageCold}, {character.LandTemperate, content.DamageLightning}, {character.LandTropical, content.DamagePoison}, {0, 0}} {
		u := mustUsed(t, druCase{level: 14, circle: true, f: character.DruidStateFields{WildShapeUsesRemaining: 1, LandType: c.land}, use: create(120)})
		ev, ok := u.Events()[0].(character.NaturesSanctuaryCreated)
		r, hasResistance := ev.Resistance()
		if !ok || r != c.want || hasResistance != (c.want != 0) || ev.ActionCost() != character.DruidActionCostMagic || !u.State().SanctuaryActive() || u.State().WildShapeUsesRemaining() != 0 {
			t.Errorf("land %v: %#v %+v", c.land, u.Events(), u.State())
		}
	}
	if z := mustUsed(t, druCase{level: 14, circle: true, f: has, use: create(0)}); !z.State().SanctuaryActive() {
		t.Errorf("a Cube at distance 0 was not created: %+v", z.State())
	}
	mv := mustUsed(t, druCase{level: 14, circle: true, f: active, use: move(60, 120)})
	if ev, ok := mv.Events()[0].(character.NaturesSanctuaryMoved); !ok || ev.ActionCost() != character.DruidActionCostBonus || !mv.State().SanctuaryActive() {
		t.Errorf("move: %#v", mv.Events())
	}
	for _, r := range []character.SanctuaryEndReason{character.SanctuaryEndMinuteElapsed, character.SanctuaryEndIncapacitated, character.SanctuaryEndDied} {
		u := mustUsed(t, druCase{level: 14, circle: true, f: active, use: character.DruidUseEndSanctuary{Reason: r}})
		if ev, ok := u.Events()[0].(character.NaturesSanctuaryEnded); !ok || ev.Reason() != r || u.State().SanctuaryActive() || u.State().WildShapeUsesRemaining() != 1 {
			t.Errorf("end %v: %#v %+v", r, u.Events(), u.State())
		}
	}
	for _, c := range []struct {
		label string
		c     druCase
		want  character.DruidRejection
	}{
		{"create at 13", druCase{level: 13, circle: true, f: has, use: create(10)}, character.DruidFeatureNotGranted},
		{"not granted without the subclass", druCase{level: 14, f: has, use: create(10)}, character.DruidFeatureNotGranted},
		{"Cube active and no uses", druCase{level: 14, circle: true, f: character.DruidStateFields{SanctuaryActive: true}, use: create(10)}, character.DruidSanctuaryAlreadyActive},
		{"no uses, no Cube", druCase{level: 14, circle: true, use: create(10)}, character.DruidNoUsesRemaining},
		{"121 feet", druCase{level: 14, circle: true, f: has, use: create(121)}, character.DruidMoveTooFar},
		{"121 feet and no uses", druCase{level: 14, circle: true, use: create(121)}, character.DruidMoveTooFar},
		{"move without a Cube, too far", druCase{level: 14, circle: true, use: move(61, 10)}, character.DruidNoActiveSanctuary},
		{"move 61 feet", druCase{level: 14, circle: true, f: active, use: move(61, 10)}, character.DruidMoveTooFar},
		{"move to 121 feet", druCase{level: 14, circle: true, f: active, use: move(10, 121)}, character.DruidMoveTooFar},
		{"move at 13", druCase{level: 13, circle: true, use: move(10, 10)}, character.DruidFeatureNotGranted},
		{"end without a Cube", druCase{level: 14, circle: true, use: character.DruidUseEndSanctuary{Reason: character.SanctuaryEndDied}}, character.DruidNoActiveSanctuary},
		{"end without the subclass", druCase{level: 14, use: character.DruidUseEndSanctuary{Reason: character.SanctuaryEndDied}}, character.DruidFeatureNotGranted},
	} {
		if _, reason := c.c.run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	if _, reason := (druCase{level: 14, circle: true, f: active, use: move(60, 120)}).run(t); reason != 0 {
		t.Errorf("a move of 60 feet to 120 feet: %v", reason)
	}
	for _, bad := range []character.DruidFeatureUse{create(-1), move(-1, 0), move(0, -1), character.DruidUseEndSanctuary{}, character.DruidUseEndSanctuary{Reason: 9}} {
		if _, err := character.UseDruidFeature(druCase{level: 14, circle: true, f: active, use: bad}.input(t)); err == nil {
			t.Errorf("%#v was accepted", bad)
		} else {
			wantInvalid(t, err, "sanctuary")
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-113, FR-114, FR-116, FR-160 and table I (SRD 5.2.1, "Level 20: Archdruid", p. 43) - Nature Magician converts n unexpended Wild Shape uses into one expended slot of level 2n
// (a DruidSlotChangeRegain), no action, once until a Long Rest; FeatureNotGranted below 20, InvalidConversion for 0, NoUsesRemaining for more than remain, NoExpendedSlot without an expended slot of the level,
// AlreadyUsedSinceLongRest after use, and the first reason of the order when several hold.
// Path: per case (happy / unhappy: boundary, state error, rule interaction).
func TestUseDruidFeatureNatureMagician(t *testing.T) {
	t.Parallel()
	nm := func(n int) character.DruidUseNatureMagician {
		return character.DruidUseNatureMagician{Pool: character.PoolSpellcasting, Uses: n}
	}
	uses := func(n int) character.DruidStateFields { return character.DruidStateFields{WildShapeUsesRemaining: n} }
	for _, c := range []struct {
		label  string
		c      druCase
		want   character.DruidRejection
		uses   int
		regain int
	}{
		{"one use", druCase{level: 20, f: uses(4), slots: map[int]int{2: 1}, use: nm(1)}, 0, 3, 2},
		{"two uses: the SRD example", druCase{level: 20, f: uses(4), slots: map[int]int{4: 1}, use: nm(2)}, 0, 2, 4},
		{"all uses", druCase{level: 20, f: uses(4), slots: map[int]int{8: 1}, use: nm(4)}, 0, 0, 8},
		{"none", druCase{level: 20, f: uses(4), use: nm(0)}, character.DruidInvalidConversion, 0, 0},
		{"negative", druCase{level: 20, f: uses(4), use: nm(-1)}, character.DruidInvalidConversion, 0, 0},
		{"more than remain", druCase{level: 20, f: uses(3), slots: map[int]int{8: 1}, use: nm(4)}, character.DruidNoUsesRemaining, 0, 0},
		{"no expended slot", druCase{level: 20, f: uses(4), use: nm(2)}, character.DruidNoExpendedSlot, 0, 0},
		{"used already", druCase{level: 20, f: character.DruidStateFields{WildShapeUsesRemaining: 4, NatureMagician: true}, slots: map[int]int{2: 1}, use: nm(1)}, character.DruidAlreadyUsedSinceLongRest, 0, 0},
		{"zero and used", druCase{level: 20, f: character.DruidStateFields{WildShapeUsesRemaining: 4, NatureMagician: true}, use: nm(0)}, character.DruidAlreadyUsedSinceLongRest, 0, 0},
		{"too many and no expended slot", druCase{level: 20, f: uses(3), use: nm(4)}, character.DruidNoUsesRemaining, 0, 0},
		{"zero and no expended slot", druCase{level: 20, f: uses(4), use: nm(0)}, character.DruidInvalidConversion, 0, 0},
		{"level 19", druCase{level: 19, f: uses(4), slots: map[int]int{2: 1}, use: nm(1)}, character.DruidFeatureNotGranted, 0, 0},
		{"five uses would be a level 10 slot", druCase{level: 20, f: uses(4), use: nm(5)}, character.DruidNoUsesRemaining, 0, 0},
	} {
		u, reason := c.c.run(t)
		if reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
			continue
		}
		if reason == 0 {
			ev, ok := u.Events()[0].(character.SpellSlotsExchanged)
			if u.State().WildShapeUsesRemaining() != c.uses || !u.State().NatureMagician() || !slices.Equal(u.SlotChanges(), []character.DruidSlotChange{character.DruidSlotChangeRegain{Pool: character.PoolSpellcasting, Level: c.regain}}) ||
				!ok || ev.Kind() != character.SlotExchangeNatureMagician || ev.SlotLevel() != c.regain || ev.Uses() != c.regain/2 {
				t.Errorf("%s: %#v %+v", c.label, u.Events(), u.State())
			}
		}
	}
	if _, err := character.UseDruidFeature(druCase{level: 20, f: uses(4), use: character.DruidUseNatureMagician{Uses: 1}}.input(t)); err == nil {
		t.Error("an undefined pool was accepted")
	} else {
		wantInvalid(t, err, "pool")
	}
	if _, err := character.UseDruidFeature(druCase{level: 20, f: uses(4), use: character.DruidUseNatureMagician{Uses: 0}}.input(t)); err == nil {
		t.Error("an undefined pool was accepted for zero uses")
	}
}

// Requirement: doc/specs/class-druid.md FR-104 to FR-109, FR-106, D-7 and table P (SRD 5.2.1, "Level 7: Elemental Fury", p. 43) - Primal Strike needs level 7 (FeatureNotGranted), the Primal Strike option
// (FeatureNotChosen, before the attack is looked at), once each turn (AlreadyUsedThisTurn, before the attack) and an eligible attack (AttackNotEligible); it emits PrimalStrikeUsed with 1d8, 2d8 from level 15,
// of one of four damage types, also for a weapon attack in Wild Shape.
// Path: per case (happy / unhappy: boundary, state error, rule interaction).
func TestUseDruidFeatureTurnsPrimalStrike(t *testing.T) {
	t.Parallel()
	hit, miss := rogAttack(t, true), rogAttack(t, false)
	strike := character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike}
	used := character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike, PrimalStrikeThisTurn: true}
	inForm := character.DruidStateFields{ElementalFury: character.ElementalFuryPrimalStrike, KnownForms: wolfKnown, ActiveForm: "wolf"}
	ps := func(s character.PrimalStrikeAttackSource) character.DruidUsePrimalStrike {
		return character.DruidUsePrimalStrike{Source: s, Attack: hit}
	}
	for _, c := range []struct {
		label string
		level int
		f     character.DruidStateFields
		use   character.DruidUsePrimalStrike
		want  character.DruidRejection
		dice  int
	}{
		{"level 7", 7, strike, ps(character.PrimalStrikeSourceWeapon), 0, 1},
		{"level 14", 14, strike, ps(character.PrimalStrikeSourceWeapon), 0, 1},
		{"level 15", 15, strike, ps(character.PrimalStrikeSourceWeapon), 0, 2},
		{"weapon attack in Wild Shape", 7, inForm, ps(character.PrimalStrikeSourceWeapon), 0, 1},
		{"Beast attack in Wild Shape", 7, inForm, ps(character.PrimalStrikeSourceBeastForm), 0, 1},
		{"level 6", 6, character.DruidStateFields{}, ps(character.PrimalStrikeSourceWeapon), character.DruidFeatureNotGranted, 0},
		{"Potent Spellcasting chosen", 7, character.DruidStateFields{ElementalFury: character.ElementalFuryPotentSpellcasting}, ps(character.PrimalStrikeSourceWeapon), character.DruidFeatureNotChosen, 0},
		{"Beast attack outside Wild Shape", 7, strike, ps(character.PrimalStrikeSourceBeastForm), character.DruidAttackNotEligible, 0},
		{"a spell attack", 7, strike, ps(character.PrimalStrikeSourceSpell), character.DruidAttackNotEligible, 0},
		{"second use and a spell", 7, used, ps(character.PrimalStrikeSourceSpell), character.DruidAlreadyUsedThisTurn, 0},
		{"second use", 7, used, ps(character.PrimalStrikeSourceWeapon), character.DruidAlreadyUsedThisTurn, 0},
	} {
		u, reason := druCase{level: c.level, f: c.f, use: c.use}.run(t)
		if reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
			continue
		}
		if reason == 0 {
			ev, ok := u.Events()[0].(character.PrimalStrikeUsed)
			if !ok || ev.Damage() != dice.MustExpression(c.dice, dice.D8(), 0) || !u.State().PrimalStrikeThisTurn() || len(ev.DamageTypes()) != 4 {
				t.Errorf("%s: %#v %+v", c.label, u.Events(), u.State())
			}
		}
	}
	for _, c := range []struct {
		label string
		f     character.DruidStateFields
		use   character.DruidUsePrimalStrike
		want  character.DruidRejection
	}{
		{"a miss", strike, character.DruidUsePrimalStrike{Source: character.PrimalStrikeSourceWeapon, Attack: miss}, character.DruidAttackNotEligible},
		{"other option chosen, miss", character.DruidStateFields{ElementalFury: character.ElementalFuryPotentSpellcasting}, character.DruidUsePrimalStrike{Source: character.PrimalStrikeSourceWeapon, Attack: miss}, character.DruidFeatureNotChosen},
		{"second use and a miss", used, character.DruidUsePrimalStrike{Source: character.PrimalStrikeSourceWeapon, Attack: miss}, character.DruidAlreadyUsedThisTurn},
	} {
		if _, reason := (druCase{level: 7, f: c.f, use: c.use}).run(t); reason != c.want {
			t.Errorf("%s: reason = %v, want %v", c.label, reason, c.want)
		}
	}
	if _, err := character.UseDruidFeature(druCase{level: 7, f: strike, use: character.DruidUsePrimalStrike{Attack: hit}}.input(t)); err == nil {
		t.Error("an undefined attack source was accepted")
	} else {
		wantInvalid(t, err, "source")
	}
}

// Requirement: doc/specs/class-druid.md FR-141, FR-147, FR-148, FR-164, FR-173 - UseDruidFeature returns an error matching ErrInvalidInput for zero facts or tables, a state that breaks an invariant and a nil use;
// a rejection is a result whose Slot method gives the slot rejection only for NoSlotAvailable and NoExpendedSlot; equal inputs give equal results and the input state is never modified.
// Path: per case (unhappy: invalid input, determinism, immutability).
func TestUseDruidFeatureInvalidInputAndResults(t *testing.T) {
	t.Parallel()
	ok := druCase{level: 5, f: character.DruidStateFields{WildShapeUsesRemaining: 2, KnownForms: wolfKnown}, use: character.DruidUseAssumeWildShape{Form: "wolf"}}.input(t)
	for _, c := range []struct {
		label  string
		mutate func(*character.UseDruidFeatureInput)
	}{
		{"zero facts", func(in *character.UseDruidFeatureInput) { in.Facts = character.DruidCharacterFacts{} }},
		{"zero tables", func(in *character.UseDruidFeatureInput) { in.Tables = character.DruidFeatureTables{} }},
		{"state breaks an invariant", func(in *character.UseDruidFeatureInput) {
			in.State = druState(t, 5, true, character.DruidStateFields{LandType: character.LandArid})
			in.Facts = druFacts(t, 5, false)
		}},
		{"nil use", func(in *character.UseDruidFeatureInput) { in.Use = nil }},
	} {
		in := ok
		c.mutate(&in)
		if _, err := character.UseDruidFeature(in); err == nil {
			t.Errorf("%s: no error", c.label)
		} else {
			wantInvalid(t, err, c.label)
		}
	}
	a, err1 := character.UseDruidFeature(ok)
	b, err2 := character.UseDruidFeature(ok)
	ua, _ := a.(character.UseDruidFeatureUsed)
	ub, _ := b.(character.UseDruidFeatureUsed)
	if err1 != nil || err2 != nil || ua.State() != ub.State() || len(ua.Events()) != len(ub.Events()) {
		t.Error("equal inputs gave different results")
	}
	ua.Events()[0] = nil
	ua.SlotChanges()
	if ua.Events()[0] == nil {
		t.Error("Events shares storage")
	}
	for _, c := range []struct {
		reason character.DruidRejection
		slot   character.SlotOperationRejection
		has    bool
	}{
		{character.DruidNoSlotAvailable, character.SlotOperationNoSlotAvailable, true}, {character.DruidNoExpendedSlot, character.SlotOperationNoExpendedSlot, true}, {character.DruidNoUsesRemaining, 0, false},
	} {
		if s, has := (character.UseDruidFeatureRejected{Reason: c.reason}).Slot(); s != c.slot || has != c.has {
			t.Errorf("Slot of %v = %v, %v", c.reason, s, has)
		}
	}
}

// Requirement: doc/specs/class-druid.md FR-165 - FeatureAttribute maps every use to one srd5e.feature value (wild_shape for assuming or ending, wild_companion, wild_resurgence for either exchange, lands_aid,
// natural_recovery for the cast or the recovery, natures_sanctuary for create, move and end, nature_magician, primal_strike) and nothing for nil; srd5e.class is ValueClassDruid.
// Path: per case (happy / unhappy: boundary).
func TestFeatureAttribute(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		use  character.DruidFeatureUse
		want character.DruidFeatureAttribute
	}{
		{character.DruidUseAssumeWildShape{}, character.DruidFeatureWildShape}, {character.DruidUseEndWildShape{}, character.DruidFeatureWildShape},
		{character.DruidUseWildCompanion{}, character.DruidFeatureWildCompanion}, {character.DruidUseSlotForWildShapeUse{}, character.DruidFeatureWildResurgence},
		{character.DruidUseWildShapeUseForSlot{}, character.DruidFeatureWildResurgence}, {character.DruidUseLandsAid{}, character.DruidFeatureLandsAid},
		{character.DruidUseNaturalRecoveryCast{}, character.DruidFeatureNaturalRecovery}, {character.DruidUseNaturalRecoveryRecover{}, character.DruidFeatureNaturalRecovery},
		{character.DruidUseCreateSanctuary{}, character.DruidFeatureNaturesSanctuary}, {character.DruidUseMoveSanctuary{}, character.DruidFeatureNaturesSanctuary},
		{character.DruidUseEndSanctuary{}, character.DruidFeatureNaturesSanctuary}, {character.DruidUseNatureMagician{}, character.DruidFeatureNatureMagician},
		{character.DruidUsePrimalStrike{}, character.DruidFeaturePrimalStrike},
	} {
		if got, ok := druRules.FeatureAttribute(c.use); !ok || got != c.want {
			t.Errorf("FeatureAttribute(%T) = %v, %v; want %v", c.use, got, ok, c.want)
		}
	}
	if got, ok := druRules.FeatureAttribute(nil); ok || got != 0 {
		t.Errorf("FeatureAttribute(nil) = %v, %v", got, ok)
	}
	if character.ValueClassDruid != "druid" || character.DruidFeatureNaturesSanctuary.String() != "natures_sanctuary" {
		t.Error("class or feature value misnamed")
	}
}
