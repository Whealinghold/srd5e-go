package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// PaladinChannelDivinityOption is a Channel Divinity effect option of the Paladin. The zero value is invalid.
type PaladinChannelDivinityOption int

const (
	// PaladinChannelDivineSense: Divine Sense, from the level of the Channel Divinity entry.
	PaladinChannelDivineSense PaladinChannelDivinityOption = iota + 1
	// PaladinChannelAbjureFoes: Abjure Foes, from the level of its entry.
	PaladinChannelAbjureFoes
	// PaladinChannelSacredWeapon: Sacred Weapon, from the level of its entry with Oath of Devotion chosen.
	PaladinChannelSacredWeapon
)

// Valid reports whether o is one of the options.
func (o PaladinChannelDivinityOption) Valid() bool {
	return o >= PaladinChannelDivineSense && o <= PaladinChannelSacredWeapon
}

// String returns the srd5e.feature value, "divine_sense", "abjure_foes" or "sacred_weapon" ("unknown" for another value).
func (o PaladinChannelDivinityOption) String() string {
	switch o {
	case PaladinChannelDivineSense:
		return ValueFeatureDivineSense
	case PaladinChannelAbjureFoes:
		return ValueFeatureAbjureFoes
	case PaladinChannelSacredWeapon:
		return ValueFeatureSacredWeapon
	}
	return "unknown"
}

func (o PaladinChannelDivinityOption) paladinFeature() PaladinFeature {
	switch o {
	case PaladinChannelAbjureFoes:
		return PaladinFeatureAbjureFoes
	case PaladinChannelSacredWeapon:
		return PaladinFeatureSacredWeapon
	case PaladinChannelDivineSense:
	}
	return PaladinFeatureChannelDivinity
}

// PaladinChannelEffect is the data of the effect a Channel Divinity use created, a closed set. Variants: DivineSenseEffect,
// AbjureFoesEffect, SacredWeaponEffect.
type PaladinChannelEffect interface{ isPaladinChannelEffect() }

func (DivineSenseEffect) isPaladinChannelEffect()  {}
func (AbjureFoesEffect) isPaladinChannelEffect()   {}
func (SacredWeaponEffect) isPaladinChannelEffect() {}

// UsePaladinChannelDivinityCommand is the input of UsePaladinChannelDivinity.Run. Subclass is the subclass the Paladin holds ("" for
// none). SpellSaveDC is the Paladin's, from the Spellcasting feature, and is never recomputed; Charisma is the Charisma modifier.
// TakingAttackAction and HoldingMeleeWeapon are host facts for Sacred Weapon, and Weapon is the host's key of the held Melee weapon.
// Construct it with a keyed literal. Run does not alter it.
type UsePaladinChannelDivinityCommand struct {
	Character          Character
	Option             PaladinChannelDivinityOption
	Subclass           content.SubclassID
	Charisma           ability.Modifier
	SpellSaveDC        SpellSaveDC
	TakingAttackAction bool
	HoldingMeleeWeapon bool
	Weapon             string
}

// UsePaladinChannelDivinityResult is the closed set of outcomes of UsePaladinChannelDivinity.Run. Variants:
// UsePaladinChannelDivinityApplied, UsePaladinChannelDivinityRejected, UsePaladinChannelDivinityNotAPaladin.
type UsePaladinChannelDivinityResult interface{ isUsePaladinChannelDivinityResult() }

// UsePaladinChannelDivinityApplied is the outcome of an accepted use. Only Run builds one.
type UsePaladinChannelDivinityApplied struct {
	character Character
	effect    PaladinChannelEffect
	replaced  string
	state     PaladinState
	events    []Event
}

// Character returns the new Character, with one use expended and the effect active.
func (r UsePaladinChannelDivinityApplied) Character() Character { return r.character }

// Effect returns the data of the effect.
func (r UsePaladinChannelDivinityApplied) Effect() PaladinChannelEffect { return r.effect }

// ReplacedWeapon returns the host key of the weapon an earlier Sacred Weapon imbued and false when none was replaced.
func (r UsePaladinChannelDivinityApplied) ReplacedWeapon() (string, bool) {
	return r.replaced, r.replaced != ""
}

// State returns the new PaladinState.
func (r UsePaladinChannelDivinityApplied) State() PaladinState { return r.state }

// Events returns a new slice holding the PaladinChannelDivinityUsed event, preceded by PaladinEffectEnded (used again) when a
// Sacred Weapon was replaced.
func (r UsePaladinChannelDivinityApplied) Events() []Event { return slices.Clone(r.events) }

// UsePaladinChannelDivinityRejected is the outcome of a use a rule refused: the Character is unchanged, there are no events and no
// use is expended. Only Run builds one.
type UsePaladinChannelDivinityRejected struct{ reasons []PaladinReason }

// Reasons returns every reason that holds in declaration order, as a new slice. It is never empty.
func (r UsePaladinChannelDivinityRejected) Reasons() []PaladinReason { return slices.Clone(r.reasons) }

