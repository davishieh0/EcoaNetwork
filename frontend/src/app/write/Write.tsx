/**
 * Writing a review: the heading and the composer grid.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import { IdentitySetup } from '@/app/identity/IdentitySetup'
import { Composer } from '@/app/write/Composer'
import { GradientASCII } from '@/app/write/GradientASCII'
import { useWriteData } from '@/app/write/useWriteData'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'

// --- GLOBALS ---
// smooth ease-in from canvas dark at the top into the full backdrop
const FADE =
  'linear-gradient(to bottom, rgb(0 0 0 / 0) 0%, rgb(0 0 0 / 0.03) 8%, ' +
  'rgb(0 0 0 / 0.12) 18%, rgb(0 0 0 / 0.28) 28%, rgb(0 0 0 / 0.5) 38%, ' +
  'rgb(0 0 0 / 0.74) 48%, rgb(0 0 0 / 0.92) 56%, rgb(0 0 0) 64%)'

// --- CODE ---
/**
 * Builds the Write screen.
 *
 * @returns The screen, or nothing until the account is known.
 */
export function Write() {
  const { t } = useTranslation()
  const data = useWriteData()
  const rise = useRise()

  // RequireAuth only renders this with a complete account
  if (!data.account?.handle) return null

  return (
    <section className="relative isolate flex min-h-svh items-center">
      <div
        className="pointer-events-none fixed inset-0 -z-10"
        style={{ maskImage: FADE }}
      >
        <GradientASCII
          colors={['#1e3fd8', '#cfe0ff', '#8bc6ff', '#38bdf8']}
          shape="horizon"
          characters=" =+xX0S8#@"
          fontFamily="'IBM Plex Mono', ui-monospace, monospace"
          mouseInteraction
          clickEffect="wave"
          intro
        />
      </div>

      <Container className="flex justify-center py-32">
        <div className="flex w-full max-w-3xl flex-col gap-10">
          <motion.h1
            {...rise()}
            className={cn(
              'font-display text-display-m leading-[1.08] font-semibold',
              'tracking-display',
            )}
          >
            {t('Write a review.')}
          </motion.h1>

          {/* signing needs an identity: made here, with its backup, first */}
          {data.hasIdentity === false ? (
            <motion.div {...rise(0.08)}>
              <IdentitySetup onDone={data.identityReady} />
            </motion.div>
          ) : data.hasIdentity ? (
            <motion.div
              {...rise(0.08)}
              className="border border-line bg-canvas"
            >
              <Composer data={data} />
            </motion.div>
          ) : null}
        </div>
      </Container>
    </section>
  )
}
