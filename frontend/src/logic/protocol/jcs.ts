/**
 * JSON Canonicalization Scheme (RFC 8785), for the values the protocol signs.
 */

// --- IMPORTS ---
import type { Json } from '@/logic/protocol/jcs.t'

// --- CODE ---
/**
 * Serializes a value the one way RFC 8785 allows.
 *
 * @param value - The value to serialize; numbers must be safe integers.
 * @returns The canonical text.
 */
export function canonicalize(value: Json): string {
  if (typeof value === 'number') {
    if (!Number.isSafeInteger(value)) {
      throw new RangeError('Only safe integers can be signed, no floats.')
    }

    return String(value)
  }

  if (value === null || typeof value !== 'object') {
    return JSON.stringify(value)
  }

  if (Array.isArray(value)) {
    return `[${value.map(canonicalize).join(',')}]`
  }

  // keys sort by UTF-16 code unit, which is what the default sort does
  const members = Object.keys(value)
    .sort()
    .filter((key) => value[key] !== undefined)
    .map((key) => `${JSON.stringify(key)}:${canonicalize(value[key] as Json)}`)

  return `{${members.join(',')}}`
}
