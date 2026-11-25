/**
 * Section 6 — history stays visible.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { VersionStack } from '@/components/ecoa/VersionStack'
import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { HISTORY } from '@/site/landing/content'
import { cn } from '@/components/ui/cn'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the History section.
 *
 * @returns The History section.
 */
export function History() {
  const { t } = useTranslation()
  const reveal = useReveal()
  const reduceMotion = useReducedMotion()

  return (
    <Section id="history" surface="light">
      <Container className="flex flex-col gap-16">
        <div
          className={cn(
            'grid grid-cols-1 items-start gap-10 lg:grid-cols-[1fr_auto]',
            'lg:gap-16',
          )}
        >
          <SectionHeading
            eyebrow={t('See its history')}
            title={t('An edit never happens in secret.')}
            lede={t(
              'When someone changes a review, the old one does not vanish. ' +
                'Both versions stay, and you can see what changed and when.',
            )}
          />

          <motion.div {...reveal(0.18)} className="justify-self-end">
            <VersionStack />
          </motion.div>
        </div>

        <ol className="flex flex-col">
          {HISTORY.map((entry, index) => {
            const isCurrent = entry.state === 'current'

            return (
              <motion.li
                key={entry.title}
                {...reveal(index * 0.08)}
                className="grid grid-cols-[2rem_1fr] gap-x-6"
              >
                <div className="flex flex-col items-center">
                  {/* Outline carries visibility, fill carries meaning. */}
                  <span
                    aria-hidden="true"
                    className={cn(
                      'size-3.5 border border-structural',
                      isCurrent ? 'bg-signal' : 'bg-transparent',
                    )}
                  />
                  {index < HISTORY.length - 1 ? (
                    <motion.span
                      aria-hidden="true"
                      className="w-px flex-1 origin-top bg-strong"
                      initial={reduceMotion ? false : { scaleY: 0 }}
                      whileInView={{ scaleY: 1 }}
                      viewport={{ once: true, amount: 0.6 }}
                      transition={
                        reduceMotion
                          ? { duration: 0 }
                          : {
                              duration: 0.4,
                              delay: index * 0.08 + 0.2,
                              ease: [0.2, 0.8, 0.2, 1],
                            }
                      }
                    />
                  ) : null}
                </div>

                <div className="flex flex-col gap-2 pb-10">
                  <div
                    className={cn(
                      'flex flex-wrap items-baseline gap-x-4 gap-y-1',
                    )}
                  >
                    {/* Ink, not the brand blue: 2.7:1 on paper. */}
                    <h3 className="text-h4 font-semibold text-primary">
                      {t(entry.title)}
                    </h3>
                    <span className="meta text-muted">{t(entry.at)}</span>
                  </div>

                  <p
                    className={cn(
                      'reading-measure text-body-m text-secondary text-pretty',
                    )}
                  >
                    {t(entry.note)}
                  </p>

                  {/* For the reader who wants the machine name. */}
                  <p className="font-mono text-mono-s text-muted">
                    {entry.event}
                  </p>
                </div>
              </motion.li>
            )
          })}
        </ol>

        <motion.p
          {...reveal()}
          className={cn(
            'reading-measure border-l-2 border-structural pl-6 text-body-l',
            'text-primary text-pretty',
          )}
        >
          {t(
            'A platform that lets an edit disappear is asking you to trust ' +
              'its memory. Ecoa asks you to trust arithmetic instead.',
          )}
        </motion.p>
      </Container>
    </Section>
  )
}
