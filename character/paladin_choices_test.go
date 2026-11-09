package character_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// pldCantrip is a synthetic spell of the level with the class list.
func pldCantrip(t *testing.T, id string, level int, classes ...string) content.Spell {
	t.Helper()
	return clcSpell(t, id, level, classes, "")
}

func pldGuidance(t *testing.T) content.Spell {
	t.Helper()
	return pldCantrip(t, "guidance", 0, "cleric", "druid")
}

func pldFlame(t *testing.T) content.Spell {
	t.Helper()
	return pldCantrip(t, "sacred_flame", 0, "cleric")
}

func pldMending(t *testing.T) content.Spell {
	t.Helper()
	return pldCantrip(t, "mending", 0, "cleric", "wizard")
}

// Requirement: doc/specs/class-paladin.md FR-038 (SRD 5.2.1, "Level 2: Fighting Style", p. 54) - the Fighting Style feat is accepted at level 2 or higher and once, and a feat of another category,
// a repeat and level 1 are rejected with not_a_fighting_style_feat, feat_not_repeatable, already_chosen and level_too_low, all reported at once in declaration order.
// Path: per case (unhappy: state error, boundary).
func TestPaladinChoiceRulesStyle(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	defense := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, FightingStyle: character.PaladinFightingStyleFeat{Feat: "defense"}})
	for _, c := range []struct {
		name  string
		level int
		state character.PaladinState
		feats character.TakenFeats
		feat  content.Feat
		want  []string
	}{
		{"level 2", 2, pldFull(t, 2), character.TakenFeats{}, fgtStyle("archery"), nil},
		{"level 1", 1, pldFull(t, 1), character.TakenFeats{}, fgtStyle("archery"), []string{"level_too_low"}},
		{"already chosen", 5, defense, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("archery"), []string{"already_chosen"}},
		{"another category", 2, pldFull(t, 2), character.TakenFeats{}, alertFeat(), []string{"not_a_fighting_style_feat"}},
		{"the chosen feat held again", 5, defense, fgtFeats(t, content.FeatFightingStyle, "defense"), fgtStyle("defense"), []string{"already_chosen"}},
		{"a held style feat", 2, pldFull(t, 2), fgtFeats(t, content.FeatFightingStyle, "archery"), fgtStyle("archery"), []string{"feat_not_repeatable"}},
		{"level 1 and another category", 1, pldFull(t, 1), character.TakenFeats{}, alertFeat(), []string{"level_too_low", "not_a_fighting_style_feat"}},
		{"already chosen and another category", 5, defense, fgtFeats(t, content.FeatOrigin, "alert"), alertFeat(), []string{"already_chosen", "not_a_fighting_style_feat"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			changed, reasons, err := character.PaladinChoiceRules{}.Style(pldPaladin(t, c.level), c.feats, p, character.MustPaladinLevel(c.level), c.state, c.feat)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(pldReasons(reasons), c.want) && (len(reasons) != 0 || len(c.want) != 0) {
				t.Errorf("reasons = %v, want %v", pldReasons(reasons), c.want)
			}
			if len(c.want) == 0 && changed.Feats().Len() != c.feats.Len()+1 {
				t.Errorf("the feat was not taken: %v", changed.Feats().All())
			}
		})
	}
	for name, call := range map[string]func() error{
		"zero character": func() error {
			_, _, err := character.PaladinChoiceRules{}.Style(character.Character{}, character.TakenFeats{}, p, character.MustPaladinLevel(2), pldFull(t, 2), fgtStyle("archery"))
			return err
		},
		"zero feat": func() error {
			_, _, err := character.PaladinChoiceRules{}.Style(pldPaladin(t, 2), character.TakenFeats{}, p, character.MustPaladinLevel(2), pldFull(t, 2), content.Feat{})
			return err
		},
	} {
		wantInvalid(t, call(), name)
	}
}

