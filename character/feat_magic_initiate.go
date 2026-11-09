package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// FeatGrantedSpells is the spells a Magic Initiate gives: the cantrips, the level
// 1 spell that is always prepared and does not count against any prepared-spell
// limit, and the ability for the feat's spells (SRD 5.2.1, "Magic Initiate", p.
// 87; "Always-Prepared Spells", p. 104). The spells belong to no class; casting
// them is the casting rules'. It is immutable and copied on read. The zero value
// grants nothing. Safe for concurrent use.
type FeatGrantedSpells struct {
	list     content.ClassID
	ability  ability.Ability
	cantrips []content.SpellID
	levelOne content.SpellID
}

func grantedSpellsOf(c MagicInitiateChoices) FeatGrantedSpells {
	return FeatGrantedSpells{list: c.list, ability: c.ability, cantrips: slices.Clone(c.cantrips), levelOne: c.levelOne}
}

// List returns the spell list the spells were chosen from.
func (g FeatGrantedSpells) List() content.ClassID { return g.list }

// Ability returns the ability for the feat's spells.
func (g FeatGrantedSpells) Ability() ability.Ability { return g.ability }

// Cantrips returns a new slice of the cantrips.
func (g FeatGrantedSpells) Cantrips() []content.SpellID { return slices.Clone(g.cantrips) }

// LevelOneSpell returns the level 1 spell.
func (g FeatGrantedSpells) LevelOneSpell() content.SpellID { return g.levelOne }

// LevelOneAlwaysPrepared reports whether the level 1 spell is always prepared:
// true.
func (FeatGrantedSpells) LevelOneAlwaysPrepared() bool { return true }

// LevelOneCountsAgainstPreparedLimit reports whether the level 1 spell counts
// against a prepared-spell limit: false.
func (FeatGrantedSpells) LevelOneCountsAgainstPreparedLimit() bool { return false }

// IsZero reports whether nothing is granted.
func (g FeatGrantedSpells) IsZero() bool { return g.list == "" }

// Equal reports whether g and o grant the same spells.
func (g FeatGrantedSpells) Equal(o FeatGrantedSpells) bool {
	return g.list == o.list && g.ability == o.ability && g.levelOne == o.levelOne && slices.Equal(g.cantrips, o.cantrips)
}

// GrantedSpells returns the spells of every held Magic Initiate, in the order
// taken.
func (f TakenFeats) GrantedSpells() []FeatGrantedSpells {
	var out []FeatGrantedSpells
	for _, t := range f.taken {
		if mi, ok := t.choices.(MagicInitiateChoices); ok {
			out = append(out, grantedSpellsOf(mi))
		}
	}
	return out
}

// MagicInitiateSpellChange is the Spell Change of Magic Initiate: on gaining a
// level, replace one of the feat's spells with another of the same level from the
// chosen list (SRD 5.2.1, "Magic Initiate", p. 87). The zero value is ready to
// use. It is stateless and safe for concurrent use.
type MagicInitiateSpellChange struct{}

// Replace replaces the spell old of the Magic Initiate whose spell list is list
// with repl. The checks run in this order and the first failure is reported:
// FeatNotHeld (no Magic Initiate with that list), SpellChangeNotAllowedNow (gained
// is not a LevelGained of this character at its current total level; the zero
// event is none), SpellChangeAlreadyUsed (once per instance and total level),
// then FeatChoicesInvalid with every problem: replacementSpell missing (old is not
// one of the feat's spells), levelMismatch, notOnSpellList, duplicate. The list,
// the ability and the state of the free cast are unchanged. Success is
// FeatsChanged with one FeatSpellReplaced.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, list
// or Spell. The inputs are unchanged. It never panics.
func (MagicInitiateSpellChange) Replace(c Character, feats TakenFeats, list content.ClassID, old content.SpellID, repl content.Spell, gained LevelGained) (FeatsTransition, error) {
	if c.IsZero() || !list.Valid() || !old.Valid() || !repl.ID().Valid() {
		return nil, invalidInput("magic initiate spell change of a zero character, list or spell")
	}
	return replaceMagicInitiateSpell(c, feats, list, old, repl, gained), nil
}

// replaceMagicInitiateSpell is Replace for arguments already checked.
func replaceMagicInitiateSpell(c Character, feats TakenFeats, list content.ClassID, old content.SpellID, repl content.Spell, gained LevelGained) FeatsTransition {
	i := feats.magicInitiate(list)
	if i < 0 {
		return FeatsRejected{Rejection: FeatNotHeld{Feat: featMagicInitiate}}
	}
	mi := feats.taken[i].choices.(MagicInitiateChoices)
	level := c.TotalLevel()
	switch {
	case gained.CharacterID() != c.ID() || gained.TotalLevel() != level:
		return FeatsRejected{Rejection: SpellChangeNotAllowedNow{}}
	case mi.SpellChangedAt(level):
		return FeatsRejected{Rejection: SpellChangeAlreadyUsed{}}
	}
	var problems []FeatChoiceProblem
	add := func(k FeatChoiceProblemKind) {
		problems = append(problems, FeatChoiceProblem{FeatChoiceReplacementSpell, k})
	}
	wantLevel := content.SpellLevel(0)
	switch {
	case slices.Contains(mi.cantrips, old):
	case mi.levelOne == old:
		wantLevel = 1
	default:
		add(FeatProblemMissing)
	}
	if len(problems) == 0 && repl.Level() != wantLevel {
		add(FeatProblemLevelMismatch)
	}
	if !slices.Contains(repl.Classes(), list) {
		add(FeatProblemNotOnSpellList)
	}
	if id := repl.ID(); id == mi.levelOne || slices.Contains(mi.cantrips, id) {
		add(FeatProblemDuplicate)
	}
	if len(problems) > 0 {
		return FeatsRejected{Rejection: FeatChoicesInvalid{problems: problems}}
	}
	next := slices.Clone(feats.taken)
	next[i].choices = mi.replacing(old, repl.ID(), level)
	return FeatsChanged{feats: TakenFeats{taken: next}, events: []FeatEvent{FeatSpellReplaced{id: c.ID(), list: list, old: old, repl: repl.ID()}}}
}
