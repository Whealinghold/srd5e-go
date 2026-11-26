package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/initiative-and-actions.md FR-044, FR-150 - StandardAction has exactly the twelve actions and SimpleStandardAction the ten other than Dash and Ready, each mapping to the StandardAction of the same name; each choice records the StandardAction it names.
// Path: happy.
func TestStandardActionSets(t *testing.T) {
	t.Parallel()
	all := []string{"attack", "dash", "disengage", "dodge", "help", "hide", "influence", "magic", "ready", "search", "study", "utilize"}
	for i, name := range all {
		a := combat.StandardAction(i + 1)
		if a.String() != name || !a.Valid() {
			t.Errorf("StandardAction %d = %q valid %v, want %q", i+1, a, a.Valid(), name)
		}
	}
	for _, bad := range []combat.StandardAction{0, 13} {
		if bad.Valid() || bad.String() != "unknown" {
			t.Errorf("StandardAction %d is valid or named %q", bad, bad)
		}
	}
	var simple []string
	for s := combat.SimpleStandardAction(1); s.Valid(); s++ {
		simple = append(simple, s.String())
		if s.Standard().String() != s.String() || !s.Standard().Valid() {
			t.Errorf("simple %d maps to %v", s, s.Standard())
		}
	}
	want := slices.DeleteFunc(slices.Clone(all), func(s string) bool { return s == "dash" || s == "ready" })
	if !slices.Equal(simple, want) {
		t.Errorf("SimpleStandardAction members = %v, want %v [FR-044]", simple, want)
	}
	if combat.SimpleStandardAction(0).Standard() != 0 || combat.SimpleStandardAction(11).Standard() != 0 || combat.SimpleStandardAction(11).Valid() {
		t.Error("an undefined simple action maps to an action")
	}
	choices := []struct {
		c    combat.ActionChoice
		want combat.StandardAction
		ok   bool
	}{
		{combat.DashChoice{Speed: combat.SpeedNameWalk}, combat.StandardActionDash, true},
		{trkAttack, combat.StandardActionAttack, true},
		{combat.ReadyChoice{}, combat.StandardActionReady, true},
		{combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsDash{Speed: combat.SpeedNameFly}}, combat.StandardActionDash, true},
		{combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsSimpleAction{Action: combat.SimpleStandardActionHide}}, combat.StandardActionHide, true},
		{combat.FeatureActionChoice{Feature: "f", Granted: combat.NoStandardAction{}}, 0, false},
		{combat.FeatureActionChoice{Feature: "f"}, 0, false},
		{combat.ImprovisedActionChoice{}, 0, false},
	}
	for _, c := range choices {
		if got, ok := c.c.Standard(); got != c.want || ok != c.ok {
			t.Errorf("%#v records %v, %v; want %v, %v [FR-150]", c.c, got, ok, c.want, c.ok)
		}
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-087 - NewBonusActionGrant, NewAdditionalActionGrant, NewAnyActionBut and NewOnlyActions reject an invalid source, feature, action or an empty OnlyActions with an invalid-input error.
// Path: unhappy (invalid input).
func TestGrantConstructors(t *testing.T) {
	t.Parallel()
	bad := []struct {
		name string
		err  error
	}{
		{"nil source", func() error { _, e := combat.NewBonusActionGrant(nil, combat.NoStandardAction{}); return e }()},
		{"feature source without id", func() error {
			_, e := combat.NewBonusActionGrant(combat.BonusActionFromFeature{}, combat.NoStandardAction{})
			return e
		}()},
		{"item source without id", func() error {
			_, e := combat.NewBonusActionGrant(combat.BonusActionFromItem{}, combat.NoStandardAction{})
			return e
		}()},
		{"nil granted", func() error {
			_, e := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "f"}, nil)
			return e
		}()},
		{"granted Dash without a speed", func() error {
			_, e := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "f"}, combat.GrantsDash{})
			return e
		}()},
		{"granted simple action undefined", func() error {
			_, e := combat.NewBonusActionGrant(combat.BonusActionFromFeature{Feature: "f"}, combat.GrantsSimpleAction{})
			return e
		}()},
		{"additional action without a scope", func() error { _, e := combat.NewAdditionalActionGrant("f", nil); return e }()},
		{"additional action without a feature", func() error { _, e := combat.NewAdditionalActionGrant("", combat.AnyActionBut{}); return e }()},
		{"empty OnlyActions", func() error { _, e := combat.NewOnlyActions(); return e }()},
		{"OnlyActions with an undefined action", func() error { _, e := combat.NewOnlyActions(0); return e }()},
		{"AnyActionBut with an undefined action", func() error { _, e := combat.NewAnyActionBut(13); return e }()},
	}
	for _, c := range bad {
		if !errors.Is(c.err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, want invalid input", c.name, c.err)
		}
	}
	any1, err := combat.NewAnyActionBut(combat.StandardActionMagic, combat.StandardActionAttack, combat.StandardActionMagic)
	if err != nil || !slices.Equal(any1.Actions(), []combat.StandardAction{combat.StandardActionAttack, combat.StandardActionMagic}) {
		t.Errorf("NewAnyActionBut = %v, %v, want sorted and deduplicated", any1.Actions(), err)
	}
	only, _ := combat.NewOnlyActions(combat.StandardActionDash)
	g, _ := combat.NewAdditionalActionGrant("haste", only)
	if g.Feature() != "haste" || !slices.Equal(g.Scope().(combat.OnlyActions).Actions(), only.Actions()) || g.IsZero() {
		t.Errorf("grant = %+v", g)
	}
	g2, _ := combat.NewAdditionalActionGrant("haste", combat.AnyActionBut{})
	if g.Equal(g2) || !g.Equal(g) || g2.Equal(combat.AdditionalActionGrant{}) || !(combat.AdditionalActionGrant{}).Equal(combat.AdditionalActionGrant{}) {
		t.Error("Equal of grants is wrong")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-159, FR-158 - the five ActivationTime values map to the Magic action, the Utilize action, the item's Bonus Action, and the described cost.
// Path: per case.
func TestActivationCosts(t *testing.T) {
	t.Parallel()
	got := func(at content.ActivationTime) combat.ActivationCost {
		return combat.ActivationCosts{}.Of(at, "potion_of_healing")
	}
	if c := got(content.ActivationMagicAction); c != (combat.ActivatesByAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}}) {
		t.Errorf("MagicAction: %#v", c)
	}
	if c := got(content.ActivationAction); c != (combat.ActivatesByAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionUtilize}}) {
		t.Errorf("Action: %#v", c)
	}
	bonus, ok := got(content.ActivationBonusAction).(combat.ActivatesByBonusAction)
	if !ok || bonus.Grant.Source() != (combat.BonusActionFromItem{Item: "potion_of_healing"}) || bonus.Grant.Granted() != (combat.NoStandardAction{}) {
		t.Errorf("BonusAction: %#v", bonus)
	}
	for _, at := range []content.ActivationTime{content.ActivationReaction, content.ActivationDescribedElsewhere, 0, 99} {
		if got(at) != (combat.ActivationCostDescribed{}) {
			t.Errorf("%v: %#v, want described", at, got(at))
		}
	}
	// The Bonus Action cost is usable as a grant.
	tr, ev := trkApplied(t)(trkABC(t).TakeBonusAction(trkA, trkAvail, combat.EntitledToBonusAction(bonus)))
	if tr.Resources().BonusAction() != combat.ResourceSpent || ev[0].(combat.BonusActionUsed).Source() != bonus.Grant.Source() {
		t.Errorf("the item Bonus Action: %+v", ev)
	}
}

