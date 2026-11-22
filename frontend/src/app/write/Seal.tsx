/**
 * The seal from the landing page, drawn open: the text is there, the
 * signature is not.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'

import type { SealProps } from '@/app/write/Write.t'

// --- GLOBALS ---
const EASE = [0.2, 0.8, 0.2, 1] as const

// the review text, as three bars
const LINES = [
  { y: 22, width: 92 },
  { y: 34, width: 108 },
  { y: 46, width: 76 },
]

// --- CODE ---
/**
 * Draws the text bars, the place the signature goes, and the open diamond.
 *
 * @param props - Whether the draft is ready to sign, and whether it is
 * sealed.
 * @returns The drawing.
 */
export function Seal({ ready, sealed = false }: SealProps) {
  const reduceMotion = useReducedMotion()
  const transition = (delay: number) =>
    reduceMotion ? { duration: 0 } : { duration: 0.4, delay, ease: EASE }

  return (
    <svg
      viewBox="0 0 220 96"
      aria-hidden="true"
      focusable="false"
      className="h-auto w-full overflow-visible"
    >
      {LINES.map((line, index) => (
        <motion.rect
          key={line.y}
          x={8}
          y={line.y}
          height={5}
          initial={false}
          animate={{
            width: ready ? line.width : line.width * 0.4,
            fill: ready
              ? 'var(--ecoa-text-primary)'
              : 'var(--ecoa-border-strong)',
          }}
          transition={transition(index * 0.08)}
        />
      ))}

      <path
        d="M8 74c14-10 22 6 34-2s18 6 30-3 20 5 32-4 18 4 30-3"
        fill="none"
        stroke="var(--ecoa-border-strong)"
        strokeWidth={2}
        strokeDasharray="4 6"
        strokeLinecap="round"
        vectorEffect="non-scaling-stroke"
      />

      <motion.path
        d="M186 26 L206 48 L186 70 L166 48 Z"
        fill="var(--ecoa-signal)"
        stroke="var(--ecoa-signal)"
        strokeWidth={2}
        strokeDasharray={sealed ? undefined : '5 5'}
        vectorEffect="non-scaling-stroke"
        initial={false}
        animate={{
          opacity: ready ? 1 : 0.45,
          rotate: sealed ? 180 : ready ? 90 : 0,
          scale: sealed ? [1, 1.35, 1] : 1,
          fillOpacity: sealed ? 1 : 0,
        }}
        transition={transition(sealed ? 0 : 0.24)}
        style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
      />
    </svg>
  )
}
