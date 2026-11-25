/**
 * Section 7 — the open network.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { Propagation } from '@/site/landing/motifs/Propagation'
import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { NETWORK_FACTS, NETWORK_POINTS } from '@/site/landing/content'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the Network section.
 *
 * @returns The Network section.
 */
export function Network() {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <Section id="network">
      <Container className="flex flex-col gap-16">
        <SectionHeading
          eyebrow={t('Understand the network')}
          title={t('No server gets to be the authority.')}
        />

        <div
          className={cn(
            'grid grid-cols-1 gap-12 lg:grid-cols-[1fr_24rem] lg:gap-20',
          )}
        >
          <ul className="flex flex-col border-t border-line">
            {NETWORK_POINTS.map((point, index) => (
              <motion.li
                key={point}
                {...reveal(index * 0.06)}
                className="flex items-start gap-4 border-b border-line py-6"
              >
                {/* The circle is an actor. */}
                <span
                  aria-hidden="true"
                  className={cn(
                    'mt-2 size-2.5 shrink-0 rounded-full border border-signal',
                  )}
                />
                <p className="text-body-l text-secondary text-pretty">
                  {t(point)}
                </p>
              </motion.li>
            ))}
          </ul>

          <motion.div {...reveal(0.1)} className="flex flex-col gap-8">
            <div className="border border-structural bg-surface p-8">
              <Propagation />
            </div>

            <dl className="grid grid-cols-3 border-t border-line">
              {NETWORK_FACTS.map((fact) => (
                <div key={fact.label} className="flex flex-col gap-2 py-5">
                  <dt className="sr-only">{t(fact.label)}</dt>
                  <dd
                    className={cn(
                      'font-display text-h2 font-semibold tracking-display',
                      'text-signal',
                    )}
                  >
                    {fact.value}
                  </dd>
                  <p aria-hidden="true" className="meta text-muted">
                    {t(fact.label)}
                  </p>
                </div>
              ))}
            </dl>
          </motion.div>
        </div>
      </Container>
    </Section>
  )
}
