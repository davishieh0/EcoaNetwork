/**
 * One read, done again when what it reads changes.
 */

// --- IMPORTS ---
import { useEffect, useState } from 'react'
import type { ReadState } from '@/app/reads/useRead.t'

// --- CODE ---
/**
 * Runs a read and keeps its answer or its failure. A new reader starts it
 * over; what an old one answers late is dropped.
 *
 * @param read - The read. Keep it stable (`useCallback`): a new one reads
 * again.
 * @returns The answer, the failure, and whether it is still on its way.
 */
export function useRead<T>(read: () => Promise<T>) {
  const [state, setState] = useState<ReadState<T>>({
    source: null,
    data: null,
    error: null,
  })

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
  }, [read])

  const current = state.source === read

  return {
    loading: !current,
    data: current ? state.data : null,
    error: current ? state.error : null,
  }
}
