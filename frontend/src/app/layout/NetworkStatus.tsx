/**
 * Compact header readout of the P2P nodes this indexer observes.
 */

// --- IMPORTS ---
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { momentOf } from '@/app/reads/words'
import { cn } from '@/components/ui/cn'
import {
  getObservedNodes,
  type ObservedNode,
} from '@/logic/datasource/system'

// --- GLOBALS ---
const POLL_MS = 5000

// --- CODE ---
/**
 * Displays the live node count observed by the indexer and expands into a
 * per-node replication table on click.
 *
 * @returns The header node badge and popover.
 */
export function NetworkStatus() {
  const { t } = useTranslation()
  const [nodes, setNodes] = useState<ObservedNode[]>([])
  const [open, setOpen] = useState(false)
  const container = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let live = true

    const load = () => {
      getObservedNodes()
        .then((res) => {
          if (live) setNodes(res.nodes)
        })
        .catch(() => {
          // keep last known observation on transient failure
        })
    }

    load()
    const timer = setInterval(load, POLL_MS)
    return () => {
      live = false
      clearInterval(timer)
    }
  }, [])

  useEffect(() => {
    if (!open) return

    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false)
    }
    const onPointer = (event: MouseEvent) => {
      if (
        container.current &&
        !container.current.contains(event.target as Node)
      ) {
        setOpen(false)
      }
    }

    document.addEventListener('keydown', onKey)
    document.addEventListener('mousedown', onPointer)
    return () => {
      document.removeEventListener('keydown', onKey)
      document.removeEventListener('mousedown', onPointer)
    }
  }, [open])

  if (nodes.length === 0) return null

  const maxEvents = Math.max(...nodes.map((n) => n.events), 0)
  const syncedCount = nodes.filter((n) => n.events === maxEvents).length

  return (
    <div ref={container} className="relative">
      <button
        type="button"
        aria-expanded={open}
        aria-label={t('Network nodes: {{n}} observed', { n: nodes.length })}
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'meta inline-flex cursor-pointer items-center gap-2 border',
          'border-structural bg-surface px-2.5 py-1.5 text-secondary',
          'transition-colors duration-interface hover:border-strong',
          'hover:text-primary',
        )}
      >
        <span
          aria-hidden="true"
          className={cn(
            'inline-block h-2 w-2 rounded-full',
            syncedCount === nodes.length ? 'bg-valid' : 'bg-signal',
          )}
        />
        <span>
          {String(nodes.length).padStart(2, '0')}{' '}
          {nodes.length === 1 ? t('node') : t('nodes')}
        </span>
      </button>

      {open ? (
        <div
          role="region"
          aria-label={t('Observed P2P nodes')}
          className={cn(
            'absolute right-0 mt-2 w-[min(24rem,calc(100vw-2rem))] border',
            'border-structural bg-canvas p-4 shadow-lg',
          )}
        >
          <div
            className={cn(
              'flex items-baseline justify-between gap-2 border-b',
              'border-line pb-2',
            )}
          >
            <p className="meta text-primary">{t('Observed P2P network')}</p>
            <span className="meta text-muted">
              {t('{{synced}}/{{total}} in sync', {
                synced: syncedCount,
                total: nodes.length,
              })}
            </span>
          </div>

          <ul className="mt-2 flex flex-col divide-y divide-line">
            {nodes.map((item) => {
              const inSync = item.events === maxEvents
              return (
                <li
                  key={item.node}
                  className="flex items-center justify-between gap-3 py-2"
                >
                  <div className="flex flex-col">
                    <span className="font-mono text-mono-s text-primary">
                      {item.node}
                    </span>
                    <span className="meta text-muted">
                      {item.lastReceivedAt
                        ? momentOf(item.lastReceivedAt)
                        : t('Waiting for events')}
                    </span>
                  </div>
                  <div className="flex flex-col items-end">
                    <span
                      className={cn(
                        'font-mono text-mono-s',
                        inSync ? 'text-valid' : 'text-signal',
                      )}
                    >
                      #{item.sequence} · {item.events} ev
                    </span>
                    <span className="meta text-muted">
                      {inSync
                        ? t('◆ synced')
                        : `● -${maxEvents - item.events}`}
                    </span>
                  </div>
                </li>
              )
            })}
          </ul>

          <p className="mt-3 border-t border-line pt-2 text-body-s text-muted">
            {t(
              'Read from the indexer’s own cursors. Reads never query nodes ' +
                'directly.',
            )}
          </p>
        </div>
      ) : null}
    </div>
  )
}
