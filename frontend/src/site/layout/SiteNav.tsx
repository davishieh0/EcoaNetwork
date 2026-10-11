/**
 * The site's top bar, and the menu it collapses into.
 */

// --- IMPORTS ---
import { useEffect, useState } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { motion, useReducedMotion } from 'motion/react'
import { Button } from '@/components/ui/Button'
import { CircleMark } from '@/components/ecoa/CircleMark'
import { Wordmark } from '@/components/ecoa/Wordmark'
import { Container } from '@/components/ui/Container'
import { NAV_LINKS } from '@/site/layout/nav'
import { ROUTES } from '@/routes'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
const WORDMARK_SYMBOL = 'transition-colors duration-surface ease-standard'

// --- CODE ---
/**
 * Renders the menu the bar collapses into below `lg`.
 *
 * @param props - The surface the bar is currently showing.
 * @returns The menu trigger, and the panel it opens.
 */
function MobileMenu({ surface }: { surface: 'dark' | 'light' }) {
  const [open, setOpen] = useState(false)
  const reduceMotion = useReducedMotion()

  /**
   * Closes the menu, then scrolls to the anchor.
   *
   * @param href - The in-page anchor to go to.
   * @returns Nothing.
   */
  const follow = (href: string) => {
    setOpen(false)
    requestAnimationFrame(() => {
      document.querySelector(href)?.scrollIntoView({ block: 'start' })
      history.replaceState(null, '', href)
    })
  }

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="duration-surface lg:hidden"
        >
          Menu
        </Button>
      </Dialog.Trigger>

      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-70 bg-scrim" />

        {/* The panel keeps the bar's surface, so nothing recolours. */}
        <Dialog.Content
          asChild
          aria-describedby={undefined}
          data-surface={surface === 'light' ? 'light' : undefined}
        >
          <motion.div
            className={cn(
              'fixed inset-x-0 top-0 z-70 flex max-h-svh flex-col',
              'overflow-y-auto',
              'border-b border-structural bg-canvas',
            )}
            initial={reduceMotion ? false : { y: '-100%' }}
            animate={{ y: 0 }}
            transition={
              reduceMotion
                ? { duration: 0 }
                : { duration: 0.28, ease: [0.2, 0.8, 0.2, 1] }
            }
          >
            <Container
              className={cn(
                'flex h-[var(--header-h)] shrink-0 items-center',
                'justify-between',
                'gap-6',
              )}
            >
              <Dialog.Title className="sr-only">Menu</Dialog.Title>
              <Wordmark />
              <Dialog.Close asChild>
                <Button variant="ghost" size="sm">
                  Close
                </Button>
              </Dialog.Close>
            </Container>

            <Container as="nav" aria-label="Sections">
              <ul className="flex flex-col border-t border-line">
                {NAV_LINKS.map((link) => (
                  <li key={link.href}>
                    <a
                      href={link.href}
                      onClick={(event) => {
                        event.preventDefault()
                        follow(link.href)
                      }}
                      className={cn(
                        'meta block border-b border-line py-5 text-secondary',
                        'transition-colors duration-interface',
                        'hover:text-primary',
                      )}
                    >
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </Container>

            <Container className="flex flex-wrap gap-2 py-6">
              <Button as="a" href={ROUTES.explore} variant="secondary">
                Explore
              </Button>
              <Button as="a" href={ROUTES.login} variant="ghost">
                Login
              </Button>
            </Container>
          </motion.div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

/**
 * Renders the top bar and tracks the surface under it.
 *
 * @returns The top bar.
 */
export function SiteNav() {
  const [surface, setSurface] = useState<'dark' | 'light'>('dark')
  const [active, setActive] = useState('')

  useEffect(() => {
    let ticking = false

    /**
     * Syncs the tracked value with its source.
     *
     * @returns Nothing.
     */
    const update = () => {
      // The bar adopts whichever surface it currently covers.
      const headerHeight = parseFloat(
        getComputedStyle(document.documentElement).getPropertyValue(
          '--header-h',
        ),
      )
      const line = headerHeight / 2
      const blocks = document.querySelectorAll<HTMLElement>('section, footer')

      // Dark is the canvas, so it is the answer for any gap.
      let next: 'dark' | 'light' = 'dark'
      // The same block answers both questions, so this costs nothing.
      let current = ''

      for (const block of blocks) {
        const rect = block.getBoundingClientRect()
        if (rect.top <= line && rect.bottom > line) {
          next = block.dataset.surface === 'light' ? 'light' : 'dark'
          current = block.id
          break
        }
      }

      setSurface(next)
      setActive(current)
      ticking = false
    }

    /**
     * Schedules one surface check per frame.
     *
     * @returns Nothing.
     */
    const onScroll = () => {
      if (ticking) return
      ticking = true
      window.requestAnimationFrame(update)
    }

    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll, { passive: true })
    update()

    return () => {
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [])

  return (
    <header
      data-surface={surface === 'light' ? 'light' : undefined}
      className={cn(
        'fixed inset-x-0 top-0 z-60 bg-canvas',
        // NOTE: custom properties do not interpolate, but the properties
        // consuming them do — so a transition here fades the surface swap
        // instead of cutting it.
        'transition-colors duration-surface ease-standard',
      )}
    >
      {/* The bar is full-bleed; its contents stay on the content grid. */}
      <Container
        className={cn(
          'grid h-[var(--header-h)] grid-cols-[1fr_auto_1fr] items-center',
          'gap-6',
        )}
      >
        <a
          href="#top"
          className="flex items-center gap-3 justify-self-start"
          aria-label="Ecoa — home"
        >
          <Wordmark symbolClassName={WORDMARK_SYMBOL} />
        </a>

        <nav
          aria-label="Sections"
          // 16px, not the 24px the bar uses elsewhere.
          className="hidden items-center justify-center gap-4 lg:flex xl:gap-6"
        >
          {NAV_LINKS.map((link) => {
            const isActive = link.href === `#${active}`

            return (
              <a
                key={link.href}
                href={link.href}
                aria-current={isActive ? 'true' : undefined}
                className={cn(
                  // 320ms, matching the bar.
                  'meta relative py-1',
                  'transition-colors duration-surface ease-standard',
                  isActive ? 'text-primary' : 'text-secondary',
                  'hover:text-primary focus-visible:text-primary',
                )}
              >
                {link.label}
                {isActive ? (
                  // The same hand that underlines "someone else’s" below.
                  <CircleMark
                    variant="underline"
                    on="mount"
                    weight={2}
                    className={cn(
                      'absolute -bottom-1 left-0 h-2 w-full overflow-visible',
                    )}
                  />
                ) : null}
              </a>
            )
          })}
        </nav>

        <div className="flex items-center gap-2 justify-self-end">
          {/* One breakpoint, not three. */}
          <Button
            as="a"
            href={ROUTES.explore}
            variant="ghost"
            size="sm"
            className="hidden duration-surface lg:inline-flex"
          >
            Explore
          </Button>
          <Button
            as="a"
            href={ROUTES.login}
            variant="ghost"
            size="sm"
            className="hidden duration-surface lg:inline-flex"
          >
            Login
          </Button>
          {/* White label, by decision: 2.7:1 where ink is 7.0:1. */}
          <Button
            as="a"
            href={ROUTES.register}
            variant="primary"
            size="sm"
            className="text-white hover:text-white"
          >
            Register
          </Button>

          <MobileMenu surface={surface} />
        </div>
      </Container>
    </header>
  )
}
