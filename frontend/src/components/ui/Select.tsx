/**
 * A native select drawn as a bordered mono control with a chevron.
 */

// --- IMPORTS ---
import type { SelectProps } from '@/components/ui/Select.t'
import { cn } from '@/components/ui/cn'

// --- GLOBALS ---
const CONTROL = cn(
  'peer w-full min-h-9 cursor-pointer appearance-none border border-line',
  'bg-transparent pr-8 pl-3 font-mono text-mono-s text-secondary',
  'transition-colors duration-interface ease-standard',
  'hover:border-strong hover:text-primary focus:border-signal',
  'focus:text-primary focus-visible:shadow-none focus-visible:outline-none',
)

// --- CODE ---
/**
 * Renders the select.
 *
 * @param props - Its name, value, choices and change handler.
 * @returns The select.
 */
export function Select<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
}: SelectProps<T>) {
  return (
    <label className="relative flex items-center">
      <span className="sr-only">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value as T)}
        className={cn(CONTROL, className)}
      >
        {options.map((option) => (
          <option
            key={option.value}
            value={option.value}
            className="bg-surface-raised text-primary"
          >
            {option.label}
          </option>
        ))}
      </select>
      <svg
        aria-hidden="true"
        viewBox="0 0 10 6"
        className={cn(
          'pointer-events-none absolute right-3 h-1.5 w-2.5 text-muted',
          'transition-colors peer-hover:text-primary peer-focus:text-signal',
        )}
      >
        <path
          d="M1 1l4 4 4-4"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
        />
      </svg>
    </label>
  )
}
