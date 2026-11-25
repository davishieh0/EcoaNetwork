/**
 * The toggle for what moderation hides.
 */

// --- IMPORTS ---
import { useState } from 'react'
import type { ShowLabeledProps } from '@/app/reads/ShowLabeled.t'

// --- CODE ---
/**
 * Shows the toggle as switched at once, while the address catches up: the
 * address is what the list reads, and it changes a moment later.
 *
 * @param props - Whether the address shows it, and what changes it.
 * @returns The toggle.
 */
export function ShowLabeled({ checked, onChange }: ShowLabeledProps) {
  const [wanted, setWanted] = useState(checked)
  const [seen, setSeen] = useState(checked)

  // the address moved on its own, by going back for instance: follow it
  if (checked !== seen) {
    setSeen(checked)
    setWanted(checked)
  }

  return (
    <label className="meta flex cursor-pointer items-center gap-2">
      <input
        type="checkbox"
        checked={wanted}
        onChange={(event) => {
          setWanted(event.target.checked)
          onChange(event.target.checked)
        }}
      />
      Show what moderation hides
    </label>
  )
}
