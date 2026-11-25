/**
 * What each failure the backend reports means to a person.
 */

// --- IMPORTS ---
import { i18n } from '@/i18n/i18n'
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
  indexer_unavailable_error:
    'The index is not answering right now. Try again soon.',
  network_error: 'Ecoa is not answering. Check your connection and try again.',
  entity_not_found_error: 'Nothing on the network names this entity yet.',
  review_not_found_error: 'No review with this id is on the network.',
  event_not_found_error: 'No event with this id is on the network.',
  profile_not_found_error: 'This identity has no profile.',
  review_withdrawn_error: 'Its author withdrew this review.',
  removed_by_moderation_error:
    'A moderator this index follows removed this, as illegal or as ' +
    'personal data.',
  invalid_event_id_error: 'That is not an event id.',
  invalid_entity_id_error: 'That is not an entity id.',
  invalid_identity_error: 'That is not an identity.',
  invalid_query_error: 'The search is too long. Use up to 120 characters.',
  invalid_cursor_error: 'The list changed. Load it again from the start.',
  invalid_captcha_error:
    'That human check did not verify. Try solving it again.',
  verifier_disabled_error: 'This gateway does not issue human claims.',
  invalid_category_error: 'That is not a category the protocol knows.',
  invalid_moderators_error: 'One of the moderator keys is not a valid did:key.',
  invalid_labeled_error: 'Invalid moderation filter option.',
  invalid_limit_error: 'Invalid page size requested.',
  email_not_verified_error: 'Confirm your email address before signing in.',
  profile_incomplete_error:
    'Choose a handle first to finish setting up your account.',
  auth_provider_error: 'The authentication service is not answering right now.',
  database_unavailable_error: 'The gateway database is unavailable right now.',
  internal_error: 'Something went wrong on our side. Try again soon.',
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
  if (!(error instanceof HttpError)) return i18n.t(UNAVAILABLE)

  // a request the form should have caught: say so plainly
  if (error.code === 'request_validation_error') {
    return i18n.t(
      'Some of the details are not valid. Check them and try again.',
    )
  }

  return i18n.t(MESSAGES[error.code] ?? UNAVAILABLE)
}
