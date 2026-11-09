package character_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// scribeSpell decodes a synthetic spell of the given level; material is "" for
// none, "kept" for a Material component that is not consumed and "consumed" for
// one that is.
func scribeSpell(tb testing.TB, id string, level int, material string) content.Spell {
	tb.Helper()
	comp := `{"verbal":true,"somatic":false}`
	if material != "" {
		comp = fmt.Sprintf(`{"verbal":true,"somatic":false,"material":{"description":"a pinch of dust","costsGp":[],"consumesMaterial":%t}}`, material == "consumed")
	}
	doc := fmt.Sprintf(`{"schemaVersion":1,"spells":[{"id":%q,"name":"Synthetic","source":{"document":"SRD 5.2.1","section":"Spell Descriptions","page":121},"level":%d,"school":"Illusion",
"classes":["wizard"],"castingTimes":[{"kind":"magicAction"}],"ritual":false,"range":{"kind":"self"},"components":%s,
"duration":{"kind":"instantaneous"},"description":[{"kind":"paragraph","text":"Synthetic."}]}]}`, id, level, comp)
	set, err := content.JSONDecoder{}.Decode([]byte(doc))
	if err != nil {
		tb.Fatalf("fixture spell: %v", err)
	}
	return set.Entries()[0].(content.Spell)
}

var (
	qualified   = character.QualifyScribe(skillsOf([]content.Skill{content.SkillArcana}, nil), character.ToolProficiencies{})
	scribeTable = character.MustScrollScribing(scrollRows())
	scribeStats = character.Stats(character.CasterWizard, must(ability.NewScore(16)).Modifier(), character.ProficiencyBonusFor(character.MustLevel(1)))
)

func start(spell content.Spell, q character.ScribeQualification) character.ScribingStart {
	return character.ScribingStart{Qualification: q, Spell: spell, Table: scribeTable, ScribeLevel: character.MustLevel(5), SaveDC: scribeStats.SaveDC(), AttackBonus: scribeStats.AttackBonus()}
}

func begin(tb testing.TB, spell content.Spell) character.ScrollScribingJob {
	tb.Helper()
	b, ok := character.BeginScribing(start(spell, qualified)).(character.ScribingBegun)
	if !ok {
		tb.Fatal("BeginScribing was rejected")
	}
	return b.Job()
}

