/**
 * Types for checking an envelope.
 */

// --- GLOBALS ---
/**
 * Whether an envelope holds, and why not when it does not; `ok: null` when
 * this browser cannot tell, which says nothing about the envelope.
 */
export type Verdict =
  | { ok: true }
  | { ok: false; reason: string }
  | { ok: null; reason: string }
