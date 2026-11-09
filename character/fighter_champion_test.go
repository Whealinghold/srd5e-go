package character_test

import (
	"strconv"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

const champion = character.ChampionSubclassID

// Requirement: doc/specs/class-fighter.md Table G, FR-087, FR-088, FR-089, FR-090, FR-130, FR-131, FR-132 - the critical-hit threshold is 19 for
// a Champion of Fighter level 3 to 14 and 18 from level 15 on weapon attack rolls and Unarmed Strikes, 20 for every other case (no
// subclass at any level, another subclass, an attack roll of another kind); a Champion below level 3 or an invalid kind is an error.
// Path: per case (rows CR1 to CR9 and the level boundaries).
func TestChampionCriticalHitThreshold(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		sub   content.SubclassID
		level int
		kind  character.ChampionAttackKind
		want  int
	}{
		{"CR2", champion, 3, character.ChampionWeaponAttack, 19}, {"CR3", champion, 3, character.ChampionUnarmedStrike, 19},
		{"CR4", champion, 14, character.ChampionWeaponAttack, 19}, {"CR5", champion, 15, character.ChampionWeaponAttack, 18},
		{"CR6", champion, 20, character.ChampionUnarmedStrike, 18}, {"CR7", "", 3, character.ChampionWeaponAttack, 20},
		{"CR8", "", 15, character.ChampionWeaponAttack, 20}, {"CR9", champion, 15, character.ChampionOtherAttack, 20},
		{"another subclass", "another", 20, character.ChampionWeaponAttack, 20}, {"no subclass at level 1", "", 1, character.ChampionOtherAttack, 20},
		{"the Champion's other attack at 3", champion, 3, character.ChampionOtherAttack, 20},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.ChampionEffects{}.CriticalHitThreshold(c.sub, character.MustFighterLevel(c.level), c.kind)
			if err != nil || got != c.want {
				t.Errorf("= %d, %v; want %d", got, err, c.want)
			}
		})
	}
	for n := 1; n <= 2; n++ {
		_, err := character.ChampionEffects{}.CriticalHitThreshold(champion, character.MustFighterLevel(n), character.ChampionWeaponAttack)
		wantInvalid(t, err, "CR1 Champion at level "+strconv.Itoa(n))
	}
	_, err := character.ChampionEffects{}.CriticalHitThreshold(champion, character.MustFighterLevel(5), 0)
	wantInvalid(t, err, "invalid kind")
	_, err = character.ChampionEffects{}.CriticalHitThreshold("", character.MustFighterLevel(5), 4)
	wantInvalid(t, err, "invalid kind without a subclass")
	for k, name := range map[character.ChampionAttackKind]string{character.ChampionWeaponAttack: "weapon", character.ChampionUnarmedStrike: "unarmed_strike", character.ChampionOtherAttack: "other", 0: "unknown"} {
		if k.String() != name {
			t.Errorf("kind %d = %q, want %q", int(k), k.String(), name)
		}
	}
}

// Requirement: doc/specs/class-fighter.md FR-091, FR-092 - at Champion level 3 or more the character has Advantage on Initiative rolls and
// on Strength (Athletics) checks, two declared Advantage sources, and the move after a Critical Hit is offered; below level 3 or without
// the Champion neither is.
// Path: per case (unhappy: boundary, state error).
func TestChampionAdvantageSourcesAndCriticalHitMove(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		sub   content.SubclassID
		level int
		has   bool
	}{{"Champion level 2", champion, 2, false}, {"Champion level 3", champion, 3, true}, {"Champion level 20", champion, 20, true}, {"no subclass", "", 20, false}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			sources := character.ChampionEffects{}.AdvantageSources(c.sub, character.MustFighterLevel(c.level))
			_, moves := character.ChampionEffects{}.CriticalHitMove(c.sub, character.MustFighterLevel(c.level)).(character.TacticalShiftOffered)
			if moves != c.has || (len(sources) == 2) != c.has || (len(sources) == 0) == c.has || sources == nil {
				t.Fatalf("sources %v, move %v; want present %v", sources, moves, c.has)
			}
			if !c.has {
				return
			}
			skill := sources[1].(character.ChampionSkillCheckAdvantage)
			if _, ok := sources[0].(character.ChampionInitiativeAdvantage); !ok || skill.Skill != content.SkillAthletics || skill.Ability != ability.Strength {
				t.Errorf("sources = %v", sources)
			}
			for _, s := range sources {
				if _, ok := s.Circumstance().(modifier.Advantage); !ok {
					t.Errorf("source %v gives %v, want Advantage", s, s.Circumstance())
				}
			}
		})
	}
}

