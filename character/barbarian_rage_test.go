package character_test

import (
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

func enterRage(t *testing.T, level int, state character.BarbarianState, turn character.BarbarianTurnState, facts character.BarbarianHostFacts, sub content.SubclassID, speed int) character.RageEntry {
	t.Helper()
	return must(character.BarbarianRage{}.Enter(brbID(t), brbStats(t, level), state, turn, facts, sub, speed))
}

// Requirement: doc/specs/class-barbarian.md FR-021, FR-022, FR-023, FR-024, FR-087, FR-095, FR-125 - entering a Rage is refused with the first reason of: no Barbarian levels, a Rage
// already active, Heavy armor, Incapacitated, no Bonus Action, no use remaining; a refusal spends nothing.
// Path: per case (unhappy: state error, rule interaction).
func TestBarbarianRageEnterRefusals(t *testing.T) {
	t.Parallel()
	spentTurn := character.NewBarbarianTurn().SpendBonusAction()
	for _, c := range []struct {
		name  string
		level int
		state character.BarbarianState
		turn  character.BarbarianTurnState
		facts character.BarbarianHostFacts
		want  character.BarbarianReason
		zero  bool
	}{
		{"no Barbarian levels", 1, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, character.BarbarianNotABarbarian, true},
		{"a Rage already active", 3, brbRaging(t, 3, 0), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, character.BarbarianRageAlreadyActive, false},
		{"Heavy armor", 3, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{WearingHeavyArmor: true}, character.BarbarianWearingHeavyArmor, false},
		{"Incapacitated", 3, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{Incapacitated: true}, character.BarbarianIncapacitated, false},
		{"no Bonus Action", 3, character.InitialBarbarianState(), spentTurn, character.BarbarianHostFacts{}, character.BarbarianNoBonusAction, false},
		{"the zero turn state has no Bonus Action", 3, character.InitialBarbarianState(), character.BarbarianTurnState{}, character.BarbarianHostFacts{}, character.BarbarianNoBonusAction, false},
		{"no use remaining", 3, character.MustBarbarianState(3, 0), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, character.BarbarianNoRageUsesRemaining, false},
		{"Heavy armor beats Incapacitated", 3, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{WearingHeavyArmor: true, Incapacitated: true}, character.BarbarianWearingHeavyArmor, false},
		{"Incapacitated beats the Bonus Action", 3, character.InitialBarbarianState(), spentTurn, character.BarbarianHostFacts{Incapacitated: true}, character.BarbarianIncapacitated, false},
		{"the Bonus Action beats the uses", 3, character.MustBarbarianState(3, 0), spentTurn, character.BarbarianHostFacts{}, character.BarbarianNoBonusAction, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := brbStats(t, c.level)
			if c.zero {
				stats = character.BarbarianLevelStats{}
			}
			res := must(character.BarbarianRage{}.Enter(brbID(t), stats, c.state, c.turn, c.facts, "", 30))
			if got, ok := res.(character.RageNotEntered); !ok || got.Reason != c.want {
				t.Errorf("Enter = %#v, want RageNotEntered %v", res, c.want)
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-023, FR-037, FR-083, FR-120, FR-127 - entering a Rage expends exactly one use, starts the Rage, spends the Bonus Action, produces RageStarted
// for the character and reports that Concentration ends; the inputs are unchanged and the events are a copy.
// Path: happy.
func TestBarbarianRageEnter(t *testing.T) {
	t.Parallel()
	id := brbID(t)
	in := character.MustBarbarianState(1, 2)
	turn := character.NewBarbarianTurn()
	res := enterRage(t, 3, in, turn, character.BarbarianHostFacts{}, "", 30).(character.RageEntered)
	if res.State().RagesExpended() != 2 || !res.State().RageActive() || res.State().RelentlessRageUses() != 2 || res.Turn().BonusActionAvailable() || !res.Turn().ReactionAvailable() || !res.ConcentrationEnds() {
		t.Errorf("entered = %#v", res)
	}
	if in.RagesExpended() != 1 || in.RageActive() || !turn.BonusActionAvailable() {
		t.Error("Enter altered its inputs")
	}
	ev := res.Events()
	if started, ok := ev[0].(character.RageStarted); len(ev) != 1 || !ok || started.CharacterID() != id {
		t.Errorf("events = %#v", ev)
	}
	ev[0] = nil
	if res.Events()[0] == nil {
		t.Error("Events returned a shared slice")
	}
	if _, err := (character.BarbarianRage{}).Enter(id, brbStats(t, 3), in, turn, character.BarbarianHostFacts{}, "", -1); err == nil {
		t.Error("a negative Speed must be refused")
	} else {
		wantInvalid(t, err, "negative Speed")
	}
	_, err := character.BarbarianRage{}.Enter(character.Identifier{}, brbStats(t, 3), in, turn, character.BarbarianHostFacts{}, "", 30)
	wantInvalid(t, err, "zero id")
}

// Requirement: doc/specs/class-barbarian.md FR-059 and table 4.7.1 (HS-01 to HS-05) - from level 7 entering a Rage reports an allowance to move up to half the Speed, rounded
// down, with the Speed it came from; not at level 6.
// Path: per case (HS-01 to HS-05 happy / unhappy: boundary).
func TestBarbarianRagePounce(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		label       string
		speed, half int
	}{{"HS-01", 30, 15}, {"HS-02", 25, 12}, {"HS-03", 0, 0}, {"HS-04", 1, 0}, {"HS-05", 40, 20}} {
		p := enterRage(t, 7, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, "", c.speed).(character.RageEntered).Pounce()
		if !p.Allowed() || p.Speed() != c.speed || p.HalfSpeed() != c.half {
			t.Errorf("%s: pounce = %+v, want allowed with half %d", c.label, p, c.half)
		}
	}
	p := enterRage(t, 6, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{}, "", 30).(character.RageEntered).Pounce()
	if p.Allowed() || p.HalfSpeed() != 0 || p.Speed() != 0 {
		t.Errorf("level 6 pounce = %+v, want none", p)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-079, FR-122 - from level 6 the Path of the Berserker reports Charmed and Frightened ended when entering a Rage, in condition order and
// only those the Barbarian has; below level 6, with another or no subclass or with other conditions nothing is reported (an empty slice, a copy).
// Path: per case (unhappy: boundary, wrong subclass).
func TestBarbarianRageMindlessRageEndsConditions(t *testing.T) {
	t.Parallel()
	both := []content.ConditionKind{content.ConditionFrightened, content.ConditionCharmed, content.ConditionPoisoned}
	for _, c := range []struct {
		name  string
		level int
		sub   content.SubclassID
		have  []content.ConditionKind
		want  []content.ConditionKind
	}{
		{"both at level 6", 6, character.BerserkerSubclassID, both, []content.ConditionKind{content.ConditionCharmed, content.ConditionFrightened}},
		{"only Frightened", 6, character.BerserkerSubclassID, []content.ConditionKind{content.ConditionFrightened}, []content.ConditionKind{content.ConditionFrightened}},
		{"only Charmed", 6, character.BerserkerSubclassID, []content.ConditionKind{content.ConditionCharmed}, []content.ConditionKind{content.ConditionCharmed}},
		{"level 5", 5, character.BerserkerSubclassID, both, nil},
		{"another subclass", 6, "wild_heart", both, nil},
		{"no subclass", 20, "", both, nil},
		{"other conditions", 6, character.BerserkerSubclassID, []content.ConditionKind{content.ConditionPoisoned}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := enterRage(t, c.level, character.InitialBarbarianState(), character.NewBarbarianTurn(), character.BarbarianHostFacts{ActiveConditions: c.have}, c.sub, 30).(character.RageEntered)
			got := res.EndedConditions()
			if got == nil || len(got) != len(c.want) {
				t.Fatalf("ended = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("ended = %v, want %v", got, c.want)
				}
			}
			if len(got) > 0 {
				got[0] = 0
				if res.EndedConditions()[0] == 0 {
					t.Error("EndedConditions returned a shared slice")
				}
			}
		})
	}
}

// Requirement: doc/specs/class-barbarian.md FR-031, FR-032, FR-033 - an extension: a Rage that began this turn, one that is not active, one already extended this turn, a zero-stats Barbarian
// and (with a spent Bonus Action) the Bonus Action trigger return a no-effect result and change nothing; each of the three triggers is accepted on the next turn, the Bonus Action
// trigger spends the Bonus Action, and an unknown trigger is an error.
// Path: per case (happy / unhappy: state error).
func TestBarbarianRageExtend(t *testing.T) {
	t.Parallel()
	later := brbRagingLater(t, 5)
	fresh := character.NewBarbarianTurn()
	extendedAlready := must(character.BarbarianRage{}.Extend(brbStats(t, 5), later, fresh, character.RageExtendedByAttackRoll)).(character.RageExtended).State()
	for _, c := range []struct {
		name    string
		level   int
		state   character.BarbarianState
		turn    character.BarbarianTurnState
		trigger character.RageExtension
		want    character.BarbarianReason
	}{
		{"on the turn it began", 5, brbRaging(t, 5, 0), fresh, character.RageExtendedByAttackRoll, character.BarbarianRageBeganThisTurn},
		{"with no Rage", 5, character.InitialBarbarianState(), fresh, character.RageExtendedByAttackRoll, character.BarbarianRageNotActive},
		{"already extended", 5, extendedAlready, fresh, character.RageExtendedByForcingSave, character.BarbarianRageAlreadyExtended},
		{"Bonus Action trigger without a Bonus Action", 5, later, fresh.SpendBonusAction(), character.RageExtendedByBonusAction, character.BarbarianNoBonusAction},
		{"Persistent Rage needs none", 15, brbRagingLater(t, 15), fresh, character.RageExtendedByAttackRoll, character.BarbarianRageLastsWithoutExtension},
		{"no Barbarian levels", 0, later, fresh, character.RageExtendedByAttackRoll, character.BarbarianNotABarbarian},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			stats := brbStats(t, max(c.level, 1))
			if c.level == 0 {
				stats = character.BarbarianLevelStats{}
			}
			res := must(character.BarbarianRage{}.Extend(stats, c.state, c.turn, c.trigger))
			if got, ok := res.(character.RageAdvanceIgnored); !ok || got.Reason != c.want {
				t.Errorf("Extend = %#v, want ignored %v", res, c.want)
			}
		})
	}
	for _, trigger := range []character.RageExtension{character.RageExtendedByAttackRoll, character.RageExtendedByForcingSave, character.RageExtendedByBonusAction} {
		res := must(character.BarbarianRage{}.Extend(brbStats(t, 5), later, fresh, trigger)).(character.RageExtended)
		wantBonus := trigger != character.RageExtendedByBonusAction
		if !res.State().RageActive() || res.Turn().BonusActionAvailable() != wantBonus {
			t.Errorf("trigger %d: state %v turn %v", trigger, res.State(), res.Turn())
		}
		end := must(character.BarbarianRage{}.End(brbID(t), brbStats(t, 5), res.State(), character.RageTurnEnded{})).(character.RageEndApplied)
		if !end.State().RageActive() || len(end.Events()) != 0 {
			t.Errorf("trigger %d: the extended Rage must survive the end of the turn", trigger)
		}
	}
	for _, trigger := range []character.RageExtension{0, 4} {
		_, err := character.BarbarianRage{}.Extend(brbStats(t, 5), later, fresh, trigger)
		wantInvalid(t, err, "unknown trigger")
	}
}

// Requirement: doc/specs/class-barbarian.md FR-031, FR-037 - at the end of the Barbarian's own turn a Rage that began that turn stays and a later one ends with the reason not extended
// unless it was extended; the end produces a RageEnded event for the character; the marks are cleared so the Rage then needs an extension.
// Path: per case (happy / unhappy: state error).
func TestBarbarianRageTurnEnded(t *testing.T) {
	t.Parallel()
	id := brbID(t)
	stats := brbStats(t, 5)
	began := brbRaging(t, 5, 0)
	end := must(character.BarbarianRage{}.End(id, stats, began, character.RageTurnEnded{})).(character.RageEndApplied)
	if !end.State().RageActive() || len(end.Events()) != 0 {
		t.Fatalf("the Rage that began this turn must stay: %v", end.State())
	}
	next := must(character.BarbarianRage{}.End(id, stats, end.State(), character.RageTurnEnded{})).(character.RageEndApplied)
	ended, ok := next.Events()[0].(character.RageEnded)
	if next.State().RageActive() || len(next.Events()) != 1 || !ok || ended.Reason() != character.RageNotExtended || ended.CharacterID() != id {
		t.Errorf("a Rage not extended must end: %v %#v", next.State(), next.Events())
	}
	if again := must(character.BarbarianRage{}.End(id, stats, next.State(), character.RageTurnEnded{})); again != (character.RageAdvanceIgnored{Reason: character.BarbarianRageNotActive}) {
		t.Errorf("ending an inactive Rage = %#v", again)
	}
}

// Requirement: doc/specs/class-barbarian.md FR-034, FR-035, FR-036, FR-037, FR-072, FR-096 - the early ends: Heavy armor always ends a Rage; Incapacitated ends it below level 15 and not with
// Persistent Rage; Unconscious ends it with reason incapacitated below level 15 and unconscious from level 15; 10 minutes ends it, 9:59 does not; the event names the reason.
// Path: per case (unhappy: rule interaction, boundary).
func TestBarbarianRageEnd(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		level int
		cause character.RageEndCause
		want  character.RageEndReason // 0: the Rage goes on
	}{
		{"Heavy armor at 14", 14, character.RageHeavyArmorDonned{}, character.RageEndedByHeavyArmor},
		{"Heavy armor at 15", 15, character.RageHeavyArmorDonned{}, character.RageEndedByHeavyArmor},
		{"Incapacitated at 14", 14, character.RageBecameIncapacitated{}, character.RageEndedByIncapacitated},
		{"Incapacitated at 15", 15, character.RageBecameIncapacitated{}, 0},
		{"Unconscious at 14", 14, character.RageBecameUnconscious{}, character.RageEndedByIncapacitated},
		{"Unconscious at 15", 15, character.RageBecameUnconscious{}, character.RageEndedByUnconscious},
		{"9:59 at 3", 3, character.RageElapsedTime{Elapsed: 10*time.Minute - time.Second}, 0},
		{"10:00 at 3", 3, character.RageElapsedTime{Elapsed: 10 * time.Minute}, character.RageEndedByTenMinutes},
		{"10:01 at 15", 15, character.RageElapsedTime{Elapsed: 10*time.Minute + time.Second}, character.RageEndedByTenMinutes},
		{"nothing elapsed", 3, character.RageElapsedTime{}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			state := brbRaging(t, c.level, 0)
			res := must(character.BarbarianRage{}.End(brbID(t), brbStats(t, c.level), state, c.cause)).(character.RageEndApplied)
			if c.want == 0 {
				if res.State() != state || len(res.Events()) != 0 {
					t.Errorf("the Rage must go on: %v %v", res.State(), res.Events())
				}
				return
			}
			ev, ok := res.Events()[0].(character.RageEnded)
			if res.State().RageActive() || len(res.Events()) != 1 || !ok || ev.Reason() != c.want {
				t.Errorf("= %v %#v, want ended with %v", res.State(), res.Events(), c.want)
			}
		})
	}
	stats, id := brbStats(t, 3), brbID(t)
	if got := must(character.BarbarianRage{}.End(id, stats, character.InitialBarbarianState(), character.RageHeavyArmorDonned{})); got != (character.RageAdvanceIgnored{Reason: character.BarbarianRageNotActive}) {
		t.Errorf("ending with no Rage = %#v", got)
	}
	if got := must(character.BarbarianRage{}.End(id, character.BarbarianLevelStats{}, brbRaging(t, 3, 0), character.RageHeavyArmorDonned{})); got != (character.RageAdvanceIgnored{Reason: character.BarbarianNotABarbarian}) {
		t.Errorf("ending with zero stats = %#v", got)
	}
	for _, bad := range []character.RageEndCause{nil, character.RageElapsedTime{Elapsed: -time.Second}} {
		_, err := character.BarbarianRage{}.End(id, stats, brbRaging(t, 3, 0), bad)
		wantInvalid(t, err, "bad cause")
	}
	_, err := character.BarbarianRage{}.End(character.Identifier{}, stats, brbRaging(t, 3, 0), character.RageTurnEnded{})
	wantInvalid(t, err, "zero id")
}

// Requirement: doc/specs/class-barbarian.md FR-072 - with Persistent Rage a Rage lasts through the end of the turn without an extension.
// Path: unhappy (rule interaction).
func TestBarbarianPersistentRageLastsThroughTurns(t *testing.T) {
	t.Parallel()
	stats, id := brbStats(t, 15), brbID(t)
	state := brbRaging(t, 15, 0)
	for range 3 {
		res := must(character.BarbarianRage{}.End(id, stats, state, character.RageTurnEnded{})).(character.RageEndApplied)
		state = res.State()
		if !state.RageActive() || len(res.Events()) != 0 {
			t.Fatalf("Persistent Rage ended at the end of a turn: %v", state)
		}
	}
}
