/**
 * Moderation: whose labels this reader follows, and whether they label
 * themselves. A label hides nothing from the network; it changes what the
 * readers who follow its moderator see.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'
import {
  MAX_MODERATORS,
  setModeration,
  useModeration,
} from '@/logic/moderation/moderation'
import { peekIdentity } from '@/logic/protocol/identity'

// --- GLOBALS ---
const CARD = 'flex flex-col gap-4 border border-structural bg-surface p-6'

const DID = /^did:key:z[1-9A-HJ-NP-Za-km-z]{46,48}$/

// --- CODE ---
/**
 * One choice of whom to follow.
 *
 * @param props - Whether it is chosen, its words, and what choosing does.
 * @returns The option.
 */
function Option({
  chosen,
  title,
  body,
  onChoose,
}: {
  chosen: boolean
  title: string
  body: string
  onChoose: () => void
}) {
  const { t } = useTranslation()

  return (
    <button
      type="button"
      role="radio"
      aria-checked={chosen}
      onClick={onChoose}
      className={cn(
        'flex cursor-pointer flex-col gap-1 border p-4 text-left',
        'transition-colors duration-interface',
        chosen
          ? 'border-signal bg-signal-wash'
          : 'border-structural hover:border-strong',
      )}
    >
      <span className="font-display text-h4">{t(title)}</span>
      <span className="text-body-s text-secondary">{t(body)}</span>
    </button>
  )
}

/**
 * Renders the moderation screen.
 *
 * @returns The screen.
 */
export function Moderation() {
  const { t } = useTranslation()
  const { choice } = useModeration()
  const rise = useRise()
  const [typed, setTyped] = useState('')
  const [me, setMe] = useState<string | null>(null)

  useEffect(() => {
    peekIdentity().then(setMe, () => setMe(null))
  }, [])

  const candidate = typed.trim()
  const valid = DID.test(candidate)
  const full = choice.moderators.length >= MAX_MODERATORS

  const add = (did: string) => {
    if (choice.moderators.includes(did)) return
    setModeration({ follow: 'own', moderators: [...choice.moderators, did] })
    setTyped('')
  }

  return (
    <Container className="max-w-3xl pt-28 pb-16">
      <motion.h1 {...rise()} className="font-display text-h1">
        {t('Moderation')}
      </motion.h1>
      <motion.p {...rise(0.06)} className="reading-measure text-secondary">
        {t(
          'Anyone can label a review as spam, harassment or illegal. A ' +
            'label counts for the readers who follow its moderator, and for ' +
            'nobody else: nothing leaves the network, it is how you read it.',
        )}
      </motion.p>

      <motion.section
        {...rise(0.12)}
        role="radiogroup"
        aria-label={t('Whose labels you follow')}
        className="mt-8 grid gap-3 sm:grid-cols-2"
      >
        <Option
          chosen={choice.follow === 'index'}
          title="This index’s moderators"
          body="The list the index you read through follows."
          onChoose={() => setModeration({ follow: 'index' })}
        />
        <Option
          chosen={choice.follow === 'own'}
          title="My own list"
          body="The moderators you pick below; none, and nothing is hidden."
          onChoose={() => setModeration({ follow: 'own' })}
        />
      </motion.section>

      {choice.follow === 'own' ? (
        <motion.section {...rise(0.18)} className={cn(CARD, 'mt-6')}>
          <h2 className="font-display text-h4">{t('Moderators you follow')}</h2>

          {choice.moderators.length === 0 ? (
            <p className="meta text-muted">
              {t('Nobody: every review shows, whatever its labels.')}
            </p>
          ) : (
            <ul className="flex flex-col gap-2">
              {choice.moderators.map((did) => (
                <li
                  key={did}
                  className="flex items-center justify-between gap-3"
                >
                  <span className="font-mono text-mono-s break-all">
                    {did}
                    {did === me ? (
                      <span className="meta ml-2 text-muted">{t('(you)')}</span>
                    ) : null}
                  </span>
                  <button
                    type="button"
                    onClick={() =>
                      setModeration({
                        moderators: choice.moderators.filter((m) => m !== did),
                      })
                    }
                    className={
                      'meta cursor-pointer text-muted hover:text-danger'
                    }
                  >
                    {t('Remove')}
                  </button>
                </li>
              ))}
            </ul>
          )}

          <form
            className="flex gap-2"
            onSubmit={(event) => {
              event.preventDefault()
              if (valid && !full) add(candidate)
            }}
          >
            <input
              value={typed}
              onChange={(event) => setTyped(event.target.value)}
              placeholder="did:key:z6Mk…"
              aria-label={t("A moderator's did:key")}
              className={cn(
                'flex-1 border border-structural bg-canvas p-3 font-mono',
                'text-mono-s focus:border-strong',
              )}
            />
            <Button type="submit" disabled={!valid || full}>
              {t('Follow')}
            </Button>
          </form>
          {candidate && !valid ? (
            <p className="text-body-s text-danger">
              {t('That is not a did:key.')}
            </p>
          ) : null}
          {me && !choice.moderators.includes(me) ? (
            <div>
              <Button variant="ghost" size="sm" onClick={() => add(me)}>
                {t('Follow yourself, to see your own labels count')}
              </Button>
            </div>
          ) : null}
        </motion.section>
      ) : null}

      <motion.section {...rise(0.24)} className={cn(CARD, 'mt-6')}>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="font-display text-h4">{t('Moderator mode')}</h2>
          {choice.moderating ? (
            <Badge tone="signal" glyph="●">
              {t('On')}
            </Badge>
          ) : null}
        </div>
        <p className="text-body-s text-secondary">
          {t(
            'Shows an action to label reviews and responses, signed by ' +
              'your identity. Your labels count for whoever follows you.',
          )}
        </p>
        <div>
          <Button
            variant={choice.moderating ? 'secondary' : 'primary'}
            onClick={() => setModeration({ moderating: !choice.moderating })}
          >
            {choice.moderating ? t('Turn off') : t('Turn on')}
          </Button>
        </div>
        {me ? (
          <p className="meta text-muted">
            {t('You label as')}{' '}
            <span className="font-mono normal-case break-all">{me}</span>
          </p>
        ) : null}
      </motion.section>
    </Container>
  )
}
