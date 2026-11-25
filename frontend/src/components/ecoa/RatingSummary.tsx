/**
 * An index's summary of an entity's ratings: its own way of adding them up,
 * which another index may do differently.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'
import type { RatingSummaryProps } from '@/components/ecoa/RatingSummary.t'
import { Rating } from '@/components/ecoa/Rating'
import { cn } from '@/components/ui/cn'
import { decimal } from '@/i18n/format'

// --- CODE ---
/**
 * Renders the mean, the stars, and a bar for each rating, 5 first.
 *
 * @param props - The counts, the mean, and how many were left out.
 * @returns The summary.
 */
export function RatingSummary({
  count,
  mean,
  distribution,
  conflicted,
  className,
}: RatingSummaryProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()
  const most = Math.max(1, ...distribution)

  return (
    <section
      aria-label={t('Rating summary')}
      className={cn(
        'grid gap-6 border border-structural bg-surface p-6',
        'sm:grid-cols-[auto_1fr]',
        className,
      )}
    >
      <div className="flex flex-col gap-2">
        <p className="font-display text-display-m leading-none">
          {mean !== undefined ? decimal(mean) : '—'}
        </p>
        {mean !== undefined ? <Rating value={mean} /> : null}
        <p className="meta text-muted">
          {count === 1
            ? t('1 review counts')
            : t('{{n}} reviews count', { n: count })}
        </p>
      </div>

      <ol className="flex flex-col gap-1.5" aria-label={t('Reviews by rating')}>
        {[5, 4, 3, 2, 1].map((stars) => {
          const n = distribution[stars - 1] ?? 0

          return (
            <li key={stars} className="flex items-center gap-3">
              <span className="meta w-6 text-muted">{stars}★</span>
              <span className="h-2 flex-1 bg-surface-sunken">
                <motion.span
                  className="block h-full bg-signal"
                  initial={reduceMotion ? false : { width: 0 }}
                  animate={{ width: `${(n / most) * 100}%` }}
                  transition={{
                    duration: reduceMotion ? 0 : 0.6,
                    delay: reduceMotion ? 0 : (5 - stars) * 0.05,
                    ease: [0.2, 0.8, 0.2, 1],
                  }}
                />
              </span>
              <span className="meta w-8 text-right text-secondary">{n}</span>
            </li>
          )
        })}
      </ol>

      <p className="text-body-s text-muted sm:col-span-2">
        {t(
          'This index counts the current version of each review, leaving ' +
            'out withdrawn ones and what a moderator it follows hid.',
        )}
        {conflicted > 0
          ? ' ' +
            (conflicted === 1
              ? t(
                  '{{n}} by the entity’s own representative is shown but ' +
                    'not counted.',
                  { n: conflicted },
                )
              : t(
                  '{{n}} by the entity’s own representative are shown but ' +
                    'not counted.',
                  { n: conflicted },
                ))
          : ''}
      </p>
    </section>
  )
}
