/**
 * The actions at the foot of a review: its author edits or withdraws it,
 * anyone else endorses or disputes this version.
 */

// --- IMPORTS ---
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { ReviewActionsProps } from '@/app/reads/review/ReviewActions.t'
import { cn } from '@/components/ui/cn'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
const ACTION = cn(
  'meta cursor-pointer text-secondary transition-colors hover:text-primary',
  'disabled:cursor-not-allowed disabled:opacity-50',
)

// --- CODE ---
/**
 * Offers what this reader may do with the review.
 *
 * @param props - Who is reading, and the handlers.
 * @returns The actions, or a way to sign in.
 */
export function ReviewActions({
  canAct,
  sessionKnown,
  mine,
  busy,
  onEdit,
  onWithdraw,
  onVote,
  onLabel,
}: ReviewActionsProps) {
  const { t } = useTranslation()

  if (!sessionKnown) return null

  if (!canAct) {
    return (
      <Link to={ROUTES.login} className={ACTION}>
        {t('Sign in to vote or respond')}
      </Link>
    )
  }

  return (
    <span className="flex flex-wrap items-center gap-4">
      {mine ? (
        <>
          <button
            type="button"
            className={ACTION}
            disabled={busy}
            onClick={onEdit}
          >
            {t('Edit')}
          </button>
          <button
            type="button"
            className={cn(ACTION, 'hover:text-danger')}
            disabled={busy}
            onClick={onWithdraw}
          >
            {t('Withdraw')}
          </button>
        </>
      ) : (
        <>
          <button
            type="button"
            className={ACTION}
            disabled={busy}
            onClick={() => onVote('endorsed')}
          >
            ▲ {t('Endorse')}
          </button>
          <button
            type="button"
            className={ACTION}
            disabled={busy}
            onClick={() => onVote('disputed')}
          >
            ▼ {t('Dispute')}
          </button>
        </>
      )}
      {onLabel ? (
        <button
          type="button"
          className={cn(ACTION, 'hover:text-warning')}
          disabled={busy}
          onClick={onLabel}
        >
          {t('Label…')}
        </button>
      ) : null}
    </span>
  )
}
