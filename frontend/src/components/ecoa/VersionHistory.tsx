/**
 * Every version of a review, oldest first: an edit adds one, never erases.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import { useTranslation } from 'react-i18next'
import type { VersionHistoryProps } from '@/components/ecoa/VersionHistory.t'
import { Rating } from '@/components/ecoa/Rating'
import { Badge } from '@/components/ui/Badge'
import { cn } from '@/components/ui/cn'
import { useReveal } from '@/components/ui/useReveal'

// --- CODE ---
/**
 * Lists a review's versions, the current one marked, each still readable.
 *
 * @param props - The versions, an optional inspect handler, extra classes.
 * @returns The history.
 */
export function VersionHistory({
  versions,
  onInspect,
  className,
}: VersionHistoryProps) {
  const { t } = useTranslation()
  const reveal = useReveal()

  return (
    <ol className={cn('flex flex-col', className)}>
      {versions.map((version) => (
        <motion.li
          {...reveal()}
          key={version.eventId}
          className={cn(
            'relative border-l-2 py-4 pl-6',
            version.current ? 'border-signal' : 'border-structural',
          )}
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-mono-m text-primary">
              v{version.number}
            </span>
            {version.current ? (
              <Badge tone="signal" glyph="●">
                {t('Current')}
              </Badge>
            ) : null}
            {version.asked && !version.current ? (
              <Badge tone="info" glyph="◆">
                {t('The version linked')}
              </Badge>
            ) : null}
            <span className="meta text-muted">{version.publishedAt}</span>
            <Rating value={version.rating} className="ml-auto" />
          </div>

          {/* Human content: an old version stays readable, not a ghost. */}
          {version.title ? (
            <p className="mt-2 font-semibold text-pretty">{version.title}</p>
          ) : null}
          <p
            className={cn(
              'reading-measure mt-1 whitespace-pre-line text-secondary',
              'text-pretty',
            )}
          >
            {version.body}
          </p>

          <div className="mt-2 flex flex-wrap items-center gap-3">
            <span className="meta text-muted">
              ▲ {version.endorsements} · ▼ {version.disputes}
              {version.current ? '' : ' ' + t('on this version')}
            </span>
            {onInspect ? (
              <button
                type="button"
                onClick={() => onInspect(version.eventId)}
                className={cn(
                  'meta cursor-pointer border-b border-signal text-primary',
                  'hover:text-secondary',
                )}
              >
                Inspect v{version.number} ↗
              </button>
            ) : null}
          </div>
        </motion.li>
      ))}
    </ol>
  )
}
