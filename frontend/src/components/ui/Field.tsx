/**
 * A labelled text control, with room for a hint or an error.
 */

// --- IMPORTS ---
import { useId, useState } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'

import type { FieldProps } from '@/components/ui/Field.t'
import { cn } from '@/components/ui/cn'

// --- CODE ---
/**
 * Renders a label, a control and the note under it.
 *
 * @param props - Label, optional hint or error, and the control's own props.
 * @returns The field.
 */
export function Field({
  label,
  hint,
  error,
  className,
  children,
  revealPassword = false,
  ...props
}: FieldProps) {
  const generated = useId()
  const [visible, setVisible] = useState(false)
  const canReveal = revealPassword && props.type === 'password'
  const reduceMotion = useReducedMotion()
  const id = props.id ?? generated
  const note = error ?? hint
  const noteId = note ? `${id}-note` : undefined

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <label htmlFor={id} className="meta text-secondary">
        {label}
      </label>

      <div className="relative flex flex-col">
        <input
          {...props}
          id={id}
          type={canReveal && visible ? 'text' : props.type}
          aria-describedby={noteId}
          aria-invalid={error ? true : undefined}
          className={cn(
            'min-h-11 rounded-none border bg-surface px-4',
            'text-body-m text-primary placeholder:text-muted',
            'transition-colors duration-interface ease-standard',
            error ? 'border-danger' : 'border-structural focus:border-signal',
            canReveal && 'pr-14',
          )}
        />
        {canReveal ? (
          <button
            type="button"
            aria-label={`${visible ? 'Hide' : 'Show'} ${label.toLowerCase()}`}
            aria-controls={id}
            disabled={props.disabled}
            onClick={() => setVisible((current) => !current)}
            className={cn(
              'absolute inset-y-0 right-0 flex w-11 items-center',
              'justify-center text-secondary hover:text-primary',
              'cursor-pointer disabled:cursor-default disabled:opacity-40',
              'focus-visible:outline-2 focus-visible:outline-signal',
              'focus-visible:-outline-offset-2',
            )}
          >
            <svg
              aria-hidden="true"
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z" />
              <circle cx="12" cy="12" r="3" />
              {visible ? <path d="m3 3 18 18" /> : null}
            </svg>
          </button>
        ) : null}
      </div>

      {children}

      {/* The note swaps between hint and error, so it animates instead of
          snapping the layout open under the control. */}
      <AnimatePresence initial={false} mode="wait">
        {note ? (
          <motion.p
            key={note}
            id={noteId}
            role={error ? 'alert' : undefined}
            initial={reduceMotion ? false : { opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={reduceMotion ? undefined : { opacity: 0, height: 0 }}
            transition={{ duration: reduceMotion ? 0 : 0.18 }}
            className={cn(
              'overflow-hidden text-body-s',
              error ? 'text-danger' : 'text-muted',
            )}
          >
            {note}
          </motion.p>
        ) : null}
      </AnimatePresence>
    </div>
  )
}
