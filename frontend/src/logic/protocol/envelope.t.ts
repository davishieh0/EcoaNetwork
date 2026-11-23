/**
 * Types for the event envelope.
 */

// --- IMPORTS ---
import type { Did } from '@/logic/protocol/identity.t'
import type { JsonObject } from '@/logic/protocol/jcs.t'

// --- GLOBALS ---
/** The envelope's signed fields, in the order the protocol lists them. */
export type UnsignedEnvelope = {
  protocol: string
  version: 1
  author: Did
  deviceKey: null
  eventType: string
  createdAt: string
  payload: JsonObject
}

/** The event as it travels: the signed fields, the id and the signature. */
export type Envelope = {
  protocol: string
  version: 1
  eventId: string
  author: Did
  deviceKey: null
  eventType: string
  createdAt: string
  payload: JsonObject
  signature: string
}
