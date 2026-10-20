package content_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

const miSpec = "doc/specs/content-magic-items.md"

// The fixtures are synthetic: made-up ids, names and texts, never SRD data.

func miFields() content.MagicItemFields {
	return content.MagicItemFields{
		Category:    content.MagicItemRod,
		Rarity:      content.RarityVaries{},
		Description: []content.TextBlock{content.MustTextParagraph("Text.")},
	}
}

func miEntry(t *testing.T, f content.MagicItemFields) content.MagicItemEntry {
	t.Helper()
	e, err := content.NewMagicItemEntry("rod_x", "Rod X", content.SourceCitation{}, f)
	if err != nil {
		t.Fatalf("NewMagicItemEntry: %v", err)
	}
	return e
}

// Requirement: miSpec FR-011, FR-012, FR-014 - the nine categories and six
// rarities exist in table order with their printed wire spelling, matched
// exactly and case-sensitively; plurals, other case and padding are rejected.
// Path: per case.
func TestMagicItemCategoryAndRarityVocabulary(t *testing.T) {
	t.Parallel()
	categories := []content.MagicItemCategory{
		content.MagicItemArmor, content.MagicItemPotion, content.MagicItemRing, content.MagicItemRod, content.MagicItemScroll,
		content.MagicItemStaff, content.MagicItemWand, content.MagicItemWeapon, content.MagicItemWondrousItem,
	}
	names := []string{"Armor", "Potion", "Ring", "Rod", "Scroll", "Staff", "Wand", "Weapon", "Wondrous Item"}
	for i, c := range categories {
		if int(c) != i+1 || c.String() != names[i] || !c.Valid() {
			t.Errorf("category %d = %q valid=%v, want %q [FR-011]", c, c.String(), c.Valid(), names[i])
		}
		if got, err := content.ParseMagicItemCategory(names[i]); err != nil || got != c {
			t.Errorf("ParseMagicItemCategory(%q) = %v, %v [FR-012]", names[i], got, err)
		}
	}
	for _, bad := range []string{"", "Potions", "Wondrous Items", "armor", "WONDROUS ITEM", " Armor", "Rod "} {
		if _, err := content.ParseMagicItemCategory(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ParseMagicItemCategory(%q) = %v, want ErrInvalidInput [FR-012]", bad, err)
		}
	}
	if content.MagicItemCategory(0).Valid() || content.MagicItemCategory(10).Valid() || content.MagicItemCategory(10).String() != "unknown" {
		t.Error("an undefined category is valid or named [FR-011]")
	}
	rarities := []content.ItemRarity{
		content.RarityCommon, content.RarityUncommon, content.RarityRare, content.RarityVeryRare, content.RarityLegendary, content.RarityArtifact,
	}
	rnames := []string{"Common", "Uncommon", "Rare", "Very Rare", "Legendary", "Artifact"}
	for i, r := range rarities {
		if int(r) != i+1 || r.String() != rnames[i] || !r.Valid() {
			t.Errorf("rarity %d = %q, want %q [FR-014]", r, r.String(), rnames[i])
		}
		if got, err := content.ParseItemRarity(rnames[i]); err != nil || got != r {
			t.Errorf("ParseItemRarity(%q) = %v, %v", rnames[i], got, err)
		}
	}
	for _, bad := range []string{"", "Very rare", "very rare", "Mythic", " Rare", "Rarity Varies"} {
		if _, err := content.ParseItemRarity(bad); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("ParseItemRarity(%q) = %v, want ErrInvalidInput [FR-017]", bad, err)
		}
	}
	if content.ItemRarity(0).Valid() || content.ItemRarity(7).Valid() {
		t.Error("an undefined rarity is valid [FR-014]")
	}
}

