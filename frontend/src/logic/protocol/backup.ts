/**
 * An identity's encrypted backup: its private key sealed with a password,
 * in a file only its owner keeps. Without the file and the password, nobody
 * signs as the identity again once this browser forgets it.
 */

// --- IMPORTS ---
import { base64url, fromBase64url } from '@/logic/protocol/encoding'
import type { Backup, OpenedBackup } from '@/logic/protocol/backup.t'

// --- GLOBALS ---
export const BACKUP_FORMAT = 'ecoa-identity-backup'

// slow on purpose: each guess at the password costs this much work
const ITERATIONS = 600_000

// --- CODE ---
/**
 * A failure already worded for the screen.
 */
export class BackupError extends Error {
  /**
   * @param message - What the screen says.
   */
  constructor(message: string) {
    super(message)
    this.name = 'BackupError'
  }
}

/**
 * Derives the AES key a password and salt give.
 *
 * @param password - The backup's password.
 * @param salt - Its salt.
 * @param iterations - How much work each guess costs.
 * @returns The key.
 */
async function keyOf(
  password: string,
  salt: Uint8Array<ArrayBuffer>,
  iterations: number,
): Promise<CryptoKey> {
  const material = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(password.normalize('NFC')),
    'PBKDF2',
    false,
    ['deriveKey'],
  )

  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', hash: 'SHA-256', salt, iterations },
    material,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt'],
  )
}

/**
 * Seals a private key with a password.
 *
 * @param pkcs8 - The private key, as PKCS #8.
 * @param password - The password.
 * @param did - The identity it is the key of, written in the clear.
 * @returns The backup file's text.
 */
export async function sealKey(
  pkcs8: Uint8Array<ArrayBuffer>,
  password: string,
  did: string,
): Promise<string> {
  const salt = crypto.getRandomValues(new Uint8Array(16))
  const iv = crypto.getRandomValues(new Uint8Array(12))
  const key = await keyOf(password, salt, ITERATIONS)

  const sealed = new Uint8Array(
    await crypto.subtle.encrypt(
      // the identity is bound in: a file edited to another did fails
      { name: 'AES-GCM', iv, additionalData: new TextEncoder().encode(did) },
      key,
      pkcs8,
    ),
  )

  const backup: Backup = {
    format: BACKUP_FORMAT,
    version: 1,
    did,
    createdAt: new Date().toISOString(),
    kdf: {
      name: 'PBKDF2',
      hash: 'SHA-256',
      iterations: ITERATIONS,
      salt: base64url(salt),
    },
    cipher: { name: 'AES-GCM', iv: base64url(iv) },
    privateKey: base64url(sealed),
  }

  return JSON.stringify(backup, null, 2)
}

/**
 * Opens a backup with its password.
 *
 * @param text - The backup file's text.
 * @param password - The password.
 * @returns The identity and its private key, as PKCS #8.
 * @throws BackupError - When the file is not a backup, or the password
 * does not open it.
 */
export async function openKey(
  text: string,
  password: string,
): Promise<OpenedBackup> {
  let backup: Backup
  try {
    backup = JSON.parse(text) as Backup
  } catch {
    throw new BackupError('That file is not an Ecoa identity backup.')
  }

  const salt = fromBase64url(backup?.kdf?.salt ?? '')
  const iv = fromBase64url(backup?.cipher?.iv ?? '')
  const sealed = fromBase64url(backup?.privateKey ?? '')

  if (
    backup?.format !== BACKUP_FORMAT ||
    backup.version !== 1 ||
    typeof backup.did !== 'string' ||
    !salt ||
    !iv ||
    !sealed ||
    !Number.isInteger(backup.kdf.iterations)
  ) {
    throw new BackupError('That file is not an Ecoa identity backup.')
  }

  const key = await keyOf(
    password,
    new Uint8Array(salt),
    backup.kdf.iterations,
  )

  try {
    const pkcs8 = await crypto.subtle.decrypt(
      {
        name: 'AES-GCM',
        iv: new Uint8Array(iv),
        additionalData: new TextEncoder().encode(backup.did),
      },
      key,
      new Uint8Array(sealed),
    )

    return { did: backup.did, pkcs8: new Uint8Array(pkcs8) }
  } catch {
    throw new BackupError('The password does not open this backup.')
  }
}
