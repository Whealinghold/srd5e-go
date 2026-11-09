package character

// OverchannelUsed is the event of the Evoker's Overchannel being used (SRD 5.2.1, "Level
// 14: Overchannel", p. 82). It carries the use number since the last Long Rest and the
// level of the slot expended.
type OverchannelUsed struct {
	id        Identifier
	use       int
	slotLevel int
}

// CharacterID returns the id of the Character.
func (e OverchannelUsed) CharacterID() Identifier { return e.id }

// Use returns the use number since the last Long Rest (1 is the first, with no Necrotic damage).
func (e OverchannelUsed) Use() int { return e.use }

// SlotLevel returns the level of the Spell Slot expended.
func (e OverchannelUsed) SlotLevel() int { return e.slotLevel }

func (OverchannelUsed) isEvent()       {}
func (OverchannelUsed) isWizardEvent() {}
