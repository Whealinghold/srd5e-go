package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// ScribingDayReason is why a day of scribing did not count. The zero value is invalid.
type ScribingDayReason int

// The reasons, in the order a rejection lists them.
const (
	ScribingReasonSpellNotPrepared ScribingDayReason = iota + 1
	ScribingReasonMaterialsNotAtHand
	ScribingReasonAlreadyCompleted
)

// String returns "spellNotPrepared", "materialsNotAtHand", "alreadyCompleted" or "unknown".
func (r ScribingDayReason) String() string {
	switch r {
	case ScribingReasonSpellNotPrepared:
		return "spellNotPrepared"
	case ScribingReasonMaterialsNotAtHand:
		return "materialsNotAtHand"
	case ScribingReasonAlreadyCompleted:
		return "alreadyCompleted"
	}
	return "unknown"
}

// ScribingRejection is why [BeginScribing] gave no job. The zero value is invalid.
type ScribingRejection int

// The rejections.
const (
	// ScribingRejectionNotQualified: the scribe has neither Arcana nor Calligrapher's Supplies.
	ScribingRejectionNotQualified ScribingRejection = iota + 1
	// ScribingRejectionTableMissing: the Spell Scroll Costs table is the zero value.
	ScribingRejectionTableMissing
	// ScribingRejectionInvalidStart: the spell has no valid id or the scribe level is zero.
	ScribingRejectionInvalidStart
)

// String returns "notQualified", "tableMissing", "invalidStart" or "unknown".
func (r ScribingRejection) String() string {
	switch r {
	case ScribingRejectionNotQualified:
		return "notQualified"
	case ScribingRejectionTableMissing:
		return "tableMissing"
	case ScribingRejectionInvalidStart:
		return "invalidStart"
	}
	return "unknown"
}

// ScribedScroll is a finished Spell Scroll: the spell, its level, the scribe's
// spell save DC and spell attack bonus at the start (the scroll "uses your"
// values; the 13 and +5 of SRD 5.2.1 p. 99 are the default of a found scroll),
// a caster level for a cantrip, and whether the materials were consumed.
// Immutable. Using a scroll is out of scope of this package.
type ScribedScroll struct {
	spell       content.SpellID
	level       content.SpellLevel
	saveDC      SpellSaveDC
	attack      SpellAttackBonus
	casterLevel Level
	consumed    bool
}

// Spell returns the spell id.
func (s ScribedScroll) Spell() content.SpellID { return s.spell }

// SpellLevel returns the level of the spell.
func (s ScribedScroll) SpellLevel() content.SpellLevel { return s.level }

// SaveDC returns the scribe's spell save DC.
func (s ScribedScroll) SaveDC() SpellSaveDC { return s.saveDC }

// AttackBonus returns the scribe's spell attack bonus.
func (s ScribedScroll) AttackBonus() SpellAttackBonus { return s.attack }

// CasterLevel returns the scribe's level for a cantrip, and false for any other spell.
func (s ScribedScroll) CasterLevel() (Level, bool) { return s.casterLevel, !s.casterLevel.IsZero() }

// MaterialsConsumed reports whether the spell's Material component consumes the
// materials, which happens once, when the scroll is completed.
func (s ScribedScroll) MaterialsConsumed() bool { return s.consumed }

// ScrollScribingJob is a Spell Scroll being scribed: days worked of days
// required, with the cost. Immutable; every day returns a new value. The zero
// value is invalid. Comparable and safe for concurrent use.
type ScrollScribingJob struct {
	scroll       ScribedScroll
	needMaterial bool
	cost         content.Cost
	requiredDays int
	daysWorked   int
}

