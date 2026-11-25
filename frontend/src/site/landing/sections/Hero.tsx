/**
 * Hero — the brand moment.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { CircleMark } from '@/components/ecoa/CircleMark'
import { HeroField } from '@/site/landing/motifs/HeroField'
import { Button } from '@/components/ui/Button'
import { ROUTES } from '@/routes'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'

// --- GLOBALS ---
/** Small artifacts floating in the field, in Ecoa vocabulary. */
const CHIPS = [
  { label: '◆ Domain verified', tone: 'valid', top: '22%', left: '7%' },
  { label: '◆ Human claim', tone: 'neutral', top: '34%', left: '82%' },
  { label: '★★★★☆ 4.0', tone: 'neutral', top: '62%', left: '5%' },
  { label: '■ Replaced · v2', tone: 'neutral', top: '71%', left: '78%' },
  { label: 'Observed by / 06 nodes', tone: 'signal', top: '14%', left: '62%' },
] as const

// --- CODE ---
/**
 * Draws the artifacts floating around the headline.
 *
 * @returns The chips, positioned around the headline.
 */
function Chips() {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-0 hidden lg:block"
    >
      {CHIPS.map((chip, index) => (
        <motion.span
          key={chip.label}
          style={{ top: chip.top, left: chip.left }}
          className={cn(
            'meta absolute border px-3 py-2 whitespace-nowrap',
            'border-strong bg-surface',
            chip.tone === 'valid' && 'text-valid',
            chip.tone === 'signal' && 'border-signal text-signal',
            chip.tone === 'neutral' && 'text-secondary',
          )}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={
            reduceMotion
              ? { duration: 0 }
              : {
                  duration: 0.32,
                  delay: 0.5 + index * 0.09,
                  ease: [0.2, 0.8, 0.2, 1],
                }
          }
        >
          {t(chip.label)}
        </motion.span>
      ))}
    </div>
  )
}

/**
 * Builds the hero section.
 *
 * @returns The hero section.
 */
export function Hero() {
  const { t } = useTranslation()
  const rise = useRise()

  return (
    <section
      id="top"
      data-hero=""
      className={cn(
        'relative isolate flex min-h-svh flex-col items-center justify-center',
        'overflow-hidden px-6',
      )}
    >
      <HeroField />
      <Chips />

      <div
        className={cn(
          'relative z-10 flex w-full max-w-5xl flex-col items-center gap-8',
          'py-32',
          'text-center',
        )}
      >
        <motion.h1
          {...rise(0.1)}
          className={cn(
            'font-display text-display-xl font-bold tracking-[-0.045em]',
            'uppercase',
          )}
        >
          {t('Every experience')}
          <br />
          {t('leaves an')}{' '}
          <span className="relative inline-block">
            {t('echo')}
            {/* Circles the word the product is named for. */}
            <CircleMark
              className="absolute -inset-x-6 -inset-y-5 -z-10 overflow-visible"
              delay={0.45}
              on="mount"
            />
          </span>
          .
        </motion.h1>

        <motion.p
          {...rise(0.22)}
          className="max-w-2xl text-body-l text-secondary text-pretty"
        >
          {t(
            'An open network of reviews. Signed by their authors, ' +
              'verifiable by anyone.',
          )}
        </motion.p>

        <motion.div
          {...rise(0.32)}
          className="flex flex-wrap items-center justify-center gap-3"
        >
          <Button as={Link} to={ROUTES.explore} size="lg">
            {t('Explore the network')}
          </Button>
          <Button as={Link} to={ROUTES.write} variant="secondary" size="lg">
            {t('Write a review')}
          </Button>
        </motion.div>
      </div>

      <motion.p
        {...rise(0.5)}
        className="meta absolute bottom-8 left-1/2 -translate-x-1/2 text-muted"
      >
        {t('No account needed to read')}
      </motion.p>
    </section>
  )
}
