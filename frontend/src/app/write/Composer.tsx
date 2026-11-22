/**
 * The review composer as a cell grid: the subject, the rating, the words,
 * and what gets signed.
 */

// --- IMPORTS ---
import { KINDS, LANGUAGES, RATINGS } from '@/app/write/options'
import { SignedPreview } from '@/app/write/SignedPreview'
import type { ComposerProps, WritePhase } from '@/app/write/Write.t'
import { Button } from '@/components/ui/Button'
import { cn } from '@/components/ui/cn'
import { Select } from '@/components/ui/Select'
import { Tooltip } from '@/components/ui/Tooltip'
import { BODY_MAX, characters, TITLE_MAX } from '@/logic/protocol/review'

// --- GLOBALS ---
const CONTROL = cn(
  'min-h-11 border border-line bg-transparent px-3 text-body-m text-primary',
  'transition-colors duration-interface hover:border-strong',
  'focus:border-signal focus-visible:shadow-none focus-visible:outline-none',
)

// what the footer says about where the review is
const NOTES: Record<WritePhase['name'], string> = {
  editing: 'nothing leaves this computer until you publish',
  checked: 'nothing has been sent',
  publishing: 'signing in this browser, then sending',
  published: 'sent',
}

// --- CODE ---
/**
 * Builds the composer.
 *
 * @param props - The write data.
 * @returns The composer, or nothing until the account is known.
 */
export function Composer({ data }: ComposerProps) {
  const {
    account,
    draft,
    entity,
    phase,
    problems,
    update,
    check,
    edit,
    publish,
    reset,
  } = data

  if (!account?.handle) return null

  const signed = phase.name === 'editing' ? null : phase
  const locked = phase.name === 'publishing' || phase.name === 'published'
  const kind = KINDS.find((option) => option.value === draft.kind) ?? KINDS[0]

  return (
    <form
      id="compose"
      noValidate
      onSubmit={(event) => {
        event.preventDefault()
        check()
      }}
      className="flex scroll-mt-[calc(var(--header-h)+1.5rem)] flex-col"
    >
      <div className="grid border-b border-line md:grid-cols-[1.25fr_1fr]">
        <div
          className={cn(
            'flex flex-col gap-3 border-b border-line p-6 md:border-r',
            'md:border-b-0',
          )}
        >
          <span
            className={cn(
              'flex items-center gap-2',
              'font-mono text-mono-s text-muted',
            )}
          >
            What it is about
            <Tooltip label={`About ${kind.label}`}>{kind.hint}</Tooltip>
          </span>
          <fieldset
            disabled={locked}
            className={cn(
              'grid min-w-0 grid-cols-[6rem_1fr]',
              'sm:grid-cols-[8.5rem_1fr]',
            )}
          >
            <Select
              label="Identifier type"
              value={draft.kind}
              options={KINDS}
              onChange={(value) => update('kind', value)}
              className="min-h-11 border-r-0"
            />
            <input
              value={draft.identifier}
              onChange={(event) => update('identifier', event.target.value)}
              placeholder={kind.placeholder}
              aria-label={kind.label}
              aria-invalid={problems.identifier ? true : undefined}
              inputMode={draft.kind === 'domain' ? 'url' : 'numeric'}
              autoComplete="off"
              spellCheck={false}
              className={cn(
                CONTROL,
                'min-w-0 font-mono tabular-nums placeholder:text-muted',
                problems.identifier && 'border-danger',
              )}
            />
          </fieldset>
          <p
            className={cn(
              'min-h-5 font-mono text-mono-s break-all',
              problems.identifier ? 'text-danger' : 'text-muted',
            )}
          >
            {problems.identifier ?? (entity.ok ? `signs as ${entity.id}` : '')}
          </p>
        </div>

        <div className="flex flex-col gap-3 p-6">
          <span className="font-mono text-mono-s text-muted">
            Rating · 1 worst, 5 best
          </span>
          <div
            role="radiogroup"
            aria-label="Rating, 1 worst to 5 best"
            className={cn(
              'grid grid-cols-5 border',
              problems.rating ? 'border-danger' : 'border-line',
            )}
          >
            {RATINGS.map((rating) => (
              <button
                key={rating}
                type="button"
                role="radio"
                aria-checked={draft.rating === rating}
                disabled={locked}
                onClick={() => update('rating', rating)}
                className={cn(
                  'min-h-11 cursor-pointer border-r border-line',
                  'font-display text-h4 font-semibold tabular-nums',
                  'transition-colors duration-interface last:border-r-0',
                  draft.rating === rating
                    ? 'bg-signal text-on-signal'
                    : 'text-secondary hover:bg-surface hover:text-primary',
                )}
              >
                {rating}
              </button>
            ))}
          </div>
          <p className="min-h-5 font-mono text-mono-s text-danger">
            {problems.rating}
          </p>
        </div>
      </div>

      <div className="flex flex-col gap-4 border-b border-line p-6">
        <textarea
          rows={1}
          value={draft.title}
          onChange={(event) =>
            update('title', event.target.value.replace(/\n/g, ' '))
          }
          maxLength={TITLE_MAX}
          readOnly={locked}
          placeholder="A title, if you want one"
          aria-label="Title (optional)"
          className={cn(
            'field-sizing-content resize-none bg-transparent font-display',
            'text-h2 font-semibold tracking-heading outline-none',
            'placeholder:text-strong focus-visible:shadow-none',
            'focus-visible:outline-none',
          )}
        />
        {problems.title ? (
          <p className="font-mono text-mono-s text-danger">{problems.title}</p>
        ) : null}
        <textarea
          value={draft.body}
          onChange={(event) => update('body', event.target.value)}
          placeholder="Tell it the way it happened."
          aria-label="What happened"
          readOnly={locked}
          rows={3}
          className={cn(
            'field-sizing-content min-h-28 resize-none border-l-2',
            'bg-transparent pl-5 text-body-l leading-relaxed outline-none',
            'placeholder:text-strong focus-visible:shadow-none',
            'focus-visible:outline-none',
            problems.body ? 'border-danger' : 'border-line focus:border-signal',
          )}
        />
        {problems.body ? (
          <p className="font-mono text-mono-s text-danger">{problems.body}</p>
        ) : null}
      </div>

      {signed ? (
        <div className="border-b border-line px-4 py-6 sm:px-6">
          <SignedPreview phase={signed} onEdit={edit} />
        </div>
      ) : null}

      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-4 px-6 py-4',
          'font-mono text-mono-s text-muted',
        )}
      >
        <div className="flex items-center gap-6">
          <span className="tabular-nums">
            {characters(draft.body)} / {BODY_MAX}
          </span>
          <fieldset disabled={locked} className="contents">
            <Select
              label="Language"
              value={draft.lang}
              options={LANGUAGES}
              onChange={(value) => update('lang', value)}
            />
          </fieldset>
          <span className="hidden lg:inline">{NOTES[phase.name]}</span>
        </div>
        <div className="flex flex-wrap items-center justify-end gap-4">
          {phase.name === 'checked' && phase.error ? (
            <p role="alert" className="text-danger">
              {phase.error}
            </p>
          ) : null}
          {phase.name === 'published' ? (
            <Button variant="secondary" type="button" onClick={reset}>
              Write another
            </Button>
          ) : phase.name === 'editing' ? (
            <Button type="submit">Preview</Button>
          ) : (
            <Button
              type="button"
              pending={phase.name === 'publishing'}
              onClick={publish}
            >
              {phase.name === 'publishing' ? 'Signing and sending' : 'Publish'}
            </Button>
          )}
        </div>
      </div>
    </form>
  )
}
