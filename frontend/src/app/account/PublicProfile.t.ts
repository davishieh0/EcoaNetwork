/**
 * Types for PublicProfile.
 */

// --- GLOBALS ---
export type PublicProfileProps = {
  /** The account's handle, offered as a display name to start from. */
  handle: string
}

/** What a published profile says. */
export type ProfileText = {
  displayName?: string
  bio?: string
}