// Requirement: miSpec FR-015, FR-016 - SingleRarity needs a member; a
// RarityOption needs a member rarity and a non-blank variant; RarityByVariant
// holds two or more options with distinct variants (a rarity may repeat) and
// copies them on the way in and out.
// Path: per case.
func TestRaritySpecConstructors(t *testing.T) {
	t.Parallel()
	if _, err := content.NewSingleRarity(0); err == nil {
		t.Error("NewSingleRarity(0) accepted [FR-015]")
	} else {
		wantBlockError(t, err, "SingleRarity", content.BlockInvalidMember)
	}
	if s, err := content.NewSingleRarity(content.RarityRare); err != nil || s.Rarity() != content.RarityRare {
		t.Errorf("NewSingleRarity(Rare) = %v, %v", s, err)
	}
	_, err := content.NewRarityOption(7, "+1")
	wantBlockError(t, err, "RarityOption", content.BlockInvalidMember)
	_, err = content.NewRarityOption(content.RarityRare, " ")
	wantBlockError(t, err, "RarityOption", content.BlockBlankText)
	_, err = content.NewRarityOption(content.RarityRare, "")
	wantBlockError(t, err, "RarityOption", content.BlockBlankText)
	_, err = content.NewRarityOption(content.RarityRare, "a\tb")
	wantBlockError(t, err, "RarityOption", content.BlockInvalidCharacter)
	opt := func(r content.ItemRarity, v string) content.RarityOption {
		o, err := content.NewRarityOption(r, v)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	a, b := opt(content.RarityRare, "+1"), opt(content.RarityRare, "+2")
	if a.Rarity() != content.RarityRare || a.Variant() != "+1" {
		t.Errorf("option = %v, %q", a.Rarity(), a.Variant())
	}
	for _, c := range []struct {
		name string
		opts []content.RarityOption
		want content.BlockProblem
	}{
		{"none", nil, content.BlockOutOfRange},
		{"one", []content.RarityOption{a}, content.BlockOutOfRange},
		{"repeated variant", []content.RarityOption{a, a}, content.BlockDuplicate},
		{"zero option", []content.RarityOption{a, {}}, content.BlockInvalidMember},
	} {
		_, err := content.NewRarityByVariant(c.opts)
		if err == nil {
			t.Errorf("%s: accepted [FR-016]", c.name)
			continue
		}
		wantBlockError(t, err, "RarityByVariant", c.want)
	}
	in := []content.RarityOption{a, b}
	v, err := content.NewRarityByVariant(in)
	if err != nil {
		t.Fatalf("two options with one rarity: %v [FR-016]", err)
	}
	in[0] = b
	out := v.Options()
	if out[0] != a {
		t.Error("the constructor kept the caller's slice [FR-008]")
	}
	out[1] = a
	if got := v.Options(); got[1] != b {
		t.Error("Options returned the internal slice [FR-008]")
	}
	if three, err := content.NewRarityByVariant(optionsOf(t, "+1", "+2", "+3")); err != nil || len(three.Options()) != 3 {
		t.Errorf("three options: %v [FR-016]", err)
	}
	w, _ := content.NewRarityByVariant([]content.RarityOption{a, b})
	if !v.Equal(w) || v.Equal(content.RarityByVariant{}) {
		t.Error("RarityByVariant.Equal is wrong [FR-009]")
	}
}

// Requirement: miSpec FR-018 - AttunementRequiredBy needs a non-blank
// verbatim prerequisite.
// Path: per case.
func TestAttunementRequiredBy(t *testing.T) {
	t.Parallel()
	_, err := content.NewAttunementRequiredBy("  ")
	wantBlockError(t, err, "AttunementRequiredBy", content.BlockBlankText)
	_, err = content.NewAttunementRequiredBy("")
	wantBlockError(t, err, "AttunementRequiredBy", content.BlockBlankText)
	_, err = content.NewAttunementRequiredBy("a\tb")
	wantBlockError(t, err, "AttunementRequiredBy", content.BlockInvalidCharacter)
	a, err := content.NewAttunementRequiredBy("a Wizard, or a Cleric")
	if err != nil || a.Prerequisite() != "a Wizard, or a Cleric" {
		t.Errorf("NewAttunementRequiredBy = %v, %v", a, err)
	}
}

// Requirement: miSpec FR-001, FR-003, FR-004, FR-040 - the entry has the key
// of its id under kind magicItem, implements the one marker, and the smallest
// valid entry has no note and NoAttunement.
// Path: happy.
func TestMagicItemEntryHappy(t *testing.T) {
	t.Parallel()
	e := miEntry(t, miFields())
	if e.Key() != content.NewKey(content.KindMagicItem, content.MustID("rod_x")) || e.ID() != "rod_x" || e.Name() != "Rod X" {
		t.Errorf("identity = %v %v %q [FR-001]", e.Key(), e.ID(), e.Name())
	}
	var entry content.Entry = e
	if _, ok := entry.(content.MagicItem); !ok {
		t.Error("MagicItemEntry does not implement MagicItem [FR-003]")
	}
	if e.CategoryNote() != "" || e.Attunement() != (content.NoAttunement{}) || e.Category() != content.MagicItemRod || e.Source() != (content.SourceCitation{}) {
		t.Errorf("minimal entry = %+v [FR-040]", e)
	}
	if _, ok := e.Rarity().(content.RarityVaries); !ok {
		t.Errorf("rarity = %T", e.Rarity())
	}
	f := miFields()
	f.CategoryNote = "Any Medium or Heavy, Except Hide Armor"
	f.Attunement = content.AttunementRequired{}
	src := content.MustSourceCitation("SRD 5.2.1", "Magic Items A–Z", 209)
	full, err := content.NewMagicItemEntry("rod_y", "Rod, +1, +2, or +3 (Y)", src, f)
	if err != nil || full.CategoryNote() != f.CategoryNote || full.Attunement() != (content.AttunementRequired{}) || full.Source() != src {
		t.Errorf("full entry = %+v, %v [FR-005, FR-013]", full, err)
	}
	if got := content.MustMagicItemEntry("rod_x", "Rod X", content.SourceCitation{}, miFields()); !got.Equal(e) {
		t.Error("MustMagicItemEntry differs from NewMagicItemEntry")
	}
}

// Requirement: miSpec FR-002, FR-004, FR-013, FR-016 to FR-019, FR-022 - each
// rule of the constructor, in the documented order.
// Path: per case.
func TestNewMagicItemEntryRejects(t *testing.T) {
	t.Parallel()
	with := func(f func(*content.MagicItemFields)) content.MagicItemFields {
		x := miFields()
		f(&x)
		return x
	}
	byVariant := content.RarityByVariant{}
	cases := []struct {
		name string
		id   content.MagicItemID
		item string
		f    content.MagicItemFields
		want content.BlockProblem
	}{
		{"empty name", "rod_x", "", miFields(), content.BlockInvalidName},
		{"blank name", "rod_x", "  ", miFields(), content.BlockInvalidName},
		{"untrimmed name", "rod_x", " Rod", miFields(), content.BlockInvalidName},
		{"trailing space", "rod_x", "Rod ", miFields(), content.BlockInvalidName},
		{"no category", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Category = 0 }), content.BlockInvalidMember},
		{"blank note", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.CategoryNote = " " }), content.BlockBlankText},
		{"note with a control character", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.CategoryNote = "a\tb" }), content.BlockInvalidCharacter},
		{"nil rarity", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Rarity = nil }), content.BlockMissing},
		{"zero single rarity", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Rarity = content.SingleRarity{} }), content.BlockInvalidMember},
		{"zero by-variant rarity", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Rarity = byVariant }), content.BlockInvalidMember},
		{"zero required-by", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Attunement = content.AttunementRequiredBy{} }), content.BlockInvalidMember},
		{"no description", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Description = nil }), content.BlockEmpty},
		{"zero block", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Description = []content.TextBlock{content.TextParagraph{}} }), content.BlockInvalidMember},
		{"nil block", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Description = []content.TextBlock{nil} }), content.BlockInvalidMember},
		{"nil block pointer", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Description = []content.TextBlock{(*content.TextParagraph)(nil)} }), content.BlockInvalidMember},
		{"valid block pointer", "rod_x", "Rod", with(func(f *content.MagicItemFields) {
			f.Description = []content.TextBlock{ptr(content.MustTextParagraph("t"))}
		}), content.BlockInvalidMember},
		{"nil rarity pointer", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Rarity = (*content.SingleRarity)(nil) }), content.BlockInvalidMember},
		{"nil attunement pointer", "rod_x", "Rod", with(func(f *content.MagicItemFields) { f.Attunement = (*content.NoAttunement)(nil) }), content.BlockInvalidMember},
	}
	for _, c := range cases {
		_, err := content.NewMagicItemEntry(c.id, c.item, content.SourceCitation{}, c.f)
		if err == nil {
			t.Errorf("%s: accepted", c.name)
			continue
		}
		wantBlockError(t, err, "MagicItemEntry", c.want)
	}
	for _, id := range []content.MagicItemID{"", "Rod", "1rod", "a b"} {
		_, err := content.NewMagicItemEntry(id, "Rod", content.SourceCitation{}, miFields())
		var ie *content.IDError
		if !errors.As(err, &ie) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("id %q: error = %v, want *IDError [FR-002]", id, err)
		}
	}
	if !mustPanicsWith(func() { content.MustMagicItemEntry("", "Rod", content.SourceCitation{}, miFields()) }) {
		t.Error("MustMagicItemEntry did not panic")
	}
}

