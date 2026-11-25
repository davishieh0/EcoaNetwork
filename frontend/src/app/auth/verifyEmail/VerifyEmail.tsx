/**
 * Confirming the email with the code it was sent.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'

import { AuthShell } from '@/app/auth/AuthShell'
import { CODE_LENGTH, EMAIL, INVALID } from '@/app/auth/rules'
import { useResend } from '@/app/auth/useResend'
import { useVerifyData } from '@/app/auth/verifyEmail/useVerifyData'
import type { VerifyErrors } from '@/app/auth/verifyEmail/VerifyEmail.t'
import { placeFailure } from '@/app/forms/placeFailure'
import { Button } from '@/components/ui/Button'
import { CodeField } from '@/components/ui/CodeField'
import { EmailField } from '@/components/ui/EmailField'
import { FormAlert } from '@/components/ui/FormAlert'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const TITLE_UNVERIFIED = 'Confirm your email first.'

// failures that are about the code itself, shown under it
const FIELD_OF = {
  invalid_code_error: 'code',
  expired_code_error: 'code',
} as const

// --- CODE ---
/**
 * Builds the screen that confirms an email.
 *
 * @returns The verification screen.
 */
export function VerifyEmail() {
  const { t } = useTranslation()
  const { confirm, resend, state } = useVerifyData()
  // a sign-in that just sent a code counts as the first ask
  const newCode = useResend(resend, Boolean(state.sent))
  const [email, setEmail] = useState(state.email ?? '')
  const [code, setCode] = useState('')
  const [errors, setErrors] = useState<VerifyErrors>({})

  // a direct visit has no address carried over: ask for it
  const knownEmail = Boolean(state.email)

  /**
   * Checks the address and the code, then confirms.
   *
   * @param value - The code to send, the latest typed.
   * @returns Nothing.
   */
  const send = (value: string) => {
    const next: VerifyErrors = {}
    if (!EMAIL.test(email)) next.email = INVALID.email
    if (value.length < CODE_LENGTH) next.code = INVALID.code

    setErrors(next)
    if (Object.keys(next).length === 0) void confirm.run(email, value)
  }

  /**
   * Submits the form.
   *
   * @param event - The form submission.
   * @returns Nothing.
   */
  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    send(code)
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

    confirm.reset()
    setCode('')
    void newCode.ask(email)
  }

  const shown = placeFailure(errors, confirm, FIELD_OF)

  return (
    <AuthShell
      eyebrow="Verify email"
      title={state.unverified ? TITLE_UNVERIFIED : 'Check your inbox.'}
      lede={
        !knownEmail
          ? t(
              'Enter the address you signed up with and the {{length}}-' +
                'digit code we sent to it.',
              { length: CODE_LENGTH },
            )
          : state.unverified && !state.sent
            ? t(
                '{{email}} is not confirmed yet. Ask for a new code below, ' +
                  'then enter it here.',
                { email },
              )
            : t(
                'We sent a {{length}}-digit code to {{email}}. Enter it to ' +
                  'finish creating your account.',
                { length: CODE_LENGTH, email },
              )
      }
      footer={
        <>
          {t('Wrong address?')}{' '}
          <Link to={ROUTES.register} className="text-primary underline">
            {t('Start again')}
          </Link>
        </>
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

        <CodeField
          label="Confirmation code"
          value={code}
          onValueChange={(value) => {
            setCode(value)
            confirm.reset()
            // a complete code goes out on its own
            if (value.length === CODE_LENGTH) send(value)
          }}
          length={CODE_LENGTH}
          error={shown.errors.code}
          autoFocus={knownEmail}
        />

        <div className="flex flex-col gap-3">
          <Button type="submit" size="lg" pending={confirm.pending}>
            {confirm.pending ? t('Confirming') : t('Confirm email')}
          </Button>

          <Button
            type="button"
            variant="ghost"
            pending={newCode.pending}
            disabled={newCode.disabled}
            onClick={onResend}
          >
            {newCode.label}
          </Button>
        </div>

        <p className="text-body-s text-muted text-pretty">
          {t(
            'Nothing arrived? Check spam before asking for another. A new ' +
              'code retires the one before it.',
          )}
        </p>
      </form>
    </AuthShell>
  )
}
