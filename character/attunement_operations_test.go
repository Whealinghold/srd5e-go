package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type attuneCase struct {
	name, requirement, path string
	set                     character.AttunementSet
	item                    character.AttunableItem
	who                     character.AttuneCandidate
	judgement               character.PrerequisiteJudgement
	focus                   character.ShortRestFocus
	limit                   int
	want                    character.AttuneRejection // 0 accepted
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-040 to FR-043, FR-047 to FR-051 and table "Attunement limit").
// Path: per case.
func TestAttunementSetAttune(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Attune during a Short Rest", "No More Than Three Items", p. 102`
	req := content.AttunementRequired{}
	good := focusOf(t, "n", character.RestPurposeAttune, character.InContact, true)
	item := attunable(t, "new_item", "n", req)
	three := attunedSet(t, "a", "a1", "b", "b1", "c", "c1")
	four := attunedSet(t, "a", "a1", "b", "b1", "c", "c1", "d", "d1")
	cases := []attuneCase{
		{"none attuned", p, "unhappy (empty case)", attunedSet(t), item, candidate(t, character.CanCast), 0, good, 3, 0},
		{"two attuned fills the set", p, "unhappy (boundary)", attunedSet(t, "a", "a1", "b", "b1"), item, candidate(t, character.CanCast), 0, good, 3, 0},
		{"three attuned", p, "unhappy (boundary)", three, item, candidate(t, character.CanCast), 0, good, 3, character.AttuneLimitReached},
		{"limit 4, three attuned", p, "unhappy (boundary)", three, item, candidate(t, character.CanCast), 0, good, 4, 0},
		{"limit 4, four attuned", p, "unhappy (boundary)", four, item, candidate(t, character.CanCast), 0, good, 4, character.AttuneLimitReached},
		{"limit dropped below the count", p, "unhappy (boundary)", four, item, candidate(t, character.CanCast), 0, good, 3, character.AttuneLimitReached},
		{"interrupted rest", p, "unhappy (state error)", attunedSet(t), item, candidate(t, character.CanCast), 0, focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), 3, character.AttuneRestInterrupted},
		{"learning rest", p, "unhappy (state error)", attunedSet(t), item, candidate(t, character.CanCast), 0, focusOf(t, "n", character.RestPurposeLearnProperties, character.InContact, true), 3, character.AttuneWrongRestPurpose},
		{"ending rest", p, "unhappy (state error)", attunedSet(t), item, candidate(t, character.CanCast), 0, focusOf(t, "n", character.RestPurposeEndAttunement, character.InContact, true), 3, character.AttuneWrongRestPurpose},
		{"other instance", p, "unhappy (invalid input)", attunedSet(t), item, candidate(t, character.CanCast), 0, focusOf(t, "other", character.RestPurposeAttune, character.InContact, true), 3, character.AttuneRestNotFocusedOnItem},
		{"no contact", p, "unhappy (invalid input)", attunedSet(t), item, candidate(t, character.CanCast), 0, focusOf(t, "n", character.RestPurposeAttune, character.NotInContact, true), 3, character.AttuneRestNotFocusedOnItem},
		{"already attuned to the instance", p, "unhappy (state error)", attunedSet(t, "new_item", "n"), item, candidate(t, character.CanCast), 0, good, 3, character.AttuneAlreadyAttunedToItem},
		{"another copy", p, "unhappy (state error)", attunedSet(t, "new_item", "n2"), item, candidate(t, character.CanCast), 0, good, 3, character.AttuneDuplicateCopy},
		{"prerequisite not met", p, "unhappy (rule interaction)", attunedSet(t), attunable(t, "new_item", "n", requiredBy(t, "a Wizard")), candidate(t, character.CanCast, "fighter"), 0, good, 3, character.AttunePrerequisiteNotMet},
		{"prerequisite met", p, "happy", attunedSet(t), attunable(t, "new_item", "n", requiredBy(t, "a Wizard")), candidate(t, character.CanCast, "wizard"), 0, good, 3, 0},
		{"described, judgement missing", p, "unhappy (missing content)", attunedSet(t), attunable(t, "new_item", "n", requiredBy(t, "a Dwarf")), candidate(t, character.CanCast), 0, good, 3, character.AttunePrerequisiteJudgementMissing},
		{"described, judged met", p, "unhappy (missing content)", attunedSet(t), attunable(t, "new_item", "n", requiredBy(t, "a Dwarf")), candidate(t, character.CanCast), character.PrerequisiteMet, good, 3, 0},
		{"described, judged not met", p, "unhappy (missing content)", attunedSet(t), attunable(t, "new_item", "n", requiredBy(t, "a Dwarf")), candidate(t, character.CanCast), character.PrerequisiteNotMet, good, 3, character.AttunePrerequisiteNotMet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			before := c.set.Items()
			next, ev, rej, err := c.set.Attune(holder, c.item, c.who, c.judgement, c.focus, character.AttunementLimit(c.limit))
			if err != nil || rej != c.want {
				t.Fatalf("rejection = %v, %v, want %v [%s]", rej, err, c.want, c.requirement)
			}
			if !c.set.Equal(attunedSetOf(t, before)) {
				t.Error("input changed")
			}
			if c.want != 0 {
				if !next.Equal(c.set) || ev != (character.MagicItemAttuned{}) {
					t.Errorf("rejected attempt changed the set or raised %+v", ev)
				}
				return
			}
			if !next.IsAttunedTo(c.item.InstanceID()) || next.Len() != c.set.Len()+1 || ev.CharacterID() != holder || ev.ItemID() != c.item.ItemID() || ev.InstanceID() != c.item.InstanceID() {
				t.Errorf("accepted: %v, %+v", next.Items(), ev)
			}
		})
	}
}

