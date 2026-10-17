# Source material

Which document this package takes game rules from. Licensing rules are in
"Licensing and source material" in [AGENTS.md](../../AGENTS.md); the notice
that ships with bundled content is [NOTICE.md](../../NOTICE.md).

## The one source

| Document | Version | Licence | Download |
|---|---|---|---|
| System Reference Document | 5.2.1 (released 2025-05-01) | CC-BY-4.0 | https://media.dndbeyond.com/compendium-images/srd/5.2/SRD_CC_v5.2.1.pdf |

The document's landing page is https://www.dndbeyond.com/srd. The licence
statement and the required attribution are on page 1 of the PDF ("Legal
Information"). The PDF is the only authority: if a rule is not in it, it is
not in scope.

## Structure of the document

From the PDF's own contents page:

| Section | Pages | Likely package |
|---|---|---|
| Playing the Game (abilities, D20 Tests, proficiency, actions, social interaction, exploration, combat, damage and healing) | 5-18 | `d20`, `character`, `combat` |
| Character Creation | 19-27 | `character` |
| Classes | 28-82 | `content`, `character` |
| Character Origins (backgrounds, species) | 83-86 | `content` |
| Feats | 87-88 | `content` |
| Equipment | 89-103 | `content` |
| Spells | 104-175 | `content`, `character` |
| Rules Glossary | 176-191 | all |
| Gameplay Toolbox | 192-203 | `toolbox` |
| Magic Items | 204-253 | `content` |
| Monsters, Monsters A-Z, Animals | 254-364 | `content`, `combat` |

## Not covered

- Every other fifth-edition book. Only what is in the SRD is licensed to us.
- The earlier SRD 5.1. It is a different rules edition, so the two are not
  mixed.
- Third-party and community material, and anything from memory or play.
- Wizards' trademarks and logos. The licence does not grant them.

## Using it

- The PDF is not stored in this repository. Download it from the link above.
- Cite by section and page. Test `Requirement:` comments and specs use the
  form `SRD 5.2.1, "Damage Rolls", p. 16`.
- To check completeness, walk the contents table above section by section and
  record each rule with an ID in `doc/specs/`.
