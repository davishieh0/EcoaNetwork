/**
 * The identity this browser signs with: one Ed25519 key pair, kept in
 * IndexedDB with a private key nobody can read out.
 */

// --- IMPORTS ---
import { base58btc } from '@/logic/protocol/encoding'
import type { Did, Identity } from '@/logic/protocol/identity.t'

// --- GLOBALS ---
const DATABASE = 'ecoa'

const STORE = 'identity'

// the one record the store holds
const RECORD = 'browser'

// the multicodec prefix of an Ed25519 public key, 0xed as a varint
const ED25519_PREFIX = [0xed, 0x01]

// one load at a time, for every caller in this tab
let loading: Promise<Identity> | null = null

// --- CODE ---
/**
 * Waits for an IndexedDB request.
 *
 * @param request - The request to wait for.
 * @returns What the request produced.
 */
function settled<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

/**
 * Opens the database, creating the store the first time.
 *
 * @returns The open database.
 */
function open(): Promise<IDBDatabase> {
  const request = indexedDB.open(DATABASE, 1)
  request.onupgradeneeded = () => request.result.createObjectStore(STORE)

  return settled(request)
}

/**
 * Derives the did:key from the public key.
 *
 * @param publicKey - The Ed25519 public key.
 * @returns The `did:key`.
 */
async function didOf(publicKey: CryptoKey): Promise<Did> {
  const raw = new Uint8Array(await crypto.subtle.exportKey('raw', publicKey))
  const bytes = new Uint8Array([...ED25519_PREFIX, ...raw])

  return `did:key:z${base58btc(bytes)}`
}

/**
 * Reads the stored key pair, if there is one.
 *
 * @param database - The open database.
 * @returns The key pair, or undefined.
 */
function read(database: IDBDatabase): Promise<CryptoKeyPair | undefined> {
  const store = database.transaction(STORE).objectStore(STORE)

  return settled<CryptoKeyPair | undefined>(store.get(RECORD))
}

/**
 * Stores a key pair unless another tab stored one first.
 *
 * @param database - The open database.
 * @param keys - The key pair to keep.
 * @returns The key pair that ends up stored: the winner's, if it lost.
 */
async function keep(
  database: IDBDatabase,
  keys: CryptoKeyPair,
): Promise<CryptoKeyPair> {
  const store = database.transaction(STORE, 'readwrite').objectStore(STORE)

  try {
    await settled(store.add(keys, RECORD))

    return keys
  } catch (error) {
    if (!(error instanceof DOMException) || error.name !== 'ConstraintError') {
      throw error
    }

    const winner = await read(database)
    if (!winner) throw error

    return winner
  }
}

/**
 * Loads the identity from the database, creating it on the first use.
 *
 * @returns The identity.
 */
async function load(): Promise<Identity> {
  if (!globalThis.crypto?.subtle) {
    throw new Error('This browser cannot sign: it needs a secure page.')
  }

  const database = await open()

  try {
    const stored = await read(database)

    const keys =
      stored ??
      (await keep(
        database,
        await crypto.subtle.generateKey({ name: 'Ed25519' }, false, [
          'sign',
          'verify',
        ]),
      ))

    return { did: await didOf(keys.publicKey), keys }
  } finally {
    database.close()
  }
}

/**
 * Loads this browser's identity, creating it on the first call ever.
 *
 * @returns The identity; every call in a tab gets the same one.
 */
export function loadIdentity(): Promise<Identity> {
  loading ??= load().catch((error) => {
    loading = null
    throw error
  })

  return loading
}
