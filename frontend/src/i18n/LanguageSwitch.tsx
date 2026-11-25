/**
 * The control that switches the interface between English and pt-BR.
 */

// --- IMPORTS ---
import { useTranslation } from 'react-i18next'

import { cn } from '@/components/ui/cn'
import { currentLanguage, setLanguage } from '@/i18n/i18n'
import type {
  LanguageOption,
  LanguageSwitchProps,
} from '@/i18n/LanguageSwitch.t'

// --- GLOBALS ---
const OPTIONS: LanguageOption[] = [
  { value: 'en', short: 'EN', name: 'English' },
  { value: 'pt-BR', short: 'PT', name: 'Português (Brasil)' },
]

// --- CODE ---
/**
 * Renders one button per language, the current one marked.
 *
 * @param props - Extra classes.
 * @returns The switch.
 */
export function LanguageSwitch({ className }: LanguageSwitchProps) {
  const { t } = useTranslation()
  const current = currentLanguage()

  return (
    <div
      role="group"
      aria-label={t('Language')}
      className={cn(
        // a framed control of its own, so it never reads as a nav link
        'inline-flex items-center gap-0.5 border border-line p-0.5',
        'font-mono text-mono-s',
        className,
      )}
    >
      {/* a globe: the sign for language, before any word is read */}
      <svg
        aria-hidden="true"
        viewBox="0 0 16 16"
        className="mx-1 size-3.5 shrink-0 text-muted"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.25"
      >
        <circle cx="8" cy="8" r="6.25" />
        <ellipse cx="8" cy="8" rx="2.75" ry="6.25" />
        <path d="M1.75 8h12.5" />
      </svg>

      {OPTIONS.map((option) => {
        const active = option.value === current

        return (
          <button
            key={option.value}
            type="button"
            lang={option.value}
            title={option.name}
            aria-label={option.name}
            aria-pressed={active}
            onClick={() => setLanguage(option.value)}
            className={cn(
              'min-h-7 min-w-8 cursor-pointer px-1.5 font-semibold',
              'tracking-metadata transition-colors duration-interface',
              'ease-standard',
              active
                ? 'bg-surface-raised text-primary'
                : 'text-muted hover:text-primary',
            )}
          >
            {option.short}
          </button>
        )
      })}
    </div>
  )
}
