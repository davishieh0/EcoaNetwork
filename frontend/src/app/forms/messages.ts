/**
 * What each failure the backend reports means to a person.
 */

// --- IMPORTS ---
import { HttpError } from '@/logic/errors/httpError'

// --- GLOBALS ---
const MESSAGES: Record<string, string> = {
  invalid_credentials_error: 'That email and password do not match.',
  email_already_registered_error:
    'An account already uses this email. Sign in instead.',
  handle_taken_error: 'That handle is taken. Try another.',
  invalid_password_error: 'The password does not meet every rule below.',
  invalid_code_error: 'That code is not right. Check it and try again.',
  expired_code_error: 'That code has expired. Ask for a new one.',
  too_many_requests_error: 'Too many attempts. Wait a moment and try again.',
  invalid_session_error: 'Your session ended. Sign in again.',
  not_authenticated_error: 'Your session ended. Sign in again.',
  access_token_expired_error: 'Your session ended. Sign in again.',
  profile_exists_error: 'This account already has a handle.',
  invalid_request_error:
    'The review could not be read by the gateway. Check it and try again.',
  node_unavailable_error: 'No network node answered right now. Try again soon.',
}

const UNAVAILABLE = 'Something on our side is not answering. Try again soon.'

// --- CODE ---
/**
 * Words a failure for the screen.
 *
 * @param error - Whatever the request threw.
 * @returns The message to show.
 */
export function messageOf(error: unknown) {
  if (!(error instanceof HttpError)) return UNAVAILABLE

  // a request the form should have caught: say so plainly
  if (error.code === 'request_validation_error') {
    return 'Some of the details are not valid. Check them and try again.'
  }

  return MESSAGES[error.code] ?? UNAVAILABLE
}
