/**
 * Section 3 — the flow, in four steps.
 */

// --- IMPORTS ---
import { useRef, useState } from 'react'
import {
  motion,
  useMotionValueEvent,
  useReducedMotion,
  useScroll,
} from 'motion/react'
import { useTranslation } from 'react-i18next'

import { FlowDiagram } from '@/site/landing/motifs/FlowDiagram'
import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { STEPS } from '@/site/landing/content'
import { cn } from '@/components/ui/cn'
import { useMediaQuery } from '@/components/ui/useMediaQuery'

// --- CODE ---
/**
 * Builds the HowItWorks section.
 *
 * @returns The HowItWorks section.
 */
export function HowItWorks() {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()
  const stepsRef = useRef<HTMLOListElement>(null)
  const [active, setActive] = useState(0)
  // Only where the diagram can stay on screen beside the steps.
  const canStick = useMediaQuery('(min-width: 1024px)')

  const { scrollYProgress } = useScroll({
    target: stepsRef,
    offset: ['start 65%', 'end 65%'],
  })

  useMotionValueEvent(scrollYProgress, 'change', (value) => {
    if (reduceMotion || !canStick) return
    const index = Math.floor(value * STEPS.length)
    setActive(Math.max(0, Math.min(STEPS.length - 1, index)))
  })

  // The diagram stands finished in two cases.
  const step = reduceMotion || !canStick ? STEPS.length - 1 : active

  return (
    // No exposed grid here.
    <Section id="how-it-works">
      <Container className="flex flex-col gap-16">
        <SectionHeading
          eyebrow={t('How Ecoa works')}
          title={t('Four steps, and none of them need our permission.')}
          lede={t(
            'A review is not a row in our database. It is a signed object ' +
              'that carries everything a stranger needs in order to check it.',
          )}
        />

        <div
          className={cn(
            'grid grid-cols-1 gap-12 lg:grid-cols-[1fr_28rem] lg:gap-20',
          )}
        >
          <ol ref={stepsRef} className="flex flex-col border-t border-line">
            {STEPS.map((item, index) => {
              const reached = step >= index

              return (
                <li
                  key={item.index}
                  className={cn(
                    'grid grid-cols-[3rem_1fr] gap-x-6 border-b border-line',
                    'py-10',
                  )}
                >
                  <div className="flex flex-col items-center gap-3">
                    {/* The column doubles as a progress rail. */}
                    <motion.span
                      aria-hidden="true"
                      className="size-3 border border-signal"
                      animate={{
                        backgroundColor: reached
                          ? 'var(--ecoa-signal)'
                          : 'rgba(0,0,0,0)',
                      }}
                      transition={
                        reduceMotion ? { duration: 0 } : { duration: 0.3 }
                      }
                    />
                    {index < STEPS.length - 1 ? (
                      <span
                        aria-hidden="true"
                        className="w-px flex-1 bg-strong"
                      />
                    ) : null}
                  </div>

                  <div className="flex flex-col gap-3">
                    <span className="meta text-signal">{item.index}</span>
                    <h3
                      className={cn(
                        'font-display text-h3 font-semibold tracking-heading',
                        'transition-colors duration-surface',
                        reached ? 'text-primary' : 'text-muted',
                      )}
                    >
                      {t(item.title)}
                    </h3>
                    <p className="text-body-m text-secondary text-pretty">
                      {t(item.body)}
                    </p>
                  </div>
                </li>
              )
            })}
          </ol>

          {/* Sticky only where there is room for it. */}
          <div className="order-first lg:order-none">
            <div className="lg:sticky lg:top-[calc(var(--header-h)+3rem)]">
              <div
                className={cn(
                  'flex flex-col gap-4 border border-structural bg-surface p-8',
                )}
              >
                <p className="meta text-muted">
                  {t('Step {{n}} / 04', {
                    n: String(step + 1).padStart(2, '0'),
                  })}
                </p>
                <FlowDiagram step={step} />
                <p className="meta text-secondary">{t(STEPS[step].title)}</p>
              </div>
            </div>
          </div>
        </div>
      </Container>
    </Section>
  )
}
