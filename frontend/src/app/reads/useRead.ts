/**
 * One read, done again when what it reads changes.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useState } from 'react'
import type { ReadState } from '@/app/reads/useRead.t'

// --- CODE ---
/**
 * Runs a read and keeps its answer or its failure. A new reader starts it
 * over; what an old one answers late is dropped.
 *
 * @param read - The read. Keep it stable (`useCallback`): a new one reads
 * again.
 * @returns The answer, the failure, whether it is still on its way, and
 * reload, which reads again keeping the answer on screen until the next.
 */
export function useRead<T>(read: () => Promise<T>) {
  const [state, setState] = useState<ReadState<T>>({
    source: null,
    data: null,
    error: null,
  })
  const [round, setRound] = useState(0)

  useEffect(() => {
    let live = true

    read().then(
      (data) => live && setState({ source: read, data, error: null }),
      (error: unknown) =>
        live && setState({ source: read, data: null, error }),
    )

    return () => {
      live = false
    }
    // a new round reads again with the same reader
  }, [read, round])

  const current = state.source === read
  const reload = useCallback(() => setRound((now) => now + 1), [])

  return {
    loading: !current,
    data: current ? state.data : null,
    error: current ? state.error : null,
    reload,
  }
}
