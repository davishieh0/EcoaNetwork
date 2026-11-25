/**
 * The claim routes: the human claim the gateway issues as a verifier.
 */

// --- IMPORTS ---
import type { HumanClaimResult } from '@/logic/datasource/claims.t'
import { BASE, request } from '@/logic/datasource/http'

// --- GLOBALS ---
/** Where the check widget fetches a proof of work to solve. */
export const HUMAN_CHALLENGE_URL = `${BASE}/claims/human/challenge`

// --- CODE ---
/**
 * Asks the gateway for a human claim about an identity.
 *
 * @param subject - The did:key the claim is about.
 * @param altcha - The solved proof of work, as the widget hands it.
 * @returns The claim issued and what the node did with it.
 */
export function issueHumanClaim(subject: string, altcha: string) {
  return request<HumanClaimResult>('/claims/human', {
    method: 'POST',
    body: { subject, altcha },
  })
}
