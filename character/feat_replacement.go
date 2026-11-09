package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// FeatReplacement replaces a held feat by another: a Fighter's Fighting Style (SRD
// 5.2.1, "Fighting Style", p. 47) or the Wish spell's Sudden Learning (p. 175).
// The zero value is ready to use. It is stateless and safe for concurrent use.
type FeatReplacement struct{}

// Replace replaces the first instance of feat old by feat. The checks run in this
// order: FeatNotHeld (old is not held), FeatNotReplaceable (old gave an ability
// score increase, which cannot be undone). The old feat is then removed with its
// choices and use state, and FeatTakingRules.Take runs for the new feat on the
// result, so repeatability and prerequisites are seen without the old feat and a
// feat replaced by itself is accepted; the rejections of Take are returned
// unchanged. Success is FeatTakingChanged whose events are FeatReplaced, then
// FeatTaken, then those of AbilityScoreIncreaser.
//
// It returns an error matching srderr.ErrInvalidInput for the reasons Take does
// and for an invalid old id. The inputs are unchanged. It never panics.
func (FeatReplacement) Replace(c Character, feats TakenFeats, old content.FeatID, feat content.Feat, grant FeatGrant, choices FeatChoices, held HeldFeatures, resolved map[content.SpellID]content.Spell) (FeatTaking, error) {
	if !old.Valid() {
		return nil, invalidInput("replace an invalid feat id")
	}
	i := feats.indexOf(old)
	if i < 0 {
		return FeatTakingRejected{Rejection: FeatNotHeld{Feat: old}}, nil
	}
	if shape, _ := shapeOf(old); shape.givesIncrease() {
		return FeatTakingRejected{Rejection: FeatNotReplaceable{Feat: old}}, nil
	}
	tk, err := FeatTakingRules{}.Take(c, feats.without(i), feat, grant, choices, held, resolved)
	if err != nil {
		return nil, err
	}
	if ch, ok := tk.(FeatTakingChanged); ok {
		ch.events = slices.Insert(ch.events, 0, Event(FeatReplaced{id: c.ID(), old: old, repl: feat.ID()}))
		return ch, nil
	}
	return tk, nil
}
