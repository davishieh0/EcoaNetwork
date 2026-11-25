/**
 * The identities this browser signs with, kept in IndexedDB with private
 * keys nobody can read out:
 *
 * - the main identity, created on purpose, with an encrypted backup when a
 *   password is given: the key is exportable for an instant, sealed into a
 *   file for its owner, and kept non-extractable;
 * - fresh identities, made for one review each, kept so their author can
 *   edit or withdraw it, never backed up.
 */

// --- IMPORTS ---
import { BackupError, openKey, sealKey } from '@/logic/protocol/backup'
import { base58btc, fromBase58btc } from '@/logic/protocol/encoding'
import type {
  Did,
  Identity,
  IdentityInfo,
  KnownIdentity,
} from '@/logic/protocol/identity.t'

// --- GLOBALS ---
const DATABASE = 'ecoa'

const STORE = 'identity'

// the main identity's record, and what this browser knows about it
const RECORD = 'browser'
const META = 'browser:meta'

// a fresh identity's record is this and its did:key
const FRESH = 'fresh:'

// the multicodec prefix of an Ed25519 public key, 0xed as a varint
const ED25519_PREFIX = [0xed, 0x01]

// --- CODE ---
/**
 * There is no identity in this browser yet: one has to be created or
 * restored before anything is signed.
 */
export class NoIdentityError extends Error {
  constructor() {
    super('This browser has no identity yet. Create or restore one.')
    this.name = 'NoIdentityError'
  }
}

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
 * Runs work against the open database, closing it after.
 *
 * @param work - What to do with it.
 * @returns What the work returned.
 */
async function withDatabase<T>(
  work: (database: IDBDatabase) => Promise<T>,
): Promise<T> {
  if (!globalThis.crypto?.subtle || !globalThis.indexedDB) {
    throw new Error('This browser cannot sign: it needs a secure page.')
  }

  const database = await open()
  try {
    return await work(database)
  } finally {
    database.close()
  }
}

/**
 * Reads one record.
 *
 * @param database - The open database.
 * @param key - The record's key.
 * @returns What it holds, or undefined.
 */
function get<T>(database: IDBDatabase, key: string): Promise<T | undefined> {
  const store = database.transaction(STORE).objectStore(STORE)
  return settled<T | undefined>(store.get(key))
}

/**
 * Writes one record.
 *
 * @param database - The open database.
 * @param key - The record's key.
 * @param value - What it holds.
 * @returns Nothing.
 */
