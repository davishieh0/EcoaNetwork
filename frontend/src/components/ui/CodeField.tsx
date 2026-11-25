/**
 * A one-time code, one box per digit.
 */

// --- IMPORTS ---
import { useId, useRef } from 'react'
import type { ChangeEvent, ClipboardEvent, KeyboardEvent } from 'react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { CodeFieldProps } from '@/components/ui/CodeField.t'
import { cn } from '@/components/ui/cn'
import { known } from '@/i18n/i18n'

// --- CODE ---
/**
 * Reads what a box received. When the box kept its digit, because a click
 * undid the select on focus, the one new character sits beside the old.
 *
 * @param digit - The digit the box held.
 * @param event - The change.
 * @returns What was typed, without the digit already there.
 */
function typedInto(digit: string, event: ChangeEvent<HTMLInputElement>) {
  const typed = event.target.value
  if (!digit || typed.length !== 2) return typed

  return typed.startsWith(digit) ? typed.slice(1) : typed.slice(0, 1)
}

/**
 * Renders the boxes. Typing moves forward, Backspace moves back, and pasting
 * a whole code fills every box at once.
 *
 * @param props - Label, the code so far and its setter, error or hint.
 * @returns The field.
 */
export function CodeField({
  label,
  value,
  onValueChange,
  length = 6,
  error,
  hint,
  autoFocus,
}: CodeFieldProps) {
  const { t } = useTranslation()
  const id = useId()
  const reduceMotion = useReducedMotion()
  const boxes = useRef<(HTMLInputElement | null)[]>([])
  const note = error ?? hint
  const noteId = note ? `${id}-note` : undefined

  /**
   * Moves the caret to a box.
   *
   * @param index - The box, clamped to the field.
   * @returns Nothing.
   */
  const focus = (index: number) => {
    boxes.current[Math.max(0, Math.min(index, length - 1))]?.focus()
  }

  /**
   * Writes digits from a box onward, then moves past them. A box past the
   * end of the code continues it, so the code never has a gap.
   *
   * @param index - The box the digits start in.
   * @param typed - What was typed or pasted; non-digits are dropped.
   * @returns Nothing.
   */
  const write = (index: number, typed: string) => {
    const digits = typed.replace(/\D/g, '')
    if (!digits) return

    const at = Math.min(index, value.length)
    const next = (
      value.slice(0, at) +
      digits +
      value.slice(at + digits.length)
    ).slice(0, length)

    onValueChange(next)
    focus(at + digits.length)
  }

  /**
   * Handles Backspace and the arrow keys.
   *
   * @param index - The box with the caret.
   * @param event - The key pressed.
   * @returns Nothing.
   */
  const onKeyDown = (index: number, event: KeyboardEvent) => {
    if (event.key === 'Backspace') {
      event.preventDefault()

      // an empty box clears the one before it, like a single text field
      const at = value[index] ? index : index - 1
      if (at < 0) return

      onValueChange(value.slice(0, at))
      focus(at)
      return
    }

    if (event.key === 'ArrowLeft') focus(index - 1)
    if (event.key === 'ArrowRight') focus(index + 1)
  }

  /**
   * Fills the boxes from a pasted code.
   *
   * @param index - The box the paste landed in.
   * @param event - The paste.
   * @returns Nothing.
   */
  const onPaste = (index: number, event: ClipboardEvent) => {
    event.preventDefault()
    write(index, event.clipboardData.getData('text'))
  }

  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={`${id}-0`} className="meta text-secondary">
        {known(t, label)}
      </label>

      <div
        role="group"
        aria-label={known(t, label)}
        aria-describedby={noteId}
        className="flex gap-2"
      >
        {Array.from({ length }, (_, index) => {
          const digit = value[index] ?? ''

          return (
            <div key={index} className="relative flex-1">
              <input
                ref={(element) => {
                  boxes.current[index] = element
                }}
                id={`${id}-${index}`}
                value={digit}
                inputMode="numeric"
                autoComplete={index === 0 ? 'one-time-code' : 'off'}
                autoFocus={autoFocus && index === 0}
                aria-label={t('Digit {{n}} of {{total}}', {
                  n: index + 1,
                  total: length,
                })}
                aria-invalid={error ? true : undefined}
                onChange={(event) => write(index, typedInto(digit, event))}
                onKeyDown={(event) => onKeyDown(index, event)}
                onPaste={(event) => onPaste(index, event)}
                onFocus={(event) => event.target.select()}
                className={cn(
                  'h-14 w-full min-w-0 rounded-none border bg-surface',
                  'text-center font-mono text-h4 text-transparent',
                  'caret-signal transition-colors duration-interface',
                  'ease-standard',
                  error
                    ? 'border-danger'
                    : digit
                      ? 'border-structural focus:border-signal'
                      : 'border-line focus:border-signal',
                )}
              />

              {/* The digit is drawn over the box so it can land with a
                  small drop instead of appearing flat. */}
              <AnimatePresence>
                {digit ? (
                  <motion.span
                    key={digit}
                    aria-hidden="true"
                    initial={reduceMotion ? false : { opacity: 0, y: -8 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={reduceMotion ? undefined : { opacity: 0, y: 6 }}
                    transition={{ duration: reduceMotion ? 0 : 0.16 }}
                    className={cn(
                      'pointer-events-none absolute inset-0 flex',
                      'items-center justify-center font-mono text-h4',
                      'text-primary',
                    )}
                  >
                    {digit}
                  </motion.span>
                ) : null}
              </AnimatePresence>
            </div>
          )
        })}
      </div>

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
            {known(t, note)}
          </motion.p>
        ) : null}
      </AnimatePresence>
    </div>
  )
}
