/**
 * Builds and signs the event envelope, as docs/protocol.md defines it.
 */

// --- IMPORTS ---
import { base32, base64url } from '@/logic/protocol/encoding'
import type {
  Envelope,
  UnsignedEnvelope,
} from '@/logic/protocol/envelope.t'
import type { Identity } from '@/logic/protocol/identity.t'
import { canonicalize } from '@/logic/protocol/jcs'
import type { JsonObject } from '@/logic/protocol/jcs.t'

// --- GLOBALS ---
/** The network this client publishes to. */
export const NETWORK = import.meta.env.VITE_ECOA_NETWORK ?? 'ecoa-dev'

/** The largest envelope a node accepts, in bytes. */
export const MAX_ENVELOPE = 16 * 1024

// the domain separation prefix, with the 0x00 that ends it
const PREFIX = new TextEncoder().encode('ecoa-event-v1\0')

// --- CODE ---
/**
 * Builds the bytes that are hashed and signed.
 *
 * @param unsigned - The envelope's signed fields.
 * @returns The prefix followed by the canonical JSON.
 */
export function signingInput(
  unsigned: UnsignedEnvelope,
): Uint8Array<ArrayBuffer> {
  const canonical = new TextEncoder().encode(canonicalize(unsigned))
  const input = new Uint8Array(PREFIX.length + canonical.length)

  input.set(PREFIX)
  input.set(canonical, PREFIX.length)

  return input
}

/**
 * Signs an event with the identity's key.
 *
 * @param identity - Who signs.
 * @param eventType - The event type, e.g. `review.created`.
 * @param payload - The event's payload.
 * @param createdAt - RFC 3339 UTC, whole seconds.
 * @returns The envelope, with its id and signature.
 */
export async function signEnvelope(
  identity: Identity,
  eventType: string,
  payload: JsonObject,
  createdAt: string,
): Promise<Envelope> {
  const unsigned: UnsignedEnvelope = {
    protocol: NETWORK,
    version: 1,
    author: identity.did,
    deviceKey: null,
    eventType,
    createdAt,
    payload,
  }

  const input = signingInput(unsigned)
  const digest = new Uint8Array(await crypto.subtle.digest('SHA-256', input))
  const signature = new Uint8Array(
    await crypto.subtle.sign('Ed25519', identity.keys.privateKey, input),
  )

  return {
    protocol: unsigned.protocol,
    version: unsigned.version,
    eventId: `evt_${base32(digest)}`,
    author: unsigned.author,
    deviceKey: unsigned.deviceKey,
    eventType: unsigned.eventType,
    createdAt: unsigned.createdAt,
    payload: unsigned.payload,
    signature: base64url(signature),
  }
}


/**
 * Measures an envelope the way the node does: its JSON, in UTF-8 bytes.
 *
 * @param envelope - The signed envelope.
 * @returns Its size in bytes.
 */
export function sizeOf(envelope: Envelope): number {
  return new TextEncoder().encode(JSON.stringify(envelope)).length
}
