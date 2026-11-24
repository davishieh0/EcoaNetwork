/**
 * The review screen's data: the review with its chain, its conversation,
 * and the names of everyone on it.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useState } from 'react'
import { useLocation, useParams, useSearchParams } from 'react-router'
import type { ReviewState } from '@/app/reads/review/ReviewPage.t'
import { useAuthors } from '@/app/reads/useAuthors'
import { usePaged } from '@/app/reads/usePaged'
import { useRead } from '@/app/reads/useRead'
import type { ReviewItem } from '@/logic/datasource/entities.t'
import { getReview, listReviewResponses } from '@/logic/datasource/reviews'
import { HttpError } from '@/logic/errors/httpError'
import { useModeration } from '@/logic/moderation/moderation'
import { readEnvelope, reviewTextOf } from '@/logic/protocol/signed'

// --- GLOBALS ---
// how long to wait for the index to have a review just published
const PATIENCE = { tries: 10, every: 1500 }

// --- CODE ---
/**
 * Reads the review in the address, any of its versions, and its
 * conversation, a page at a time.
 *
 * @returns The review, its current version and entity, the conversation,
 * and the authors' names.
 */
export function useReviewData() {
  const eventId = useParams().eventId ?? ''
  // a review's path holds its id, so this screen is not in RouteStates
  const { published } = (useLocation().state ?? {}) as ReviewState

  const [params, setParams] = useSearchParams()
  const labeled = params.get('labeled') === 'show' ? 'show' : undefined

  // the review just published, until the index has it: read it again
  const [attempt, setAttempt] = useState(0)

  const { param } = useModeration()

  const readReview = useCallback(
    () => getReview(eventId, param),
    // a new attempt is a new read
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [eventId, param, attempt],
  )
  const review = useRead(readReview)

  const loadResponses = useCallback(
    (after?: string) => listReviewResponses(eventId, after, labeled, param),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [eventId, labeled, param, attempt],
  )
  const responses = usePaged(loadResponses)

  // while reading again too, so the review does not blink in and out
  const waiting =
    published?.eventId === eventId &&
    !review.data &&
    (review.loading ||
      (review.error instanceof HttpError && review.error.status === 404))

  useEffect(() => {
    if (!waiting || review.loading || attempt >= PATIENCE.tries) return
    const timer = setTimeout(
      () => setAttempt((now) => now + 1),
      PATIENCE.every,
    )
    return () => clearTimeout(timer)
  }, [waiting, review.loading, attempt])

  // what the browser signed stands in, marked, while the index catches up
  const pending: ReviewItem | null =
    waiting && published
      ? {
          envelope: JSON.stringify(published),
          derived: {
            chainRoot: published.eventId,
            head: true,
            withdrawn: false,
            endorsements: 0,
            disputes: 0,
          },
          observed: {},
        }
      : null

  const versions = review.data?.versions ?? []
  const head = versions.find((version) => version.derived.head) ?? pending

  // the entity is named by the original, the one that is not an edit
  const entity =
    versions
      .map((version) => reviewTextOf(readEnvelope(version.envelope)).entity)
      .find((named) => named !== undefined) ?? null

  const names = useAuthors(
    [...versions, ...responses.items].map(
      (item) => readEnvelope(item.envelope).author,
    ),
    param,
  )

  const showLabeled = useCallback(
    (show: boolean) =>
      // keeps what the previous screen handed over
      setParams(show ? { labeled: 'show' } : {}, {
        replace: true,
        state: published ? { published } : undefined,
      }),
    [setParams, published],
  )

  return {
    eventId,
    labeled: !!labeled,
    showLabeled,
    review,
    head,
    pending: pending !== null,
    entity:
      entity ??
      (published && pending ? reviewTextOf(published).entity : null) ??
      null,
    responses,
    names,
  }
}
