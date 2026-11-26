package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sumFamiliarCast casts Find Familiar for "pc" with the form, creature type and origin, with the monsters given.
func sumFamiliarCast(tb testing.TB, roster combat.SummonedRoster, id string, form character.FamiliarForm, typ content.CreatureType, origin combat.FamiliarOrigin, monsters []content.Monster) combat.SummonedTransition {
	tb.Helper()
	req := combat.FindFamiliarRequest{Creature: cid(id), Level: 1, Form: form, Type: typ, Origin: origin}
	return roster.Summon(sumCmd(tb, roster, "pc", req, sumScene()), monsters...)
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-043, FR-054, Table P - a familiar form is eligible when it is one of bat, cat, frog, hawk, lizard, octopus, owl, rat, raven, spider and weasel, or a looked-up Beast of Challenge Rating 0; a special form (Imp, Pseudodragon, Quasit, Skeleton, Sphinx of Wonder, Sprite, Venomous Snake) only for Pact of the Chain; every other form is refused with FormNotEligibleRejection (SRD 5.2.1, "Find Familiar", p. 130; "Pact of the Chain", p. 74).
// Path: per case.
func TestFamiliarFormEligibility(t *testing.T) {
	t.Parallel()
	spell, pact, wild := combat.FamiliarOriginFindFamiliarSpell, combat.FamiliarOriginPactOfTheChain, combat.FamiliarOriginWildCompanion
	type row struct {
		name, requirement, path string
		form                    character.FamiliarForm
		origin                  combat.FamiliarOrigin
		eligible                bool
	}
	rows := []row{
		{"a looked-up Beast of Challenge Rating 0", "FR-043", "happy", normalForm("fixture_beast"), spell, true},
		{"a Beast of Challenge Rating 1", "FR-043", "unhappy (boundary)", normalForm("fixture_strong_beast"), spell, false},
		{"a creature of CR 0 that is not a Beast", "FR-043", "unhappy (boundary)", normalForm("fixture_giant"), spell, false},
		{"a special form for Pact of the Chain", "FR-054, p. 74", "happy", specialForm("imp"), pact, true},
		{"a special form for the spell", "FR-054", "unhappy (rule interaction)", specialForm("imp"), spell, false},
		{"a special form for Wild Companion", "FR-054", "unhappy (rule interaction)", specialForm("imp"), wild, false},
		{"a Beast of CR 0 for Pact of the Chain", "FR-054", "happy", normalForm("fixture_beast"), pact, true},
		{"an eighth special form", "FR-054, p. 74", "unhappy (invalid input)", specialForm("fixture_beast"), pact, false},
		{"a normal form named as special", "FR-054", "unhappy (invalid input)", specialForm("cat"), pact, false},
	}
	for _, id := range []string{"bat", "cat", "frog", "hawk", "lizard", "octopus", "owl", "rat", "raven", "spider", "weasel"} {
		rows = append(rows, row{"the listed form " + id, "FR-043, p. 130", "happy", normalForm(id), spell, true})
	}
	for _, id := range []string{"imp", "pseudodragon", "quasit", "skeleton", "sphinx_of_wonder", "sprite", "venomous_snake"} {
		rows = append(rows, row{"the special form " + id, "FR-054, p. 74", "happy", specialForm(id), pact, true})
	}
	for _, c := range rows {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			// Every form is a Fiend of Challenge Rating 1, so the list of names is the only way to be eligible; the Beast rows override.
			monsters := sumMonsters(t)
			if id := c.form.Monster().String(); !slices.ContainsFunc(monsters, func(m content.Monster) bool { return m.ID().String() == id }) {
				monsters = append(monsters, sumMonster(t, id, content.CreatureTypeFiend, content.SizeTiny, "1", 200))
			} else if slices.Contains([]string{"cat"}, id) {
				monsters[slices.IndexFunc(monsters, func(m content.Monster) bool { return m.ID().String() == id })] = sumMonster(t, id, content.CreatureTypeFiend, content.SizeTiny, "1", 200)
			}
			got := sumFamiliarCast(t, combat.SummonedRoster{}, "f1", c.form, content.CreatureTypeFey, c.origin, monsters)
			if c.eligible {
				sumApplied(t, got)
				return
			}
			if _, ok := sumRejection(t, got).(combat.FormNotEligibleRejection); !ok {
				t.Errorf("transition = %#v, want FormNotEligibleRejection [%s]", got, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-043, FR-053 - the creature type of a familiar is the summoner's choice of Celestial, Fey or Fiend for a normal form, Fey for a Wild Companion familiar whatever the summoner would choose, and the printed type of a special form; a normal form with a choice that is not one of the three is invalid input (SRD 5.2.1, "Find Familiar", p. 130; "Level 2: Wild Companion", p. 43).
// Path: per case.
func TestFamiliarCreatureType(t *testing.T) {
	t.Parallel()
	spell, pact, wild := combat.FamiliarOriginFindFamiliarSpell, combat.FamiliarOriginPactOfTheChain, combat.FamiliarOriginWildCompanion
	cases := []struct {
		name, requirement, path string
		form                    character.FamiliarForm
		choice                  content.CreatureType
		origin                  combat.FamiliarOrigin
		want                    content.CreatureType
	}{
		{"Celestial", "FR-043", "happy", normalForm("cat"), content.CreatureTypeCelestial, spell, content.CreatureTypeCelestial},
		{"Fey", "FR-043", "happy", normalForm("cat"), content.CreatureTypeFey, spell, content.CreatureTypeFey},
		{"Fiend", "FR-043", "happy", normalForm("cat"), content.CreatureTypeFiend, pact, content.CreatureTypeFiend},
		{"Wild Companion is Fey whatever the choice", "FR-053", "unhappy (rule interaction)", normalForm("cat"), content.CreatureTypeFiend, wild, content.CreatureTypeFey},
		{"Wild Companion needs no choice", "FR-053", "unhappy (rule interaction)", normalForm("cat"), 0, wild, content.CreatureTypeFey},
		{"a special form keeps its printed type", "FR-043", "unhappy (rule interaction)", specialForm("imp"), content.CreatureTypeCelestial, pact, content.CreatureTypeFiend},
		{"another special form", "FR-043", "unhappy (rule interaction)", specialForm("pseudodragon"), 0, pact, content.CreatureTypeDragon},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := sumApplied(t, sumFamiliarCast(t, combat.SummonedRoster{}, "f1", c.form, c.choice, c.origin, sumMonsters(t)))
			l, _ := a.Roster().Link("f1")
			st, ok := l.Familiar()
			if !ok || st.CreatureType() != c.want || a.Creatures()[0].StatBlock().Type() != c.want || st.Origin() != c.origin || st.Form() != c.form || st.SightLinkActive() {
				t.Errorf("familiar = %+v block type %v, want type %v of origin %v [%s]", st, a.Creatures()[0].StatBlock().Type(), c.want, c.origin, c.requirement)
			}
		})
	}
	for _, typ := range []content.CreatureType{0, content.CreatureTypeBeast, content.CreatureTypeUndead} {
		req := combat.FindFamiliarRequest{Creature: cid("f1"), Level: 1, Form: normalForm("cat"), Type: typ, Origin: combat.FamiliarOriginFindFamiliarSpell}
		if _, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "pc", combat.SideA, req, sumFacts(t), sumScene()); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("a choice of %v error = %v, want srderr.ErrInvalidInput [FR-043, FR-067]", typ, err)
		}
	}
	for _, bad := range []combat.FindFamiliarRequest{
		{Creature: cid("f1"), Level: 1, Form: character.FamiliarForm{}, Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell},
		{Creature: cid("f1"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey},
	} {
		if _, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "pc", combat.SideA, bad, sumFacts(t), sumScene()); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%#v error = %v, want srderr.ErrInvalidInput [FR-067]", bad, err)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-044, FR-054, Table P (familiar recast) - a summoner has at most one familiar: casting again while one exists, present or in the pocket dimension, gives FamiliarFormChanged for the same CombatantID, never a second creature and never a refusal for existing; the new stat block replaces the old, the origin becomes that of the latest cast and the new form must be eligible for it.
// Path: per case.
func TestOnlyOneFamiliar(t *testing.T) {
	t.Parallel()
	spell, pact, wild := combat.FamiliarOriginFindFamiliarSpell, combat.FamiliarOriginPactOfTheChain, combat.FamiliarOriginWildCompanion
	monsters := sumMonsters(t)
	cases := []struct {
		name, requirement, path string
		first, second           combat.FamiliarOrigin
		form                    character.FamiliarForm
		inPocket                bool
		eligible                bool
		wantType                content.CreatureType
	}{
		{"a Wild Companion familiar recast by the spell", "Table P, Q-15", "unhappy (rule interaction)", wild, spell, normalForm("fixture_beast"), false, true, content.CreatureTypeFiend},
		{"a spell familiar recast by Wild Companion", "Table P, Q-15", "unhappy (rule interaction)", spell, wild, normalForm("fixture_beast"), false, true, content.CreatureTypeFey},
		{"a spell familiar recast as a special form by Pact of the Chain", "FR-054", "unhappy (rule interaction)", spell, pact, specialForm("imp"), false, true, content.CreatureTypeFiend},
		{"a Pact familiar recast as a special form by the spell", "FR-054", "unhappy (rule interaction)", pact, spell, specialForm("imp"), false, false, 0},
		{"a familiar in the pocket dimension", "FR-044", "unhappy (state error)", spell, spell, normalForm("fixture_beast"), true, true, content.CreatureTypeFiend},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roster := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", c.first)
			if c.inPocket {
				roster = sumApplied(t, sumChange(t, roster, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
			}
			prior := roster
			got := sumFamiliarCast(t, roster, "second", c.form, content.CreatureTypeFiend, c.second, monsters)
			if !c.eligible {
				if _, ok := sumRejection(t, got).(combat.FormNotEligibleRejection); !ok || !prior.Equal(roster) {
					t.Errorf("transition = %#v, want FormNotEligibleRejection and the roster unchanged [FR-054]", got)
				}
				return
			}
			a := sumApplied(t, got)
			ev, ok := a.Events()[0].(combat.FamiliarFormChanged)
			if !ok || len(a.Events()) != 1 || !slices.Equal(ev.Combatants(), []combat.CombatantID{"fam", "pc"}) || ev.Form() != c.form || ev.Origin() != c.second ||
				ev.CreatureType() != c.wantType || ev.HitPointMaximum() != 13 {
				t.Errorf("events = %#v, want FamiliarFormChanged for fam of the new form, origin and type [FR-044]", a.Events())
			}
			l, _ := a.Roster().Link("fam")
			st, _ := l.Familiar()
			if a.Roster().Len() != 1 || len(a.RosterActions()) != 0 || st.Origin() != c.second || st.CreatureType() != c.wantType || l.InPocketDimension() != c.inPocket {
				t.Errorf("roster = %d links, familiar %+v, pocket %v, want one familiar of the latest origin where it was [FR-044, Q-15]", a.Roster().Len(), st, l.InPocketDimension())
			}
			if spec := a.Creatures(); len(spec) != 1 || spec[0].Creature().ID != cid("fam") || spec[0].StatBlock().HitPointMaximum() != 13 || spec[0].StatBlock().Monster() != c.form.Monster() {
				t.Errorf("specs = %v, want the new stat block of the same creature, whose Hit Points become the form's maximum [Q-6]", spec)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-045, FR-058 - the summoner has NewTelepathic(100) toward a present familiar within 100 feet (exactly 100 accepted, 101 none, none in the pocket dimension); a steed has NewTelepathic(5280) and refuses a contact with anyone but its summoner (SRD 5.2.1, "Find Familiar", p. 130; "Otherworldly Steed", p. 131).
// Path: per case.
func TestCompanionTelepathy(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	pocket := sumApplied(t, sumChange(t, roster, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	for _, c := range []struct {
		name     string
		roster   combat.SummonedRoster
		summoner string
		distance int
		want     int
	}{
		{"within 100 feet", roster, "pc", 100, 100}, {"at 0", roster, "pc", 0, 100}, {"at 101", roster, "pc", 101, 0}, {"a negative distance", roster, "pc", -1, 0},
		{"in the pocket dimension", pocket, "pc", 10, 0}, {"no familiar", roster, "other", 10, 0},
	} {
		got := c.roster.FamiliarTelepathy(cid(c.summoner), c.distance)
		if got.RangeFeet() != c.want || got.IsTelepathic() != (c.want > 0) {
			t.Errorf("%s: FamiliarTelepathy = %v, want range %d [FR-045]", c.name, got, c.want)
		}
	}
	if got := roster.SteedTelepathy("st"); got != combat.MustTelepathic(5280) {
		t.Errorf("SteedTelepathy = %v, want 5280 feet [FR-058]", got)
	}
	if got := roster.SteedTelepathy("fam"); got.IsTelepathic() {
		t.Errorf("a familiar has the steed's telepathy [FR-058]")
	}
	if rej := roster.CheckSteedContact("st", "pc"); rej != nil {
		t.Errorf("a contact with the summoner = %#v, want none [FR-058]", rej)
	}
	for _, other := range []string{"stranger", "fam", "other"} {
		if rej := roster.CheckSteedContact("st", cid(other)); rej != (combat.TelepathyOnlyWithSummonerRejection{}) {
			t.Errorf("a contact with %s = %#v, want TelepathyOnlyWithSummonerRejection [FR-058]", other, rej)
		}
	}
	if rej := roster.CheckSteedContact("i1", "stranger"); rej != nil {
		t.Errorf("an insect's contact = %#v, want none: only a steed is limited [FR-058]", rej)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-046, Table P (sight link, 100 feet) - sharing a familiar's senses starts a sight link until the start of the summoner's next turn, accepted at 100 feet and refused at 101 (an interpretation), refused for a familiar in the pocket dimension or gone; a second link while one is active starts again; the start of the summoner's turn ends it with FamiliarSightLinkEnded (SRD 5.2.1, "Find Familiar", p. 130).
// Path: per case.
func TestFamiliarSightLink(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	pocket := sumApplied(t, sumChange(t, roster, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	gone := sumApplied(t, sumChange(t, roster, combat.ReportZeroHitPoints{Creature: cid("fam")}, "")).Roster()
	for _, c := range []struct {
		name, requirement, path string
		roster                  combat.SummonedRoster
		summoner                string
		distance                int
		ok                      bool
	}{
		{"at 100 feet", "Table P, Q-17", "unhappy (boundary)", roster, "pc", 100, true},
		{"at 101 feet", "Table P, Q-17", "unhappy (boundary)", roster, "pc", 101, false},
		{"adjacent", "FR-046", "happy", roster, "pc", 0, true},
		{"in the pocket dimension", "FR-046", "unhappy (state error)", pocket, "pc", 5, false},
		{"gone", "FR-046", "unhappy (state error)", gone, "pc", 5, false},
		{"a summoner with no familiar", "FR-046", "unhappy (state error)", roster, "other", 5, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumChange(t, c.roster, combat.ShareFamiliarSenses{Summoner: cid(c.summoner), DistanceFeet: c.distance}, "")
			if !c.ok {
				if _, ok := sumRejection(t, got).(combat.SensesNotSharableRejection); !ok {
					t.Errorf("transition = %#v, want SensesNotSharableRejection [%s]", got, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			if ev, ok := a.Events()[0].(combat.FamiliarSightLinkStarted); !ok || len(a.Events()) != 1 || !slices.Equal(ev.Combatants(), []combat.CombatantID{"fam", "pc"}) {
				t.Errorf("events = %v, want FamiliarSightLinkStarted [%s]", sumKinds(a.Events()), c.requirement)
			}
		})
	}
	started := sumApplied(t, sumChange(t, roster, combat.ShareFamiliarSenses{Summoner: cid("pc"), DistanceFeet: 5}, "")).Roster()
	l, _ := started.Link("fam")
	if st, _ := l.Familiar(); !st.SightLinkActive() {
		t.Fatalf("the sight link is not active")
	}
	again := sumApplied(t, sumChange(t, started, combat.ShareFamiliarSenses{Summoner: cid("pc"), DistanceFeet: 5}, ""))
	if got := sumKinds(again.Events()); !slices.Equal(got, []string{"familiar_sight_link_started"}) {
		t.Errorf("a second link gave %v, want it to start again [FR-046]", got)
	}
	ended := sumApplied(t, sumChange(t, started, combat.ReportTurnStarted{Combatant: cid("pc")}, ""))
	if got := sumKinds(ended.Events()); !slices.Equal(got, []string{"familiar_sight_link_ended"}) {
		t.Errorf("events = %v, want familiar_sight_link_ended at the start of the summoner's turn [FR-046]", got)
	}
	if l, _ := ended.Roster().Link("fam"); func() bool { st, _ := l.Familiar(); return st.SightLinkActive() }() {
		t.Errorf("the sight link outlived the turn start")
	}
	for _, id := range []string{"pc", "fam", "nobody"} {
		if _, ok := sumChange(t, ended.Roster(), combat.ReportTurnStarted{Combatant: cid(id)}, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("a turn start of %s with no link active changed the roster", id)
		}
	}
	if _, ok := sumChange(t, started, combat.ReportTurnStarted{Combatant: cid("other")}, "").(combat.SummonedUnchanged); !ok {
		t.Errorf("another combatant's turn ended the link")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-047 - a familiar delivers a touch spell when the spell has range Touch, the familiar is present within 100 feet and can take a Reaction; it spends its own Reaction through FeatureReaction(find_familiar, NoStandardAction); otherwise TouchNotDeliverableRejection with the first of the reasons not present, not Touch, too far, cannot react (SRD 5.2.1, "Find Familiar", p. 130; "Reactions", p. 10).
// Path: per case.
func TestAFamiliarDeliversATouch(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	pocket := sumApplied(t, sumChange(t, roster, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	avail, none := combat.ConditionActionAvailabilityAvailable, combat.ConditionActionAvailabilityNone
	touch := func(r combat.SummonedRoster, summoner string, spell bool, d int, av combat.ConditionActionAvailability) combat.SummonedTransition {
		return sumChange(t, r, combat.DeliverTouch{Summoner: cid(summoner), SpellIsTouch: spell, DistanceFeet: d, Familiar: av}, "")
	}
	cases := []struct {
		name, requirement, path string
		got                     combat.SummonedTransition
		reason                  combat.TouchReason
	}{
		{"delivered", "FR-047", "happy", touch(roster, "pc", true, 30, avail), 0},
		{"delivered at 100 feet", "FR-047", "unhappy (boundary)", touch(roster, "pc", true, 100, avail), 0},
		{"at 101 feet", "FR-047", "unhappy (boundary)", touch(roster, "pc", true, 101, avail), combat.TouchTooFar},
		{"a spell that is not Touch", "FR-047", "unhappy (invalid input)", touch(roster, "pc", false, 5, avail), combat.TouchNotTouch},
		{"a familiar that cannot react", "FR-047", "unhappy (state error)", touch(roster, "pc", true, 5, none), combat.TouchFamiliarCannotReact},
		{"in the pocket dimension", "FR-047", "unhappy (state error)", touch(pocket, "pc", true, 5, avail), combat.TouchNotPresent},
		{"no familiar", "FR-047", "unhappy (state error)", touch(roster, "other", true, 5, avail), combat.TouchNotPresent},
		{"not present before not Touch", "FR-047", "unhappy (rule interaction)", touch(pocket, "pc", false, 500, none), combat.TouchNotPresent},
		{"not Touch before too far", "FR-047", "unhappy (rule interaction)", touch(roster, "pc", false, 500, none), combat.TouchNotTouch},
		{"too far before cannot react", "FR-047", "unhappy (rule interaction)", touch(roster, "pc", true, 500, none), combat.TouchTooFar},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.reason == 0 {
				a := sumApplied(t, c.got)
				r, ok := a.Reaction()
				if want := (combat.FeatureReaction{Feature: "find_familiar", Granted: combat.NoStandardAction{}}); !ok || r != want || len(a.Events()) != 0 || !a.Roster().Equal(roster) {
					t.Errorf("reaction = %v, %v, events %v, want FeatureReaction(find_familiar) and nothing else changed [FR-047]", r, ok, a.Events())
				}
				return
			}
			if rej := sumRejection(t, c.got); rej != (combat.TouchNotDeliverableRejection{Reason: c.reason}) {
				t.Errorf("rejection = %#v, want TouchNotDeliverableRejection{%v} [%s]", rej, c.reason, c.requirement)
			}
		})
	}
	if r, ok := sumApplied(t, touch(roster, "pc", true, 5, avail)).Reaction(); !ok || r.Feature != "find_familiar" {
		t.Errorf("Reaction = %v", r)
	}
	if _, ok := sumApplied(t, sumChange(t, roster, combat.ShareFamiliarSenses{Summoner: cid("pc")}, "")).Reaction(); ok {
		t.Errorf("a sight link spent a Reaction")
	}
	if got := combat.TouchReason(0).String(); got != "unknown" {
		t.Errorf("TouchReason(0) = %q", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-048 (named case: a refused Attack action for a familiar) - a familiar is refused the Attack action with FamiliarCannotAttackRejection unless the request carries a FamiliarAttackGrant (via Reaction or via a Bonus Action command) from a feature; every other action is allowed; a creature that is not a familiar of the roster is refused (SRD 5.2.1, "Find Familiar", p. 130; "Pact of the Chain", p. 74).
// Path: per case.
func TestAFamiliarCannotAttackWithoutAGrant(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	cases := []struct {
		name, requirement, path string
		creature                string
		action                  combat.StandardAction
		grant                   combat.FamiliarAttackGrant
		want                    combat.SummonRejection
	}{
		{"the Attack action", "FR-048", "unhappy (rule interaction)", "fam", combat.StandardActionAttack, 0, combat.FamiliarCannotAttackRejection{}},
		{"the Attack action with a grant via Reaction", "FR-048, FR-091", "happy", "fam", combat.StandardActionAttack, combat.FamiliarAttackViaReaction, nil},
		{"the Attack action with a grant via a Bonus Action command", "FR-048, FR-093", "happy", "fam", combat.StandardActionAttack, combat.FamiliarAttackViaBonusActionCommand, nil},
		{"the Dodge action", "FR-048", "happy", "fam", combat.StandardActionDodge, 0, nil},
		{"the Help action", "FR-048", "happy", "fam", combat.StandardActionHelp, 0, nil},
		{"the Dash action", "FR-048", "happy", "fam", combat.StandardActionDash, 0, nil},
		{"a creature that is not a familiar", "FR-048", "unhappy (invalid input)", "i1", combat.StandardActionAttack, 0, combat.CommandNotYourCreatureRejection{Creature: cid("i1")}},
		{"an unknown creature", "FR-048", "unhappy (invalid input)", "ghost", combat.StandardActionDodge, 0, combat.CommandNotYourCreatureRejection{Creature: cid("ghost")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := sumChange(t, roster, combat.RequestFamiliarAction{Creature: cid(c.creature), Action: c.action, Grant: c.grant}, "")
			if c.want == nil {
				if a := sumApplied(t, got); !a.Roster().Equal(roster) || len(a.Events()) != 0 {
					t.Errorf("an allowed action changed the roster [%s]", c.requirement)
				}
				return
			}
			if rej := sumRejection(t, got); rej != c.want {
				t.Errorf("rejection = %#v, want %#v [%s]", rej, c.want, c.requirement)
			}
		})
	}
	for _, g := range []combat.FamiliarAttackGrant{combat.FamiliarAttackViaReaction, combat.FamiliarAttackViaBonusActionCommand, 0} {
		if (g.String() == "none") == g.Valid() {
			t.Errorf("FamiliarAttackGrant %d String %q Valid %v disagree", int(g), g.String(), g.Valid())
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-049, FR-019 - a familiar at 0 Hit Points departs and the summoner is marked FamiliarGone; casting Find Familiar again brings back a fresh creature at the new form's maximum Hit Points and clears the mark (SRD 5.2.1, "Find Familiar", p. 130).
// Path: unhappy (state error).
func TestAFamiliarThatDroppedToZeroCanBeCastAgain(t *testing.T) {
	t.Parallel()
	roster := sumWorld(t)
	if roster.FamiliarGone(cid("pc")) {
		t.Fatalf("FamiliarGone before anything happened")
	}
	gone := sumApplied(t, sumChange(t, roster, combat.ReportZeroHitPoints{Creature: cid("fam")}, "")).Roster()
	if !gone.FamiliarGone(cid("pc")) || gone.FamiliarGone(cid("other")) {
		t.Errorf("FamiliarGone = %v, want true for the summoner only [FR-049]", gone.FamiliarGone(cid("pc")))
	}
	a := sumDo(t, gone, "pc", combat.FindFamiliarRequest{Creature: cid("fam2"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFiend, Origin: combat.FamiliarOriginFindFamiliarSpell}, sumCombat("", sumInit(5)))
	if a.Roster().FamiliarGone(cid("pc")) || len(a.Creatures()) != 1 || a.Creatures()[0].StatBlock().HitPointMaximum() != 13 || a.Creatures()[0].Creature().ID != cid("fam2") {
		t.Errorf("recast = %v, want a fresh familiar at its maximum and the mark cleared [FR-049]", a.Creatures())
	}
	if got := sumKinds(a.Events()); !slices.Equal(got, []string{"summoned_creature_joined"}) {
		t.Errorf("events = %v, want a new creature, not a form change [FR-049]", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-050, FR-051, FR-052, FR-055, FR-006 - a temporary dismissal moves a present familiar to the pocket dimension (it drops what it carried, a SummonedRosterLeave of RemovalCauseDeparted, actionCost magic) and is refused for one already there; recalling needs a familiar in the pocket dimension and a host-reported space within 30 feet and joins it at the RosterAtInitiative it had; dismissing forever ends the link from either place and allows a new familiar (SRD 5.2.1, "Find Familiar", p. 130; "Magic [Action]", p. 185).
// Path: per case.
func TestDismissingAndRecallingAFamiliar(t *testing.T) {
	t.Parallel()
	inCombat := sumDo(t, combat.SummonedRoster{}, "pc", combat.FindFamiliarRequest{Creature: cid("fam"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}, sumCombat("", sumInit(14))).Roster()
	temp := combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}
	forever := combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalForever}
	a := sumApplied(t, sumChange(t, inCombat, temp, ""))
	ev, ok := a.Events()[0].(combat.SummonedCreatureDeparted)
	pocketEv, isPocket := a.Events()[0].(combat.FamiliarDismissedTemporarily)
	if ok || !isPocket || len(a.Events()) != 1 || !pocketEv.DropsCarriedItems() || pocketEv.ActionCost() != combat.StandardActionMagic || pocketEv.PendingRemoval() ||
		!slices.Equal(pocketEv.Combatants(), []combat.CombatantID{"fam", "pc"}) {
		t.Fatalf("events = %#v (%v), want FamiliarDismissedTemporarily dropping items at the cost of the Magic action [FR-050, FR-055]", a.Events(), ev)
	}
	if got, want := sumLeaves(a), []combat.SummonedRosterLeave{{Creature: cid("fam"), Cause: combat.RemovalCauseDeparted}}; !reflect.DeepEqual(got, want) {
		t.Errorf("leaves = %v, want %v [FR-006, FR-050]", got, want)
	}
	pocket := a.Roster()
	if l, has := pocket.Link("fam"); !has || !l.InPocketDimension() || pocket.FamiliarGone(cid("pc")) {
		t.Errorf("link = %+v, want the familiar kept in the pocket dimension, not gone [FR-050]", l)
	}
	if rej := sumRejection(t, sumChange(t, pocket, temp, "")); rej != (combat.FamiliarAlreadyDismissedRejection{}) {
		t.Errorf("a second dismissal = %#v, want FamiliarAlreadyDismissedRejection [FR-050]", rej)
	}
	active := sumApplied(t, sumChange(t, inCombat, temp, "fam"))
	if e := active.Events()[0].(combat.FamiliarDismissedTemporarily); !e.PendingRemoval() || !sumLeaves(active)[0].AfterTurn {
		t.Errorf("a dismissal of the active familiar is not pending [FR-077]")
	}
	// Recall.
	recall := func(space bool) combat.RecallFamiliar {
		return combat.RecallFamiliar{Summoner: cid("pc"), SpaceAvailable: space}
	}
	if rej := sumRejection(t, sumChange(t, pocket, recall(false), "")); rej != (combat.SpaceUnavailableRejection{}) {
		t.Errorf("a recall with no space = %#v, want SpaceUnavailableRejection [FR-051]", rej)
	}
	if rej := sumRejection(t, sumChange(t, inCombat, recall(true), "")); rej != (combat.FamiliarNotDismissedRejection{}) {
		t.Errorf("a recall of a present familiar = %#v, want FamiliarNotDismissedRejection [FR-051]", rej)
	}
	if rej := sumRejection(t, sumChange(t, pocket, combat.RecallFamiliar{Summoner: cid("other"), SpaceAvailable: true}, "")); rej != (combat.FamiliarNotDismissedRejection{}) {
		t.Errorf("a recall by a summoner with no familiar = %#v, want FamiliarNotDismissedRejection [FR-051]", rej)
	}
	if rej := sumRejection(t, sumChange(t, pocket, combat.RecallFamiliar{Summoner: cid("pc")}, "")); rej != (combat.SpaceUnavailableRejection{}) {
		t.Errorf("a recall with no space reported = %#v, want SpaceUnavailableRejection", rej)
	}
	back := sumApplied(t, sumChange(t, pocket, combat.RecallFamiliar{Summoner: cid("pc"), SpaceAvailable: true, InCombat: true, Initiative: sumInit(2)}, ""))
	if re, ok := back.Events()[0].(combat.FamiliarReappeared); !ok || re.ActionCost() != combat.StandardActionMagic || !slices.Equal(re.Combatants(), []combat.CombatantID{"fam", "pc"}) {
		t.Errorf("events = %#v, want FamiliarReappeared at the cost of the Magic action [FR-051, FR-055]", back.Events())
	}
	if want := []combat.SummonedRosterAction{combat.SummonedRosterJoin{Creature: cid("fam"), Position: sumInit(14), Reason: combat.JoinReasonSummoned}}; !reflect.DeepEqual(back.RosterActions(), want) {
		t.Errorf("recall actions = %v, want a join at the RosterAtInitiative it had (not the one given) [FR-006]", back.RosterActions())
	}
	if l, _ := back.Roster().Link("fam"); l.InPocketDimension() {
		t.Errorf("a recalled familiar is still in the pocket dimension")
	}
	// A familiar made out of combat has no retained position: the host gives one in combat and none outside it.
	outside := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
	outPocket := sumApplied(t, sumChange(t, outside, temp, "")).Roster()
	if got := sumApplied(t, sumChange(t, outPocket, recall(true), "")).RosterActions(); len(got) != 0 {
		t.Errorf("a recall out of combat joined %v, want none [FR-006]", got)
	}
	fresh := sumApplied(t, sumChange(t, outPocket, combat.RecallFamiliar{Summoner: cid("pc"), SpaceAvailable: true, InCombat: true, Initiative: sumInit(2)}, ""))
	if want := []combat.SummonedRosterAction{combat.SummonedRosterJoin{Creature: cid("fam"), Position: sumInit(2), Reason: combat.JoinReasonSummoned}}; !reflect.DeepEqual(fresh.RosterActions(), want) {
		t.Errorf("recall actions = %v, want a join at the position the host gave", fresh.RosterActions())
	}
	if l, _ := fresh.Roster().Link("fam"); func() bool { p, has := l.RetainedPosition(); return !has || p != sumInit(2) }() {
		t.Errorf("the position given was not retained")
	}
	// Forever.
	for _, from := range []combat.SummonedRoster{inCombat, pocket} {
		f := sumApplied(t, sumChange(t, from, forever, ""))
		d := sumDeparted(f)
		if len(d) != 1 || d[0].Cause() != combat.SummonEndDismissedForever || !d[0].DropsCarriedItems() || f.Roster().Len() != 0 {
			t.Errorf("departures = %#v, want the link ended with DismissedForever [FR-052]", d)
		}
	}
	if got := sumLeaves(sumApplied(t, sumChange(t, inCombat, forever, ""))); !reflect.DeepEqual(got, []combat.SummonedRosterLeave{{Creature: cid("fam"), Cause: combat.RemovalCauseDeparted}}) {
		t.Errorf("leaves = %v, want RemovalCauseDeparted [FR-006]", got)
	}
	if got := sumLeaves(sumApplied(t, sumChange(t, pocket, forever, ""))); len(got) != 0 {
		t.Errorf("a familiar in the pocket dimension left the encounter again: %v", got)
	}
	gone := sumApplied(t, sumChange(t, inCombat, forever, "")).Roster()
	if _, ok := sumTry(t, gone, "pc", combat.FindFamiliarRequest{Creature: cid("fam2"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}, sumScene()).(combat.SummonedApplied); !ok {
		t.Errorf("a new familiar cannot be cast after dismissing forever [FR-052]")
	}
	for _, c := range []combat.SummonedChange{temp, forever} {
		if _, ok := sumChange(t, combat.SummonedRoster{}, c, "").(combat.SummonedUnchanged); !ok {
			t.Errorf("dismissing a familiar that is not there changed the roster [FR-026]")
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-053, FR-044, Table P (familiar recast, other origin; recast to Wild Companion) - finishing a Long Rest departs a Wild Companion familiar (cause LongRestFinished, RemovalCauseDeparted), present or in the pocket dimension; a familiar of another origin stays, and the origin is that of the latest cast, so a Wild Companion familiar recast by the spell stays at the next Long Rest and the reverse departs (SRD 5.2.1, "Level 2: Wild Companion", p. 43).
// Path: per case.
func TestAWildCompanionFamiliarLeavesAtALongRest(t *testing.T) {
	t.Parallel()
	monsters := sumMonsters(t)
	spell, pact, wild := combat.FamiliarOriginFindFamiliarSpell, combat.FamiliarOriginPactOfTheChain, combat.FamiliarOriginWildCompanion
	cases := []struct {
		name, requirement, path string
		build                   func() combat.SummonedRoster
		departs                 bool
	}{
		{"a Wild Companion familiar", "FR-053, p. 43", "happy", func() combat.SummonedRoster { return sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", wild) }, true},
		{"a Wild Companion familiar in the pocket dimension", "FR-053", "unhappy (state error)", func() combat.SummonedRoster {
			return sumApplied(t, sumChange(t, sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", wild), combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
		}, true},
		{"a spell familiar", "FR-053", "unhappy (rule interaction)", func() combat.SummonedRoster { return sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", spell) }, false},
		{"a Pact of the Chain familiar", "FR-053", "unhappy (rule interaction)", func() combat.SummonedRoster { return sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", pact) }, false},
		{"Wild Companion recast by the spell", "Table P, Q-15", "unhappy (rule interaction)", func() combat.SummonedRoster {
			r := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", wild)
			return sumApplied(t, sumFamiliarCast(t, r, "x", normalForm("cat"), content.CreatureTypeFiend, spell, monsters)).Roster()
		}, false},
		{"a spell familiar recast by Wild Companion", "Table P, Q-15", "unhappy (rule interaction)", func() combat.SummonedRoster {
			r := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", spell)
			return sumApplied(t, sumFamiliarCast(t, r, "x", normalForm("cat"), content.CreatureTypeFiend, wild, monsters)).Roster()
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			roster := c.build()
			got := sumChange(t, roster, combat.ReportLongRestFinished{Summoner: cid("pc")}, "")
			if !c.departs {
				if _, ok := got.(combat.SummonedUnchanged); !ok {
					t.Errorf("transition = %#v, want SummonedUnchanged [%s]", got, c.requirement)
				}
				return
			}
			a := sumApplied(t, got)
			d := sumDeparted(a)
			if len(d) != 1 || d[0].Cause() != combat.SummonEndLongRestFinished || a.Roster().Len() != 0 {
				t.Errorf("departures = %#v, want the familiar gone with LongRestFinished [%s]", d, c.requirement)
			}
			if l, _ := roster.Link("fam"); !l.InPocketDimension() {
				if want := []combat.SummonedRosterLeave{{Creature: cid("fam"), Cause: combat.RemovalCauseDeparted}}; !reflect.DeepEqual(sumLeaves(a), want) {
					t.Errorf("leaves = %v, want %v [FR-006, FR-053]", sumLeaves(a), want)
				}
			} else if len(sumLeaves(a)) != 0 {
				t.Errorf("a familiar in the pocket dimension left the encounter")
			}
		})
	}
	if _, ok := sumChange(t, sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", wild), combat.ReportLongRestFinished{Summoner: cid("other")}, "").(combat.SummonedUnchanged); !ok {
		t.Errorf("another summoner's Long Rest departed the familiar")
	}
	for _, o := range []combat.FamiliarOrigin{spell, pact, wild, 0} {
		if (o.String() == "unknown") == o.Valid() {
			t.Errorf("FamiliarOrigin %d String %q Valid %v disagree", int(o), o.String(), o.Valid())
		}
	}
}
