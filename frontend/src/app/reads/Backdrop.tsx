/**
 * The read screens' backdrop: the network drifting in a band across the
 * top, fading out before the reading starts.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { NetworkField } from '@/components/ecoa/NetworkField'

// --- GLOBALS ---
// full at the top, gone by the band's foot
const FADE =
  'linear-gradient(to bottom, rgb(0 0 0) 0%, rgb(0 0 0 / 0.85) 25%, ' +
  'rgb(0 0 0 / 0.5) 50%, rgb(0 0 0 / 0.18) 75%, rgb(0 0 0 / 0) 100%)'

// --- CODE ---
/**
 * Draws the band behind the page's header.
 *
 * @returns The backdrop, out of the way of the pointer and of readers.
 */
export function Backdrop() {
  const reduceMotion = useReducedMotion()

  return (
    <motion.div
      aria-hidden="true"
      className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[70svh]"
      style={{ maskImage: FADE }}
      initial={reduceMotion ? false : { opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: reduceMotion ? 0 : 1.2 }}
    >
      <NetworkField />
    </motion.div>
  )
}
