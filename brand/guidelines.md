# Ecoa — Logo Guidelines

The name is always written **Ecoa**. Never "EcoaNetwork" or "Ecoa Network".
Tagline: *Every experience leaves an echo.*

## 1. The logo

**Idea:** a half-circle, then the same shape again at half the size, and again. One form, repeating and fading: an echo.

| Version | Use it for | Files |
|---|---|---|
| Horizontal | Default. Site header, README, documents | `logo/horizontal/` |
| Stacked | Square or narrow spaces, splash screens | `logo/stacked/` |
| Symbol | When the name is already nearby; avatars | `logo/symbol/` |
| Symbol, small cut | Anything below 24 px | `logo/symbol/ecoa-symbol-small-*.svg` |
| Wordmark | When the symbol is already shown separately | `logo/wordmark/` |
| App icon | App tiles, social avatars | `logo/app-icon/` |

SVG files are the masters. PNG exports are in `logo/png/`. Browser icons are in `web/`.

### Construction

- The symbol is three half-discs standing on one baseline, flat side to the left.
- Each radius is half the one before: 96, 48 and 24 units on a 256-unit canvas.
- The gap between pieces is 16 units, one sixth of the large radius.
- In the horizontal lockup the wordmark sits on the same baseline as the symbol.
- The small cut drops the third piece and widens the gap so every edge lands on a pixel at 16 px.

## 2. Clear space

Keep a clear zone of **1 × U** on all sides, where **U** is the width of the middle half-disc
(one quarter of the symbol's height). The zone scales with the logo; never use a fixed distance.

## 3. Minimum size

| Version | Screen | Print |
|---|---|---|
| Horizontal | 80 px wide (24 px tall) | 24 mm wide |
| Stacked | 40 px wide | 14 mm wide |
| Symbol (three pieces) | 24 px | 8 mm |
| Symbol, small cut (two pieces) | 16 px | 4 mm |

Below 24 px always switch to the small cut. Print sizes are starting points; confirm with a proof.

## 4. Colour

| Name | HEX | RGB | Role |
|---|---|---|---|
| Echo blue | `#6d9bff` | 109, 155, 255 | Symbol on dark; tile fill on paper |
| Dark | `#0e0f10` | 14, 15, 16 | Default surface |
| Paper | `#fafaf8` | 250, 250, 248 | Light surface; wordmark on dark |
| Ink | `#11110f` | 17, 17, 15 | Logo on paper; symbol on a blue tile |

CMYK and Pantone values are not defined yet. Have a printer match Echo blue from a calibrated proof before any print run.

### Approved pairings

| Logo | Background | Contrast | File suffix |
|---|---|---|---|
| Blue symbol, paper wordmark | Dark | 7.1 : 1 and 18.4 : 1 | `-color-on-dark` |
| All ink | Paper | 18.1 : 1 | `-ink` |
| All paper | Dark, or a calm dark photo | 18.4 : 1 | `-paper` |
| Ink symbol on a blue tile, ink wordmark | Paper | 7.0 : 1 on the tile | `-tile-on-paper` |
| All blue | Dark only | 7.1 : 1 | `-blue` |

Echo blue on paper is only 2.6 : 1. On paper, blue appears only as the tile fill with ink on top, never as the
symbol or the wordmark by themselves.

## 5. Typography

- Wordmark: Space Grotesk SemiBold, converted to outlines. Use the files; do not retype it.
- Display: Space Grotesk. Body: Instrument Sans. Code and data: IBM Plex Mono.
- All three are open source under the SIL Open Font License, which permits logo use.

## 6. In the product

The interface already uses ■ ● ◆ for artifact, actor and claim. The symbol is built from half-discs so it stays
distinct from them. Do not use a single half-disc as a UI glyph, and do not place the symbol inline next to ●.

## 7. Don'ts

- Don't stretch, squash, rotate or mirror the logo.
- Don't change the order, size or spacing of the three pieces.
- Don't recolour it outside the pairings above.
- Don't add shadows, gradients, outlines or effects.
- Don't use the blue symbol or blue wordmark directly on paper.
- Don't use the three-piece symbol below 24 px.
- Don't place the logo on a busy background without a tile.
- Don't recreate the wordmark by typing it.

## 8. Web icons

Copy the contents of `web/` to the site's public root and add `web/head-snippet.html` to the document head.

| File | Purpose |
|---|---|
| `favicon.svg` | Small cut; ink in light mode, blue in dark mode |
| `favicon.ico`, `favicon-16/32/48.png` | Blue small cut on a dark tile, pixel-fitted |
| `apple-touch-icon.png` | 180 px, ink on blue |
| `icon-192.png`, `icon-512.png`, `maskable-512.png` | Install icons |
| `site.webmanifest` | Icon manifest |

## 9. Banners

| File | Purpose |
|---|---|
| `banner/ecoa-banner-dark.*`, `banner/ecoa-banner-light.*` | README header, 1600 × 480; the README switches between them by colour scheme |
| `banner/ecoa-social-preview.*` | Repository social preview, 1280 × 640; upload it in the repository settings |

## 10. Open items

- No trademark search has been run. Do one before public launch.
- Print colour values (CMYK, Pantone) are not defined.
- `exploration/` holds the earlier rounds for reference; nothing in it is approved.
