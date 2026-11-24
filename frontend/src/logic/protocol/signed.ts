/**
 * Reading a signed envelope the index handed out, to render it.
 */

// --- IMPORTS ---
import type { Envelope } from '@/logic/protocol/envelope.t'

// --- CODE ---
/**
 * Reads the fields of an envelope kept as the string the network stored.
 * The string itself stays as it is, for checking its signature.
 *
 * @param raw - The envelope, as the index answered it.
 * @returns The envelope's fields.
 * @throws Error - When it is not an envelope.
 */
export function readEnvelope(raw: string): Envelope {
  const parsed: unknown = JSON.parse(raw)

  if (
    typeof parsed !== 'object' ||
    parsed === null ||
    typeof (parsed as Envelope).eventId !== 'string' ||
    typeof (parsed as Envelope).author !== 'string' ||
    typeof (parsed as Envelope).eventType !== 'string' ||
    typeof (parsed as Envelope).payload !== 'object'
  ) {
    throw new Error('Not an envelope.')
  }

  return parsed as Envelope
}
