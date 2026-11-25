/**
 * Deleting the account, behind a second look and the password.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import * as Dialog from '@radix-ui/react-dialog'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { DeleteAccountProps } from '@/app/account/DeleteAccount.t'
import { placeFailure } from '@/app/forms/placeFailure'
import { Button } from '@/components/ui/Button'
import { Field } from '@/components/ui/Field'
import { FormAlert } from '@/components/ui/FormAlert'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
const EASE = [0.2, 0.8, 0.2, 1] as const

// the destructive action wears danger, the one place in the app that does
const DANGER = cn(
  'border-danger text-danger',
  'hover:bg-danger hover:border-danger hover:text-on-danger',
)

// --- CODE ---
/**
 * Renders the button that opens the dialog, and the dialog itself.
 *
 * @param props - The handle, and the delete runner.
 * @returns The trigger and the dialog.
 */
export function DeleteAccount({ handle, remove }: DeleteAccountProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()
  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  /**
   * Checks a password was typed, then deletes.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()

    if (!password) {
      setError('Enter your password to confirm.')
      return
    }

    setError('')
    void remove.run(password)
  }

  /**
   * Opens or closes the dialog, starting clean each time.
   *
   * @param next - Whether it should be open.
   * @returns Nothing.
   */
  const toggle = (next: boolean) => {
    // never close while the deletion is under way
    if (remove.pending) return

    setOpen(next)
    setPassword('')
    setError('')
    remove.reset()
  }

  // a wrong password is said under the field, anything else above the form
  const shown = placeFailure({ password: error }, remove, {
    invalid_credentials_error: 'password',
  })

  const panel = reduceMotion ? { duration: 0 } : { duration: 0.32, ease: EASE }

  return (
    <Dialog.Root open={open} onOpenChange={toggle}>
      <Dialog.Trigger asChild>
        <Button variant="secondary" className={DANGER}>
          {t('Delete account')}
        </Button>
      </Dialog.Trigger>

      <AnimatePresence>
        {open ? (
          <Dialog.Portal forceMount>
            <Dialog.Overlay asChild forceMount>
              <motion.div
                className="fixed inset-0 z-70 bg-scrim"
                initial={reduceMotion ? false : { opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={reduceMotion ? undefined : { opacity: 0 }}
                transition={{ duration: reduceMotion ? 0 : 0.2 }}
              />
            </Dialog.Overlay>

            <Dialog.Content asChild forceMount>
              <motion.div
                className={cn(
                  'fixed top-1/2 left-1/2 z-70 w-[calc(100%-2rem)]',
                  'max-w-md -translate-x-1/2 -translate-y-1/2',
                  'flex flex-col gap-6 border border-structural',
                  'bg-surface-raised p-8',
                )}
                initial={reduceMotion ? false : { opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
                exit={reduceMotion ? undefined : { opacity: 0, y: 8 }}
                transition={panel}
              >
                <div className="flex flex-col gap-3">
                  <Dialog.Title
                    className={cn(
                      'font-display text-h3 font-semibold tracking-heading',
                    )}
                  >
                    {t('Delete @{{handle}}?', { handle })}
                  </Dialog.Title>

                  <Dialog.Description className="text-body-m text-secondary">
                    {t(
                      'This cannot be undone. The account, its handle and ' +
                        'every session go now, and the handle becomes free ' +
                        'for anyone.',
                    )}
                  </Dialog.Description>
                </div>

                <form
                  onSubmit={onSubmit}
                  className="flex flex-col gap-6"
                  noValidate
                >
                  <FormAlert>{shown.alert}</FormAlert>

                  <Field
                    label="Password"
                    name="password"
                    type="password"
                    value={password}
                    onChange={(event) => {
                      setPassword(event.target.value)
                      remove.reset()
                    }}
                    autoComplete="current-password"
                    autoFocus
                    error={shown.errors.password}
                  />

                  <div className="flex flex-col-reverse gap-3 sm:flex-row">
                    <Dialog.Close asChild>
                      <Button
                        type="button"
                        variant="ghost"
                        className="sm:flex-1"
                        disabled={remove.pending}
                      >
                        {t('Keep account')}
                      </Button>
                    </Dialog.Close>

                    <Button
                      type="submit"
                      variant="secondary"
                      className={cn(DANGER, 'sm:flex-1')}
                      pending={remove.pending}
                    >
                      {remove.pending ? t('Deleting') : t('Delete for good')}
                    </Button>
                  </div>
                </form>
              </motion.div>
            </Dialog.Content>
          </Dialog.Portal>
        ) : null}
      </AnimatePresence>
    </Dialog.Root>
  )
}