// Requirement: miSpec FR-008, FR-009 - the entry copies its description on
// the way in and out, and Equal is true iff every field is equal.
// Path: unhappy (immutability).
func TestMagicItemEntryImmutableAndEqual(t *testing.T) {
	t.Parallel()
	f := miFields()
	e := miEntry(t, f)
	f.Description[0] = content.MustTextParagraph("Changed.")
	if got := e.Description(); !textBlocksEqual(got, []content.TextBlock{content.MustTextParagraph("Text.")}) {
		t.Errorf("the constructor kept the caller's slice: %v [FR-008]", got)
	}
	d := e.Description()
	d[0] = content.MustTextParagraph("Other.")
	if e.Description()[0] != content.TextBlock(content.MustTextParagraph("Text.")) {
		t.Error("Description returned the internal slice [FR-008]")
	}
	if !e.Equal(miEntry(t, miFields())) {
		t.Error("equal entries are unequal [FR-009]")
	}
	table := content.MustTextTable([]string{"A"}, [][]string{{"1"}})
	other := func(mod func(*content.MagicItemFields)) content.MagicItemFields {
		x := miFields()
		mod(&x)
		return x
	}
	base := other(func(f *content.MagicItemFields) {
		f.Description = []content.TextBlock{content.MustTextParagraph("Text."), table}
		f.Rarity = content.SingleRarity{}
		r, _ := content.NewSingleRarity(content.RarityRare)
		f.Rarity = r
	})
	be := miEntry(t, base)
	r2, _ := content.NewSingleRarity(content.RarityUncommon)
	variants := map[string]content.MagicItemFields{
		"category":    other(func(f *content.MagicItemFields) { *f = base; f.Category = content.MagicItemWand }),
		"note":        other(func(f *content.MagicItemFields) { *f = base; f.CategoryNote = "n" }),
		"rarity":      other(func(f *content.MagicItemFields) { *f = base; f.Rarity = r2 }),
		"rarity kind": other(func(f *content.MagicItemFields) { *f = base; f.Rarity = content.RarityVaries{} }),
		"attunement":  other(func(f *content.MagicItemFields) { *f = base; f.Attunement = content.AttunementRequired{} }),
		"description": other(func(f *content.MagicItemFields) { *f = base; f.Description = base.Description[:1] }),
		"block kind": other(func(f *content.MagicItemFields) {
			*f = base
			f.Description = []content.TextBlock{content.MustTextParagraph("Text."), content.MustTextBulletList([]string{"1"})}
		}),
		"table cell": other(func(f *content.MagicItemFields) {
			*f = base
			f.Description = []content.TextBlock{content.MustTextParagraph("Text."), content.MustTextTable([]string{"A"}, [][]string{{"2"}})}
		}),
	}
	// Each pair is compared both ways, so an ordering mutation of == cannot hide.
	unequal := func(label string, a, b content.MagicItemEntry) {
		t.Helper()
		if a.Equal(b) || b.Equal(a) {
			t.Errorf("entries differing in %s are equal [FR-009]", label)
		}
	}
	for name, v := range variants {
		unequal(name, be, miEntry(t, v))
	}
	unequal("id", be, content.MustMagicItemEntry("rod_y", "Rod X", content.SourceCitation{}, base))
	unequal("name", be, content.MustMagicItemEntry("rod_x", "Rod Y", content.SourceCitation{}, base))
	unequal("source", be, content.MustMagicItemEntry("rod_x", "Rod X", content.MustSourceCitation("SRD 5.2.1", "S", 5), base))
	blocks := func(bs ...content.TextBlock) content.MagicItemFields {
		return other(func(f *content.MagicItemFields) { f.Description = bs })
	}
	para := content.MustTextParagraph("Text.")
	list := func(items ...string) content.TextBlock { return content.MustTextBulletList(items) }
	stat := func(lines ...string) content.TextBlock { return content.MustTextEmbeddedStatBlock(lines) }
	titled := func(title string) content.TextBlock {
		return content.MustTitledTextTable(title, []string{"A"}, [][]string{{"1"}})
	}
	entries := map[string]content.MagicItemEntry{}
	for name, f := range map[string]content.MagicItemFields{
		"paragraph": blocks(para), "entry": blocks(content.MustTextEntry("N", "Text.")), "list a": blocks(list("a")), "list b": blocks(list("b")),
		"stat a": blocks(stat("a")), "stat b": blocks(stat("b")), "table": blocks(table), "titled t": blocks(titled("t")), "titled u": blocks(titled("u")),
	} {
		entries[name] = miEntry(t, f)
	}
	for a, ea := range entries {
		for b, eb := range entries {
			if (a == b) != ea.Equal(eb) {
				t.Errorf("blocks %q and %q: Equal = %v [FR-009]", a, b, ea.Equal(eb))
			}
		}
	}
	byA, _ := content.NewRarityByVariant(optionsOf(t, "+1", "+2"))
	byB, _ := content.NewRarityByVariant(optionsOf(t, "+1", "+3"))
	ea := miEntry(t, other(func(f *content.MagicItemFields) { f.Rarity = byA }))
	if ea.Equal(miEntry(t, other(func(f *content.MagicItemFields) { f.Rarity = byB }))) || !ea.Equal(miEntry(t, other(func(f *content.MagicItemFields) { f.Rarity = byA }))) ||
		ea.Equal(miEntry(t, miFields())) {
		t.Error("by-variant rarities compare wrongly [FR-009]")
	}
}

