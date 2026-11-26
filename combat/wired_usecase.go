package combat

import "context"

// The use cases of this file's family wrap the pure services of the domain-only specs (combat-action-resolution.md
// FR-167, FR-168): readying a spell, telepathic contact, persistent zones, shape-changing spells and the revival and
// restoration spells, with Dispel Magic, Counterspell and Antimagic Field (their own files). Each holds a command with
// the wrapped operation's arguments in the owner's order, no Encounter and no ActionLedger, has one Run method, runs in
// one Instrumenter operation (FR-170) and returns the owner's own closed result unchanged. Only DispelOngoingSpells
// and CounterCastSpell roll a die. The state a result returns (held spell, contact, zone, registry, transformation,
// ordeal, ward) is the host's: the root has no method to commit it (Q-51). A use case holds no rule of the owner and
// calls no other use case. An error matching srderr.ErrInvalidInput from the owner is returned as it is, recorded on
// the span only; a cancelled ctx is returned unwrapped before any signal. Each is safe for concurrent use if its
// telemetry and clock are.

// wired runs work under op and classifies its result with describe (FR-170). A cancelled ctx is returned before any
// signal.
func wired[R any](ctx context.Context, in instrumenter, op combatOperation, work func() (R, error), describe func(R) description) (R, error) {
	var zero R
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return run(ctx, in, op, func(context.Context) (R, error) { return work() }, describe)
}

// newWired returns the instrumenter of a wired use case for the options.
func newWired(opts []Option) instrumenter {
	o := newOptions(opts)
	return newInstrumenter(o.tel, o.clock)
}

// appliedWith, refusedWith and invalidWith describe a result of the class with its srd5e.combat.result name and events.
func appliedWith(name string, events ...Event) description {
	return description{Class: resultApplied, Name: name, Events: events}
}

func refusedWith(name string) description { return description{Class: resultRefused, Name: name} }

func invalidWith(name string) description { return description{Class: resultInvalid, Name: name} }
