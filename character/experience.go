package character

// AwardExperience returns the Character with amount more Experience Points and
// one ExperiencePointsGained event (SRD 5.2.1, "Level Advancement", p. 23). It
// never gains a level: only LevelUp does. Awards past 355,000 succeed; a total
// past the largest int stays at it. The receiver is unchanged.
func (c Character) AwardExperience(amount ExperienceAmount) TransitionChanged[NoRejection] {
	next := c
	next.xp = c.xp.plus(amount)
	return changed[NoRejection](next, ExperiencePointsGained{id: c.id, amount: amount, total: next.xp})
}