async function put(database: IDBDatabase, key: string, value: unknown) {
  const store = database.transaction(STORE, 'readwrite').objectStore(STORE)
  await settled(store.put(value, key))
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
 * Reads the public key a did:key carries.
 *
 * @param did - The did:key.
 * @returns The public key.
 */
async function publicKeyOf(did: string): Promise<CryptoKey> {
  const bytes = fromBase58btc(did.slice('did:key:z'.length))
  if (!bytes || bytes.length !== 34 || bytes[0] !== 0xed || bytes[1] !== 1) {
    throw new BackupError('The backup names an identity that is not Ed25519.')
  }

  return crypto.subtle.importKey(
    'raw',
    new Uint8Array(bytes.slice(2)),
    { name: 'Ed25519' },
    true,
    ['verify'],
  )
}

/**
 * Says which identity this browser signs with, without making one: a page
 * that only reads must not leave a key behind.
 *
 * @returns The did:key, or null when this browser has none yet.
 */
export async function peekIdentity(): Promise<Did | null> {
  if (!globalThis.crypto?.subtle || !globalThis.indexedDB) return null

  return withDatabase(async (database) => {
    const keys = await get<CryptoKeyPair>(database, RECORD)
    return keys ? didOf(keys.publicKey) : null
  })
}

/**
 * What this browser knows about its main identity.
 *
 * @returns Its did:key, and whether a backup was made, or null.
 */
export async function identityInfo(): Promise<IdentityInfo | null> {
  if (!globalThis.crypto?.subtle || !globalThis.indexedDB) return null

  return withDatabase(async (database) => {
    const keys = await get<CryptoKeyPair>(database, RECORD)
    if (!keys) return null

    const meta = await get<Omit<IdentityInfo, 'did'>>(database, META)
    return {
      did: await didOf(keys.publicKey),
      backedUp: meta?.backedUp ?? false,
      createdAt: meta?.createdAt ?? null,
    }
  })
}

/**
 * Loads this browser's main identity.
 *
 * @returns The identity.
 * @throws NoIdentityError - When none was created or restored yet.
 */
export async function loadIdentity(): Promise<Identity> {
  return withDatabase(async (database) => {
    const keys = await get<CryptoKeyPair>(database, RECORD)
    if (!keys) throw new NoIdentityError()

    return { did: await didOf(keys.publicKey), keys }
  })
}

/**
 * Loads the identity that signs as did: the main one or a fresh one.
 *
 * @param did - The did:key.
 * @returns The identity.
 * @throws NoIdentityError - When this browser does not hold it.
 */
export async function identityFor(did: string): Promise<Identity> {
  return withDatabase(async (database) => {
    const main = await get<CryptoKeyPair>(database, RECORD)
    if (main && (await didOf(main.publicKey)) === did) {
      return { did: did as Did, keys: main }
    }

    const fresh = await get<CryptoKeyPair>(database, FRESH + did)
    if (fresh) return { did: did as Did, keys: fresh }

    throw new NoIdentityError()
  })
}

/**
 * Every identity this browser can sign as.
 *
 * @returns The main one first, then the fresh ones.
 */
export async function knownIdentities(): Promise<KnownIdentity[]> {
  if (!globalThis.crypto?.subtle || !globalThis.indexedDB) return []

  return withDatabase(async (database) => {
    const store = database.transaction(STORE).objectStore(STORE)
    const keys = (await settled(store.getAllKeys())) as string[]
    const known: KnownIdentity[] = []

    const main = await get<CryptoKeyPair>(database, RECORD)
    if (main) known.push({ did: await didOf(main.publicKey), kind: 'main' })

    for (const key of keys) {
      if (key.startsWith(FRESH)) {
        known.push({ did: key.slice(FRESH.length) as Did, kind: 'fresh' })
      }
    }

    return known
  })
}

/**
 * Creates this browser's main identity. With a password, the key is made
 * exportable for an instant, sealed into a backup, and kept non-extractable;
 * without one, it is non-extractable from the start, and lost with this
 * browser's data.
 *
 * @param password - The backup's password, or null for no backup.
 * @returns The identity, and the backup file's text when one was made.
 */
export async function createIdentity(
  password: string | null,
): Promise<{ identity: Identity; backup: string | null }> {
  return withDatabase(async (database) => {
    if (await get<CryptoKeyPair>(database, RECORD)) {
      throw new Error('This browser already has an identity.')
    }

    if (!password) {
      const keys = await crypto.subtle.generateKey({ name: 'Ed25519' }, false, [
        'sign',
        'verify',
      ])
      const did = await didOf(keys.publicKey)
      await put(database, RECORD, keys)
      await put(database, META, {
        backedUp: false,
        createdAt: new Date().toISOString(),
      })

      return { identity: { did, keys }, backup: null }
    }

    // exportable for as long as it takes to seal a copy
    const made = await crypto.subtle.generateKey({ name: 'Ed25519' }, true, [
      'sign',
      'verify',
    ])
    const did = await didOf(made.publicKey)
    const pkcs8 = new Uint8Array(
      await crypto.subtle.exportKey('pkcs8', made.privateKey),
    )
    const backup = await sealKey(pkcs8, password, did)

    const keys = {
      privateKey: await crypto.subtle.importKey(
        'pkcs8',
        pkcs8,
        { name: 'Ed25519' },
        false,
        ['sign'],
      ),
      publicKey: made.publicKey,
    }
    pkcs8.fill(0)

    await put(database, RECORD, keys)
    await put(database, META, {
      backedUp: true,
      createdAt: new Date().toISOString(),
    })

    return { identity: { did, keys }, backup }
  })
}

/**
 * Restores the main identity from its backup, kept non-extractable.
 *
 * @param text - The backup file's text.
 * @param password - Its password.
 * @returns The identity.
 * @throws BackupError - When the file is not a backup, the password does
 * not open it, or the key is not the identity's.
 */
export async function restoreIdentity(
  text: string,
  password: string,
): Promise<Identity> {
  const opened = await openKey(text, password)

  return withDatabase(async (database) => {
    if (await get<CryptoKeyPair>(database, RECORD)) {
      throw new BackupError('This browser already has an identity.')
    }

    const privateKey = await crypto.subtle.importKey(
      'pkcs8',
      opened.pkcs8,
      { name: 'Ed25519' },
      false,
      ['sign'],
    )
    opened.pkcs8.fill(0)
    const publicKey = await publicKeyOf(opened.did)

    // the key must be the identity's: sign, and check with the did's key
    const probe = new TextEncoder().encode('ecoa-restore-check')
    const signature = await crypto.subtle.sign('Ed25519', privateKey, probe)
    const holds = await crypto.subtle.verify(
      'Ed25519',
      publicKey,
      signature,
      probe,
    )
    if (!holds) {
      throw new BackupError('The key in this backup is not its identity’s.')
    }

    const keys = { privateKey, publicKey }
    await put(database, RECORD, keys)
    await put(database, META, {
      backedUp: true,
      createdAt: new Date().toISOString(),
    })

    return { did: opened.did as Did, keys }
  })
}

/**
 * Makes a fresh identity for one review: unlinked to the main one, kept so
 * its author can edit or withdraw that review, and never backed up.
 *
 * @returns The fresh identity.
 */
export async function createFresh(): Promise<Identity> {
  return withDatabase(async (database) => {
    const keys = await crypto.subtle.generateKey({ name: 'Ed25519' }, false, [
      'sign',
      'verify',
    ])
    const did = await didOf(keys.publicKey)
    await put(database, FRESH + did, keys)

    return { did, keys }
  })
}
