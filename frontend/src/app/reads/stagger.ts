/**
 * The delay of each item of a list as it arrives, so a page cascades in.
 */

// --- GLOBALS ---
// a page holds this many; the page after starts its cascade over
const PAGE = 20

// past this many, the rest arrive together: nobody waits on a long cascade
const MOST = 8

const STEP = 0.045

// --- CODE ---
/**
 * The delay of one item, counted within its page.
 *
 * @param index - The item's place in the whole list.
 * @param start - Seconds before the first one.
 * @returns Seconds before it arrives.
 */
export function stagger(index: number, start = 0): number {
  return start + Math.min(index % PAGE, MOST) * STEP
}
