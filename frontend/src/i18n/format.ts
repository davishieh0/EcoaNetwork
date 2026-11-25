/**
 * Dates as the interface's language writes them.
 */

// --- IMPORTS ---
import { currentLanguage } from '@/i18n/i18n'

// --- CODE ---
/**
 * The locale a date formatter should use for the current language.
 *
 * @returns 'en' or 'pt-BR'.
 */
export function dateLocale(): string {
  return currentLanguage()
}
