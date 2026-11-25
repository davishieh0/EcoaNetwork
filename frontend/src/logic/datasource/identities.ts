/**
 * The identity routes.
 */

// --- IMPORTS ---
import { request, route } from '@/logic/datasource/http'
import type { IdentityAnswer } from '@/logic/datasource/identities.t'

// --- CODE ---
/**
 * Reads the identity a key belongs to, with the claims that count for it.
 *
 * @param did - Any did:key of the identity.
 * @returns The identity.
 */
export function getIdentity(did: string) {
  return request<IdentityAnswer>(route(['identities', did]))
}