// NewScrollScribingJob returns a job for spell at the scribe's level, with the
// scribe's spell save DC and spell attack bonus and the cost of the scroll. The
// scroll holds the scribe's level only for a cantrip.
//
// It returns an error matching srderr.ErrInvalidInput for requiredDays below 1,
// daysWorked negative or above requiredDays, a spell with an invalid id or a
// zero scribeLevel. It never panics.
func NewScrollScribingJob(requiredDays, daysWorked int, spell content.Spell, scribeLevel Level, saveDC SpellSaveDC, attackBonus SpellAttackBonus, cost content.Cost) (ScrollScribingJob, error) {
	if requiredDays < 1 || daysWorked < 0 || daysWorked > requiredDays || !spell.ID().Valid() || scribeLevel.IsZero() {
		return ScrollScribingJob{}, invalidInput("scroll scribing job: days, spell or scribe level")
	}
	material, needed := spell.Components().Material()
	scroll := ScribedScroll{spell: spell.ID(), level: spell.Level(), saveDC: saveDC, attack: attackBonus, consumed: needed && material.ConsumesMaterial()}
	if spell.Level() == 0 {
		scroll.casterLevel = scribeLevel
	}
	return ScrollScribingJob{scroll: scroll, needMaterial: needed, cost: cost, requiredDays: requiredDays, daysWorked: daysWorked}, nil
}

// MustScrollScribingJob is NewScrollScribingJob for known-valid input. It panics
// with the error NewScrollScribingJob returns, and only then.
func MustScrollScribingJob(requiredDays, daysWorked int, spell content.Spell, scribeLevel Level, saveDC SpellSaveDC, attackBonus SpellAttackBonus, cost content.Cost) ScrollScribingJob {
	j, err := NewScrollScribingJob(requiredDays, daysWorked, spell, scribeLevel, saveDC, attackBonus, cost)
	if err != nil {
		panic(err)
	}
	return j
}

// Spell returns the spell id.
func (j ScrollScribingJob) Spell() content.SpellID { return j.scroll.spell }

// RequiredDays returns the days the scroll takes.
func (j ScrollScribingJob) RequiredDays() int { return j.requiredDays }

// DaysWorked returns the days counted so far.
func (j ScrollScribingJob) DaysWorked() int { return j.daysWorked }

// DaysRemaining returns the days still to work.
func (j ScrollScribingJob) DaysRemaining() int { return j.requiredDays - j.daysWorked }

// Cost returns the cost of the scroll.
func (j ScrollScribingJob) Cost() content.Cost { return j.cost }

// ScribingStart is what [BeginScribing] reads. Construct it with a keyed
// literal. A nil or foreign Qualification counts as not qualified.
type ScribingStart struct {
	Qualification ScribeQualification
	Spell         content.Spell
	Table         ScrollScribing
	ScribeLevel   Level
	SaveDC        SpellSaveDC
	AttackBonus   SpellAttackBonus
}

// ScribingBeginResult is the closed set of outcomes of [BeginScribing]. Variants:
// ScribingBegun, ScribingRejected.
type ScribingBeginResult interface{ isScribingBeginResult() }

// ScribingBegun holds the job that was started, with no day worked.
type ScribingBegun struct{ job ScrollScribingJob }

// Job returns the job.
func (b ScribingBegun) Job() ScrollScribingJob { return b.job }

// ScribingRejected says no job was started.
type ScribingRejected struct{ reason ScribingRejection }

// Reason returns why no job was started.
func (r ScribingRejected) Reason() ScribingRejection { return r.reason }

func (ScribingBegun) isScribingBeginResult()    {}
func (ScribingRejected) isScribingBeginResult() {}

