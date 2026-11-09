package character

// BonusFeatsAtLevel20 is the table option for "Bonus Feats at Level 20" (SRD
// 5.2.1, p. 24): how many extra feats Experience Points above the level 20
// minimum earn. It is a closed set. Variants: NoBonusFeats, BonusFeatsFromXp.
type BonusFeatsAtLevel20 interface {
	// FeatsEarned returns the number of bonus feats xp earns.
	FeatsEarned(xp ExperiencePoints) int
	isBonusFeatsAtLevel20()
}

// NoBonusFeats is the default option: no bonus feats, whatever the XP.
type NoBonusFeats struct{}

// BonusFeatsFromXp awards one feat for every 30,000 Experience Points above the
// level 20 minimum (SRD 5.2.1, "Bonus Feats at Level 20", p. 24).
type BonusFeatsFromXp struct{}

// xpPerFeat is the step the SRD fixes, not a parameter.
const xpPerFeat = 30000

// FeatsEarned returns 0.
func (NoBonusFeats) FeatsEarned(ExperiencePoints) int { return 0 }

// FeatsEarned returns the complete 30,000 XP steps above the level 20 minimum,
// rounded down (SRD 5.2.1, "Round Down", p. 5), and 0 at or below it.
func (BonusFeatsFromXp) FeatsEarned(xp ExperiencePoints) int {
	threshold := ExperiencePoints{}.MinimumForLevel(MustLevel(MaxLevel))
	if xp.n <= threshold.n {
		return 0
	}
	return (xp.n - threshold.n) / xpPerFeat
}

func (NoBonusFeats) isBonusFeatsAtLevel20()     {}
func (BonusFeatsFromXp) isBonusFeatsAtLevel20() {}

// BonusFeats counts the bonus feats a character may still take. The zero value is
// ready to use. It is stateless and safe for concurrent use.
type BonusFeats struct{}

// Remaining returns the bonus feats not yet taken: what option earns for xp less
// the feats held that came from BonusFeat, never below 0. NoBonusFeats and a nil
// option give 0. A positive count lets the host call TakeFeat with a BonusFeat
// source and an AnyFeat scope; TakeFeat verifies no entitlement.
func (BonusFeats) Remaining(option BonusFeatsAtLevel20, xp ExperiencePoints, feats TakenFeats) int {
	if option == nil {
		return 0
	}
	return max(0, option.FeatsEarned(xp)-feats.TakenFrom(BonusFeat{}))
}
