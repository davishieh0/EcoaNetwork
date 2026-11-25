/**
 * Section 2 — why Ecoa needs to exist.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { CircleMark } from '@/components/ecoa/CircleMark'
import { ProblemGlyph } from '@/site/landing/motifs/ProblemGlyph'
import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { COMPARISON, PROBLEMS } from '@/site/landing/content'
import { cn } from '@/components/ui/cn'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the Problem section.
 *
 * @returns The Problem section.
 */
export function Problem() {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <Section id="why-ecoa" surface="light">
      <Container className="flex flex-col gap-20">
        <SectionHeading
          eyebrow={t('Why this needs to exist')}
          title={
            <>
              {t('A review you')}{' '}
              <span className="relative z-0 inline-block">
                {t('cannot check')}
                {/* The same hand as the hero, on this section's phrase. */}
                <CircleMark
                  className={cn(
                    'absolute -inset-x-4 -inset-y-2 -z-10 overflow-visible',
                  )}
                  delay={0.25}
                />
              </span>{' '}
              {t('is just a rumour with stars.')}
            </>
          }
          lede={t(
            'Reviews already decide what people buy, eat and trust. The ' +
              'problem is not that they exist. It is that almost none of ' +
              'them can be verified by the person reading them.',
          )}
        />

        <ol className="flex flex-col border-t border-line">
          {PROBLEMS.map((problem, index) => (
            <motion.li
              key={problem.index}
              {...reveal(index * 0.06)}
              className={cn(
                'grid grid-cols-1 gap-x-8 gap-y-4 border-b border-line py-10',
                'md:grid-cols-[4rem_5rem_15rem_1fr] md:items-start',
              )}
            >
              <span
                className={cn(
                  'font-display text-h1 font-semibold tracking-display',
                  'text-accent',
                )}
              >
                {problem.index}
              </span>
              <ProblemGlyph kind={problem.glyph} />
              <h3 className="text-h4 font-semibold text-pretty md:pt-1">
                {t(problem.title)}
              </h3>
              <p className="text-body-m text-secondary text-pretty md:pt-1.5">
                {t(problem.body)}
              </p>
            </motion.li>
          ))}
        </ol>

        <div className="flex flex-col gap-6">
          <motion.h3 {...reveal()} className="meta text-muted">
            {t('The same question, answered twice')}
          </motion.h3>

          {/* Paired rows make it an argument, not two lists. */}
          <div className="-mx-5 overflow-x-auto px-5">
            <table className="w-full min-w-[42rem] border-collapse text-left">
              <thead>
                <tr className="border-y border-structural">
                  <th
                    scope="col"
                    className="meta w-[14rem] px-4 py-4 text-muted"
                  >
                    <span className="sr-only">{t('Dimension')}</span>
                  </th>
                  <th scope="col" className="meta px-4 py-4 text-secondary">
                    {t('Traditional platform')}
                  </th>
                  {/* The hero's blue arrives as a fill. */}
                  <th
                    scope="col"
                    className="meta bg-accent px-4 py-4 text-on-accent"
                  >
                    Ecoa
                  </th>
                </tr>
              </thead>
              <tbody>
                {COMPARISON.map((row, index) => (
                  <motion.tr
                    key={row.aspect}
                    {...reveal(index * 0.05)}
                    className="border-b border-line align-top"
                  >
                    <th
                      scope="row"
                      className="meta px-4 py-5 font-medium text-muted"
                    >
                      {t(row.aspect)}
                    </th>
                    <td
                      className={cn(
                        'px-4 py-5 text-body-s text-secondary text-pretty',
                      )}
                    >
                      <span aria-hidden="true" className="mr-2 text-muted">
                        ○
                      </span>
                      {t(row.traditional)}
                    </td>
                    <td
                      className={cn(
                        'bg-accent-wash px-4 py-5 text-body-s text-primary',
                        'text-pretty',
                      )}
                    >
                      {/* The square is the artifact. */}
                      <span aria-hidden="true" className="mr-2 text-primary">
                        &#9632;
                      </span>
                      {t(row.ecoa)}
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </Container>
    </Section>
  )
}
