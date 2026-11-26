package scenarios_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The scenarios of this file take characters from creation to combat over the bundled SRD content: the twelve classes, the species and the
// spells of content/data. The bundle holds no backgrounds, feats, weapons or monsters yet, so a synthetic Sage background with its Alert
// feat, a goblin of the host's own and the host-modelled effects of Fire Bolt are layered over it; nothing else is invented.

// grpESource is the bundled content under a synthetic Sage background and its Origin feat.
func grpESource() character.ContentSource {
	sage := content.MustBackground("sage", "Sage", content.SourceCitation{}, content.BackgroundFields{
		AbilityScores:      []ability.Ability{ability.Constitution, ability.Intelligence, ability.Wisdom},
		Feat:               content.MustBackgroundFeatReference("alert", ""),
		SkillProficiencies: []content.Skill{content.SkillArcana, content.SkillHistory},
		ToolProficiency:    must(content.NewSpecificToolProficiency("Calligrapher’s Supplies")),
		EquipmentOptions:   content.NewEquipmentOptions(must(content.NewEquipmentOption(nil, 8)), must(content.NewEquipmentOption(nil, 50))),
	})
	alert := content.MustFeat("alert", "Alert", content.SourceCitation{}, content.FeatFields{
		Category: content.FeatOrigin, Benefits: []content.FeatBenefit{content.MustFeatBenefit("Initiative Proficiency", "Text.")},
	})
	return layeredSource{top: contenttest.NewFakeSource(sage, alert), base: data.MustBundledSource()}
}

// grpEHero creates a level 1 character of the class with the standard array assigned as given, a Sage background that raises Constitution by
// 2 and the other ability by 1, and Human as the species, through CreateCharacter over the bundled content.
func grpEHero(t *testing.T, id string, class content.ClassID, scores map[ability.Ability]int, plusOne ability.Ability, skills ...content.Skill) character.Character {
	t.Helper()
	cmd := character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier(id), ClassID: class, BackgroundID: "sage", SpeciesID: "human",
		AbilityScores:      character.MustAbilityScoreAssignment(character.StandardArray, scores),
		BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Constitution, plusOne),
		ClassSkills:        skills, ClassEquipment: character.OptionA, BackgroundEquipment: character.OptionB,
		Size: character.ChosenSize{Size: content.SizeMedium}, Languages: []content.Language{content.LanguageDwarvish, content.LanguageElvish},
		Alignment: content.AlignmentNeutralGood,
	}
	creator := must(character.NewCreateCharacter(character.WithSource(grpESource()), character.WithRoller(dicetest.NewFixedRoller())))
	res, err := creator.Run(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	created, ok := res.(character.CreateCharacterSucceeded)
	if !ok {
		t.Fatalf("CreateCharacter(%s) = %#v, want success", id, res)
	}
	return created.Character()
}

func grpEFighter(t *testing.T) character.Character {
	t.Helper()
	return grpEHero(t, "pc-fighter", "fighter", map[ability.Ability]int{
		ability.Strength: 15, ability.Dexterity: 13, ability.Constitution: 14, ability.Intelligence: 8, ability.Wisdom: 12, ability.Charisma: 10},
		ability.Wisdom, content.SkillAthletics, content.SkillPerception)
}

func grpEWizard(t *testing.T) character.Character {
	t.Helper()
	return grpEHero(t, "pc-wizard", "wizard", map[ability.Ability]int{
		ability.Strength: 8, ability.Dexterity: 14, ability.Constitution: 13, ability.Intelligence: 15, ability.Wisdom: 12, ability.Charisma: 10},
		ability.Intelligence, content.SkillInsight, content.SkillInvestigation)
}

// grpEStats derives the character's stats over the bundled content.
func grpEStats(t *testing.T, pc character.Character) character.DerivedStats {
	t.Helper()
	uc, err := character.NewCalculateDerivedStats(character.WithSource(data.MustBundledSource()))
	if err != nil {
		t.Fatal(err)
	}
	res, err := uc.Run(context.Background(), character.CalculateDerivedStatsCommand{
		ClassLevels: pc.ClassLevels(), AbilityScores: pc.AbilityScores(), HitPointMaximum: pc.HitPointMaximum(), HitDicePool: pc.HitDicePool()})
	if err != nil {
		t.Fatal(err)
	}
	ok, isOK := res.(character.CalculateDerivedStatsSucceeded)
	if !isOK {
		t.Fatalf("CalculateDerivedStats = %#v, want success", res)
	}
	return ok.Stats
}

// grpEPlayer is a combatant for a character: its Hit Point maximum is the character's, its current Hit Points are as given.
func grpEPlayer(t *testing.T, id combat.CombatantID, pc character.Character, current int) combat.Combatant {
	t.Helper()
	hp := must(combat.RestoreHitPointStatus(combat.MustHitPoints(current, pc.HitPointMaximum().Value(), 0), combat.Alive{}, combat.FallsUnconscious, combat.NoAffinities()))
	return must(combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.PlayerCharacterCombatant{ID: pc.ID()}}, combat.SideA, hp, combat.ConditionSet{}))
}

