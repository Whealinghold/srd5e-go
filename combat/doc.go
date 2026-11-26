// Package combat holds the combat rules of System Reference Document 5.2.1:
// hit points and damage, conditions, attacks, initiative and turns, movement,
// spell effects on combatants and the encounter aggregate.
//
// Inputs are combatant state built from the [character] and [content]
// published languages by id and value, host-supplied rollers and content
// sources, and the rule options a host passes in. Outputs are typed results
// and sealed event families, never panics for bad input.
//
// Handoff types (this package grows with each Phase 4 step): [CombatantID]
// (made with [NewCombatantID]), [CombatantKind] with its two variants
// [PlayerCharacterCombatant] and [MonsterCombatant], [CombatantRef], the sealed
// [Event] base of every combat event, and [RuleOptions] with [PlayingOnAGrid]
// ([NoGrid], [SquareGrid]). An option is a value in a command, never a global
// setting.
//
// Hit points and damage: [HitPointStatus] (made with [NewHitPointStatus] or
// [RestoreHitPointStatus]) holds [HitPoints], a [LifeState] and
// [DamageAffinities]; [Damage] goes through the pipeline of the SRD. The rules
// are pure methods of the status returning a [HitPointTransition]
// ([HitPointTransitionApplied] or [HitPointTransitionRefused]) and a family of
// [HitPointEvent] values; the use cases [ApplyDamage], [RestoreHitPoints],
// [GainTemporaryHitPoints], [EndTemporaryHitPoints], [MakeDeathSavingThrow],
// [StabilizeCreature], [RollStableRecoveryDelay], [RecoverStableCreature],
// [ReduceHitPointMaximum] and [RecordExhaustionDeath] each take a command and
// return a closed result. Options are [WithTelemetry] and [WithClock]; the only
// port is the one-method [Roller].
//
// Conditions: [Condition] is the closed set of the fifteen conditions, one value
// type per content.ConditionKind ([Blinded], [Charmed] with a [Charmer],
// [ExhaustionCondition] with an [ExhaustionLevel], [Frightened], [Grappled] and
// so on). A [ConditionSet] holds a creature's explicit conditions and immunities
// and derives the effective ones: implied kinds are added, an immune condition is
// suppressed and not deleted. Its pure methods Apply, Remove and ReduceExhaustion
// return the closed results [ApplyConditionToSetResult],
// [RemoveConditionFromSetResult] and [ReduceExhaustionOnSetResult] with
// [ConditionEvent] values ([ConditionApplied], [ConditionRemoved],
// [ConditionLevelReduced]). Each condition's rules are data in a
// [ConditionDefinition] of [ConditionEffect] values, read by the pure
// [ConditionEffects] queries (attack rolls, saving throws, Speed, targets and so
// on). The use cases [ApplyConditionToSet], [RemoveConditionFromSet] and
// [ReduceExhaustionOnSet] wrap the three operations.
//
// Grappling: a [Grapple] is one creature holding another with a [GraspingPart]
// ([GraspingHand] or [GraspingBodyPart]); the immutable [Grapples] value is the host's
// record of who holds whom, with one grapple per part and per target. Its pure
// transitions Begin, Release, Escape, Review and ConditionEnded return a
// [GrappleTransition]; a grapple that ended comes back as a [GrappleEnding] that
// the host passes to [GrappleRules].EndOnSet, which removes Grappled and emits the
// [GrappleEvent] values ([GrappleStarted], [GrappleEnded], [GrappleRecordDropped]).
// [GrappleRules] also holds the DC, the size and free-hand check, the hold rule and
// the dragging cost. The use cases [BeginGrapple], [ReleaseGrapple],
// [EscapeGrappleFromRecord], [ReviewGrapples] and [DropGrappleRecord] wrap them;
// the escape check uses the dice port and the pure d20 functions.
//
// Hazards and objects: [HazardKind] names the five hazards. Burning
// ([BurningState]), Falling ([FallDistance], [FallSurface], [FallLandingCheck]),
// Dehydration and Malnutrition ([WaterNeedsPerDay], [FoodNeedsPerDay],
// [WaterAmount], [FoodAmount], [FoodSavingThrow]) and Suffocation ([BreathState],
// [BreathChange], [BreathCapacity]) are used through [RollBurningDamage],
// [ExtinguishBurning], [ResolveFall], [LandAfterFall], [ResolveDehydrationDayEnd],
// [ResolveMalnutritionDayEnd], [AdvanceBreath], [ResolveSuffocationTurnEnd] and
// [RestoreBreathing]. Damage hazards roll and return a [Damage] for the host to pass
// to [ApplyDamage]; Exhaustion hazards change a [ConditionSet] only through
// ConditionSet.Apply and ConditionSet.ReduceExhaustion. The host keeps a
// [HazardRecord] beside the set, and [RemoveExhaustionLevels] removes Exhaustion
// without breaking the lock Dehydration, Malnutrition and contagion levels put on it.
// An [ObjectStatus] holds an object's size, Armor Class ([ObjectArmorClassTable]),
// Hit Points ([ObjectHitPointsTable], [RollObjectHitPoints]), [DamageThreshold],
// [ObjectAffinities] and ability scores; [ApplyDamageToObject] damages or breaks it
// ([ObjectDamageInstance], [ObjectBrokenAutomatically]) and a [CompositeObject]
// models a building or a vehicle as sections. [DamageThresholdGate] serves a
// creature with a threshold too. The [EnvironmentEvent] values are [BurningEnded],
// [DehydrationRecovered], [MalnutritionRecovered], [BreathHeldStarted],
// [RanOutOfBreath], [BeganChoking], [BreathRestored], [ObjectDamaged],
// [ObjectDamageDeflected], [ObjectDestroyed] and [ObjectCollapsed].
//
// Sight, hiding, searching and studying are pure functions over facts the host
// supplies, with no use case and no signal. [LightRules] gives the effect of
// obscurement on a check made to see; [SpecialSenseRules] reads the ranges in a
// [SenseRanges] value (Darkvision, Blindsight, Tremorsense, Truesight) and gives the
// effective obscurement; [SightRules].CanSee decides a [SightScene] into a
// [SightRelation]. [HideRules] assesses a [HideScene] into [HideAvailable] or
// [HideBlocked], resolves a rolled or forced-failure check, and keeps a
// [HidingStatus] with [HiddenState]; hiding ends with the [HidingEvent] values
// ([CreatureHid], [HidingEnded]). [SearchRules] and [StudyRules] name the skills
// the Search and Study tables suggest.
//
// Attacks: [ResolveAttack] resolves one attack roll and its damage, and [ResolveUnarmedStrike]
// one Unarmed Strike with its [UnarmedStrikeDamageOption], [UnarmedStrikeGrappleOption] or
// [UnarmedStrikeShoveOption]. The host states the facts in a [ResolveAttackCommand]
// (built from [ResolveAttackParams]) or a [ResolveUnarmedStrikeCommand]: an [AttackForm],
// the [Cover] sources, an [AttackRange] or [Reach], [NearbyEnemy] values, an
// [AttackSituation], a [DamageSpec] and the consequences of conditions as circumstances and
// [AutomaticCritical]. The pure services [CoverResolver], [CircumstanceCollector] (an
// [AttackPlan]) and [AttackDamageComposer] decide cover, what cannot be done, Advantage and
// Disadvantage, and the damage; the roll itself is the pure d20 functions, never the d20 use
// case. The results are [ResolveAttackResult] ([AttackResolved], [AttackAutomaticallyMissed],
// [AttackNotPossible]) and [ResolveUnarmedStrikeResult] ([UnarmedStrikeDamageResolved],
// [UnarmedStrikeGrappleResolved], [UnarmedStrikeShoveResolved], [UnarmedStrikeNotPossible]);
// a miss and a refusal are results, not errors, and nothing is applied: the host passes the
// [Damage] on to [ApplyDamage] and a succeeded Grapple to [BeginGrapple].
//
// The Attack action and the Help action are pure values with no use case and no signal.
// An [AttackActionProgress] (made with [StartAttackActionProgress]) counts the attacks of
// one Attack action against an [AttackAllowance] whose counts the host passes in as plain
// integers: [AttackActionProgress.RecordAttack] takes an [AttackMeans] and
// [AttackActionProgress.RecordReplacement] an [AttackReplacement], and
// [AttackActionProgress.HandleWeapon] records one [AttackActionWeaponChange] in the slot of
// an attack. They return the closed outcomes [AttackRecordOutcome] and [WeaponChangeOutcome]
// with [AttackActionEvent] values ([AttackMadeInAttackAction], [AttackReplacedInAttackAction],
// [WeaponChangedInAttackAction]). [HelpBenefits] holds the pending [AbilityCheckHelp] and
// [AttackRollHelp] benefits and gives, uses, expires and drops them, returning
// [AbilityCheckHelpOutcome], [AttackRollHelpOutcome], [AbilityCheckHelpUseOutcome],
// [AttackRollHelpUseOutcome], [HelpBenefitsExpiry] and [HelpBenefitsDrop] with [HelpEvent]
// values ([HelpGiven], [HelpBenefitUsed], [HelpBenefitExpired], [HelpBenefitDropped]); a used
// benefit yields one modifier.Advantage circumstance and the roll mode is left to the kernel.
//
// Deployed and restraining gear (SRD 5.2.1, pp. 96-99) is [DeployedGears], [EntryLedger] and
// [GearHolds], immutable values the host stores, with the rules [AreaEntryRules], [NetRules],
// [BindingRules] and [HoldRules]. [DeployGear], [MaintainDeployedGear], [ResolveGearEntry],
// [ThrowNet], [BindCreature] and [ReleaseFromHold] return consequences (damage, conditions,
// holds) and apply none; the events are the closed [GearEvent] family.
//
// Monster Swallow and Engulf (SRD 5.2.1, the Swallow, Engulf and Whelm actions) is the state
// machine of a creature held inside another. The host transcribes a stat block into
// [ContainmentTerms] (made with [NewContainmentTerms] from a [ContainmentTermsInput]) and stores
// the immutable [Containments], beside its [Grapples]; [ContainmentRules] answers cover,
// restrictions, movement and locked actions and works out a release on a [ConditionSet]. The use
// cases [EnterContainer], [ExitContainer] and [ResolveRegurgitation] roll through the [Roller]
// port and the pure d20 functions, and [AdvanceContainerTurn] and [ReleaseOccupants] roll nothing.
// They report damage as [ContainmentDamageDue] for the host to roll and pass to [ApplyDamage],
// refusals as [ContainmentRejection] and unusable input as [ContainmentProblem]; the events are
// the closed [ContainmentEvent] family ([OccupantContained], [OccupantReleased],
// [ContainerTrapped]).
//
// Initiative, turns, actions and movement: a [TurnTracker] holds the Initiative order
// ([TurnOrder] of [InitiativeStanding] values), the round, the active combatant and its
// [TurnResources], the Reactions spent, those dodging, the readied actions ([ReadiedAction]),
// the profiles ([CombatantProfile] with [MovementSpeeds]) and the turn-limited feat uses. It has
// no exported constructor: [EstablishInitiativeOrder] makes one from the counts that
// [RollInitiative] produced ([InitiativeCount], [InitiativeParticipant], [InitiativeBreakdown]),
// applying [TieBreak] values or reporting [InitiativeTiesUnresolved] and [InitiativeTieBreakRejected].
// Its pure operations Advance, UpdateMovementSpeeds, TakeAction, TakeBonusAction, TakeReaction,
// Move, InteractWithObject, ChangePosture, GrantAdditionalAction and RecordFeatUse return a
// [TrackerStep] ([TrackerStepApplied] or [TrackerStepRejected]) with [TurnEvent] values, and the use
// cases [AdvanceTrackerTurn], [UpdateMovementSpeeds], [TakeAction], [TakeBonusAction], [TakeReaction],
// [Move], [InteractWithObject], [ChangePosture], [GrantAdditionalAction], [RecordFeatUse] and
// [SwapInitiative] wrap them with a command and a closed result each. A refusal is a [TurnRejection];
// choices are the closed [ActionChoice], [GrantedAction], [BonusActionEntitlement], [ReactionChoice] and
// [PostureChange] sets, and a move is a list of [MoveSegment] values judged against the speeds.
//
// Item use (SRD 5.2.1, "Adventuring Gear", pp. 94-99): [ResolveThrownItem] throws Acid, Alchemist's
// Fire, Holy Water, Oil or a Net ([ThrownItemKind], with its [ThrownItemFacts]) at a [ThrownTarget]
// and has the target make a Dexterity saving throw against [ItemSaveDCOf]; [UsePotionOfHealing]
// drinks or administers a Potion of Healing ([PotionUse]). Their results ([ThrownItemResolved],
// [ThrownItemNotPossible], [PotionOfHealingUsed], [PotionOfHealingNotPossible]) hand the host the
// consequence as data (a [ThrownEffect], an [ItemActionCost] to spend, an [ItemConsumption] to
// decrement) and apply none. The pure rules are [ThrownItemRules], [ItemUseRules], [NetRestraint],
// [OilCoat], [BasicPoisonRules] with [BasicPoisonCoating], [AntitoxinRules] with [AntitoxinEffect]
// and [HealersKit]; the clocks of the timed items are durations the host passes in. A failure of
// a constructor or rule is an [ItemUseError]; the events are the closed [ItemUseEvent] family.
//
// Magical effects (SRD 5.2.1, "Magical Effect" and the Glossary phenomena, pp. 177-190), all pure
// and silent: an [EffectSource] with [EffectSource.IsMagical], the [OngoingEffects] list of
// [OngoingEffect] values, the origin rules of an area ([ConeWidthAt], [OverridableAreaShape],
// [OriginInclusionDecision], [EmanationEffectTiming], [AreaOrigin]) and [AreaMembership] over
// host facts ([AreaLocationFacts]); [CurseRemoval] with its [CurseRemovalMeans]; the spells named
// by Possession; [IllusionTraits]; [ShapeShifting] with its [ShapeShiftState] and [CarryOverRule];
// [Teleportation] over a [TeleportCommand]; and [SimultaneousEffects]. Each operation returns its
// own closed result and the events are the closed [MagicalEffectEvent] family. The engine models
// no map: positions, containment and Cover per line are facts the host passes in.
//
// Shape-changing spells (SRD 5.2.1, "Polymorph" p. 153, "True Polymorph" p. 170, "Shapechange" p. 161,
// "Animal Shapes" p. 108), pure and silent, with no use case yet: the stateless [ShapeShiftSpells]
// (not [ShapeShifting], which carries the form over) judges a form ([ShapeShiftSpells.Eligibility]
// over a [FormQuery], from [FormFacts], an [AssumedForm] and a [TargetRating]), begins a
// transformation ([ShapeShiftSpells.Begin]), changes its form, reads the Temporary Hit Points of the
// form ([ShapeShiftSpells.Assess]) and ends it ([ShapeShiftSpells.End],
// [ShapeShiftSpells.ElapseFullDuration]). The state is one [ActiveTransformation] per target; each
// operation returns its own closed result, the host passes the effect source and [CarryOverAll] of a
// begin or end to [ShapeShifting], and the events are the closed [ShapeShiftEvent] family.
//
// Movement beyond walking (SRD 5.2.1, pp. 178-189), all pure and silent: [LimitSpeedsByLoad]
// caps the walking Speed of a [character.LoadSpeedLimited] load and [ApplyArmorSpeedPenalty]
// applies the armor Speed penalty, both to [MovementSpeeds]; [AssessJump] judges a
// [JumpAttempt] (a [JumpKind] and [JumpApproach], with [JumpMaximumFeet]) as a
// [JumpAssessment], [JumpLandingFor] and [ObstacleForLongJump] name the checks a landing or a
// low obstacle needs, and [HighJumpReachFeet] gives the reach; [RequirementFor] describes the
// four [MovementCheckRequirement] values without rolling them; [AssessFlight] says from
// [FlightFacts] whether a flier falls ([FlightFallCauses]) unless it can hover; and
// [BurrowPermissionFor] says whether a burrow through a [BurrowMedium] is allowed.
//
// Readying a spell (SRD 5.2.1, "Ready [Action]", pp. 186-187), pure and silent: [ReadySpell]
// holds a Magic action spell as a [ReadiedSpell] with a [ConcentrationHold] request,
// [ReleaseHeldSpell] releases it with the caster's readied-action [ReactionTaken], and
// [DissipateHeldSpell] drops it for a [DissipationCause]. Each returns its own closed
// result and the events are the closed [HeldSpellEvent] family. Concentration itself is
// the host's, through the spellcasting rules.
//
// Spell effects (SRD 5.2.1, "Effects", "Targets", "Saving Throws", "Attack Rolls", p. 106):
// [ResolveSpellEffects] interprets the effect model of a [content.Spell] for a
// [character.CastReport] and the [SpellTarget] values the host chose, and returns the closed
// [ResolveSpellEffectsResult]: a plan of [PlannedDamage], [PlannedAmount] and [ConditionToApply]
// per [TargetEffects] with the evidence of every roll ([ResolvedPart], [ResolvedInstance]),
// [SpellEffectsTextOnly] for a spell without structured effects, or a [SpellEffectsRejection].
// It applies nothing and tracks no duration; the host passes the plan to the hit point and
// condition use cases.
//
// Returning to life (SRD 5.2.1, "Dead", p. 180), pure and silent: [Revive] over a
// [ReviveCreatureCommand] returns a [ReviveCreatureResult] ([ReviveCreatureApplied],
// [ReviveCreatureNotDead], [ReviveCreatureRefused], [ReviveCreatureHitPointsBelowOne] or
// [ReviveCreatureMaximumBelowOne]). The spirit's answer is a [RevivalConsent]; the
// host reports which conditions and effects have elapsed, since the package keeps no
// clock. The event is [CreatureReturnedToLife] of the closed [RevivalEvent] family.
//
// Revival and restoration spells (SRD 5.2.1, "Revivify" p. 159, "Raise Dead" p. 157, "Resurrection"
// pp. 158-159, "True Resurrection" p. 170, "Aid" p. 107, "Heroes' Feast" p. 140, "Lesser Restoration"
// p. 144, "Greater Restoration" p. 137, "Heal" p. 139, "Power Word Heal" p. 154, "Mass Heal" p. 147, "Death
// Ward" p. 122), pure and silent, with no use case yet. [ReviveBySpell] judges a [RevivalSpell] against the
// [DeathCircumstances] and composes [Revive], giving a [ReviveOrdeal] and a [CasterStrain] that the host ages
// with AfterRest. [IncreaseMaximum], [EndMaximumIncrease] and [RestoreMaximum] keep the Hit Point maximum
// through a [HitPointMaximumIncreases] ledger; [ApplyLesserRestoration], [ApplyGreaterRestoration] (one
// [GreaterRestorationChoice]) and [ApplyRestorativeHealing] (with a [MassHealAllotment]) end conditions and
// restore Hit Points; [ApplyDamageWithDeathWard] and [NegateInstantKill] spend a [DeathWard]. Each returns a
// closed result; the events are [HitPointMaximumIncreased], [HitPointMaximumIncreaseEnded] and
// [DeathWardSpent] of the closed [RestorationEvent] family, beside those of the operations they compose.
//
// Stat-block actions (SRD 5.2.1, "Parts of a Stat Block", pp. 254-257) say what a stat-block
// entry does when a monster uses it. [UseStatBlock] uses one part as an action, a Bonus Action,
// a Reaction or a Legendary Action by its [StatBlockActionCost], spends its usage limit through
// the [LimitedUseState] and returns the [FeatureActionChoice], [BonusActionGrant] or
// [FeatureReaction] to record, or a closed [StatBlockUseRefusal]. [StartMultiattack] returns a
// [MultiattackProgress], a ceiling of attacks, additional uses and replacements spent in any
// order. [PlanStatBlockAttack] and [AppliedClausesForAttack] turn an Attack Notation into the
// pieces [ResolveAttack] takes and the clauses a hit or a miss applies; [PlanStatBlockSaveEffect]
// and [ResolveStatBlockSaveEffect] do the same for a saving throw effect, with the half-damage
// adjustment, and keep the 24-hour immunity in [SaveImmunities]. All of it is pure and silent; the
// events are the closed [StatBlockActionEvent] family. [RollStatBlockDamage] is the one use case: it
// rolls the printed number or dice of each damage instance, once for the whole use.
//
// Monster traits (SRD 5.2.1, "Traits", p. 256), pure and silent: [RecogniseTraits] turns the twelve
// recurring traits of a stat block into [MonsterTrait] values by name and text template and reports
// the parts that fail a check as [UnrecognisedTrait]. [UseLegendaryResistance] turns a failed
// saving throw into a success through the [LimitedUseState]; [MagicResistanceCircumstances],
// [PackTacticsCircumstances] and [SunlightPenaltyCircumstances] return roll circumstances;
// [AssessUndeadFortitude] and [SettleUndeadFortitude] keep a creature at 1 Hit Point; a
// [RegenerationState] heals at the start of a turn, is suppressed by its Damage Types and defers a
// death ([DeferRegenerationDeath]); [TriggerDeathBurst], [CheckSwarmChange],
// [FlybyExemptsFromOpportunityAttack], [ApplySiegeMonster], [AssessParry] and [RespondToParry]
// finish the set. The events are the closed [MonsterTraitEvent] family.
//
// Monster spellcasting (SRD 5.2.1, "Spellcasting", p. 257), pure and silent: [MonsterSpellcastingRules] casts a
// spell for a [MonsterSpellcaster] (its trait and the Modifier of each score) and a
// [MonsterSpellcastingState] (Concentration, a casting of 1 minute or more and the entry that began it) through
// the [LimitedUseState]. A [MonsterListCast] names an entry of the trait and a [MonsterPartCast] a stat-block
// part with its own [PartSpellcasting]; neither can raise the level. [MonsterSpellcastingRules.Cast] returns the
// closed [MonsterCastOutcome]: [MonsterCastPerformed] with the [MonsterCastReport] (the printed save DC and attack
// bonus as printed, the ability and its Modifier, a [TargetLimit]), [MonsterCastStarted] for a casting of 1
// minute or more, or [MonsterCastRefused] with a [MonsterCastRefusal]. [MonsterSpellcastingRules.CompleteCasting],
// [MonsterSpellcastingRules.EndMagicActionTurn] and [MonsterSpellcastingRules.BreakConcentration] finish or end a
// casting. [MonsterSpellRestrictions] and [ParseComponentWaiver] read an entry's note and a components statement.
// The events are the closed [MonsterSpellcastingEvent] family.
//
// The encounter (SRD 5.2.1, "Combat", p. 13): an [Encounter] is the aggregate root of one combat, built by
// Encounter.Start from the [InitiativeOrderEstablished] a host got from [EstablishInitiativeOrder] and its
// [Combatant] values (a [CombatantRef], a [Side] of [SideA] or [SideB], a [HitPointStatus], a [ConditionSet] and the
// knocked-out and per-turn Death Saving Throw markers). It is opaque: a copy goes in and out, the zero value is
// invalid, and a [TurnTracker] is stored as given. Its transitions (AdvanceTurn, ApplyTurnChange, ApplyDamage,
// ApplyHealing, ApplyCondition, RemoveCondition, ReduceExhaustion, ApplyDeathSavingThrow, PrecheckDeathSavingThrow,
// Stabilize, RecoverStable, GainTemporaryHitPoints, ReduceHitPointMaximum, CheckGrapple, ReleaseGrapple and
// EscapeGrapple) compose the rules of the hit point, conditions, grapple and tracker files and return the new
// encounter with the [Event] values in the order they happened, or an [EncounterRejection]; an invalid input is an
// error. [EncounterID] is made with [ParseEncounterID]. The closed [EncounterEvent] family is [EncounterStarted],
// [EncounterEnded], [DyingCombatantTurnStarted] and the four transport envelopes [TurnEventOccurred],
// [HitPointEventOccurred], [ConditionEventOccurred] and [GrappleEventOccurred], each wrapping a sibling event
// unchanged. The use cases [StartEncounter], [AdvanceEncounterTurn], [ApplyDamageToCombatant],
// [ApplyHealingToCombatant], [ApplyConditionToCombatant], [RemoveConditionFromCombatant],
// [ReduceCombatantExhaustion], [ApplyDeathSavingThrowToCombatant], [StabilizeCombatant], [RecoverStableCombatant],
// [GiveCombatantTemporaryHitPoints], [ReduceCombatantHitPointMaximum], [EndEncounter], [CheckCombatantGrapple],
// [ReleaseCombatantGrapple] and [EscapeCombatantGrapple] each take a command and return their own closed result,
// Applied or Rejected (the start: Applied or Invalid). The encounter calls no port and rolls no dice.
//
// Active effects (SRD 5.2.1, "Duration", p. 106, "Combining Spell Effects", p. 106, "Concentration", p. 179): the host
// holds an [ActiveEffects] registry of [ActiveEffect] values and passes it in; nothing here applies damage or a condition.
// An effect is on a creature or a place ([CreatureEffectSubject], [PlaceEffectSubject]), comes from a spell or another source
// ([SpellEffectOrigin], [OtherEffectOrigin]), may be bound to its caster's Concentration ([ConcentrationBinding]) and follows an
// [EffectTemplate]: [EffectStage]s with an [EffectDuration] ([RoundsInTimeSpan], [EffectDurationFromSpell] and
// [EffectDurationFromCondition] build one from content), [EffectTrigger]s of [RepeatSave], [DealDamage], [GrowStoredDamage] and
// [DealStoredDamage] actions, a gate and onEnd actions. [EffectCombination] answers which effects apply. The use cases
// [StartActiveEffect], [AdvanceActiveEffects] (one [EffectClockTick] at a time, see [ClockTickFromTurnEvent]), [EndActiveEffects]
// (an [EndingCause]) and [ResolveEffectDamage] each take a command and return their own closed result with the registry, the
// [ActiveEffectEvent]s in order and plans for the host ([EffectConditionPlan], [EffectDamagePlan]), or an [ActiveEffectRejection].
// Only [AdvanceActiveEffects] and [ResolveEffectDamage] roll dice, through the one-method [Roller].
//
// The use cases record their telemetry through one unexported instrumenter:
// the shared signals srd5e.combat.operations, srd5e.combat.operation_duration,
// srd5e.combat.errors and srd5e.combat.events. A refusal is a result that is
// not an error status. The domain files are silent. Test support for the event
// family is in the combattest package.
//
// [character]: github.com/Whealinghold/srd5e-go/character
// [content]: github.com/Whealinghold/srd5e-go/content// Persistent spell zones: [PersistentZone] (made with [CreateZone] from a
// [CreateZoneCommand]) holds a zone's facts, designations, ledger and end. The
// operations [AssessZoneTrigger], [ApplyOptionalZoneEffect],
// [RecordZoneCastApplication], [MoveZone], [ZoneStartOfTurn], [ZoneRoundStarted],
// [RecordZoneDamage], [EndZone] and [ZoneMovementEffectsFor] are pure: each takes
// a zone and returns the new zone with [ZoneEvent] values, and the effects the
// host resolves are [ZoneEffectDue] values. The host reports events through
// [ZoneTriggerEvent] values.
//
// Spell outcome tables: the printed tables of Teleport ([TeleportKindFor],
// [TeleportOutcome]), Confusion ([ConfusionRowFor], [ConfusionTurn]), Blink
// ([BlinkVanishesOn], [BlinkReturnFor]), Reincarnate ([ReincarnationSpeciesFor],
// [ReincarnationOutcome]), Prismatic Spray ([PrismaticRayFor],
// [PrismaticStrikeFor], [IndigoSaveTrack], [ResolveVioletFollowUp]) and Prismatic
// Wall ([PrismaticWallState], [PrismaticLayerDestroyer]) are pure values. The use
// case [ResolveSpellOutcomeTable] takes a [ResolveSpellOutcomeTableCommand] and
// returns a closed [ResolveSpellOutcomeTableResult]; it applies nothing.
//
// Combat action resolution (SRD 5.2.1, "Combat", pp. 13-18; steps 10a and 10b of the Phase 4 plan): the eight use cases that wire the
// pure parts of this package into whole actions, state in and state out. Each takes an [Encounter] (and the host's
// [ActionLedger] where it says so) in its command and returns its own closed result, Completed with the new encounter,
// ledger and the events in order, Rejected with a [ResolutionRejection] or Invalid with a [ResolutionProblem]; a rejected or
// invalid call carries the state that came in and no event. [PerformAttack] (a [PerformAttackCommand] from [PerformAttackParams]:
// an [AttackCost], an [AttackScene], the base [ResolveAttackCommand], a [KnockOutChoice]) and [PerformUnarmedStrike] spend the
// cost, derive the inputs of the attacker's and target's conditions ([AttackConditionInputs]), roll, hand the damage to the
// encounter ([DamageHandoff]) and settle the consequences ([ConsequenceRules]: [MovementSpeedsReviewBecameDue] and
// [GrappleCheckBecameDue]); a Grapple or Shove applies its condition ([UnarmedEffects], [UnarmedEffectOutcome]).
// [TakeCombatantAction] carries out a [TurnRequest] ([SpendAction], [SpendBonusAction], [SpendReaction], [SpendInteraction],
// [TakeHelpAction], [HandleWeapon], [ReplaceAttack]); [MoveCombatant] a [MovementRequest] ([MoveAlong], [DropProne], [StandUp],
// checked by [MovementChecks]); [AdvanceCombatTurn] ends a turn and starts the next, applies Prone from a shared space, ends the
// Attack action and expires Help benefits; [RollCombatantDeathSavingThrow], [AttemptToStabilizeCombatant] and
// [DealDamageToCombatant] roll or apply what their names say. [ResolutionPrecedence] fixes which stop is reported when several
// apply. The events are [AttackPerformed], [UnarmedStrikePerformed], [MovementSpeedsReviewBecameDue] and
// [GrappleCheckBecameDue] ([ResolutionEvent]) besides the owners' own. The attack, Death Saving Throw and stabilizing paths roll
// through the one-method [Roller]; the others roll nothing.
//
// Step 10b adds the spell and Concentration paths. A [ConcentrationInput] ([NoConcentrationTracking] or [TracksConcentration],
// with a [ConcentrationDamageRule]) names the Concentration of the damaged combatant: after the damage and its consequences a
// use case asks [ConsequenceRules.Concentration], rolls the Constitution saving throw when it is due and returns a
// [ConcentrationCheckOutcome] ([ConcentrationNotChecked], [ConcentrationHeld], [ConcentrationCheckFailed],
// [ConcentrationEndedWithoutSave]), the new SpellcastingState of the target and a [ConcentrationChecked] event. [PerformAttack] and
// [PerformUnarmedStrike] also take the attacker's [GearHolds] and the target's [ParryChoice] ([NoParry], [ParryIfAble]) and return
// a [ParryOutcome]. [TakeCombatantAction] gains [UseStatBlockPart], [BeginMultiattack], [RecordMultiattackUse] and
// [RecordMultiattackReplacement], an attack may cost [RecordsMultiattack], and the [ActionLedger] holds a [MultiattackState];
// a refusal of the stat block rules is a [StatBlockRejection]. [ResolveStatBlockSave] resolves the saving throw effect of a stat
// block part on one target. The use cases that wrap the pure services of the domain-only specs take no encounter and no
// ledger and return the owner's own closed result: [ReadyCombatantSpell], [ReleaseCombatantSpell], [DissipateCombatantSpell],
// [ChangeTelepathicContact] (a [TelepathyRequest]), [CreatePersistentZone], [AssessPersistentZoneTrigger], [MovePersistentZone]
// and [EndPersistentZone] (a [ZoneTriggerRequest], [ZoneMoveRequest] or [ZoneEndRequest]), [BeginShapeShifting],
// [ChangeShapeShiftForm], [AssessShapeShift], [EndShapeShifting], [ReviveCreatureBySpell], [IncreaseHitPointMaximum],
// [EndHitPointMaximumIncrease], [RestoreHitPointMaximum], [CastLesserRestoration], [CastGreaterRestoration],
// [CastRestorativeHealing] and [GuardDamageWithDeathWard] (a [DeathWardRequest]), and the five of the magical-effects rules, [RemoveCombatantCurse],
// [ShiftCombatantShape] (a [ShapeShiftingRequest]), [TeleportCombatant], [ResolveEffectOrder] and [AssessAreaLocation] (an [AreaLocationRequest]), with [DispelOngoingSpells], [CounterCastSpell]
// and [AssessAntimagicActivity] of their own steps. The state they return is the host's. Of these only [DispelOngoingSpells]
// and [CounterCastSpell] roll a die; a call takes at most 4096 events and 1000 steps, and over
// either it is Invalid with [LimitExceededProblem].
//
// Combatant roster changes (SRD 5.2.1, "Initiative Order", p. 13; "Ending Combat", p. 14; "Controlling a Mount", p. 16): after combat
// started the host adds, removes and repositions a combatant of an [Encounter] with [AddCombatantToEncounter],
// [RemoveCombatantFromEncounter] and [RepositionCombatantInEncounter], each taking its command ([AddCombatantToEncounterCommand],
// [RemoveCombatantFromEncounterCommand], [RepositionCombatantInEncounterCommand]) and returning its own closed result
// ([AddCombatantToEncounterResult] with [AddCombatantToEncounterApplied] or [AddCombatantToEncounterRejected];
// [RemoveCombatantFromEncounterResult] with [RemoveCombatantFromEncounterApplied] or [RemoveCombatantFromEncounterRejected];
// [RepositionCombatantInEncounterResult] with [RepositionCombatantInEncounterApplied] or [RepositionCombatantInEncounterRejected])
// holding a [RosterRejection] for a refusal. A [RosterPosition] says where the combatant takes its place:
// [RosterAfterCombatant] (its anchor's count, directly after it, as a summoned creature) or [RosterAtInitiative] (a count the host
// rolled and a [TiePlacement], the host's decision on a tie). A [JoinReason] and a [RemovalCause] say why; they change no rule. The
// rejections are [RosterEncounterEndedRejection], [RosterUnknownCombatantRejection], [RosterDuplicateCombatantRejection],
// [RosterCombatantIsActiveRejection] (the host ends the active turn first with [AdvanceCombatTurn]), [RosterWouldEmptySideRejection]
// (the encounter does not end by itself) and [RosterLimitExceededRejection] (256 combatants, 4096 events). A removal first ends every
// grapple that involves the leaver, as out of range, and drops the leaver's pending Help benefits from the [ActionLedger]; it touches
// no effect, readied action of another or containment. The events are the closed [RosterEvent] family ([CombatantJoined],
// [CombatantRemoved], [CombatantRepositioned]). The pure [TurnTracker] operations are InsertCombatant, RemoveCombatant and
// MoveCombatant, returning a [RosterStep] ([RosterStepApplied] or [RosterStepRejected]) and no turn event; the host holding a tracker from
// before a roster change is refused by [Encounter.ApplyTurnChange] until it takes the tracker from the new encounter. The roster use
// cases roll no die and call no port.
//
// Summoned and companion creatures (SRD 5.2.1, "Animate Objects", p. 108; "Animate Dead", p. 108; "Create Undead", p. 121; "Find Familiar" and
// "Find Steed", p. 130; "Giant Insect", p. 135; "Simulacrum", p. 162; "Summon Dragon", p. 166): the host stores a [SummonedRoster] beside the
// [Encounter], as it does a grapple record. [SummonCreature] takes a [SummonCreatureCommand] (the closed [SummonCreatureResult] is its outcome) of one [SummonRequest] (Animate Objects, Giant
// Insect, Summon Dragon, Find Steed, Animate Dead, Create Undead, Find Familiar, Simulacrum or a reassert of control), a [SummonerSpellFacts] and
// a [SummonScene] (combat, night, space, the active combatant and the Initiative of each creature that rolls its own), reads the stat blocks
// [MonsterIDsNeeded] names through a [ContentSource] and returns [SummonCreatureApplied] (the new roster, a [SummonedCreatureSpec] for each
// creature, the [SummonedRosterAction] instructions and the [SummonEvent] events), [SummonCreatureRejected] with a [SummonRejection] or
// [SummonCreatureContentMissing]. [ApplySummonedChange] takes an [ApplySummonedChangeCommand] holding a [SummonedChange] (an order, a report of 0 Hit Points, of a spell that ended, of the
// summoner's death, of a Long Rest, of time, of a turn, a dismissal, a recall, a sight link, a touch, a repair, Life Bond or a familiar's action)
// and returns an [ApplySummonedChangeResult]: [ApplySummonedChangeApplied], [ApplySummonedChangeRejected] or [ApplySummonedChangeUnchanged]. Both use cases wrap pure methods of
// the roster, which return a [SummonedTransition] ([SummonedApplied], [SummonedRejected], [SummonedUnchanged]); no use case calls another and none
// calls a roster operation: a [SummonedRosterJoin] or [SummonedRosterLeave] is data for the host to pass to AddCombatantToEncounter or
// RemoveCombatantFromEncounter, and a departing active combatant is pending until the host has ended its turn (plan P10: 32 creatures a summoner,
// link cycles refused). A [SummonedLink] holds the [SummonKind], the [PlacementRule], the [ControlWindow] of an undead creature, its
// [SummonedOrder] (the plan on its turn is a [SummonedTurnPlan]), a familiar's [FamiliarState] with its [FamiliarOrigin], and for a simulacrum
// the creatures it is friendly to. The numbers that depend on the cast level are in a [SummonedStatBlock] with its [SummonedAttack] entries,
// built by NewAnimatedObjectStatBlock, NewGiantInsectStatBlock, NewDraconicSpiritStatBlock, NewOtherworldlySteedStatBlock and
// NewSimulacrumStatBlock (from a [SimulacrumOriginal]); a [SharedResistanceGrant] records the Resistance a Draconic Spirit shares. The
// [CommandChannel] of each kind, [BonusActionGrantFor], [SteedModeFor] with its [SteedMode], a [SteedType], an [InsectForm], a
// [FamiliarAttackGrant] and [OverflowDamage] serve the host. The events are the closed [SummonEvent] family: [SummonedCreatureJoined],
// [SummonedCreatureCommanded], [SummonedCreatureDeparted] (with a [SummonEndCause]), [SharedResistanceEnded], [SummonControlReasserted],
// [SummonControlLapsed], [FamiliarFormChanged], [FamiliarDismissedTemporarily], [FamiliarReappeared], [FamiliarSightLinkStarted],
// [FamiliarSightLinkEnded] and [SimulacrumRepaired]. Refusals include [SummonCreatureIDInUseRejection], [LevelBelowSpellRejection],
// [TooManyCreaturesRejection], [FamiliarCannotAttackRejection] and [SummonLimitExceededRejection] (class invalid). The pure operations roll no
// die; the use cases never log.
package combat
