/**
 * Asking for a code that lets you set a new password.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { AuthShell } from '@/app/auth/AuthShell'
import { CODE_LENGTH, EMAIL, INVALID } from '@/app/auth/rules'
import { useForgotPasswordData } from '@/app/auth/forgotPassword/useForgotPasswordData'
import { Button } from '@/components/ui/Button'
import { EmailField } from '@/components/ui/EmailField'
import { FormAlert } from '@/components/ui/FormAlert'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const LEDE =
  `Tell us the address on the account and we will send a ${CODE_LENGTH}-` +
  'digit code that lets you set a new password.'

// --- CODE ---
/**
 * Builds the screen that requests a password reset code.
 *
 * @returns The forgot-password screen.
 */
export function ForgotPassword() {
  const { t } = useTranslation()
  const { submit, state } = useForgotPasswordData()
  const [email, setEmail] = useState(state.email ?? '')
  const [error, setError] = useState('')

  /**
   * Validates the address, then asks for the code.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()

    if (!EMAIL.test(email)) {
      setError(INVALID.email)
      return
    }

    setError('')
    void submit.run(email)
  }

  return (
    <AuthShell
      eyebrow="Reset password"
      title="Forgot your password?"
      lede={LEDE}
      footer={
        <>
          {t('Remembered it?')}{' '}
          <Link to={ROUTES.login} className="text-primary underline">
            {t('Sign in')}
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-6" noValidate>
        <FormAlert>{submit.message}</FormAlert>

        <EmailField
          label="Email"
          name="email"
          value={email}
          onValueChange={(value) => {
            setEmail(value)
            submit.reset()
          }}
          autoComplete="email"
          placeholder="you@example.com"
          error={error}
        />

        <Button type="submit" size="lg" pending={submit.pending}>
          {submit.pending ? t('Sending code') : t('Send reset code')}
        </Button>
      </form>
    </AuthShell>
  )
}
