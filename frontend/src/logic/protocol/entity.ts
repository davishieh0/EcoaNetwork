/**
 * Entity identifiers, normalized the way docs/protocol.md requires.
 */

// --- IMPORTS ---
import type {
  EntityId,
  EntityKind,
  Normalized,
} from '@/logic/protocol/entity.t'

// --- GLOBALS ---
// digit counts a GTIN comes in, before padding to GTIN-14
const GTIN_LENGTHS = [8, 12, 13, 14]

// one DNS label, lowercase ASCII
const LABEL = '[a-z0-9]([a-z0-9-]*[a-z0-9])?'

// a host of two or more labels
const DOMAIN = new RegExp(`^${LABEL}(\\.${LABEL})+$`)

const NORMALIZERS: Record<EntityKind, (typed: string) => Normalized> = {
  cnpj: normalizeCnpj,
  domain: normalizeDomain,
  gtin: normalizeGtin,
}

// --- CODE ---
/**
 * Normalizes what was typed into the identifier that gets signed.
 *
 * @param kind - The identifier kind the user picked.
 * @param typed - The identifier as typed.
 * @returns The normalized identifier, or why it cannot be one.
 */
export function normalizeEntity(kind: EntityKind, typed: string): Normalized {
  return NORMALIZERS[kind](typed.trim())
}

/**
 * Keeps the digits of a CNPJ and pads it to 14.
 *
 * @param typed - The CNPJ, with or without punctuation.
 * @returns The `cnpj:` identifier, or why it is not one.
 */
function normalizeCnpj(typed: string): Normalized {
  if (/[^\d./\-\s]/.test(typed)) {
    return { ok: false, reason: 'A CNPJ has only digits.' }
  }

  const digits = typed.replace(/\D/g, '')

  if (!digits) return { ok: false, reason: 'Type the CNPJ.' }

  if (digits.length > 14) {
    return { ok: false, reason: 'A CNPJ has 14 digits.' }
  }

  return { ok: true, id: `cnpj:${digits.padStart(14, '0')}` }
}

/**
 * Keeps the digits of a barcode and pads it to GTIN-14.
 *
 * @param typed - The GTIN-8, -12, -13 or -14.
 * @returns The `gtin:` identifier, or why it is not one.
 */
function normalizeGtin(typed: string): Normalized {
  if (/[^\d\s]/.test(typed)) {
    return { ok: false, reason: 'A barcode has only digits.' }
  }

  const digits = typed.replace(/\s/g, '')

  if (!digits) return { ok: false, reason: 'Type the barcode number.' }

  if (!GTIN_LENGTHS.includes(digits.length)) {
    return { ok: false, reason: 'A barcode has 8, 12, 13 or 14 digits.' }
  }

  return { ok: true, id: `gtin:${digits.padStart(14, '0')}` }
}

/**
 * Reduces an address to its lowercase host, IDN as punycode, no `www.`.
 *
 * @param typed - A domain or a full address.
 * @returns The `domain:` identifier, or why it is not one.
 */
function normalizeDomain(typed: string): Normalized {
  if (!typed) return { ok: false, reason: 'Type the website address.' }

  const withScheme = /^[a-z][a-z\d+.-]*:\/\//i.test(typed)
    ? typed
    : `http://${typed}`

  let host: string

  try {
    host = new URL(withScheme).hostname
  } catch {
    return { ok: false, reason: 'That is not a website address.' }
  }

  host = host.replace(/\.$/, '').replace(/^(www\.)+/, '')

  if (!DOMAIN.test(host)) {
    return { ok: false, reason: 'That is not a website address.' }
  }

  return { ok: true, id: `domain:${host}` as EntityId }
}
