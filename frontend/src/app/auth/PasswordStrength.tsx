/**
 * The password policy, ticked off as you type.
 */

// --- IMPORTS ---
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { PasswordStrengthProps } from '@/app/auth/PasswordStrength.t'
import { PASSWORD_RULES } from '@/app/auth/rules'
import { cn } from '@/components/ui/cn'

// --- CODE ---
/**
 * Draws a bar that fills with each rule met, and lists the rules with the
 * met ones ticked.
 *
 * @param props - The password being judged.
 * @returns The meter, or nothing while the field is empty.
 */
export function PasswordStrength({ password }: PasswordStrengthProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()
  const met = PASSWORD_RULES.map((rule) => rule.test.test(password))
  const score = met.filter(Boolean).length

  return (
    <AnimatePresence initial={false}>
      {password ? (
        <motion.div
          initial={reduceMotion ? false : { opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={reduceMotion ? undefined : { opacity: 0, height: 0 }}
          transition={{ duration: reduceMotion ? 0 : 0.22 }}
          className="flex flex-col gap-3 overflow-hidden"
        >
          <span aria-hidden="true" className="flex gap-1">
            {PASSWORD_RULES.map((rule, step) => (
              <span
                key={rule.label}
                className={cn(
                  'h-1 flex-1 transition-colors duration-interface',
                  'ease-standard',
                  step < score
                    ? score === PASSWORD_RULES.length
                      ? 'bg-valid'
                      : 'bg-primary'
                    : 'bg-quiet',
                )}
              />
            ))}
          </span>

          <ul className="flex flex-col gap-1">
            {PASSWORD_RULES.map((rule, index) => (
              <li
                key={rule.label}
                className={cn(
                  'flex items-center gap-2 text-body-s',
                  'transition-colors duration-interface ease-standard',
                  met[index] ? 'text-valid' : 'text-muted',
                )}
              >
                <span
                  aria-hidden="true"
                  className="relative size-3 shrink-0 border border-current"
                >
                  <AnimatePresence initial={false}>
                    {met[index] ? (
                      <motion.span
                        className="absolute inset-0.5 bg-current"
                        initial={reduceMotion ? false : { scale: 0 }}
                        animate={{ scale: 1 }}
                        exit={reduceMotion ? undefined : { scale: 0 }}
                        transition={{ duration: reduceMotion ? 0 : 0.16 }}
                      />
                    ) : null}
                  </AnimatePresence>
                </span>
                {t(rule.label)}
                <span className="sr-only">
                  {met[index] ? t(', done') : t(', missing')}
                </span>
              </li>
            ))}
          </ul>
        </motion.div>
      ) : null}
    </AnimatePresence>
  )
}
