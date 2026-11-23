/**
 * Types for the Select.
 */

// --- GLOBALS ---
/** One choice in the list. */
export type SelectOption<T extends string> = {
  value: T
  label: string
}

export type SelectProps<T extends string> = {
  /** Accessible name of the control. */
  label: string
  value: T
  options: SelectOption<T>[]
  onChange: (value: T) => void
  className?: string
}