// UsePaladinChannelDivinityNotAPaladin is the outcome for a character with no Paladin levels.
type UsePaladinChannelDivinityNotAPaladin struct{}

func (UsePaladinChannelDivinityApplied) isUsePaladinChannelDivinityResult()     {}
func (UsePaladinChannelDivinityRejected) isUsePaladinChannelDivinityResult()    {}
func (UsePaladinChannelDivinityNotAPaladin) isUsePaladinChannelDivinityResult() {}

// UsePaladinChannelDivinity is the use case that uses a Paladin's Channel Divinity (SRD 5.2.1, "Level 3: Channel Divinity", p.
// 54): Divine Sense, Abjure Foes or the Oath of Devotion's Sacred Weapon. It holds no rules logic and rolls no dice. Create it with
// NewUsePaladinChannelDivinity; the zero value is not usable. It is safe for concurrent use only if its telemetry and clock are.
type UsePaladinChannelDivinity struct{ paladinUse }

// NewUsePaladinChannelDivinity returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error
// matching srderr.ErrInvalidInput for the zero progression.
func NewUsePaladinChannelDivinity(p PaladinProgression, opts ...Option) (UsePaladinChannelDivinity, error) {
	u, err := newPaladinUse("use Paladin Channel Divinity", p, opts)
	return UsePaladinChannelDivinity{u}, err
}

// Run uses Channel Divinity for cmd.Character. Every reason that holds is reported in declaration order: feature_not_gained
// (below the level of the Channel Divinity entry), option_not_available, not_taking_attack_action and no_melee_weapon_held (Sacred
// Weapon only), no_uses_remaining. A Sacred Weapon used while another is active spends another use and replaces it.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character, an invalid
// option or a Sacred Weapon held without a weapon key, and ctx.Err() unwrapped, before any signal, when ctx is already cancelled. A
// refused use is a result.
func (u UsePaladinChannelDivinity) Run(ctx context.Context, cmd UsePaladinChannelDivinityCommand) (UsePaladinChannelDivinityResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	if !cmd.Option.Valid() || (cmd.Option == PaladinChannelSacredWeapon && cmd.HoldingMeleeWeapon && cmd.Weapon == "") {
		return nil, invalidInput("use Paladin Channel Divinity: undefined option, or Sacred Weapon without a weapon key")
	}
	ctx, run := u.sig.start(ctx, SpanUsePaladinChannelDivinity, ValueClassPaladin, cmd.Option.String())
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return UsePaladinChannelDivinityNotAPaladin{}, nil
	}
	var reasons []PaladinReason
	if !u.progression.has(PaladinFeatureChannelDivinity, level) {
		reasons = append(reasons, PaladinFeatureNotGained)
	}
	if cmd.Option != PaladinChannelDivineSense && !u.progression.has(cmd.Option.paladinFeature(), level) || cmd.Option == PaladinChannelSacredWeapon && !oathChosen(cmd.Subclass) {
		reasons = append(reasons, PaladinOptionNotAvailable)
	}
	if cmd.Option == PaladinChannelSacredWeapon {
		if !cmd.TakingAttackAction {
			reasons = append(reasons, PaladinNotTakingAttackAction)
		}
		if !cmd.HoldingMeleeWeapon {
			reasons = append(reasons, PaladinNoMeleeWeaponHeld)
		}
	}
	if state.channelRemaining < 1 {
		reasons = append(reasons, PaladinNoUsesRemaining)
	}
	if len(reasons) > 0 {
		return UsePaladinChannelDivinityRejected{reasons: paladinReject(ctx, run, reasons)}, nil
	}
	state.channelRemaining--
	var effect PaladinChannelEffect
	var events []Event
	replaced := ""
	switch cmd.Option {
	case PaladinChannelDivineSense:
		state.divineSense = true
		effect = DivineSenseEffect{}
	case PaladinChannelAbjureFoes:
		effect = AbjureFoesEffect{targets: minOne(cmd.Charisma), dc: cmd.SpellSaveDC}
	case PaladinChannelSacredWeapon:
		if replaced = state.sacredWeapon; replaced != "" {
			events = append(events, PaladinEffectEnded{id: cmd.Character.ID(), effect: PaladinEffectSacredWeapon, reason: PaladinEndUsedAgain})
		}
		state.sacredWeapon = cmd.Weapon
		effect = SacredWeaponEffect{weapon: cmd.Weapon, bonus: minOne(cmd.Charisma)}
	}
	next, err := paladinStore(run, cmd.Character, state)
	if err != nil {
		return nil, err
	}
	events = append(events, PaladinChannelDivinityUsed{id: next.ID(), option: cmd.Option.String(), remaining: state.channelRemaining})
	run.succeeded(ctx)
	return UsePaladinChannelDivinityApplied{character: next, effect: effect, replaced: replaced, state: state, events: events}, nil
}
