/**
 * Types for profiles.
 */

// --- GLOBALS ---
/** What a profile is typed as. */
export type ProfileFields = {
  displayName: string
  bio: string
}

/** What checking a profile gave. */
export type CheckedProfile =
  | { ok: true; payload: Record<string, string> }
  | { ok: false; problems: Partial<Record<keyof ProfileFields, string>> }
