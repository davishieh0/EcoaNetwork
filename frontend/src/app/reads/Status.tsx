/**
 * A list's status line: what it holds, or why it failed, crossfading as it
 * changes.
 */

// --- IMPORTS ---
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'
import type { StatusProps } from '@/app/reads/Status.t'
import { cn } from '@/components/ui/cn'
import { known } from '@/i18n/i18n'

// --- CODE ---
/**
 * Says where a list is; a failure is announced as an alert.
 *
 * @param props - The text, whether it is a failure, and extra classes.
 * @returns The line.
 */
export function Status({ text, failed = false, className }: StatusProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  return (
    <div className={cn('min-h-5', className)}>
      <AnimatePresence mode="wait" initial={false}>
        <motion.p
          key={text}
          role={failed ? 'alert' : 'status'}
          className={failed ? 'text-danger' : 'meta text-muted'}
          initial={reduceMotion ? false : { opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={reduceMotion ? undefined : { opacity: 0, y: -4 }}
          transition={{ duration: reduceMotion ? 0 : 0.18 }}
        >
          {known(t, text)}
        </motion.p>
      </AnimatePresence>
    </div>
  )
}
