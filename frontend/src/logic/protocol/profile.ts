/**
 * A profile, checked before it is signed: the payload of profile.updated.
 */

// --- IMPORTS ---
import type { CheckedProfile, ProfileFields } from '@/logic/protocol/profile.t'
import { characters } from '@/logic/protocol/review'

// --- GLOBALS ---
export const NAME_MAX = 60
export const BIO_MAX = 500

// --- CODE ---
/**
 * Whether text holds a control character; a bio may hold line feeds.
 *
 * @param text - The text.
 * @param newline - Whether a line feed is allowed.
 * @returns True when it holds one it may not.
 */
function hasControl(text: string, newline: boolean): boolean {
  return [...text].some((char) => {
    const code = char.codePointAt(0) ?? 0
    if (newline && code === 0x0a) return false
    return code <= 0x1f || (code >= 0x7f && code <= 0x9f)
  })
}

/**
 * Checks a profile and builds the payload it signs to. A profile replaces
 * the one before entirely, so an empty field is left out, not kept.
 *
 * @param fields - The display name and bio typed.
 * @returns The payload, or the problem with each field.
 */
export function checkProfile(fields: ProfileFields): CheckedProfile {
  const name = fields.displayName.normalize('NFC').trim()
  const bio = fields.bio.normalize('NFC').trim()
  const problems: Partial<Record<keyof ProfileFields, string>> = {}

  if (characters(name) > NAME_MAX) {
    problems.displayName = `Keep it under ${NAME_MAX} characters.`
  } else if (hasControl(name, false)) {
    problems.displayName = 'A display name fits on one line.'
  }

  if (characters(bio) > BIO_MAX) {
    problems.bio = `Keep it under ${BIO_MAX} characters.`
  } else if (hasControl(bio, true)) {
    problems.bio = 'Remove the invisible control characters.'
  }

  if (Object.keys(problems).length) return { ok: false, problems }

  const payload: Record<string, string> = {}
  if (name) payload.displayName = name
  if (bio) payload.bio = bio

  return { ok: true, payload }
}
