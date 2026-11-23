/**
 * Setting a new password with the code from the email.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router'

import { AuthShell } from '@/app/auth/AuthShell'
import { PasswordStrength } from '@/app/auth/PasswordStrength'
import type { ResetPasswordErrors } from '@/app/auth/resetPassword/ResetPassword.t'
import {
  CODE_LENGTH,
  EMAIL,
  INVALID,
  passwordMeetsPolicy,
} from '@/app/auth/rules'
import { useResend } from '@/app/auth/useResend'
import { useResetPasswordData } from '@/app/auth/resetPassword/useResetPasswordData'
import { placeFailure } from '@/app/forms/placeFailure'
import type { FieldOf } from '@/app/forms/placeFailure.t'
import { Button } from '@/components/ui/Button'
import { CodeField } from '@/components/ui/CodeField'
import { EmailField } from '@/components/ui/EmailField'
import { Field } from '@/components/ui/Field'
import { FormAlert } from '@/components/ui/FormAlert'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
// failures that belong to one field, shown under it instead of above
const FIELD_OF: FieldOf<ResetPasswordErrors> = {
  invalid_code_error: 'code',
  expired_code_error: 'code',
  invalid_password_error: 'password',
}

// --- CODE ---
/**
 * Builds the screen that sets a new password.
 *
 * @returns The reset screen.
 */
export function ResetPassword() {
  const { change, resend, state } = useResetPasswordData()
  const newCode = useResend(resend)
  const [email, setEmail] = useState(state.email ?? '')
  const [code, setCode] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [errors, setErrors] = useState<ResetPasswordErrors>({})

  const knownEmail = Boolean(state.email)

  /**
   * Validates every field, then sets the new password.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()

    const next: ResetPasswordErrors = {}
    if (!EMAIL.test(email)) next.email = INVALID.email
    if (code.length < CODE_LENGTH) next.code = INVALID.code
    if (!passwordMeetsPolicy(password)) next.password = INVALID.password
    if (confirm !== password) next.confirm = INVALID.confirm

    setErrors(next)
    if (Object.keys(next).length === 0) {
      void change.run(email, code, password)
    }
  }

  /**
   * Asks for a new code, which retires the previous one.
   *
   * @returns Nothing.
   */
  const onResend = () => {
    if (!EMAIL.test(email)) {
      setErrors({ email: INVALID.email })
      return
    }

    change.reset()
    setCode('')
    void newCode.ask(email)
  }

  const shown = placeFailure(errors, change, FIELD_OF)

  return (
    <AuthShell
      eyebrow="Reset password"
      title="Set a new password."
      lede={
        knownEmail
          ? `Enter the ${CODE_LENGTH}-digit code we sent to ${email}, then ` +
            'choose a password you have not used here before.'
          : 'Enter your address, the code we sent to it, and a new password.'
      }
      footer={
        <Link to={ROUTES.login} className="text-primary underline">
          Back to sign in
        </Link>
      }
    >
      <form onSubmit={onSubmit} className="flex flex-col gap-6" noValidate>
        <FormAlert tone="valid">{newCode.notice}</FormAlert>
        <FormAlert>{shown.alert || newCode.message}</FormAlert>

        {knownEmail ? null : (
          <EmailField
            label="Email"
            name="email"
            value={email}
            onValueChange={setEmail}
            autoComplete="email"
            placeholder="you@example.com"
            error={shown.errors.email}
          />
        )}

        <div className="flex flex-col gap-3">
          <CodeField
            label="Reset code"
            value={code}
            onValueChange={(value) => {
              setCode(value)
              change.reset()
            }}
            length={CODE_LENGTH}
            error={shown.errors.code}
            autoFocus={knownEmail}
          />

          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="self-start"
            pending={newCode.pending}
            disabled={newCode.disabled}
            onClick={onResend}
          >
            {newCode.label}
          </Button>
        </div>

        <div className="flex flex-col gap-3">
          <Field
            label="New password"
            name="password"
            type="password"
            revealPassword
            value={password}
            onChange={(event) => {
              setPassword(event.target.value)
              change.reset()
            }}
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
          onChange={(event) => setConfirm(event.target.value)}
          autoComplete="new-password"
          error={shown.errors.confirm}
        />

        <Button type="submit" size="lg" pending={change.pending}>
          {change.pending ? 'Changing password' : 'Change password'}
        </Button>
      </form>
    </AuthShell>
  )
}
