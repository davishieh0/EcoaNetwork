/**
 * The foot of a paged list: the button for the page after.
 */

// --- IMPORTS ---
import { useTranslation } from 'react-i18next'
import type { LoadMoreProps } from '@/app/reads/LoadMore.t'
import { Button } from '@/components/ui/Button'

// --- CODE ---
/**
 * Offers the page after while there is one, and says when it failed.
 *
 * @param props - Whether there is more, where loading it is, and loadMore.
 * @returns The button, or nothing at the end of the list.
 */
export function LoadMore({ hasMore, more, onLoad }: LoadMoreProps) {
  const { t } = useTranslation()
  if (!hasMore) return null

  return (
    <div className="mt-6 flex flex-col items-center gap-2">
      <Button
        variant="secondary"
        onClick={onLoad}
        pending={more === 'loading'}
      >
        {t('Load more')}
      </Button>
      {more === 'failed' ? (
        <p role="alert" className="meta text-danger">
          {t('The next page did not load. Try again.')}
        </p>
      ) : null}
    </div>
  )
}
