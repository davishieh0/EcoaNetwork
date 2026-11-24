/**
 * Types for the reader's moderation choice.
 */

// --- GLOBALS ---
/** Whose labels this reader follows, and whether they act as a moderator. */
export type ModerationChoice = {
  /** `index`: the moderators the index follows; `own`: the list below. */
  follow: 'index' | 'own'
  /** The did:key followed when `follow` is `own`, empty for nobody. */
  moderators: string[]
  /** Shows the actions to label reviews and responses. */
  moderating: boolean
}
