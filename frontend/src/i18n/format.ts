/**
 * Numbers and dates as the interface's language writes them.
 */

// --- IMPORTS ---
import { currentLanguage } from '@/i18n/i18n'

// --- CODE ---
/**
 * Writes a number with one decimal: "4.5" in English, "4,5" in pt-BR.
 *
 * @param value - The number.
 * @returns The number, with one decimal.
 */
export function decimal(value: number): string {
  const fixed = value.toFixed(1)
  return currentLanguage() === 'pt-BR' ? fixed.replace('.', ',') : fixed
}

/**
 * The locale a date formatter should use for the current language.
 *
 * @returns 'en' or 'pt-BR'.
 */
export function dateLocale(): string {
  return currentLanguage()
}
