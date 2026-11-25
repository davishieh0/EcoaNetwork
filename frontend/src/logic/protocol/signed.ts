/**
 * Reading a signed envelope the index handed out, to render it.
 */

// --- IMPORTS ---
import type { Envelope } from '@/logic/protocol/envelope.t'
import type { ReviewText } from '@/logic/protocol/signed.t'

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
    typeof (parsed as Envelope).payload !== 'object' ||
    (parsed as Envelope).payload === null
  ) {
    throw new Error('Not an envelope.')
  }

  return parsed as Envelope
}

/**
 * What a review or one of its versions says, read from its payload.
 *
 * @param envelope - A `review.created` or `review.replaced`.
 * @returns Its rating, title, body and language.
 */
export function reviewTextOf(envelope: Envelope): ReviewText {
  const { rating, title, body, lang, entity, replaces } = envelope.payload

  return {
    rating: typeof rating === 'number' ? rating : 0,
    title: typeof title === 'string' ? title : undefined,
    body: typeof body === 'string' ? body : '',
    lang: typeof lang === 'string' ? lang : undefined,
    entity: typeof entity === 'string' ? entity : undefined,
    replaces: typeof replaces === 'string' ? replaces : undefined,
  }
}

/**
 * A text field of a payload, when it is one.
 *
 * @param envelope - Any envelope.
 * @param field - The payload field, such as `displayName`.
 * @returns The text, or undefined.
 */
export function textOf(envelope: Envelope, field: string): string | undefined {
  const value = envelope.payload[field]
  return typeof value === 'string' ? value : undefined
}
