/**
 * What a reader does on a review: its author edits or withdraws it, anyone
 * else votes on it, and everyone can respond.
 */

// --- IMPORTS ---
import { useState } from 'react'
import {
  PublishFailure,
  failureOf,
  publishSigned,
} from '@/app/reads/publish'
import type {
  ActionMode,
  Expectation,
  WithdrawReason,
} from '@/app/reads/review/ReviewPage.t'
import { i18n } from '@/i18n/i18n'
import type { ReviewItem } from '@/logic/datasource/entities.t'
import type { VersionText } from '@/logic/protocol/review.t'
import { readEnvelope } from '@/logic/protocol/signed'

// --- CODE ---
/**
 * Holds which action is open and runs each one: sign, publish, then
 * expect the index to show it.
 *
 * @param head - The review's current version, null until it is read.
 * @param expect - Reads again until the index shows what was done.
 * @param known - Tells the page this browser now has an identity.
 * @returns The open action, what is on its way, the last outcome, and the
 * actions.
 */
export function useReviewActions(
  head: ReviewItem | null,
  expect: (check: Expectation['check']) => void,
  known: () => void,
) {
  const [mode, setMode] = useState<ActionMode>('idle')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [needsIdentity, setNeedsIdentity] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  // the response being answered; null answers the review itself
  const [replyTo, setReplyTo] = useState<string | null>(null)
  // what a moderator is labeling: the review's head or a response
  const [labeling, setLabeling] = useState<string | null>(null)

  const headId = head ? readEnvelope(head.envelope).eventId : null
  // an edit or a withdrawal is signed by the key that wrote the review
  const author = head ? readEnvelope(head.envelope).author : undefined

  /**
   * Signs and publishes, then hands the new eventId on.
   *
   * @param eventType - The event's type.
   * @param payload - Its payload.
   * @param done - What to do once the network took it.
   * @returns Whether it was published.
   */
  const run = async (
    eventType: string,
    payload: Record<string, string | number>,
    done: (eventId: string) => void,
    signer?: string,
  ) => {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      const envelope = await publishSigned(eventType, payload, signer)
      known()
      done(envelope.eventId)
      return true
    } catch (failure) {
      setError(failureOf(failure))
      setNeedsIdentity(
        failure instanceof PublishFailure && failure.needsIdentity,
      )
      return false
    } finally {
      setBusy(false)
    }
  }

  const edit = (text: VersionText) => {
    if (!headId) return Promise.resolve(false)

    return run('review.replaced', { replaces: headId, ...text }, (id) => {
      setMode('idle')
      setNotice('Edited. The new version is on the network.')
      expect(({ detail }) =>
        Boolean(
          detail?.versions.some(
            (version) =>
              version.derived.head &&
              readEnvelope(version.envelope).eventId === id,
          ),
        ),
      )
    }, author)
  }

  const withdraw = (reason: WithdrawReason | null) => {
    if (!headId) return Promise.resolve(false)

    const payload: Record<string, string> = { target: headId }
    if (reason) payload.reason = reason

    return run('review.withdrawn', payload, () => {
      setMode('idle')
      expect(({ status }) => status === 410)
    }, author)
  }

  const vote = (kind: 'endorsed' | 'disputed') => {
    if (!headId || !head) return Promise.resolve(false)
    const before = head.derived

    return run(`review.${kind}`, { target: headId }, () => {
      setNotice(
        kind === 'endorsed'
          ? 'You endorsed this version.'
          : 'You disputed this version.',
      )
      // a vote counts on this very version, and only its latest counts
      expect(({ detail }) => {
        const now = detail?.versions.find((version) => version.derived.head)
        return (
          !now ||
          now.derived.endorsements !== before.endorsements ||
          now.derived.disputes !== before.disputes
        )
      })
    })
  }

  const respond = (body: string) => {
    if (!headId) return Promise.resolve(false)

    return run(
      'review.response_created',
      { target: replyTo ?? headId, body },
      (id) => {
        setReplyTo(null)
        setNotice('Your response is on the network.')
        expect(({ responses }) =>
          responses.some((item) => readEnvelope(item.envelope).eventId === id),
        )
      },
    )
  }

  const label = (labelName: string, note: string | null) => {
    const target = labeling
    if (!target) return Promise.resolve(false)

    const payload: Record<string, string> = { target, label: labelName }
    if (note) payload.note = note

    return run('moderation.label_created', payload, (id) => {
      setLabeling(null)
      setNotice(
        i18n.t('Labeled {{label}}. It counts for whoever follows you.', {
          label: labelName,
        }),
      )
      // shown marked, hidden from the list, or removed: any of them shows
      // the index has it, for a reader who follows this moderator
      expect((seen) => {
        const text = JSON.stringify(seen)
        return (
          text.includes(id) || !text.includes(target) || seen.status === 451
        )
      })
    })
  }

  const revoke = (labelId: string) =>
    run('moderation.label_revoked', { target: labelId }, () => {
      setNotice('Label revoked.')
      expect((seen) => !JSON.stringify(seen).includes(labelId))
    })

  return {
    mode,
    labeling,
    setLabeling: (target: string | null) => {
      setError(null)
      setNotice(null)
      setLabeling(target)
    },
    label,
    revoke,
    open: (next: ActionMode) => {
      setError(null)
      setNotice(null)
      setMode(next)
    },
    busy,
    error,
    needsIdentity,
    notice,
    replyTo,
    setReplyTo,
    edit,
    withdraw,
    vote,
    respond,
  }
}
