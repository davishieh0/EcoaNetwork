/**
 * The catch-all route.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { ROUTES } from '@/routes'

// --- CODE ---
/**
 * Builds the 404 page for unknown routes.
 *
 * @returns The 404 page.
 */
export function NotFound() {
  const { t } = useTranslation()

  return (
    <section className="flex min-h-svh items-center">
      <Container className="flex flex-col items-start gap-6 py-32">
        <p className="meta text-muted">{t('Error / 404')}</p>
        <h1
          className={cn(
            'font-display text-display-m font-semibold tracking-display',
            'text-balance',
          )}
        >
          {t('This page does not exist.')}
        </h1>
        <Button as={Link} to={ROUTES.home} size="lg">
          {t('Back to the start')}
        </Button>
      </Container>
    </section>
  )
}
