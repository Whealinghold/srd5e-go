package content

// The limits of a stat block (content-monsters FR-095). They are constants and
// a function, not a package-level value (plan item P6, architecture A4), and
// the constructors and the decoder both use them, so the two cannot disagree.
// The page bounds are the printed pages of "Monsters A-Z" and "Animals" (SRD
// 5.2.1, contents pages, pp. 2-4).
const (
	// MonsterMinPage is the first printed page a stat block may begin on.
	MonsterMinPage = 258
	// MonsterMaxPage is the last printed page a stat block may begin on.
	MonsterMaxPage = 364
	// MonsterSection is the SRD section of the stat blocks in alphabetical
	// order, a valid source section of a [Monster].
	MonsterSection = "Monsters A-Z"
	// MonsterAnimalsSection is the SRD section of the animal stat blocks, the
	// other valid source section of a [Monster].
	MonsterAnimalsSection = "Animals"
	// MonsterMinHitPointsAverage is the lowest printed Hit Points average.
	MonsterMinHitPointsAverage = 1
	// MonsterMinHitDiceCount is the fewest Hit Point Dice of a monster.
	MonsterMinHitDiceCount = 1
	// MonsterMinSpellLevel is the lowest level a spell of a stat block is cast
	// at.
	MonsterMinSpellLevel = 1
	// MonsterMaxSpellLevel is the highest level a spell of a stat block is
	// cast at.
	MonsterMaxSpellLevel = 9
	// MonsterMinRecharge is the lowest face of a recharge die.
	MonsterMinRecharge = 1
	// MonsterMaxRecharge is the highest face of a recharge die.
	MonsterMaxRecharge = 6
	// MonsterMinUses is the fewest uses of a limited-use part, a spell
	// frequency or the legendary actions of a stat block.
	MonsterMinUses = 1
	// MonsterMinSaveDC is the lowest printed difficulty class of a saving
	// throw or a spell save.
	MonsterMinSaveDC = 1
	// MonsterMinRange is the shortest normal range of a ranged attack, in
	// feet.
	MonsterMinRange = 1
	// MonsterMinAttackCount is the fewest attacks of a Multiattack group and
	// the fewest attacks one replacement rule replaces.
	MonsterMinAttackCount = 1
	// MonsterMaxNumber is the largest magnitude of a printed whole number of a
	// stat block, 2^31 - 1, so the decoder reads it the same on every
	// platform integer width.
	MonsterMaxNumber = 1<<31 - 1
)

// MonsterHitPointDieFaces returns the faces a Hit Point Die or a damage die
// of a stat block may have, as a new slice: 4, 6, 8, 10, 12 and 20 (SRD
// 5.2.1, "Hit Points", p. 255; "Damage Notation", p. 257).
func MonsterHitPointDieFaces() []int { return []int{4, 6, 8, 10, 12, 20} }

// monsterDieFacesOK reports whether sides is one of the faces of
// [MonsterHitPointDieFaces].
func monsterDieFacesOK(sides int) bool {
	switch sides {
	case 4, 6, 8, 10, 12, 20:
		return true
	}
	return false
}

// monsterNumberOK reports whether v is a printed whole number a stat block
// may hold, of magnitude at most [MonsterMaxNumber].
func monsterNumberOK(v int) bool { return v >= -MonsterMaxNumber && v <= MonsterMaxNumber }
