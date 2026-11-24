/**
 * The review routes: one review with its chain, and its conversation.
 */

// --- IMPORTS ---
import { request, route } from '@/logic/datasource/http'
import type { ReviewDetail } from '@/logic/datasource/reviews.t'

// --- CODE ---
/**
 * Reads one version of a review, with every version of its chain.
 *
 * @param eventId - The eventId of any version.
 * @returns The version asked for and the chain.
 */
export function getReview(eventId: string) {
  return request<ReviewDetail>(route(['reviews', eventId]))
}
