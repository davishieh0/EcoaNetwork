/**
 * The `review.created` payload, built and checked against its schema.
 */

// --- IMPORTS ---
import { normalizeEntity } from '@/logic/protocol/entity'
import type {
  Checked,
  DraftField,
  ReviewDraft,
  ReviewPayload,
} from '@/logic/protocol/review.t'

// --- GLOBALS ---
/** Longest body the schema accepts, in characters. */
export const BODY_MAX = 5000

/** Longest title the schema accepts, in characters. */
export const TITLE_MAX = 120

// the Unicode White_Space set the schema trims, listed as it does
const SPACE =
  '[\\s\\u0085\\u00a0\\u1680\\u2000-\\u200a\\u2028\\u2029' +
  '\\u202f\\u205f\\u3000]'

const EDGES = new RegExp(`^${SPACE}+|${SPACE}+$`, 'gu')

// the line feed, the one control character a body may hold
const NEWLINE = 0x0a

// half of a UTF-16 surrogate pair, standing without the other half
const LONE_SURROGATE =
  /[\ud800-\udbff](?![\udc00-\udfff])|(?<![\ud800-\udbff])[\udc00-\udfff]/

const BROKEN = 'Remove the broken characters pasted in.'

const LANG = /^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$/

// --- CODE ---
/**
 * Counts characters the way the schema's maxLength does, by code point.
 *
 * @param text - The text to measure.
 * @returns Its length in code points.
 */
export function characters(text: string): number {
  return [...text].length
}

/**
 * Finds a C0 or C1 control character, optionally letting newlines pass.
 *
 * @param text - The text to scan.
 * @param newline - Whether a line feed is allowed.
 * @returns Whether the text holds a forbidden control character.
 */
function hasControl(text: string, newline: boolean): boolean {
  return [...text].some((char) => {
    const code = char.codePointAt(0) ?? 0

    if (newline && code === NEWLINE) return false

    return code <= 0x1f || (code >= 0x7f && code <= 0x9f)
  })
}

/**
 * Trims the schema's white space and composes the text to NFC.
 *
 * @param text - The text as typed.
 * @returns The text as it will be signed.
 */
function clean(text: string): string {
  return text.normalize('NFC').replace(EDGES, '')
}

/**
 * Checks the draft and builds the payload it signs to.
 *
 * @param draft - What the user typed.
 * @returns The payload, or the problem with each field that has one.
 */
export function checkReview(draft: ReviewDraft): Checked {
  const problems: Partial<Record<DraftField, string>> = {}

  const entity = normalizeEntity(draft.kind, draft.identifier)
  if (!entity.ok) problems.identifier = entity.reason

  if (draft.rating === null) problems.rating = 'Pick a rating from 1 to 5.'

  const body = clean(draft.body)
  if (!body) {
    problems.body = 'Say what happened.'
  } else if (characters(body) > BODY_MAX) {
    problems.body = `Keep it under ${BODY_MAX} characters.`
  } else if (hasControl(body, true)) {
    problems.body = 'Remove the invisible control characters.'
  } else if (LONE_SURROGATE.test(body)) {
    problems.body = BROKEN
  }

  const title = clean(draft.title)
  if (characters(title) > TITLE_MAX) {
    problems.title = `Keep the title under ${TITLE_MAX} characters.`
  } else if (hasControl(title, false)) {
    problems.title = 'The title fits on one line.'
  } else if (LONE_SURROGATE.test(title)) {
    problems.title = BROKEN
  }

  if (!entity.ok || draft.rating === null || Object.keys(problems).length) {
    return { ok: false, problems }
  }

  const payload: ReviewPayload = {
    entity: entity.id,
    rating: draft.rating,
    body,
  }

  if (title) payload.title = title
  if (LANG.test(draft.lang)) payload.lang = draft.lang

  return { ok: true, payload }
}
