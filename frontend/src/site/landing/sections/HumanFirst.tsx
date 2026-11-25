/**
 * Section 8 — back to the person.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { CircleMark } from '@/components/ecoa/CircleMark'
import { Rating } from '@/components/ecoa/Rating'
import { Container } from '@/components/ui/Container'
import { Section } from '@/site/landing/Section'
import { HUMAN_REVIEWS } from '@/site/landing/content'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the HumanFirst section.
 *
 * @returns The HumanFirst section.
 */
export function HumanFirst() {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <Section id="human-first" surface="light">
      <Container className="flex flex-col gap-16">
        <motion.blockquote
          {...reveal()}
          className={cn(
            'max-w-5xl font-display text-display-m font-semibold',
            'tracking-display text-balance',
          )}
        >
          {t('Your experience shouldn’t disappear inside')}{' '}
          <span className="relative inline-block">
            {t('someone else’s')}
            <CircleMark
              variant="underline"
              className="absolute -bottom-2 left-0 h-3 w-full overflow-visible"
              delay={0.25}
            />
          </span>{' '}
          {t('database.')}
        </motion.blockquote>

        <motion.p
          {...reveal(0.1)}
          className="reading-measure text-body-l text-secondary text-pretty"
        >
          {t(
            'Everything above exists so that this part can be ordinary: a ' +
              'person writes what happened, and it stays theirs.',
          )}
        </motion.p>

        <ul
          className={cn(
            'grid grid-cols-1 gap-px border border-structural bg-strong',
            'md:grid-cols-3',
          )}
        >
          {HUMAN_REVIEWS.map((review, index) => (
            <motion.li
              key={review.entity}
              {...reveal(index * 0.08)}
              className="flex flex-col gap-5 bg-surface p-8"
            >
              <div className="flex flex-col gap-1">
                <span className="meta text-muted">{t(review.kind)}</span>
                <span
                  className={cn(
                    'font-display text-h4 font-semibold tracking-heading',
                  )}
                >
                  {review.entity}
                </span>
              </div>

              <Rating value={review.rating} />

              {/* Human content: no tracking, no mono. */}
              <p className="flex-1 text-body-m text-secondary text-pretty">
                {t(review.body)}
              </p>

              <span className="meta text-muted">{review.author}</span>
            </motion.li>
          ))}
        </ul>
      </Container>
    </Section>
  )
}
