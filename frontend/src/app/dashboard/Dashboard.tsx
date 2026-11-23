/**
 * Where a signed-in account lands.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { Link } from 'react-router'

import type { Shortcut } from '@/app/dashboard/Dashboard.t'
import { useDashboardData } from '@/app/dashboard/useDashboardData'
import { Badge } from '@/components/ui/Badge'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const SHORTCUTS: Shortcut[] = [
  {
    title: 'Write a review',
    body: 'Sign an experience and publish it to the network.',
    to: ROUTES.write,
  },
  {
    title: 'Explore',
    body: 'Browse example reviews of businesses, products, and services.',
    to: ROUTES.explore,
  },
  {
    title: 'Account',
    body: 'Your handle and email, signing out, deleting the account.',
    to: ROUTES.account,
  },
]

const TILE = cn(
  'flex h-full flex-col gap-3 border p-6',
  'transition-colors duration-interface ease-standard',
)

// --- CODE ---
/**
 * Draws one shortcut: a link when its screen exists, a quiet tile until then.
 *
 * @param props - The shortcut.
 * @returns The tile.
 */
function Tile({ title, body, to }: Shortcut) {
  const content = (
    <>
      <span className="flex items-baseline justify-between gap-4">
        <span className="font-display text-h4 font-semibold tracking-heading">
          {title}
        </span>
        {to ? (
          <span
            aria-hidden="true"
            className={cn(
              'font-mono text-mono-m text-muted transition-transform',
              'duration-interface ease-standard group-hover:translate-x-1',
              'group-hover:text-primary',
            )}
          >
            →
          </span>
        ) : null}
      </span>
      <span className="text-body-s text-secondary text-pretty">{body}</span>
      {to ? null : (
        <span className="mt-auto pt-2">
          <Badge tone="neutral" className="whitespace-nowrap">
            Coming soon
          </Badge>
        </span>
      )}
    </>
  )

  if (!to) {
    return <div className={cn(TILE, 'border-quiet text-muted')}>{content}</div>
  }

  return (
    <Link
      to={to}
      className={cn(
        TILE,
        'group border-line hover:border-structural hover:bg-surface',
      )}
    >
      {content}
    </Link>
  )
}

/**
 * Builds the dashboard.
 *
 * @returns The dashboard, or nothing until the account is known.
 */
export function Dashboard() {
  const { account } = useDashboardData()
  const rise = useRise()

  // RequireAuth only renders this with a complete account
  if (!account?.handle) return null

  return (
    <section className="flex min-h-svh items-center">
      <Container className="flex justify-center py-32">
        <div className="flex w-full max-w-3xl flex-col gap-10">
          <header className="flex flex-col gap-3">
            <motion.p {...rise()} className="meta text-muted">
              Home
            </motion.p>

            <motion.h1
              {...rise(0.06)}
              className={cn(
                'font-display text-h1 font-semibold tracking-display',
                'text-balance break-words',
              )}
            >
              Welcome, @{account.handle}.
            </motion.h1>

            <motion.p
              {...rise(0.12)}
              className="text-body-m text-secondary text-pretty"
            >
              Your reviews and replies will live here. For now, this is where
              signing in lands.
            </motion.p>
          </header>

          <ul className="grid gap-4 sm:grid-cols-3">
            {SHORTCUTS.map((shortcut, index) => (
              <motion.li key={shortcut.title} {...rise(0.18 + index * 0.06)}>
                <Tile {...shortcut} />
              </motion.li>
            ))}
          </ul>
        </div>
      </Container>
    </section>
  )
}
