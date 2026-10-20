package content_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

func spells(ids ...string) []content.SpellID {
	var out []content.SpellID
	for _, id := range ids {
		out = append(out, content.MustSpellID(id))
	}
	return out
}

// Requirement: doc/specs/content-classes.md FR-051 to FR-066 - the fact,
// option, table and feature entry constructors: happy values, each rejection by
// block and problem, copy on read.
// Path: per case.
func TestClassFeatureConstructors(t *testing.T) {
	t.Parallel()
	grant := mustValue(content.NewSpellGrant(3, spells("a", "b")))
	spellsFact := mustValue(content.NewAlwaysPreparedSpells([]content.SpellGrant{grant}))
	feat := mustValue(content.NewGrantsFeat(content.FeatGeneral))
	text := mustValue(content.NewTableCellText("x"))
	num := mustValue(content.NewTableCellInt(2))
	opt := func(id string, f content.FeatureOptionFields) content.FeatureOption {
		return mustValue(content.NewFeatureOption(content.FeatureOptionID(id), "Opt "+id, f))
	}
	ok := content.FeatureOptionFields{Text: "t"}
	entry := func(f content.FeatureEntryFields) error {
		return errOf(content.NewFeatureEntry("a_b", "Name", f))
	}
	good := content.FeatureEntryFields{Level: 1, Text: "t"}
	with := func(edit func(*content.FeatureEntryFields)) content.FeatureEntryFields {
		f := good
		edit(&f)
		return f
	}
	cases := []blockCase{
		{"grants feat invalid", errOf(content.NewGrantsFeat(0)), "GrantsFeat", content.BlockInvalidMember},
		{"grant level 0", errOf(content.NewSpellGrant(0, spells("a"))), "SpellGrant", content.BlockOutOfRange},
		{"grant level 21", errOf(content.NewSpellGrant(21, spells("a"))), "SpellGrant", content.BlockOutOfRange},
		{"grant no spells", errOf(content.NewSpellGrant(1, nil)), "SpellGrant", content.BlockOutOfRange},
		{"grant repeated spell", errOf(content.NewSpellGrant(1, spells("a", "a"))), "SpellGrant", content.BlockDuplicate},
		{"grant invalid spell", errOf(content.NewSpellGrant(1, []content.SpellID{"Bad"})), "SpellGrant", content.BlockInvalidMember},
		{"always prepared empty", errOf(content.NewAlwaysPreparedSpells(nil)), "AlwaysPreparedSpells", content.BlockEmpty},
		{"always prepared invalid grant", errOf(content.NewAlwaysPreparedSpells([]content.SpellGrant{{}})), "AlwaysPreparedSpells", content.BlockInvalidMember},
		{"always prepared same level twice", errOf(content.NewAlwaysPreparedSpells([]content.SpellGrant{grant, grant})), "AlwaysPreparedSpells", content.BlockInvalidMember},
		{"prerequisite level 21", errOf(content.NewOptionPrerequisite(21, nil, "")), "OptionPrerequisite", content.BlockOutOfRange},
		{"prerequisite empty ids", errOf(content.NewOptionPrerequisite(0, []content.FeatureOptionID{}, "")), "OptionPrerequisite", content.BlockOutOfRange},
		{"prerequisite repeated ids", errOf(content.NewOptionPrerequisite(0, []content.FeatureOptionID{"a", "a"}, "")), "OptionPrerequisite", content.BlockDuplicate},
		{"prerequisite invalid id", errOf(content.NewOptionPrerequisite(0, []content.FeatureOptionID{"A"}, "")), "OptionPrerequisite", content.BlockInvalidMember},
		{"prerequisite blank note", errOf(content.NewOptionPrerequisite(1, nil, " ")), "OptionPrerequisite", content.BlockBlankText},
		{"prerequisite nothing", errOf(content.NewOptionPrerequisite(0, nil, "")), "OptionPrerequisite", content.BlockConstraint},
		{"option invalid id", errOf(content.NewFeatureOption("Bad", "A", ok)), "", 0},
		{"option blank name", errOf(content.NewFeatureOption("a", "", ok)), "FeatureOption", content.BlockInvalidName},
		{"option blank text", errOf(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: " "})), "FeatureOption", content.BlockBlankText},
		{"option negative cost", errOf(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: "t", PointCost: -1})), "FeatureOption", content.BlockOutOfRange},
		{"option invalid fact", errOf(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: "t", Facts: []content.FeatureFact{content.GrantsFeat{}}})), "FeatureOption", content.BlockInvalidMember},
		{"option subclass fact", errOf(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: "t", Facts: []content.FeatureFact{content.GrantsSubclass{}}})), "FeatureOption", content.BlockConstraint},
		{"cell blank", errOf(content.NewTableCellText(" ")), "TableCellText", content.BlockBlankText},
		{"cell beyond cap", errOf(content.NewTableCellInt(over())), "TableCellInt", content.BlockOutOfRange},
		{"table blank name", errOf(content.NewFeatureTable("", []string{"A"}, [][]content.TableCell{{text}})), "FeatureTable", content.BlockInvalidName},
		{"table no columns", errOf(content.NewFeatureTable("T", nil, [][]content.TableCell{{text}})), "FeatureTable", content.BlockEmpty},
		{"table no rows", errOf(content.NewFeatureTable("T", []string{"A"}, nil)), "FeatureTable", content.BlockEmpty},
		{"table blank header", errOf(content.NewFeatureTable("T", []string{" "}, [][]content.TableCell{{text}})), "FeatureTable", content.BlockBlankText},
		{"table ragged", errOf(content.NewFeatureTable("T", []string{"A", "B"}, [][]content.TableCell{{text, num}, {text}})), "FeatureTable", content.BlockRaggedRow},
		{"table invalid cell", errOf(content.NewFeatureTable("T", []string{"A"}, [][]content.TableCell{{content.TableCellText{}}})), "FeatureTable", content.BlockInvalidMember},
		{"table nil cell", errOf(content.NewFeatureTable("T", []string{"A"}, [][]content.TableCell{{nil}})), "FeatureTable", content.BlockInvalidMember},
		{"table mixed column", errOf(content.NewFeatureTable("T", []string{"A"}, [][]content.TableCell{{text}, {num}})), "FeatureTable", content.BlockConstraint},
		{"entry level 0", entry(with(func(f *content.FeatureEntryFields) { f.Level = 0 })), "FeatureEntry", content.BlockOutOfRange},
		{"entry level 21", entry(with(func(f *content.FeatureEntryFields) { f.Level = 21 })), "FeatureEntry", content.BlockOutOfRange},
		{"entry blank text", entry(with(func(f *content.FeatureEntryFields) { f.Text = " " })), "FeatureEntry", content.BlockBlankText},
		{"entry invalid fact", entry(with(func(f *content.FeatureEntryFields) { f.Facts = []content.FeatureFact{nil} })), "FeatureEntry", content.BlockInvalidMember},
		{"entry duplicate fact", entry(with(func(f *content.FeatureEntryFields) { f.Facts = []content.FeatureFact{feat, feat} })), "FeatureEntry", content.BlockDuplicate},
		{"entry duplicate spells fact", entry(with(func(f *content.FeatureEntryFields) { f.Facts = []content.FeatureFact{spellsFact, spellsFact} })), "FeatureEntry", content.BlockDuplicate},
		{"entry invalid option", entry(with(func(f *content.FeatureEntryFields) { f.Options = []content.FeatureOption{{}} })), "FeatureEntry", content.BlockInvalidMember},
		{"entry duplicate option id", entry(with(func(f *content.FeatureEntryFields) {
			f.Options = []content.FeatureOption{opt("a", ok), opt("a", ok)}
		})), "FeatureEntry", content.BlockDuplicate},
		{"entry invalid table", entry(with(func(f *content.FeatureEntryFields) { f.Tables = []content.FeatureTable{{}} })), "FeatureEntry", content.BlockInvalidMember},
		{"entry grant below the level", entry(with(func(f *content.FeatureEntryFields) {
			f.Level = 5
			f.Facts = []content.FeatureFact{spellsFact}
		})), "FeatureEntry", content.BlockInvalidMember},
		{"entry unknown required option", entry(with(func(f *content.FeatureEntryFields) {
			f.Options = []content.FeatureOption{opt("a", content.FeatureOptionFields{Text: "t", Prerequisite: mustValue(content.NewOptionPrerequisite(0, []content.FeatureOptionID{"zz"}, ""))})}
		})), "FeatureEntry", content.BlockConstraint},
		{"entry invalid id", errOf(content.NewFeatureEntry("Bad", "N", good)), "", 0},
		{"entry blank name", errOf(content.NewFeatureEntry("a_b", "", good)), "FeatureEntry", content.BlockInvalidName},
	}
	for _, c := range cases {
		if c.block == "" {
			if c.err == nil {
				t.Errorf("%s: accepted [unhappy (invalid input)]", c.name)
			}
			continue
		}
		wantBlockError(t, c.err, c.block, c.want)
	}
	// Happy values and immutability.
	if feat.Category() != content.FeatGeneral || grant.Level() != 3 || !slices.Equal(grant.SpellIDs(), spells("a", "b")) || len(spellsFact.Grants()) != 1 {
		t.Error("fact accessors lost their data [happy]")
	}
	ids := grant.SpellIDs()
	ids[0] = "zz"
	if !slices.Equal(grant.SpellIDs(), spells("a", "b")) {
		t.Error("mutating SpellIDs() changed the grant [unhappy (immutability)]")
	}
	req := []content.FeatureOptionID{"a", "b"}
	pre := mustValue(content.NewOptionPrerequisite(5, req, "note"))
	req[0] = "zz"
	if pre.MinClassLevel() != 5 || pre.Note() != "note" || !slices.Equal(pre.RequiresOptionIDs(), []content.FeatureOptionID{"a", "b"}) {
		t.Errorf("prerequisite = %+v [unhappy (immutability)]", pre)
	}
	full := opt("z", content.FeatureOptionFields{Text: "t", Prerequisite: pre, PointCost: 2, Repeatable: true, Facts: []content.FeatureFact{spellsFact}})
	if p, has := full.Prerequisite(); !has || p.Note() != "note" || full.PointCost() != 2 || !full.Repeatable() || len(full.Facts()) != 1 || full.ID() != "z" || full.Name() != "Opt z" || full.Text() != "t" {
		t.Errorf("option = %+v [happy]", full)
	}
	if _, has := opt("b", ok).Prerequisite(); has {
		t.Error("an option without a prerequisite has one [happy]")
	}
	cells := [][]content.TableCell{{text, num}, {text, num}}
	tbl := mustValue(content.NewFeatureTable("T", []string{"A", "B"}, cells))
	cells[0][0] = nil
	rows := tbl.Rows()
	rows[1][1] = nil
	cols := tbl.Columns()
	cols[0] = "zz"
	if tbl.Name() != "T" || !slices.Equal(tbl.Columns(), []string{"A", "B"}) || tbl.Rows()[0][0] != content.TableCell(text) || tbl.Rows()[1][1] != content.TableCell(num) || text.Text() != "x" || num.Value() != 2 {
		t.Errorf("table = %+v [unhappy (immutability)]", tbl)
	}
	e := mustValue(content.NewFeatureEntry("a_b", "Name", content.FeatureEntryFields{
		Level: 3, Text: "t", Facts: []content.FeatureFact{content.GrantsSubclass{}, feat}, Options: []content.FeatureOption{opt("a", ok), opt("b", ok), full}, Tables: []content.FeatureTable{tbl},
	}))
	if e.ID() != "a_b" || e.Name() != "Name" || e.Level() != 3 || e.Text() != "t" || len(e.Facts()) != 2 || len(e.Options()) != 3 || len(e.Tables()) != 1 {
		t.Errorf("entry = %+v [happy]", e)
	}
	e.Facts()[0] = nil
	if e.Facts()[0] == nil {
		t.Error("mutating Facts() changed the entry [unhappy (immutability)]")
	}
	// A grant at the entry level, an option required by a later option, and a chain.
	chain := mustValue(content.NewFeatureEntry("a_b", "Name", content.FeatureEntryFields{Level: 3, Text: "t", Options: []content.FeatureOption{
		opt("x", content.FeatureOptionFields{Text: "t", Prerequisite: mustValue(content.NewOptionPrerequisite(0, []content.FeatureOptionID{"y"}, ""))}),
		opt("y", ok),
	}, Facts: []content.FeatureFact{spellsFact}}))
	if len(chain.Options()) != 2 {
		t.Error("a prerequisite chain was rejected [happy]")
	}
}