func optionsOf(t *testing.T, variants ...string) []content.RarityOption {
	t.Helper()
	var out []content.RarityOption
	for _, v := range variants {
		o, err := content.NewRarityOption(content.RarityRare, v)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, o)
	}
	return out
}

func textBlocksEqual(a, b []content.TextBlock) bool { return slices.EqualFunc(a, b, textBlockEq) }

func textBlockEq(a, b content.TextBlock) bool {
	return content.MustMagicItemEntry("x", "X", content.SourceCitation{}, content.MagicItemFields{
		Category: content.MagicItemRod, Rarity: content.RarityVaries{}, Description: []content.TextBlock{a},
	}).Equal(content.MustMagicItemEntry("x", "X", content.SourceCitation{}, content.MagicItemFields{
		Category: content.MagicItemRod, Rarity: content.RarityVaries{}, Description: []content.TextBlock{b},
	}))
}

// Requirement: miSpec FR-009 - an entry with a fixed rarity header is not
// equal to one with a per-variant header, whichever side holds which.
// Path: unhappy (boundary).
func TestMagicItemEntryEqualRarityVariesVersusByVariant(t *testing.T) {
	t.Parallel()
	var opts []content.RarityOption
	for _, v := range []string{"+1", "+2"} {
		o, err := content.NewRarityOption(content.RarityRare, v)
		if err != nil {
			t.Fatal(err)
		}
		opts = append(opts, o)
	}
	by, err := content.NewRarityByVariant(opts)
	if err != nil {
		t.Fatal(err)
	}
	varies := miEntry(t, miFields())
	f := miFields()
	f.Rarity = by
	byEntry := miEntry(t, f)
	if varies.Equal(byEntry) || byEntry.Equal(varies) {
		t.Error("RarityVaries entry equals a RarityByVariant entry [FR-009]")
	}
}
