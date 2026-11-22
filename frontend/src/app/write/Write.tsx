/**
 * Writing a review: the heading and the composer grid.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'

import { Composer } from '@/app/write/Composer'
import { useWriteData } from '@/app/write/useWriteData'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'

// --- CODE ---
/**
 * Builds the Write screen.
 *
 * @returns The screen, or nothing until the account is known.
 */
export function Write() {
  const data = useWriteData()
  const rise = useRise()

  // RequireAuth only renders this with a complete account
  if (!data.account?.handle) return null

  return (
    <section className="flex min-h-svh items-center">
      <Container className="flex justify-center py-32">
        <div className="flex w-full max-w-3xl flex-col gap-10">
          <motion.h1
            {...rise()}
            className={cn(
              'font-display text-display-m leading-[1.08] font-semibold',
              'tracking-display',
            )}
          >
            Write a review.
          </motion.h1>

          <motion.div
            {...rise(0.08)}
            className="border border-line bg-canvas"
          >
            <Composer data={data} />
          </motion.div>
        </div>
      </Container>
    </section>
  )
}