// Requirement: doc/specs/mounted-and-underwater-combat.md F-8 (amendment in initiative-and-actions.md) - a controlled mount permits exactly Dash, Disengage and Dodge as simple choices; an independent mount permits everything; a control that is not valid or a nil choice is not permitted.
// Path: per case.
func TestPermitsMountAction(t *testing.T) {
	t.Parallel()
	var controlled []string
	for s := combat.SimpleStandardAction(1); s.Valid(); s++ {
		if combat.PermitsMountAction(combat.MountControlControlled, combat.SimpleActionChoice{Action: s}) {
			controlled = append(controlled, s.String())
		}
		if !combat.PermitsMountAction(combat.MountControlIndependent, combat.SimpleActionChoice{Action: s}) {
			t.Errorf("an independent mount forbids %v", s)
		}
	}
	if !slices.Equal(controlled, []string{"disengage", "dodge"}) {
		t.Errorf("a controlled mount permits %v of the ten simple actions, want disengage and dodge", controlled)
	}
	if !combat.PermitsMountAction(combat.MountControlControlled, combat.DashChoice{Speed: combat.SpeedNameFly}) {
		t.Error("a controlled mount forbids Dash")
	}
	for name, c := range map[string]combat.ActionChoice{"improvised": combat.ImprovisedActionChoice{}, "feature": combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsDash{Speed: combat.SpeedNameWalk}}, "ready": combat.ReadyChoice{}} {
		if combat.PermitsMountAction(combat.MountControlControlled, c) || !combat.PermitsMountAction(combat.MountControlIndependent, c) {
			t.Errorf("%s: controlled permits it or independent forbids it", name)
		}
	}
	if combat.PermitsMountAction(0, trkAttack) || combat.PermitsMountAction(combat.MountControlIndependent, nil) || combat.PermitsMountAction(combat.MountControlControlled, nil) {
		t.Error("an invalid control or a nil choice is permitted")
	}
}

