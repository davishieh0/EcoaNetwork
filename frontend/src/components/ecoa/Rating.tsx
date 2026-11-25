/**
 * A star rating, in ink. Information, not a verdict.
 */

// --- IMPORTS ---
import { useTranslation } from 'react-i18next'
import type { RatingProps } from '@/components/ecoa/Rating.t'
import { cn } from '@/components/ui/cn'
import { decimal } from '@/i18n/format'

// --- CODE ---
/**
 * Renders a rating as stars plus a number.
 *
 * @param props - The rating, and extra classes.
 * @returns The stars and their numeric reading.
 */
export function Rating({ value, className }: RatingProps) {
  // subscribes to the language, so the decimal follows it
  useTranslation()
  const rounded = Math.round(value)

  return (
    <p className={cn('flex items-center gap-2', className)}>
      <span aria-hidden="true" className="text-body-m tracking-[0.1em]">
        {'★'.repeat(rounded)}
        <span className="text-muted">{'☆'.repeat(5 - rounded)}</span>
      </span>
      <span className="meta text-secondary">{decimal(value)} / 5</span>
    </p>
  )
}
