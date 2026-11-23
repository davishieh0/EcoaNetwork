/**
 * Types for the Field.
 */

// --- IMPORTS ---
import type { ComponentPropsWithoutRef, ReactNode } from 'react'

// --- GLOBALS ---
export type FieldProps = {
  label: string
  /** Guidance shown under the control, replaced by `error` when set. */
  hint?: string
  error?: string
  /** Offers a visibility toggle when the input type is password. */
  revealPassword?: boolean
  /** Classes for the wrapper; the control keeps its own. */
  className?: string
  /** Overlays the control, for a suggestion list. */
  children?: ReactNode
} & Omit<ComponentPropsWithoutRef<'input'>, 'className'>
