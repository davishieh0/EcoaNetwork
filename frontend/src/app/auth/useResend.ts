/**
 * Asking for a new code, with a wait between two asks.
 */

// --- IMPORTS ---
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { RESEND_WAIT } from '@/app/auth/rules'
import { useCooldown } from '@/app/auth/useCooldown'
import type { Submit } from '@/app/forms/useSubmit.t'

// --- GLOBALS ---
const SENT = 'A new code is on its way. The previous one no longer works.'

// --- CODE ---
/**
 * Wraps the runner that sends a new code with its wait and its notice.
 *
 * @param send - The runner that asks the backend for a new code.
 * @param waiting - Whether a code was just sent on the way here.
 * @returns The ask, the button's state and label, and what to say above.
 */
export function useResend(send: Submit<[email: string]>, waiting = false) {
  const { t } = useTranslation()
  const cooldown = useCooldown(RESEND_WAIT, waiting)
  const [sent, setSent] = useState(false)

  /**
   * Asks for a new code, then waits before the next ask.
   *
   * @param email - The address to send it to.
   * @returns Nothing.
   */
  const ask = async (email: string) => {
    if (await send.run(email)) {
      setSent(true)
      cooldown.start()
    }
  }

  return {
    ask,
    pending: send.pending,
    disabled: cooldown.left > 0,
    label:
      cooldown.left > 0
        ? t('Send a new code in {{n}}s', { n: cooldown.left })
        : t('Send a new code'),
    notice: sent && !send.message ? t(SENT) : '',
    message: send.message,
  }
}