// Requirement: per case tool-proficiency-and-crafting.md FR-034, FR-035, FR-041
// (SRD 5.2.1, "Scribing Spell Scrolls", Spell Scroll Costs, p. 103) - a qualified
// scribe begins a job with the row's days and cost and no day worked; any other
// start is a rejection, never an error.
// Path: per case (happy; unhappy rule interaction, missing content).
func TestBeginScribing(t *testing.T) {
	t.Parallel()
	level1, level3 := scribeSpell(t, "spell_one", 1, ""), scribeSpell(t, "spell_three", 3, "consumed")
	cases := []struct {
		name         string
		in           character.ScribingStart
		wantDays     int
		wantCost     content.Cost
		wantRejected character.ScribingRejection
	}{
		{"level 1, qualified", start(level1, qualified), 1, gp(25), 0},
		{"level 3, qualified", start(level3, qualified), 5, gp(150), 0},
		{"not qualified", start(level1, character.ScribeNotQualified{}), 0, content.Cost{}, character.ScribingRejectionNotQualified},
		{"a nil qualification is not qualified", start(level1, nil), 0, content.Cost{}, character.ScribingRejectionNotQualified},
		{"the zero table", character.ScribingStart{Qualification: qualified, Spell: level1, ScribeLevel: character.MustLevel(5)}, 0, content.Cost{}, character.ScribingRejectionTableMissing},
		{"a zero scribe level", character.ScribingStart{Qualification: qualified, Spell: level1, Table: scribeTable}, 0, content.Cost{}, character.ScribingRejectionInvalidStart},
		{"a zero spell", character.ScribingStart{Qualification: qualified, Table: scribeTable, ScribeLevel: character.MustLevel(5)}, 0, content.Cost{}, character.ScribingRejectionInvalidStart},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			switch got := character.BeginScribing(c.in).(type) {
			case character.ScribingBegun:
				j := got.Job()
				if c.wantRejected != 0 || j.RequiredDays() != c.wantDays || j.Cost() != c.wantCost || j.DaysWorked() != 0 || j.DaysRemaining() != c.wantDays || j.Spell() != c.in.Spell.ID() {
					t.Errorf("job = %d days, %v, worked %d", j.RequiredDays(), j.Cost(), j.DaysWorked())
				}
			case character.ScribingRejected:
				if got.Reason() != c.wantRejected {
					t.Errorf("reason = %v, want %v", got.Reason(), c.wantRejected)
				}
			}
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-036, FR-038 (SRD 5.2.1,
// "Time and Cost", p. 103) - days 1 to 4 of a five-day scroll are worked with 4,
// 3, 2, 1 days remaining and day 5 completes it; a completed job takes no more.
// Path: unhappy (boundary).
func TestScribingMultiDay(t *testing.T) {
	t.Parallel()
	spell := scribeSpell(t, "spell_three", 3, "")
	job := begin(t, spell)
	prepared := character.NewPreparedSpellSet(spell.ID())
	for _, remaining := range []int{4, 3, 2, 1} {
		w, ok := job.WorkDay(prepared, false).(character.ScribingDayWorked)
		if !ok || w.DaysRemaining() != remaining || w.Job().DaysWorked() != 5-remaining {
			t.Fatalf("day = %#v, want worked with %d remaining", w, remaining)
		}
		job = w.Job()
	}
	done, ok := job.WorkDay(prepared, false).(character.ScribingCompleted)
	if !ok || done.Scroll().Spell() != spell.ID() || done.Scroll().SpellLevel() != 3 {
		t.Fatalf("day 5 = %#v, want ScribingCompleted", job.WorkDay(prepared, false))
	}
	full := character.MustScrollScribingJob(5, 5, spell, character.MustLevel(5), scribeStats.SaveDC(), scribeStats.AttackBonus(), gp(150))
	rej, ok := full.WorkDay(prepared, true).(character.ScribingDayRejected)
	if !ok || !slices.Equal(rej.Reasons(), []character.ScribingDayReason{character.ScribingReasonAlreadyCompleted}) {
		t.Errorf("a completed job = %#v, want alreadyCompleted", full.WorkDay(prepared, true))
	}
	one := begin(t, scribeSpell(t, "spell_one", 1, ""))
	if _, ok := one.WorkDay(character.NewPreparedSpellSet("spell_one"), false).(character.ScribingCompleted); !ok {
		t.Error("a one-day scroll must complete on day 1")
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-036, FR-037, FR-042
// (SRD 5.2.1, "Prerequisites for the Scribe", p. 103) - a day counts only with the
// spell prepared and any Material at hand; the reasons come in order, the job is
// unchanged and a later good day counts as the next day.
// Path: per case (unhappy rule interaction).
func TestScribingWorkDayRejections(t *testing.T) {
	t.Parallel()
	notPrepared, noMaterials := character.ScribingReasonSpellNotPrepared, character.ScribingReasonMaterialsNotAtHand
	cases := []struct {
		name      string
		material  string
		prepared  bool
		atHand    bool
		want      []character.ScribingDayReason
		wantCount bool
	}{
		{"not prepared", "", false, true, []character.ScribingDayReason{notPrepared}, false},
		{"not prepared, no Material needed, materials flag false", "", false, false, []character.ScribingDayReason{notPrepared}, false},
		{"Material not at hand", "kept", true, false, []character.ScribingDayReason{noMaterials}, false},
		{"neither prepared nor at hand, in that order", "kept", false, false, []character.ScribingDayReason{notPrepared, noMaterials}, false},
		{"no Material, flag false is ignored", "", true, false, nil, true},
		{"Material at hand", "kept", true, true, nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			spell := scribeSpell(t, "spell_three", 3, c.material)
			job := begin(t, spell)
			set := character.NewPreparedSpellSet()
			if c.prepared {
				set = character.NewPreparedSpellSet(spell.ID())
			}
			before := job
			switch got := job.WorkDay(set, c.atHand).(type) {
			case character.ScribingDayRejected:
				if c.wantCount || !slices.Equal(got.Reasons(), c.want) {
					t.Errorf("reasons = %v, want %v", got.Reasons(), c.want)
				}
				got.Reasons()[0] = 0
				if again := job.WorkDay(set, c.atHand).(character.ScribingDayRejected).Reasons(); again[0] == 0 {
					t.Error("Reasons aliases the rejection's own slice")
				}
			case character.ScribingDayWorked:
				if !c.wantCount || got.Job().DaysWorked() != 1 {
					t.Errorf("day counted = %v", got.Job().DaysWorked())
				}
			}
			if job != before {
				t.Error("WorkDay changed its receiver")
			}
		})
	}
	t.Run("a rejected day loses no progress", func(t *testing.T) {
		t.Parallel()
		spell := scribeSpell(t, "spell_three", 3, "")
		job := character.MustScrollScribingJob(5, 2, spell, character.MustLevel(5), scribeStats.SaveDC(), scribeStats.AttackBonus(), gp(150))
		if _, ok := job.WorkDay(character.NewPreparedSpellSet(), true).(character.ScribingDayRejected); !ok {
			t.Fatal("want a rejection")
		}
		w, ok := job.WorkDay(character.NewPreparedSpellSet(spell.ID()), true).(character.ScribingDayWorked)
		if !ok || w.Job().DaysWorked() != 3 {
			t.Errorf("the next good day = %#v, want day 3", w)
		}
	})
	t.Run("a non-caster with an empty prepared set never advances", func(t *testing.T) {
		t.Parallel()
		job := begin(t, scribeSpell(t, "spell_one", 1, ""))
		rej, ok := job.WorkDay(character.PreparedSpellSetFromSpellcasting(nil), true).(character.ScribingDayRejected)
		if !ok || !slices.Equal(rej.Reasons(), []character.ScribingDayReason{notPrepared}) {
			t.Errorf("got %#v", rej)
		}
	})
}

// Requirement: per case tool-proficiency-and-crafting.md FR-039, FR-040 (SRD
// 5.2.1, "Prerequisites for the Scribe", "Cantrips", p. 103; "Spell Scroll",
// p. 99) - Materials are consumed only on completion and only if the spell
// consumes them; the scroll snapshots the scribe's DC 13 and +5 and carries the
// scribe's level for a cantrip only.
// Path: per case (unhappy rule interaction; happy for the snapshot).
func TestScribedScroll(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		spell        content.Spell
		days         int
		wantConsumed bool
		wantCaster   bool
	}{
		{"Revivify-like: consumed, level 3", scribeSpell(t, "spell_three", 3, "consumed"), 5, true, false},
		{"Material not consumed", scribeSpell(t, "spell_kept", 3, "kept"), 5, false, false},
		{"cantrip with no Material", scribeSpell(t, "fire_bolt_x", 0, ""), 1, false, true},
		{"level 1 spell has no caster level", scribeSpell(t, "spell_one", 1, ""), 1, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			job := begin(t, c.spell)
			set := character.NewPreparedSpellSet(c.spell.ID())
			for day := 1; day < c.days; day++ {
				w, ok := job.WorkDay(set, true).(character.ScribingDayWorked)
				if !ok {
					t.Fatalf("day %d was not a worked day", day)
				}
				job = w.Job()
			}
			done, ok := job.WorkDay(set, true).(character.ScribingCompleted)
			if !ok {
				t.Fatal("the last day did not complete the scroll")
			}
			s := done.Scroll()
			level, has := s.CasterLevel()
			if s.MaterialsConsumed() != c.wantConsumed || has != c.wantCaster || (has && level.Value() != 5) {
				t.Errorf("consumed %v, caster level %v (%v)", s.MaterialsConsumed(), level.Value(), has)
			}
			if s.SaveDC().Value() != 13 || s.AttackBonus().Value() != 5 {
				t.Errorf("snapshot = DC %d, +%d, want 13 and +5", s.SaveDC().Value(), s.AttackBonus().Value())
			}
		})
	}
}

// Requirement: per case tool-proficiency-and-crafting.md FR-043 (architecture.md
// section 6) - a malformed job is an invalid-input error and MustScrollScribingJob
// panics with it.
// Path: per case (unhappy invalid input).
func TestNewScrollScribingJobRejectsBadInput(t *testing.T) {
	t.Parallel()
	spell := scribeSpell(t, "spell_one", 1, "")
	level := character.MustLevel(5)
	cases := []struct {
		name             string
		required, worked int
		spell            content.Spell
		scribeLevel      character.Level
		wantOK           bool
	}{
		{"requiredDays 0", 0, 0, spell, level, false},
		{"daysWorked -1", 5, -1, spell, level, false},
		{"daysWorked above required", 5, 6, spell, level, false},
		{"a zero spell", 5, 0, content.Spell{}, level, false},
		{"a zero scribe level", 5, 0, spell, character.Level{}, false},
		{"daysWorked equal to required (completed)", 5, 5, spell, level, true},
		{"nothing worked", 1, 0, spell, level, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			j, err := character.NewScrollScribingJob(c.required, c.worked, c.spell, c.scribeLevel, scribeStats.SaveDC(), scribeStats.AttackBonus(), gp(25))
			if c.wantOK {
				if err != nil || j.DaysWorked() != c.worked || j.RequiredDays() != c.required {
					t.Errorf("job = %v, %v", j, err)
				}
				return
			}
			wantInvalid(t, err, c.name)
			defer func() {
				e, ok := recover().(error)
				if !ok || !errors.Is(e, srderr.ErrInvalidInput) {
					t.Errorf("panic = %v, want an error matching ErrInvalidInput", e)
				}
			}()
			character.MustScrollScribingJob(c.required, c.worked, c.spell, c.scribeLevel, scribeStats.SaveDC(), scribeStats.AttackBonus(), gp(25))
		})
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-043 - the String of the
// scribing enums, including the unknown value.
// Path: unhappy (invalid input).
func TestScribingEnumStrings(t *testing.T) {
	t.Parallel()
	for r, want := range map[character.ScribingDayReason]string{character.ScribingReasonSpellNotPrepared: "spellNotPrepared", character.ScribingReasonMaterialsNotAtHand: "materialsNotAtHand", character.ScribingReasonAlreadyCompleted: "alreadyCompleted", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("day reason %d = %q, want %q", int(r), r.String(), want)
		}
	}
	for r, want := range map[character.ScribingRejection]string{character.ScribingRejectionNotQualified: "notQualified", character.ScribingRejectionTableMissing: "tableMissing", character.ScribingRejectionInvalidStart: "invalidStart", 0: "unknown"} {
		if r.String() != want {
			t.Errorf("rejection %d = %q, want %q", int(r), r.String(), want)
		}
	}
}

// Requirement: tool-proficiency-and-crafting.md FR-036, FR-038, FR-043 - for any
// sequence of days the days worked never pass the days required and never fall.
// Path: unhappy (boundary).
func FuzzScribingDays(f *testing.F) {
	f.Add([]byte{1, 1, 1, 1, 1, 1})
	f.Add([]byte{0, 3, 2, 1, 0})
	f.Add([]byte{})
	spell := scribeSpell(f, "spell_three", 3, "kept")
	f.Fuzz(func(t *testing.T, days []byte) {
		job := begin(t, spell)
		for _, d := range days {
			prepared := character.NewPreparedSpellSet()
			if d&1 == 1 {
				prepared = character.NewPreparedSpellSet(spell.ID())
			}
			before := job.DaysWorked()
			switch out := job.WorkDay(prepared, d&2 == 2).(type) {
			case character.ScribingDayWorked:
				job = out.Job()
			case character.ScribingCompleted:
				job = character.MustScrollScribingJob(job.RequiredDays(), job.RequiredDays(), spell, character.MustLevel(5), scribeStats.SaveDC(), scribeStats.AttackBonus(), job.Cost())
			case character.ScribingDayRejected:
			}
			if job.DaysWorked() < before || job.DaysWorked() > job.RequiredDays() {
				t.Fatalf("days worked %d (before %d) outside 0..%d", job.DaysWorked(), before, job.RequiredDays())
			}
		}
	})
}
