/**
 * The Protocol Layer on demand: the signed event behind a review.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import * as Dialog from '@radix-ui/react-dialog'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import type { EventInspectorProps } from '@/components/ecoa/EventInspector.t'
import { Badge } from '@/components/ui/Badge'
import { known } from '@/i18n/i18n'

// --- GLOBALS ---
// never colour alone: each state has a glyph and a label
const CHECKS = {
  checking: { tone: 'neutral', glyph: '●', label: 'Checking the signature…' },
  valid: { tone: 'valid', glyph: '◆', label: 'Signature valid' },
  invalid: { tone: 'danger', glyph: '■', label: 'Signature does not hold' },
  unchecked: { tone: 'warning', glyph: '●', label: 'Not checked here' },
} as const

/**
 * Shows an envelope formatted for reading; the string as stored when it
 * does not parse.
 *
 * @param raw - The envelope, as the network stored it.
 * @returns The text to show.
 */
function readable(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

// --- CODE ---
/**
 * Renders one labelled line of the envelope.
 *
 * @param props - The row label and its value.
 * @returns The row.
 */
function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div
      className={cn(
        'grid grid-cols-[9rem_1fr] gap-4 border-b border-line px-6 py-4',
      )}
    >
      <dt className="meta text-muted">{label}</dt>
      <dd className="font-mono text-mono-s break-all text-primary">
        {children}
      </dd>
    </div>
  )
}

/**
 * Renders the drawer holding a review's signed event.
 *
 * @param props - Whether the drawer is open, the handler that changes it,
 * and the event, or why it could not be read.
 * @returns The drawer.
 */
export function EventInspector({
  open,
  onOpenChange,
  event,
  error,
}: EventInspectorProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay asChild>
          <motion.div
            className="fixed inset-0 z-100 bg-scrim"
            initial={reduceMotion ? false : { opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: reduceMotion ? 0 : 0.2 }}
          />
        </Dialog.Overlay>

        <Dialog.Content asChild>
          <motion.aside
            data-surface="dark"
            className={cn(
              'fixed inset-y-0 right-0 z-100 flex w-[min(40rem,100%)] flex-col',
              'overflow-y-auto border-l border-structural bg-canvas',
            )}
            initial={reduceMotion ? false : { x: '100%' }}
            animate={{ x: 0 }}
            transition={
              reduceMotion
                ? { duration: 0 }
                : { duration: 0.32, ease: [0.2, 0.8, 0.2, 1] }
            }
          >
            <header
              className={cn(
                'flex items-start justify-between gap-4 border-b',
                'border-structural px-6 py-5',
              )}
            >
              <div className="flex flex-col gap-2">
                <Dialog.Title
                  className={cn(
                    'font-display text-h3 font-semibold tracking-heading',
                  )}
                >
                  {t('Signed event')}
                </Dialog.Title>
                <Dialog.Description className="text-body-s text-secondary">
                  {t(
                    'Everything a stranger needs to check this review, ' +
                      'without asking Ecoa for permission.',
                  )}
                </Dialog.Description>
              </div>
              <Dialog.Close
                aria-label={t('Close event inspector')}
                className={cn(
                  'meta cursor-pointer border border-structural px-3 py-2',
                  'text-secondary transition-colors duration-interface',
                  'hover:text-primary',
                )}
              >
                {t('Close')}
              </Dialog.Close>
            </header>

            {error ? (
              <p role="alert" className="px-6 py-5 text-danger">
                {error}
              </p>
            ) : !event ? (
              <p role="status" className="meta px-6 py-5 text-muted">
                {t('Reading the event…')}
              </p>
            ) : (
              <motion.div
                // the event fades in where "Reading the event…" was
                initial={reduceMotion ? false : { opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: reduceMotion ? 0 : 0.25 }}
              >
                <dl className="flex flex-col">
                  {event.rows.map((row) => (
                    <Row
                      key={`${row.label}:${row.value}`}
                      label={known(t, row.label)}
                    >
                      {row.value}
                    </Row>
                  ))}
                </dl>

                <div
                  className={cn(
                    'flex flex-wrap items-center gap-2 border-b border-line',
                    'px-6 py-5',
                  )}
                >
                  {/* the verdict pops in when the check is done */}
                  <AnimatePresence mode="wait" initial={false}>
                    {event.check ? (
                      <motion.span
                        key={event.check.state}
                        className="inline-flex"
                        initial={
                          reduceMotion ? false : { opacity: 0, scale: 0.85 }
                        }
                        animate={{ opacity: 1, scale: 1 }}
                        exit={reduceMotion ? undefined : { opacity: 0 }}
                        transition={
                          reduceMotion
                            ? { duration: 0 }
                            : { type: 'spring', stiffness: 420, damping: 22 }
                        }
                      >
                        <Badge
                          tone={CHECKS[event.check.state].tone}
                          glyph={CHECKS[event.check.state].glyph}
                        >
                          {t(CHECKS[event.check.state].label)}
                        </Badge>
                      </motion.span>
                    ) : null}
                  </AnimatePresence>
                  {event.badges?.map((badge) => (
                    <Badge
                      key={badge.label}
                      tone={badge.tone}
                      glyph={badge.glyph}
                    >
                      {known(t, badge.label)}
                    </Badge>
                  ))}
                  {event.check?.state === 'invalid' ? (
                    <p className="w-full text-body-s text-danger">
                      {known(t, event.check.reason)}
                    </p>
                  ) : null}
                  {event.check?.state === 'unchecked' ? (
                    <p className="w-full text-body-s text-secondary">
                      {known(t, event.check.reason)}
                    </p>
                  ) : null}
                </div>

                {event.notes?.length ? (
                  <ul
                    className={cn(
                      'flex flex-col gap-1 border-b border-line px-6 py-4',
                      'text-body-s text-secondary',
                    )}
                  >
                    {event.notes.map((note) => (
                      <li key={note}>{known(t, note)}</li>
                    ))}
                  </ul>
                ) : null}

                <div className="flex flex-col gap-3 px-6 py-5">
                  <p className="meta text-muted">{t('Raw envelope')}</p>
                  <pre
                    className={cn(
                      'overflow-x-auto border border-line bg-surface-sunken',
                      'p-4 font-mono text-mono-s text-secondary',
                    )}
                  >
                    {readable(event.raw)}
                  </pre>
                  <p className="text-body-s text-muted">
                    {t(
                      'Formatted for reading. The signature covers its ' +
                        'canonical form (JCS), which any copy of it gives ' +
                        'back.',
                    )}
                  </p>
                </div>
              </motion.div>
            )}
          </motion.aside>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
