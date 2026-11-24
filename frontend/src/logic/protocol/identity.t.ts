/**
 * Types for the browser's identity.
 */

// --- GLOBALS ---
/** An identity's `did:key`, as the protocol writes `author`. */
export type Did = `did:key:z${string}`

/** Who signs: the `did:key` and the keys it was made from. */
export type Identity = {
  did: Did
  keys: CryptoKeyPair
}

/** What this browser knows about its main identity. */
export type IdentityInfo = {
  did: Did
  /** A backup file was made when it was created, or it was restored. */
  backedUp: boolean
  createdAt: string | null
}

/** An identity this browser can sign as. */
export type KnownIdentity = {
  did: Did
  kind: 'main' | 'fresh'
}
