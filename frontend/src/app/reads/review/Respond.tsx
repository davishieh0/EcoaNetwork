/**
 * Responding: to the review, or to a response in its conversation.
 */

// --- IMPORTS ---
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { RespondProps } from '@/app/reads/review/Respond.t'
import { Button } from '@/components/ui/Button'
import { cn } from '@/components/ui/cn'
import { BODY_MAX, characters, checkResponse } from '@/logic/protocol/review'

// --- CODE ---
/**
 * Renders the response box, saying what it answers.
 *
 * @param props - Who it answers, whether it is on its way, and the
 * handlers.
 * @returns The form.
 */
export function Respond({
  answering,
  busy,
  onRespond,
  onCancel,
}: RespondProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  const [tried, setTried] = useState(false)
  const box = useRef<HTMLTextAreaElement>(null)

  // choosing a response to answer brings the box to it
  useEffect(() => {
    if (!answering) return
    box.current?.focus({ preventScroll: true })
    box.current?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }, [answering])

  const checked = checkResponse(text)
  const problem = tried && !checked.ok ? checked.problem : null

  return (
    <form
      aria-label={t('Respond')}
      className="mt-6 flex flex-col gap-3"
      onSubmit={async (event) => {
        event.preventDefault()
        setTried(true)
        if (!checked.ok) return
        if (await onRespond(checked.body)) {
          setText('')
          setTried(false)
        }
      }}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="meta text-muted">
          {answering
            ? `↳ ${t('Replying to {{who}}', { who: answering })}`
            : t('Respond to the review')}
          {' · '}
          {characters(text)}/{BODY_MAX}
        </span>
        {answering ? (
          <button
            type="button"
            onClick={onCancel}
            className="meta cursor-pointer text-muted hover:text-primary"
          >
            {t('Reply to the review instead')}
          </button>
        ) : null}
      </div>
      <textarea
        ref={box}
        value={text}
        onChange={(event) => setText(event.target.value)}
        rows={3}
        placeholder={t('Your response is signed by you, and public.')}
        className={cn(
          'w-full border border-structural bg-surface p-3',
          'transition-colors duration-interface focus:border-strong',
          problem ? 'border-danger' : '',
        )}
      />
      {problem ? <p className="text-body-s text-danger">{t(problem)}</p> : null}
      <div className="flex justify-end">
        <Button type="submit" pending={busy}>
          {busy ? t('Signing and sending') : t('Respond')}
        </Button>
      </div>
    </form>
  )
}
