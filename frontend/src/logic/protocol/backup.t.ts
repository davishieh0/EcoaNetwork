/**
 * Types for identity backups.
 */

// --- GLOBALS ---
/** The backup file: the private key sealed, and how to open it. */
export type Backup = {
  format: string
  version: 1
  did: string
  createdAt: string
  kdf: { name: 'PBKDF2'; hash: 'SHA-256'; iterations: number; salt: string }
  cipher: { name: 'AES-GCM'; iv: string }
  /** The PKCS #8 private key, sealed with AES-GCM. */
  privateKey: string
}

/** A backup opened: the identity, and its private key. */
export type OpenedBackup = {
  did: string
  pkcs8: Uint8Array<ArrayBuffer>
}
