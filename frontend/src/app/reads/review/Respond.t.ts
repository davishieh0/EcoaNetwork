/**
 * Types for Respond.
 */

// --- GLOBALS ---
export type RespondProps = {
  /** Who the response being answered is by; null answers the review. */
  answering: string | null
  busy: boolean
  onRespond: (body: string) => Promise<boolean>
  onCancel: () => void
}
