/**
 * What a guarded screen shows when the backend does not answer.
 */

// --- IMPORTS ---
import { useTranslation } from 'react-i18next'
import { AuthShell } from '@/app/auth/AuthShell'
import { Button } from '@/components/ui/Button'
import { loadSession } from '@/logic/session/session'

// --- CODE ---
/**
 * Says the session could not be checked, and offers to try again.
 *
 * @returns The screen.
 */
export function SessionUnavailable() {
  const { t } = useTranslation()

  return (
    <AuthShell
      eyebrow="Unavailable"
      title="We could not reach Ecoa."
      lede="Your session could not be checked. This is on our side."
      footer="Nothing you did was lost."
    >
      <Button size="lg" onClick={() => void loadSession()}>
        {t('Try again')}
      </Button>
    </AuthShell>
  )
}
