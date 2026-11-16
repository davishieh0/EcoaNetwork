/**
 * A small "i" that shows a note on hover, focus or tap.
 */

// --- IMPORTS ---
import { useId, useState } from 'react'

import type { TooltipProps } from '@/components/ui/Tooltip.t'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
type TooltipState = 'closed' | 'hover' | 'focus' | 'pinned'

// --- CODE ---
/**
 * Renders the trigger and its note.
 *
 * @param props - The trigger's name and the note.
 * @returns The tooltip.
 */
export function Tooltip({ label, children }: TooltipProps) {
  const id = useId()
  const [state, setState] = useState<TooltipState>('closed')
  const open = state !== 'closed'

  return (
    <span
      className="relative inline-flex"
      onMouseEnter={() =>
        setState((current) => (current === 'closed' ? 'hover' : current))
      }
      onMouseLeave={() =>
        setState((current) => (current === 'hover' ? 'closed' : current))
      }
    >
      <button
        type="button"
        aria-label={label}
        aria-describedby={open ? id : undefined}
        aria-expanded={open}
        onClick={() =>
          setState((current) => (current === 'pinned' ? 'closed' : 'pinned'))
        }
        onFocus={(event) => {
          if (event.currentTarget.matches(':focus-visible')) {
            setState((current) => (current === 'pinned' ? 'pinned' : 'focus'))
          }
        }}
        onBlur={() => setState('closed')}
        onKeyDown={(event) => {
          if (event.key === 'Escape') setState('closed')
        }}
        className={cn(
          'flex size-4 cursor-default items-center justify-center rounded-full',
          'border border-line font-mono text-[0.625rem] leading-none',
          'text-muted transition-colors duration-interface ease-standard',
          'hover:border-strong hover:text-primary aria-expanded:border-signal',
          'aria-expanded:text-signal focus-visible:shadow-none',
          'focus-visible:outline-none',
        )}
      >
        i
      </button>
      <span
        id={id}
        role="tooltip"
        hidden={!open}
        className={cn(
          'absolute top-full left-1/2 z-10 mt-2 w-64 -translate-x-1/2 border',
          'border-line bg-surface-raised px-3 py-2 font-sans text-body-s',
          'text-secondary normal-case tracking-normal text-pretty',
        )}
      >
        {children}
      </span>
    </span>
  )
}
