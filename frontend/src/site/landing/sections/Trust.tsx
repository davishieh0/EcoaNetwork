/**
 * Section 5 — trust without a universal checkmark.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { ClaimGlyph } from '@/components/ecoa/ClaimGlyph'
import { Badge } from '@/components/ui/Badge'
import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { CLAIMS } from '@/site/landing/content'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the Trust section.
 *
 * @returns The Trust section.
 */
export function Trust() {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <Section id="trust">
      <Container className="flex flex-col gap-16">
        <SectionHeading
          eyebrow={t('Understand trust')}
          title={t('There is no blue tick, and that is the feature.')}
          lede={t(
            'Ecoa never certifies a person. It shows which specific claim ' +
              'someone made, who checked it, and what that check does not ' +
              'cover.',
          )}
        />

        <ul className="grid grid-cols-1 border-t border-line md:grid-cols-2">
          {CLAIMS.map((claim, index) => (
            <motion.li
              key={claim.label}
              {...reveal(index * 0.06)}
              className={cn(
                'flex flex-col gap-4 border-b border-line py-8 md:odd:pr-10',
                'md:even:border-l md:even:pl-10',
              )}
            >
              <div
                className={cn(
                  'flex flex-wrap items-center justify-between gap-4',
                )}
              >
                <Badge tone={claim.tone} glyph="◆">
                  {t(claim.label)}
                </Badge>
                <ClaimGlyph kind={claim.glyph} />
              </div>

              <dl className="flex flex-col gap-3">
                <div className="flex flex-col gap-1">
                  <dt className="meta text-muted">{t('Asserts')}</dt>
                  <dd className="text-body-m text-primary text-pretty">
                    {t(claim.asserts)}
                  </dd>
                </div>
                <div className="flex flex-col gap-1">
                  <dt className="meta text-muted">{t('Checked by')}</dt>
                  <dd className="text-body-s text-secondary text-pretty">
                    {t(claim.checkedBy)}
                  </dd>
                </div>
                <div className="flex flex-col gap-1">
                  {/* The line a single badge always leaves out. */}
                  <dt className="meta text-muted">{t('Does not mean')}</dt>
                  <dd className="text-body-s text-secondary text-pretty">
                    {t(claim.doesNotMean)}
                  </dd>
                </div>
              </dl>
            </motion.li>
          ))}
        </ul>
      </Container>
    </Section>
  )
}
