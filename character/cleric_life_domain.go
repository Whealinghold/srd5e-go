package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

const (
	discipleOfLifeLevel = 3
	blessedHealerLevel  = 6
	// lifeDomainSpellsFeature is the id of the "Life Domain Spells" entry of the bundled Life Domain.
	lifeDomainSpellsFeature content.FeatureID = "life_domain_life_domain_spells"
)

// LifeDomainSpells reads the always-prepared spells of a Life Domain Cleric from the subclass content (SRD 5.2.1,
// "Level 3: Life Domain Spells", p. 40). It holds no copy of the grants. It is stateless; the zero value is ready to
// use. Safe for concurrent use.
type LifeDomainSpells struct{}

// GrantedAt returns the always-prepared spells of a Cleric of clericLevel who holds the subclass: the union of the
// grants whose level is at or below clericLevel, in grant order, as a new slice. It is empty for no subclass (nil),
// for a subclass other than the Life Domain, below the first grant and when the Life Domain Spells entry carries no
// AlwaysPreparedSpells fact. The caller passes the SubclassDefinition it holds.
func (LifeDomainSpells) GrantedAt(held *content.SubclassDefinition, clericLevel int) []content.SpellID {
	out := []content.SpellID{}
	if held == nil || held.ID() != LifeDomainSubclassID {
		return out
	}
	for _, f := range held.Features() {
		if f.ID() != lifeDomainSpellsFeature {
			continue
		}
		for _, fact := range f.Facts() {
			if a, ok := fact.(content.AlwaysPreparedSpells); ok {
				for _, g := range a.Grants() {
					if g.Level() <= clericLevel {
						out = append(out, g.SpellIDs()...)
					}
				}
			}
		}
	}
	return out
}

// NewlyGrantedAt returns the spells granted at clericLevel and not at the level before: those of GrantedAt(held, L)
// not in GrantedAt(held, L-1) (levels 5, 7 and 9 for the SRD data).
func (l LifeDomainSpells) NewlyGrantedAt(held *content.SubclassDefinition, clericLevel int) []content.SpellID {
	before := l.GrantedAt(held, clericLevel-1)
	return slices.DeleteFunc(l.GrantedAt(held, clericLevel), func(id content.SpellID) bool { return slices.Contains(before, id) })
}

// HealedCreature is a creature a healing spell restored Hit Points to. IsCleric is true for the Cleric itself. Key
// is the host's opaque key and is never an attribute. Construct it with a keyed literal.
type HealedCreature struct {
	Key      string
	IsCleric bool
}

// HealingCast is a spell that restored Hit Points: the level of the spell slot spent (0 when it was cast without a
// slot: a cantrip, a Ritual, a Divine Intervention cast, a Channel Divinity effect) and the creatures healed.
// Construct it with a keyed literal.
type HealingCast struct {
	SlotLevel int
	Healed    []HealedCreature
}

// HealingDice are the dice a spell or Channel Divinity effect would roll to restore Hit Points. Construct it with a
// keyed literal.
type HealingDice struct{ Count, Sides int }

// DiscipleOfLifeBonus is the additional Hit Points one creature regains from Disciple of Life.
type DiscipleOfLifeBonus struct {
	Key       string
	HitPoints int
}

// ResolvedHealingDice is the healing dice after Supreme Healing: Maximised is true when the dice are not rolled and
// Total is then the highest number possible (2d6 gives 12); otherwise the dice are rolled and Total is 0.
type ResolvedHealingDice struct {
	Count, Sides int
	Maximised    bool
	Total        int
}

// ResolveClericHealingResult is the closed set of outcomes of ResolveClericHealing. Variants:
// ResolveClericHealingResolved, ResolveClericHealingNotACleric.
type ResolveClericHealingResult interface{ isResolveClericHealingResult() }

// ResolveClericHealingResolved is the outcome of ResolveClericHealing. Only ResolveClericHealing builds one.
type ResolveClericHealingResolved struct {
	disciple []DiscipleOfLifeBonus
	healer   int
	dice     ResolvedHealingDice
	hasDice  bool
}

// DiscipleOfLife returns the additional Hit Points each creature regains (2 plus the slot's level), in the order of
// the creatures healed, as a new slice; empty when Disciple of Life does not apply.
func (r ResolveClericHealingResolved) DiscipleOfLife() []DiscipleOfLifeBonus {
	return slices.Clone(r.disciple)
}

// BlessedHealer returns the Hit Points the Cleric regains (2 plus the slot's level, once per cast); 0 when Blessed
// Healer does not apply.
func (r ResolveClericHealingResolved) BlessedHealer() int { return r.healer }

// Dice returns the healing dice after Supreme Healing and true when dice were supplied.
func (r ResolveClericHealingResolved) Dice() (ResolvedHealingDice, bool) { return r.dice, r.hasDice }

// ResolveClericHealingNotACleric is the outcome for a character with no Cleric levels.
type ResolveClericHealingNotACleric struct{}

func (ResolveClericHealingResolved) isResolveClericHealingResult()   {}
func (ResolveClericHealingNotACleric) isResolveClericHealingResult() {}

// ResolveClericHealing resolves the Life Domain healing features of a Cleric who held subclass (empty for none) when
// a healing spell was cast: Disciple of Life (level 3), Blessed Healer (level 6) and Supreme Healing (level 17) (SRD
// 5.2.1, pp. 40). The Life Domain features give nothing to a Cleric without the Life Domain, nothing for a cast with no
// spell slot, and Blessed Healer nothing when the Cleric is the only creature healed. dice, when not nil, are the healing
// dice of the spell or Channel Divinity effect; Supreme Healing maximises them.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, a slot level outside 0 to 9 or healing dice
// of fewer than 1 die or fewer than 2 sides. It never panics.
func ResolveClericHealing(c Character, subclass content.SubclassID, cast HealingCast, dice *HealingDice) (ResolveClericHealingResult, error) {
	if c.IsZero() || cast.SlotLevel < 0 || cast.SlotLevel > 9 {
		return nil, invalidInput("ResolveClericHealing: zero Character or a slot level outside 0 to 9")
	}
	level, ok := clericLevelOf(c)
	if !ok {
		return ResolveClericHealingNotACleric{}, nil
	}
	res := ResolveClericHealingResolved{disciple: []DiscipleOfLifeBonus{}}
	if dice != nil {
		total, err := (SupremeHealing{}).Maximum(dice.Count, dice.Sides)
		if err != nil {
			return nil, err
		}
		res.hasDice = true
		res.dice = ResolvedHealingDice{Count: dice.Count, Sides: dice.Sides}
		supreme := SupremeHealing{}
		if supreme.Applies(level, subclass) {
			res.dice.Maximised, res.dice.Total = true, total
		}
	}
	if subclass != LifeDomainSubclassID || cast.SlotLevel == 0 {
		return res, nil
	}
	bonus := 2 + cast.SlotLevel
	if level.n >= discipleOfLifeLevel {
		for _, h := range cast.Healed {
			res.disciple = append(res.disciple, DiscipleOfLifeBonus{Key: h.Key, HitPoints: bonus})
		}
	}
	if level.n >= blessedHealerLevel && slices.ContainsFunc(cast.Healed, func(h HealedCreature) bool { return !h.IsCleric }) {
		res.healer = bonus
	}
	return res, nil
}