// Requirement: doc/specs/content-classes.md FR-051 to FR-066 - the bounds of the
// fact, option, cell and table constructors, and the distinctness rules, at and
// either side of each limit.
// Path: unhappy (boundary).
func TestClassFeatureBoundaries(t *testing.T) {
	t.Parallel()
	grant := func(level int) content.SpellGrant { return mustValue(content.NewSpellGrant(level, spells("a"))) }
	feat := mustValue(content.NewGrantsFeat(content.FeatGeneral))
	asSpells := func(gs ...content.SpellGrant) content.FeatureFact {
		return mustValue(content.NewAlwaysPreparedSpells(gs))
	}
	opt := func(id string) content.FeatureOption {
		return mustValue(content.NewFeatureOption(content.FeatureOptionID(id), "O", content.FeatureOptionFields{Text: "t"}))
	}
	entry := func(f content.FeatureEntryFields) error { return errOf(content.NewFeatureEntry("a_b", "N", f)) }
	for _, level := range []int{1, 20} {
		if _, err := content.NewSpellGrant(level, spells("a")); err != nil {
			t.Errorf("NewSpellGrant(%d) = %v [unhappy (boundary)]", level, err)
		}
	}
	for _, c := range []blockCase{
		{"grants descending", errOf(content.NewAlwaysPreparedSpells([]content.SpellGrant{grant(5), grant(3)})), "AlwaysPreparedSpells", content.BlockInvalidMember},
		{"grants equal", errOf(content.NewAlwaysPreparedSpells([]content.SpellGrant{grant(3), grant(3)})), "AlwaysPreparedSpells", content.BlockInvalidMember},
		{"prerequisite level -1", errOf(content.NewOptionPrerequisite(-1, nil, "n")), "OptionPrerequisite", content.BlockOutOfRange},
		{"point cost beyond the cap", errOf(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: "t", PointCost: over()})), "FeatureOption", content.BlockOutOfRange},
		{"cell below the cap", errOf(content.NewTableCellInt(-over())), "TableCellInt", content.BlockOutOfRange},
		{"row longer than the header", errOf(content.NewFeatureTable("T", []string{"A"}, [][]content.TableCell{{content.TableCellInt{}, content.TableCellInt{}}})), "FeatureTable", content.BlockRaggedRow},
	} {
		wantBlockError(t, c.err, c.block, c.want)
	}
	for _, ok := range []error{
		errOf(content.NewAlwaysPreparedSpells([]content.SpellGrant{grant(3), grant(5)})),
		errOf(content.NewOptionPrerequisite(1, nil, "")), errOf(content.NewOptionPrerequisite(20, nil, "")),
		errOf(content.NewOptionPrerequisite(0, nil, "a note")), errOf(content.NewOptionPrerequisite(5, nil, "")),
		errOf(content.NewOptionPrerequisite(0, []content.FeatureOptionID{"a"}, "")),
		errOf(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: "t", PointCost: content.ClassMaxCount})),
		errOf(content.NewTableCellInt(content.ClassMaxCount)), errOf(content.NewTableCellInt(-content.ClassMaxCount)),
		// Facts of different kinds, and spell facts that differ, are distinct.
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Facts: []content.FeatureFact{feat, asSpells(grant(3))}}),
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Facts: []content.FeatureFact{asSpells(grant(3)), feat}}),
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Facts: []content.FeatureFact{asSpells(grant(3)), asSpells(grant(4))}}),
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Facts: []content.FeatureFact{asSpells(grant(4)), asSpells(grant(3))}}),
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Facts: []content.FeatureFact{asSpells(grant(3)), asSpells(mustValue(content.NewSpellGrant(3, spells("b"))))}}),
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Facts: []content.FeatureFact{asSpells(grant(3)), asSpells(grant(3), grant(4))}}),
		// Distinct option ids in any order.
		entry(content.FeatureEntryFields{Level: 3, Text: "t", Options: []content.FeatureOption{opt("b"), opt("a")}}),
	} {
		if ok != nil {
			t.Errorf("valid value rejected: %v [unhappy (boundary)]", ok)
		}
	}
	// Each part of a prerequisite alone is a prerequisite.
	for i, p := range []content.OptionPrerequisite{
		mustValue(content.NewOptionPrerequisite(5, nil, "")), mustValue(content.NewOptionPrerequisite(0, []content.FeatureOptionID{"a"}, "")), mustValue(content.NewOptionPrerequisite(0, nil, "n")),
	} {
		o := mustValue(content.NewFeatureOption("a", "A", content.FeatureOptionFields{Text: "t", Prerequisite: p}))
		if got, has := o.Prerequisite(); !has || got.MinClassLevel() != p.MinClassLevel() || got.Note() != p.Note() {
			t.Errorf("prerequisite %d lost on the option [happy]", i)
		}
	}
}
