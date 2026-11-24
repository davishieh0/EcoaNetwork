/**
 * The Write screen's data: the account, the draft and where it stands.
 */

// --- IMPORTS ---
import { useEffect, useState } from 'react'

import { messageOf } from '@/app/forms/messages'
import { useRouteState } from '@/app/navigation/navigation'
import type { WritePhase, WriteState } from '@/app/write/Write.t'
import { publishEvent } from '@/logic/datasource/events'
import { HttpError } from '@/logic/errors/httpError'
import { MAX_ENVELOPE, signEnvelope, sizeOf } from '@/logic/protocol/envelope'
import { loadIdentity, peekIdentity } from '@/logic/protocol/identity'
import { normalizeEntity } from '@/logic/protocol/entity'
import { checkReview } from '@/logic/protocol/review'
import type { EntityKind } from '@/logic/protocol/entity.t'
import type { ReviewDraft } from '@/logic/protocol/review.t'
import { useSession } from '@/logic/session/session'

// --- GLOBALS ---
const EMPTY: ReviewDraft = {
  kind: 'cnpj',
  identifier: '',
  rating: null,
  title: '',
  body: '',
  lang: '',
}

const NOT_SIGNED = 'This browser could not sign the review. Nothing was sent.'

const REFUSED = 'The network refused this review. Nothing was published.'

const TOO_LARGE =
  'Signed, this review is over the 16 KB the network accepts. Shorten it.'

// --- CODE ---
/**
 * Words a failed publish for the screen.
 *
 * @param error - Whatever signing or sending threw.
 * @returns The message to show.
 */
function failureOf(error: unknown): string {
  if (error instanceof HttpError) return messageOf(error)

  return NOT_SIGNED
}

/**
 * Words the node's refusal for the screen.
 *
 * @param reason - Why the node rejected the event, if it said.
 * @returns The message to show.
 */
function refusalOf(reason: string | null): string {
  return reason ? `${REFUSED} Reason: ${reason}` : REFUSED
}

/**
 * Starts a draft, about the entity the previous screen handed over when it
 * is one the composer offers.
 *
 * @param state - What the previous screen handed over.
 * @returns The draft to start from.
 */
function draftFor(state: WriteState): ReviewDraft {
  const [kind, ...rest] = (state.entity ?? '').split(':')
  const known = ['cnpj', 'domain', 'gtin'].includes(kind)

  return known
    ? { ...EMPTY, kind: kind as EntityKind, identifier: rest.join(':') }
    : EMPTY
}

/**
 * Formats now as createdAt: RFC 3339, UTC, whole seconds, literal Z.
 *
 * @returns The timestamp.
 */
function now(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, 'Z')
}

/**
 * Holds the draft and moves it from writing to checked to published.
 *
 * @returns The account, the draft, its problems and the actions on it.
 */
export function useWriteData() {
  const session = useSession()
  const handed = useRouteState<'write'>()

  // whether this browser has an identity to sign with; undefined until read
  const [hasIdentity, setHasIdentity] = useState<boolean | undefined>()
  useEffect(() => {
    peekIdentity().then(
      (did) => setHasIdentity(did !== null),
      () => setHasIdentity(false),
    )
  }, [])
  const [draft, setDraft] = useState<ReviewDraft>(() => draftFor(handed))
  const [phase, setPhase] = useState<WritePhase>({
    name: 'editing',
    attempted: false,
  })

  const result = checkReview(draft)
  const attempted = phase.name === 'editing' && phase.attempted

  // the draft is frozen once it is on its way
  const locked = phase.name === 'publishing' || phase.name === 'published'

  // a field's problem shows once the user has tried to check
  const problems = attempted && !result.ok ? result.problems : {}

  /**
   * Changes one field; a checked draft goes back to writing.
   *
   * @param field - The field to change.
   * @param value - Its new value.
   */
  const update = <K extends keyof ReviewDraft>(
    field: K,
    value: ReviewDraft[K],
  ) => {
    if (locked) return

    setDraft((current) => ({ ...current, [field]: value }))
    setPhase((current) =>
      current.name === 'checked'
        ? { name: 'editing', attempted: true }
        : current,
    )
  }

  /**
   * Freezes the payload with its time, or shows what is missing.
   */
  const check = () => {
    if (!result.ok) {
      setPhase({ name: 'editing', attempted: true })

      return
    }

    setPhase({
      name: 'checked',
      payload: result.payload,
      createdAt: now(),
      error: null,
    })
  }

  /**
   * Leaves the checked view and returns to writing.
   */
  const edit = () => {
    if (phase.name === 'checked') setPhase({ name: 'editing', attempted: true })
  }

  /**
   * Signs the checked review and sends it; a failure returns to checked.
   * The same createdAt signs to the same eventId, so a retry cannot double it.
   */
  const publish = async () => {
    if (phase.name !== 'checked') return

    const { payload, createdAt } = phase
    setPhase({ name: 'publishing', payload, createdAt })

    try {
      const identity = await loadIdentity()
      const envelope = await signEnvelope(
        identity,
        'review.created',
        payload,
        createdAt,
      )

      if (sizeOf(envelope) > MAX_ENVELOPE) {
        setPhase({ name: 'checked', payload, createdAt, error: TOO_LARGE })

        return
      }

      const published = await publishEvent(envelope)

      // the node refused it: back to checked, so the draft can be edited
      if (published.outcome === 'rejected') {
        const error = refusalOf(published.reason)
        setPhase({ name: 'checked', payload, createdAt, error })

        return
      }

      setPhase({
        name: 'published',
        payload,
        createdAt,
        envelope,
        outcome: published.outcome,
      })
    } catch (error) {
      setPhase({ name: 'checked', payload, createdAt, error: failureOf(error) })
    }
  }

  /**
   * Clears the draft to write another review.
   */
  const reset = () => {
    setDraft(draftFor(handed))
    setPhase({ name: 'editing', attempted: false })
  }

  return {
    hasIdentity,
    identityReady: () => setHasIdentity(true),
    account: session.status === 'signed-in' ? session.account : null,
    draft,
    entity: normalizeEntity(draft.kind, draft.identifier),
    phase,
    problems,
    update,
    check,
    edit,
    publish,
    reset,
  }
}
