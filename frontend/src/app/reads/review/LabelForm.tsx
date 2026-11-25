/**
 * Labeling: a moderator's signed verdict on a review or a response. It
 * deletes nothing; it changes what the readers who follow them see.
 */

// --- IMPORTS ---
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { LabelFormProps } from '@/app/reads/review/LabelForm.t'
import { Button } from '@/components/ui/Button'
import { cn } from '@/components/ui/cn'
import { characters } from '@/logic/protocol/review'

// --- GLOBALS ---
// the protocol's vocabulary, and what an index following you does with each
const LABELS = [
  { value: 'spam', effect: 'Hidden, unless a reader asks to see it' },
  { value: 'harassment', effect: 'Hidden, unless a reader asks to see it' },
  { value: 'impersonation', effect: 'Hidden, unless a reader asks to see it' },
  { value: 'off-topic', effect: 'Shown, marked' },
  { value: 'other', effect: 'Shown, marked' },
  { value: 'illegal', effect: 'Removed: never shown' },
  { value: 'personal-data', effect: 'Removed: never shown' },
] as const

const NOTE_MAX = 280

// --- CODE ---
/**
 * Asks for the label and an optional note, and signs it.
 *
 * @param props - What is labeled, whether it is on its way, and the
 * handlers.
 * @returns The form.
 */
export function LabelForm({
  what,
  busy,
  onLabel,
  onCancel,
}: LabelFormProps) {
  const { t } = useTranslation()
  const [label, setLabel] = useState<string>('spam')
  const [note, setNote] = useState('')
  const tooLong = characters(note) > NOTE_MAX

  return (
    <form
      aria-label={t('Label {{what}}', { what: t(what) })}
      className="flex flex-col gap-4 border border-warning bg-surface p-5"
      onSubmit={(event) => {
        event.preventDefault()
        if (!tooLong) void onLabel(label, note.trim() || null)
      }}
    >
      <div className="flex flex-col gap-1">
        <h2 className="font-display text-h4">
          {t('Label {{what}}', { what: t(what) })}
        </h2>
        <p className="text-body-s text-secondary">
          {t(
            'Signed by you. It counts for the readers who follow you; the ' +
              'event stays on the network either way.',
          )}
        </p>
      </div>

      <div role="radiogroup" aria-label={t('Label')} className="grid gap-2">
        {LABELS.map((option) => (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={label === option.value}
            onClick={() => setLabel(option.value)}
            className={cn(
              'flex cursor-pointer items-baseline justify-between gap-3',
              'border px-3 py-2 text-left transition-colors',
              label === option.value
                ? 'border-signal bg-signal-wash'
                : 'border-structural hover:border-strong',
            )}
          >
            <span className="font-mono text-mono-s">{option.value}</span>
            <span className="text-body-s text-muted">{t(option.effect)}</span>
          </button>
        ))}
      </div>

      <label className="flex flex-col gap-1">
        <span className="meta text-muted">
          {t('Note, optional')} · {characters(note)}/{NOTE_MAX}
        </span>
        <input
          value={note}
          onChange={(event) => setNote(event.target.value)}
          className={cn(
            'w-full border bg-canvas p-3 focus:border-strong',
            tooLong ? 'border-danger' : 'border-structural',
          )}
        />
      </label>

      <div className="flex flex-wrap justify-end gap-3">
        <Button variant="ghost" type="button" onClick={onCancel}>
          {t('Cancel')}
        </Button>
        <Button type="submit" pending={busy} disabled={tooLong}>
          {busy ? t('Signing and sending') : t('Publish the label')}
        </Button>
      </div>
    </form>
  )
}