// Requirement: doc/specs/class-fighter.md Table H, FR-095, FR-096, FR-100, FR-137 - Defy Death gives a Champion of Fighter level 18 or more
// Advantage on Death Saving Throws and counts a natural 18 to 20 as 20, with the benefit of a 20 (1 Hit Point); at level 17, without
// the Champion and for a natural roll below 18 it does not; a natural roll outside 1 to 20 is an error.
// Path: per case (rows DD1 to DD6 and the boundaries).
func TestChampionDefyDeath(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name    string
		sub     content.SubclassID
		level   int
		natural int
		counts  int
		advant  bool
	}{
		{"DD1", champion, 18, 17, 17, true}, {"DD2", champion, 18, 18, 20, true}, {"DD3", champion, 18, 19, 20, true}, {"DD4", champion, 18, 20, 20, true},
		{"DD5", champion, 17, 18, 18, false}, {"DD6", "", 20, 18, 18, false}, {"natural 1", champion, 18, 1, 1, true}, {"a natural 20 at 17", champion, 17, 20, 20, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d, err := character.ChampionEffects{}.DefyDeath(c.sub, character.MustFighterLevel(c.level), c.natural)
			if err != nil || d.CountsAs() != c.counts || d.Natural() != c.natural || d.AdvantageOnDeathSaves() != c.advant || d.RegainsOneHitPoint() != (c.counts == 20) {
				t.Errorf("= %+v, %v; want counts %d advantage %v", d, err, c.counts, c.advant)
			}
		})
	}
	for _, n := range []int{0, 21} {
		_, err := character.ChampionEffects{}.DefyDeath(champion, character.MustFighterLevel(18), n)
		wantInvalid(t, err, "natural "+strconv.Itoa(n))
	}
	for _, c := range []struct {
		sub   content.SubclassID
		level int
		want  bool
	}{{champion, 17, false}, {champion, 18, true}, {champion, 19, true}, {champion, 20, true}, {"", 20, false}, {"another", 20, false}} {
		if got := (character.ChampionEffects{}).HasSurvivor(c.sub, character.MustFighterLevel(c.level)); got != c.want {
			t.Errorf("HasSurvivor(%q, %d) = %v, want %v", c.sub, c.level, got, c.want)
		}
	}
}

// Requirement: doc/specs/class-fighter.md FR-093, FR-094, FR-135 - Heroic Warrior gives Heroic Inspiration to a Champion of Fighter level 10
// or more who is NotInspired and in combat, with the trigger start of each of the character's turns; otherwise the first reason of
// featureNotGained (below level 10 or without the Champion), notInCombat, alreadyHasInspiration.
// Path: per case (unhappy: boundary, state error).
func TestHeroicWarriorEvaluate(t *testing.T) {
	t.Parallel()
	in, out := character.FighterInCombat, character.FighterOutOfCombat
	for _, c := range []struct {
		name     string
		sub      content.SubclassID
		level    int
		insp     character.HeroicInspiration
		presence character.FighterCombatPresence
		want     character.FighterReason
	}{
		{"level 9", champion, 9, character.NotInspired, in, character.FighterFeatureNotGained},
		{"no subclass", "", 20, character.NotInspired, in, character.FighterFeatureNotGained},
		{"out of combat", champion, 10, character.NotInspired, out, character.FighterNotInCombat},
		{"out of combat and inspired", champion, 10, character.Inspired, out, character.FighterNotInCombat},
		{"already inspired", champion, 10, character.Inspired, in, character.FighterAlreadyHasInspiration},
		{"level 10 in combat", champion, 10, character.NotInspired, in, 0},
		{"level 20 in combat", champion, 20, character.NotInspired, in, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := character.HeroicWarrior{}.Evaluate(c.sub, character.MustFighterLevel(c.level), c.insp, c.presence)
			if c.want == 0 {
				if give, ok := got.(character.HeroicWarriorGive); !ok || give.Trigger() != character.TriggerStartOfEachOfYourTurns {
					t.Errorf("= %v, want a give at the start of each turn", got)
				}
				return
			}
			if un, ok := got.(character.HeroicWarriorNotAvailable); !ok || un.Reason != c.want {
				t.Errorf("= %v, want not available %v", got, c.want)
			}
		})
	}
	if got, ok := (character.HeroicWarrior{}).Evaluate(champion, character.MustFighterLevel(10), character.NotInspired, character.FighterCombatPresence(0)).(character.HeroicWarriorNotAvailable); !ok || got.Reason != character.FighterNotInCombat {
		t.Errorf("an invalid presence = %v, want not in combat", got)
	}
	if character.FighterCombatPresence(0).String() != "unknown" || character.FighterCombatPresence(5).String() != "unknown" || in == 0 {
		t.Error("the zero and out-of-range presences must be unknown, and the zero value is no presence")
	}
	for v, name := range map[character.FighterCombatPresence]string{in: "in_combat", out: "not_in_combat", 0: "unknown"} {
		if v.String() != name {
			t.Errorf("presence %d = %q, want %q", int(v), v.String(), name)
		}
	}
}

