/**
 * The one door to the backend.
 */

// --- IMPORTS ---
import type { RequestOptions } from '@/logic/datasource/http.t'
import { errorOf } from '@/logic/errors/envelope'
import { HttpError } from '@/logic/errors/httpError'

// --- GLOBALS ---
export const BASE = import.meta.env.VITE_API_URL ?? '/api'

// --- CODE ---
// several requests can expire at once: they all wait on the same renewal
let renewing: Promise<boolean> | null = null

// told when a session can no longer be renewed
const lostListeners = new Set<() => void>()

/**
 * Subscribes to sessions that end on their own, when renewal fails.
 *
 * @param listener - Called once per session lost.
 * @returns The function that unsubscribes.
 */
export function onSessionLost(listener: () => void) {
  lostListeners.add(listener)
  return () => lostListeners.delete(listener)
}

/**
 * Renews the session with the refresh token, once for every caller waiting.
 *
 * @returns Whether the session could be renewed.
 */
function renew() {
  renewing ??= request<void>('/auth/refresh', { method: 'POST', retry: false })
    .then(() => true)
    .catch(() => false)
    .finally(() => {
      renewing = null
    })

  return renewing
}

/**
 * Calls the backend, with the session cookies.
 *
 * @param path - The route under the API prefix, e.g. `/auth/me`.
 * @param options - Method, JSON body, and whether to renew on expiry.
 * @returns The JSON answer, or undefined when there is none.
 */
export async function request<T>(
  path: string,
  { method = 'GET', body, retry = true }: RequestOptions = {},
): Promise<T> {
  let response: Response

  try {
    response = await fetch(`${BASE}${path}`, {
      method,
      credentials: 'include',
      headers: body === undefined ? {} : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new HttpError(0, 'network_error', 'The backend did not answer.')
  }

  if (response.ok) {
    const text = await response.text()
    return (text ? JSON.parse(text) : undefined) as T
  }

  const error = await errorOf(response)

  // the session outlived its access token: renew it and try once more
  if (retry && error.code === 'access_token_expired_error') {
    if (await renew()) return request<T>(path, { method, body, retry: false })

    // the refresh token is gone too: the session is over
    lostListeners.forEach((listener) => listener())
  }

  throw error
}