// BeginScribing starts scribing a Spell Scroll: the days and cost come from the
// Spell Scroll Costs row of the spell's level, no day is worked yet (SRD 5.2.1,
// "Scribing Spell Scrolls", p. 103). Scribing is solo: one scribe, no helpers
// (interpretation: the helper rule sits under nonmagical crafting). It returns
// [ScribingRejected] for an unqualified scribe or a zero table, and never an
// error or a panic; a start [NewScrollScribingJob] refuses (a spell without a
// valid id, a zero ScribeLevel) returns [ScribingRejected] with
// [ScribingRejectionInvalidStart].
func BeginScribing(in ScribingStart) ScribingBeginResult {
	if _, ok := in.Qualification.(ScribeQualified); !ok {
		return ScribingRejected{reason: ScribingRejectionNotQualified}
	}
	if in.Table.IsZero() {
		return ScribingRejected{reason: ScribingRejectionTableMissing}
	}
	row := in.Table.ForSpellLevel(in.Spell.Level())
	job, err := NewScrollScribingJob(row.Days(), 0, in.Spell, in.ScribeLevel, in.SaveDC, in.AttackBonus, row.Price().Cost())
	if err != nil {
		return ScribingRejected{reason: ScribingRejectionInvalidStart}
	}
	return ScribingBegun{job: job}
}

// ScribingDayOutcome is the closed set of outcomes of [ScrollScribingJob.WorkDay].
// Variants: ScribingDayWorked, ScribingCompleted, ScribingDayRejected.
type ScribingDayOutcome interface{ isScribingDayOutcome() }

// ScribingDayWorked says the day counted and the scroll is not finished. Nothing is consumed.
type ScribingDayWorked struct{ job ScrollScribingJob }

// Job returns the job with the day counted.
func (w ScribingDayWorked) Job() ScrollScribingJob { return w.job }

// DaysRemaining returns the days still to work.
func (w ScribingDayWorked) DaysRemaining() int { return w.job.DaysRemaining() }

// ScribingCompleted says the last day counted and the scroll is finished.
type ScribingCompleted struct{ scroll ScribedScroll }

// Scroll returns the finished scroll.
func (c ScribingCompleted) Scroll() ScribedScroll { return c.scroll }

// ScribingDayRejected says the day did not count; the job is unchanged and earlier progress is kept.
type ScribingDayRejected struct{ reasons []ScribingDayReason }

// Reasons returns why the day did not count, in order, as a new slice.
func (r ScribingDayRejected) Reasons() []ScribingDayReason { return slices.Clone(r.reasons) }

func (ScribingDayWorked) isScribingDayOutcome()   {}
func (ScribingCompleted) isScribingDayOutcome()   {}
func (ScribingDayRejected) isScribingDayOutcome() {}

// WorkDay counts one day (8 hours of work) only when the spell is in
// preparedToday and, if the spell has a Material component, the materials are at
// hand; days need not be consecutive and earlier progress is kept (SRD 5.2.1,
// "Prerequisites for the Scribe" and "Time and Cost", p. 103). Otherwise it
// returns [ScribingDayRejected] with the reasons in order (spell not prepared,
// materials not at hand). Whether a Component Pouch or Spellcasting Focus
// stands in for a material is the host's decision when it passes materialsAtHand.
// The materials are consumed only on completion. A job already completed returns
// [ScribingDayRejected] with ScribingReasonAlreadyCompleted. The receiver is
// never modified and no error is returned.
func (j ScrollScribingJob) WorkDay(preparedToday PreparedSpellSet, materialsAtHand bool) ScribingDayOutcome {
	if j.daysWorked >= j.requiredDays {
		return ScribingDayRejected{reasons: []ScribingDayReason{ScribingReasonAlreadyCompleted}}
	}
	var reasons []ScribingDayReason
	if !preparedToday.Contains(j.scroll.spell) {
		reasons = append(reasons, ScribingReasonSpellNotPrepared)
	}
	if j.needMaterial && !materialsAtHand {
		reasons = append(reasons, ScribingReasonMaterialsNotAtHand)
	}
	if len(reasons) > 0 {
		return ScribingDayRejected{reasons: reasons}
	}
	j.daysWorked++
	if j.daysWorked == j.requiredDays {
		return ScribingCompleted{scroll: j.scroll}
	}
	return ScribingDayWorked{job: j}
}
