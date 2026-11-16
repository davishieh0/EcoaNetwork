/**
 * Types for the Tooltip.
 */

// --- IMPORTS ---
import type { ReactNode } from 'react'

// --- GLOBALS ---
export type TooltipProps = {
  /** Accessible name of the trigger. */
  label: string
  children: ReactNode
}
