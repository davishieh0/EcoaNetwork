/**
 * The reader's moderation choice: whose labels they follow. Moderation is a
 * projection, so following other moderators changes what this reader sees
 * and nothing for anyone else.
 *
 * Kept in this browser only, a reader's preference like a remembered
 * filter; the index reads it from each request.
 */

// --- IMPORTS ---
import { useSyncExternalStore } from 'react'
import type { ModerationChoice } from '@/logic/moderation/moderation.t'

// --- GLOBALS ---
const KEY = 'ecoa.moderation'

const DEFAULT: ModerationChoice = {
  follow: 'index',
  moderators: [],
  moderating: false,
}

// the most moderators a request may name
export const MAX_MODERATORS = 20

const listeners = new Set<() => void>()

let current: ModerationChoice = read()

// --- CODE ---
/**
 * Reads the choice kept in this browser; the default when there is none or
 * the storage cannot be read.
 *
 * @returns The choice.
 */
function read(): ModerationChoice {
  try {
    const raw = globalThis.localStorage?.getItem(KEY)
    if (!raw) return DEFAULT

    const parsed = JSON.parse(raw) as Partial<ModerationChoice>
    return {
      follow: parsed.follow === 'own' ? 'own' : 'index',
      moderators: Array.isArray(parsed.moderators)
        ? parsed.moderators.filter((m) => typeof m === 'string')
        : [],
      moderating: parsed.moderating === true,
    }
  } catch {
    return DEFAULT
  }
}

/**
 * Changes the choice, keeps it, and tells every subscriber.
 *
 * @param change - The fields to change.
 * @returns Nothing.
 */
export function setModeration(change: Partial<ModerationChoice>) {
  current = { ...current, ...change }

  try {
    globalThis.localStorage?.setItem(KEY, JSON.stringify(current))
  } catch {
    // private windows and blocked storage: it holds for this visit
  }

  listeners.forEach((listener) => listener())
}

/**
 * Subscribes to the choice.
 *
 * @param listener - Called on each change.
 * @returns The unsubscriber.
 */
function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/**
 * What a read sends as `?moderators=`: undefined lets the index follow its
 * own list; a string, even empty, is the reader's own.
 *
 * @param choice - The reader's choice.
 * @returns The parameter.
 */
export function moderatorsParam(choice: ModerationChoice): string | undefined {
  return choice.follow === 'own' ? choice.moderators.join(',') : undefined
}

/**
 * Reads the choice, and renders again when it changes.
 *
 * @returns The choice and the parameter reads send.
 */
export function useModeration() {
  const choice = useSyncExternalStore(subscribe, () => current, () => current)

  return { choice, param: moderatorsParam(choice) }
}
