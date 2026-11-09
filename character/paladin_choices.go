package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

const paladinStyleFeature content.FeatureID = "paladin_fighting_style"

// PaladinChoice is a Paladin option choice, a closed set. Variants: ChoosePaladinFightingStyle, ChooseBlessedWarriorCantrips,
// ReplaceBlessedWarriorCantrip.
type PaladinChoice interface{ isPaladinChoice() }

// ChoosePaladinFightingStyle takes a Fighting Style feat as the Paladin's Fighting Style. Feat is the feat's content. Construct
// it with a keyed literal.
type ChoosePaladinFightingStyle struct{ Feat content.Feat }

// ChooseBlessedWarriorCantrips takes the Blessed Warrior option with two Cleric cantrips. Construct it with a keyed literal.
type ChooseBlessedWarriorCantrips struct{ Cantrips []content.Spell }

// ReplaceBlessedWarriorCantrip replaces one Blessed Warrior cantrip with another Cleric cantrip, whenever a Paladin level is
// gained. Replacing none is not a choice. Construct it with a keyed literal.
type ReplaceBlessedWarriorCantrip struct {
	Old content.SpellID
	New content.Spell
}

func (ChoosePaladinFightingStyle) isPaladinChoice()   {}
func (ChooseBlessedWarriorCantrips) isPaladinChoice() {}
func (ReplaceBlessedWarriorCantrip) isPaladinChoice() {}

// PaladinChoiceRules decides the Fighting Style choice and the Blessed Warrior cantrips (SRD 5.2.1, "Level 2: Fighting Style",
// p. 54; class-paladin.md FR-038 to FR-041). The feat rules own the feat; it only maps their outcomes to PaladinReason. It is
// stateless; the zero value is ready to use. Safe for concurrent use.
type PaladinChoiceRules struct{}

// cantripReasons returns the FR-039 reasons that hold for the cantrips, in the order wrong_count, duplicate, not_a_cantrip,
// not_on_cleric_list. held are cantrips already held, which count as duplicates.
func cantripReasons(cantrips []content.Spell, held []content.SpellID, wantCount int) []PaladinReason {
	var out []PaladinReason
	add := func(r PaladinReason) {
		if !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	if wantCount > 0 && len(cantrips) != wantCount {
		add(PaladinWrongCount)
	}
	seen := slices.Clone(held)
	for _, s := range cantrips {
		if slices.Contains(seen, s.ID()) {
			add(PaladinDuplicate)
		}
		seen = append(seen, s.ID())
	}
	for _, s := range cantrips {
		if s.Level() != 0 {
			add(PaladinNotACantrip)
		}
	}
	for _, s := range cantrips {
		if !slices.Contains(s.Classes(), "cleric") {
			add(PaladinNotOnClericList)
		}
	}
	return out
}

// Style decides a Fighting Style feat choice for the Paladin of level with state. The feat is taken through the feat rules under
// the Fighting Style category; the changed Character and feats come back in the FeatTakingChanged. It returns the reasons that
// hold (and a nil FeatTakingChanged) or the change and no reasons: PaladinLevelTooLow below the level of the Fighting Style entry,
// PaladinAlreadyChosen for a repeat, PaladinNotAFightingStyleFeat for a feat of another category (FeatOutsideGrant) and
// PaladinFeatNotRepeatable. It returns an error matching srderr.ErrInvalidInput for a zero Character or feat.
func (PaladinChoiceRules) Style(c Character, feats TakenFeats, p PaladinProgression, level PaladinLevel, state PaladinState, feat content.Feat) (FeatTakingChanged, []PaladinReason, error) {
	if c.IsZero() || !feat.ID().Valid() {
		return FeatTakingChanged{}, nil, invalidInput("Fighting Style choice with a zero Character or feat")
	}
	var reasons []PaladinReason
	if !p.has(PaladinFeatureFightingStyle, level) {
		reasons = append(reasons, PaladinLevelTooLow)
	}
	_, chosen := state.FightingStyle()
	if chosen {
		reasons = append(reasons, PaladinAlreadyChosen)
	}
	grant, err := NewFeatGrant(ClassFeatureFeat{Class: PaladinClassID, Feature: paladinStyleFeature}, FeatFromCategory{Category: content.FeatFightingStyle})
	if err != nil {
		return FeatTakingChanged{}, nil, err
	}
	held, err := NewHeldFeatures(paladinStyleFeature)
	if err != nil {
		return FeatTakingChanged{}, nil, err
	}
	taking, err := FeatTakingRules{}.Take(c, feats, feat, grant, NoFeatChoices{}, held, nil)
	if err != nil {
		return FeatTakingChanged{}, nil, err
	}
	var changed FeatTakingChanged
	switch t := taking.(type) {
	case FeatTakingChanged:
		changed = t
	case FeatTakingRejected:
		switch t.Rejection.(type) {
		case FeatNotRepeatable:
			if !chosen {
				reasons = append(reasons, PaladinFeatNotRepeatable)
			}
		default:
			reasons = append(reasons, PaladinNotAFightingStyleFeat)
		}
	}
	slices.Sort(reasons)
	if len(reasons) > 0 {
		return FeatTakingChanged{}, reasons, nil
	}
	return changed, nil, nil
}

// BlessedWarrior decides taking the Blessed Warrior option with the cantrips: level and repeat as for Style, then exactly two
// distinct cantrips of spell level 0 with Cleric in their class list. It returns the new state and no reasons when accepted.
func (PaladinChoiceRules) BlessedWarrior(p PaladinProgression, level PaladinLevel, state PaladinState, cantrips []content.Spell) (PaladinState, []PaladinReason) {
	var reasons []PaladinReason
	if !p.has(PaladinFeatureFightingStyle, level) {
		reasons = append(reasons, PaladinLevelTooLow)
	}
	if _, chosen := state.FightingStyle(); chosen {
		reasons = append(reasons, PaladinAlreadyChosen)
	}
	reasons = append(reasons, cantripReasons(cantrips, nil, 2)...)
	slices.Sort(reasons)
	if len(reasons) > 0 {
		return state, reasons
	}
	state.style = BlessedWarrior{Cantrips: [2]content.SpellID{cantrips[0].ID(), cantrips[1].ID()}}
	return state, nil
}

// Replace decides replacing the Blessed Warrior cantrip old with the cantrip repl: PaladinNotBlessedWarrior without that option,
// PaladinNotHeld when old is not one of its cantrips, then the cantrip checks of BlessedWarrior with the other held cantrip
// counting for duplicates. The trigger (a gained level) and the limit of one replacement are the spellcasting rules'.
func (PaladinChoiceRules) Replace(state PaladinState, old content.SpellID, repl content.Spell) (PaladinState, []PaladinReason) {
	bw, ok := state.style.(BlessedWarrior)
	if !ok {
		return state, []PaladinReason{PaladinNotBlessedWarrior}
	}
	i := slices.Index(bw.Cantrips[:], old)
	if i < 0 {
		return state, []PaladinReason{PaladinNotHeld}
	}
	reasons := cantripReasons([]content.Spell{repl}, []content.SpellID{bw.Cantrips[1-i]}, 0)
	if repl.ID() == old {
		reasons = append(reasons, PaladinDuplicate)
	}
	slices.Sort(reasons)
	reasons = slices.Compact(reasons)
	if len(reasons) > 0 {
		return state, reasons
	}
	bw.Cantrips[i] = repl.ID()
	state.style = bw
	return state, nil
}
