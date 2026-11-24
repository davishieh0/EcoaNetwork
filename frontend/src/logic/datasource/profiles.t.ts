/**
 * Types for the profile routes.
 */

// --- IMPORTS ---
import type { Signed } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
/** An identity's current `profile.updated`, signed. */
export type ProfileItem = Signed<{ identity: string }>

/** The profiles of the identities asked for, those with none left out. */
export type Profiles = {
  items: ProfileItem[]
}
