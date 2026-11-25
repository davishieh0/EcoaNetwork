/**
 * What a guarded screen shows while it learns who is signed in.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { Container } from '@/components/ui/Container'

// --- CODE ---
/**
 * Draws a slim bar that sweeps across while the session loads.
 *
 * @returns The placeholder.
 */
export function SessionPending() {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  return (
    <section className="flex min-h-svh items-center" aria-busy="true">
      <Container className="flex justify-center">
        <div className="relative h-px w-full max-w-md overflow-hidden bg-quiet">
          <motion.span
            className="absolute inset-y-0 w-1/3 bg-signal"
            initial={{ x: '-100%' }}
            animate={reduceMotion ? { x: '100%' } : { x: ['-100%', '300%'] }}
            transition={
              reduceMotion
                ? { duration: 0 }
                : { duration: 1.1, repeat: Infinity, ease: 'easeInOut' }
            }
          />
        </div>
        <span className="sr-only">{t('Checking your session')}</span>
      </Container>
    </section>
  )
}