func grpEMonster(t *testing.T, id combat.CombatantID, hitPoints int) combat.Combatant {
	t.Helper()
	hp := must(combat.RestoreHitPointStatus(combat.MustHitPoints(hitPoints, hitPoints, 0), combat.Alive{}, combat.DiesInstantly, combat.NoAffinities()))
	return must(combat.NewCombatant(combat.CombatantRef{ID: id, Kind: combat.MonsterCombatant{ID: "fixture"}}, combat.SideB, hp, combat.ConditionSet{}))
}

// grpEFightStart rolls Initiative for the combatants in the given order with one d20 each (the roller answers in that order), establishes
// the order and starts the encounter.
type grpEEntry struct {
	member   combat.Combatant
	modifier int
}

func grpEFightStart(t *testing.T, name string, roller combat.Roller, entries ...grpEEntry) combat.Encounter {
	t.Helper()
	ctx := context.Background()
	var participants []combat.InitiativeParticipant
	var members []combat.Combatant
	for _, e := range entries {
		id := e.member.Ref().ID
		participants = append(participants, must(combat.NewInitiativeParticipant(combat.InitiativeParticipantInput{IDs: []combat.CombatantID{id}, Modifier: modifier.New(e.modifier)})))
		members = append(members, e.member)
	}
	rollCmd, err := combat.NewRollInitiativeCommand(combat.InitiativeMethodRolled, participants)
	if err != nil {
		t.Fatal(err)
	}
	rolled, err := must(combat.NewRollInitiative(roller)).Run(ctx, rollCmd)
	if err != nil {
		t.Fatal(err)
	}
	walk := must(combat.NewMovementSpeeds(30))
	var standings []combat.InitiativeStanding
	var profiles []combat.CombatantProfile
	for _, b := range rolled.(combat.RollInitiativeRolled).Breakdowns() {
		for _, id := range b.IDs() {
			standings = append(standings, combat.InitiativeStanding{Combatant: id, Count: b.Count()})
			profiles = append(profiles, combat.CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: walk})
		}
	}
	est, err := combat.NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	order, err := combat.NewEstablishInitiativeOrder().Run(ctx, est)
	if err != nil {
		t.Fatal(err)
	}
	est2, ok := order.(combat.InitiativeOrderEstablished)
	if !ok {
		t.Fatalf("EstablishInitiativeOrder = %#v, want an established order (no ties in this fixture)", order)
	}
	started, err := combat.NewStartEncounter().Run(ctx, combat.NewStartEncounterCommand(combat.MustEncounterID(name), est2, members))
	if err != nil {
		t.Fatal(err)
	}
	return started.(combat.StartEncounterApplied).Encounter()
}

func grpECurrent(t *testing.T, e combat.Encounter, id combat.CombatantID) int {
	t.Helper()
	c, ok := e.Find(id)
	if !ok {
		t.Fatalf("%v is not in the encounter", id)
	}
	return c.HitPoints().HitPoints().Current()
}

// grpEFireBoltModel is the host's model of Fire Bolt under the bundled id: a ranged spell attack for 1d10 Fire (SRD 5.2.1, "Fire Bolt", p. 132).
// The bundle holds the spell's text and casting data but no effect model yet.
func grpEFireBoltModel(t *testing.T) content.Spell {
	t.Helper()
	d10 := must(content.NewRolledSpellAmount(dice.MustParseExpression("1d10")))
	dmg := must(content.NewSpellDealsDamage(must(content.NewScalableAmount(d10, nil)), must(content.NewFixedDamageType(content.DamageFire)), content.AddsNothing))
	attack := must(content.NewAttackRollResolution(content.SpellAttackRanged, []content.SpellOutcome{dmg}, nil))
	part := must(content.NewSpellEffectPart(attack, must(content.NewPartInstances(1, nil))))
	chosen := must(content.NewTargetsChosen(content.SubjectCreatureOrObject, must(content.NewExactlyTargets(1, nil)), nil, content.MayIncludeCaster))
	return spellFixture(t, "fire_bolt", 0, must(content.NewModelledEffects(content.CoverageComplete, chosen, []content.SpellEffectPart{part})))
}

// grpEWizardSpells creates the wizard's spellcasting state over the bundled spells: three cantrips including Fire Bolt and two prepared spells.
func grpEWizardSpells(t *testing.T, pc character.Character) (character.SpellcastingState, map[content.SpellID]content.Spell) {
	t.Helper()
	spells := bundledSpells(t)
	created, err := character.NewCreateSpellcasting().Do(context.Background(), character.CreateSpellcastingCommand{Character: pc.ID(), Levels: pc.ClassLevels(), Spells: spells,
		Classes: []character.ClassSpellcastingSetup{{Class: character.CasterWizard, Cantrips: ids("fire_bolt", "prestidigitation", "dancing_lights"),
			Prepared: ids("magic_missile", "sleep"), Spellbook: character.NewSpellbook(ids("magic_missile", "sleep", "detect_magic", "burning_hands", "chromatic_orb", "fog_cloud"))}}})
	if err != nil {
		t.Fatal(err)
	}
	state, ok := created.(character.CreateSpellcastingCreated)
	if !ok {
		t.Fatalf("CreateSpellcasting = %#v, want created", created)
	}
	return state.State(), spells
}

