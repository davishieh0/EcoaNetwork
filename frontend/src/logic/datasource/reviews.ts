/**
 * The review routes: one review with its chain, and its conversation.
 */

// --- IMPORTS ---
import { request, route } from '@/logic/datasource/http'
import type {
  ResponseItem,
  ReviewDetail,
} from '@/logic/datasource/reviews.t'
import type { Labeled, Page } from '@/logic/datasource/signed.t'

// --- CODE ---
/**
 * Reads one version of a review, with every version of its chain.
 *
 * @param eventId - The eventId of any version.
 * @returns The version asked for and the chain.
 */
export function getReview(eventId: string, moderators?: string) {
  return request<ReviewDetail>(route(['reviews', eventId], { moderators }))
}

/**
 * Lists the conversation on a review, in order.
 *
 * @param eventId - The eventId of any version of the review.
 * @param after - The next of a previous page.
 * @param labeled - 'show', to see what moderation hides, marked.
 * @returns A page of responses.
 */
export function listReviewResponses(
  eventId: string,
  after?: string,
  labeled?: Labeled,
  moderators?: string,
) {
  return request<Page<ResponseItem>>(
    route(['reviews', eventId, 'responses'], { after, labeled, moderators }),
  )
}
