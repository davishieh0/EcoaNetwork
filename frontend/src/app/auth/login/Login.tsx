/**
 * Signing in to the platform.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { AuthShell } from '@/app/auth/AuthShell'
import type { LoginErrors } from '@/app/auth/login/Login.t'
import { EMAIL, INVALID } from '@/app/auth/rules'
import { useLoginData } from '@/app/auth/login/useLoginData'
import { routeTo } from '@/app/navigation/navigation'
import { Button } from '@/components/ui/Button'
import { EmailField } from '@/components/ui/EmailField'
import { Field } from '@/components/ui/Field'
import { FormAlert } from '@/components/ui/FormAlert'
import { cn } from '@/components/ui/cn'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const LEDE = 'Sign in to write, reply and manage what is yours.'

const NOTICES = {
  verified: 'Email confirmed. Sign in to finish setting up.',
  reset: 'Password changed. Sign in with the new one.',
  logout: 'You signed out.',
  deleted: 'Your account was deleted, with its handle and every session.',
  none: '',
} as const

// --- CODE ---
/**
 * Builds the screen that signs in to the platform.
 *
 * @returns The login screen.
 */
export function Login() {
  const { t } = useTranslation()
  const { submit, state } = useLoginData()
  const [email, setEmail] = useState(state.email ?? '')
  const [password, setPassword] = useState('')
  const [errors, setErrors] = useState<LoginErrors>({})

  /**
   * Validates both fields, then signs in.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()

    const next: LoginErrors = {}
    if (!EMAIL.test(email)) next.email = INVALID.email
    if (!password) next.password = 'Enter your password.'

    setErrors(next)
    if (Object.keys(next).length === 0) void submit.run(email, password)
  }

  // what the previous screen finished, said once above the form
  const notice =
    NOTICES[
      state.verified
        ? 'verified'
        : state.reset
          ? 'reset'
          : (state.ended ?? 'none')
    ]

  return (
    <AuthShell
      eyebrow="Sign in"
      title="Welcome back."
      lede={LEDE}
      footer={
        <>
          {t('No account yet?')}{' '}
          <Link to={ROUTES.register} className="text-primary underline">
            {t('Create one')}
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-6" noValidate>
        <FormAlert tone="valid">{submit.message ? '' : notice}</FormAlert>
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
          error={errors.email}
        />

        <Field
          label="Password"
          name="password"
          type="password"
          revealPassword
          value={password}
          onChange={(event) => {
            setPassword(event.target.value)
            submit.reset()
          }}
          autoComplete="current-password"
          error={errors.password}
        />

        <Link
          {...routeTo('forgotPassword', { email })}
          className={cn(
            'meta self-start text-muted transition-colors',
            'duration-interface hover:text-primary',
          )}
        >
          {t('Forgot your password?')}
        </Link>

        <Button type="submit" size="lg" pending={submit.pending}>
          {submit.pending ? t('Signing in') : t('Sign in')}
        </Button>
      </form>
    </AuthShell>
  )
}