// Requirement: doc/specs/class-fighter.md Table I, FR-097, FR-098, FR-099, FR-133, FR-134 - Heroic Rally regains 5 plus the Constitution modifier
// (never below 0) when Bloodied with at least 1 Hit Point (HR1, HR3, HR6, HR7), is not eligible at 0 Hit Points (HR4) or when not Bloodied
// (HR2, HR5, pinning the one Bloodied predicate), and refuses current Hit Points above the maximum (HR8), a negative amount and the zero
// maximum with an error; the trigger is the start of each turn.
// Path: per case (rows HR1 to HR8).
func TestHeroicRallyHealing(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name        string
		con, hp, hm int
		want        int
		reason      character.FighterReason
	}{
		{"HR1", 2, 10, 20, 7, 0}, {"HR2", 2, 11, 20, 0, character.FighterNotBloodied}, {"HR3", -5, 5, 20, 0, 0},
		{"HR4", 2, 0, 20, 0, character.FighterNoHitPoints}, {"HR5", 2, 1, 1, 0, character.FighterNotBloodied},
		{"HR6", 2, 1, 2, 7, 0}, {"HR7", 10, 10, 20, 15, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := character.HeroicRallyHealing{}.Healing(modifierOf(t, c.con), c.hp, character.MustHitPointMaximum(c.hm))
			if err != nil {
				t.Fatal(err)
			}
			if c.reason != 0 {
				if ne, ok := got.(character.HeroicRallyNotEligible); !ok || ne.Reason != c.reason {
					t.Errorf("= %v, want not eligible %v", got, c.reason)
				}
				return
			}
			if r, ok := got.(character.HeroicRallyRegains); !ok || r.HitPoints() != c.want || r.Trigger() != character.TriggerStartOfEachOfYourTurns {
				t.Errorf("= %v, want regains %d", got, c.want)
			}
		})
	}
	hm := character.MustHitPointMaximum(20)
	for name, hp := range map[string]int{"HR8 above the maximum": 21, "negative": -1} {
		_, err := character.HeroicRallyHealing{}.Healing(modifierOf(t, 2), hp, hm)
		wantInvalid(t, err, name)
	}
	_, err := character.HeroicRallyHealing{}.Healing(modifierOf(t, 2), 0, character.HitPointMaximum{})
	wantInvalid(t, err, "zero maximum")
}

// Requirement: doc/specs/class-fighter.md FR-100, FR-137 - ChampionEffects.HeroicRally reports the Heroic Rally only for a Champion of Fighter
// level 18 or more and false below level 18 or without the Champion, and passes the errors of the calculation through.
// Path: per case (unhappy: boundary, state error).
func TestChampionHeroicRallyIsGatedBySurvivor(t *testing.T) {
	t.Parallel()
	hm := character.MustHitPointMaximum(20)
	for _, c := range []struct {
		name  string
		sub   content.SubclassID
		level int
		has   bool
	}{{"level 17", champion, 17, false}, {"level 18", champion, 18, true}, {"no subclass", "", 20, false}} {
		out, has, err := character.ChampionEffects{}.HeroicRally(c.sub, character.MustFighterLevel(c.level), modifierOf(t, 2), 10, hm)
		if err != nil || has != c.has || (out != nil) != c.has {
			t.Errorf("%s: = %v, %v, %v; want reported %v", c.name, out, has, err, c.has)
		}
	}
	_, has, err := character.ChampionEffects{}.HeroicRally(champion, character.MustFighterLevel(18), modifierOf(t, 2), 21, hm)
	wantInvalid(t, err, "current Hit Points above the maximum")
	if has {
		t.Error("an error must not be reported as an effect")
	}
}

// Requirement: doc/specs/class-fighter.md FR-105 - the subclass features reported are those of the SubclassDefinition at or below the Fighter
// level (Champion: 3, 3, 7, 10, 15, 18), taken from the content and not restated.
// Path: per case (unhappy: boundary).
func TestChampionFeaturesAtFighterLevel(t *testing.T) {
	t.Parallel()
	def := fgtDef(t)
	sub := def.Subclasses()[0]
	for _, c := range []struct {
		level int
		want  []string
	}{
		{2, nil}, {3, []string{"Improved Critical", "Remarkable Athlete"}}, {7, []string{"Improved Critical", "Remarkable Athlete", "Additional Fighting Style"}},
		{14, []string{"Improved Critical", "Remarkable Athlete", "Additional Fighting Style", "Heroic Warrior"}},
		{20, []string{"Improved Critical", "Remarkable Athlete", "Additional Fighting Style", "Heroic Warrior", "Superior Critical", "Survivor"}},
	} {
		got, err := character.ChampionEffects{}.FeaturesAt(sub, character.MustFighterLevel(c.level))
		var names []string
		for _, f := range got {
			names = append(names, f.Name())
		}
		if err != nil || len(names) != len(c.want) {
			t.Errorf("level %d: %v, %v; want %v", c.level, names, err, c.want)
			continue
		}
		for i := range names {
			if names[i] != c.want[i] {
				t.Errorf("level %d: %v, want %v", c.level, names, c.want)
			}
		}
	}
}
