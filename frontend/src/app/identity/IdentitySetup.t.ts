/**
 * Types for IdentitySetup.
 */

// --- GLOBALS ---
export type IdentitySetupProps = {
  /** Called with the did:key once an identity is created or restored. */
  onDone: (did: string) => void
}
