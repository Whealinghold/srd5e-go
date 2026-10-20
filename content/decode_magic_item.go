package content

// The wire names of the "kind" of a magic item rarity (content-magic-items
// FR-017).
const (
	rarityKindSingle    = "single"
	rarityKindVaries    = "varies"
	rarityKindByVariant = "byVariant"
)

// decodeMagicItem is the row of the magicItems array (content-magic-items
// FR-037 to FR-042): id, name, source?, category, categoryNote?, rarity,
// attunement?, description. The entry holds no reference to another kind, so
// the row registers no site.
func decodeMagicItem(o *object) decodedEntry {
	d := o.d
	h := o.header()
	var f MagicItemFields
	if n, p, ok := o.field("category", true); ok {
		f.Category, _ = enumField(d, n, p, ParseMagicItemCategory)
	}
	if n, p, ok := o.field("categoryNote", false); ok {
		f.CategoryNote, _ = d.text(n, p)
	}
	if n, p, ok := o.field("rarity", true); ok {
		f.Rarity = decodeRarity(d, n, p)
	}
	if n, p, ok := o.field("attunement", false); ok {
		f.Attunement = decodeAttunement(d, n, p)
	}
	if n, p, ok := o.field("description", true); ok {
		f.Description = decodeTextBlocks(d, n, p)
	}
	res := decodedEntry{id: h.id, idOK: h.idOK}
	if e, err := NewMagicItemEntry(MagicItemID(h.id.String()), h.name, h.source, f); err == nil {
		res.entry = e
	}
	return res
}

// decodeRarity reads {kind, rarity?, options?} (FR-017). A missing, non-string
// or unknown kind is the only problem of the object. It returns nil when the
// rarity could not be built.
func decodeRarity(d *decoder, n node, p string) RaritySpec {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	kn, kp, present := o.field("kind", true)
	if !present {
		o.skipAll()
		return nil
	}
	kind, ok := d.str(kn, kp)
	if !ok {
		o.skipAll()
		return nil
	}
	switch kind {
	case rarityKindSingle:
		if n, rp, ok := o.field("rarity", true); ok {
			if r, ok := enumField(d, n, rp, ParseItemRarity); ok {
				v, _ := NewSingleRarity(r)
				return v
			}
		}
	case rarityKindVaries:
		return RarityVaries{}
	case rarityKindByVariant:
		if n, op, ok := o.field("options", true); ok {
			if v, err := NewRarityByVariant(decodeRarityOptions(d, n, op)); err == nil { // a bad option has reported its problem
				return v
			}
		}
	default:
		d.probs.add(kp, ProblemInvalidValue)
		o.skipAll()
	}
	return nil
}

// decodeRarityOptions reads the options of a byVariant rarity: at least two,
// each {rarity, variant}, a repeated variant duplicate at the later variant.
// It returns the options that decoded; any other has reported a problem.
func decodeRarityOptions(d *decoder, n node, p string) []RarityOption {
	var out []RarityOption
	seen := map[string]struct{}{}
	d.list(n, p, MinRarityOptions, noUpperBound, func(_ int, el node, ep string) {
		o, ok := d.object(el, ep)
		if !ok {
			return
		}
		var r ItemRarity
		var variant string
		repeated := false
		if n, rp, ok := o.field("rarity", true); ok {
			r, _ = enumField(d, n, rp, ParseItemRarity)
		}
		if n, vp, ok := o.field("variant", true); ok {
			if variant, ok = d.text(n, vp); ok {
				_, repeated = seen[variant]
				seen[variant] = struct{}{}
				if repeated {
					d.probs.add(vp, ProblemDuplicate)
				}
			}
		}
		o.finish()
		if opt, err := NewRarityOption(r, variant); err == nil && !repeated {
			out = append(out, opt)
		}
	})
	return out
}

// decodeAttunement reads {} or {by} (FR-019); a blank by is invalidValue. It
// returns nil when the Attunement could not be built.
func decodeAttunement(d *decoder, n node, p string) Attunement {
	o, ok := d.object(n, p)
	if !ok {
		return nil
	}
	defer o.finish()
	bn, bp, present := o.field("by", false)
	if !present {
		return AttunementRequired{}
	}
	by, ok := d.text(bn, bp)
	if !ok {
		return nil
	}
	v, _ := NewAttunementRequiredBy(by)
	return v
}
