/**
 * What the site's chrome links to.
 */

// --- IMPORTS ---
import { ROUTES } from '@/routes'

// --- GLOBALS ---
/** What the marketing site's chrome links to. */
/** In document order, and complete. */
export const NAV_LINKS = [
  { label: 'Why Ecoa', href: '#why-ecoa' },
  { label: 'How it works', href: '#how-it-works' },
  { label: 'A real review', href: '#a-real-review' },
  { label: 'Trust', href: '#trust' },
  { label: 'History', href: '#history' },
  { label: 'Network', href: '#network' },
  { label: 'Human first', href: '#human-first' },
  { label: 'FAQ', href: '#faq' },
] as const

const REPO = 'https://github.com/eduardokairalla1/ecoa'

export const FOOTER_LINKS = [
  {
    heading: 'Protocol',
    links: [
      { label: 'Specification', href: `${REPO}/blob/main/docs/protocol.md` },
      { label: 'Event types', href: `${REPO}/tree/main/protocol/schemas` },
      { label: 'Documentation', href: `${REPO}/tree/main/docs` },
      { label: 'GitHub', href: REPO },
    ],
  },
  {
    heading: 'Network',
    links: [
      { label: 'Explore', href: ROUTES.explore },
      { label: 'Identity', href: ROUTES.identity },
      { label: 'Moderation', href: ROUTES.moderation },
      { label: 'Run a node', href: `${REPO}/blob/main/node/README.md` },
    ],
  },
  {
    heading: 'About',
    links: [
      { label: 'How it works', href: '/#how-it-works' },
      { label: 'Trust model', href: '/#trust' },
      { label: 'FAQ', href: '/#faq' },
    ],
  },
] as const
