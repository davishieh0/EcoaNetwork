/**
 * Two versions of a review, both still there.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

// --- GLOBALS ---
/** Two versions, both still here. */
const EASE = [0.2, 0.8, 0.2, 1] as const

// --- CODE ---
/**
 * Draws two versions of a review, both kept.
 *
 * @returns The stack, as inline SVG.
 */
export function VersionStack() {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  /**
   * Builds motion props for one version card.
   *
   * @param delay - Seconds before it arrives.
   * @returns Props to spread onto a motion element.
   */
  const card = (delay: number) =>
    reduceMotion
      ? { initial: false as const }
      : {
          initial: { opacity: 0, y: 10 },
          whileInView: { opacity: 1, y: 0 },
          viewport: { once: true, amount: 0.6 },
          transition: { duration: 0.4, delay, ease: EASE },
        }

  return (
    <svg
      viewBox="0 0 240 140"
      aria-hidden="true"
      focusable="false"
      className="h-auto w-full max-w-xs overflow-visible"
    >
      {/* Version 1 — still readable, not greyed out into a ghost. */}
      <motion.g {...card(0)}>
        <rect
          x={8}
          y={56}
          width={104}
          height={76}
          fill="var(--ecoa-surface)"
          stroke="var(--ecoa-border-structural)"
          strokeWidth={1.5}
          vectorEffect="non-scaling-stroke"
        />
        <rect
          x={20}
          y={72}
          width={62}
          height={5}
          fill="var(--ecoa-border-strong)"
        />
        <rect
          x={20}
          y={84}
          width={78}
          height={5}
          fill="var(--ecoa-border-strong)"
        />
        <text
          x={20}
          y={116}
          className="font-mono"
          fontSize={9}
          letterSpacing={1}
          fill="var(--ecoa-text-muted)"
        >
          v1
        </text>
      </motion.g>

      <motion.path
        d="M112 94 H128"
        stroke="var(--ecoa-border-strong)"
        strokeWidth={1.5}
        vectorEffect="non-scaling-stroke"
        initial={reduceMotion ? false : { pathLength: 0 }}
        whileInView={{ pathLength: 1 }}
        viewport={{ once: true, amount: 0.6 }}
        transition={
          reduceMotion
            ? { duration: 0 }
            : { duration: 0.3, delay: 0.3, ease: EASE }
        }
      />

      {/* Version 2 — beside it, not on top of it. */}
      <motion.g {...card(0.42)}>
        <rect
          x={128}
          y={8}
          width={104}
          height={76}
          fill="var(--ecoa-surface)"
          // Ink, and heavier than v1's 1.5px.
          stroke="var(--ecoa-border-structural)"
          strokeWidth={2.5}
          vectorEffect="non-scaling-stroke"
        />
        <rect
          x={140}
          y={24}
          width={62}
          height={5}
          fill="var(--ecoa-border-strong)"
        />
        <rect x={140} y={36} width={78} height={5} fill="var(--ecoa-signal)" />
        <text
          x={140}
          y={68}
          className="font-mono"
          fontSize={9}
          letterSpacing={1}
          fill="var(--ecoa-text-primary)"
        >
          v2 · {t('current')}
        </text>
      </motion.g>
    </svg>
  )
}
