package scenarios_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpAMust returns v, or panics with err (a scenario's own setup, never the code under test).
func grpAMust[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// grpAEffects builds the ongoing effects of a cursed, blessed hero: the spell Bless and the hag's curse c1.
func grpAEffects() combat.OngoingEffects {
	eid := func(s string) combat.OngoingEffectID { return grpAMust(combat.NewOngoingEffectID(s)) }
	bless := grpAMust(combat.NewOngoingEffect(eid("bless"), combat.OngoingEffectSpell, grpAMust(combat.SpellSource(content.MustSpellID("bless")))))
	curse := grpAMust(combat.NewOngoingEffect(eid("c1"), combat.OngoingEffectCurse, grpAMust(combat.LabeledMagicalSource(content.MustFeatureID("hag_curse")))))
	return grpAMust(combat.NewOngoingEffects([]combat.OngoingEffect{bless, curse}))
}

func grpAEffectIDs(e combat.OngoingEffects) []string {
	var out []string
	for _, x := range e.All() {
		out = append(out, x.ID().String())
	}
	return out
}

func grpAMagicKinds(events []combat.MagicalEffectEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// Requirement: doc/specs/magical-effects.md FR-030, FR-031, FR-032, FR-033, FR-148 - Remove Curse ends the hag's curse on a hero and leaves Bless; removing it again finds nothing; naming Bless is not naming a curse and reports its category; the event carries the curse and the means.
// Path: unhappy (state error).
func TestMagicalEffectsCurseRemovalScenario(t *testing.T) {
	t.Parallel()
	ctx, hero := context.Background(), combat.MustCombatantID("hero")
	effects := grpAEffects()
	remove := func(effects combat.OngoingEffects, id string, means combat.CurseRemovalMeans) combat.RemoveCombatantCurseResult {
		t.Helper()
		res, err := combat.NewRemoveCombatantCurse().Run(ctx, combat.RemoveCombatantCurseCommand{Creature: hero, Effects: effects, Curse: grpAMust(combat.NewOngoingEffectID(id)), Means: means})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	applied, ok := remove(effects, "c1", combat.RemoveCurseSpell).(combat.RemoveCurseApplied)
	if !ok || !slices.Equal(grpAEffectIDs(applied.Effects()), []string{"bless"}) || !slices.Equal(grpAMagicKinds(applied.Events()), []string{"curse_removed"}) {
		t.Fatalf("remove curse = %#v, want only Bless left and one curse_removed", applied)
	}
	if ev := applied.Events()[0].(combat.CurseRemoved); ev.Effect().String() != "c1" || ev.Means() != combat.RemoveCurseSpell {
		t.Errorf("event = %+v, want curse c1 removed by the Remove Curse spell", ev)
	}
	if !slices.Equal(grpAEffectIDs(effects), []string{"bless", "c1"}) {
		t.Errorf("the effects given were changed")
	}
	again, ok := remove(applied.Effects(), "c1", combat.RemoveCurseSpell).(combat.RemoveCurseNotPresent)
	if !ok || again.Effects().Len() != 1 {
		t.Errorf("removing it twice = %#v, want NotPresent with the effects unchanged", again)
	}
	notCurse, ok := remove(effects, "bless", combat.GreaterRestorationSpell).(combat.RemoveCurseNotACurse)
	if !ok || notCurse.Category() != combat.OngoingEffectSpell || notCurse.Effects().Len() != 2 {
		t.Errorf("naming Bless = %#v, want NotACurse of category spell, effects unchanged", notCurse)
	}
}

// Requirement: doc/specs/magical-effects.md FR-047 to FR-055, FR-148 - a hero under a curse shifts into a polymorph form: everything carries over by default, an exception drops the named condition and the curse and reports them, an exception naming a condition not held is refused, the hero reverts (the curse comes back with it), and on death a shifted creature reverts with the cause died while a living one does not.
// Path: unhappy (rule interaction).
func TestMagicalEffectsShapeShiftScenario(t *testing.T) {
	t.Parallel()
	ctx, hero := context.Background(), combat.MustCombatantID("hero")
	run := func(req combat.ShapeShiftingRequest) combat.ShiftCombatantShapeResult {
		t.Helper()
		res, err := combat.NewShiftCombatantShape().Run(ctx, combat.ShiftCombatantShapeCommand{Request: req})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	polymorph := grpAMust(combat.SpellSource(content.MustSpellID("polymorph")))
	conditions := combat.MustConditionSet([]combat.Condition{combat.Poisoned{}, combat.Frightened{Source: combat.MustCombatantID("scary")}}, nil)
	effects := grpAEffects()
	shift := func(rule combat.CarryOverRule) combat.ShiftShapeCommand {
		return combat.ShiftShapeCommand{Creature: hero, State: combat.TrueForm(), Conditions: conditions, Effects: effects, Effect: polymorph, Rule: rule}
	}

	all, ok := run(combat.ShiftRequest{Command: shift(nil)}).(combat.ShiftShapeApplied)
	if !ok || !all.State().IsShifted() || !all.Conditions().Equal(conditions) || all.Effects().Len() != 2 || !all.NotCarriedOver().Empty() ||
		!slices.Equal(grpAMagicKinds(all.Events()), []string{"shape_shifted"}) {
		t.Fatalf("shift with the default rule = %#v, want shifted, everything carried over, one shape_shifted", all)
	}
	except := grpAMust(combat.NewCarryOverExcept([]content.ConditionKind{content.ConditionPoisoned}, []combat.OngoingEffectID{grpAMust(combat.NewOngoingEffectID("c1"))}))
	dropped, ok := run(combat.ShiftRequest{Command: shift(except)}).(combat.ShiftShapeApplied)
	if !ok || slices.Contains(dropped.Conditions().Effective(), content.ConditionPoisoned) || !slices.Equal(grpAEffectIDs(dropped.Effects()), []string{"bless"}) || dropped.NotCarriedOver().Empty() {
		t.Fatalf("shift with an exception = %#v, want no Poisoned, no curse and the omissions reported", dropped)
	}
	missing := grpAMust(combat.NewCarryOverExcept([]content.ConditionKind{content.ConditionBlinded}, nil))
	if got, ok := run(combat.ShiftRequest{Command: shift(missing)}).(combat.ShiftShapeExceptionNotHeld); !ok || !slices.Equal(got.Conditions(), []content.ConditionKind{content.ConditionBlinded}) {
		t.Errorf("an exception for a condition not held = %#v, want ExceptionNotHeld naming Blinded", got)
	}

	back, ok := run(combat.RevertRequest{Command: combat.RevertToTrueFormCommand{Creature: hero, State: dropped.State(), Conditions: dropped.Conditions(), Effects: dropped.Effects()}}).(combat.RevertToTrueFormApplied)
	if !ok || back.State().IsShifted() || !slices.Equal(grpAMagicKinds(back.Events()), []string{"reverted_to_true_form"}) {
		t.Fatalf("revert = %#v, want true form and one reverted_to_true_form", back)
	}
	trueForm, ok := run(combat.RevertRequest{Command: combat.RevertToTrueFormCommand{Creature: hero, State: combat.TrueForm(), Conditions: conditions, Effects: effects}}).(combat.RevertToTrueFormApplied)
	if !ok || len(trueForm.Events()) != 0 {
		t.Errorf("reverting a creature already in true form = %#v, want applied and no event", trueForm)
	}

	dead := grpAMust(combat.NewDead(combat.MassiveDamage))
	died, ok := run(combat.RevertOnDeathRequest{Creature: hero, State: all.State(), Life: dead}).(combat.RevertOnDeathApplied)
	if !ok || died.Events()[0].(combat.RevertedToTrueForm).Cause() != combat.RevertCauseDied {
		t.Errorf("death while shifted = %#v, want reverted with cause died", died)
	}
	if got := run(combat.RevertOnDeathRequest{Creature: hero, State: all.State(), Life: combat.Alive{}}); got != combat.ShiftCombatantShapeResult(combat.RevertOnDeathNotDead{}) {
		t.Errorf("a living shifted creature = %#v, want NotDead", got)
	}
	if _, err := combat.NewShiftCombatantShape().Run(ctx, combat.ShiftCombatantShapeCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command with no request err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewShiftCombatantShape().Run(ctx, combat.ShiftCombatantShapeCommand{Request: combat.ShiftRequest{Command: combat.ShiftShapeCommand{}}}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a shift with no creature err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/magical-effects.md FR-056 to FR-063, FR-066 to FR-069 - a mage teleports: to a free, seen space it arrives at the destination; to an occupied one it arrives near it; a destination it must see and cannot is refused before placement is considered; a touched creature stays behind unless the effect says it accompanies; the teleporter in the touched set is an invalid result naming the offender; a negative cost is invalid input.
// Path: unhappy (rule interaction).
func TestMagicalEffectsTeleportScenario(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), combat.MustCombatantID
	mage, squire := id("mage"), id("squire")
	run := func(f combat.TeleportFields) combat.TeleportCombatantResult {
		t.Helper()
		f.Teleporter = mage
		res, err := combat.NewTeleportCombatant().Run(ctx, combat.TeleportCombatantCommand{Command: grpAMust(combat.NewTeleportCommand(f))})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	at, ok := run(combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightRequired, DestinationSeen: true, MovementCostFeet: 0, Touched: []combat.CombatantID{squire}}).(combat.TeleportArrivedAtDestination)
	if !ok || len(at.TeleportingCreatures()) != 0 || at.MovementCostFeet() != 0 || !slices.Equal(grpAMagicKinds(at.Events()), []string{"creature_teleported"}) {
		t.Fatalf("a free, seen destination with a touched creature = %#v, want arrival at it, the squire left behind, no cost, one creature_teleported", at)
	}
	if ev := at.Events()[0].(combat.CreatureTeleported); ev.Arrival() != combat.ArrivalAtDestination || len(ev.Accompanying()) != 0 {
		t.Errorf("event = %+v, want arrival at the destination with nobody accompanying", ev)
	}
	near, ok := run(combat.TeleportFields{Destination: combat.OccupiedByCreature, Sight: combat.DestinationSightNotRequired, Policy: combat.Accompany, MovementCostFeet: 15, Touched: []combat.CombatantID{squire}}).(combat.TeleportArrivedNearDestination)
	if !ok || !slices.Equal(near.TeleportingCreatures(), []combat.CombatantID{squire}) || near.MovementCostFeet() != 15 {
		t.Errorf("an occupied destination with an accompanying squire = %#v, want arrival near it, squire teleporting, cost 15", near)
	}
	if blocked, ok := run(combat.TeleportFields{Destination: combat.BlockedBySolidObstacle, Sight: combat.DestinationSightNotRequired}).(combat.TeleportArrivedNearDestination); !ok {
		t.Errorf("a destination inside a solid obstacle = %#v, want arrival near it", blocked)
	}
	if got := run(combat.TeleportFields{Destination: combat.OccupiedByCreature, Sight: combat.DestinationSightRequired}); got != combat.TeleportCombatantResult(combat.TeleportDestinationNotSeen{}) {
		t.Errorf("an unseen destination that must be seen = %#v, want NotSeen before placement", got)
	}
	if _, ok := run(combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired}).(combat.TeleportArrivedAtDestination); !ok {
		t.Errorf("an unseen destination that need not be seen must be reachable")
	}
	bad, ok := run(combat.TeleportFields{Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, Touched: []combat.CombatantID{mage, squire, squire}}).(combat.TeleportInvalidTouchedSet)
	if !ok || !slices.Equal(bad.Offending(), []combat.CombatantID{mage, squire}) {
		t.Errorf("the teleporter and a repeat in the touched set = %#v, want Invalid naming both", bad)
	}
	if _, err := combat.NewTeleportCommand(combat.TeleportFields{Teleporter: mage, Destination: combat.Unoccupied, Sight: combat.DestinationSightNotRequired, MovementCostFeet: -5}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a negative cost err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewTeleportCombatant().Run(ctx, combat.TeleportCombatantCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command not built by NewTeleportCommand err = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/magical-effects.md FR-091 to FR-095, FR-144 - two effects trigger at once: the player decides on a player's turn and the Game Master on a monster's, the chosen order is returned exactly as chosen, an order that omits, repeats or invents an id is rejected with every problem listed, a single pending effect needs no decision and a pending id listed twice is rejected.
// Path: unhappy (invalid input).
func TestMagicalEffectsSimultaneousOrderScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pending := func(ids ...string) []combat.PendingEffectID {
		var out []combat.PendingEffectID
		for _, s := range ids {
			out = append(out, grpAMust(combat.NewPendingEffectID(s)))
		}
		return out
	}
	run := func(h combat.TurnHolder, pend, chosen []combat.PendingEffectID) combat.ResolveEffectOrderResult {
		t.Helper()
		res, err := combat.NewResolveEffectOrder().Run(ctx, combat.ResolveEffectOrderCommand{Holder: h, Pending: pend, Chosen: chosen})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	for holder, want := range map[combat.TurnHolder]combat.OrderingAuthority{
		combat.TurnHolderPlayer: combat.OrderingAuthorityPlayer, combat.TurnHolderGameMaster: combat.OrderingAuthorityGameMaster} {
		ordered, ok := run(holder, pending("fire", "cold", "acid"), pending("acid", "fire", "cold")).(combat.OrderSimultaneousEffectsOrdered)
		if !ok || ordered.Authority() != want || !slices.Equal(ordered.Order(), pending("acid", "fire", "cold")) {
			t.Errorf("holder %v: result = %#v, want authority %v and the order exactly as chosen", holder, ordered, want)
		}
	}
	if one, ok := run(combat.TurnHolderPlayer, pending("fire"), nil).(combat.OrderSimultaneousEffectsOrdered); !ok || !slices.Equal(one.Order(), pending("fire")) {
		t.Errorf("a single pending effect = %#v, want ordered as it is", one)
	}
	rejected, ok := run(combat.TurnHolderPlayer, pending("fire", "cold"), pending("fire", "fire", "ghost")).(combat.OrderSimultaneousEffectsRejected)
	if !ok {
		t.Fatalf("a bad order must be rejected")
	}
	var kinds []combat.OrderProblemKind
	for _, p := range rejected.Problems() {
		kinds = append(kinds, p.Kind)
	}
	for _, want := range []combat.OrderProblemKind{combat.OrderProblemRepeated, combat.OrderProblemUnknown, combat.OrderProblemMissing} {
		if !slices.Contains(kinds, want) {
			t.Errorf("problems = %v, want %v among them (every problem is listed)", kinds, want)
		}
	}
	if dup, ok := run(combat.TurnHolderPlayer, pending("fire", "fire"), pending("fire", "fire")).(combat.OrderSimultaneousEffectsRejected); !ok || dup.Problems()[0].Kind != combat.OrderProblemDuplicatePending {
		t.Errorf("a pending id twice = %#v, want rejected with DuplicatePending first", dup)
	}
}

// Requirement: doc/specs/magical-effects.md FR-018, FR-019 to FR-025, FR-144 - a Fireball's area is assessed location by location: inside with a line open through Half Cover is in; outside is out whatever its lines; inside with every line blocked by Total Cover is out; inside with no lines is not decidable; the origin is in exactly when the shape (or its creator) says so; an origin behind an obstruction resolves to the near side, and an undefined placement is invalid input.
// Path: unhappy (rule interaction).
func TestMagicalEffectsAreaLocationScenario(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assess := func(f combat.AreaLocationFacts) combat.AssessAreaLocationResult {
		t.Helper()
		res, err := combat.NewAssessAreaLocation().Run(ctx, combat.AssessAreaLocationCommand{Request: combat.AssessMembershipRequest{Facts: f}})
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	other := func(c combat.ShapeContainment, lines ...combat.Cover) combat.AreaLocationFacts {
		return grpAMust(combat.NewOtherLocationFacts(c, lines))
	}
	for name, c := range map[string]struct {
		facts combat.AreaLocationFacts
		want  combat.AssessAreaLocationResult
	}{
		"inside with a half-cover line":           {other(combat.InsideShape, combat.TotalCover, combat.HalfCover), combat.AreaMembershipIncluded{}},
		"outside with open lines":                 {other(combat.OutsideShape, combat.Cover(0)), combat.AreaMembershipExcluded{Reason: combat.ExcludedOutsideShape}},
		"outside and fully blocked: outside wins": {other(combat.OutsideShape, combat.TotalCover), combat.AreaMembershipExcluded{Reason: combat.ExcludedOutsideShape}},
		"inside with every line blocked":          {other(combat.InsideShape, combat.TotalCover, combat.TotalCover), combat.AreaMembershipExcluded{Reason: combat.ExcludedAllLinesBlocked}},
		"inside with no lines":                    {other(combat.InsideShape), combat.AreaMembershipNoLinesConsidered{}},
	} {
		if got := assess(c.facts); got != c.want {
			t.Errorf("%s: result = %#v, want %#v", name, got, c.want)
		}
	}
	sphere := grpAMust(content.NewCylinderArea(10, 40)) // a cylinder includes its origin by default
	byDefault := grpAMust(combat.ShapeDefaultOrigin(sphere))
	if got := assess(grpAMust(combat.NewOriginLocationFacts(byDefault))); got != combat.AssessAreaLocationResult(combat.AreaMembershipIncluded{}) {
		t.Errorf("the origin of a cylinder = %#v, want included", got)
	}
	cone := grpAMust(content.NewConeArea(60)) // a cone leaves its origin out unless its creator says otherwise
	excluded := grpAMust(combat.ShapeDefaultOrigin(cone))
	if got := assess(grpAMust(combat.NewOriginLocationFacts(excluded))); got != combat.AssessAreaLocationResult(combat.AreaMembershipExcluded{Reason: combat.ExcludedOrigin}) {
		t.Errorf("the origin of a cone = %#v, want excluded as the origin", got)
	}
	chosen := grpAMust(combat.CreatorIncludesOrigin(combat.OverridableCone{Area: cone}))
	if got := assess(grpAMust(combat.NewOriginLocationFacts(chosen))); got != combat.AssessAreaLocationResult(combat.AreaMembershipIncluded{}) {
		t.Errorf("the origin of a cone its creator includes = %#v, want included", got)
	}

	origin := func(p combat.OriginPlacement) (combat.AssessAreaLocationResult, error) {
		return combat.NewAssessAreaLocation().Run(ctx, combat.AssessAreaLocationCommand{Request: combat.ResolveOriginRequest{Placement: p}})
	}
	for p, want := range map[combat.OriginPlacement]combat.AreaOriginResolution{
		combat.SeenPoint: combat.ChosenPoint, combat.UnseenPointNoObstruction: combat.ChosenPoint, combat.UnseenPointBehindObstruction: combat.NearSideOfObstruction} {
		if got, err := origin(p); err != nil || got != combat.AssessAreaLocationResult(want) {
			t.Errorf("placement %v = %#v, %v, want %v", p, got, err, want)
		}
	}
	if _, err := origin(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined placement err = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewAssessAreaLocation().Run(ctx, combat.AssessAreaLocationCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a command with no request err = %v, want ErrInvalidInput", err)
	}
}
