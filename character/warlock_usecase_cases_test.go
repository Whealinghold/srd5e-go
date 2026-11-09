package character_test

import (
	"reflect"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// warMonsterSource is a content source holding the monsters the familiar tests use.
func warMonsterSource(tb testing.TB) character.ContentSource {
	tb.Helper()
	return contenttest.NewFakeSource(warMonster(tb, "imp", content.CreatureTypeFiend, "1"), warMonster(tb, "wolf", content.CreatureTypeBeast, "1/4"), warMonster(tb, "cat", content.CreatureTypeBeast, "0"))
}

// warWeaponSource is a content source holding a Melee and a Ranged weapon and a piece of armor-like non-weapon entry.
func warWeaponSource(tb testing.TB) character.ContentSource {
	tb.Helper()
	return contenttest.NewFakeSource(
		wpn("longsword", content.WeaponMartial, content.WeaponMelee, diceDamage(1, dice.D8(), content.DamageSlashing), content.MasterySap),
		wpn("longbow", content.WeaponMartial, content.WeaponRanged, diceDamage(1, dice.D8(), content.DamagePiercing), content.MasterySap))
}

// warMoreCases are the use cases of the finishing step, driven like warCases in the three modes.
func warMoreCases() []warCase {
	return []warCase{
		{name: "ChooseWarlockMysticArcanum", span: character.SpanChooseWarlockMysticArcanum, feature: "mystic_arcanum", rejection: "wrong_spell_level", loads: true,
			run: func(t *testing.T, mode warMode, src character.ContentSource, _ character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 17, 16, "", character.WarlockState{})
				if mode == warNot {
					c = warFighter(t)
				}
				spell := content.SpellID("power_word_kill")
				if mode == warBad {
					spell = "fire_bolt"
				}
				return must(character.NewChooseWarlockMysticArcanum(src, opts...)).Run(t.Context(), character.ChooseWarlockMysticArcanumCommand{Character: c, Spell: spell})
			}},
		{name: "CastInvocationSpell", span: character.SpanCastInvocationSpell, feature: "invocation_spell", rejection: "invocation_not_known", loads: true,
			run: func(t *testing.T, mode warMode, src character.ContentSource, _ character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 5, 16, "", warKnows("armor_of_shadows"))
				if mode == warNot {
					c = warFighter(t)
				}
				option := content.FeatureOptionID("armor_of_shadows")
				if mode == warBad {
					option = "misty_visions"
				}
				return must(character.NewCastInvocationSpell(src, opts...)).Run(t.Context(), character.CastInvocationSpellCommand{Character: c, Option: option, Light: content.LightBright})
			}},
		{name: "ChooseWarlockFamiliarForm", span: character.SpanChooseWarlockFamiliarForm, feature: "familiar_form", rejection: "familiar_form_not_offered", loads: true,
			run: func(t *testing.T, mode warMode, _ character.ContentSource, _ character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 5, 16, "", warKnows("pact_of_the_chain"))
				if mode == warNot {
					c = warFighter(t)
				}
				form := character.SpecialFamiliarForm("imp")
				if mode == warBad {
					form = character.SpecialFamiliarForm("wolf")
				}
				return must(character.NewChooseWarlockFamiliarForm(warMonsterSource(t), opts...)).Run(t.Context(), character.ChooseWarlockFamiliarFormCommand{Character: c, Form: form})
			}},
		{name: "ConjureBookOfShadows", span: character.SpanConjureBookOfShadows, feature: "book_of_shadows", rejection: "not_after_rest", loads: true,
			run: func(t *testing.T, mode warMode, src character.ContentSource, _ character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 5, 16, "", warKnows("pact_of_the_tome"))
				if mode == warNot {
					c = warFighter(t)
				}
				rest := character.ShortRest
				if mode == warBad {
					rest = 0
				}
				return must(character.NewConjureBookOfShadows(src, opts...)).Run(t.Context(), character.ConjureBookOfShadowsCommand{Character: c, Rest: rest,
					Cantrips: []content.SpellID{"acid_splash", "dancing_lights", "druidcraft"}, Spells: []content.SpellID{"detect_magic", "speak_with_animals"}})
			}},
		{name: "BondPactWeapon", span: character.SpanBondPactWeapon, feature: "pact_weapon", rejection: "not_melee_simple_or_martial", loads: true,
			run: func(t *testing.T, mode warMode, _ character.ContentSource, _ character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 5, 16, "", warKnows("pact_of_the_blade"))
				if mode == warNot {
					c = warFighter(t)
				}
				weapon := content.ItemID("longsword")
				if mode == warBad {
					weapon = "longbow"
				}
				return must(character.NewBondPactWeapon(warWeaponSource(t), opts...)).Run(t.Context(), character.BondPactWeaponCommand{Character: c, Weapon: weapon})
			}},
		{name: "UseLifedrinker", span: character.SpanUseLifedrinker, feature: "lifedrinker", rejection: "damage_type_not_offered",
			run: func(t *testing.T, mode warMode, _ character.ContentSource, roller character.Roller, opts ...character.Option) (any, error) {
				t.Helper()
				c := warChar(t, 5, 16, "", warKnows("lifedrinker"))
				if mode == warNot {
					c = warFighter(t)
				}
				dt := content.DamageNecrotic
				if mode == warBad {
					dt = content.DamageFire
				}
				return must(character.NewUseLifedrinker(roller, opts...)).Run(t.Context(), character.UseLifedrinkerCommand{Character: c, Attack: character.WarlockPactWeaponHit, DamageType: dt,
					Choice: character.HitDieExpend, Pool: warPool(t, 3), Die: content.HitDieD8, Constitution: warMod(14)})
			}},
	}
}

// warContentFailures asserts the content failures of a use case that loads entries: an empty source gives the ContentMissing variant listing
// want (error status, rejection content_missing); a failing source and a source answering with another entry are PortErrors recorded once.
func warContentFailures(t *testing.T, span, feature string, want []content.MissingContent, run func(character.ContentSource, ...character.Option) (any, error), missingOf func(any) ([]content.MissingContent, bool)) {
	t.Helper()
	rec, opts := warTelemetry()
	res, err := run(contenttest.NewFakeSource(), opts...)
	got, ok := missingOf(res)
	if err != nil || !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("missing = %#v (%v), %v; want %v", res, got, err, want)
	}
	warCheck(t, rec, warCall(span, feature, "content_missing", telemetry.StatusError, ""))
	for name, src := range map[string]character.ContentSource{"an error": contenttest.NewFakeSource().FailLookup(errBoom), "wrong entry": wrongEntrySource{entry: warSource(t)}} {
		rec, opts = warTelemetry()
		res, err = run(src, opts...)
		var pe *srderr.PortError
		if res != nil || !asPortError(err, &pe) || pe.Port != content.Port {
			t.Fatalf("%s: = %v, %v; want a PortError for content", name, res, err)
		}
		warCheck(t, rec, warWant{span: span, feature: feature, outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err), loaded: true})
	}
}
