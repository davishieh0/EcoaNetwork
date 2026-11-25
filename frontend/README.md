# ecoa-frontend

The Ecoa web client.

## Getting started

```bash
npm install
npm run dev
```

| Script | What it does |
|---|---|
| `npm run dev` | Dev server with hot reload, on port 5173 |
| `npm run build` | Type-checks with `tsc -b`, then builds to `dist/` |
| `npm run preview` | Serves `dist/` as a static host would |
| `npm run lint` | ESLint over the whole project |
| `npm run format` | Prettier over sources and configs |

`npm run dev` does **not** type-check — only `build` does. Keep an eye on the
editor, or run `build` before pushing.

The dev server sends `/api` to a gateway on `localhost:8000`. The whole network
— three nodes, the indexer, the gateway and this site behind nginx — runs from
[infra/local](../infra/local/README.md).

## Deploying

The `Dockerfile` builds the site and serves it with nginx, which also proxies
`/api` to the gateway named by `BACKEND_ORIGIN` (`nginx.conf.template`). One
image serves any gateway: pointing the client at another one is a setting, not
a code change. The network the client signs for is `ecoa-dev` unless
`VITE_ECOA_NETWORK` is set when building, which the image does not do yet.

## Stack

React 19 · TypeScript 6 · Vite 8 · Tailwind CSS 4 · React Router 8 ·
Motion · Radix (Dialog, Accordion) · ALTCHA (the human check) ·
i18next (the interface's languages)

## Where things are

- [docs/structure.md](docs/structure.md) — what each folder does, and where a
  new file goes
- [docs/tokens.md](docs/tokens.md) — fonts, colour, spacing, motion

Two entries worth knowing before anything else: `src/routes.ts` is the single
map of every destination the client links to, and `src/styles/tokens.css`
declares every visual value in the product.

## Conventions

- **Three sections per file**, in order: `// --- IMPORTS ---`,
  `// --- GLOBALS ---` (constants and types), `// --- CODE ---` (functions).
- **A docstring on the file and on every function**, one line plus `@param`
  and `@returns`.
- **Comments are one line.** Longer only with a `NOTE:` prefix, and only where
  removing the note would invite a silent bug.
- **80 columns.** Prettier handles the code; ESLint's `max-len` catches the
  strings and comments it cannot break.
- **Types live beside the code**: `Button.tsx` and `Button.t.ts`.

## Languages

The interface is written in English, which is the default; a reader can
switch to pt-BR (the EN · PT control in both headers, kept in
`localStorage` as `ecoa.language`). Setup lives in `src/i18n/`.

- **The English text is the key.** Wrap it where it renders:
  `t('Sign in')`, with `const { t } = useTranslation()`, or `i18n.t()`
  outside a component. Values go in as `{{name}}`:
  `t('since {{date}}', { date })`.
- **Add the Portuguese** to `src/i18n/locales/pt-BR/site.json` (the
  landing) or `app.json` (the platform). A missing key shows the English.
- `Field`, `CodeField`, `Select`, `FormAlert`, `Tooltip`, `Status` and
  `AuthShell` translate the strings they are given, so a label or error can
  be passed as plain English. They go through `known()`, which translates
  only what the dictionary holds: text already worded, or carrying what a
  person typed, passes through untouched.
