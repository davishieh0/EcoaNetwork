/**
 * Section 9 — the close.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { motion } from 'motion/react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { CircleMark } from '@/components/ecoa/CircleMark'
import { Button } from '@/components/ui/Button'
import { ROUTES } from '@/routes'
import { Container } from '@/components/ui/Container'
import { Section } from '@/site/landing/Section'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the FinalCta section.
 *
 * @returns The FinalCta section.
 */
export function FinalCta() {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <Section id="join" className="py-32 md:py-40">
      <Container className="flex flex-col items-center gap-10 text-center">
        <motion.h2
          {...reveal()}
          className={cn(
            'font-display text-display-l font-bold tracking-[-0.04em]',
            'uppercase text-balance',
          )}
        >
          {t('Every experience')}
          <br />
          {t('leaves an')}{' '}
          <span className="relative z-0 inline-block">
            {t('echo')}
            <CircleMark
              className="absolute -inset-x-6 -inset-y-5 -z-10 overflow-visible"
              delay={0.2}
            />
          </span>
          .
        </motion.h2>

        <motion.p
          {...reveal(0.1)}
          className={cn(
            'font-display text-h2 font-medium tracking-heading text-secondary',
          )}
        >
          {t('What will yours say?')}
        </motion.p>

        <motion.div
          {...reveal(0.18)}
          className="flex flex-wrap items-center justify-center gap-3"
        >
          <Button as={Link} to={ROUTES.explore} size="lg">
            {t('Explore Ecoa')}
          </Button>
          <Button as={Link} to={ROUTES.register} variant="secondary" size="lg">
            {t('Create identity')}
          </Button>
          <Button as={Link} to={ROUTES.write} variant="ghost" size="lg">
            {t('Write a review')}
          </Button>
        </motion.div>

        <motion.p {...reveal(0.26)} className="meta text-muted">
          {t('Reading needs no account')}
        </motion.p>
      </Container>
    </Section>
  )
}