func attunedSetOf(t *testing.T, items []character.AttunedItem) character.AttunementSet {
	t.Helper()
	return must(character.NewAttunementSet(items))
}

// Requirement: doc/specs/magic-item-rules.md FR-051 - when several rejections apply the first in the order wrongRestPurpose,
// restNotFocusedOnItem, alreadyAttunedToItem, duplicateCopy, prerequisiteJudgementMissing, prerequisiteNotMet, limitReached,
// restInterrupted wins; tested for each adjacent pair and with all eight true.
// Path: unhappy (rule interaction).
func TestAttuneRejectionOrder(t *testing.T) {
	t.Parallel()
	described := requiredBy(t, "a Dwarf")
	item := attunable(t, "x", "n", described)
	// worlds[i] makes rejections i.. true and the earlier ones false, except where noted.
	set := attunedSet(t, "x", "n", "x", "other", "p", "p1", "q", "q1") // attuned to n, holds a copy of x, four items
	who := candidate(t, character.CanCast)
	worst := focusOf(t, "elsewhere", character.RestPurposeLearnProperties, character.NotInContact, false)
	_, _, rej, err := set.Attune(holder, item, who, 0, worst, 3)
	if err != nil || rej != character.AttuneWrongRestPurpose {
		t.Fatalf("all true: %v, %v", rej, err)
	}
	steps := []struct {
		focus character.ShortRestFocus
		set   character.AttunementSet
		judge character.PrerequisiteJudgement
		limit int
		want  character.AttuneRejection
	}{
		{focusOf(t, "elsewhere", character.RestPurposeAttune, character.NotInContact, false), set, 0, 3, character.AttuneRestNotFocusedOnItem},
		{focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), set, 0, 3, character.AttuneAlreadyAttunedToItem},
		{focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), attunedSet(t, "x", "other", "p", "p1", "q", "q1"), 0, 3, character.AttuneDuplicateCopy},
		{focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), attunedSet(t, "p", "p1", "q", "q1", "r", "r1"), 0, 3, character.AttunePrerequisiteJudgementMissing},
		{focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), attunedSet(t, "p", "p1", "q", "q1", "r", "r1"), character.PrerequisiteNotMet, 3, character.AttunePrerequisiteNotMet},
		{focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), attunedSet(t, "p", "p1", "q", "q1", "r", "r1"), character.PrerequisiteMet, 3, character.AttuneLimitReached},
		{focusOf(t, "n", character.RestPurposeAttune, character.InContact, false), attunedSet(t, "p", "p1"), character.PrerequisiteMet, 3, character.AttuneRestInterrupted},
	}
	for i, s := range steps {
		if _, _, rej, err := s.set.Attune(holder, item, who, s.judge, s.focus, character.AttunementLimit(s.limit)); err != nil || rej != s.want {
			t.Errorf("step %d: %v, %v, want %v", i, rej, err, s.want)
		}
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-040, FR-120 - invalid input is refused with srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func TestAttunementSetAttuneInvalidInput(t *testing.T) {
	t.Parallel()
	item := attunable(t, "x", "n", content.AttunementRequired{})
	good := focusOf(t, "n", character.RestPurposeAttune, character.InContact, true)
	who := candidate(t, character.CanCast)
	s := attunedSet(t)
	for name, run := range map[string]func() error{
		"zero holder":    func() error { _, _, _, e := s.Attune(character.Identifier{}, item, who, 0, good, 3); return e },
		"zero item":      func() error { _, _, _, e := s.Attune(holder, character.AttunableItem{}, who, 0, good, 3); return e },
		"zero focus":     func() error { _, _, _, e := s.Attune(holder, item, who, 0, character.ShortRestFocus{}, 3); return e },
		"limit below 3":  func() error { _, _, _, e := s.Attune(holder, item, who, 0, good, 2); return e },
		"zero candidate": func() error { _, _, _, e := s.Attune(holder, item, character.AttuneCandidate{}, 0, good, 3); return e },
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Requirement: per case (doc/specs/magic-item-rules.md FR-056 to FR-059, FR-115 to FR-118 - voluntary end, Remove Curse,
// Greater Restoration and the ending triggers).
// Path: per case.
func TestAttunementSetEnd(t *testing.T) {
	t.Parallel()
	const p = `SRD 5.2.1, "Ending Attunement", pp. 102-103; "Remove Curse", p. 158; "Greater Restoration", p. 137`
	set := attunedSet(t, "ring", "r1", "cloak", "c1")
	rest := func(instance string, purpose character.RestPurpose, contact character.PhysicalContact, done bool) character.EndMeans {
		return character.EndByShortRest{Focus: focusOf(t, instance, purpose, contact, done)}
	}
	spell := func(id content.SpellID) character.EndMeans { return character.EndBySpell{Spell: id} }
	trig := func(tr character.AttunementEndTrigger) character.EndMeans { return character.EndByTrigger{Trigger: tr} }
	dist := func(feet, hours int) character.EndMeans {
		return trig(must(character.NewAttunementEndDistanceCheck(feet, hours)))
	}
	pre := func(who character.AttuneCandidate) character.EndMeans {
		return trig(must(character.NewAttunementEndPrerequisiteCheck(content.SpellcasterPrerequisite{}, who, 0)))
	}
	end := func(r character.AttunementEndReason) []character.AttunementEndReason {
		return []character.AttunementEndReason{r}
	}
	cases := []struct {
		name         string
		instance     string
		curse        content.CurseStatus
		broken       character.CurseBreak
		means        character.EndMeans
		want         character.EndRejection
		reasons      []character.AttunementEndReason
		remainingLen int
	}{
		{"voluntary", "r1", content.NotCursed, 0, rest("r1", character.RestPurposeEndAttunement, character.InContact, true), 0, end(character.AttunementEndedByShortRest), 1},
		{"voluntary without contact", "r1", content.NotCursed, 0, rest("r1", character.RestPurposeEndAttunement, character.NotInContact, true), 0, end(character.AttunementEndedByShortRest), 1},
		{"cursed, curse intact", "r1", content.Cursed, character.CurseIntact, rest("r1", character.RestPurposeEndAttunement, character.InContact, true), character.EndCurseBlocksEnding, nil, 2},
		{"cursed, curse broken", "r1", content.Cursed, character.CurseBroken, rest("r1", character.RestPurposeEndAttunement, character.InContact, true), 0, end(character.AttunementEndedByShortRest), 1},
		{"interrupted rest", "r1", content.NotCursed, 0, rest("r1", character.RestPurposeEndAttunement, character.InContact, false), character.EndRestInterrupted, nil, 2},
		{"wrong instance", "r1", content.NotCursed, 0, rest("c1", character.RestPurposeEndAttunement, character.InContact, true), character.EndRestNotFocusedOnItem, nil, 2},
		{"wrong purpose", "r1", content.NotCursed, 0, rest("r1", character.RestPurposeAttune, character.InContact, true), character.EndWrongRestPurpose, nil, 2},
		{"not in the set", "zz", content.NotCursed, 0, rest("zz", character.RestPurposeEndAttunement, character.InContact, true), character.EndNotAttuned, nil, 2},
		{"not attuned before wrong purpose", "zz", content.NotCursed, 0, rest("zz", character.RestPurposeAttune, character.InContact, false), character.EndNotAttuned, nil, 2},
		{"wrong purpose before wrong instance", "r1", content.Cursed, 0, rest("c1", character.RestPurposeAttune, character.NotInContact, false), character.EndWrongRestPurpose, nil, 2},
		{"wrong instance before curse", "r1", content.Cursed, 0, rest("c1", character.RestPurposeEndAttunement, character.InContact, false), character.EndRestNotFocusedOnItem, nil, 2},
		{"curse before interruption", "r1", content.Cursed, 0, rest("r1", character.RestPurposeEndAttunement, character.InContact, false), character.EndCurseBlocksEnding, nil, 2},
		{"Remove Curse, cursed", "r1", content.Cursed, 0, spell(character.SpellRemoveCurse), 0, end(character.AttunementEndedCurseBroken), 1},
		{"Greater Restoration, cursed", "r1", content.Cursed, 0, spell(character.SpellGreaterRestoration), 0, end(character.AttunementEndedCurseBroken), 1},
		{"Remove Curse, not cursed", "r1", content.NotCursed, 0, spell(character.SpellRemoveCurse), character.EndNotCursed, nil, 2},
		{"Greater Restoration, not cursed", "r1", content.NotCursed, 0, spell(character.SpellGreaterRestoration), character.EndNotCursed, nil, 2},
		{"other spell", "r1", content.Cursed, 0, spell("identify"), character.EndSpellDoesNotEndAttunement, nil, 2},
		{"other spell before not attuned", "zz", content.Cursed, 0, spell("identify"), character.EndSpellDoesNotEndAttunement, nil, 2},
		{"spell, not attuned before not cursed", "zz", content.NotCursed, 0, spell(character.SpellRemoveCurse), character.EndNotAttuned, nil, 2},
		{"death ends all", "r1", content.NotCursed, 0, trig(character.AttunementEndDied{}), 0, []character.AttunementEndReason{character.AttunementEndedDied, character.AttunementEndedDied}, 0},
		{"another creature attunes", "r1", content.NotCursed, 0, trig(character.AttunementEndAnotherCreatureAttuned{}), 0, end(character.AttunementEndedAnotherCreatureAttuned), 1},
		{"another creature, not attuned", "zz", content.NotCursed, 0, trig(character.AttunementEndAnotherCreatureAttuned{}), character.EndNotAttuned, nil, 2},
		{"far for a day", "r1", content.NotCursed, 0, dist(5000, 24), 0, end(character.AttunementEndedItemTooFarAway), 1},
		{"distance not far enough", "r1", content.NotCursed, 0, dist(50, 1), character.EndEndingConditionNotMet, nil, 2},
		{"distance check, not attuned first", "zz", content.NotCursed, 0, dist(50, 1), character.EndNotAttuned, nil, 2},
		{"prerequisite lost", "r1", content.NotCursed, 0, pre(candidate(t, character.CannotCast)), 0, end(character.AttunementEndedPrerequisiteNoLongerMet), 1},
		{"prerequisite still met", "r1", content.NotCursed, 0, pre(candidate(t, character.CanCast)), character.EndEndingConditionNotMet, nil, 2},
		{"prerequisite check, not attuned first", "zz", content.NotCursed, 0, pre(candidate(t, character.CanCast)), character.EndNotAttuned, nil, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			next, events, rej, err := set.End(holder, inst(c.instance), c.curse, c.broken, c.means)
			if err != nil || rej != c.want || next.Len() != c.remainingLen || len(events) != len(c.reasons) {
				t.Fatalf("rejection %v, err %v, left %d, %d events; want %v, left %d, %v [%s]", rej, err, next.Len(), len(events), c.want, c.remainingLen, c.reasons, p)
			}
			for i, e := range events {
				if e.Reason() != c.reasons[i] || e.CharacterID() != holder || e.ItemID() == "" || e.InstanceID() == "" {
					t.Errorf("event %d = %+v", i, e)
				}
			}
			if c.want != 0 && !next.Equal(set) {
				t.Error("rejected end changed the set")
			}
			if set.Len() != 2 {
				t.Error("input changed")
			}
		})
	}
}

// Requirement: doc/specs/magic-item-rules.md FR-054, FR-116 - death on an empty set returns the empty set and no event;
// a revival restores nothing; invalid input and negative trigger values are refused.
// Path: unhappy (state error).
func TestAttunementSetEndDeathAndInvalidInput(t *testing.T) {
	t.Parallel()
	empty := attunedSet(t)
	next, events, rej, err := empty.End(holder, inst("r1"), content.NotCursed, 0, character.EndByTrigger{Trigger: character.AttunementEndDied{}})
	if err != nil || rej != 0 || next.Len() != 0 || len(events) != 0 {
		t.Errorf("empty set: %v %v %v %v", next.Items(), events, rej, err)
	}
	for name, run := range map[string]func() error{
		"zero holder": func() error {
			_, _, _, e := empty.End(character.Identifier{}, inst("r"), 0, 0, character.EndBySpell{})
			return e
		},
		"blank instance": func() error { _, _, _, e := empty.End(holder, "", 0, 0, character.EndBySpell{}); return e },
		"nil means":      func() error { _, _, _, e := empty.End(holder, inst("r"), 0, 0, nil); return e },
		"nil trigger":    func() error { _, _, _, e := empty.End(holder, inst("r"), 0, 0, character.EndByTrigger{}); return e },
		"negative distance": func() error {
			_, e := character.NewAttunementEndDistanceCheck(-1, 5)
			return e
		},
		"negative hours": func() error { _, e := character.NewAttunementEndDistanceCheck(101, -1); return e },
		"nil prerequisite": func() error {
			_, e := character.NewAttunementEndPrerequisiteCheck(nil, character.AttuneCandidate{}, 0)
			return e
		},
	} {
		if err := run(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A prerequisite check with an invalid class prerequisite surfaces the eligibility error.
	bad := must(character.NewAttunementEndPrerequisiteCheck(content.ClassPrerequisite{}, candidate(t, character.CanCast), 0))
	if _, _, _, err := attunedSet(t, "r", "r1").End(holder, inst("r1"), 0, 0, character.EndByTrigger{Trigger: bad}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("bad prerequisite: %v", err)
	}
	for _, r := range []character.AttunementEndReason{0, 99} {
		if r.String() != "unknown" {
			t.Errorf("reason %d", r)
		}
	}
	for i := character.AttuneRejection(1); i <= 8; i++ {
		if i.String() == "unknown" {
			t.Errorf("attune rejection %d", i)
		}
	}
	for i := character.EndRejection(1); i <= 8; i++ {
		if i.String() == "unknown" {
			t.Errorf("end rejection %d", i)
		}
	}
	if character.AttuneRejection(0).String() != "unknown" || character.EndRejection(0).String() != "unknown" || character.AttunementEndReason(1).String() != "prerequisite_no_longer_met" {
		t.Error("String")
	}
}
