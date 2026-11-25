/**
 * The events API: where a signed envelope goes to the network, and where
 * one is read back alone, for inspection.
 */

// --- IMPORTS ---
import type { EventItem, PublishResult } from '@/logic/datasource/events.t'
import { BASE, request, route } from '@/logic/datasource/http'
import { errorOf } from '@/logic/errors/envelope'
import type { ErrorEnvelope } from '@/logic/errors/envelope.t'
import { HttpError } from '@/logic/errors/httpError'
import type { Envelope } from '@/logic/protocol/envelope.t'

// --- GLOBALS ---
// the statuses that carry a PublishResult; a rejection is one of them
const RESULT_STATUSES = [200, 202, 400]

const OUTCOMES = ['accepted', 'already_exists', 'orphan', 'rejected']

// --- CODE ---
/**
 * Checks that a body is a PublishResult.
 *
 * @param body - The parsed JSON.
 * @returns Whether it has the shape.
 */
function isResult(body: unknown): body is PublishResult {
  if (typeof body !== 'object' || body === null) return false

  const { outcome, reason } = body as Record<string, unknown>

  return (
    typeof outcome === 'string' &&
    OUTCOMES.includes(outcome) &&
    (reason === null || typeof reason === 'string')
  )
}

/**
 * Checks that a body is an ErrorEnvelope.
 *
 * @param body - The parsed JSON.
 * @returns Whether it has the shape.
 */
function isErrorEnvelope(body: unknown): body is ErrorEnvelope {
  if (typeof body !== 'object' || body === null) return false

  const { error, message } = body as Record<string, unknown>

  return (
    typeof error === 'string' &&
    (typeof message === 'string' ||
      (Array.isArray(message) &&
        message.every((item) => typeof item === 'string')))
  )
}

/**
 * Submits a signed event through the gateway.
 *
 * @param envelope - The signed envelope, sent as it is.
 * @returns What the node did with it; a rejection is a result, not an error.
 */
export async function publishEvent(envelope: Envelope): Promise<PublishResult> {
  let response: Response

  try {
    response = await fetch(`${BASE}/events`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(envelope),
    })
  } catch {
    throw new HttpError(0, 'network_error', 'The backend did not answer.')
  }

  if (!RESULT_STATUSES.includes(response.status)) throw await errorOf(response)

  const body: unknown = await response
    .clone()
    .json()
    .catch(() => null)

  if (isResult(body)) return body

  if (response.status === 400 && isErrorEnvelope(body)) {
    throw await errorOf(response)
  }

  throw new HttpError(response.status, 'unknown_error', 'Unreadable result.')
}

/**
 * Reads one signed event alone, whatever its type, for inspection. A review
 * its author withdrew is still here: the envelope stays on the logs.
 *
 * @param eventId - The eventId.
 * @returns The event.
 */
export function getEvent(eventId: string) {
  return request<EventItem>(route(['events', eventId]))
}
