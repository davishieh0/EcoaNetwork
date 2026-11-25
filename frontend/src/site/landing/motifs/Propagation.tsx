/**
 * One signed event reaching the nodes that observed it.
 */

// --- IMPORTS ---
import { motion, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

import type { PropagationProps } from '@/site/landing/motifs/Propagation.t'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
/** The hero motif. */
const EDGES = [
  { d: 'M76 210 L186 118', tier: 0 },
  { d: 'M76 210 L186 210', tier: 0 },
  { d: 'M76 210 L186 302', tier: 0 },
  { d: 'M186 118 L320 64', tier: 1 },
  { d: 'M186 118 L320 158', tier: 1 },
  { d: 'M186 210 L320 158', tier: 1 },
  { d: 'M186 302 L320 256', tier: 1 },
  { d: 'M186 302 L320 352', tier: 1 },
] as const

const NODES = [
  { cx: 186, cy: 118, r: 9, tier: 1 },
  { cx: 186, cy: 210, r: 9, tier: 1 },
  { cx: 186, cy: 302, r: 9, tier: 1 },
  { cx: 320, cy: 64, r: 7, tier: 2 },
  { cx: 320, cy: 158, r: 7, tier: 2 },
  { cx: 320, cy: 256, r: 7, tier: 2 },
  { cx: 320, cy: 352, r: 7, tier: 2 },
] as const

// --- CODE ---
/**
 * Draws one event reaching the nodes that observed it.
 *
 * @param props - Extra classes, and a key that replays the animation.
 * @returns The graph, as inline SVG.
 */
export function Propagation({ className, runKey = 0 }: PropagationProps) {
  const { t } = useTranslation()
  const reduceMotion = useReducedMotion()

  /**
   * Builds the transition for an edge at a tier.
   *
   * @param tier - Distance from the origin, in hops.
   * @returns The transition for that edge.
   */
  const edgeTransition = (tier: number) =>
    reduceMotion
      ? { duration: 0 }
      : {
          duration: 0.72,
          delay: tier * 0.26,
          ease: [0.2, 0.8, 0.2, 1] as const,
        }

  /**
   * Builds the transition for a node at a tier.
   *
   * @param tier - Distance from the origin, in hops.
   * @returns The transition for that node.
   */
  const nodeTransition = (tier: number) =>
    reduceMotion
      ? { duration: 0 }
      : {
          duration: 0.18,
          delay: tier * 0.26,
          ease: [0.2, 0.8, 0.2, 1] as const,
        }

  return (
    <svg
      key={runKey}
      viewBox="0 0 420 420"
      role="img"
      aria-labelledby="propagation-title"
      className={cn('h-auto w-full overflow-visible', className)}
    >
      <title id="propagation-title">
        {t(
          'A signed event published at one node and observed by seven ' +
            'others',
        )}
      </title>

      <g>
        {EDGES.map((edge) => (
          <motion.path
            key={edge.d}
            d={edge.d}
            fill="none"
            stroke="var(--ecoa-border-strong)"
            strokeWidth={1.5}
            vectorEffect="non-scaling-stroke"
            initial={{ pathLength: 0 }}
            animate={{ pathLength: 1 }}
            transition={edgeTransition(edge.tier)}
          />
        ))}
      </g>

      {/* The artifact: the event itself. */}
      <motion.rect
        x={62}
        y={196}
        width={28}
        height={28}
        fill="var(--ecoa-signal)"
        style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
        initial={{ scale: 1 }}
        animate={reduceMotion ? { scale: 1 } : { scale: [1, 1.18, 1] }}
        transition={{ duration: 0.32, ease: [0.2, 0.8, 0.2, 1] }}
      />

      {/* The actors: nodes that took a copy. */}
      <g>
        {NODES.map((node) => (
          <motion.circle
            key={`${node.cx}-${node.cy}`}
            cx={node.cx}
            cy={node.cy}
            r={node.r}
            fill="var(--ecoa-surface)"
            stroke="var(--ecoa-signal)"
            strokeWidth={2}
            vectorEffect="non-scaling-stroke"
            style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
            initial={{ opacity: 0, scale: 0.4 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={nodeTransition(node.tier)}
          />
        ))}
      </g>
    </svg>
  )
}
