package character_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/class-warlock.md FR-156, FR-157, FR-047, FR-055, FR-057, FR-058, table 4.6.8 (SRD 5.2.1, spell entries pp. 115, 127, 145, 149, 153, 154, 171) - for each cantrip fact row the kind (deals damage; deals damage via an attack roll) and the 10 foot range test give the printed eligibility; a range of miles qualifies and Self, Touch, Sight, Unlimited and Special do not.
// Path: per case (CC-01 to CC-07 and the range boundaries).
func TestCantripFactsKindsAndRange(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id                            string
		damage, attack, spearEligible bool
	}{
		{"chill_touch", true, true, false}, {"eldritch_blast", true, true, true}, {"mage_hand", false, false, false}, {"minor_illusion", false, false, false},
		{"poison_spray", true, true, true}, {"prestidigitation", false, false, false}, {"true_strike", true, true, false},
	}
	facts := warCantrips()
	for _, c := range cases {
		f := facts[c.id]
		spear := f.IsKind(character.CantripDealsDamage) && f.HasRangeOfTenFeet()
		if f.IsKind(character.CantripDealsDamage) != c.damage || f.IsKind(character.CantripDealsDamageViaAttackRoll) != c.attack || spear != c.spearEligible || f.Spell() != content.SpellID(c.id) {
			t.Errorf("%s: damage %v attack %v spear %v", c.id, f.IsKind(character.CantripDealsDamage), f.IsKind(character.CantripDealsDamageViaAttackRoll), spear)
		}
	}
	// A cantrip that deals damage without an attack roll meets the first kind only.
	d := warFacts("acid_splash", character.CantripDamage, warFeet(60))
	if !d.IsKind(character.CantripDealsDamage) || d.IsKind(character.CantripDealsDamageViaAttackRoll) || d.IsKind(0) || d.Effect() != character.CantripDamage {
		t.Error("CantripDamage must be a damage cantrip without an attack roll")
	}
	for _, c := range []struct {
		name string
		r    content.SpellRange
		want bool
	}{
		{"9 feet", warFeet(9), false}, {"10 feet", warFeet(10), true}, {"11 feet", warFeet(11), true}, {"1 mile", must(content.NewSpellRangeDistance(1, content.DistanceMiles)), true},
		{"Self", content.NewSpellRangeSelf(), false}, {"Touch", content.NewSpellRangeTouch(), false}, {"Sight", content.NewSpellRangeSight(), false},
		{"Unlimited", content.NewSpellRangeUnlimited(), false}, {"Special", content.NewSpellRangeSpecial(), false},
	} {
		if got := warFacts("x", character.CantripDamage, c.r).HasRangeOfTenFeet(); got != c.want {
			t.Errorf("%s: HasRangeOfTenFeet = %v, want %v", c.name, got, c.want)
		}
	}
	if r := facts["eldritch_blast"].Range(); r != warFeet(120) {
		t.Errorf("Range = %v", r)
	}
}

