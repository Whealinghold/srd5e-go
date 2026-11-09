package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
)

// GiantStrengthType is a kind of giant a Belt or Potion of Giant Strength is
// named for. The zero value is invalid.
type GiantStrengthType int

// The five types.
const (
	GiantHill GiantStrengthType = iota + 1
	GiantFrostOrStone
	GiantFire
	GiantCloud
	GiantStorm
)

// StrengthScore returns the Strength score the type grants (21, 23, 25, 27, 29),
// or 0 for an invalid type (SRD 5.2.1, "Belt of Giant Strength", p. 213).
func (g GiantStrengthType) StrengthScore() int {
	if g < GiantHill || g > GiantStorm {
		return 0
	}
	return 19 + 2*int(g)
}

// String returns the snake_case name of the type, or "unknown".
func (g GiantStrengthType) String() string {
	names := []string{"hill", "frost_or_stone", "fire", "cloud", "storm"}
	if g < 1 || int(g) > len(names) {
		return "unknown"
	}
	return names[g-1]
}

// AbilityScoreSourceKind says what decided an effective score. The zero value is invalid.
type AbilityScoreSourceKind int

// The kinds.
const (
	BaseScore AbilityScoreSourceKind = iota + 1
	SetByItem
	RaisedByItem
)

// AbilityScoreSource says what decided one effective score: the base score or an
// item copy. Comparable; the zero value is invalid.
type AbilityScoreSource struct {
	kind     AbilityScoreSourceKind
	instance MagicItemInstanceID
}

// Kind returns BaseScore, SetByItem or RaisedByItem.
func (s AbilityScoreSource) Kind() AbilityScoreSourceKind { return s.kind }

// InstanceID returns the item copy, or "" for BaseScore.
func (s AbilityScoreSource) InstanceID() MagicItemInstanceID { return s.instance }

// EffectiveAbilityScores are the six scores after the active item effects, and
// what decided each. Immutable; the zero value is invalid.
type EffectiveAbilityScores struct {
	scores  ability.Scores
	sources [6]AbilityScoreSource
}

// Scores returns the effective scores.
func (e EffectiveAbilityScores) Scores() ability.Scores { return e.scores }

// Source returns what decided a's score, or the zero source for an invalid a.
func (e EffectiveAbilityScores) Source(a ability.Ability) AbilityScoreSource {
	if !a.Valid() {
		return AbilityScoreSource{}
	}
	return e.sources[a-1]
}

// boostFrom is the ability the Hammer of Thunderbolts boosts.
const boostFrom = ability.Strength

// EffectiveScoresOf applies the active score effects to base (SRD 5.2.1, "Belt of
// Giant Strength", p. 213; "Belt of Dwarvenkind", p. 213; "Hammer of
// Thunderbolts", p. 224). Per ability the highest set wins, and a set at or below
// the base has no effect. A boost then adds to the winning set when that set is
// marked for bestowing and its item is worn; a raise applies last, giving
// max(current, min(current + amount, cap)). Raises apply in input order. The
// input is not changed. It returns an error matching srderr.ErrInvalidInput for a
// zero base.
func EffectiveScoresOf(base ability.Scores, active ActiveItemEffects) (EffectiveAbilityScores, error) {
	if base.IsZero() {
		return EffectiveAbilityScores{}, invalidInput("effective ability scores: zero base scores")
	}
	var out EffectiveAbilityScores
	out.scores = base
	for _, a := range ability.All() {
		cur := base.Score(a).Value()
		src := AbilityScoreSource{kind: BaseScore}
		bestowed := false
		for _, s := range active.scores {
			if set, ok := s.effect.(SetAbilityScore); ok && set.ability == a && set.score.Value() > cur {
				cur, src, bestowed = set.score.Value(), AbilityScoreSource{kind: SetByItem, instance: s.instance}, set.bestows && s.worn
			}
		}
		if a == boostFrom && bestowed {
			for _, s := range active.scores {
				if b, ok := s.effect.(BoostBestowedScore); ok {
					cur = min(cur+b.amount, b.cap)
				}
			}
		}
		for _, s := range active.scores {
			if r, ok := s.effect.(RaiseAbilityScore); ok && r.ability == a {
				if next := max(cur, min(cur+r.amount, r.cap)); next != cur {
					cur, src = next, AbilityScoreSource{kind: RaisedByItem, instance: s.instance}
				}
			}
		}
		sc, err := ability.NewScore(cur)
		if err != nil { // unreachable: sets and caps are 1 to 30
			return EffectiveAbilityScores{}, err
		}
		if out.scores, err = out.scores.With(a, sc); err != nil { // unreachable: a is valid
			return EffectiveAbilityScores{}, err
		}
		out.sources[a-1] = src
	}
	return out, nil
}
