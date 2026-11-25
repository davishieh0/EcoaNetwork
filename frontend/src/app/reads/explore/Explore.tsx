/**
 * Search: the entities on the network, found by name.
 */

// --- IMPORTS ---
import { motion } from 'motion/react'
import type { FormEvent } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { messageOf } from '@/app/forms/messages'
import { LoadMore } from '@/app/reads/LoadMore'
import { Placeholder } from '@/app/reads/Placeholder'
import { Status } from '@/app/reads/Status'
import { stagger } from '@/app/reads/stagger'
import { useExploreData } from '@/app/reads/explore/useExploreData'
import { categoryOf, identifierOf, kindOf, nameOf } from '@/app/reads/words'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { cn } from '@/components/ui/cn'
import { useRise } from '@/components/ui/useRise'
import { ROUTES } from '@/routes'

// --- GLOBALS ---
// the categories most entities fall in, from the protocol's fixed list
const CATEGORIES = [
  'food-and-drink',
  'retail',
  'health',
  'education',
  'technology',
  'financial-services',
  'professional-services',
  'travel-and-lodging',
  'transport',
  'entertainment',
  'telecom',
  'real-estate',
  'utilities',
  'government',
  'manufacturing',
  'nonprofit',
  'other',
]

// --- CODE ---
/**
 * Renders the search box and the entities it finds.
 *
 * @returns The search screen.
 */
export function Explore() {
  const {
    query,
    category,
    search,
    pickCategory,
    loading,
    items,
    error,
    hasMore,
    more,
    loadMore,
  } = useExploreData()
  const rise = useRise()
  const { t } = useTranslation()

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    search(String(new FormData(event.currentTarget).get('q') ?? ''))
  }

  const status = loading
    ? t('Searching…')
    : error
      ? messageOf(error)
      : items.length === 0
        ? query || category
          ? t('Nothing on the network matches this search.')
          : t('Nothing on the network is named yet.')
        : query
          ? t('Entities named “{{query}}”', { query })
          : t('Recently named on the network')

  return (
    <Container className="max-w-4xl pt-28 pb-16">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <motion.h1 {...rise()} className="font-display text-h1">
            {t('Explore')}
          </motion.h1>
          <motion.p {...rise(0.06)} className="text-secondary">
            {t(
              'Find a business, a website or a product, and read what ' +
                'people signed about it.',
            )}
          </motion.p>
        </div>
        <motion.div {...rise(0.12)}>
          <Button as={Link} to={ROUTES.write} variant="secondary">
            {t('Write a review')}
          </Button>
        </motion.div>
      </header>

      <motion.form
        {...rise(0.18)}
        role="search"
        onSubmit={submit}
        className="mt-8 flex gap-2"
      >
        <label className="flex flex-1 flex-col gap-2">
          <span className="sr-only">{t('Search by name')}</span>
          <input
            // a new search in the address resets what the box shows
            key={query}
            name="q"
            type="search"
            defaultValue={query}
            maxLength={120}
            placeholder={t('Business, website, or product name')}
            className={cn(
              'w-full border border-structural bg-surface p-3',
              'transition-colors duration-interface focus:border-strong',
            )}
          />
        </label>
        <Button type="submit">{t('Search')}</Button>
      </motion.form>

      <motion.div {...rise(0.22)} className="mt-3 flex flex-wrap gap-2">
        {['', ...CATEGORIES].map((kind) => (
          <button
            key={kind || 'all'}
            type="button"
            aria-pressed={category === kind}
            onClick={() => pickCategory(kind)}
            className={cn(
              'meta cursor-pointer border px-3 py-1.5 transition-colors',
              'duration-interface',
              category === kind
                ? 'border-signal bg-signal text-on-signal'
                : 'border-structural text-secondary hover:text-primary',
            )}
          >
            {kind ? categoryOf(kind) : t('All')}
          </button>
        ))}
      </motion.div>

      <Status text={status} failed={!!error} className="my-4" />

      {loading ? (
        <Placeholder count={4} lines={1} />
      ) : (
        <ul className="grid gap-3">
          {items.map((entity, index) => (
            <motion.li key={entity.entity} {...rise(stagger(index))}>
              <Link
                to={ROUTES.entity(entity.entity)}
                className={cn(
                  'group flex items-center justify-between gap-4 border',
                  'border-structural bg-surface p-5',
                  'transition-colors duration-interface hover:border-strong',
                  'hover:bg-surface-raised',
                )}
              >
                <span className="flex min-w-0 flex-col gap-1">
                  <span className="meta text-muted">
                    {kindOf(entity.entity)}
                    {entity.observed.category
                      ? ` · ${categoryOf(entity.observed.category.value)}`
                      : ''}
                  </span>
                  <span className="font-display text-h4 font-semibold">
                    {nameOf(entity)}
                  </span>
                  <span className="font-mono text-mono-s text-secondary">
                    {entity.observed.name
                      ? identifierOf(entity.entity)
                      : t('Not named yet')}
                  </span>
                </span>
                {/* slides in as the pointer arrives */}
                <span
                  aria-hidden="true"
                  className={cn(
                    'font-display text-h4 text-muted opacity-0',
                    'transition-all duration-interface ease-standard',
                    'motion-safe:-translate-x-2',
                    'group-hover:translate-x-0 group-hover:text-primary',
                    'group-hover:opacity-100 group-focus-visible:opacity-100',
                  )}
                >
                  →
                </span>
              </Link>
            </motion.li>
          ))}
        </ul>
      )}

      <LoadMore hasMore={hasMore} more={more} onLoad={loadMore} />
    </Container>
  )
}
