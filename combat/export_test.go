package combat

// StartOverTheCap is Encounter.Start without the cap of 256 combatants, for the tests of the cap of 4096
// events per call, which only an encounter larger than the cap can reach (plan P10).
func StartOverTheCap(id EncounterID, established InitiativeOrderEstablished, combatants []Combatant) (Encounter, []Event, error) {
	return start(id, established, combatants, false)
}

// EstablishCommandOverTheCap builds an EstablishInitiativeOrderCommand without the cap of 256 standings, for the
// tests of the cap of 4096 events per call, which only an encounter larger than the cap can reach (plan P10). The
// caller passes valid, matching lists.
func EstablishCommandOverTheCap(standings []InitiativeStanding, profiles []CombatantProfile) EstablishInitiativeOrderCommand {
	return EstablishInitiativeOrderCommand{standings: standings, profiles: profiles}
}
