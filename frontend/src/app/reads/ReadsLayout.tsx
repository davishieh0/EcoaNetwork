/**
 * What the read screens share: one backdrop, kept as the reader moves from
 * search to an entity to a review.
 */

// --- IMPORTS ---
import { Outlet } from 'react-router'
import { Backdrop } from '@/app/reads/Backdrop'

// --- CODE ---
/**
 * Lays the backdrop behind whichever read screen is open.
 *
 * @returns The backdrop and the screen.
 */
export function ReadsLayout() {
  return (
    <div className="relative isolate">
      <Backdrop />
      <Outlet />
    </div>
  )
}
