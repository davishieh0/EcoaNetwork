/**
 * The block at the foot of the composer: what will be signed once the draft is
 * checked.
 */

// --- IMPORTS ---
import { useEffect, useRef } from 'react'
import { motion, useReducedMotion } from 'motion/react'

import { Seal } from '@/app/write/Seal'
import type {
  Published,
  SignedPhase,
  SignedPreviewProps,
} from '@/app/write/Write.t'
import { Button } from '@/components/ui/Button'
import { cn } from '@/components/ui/cn'
import { NETWORK } from '@/logic/protocol/envelope'
import { characters } from '@/logic/protocol/review'

// --- GLOBALS ---
const LANGUAGES: Record<string, string> = {
  en: 'English',
  'pt-BR': 'Portuguese (Brazil)',
  es: 'Spanish',
}

const OUTCOMES: Record<Published, string> = {
  accepted: 'Signed and published',
  already_exists: 'Already on the network',
  orphan: 'Held until what it refers to arrives',
}

// one half-circle cut into the ticket's edge, filled with the page colour
const NOTCH = 'absolute size-5 rounded-full bg-canvas'

// the ticket's paper: grey while a draft, blue once the node took it
const PAPER = cn(
  'transition-colors delay-200 duration-700 ease-standard',
  'motion-reduce:transition-none',
)

// --- CODE ---
/**
 * Renders one labelled segment of the block.
 *
 * @param props - The label, the value, and whether the value is data.
 * @returns The segment.
 */
function Segment({
  label,
  value,
  mono = false,
  breakAll = false,
}: {
  label: string
  value: string
  mono?: boolean
  breakAll?: boolean
}) {
  return (
    <div
      className={cn(
        'grid grid-cols-[7.5rem_1fr] gap-3 border-b border-quiet py-2',
        'last:border-b-0',
      )}
    >
      <dt className="meta text-muted">{label}</dt>
      <dd
        className={cn(
          'min-w-0 text-primary',
          breakAll ? 'break-all' : 'break-words',
          mono ? 'font-mono text-mono-m tabular-nums' : 'text-body-s',
        )}
      >
        {value}
      </dd>
    </div>
  )
}

/**
 * Draws the tear line: a dashed rule between two notches cut into the edges.
 *
 * @param props - Whether the ticket was torn along it.
 * @returns The tear line.
 */
function Perforation({ torn = false }: { torn?: boolean }) {
  return (
    <div aria-hidden="true" className="relative flex h-5 items-center">
      <span className={cn(NOTCH, '-left-2.5')} />
      <span
        className={cn(
          'mx-4 w-full border-t-2 border-dashed border-line',
          'transition-opacity duration-300',
          torn && 'opacity-0',
        )}
      />
      <span className={cn(NOTCH, '-right-2.5')} />
    </div>
  )
}

/**
 * Words where the review stands, under the title.
 *
 * @param phase - The phase the review is in.
 * @returns The status line.
 */
function statusOf(phase: SignedPhase): string {
  if (phase.name === 'checked') return 'Not signed yet'
  if (phase.name === 'publishing') return 'Signing and sending'

  return OUTCOMES[phase.outcome]
}

/**
 * Shows the segments once the draft is checked.
 *
 * @param props - The phase the review is in and the way back to edit.
 * @returns The block.
 */
export function SignedPreview({ phase, onEdit }: SignedPreviewProps) {
  const reduceMotion = useReducedMotion()
  const block = useRef<HTMLElement>(null)

  // the node took the event, so the ticket is sealed
  const sealed =
    phase.name === 'published' &&
    ['accepted', 'already_exists'].includes(phase.outcome)

  useEffect(() => {
    const reduce =
      window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false

    // scroll the opened block under the header once it has its full height
    const frame = requestAnimationFrame(() => {
      block.current?.scrollIntoView({
        behavior: reduce ? 'auto' : 'smooth',
        block: 'start',
      })
    })

    return () => cancelAnimationFrame(frame)
  }, [])

  return (
    <section
      ref={block}
      aria-labelledby="signed-preview"
      className="flex scroll-mt-[calc(var(--header-h)+1.5rem)] flex-col"
    >
      <div
        className={cn(
          'flex flex-col',
          PAPER,
          sealed ? 'bg-signal-wash' : 'bg-surface-raised',
        )}
      >
        <header className="grid grid-cols-[1fr_7rem] items-end gap-4 px-6 pt-5">
          <div className="flex flex-col gap-1">
            <h2
              id="signed-preview"
              className="font-display text-h3 font-semibold tracking-heading"
            >
              What gets signed
            </h2>
            <p className="meta text-muted" aria-live="polite">
              {statusOf(phase)}
            </p>
          </div>
          <Seal ready sealed={sealed} />
        </header>

        <dl className="flex flex-col px-6 pt-4 pb-2">
          <Segment label="Event type" value="review.created" mono />
          <Segment label="Entity" value={phase.payload.entity} mono />
          <Segment label="Rating" value={`${phase.payload.rating} of 5`} mono />
          {phase.payload.title ? (
            <Segment label="Title" value={phase.payload.title} />
          ) : null}
          <Segment
            label="Body"
            value={`${characters(phase.payload.body)} characters`}
            mono
          />
          {phase.payload.lang ? (
            <Segment
              label="Language"
              value={LANGUAGES[phase.payload.lang] ?? phase.payload.lang}
            />
          ) : null}
          <Segment label="Created at" value={phase.createdAt} mono />
          <Segment label="Network" value={NETWORK} mono />
          {phase.name === 'published' ? (
            <>
              <Segment
                label="Author"
                value={phase.envelope.author}
                mono
                breakAll
              />
              <Segment
                label="Event ID"
                value={phase.envelope.eventId}
                mono
                breakAll
              />
            </>
          ) : null}
        </dl>
      </div>

      <motion.div
        className={cn(
          'flex flex-col',
          PAPER,
          sealed ? 'bg-signal-wash' : 'bg-surface-raised',
        )}
        style={{ transformOrigin: 'top left' }}
        initial={false}
        animate={
          sealed
            ? { y: 10, rotate: 1.5, marginBottom: 28 }
            : { y: 0, rotate: 0, marginBottom: 0 }
        }
        transition={
          reduceMotion
            ? { duration: 0 }
            : { type: 'spring', stiffness: 140, damping: 9, delay: 0.25 }
        }
      >
        <Perforation torn={sealed} />

        <div
          className={cn(
            'flex flex-wrap items-end justify-between gap-4 p-6 pt-4',
          )}
        >
          <p className="max-w-md text-body-s text-secondary text-pretty">
            Once published, this stays on the network. You can publish a new
            version or withdraw it later, but the original is never erased.
          </p>
          {phase.name === 'checked' ? (
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={onEdit}
            >
              Edit review
            </Button>
          ) : null}
        </div>
      </motion.div>
    </section>
  )
}
