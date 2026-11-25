/**
 * Editing a review: a new version, signed by its author, that replaces the
 * current one without erasing it.
 */

// --- IMPORTS ---
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { EditReviewProps } from '@/app/reads/review/EditReview.t'
import { Button } from '@/components/ui/Button'
import { cn } from '@/components/ui/cn'
import {
  BODY_MAX,
  characters,
  checkVersion,
  TITLE_MAX,
} from '@/logic/protocol/review'
import type { Rating } from '@/logic/protocol/review.t'

// --- GLOBALS ---
const RATINGS: Rating[] = [1, 2, 3, 4, 5]

const FIELD = cn(
  'w-full border border-structural bg-surface p-3',
  'transition-colors duration-interface focus:border-strong',
)

// --- CODE ---
/**
 * Renders the edit form, filled with what the current version says.
 *
 * @param props - The current text, whether it is on its way, and the
 * handlers.
 * @returns The form.
 */
export function EditReview({
  current,
  busy,
  onSave,
  onCancel,
}: EditReviewProps) {
  const { t } = useTranslation()
  const [rating, setRating] = useState<Rating | null>(
    RATINGS.includes(current.rating as Rating)
      ? (current.rating as Rating)
      : null,
  )
  const [title, setTitle] = useState(current.title ?? '')
  const [body, setBody] = useState(current.body)
  const [tried, setTried] = useState(false)

  const lang = current.lang ?? ''
  const checked = checkVersion({ rating, title, body, lang })
  const problems = tried && !checked.ok ? checked.problems : {}
  const unchanged =
    rating === current.rating &&
    title.trim() === (current.title ?? '') &&
    body.trim() === current.body

  return (
    <form
      aria-label={t('Edit your review')}
      className="flex flex-col gap-4 border border-structural bg-surface p-6"
      onSubmit={(event) => {
        event.preventDefault()
        setTried(true)
        if (checked.ok && !unchanged) void onSave(checked.text)
      }}
    >
      <div className="flex flex-col gap-1">
        <h2 className="font-display text-h4">{t('Edit your review')}</h2>
        <p className="text-body-s text-secondary">
          {t(
            'The edit is a new version, signed by you. The one before stays ' +
              'in the history, and votes stay with the version they were ' +
              'cast on.',
          )}
        </p>
      </div>

      <div
        role="radiogroup"
        aria-label={t('Rating, 1 worst to 5 best')}
        className="grid grid-cols-5 border border-line"
      >
        {RATINGS.map((value) => (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={rating === value}
            onClick={() => setRating(value)}
            className={cn(
              'min-h-11 cursor-pointer border-r border-line last:border-r-0',
              'font-display text-h4 tabular-nums transition-colors',
              rating === value
                ? 'bg-signal text-on-signal'
                : 'text-secondary hover:bg-surface-raised hover:text-primary',
            )}
          >
            {value}
          </button>
        ))}
      </div>
      {problems.rating ? (
        <p className="text-body-s text-danger">{t(problems.rating)}</p>
      ) : null}

      <label className="flex flex-col gap-1">
        <span className="meta text-muted">
          {t('Title')} · {characters(title)}/{TITLE_MAX}
        </span>
        <input
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          className={FIELD}
        />
        {problems.title ? (
          <span className="text-body-s text-danger">{t(problems.title)}</span>
        ) : null}
      </label>

      <label className="flex flex-col gap-1">
        <span className="meta text-muted">
          {t('What happened')} · {characters(body)}/{BODY_MAX}
        </span>
        <textarea
          value={body}
          onChange={(event) => setBody(event.target.value)}
          rows={5}
          className={FIELD}
        />
        {problems.body ? (
          <span className="text-body-s text-danger">{t(problems.body)}</span>
        ) : null}
      </label>

      <div className="flex flex-wrap justify-end gap-3">
        <Button variant="ghost" type="button" onClick={onCancel}>
          {t('Cancel')}
        </Button>
        <Button type="submit" pending={busy} disabled={unchanged}>
          {busy ? t('Signing and sending') : t('Publish the edit')}
        </Button>
      </div>
    </form>
  )
}
