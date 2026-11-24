/**
 * An entity: what it is called, and the reviews signed about it.
 */

// --- IMPORTS ---
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router'
import { messageOf } from '@/app/forms/messages'
import { routeTo } from '@/app/navigation/navigation'
import { cardOf } from '@/app/reads/cards'
import { useEntityData } from '@/app/reads/entity/useEntityData'
import { LoadMore } from '@/app/reads/LoadMore'
import { Placeholder } from '@/app/reads/Placeholder'
import { ShowLabeled } from '@/app/reads/ShowLabeled'
import { Status } from '@/app/reads/Status'
import { stagger } from '@/app/reads/stagger'
import { useInspection } from '@/app/reads/useInspection'
import {
  categoryOf,
  dateOf,
  identifierOf,
  kindOf,
  nameOf,
} from '@/app/reads/words'
import { EventInspector } from '@/components/ecoa/EventInspector'
import { RatingSummary } from '@/components/ecoa/RatingSummary'
import { ReviewCard } from '@/components/ecoa/ReviewCard'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Container } from '@/components/ui/Container'
import { useRise } from '@/components/ui/useRise'
import { HttpError } from '@/logic/errors/httpError'
import { ROUTES } from '@/routes'

// --- CODE ---
/**
 * Renders the entity's header and its reviews, newest first.
 *
 * @returns The entity screen.
 */
export function EntityPage() {
  const { entityId, entity, reviews, names, labeled, showLabeled } =
    useEntityData()
  const { inspect, inspector } = useInspection()
  const rise = useRise()
  const reduceMotion = useReducedMotion()

  // what arrives after the header fades in where it will stay
  const arrive = {
    initial: reduceMotion ? false : { opacity: 0 },
    animate: { opacity: 1 },
    transition: { duration: reduceMotion ? 0 : 0.3 },
  } as const

  const title = entity.data ? nameOf(entity.data) : identifierOf(entityId)

  // nothing names it: it may still have reviews, from before a name
  const unnamed =
    entity.error instanceof HttpError && entity.error.status === 404
  const failed = entity.error && !unnamed

  const status = reviews.loading
    ? 'Reading the reviews…'
    : // the entity's failure is already said above
      reviews.error && !failed
      ? messageOf(reviews.error)
      : reviews.items.length === 0
      ? 'No reviews yet. Be the first to sign one.'
      : labeled
        ? 'Reviews, newest first, with what moderation hides marked'
        : 'Reviews, newest first'

  return (
    <Container className="max-w-4xl pt-28 pb-16">
      <motion.p {...rise()} className="meta">
        <Link
          to={ROUTES.explore}
          className="text-muted transition-colors hover:text-primary"
        >
          ← Explore
        </Link>
      </motion.p>

      <header className="mt-4 flex flex-wrap items-start justify-between gap-4">
        <div className="flex flex-col gap-2">
          <motion.p {...rise(0.04)} className="meta text-muted">
            {kindOf(entityId)}
            {entity.data?.observed.category
              ? ` · ${categoryOf(entity.data.observed.category.value)}`
              : ''}
          </motion.p>
          <motion.div {...rise(0.08)}>
            {/* the identifier, then the name once it arrives */}
            <AnimatePresence mode="wait" initial={false}>
              <motion.h1
                key={title}
                className="font-display text-h1"
                initial={reduceMotion ? false : { opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={reduceMotion ? undefined : { opacity: 0, y: -6 }}
                transition={{ duration: reduceMotion ? 0 : 0.2 }}
              >
                {title}
              </motion.h1>
            </AnimatePresence>
          </motion.div>
          <motion.p
            {...rise(0.12)}
            className="font-mono text-mono-m text-secondary"
          >
            {identifierOf(entityId)}
          </motion.p>
          {entity.data?.observed.description ? (
            <motion.p {...arrive} className="reading-measure text-secondary">
              {entity.data.observed.description.value}
            </motion.p>
          ) : null}
          {entity.data ? (
            <motion.p {...arrive} className="meta text-muted">
              {entity.data.derived.proposals === 1
                ? '1 proposal for its details'
                : `${entity.data.derived.proposals} proposals for its details`}
              {' · anyone can propose, the network weighs them'}
            </motion.p>
          ) : null}
          {entity.data?.observed.representative ? (
            <motion.p {...arrive} className="flex flex-wrap items-center gap-2">
              <Badge tone="valid" glyph="◆">
                Verified representative
              </Badge>
              <span className="font-mono text-mono-s break-all text-secondary">
                {entity.data.observed.representative.identity}
              </span>
              {entity.data.observed.representative.since ? (
                <span className="meta text-muted">
                  since {dateOf(entity.data.observed.representative.since)}
                </span>
              ) : null}
            </motion.p>
          ) : null}
          {unnamed ? (
            <motion.p {...arrive} className="meta text-muted">
              Nobody has proposed a name for it yet.
            </motion.p>
          ) : null}
        </div>
        <motion.div {...rise(0.16)}>
          <Button
            as={Link}
            {...routeTo('write', { entity: entityId })}
            variant="secondary"
          >
            Write a review
          </Button>
        </motion.div>
      </header>

      {entity.data?.observed.summary &&
      (entity.data.observed.summary.count > 0 ||
        entity.data.observed.summary.conflicted > 0) ? (
        <motion.div {...arrive} className="mt-8">
          <RatingSummary {...entity.data.observed.summary} />
        </motion.div>
      ) : null}

      {failed ? (
        <motion.p {...arrive} role="alert" className="mt-6 text-danger">
          {messageOf(entity.error)}
        </motion.p>
      ) : null}

      <motion.div
        {...rise(0.2)}
        className={
          'mt-10 flex flex-wrap items-center justify-between gap-3 border-b ' +
          'border-structural pb-3'
        }
      >
        <Status
          text={status}
          failed={!!reviews.error && !failed}
        />
        <ShowLabeled checked={labeled} onChange={showLabeled} />
      </motion.div>

      {reviews.loading ? (
        <div className="mt-6">
          <Placeholder count={3} lines={2} />
        </div>
      ) : null}

      <ul className="mt-6 grid gap-4">
        {reviews.items.map((item, index) => {
          const review = cardOf(item, names, entityId)

          return (
            <motion.li key={review.eventId} {...rise(stagger(index))}>
              <ReviewCard
                review={review}
                onInspect={() => inspect(review.eventId)}
                actions={
                  <Link
                    to={ROUTES.review(review.eventId)}
                    className={
                      'meta border-b-2 border-signal text-primary ' +
                      'transition-colors hover:text-secondary'
                    }
                  >
                    Read review →
                  </Link>
                }
              />
            </motion.li>
          )
        })}
      </ul>

      <LoadMore
        hasMore={reviews.hasMore}
        more={reviews.more}
        onLoad={reviews.loadMore}
      />

      <EventInspector {...inspector} />
    </Container>
  )
}
