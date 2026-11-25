/**
 * The marketing site's chrome, around each site page.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { Outlet } from 'react-router'
import { useTranslation } from 'react-i18next'

import type { SiteLayoutProps } from '@/site/layout/SiteLayout.t'
import { SiteFooter } from '@/site/layout/SiteFooter'
import { SiteNav } from '@/site/layout/SiteNav'

// --- CODE ---
/**
 * Wraps every site page in the shared chrome.
 *
 * @param props - Whether the top bar is shown.
 * @returns The chrome, with the matched route rendered inside it.
 */
export function SiteLayout({ nav = true }: SiteLayoutProps) {
  const { t } = useTranslation()

  return (
    <>
      {nav ? (
        <>
          <a
            href="#main"
            className={cn(
              'meta fixed top-2 left-2 z-100 -translate-y-[140%] border',
              'border-signal bg-signal px-3 py-2 text-on-signal',
              'focus:translate-y-0',
            )}
          >
            {t('Skip to content')}
          </a>

          <SiteNav />
        </>
      ) : null}

      <main id="main">
        <Outlet />
      </main>

      <SiteFooter />
    </>
  )
}
