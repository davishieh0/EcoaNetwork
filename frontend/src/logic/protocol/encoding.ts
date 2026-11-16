/**
 * The byte encodings the protocol uses.
 */

// --- GLOBALS ---
// Bitcoin's base58 alphabet, the one multibase `z` names
const BASE58 =
  '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz'

// --- CODE ---

/**
 * Encodes bytes as base58btc, without the multibase prefix.
 *
 * @param bytes - The bytes to encode.
 * @returns The text.
 */
export function base58btc(bytes: Uint8Array): string {
  // the number the bytes spell, as base-58 digits, least significant first
  const digits: number[] = []

  for (const byte of bytes) {
    let carry = byte

    for (let index = 0; index < digits.length; index++) {
      carry += digits[index] * 256
      digits[index] = carry % 58
      carry = Math.floor(carry / 58)
    }

    while (carry > 0) {
      digits.push(carry % 58)
      carry = Math.floor(carry / 58)
    }
  }

  // each leading zero byte is written as its own first character
  let text = ''
  for (const byte of bytes) {
    if (byte !== 0) break
    text += BASE58[0]
  }

  for (let index = digits.length - 1; index >= 0; index--) {
    text += BASE58[digits[index]]
  }

  return text
}
