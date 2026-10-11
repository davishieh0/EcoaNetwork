# Tokens

Every value the frontend uses, and where it comes from.

All of it is declared in `src/styles/tokens.css`. Always use the token,
never the raw value.

## Fonts

| Role | Family | Utility |
|---|---|---|
| Display | Space Grotesk | `font-display` |
| Body | Instrument Sans | `font-sans` |
| Mono | IBM Plex Mono | `font-mono` |

## Type scale

| Utility | Size | Line height |
|---|---|---|
| `text-display-xl` | clamp(2.5rem, 8vw, 6rem) | 0.96 |
| `text-display-l` | clamp(3rem, 7vw, 4.5rem) | 0.98 |
| `text-display-m` | clamp(2.25rem, 5vw, 3.5rem) | 1.02 |
| `text-h1` | clamp(2rem, 3.5vw, 2.5rem) | 1.1 |
| `text-h2` | 2rem | 1.12 |
| `text-h3` | 1.5rem | 1.25 |
| `text-h4` | 1.25rem | 1.3 |
| `text-body-l` | 1.125rem | 1.55 |
| `text-body-m` | 1rem | 1.5 |
| `text-body-s` | 0.875rem | 1.45 |
| `text-label-m` | 0.8125rem | 1.25 |
| `text-label-s` | 0.6875rem | 1.3 |
| `text-mono-m` | 0.8125rem | 1.5 |
| `text-mono-s` | 0.6875rem | 1.45 |

Tracking: `tracking-display` -0.03em · `tracking-heading` -0.02em ·
`tracking-metadata` 0.06em.

## Colour

Colour is **contextual**. Dark is the default surface; `data-surface="light"`
flips an entire subtree to paper. A component never knows which surface it is
on — it just uses the token. All 31 colour tokens are below.

### Surfaces

| Utility | Dark | Light |
|---|---|---|
| `canvas` | `#0e0f10` | `#fafaf8` |
| `surface` | `#141618` | `#ffffff` |
| `surface-raised` | `#1b1e21` | `#ffffff` |
| `surface-sunken` | `#0a0b0c` | `#f1f1ee` |
| `scrim` | ink 72% | ink 72% |

### Text

| Utility | Dark | Light |
|---|---|---|
| `primary` | `#f2f3f4` | `#111316` |
| `secondary` | `#a8adb3` | `#55595f` |
| `muted` | `#868c93` | `#676b72` |

### Borders

| Utility | Dark | Light |
|---|---|---|
| `quiet` | paper 10% | `#ebebe7` |
| `line` | paper 18% | `#dcdcd7` |
| `strong` | paper 34% | `#8b8b85` |
| `structural` | `#68707b` | `#111316` |

### Brand

The blue is the same on both surfaces, by decision. Only `signal-dim` differs,
because the hover has to keep ink readable on paper.

| Utility | Dark | Light |
|---|---|---|
| `signal` | `#6d9bff` | `#6d9bff` |
| `signal-bright` | `#8fb4ff` | `#8fb4ff` |
| `signal-dim` | `#3f6fd6` | `#4e82ea` |
| `signal-wash` | blue 16% | blue 12% |
| `on-signal` | `#11110f` | `#11110f` |
| `focus` | `#6d9bff` | `#6d9bff` |
| `focus-halo` | `#fafaf8` | `#11110f` |
| `hand` | `#6d9bff` | `#6d9bff` |
| `logo` | `#6d9bff` | `#111316` |

`focus-halo` is the second ring, in the opposite tone. The blue alone is 2.6:1
on paper, under the 3:1 a focus indicator has to clear; the pair always clears
it.

`logo` fills the logo's symbol. It goes to ink on paper for the same reason:
the blue is too faint there to carry a shape on its own.

### Semantics

Text and glyph only — a semantic colour never fills.

| Utility | Dark | Light |
|---|---|---|
| `valid` | `#4ed4a0` | `#147a5a` |
| `warning` | `#ffc845` | `#8a5300` |
| `danger` | `#ff7b72` | `#b42318` |
| `info` | `#5ccbee` | `#0b6e8c` |
| `info-wash` | cyan 12% | `#e6f4f9` |
| `on-danger` | `#0e0f10` | `#ffffff` |

Info is teal-cyan, not blue: the brand owns blue, and a link and an info badge
must not look the same.

### Accents

Deliberately identical on both surfaces. They are **fill only**, with ink on
top — the blue is 2.6:1 as text on paper and the green 1.8:1, but 6.9:1 and
10.0:1 as fills. A paper section carries the brand as blocks, never as letters.

| Utility | Both surfaces |
|---|---|
| `accent` | `#6d9bff` |
| `accent-valid` | `#4ed4a0` |
| `accent-wash` | blue 12% |
| `on-accent` | `#11110f` |

### Alternate palette

`data-palette="mono"` swaps the brand family for ink and paper — `signal`
becomes `#f4f3ee` on dark and `#111316` on light. `hand` stays blue in every
palette: the human mark is the one thing that always carries colour.

### Rules

- On dark, hierarchy comes from **borders**, never stacked fills — `#141618` on
  `#0e0f10` is 1.06:1.
- The blue is 2.6:1 on paper. There it is a **fill**, and ink sits on it (7.0:1).
- Semantic colours write and mark; they never fill.
- State is never colour alone — a glyph and a label go with it (`■` artifact,
  `●` actor, `◆` claim).

## Shape

`rounded-none` 0 · `rounded-micro` 2px · `rounded-small` 4px ·
`rounded-full` 999px. Nothing above 8px exists in this language, and nothing
casts a shadow.

## Spacing

4px base, on the standard Tailwind scale: `0 1 2 3 4 5 6 8 10 12 16 20 24 32`.

| Case | Range |
|---|---|
| Small control | 8–12 |
| Normal control | 12–16 |
| Card interior | 20–24 |
| Component gap | 24–32 |
| Section gap | 64–128 |

Half-steps (`gap-1.5`, `size-2.5`) are only for glyph geometry and optical
nudges, never for layout.

## Layout

`--container-content` 1280px · `--container-reading` 720px (`reading-measure`).
Use `<Container>` for the page grid; the gutter never drops below 16px.

## Motion

`ease-standard` `cubic-bezier(0.2, 0.8, 0.2, 1)` ·
`duration-interface` 180ms · `duration-surface` 320ms (surface crossings only).

Information must never depend on animation: every motion path has a
`prefers-reduced-motion` branch that renders the finished state.

## Custom utilities

| Class | What it does |
|---|---|
| `.meta` | The metadata voice: mono, small, uppercase, tracked. Never on prose. |
| `.grid-field` | The exposed grid. Brand sections only, never behind dense content. |
| `.reading-measure` | Caps a text block at 720px. |
