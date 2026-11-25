/**
 * The check behind the human claim: a proof of work the browser solves,
 * with no third party watching. ALTCHA, run against this gateway.
 */

// --- IMPORTS ---
import 'altcha'
import type {} from 'altcha/types/react'
import { useEffect, useRef } from 'react'
import type { HumanCheckProps } from '@/app/claims/HumanCheck.t'
import { HUMAN_CHALLENGE_URL } from '@/logic/datasource/claims'

// --- CODE ---
/**
 * Renders the check, and hands on its payload once it is solved.
 *
 * @param props - What to do with the solved check.
 * @returns The widget.
 */
export function HumanCheck({ onSolved }: HumanCheckProps) {
  const widget = useRef<HTMLElement>(null)

  useEffect(() => {
    const element = widget.current
    if (!element) return

    const changed = (event: Event) => {
      const { state, payload } = (event as CustomEvent).detail ?? {}
      if (state === 'verified' && typeof payload === 'string') {
        onSolved(payload)
      }
    }

    element.addEventListener('statechange', changed)
    return () => element.removeEventListener('statechange', changed)
  }, [onSolved])

  return (
    <altcha-widget
      ref={widget}
      challenge={HUMAN_CHALLENGE_URL}
      type="checkbox"
      display="standard"
      name="altcha"
    />
  )
}
