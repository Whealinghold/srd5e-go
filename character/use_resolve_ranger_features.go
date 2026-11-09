package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ResolveRangerFeaturesCommand is the input of ResolveRangerFeatures.Run. Subclass is the subclass the Ranger holds ("" for none);
// Speed, HeavyArmor, Wisdom and HuntersMarkDie are the host's facts: the Speed input, whether Heavy armor is worn, the Wisdom
// modifier and the spell’s own damage die (a d6, SRD 5.2.1, "Hunter’s Mark", p. 141, an input from spell content).
// Construct it with a keyed literal. Run does not alter it.
type ResolveRangerFeaturesCommand struct {
	Character      Character
	Subclass       content.SubclassID
	Speed          SpeedInput
	HeavyArmor     bool
	Wisdom         ability.Modifier
	HuntersMarkDie dice.Die
}

// ResolveRangerFeaturesResult is the closed set of outcomes of ResolveRangerFeatures.Run. Variants:
// ResolveRangerFeaturesResolved, ResolveRangerFeaturesNotARanger.
type ResolveRangerFeaturesResult interface{ isResolveRangerFeaturesResult() }

// ResolveRangerFeaturesResolved is the Ranger features at the character's level and choices, computed from the definition.
// Effects of a choice not yet made are absent. Only Run builds one.
type ResolveRangerFeaturesResolved struct {
	attacks                 int
	speed                   SpeedProfile
	senses                  []RangerSenseGrant
	alwaysPrepared          []content.SpellID
	cantrips                []GrantedCantrip
	markDie                 dice.Die
	relentless              bool
	favored, tireless, veil int
	hunterFeatures          []RangerFeature
	pending                 []RangerPendingChoice
}

// AttacksPerAttackAction returns 1 or, from Extra Attack, 2: a total, not a bonus.
func (r ResolveRangerFeaturesResolved) AttacksPerAttackAction() int { return r.attacks }

// Speed returns the Speeds with Roving.
func (r ResolveRangerFeaturesResolved) Speed() SpeedProfile { return r.speed }

// Senses returns the senses the features grant, as a new slice.
func (r ResolveRangerFeaturesResolved) Senses() []RangerSenseGrant { return slices.Clone(r.senses) }

// AlwaysPreparedSpells returns the spells always prepared, as a new slice.
func (r ResolveRangerFeaturesResolved) AlwaysPreparedSpells() []content.SpellID {
	return slices.Clone(r.alwaysPrepared)
}

// GrantedCantrips returns the Druidic Warrior cantrips to hand to spellcasting, as a new slice.
func (r ResolveRangerFeaturesResolved) GrantedCantrips() []GrantedCantrip {
	return slices.Clone(r.cantrips)
}

// HuntersMarkDie returns the damage die of Hunter’s Mark: a d10 from Foe Slayer, else the die given.
func (r ResolveRangerFeaturesResolved) HuntersMarkDie() dice.Die { return r.markDie }

// RelentlessHunter reports whether damage cannot break Concentration on Hunter’s Mark.
func (r ResolveRangerFeaturesResolved) RelentlessHunter() bool { return r.relentless }

// FavoredEnemyUses returns the free casts of Hunter’s Mark at the level (the maximum, not what is left).
func (r ResolveRangerFeaturesResolved) FavoredEnemyUses() int { return r.favored }

// TirelessUses returns the Tireless uses (the maximum), and 0 below its level.
func (r ResolveRangerFeaturesResolved) TirelessUses() int { return r.tireless }

// NaturesVeilUses returns the Nature’s Veil uses (the maximum), and 0 below its level.
func (r ResolveRangerFeaturesResolved) NaturesVeilUses() int { return r.veil }

// HunterFeatures returns the Hunter features held, as a new slice.
func (r ResolveRangerFeaturesResolved) HunterFeatures() []RangerFeature {
	return slices.Clone(r.hunterFeatures)
}

// PendingChoices returns the choices owed at the level and not yet made, as a new slice.
func (r ResolveRangerFeaturesResolved) PendingChoices() []RangerPendingChoice {
	return slices.Clone(r.pending)
}

// ResolveRangerFeaturesNotARanger is the outcome for a character with no Ranger levels.
type ResolveRangerFeaturesNotARanger struct{}

func (ResolveRangerFeaturesResolved) isResolveRangerFeaturesResult()   {}
func (ResolveRangerFeaturesNotARanger) isResolveRangerFeaturesResult() {}

