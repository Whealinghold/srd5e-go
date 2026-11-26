package combat_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// sumRun builds one call of SummonCreature with the source.
func sumRun(tb testing.TB, source combat.ContentSource, cmd combat.SummonCreatureCommand) func(...combat.Option) (any, error) {
	tb.Helper()
	return func(opts ...combat.Option) (any, error) {
		uc, err := combat.NewSummonCreature(source, opts...)
		if err != nil {
			tb.Fatalf("NewSummonCreature: %v", err)
		}
		res, err := uc.Run(context.Background(), cmd)
		if res == nil {
			return nil, err
		}
		return res, err
	}
}

// sumPanicSource is a content source that panics, a bug of the host's adapter.
type sumPanicSource struct{}

func (sumPanicSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	panic("source bug")
}

// sumOddSource answers a lookup with the entry it holds whatever the key, or with nothing at all.
type sumOddSource struct {
	entry content.Entry
	nil   bool
}

func (s sumOddSource) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	if s.nil {
		return nil, nil //nolint:nilnil // the contract break this double exists to produce
	}
	return content.LookupFound{Entry: s.entry}, nil
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-070, FR-073, FR-074, section 7 - SummonCreature records the shared signals under summon_creature with srd5e.combat.result summoned, form_changed, replaced, control_reasserted, rejected_<reason> (class refused), limit_exceeded or content_missing (class invalid, error status), counts one srd5e.combat.summons for each creature made and one srd5e.combat.summon_endings for each that left, carries no id or count as an attribute, logs nothing and behaves the same with the Noop and the recording telemetry.
// Path: per case.
func TestSummonCreatureSignals(t *testing.T) {
	t.Parallel()
	const op = "summon_creature"
	insect := combat.GiantInsectRequest{Creature: cid("c1"), Level: 4, Form: combat.InsectWasp}
	steed := sumDo(t, combat.SummonedRoster{}, "pc", sumSteedRequest("st1", 2), sumScene()).Roster()
	familiar := sumFamiliar(t, combat.SummonedRoster{}, "pc", "fam", combat.FamiliarOriginFindFamiliarSpell)
	undead := sumRaise(t, 1)
	full := func() combat.SummonedRoster {
		r := combat.SummonedRoster{}
		for i := 0; i < 32; i++ {
			r = sumInsect(t, r, "pc", sumLabel("i", i))
		}
		return r
	}()
	deadRequest := combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}
	rows := []struct {
		name, requirement, path string
		source                  combat.ContentSource
		roster                  combat.SummonedRoster
		request                 combat.SummonRequest
		scene                   combat.SummonScene
		class, result           string
		events                  []string
		summons, endings        []string
	}{
		{"insect", "FR-074", "happy", sumSource(t), combat.SummonedRoster{}, insect, sumScene(), "applied", "summoned", []string{"summoned_creature_joined"}, []string{"giant_insect"}, nil},
		{"two objects", "FR-074", "happy", sumSource(t), combat.SummonedRoster{}, sumObjects("a", "b"), sumScene(), "applied", "summoned", []string{"summoned_creature_joined", "summoned_creature_joined"}, []string{"animated_object", "animated_object"}, nil},
		{"a skeleton", "FR-070", "happy", sumSource(t), combat.SummonedRoster{}, deadRequest, sumScene(), "applied", "summoned", []string{"summoned_creature_joined"}, []string{"animated_undead"}, nil},
		{"a familiar", "FR-070", "happy", sumSource(t), combat.SummonedRoster{}, combat.FindFamiliarRequest{Creature: cid("f"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}, sumScene(), "applied", "summoned", []string{"summoned_creature_joined"}, []string{"familiar"}, nil},
		{"a form change", "FR-044", "unhappy (rule interaction)", sumSource(t), familiar, combat.FindFamiliarRequest{Creature: cid("x"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFiend, Origin: combat.FamiliarOriginFindFamiliarSpell}, sumScene(), "applied", "form_changed", []string{"familiar_form_changed"}, nil, nil},
		{"a replaced steed", "FR-056", "unhappy (rule interaction)", sumSource(t), steed, sumSteedRequest("st2", 2), sumScene(), "applied", "replaced", []string{"summoned_creature_departed", "summoned_creature_joined"}, []string{"otherworldly_steed"}, []string{"otherworldly_steed:replaced"}},
		{"control reasserted", "FR-041", "unhappy (rule interaction)", sumSource(t), undead, sumReassert(combat.SummonAnimatedUndead, 3, "u00"), sumScene(), "applied", "control_reasserted", []string{"summon_control_reasserted"}, nil, nil},
		{"the level", "Table R", "unhappy (boundary)", sumSource(t), combat.SummonedRoster{}, combat.GiantInsectRequest{Creature: cid("c1"), Level: 3, Form: combat.InsectWasp}, sumScene(), "refused", "rejected_level_below_spell", nil, nil, nil},
		{"night", "FR-039", "unhappy (state error)", sumSource(t), combat.SummonedRoster{}, combat.CreateUndeadRequest{Level: 6, Corpses: []combat.CorpseTarget{sumGhoul("g", "r", combat.UndeadGhoul)}}, combat.SummonScene{SpaceAvailable: true}, "refused", "rejected_not_night", nil, nil, nil},
		{"nothing to animate", "FR-042", "unhappy (empty case)", sumSource(t), combat.SummonedRoster{}, combat.AnimateDeadRequest{Level: 3}, sumScene(), "refused", "rejected_nothing_to_animate", nil, nil, nil},
		{"the cap", "P10", "unhappy (boundary)", sumSource(t), full, insect, sumScene(), "invalid", "limit_exceeded", nil, nil, nil},
		{"a cycle", "P10", "unhappy (rule interaction)", sumSource(t), sumInsect(t, combat.SummonedRoster{}, "pc", "mid"), combat.GiantInsectRequest{Creature: cid("pc"), Level: 4, Form: combat.InsectWasp}, sumScene(), "refused", "rejected_summon_link_cycle", nil, nil, nil},
		{"content missing", "FR-071", "unhappy (missing content)", sumSource(t, "skeleton"), combat.SummonedRoster{}, deadRequest, sumScene(), "invalid", "content_missing", nil, nil, nil},
	}
	for _, c := range rows {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			summoner := "pc"
			if c.name == "a cycle" {
				summoner = "mid" // the summoned creature tries to summon its own summoner
			}
			cmd := sumCmd(t, c.roster, summoner, c.request, c.scene)
			run := sumRun(t, c.source, cmd)
			got, err := run(opts...)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if plain, err := run(); err != nil || !reflect.DeepEqual(got, plain) {
				t.Errorf("recording bundle and Noop disagree: %#v vs %#v (%v) [FR-074]", got, plain, err)
			}
			outcome := "failure"
			if c.class == "applied" {
				outcome = "success"
			}
			want := map[string]any{"srd5e.combat.operation": op, "srd5e.outcome": outcome, "srd5e.combat.result": c.result, "srd5e.combat.result_class": c.class}
			spans := rec.Tracer.SpansNamed("srd5e.combat." + op)
			if len(spans) != 1 || !spans[0].Ended || (spans[0].Status == telemetry.StatusError) != (c.class == "invalid") || len(spans[0].Errors) != 0 {
				t.Fatalf("spans = %+v, want one ended span with error status only for invalid [FR-074; %s]", spans, c.requirement)
			}
			if sa := attrMap(spans[0].Attrs); !reflect.DeepEqual(sa, want) {
				t.Errorf("span attrs = %v, want %v [FR-074]", sa, want)
			}
			if ops := measured(rec, "srd5e.combat.operations"); len(ops) != 1 || !reflect.DeepEqual(ops[0], want) {
				t.Errorf("operations = %v, want one with %v [FR-074]", ops, want)
			}
			wantKinds := slices.Clone(c.events)
			slices.Sort(wantKinds)
			if wantKinds == nil {
				wantKinds = []string{}
			}
			if counted := values(measured(rec, "srd5e.combat.events"), "srd5e.combat.event"); !slices.Equal(counted, wantKinds) {
				t.Errorf("event kinds counted = %v, want %v [FR-069]", counted, wantKinds)
			}
			summons := values(measured(rec, "srd5e.combat.summons"), "srd5e.combat.summon_kind")
			wantSummons := slices.Clone(c.summons)
			slices.Sort(wantSummons)
			if wantSummons == nil {
				wantSummons = []string{}
			}
			if !slices.Equal(summons, wantSummons) {
				t.Errorf("summons counted = %v, want %v [section 7]", summons, wantSummons)
			}
			var endings []string
			for _, m := range measured(rec, "srd5e.combat.summon_endings") {
				endings = append(endings, m["srd5e.combat.summon_kind"].(string)+":"+m["srd5e.combat.summon_end_cause"].(string))
			}
			if !slices.Equal(endings, c.endings) {
				t.Errorf("endings counted = %v, want %v [section 7]", endings, c.endings)
			}
			if dur := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(dur) != 1 || dur[0].Value != 2 {
				t.Errorf("duration = %+v, want one sample of 2 s", dur)
			}
			if n := len(rec.Meter.MeasurementsNamed("srd5e.combat.errors")); n != 0 {
				t.Errorf("errors = %d, want none for a result [FR-071]", n)
			}
			checkTelemetryHygiene(t, rec)
			for _, m := range rec.Meter.Measurements() {
				for _, at := range m.Attrs {
					for _, id := range []string{"pc", "c1", "a", "b", "u1", "f", "fam", "x", "st1", "st2", "mid", "g"} {
						if at.Value() == id {
							t.Errorf("%s attribute %s carries the id %q [observability.md section 5]", m.Name, at.Key(), id)
						}
					}
				}
			}
		})
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-070, FR-071 - SummonCreature looks up through the content source exactly the monsters the cast needs, once each (none for the expression kinds, a reassert or a simulacrum), and returns the new roster, the specs, the instructions and the events of the cast.
// Path: per case.
func TestSummonCreatureLooksUpWhatTheCastNeeds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		request combat.SummonRequest
		roster  combat.SummonedRoster
		keys    []content.MonsterID
	}{
		{"an insect", combat.GiantInsectRequest{Creature: cid("c"), Level: 4, Form: combat.InsectWasp}, combat.SummonedRoster{}, nil},
		{"a steed", sumSteedRequest("c", 2), combat.SummonedRoster{}, nil},
		{"a simulacrum", sumSimulacrumRequest("c", 7), combat.SummonedRoster{}, nil},
		{"a reassert", sumReassert(combat.SummonAnimatedUndead, 3, "u00"), sumRaise(t, 1), nil},
		{"bones and corpses", combat.AnimateDeadRequest{Level: 5, Targets: []combat.UndeadTarget{sumBones("a", "1"), sumCorpse("b", "2", content.SizeSmall, true), sumBones("c", "3")}}, combat.SummonedRoster{}, []content.MonsterID{"skeleton", "zombie"}},
		{"a familiar", combat.FindFamiliarRequest{Creature: cid("c"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFey, Origin: combat.FamiliarOriginFindFamiliarSpell}, combat.SummonedRoster{}, []content.MonsterID{"cat"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			src := sumSource(t)
			res, err := sumRun(t, src, sumCmd(t, c.roster, "pc", c.request, sumScene()))()
			if err != nil {
				t.Fatal(err)
			}
			a, ok := res.(combat.SummonCreatureApplied)
			if !ok {
				t.Fatalf("result = %#v, want SummonCreatureApplied", res)
			}
			var keys []content.MonsterID
			for _, q := range src.Queries() {
				keys = append(keys, content.MonsterID(q.(content.LookupQuery).Key.ID().String()))
			}
			if !slices.Equal(keys, c.keys) {
				t.Errorf("looked up %v, want %v [FR-070]", keys, c.keys)
			}
			if len(a.Events()) == 0 || a.Roster().Len() == 0 {
				t.Errorf("result = %#v, want events and a roster", a)
			}
		})
	}
	res, _ := sumRun(t, sumSource(t), sumCmd(t, combat.SummonedRoster{}, "pc", sumObjects("a", "b"), combat.SummonScene{InCombat: true}))()
	a := res.(combat.SummonCreatureApplied)
	if len(a.Creatures()) != 2 || len(a.RosterActions()) != 2 || a.Roster().Len() != 2 || len(a.Events()) != 2 {
		t.Errorf("result = %#v, want two specs, two joins, two links and two events", a)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-071 - a monster the source lacks is a SummonCreatureContentMissing result with every missing entry ordered by id and the roster unchanged, not an error.
// Path: unhappy (missing content).
func TestSummonCreatureContentMissing(t *testing.T) {
	t.Parallel()
	roster := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	prior := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	req := combat.AnimateDeadRequest{Level: 5, Targets: []combat.UndeadTarget{sumCorpse("a", "1", content.SizeSmall, true), sumBones("b", "2")}}
	res, err := sumRun(t, sumSource(t, "zombie", "skeleton"), sumCmd(t, roster, "pc", req, sumScene()))()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	m, ok := res.(combat.SummonCreatureContentMissing)
	if !ok {
		t.Fatalf("result = %#v, want SummonCreatureContentMissing", res)
	}
	var ids []string
	for _, x := range m.Missing() {
		if x.Kind() != content.KindMonster {
			t.Errorf("missing kind = %v, want monster", x.Kind())
		}
		ids = append(ids, x.ID().String())
	}
	if !slices.Equal(ids, []string{"skeleton", "zombie"}) || !m.Roster().Equal(prior) || !roster.Equal(prior) {
		t.Errorf("missing = %v, want [skeleton zombie] in order with the roster unchanged [FR-071]", ids)
	}
	m.Missing()[0] = content.MissingContent{}
	if len(m.Missing()) != 2 || m.Missing()[0].ID().String() != "skeleton" {
		t.Errorf("Missing is not a copy [FR-065]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-071, errors-and-results.md FR-031, FR-034, FR-051 - a failing, panicking or contract-breaking content source is returned wrapped once as a *srderr.PortError for port content with the original cause reachable, a nil result, no log and srd5e.combat.errors counted once with error.type content_error and no operation, summon or event counted; a cancellation during a lookup stays visible through errors.Is; the roster is unchanged.
// Path: per case.
func TestSummonCreatureSourceFailures(t *testing.T) {
	t.Parallel()
	cause := errors.New("store down")
	req := combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}
	roster := combat.SummonedRoster{}
	cases := map[string]struct {
		source combat.ContentSource
		is     error
		op     string
	}{
		"fails":                     {sumSource(t).FailLookup(cause), cause, "lookup"},
		"is cancelled":              {sumSource(t).FailLookup(context.Canceled), context.Canceled, "lookup"},
		"panics":                    {sumPanicSource{}, srderr.ErrPortPanic, "lookup"},
		"answers with nothing":      {sumOddSource{nil: true}, srderr.ErrContractViolation, "lookup"},
		"answers with the wrong id": {sumOddSource{entry: sumMonster(t, "zombie", content.CreatureTypeUndead, content.SizeMedium, "1/4", 50)}, srderr.ErrContractViolation, "lookup"},
		"answers with no entry":     {sumOddSource{}, srderr.ErrContractViolation, "lookup"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec, opts := recording()
			res, err := sumRun(t, c.source, sumCmd(t, roster, "pc", req, sumScene()))(opts...)
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || !errors.Is(err, c.is) || pe.Port != "content" || pe.Op != c.op {
				t.Fatalf("= %v, %v, want a nil result and a content PortError reaching %v [FR-071]", res, err, c.is)
			}
			errs := rec.Meter.MeasurementsNamed("srd5e.combat.errors")
			if len(errs) != 1 || attrMap(errs[0].Attrs)["error.type"] != "content_error" || attrMap(errs[0].Attrs)["srd5e.combat.operation"] != "summon_creature" {
				t.Errorf("errors = %+v, want one content_error [errors-and-results FR-031]", errs)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.summon_creature")
			if len(spans) != 1 || !spans[0].Ended || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 {
				t.Errorf("spans = %+v, want one ended span recording the error once", spans)
			}
			for _, name := range []string{"srd5e.combat.operations", "srd5e.combat.events", "srd5e.combat.summons", "srd5e.combat.summon_endings"} {
				if n := len(rec.Meter.MeasurementsNamed(name)); n != 0 {
					t.Errorf("%s counted %d after a port failure, want none", name, n)
				}
			}
			if got := rec.Logger.Records(); len(got) != 0 {
				t.Errorf("logged %v, want nothing", got)
			}
		})
	}
	if !roster.Equal(combat.SummonedRoster{}) {
		t.Errorf("a failing source changed the roster [FR-065]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-067, errors-and-results.md FR-051 - the zero use case, the zero command and a nil source are invalid input before any signal, an already cancelled ctx returns ctx.Err() unwrapped before any signal and any lookup, and a panic in a telemetry backend propagates with the span ended and no error counted.
// Path: unhappy (failure).
func TestSummonCreatureGuards(t *testing.T) {
	t.Parallel()
	req := combat.AnimateDeadRequest{Level: 3, Targets: []combat.UndeadTarget{sumBones("u1", "b1")}}
	cmd := sumCmd(t, combat.SummonedRoster{}, "pc", req, sumScene())
	rec, opts := recording()
	if _, err := combat.NewSummonCreature(nil, opts...); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a nil source error = %v, want srderr.ErrInvalidInput", err)
	}
	var zero combat.SummonCreature
	if res, err := zero.Run(context.Background(), cmd); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero use case = %v, %v, want srderr.ErrInvalidInput", res, err)
	}
	uc, _ := combat.NewSummonCreature(sumSource(t), opts...)
	if res, err := uc.Run(context.Background(), combat.SummonCreatureCommand{}); res != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero command = %v, %v, want srderr.ErrInvalidInput", res, err)
	}
	src := sumSource(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled, _ := combat.NewSummonCreature(src, opts...)
	if res, err := cancelled.Run(ctx, cmd); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled ctx = %v, %v, want a nil result and context.Canceled itself", res, err)
	}
	if n := len(src.Queries()) + len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()); n != 0 {
		t.Errorf("a refused call looked up or recorded %d things, want none", n)
	}
	loud := telemetrytest.NewRecorder()
	bad, _ := combat.NewSummonCreature(sumSource(t), combat.WithTelemetry(telemetry.Telemetry{Tracer: loud.Tracer, Meter: panicMeter{}}))
	func() {
		defer func() {
			if r := recover(); r == nil || r.(error).Error() != "backend bug" {
				t.Errorf("recovered %v, want the original panic", r)
			}
		}()
		_, _ = bad.Run(context.Background(), cmd)
	}()
	if spans := loud.Tracer.SpansNamed("srd5e.combat.summon_creature"); len(spans) != 1 || !spans[0].Ended {
		t.Errorf("spans = %+v, want one ended span after the panic", spans)
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-073, FR-065 - a refusal is SummonCreatureRejected carrying the SummonRejection and the roster that came in, unchanged, with no events; the command copies its request, so changing it after the command was built changes nothing.
// Path: unhappy (immutability).
func TestSummonCreatureRefusalHoldsTheRosterThatCameIn(t *testing.T) {
	t.Parallel()
	roster := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	prior := sumInsect(t, combat.SummonedRoster{}, "pc", "i1")
	res, err := sumRun(t, sumSource(t), sumCmd(t, roster, "pc", combat.GiantInsectRequest{Creature: cid("i1"), Level: 4, Form: combat.InsectWasp}, sumScene()))()
	if err != nil {
		t.Fatal(err)
	}
	r, ok := res.(combat.SummonCreatureRejected)
	if !ok || r.Rejection != (combat.SummonCreatureIDInUseRejection{ID: cid("i1")}) || !r.Roster().Equal(prior) || !roster.Equal(prior) {
		t.Errorf("result = %#v, want SummonCreatureRejected with SummonCreatureIDInUseRejection and the roster unchanged [FR-073]", res)
	}
	req := combat.AnimateObjectsRequest{Level: 5, Objects: []combat.ObjectTarget{{Creature: cid("a"), Size: content.SizeSmall}}}
	cmd := sumCmd(t, combat.SummonedRoster{}, "pc", req, sumCombat(""))
	req.Objects[0].Creature = "mutated"
	res, _ = sumRun(t, sumSource(t), cmd)()
	if id := res.(combat.SummonCreatureApplied).Creatures()[0].Creature().ID; id != "a" {
		t.Errorf("creature = %v, want a: the command copied its request [FR-065]", id)
	}
	a := res.(combat.SummonCreatureApplied)
	events, actions, creatures := a.Events(), a.RosterActions(), a.Creatures()
	events[0], actions[0], creatures[0] = nil, nil, combat.SummonedCreatureSpec{}
	if a.Events()[0] == nil || a.RosterActions()[0] == nil || a.Creatures()[0].SummonKind() == 0 {
		t.Errorf("a returned slice changed the result [FR-065]")
	}
}

// Requirement: doc/specs/summoned-and-companion-creatures.md FR-076 - a familiar made by Pact of the Chain or Wild Companion passes through SummonCreature with the matching FamiliarOrigin, and Wild Companion's departure at a Long Rest is ReportLongRestFinished (FR-053): the origin survives the use case and decides.
// Path: unhappy (overlap).
func TestFamiliarOriginsPassThroughTheUseCase(t *testing.T) {
	t.Parallel()
	pact := combat.FindFamiliarRequest{Creature: cid("fam"), Level: 1, Form: specialForm("imp"), Origin: combat.FamiliarOriginPactOfTheChain}
	wild := combat.FindFamiliarRequest{Creature: cid("fam"), Level: 1, Form: normalForm("cat"), Type: content.CreatureTypeFiend, Origin: combat.FamiliarOriginWildCompanion}
	for _, c := range []struct {
		name    string
		request combat.FindFamiliarRequest
		typ     content.CreatureType
		leaves  bool
	}{{"Pact of the Chain", pact, content.CreatureTypeFiend, false}, {"Wild Companion", wild, content.CreatureTypeFey, true}} {
		res, err := sumRun(t, sumSource(t), sumCmd(t, combat.SummonedRoster{}, "pc", c.request, sumScene()))()
		if err != nil {
			t.Fatal(err)
		}
		a := res.(combat.SummonCreatureApplied)
		l, _ := a.Roster().Link("fam")
		st, _ := l.Familiar()
		if st.Origin() != c.request.Origin || st.CreatureType() != c.typ {
			t.Errorf("%s familiar = %+v, want origin %v and type %v [FR-076]", c.name, st, c.request.Origin, c.typ)
		}
		uc := combat.NewApplySummonedChange()
		change, _ := combat.NewApplySummonedChangeCommand(a.Roster(), combat.ReportLongRestFinished{Summoner: cid("pc")}, "")
		got, err := uc.Run(context.Background(), change)
		if err != nil {
			t.Fatal(err)
		}
		_, departed := got.(combat.ApplySummonedChangeApplied)
		if departed != c.leaves {
			t.Errorf("%s: a Long Rest departed it = %v, want %v [FR-053]", c.name, departed, c.leaves)
		}
	}
}