// grpEAttackScene is a melee scene within reach with a clear line of sight.
func grpEAttackScene(t *testing.T) combat.AttackScene {
	t.Helper()
	return must(combat.NewAttackScene(5, combat.SightRelationCanSee, combat.SightRelationCanSee, combat.LineOfSightBlocked))
}

// Requirement: doc/specs/character-creation.md FR-002, FR-013; doc/specs/derived-stats.md FR-009, FR-030, FR-050; doc/specs/casting-and-concentration.md
// FR-057, FR-072; doc/specs/spell-effects.md FR-064, FR-065; doc/specs/combat-action-resolution.md FR-091, FR-092; doc/specs/encounter-aggregate.md FR-021,
// FR-024; plan-phase-4.md "2. What already exists (reused as-is)" (the character context feeds combat) (SRD 5.2.1, "Making an Attack", pp. 14-15; "Fire Bolt", p. 132; "Armor Class", p. 7) - a Fighter
// and a Wizard are created from the bundled classes, their derived Hit Points, Armor Class and bonuses feed an encounter against a goblin: the goblin
// hits the Fighter on exactly their Armor Class, the Fighter hits the goblin on exactly its Armor Class, and the Wizard casts the bundled Fire Bolt
// through CastSpell, whose report resolves through the spell-effects path and kills the goblin; the host ends the encounter.
// Path: happy.
func TestBundledFighterAndWizardDefeatAGoblin(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fighter, wizard := grpEFighter(t), grpEWizard(t)
	if fighter.HitPointMaximum().Value() != 13 || wizard.HitPointMaximum().Value() != 8 {
		t.Fatalf("Hit Point maximums = %d and %d, want 13 (d10 + 3) and 8 (d6 + 2) from the bundled class dice [FR-013]",
			fighter.HitPointMaximum().Value(), wizard.HitPointMaximum().Value())
	}
	fStats, wStats := grpEStats(t, fighter), grpEStats(t, wizard)
	if fStats.ArmorClass() != 11 || wStats.ArmorClass() != 12 || fStats.ProficiencyBonus().Value() != 2 || fStats.HitPointMaximum().Value() != 13 {
		t.Fatalf("Armor Class %d and %d, Proficiency Bonus %d, maximum %d, want 11, 12, +2 and 13 [FR-030, FR-009, FR-050]",
			fStats.ArmorClass(), wStats.ArmorClass(), fStats.ProficiencyBonus().Value(), fStats.HitPointMaximum().Value())
	}

	hero, mage, goblin := combat.MustCombatantID("fighter"), combat.MustCombatantID("wizard"), combat.MustCombatantID("goblin")
	dexMod := func(pc character.Character) int { return pc.AbilityScores().Modifier(ability.Dexterity).Value() }
	// Initiative d20s in the order of the entries: goblin 16 + 1 = 17, Fighter 14 + 1 = 15, Wizard 12 + 2 = 14.
	e := grpEFightStart(t, "bundled-goblin", dicetest.NewFixedRoller(16, 14, 12),
		grpEEntry{grpEMonster(t, goblin, 18), 1}, grpEEntry{grpEPlayer(t, hero, fighter, fighter.HitPointMaximum().Value()), dexMod(fighter)},
		grpEEntry{grpEPlayer(t, mage, wizard, wizard.HitPointMaximum().Value()), dexMod(wizard)})
	var order []combat.CombatantID
	for _, s := range e.Tracker().Order().Standings() {
		order = append(order, s.Combatant)
	}
	if !slices.Equal(order, []combat.CombatantID{goblin, hero, mage}) || e.Tracker().Active().Combatant != goblin {
		t.Fatalf("order = %v active %v, want goblin, fighter, wizard with the goblin first", order, e.Tracker().Active().Combatant)
	}
	var ledger combat.ActionLedger
	advance := func() {
		t.Helper()
		cmd, err := combat.NewAdvanceCombatTurnCommand(e, ledger, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := combat.NewAdvanceCombatTurn().Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		done := res.(combat.AdvanceCombatTurnCompleted)
		e, ledger = done.Encounter(), done.Ledger()
	}
	attack := func(roller combat.Roller, attacker, target combat.CombatantID, ac d20.ArmorClass, mods d20.RollModifiers, spec combat.DamageSpec) combat.PerformAttackResult {
		t.Helper()
		atk := combat.MustResolveAttackCommand(combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: ac, Modifiers: mods, Distance: 5, Damage: spec})
		cmd, err := combat.NewPerformAttackCommand(combat.PerformAttackParams{Encounter: e, Ledger: ledger, AttackerID: attacker, TargetID: target,
			Cost: combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansOtherWeapon}, Scene: grpEAttackScene(t), Attack: atk})
		if err != nil {
			t.Fatal(err)
		}
		res, err := must(combat.NewPerformAttack(roller)).Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	mods := func(abilityMod, proficiency int) d20.RollModifiers {
		return d20.NewRollModifiers(modifier.New(abilityMod)).WithProficiencyBonus(modifier.New(proficiency))
	}

	// The goblin (+4 to hit) rolls a 7 for 11: exactly the Fighter's Armor Class of 11, which hits; 1d6 of 3 and +2 is 5 damage.
	strike := attack(dicetest.NewFixedRoller(7, 3), goblin, hero, fStats.ArmorClass(), mods(2, 2), combat.MustDamageSpec(combat.MustDiceDamage(1, dice.D6(), content.DamageSlashing, 0, true)))
	hit := strike.(combat.PerformAttackCompleted)
	e, ledger = hit.Encounter(), hit.Ledger()
	if got := grpECurrent(t, e, hero); got != 13-5 || len(hit.Resolutions()) != 1 {
		t.Fatalf("Fighter Hit Points = %d with %d damage resolutions, want 8 and one: a total equal to Armor Class hits [d20-tests.md]", got, len(hit.Resolutions()))
	}
	advance()

	// The Fighter (Strength +2, Proficiency +2) rolls 11 for 15: exactly the goblin's Armor Class of 15; 2d6 of 4 and 5 and +2 is 11.
	spec := combat.MustDamageSpec(combat.MustDiceDamage(2, dice.D6(), content.DamageSlashing, 0, true))
	swing := attack(dicetest.NewFixedRoller(11, 4, 5), hero, goblin, 15, mods(fighter.AbilityScores().Modifier(ability.Strength).Value(), fStats.ProficiencyBonus().Value()), spec)
	cut := swing.(combat.PerformAttackCompleted)
	e, ledger = cut.Encounter(), cut.Ledger()
	if got := grpECurrent(t, e, goblin); got != 18-11 {
		t.Fatalf("goblin Hit Points = %d, want 7 [FR-091, FR-092]", got)
	}
	advance()

	// The Wizard spends the Magic action, casts the bundled Fire Bolt (a cantrip: no slot) and the report resolves the spell attack: a 10 and +5 is 15,
	// exactly the goblin's Armor Class; 1d10 of 7 kills it.
	if e.Tracker().Active().Combatant != mage {
		t.Fatalf("active = %v, want the Wizard", e.Tracker().Active().Combatant)
	}
	magicCmd, err := combat.NewTakeCombatantActionCommand(e, ledger, mage, combat.SpendAction{Choice: combat.SimpleActionChoice{Action: combat.SimpleStandardActionMagic}})
	if err != nil {
		t.Fatal(err)
	}
	magicRes, err := combat.NewTakeCombatantAction().Run(ctx, magicCmd)
	if err != nil {
		t.Fatal(err)
	}
	spent := magicRes.(combat.TakeCombatantActionCompleted)
	e, ledger = spent.Encounter(), spent.Ledger()

	state, spells := grpEWizardSpells(t, wizard)
	stats := character.Stats(character.CasterWizard, wizard.AbilityScores().Modifier(ability.Intelligence), wizard.ProficiencyBonus())
	cast, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Class: character.CasterWizard, Stats: &stats,
		Spell: spells[content.MustSpellID("fire_bolt")], Mode: character.CastModeNormal, Context: castingFacts(t)})
	if err != nil {
		t.Fatal(err)
	}
	performed, ok := cast.(character.CastSpellCast)
	if !ok {
		t.Fatalf("CastSpell = %#v, want CastSpellCast", cast)
	}
	bonus, hasBonus := performed.Report().AttackBonus()
	if !hasBonus || bonus.Value() != 5 || performed.Report().CountsAsSlotSpell() || performed.State().Slots(character.PoolSpellcasting) != state.Slots(character.PoolSpellcasting) {
		t.Fatalf("report attack bonus %v %v, slot spell %v, want +5 (Intelligence +3, Proficiency +2) and no slot spent [FR-057]",
			bonus, hasBonus, performed.Report().CountsAsSlotSpell())
	}
	facts := must(combat.NewSpellAttackFacts(combat.SpellAttackFactsParams{ArmorClass: 15, Distance: 30, Range: must(combat.NewAttackRange(120, 0))}))
	target := must(combat.NewSpellTarget(combat.SpellTargetParams{Ref: combat.CreatureTargetRef{ID: goblin}, AttackFacts: facts}))
	resolved, err := must(combat.NewResolveSpellEffects(dicetest.NewFixedRoller(10, 7))).Run(ctx, combat.ResolveSpellEffectsCommand{
		Spell: grpEFireBoltModel(t), Report: performed.Report(), Caster: mage, AttackModifiers: stats.AttackRollModifiers(), Targets: []combat.SpellTarget{target}})
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := resolved.(combat.SpellEffectsResolved)
	if !ok || len(plan.Targets()) != 1 || len(plan.Targets()[0].Damage()) != 1 {
		t.Fatalf("ResolveSpellEffects = %#v, want one planned damage [FR-064, FR-065]", resolved)
	}
	pd := plan.Targets()[0].Damage()[0]
	kill, err := combat.NewDealDamageToCombatantCommand(e, goblin, pd.Damages(), pd.Adjustments(), combat.OtherSource, pd.HitKind())
	if err != nil {
		t.Fatal(err)
	}
	dealt, err := must(combat.NewDealDamageToCombatant(dicetest.NewFixedRoller())).Run(ctx, kill)
	if err != nil {
		t.Fatal(err)
	}
	done := dealt.(combat.DealDamageToCombatantCompleted)
	e = done.Encounter()
	dead, _ := e.Find(goblin)
	if cause, isDead := dead.Death(); !isDead || grpECurrent(t, e, goblin) != 0 || !slices.Contains(kinds(done.Events()), "died") {
		t.Fatalf("goblin = %d Hit Points, death %v %v, events %v, want dead at 0 [FR-024]", grpECurrent(t, e, goblin), cause, isDead, kinds(done.Events()))
	}

	end, err := combat.NewEndEncounterCommand(e, combat.EndReasonSideDefeated{Side: combat.SideB})
	if err != nil {
		t.Fatal(err)
	}
	over, err := combat.NewEndEncounter().Run(ctx, end)
	if err != nil {
		t.Fatal(err)
	}
	if _, ended := over.(combat.EndEncounterApplied); !ended {
		t.Fatalf("EndEncounter = %#v, want applied", over)
	}
	// The goblin is dead, so the encounter is over: another attack is refused with nothing rolled.
	refusedRoller := dicetest.NewFixedRoller()
	e = over.(combat.EndEncounterApplied).Encounter()
	late := attack(refusedRoller, hero, goblin, 15, mods(2, 2), spec)
	if _, refused := late.(combat.PerformAttackRejected); !refused {
		t.Errorf("an attack after the end = %#v, want a rejection [FR-092]", late)
	}
}

