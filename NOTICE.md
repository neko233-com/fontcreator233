# Third-party font notice

fontcreator233 embeds two modified static base fonts derived from Google's Noto
family. Both are licensed under the SIL Open Font License 1.1. The full license
text is in [`assets/base/OFL.txt`](assets/base/OFL.txt).

## fontcreator233 Base SC

- Derived from: Noto Sans SC, version 2.004-H2, variable `wght` font
- Source: <https://github.com/google/fonts/tree/main/ofl/notosanssc>
- Modification: instantiated at `wght=400`, renamed per the OFL, stripped
  hinting/layout tables, fixed timestamps, `post` format 3.0
- SHA-256: `49d4bf9374ea6d2c480a0db25c54c128727743f04803bf383600b09e2881d7e2`

## fontcreator233 Base KR

- Derived from: Noto Sans KR, version 2.004-H2, variable `wght` font
- Source: <https://github.com/google/fonts/tree/main/ofl/notosanskr>
- Modification: instantiated at `wght=400`, subset to Hangul/Jamo/compatibility
  and fullwidth Hangul, renamed per the OFL, stripped hinting/layout tables,
  fixed timestamps, `post` format 3.0
- SHA-256: `ae1212653d0b95a09341f33ceb488038a3afb68f8097a98679102c316c50a590`

## Latin starter outline packs

The `assets/latin/` packs are static Regular ASCII outlines extracted with
`fontcreator import font --extract-glyphs --unicodes 0020-007E`. They are
starter shapes for the `hei`, `song`, `puhuiti`, and `mono` templates, not
variable fonts. Generated family names never reuse the source family names.

| Pack | Source | Version | Copyright | License |
|---|---|---|---|---|
| `hei` | Noto Sans Regular | 2.015 | Adobe, with Reserved Font Name "Source" | OFL-1.1 |
| `song` | Noto Serif Regular | 2.015 | Adobe, with Reserved Font Name "Source" | OFL-1.1 |
| `puhui` | Poppins Regular | 4.004 | Indian Type Foundry / Jonny Pinhorn | OFL-1.1 |
| `mono` | JetBrains Mono Regular | 2.305 | JetBrains | OFL-1.1 |

Sources:
<https://github.com/notofonts/notofonts.github.io>,
<https://github.com/google/fonts/tree/main/ofl/poppins>,
<https://github.com/JetBrains/JetBrainsMono>.

The full license text is in [`assets/latin/OFL.txt`](assets/latin/OFL.txt).
Alibaba PuHuiTi is **not** bundled: its license forbids modifying or
redistributing the font files. `puhuiti` is a Poppins-derived geometric
starter in the same style family, not the Alibaba font.

## License and reserved names

The OFL copyright notice reserves the font name "Source". fontcreator233 does
not use "Source", "Noto", or "Noto Sans" as a family name for its base fonts or
generated fonts. The same rule applies to "Poppins" and "JetBrains Mono" for
the Latin starter packs. Generated fonts that include outlines derived from
these packs are distributed under OFL-1.1; the generator enforces this.

The original copyright notices are preserved in the embedded fonts' `name`
table and in the OFL text.
