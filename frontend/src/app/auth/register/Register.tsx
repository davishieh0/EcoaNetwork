/**
 * Creating a platform account.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { AuthShell } from '@/app/auth/AuthShell'
import { PasswordStrength } from '@/app/auth/PasswordStrength'
import type { RegisterErrors } from '@/app/auth/register/Register.t'
import { EMAIL, HANDLE, INVALID, passwordMeetsPolicy } from '@/app/auth/rules'
import { useRegisterData } from '@/app/auth/register/useRegisterData'
import { placeFailure } from '@/app/forms/placeFailure'
import type { FieldOf } from '@/app/forms/placeFailure.t'
import { Button } from '@/components/ui/Button'
import { EmailField } from '@/components/ui/EmailField'
import { Field } from '@/components/ui/Field'
import { FormAlert } from '@/components/ui/FormAlert'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const LEDE =
  'An account to sign in with. Reading Ecoa never needs one. This is for ' +
  'writing, replying and managing what is yours.'

// failures that belong to one field, shown under it instead of above the form
const FIELD_OF: FieldOf<RegisterErrors> = {
  handle_taken_error: 'handle',
  email_already_registered_error: 'email',
  invalid_password_error: 'password',
}

// --- CODE ---
/**
 * Builds the screen that creates a platform account.
 *
 * @returns The register screen.
 */
export function Register() {
  const { t } = useTranslation()
  const { submit } = useRegisterData()
  const [handle, setHandle] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [errors, setErrors] = useState<RegisterErrors>({})

  /**
   * Validates the fields, then creates the account.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()

    const next: RegisterErrors = {}
    if (!HANDLE.test(handle)) next.handle = INVALID.handle
    if (!EMAIL.test(email)) next.email = INVALID.email
    if (!passwordMeetsPolicy(password)) {
      next.password = INVALID.password
    }
    if (confirm !== password) next.confirm = INVALID.confirm

    setErrors(next)
    if (Object.keys(next).length === 0) {
      void submit.run(handle, email, password)
    }
  }

  // the backend's verdict, under the field it is about
  const shown = placeFailure(errors, submit, FIELD_OF)

  /**
   * Builds a change handler that also clears the backend's verdict.
   *
   * @param set - The field's setter.
   * @returns The handler.
   */
  const edit = (set: (value: string) => void) => (value: string) => {
    set(value)
    submit.reset()
  }

  return (
    <AuthShell
      eyebrow="Create account"
      title="Create your account."
      lede={LEDE}
      footer={
        <>
          {t('Already have an account?')}{' '}
          <Link to={ROUTES.login} className="text-primary underline">
            {t('Sign in')}
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-6" noValidate>
        <FormAlert>{shown.alert}</FormAlert>

        <Field
          label="Handle"
          name="handle"
          value={handle}
          onChange={(event) =>
            edit(setHandle)(event.target.value.trim().toLowerCase())
          }
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          placeholder="joana_k"
          error={shown.errors.handle}
          hint="What appears on everything you publish. Chosen once."
        />

        <EmailField
          label="Email"
          name="email"
          value={email}
          onValueChange={edit(setEmail)}
          autoComplete="email"
          placeholder="you@example.com"
          error={shown.errors.email}
        />

        <div className="flex flex-col gap-3">
          <Field
            label="Password"
            name="password"
            type="password"
            revealPassword
            value={password}
            onChange={(event) => edit(setPassword)(event.target.value)}
            autoComplete="new-password"
            error={shown.errors.password}
          />

          <PasswordStrength password={password} />
        </div>

        <Field
          label="Confirm password"
          name="confirm"
          type="password"
          revealPassword
          value={confirm}
          onChange={(event) => edit(setConfirm)(event.target.value)}
          autoComplete="new-password"
          error={shown.errors.confirm}
        />

        <Button type="submit" size="lg" pending={submit.pending}>
          {submit.pending ? t('Creating account') : t('Create account')}
        </Button>
      </form>
    </AuthShell>
  )
}
