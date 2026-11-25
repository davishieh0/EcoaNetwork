/**
 * The platform's chrome, around each app page.
 */

// --- IMPORTS ---
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { Link, NavLink, Outlet } from 'react-router'

import { Wordmark } from '@/components/ecoa/Wordmark'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useSession } from '@/logic/session/session'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const APP_LINKS = [
  { label: 'Explore', to: ROUTES.explore },
  { label: 'Write a review', to: ROUTES.write },
  { label: 'Moderation', to: ROUTES.moderation },
] as const

// --- CODE ---
/**
 * Shows who is signed in, or the way to sign in.
 *
 * @returns The identity slot of the header.
 */
function Identity() {
  const session = useSession()
  const reduceMotion = useReducedMotion()

  // nothing until the session is known, so the slot never flickers
  const content =
    session.status === 'signed-in' ? (
      <Link
        to={session.account.handle ? ROUTES.account : ROUTES.welcome}
        className={cn(
          'font-mono text-mono-m text-secondary transition-colors',
          'duration-interface hover:text-primary',
        )}
      >
        {session.account.handle ? `@${session.account.handle}` : 'Finish setup'}
      </Link>
    ) : session.status === 'signed-out' ? (
      <Button as={Link} to={ROUTES.login} variant="ghost" size="sm">
        Sign in
      </Button>
    ) : null

  return (
    <AnimatePresence mode="wait" initial={false}>
      {content ? (
        <motion.div
          key={session.status}
          initial={reduceMotion ? false : { opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={reduceMotion ? undefined : { opacity: 0 }}
          transition={{ duration: reduceMotion ? 0 : 0.18 }}
        >
          {content}
        </motion.div>
      ) : null}
    </AnimatePresence>
  )
}

/**
 * Wraps every app page in the shared chrome.
 *
 * @returns The chrome, with the matched route rendered inside it.
 */
export function AppLayout() {
  const session = useSession()

  // the site is for visitors: someone signed in is home on the dashboard
  const home = session.status === 'signed-in' ? ROUTES.dashboard : ROUTES.home

  return (
    <>
      <header
        // solid, as the site's nav: nothing shows through it
        className="fixed inset-x-0 top-0 z-60 bg-canvas"
      >
        <Container
          className={cn(
            'grid min-h-[var(--header-h)] grid-cols-[1fr_auto] items-center',
            'gap-x-4 gap-y-2 py-3 sm:grid-cols-[1fr_auto_1fr]',
          )}
        >
          <Link
            to={home}
            className="col-start-1 row-start-1 flex items-center gap-3"
            aria-label="Ecoa — home"
          >
            <Wordmark />
          </Link>

          <nav
            aria-label="App navigation"
            className={cn(
              'col-span-2 row-start-2 flex flex-wrap gap-x-5 gap-y-2',
              'sm:col-span-1 sm:col-start-2 sm:row-start-1',
            )}
          >
            {APP_LINKS.map(({ label, to }) => (
              <NavLink
                key={to}
                to={to}
                className={cn(
                  'border-b-2 border-transparent py-2 font-mono text-mono-m',
                  'text-secondary hover:text-primary',
                  'aria-[current=page]:border-signal',
                  'aria-[current=page]:text-primary',
                )}
              >
                {label}
              </NavLink>
            ))}
          </nav>

          <div
            className={cn(
              'col-start-2 row-start-1 justify-self-end sm:col-start-3',
            )}
          >
            <Identity />
          </div>
        </Container>
      </header>

      <main id="main">
        <Outlet />
      </main>
    </>
  )
}