// Requirement: doc/specs/class-paladin.md FR-039, FR-040 (SRD 5.2.1, "Level 2: Fighting Style", p. 54) - Blessed Warrior takes exactly two distinct cantrips of spell level 0 with Cleric in their class
// list and stores them; fewer, more, a duplicate, a non-cantrip and a non-Cleric spell are rejected with wrong_count, duplicate, not_a_cantrip and not_on_cleric_list; level 1 and a second choice are rejected too.
// Path: per case (unhappy: invalid input, boundary, state error).
func TestPaladinChoiceRulesBlessedWarrior(t *testing.T) {
	t.Parallel()
	p := pldProgression(t)
	g, f, m := pldGuidance(t), pldFlame(t), pldMending(t)
	bless := pldCantrip(t, "bless", 1, "cleric")
	fireBolt := pldCantrip(t, "fire_bolt", 0, "wizard")
	chosen := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, FightingStyle: character.PaladinFightingStyleFeat{Feat: "defense"}})
	for _, c := range []struct {
		name  string
		level int
		state character.PaladinState
		in    []content.Spell
		want  []string
	}{
		{"two non-cantrips", 2, pldFull(t, 2), []content.Spell{pldCantrip(t, "bless", 1, "cleric"), pldCantrip(t, "sanctuary", 1, "cleric")}, []string{"not_a_cantrip"}},
		{"two non-Cleric spells", 2, pldFull(t, 2), []content.Spell{pldCantrip(t, "fire_bolt", 0, "wizard"), pldCantrip(t, "light", 0, "wizard")}, []string{"not_on_cleric_list"}},
		{"two Cleric cantrips", 2, pldFull(t, 2), []content.Spell{g, f}, nil},
		{"one", 2, pldFull(t, 2), []content.Spell{g}, []string{"wrong_count"}},
		{"three", 2, pldFull(t, 2), []content.Spell{g, f, m}, []string{"wrong_count"}},
		{"none", 2, pldFull(t, 2), nil, []string{"wrong_count"}},
		{"duplicate", 2, pldFull(t, 2), []content.Spell{g, g}, []string{"duplicate"}},
		{"not a cantrip", 2, pldFull(t, 2), []content.Spell{g, bless}, []string{"not_a_cantrip"}},
		{"not a Cleric spell", 2, pldFull(t, 2), []content.Spell{g, fireBolt}, []string{"not_on_cleric_list"}},
		{"a level 1 non-Cleric spell", 2, pldFull(t, 2), []content.Spell{pldCantrip(t, "x", 1, "wizard"), g}, []string{"not_a_cantrip", "not_on_cleric_list"}},
		{"level 1", 1, pldFull(t, 1), []content.Spell{g, f}, []string{"level_too_low"}},
		{"already chosen", 5, chosen, []content.Spell{g, f}, []string{"already_chosen"}},
		{"everything", 1, chosen, []content.Spell{g}, []string{"level_too_low", "already_chosen", "wrong_count"}},
	} {
		next, reasons := character.PaladinChoiceRules{}.BlessedWarrior(p, character.MustPaladinLevel(c.level), c.state, c.in)
		if !slices.Equal(pldReasons(reasons), c.want) && (len(reasons) != 0 || len(c.want) != 0) {
			t.Errorf("%s: reasons = %v, want %v", c.name, pldReasons(reasons), c.want)
		}
		style, ok := next.FightingStyle()
		if len(c.want) == 0 {
			if bw, isBW := style.(character.BlessedWarrior); !ok || !isBW || bw.Cantrips != [2]content.SpellID{c.in[0].ID(), c.in[1].ID()} {
				t.Errorf("%s: style = %v", c.name, style)
			}
		} else if !next.Equal(c.state) {
			t.Errorf("%s: a rejection changed the state", c.name)
		}
	}
}

// Requirement: doc/specs/class-paladin.md FR-041 (SRD 5.2.1, "Level 2: Fighting Style", p. 54) - a Blessed Warrior cantrip may be replaced by another Cleric cantrip; a replacement without Blessed Warrior is
// not_blessed_warrior, a cantrip that is not held is not_held, one already held (the other cantrip, or the same one) is duplicate, and the replacement also fails not_a_cantrip and not_on_cleric_list.
// Path: per case (unhappy: boundary, state error, invalid input).
func TestPaladinChoiceRulesReplace(t *testing.T) {
	t.Parallel()
	bw := pldState(t, 5, character.PaladinStateParts{LayOnHandsRemaining: 25, ChannelDivinityRemaining: 2, FightingStyle: character.BlessedWarrior{Cantrips: [2]content.SpellID{"guidance", "sacred_flame"}}})
	for _, c := range []struct {
		name  string
		state character.PaladinState
		old   content.SpellID
		repl  content.Spell
		want  []string
		now   [2]content.SpellID
	}{
		{"replace the first", bw, "guidance", pldMending(t), nil, [2]content.SpellID{"mending", "sacred_flame"}},
		{"replace the second", bw, "sacred_flame", pldMending(t), nil, [2]content.SpellID{"guidance", "mending"}},
		{"without Blessed Warrior", pldFull(t, 5), "guidance", pldMending(t), []string{"not_blessed_warrior"}, [2]content.SpellID{}},
		{"not held", bw, "mending", pldGuidance(t), []string{"not_held"}, [2]content.SpellID{}},
		{"the other cantrip", bw, "guidance", pldFlame(t), []string{"duplicate"}, [2]content.SpellID{}},
		{"the same cantrip", bw, "guidance", pldGuidance(t), []string{"duplicate"}, [2]content.SpellID{}},
		{"not a cantrip", bw, "guidance", pldCantrip(t, "bless", 1, "cleric"), []string{"not_a_cantrip"}, [2]content.SpellID{}},
		{"not a Cleric cantrip", bw, "guidance", pldCantrip(t, "fire_bolt", 0, "wizard"), []string{"not_on_cleric_list"}, [2]content.SpellID{}},
	} {
		next, reasons := character.PaladinChoiceRules{}.Replace(c.state, c.old, c.repl)
		if !slices.Equal(pldReasons(reasons), c.want) && (len(reasons) != 0 || len(c.want) != 0) {
			t.Errorf("%s: reasons = %v, want %v", c.name, pldReasons(reasons), c.want)
			continue
		}
		if len(c.want) == 0 {
			if style, _ := next.FightingStyle(); style != (character.BlessedWarrior{Cantrips: c.now}) {
				t.Errorf("%s: style = %v, want %v", c.name, style, c.now)
			}
		} else if !next.Equal(c.state) {
			t.Errorf("%s: a rejection changed the state", c.name)
		}
	}
}
