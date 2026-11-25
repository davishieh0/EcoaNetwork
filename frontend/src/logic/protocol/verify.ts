/**
 * Checking a signed envelope in the browser: its eventId and its signature.
 *
 * WebCrypto verifies Ed25519 under RFC 8032's rules; the protocol verifies
 * under ZIP-215. They agree on every signature an honest client makes, and
 * differ only on ones built for the edge cases, which this check could call
 * invalid although every node accepts them. When `protocol/ts` exists, this
 * file verifies through it, and nothing else changes
 * (frontend/docs/reading.md, decision 1).
 */

// --- IMPORTS ---
import {
  base32,
  fromBase58btc,
  fromBase64url,
} from '@/logic/protocol/encoding'
import { signingInput } from '@/logic/protocol/envelope'
import type { UnsignedEnvelope } from '@/logic/protocol/envelope.t'
import { readEnvelope } from '@/logic/protocol/signed'
import type { Verdict } from '@/logic/protocol/verify.t'

// --- GLOBALS ---
// the multicodec of an Ed25519 public key, before its 32 bytes
const ED25519_PREFIX = [0xed, 0x01]

// --- CODE ---
/**
 * Reads the Ed25519 public key a did:key carries.
 *
 * @param did - The did:key.
 * @returns The 32 bytes of the key, or null when it carries none.
 */
function keyOf(did: string): Uint8Array<ArrayBuffer> | null {
  if (!did.startsWith('did:key:z')) return null

  const bytes = fromBase58btc(did.slice('did:key:z'.length))
  if (
    !bytes ||
    bytes.length !== 34 ||
    bytes[0] !== ED25519_PREFIX[0] ||
    bytes[1] !== ED25519_PREFIX[1]
  ) {
    return null
  }

  return new Uint8Array(bytes.slice(2))
}

/**
 * Checks an envelope as the network stored it: that its eventId is the hash
 * of what was signed, and that the signature is its signer's.
 *
 * @param raw - The envelope, the string the index answered.
 * @returns Whether it holds, why not when it does not, or that this browser
 * cannot tell.
 */
export async function verifyEnvelope(raw: string): Promise<Verdict> {
  if (!globalThis.crypto?.subtle) {
    return { ok: null, reason: 'This browser can only check on a secure page.' }
  }

  let envelope
  try {
    envelope = readEnvelope(raw)
  } catch {
    return { ok: false, reason: 'It is not an envelope.' }
  }

  const unsigned = {
    protocol: envelope.protocol,
    version: envelope.version,
    author: envelope.author,
    deviceKey: envelope.deviceKey,
    eventType: envelope.eventType,
    createdAt: envelope.createdAt,
    payload: envelope.payload,
  } as UnsignedEnvelope
  let input
  try {
    input = signingInput(unsigned)
  } catch {
    return { ok: false, reason: 'Its payload cannot be canonicalized.' }
  }

  // the eventId names exactly what was signed
  const digest = new Uint8Array(await crypto.subtle.digest('SHA-256', input))
  if (`evt_${base32(digest)}` !== envelope.eventId) {
    return { ok: false, reason: 'Its eventId is not the hash of what it says.' }
  }

  // a device signs for its identity when it names one
  const signer = (envelope.deviceKey as string | null) ?? envelope.author
  const key = keyOf(signer)
  if (!key) return { ok: false, reason: 'Its signer is not an Ed25519 key.' }

  const signature = fromBase64url(envelope.signature)
  if (!signature || signature.length !== 64) {
    return { ok: false, reason: 'Its signature is malformed.' }
  }

  try {
    const publicKey = await crypto.subtle.importKey(
      'raw',
      key,
      { name: 'Ed25519' },
      false,
      ['verify'],
    )
    const holds = await crypto.subtle.verify(
      'Ed25519',
      publicKey,
      new Uint8Array(signature),
      input,
    )

    return holds
      ? { ok: true }
      : { ok: false, reason: 'The signature is not its signer’s.' }
  } catch {
    // an older browser without Ed25519 in WebCrypto: it cannot tell
    return { ok: null, reason: 'This browser cannot check Ed25519 signatures.' }
  }
}
