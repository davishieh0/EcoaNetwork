/**
 * The profile routes.
 */

// --- IMPORTS ---
import { request, route } from '@/logic/datasource/http'
import type { ProfileItem, Profiles } from '@/logic/datasource/profiles.t'

// --- CODE ---
/**
 * Reads the current profiles of some identities in one call.
 *
 * @param ids - 1 to 100 did:key.
 * @returns Their profiles, in the order asked, those with none left out.
 */
export function getProfiles(ids: string[], moderators?: string) {
  return request<Profiles>(
    route(['profiles'], { ids: ids.join(','), moderators }),
  )
}

/**
 * Reads one identity's current profile.
 *
 * @param did - Any did:key of the identity.
 * @returns The profile, signed.
 */
export function getProfile(did: string) {
  return request<ProfileItem>(route(['profiles', did]))
}
