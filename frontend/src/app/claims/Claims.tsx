/**
 * Verification: the claims this browser's identity holds, and the two it
 * can get now — human, through a check, and domain, through a DNS record.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { HumanCheck } from '@/app/claims/HumanCheck'
import { IdentitySetup } from '@/app/identity/IdentitySetup'
import { useClaimsData } from '@/app/claims/useClaimsData'
import { Status } from '@/app/reads/Status'
import { dateOf } from '@/app/reads/words'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'
import { known } from '@/i18n/i18n'
import { normalizeEntity } from '@/logic/protocol/entity'

// --- GLOBALS ---
const CARD = 'flex flex-col gap-4 border border-structural bg-surface p-6'

const FIELD = cn(
  'w-full border border-structural bg-canvas p-3 font-mono text-mono-m',
  'transition-colors duration-interface focus:border-strong',
)

// --- CODE ---
/**
 * One line to copy into a DNS zone.
 *
 * @param props - What the line is, and its text.
 * @returns The line, with a copy button.
 */
function Copyable({ label, value }: { label: string; value: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  return (
    <div className="flex flex-col gap-1">
      <span className="meta text-muted">{t(label)}</span>
      <div className="flex items-stretch gap-2">
        <code
          className={cn(
            'flex-1 border border-line bg-surface-sunken p-3 font-mono',
            'text-mono-s break-all text-primary',
          )}
        >
          {value}
        </code>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => {
            void navigator.clipboard?.writeText(value)
            setCopied(true)
          }}
        >
          {copied ? t('Copied') : t('Copy')}
        </Button>
      </div>
    </div>
  )
}

/**
 * Renders the verification screen.
 *
 * @returns The screen.
 */
export function Claims() {
  const {
    did,
    missing,
    identityReady,
    claims,
    loading,
    error,
    busy,
    humanAttempt,
    waitingFor,
    human,
    domain,
  } = useClaimsData()
  const { t } = useTranslation()
  const rise = useRise()
  const [typed, setTyped] = useState('')

  const normalized = typed.trim() ? normalizeEntity('domain', typed) : null
  const name = normalized?.ok ? normalized.id.slice('domain:'.length) : null
  const isHuman = claims.some((c) => c.claimType === 'human')

  const status = error
    ? error
    : waitingFor
      ? t(
          'Published. Waiting for the index to recognize {{what}}… A ' +
            'domain is looked up every 30 seconds.',
          { what: known(t, waitingFor) },
        )
      : ''

  return (
    <Container className="max-w-3xl pt-28 pb-16">
      <motion.h1 {...rise()} className="font-display text-h1">
        {t('Verification')}
      </motion.h1>
      <motion.p {...rise(0.06)} className="reading-measure text-secondary">
        {t(
          'Claims are signed statements about your identity. Nobody needs ' +
            'to approve them: an index checks a domain claim itself, and ' +
            'decides whose word it takes for the rest.',
        )}
      </motion.p>

      {missing ? (
        <motion.div {...rise(0.12)} className="mt-8">
          <IdentitySetup onDone={identityReady} />
        </motion.div>
      ) : (
        <>
          <motion.section {...rise(0.12)} className={cn(CARD, 'mt-8')}>
            <p className="meta text-muted">{t('This browser’s identity')}</p>
            <p className="font-mono text-mono-s break-all">
              {did ?? t('Opening the key…')}
            </p>
            <div className="flex flex-wrap gap-2">
              {loading ? (
                <span className="meta text-muted">
                  {t('Reading your claims…')}
                </span>
              ) : claims.length === 0 ? (
                <span className="meta text-muted">
                  {t('No claim counts yet.')}
                </span>
              ) : (
                claims.map((c) => (
                  <Badge
                    key={c.eventId}
                    tone={c.claimType === 'domain' ? 'valid' : 'info'}
                    glyph="◆"
                  >
                    {c.claimType === 'domain'
                      ? `${t('Domain')} · ` +
                        `${c.entity?.slice('domain:'.length)}` +
                        (c.since
                          ? ` · ${t('since {{date}}', {
                              date: dateOf(c.since),
                            })}`
                          : '')
                      : t('Human')}
                  </Badge>
                ))
              )}
            </div>
          </motion.section>

          <Status text={status} failed={!!error} className="mt-4" />

          <motion.section {...rise(0.18)} className={cn(CARD, 'mt-6')}>
            <div className="flex flex-col gap-1">
              <h2 className="font-display text-h4">{t('Human')}</h2>
              <p className="text-body-s text-secondary">
                {t(
                  'A small proof of work, solved in this browser. It ' +
                    'identifies nobody: the gateway signs that this key paid ' +
                    'the cost, which counts a little more than a brand-new ' +
                    'key.',
                )}
              </p>
            </div>
            {isHuman ? (
              <Badge tone="info" glyph="◆">
                {t('You hold a human claim')}
              </Badge>
            ) : did ? (
              <div
                aria-busy={busy === 'human'}
                className="flex flex-col gap-2"
              >
                <HumanCheck
                  key={humanAttempt}
                  onSolved={(payload) => void human(payload)}
                />
                {busy === 'human' ? (
                  <span className="meta text-muted">
                    {t('Issuing human claim…')}
                  </span>
                ) : null}
              </div>
            ) : null}
          </motion.section>

          <motion.section {...rise(0.24)} className={cn(CARD, 'mt-6')}>
            <div className="flex flex-col gap-1">
              <h2 className="font-display text-h4">{t('Domain')}</h2>
              <p className="text-body-s text-secondary">
                {t(
                  'Prove you control a website by publishing a DNS record. ' +
                    'The index looks it up itself: there is no verifier to ' +
                    'trust. A verified domain makes you its representative: ' +
                    'your details for it hold, and your responses to its ' +
                    'reviews are marked official.',
                )}
              </p>
            </div>

            <label className="flex flex-col gap-1">
              <span className="meta text-muted">{t('Your domain')}</span>
              <input
                value={typed}
                onChange={(event) => setTyped(event.target.value)}
                placeholder="example.com"
                className={FIELD}
              />
              {normalized && !normalized.ok ? (
                <span className="text-body-s text-danger">
                  {t(normalized.reason)}
                </span>
              ) : null}
            </label>

            {name && did ? (
              <>
                <p className="text-body-s text-secondary">
                  {t(
                    '1. Add this TXT record to the domain’s DNS. It can stay ' +
                      'next to others, and it has to stay: the claim is ' +
                      'checked again.',
                  )}
                </p>
                <Copyable label="Name" value={`_ecoa-verify.${name}`} />
                <Copyable label="Value" value={`ecoa-identity=${did}`} />
                <p className="text-body-s text-secondary">
                  {t(
                    '2. Sign the claim. It says this identity holds ' +
                      '{{name}} for a year, and it counts once the index ' +
                      'finds the record.',
                    { name },
                  )}
                </p>
                <div>
                  <Button
                    type="button"
                    pending={busy === 'domain'}
                    onClick={() => void domain(typed)}
                  >
                    {busy === 'domain'
                      ? t('Signing and sending')
                      : t('Sign the claim')}
                  </Button>
                </div>
              </>
            ) : null}
          </motion.section>
        </>
      )}
    </Container>
  )
}