// Requirement: doc/specs/character-aggregate.md FR-020, FR-025, FR-045; doc/specs/rests-and-hit-dice.md FR-040, FR-053, FR-054; doc/specs/hit-points-and-damage.md
// FR-101; doc/specs/derived-stats.md FR-050, FR-071 (SRD 5.2.1, "Level Advancement", p. 23; "Short Rest", p. 187; "Bloodied", p. 187) - a Fighter takes
// 8 damage in a first fight and survives on 5, earns 300 XP and a level (a new maximum of 22 from the bundled d10), spends one Hit Point Die on a Short
// Rest (a 7 and +3 regains 10, to 15), and in the next encounter is hit again: the new maximum and the Bloodied threshold of 11 are the ones the combat
// context sees.
// Path: happy.
func TestBundledFighterLevelsUpRestsAndFightsAgain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fighter, goblin, hero := grpEFighter(t), combat.MustCombatantID("goblin"), combat.MustCombatantID("fighter")

	// Fight one: the goblin hits for 8; the Fighter (13) is at 5 and the host ends the encounter with the goblin beaten.
	first := grpEFightStart(t, "level-up-fight-1", dicetest.NewFixedRoller(5, 15),
		grpEEntry{grpEMonster(t, goblin, 7), 1}, grpEEntry{grpEPlayer(t, hero, fighter, 13), 1})
	hurt, err := combat.NewDealDamageToCombatantCommand(first, hero, []combat.Damage{combat.MustDamage(8, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	res, err := must(combat.NewDealDamageToCombatant(dicetest.NewFixedRoller())).Run(ctx, hurt)
	if err != nil {
		t.Fatal(err)
	}
	wounded := res.(combat.DealDamageToCombatantCompleted)
	current := grpECurrent(t, wounded.Encounter(), hero)
	if current != 5 {
		t.Fatalf("Hit Points after fight one = %d, want 5", current)
	}
	if _, err := combat.NewEndEncounter().Run(ctx, must(combat.NewEndEncounterCommand(wounded.Encounter(), combat.EndReasonSideDefeated{Side: combat.SideB}))); err != nil {
		t.Fatal(err)
	}

	// Too little XP: no level (FR-025's precondition), then 300 XP and the level.
	levelUp := must(character.NewLevelUpCharacter(character.WithSource(data.MustBundledSource()), character.WithRoller(dicetest.NewFixedRoller())))
	tooSoon, err := levelUp.Run(ctx, character.LevelUpCharacterCommand{Character: fighter, Class: "fighter", HitPointChoice: character.TakeFixedValue{}})
	if rej, ok := tooSoon.(character.LevelUpCharacterRejected); err != nil || !ok || rej.Rejection != (character.LevelUpInsufficientExperience{}) {
		t.Fatalf("level up with 0 XP = %#v, %v, want insufficient experience", tooSoon, err)
	}
	awarded, err := character.NewAwardExperience().Run(ctx, character.AwardExperienceCommand{Character: fighter, Amount: character.MustExperienceAmount(300)})
	if err != nil {
		t.Fatal(err)
	}
	leveled, err := levelUp.Run(ctx, character.LevelUpCharacterCommand{Character: awarded.(character.AwardExperienceSucceeded).Character(), Class: "fighter", HitPointChoice: character.TakeFixedValue{}})
	if err != nil {
		t.Fatal(err)
	}
	fighter = leveled.(character.LevelUpCharacterSucceeded).Character()
	if fighter.TotalLevel().Value() != 2 || fighter.HitPointMaximum().Value() != 13+6+3 || fighter.HitDicePool().Total() != 2 {
		t.Fatalf("level %d maximum %d dice %d, want level 2, maximum 22 (13 + fixed d10 6 + Constitution 3) and two dice [FR-025, FR-045]",
			fighter.TotalLevel().Value(), fighter.HitPointMaximum().Value(), fighter.HitDicePool().Total())
	}
	if stats := grpEStats(t, fighter); stats.HitPointMaximum().Value() != 22 {
		t.Errorf("derived maximum = %d, want the stored 22 [FR-050]", stats.HitPointMaximum().Value())
	}

	// Short Rest: one Hit Point Die, a 7 plus Constitution +3 regains 10; a second spend is possible, the pool of two then runs out.
	conMod := fighter.AbilityScores().Modifier(ability.Constitution)
	spender := must(character.NewHitDieSpender(dicetest.NewFixedRoller(7, 4)))
	spend := func(spentSoFar map[content.HitPointDie]int) character.SpendHitPointDieResult {
		t.Helper()
		cmd, err := character.NewSpendHitPointDieCommand(fighter.ID(), fighter.HitDicePool(), spentSoFar, content.HitDieD10, conMod, character.ShortRest)
		if err != nil {
			t.Fatal(err)
		}
		out, err := spender.SpendHitPointDie(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	one := spend(nil).(character.SpendHitPointDieApplied)
	if one.HitPointsRegained() != 10 || one.SpentHitDice().Count(content.HitDieD10) != 1 {
		t.Fatalf("first spend regained %d with %d spent, want 10 and 1 [FR-054]", one.HitPointsRegained(), one.SpentHitDice().Count(content.HitDieD10))
	}
	current += one.HitPointsRegained()
	two := spend(map[content.HitPointDie]int{content.HitDieD10: 1}).(character.SpendHitPointDieApplied)
	if two.HitPointsRegained() != 7 {
		t.Fatalf("second spend regained %d, want 7 (4 + 3)", two.HitPointsRegained())
	}
	if rej, ok := spend(map[content.HitPointDie]int{content.HitDieD10: 2}).(character.SpendHitPointDieRejected); !ok || rej.Reason() != character.NoneRemaining {
		t.Errorf("a third spend from a pool of two = %#v, want NoneRemaining [FR-040]", rej)
	}
	// The host keeps the Hit Points the rest gave, capped at the maximum: 5 + 10 = 15, the second die left unspent for this story.
	if current != 15 || current > fighter.HitPointMaximum().Value() {
		t.Fatalf("current Hit Points = %d, want 15 of 22", current)
	}

	// Fight two: the encounter is built from the new maximum and the Hit Points the rest gave; the goblin hits for 5, leaving 10.
	second := grpEFightStart(t, "level-up-fight-2", dicetest.NewFixedRoller(5, 15),
		grpEEntry{grpEMonster(t, goblin, 7), 1}, grpEEntry{grpEPlayer(t, hero, fighter, current), 1})
	if c, _ := second.Find(hero); c.HitPoints().HitPoints() != combat.MustHitPoints(15, 22, 0) {
		t.Fatalf("Fighter in fight two = %+v, want 15 of 22", c.HitPoints().HitPoints())
	}
	hit, err := combat.NewDealDamageToCombatantCommand(second, hero, []combat.Damage{combat.MustDamage(5, content.DamageSlashing)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		t.Fatal(err)
	}
	out, err := must(combat.NewDealDamageToCombatant(dicetest.NewFixedRoller())).Run(ctx, hit)
	if err != nil {
		t.Fatal(err)
	}
	after := out.(combat.DealDamageToCombatantCompleted)
	if got := grpECurrent(t, after.Encounter(), hero); got != 10 || !slices.Contains(kinds(after.Events()), "became_bloodied") {
		t.Errorf("Hit Points %d events %v, want 10 of 22 and became_bloodied at or below half of 22 [hit-points-and-damage.md FR-101]", got, kinds(after.Events()))
	}
	if bloodied, err := character.IsBloodied(10, fighter.HitPointMaximum()); err != nil || !bloodied {
		t.Errorf("IsBloodied(10, 22) = %v, %v, want true (the character context agrees) [derived-stats.md FR-071]", bloodied, err)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-053, FR-054 and doc/specs/level-advancement-and-multiclassing.md FR-020 (SRD 5.2.1, "Level
// Advancement", p. 23; "Short Rest", p. 187) - a failing dice port while spending a Hit Point Die is a port error and spends nothing, an unlisted die
// is rejected without a roll, and a class the content source lacks is reported rather than levelled.
// Path: unhappy (failure of collaborators, missing content).
func TestBundledRestAndLevelUpFailures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fighter := grpEFighter(t)
	conMod := fighter.AbilityScores().Modifier(ability.Constitution)

	down := errors.New("backend down")
	asked := 0
	failing := must(character.NewHitDieSpender(dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { asked++; return dice.Result{}, down })))
	cmd, err := character.NewSpendHitPointDieCommand(fighter.ID(), fighter.HitDicePool(), nil, content.HitDieD10, conMod, character.ShortRest)
	if err != nil {
		t.Fatal(err)
	}
	_, err = failing.SpendHitPointDie(ctx, cmd)
	var pe *srderr.PortError
	if !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, down) {
		t.Errorf("failing roller err = %v, want a dice port error wrapping the cause [FR-054]", err)
	}
	wrongDie, err := character.NewSpendHitPointDieCommand(fighter.ID(), fighter.HitDicePool(), nil, content.HitDieD6, conMod, character.ShortRest)
	if err != nil {
		t.Fatal(err)
	}
	asked = 0
	if res, err := failing.SpendHitPointDie(ctx, wrongDie); err != nil {
		t.Fatal(err)
	} else if r, ok := res.(character.SpendHitPointDieRejected); !ok || r.Reason() != character.NotInPool || asked != 0 {
		t.Errorf("a d6 from a d10 pool = %#v after %d rolls, want NotInPool with no roll [FR-053]", res, asked)
	}
	if _, err := character.NewSpendHitPointDieCommand(fighter.ID(), fighter.HitDicePool(), nil, 0, conMod, character.ShortRest); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an invalid die err = %v, want ErrInvalidInput", err)
	}

	empty := must(character.NewLevelUpCharacter(character.WithSource(contenttest.NewFakeSource()), character.WithRoller(dicetest.NewFixedRoller())))
	awarded := must(character.NewAwardExperience().Run(ctx, character.AwardExperienceCommand{Character: fighter, Amount: character.MustExperienceAmount(300)}))
	missing, err := empty.Run(ctx, character.LevelUpCharacterCommand{Character: awarded.(character.AwardExperienceSucceeded).Character(), Class: "fighter", HitPointChoice: character.TakeFixedValue{}})
	if m, ok := missing.(character.LevelUpCharacterContentMissing); err != nil || !ok || len(m.Missing()) != 1 {
		t.Errorf("a class the source lacks = %#v, %v, want content missing [FR-020]", missing, err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-078, FR-079, FR-103, FR-163; doc/specs/casting-and-concentration.md FR-072, FR-079, FR-128
// (SRD 5.2.1, "Concentration", p. 179; "Sleep", p. 160) - a Wizard created from the bundled class casts the bundled Sleep with a level 1 slot and is
// concentrating; a goblin hits them for 6 and the engine rolls the Constitution save against DC 10 (damage 6, below the DC floor): a 12 plus the
// derived +2 holds, a 6 plus +2 breaks Concentration, and the state the engine returns equals the one the character context's BreakConcentration gives
// for the same reason.
// Path: happy (held) and unhappy (rule interaction, failed save).
func TestBundledWizardConcentrationBrokenByGoblinDamage(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		save  int
		holds bool
	}{{"the save is held", 12, true}, {"the save fails and Concentration ends", 6, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			wizard := grpEWizard(t)
			stats := grpEStats(t, wizard)
			state, spells := grpEWizardSpells(t, wizard)
			slot := character.NewSpellSlotChoiceSpellcasting(character.MustSlotLevel(1))
			spell := spells[content.MustSpellID("sleep")]
			sheet := character.Stats(character.CasterWizard, wizard.AbilityScores().Modifier(ability.Intelligence), wizard.ProficiencyBonus())
			cast, err := character.NewCastSpell().Do(ctx, character.CastSpellCommand{State: state, Class: character.CasterWizard, Stats: &sheet, Spell: spell, Mode: character.CastModeNormal, Slot: slot, Context: castingFacts(t)})
			if err != nil {
				t.Fatal(err)
			}
			performed, ok := cast.(character.CastSpellCast)
			if !ok {
				t.Fatalf("CastSpell(sleep) = %#v, want cast", cast)
			}
			concentrating := performed.State()
			if _, going := concentrating.Concentration(); !going || concentrating.Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(1)) != state.Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(1))-1 {
				t.Fatalf("after Sleep Concentration going = %v, want Concentration and one level 1 slot spent", going)
			}

			mage, goblin := combat.MustCombatantID("wizard"), combat.MustCombatantID("goblin")
			e := grpEFightStart(t, "bundled-concentration", dicetest.NewFixedRoller(16, 12),
				grpEEntry{grpEMonster(t, goblin, 7), 1}, grpEEntry{grpEPlayer(t, mage, wizard, wizard.HitPointMaximum().Value()), 2})
			// The save modifier is the Constitution saving throw the derived stats give the combat context: the ability modifier, no proficiency.
			conSave, err := stats.SavingThrow(ability.Constitution).Breakdown().ToRollModifiers()
			if err != nil {
				t.Fatal(err)
			}
			track, err := combat.NewTracksConcentration(concentrating, conSave, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			// The goblin (+4) rolls a 13 for 17 against the Wizard's Armor Class of 12; 1d6 of 4 and +2 is 6. The save is the next d20.
			atk := combat.MustResolveAttackCommand(combat.ResolveAttackParams{Form: combat.MeleeWeaponAttack, ArmorClass: stats.ArmorClass(),
				Modifiers: d20.NewRollModifiers(modifier.New(2)).WithProficiencyBonus(modifier.New(2)), Distance: 5,
				Damage: combat.MustDamageSpec(combat.MustDiceDamage(1, dice.D6(), content.DamageSlashing, 0, true))})
			cmd, err := combat.NewPerformAttackCommand(combat.PerformAttackParams{Encounter: e, AttackerID: goblin, TargetID: mage,
				Cost:  combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansOtherWeapon},
				Scene: grpEAttackScene(t), Attack: atk, TargetConcentration: track})
			if err != nil {
				t.Fatal(err)
			}
			res, err := must(combat.NewPerformAttack(dicetest.NewFixedRoller(13, 4, c.save))).Run(ctx, cmd)
			if err != nil {
				t.Fatal(err)
			}
			done := res.(combat.PerformAttackCompleted)
			if got := grpECurrent(t, done.Encounter(), mage); got != wizard.HitPointMaximum().Value()-6 {
				t.Fatalf("Wizard Hit Points = %d, want %d [FR-078]", got, wizard.HitPointMaximum().Value()-6)
			}
			engineState, _ := done.TargetSpellcasting()
			_, going := engineState.Concentration()
			if going != c.holds {
				t.Fatalf("Concentration going = %v after save %d, want %v [FR-079]", going, c.save, c.holds)
			}
			if c.holds {
				if held, ok := done.Concentration().(combat.ConcentrationHeld); !ok || held.DC() != 10 {
					t.Errorf("outcome = %#v, want held against DC 10 [FR-078]", done.Concentration())
				}
				return
			}
			if _, ok := done.Concentration().(combat.ConcentrationCheckFailed); !ok {
				t.Fatalf("outcome = %#v, want a failed check [FR-079]", done.Concentration())
			}
			// The character context's own BreakConcentration, given the failed save, ends it the same way (FR-128, combat FR-163).
			owner, err := character.NewBreakConcentration().Do(ctx, character.BreakConcentrationCommand{State: concentrating, Reason: character.BreakFailedSave})
			ended, ok := owner.(character.BreakConcentrationEnded)
			if err != nil || !ok {
				t.Fatalf("BreakConcentration = %#v, %v, want ended", owner, err)
			}
			if !reflect.DeepEqual(ended.State(), engineState) {
				t.Errorf("the engine's state after the failed save differs from the character context's BreakConcentration state [FR-072, FR-128]")
			}
			if !slices.Contains(kinds(done.Events()), "concentration_checked") {
				t.Errorf("events = %v, want concentration_checked [FR-103]", kinds(done.Events()))
			}
		})
	}
}

// Requirement: doc/specs/character-creation.md FR-002, FR-013 and doc/specs/derived-stats.md FR-030 (SRD 5.2.1, "Create Your Character", p. 19) -
// creation over the bundled content is refused, with the first problem reported, when the class is unknown to the source or the languages are not
// the two a character of this species chooses, and nothing is built.
// Path: unhappy (missing content, invalid input).
func TestBundledCreationRefusals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	creator := must(character.NewCreateCharacter(character.WithSource(grpESource()), character.WithRoller(dicetest.NewFixedRoller())))
	cmd := character.CreateCharacterCommand{
		CharacterID: character.MustIdentifier("pc-bad"), ClassID: "fighter", BackgroundID: "sage", SpeciesID: "human",
		AbilityScores: character.MustAbilityScoreAssignment(character.StandardArray, map[ability.Ability]int{
			ability.Strength: 15, ability.Dexterity: 13, ability.Constitution: 14, ability.Intelligence: 8, ability.Wisdom: 12, ability.Charisma: 10}),
		BackgroundIncrease: character.MustPlusTwoPlusOne(ability.Constitution, ability.Wisdom),
		ClassSkills:        []content.Skill{content.SkillAthletics, content.SkillPerception}, ClassEquipment: character.OptionA, BackgroundEquipment: character.OptionB,
		Size: character.ChosenSize{Size: content.SizeMedium}, Alignment: content.AlignmentNeutralGood,
	}
	noLanguages, err := creator.Run(ctx, cmd) // the species grants two languages to choose: none chosen is refused
	if err != nil {
		t.Fatal(err)
	}
	if _, rejected := noLanguages.(character.CreateCharacterRejected); !rejected {
		t.Errorf("no languages = %#v, want a rejection", noLanguages)
	}
	cmd.Languages = []content.Language{content.LanguageDwarvish, content.LanguageElvish}
	cmd.ClassID = "artificer"
	missing, err := creator.Run(ctx, cmd)
	if m, ok := missing.(character.CreateCharacterContentMissing); err != nil || !ok || len(m.Missing()) != 1 {
		t.Errorf("a class the bundle lacks = %#v, %v, want content missing", missing, err)
	}
	cmd.ClassID = "fighter"
	cmd.ClassSkills = []content.Skill{content.SkillAthletics} // one skill short of the two the class grants
	if res, err := creator.Run(ctx, cmd); err != nil {
		t.Fatal(err)
	} else if _, rejected := res.(character.CreateCharacterRejected); !rejected {
		t.Errorf("a missing class skill = %#v, want a rejection", res)
	}
}