// Requirement: doc/specs/initiative-and-actions.md FR-150, FR-151, FR-155 - every part of a feature choice must be valid; a control that is not defined permits nothing; a grant compares by feature, scope kind and actions.
// Path: unhappy (invalid input).
func TestActionChoiceValidityAndGrantEquality(t *testing.T) {
	t.Parallel()
	base := trkABC(t)
	for name, c := range map[string]combat.ActionChoice{
		"feature without an id":      combat.FeatureActionChoice{Granted: combat.NoStandardAction{}},
		"feature without a grant":    combat.FeatureActionChoice{Feature: "f"},
		"feature with a bad grant":   combat.FeatureActionChoice{Feature: "f", Granted: combat.GrantsDash{}},
		"feature id and bad grant":   combat.FeatureActionChoice{Feature: "", Granted: combat.GrantsDash{}},
		"dash without a speed":       combat.DashChoice{},
		"simple action not defined":  combat.SimpleActionChoice{},
		"ready without an action":    combat.ReadyChoice{},
		"simple action past the end": combat.SimpleActionChoice{Action: 11},
	} {
		if _, err := base.TakeAction(trkA, trkAvail, c); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v, want invalid input", name, err)
		}
	}
	for _, control := range []combat.MountControl{-1, 0, 3} {
		if combat.PermitsMountAction(control, trkAttack) || combat.PermitsMountAction(control, combat.DashChoice{Speed: combat.SpeedNameWalk}) {
			t.Errorf("control %d permits an action", control)
		}
	}
	anyMagic, _ := combat.NewAnyActionBut(combat.StandardActionMagic)
	onlyMagic, _ := combat.NewOnlyActions(combat.StandardActionMagic)
	onlyDash, _ := combat.NewOnlyActions(combat.StandardActionDash)
	g := func(f string, s combat.AdditionalActionScope) combat.AdditionalActionGrant {
		x, err := combat.NewAdditionalActionGrant(content.FeatureID(f), s)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	if g("f", anyMagic).Equal(g("f", onlyMagic)) || g("f", onlyMagic).Equal(g("f", anyMagic)) {
		t.Error("an AnyActionBut and an OnlyActions with the same actions compare equal")
	}
	if g("f", onlyMagic).Equal(g("g", onlyMagic)) || g("f", onlyMagic).Equal(g("f", onlyDash)) || !g("f", onlyMagic).Equal(g("f", onlyMagic)) {
		t.Error("grants compare by feature and actions wrongly")
	}
	if (combat.AdditionalActionGrant{}).Equal(g("f", onlyMagic)) || g("f", onlyMagic).Equal(combat.AdditionalActionGrant{}) {
		t.Error("the zero grant equals a real one")
	}
}
