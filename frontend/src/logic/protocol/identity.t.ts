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
