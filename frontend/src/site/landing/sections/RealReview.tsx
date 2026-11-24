/**
 * Section 4 — a review is more than a card.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { useState } from 'react'
import { motion } from 'motion/react'

import { SealMotif } from '@/site/landing/motifs/SealMotif'
import { EventInspector } from '@/components/ecoa/EventInspector'
import type { InspectedEvent } from '@/components/ecoa/EventInspector.t'
import type { Review } from '@/components/ecoa/ReviewCard.t'
import { ReviewCard } from '@/components/ecoa/ReviewCard'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { Section, SectionHeading } from '@/site/landing/Section'
import { useReveal } from '@/components/ui/useReveal'

// --- GLOBALS ---
const REVIEW: Review = {
  entity: 'Marés Bakery',
  entityKind: 'Local business · Lisbon',
  rating: 4,
  title: 'They fixed a mistake nobody would have noticed',
  body:
    'The order was short one loaf. I mentioned it the next morning and they ' +
    'had already logged it, no argument and no receipt hunt. That is rarer ' +
    'than good bread.',
  author: 'joana.k',
  authorClaim: { label: 'Human claim', tone: 'valid' },
  publishedAt: '2026-08-14',
  eventId: '8f2c…a41',
  observedBy: 6,
  edited: 2,
}

/** The card's signed event, as the inspector shows it. */
const EVENT: InspectedEvent = {
  rows: [
    { label: 'Event ID', value: 'evt_8f2c9e14a41b7d3f…' },
    { label: 'Type', value: 'review.replaced' },
    { label: 'Author', value: 'did:key:z6Mkf5rGMoatrSj1f4CyvuHBeXJELe9RPd' },
    { label: 'Entity', value: 'domain:maresbakery.pt' },
    { label: 'Version', value: '2 of 2 · replaces evt_4d10…8c22' },
  ],
  check: { state: 'valid' },
  badges: [
    { label: 'Domain verified', tone: 'valid', glyph: '◆' },
    { label: 'Observed by 06 nodes', tone: 'neutral', glyph: '●' },
  ],
  raw: JSON.stringify({
    author: 'did:key:z6Mkf5rGMoatrSj1f4CyvuHBeXJELe9RPd',
    createdAt: '2026-08-14T09:12:40Z',
    deviceKey: null,
    eventId: 'evt_8f2c9e14a41b7d3f…',
    eventType: 'review.replaced',
    payload: { replaces: 'evt_4d10…8c22', rating: 4, body: '…' },
    protocol: 'ecoa',
    signature: 'c7f1…a9e2',
    version: 1,
  }),
}

/** What the card carries, as questions a reader would ask. */
const UNDERNEATH = [
  {
    question: 'Who wrote it?',
    answer: 'The same person who wrote 40 other reviews, and you can prove it.',
    technical: 'did:key:z6Mkf5rG…e9RPd',
  },
  {
    question: 'Could someone change the words?',
    answer: 'No. Any edit to the text breaks the seal, and anyone can notice.',
    technical: 'sig 3045022100c7f1…a9e2',
  },
  {
    question: 'Has it been edited?',
    answer: 'Once, two days later. The first version is still there to read.',
    technical: 'version 2 of 2',
  },
  {
    question: 'Who else has a copy?',
    answer: 'Six independent computers, none of them ours.',
    technical: 'observed by 06 nodes',
  },
  {
    question: 'What was actually checked?',
    answer:
      'That a real person is behind the account, and that a purchase happened.',
    technical: 'claims: human, purchase',
  },
]

// --- CODE ---
/**
 * Builds the RealReview section.
 *
 * @returns The RealReview section.
 */
export function RealReview() {
  const reveal = useReveal()
  const [inspecting, setInspecting] = useState(false)

  return (
    <Section id="a-real-review" surface="light">
      <Container className="flex flex-col gap-16">
        <SectionHeading
          eyebrow="See a real review"
          title="A review is more than a card."
          lede={
            'On the left, what you normally get. On the right, the questions ' +
            'you could never ask about it before.'
          }
        />

        <div className="grid grid-cols-1 gap-10 lg:grid-cols-2 lg:gap-16">
          <motion.div {...reveal()} className="flex flex-col gap-8">
            <ReviewCard review={REVIEW} onInspect={() => setInspecting(true)} />
            <SealMotif />
          </motion.div>

          <div className="flex flex-col">
            <motion.p
              {...reveal(0.08)}
              className="meta border-b border-structural pb-4 text-muted"
            >
              What you can ask about it
            </motion.p>

            <dl className="flex flex-col">
              {UNDERNEATH.map((row, index) => (
                <motion.div
                  key={row.question}
                  {...reveal(0.1 + index * 0.05)}
                  className="flex flex-col gap-2 border-b border-line py-5"
                >
                  <dt
                    className={cn(
                      'text-body-m font-semibold text-primary text-pretty',
                    )}
                  >
                    {row.question}
                  </dt>
                  <dd className="text-body-m text-secondary text-pretty">
                    {row.answer}
                  </dd>
                  <dd className="font-mono text-mono-s text-muted">
                    {row.technical}
                  </dd>
                </motion.div>
              ))}
            </dl>

            <motion.div {...reveal(0.4)} className="pt-8">
              <Button onClick={() => setInspecting(true)}>
                See the proof ↗
              </Button>
            </motion.div>
          </div>
        </div>
      </Container>

      <EventInspector
        open={inspecting}
        onOpenChange={setInspecting}
        event={EVENT}
      />
    </Section>
  )
}
