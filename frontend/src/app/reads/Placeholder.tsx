/**
 * Where a list's items will be, while they are on their way.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import type { PlaceholderProps } from '@/app/reads/Placeholder.t'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
const BAR = 'bg-surface-raised motion-safe:animate-pulse'

// --- CODE ---
/**
 * Draws a few quiet cards in the shape of what is coming.
 *
 * @param props - How many cards, and how many lines of text each.
 * @returns The placeholder, hidden from assistive technology.
 */
export function Placeholder({ count = 3, lines = 2 }: PlaceholderProps) {
  const reduceMotion = useReducedMotion()

  return (
    <motion.ul
      aria-hidden="true"
      className="grid gap-3"
      initial={reduceMotion ? false : { opacity: 0 }}
      animate={{ opacity: 1 }}
      // a quick answer never flashes it: it fades in only after a beat
      transition={{ duration: 0.3, delay: reduceMotion ? 0 : 0.15 }}
    >
      {Array.from({ length: count }, (_, card) => (
        <li
          key={card}
          className="flex flex-col gap-3 border border-structural p-5"
        >
          <span className={cn(BAR, 'h-3 w-28')} />
          <span className={cn(BAR, 'h-5 w-1/2')} />
          {Array.from({ length: lines }, (_, line) => (
            <span
              key={line}
              className={cn(
                BAR,
                'h-3',
                line === lines - 1 ? 'w-2/3' : 'w-full',
              )}
            />
          ))}
        </li>
      ))}
    </motion.ul>
  )
}
