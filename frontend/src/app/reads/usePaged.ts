/**
 * A list read a page at a time, by cursor.
 */

// --- IMPORTS ---
import { useCallback, useEffect, useState } from 'react'
import type { Loader, PagedState } from '@/app/reads/usePaged.t'

// --- CODE ---
/**
 * Reads the first page, and the page after on demand. A new loader starts
 * the list over; what an old one answers late is dropped.
 *
 * @param load - Reads one page. Keep it stable (`useCallback`): a new one
 * means a new list.
 * @returns The items so far, where the list is, and loadMore.
 */
export function usePaged<Item>(load: Loader<Item>) {
  const [state, setState] = useState<PagedState<Item>>({
    source: null,
    items: [],
    next: null,
    error: null,
    more: 'idle',
  })

  useEffect(() => {
    let live = true

    load().then(
      (page) => {
        if (!live) return
        setState({
          source: load,
          items: page.items,
          next: page.next,
          ranking: page.ranking,
          error: null,
          more: 'idle',
        })
      },
      (error: unknown) => {
        if (!live) return
        setState({ source: load, items: [], next: null, error, more: 'idle' })
      },
    )

    return () => {
      live = false
    }
  }, [load])

  const current = state.source === load
  const { next } = state

  const loadMore = useCallback(() => {
    if (!next) return
    setState((now) => ({ ...now, more: 'loading' }))

    load(next).then(
      (page) =>
        setState((now) =>
          now.source === load && now.next === next
            ? {
                ...now,
                items: [...now.items, ...page.items],
                next: page.next,
                more: 'idle',
              }
            : now,
        ),
      () =>
        setState((now) =>
          now.source === load ? { ...now, more: 'failed' } : now,
        ),
    )
  }, [load, next])

  return {
    loading: !current,
    items: current ? state.items : [],
    error: current ? state.error : null,
    hasMore: current && state.next !== null,
    ranking: current ? state.ranking : undefined,
    more: current ? state.more : 'idle',
    loadMore,
  }
}
