/**
 * Types for the identity routes.
 */

// --- IMPORTS ---
import type { Recognized } from '@/logic/datasource/signed.t'

// --- GLOBALS ---
/** The identity a key belongs to, and the claims that count for it. */
export type IdentityAnswer = {
  identity: string
  keys: string[]
  current: string
  contested: boolean
  revoked: { key: string; eventId: string; reason?: string }[]
  claims: Recognized[]
}
