package combat

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-006 - two links are Equal only when every field is the same; a change
// of any one field, up or down, makes them differ, in either order of comparison.
// Path: per case (one field at a time).
func TestSummonedLinkEqualComparesEveryField(t *testing.T) {
	t.Parallel()
	base := SummonedLink{
		creature: "m", summoner: "p", monster: "mon", kind: SummonFamiliar, level: 5, side: SideA, presence: presenceInEncounter,
		window: startedWindow(), undead: UndeadGhoul, order: GeneralOrder{Label: "guard"}, retained: RosterAfterCombatant{Anchor: "p"},
		familiar: FamiliarState{form: character.NormalFamiliarForm("cat"), typ: content.CreatureTypeFey, origin: FamiliarOriginPactOfTheChain, sight: true},
		friends:  []CombatantID{"f"},
	}
	if !base.Equal(base) {
		t.Fatal("a link is not Equal to itself")
	}
	edits := map[string]func(*SummonedLink){
		"creature": func(l *SummonedLink) { l.creature = "n" }, "summoner": func(l *SummonedLink) { l.summoner = "q" },
		"monster": func(l *SummonedLink) { l.monster = "moo" }, "kind": func(l *SummonedLink) { l.kind = SummonSimulacrum },
		"level": func(l *SummonedLink) { l.level = 6 }, "side": func(l *SummonedLink) { l.side = SideB },
		"presence": func(l *SummonedLink) { l.presence = presencePocket }, "window": func(l *SummonedLink) { l.window = ControlWindow{} },
		"undead": func(l *SummonedLink) { l.undead = UndeadGhast }, "order": func(l *SummonedLink) { l.order = GeneralOrder{Label: "stay"} },
		"retained": func(l *SummonedLink) { l.retained = RosterAfterCombatant{Anchor: "z"} },
		"familiar": func(l *SummonedLink) { l.familiar.sight = false }, "friends": func(l *SummonedLink) { l.friends = nil },
	}
	for name, edit := range edits {
		other := base
		edit(&other)
		if base.Equal(other) || other.Equal(base) {
			t.Errorf("a link that differs in %s is Equal", name)
		}
	}
}
