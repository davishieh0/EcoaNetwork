/**
 * Types for the LanguageSwitch.
 */

// --- IMPORTS ---
import type { Language } from '@/i18n/i18n.t'

// --- GLOBALS ---
export type LanguageSwitchProps = {
  className?: string
}

export type LanguageOption = {
  value: Language
  /** What the button shows. */
  short: string
  /** The language's own name for itself. */
  name: string
}
