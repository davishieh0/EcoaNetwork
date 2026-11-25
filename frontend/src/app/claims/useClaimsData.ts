/**
 * The verification screen's data: this browser's identity, the claims that
 * count for it, and issuing a human or a domain claim.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useMemo, useState } from 'react'
import { messageOf } from '@/app/forms/messages'
import { failureOf, publishSigned } from '@/app/reads/publish'
import { useRead } from '@/app/reads/useRead'
import { i18n } from '@/i18n/i18n'
import { issueHumanClaim } from '@/logic/datasource/claims'
import { getIdentity } from '@/logic/datasource/identities'
import { normalizeEntity } from '@/logic/protocol/entity'
import { NoIdentityError, loadIdentity } from '@/logic/protocol/identity'
import type { Recognized } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
// how long to read the claims again until the index shows a new one: a
// domain claim waits on the next DNS lookup, every 30 s at most
const FOLLOW = { tries: 40, every: 5000 }

// a domain claim lives at most a year
const DOMAIN_LIFETIME_DAYS = 364

// --- CODE ---
/**
 * A day, `days` from now, as the protocol writes times.
 *
 * @param days - How many days ahead.
 * @returns RFC 3339, UTC, whole seconds.
 */
function daysAhead(days: number): string {
  const at = new Date(Date.now() + days * 24 * 60 * 60 * 1000)
  return at.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

/**
 * Reads the identity of this browser and its claims, and issues new ones.
 *
 * @returns The did:key, its claims, the domain form's state, and the
 * actions.
 */
export function useClaimsData() {
  const [did, setDid] = useState<string | null>(null)
  const [missing, setMissing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<'human' | 'domain' | null>(null)
  const [humanAttempt, setHumanAttempt] = useState(0)
  const [waiting, setWaiting] = useState<{
    check: (claims: Recognized[]) => boolean
    tries: number
    what: string
  } | null>(null)

  // the identity this browser signs with; the screen makes one if none
  useEffect(() => {
    loadIdentity().then(
      (identity) => setDid(identity.did),
      (failure) =>
        failure instanceof NoIdentityError
          ? setMissing(true)
          : setError('This browser cannot keep a signing key.'),
    )
  }, [])

  const read = useCallback(
    () => (did ? getIdentity(did) : Promise.resolve(null)),
    [did],
  )
  const identity = useRead(read)
  const claims = useMemo(() => identity.data?.claims ?? [], [identity.data])
  const { reload } = identity

  // read again until the index shows what was just issued
  useEffect(() => {
    if (!waiting || identity.loading) return

    if (waiting.check(claims)) {
      const done = setTimeout(() => setWaiting(null), 0)
      return () => clearTimeout(done)
    }

    const timer = setTimeout(() => {
      setWaiting((now) =>
        now && now.tries > 1 ? { ...now, tries: now.tries - 1 } : null,
      )
      reload()
    }, FOLLOW.every)
    return () => clearTimeout(timer)
  }, [waiting, identity.loading, claims, reload])

  const human = useCallback(
    async (payload: string) => {
      if (!did) return
      setBusy('human')
      setError(null)

      try {
        const result = await issueHumanClaim(did, payload)
        if (result.outcome === 'rejected') {
          setError(
            i18n.t('The network refused the claim: {{reason}}', {
              reason: result.reason ?? '',
            }),
          )
          setHumanAttempt((n) => n + 1)
          return
        }

        setWaiting({
          check: (now) => now.some((c) => c.claimType === 'human'),
          tries: FOLLOW.tries,
          what: 'the human claim',
        })
      } catch (failure) {
        setError(messageOf(failure))
        setHumanAttempt((n) => n + 1)
      } finally {
        setBusy(null)
      }
    },
    [did],
  )

  const domain = useCallback(
    async (typed: string) => {
      const normalized = normalizeEntity('domain', typed)
      if (!did || !normalized.ok) return
      const name = normalized.id.slice('domain:'.length)

      setBusy('domain')
      setError(null)

      try {
        await publishSigned('verification.claim_created', {
          claimType: 'domain',
          subject: did,
          entity: normalized.id,
          evidenceRef: `dns-txt:_ecoa-verify.${name}`,
          expiresAt: daysAhead(DOMAIN_LIFETIME_DAYS),
        })

        setWaiting({
          check: (now) =>
            now.some(
              (c) => c.claimType === 'domain' && c.entity === normalized.id,
            ),
          tries: FOLLOW.tries,
          what: name,
        })
      } catch (failure) {
        setError(failureOf(failure))
      } finally {
        setBusy(null)
      }
    },
    [did],
  )

  return {
    did,
    missing,
    identityReady: (made: string) => {
      setMissing(false)
      setDid(made)
    },
    claims,
    loading: !missing && (did === null || identity.loading),
    error,
    busy,
    humanAttempt,
    waitingFor: waiting?.what ?? null,
    human,
    domain,
  }
}