// ResolveRangerFeatures is the use case that summarises the Ranger features a level and its choices give (class-ranger.md FR-096,
// FR-097). It changes nothing and has no rejection. Create it with NewResolveRangerFeatures; the zero value is not usable. It is
// safe for concurrent use only if its telemetry and clock are.
type ResolveRangerFeatures struct{ rangerUse }

// NewResolveRangerFeatures returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression.
func NewResolveRangerFeatures(p RangerProgression, opts ...Option) (ResolveRangerFeatures, error) {
	u, err := newRangerUse("resolve Ranger features", p, opts)
	return ResolveRangerFeatures{u}, err
}

// Run resolves the features of cmd.Character.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character or a zero
// HuntersMarkDie, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled.
func (u ResolveRangerFeatures) Run(ctx context.Context, cmd ResolveRangerFeaturesCommand) (ResolveRangerFeaturesResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	if cmd.HuntersMarkDie.IsZero() {
		return nil, invalidInput("resolve Ranger features without the Hunter’s Mark die")
	}
	ctx, run := u.sig.start(ctx, SpanResolveRangerFeatures, ValueClassRanger, ValueFeatureRangerFeatures)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return ResolveRangerFeaturesNotARanger{}, nil
	}
	f := u.facts(level, state, cmd.Subclass)
	masteries, _, err := ReadFeatureState(cmd.Character.FeatureStates(), WeaponMasteryChoiceKey())
	if err != nil {
		run.commandError(err)
		return nil, err
	}
	r := ResolveRangerFeaturesResolved{
		attacks: f.AttacksPerAttackAction(), speed: f.Speed(cmd.Speed, cmd.HeavyArmor), senses: f.Senses(),
		alwaysPrepared: u.progression.AlwaysPrepared(level), cantrips: state.DruidicWarriorCantrips(), markDie: f.HuntersMarkDie(cmd.HuntersMarkDie),
		relentless: f.has(RangerFeatureRelentlessHunter), favored: u.progression.FavoredEnemyUses(level), hunterFeatures: []RangerFeature{},
		pending: f.pendingChoices(len(masteries.Choice().kinds) > 0),
	}
	if f.has(RangerFeatureTireless) {
		r.tireless = TirelessUses(cmd.Wisdom)
	}
	if f.has(RangerFeatureNaturesVeil) {
		r.veil = TirelessUses(cmd.Wisdom)
	}
	if f.hunter() {
		for feat := RangerFeatureHuntersLore; feat <= RangerFeatureSuperiorHuntersDefense; feat++ {
			if f.has(feat) {
				r.hunterFeatures = append(r.hunterFeatures, feat)
			}
		}
	}
	run.succeeded(ctx)
	return r, nil
}

// pendingChoices returns the choices owed at the level and not yet made, in the order of the level of their feature (class-ranger.md
// FR-097, T11). masteryChosen is whether the Weapon Mastery pair is held.
func (f RangerFacts) pendingChoices(masteryChosen bool) []RangerPendingChoice {
	out := []RangerPendingChoice{}
	add := func(owed bool, c RangerPendingChoice) {
		if owed {
			out = append(out, c)
		}
	}
	_, deft := f.State.DeftExplorer()
	_, expertise := f.State.Expertise()
	_, style := f.State.FightingStyle()
	prey, _ := f.State.HuntersPrey()
	tactics, _ := f.State.DefensiveTactics()
	general, epic := f.Progression.FeatGrantsUpTo(f.Level)
	takenGeneral, takenEpic := f.State.FeatsTaken()
	add(f.has(RangerFeatureWeaponMastery) && !masteryChosen, RangerPendingWeaponMastery)
	add(f.has(RangerFeatureDeftExplorer) && !deft, RangerPendingDeftExplorer)
	add(f.has(RangerFeatureFightingStyle) && !style, RangerPendingFightingStyle)
	add(f.has(RangerFeatureSubclass) && f.Subclass == "", RangerPendingSubclass)
	add(f.hunterHas(RangerFeatureHuntersPrey) && prey == "", RangerPendingHuntersPrey)
	for range max(general-takenGeneral, 0) {
		out = append(out, RangerPendingAbilityScoreImprovement)
	}
	add(f.hunterHas(RangerFeatureDefensiveTactics) && tactics == "", RangerPendingDefensiveTactics)
	add(f.has(RangerFeatureExpertise) && !expertise, RangerPendingExpertise)
	add(epic-takenEpic > 0, RangerPendingEpicBoon)
	return out
}
