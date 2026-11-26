package combat_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The creatures of these tests are synthetic: made-up stat blocks that carry the numbers the spec's case tables quote, never a
// stat block of SRD 5.2.1.

// sumFacts are the summoner's numbers of the case tables: attack bonus 7, save DC 15, modifier 4, Proficiency Bonus 3.
func sumFacts(tb testing.TB) combat.SummonerSpellFacts {
	tb.Helper()
	return luMust(combat.NewSummonerSpellFacts(7, 15, 4, 3))
}

// sumMonster builds a synthetic monster of the id, type, size, Challenge Rating and XP.
func sumMonster(tb testing.TB, id string, typ content.CreatureType, size content.CreatureSize, cr string, xp int) content.Monster {
	tb.Helper()
	f := msFields(tb)
	f.Sizes, f.Type, f.ArmorClass = []content.CreatureSize{size}, typ, 12
	f.HitPoints = luMust(content.NewHitPointsEntry(13, dice.MustExpression(2, dice.D8(), 4)))
	f.Speeds = []content.SpeedEntry{msSpeed(30, "", content.SpeedWalk), msSpeed(20, "", content.SpeedClimb)}
	f.ChallengeRating, f.ExperiencePoints = content.MustChallengeRating(cr), xp
	return luMust(content.NewMonster(content.MonsterID(id), strings.ReplaceAll(id, "_", " "), content.MustSourceCitation("Fixture", content.MonsterSection, 258), f))
}

// sumMonsters are the stat blocks the casts of these tests name.
func sumMonsters(tb testing.TB) []content.Monster {
	tb.Helper()
	m := func(id string, typ content.CreatureType, size content.CreatureSize, cr string, xp int) content.Monster {
		return sumMonster(tb, id, typ, size, cr, xp)
	}
	return []content.Monster{
		m("skeleton", content.CreatureTypeUndead, content.SizeMedium, "1/4", 50), m("zombie", content.CreatureTypeUndead, content.SizeMedium, "1/4", 50),
		m("ghoul", content.CreatureTypeUndead, content.SizeMedium, "1", 200), m("ghast", content.CreatureTypeUndead, content.SizeMedium, "2", 450),
		m("wight", content.CreatureTypeUndead, content.SizeMedium, "3", 700), m("mummy", content.CreatureTypeUndead, content.SizeMedium, "3", 700),
		m("cat", content.CreatureTypeBeast, content.SizeTiny, "0", 10), m("fixture_beast", content.CreatureTypeBeast, content.SizeTiny, "0", 10),
		m("fixture_strong_beast", content.CreatureTypeBeast, content.SizeMedium, "1", 200), m("fixture_giant", content.CreatureTypeGiant, content.SizeTiny, "0", 10),
		m("imp", content.CreatureTypeFiend, content.SizeTiny, "1", 200), m("pseudodragon", content.CreatureTypeDragon, content.SizeTiny, "1/4", 50),
	}
}

// sumSource is a content source holding sumMonsters.
func sumSource(tb testing.TB, without ...string) *contenttest.FakeSource {
	tb.Helper()
	var entries []content.Entry
	for _, m := range sumMonsters(tb) {
		skip := false
		for _, w := range without {
			skip = skip || m.ID().String() == w
		}
		if !skip {
			entries = append(entries, m)
		}
	}
	return contenttest.NewFakeSource(entries...)
}

// sumInit is the position of a creature that rolls its own Initiative: count, placed after the tied ones.
func sumInit(count int) combat.RosterAtInitiative {
	return combat.RosterAtInitiative{Count: combat.InitiativeCount(count), Tie: combat.TiePlacementAfterTied}
}

// sumScene is a scene out of combat at night with space for a steed; sumCombat is the same in combat with the initiatives.
func sumScene() combat.SummonScene { return combat.SummonScene{IsNight: true, SpaceAvailable: true} }

func sumCombat(active string, inits ...combat.RosterAtInitiative) combat.SummonScene {
	s := combat.SummonScene{InCombat: true, IsNight: true, SpaceAvailable: true, Initiatives: inits}
	if active != "" {
		s.Active = cid(active)
	}
	return s
}

// sumCmd builds the command of the summoner's cast on side A with the standard facts.
func sumCmd(tb testing.TB, roster combat.SummonedRoster, summoner string, req combat.SummonRequest, scene combat.SummonScene) combat.SummonCreatureCommand {
	tb.Helper()
	cmd, err := combat.NewSummonCreatureCommand(roster, cid(summoner), combat.SideA, req, sumFacts(tb), scene)
	if err != nil {
		tb.Fatalf("NewSummonCreatureCommand(%T): %v", req, err)
	}
	return cmd
}

// sumTry makes the cast and returns the transition.
func sumTry(tb testing.TB, roster combat.SummonedRoster, summoner string, req combat.SummonRequest, scene combat.SummonScene) combat.SummonedTransition {
	tb.Helper()
	return roster.Summon(sumCmd(tb, roster, summoner, req, scene), sumMonsters(tb)...)
}

// sumDo makes the cast and returns the Applied transition, failing when it is anything else.
func sumDo(tb testing.TB, roster combat.SummonedRoster, summoner string, req combat.SummonRequest, scene combat.SummonScene) combat.SummonedApplied {
	tb.Helper()
	a, ok := sumTry(tb, roster, summoner, req, scene).(combat.SummonedApplied)
	if !ok {
		tb.Fatalf("Summon(%T) was not applied", req)
	}
	return a
}

