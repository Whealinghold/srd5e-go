package combat_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-003, FR-035, FR-067 - NewSummonCreatureCommand returns an error matching srderr.ErrInvalidInput, and never panics, for a blank summoner, a side or request that is not defined, a cast level outside 1 to 9, a blank id, a creature that is its own summoner, an id used twice, an enum that is not defined, the zero facts for a spell that reads them, and a scene whose Initiatives are present when the summoner is not in combat, absent or of another number than the creatures that roll their own when it is, or hold a tie placement that is not defined.
// Path: per case.
func TestSummonCreatureCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	scene := sumScene()
	good := combat.GiantInsectRequest{Creature: cid("c"), Level: 4, Form: combat.InsectWasp}
	bones := func(ids ...string) combat.AnimateDeadRequest {
		q := combat.AnimateDeadRequest{Level: 5}
		for i, id := range ids {
			q.Targets = append(q.Targets, combat.UndeadTarget{Creature: combat.CombatantID(id), Remains: combat.RemainsID(sumLabel("r", i)), Kind: combat.RemainsBones, Size: content.SizeMedium})
		}
		return q
	}
	cases := []struct {
		name     string
		summoner string
		side     combat.Side
		request  combat.SummonRequest
		facts    combat.SummonerSpellFacts
		scene    combat.SummonScene
	}{
		{"a blank summoner", "", combat.SideA, good, sumFacts(t), scene},
		{"a side that is not defined", "pc", 0, good, sumFacts(t), scene},
		{"no request", "pc", combat.SideA, nil, sumFacts(t), scene},
		{"level 0", "pc", combat.SideA, combat.GiantInsectRequest{Creature: cid("c"), Level: 0, Form: combat.InsectWasp}, sumFacts(t), scene},
		{"level 10", "pc", combat.SideA, combat.GiantInsectRequest{Creature: cid("c"), Level: 10, Form: combat.InsectWasp}, sumFacts(t), scene},
		{"a negative level", "pc", combat.SideA, combat.GiantInsectRequest{Creature: cid("c"), Level: -4, Form: combat.InsectWasp}, sumFacts(t), scene},
		{"a blank creature id", "pc", combat.SideA, combat.GiantInsectRequest{Level: 4, Form: combat.InsectWasp}, sumFacts(t), scene},
		{"a creature that is its own summoner", "pc", combat.SideA, combat.GiantInsectRequest{Creature: cid("pc"), Level: 4, Form: combat.InsectWasp}, sumFacts(t), scene},
		{"an id used twice", "pc", combat.SideA, bones("u", "u"), combat.SummonerSpellFacts{}, scene},
		{"an insect form that is not defined", "pc", combat.SideA, combat.GiantInsectRequest{Creature: cid("c"), Level: 4}, sumFacts(t), scene},
		{"zero facts for an insect", "pc", combat.SideA, good, combat.SummonerSpellFacts{}, scene},
		{"zero facts for objects", "pc", combat.SideA, sumObjects("o"), combat.SummonerSpellFacts{}, scene},
		{"zero facts for a spirit", "pc", combat.SideA, combat.SummonDragonRequest{Creature: cid("c"), Level: 5, SharedResistance: content.DamageCold, BreathDamageType: content.DamageCold}, combat.SummonerSpellFacts{}, scene},
		{"zero facts for a steed", "pc", combat.SideA, sumSteedRequest("c", 2), combat.SummonerSpellFacts{}, scene},
		{"a damage type that is not defined", "pc", combat.SideA, combat.SummonDragonRequest{Creature: cid("c"), Level: 5, BreathDamageType: content.DamageCold}, sumFacts(t), scene},
		{"an object size that is not defined", "pc", combat.SideA, combat.AnimateObjectsRequest{Level: 5, Objects: []combat.ObjectTarget{{Creature: cid("o")}}}, sumFacts(t), scene},
		{"an object with a blank id", "pc", combat.SideA, combat.AnimateObjectsRequest{Level: 5, Objects: []combat.ObjectTarget{{Size: content.SizeSmall}}}, sumFacts(t), scene},
		{"a steed type that is not defined", "pc", combat.SideA, combat.FindSteedRequest{Level: 2, Choice: combat.SteedChoiceDifferent{Creature: cid("c")}, Anatomy: combat.MountAnatomyAppropriate, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleFitted}, sumFacts(t), scene},
		{"a steed with no choice", "pc", combat.SideA, combat.FindSteedRequest{Level: 2, Type: combat.SteedFey, Anatomy: combat.MountAnatomyAppropriate, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleFitted}, sumFacts(t), scene},
		{"a steed with a blank id", "pc", combat.SideA, combat.FindSteedRequest{Level: 2, Type: combat.SteedFey, Choice: combat.SteedChoiceSame{}, Anatomy: combat.MountAnatomyAppropriate, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleFitted}, sumFacts(t), scene},
		{"a steed with no anatomy", "pc", combat.SideA, combat.FindSteedRequest{Level: 2, Type: combat.SteedFey, Choice: combat.SteedChoiceSame{Creature: cid("c")}, Habitat: combat.MountHabitatOther, Saddle: combat.ExoticSaddleFitted}, sumFacts(t), scene},
		{"bones of a kind that is not defined", "pc", combat.SideA, combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{{Creature: cid("u"), Remains: "r", Size: content.SizeSmall}}}, sumFacts(t), scene},
		{"bones with a blank pile", "pc", combat.SideA, combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{{Creature: cid("u"), Kind: combat.RemainsBones, Size: content.SizeSmall}}}, sumFacts(t), scene},
		{"a corpse with no form", "pc", combat.SideA, combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{{Creature: cid("u"), Remains: "r", Size: content.SizeSmall}}}, sumFacts(t), scene},
		{"a simulacrum with a blank original", "pc", combat.SideA, combat.SimulacrumRequest{Creature: cid("s"), Level: 7}, sumFacts(t), scene},
		{"a simulacrum with a blank friend", "pc", combat.SideA, func() combat.SimulacrumRequest {
			q := sumSimulacrumRequest("s", 7)
			q.Friends = []combat.CombatantID{""}
			return q
		}(), sumFacts(t), scene},
		{"a simulacrum friend that is the summoner", "pc", combat.SideA, sumSimulacrumRequest("s", 7, "pc"), sumFacts(t), scene},
		{"a familiar of an origin that is not defined", "pc", combat.SideA, combat.FindFamiliarRequest{Creature: cid("f"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey}, sumFacts(t), scene},
		{"initiatives out of combat", "pc", combat.SideA, bones("u"), sumFacts(t), combat.SummonScene{Initiatives: []combat.RosterAtInitiative{sumInit(1)}}},
		{"no initiative in combat for an undead creature", "pc", combat.SideA, bones("u"), sumFacts(t), combat.SummonScene{InCombat: true}},
		{"too few initiatives in combat", "pc", combat.SideA, bones("u", "v"), sumFacts(t), combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{sumInit(1)}}},
		{"a surplus initiative in combat", "pc", combat.SideA, bones("u"), sumFacts(t), combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{sumInit(1), sumInit(2)}}},
		{"an initiative for a creature that shares its summoner's", "pc", combat.SideA, good, sumFacts(t), combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{sumInit(1)}}},
		{"a tie placement that is not defined", "pc", combat.SideA, bones("u"), sumFacts(t), combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{{Count: 3}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, combat.CombatantID(c.summoner), c.side, c.request, c.facts, c.scene)
			if !errors.Is(err, srderr.ErrInvalidInput) || !cmd.IsZero() {
				t.Errorf("command = %v, error = %v, want the zero command and srderr.ErrInvalidInput [FR-067]", cmd, err)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-003, FR-042, FR-067 - NewSummonCreatureCommand accepts the values at their bounds: levels 1 and 9, an empty target list (a refusal of the cast, not an error), the zero facts for a spell that does not read them, and the initiatives a familiar needs only when it has none yet.
// Path: per case.
func TestSummonCreatureCommandAcceptsTheBounds(t *testing.T) {
	t.Parallel()
	withFamiliar := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
	recast := combat.FindFamiliarRequest{Creature: cid("f2"), Level: 9, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}
	fresh := combat.FindFamiliarRequest{Creature: cid("f2"), Level: 1, Form: specialForm("imp"), Origin: combat.FamiliarOriginPactOfTheChain}
	cases := []struct {
		name    string
		roster  combat.SummonedRoster
		request combat.SummonRequest
		facts   combat.SummonerSpellFacts
		scene   combat.SummonScene
	}{
		{"an insect at level 9", combat.SummonedRoster{}, combat.GiantInsectRequest{Creature: cid("c"), Level: 9, Form: combat.InsectWasp}, sumFacts(t), sumScene()},
		{"an empty animate request", combat.SummonedRoster{}, combat.AnimateDeadRequest{Level: 3}, combat.SummonerSpellFacts{}, sumScene()},
		{"an empty objects request", combat.SummonedRoster{}, combat.AnimateObjectsRequest{Level: 5}, sumFacts(t), sumScene()},
		{"an empty reassert request", combat.SummonedRoster{}, combat.ReassertControlRequest{Kind: combat.SummonAnimatedUndead, Level: 3}, combat.SummonerSpellFacts{}, sumScene()},
		{"an undead cast with no facts", combat.SummonedRoster{}, combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u", "r")}}, combat.SummonerSpellFacts{}, sumScene()},
		{"a simulacrum with no facts", combat.SummonedRoster{}, sumSimulacrumRequest("s", 7), combat.SummonerSpellFacts{}, sumScene()},
		{"a recast needs no initiative in combat", withFamiliar, recast, combat.SummonerSpellFacts{}, combat.SummonScene{InCombat: true}},
		{"a first familiar needs one in combat", combat.SummonedRoster{}, fresh, combat.SummonerSpellFacts{}, combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{sumInit(4)}}},
		{"initiatives with a tie placement before", combat.SummonedRoster{}, fresh, combat.SummonerSpellFacts{}, combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{{Count: -3, Tie: combat.TiePlacementBeforeTied}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := combat.NewSummonCreatureCommand(c.roster, "pc", combat.SideA, c.request, c.facts, c.scene)
			if err != nil || cmd.IsZero() {
				t.Errorf("command = %v, error = %v, want a command [FR-067]", cmd, err)
			}
		})
	}
	scene := combat.SummonScene{InCombat: true, Initiatives: []combat.RosterAtInitiative{sumInit(7)}}
	in := scene.Initiatives
	cmd, err := combat.NewSummonCreatureCommand(combat.SummonedRoster{}, "pc", combat.SideA, fresh, combat.SummonerSpellFacts{}, scene)
	if err != nil {
		t.Fatal(err)
	}
	in[0] = sumInit(99)
	a := sumApplied(t, (combat.SummonedRoster{}).Summon(cmd, sumMonsters(t)...))
	if got := a.RosterActions()[0].(combat.SummonedRosterJoin).Position; got != sumInit(7) {
		t.Errorf("position = %v, want the initiative given when the command was built, not a later change [FR-065]", got)
	}
	if got := (combat.SummonedRoster{}).Summon(combat.SummonCreatureCommand{}); got != (combat.SummonedUnchanged{}) {
		t.Errorf("a zero command = %#v, want SummonedUnchanged", got)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-070 - a cast whose stat block the host did not look up leaves the roster unchanged (SummonedUnchanged) rather than guess: Animate Dead without the Skeleton or the Zombie, Create Undead without a form, Find Familiar without its form.
// Path: per case.
func TestACastWithoutItsStatBlocksChangesNothing(t *testing.T) {
	t.Parallel()
	roster := combat.SummonedRoster{}
	for _, c := range []struct {
		name    string
		request combat.SummonRequest
	}{
		{"no Skeleton", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u", "r")}}},
		{"no Zombie", combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumCorpse("u", "r", content.SizeSmall, true)}}},
		{"no Ghoul", combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("u", "r", combat.UndeadGhoul)}}},
		{"no familiar form", combat.FindFamiliarRequest{Creature: cid("f"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}},
	} {
		if got := roster.Summon(sumCmd(t, roster, "pc", c.request, sumScene())); got != (combat.SummonedUnchanged{}) {
			t.Errorf("%s: transition = %#v, want SummonedUnchanged [FR-070]", c.name, got)
		}
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-068, Table R - when several rejections of a cast hold, the reported one is the first of Table R that the cast can have together: the id before the cycle, the cycle before the level, the level before an object or target, and the cap last.
// Path: per case.
func TestRejectionsOfACastFollowTableR(t *testing.T) {
	t.Parallel()
	roster := sumInsect(t, combat.SummonedRoster{}, "pc", "used")
	chain := sumInsect(t, roster, "used", "leaf")
	cases := []struct {
		name     string
		roster   combat.SummonedRoster
		summoner string
		request  combat.SummonRequest
		want     combat.SummonRejection
	}{
		{"id in use before the level", roster, "pc", combat.GiantInsectRequest{Creature: cid("used"), Level: 3, Form: combat.InsectWasp}, combat.SummonCreatureIDInUseRejection{ID: cid("used")}},
		{"the cycle before the level", chain, "leaf", combat.GiantInsectRequest{Creature: cid("pc"), Level: 3, Form: combat.InsectWasp}, combat.SummonLinkCycleRejection{Creature: cid("pc"), Summoner: cid("leaf")}},
		{"the level before an object", roster, "pc", combat.AnimateObjectsRequest{Level: 4, Objects: []combat.ObjectTarget{{Creature: cid("o"), Size: content.SizeGargantuan}}}, combat.LevelBelowSpellRejection{Minimum: 5}},
		{"an object before the weights", roster, "pc", combat.AnimateObjectsRequest{Level: 5, Objects: []combat.ObjectTarget{{Creature: cid("o"), Size: content.SizeGargantuan}, {Creature: cid("p"), Size: content.SizeHuge}}}, combat.ObjectNotEligibleRejection{Creature: cid("o")}},
		{"the level before the original", roster, "pc", func() combat.SimulacrumRequest {
			q := sumSimulacrumRequest("s", 6)
			q.Original.WithinTenFeet = false
			return q
		}(), combat.LevelBelowSpellRejection{Minimum: 7}},
		{"the level before the resistances", roster, "pc", combat.SummonDragonRequest{Creature: cid("d"), Level: 4, SharedResistance: content.DamageForce, BreathDamageType: content.DamageForce}, combat.LevelBelowSpellRejection{Minimum: 5}},
		{"the shared resistance before the breath", roster, "pc", combat.SummonDragonRequest{Creature: cid("d"), Level: 5, SharedResistance: content.DamageForce, BreathDamageType: content.DamageForce}, combat.ResistanceNotOfferedRejection{Field: combat.SharedResistanceField}},
		{"the form before the stat block's level", roster, "pc", combat.FindFamiliarRequest{Creature: cid("f"), Level: 1, Form: specialForm("imp"), Origin: combat.FamiliarOriginFindFamiliarSpell}, combat.FormNotEligibleRejection{}},
		{"the id of a new familiar before its form", roster, "pc", combat.FindFamiliarRequest{Creature: cid("used"), Level: 1, Form: specialForm("imp"), Origin: combat.FamiliarOriginFindFamiliarSpell}, combat.SummonCreatureIDInUseRejection{ID: cid("used")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := sumRejection(t, sumTry(t, c.roster, c.summoner, c.request, sumScene())); got != c.want {
				t.Errorf("rejection = %#v, want %#v [FR-068]", got, c.want)
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md section 3 - the closed enums of the vocabulary have a String, a Valid and the values the spec names, the zero value being invalid.
// Path: per case.
func TestVocabularyEnums(t *testing.T) {
	t.Parallel()
	type enum interface{ String() string }
	cases := []struct {
		name  string
		value enum
		want  string
	}{
		{"SummonAnimatedObject", combat.SummonAnimatedObject, "animated_object"}, {"SummonGiantInsect", combat.SummonGiantInsect, "giant_insect"},
		{"SummonDraconicSpirit", combat.SummonDraconicSpirit, "draconic_spirit"}, {"SummonOtherworldlySteed", combat.SummonOtherworldlySteed, "otherworldly_steed"},
		{"SummonAnimatedUndead", combat.SummonAnimatedUndead, "animated_undead"}, {"SummonCreatedUndead", combat.SummonCreatedUndead, "created_undead"},
		{"SummonFamiliar", combat.SummonFamiliar, "familiar"}, {"SummonSimulacrum", combat.SummonSimulacrum, "simulacrum"}, {"SummonKind(0)", combat.SummonKind(0), "unknown"},
		{"SharesSummonerCount", combat.SharesSummonerCount, "shares_summoner_count"}, {"OwnCount", combat.OwnCount, "own_count"}, {"PlacementRule(0)", combat.PlacementRule(0), "unknown"},
		{"SummonEndDroppedToZeroHitPoints", combat.SummonEndDroppedToZeroHitPoints, "dropped_to_zero_hit_points"}, {"SummonEndSpellEnded", combat.SummonEndSpellEnded, "spell_ended"},
		{"SummonEndSummonerDied", combat.SummonEndSummonerDied, "summoner_died"}, {"SummonEndLongRestFinished", combat.SummonEndLongRestFinished, "long_rest_finished"},
		{"SummonEndDismissedForever", combat.SummonEndDismissedForever, "dismissed_forever"}, {"SummonEndDismissedTemporarily", combat.SummonEndDismissedTemporarily, "dismissed_temporarily"},
		{"SummonEndReplaced", combat.SummonEndReplaced, "replaced"}, {"SummonEndControlLapsed", combat.SummonEndControlLapsed, "control_lapsed"}, {"SummonEndCause(0)", combat.SummonEndCause(0), "unknown"},
		{"FamiliarOriginFindFamiliarSpell", combat.FamiliarOriginFindFamiliarSpell, "find_familiar_spell"}, {"FamiliarOriginPactOfTheChain", combat.FamiliarOriginPactOfTheChain, "pact_of_the_chain"},
		{"FamiliarOriginWildCompanion", combat.FamiliarOriginWildCompanion, "wild_companion"},
		{"SteedCelestial", combat.SteedCelestial, "celestial"}, {"SteedFey", combat.SteedFey, "fey"}, {"SteedFiend", combat.SteedFiend, "fiend"}, {"SteedType(0)", combat.SteedType(0), "unknown"},
		{"InsectWasp", combat.InsectWasp, "wasp"}, {"InsectSpider", combat.InsectSpider, "spider"}, {"InsectCentipede", combat.InsectCentipede, "centipede"}, {"InsectForm(0)", combat.InsectForm(0), "unknown"},
		{"UndeadGhoul", combat.UndeadGhoul, "ghoul"}, {"UndeadGhast", combat.UndeadGhast, "ghast"}, {"UndeadWight", combat.UndeadWight, "wight"}, {"UndeadMummy", combat.UndeadMummy, "mummy"}, {"UndeadForm(0)", combat.UndeadForm(0), "unknown"},
		{"SharedResistanceField", combat.SharedResistanceField, "shared_resistance"}, {"BreathDamageTypeField", combat.BreathDamageTypeField, "breath_damage_type"}, {"ResistanceField(0)", combat.ResistanceField(0), "unknown"},
		{"SummonedAttackSlam", combat.SummonedAttackSlam, "slam"}, {"SummonedAttackHealingTouch", combat.SummonedAttackHealingTouch, "healing_touch"}, {"SummonedAttackKind(0)", combat.SummonedAttackKind(0), "unknown"},
		{"FamiliarAttackViaReaction", combat.FamiliarAttackViaReaction, "via_reaction"}, {"FamiliarAttackViaBonusActionCommand", combat.FamiliarAttackViaBonusActionCommand, "via_bonus_action_command"},
		{"TouchNotPresent", combat.TouchNotPresent, "not_present"}, {"TouchNotTouch", combat.TouchNotTouch, "not_touch"}, {"TouchTooFar", combat.TouchTooFar, "too_far"}, {"TouchFamiliarCannotReact", combat.TouchFamiliarCannotReact, "familiar_cannot_react"},
		{"SummonSpellEndConcentrationEnded", combat.SummonSpellEndConcentrationEnded, "concentration_ended"}, {"SummonSpellEndDurationElapsed", combat.SummonSpellEndDurationElapsed, "duration_elapsed"},
		{"SummonSpellEndDispelled", combat.SummonSpellEndDispelled, "dispelled"},
	}
	for _, c := range cases {
		if got := c.value.String(); got != c.want {
			t.Errorf("%s.String() = %q, want %q", c.name, got, c.want)
		}
	}
	for _, c := range []struct {
		name  string
		valid func() []bool
	}{
		{"SummonKind", func() []bool {
			return []bool{combat.SummonKind(0).Valid(), combat.SummonAnimatedObject.Valid(), combat.SummonSimulacrum.Valid(), combat.SummonKind(9).Valid()}
		}},
		{"PlacementRule", func() []bool {
			return []bool{combat.PlacementRule(0).Valid(), combat.SharesSummonerCount.Valid(), combat.OwnCount.Valid(), combat.PlacementRule(3).Valid()}
		}},
		{"SummonEndCause", func() []bool {
			return []bool{combat.SummonEndCause(0).Valid(), combat.SummonEndDroppedToZeroHitPoints.Valid(), combat.SummonEndControlLapsed.Valid(), combat.SummonEndCause(9).Valid()}
		}},
		{"SteedType", func() []bool {
			return []bool{combat.SteedType(0).Valid(), combat.SteedCelestial.Valid(), combat.SteedFiend.Valid(), combat.SteedType(4).Valid()}
		}},
		{"InsectForm", func() []bool {
			return []bool{combat.InsectForm(0).Valid(), combat.InsectWasp.Valid(), combat.InsectCentipede.Valid(), combat.InsectForm(4).Valid()}
		}},
		{"UndeadForm", func() []bool {
			return []bool{combat.UndeadForm(0).Valid(), combat.UndeadGhoul.Valid(), combat.UndeadMummy.Valid(), combat.UndeadForm(5).Valid()}
		}},
		{"RemainsKind", func() []bool {
			return []bool{combat.RemainsKind(0).Valid(), combat.RemainsBones.Valid(), combat.RemainsCorpse.Valid(), combat.RemainsKind(3).Valid()}
		}},
		{"Dismissal", func() []bool {
			return []bool{combat.Dismissal(0).Valid(), combat.DismissalTemporary.Valid(), combat.DismissalForever.Valid(), combat.Dismissal(3).Valid()}
		}},
		{"SummonedAttackKind", func() []bool {
			return []bool{combat.SummonedAttackKind(0).Valid(), combat.SummonedAttackSlam.Valid(), combat.SummonedAttackHealingTouch.Valid(), combat.SummonedAttackKind(11).Valid()}
		}},
	} {
		if got := c.valid(); len(got) != 4 || got[0] || !got[1] || !got[2] || got[3] {
			t.Errorf("%s.Valid at 0, first, last, past = %v, want false true true false", c.name, got)
		}
	}
	if got := combat.SummonKind(9).Spell(); got != "" {
		t.Errorf("Spell of an undefined kind = %q", got)
	}
	for kind, spell := range map[combat.SummonKind]content.SpellID{combat.SummonAnimatedObject: "animate_objects", combat.SummonGiantInsect: "giant_insect", combat.SummonDraconicSpirit: "summon_dragon",
		combat.SummonOtherworldlySteed: "find_steed", combat.SummonAnimatedUndead: "animate_dead", combat.SummonCreatedUndead: "create_undead", combat.SummonFamiliar: "find_familiar", combat.SummonSimulacrum: "simulacrum"} {
		if kind.Spell() != spell {
			t.Errorf("%v.Spell() = %q, want %q [section 3]", kind, kind.Spell(), spell)
		}
	}
	for _, k := range []combat.SummonKind{combat.SummonAnimatedUndead, combat.SummonCreatedUndead, combat.SummonFamiliar} {
		if k.Placement() != combat.OwnCount {
			t.Errorf("%v placement = %v, want OwnCount [FR-003]", k, k.Placement())
		}
	}
	if combat.SteedFey.CreatureType() != content.CreatureTypeFey || combat.SteedType(0).CreatureType() != 0 {
		t.Errorf("SteedType.CreatureType is wrong")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-067 - each Must form builds what its New form builds for valid values and panics with the error the New form returns (an ErrInvalidInput) for invalid ones.
// Path: per case.
func TestSummonedMustForms(t *testing.T) {
	t.Parallel()
	good := combat.GiantInsectRequest{Creature: cid("c"), Level: 4, Form: combat.InsectWasp}
	roster := combat.SummonedRoster{}
	panics := func(f func()) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err, _ = r.(error)
			}
		}()
		f()
		return nil
	}
	if f := combat.MustSummonerSpellFacts(7, 15, 4, 3); f != sumFacts(t) {
		t.Errorf("MustSummonerSpellFacts = %v, want the facts of NewSummonerSpellFacts [FR-067]", f)
	}
	if c := combat.MustSummonCreatureCommand(roster, cid("pc"), combat.SideA, good, sumFacts(t), sumScene()); c.IsZero() {
		t.Errorf("MustSummonCreatureCommand returned the zero command [FR-067]")
	}
	if c := combat.MustApplySummonedChangeCommand(roster, combat.ElapseControlTime{}, ""); c.IsZero() {
		t.Errorf("MustApplySummonedChangeCommand returned the zero command [FR-067]")
	}
	for name, f := range map[string]func(){
		"facts":   func() { combat.MustSummonerSpellFacts(0, 0, 0, 0) },
		"summon":  func() { combat.MustSummonCreatureCommand(roster, "", combat.SideA, good, sumFacts(t), sumScene()) },
		"change":  func() { combat.MustApplySummonedChangeCommand(roster, nil, "") },
		"elapsed": func() { combat.MustApplySummonedChangeCommand(roster, combat.ElapseControlTime{Elapsed: -1}, "") },
	} {
		if err := panics(f); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: panic value = %v, want an error matching ErrInvalidInput [FR-067]", name, err)
		}
	}
}
