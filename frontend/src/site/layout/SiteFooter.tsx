/**
 * The site's footer.
 */

// --- IMPORTS ---
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router'
import { cn } from '@/components/ui/cn'
import { Wordmark } from '@/components/ecoa/Wordmark'
import { Container } from '@/components/ui/Container'
import { ROUTES } from '@/routes'
import { HeroField } from '@/site/landing/motifs/HeroField'
import { FOOTER_LINKS } from '@/site/layout/nav'

// --- CODE ---
/**
 * Renders the site footer.
 *
 * @returns The footer.
 */
export function SiteFooter() {
  const { t } = useTranslation()

  return (
    <footer
      className={cn(
        'relative isolate overflow-hidden border-t border-structural bg-canvas',
        'py-16',
      )}
    >
      {/* The hero's field, at a whisper. */}
      <HeroField className="opacity-30" />

      <Container className="relative z-10 flex flex-col gap-12">
        <div
          className={cn(
            'grid grid-cols-1 gap-10 md:grid-cols-[1fr_auto] md:gap-20',
          )}
        >
          <div className="flex flex-col gap-4">
            <Link
              to={ROUTES.home}
              className="flex items-center gap-3"
              aria-label={t('Ecoa — home')}
            >
              <Wordmark />
            </Link>
            <p
              className={cn(
                'font-display text-h3 font-medium tracking-heading',
                'text-secondary text-balance',
              )}
            >
              {t('Every experience leaves an echo.')}
            </p>
          </div>

          <nav
            className="grid grid-cols-2 gap-10 sm:grid-cols-3"
            aria-label={t('Footer')}
          >
            {FOOTER_LINKS.map((group) => (
              <div key={group.heading} className="flex flex-col gap-3">
                <h2 className="meta text-muted">{t(group.heading)}</h2>
                <ul className="flex flex-col gap-2">
                  {group.links.map((link) => {
                    const cls = cn(
                      'text-body-s text-secondary transition-colors',
                      'duration-interface hover:text-primary',
                    )
                    const isExternal = link.href.startsWith('http')
                    const isRoute =
                      link.href.startsWith('/') && !link.href.includes('#')

                    return (
                      <li key={link.label}>
                        {isRoute ? (
                          <Link to={link.href} className={cls}>
                            {t(link.label)}
                          </Link>
                        ) : (
                          <a
                            href={link.href}
                            className={cls}
                            target={isExternal ? '_blank' : undefined}
                            rel={isExternal ? 'noreferrer' : undefined}
                          >
                            {t(link.label)}
                          </a>
                        )}
                      </li>
                    )
                  })}
                </ul>
              </div>
            ))}
          </nav>
        </div>

        <p className="meta border-t border-line pt-6 text-muted">
          {t('Ecoa / Open network / 2026')}
        </p>
      </Container>
    </footer>
  )
}
