/**
 * Runs one backend action for a form, and keeps its state.
 */

// --- IMPORTS ---
import { useCallback, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { messageOf } from '@/app/forms/messages'
import type { Submit } from '@/app/forms/useSubmit.t'
import { HttpError } from '@/logic/errors/httpError'

// --- CODE ---
/**
 * Wraps an action with a pending flag and its last failure.
 *
 * @param action - The call to the backend, and what follows its success.
 * @returns The runner and its state.
 */
export function useSubmit<A extends unknown[]>(
  action: (...args: A) => Promise<void>,
): Submit<A> {
  // the message is worded at render, in the current language
  useTranslation()
  const [pending, setPending] = useState(false)
  const [failure, setFailure] = useState<unknown>(null)

  const run = useCallback(
    async (...args: A) => {
      setPending(true)
      setFailure(null)

      try {
        await action(...args)
        return true
      } catch (error) {
        setFailure(error)
        return false
      } finally {
        setPending(false)
      }
    },
    [action],
  )

  const reset = useCallback(() => setFailure(null), [])

  return {
    run,
    pending,
    code: failure instanceof HttpError ? failure.code : '',
    message: failure ? messageOf(failure) : '',
    reset,
  }
}
