package character

import (
	"context"
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// HolyNimbusAction is what a UseHolyNimbus call does. The zero value is invalid.
type HolyNimbusAction int

const (
	// HolyNimbusActivate activates Holy Nimbus: a Bonus Action.
	HolyNimbusActivate HolyNimbusAction = iota + 1
	// HolyNimbusRestoreWithSlot restores the use by expending a level 5 spell slot: no action.
	HolyNimbusRestoreWithSlot
)

// holyNimbusRestoreSlot is the level of the spell slot that restores Holy Nimbus (SRD 5.2.1, "Level 20: Holy Nimbus", p. 57).
const holyNimbusRestoreSlot = 5

// UseHolyNimbusCommand is the input of UseHolyNimbus.Run. Subclass is the subclass the Paladin holds ("" for none). Charisma and
// Proficiency give the Radiant damage and Incapacitated is a host fact; they matter for HolyNimbusActivate. Spellcasting and Pool
// are the SpellcastingState and the SpellSlotPool (PoolSpellcasting or PoolPactMagic) to draw the slot from, for
// HolyNimbusRestoreWithSlot only. Construct it with a keyed literal. Run does not alter it.
type UseHolyNimbusCommand struct {
	Character     Character
	Subclass      content.SubclassID
	Action        HolyNimbusAction
	Charisma      ability.Modifier
	Proficiency   ProficiencyBonus
	Incapacitated bool
	Spellcasting  SpellcastingState
	Pool          SpellSlotPool
}

// UseHolyNimbusResult is the closed set of outcomes of UseHolyNimbus.Run. Variants: UseHolyNimbusApplied, UseHolyNimbusRejected,
// UseHolyNimbusNotAPaladin.
type UseHolyNimbusResult interface{ isUseHolyNimbusResult() }

// UseHolyNimbusApplied is the outcome of an accepted call. Only Run builds one.
type UseHolyNimbusApplied struct {
	character    Character
	effect       HolyNimbusEffect
	hasEffect    bool
	spellcasting SpellcastingState
	state        PaladinState
	events       []Event
}

// Character returns the new Character.
func (r UseHolyNimbusApplied) Character() Character { return r.character }

// Effect returns the effect data and true for HolyNimbusActivate, and false for HolyNimbusRestoreWithSlot.
func (r UseHolyNimbusApplied) Effect() (HolyNimbusEffect, bool) { return r.effect, r.hasEffect }

// Spellcasting returns the SpellcastingState with the slot expended and true for HolyNimbusRestoreWithSlot, and false otherwise.
func (r UseHolyNimbusApplied) Spellcasting() (SpellcastingState, bool) {
	return r.spellcasting, !r.spellcasting.IsZero()
}

// State returns the new PaladinState.
func (r UseHolyNimbusApplied) State() PaladinState { return r.state }

// Events returns a new slice of the events: HolyNimbusActivated, or SpellSlotExpended then HolyNimbusRestored.
func (r UseHolyNimbusApplied) Events() []Event { return slices.Clone(r.events) }

// UseHolyNimbusRejected is the outcome of a call a rule refused: nothing changes and there are no events. Only Run builds one.
type UseHolyNimbusRejected struct{ reasons []PaladinReason }

// Reasons returns every reason that holds in declaration order, as a new slice. It is never empty.
func (r UseHolyNimbusRejected) Reasons() []PaladinReason { return slices.Clone(r.reasons) }

// UseHolyNimbusNotAPaladin is the outcome for a character with no Paladin levels.
type UseHolyNimbusNotAPaladin struct{}

func (UseHolyNimbusApplied) isUseHolyNimbusResult()     {}
func (UseHolyNimbusRejected) isUseHolyNimbusResult()    {}
func (UseHolyNimbusNotAPaladin) isUseHolyNimbusResult() {}

// UseHolyNimbus is the use case that activates the Oath of Devotion's Holy Nimbus or restores its use with a level 5 spell slot
// (SRD 5.2.1, "Level 20: Holy Nimbus", p. 57). It spends no Channel Divinity use. Create it with NewUseHolyNimbus; the zero value
// is not usable. It is safe for concurrent use only if its telemetry and clock are.
type UseHolyNimbus struct{ paladinUse }

// NewUseHolyNimbus returns the use case. Options: WithTelemetry, WithClock; the others are ignored. It returns an error matching
// srderr.ErrInvalidInput for the zero progression.
func NewUseHolyNimbus(p PaladinProgression, opts ...Option) (UseHolyNimbus, error) {
	u, err := newPaladinUse("use Holy Nimbus", p, opts)
	return UseHolyNimbus{u}, err
}

// Run activates Holy Nimbus or restores its use. PaladinFeatureNotGained is reported below the level of its entry or without Oath
// of Devotion; activating a spent use is PaladinAlreadyUsedSinceLongRest; restoring an available use is PaladinNotExpended;
// restoring from a pool with no level 5 slot is PaladinNoLevel5SlotAvailable. The slot is expended through
// SpellcastingState.ExpendSlot, so the slot rules are not repeated here.
//
// It returns an error matching srderr.ErrInvalidInput, before any signal, for the zero use case, a zero Character, an undefined
// action or, for the restore, a zero SpellcastingState or an invalid pool, and ctx.Err() unwrapped, before any signal, when ctx is
// already cancelled. A refused call is a result.
func (u UseHolyNimbus) Run(ctx context.Context, cmd UseHolyNimbusCommand) (UseHolyNimbusResult, error) {
	if err := u.precheck(ctx, cmd.Character); err != nil {
		return nil, err
	}
	restore := cmd.Action == HolyNimbusRestoreWithSlot
	if (!restore && cmd.Action != HolyNimbusActivate) || (restore && (cmd.Spellcasting.IsZero() || !cmd.Pool.Valid())) {
		return nil, invalidInput("use Holy Nimbus: undefined action, or a restore without a SpellcastingState or a valid pool")
	}
	ctx, run := u.sig.start(ctx, SpanUseHolyNimbus, ValueClassPaladin, ValueFeatureHolyNimbus)
	defer run.end(ctx)
	level, state, ok, err := u.load(ctx, run, cmd.Character)
	if err != nil {
		return nil, err
	}
	if !ok {
		return UseHolyNimbusNotAPaladin{}, nil
	}
	var reasons []PaladinReason
	if !u.progression.has(PaladinFeatureHolyNimbus, level) || !oathChosen(cmd.Subclass) {
		reasons = append(reasons, PaladinFeatureNotGained)
	}
	if !restore && state.nimbus == LongRestUseSpent {
		reasons = append(reasons, PaladinAlreadyUsedSinceLongRest)
	}
	if restore && state.nimbus == LongRestUseAvailable {
		reasons = append(reasons, PaladinNotExpended)
	}
	var expended SlotExpenditureExpended
	if restore && len(reasons) == 0 {
		switch s := cmd.Spellcasting.ExpendSlot(cmd.Pool, MustSlotLevel(holyNimbusRestoreSlot)).(type) {
		case SlotExpenditureExpended:
			expended = s
		default:
			reasons = append(reasons, PaladinNoLevel5SlotAvailable)
		}
	}
	if len(reasons) > 0 {
		return UseHolyNimbusRejected{reasons: paladinReject(ctx, run, reasons)}, nil
	}
	out := UseHolyNimbusApplied{}
	if restore {
		state.nimbus = LongRestUseAvailable
		out.spellcasting = expended.State()
		out.events = []Event{expended.Event(), HolyNimbusRestored{id: cmd.Character.ID()}}
	} else {
		state.nimbus, state.holyNimbus = LongRestUseSpent, true
		aura := PaladinAuras{}.Compute(u.progression, level, true, cmd.Charisma, cmd.Incapacitated)
		out.effect, out.hasEffect = HolyNimbusEffect{active: aura.active, damage: cmd.Charisma.Value() + cmd.Proficiency.Value()}, true
		out.events = []Event{HolyNimbusActivated{id: cmd.Character.ID()}}
	}
	next, err := paladinStore(run, cmd.Character, state)
	if err != nil {
		return nil, err
	}
	out.character, out.state = next, state
	run.succeeded(ctx)
	return out, nil
}
