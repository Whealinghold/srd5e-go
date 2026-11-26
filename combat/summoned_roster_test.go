package combat_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-008, FR-065, section 5 - a link names its creature (a MonsterCombatant of its monster id) and its summoner, the roster lists the creatures of one summoner in creation order and copies what it returns, and the zero roster is the empty roster.
// Path: happy.
func TestTheRosterAndItsLinks(t *testing.T) {
	t.Parallel()
	world := sumWorld(t)
	l, ok := world.Link("o2")
	if !ok || !slices.Equal(l.Combatants(), []combat.CombatantID{"o2", "pc"}) || l.Creature() != (combat.CombatantRef{ID: cid("o2"), Kind: combat.MonsterCombatant{ID: "animated_object"}}) {
		t.Errorf("link = %+v, want o2 of pc as an animated object", l)
	}
	if _, ok := world.Link("ghost"); ok {
		t.Errorf("a link for a creature that is not there")
	}
	var ids []string
	for _, l := range world.LinksOf("pc") {
		ids = append(ids, string(l.Combatants()[0]))
	}
	if !slices.Equal(ids, []string{"o1", "o2", "i1", "d1", "u1", "u2", "sim", "st", "fam"}) || len(world.LinksOf("other")) != 0 {
		t.Errorf("LinksOf(pc) = %v, want the nine creatures in creation order", ids)
	}
	links := world.LinksOf("pc")
	links[0] = combat.SummonedLink{}
	if first := world.LinksOf("pc")[0]; first.SummonKind() == 0 {
		t.Errorf("LinksOf is not a copy [FR-065]")
	}
	empty := combat.SummonedRoster{}
	if empty.Len() != 0 || len(empty.Links()) != 0 || !empty.Equal(combat.SummonedRoster{}) || empty.Equal(world) || world.Equal(empty) {
		t.Errorf("the zero roster is not the empty roster")
	}
	if got := sumApplied(t, sumChange(t, world, combat.ReportZeroHitPoints{Creature: cid("i1")}, "")).Roster(); got.Equal(world) {
		t.Errorf("a roster without a creature equals the one with it")
	}
	pocket := sumApplied(t, sumChange(t, world, combat.DismissFamiliar{Summoner: cid("pc"), How: combat.DismissalTemporary}, "")).Roster()
	if pocket.Equal(world) {
		t.Errorf("a roster with the familiar in its pocket dimension equals the one with it present")
	}
	fam, _ := world.Link("fam")
	if _, ok := fam.UndeadForm(); ok {
		t.Errorf("a familiar has an undead form")
	}
	if _, ok := fam.ControlWindow(); ok {
		t.Errorf("a familiar has a control window")
	}
	if _, ok := fam.RetainedPosition(); ok {
		t.Errorf("a familiar made out of combat has a retained position")
	}
	if _, ok := fam.Familiar(); !ok {
		t.Errorf("a familiar link has no familiar state")
	}
	if _, ok := l.Familiar(); ok {
		t.Errorf("an object has a familiar state")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-040 - the lapse event reports the kind of the undead creature whose control ran out.
// Path: happy.
func TestTheLapseEventNamesTheKind(t *testing.T) {
	t.Parallel()
	g := sumGhouls(t)
	a := sumApplied(t, g.ElapseControl(24*time.Hour))
	if ev, ok := a.Events()[0].(combat.SummonControlLapsed); !ok || ev.SummonKind() != combat.SummonCreatedUndead {
		t.Errorf("events = %#v, want SummonControlLapsed of a created undead creature", a.Events())
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-007, FR-060, FR-067 - a stat block is built from the content stat block of a familiar with its Fly and Swim Speeds; the stat block has no Burrow Speed; a simulacrum original with a blank monster id, no size, no type or no Hit Points is invalid input.
// Path: per case.
func TestStatBlockSpeedsAndTheSimulacrumOriginal(t *testing.T) {
	t.Parallel()
	f := msFields(t)
	f.Sizes, f.Type, f.ArmorClass = []content.CreatureSize{content.SizeTiny}, content.CreatureTypeBeast, 11
	f.Speeds = []content.SpeedEntry{msSpeed(10, "", content.SpeedWalk), msSpeed(60, "", content.SpeedFly), msSpeed(20, "", content.SpeedSwim)}
	f.ChallengeRating = content.MustChallengeRating("0")
	bird := luMust(content.NewMonster("raven", "raven", content.MustSourceCitation("Fixture", content.MonsterSection, 258), f))
	roster := combat.SummonedRoster{}
	req := combat.FindFamiliarRequest{Creature: cid("f"), Level: 1, Form: normalForm("raven"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}
	a := sumApplied(t, roster.Summon(sumCmd(t, roster, "pc", req, sumScene()), bird))
	b := a.Creatures()[0].StatBlock()
	if b.Speed(content.SpeedWalk) != 10 || b.Speed(content.SpeedFly) != 60 || b.Speed(content.SpeedSwim) != 20 || b.Speed(content.SpeedBurrow) != 0 || b.Speed(0) != 0 {
		t.Errorf("Speeds = %d %d %d burrow %d, want 10, 60, 20 and no burrow [FR-043]", b.Speed(content.SpeedWalk), b.Speed(content.SpeedFly), b.Speed(content.SpeedSwim), b.Speed(content.SpeedBurrow))
	}
	good := combat.SimulacrumOriginal{Monster: "hero", Size: content.SizeMedium, Type: content.CreatureTypeHumanoid, HitPointMaximum: 10, WithinTenFeet: true}
	for name, o := range map[string]combat.SimulacrumOriginal{
		"no monster": {Size: good.Size, Type: good.Type, HitPointMaximum: 10}, "no size": {Monster: "hero", Type: good.Type, HitPointMaximum: 10},
		"no type": {Monster: "hero", Size: good.Size, HitPointMaximum: 10}, "no Hit Points": {Monster: "hero", Size: good.Size, Type: good.Type},
	} {
		if b, rej, err := combat.NewSimulacrumStatBlock(o); !errors.Is(err, srderr.ErrInvalidInput) || rej != nil || !b.IsZero() {
			t.Errorf("%s: = %v, %#v, %v, want srderr.ErrInvalidInput", name, b, rej, err)
		}
	}
	if _, rej, err := combat.NewSimulacrumStatBlock(good); rej != nil || err != nil {
		t.Errorf("a good original = %#v, %v", rej, err)
	}
}