// Requirement: doc/specs/class-warlock.md FR-156 - NewCantripFacts refuses an invalid spell id, an invalid effect and a nil range with ErrInvalidInput; CantripKind has its own String.
// Path: unhappy (invalid input).
func TestNewCantripFactsRefusals(t *testing.T) {
	t.Parallel()
	for name, args := range map[string]struct {
		id     content.SpellID
		effect character.CantripEffect
		r      content.SpellRange
	}{"id": {"", character.CantripDamage, warFeet(5)}, "effect": {"x", 0, warFeet(5)}, "range": {"x", character.CantripDamage, nil}} {
		if _, err := character.NewCantripFacts(args.id, args.effect, args.r); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if character.CantripDealsDamage.String() != "deals_damage" || character.CantripDealsDamageViaAttackRoll.String() != "deals_damage_via_attack_roll" || character.CantripKind(0).String() != "unknown" {
		t.Error("CantripKind strings")
	}
	if !character.CantripNoDamage.Valid() || !character.CantripDamageViaAttackRoll.Valid() || character.CantripEffect(0).Valid() || character.CantripEffect(4).Valid() {
		t.Error("CantripEffect.Valid")
	}
}

// Requirement: doc/specs/class-warlock.md FR-045 to FR-048, section 4.6.1 (SRD 5.2.1, "Eldritch Invocation Options", pp. 72-74) - over every option of the bundled Warlock: a minimum level N is met at N and not at N minus 1 (UnmetLevel with the level), a required invocation is met only when chosen (UnmetOption naming the first missing one), the cantrip kinds of Agonizing Blast, Eldritch Spear and Repelling Blast are decided by option id from the known CantripFacts (UnmetCantrip), and at level 1 exactly the five options with no prerequisite qualify.
// Path: per case (every option; boundary N-1 and N; rule interaction).
func TestInvocationEligibility(t *testing.T) {
	t.Parallel()
	known := []character.CantripFacts{warCantrips()["eldritch_blast"]}
	var atLevelOne []string
	for _, o := range warOptions(t) {
		pre, has := o.Prerequisite()
		var chosen []content.FeatureOptionID
		chosen = append(chosen, pre.RequiresOptionIDs()...)
		if has && pre.MinClassLevel() > 1 {
			unmet, ok := character.InvocationEligibility(o, character.MustWarlockLevel(pre.MinClassLevel()-1), chosen, known)
			if ok || unmet != (character.UnmetPrerequisite{Kind: character.UnmetLevel, Level: pre.MinClassLevel()}) {
				t.Errorf("%s at level %d = %+v, %v; want UnmetLevel %d", o.ID(), pre.MinClassLevel()-1, unmet, ok, pre.MinClassLevel())
			}
		}
		level := 1
		if has && pre.MinClassLevel() > 0 {
			level = pre.MinClassLevel()
		}
		if unmet, ok := character.InvocationEligibility(o, character.MustWarlockLevel(level), chosen, known); !ok {
			t.Errorf("%s at level %d with its requirements = %+v, want eligible", o.ID(), level, unmet)
		}
		if len(pre.RequiresOptionIDs()) > 0 {
			unmet, ok := character.InvocationEligibility(o, character.MustWarlockLevel(20), nil, known)
			if ok || unmet != (character.UnmetPrerequisite{Kind: character.UnmetOption, Option: pre.RequiresOptionIDs()[0]}) {
				t.Errorf("%s without %v = %+v, %v", o.ID(), pre.RequiresOptionIDs(), unmet, ok)
			}
		}
		if _, ok := character.InvocationEligibility(o, character.MustWarlockLevel(1), chosen, known); ok && level == 1 && !has {
			atLevelOne = append(atLevelOne, string(o.ID()))
		}
	}
	if want := []string{"armor_of_shadows", "eldritch_mind", "pact_of_the_blade", "pact_of_the_chain", "pact_of_the_tome"}; !slices.Equal(atLevelOne, want) {
		t.Errorf("options with no prerequisite = %v, want %v", atLevelOne, want)
	}
	need := map[string]character.CantripKind{"agonizing_blast": character.CantripDealsDamage, "eldritch_spear": character.CantripDealsDamage, "repelling_blast": character.CantripDealsDamageViaAttackRoll}
	facts := warCantrips()
	for id, kind := range need {
		o := warOption(t, id)
		unmet, ok := character.InvocationEligibility(o, character.MustWarlockLevel(5), nil, []character.CantripFacts{facts["mage_hand"]})
		if ok || unmet != (character.UnmetPrerequisite{Kind: character.UnmetCantrip, Cantrip: kind}) {
			t.Errorf("%s with only Mage Hand = %+v, %v; want UnmetCantrip %v", id, unmet, ok, kind)
		}
		if _, ok := character.InvocationEligibility(o, character.MustWarlockLevel(5), nil, nil); ok {
			t.Errorf("%s with no known cantrip must be refused", id)
		}
		if _, ok := character.InvocationEligibility(o, character.MustWarlockLevel(5), nil, []character.CantripFacts{facts["chill_touch"]}); !ok {
			t.Errorf("%s with Chill Touch must be eligible (Eldritch Spear's range is tested on the pick)", id)
		}
	}
	// No other option names a cantrip kind, whatever its printed note.
	for _, o := range warOptions(t) {
		pre, _ := o.Prerequisite()
		if _, listed := need[string(o.ID())]; !listed && pre.Note() != "" {
			t.Errorf("%s has the note %q, which no option id decides", o.ID(), pre.Note())
		}
	}
	// An option with no prerequisite at all is always eligible.
	if _, ok := character.InvocationEligibility(warOption(t, "eldritch_mind"), character.MustWarlockLevel(1), nil, nil); !ok {
		t.Error("Eldritch Mind has no prerequisite")
	}
}

func warPick(option string, spell, feat string) character.InvocationPick {
	return character.InvocationPick{Option: content.FeatureOptionID(option), Cantrip: content.SpellID(spell), Feat: content.FeatID(feat)}
}

func warProblemKinds(ps []character.InvocationProblem) []character.InvocationProblemKind {
	var out []character.InvocationProblemKind
	for _, p := range ps {
		out = append(out, p.Kind)
	}
	return out
}

// Requirement: doc/specs/class-warlock.md FR-043 to FR-053, FR-104 (SRD 5.2.1, "Level 1: Eldritch Invocations", pp. 70-71) - SelectInvocations accepts a valid full selection and lists every problem of a bad one: the wrong count with the expected and given numbers, an unknown option, an unmet prerequisite, a duplicate of a non-Repeatable option, a repeat of the same cantrip or feat, a cantrip not known or of the wrong kind (and Chill Touch for Eldritch Spear), a missing feat, more than one replacement and the replacement of a prerequisite.
// Path: per case (happy, and each problem kind).
func TestSelectInvocations(t *testing.T) {
	t.Parallel()
	listed := warOptions(t)
	facts := warCantrips()
	known := []character.CantripFacts{facts["eldritch_blast"], facts["chill_touch"], facts["mage_hand"]}
	ids := func(opts ...string) []content.FeatureOptionID {
		var out []content.FeatureOptionID
		for _, o := range opts {
			out = append(out, content.FeatureOptionID(o))
		}
		return out
	}
	cases := []struct {
		name, requirement, path string
		level, entitled         int
		picks                   []character.InvocationPick
		current                 []content.FeatureOptionID
		want                    []character.InvocationProblemKind
		detail                  func(ps []character.InvocationProblem) bool
	}{
		{"level 1 with one of the five", "FR-048", "happy", 1, 1, []character.InvocationPick{warPick("pact_of_the_blade", "", "")}, nil, nil, nil},
		{"level 2 with three", "FR-043", "happy", 2, 3, []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("agonizing_blast", "eldritch_blast", ""), warPick("devils_sight", "", "")}, nil, nil, nil},
		{"repeatable with different cantrips", "FR-050", "happy", 5, 5, []character.InvocationPick{warPick("agonizing_blast", "eldritch_blast", ""), warPick("agonizing_blast", "chill_touch", ""),
			warPick("repelling_blast", "eldritch_blast", ""), warPick("eldritch_spear", "eldritch_blast", ""), warPick("lessons_of_the_first_ones", "", "alert")}, nil, nil, nil},
		{"too few", "FR-043", "unhappy (boundary)", 1, 1, nil, nil, []character.InvocationProblemKind{character.InvocationWrongCount},
			func(ps []character.InvocationProblem) bool { return ps[0].Expected == 1 && ps[0].Given == 0 }},
		{"too many", "FR-043", "unhappy (boundary)", 1, 1, []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("pact_of_the_tome", "", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationWrongCount}, func(ps []character.InvocationProblem) bool { return ps[0].Expected == 1 && ps[0].Given == 2 }},
		{"unknown option", "FR-044", "unhappy (invalid input)", 1, 1, []character.InvocationPick{warPick("fireball", "", "")}, nil, []character.InvocationProblemKind{character.InvocationUnknown},
			func(ps []character.InvocationProblem) bool { return ps[0].Option == "fireball" }},
		{"level prerequisite at N-1", "FR-045", "unhappy (boundary)", 4, 1, []character.InvocationPick{warPick("ascendant_step", "", "")}, nil, []character.InvocationProblemKind{character.InvocationPrerequisiteUnmet},
			func(ps []character.InvocationProblem) bool {
				return ps[0].Unmet == character.UnmetPrerequisite{Kind: character.UnmetLevel, Level: 5}
			}},
		{"required invocation missing", "FR-046", "unhappy (rule interaction)", 12, 1, []character.InvocationPick{warPick("devouring_blade", "", "")}, nil, []character.InvocationProblemKind{character.InvocationPrerequisiteUnmet},
			func(ps []character.InvocationProblem) bool {
				return ps[0].Unmet == character.UnmetPrerequisite{Kind: character.UnmetOption, Option: "thirsting_blade"}
			}},
		{"no cantrip known for the kind", "FR-047", "unhappy (rule interaction)", 2, 1, []character.InvocationPick{warPick("repelling_blast", "mage_hand", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationCantripNotEligible}, func(ps []character.InvocationProblem) bool { return ps[0].Cantrip == "mage_hand" }},
		{"duplicate of a non-repeatable", "FR-049", "unhappy (invalid input)", 2, 2, []character.InvocationPick{warPick("devils_sight", "", ""), warPick("devils_sight", "", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationDuplicateNotRepeatable}, func(ps []character.InvocationProblem) bool { return ps[0].Option == "devils_sight" }},
		{"repeat of the same cantrip", "FR-050", "unhappy (rule interaction)", 2, 2, []character.InvocationPick{warPick("agonizing_blast", "eldritch_blast", ""), warPick("agonizing_blast", "eldritch_blast", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationRepeatChoiceDuplicated}, func(ps []character.InvocationProblem) bool { return ps[0].Cantrip == "eldritch_blast" }},
		{"repeat of the same feat", "FR-050, FR-104", "unhappy (rule interaction)", 2, 2, []character.InvocationPick{warPick("lessons_of_the_first_ones", "", "alert"), warPick("lessons_of_the_first_ones", "", "alert")}, nil,
			[]character.InvocationProblemKind{character.InvocationRepeatChoiceDuplicated}, func(ps []character.InvocationProblem) bool { return ps[0].Feat == "alert" }},
		{"lessons without a feat", "FR-104", "unhappy (invalid input)", 2, 1, []character.InvocationPick{warPick("lessons_of_the_first_ones", "", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationFeatMissing}, nil},
		{"Eldritch Spear on Chill Touch", "FR-057", "unhappy (boundary)", 2, 1, []character.InvocationPick{warPick("eldritch_spear", "chill_touch", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationCantripNotEligible}, func(ps []character.InvocationProblem) bool { return ps[0].Cantrip == "chill_touch" }},
		{"cantrip not named", "FR-055", "unhappy (invalid input)", 2, 1, []character.InvocationPick{warPick("agonizing_blast", "", "")}, nil, []character.InvocationProblemKind{character.InvocationCantripNotEligible}, nil},
		{"cantrip not known", "FR-055", "unhappy (invalid input)", 2, 1, []character.InvocationPick{warPick("agonizing_blast", "poison_spray", "")}, nil, []character.InvocationProblemKind{character.InvocationCantripNotEligible}, nil},
		{"replace one", "FR-051", "happy", 3, 3, []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("eldritch_mind", "", ""), warPick("devils_sight", "", "")},
			ids("pact_of_the_blade", "eldritch_mind", "armor_of_shadows"), nil, nil},
		{"replace two", "FR-051", "unhappy (boundary)", 3, 3, []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("devils_sight", "", ""), warPick("misty_visions", "", "")},
			ids("pact_of_the_blade", "eldritch_mind", "armor_of_shadows"), []character.InvocationProblemKind{character.InvocationTooManyReplacements}, func(ps []character.InvocationProblem) bool { return ps[0].Given == 2 }},
		{"replace a prerequisite", "FR-052", "unhappy (rule interaction)", 5, 3, []character.InvocationPick{warPick("eldritch_mind", "", ""), warPick("pact_of_the_tome", "", ""), warPick("eldritch_smite", "", "")},
			ids("pact_of_the_blade", "eldritch_smite", "pact_of_the_tome"), []character.InvocationProblemKind{character.InvocationPrerequisiteUnmet, character.InvocationIsPrerequisite},
			func(ps []character.InvocationProblem) bool {
				return ps[1].Option == "pact_of_the_blade" && ps[1].Dependent == "eldritch_smite"
			}},
		{"gaining at a level keeps the old ones", "FR-053", "happy", 5, 5, []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("eldritch_mind", "", ""), warPick("devils_sight", "", ""),
			warPick("thirsting_blade", "", ""), warPick("armor_of_shadows", "", "")}, ids("pact_of_the_blade", "eldritch_mind", "devils_sight"), nil, nil},
		{"short of the count after a level", "FR-053", "unhappy (boundary)", 5, 5, []character.InvocationPick{warPick("pact_of_the_blade", "", ""), warPick("eldritch_mind", "", ""), warPick("devils_sight", "", "")},
			ids("pact_of_the_blade", "eldritch_mind", "devils_sight"), []character.InvocationProblemKind{character.InvocationWrongCount}, nil},
		{"every problem is listed together", "FR-043 to FR-047", "unhappy (invalid input)", 2, 2, []character.InvocationPick{warPick("fireball", "", ""), warPick("eldritch_smite", "", ""), warPick("pact_of_the_chain", "", ""), warPick("armor_of_shadows", "", "")}, nil,
			[]character.InvocationProblemKind{character.InvocationWrongCount, character.InvocationUnknown, character.InvocationPrerequisiteUnmet}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			picksBefore := slices.Clone(c.picks)
			sel, problems := character.SelectInvocations(character.MustWarlockLevel(c.level), c.entitled, c.picks, c.current, listed, known)
			if !slices.Equal(c.picks, picksBefore) {
				t.Error("SelectInvocations altered the picks")
			}
			if c.want == nil {
				if len(problems) != 0 || !slices.Equal(sel.Picks(), c.picks) {
					t.Errorf("[%s] %s: problems = %+v, picks %v", c.requirement, c.path, problems, sel.Picks())
				}
				return
			}
			if got := warProblemKinds(problems); !slices.Equal(got, c.want) || len(sel.Picks()) != 0 {
				t.Fatalf("[%s] %s: problems = %v (%+v), want %v with an empty selection", c.requirement, c.path, got, problems, c.want)
			}
			if c.detail != nil && !c.detail(problems) {
				t.Errorf("[%s] detail of %+v", c.requirement, problems)
			}
		})
	}
}

// Requirement: doc/specs/class-warlock.md FR-144 - the picks a selection returns are a copy: changing the slice read from Picks does not change the selection.
// Path: unhappy (immutability).
func TestInvocationSelectionPicksAreACopy(t *testing.T) {
	t.Parallel()
	sel, problems := character.SelectInvocations(character.MustWarlockLevel(1), 1, []character.InvocationPick{warPick("pact_of_the_blade", "", "")}, nil, warOptions(t), nil)
	if len(problems) != 0 {
		t.Fatal(problems)
	}
	sel.Picks()[0].Option = "changed"
	if sel.Picks()[0].Option != "pact_of_the_blade" {
		t.Error("Picks leaked the selection's slice")
	}
	if character.InvocationProblemKind(0).String() != "unknown" || character.InvocationWrongCount.String() != "wrong_count" || character.InvocationIsPrerequisite.String() != "invocation_is_prerequisite" || character.InvocationUnknown.String() != "unknown_invocation" {
		t.Error("InvocationProblemKind strings")
	}
}

// Requirement: doc/specs/class-warlock.md FR-059 to FR-063, FR-065, FR-089, FR-165, table 4.6.3 (SRD 5.2.1, pp. 72-74) - each spell-granting invocation returns its spell with CastModeFeatureGranted; Fiendish Vigor takes the maximum of the die and the others roll; One with Shadows needs Dim Light or Darkness (Bright Light is refused with the light level); Gift of the Depths is cast once until a Long Rest; an invocation not known or granting no spell is refused; the state changes for Gift of the Depths only.
// Path: per case (SG-01 to SG-12; state errors).
func TestGrantInvocationSpell(t *testing.T) {
	t.Parallel()
	id := character.MustIdentifier("pc-1")
	grants := map[string]string{"armor_of_shadows": "mage_armor", "ascendant_step": "levitate", "fiendish_vigor": "false_life", "mask_of_many_faces": "disguise_self",
		"master_of_myriad_forms": "alter_self", "misty_visions": "silent_image", "one_with_shadows": "invisibility", "otherworldly_leap": "jump",
		"visions_of_distant_realms": "arcane_eye", "whispers_of_the_grave": "speak_with_dead", "gift_of_the_depths": "water_breathing", "pact_of_the_chain": "find_familiar"}
	for option, spell := range grants {
		s := warKnows(content.FeatureOptionID(option))
		got, ok := character.GrantInvocationSpell(id, s, content.FeatureOptionID(option), content.LightDim).(character.WarlockChanged)
		if !ok {
			t.Errorf("%s: refused", option)
			continue
		}
		g, has := got.SpellGrant()
		wantDie := character.SpellDieRoll
		if option == "fiendish_vigor" {
			wantDie = character.SpellDieMaximum
		}
		if !has || g.Spell != content.SpellID(spell) || g.Mode != character.CastModeFeatureGranted || g.DieRule != wantDie {
			t.Errorf("%s: grant = %+v, %v; want %s feature-granted %v", option, g, has, spell, wantDie)
		}
		events := got.Events()
		if len(events) != 1 || events[0].(character.InvocationSpellCastWithoutSlot).Spell() != content.SpellID(spell) || events[0].(character.InvocationSpellCastWithoutSlot).Option() != content.FeatureOptionID(option) ||
			events[0].CharacterID() != id {
			t.Errorf("%s: events = %+v", option, events)
		}
		if _, slot := got.SlotChange(); slot {
			t.Errorf("%s: a slot-free cast must change no slot", option)
		}
		if (got.State() != s) != (option == "gift_of_the_depths") {
			t.Errorf("%s: state change = %v", option, got.State() != s)
		}
	}
	rejection := func(s character.WarlockState, option string, light content.LightLevel) character.WarlockRejection {
		r, ok := character.GrantInvocationSpell(id, s, content.FeatureOptionID(option), light).(character.WarlockRefused)
		if !ok {
			t.Fatalf("%s: not refused", option)
		}
		return r.Rejection()
	}
	if r := rejection(warKnows(), "armor_of_shadows", content.LightDim); r.Kind != character.WarlockInvocationNotKnown || r.Option != "armor_of_shadows" {
		t.Errorf("not known = %+v", r)
	}
	if r := rejection(warKnows("devils_sight"), "devils_sight", content.LightDim); r.Kind != character.WarlockNotASpellGrant || r.Option != "devils_sight" {
		t.Errorf("not a grant = %+v", r)
	}
	one := warKnows("one_with_shadows")
	for _, light := range []content.LightLevel{content.LightDim, content.LightDarkness} {
		if _, ok := character.GrantInvocationSpell(id, one, "one_with_shadows", light).(character.WarlockChanged); !ok {
			t.Errorf("One with Shadows in %v must be accepted", light)
		}
	}
	if r := rejection(one, "one_with_shadows", content.LightBright); r.Kind != character.WarlockLightLevelWrong || r.LightLevel != content.LightBright {
		t.Errorf("bright light = %+v", r)
	}
	if r := rejection(one, "one_with_shadows", 0); r.Kind != character.WarlockLightLevelWrong {
		t.Errorf("no light level = %+v", r)
	}
	first := character.GrantInvocationSpell(id, warKnows("gift_of_the_depths"), "gift_of_the_depths", content.LightBright).(character.WarlockChanged)
	if !first.State().GiftOfTheDepthsCast() {
		t.Error("the first cast must spend Gift of the Depths")
	}
	if r := rejection(first.State(), "gift_of_the_depths", content.LightBright); r.Kind != character.WarlockAlreadyCast || r.Option != "gift_of_the_depths" {
		t.Errorf("second cast = %+v", r)
	}
	rest := must(character.ApplyWarlockRest(id, first.State(), character.LongRest)).(character.WarlockChanged)
	if _, ok := character.GrantInvocationSpell(id, rest.State(), "gift_of_the_depths", content.LightBright).(character.WarlockChanged); !ok {
		t.Error("a Long Rest must restore the Gift of the Depths cast")
	}
}

// Requirement: doc/specs/class-warlock.md FR-064, FR-066, FR-067, FR-068 (SRD 5.2.1, pp. 72-74) - the standing effects appear with their invocation and not without, in the order Gift of the Depths, Devil's Sight (120 feet), Eldritch Mind (Advantage on Constitution saves to maintain Concentration only), Witch Sight (30 feet).
// Path: per case (present and absent).
func TestDeriveInvocationEffects(t *testing.T) {
	t.Parallel()
	if got := character.DeriveInvocationEffects(warKnows("pact_of_the_blade", "armor_of_shadows")); len(got) != 0 || got == nil {
		t.Errorf("no effect invocations = %v, want an empty non-nil slice", got)
	}
	all := character.DeriveInvocationEffects(warKnows("witch_sight", "eldritch_mind", "devils_sight", "gift_of_the_depths"))
	want := []character.InvocationEffect{character.InvocationEffectUnderwater{}, character.InvocationEffectDevilsSight{RangeFeet: 120},
		character.InvocationEffectConcentrationAdvantage{Circumstance: modifier.Advantage{}}, character.InvocationEffectWitchSight{RangeFeet: 30}}
	if len(all) != 4 {
		t.Fatalf("effects = %v", all)
	}
	for i := range want {
		if all[i] != want[i] {
			t.Errorf("effect %d = %#v, want %#v", i, all[i], want[i])
		}
	}
	for _, one := range []struct {
		id   string
		want character.InvocationEffect
	}{{"gift_of_the_depths", want[0]}, {"devils_sight", want[1]}, {"eldritch_mind", want[2]}, {"witch_sight", want[3]}} {
		if got := character.DeriveInvocationEffects(warKnows(content.FeatureOptionID(one.id))); len(got) != 1 || got[0] != one.want {
			t.Errorf("%s alone = %v", one.id, got)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-055 (SRD 5.2.1, "Agonizing Blast", p. 72) - the Charisma modifier is added as written, negative included, and only for the chosen cantrip.
// Path: per case (boundary: negative, zero, other cantrip).
func TestAgonizingBlastBonus(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		cantrip string
		cha     int
		want    int
		ok      bool
	}{{"eldritch_blast", 4, 4, true}, {"eldritch_blast", 0, 0, true}, {"eldritch_blast", -1, -1, true}, {"poison_spray", 4, 0, false}} {
		got, ok := character.AgonizingBlastBonus("eldritch_blast", content.SpellID(c.cantrip), c.cha)
		if got != c.want || ok != c.ok {
			t.Errorf("%s cha %d: got %d, %v", c.cantrip, c.cha, got, ok)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-058 (SRD 5.2.1, "Repelling Blast", p. 74) - Large or smaller is pushed 10 feet, Huge or larger and an invalid size never, and only for the chosen cantrip.
// Path: per case (boundary Large/Huge).
func TestRepellingBlastPush(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		cantrip string
		size    content.CreatureSize
		want    int
		ok      bool
	}{
		{"eldritch_blast", content.SizeTiny, 10, true}, {"eldritch_blast", content.SizeLarge, 10, true}, {"eldritch_blast", content.SizeHuge, 0, false},
		{"eldritch_blast", content.CreatureSize(0), 0, false}, {"poison_spray", content.SizeLarge, 0, false},
	} {
		got, ok := character.RepellingBlastPush("eldritch_blast", content.SpellID(c.cantrip), c.size)
		if got != c.want || ok != c.ok {
			t.Errorf("%s %v: got %d, %v", c.cantrip, c.size, got, ok)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-056 (SRD 5.2.1, "Eldritch Spear", p. 73) - the range grows by 30 feet per Warlock level, for the chosen cantrip only.
// Path: per case.
func TestEldritchSpearRangeFor(t *testing.T) {
	t.Parallel()
	if got, ok := character.EldritchSpearRangeFor("eldritch_blast", "eldritch_blast", character.MustWarlockLevel(5)); got != 150 || !ok {
		t.Errorf("chosen: %d, %v", got, ok)
	}
	if got, ok := character.EldritchSpearRangeFor("eldritch_blast", "poison_spray", character.MustWarlockLevel(5)); got != 0 || ok {
		t.Errorf("other: %d, %v", got, ok)
	}
}
