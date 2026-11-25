/**
 * Choosing the handle, for an account that does not have one yet.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { AuthShell } from '@/app/auth/AuthShell'
import { HANDLE, INVALID } from '@/app/auth/rules'
import { useChooseHandleData } from '@/app/auth/chooseHandle/useChooseHandleData'
import { placeFailure } from '@/app/forms/placeFailure'
import { Button } from '@/components/ui/Button'
import { Field } from '@/components/ui/Field'
import { FormAlert } from '@/components/ui/FormAlert'

// --- CODE ---
/**
 * Builds the screen that completes an account with its handle.
 *
 * @returns The welcome screen.
 */
export function ChooseHandle() {
  const { t } = useTranslation()
  const { submit, email } = useChooseHandleData()
  const [handle, setHandle] = useState('')
  const [error, setError] = useState('')

  /**
   * Validates the handle, then claims it.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()

    if (!HANDLE.test(handle)) {
      setError(INVALID.handle)
      return
    }

    setError('')
    void submit.run(handle)
  }

  // a taken handle is said under the field, anything else above the form
  const shown = placeFailure({ handle: error }, submit, {
    handle_taken_error: 'handle',
  })

  return (
    <AuthShell
      eyebrow="One last step"
      title="Choose your handle."
      lede={t(
        'You are signed in as {{email}}. Pick the name that appears on ' +
          'everything you publish. It cannot be changed later.',
        { email },
      )}
      footer="Reading Ecoa never needs a handle. Writing does."
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-6" noValidate>
        <FormAlert>{shown.alert}</FormAlert>

        <Field
          label="Handle"
          name="handle"
          value={handle}
          onChange={(event) => {
            setHandle(event.target.value.trim().toLowerCase())
            submit.reset()
          }}
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          autoFocus
          placeholder="joana_k"
          error={shown.errors.handle}
          hint={INVALID.handle}
        />

        <Button type="submit" size="lg" pending={submit.pending}>
          {submit.pending ? t('Claiming handle') : t('Claim handle')}
        </Button>
      </form>
    </AuthShell>
  )
}
