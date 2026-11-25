/**
 * Types for LabelForm.
 */

// --- GLOBALS ---
export type LabelFormProps = {
  /** What is labeled, in words: "this review", "this response". */
  what: string
  busy: boolean
  onLabel: (label: string, note: string | null) => Promise<boolean>
  onCancel: () => void
}
