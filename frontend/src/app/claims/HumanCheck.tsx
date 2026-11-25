/**
 * The check behind the human claim: a proof of work the browser solves,
 * with no third party watching. ALTCHA, run against this gateway.
 */

// --- IMPORTS ---
import 'altcha'
import 'altcha/i18n/pt-br'
import type {} from 'altcha/types/react'
import { Fragment, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import type { HumanCheckProps } from '@/app/claims/HumanCheck.t'
import { currentLanguage } from '@/i18n/i18n'
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
  // the widget speaks pt-BR when the interface does, English otherwise
  useTranslation()
  const language = currentLanguage() === 'pt-BR' ? 'pt-br' : undefined

  // a new language makes a new widget: listen to that one
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
  }, [onSolved, language])

  return (
    <Fragment key={language ?? 'en'}>
      <altcha-widget
        ref={widget}
        language={language}
        challenge={HUMAN_CHALLENGE_URL}
        type="checkbox"
        display="standard"
        name="altcha"
      />
    </Fragment>
  )
}
