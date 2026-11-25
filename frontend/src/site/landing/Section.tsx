/**
 * The landing's section rhythm and heading arrangement.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'

import type {
  SectionHeadingProps,
  SectionProps,
} from '@/site/landing/Section.t'
import { cn } from '@/components/ui/cn'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/** The landing's rhythm, and only the landing's. */

/**
 * Wraps one landing section and sets its surface.
 *
 * @param props - Anchor id, surface, whether to expose the grid, extra
 * classes and the content.
 * @returns The section element.
 */
export function Section({
  id,
  surface = 'dark',
  grid = false,
  className,
  children,
}: SectionProps) {
  return (
    <section
      id={id}
      data-surface={surface === 'light' ? 'light' : undefined}
      className={cn(
        'relative isolate border-t border-structural py-20 md:py-28',
        grid && 'grid-field',
        className,
      )}
    >
      {children}
    </section>
  )
}

/**
 * Renders a section's eyebrow, headline and lede.
 *
 * @param props - Eyebrow, headline, optional lede and extra classes.
 * @returns The header block.
 */
export function SectionHeading({
  eyebrow,
  title,
  lede,
  className,
}: SectionHeadingProps) {
  const reveal = useReveal()

  return (
    <header className={cn('flex flex-col gap-6', className)}>
      {/* The eyebrow speaks in the metadata voice. */}
      <motion.p {...reveal()} className="meta text-muted">
        {eyebrow}
      </motion.p>

      <motion.h2
        {...reveal(0.06)}
        className={cn(
          'max-w-4xl font-display text-display-m font-semibold',
          'tracking-display text-balance',
        )}
      >
        {title}
      </motion.h2>

      {lede ? (
        <motion.p
          {...reveal(0.12)}
          className="reading-measure text-body-l text-secondary text-pretty"
        >
          {lede}
        </motion.p>
      ) : null}
    </header>
  )
}
