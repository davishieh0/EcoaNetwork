/**
 * The four-step flow, built as the reader scrolls through it.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { FlowDiagramProps } from '@/site/landing/motifs/FlowDiagram.t'

// --- GLOBALS ---
/** The flow, built one step at a time as the reader moves through it. */
const EASE = [0.2, 0.8, 0.2, 1] as const

const OBSERVERS = [
  { cx: 268, cy: 84 },
  { cx: 288, cy: 160 },
  { cx: 268, cy: 236 },
] as const

// --- CODE ---
/**
 * Draws the flow up to the given step.
 *
 * @param props - The step reached.
 * @returns The diagram at that step.
 */
export function FlowDiagram({ step }: FlowDiagramProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  /**
   * Builds motion props visible from a given step.
   *
   * @param from - Step at which it appears.
   * @param extra - Extra animated values.
   * @returns Props to spread onto a motion element.
   */
  const at = (from: number, extra: Record<string, number> = {}) => ({
    animate: {
      opacity: step >= from ? 1 : 0,
      scale: step >= from ? 1 : 0.6,
      ...extra,
    },
    transition: reduceMotion ? { duration: 0 } : { duration: 0.42, ease: EASE },
    style: {
      transformBox: 'fill-box' as const,
      transformOrigin: 'center' as const,
    },
  })

  /**
   * Builds motion props that draw from a given step.
   *
   * @param from - Step at which it draws.
   * @returns Props to spread onto a motion path.
   */
  const drawAt = (from: number) => ({
    animate: {
      pathLength: step >= from ? 1 : 0,
      opacity: step >= from ? 1 : 0,
    },
    transition: reduceMotion ? { duration: 0 } : { duration: 0.5, ease: EASE },
  })

  return (
    <svg
      viewBox="0 0 320 320"
      role="img"
      aria-label={t('Step {{n}} of 4 of the publishing flow', {
        n: step + 1,
      })}
      className="h-auto w-full overflow-visible"
    >
      {/* 01 — the person. */}
      <motion.circle
        cx={48}
        cy={160}
        r={15}
        fill="none"
        stroke="var(--ecoa-text-primary)"
        strokeWidth={2}
        vectorEffect="non-scaling-stroke"
        {...at(0)}
      />

      {/* 02 — they write something, and sign it. */}
      <motion.line
        x1={63}
        y1={160}
        x2={130}
        y2={160}
        stroke="var(--ecoa-border-strong)"
        strokeWidth={1.5}
        vectorEffect="non-scaling-stroke"
        {...drawAt(1)}
      />

      <motion.rect
        x={130}
        y={138}
        width={44}
        height={44}
        fill="none"
        stroke="var(--ecoa-text-primary)"
        strokeWidth={2}
        vectorEffect="non-scaling-stroke"
        {...at(1)}
      />

      {/* A hand mark means a person authored this. */}
      <motion.path
        d="M136 196c6-7 12 5 18-2s10 4 16-3 12 2 18-4"
        fill="none"
        stroke="var(--ecoa-hand)"
        strokeWidth={2.5}
        strokeLinecap="round"
        vectorEffect="non-scaling-stroke"
        {...drawAt(1)}
      />

      {/* 03 — signed, it becomes an event: a filled artifact with an id. */}
      <motion.rect
        x={130}
        y={138}
        width={44}
        height={44}
        fill="var(--ecoa-signal)"
        stroke="none"
        {...at(2)}
      />

      <motion.text
        x={152}
        y={122}
        textAnchor="middle"
        className="font-mono"
        fontSize={11}
        letterSpacing={1}
        fill="var(--ecoa-text-muted)"
        {...at(2)}
      >
        8f2c…a41
      </motion.text>

      {/* 04 — independent nodes take a copy. */}
      {OBSERVERS.map((observer, index) => (
        <motion.line
          key={`edge-${index}`}
          x1={174}
          y1={160}
          x2={observer.cx}
          y2={observer.cy}
          stroke="var(--ecoa-border-strong)"
          strokeWidth={1.5}
          strokeDasharray="3 5"
          vectorEffect="non-scaling-stroke"
          animate={{
            pathLength: step >= 3 ? 1 : 0,
            opacity: step >= 3 ? 1 : 0,
          }}
          transition={
            reduceMotion
              ? { duration: 0 }
              : { duration: 0.45, delay: index * 0.08, ease: EASE }
          }
        />
      ))}

      {OBSERVERS.map((observer, index) => (
        <motion.circle
          key={`node-${index}`}
          cx={observer.cx}
          cy={observer.cy}
          r={10}
          fill="var(--ecoa-canvas)"
          stroke="var(--ecoa-signal)"
          strokeWidth={2}
          vectorEffect="non-scaling-stroke"
          animate={{
            opacity: step >= 3 ? 1 : 0,
            scale: step >= 3 ? 1 : 0.4,
          }}
          transition={
            reduceMotion
              ? { duration: 0 }
              : { duration: 0.3, delay: 0.2 + index * 0.08, ease: EASE }
          }
          style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
        />
      ))}
    </svg>
  )
}
