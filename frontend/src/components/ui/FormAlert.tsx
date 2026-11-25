/**
 * A message about the whole form, not one field.
 */

// --- IMPORTS ---
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { FormAlertProps, FormAlertTone } from '@/components/ui/FormAlert.t'
import { cn } from '@/components/ui/cn'
import { known } from '@/i18n/i18n'

// --- GLOBALS ---
// the colour writes the rule and the text; the panel itself stays neutral
const TONES: Record<FormAlertTone, string> = {
  danger: 'border-danger text-danger',
  valid: 'border-valid text-valid',
}

// --- CODE ---
/**
 * Opens a ruled note above the form, and closes it when cleared.
 *
 * @param props - The message and its tone.
 * @returns The alert, or nothing.
 */
export function FormAlert({ children, tone = 'danger' }: FormAlertProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  return (
    <AnimatePresence initial={false}>
      {children ? (
        <motion.div
          role={tone === 'danger' ? 'alert' : 'status'}
          initial={reduceMotion ? false : { opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={reduceMotion ? undefined : { opacity: 0, height: 0 }}
          transition={{ duration: reduceMotion ? 0 : 0.22 }}
          className="overflow-hidden"
        >
          <p
            className={cn(
              'border-l-2 bg-surface px-4 py-3 text-body-s text-pretty',
              TONES[tone],
            )}
          >
            {typeof children === 'string' ? known(t, children) : children}
          </p>
        </motion.div>
      ) : null}
    </AnimatePresence>
  )
}