// sumRejection returns the rejection of a cast that was refused, failing when it was not.
func sumRejection(tb testing.TB, t combat.SummonedTransition) combat.SummonRejection {
	tb.Helper()
	r, ok := t.(combat.SummonedRejected)
	if !ok {
		tb.Fatalf("transition = %#v, want SummonedRejected", t)
	}
	return r.Rejection
}

// sumInsect summons a Giant Insect named id for the summoner at level 4 and returns the new roster.
func sumInsect(tb testing.TB, roster combat.SummonedRoster, summoner, id string) combat.SummonedRoster {
	tb.Helper()
	return sumDo(tb, roster, summoner, combat.GiantInsectRequest{Creature: cid(id), Level: 4, Form: combat.InsectSpider}, sumScene()).Roster()
}

// sumChange applies the change and returns the transition.
func sumChange(tb testing.TB, roster combat.SummonedRoster, change combat.SummonedChange, active string) combat.SummonedTransition {
	tb.Helper()
	if _, err := combat.NewApplySummonedChangeCommand(roster, change, combat.CombatantID(active)); err != nil {
		tb.Fatalf("NewApplySummonedChangeCommand(%T): %v", change, err)
	}
	return roster.Apply(change, combat.CombatantID(active))
}

// sumApplied returns the Applied transition, failing when it is anything else.
func sumApplied(tb testing.TB, t combat.SummonedTransition) combat.SummonedApplied {
	tb.Helper()
	a, ok := t.(combat.SummonedApplied)
	if !ok {
		tb.Fatalf("transition = %#v, want SummonedApplied", t)
	}
	return a
}

// sumKinds returns the Kind of each event.
func sumKinds(events []combat.SummonEvent) []string {
	out := []string{}
	for _, e := range events {
		out = append(out, e.Kind())
	}
	return out
}

// sumFamiliar summons a familiar for the summoner with the origin and form, out of combat.
func sumFamiliar(tb testing.TB, roster combat.SummonedRoster, summoner, id string, origin combat.FamiliarOrigin) combat.SummonedRoster {
	tb.Helper()
	req := combat.FindFamiliarRequest{Creature: cid(id), Level: 1, Form: character.NormalFamiliarForm("cat"), Type: content.CreatureTypeFey, Origin: origin}
	return sumDo(tb, roster, summoner, req, sumScene()).Roster()
}

// sumLabel names the n-th creature of a cap test.
func sumLabel(prefix string, n int) string { return fmt.Sprintf("%s%02d", prefix, n) }

// normalForm and specialForm are the familiar forms of a Find Familiar cast and of Pact of the Chain.
func normalForm(id string) character.FamiliarForm {
	return character.NormalFamiliarForm(content.MonsterID(id))
}

func specialForm(id string) character.FamiliarForm {
	return character.SpecialFamiliarForm(content.MonsterID(id))
}

// sumResolutions returns the resolutions of the amounts of Force damage taken through the real pipeline by a creature with ample
// Hit Points, so each has the amount as the damage it dealt.
func sumResolutions(tb testing.TB, amounts ...int) []combat.DamageResolution {
	tb.Helper()
	e := encWith(tb, encPC(tb, "a", combat.SideA, 100000, 100000))
	var damages []combat.Damage
	for _, n := range amounts {
		damages = append(damages, combat.MustDamage(n, content.DamageForce))
	}
	tr, err := e.ApplyDamage(cid("a"), damages, nil, combat.OtherSource, combat.NormalHit)
	encOK(tb, tr, err)
	return tr.Resolutions()
}

// sumWorld is the roster of the summoner "pc" with two Animated Objects (o1, o2), a Giant Insect (i1), a Draconic Spirit (d1), two
// animated undead (u1, u2), a simulacrum (sim), an Otherworldly Steed (st) and a familiar (fam), made by real casts out of combat.
func sumWorld(tb testing.TB) combat.SummonedRoster {
	tb.Helper()
	r := sumDo(tb, combat.SummonedRoster{}, "pc", sumObjects("o1", "o2"), sumScene()).Roster()
	r = sumInsect(tb, r, "pc", "i1")
	r = sumDo(tb, r, "pc", combat.SummonDragonRequest{Creature: cid("d1"), Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageFire}, sumScene()).Roster()
	r = sumDo(tb, r, "pc", combat.AnimateDeadRequest{Level: 4, Targets: []combat.UndeadTarget{sumBones("u1", "b1"), sumBones("u2", "b2")}}, sumScene()).Roster()
	r = sumDo(tb, r, "pc", sumSimulacrumRequest("sim", 7, "ally"), sumScene()).Roster()
	r = sumDo(tb, r, "pc", sumSteedRequest("st", 2), sumScene()).Roster()
	return sumFamiliar(tb, r, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
}

// sumOrder is a general order, the same in every test that needs one.
func sumOrder() combat.SummonedOrder { return combat.GeneralOrder{Label: "guard_the_door"} }

// sumTargets are command targets at the distance.
func sumTargets(distance int, ids ...string) []combat.CommandTarget {
	var out []combat.CommandTarget
	for _, id := range ids {
		out = append(out, combat.CommandTarget{Creature: combat.CombatantID(id), DistanceFeet: distance})
	}
	return out
}
