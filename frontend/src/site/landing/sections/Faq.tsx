/**
 * Section 10 — the FAQ.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import * as Accordion from '@radix-ui/react-accordion'
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { FAQ } from '@/site/landing/content'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Builds the Faq section.
 *
 * @returns The Faq section.
 */
export function Faq() {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <Section id="faq" surface="light">
      <Container
        className={cn(
          'grid grid-cols-1 gap-12 lg:grid-cols-[20rem_1fr] lg:gap-20',
        )}
      >
        <SectionHeading
          eyebrow={t('FAQ')}
          title={t('Short answers.')}
          className="gap-4"
        />

        <Accordion.Root
          type="single"
          collapsible
          className="border-t border-structural"
        >
          {FAQ.map((item, index) => (
            <motion.div key={item.question} {...reveal(index * 0.05)}>
              <Accordion.Item
                value={item.question}
                className="border-b border-line"
              >
                <Accordion.Header>
                  <Accordion.Trigger
                    className={cn(
                      'group flex w-full cursor-pointer items-center gap-6',
                      'py-6',
                      'text-left',
                    )}
                  >
                    <span
                      aria-hidden="true"
                      className="meta w-8 shrink-0 text-muted"
                    >
                      {String(index + 1).padStart(2, '0')}
                    </span>
                    <span className="flex-1 text-h4 font-semibold text-pretty">
                      {t(item.question)}
                    </span>
                    {/* One identity that moves, not two that alternate. */}
                    <span
                      aria-hidden="true"
                      className={cn(
                        'shrink-0 text-h4 text-primary transition-transform',
                        'duration-interface ease-standard',
                        'group-data-[state=open]:rotate-45',
                      )}
                    >
                      +
                    </span>
                  </Accordion.Trigger>
                </Accordion.Header>

                <Accordion.Content
                  className={cn(
                    'overflow-hidden data-[state=closed]:animate-accordion-up',
                    'data-[state=open]:animate-accordion-down',
                  )}
                >
                  <p
                    className={cn(
                      'reading-measure pb-6 text-body-m text-secondary',
                      'text-pretty',
                    )}
                  >
                    {t(item.answer)}
                  </p>
                </Accordion.Content>
              </Accordion.Item>
            </motion.div>
          ))}
        </Accordion.Root>
      </Container>
    </Section>
  )
}
