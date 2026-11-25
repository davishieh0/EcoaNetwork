/**
 * Signing an event in this browser and handing it to the network, for what
 * a reader does on a review: edit, withdraw, vote, respond.
 */

// --- IMPORTS ---
import { messageOf } from '@/app/forms/messages'
import { publishEvent } from '@/logic/datasource/events'
import { HttpError } from '@/logic/errors/httpError'
import {
  createdAtNow,
  MAX_ENVELOPE,
  signEnvelope,
  sizeOf,
} from '@/logic/protocol/envelope'
import type { Envelope } from '@/logic/protocol/envelope.t'
import {
  NoIdentityError,
  identityFor,
  loadIdentity,
} from '@/logic/protocol/identity'
import type { JsonObject } from '@/logic/protocol/jcs.t'

// --- GLOBALS ---
const NOT_SIGNED = 'This browser could not sign it. Nothing was sent.'
const NO_IDENTITY =
  'This browser has no identity yet: create or restore it first.'
const REFUSED = 'The network refused it. Nothing was published.'
const TOO_LARGE =
  'Signed, it is over the 16 KB the network accepts. Shorten it.'

// --- CODE ---
/**
 * A failure already worded for the screen.
 */
export class PublishFailure extends Error {
  /** There is no identity to sign with: the screen offers to make one. */
  readonly needsIdentity: boolean

  /**
   * @param message - What the screen says.
   * @param needsIdentity - Whether it failed for want of an identity.
   */
  constructor(message: string, needsIdentity = false) {
    super(message)
    this.name = 'PublishFailure'
    this.needsIdentity = needsIdentity
  }
}

/**
 * Signs an event with this browser's identity and publishes it.
 *
 * @param eventType - Such as `review.endorsed`.
 * @param payload - The payload, as it will be signed.
 * @param signer - The did:key to sign as, when not the main identity: the
 * one that wrote what an edit or a withdrawal acts on.
 * @returns The envelope the network took.
 * @throws PublishFailure - When it could not be signed, was too large,
 * did not reach the network, or the network refused it.
 */
export async function publishSigned(
  eventType: string,
  payload: JsonObject,
  signer?: string,
): Promise<Envelope> {
  let envelope: Envelope
  try {
    const identity = signer ? await identityFor(signer) : await loadIdentity()
    envelope = await signEnvelope(identity, eventType, payload, createdAtNow())
  } catch (error) {
    if (error instanceof NoIdentityError) {
      throw new PublishFailure(NO_IDENTITY, true)
    }
    throw new PublishFailure(NOT_SIGNED)
  }

  if (sizeOf(envelope) > MAX_ENVELOPE) throw new PublishFailure(TOO_LARGE)

  try {
    const result = await publishEvent(envelope)
    if (result.outcome === 'rejected') {
      throw new PublishFailure(
        result.reason ? `${REFUSED} Reason: ${result.reason}` : REFUSED,
      )
    }
  } catch (error) {
    if (error instanceof PublishFailure) throw error
    throw new PublishFailure(
      error instanceof HttpError ? messageOf(error) : NOT_SIGNED,
    )
  }

  return envelope
}

/**
 * Words whatever publishing threw.
 *
 * @param error - The failure.
 * @returns The message to show.
 */
export function failureOf(error: unknown): string {
  return error instanceof PublishFailure ? error.message : NOT_SIGNED
}
