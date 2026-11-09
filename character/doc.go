// Package character is the character bounded context: abilities, Proficiency
// Bonus, level, derived stats, spellcasting and equipment rule maths as SRD
// 5.2.1 defines them.
//
// Inputs are the content types published by package content and the D20 Test
// types of package d20, plus the kernel packages. Outputs and handoff types
// (the character aggregate and its derived values) are added step by step and
// are what package combat will consume.
//
// Equipment rule maths (stateless services over content equipment values):
// [ArmorRules], [ArmorTrainingRules], [CarryingCapacity], [DrawnLoad],
// [DeriveBarding], [Crafting], [ScrollScribing] and [SpellcastingServices].
//
// Derived stats: the foundation value types [Level], [ClassLevels],
// [ProficiencyBonus], [HitPointMaximum], [HitDicePool] and [HitPointGain]; the
// one Bloodied predicate [IsBloodied]; the [ModifierBreakdown] that carries the
// typed terms of every derived number; the pure [DerivedStatsCalculator]; and
// the use case [CalculateDerivedStats], which looks up the initial class, armor
// and Shield through a [ContentSource] and returns a [CalculateDerivedStatsResult].
// Its handoff types are [DerivedStats], [SavingThrowModifier], [SkillModifier]
// and [PassivePerception].
//
// The aggregate: [Character] is the root, built by [New] or [Restore] and
// identified by an [Identifier]. It holds [ExperiencePoints], [ClassLevels], the
// stored [HitPointMaximum] and [HitDicePool], [HeroicInspiration] and the typed
// [FeatureStates] that class specs fill. Its transitions are pure: they return a
// [Transition] (a [TransitionChanged] with the new Character and its [Event]s, or
// a [TransitionRejected]) and never alter the receiver. The events are [Created],
// [ExperiencePointsGained], [LevelGained], [AbilityScoreChanged],
// [HitPointMaximumChanged], [HeroicInspirationGained] and
// [HeroicInspirationExpended] (the closed [RecordEvent]). The use cases are
// [LevelUpCharacter] (with a [HitPointChoice]), [AwardExperience] and
// [ChangeAbilityScore], each with a closed result and a command. [RuleOptions]
// is a value in a command, never a global setting.
//
// Rests and Hit Point Dice: [RestKind] (ShortRest, LongRest), the host-reported
// [RestReport] (built by [NewRestReport]) and the domain functions
// [CheckRestStart], [ResolveRest], [NewRestBenefits] and [HitPointsRegained].
// [SpentHitDice] is the spent state, stored under [SpentHitDiceKey]. The use
// cases are [RestTaker] ([TakeRestCommand], the closed [TakeRestResult]) and
// [HitDieSpender] ([SpendHitPointDieCommand], the closed
// [SpendHitPointDieResult]). They return the closed [RestEffect] list and the
// closed [RestEvent] family ([RestApplied], [RestInterrupted],
// [HitPointDieSpent]) as data for the host to apply; the package holds no
// current Hit Points and no clock.
//
// Heroic Inspiration: the use cases [GainHeroicInspiration] (with a
// [GainHeroicInspirationCommand] holding an [ExcessInspirationChoice], and the
// closed [GainHeroicInspirationResult]) and [ExpendHeroicInspiration] (with an
// [ExpendHeroicInspirationCommand] holding a [RerollTarget], and the closed
// [ExpendHeroicInspirationResult] carrying a [RerolledDie]). The pure
// [ResolveExcessInspiration] returns the closed [ExcessResolution]. The state and
// its events stay with the aggregate; the host applies the new face to a d20 with
// d20.Roll.ApplyReroll.
//
// Weapons and Weapon Mastery: [WeaponProficiencies] (rules [CategoryProficiency],
// [CategoryWithAnyPropertyProficiency], [AnyStatBlockWeaponProficiency]) give a
// [WeaponProficiencyStatus]; [CreateWeaponMasteryChoice],
// [ExtendWeaponMasteryChoice] and [ChangeWeaponMasteryChoice] return a closed
// [WeaponMasteryChoiceResult] and the choice is stored as a
// [WeaponMasteryChoiceState] under [WeaponMasteryChoiceKey], read back as a
// [WeaponMasteryAccess]. [PlanWeaponAttack] applies the weapon properties to one
// attack and returns a closed [WeaponAttackPlanResult] (a [WeaponAttackPlan] or a
// [WeaponAttackBlockReason]); [PlanLightExtraAttack] decides the Light extra
// attack from a [WeaponTurnState]; [OfferMastery] reports a mastery effect as a
// closed [MasteryOffer] and never applies it. The use case [ToppleSaveResolver]
// ([ResolveToppleSaveCommand], the closed [ResolveToppleSaveResult]) makes the
// Topple saving throw. These plan and offer types are the published language the
// combat context maps onto its own types.
//
// Social interaction: [Attitude] (with [MonsterDefaultAttitude] and the pure
// [AttitudeCircumstances]), [InfluenceApproach], [InfluenceComprehension],
// [InfluenceTarget] (built by [NewInfluenceTarget]), [DCBasis], and the pure
// [IsSuggested] and [DefaultDC]. The use case [InfluenceResolver] takes a closed
// [InfluenceCommand] ([InfluenceWillingCommand], [InfluenceUnwillingCommand],
// [InfluenceHesitantCommand]) with an [InfluenceAttempt] ([InfluenceRoll],
// [InfluenceAutomaticFailure]) and returns a closed [ResolveInfluenceResult]. It
// changes no attitude and spends no action; both are the host's.
//
// Exploration (pure values, no dice, port or clock): travel and light sources.
// [TravelPace] (with [TravelPaces], [TravelPaceRow] read through Row), the closed
// [TravelMode] ([LandTravel], [VehicleTravel]) and [VehicleSpeed] (built by
// [NewVehicleSpeed] or [VehicleSpeedFromMilesPerHour]) feed [TravelRules]:
// Circumstances returns the Advantage or Disadvantage a pace puts on one ability
// check, MilesPerHour and MountedHourMiles the distances. [MountedPaceBoost]
// tracks the mounts' doubled hour: Spend returns a closed
// [MountedBoostSpendResult] and RestFinished a closed [MountedBoostRestResult],
// with the [MountedPaceBoostEvent] family ([MountedPaceBoostSpent],
// [MountedPaceBoostRestored]). [LightEmission] (a [LightShape] and two distances),
// [LightSpace], [SrdLightSources] (with [HoodedLanternHood]) and [LightSourceRules]
// (LevelAt, Strongest) give the content.LightLevel a source alone gives; a
// [FuelSupply] of a [FuelKind] is burned with Burn, a closed [FuelBurnResult], and
// raises [LightFuelExhausted] (the [LightFuelEvent] family). The host reports
// distances, elapsed time, the pace chosen and finished rests. A rejected input is
// an [ExplorationError].
//
// Spellcasting: [SpellcastingClass] (eight classes, with [SpellcastingFeature] and
// [SpellcastingFocusKind]), the pure tables [PreparedLimit], [CantripLimit] and
// [MaxPreparableLevel], [SlotsFor] (a [SpellSlotPools] of [SpellSlots] and
// [PactMagicSlots], read through a [SlotPoolView]) and [Stats] and [StatsForAbility]
// ([SpellcastingStats] with [SpellSaveDC], [SpellAttackBonus] and the d20
// RollModifiers of the attack roll). A class keeps its lists as a [ClassSpellcasting]
// ([PreparedSpells], [CantripsKnown], [Spellbook], [GrantedCantrip]); the host holds a
// [SpellcastingState] beside the Character, whose transitions ExpendSlot, RegainSlot,
// RestoreSlots and Advance return closed results and the closed [SpellcastingEvent]
// family ([SpellSlotExpended], [SpellSlotsRestored], [PreparedSpellsChanged]).
// [ChoosePreparation] and [ChangePreparation] return a closed [PreparationOutcome]
// ([PreparationAccepted], [PreparationRefused] with a [PreparationRejection],
// [PreparationContentMissing]). The use cases are [CalculateSpellcasting],
// [CreateSpellcasting], [ChooseAdditionalSpells], [ChangePreparedSpells],
// [ExpendSpellSlot], [RegainSpellSlot], [RestoreSpellSlots] and [AdvanceSpellcasting],
// each with a command and a closed result. The state shapes the casting rules fill are
// [Concentration] (a [ConcentrationSubject]), [CastSource], [SpellSlotChoice] and
// [CastingInProgress]. The caller passes the spells as a map of content.Spell; a
// spell absent from it is a content-missing result.
//
// Magic item rules: [MagicItemValues], [IncorporatedItemCost], [ResolvedRarity] (a
// closed [RarityResolution]; [RarityChoice]), [MagicItemFactResolution],
// [MagicItemResilienceRules] and [BaseItemSelection] read the content magic item
// vocabulary. [MagicItemUse], [CommandWord], [ScrollReading], [SpellFromItem],
// [ItemSpellcastingAbility] and [ItemCharges] (Spend, [RegainedFor]) cover
// activation. A creature's [AttunementSet] is attuned to with Attune and ended with
// End (closed [EndMeans]: [EndByShortRest], [EndBySpell], [EndByTrigger]) after a
// [ShortRestFocus], a [AttuneCandidate] and an [AttunementLimit]. [WornItems] wears
// [WornItem]s by [WornSlot]. [MagicItemCrafting] (a [CraftingTeam], [CrafterQualification]s,
// a closed [MagicItemCraftingOutcome]) and [CraftingMaterials] plan crafting. A
// sentient item has [SentientTraits] (a closed [SpecialPurpose]) and a
// [SentientConflict]. The closed [MagicItemEvent] family ([MagicItemAttuned],
// [MagicItemAttunementEnded], [MagicItemConsumed], [MagicItemChargesSpent],
// [MagicItemRecharged], [SentientControlTaken], [SentientControlEnded]) joins [Event].
// The use cases are [AttuneToMagicItem], [EndMagicItemAttunement], [RechargeMagicItem],
// [RollPotionMiscibility], [CheckCraftingMaterials], [PlanMagicItemCrafting] (reading a
// [MagicItemCraftingSource]), [GenerateSentientItemTraits], [ResolveSentientItemConflict],
// [AttemptSentientItemControl] and [RepeatSentientControlSave]. The host supplies the
// item's facts, instance ids, the moment of a recharge, death and revival, and the
// judgement of a described prerequisite; the Charmed condition of a sentient item's
// control is data the host applies.
//
// Level advancement and multiclassing: the pure [LevelAdvancementPreviewer] returns a
// closed [LevelAdvancementPreview] ([LevelAdvancementGain] or [LevelAdvancementBlock]) with
// an [AdvancementClassStatus] ([AdvancementClassStatusExisting], [AdvancementClassStatusNew]
// holding the class's multiclass grants) and the [PendingAdvancementChoice]s a level leaves
// open ([PendingSubclassChoice], [PendingFeatGrant]). An [AbilityScoreIncrease] (built by
// [NewAbilityScoreIncrease], [NewAbilityScoreImprovementOneByTwo],
// [NewAbilityScoreImprovementTwoByOne] or [NewEpicBoonIncrease]) is applied all or nothing by
// [AbilityScoreIncreaser], refused with [IncreaseExceedsMaximum]. [SubclassSelection] chooses
// and reports pending subclasses; the subclass is a [ChosenSubclass] under [SubclassStateKey],
// a refusal is a [SubclassRejected] and the event [SubclassChosen] is the closed
// [LevelAdvancementEvent] family. [MulticlassTraining] lists the grants of the classes after the
// first and [ExtraAttackCombination] combines per-class Extra Attack counts. The use cases are
// [PreviewLevelAdvancement] and [ChooseSubclass], each with a command and a closed result.
// The host supplies the class definitions through the content source and takes a reported
// [PendingFeatGrant] to the feats rules.
//
// Casting and Concentration: [CastSpell] casts a spell in a [CastMode] under a
// [CastingContext] (built from [CastingFacts]) and returns a closed [CastSpellResult]:
// [CastSpellCast] with a [CastReport], [CastSpellStarted] for a casting of a minute or
// more (finished by [CompleteCasting], a closed [CompleteCastingResult]) or
// [CastSpellRejected] with a [CastRejection] of [CastRejectionKind]s in a fixed order.
// [BreakConcentration] and [BeginConcentration] end and begin Concentration, each with a
// closed result; the pure [ConcentrationRules] gives [ConcentrationBreakTransition] and
// [CounterCastTransition], and [ConcentrationSaveDCForDamage] gives a
// [ConcentrationSaveDC]. A cast without a casting class takes [SpellCastTerms]. The
// closed [CastingEvent] family ([SpellCast], [CastingCountered], [ConcentrationBegan],
// [ConcentrationEnded], [CastingFailed]) joins [Event] beside [SpellSlotExpended]; the
// reasons are [ConcentrationBreakReason] and [ConcentrationEndReason]. The host reports
// the caster's facts, sets SlotSpentThisTurn when a report says [CastReport.CountsAsSlotSpell],
// rolls the Constitution saving throw after damage and passes the result on.
//
// Magic item effects: [EnhancementBonusLookup] (an [EnhancementFamily] and a rarity give an
// [EnhancementBonusResult]) and [MagicAmmunition]; the closed [MagicItemEffect]
// ([ArmorClassBonus], [SavingThrowBonus], [AttackAndDamageBonus], [SpellAttackRollBonus],
// [IgnoresHalfCoverOnSpellAttacks], [SetAbilityScore], [RaiseAbilityScore],
// [BoostBestowedScore]) of the [MagicItemEffectTable], switched on by an [ItemEffectContext]
// in [ActiveEffectsOf] and read through [ActiveItemEffects] and [MagicItemBonus]; and
// [EffectiveScoresOf], which gives [EffectiveAbilityScores] with an [AbilityScoreSource] per
// ability. Potions: [PotionOfHealingGrades], [ResistanceTypeTable], [GrantedResistances].
// Charged spell items: a [ChargedSpellOption] ([FixedChargeCost], [ChargesPerLevel]) is
// planned and paid by [ChargedSpellCasting]; a [LastChargeRule] ([LastChargeRules] holds
// the table) is resolved by [ResolveLastCharge]; [StaffSpellAbsorption] and [RetributiveStrike]
// cover the Staff of the Magi; [SpellStorage] holds the [StoredSpell]s of a Ring of Spell Storing; an
// [ExtradimensionalSpace] ([SpaceCheck], [ContainerFates], [ExtradimensionalNesting],
// [AirSupplies]) covers the three bags. The closed [ChargedItemEvent] family
// ([MagicItemLastChargeResolved], [RingSpellStored], [RingStoredSpellCast],
// [StaffSpellAbsorbed]) joins [Event]. The use cases are [RollPotionOfHealing],
// [ResolveLastCharge] and [RollSpellStoringRingLevels]. Healing, Resistance, Strength and
// the clock stay the host's.
//
// Tools, crafting and scribing: pure functions over loaded values, with no use case and no
// signal. [ToolProficiencies] (by tool id; [MonsterToolProficiencies]) and [ResolveTool] give a
// [ResolvedTool] or a closed [ToolResolution] ([ToolResolved], [ToolFamilyNeedsVariant],
// [ToolFamilyMissing]). [EvaluateToolCheck] reads a [ToolCheckInput] and gives
// [ToolCheckBenefits] (a Proficiency Bonus multiple of 0 to 2, Advantage when proficient in
// the tool and the skill used, [ToolCheckBenefits.RollModifiers]). [UtilizeActions],
// [UtilizeDC] (a closed [ToolUtilizeDCResult]) and [UtilizeCheck] (a closed
// [ToolUtilizeCheckResult]) give a Utilize action's DC. [PlanItemCrafting] reads an
// [ItemCraftingInput] (a [CraftingParty] of [CraftingWorker]s) and gives a closed
// [ItemCraftingPlan]: [ItemCraftingReady] with exact [WorkHours], or [ItemCraftingBlocked]
// with every [ToolCraftingShortfall]. [QualifyScribe] gives a closed [ScribeQualification];
// [BeginScribing] a closed [ScribingBeginResult] holding a [ScrollScribingJob], whose
// [ScrollScribingJob.WorkDay] takes a [PreparedSpellSet] (from
// [PreparedSpellSetFromSpellcasting]) and gives a closed [ScribingDayOutcome] ending in a
// [ScribedScroll]. The host holds the proficiencies beside the Character, passes the
// materials-at-hand fact and resolves the checks with the d20 resolver.
//
// Feats: [TakenFeats] holds the [TakenFeat]s of a character beside the Character (the zero value
// is empty), with [HeldFeatures] and [ProficiencyGrants] as host-merged inputs and outputs. A
// [FeatGrant] ([FeatSource] and a [FeatGrantScope]: [SpecifiedFeat], [FeatFromCategory], [AnyFeat]) says
// which feats a source allows. [FeatTakingRules] checks in a fixed order and applies a feat
// ([FeatTaking]: [FeatTakingChanged] or [FeatTakingRejected] with a [FeatRejection]); the
// [FeatChoices] it takes are [NoFeatChoices], [SingleAbilityIncrease], [AbilityScoreImprovementChoices],
// [MagicInitiateChoices] and [SkilledChoices] (built by validating constructors that return a
// [FeatChoicesError]), and [FightingStyleChoice] picks a Fighting Style feat. [FeatQualification]
// checks prerequisites, [FeatUses] expends and renews limited uses ([FeatRenewal], [UseRenewedBy]),
// [MagicInitiateSpellChange] replaces a Magic Initiate spell, [FeatReplacement] replaces a held feat and
// [BonusFeats] counts the Bonus Feats at Level 20 still owed. Each answers with a closed result
// ([FeatsTransition]). The declared effects of held feats are a closed [FeatEffect] set for the combat
// rules or the host to interpret, read with [FeatEffects]; [FreeCastingRule] and [GreatWeaponFighting] decide
// the two feats that roll. The closed [FeatEvent] family ([FeatTaken], [FeatSpellReplaced],
// [FeatUseExpended], [FeatUsesRenewed], [FeatReplaced]) joins [Event]. The use cases are [TakeFeat],
// [ChangeMagicInitiateSpell], [ExpendFeatUse], [RenewFeatUses] and [RollFreeCasting]. The host
// calls [RenewFeatUses] after each rest, passes [LevelGained] to [ChangeMagicInitiateSpell], casts
// the spells of [FeatGrantedSpells] and merges [ProficiencyGrants].
//
// Starting at higher levels, higher-level equipment and Trinkets: the pure [HigherLevelStart] folds a
// plan of [HigherLevelStep]s (built by [NewHigherLevelStep]) over a level 1 Character and returns a closed
// [HigherLevelStartOutcome] ([HigherLevelStartBuilt] with the events and the [ConsumedFeatGrant]s of the
// applied increases, or [HigherLevelStartRefused] with a closed [HigherLevelStartRejection]); its
// StartingExperience gives the Experience Points of the starting level. [HigherLevelEquipmentRules]
// answers the Starting Equipment at Higher Levels table with a closed [HigherLevelEquipment]
// ([NoHigherLevelEquipment] or a [HigherLevelEquipmentAllowance] with [RolledGold] and TotalGold). A
// [TrinketTable] of [TrinketCount] [Trinket]s is supplied by the host. The use cases are
// [StartAtHigherLevel] (a [StartAtHigherLevelCommand] of [StartingLevelStep]s and a closed
// [StartAtHigherLevelResult]), [RollHigherLevelStartingMoney] and [RollTrinket], each with a command and
// a closed result. The host keeps the reported [ConsumedFeatGrant]s beside the character so the same
// Ability Score Improvement is never offered twice, and supplies the Trinkets.
//
// Spell Scroll use: the pure functions [AssessScrollReading], [ResolveScrollCheck], [SettleScrollReading],
// [AssessScribing], [AssessScrollCopying] and [ResolveScrollCopy] (no use case, no signal). A [SpellScroll]
// is built with [NewSpellScroll] from a [FoundScroll] or a [ScribedScrollOrigin] ([ScribedScrollStats]); a
// [ScrollReaderProfile] (from [ScrollReaderFacts], [ScrollClassRoute] and [AnyScrollRoute]) is assessed to a
// closed [ScrollAssessmentResult] ([ScrollReady], [ScrollCheckRequired], [ScrollNotReadable],
// [ScrollNotOnSpellList], [ScrollAlreadyUsed]) whose [ScrollCastTerms] become the terms of a CastSpell with
// CastModeSpellScroll. The host rolls the check as a d20 ability check and passes the d20 result to
// [ResolveScrollCheck] (a closed [ScrollCheckResolutionResult]: [ScrollCheckPassed], [ScrollSpellErased]); it
// derives a [ScrollCastProgress] from the cast result, and [SettleScrollReading] returns the
// [ScrollDisposition] that says whether the scroll is consumed.
//
// Character creation: [CreateCharacter] creates a Character from a [CreateCharacterCommand] (every choice of
// "Create Your Character": the ids, an [AbilityScoreAssignment], a [BackgroundAbilityIncrease], class skills
// and tools, [EquipmentOptionLetter]s, a [ToolSelection], a [SizeChoice], [SpeciesTraitSelection]s,
// [SpeciesGrantSelections], the Origin feats' choices, the chosen languages, the alignment, the
// [ClassCreationChoices], a [TrinketChoice], the [RuleOptions], a plan of [StartingLevelStep]s and a
// [HigherLevelEquipmentChoice]) and returns a closed [CreateCharacterResult]: [CreateCharacterSucceeded] (the
// Character, a [CreationRecord] of the choices the aggregate does not store, and the events),
// [CreateCharacterRejected] (every [CreateCharacterProblem] of the first phase that found any, by
// [CreationStep] and [CreateCharacterProblemKind]) or [CreateCharacterContentMissing]. The record holds the
// skills, [Languages], [StartingEquipmentGrant], [TrinketRoll], allowance and taken feats. [RollAbilityScores]
// rolls the six totals of Random Generation ([RolledAbilityScores]) and [RollLanguages] two Standard languages
// ([StandardLanguageRoll]), so a host has the values before it builds the command. The pure rules are
// [StandardArrayScores], [PointCostOf], [PlusTwoPlusOne] and [PlusOneToAll]. The host applies the species
// choices and the class features after creation.
//
// Wish: [WishResolver] has two use cases. ResolveWish takes a [ResolveWishCommand] (the Character and a [WishChoice]: [DuplicateSpell],
// [ObjectCreation], [InstantHealth], [ResistanceGrant], [SpellImmunityGrant], [SuddenLearning], [RollRedo], [ReshapeReality], each built by a
// validating constructor) and returns a closed [ResolveWishResult]: [ResolveWishDuplicated], [ResolveWishApplied] (the stress rolled and written
// to the Character, the new [TakenFeats] for Sudden Learning) or [ResolveWishRejected] (a [ResolveWishRejection]). The effect is a handoff for the
// host or combat (a closed [WishEffect]: [WishDuplicate], [WishObject], [WishInstantHealth], [WishResistance], [WishSpellImmunity],
// [WishSuddenLearning], [WishRollRedo], [WishReshapeReality]). RollWishStressDamage returns [RollWishStressDamageRolled] with an
// [UnpreventableDamage] of Necrotic damage or [RollWishStressDamageNone]. The stress is the [WishStress] feature state ([WishStressKey]);
// the pure [CheckWish], [ApplyWish], [IncurWishStress], [WishEffectiveScores], [WishAfterDay] and [WishAfterRest] (a [WishTransition])
// carry the rules, and the [WishEvent] family ([WishDuplicated], [WishStressEndured], [WishWeaknessEnded], [WishCastDamageEnded],
// [WishStressDamageRolled]) joins [Event]. The host calls WishAfterDay per day and WishAfterRest after each [RestApplied], and refuses
// the cast of Wish itself when [WishStress.CanCastWish] is false.
//
// Fighter, the reference implementation of a class (a later class copies its shape): [FighterProgression] (built from the Fighter
// content with [NewFighterProgression]) turns a [FighterLevel] into [FighterLevelStats] and the [FighterChoice]s a level brings; the
// [FighterState] feature state ([FighterStateKey]) holds the expended uses and the Fighting Style markers ([FighterState.AfterRest] applies a
// rest), and [FighterCreationChoices] seeds it. The pure services are [SecondWind], [TacticalMind], [Indomitable], [ActionSurge] and
// [FighterChoiceRules]; the declared effects [FighterEffects], [ChampionEffects], [HeroicWarrior] and [HeroicRallyHealing]. The five use cases
// ([UseSecondWind], [UseTacticalMind], [UseIndomitable], [UseActionSurge], [ApplyFighterOptionChoice]) take a command and return a closed result
// ([SecondWindApplied] or [SecondWindNotAvailable], [TacticalMindSucceeded], [TacticalMindStillFailed] or [TacticalMindNotAvailable],
// [IndomitableSucceeded], [IndomitableStillFailed] or [IndomitableNotAvailable], [ActionSurgeApplied] or [ActionSurgeNotAvailable],
// [FighterOptionApplied], [FighterOptionRejected] or [FighterOptionContentMissing]); a refusal carries a [FighterReason]. The [FighterEvent]
// family ([SecondWindUsed], [TacticalMindUsed], [ActionSurgeUsed], [IndomitableUsed], [FighterFightingStyleChosen],
// [FighterFightingStyleReplaced], [FighterWeaponMasteriesChosen], [FighterWeaponMasterySwapped]) joins [Event]. The Weapon Mastery choice is
// kept under [WeaponMasteryChoiceKey]. The host applies Hit Points, the action economy and the effects.
//
// Barbarian (the second class, copying the Fighter's shape): [BarbarianProgression] (built from the Barbarian content with
// [NewBarbarianProgression]) turns a [BarbarianLevel] into [BarbarianLevelStats] (Rages, Rage Damage, Weapon Mastery kinds) and the
// [BarbarianChoice]s a level brings. The [BarbarianState] feature state ([BarbarianStateKey]) holds the expended Rage uses, whether a Rage
// is active and how it stands this turn, the Relentless Rage uses and the spent Persistent Rage and Intimidating Presence
// ([BarbarianState.AfterRest] applies a rest); [BarbarianCreationChoices] seeds it. The host carries the per-turn [BarbarianTurnState] and the
// [BarbarianHostFacts] and [BarbarianAttackFacts] it knows. The pure services are [BarbarianRage] (enter, extend and end a Rage, with the
// [RageEndCause] variants), [BarbarianAttackRules] (Reckless Attack and Brutal Strike with its [BrutalStrikeEffect]s), [BerserkerRules] (Frenzy,
// Retaliation, Intimidating Presence), [RelentlessRage], [PersistentRage] and [BarbarianChoiceRules]; the declared effects are
// [BarbarianEffects], [BarbarianUnarmoredDefenseBasis] and [PrimalChampion] (the ability changes for the host to feed to ChangeAbilityScore).
// The two use cases ([UseRelentlessRage], [ApplyBarbarianOptionChoice]) return a closed result ([RelentlessRageSucceeded],
// [RelentlessRageStillFailed] or [RelentlessRageNotAvailable], [BarbarianOptionApplied], [BarbarianOptionRejected] or
// [BarbarianOptionContentMissing]); a refusal carries a [BarbarianReason]. The [BarbarianEvent] family ([RageStarted], [RageEnded],
// [RelentlessRageUsed], [RagesRegained], [IntimidatingPresenceUsed], [BarbarianWeaponMasteriesChosen], [BarbarianWeaponMasterySwapped]) joins [Event].
// The extra damage of Brutal Strike and Frenzy is returned as dice for the combat caller to roll; the host applies Hit Points, the action economy
// and the effects.
//
// Monk (the third class, copying the Fighter's shape): [MonkProgression] (built from the Monk content with [NewMonkProgression]) turns a
// [MonkLevel] into [MonkLevelStats] (the Martial Arts die, the Focus Points maximum, the Unarmored Movement feet). The [MonkState] feature
// state ([MonkStateKey]) holds the expended Focus Points ([FocusPoints] reads them against the maximum of the level), whether Uncanny
// Metabolism was used, the spent Wholeness of Body uses and the [QuiveringPalmVibrations]; [MonkState.AfterRest] applies a rest and
// [MonkCreationChoices] seeds it. The host passes the [MonkAbilities] (modifiers and Proficiency Bonus, [MonkAbilitiesOf] reads them) and the
// target saving throws as a [d20.Outcome]. The pure services are [MonkFocus] (Flurry of Blows, Patient Defense, Step of the Wind),
// [UncannyMetabolism], [PerfectFocus], [DeflectAttacks] and [DeflectRedirect], [StunningStrike] (with [StunApplied] or [StunResisted]),
// [DisciplinedSurvivor], [SuperiorDefense] (a [SuperiorDefenseEffect]), [OpenHandRules], [WholenessOfBody] and [QuiveringPalm]; their
// Availability returns a [MonkAvailability] with the first [MonkReason]. The declared effects are [MonkEffects], [MonkUnarmoredDefenseBasis]
// and [IsMonkWeapon]; the ability changes of Body and Mind are [MonkAbilityChange]s for the host to feed to ChangeAbilityScore. The six use
// cases ([UseFocusFeature], [UseUncannyMetabolism], [ResolveDeflectAttacks], [RedirectDeflectedAttack], [UseWholenessOfBody],
// [EndQuiveringPalm]) return a closed result ending in Applied or NotAvailable. The [MonkEvent] family ([FocusPointsExpended],
// [FocusPointsRegained], [UncannyMetabolismUsed], [AttackDeflected], [DeflectedForceRedirected], [WholenessOfBodyUsed],
// [StunningStrikeAttempted], [QuiveringPalmStarted], [QuiveringPalmEnded], [DisciplinedSurvivorUsed], [SuperiorDefenseActivated]) joins [Event].
// The host applies Hit Points, Temporary Hit Points, the conditions and the action economy.
//
// Rogue (domain only, no use case or port): [RogueProgression] (built from the Rogue content with [NewRogueProgression]) gives the Sneak Attack
// dice ([SneakAttackDice], with an Expression for the Critical Hit doubling), the feature ids of each level, the saving throws of Slippery Mind
// and the constants of the Rogue Features table; [RogueLevel] and [RogueEntry] are its inputs. [RogueChoices] lists the choices a level requires
// ([RogueChoiceKind]) and validates the ones the Rogue owns (skills, [RogueExpertise], the Thieves' Cant language, the Starting Equipment
// option); a constructor rejects untrusted input with a [RogueProblemsError] listing every problem, and [NewRogueCreationChoices] seeds the
// [RogueState] ([RogueStateKey]: Stroke of Luck, the Expertise windows used, the language). The services are [SneakAttack] (a [SneakAttackResult]),
// [NewCunningStrikeSelection] (a [CunningStrikeSelection] of [CunningStrikeOutcome]s), [RogueGrantedActions], [SteadyAim], [RogueUncannyDodge],
// [RogueEvasion], [RogueReliableTalent], [RogueElusive], [StrokeOfLuck] (returning the new roll, the new state and a [StrokeOfLuckUsed] event of the
// [RogueEvent] family), and the Thief's [ThiefFeatures], [UseMagicDevice] and [ThiefReflexes]. The host rolls the dice, applies the damage, the
// conditions and the saves, stores the state with WithFeatureState and applies [RogueState.AfterRest].
//
// Wizard (spell use cases, no dice): [WizardRules] (stateless maths over the bundled Wizard [content.ClassDefinition]: highest Spell Level, cantrips gained and pending,
// the two spells a level owes), [WizardSpellFacts] (what the rules need to know of a spell; the caller resolves spells and supplies them) and [SpellbookRules] (start, level up,
// copy in, copy out and replace a lost book, with a [SpellCopyCost]). [WizardResources] ([WizardResourcesKey]: Arcane Recovery used, the Overchannel count held for the Evoker,
// Scholar, [SpellMastery] and [SignatureSpells] with their free casts) is the [FeatureState]; the host applies [WizardResources.AfterRest] on a [RestApplied] and passes
// [WizardAlwaysPrepared] to spellcasting. [NewWizardCreationChoices] validates the level 1 choices and seeds the state. Fifteen use cases, each with a closed result set:
// [ChooseWizardSkills], [ChooseWizardEquipment], [ChooseWizardCantrips], [ChooseScholarExpertise], [StartSpellbook], [AddLevelUpSpells], [CopySpellIntoBook],
// [CopySpellbook], [ReplaceLostSpellbook], [UseArcaneRecovery], [MemorizeSpell], [CheckRitualAdept], [ChooseSpellMastery], [ReplaceSpellMastery] and [ChooseSignatureSpells].
// The twelve that read the class look it up through the content source and report [content.MissingContent]; a rule refusal is a typed rejection (for example
// [SpellbookRejection]), not an error. Events are [WizardEvent]s.
//
// Wizard Evoker (SRD 5.2.1, "Wizard Subclass: Evoker", p. 82): [EvokerRules] (stateless: [EvokerRules.Gate] decides from the Evoker [content.SubclassDefinition] the host passes
// whether a feature is held, [EvokerGate]; then the spellbook additions, Potent Cantrip, Sculpt Spells, Empowered Evocation and Overchannel), the closed outcomes
// [PotentCantripOutcome], [SculptedCreatures], [EmpoweredRoll] and [OverchannelPlan], and the five rejection types ([EvokerSpellbookRejection], [PotentCantripRejection],
// [SculptRejection], [EmpoweredRejection], [OverchannelRejection]). Six use cases: [AddEvocationSavantSpells] and [AddEvokerFreeSpell] (which look the Wizard up and resolve the
// subclass id the host names) and the pure [ResolvePotentCantrip], [ResolveSculptSpells], [ResolveEmpoweredEvocation] and [UseOverchannel] (which rolls the Necrotic damage).
// The host supplies the spell and creature facts, rolls and applies the damage (maximum damage, half damage, Sculpt Spells' automatic successes and the Necrotic damage that
// ignores Resistance and Immunity are handoff values), stores the [WizardResources] that [UseOverchannel] returns, and applies [WizardResources.AfterRest]. [OverchannelUsed] is a [WizardEvent].
//
// Cleric (SRD 5.2.1, pp. 36-40): [ClericProgression] (built from the Cleric content with [NewClericProgression]; [ClericLevelStats] gives the Channel
// Divinity maximum read from the Features table column and the Divine Spark dice; the features and [ClericPendingChoice]s of each level), [ClericLevel],
// [ClericChoices] ([ClericChoicesKey]: [DivineOrder], the Thaumaturge's extra cantrip, [BlessedStrikesOption]) and [ClericResources] ([ClericResourcesKey]:
// Channel Divinity uses and the [DivineInterventionState]; [ClericResources.AfterRest]) are the inputs and the stored states, seeded by [ClericCreationChoices]
// ([NewClericCreationChoices]). [ClericRuleData] holds the recommended spells and the [ExtraDivineInterventionGrant]s (Wish at level 20, 2d4). The pure domain
// functions are [ChooseDivineOrder], [ThaumaturgeCheckBonus], [ChooseBlessedStrikes], [ResolveBlessedStrikes], [AdvanceCleric], [ApplyClericRest],
// [ResolveClericHealing] (Disciple of Life, Blessed Healer, Supreme Healing) and the services [DivineSpark], [TurnUndead], [SearUndead], [PreserveLife],
// [DivineIntervention], [SupremeHealing] and [LifeDomainSpells] (read from the held Life Domain subclass). The two use cases are [UseClericChannelDivinity]
// (Divine Spark, Turn Undead with Sear Undead, Preserve Life; the effect data is a [ClericChannelOutcome]) and [UseDivineIntervention] (a free-cast request
// the host passes to CastSpell with CastModeFeatureGranted; Wish rolls the recharge dice). Refusals are results carrying a [ClericReason]; events are
// [ClericEvent]s. The host supplies the target facts, rolls the saving throws and applies the Hit Points, Temporary Hit Points and conditions.
//
// Druid (SRD 5.2.1, pp. 41-46; domain only, no use case, port or signal): [DruidFeatureTables] (built from the Druid content with [DruidTablesFromClass], or by hand
// with [NewDruidFeatureTables]: Wild Shape uses by level, the [BeastShapeTier]s, the [DruidLand]s with their Circle Spells and Nature's Ward Resistance, the features
// granted at each level), [DruidCharacterFacts] (the id, the Druid level, the subclass and the Proficiency Bonus of the total level) and [DruidFeatureState]
// ([DruidFeatureStateKey]: [PrimalOrder], [ElementalFury], [LandType], the Wild Shape uses, [KnownForms] and active form, the once-per-turn and once-until-Long-Rest
// flags, the open window of [RestWindowPermission]s, the Wild Companion familiar and the Nature's Sanctuary Cube; restored with [NewDruidFeatureState]) are the
// inputs and the stored state, seeded by [DruidCreationChoices]. The four pure operations are [GainDruidLevel], [ApplyDruidChoice] ([DruidChoice]), [UseDruidFeature]
// ([DruidFeatureUse]) and [ApplyDruidTrigger] ([DruidTrigger]); each returns its own closed result, a refusal carries a [DruidRejection] (the first of the D-5 order)
// and the events are [DruidEvent]s. A feature that spends or regains a spell slot returns [DruidSlotChange] data for the host to apply with ExpendSlot or RegainSlot;
// the Druid reads the character's [SpellcastingState] and never changes it. [DruidRules] holds the stateless rules: [DruidRules.MagicianCheckBonus],
// [DruidRules.BeastFormCastCheck], [DruidRules.PotentSpellcastingBonus], [DruidRules.ImprovedPotentRange], [DruidRules.PrimalStrikeDice], [DruidRules.LandsAidDice],
// [DruidRules.NaturalRecoveryBudget], [DruidRules.CircleSpells], [DruidRules.FormEligibility] (of a [BeastFormFacts], from [BeastFormFactsFromMonster]),
// [DruidRules.ShapeShiftedModifier], [DruidRules.ShapeShiftedStats], [DruidRules.DruidicDetection], [DruidRules.Languages], [DruidRules.AlwaysPreparedSpells], [DruidRules.NaturesWard] and [DruidRules.FeatureAttribute] (the srd5e.feature value of a
// use, for the host to record). The host supplies the attack results, spell facts, distances and the rest and turn triggers, rolls the dice and applies the
// Temporary Hit Points, the damage and the Resistance.
//
// Sorcerer (SRD 5.2.1, pp. 64-70): [SorcererLevel] and [SorcererProfile] (a level and the chosen subclass, built by [SorcererProfileOf] and never stored) are the inputs
// of the pure domain functions, and the [SorcererState] feature state ([SorcererStateKey]; restored with [SorcererStateParts] by [RestoreSorcererState]) holds the Metamagic
// options ([MetamagicEffect]), the Sorcery Points, the Innate Sorcery uses, the created slots and the Draconic Sorcery flags; [SorcererCreationChoices] seeds it. The domain
// functions return a closed [SorcererTransition] ([SorcererChanged] or [SorcererRefused] with a [SorcererRejection] of a [SorcererRejectionKind]): [ActivateInnateSorcery] and
// [EndInnateSorcery], [ConvertSpellSlot] (a [SlotConversionSource]), [CreateSlot], [SpendCreatedSlot], [ApplySorcererRest], [ChooseMetamagic], [ChooseElementalAffinity],
// [ApplyElementalAffinity], [UseDragonWings] and [UseDragonCompanion]; [ApplyMetamagic] takes a [MetamagicRequest] (the host's [MetamagicSpellFacts], [MetamagicCastContext] and
// [MetamagicParameters]) and gives a closed [MetamagicTransition] ([MetamagicApplied] with a [MetamagicApplication], or [MetamagicRefused]). The declared effects are
// [SorcererInnateSorceryEffects] and [SorcererEffects]; [SorcererLevelGrantsFor] gives the [SorcererLevelGrants] of a level. The four use cases ([ChooseMetamagicOptions],
// [ApplySorcererLevelUp], [CreateSpellSlot], [UseMetamagic]) read the class definition through a [ContentSource] and return a closed result with a refusal as a
// [SorcererRejection]; the [SorcererEvent] family joins [Event]. The host rolls the Empowered Spell and Seeking Spell dice, times Innate Sorcery and Dragon Wings, casts with
// CastModeFeatureGranted and applies the Resistance, the Fly Speed and the grants.
//
// Warlock (SRD 5.2.1, pp. 70-76): [WarlockLevel] and [WarlockProfile] (a level and the chosen subclass, built by [WarlockProfileOf] and never stored) are the inputs of the pure
// domain functions, and the [WarlockState] feature state ([WarlockStateKey]; restored with [WarlockStateParts] by [RestoreWarlockState]) holds the Eldritch Invocations known, the
// Mystic Arcanum spells, the uses spent of Magical Cunning, Contact Patron, Gift of the Depths, Dark One's Own Luck and Hurl Through Hell and the Fiendish Resilience type;
// [WarlockCreationChoices] seeds it. The domain functions return a closed [WarlockTransition] ([WarlockChanged] or [WarlockRefused] with a [WarlockRejection] of a
// [WarlockRejectionKind]): [CompleteMagicalCunning], [UseContactPatron], [ChooseMysticArcanum], [CastMysticArcanum], [ChooseInvocations], [GrantInvocationSpell], [UseEldritchSmite],
// [ResolveDarkOnesBlessing], [ChooseFiendishResilience], [AddDarkOnesOwnLuck], [UseHurlThroughHell], [RestoreHurlThroughHell] and [ApplyWarlockRest]. A [WarlockChanged] may carry
// a [WarlockSpellGrant] (a spell to cast with CastModeFeatureGranted and no slot), a [WarlockSlotChange] ([WarlockSlotChangeExpendPact] or [WarlockSlotChangeRegainPact]) for the
// host to apply with the spellcasting service, a Modifier or extra damage dice; the functions read the Pact Magic slots from a [SpellcastingState] and never change them.
// [SelectInvocations] validates a full selection of [InvocationPick] values against the options of the class definition, the known cantrips ([CantripFacts], supplied by the host)
// and [InvocationEligibility]; [WarlockAlwaysPrepared] reads the Contact Patron and Fiend Spells facts; [DeriveInvocationEffects] gives the standing [InvocationEffect] values.
// The pact services keep their own [WarlockPactState] ([WarlockPactStateKey]; restored with [WarlockPactStateParts] by [RestoreWarlockPactState]), which holds the [PactWeaponBond],
// the [FamiliarPact], the [BookOfShadows] (with the Gift of the Protectors page) and the [SensorLink]; its functions return a closed [WarlockPactTransition] ([WarlockPactChanged] or
// [WarlockPactRefused]): [FormPactWeaponBond], [EndPactWeaponBond], [SelectFamiliarForm], [ForgoAttackForFamiliar], [CommandFamiliarAttack], [ChooseFamiliarSpeed],
// [GrantFamiliarResistance], [ConjureBook], [EndBookOfShadows], [WriteGiftOfTheProtectorsName], [EraseGiftOfTheProtectorsName] and [UseGazeOfTwoMinds]. The value functions
// ([PactWeaponAttackOptions], [ChoosePactWeaponDamageType], [ChangeFamiliarDamageType], [FamiliarSaveDc], [ApplyBookOfShadows]) return the data for combat and the spellcasting
// service, and [ResolveGiftOfTheProtectors] and [ApplyLifedrinker] work on the [WarlockState]. The host supplies the hit, the turn facts ([WarlockTurnFacts]: actions, attacks left),
// the distance and the time of a separated weapon or a Gaze of Two Minds link. The eight use cases ([ChooseWarlockInvocations], [ChooseWarlockMysticArcanum], [CastInvocationSpell],
// [ChooseWarlockFamiliarForm], [ConjureBookOfShadows] and [BondPactWeapon] read content through a [ContentSource]; [UseDarkOnesOwnLuck] and [UseLifedrinker] roll through a [Roller])
// return a closed result with a refusal as a [WarlockRejection]; the [WarlockEvent] family joins [Event]. The host supplies the attack results, turn facts, light level, rest
// and the cantrip facts, times Magical Cunning and the Dark One's Own Luck timing, and applies the damage, the Temporary Hit Points and the Resistance.
//
// Paladin (SRD 5.2.1, pp. 53-57): [PaladinProgression] (built from the Paladin content and, optionally, the Oath of Devotion with [NewPaladinProgression]; [PaladinFeature] names
// the gating features and LevelOf reads their level from the definition, so a moved level moves the gate; it also reads the Channel Divinity column, the always-prepared grants and the
// feat entitlements), [PaladinLevel], [PaladinState] ([PaladinStateKey], restored with [PaladinStateParts] by [RestorePaladinState]: the Lay On Hands pool, the Channel Divinity uses, the
// [LongRestUse] of Paladin's Smite, Faithful Steed and Holy Nimbus, the [PaladinFightingStyle] and the active effects) and [PaladinCreationChoices] (the level 1 state and the Weapon Mastery
// choice) are the inputs and the stored states. The pure domain functions are [LayOnHandsRules], [PaladinAuras], [AttacksPerAttackAction], [RadiantStrikes], [PaladinChoiceRules]
// (the Fighting Style feat, [BlessedWarrior] and its cantrip replacement), [EndPaladinEffect], [AdvancePaladin], [ApplyPaladinRest] and [StartPaladin]; the effect data is [DivineSenseEffect],
// [AbjureFoesEffect], [SacredWeaponEffect], [AuraEffects], [SmiteOfProtectionEffect] and [HolyNimbusEffect]. The seven use cases are [UseLayOnHands], [UsePaladinChannelDivinity],
// [UseFreeCast] (it spends the availability and returns the spell for the host to cast with CastModeFeatureGranted), [UseHolyNimbus] (it expends a level 5 slot of the chosen pool with
// SpellcastingState.ExpendSlot), [ApplySmiteOfProtection], [ChoosePaladinOption] and [CalculatePaladinFeatures]; a refusal is a result carrying every [PaladinReason] that holds, and
// the [PaladinEvent] family joins [Event]. The host supplies the positions, the Incapacitated condition, the held weapon and the Attack action, rolls nothing (no Paladin feature rolls dice)
// and applies the Hit Points, the conditions, the cover, the light and the damage.
//
// Ranger (SRD 5.2.1, pp. 57-61): [RangerProgression] (built from the Ranger content and, optionally, the Hunter with [NewRangerProgression]; [RangerFeature] names the gating features
// and LevelOf reads their level from the definition, so a moved level moves the gate; it also reads the Favored Enemy column, the Hunter’s Mark fact, the Hunter’s Prey and Defensive
// Tactics options and the feat grants), [RangerLevel], [RangerState] ([RangerStateKey], restored with [RangerStateParts] by [RestoreRangerState]: the expended uses of Favored Enemy,
// Tireless and Nature’s Veil, the [DeftExplorerChoice], the Expertise skills, the [RangerFightingStyle] with [RangerFightingStyleFeat] and [DruidicWarrior], the two Hunter options
// and the feat grants used) and [RangerCreationChoices] (the initial state and the Weapon Mastery choice) are the inputs and the stored states. The pure domain functions are the
// methods of [RangerFacts] (Extra Attack, [SpeedProfile] from Roving, [RangerSenseGrant], the Hunter’s Mark die, the Concentration rule, Precise Hunter, Hunter’s Lore, Colossus Slayer,
// Horde Breaker, Superior Hunter’s Prey, Escape the Horde and the [MultiattackDefenseRecord]), [RangerChoiceRules], [UseFavoredEnemy], [UseNaturesVeil],
// [UseSuperiorHuntersDefense], [ApplyRangerRest] and [TirelessTemporaryHitPoints]. The three use cases are [ChooseRangerOption], [ResolveRangerFeatures] and [UseTireless] (the one that
// rolls, 1d8); a refusal is a result carrying every [RangerReason] that holds, and the [RangerEvent] family joins [Event]. A feat taken from a feat grant is validated here and handed to
// TakeFeat with the grant the accepted choice returns. The host supplies the marks, positions, the weapon and the damage, and applies the Temporary Hit Points, the Invisible
// condition and the Resistance.
//
// Species traits (SRD 5.2.1, pp. 83-86): [SpeciesTraits] holds the species, its [SpeciesTraitChoices] (a closed set, one variant per species with a choice: [DragonbornChoices],
// [ElfChoices], [GnomeChoices], [GoliathChoices], [TieflingChoices], [HumanChoices] and [NoSpeciesChoices], built with [ParseSpeciesTraitChoices] from the creation selections and
// reporting every [SpeciesChoiceProblem]), how many times each limited [SpeciesUse] is expended, and the High Elf cantrip and its swap window; the zero value holds no species and the
// use cases reject it with [SpeciesBehaviourUnknown]. The enums are [DraconicAncestry], [ElvenLineage], [GnomishLineage], [GiantAncestry] and [FiendishLegacy]. The pure domain services
// are [SpeciesPassives] (Resistances, [SpeciesAdvantage] sources for saves and checks, the Hit Point maximum bonus, Darkvision, Speed, Trance, the Halfling size rules and the skills),
// [SpeciesGrantedSpells] ([SpeciesGrantedSpell] by level, [ClockworkDevice]), [SpeciesUses] (Expend and Renew, with a closed [SpeciesTraitsTransition]), [SpeciesEffects] (the declared
// [SpeciesTraitEffect]s, [GiantAncestryEffect] and [TemporaryHitPointsGrant] among them, which combat or the host interprets), [StonesEnduranceRule], [RelentlessEndurance]
// ([ZeroHitPointsOutcome] is the host's reading of combat), [HalflingLuck] and [HighElfCantripSwap]. A refusal is a closed [SpeciesTraitRejection] and the [SpeciesTraitEvent] family
// joins [Event]. The six use cases are [ExpendSpeciesTraitUse] (the request is a [SpeciesUseRequest]), [RenewSpeciesTraitUses] (the host calls it with each [RestApplied] and acts on the
// [SpeciesWindow]s it reports, granting Heroic Inspiration for Resourceful), [UseRelentlessEndurance], [RollHalflingLuck], [RollStonesEndurance] and [ReplaceHighElfCantrip]. Casting the
// granted spells, applying the effects and the damage are the host's and combat's.
//
// Use cases are configured with [Option] values ([WithSource], [WithRoller],
// [WithTelemetry], [WithClock]); their signals are named in signals.go.
package character
