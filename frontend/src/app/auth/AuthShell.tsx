/**
 * The frame both identity screens share.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { AuthShellProps } from '@/app/auth/AuthShell.t'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'
import { known } from '@/i18n/i18n'

// --- CODE ---
/**
 * Renders a narrow column with a heading and the form under it.
 *
 * @param props - Eyebrow, headline, lede, the form and the link out.
 * @returns The framed screen.
 */
export function AuthShell({
  eyebrow,
  title,
  lede,
  children,
  footer,
}: AuthShellProps) {
  const { t } = useTranslation()
  const rise = useRise()

  return (
    <section className="flex min-h-svh items-center">
      <Container className="flex justify-center py-32">
        <div className="flex w-full max-w-md flex-col gap-8">
          <header className="flex flex-col gap-3">
            <motion.p {...rise()} className="meta text-muted">
              {known(t, eyebrow)}
            </motion.p>

            <motion.h1
              {...rise(0.06)}
              className={cn(
                'font-display text-h1 font-semibold tracking-display',
                'text-balance',
              )}
            >
              {known(t, title)}
            </motion.h1>

            <motion.p
              {...rise(0.12)}
              className="text-body-m text-secondary text-pretty"
            >
              {known(t, lede)}
            </motion.p>
          </header>

          <motion.div {...rise(0.18)}>{children}</motion.div>

          <motion.p {...rise(0.26)} className="text-body-s text-secondary">
            {typeof footer === 'string' ? known(t, footer) : footer}
          </motion.p>
        </div>
      </Container>
    </section>
  )
}
