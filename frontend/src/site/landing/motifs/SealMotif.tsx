/**
 * The seal, and what breaks it.
 */

// --- IMPORTS ---
import { cn } from '@/components/ui/cn'
import { useRef, useState } from 'react'
import { motion, useInView, useReducedMotion } from 'motion/react'
import { useTranslation } from 'react-i18next'

// --- GLOBALS ---
/** The seal, and what breaks it. */
const EASE = [0.2, 0.8, 0.2, 1] as const

const LINES = [
  { y: 22, width: 92 },
  { y: 34, width: 108 },
  { y: 46, width: 76 },
]

// --- CODE ---
/**
 * Draws the seal and the control that breaks it.
 *
 * @returns The figure, with its tamper button.
 */
export function SealMotif() {
  const reduceMotion = useReducedMotion()
  const [tampered, setTampered] = useState(false)
  const { t } = useTranslation()
  const svgRef = useRef<SVGSVGElement>(null)
  const inView = useInView(svgRef, { once: true, amount: 0.6 })

  return (
    <figure
      className={cn(
        'flex flex-col gap-4 border border-structural bg-surface p-8',
      )}
    >
      <figcaption className="meta text-muted">{t('The seal')}</figcaption>

      <svg
        ref={svgRef}
        viewBox="0 0 220 96"
        aria-hidden="true"
        focusable="false"
        className="h-auto w-full overflow-visible"
      >
        {/* The review text. */}
        {LINES.map((line, index) => (
          <motion.rect
            key={line.y}
            x={8}
            y={line.y}
            height={5}
            fill="var(--ecoa-border-strong)"
            initial={{ width: 0 }}
            animate={
              inView
                ? {
                    width:
                      tampered && index === 1 ? line.width + 26 : line.width,
                  }
                : { width: 0 }
            }
            transition={
              reduceMotion
                ? { duration: 0 }
                : {
                    duration: 0.4,
                    delay: tampered ? 0 : index * 0.08,
                    ease: EASE,
                  }
            }
          />
        ))}

        {/* The signature over it. */}
        <motion.path
          d="M8 74c14-10 22 6 34-2s18 6 30-3 20 5 32-4 18 4 30-3"
          fill="none"
          stroke="var(--ecoa-hand)"
          // The token, not a literal.
          strokeWidth="var(--hand-weight)"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
          initial={reduceMotion ? false : { pathLength: 0 }}
          whileInView={{ pathLength: 1 }}
          viewport={{ once: true, amount: 0.6 }}
          transition={
            reduceMotion
              ? { duration: 0 }
              : { duration: 0.8, delay: 0.3, ease: EASE }
          }
          animate={tampered ? { opacity: 0.25 } : { opacity: 1 }}
        />

        {/* The diamond marks verification. */}
        <motion.path
          d="M186 26 L206 48 L186 70 L166 48 Z"
          fill={tampered ? 'none' : 'var(--ecoa-signal)'}
          stroke={tampered ? 'var(--ecoa-danger)' : 'none'}
          strokeWidth={2}
          strokeDasharray={tampered ? '5 5' : undefined}
          vectorEffect="non-scaling-stroke"
          initial={reduceMotion ? false : { opacity: 0, scale: 0.5 }}
          whileInView={{ opacity: 1, scale: 1 }}
          viewport={{ once: true, amount: 0.6 }}
          transition={
            reduceMotion
              ? { duration: 0 }
              : { duration: 0.3, delay: 0.9, ease: EASE }
          }
          style={{ transformBox: 'fill-box', transformOrigin: 'center' }}
        />
      </svg>

      <div
        className={cn(
          'flex flex-wrap items-center justify-between gap-3 border-t',
          'border-line pt-4',
        )}
      >
        {/* aria-live: the change is in the drawing, which is hidden. */}
        <p aria-live="polite" className="meta">
          {tampered ? (
            <span className="text-danger">
              {t('◆ Signature no longer valid')}
            </span>
          ) : (
            <span className="text-valid">{t('◆ Signature valid')}</span>
          )}
        </p>

        <button
          type="button"
          onClick={() => setTampered((value) => !value)}
          className={cn(
            'meta cursor-pointer border-b-2 border-signal text-primary',
            'transition-colors duration-interface hover:text-secondary',
          )}
        >
          {tampered ? t('Restore the text') : t('Try changing the text')}
        </button>
      </div>
    </figure>
  )
}
